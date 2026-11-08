package geom

import (
	"fmt"
	"math"
)

func hwThreadHole(m metric, z0, z1 float64, p Params, threads bool) *Mesh {
	if threads {
		return Thread(ThreadSpec{Major: m.D + 0.05, Pitch: m.Pitch, Profile: "ISO metric 60°", Starts: 1}, z0, z1, p.Q())
	}
	return cylAt(0, 0, m.D, z0, z1, p.Segs(m.D))
}

func hwShank(m metric, z0, z1 float64, p Params, threads bool) *Mesh {
	if threads {
		return Thread(ThreadSpec{Major: m.D, Pitch: m.Pitch, Profile: "ISO metric 60°", Starts: 1, Chamfer: true}, z0, z1, p.Q())
	}
	return cylAt(0, 0, m.D, z0, z1, p.Segs(m.D))
}

func init() {
	sz := metricList()
	hw := func(typ, name, desc, tags string, params []Param, gen func(p Params) (*Mesh, error)) {
		register(&ShapeDef{Type: typ, Name: name, Category: "Hardware", Desc: desc, Tags: tags, Fixed: true, Color: "#94a3b8", Params: params, Gen: gen})
	}
	thr := boolp("threads", "Show real threads", true)
	size := func(def string) Param { return sel("size", "Size", def, sz...) }

	hw("wingnut", "Wing nut", "Nut with two wings for tightening by hand.", "wing nut butterfly hand",
		[]Param{size("M5"), thr},
		func(p Params) (*Mesh, error) {
			m := getMetric(p)
			h := m.NutH * 1.6
			hub := coneAt(m.NutAF*1.15, m.NutAF*0.9, 0, h, p.Segs(m.NutAF))
			w := m.NutAF * 1.25
			wing := Loop{{m.NutAF * 0.3, 0}, {m.NutAF*0.4 + w, h * 0.5}, {m.NutAF*0.4 + w, h * 1.9}, {m.NutAF * 0.25, h}}
			t := math.Max(m.D*0.35, 1.2)
			wings := Union(prismY(wing, t), prismY(wing, t).Transformed(RotZ(180)))
			body := Union(hub, wings)
			return Subtract(body, hwThreadHole(m, -1, h+1, p, p.B("threads", true))), nil
		})

	hw("capnut", "Acorn cap nut", "Domed nut that covers the end of the bolt.", "acorn cap dome nut",
		[]Param{size("M5"), thr},
		func(p Params) (*Mesh, error) {
			m := getMetric(p)
			h := m.NutH
			body := Union(hexPrism(m.NutAF, 0, h), RevolveProfile(append([]P2{{0, h - 0.01}}, func() []P2 {
				var pts []P2
				r := m.NutAF * 0.48
				for i := 0; i <= 10; i++ {
					a := math.Pi / 2 * float64(i) / 10
					pts = append(pts, P2{r * math.Cos(a), h - 0.01 + r*1.1*math.Sin(a)})
				}
				pts[len(pts)-1].X = 0
				return pts
			}()...), p.Segs(m.NutAF)))
			return Subtract(body, hwThreadHole(m, -1, h+m.D*0.5, p, p.B("threads", true))), nil
		})

	hw("flangenut", "Flange nut", "Hex nut with a built-in washer flange that spreads the load.", "flange serrated nut",
		[]Param{size("M5"), thr},
		func(p Params) (*Mesh, error) {
			m := getMetric(p)
			fl := coneAt(m.NutAF*1.45, m.NutAF*1.05, 0, m.NutH*0.3, p.Segs(m.NutAF*1.5))
			body := Union(fl, hexPrism(m.NutAF, m.NutH*0.3-0.01, m.NutH*1.2))
			return Subtract(body, hwThreadHole(m, -1, m.NutH*1.3, p, p.B("threads", true))), nil
		})

	hw("couplingnut", "Coupling nut", "Long hex nut that joins two threaded rods.", "coupling long nut extension",
		[]Param{size("M6"), thr},
		func(p Params) (*Mesh, error) {
			m := getMetric(p)
			h := m.D * 3
			return Subtract(hexPrism(m.NutAF, 0, h), hwThreadHole(m, -1, h+1, p, p.B("threads", true))), nil
		})

	hw("knurlnut", "Knurled thumb nut", "Round knurled nut for finger tightening.", "knurled thumb nut finger",
		[]Param{size("M4"), thr},
		func(p Params) (*Mesh, error) {
			m := getMetric(p)
			d, h := m.NutAF*1.9, m.NutH*1.4
			body := Extrude(Shape{Outer: knurlLoop(d, d*0.06, 24, p.Segs(d))}, h, 0, 1, 1)
			return Subtract(body, hwThreadHole(m, -1, h+1, p, p.B("threads", true))), nil
		})

	hw("thumbscrew", "Thumb screw", "Screw with a big knurled head — no tools needed.", "thumb screw knurled hand",
		[]Param{size("M4"), num("length", "Length", 12, 2, 200, 1, "mm"), thr},
		func(p Params) (*Mesh, error) {
			m := getMetric(p)
			d := m.NutAF * 2
			head := Extrude(Shape{Outer: knurlLoop(d, d*0.06, 20, p.Segs(d))}, m.D*1.3, 0, 1, 1)
			return Union(head, hwShank(m, -p.F("length", 12), 0.01, p, p.B("threads", true))), nil
		})

	hw("eyebolt", "Eye bolt", "Bolt with a ring on top for ropes, springs and hooks.", "eye bolt ring loop anchor",
		[]Param{size("M6"), num("length", "Thread length", 20, 3, 200, 1, "mm"), thr},
		func(p Params) (*Mesh, error) {
			m := getMetric(p)
			R := m.D * 1.3
			eye := Revolve(Shape{Outer: moveLoop(Circle(m.D*0.45, p.Segs(m.D)), R, 0)}, 360, p.Segs(R*2)).Transformed(Translate(0, 0, R+m.D*0.6).Mul(RotX(90)))
			collar := cylAt(0, 0, m.D*1.8, 0, m.D*0.6+0.2, p.Segs(m.D*2))
			return UnionAll([]*Mesh{eye, collar, hwShank(m, -p.F("length", 20), 0.01, p, p.B("threads", true))}), nil
		})

	hw("ubolt", "U-bolt", "U-shaped bolt for clamping tubes and posts.", "u bolt clamp pipe",
		[]Param{size("M6"), num("pipe", "Fits pipe Ø", 25, 4, 300, 1, "mm"), num("legs", "Leg length", 30, 5, 300, 1, "mm")},
		func(p Params) (*Mesh, error) {
			m := getMetric(p)
			r := p.F("pipe", 25)/2 + m.D/2
			L := p.F("legs", 30)
			var pts []Vec3
			pts = append(pts, V(-r, 0, -L), V(-r, 0, 0))
			pts = append(pts, arcPts(0, 0, r, 180, 0, 24, false, 0)[1:]...)
			pts = append(pts, V(r, 0, -L))
			return tubePath(pts, m.D, p.Segs(m.D)).Transformed(Translate(0, 0, L)), nil
		})

	hw("screwhook", "Screw hook", "Hook with a threaded shank to screw into a part or wall.", "hook screw cup j hook",
		[]Param{size("M5"), num("length", "Thread length", 15, 3, 200, 1, "mm"), num("open", "Hook size", 14, 4, 200, 1, "mm")},
		func(p Params) (*Mesh, error) {
			m := getMetric(p)
			r := p.F("open", 14) / 2
			pts := []Vec3{V(0, 0, 0), V(0, 0, r)}
			pts = append(pts, arcPts(r, r, r, 180, -60, 30, false, 0)[1:]...)
			hook := tubePath(pts, m.D*0.9, p.Segs(m.D))
			return Union(hook, hwShank(m, -p.F("length", 15), 0.01, p, false)), nil
		})

	hw("pushrivet", "Push rivet", "Printable press-in rivet with springy split legs — joins two plates without tools.", "rivet push pin clip fastener",
		[]Param{num("hole", "Hole Ø", 4, 2, 20, 0.1, "mm"), num("grip", "Material thickness", 4, 1, 50, 0.5, "mm")},
		func(p Params) (*Mesh, error) {
			d, g := p.F("hole", 4), p.F("grip", 4)
			head := coneAt(d*2.2, d*1.6, g, g+d*0.5, p.Segs(d*2))
			shank := cylAt(0, 0, d*0.95, 0, g+0.01, p.Segs(d))
			barb := coneAt(d*1.25, d*0.7, -d*0.9, 0.01, p.Segs(d))
			tip := Union(barb, cylAt(0, 0, d*0.75, -d*1.2, -d*0.89, p.Segs(d)))
			body := UnionAll([]*Mesh{head, shank, tip})
			slot := boxAt(V(-d, -d*0.1, -d*1.3), V(d, d*0.1, g*0.6))
			return Subtract(body, slot), nil
		})

	hw("dowel", "Dowel pin", "Chamfered locating pin for precise alignment between parts.", "dowel pin locating peg",
		[]Param{num("d", "Diameter", 6, 1, 50, 0.5, "mm"), num("length", "Length", 20, 2, 300, 1, "mm")},
		func(p Params) (*Mesh, error) {
			d, l := p.F("d", 6), p.F("length", 20)
			c := d * 0.12
			return RevolveProfile([]P2{{0, 0}, {d/2 - c, 0}, {d / 2, c}, {d / 2, l - c}, {d/2 - c, l}, {0, l}}, p.Segs(d)), nil
		})

	hw("lockwasher", "Spring lock washer", "Split helical washer that stops nuts loosening.", "lock spring split washer",
		[]Param{size("M5")},
		func(p Params) (*Mesh, error) {
			m := getMetric(p)
			id, od := m.Normal, m.Normal+m.D*0.9
			t := m.D * 0.25
			sq := Shape{Outer: Loop{{id / 2, 0}, {od / 2, 0}, {od / 2, t}, {id / 2, t}}}
			var frames []Frame
			n := p.Segs(od)
			for i := 0; i <= n; i++ {
				a := 2 * math.Pi * float64(i) / float64(n) * 0.97
				sn, cs := math.Sincos(a)
				frames = append(frames, Frame{V(0, 0, t*0.9*float64(i)/float64(n)), V(cs, sn, 0), V(0, 0, 1)})
			}
			return Sweep(sq, frames, false), nil
		})

	hw("shoulderbolt", "Shoulder bolt", "Bolt with a smooth precision shoulder — a perfect pivot pin for levers and pulleys.", "shoulder bolt pivot axle",
		[]Param{size("M5"), num("shoulder", "Shoulder length", 10, 1, 200, 0.5, "mm"), num("length", "Thread length", 8, 2, 100, 1, "mm"), thr},
		func(p Params) (*Mesh, error) {
			m := getMetric(p)
			sd := m.D * 1.25
			sl := p.F("shoulder", 10)
			head := RevolveProfile([]P2{{0, 0}, {m.SocketD * 0.55, 0}, {m.SocketD * 0.55, m.D * 0.9}, {0, m.D * 0.9}}, p.Segs(m.SocketD))
			head = Subtract(head, hexPrism(m.Key, m.D*0.4, m.D))
			sh := cylAt(0, 0, sd, -sl, 0.01, p.Segs(sd))
			return UnionAll([]*Mesh{head, sh, hwShank(m, -sl-p.F("length", 8), -sl+0.01, p, p.B("threads", true))}), nil
		})

	hw("carriagebolt", "Carriage bolt", "Dome-headed bolt with a square neck that locks into a square hole.", "carriage coach bolt dome square neck",
		[]Param{size("M6"), num("length", "Length", 25, 3, 300, 1, "mm"), thr},
		func(p Params) (*Mesh, error) {
			m := getMetric(p)
			r := m.D * 1.15
			var pts []P2
			for i := 0; i <= 10; i++ {
				a := math.Pi / 2 * float64(i) / 10
				pts = append(pts, P2{r * math.Cos(a), m.D * 0.55 * math.Sin(a)})
			}
			pts = append([]P2{{0, 0}}, pts...)
			pts[len(pts)-1].X = 0
			head := RevolveProfile(pts, p.Segs(r*2))
			neck := boxAt(V(-m.D/2, -m.D/2, -m.D*0.6), V(m.D/2, m.D/2, 0.01))
			return UnionAll([]*Mesh{head, neck, hwShank(m, -p.F("length", 25), -m.D*0.59, p, p.B("threads", true))}), nil
		})

	hw("brassinsert", "Heat-set insert", "Model of a knurled brass threaded insert (to check fit in your design).", "heat set insert brass knurled",
		[]Param{sel("size", "Size", "M3", "M2", "M2.5", "M3", "M4", "M5", "M6", "M8")},
		func(p Params) (*Mesh, error) {
			m := getMetric(p)
			d, l := m.InsertD+0.3, m.InsertL
			body := Extrude(Shape{Outer: knurlLoop(d, d*0.08, 14, p.Segs(d))}, l, 25, 1, 6)
			body = Subtract(body, ring(d+2, d*0.85, l*0.4, l*0.55, p.Segs(d)))
			return Subtract(body, hwThreadHole(m, -1, l+1, p, false)), nil
		})

	hw("clevispin", "Clevis pin", "Headed pin with a cross hole for a split pin or R-clip.", "clevis pin cotter hinge pin",
		[]Param{num("d", "Pin Ø", 6, 2, 40, 0.5, "mm"), num("length", "Grip length", 20, 3, 200, 1, "mm")},
		func(p Params) (*Mesh, error) {
			d, l := p.F("d", 6), p.F("length", 20)
			pin := RevolveProfile([]P2{{0, -l - d*0.8}, {d/2 - d*0.1, -l - d*0.8}, {d / 2, -l - d*0.7}, {d / 2, 0}, {d * 0.85, 0}, {d * 0.85, d * 0.4}, {0, d * 0.4}}, p.Segs(d*1.7))
			cross := along(cylAt(0, 0, d*0.35, -d, d, p.Segs(d)), 1).Transformed(Translate(0, 0, -l-d*0.4))
			return Subtract(pin, cross), nil
		})

	hw("circlip", "Circlip / E-clip", "Retaining clip that snaps into a groove on a shaft.", "circlip e-clip retaining ring snap ring",
		[]Param{num("shaft", "Shaft Ø", 8, 2, 60, 0.5, "mm"), num("t", "Thickness", 1, 0.4, 5, 0.1, "mm")},
		func(p Params) (*Mesh, error) {
			s, t := p.F("shaft", 8), p.F("t", 1)
			id, od := s*0.82, s*1.6
			n := p.Segs(od)
			var outer, inner Loop
			gap := 55.0
			for i := 0; i <= n; i++ {
				a := (gap/2 + (360-gap)*float64(i)/float64(n)) * math.Pi / 180
				outer = append(outer, P2{od / 2 * math.Cos(a), od / 2 * math.Sin(a)})
			}
			for i := n; i >= 0; i-- {
				a := (gap/2 + 8 + (360-gap-16)*float64(i)/float64(n)) * math.Pi / 180
				inner = append(inner, P2{id / 2 * math.Cos(a), id / 2 * math.Sin(a)})
			}
			return Extrude(Shape{Outer: append(outer, inner...)}, t, 0, 1, 1), nil
		})

	hw("lmbearing", "Linear bearing", "LM-series linear ball bushing (e.g. LM8UU) for sliding on smooth rods.", "linear bearing lm8uu bushing rod slide",
		[]Param{sel("model", "Model", "LM8UU (8×15×24)", "LM6UU (6×12×19)", "LM8UU (8×15×24)", "LM10UU (10×19×29)", "LM12UU (12×21×30)")},
		func(p Params) (*Mesh, error) {
			dims := map[string][3]float64{"LM6UU (6×12×19)": {6, 12, 19}, "LM8UU (8×15×24)": {8, 15, 24}, "LM10UU (10×19×29)": {10, 19, 29}, "LM12UU (12×21×30)": {12, 21, 30}}
			d := dims[p.S("model", "LM8UU (8×15×24)")]
			if d[0] == 0 {
				d = dims["LM8UU (8×15×24)"]
			}
			b := ring(d[1], d[0], 0, d[2], p.Segs(d[1]))
			g := d[1] * 0.06
			for _, z := range []float64{d[2] * 0.12, d[2] * 0.88} {
				b = Subtract(b, ring(d[1]+2, d[1]-2*g, z-0.6, z+0.6, p.Segs(d[1])))
			}
			return b, nil
		})

	hw("ballbearing", "Ball bearing", "Model of a deep-groove ball bearing (608, 625, 6001…).", "ball bearing 608 skate",
		[]Param{sel("bearing", "Bearing", "608 (8×22×7)", "623 (3×10×4)", "MR105 (5×10×4)", "625 (5×16×5)", "688 (8×16×5)", "608 (8×22×7)", "6001 (12×28×8)", "6200 (10×30×9)")},
		func(p Params) (*Mesh, error) {
			bs := map[string][3]float64{"623 (3×10×4)": {3, 10, 4}, "625 (5×16×5)": {5, 16, 5}, "608 (8×22×7)": {8, 22, 7}, "688 (8×16×5)": {8, 16, 5}, "6001 (12×28×8)": {12, 28, 8}, "6200 (10×30×9)": {10, 30, 9}, "MR105 (5×10×4)": {5, 10, 4}}
			b := bs[p.S("bearing", "608 (8×22×7)")]
			if b[1] == 0 {
				b = bs["608 (8×22×7)"]
			}
			w := (b[1] - b[0]) / 2
			outer := ring(b[1], b[1]-w*0.6, 0, b[2], p.Segs(b[1]))
			inner := ring(b[0]+w*0.6, b[0], 0, b[2], p.Segs(b[1]))
			shield := ring(b[1]-w*0.55, b[0]+w*0.55, b[2]*0.1, b[2]*0.9, p.Segs(b[1]))
			return UnionAll([]*Mesh{outer, inner, shield}), nil
		})

	hw("wallplug", "Wall plug", "Ribbed expansion anchor for screwing into walls.", "wall plug anchor rawl expansion",
		[]Param{num("d", "Plug Ø", 6, 4, 16, 1, "mm"), num("length", "Length", 30, 10, 100, 1, "mm")},
		func(p Params) (*Mesh, error) {
			d, l := p.F("d", 6), p.F("length", 30)
			pts := []P2{{0, 0}, {d * 0.38, 0}}
			n := int(l / (d * 0.5))
			for i := 0; i < n; i++ {
				z := l*0.15 + (l*0.8)*float64(i)/float64(n)
				pts = append(pts, P2{d * 0.45, z}, P2{d * 0.55, z + d*0.3})
			}
			pts = append(pts, P2{d * 0.5, l - d*0.2}, P2{d * 0.75, l - d*0.2}, P2{d * 0.75, l}, P2{0, l})
			body := RevolveProfile(pts, p.Segs(d*1.5))
			body = Subtract(body, cylAt(0, 0, d*0.45, l*0.12, l+1, p.Segs(d)))
			return Subtract(body, boxAt(V(-d, -d*0.07, -1), V(d, d*0.07, l*0.6))), nil
		})

	hw("rollpin", "Roll pin", "Slotted spring pin that grips in its hole.", "roll pin spring pin slotted",
		[]Param{num("d", "Diameter", 5, 1.5, 30, 0.5, "mm"), num("length", "Length", 20, 3, 200, 1, "mm")},
		func(p Params) (*Mesh, error) {
			d, l := p.F("d", 5), p.F("length", 20)
			t := ring(d, d*0.7, 0, l, p.Segs(d))
			return Subtract(t, boxAt(V(0, -d*0.08, -1), V(d, d*0.08, l+1))), nil
		})

	hw("hexkey", "Hex key (Allen key)", "L-shaped hex key — print one that fits your design's screws.", "hex key allen wrench tool",
		[]Param{num("af", "Across flats", 4, 1.5, 14, 0.5, "mm"), num("long", "Long arm", 60, 10, 300, 1, "mm"), num("short", "Short arm", 20, 5, 100, 1, "mm")},
		func(p Params) (*Mesh, error) {
			af, L, s := p.F("af", 4), p.F("long", 60), p.F("short", 20)
			a := hexPrism(af, 0, L)
			b := along(hexPrism(af, 0, s), 0)
			return Union(a, b.Transformed(Translate(-af*0.5, 0, 0))), nil
		})

	hw("cagenut", "Cage nut", "Square nut in a spring cage for square-hole racks.", "cage nut rack server",
		[]Param{size("M6")},
		func(p Params) (*Mesh, error) {
			m := getMetric(p)
			s := m.D * 1.6
			nut := boxAt(V(-s/2, -s/2, 0), V(s/2, s/2, m.D*0.7))
			cage := boxAt(V(-s*0.75, -s*0.6, m.D*0.7-0.01), V(s*0.75, s*0.6, m.D*0.7+0.6))
			wing := prismY(Loop{{s * 0.75, 0}, {s * 0.95, m.D * 1.3}, {s * 0.85, m.D * 1.4}, {s * 0.65, m.D * 0.6}}, s*0.8)
			body := UnionAll([]*Mesh{nut, cage, wing, wing.Transformed(RotZ(180))})
			return Subtract(body, cylAt(0, 0, m.D, -1, m.D*2, p.Segs(m.D))), nil
		})

	hw("bushing", "Flanged bushing", "Plain sleeve bearing with a flange — print in nylon or PETG for low friction.", "bushing sleeve bearing flange plain",
		[]Param{num("shaft", "Shaft Ø", 8, 1, 100, 0.5, "mm"), num("wall", "Wall", 2, 0.8, 20, 0.2, "mm"), num("length", "Length", 10, 2, 200, 1, "mm"), num("clear", "Running clearance", 0.15, 0, 1, 0.05, "mm")},
		func(p Params) (*Mesh, error) {
			s, w, l := p.F("shaft", 8)+p.F("clear", 0.15), p.F("wall", 2), p.F("length", 10)
			return Revolve(Shape{Outer: Loop{{s / 2, 0}, {s/2 + w*2.2, 0}, {s/2 + w*2.2, w}, {s/2 + w, w}, {s/2 + w, l}, {s / 2, l}}}, 360, p.Segs(s+w*4)), nil
		})

	hw("levelfoot", "Levelling foot", "Adjustable foot: threaded stud on a swivel pad.", "levelling foot leg adjustable",
		[]Param{size("M8"), num("pad", "Pad Ø", 30, 10, 150, 1, "mm"), num("length", "Stud length", 25, 5, 200, 1, "mm")},
		func(p Params) (*Mesh, error) {
			m := getMetric(p)
			D := p.F("pad", 30)
			pad := RevolveProfile([]P2{{0, 0}, {D / 2, 0}, {D / 2, D * 0.08}, {D * 0.3, D * 0.2}, {0, D * 0.2}}, p.Segs(D))
			nut := hexPrism(m.NutAF, D*0.2-0.01, D*0.2+m.NutH)
			st := hwShank(m, D*0.2+m.NutH-0.01, D*0.2+m.NutH+p.F("length", 25), p, false)
			return UnionAll([]*Mesh{pad, nut, st}), nil
		})

	hw("tnut", "Extrusion T-nut", "Drop-in T-nut for 2020 / 3030 / 4040 aluminium extrusion slots.", "t nut extrusion 2020 v-slot slot nut",
		[]Param{sel("profile", "Extrusion", "2020", "2020", "3030", "4040"), size("M5")},
		func(p Params) (*Mesh, error) {
			m := getMetric(p)
			k := map[string]float64{"2020": 1, "3030": 1.4, "4040": 1.8}[p.S("profile", "2020")]
			if k == 0 {
				k = 1
			}
			w, l := 10*k, 10*k
			prof := Loop{{-w / 2, 0}, {w / 2, 0}, {w/2 - 1.5*k, 2.2 * k}, {3 * k, 2.2 * k}, {3 * k, 4 * k}, {-3 * k, 4 * k}, {-3 * k, 2.2 * k}, {-w/2 + 1.5*k, 2.2 * k}}
			body := prismY(prof, l)
			hd := m.D * 0.85
			if hd+1.6 > 6*k { // the slot walls support the nut, so 0.8 mm each side is enough
				return nil, fmt.Errorf("%s is too big for a %s T-nut (the neck is only %.0f mm wide) — use a smaller screw or a bigger extrusion", p.S("size", "M5"), p.S("profile", "2020"), 6*k)
			}
			return Subtract(body, cylAt(0, 0, hd, -1, 5*k, p.Segs(m.D))), nil
		})
}
