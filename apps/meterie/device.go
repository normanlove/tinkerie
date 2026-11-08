package meterie

// device.go: connection runtime. Each connected meter runs in its own
// goroutine and pushes readings into the app's inbox.

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"sync"
	"time"
)

// Port is a byte transport. Read returns (0, nil) on timeout.
type Port interface {
	Read(p []byte) (int, error)
	Write(p []byte) (int, error)
	Close() error
}

// PortInfo is one enumerated port/cable.
type PortInfo struct {
	Kind     Transport
	Path     string // COM5, HID device path, or BLE address
	Name     string // friendly name
	VID, PID uint16
	Hint     string
	Meters   []string
	BLEAddr  uint64
	RSSI     int
}

// Device is a live connection.
type Device struct {
	ID        int
	Model     *MeterModel
	Kind      Transport
	Path      string
	Label     string
	Cfg       SerialCfg
	BLEAddr   uint64
	ChannelID int

	mu        sync.Mutex
	status    string
	err       string
	connected bool
	rx        int
	frames    int
	lastRaw   []byte
	lastRx    time.Time
	stop      chan struct{}
	stopped   chan struct{}
	demo      *DemoSim
}

func (d *Device) setStatus(s string, err error, conn bool) {
	d.mu.Lock()
	d.status = s
	if err != nil {
		d.err = err.Error()
	} else {
		d.err = ""
	}
	d.connected = conn
	d.mu.Unlock()
	appWake()
}

// Snapshot returns status fields safely.
func (d *Device) Snapshot() (status, err string, conn bool, rx, frames int, raw []byte, last time.Time) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.status, d.err, d.connected, d.rx, d.frames, append([]byte(nil), d.lastRaw...), d.lastRx
}

// inbox receives readings from device goroutines.
type inboxItem struct {
	dev int
	ch  int // explicit channel (demo pairs) or 0
	r   Reading
}

var (
	inboxMu sync.Mutex
	inbox   []inboxItem
	wakeFn  func()
)

func pushReading(dev, ch int, r Reading) {
	inboxMu.Lock()
	inbox = append(inbox, inboxItem{dev, ch, r})
	inboxMu.Unlock()
	appWake()
}

func drainInbox() []inboxItem {
	inboxMu.Lock()
	defer inboxMu.Unlock()
	out := inbox
	inbox = nil
	return out
}

func appWake() {
	if wakeFn != nil {
		wakeFn()
	}
}

// openPortFor opens the platform transport for a device.
func openPortFor(d *Device) (Port, error) {
	switch d.Kind {
	case TrSerial:
		return platformOpenSerial(d.Path, d.Cfg)
	case TrCH9325:
		return platformOpenCH9325(d.Path, d.Cfg)
	case TrCP2110:
		return platformOpenCP2110(d.Path, d.Cfg)
	case TrVictor:
		return platformOpenHIDRaw(d.Path)
	case TrBU86X:
		return platformOpenBU86X(d.Path)
	case TrBLE:
		svc, chr := "", ""
		if d.Model != nil {
			svc, chr = d.Model.BLEService, d.Model.BLEChar
		}
		return platformOpenBLE(d.BLEAddr, svc, chr)
	}
	return nil, errors.New("unsupported transport")
}

// Start runs the device loop.
func (d *Device) Start() {
	d.stop = make(chan struct{})
	d.stopped = make(chan struct{})
	switch d.Kind {
	case TrDemo:
		go d.runDemo()
		return
	case TrManual, TrCamera:
		close(d.stopped)
		d.setStatus("Ready", nil, true)
		return
	}
	go d.run()
}

func (d *Device) Stop() {
	if d.stop == nil {
		return
	}
	select {
	case <-d.stop:
	default:
		close(d.stop)
	}
	select {
	case <-d.stopped:
	case <-time.After(1500 * time.Millisecond):
	}
}

func (d *Device) stopping() bool {
	select {
	case <-d.stop:
		return true
	default:
		return false
	}
}

func (d *Device) run() {
	defer close(d.stopped)
	defer func() {
		if r := recover(); r != nil {
			d.setStatus("Driver crashed", fmt.Errorf("%v", r), false)
		}
	}()
	backoff := time.Second
	for !d.stopping() {
		d.setStatus("Connecting…", nil, false)
		p, err := openPortFor(d)
		if err != nil {
			d.setStatus("Not connected", err, false)
			if !d.sleep(backoff) {
				return
			}
			if backoff < 5*time.Second {
				backoff += time.Second
			}
			continue
		}
		backoff = time.Second
		d.setStatus("Waiting for data…", nil, true)
		err = d.pump(p)
		p.Close()
		if d.stopping() {
			return
		}
		d.setStatus("Disconnected", err, false)
		if !d.sleep(time.Second) {
			return
		}
	}
}

func (d *Device) sleep(t time.Duration) bool {
	select {
	case <-d.stop:
		return false
	case <-time.After(t):
		return true
	}
}

func (d *Device) pump(p Port) error {
	dec := newDecoder(d.Model)
	poller, _ := dec.(Poller)
	framer, isFramer := dec.(FrameDecoder)
	frameTransport := d.Kind == TrBLE || d.Kind == TrVictor
	var nextPoll time.Time
	buf := make([]byte, 512)
	lastData := time.Now()
	gotAny := false
	for !d.stopping() {
		if poller != nil && time.Now().After(nextPoll) {
			if req := poller.PollRequest(); len(req) > 0 {
				if _, err := p.Write(req); err != nil {
					return err
				}
			}
			nextPoll = time.Now().Add(poller.PollInterval())
		}
		n, err := p.Read(buf)
		if err != nil {
			return err
		}
		if n == 0 {
			if time.Since(lastData) > 4*time.Second {
				msg := "No data — is the meter on with its PC link enabled?"
				if gotAny {
					msg = "Data stopped — meter asleep or link turned off?"
				}
				d.setStatus(msg, nil, true)
				lastData = time.Now()
			}
			continue
		}
		chunk := buf[:n]
		d.mu.Lock()
		d.rx += n
		d.lastRaw = append(d.lastRaw, chunk...)
		if len(d.lastRaw) > 64 {
			d.lastRaw = d.lastRaw[len(d.lastRaw)-64:]
		}
		d.lastRx = time.Now()
		d.mu.Unlock()
		var rs []Reading
		if isFramer && frameTransport {
			rs = framer.FeedFrame(chunk)
		} else {
			rs = dec.Feed(chunk)
		}
		for _, r := range rs {
			pushReading(d.ID, 0, r)
			d.mu.Lock()
			d.frames++
			d.mu.Unlock()
			if !gotAny || time.Since(lastData) > 2*time.Second {
				d.setStatus("Live", nil, true)
			}
			gotAny = true
			lastData = time.Now()
		}
	}
	return nil
}

// ProbeSerial tries each plan on a port and returns the first meter that yields valid frames.
func ProbeSerial(path string, progress func(string)) (*MeterModel, SerialCfg, error) {
	for _, pl := range serialProbePlans {
		m := meterByID(pl.MeterID)
		if m == nil {
			continue
		}
		if progress != nil {
			progress(fmt.Sprintf("Trying %s at %s…", m.Name(), pl.Cfg))
		}
		p, err := platformOpenSerial(path, pl.Cfg)
		if err != nil {
			return nil, SerialCfg{}, err
		}
		ok := probeOnce(p, newDecoder(m), 2300*time.Millisecond)
		p.Close()
		if ok {
			return m, pl.Cfg, nil
		}
	}
	return nil, SerialCfg{}, errors.New("no known meter protocol answered on " + path)
}

// ProbeHIDUART tries the protocols a UT-D04 cable is used with.
func ProbeHIDUART(path string, progress func(string)) (*MeterModel, SerialCfg, error) {
	for _, id := range []string{"uni-t-ut61e", "uni-t-ut61bcd", "uni-t-ut60e", "uni-t-ut71", "uni-t-ut60g"} {
		m := meterByID(id)
		if progress != nil {
			progress("Trying " + m.Name() + "…")
		}
		p, err := platformOpenCH9325(path, m.Serial)
		if err != nil {
			return nil, SerialCfg{}, err
		}
		ok := probeOnce(p, newDecoder(m), 2300*time.Millisecond)
		p.Close()
		if ok {
			return m, m.Serial, nil
		}
	}
	return nil, SerialCfg{}, errors.New("cable found, but no data — press the meter's RS232 button")
}

// ProbeCP2110 tells a UT61E+ family meter from a UT181A (same CP2110 cable).
func ProbeCP2110(path string, progress func(string)) (*MeterModel, SerialCfg, error) {
	for _, id := range []string{"uni-t-ut61eplus", "uni-t-ut181a"} {
		m := meterByID(id)
		if progress != nil {
			progress("Trying " + m.Name() + "…")
		}
		p, err := platformOpenCP2110(path, m.Serial)
		if err != nil {
			return nil, SerialCfg{}, err
		}
		ok := probeOnce(p, newDecoder(m), 3200*time.Millisecond)
		p.Close()
		if ok {
			return m, m.Serial, nil
		}
	}
	return nil, SerialCfg{}, errors.New("UNI-T cable found, but the meter didn't answer — is it switched on (UT181A: SETUP → Communication ON)?")
}

func probeOnce(p Port, dec Decoder, dur time.Duration) bool {
	poller, _ := dec.(Poller)
	end := time.Now().Add(dur)
	buf := make([]byte, 256)
	var next time.Time
	good := 0
	for time.Now().Before(end) {
		if poller != nil && time.Now().After(next) {
			if req := poller.PollRequest(); len(req) > 0 {
				p.Write(req)
			}
			next = time.Now().Add(poller.PollInterval())
		}
		n, err := p.Read(buf)
		if err != nil {
			return false
		}
		if n > 0 {
			good += len(dec.Feed(buf[:n]))
			if good >= 2 {
				return true
			}
		}
	}
	return good >= 1
}

// ---------------------------------------------------------------------------
// Demo / simulated meters so the app can be explored without hardware.

type DemoScenario int

const (
	DemoBattery DemoScenario = iota
	DemoPSU
	DemoThermo
	DemoMotor
	DemoResistors
	DemoLED
	DemoSolar
	DemoMains
)

var demoScenarioNames = []string{
	"Battery discharge (V)", "Bench supply with ripple (V)", "Soldering iron warm-up (°C)",
	"Motor current (A)", "Resistor sorting (Ω)", "LED forward voltage (diode)", "Solar panel sweep (V + I)",
	"AC mains (V AC)",
}

type DemoSim struct {
	Scenario DemoScenario
	t0       time.Time
	rng      *rand.Rand
	part     float64
	partT    time.Time
	hold     bool
}

func (d *Device) runDemo() {
	defer close(d.stopped)
	d.setStatus("Live (simulated)", nil, true)
	sim := d.demo
	sim.t0 = time.Now()
	sim.rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	tick := time.NewTicker(250 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-d.stop:
			return
		case now := <-tick.C:
			rs := sim.next(now)
			for i, r := range rs {
				ch := 0
				if i == 1 {
					ch = -1 // second channel of a pair
				}
				pushReading(d.ID, ch, r)
			}
			d.mu.Lock()
			d.frames += len(rs)
			d.mu.Unlock()
		}
	}
}

func (s *DemoSim) noise(a float64) float64 { return (s.rng.Float64()*2 - 1) * a }

func roundTo(v float64, step float64) float64 { return math.Round(v/step) * step }

func (s *DemoSim) next(now time.Time) []Reading {
	t := now.Sub(s.t0).Seconds()
	mk := func(q Quantity, unit string, v float64, f Flags, counts int, sig int) Reading {
		r := Reading{T: now, Q: q, Unit: unit, Value: v, Flags: f | FAuto, Counts: counts}
		num, pu := SplitSI(v, unit, sig)
		r.Display, r.Prefix = num, strings.TrimSuffix(pu, unit)
		return r
	}
	switch s.Scenario {
	case DemoBattery:
		// Li-ion like curve over ~6 minutes, then repeats
		x := math.Mod(t, 360) / 360
		v := 4.18 - 0.35*x - 0.25*math.Pow(x, 6) - 0.04*math.Exp(-t*2)
		v += s.noise(0.002)
		return []Reading{mk(QVoltage, "V", roundTo(v, 0.001), FDC, 4000, 4)}
	case DemoPSU:
		v := 5.02 + 0.03*math.Sin(t*2*math.Pi*0.7) + s.noise(0.004)
		if math.Mod(t, 40) > 30 {
			v -= 0.25 // load step
		}
		return []Reading{mk(QVoltage, "V", roundTo(v, 0.001), FDC, 6000, 4)}
	case DemoThermo:
		target := 350.0
		v := 22 + (target-22)*(1-math.Exp(-t/25)) + 3*math.Sin(t*0.9) + s.noise(0.6)
		return []Reading{mk(QTemperature, "°C", roundTo(v, 1), 0, 4000, 4)}
	case DemoMotor:
		v := 0.42 + 0.08*math.Sin(t*3) + s.noise(0.02)
		if math.Mod(t, 15) < 0.6 {
			v = 2.4 + s.noise(0.2) // inrush
		}
		return []Reading{mk(QCurrent, "A", roundTo(v, 0.001), FDC, 4000, 4)}
	case DemoResistors:
		if s.partT.IsZero() || now.Sub(s.partT) > 4*time.Second {
			s.partT = now
			if s.rng.Float64() < 0.25 {
				s.part = math.NaN()
			} else {
				vals := []float64{100, 220, 330, 470, 1000, 2200, 4700, 10000}
				s.part = vals[s.rng.Intn(len(vals))] * (1 + s.noise(0.04))
			}
		}
		if math.IsNaN(s.part) {
			r := mk(QResistance, "Ω", math.NaN(), 0, 4000, 4)
			r.Flags |= FOL
			r.Display = "OL"
			r.Prefix = "M"
			return []Reading{r}
		}
		return []Reading{mk(QResistance, "Ω", s.part*(1+s.noise(0.0005)), 0, 4000, 4)}
	case DemoLED:
		v := 1.82 + 0.4*math.Floor(math.Mod(t/6, 4)) + s.noise(0.002)
		r := mk(QDiode, "V", roundTo(v, 0.001), FDiode, 4000, 4)
		return []Reading{r}
	case DemoSolar:
		// sweep load: I from 0 to Isc
		x := math.Mod(t, 30) / 30
		isc, voc := 0.58, 21.6
		i := isc * x
		v := voc * (1 - math.Pow(x, 9)) * (1 - 0.06*x)
		if v < 0 {
			v = 0
		}
		return []Reading{
			mk(QVoltage, "V", roundTo(v+s.noise(0.01), 0.01), FDC, 6000, 4),
			mk(QCurrent, "A", roundTo(i+s.noise(0.002), 0.001), FDC, 6000, 4),
		}
	case DemoMains:
		v := 231.4 + 1.5*math.Sin(t*0.2) + s.noise(0.3)
		r := mk(QVoltage, "V", roundTo(v, 0.1), FAC, 6000, 4)
		r.Flags |= FHV
		return []Reading{r}
	}
	return nil
}
