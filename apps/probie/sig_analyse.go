// Probie — Signals: working out what each channel of a capture is doing, and
// explaining it in plain words, the way Probie explains lines of code.
// Part of the Tinkerie maker suite. Created by nrmluvdev.

package probie

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// SigReport is what the window shows for a capture.
type SigReport struct {
	File     string        `json:"file"`
	Source   string        `json:"source"`
	App      string        `json:"app"`
	Duration float64       `json:"duration"`
	Summary  []string      `json:"summary"`
	Chans    []SigChanView `json:"chans"`
	Cards    []SigCard     `json:"cards"`
	Decoded  []SigFrame    `json:"decoded"`
	Concepts []SigConcept  `json:"concepts"`
	Warnings []SigWarn     `json:"warnings"`
	Serial   []SerialLine  `json:"serial,omitempty"`
	Markers  []Marker      `json:"markers,omitempty"`
}

// SigChanView is a channel ready to draw.
type SigChanView struct {
	Name  string       `json:"name"`
	Unit  string       `json:"unit"`
	Kind  string       `json:"kind"`
	Role  string       `json:"role"`
	Min   float64      `json:"min"`
	Max   float64      `json:"max"`
	Plot  [][3]float64 `json:"plot,omitempty"`  // analog/meter: t, low, high per column
	Edges [][2]float64 `json:"edges,omitempty"` // digital: t, level (2 = too busy to draw)
}

// SigCard is one explanation, like a line card in the code view.
type SigCard struct {
	Ch    int         `json:"ch"` // -1 = the whole capture
	Tag   string      `json:"tag"`
	Title string      `json:"title"`
	What  string      `json:"what"`
	Why   string      `json:"why"`
	Facts [][2]string `json:"facts,omitempty"`
	T0    float64     `json:"t0"`
	T1    float64     `json:"t1"`
}

// SigFrame is one decoded piece of a protocol.
type SigFrame struct {
	T0    float64 `json:"t0"`
	T1    float64 `json:"t1"`
	Ch    int     `json:"ch"`
	Proto string  `json:"proto"`
	Text  string  `json:"text"`
	Note  string  `json:"note"`
	Kind  string  `json:"kind"` // data, address, start, stop, ack, nack, error
}

// SigConcept is a glossary entry used by this capture.
type SigConcept struct {
	Name  string `json:"name"`
	Text  string `json:"text"`
	Chans []int  `json:"chans"`
}

// SigWarn is something worth checking.
type SigWarn struct {
	Title string  `json:"title"`
	Text  string  `json:"text"`
	Ch    int     `json:"ch"`
	T0    float64 `json:"t0"`
	T1    float64 `json:"t1"`
}

type sigCtx struct {
	c        *SigCapture
	r        *SigReport
	concepts map[string]*SigConcept
	order    []string
	used     map[int]bool // channels explained by a multi-channel protocol
	t0, t1   float64
}

func (x *sigCtx) concept(name string, ch int) {
	k, ok := x.concepts[name]
	if !ok {
		k = &SigConcept{Name: name, Text: sigGlossary[name]}
		x.concepts[name] = k
		x.order = append(x.order, name)
	}
	if ch >= 0 {
		for _, c := range k.Chans {
			if c == ch {
				return
			}
		}
		k.Chans = append(k.Chans, ch)
	}
}

func (x *sigCtx) card(c SigCard) { x.r.Cards = append(x.r.Cards, c) }
func (x *sigCtx) warn(w SigWarn) { x.r.Warnings = append(x.r.Warnings, w) }

// AnalyseCapture explains a capture.
func AnalyseCapture(c *SigCapture) *SigReport {
	r := &SigReport{File: c.File, Source: c.Source, App: c.App, Markers: c.Markers}
	x := &sigCtx{c: c, r: r, concepts: map[string]*SigConcept{}, used: map[int]bool{}}
	x.t0, x.t1 = math.Inf(1), math.Inf(-1)
	for _, ch := range c.Chans {
		if len(ch.T) > 0 {
			x.t0 = math.Min(x.t0, ch.T[0])
			x.t1 = math.Max(x.t1, ch.T[len(ch.T)-1])
		}
	}
	for _, l := range c.Serial {
		x.t0 = math.Min(x.t0, l.T)
		x.t1 = math.Max(x.t1, l.T)
	}
	if math.IsInf(x.t0, 0) {
		x.t0, x.t1 = 0, 0
	}
	r.Duration = x.t1 - x.t0
	for _, ch := range c.Chans {
		r.Chans = append(r.Chans, viewOf(ch, x.t0, x.t1))
	}

	// digital views of every channel (analog channels that switch between two levels count too)
	dig := make([]*digi, len(c.Chans))
	for i, ch := range c.Chans {
		switch ch.Kind {
		case KindDigital:
			dig[i] = newDigi(ch.T, ch.V, x.t0, x.t1)
		case KindAnalog:
			if d := analogToDigi(ch, x.t0, x.t1); d != nil {
				dig[i] = d
			}
		}
	}
	// protocols that need several wires
	if len(c.Serial) > 0 {
		x.explainSerialLog()
	}
	x.findI2C(dig)
	x.findSPI(dig)
	for i, ch := range c.Chans {
		switch ch.Kind {
		case KindAnalog:
			x.explainAnalog(i, ch, dig[i])
		case KindDigital:
			if !x.used[i] {
				x.explainDigital(i, ch.Name, dig[i], nil)
			}
		case KindMeter:
			x.explainMeter(i, ch)
		}
	}
	x.explainPower()
	x.overview()
	for _, k := range x.order {
		r.Concepts = append(r.Concepts, *x.concepts[k])
	}
	sort.SliceStable(r.Decoded, func(a, b int) bool { return r.Decoded[a].T0 < r.Decoded[b].T0 })
	if len(r.Decoded) > 5000 {
		r.Decoded = r.Decoded[:5000]
	}
	if len(c.Serial) > 3000 {
		r.Serial = c.Serial[:3000]
	} else {
		r.Serial = c.Serial
	}
	// empty lists, not null, for the window
	if r.Decoded == nil {
		r.Decoded = []SigFrame{}
	}
	if r.Warnings == nil {
		r.Warnings = []SigWarn{}
	}
	if r.Concepts == nil {
		r.Concepts = []SigConcept{}
	}
	if r.Chans == nil {
		r.Chans = []SigChanView{}
	}
	return r
}

// ---------------------------------------------------------------- drawing data

func viewOf(ch *SigChan, t0, t1 float64) SigChanView {
	v := SigChanView{Name: ch.Name, Unit: ch.Unit, Kind: ch.Kind}
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, y := range ch.V {
		if !math.IsNaN(y) {
			lo, hi = math.Min(lo, y), math.Max(hi, y)
		}
	}
	if math.IsInf(lo, 0) {
		lo, hi = 0, 0
	}
	v.Min, v.Max = lo, hi
	if ch.Kind == KindDigital {
		const maxEdges = 60000
		if len(ch.T) <= maxEdges {
			for i := range ch.T {
				v.Edges = append(v.Edges, [2]float64{ch.T[i], ch.V[i]})
			}
			return v
		}
		// too many edges to draw one by one: mark busy stretches
		w := (t1 - t0) / 30000
		i := 0
		for i < len(ch.T) {
			j := i
			for j+1 < len(ch.T) && ch.T[j+1]-ch.T[j] < w {
				j++
			}
			if j-i >= 3 {
				v.Edges = append(v.Edges, [2]float64{ch.T[i], 2})
				v.Edges = append(v.Edges, [2]float64{ch.T[j], ch.V[j]})
			} else {
				for k := i; k <= j; k++ {
					v.Edges = append(v.Edges, [2]float64{ch.T[k], ch.V[k]})
				}
			}
			i = j + 1
		}
		return v
	}
	const cols = 1600
	if len(ch.T) <= cols {
		for i := range ch.T {
			if !math.IsNaN(ch.V[i]) {
				v.Plot = append(v.Plot, [3]float64{ch.T[i], ch.V[i], ch.V[i]})
			}
		}
		return v
	}
	span := t1 - t0
	if span <= 0 {
		span = 1
	}
	k := 0
	for c := 0; c < cols; c++ {
		ce := t0 + span*float64(c+1)/cols
		mn, mx, tt := math.Inf(1), math.Inf(-1), 0.0
		for k < len(ch.T) && ch.T[k] <= ce {
			if y := ch.V[k]; !math.IsNaN(y) {
				mn, mx, tt = math.Min(mn, y), math.Max(mx, y), ch.T[k]
			}
			k++
		}
		if !math.IsInf(mn, 0) {
			v.Plot = append(v.Plot, [3]float64{tt, mn, mx})
		}
	}
	return v
}

// ---------------------------------------------------------------- number words

func siFmt(v float64, unit string) string { return strings.TrimSpace(siFmt0(v, unit)) }

func siFmt0(v float64, unit string) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "—"
	}
	if unit == "" {
		return trimNum(fmt.Sprintf("%.4g", v))
	}
	a := math.Abs(v)
	type p struct {
		f float64
		s string
	}
	ps := []p{{1e9, "G"}, {1e6, "M"}, {1e3, "k"}, {1, ""}, {1e-3, "m"}, {1e-6, "µ"}, {1e-9, "n"}, {1e-12, "p"}}
	if a == 0 {
		return "0 " + unit
	}
	if unit == "°C" || unit == "%" {
		return trimNum(fmt.Sprintf("%.3g", v)) + " " + unit
	}
	for _, q := range ps {
		if a >= q.f*0.9995 {
			return trimNum(fmt.Sprintf("%.3g", v/q.f)) + " " + q.s + unit
		}
	}
	return trimNum(fmt.Sprintf("%.3g", v/1e-12)) + " p" + unit
}

func trimNum(s string) string {
	if strings.Contains(s, "e") {
		return s
	}
	return s
}

func fmtTime(s float64) string { return siFmt(s, "s") }
func fmtHz(f float64) string   { return siFmt(f, "Hz") }
func fmtV(v float64) string    { return siFmt(v, "V") }
func pct(f float64) string     { return fmt.Sprintf("%.0f%%", f*100) }

func median(a []float64) float64 {
	if len(a) == 0 {
		return 0
	}
	b := append([]float64(nil), a...)
	sort.Float64s(b)
	return b[len(b)/2]
}

func meanStd(a []float64) (m, s float64) {
	n := 0.0
	for _, v := range a {
		if !math.IsNaN(v) {
			m += v
			n++
		}
	}
	if n == 0 {
		return 0, 0
	}
	m /= n
	for _, v := range a {
		if !math.IsNaN(v) {
			s += (v - m) * (v - m)
		}
	}
	return m, math.Sqrt(s / n)
}

// ---------------------------------------------------------------- analog channels

// analogToDigi turns a scope channel that switches between two levels into edges.
func analogToDigi(ch *SigChan, t0, t1 float64) *digi {
	if len(ch.V) < 20 {
		return nil
	}
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, v := range ch.V {
		lo, hi = math.Min(lo, v), math.Max(hi, v)
	}
	vpp := hi - lo
	if vpp < 0.3 {
		return nil
	}
	near := 0
	for _, v := range ch.V {
		if v < lo+0.2*vpp || v > hi-0.2*vpp {
			near++
		}
	}
	if float64(near)/float64(len(ch.V)) < 0.85 {
		return nil
	}
	th, hy := (lo+hi)/2, 0.1*vpp
	var et, ev []float64
	state := 0.0
	if ch.V[0] > th {
		state = 1
	}
	et, ev = append(et, ch.T[0]), append(ev, state)
	for i, v := range ch.V {
		if state == 0 && v > th+hy || state == 1 && v < th-hy {
			state = 1 - state
			et, ev = append(et, ch.T[i]), append(ev, state)
		}
	}
	d := newDigi(et, ev, t0, t1)
	// logic levels: typical low and high voltages
	var lows, highs []float64
	for _, v := range ch.V {
		if v < th {
			lows = append(lows, v)
		} else {
			highs = append(highs, v)
		}
	}
	d.vlo, d.vhi, d.analog = median(lows), median(highs), true
	d.dt = (ch.T[len(ch.T)-1] - ch.T[0]) / float64(len(ch.T)-1)
	return d
}

func (x *sigCtx) explainAnalog(i int, ch *SigChan, d *digi) {
	n := len(ch.V)
	if n < 4 {
		return
	}
	T0, T1 := ch.T[0], ch.T[n-1]
	dt := (T1 - T0) / float64(n-1)
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, v := range ch.V {
		lo, hi = math.Min(lo, v), math.Max(hi, v)
	}
	mean, std := meanStd(ch.V)
	rms := math.Sqrt(mean*mean + std*std)
	vpp := hi - lo
	unit := ch.Unit
	if unit == "" {
		unit = "V"
	}
	f := func(v float64) string { return siFmt(v, unit) }
	facts := [][2]string{{"Lowest", f(lo)}, {"Highest", f(hi)}, {"Peak to peak", f(vpp)}, {"Average", f(mean)}, {"RMS", f(rms)}, {"Samples", fmt.Sprintf("%d, one every %s (%s per second)", n, fmtTime(dt), siFmt(1/dt, "Sa"))}}
	x.concept("Sample rate", i)
	x.concept("Peak to peak", i)

	// steady DC?
	if vpp < math.Max(0.02, 0.04*math.Abs(mean)) || std < 0.004*math.Abs(mean) {
		what := fmt.Sprintf("**%s** is steady at **%s**: a DC level, not a changing signal.", ch.Name, f(mean))
		why := dcMeaning(mean, unit)
		x.concept("DC", i)
		rip := vpp
		facts = append(facts, [2]string{"Ripple and noise", f(rip) + " peak to peak"})
		if rf := rippleFreq(ch); rf > 0 && rip > 0.005 {
			facts = append(facts, [2]string{"Ripple repeats at", fmtHz(rf)})
			why += " " + rippleMeaning(rf)
			x.concept("Ripple", i)
		}
		x.card(SigCard{Ch: i, Tag: "channel", Title: ch.Name + ": steady DC", What: what, Why: why, Facts: facts, T0: T0, T1: T1})
		if math.Abs(mean) > 0.5 && rip > 0.1*math.Abs(mean) {
			x.warn(SigWarn{Title: ch.Name + ": a lot of ripple", Text: fmt.Sprintf("The ripple is %s, %.0f%% of the %s supply. Most chips want less than about 5%%: a bigger capacitor near the load, or a better regulator, usually fixes it.", f(rip), 100*rip/math.Abs(mean), f(mean)), Ch: i, T0: T0, T1: T1})
		}
		return
	}

	// a signal switching between two levels: digital, PWM, UART…
	if d != nil {
		facts = append(facts, [2]string{"Low level", f(d.vlo)}, [2]string{"High level", f(d.vhi)})
		x.concept("Logic levels", i)
		fam := logicFamily(d.vhi)
		if fam != "" {
			facts = append(facts, [2]string{"Logic family", fam})
		}
		x.explainDigital(i, ch.Name, d, facts)
		x.edgeQuality(i, ch, d)
		if d.vhi > 4.5 {
			x.warn(SigWarn{Title: ch.Name + ": 5 V logic", Text: "This signal goes up to " + f(d.vhi) + ". That's fine for an Arduino Uno, but too much for 3.3 V boards such as the ESP32, Raspberry Pi Pico or a Raspberry Pi: use a level shifter or a resistor divider before connecting it to one of those.", Ch: i, T0: T0, T1: T1})
		} else if d.vhi > 1.2 && d.vhi < 2.6 {
			x.warn(SigWarn{Title: ch.Name + ": a weak HIGH", Text: "The high level is only " + f(d.vhi) + ". A 5 V chip may not see that as HIGH (many need at least 3 V): check the supply, or use a level shifter.", Ch: i, T0: T0, T1: T1})
		}
		return
	}

	// a periodic waveform?
	freq, cv, cross := periodOf(ch, (lo+hi)/2, 0.08*vpp)
	if freq > 0 && cv < 0.12 {
		x.concept("Frequency", i)
		ac := std
		ratio := ac / vpp
		shape, shapeWhy := "sine wave", "A smooth, rounded wave: the shape of mains electricity, audio tones and radio signals, and what you get after filtering."
		x.concept("Sine wave", i)
		rise := riseFraction(ch, cross)
		switch {
		case rise > 0.8 || rise < 0.2:
			shape, shapeWhy = "sawtooth (ramp)", "It climbs steadily, then drops quickly (or the reverse): a ramp, like a timer charging a capacitor, a sweep or a simple synthesizer note."
		case math.Abs(ratio-0.289) < math.Abs(ratio-0.354):
			shape, shapeWhy = "triangle wave", "It rises and falls in straight lines at the same speed: often a capacitor charged and discharged by a constant current, or a function generator."
		}
		clipped := clipFraction(ch, lo, hi, vpp)
		what := fmt.Sprintf("**%s** is a **%s** at **%s**, swinging %s peak to peak around %s.", ch.Name, shape, fmtHz(freq), f(vpp), f(mean))
		why := shapeWhy + " " + freqMeaning(freq, shape)
		facts = append(facts, [2]string{"Frequency", fmtHz(freq)}, [2]string{"Period", fmtTime(1 / freq)})
		if strings.HasPrefix(shape, "sine") {
			facts = append(facts, [2]string{"RMS of the swing", f(ac)})
		}
		if note := musicNote(freq); note != "" && freq >= 20 && freq <= 20000 {
			facts = append(facts, [2]string{"As a musical note", note})
		}
		spc := 1 / (freq * dt)
		facts = append(facts, [2]string{"Samples per cycle", fmt.Sprintf("%.0f", spc)})
		x.card(SigCard{Ch: i, Tag: "channel", Title: fmt.Sprintf("%s: %s, %s", ch.Name, shape, fmtHz(freq)), What: what, Why: why, Facts: facts, T0: T0, T1: T1})
		if spc < 8 {
			x.concept("Aliasing", i)
			x.warn(SigWarn{Title: ch.Name + ": too few samples per cycle", Text: fmt.Sprintf("The scope took only %.1f samples per cycle, so the shape (and maybe even the frequency) can't be trusted. Use a faster timebase so there are at least 10 samples per cycle.", spc), Ch: i, T0: T0, T1: T1})
		}
		if clipped > 0.04 {
			x.warn(SigWarn{Title: ch.Name + ": the tops are flat (clipped)", Text: fmt.Sprintf("%.0f%% of the samples sit exactly at the highest or lowest value. Either the signal went off the edge of the scope's screen (turn the volts/div up), or the circuit itself is clipping: an amplifier running out of supply voltage, for example.", clipped*100), Ch: i, T0: T0, T1: T1})
		}
		return
	}
	// changing, but not regular
	slope := (avgOf(ch.V[n*9/10:]) - avgOf(ch.V[:n/10+1]))
	what := fmt.Sprintf("**%s** changes between %s and %s without repeating regularly.", ch.Name, f(lo), f(hi))
	why := "It isn't a steady level or a repeating wave."
	switch {
	case math.Abs(slope) > 0.6*vpp:
		dir := "rises"
		if slope < 0 {
			dir = "falls"
		}
		what = fmt.Sprintf("**%s** %s from about %s to %s over %s.", ch.Name, dir, f(avgOf(ch.V[:n/10+1])), f(avgOf(ch.V[n*9/10:])), fmtTime(T1-T0))
		why = "A one-off change, like a supply switching on or off, a capacitor charging or discharging, or a button being pressed. Set the scope to trigger on it (single shot) to catch it again."
		if tc := timeConstant(ch); tc > 0 {
			facts = append(facts, [2]string{"Time constant (63%)", fmtTime(tc)})
			why += fmt.Sprintf(" The curve reaches 63%% of its change in about %s: that's the RC time constant if a resistor is charging a capacitor (R × C = %s).", fmtTime(tc), fmtTime(tc))
			x.concept("Time constant", i)
		}
	case std > 0.25*vpp:
		why = "It looks like noise or a busy signal (audio, speech, a sensor that keeps changing). A slower timebase, or averaging, can show if there's a pattern underneath."
		x.concept("Noise", i)
	}
	x.card(SigCard{Ch: i, Tag: "channel", Title: ch.Name + ": changing signal", What: what, Why: why, Facts: facts, T0: T0, T1: T1})
}

func avgOf(a []float64) float64 { m, _ := meanStd(a); return m }

func logicFamily(vhi float64) string {
	switch {
	case vhi > 4.4 && vhi < 5.6:
		return "5 V logic (classic Arduino, 74HC at 5 V)"
	case vhi > 2.9 && vhi < 3.7:
		return "3.3 V logic (ESP32, Pico, STM32, most modern sensors)"
	case vhi > 1.6 && vhi < 2.0:
		return "1.8 V logic (some phone-era chips and sensors)"
	case vhi > 2.3 && vhi < 2.8:
		return "2.5 V logic, or a weak 3.3 V"
	}
	return ""
}

func dcMeaning(v float64, unit string) string {
	if unit != "V" {
		return "It doesn't change during the capture."
	}
	a := math.Abs(v)
	switch {
	case a < 0.05:
		return "That's 0 V: ground, a switched-off line, or a probe that isn't touching anything."
	case a > 1.7 && a < 1.9:
		return "That's very close to 1.8 V, a common supply for low-voltage chips."
	case a > 3.1 && a < 3.5:
		return "That's a 3.3 V supply rail: what ESP32s, Picos, STM32s and most sensors run on."
	case a > 4.6 && a < 5.4:
		return "That's a 5 V rail: USB power, or what a classic Arduino runs on."
	case a > 11.4 && a < 12.6:
		return "That's a 12 V supply: LED strips, fans and motors often use it."
	case a > 3.5 && a < 4.25:
		return "That's in the range of a single lithium-ion cell (about 3.7 V on average, 4.2 V full)."
	case a > 1.1 && a < 1.6:
		return "That's in the range of one AA or AAA battery (1.2–1.5 V)."
	case a > 8.5 && a < 9.6:
		return "That's close to 9 V, like a PP3 battery."
	}
	return "It stays put during the whole capture."
}

func rippleMeaning(f float64) string {
	switch {
	case f > 45 && f < 65:
		return "Ripple at mains frequency: hum from the mains supply, often from a ground loop or a poorly filtered transformer."
	case f > 90 && f < 130:
		return "Ripple at twice the mains frequency: what's left over after a bridge rectifier. A bigger smoothing capacitor reduces it."
	case f > 5000:
		return "Ripple this fast usually comes from a switching regulator (buck or boost converter) doing its job; small amounts are normal."
	}
	return ""
}

func freqMeaning(f float64, shape string) string {
	switch {
	case f > 47 && f < 63 && strings.HasPrefix(shape, "sine"):
		return "At " + fmtHz(f) + " it's very likely mains electricity (50 Hz in the UK and Europe, 60 Hz in North America), seen through a transformer or picked up as hum."
	case f >= 20 && f <= 20000:
		return "At " + fmtHz(f) + " it's in the range people can hear: played through a speaker, it would be a tone."
	case f > 20000 && f < 100000:
		return "That's above human hearing (ultrasonic): ultrasonic distance sensors use 40 kHz, for example."
	case f >= 1e6:
		return "That's a radio-frequency signal, like a crystal oscillator or a clock."
	case f < 20:
		return "That's slow: a few cycles a second, like a breathing LED or a slow sensor."
	}
	return ""
}

var noteNames = []string{"C", "C♯", "D", "D♯", "E", "F", "F♯", "G", "G♯", "A", "A♯", "B"}

func musicNote(f float64) string {
	if f <= 0 {
		return ""
	}
	n := 12*math.Log2(f/440) + 69
	k := int(math.Round(n))
	cents := (n - float64(k)) * 100
	if math.Abs(cents) < 1 {
		return fmt.Sprintf("%s%d, in tune", noteNames[((k%12)+12)%12], k/12-1)
	}
	return fmt.Sprintf("%s%d (%+.0f cents)", noteNames[((k%12)+12)%12], k/12-1, cents)
}

// periodOf finds the frequency from crossings of a level with hysteresis.
func periodOf(ch *SigChan, level, hyst float64) (freq, cv float64, rises []int) {
	state := ch.V[0] > level
	for i, v := range ch.V {
		if !state && v > level+hyst {
			state = true
			rises = append(rises, i)
		} else if state && v < level-hyst {
			state = false
		}
	}
	if len(rises) < 3 {
		return 0, 1, rises
	}
	var p []float64
	for k := 1; k < len(rises); k++ {
		p = append(p, ch.T[rises[k]]-ch.T[rises[k-1]])
	}
	m, s := meanStd(p)
	if m <= 0 {
		return 0, 1, rises
	}
	return 1 / median(p), s / m, rises
}

// riseFraction: how much of each cycle the signal spends going up.
func riseFraction(ch *SigChan, rises []int) float64 {
	if len(rises) < 2 {
		return 0.5
	}
	a, b := rises[0], rises[len(rises)-1]
	up, all := 0, 0
	// smooth a little so noise doesn't count
	w := max(1, (b-a)/(len(rises)*40))
	for i := a + w; i <= b; i += w {
		if ch.V[i] > ch.V[i-w] {
			up++
		}
		all++
	}
	if all == 0 {
		return 0.5
	}
	return float64(up) / float64(all)
}

func clipFraction(ch *SigChan, lo, hi, vpp float64) float64 {
	n := 0
	for _, v := range ch.V {
		if v >= hi-0.002*vpp || v <= lo+0.002*vpp {
			n++
		}
	}
	return float64(n) / float64(len(ch.V))
}

func rippleFreq(ch *SigChan) float64 {
	m, s := meanStd(ch.V)
	if s == 0 {
		return 0
	}
	f, cv, _ := periodOf(ch, m, 0.3*s)
	if cv < 0.2 {
		return f
	}
	return 0
}

// timeConstant: time to cover 63% of a one-off change.
func timeConstant(ch *SigChan) float64 {
	n := len(ch.V)
	a, b := avgOf(ch.V[:n/20+1]), avgOf(ch.V[n*19/20:])
	if math.Abs(b-a) < 1e-9 {
		return 0
	}
	start := -1
	for i, v := range ch.V {
		fr := (v - a) / (b - a)
		if start < 0 && fr > 0.05 {
			start = i
		}
		if start >= 0 && fr >= 0.632 {
			return ch.T[i] - ch.T[start]
		}
	}
	return 0
}

// edgeQuality: rise time and overshoot of a scope channel that switches.
func (x *sigCtx) edgeQuality(i int, ch *SigChan, d *digi) {
	lo, hi := d.vlo, d.vhi
	sw := hi - lo
	if sw <= 0 {
		return
	}
	var rts []float64
	var over float64
	for k := 1; k < len(ch.V) && len(rts) < 200; k++ {
		if ch.V[k-1] < lo+0.5*sw && ch.V[k] >= lo+0.5*sw {
			a, b := k, k
			for a > 0 && ch.V[a] > lo+0.1*sw {
				a--
			}
			for b < len(ch.V)-1 && ch.V[b] < lo+0.9*sw {
				b++
			}
			rts = append(rts, ch.T[b]-ch.T[a])
			for c := b; c < len(ch.V) && c < b+40; c++ {
				over = math.Max(over, (ch.V[c]-hi)/sw)
			}
		}
	}
	if len(rts) == 0 {
		return
	}
	rt := median(rts)
	last := &x.r.Cards[len(x.r.Cards)-1]
	if rt <= 2.5*d.dt {
		last.Facts = append(last.Facts, [2]string{"Rise time", "faster than this capture can show (under " + fmtTime(2.5*d.dt) + ")"})
	} else {
		last.Facts = append(last.Facts, [2]string{"Rise time (10–90%)", fmtTime(rt)})
		x.concept("Rise time", i)
		if d.period > 0 && rt > 0.15*d.period {
			x.warn(SigWarn{Title: ch.Name + ": slow edges", Text: fmt.Sprintf("Each edge takes %s, a big part of the %s cycle. A long wire, too much capacitance or a weak driver (or a large pull-up resistor) rounds the edges off. Shorter wires or a stronger pull-up help.", fmtTime(rt), fmtTime(d.period)), Ch: i})
		}
	}
	if over > 0.2 {
		last.Facts = append(last.Facts, [2]string{"Overshoot", pct(over)})
		x.concept("Overshoot and ringing", i)
		x.warn(SigWarn{Title: ch.Name + ": overshoot and ringing", Text: fmt.Sprintf("The edges shoot %.0f%% past the high level before settling. Some of that is often the scope probe's ground lead: use the short ground spring. If it's really there, a small series resistor (22–100 Ω) at the driver calms it.", over*100), Ch: i})
	}
}

// ---------------------------------------------------------------- meters (slow readings)

func (x *sigCtx) explainMeter(i int, ch *SigChan) {
	var t, v []float64
	ol := 0
	for k := range ch.V {
		if math.IsNaN(ch.V[k]) {
			ol++
			continue
		}
		t, v = append(t, ch.T[k]), append(v, ch.V[k])
	}
	if len(v) < 2 {
		return
	}
	n := len(v)
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, y := range v {
		lo, hi = math.Min(lo, y), math.Max(hi, y)
	}
	mean, std := meanStd(v)
	dur := t[n-1] - t[0]
	unit := ch.Unit
	f := func(q float64) string { return siFmt(q, unit) }
	// straight-line trend
	tm, _ := meanStd(t)
	var sxy, sxx float64
	for k := range t {
		sxy += (t[k] - tm) * (v[k] - mean)
		sxx += (t[k] - tm) * (t[k] - tm)
	}
	slope := 0.0
	if sxx > 0 {
		slope = sxy / sxx
	}
	start, end := avgOf(v[:max(1, n/20)]), avgOf(v[n-max(1, n/20):])
	facts := [][2]string{{"Readings", fmt.Sprintf("%d over %s", n, humanDur(dur))}, {"Start → end", f(start) + " → " + f(end)}, {"Lowest / highest", f(lo) + " / " + f(hi)}, {"Average", f(mean)}}
	what := fmt.Sprintf("**%s** was logged %d times over %s, averaging **%s**.", ch.Name, n, humanDur(dur), f(mean))
	why := ""
	trend := slope * dur
	if math.Abs(trend) > math.Max(3*std/math.Sqrt(float64(n))*4, 0.02*math.Abs(mean)) {
		dir := "rose"
		if trend < 0 {
			dir = "fell"
		}
		what += fmt.Sprintf(" It %s steadily, by about %s %s.", dir, f(math.Abs(slope)*perUnit(dur)), perWord(dur))
	} else {
		what += " It stayed about the same."
	}
	switch unit {
	case "V":
		why = batteryMeaning(start, end, lo, hi)
		x.concept("Voltage", i)
	case "A":
		x.concept("Current draw", i)
		why = currentMeaning(v, mean, hi)
		if mean > 0 {
			h := 2.0 / mean // a 2000 mAh battery
			facts = append(facts, [2]string{"On a 2000 mAh battery", "about " + humanDur(h*3600)})
		}
	case "°C":
		if math.Abs(trend) > 1 {
			why = "A temperature that changes like this is something warming up or cooling down. Where it levels off is its working temperature."
			if trend > 0 && hi > 70 {
				why += " Above about 70 °C most parts are getting hot: check there's enough cooling."
			}
		} else {
			why = "The temperature held steady."
		}
	case "Ω":
		why = "Resistance readings. A changing resistance is usually a sensor (a thermistor, LDR or strain gauge) or a poor contact."
	case "Hz":
		why = "Frequency readings, measured by the meter itself."
	default:
		why = "Readings logged by a meter or a sensor."
	}
	// sudden jumps
	var jumps []int
	diffs := make([]float64, 0, n)
	for k := 1; k < n; k++ {
		diffs = append(diffs, math.Abs(v[k]-v[k-1]))
	}
	md := median(diffs)
	for k := 1; k < n; k++ {
		if d := math.Abs(v[k] - v[k-1]); d > 8*md && d > 0.1*(hi-lo) && d > 0 && len(jumps) < 20 {
			jumps = append(jumps, k)
		}
	}
	if len(jumps) > 0 {
		var w []string
		for _, k := range jumps[:min(5, len(jumps))] {
			w = append(w, fmt.Sprintf("%s at %s", f(v[k]-v[k-1]), humanDur(t[k])))
		}
		facts = append(facts, [2]string{"Sudden changes", strings.Join(w, ", ")})
		why += " The sudden changes mark moments when something switched on or off."
	}
	if ol > 0 {
		facts = append(facts, [2]string{"Over range (OL)", fmt.Sprintf("%d readings", ol)})
		x.warn(SigWarn{Title: ch.Name + ": over range", Text: fmt.Sprintf("%d readings were OL: the value was too big for the meter's range (or, for resistance, nothing was connected). Pick a higher range, or use auto-range.", ol), Ch: i})
	}
	x.card(SigCard{Ch: i, Tag: "channel", Title: ch.Name + " (" + unitWord(unit) + ")", What: what, Why: strings.TrimSpace(why), Facts: facts, T0: t[0], T1: t[n-1]})
}

func unitWord(u string) string {
	switch u {
	case "V":
		return "voltage"
	case "A":
		return "current"
	case "°C":
		return "temperature"
	case "Ω":
		return "resistance"
	case "Hz":
		return "frequency"
	case "F":
		return "capacitance"
	case "W":
		return "power"
	case "%":
		return "percent"
	case "":
		return "values"
	}
	return u
}

func perUnit(dur float64) float64 {
	switch {
	case dur > 7200:
		return 3600
	case dur > 120:
		return 60
	}
	return 1
}
func perWord(dur float64) string {
	switch {
	case dur > 7200:
		return "per hour"
	case dur > 120:
		return "per minute"
	}
	return "per second"
}

func humanDur(s float64) string {
	switch {
	case s < 1:
		return fmtTime(s)
	case s < 120:
		return fmt.Sprintf("%.1f s", s)
	case s < 7200:
		return fmt.Sprintf("%.1f minutes", s/60)
	case s < 3*86400:
		return fmt.Sprintf("%.1f hours", s/3600)
	}
	return fmt.Sprintf("%.1f days", s/86400)
}

func batteryMeaning(start, end, lo, hi float64) string {
	falling := end < start-0.01
	rising := end > start+0.01
	switch {
	case hi <= 4.3 && lo >= 2.8 && (falling || rising):
		s := "This looks like a single lithium-ion (Li-ion or LiPo) cell"
		if falling {
			s += fmt.Sprintf(" discharging: it went from about %s to %s (very roughly %.0f%% to %.0f%% charge).", fmtV(start), fmtV(end), liionPct(start), liionPct(end))
			if end < 3.3 {
				s += " It's nearly empty: below about 3.0 V a Li-ion cell can be damaged."
			}
		} else {
			s += fmt.Sprintf(" charging, from about %.0f%% to %.0f%%.", liionPct(start), liionPct(end))
		}
		return s
	case hi < 1.7 && lo > 0.8 && falling:
		return "This looks like an AA or AAA battery running down. Alkaline cells start near 1.5 V; below about 1.0 V they're flat."
	case hi < 14.8 && lo > 11 && (falling || rising):
		if hi > 13.4 {
			return "That's a 12 V lead-acid battery being charged (13.5–14.7 V is the charging range)."
		}
		return "That's a 12 V lead-acid battery: about 12.7 V is full, 12.0 V is around half, 11.8 V is nearly flat."
	case hi < 5.6 && lo > 4.4:
		return "That's a 5 V USB supply. Dips under load show the cable or supply struggling: 4.75 V is the lowest USB allows."
	}
	return "Voltage readings over time."
}

func liionPct(v float64) float64 {
	pts := [][2]float64{{3.0, 0}, {3.3, 5}, {3.6, 15}, {3.7, 35}, {3.8, 55}, {3.9, 70}, {4.0, 82}, {4.1, 92}, {4.2, 100}}
	if v <= pts[0][0] {
		return 0
	}
	for k := 1; k < len(pts); k++ {
		if v <= pts[k][0] {
			a, b := pts[k-1], pts[k]
			return a[1] + (v-a[0])/(b[0]-a[0])*(b[1]-a[1])
		}
	}
	return 100
}

func currentMeaning(v []float64, mean, hi float64) string {
	// sleep / active?
	lo := math.Inf(1)
	for _, y := range v {
		lo = math.Min(lo, y)
	}
	if hi > 5*math.Max(lo, 1e-6) && hi > 0.005 {
		th := (hi + lo) / 2
		on := 0
		for _, y := range v {
			if y > th {
				on++
			}
		}
		fr := float64(on) / float64(len(v))
		return fmt.Sprintf("The current jumps between about %s and %s, spending %.0f%% of the time at the higher current. That's typical of a device that sleeps and wakes up to do something (read a sensor, send over Wi-Fi). The average is what drains a battery.", siFmt(lo, "A"), siFmt(hi, "A"), fr*100)
	}
	return fmt.Sprintf("A fairly steady current of about %s.", siFmt(mean, "A"))
}

// explainPower: a voltage and a current logged together give power and energy.
func (x *sigCtx) explainPower() {
	var vi, ai = -1, -1
	for i, ch := range x.c.Chans {
		if ch.Kind != KindMeter {
			continue
		}
		if ch.Unit == "V" && vi < 0 {
			vi = i
		}
		if ch.Unit == "A" && ai < 0 {
			ai = i
		}
	}
	if vi < 0 || ai < 0 {
		return
	}
	V, A := x.c.Chans[vi], x.c.Chans[ai]
	var p []float64
	var e float64
	for k := 1; k < len(A.T); k++ {
		vv := valueAt(V, A.T[k])
		if math.IsNaN(vv) || math.IsNaN(A.V[k]) {
			continue
		}
		pw := vv * A.V[k]
		p = append(p, pw)
		e += pw * (A.T[k] - A.T[k-1])
	}
	if len(p) < 2 {
		return
	}
	pm, _ := meanStd(p)
	x.concept("Power and energy", -1)
	x.card(SigCard{Ch: -1, Tag: "power", Title: "Power and energy", What: fmt.Sprintf("With **%s** and **%s** logged together, Probie can work out the power: about **%s** on average, **%s** of energy over the whole log.", V.Name, A.Name, siFmt(pm, "W"), siFmt(e/3600, "Wh")),
		Why: "Power is voltage × current. Energy is power over time: it's what a battery stores (in Wh, or mAh at its voltage).", Facts: [][2]string{{"Average power", siFmt(pm, "W")}, {"Energy used", siFmt(e/3600, "Wh") + " (" + siFmt(e, "J") + ")"}}, T0: A.T[0], T1: A.T[len(A.T)-1]})
}

func valueAt(ch *SigChan, t float64) float64 {
	k := sort.SearchFloat64s(ch.T, t)
	if k >= len(ch.T) {
		k = len(ch.T) - 1
	}
	if k > 0 && math.Abs(ch.T[k-1]-t) < math.Abs(ch.T[k]-t) {
		k--
	}
	if k < 0 {
		return math.NaN()
	}
	return ch.V[k]
}

// ---------------------------------------------------------------- overview

func (x *sigCtx) overview() {
	c, r := x.c, x.r
	var s []string
	what := c.Source
	if what == "" {
		what = "capture"
	}
	parts := []string{}
	nd, na, nm := 0, 0, 0
	for _, ch := range c.Chans {
		switch ch.Kind {
		case KindDigital:
			nd++
		case KindAnalog:
			na++
		case KindMeter:
			nm++
		}
	}
	if na > 0 {
		parts = append(parts, sigPlural(na, "scope channel", "scope channels"))
	}
	if nd > 0 {
		parts = append(parts, sigPlural(nd, "logic channel", "logic channels"))
	}
	if nm > 0 {
		parts = append(parts, sigPlural(nm, "logged reading", "logged readings"))
	}
	if len(c.Serial) > 0 {
		parts = append(parts, sigPlural(len(c.Serial), "line of serial text", "lines of serial text"))
	}
	s = append(s, fmt.Sprintf("This is a **%s** (%s): %s over %s.", what, c.File, strings.Join(parts, ", "), humanDur(r.Duration)))
	// headline findings
	var heads []string
	for _, cd := range r.Cards {
		if cd.Tag == "protocol" || cd.Tag == "channel" || cd.Tag == "serial" {
			h := strings.TrimSuffix(firstSentence(cd.What), ".")
			if i := strings.Index(h, ", which read as text"); i > 0 {
				h = h[:i] // the text itself is on the card
			}
			heads = append(heads, trunc(h, 160))
		}
		if len(heads) >= 4 {
			break
		}
	}
	if len(heads) > 0 {
		s = append(s, "In short: "+strings.Join(heads, "; ")+".")
	}
	if len(r.Warnings) > 0 {
		s = append(s, fmt.Sprintf("There %s worth checking: see **Warnings**.", plural2(len(r.Warnings), "is 1 thing", "are %d things")))
	}
	s = append(s, "Click a card to see where it happens on the capture; click the capture to find the card that explains it.")
	r.Summary = s
	r.Cards = append([]SigCard{{Ch: -1, Tag: "overview", Title: "The capture", What: s[0], Why: "Probie reads captures but never changes them. It explains what each channel is doing, decodes common protocols, and points out things worth checking.", T0: x.t0, T1: x.t1}}, r.Cards...)
}

func firstSentence(s string) string {
	if i := strings.Index(s, ". "); i > 0 {
		return s[:i+1]
	}
	return s
}

func sigPlural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, many)
}

func plural2(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return fmt.Sprintf(many, n)
}
