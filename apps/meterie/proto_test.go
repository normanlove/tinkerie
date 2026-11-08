package meterie

import (
	"math"
	"testing"
)

func near(a, b float64) bool { return math.Abs(a-b) <= 1e-9*math.Max(1, math.Abs(b)) }

// sigrok wiki example packet: DC 0.000 V
func TestFS9721SigrokExample(t *testing.T) {
	pkt := []byte{0x17, 0x27, 0x3D, 0x4F, 0x5D, 0x67, 0x7D, 0x87, 0x9D, 0xA0, 0xB0, 0xC0, 0xD4, 0xE0}
	d := NewFS9721(2, 4000)
	rs := d.Feed(append([]byte{0x33, 0x99}, pkt...)) // garbage before
	if len(rs) != 1 {
		t.Fatalf("want 1 reading, got %d", len(rs))
	}
	r := rs[0]
	if r.Q != QVoltage || r.Unit != "V" || !r.Flags.Has(FDC) || !r.Flags.Has(FAuto) || r.Flags.Has(FAC) {
		t.Fatalf("bad mode %+v", r)
	}
	if r.Value != 0 || r.Display != "0.000" {
		t.Fatalf("bad value %v %q", r.Value, r.Display)
	}
}

// encodeFS9721 builds a packet (test helper, the exact inverse of the decoder).
func encodeFS9721(disp string, neg bool, b0, b9, b10, b11, b12, b13 byte) []byte {
	// std mask -> fs9721 raw
	toRaw := func(m byte) byte {
		var raw byte
		if m&1 != 0 {
			raw |= 1 << 4
		}
		if m&2 != 0 {
			raw |= 1 << 0
		}
		if m&4 != 0 {
			raw |= 1 << 2
		}
		if m&8 != 0 {
			raw |= 1 << 3
		}
		if m&16 != 0 {
			raw |= 1 << 6
		}
		if m&32 != 0 {
			raw |= 1 << 5
		}
		if m&64 != 0 {
			raw |= 1 << 1
		}
		return raw
	}
	var digits []byte
	dpBefore := -1
	for _, c := range []byte(disp) {
		if c == '.' {
			dpBefore = len(digits)
			continue
		}
		digits = append(digits, c)
	}
	p := make([]byte, 14)
	p[0] = 0x10 | b0
	for i := 0; i < 4; i++ {
		raw := toRaw(charSegs[digits[i]])
		hi := (raw >> 4) & 7
		lo := raw & 0xF
		if i == 0 && neg {
			hi |= 8
		}
		if i > 0 && i == dpBefore {
			hi |= 8
		}
		p[1+2*i] = byte(2+2*i)<<4 | hi
		p[2+2*i] = byte(3+2*i)<<4 | lo
	}
	p[9] = 0xA0 | b9
	p[10] = 0xB0 | b10
	p[11] = 0xC0 | b11
	p[12] = 0xD0 | b12
	p[13] = 0xE0 | b13
	return p
}

func TestFS9721HP90EPC(t *testing.T) {
	d := NewFS9721(2, 4000) // HP-90EPC: byte14 bit2 = °C
	cases := []struct {
		disp                       string
		neg                        bool
		b0, b9, b10, b11, b12, b13 byte
		q                          Quantity
		val                        float64
		ac, hold, rel              bool
	}{
		{"12.34", false, 0x4 | 0x2, 0, 0, 0, 0x4, 0, QVoltage, 12.34, false, false, false},
		{"230.1", false, 0x8 | 0x2, 0, 0, 0, 0x4, 0, QVoltage, 230.1, true, false, false},
		{"1.234", true, 0x4, 0, 0x8, 0x1, 0x8, 0, QCurrent, -1.234e-3, false, true, false},        // -1.234 mA, hold
		{"4.700", false, 0x2, 0x2, 0, 0x4, 0, 0, QResistance, 4700, false, false, false},          // 4.700 kΩ
		{"22.10", false, 0x2, 0x4, 0, 0x8 | 0x2, 0, 0, QCapacitance, 22.1e-9, false, false, true}, // nF rel
		{"0024", false, 0x2, 0, 0, 0, 0, 0x4, QTemperature, 24, false, false, false},
		{"0.612", false, 0x4, 0x1, 0, 0, 0x4, 0, QDiode, 0.612, false, false, false},
		{"1.000", false, 0x2, 0x2, 0, 0, 0x2, 0, QFrequency, 1000, false, false, false},
	}
	for _, c := range cases {
		pkt := encodeFS9721(c.disp, c.neg, c.b0, c.b9, c.b10, c.b11, c.b12, c.b13)
		rs := d.Feed(pkt)
		if len(rs) != 1 {
			t.Fatalf("%s: no reading", c.disp)
		}
		r := rs[0]
		if r.Q != c.q || !near(r.Value, c.val) || r.Flags.Has(FAC) != c.ac || r.Flags.Has(FHold) != c.hold || r.Flags.Has(FRel) != c.rel {
			t.Errorf("%s: got q=%v v=%g flags=%b disp=%q", c.disp, r.Q, r.Value, r.Flags, r.Display)
		}
	}
	// overload: " 0.L " → digits ' ','0','L',' '
	pkt := encodeFS9721(" 0.L ", false, 0x2, 0x2, 0, 0x4, 0, 0)
	rs := d.Feed(pkt)
	if len(rs) != 1 || !rs[0].IsOL() {
		t.Fatalf("expected OL, got %+v", rs)
	}
}

func TestFS9922(t *testing.T) {
	pkt := []byte{'+', '1', '2', '3', '4', ' ', '2', 0x30, 0x00, 0x00, 0x80, 0x00, '\r', '\n'}
	d := NewFS9922(6000)
	rs := d.Feed(append([]byte("junk\r\n"), pkt...))
	if len(rs) != 1 {
		t.Fatalf("got %d", len(rs))
	}
	r := rs[0]
	if r.Q != QVoltage || !near(r.Value, 12.34) || !r.Flags.Has(FDC) || !r.Flags.Has(FAuto) {
		t.Fatalf("bad %+v", r)
	}
	// -0.123 mA with hold
	pkt = []byte{'-', '0', '1', '2', '3', ' ', '1', 0x02 | 0x10, 0x00, 0x40, 0x40, 0x00, '\r', '\n'}
	rs = d.Feed(pkt)
	if len(rs) != 1 || !near(rs[0].Value, -0.123e-3) || !rs[0].Flags.Has(FHold) || rs[0].Q != QCurrent {
		t.Fatalf("bad %+v", rs)
	}
	// OL ohms
	pkt = []byte{'+', '?', '0', ':', '?', ' ', '2', 0x20, 0x00, 0x20, 0x20, 0x00, '\r', '\n'}
	rs = d.Feed(pkt)
	if len(rs) != 1 || !rs[0].IsOL() || rs[0].Q != QResistance {
		t.Fatalf("bad OL %+v", rs)
	}
}

func TestES51922(t *testing.T) {
	// 1.2345 V DC auto (range 0, fn 0x3B)
	pkt := []byte{0x30, '1', '2', '3', '4', '5', 0x3B, 0x30, 0x30, 0x30, 0x3A, 0x30, '\r', '\n'}
	var d ES51922
	rs := d.Feed(pkt)
	if len(rs) != 1 || !near(rs[0].Value, 1.2345) || rs[0].Display != "1.2345" || !rs[0].Flags.Has(FDC) {
		t.Fatalf("bad %+v", rs)
	}
	// 22.000 kΩ: range 2, fn 0x33
	pkt = []byte{0x32, '2', '2', '0', '0', '0', 0x33, 0x30, 0x30, 0x30, 0x32, 0x30, '\r', '\n'}
	rs = d.Feed(pkt)
	if len(rs) != 1 || !near(rs[0].Value, 22000) || rs[0].Prefix != "k" || rs[0].Display != "22.000" {
		t.Fatalf("bad %+v", rs)
	}
	// mV range: -85.00 mV
	pkt = []byte{0x34, '0', '8', '5', '0', '0', 0x3B, 0x34, 0x30, 0x30, 0x38, 0x30, '\r', '\n'}
	rs = d.Feed(pkt)
	if len(rs) != 1 || !near(rs[0].Value, -0.085) || rs[0].Prefix != "m" || rs[0].Display != "-85.00" {
		t.Fatalf("bad %+v", rs)
	}
}

func TestUT61EPlus(t *testing.T) {
	// frame from mwuertinger/ut61ep README (with the report-ID artefact removed)
	f := []byte{0xAB, 0xCD, 0x10, 0x00, 0x30, 0x20, 0x30, 0x2e, 0x30, 0x38, 0x35, 0x30, 0x00, 0x01, 0x30, 0x30, 0x30, 0x03, 0x94}
	var d UT61EPlus
	rs := d.Feed(append([]byte{0x01, 0x02}, f...))
	if len(rs) != 1 {
		t.Fatalf("no reading")
	}
	r := rs[0]
	if r.Q != QVoltage || !r.Flags.Has(FAC) || !near(r.Value, 0.085) || !r.Flags.Has(FAuto) {
		t.Fatalf("bad %+v", r)
	}
	// request checksum is self-consistent
	s := 0
	for _, c := range ut61epRequest[:4] {
		s += int(c)
	}
	if byte(s>>8) != ut61epRequest[4] || byte(s) != ut61epRequest[5] {
		t.Fatalf("request checksum mismatch")
	}
}

func TestOwon(t *testing.T) {
	// From an OWON B41T+ capture posted on the MIT App Inventor forum
	r, ok := ParseOwon6([]byte{36, 240, 4, 0, 3, 0})
	if !ok || r.Q != QVoltage || !r.Flags.Has(FDC) || !r.Flags.Has(FAuto) || !near(r.Value, 0.0003) {
		t.Fatalf("bad %+v", r)
	}
	// DC mV with 2 decimals, value -1234 -> -12.34 mV
	w0 := uint16(0xF000 | 0<<6 | 3<<3 | 2)
	w2 := uint16(0x8000 | 1234)
	r, ok = ParseOwon6([]byte{byte(w0), byte(w0 >> 8), 1, 0, byte(w2), byte(w2 >> 8)})
	if !ok || !near(r.Value, -0.01234) || !r.Flags.Has(FHold) {
		t.Fatalf("bad %+v", r)
	}
}

func TestFluke(t *testing.T) {
	d := &FlukeQM{}
	rs := d.Feed([]byte("0\r1.2345E+00,VDC,NORMAL,NONE\r0\r9.99999999E+37,OHM,OL,NONE\r"))
	if len(rs) != 2 || !near(rs[0].Value, 1.2345) || !rs[0].Flags.Has(FDC) || !rs[1].IsOL() {
		t.Fatalf("bad %+v", rs)
	}
	rs = d.Feed([]byte("0\rQM,+47.66 KOhms\r0\rQM,-121.43 mVDC\r"))
	if len(rs) != 2 || !near(rs[0].Value, 47660) || rs[0].Q != QResistance || !near(rs[1].Value, -0.12143) {
		t.Fatalf("bad %+v", rs)
	}
}

func TestMetex(t *testing.T) {
	var d Metex14
	rs := d.Feed([]byte("DC -1.999   V\rOH  10.00kOhm\r"))
	if len(rs) != 2 || !near(rs[0].Value, -1.999) || rs[0].Q != QVoltage || !near(rs[1].Value, 10000) {
		t.Fatalf("bad %+v", rs)
	}
}

func TestVictorRoundTrip(t *testing.T) {
	// Build an FS9922 frame, obfuscate it with variant 0, decode.
	plain := []byte{'+', '1', '2', '3', '4', ' ', '2', 0x30, 0x00, 0x00, 0x80, 0x00, '\r', '\n'}
	obf := make([]byte, 14)
	for i := 0; i < 14; i++ {
		obf[i] = bitrev8(plain[victorShuffle[i]]) + victorKey[i]
	}
	d := &VictorHID{variant: -1}
	rs := d.FeedFrame(obf)
	if len(rs) != 1 || !near(rs[0].Value, 12.34) {
		t.Fatalf("bad %+v", rs)
	}
}

func TestSCPI(t *testing.T) {
	d := &SCPIMeter{}
	d.PollRequest()
	rs := d.Feed([]byte("\"VOLT AC\"\n+2.301234E+02\n"))
	if len(rs) != 1 || !near(rs[0].Value, 230.1234) || !rs[0].Flags.Has(FAC) {
		t.Fatalf("bad %+v", rs)
	}
}

func TestFormat(t *testing.T) {
	if s := FormatSI(0.0123, "V", 4); s != "12.30 mV" {
		t.Fatalf("got %q", s)
	}
	if s := FormatSI(4700, "Ω", 3); s != "4.70 kΩ" {
		t.Fatalf("got %q", s)
	}
}

func TestMathExpr(t *testing.T) {
	m := &MathDef{Preset: MathCustom, Expr: "ch1*ch2 + max(A, 2)^2 - abs(-1)"}
	if err := m.Prepare(); err != nil {
		t.Fatal(err)
	}
	v, ok := m.Eval(map[int]float64{1: 3, 2: 4}, 0)
	if !ok || v != 12+9-1 {
		t.Fatalf("got %v %v", v, ok)
	}
	if NearestE(4650, "E12") != 4700 || NearestE(1.02e-9, "E6") != 1e-9 || len(eSeries["E96"]) != 96 || len(eSeries["E48"]) != 48 {
		t.Fatalf("E series wrong: %v %v %d", NearestE(4650, "E12"), NearestE(1.02e-9, "E6"), len(eSeries["E96"]))
	}
	e := &MathDef{Preset: MathEnergy, A: 1, B: 2}
	for i := 0; i <= 3600; i++ { // 1 hour at 1 W, 1 s steps
		e.Eval(map[int]float64{1: 1, 2: 1}, int64(i)*1e9)
	}
	if math.Abs(e.acc-1) > 1e-9 {
		t.Fatalf("energy %v", e.acc)
	}
}
