package probie

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ---------------------------------------------------------------- capture

type Capture struct {
	ID      int
	Device  string
	Rate    float64
	Data    []byte
	T0      float64 // unix ms of sample 0
	Trig    int     // sample index of the trigger, -1 if none
	Edges   [8][]uint32
	Changes []uint32 // sample indices where any channel changes
	Rows    []*DecRow
	Decoded int
}

func (c *Capture) N() int { return len(c.Data) }

func (c *Capture) index() {
	for ch := range c.Edges {
		c.Edges[ch] = nil
	}
	c.Changes = c.Changes[:0]
	d := c.Data
	for i := 1; i < len(d); i++ {
		x := d[i] ^ d[i-1]
		if x == 0 {
			continue
		}
		c.Changes = append(c.Changes, uint32(i))
		for b := 0; b < 8; b++ {
			if x&(1<<b) != 0 {
				c.Edges[b] = append(c.Edges[b], uint32(i))
			}
		}
	}
}

func (c *Capture) bit(ch int, i int) byte {
	if i < 0 {
		i = 0
	}
	if i >= len(c.Data) {
		i = len(c.Data) - 1
	}
	return (c.Data[i] >> ch) & 1
}

// countLE returns how many edges are at or before sample i.
func countLE(e []uint32, i int) int {
	if i < 0 {
		return 0
	}
	return sort.Search(len(e), func(k int) bool { return int(e[k]) > i })
}

// view summarises [s,e) into w pixel columns per channel. Each byte is
// level | min(transitions,127)<<1, or 0xFF where there's no data.
func (c *Capture) view(s, e float64, w int) []byte {
	out := make([]byte, 8*w)
	n := len(c.Data)
	spp := (e - s) / float64(w)
	for ch := 0; ch < 8; ch++ {
		edges := c.Edges[ch]
		init := int(c.Data[0]>>ch) & 1
		for px := 0; px < w; px++ {
			a := s + float64(px)*spp
			b := a + spp
			ia, ib := int(math.Floor(a)), int(math.Floor(b))
			if b <= 0 || ia >= n {
				out[ch*w+px] = 0xFF
				continue
			}
			if ia < 0 {
				ia = 0
			}
			if ib > n-1 {
				ib = n - 1
			}
			ka := countLE(edges, ia)
			kb := countLE(edges, ib)
			lvl := (init ^ (ka & 1)) & 1
			t := kb - ka
			if t > 127 {
				t = 127
			}
			out[ch*w+px] = byte(lvl | t<<1)
		}
	}
	return out
}

// ---------------------------------------------------------------- trigger + collection

type trigCond struct{ hi, lo, rise, fall, edge byte }

func parseTrig(t []string) (trigCond, bool) {
	var c trigCond
	for i, s := range t {
		if i > 7 {
			break
		}
		b := byte(1 << i)
		switch s {
		case "1":
			c.hi |= b
		case "0":
			c.lo |= b
		case "r":
			c.rise |= b
		case "f":
			c.fall |= b
		case "e":
			c.edge |= b
		}
	}
	return c, c.hi|c.lo|c.rise|c.fall|c.edge != 0
}

func (c trigCond) match(p, s byte) bool {
	return s&c.hi == c.hi && ^s&c.lo == c.lo &&
		(^p&s)&c.rise == c.rise && (p&^s)&c.fall == c.fall && (p^s)&c.edge == c.edge
}

type collector struct {
	n, pre    int
	cond      trigCond
	hasTrig   bool
	buf       []byte
	triggered bool
	trigIdx   int
	prev      byte
	absStart  int64 // absolute sample number of buf[0]
	total     int64 // samples received
}

// feed returns true once n samples (post-trigger window) are collected.
func (c *collector) feed(chunk []byte) bool {
	c.total += int64(len(chunk))
	if !c.hasTrig || c.triggered {
		need := c.n - len(c.buf)
		if need > len(chunk) {
			need = len(chunk)
		}
		c.buf = append(c.buf, chunk[:need]...)
		return len(c.buf) >= c.n
	}
	base := len(c.buf)
	c.buf = append(c.buf, chunk...)
	for i := base; i < len(c.buf); i++ {
		p := c.prev
		if i > 0 {
			p = c.buf[i-1]
		}
		if c.cond.match(p, c.buf[i]) {
			c.triggered = true
			start := i - c.pre
			if start < 0 {
				start = 0
			}
			nb := make([]byte, 0, c.n)
			end := len(c.buf)
			if end-start > c.n {
				end = start + c.n
			}
			nb = append(nb, c.buf[start:end]...)
			c.absStart += int64(start)
			c.trigIdx = i - start
			c.buf = nb
			return len(c.buf) >= c.n
		}
	}
	// Still waiting: keep only the pre-trigger history we might need.
	if len(c.buf) > c.pre+(4<<20) {
		drop := len(c.buf) - c.pre
		c.prev = c.buf[drop-1]
		c.absStart += int64(drop)
		c.buf = append(c.buf[:0:0], c.buf[drop:]...)
	}
	return false
}

// ---------------------------------------------------------------- manager

type LogicState struct {
	Status    string  `json:"status"` // idle, arming, waiting, capturing, processing, done, error
	Message   string  `json:"message,omitempty"`
	Progress  float64 `json:"progress"`
	Error     string  `json:"error,omitempty"`
	CaptureID int     `json:"captureId"`
	Device    string  `json:"device,omitempty"`
}

type CaptureReq struct {
	Device  string   `json:"device"`
	Rate    float64  `json:"rate"`
	Samples int      `json:"samples"`
	PreTrig float64  `json:"pretrig"` // 0..1
	Trig    []string `json:"trig"`
}

type LogicDevInfo struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail"`
	Demo   bool   `json:"demo,omitempty"`
}

type LogicManager struct {
	hub     *Hub
	dataDir string
	serial  *SerialManager

	mu    sync.Mutex
	state LogicState
	cap   *Capture
	stop  chan struct{}
	runID int
	decs  []DecoderCfg
}

func newLogicManager(h *Hub, dataDir string, s *SerialManager) *LogicManager {
	return &LogicManager{hub: h, dataDir: dataDir, serial: s, state: LogicState{Status: "idle"}}
}

func (m *LogicManager) State() LogicState {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state
}

func (m *LogicManager) setState(f func(*LogicState)) {
	m.mu.Lock()
	f(&m.state)
	st := m.state
	m.mu.Unlock()
	m.hub.publish("logicState", st)
}

func (m *LogicManager) Stop() {
	m.mu.Lock()
	if m.stop != nil {
		close(m.stop)
		m.stop = nil
	}
	m.mu.Unlock()
}

func (m *LogicManager) Devices() []LogicDevInfo {
	var out []LogicDevInfo
	for _, u := range listUSBDevices() {
		prof := fx2ProfileFor(u.VID, u.PID)
		if prof == nil {
			continue
		}
		p := probeFX2(u, m.dataDir)
		out = append(out, LogicDevInfo{ID: u.Path, Name: prof.Name, Status: p.status, Detail: p.detail})
	}
	out = append(out, LogicDevInfo{ID: "demo", Name: "Demo signal generator", Status: "ready",
		Detail: "UART · I²C · SPI · PWM test signals", Demo: true})
	return out
}

func (m *LogicManager) Start(req CaptureReq) error {
	if req.Rate <= 0 {
		return errors.New("choose a sample rate")
	}
	req.Samples = clampInt(req.Samples, 1000, 200_000_000)
	if req.PreTrig < 0 || req.PreTrig > 0.95 {
		req.PreTrig = 0.1
	}
	m.Stop()
	m.mu.Lock()
	m.runID++
	id := m.runID
	stop := make(chan struct{})
	m.stop = stop
	m.mu.Unlock()
	m.setState(func(s *LogicState) {
		*s = LogicState{Status: "arming", Message: "Starting…", CaptureID: s.CaptureID, Device: req.Device}
	})
	go m.run(id, req, stop)
	return nil
}

func (m *LogicManager) run(id int, req CaptureReq, stop chan struct{}) {
	cond, hasTrig := parseTrig(req.Trig)
	pre := 0
	if hasTrig {
		pre = int(float64(req.Samples) * req.PreTrig)
	}
	col := &collector{n: req.Samples, pre: pre, cond: cond, hasTrig: hasTrig, buf: make([]byte, 0, minInt(req.Samples, 64<<20))}
	var firstT float64
	var firstLen int

	progress := func() {
		m.setState(func(s *LogicState) {
			if hasTrig && !col.triggered {
				s.Status = "waiting"
				s.Message = fmt.Sprintf("Waiting for trigger… (%s scanned)", humanSamples(col.total))
				s.Progress = 0
			} else {
				s.Status = "capturing"
				s.Message = "Capturing"
				s.Progress = float64(len(col.buf)) / float64(col.n)
			}
		})
	}

	var err error
	devName := "Demo"
	if req.Device == "demo" {
		total := req.Samples*2 + pre
		data := generateDemo(req.Rate, total)
		firstT = nowMs() - float64(total)/req.Rate*1000
		firstLen = 0
		chunk := 1 << 20
		for off := 0; off < len(data); off += chunk {
			end := minInt(off+chunk, len(data))
			if col.feed(data[off:end]) {
				break
			}
		}
	} else {
		devName = "Logic analyser"
		err = m.runFX2(req, col, stop, &firstT, &firstLen, progress)
	}

	if hasTrig && !col.triggered {
		col.buf = nil
	}
	m.mu.Lock()
	stale := m.runID != id
	m.mu.Unlock()
	if stale {
		return
	}
	if err != nil {
		log.Printf("capture: %v", err)
		m.setState(func(s *LogicState) { s.Status = "error"; s.Error = err.Error(); s.Message = "" })
		return
	}
	if len(col.buf) < 2 {
		m.setState(func(s *LogicState) {
			s.Status = "idle"
			s.Message = "Stopped before any data arrived"
			if hasTrig && !col.triggered {
				s.Message = "Stopped — trigger never fired"
			}
		})
		return
	}
	m.setState(func(s *LogicState) { s.Status = "processing"; s.Message = "Indexing edges…"; s.Progress = 1 })

	c := &Capture{Device: devName, Rate: req.Rate, Data: col.buf, Trig: -1}
	if col.triggered {
		c.Trig = col.trigIdx
	}
	wall0 := firstT - float64(firstLen)/req.Rate*1000
	c.T0 = wall0 + float64(col.absStart)/req.Rate*1000
	c.index()

	m.mu.Lock()
	if m.runID != id {
		m.mu.Unlock()
		return
	}
	prevID := 0
	if m.cap != nil {
		prevID = m.cap.ID
	}
	c.ID = prevID + 1
	m.cap = c
	decs := m.decs
	m.stop = nil
	m.mu.Unlock()
	runDecoders(c, decs)
	m.setState(func(s *LogicState) {
		s.Status = "done"
		s.Message = fmt.Sprintf("%s samples @ %s", humanSamples(int64(c.N())), humanRate(c.Rate))
		s.CaptureID = c.ID
		s.Progress = 1
	})
}

func (m *LogicManager) runFX2(req CaptureReq, col *collector, stop chan struct{}, firstT *float64, firstLen *int, progress func()) error {
	status := func(msg string) { m.setState(func(s *LogicState) { s.Message = msg }) }
	d, err := fx2Open(req.Device, m.dataDir, status)
	if err != nil {
		return err
	}
	defer d.Close()
	d.ResetPipe(fx2DataEP)

	// ~10 ms per transfer, ~250 ms of buffering in flight.
	size := int(req.Rate/100) &^ 511
	if size < 512 {
		size = 512
	}
	if size > 256*1024 {
		size = 256 * 1024
	}
	count := clampInt(int(math.Ceil(0.25*req.Rate/float64(size))), 4, 32)

	if err := fx2StartAcq(d, req.Rate); err != nil {
		return fmt.Errorf("start acquisition: %w", err)
	}
	lastProg := time.Now()
	got := false
	done := false
	err = d.Stream(fx2DataEP, size, count, stop, func(b []byte) bool {
		if len(b) == 0 {
			return true
		}
		if !got {
			got = true
			*firstT = nowMs()
			*firstLen = len(b)
		}
		if col.feed(b) {
			done = true
			return false
		}
		if time.Since(lastProg) > 120*time.Millisecond {
			lastProg = time.Now()
			progress()
		}
		return true
	})
	if err != nil && !done {
		return err
	}
	return nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func humanSamples(n int64) string {
	switch {
	case n >= 1e6:
		return strconv.FormatFloat(float64(n)/1e6, 'f', -1, 64) + " M"
	case n >= 1e3:
		return strconv.FormatFloat(float64(n)/1e3, 'f', -1, 64) + " k"
	}
	return strconv.FormatInt(n, 10)
}

func humanRate(r float64) string {
	switch {
	case r >= 1e6:
		return strconv.FormatFloat(r/1e6, 'f', -1, 64) + " MHz"
	case r >= 1e3:
		return strconv.FormatFloat(r/1e3, 'f', -1, 64) + " kHz"
	}
	return strconv.FormatFloat(r, 'f', -1, 64) + " Hz"
}

// ---------------------------------------------------------------- HTTP

func (m *LogicManager) current() *Capture {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cap
}

func qf(r *http.Request, k string) float64 {
	v, _ := strconv.ParseFloat(r.URL.Query().Get(k), 64)
	return v
}

func (m *LogicManager) routes(mux *http.ServeMux) {
	mux.HandleFunc("/api/logic/devices", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"devices": m.Devices(), "rates": fx2SampleRates})
	})
	mux.HandleFunc("/api/logic/start", func(w http.ResponseWriter, r *http.Request) {
		var req CaptureReq
		if err := readJSON(r, &req); err != nil {
			writeErr(w, 400, err.Error())
			return
		}
		if err := m.Start(req); err != nil {
			writeErr(w, 400, err.Error())
			return
		}
		writeJSON(w, m.State())
	})
	mux.HandleFunc("/api/logic/stop", func(w http.ResponseWriter, r *http.Request) {
		m.Stop()
		writeJSON(w, m.State())
	})
	mux.HandleFunc("/api/logic/info", func(w http.ResponseWriter, r *http.Request) {
		c := m.current()
		if c == nil {
			writeJSON(w, map[string]any{"has": false})
			return
		}
		writeJSON(w, map[string]any{"has": true, "id": c.ID, "rate": c.Rate, "n": c.N(), "t0": c.T0,
			"trig": c.Trig, "device": c.Device, "edges": len(c.Changes), "rows": c.rowInfo()})
	})
	mux.HandleFunc("/api/logic/view", func(w http.ResponseWriter, r *http.Request) {
		c := m.current()
		if c == nil {
			writeErr(w, 404, "no capture")
			return
		}
		px := clampInt(int(qf(r, "w")), 1, 8192)
		s, e := qf(r, "s"), qf(r, "e")
		if !(e > s) {
			writeErr(w, 400, "bad range")
			return
		}
		writeJSON(w, map[string]any{"id": c.ID, "w": px, "d": base64.StdEncoding.EncodeToString(c.view(s, e, px)),
			"rows": c.annos(s, e, px)})
	})
	mux.HandleFunc("/api/logic/decode", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Decoders []DecoderCfg `json:"decoders"`
		}
		if err := readJSON(r, &req); err != nil {
			writeErr(w, 400, err.Error())
			return
		}
		m.mu.Lock()
		m.decs = req.Decoders
		c := m.cap
		m.mu.Unlock()
		if c == nil {
			writeJSON(w, map[string]any{"rows": []any{}})
			return
		}
		runDecoders(c, req.Decoders)
		writeJSON(w, map[string]any{"rows": c.rowInfo()})
	})
	mux.HandleFunc("/api/logic/table", func(w http.ResponseWriter, r *http.Request) {
		c := m.current()
		if c == nil {
			writeJSON(w, []any{})
			return
		}
		writeJSON(w, c.table(20000))
	})
	mux.HandleFunc("/api/logic/sample", func(w http.ResponseWriter, r *http.Request) {
		c := m.current()
		if c == nil {
			writeErr(w, 404, "no capture")
			return
		}
		i := clampInt(int(qf(r, "i")), 0, c.N()-1)
		writeJSON(w, map[string]any{"i": i, "v": c.Data[i]})
	})
	mux.HandleFunc("/api/logic/measure", func(w http.ResponseWriter, r *http.Request) {
		c := m.current()
		if c == nil {
			writeErr(w, 404, "no capture")
			return
		}
		writeJSON(w, c.measure(int(qf(r, "ch")), qf(r, "s"), qf(r, "e")))
	})
	// Explain: save the capture as a VCD; the window then opens it on the Explain tab
	mux.HandleFunc("/api/logic/explain", func(w http.ResponseWriter, r *http.Request) {
		c := m.current()
		if c == nil {
			writeErr(w, 404, "Capture something first")
			return
		}
		p := filepath.Join(logsDir(), "probie_"+time.Now().Format("20060102_150405")+".vcd")
		f, err := os.Create(p)
		if err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		c.writeVCD(f, strings.Split(r.URL.Query().Get("names"), ","))
		f.Close()
		writeJSON(w, map[string]any{"path": p})
	})
	mux.HandleFunc("/api/logic/export.vcd", func(w http.ResponseWriter, r *http.Request) {
		c := m.current()
		if c == nil {
			http.Error(w, "no capture", 404)
			return
		}
		names := strings.Split(r.URL.Query().Get("names"), ",")
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"probie_%s.vcd\"", time.Now().Format("20060102_150405")))
		c.writeVCD(w, names)
	})
	mux.HandleFunc("/api/logic/firmware", func(w http.ResponseWriter, r *http.Request) {
		name := filepath.Base(r.URL.Query().Get("name"))
		low := strings.ToLower(name)
		if !(strings.HasSuffix(low, ".fw") || strings.HasSuffix(low, ".ihx") || strings.HasSuffix(low, ".hex")) {
			writeErr(w, 400, "expected a .fw (or .ihx) firmware file")
			return
		}
		b, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil || len(b) == 0 {
			writeErr(w, 400, "empty file")
			return
		}
		if _, err := firmwareImage(b); err != nil {
			writeErr(w, 400, err.Error())
			return
		}
		dir := filepath.Join(m.dataDir, "firmware")
		_ = os.MkdirAll(dir, 0o755)
		if err := os.WriteFile(filepath.Join(dir, name), b, 0o644); err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		writeJSON(w, map[string]any{"ok": true, "path": filepath.Join(dir, name)})
	})
}

// measure reports frequency / duty / pulse widths for one channel in a range.
func (c *Capture) measure(ch int, s, e float64) map[string]any {
	if ch < 0 || ch > 7 {
		return map[string]any{}
	}
	a, b := int(math.Max(0, s)), int(math.Min(float64(c.N()-1), e))
	edges := c.Edges[ch]
	i := countLE(edges, a)
	var rises []int
	hiTotal, loTotal := 0, 0
	last := a
	lvl := c.bit(ch, a)
	for ; i < len(edges) && int(edges[i]) <= b; i++ {
		p := int(edges[i])
		if lvl == 1 {
			hiTotal += p - last
		} else {
			loTotal += p - last
		}
		lvl ^= 1
		if lvl == 1 {
			rises = append(rises, p)
		}
		last = p
	}
	if lvl == 1 {
		hiTotal += b - last
	} else {
		loTotal += b - last
	}
	out := map[string]any{"edges": i - countLE(edges, a), "duty": 0.0}
	if hiTotal+loTotal > 0 {
		out["duty"] = float64(hiTotal) / float64(hiTotal+loTotal)
	}
	if len(rises) >= 2 {
		period := float64(rises[len(rises)-1]-rises[0]) / float64(len(rises)-1) / c.Rate
		out["freq"] = 1 / period
		out["period"] = period
	}
	return out
}

func (c *Capture) writeVCD(w io.Writer, names []string) {
	bw := bufio.NewWriterSize(w, 1<<20)
	defer bw.Flush()
	fmt.Fprintf(bw, "$date %s $end\n$version Probie %s by %s $end\n$timescale 1 ps $end\n$scope module probie $end\n",
		time.UnixMilli(int64(c.T0)).Format(time.RFC1123), appVersion, appAuthor)
	ids := "!\"#$%&'("
	for ch := 0; ch < 8; ch++ {
		n := fmt.Sprintf("CH%d", ch)
		if ch < len(names) && strings.TrimSpace(names[ch]) != "" {
			n = strings.ReplaceAll(strings.TrimSpace(names[ch]), " ", "_")
		}
		fmt.Fprintf(bw, "$var wire 1 %c %s $end\n", ids[ch], n)
	}
	fmt.Fprint(bw, "$upscope $end\n$enddefinitions $end\n#0\n$dumpvars\n")
	for ch := 0; ch < 8; ch++ {
		fmt.Fprintf(bw, "%d%c\n", c.bit(ch, 0), ids[ch])
	}
	fmt.Fprint(bw, "$end\n")
	ps := 1e12 / c.Rate
	for _, i := range c.Changes {
		x := c.Data[i] ^ c.Data[i-1]
		fmt.Fprintf(bw, "#%d\n", int64(math.Round(float64(i)*ps)))
		for ch := 0; ch < 8; ch++ {
			if x&(1<<ch) != 0 {
				fmt.Fprintf(bw, "%d%c\n", (c.Data[i]>>ch)&1, ids[ch])
			}
		}
	}
	fmt.Fprintf(bw, "#%d\n", int64(math.Round(float64(len(c.Data))*ps)))
}
