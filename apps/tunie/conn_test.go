package tunie

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeMarlin pretends to be a Marlin board on a serial port.
type fakeMarlin struct {
	mu     sync.Mutex
	out    []byte
	inbuf  string
	closed bool
	got    []string
}

func (f *fakeMarlin) Read(b []byte) (int, error) {
	for i := 0; i < 20; i++ {
		f.mu.Lock()
		if f.closed {
			f.mu.Unlock()
			return 0, io.EOF
		}
		if len(f.out) > 0 {
			n := copy(b, f.out)
			f.out = f.out[n:]
			f.mu.Unlock()
			return n, nil
		}
		f.mu.Unlock()
		time.Sleep(5 * time.Millisecond)
	}
	return 0, nil
}

func (f *fakeMarlin) Write(b []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.inbuf += string(b)
	for {
		i := strings.IndexByte(f.inbuf, '\n')
		if i < 0 {
			break
		}
		l := f.inbuf[:i]
		f.inbuf = f.inbuf[i+1:]
		f.got = append(f.got, l)
		switch {
		case strings.HasPrefix(l, "M115"):
			f.out = append(f.out, "FIRMWARE_NAME:Marlin 2.1.2\nok\n"...)
		case strings.HasPrefix(l, "M503"):
			f.out = append(f.out, "echo:; Steps per unit:\necho:  M92 X80.00 Y80.00 Z400.00 E93.00\nok\n"...)
		case strings.HasPrefix(l, "M109"):
			f.out = append(f.out, " T:180.0 /210.0 B:60.0 /60.0 @:127\n T:209.8 /210.0 B:60.0 /60.0\nok\n"...)
		case strings.HasPrefix(l, "M303"):
			f.out = append(f.out, "PID Autotune start\necho:busy: processing\n#define DEFAULT_Kp 21.73\n#define DEFAULT_Ki 1.54\n#define DEFAULT_Kd 76.55\nok\n"...)
		default:
			f.out = append(f.out, "ok\n"...)
		}
	}
	return len(b), nil
}

func (f *fakeMarlin) Close() error { f.mu.Lock(); f.closed = true; f.mu.Unlock(); return nil }

func TestSerialProtocol(t *testing.T) {
	fm := &fakeMarlin{out: []byte("start\necho: External Reset\n")}
	lg := &LogBuf{}
	s := &SerialConn{port: fm, name: "FAKE", lines: make(chan string, 64), log: lg, closed: make(chan struct{}), temps: map[string]float64{}}
	go s.reader()
	time.Sleep(50 * time.Millisecond)
	s.drain()
	r, err := s.Send([]string{"M503"})
	if err != nil || PARSEcheck(r, "M92") == "" {
		t.Fatalf("M503: %v %v", r, err)
	}
	r, err = s.Send([]string{"M303 E0 S210 C8 U1"})
	if err != nil || !strings.Contains(strings.Join(r, "\n"), "DEFAULT_Kp 21.73") {
		t.Fatalf("M303: %v %v", r, err)
	}
	r, err = s.Send([]string{"M109 S210"})
	if err != nil || len(r) != 0 {
		t.Fatalf("M109 should hide temperature reports: %v %v", r, err)
	}
	if s.Status()["temps"].(map[string]any)["hotend"] != 209.8 {
		t.Fatalf("temps not parsed: %v", s.Status())
	}
	// streaming a whole job
	m := NewConnManager()
	m.c = s
	res, _ := Generate(GenRequest{Test: "first-layer", Printer: testPrinters[0], Settings: Settings{Hotend: 200, Bed: 60}})
	lines := cleanLines(strings.Split(res.Gcode, "\n"))
	job := &Job{Name: "x", Total: len(lines), State: "running"}
	m.job = job
	m.stream(s, job, lines)
	if job.State != "done" || job.Done != len(lines) {
		t.Fatalf("stream: %+v", job)
	}
	s.Close()
}

func PARSEcheck(r []string, sub string) string {
	for _, l := range r {
		if strings.Contains(l, sub) {
			return l
		}
	}
	return ""
}

func TestMoonraker(t *testing.T) {
	var scripts []string
	var uploaded string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/server/info":
			w.Write([]byte(`{"result":{"klippy_state":"ready"}}`))
		case "/printer/gcode/script":
			var b map[string]string
			json.NewDecoder(r.Body).Decode(&b)
			scripts = append(scripts, b["script"])
			w.Write([]byte(`{"result":"ok"}`))
		case "/server/gcode_store":
			if len(scripts) == 0 {
				w.Write([]byte(`{"result":{"gcode_store":[{"message":"old","time":10,"type":"response"}]}}`))
			} else {
				w.Write([]byte(`{"result":{"gcode_store":[{"message":"old","time":10,"type":"response"},{"message":"PID_CALIBRATE HEATER=extruder TARGET=210","time":11,"type":"command"},{"message":"// PID parameters: pid_Kp=22.865 pid_Ki=1.292 pid_Kd=101.178","time":12,"type":"response"}]}}`))
			}
		case "/server/files/upload":
			r.ParseMultipartForm(1 << 20)
			f, h, _ := r.FormFile("file")
			b, _ := io.ReadAll(f)
			uploaded = h.Filename + ":" + r.FormValue("print") + ":" + string(b[:10])
			w.Write([]byte(`{"result":"ok"}`))
		case "/printer/objects/query":
			w.Write([]byte(`{"result":{"status":{"extruder":{"temperature":25.1,"target":0},"heater_bed":{"temperature":24,"target":0},"print_stats":{"state":"standby"}}}}`))
		}
	}))
	defer srv.Close()
	m, err := openMoonraker(ConnConfig{URL: srv.URL}, &LogBuf{})
	if err != nil {
		t.Fatal(err)
	}
	out, err := m.Send([]string{"PID_CALIBRATE HEATER=extruder TARGET=210"})
	if err != nil || len(out) != 1 || !strings.Contains(out[0], "pid_Kp=22.865") {
		t.Fatalf("moonraker send: %v %v", out, err)
	}
	if err := m.Upload("t.gcode", []byte("; Tunie test file"), true); err != nil || uploaded != "t.gcode:true:; Tunie te" {
		t.Fatalf("upload: %v %q", err, uploaded)
	}
	if st := m.Status(); st["state"] != "standby" {
		t.Fatalf("status: %v", st)
	}
}

func TestOctoPrint(t *testing.T) {
	var cmds []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "KEY" {
			w.WriteHeader(403)
			return
		}
		switch r.URL.Path {
		case "/api/version":
			w.Write([]byte(`{"api":"0.1"}`))
		case "/api/printer/command":
			var b struct{ Commands []string }
			json.NewDecoder(r.Body).Decode(&b)
			cmds = b.Commands
			w.WriteHeader(204)
		case "/api/printer":
			w.Write([]byte(`{"temperature":{"tool0":{"actual":200.5,"target":210},"bed":{"actual":60,"target":60}},"state":{"text":"Operational"}}`))
		case "/api/files/local":
			w.WriteHeader(201)
			w.Write([]byte(`{"done":true}`))
		}
	}))
	defer srv.Close()
	o, err := openOctoPrint(ConnConfig{URL: srv.URL, APIKey: "KEY"}, &LogBuf{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := o.Send([]string{"G28", "M105"}); err != nil || len(cmds) != 2 {
		t.Fatalf("send %v %v", cmds, err)
	}
	if err := o.Upload("a.gcode", []byte("G28"), true); err != nil {
		t.Fatal(err)
	}
	if o.Status()["state"] != "Operational" {
		t.Fatal(o.Status())
	}
}
