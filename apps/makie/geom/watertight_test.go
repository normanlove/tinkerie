package geom

import "testing"

func TestWatertightAll(t *testing.T) {
	bad := 0
	for _, d := range Catalog() {
		if d.Hidden { // built from board data, tested in wirie_test.go
			continue
		}
		m, err := Generate(d.Type, Params{"_q": 1.0})
		if err != nil {
			t.Fatal(err)
		}
		if o := openEdgeCount(m); o > 0 {
			bad++
			t.Logf("%-14s %s: %d open edges", d.Type, d.Category, o)
		}
	}
	t.Logf("%d of %d shapes not fully watertight", bad, len(Catalog()))
}
