// The display side: what the Python version did in its 60 fps window timer.
//
// The interface polls /api/frame once per animation frame. Each poll takes the
// newest data from the engine, folds every captured waveform into averaging
// and persistence, converts volts to screen divisions and returns compact
// binary arrays that the page just draws. Measurements, the status line and
// auto setup run here too, so the page only has to look after controls.
package scopie

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

var chColors = []string{"#ffd400", "#2fd5ff", "#ff5ad2", "#4f8dff", "#c38bff"}

type View struct {
	mu    sync.Mutex
	eng   *Engine
	hub   *Hub
	store *settingsStore

	s       Settings
	info    *DeviceInfo
	infoVer int
	nch     int

	frame        *Frame
	imported     bool
	importedName string
	lastSeq      int64
	gen          int64
	avg          [maxCh][]float64
	hist         [maxCh][]float32
	histMeta     [maxCh]CodeRange
	volts        [maxCh + 1][]float64
	voltsSeq     int64
	clipped      [maxCh]bool
	state        string
	message      string
	connected    bool
	running      bool
	singleArmed  bool
	dirty        bool
	renderSeq    int64
	lastTick     time.Time
	lastPoll     time.Time
	autosetting  bool
	waveforms    int64
	bytesIn      int64
	statT        time.Time
	statW, statB int64
	wfmRate      float64
	mbps         float64
	polls        int
	pollT        time.Time
	fps          float64
	meas         map[string][maxCh + 1]string
	fft          *fftResult
	startupDone  bool
}

type fftResult struct {
	f, db         []float64
	peakF, peakDB float64
	color         string
}

func newView(eng *Engine, hub *Hub, store *settingsStore) *View {
	v := &View{eng: eng, hub: hub, store: store, state: "disconnected", running: true, dirty: true,
		lastTick: time.Now(), statT: time.Now(), pollT: time.Now()}
	v.s = store.load()
	v.applyInfoLocked(notConnectedInfo(), true)
	v.applyLocked(false)
	return v
}

func notConnectedInfo() *DeviceInfo {
	return defaultInfo("Not connected")
}

// ---------------------------------------------------------------- settings -> engine

func (v *View) hwACAvailable() bool {
	return v.info.HWAC || (v.info.ACDCModOption && v.s.HWACDC)
}

func (v *View) softAC(c int) bool { return v.s.Ch[c].Coupling == "AC" && !v.hwACAvailable() }

func (v *View) chOn(c int) bool { return c < v.nch && v.s.Ch[c].On }

func (v *View) hx() float64 { return float64(v.info.HDivs) / 2 }

func (v *View) isRoll() bool {
	return v.info.RollMode && v.s.Roll && v.s.TDiv >= rollThreshold && !v.s.XY
}

func (v *View) persistTime() float64 {
	if v.s.Persist < 0 {
		return math.Inf(1)
	}
	return v.s.Persist
}

func (v *View) computePanel() Panel {
	s := &v.s
	p := defaultPanel()
	for c := 0; c < maxCh; c++ {
		p.Ch[c] = ChannelPanel{Enabled: v.chOn(c), VDiv: s.Ch[c].VDiv, Pos: s.Ch[c].Pos, Coupling: s.Ch[c].Coupling}
	}
	p.TDiv, p.TrigX, p.Rate = s.TDiv, s.TrigX, s.Rate
	p.TrigSource = clampInt(s.TrigSource, 0, v.nch-1)
	p.TrigEdge, p.TrigMode = s.TrigEdge, s.TrigMode
	p.TrigLevel = s.TrigLevel / s.probe(p.TrigSource)
	p.Holdoff = s.Holdoff * 1e-6
	p.NoiseReject = s.NoiseReject
	p.Roll = v.isRoll()
	p.Persist = s.Persist != 0
	p.Average = s.Average > 1
	p.CalFreq = s.CalFreq
	p.HWACDCMod = s.HWACDC
	need := map[int]bool{p.TrigSource: true}
	for c := 0; c < v.nch; c++ {
		if v.chOn(c) {
			need[c] = true
		}
	}
	if s.MathOn {
		need[s.MathA], need[s.MathB] = true, true
	}
	if s.XY {
		need[0], need[1] = true, true
	}
	if s.FFTOn && s.FFTSource < v.nch {
		need[s.FFTSource] = true
	}
	p.ChannelsNeeded = nil
	for c := range need {
		if c < v.nch {
			p.ChannelsNeeded = append(p.ChannelsNeeded, c)
			p.Ch[c].Enabled = true // needed channels must be captured
		}
	}
	sort.Ints(p.ChannelsNeeded)
	return p
}

// applyLocked sends the settings to the engine, saves them and (if the change
// came from Scopie rather than the page) tells the page.
func (v *View) applyLocked(broadcast bool) {
	s := &v.s
	src := clampInt(s.TrigSource, 0, v.nch-1)
	fs := (4 + math.Abs(s.Ch[src].Pos)) * s.vdivTip(src)
	s.TrigLevel = clampF(s.TrigLevel, -2*fs, 2*fs)
	v.eng.Apply(v.computePanel())
	v.pushDemoLocked()
	v.dirty = true
	if broadcast {
		s.Rev++
		v.hub.publish("settings", s)
	}
	v.store.save(*s)
}

func (v *View) pushDemoLocked() {
	sigs := make([]string, maxCh)
	probes := make([]float64, maxCh)
	for c := range sigs {
		sigs[c], probes[c] = v.s.Ch[c].Demo, v.s.Ch[c].Probe
	}
	v.eng.SetDemo(sigs, probes)
}

// SetSettings is called when the page changed a control.
func (v *View) SetSettings(ns Settings) Settings {
	v.mu.Lock()
	defer v.mu.Unlock()
	ns.sanitize()
	old := v.s
	ns.Rev = old.Rev
	// a change of averaging or persistence starts them afresh
	if ns.Persist != old.Persist || ns.Average != old.Average {
		v.clearLocked()
	}
	v.s = ns
	v.snapToDeviceLocked()
	v.applyLocked(false)
	return v.s
}

func (v *View) Settings() Settings {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.s
}

// snapToDeviceLocked moves every value onto one the connected scope supports.
func (v *View) snapToDeviceLocked() {
	s, info := &v.s, v.info
	vd := info.VDivs
	if len(vd) == 0 {
		vd = defaultInfo("").VDivs
	}
	for c := range s.Ch {
		s.Ch[c].VDiv = vd[nearestIndex(vd, s.Ch[c].VDiv)]
	}
	td := info.TDivs
	if len(td) == 0 {
		td = defaultInfo("").TDivs
	}
	s.TDiv = td[nearestIndex(td, s.TDiv)]
	if info.Kind != "stream" || len(info.SampleRates) == 0 {
		s.Rate = 0
	} else if s.Rate > 0 {
		s.Rate = info.SampleRates[nearestIndex(info.SampleRates, s.Rate)]
	}
	s.TrigSource = clampInt(s.TrigSource, 0, v.nch-1)
	s.MathA = clampInt(s.MathA, 0, v.nch-1)
	s.MathB = clampInt(s.MathB, 0, v.nch-1)
	if s.FFTSource < maxCh && s.FFTSource >= v.nch {
		s.FFTSource = 0
	}
	if s.CursorSource < maxCh && s.CursorSource >= v.nch {
		s.CursorSource = 0
	}
	if !info.TriggerBothEdges && s.TrigEdge == "both" {
		s.TrigEdge = "rising"
	}
	if info.CalOutput && len(info.CalFreqs) > 0 {
		s.CalFreq = info.CalFreqs[nearestIndex(info.CalFreqs, s.CalFreq)]
	}
	hx := float64(info.HDivs) / 2
	s.TrigX = clampF(s.TrigX, -hx, hx)
	for i := 0; i < 2; i++ {
		s.Cursors[i] = clampF(s.Cursors[i], -hx, hx)
	}
}

func (v *View) applyInfoLocked(info *DeviceInfo, keep bool) {
	v.info = info
	v.nch = clampInt(info.Channels, 1, maxCh)
	if !keep {
		v.s.Rate = 0
	}
	v.snapToDeviceLocked()
	v.infoVer++
}

// ---------------------------------------------------------------- taking data

// update takes the newest data from the engine. Called for every page poll,
// and by a timer while the window is hidden so measurements stay live.
func (v *View) update() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.updateLocked()
}

func (v *View) updateLocked() {
	t := v.eng.Take()
	if t.Error != "" {
		v.connected = false
		v.hub.publish("error", map[string]string{"message": t.Error})
	}
	if t.Info != nil && t.Info != v.info {
		v.applyInfoLocked(t.Info, false)
		v.applyLocked(true)
		v.hub.publish("info", v.infoPayloadLocked())
	}
	if t.State != v.state {
		v.stateChangedLocked(t.State, &t)
	}
	v.message = t.Message
	v.waveforms, v.bytesIn = t.Waveforms, t.Bytes
	if t.Gen != v.gen {
		v.gen = t.Gen
		v.avg = [maxCh][]float64{}
		v.hist = [maxCh][]float32{}
	}
	newFrame := t.Frame != nil && t.Frame.Seq != v.lastSeq && !v.imported
	if newFrame {
		v.frame = t.Frame
		v.lastSeq = t.Frame.Seq
	}
	now := time.Now()
	dtTick := now.Sub(v.lastTick).Seconds()
	v.lastTick = now

	if t.AvgN > 0 {
		n := float64(v.s.Average)
		if n < 1 {
			n = 1
		}
		w := math.Pow(1-1/n, float64(t.AvgN))
		for c := 0; c < maxCh; c++ {
			sum := t.AvgSum[c]
			if sum == nil {
				continue
			}
			inv := 1 / float64(t.AvgN)
			if v.avg[c] == nil || len(v.avg[c]) != len(sum) {
				a := make([]float64, len(sum))
				for i, x := range sum {
					a[i] = x * inv
				}
				v.avg[c] = a
			} else {
				a := v.avg[c]
				for i, x := range sum {
					a[i] = a[i]*w + x*inv*(1-w)
				}
			}
		}
	}
	pt := v.persistTime()
	persisting := pt > 0 && !v.isRoll() && !v.imported
	if persisting {
		decay := 1.0
		if !math.IsInf(pt, 1) {
			decay = math.Exp(-dtTick / pt)
		}
		for c := 0; c < maxCh; c++ {
			h := t.Hist[c]
			if v.hist[c] != nil {
				if h != nil && (len(h) != len(v.hist[c]) || t.HistMeta[c] != v.histMeta[c]) {
					v.hist[c] = nil
				} else if decay < 1 {
					d := float32(decay)
					for i := range v.hist[c] {
						v.hist[c][i] *= d
					}
				}
			}
			if h != nil {
				if v.hist[c] == nil {
					v.hist[c] = append([]float32(nil), h...)
				} else {
					for i, x := range h {
						v.hist[c][i] += x
					}
				}
				v.histMeta[c] = t.HistMeta[c]
			}
		}
	}
	if newFrame || v.dirty || persisting {
		v.computeVoltsLocked()
		v.renderSeq++
		v.dirty = false
	}
	// status figures
	if el := now.Sub(v.statT).Seconds(); el >= 0.5 {
		v.wfmRate = float64(v.waveforms-v.statW) / el
		v.mbps = float64(v.bytesIn-v.statB) / el / 1e6
		v.fps = float64(v.polls) / now.Sub(v.pollT).Seconds()
		v.statT, v.statW, v.statB = now, v.waveforms, v.bytesIn
		v.polls, v.pollT = 0, now
	}
}

func (v *View) stateChangedLocked(state string, t *Taken) {
	old := v.state
	v.state = state
	switch state {
	case "running", "stopped", "waiting":
		v.connected = true
	case "disconnected":
		v.connected = false
		v.applyInfoLocked(notConnectedInfo(), true)
		v.applyLocked(true)
		v.hub.publish("info", v.infoPayloadLocked())
	}
	if state == "stopped" && v.singleArmed && (old == "running" || old == "waiting") {
		v.singleArmed = false
		v.running = false
	}
}

// frameVolts gives volts at the probe tip for channel c (averaged if enabled).
func (v *View) frameVolts(c int) []float64 {
	f := v.frame
	if f == nil || f.Data[c] == nil {
		return nil
	}
	raw := f.Data[c]
	sc := f.Scales[c]
	probe := v.s.probe(c)
	if v.imported {
		probe = 1
	}
	k := sc.K * probe
	out := make([]float64, len(raw))
	if v.s.Average > 1 && !v.imported && v.avg[c] != nil && len(v.avg[c]) == len(raw) {
		for i, a := range v.avg[c] {
			out[i] = (a - sc.Zero) * k
		}
	} else {
		for i, x := range raw {
			out[i] = (float64(x) - sc.Zero) * k
		}
	}
	if v.softAC(c) && !v.imported && len(out) > 0 {
		m := mean(out)
		for i := range out {
			out[i] -= m
		}
	}
	return out
}

func (v *View) computeVoltsLocked() {
	if v.frame == nil {
		v.volts = [maxCh + 1][]float64{}
		return
	}
	for c := 0; c < maxCh; c++ {
		v.volts[c] = v.frameVolts(c)
	}
	v.volts[maxCh] = nil
	if v.s.MathOn {
		a, b := v.volts[v.s.MathA], v.volts[v.s.MathB]
		if a != nil && b != nil && len(a) == len(b) {
			m := make([]float64, len(a))
			for i := range a {
				switch v.s.MathOp {
				case 0:
					m[i] = a[i] + b[i]
				case 1:
					m[i] = a[i] - b[i]
				default:
					m[i] = a[i] * b[i]
				}
			}
			v.volts[maxCh] = m
		}
	}
	f := v.frame
	for c := 0; c < maxCh; c++ {
		clip := false
		if d := f.Data[c]; d != nil && v.chOn(c) && !v.imported && len(d) > 0 {
			lo, hi := float32(f.Ranges[c].Lo), float32(f.Ranges[c].Hi)
			for _, x := range d {
				if x <= lo || x >= hi {
					clip = true
					break
				}
			}
		}
		v.clipped[c] = clip
	}
	v.voltsSeq++
}

func (v *View) clearLocked() {
	v.avg = [maxCh][]float64{}
	v.hist = [maxCh][]float32{}
	v.dirty = true
}

// ---------------------------------------------------------------- binary frame payload

type payload struct {
	hdr  map[string]any
	blob bytes.Buffer
}

func (p *payload) f32(vals []float64) int {
	off := p.blob.Len()
	b := make([]byte, 4*len(vals))
	for i, x := range vals {
		binary.LittleEndian.PutUint32(b[4*i:], math.Float32bits(float32(x)))
	}
	p.blob.Write(b)
	return off
}

func (p *payload) u8(b []byte) int {
	off := p.blob.Len()
	p.blob.Write(b)
	for p.blob.Len()%4 != 0 {
		p.blob.WriteByte(0)
	}
	return off
}

// bytes lays out: u32 header length, JSON header, padding to 4, data blocks.
func (p *payload) bytes() []byte {
	h, _ := json.Marshal(p.hdr)
	pad := (4 - (4+len(h))%4) % 4
	out := make([]byte, 0, 4+len(h)+pad+p.blob.Len())
	var n [4]byte
	binary.LittleEndian.PutUint32(n[:], uint32(len(h)+pad))
	out = append(out, n[:]...)
	out = append(out, h...)
	for i := 0; i < pad; i++ {
		out = append(out, ' ')
	}
	return append(out, p.blob.Bytes()...)
}

// Frame returns the drawing for the page, or nil if nothing changed since have.
func (v *View) Frame(have int64) []byte {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.lastPoll = time.Now()
	v.polls++
	v.updateLocked()
	if have == v.renderSeq {
		return nil
	}
	return v.renderLocked()
}

func (v *View) renderLocked() []byte {
	p := &payload{hdr: map[string]any{"seq": v.renderSeq}}
	f := v.frame
	s := &v.s
	chip, chipCol := v.stateChipLocked()
	p.hdr["chip"] = chip
	p.hdr["chipColor"] = chipCol
	p.hdr["clipped"] = v.clipped
	p.hdr["hdivs"] = v.info.HDivs
	if f == nil {
		return p.bytes()
	}
	n := len(f.first())
	p.hdr["frame"] = map[string]any{"rate": 1 / f.DT, "points": n, "triggered": f.Triggered, "roll": f.Roll, "file": v.imported}
	hx := v.hx()
	if s.XY {
		x0, y0 := v.volts[0], v.volts[1]
		if x0 != nil && y0 != nil && len(x0) == len(y0) {
			step := 1
			if len(x0) > 20000 {
				step = len(x0)/20000 + 1
			}
			var xs, ys []float64
			for i := 0; i < len(x0); i += step {
				xs = append(xs, x0[i]*s.sign(0)/s.vdivTip(0)+s.Ch[0].Pos)
				ys = append(ys, y0[i]*s.sign(1)/s.vdivTip(1)+s.Ch[1].Pos)
			}
			p.hdr["xy"] = map[string]any{"n": len(xs), "x": p.f32(xs), "y": p.f32(ys)}
		}
		return p.bytes()
	}
	pt := v.persistTime()
	persisting := pt > 0 && !f.Roll && !v.imported
	dots := s.Style == "dots"
	sinc := s.Interp == "sinc"
	var traces, images []map[string]any
	for c := 0; c <= maxCh; c++ {
		vv := v.volts[c]
		show := vv != nil
		if c < maxCh {
			show = show && v.chOn(c)
		} else {
			show = show && s.MathOn
		}
		if !show {
			continue
		}
		if c < maxCh && persisting && v.hist[c] != nil {
			if img := v.persistImageLocked(c, p); img != nil {
				images = append(images, img)
				continue
			}
		}
		nn := len(vv)
		x := make([]float64, nn)
		y := make([]float64, nn)
		for i := range vv {
			t := f.TFirst + float64(i)*f.DT
			if f.Roll {
				x[i] = hx + t/s.TDiv
			} else {
				x[i] = t/s.TDiv + s.TrigX
			}
			if c < maxCh {
				y[i] = vv[i]*s.sign(c)/s.vdivTip(c) + s.Ch[c].Pos
			} else {
				y[i] = vv[i]/s.MathVDiv + s.MathPos
			}
		}
		var xx, yy []float64
		switch {
		case dots:
			xx, yy = decimateXY(x, y, 6000)
		case sinc && nn >= 4 && nn < 500:
			factor := minInt(32, int(math.Ceil(2000/float64(nn))))
			yy = sincInterpolate(y, factor)
			xx = make([]float64, len(yy))
			for i := range xx {
				xx[i] = x[0] + (x[nn-1]-x[0])*float64(i)/float64(len(yy)-1)
			}
		case nn > 4000:
			xx, yy = decimateXY(x, y, 4000)
		default:
			xx, yy = x, y
		}
		traces = append(traces, map[string]any{"c": c, "n": len(xx), "x": p.f32(xx), "y": p.f32(yy), "dots": dots})
	}
	p.hdr["traces"] = traces
	p.hdr["persist"] = images
	if s.FFTOn && v.fft != nil {
		p.hdr["fft"] = map[string]any{"n": len(v.fft.f), "f": p.f32(v.fft.f), "db": p.f32(v.fft.db),
			"peakF": v.fft.peakF, "peakDb": v.fft.peakDB, "color": v.fft.color}
	}
	return p.bytes()
}

// decimateXY keeps every peak: (min,max) pairs of y with the x of each block's ends.
func decimateXY(x, y []float64, max int) ([]float64, []float64) {
	yy, step := peakDecimate(y, max)
	if step == 1 {
		return x, y
	}
	xx := make([]float64, len(yy))
	for p := 0; p < len(yy)/2; p++ {
		xx[2*p], xx[2*p+1] = x[p*step], x[(p+1)*step-1]
	}
	return xx, yy
}

// persistImageLocked renders the persistence histogram as an 8-bit image and
// says where it goes on screen (in divisions).
func (v *View) persistImageLocked(c int, p *payload) map[string]any {
	h := v.hist[c]
	meta := v.histMeta[c]
	f := v.frame
	s := &v.s
	mx := float32(0)
	for _, x := range h {
		if x > mx {
			mx = x
		}
	}
	if mx <= 0 || f.Data[c] == nil {
		return nil
	}
	img := make([]byte, len(h))
	inv := 255 / math.Log1p(float64(mx))
	for i, x := range h {
		if x > 0 {
			img[i] = byte(math.Min(255, math.Log1p(float64(x))*inv+0.5))
		}
	}
	n := len(f.Data[c])
	cols, bins := persistCols, histBins
	var sx, x0 float64
	if n >= cols {
		sx = float64(n) / float64(cols) * f.DT / s.TDiv
		x0 = f.TFirst/s.TDiv + s.TrigX
	} else {
		sx = float64(n-1) / float64(maxInt(1, cols-1)) * f.DT / s.TDiv
		x0 = f.TFirst/s.TDiv + s.TrigX - sx/2
	}
	span := meta.Hi - meta.Lo + 1
	sc := f.Scales[c]
	perCode := sc.K * s.probe(c) * s.sign(c) / s.vdivTip(c)
	sy := perCode * span / float64(bins)
	ac := 0.0
	if v.softAC(c) {
		m := 0.0
		for _, x := range f.Data[c] {
			m += float64(x)
		}
		m /= float64(n)
		ac = (m - sc.Zero) * perCode
	}
	y0 := (meta.Lo-sc.Zero)*perCode + s.Ch[c].Pos - ac
	return map[string]any{"c": c, "off": p.u8(img), "cols": cols, "bins": bins, "x0": x0, "sx": sx, "y0": y0, "sy": sy}
}

func (v *View) stateChipLocked() (string, string) {
	f := v.frame
	switch {
	case v.autosetting:
		return "AUTO SETUP…", "#ffffff"
	case v.imported:
		return "FILE", "#b0b0ff"
	case v.state == "disconnected":
		return "NO SCOPE", "#888888"
	case v.state == "connecting":
		return "CONNECTING", "#cccccc"
	case v.state == "stopped" || !v.running:
		return "STOP", "#ff4d4d"
	case v.state == "waiting":
		return "WAIT", "#ffa64d"
	case f != nil && f.Roll:
		return "ROLL", "#4dc3ff"
	case f != nil && f.Triggered:
		return "TRIG'D", "#4dff88"
	}
	return "AUTO", "#ffe14d"
}

// ---------------------------------------------------------------- measurements, FFT

// updateMeasurements runs on the status timer, outside the lock, so a long
// waveform never delays drawing. The volts slices are never modified once
// made, so a snapshot of them is safe to read.
func (v *View) updateMeasurements() {
	v.mu.Lock()
	f := v.frame
	volts := v.volts
	var on [maxCh + 1]bool
	for c := 0; c < maxCh; c++ {
		on[c] = v.chOn(c)
	}
	on[maxCh] = v.s.MathOn
	keys := append([]string(nil), v.s.Meas...)
	mathMul := v.s.MathOp == 2
	v.mu.Unlock()
	out := map[string][maxCh + 1]string{}
	if f == nil {
		v.mu.Lock()
		v.meas = out
		v.mu.Unlock()
		return
	}
	var res [maxCh + 1]Meas
	for c := 0; c <= maxCh; c++ {
		if volts[c] != nil && on[c] {
			res[c] = measure(volts[c], f.DT)
		}
	}
	d, ph := phaseDelay(volts[0], volts[1], f.DT)
	units := map[string]string{"phase": "°", "delay": "s"}
	for _, m := range measurements {
		units[m.Key] = m.Unit
	}
	for _, k := range keys {
		var row [maxCh + 1]string
		for c := 0; c <= maxCh; c++ {
			if k == "phase" || k == "delay" {
				if c != 1 {
					continue
				}
				val := d
				if k == "phase" {
					val = ph
				}
				switch {
				case math.IsNaN(val):
					row[c] = "--"
				case k == "phase":
					row[c] = fmt.Sprintf("%+.1f°", val)
				default:
					row[c] = fmtSI(val, "s", 4)
				}
				continue
			}
			if res[c] == nil {
				continue
			}
			val := res[c][k]
			unit := units[k]
			if c == maxCh && mathMul && unit == "V" {
				unit = "V²"
			}
			switch {
			case (k == "rise" || k == "fall") && !math.IsNaN(val) && val <= f.DT*1.01:
				row[c] = "< " + fmtSI(f.DT, "s", 3)
			case k == "duty":
				if math.IsNaN(val) {
					row[c] = "--"
				} else {
					row[c] = fmt.Sprintf("%.1f %%", val)
				}
			default:
				row[c] = fmtSI(val, unit, 4)
			}
		}
		out[k] = row
	}
	v.mu.Lock()
	v.meas = out
	v.mu.Unlock()
}

// updateFFT runs on its own timer, outside the lock (see updateMeasurements).
func (v *View) updateFFT() {
	v.mu.Lock()
	if v.frame == nil || !v.s.FFTOn {
		v.fft = nil
		v.mu.Unlock()
		return
	}
	key := v.s.FFTSource
	if key >= v.nch {
		key = maxCh
	}
	vv, dt, win := v.volts[key], v.frame.DT, v.s.FFTWindow
	v.mu.Unlock()
	if vv == nil {
		v.mu.Lock()
		v.fft = nil
		v.mu.Unlock()
		return
	}
	db, df := spectrum(vv, dt, win, 65536)
	if len(db) < 3 {
		v.mu.Lock()
		v.fft = nil
		v.mu.Unlock()
		return
	}
	db = db[1:] // skip DC
	pi := 0
	for i, x := range db {
		if x > db[pi] {
			pi = i
		}
	}
	r := &fftResult{peakF: float64(pi+1) * df, peakDB: db[pi], color: chColors[key]}
	// reduce to at most 4096 points, keeping the peaks
	const maxPts = 4096
	if len(db) <= maxPts {
		r.f = make([]float64, len(db))
		r.db = db
		for i := range db {
			r.f[i] = float64(i+1) * df
		}
	} else {
		step := float64(len(db)) / maxPts
		for p := 0; p < maxPts; p++ {
			a, b := int(float64(p)*step), int(float64(p+1)*step)
			best := a
			for i := a; i < b && i < len(db); i++ {
				if db[i] > db[best] {
					best = i
				}
			}
			r.f = append(r.f, float64(best+1)*df)
			r.db = append(r.db, db[best])
		}
	}
	v.mu.Lock()
	v.fft = r
	v.renderSeq++ // the page should pick up the new spectrum
	v.mu.Unlock()
}

// fmtSI formats a number with an SI prefix: 0.0012 V -> "1.2 mV".
func fmtSI(value float64, unit string, digits int) string {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return "--"
	}
	if value == 0 {
		return strings.TrimSpace("0 " + unit)
	}
	prefixes := []struct {
		s float64
		p string
	}{{1e9, "G"}, {1e6, "M"}, {1e3, "k"}, {1, ""}, {1e-3, "m"}, {1e-6, "µ"}, {1e-9, "n"}, {1e-12, "p"}}
	a := math.Abs(value)
	scale, p := 1e-12, "p"
	for _, x := range prefixes {
		if a >= x.s*0.9995 {
			scale, p = x.s, x.p
			break
		}
	}
	// %.Ng matches Python's "{:.Ng}"
	return strings.TrimSpace(fmt.Sprintf("%.*g", digits, value/scale) + " " + p + unit)
}

// ---------------------------------------------------------------- status

type statusMsg struct {
	State     string                       `json:"state"`
	Message   string                       `json:"message"`
	Connected bool                         `json:"connected"`
	Running   bool                         `json:"running"`
	Imported  string                       `json:"imported"`
	Device    string                       `json:"device"`
	Notes     string                       `json:"notes"`
	WfmRate   float64                      `json:"wfmRate"`
	FPS       float64                      `json:"fps"`
	MBps      float64                      `json:"mbps"`
	Meas      map[string][maxCh + 1]string `json:"meas"`
	Clipped   [maxCh]bool                  `json:"clipped"`
	Autoset   bool                         `json:"autoset"`
	Applied   string                       `json:"applied"`
}

func (v *View) status() statusMsg {
	v.mu.Lock()
	defer v.mu.Unlock()
	// keep things moving while the window is hidden (no polls)
	if time.Since(v.lastPoll) > 300*time.Millisecond {
		v.updateLocked()
	}
	st := statusMsg{State: v.state, Message: v.message, Connected: v.connected, Running: v.running,
		Device: v.info.Name, Notes: v.info.Notes, WfmRate: v.wfmRate, FPS: v.fps, MBps: v.mbps,
		Meas: v.meas, Clipped: v.clipped, Autoset: v.autosetting}
	if v.imported {
		st.Imported = v.importedName
	}
	if !v.connected && v.state != "connecting" {
		st.Device = "Not connected"
	}
	if time.Since(v.lastPoll) > time.Second { // window hidden: nothing is being drawn
		st.FPS = 0
	}
	return st
}

func (v *View) infoPayloadLocked() map[string]any {
	return map[string]any{"info": v.info, "connected": v.connected, "ver": v.infoVer}
}

func (v *View) InfoPayload() map[string]any {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.infoPayloadLocked()
}

// ---------------------------------------------------------------- actions

func (v *View) ToggleRun(want *bool) bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	run := !v.running
	if want != nil {
		run = *want
	}
	v.running = run
	v.singleArmed = false
	if run {
		v.imported = false
		v.eng.Run()
	} else {
		v.eng.StopAcq()
	}
	v.dirty = true
	return run
}

func (v *View) Single() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.running, v.singleArmed, v.imported = true, true, false
	v.clearLocked()
	v.eng.Single()
}

func (v *View) Clear() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.clearLocked()
}

func (v *View) DefaultSetup() {
	v.mu.Lock()
	defer v.mu.Unlock()
	d := defaultSettings()
	d.LAN, d.LastDevice, d.Rev = v.s.LAN, v.s.LastDevice, v.s.Rev
	v.s = d
	v.snapToDeviceLocked()
	v.clearLocked()
	v.applyLocked(true)
}

// Trigger50 puts the trigger level halfway between the signal's extremes.
func (v *View) Trigger50() {
	v.mu.Lock()
	defer v.mu.Unlock()
	src := clampInt(v.s.TrigSource, 0, v.nch-1)
	vv := v.volts[src]
	if len(vv) == 0 {
		return
	}
	lo, hi := minMax(vv)
	v.s.TrigLevel = (lo + hi) / 2
	v.applyLocked(true)
}

func (v *View) Connect(c *Candidate) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.imported = false
	if c.Family != "Demo mode" {
		v.s.LastDevice = c.Label
		v.store.save(v.s)
	}
	v.pushDemoLocked()
	v.eng.Connect(c.Factory)
	if v.running {
		v.eng.Run()
	}
}

func (v *View) Disconnect() { v.eng.Disconnect() }

// ---------------------------------------------------------------- auto setup

func (v *View) Autoset() string {
	v.mu.Lock()
	if !v.connected {
		v.mu.Unlock()
		return "Auto setup needs a connected scope (or demo mode)."
	}
	if v.autosetting {
		v.mu.Unlock()
		return ""
	}
	if !v.running {
		v.running, v.singleArmed, v.imported = true, false, false
		v.eng.Run()
	}
	s := &v.s
	vd := v.info.VDivs
	for c := 0; c < v.nch; c++ {
		want := 1.0
		if p := s.probe(c); p > 1 {
			want = 1 / p
		}
		s.Ch[c].VDiv = vd[nearestIndex(vd, want)]
		s.Ch[c].Pos = 0
	}
	s.TDiv = v.info.TDivs[nearestIndex(v.info.TDivs, 1e-3)]
	s.Rate, s.TrigX, s.TrigMode, s.XY = 0, 0, "auto", false
	v.autosetting = true
	v.applyLocked(true)
	frameKind := v.info.Kind == "frame"
	v.mu.Unlock()
	go v.autosetRun(frameKind)
	return ""
}

func (v *View) autosetRun(frameKind bool) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("autoset panic: %v", r)
		}
		v.mu.Lock()
		v.autosetting = false
		v.dirty = true
		v.mu.Unlock()
	}()
	limit := 12
	if frameKind {
		limit = 40
	}
	wait := func(d time.Duration) { time.Sleep(d) }
	setT := func(t float64) {
		v.mu.Lock()
		v.s.TDiv = v.info.TDivs[nearestIndex(v.info.TDivs, t)]
		v.applyLocked(true)
		v.mu.Unlock()
	}
	wait(400 * time.Millisecond)
	step := "measure1"
	for {
		stats := v.autosetStats(limit)
		if stats == nil {
			return // disconnected meanwhile
		}
		if step == "measure1" {
			f := bestFreq(stats)
			if math.IsNaN(f) || f < 150 {
				step = "slow"
				setT(20e-3)
				wait(500 * time.Millisecond)
				continue
			}
			if f > 40e3 {
				step = "fast"
				setT(10e-6)
				wait(300 * time.Millisecond)
				continue
			}
		}
		v.autosetFinish(stats)
		return
	}
}

// autosetStats waits (up to limit tries) for a waveform captured with the
// newest settings, then measures every visible channel.
func (v *View) autosetStats(limit int) map[int]Meas {
	for tries := 0; ; tries++ {
		v.mu.Lock()
		if !v.connected {
			v.mu.Unlock()
			return nil
		}
		v.updateLocked()
		fresh := v.frame != nil && !v.imported && v.frame.Gen == v.eng.CurrentGen()
		if fresh || tries >= limit {
			stats := map[int]Meas{}
			if v.frame != nil {
				for c := 0; c < v.nch; c++ {
					if v.chOn(c) && len(v.volts[c]) >= 8 {
						stats[c] = measure(v.volts[c], v.frame.DT)
					}
				}
			}
			v.mu.Unlock()
			return stats
		}
		v.mu.Unlock()
		time.Sleep(150 * time.Millisecond)
	}
}

func bestFreq(stats map[int]Meas) float64 {
	keys := make([]int, 0, len(stats))
	for c := range stats {
		keys = append(keys, c)
	}
	sort.Ints(keys)
	for _, c := range keys {
		m := stats[c]
		if !math.IsNaN(m["freq"]) && m["vpp"] > 0.05 {
			return m["freq"]
		}
	}
	return nan
}

func (v *View) autosetFinish(stats map[int]Meas) {
	v.mu.Lock()
	defer v.mu.Unlock()
	s := &v.s
	vd := v.info.VDivs
	var chans []int
	for c := range stats {
		chans = append(chans, c)
	}
	sort.Ints(chans)
	var active []int
	for _, c := range chans {
		if stats[c]["vpp"] > 0 {
			active = append(active, c)
		}
	}
	slots := map[int]float64{}
	for i, c := range active {
		slots[c] = float64(len(active)-1-2*i) * (3 / float64(maxInt(1, len(active))))
	}
	bestSrc, bestVpp := 0, -1.0
	for _, c := range chans {
		m := stats[c]
		vpp := m["vpp"]
		p := s.probe(c)
		if vpp < 0.02*p {
			continue
		}
		perTrace := 6.0
		if len(active) >= 2 {
			perTrace = math.Max(2, 7/float64(len(active)))
		}
		choice := len(vd) - 1
		for i, x := range vd {
			if vpp <= perTrace*x*p {
				choice = i
				break
			}
		}
		s.Ch[c].VDiv = vd[choice]
		vdt := vd[choice] * p
		mid := 0.0
		if s.Ch[c].Coupling != "AC" {
			mid = (m["vmax"] + m["vmin"]) / 2
		}
		target := 0.0
		if len(active) > 1 {
			target = slots[c]
		}
		s.Ch[c].Pos = clampF(target-mid*s.sign(c)/vdt, -4, 4)
		s.Ch[c].Pos = math.Round(s.Ch[c].Pos*100) / 100
		if vpp > bestVpp {
			bestSrc, bestVpp = c, vpp
		}
	}
	if f := bestFreq(stats); !math.IsNaN(f) {
		want := 0.25 / f
		td := v.info.TDivs
		idx := len(td) - 1
		for i, t := range td {
			if t >= want {
				idx = i
				break
			}
		}
		s.TDiv = td[idx]
	}
	s.TrigSource, s.TrigEdge = bestSrc, "rising"
	if m, ok := stats[bestSrc]; ok {
		mid := 0.0
		if s.Ch[bestSrc].Coupling != "AC" {
			mid = (m["vmax"] + m["vmin"]) / 2
		}
		s.TrigLevel = mid
	}
	v.autosetting = false
	v.clearLocked()
	v.applyLocked(true)
	v.hub.publish("help", map[string]string{"html": "<b>Auto setup finished.</b> Fine-tune volts/div and time/div if you want, or hover over any control to learn more."})
}

// ---------------------------------------------------------------- files

// ExportCSV writes the waveforms on screen as "time, ch1, ch2, …" columns.
func (v *View) ExportCSV() ([]byte, int, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	f := v.frame
	if f == nil {
		return nil, 0, devErr("There is no waveform to export yet.")
	}
	type col struct {
		name string
		d    []float64
	}
	var cols []col
	for c := 0; c < maxCh; c++ {
		if v.volts[c] != nil && v.chOn(c) {
			cols = append(cols, col{fmt.Sprintf("ch%d_V", c+1), v.volts[c]})
		}
	}
	if v.volts[maxCh] != nil && v.s.MathOn {
		cols = append(cols, col{"math", v.volts[maxCh]})
	}
	if len(cols) == 0 {
		return nil, 0, devErr("No channel is switched on.")
	}
	n := len(cols[0].d)
	for _, c := range cols {
		n = minInt(n, len(c.d))
	}
	var b strings.Builder
	b.WriteString("time_s")
	for _, c := range cols {
		b.WriteString("," + c.name)
	}
	b.WriteString("\n")
	for i := 0; i < n; i++ {
		b.WriteString(fmt.Sprintf("%.9g", f.TFirst+float64(i)*f.DT))
		for _, c := range cols {
			b.WriteString(fmt.Sprintf(",%.9g", c.d[i]))
		}
		b.WriteString("\n")
	}
	return []byte(b.String()), n, nil
}

// OpenCSV shows a waveform from a CSV file and picks scales that fit it.
func (v *View) OpenCSV(name string, text []byte) error {
	fr, err := loadCSV(text)
	if err != nil {
		return devErr("Could not read that file:\n%v", err)
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.running {
		v.running, v.singleArmed = false, false
		v.eng.StopAcq()
	}
	v.imported, v.importedName = true, name
	v.frame = fr
	s := &v.s
	nData := 0
	for _, d := range fr.Data {
		if d != nil {
			nData++
		}
	}
	for c := 0; c < maxCh; c++ {
		s.Ch[c].Probe = 1
		if c < nData {
			s.Ch[c].On = true
		}
	}
	first := fr.first()
	span := float64(len(first)) * fr.DT
	td := v.info.TDivs
	s.TDiv = td[nearestIndex(td, math.Max(span/v.hx()/2, 1e-9))]
	vd := v.info.VDivs
	for c := 0; c < minInt(nData, v.nch); c++ {
		d := fr.Data[c]
		lo, hi := 0.0, 1.0
		if len(d) > 0 {
			lo, hi = float64(d[0]), float64(d[0])
			for _, x := range d {
				lo, hi = math.Min(lo, float64(x)), math.Max(hi, float64(x))
			}
		}
		vpp := hi - lo
		idx := len(vd) - 1
		for i, x := range vd {
			if vpp <= 6*x {
				idx = i
				break
			}
		}
		s.Ch[c].VDiv = vd[idx]
		s.Ch[c].Pos = clampF(math.Round(-((hi+lo)/2)/vd[idx]*100)/100, -4, 4)
	}
	s.TrigX = clampF(math.Round(-(fr.TFirst+span/2)/s.TDiv*100)/100, -v.hx(), v.hx())
	v.clearLocked()
	v.applyLocked(true)
	v.computeVoltsLocked()
	v.renderSeq++
	return nil
}

// LoadSetup applies a setup file saved earlier.
func (v *View) LoadSetup(b []byte) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	ns := v.s
	if err := json.Unmarshal(b, &ns); err != nil {
		return devErr("That is not a Scopie setup file (%v).", err)
	}
	ns.sanitize()
	ns.Rev = v.s.Rev
	v.s = ns
	v.snapToDeviceLocked()
	v.clearLocked()
	v.applyLocked(true)
	return nil
}
