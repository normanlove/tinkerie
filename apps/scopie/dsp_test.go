package scopie

import (
	"encoding/json"
	"math"
	"os"
	"testing"
)

// testdata/dsp_expected.json was produced by the original Python/numpy dsp.py
// on the same signals, so these tests prove the Go port gives the same answers.
type dspExpected struct {
	DT      float64                        `json:"dt"`
	Signals map[string][]float64           `json:"signals"`
	Meas    map[string]map[string]*float64 `json:"meas"`
	Phase   [2]float64                     `json:"phase"`
	FFT     map[string]struct {
		DF float64   `json:"df"`
		DB []float64 `json:"db"`
	} `json:"fft"`
	Sinc struct {
		Y   []float64 `json:"y"`
		Out []float64 `json:"out"`
	} `json:"sinc"`
}

func loadExpected(t *testing.T) dspExpected {
	b, err := os.ReadFile("testdata/dsp_expected.json")
	if err != nil {
		t.Fatal(err)
	}
	var e dspExpected
	if err := json.Unmarshal(b, &e); err != nil {
		t.Fatal(err)
	}
	return e
}

func near(a, b, rel float64) bool {
	if math.IsNaN(a) && math.IsNaN(b) {
		return true
	}
	return math.Abs(a-b) <= rel*math.Max(1e-9, math.Max(math.Abs(a), math.Abs(b)))
}

func TestMeasureMatchesNumpy(t *testing.T) {
	e := loadExpected(t)
	for name, sig := range e.Signals {
		got := measure(sig, e.DT)
		for k, want := range e.Meas[name] {
			w := nan
			if want != nil {
				w = *want
			}
			if !near(got[k], w, 1e-6) {
				t.Errorf("%s %s: got %v want %v", name, k, got[k], w)
			}
		}
	}
}

func TestPhaseMatchesNumpy(t *testing.T) {
	e := loadExpected(t)
	d, ph := phaseDelay(e.Signals["sine"], e.Signals["sine2"], e.DT)
	if !near(d, e.Phase[0], 1e-6) || !near(ph, e.Phase[1], 1e-6) {
		t.Errorf("phase: got %v %v want %v", d, ph, e.Phase)
	}
}

func TestSpectrumMatchesNumpy(t *testing.T) {
	e := loadExpected(t)
	sig := e.Signals["square"][:3000]
	for w, want := range e.FFT {
		db, df := spectrum(sig, e.DT, w, 65536)
		if !near(df, want.DF, 1e-9) {
			t.Errorf("%s df %v want %v", w, df, want.DF)
		}
		for i, x := range want.DB {
			if math.Abs(db[i]-x) > 1e-6 {
				t.Errorf("%s bin %d: %v want %v", w, i, db[i], x)
				break
			}
		}
	}
}

func TestSincMatchesNumpy(t *testing.T) {
	e := loadExpected(t)
	got := sincInterpolate(e.Sinc.Y, 8)
	if len(got) != len(e.Sinc.Out) {
		t.Fatalf("len %d want %d", len(got), len(e.Sinc.Out))
	}
	for i := range got {
		if math.Abs(got[i]-e.Sinc.Out[i]) > 1e-9 {
			t.Fatalf("sinc %d: %v want %v", i, got[i], e.Sinc.Out[i])
		}
	}
}

func TestFmtSI(t *testing.T) {
	for _, c := range []struct {
		v    float64
		u    string
		want string
	}{{0.0012, "V", "1.2 mV"}, {1000, "Hz", "1 kHz"}, {0, "V", "0 V"}, {-2.5e-6, "s", "-2.5 µs"},
		{999.96, "Hz", "1 kHz"}, {math.NaN(), "V", "--"}, {12345678, "S/s", "12.35 MS/s"}} {
		if got := fmtSI(c.v, c.u, 4); got != c.want {
			t.Errorf("fmtSI(%v) = %q want %q", c.v, got, c.want)
		}
	}
}

func TestCalByte(t *testing.T) {
	// values from the Python _cal_byte
	for f, want := range map[float64]int{32: 103, 50: 105, 1000: 210, 2000: 220, 5000: 250, 10000: 10, 100000: 100} {
		if got := calByte(f); got != want {
			t.Errorf("calByte(%v) = %d want %d", f, got, want)
		}
	}
}

func TestLoadCSV(t *testing.T) {
	text := "time_s;ch1_V;ch2_V\n0;0.1;1\n0.001;0.2;1.5\n0.002;0.3;2\n0.003;0.4;2.5\n0.004;0.5;3\n"
	f, err := loadCSV([]byte(text))
	if err != nil {
		t.Fatal(err)
	}
	if !near(f.DT, 0.001, 1e-9) || f.Data[0] == nil || f.Data[1] == nil || f.Data[2] != nil {
		t.Fatalf("bad frame: dt=%v", f.DT)
	}
	if len(f.Data[1]) != 5 || f.Data[1][4] != 3 {
		t.Errorf("data %v", f.Data[1])
	}
}
