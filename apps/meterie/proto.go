package meterie

// proto.go: byte-stream decoders for multimeter protocols.
//
// Every decoder is pure Go and platform independent so it can be unit tested.
// A decoder is fed raw bytes from whatever transport (COM port, HID-UART
// bridge, BLE notifications ...) and emits complete readings.
//
// Protocol references: sigrok wiki / libsigrok serial-dmm notes, vendor
// documents, and community reverse-engineering (credited in README.md).

import (
	"bytes"
	"math"
	"strconv"
	"strings"
	"time"
)

// Decoder turns a byte stream into readings.
type Decoder interface {
	Feed(b []byte) []Reading
}

// Poller is implemented by request/response protocols.
type Poller interface {
	PollRequest() []byte
	PollInterval() time.Duration
}

// FrameDecoder is used for transports that deliver whole frames (BLE notifications).
type FrameDecoder interface {
	FeedFrame(b []byte) []Reading
}

// ---------------------------------------------------------------------------
// Seven-segment helpers (shared with the camera OCR).
// Standard mask: bit0=a, bit1=b, bit2=c, bit3=d, bit4=e, bit5=f, bit6=g.

var segChars = map[byte]byte{
	0x3F: '0', 0x06: '1', 0x5B: '2', 0x4F: '3', 0x66: '4', 0x6D: '5', 0x7D: '6',
	0x07: '7', 0x27: '7', 0x7F: '8', 0x6F: '9', 0x67: '9', 0x7C: '6', // 6 without top on some LCDs is 'b' – treat as b below
	0x00: ' ', 0x40: '-', 0x38: 'L', 0x79: 'E', 0x71: 'F', 0x77: 'A', 0x39: 'C',
	0x58: 'c', 0x5E: 'd', 0x76: 'H', 0x54: 'n', 0x5C: 'o', 0x73: 'P', 0x50: 'r',
	0x78: 't', 0x3E: 'U', 0x1C: 'u', 0x74: 'h', 0x1E: 'J', 0x6E: 'y', 0x08: '_',
}

func init() {
	segChars[0x7C] = 'b'
}

// charSegs is the inverse, used to draw characters on the virtual LCD.
var charSegs = map[byte]byte{
	'0': 0x3F, '1': 0x06, '2': 0x5B, '3': 0x4F, '4': 0x66, '5': 0x6D, '6': 0x7D, '7': 0x07,
	'8': 0x7F, '9': 0x6F, ' ': 0x00, '-': 0x40, 'L': 0x38, 'E': 0x79, 'F': 0x71, 'A': 0x77,
	'C': 0x39, 'c': 0x58, 'd': 0x5E, 'H': 0x76, 'n': 0x54, 'o': 0x5C, 'O': 0x3F, 'P': 0x73,
	'r': 0x50, 't': 0x78, 'U': 0x3E, 'u': 0x1C, 'h': 0x74, 'J': 0x1E, 'y': 0x6E, '_': 0x08,
	'b': 0x7C, 'S': 0x6D, 'I': 0x06, 'l': 0x30,
}

func segToChar(m byte) (byte, bool) {
	c, ok := segChars[m&0x7F]
	return c, ok
}

// ---------------------------------------------------------------------------
// Fortune Semiconductor FS9721_LP3 (and pin-compatible FS9711 / CS7721).
// 14 bytes, 2400 8N1, TX only. High nibble of byte i is i+1; low nibble = LCD bits.
// Used by: HoldPeak HP-90EPC, TekPower TP4000ZC, Digitek DT4000ZC, UNI-T UT60A/E,
// V&A VA18B, Mastech MS8250B, PeakTech 3330/3415, Voltcraft VC-820/VC-840 ...

type FS9721 struct {
	buf     []byte
	TempBit int // which bit of byte 14 means °C (-1 = none)
	Counts  int
}

func NewFS9721(tempBit, counts int) *FS9721 { return &FS9721{TempBit: tempBit, Counts: counts} }

func (d *FS9721) Feed(b []byte) []Reading {
	var out []Reading
	for _, c := range b {
		idx := int(c >> 4)
		if idx == 1 {
			d.buf = d.buf[:0]
			d.buf = append(d.buf, c)
		} else if len(d.buf) > 0 && idx == len(d.buf)+1 {
			d.buf = append(d.buf, c)
		} else {
			d.buf = d.buf[:0]
			continue
		}
		if len(d.buf) == 14 {
			var p [14]byte
			copy(p[:], d.buf)
			if r, ok := ParseFS9721(p, d.TempBit, d.Counts); ok {
				out = append(out, r)
			}
			d.buf = d.buf[:0]
		}
	}
	return out
}

// fs9721 segment bit order -> standard mask
func fs9721Seg(hi, lo byte) byte {
	raw := ((hi & 7) << 4) | (lo & 0x0F)
	var m byte
	if raw&(1<<4) != 0 {
		m |= 1 << 0 // a
	}
	if raw&(1<<0) != 0 {
		m |= 1 << 1 // b
	}
	if raw&(1<<2) != 0 {
		m |= 1 << 2 // c
	}
	if raw&(1<<3) != 0 {
		m |= 1 << 3 // d
	}
	if raw&(1<<6) != 0 {
		m |= 1 << 4 // e
	}
	if raw&(1<<5) != 0 {
		m |= 1 << 5 // f
	}
	if raw&(1<<1) != 0 {
		m |= 1 << 6 // g
	}
	return m
}

// ParseFS9721 decodes one validated 14-byte packet.
func ParseFS9721(b [14]byte, tempBit, counts int) (Reading, bool) {
	for i := 0; i < 14; i++ {
		if int(b[i]>>4) != i+1 {
			return Reading{}, false
		}
	}
	var r Reading
	r.T = time.Now()
	if counts <= 0 {
		counts = 4000
	}
	r.Counts = counts
	// flags byte 1
	if b[0]&8 != 0 {
		r.Flags |= FAC
	}
	if b[0]&4 != 0 {
		r.Flags |= FDC
	}
	if b[0]&2 != 0 {
		r.Flags |= FAuto
	} else {
		r.Flags |= FManual
	}
	neg := b[1]&8 != 0
	var disp []byte
	ol := false
	dpAfter := -1
	for i := 0; i < 4; i++ {
		hi, lo := b[1+2*i], b[2+2*i]
		if i > 0 && hi&8 != 0 {
			dpAfter = i - 1
		}
		ch, ok := segToChar(fs9721Seg(hi, lo))
		if !ok {
			ch = '?'
		}
		if ch == 'L' {
			ol = true
		}
		disp = append(disp, ch)
	}
	// build display & numeric value
	var sb strings.Builder
	if neg {
		sb.WriteByte('-')
	}
	for i, ch := range disp {
		sb.WriteByte(ch)
		if i == dpAfter {
			sb.WriteByte('.')
		}
	}
	display := sb.String()
	// unit bytes
	b9, b10, b11, b12, b13 := b[9]&0xF, b[10]&0xF, b[11]&0xF, b[12]&0xF, b[13]&0xF
	prefix := ""
	if b9&8 != 0 {
		prefix = "µ"
	}
	if b9&4 != 0 {
		prefix = "n"
	}
	if b9&2 != 0 {
		prefix = "k"
	}
	if b10&8 != 0 {
		prefix = "m"
	}
	if b10&2 != 0 {
		prefix = "M"
	}
	if b9&1 != 0 {
		r.Flags |= FDiode
	}
	if b10&1 != 0 {
		r.Flags |= FBeep
	}
	if b11&2 != 0 {
		r.Flags |= FRel
	}
	if b11&1 != 0 {
		r.Flags |= FHold
	}
	if b12&1 != 0 {
		r.Flags |= FLowBat
	}
	switch {
	case b12&4 != 0:
		r.Q, r.Unit = QVoltage, "V"
		if r.Flags.Has(FDiode) {
			r.Q = QDiode
		}
	case b12&8 != 0:
		r.Q, r.Unit = QCurrent, "A"
	case b11&4 != 0:
		r.Q, r.Unit = QResistance, "Ω"
		if r.Flags.Has(FBeep) {
			r.Q = QContinuity
		}
	case b11&8 != 0:
		r.Q, r.Unit = QCapacitance, "F"
	case b12&2 != 0:
		r.Q, r.Unit = QFrequency, "Hz"
	case b10&4 != 0:
		r.Q, r.Unit = QDuty, "%"
	case tempBit >= 0 && b13&(1<<uint(tempBit)) != 0, tempBit == -2 && b13 != 0:
		r.Q, r.Unit = QTemperature, "°C"
	default:
		r.Q, r.Unit = QUnitless, ""
	}
	r.Prefix = prefix
	if ol {
		r.Flags |= FOL
		r.Value = math.NaN()
		r.Display = "OL"
		if strings.Contains(display, "0L") || strings.Contains(display, "0.L") {
			r.Display = "0.L"
		}
		return r, true
	}
	num := strings.ReplaceAll(display, " ", "")
	v, err := strconv.ParseFloat(num, 64)
	if err != nil {
		return Reading{}, false
	}
	r.Value = v * prefixMult(prefix)
	r.Display = display
	return r, true
}

// ---------------------------------------------------------------------------
// Fortune Semiconductor FS9922-DMM3/DMM4. 14 bytes ending CR LF, 2400 8N1.
// Used by: UNI-T UT61B/C/D, Voltcraft VC-830, Victor 70C/86C (obfuscated),
// early OWON B35 (over BLE), many 6000-count meters.

type FS9922 struct {
	buf    []byte
	Counts int
}

func NewFS9922(counts int) *FS9922 { return &FS9922{Counts: counts} }

func (d *FS9922) Feed(b []byte) []Reading {
	d.buf = append(d.buf, b...)
	var out []Reading
	for {
		i := bytes.Index(d.buf, []byte{'\r', '\n'})
		if i < 0 {
			if len(d.buf) > 256 {
				d.buf = d.buf[len(d.buf)-32:]
			}
			return out
		}
		end := i + 2
		if end >= 14 {
			pkt := d.buf[end-14 : end]
			if r, ok := ParseFS9922(pkt, d.Counts); ok {
				out = append(out, r)
			}
		}
		d.buf = d.buf[end:]
	}
}

func ParseFS9922(b []byte, counts int) (Reading, bool) {
	if len(b) != 14 || (b[0] != '+' && b[0] != '-') || b[5] != ' ' || b[12] != '\r' || b[13] != '\n' {
		return Reading{}, false
	}
	var r Reading
	r.T = time.Now()
	if counts <= 0 {
		counts = 6000
	}
	r.Counts = counts
	digits := string(b[1:5])
	ol := digits == "?0:?"
	if !ol {
		for _, c := range b[1:5] {
			if c < '0' || c > '9' {
				return Reading{}, false
			}
		}
	}
	dp := -1
	switch b[6] {
	case '0':
		dp = -1
	case '1':
		dp = 0
	case '2':
		dp = 1
	case '4':
		dp = 2
	default:
		return Reading{}, false
	}
	s7, s8, s9, s10 := b[7], b[8], b[9], b[10]
	if s7&(1<<1) != 0 {
		r.Flags |= FHold
	}
	if s7&(1<<2) != 0 {
		r.Flags |= FRel
	}
	if s7&(1<<3) != 0 {
		r.Flags |= FAC
	}
	if s7&(1<<4) != 0 {
		r.Flags |= FDC
	}
	if s7&(1<<5) != 0 {
		r.Flags |= FAuto
	} else {
		r.Flags |= FManual
	}
	if s8&(1<<5) != 0 {
		r.Flags |= FMax
	}
	if s8&(1<<4) != 0 {
		r.Flags |= FMin
	}
	if s8&(1<<2) != 0 {
		r.Flags |= FLowBat
	}
	prefix := ""
	if s8&(1<<1) != 0 {
		prefix = "n"
	}
	if s9&(1<<7) != 0 {
		prefix = "µ"
	}
	if s9&(1<<6) != 0 {
		prefix = "m"
	}
	if s9&(1<<5) != 0 {
		prefix = "k"
	}
	if s9&(1<<4) != 0 {
		prefix = "M"
	}
	if s9&(1<<3) != 0 {
		r.Flags |= FBeep
	}
	if s9&(1<<2) != 0 {
		r.Flags |= FDiode
	}
	switch {
	case s10&(1<<7) != 0:
		r.Q, r.Unit = QVoltage, "V"
		if r.Flags.Has(FDiode) {
			r.Q = QDiode
		}
	case s10&(1<<6) != 0:
		r.Q, r.Unit = QCurrent, "A"
	case s10&(1<<5) != 0:
		r.Q, r.Unit = QResistance, "Ω"
		if r.Flags.Has(FBeep) {
			r.Q = QContinuity
		}
	case s10&(1<<4) != 0:
		r.Q, r.Unit = QHFE, ""
	case s10&(1<<3) != 0:
		r.Q, r.Unit = QFrequency, "Hz"
	case s10&(1<<2) != 0:
		r.Q, r.Unit = QCapacitance, "F"
	case s10&(1<<1) != 0:
		r.Q, r.Unit = QTemperature, "°C"
	case s10&(1<<0) != 0:
		r.Q, r.Unit = QTemperature, "°F"
		r.Flags |= FFahrenheit
	case s9&(1<<1) != 0:
		r.Q, r.Unit = QDuty, "%"
	default:
		r.Q, r.Unit = QUnitless, ""
	}
	if s9&(1<<1) != 0 && r.Q == QFrequency {
		r.Q, r.Unit = QDuty, "%"
	}
	r.Prefix = prefix
	if ol {
		r.Flags |= FOL
		r.Value = math.NaN()
		r.Display = "OL"
		return r, true
	}
	var sb strings.Builder
	if b[0] == '-' {
		sb.WriteByte('-')
	}
	for i, c := range b[1:5] {
		sb.WriteByte(c)
		if i == dp {
			sb.WriteByte('.')
		}
	}
	r.Display = sb.String()
	v, err := strconv.ParseFloat(r.Display, 64)
	if err != nil {
		return Reading{}, false
	}
	r.Value = v * prefixMult(prefix)
	return r, true
}

// ---------------------------------------------------------------------------
// Cyrustek ES51922 (UNI-T UT61E). 14 bytes ending CR LF, 19200 7O1.

type ES51922 struct{ buf []byte }

func (d *ES51922) Feed(b []byte) []Reading {
	for _, c := range b {
		d.buf = append(d.buf, c&0x7F)
	}
	var out []Reading
	for {
		i := bytes.Index(d.buf, []byte{'\r', '\n'})
		if i < 0 {
			if len(d.buf) > 256 {
				d.buf = d.buf[len(d.buf)-32:]
			}
			return out
		}
		end := i + 2
		if end >= 14 {
			if r, ok := ParseES51922(d.buf[end-14 : end]); ok {
				out = append(out, r)
			}
		}
		d.buf = d.buf[end:]
	}
}

type rangeDef struct {
	mult   float64 // base units per count
	prefix string  // display prefix
}

func pow10(n int) float64 { return math.Pow(10, float64(n)) }

func es51922Range(fn byte, rng int) (Quantity, string, rangeDef, bool) {
	switch fn {
	case 0x3B: // voltage
		t := []rangeDef{{1e-4, ""}, {1e-3, ""}, {1e-2, ""}, {1e-1, ""}, {1e-5, "m"}}
		if rng < len(t) {
			return QVoltage, "V", t[rng], true
		}
	case 0x3D: // µA
		t := []rangeDef{{1e-8, "µ"}, {1e-7, "µ"}}
		if rng < len(t) {
			return QCurrent, "A", t[rng], true
		}
	case 0x3F: // mA
		t := []rangeDef{{1e-6, "m"}, {1e-5, "m"}}
		if rng < len(t) {
			return QCurrent, "A", t[rng], true
		}
	case 0x30, 0x39: // A
		return QCurrent, "A", rangeDef{1e-3, ""}, true
	case 0x33: // Ω
		if rng <= 6 {
			p := []string{"", "k", "k", "k", "M", "M", "M"}[rng]
			return QResistance, "Ω", rangeDef{pow10(rng - 2), p}, true
		}
	case 0x35:
		return QContinuity, "Ω", rangeDef{1e-2, ""}, true
	case 0x31:
		return QDiode, "V", rangeDef{1e-4, ""}, true
	case 0x32: // frequency
		if rng <= 7 {
			p := []string{"", "", "k", "k", "k", "M", "M", "M"}[rng]
			return QFrequency, "Hz", rangeDef{pow10(rng - 3), p}, true
		}
	case 0x36: // capacitance
		if rng <= 7 {
			p := []string{"n", "n", "µ", "µ", "µ", "m", "m", "m"}[rng]
			return QCapacitance, "F", rangeDef{pow10(rng - 12), p}, true
		}
	case 0x34:
		return QTemperature, "°C", rangeDef{1e-1, ""}, true
	}
	return QNone, "", rangeDef{}, false
}

// displayFromCount renders a count the way the LCD shows it.
func displayFromCount(count int, mult float64, prefix string, neg bool, digits int) string {
	scale := mult / prefixMult(prefix)
	dec := int(math.Round(-math.Log10(scale)))
	if dec < 0 {
		dec = 0
	}
	s := strconv.Itoa(count)
	for len(s) < digits {
		s = "0" + s
	}
	if dec > 0 && dec < len(s) {
		s = s[:len(s)-dec] + "." + s[len(s)-dec:]
	} else if dec >= len(s) {
		s = "0." + strings.Repeat("0", dec-len(s)) + s
	}
	// trim superfluous leading zeros like the LCD (keep one before the point)
	for len(s) > 1 && s[0] == '0' && s[1] != '.' {
		s = s[1:]
	}
	if neg {
		s = "-" + s
	}
	return s
}

func ParseES51922(b []byte) (Reading, bool) {
	if len(b) != 14 || b[12] != '\r' || b[13] != '\n' {
		return Reading{}, false
	}
	for i := 0; i < 12; i++ {
		if b[i]&0xF0 != 0x30 {
			return Reading{}, false
		}
	}
	rng := int(b[0] & 0x07)
	fn := b[6]
	status, o1, o2, o3, o4 := b[7], b[8], b[9], b[10], b[11]
	var r Reading
	r.T = time.Now()
	r.Counts = 22000
	q, unit, rd, ok := es51922Range(fn, rng)
	if !ok {
		return Reading{}, false
	}
	if fn == 0x32 && status&8 != 0 { // duty cycle
		q, unit, rd = QDuty, "%", rangeDef{0.1, ""}
	}
	r.Q, r.Unit, r.Prefix = q, unit, rd.prefix
	if status&2 != 0 {
		r.Flags |= FLowBat
	}
	if o1&8 != 0 {
		r.Flags |= FMax
	}
	if o1&4 != 0 {
		r.Flags |= FMin
	}
	if o1&2 != 0 {
		r.Flags |= FRel
	}
	if o2&8 != 0 {
		r.Flags |= FUnderload
	}
	if o2&4 != 0 {
		r.Flags |= FPeakMax
	}
	if o2&2 != 0 {
		r.Flags |= FPeakMin
	}
	if o3&8 != 0 {
		r.Flags |= FDC
	}
	if o3&4 != 0 {
		r.Flags |= FAC
	}
	if o3&2 != 0 {
		r.Flags |= FAuto
	} else {
		r.Flags |= FManual
	}
	if o4&2 != 0 {
		r.Flags |= FHold
	}
	if q == QDiode {
		r.Flags |= FDiode
	}
	if q == QContinuity {
		r.Flags |= FBeep
	}
	if status&1 != 0 {
		r.Flags |= FOL
		r.Value = math.NaN()
		r.Display = "OL"
		return r, true
	}
	cnt := 0
	for _, c := range b[1:6] {
		if c < '0' || c > '9' {
			return Reading{}, false
		}
		cnt = cnt*10 + int(c-'0')
	}
	neg := status&4 != 0
	r.Value = float64(cnt) * rd.mult
	if neg {
		r.Value = -r.Value
	}
	r.Display = displayFromCount(cnt, rd.mult, rd.prefix, neg, 5)
	return r, true
}

// ---------------------------------------------------------------------------
// UNI-T UT61E+ / UT61B+ / UT61D+ / UT161 series (CP2110 USB HID bridge, 9600 8N1).
// Request: AB CD 03 5E 01 D9. Reply: AB CD 10 mode range disp[7] bar[2] f1 f2 f3 sumH sumL.

type UT61EPlus struct{ buf []byte }

var ut61epRequest = []byte{0xAB, 0xCD, 0x03, 0x5E, 0x01, 0xD9}

func (d *UT61EPlus) PollRequest() []byte         { return ut61epRequest }
func (d *UT61EPlus) PollInterval() time.Duration { return 250 * time.Millisecond }

func (d *UT61EPlus) Feed(b []byte) []Reading {
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
		if len(d.buf) < 3 {
			return out
		}
		n := int(d.buf[2])
		total := 3 + n
		if n < 3 || n > 64 {
			d.buf = d.buf[2:]
			continue
		}
		if len(d.buf) < total {
			return out
		}
		frame := d.buf[:total]
		if r, ok := ParseUT61EPlus(frame); ok {
			out = append(out, r)
			d.buf = d.buf[total:]
		} else {
			d.buf = d.buf[2:]
		}
	}
}

type utMode struct {
	q      Quantity
	unit   string
	ac, dc bool
	pfx    []string // display prefix by range (last entry repeats)
}

var ut61epModes = []utMode{
	0:  {QVoltage, "V", true, false, []string{""}},
	1:  {QVoltage, "V", true, false, []string{"m"}},
	2:  {QVoltage, "V", false, true, []string{""}},
	3:  {QVoltage, "V", false, true, []string{"m"}},
	4:  {QFrequency, "Hz", false, false, []string{"", "", "k", "k", "k", "M", "M", "M"}},
	5:  {QDuty, "%", false, false, []string{""}},
	6:  {QResistance, "Ω", false, false, []string{"", "k", "k", "k", "M", "M", "M"}},
	7:  {QContinuity, "Ω", false, false, []string{""}},
	8:  {QDiode, "V", false, false, []string{""}},
	9:  {QCapacitance, "F", false, false, []string{"n", "n", "µ", "µ", "µ", "m", "m", "m"}},
	10: {QTemperature, "°C", false, false, []string{""}},
	11: {QTemperature, "°F", false, false, []string{""}},
	12: {QCurrent, "A", false, true, []string{"µ"}},
	13: {QCurrent, "A", true, false, []string{"µ"}},
	14: {QCurrent, "A", false, true, []string{"m"}},
	15: {QCurrent, "A", true, false, []string{"m"}},
	16: {QCurrent, "A", false, true, []string{""}},
	17: {QCurrent, "A", true, false, []string{""}},
	18: {QHFE, "", false, false, []string{""}},
	19: {QVoltage, "V", true, false, []string{""}},  // Live
	20: {QUnitless, "", false, false, []string{""}}, // NCV
	21: {QVoltage, "V", true, false, []string{""}},  // LoZ V
	22: {QCurrent, "A", true, false, []string{""}},
	23: {QCurrent, "A", false, true, []string{""}},
	24: {QVoltage, "V", true, false, []string{""}}, // LPF
	25: {QVoltage, "V", true, true, []string{""}},  // AC/DC
	26: {QVoltage, "V", true, false, []string{""}}, // LPF
	27: {QCurrent, "A", true, true, []string{""}},  // AC+DC
	28: {QCurrent, "A", true, false, []string{""}}, // LPF
	29: {QCurrent, "A", true, true, []string{""}},  // AC+DC2
	30: {QCurrent, "A", true, false, []string{""}}, // INRUSH
}

func ParseUT61EPlus(f []byte) (Reading, bool) {
	if len(f) < 19 || f[0] != 0xAB || f[1] != 0xCD {
		return Reading{}, false
	}
	n := int(f[2])
	if len(f) != 3+n || n < 16 {
		return Reading{}, false
	}
	sum := 0
	for _, c := range f[:len(f)-2] {
		sum += int(c)
	}
	if uint16(sum) != uint16(f[len(f)-2])<<8|uint16(f[len(f)-1]) {
		return Reading{}, false
	}
	mode := int(f[3])
	rng := int(f[4] & 0x0F)
	disp := string(f[5:12])
	f1, f2, f3 := f[14], f[15], f[16]
	if mode < 0 || mode >= len(ut61epModes) {
		return Reading{}, false
	}
	m := ut61epModes[mode]
	var r Reading
	r.T = time.Now()
	r.Counts = 22000
	r.Q, r.Unit = m.q, m.unit
	pfx := m.pfx[len(m.pfx)-1]
	if rng < len(m.pfx) {
		pfx = m.pfx[rng]
	}
	r.Prefix = pfx
	if m.ac {
		r.Flags |= FAC
	}
	if m.dc || f3&8 != 0 {
		r.Flags |= FDC
	}
	if f1&8 != 0 {
		r.Flags |= FMax
	}
	if f1&4 != 0 {
		r.Flags |= FMin
	}
	if f1&2 != 0 {
		r.Flags |= FHold
	}
	if f1&1 != 0 {
		r.Flags |= FRel
	}
	if f2&4 == 0 {
		r.Flags |= FAuto
	} else {
		r.Flags |= FManual
	}
	if f2&2 != 0 {
		r.Flags |= FLowBat
	}
	if f2&1 != 0 {
		r.Flags |= FHV
	}
	if f3&4 != 0 {
		r.Flags |= FPeakMax
	}
	if f3&2 != 0 {
		r.Flags |= FPeakMin
	}
	if m.q == QDiode {
		r.Flags |= FDiode
	}
	if m.q == QContinuity {
		r.Flags |= FBeep
	}
	if m.unit == "°F" {
		r.Flags |= FFahrenheit
	}
	clean := strings.ReplaceAll(disp, " ", "")
	clean = strings.TrimRight(clean, "\x00")
	if strings.Contains(strings.ToUpper(clean), "OL") {
		r.Flags |= FOL
		r.Value = math.NaN()
		r.Display = "OL"
		return r, true
	}
	v, err := strconv.ParseFloat(clean, 64)
	if err != nil {
		return Reading{}, false
	}
	r.Display = clean
	r.Value = v * prefixMult(pfx)
	return r, true
}

// ---------------------------------------------------------------------------
// OWON Bluetooth LE meters (B35T+, B41T+, OW18E, OW16 ...): each notification on
// characteristic 0xFFF4 is three little-endian uint16 words.
// Older B35T (FS9922 chip) instead streams FS9922 packets; handled automatically.

type OwonBLE struct {
	legacy *FS9922
}

func (d *OwonBLE) Feed(b []byte) []Reading { return d.FeedFrame(b) }

func (d *OwonBLE) FeedFrame(b []byte) []Reading {
	if len(b) == 6 {
		if r, ok := ParseOwon6(b); ok {
			return []Reading{r}
		}
		return nil
	}
	if d.legacy == nil {
		d.legacy = NewFS9922(6000)
	}
	return d.legacy.Feed(b)
}

func ParseOwon6(b []byte) (Reading, bool) {
	if len(b) != 6 {
		return Reading{}, false
	}
	w0 := uint16(b[0]) | uint16(b[1])<<8
	w1 := uint16(b[2]) | uint16(b[3])<<8
	w2 := uint16(b[4]) | uint16(b[5])<<8
	fn := int(w0>>6) & 0x0F
	scale := int(w0>>3) & 7
	dec := int(w0) & 7
	var r Reading
	r.T = time.Now()
	r.Counts = 6000
	switch fn {
	case 0:
		r.Q, r.Unit, r.Flags = QVoltage, "V", FDC
	case 1:
		r.Q, r.Unit, r.Flags = QVoltage, "V", FAC
	case 2:
		r.Q, r.Unit, r.Flags = QCurrent, "A", FDC
	case 3:
		r.Q, r.Unit, r.Flags = QCurrent, "A", FAC
	case 4:
		r.Q, r.Unit = QResistance, "Ω"
	case 5:
		r.Q, r.Unit = QCapacitance, "F"
	case 6:
		r.Q, r.Unit = QFrequency, "Hz"
	case 7:
		r.Q, r.Unit = QDuty, "%"
	case 8:
		r.Q, r.Unit = QTemperature, "°C"
	case 9:
		r.Q, r.Unit = QTemperature, "°F"
		r.Flags |= FFahrenheit
	case 10:
		r.Q, r.Unit = QDiode, "V"
		r.Flags |= FDiode
	case 11:
		r.Q, r.Unit = QContinuity, "Ω"
		r.Flags |= FBeep
	case 12:
		r.Q, r.Unit = QHFE, ""
	default:
		r.Q, r.Unit = QUnitless, ""
	}
	pfx := map[int]string{1: "n", 2: "µ", 3: "m", 4: "", 5: "k", 6: "M"}[scale]
	if !unitAllowsPrefix(r.Unit) {
		pfx = ""
	}
	r.Prefix = pfx
	if w1&1 != 0 {
		r.Flags |= FHold
	}
	if w1&2 != 0 {
		r.Flags |= FRel
	}
	if w1&4 != 0 {
		r.Flags |= FAuto
	} else {
		r.Flags |= FManual
	}
	if w1&8 != 0 {
		r.Flags |= FLowBat
	}
	if w1&16 != 0 {
		r.Flags |= FMin
	}
	if w1&32 != 0 {
		r.Flags |= FMax
	}
	if dec == 7 {
		r.Flags |= FOL
		r.Value = math.NaN()
		r.Display = "OL"
		return r, true
	}
	mag := int(w2 & 0x7FFF)
	neg := w2&0x8000 != 0
	v := float64(mag) / pow10(dec)
	if neg {
		v = -v
	}
	r.Display = strconv.FormatFloat(v, 'f', dec, 64)
	r.Value = v * prefixMult(pfx)
	return r, true
}

// ---------------------------------------------------------------------------
// Fluke 287/289 (IR cable, 115200 8N1): "QM\r" -> "0\r" + "value,unit,state,attr\r"
// Fluke 187/189 & 87-IV/89-IV (9600 8N1): "QM\r" -> "0\r" + "QM,+1.234 VDC\r"

type FlukeQM struct {
	buf  []byte
	Is28 bool
}

func (d *FlukeQM) PollRequest() []byte         { return []byte("QM\r") }
func (d *FlukeQM) PollInterval() time.Duration { return 250 * time.Millisecond }

func (d *FlukeQM) Feed(b []byte) []Reading {
	d.buf = append(d.buf, b...)
	var out []Reading
	for {
		i := bytes.IndexAny(d.buf, "\r\n")
		if i < 0 {
			if len(d.buf) > 512 {
				d.buf = d.buf[:0]
			}
			return out
		}
		line := strings.TrimSpace(string(d.buf[:i]))
		d.buf = d.buf[i+1:]
		if line == "" || len(line) == 1 { // ack digits
			continue
		}
		if r, ok := ParseFlukeLine(line); ok {
			out = append(out, r)
		}
	}
}

func ParseFlukeLine(line string) (Reading, bool) {
	var r Reading
	r.T = time.Now()
	r.Counts = 50000
	if strings.HasPrefix(line, "QM,") { // 18x / 8x-IV format
		body := strings.TrimSpace(line[3:])
		if strings.Contains(strings.ToLower(body), "out of range") || strings.Contains(body, "OL") {
			unit := strings.TrimSpace(strings.TrimPrefix(strings.ToLower(body), "out of range"))
			r.Q, r.Unit, r.Prefix, r.Flags = flukeUnit18x(unit)
			r.Flags |= FOL
			r.Value = math.NaN()
			r.Display = "OL"
			return r, true
		}
		sp := strings.IndexAny(body, " \t")
		if sp < 0 {
			return r, false
		}
		num, unit := body[:sp], strings.TrimSpace(body[sp:])
		v, err := strconv.ParseFloat(num, 64)
		if err != nil {
			return r, false
		}
		r.Q, r.Unit, r.Prefix, r.Flags = flukeUnit18x(unit)
		r.Display = strings.TrimPrefix(num, "+")
		r.Value = v * prefixMult(r.Prefix)
		return r, true
	}
	parts := strings.Split(line, ",")
	if len(parts) < 2 {
		return r, false
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return r, false
	}
	unit := strings.ToUpper(strings.TrimSpace(parts[1]))
	state := ""
	if len(parts) > 2 {
		state = strings.ToUpper(strings.TrimSpace(parts[2]))
	}
	switch {
	case strings.HasPrefix(unit, "V") || strings.HasPrefix(unit, "MV"):
		r.Q, r.Unit = QVoltage, "V"
	case strings.HasPrefix(unit, "A") || strings.HasPrefix(unit, "MA") || strings.HasPrefix(unit, "UA"):
		r.Q, r.Unit = QCurrent, "A"
	case strings.HasPrefix(unit, "OHM"):
		r.Q, r.Unit = QResistance, "Ω"
	case unit == "F":
		r.Q, r.Unit = QCapacitance, "F"
	case strings.HasPrefix(unit, "HZ"):
		r.Q, r.Unit = QFrequency, "Hz"
	case strings.HasPrefix(unit, "CEL"):
		r.Q, r.Unit = QTemperature, "°C"
	case strings.HasPrefix(unit, "FAR"):
		r.Q, r.Unit = QTemperature, "°F"
		r.Flags |= FFahrenheit
	case strings.HasPrefix(unit, "PCT"):
		r.Q, r.Unit = QDuty, "%"
	case strings.HasPrefix(unit, "S"):
		r.Q, r.Unit = QUnitless, "S"
	default:
		r.Q, r.Unit = QUnitless, strings.ToLower(unit)
	}
	if strings.Contains(unit, "AC") {
		r.Flags |= FAC
	}
	if strings.Contains(unit, "DC") {
		r.Flags |= FDC
	}
	if strings.HasPrefix(state, "OL") || math.Abs(v) > 9e37 {
		r.Flags |= FOL
		r.Value = math.NaN()
		r.Display = "OL"
		return r, true
	}
	if state == "INVALID" || state == "OPEN_TC" {
		return r, false
	}
	r.Value = v
	num, pfx := SplitSI(v, r.Unit, 5)
	r.Display, r.Prefix = num, strings.TrimSuffix(pfx, r.Unit)
	return r, true
}

func flukeUnit18x(u string) (Quantity, string, string, Flags) {
	var f Flags
	lu := strings.ToLower(u)
	if strings.Contains(lu, "ac") {
		f |= FAC
	}
	if strings.Contains(lu, "dc") {
		f |= FDC
	}
	pfx := ""
	if len(u) > 1 {
		switch u[0] {
		case 'm':
			pfx = "m"
		case 'u':
			pfx = "µ"
		case 'n':
			pfx = "n"
		case 'k', 'K':
			pfx = "k"
		case 'M':
			pfx = "M"
		}
	}
	switch {
	case strings.Contains(lu, "ohm"):
		return QResistance, "Ω", pfx, f
	case strings.Contains(lu, "hz"):
		return QFrequency, "Hz", pfx, f
	case strings.HasSuffix(lu, "f") && !strings.Contains(lu, "v"):
		return QCapacitance, "F", pfx, f
	case strings.Contains(lu, "v"):
		return QVoltage, "V", pfx, f
	case strings.Contains(lu, "a"):
		return QCurrent, "A", pfx, f
	case lu == "c":
		return QTemperature, "°C", "", f
	case lu == "%":
		return QDuty, "%", "", f
	}
	return QUnitless, u, "", f
}

// ---------------------------------------------------------------------------
// Metex / Mastech / Voltcraft 14-byte ASCII ("DC -1.999  V\r"), host sends 'D'.

type Metex14 struct{ buf []byte }

func (d *Metex14) PollRequest() []byte         { return []byte("D") }
func (d *Metex14) PollInterval() time.Duration { return 500 * time.Millisecond }

func (d *Metex14) Feed(b []byte) []Reading {
	for _, c := range b {
		d.buf = append(d.buf, c&0x7F)
	}
	var out []Reading
	for {
		i := bytes.IndexByte(d.buf, '\r')
		if i < 0 {
			if len(d.buf) > 128 {
				d.buf = d.buf[:0]
			}
			return out
		}
		if i >= 13 {
			if r, ok := ParseMetex14(d.buf[i-13 : i+1]); ok {
				out = append(out, r)
			}
		}
		d.buf = d.buf[i+1:]
	}
}

func ParseMetex14(b []byte) (Reading, bool) {
	if len(b) != 14 || b[13] != '\r' {
		return Reading{}, false
	}
	mode := strings.ToUpper(strings.TrimSpace(string(b[0:2])))
	val := strings.TrimSpace(string(b[3:9]))
	unit := strings.TrimSpace(string(b[9:13]))
	var r Reading
	r.T = time.Now()
	r.Counts = 4000
	switch mode {
	case "DC":
		r.Flags |= FDC
	case "AC":
		r.Flags |= FAC
	case "DI":
		r.Flags |= FDiode
	}
	lu := strings.ToLower(unit)
	pfx := ""
	base := lu
	if len(lu) > 1 {
		switch lu[0] {
		case 'm':
			if unit[0] == 'M' {
				pfx = "M"
			} else {
				pfx = "m"
			}
			base = lu[1:]
		case 'u':
			pfx, base = "µ", lu[1:]
		case 'n':
			pfx, base = "n", lu[1:]
		case 'p':
			pfx, base = "p", lu[1:]
		case 'k':
			pfx, base = "k", lu[1:]
		}
	}
	switch {
	case base == "v":
		r.Q, r.Unit = QVoltage, "V"
		if mode == "DI" {
			r.Q = QDiode
		}
	case base == "a":
		r.Q, r.Unit = QCurrent, "A"
	case strings.HasPrefix(base, "ohm"):
		r.Q, r.Unit = QResistance, "Ω"
	case base == "f":
		r.Q, r.Unit = QCapacitance, "F"
	case base == "hz":
		r.Q, r.Unit = QFrequency, "Hz"
	case base == "c" || mode == "TE":
		r.Q, r.Unit, pfx = QTemperature, "°C", ""
	default:
		r.Q, r.Unit, pfx = QUnitless, unit, ""
	}
	r.Prefix = pfx
	if strings.Contains(strings.ToUpper(val), "OL") || strings.Contains(val, "L") {
		r.Flags |= FOL
		r.Value = math.NaN()
		r.Display = "OL"
		return r, true
	}
	v, err := strconv.ParseFloat(strings.ReplaceAll(val, " ", ""), 64)
	if err != nil {
		return Reading{}, false
	}
	r.Display = strings.TrimPrefix(strings.ReplaceAll(val, " ", ""), "+")
	r.Value = v * prefixMult(pfx)
	return r, true
}

// ---------------------------------------------------------------------------
// Victor 70C / 86C USB HID cable: FS9922 packets, obfuscated by the cable.
// The de-obfuscation (subtract key, reorder, bit-reverse) follows sigrok's notes.
// Several orderings are tried and the one that yields valid FS9922 frames is kept,
// so a wrong guess can never produce bogus readings.

type VictorHID struct {
	variant int // -1 unknown
	fails   int
}

const victorKey = "jodenxunickxia"

var victorShuffle = [14]int{6, 13, 5, 11, 2, 7, 9, 8, 3, 10, 12, 0, 4, 1}

func bitrev8(b byte) byte {
	var r byte
	for i := 0; i < 8; i++ {
		if b&(1<<uint(i)) != 0 {
			r |= 0x80 >> uint(i)
		}
	}
	return r
}

func victorDecode(in []byte, variant int) []byte {
	out := make([]byte, 14)
	if variant == 4 { // exactly as libsigrok serial_hid_victor.c
		for i := 0; i < 14; i++ {
			out[13-victorShuffle[i]] = bitrev8(in[i] - victorKey[i])
		}
		return out
	}
	for i := 0; i < 14; i++ {
		v := in[i] - victorKey[i]
		switch variant & 1 {
		case 0:
			out[victorShuffle[i]] = v
		case 1:
			out[i] = in[victorShuffle[i]] - victorKey[victorShuffle[i]]
		}
	}
	if variant&2 == 0 {
		for i := range out {
			out[i] = bitrev8(out[i])
		}
	}
	return out
}

func (d *VictorHID) Feed(b []byte) []Reading { return d.FeedFrame(b) }

func (d *VictorHID) FeedFrame(b []byte) []Reading {
	if len(b) < 14 {
		return nil
	}
	b = b[:14]
	if d.variant < 0 || d.variant > 4 {
		for _, v := range []int{4, 0, 1, 2, 3} {
			if r, ok := ParseFS9922(victorDecode(b, v), 6000); ok {
				d.variant = v
				return []Reading{r}
			}
		}
		return nil
	}
	if r, ok := ParseFS9922(victorDecode(b, d.variant), 6000); ok {
		d.fails = 0
		return []Reading{r}
	}
	d.fails++
	if d.fails > 8 {
		d.variant = -1
	}
	return nil
}

// ---------------------------------------------------------------------------
// SCPI bench meters over USB-serial (OWON XDM1041/XDM1241/XDM2041, and other
// SCPI-speaking meters). Polls "MEAS?" and periodically "FUNC?".

type SCPIMeter struct {
	buf      []byte
	Query    string
	n        int
	fn       string
	lastPoll string
}

func (d *SCPIMeter) PollRequest() []byte {
	d.n++
	if d.fn == "" || d.n%8 == 0 {
		d.lastPoll = "FUNC?"
		return []byte("FUNC?\n")
	}
	q := d.Query
	if q == "" {
		q = "MEAS?"
	}
	d.lastPoll = q
	return []byte(q + "\n")
}
func (d *SCPIMeter) PollInterval() time.Duration { return 200 * time.Millisecond }

func (d *SCPIMeter) Feed(b []byte) []Reading {
	d.buf = append(d.buf, b...)
	var out []Reading
	for {
		i := bytes.IndexAny(d.buf, "\r\n")
		if i < 0 {
			if len(d.buf) > 512 {
				d.buf = d.buf[:0]
			}
			return out
		}
		line := strings.Trim(strings.TrimSpace(string(d.buf[:i])), "\"")
		d.buf = d.buf[i+1:]
		if line == "" {
			continue
		}
		if v, err := strconv.ParseFloat(line, 64); err == nil {
			out = append(out, d.reading(v))
			continue
		}
		up := strings.ToUpper(line)
		if strings.Contains(up, "OVER") || up == "OL" {
			out = append(out, d.reading(math.NaN()))
			continue
		}
		if len(up) < 24 && !strings.ContainsAny(up, "0123456789") {
			d.fn = up
		}
	}
}

func (d *SCPIMeter) reading(v float64) Reading {
	var r Reading
	r.T = time.Now()
	r.Counts = 60000
	fn := d.fn
	switch {
	case strings.HasPrefix(fn, "VOLT"):
		r.Q, r.Unit = QVoltage, "V"
		if strings.Contains(fn, "AC") {
			r.Flags |= FAC
		} else {
			r.Flags |= FDC
		}
	case strings.HasPrefix(fn, "CURR"):
		r.Q, r.Unit = QCurrent, "A"
		if strings.Contains(fn, "AC") {
			r.Flags |= FAC
		} else {
			r.Flags |= FDC
		}
	case strings.HasPrefix(fn, "RES"), strings.HasPrefix(fn, "FRES"):
		r.Q, r.Unit = QResistance, "Ω"
	case strings.HasPrefix(fn, "CONT"):
		r.Q, r.Unit = QContinuity, "Ω"
		r.Flags |= FBeep
	case strings.HasPrefix(fn, "DIOD"):
		r.Q, r.Unit = QDiode, "V"
		r.Flags |= FDiode
	case strings.HasPrefix(fn, "CAP"):
		r.Q, r.Unit = QCapacitance, "F"
	case strings.HasPrefix(fn, "FREQ"):
		r.Q, r.Unit = QFrequency, "Hz"
	case strings.HasPrefix(fn, "PER"):
		r.Q, r.Unit = QTime, "s"
	case strings.HasPrefix(fn, "TEMP"):
		r.Q, r.Unit = QTemperature, "°C"
	default:
		r.Q, r.Unit = QUnitless, ""
	}
	r.Flags |= FAuto
	if math.IsNaN(v) || math.Abs(v) > 9e36 {
		r.Flags |= FOL
		r.Value = math.NaN()
		r.Display = "OL"
		return r
	}
	r.Value = v
	num, pu := SplitSI(v, r.Unit, 6)
	r.Display, r.Prefix = num, strings.TrimSuffix(pu, r.Unit)
	return r
}

// ---------------------------------------------------------------------------
// Raw passthrough decoder for diagnostics (never yields readings).

type RawDecoder struct{}

func (RawDecoder) Feed(b []byte) []Reading { return nil }
