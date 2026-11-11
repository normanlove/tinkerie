package geom

import (
	"os"
	"strings"
	"testing"
)

// count the feet and indents a case was given
func countFeet(nodes []*Node) (feet, indents int) {
	var walk func(ns []*Node)
	walk = func(ns []*Node) {
		for _, n := range ns {
			if strings.HasPrefix(n.Name, "Foot ") || strings.HasPrefix(n.Name, "Rear foot") {
				feet++
			}
			if strings.HasPrefix(n.Name, "Rubber foot indent") {
				indents++
				if !n.Hole {
					panic("an indent must cut into the case")
				}
			}
			walk(n.Children)
		}
	}
	walk(nodes)
	return
}

func TestCaseFeet(t *testing.T) {
	c := loadExample(t)
	for _, tc := range []struct {
		feet          string
		feetN, indent int
	}{{"No feet", 0, 0}, {"Printed feet", 4, 0}, {"Rubber feet in indents", 0, 4}} {
		nodes, info := c.BuildCase(Params{"board": c.Compact(), "feet": tc.feet, "footSize": "8 × 2 mm (small round)"})
		f, i := countFeet(nodes)
		for name, m := range buildAll(t, nodes) {
			if len(m.Tris) == 0 {
				t.Errorf("%s: %s built empty", tc.feet, name)
			}
		}
		if f != tc.feetN || i != tc.indent {
			t.Errorf("%s: %d feet, %d indents; want %d, %d (%v)", tc.feet, f, i, tc.feetN, tc.indent, info.Warnings)
		}
	}
	if f, i := countFeet(func() []*Node { n, _ := c.BuildCase(Params{"board": c.Compact()}); return n }()); f+i != 0 {
		t.Errorf("board cases have a flat bottom unless feet are chosen")
	}
}

func TestKeyboardFeet(t *testing.T) {
	dir := os.Getenv("KB_CASE_DIR")
	if dir == "" {
		t.Skip("set KB_CASE_DIR to a folder of Keebie .makie.json files")
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".makie.json") {
			continue
		}
		b, _ := os.ReadFile(dir + "/" + e.Name())
		c, _, err := ParseWirie(b)
		if err != nil || !c.isKeyboard() {
			continue
		}
		nodes, _ := c.BuildCase(Params{})
		if f, i := countFeet(nodes); f != 2 || i < 4 { // two tilt feet at the back, rubber indents under all
			t.Errorf("%s default: %d feet, %d indents", e.Name(), f, i)
		}
		nodes, info := c.BuildCase(Params{"feet": "No feet"})
		if f, i := countFeet(nodes); f+i != 0 {
			t.Errorf("%s no feet: %d feet, %d indents", e.Name(), f, i)
		}
		_ = info
		return
	}
	t.Skip("no keyboard file found")
}
