package meterie

// proto2.go: more meter protocols, added after checking them against the
// libsigrok decoder sources (src/dmm/ut71x.c, es519xx.c, bm25x.c, bm86x.c,
// hardware/uni-t-ut181a) and the community Bluetooth-DMM decoder for the
// ANENG / ZOYI / BSIDE family (XOR key + bit-reversal + 7-segment map).

import (
	"bytes"
	"encoding/binary"
	"math"
	"strconv"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Shared helpers

// prefixForExp picks the display prefix the LCD most likely uses for a value
// that is (integer count) × 10^exp, given how many digits the display has:
// the largest SI prefix that keeps the number of decimals ≤ digits-1.
func prefixForExp(exp, digits int, unit string) (prefix string, decimals int) {
	if !unitAllowsPrefix(unit) {
		d := -exp
		if d < 0 {
			d = 0
		}
		return "", d
	}
	for p3 := 9; p3 >= -12; p3 -= 3 {
		dec := p3 - exp
		if dec >= 0 && dec <= digits-1 {
			return prefixFor3(p3), dec
		}
	}
	d := -exp
	if d < 0 {
		d = 0
	}
	return "", d
}

func prefixFor3(p3 int) string {
	switch p3 {
	case -12:
		return "p"
	case -9:
		return "n"
	case -6:
		return "µ"
	case -3:
		return "m"
	case 3:
		return "k"
	case 6:
		return "M"
	case 9:
		return "G"
	}
	return ""
}

// countDisplay renders count×10^exp in the given prefix with dec decimals.
func countDisplay(count int, dec int, neg bool) string {
	s := strconv.Itoa(count)
	if dec > 0 {
		for len(s) <= dec {
			s = "0" + s
		}
		s = s[:len(s)-dec] + "." + s[len(s)-dec:]
	}
	if neg {
		s = "-" + s
	}
	return s
}

// fillCountReading fills Value/Display/Prefix for a count × 10^exp reading.
func fillCountReading(r *Reading, count, exp, digits int, neg bool) {
	p, dec := prefixForExp(exp, digits, r.Unit)
	r.Prefix = p
	r.Value = float64(count) * pow10(exp)
	if neg {
		r.Value = -r.Value
	}
	r.Display = countDisplay(count, dec, neg)
}

func setOL(r *Reading) {
	r.Flags |= FOL
	r.Value = math.NaN()
	r.Display = "OL"
}

// crlfFramer collects fixed-size packets that end in CR LF (7-bit data).
type crlfFramer struct {
	buf  []byte
	size int
	mask byte
}

func (f *crlfFramer) feed(b []byte, parse func([]byte) (Reading, bool)) []Reading {
	for _, c := range b {
		f.buf = append(f.buf, c&f.mask)
	}
	var out []Reading
	for {
		i := bytes.Index(f.buf, []byte{'\r', '\n'})
		if i < 0 {
			if len(f.buf) > 256 {
				f.buf = f.buf[len(f.buf)-32:]
			}
			return out
		}
		end := i + 2
		if end >= f.size {
			if r, ok := parse(f.buf[end-f.size : end]); ok {
				out = append(out, r)
			}
		}
		f.buf = f.buf[end:]
	}
}

// ---------------------------------------------------------------------------
// UNI-T UT71A/B/C/D/E, UT804, Voltcraft VC-920/940/960, Tenma 72-7730/7732/9380A.
// 11 bytes, 2400 7O1: digits[5] range func flags1 flags2 CR LF (libsigrok ut71x.c).

var ut71xExp = [16][8]int{
	{-5, 0, 0, 0, 0, 0, 0, 0},          // AC mV
	{0, -4, -3, -2, -1, 0, 0, 0},       // DC V
	{0, -4, -3, -2, -1, 0, 0, 0},       // AC V
	{-5, 0, 0, 0, 0, 0, 0, 0},          // DC mV
	{0, -2, -1, 0, 1, 2, 3, 0},         // resistance
	{0, -12, -11, -10, -9, -8, -7, -6}, // capacitance
	{-1, 0, 0, 0, 0, 0, 0, 0},          // temp °C
	{-8, -7, 0, 0, 0, 0, 0, 0},         // µA
	{-6, -5, 0, 0, 0, 0, 0, 0},         // mA
	{0, -3, 0, 0, 0, 0, 0, 0},          // 10 A
	{-1, 0, 0, 0, 0, 0, 0, 0},          // continuity
	{-4, 0, 0, 0, 0, 0, 0, 0},          // diode
	{-3, -2, -1, 0, 1, 2, 3, 4},        // frequency
	{-1, 0, 0, 0, 0, 0, 0, 0},          // temp °F
	{0, 0, 0, 0, 0, 0, 0, 0},           // power (UT71E)
	{-2, 0, 0, 0, 0, 0, 0, 0},          // 4-20 mA loop, %
}

type UT71x struct{ f crlfFramer }

func NewUT71x() *UT71x { return &UT71x{f: crlfFramer{size: 11, mask: 0x7F}} }

func (d *UT71x) Feed(b []byte) []Reading { return d.f.feed(b, ParseUT71x) }

func ParseUT71x(b []byte) (Reading, bool) {
	if len(b) != 11 || b[9] != '\r' || b[10] != '\n' {
		return Reading{}, false
	}
	idx := int(b[5]) - '0'
	fn := int(b[6]) - '0'
	if idx < 0 || idx > 7 || fn < 0 || fn > 15 {
		return Reading{}, false
	}
	var r Reading
	r.T = time.Now()
	r.Counts = 40000
	switch fn {
	case 0, 2:
		r.Q, r.Unit, r.Flags = QVoltage, "V", FAC
	case 1, 3:
		r.Q, r.Unit, r.Flags = QVoltage, "V", FDC
	case 4:
		r.Q, r.Unit = QResistance, "Ω"
	case 5:
		r.Q, r.Unit = QCapacitance, "F"
	case 6:
		r.Q, r.Unit = QTemperature, "°C"
	case 7, 8, 9:
		r.Q, r.Unit, r.Flags = QCurrent, "A", FDC
	case 10:
		r.Q, r.Unit, r.Flags = QContinuity, "Ω", FBeep
	case 11:
		r.Q, r.Unit, r.Flags = QDiode, "V", FDiode|FDC
	case 12:
		r.Q, r.Unit = QFrequency, "Hz"
	case 13:
		r.Q, r.Unit, r.Flags = QTemperature, "°F", FFahrenheit
	case 14:
		r.Q, r.Unit = QPower, "W"
	case 15:
		r.Q, r.Unit = QPercent, "%"
	}
	if b[7]&1 != 0 {
		r.Flags |= FAC
	}
	if b[7]&2 != 0 {
		r.Flags |= FDC
	}
	if b[8]&1 != 0 {
		r.Flags |= FAuto
	}
	if b[8]&2 != 0 {
		r.Flags |= FManual
	}
	neg := b[8]&4 != 0
	if r.Q == QFrequency && neg { // frequency + sign bit = duty cycle
		r.Q, r.Unit, neg = QDuty, "%", false
	}
	ds := string(b[0:5])
	if ds == "::0<:" || ds == ":<0::" {
		setOL(&r)
		return r, true
	}
	n := 5
	if b[4] == ':' {
		n = 4
	}
	cnt := 0
	for _, c := range b[:n] {
		if c < '0' || c > '9' {
			return Reading{}, false
		}
		cnt = cnt*10 + int(c-'0')
	}
	exp := ut71xExp[fn][idx]
	if n == 4 { // 4-digit modes: sigrok scales ×10 to keep the table uniform
		exp++
	}
	fillCountReading(&r, cnt, exp, n, neg)
	return r, true
}

// ---------------------------------------------------------------------------
// Cyrustek ES519xx 11-byte packets (UNI-T UT60G, Tenma 72-7750: 19200 7O1;
// ISO-TECH IDM103N: 2400 7O1). range digits[4] func status opt1 opt2 CR LF.

type ES519xx11 struct {
	f    crlfFramer
	baud int
}

func NewES519xx11(baud int) *ES519xx11 {
	return &ES519xx11{f: crlfFramer{size: 11, mask: 0x7F}, baud: baud}
}

func (d *ES519xx11) Feed(b []byte) []Reading {
	return d.f.feed(b, func(p []byte) (Reading, bool) { return ParseES519xx11(p, d.baud) })
}

var es11Exp2400 = [9][8]int{
	{-4, -3, -2, -1, 0, 0, 0, 0},
	{-7, -6, 0, 0, 0, 0, 0, 0},
	{-5, -4, 0, 0, 0, 0, 0, 0},
	{-2, 0, 0, 0, 0, 0, 0, 0},
	{1, 2, 3, 4, 5, 6, 0, 0},
	{-1, 0, 1, 2, 3, 4, 0, 0},
	{0, 1, 2, 3, 4, 5, 0, 0},
	{-12, -11, -10, -9, -8, -7, -6, -5},
	{-3, 0, 0, 0, 0, 0, 0, 0},
}

var es11Exp19200 = [9][8]int{
	{-3, -2, -1, 0, -4, 0, 0, 0},
	{-7, -6, 0, 0, 0, 0, 0, 0},
	{-5, -4, 0, 0, 0, 0, 0, 0},
	{-3, -2, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0, 0, 0, 0},
	{-1, 0, 1, 2, 3, 4, 0, 0},
	{0, 1, 2, 3, 4, 0, 0, 0},
	{-12, -11, -10, -9, -8, -7, -6, 0},
	{-3, 0, 0, 0, 0, 0, 0, 0},
}

func ParseES519xx11(b []byte, baud int) (Reading, bool) {
	if len(b) != 11 || b[9] != '\r' || b[10] != '\n' {
		return Reading{}, false
	}
	fnb, st, o1, o2 := b[5], b[6], b[7], b[8]
	judge := st&8 != 0
	neg := st&4 != 0
	lowbat := st&2 != 0
	ol := st&1 != 0
	var r Reading
	r.T = time.Now()
	r.Counts = 4000
	if baud != 2400 {
		r.Counts = 22000
	}
	vahz := false
	if baud == 2400 {
		if o1&8 != 0 {
			r.Flags |= FPeakMax
		}
		if o1&4 != 0 {
			r.Flags |= FPeakMin
		}
		vahz = o1&1 != 0
	} else {
		if o1&8 != 0 {
			r.Flags |= FHold
		}
		if o1&4 != 0 {
			r.Flags |= FMax
		}
		if o1&2 != 0 {
			r.Flags |= FMin
		}
		vahz = o2&1 != 0
	}
	if o2&8 != 0 {
		r.Flags |= FDC
	}
	if o2&4 != 0 {
		r.Flags |= FAC
	}
	auto := o2&2 != 0
	if lowbat {
		r.Flags |= FLowBat
	}
	// function byte
	mode := -1
	micro, milli := false, false
	switch fnb {
	case 0x3B:
		r.Q, r.Unit, mode = QVoltage, "V", 0
	case 0x3D:
		r.Q, r.Unit, micro, auto = QCurrent, "A", true, true
	case 0x3F:
		r.Q, r.Unit, milli, auto = QCurrent, "A", true, true
	case 0x30:
		r.Q, r.Unit, auto = QCurrent, "A", true
	case 0x39:
		r.Q, r.Unit, auto = QCurrent, "A", false
	case 0x33:
		r.Q, r.Unit, mode = QResistance, "Ω", 5
	case 0x35:
		r.Q, r.Unit, mode = QContinuity, "Ω", 5
		r.Flags |= FBeep
	case 0x31:
		r.Q, r.Unit, mode = QDiode, "V", 8
		r.Flags |= FDiode | FDC
	case 0x32:
		if judge {
			r.Q, r.Unit = QUnitless, "RPM"
			mode = 4
		} else {
			r.Q, r.Unit, mode = QFrequency, "Hz", 6
		}
	case 0x36:
		r.Q, r.Unit, mode = QCapacitance, "F", 7
	case 0x34:
		if judge {
			r.Q, r.Unit = QTemperature, "°C"
		} else {
			r.Q, r.Unit = QTemperature, "°F"
			r.Flags |= FFahrenheit
		}
	default:
		return Reading{}, false
	}
	if vahz && (r.Q == QVoltage || r.Q == QCurrent) {
		micro, milli = false, false
		if judge {
			r.Q, r.Unit, mode = QUnitless, "RPM", 4
		} else {
			r.Q, r.Unit, mode = QFrequency, "Hz", 6
		}
	}
	if baud == 2400 && r.Q == QCurrent && (milli || !auto) { // inverted mA/A on 2400-baud chips
		milli = !milli
		auto = true
	}
	if r.Q == QCurrent {
		switch {
		case micro:
			mode = 1
		case milli:
			mode = 2
		case auto:
			mode = 3
		default:
			mode = 4
		}
	}
	if auto {
		r.Flags |= FAuto
	} else {
		r.Flags |= FManual
	}
	if ol {
		setOL(&r)
		return r, true
	}
	cnt := 0
	for _, c := range b[1:5] {
		if c < '0' || c > '9' {
			return Reading{}, false
		}
		cnt = cnt*10 + int(c-'0')
	}
	idx := int(b[0]) - '0'
	if idx < 0 || idx > 7 {
		return Reading{}, false
	}
	exp := 0
	switch {
	case r.Q == QTemperature:
		exp = 0
	case mode < 0:
		return Reading{}, false
	case baud == 2400:
		exp = es11Exp2400[mode][idx]
	default:
		exp = es11Exp19200[mode][idx]
	}
	fillCountReading(&r, cnt, exp, 4, neg)
	return r, true
}

// ---------------------------------------------------------------------------
// Brymen BM25x (BM257s and friends), 9600 8N1 with DTR+RTS high to power the
// IR adapter. 15 bytes: 0x02, then bytes whose high nibble is their index.

type BM25x struct{ buf []byte }

func (d *BM25x) Feed(b []byte) []Reading {
	d.buf = append(d.buf, b...)
	var out []Reading
	for len(d.buf) >= 15 {
		if d.buf[0] == 0x02 {
			ok := true
			for i := 1; i < 15; i++ {
				if int(d.buf[i]>>4) != i {
					ok = false
					break
				}
			}
			if ok {
				if r, ok2 := ParseBM25x(d.buf[:15]); ok2 {
					out = append(out, r)
				}
				d.buf = d.buf[15:]
				continue
			}
		}
		d.buf = d.buf[1:]
	}
	return out
}

func bm25xDigit(num int, b []byte) int {
	v := int(b[3+2*num]&0x0E) | int(b[4+2*num]<<4)&0xF0
	switch v {
	case 0xBE:
		return 0
	case 0xA0:
		return 1
	case 0xDA:
		return 2
	case 0xF8:
		return 3
	case 0xE4:
		return 4
	case 0x7C:
		return 5
	case 0x7E:
		return 6
	case 0xA8:
		return 7
	case 0xFE:
		return 8
	case 0xFC:
		return 9
	case 0x00:
		return ' '
	case 0x40:
		return '-'
	case 0x16:
		return 'L'
	case 0x1E:
		return 'C'
	case 0x4E:
		return 'F'
	case 0x5E:
		return 'E'
	case 0x62:
		return 'n'
	case 0x42:
		return 'r'
	}
	return -1
}

func ParseBM25x(b []byte) (Reading, bool) {
	if len(b) != 15 || b[0] != 2 {
		return Reading{}, false
	}
	var r Reading
	r.T = time.Now()
	r.Counts = 6000
	r.Q, r.Unit = QUnitless, ""
	if b[1]&8 != 0 {
		r.Flags |= FAuto
	} else {
		r.Flags |= FManual
	}
	if b[1]&4 != 0 {
		r.Flags |= FDC
	}
	if b[1]&2 != 0 {
		r.Flags |= FAC
	}
	if b[1]&1 != 0 {
		r.Flags |= FRel
	}
	if b[11]&8 != 0 {
		r.Flags |= FHold
	}
	if b[13]&8 != 0 {
		r.Flags |= FMax
	}
	if b[14]&8 != 0 {
		r.Flags |= FMin
	}
	if b[14]&4 != 0 {
		r.Q, r.Unit = QVoltage, "V"
		if !r.Flags.Has(FAC) && !r.Flags.Has(FDC) {
			r.Q = QDiode
			r.Flags |= FDiode | FDC
		}
	}
	if b[14]&2 != 0 {
		r.Q, r.Unit = QCurrent, "A"
	}
	if b[12]&4 != 0 {
		r.Q, r.Unit = QResistance, "Ω"
	}
	if b[13]&4 != 0 {
		r.Q, r.Unit = QCapacitance, "F"
	}
	if b[12]&2 != 0 {
		r.Q, r.Unit = QFrequency, "Hz"
	}
	switch bm25xDigit(3, b) {
	case 'C':
		r.Q, r.Unit = QTemperature, "°C"
	case 'F':
		r.Q, r.Unit = QTemperature, "°F"
		r.Flags |= FFahrenheit
	}
	// prefix
	pfx := ""
	switch {
	case b[11]&2 != 0:
		pfx = "M"
	case b[11]&1 != 0:
		pfx = "k"
	case b[13]&1 != 0:
		pfx = "m"
	case b[13]&2 != 0:
		pfx = "µ"
	case b[12]&1 != 0:
		pfx = "n"
	}
	if !unitAllowsPrefix(r.Unit) {
		pfx = ""
	}
	r.Prefix = pfx
	// digits
	val, n := 0, 0
	for i := 0; i < 4; i++ {
		dg := bm25xDigit(i, b)
		if i == 3 && (dg == 'C' || dg == 'F') {
			break
		}
		if dg < 0 || dg > 9 {
			if bm25xDigit(1, b) == 0 && bm25xDigit(2, b) == 'L' {
				setOL(&r)
				return r, true
			}
			return Reading{}, false
		}
		val = val*10 + dg
		n++
	}
	// decimal point: bit0 of b[11-2*i] for i = 1..3
	point := 0
	for i := 1; i < 4; i++ {
		if b[11-2*i]&1 == 0 {
			continue
		}
		if point != 0 {
			return Reading{}, false
		}
		point = i
	}
	dec := 0
	if point != 0 {
		dec = point + n - 4
		if dec < 0 || dec > 3 {
			dec = 0
		}
	}
	neg := b[3]&1 != 0
	r.Display = countDisplay(val, dec, neg)
	v := float64(val) / pow10(dec)
	if neg {
		v = -v
	}
	r.Value = v * prefixMult(pfx)
	return r, true
}

// ---------------------------------------------------------------------------
// Brymen BM869s / BM867s / BM859s via the BU-86X USB cable (HID 0820:0001).
// The PC sends 00 86 66 (after report ID 0); the meter answers with a
// 24-byte LCD-segment packet (bytes 16..19 are 0x86). Main + secondary display.

type BM86x struct {
	buf   []byte
	Which int // 0 main display, 1 secondary
}

func (d *BM86x) PollRequest() []byte         { return []byte{0x00, 0x00, 0x86, 0x66} }
func (d *BM86x) PollInterval() time.Duration { return 300 * time.Millisecond }

func (d *BM86x) Feed(b []byte) []Reading {
	d.buf = append(d.buf, b...)
	var out []Reading
	for len(d.buf) >= 24 {
		if d.buf[16] == 0x86 && d.buf[17] == 0x86 && d.buf[18] == 0x86 && d.buf[19] == 0x86 {
			if r, ok := ParseBM86x(d.buf[:24]); ok {
				out = append(out, r)
			}
			d.buf = d.buf[24:]
			continue
		}
		d.buf = d.buf[1:]
	}
	if len(d.buf) > 512 {
		d.buf = d.buf[len(d.buf)-48:]
	}
	return out
}

func bm86xChar(b byte) byte {
	switch b >> 1 {
	case 0x20:
		return '-'
	case 0x5F:
		return '0'
	case 0x50:
		return '1'
	case 0x6D:
		return '2'
	case 0x7C:
		return '3'
	case 0x72:
		return '4'
	case 0x3E:
		return '5'
	case 0x3F:
		return '6'
	case 0x54:
		return '7'
	case 0x7F:
		return '8'
	case 0x7E:
		return '9'
	case 0x0F:
		return 'C'
	case 0x27:
		return 'F'
	case 0x0B:
		return 'L'
	case 0x79:
		return 'd'
	case 0x10:
		return 'i'
	case 0x39:
		return 'o'
	}
	return 0
}

// bm86xDigits mirrors brymen_bm86x_parse_digits: returns text, decimals and temp unit.
func bm86xDigits(pkt []byte, n int, signflag byte) (string, int, byte) {
	var sb strings.Builder
	if pkt[0]&signflag != 0 {
		sb.WriteByte('-')
	}
	digits := math.MinInt32
	var tu byte
	for pos := 0; pos < n; pos++ {
		c := pkt[1+pos]
		if pos > 0 && pos < 5 && c&1 != 0 {
			sb.WriteByte('.')
			digits = 0
		}
		ch := bm86xChar(c)
		if pos == 5 && (ch == 'C' || ch == 'F') {
			tu = ch
		} else if ch != 0 {
			sb.WriteByte(ch)
			digits++
		}
	}
	if digits < 0 {
		digits = 0
	}
	return sb.String(), digits, tu
}

func ParseBM86x(b []byte) (Reading, bool) {
	if len(b) != 24 || b[16] != 0x86 || b[17] != 0x86 || b[18] != 0x86 || b[19] != 0x86 {
		return Reading{}, false
	}
	sec, _, _ := bm86xDigits(b[9:], 4, 0)
	isDiode := sec == "diod"
	txt, _, tu := bm86xDigits(b[2:], 6, 0x80)
	ol := strings.Contains(txt, "0L") || strings.Contains(txt, "0.L")
	var r Reading
	r.T = time.Now()
	r.Counts = 50000
	switch {
	case b[8]&0x01 != 0:
		r.Q, r.Unit = QVoltage, "V"
		if isDiode {
			r.Q = QDiode
			r.Flags |= FDiode | FDC
		}
	case b[14]&0x80 != 0:
		r.Q, r.Unit = QCurrent, "A"
	case b[14]&0x20 != 0:
		r.Q, r.Unit = QCapacitance, "F"
	case b[14]&0x10 != 0:
		r.Q, r.Unit = QUnitless, "S"
	case b[15]&0x01 != 0:
		r.Q, r.Unit = QFrequency, "Hz"
	case b[10]&0x01 != 0:
		r.Q, r.Unit = QContinuity, "Ω"
		r.Flags |= FBeep
	case b[15]&0x10 != 0:
		r.Q, r.Unit = QResistance, "Ω"
	case b[15]&0x02 != 0:
		r.Q, r.Unit = QUnitless, "dBm"
	case b[15]&0x80 != 0:
		r.Q, r.Unit = QDuty, "%"
	case b[2]&0x0A != 0 && tu != 0:
		r.Q, r.Unit = QTemperature, "°C"
		if tu == 'F' {
			r.Unit = "°F"
			r.Flags |= FFahrenheit
		}
	default:
		r.Q, r.Unit = QUnitless, ""
	}
	ind1 := b[1]
	if ind1&0xE0 == 0xE0 {
		ind1 &^= 0xE0
	}
	if b[1]&0x10 != 0 {
		r.Flags |= FDC
	}
	if b[2]&0x01 != 0 {
		r.Flags |= FAC
	}
	if b[1]&0x01 != 0 {
		r.Flags |= FAuto
	} else {
		r.Flags |= FManual
	}
	if b[1]&0x08 != 0 {
		r.Flags |= FHold
	}
	if ind1&0x20 != 0 {
		r.Flags |= FMax
	}
	if ind1&0x40 != 0 {
		r.Flags |= FMin
	}
	if b[3]&0x01 != 0 {
		r.Flags |= FRel
	}
	ind15 := b[15]
	if ind15&0x02 != 0 {
		ind15 &^= 0x04
	}
	pfx := ""
	switch {
	case b[15]&0x20 != 0:
		pfx = "M"
	case b[15]&0x40 != 0:
		pfx = "k"
	case ind15&0x04 != 0:
		pfx = "m"
	case b[15]&0x08 != 0:
		pfx = "µ"
	case b[14]&0x40 != 0:
		pfx = "n"
	}
	if !unitAllowsPrefix(r.Unit) {
		pfx = ""
	}
	r.Prefix = pfx
	if ol {
		setOL(&r)
		return r, true
	}
	v, err := strconv.ParseFloat(txt, 64)
	if err != nil {
		return Reading{}, false
	}
	r.Display = txt
	r.Value = v * prefixMult(pfx)
	return r, true
}

// ---------------------------------------------------------------------------
// UNI-T UT181A over its CP2110 USB cable (9600 8N1). Frames: AB CD len16LE
// payload sum16LE, where the sum covers the length field and payload.
// We switch "monitor" on and decode measurement packets (type 0x02).

type UT181A struct {
	buf     []byte
	started bool
	last    time.Time
}

func ut181aFrame(payload []byte) []byte {
	f := []byte{0xAB, 0xCD}
	l := len(payload) + 2
	f = append(f, byte(l), byte(l>>8))
	f = append(f, payload...)
	var s uint16
	for _, c := range f[2:] {
		s += uint16(c)
	}
	return append(f, byte(s), byte(s>>8))
}

// PollRequest re-sends "monitor on" until measurements flow (and every 5 s, harmlessly).
func (d *UT181A) PollRequest() []byte {
	if d.started && time.Since(d.last) < 5*time.Second {
		return nil
	}
	return ut181aFrame([]byte{0x05, 0x01})
}
func (d *UT181A) PollInterval() time.Duration { return 1500 * time.Millisecond }

func (d *UT181A) Feed(b []byte) []Reading {
	d.buf = append(d.buf, b...)
	var out []Reading
	for {
		i := bytes.Index(d.buf, []byte{0xAB, 0xCD})
		if i < 0 {
			if len(d.buf) > 1 {
				d.buf = d.buf[len(d.buf)-1:]
			}
			return out
		}
		d.buf = d.buf[i:]
		if len(d.buf) < 4 {
			return out
		}
		l := int(binary.LittleEndian.Uint16(d.buf[2:4]))
		if l < 3 || l > 1024 {
			d.buf = d.buf[2:]
			continue
		}
		if len(d.buf) < 4+l {
			return out
		}
		frame := d.buf[:4+l]
		var s uint16
		for _, c := range frame[2 : 4+l-2] {
			s += uint16(c)
		}
		if s != binary.LittleEndian.Uint16(frame[4+l-2:]) {
			d.buf = d.buf[2:]
			continue
		}
		payload := frame[4 : 4+l-2]
		if r, ok := ParseUT181A(payload); ok {
			d.started = true
			d.last = time.Now()
			out = append(out, r)
		}
		d.buf = d.buf[4+l:]
	}
}

// ut181aUnit decodes the meter's unit text (Latin-1, e.g. "mVDC", "k~", "°C").
func ut181aUnit(raw []byte) (Quantity, string, string, Flags) {
	t := string(bytes.TrimRight(raw, "\x00 "))
	t = strings.ReplaceAll(t, "\xb0", "°")
	pfx := ""
	if len(t) > 1 {
		switch t[0] {
		case 'p', 'n', 'u', 'm', 'k', 'M', 'G':
			rest := t[1:]
			if rest != "" && !strings.HasPrefix(rest, "in") { // "min" etc. never occur, but be safe
				pfx = string(t[0])
				if pfx == "u" {
					pfx = "µ"
				}
				t = rest
			}
		}
	}
	var fl Flags
	if strings.HasSuffix(t, "ac+dc") {
		fl |= FAC | FDC
		t = strings.TrimSuffix(t, "ac+dc")
	} else if strings.HasSuffix(t, "AC") {
		fl |= FAC
		t = strings.TrimSuffix(t, "AC")
	} else if strings.HasSuffix(t, "DC") {
		fl |= FDC
		t = strings.TrimSuffix(t, "DC")
	}
	switch t {
	case "V":
		return QVoltage, "V", pfx, fl
	case "A":
		return QCurrent, "A", pfx, fl
	case "~":
		return QResistance, "Ω", pfx, fl
	case "F":
		return QCapacitance, "F", pfx, fl
	case "Hz":
		return QFrequency, "Hz", pfx, fl
	case "%":
		return QDuty, "%", "", fl
	case "°C":
		return QTemperature, "°C", "", fl
	case "°F":
		return QTemperature, "°F", "", fl | FFahrenheit
	case "S":
		return QUnitless, "S", pfx, fl
	case "s":
		return QTime, "s", pfx, fl
	case "dBV":
		return QUnitless, "dBV", "", fl
	case "dBm":
		return QUnitless, "dBm", "", fl
	}
	return QUnitless, t, pfx, fl
}

func ParseUT181A(p []byte) (Reading, bool) {
	if len(p) < 1 || p[0] != 0x02 { // measurement packets only
		return Reading{}, false
	}
	p = p[1:]
	if len(p) < 5 {
		return Reading{}, false
	}
	misc1, misc2 := p[0], p[1]
	mode := binary.LittleEndian.Uint16(p[2:4])
	p = p[5:] // skip range byte
	typ := (misc1 & 0x70) >> 4
	var r Reading
	r.T = time.Now()
	r.Counts = 60000
	if misc1&0x80 != 0 {
		r.Flags |= FHold
	}
	if misc2&0x01 != 0 {
		r.Flags |= FAuto
	} else {
		r.Flags |= FManual
	}
	if misc2&0x02 != 0 {
		r.Flags |= FHV
	}
	var val float32
	var prec byte
	var unitRaw []byte
	switch typ {
	case 0, 1, 4: // normal, relative, peak: first value is 4/1/8
		if len(p) < 13 {
			return Reading{}, false
		}
		val = math.Float32frombits(binary.LittleEndian.Uint32(p[0:4]))
		prec = p[4]
		unitRaw = p[5:13]
		if typ == 1 {
			r.Flags |= FRel
		}
		if typ == 4 {
			r.Flags |= FPeakMax
		}
	case 2: // min/max: cur(4/1) max(4/1/4) avg(4/1/4) min(4/1/4) unit(8)
		if len(p) < 5+9+9+9+8 {
			return Reading{}, false
		}
		val = math.Float32frombits(binary.LittleEndian.Uint32(p[0:4]))
		prec = p[4]
		unitRaw = p[32:40]
		r.Flags |= FMin | FMax
	default:
		return Reading{}, false
	}
	q, unit, pfx, fl := ut181aUnit(unitRaw)
	r.Q, r.Unit, r.Prefix = q, unit, pfx
	r.Flags |= fl
	// mode words (libsigrok uni-t-ut181a/protocol.h)
	switch mode {
	case 0x5211, 0x5212: // continuity (short / open)
		if q == QResistance {
			r.Q = QContinuity
			r.Flags |= FBeep
		}
	case 0x6111, 0x6112: // diode (with alarm)
		if q == QVoltage {
			r.Q = QDiode
			r.Flags |= FDiode | FDC
		}
	}
	// precision byte: high nibble = decimals, bit0 = OL+, bit1 = OL-
	if prec&0x03 != 0 || math.IsInf(float64(val), 0) || math.IsNaN(float64(val)) {
		setOL(&r)
		return r, true
	}
	dec := int(prec>>4) & 0x0F
	if dec > 6 {
		dec = 3
	}
	v := float64(val)
	r.Display = strconv.FormatFloat(v, 'f', dec, 64)
	r.Value = v * prefixMult(pfx)
	return r, true
}

// ---------------------------------------------------------------------------
// ANENG / ZOYI / BSIDE / BABATools Bluetooth meters ("Bluetooth DMM",
// service FFF0, notify FFF4). 10- or 11-byte notifications: XOR with a fixed
// key, reverse the bits of every byte, then read 7-segment digits and icons.

var zoyiKey = []byte{0x41, 0x21, 0x73, 0x55, 0xA2, 0xC1, 0x32, 0x71, 0x66, 0xAA, 0x3B, 0xD0, 0xE2, 0xA8, 0x33, 0x14, 0x20, 0x21, 0xAA, 0xBB}

// segment bits as transmitted (after the per-digit leading bit): order "abc..."
// strings from the community decoder, e.g. "1111101" = 0.
var zoyiDigits = map[string]byte{
	"1111101": '0', "0000101": '1', "1011011": '2', "0011111": '3', "0100111": '4',
	"0111110": '5', "1111110": '6', "0010101": '7', "1111111": '8', "0111111": '9',
	"1110111": 'A', "1001100": 'u', "1101010": 't', "1001110": 'o', "1101000": 'L',
	"1111010": 'E', "1110010": 'F', "0000000": ' ', "0000010": '-',
}

var zoyiIcons11 = []string{"LowBattery", "Delta", "BT", "BUZ", "HOLD", "°F", "°C", "DIODE",
	"MAX", "MIN", "%", "AC", "F", "u(F)", "m(F)", "n(F)", "Hz",
	"ohm", "k(ohm)", "M(ohm)", "V", "m(V)", "DC", "A", "AUTO",
	"?7", "u(A)", "m(A)", "?8", "?9", "?10", "?11"}

var zoyiIcons01 = []string{"?1", "HOLD", "FLASH", "BUZ", " ", " ", " ", " ",
	"NANO", "V", "DC", "AC", "F", "DIODE", "A",
	"u(F)", "ohm", "k(ohm)", "M(ohm)", " ", "Hz", "°F", "°C"}

// ZoyiDecode returns (display text, icon set, ok).
func ZoyiDecode(data []byte) (string, map[string]bool, bool) {
	if len(data) < 10 || len(data) > len(zoyiKey) {
		return "", nil, false
	}
	var bits strings.Builder
	for i, c := range data {
		v := bitrev8(c ^ zoyiKey[i])
		for k := 7; k >= 0; k-- {
			if v&(1<<uint(k)) != 0 {
				bits.WriteByte('1')
			} else {
				bits.WriteByte('0')
			}
		}
	}
	s := bits.String()
	if len(s) < 87 {
		// 10-byte packets carry 80 bits; pad so the icon slice is valid
		s += strings.Repeat("0", 87-len(s))
	}
	typeID := s[16:18]
	var num strings.Builder
	for g := 0; g < 4; g++ {
		grp := s[28+g*8 : 36+g*8]
		if g == 0 && grp[0] == '1' {
			num.WriteByte('-')
		}
		if g > 0 && grp[0] == '1' {
			num.WriteByte('.')
		}
		ch, ok := zoyiDigits[grp[1:]]
		if !ok {
			return "", nil, false
		}
		num.WriteByte(ch)
	}
	icBits := s[24:28] + s[60:87]
	names := zoyiIcons11
	if typeID == "01" {
		names = zoyiIcons01
	} else if typeID != "11" {
		return "", nil, false
	}
	icons := map[string]bool{}
	for i := 0; i < len(icBits) && i < len(names); i++ {
		if icBits[i] == '1' {
			icons[names[i]] = true
		}
	}
	return num.String(), icons, true
}

type ZoyiBLE struct{}

func (ZoyiBLE) Feed(b []byte) []Reading { return ZoyiBLE{}.FeedFrame(b) }

func (ZoyiBLE) FeedFrame(b []byte) []Reading {
	if r, ok := ParseZoyi(b); ok {
		return []Reading{r}
	}
	return nil
}

func ParseZoyi(b []byte) (Reading, bool) {
	disp, ic, ok := ZoyiDecode(b)
	if !ok {
		return Reading{}, false
	}
	var r Reading
	r.T = time.Now()
	r.Counts = 6000
	pfx := ""
	switch {
	case ic["V"]:
		r.Q, r.Unit = QVoltage, "V"
		if ic["m(V)"] {
			pfx = "m"
		}
	case ic["A"]:
		r.Q, r.Unit = QCurrent, "A"
		if ic["m(A)"] {
			pfx = "m"
		} else if ic["u(A)"] {
			pfx = "µ"
		}
	case ic["ohm"]:
		r.Q, r.Unit = QResistance, "Ω"
		if ic["k(ohm)"] {
			pfx = "k"
		} else if ic["M(ohm)"] {
			pfx = "M"
		}
	case ic["F"]:
		r.Q, r.Unit = QCapacitance, "F"
		switch {
		case ic["n(F)"] || ic["NANO"]:
			pfx = "n"
		case ic["u(F)"]:
			pfx = "µ"
		case ic["m(F)"]:
			pfx = "m"
		}
	case ic["Hz"]:
		r.Q, r.Unit = QFrequency, "Hz"
	case ic["%"]:
		r.Q, r.Unit = QDuty, "%"
	case ic["°C"]:
		r.Q, r.Unit = QTemperature, "°C"
	case ic["°F"]:
		r.Q, r.Unit = QTemperature, "°F"
		r.Flags |= FFahrenheit
	default:
		r.Q, r.Unit = QUnitless, ""
	}
	if ic["DIODE"] {
		r.Q, r.Unit = QDiode, "V"
		r.Flags |= FDiode | FDC
	}
	if ic["BUZ"] && r.Q == QResistance {
		r.Q = QContinuity
		r.Flags |= FBeep
	}
	if ic["AC"] {
		r.Flags |= FAC
	}
	if ic["DC"] {
		r.Flags |= FDC
	}
	if ic["AUTO"] {
		r.Flags |= FAuto
	} else {
		r.Flags |= FManual
	}
	if ic["HOLD"] {
		r.Flags |= FHold
	}
	if ic["Delta"] {
		r.Flags |= FRel
	}
	if ic["MAX"] {
		r.Flags |= FMax
	}
	if ic["MIN"] {
		r.Flags |= FMin
	}
	if ic["LowBattery"] {
		r.Flags |= FLowBat
	}
	r.Prefix = pfx
	t := strings.TrimSpace(disp)
	if strings.Contains(t, "L") && (strings.Contains(t, "0") || strings.Contains(t, "o")) {
		setOL(&r)
		return r, true
	}
	v, err := strconv.ParseFloat(strings.ReplaceAll(t, " ", ""), 64)
	if err != nil {
		if t == "" || strings.Trim(t, "-") == "" {
			return Reading{}, false
		}
		// text displays like "Auto", "LEAd" — keep as a non-numeric annunciation
		return Reading{}, false
	}
	r.Display = strings.ReplaceAll(t, " ", "")
	r.Value = v * prefixMult(pfx)
	return r, true
}
