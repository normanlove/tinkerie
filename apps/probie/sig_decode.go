// Probie — Signals: logic channels and the protocols they carry (UART, I2C,
// SPI, PWM and servo, WS2812 LEDs, NEC infrared remotes, 1-Wire).
// Part of the Tinkerie maker suite. Created by nrmluvdev.

package probie

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"unicode"
)

// digi is a logic channel: the level at the start, then every change.
type digi struct {
	t, v         []float64
	t0, t1       float64
	hiW, loW     []float64 // complete pulse widths
	vlo, vhi, dt float64   // scope channels: their voltages and sample time
	analog       bool
	period       float64
	idle         int
}

func newDigi(t, v []float64, t0, t1 float64) *digi {
	d := &digi{t0: t0, t1: t1}
	for i := range t {
		lv := 0.0
		if v[i] >= 0.5 {
			lv = 1
		}
		if len(d.v) == 0 || d.v[len(d.v)-1] != lv {
			d.t, d.v = append(d.t, t[i]), append(d.v, lv)
		}
	}
	for k := 1; k+1 < len(d.t); k++ {
		w := d.t[k+1] - d.t[k]
		if d.v[k] == 1 {
			d.hiW = append(d.hiW, w)
		} else {
			d.loW = append(d.loW, w)
		}
	}
	// idle level: where the line spends most of its time
	hi := 0.0
	for k := range d.t {
		end := t1
		if k+1 < len(d.t) {
			end = d.t[k+1]
		}
		if d.v[k] == 1 {
			hi += end - d.t[k]
		}
	}
	if t1 > t0 && hi > (t1-t0)/2 {
		d.idle = 1
	}
	return d
}

func (d *digi) edges() int { return len(d.t) - 1 }

// at is the level at time t.
func (d *digi) at(t float64) int {
	k := sort.SearchFloat64s(d.t, t+1e-15) - 1
	if k < 0 {
		k = 0
	}
	return int(d.v[k])
}

func (d *digi) falls() []float64 {
	var out []float64
	for k := 1; k < len(d.t); k++ {
		if d.v[k] == 0 {
			out = append(out, d.t[k])
		}
	}
	return out
}
func (d *digi) rises() []float64 {
	var out []float64
	for k := 1; k < len(d.t); k++ {
		if d.v[k] == 1 {
			out = append(out, d.t[k])
		}
	}
	return out
}

func pctile(a []float64, p float64) float64 {
	if len(a) == 0 {
		return 0
	}
	b := append([]float64(nil), a...)
	sort.Float64s(b)
	return b[int(math.Min(float64(len(b)-1), p*float64(len(b))))]
}

func hexBytes(b []byte, maxN int) string {
	var s []string
	for i, x := range b {
		if i >= maxN {
			s = append(s, "…")
			break
		}
		s = append(s, fmt.Sprintf("%02X", x))
	}
	return strings.Join(s, " ")
}

func printable(b []byte) (string, bool) {
	ok := 0
	var sb strings.Builder
	for _, x := range b {
		switch {
		case x == '\n':
			sb.WriteString("⏎")
			ok++
		case x == '\r':
			ok++
		case x == '\t':
			sb.WriteByte(' ')
			ok++
		case x >= 32 && x < 127:
			sb.WriteByte(x)
			ok++
		default:
			sb.WriteString("·")
		}
	}
	return sb.String(), len(b) > 0 && float64(ok)/float64(len(b)) > 0.85
}

// ---------------------------------------------------------------- one logic channel

func (x *sigCtx) explainDigital(i int, name string, d *digi, facts [][2]string) {
	if d == nil {
		return
	}
	lvl := map[int]string{0: "LOW", 1: "HIGH"}
	T0, T1 := x.t0, x.t1
	if d.edges() == 0 {
		x.card(SigCard{Ch: i, Tag: "channel", Title: name + ": no activity", What: fmt.Sprintf("**%s** stays **%s** the whole time.", name, lvl[int(d.v[0])]),
			Why: "Nothing happened on this wire during the capture. It may be an idle line (a UART or I2C bus waiting), a signal that only changes now and then, or a probe that isn't connected. If you expected activity, check the wiring and the trigger.", Facts: facts, T0: T0, T1: T1})
		return
	}
	x.concept("Logic levels", i)
	if x.periodic(i, name, d, facts) {
		return
	}
	if x.ws2812(i, name, d, facts) || x.nec(i, name, d, facts) || x.onewire(i, name, d, facts) || x.uart(i, name, d, facts) {
		return
	}
	all := append(append([]float64(nil), d.hiW...), d.loW...)
	f := append(facts, [2]string{"Changes", fmt.Sprintf("%d", d.edges())})
	if len(all) > 0 {
		f = append(f, [2]string{"Shortest pulse", fmtTime(pctile(all, 0))}, [2]string{"Longest pulse", fmtTime(pctile(all, 1))})
	}
	what := fmt.Sprintf("**%s** changes %d times, resting **%s** most of the time, with no pattern Probie recognises.", name, d.edges(), lvl[d.idle])
	why := "It could be a button, an interrupt or enable line, a chip select, or a protocol Probie doesn't know yet. Try naming the channel in Probie (TX, SDA, SCK…): Probie uses the names."
	if d.edges() <= 4 {
		what = fmt.Sprintf("**%s** changes only %d time%s: ", name, d.edges(), map[bool]string{true: "", false: "s"}[d.edges() == 1])
		var w []string
		for k := 1; k < len(d.t) && k < 5; k++ {
			w = append(w, fmt.Sprintf("%s at %s", map[float64]string{0: "falls LOW", 1: "rises HIGH"}[d.v[k]], fmtTime(d.t[k])))
		}
		what += strings.Join(w, ", then ") + "."
		why = "A line that changes once or twice is usually a control signal: power or enable switching on, a reset, a button press, or an interrupt saying 'something happened'."
	}
	x.card(SigCard{Ch: i, Tag: "channel", Title: name + ": control or data line", What: what, Why: why, Facts: f, T0: d.t[min(1, len(d.t)-1)], T1: d.t[len(d.t)-1]})
}

// periodic: a clock, PWM or a servo signal.
func (x *sigCtx) periodic(i int, name string, d *digi, facts [][2]string) bool {
	r := d.rises()
	if len(r) < 4 {
		return false
	}
	var p []float64
	for k := 1; k < len(r); k++ {
		p = append(p, r[k]-r[k-1])
	}
	m, s := meanStd(p)
	if m <= 0 || s/m > 0.06 {
		return false
	}
	// the pulses must fill the capture (not one burst)
	if (r[len(r)-1]-r[0])/(x.t1-x.t0) < 0.5 {
		return false
	}
	per := median(p)
	d.period = per
	hw := median(d.hiW)
	duty := hw / per
	freq := 1 / per
	f := append(facts, [2]string{"Frequency", fmtHz(freq)}, [2]string{"Period", fmtTime(per)}, [2]string{"High for", fmtTime(hw)}, [2]string{"Duty cycle", pct(duty)})
	x.concept("Frequency", i)
	x.concept("Duty cycle", i)
	T0, T1 := r[0], r[len(r)-1]
	switch {
	case freq > 40 && freq < 400 && hw > 0.4e-3 && hw < 2.7e-3:
		ang := (hw - 1e-3) / 1e-3 * 180
		ang = math.Max(0, math.Min(180, ang))
		x.concept("Servo signal", i)
		x.card(SigCard{Ch: i, Tag: "channel", Title: fmt.Sprintf("%s: servo signal, %s pulses", name, fmtTime(hw)), What: fmt.Sprintf("**%s** is a **servo control signal**: a %s pulse every %s (%s).", name, fmtTime(hw), fmtTime(per), fmtHz(freq)),
			Why: fmt.Sprintf("Hobby servos read the width of the pulse, not the duty cycle: about 1 ms is one end, 1.5 ms the middle and 2 ms the other end. %s puts a typical servo at about **%.0f°** of 180°. ESCs for drone and RC motors use the same signal for speed.", fmtTime(hw), ang), Facts: f, T0: T0, T1: T1})
	case duty > 0.44 && duty < 0.56:
		x.concept("Clock", i)
		why := "A steady on-off signal with equal high and low times is a clock: it paces another chip, or it's a square wave from a timer or tone() for a buzzer."
		if freq >= 20 && freq <= 20000 {
			why += " At " + fmtHz(freq) + " it could also be a tone for a buzzer (" + musicNote(freq) + ")."
		}
		if math.Abs(freq-32768) < 50 {
			why = "32.768 kHz is the classic watch-crystal frequency, used by real-time clocks."
		}
		x.card(SigCard{Ch: i, Tag: "channel", Title: fmt.Sprintf("%s: clock or square wave, %s", name, fmtHz(freq)), What: fmt.Sprintf("**%s** is a **square wave / clock** at **%s**, high half the time.", name, fmtHz(freq)), Why: why, Facts: f, T0: T0, T1: T1})
	default:
		x.concept("PWM", i)
		if len(d.hiW) > 8 {
			lo, hi := pctile(d.hiW, 0.05)/per, pctile(d.hiW, 0.95)/per
			if hi-lo > 0.1 {
				f = append(f, [2]string{"Duty changes", pct(lo) + " to " + pct(hi)})
				defer func() {
					c := &x.r.Cards[len(x.r.Cards)-1]
					c.What += fmt.Sprintf(" The duty cycle changes during the capture, from about %s to %s: something is fading an LED or ramping a motor.", pct(lo), pct(hi))
				}()
			}
		}
		use := "dimming an LED, setting a motor's speed or a fan's"
		if freq > 20000 {
			use = "driving a motor or fan quietly (above hearing), or a switching regulator"
		} else if freq < 2000 {
			use = "dimming an LED (Arduino's analogWrite uses about 490 Hz or 980 Hz), or a motor's speed"
		}
		x.card(SigCard{Ch: i, Tag: "channel", Title: fmt.Sprintf("%s: PWM at %s, %s duty", name, fmtHz(freq), pct(duty)), What: fmt.Sprintf("**%s** is **PWM** (pulse-width modulation) at **%s**, high **%s** of the time.", name, fmtHz(freq), pct(duty)),
			Why: fmt.Sprintf("PWM switches fully on and off quickly; what matters is the fraction of time it's on. %s on means roughly %s power: %s. The average voltage is about %s of the high level.", pct(duty), pct(duty), use, pct(duty)), Facts: f, T0: T0, T1: T1})
	}
	return true
}

// ---------------------------------------------------------------- UART

var bauds = []float64{300, 1200, 2400, 4800, 9600, 14400, 19200, 31250, 38400, 57600, 74880, 115200, 230400, 250000, 460800, 500000, 921600, 1000000, 1500000, 2000000}

func (x *sigCtx) uart(i int, name string, d *digi, facts [][2]string) bool {
	if d.idle != 1 || d.edges() < 6 {
		return false
	}
	all := append(append([]float64(nil), d.hiW...), d.loW...)
	minW := pctile(all, 0.04)
	if minW <= 0 {
		return false
	}
	best, bestErr := 0.0, 1.0
	for _, b := range bauds {
		bit := 1 / b
		r := minW / bit
		if r < 0.75 || r > 1.3 {
			continue
		}
		// pulse widths should be whole numbers of bits
		e, n := 0.0, 0
		for _, w := range all {
			q := w / bit
			if q > 10.5 {
				continue
			}
			e += math.Abs(q - math.Round(q))
			n++
		}
		if n == 0 {
			continue
		}
		if e /= float64(n); e < bestErr {
			best, bestErr = b, e
		}
	}
	if best == 0 || bestErr > 0.2 {
		return false
	}
	bit := 1 / best
	var bytes []byte
	var frames []SigFrame
	errs := 0
	next := -1.0
	for _, f := range d.falls() {
		if f < next {
			continue
		}
		var b byte
		for k := 0; k < 8; k++ {
			if d.at(f+(1.5+float64(k))*bit) == 1 {
				b |= 1 << k
			}
		}
		stopOK := d.at(f+9.5*bit) == 1
		kind := "data"
		note := ""
		if !stopOK {
			errs++
			kind, note = "error", "no stop bit (framing error)"
		}
		bytes = append(bytes, b)
		frames = append(frames, SigFrame{T0: f, T1: f + 10*bit, Ch: i, Proto: "UART", Text: uartText(b), Note: note, Kind: kind})
		next = f + 9.6*bit
	}
	if len(bytes) == 0 || float64(errs) > 0.25*float64(len(bytes)) {
		return false
	}
	x.r.Decoded = append(x.r.Decoded, frames...)
	x.concept("UART serial", i)
	x.concept("Baud rate", i)
	txt, isText := printable(bytes)
	what := fmt.Sprintf("**%s** carries **serial data (UART)** at **%.0f baud**: %d byte%s", name, best, len(bytes), map[bool]string{true: "", false: "s"}[len(bytes) == 1])
	if isText {
		show := txt
		if len([]rune(show)) > 160 {
			show = string([]rune(show)[:160]) + "…"
		}
		what += fmt.Sprintf(", which read as text: “%s”", show)
	} else {
		what += ": " + hexBytes(bytes, 24)
	}
	what += "."
	why := "UART is the simplest serial link: one wire per direction (TX → RX), no clock. The line rests HIGH; each byte starts with a LOW start bit, then 8 data bits (lowest first) and a HIGH stop bit. Both ends must agree on the speed (the baud rate)."
	if isText {
		why += " Text like this is usually a board printing messages: Serial.print() on an Arduino, print() in MicroPython, or a module such as a GPS."
	}
	if math.Abs(best-74880) < 1 {
		why += " 74880 baud is what an ESP8266 uses for its boot messages."
	}
	f := append(facts, [2]string{"Baud rate", fmt.Sprintf("%.0f (one bit = %s)", best, fmtTime(bit))}, [2]string{"Bytes", fmt.Sprintf("%d", len(bytes))}, [2]string{"Format", "8 data bits, no parity, 1 stop bit (8N1)"})
	if errs > 0 {
		f = append(f, [2]string{"Framing errors", fmt.Sprintf("%d", errs)})
		x.warn(SigWarn{Title: name + ": framing errors", Text: fmt.Sprintf("%d byte(s) had no stop bit where one should be. The baud rate may be slightly off (check both ends use the same speed), the line may be noisy, or the format may not be 8N1.", errs), Ch: i, T0: frames[0].T0, T1: frames[len(frames)-1].T1})
	}
	x.card(SigCard{Ch: i, Tag: "protocol", Title: fmt.Sprintf("%s: UART serial, %.0f baud", name, best), What: what, Why: why, Facts: f, T0: frames[0].T0, T1: frames[len(frames)-1].T1})
	return true
}

func uartText(b byte) string {
	switch {
	case b == '\n':
		return "0x0A '\\n' (new line)"
	case b == '\r':
		return "0x0D '\\r' (carriage return)"
	case b >= 32 && b < 127:
		return fmt.Sprintf("0x%02X '%c'", b, b)
	}
	return fmt.Sprintf("0x%02X", b)
}

// ---------------------------------------------------------------- WS2812 / NeoPixel

func (x *sigCtx) ws2812(i int, name string, d *digi, facts [][2]string) bool {
	if len(d.hiW) < 24 {
		return false
	}
	for _, w := range d.hiW {
		if w < 0.12e-6 || w > 1.1e-6 {
			return false
		}
	}
	short := 0
	for _, w := range d.loW {
		if w < 1.2e-6 {
			short++
		}
	}
	if float64(short) < 0.8*float64(len(d.loW)) {
		return false
	}
	// decode: a bit is 1 when the high pulse is long
	var frames [][]byte
	var cur []byte
	var acc byte
	nb := 0
	var starts []float64
	for k := 1; k+1 < len(d.t); k++ {
		if d.v[k] != 1 {
			if d.t[k+1]-d.t[k] > 40e-6 && (len(cur) > 0 || nb > 0) { // reset
				frames = append(frames, cur)
				cur, acc, nb = nil, 0, 0
			}
			continue
		}
		if len(cur) == 0 && nb == 0 {
			starts = append(starts, d.t[k])
		}
		acc <<= 1
		if d.t[k+1]-d.t[k] > 0.55e-6 {
			acc |= 1
		}
		if nb++; nb == 8 {
			cur, acc, nb = append(cur, acc), 0, 0
		}
	}
	if len(cur) > 0 {
		frames = append(frames, cur)
	}
	if len(frames) == 0 {
		return false
	}
	x.concept("WS2812 addressable LEDs", i)
	last := frames[len(frames)-1]
	n := len(last) / 3
	var cols []string
	for k := 0; k+2 < len(last) && k/3 < 8; k += 3 {
		g, r, b := last[k], last[k+1], last[k+2]
		cols = append(cols, fmt.Sprintf("LED %d: %s", k/3+1, colourName(r, g, b)))
	}
	for fi, fr := range frames {
		t0 := x.t0
		if fi < len(starts) {
			t0 = starts[fi]
		}
		for k := 0; k+2 < len(fr) && k/3 < 300; k += 3 {
			x.r.Decoded = append(x.r.Decoded, SigFrame{T0: t0 + float64(k*8)*1.25e-6, T1: t0 + float64(k*8+24)*1.25e-6, Ch: i, Proto: "WS2812", Kind: "data", Text: fmt.Sprintf("LED %d: R %d G %d B %d", k/3+1, fr[k+1], fr[k], fr[k+2]), Note: colourName(fr[k+1], fr[k], fr[k+2])})
		}
	}
	x.card(SigCard{Ch: i, Tag: "protocol", Title: fmt.Sprintf("%s: addressable LED data (%d LEDs)", name, n),
		What:  fmt.Sprintf("**%s** is **WS2812 / NeoPixel LED data**: %d update%s, the last one setting **%d LEDs**. %s.", name, len(frames), map[bool]string{true: "", false: "s"}[len(frames) == 1], n, strings.Join(cols, ", ")),
		Why:   "Addressable LEDs (WS2812, SK6812, NeoPixel) share one data wire. Each LED takes 24 bits (green, red, blue, 8 bits each) and passes the rest along the strip. A short HIGH pulse is a 0, a long one a 1; a pause of more than 50 µs means 'show the colours now'.",
		Facts: append(facts, [2]string{"Updates", fmt.Sprintf("%d", len(frames))}, [2]string{"LEDs in the last update", fmt.Sprintf("%d", n)}), T0: starts[0], T1: d.t[len(d.t)-1]})
	return true
}

func colourName(r, g, b byte) string {
	if r < 8 && g < 8 && b < 8 {
		return "off"
	}
	mx := math.Max(float64(r), math.Max(float64(g), float64(b)))
	near := func(v byte) bool { return float64(v) > 0.6*mx }
	name := ""
	switch {
	case near(r) && near(g) && near(b):
		name = "white"
	case near(r) && near(g):
		name = "yellow"
	case near(r) && near(b):
		name = "magenta"
	case near(g) && near(b):
		name = "cyan"
	case near(r):
		name = "red"
	case near(g):
		name = "green"
	default:
		name = "blue"
	}
	return fmt.Sprintf("%s (%d, %d, %d)", name, r, g, b)
}

// ---------------------------------------------------------------- NEC infrared remotes

func (x *sigCtx) nec(i int, name string, d *digi, facts [][2]string) bool {
	if d.idle != 1 {
		return false
	}
	var codes []string
	found := 0
	for k := 1; k+2 < len(d.t); k++ {
		if d.v[k] != 0 {
			continue
		}
		lw, hw := d.t[k+1]-d.t[k], d.t[k+2]-d.t[k+1]
		if lw < 8e-3 || lw > 10e-3 {
			continue
		}
		if hw > 2e-3 && hw < 2.6e-3 {
			found++
			x.r.Decoded = append(x.r.Decoded, SigFrame{T0: d.t[k], T1: d.t[k+2], Ch: i, Proto: "NEC IR", Kind: "control", Text: "repeat", Note: "the button is being held down"})
			continue
		}
		if hw < 4e-3 || hw > 5e-3 {
			continue
		}
		var v uint32
		j := k + 2
		nb := 0
		for nb < 32 && j+1 < len(d.t) {
			sp := d.t[j+1] - d.t[j] // low mark, then a high space
			if j+2 >= len(d.t) {
				break
			}
			gap := d.t[j+2] - d.t[j+1]
			_ = sp
			if gap > 1.2e-3 {
				v |= 1 << nb
			}
			nb++
			j += 2
		}
		if nb < 32 {
			continue
		}
		found++
		a, na, c, nc := byte(v), byte(v>>8), byte(v>>16), byte(v>>24)
		ok := c^nc == 0xFF
		addr := fmt.Sprintf("0x%02X", a)
		if a^na != 0xFF {
			addr = fmt.Sprintf("0x%04X (extended)", uint16(v))
		}
		txt := fmt.Sprintf("address %s, command 0x%02X", addr, c)
		codes = append(codes, txt)
		kind, note := "data", "a button press"
		if !ok {
			kind, note = "error", "the check byte doesn't match"
		}
		x.r.Decoded = append(x.r.Decoded, SigFrame{T0: d.t[k], T1: d.t[min(j, len(d.t)-1)], Ch: i, Proto: "NEC IR", Kind: kind, Text: txt, Note: note})
	}
	if found == 0 {
		return false
	}
	x.concept("Infrared remotes (NEC)", i)
	what := fmt.Sprintf("**%s** is an **infrared remote control signal (NEC)**", name)
	if len(codes) > 0 {
		what += ": " + strings.Join(sigUniq(codes)[:min(4, len(sigUniq(codes)))], "; ")
	}
	x.card(SigCard{Ch: i, Tag: "protocol", Title: name + ": infrared remote (NEC)", What: what + ".",
		Why:   "This is the output of an IR receiver (like a VS1838B or TSOP38238): it goes LOW while it sees the remote's 38 kHz light. NEC codes start with a 9 ms burst, then 32 bits: the address of the device and the command (which button), each followed by its inverse as a check. In code, you'd compare the command number to decide what the button does.",
		Facts: append(facts, [2]string{"Codes", fmt.Sprintf("%d", found)}), T0: d.t[1], T1: d.t[len(d.t)-1]})
	return true
}

func sigUniq(a []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range a {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// ---------------------------------------------------------------- 1-Wire

func (x *sigCtx) onewire(i int, name string, d *digi, facts [][2]string) bool {
	if d.idle != 1 {
		return false
	}
	resets := 0
	var bytesAll [][]byte
	var cur []byte
	var acc byte
	nb := 0
	var tStart float64
	for k := 1; k+1 < len(d.t); k++ {
		if d.v[k] != 0 {
			continue
		}
		w := d.t[k+1] - d.t[k]
		switch {
		case w > 400e-6 && w < 1000e-6:
			if resets == 0 {
				tStart = d.t[k]
			}
			resets++
			if len(cur) > 0 {
				bytesAll = append(bytesAll, cur)
			}
			cur, acc, nb = nil, 0, 0
			// skip the presence pulse
			if k+3 < len(d.t) {
				if pw := d.t[k+3] - d.t[k+2]; pw > 50e-6 && pw < 300e-6 {
					k += 2
				}
			}
		case w < 130e-6 && resets > 0:
			if w < 30e-6 {
				acc |= 1 << nb
			}
			if nb++; nb == 8 {
				cur, acc, nb = append(cur, acc), 0, 0
			}
		}
	}
	if len(cur) > 0 {
		bytesAll = append(bytesAll, cur)
	}
	if resets == 0 || len(bytesAll) == 0 {
		return false
	}
	x.concept("1-Wire", i)
	var lines []string
	temp := math.NaN()
	for _, b := range bytesAll {
		var parts []string
		for j, v := range b {
			if j == 0 || (j == 1 && b[0] == 0xCC) {
				if n := oneWireCmd[v]; n != "" {
					parts = append(parts, fmt.Sprintf("%02X (%s)", v, n))
					continue
				}
			}
			parts = append(parts, fmt.Sprintf("%02X", v))
		}
		lines = append(lines, strings.Join(parts, " "))
		if len(b) >= 4 && b[0] == 0xCC && b[1] == 0xBE {
			raw := int16(uint16(b[2]) | uint16(b[3])<<8)
			temp = float64(raw) / 16
		}
		x.r.Decoded = append(x.r.Decoded, SigFrame{T0: tStart, T1: tStart, Ch: i, Proto: "1-Wire", Kind: "data", Text: strings.Join(parts, " ")})
	}
	what := fmt.Sprintf("**%s** is a **1-Wire bus**: %d reset%s, then: %s.", name, resets, map[bool]string{true: "", false: "s"}[resets == 1], strings.Join(lines[:min(3, len(lines))], " | "))
	if !math.IsNaN(temp) {
		what += fmt.Sprintf(" The sensor reported **%.2f °C**.", temp)
	}
	x.card(SigCard{Ch: i, Tag: "protocol", Title: name + ": 1-Wire bus", What: what,
		Why:   "1-Wire uses a single data wire (plus ground) with a pull-up resistor, usually 4.7 kΩ. The controller starts with a long LOW 'reset'; each device answers with a short 'presence' pulse. Bits are sent as short (1) or long (0) LOW pulses. The DS18B20 temperature sensor is the classic 1-Wire part: 'Skip ROM, Convert T', wait, then 'Skip ROM, Read Scratchpad'.",
		Facts: append(facts, [2]string{"Resets", fmt.Sprintf("%d", resets)}), T0: tStart, T1: d.t[len(d.t)-1]})
	return true
}

var oneWireCmd = map[byte]string{0x33: "Read ROM", 0x55: "Match ROM", 0xCC: "Skip ROM", 0xF0: "Search ROM", 0xEC: "Alarm Search", 0x44: "Convert T", 0xBE: "Read Scratchpad", 0x4E: "Write Scratchpad", 0x48: "Copy Scratchpad", 0xB8: "Recall EEPROM", 0xB4: "Read Power Supply"}

// ---------------------------------------------------------------- I2C (two wires)

func roleOf(name string) string {
	n := strings.ToUpper(strings.TrimSpace(name))
	for _, r := range []string{"SCL", "SDA", "SCLK", "SCK", "CLK", "MOSI", "MISO", "COPI", "CIPO", "SDO", "SDI", "CS", "SS", "NSS", "CE", "TX", "RX"} {
		if n == r || strings.HasPrefix(n, r+"_") || strings.HasSuffix(n, "_"+r) || strings.HasSuffix(n, " "+r) {
			return r
		}
	}
	return ""
}

func (x *sigCtx) findI2C(dig []*digi) {
	names := func(i int) string { return x.c.Chans[i].Name }
	type cand struct{ n, a, b int }
	var cands []cand
	for a := range dig {
		for b := range dig {
			if a == b || dig[a] == nil || dig[b] == nil || x.used[a] || x.used[b] {
				continue
			}
			ra, rb := roleOf(names(a)), roleOf(names(b))
			if ra != "" && ra != "SCL" || rb != "" && rb != "SDA" {
				continue
			}
			scl, sda := dig[a], dig[b]
			if scl.idle != 1 || sda.idle != 1 || scl.edges() < 18 {
				continue
			}
			starts, whileHigh := 0, 0
			for k := 1; k < len(sda.t); k++ {
				if scl.at(sda.t[k]) == 1 && scl.at(sda.t[k]-1e-12) == 1 {
					whileHigh++
					if sda.v[k] == 0 {
						starts++
					}
				}
			}
			// on a real I2C bus SDA only changes while SCL is high for START and STOP
			if sda.edges() > 0 && float64(whileHigh) > 0.5*float64(sda.edges()) && ra != "SCL" {
				continue
			}
			if ra == "SCL" && rb == "SDA" {
				starts += 1000
			}
			if starts > 0 {
				cands = append(cands, cand{starts, a, b})
			}
		}
	}
	sort.Slice(cands, func(i, j int) bool { return cands[i].n > cands[j].n })
	for k, cd := range cands {
		if k >= 12 {
			break
		}
		if x.decodeI2C(cd.a, cd.b, dig[cd.a], dig[cd.b]) {
			x.used[cd.a], x.used[cd.b] = true, true
			return
		}
	}
}

type i2cTx struct {
	t0, t1 float64
	addr   int
	read   bool
	data   []byte
	ack    []bool
	addrOK bool
}

func (x *sigCtx) decodeI2C(ci, di int, scl, sda *digi) bool {
	type ev struct {
		t    float64
		line int // 0 scl, 1 sda
		v    int
	}
	var evs []ev
	for k := 1; k < len(scl.t); k++ {
		evs = append(evs, ev{scl.t[k], 0, int(scl.v[k])})
	}
	for k := 1; k < len(sda.t); k++ {
		evs = append(evs, ev{sda.t[k], 1, int(sda.v[k])})
	}
	sort.SliceStable(evs, func(a, b int) bool { return evs[a].t < evs[b].t })
	sclV, sdaV := int(scl.v[0]), int(sda.v[0])
	var txs []*i2cTx
	var cur *i2cTx
	var bits []int
	var byteT0 float64
	var frames []SigFrame
	flush := func(t float64) {
		if cur != nil {
			cur.t1 = t
			txs = append(txs, cur)
			frames = append(frames, SigFrame{T0: t, T1: t, Ch: di, Proto: "I2C", Kind: "stop", Text: "STOP"})
		}
		cur = nil
	}
	for _, e := range evs {
		if e.line == 1 {
			if sclV == 1 && e.v == 0 { // START or repeated START
				if cur != nil {
					cur.t1 = e.t
					txs = append(txs, cur)
				}
				cur, bits = &i2cTx{t0: e.t, addr: -1}, nil
				frames = append(frames, SigFrame{T0: e.t, T1: e.t, Ch: di, Proto: "I2C", Kind: "start", Text: "START"})
			} else if sclV == 1 && e.v == 1 { // STOP
				flush(e.t)
			}
			sdaV = e.v
			continue
		}
		sclV = e.v
		if e.v != 1 || cur == nil {
			continue
		}
		if len(bits) == 0 {
			byteT0 = e.t
		}
		bits = append(bits, sdaV)
		if len(bits) == 9 {
			var b byte
			for k := 0; k < 8; k++ {
				b = b<<1 | byte(bits[k])
			}
			ack := bits[8] == 0
			if cur.addr < 0 {
				cur.addr, cur.read, cur.addrOK = int(b>>1), b&1 == 1, ack
				rw := map[bool]string{true: "read", false: "write"}[cur.read]
				dev := i2cDevice(cur.addr)
				note := ""
				if dev != "" {
					note = dev
				}
				frames = append(frames, SigFrame{T0: byteT0, T1: e.t, Ch: di, Proto: "I2C", Kind: "address", Text: fmt.Sprintf("Address 0x%02X, %s, %s", cur.addr, rw, map[bool]string{true: "ACK", false: "NACK"}[ack]), Note: note})
			} else {
				cur.data = append(cur.data, b)
				cur.ack = append(cur.ack, ack)
				k := "data"
				if !ack && !cur.read {
					k = "nack"
				}
				frames = append(frames, SigFrame{T0: byteT0, T1: e.t, Ch: di, Proto: "I2C", Kind: k, Text: fmt.Sprintf("0x%02X %s", b, map[bool]string{true: "ACK", false: "NACK"}[ack])})
			}
			bits = nil
		}
	}
	if cur != nil {
		flush(x.t1)
	}
	var good []*i2cTx
	acked := 0
	for _, t := range txs {
		if t.addr >= 0 {
			good = append(good, t)
			if t.addrOK {
				acked++
			}
		}
	}
	// real I2C: most addresses are answered, and the clock really clocks bytes
	if len(good) == 0 || acked == 0 || float64(acked) < 0.4*float64(len(good)) {
		return false
	}
	x.r.Decoded = append(x.r.Decoded, frames...)
	cn, dn := x.c.Chans[ci].Name, x.c.Chans[di].Name
	x.concept("I2C", ci)
	x.concept("I2C", di)
	x.concept("I2C address", di)
	x.concept("ACK and NACK", di)
	devs := map[int]int{}
	var order []int
	nacks := map[int]int{}
	for _, t := range good {
		if devs[t.addr] == 0 {
			order = append(order, t.addr)
		}
		devs[t.addr]++
		if !t.addrOK {
			nacks[t.addr]++
		}
	}
	var dl []string
	for _, a := range order {
		s := fmt.Sprintf("0x%02X", a)
		if n := i2cDevice(a); n != "" {
			s += " (" + n + ")"
		}
		dl = append(dl, s)
	}
	what := fmt.Sprintf("**%s** (clock) and **%s** (data) form an **I2C bus**. There are %d conversation%s, with %s.", cn, dn, len(good), map[bool]string{true: "", false: "s"}[len(good) == 1], strings.Join(dl, ", "))
	x.card(SigCard{Ch: ci, Tag: "protocol", Title: fmt.Sprintf("%s + %s: I2C bus", cn, dn), What: what,
		Why:   "I2C connects several chips over two wires: SCL is the clock from the controller (your board), SDA carries the data both ways. Each chip has a 7-bit address. A conversation starts with START (SDA falls while SCL is high), then the address and whether it's a write or a read, then the data. The receiver pulls SDA low after each byte to say it got it (ACK). Both wires need pull-up resistors (often 4.7 kΩ).",
		Facts: [][2]string{{"Clock (SCL)", cn}, {"Data (SDA)", dn}, {"Devices", strings.Join(dl, ", ")}, {"Conversations", fmt.Sprintf("%d", len(good))}, {"Clock speed", i2cSpeed(scl)}}, T0: good[0].t0, T1: good[len(good)-1].t1})
	// one card per kind of conversation (up to 6), with how often it happened
	seen := map[string]int{}
	var kinds []*i2cTx
	for _, t := range good {
		k := fmt.Sprintf("%d %v %v %s", t.addr, t.read, t.addrOK, hexBytes(t.data, 2))
		if seen[k] == 0 {
			kinds = append(kinds, t)
		}
		seen[k]++
	}
	for n, t := range kinds {
		if n >= 6 {
			break
		}
		times := seen[fmt.Sprintf("%d %v %v %s", t.addr, t.read, t.addrOK, hexBytes(t.data, 2))]
		dir := "writes to"
		if t.read {
			dir = "reads from"
		}
		dev := i2cDevice(t.addr)
		title := fmt.Sprintf("I2C: write to 0x%02X", t.addr)
		if t.read {
			title = fmt.Sprintf("I2C: read from 0x%02X", t.addr)
		}
		what := fmt.Sprintf("The board **%s** the device at **0x%02X**", dir, t.addr)
		if dev != "" {
			what += " (" + dev + ")"
		}
		if !t.addrOK {
			what += ", but **nothing answered** (NACK)."
		} else if len(t.data) > 0 {
			what += ": " + hexBytes(t.data, 16) + "."
		} else {
			what += " with no data (just checking it's there, like an I2C scanner does)."
		}
		why := i2cWhy(t)
		if times > 1 {
			what += fmt.Sprintf(" This happens **%d times** in the capture.", times)
		}
		x.card(SigCard{Ch: di, Tag: "event", Title: title, What: what, Why: why, T0: t.t0, T1: t.t1})
	}
	for a, n := range nacks {
		x.warn(SigWarn{Title: fmt.Sprintf("I2C: nothing answered at 0x%02X", a), Text: fmt.Sprintf("%d conversation(s) with address 0x%02X got no ACK. Either no device has that address (check the datasheet: many have two, such as 0x3C/0x3D or 0x76/0x77), it isn't powered, SDA and SCL are swapped, or the pull-up resistors are missing.", n, a), Ch: di})
	}
	return true
}

func i2cSpeed(scl *digi) string {
	var p []float64
	r := scl.rises()
	for k := 1; k < len(r); k++ {
		if d := r[k] - r[k-1]; d < 50e-6 {
			p = append(p, d)
		}
	}
	if len(p) == 0 {
		return "—"
	}
	f := 1 / median(p)
	mode := "standard mode"
	switch {
	case f > 1.2e6:
		mode = "fast mode plus"
	case f > 150e3:
		mode = "fast mode"
	}
	return fmtHz(f) + " (" + mode + ")"
}

func i2cWhy(t *i2cTx) string {
	if !t.addrOK {
		return "No chip pulled SDA low after the address, so nobody is listening at that address."
	}
	switch {
	case (t.addr == 0x3C || t.addr == 0x3D) && !t.read && len(t.data) > 0:
		if t.data[0] == 0x00 {
			return "On an SSD1306 OLED, a first byte of 0x00 means 'commands follow' (switch on, set contrast, set where to draw)."
		}
		if t.data[0] == 0x40 {
			return "On an SSD1306 OLED, a first byte of 0x40 means 'pixel data follows': the picture being drawn."
		}
	case !t.read && len(t.data) == 1:
		return "Writing a single byte usually picks a register inside the chip; the next conversation then reads from it."
	case t.read:
		return "The board reads bytes back: sensor readings, a chip ID or settings. The last byte gets a NACK on purpose, meaning 'that's enough'."
	}
	return "A write: the first byte usually picks a register inside the chip, and the rest are the value written to it."
}

func i2cDevice(a int) string {
	m := map[int]string{
		0x0C: "AK8963 compass", 0x10: "VEML7700 light sensor", 0x18: "LIS3DH accelerometer or MCP9808", 0x19: "LIS3DH accelerometer", 0x1D: "ADXL345 / MMA8452 accelerometer", 0x1E: "HMC5883L compass",
		0x20: "PCF8574 / MCP23017 I/O expander", 0x21: "PCF8574 / MCP23017 I/O expander", 0x23: "BH1750 light sensor", 0x27: "PCF8574 LCD backpack", 0x29: "VL53L0X distance or TCS34725 colour sensor",
		0x38: "AHT10/AHT20 humidity sensor or FT6206 touch", 0x39: "APDS-9960 gesture / TSL2561 light sensor", 0x3C: "SSD1306 / SH1106 OLED display", 0x3D: "SSD1306 OLED display", 0x3F: "PCF8574 LCD backpack",
		0x40: "INA219 power monitor, PCA9685 servo driver or HTU21D", 0x41: "INA219 power monitor", 0x44: "SHT31 humidity sensor", 0x45: "SHT31 humidity sensor", 0x48: "ADS1115 ADC or TMP102 temperature", 0x49: "ADS1115 ADC",
		0x4A: "ADS1115 ADC", 0x4B: "ADS1115 ADC", 0x50: "24Cxx EEPROM", 0x51: "24Cxx EEPROM", 0x53: "ADXL345 accelerometer", 0x57: "MAX30102 pulse sensor or AT24C32 EEPROM", 0x5A: "MLX90614 IR thermometer or MPR121 touch",
		0x5C: "AM2320 humidity sensor", 0x60: "Si5351 clock or MCP4725 DAC", 0x62: "MCP4725 DAC or SCD40 CO₂ sensor", 0x68: "MPU6050 motion sensor or DS3231/DS1307 clock", 0x69: "MPU6050 motion sensor", 0x70: "TCA9548A I2C multiplexer or HT16K33 LED driver",
		0x76: "BME280/BMP280 pressure sensor", 0x77: "BME280/BMP280/BMP180 pressure sensor",
	}
	return m[a]
}

// ---------------------------------------------------------------- SPI (clock + data, maybe chip select)

func (x *sigCtx) findSPI(dig []*digi) {
	var free []int
	for i, d := range dig {
		if d != nil && !x.used[i] && d.edges() > 0 {
			free = append(free, i)
		}
	}
	if len(free) < 2 {
		return
	}
	// the clock: named, or the busiest channel whose edges come in tidy bursts
	clk := -1
	for _, i := range free {
		if r := roleOf(x.c.Chans[i].Name); r == "SCK" || r == "SCLK" || r == "CLK" {
			clk = i
		}
	}
	if clk < 0 {
		best := 0
		for _, i := range free {
			d := dig[i]
			if d.edges() < 16 || d.edges() <= best {
				continue
			}
			var gaps []float64
			for k := 2; k < len(d.t); k++ {
				gaps = append(gaps, d.t[k]-d.t[k-1])
			}
			m := median(gaps)
			tidy := 0
			for _, g := range gaps {
				if g > 0.7*m && g < 1.4*m {
					tidy++
				}
			}
			if float64(tidy) > 0.6*float64(len(gaps)) {
				// a steady clock running the whole time is a clock, not SPI
				r := d.rises()
				var p []float64
				for k := 1; k < len(r); k++ {
					p = append(p, r[k]-r[k-1])
				}
				mm, ss := meanStd(p)
				if mm > 0 && ss/mm < 0.06 && (r[len(r)-1]-r[0])/(x.t1-x.t0) > 0.5 {
					continue
				}
				best, clk = d.edges(), i
			}
		}
	}
	if clk < 0 {
		return
	}
	ck := dig[clk]
	cs := -1
	var data []int
	for _, i := range free {
		if i == clk {
			continue
		}
		r := roleOf(x.c.Chans[i].Name)
		d := dig[i]
		if r == "CS" || r == "SS" || r == "NSS" || r == "CE" || (r == "" && cs < 0 && d.idle == 1 && d.edges() >= 2 && d.edges() < ck.edges()/8 && clocksWhileLow(d, ck) > 0.9) {
			cs = i
			continue
		}
		if r == "TX" || r == "RX" || r == "SCL" || r == "SDA" {
			continue
		}
		data = append(data, i) // checked below: they must only change during transfers
	}
	// transfers: chip select low, or bursts of clock
	type win struct{ a, b float64 }
	var wins []win
	if cs >= 0 {
		d := dig[cs]
		for k := 1; k < len(d.t); k++ {
			if d.v[k] == 0 {
				end := x.t1
				if k+1 < len(d.t) {
					end = d.t[k+1]
				}
				wins = append(wins, win{d.t[k], end})
			}
		}
	} else {
		var gaps []float64
		for k := 2; k < len(ck.t); k++ {
			gaps = append(gaps, ck.t[k]-ck.t[k-1])
		}
		m := median(gaps)
		a := ck.t[1]
		for k := 2; k < len(ck.t); k++ {
			if ck.t[k]-ck.t[k-1] > 6*m {
				wins = append(wins, win{a - m, ck.t[k-1] + m})
				a = ck.t[k]
			}
		}
		wins = append(wins, win{a - m, ck.t[len(ck.t)-1] + m})
	}
	// keep only lines whose changes happen inside the transfers
	inWin := func(t float64) bool {
		k := sort.Search(len(wins), func(j int) bool { return wins[j].b >= t })
		return k < len(wins) && wins[k].a <= t
	}
	var kept []int
	for _, ch := range data {
		d := dig[ch]
		in := 0
		for k := 1; k < len(d.t); k++ {
			if inWin(d.t[k]) {
				in++
			}
		}
		if d.edges() > 0 && float64(in) >= 0.9*float64(d.edges()) {
			kept = append(kept, ch)
		}
	}
	data = kept
	if len(data) == 0 {
		return
	}
	cpol := ck.idle
	// sample on the first edge after idle (mode 0 and mode 2)
	var sampleEdges []float64
	if cpol == 0 {
		sampleEdges = ck.rises()
	} else {
		sampleEdges = ck.falls()
	}
	type xfer struct {
		a, b  float64
		bytes [][]byte
	}
	var xs []xfer
	for _, w := range wins {
		xf := xfer{a: w.a, b: w.b, bytes: make([][]byte, len(data))}
		for di, ch := range data {
			var acc byte
			n := 0
			for _, e := range sampleEdges {
				if e < w.a || e > w.b {
					continue
				}
				acc = acc<<1 | byte(dig[ch].at(e))
				if n++; n == 8 {
					xf.bytes[di] = append(xf.bytes[di], acc)
					acc, n = 0, 0
				}
			}
		}
		if len(xf.bytes[0]) > 0 {
			xs = append(xs, xf)
		}
	}
	if len(xs) == 0 {
		return
	}
	x.used[clk] = true
	if cs >= 0 {
		x.used[cs] = true
	}
	// which line is which: a known command in the first byte marks MOSI (from the board)
	mosi := -1
	if len(data) == 2 && len(xs[0].bytes[0]) > 0 && len(xs[0].bytes[1]) > 0 {
		if spiCommand(xs[0].bytes[0][0]) != "" {
			mosi = 0
		} else if spiCommand(xs[0].bytes[1][0]) != "" {
			mosi = 1
		}
	}
	dname := func(di int) string {
		n := x.c.Chans[data[di]].Name
		if r := roleOf(n); r != "" {
			return n
		}
		if mosi >= 0 {
			if di == mosi {
				return n + " (MOSI, board → chip)"
			}
			return n + " (MISO, chip → board)"
		}
		if len(data) == 1 {
			return n + " (data)"
		}
		return n + fmt.Sprintf(" (data %c)", 'A'+di)
	}
	for _, xf := range xs {
		var parts []string
		for di := range data {
			parts = append(parts, dname(di)+": "+hexBytes(xf.bytes[di], 16))
		}
		note := ""
		if len(xf.bytes[0]) > 0 {
			note = spiCommand(xf.bytes[0][0])
		}
		x.r.Decoded = append(x.r.Decoded, SigFrame{T0: xf.a, T1: xf.b, Ch: data[0], Proto: "SPI", Kind: "data", Text: strings.Join(parts, " · "), Note: note})
	}
	for _, d := range data {
		x.used[d] = true
		x.concept("SPI", d)
	}
	x.concept("SPI", clk)
	if cs >= 0 {
		x.concept("Chip select", cs)
	}
	mode := 0
	if cpol == 1 {
		mode = 2
	}
	cn := x.c.Chans[clk].Name
	var dn []string
	for di := range data {
		dn = append(dn, dname(di))
	}
	csn := "no chip select (transfers split by pauses)"
	if cs >= 0 {
		csn = x.c.Chans[cs].Name
	}
	first := xs[0]
	var fp []string
	for di := range data {
		fp = append(fp, dname(di)+" "+hexBytes(first.bytes[di], 8))
	}
	what := fmt.Sprintf("**%s** is the clock of an **SPI bus**, with data on %s. There are %d transfer%s; the first carries %s.", cn, strings.Join(dn, " and "), len(xs), map[bool]string{true: "", false: "s"}[len(xs) == 1], strings.Join(fp, ", "))
	if mosi >= 0 {
		if c := spiCommand(first.bytes[mosi][0]); c != "" {
			what += " That first byte looks like " + c + "."
			if first.bytes[mosi][0] == 0x9F && len(first.bytes[1-mosi]) >= 4 {
				if id := jedecName(first.bytes[1-mosi][1:4]); id != "" {
					what += " The chip answered " + hexBytes(first.bytes[1-mosi][1:4], 3) + ": **" + id + "**."
				}
			}
		}
	}
	var cp []float64
	r := ck.rises()
	for k := 1; k < len(r); k++ {
		if d := r[k] - r[k-1]; d < 20e-6 {
			cp = append(cp, d)
		}
	}
	speed := "—"
	if len(cp) > 0 {
		speed = fmtHz(1 / median(cp))
	}
	x.card(SigCard{Ch: clk, Tag: "protocol", Title: cn + ": SPI bus", What: what,
		Why:   "SPI is a fast bus for displays, SD cards, flash memory and some sensors. The controller drives the clock (SCK); on each clock edge one bit goes out on MOSI and one comes back on MISO at the same time. Chip select (CS) goes LOW to pick which chip is listening. It has no addresses and no ACKs, which is why it's faster than I2C.",
		Facts: [][2]string{{"Clock", cn + " at " + speed}, {"Data", strings.Join(dn, ", ")}, {"Chip select", csn}, {"Mode", fmt.Sprintf("%d (clock idles %s, read on the first edge)", mode, map[int]string{0: "LOW", 1: "HIGH"}[cpol])}, {"Transfers", fmt.Sprintf("%d", len(xs))}}, T0: xs[0].a, T1: xs[len(xs)-1].b})
}

func clocksWhileLow(cs, ck *digi) float64 {
	in, all := 0, 0
	for k := 1; k < len(ck.t); k++ {
		all++
		if cs.at(ck.t[k]) == 0 {
			in++
		}
	}
	if all == 0 {
		return 0
	}
	return float64(in) / float64(all)
}

func spiCommand(b byte) string {
	m := map[byte]string{0x9F: "a 'read ID' (JEDEC ID) command for a flash chip", 0x03: "a 'read data' command for a flash chip", 0x0B: "a 'fast read' command for a flash chip", 0x06: "a 'write enable' command for a flash chip", 0x05: "a 'read status' command for a flash chip", 0x02: "a 'page program' (write) command for a flash chip", 0x20: "a 'sector erase' command for a flash chip", 0xAB: "a 'wake up' command for a flash chip", 0xB9: "a 'power down' command for a flash chip"}
	return m[b]
}

// isTexty is true if most runes are ordinary text.
func isTexty(s string) bool {
	n, ok := 0, 0
	for _, r := range s {
		n++
		if unicode.IsPrint(r) || r == '\t' {
			ok++
		}
	}
	return n > 0 && float64(ok)/float64(n) > 0.9
}

// jedecName names a flash chip from its JEDEC ID (maker, type, capacity).
func jedecName(id []byte) string {
	makers := map[byte]string{0xEF: "Winbond", 0xC8: "GigaDevice", 0xC2: "Macronix", 0x20: "Micron/XMC", 0x1F: "Adesto", 0xBF: "Microchip SST", 0x9D: "ISSI", 0x01: "Spansion/Infineon"}
	m := makers[id[0]]
	if m == "" {
		return ""
	}
	if id[2] >= 0x10 && id[2] <= 0x1A {
		bytes := 1 << id[2]
		size := fmt.Sprintf("%d KB", bytes>>10)
		if bytes >= 1<<20 {
			size = fmt.Sprintf("%d MB", bytes>>20)
		}
		name := m + " flash, " + size
		if id[0] == 0xEF && id[1] == 0x40 {
			name = fmt.Sprintf("Winbond W25Q%d flash (%s)", (bytes*8)>>20, size)
		}
		return name
	}
	return m + " flash"
}
