package geom

import (
	_ "embed"
	"encoding/json"
	"math"
	"sort"
	"strings"
	"sync"
)

//go:embed fonts/fonts.json
var fontJSON []byte

type glyph struct {
	A float64     `json:"a"`
	L [][]float64 `json:"l"`
}

var (
	fontOnce sync.Once
	fonts    map[string]map[string]glyph
)

func loadFonts() {
	fontOnce.Do(func() {
		fonts = map[string]map[string]glyph{}
		_ = json.Unmarshal(fontJSON, &fonts)
	})
}

// FontNames lists embedded fonts.
func FontNames() []string {
	loadFonts()
	var n []string
	for k := range fonts {
		n = append(n, k)
	}
	sort.Strings(n)
	return n
}

// TextShapes lays out text and returns 2D shapes (units: mm).
func TextShapes(text, font string, size, spacing, lineGap float64, align string) []Shape {
	loadFonts()
	f, ok := fonts[font]
	if !ok {
		f = fonts["Sans Bold"]
	}
	lines := strings.Split(text, "\n")
	type placed struct {
		loops []Loop
	}
	var all []Shape
	lh := size * (1.2 + lineGap)
	for li, line := range lines {
		var loops []Loop
		x := 0.0
		for _, r := range line {
			g, ok := f[string(r)]
			if !ok {
				g, ok = f["?"]
				if !ok {
					continue
				}
			}
			for _, fl := range g.L {
				var l Loop
				for i := 0; i+1 < len(fl); i += 2 {
					l = append(l, P2{x + fl[i]*size, fl[i+1] * size})
				}
				loops = append(loops, l)
			}
			x += g.A*size + spacing
		}
		width := x - spacing
		dx := -width / 2
		switch align {
		case "Left":
			dx = 0
		case "Right":
			dx = -width
		}
		dy := -float64(li) * lh
		for _, s := range ShapesFromLoops(loops) {
			s = offsetShape(s, dx, dy)
			all = append(all, s)
		}
	}
	return all
}

func offsetShape(s Shape, dx, dy float64) Shape {
	mv := func(l Loop) Loop {
		o := make(Loop, len(l))
		for i, p := range l {
			o[i] = P2{p.X + dx, p.Y + dy}
		}
		return o
	}
	r := Shape{Outer: mv(s.Outer)}
	for _, h := range s.Holes {
		r.Holes = append(r.Holes, mv(h))
	}
	return r
}

func init() {
	register(&ShapeDef{Type: "text", Name: "Text", Category: "Basic", Color: "#e11d48",
		Desc: "3D lettering. Make it a hole to engrave text into a part.", Tags: "text letters label engrave emboss name",
		Axes: map[string]string{"z": "h"},
		Params: []Param{
			{Key: "text", Label: "Text", Type: "text", Def: "Hello"},
			sel("font", "Font", "Sans Bold", FontNames()...),
			num("size", "Letter size", 10, 1, 500, 0.5, "mm"),
			num("h", "Thickness", 2, 0.2, 200, 0.2, "mm"),
			num("spacing", "Letter spacing", 0, -5, 50, 0.1, "mm"),
			sel("align", "Align", "Center", "Left", "Center", "Right"),
		},
		Gen: func(p Params) (*Mesh, error) {
			shapes := TextShapes(p.S("text", "Hello"), p.S("font", "Sans Bold"), p.F("size", 10), p.F("spacing", 0), 0, p.S("align", "Center"))
			m := &Mesh{}
			h := p.F("h", 2)
			for _, s := range shapes {
				m.Append(Extrude(s, h, 0, 1, 1))
			}
			if len(m.Tris) == 0 {
				return boxAt(V(-0.5, -0.5, 0), V(0.5, 0.5, h)), nil
			}
			// centre vertically on the capital height
			lo, hi := m.Bounds()
			return m.Transformed(Translate(0, -(lo.Y+hi.Y)/2, 0)), nil
		}})

	register(&ShapeDef{Type: "sketch", Name: "Custom shape (sketch)", Category: "Sketch", Color: "#0d9488",
		Desc: "Draw any outline in 2D, then extrude it upward or spin it around (revolve) to make vases, knobs and pulleys.",
		Tags: "sketch draw polygon extrude revolve profile custom outline lathe",
		Params: []Param{
			{Key: "points", Label: "Outline", Type: "points", Def: []any{[]any{-10.0, -10.0}, []any{10.0, -10.0}, []any{10.0, 5.0}, []any{0.0, 12.0}, []any{-10.0, 5.0}}},
			sel("mode", "Make 3D by", "Extrude", "Extrude", "Revolve"),
			boolp("smooth", "Smooth curves", false),
			num("h", "Extrude height", 10, 0.1, 2000, 0.5, "mm").show("mode=Extrude"),
			num("twist", "Twist", 0, -720, 720, 5, "°").show("mode=Extrude"),
			num("taper", "Top scale", 100, 0, 500, 5, "%").show("mode=Extrude"),
			num("angle", "Revolve angle", 360, 1, 360, 5, "°").show("mode=Revolve"),
		},
		Gen: func(p Params) (*Mesh, error) {
			pts := p.Pts("points")
			if len(pts) < 3 {
				return boxAt(V(-5, -5, 0), V(5, 5, 5)), nil
			}
			l := Loop(pts)
			if p.B("smooth", false) {
				l = SmoothLoop(l, int(math.Max(4, 8*p.Q())), true)
			}
			if p.S("mode", "Extrude") == "Revolve" {
				// sketch X = radius (|x|), Y = height; use right half of the outline
				var rl Loop
				for _, q := range l {
					rl = append(rl, P2{math.Max(math.Abs(q.X), 0), q.Y})
				}
				if math.Abs(rl.Area()) < 1e-6 {
					return boxAt(V(-5, -5, 0), V(5, 5, 5)), nil
				}
				lo := math.MaxFloat64
				for _, q := range rl {
					lo = math.Min(lo, q.Y)
				}
				for i := range rl {
					rl[i].Y -= lo
				}
				return Revolve(Shape{Outer: rl}, p.F("angle", 360), p.Segs(60)), nil
			}
			tw := p.F("twist", 0)
			sc := p.F("taper", 100) / 100
			lay := 1
			if tw != 0 || sc != 1 {
				lay = int(math.Max(4, math.Abs(tw)/4))
			}
			shapes := ShapesFromLoops([]Loop{l})
			if len(shapes) == 0 {
				return boxAt(V(-5, -5, 0), V(5, 5, 5)), nil
			}
			return Extrude(shapes[0], p.F("h", 10), tw, sc, lay), nil
		}})
}
