//go:build !windows

package meterie

// preview_text.go: the text engine of the headless preview renderer
// (tinkerie --app meterie -preview outdir) and of the UI tests.

import (
	"encoding/json"
	"os"
	"strings"
)

var previewPorts []PortInfo
var previewBLE []PortInfo

// ---------------------------------------------------------------------------
// previewText approximates text metrics (Inter) and records draw calls so a
// script can composite real glyphs onto the rendered PNG.

type textOp struct {
	S      string   `json:"s"`
	X      int      `json:"x"`
	Y      int      `json:"y"`
	Size   float64  `json:"size"`
	Weight int      `json:"w"`
	Mono   bool     `json:"mono"`
	Icon   bool     `json:"icon"`
	Col    [4]uint8 `json:"c"`
	Clip   [4]int   `json:"clip"`
}

type previewText struct {
	ops  []textOp
	clip Rect
	adv  map[rune]float64
}

func (t *previewText) loadMetrics() {
	if t.adv != nil {
		return
	}
	t.adv = map[rune]float64{}
	b, err := os.ReadFile(os.Getenv("METERIE_METRICS"))
	if err == nil {
		var m map[string]float64
		if json.Unmarshal(b, &m) == nil {
			for k, v := range m {
				r := []rune(k)
				if len(r) == 1 {
					t.adv[r[0]] = v
				}
			}
		}
	}
}

func (t *previewText) Measure(s string, f Font) int {
	t.loadMetrics()
	w := 0.0
	for _, r := range s {
		if f.Icon || (r >= 0xE000 && r <= 0xF8FF) {
			w += f.Size
			continue
		}
		a, ok := t.adv[r]
		if !ok {
			a = 0.58
		}
		if f.Mono {
			a = 0.6
		}
		w += a * f.Size * (1 + float64(f.Weight-400)/4000)
	}
	return int(w + 0.5)
}

func (t *previewText) LineHeight(f Font) int { return int(f.Size*1.30 + 0.5) }

func (t *previewText) Draw(s string, x, y int, f Font, col Color) {
	if strings.TrimSpace(s) == "" {
		return
	}
	t.ops = append(t.ops, textOp{S: s, X: x, Y: y, Size: f.Size, Weight: f.Weight, Mono: f.Mono, Icon: f.Icon,
		Col: [4]uint8{col.R, col.G, col.B, col.A}, Clip: [4]int{t.clip.X, t.clip.Y, t.clip.W, t.clip.H}})
}

func (t *previewText) SetClip(r Rect) { t.clip = r }
