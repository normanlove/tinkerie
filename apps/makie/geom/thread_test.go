package geom

import (
	"testing"
	"time"
)

// a threaded solid and a threaded hole of the same nominal size must not collide
func TestThreadMate(t *testing.T) {
	for _, prof := range threadProfiles {
		for _, starts := range []float64{1, 2} {
			start := time.Now()
			p := Params{"tFit": "Normal (0.35 mm)", "d": 16.0, "h": 12.0, "thread": "Threaded", "tProfile": prof, "tStarts": starts, "_q": 1.0}
			screw, _ := Generate("cylinder", p)
			ph := Params{}
			for k, v := range p {
				ph[k] = v
			}
			ph["_hole"] = true
			cutter, _ := Generate("cylinder", ph)
			t0 := time.Now()
			nut := Subtract(boxAt(V(-14, -14, 0), V(14, 14, 12)), cutter)
			_ = nut
			t.Logf("nut subtract took %v (%d tris)", time.Since(t0).Round(time.Millisecond), len(cutter.Tris))
			clash := Intersect(screw, Subtract(boxAt(V(-14, -14, 0.5), V(14, 14, 11.5)), cutter)).Volume()
			t.Logf("%-16s starts=%v screw %.0f mm³ clash %.3f mm³ (%v)", prof, starts, screw.Volume(), clash, time.Since(start).Round(time.Millisecond))
			if clash > 1 {
				t.Errorf("%s: screw and nut overlap by %.2f mm³", prof, clash)
			}
		}
	}
}
