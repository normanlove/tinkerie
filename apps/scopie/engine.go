// The acquisition engine: one goroutine, completely separate from the screen.
//
// Streaming scopes (Hantek 6022 family, demo): samples arrive as fast as USB
// allows into a ring buffer; every block is searched for trigger events and
// every trigger becomes a captured waveform - often thousands per second.
//
// Self-triggering scopes (bench scopes, PicoScope, handhelds, old Hantek
// DSO): the driver hands over complete captured waveforms.
//
// Either way the display (60 fps) picks up the newest waveform plus a running
// sum (for averaging) and a 2-D histogram (for persistence) of *every*
// waveform captured since its last refresh, so nothing is thrown away.
package scopie

import (
	"fmt"
	"log"
	"math"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

const (
	ringSize          = 1 << 23 // samples kept per channel (8 M)
	maxFramesPerBlock = 256
	targetPerScreen   = 10_000
	maxFrameSamples   = 2_000_000
	persistCols       = 1000
	histBins          = 256
)

// Frame is one waveform handed to the display.
type Frame struct {
	Data      [maxCh][]float32 // ADC codes, nil for channels not captured
	DT        float64
	TFirst    float64 // time of Data[*][0] relative to the trigger point
	Triggered bool
	Roll      bool
	Gen       int64
	Seq       int64
	Scales    [maxCh]Scale
	Ranges    [maxCh]CodeRange
	PreLen    int
}

func (f *Frame) first() []float32 {
	for _, d := range f.Data {
		if d != nil {
			return d
		}
	}
	return nil
}

// Applied says what the engine actually did with the requested settings.
type Applied struct {
	Rate      float64        `json:"rate"`
	FrameLen  int            `json:"frameLen"`
	PreLen    int            `json:"preLen"`
	NCh       int            `json:"nch"`
	FullScale [maxCh]float64 `json:"fullScale"`
	Roll      bool           `json:"roll"`
	Persist   bool           `json:"persist"`
	Average   bool           `json:"average"`
}

// Taken is what the display gets on each refresh.
type Taken struct {
	Frame     *Frame
	Gen       int64
	State     string
	Message   string
	Error     string
	Info      *DeviceInfo
	InfoVer   int
	Waveforms int64
	Bytes     int64
	Hist      [maxCh][]float32
	HistMeta  [maxCh]CodeRange
	AvgSum    [maxCh][]float64
	AvgN      int
	Applied   Applied
}

type engineCmd struct {
	name string
	arg  any
}

type demoArgs struct {
	Signals []string
	Probes  []float64
}

type demoSource interface {
	SetDemo(signals []string, probes []float64)
}

type streamPlan struct {
	rid, nch      int
	rate          float64
	gains         [2]int
	full          [2]float64
	frameLen, pre int
	dc            [2]bool
	hwAC, roll    bool
}

type Engine struct {
	cmds chan engineCmd

	// output, guarded by mu
	mu        sync.Mutex
	oFrame    *Frame
	oHist     [maxCh][]float32
	oHistMeta [maxCh]CodeRange
	oAvgSum   [maxCh][]float64
	oAvgN     int
	oGen      int64
	oState    string
	oMessage  string
	oError    string
	oInfo     *DeviceInfo
	oInfoVer  int
	oWaveform int64
	oBytes    int64
	oApplied  Applied

	// engine goroutine state
	dev       Device
	panel     Panel
	running   bool
	single    bool
	streaming bool
	gen       int64
	seq       int64
	quit      bool
	stopReq   bool
	demo      *demoArgs
	acMean    [maxCh]float64
	hwKey     string
	procKey   string
	plan      streamPlan
	applied   Applied
	nch       int
	scales    [2]Scale
	ring      [2][]uint8
	total     int64
	pending   []int64
	lastTrig  int64
	trigState [2]bool
	skip      int
	lastEmit  time.Time
	lastTrigW time.Time
	dataCh    chan []byte
	failCh    chan error
	dropped   int64
	colMap    []int
	colMapN   int
	codeRange CodeRange
	done      chan struct{}
}

func newEngine() *Engine {
	e := &Engine{cmds: make(chan engineCmd, 512), panel: defaultPanel(), oState: "disconnected", done: make(chan struct{})}
	e.applied.FullScale = [maxCh]float64{5, 5, 5, 5}
	e.oApplied = e.applied
	e.codeRange = CodeRange{0, 255}
	e.resetBuffers()
	return e
}

// ---------------------------------------------------------------- API

func (e *Engine) send(name string, arg any) { e.cmds <- engineCmd{name, arg} }

func (e *Engine) Connect(f func() (Device, error)) { e.send("connect", f) }
func (e *Engine) Disconnect()                      { e.send("disconnect", nil) }
func (e *Engine) Apply(p Panel)                    { e.send("panel", p) }
func (e *Engine) Run()                             { e.send("run", nil) }
func (e *Engine) StopAcq()                         { e.send("stop", nil) }
func (e *Engine) Single()                          { e.send("single", nil) }
func (e *Engine) SetDemo(sigs []string, probes []float64) {
	e.send("demo", &demoArgs{sigs, probes})
}
func (e *Engine) Quit() {
	e.send("quit", nil)
	select {
	case <-e.done:
	case <-time.After(3 * time.Second):
	}
}

// CurrentGen is the generation of settings the engine is capturing with.
func (e *Engine) CurrentGen() int64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.oGen
}

// Take returns the newest data and resets the accumulators.
func (e *Engine) Take() Taken {
	e.mu.Lock()
	defer e.mu.Unlock()
	t := Taken{
		Frame: e.oFrame, Gen: e.oGen, State: e.oState, Message: e.oMessage, Error: e.oError,
		Info: e.oInfo, InfoVer: e.oInfoVer, Waveforms: e.oWaveform, Bytes: e.oBytes,
		Hist: e.oHist, HistMeta: e.oHistMeta, AvgSum: e.oAvgSum, AvgN: e.oAvgN, Applied: e.oApplied,
	}
	e.oError = ""
	e.oHist = [maxCh][]float32{}
	e.oAvgSum = [maxCh][]float64{}
	e.oAvgN = 0
	return t
}

// ---------------------------------------------------------------- goroutine

func (e *Engine) Start() { go e.loop() }

func (e *Engine) loop() {
	defer close(e.done)
	for !e.quit {
		e.step()
	}
	if e.dev != nil {
		e.stopStream()
		e.dev.Close()
		e.dev = nil
	}
}

func (e *Engine) step() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("engine panic: %v\n%s", r, debug.Stack())
			e.fail(fmt.Sprintf("Unexpected error: %v", r))
		}
	}()
	if !e.streaming {
		select {
		case c := <-e.cmds:
			e.handleBatch(c)
		case <-time.After(50 * time.Millisecond):
		}
		return
	}
	if e.dev.Info().Kind == "stream" {
		timer := time.NewTimer(10 * time.Millisecond)
		select {
		case c := <-e.cmds:
			e.handleBatch(c)
		case d := <-e.dataCh:
			e.onData(d)
			// catch up with whatever else has arrived
			for i := 0; i < 16; i++ {
				select {
				case d2 := <-e.dataCh:
					e.onData(d2)
					continue
				default:
				}
				break
			}
		case err := <-e.failCh:
			e.fail(err.Error())
		case <-timer.C:
		}
		timer.Stop()
		if !e.streaming {
			return
		}
		if e.stopReq {
			e.stopReq = false
			e.stopStream()
			e.setState("stopped", nil)
		} else {
			e.afterPump()
		}
		return
	}
	select {
	case c := <-e.cmds:
		e.handleBatch(c)
	default:
	}
	if e.streaming {
		e.frameStep()
	}
}

func (e *Engine) fail(msg string) {
	log.Printf("device problem: %s", msg)
	if e.dev != nil {
		e.stopStream()
		e.dev.Close()
	}
	e.dev = nil
	e.streaming = false
	e.running = false
	e.mu.Lock()
	e.oState = "disconnected"
	e.oError = msg
	e.oMessage = ""
	e.oInfo = nil
	e.oInfoVer++
	e.mu.Unlock()
}

func (e *Engine) setState(state string, msg *string) {
	e.mu.Lock()
	e.oState = state
	if msg != nil {
		e.oMessage = *msg
	}
	e.mu.Unlock()
}

func strp(s string) *string { return &s }

func (e *Engine) handleBatch(first engineCmd) {
	batch := []engineCmd{first}
	for {
		select {
		case c := <-e.cmds:
			batch = append(batch, c)
			continue
		default:
		}
		break
	}
	// Only the newest front-panel settings matter: a slider drag can queue
	// dozens, and sending each one to a bench scope would take a while.
	lastPanel := -1
	for i, c := range batch {
		if c.name == "panel" {
			lastPanel = i
		}
	}
	for i, c := range batch {
		if c.name == "panel" && i != lastPanel {
			continue
		}
		if err := e.do(c); err != nil {
			e.fail(err.Error())
		}
	}
}

func (e *Engine) do(c engineCmd) error {
	switch c.name {
	case "quit":
		e.quit = true
	case "connect":
		return e.connect(c.arg.(func() (Device, error)))
	case "disconnect":
		if e.dev != nil {
			e.stopStream()
			e.dev.Close()
			e.dev = nil
		}
		e.running = false
		e.mu.Lock()
		e.oState = "disconnected"
		e.oInfo = nil
		e.oInfoVer++
		e.mu.Unlock()
	case "panel":
		p := c.arg.(Panel)
		p.sanitize()
		return e.applyPanel(p)
	case "run":
		e.single = false
		e.running = true
		return e.ensureStream(e.dev != nil && e.dev.Info().Kind == "frame")
	case "single":
		e.single = true
		e.running = true
		return e.ensureStream(true)
	case "stop":
		e.running = false
		e.single = false
		e.stopStream()
		if e.dev != nil {
			e.setState("stopped", nil)
		}
	case "demo":
		e.demo = c.arg.(*demoArgs)
		e.pushDemo()
	}
	return nil
}

func (e *Engine) pushDemo() {
	if e.dev == nil || e.demo == nil {
		return
	}
	if d, ok := e.dev.(demoSource); ok {
		d.SetDemo(e.demo.Signals, e.demo.Probes)
	}
}

func (e *Engine) connect(factory func() (Device, error)) error {
	if e.dev != nil {
		e.stopStream()
		e.dev.Close()
		e.dev = nil
	}
	e.mu.Lock()
	e.oInfo = nil
	e.oInfoVer++
	e.mu.Unlock()
	e.setState("connecting", strp("Connecting…"))
	dev, err := factory()
	if err != nil {
		return err
	}
	if err := dev.Open(func(m string) { e.setState("connecting", strp(m)) }); err != nil {
		dev.Close()
		return err
	}
	log.Printf("connected: %s", dev.Info().Name)
	e.dev = dev
	e.pushDemo()
	e.mu.Lock()
	e.oInfo = dev.Info()
	e.oInfoVer++
	e.oState = "stopped"
	e.oMessage = ""
	e.mu.Unlock()
	e.hwKey = ""
	e.newGeneration()
	if e.running {
		return e.ensureStream(true)
	}
	return nil
}

// ---------------------------------------------------------------- settings

func frameKey(p *Panel) string {
	s := ""
	for _, c := range p.Ch {
		s += fmt.Sprintf("%v,%g,%.3f,%s;", c.Enabled, c.VDiv, c.Pos, c.Coupling)
	}
	return s + fmt.Sprintf("%g,%.3f,%d,%s,%.6g,%s,%g", p.TDiv, p.TrigX, p.TrigSource, p.TrigEdge, p.TrigLevel, p.TrigMode, p.CalFreq)
}

func (e *Engine) applyPanel(p Panel) error {
	old := e.panel
	e.panel = p
	if e.dev == nil {
		return nil
	}
	if e.dev.Info().Kind == "frame" {
		if frameKey(&old) != frameKey(&p) {
			if err := e.dev.(FrameDevice).ApplyPanel(&e.panel); err != nil {
				return err
			}
			e.newGeneration()
		}
		e.updateAppliedFrame()
		return nil
	}
	key, proc := e.streamPlan(&p)
	if key != e.hwKey || proc != e.procKey {
		e.newGeneration()
	}
	if key != e.hwKey && e.streaming {
		if err := e.ensureStream(true); err != nil {
			return err
		}
	}
	e.procKey = proc
	return nil
}

// streamPlan translates the front panel into hardware settings for a streaming scope.
func (e *Engine) streamPlan(p *Panel) (key, proc string) {
	dev := e.dev.(StreamDevice)
	info := dev.Info()
	nch := 1
	for _, c := range p.ChannelsNeeded {
		if c >= 1 {
			nch = 2
		}
	}
	rates := dev.RateIDs(nch)
	screen := float64(info.HDivs) * p.TDiv
	var best RateID
	if p.Rate <= 0 {
		best = rates[0]
		for _, r := range rates {
			if r.Rate*screen <= targetPerScreen {
				best = r
			}
		}
	} else {
		bd := math.Inf(1)
		for _, r := range rates {
			if d := math.Abs(math.Log(r.Rate / p.Rate)); d < bd {
				bd, best = d, r
			}
		}
		if best.Rate*screen > maxFrameSamples {
			best = rates[0]
			for _, r := range rates {
				if r.Rate*screen <= maxFrameSamples {
					best = r
				}
			}
		}
	}
	frameLen := int(math.Round(screen * best.Rate))
	if frameLen < 32 {
		frameLen = 32
	}
	pre := int(math.Round((p.TrigX + float64(info.HDivs)/2) / float64(info.HDivs) * float64(frameLen)))
	pre = clampInt(pre, 1, frameLen-1)
	gs := dev.Gains()
	sorted := append([]Gain(nil), gs...)
	for i := 1; i < len(sorted); i++ { // insertion sort by full scale
		for j := i; j > 0 && sorted[j].FS < sorted[j-1].FS; j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}
	var pl streamPlan
	for c := 0; c < 2; c++ {
		cp := p.Ch[c]
		need := (4 + math.Abs(cp.Pos)) * cp.VDiv
		choice := sorted[len(sorted)-1]
		for _, g := range sorted {
			if g.FS*0.98 >= need {
				choice = g
				break
			}
		}
		pl.gains[c] = choice.ID
		pl.full[c] = choice.FS
		pl.dc[c] = cp.Coupling == "DC"
	}
	pl.hwAC = info.HWAC || (info.ACDCModOption && p.HWACDCMod)
	pl.rid, pl.rate, pl.nch, pl.frameLen, pl.pre = best.ID, best.Rate, nch, frameLen, pre
	pl.roll = p.Roll && info.RollMode
	dcKey := "none"
	if pl.hwAC {
		dcKey = fmt.Sprint(pl.dc)
	}
	key = fmt.Sprint(pl.rid, nch, pl.gains, dcKey, p.CalFreq, p.HWACDCMod)
	proc = fmt.Sprint(frameLen, pre, p.TrigSource, p.TrigEdge, pl.roll, p.Persist, p.Average)
	e.plan = pl
	a := Applied{Rate: pl.rate, FrameLen: frameLen, PreLen: pre, NCh: nch,
		FullScale: [maxCh]float64{pl.full[0], pl.full[1], 5, 5},
		Roll:      pl.roll, Persist: p.Persist && !pl.roll, Average: p.Average && !pl.roll}
	e.applied = a
	e.mu.Lock()
	e.oApplied = a
	e.mu.Unlock()
	return key, proc
}

// triggerCode gives the trigger level and hysteresis in ADC codes of the source channel.
func (e *Engine) triggerCode() (src int, level, hyst float64, softAC bool) {
	p := &e.panel
	src = p.TrigSource
	if src > e.nch-1 {
		src = e.nch - 1
	}
	sc := e.scales[src]
	softAC = p.Ch[src].Coupling == "AC" && !e.plan.hwAC
	base := sc.Zero
	if softAC {
		base = e.acMean[src]
	}
	level = p.TrigLevel/sc.K + base
	vd := p.Ch[src].VDiv / sc.K
	if p.NoiseReject {
		hyst = math.Max(4, 0.3*vd)
	} else {
		hyst = math.Max(1.5, 0.05*vd)
	}
	return
}

func (e *Engine) updateAppliedFrame() {
	a := Applied{NCh: e.dev.Info().Channels, Persist: e.panel.Persist, Average: e.panel.Average}
	for c := range a.FullScale {
		a.FullScale[c] = 4 * e.panel.Ch[c].VDiv
	}
	e.applied = a
	e.mu.Lock()
	e.oApplied = a
	e.mu.Unlock()
}

func (e *Engine) newGeneration() {
	e.gen++
	e.pending = e.pending[:0]
	e.lastTrig = -(1 << 62)
	e.trigState = [2]bool{}
	e.mu.Lock()
	e.oGen = e.gen
	e.oHist = [maxCh][]float32{}
	e.oAvgSum = [maxCh][]float64{}
	e.oAvgN = 0
	e.oFrame = nil
	e.mu.Unlock()
	e.colMap = nil
}

func (e *Engine) resetBuffers() {
	e.total = 0
	e.pending = e.pending[:0]
	e.lastTrig = -(1 << 62)
	e.trigState = [2]bool{}
	e.skip = 2
	e.lastEmit = time.Time{}
	e.lastTrigW = time.Time{}
	e.colMap = nil
}

// ---------------------------------------------------------------- streaming

func chunkBytes(rate float64, nch int) int {
	want := int(rate * float64(nch) * 0.02) // about 20 ms per USB block
	want = clampInt(want, 1024, 1<<20)
	return (want + 511) / 512 * 512
}

func (e *Engine) ensureStream(restart bool) error {
	if e.dev == nil || !e.running {
		return nil
	}
	if e.streaming && !restart {
		return nil
	}
	e.stopStream()
	e.stopReq = false
	if e.dev.Info().Kind == "frame" {
		fd := e.dev.(FrameDevice)
		if err := fd.ApplyPanel(&e.panel); err != nil {
			return err
		}
		e.updateAppliedFrame()
		e.newGeneration()
		if err := fd.Start(e.single); err != nil {
			return err
		}
		e.streaming = true
		e.lastTrigW = time.Now()
		e.setState("running", strp(""))
		return nil
	}
	dev := e.dev.(StreamDevice)
	e.hwKey, e.procKey = e.streamPlan(&e.panel)
	pl := e.plan
	nch, err := dev.Configure(pl.rid, pl.nch, pl.gains, pl.dc, e.panel.HWACDCMod, e.panel.CalFreq)
	if err != nil {
		return err
	}
	e.nch = nch
	for c := 0; c < 2; c++ {
		e.scales[c] = dev.Scale(c, pl.gains[c])
		e.acMean[c] = e.scales[c].Zero
	}
	e.acMean[2], e.acMean[3] = 0, 0
	if e.ring[0] == nil {
		for c := range e.ring {
			e.ring[c] = make([]uint8, 2*ringSize)
		}
	}
	e.resetBuffers()
	e.newGeneration()
	e.lastTrigW = time.Now()
	ch := make(chan []byte, 96)
	fail := make(chan error, 1)
	e.dataCh, e.failCh = ch, fail
	err = dev.StartStream(chunkBytes(pl.rate, nch), func(b []byte) {
		cp := make([]byte, len(b))
		copy(cp, b)
		select {
		case ch <- cp:
		default:
			atomic.AddInt64(&e.dropped, 1) // the engine fell behind: drop this block
		}
	}, func(err error) {
		select {
		case fail <- err:
		default:
		}
	})
	if err != nil {
		return err
	}
	e.streaming = true
	e.setState("running", strp(""))
	return nil
}

func (e *Engine) stopStream() {
	if e.streaming && e.dev != nil {
		if e.dev.Info().Kind == "frame" {
			_ = e.dev.(FrameDevice).Stop()
		} else {
			e.dev.(StreamDevice).StopStream()
		}
	}
	e.streaming = false
	if e.dataCh != nil {
		for len(e.dataCh) > 0 {
			<-e.dataCh
		}
	}
}

func (e *Engine) onData(data []byte) {
	if e.skip > 0 {
		e.skip--
		return
	}
	nch := e.nch
	n := len(data) / nch
	if n == 0 {
		return
	}
	pos := int(e.total % ringSize)
	if nch == 2 {
		r0, r1 := e.ring[0], e.ring[1]
		for i := 0; i < n; i++ {
			p := pos + i
			if p >= ringSize {
				p -= ringSize
			}
			a, b := data[2*i], data[2*i+1]
			r0[p], r0[p+ringSize] = a, a
			r1[p], r1[p+ringSize] = b, b
		}
	} else {
		r := e.ring[0]
		for i := 0; i < n; i++ {
			p := pos + i
			if p >= ringSize {
				p -= ringSize
			}
			r[p], r[p+ringSize] = data[i], data[i]
		}
	}
	startAbs := e.total
	e.total += int64(n)
	e.mu.Lock()
	e.oBytes += int64(len(data))
	e.mu.Unlock()

	src, level, hyst, softAC := e.triggerCode()
	block := e.slice(src, startAbs, n)
	if softAC { // software AC coupling: the trigger follows the signal's average
		s := 0
		for _, v := range block {
			s += int(v)
		}
		e.acMean[src] = 0.9*e.acMean[src] + 0.1*float64(s)/float64(n)
	}
	if e.plan.roll || !e.running {
		return
	}
	e.findTriggers(block, startAbs, level, hyst)
	e.extractFrames(level)
}

// findTriggers appends absolute sample indices of trigger events in block.
func (e *Engine) findTriggers(block []uint8, startAbs int64, level, hyst float64) {
	edge := e.panel.TrigEdge
	rs, fs := e.trigState[0], e.trigState[1]
	lowR, highF := level-hyst, level+hyst
	doR := edge == "rising" || edge == "both"
	doF := edge == "falling" || edge == "both"
	for i, u := range block {
		v := float64(u)
		hit := false
		if doR {
			if v < lowR {
				rs = true
			} else if v >= level {
				if rs {
					hit = true
				}
				rs = false
			}
		}
		if doF {
			// a falling edge of x is a rising edge of -x
			if v > highF {
				fs = true
			} else if v <= level {
				if fs {
					hit = true
				}
				fs = false
			}
		}
		if hit {
			e.pending = append(e.pending, startAbs+int64(i))
		}
	}
	e.trigState = [2]bool{rs, fs}
}

func (e *Engine) slice(c int, start int64, n int) []uint8 {
	p := int(start % ringSize)
	return e.ring[c][p : p+n]
}

func (e *Engine) extractFrames(level float64) {
	pl := e.plan
	pre, post := int64(pl.pre), int64(pl.frameLen-pl.pre)
	holdoff := int64(e.panel.Holdoff * pl.rate)
	minGap := int64(pl.frameLen) + holdoff
	oldest := e.total - ringSize + 1
	if oldest < 0 {
		oldest = 0
	}
	var accepted []int64
	wait := -1
	for i, t := range e.pending {
		if t < e.lastTrig+minGap {
			continue
		}
		if t-pre < oldest || t < 1 {
			continue
		}
		if t+post > e.total {
			wait = i
			break
		}
		accepted = append(accepted, t)
		e.lastTrig = t
	}
	if wait >= 0 {
		e.pending = append(e.pending[:0], e.pending[wait:]...)
	} else {
		e.pending = e.pending[:0]
	}
	if len(accepted) == 0 {
		return
	}
	cp := 2_000_000 / maxInt(1, pl.frameLen)
	cp = minInt(maxFramesPerBlock, maxInt(8, cp))
	if len(accepted) > cp {
		step := float64(len(accepted)) / float64(cp)
		sub := make([]int64, 0, cp)
		for i := 0; i < cp-1; i++ {
			sub = append(sub, accepted[int(float64(i)*step)])
		}
		accepted = append(sub, accepted[len(accepted)-1])
	}
	// sub-sample position of the last trigger, for a jitter-free display
	src := minInt(e.panel.TrigSource, e.nch-1)
	a := e.slice(src, accepted[len(accepted)-1]-1, 2)
	a0, a1 := float64(a[0]), float64(a[1])
	var frac float64
	if e.panel.TrigEdge == "falling" || (e.panel.TrigEdge == "both" && a1 < a0) {
		frac = crossingFraction(-a0, -a1, -level)
	} else {
		frac = crossingFraction(a0, a1, level)
	}
	dt := 1 / pl.rate
	starts := make([]int64, len(accepted))
	for i, t := range accepted {
		starts[i] = t - pre
	}
	e.publishStream(starts, pl.frameLen, (1-frac-float64(pre))*dt, true)
	if e.single {
		e.single = false
		e.running = false
		e.stopReq = true // stopped from the main loop
	}
}

// publishStream publishes waveforms that start at the given absolute ring
// positions. Every one feeds persistence/averaging; the last is displayed.
func (e *Engine) publishStream(starts []int64, n int, tFirst float64, triggered bool) {
	pl := e.plan
	last := starts[len(starts)-1]
	f := &Frame{DT: 1 / pl.rate, TFirst: tFirst, Triggered: triggered, Roll: pl.roll, PreLen: pl.pre}
	for c := 0; c < maxCh; c++ {
		f.Scales[c] = Scale{1, 0}
		f.Ranges[c] = e.codeRange
	}
	for c := 0; c < e.nch; c++ {
		f.Data[c] = u8ToF32(e.slice(c, last, n))
		f.Scales[c] = e.scales[c]
	}
	if triggered {
		e.lastTrigW = time.Now()
	}
	var hist [maxCh][]float32
	var avg [maxCh][]float64
	a := e.applied
	if (a.Persist || a.Average) && !pl.roll {
		for c := 0; c < e.nch; c++ {
			if a.Persist {
				h := make([]float32, persistCols*histBins)
				for _, s := range starts {
					histAddU8(h, e.slice(c, s, n), e.codeRange, e.colMapFor(n))
				}
				hist[c] = h
			}
			if a.Average {
				s := make([]float64, n)
				for _, st := range starts {
					for i, v := range e.slice(c, st, n) {
						s[i] += float64(v)
					}
				}
				avg[c] = s
			}
		}
	}
	e.publish(f, len(starts), hist, avg)
}

func u8ToF32(b []uint8) []float32 {
	out := make([]float32, len(b))
	for i, v := range b {
		out[i] = float32(v)
	}
	return out
}

func (e *Engine) colMapFor(n int) []int {
	if n < persistCols {
		return nil
	}
	if e.colMap == nil || e.colMapN != n {
		e.colMap = make([]int, n)
		for i := range e.colMap {
			e.colMap[i] = i * persistCols / n
		}
		e.colMapN = n
	}
	return e.colMap
}

// histAddU8 adds one waveform to a columns x levels histogram. Jumps between
// neighbouring samples are filled in (dimmer), like the beam of an analog
// scope sweeping across an edge, so steep edges stay visible.
func histAddU8(h []float32, d []uint8, cr CodeRange, colMap []int) {
	span := cr.Hi - cr.Lo + 1
	scale := histBins / span
	if colMap != nil {
		prev := -1
		for i, v := range d {
			b := clampInt(int((float64(v)-cr.Lo)*scale), 0, histBins-1)
			row := h[colMap[i]*histBins : colMap[i]*histBins+histBins]
			row[b]++
			if prev >= 0 {
				fillJump(row, prev, b)
			}
			prev = b
		}
		return
	}
	histInterp(h, len(d), func(i int) float64 { return float64(d[i]) }, cr)
}

// fillJump lights the levels strictly between a and b in one column.
func fillJump(row []float32, a, b int) {
	if a > b {
		a, b = b, a
	}
	for k := a + 1; k < b; k++ {
		row[k] += 0.35
	}
}

func histAddF32(h []float32, d []float32, cr CodeRange) {
	n := len(d)
	span := cr.Hi - cr.Lo + 1
	scale := histBins / span
	if n >= persistCols {
		prev := -1
		for i, v := range d {
			b := clampInt(int((float64(v)-cr.Lo)*scale), 0, histBins-1)
			col := i * persistCols / n
			row := h[col*histBins : col*histBins+histBins]
			row[b]++
			if prev >= 0 {
				fillJump(row, prev, b)
			}
			prev = b
		}
		return
	}
	histInterp(h, n, func(i int) float64 { return float64(d[i]) }, cr)
}

// histInterp stretches a short waveform over all columns by linear interpolation.
func histInterp(h []float32, n int, at func(int) float64, cr CodeRange) {
	if n < 1 {
		return
	}
	span := cr.Hi - cr.Lo + 1
	scale := histBins / span
	prev := -1
	for col := 0; col < persistCols; col++ {
		x := float64(col) * float64(n-1) / float64(persistCols-1)
		i := int(x)
		var v float64
		if i >= n-1 {
			v = at(n - 1)
		} else {
			fr := x - float64(i)
			v = at(i)*(1-fr) + at(i+1)*fr
		}
		b := clampInt(int((v-cr.Lo)*scale), 0, histBins-1)
		h[col*histBins+b]++
		if prev >= 0 {
			fillJump(h[col*histBins:col*histBins+histBins], prev, b)
		}
		prev = b
	}
}

func (e *Engine) publish(f *Frame, count int, hist [maxCh][]float32, avg [maxCh][]float64) {
	e.seq++
	f.Gen, f.Seq = e.gen, e.seq
	e.lastEmit = time.Now()
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.oGen != e.gen {
		return
	}
	e.oFrame = f
	e.oWaveform += int64(count)
	for c := 0; c < maxCh; c++ {
		if hist[c] != nil {
			if e.oHist[c] == nil || e.oHistMeta[c] != f.Ranges[c] {
				e.oHist[c] = hist[c]
			} else {
				for i, v := range hist[c] {
					e.oHist[c][i] += v
				}
			}
			e.oHistMeta[c] = f.Ranges[c]
		}
		if avg[c] != nil {
			if e.oAvgSum[c] == nil || len(e.oAvgSum[c]) != len(avg[c]) {
				e.oAvgSum[c] = avg[c]
				e.oAvgN = 0
			} else {
				for i, v := range avg[c] {
					e.oAvgSum[c][i] += v
				}
			}
		}
	}
	if e.applied.Average {
		e.oAvgN += count
	}
	if e.oState != "running" {
		e.oState = "running"
	}
}

func (e *Engine) afterPump() {
	if !e.running || e.total == 0 {
		return
	}
	pl := e.plan
	now := time.Now()
	n := pl.frameLen
	dt := 1 / pl.rate
	if pl.roll {
		if now.Sub(e.lastEmit) >= time.Second/60 {
			m := int(minI64(e.total, int64(n)))
			e.publishStream([]int64{e.total - int64(m)}, m, -float64(m-1)*dt, false)
		}
		return
	}
	if e.panel.TrigMode == "auto" && !e.single {
		timeout := math.Max(0.1, 2.5*float64(n)*dt)
		if now.Sub(e.lastTrigW).Seconds() > timeout && now.Sub(e.lastEmit) >= time.Second/60 && e.total >= int64(n) {
			e.publishStream([]int64{e.total - int64(n)}, n, -float64(pl.pre)*dt, false)
		}
	} else {
		e.mu.Lock()
		if e.oState == "running" && now.Sub(e.lastTrigW) > 250*time.Millisecond {
			e.oState = "waiting"
		}
		e.mu.Unlock()
	}
}

// ---------------------------------------------------------------- frame devices

func (e *Engine) frameStep() {
	fd, err := e.dev.(FrameDevice).ReadFrame(0.05)
	if err != nil {
		e.fail(err.Error())
		return
	}
	now := time.Now()
	if fd == nil {
		if e.panel.TrigMode == "normal" || e.single {
			e.mu.Lock()
			if e.oState == "running" && now.Sub(e.lastTrigW) > 500*time.Millisecond {
				e.oState = "waiting"
			}
			e.mu.Unlock()
		}
		return
	}
	bpc := int64((e.dev.Info().Bits + 7) / 8)
	nbytes := int64(0)
	for _, c := range fd.Codes {
		nbytes += int64(len(c)) * bpc
	}
	e.mu.Lock()
	e.oBytes += nbytes
	e.mu.Unlock()
	if fd.Triggered {
		e.lastTrigW = now
	}
	pre := 0
	if fd.DT > 0 {
		pre = int(math.Round(-fd.TFirst / fd.DT))
	}
	f := &Frame{Data: fd.Codes, DT: fd.DT, TFirst: fd.TFirst, Triggered: fd.Triggered,
		Scales: fd.Scales, Ranges: fd.Ranges, PreLen: pre}
	var hist [maxCh][]float32
	var avg [maxCh][]float64
	a := e.applied
	if a.Persist || a.Average {
		for c := 0; c < maxCh; c++ {
			d := fd.Codes[c]
			if d == nil {
				continue
			}
			if a.Persist {
				h := make([]float32, persistCols*histBins)
				histAddF32(h, d, fd.Ranges[c])
				hist[c] = h
			}
			if a.Average {
				s := make([]float64, len(d))
				for i, v := range d {
					s[i] = float64(v)
				}
				avg[c] = s
			}
		}
	}
	e.publish(f, 1, hist, avg)
	if e.single && fd.Triggered {
		e.single = false
		e.running = false
		e.stopStream()
		e.setState("stopped", nil)
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minI64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
