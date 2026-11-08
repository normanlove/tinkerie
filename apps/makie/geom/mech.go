package geom

import "math"

func involute(a float64) float64 { return math.Tan(a) - a }

// GearOutline returns an involute spur gear outline.
func GearOutline(mod float64, teeth int, pa, backlash float64, q float64) Loop {
	z := float64(teeth)
	r := mod * z / 2
	rb := r * math.Cos(pa*math.Pi/180)
	ra := r + mod
	rf := math.Max(r-1.25*mod, 0.5)
	invPa := involute(pa * math.Pi / 180)
	half := math.Pi/(2*z) - backlash/(2*r)
	psi := func(rr float64) float64 {
		if rr < rb {
			rr = rb
		}
		al := math.Acos(rb / rr)
		return half + invPa - involute(al)
	}
	n := int(math.Max(5, 8*q))
	var l Loop
	pt := func(rr, ang float64) P2 { return P2{rr * math.Cos(ang), rr * math.Sin(ang)} }
	start := math.Max(rb, rf)
	for k := 0; k < teeth; k++ {
		c := 2 * math.Pi * float64(k) / z
		// right flank (angles increase with radius)
		if rf < rb {
			l = append(l, pt(rf, c-psi(rb)))
		}
		for i := 0; i <= n; i++ {
			rr := start + (ra-start)*float64(i)/float64(n)
			l = append(l, pt(rr, c-psi(rr)))
		}
		// tip
		l = append(l, pt(ra, c))
		for i := n; i >= 0; i-- {
			rr := start + (ra-start)*float64(i)/float64(n)
			l = append(l, pt(rr, c+psi(rr)))
		}
		if rf < rb {
			l = append(l, pt(rf, c+psi(rb)))
		}
		// root arc to the next tooth
		next := c + 2*math.Pi/z
		a0 := c + psi(start)
		a1 := next - psi(start)
		for i := 1; i < 3; i++ {
			l = append(l, pt(rf, a0+(a1-a0)*float64(i)/3))
		}
	}
	return l
}

func boreLoop(d float64, kind string, segs int) Loop {
	c := Circle(d/2, segs)
	if kind == "D-shaft" {
		flat := d/2 - d*0.1
		for i := range c {
			if c[i].X > flat {
				c[i].X = flat
			}
		}
	}
	if kind == "Hex" {
		return NGon(d/math.Sqrt(3), 6, 0)
	}
	return c
}

func init() {
	register(&ShapeDef{Type: "gear", Name: "Spur gear", Category: "Mechanical", Color: "#0ea5e9",
		Desc: "Involute spur gear. Gears mesh when they share the same module. Distance between centres = module × (teeth1 + teeth2) / 2.",
		Tags: "gear cog spur helical herringbone",
		Params: []Param{
			num("module", "Module (tooth size)", 1.5, 0.3, 10, 0.1, "mm"),
			num("teeth", "Teeth", 20, 6, 200, 1, ""),
			num("h", "Thickness", 6, 0.5, 200, 0.5, "mm"),
			num("bore", "Bore diameter", 5, 0, 200, 0.1, "mm"),
			sel("boreType", "Bore shape", "Round", "Round", "D-shaft", "Hex"),
			sel("style", "Style", "Straight", "Straight", "Helical", "Herringbone"),
			num("helix", "Helix angle", 20, 0, 45, 1, "°").show("style!=Straight"),
			num("pa", "Pressure angle", 20, 14.5, 30, 0.5, "°"),
			num("backlash", "Backlash", 0.15, 0, 1, 0.05, "mm"),
			num("hubD", "Hub diameter (0 = none)", 0, 0, 200, 1, "mm"),
			num("hubH", "Hub height", 5, 0, 100, 0.5, "mm").show("hubD!=0"),
		},
		Gen: func(p Params) (*Mesh, error) {
			mod, teeth, h := p.F("module", 1.5), p.I("teeth", 20), p.F("h", 6)
			outline := GearOutline(mod, teeth, p.F("pa", 20), p.F("backlash", 0.15), p.Q())
			s := Shape{Outer: outline}
			bore := p.F("bore", 5)
			hubD := p.F("hubD", 0)
			rp := mod * float64(teeth) / 2
			if bore > 0 && bore < 2*(rp-1.25*mod) && hubD == 0 {
				s.Holes = []Loop{boreLoop(bore, p.S("boreType", "Round"), p.Segs(bore))}
			}
			var g *Mesh
			switch p.S("style", "Straight") {
			case "Helical":
				tw := h * math.Tan(p.F("helix", 20)*math.Pi/180) / rp * 180 / math.Pi
				g = Extrude(s, h, tw, 1, int(math.Max(4, math.Abs(tw)/3)))
			case "Herringbone":
				tw := h / 2 * math.Tan(p.F("helix", 20)*math.Pi/180) / rp * 180 / math.Pi
				lay := int(math.Max(3, math.Abs(tw)/3))
				var frames []Frame
				for i := 0; i <= 2*lay; i++ {
					t := float64(i) / float64(2*lay)
					tt := t * 2
					if t > 0.5 {
						tt = 2 - 2*t
					}
					a := tw * tt * math.Pi / 180
					sn, cs := math.Sincos(a)
					frames = append(frames, Frame{V(0, 0, h*t), V(cs, sn, 0), V(-sn, cs, 0)})
				}
				g = Sweep(s, frames, false)
			default:
				g = Extrude(s, h, 0, 1, 1)
			}
			if hubD > 0 {
				g = Union(g, cylAt(0, 0, hubD, h-0.01, h+p.F("hubH", 5), p.Segs(hubD)))
				if bore > 0 {
					hl := Extrude(Shape{Outer: boreLoop(bore, p.S("boreType", "Round"), p.Segs(bore))}, h+p.F("hubH", 5)+2, 0, 1, 1).Transformed(Translate(0, 0, -1))
					g = Subtract(g, hl)
				}
			}
			return g, nil
		}})

	register(&ShapeDef{Type: "rack", Name: "Gear rack", Category: "Mechanical", Color: "#0ea5e9",
		Desc: "Straight toothed rack that meshes with a spur gear of the same module.", Tags: "rack linear gear",
		Params: []Param{num("module", "Module", 1.5, 0.3, 10, 0.1, "mm"), num("teeth", "Teeth", 15, 2, 500, 1, ""), num("h", "Thickness", 6, 0.5, 200, 0.5, "mm"), num("base", "Base height", 4, 0.5, 100, 0.5, "mm")},
		Gen: func(p Params) (*Mesh, error) {
			mod, n, h, base := p.F("module", 1.5), p.I("teeth", 15), p.F("h", 6), p.F("base", 4)
			pitch := math.Pi * mod
			tn := math.Tan(20 * math.Pi / 180)
			add, ded := mod, 1.25*mod
			L := pitch * float64(n)
			l := Loop{{0, 0}, {L, 0}, {L, base}}
			for i := n - 1; i >= 0; i-- {
				x := float64(i)*pitch + pitch/2
				hw := pitch / 4
				l = append(l, P2{x + hw + ded*tn, base}, P2{x + hw - add*tn, base + ded + add}, P2{x - hw + add*tn, base + ded + add}, P2{x - hw - ded*tn, base})
			}
			l = append(l, P2{0, base})
			return Sweep(Shape{Outer: l}, []Frame{{V(-L/2, 0, 0), V(1, 0, 0), V(0, 0, 1)}, {V(-L/2, h, 0), V(1, 0, 0), V(0, 0, 1)}}, false).Transformed(Translate(0, -h/2, 0)), nil
		}})

	register(&ShapeDef{Type: "spring", Name: "Coil spring", Category: "Mechanical", Color: "#8b5cf6",
		Desc: "A helical coil spring.", Tags: "spring coil helix",
		Params: []Param{num("D", "Coil diameter", 20, 2, 500, 1, "mm"), num("wire", "Wire thickness", 2, 0.4, 50, 0.1, "mm"), num("turns", "Turns", 5, 0.5, 100, 0.5, ""), num("pitch", "Pitch (gap per turn)", 5, 0.5, 100, 0.5, "mm")},
		Gen: func(p Params) (*Mesh, error) {
			D, wire, turns, pitch := p.F("D", 20), p.F("wire", 2), p.F("turns", 5), p.F("pitch", 5)
			if pitch < wire*1.05 {
				pitch = wire * 1.05
			}
			R := D/2 - wire/2
			steps := int(turns * float64(p.Segs(D)))
			prof := Shape{Outer: Circle(wire/2, int(math.Max(10, float64(p.Segs(wire)))))}
			var frames []Frame
			for i := 0; i <= steps; i++ {
				t := float64(i) / float64(steps) * turns * 2 * math.Pi
				z := wire/2 + pitch*t/(2*math.Pi)
				pos := V(R*math.Cos(t), R*math.Sin(t), z)
				tan := V(-R*math.Sin(t), R*math.Cos(t), pitch/(2*math.Pi)).Norm()
				radial := V(math.Cos(t), math.Sin(t), 0)
				bin := tan.Cross(radial).Norm()
				frames = append(frames, Frame{pos, radial, bin})
			}
			return Sweep(prof, frames, false), nil
		}})

	register(&ShapeDef{Type: "knob", Name: "Knurled knob", Category: "Mechanical", Color: "#f43f5e",
		Desc: "A grippy knob with a bore for a shaft or a hex pocket for a bolt head.", Tags: "knob handle grip thumb screw",
		Params: []Param{
			num("d", "Diameter", 30, 5, 300, 1, "mm"), num("h", "Height", 14, 2, 200, 1, "mm"),
			num("grips", "Grip lobes", 12, 3, 60, 1, ""), num("depth", "Grip depth", 1.5, 0, 20, 0.25, "mm"),
			sel("bore", "Bore", "Hex nut pocket", "None", "Round", "D-shaft", "Hex nut pocket"),
			num("boreD", "Bore diameter", 6, 1, 100, 0.1, "mm").show("bore!=Hex nut pocket"),
			sel("size", "Nut size", "M5", metricSizes()...).show("bore=Hex nut pocket"),
		},
		Gen: func(p Params) (*Mesh, error) {
			d, h := p.F("d", 30), p.F("h", 14)
			n, dep := p.I("grips", 12), p.F("depth", 1.5)
			segs := int(math.Max(float64(n*8), float64(p.Segs(d))))
			var l Loop
			for i := 0; i < segs; i++ {
				a := 2 * math.Pi * float64(i) / float64(segs)
				r := d/2 - dep*(0.5-0.5*math.Cos(float64(n)*a))
				l = append(l, P2{r * math.Cos(a), r * math.Sin(a)})
			}
			body := Extrude(Shape{Outer: l}, h, 0, 1, 1)
			// soften the top
			cap := RevolveProfile([]P2{{0, -0.01}, {d/2 + 1, -0.01}, {d/2 + 1, h - d*0.08}, {d/2 - d*0.08, h + 0.01}, {0, h + 0.01}}, p.Segs(d))
			body = Intersect(body, cap)
			switch p.S("bore", "") {
			case "Round", "D-shaft":
				bd := p.F("boreD", 6)
				hl := Extrude(Shape{Outer: boreLoop(bd, p.S("bore", ""), p.Segs(bd))}, h*0.8+1, 0, 1, 1).Transformed(Translate(0, 0, -1))
				body = Subtract(body, hl)
			case "Hex nut pocket":
				m := getMetric(p)
				body = Subtract(body, hexPrism(m.NutAF+0.4, -1, m.NutH+0.5))
				body = Subtract(body, cylAt(0, 0, m.Normal+0.2, -1, h*0.85, p.Segs(m.D)))
			}
			return body, nil
		}})

	register(&ShapeDef{Type: "lbracket", Name: "L-bracket", Category: "Mechanical", Color: "#64748b",
		Desc: "Right-angle bracket with screw holes and an optional gusset.", Tags: "bracket angle corner mount",
		Params: []Param{
			num("w", "Width", 20, 4, 500, 1, "mm"), num("a", "Leg A length", 30, 5, 500, 1, "mm"), num("b", "Leg B length", 30, 5, 500, 1, "mm"),
			num("t", "Thickness", 4, 1, 50, 0.5, "mm"), boolp("gusset", "Gusset", true),
			sel("size", "Screw holes", "M4", append([]string{"None"}, metricSizes()...)...),
		},
		Gen: func(p Params) (*Mesh, error) {
			w, a, b, t := p.F("w", 20), p.F("a", 30), p.F("b", 30), p.F("t", 4)
			prof := Loop{{0, 0}, {a, 0}, {a, t}, {t, t}, {t, b}, {0, b}}
			m := Sweep(Shape{Outer: prof}, []Frame{{V(0, -w/2, 0), V(1, 0, 0), V(0, 0, 1)}, {V(0, w/2, 0), V(1, 0, 0), V(0, 0, 1)}}, false)
			if p.B("gusset", true) {
				g := math.Min(a, b) * 0.6
				gt := math.Min(t, w/3)
				gp := Loop{{t - 0.01, t - 0.01}, {g, t - 0.01}, {t - 0.01, g}}
				m = Union(m, Sweep(Shape{Outer: gp}, []Frame{{V(0, -gt/2, 0), V(1, 0, 0), V(0, 0, 1)}, {V(0, gt/2, 0), V(1, 0, 0), V(0, 0, 1)}}, false))
			}
			if mt, ok := metrics[p.S("size", "M4")]; ok {
				hd := mt.Normal + 0.2
				off := w / 4
				if p.B("gusset", true) {
					off = w/4 + math.Min(t, w/3)/4
				}
				ys := []float64{-off, off}
				if lim := w/2 - holeEdge(hd); off > lim {
					ys = []float64{0} // not enough width for two holes: one in the middle
				}
				if math.Min(a, b)-t < 2*holeEdge(hd) || w < 2*holeEdge(hd) {
					ys = nil // leg too small to take this screw safely
				}
				for _, y := range ys {
					m = Subtract(m, cylAt(a*0.65+t*0.35, y, hd, -1, t+1, p.Segs(hd)))
					m = Subtract(m, cylAt(0, 0, hd, -1, t+1, p.Segs(hd)).Transformed(Translate(0, y, b*0.65+t*0.35).Mul(RotY(90))))
				}
			}
			return m, nil
		}})

	register(&ShapeDef{Type: "enclosure", Name: "Project box", Category: "Mechanical", Color: "#475569",
		Desc: "Hollow enclosure with rounded corners, open top and optional corner screw posts. Pair with a Lid.",
		Tags: "enclosure box case container housing", Axes: map[string]string{"x": "w", "y": "d", "z": "h"},
		Params: []Param{
			num("w", "Outside width", 80, 10, 1000, 1, "mm"), num("d", "Outside depth", 60, 10, 1000, 1, "mm"), num("h", "Height", 30, 3, 1000, 1, "mm"),
			num("wall", "Wall thickness", 2, 0.8, 20, 0.2, "mm"), num("floor", "Floor thickness", 2, 0.4, 20, 0.2, "mm"), num("r", "Corner radius", 4, 0, 100, 0.5, "mm"),
			sel("posts", "Corner screw posts", "M3", append([]string{"None"}, "M2", "M2.5", "M3", "M4")...),
		},
		Gen: func(p Params) (*Mesh, error) {
			w, d, h := p.F("w", 80), p.F("d", 60), p.F("h", 30)
			wall, fl, r := p.F("wall", 2), p.F("floor", 2), p.F("r", 4)
			segs := int(math.Max(3, 6*p.Q()))
			outer := roundedBlock(w, d, h, r, 0, 0, false, false, segs)
			inner := roundedBlock(w-2*wall, d-2*wall, h, math.Max(r-wall, 0), 0, 0, false, false, segs).Transformed(Translate(0, 0, fl))
			box := Subtract(outer, inner)
			if mt, ok := metrics[p.S("posts", "M3")]; ok {
				pd := mt.InsertD + 3.2
				var posts []*Mesh
				for _, sx := range []float64{-1, 1} {
					for _, sy := range []float64{-1, 1} {
						cx := sx * (w/2 - wall - pd/2 + 0.6)
						cy := sy * (d/2 - wall - pd/2 + 0.6)
						post := cylAt(cx, cy, pd, fl-0.01, h-0.01, p.Segs(pd))
						post = Subtract(post, cylAt(cx, cy, mt.D*0.85, fl+1, h+1, p.Segs(mt.D)))
						posts = append(posts, post)
					}
				}
				for _, ps := range posts {
					box = Union(box, ps)
				}
			}
			return box, nil
		}})

	register(&ShapeDef{Type: "lid", Name: "Box lid", Category: "Mechanical", Color: "#475569",
		Desc: "Lid that fits the Project box with the same size settings. Printed lip-up; flip it over to fit. Has a locating lip and screw holes.",
		Tags: "lid cover top", Axes: map[string]string{"x": "w", "y": "d"},
		Params: []Param{
			num("w", "Outside width", 80, 10, 1000, 1, "mm"), num("d", "Outside depth", 60, 10, 1000, 1, "mm"), num("t", "Thickness", 2, 0.6, 20, 0.2, "mm"),
			num("wall", "Box wall thickness", 2, 0.8, 20, 0.2, "mm"), num("r", "Corner radius", 4, 0, 100, 0.5, "mm"),
			num("lip", "Lip height", 2, 0, 20, 0.5, "mm"), num("tol", "Fit tolerance", 0.25, 0, 2, 0.05, "mm"),
			sel("posts", "Screw holes for", "M3", "None", "M2", "M2.5", "M3", "M4"),
		},
		Gen: func(p Params) (*Mesh, error) {
			w, d, t := p.F("w", 80), p.F("d", 60), p.F("t", 2)
			wall, r, lip, tol := p.F("wall", 2), p.F("r", 4), p.F("lip", 2), p.F("tol", 0.25)
			segs := int(math.Max(3, 6*p.Q()))
			plate := roundedBlock(w, d, t, r, 0, 0, false, false, segs)
			if lip > 0 {
				lw, ld := w-2*wall-2*tol, d-2*wall-2*tol
				ring := Extrude(Shape{Outer: RoundedRect(lw, ld, math.Max(r-wall-tol, 0), segs), Holes: []Loop{RoundedRect(lw-2*1.6, ld-2*1.6, math.Max(r-wall-tol-1.6, 0), segs)}}, lip+0.01, 0, 1, 1)
				plate = Union(plate, ring.Transformed(Translate(0, 0, t-0.01)))
			}
			if mt, ok := metrics[p.S("posts", "M3")]; ok {
				pd := mt.InsertD + 3.2
				for _, sx := range []float64{-1, 1} {
					for _, sy := range []float64{-1, 1} {
						cx := sx * (w/2 - wall - pd/2 + 0.6)
						cy := sy * (d/2 - wall - pd/2 + 0.6)
						plate = Subtract(plate, cylAt(cx, cy, mt.Normal+0.2, -1, t+lip+1, p.Segs(mt.D)))
					}
				}
				// cut lip away around posts
				for _, sx := range []float64{-1, 1} {
					for _, sy := range []float64{-1, 1} {
						cx := sx * (w/2 - wall - pd/2 + 0.6)
						cy := sy * (d/2 - wall - pd/2 + 0.6)
						plate = Subtract(plate, cylAt(cx, cy, pd+0.8, t, t+lip+1, p.Segs(pd)))
					}
				}
			}
			return plate, nil
		}})

	register(&ShapeDef{Type: "elbow", Name: "Pipe bend", Category: "Mechanical", Color: "#0891b2",
		Desc: "Curved tube, e.g. for ducts, cable guides or handles.", Tags: "elbow bend pipe duct handle",
		Params: []Param{num("od", "Tube outer diameter", 10, 1, 500, 0.5, "mm"), num("wall", "Wall (0 = solid)", 1.5, 0, 100, 0.2, "mm"), num("R", "Bend radius", 20, 1, 1000, 1, "mm"), num("angle", "Bend angle", 90, 1, 360, 5, "°")},
		Gen: func(p Params) (*Mesh, error) {
			od, wall, R := p.F("od", 10), p.F("wall", 1.5), p.F("R", 20)
			if R < od/2+0.1 {
				R = od/2 + 0.1
			}
			s := Shape{Outer: Circle(od/2, p.Segs(od))}
			if wall > 0 && wall < od/2 {
				s.Holes = []Loop{Circle(od/2-wall, p.Segs(od))}
			}
			for i := range s.Outer {
				s.Outer[i].X += R
			}
			for _, h := range s.Holes {
				for i := range h {
					h[i].X += R
				}
			}
			// profile in XZ plane, revolve around Z, then lay it down
			m := Revolve(s, p.F("angle", 90), p.Segs(2*R))
			return m.Transformed(Translate(0, 0, od/2)), nil
		}})

	register(&ShapeDef{Type: "hinge", Name: "Print-in-place hinge", Category: "Mechanical", Color: "#7c3aed",
		Desc: "Two leaves joined by a pin that prints already assembled — no supports needed.", Tags: "hinge print in place pivot",
		Params: []Param{num("w", "Length", 40, 10, 500, 1, "mm"), num("leaf", "Leaf width", 15, 5, 200, 1, "mm"), num("t", "Leaf thickness", 3, 1, 20, 0.2, "mm"), num("knuckles", "Knuckles", 5, 3, 31, 2, ""), num("gap", "Clearance", 0.4, 0.2, 1, 0.05, "mm")},
		Gen: func(p Params) (*Mesh, error) {
			w, leaf, t := p.F("w", 40), p.F("leaf", 15), p.F("t", 3)
			n := p.I("knuckles", 5)
			if n%2 == 0 {
				n++
			}
			gap := p.F("gap", 0.4)
			kd := t * 2.2
			pin := kd * 0.45
			segs := p.Segs(kd)
			seg := w / float64(n)
			zc := kd / 2
			rot := RotY(90)
			cylX := func(x0, x1, d float64) *Mesh {
				c := RevolveProfile([]P2{{0, 0}, {d / 2, 0}, {d / 2, x1 - x0}, {0, x1 - x0}}, segs)
				return c.Transformed(Translate(x0, 0, zc).Mul(rot))
			}
			// cone ends on the pin so it prints without support
			var A, B []*Mesh
			A = append(A, boxAt(V(-w/2, kd/2+gap, 0), V(w/2, kd/2+gap+leaf, t)))
			B = append(B, boxAt(V(-w/2, -kd/2-gap-leaf, 0), V(w/2, -kd/2-gap, t)))
			for i := 0; i < n; i++ {
				x0 := -w/2 + float64(i)*seg
				x1 := x0 + seg
				if i%2 == 0 {
					k := cylX(x0+gap/2, x1-gap/2, kd)
					A = append(A, k, boxAt(V(x0+gap/2, 0, 0), V(x1-gap/2, kd/2+gap+0.5, t)))
				} else {
					k := cylX(x0+gap/2, x1-gap/2, kd)
					k = Subtract(k, cylX(x0, x1, pin*2+2*gap))
					B = append(B, k, boxAt(V(x0+gap/2, -kd/2-gap-0.5, 0), V(x1-gap/2, 0, t)))
				}
			}
			a := UnionAll(A)
			a = Union(a, cylX(-w/2+gap/2, w/2-gap/2, pin*2))
			b := UnionAll(B)
			m := a.Clone()
			m.Append(b)
			return m, nil
		}})

	register(&ShapeDef{Type: "honeycomb", Name: "Honeycomb panel", Category: "Mechanical", Color: "#ca8a04",
		Desc: "Light, strong hexagon-pattern panel for vents and grilles.", Tags: "honeycomb hex vent grille mesh",
		Axes:   map[string]string{"z": "h"},
		Params: []Param{num("w", "Width", 60, 5, 1000, 1, "mm"), num("d", "Depth", 40, 5, 1000, 1, "mm"), num("h", "Thickness", 2, 0.4, 50, 0.2, "mm"), num("cell", "Cell size", 6, 1, 100, 0.5, "mm"), num("wall", "Wall", 1.2, 0.4, 20, 0.1, "mm"), num("border", "Border", 3, 0, 100, 0.5, "mm")},
		Gen: func(p Params) (*Mesh, error) {
			return honeycomb(p.F("w", 60), p.F("d", 40), p.F("h", 2), p.F("cell", 6), p.F("wall", 1.2), p.F("border", 3)), nil
		}})

	register(&ShapeDef{Type: "clip", Name: "Snap-fit clip", Category: "Mechanical", Color: "#16a34a",
		Desc: "Cantilever snap hook. Print it lying flat for strength.", Tags: "snap fit clip hook latch",
		Params: []Param{num("len", "Arm length", 15, 4, 200, 0.5, "mm"), num("w", "Width", 6, 1, 100, 0.5, "mm"), num("t", "Arm thickness", 1.6, 0.6, 10, 0.1, "mm"), num("hook", "Hook depth", 1.2, 0.2, 10, 0.1, "mm")},
		Gen: func(p Params) (*Mesh, error) {
			l, w, t, hk := p.F("len", 15), p.F("w", 6), p.F("t", 1.6), p.F("hook", 1.2)
			prof := Loop{{0, 0}, {t, 0}, {t, l - hk*2}, {t + hk, l - hk*1.2}, {t + hk, l - hk*0.8}, {t, l}, {0, l}}
			return Sweep(Shape{Outer: prof}, []Frame{{V(0, -w/2, 0), V(1, 0, 0), V(0, 0, 1)}, {V(0, w/2, 0), V(1, 0, 0), V(0, 0, 1)}}, false), nil
		}})
}

// clipConvex clips a convex polygon to an axis-aligned rectangle (Sutherland–Hodgman).
func clipConvex(poly Loop, x0, y0, x1, y1 float64) Loop {
	clip := func(in Loop, inside func(P2) bool, inter func(a, b P2) P2) Loop {
		var out Loop
		for i := range in {
			a, b := in[i], in[(i+1)%len(in)]
			ia, ib := inside(a), inside(b)
			if ia {
				out = append(out, a)
			}
			if ia != ib {
				out = append(out, inter(a, b))
			}
		}
		return out
	}
	lx := func(x float64) func(a, b P2) P2 {
		return func(a, b P2) P2 { t := (x - a.X) / (b.X - a.X); return P2{x, a.Y + t*(b.Y-a.Y)} }
	}
	ly := func(y float64) func(a, b P2) P2 {
		return func(a, b P2) P2 { t := (y - a.Y) / (b.Y - a.Y); return P2{a.X + t*(b.X-a.X), y} }
	}
	poly = clip(poly, func(p P2) bool { return p.X >= x0 }, lx(x0))
	if len(poly) > 0 {
		poly = clip(poly, func(p P2) bool { return p.X <= x1 }, lx(x1))
	}
	if len(poly) > 0 {
		poly = clip(poly, func(p P2) bool { return p.Y >= y0 }, ly(y0))
	}
	if len(poly) > 0 {
		poly = clip(poly, func(p P2) bool { return p.Y <= y1 }, ly(y1))
	}
	return poly.Clean()
}

// ringTris triangulates the region between a convex outer loop and a convex hole
// (both CCW, hole star-shaped around c) by merging the two loops by angle.
func ringTris(outer, hole Loop, c P2) [][3]P2 {
	ang := func(p P2) float64 { return math.Atan2(p.Y-c.Y, p.X-c.X) }
	rot := func(l Loop) Loop {
		k := 0
		for i := range l {
			if ang(l[i]) < ang(l[k]) {
				k = i
			}
		}
		return append(append(Loop{}, l[k:]...), l[:k]...)
	}
	o, hl := rot(outer), rot(hole)
	var out [][3]P2
	i, j := 0, 0
	no, nh := len(o), len(hl)
	for i < no || j < nh {
		a, b := o[i%no], hl[j%nh]
		var na, nb float64 = 1e9, 1e9
		if i < no {
			na = ang(o[(i+1)%no])
			if i+1 == no {
				na += 2 * math.Pi
			}
		}
		if j < nh {
			nb = ang(hl[(j+1)%nh])
			if j+1 == nh {
				nb += 2 * math.Pi
			}
		}
		if na <= nb {
			out = append(out, [3]P2{a, o[(i+1)%no], b})
			i++
		} else {
			out = append(out, [3]P2{a, hl[(j+1)%nh], b})
			j++
		}
	}
	return out
}

// honeycomb builds a plate with hexagonal holes from exact lattice tiles so it is always watertight.
func honeycomb(w, d, h, cell, wall, border float64) *Mesh {
	m := &Mesh{}
	x0, y0, x1, y1 := -w/2, -d/2, w/2, d/2
	pitch := cell + wall
	R := pitch / math.Sqrt(3) // lattice tile (Voronoi) hexagon circumradius
	r := cell / math.Sqrt(3)
	dy := pitch * math.Sqrt(3) / 2
	var holes []Loop
	addFace := func(a, b, c P2) {
		m.Add(V(a.X, a.Y, h), V(b.X, b.Y, h), V(c.X, c.Y, h))
		m.Add(V(a.X, a.Y, 0), V(c.X, c.Y, 0), V(b.X, b.Y, 0))
	}
	rows := int(d/dy) + 3
	cols := int(w/pitch) + 3
	for row := -rows; row <= rows; row++ {
		yc := float64(row) * dy
		off := 0.0
		if row%2 != 0 {
			off = pitch / 2
		}
		for col := -cols; col <= cols; col++ {
			xc := float64(col)*pitch + off
			tile := NGon(R, 6, 30)
			for i := range tile {
				tile[i].X += xc
				tile[i].Y += yc
			}
			tile = clipConvex(tile, x0, y0, x1, y1)
			if len(tile) < 3 || math.Abs(tile.Area()) < 1e-9 {
				continue
			}
			c := P2{xc, yc}
			inside := math.Abs(xc)+r <= w/2-border && math.Abs(yc)+cell/2 <= d/2-border && len(holes) < 4000
			if inside {
				hx := NGon(r, 6, 30)
				for i := range hx {
					hx[i].X += xc
					hx[i].Y += yc
				}
				holes = append(holes, hx)
				for _, t := range ringTris(tile, hx, c) {
					addFace(t[0], t[1], t[2])
				}
			} else {
				for i := 1; i+1 < len(tile); i++ {
					addFace(tile[0], tile[i], tile[i+1])
				}
			}
		}
	}
	// walls: outer rectangle (CCW, outward) and holes (CW, facing into the hole)
	wallLoop := func(l Loop) {
		for i := range l {
			a, b := l[i], l[(i+1)%len(l)]
			m.AddQuad(V(a.X, a.Y, 0), V(b.X, b.Y, 0), V(b.X, b.Y, h), V(a.X, a.Y, h))
		}
	}
	wallLoop(Loop{{x0, y0}, {x1, y0}, {x1, y1}, {x0, y1}})
	for _, hx := range holes {
		wallLoop(hx.Reversed())
	}
	return m.FixTJunctions().Orient()
}
