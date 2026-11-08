package tunie

import (
	"bufio"
	"math"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var testPrinters = []Printer{
	{Name: "Ender 3 V2", Firmware: "marlin", BedShape: "rect", BedX: 220, BedY: 220, BedZ: 250, Nozzle: 0.4, FilamentDia: 1.75, MaxHotend: 260, MaxBed: 100, MaxSpeed: 150, MaxAccel: 1000, Drive: "bowden"},
	{Name: "Voron 2.4 300", Firmware: "klipper", BedShape: "rect", BedX: 300, BedY: 300, BedZ: 280, Nozzle: 0.4, FilamentDia: 1.75, MaxHotend: 300, MaxBed: 120, MaxSpeed: 500, MaxAccel: 10000, Drive: "direct", HasInputShaper: true},
	{Name: "Kossel delta", Firmware: "rrf", BedShape: "circle", BedX: 230, BedY: 230, BedZ: 300, OriginCenter: true, Nozzle: 0.4, FilamentDia: 1.75, MaxHotend: 280, MaxBed: 110, MaxSpeed: 200, MaxAccel: 3000, Drive: "bowden"},
	{Name: "Tiny DIY", Firmware: "marlin", BedShape: "rect", BedX: 60, BedY: 60, BedZ: 60, Nozzle: 0.25, FilamentDia: 1.75, MaxHotend: 250, MaxBed: 80, MaxSpeed: 60, MaxAccel: 800, Drive: "direct"},
	{Name: "Prusa MK4", Firmware: "prusa-buddy", BedShape: "rect", BedX: 250, BedY: 210, BedZ: 220, Nozzle: 0.4, FilamentDia: 1.75, MaxHotend: 290, MaxBed: 120, MaxSpeed: 200, MaxAccel: 4000, Drive: "direct", HasInputShaper: true},
}

var reAxis = regexp.MustCompile(`([XYZE])(-?[0-9.]+)`)

func TestAllGenerators(t *testing.T) {
	set := Settings{Hotend: 210, Bed: 60, Fan: 100, RetractLen: 0.8, RetractSpeed: 35, Speed: 50, FirstSpeed: 20, TravelSpeed: 150, Flow: 100, FilamentName: "Test PLA"}
	for _, p := range testPrinters {
		for id := range generators {
			req := GenRequest{Test: id, Printer: p, Settings: set, Params: map[string]float64{}}
			if p.Name == "Tiny DIY" {
				// small printers need smaller tests — mimic the UI's defaults
				req.Params = map[string]float64{"size": 15, "patch": 12, "max": 40, "height": 20, "radius": 10, "distance": 20,
					"bands": 6, "start": 0, "end": 0, "step": 0}
				switch id {
				case "temp-tower":
					req.Params = map[string]float64{"start": 220, "end": 200, "step": 5}
				case "retract-tower":
					req.Params = map[string]float64{"start": 0.2, "end": 1.4, "step": 0.4}
				case "pa-lines":
					continue // documented as too small: UI suggests tower
				case "pa-tower":
					req.Params = map[string]float64{"size": 20, "height": 20, "start": 0, "factor": 0.005}
				case "maxflow":
					req.Params = map[string]float64{"start": 2, "end": 8, "perMM": 0.5, "radius": 10}
				case "overhang":
					req.Params = map[string]float64{"start": 30, "end": 60, "step": 15}
				case "bridge":
					req.Params = map[string]float64{"start": 5, "end": 15, "step": 5}
				case "tolerance":
					req.Params = map[string]float64{"start": 0, "end": 0.4, "step": 0.1, "diameter": 6, "wall": 2}
				case "flow-patches":
					req.Params = map[string]float64{"from": -10, "to": 10, "step": 10, "patch": 12}
				case "ringing-tower":
					req.Params = map[string]float64{"size": 30, "bands": 6}
				case "accuracy-frame":
					req.Params = map[string]float64{"step": 10, "max": 40}
				case "first-layer":
					req.Params = map[string]float64{"patch": 15}
				case "cube":
					req.Params = map[string]float64{"size": 15}
				case "stringing":
					req.Params = map[string]float64{"distance": 20, "height": 20}
				case "flow-wall":
					req.Params = map[string]float64{"size": 15}
				}
			}
			res, err := Generate(req)
			if err != nil {
				t.Errorf("%s / %s: %v", p.Name, id, err)
				continue
			}
			checkGcode(t, p, id, res)
		}
	}
}

func checkGcode(t *testing.T, p Printer, id string, res GenResult) {
	sc := bufio.NewScanner(strings.NewReader(res.Gcode))
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	moves := 0
	for sc.Scan() {
		l := sc.Text()
		if strings.Contains(l, "NaN") || strings.Contains(l, "Inf") {
			t.Fatalf("%s/%s: bad number in %q", p.Name, id, l)
		}
		if !(strings.HasPrefix(l, "G0 ") || strings.HasPrefix(l, "G1 ")) {
			continue
		}
		moves++
		for _, m := range reAxis.FindAllStringSubmatch(strings.SplitN(l, ";", 2)[0], -1) {
			v, _ := strconv.ParseFloat(m[2], 64)
			switch m[1] {
			case "X", "Y":
				lim := p.BedX
				if m[1] == "Y" {
					lim = p.BedY
				}
				if p.OriginCenter || p.BedShape == "circle" {
					if math.Abs(v) > lim/2+0.5 {
						t.Errorf("%s/%s: %s out of bed: %q", p.Name, id, m[1], l)
						return
					}
				} else if v < -0.5 || v > lim+0.5 {
					t.Errorf("%s/%s: %s out of bed: %q", p.Name, id, m[1], l)
					return
				}
			case "Z":
				if v > p.BedZ+12 {
					t.Errorf("%s/%s: Z too high: %q", p.Name, id, l)
					return
				}
			case "E":
				if v > 20 {
					if id == "esteps" {
						continue
					}
					t.Errorf("%s/%s: huge E move: %q", p.Name, id, l)
					return
				}
			}
		}
	}
	if moves == 0 && id != "pid" {
		t.Errorf("%s/%s: no moves", p.Name, id)
	}
}

func TestTextRuns(t *testing.T) {
	if textCols("215") != 17 {
		t.Fatal(textCols("215"))
	}
	r := textRuns("1", 6)
	if len(r) != 1 || r[0] != [2]int{1, 3} {
		t.Fatal(r)
	}
}
