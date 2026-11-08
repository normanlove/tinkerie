// Flashie — MicroPython raw-REPL client (upload files, run, interrupt) and
// the live serial console used by the Firmware Creator. Made by nrmluvdev.

package flashie

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// --------------------------------------------------------------- events ---

func emitConsole(text, src string) {
	events.send(map[string]any{"type": "console", "text": text, "src": src})
}
func emitCStatus(s string)        { events.send(map[string]any{"type": "cstatus", "text": s}) }
func emitCDone(ok bool, m string) { events.send(map[string]any{"type": "cdone", "ok": ok, "text": m}) }

func creatorReporter() *reporter {
	return &reporter{
		Log:    func(s string) { emitConsole(s+"\n", "info") },
		Status: func(s string) { emitCStatus(s); emitConsole("» "+s+"\n", "info") },
		Progress: func() func(int, int) {
			var last time.Time
			return func(d, t int) {
				if time.Since(last) > 80*time.Millisecond || d == t {
					last = time.Now()
					events.send(map[string]any{"type": "cprogress", "done": d, "total": t})
				}
			}
		}(),
		Cancel: cancelReq.Load,
	}
}

// ---------------------------------------------------------------- REPL ---

type mpyRepl struct {
	p Port
	k *link
}

func newRepl(p Port) *mpyRepl { return &mpyRepl{p: p, k: newLink(p)} }

// readUntil reads until marker appears, returning everything before it.
func (m *mpyRepl) readUntil(marker []byte, timeout time.Duration) ([]byte, error) {
	var got []byte
	deadline := time.Now().Add(timeout)
	for !bytes.Contains(got, marker) {
		if time.Now().After(deadline) {
			return got, errTimeout
		}
		b, err := m.k.readByte(time.Until(deadline))
		if err != nil {
			return got, err
		}
		got = append(got, b)
	}
	return got[:len(got)-len(marker)], nil
}

func (m *mpyRepl) send(b []byte) error {
	// Small chunks with pauses so boards with tiny UART buffers keep up.
	for len(b) > 0 {
		n := min(256, len(b))
		if err := m.k.write(b[:n]...); err != nil {
			return err
		}
		b = b[n:]
		if len(b) > 0 {
			time.Sleep(10 * time.Millisecond)
		}
	}
	return nil
}

// enterRaw stops whatever is running and switches to raw REPL mode.
func (m *mpyRepl) enterRaw() error {
	for attempt := 0; attempt < 2; attempt++ {
		m.k.write('\r', 0x03)
		time.Sleep(100 * time.Millisecond)
		m.k.write(0x03)
		time.Sleep(100 * time.Millisecond)
		m.k.drain(150 * time.Millisecond)
		m.k.write('\r', 0x01)
		if _, err := m.readUntil([]byte("raw REPL; CTRL-B to exit\r\n>"), 2*time.Second); err == nil {
			return nil
		}
	}
	return errors.New("no MicroPython prompt — MicroPython may not be installed on this board yet")
}

func (m *mpyRepl) exitRaw() { m.k.write(0x02) }

// exec runs code in raw mode and returns its printed output.
func (m *mpyRepl) exec(code string, timeout time.Duration) (string, error) {
	if err := m.send([]byte(code)); err != nil {
		return "", err
	}
	if err := m.k.write(0x04); err != nil {
		return "", err
	}
	if _, err := m.readUntil([]byte("OK"), 3*time.Second); err != nil {
		return "", errors.New("the board didn't accept the command (is a program still running?)")
	}
	out, err := m.readUntil([]byte{0x04}, timeout)
	if err != nil {
		return string(out), err
	}
	errOut, err := m.readUntil([]byte{0x04}, timeout)
	if err != nil {
		return string(out), err
	}
	m.readUntil([]byte(">"), time.Second)
	if len(bytes.TrimSpace(errOut)) > 0 {
		return string(out), fmt.Errorf("board error: %s", strings.TrimSpace(string(errOut)))
	}
	return string(out), nil
}

type mpyInfo struct {
	MicroPython bool   `json:"micropython"`
	Version     string `json:"version,omitempty"`
	Platform    string `json:"platform,omitempty"`
	Machine     string `json:"machine,omitempty"`
}

func (m *mpyRepl) info() (*mpyInfo, error) {
	out, err := m.exec("import sys, os\nprint(sys.implementation.name, '.'.join(str(v) for v in sys.implementation.version[:3]), sys.platform)\nprint(os.uname().machine)\n", 5*time.Second)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.ReplaceAll(strings.TrimSpace(out), "\r", ""), "\n")
	f := strings.Fields(lines[0])
	inf := &mpyInfo{}
	if len(f) >= 3 && f[0] == "micropython" {
		inf.MicroPython, inf.Version, inf.Platform = true, f[1], f[2]
	}
	if len(lines) > 1 {
		inf.Machine = strings.TrimSpace(lines[1])
	}
	return inf, nil
}

// writeFile copies a file onto the board's flash filesystem.
func (m *mpyRepl) writeFile(name string, data []byte, r *reporter, done, total *int) error {
	if strings.ContainsAny(name, "'\\\n") {
		return errors.New("bad file name")
	}
	if _, err := m.exec(fmt.Sprintf("from binascii import unhexlify as _u\n_f=open('%s','wb')\n_w=_f.write\n", name), 3*time.Second); err != nil {
		return err
	}
	const chunk = 512
	for off := 0; off < len(data); off += chunk {
		end := min(off+chunk, len(data))
		if _, err := m.exec("_w(_u('"+hex.EncodeToString(data[off:end])+"'))\n", 5*time.Second); err != nil {
			m.exec("_f.close()\n", 2*time.Second)
			return fmt.Errorf("writing %s: %w", name, err)
		}
		*done += end - off
		r.progress(*done, *total)
	}
	_, err := m.exec("_f.close()\ndel _f,_w,_u\n", 3*time.Second)
	return err
}

// ------------------------------------------------------------- sessions ---

// serialSession keeps a port open and streams whatever the board prints.
type serialSession struct {
	p    Port
	name string
	kind string // repl | serial
	stop chan struct{}
	done chan struct{}
}

var (
	sessMu sync.Mutex
	sess   *serialSession
)

func startSession(p Port, name, kind string) {
	stopSession()
	s := &serialSession{p: p, name: name, kind: kind, stop: make(chan struct{}), done: make(chan struct{})}
	sessMu.Lock()
	sess = s
	sessMu.Unlock()
	events.send(map[string]any{"type": "csession", "open": true, "port": name, "kind": kind})
	go func() {
		defer close(s.done)
		buf := make([]byte, 2048)
		var pending []byte
		last := time.Now()
		for {
			select {
			case <-s.stop:
				if len(pending) > 0 {
					emitConsole(string(pending), "board")
				}
				return
			default:
			}
			n, err := p.ReadTimeout(buf, 40*time.Millisecond)
			if err != nil {
				emitConsole("\n[connection to "+name+" lost — was the board unplugged?]\n", "info")
				events.send(map[string]any{"type": "csession", "open": false})
				p.Close()
				sessMu.Lock()
				if sess == s {
					sess = nil
				}
				sessMu.Unlock()
				return
			}
			pending = append(pending, buf[:n]...)
			if len(pending) > 0 && (time.Since(last) > 60*time.Millisecond || len(pending) > 4096) {
				emitConsole(string(pending), "board")
				pending = nil
				last = time.Now()
			}
		}
	}()
}

// stopSession closes the console so the port is free for something else.
func stopSession() {
	sessMu.Lock()
	s := sess
	sess = nil
	sessMu.Unlock()
	if s == nil {
		return
	}
	close(s.stop)
	<-s.done
	s.p.Close()
	events.send(map[string]any{"type": "csession", "open": false})
}

func sessionWrite(b []byte) error {
	sessMu.Lock()
	s := sess
	sessMu.Unlock()
	if s == nil {
		return errors.New("the console isn't connected — press Run or Check board first")
	}
	_, err := s.p.Write(b)
	return err
}

func openReplPort(name string) (Port, error) {
	if demoMode {
		if p := demoMPY(name); p != nil {
			return p, nil
		}
		return demoOpen(name)
	}
	return openSerialEx(name, 115200, false)
}

// ------------------------------------------------------------- handlers ---

func decodeJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

func handleCreatorCheck(w http.ResponseWriter, r *http.Request) {
	var in struct{ Port string }
	decodeJSON(r, &in)
	if in.Port == "" {
		jsonErr(w, "Pick your board's port first.")
		return
	}
	if !busy.CompareAndSwap(false, true) {
		jsonErr(w, "Flashie is busy with another job — wait for it to finish.")
		return
	}
	defer busy.Store(false)
	stopSession()
	pi, _ := findDevice(in.Port)
	if pi.Kind == "drive" {
		writeJSON(w, mpyInfo{})
		return
	}
	p, err := openReplPort(in.Port)
	if err != nil {
		jsonErr(w, friendly(err))
		return
	}
	m := newRepl(p)
	if err := m.enterRaw(); err != nil {
		p.Close()
		writeJSON(w, mpyInfo{})
		return
	}
	inf, err := m.info()
	m.exitRaw()
	if err != nil || !inf.MicroPython {
		p.Close()
		writeJSON(w, mpyInfo{})
		return
	}
	startSession(p, in.Port, "repl")
	writeJSON(w, inf)
}

func handleCreatorRun(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Port  string
		Files map[string]string
	}
	if err := decodeJSON(r, &in); err != nil || in.Port == "" || len(in.Files) == 0 {
		jsonErr(w, "Nothing to upload — pick a board and build a program first.")
		return
	}
	if !busy.CompareAndSwap(false, true) {
		jsonErr(w, "Flashie is busy with another job — wait for it to finish.")
		return
	}
	cancelReq.Store(false)
	writeJSON(w, map[string]any{"started": true})
	go func() {
		defer busy.Store(false)
		rep := creatorReporter()
		stopSession()
		rep.status("Connecting to MicroPython on " + in.Port + "…")
		p, err := openReplPort(in.Port)
		if err != nil {
			emitCDone(false, friendly(err))
			return
		}
		m := newRepl(p)
		if err := m.enterRaw(); err != nil {
			p.Close()
			emitCDone(false, friendly(err))
			return
		}
		// main.py last, so a half-finished upload never runs.
		names := make([]string, 0, len(in.Files))
		total := 0
		for n, c := range in.Files {
			if n != "main.py" {
				names = append(names, n)
			}
			total += len(c)
		}
		if _, ok := in.Files["main.py"]; ok {
			names = append(names, "main.py")
		}
		done := 0
		for _, n := range names {
			rep.status(fmt.Sprintf("Copying %s to the board (%s)…", n, humanBytes(len(in.Files[n]))))
			if err := m.writeFile(n, []byte(in.Files[n]), rep, &done, &total); err != nil {
				m.exitRaw()
				p.Close()
				emitCDone(false, friendly(err))
				return
			}
		}
		rep.status("Restarting MicroPython so your program runs…")
		m.exitRaw()
		time.Sleep(100 * time.Millisecond)
		p.Write([]byte{0x04}) // soft reboot → runs main.py
		startSession(p, in.Port, "repl")
		emitCDone(true, "Uploaded and running — output appears in the console. main.py also runs every time the board starts.")
	}()
}

func handleCreatorStop(w http.ResponseWriter, r *http.Request) {
	if err := sessionWrite([]byte{0x03, 0x03}); err != nil {
		jsonErr(w, err.Error())
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

func handleCreatorSend(w http.ResponseWriter, r *http.Request) {
	var in struct{ Text string }
	decodeJSON(r, &in)
	if err := sessionWrite([]byte(in.Text)); err != nil {
		jsonErr(w, err.Error())
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

func handleCreatorDisconnect(w http.ResponseWriter, r *http.Request) {
	stopSession()
	writeJSON(w, map[string]any{"ok": true})
}

func handleCreatorMonitor(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Port string
		Baud int
	}
	decodeJSON(r, &in)
	if in.Baud <= 0 {
		in.Baud = 115200
	}
	if busy.Load() {
		jsonErr(w, "Flashie is busy with another job — wait for it to finish.")
		return
	}
	stopSession()
	var p Port
	var err error
	if demoMode {
		p, err = openReplPort(in.Port)
	} else {
		p, err = openSerialEx(in.Port, in.Baud, false)
	}
	if err != nil {
		jsonErr(w, friendly(err))
		return
	}
	startSession(p, in.Port, "serial")
	writeJSON(w, map[string]any{"ok": true})
}
