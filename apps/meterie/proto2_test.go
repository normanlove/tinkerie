package meterie

import (
	"encoding/binary"
	"math"
	"testing"
)

// ZOYI / ANENG vectors from the community decoder's test script
// (Shiro-Nek0/Bluetooth-DMM.py): 11-byte -> 22.0 °C, 10-byte -> -1.359 V DC.
func TestZoyiVectors(t *testing.T) {
	r, ok := ParseZoyi([]byte{0x1b, 0x84, 0x70, 0xb1, 0x49, 0x6a, 0x9f, 0x3c, 0x66, 0xaa, 0x3b})
	if !ok || r.Q != QTemperature || r.Unit != "°C" || !near(r.Value, 22.0) {
		t.Fatalf("11-byte: %+v ok=%v", r, ok)
	}
	r, ok = ParseZoyi([]byte{0x1b, 0x84, 0x71, 0x45, 0x38, 0x0e, 0xf5, 0xfe, 0x60, 0xaa})
	if !ok || r.Q != QVoltage || !r.Flags.Has(FDC) || !near(r.Value, -1.359) || r.Display != "-1.359" {
		t.Fatalf("10-byte: %+v ok=%v", r, ok)
	}
	if rs := (ZoyiBLE{}).FeedFrame([]byte{1, 2, 3}); len(rs) != 0 {
		t.Fatal("short frame accepted")
	}
}

func ut71pkt(digits string, rng, fn byte, f1, f2 byte) []byte {
	b := []byte(digits)
	return append(b, '0'+rng, '0'+fn, 0x30|f1, 0x30|f2, '\r', '\n')
}

func TestUT71x(t *testing.T) {
	d := NewUT71x()
	stream := append([]byte{0x55, 0x0d}, ut71pkt("12345", 1, 1, 2, 1)...) // DC V 4 V range, auto
	stream = append(stream, ut71pkt("12345", 0, 3, 2, 1)...)              // DC mV
	stream = append(stream, ut71pkt("12345", 3, 4, 0, 1)...)              // 40 kΩ range
	stream = append(stream, ut71pkt("::0<:", 3, 4, 0, 1)...)              // OL
	stream = append(stream, ut71pkt("05000", 0, 12, 0, 1|4)...)           // freq + sign = duty
	for i := range stream {
		stream[i] |= 0x80 // parity bit set: must be masked off
	}
	rs := d.Feed(stream)
	if len(rs) != 5 {
		t.Fatalf("want 5, got %d", len(rs))
	}
	if r := rs[0]; r.Q != QVoltage || !r.Flags.Has(FDC) || !r.Flags.Has(FAuto) || !near(r.Value, 1.2345) || r.Display != "1.2345" || r.Prefix != "" {
		t.Fatalf("V: %+v", r)
	}
	if r := rs[1]; !near(r.Value, 0.12345) || r.Display != "123.45" || r.Prefix != "m" {
		t.Fatalf("mV: %+v", r)
	}
	if r := rs[2]; r.Q != QResistance || !near(r.Value, 12345) || r.Display != "12.345" || r.Prefix != "k" {
		t.Fatalf("ohm: %+v", r)
	}
	if !rs[3].IsOL() {
		t.Fatalf("OL: %+v", rs[3])
	}
	if r := rs[4]; r.Q != QDuty || r.Unit != "%" || !near(r.Value, 5.0) {
		t.Fatalf("duty: %+v", r)
	}
}

func TestES519xx11(t *testing.T) {
	// UT60G @19200: range 1, "1234", V, status 0, opt1 0, opt2 DC|AUTO
	p := []byte{'1', '1', '2', '3', '4', 0x3B, 0x30, 0x30, 0x30 | 8 | 2, '\r', '\n'}
	rs := NewES519xx11(19200).Feed(append(p, p...)) // meters send each packet twice
	if len(rs) != 2 {
		t.Fatalf("got %d", len(rs))
	}
	if r := rs[0]; r.Q != QVoltage || !r.Flags.Has(FDC) || !r.Flags.Has(FAuto) || !near(r.Value, 12.34) || r.Display != "12.34" {
		t.Fatalf("%+v", r)
	}
	// 2400 baud chip: overload flag
	ol := []byte{'0', '0', '0', '0', '0', 0x33, 0x30 | 1, 0x30, 0x30, '\r', '\n'}
	if r, ok := ParseES519xx11(ol, 2400); !ok || !r.IsOL() || r.Q != QResistance {
		t.Fatalf("OL: %+v %v", r, ok)
	}
}

// encodeBM25x builds a BM25x packet showing four digits (inverse of the decoder).
func encodeBM25x(digits [4]int, point int, neg bool, flags1, b11, b12, b13, b14 byte) []byte {
	segs := map[int]int{0: 0xBE, 1: 0xA0, 2: 0xDA, 3: 0xF8, 4: 0xE4, 5: 0x7C, 6: 0x7E, 7: 0xA8, 8: 0xFE, 9: 0xFC}
	b := make([]byte, 15)
	b[0] = 2
	b[1] = flags1 & 0x0F
	b[11], b[12], b[13], b[14] = b11&0x0F, b12&0x0F, b13&0x0F, b14&0x0F
	for n := 0; n < 4; n++ {
		v := segs[digits[n]]
		b[3+2*n] |= byte(v & 0x0E)
		b[4+2*n] |= byte(v>>4) & 0x0F
	}
	if neg {
		b[3] |= 1
	}
	if point > 0 {
		b[11-2*point] |= 1
	}
	for i := 1; i < 15; i++ {
		b[i] = b[i]&0x0F | byte(i<<4)
	}
	return b
}

func TestBM25x(t *testing.T) {
	// DC volts, auto, "1.234" (point after first digit -> i=3)
	p := encodeBM25x([4]int{1, 2, 3, 4}, 3, false, 8|4, 0, 0, 0, 4)
	rs := (&BM25x{}).Feed(append([]byte{0xFF, 0x02}, p...))
	if len(rs) != 1 {
		t.Fatalf("got %d", len(rs))
	}
	if r := rs[0]; r.Q != QVoltage || !r.Flags.Has(FDC) || !r.Flags.Has(FAuto) || !near(r.Value, 1.234) || r.Display != "1.234" {
		t.Fatalf("%+v", r)
	}
	// kΩ: "12.34" k
	p = encodeBM25x([4]int{1, 2, 3, 4}, 2, false, 8, 1, 4, 0, 0)
	if r, ok := ParseBM25x(p); !ok || r.Q != QResistance || r.Prefix != "k" || !near(r.Value, 12340) {
		t.Fatalf("kohm: %+v %v", r, ok)
	}
}

// encodeBM86xChar is the inverse of bm86xChar.
func encodeBM86xChar(c byte) byte {
	m := map[byte]byte{'-': 0x20, '0': 0x5F, '1': 0x50, '2': 0x6D, '3': 0x7C, '4': 0x72, '5': 0x3E, '6': 0x3F, '7': 0x54, '8': 0x7F, '9': 0x7E, 'L': 0x0B}
	return m[c] << 1
}

func TestBM86x(t *testing.T) {
	b := make([]byte, 24)
	b[16], b[17], b[18], b[19] = 0x86, 0x86, 0x86, 0x86
	// main display "12.345" V DC auto: digits at b[3..8], point flag on pos 2 (b[5]&1)
	for i, c := range []byte("12345") {
		b[3+i] = encodeBM86xChar(c)
	}
	b[5] |= 1          // decimal point before 3rd digit
	b[8] |= 0x01       // V
	b[1] = 0x10 | 0x01 // DC + auto
	d := &BM86x{}
	if req := d.PollRequest(); len(req) != 4 || req[2] != 0x86 || req[3] != 0x66 {
		t.Fatalf("bad request %x", req)
	}
	rs := d.Feed(append([]byte{9, 9, 9}, b...))
	if len(rs) != 1 {
		t.Fatalf("got %d", len(rs))
	}
	if r := rs[0]; r.Q != QVoltage || !r.Flags.Has(FDC) || !r.Flags.Has(FAuto) || !near(r.Value, 12.345) || r.Display != "12.345" {
		t.Fatalf("%+v", r)
	}
	// OL in kΩ
	c := make([]byte, 24)
	copy(c, b)
	for i := range c[3:9] {
		c[3+i] = 0
	}
	c[8] = 0
	c[4], c[5], c[6] = encodeBM86xChar('0'), encodeBM86xChar('L'), 0
	c[15] = 0x10 | 0x40
	if r, ok := ParseBM86x(c); !ok || !r.IsOL() || r.Q != QResistance {
		t.Fatalf("OL: %+v %v", r, ok)
	}
}

func TestUT181A(t *testing.T) {
	pl := []byte{0x02, 0x00, 0x01, 0x11, 0x31, 0x01}
	f := make([]byte, 4)
	binary.LittleEndian.PutUint32(f, math.Float32bits(1.2345))
	pl = append(pl, f...)
	pl = append(pl, 0x40) // 4 decimals
	pl = append(pl, []byte("VDC\x00\x00\x00\x00\x00")...)
	frame := ut181aFrame(pl)
	d := &UT181A{}
	if req := d.PollRequest(); len(req) != 8 || req[4] != 0x05 || req[5] != 0x01 {
		t.Fatalf("monitor frame %x", req)
	}
	// split delivery + noise
	rs := d.Feed(append([]byte{0x00, 0xAB}, frame[:5]...))
	rs = append(rs, d.Feed(frame[5:])...)
	if len(rs) != 1 {
		t.Fatalf("got %d", len(rs))
	}
	r := rs[0]
	if r.Q != QVoltage || !r.Flags.Has(FDC) || !r.Flags.Has(FAuto) || r.Display != "1.2345" || math.Abs(r.Value-1.2345) > 1e-6 {
		t.Fatalf("%+v", r)
	}
	if d.PollRequest() != nil {
		t.Fatal("should stop re-sending monitor-on while data flows")
	}
	// kΩ + OL flag
	pl2 := []byte{0x02, 0x00, 0x01, 0x11, 0x51, 0x02, 0, 0, 0, 0, 0x01}
	pl2 = append(pl2, []byte("k~\x00\x00\x00\x00\x00\x00")...)
	rs = d.Feed(ut181aFrame(pl2))
	if len(rs) != 1 || !rs[0].IsOL() || rs[0].Q != QResistance || rs[0].Prefix != "k" {
		t.Fatalf("OL: %+v", rs)
	}
	// bad checksum rejected
	bad := ut181aFrame(pl)
	bad[len(bad)-1] ^= 0xFF
	if rs := (&UT181A{}).Feed(bad); len(rs) != 0 {
		t.Fatal("bad checksum accepted")
	}
}

// Victor: obfuscate exactly the way libsigrok's serial_hid_victor.c undoes it.
func TestVictorSigrokExact(t *testing.T) {
	plain := []byte{'-', '0', '4', '5', '6', ' ', '4', 0x30, 0x00, 0x00, 0x80, 0x00, '\r', '\n'}
	obf := make([]byte, 14)
	for i := 0; i < 14; i++ {
		obf[i] = bitrev8(plain[13-victorShuffle[i]]) + victorKey[i]
	}
	d := &VictorHID{variant: -1}
	rs := d.FeedFrame(obf)
	if len(rs) != 1 || !near(rs[0].Value, -45.6) || d.variant != 4 {
		t.Fatalf("bad %+v variant %d", rs, d.variant)
	}
}

func TestPrefixForExp(t *testing.T) {
	cases := []struct {
		exp, dig int
		unit     string
		p        string
		dec      int
	}{
		{-4, 5, "V", "", 4}, {-5, 5, "V", "m", 2}, {-1, 5, "Ω", "k", 4}, {2, 5, "Ω", "M", 4},
		{-12, 5, "F", "n", 3}, {-1, 4, "°C", "", 1},
	}
	for _, c := range cases {
		p, d := prefixForExp(c.exp, c.dig, c.unit)
		if p != c.p || d != c.dec {
			t.Errorf("exp %d: got %q/%d want %q/%d", c.exp, p, d, c.p, c.dec)
		}
	}
}

func TestCatalogDecoders(t *testing.T) {
	for _, m := range meterCatalog {
		if _, raw := newDecoder(m).(RawDecoder); raw {
			t.Errorf("%s (%s) has no decoder", m.ID, m.Proto)
		}
		if len(m.Transports) == 0 {
			t.Errorf("%s has no transport", m.ID)
		}
	}
	if m := bleNameMatch("Bluetooth DMM"); m == nil || m.ID != "zoyi-ble" {
		t.Errorf("ZOYI name match: %v", m)
	}
	if m := bleNameMatch("BDM"); m == nil || m.ID != "owon-ble" {
		t.Errorf("OWON name match: %v", m)
	}
	if u := lookupUSB(0x0820, 0x0001, TrBU86X); u == nil || u.Kind != TrBU86X {
		t.Error("BU-86X USB id missing")
	}
}
