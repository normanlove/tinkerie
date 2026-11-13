// Fixie working with the rest of Tinkerie: opening Meterie, Scopie, Probie,
// Flashie or Wirie for a step, and reading live values from them so a step's
// reading box can fill itself in. Created by nrmluvdev.
package fixie

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var suiteApps = map[string]bool{"meterie": true, "scopie": true, "probie": true, "flashie": true, "wirie": true}

// The suite's fixed ports (see PORTING.md).
const (
	meteriePort = 47823 // Meterie's suite API
	scopiePort  = 47831
	probiePort  = 47822
)

// openSuiteApp starts another app from this same program, as the launcher
// does. If it's already open it brings its window to the front itself.
func openSuiteApp(name string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, "--app", name)
	cmd.Dir = filepath.Dir(exe)
	detach(cmd)
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait()
	return nil
}

// Reading is a value from another app, in base SI units (V, A, Ω, F, Hz, s, %).
type Reading struct {
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
	Text  string  `json:"text"`
	OL    bool    `json:"ol,omitempty"`
}

var peekClient = &http.Client{Timeout: 2500 * time.Millisecond}

func peek(src, m string, ch int) (Reading, error) {
	switch src {
	case "meterie":
		return peekMeterie()
	case "scopie":
		return peekScopie(m, ch)
	case "probie":
		return peekProbie(m, ch)
	}
	return Reading{}, errors.New("unknown app")
}

func peekMeterie() (Reading, error) {
	resp, err := peekClient.Get(fmt.Sprintf("http://127.0.0.1:%d/api/v1/reading", meteriePort))
	if err != nil {
		return Reading{}, errors.New("Meterie isn't running, or its suite connection is off (Meterie: Settings)")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return Reading{}, errors.New("Meterie has no reading yet: connect your meter in Meterie first")
	}
	var rd struct {
		Value *float64 `json:"value"`
		Unit  string   `json:"unit"`
		Text  string   `json:"text"`
		OL    bool     `json:"ol"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rd); err != nil {
		return Reading{}, err
	}
	if rd.OL || rd.Value == nil {
		return Reading{Value: math.Inf(1), Unit: rd.Unit, Text: "OL", OL: true}, nil
	}
	return Reading{Value: *rd.Value, Unit: rd.Unit, Text: rd.Text}, nil
}

// peekScopie reads one "status" message from Scopie's event stream and takes
// a measurement from it (m: freq, vpp, mean, rms, duty… ch: 0 for CH1).
func peekScopie(m string, ch int) (Reading, error) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("http://127.0.0.1:%d/api/events", scopiePort), nil)
	resp, err := peekClient.Do(req)
	if err != nil {
		return Reading{}, errors.New("Scopie isn't running: open it and connect your scope")
	}
	defer resp.Body.Close()
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 1<<20), 8<<20)
	event := ""
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "event:") {
			event = strings.TrimSpace(line[6:])
			continue
		}
		if event != "status" || !strings.HasPrefix(line, "data:") {
			continue
		}
		var st struct {
			Connected bool                `json:"connected"`
			Meas      map[string][]string `json:"meas"`
		}
		if json.Unmarshal([]byte(strings.TrimSpace(line[5:])), &st) != nil {
			continue
		}
		vals := st.Meas[m]
		if ch < 0 || ch >= len(vals) || vals[ch] == "" || vals[ch] == "--" {
			if !st.Connected {
				return Reading{}, errors.New("Scopie isn't connected to a scope yet")
			}
			return Reading{}, fmt.Errorf("Scopie has no %s measurement on that channel: turn the measurement on in Scopie and get a steady trace", m)
		}
		return parseSI(vals[ch])
	}
	return Reading{}, errors.New("Scopie didn't answer in time")
}

func peekProbie(m string, ch int) (Reading, error) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("http://127.0.0.1:%d/api/logic/measure?ch=%d&s=0&e=1e15", probiePort, ch), nil)
	req.Header.Set("X-Probie", "1")
	resp, err := peekClient.Do(req)
	if err != nil {
		return Reading{}, errors.New("Probie isn't running: open it and take a logic capture")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return Reading{}, errors.New("Probie has no capture yet: take one on its Logic tab")
	}
	var out map[string]float64
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return Reading{}, err
	}
	switch m {
	case "duty":
		return Reading{Value: out["duty"] * 100, Unit: "%", Text: fmt.Sprintf("%.1f %%", out["duty"]*100)}, nil
	default:
		f, ok := out["freq"]
		if !ok || f <= 0 {
			return Reading{}, errors.New("that channel isn't switching in Probie's capture")
		}
		return Reading{Value: f, Unit: "Hz", Text: fmt.Sprintf("%.4g Hz", f)}, nil
	}
}

var siRe = regexp.MustCompile(`^\s*([<>]?)\s*([-+]?[0-9]*\.?[0-9]+(?:[eE][-+]?[0-9]+)?)\s*([pnuµmkMG]?)\s*(\S*)`)

// parseSI turns "1.2 mV" or "15.62 kHz" into a value in base units.
func parseSI(s string) (Reading, error) {
	m := siRe.FindStringSubmatch(s)
	if m == nil {
		return Reading{}, fmt.Errorf("couldn't read %q", s)
	}
	v, err := strconv.ParseFloat(m[2], 64)
	if err != nil {
		return Reading{}, err
	}
	mult := map[string]float64{"p": 1e-12, "n": 1e-9, "u": 1e-6, "µ": 1e-6, "m": 1e-3, "k": 1e3, "M": 1e6, "G": 1e9, "": 1}[m[3]]
	unit := m[4]
	if m[3] == "m" && unit == "" { // a bare "m" is a unit (metres), not milli
		mult, unit = 1, "m"
	}
	return Reading{Value: v * mult, Unit: unit, Text: strings.TrimSpace(s)}, nil
}
