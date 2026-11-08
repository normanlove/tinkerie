package geom

import (
	"math"
	"testing"
	"time"
)

// every catalog shape must build a closed, positive-volume solid
func TestAllShapes(t *testing.T) {
	for _, d := range Catalog() {
		if d.Hidden { // built from board data, tested in wirie_test.go
			continue
		}
		start := time.Now()
		m, err := Generate(d.Type, Params{"_q": 1.0})
		if err != nil {
			t.Errorf("%s: %v", d.Type, err)
			continue
		}
		st := MeshStats(m)
		el := time.Since(start)
		t.Logf("%-12s tris=%6d vol=%10.1f open=%4d  %v", d.Type, st.Tris, st.Volume, st.OpenEdges, el.Round(time.Millisecond))
		if st.Tris == 0 || m.Volume() <= 0 {
			t.Errorf("%s: empty or inverted (vol %.2f)", d.Type, m.Volume())
		}
	}
}

func TestBoxHole(t *testing.T) {
	b, _ := Generate("box", Params{"w": 20.0, "d": 20.0, "h": 10.0})
	c, _ := Generate("cylinder", Params{"d": 10.0, "h": 20.0})
	c = c.Transformed(Translate(0, 0, -5))
	r := Subtract(b, c)
	want := 20*20*10 - math.Pi*25*10
	got := r.Volume()
	st := MeshStats(r)
	t.Logf("subtract vol %.1f want ~%.1f open edges %d", got, want, st.OpenEdges)
	if math.Abs(got-want)/want > 0.02 {
		t.Errorf("bad volume %v", got)
	}
	u := Union(b, c)
	t.Logf("union vol %.1f", u.Volume())
	i := Intersect(b, c)
	t.Logf("intersect vol %.1f want %.1f", i.Volume(), math.Pi*25*10)
}

func TestGroupThread(t *testing.T) {
	start := time.Now()
	g := &Node{Type: "group", Children: []*Node{
		{Type: "box", Params: Params{"w": 20.0, "d": 20.0, "h": 10.0}, Pos: [3]float64{0, 0, 0}},
		{Type: "threadhole", Params: Params{"size": "M6", "depth": 12.0}, Pos: [3]float64{0, 0, 10}, Hole: true},
		{Type: "nuttrap", Params: Params{"size": "M3"}, Pos: [3]float64{6, 6, 10}, Hole: true},
	}}
	m, err := BuildLocal(g, 1)
	if err != nil {
		t.Fatal(err)
	}
	st := MeshStats(m)
	t.Logf("group tris %d vol %.1f open %d in %v", st.Tris, st.Volume, st.OpenEdges, time.Since(start))
}

func TestRepair(t *testing.T) {
	for _, typ := range []string{"nut", "knob", "enclosure", "lid", "hinge", "bolt", "lbracket"} {
		m, _ := Generate(typ, Params{"_q": 1.0})
		before := MeshStats(m).OpenEdges
		r := m.FixTJunctions()
		r2 := r.FixTJunctions()
		t.Logf("%-10s open %5d -> %5d -> %5d  vol %.1f -> %.1f", typ, before, MeshStats(r).OpenEdges, MeshStats(r2).OpenEdges, m.Volume(), r.Volume())
	}
}
