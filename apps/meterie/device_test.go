package meterie

import (
	"sync"
	"testing"
	"time"
)

// fakePort plays back a byte stream, answering polls if respond is set.
type fakePort struct {
	mu      sync.Mutex
	data    []byte
	respond func(req []byte) []byte
	writes  int
}

func (f *fakePort) Read(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.data) == 0 {
		time.Sleep(5 * time.Millisecond)
		return 0, nil
	}
	n := copy(p, f.data[:min(len(f.data), 7)]) // deliver in small chunks like a UART
	f.data = f.data[n:]
	return n, nil
}

func (f *fakePort) Write(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.writes++
	if f.respond != nil {
		f.data = append(f.data, f.respond(p)...)
	}
	return len(p), nil
}
func (f *fakePort) Close() error { return nil }

func TestHP90EPCEndToEnd(t *testing.T) {
	drainInbox()
	a := NewApp()
	m := meterByID("holdpeak-hp90epc")
	d := &Device{ID: 1, Model: m, Kind: TrSerial, Path: "COM5", Cfg: m.Serial}
	ch := a.addChannel("CH1", KindMeter, m.Name(), d.ID)
	d.ChannelID = ch.ID
	a.devices = append(a.devices, d)
	d.stop = make(chan struct{})
	var stream []byte
	stream = append(stream, encodeFS9721("12.34", false, 0x4|0x2, 0, 0, 0, 0x4, 0)...)
	stream = append(stream, 0xFF, 0x00) // line noise
	stream = append(stream, encodeFS9721("12.35", false, 0x4|0x2, 0, 0, 0, 0x4, 0)...)
	stream = append(stream, encodeFS9721("0024", false, 0x2, 0, 0, 0, 0, 0x4)...) // 24 °C
	fp := &fakePort{data: stream}
	go func() { time.Sleep(300 * time.Millisecond); close(d.stop) }()
	d.pump(fp)
	a.processInbox()
	if len(ch.Samples) != 3 {
		t.Fatalf("want 3 samples, got %d", len(ch.Samples))
	}
	if ch.Last.Q != QTemperature || ch.Last.Value != 24 || ch.Samples[1].V != 12.35 {
		t.Fatalf("bad data: %+v %+v", ch.Last, ch.Samples)
	}
	if len(ch.Units) != 2 {
		t.Fatalf("unit change not tracked: %+v", ch.Units)
	}
}

func TestUT61EPlusPolling(t *testing.T) {
	drainInbox()
	a := NewApp()
	m := meterByID("uni-t-ut61eplus")
	d := &Device{ID: 2, Model: m, Kind: TrCP2110}
	ch := a.addChannel("CH1", KindMeter, m.Name(), d.ID)
	d.ChannelID = ch.ID
	a.devices = append(a.devices, d)
	d.stop = make(chan struct{})
	frame := []byte{0xAB, 0xCD, 0x10, 0x02, 0x31, 0x20, 0x31, 0x32, 0x2e, 0x33, 0x34, 0x35, 0x00, 0x01, 0x30, 0x30, 0x38}
	sum := 0
	for _, c := range frame {
		sum += int(c)
	}
	frame = append(frame, byte(sum>>8), byte(sum))
	fp := &fakePort{respond: func(req []byte) []byte {
		if string(req) == string(ut61epRequest) {
			return frame
		}
		return nil
	}}
	go func() { time.Sleep(700 * time.Millisecond); close(d.stop) }()
	d.pump(fp)
	a.processInbox()
	if fp.writes < 2 || len(ch.Samples) < 2 {
		t.Fatalf("polling failed: writes=%d samples=%d", fp.writes, len(ch.Samples))
	}
	if ch.Last.Q != QVoltage || !ch.Last.Flags.Has(FDC) || !near(ch.Last.Value, 12.345) {
		t.Fatalf("bad reading %+v", ch.Last)
	}
}

func TestMathAndLogging(t *testing.T) {
	drainInbox()
	a := NewApp()
	v := a.addChannel("V", KindMeter, "", 0)
	i := a.addChannel("I", KindMeter, "", 0)
	p := a.addChannel("P", KindMath, "", 0)
	p.Math = &MathDef{Preset: MathPower, A: v.ID, B: i.ID}
	now := time.Now()
	a.ingest(v, Reading{T: now, Value: 12, Q: QVoltage, Unit: "V"})
	a.ingest(i, Reading{T: now, Value: 0.5, Q: QCurrent, Unit: "A"})
	a.updateMath(map[int]bool{i.ID: true})
	if !p.HasLast || p.Last.Value != 6 || p.Last.Unit != "W" {
		t.Fatalf("power math wrong: %+v", p.Last)
	}
	// logging with a start trigger
	dir := t.TempDir()
	a.log.Path = dir + "/log.csv"
	a.log.Interval = 0
	a.log.StartMode = 2 // when above
	a.log.StartChan = v.ID
	a.log.StartVal = 13
	a.log.Start(a)
	if a.log.state != 1 {
		t.Fatalf("should be armed")
	}
	a.ingest(v, Reading{T: time.Now(), Value: 13.5, Q: QVoltage, Unit: "V"})
	a.log.tick(a)
	if a.log.state != 2 {
		t.Fatalf("trigger didn't start logging")
	}
	a.ingest(v, Reading{T: time.Now(), Value: 13.6, Q: QVoltage, Unit: "V"})
	a.log.Stop(a, "test")
	if a.log.rows < 1 {
		t.Fatalf("no rows written")
	}
	// alarms
	a.alarms = []*Alarm{{Chan: v.ID, UseHigh: true, High: 14, Enabled: true}}
	a.ingest(v, Reading{T: time.Now(), Value: 14.2, Q: QVoltage, Unit: "V"})
	if !a.anyAlarmActive() {
		t.Fatalf("alarm should be active")
	}
	// session round trip
	sp := dir + "/s.meterie"
	if err := a.saveSession(sp); err != nil {
		t.Fatal(err)
	}
	b := NewApp()
	if err := b.loadSession(sp); err != nil {
		t.Fatal(err)
	}
	if len(b.channels) != 3 || len(b.chanByID(v.ID).Samples) != len(v.Samples) || b.chanByID(p.ID).Math == nil {
		t.Fatalf("session round trip lost data")
	}
	// suite timeline round trip
	tp := dir + "/t.nst.csv"
	if err := a.writeTimeline(tp, -1<<62, 1<<62); err != nil {
		t.Fatal(err)
	}
	c := NewApp()
	n, err := c.importTimeline(tp)
	if err != nil || n != 3 {
		t.Fatalf("timeline import: n=%d err=%v", n, err)
	}
}
