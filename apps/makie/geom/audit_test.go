package geom

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
)

// variants: defaults, every select option, each bool flipped
func variants(d *ShapeDef) []Params {
	out := []Params{{}}
	for _, p := range d.Params {
		switch p.Type {
		case "select":
			if (p.Key == "board" && len(p.Options) > 20) || p.Key == "tStd" || p.Key == "tHand" || p.Key == "tProfile" || p.Key == "tFit" {
				continue
			}
			for _, o := range p.Options {
				if o != p.Def {
					out = append(out, Params{p.Key: o})
				}
			}
		case "bool":
			b, _ := p.Def.(bool)
			out = append(out, Params{p.Key: !b})
		}
	}
	return out
}

type finding struct {
	name, variant string
	t             Thickness
}

func auditOne(t *testing.T, d *ShapeDef, v Params) *finding {
	p := Params{"_q": 0.8, "threads": false}
	for k, x := range v {
		p[k] = x
	}
	if p["part"] == "Assembled" {
		return nil // assembled machines are checked part by part ("… (print)" variants)
	}
	m, err := Generate(d.Type, p)
	if err != nil && (strings.Contains(err.Error(), "too big") || strings.Contains(err.Error(), "must be smaller")) {
		return nil // the part refuses an unsafe combination — that is the rule working
	}
	if err != nil || m == nil || len(m.Tris) == 0 {
		t.Errorf("%s %v: %v", d.Type, v, err)
		return nil
	}
	th := WallThickness(m, 2500, 1.15, 0.95, false)
	bad := th.CritArea > maxf(2, 0.005*th.TotalArea) || th.ThinArea > maxf(8, 0.03*th.TotalArea)
	if !bad {
		return nil
	}
	return &finding{d.Type, fmt.Sprint(v), th}
}

func maxf(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// Parts that are thin on purpose. Everything else must pass the wall check.
var thinByDesign = map[string]string{
	"washer":       "standard washer — thickness fixed by ISO 7089",
	"lockwasher":   "standard spring washer — thin by design",
	"brassinsert":  "model of a metal insert (for fit checks), not a printed part",
	"cagenut":      "model of a spring-steel cage nut",
	"circlip":      "spring clip — must be thin to flex",
	"rollpin":      "split spring pin — must be thin to flex",
	"ballbearing":  "model of a steel bearing (for fit checks)",
	"bolt":         "ISO bolt dimensions (small sizes are naturally thin)",
	"nut":          "ISO nut dimensions",
	"capnut":       "DIN 1587 dome nut dimensions",
	"sqnut":        "DIN 562 square nut dimensions",
	"wingnut":      "DIN 315 wing nut dimensions",
	"couplingnut":  "DIN 6334 coupling nut dimensions",
	"flangenut":    "DIN 6923 flange nut dimensions",
	"spiralspring": "flat spiral spring — thin to flex",
	"flexure":      "flexure hinge — the web must be thin to bend",
	"pcbpin":       "snap-pin halves must flex to clip into the board hole",
	"pushrivet":    "split legs must flex to push through the hole",
	"tnut":         "neck walls are backed by the aluminium extrusion slot",
	"pdiamond":     "diamond lattice spring — the struts must be thin to flex",
}

func TestAuditThickness(t *testing.T) {
	if testing.Short() {
		t.Skip("slow")
	}
	var fs []finding
	for _, d := range Catalog() {
		if c := os.Getenv("AUDIT_CAT"); c != "" && d.Category != c {
			continue
		}
		if d.Hidden || d.Hole || d.Category == "Basic" || d.Category == "Sketch" || d.Type == "rod" || d.Type == "tscrew" || d.Type == "worm" {
			continue
		}
		for _, v := range variants(d) {
			if f := auditOne(t, d, v); f != nil {
				fs = append(fs, *f)
			}
		}
	}
	sort.Slice(fs, func(i, j int) bool { return fs[i].name < fs[j].name })
	var b strings.Builder
	for _, f := range fs {
		fmt.Fprintf(&b, "%-14s %-40s min %.2f  thin %.0f mm² (%.1f%%)  crit %.0f mm²  spot %v\n", f.name, f.variant, f.t.Min, f.t.ThinArea, 100*f.t.ThinArea/f.t.TotalArea, f.t.CritArea, firstSpot(f.t))
	}
	if out := os.Getenv("AUDIT_OUT"); out != "" {
		os.WriteFile(out, []byte(b.String()), 0o644)
	}
	n := 0
	for _, f := range fs {
		if _, ok := thinByDesign[f.name]; !ok {
			n++
			t.Errorf("thin wall: %s %s min %.2f mm at %s", f.name, f.variant, f.t.Min, firstSpot(f.t))
		}
	}
	t.Logf("%d thin findings (%d intentionally thin)", len(fs), len(fs)-n)
}

func firstSpot(t Thickness) string {
	if len(t.Spots) == 0 {
		return "-"
	}
	s := t.Spots[0]
	return fmt.Sprintf("(%.1f,%.1f,%.1f)=%.2f", s.At[0], s.At[1], s.At[2], s.Thickness)
}
