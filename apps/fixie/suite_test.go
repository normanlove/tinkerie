// Tests for reading values from the other Tinkerie apps. Created by nrmluvdev.
package fixie

import (
	"math"
	"testing"
)

func TestParseSI(t *testing.T) {
	for _, c := range []struct {
		in   string
		v    float64
		unit string
	}{
		{"1.2 mV", 0.0012, "V"}, {"15.62 kHz", 15620, "Hz"}, {"16.00 MHz", 16e6, "Hz"}, {"3.300 V", 3.3, "V"},
		{"50.0 %", 50, "%"}, {"2.5 µs", 2.5e-6, "s"}, {"< 4.00 ns", 4e-9, "s"}, {"-1.5 V", -1.5, "V"},
	} {
		r, err := parseSI(c.in)
		if err != nil || math.Abs(r.Value-c.v) > math.Abs(c.v)*1e-9 || r.Unit != c.unit {
			t.Errorf("%q: got %v %q (%v), want %v %q", c.in, r.Value, r.Unit, err, c.v, c.unit)
		}
	}
}

func TestPeekNotRunning(t *testing.T) {
	// with nothing listening, each source says plainly what to do
	for _, src := range []string{"meterie", "scopie", "probie"} {
		if _, err := peek(src, "freq", 0); err == nil || err.Error() == "" {
			t.Errorf("%s: expected a helpful error", src)
		}
	}
	if _, err := peek("nope", "", 0); err == nil {
		t.Error("unknown source accepted")
	}
}
