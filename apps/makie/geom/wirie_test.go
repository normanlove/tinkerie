package geom

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

func loadExample(t *testing.T) *WirieCase {
	b, err := os.ReadFile("testdata/example.makie.json")
	if err != nil {
		t.Fatal(err)
	}
	c, notes, err := ParseWirie(b)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("notes: %v", notes)
	return c
}

func TestWirieParse(t *testing.T) {
	c := loadExample(t)
	if c.Project != "Pico weather station" || len(c.Holes) != 4 || len(c.Parts) != 12 || len(c.Openings) != 3 || len(c.LidFeatures) != 4 {
		t.Fatalf("unexpected contents: %s %d holes %d parts %d openings %d features", c.Project, len(c.Holes), len(c.Parts), len(c.Openings), len(c.LidFeatures))
	}
	if c.Textures == nil || !strings.HasPrefix(c.Textures.Top, "data:image/png") {
		t.Fatal("textures missing")
	}
	if strings.Contains(c.Compact(), "base64") {
		t.Fatal("compact form must not carry textures")
	}
	bad := map[string]string{
		``:                                    "empty",
		`{"format":"wirie-case"`:              "readable JSON",
		`[1,2]`:                               "JSON object",
		`{"app":"Makie","nodes":[]}`:          "Makie project",
		`{"format":"gerber"}`:                 `"gerber"`,
		`{"format":"wirie-case","version":2}`: "newer Wirie",
		`{"format":"wirie-case","version":1,"board":{"width":0,"height":5}}`:    "no size",
		`{"format":"wirie-case","version":1,"board":{"width":"x","height":5}}`:  "can't read",
		`{"format":"wirie-case","version":"1","board":{"width":10,"height":5}}`: "version",
	}
	for in, want := range bad {
		_, _, err := ParseWirie([]byte(in))
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: got %v, want error containing %q", in, err, want)
		}
	}
	// unknown keys are ignored, missing outline / envelope are filled in
	c2, _, err := ParseWirie([]byte(`{"format":"wirie-case","version":1,"future":{"x":1},"board":{"width":40,"height":30,"thickness":1.6,"shape":"rect","cornerRadius":2,"newThing":5},"parts":[{"ref":"U1","kind":"module","side":"top","box":{"x0":5,"y0":5,"x1":15,"y1":15,"z0":1.6,"z1":9}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(c2.Board.Outline) < 8 || c2.Envelope.MaxPartHeightTop < 7 || c2.Envelope.SuggestedFit <= 0 {
		t.Fatalf("defaults not filled: %+v", c2.Envelope)
	}
}

func buildAll(t *testing.T, nodes []*Node) map[string]*Mesh {
	out := map[string]*Mesh{}
	for _, n := range nodes {
		m, err := BuildWorld(n, 1)
		if err != nil {
			t.Fatalf("%s: %v", n.Name, err)
		}
		st := MeshStats(m)
		if st.OpenEdges > 0 {
			t.Errorf("%s: %d open edges", n.Name, st.OpenEdges)
		}
		if st.Volume < 1 {
			t.Errorf("%s: volume %.2f", n.Name, st.Volume)
		}
		out[n.Name] = m
	}
	return out
}

func TestWirieCase(t *testing.T) {
	c := loadExample(t)
	bm := c.BoardMesh(16)
	solid := 0
	for _, p := range c.Parts {
		if partSolid(p) {
			solid++
		}
	}
	if len(bm.Tris) < 12*solid+20 {
		t.Fatal("board mesh too small")
	}
	for _, lid := range lidStyles {
		for _, extra := range []bool{false, true} {
			p := Params{"board": c.Compact(), "lid": lid, "_q": 1.0}
			if extra {
				p["vents"], p["bezel"], p["ledPipes"], p["text"], p["logo"], p["csk"], p["fixing"] = true, true, true, "WEATHER", "Lightning bolt", true, "Heat-set insert"
			}
			// round-trip through JSON like the editor does
			var pp Params
			b, _ := json.Marshal(p)
			_ = json.Unmarshal(b, &pp)
			start := time.Now()
			nodes, info := c.BuildCase(pp)
			ms := buildAll(t, nodes)
			cl, err := c.ClashCheck(nodes, 1)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("%s extras=%v: %d nodes in %v  %v  warn=%v", lid, extra, len(nodes), time.Since(start).Round(time.Millisecond), info.Lines, info.Warnings)
			for _, x := range cl {
				t.Errorf("%s: clash %s with %s", lid, x.Ref, x.With)
			}
			base := ms[nodes[0].Name]
			lo, hi := base.Bounds()
			if hi.X-lo.X < 84 || hi.Y-lo.Y < 64 {
				t.Errorf("base too small: %v %v", lo, hi)
			}
			// the openings must really go through the walls
			for _, op := range c.Openings {
				pt := V(op.X+op.Normal[0]*(0.3+1), op.Y+op.Normal[1]*(0.3+1), op.Z)
				if pointInMesh(base, pt) {
					t.Errorf("%s: opening %s not cut at %v", lid, op.Ref, pt)
				}
			}
			// lid holes over each top feature
			lidM := ms[nodes[1].Name]
			dm := c.dims(caseOpts(c, pp))
			for _, f := range c.LidFeatures {
				if f.Side == "top" && pointInMesh(lidM, V(f.X+0.01, f.Y+0.01, dm.zLidTop-0.4)) {
					t.Errorf("%s: no lid hole for %s", lid, f.Ref)
				}
			}
			// the board sits on the standoffs: solid just under each hole ring
			for _, h := range c.Holes {
				if !pointInMesh(base, V(h.X+h.D/2+0.6, h.Y, -0.3)) {
					t.Errorf("%s: no standoff under %s", lid, h.Ref)
				}
			}
		}
	}
}

func TestWirieClashFound(t *testing.T) {
	c := loadExample(t)
	nodes, _ := c.BuildCase(Params{"board": c.Compact(), "lid": "Snap-fit", "clear": 3.0})
	// clearance is lifted to clear the tallest part, so force a clash with a block
	blk := newNode("Block", "box", Params{"w": 4.0, "d": 4.0, "h": 4.0}, [3]float64{42, 52, 3}, [3]float64{}, false, "")
	cl, err := c.ClashCheck(append(nodes, blk), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(cl) != 1 || cl[0].Ref != "D1" {
		t.Fatalf("expected a clash with D1, got %+v", cl)
	}
}

// the case must pass the same wall check as the part library
func TestWirieCaseWalls(t *testing.T) {
	if testing.Short() {
		t.Skip("slow")
	}
	c := loadExample(t)
	for _, lid := range lidStyles {
		nodes, _ := c.BuildCase(Params{"board": c.Compact(), "lid": lid, "_q": 1.0, "vents": true, "bezel": true, "ledPipes": true, "text": "WX", "fixing": "Heat-set insert"})
		for _, n := range nodes {
			m, err := BuildWorld(n, 1)
			if err != nil {
				t.Fatal(err)
			}
			th := WallThickness(m, 2500, 1.15, 0.95, false)
			t.Logf("%s %-40s min %.2f thin %.1f mm² crit %.1f mm² %s", lid, n.Name, th.Min, th.ThinArea, th.CritArea, firstSpot(th))
			if (th.Min < 1.15 || th.CritArea > 1) && n.Wirie != "extra" {
				t.Errorf("%s: %s has a thin wall (%.2f mm at %s)", lid, n.Name, th.Min, firstSpot(th))
			}
		}
	}
}
