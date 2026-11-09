# Tunie ⇄ Makie integration

*Tunie and Makie are both by nrmluvdev, part of the nrmluvdev maker suite.*

## 1. Opening Tunie from Makie

Makie can launch Tunie and pick the printer/filament/page:

```
Tinkerie.exe --app tunie --from makie --printer "Ender 3 V2 / V2 Neo" --filament "Generic PLA" --page tolerance
```

If Tunie is already open, the new launch hands over to the open window (it switches printer/filament and page) and exits.

## 2. Reading the active profile

Whenever the active printer, filament or any calibrated value changes, Tunie writes:

```
%APPDATA%\nrmluvdev\Suite\tunie-active.json
```

(`os.UserConfigDir()/nrmluvdev/Suite/tunie-active.json` on other systems.) It is written atomically, so it is always complete.

```json
{
  "app": "Tunie", "author": "nrmluvdev", "version": "0.5.0-alpha", "schema": 1,
  "updated": "2026-11-07T12:06:09.911Z",
  "printer": {
    "id": "…", "name": "Ender 3 V2 / V2 Neo", "firmware": "marlin", "bedShape": "rect",
    "bed": [220, 220, 250], "nozzle": 0.4, "filamentDiameter": 1.75, "lineWidth": 0.45,
    "maxSpeed": 150, "maxAccel": 1000, "values": { "eSteps": 93.4 }
  },
  "filament": {
    "id": "…", "brand": "Generic", "type": "PLA", "colour": "#d8d8d8", "colourName": "Natural",
    "temperature": 210, "bedTemperature": 60, "flow": 98.5, "pressureAdvance": 0.045,
    "retractLength": 0.8, "retractSpeed": 35, "maxVolumetricFlow": 12.5
  },
  "fit": { "press": 0.2, "slide": 0.3, "loose": 0.5, "holeCompensation": 0.05, "measured": "…" },
  "shrinkage": { "x": 99.75, "y": 99.75, "z": 100 },
  "xyCompensation": -0.025,
  "maxOverhang": 55, "maxBridge": 30
}
```

Any value may be `null` if that test hasn't been done yet.

### Meaning of the fit values

All clearances are **diametral** (hole diameter minus shaft/peg diameter), in mm, measured with Tunie's tolerance test:

| Field | Use it for | Hole to model for a shaft of diameter *d* |
|---|---|---|
| `press` | Parts that should stay put when pushed together | `d + press` |
| `slide` | Parts that slide or rotate smoothly | `d + slide` |
| `loose` | Free fit, easy assembly | `d + loose` |

`shrinkage.x/y/z` are slicer scale percentages (100 = no change). `xyCompensation` is the slicer's XY size / contour compensation in mm.

## 3. Example: reading it in Go (Makie side)

```go
type TunieFit struct {
	Press *float64 `json:"press"`
	Slide *float64 `json:"slide"`
	Loose *float64 `json:"loose"`
}
type TunieActive struct {
	Printer struct {
		Name      string  `json:"name"`
		Nozzle    float64 `json:"nozzle"`
		LineWidth float64 `json:"lineWidth"`
	} `json:"printer"`
	Filament struct {
		Brand string `json:"brand"`
		Type  string `json:"type"`
	} `json:"filament"`
	Fit *TunieFit `json:"fit"`
}

func loadTunie() (*TunieActive, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(filepath.Join(dir, "nrmluvdev", "Suite", "tunie-active.json"))
	if err != nil {
		return nil, err // Tunie not used yet — fall back to defaults (e.g. 0.2 / 0.3 / 0.5)
	}
	var t TunieActive
	return &t, json.Unmarshal(b, &t)
}

// holeFor returns the hole diameter to model for a shaft, for the chosen fit.
func holeFor(t *TunieActive, shaft float64, fit string) float64 {
	def := map[string]float64{"press": 0.2, "slide": 0.3, "loose": 0.5}[fit]
	if t != nil && t.Fit != nil {
		switch fit {
		case "press":
			if t.Fit.Press != nil { def = *t.Fit.Press }
		case "slide":
			if t.Fit.Slide != nil { def = *t.Fit.Slide }
		case "loose":
			if t.Fit.Loose != nil { def = *t.Fit.Loose }
		}
	}
	return shaft + def
}
```

To let users jump from Makie into calibration: `exec.Command(tuniePath, "--from", "makie", "--page", "tolerance").Start()`.
