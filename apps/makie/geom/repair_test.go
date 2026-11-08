package geom

import "testing"

func TestRepairAllQ(t *testing.T) {
	for _, q := range []float64{1, 1.6} {
		for _, d := range Catalog() {
			if d.Hidden { // built from board data, tested in wirie_test.go
				continue
			}
			p := Params{"_q": q}
			if d.Type == "gear" {
				p["style"] = "Herringbone"
				p["bore"] = 8.0
				p["teeth"] = 24.0
			}
			m, err := Generate(d.Type, p)
			if err != nil {
				t.Fatal(err)
			}
			r := m.FixTJunctions()
			if o := MeshStats(r).OpenEdges; o > 0 {
				t.Logf("q=%.1f %-12s open after repair: %d (before %d)", q, d.Type, o, MeshStats(m).OpenEdges)
			}
		}
	}
}
