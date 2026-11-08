package scopie

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func waitFrame(t *testing.T, e *Engine, d time.Duration, ok func(*Frame) bool) (*Frame, Taken) {
	t.Helper()
	deadline := time.Now().Add(d)
	var last Taken
	for time.Now().Before(deadline) {
		last = e.Take()
		if last.Error != "" {
			t.Fatalf("engine error: %s", last.Error)
		}
		if f := last.Frame; f != nil && ok(f) {
			return f, last
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("no suitable frame (state %s)", last.State)
	return nil, last
}

// The demo stream (a simulated 6022BE) must trigger in software, many times per second,
// with a stable trigger point.
func TestEngineStreamTriggers(t *testing.T) {
	e := newEngine()
	e.Start()
	defer e.Quit()
	sim := newSimStream()
	p := defaultPanel()
	p.TDiv = 200e-6
	p.Ch[0].VDiv = 0.5
	p.TrigLevel = 0.75 // middle of the 1 kHz sine (1.5 V amplitude, probe x1)
	p.ChannelsNeeded = []int{0, 1}
	e.SetDemo([]string{"sine", "square"}, []float64{1, 1})
	e.Apply(p)
	e.Connect(func() (Device, error) { return sim, nil })
	e.Run()
	f, _ := waitFrame(t, e, 5*time.Second, func(f *Frame) bool { return f.Triggered })
	if f.Data[0] == nil || f.Data[1] == nil {
		t.Fatal("expected two channels")
	}
	// the sample just after the trigger point is at or above the level, the one before below
	sc := f.Scales[0]
	at := func(i int) float64 { return (float64(f.Data[0][i]) - sc.Zero) * sc.K }
	if f.PreLen < 2 || f.PreLen >= len(f.Data[0]) {
		t.Fatalf("pre %d", f.PreLen)
	}
	before, after := at(f.PreLen-1), at(f.PreLen)
	if !(before < 0.75+0.15 && after > 0.75-0.15) {
		t.Errorf("trigger point not on a rising crossing: %v -> %v", before, after)
	}
	t0 := e.Take().Waveforms
	time.Sleep(time.Second)
	rate := e.Take().Waveforms - t0
	if rate < 100 {
		t.Errorf("only %d waveforms/s", rate)
	}
	t.Logf("%d waveforms/s, %d samples per frame, rate %.0f S/s", rate, len(f.Data[0]), 1/f.DT)
}

func TestEngineSingleAndNormal(t *testing.T) {
	e := newEngine()
	e.Start()
	defer e.Quit()
	p := defaultPanel()
	p.TrigMode = "normal"
	p.TrigLevel = 10 // never reached
	p.Ch[0].VDiv = 0.5
	e.SetDemo([]string{"sine"}, []float64{1})
	e.Apply(p)
	e.Connect(func() (Device, error) { return newSimStream(), nil })
	e.Run()
	time.Sleep(800 * time.Millisecond)
	tk := e.Take()
	if tk.Frame != nil {
		t.Errorf("normal mode drew a frame without a trigger")
	}
	if tk.State != "waiting" {
		t.Errorf("state %q want waiting", tk.State)
	}
	p.TrigLevel = 0.5
	e.Apply(p)
	e.Single()
	waitFrame(t, e, 3*time.Second, func(f *Frame) bool { return f.Triggered })
	time.Sleep(300 * time.Millisecond)
	if st := e.Take().State; st != "stopped" {
		t.Errorf("after single: state %q", st)
	}
}

func TestEngineBenchPersistAverage(t *testing.T) {
	e := newEngine()
	e.Start()
	defer e.Quit()
	p := defaultPanel()
	p.TDiv = 500e-6
	p.TrigLevel = 1
	p.Persist, p.Average = true, true
	for c := range p.Ch {
		p.Ch[c].VDiv = 0.5
	}
	e.Apply(p)
	e.Connect(func() (Device, error) { return newSimBench(), nil })
	e.Run()
	_, tk := waitFrame(t, e, 3*time.Second, func(f *Frame) bool { return f.Triggered })
	time.Sleep(300 * time.Millisecond)
	tk = e.Take()
	if tk.Hist[0] == nil || tk.AvgSum[0] == nil || tk.AvgN == 0 {
		t.Errorf("persistence/averaging data missing")
	}
	if tk.Frame == nil || tk.Frame.Data[3] == nil {
		t.Errorf("4-channel bench demo should deliver CH4")
	}
}

// ---------------------------------------------------------------- fake bench scope

// fakeRigol answers enough of a Rigol DS1054Z's SCPI to capture a sine.
type fakeRigol struct {
	mu   sync.Mutex
	cmds []string
	scal float64
}

func (r *fakeRigol) reply(cmd string) []byte {
	r.mu.Lock()
	r.cmds = append(r.cmds, cmd)
	r.mu.Unlock()
	up := strings.ToUpper(cmd)
	switch {
	case up == "*IDN?":
		return []byte("RIGOL TECHNOLOGIES,DS1054Z,DS1ZA000000001,00.04.04\n")
	case up == ":TRIG:STAT?":
		return []byte("TD\n")
	case up == ":WAV:PRE?":
		// format,type,points,count,xinc,xorig,xref,yinc,yorig,yref
		return []byte("0,0,1200,1,1.000000e-06,-6.000000e-04,0,4.000000e-02,0,128\n")
	case up == ":WAV:DATA?":
		data := make([]byte, 1200)
		for i := range data {
			data[i] = byte(128 + 50*math.Sin(2*math.Pi*float64(i)/600))
		}
		hdr := fmt.Sprintf("#9%09d", len(data))
		return append(append([]byte(hdr), data...), '\n')
	}
	return nil
}

func serveRaw(t *testing.T, r *fakeRigol) (string, func()) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				br := bufio.NewReader(c)
				for {
					ln, err := br.ReadString('\n')
					if err != nil {
						return
					}
					if out := r.reply(strings.TrimSpace(ln)); out != nil {
						c.Write(out)
					}
				}
			}(c)
		}
	}()
	return ln.Addr().String(), func() { ln.Close() }
}

func TestSCPIRigolOverRawSocket(t *testing.T) {
	r := &fakeRigol{}
	addr, stop := serveRaw(t, r)
	defer stop()
	cs := scanSCPI([]string{addr}, nil)
	if len(cs) != 1 || !cs[0].Ready || cs[0].Family != "Bench scope" {
		t.Fatalf("scan: %+v", cs)
	}
	if cs[0].Label != "Rigol Technologies DS1054Z" {
		t.Errorf("label %q", cs[0].Label)
	}
	e := newEngine()
	e.Start()
	defer e.Quit()
	p := defaultPanel()
	p.Ch[2].Enabled, p.Ch[3].Enabled = false, false
	e.Apply(p)
	e.Connect(cs[0].Factory)
	e.Run()
	f, tk := waitFrame(t, e, 5*time.Second, func(f *Frame) bool { return f.Data[0] != nil })
	if tk.Info.HDivs != 12 || tk.Info.Channels != 4 {
		t.Errorf("DS1054Z: hdivs %d channels %d", tk.Info.HDivs, tk.Info.Channels)
	}
	if !near(f.DT, 1e-6, 1e-9) || !near(f.TFirst, -6e-4, 1e-9) || len(f.Data[0]) != 1200 {
		t.Errorf("timing dt=%v t0=%v n=%d", f.DT, f.TFirst, len(f.Data[0]))
	}
	sc := f.Scales[0]
	peak := 0.0
	for _, x := range f.Data[0] {
		peak = math.Max(peak, (float64(x)-sc.Zero)*sc.K)
	}
	if !near(peak, 2.0, 0.03) { // 50 codes * 0.04 V
		t.Errorf("peak %v V want 2 V", peak)
	}
	r.mu.Lock()
	joined := strings.Join(r.cmds, "|")
	r.mu.Unlock()
	for _, want := range []string{":CHAN1:SCAL 0.1", ":TIM:MAIN:SCAL 0.001", ":TRIG:EDG:SLOP POS", ":TRIG:SWE AUTO", ":RUN"} {
		if !strings.Contains(joined, want) {
			t.Errorf("scope never got %q", want)
		}
	}
}

// ---------------------------------------------------------------- fake VXI-11

func rpcServe(c net.Conn, handle func(prog, proc uint32, args *xdrR) []byte) {
	defer c.Close()
	for {
		hdr := make([]byte, 4)
		if _, err := io.ReadFull(c, hdr); err != nil {
			return
		}
		n := binary.BigEndian.Uint32(hdr) & 0x7FFFFFFF
		rec := make([]byte, n)
		if _, err := io.ReadFull(c, rec); err != nil {
			return
		}
		r := &xdrR{b: rec}
		xid := r.u32()
		r.u32() // call
		r.u32() // rpc version
		prog := r.u32()
		r.u32() // version
		proc := r.u32()
		r.u32()
		r.opaque()
		r.u32()
		r.opaque()
		res := handle(prog, proc, r)
		var m xdr
		m.u32(xid)
		m.u32(1)
		m.u32(0)
		m.u32(0)
		m.u32(0)
		m.u32(0)
		m.Write(res)
		binary.BigEndian.PutUint32(hdr, 0x80000000|uint32(m.Len()))
		c.Write(append(hdr, m.Bytes()...))
	}
}

func TestVXI11(t *testing.T) {
	r := &fakeRigol{}
	core, _ := net.Listen("tcp", "127.0.0.1:0")
	pm, _ := net.Listen("tcp", "127.0.0.1:0")
	defer core.Close()
	defer pm.Close()
	_, corePort, _ := net.SplitHostPort(core.Addr().String())
	_, pmPort, _ := net.SplitHostPort(pm.Addr().String())
	old := vxiPortmap
	vxiPortmap = pmPort
	defer func() { vxiPortmap = old }()
	go func() {
		for {
			c, err := pm.Accept()
			if err != nil {
				return
			}
			go rpcServe(c, func(prog, proc uint32, a *xdrR) []byte {
				var x xdr
				p, _ := strconv.Atoi(corePort)
				x.u32(uint32(p))
				return x.Bytes()
			})
		}
	}()
	go func() {
		for {
			c, err := core.Accept()
			if err != nil {
				return
			}
			var pending []byte
			go rpcServe(c, func(prog, proc uint32, a *xdrR) []byte {
				var x xdr
				switch proc {
				case 10: // create_link
					x.u32(0)
					x.u32(7)
					x.u32(0)
					x.u32(1024)
				case 11: // device_write
					a.u32()
					a.u32()
					a.u32()
					a.u32()
					data := a.opaque()
					pending = r.reply(strings.TrimSpace(string(data)))
					x.u32(0)
					x.u32(uint32(len(data)))
				case 12: // device_read, in pieces of up to 500 bytes
					n := minInt(500, len(pending))
					reason := uint32(0)
					if n == len(pending) {
						reason = 4
					}
					x.u32(0)
					x.u32(reason)
					x.opaque(pending[:n])
					pending = pending[n:]
				default:
					x.u32(0)
				}
				return x.Bytes()
			})
		}
	}()
	l, idn, err := openNetLink("127.0.0.1", probeIDN)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	if _, isVXI := l.(*vxiLink); !isVXI || !strings.HasPrefix(idn, "RIGOL") {
		t.Fatalf("got %T %q", l, idn)
	}
	if err := lWrite(l, ":WAV:DATA?"); err != nil {
		t.Fatal(err)
	}
	b, err := lReadBlock(l, 2000)
	if err != nil || len(b) != 1200 {
		t.Fatalf("block: %d bytes, %v", len(b), err)
	}
}
