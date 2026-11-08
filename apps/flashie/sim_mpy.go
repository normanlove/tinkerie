// Flashie — a simulated MicroPython board (raw REPL + friendly REPL) for
// tests and demo mode. It can't run Python, but it stores uploaded files and
// echoes the program's print() text so the console flow can be tried.
// Made by nrmluvdev.

package flashie

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

type simMPY struct {
	mu      sync.Mutex
	out     bytes.Buffer
	raw     bool
	buf     []byte
	line    []byte
	files   map[string][]byte
	cur     string
	running bool
	machine string
	execs   int
}

func newSimMPY(machine string) *simMPY {
	return &simMPY{files: map[string][]byte{}, machine: machine}
}

func (s *simMPY) banner() string {
	return "\r\nMicroPython v1.28.0 on 2026-04-06; " + s.machine + "\r\nType \"help()\" for more information.\r\n>>> "
}

var (
	reOpen  = regexp.MustCompile(`open\('([^']+)','wb'\)`)
	reChunk = regexp.MustCompile(`_w\(_u\('([0-9a-f]*)'\)\)`)
	rePrint = regexp.MustCompile(`print\(\s*"([^"]*)"`)
)

func (s *simMPY) execRaw(code string) (string, string) {
	s.execs++
	switch {
	case strings.Contains(code, "sys.implementation"):
		plat := "esp32"
		if strings.Contains(s.machine, "RP2") {
			plat = "rp2"
		}
		return "micropython 1.28.0 " + plat + "\r\n" + s.machine + "\r\n", ""
	case reOpen.MatchString(code):
		s.cur = reOpen.FindStringSubmatch(code)[1]
		s.files[s.cur] = nil
	case reChunk.MatchString(code):
		b, _ := hex.DecodeString(reChunk.FindStringSubmatch(code)[1])
		s.files[s.cur] = append(s.files[s.cur], b...)
	case strings.Contains(code, "_f.close()"):
		s.cur = ""
	case strings.Contains(code, "raise"):
		return "", "Traceback (most recent call last):\r\nValueError: demo error\r\n"
	}
	return "", ""
}

func (s *simMPY) runMain() {
	s.out.WriteString("MPY: soft reboot\r\n")
	main := string(s.files["main.py"])
	if main == "" {
		s.out.WriteString(s.banner())
		return
	}
	for _, m := range rePrint.FindAllStringSubmatch(main, 6) {
		s.out.WriteString(m[1] + "\r\n")
	}
	s.out.WriteString("[demo board: your program is running — on real hardware its output streams here]\r\n")
	s.running = true
}

func (s *simMPY) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range p {
		switch {
		case c == 0x03:
			if s.running {
				s.running = false
				s.out.WriteString("Traceback (most recent call last):\r\n  File \"main.py\", line 1\r\nKeyboardInterrupt: \r\n")
			}
			if !s.raw {
				s.out.WriteString("\r\n>>> ")
			}
			s.buf, s.line = nil, nil
		case c == 0x01:
			s.raw, s.buf = true, nil
			s.out.WriteString("raw REPL; CTRL-B to exit\r\n>")
		case c == 0x02:
			s.raw = false
			s.out.WriteString(s.banner())
		case c == 0x04 && s.raw:
			o, e := s.execRaw(string(s.buf))
			s.buf = nil
			s.out.WriteString("OK" + o + "\x04" + e + "\x04>")
		case c == 0x04:
			s.runMain()
		case s.raw:
			s.buf = append(s.buf, c)
		case c == '\r':
			l := string(s.line)
			s.line = nil
			s.out.WriteString("\r\n")
			if m := regexp.MustCompile(`^print\((.*)\)$`).FindStringSubmatch(strings.TrimSpace(l)); m != nil {
				s.out.WriteString(strings.Trim(m[1], `"'`) + "\r\n")
			} else if strings.TrimSpace(l) != "" {
				s.out.WriteString("[demo board: can't run Python — on real hardware this line runs]\r\n")
			}
			s.out.WriteString(">>> ")
		case c == '\n':
		default:
			s.line = append(s.line, c)
			s.out.WriteByte(c)
		}
	}
	return len(p), nil
}

func (s *simMPY) ReadTimeout(p []byte, d time.Duration) (int, error) {
	deadline := time.Now().Add(d)
	for {
		s.mu.Lock()
		if s.out.Len() > 0 {
			n, _ := s.out.Read(p)
			s.mu.Unlock()
			return n, nil
		}
		s.mu.Unlock()
		if time.Now().After(deadline) {
			return 0, nil
		}
		time.Sleep(3 * time.Millisecond)
	}
}
func (s *simMPY) SetBaud(int) error { return nil }
func (s *simMPY) SetDTR(bool) error { return nil }
func (s *simMPY) SetRTS(bool) error { return nil }
func (s *simMPY) Purge() error      { s.mu.Lock(); s.out.Reset(); s.mu.Unlock(); return nil }
func (s *simMPY) Close() error      { return nil }

// ---- demo wiring ----

var (
	demoMPYPorts = map[string]*simMPY{}
	demoExtra    []PortInfo
)

// demoMPY returns the simulated MicroPython board on a port, if installed.
func demoMPY(name string) Port {
	demoMu.Lock()
	defer demoMu.Unlock()
	if s, ok := demoMPYPorts[name]; ok {
		return s
	}
	return nil
}

// demoInstallMPY is called after "flashing" MicroPython in demo mode.
func demoInstallMPY(port, chip string) {
	demoMu.Lock()
	defer demoMu.Unlock()
	if strings.HasSuffix(port, ":") || chip == "RP2040" || chip == "RP2350" {
		// A Pico reboots into MicroPython and shows up as a new COM port.
		name := "COM11"
		demoMPYPorts[name] = newSimMPY("Raspberry Pi Pico with " + chip)
		found := false
		for _, p := range demoExtra {
			if p.Name == name {
				found = true
			}
		}
		if !found {
			pi := PortInfo{Name: name, USB: true, VID: 0x2E8A, PID: 0x0005, Friendly: "USB Serial Device (" + name + ")"}
			pi.Guess = guessBoard(pi)
			demoExtra = append(demoExtra, pi)
		}
		return
	}
	demoMPYPorts[port] = newSimMPY("Generic " + chip + " module with " + chip)
}

func demoArduinoBuild(code string, r *reporter) ([]byte, error) {
	r.status("Compiling your sketch (demo: pretending — real builds use arduino-cli)…")
	emitConsole("$ arduino-cli compile --fqbn arduino:avr:uno flashie_sketch\n", "build")
	time.Sleep(400 * time.Millisecond)
	if !strings.Contains(code, "void setup()") {
		return nil, fmt.Errorf("flashie_sketch.ino: error: 'setup' was not declared")
	}
	emitConsole("Sketch uses 1874 bytes (5%) of program storage space. Maximum is 32256 bytes.\n", "build")
	return makeHexFile(fakeImage("sketch", 1874), 0), nil
}
