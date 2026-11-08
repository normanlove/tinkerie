package geom

import (
	"testing"
	"time"
)

func TestPatterns(t *testing.T) {
	for _, k := range patternKinds {
		for _, sh := range []string{"Rectangle", "Round"} {
			t0 := time.Now()
			m, err := Generate("patternpanel", Params{"kind": k, "shape": sh, "_q": 0.8})
			if err != nil {
				t.Errorf("%s %s: %v", k, sh, err)
				continue
			}
			st := MeshStats(m)
			th := WallThickness(m, 1500, 1.15, 0.95, false)
			t.Logf("%-18s %-9s tris=%6d open=%3d min=%.2f %s %v", k, sh, len(m.Tris), st.OpenEdges, th.Min, m.Info, time.Since(t0).Round(time.Millisecond))
		}
	}
}
