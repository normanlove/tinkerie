package geom

import (
	"testing"
	"time"
)

func TestEveryBoardEveryMode(t *testing.T) {
	fails := 0
	start := time.Now()
	slowest, slowName := time.Duration(0), ""
	for _, name := range BoardNames() {
		for _, mode := range append(fitModes, "Mounting holes") {
			for _, fix := range []string{"Self-tapping screw", "Through bolt + nut"} {
				t0 := time.Now()
				p := Params{"board": name, "_q": 1.0, "fix": fix}
				b := boardFromParams(p)
				var m *Mesh
				func() {
					defer func() {
						if r := recover(); r != nil {
							t.Errorf("%s / %s: panic %v", name, mode, r)
							fails++
						}
					}()
					m = FitMesh(b, mode, p)
				}()
				if m == nil {
					continue
				}
				if el := time.Since(t0); el > slowest {
					slowest, slowName = el, name+" / "+mode
				}
				if len(m.Tris) == 0 || m.Volume() <= 0 {
					t.Errorf("%s / %s: empty (vol %.1f)", name, mode, m.Volume())
					fails++
				}
			}
		}
	}
	t.Logf("%d boards × %d modes in %v, slowest %v (%s), %d failures", len(BoardNames()), len(fitModes)+1, time.Since(start).Round(time.Millisecond), slowest.Round(time.Millisecond), slowName, fails)
}
