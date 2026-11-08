package geom

import "testing"

func TestAllFonts(t *testing.T) {
	for _, f := range FontNames() {
		m, err := Generate("text", Params{"font": f, "text": "Makie 3D £°é", "_q": 0.8})
		if err != nil || len(m.Tris) == 0 {
			t.Errorf("%s: %v", f, err)
			continue
		}
		st := MeshStats(m)
		if st.OpenEdges > 0 {
			t.Logf("%s open %d", f, st.OpenEdges)
		}
	}
}
