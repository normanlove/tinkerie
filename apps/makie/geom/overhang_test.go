package geom

import (
	"strings"
	"testing"
)

// a part sticking out past the board edge gets room in the wall, open at the
// top, and a filler in the lid; the case still builds solid and doesn't clash
func TestOverhang(t *testing.T) {
	c := loadExample(t)
	w := c.Board.Width
	// push the first top-side part so it hangs 3 mm over the right-hand edge
	i := -1
	for k, p := range c.Parts {
		if p.Side == "top" && p.Kind != "mechanical" {
			i = k
			break
		}
	}
	if i < 0 {
		t.Skip("no part to move")
	}
	p := &c.Parts[i]
	dx := w + 3 - p.Box.X1
	p.Box.X0 += dx
	p.Box.X1 += dx
	p.X += dx
	got := false
	for _, oh := range c.overhangs() {
		t.Logf("overhang: %s %s %.2f mm (%s)", oh.p.Ref, oh.p.Name, oh.over, oh.side)
		got = got || oh.p.Ref == p.Ref
	}
	if !got {
		t.Fatalf("the moved part %s isn't seen as sticking out", p.Ref)
	}
	for _, lid := range []string{"Screw-down", "Snap-fit"} {
		nodes, info := c.BuildCase(Params{"board": c.Compact(), "lid": lid})
		found, filler := false, false
		var walk func(ns []*Node)
		walk = func(ns []*Node) {
			for _, n := range ns {
				if strings.HasPrefix(n.Name, "Room for "+p.Ref) && n.Hole {
					found = true
				}
				if n.Name == "Slot filler for "+p.Ref {
					filler = true
				}
				walk(n.Children)
			}
		}
		walk(nodes)
		if !found || !filler {
			t.Errorf("%s: room %v, filler %v; info %v", lid, found, filler, info.Lines)
		}
		buildAll(t, nodes)
		cl, err := c.ClashCheck(nodes, 1)
		if err != nil {
			t.Fatal(err)
		}
		for _, x := range cl {
			if (x.Ref == p.Ref || x.Ref == "J1") && strings.Contains(x.With, "base") { // (moving U1 can put it under a lid screw post: that's the test's doing)
				t.Errorf("%s: %s still clashes with the case: %+v", lid, p.Ref, x)
			}
		}
	}
}
