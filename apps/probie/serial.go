package probie

import (
	"encoding/csv"
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Port is a byte stream with modem-control lines (a COM port or the demo).
type Port interface {
	Read([]byte) (int, error)
	Write([]byte) (int, error)
	SetDTR(bool) error
	SetRTS(bool) error
	Close() error
}

type PortConfig struct {
	Baud     int     `json:"baud"`
	DataBits int     `json:"dataBits"`
	Parity   string  `json:"parity"` // none, odd, even, mark, space
	StopBits float64 `json:"stopBits"`
	DTR      bool    `json:"dtr"`
	RTS      bool    `json:"rts"`
}

type PortInfo struct {
	Name string `json:"name"`
	Desc string `json:"desc"`
	Chip string `json:"chip"`
	VID  string `json:"vid,omitempty"`
	PID  string `json:"pid,omitempty"`
	Demo bool   `json:"demo,omitempty"`
}

const demoPortName = "DEMO"

// chipFromIDs gives a friendly hint about what's on the other end.
func chipFromIDs(vid, pid string) string {
	vid, pid = strings.ToUpper(vid), strings.ToUpper(pid)
	switch vid {
	case "2341", "2A03", "1B4F", "239A":
		return "Arduino"
	case "2E8A":
		return "Raspberry Pi Pico"
	case "303A":
		return "ESP32 (native USB)"
	case "1A86":
		return "CH340 / CH9102"
	case "10C4":
		return "CP210x"
	case "0403":
		return "FTDI"
	case "067B":
		return "PL2303"
	case "16C0":
		return "Teensy"
	case "0483":
		return "STM32"
	}
	return ""
}

type SerialEvent struct {
	Seq  uint64  `json:"q"`
	T    float64 `json:"t"` // unix milliseconds
	Dir  string  `json:"d"` // rx, tx, sys
	Data []byte  `json:"b"`
}

type SerialState struct {
	Open      bool       `json:"open"`
	Port      string     `json:"port"`
	Config    PortConfig `json:"config"`
	Status    string     `json:"status"` // closed, open, reconnecting, error
	Error     string     `json:"error,omitempty"`
	Logging   bool       `json:"logging"`
	LogPath   string     `json:"logPath,omitempty"`
	RxBytes   uint64     `json:"rx"`
	TxBytes   uint64     `json:"tx"`
	Reconnect bool       `json:"reconnect"`
}

type SerialManager struct {
	hub *Hub

	mu      sync.Mutex
	port    Port
	state   SerialState
	gen     int // bumps on every open/close so stale readers exit
	events  []SerialEvent
	evBytes int
	seq     uint64
	pending []SerialEvent

	logF     *os.File
	logW     *csv.Writer
	logStart time.Time
	lineBuf  map[string][]byte
	lineT    map[string]float64
}

func newSerialManager(h *Hub) *SerialManager {
	m := &SerialManager{hub: h, lineBuf: map[string][]byte{}, lineT: map[string]float64{}}
	m.state.Status = "closed"
	m.state.Reconnect = true
	go m.flusher()
	return m
}

func nowMs() float64 { return float64(time.Now().UnixNano()) / 1e6 }

func (m *SerialManager) State() SerialState {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state
}

func (m *SerialManager) publishState() {
	m.hub.publish("serialState", m.State())
}

// addEvent records data and queues it for the UI. Caller must NOT hold mu.
func (m *SerialManager) addEvent(dir string, data []byte, t float64) {
	m.mu.Lock()
	m.seq++
	ev := SerialEvent{Seq: m.seq, T: t, Dir: dir, Data: data}
	m.events = append(m.events, ev)
	m.evBytes += len(data) + 32
	// Keep roughly the last 24 MB / 400k events of history.
	if m.evBytes > 24<<20 || len(m.events) > 400000 {
		drop := len(m.events) / 4
		for _, e := range m.events[:drop] {
			m.evBytes -= len(e.Data) + 32
		}
		m.events = append([]SerialEvent(nil), m.events[drop:]...)
	}
	m.pending = append(m.pending, ev)
	switch dir {
	case "rx":
		m.state.RxBytes += uint64(len(data))
	case "tx":
		m.state.TxBytes += uint64(len(data))
	}
	if m.logW != nil && dir != "sys" {
		m.logLines(dir, data, t)
	}
	m.mu.Unlock()
}

func (m *SerialManager) sys(msg string) { m.addEvent("sys", []byte(msg), nowMs()) }

func (m *SerialManager) flusher() {
	t := time.NewTicker(30 * time.Millisecond)
	for range t.C {
		m.mu.Lock()
		p := m.pending
		m.pending = nil
		m.mu.Unlock()
		if len(p) > 0 {
			m.hub.publish("serial", p)
		}
	}
}

// EventsBetween returns history in a wall-clock window (for the timeline).
func (m *SerialManager) EventsBetween(t0, t1 float64) []SerialEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	i := sort.Search(len(m.events), func(i int) bool { return m.events[i].T >= t0 })
	var out []SerialEvent
	for ; i < len(m.events) && m.events[i].T <= t1; i++ {
		out = append(out, m.events[i])
	}
	return out
}

func (m *SerialManager) Open(name string, cfg PortConfig) error {
	m.Close()
	if cfg.Baud <= 0 {
		cfg.Baud = 115200
	}
	if cfg.DataBits == 0 {
		cfg.DataBits = 8
	}
	if cfg.StopBits == 0 {
		cfg.StopBits = 1
	}
	if cfg.Parity == "" {
		cfg.Parity = "none"
	}
	var p Port
	var err error
	if name == demoPortName {
		p = newDemoPort(cfg.Baud)
	} else {
		p, err = openPort(name, cfg)
	}
	if err != nil {
		m.mu.Lock()
		m.state.Status = "error"
		m.state.Error = err.Error()
		m.mu.Unlock()
		m.publishState()
		return err
	}
	m.mu.Lock()
	m.gen++
	gen := m.gen
	m.port = p
	m.state.Open = true
	m.state.Port = name
	m.state.Config = cfg
	m.state.Status = "open"
	m.state.Error = ""
	m.mu.Unlock()
	m.sys(fmt.Sprintf("Connected to %s @ %d baud", name, cfg.Baud))
	m.publishState()
	go m.reader(p, gen)
	return nil
}

func (m *SerialManager) reader(p Port, gen int) {
	buf := make([]byte, 64*1024)
	for {
		n, err := p.Read(buf)
		t := nowMs()
		if n > 0 {
			d := make([]byte, n)
			copy(d, buf[:n])
			m.addEvent("rx", d, t)
		}
		if err != nil {
			m.mu.Lock()
			stale := m.gen != gen
			m.mu.Unlock()
			if stale {
				return
			}
			log.Printf("serial read: %v", err)
			m.lostPort(gen, err)
			return
		}
	}
}

// lostPort handles an unplugged board; optionally waits for it to come back
// (ESP32-S3 / Pico native USB re-enumerate on reset).
func (m *SerialManager) lostPort(gen int, cause error) {
	m.mu.Lock()
	if m.gen != gen {
		m.mu.Unlock()
		return
	}
	name, cfg, again := m.state.Port, m.state.Config, m.state.Reconnect
	if m.port != nil {
		_ = m.port.Close()
		m.port = nil
	}
	if again {
		m.state.Status = "reconnecting"
	} else {
		m.state.Status = "error"
		m.state.Open = false
	}
	m.state.Error = "Port disconnected"
	m.mu.Unlock()
	m.sys("Port disconnected" + map[bool]string{true: " — waiting for it to come back…", false: ""}[again])
	m.publishState()
	if !again {
		return
	}
	for {
		time.Sleep(600 * time.Millisecond)
		m.mu.Lock()
		if m.gen != gen {
			m.mu.Unlock()
			return
		}
		m.mu.Unlock()
		p, err := openPort(name, cfg)
		if err != nil {
			continue
		}
		m.mu.Lock()
		if m.gen != gen {
			m.mu.Unlock()
			_ = p.Close()
			return
		}
		m.port = p
		m.state.Status = "open"
		m.state.Error = ""
		m.mu.Unlock()
		m.sys("Reconnected to " + name)
		m.publishState()
		go m.reader(p, gen)
		return
	}
}

func (m *SerialManager) Close() {
	m.mu.Lock()
	m.gen++
	p := m.port
	m.port = nil
	wasOpen := m.state.Open
	m.state.Open = false
	m.state.Status = "closed"
	m.state.Error = ""
	m.mu.Unlock()
	if p != nil {
		_ = p.Close()
	}
	if wasOpen {
		m.sys("Disconnected")
		m.publishState()
	}
}

func (m *SerialManager) Send(b []byte) error {
	m.mu.Lock()
	p := m.port
	m.mu.Unlock()
	if p == nil {
		return errors.New("port is not open")
	}
	if _, err := p.Write(b); err != nil {
		return err
	}
	m.addEvent("tx", b, nowMs())
	return nil
}

func (m *SerialManager) SetLines(dtr, rts bool) error {
	m.mu.Lock()
	p := m.port
	m.state.Config.DTR, m.state.Config.RTS = dtr, rts
	m.mu.Unlock()
	if p == nil {
		return nil
	}
	if err := p.SetDTR(dtr); err != nil {
		return err
	}
	return p.SetRTS(rts)
}

// ResetBoard pulses the auto-reset lines: works for ESP32/ESP8266 (RTS→EN)
// and classic Arduinos (DTR→RESET via capacitor).
func (m *SerialManager) ResetBoard() error {
	m.mu.Lock()
	p := m.port
	cfg := m.state.Config
	m.mu.Unlock()
	if p == nil {
		return errors.New("port is not open")
	}
	_ = p.SetDTR(false)
	_ = p.SetRTS(true) // EN low on ESP boards
	time.Sleep(120 * time.Millisecond)
	_ = p.SetRTS(false)
	time.Sleep(60 * time.Millisecond)
	_ = p.SetDTR(cfg.DTR) // asserting DTR resets classic Arduinos
	_ = p.SetRTS(cfg.RTS)
	m.sys("Reset pulse sent")
	return nil
}

// ---------------------------------------------------------------- CSV log

func (m *SerialManager) StartLog() (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.logF != nil {
		return m.state.LogPath, nil
	}
	name := "serial_" + time.Now().Format("2006-01-02_15-04-05") + ".csv"
	path := filepath.Join(logsDir(), name)
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	m.logF = f
	m.logW = csv.NewWriter(f)
	m.logStart = time.Now()
	hdr := []string{"timestamp", "elapsed_s", "direction", "line"}
	for i := 1; i <= 16; i++ {
		hdr = append(hdr, fmt.Sprintf("value%d", i))
	}
	_ = m.logW.Write(hdr)
	m.logW.Flush()
	m.lineBuf = map[string][]byte{}
	m.state.Logging = true
	m.state.LogPath = path
	return path, nil
}

func (m *SerialManager) StopLog() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.logF == nil {
		return
	}
	for dir, b := range m.lineBuf {
		if len(b) > 0 {
			m.writeLogRow(dir, string(b), m.lineT[dir])
		}
	}
	m.logW.Flush()
	_ = m.logF.Close()
	m.logF, m.logW = nil, nil
	m.state.Logging = false
}

// logLines splits data into lines and writes complete ones (mu held).
func (m *SerialManager) logLines(dir string, data []byte, t float64) {
	buf := m.lineBuf[dir]
	for _, c := range data {
		if len(buf) == 0 {
			m.lineT[dir] = t
		}
		if c == '\n' {
			m.writeLogRow(dir, strings.TrimRight(string(buf), "\r"), m.lineT[dir])
			buf = buf[:0]
			continue
		}
		if len(buf) < 16384 {
			buf = append(buf, c)
		}
	}
	m.lineBuf[dir] = buf
	m.logW.Flush()
}

func (m *SerialManager) writeLogRow(dir, line string, t float64) {
	ts := time.UnixMilli(int64(t))
	el := (t - float64(m.logStart.UnixNano())/1e6) / 1000
	row := []string{ts.Format("2006-01-02T15:04:05.000"), strconv.FormatFloat(el, 'f', 3, 64), dir, line}
	for _, v := range parseNumbers(line) {
		row = append(row, strconv.FormatFloat(v.val, 'g', -1, 64))
	}
	_ = m.logW.Write(row)
}

type namedVal struct {
	name string
	val  float64
}

// parseNumbers understands "1.2 3 4", "1,2,3", "temp:21.5 hum=40" styles —
// the same rules the plotter uses.
func parseNumbers(line string) []namedVal {
	f := strings.FieldsFunc(line, func(r rune) bool {
		return r == ',' || r == ';' || r == '\t' || r == ' ' || r == '|'
	})
	var out []namedVal
	for _, tok := range f {
		name := ""
		if i := strings.IndexAny(tok, ":="); i >= 0 {
			name, tok = tok[:i], tok[i+1:]
		}
		tok = strings.TrimRight(tok, "%")
		v, err := strconv.ParseFloat(tok, 64)
		if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
			continue
		}
		out = append(out, namedVal{name, v})
		if len(out) >= 16 {
			break
		}
	}
	return out
}

// ---------------------------------------------------------------- HTTP

func (m *SerialManager) routes(mux *http.ServeMux) {
	mux.HandleFunc("/api/serial/ports", func(w http.ResponseWriter, r *http.Request) {
		ports := listPorts()
		sort.Slice(ports, func(i, j int) bool { return comNum(ports[i].Name) < comNum(ports[j].Name) })
		ports = append(ports, PortInfo{Name: demoPortName, Desc: "Demo sensor board (virtual)", Chip: "Virtual", Demo: true})
		writeJSON(w, ports)
	})
	mux.HandleFunc("/api/serial/open", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Port      string     `json:"port"`
			Config    PortConfig `json:"config"`
			Reconnect *bool      `json:"reconnect"`
		}
		if err := readJSON(r, &req); err != nil {
			writeErr(w, 400, err.Error())
			return
		}
		if req.Reconnect != nil {
			m.mu.Lock()
			m.state.Reconnect = *req.Reconnect
			m.mu.Unlock()
		}
		if err := m.Open(req.Port, req.Config); err != nil {
			writeErr(w, 500, friendlyPortError(req.Port, err))
			return
		}
		writeJSON(w, m.State())
	})
	mux.HandleFunc("/api/serial/close", func(w http.ResponseWriter, r *http.Request) {
		m.Close()
		writeJSON(w, m.State())
	})
	mux.HandleFunc("/api/serial/send", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Data []byte `json:"data"`
		}
		if err := readJSON(r, &req); err != nil {
			writeErr(w, 400, err.Error())
			return
		}
		if err := m.Send(req.Data); err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/serial/lines", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			DTR bool `json:"dtr"`
			RTS bool `json:"rts"`
		}
		_ = readJSON(r, &req)
		if err := m.SetLines(req.DTR, req.RTS); err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		m.publishState()
		writeJSON(w, m.State())
	})
	mux.HandleFunc("/api/serial/reset", func(w http.ResponseWriter, r *http.Request) {
		if err := m.ResetBoard(); err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/serial/reconnect", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			On bool `json:"on"`
		}
		_ = readJSON(r, &req)
		m.mu.Lock()
		m.state.Reconnect = req.On
		m.mu.Unlock()
		m.publishState()
		writeJSON(w, m.State())
	})
	// Explain: the window opens the serial log (Log to CSV) on the Explain tab
	mux.HandleFunc("/api/serial/explain", func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		p := m.state.LogPath
		if m.logW != nil {
			m.logW.Flush()
		}
		m.mu.Unlock()
		if p == "" {
			writeErr(w, 400, "Switch on Log to CSV first: Probie explains the log file")
			return
		}
		writeJSON(w, map[string]any{"path": p})
	})
	mux.HandleFunc("/api/serial/log", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			On bool `json:"on"`
		}
		_ = readJSON(r, &req)
		if req.On {
			if _, err := m.StartLog(); err != nil {
				writeErr(w, 500, err.Error())
				return
			}
		} else {
			m.StopLog()
		}
		m.publishState()
		writeJSON(w, m.State())
	})
	mux.HandleFunc("/api/serial/history", func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		ev := m.events
		if len(ev) > 60000 {
			ev = ev[len(ev)-60000:]
		}
		out := append([]SerialEvent(nil), ev...)
		m.mu.Unlock()
		writeJSON(w, out)
	})
	mux.HandleFunc("/api/serial/clear", func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		m.events = nil
		m.evBytes = 0
		m.state.RxBytes, m.state.TxBytes = 0, 0
		m.mu.Unlock()
		m.publishState()
		writeJSON(w, map[string]any{"ok": true})
	})
}

func comNum(name string) int {
	n, err := strconv.Atoi(strings.TrimPrefix(strings.ToUpper(name), "COM"))
	if err != nil {
		return 1 << 20
	}
	return n
}

func friendlyPortError(port string, err error) string {
	s := err.Error()
	low := strings.ToLower(s)
	switch {
	case strings.Contains(low, "access is denied"):
		return port + " is busy — close the Arduino IDE serial monitor, PuTTY or any other app using it."
	case strings.Contains(low, "cannot find the file"):
		return port + " isn't there any more — was the board unplugged?"
	}
	return s
}

// ---------------------------------------------------------------- demo board

// demoPort pretends to be a sensor board printing CSV-ish telemetry.
type demoPort struct {
	mu     sync.Mutex
	out    chan []byte
	quit   chan struct{}
	once   sync.Once
	led    bool
	period time.Duration
	start  time.Time
}

func newDemoPort(baud int) *demoPort {
	d := &demoPort{out: make(chan []byte, 256), quit: make(chan struct{}), period: 50 * time.Millisecond, start: time.Now()}
	go d.run()
	return d
}

func (d *demoPort) emit(s string) {
	select {
	case d.out <- []byte(s):
	case <-d.quit:
	}
}

func (d *demoPort) run() {
	d.emit("\r\n=== Probie demo board ===\r\nType 'help' for commands.\r\n")
	t := time.NewTicker(d.period)
	defer t.Stop()
	n := 0
	for {
		select {
		case <-d.quit:
			return
		case <-t.C:
		}
		n++
		s := time.Since(d.start).Seconds()
		temp := 22 + 3*math.Sin(s/4) + rand.NormFloat64()*0.08
		hum := 48 + 8*math.Sin(s/7+1) + rand.NormFloat64()*0.3
		light := 512 + 380*math.Sin(s*1.3) + rand.NormFloat64()*12
		d.mu.Lock()
		led := d.led
		d.mu.Unlock()
		ledv := 0
		if led {
			ledv = 1
		}
		d.emit(fmt.Sprintf("temp:%.2f,hum:%.1f,light:%.0f,led:%d\r\n", temp, hum, light, ledv))
		if n%60 == 0 {
			d.emit(fmt.Sprintf("[info] uptime %.0fs, free heap %d bytes\r\n", s, 182000+rand.Intn(4000)))
		}
	}
}

func (d *demoPort) Read(b []byte) (int, error) {
	select {
	case p := <-d.out:
		return copy(b, p), nil
	case <-d.quit:
		return 0, errors.New("closed")
	}
}

func (d *demoPort) Write(b []byte) (int, error) {
	cmd := strings.ToLower(strings.TrimSpace(string(b)))
	go func() {
		time.Sleep(15 * time.Millisecond)
		switch {
		case cmd == "":
		case cmd == "help":
			d.emit("commands: help, ping, led on, led off, status, reset\r\n")
		case cmd == "ping":
			d.emit("pong\r\n")
		case cmd == "led on":
			d.mu.Lock()
			d.led = true
			d.mu.Unlock()
			d.emit("ok: LED on\r\n")
		case cmd == "led off":
			d.mu.Lock()
			d.led = false
			d.mu.Unlock()
			d.emit("ok: LED off\r\n")
		case cmd == "status":
			d.emit(fmt.Sprintf("status: uptime=%.1fs led=%v\r\n", time.Since(d.start).Seconds(), d.led))
		case cmd == "reset":
			d.start = time.Now()
			d.emit("\r\nets Jun  8 2016 00:22:57\r\nrst:0x1 (POWERON_RESET)\r\n=== Probie demo board ===\r\n")
		default:
			d.emit("? unknown command: " + cmd + "\r\n")
		}
	}()
	return len(b), nil
}

func (d *demoPort) SetDTR(bool) error { return nil }
func (d *demoPort) SetRTS(v bool) error {
	return nil
}
func (d *demoPort) Close() error {
	d.once.Do(func() { close(d.quit) })
	return nil
}
