// Shared types for every oscilloscope driver.
//
// Two kinds of driver exist:
//
//   - StreamDevice: sends an endless stream of samples (Hantek 6022 family,
//     demo USB scope). The engine triggers in software, which is what makes
//     thousands of waveforms per second possible.
//   - FrameDevice: the scope triggers by itself and hands over complete
//     waveforms ("frames"): bench scopes, PicoScopes, handhelds and the older
//     Hantek DSO-2xxx/5xxx.
//
// All drivers deliver integer ADC codes plus, per channel, a scale (k, zero)
// so that  volts_at_the_BNC_socket = (code - zero) * k.
// Probe attenuation is applied by the user interface, never by a driver.
package scopie

import (
	"fmt"
	"log"
	"math"
	"sort"
	"strings"
	"sync"
)

const maxCh = 4

// DeviceError is a problem a user can understand and maybe fix.
type DeviceError struct{ Msg string }

func (e *DeviceError) Error() string { return e.Msg }

func devErr(format string, a ...any) error { return &DeviceError{fmt.Sprintf(format, a...)} }

// Common front-panel steps (1-2-5). Drivers pick the ones they support.
var vdiv125 = []float64{0.001, 0.002, 0.005, 0.01, 0.02, 0.05, 0.1, 0.2, 0.5, 1, 2, 5, 10}
var tdiv125 = []float64{
	1e-9, 2e-9, 5e-9, 10e-9, 20e-9, 50e-9, 100e-9, 200e-9, 500e-9,
	1e-6, 2e-6, 5e-6, 10e-6, 20e-6, 50e-6, 100e-6, 200e-6, 500e-6,
	1e-3, 2e-3, 5e-3, 10e-3, 20e-3, 50e-3, 100e-3, 200e-3, 500e-3,
	1, 2, 5, 10, 20, 50,
}

func stepsBetween(lo, hi float64, steps []float64) []float64 {
	var out []float64
	for _, s := range steps {
		if lo*0.999 <= s && s <= hi*1.001 {
			out = append(out, s)
		}
	}
	return out
}

// DeviceInfo is what the user interface needs to know about a scope.
type DeviceInfo struct {
	Name             string    `json:"name"`
	Kind             string    `json:"kind"` // "stream" or "frame"
	Channels         int       `json:"channels"`
	VDivs            []float64 `json:"vdivs"`
	TDivs            []float64 `json:"tdivs"`
	HDivs            int       `json:"hdivs"`
	SampleRates      []float64 `json:"sampleRates"`
	MaxRate1         float64   `json:"maxRate1"`
	MaxRateMulti     float64   `json:"maxRateMulti"`
	HWAC             bool      `json:"hwAC"`          // AC coupling is done in hardware
	ACDCModOption    bool      `json:"acdcModOption"` // 6022BE/BL optional AC/DC mod
	CalOutput        bool      `json:"calOutput"`
	CalFreqs         []float64 `json:"calFreqs"`
	TriggerBothEdges bool      `json:"triggerBothEdges"`
	TriggerHoldoff   bool      `json:"triggerHoldoff"`
	RollMode         bool      `json:"rollMode"`
	Bits             int       `json:"bits"`
	Notes            string    `json:"notes"`
	Simulated        bool      `json:"simulated"`
}

func defaultInfo(name string) *DeviceInfo {
	return &DeviceInfo{
		Name: name, Kind: "stream", Channels: 2,
		VDivs:            stepsBetween(0.01, 5, vdiv125),
		TDivs:            stepsBetween(50e-9, 5, tdiv125),
		HDivs:            10,
		TriggerBothEdges: true, TriggerHoldoff: true, RollMode: true, Bits: 8,
	}
}

// frameInfo is the usual starting point for a self-triggering scope.
func frameInfo(name string, channels int) *DeviceInfo {
	return &DeviceInfo{
		Name: name, Kind: "frame", Channels: channels,
		VDivs:            stepsBetween(0.001, 10, vdiv125),
		TDivs:            stepsBetween(1e-9, 50, tdiv125),
		HDivs:            10,
		HWAC:             true,
		TriggerBothEdges: true, Bits: 8,
	}
}

type ChannelPanel struct {
	Enabled  bool    `json:"enabled"`
	VDiv     float64 `json:"vdiv"`     // volts per division at the BNC socket
	Pos      float64 `json:"pos"`      // vertical position in divisions (0 V line)
	Coupling string  `json:"coupling"` // "DC" or "AC"
}

// Panel holds the device-independent front-panel settings.
type Panel struct {
	Ch             [maxCh]ChannelPanel `json:"ch"`
	TDiv           float64             `json:"tdiv"`
	TrigX          float64             `json:"trigX"` // divisions from screen centre
	Rate           float64             `json:"rate"`  // 0 = automatic
	TrigSource     int                 `json:"trigSource"`
	TrigEdge       string              `json:"trigEdge"`  // rising | falling | both
	TrigLevel      float64             `json:"trigLevel"` // volts at the BNC socket
	TrigMode       string              `json:"trigMode"`  // auto | normal
	Holdoff        float64             `json:"holdoff"`
	NoiseReject    bool                `json:"noiseReject"`
	Roll           bool                `json:"roll"`
	Persist        bool                `json:"persist"`
	Average        bool                `json:"average"`
	CalFreq        float64             `json:"calFreq"`
	HWACDCMod      bool                `json:"hwACDCMod"`
	ChannelsNeeded []int               `json:"channelsNeeded"`
}

func defaultPanel() Panel {
	p := Panel{TDiv: 1e-3, TrigEdge: "rising", TrigMode: "auto", CalFreq: 1000, ChannelsNeeded: []int{0, 1}}
	for i := range p.Ch {
		p.Ch[i] = ChannelPanel{Enabled: true, VDiv: 0.1, Coupling: "DC"}
	}
	return p
}

func (p *Panel) sanitize() {
	if !(p.TDiv > 0) {
		p.TDiv = 1e-3
	}
	for i := range p.Ch {
		if !(p.Ch[i].VDiv > 0) {
			p.Ch[i].VDiv = 0.1
		}
		if p.Ch[i].Coupling != "AC" {
			p.Ch[i].Coupling = "DC"
		}
	}
	if p.TrigSource < 0 || p.TrigSource >= maxCh {
		p.TrigSource = 0
	}
	switch p.TrigEdge {
	case "rising", "falling", "both":
	default:
		p.TrigEdge = "rising"
	}
	if p.TrigMode != "normal" {
		p.TrigMode = "auto"
	}
	if !(p.CalFreq > 0) {
		p.CalFreq = 1000
	}
}

// Scale converts codes to volts at the BNC socket: v = (code - Zero) * K.
type Scale struct{ K, Zero float64 }

// CodeRange is the lowest and highest code a channel can produce.
type CodeRange struct{ Lo, Hi float64 }

// FrameData is one captured waveform from a FrameDevice. Codes[c] is nil for
// channels that were not captured.
type FrameData struct {
	Codes     [maxCh][]float32
	DT        float64
	TFirst    float64 // time of sample 0 relative to the trigger
	Scales    [maxCh]Scale
	Ranges    [maxCh]CodeRange
	Triggered bool
}

func newFrameData() *FrameData {
	f := &FrameData{Triggered: true}
	for c := range f.Scales {
		f.Scales[c] = Scale{1, 0}
		f.Ranges[c] = CodeRange{0, 255}
	}
	return f
}

// Device is implemented by every driver.
type Device interface {
	Info() *DeviceInfo
	// Open connects (uploads firmware etc.). progress may be nil.
	Open(progress func(string)) error
	Close()
}

type Gain struct {
	ID int
	FS float64 // +/- full-scale volts
}

type RateID struct {
	ID   int
	Rate float64
}

// StreamDevice samples continuously; the engine triggers in software.
type StreamDevice interface {
	Device
	Scale(ch, gain int) Scale
	Gains() []Gain
	RateIDs(nch int) []RateID
	// Configure applies settings and returns the number of channels streamed.
	Configure(rateID, nch int, gains [2]int, dc [2]bool, hwACDCMod bool, calFreq float64) (int, error)
	// StartStream delivers interleaved uint8 sample blocks (CH1,CH2,CH1,...
	// for two channels) to fn from its own goroutine until StopStream.
	// fn must not keep the slice. Fatal problems go to fail.
	StartStream(chunkBytes int, fn func([]byte), fail func(error)) error
	StopStream()
}

// FrameDevice captures and triggers by itself.
type FrameDevice interface {
	Device
	ApplyPanel(p *Panel) error
	Start(single bool) error
	Stop() error
	// ReadFrame returns the next waveform, or nil if nothing is ready within timeout seconds.
	ReadFrame(timeout float64) (*FrameData, error)
}

// Candidate is a scope found by a scan.
type Candidate struct {
	ID      string                 `json:"id"`
	Label   string                 `json:"label"`
	Family  string                 `json:"family"`
	Detail  string                 `json:"detail"`
	Ready   bool                   `json:"ready"`
	Factory func() (Device, error) `json:"-"`
}

// ---------------------------------------------------------------- registry

type driverModule struct {
	family string
	scan   func(lan []string, notes *[]string) []*Candidate
}

var driverModules = []driverModule{
	{"Hantek 6022 family (6022BE/BL, 6021, DDS120, DSO-2020, ISDS-205B, MDSO)", scanFX2},
	{"Hantek DSO-2090 / 2150 / 2250 / 5200 / 5200A", scanHantekDSO},
	{"Hantek handhelds 2C42 / 2C72 / 2D42 / 2D72", scanHandheld},
	{"OWON (HDS200/HDS300 handhelds, SDS/XDS/TAO bench scopes)", scanOwon},
	{"Bench scopes over USB/LAN: Rigol, Siglent, Keysight/Agilent, Tektronix", scanSCPI},
	{"PicoScope 2000 / 3000 / 4000 / 5000 series", scanPico},
	{"Demo mode", scanSim},
}

var candMu sync.Mutex
var candidates = map[string]*Candidate{}

// scanAll finds every supported scope. A failing driver never breaks the scan.
func scanAll(lan []string) (out []*Candidate, notes []string) {
	invalidateUSBCache()
	for _, m := range driverModules {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("scan %s panicked: %v", m.family, r)
					notes = append(notes, fmt.Sprintf("%s: search failed (%v)", m.family, r))
				}
			}()
			out = append(out, m.scan(lan, &notes)...)
		}()
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		ad, bd := a.Family == "Demo mode", b.Family == "Demo mode"
		if ad != bd {
			return !ad
		}
		return a.Ready && !b.Ready
	})
	candMu.Lock()
	candidates = map[string]*Candidate{}
	for i, c := range out {
		c.ID = fmt.Sprintf("c%d", i)
		candidates[c.ID] = c
	}
	candMu.Unlock()
	return out, notes
}

func candidateByID(id string) *Candidate {
	candMu.Lock()
	defer candMu.Unlock()
	return candidates[id]
}

// ---------------------------------------------------------------- helpers

func nearestIndex(values []float64, v float64) int {
	best, bi := math.Inf(1), 0
	if v <= 0 {
		return 0
	}
	for i, x := range values {
		if d := math.Abs(math.Log(x / v)); d < best {
			best, bi = d, i
		}
	}
	return bi
}

func titleCase(s string) string {
	words := strings.Fields(strings.ToLower(s))
	for i, w := range words {
		if w != "" {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
