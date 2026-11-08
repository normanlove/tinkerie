// Tunie — conn.go: optional direct printer connections.
//   - USB serial (Marlin / Prusa / RepRapFirmware over USB)
//   - Moonraker API (Klipper)
//   - OctoPrint API
//
// Part of the nrmluvdev maker suite. Created by nrmluvdev.
package tunie

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ConnConfig struct {
	Type   string `json:"type"` // serial | moonraker | octoprint
	Port   string `json:"port"`
	Baud   int    `json:"baud"`
	URL    string `json:"url"`
	APIKey string `json:"apiKey"`
}

type Conn interface {
	Kind() string
	Send(lines []string) ([]string, error)
	Upload(name string, data []byte, start bool) error
	Status() map[string]any
	Query(what string) (json.RawMessage, error)
	Close() error
}

// ---------------------------------------------------------------- console log

type LogLine struct {
	Seq  int64  `json:"seq"`
	T    int64  `json:"t"`
	Dir  string `json:"dir"` // tx | rx | info | err
	Text string `json:"text"`
}

type LogBuf struct {
	mu    sync.Mutex
	lines []LogLine
	seq   int64
}

func (l *LogBuf) Add(dir, text string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.seq++
	l.lines = append(l.lines, LogLine{l.seq, time.Now().UnixMilli(), dir, text})
	if len(l.lines) > 2000 {
		l.lines = l.lines[len(l.lines)-1500:]
	}
}

func (l *LogBuf) Since(seq int64) []LogLine {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := []LogLine{}
	for _, x := range l.lines {
		if x.Seq > seq {
			out = append(out, x)
		}
	}
	return out
}

// ---------------------------------------------------------------- manager

type Job struct {
	Name    string `json:"name"`
	Total   int    `json:"total"`
	Done    int    `json:"done"`
	State   string `json:"state"` // running | done | cancelled | error
	Err     string `json:"err,omitempty"`
	Started int64  `json:"started"`
	cancel  bool
}

type ConnManager struct {
	mu  sync.Mutex
	c   Conn
	cfg ConnConfig
	log *LogBuf
	job *Job
}

func NewConnManager() *ConnManager { return &ConnManager{log: &LogBuf{}} }

func (m *ConnManager) Busy() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.job != nil && m.job.State == "running"
}

func (m *ConnManager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.c != nil {
		_ = m.c.Close()
		m.c = nil
	}
}

func (m *ConnManager) current() (Conn, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.c == nil {
		return nil, errors.New("not connected to a printer")
	}
	return m.c, nil
}

func (m *ConnManager) Connect(cfg ConnConfig) (map[string]any, error) {
	m.Close()
	var c Conn
	var err error
	switch cfg.Type {
	case "serial":
		c, err = openSerialConn(cfg, m.log)
	case "moonraker":
		c, err = openMoonraker(cfg, m.log)
	case "octoprint":
		c, err = openOctoPrint(cfg, m.log)
	default:
		return nil, fmt.Errorf("choose a connection type")
	}
	if err != nil {
		m.log.Add("err", err.Error())
		return nil, err
	}
	m.mu.Lock()
	m.c, m.cfg = c, cfg
	m.mu.Unlock()
	m.log.Add("info", "Connected via "+c.Kind())
	return c.Status(), nil
}

func (m *ConnManager) routes(mux *http.ServeMux) {
	mux.HandleFunc("/api/conn/ports", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, listSerialPorts())
	})
	mux.HandleFunc("/api/conn/connect", func(w http.ResponseWriter, r *http.Request) {
		var cfg ConnConfig
		if err := readJSON(r, &cfg); err != nil {
			writeErr(w, 400, err)
			return
		}
		st, err := m.Connect(cfg)
		if err != nil {
			writeErr(w, 502, err)
			return
		}
		writeJSON(w, st)
	})
	mux.HandleFunc("/api/conn/disconnect", func(w http.ResponseWriter, r *http.Request) {
		m.Close()
		m.log.Add("info", "Disconnected")
		writeJSON(w, map[string]bool{"ok": true})
	})
	mux.HandleFunc("/api/conn/status", func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		c, job := m.c, m.job
		var jc *Job
		if job != nil {
			j := *job
			jc = &j
		}
		m.mu.Unlock()
		out := map[string]any{"connected": c != nil, "job": jc}
		if c != nil {
			out["kind"] = c.Kind()
			out["status"] = c.Status()
		}
		writeJSON(w, out)
	})
	mux.HandleFunc("/api/conn/log", func(w http.ResponseWriter, r *http.Request) {
		since, _ := strconv.ParseInt(r.URL.Query().Get("since"), 10, 64)
		writeJSON(w, m.log.Since(since))
	})
	mux.HandleFunc("/api/conn/send", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Lines []string `json:"lines"`
		}
		if err := readJSON(r, &req); err != nil {
			writeErr(w, 400, err)
			return
		}
		if m.Busy() {
			writeErr(w, 409, errors.New("a print is being streamed — wait for it to finish or cancel it"))
			return
		}
		c, err := m.current()
		if err != nil {
			writeErr(w, 409, err)
			return
		}
		resp, err := c.Send(cleanLines(req.Lines))
		out := map[string]any{"lines": resp}
		if err != nil {
			out["error"] = err.Error()
		}
		writeJSON(w, out)
	})
	mux.HandleFunc("/api/conn/query", func(w http.ResponseWriter, r *http.Request) {
		c, err := m.current()
		if err != nil {
			writeErr(w, 409, err)
			return
		}
		res, err := c.Query(r.URL.Query().Get("what"))
		if err != nil {
			writeErr(w, 502, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(res)
	})
	mux.HandleFunc("/api/conn/print", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name  string `json:"name"`
			Gcode string `json:"gcode"`
			Start bool   `json:"start"`
		}
		if err := readJSON(r, &req); err != nil {
			writeErr(w, 400, err)
			return
		}
		c, err := m.current()
		if err != nil {
			writeErr(w, 409, err)
			return
		}
		if sc, ok := c.(*SerialConn); ok {
			if m.Busy() {
				writeErr(w, 409, errors.New("already printing"))
				return
			}
			lines := cleanLines(strings.Split(req.Gcode, "\n"))
			job := &Job{Name: req.Name, Total: len(lines), State: "running", Started: time.Now().Unix()}
			m.mu.Lock()
			m.job = job
			m.mu.Unlock()
			go m.stream(sc, job, lines)
			writeJSON(w, map[string]any{"streaming": true, "lines": len(lines)})
			return
		}
		if err := c.Upload(req.Name, []byte(req.Gcode), req.Start); err != nil {
			writeErr(w, 502, err)
			return
		}
		writeJSON(w, map[string]any{"uploaded": true})
	})
	mux.HandleFunc("/api/conn/cancel", func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		if m.job != nil {
			m.job.cancel = true
		}
		m.mu.Unlock()
		writeJSON(w, map[string]bool{"ok": true})
	})
}

func (m *ConnManager) stream(sc *SerialConn, job *Job, lines []string) {
	m.log.Add("info", fmt.Sprintf("Streaming %s (%d lines)", job.Name, len(lines)))
	for i, l := range lines {
		m.mu.Lock()
		cancel := job.cancel
		m.mu.Unlock()
		if cancel {
			m.mu.Lock()
			job.State = "cancelled"
			m.mu.Unlock()
			_, _ = sc.Send([]string{"M104 S0", "M140 S0", "M106 S0", "M84"})
			m.log.Add("info", "Print cancelled — heaters off")
			return
		}
		if _, err := sc.sendOne(l, timeoutFor(l)); err != nil {
			m.mu.Lock()
			job.State, job.Err = "error", err.Error()
			m.mu.Unlock()
			_, _ = sc.Send([]string{"M104 S0", "M140 S0"})
			m.log.Add("err", "Print stopped: "+err.Error())
			return
		}
		m.mu.Lock()
		job.Done = i + 1
		m.mu.Unlock()
	}
	m.mu.Lock()
	job.State = "done"
	m.mu.Unlock()
	m.log.Add("info", "Print finished")
}

// cleanLines strips comments and blank lines.
func cleanLines(in []string) []string {
	out := make([]string, 0, len(in))
	for _, l := range in {
		if i := strings.IndexByte(l, ';'); i >= 0 {
			l = l[:i]
		}
		l = strings.TrimSpace(l)
		if l != "" {
			out = append(out, l)
		}
	}
	return out
}

func timeoutFor(line string) time.Duration {
	u := strings.ToUpper(line)
	switch {
	case strings.HasPrefix(u, "M303"), strings.HasPrefix(u, "PID_CALIBRATE"), strings.HasPrefix(u, "M109"), strings.HasPrefix(u, "M190"), strings.HasPrefix(u, "M116"):
		return 20 * time.Minute
	case strings.HasPrefix(u, "G29"), strings.HasPrefix(u, "G80"), strings.HasPrefix(u, "BED_MESH"):
		return 15 * time.Minute
	case strings.HasPrefix(u, "G28"):
		return 4 * time.Minute
	case strings.HasPrefix(u, "G0"), strings.HasPrefix(u, "G1"), strings.HasPrefix(u, "M400"):
		return 5 * time.Minute
	}
	return 45 * time.Second
}

// ---------------------------------------------------------------- serial

type serialPort interface {
	Read(b []byte) (int, error)
	Write(b []byte) (int, error)
	Close() error
}

type SerialConn struct {
	port    serialPort
	name    string
	lines   chan string
	log     *LogBuf
	sendMu  sync.Mutex
	closed  chan struct{}
	once    sync.Once
	mu      sync.Mutex
	temps   map[string]float64
	fwInfo  string
	partial string
}

var reTemp = regexp.MustCompile(`([TB]\d?):\s*(-?[\d.]+)\s*/\s*(-?[\d.]+)`)

func openSerialConn(cfg ConnConfig, lg *LogBuf) (*SerialConn, error) {
	if cfg.Port == "" {
		return nil, errors.New("choose a COM port")
	}
	if cfg.Baud == 0 {
		cfg.Baud = 115200
	}
	p, err := openSerial(cfg.Port, cfg.Baud)
	if err != nil {
		return nil, err
	}
	s := &SerialConn{port: p, name: cfg.Port, lines: make(chan string, 4096), log: lg, closed: make(chan struct{}), temps: map[string]float64{}}
	go s.reader()
	// Most boards reset when the port opens; wait for them to boot.
	time.Sleep(2500 * time.Millisecond)
	s.drain()
	resp, err := s.sendOne("M115", 8*time.Second)
	if err != nil {
		// some firmwares answer slowly on first contact — try once more
		resp, err = s.sendOne("M115", 8*time.Second)
	}
	if err != nil {
		s.Close()
		return nil, fmt.Errorf("the printer on %s didn't answer (check the baud rate — try 115200 or 250000): %v", cfg.Port, err)
	}
	s.fwInfo = strings.Join(resp, " ")
	return s, nil
}

func (s *SerialConn) Kind() string { return "serial " + s.name }

func (s *SerialConn) reader() {
	buf := make([]byte, 1024)
	for {
		select {
		case <-s.closed:
			return
		default:
		}
		n, err := s.port.Read(buf)
		if err != nil {
			select {
			case <-s.closed:
				return
			default:
			}
			s.log.Add("err", "serial read: "+err.Error())
			time.Sleep(300 * time.Millisecond)
			continue
		}
		if n == 0 {
			continue
		}
		s.partial += string(buf[:n])
		for {
			i := strings.IndexByte(s.partial, '\n')
			if i < 0 {
				break
			}
			line := strings.TrimSpace(strings.TrimRight(s.partial[:i], "\r"))
			s.partial = s.partial[i+1:]
			if line == "" {
				continue
			}
			s.parseTemps(line)
			if !isTempOnly(line) {
				s.log.Add("rx", line)
			}
			select {
			case s.lines <- line:
			default:
				<-s.lines
				s.lines <- line
			}
		}
	}
}

func isTempOnly(l string) bool {
	t := strings.TrimSpace(l)
	return (strings.HasPrefix(t, "T:") || strings.HasPrefix(t, "T0:")) && strings.Contains(t, "/")
}

func (s *SerialConn) parseTemps(l string) {
	ms := reTemp.FindAllStringSubmatch(l, -1)
	if len(ms) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, m := range ms {
		v, _ := strconv.ParseFloat(m[2], 64)
		t, _ := strconv.ParseFloat(m[3], 64)
		key := "hotend"
		if strings.HasPrefix(m[1], "B") {
			key = "bed"
		}
		s.temps[key] = v
		s.temps[key+"Target"] = t
	}
}

func (s *SerialConn) drain() {
	for {
		select {
		case <-s.lines:
		default:
			return
		}
	}
}

func (s *SerialConn) sendOne(line string, timeout time.Duration) ([]string, error) {
	s.drain()
	if _, err := s.port.Write([]byte(line + "\n")); err != nil {
		return nil, err
	}
	s.log.Add("tx", line)
	deadline := time.Now().Add(timeout)
	var resp []string
	for {
		wait := time.Until(deadline)
		if wait <= 0 {
			return resp, fmt.Errorf("no reply to %q", line)
		}
		select {
		case l := <-s.lines:
			ll := strings.ToLower(l)
			switch {
			case strings.HasPrefix(ll, "ok"):
				if rest := strings.TrimSpace(l[2:]); rest != "" && !strings.HasPrefix(rest, "T:") {
					resp = append(resp, rest)
				}
				return resp, nil
			case strings.HasPrefix(ll, "echo:busy"), ll == "wait":
				deadline = time.Now().Add(timeout)
			case isTempOnly(l):
				// keep waiting while heating
			default:
				resp = append(resp, l)
			}
		case <-time.After(wait):
			return resp, fmt.Errorf("no reply to %q", line)
		case <-s.closed:
			return resp, errors.New("connection closed")
		}
	}
}

func (s *SerialConn) Send(lines []string) ([]string, error) {
	s.sendMu.Lock()
	defer s.sendMu.Unlock()
	var all []string
	for _, l := range lines {
		r, err := s.sendOne(l, timeoutFor(l))
		all = append(all, r...)
		if err != nil {
			return all, err
		}
	}
	return all, nil
}

func (s *SerialConn) Upload(name string, data []byte, start bool) error {
	return errors.New("serial connections stream prints line by line")
}

func (s *SerialConn) Status() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := map[string]any{}
	for k, v := range s.temps {
		t[k] = v
	}
	return map[string]any{"port": s.name, "firmware": s.fwInfo, "temps": t}
}

func (s *SerialConn) Query(what string) (json.RawMessage, error) {
	return nil, errors.New("for serial printers Tunie reads values by sending commands (e.g. M503, G29 T)")
}

func (s *SerialConn) Close() error {
	s.once.Do(func() {
		close(s.closed)
		_ = s.port.Close()
	})
	return nil
}

// ---------------------------------------------------------------- HTTP helpers

func normURL(u string) (string, error) {
	u = strings.TrimSpace(u)
	if u == "" {
		return "", errors.New("enter the printer's address, e.g. http://192.168.1.50")
	}
	if !strings.Contains(u, "://") {
		u = "http://" + u
	}
	p, err := url.Parse(u)
	if err != nil || p.Host == "" {
		return "", errors.New("that address doesn't look right")
	}
	return strings.TrimRight(p.Scheme+"://"+p.Host+p.Path, "/"), nil
}

type webConn struct {
	base   string
	key    string
	keyHdr string
	hc     *http.Client
	log    *LogBuf
}

func (c *webConn) do(method, path string, body io.Reader, ctype string, timeout time.Duration) ([]byte, error) {
	req, err := http.NewRequest(method, c.base+path, body)
	if err != nil {
		return nil, err
	}
	if ctype != "" {
		req.Header.Set("Content-Type", ctype)
	}
	if c.key != "" {
		req.Header.Set(c.keyHdr, c.key)
	}
	hc := &http.Client{Timeout: timeout}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(b))
		if len(msg) > 300 {
			msg = msg[:300]
		}
		return b, fmt.Errorf("%s %s: HTTP %d %s", method, path, resp.StatusCode, msg)
	}
	return b, nil
}

func multipartBody(fields map[string]string, fileField, name string, data []byte) (*bytes.Buffer, string, error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for k, v := range fields {
		if err := mw.WriteField(k, v); err != nil {
			return nil, "", err
		}
	}
	fw, err := mw.CreateFormFile(fileField, name)
	if err != nil {
		return nil, "", err
	}
	if _, err := fw.Write(data); err != nil {
		return nil, "", err
	}
	if err := mw.Close(); err != nil {
		return nil, "", err
	}
	return &buf, mw.FormDataContentType(), nil
}

// ---------------------------------------------------------------- Moonraker

type MoonConn struct{ webConn }

func openMoonraker(cfg ConnConfig, lg *LogBuf) (*MoonConn, error) {
	base, err := normURL(cfg.URL)
	if err != nil {
		return nil, err
	}
	m := &MoonConn{webConn{base: base, key: cfg.APIKey, keyHdr: "X-Api-Key", log: lg}}
	if _, err := m.do("GET", "/server/info", nil, "", 6*time.Second); err != nil {
		return nil, fmt.Errorf("couldn't reach Moonraker at %s: %v", base, err)
	}
	return m, nil
}

func (m *MoonConn) Kind() string { return "moonraker " + m.base }

func (m *MoonConn) storeTime() float64 {
	b, err := m.do("GET", "/server/gcode_store?count=1", nil, "", 5*time.Second)
	if err != nil {
		return 0
	}
	var r struct {
		Result struct {
			Store []struct {
				Time float64 `json:"time"`
			} `json:"gcode_store"`
		} `json:"result"`
	}
	_ = json.Unmarshal(b, &r)
	if len(r.Result.Store) == 0 {
		return 0
	}
	return r.Result.Store[len(r.Result.Store)-1].Time
}

func (m *MoonConn) Send(lines []string) ([]string, error) {
	if len(lines) == 0 {
		return nil, nil
	}
	t0 := m.storeTime()
	script := strings.Join(lines, "\n")
	for _, l := range lines {
		m.log.Add("tx", l)
	}
	body, _ := json.Marshal(map[string]string{"script": script})
	_, err := m.do("POST", "/printer/gcode/script", bytes.NewReader(body), "application/json", 25*time.Minute)
	b, e2 := m.do("GET", "/server/gcode_store?count=200", nil, "", 8*time.Second)
	var out []string
	if e2 == nil {
		var r struct {
			Result struct {
				Store []struct {
					Message string  `json:"message"`
					Time    float64 `json:"time"`
					Type    string  `json:"type"`
				} `json:"gcode_store"`
			} `json:"result"`
		}
		_ = json.Unmarshal(b, &r)
		for _, s := range r.Result.Store {
			if s.Time > t0 && s.Type == "response" {
				out = append(out, s.Message)
				m.log.Add("rx", s.Message)
			}
		}
	}
	if err != nil {
		m.log.Add("err", err.Error())
	}
	return out, err
}

func (m *MoonConn) Upload(name string, data []byte, start bool) error {
	fields := map[string]string{"root": "gcodes"}
	if start {
		fields["print"] = "true"
	}
	body, ct, err := multipartBody(fields, "file", name, data)
	if err != nil {
		return err
	}
	_, err = m.do("POST", "/server/files/upload", body, ct, 5*time.Minute)
	if err == nil {
		m.log.Add("info", "Uploaded "+name+map[bool]string{true: " and started printing", false: ""}[start])
	}
	return err
}

func (m *MoonConn) Status() map[string]any {
	b, err := m.do("GET", "/printer/objects/query?extruder=temperature,target&heater_bed=temperature,target&print_stats=state,filename", nil, "", 5*time.Second)
	out := map[string]any{"url": m.base}
	if err != nil {
		out["error"] = err.Error()
		return out
	}
	var r struct {
		Result struct {
			Status map[string]map[string]any `json:"status"`
		} `json:"result"`
	}
	_ = json.Unmarshal(b, &r)
	t := map[string]any{}
	if e := r.Result.Status["extruder"]; e != nil {
		t["hotend"], t["hotendTarget"] = e["temperature"], e["target"]
	}
	if e := r.Result.Status["heater_bed"]; e != nil {
		t["bed"], t["bedTarget"] = e["temperature"], e["target"]
	}
	out["temps"] = t
	if ps := r.Result.Status["print_stats"]; ps != nil {
		out["state"] = ps["state"]
		out["file"] = ps["filename"]
	}
	return out
}

func (m *MoonConn) Query(what string) (json.RawMessage, error) {
	path := ""
	switch what {
	case "mesh":
		path = "/printer/objects/query?bed_mesh"
	case "config":
		path = "/printer/objects/query?configfile=settings"
	default:
		return nil, fmt.Errorf("unknown query %q", what)
	}
	b, err := m.do("GET", path, nil, "", 10*time.Second)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}

func (m *MoonConn) Close() error { return nil }

// ---------------------------------------------------------------- OctoPrint

type OctoConn struct{ webConn }

func openOctoPrint(cfg ConnConfig, lg *LogBuf) (*OctoConn, error) {
	base, err := normURL(cfg.URL)
	if err != nil {
		return nil, err
	}
	if cfg.APIKey == "" {
		return nil, errors.New("OctoPrint needs an API key (OctoPrint Settings → Application Keys)")
	}
	o := &OctoConn{webConn{base: base, key: cfg.APIKey, keyHdr: "X-Api-Key", log: lg}}
	if _, err := o.do("GET", "/api/version", nil, "", 6*time.Second); err != nil {
		return nil, fmt.Errorf("couldn't reach OctoPrint at %s: %v", base, err)
	}
	return o, nil
}

func (o *OctoConn) Kind() string { return "octoprint " + o.base }

func (o *OctoConn) Send(lines []string) ([]string, error) {
	body, _ := json.Marshal(map[string][]string{"commands": lines})
	for _, l := range lines {
		o.log.Add("tx", l)
	}
	_, err := o.do("POST", "/api/printer/command", bytes.NewReader(body), "application/json", 30*time.Second)
	if err != nil {
		return nil, err
	}
	note := "Sent. OctoPrint doesn't pass replies back to other apps — read them in OctoPrint's Terminal tab and paste them here."
	o.log.Add("info", note)
	return []string{note}, nil
}

func (o *OctoConn) Upload(name string, data []byte, start bool) error {
	fields := map[string]string{"select": "true"}
	if start {
		fields["print"] = "true"
	}
	body, ct, err := multipartBody(fields, "file", name, data)
	if err != nil {
		return err
	}
	_, err = o.do("POST", "/api/files/local", body, ct, 5*time.Minute)
	if err == nil {
		o.log.Add("info", "Uploaded "+name)
	}
	return err
}

func (o *OctoConn) Status() map[string]any {
	out := map[string]any{"url": o.base}
	b, err := o.do("GET", "/api/printer", nil, "", 5*time.Second)
	if err != nil {
		out["error"] = err.Error()
		return out
	}
	var r struct {
		Temperature map[string]struct {
			Actual float64 `json:"actual"`
			Target float64 `json:"target"`
		} `json:"temperature"`
		State struct {
			Text string `json:"text"`
		} `json:"state"`
	}
	_ = json.Unmarshal(b, &r)
	t := map[string]any{}
	if v, ok := r.Temperature["tool0"]; ok {
		t["hotend"], t["hotendTarget"] = v.Actual, v.Target
	}
	if v, ok := r.Temperature["bed"]; ok {
		t["bed"], t["bedTarget"] = v.Actual, v.Target
	}
	out["temps"] = t
	out["state"] = r.State.Text
	return out
}

func (o *OctoConn) Query(what string) (json.RawMessage, error) {
	return nil, errors.New("OctoPrint can't return printer replies — paste them from OctoPrint's Terminal tab")
}

func (o *OctoConn) Close() error { return nil }
