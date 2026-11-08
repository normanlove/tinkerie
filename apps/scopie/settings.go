// Front-panel settings: everything the user can set, saved between sessions
// and in setup files. The interface sends the whole set whenever a control
// changes; Scopie turns it into a Panel for the engine.
package scopie

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type ChSettings struct {
	On       bool    `json:"on"`
	VDiv     float64 `json:"vdiv"`  // volts/div at the BNC socket (probe x1)
	Probe    float64 `json:"probe"` // 1, 10 or 100
	Coupling string  `json:"coupling"`
	Pos      float64 `json:"pos"` // divisions
	Invert   bool    `json:"invert"`
	Demo     string  `json:"demo"` // demo signal key
}

type Settings struct {
	Rev int `json:"rev"`

	Ch [maxCh]ChSettings `json:"ch"`

	TDiv  float64 `json:"tdiv"`
	TrigX float64 `json:"trigX"` // divisions from the centre
	Rate  float64 `json:"rate"`  // 0 = automatic
	Roll  bool    `json:"roll"`

	TrigMode    string  `json:"trigMode"`
	TrigSource  int     `json:"trigSource"`
	TrigEdge    string  `json:"trigEdge"`
	TrigLevel   float64 `json:"trigLevel"` // volts at the probe tip
	Holdoff     float64 `json:"holdoff"`   // microseconds
	NoiseReject bool    `json:"noiseReject"`

	MathOn   bool    `json:"mathOn"`
	MathA    int     `json:"mathA"`
	MathB    int     `json:"mathB"`
	MathOp   int     `json:"mathOp"` // 0 A+B, 1 A-B, 2 AxB
	MathVDiv float64 `json:"mathVdiv"`
	MathPos  float64 `json:"mathPos"`

	Style   string  `json:"style"`   // lines | dots
	Interp  string  `json:"interp"`  // sinc | linear
	Average int     `json:"average"` // 1 = off
	Persist float64 `json:"persist"` // seconds; 0 off, -1 infinite
	Grid    int     `json:"grid"`
	XY      bool    `json:"xy"`

	FFTOn     bool   `json:"fftOn"`
	FFTSource int    `json:"fftSource"` // channel, or 4 = math
	FFTWindow string `json:"fftWindow"`
	FFTLog    bool   `json:"fftLog"`

	CursorMode   int        `json:"cursorMode"` // 0 off, 1 time, 2 voltage, 3 both
	CursorSource int        `json:"cursorSource"`
	Cursors      [4]float64 `json:"cursors"` // A, B (x divs), C, D (y divs)

	CalFreq float64 `json:"calFreq"`
	HWACDC  bool    `json:"hwACDC"`

	Meas []string `json:"meas"`

	LAN        string `json:"lan"`
	LastDevice string `json:"lastDevice"`
}

var mathScales = []float64{0.01, 0.02, 0.05, 0.1, 0.2, 0.5, 1, 2, 5, 10, 20, 50}
var avgOpts = []int{1, 2, 4, 8, 16, 32, 64, 128}
var persistOpts = []float64{0, 0.2, 0.5, 1, 2, 5, -1}
var defaultMeas = []string{"freq", "vpp", "mean", "rms"}

const rollThreshold = 100e-3

func defaultSettings() Settings {
	s := Settings{
		TDiv: 1e-3, Roll: true,
		TrigMode: "auto", TrigEdge: "rising", TrigLevel: 1.0,
		MathA: 0, MathB: 1, MathOp: 1, MathVDiv: 1,
		Style: "lines", Interp: "sinc", Average: 1, Grid: 60,
		FFTWindow: "Hann",
		Cursors:   [4]float64{-2, 2, 1.5, -1.5},
		CalFreq:   1000,
		Meas:      append([]string(nil), defaultMeas...),
	}
	pos := []float64{1, -2, 3, -3}
	demo := []string{"cal", "sine", "square", "triangle"}
	for c := range s.Ch {
		s.Ch[c] = ChSettings{On: c < 2, VDiv: 0.1, Probe: 10, Coupling: "DC", Pos: pos[c], Demo: demo[c]}
	}
	return s
}

func finite(v, def float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return def
	}
	return v
}

// sanitize keeps every value in range so a damaged file can't break anything.
func (s *Settings) sanitize() {
	d := defaultSettings()
	validDemo := map[string]bool{}
	for _, ds := range demoSignals {
		validDemo[ds.Key] = true
	}
	for c := range s.Ch {
		ch := &s.Ch[c]
		if !(ch.VDiv > 0) || math.IsInf(ch.VDiv, 0) {
			ch.VDiv = 0.1
		}
		if ch.Probe != 1 && ch.Probe != 10 && ch.Probe != 100 {
			ch.Probe = 10
		}
		if ch.Coupling != "AC" {
			ch.Coupling = "DC"
		}
		ch.Pos = clampF(finite(ch.Pos, 0), -4, 4)
		if !validDemo[ch.Demo] {
			ch.Demo = d.Ch[c].Demo
		}
	}
	if !(s.TDiv > 0) || math.IsInf(s.TDiv, 0) {
		s.TDiv = 1e-3
	}
	s.TrigX = clampF(finite(s.TrigX, 0), -7, 7)
	if !(s.Rate >= 0) || math.IsInf(s.Rate, 0) {
		s.Rate = 0
	}
	if s.TrigMode != "normal" {
		s.TrigMode = "auto"
	}
	switch s.TrigEdge {
	case "rising", "falling", "both":
	default:
		s.TrigEdge = "rising"
	}
	s.TrigSource = clampInt(s.TrigSource, 0, maxCh-1)
	s.TrigLevel = finite(s.TrigLevel, 0)
	s.Holdoff = clampF(finite(s.Holdoff, 0), 0, 1e7)
	s.MathA = clampInt(s.MathA, 0, maxCh-1)
	s.MathB = clampInt(s.MathB, 0, maxCh-1)
	s.MathOp = clampInt(s.MathOp, 0, 2)
	if !(s.MathVDiv > 0) || math.IsInf(s.MathVDiv, 0) {
		s.MathVDiv = 1
	}
	s.MathVDiv = mathScales[nearestIndex(mathScales, s.MathVDiv)]
	s.MathPos = clampF(finite(s.MathPos, 0), -4, 4)
	if s.Style != "dots" {
		s.Style = "lines"
	}
	if s.Interp != "linear" {
		s.Interp = "sinc"
	}
	okAvg := false
	for _, a := range avgOpts {
		if s.Average == a {
			okAvg = true
		}
	}
	if !okAvg {
		s.Average = 1
	}
	okP := false
	for _, p := range persistOpts {
		if s.Persist == p {
			okP = true
		}
	}
	if !okP {
		s.Persist = 0
	}
	s.Grid = clampInt(s.Grid, 0, 100)
	s.FFTSource = clampInt(s.FFTSource, 0, maxCh)
	okW := false
	for _, w := range fftWindows {
		if s.FFTWindow == w {
			okW = true
		}
	}
	if !okW {
		s.FFTWindow = "Hann"
	}
	s.CursorMode = clampInt(s.CursorMode, 0, 3)
	s.CursorSource = clampInt(s.CursorSource, 0, maxCh)
	for i := range s.Cursors {
		s.Cursors[i] = clampF(finite(s.Cursors[i], d.Cursors[i]), -8, 8)
	}
	if !(s.CalFreq > 0) {
		s.CalFreq = 1000
	}
	valid := map[string]bool{"phase": true, "delay": true}
	for _, m := range measurements {
		valid[m.Key] = true
	}
	var meas []string
	seen := map[string]bool{}
	for _, m := range s.Meas {
		if valid[m] && !seen[m] {
			meas = append(meas, m)
			seen[m] = true
		}
	}
	if s.Meas == nil {
		meas = append([]string(nil), defaultMeas...)
	}
	s.Meas = meas
	if s.Meas == nil {
		s.Meas = []string{}
	}
	if len(s.LAN) > 500 {
		s.LAN = s.LAN[:500]
	}
}

func (s *Settings) probe(c int) float64 { return s.Ch[c].Probe }

// vdivTip is volts per division at the probe tip.
func (s *Settings) vdivTip(c int) float64 { return s.Ch[c].VDiv * s.Ch[c].Probe }

func (s *Settings) sign(c int) float64 {
	if s.Ch[c].Invert {
		return -1
	}
	return 1
}

// ---------------------------------------------------------------- saving

type settingsStore struct {
	path  string
	mu    sync.Mutex
	timer *time.Timer
	data  []byte
}

func newSettingsStore() *settingsStore {
	return &settingsStore{path: filepath.Join(appDataDir(), "settings.json")}
}

func (st *settingsStore) load() Settings {
	s := defaultSettings()
	if b, err := os.ReadFile(st.path); err == nil {
		_ = json.Unmarshal(b, &s)
	}
	s.sanitize()
	return s
}

// save writes the settings a moment later (dragging a slider changes them often).
func (st *settingsStore) save(s Settings) {
	b, err := json.MarshalIndent(s, "", " ")
	if err != nil {
		return
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	st.data = b
	if st.timer == nil {
		st.timer = time.AfterFunc(time.Second, st.flush)
	}
}

func (st *settingsStore) flush() {
	st.mu.Lock()
	b := st.data
	st.data = nil
	st.timer = nil
	st.mu.Unlock()
	if b != nil {
		tmp := st.path + ".tmp"
		if os.WriteFile(tmp, b, 0o644) == nil {
			_ = os.Rename(tmp, st.path)
		}
	}
}
