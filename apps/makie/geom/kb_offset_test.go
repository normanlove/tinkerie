package geom

import (
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRasterOffset(t *testing.T) {
	sq := rectL(0, 0, 100, 40)
	for _, g := range []float64{3, 0.5, -0.3} {
		o := rasterOffset(sq, g)
		want := (100+2*g)*(40+2*g) - (4-math.Pi)*g*g
		if g < 0 {
			want = (100 + 2*g) * (40 + 2*g)
		}
		if a := o.Area(); math.Abs(a-want)/want > 0.01 {
			t.Fatalf("g=%v area %.1f want %.1f (%d pts)", g, a, want, len(o))
		}
	}
	// staggered rows: a concave outline with narrow notches
	l := Loop{{0, 0}, {80, 0}, {80, 19}, {84.8, 19}, {84.8, 38}, {4.8, 38}, {4.8, 19}, {0, 19}}
	o := rasterOffset(l, 2.9)
	if len(o) < 8 || o.Area() < l.Area() {
		t.Fatalf("concave offset broken: %d pts area %.1f", len(o), o.Area())
	}
	for i := range o {
		for j := i + 2; j < len(o); j++ {
			if i == 0 && j == len(o)-1 {
				continue
			}
			if segIntersect(o[i], o[(i+1)%len(o)], o[j], o[(j+1)%len(o)]) {
				t.Fatalf("offset outline crosses itself at %d/%d", i, j)
			}
		}
	}
}

func TestKeyboardCase(t *testing.T) {
	dir := os.Getenv("KB_CASE_DIR")
	if dir == "" {
		t.Skip("set KB_CASE_DIR to a folder of Keebie .makie.json files")
	}
	files, _ := filepath.Glob(filepath.Join(dir, "*.makie.json"))
	for _, f := range files {
		b, _ := os.ReadFile(f)
		c, notes, err := ParseWirie(b)
		if err != nil {
			t.Fatal(f, err)
		}
		if !c.isKeyboard() {
			t.Fatalf("%s: no keyboard data", f)
		}
		st := time.Now()
		nodes, info := c.BuildCase(Params{})
		t.Logf("%s: notes %v\n  lines %v\n  warnings %v", filepath.Base(f), notes, info.Lines, info.Warnings)
		for _, g := range nodes {
			m, err := BuildWorld(g, 1)
			if err != nil {
				t.Fatalf("%s %s: %v", f, g.Name, err)
			}
			lo, hi := m.Bounds()
			t.Logf("  %-40s %6d tris, open edges %d, %v..%v", g.Name, len(m.Tris), openEdgeCount(m), lo, hi)
		}
		t.Logf("  built in %v", time.Since(st))
	}
}
