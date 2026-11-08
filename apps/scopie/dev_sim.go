// Demo mode: simulated scopes so every feature can be tried without hardware.
//
//   - simStream behaves like a Hantek 6022BE (2 channels, raw stream)
//   - simBench behaves like a 4-channel bench scope that triggers by itself
package scopie

import (
	"math"
	"math/rand"
	"sync"
	"time"
)

type demoSignal struct{ Key, Label, Tip string }

var demoSignals = []demoSignal{
	{"cal", "Probe calibration square", "The 2 V square wave from the scope's calibration tab (follows the 'Cal output' setting)."},
	{"sine", "Sine 1 kHz, 3 Vpp", "A clean sine wave, like a signal generator or audio tone."},
	{"sine_shift", "Sine 1 kHz, 2 Vpp, 60° later", "Same frequency as the CH1 sine but delayed - try the Phase measurement."},
	{"square", "Square 5 kHz with ringing", "A fast square wave with overshoot, like a real logic signal on a long wire."},
	{"triangle", "Triangle 500 Hz", "Rises and falls in straight lines."},
	{"saw", "Sawtooth 2 kHz", "Ramps up slowly and drops instantly."},
	{"pwm", "PWM 2 kHz, 25 % duty", "Pulses like those used to dim LEDs or drive motors."},
	{"noisy", "Noisy 50 Hz sine", "A mains-like sine buried in noise - try averaging."},
	{"am", "AM radio-style 20 kHz", "A 20 kHz carrier whose size is changed by a 500 Hz tone."},
	{"burst", "Burst (5 cycles every 2 ms)", "Short bursts - great for practising triggering."},
	{"glitch", "Square with rare glitch", "A 1 kHz square wave with an occasional short spike - try persistence to catch it."},
	{"dc", "DC 1.5 V (AA battery)", "A steady voltage, like a battery."},
	{"off", "Nothing connected", "Just a little noise around 0 V."},
}

func scanSim(lan []string, notes *[]string) []*Candidate {
	return []*Candidate{
		{Label: "Demo: USB scope (like a Hantek 6022BE)", Family: "Demo mode", Detail: "Simulated signals, 2 channels", Ready: true,
			Factory: func() (Device, error) { return newSimStream(), nil }},
		{Label: "Demo: 4-channel bench scope", Family: "Demo mode", Detail: "Simulated signals, 4 channels, hardware trigger", Ready: true,
			Factory: func() (Device, error) { return newSimBench(), nil }},
	}
}

// gaussian noise table: cheap normally-distributed noise for the demo.
var noiseTab = func() []float64 {
	r := rand.New(rand.NewSource(1))
	t := make([]float64, 1<<16)
	for i := range t {
		t[i] = r.NormFloat64()
	}
	return t
}()

type noiseGen struct{ s uint32 }

func (n *noiseGen) next() float64 {
	n.s ^= n.s << 13
	n.s ^= n.s >> 17
	n.s ^= n.s << 5
	return noiseTab[n.s&0xFFFF]
}

// signalSource generates the demo signals (volts at the probe tip).
type signalSource struct {
	mu      sync.Mutex
	signals [maxCh]string
	probes  [maxCh]float64
	calFreq float64
	noise   noiseGen
}

func newSignalSource() *signalSource {
	return &signalSource{signals: [maxCh]string{"cal", "sine", "square", "triangle"},
		probes: [maxCh]float64{1, 1, 1, 1}, calFreq: 1000, noise: noiseGen{2463534242}}
}

func (s *signalSource) set(sigs []string, probes []float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := 0; i < maxCh; i++ {
		if i < len(sigs) && sigs[i] != "" {
			s.signals[i] = sigs[i]
		}
		if i < len(probes) && probes[i] > 0 {
			s.probes[i] = probes[i]
		} else {
			s.probes[i] = 1
		}
	}
}

func (s *signalSource) snapshot() ([maxCh]string, [maxCh]float64, float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.signals, s.probes, s.calFreq
}

func frac1(x float64) float64 { return x - math.Floor(x) }

// value of a demo signal at time t (volts at the probe tip)
func (s *signalSource) value(kind string, t, calFreq float64) float64 {
	const tau = 2 * math.Pi
	switch kind {
	case "cal":
		if frac1(t*calFreq) < 0.5 {
			return 2
		}
		return 0
	case "sine":
		return 1.5 * math.Sin(tau*1000*t)
	case "sine_shift":
		return math.Sin(tau*1000*t - math.Pi/3)
	case "square":
		const f = 5000.0
		ph := frac1(t * f)
		base, since, sign := 3.3, ph/f, 1.0
		if ph >= 0.5 {
			base, since, sign = 0, (ph-0.5)/f, -1
		}
		return base + 0.6*sign*math.Exp(-since/1.5e-6)*math.Cos(tau*1.2e6*since)
	case "triangle":
		ph := frac1(t * 500)
		return 2 * (2*math.Abs(2*ph-1) - 1)
	case "saw":
		return 2*frac1(t*2000) - 1
	case "pwm":
		if frac1(t*2000) < 0.25 {
			return 3.3
		}
		return 0
	case "noisy":
		return math.Sin(tau*50*t) + 0.35*s.noise.next()
	case "am":
		return (1 + 0.6*math.Sin(tau*500*t)) * 1.2 * math.Sin(tau*20e3*t)
	case "burst":
		ph := math.Mod(t, 2e-3)
		if ph < 0 {
			ph += 2e-3
		}
		if ph < 5/10e3 {
			return 1.5 * math.Sin(tau*10e3*ph)
		}
		return 0
	case "glitch":
		ph := frac1(t * 1000)
		v := -1.0
		if ph < 0.5 {
			v = 1
		}
		cyc := int64(math.Floor(t * 1000))
		if uint64(cyc*2654435761)%997 < 3 && math.Abs(ph-0.25) < 0.0015 {
			return 2.5
		}
		return v
	case "dc":
		return 1.5
	}
	return 0
}

func dcLevel(kind string) float64 {
	switch kind {
	case "cal":
		return 1
	case "pwm":
		return 0.825
	case "dc":
		return 1.5
	case "square":
		return 1.65
	}
	return 0
}

// ================================================================ simStream

var simRates = []RateID{{102, 20e3}, {104, 40e3}, {105, 50e3}, {110, 100e3}, {120, 200e3}, {150, 500e3},
	{1, 1e6}, {2, 2e6}, {4, 4e6}, {5, 5e6}, {8, 8e6}, {10, 10e6}, {12, 12e6}, {15, 15e6}, {24, 24e6}, {30, 30e6}}

// generation budget (samples per second, all channels) so the PC stays responsive
const simBudget = 16e6

type simStream struct {
	info  *DeviceInfo
	src   *signalSource
	mu    sync.Mutex
	rate  float64
	nch   int
	gains [2]int
	dc    [2]bool
	hwAC  bool
	stop  chan struct{}
	wg    sync.WaitGroup
	noise noiseGen
}

func newSimStream() *simStream {
	var rates []float64
	for _, r := range simRates {
		rates = append(rates, r.Rate)
	}
	info := defaultInfo("Demo mode (simulated USB scope)")
	info.VDivs = []float64{0.01, 0.02, 0.05, 0.1, 0.2, 0.5, 1, 2, 5}
	info.SampleRates = rates
	info.MaxRate1, info.MaxRateMulti = 30e6, 15e6
	info.ACDCModOption, info.CalOutput = true, true
	info.CalFreqs = []float64{50, 100, 200, 500, 1000, 2000, 5000, 10000, 20000, 50000, 100000}
	info.Simulated = true
	info.Notes = "Simulated signals - no hardware needed."
	return &simStream{info: info, src: newSignalSource(), rate: 1e6, nch: 2, gains: [2]int{1, 1},
		dc: [2]bool{true, true}, noise: noiseGen{0x9E3779B9}}
}

func (d *simStream) Info() *DeviceInfo               { return d.info }
func (d *simStream) Open(func(string)) error         { return nil }
func (d *simStream) Close()                          { d.StopStream() }
func (d *simStream) SetDemo(s []string, p []float64) { d.src.set(s, p) }
func (d *simStream) Gains() []Gain                   { return []Gain{{1, 5.12}, {2, 2.56}, {5, 1.024}, {10, 0.512}} }
func (d *simStream) Scale(ch, gain int) Scale        { return Scale{5.12 / (float64(gain) * 128), 128} }
func (d *simStream) RateIDs(nch int) []RateID {
	mx := 15e6
	if nch == 1 {
		mx = 30e6
	}
	var out []RateID
	for _, r := range simRates {
		if r.Rate <= mx {
			out = append(out, r)
		}
	}
	return out
}

func (d *simStream) Configure(rid, nch int, gains [2]int, dc [2]bool, mod bool, cal float64) (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, r := range simRates {
		if r.ID == rid {
			d.rate = r.Rate
		}
	}
	d.nch, d.gains, d.dc, d.hwAC = nch, gains, dc, mod
	d.src.mu.Lock()
	d.src.calFreq = cal
	d.src.mu.Unlock()
	return nch, nil
}

func (d *simStream) StartStream(chunk int, fn func([]byte), fail func(error)) error {
	d.StopStream()
	d.mu.Lock()
	rate, nch, gains, dc, hwAC := d.rate, d.nch, d.gains, d.dc, d.hwAC
	d.mu.Unlock()
	stop := make(chan struct{})
	d.stop = stop
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		perChunk := chunk / nch
		budget := math.Min(rate, simBudget/float64(nch))
		buf := make([]byte, perChunk*nch)
		var index int64
		t0 := time.Now()
		for {
			select {
			case <-stop:
				return
			default:
			}
			due := int64(time.Since(t0).Seconds() * budget)
			if due-index < int64(perChunk) {
				wait := float64(index+int64(perChunk)-due) / budget
				time.Sleep(time.Duration(math.Min(wait, 0.01) * 1e9))
				continue
			}
			nChunks := minI64(8, (due-index)/int64(perChunk))
			if due-index > int64(perChunk)*64 { // far behind: skip ahead
				index = due - int64(perChunk)*nChunks
			}
			sigs, probes, cal := d.src.snapshot()
			for k := int64(0); k < nChunks; k++ {
				for c := 0; c < nch; c++ {
					sc := d.Scale(c, gains[c])
					kind := sigs[c]
					off := 0.0
					if !dc[c] && hwAC {
						off = dcLevel(kind) / probes[c]
					}
					for i := 0; i < perChunk; i++ {
						t := float64(index+int64(i)) / rate
						v := d.src.value(kind, t, cal)/probes[c] - off
						adc := v/sc.K + sc.Zero + 0.6*d.noise.next()
						buf[i*nch+c] = uint8(clampF(math.Round(adc), 0, 255))
					}
				}
				index += int64(perChunk)
				fn(buf)
			}
		}
	}()
	return nil
}

func (d *simStream) StopStream() {
	if d.stop != nil {
		close(d.stop)
		d.wg.Wait()
		d.stop = nil
	}
}

// ================================================================ simBench

type simBench struct {
	info    *DeviceInfo
	src     *signalSource
	panel   *Panel
	running bool
	single  bool
	t0      time.Time
	last    time.Time
	noise   noiseGen
}

const simBenchPoints = 1200

func newSimBench() *simBench {
	info := frameInfo("Demo mode (simulated 4-channel bench scope)", 4)
	info.TDivs = stepsBetween(5e-9, 50, tdiv125)
	info.HDivs = 12
	info.Simulated = true
	info.Notes = "Simulated bench scope - frames arrive about 30 times a second."
	return &simBench{info: info, src: newSignalSource(), t0: time.Now(), noise: noiseGen{12345}}
}

func (d *simBench) Info() *DeviceInfo               { return d.info }
func (d *simBench) Open(func(string)) error         { return nil }
func (d *simBench) Close()                          {}
func (d *simBench) SetDemo(s []string, p []float64) { d.src.set(s, p) }
func (d *simBench) ApplyPanel(p *Panel) error       { cp := *p; d.panel = &cp; return nil }
func (d *simBench) Start(single bool) error         { d.running, d.single = true, single; return nil }
func (d *simBench) Stop() error                     { d.running = false; return nil }

func (d *simBench) ReadFrame(timeout float64) (*FrameData, error) {
	p := d.panel
	if !d.running || p == nil {
		sleepS(timeout)
		return nil, nil
	}
	wait := 1.0/30 - time.Since(d.last).Seconds()
	if wait > 0 {
		sleepS(math.Min(wait, timeout))
		if wait > timeout {
			return nil, nil
		}
	}
	d.last = time.Now()
	n := simBenchPoints
	hd := float64(d.info.HDivs)
	span := p.TDiv * hd
	dt := span / float64(n)
	now := time.Since(d.t0).Seconds()
	sigs, probes, cal := d.src.snapshot()
	src := p.TrigSource
	volts := func(c int, t float64) float64 {
		v := d.src.value(sigs[c], t, cal) / probes[c]
		if p.Ch[c].Coupling == "AC" {
			v -= dcLevel(sigs[c]) / probes[c]
		}
		return v
	}
	// find a trigger: search a window for a crossing
	pre := (p.TrigX + hd/2) / hd * span
	lvl := p.TrigLevel
	tTrig, triggered := 0.0, false
	prev := volts(src, now)
	for i := 1; i < 4*n; i++ {
		t := now + float64(i)*dt
		v := volts(src, t)
		hit := false
		switch p.TrigEdge {
		case "falling":
			hit = prev > lvl && v <= lvl
		case "both":
			hit = (prev < lvl && v >= lvl) || (prev > lvl && v <= lvl)
		default:
			hit = prev < lvl && v >= lvl
		}
		prev = v
		if hit && float64(i-1)*dt >= pre {
			tTrig, triggered = t, true
			break
		}
	}
	if !triggered {
		if p.TrigMode != "auto" {
			return nil, nil
		}
		tTrig = now + pre
	}
	fd := newFrameData()
	fd.DT, fd.TFirst, fd.Triggered = dt, -pre, triggered
	for c := 0; c < 4; c++ {
		fd.Ranges[c] = CodeRange{-128, 127}
		cp := p.Ch[c]
		if !cp.Enabled {
			continue
		}
		k := cp.VDiv / 25 // 25 codes per division, like many scopes
		zero := -cp.Pos * 25
		codes := make([]float32, n)
		for i := range codes {
			t := tTrig - pre + float64(i)*dt
			code := volts(c, t)/k + zero + 0.4*d.noise.next()
			codes[i] = float32(clampF(math.Round(code), -128, 127))
		}
		fd.Codes[c] = codes
		fd.Scales[c] = Scale{k, zero}
	}
	if d.single && triggered {
		d.running = false
	}
	return fd, nil
}

func sleepS(s float64) {
	if s > 0 {
		time.Sleep(time.Duration(s * 1e9))
	}
}
