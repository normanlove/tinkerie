package geom

import (
	"fmt"
	"math"
	"math/rand"
)

// ===================== Patterns =====================
// Decorative and plastic-saving patterns: Voronoi, honeycomb, diamonds … as a cut-out
// (group it with any part to perforate it), a ready-made panel, or a raised surface texture.

var patternKinds = []string{"Voronoi", "Voronoi (organic)", "Honeycomb", "Circles", "Squares", "Diamonds", "Triangles", "Bricks", "Slots", "Waves", "Stars", "Radial"}

type hp struct{ nx, ny, c float64 } // half-plane nx*x + ny*y <= c

func clipHalf(poly Loop, h hp) Loop {
	if len(poly) == 0 {
		return poly
	}
	var out Loop
	n := len(poly)
	for i := 0; i < n; i++ {
		a, b := poly[i], poly[(i+1)%n]
		da := h.nx*a.X + h.ny*a.Y - h.c
		db := h.nx*b.X + h.ny*b.Y - h.c
		if da <= 0 {
			out = append(out, a)
		}
		if (da < 0 && db > 0) || (da > 0 && db < 0) {
			t := da / (da - db)
			out = append(out, P2{a.X + (b.X-a.X)*t, a.Y + (b.Y-a.Y)*t})
		}
	}
	return out
}

// insetConvex shrinks a CCW convex polygon by d
func insetConvex(poly Loop, d float64) Loop {
	if poly.Area() < 0 {
		poly = poly.Reversed()
	}
	out := poly
	n := len(poly)
	for i := 0; i < n && len(out) > 0; i++ {
		a, b := poly[i], poly[(i+1)%n]
		ex, ey := b.X-a.X, b.Y-a.Y
		l := math.Hypot(ex, ey)
		if l < 1e-9 {
			continue
		}
		// outward normal of a CCW polygon is (ey, -ex)
		nx, ny := ey/l, -ex/l
		out = clipHalf(out, hp{nx, ny, nx*a.X + ny*a.Y - d})
	}
	return out
}

func chaikin(l Loop, iters int) Loop {
	for k := 0; k < iters; k++ {
		var o Loop
		n := len(l)
		for i := 0; i < n; i++ {
			a, b := l[i], l[(i+1)%n]
			o = append(o, P2{a.X*0.75 + b.X*0.25, a.Y*0.75 + b.Y*0.25}, P2{a.X*0.25 + b.X*0.75, a.Y*0.25 + b.Y*0.75})
		}
		l = o
	}
	return l
}

type patternSpec struct {
	kind        string
	w, d        float64 // region size (rect) or Ø (round)
	round       bool
	border      float64
	cell, strut float64
	seed        int64
	smooth      bool
}

// region boundary as half-planes (rect, or a 64-gon for round)
func (ps patternSpec) regionPlanes(inset float64) []hp {
	if ps.round {
		r := ps.w/2 - inset
		var out []hp
		for i := 0; i < 64; i++ {
			a := 2 * math.Pi * (float64(i) + 0.5) / 64
			out = append(out, hp{math.Cos(a), math.Sin(a), r * math.Cos(math.Pi/64)})
		}
		return out
	}
	X, Y := ps.w/2-inset, ps.d/2-inset
	return []hp{{1, 0, X}, {-1, 0, X}, {0, 1, Y}, {0, -1, Y}}
}

func (ps patternSpec) inRegion(x, y, m float64) bool {
	if ps.round {
		return math.Hypot(x, y) < ps.w/2-m
	}
	return math.Abs(x) < ps.w/2-m && math.Abs(y) < ps.d/2-m
}

func (ps patternSpec) clipRegion(l Loop) Loop {
	for _, h := range ps.regionPlanes(ps.border) {
		l = clipHalf(l, h)
		if len(l) < 3 {
			return nil
		}
	}
	return l
}

// cells returns the holes of the pattern (each a simple loop)
func (ps patternSpec) cells() ([]Loop, error) {
	s := ps.cell
	t := ps.strut
	W, D := ps.w, ps.d
	if ps.round {
		D = W
	}
	if s < t*1.5 {
		return nil, fmt.Errorf("cells must be bigger than the struts")
	}
	if (W/s)*(D/s) > 4000 {
		return nil, fmt.Errorf("too many cells — make them bigger or the area smaller")
	}
	var out []Loop
	add := func(l Loop, convex bool) {
		if convex {
			l = insetConvex(l, t/2)
			if len(l) < 3 {
				return
			}
			l = ps.clipRegion(l)
		} else {
			// non-convex shapes are only kept if they sit fully inside
			for _, q := range l {
				if !ps.inRegion(q.X, q.Y, ps.border) {
					return
				}
			}
		}
		if len(l) < 3 || math.Abs(l.Area()) < t*t*0.8 {
			return
		}
		if ps.smooth {
			l = chaikin(l, 2)
		}
		if l.Area() < 0 {
			l = l.Reversed()
		}
		out = append(out, l)
	}
	rect := func(cx, cy, hw, hh float64) Loop {
		return Loop{{cx - hw, cy - hh}, {cx + hw, cy - hh}, {cx + hw, cy + hh}, {cx - hw, cy + hh}}
	}
	hx, hy := W/2+s, D/2+s
	switch ps.kind {
	case "Voronoi", "Voronoi (organic)":
		rng := rand.New(rand.NewSource(ps.seed))
		var pts []P2
		for y := -hy; y <= hy; y += s {
			for x := -hx; x <= hx; x += s {
				pts = append(pts, P2{x + (rng.Float64()-0.5)*s*0.9, y + (rng.Float64()-0.5)*s*0.9})
			}
		}
		cell := func(i int, d float64) Loop {
			si := pts[i]
			l := rect(si.X, si.Y, s*3, s*3)
			for j, sj := range pts {
				if j == i || math.Abs(sj.X-si.X) > 3*s || math.Abs(sj.Y-si.Y) > 3*s {
					continue
				}
				nx, ny := sj.X-si.X, sj.Y-si.Y
				ln := math.Hypot(nx, ny)
				if ln < 1e-9 {
					continue
				}
				c := (sj.X*sj.X + sj.Y*sj.Y - si.X*si.X - si.Y*si.Y) / 2
				l = clipHalf(l, hp{nx / ln, ny / ln, c/ln - d})
				if len(l) < 3 {
					return nil
				}
			}
			return l
		}
		// two rounds of Lloyd relaxation for even cell sizes
		for it := 0; it < 2; it++ {
			for i := range pts {
				l := cell(i, 0)
				if len(l) < 3 {
					continue
				}
				var cx, cy, a float64
				for k := range l {
					p, q := l[k], l[(k+1)%len(l)]
					cr := p.X*q.Y - q.X*p.Y
					a += cr
					cx += (p.X + q.X) * cr
					cy += (p.Y + q.Y) * cr
				}
				if math.Abs(a) > 1e-9 {
					pts[i] = P2{cx / (3 * a), cy / (3 * a)}
				}
			}
		}
		for i := range pts {
			if !ps.inRegion(pts[i].X, pts[i].Y, -s) {
				continue
			}
			l := cell(i, t/2)
			if len(l) >= 3 {
				l = ps.clipRegion(l)
				if len(l) >= 3 && math.Abs(l.Area()) > t*t {
					if ps.kind == "Voronoi (organic)" || ps.smooth {
						l = chaikin(l, 3)
					}
					if l.Area() < 0 {
						l = l.Reversed()
					}
					out = append(out, l)
				}
			}
		}
	case "Honeycomb":
		r := s / math.Sqrt(3)
		dx, dy := s, s*math.Sqrt(3)/2
		for j, y := 0, -hy; y <= hy; j, y = j+1, y+dy {
			off := 0.0
			if j%2 == 1 {
				off = dx / 2
			}
			for x := -hx; x <= hx; x += dx {
				hex := NGon(r, 6, 30)
				for k := range hex {
					hex[k].X += x + off
					hex[k].Y += y
				}
				add(hex, true)
			}
		}
	case "Squares":
		for y := -hy; y <= hy; y += s {
			for x := -hx; x <= hx; x += s {
				add(rect(x, y, s/2, s/2), true)
			}
		}
	case "Diamonds":
		q := s / math.Sqrt2
		for j, y := 0, -hy; y <= hy; j, y = j+1, y+q {
			off := 0.0
			if j%2 == 1 {
				off = q
			}
			for x := -hx; x <= hx; x += 2 * q {
				add(Loop{{x + off, y - q}, {x + off + q, y}, {x + off, y + q}, {x + off - q, y}}, true)
			}
		}
	case "Triangles":
		h := s * math.Sqrt(3) / 2
		for j, y := 0, -hy; y <= hy; j, y = j+1, y+h {
			for i, x := 0, -hx; x <= hx; i, x = i+1, x+s/2 {
				up := (i+j)%2 == 0
				if up {
					add(Loop{{x - s/2, y}, {x + s/2, y}, {x, y + h}}, true)
				} else {
					add(Loop{{x - s/2, y + h}, {x, y}, {x + s/2, y + h}}, true)
				}
			}
		}
	case "Bricks":
		bh := s / 2
		for j, y := 0, -hy; y <= hy; j, y = j+1, y+bh {
			off := 0.0
			if j%2 == 1 {
				off = s / 2
			}
			for x := -hx; x <= hx; x += s {
				add(rect(x+off, y, s/2, bh/2), true)
			}
		}
	case "Circles":
		dx, dy := s, s*math.Sqrt(3)/2
		for j, y := 0, -hy; y <= hy; j, y = j+1, y+dy {
			off := 0.0
			if j%2 == 1 {
				off = dx / 2
			}
			for x := -hx; x <= hx; x += dx {
				add(moveLoop(Circle(s/2-t/2, 24), x+off, y), true)
			}
		}
	case "Slots":
		dy := s / 2
		for j, y := 0, -hy; y <= hy; j, y = j+1, y+dy {
			off := 0.0
			if j%2 == 1 {
				off = s
			}
			for x := -hx; x <= hx; x += 2 * s {
				add(moveLoop(stadium(2*s-t, dy-t, 12), x+off, y), false)
			}
		}
	case "Waves":
		amp := s * 0.3
		for y := -hy; y <= hy; y += s / 2 {
			var top, bot Loop
			x0, x1 := -W/2+ps.border+t, W/2-ps.border-t
			if ps.round {
				r := W/2 - ps.border - t
				if math.Abs(y) >= r {
					continue
				}
				hw := math.Sqrt(r*r-y*y) - amp
				x0, x1 = -hw, hw
			}
			if x1-x0 < s {
				continue
			}
			n := int((x1 - x0) / 1.5)
			hw := (s/2 - t) / 2
			for i := 0; i <= n; i++ {
				x := x0 + (x1-x0)*float64(i)/float64(n)
				c := y + amp*math.Sin(2*math.Pi*x/(2*s))
				bot = append(bot, P2{x, c - hw})
				top = append(top, P2{x, c + hw})
			}
			l := append(bot, top.Reversed()...)
			add(l, false)
		}
	case "Stars":
		dx, dy := s, s*math.Sqrt(3)/2
		for j, y := 0, -hy; y <= hy; j, y = j+1, y+dy {
			off := 0.0
			if j%2 == 1 {
				off = dx / 2
			}
			for x := -hx; x <= hx; x += dx {
				var l Loop
				ro := s/2 - t/2
				for k := 0; k < 10; k++ {
					r := ro
					if k%2 == 1 {
						r = ro * 0.45
					}
					a := math.Pi/2 + math.Pi*float64(k)/5
					l = append(l, P2{x + off + r*math.Cos(a), y + r*math.Sin(a)})
				}
				add(l, false)
			}
		}
	case "Radial":
		R := math.Max(W, D) / 2
		for r := s / 2; r < R; r += s {
			n := int(math.Max(6, math.Round(2*math.Pi*(r+s/2)/s)))
			for k := 0; k < n; k++ {
				a0 := 2 * math.Pi * float64(k) / float64(n)
				a1 := 2 * math.Pi * float64(k+1) / float64(n)
				r0, r1 := r, r+s
				var l Loop
				for q := 0; q <= 4; q++ {
					a := a0 + (a1-a0)*float64(q)/4
					l = append(l, P2{r0 * math.Cos(a), r0 * math.Sin(a)})
				}
				for q := 4; q >= 0; q-- {
					a := a0 + (a1-a0)*float64(q)/4
					l = append(l, P2{r1 * math.Cos(a), r1 * math.Sin(a)})
				}
				add(l, true)
			}
		}
	default:
		return nil, fmt.Errorf("unknown pattern %q", ps.kind)
	}
	var clean []Loop
	for _, l := range out {
		if l = tidyLoop(l); l != nil {
			clean = append(clean, l)
		}
	}
	return clean, nil
}

// tidyLoop drops points that are almost on top of each other or in a straight line —
// clipping leaves those behind and they upset the triangulator.
func tidyLoop(l Loop) Loop {
	for pass := 0; pass < 3; pass++ {
		var out Loop
		for i, p := range l {
			q := l[(i+1)%len(l)]
			if math.Hypot(p.X-q.X, p.Y-q.Y) > 0.05 {
				out = append(out, p)
			}
		}
		if len(out) < 3 {
			return nil
		}
		var res Loop
		for i := range out {
			a, b, c := out[(i+len(out)-1)%len(out)], out[i], out[(i+1)%len(out)]
			cr := (b.X-a.X)*(c.Y-a.Y) - (b.Y-a.Y)*(c.X-a.X)
			if math.Abs(cr) > 1e-4*math.Hypot(c.X-a.X, c.Y-a.Y) {
				res = append(res, b)
			}
		}
		l = res
		if len(l) < 3 {
			return nil
		}
	}
	return l
}

func patternParams(extra ...Param) []Param {
	ps := []Param{
		sel("kind", "Pattern", "Voronoi", patternKinds...),
		sel("shape", "Area", "Rectangle", "Rectangle", "Round"),
		num("w", "Width (or Ø)", 60, 5, 1000, 1, "mm"),
		num("d", "Depth", 40, 5, 1000, 1, "mm").show("shape=Rectangle"),
		num("cell", "Cell size", 10, 2, 200, 0.5, "mm"),
		num("strut", "Strut width", 1.6, 0.8, 20, 0.1, "mm").help("1.2 mm or more prints strongly with a 0.4 mm nozzle."),
		num("seed", "Variation", 1, 1, 999, 1, "").show("kind=Voronoi;kind=Voronoi (organic)").help("Change for a different random layout."),
		boolp("smooth", "Rounded corners", false),
	}
	return append(ps, extra...)
}

func patternFrom(p Params) patternSpec {
	return patternSpec{kind: p.S("kind", "Voronoi"), w: p.F("w", 60), d: p.F("d", 40), round: p.S("shape", "Rectangle") == "Round",
		border: p.F("border", 0), cell: p.F("cell", 10), strut: math.Max(0.8, p.F("strut", 1.6)), seed: int64(p.F("seed", 1)), smooth: p.B("smooth", false)}
}

func prismsOf(loops []Loop, z0, z1 float64) *Mesh {
	m := &Mesh{}
	for _, l := range loops {
		m.Append(extrudeAt(Shape{Outer: l}, z0, z1))
	}
	return m
}

func init() {
	register(&ShapeDef{Type: "patterncut", Name: "Pattern cut-out", Category: "Holes", Hole: true, Color: "#64748b",
		Desc:   "Voronoi, honeycomb and other patterns as a hole — group it with a plate, a box side or a lid to make it lighter, save filament and look great.",
		Tags:   "pattern voronoi honeycomb hex perforated lightweight infill grille vent texture lattice",
		Params: patternParams(num("depth", "Cut depth", 10, 0.4, 500, 0.5, "mm"), num("border", "Solid border", 0, 0, 100, 0.5, "mm")),
		Gen: func(p Params) (*Mesh, error) {
			ps := patternFrom(p)
			cells, err := ps.cells()
			if err != nil {
				return nil, err
			}
			m := prismsOf(cells, -p.F("depth", 10), over)
			m.Info = fmt.Sprintf("%d openings.", len(cells))
			return m, nil
		}})
	register(&ShapeDef{Type: "patternpanel", Name: "Pattern panel", Category: "Basic", Color: "#0ea5e9",
		Desc:   "A plate filled with a pattern — Voronoi, honeycomb, waves … Great for grilles, vents, lamp shades and light-weight panels.",
		Tags:   "pattern voronoi honeycomb panel grille vent mesh screen lattice",
		Params: patternParams(num("t", "Thickness", 2.4, 0.6, 100, 0.2, "mm"), num("border", "Frame width", 3, 0, 100, 0.5, "mm")),
		Gen: func(p Params) (*Mesh, error) {
			ps := patternFrom(p)
			cells, err := ps.cells()
			if err != nil {
				return nil, err
			}
			var outer Loop
			if ps.round {
				outer = Circle(ps.w/2, p.Segs(ps.w))
			} else {
				outer = Loop{{-ps.w / 2, -ps.d / 2}, {ps.w / 2, -ps.d / 2}, {ps.w / 2, ps.d / 2}, {-ps.w / 2, ps.d / 2}}
			}
			th := p.F("t", 2.4)
			m := Extrude(Shape{Outer: outer}, th, 0, 1, 1)
			m = Subtract(m, prismsOf(cells, -1, th+1))
			solid := 0.0
			for _, c := range cells {
				solid += math.Abs(c.Area())
			}
			m.Info = fmt.Sprintf("%.0f %% lighter than a solid plate.", 100*solid/math.Abs(outer.Area()))
			return m, nil
		}})
	register(&ShapeDef{Type: "texture", Name: "Surface texture", Category: "Basic", Color: "#a855f7",
		Desc:   "A raised pattern — stones, scales, tiles, bricks, grip diamonds — to put on any surface. Group it with the part to fuse them.",
		Tags:   "texture surface pattern emboss grip knurl stone tiles bricks scales",
		Params: patternParams(num("h", "Height", 0.8, 0.2, 10, 0.1, "mm"), num("base", "Backing sheet", 0, 0, 10, 0.2, "mm").help("0 = just the raised tiles")),
		Gen: func(p Params) (*Mesh, error) {
			ps := patternFrom(p)
			cells, err := ps.cells()
			if err != nil {
				return nil, err
			}
			base := p.F("base", 0)
			m := prismsOf(cells, base-0.01, base+p.F("h", 0.8))
			if base > 0 {
				var outer Loop
				if ps.round {
					outer = Circle(ps.w/2, p.Segs(ps.w))
				} else {
					outer = Loop{{-ps.w / 2, -ps.d / 2}, {ps.w / 2, -ps.d / 2}, {ps.w / 2, ps.d / 2}, {-ps.w / 2, ps.d / 2}}
				}
				if len(cells) < 250 {
					m = Union(extrudeAt(Shape{Outer: outer}, 0, base), m)
				} else {
					m.Append(extrudeAt(Shape{Outer: outer}, 0, base))
				}
			}
			return m, nil
		}})
}
