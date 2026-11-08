package geom

import (
	"math"
)

// Loft joins rings of equal vertex count; caps are fans from the ring centroid.
func Loft(rings [][]Vec3, capStart, capEnd bool) *Mesh {
	m := &Mesh{}
	for k := 0; k+1 < len(rings); k++ {
		a, b := rings[k], rings[k+1]
		n := len(a)
		for i := 0; i < n; i++ {
			j := (i + 1) % n
			m.AddQuad(a[i], a[j], b[j], b[i])
		}
	}
	fan := func(r []Vec3, rev bool) {
		var c Vec3
		for _, p := range r {
			c = c.Add(p)
		}
		c = c.Mul(1 / float64(len(r)))
		for i := range r {
			j := (i + 1) % len(r)
			if rev {
				m.Add(c, r[j], r[i])
			} else {
				m.Add(c, r[i], r[j])
			}
		}
	}
	if capStart {
		fan(rings[0], true)
	}
	if capEnd {
		fan(rings[len(rings)-1], false)
	}
	return m.Orient()
}

// rrRing makes a rounded-rectangle ring with a fixed vertex count at height z.
func rrRing(w, d, r float64, segs int, z float64) []Vec3 {
	r = math.Max(0, math.Min(r, math.Min(w, d)/2))
	w, d = math.Max(w, 1e-4), math.Max(d, 1e-4)
	var out []Vec3
	cs := [][3]float64{{w/2 - r, -d/2 + r, -90}, {w/2 - r, d/2 - r, 0}, {-w/2 + r, d/2 - r, 90}, {-w/2 + r, -d/2 + r, 180}}
	for _, c := range cs {
		for i := 0; i <= segs; i++ {
			a := (c[2] + 90*float64(i)/float64(segs)) * math.Pi / 180
			out = append(out, V(c[0]+r*math.Cos(a), c[1]+r*math.Sin(a), z))
		}
	}
	return out
}

// roundedBlock builds a box with optional vertical edge radius and rounded/chamfered top and bottom edges.
func roundedBlock(w, d, h, rv, rTop, rBot float64, chamTop, chamBot bool, segs int) *Mesh {
	lim := math.Min(w, d) / 2
	rv = clampF(rv, 0, lim-1e-3)
	rTop = clampF(rTop, 0, math.Min(lim, h/2)-1e-3)
	rBot = clampF(rBot, 0, math.Min(lim, h/2)-1e-3)
	if rTop < 0 {
		rTop = 0
	}
	if rBot < 0 {
		rBot = 0
	}
	cseg := segs
	if rv <= 1e-3 {
		cseg = 1
	}
	var rings [][]Vec3
	edgeRings := func(r float64, cham bool, top bool) {
		q := segs
		if cham {
			q = 1
		}
		for k := 0; k <= q; k++ {
			t := float64(k) / float64(q) // 0 at the flat face, 1 at the side
			var inset, dz float64
			if cham {
				inset, dz = r*(1-t), r*t
			} else {
				th := t * math.Pi / 2
				inset, dz = r*(1-math.Sin(th)), r*(1-math.Cos(th))
			}
			z := dz
			if top {
				z = h - dz
			}
			cr := math.Max(rv-inset, 0)
			if rv < 1e-3 {
				cr = 0
			}
			rings = append(rings, rrRing(w-2*inset, d-2*inset, cr, cseg, z))
		}
	}
	if rBot > 1e-3 {
		edgeRings(rBot, chamBot, false)
	} else {
		rings = append(rings, rrRing(w, d, rv, cseg, 0))
	}
	if rTop > 1e-3 {
		var top [][]Vec3
		save := rings
		rings = nil
		edgeRings(rTop, chamTop, true)
		// reverse so rings go upward
		for i := len(rings) - 1; i >= 0; i-- {
			top = append(top, rings[i])
		}
		rings = append(save, top...)
	} else {
		rings = append(rings, rrRing(w, d, rv, cseg, h))
	}
	return Loft(rings, true, true)
}

// cylinderProfile builds an (r,z) profile with optional edge treatment.
func cylinderProfile(r1, r2, h, eTop, eBot float64, chamfer bool, q int) []P2 {
	pts := []P2{{0, 0}}
	edge := func(cx, cz, rr float64, fromA, toA float64) {
		if chamfer {
			pts = append(pts, P2{cx + rr*math.Cos(fromA), cz + rr*math.Sin(fromA)}, P2{cx + rr*math.Cos(toA), cz + rr*math.Sin(toA)})
			return
		}
		for i := 0; i <= q; i++ {
			a := fromA + (toA-fromA)*float64(i)/float64(q)
			pts = append(pts, P2{cx + rr*math.Cos(a), cz + rr*math.Sin(a)})
		}
	}
	lim := math.Min(math.Min(r1, r2), h/2) - 1e-3
	eBot = clampF(eBot, 0, lim)
	eTop = clampF(eTop, 0, lim)
	if eBot > 1e-3 {
		edge(r1-eBot, eBot, eBot, -math.Pi/2, 0)
	} else {
		pts = append(pts, P2{r1, 0})
	}
	if eTop > 1e-3 {
		edge(r2-eTop, h-eTop, eTop, 0, math.Pi/2)
	} else {
		pts = append(pts, P2{r2, h})
	}
	pts = append(pts, P2{0, h})
	return pts
}

func edgeParams() []Param {
	return []Param{
		sel("edges", "Edges", "Sharp", "Sharp", "Rounded", "Chamfered").help("Soften edges. Rounded edges look nicer; chamfers print better on the bottom."),
		num("edge", "Edge size", 2, 0, 50, 0.5, "mm").show("edges!=Sharp"),
		sel("edgeWhere", "Apply to", "All edges", "All edges", "Top only", "Vertical only").show("edges!=Sharp"),
	}
}

func init() {
	register(&ShapeDef{Type: "box", Name: "Box", Category: "Basic", Desc: "A rectangular block. Round or chamfer its edges.",
		Color: "#ef4444", Axes: map[string]string{"x": "w", "y": "d", "z": "h"}, Tags: "cube block rectangle",
		Params: append([]Param{
			num("w", "Width (X)", 20, 0.1, 2000, 1, "mm"),
			num("d", "Depth (Y)", 20, 0.1, 2000, 1, "mm"),
			num("h", "Height (Z)", 20, 0.1, 2000, 1, "mm"),
		}, edgeParams()...),
		Gen: func(p Params) (*Mesh, error) {
			w, d, h := p.F("w", 20), p.F("d", 20), p.F("h", 20)
			e := p.F("edge", 2)
			style := p.S("edges", "Sharp")
			where := p.S("edgeWhere", "All edges")
			segs := int(math.Max(2, math.Round(6*p.Q())))
			if style == "Sharp" || e <= 0 {
				return roundedBlock(w, d, h, 0, 0, 0, false, false, 1), nil
			}
			ch := style == "Chamfered"
			switch where {
			case "Top only":
				if ch {
					return roundedBlock(w, d, h, 0, e, 0, true, false, 1), nil
				}
				return roundedBlock(w, d, h, e, e, 0, false, false, segs), nil
			case "Vertical only":
				if ch {
					return chamferVertical(w, d, h, e), nil
				}
				return roundedBlock(w, d, h, e, 0, 0, false, false, segs), nil
			}
			if ch {
				return roundedBlock(w, d, h, 0, e, e, true, true, 1), nil
			}
			return roundedBlock(w, d, h, e, e, e, false, false, segs), nil
		}})

	register(&ShapeDef{Type: "cylinder", Name: "Cylinder", Category: "Basic", Desc: "A round post. Turn on Thread to make a screw — set it as a hole for a matching threaded hole.",
		Color: "#f97316", Axes: map[string]string{"x": "d", "y": "d", "z": "h"}, Tags: "rod round circle post thread screw threaded",
		Params: append([]Param{
			num("d", "Diameter", 20, 0.1, 2000, 1, "mm"),
			num("h", "Height", 20, 0.1, 2000, 1, "mm"),
			num("sides", "Sides (0 = smooth)", 0, 0, 200, 1, "").show("thread=None"),
			sel("edges", "Edges", "Sharp", "Sharp", "Rounded", "Chamfered").show("thread=None"),
			num("edge", "Edge size", 1, 0, 50, 0.5, "mm").show("thread=None;edges!=Sharp"),
			sel("edgeWhere", "Apply to", "Top and bottom", "Top and bottom", "Top only", "Bottom only").show("thread=None;edges!=Sharp"),
			sel("thread", "Thread", "None", "None", "Threaded").help("Adds a real screw thread. The diameter is the nominal thread size. As a solid it is a screw; as a hole it cuts the matching nut thread."),
			num("tLen", "Threaded length (0 = all)", 0, 0, 2000, 1, "mm").show("thread=Threaded"),
		}, threadParams("thread=Threaded", "Metric coarse")...),
		Gen: func(p Params) (*Mesh, error) {
			d, h := p.F("d", 20), p.F("h", 20)
			if p.S("thread", "None") == "Threaded" {
				hole := p.B("_hole", false)
				spec := specFrom(p, d, hole)
				var m *Mesh
				L := p.F("tLen", 0)
				if hole || L <= 0 || L >= h {
					z0, z1 := 0.0, h
					if hole {
						z0, z1 = -0.01, h+0.01
					}
					m = Thread(spec, z0, z1, p.Q())
				} else {
					t := Thread(spec, 0, L, p.Q())
					shank := RevolveProfile([]P2{{0, L - 0.01}, {spec.Major / 2, L - 0.01}, {spec.Major / 2, h}, {0, h}}, p.Segs(d))
					m = Union(t, shank)
				}
				m.Info = ThreadInfo(spec, d, hole)
				return m, nil
			}
			segs := p.Segs(d)
			if s := p.I("sides", 0); s >= 3 {
				segs = s
			}
			et, eb := 0.0, 0.0
			if p.S("edges", "Sharp") != "Sharp" {
				e := p.F("edge", 1)
				switch p.S("edgeWhere", "") {
				case "Top only":
					et = e
				case "Bottom only":
					eb = e
				default:
					et, eb = e, e
				}
			}
			prof := cylinderProfile(d/2, d/2, h, et, eb, p.S("edges", "") == "Chamfered", int(math.Max(3, 6*p.Q())))
			return RevolveProfile(prof, segs), nil
		}})

	register(&ShapeDef{Type: "cone", Name: "Cone", Category: "Basic", Desc: "Tapered cylinder. Set top diameter above 0 for a truncated cone.",
		Color: "#eab308", Axes: map[string]string{"z": "h"}, Tags: "taper funnel",
		Params: []Param{
			num("d1", "Bottom diameter", 20, 0, 2000, 1, "mm"),
			num("d2", "Top diameter", 0, 0, 2000, 1, "mm"),
			num("h", "Height", 20, 0.1, 2000, 1, "mm"),
			num("sides", "Sides (0 = smooth)", 0, 0, 200, 1, ""),
		},
		Gen: func(p Params) (*Mesh, error) {
			d1, d2, h := p.F("d1", 20), p.F("d2", 0), p.F("h", 20)
			segs := p.Segs(math.Max(d1, d2))
			if s := p.I("sides", 0); s >= 3 {
				segs = s
			}
			return RevolveProfile([]P2{{0, 0}, {d1 / 2, 0}, {d2 / 2, h}, {0, h}}, segs), nil
		}})

	register(&ShapeDef{Type: "sphere", Name: "Sphere", Category: "Basic", Desc: "A ball. Squash it with Size to make an ellipsoid.",
		Color: "#22c55e", Axes: map[string]string{"x": "d", "y": "d", "z": "d"}, Tags: "ball round",
		Params: []Param{num("d", "Diameter", 20, 0.1, 2000, 1, "mm")},
		Gen: func(p Params) (*Mesh, error) {
			d := p.F("d", 20)
			segs := p.Segs(d)
			var pts []P2
			n := segs / 2
			for i := 0; i <= n; i++ {
				a := -math.Pi/2 + math.Pi*float64(i)/float64(n)
				pts = append(pts, P2{d / 2 * math.Cos(a), d/2 + d/2*math.Sin(a)})
			}
			pts[0].X, pts[n].X = 0, 0
			return RevolveProfile(pts, segs), nil
		}})

	register(&ShapeDef{Type: "hemisphere", Name: "Dome", Category: "Basic", Desc: "Half a sphere with a flat bottom.",
		Color: "#14b8a6", Axes: map[string]string{"x": "d", "y": "d", "z": "h"}, Tags: "half sphere hemisphere",
		Params: []Param{num("d", "Diameter", 20, 0.1, 2000, 1, "mm"), num("h", "Height", 10, 0.1, 2000, 1, "mm")},
		Gen: func(p Params) (*Mesh, error) {
			d, h := p.F("d", 20), p.F("h", 10)
			segs := p.Segs(d)
			pts := []P2{{0, 0}}
			n := segs / 4
			for i := 0; i <= n; i++ {
				a := math.Pi / 2 * float64(i) / float64(n)
				pts = append(pts, P2{d / 2 * math.Cos(a), h * math.Sin(a)})
			}
			pts[len(pts)-1].X = 0
			return RevolveProfile(pts, segs), nil
		}})

	register(&ShapeDef{Type: "torus", Name: "Torus", Category: "Basic", Desc: "A ring / donut.",
		Color: "#a855f7", Tags: "donut ring o-ring",
		Params: []Param{
			num("D", "Ring diameter", 30, 1, 2000, 1, "mm"),
			num("t", "Tube thickness", 6, 0.1, 500, 0.5, "mm"),
			num("angle", "Sweep angle", 360, 1, 360, 5, "°"),
		},
		Gen: func(p Params) (*Mesh, error) {
			D, t := p.F("D", 30), p.F("t", 6)
			R := math.Max(D/2-t/2, t/2+0.01)
			circ := Circle(t/2, p.Segs(t))
			for i := range circ {
				circ[i].X += R
				circ[i].Y += t / 2
			}
			return Revolve(Shape{Outer: circ}, p.F("angle", 360), p.Segs(D)), nil
		}})

	register(&ShapeDef{Type: "tube", Name: "Tube", Category: "Basic", Desc: "A hollow cylinder / ring / spacer. Can be threaded outside (like a bottle neck) and/or inside (like a cap or nut).",
		Color: "#06b6d4", Axes: map[string]string{"x": "od", "y": "od", "z": "h"}, Tags: "pipe ring spacer washer thread bottle cap nut coupler",
		Params: append([]Param{
			num("od", "Outer diameter", 20, 0.2, 2000, 1, "mm"),
			num("wall", "Wall thickness", 2, 0.1, 1000, 0.2, "mm"),
			num("h", "Height", 20, 0.1, 2000, 1, "mm"),
			num("angle", "Sweep angle", 360, 1, 360, 5, "°").show("thread=None"),
			sel("thread", "Thread", "None", "None", "Outside", "Inside", "Both").help("Outside = thread on the outer diameter. Inside = thread in the bore (bore diameter is the nominal size)."),
		}, threadParams("thread!=None", "Metric fine")...),
		Gen: func(p Params) (*Mesh, error) {
			od, wall, h := p.F("od", 20), p.F("wall", 2), p.F("h", 20)
			ro := od / 2
			ri := math.Max(ro-wall, 0)
			th := p.S("thread", "None")
			if th != "None" && ri > 0.05 {
				var outer, inner *Mesh
				var info []string
				if th == "Outside" || th == "Both" {
					sp := specFrom(p, od, false)
					outer = Thread(sp, 0, h, p.Q())
					info = append(info, "OUTSIDE: "+ThreadInfo(sp, od, false))
				} else {
					outer = cylAt(0, 0, od, 0, h, p.Segs(od))
				}
				if th == "Inside" || th == "Both" {
					sp := specFrom(p, 2*ri, true)
					sp.Chamfer = false
					inner = Thread(sp, -0.5, h+0.5, p.Q())
					info = append(info, "INSIDE: "+ThreadInfo(sp, 2*ri, true))
				} else {
					inner = cylAt(0, 0, 2*ri, -1, h+1, p.Segs(2*ri))
				}
				m := Subtract(outer, inner).FixTJunctions()
				for i, s := range info {
					if i > 0 {
						m.Info += "\n\n"
					}
					m.Info += s
				}
				return m, nil
			}
			if ri < 0.01 {
				return RevolveProfile([]P2{{0, 0}, {ro, 0}, {ro, h}, {0, h}}, p.Segs(od)), nil
			}
			return Revolve(Shape{Outer: Loop{{ri, 0}, {ro, 0}, {ro, h}, {ri, h}}}, p.F("angle", 360), p.Segs(od)), nil
		}})

	register(&ShapeDef{Type: "wedge", Name: "Wedge", Category: "Basic", Desc: "A ramp / right-angle triangle block. Great as a print support gusset.",
		Color: "#84cc16", Axes: map[string]string{"x": "w", "y": "d", "z": "h"}, Tags: "ramp triangle gusset",
		Params: []Param{num("w", "Width (X)", 20, 0.1, 2000, 1, "mm"), num("d", "Depth (Y)", 20, 0.1, 2000, 1, "mm"), num("h", "Height (Z)", 20, 0.1, 2000, 1, "mm")},
		Gen: func(p Params) (*Mesh, error) {
			w, d, h := p.F("w", 20), p.F("d", 20), p.F("h", 20)
			prof := Shape{Outer: Loop{{-w / 2, 0}, {w / 2, 0}, {-w / 2, h}}}
			return Sweep(prof, []Frame{{V(0, -d/2, 0), V(1, 0, 0), V(0, 0, 1)}, {V(0, d/2, 0), V(1, 0, 0), V(0, 0, 1)}}, false), nil
		}})

	register(&ShapeDef{Type: "pyramid", Name: "Pyramid", Category: "Basic", Desc: "A pyramid with any number of sides.",
		Color: "#f59e0b", Axes: map[string]string{"z": "h"}, Tags: "pyramid point",
		Params: []Param{num("sides", "Sides", 4, 3, 64, 1, ""), num("d", "Base size", 20, 0.1, 2000, 1, "mm"), num("h", "Height", 20, 0.1, 2000, 1, "mm"), num("top", "Top size (0 = point)", 0, 0, 2000, 1, "mm")},
		Gen: func(p Params) (*Mesh, error) {
			n := p.I("sides", 4)
			d, h, top := p.F("d", 20), p.F("h", 20), p.F("top", 0)
			rot := 45.0
			if n != 4 {
				rot = 90
			}
			r := d / 2
			if n == 4 {
				r = d / math.Sqrt2
			}
			rt := top / 2
			if n == 4 {
				rt = top / math.Sqrt2
			}
			var a, b []Vec3
			for _, q := range NGon(r, n, rot) {
				a = append(a, V(q.X, q.Y, 0))
			}
			for _, q := range NGon(math.Max(rt, 0), n, rot) {
				b = append(b, V(q.X, q.Y, h))
			}
			return Loft([][]Vec3{a, b}, true, true), nil
		}})

	register(&ShapeDef{Type: "prism", Name: "Polygon prism", Category: "Basic", Desc: "Hexagon, octagon or any regular polygon block.",
		Color: "#ec4899", Axes: map[string]string{"z": "h"}, Tags: "hexagon octagon triangle polygon",
		Params: []Param{num("sides", "Sides", 6, 3, 64, 1, ""), num("d", "Across corners", 20, 0.1, 2000, 1, "mm"), num("h", "Height", 10, 0.1, 2000, 1, "mm"), num("twist", "Twist", 0, -720, 720, 5, "°")},
		Gen: func(p Params) (*Mesh, error) {
			l := NGon(p.F("d", 20)/2, p.I("sides", 6), 90)
			return Extrude(Shape{Outer: l}, p.F("h", 10), p.F("twist", 0), 1, int(math.Abs(p.F("twist", 0))/5)+1), nil
		}})

	register(&ShapeDef{Type: "star", Name: "Star", Category: "Basic", Desc: "A star shape.",
		Color: "#facc15", Axes: map[string]string{"z": "h"}, Tags: "star",
		Params: []Param{num("points", "Points", 5, 3, 50, 1, ""), num("d", "Outer diameter", 30, 1, 2000, 1, "mm"), num("inner", "Inner diameter", 14, 0.5, 2000, 1, "mm"), num("h", "Height", 5, 0.1, 2000, 0.5, "mm")},
		Gen: func(p Params) (*Mesh, error) {
			n := p.I("points", 5)
			ro, ri := p.F("d", 30)/2, p.F("inner", 14)/2
			var l Loop
			for i := 0; i < 2*n; i++ {
				r := ro
				if i%2 == 1 {
					r = ri
				}
				a := math.Pi/2 + math.Pi*float64(i)/float64(n)
				l = append(l, P2{r * math.Cos(a), r * math.Sin(a)})
			}
			return Extrude(Shape{Outer: l}, p.F("h", 5), 0, 1, 1), nil
		}})

	register(&ShapeDef{Type: "capsule", Name: "Capsule", Category: "Basic", Desc: "A pill shape — cylinder with round ends.",
		Color: "#10b981", Axes: map[string]string{"z": "h"}, Tags: "pill",
		Params: []Param{num("d", "Diameter", 10, 0.1, 2000, 1, "mm"), num("h", "Total height", 30, 0.1, 2000, 1, "mm")},
		Gen: func(p Params) (*Mesh, error) {
			d, h := p.F("d", 10), p.F("h", 30)
			r := d / 2
			if h < d {
				h = d
			}
			n := p.Segs(d) / 4
			var pts []P2
			for i := 0; i <= n; i++ {
				a := -math.Pi/2 + math.Pi/2*float64(i)/float64(n)
				pts = append(pts, P2{r * math.Cos(a), r + r*math.Sin(a)})
			}
			for i := 0; i <= n; i++ {
				a := math.Pi / 2 * float64(i) / float64(n)
				pts = append(pts, P2{r * math.Cos(a), h - r + r*math.Sin(a)})
			}
			pts[0].X, pts[len(pts)-1].X = 0, 0
			return RevolveProfile(pts, p.Segs(d)), nil
		}})

	register(&ShapeDef{Type: "plate", Name: "Rounded plate", Category: "Basic", Desc: "A flat plate with rounded corners — the start of most brackets and mounts.",
		Color: "#64748b", Axes: map[string]string{"x": "w", "y": "d", "z": "h"}, Tags: "plate base slab",
		Params: []Param{num("w", "Width (X)", 60, 1, 2000, 1, "mm"), num("d", "Depth (Y)", 40, 1, 2000, 1, "mm"), num("h", "Thickness", 3, 0.2, 500, 0.2, "mm"), num("r", "Corner radius", 5, 0, 500, 0.5, "mm")},
		Gen: func(p Params) (*Mesh, error) {
			return roundedBlock(p.F("w", 60), p.F("d", 40), p.F("h", 3), p.F("r", 5), 0, 0, false, false, int(math.Max(3, 8*p.Q()))), nil
		}})
}

func chamferVertical(w, d, h, c float64) *Mesh {
	c = clampF(c, 0, math.Min(w, d)/2-1e-3)
	l := Loop{{-w/2 + c, -d / 2}, {w/2 - c, -d / 2}, {w / 2, -d/2 + c}, {w / 2, d/2 - c}, {w/2 - c, d / 2}, {-w/2 + c, d / 2}, {-w / 2, d/2 - c}, {-w / 2, -d/2 + c}}
	return Extrude(Shape{Outer: l}, h, 0, 1, 1)
}

// boxAt makes an axis-aligned box from min to max corner.
func boxAt(lo, hi Vec3) *Mesh {
	m := roundedBlock(hi.X-lo.X, hi.Y-lo.Y, hi.Z-lo.Z, 0, 0, 0, false, false, 1)
	return m.Transformed(Translate((lo.X+hi.X)/2, (lo.Y+hi.Y)/2, lo.Z))
}

// cylAt makes a cylinder of diameter d between z0 and z1 at x,y.
func cylAt(x, y, d, z0, z1 float64, segs int) *Mesh {
	m := RevolveProfile([]P2{{0, 0}, {d / 2, 0}, {d / 2, z1 - z0}, {0, z1 - z0}}, segs)
	return m.Transformed(Translate(x, y, z0))
}

// presets that reuse the cylinder generator with thread settings switched on
func init() {
	base := registry["cylinder"]
	mk := func(typ, name, cat, desc string, hole bool, defs map[string]any) {
		d := *base
		d.Type, d.Name, d.Category, d.Desc, d.Hole = typ, name, cat, desc, hole
		d.Params = make([]Param, len(base.Params))
		copy(d.Params, base.Params)
		for i := range d.Params {
			if v, ok := defs[d.Params[i].Key]; ok {
				d.Params[i].Def = v
			}
		}
		d.Color = "#0ea5e9"
		if hole {
			d.Color = "#64748b"
		}
		d.Tags = "thread threaded screw custom lead screw jar lid " + name
		register(&d)
	}
	mk("tscrew", "Threaded post", "Hardware", "A screw thread of any size, pitch and profile. Pair it with a Threaded bore of the same diameter.", false,
		map[string]any{"thread": "Threaded", "d": 16.0, "h": 20.0})
	mk("tbore", "Threaded bore", "Holes", "A hole that cuts an internal thread of any size, pitch and profile. A Threaded post of the same diameter screws in.", true,
		map[string]any{"thread": "Threaded", "d": 16.0, "h": 20.0})
}
