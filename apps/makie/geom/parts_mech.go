package geom

import "math"

func boreFor(m *Mesh, d, z0, z1 float64, segs int) *Mesh {
	if d <= 0 {
		return m
	}
	return Subtract(m, cylAt(0, 0, d, z0, z1, segs))
}

func init() {
	mech := func(typ, name, desc, tags, color string, params []Param, gen func(p Params) (*Mesh, error)) {
		register(&ShapeDef{Type: typ, Name: name, Category: "Mechanical", Desc: desc, Tags: tags, Color: color, Params: params, Gen: gen})
	}
	bore := func(def float64) Param { return num("bore", "Bore Ø", def, 0, 200, 0.1, "mm") }

	// ---------------- power transmission ----------------
	mech("gt2pulley", "GT2 timing pulley", "Toothed pulley for GT2 (2 mm pitch) belts, with flanges and a set-screw hub.", "gt2 timing pulley belt 2gt 20 tooth", "#0ea5e9",
		[]Param{num("teeth", "Teeth", 20, 10, 120, 1, ""), num("belt", "Belt width", 6, 3, 20, 1, "mm"), bore(5), boolp("hub", "Hub with set screw", true)},
		func(p Params) (*Mesh, error) {
			z, bw := p.I("teeth", 20), p.F("belt", 6)
			pd := 2 * float64(z) / math.Pi
			od := pd - 0.508
			var l Loop
			n := z * 10
			for i := 0; i < n; i++ {
				a := 2 * math.Pi * float64(i) / float64(n)
				ph := math.Mod(float64(i), 10) / 10
				r := od / 2
				if ph > 0.25 && ph < 0.75 {
					r -= 0.75 * math.Sin((ph-0.25)*2*math.Pi)
				}
				l = append(l, P2{r * math.Cos(a), r * math.Sin(a)})
			}
			fl := od + 2.5
			m := UnionAll([]*Mesh{cylAt(0, 0, fl, 0, 1.2, p.Segs(fl)), extrudeAt(Shape{Outer: l}, 1.19, bw+1.21), coneAt(fl, fl, bw+1.2, bw+2.4, p.Segs(fl))})
			top := bw + 2.4
			if p.B("hub", true) {
				hd := math.Max(p.F("bore", 5)*2.4, od*0.8)
				m = Union(m, cylAt(0, 0, hd, top-0.01, top+7, p.Segs(hd)))
				m = Subtract(m, along(cylAt(0, 0, 2.9, 0, hd, 12), 0).Transformed(Translate(0, 0, top+3.5)))
				top += 7
			}
			return boreFor(m, p.F("bore", 5)+0.15, -1, top+1, p.Segs(8)), nil
		})

	mech("vpulley", "V-belt / rope pulley", "Grooved pulley for round belts, O-rings, ropes and cords.", "v pulley sheave rope round belt wheel groove", "#0284c7",
		[]Param{num("d", "Outer Ø", 40, 8, 500, 1, "mm"), num("w", "Width", 10, 3, 100, 1, "mm"), num("groove", "Groove depth", 3, 0.5, 40, 0.5, "mm"), sel("shape", "Groove", "V", "V", "Round"), bore(6)},
		func(p Params) (*Mesh, error) {
			d, w, g := p.F("d", 40), p.F("w", 10), p.F("groove", 3)
			r := d / 2
			pts := []P2{{0, 0}, {r, 0}, {r, w * 0.2}}
			if p.S("shape", "V") == "Round" {
				for i := 0; i <= 12; i++ {
					a := math.Pi*float64(i)/12 - math.Pi/2
					pts = append(pts, P2{r - g*math.Cos(a), w/2 + w*0.3*math.Sin(a)})
				}
			} else {
				pts = append(pts, P2{r - g, w / 2})
			}
			pts = append(pts, P2{r, w * 0.8}, P2{r, w}, P2{0, w})
			return boreFor(RevolveProfile(pts, p.Segs(d)), p.F("bore", 6), -1, w+1, p.Segs(8)), nil
		})

	mech("bevelgear", "Bevel gear", "Cone-shaped gear for turning motion 90° (pair two with the same module).", "bevel gear mitre right angle", "#0891b2",
		[]Param{num("module", "Module", 1.5, 0.5, 8, 0.1, "mm"), num("teeth", "Teeth", 20, 8, 120, 1, ""), num("h", "Face width", 6, 1, 60, 0.5, "mm"), num("cone", "Pitch cone angle", 45, 10, 80, 1, "°"), bore(5)},
		func(p Params) (*Mesh, error) {
			mod, z, h := p.F("module", 1.5), p.I("teeth", 20), p.F("h", 6)
			rp := mod * float64(z) / 2
			ang := p.F("cone", 45) * math.Pi / 180
			apex := rp / math.Tan(ang)
			k := math.Max(0.2, 1-h/math.Max(apex, h*1.1))
			g := Extrude(Shape{Outer: GearOutline(mod, z, 20, 0.1, p.Q())}, h, 0, k, 6)
			return boreFor(g, p.F("bore", 5), -1, h+1, p.Segs(8)), nil
		})

	mech("ringgear", "Internal ring gear", "Gear with teeth on the inside — the outer ring of a planetary gearbox.", "ring gear internal annulus planetary", "#0e7490",
		[]Param{num("module", "Module", 1.5, 0.5, 8, 0.1, "mm"), num("teeth", "Teeth", 40, 12, 200, 1, ""), num("h", "Thickness", 6, 1, 100, 0.5, "mm"), num("rim", "Rim thickness", 4, 1, 50, 0.5, "mm")},
		func(p Params) (*Mesh, error) {
			mod, z := p.F("module", 1.5), p.I("teeth", 40)
			inner := GearOutline(mod, z, 20, -0.3, p.Q()) // negative backlash = clearance for the mating gear
			od := mod*float64(z) + 2*mod + 2*p.F("rim", 4)
			return Extrude(Shape{Outer: Circle(od/2, p.Segs(od)), Holes: []Loop{inner}}, p.F("h", 6), 0, 1, 1), nil
		})

	mech("worm", "Worm screw", "Worm that drives a spur gear at a big reduction (1 turn = 1 tooth). Use a gear with the same module.", "worm gear drive reduction screw", "#06b6d4",
		[]Param{num("module", "Module", 1.5, 0.5, 8, 0.1, "mm"), num("len", "Length", 30, 5, 300, 1, "mm"), num("d", "Pitch Ø", 12, 4, 100, 0.5, "mm"), num("starts", "Starts", 1, 1, 4, 1, ""), bore(4)},
		func(p Params) (*Mesh, error) {
			mod, l, d := p.F("module", 1.5), p.F("len", 30), p.F("d", 12)
			sp := ThreadSpec{Major: d + 2*mod, Pitch: math.Pi * mod, Profile: "Trapezoidal 30°", Starts: p.I("starts", 1), Chamfer: true}
			w := Thread(sp, 0, l, p.Q())
			return boreFor(w, p.F("bore", 4), -1, l+1, p.Segs(8)), nil
		})

	mech("sprocket", "Chain sprocket", "Sprocket for roller chain (#25 / 04C / 06B / bicycle).", "sprocket chain bike roller", "#475569",
		[]Param{num("teeth", "Teeth", 18, 8, 80, 1, ""), sel("chain", "Chain", "#25 (6.35 mm)", "#25 (6.35 mm)", "06B (9.525 mm)", "Bicycle 1/2\" (12.7 mm)"), bore(8)},
		func(p Params) (*Mesh, error) {
			ch := map[string][3]float64{"#25 (6.35 mm)": {6.35, 3.3, 2.8}, "06B (9.525 mm)": {9.525, 6.35, 5.2}, "Bicycle 1/2\" (12.7 mm)": {12.7, 7.75, 2.2}}[p.S("chain", "#25 (6.35 mm)")]
			if ch[0] == 0 {
				ch = [3]float64{6.35, 3.3, 2.8}
			}
			z := p.I("teeth", 18)
			pd := ch[0] / math.Sin(math.Pi/float64(z))
			od := pd + ch[1]*0.8
			holes := []Loop{}
			for i := 0; i < z; i++ {
				a := 2 * math.Pi * float64(i) / float64(z)
				holes = append(holes, moveLoop(Circle(ch[1]/2*1.02, 16), pd/2*math.Cos(a), pd/2*math.Sin(a)))
			}
			body := Extrude(Shape{Outer: Circle(od/2, p.Segs(od))}, ch[2], 0, 1, 1)
			cut := &Mesh{}
			for _, h := range holes {
				cut.Append(extrudeAt(Shape{Outer: h}, -1, ch[2]+1))
			}
			// open the tooth gaps outward
			for i := 0; i < z; i++ {
				a := 360 * float64(i) / float64(z)
				cut = Union(cut, boxAt(V(pd/2, -ch[1]*0.49, -1), V(od, ch[1]*0.49, ch[2]+1)).Transformed(RotZ(a)))
			}
			s := Subtract(body, cut)
			return boreFor(s, p.F("bore", 8), -1, ch[2]+1, p.Segs(8)), nil
		})

	mech("cam", "Cam", "Eccentric, heart or pear cam — turns rotation into up-and-down motion.", "cam eccentric heart follower lobe", "#9333ea",
		[]Param{sel("kind", "Shape", "Eccentric", "Eccentric", "Heart (constant speed)", "Pear (snap)"), num("base", "Base Ø", 30, 5, 300, 1, "mm"), num("lift", "Lift", 8, 0.5, 100, 0.5, "mm"), num("h", "Thickness", 6, 1, 50, 0.5, "mm"), bore(5)},
		func(p Params) (*Mesh, error) {
			b, lift := p.F("base", 30)/2, p.F("lift", 8)
			n := p.Segs(b*2 + lift)
			var l Loop
			for i := 0; i < n; i++ {
				a := 2 * math.Pi * float64(i) / float64(n)
				var r float64
				switch p.S("kind", "Eccentric") {
				case "Heart (constant speed)":
					t := a / math.Pi
					if t > 1 {
						t = 2 - t
					}
					r = b + lift*t
				case "Pear (snap)":
					r = b + lift*math.Pow(math.Max(0, math.Cos(a/2)), 6)
				default:
					x, y := lift/2+b*math.Cos(a), b*math.Sin(a)
					l = append(l, P2{x, y})
					continue
				}
				l = append(l, P2{r * math.Cos(a), r * math.Sin(a)})
			}
			return boreFor(Extrude(Shape{Outer: l}, p.F("h", 6), 0, 1, 1), p.F("bore", 5), -1, p.F("h", 6)+1, p.Segs(8)), nil
		})

	mech("crank", "Crank arm", "Arm with a shaft hub at one end and a handle pin at the other.", "crank arm handle lever winder", "#c026d3",
		[]Param{num("len", "Throw (centre to centre)", 40, 5, 500, 1, "mm"), num("t", "Thickness", 5, 1, 50, 0.5, "mm"), bore(6), boolp("knob", "Handle knob", true)},
		func(p Params) (*Mesh, error) {
			l, t := p.F("len", 40), p.F("t", 5)
			hub := 16.0
			arm := extrudeAt(Shape{Outer: append(arcLoop(0, 0, hub/2, 90, 270), arcLoop(l, 0, hub*0.35, -90, 90)...)}, 0, t)
			arm = Union(arm, cylAt(0, 0, hub, 0, t+4, p.Segs(hub)))
			if p.B("knob", true) {
				arm = Union(arm, RevolveProfile([]P2{{0, t - 0.01}, {3, t - 0.01}, {3, t + 4}, {5, t + 6}, {5, t + 18}, {3.5, t + 20}, {0, t + 20}}, 24).Transformed(Translate(l, 0, 0)))
			}
			return boreFor(arm, p.F("bore", 6), -1, t+5, p.Segs(8)), nil
		})

	mech("linkbar", "Linkage bar", "Flat link with a pivot hole at each end — for levers, four-bar linkages and arms.", "link bar linkage lever arm pivot", "#a21caf",
		[]Param{num("len", "Hole spacing", 40, 3, 1000, 1, "mm"), num("w", "Width", 10, 2, 100, 0.5, "mm"), num("t", "Thickness", 4, 0.6, 50, 0.2, "mm"), num("hole", "Hole Ø", 3.4, 0, 50, 0.1, "mm"), num("extra", "Extra holes between", 0, 0, 50, 1, "")},
		func(p Params) (*Mesh, error) {
			l, w, t, h := p.F("len", 40), p.F("w", 10), p.F("t", 4), p.F("hole", 3.4)
			b := extrudeAt(Shape{Outer: stadium(l+w, w, p.Segs(w))}, 0, t)
			pts := [][2]float64{{-l / 2, 0}, {l / 2, 0}}
			n := p.I("extra", 0)
			for i := 1; i <= n; i++ {
				pts = append(pts, [2]float64{-l/2 + l*float64(i)/float64(n+1), 0})
			}
			return holesThrough(b, pts, h, -1, t+1, p.Segs(h)), nil
		})

	mech("coupler", "Shaft coupler", "Rigid coupler joining two shafts (e.g. a 5 mm motor to an 8 mm lead screw), with set-screw holes.", "shaft coupler coupling motor lead screw rigid", "#64748b",
		[]Param{num("d1", "Shaft 1 Ø", 5, 1, 50, 0.1, "mm"), num("d2", "Shaft 2 Ø", 8, 1, 50, 0.1, "mm"), num("len", "Length", 25, 6, 200, 1, "mm"), num("od", "Outer Ø", 19, 4, 100, 0.5, "mm"), sel("style", "Style", "Set screws", "Set screws", "Flexible spiral")},
		func(p Params) (*Mesh, error) {
			d1, d2, l, od := p.F("d1", 5)+0.15, p.F("d2", 8)+0.15, p.F("len", 25), p.F("od", 19)
			m := cylAt(0, 0, od, 0, l, p.Segs(od))
			m = Subtract(m, cylAt(0, 0, d1, -1, l/2, p.Segs(d1)))
			m = Subtract(m, cylAt(0, 0, d2, l/2-0.01, l+1, p.Segs(d2)))
			for _, z := range []float64{l * 0.2, l * 0.8} {
				m = Subtract(m, along(cylAt(0, 0, 2.9, 0, od, 12), 0).Transformed(Translate(0, 0, z)))
			}
			if p.S("style", "") == "Flexible spiral" {
				var frames []Frame
				turns := 2.5
				steps := 90
				for i := 0; i <= steps; i++ {
					t := float64(i) / float64(steps)
					a := 2 * math.Pi * turns * t
					sn, cs := math.Sincos(a)
					frames = append(frames, Frame{V(0, 0, l*0.3+l*0.4*t), V(cs, sn, 0), V(0, 0, 1)})
				}
				cut := Sweep(Shape{Outer: Loop{{od * 0.2, -0.5}, {od, -0.5}, {od, 0.5}, {od * 0.2, 0.5}}}, frames, false)
				m = Subtract(m, cut)
			}
			return m, nil
		})

	mech("yoke", "Universal joint yoke", "Fork for a universal (Cardan) joint — print two plus a cross pin.", "universal joint u joint cardan yoke fork", "#334155",
		[]Param{num("d", "Size", 20, 6, 200, 1, "mm"), bore(5), num("pin", "Pin hole Ø", 3.4, 1, 30, 0.1, "mm")},
		func(p Params) (*Mesh, error) {
			d := p.F("d", 20)
			base := cylAt(0, 0, d, 0, d*0.5, p.Segs(d))
			ear := prismY(Loop{{-d / 2, 0}, {-d * 0.3, 0}, {-d * 0.3, d * 1.1}, {-d / 2, d * 1.1}}, d*0.5)
			m := UnionAll([]*Mesh{base, ear.Transformed(Translate(0, 0, d*0.49)), ear.Transformed(Translate(0, 0, d*0.49).Mul(RotZ(180)))})
			m = Subtract(m, along(cylAt(0, 0, p.F("pin", 3.4), -d, d, 16), 0).Transformed(Translate(0, 0, d*1.25)))
			return boreFor(m, p.F("bore", 5), -1, d*0.4, p.Segs(8)), nil
		})

	mech("pillowblock", "Pillow block", "Bearing housing with a bolt-down foot (holds a 608 or other ball bearing).", "pillow block bearing housing mount shaft support", "#1e40af",
		[]Param{sel("bearing", "Bearing", "608 (22×7)", "623 (10×4)", "625 (16×5)", "608 (22×7)", "6001 (28×8)"), num("bolt", "Bolt hole Ø", 4.5, 2, 20, 0.1, "mm")},
		func(p Params) (*Mesh, error) {
			b := map[string][2]float64{"623 (10×4)": {10, 4}, "625 (16×5)": {16, 5}, "608 (22×7)": {22, 7}, "6001 (28×8)": {28, 8}}[p.S("bearing", "608 (22×7)")]
			if b[0] == 0 {
				b = [2]float64{22, 7}
			}
			od := b[0] + 8
			w := b[1] + 3.2 // 1.6 mm retaining lip behind the bearing
			h := od/2 + 4
			foot := boxAt(V(-od, -w/2-1, 0), V(od, w/2+1, 5))
			up := shapeY(Shape{Outer: append(Loop{{-od / 2, 0}, {od / 2, 0}}, arcLoop(0, h, od/2, 0, 180)...)}, w)
			m := Union(foot, up)
			m = Subtract(m, along(cylAt(0, 0, b[0]+0.1, -b[1]/2, b[1]/2+5, p.Segs(b[0])), 1).Transformed(Translate(0, -b[1]/2+b[1]/2, h)))
			m = Subtract(m, along(cylAt(0, 0, b[0]*0.6, -w, w, p.Segs(b[0])), 1).Transformed(Translate(0, 0, h)))
			return holesThrough(m, [][2]float64{{-od * 0.75, 0}, {od * 0.75, 0}}, p.F("bolt", 4.5), -1, 6, 16), nil
		})

	mech("wheel", "Wheel with tread", "Wheel with a treaded tyre, spokes and a hub — robots, carts and toys.", "wheel tyre tire tread robot cart", "#18181b",
		[]Param{num("d", "Diameter", 60, 10, 500, 1, "mm"), num("w", "Width", 15, 3, 200, 1, "mm"), num("spokes", "Spokes (0 = solid)", 5, 0, 16, 1, ""), num("tread", "Tread grooves", 24, 0, 120, 1, ""), sel("boreType", "Bore", "D-shaft", "Round", "D-shaft", "Hex"), bore(5)},
		func(p Params) (*Mesh, error) {
			d, w := p.F("d", 60), p.F("w", 15)
			tg := p.I("tread", 24)
			var outer Loop
			if tg > 0 {
				outer = knurlLoop(d, math.Min(2.5, d*0.04), tg, p.Segs(d))
			} else {
				outer = Circle(d/2, p.Segs(d))
			}
			tyreT := d * 0.12
			m := Extrude(Shape{Outer: outer, Holes: []Loop{Circle(d/2-tyreT, p.Segs(d))}}, w, 0, 1, 1)
			hub := d * 0.25
			m = Union(m, cylAt(0, 0, hub, 0, w, p.Segs(hub)))
			if n := p.I("spokes", 5); n == 0 {
				m = Union(m, ring(d-2*tyreT+0.2, hub-0.2, w*0.3, w*0.7, p.Segs(d)))
			} else {
				for k := 0; k < n; k++ {
					m = Union(m, boxAt(V(hub*0.4, -d*0.04, w*0.25), V(d/2-tyreT+0.3, d*0.04, w*0.75)).Transformed(RotZ(360*float64(k)/float64(n))))
				}
			}
			bd := p.F("bore", 5)
			return Subtract(m, extrudeAt(Shape{Outer: boreLoop(bd+0.15, p.S("boreType", "D-shaft"), p.Segs(bd))}, -1, w+1)), nil
		})

	mech("balljoint", "Ball joint stud", "Ball on a neck and base — snaps into a Ball socket hole for a swivel joint.", "ball joint stud swivel snap", "#be123c",
		[]Param{num("d", "Ball Ø", 10, 2, 100, 0.5, "mm"), num("neck", "Neck length", 6, 1, 100, 0.5, "mm"), num("base", "Base Ø", 14, 2, 200, 0.5, "mm")},
		func(p Params) (*Mesh, error) {
			d, nl, bd := p.F("d", 10), p.F("neck", 6), p.F("base", 14)
			base := cylAt(0, 0, bd, 0, 3, p.Segs(bd))
			neck := cylAt(0, 0, d*0.55, 2.99, 3+nl+d*0.3, p.Segs(d))
			n := p.Segs(d) / 2
			var pts []P2
			for i := 0; i <= n; i++ {
				a := -math.Pi/2 + math.Pi*float64(i)/float64(n)
				pts = append(pts, P2{d / 2 * math.Cos(a), 3 + nl + d/2 + d/2*math.Sin(a)})
			}
			pts[0].X, pts[n].X = 0, 0
			return UnionAll([]*Mesh{base, neck, RevolveProfile(pts, p.Segs(d))}), nil
		})

	// ---------------- clips, brackets & structure ----------------
	mech("pipeclip", "Pipe / tube clip", "C-shaped clip that snaps onto a tube or pipe, with a screw mounting tab.", "pipe clip tube clamp c clip holder", "#16a34a",
		[]Param{num("pipe", "Pipe Ø", 20, 3, 300, 0.5, "mm"), num("wall", "Wall", 2.5, 0.8, 20, 0.2, "mm"), num("w", "Width", 10, 3, 200, 1, "mm"), num("open", "Opening (% of Ø)", 75, 40, 100, 1, "%")},
		func(p Params) (*Mesh, error) {
			d, t, w := p.F("pipe", 20), p.F("wall", 2.5), p.F("w", 10)
			r := d / 2
			gap := math.Asin(math.Min(0.99, p.F("open", 75)/100)) * 180 / math.Pi
			c := append(arcLoop(0, r+t, r+t, 90+gap, 450-gap), arcLoopRev(0, r+t, r, 90+gap, 450-gap)...)
			clip := shapeY(Shape{Outer: c}, w)
			tab := boxAt(V(-r-t-8, -w/2, 0), V(r+t+8, w/2, t))
			m := Union(clip, tab)
			return holesThrough(m, [][2]float64{{-r - t - 4, 0}, {r + t + 4, 0}}, 3.4, -1, t+1, 16), nil
		})

	mech("cableclip", "Cable clip", "Screw-down cable clip / saddle for tidy wiring.", "cable clip saddle wire holder", "#15803d",
		[]Param{num("cable", "Cable Ø", 6, 1, 50, 0.5, "mm"), num("w", "Width", 8, 3, 50, 0.5, "mm"), num("t", "Thickness", 1.6, 0.6, 6, 0.1, "mm"), num("screw", "Screw Ø", 3.4, 1, 10, 0.1, "mm")},
		func(p Params) (*Mesh, error) {
			c, w, t := p.F("cable", 6), p.F("w", 8), p.F("t", 1.6)
			r := c / 2
			arch := shapeY(Shape{Outer: append(arcLoop(0, 0, r+t, 0, 180), arcLoopRev(0, 0, r, 0, 180)...)}, w)
			tab := boxAt(V(-r-t-8, -w/2, 0), V(-r+0.01, w/2, t))
			tab2 := boxAt(V(r-0.01, -w/2, 0), V(r+t+2, w/2, t))
			return holesThrough(UnionAll([]*Mesh{arch, tab, tab2}), [][2]float64{{-r - t - 4, 0}}, p.F("screw", 3.4), -1, t+1, 16), nil
		})

	mech("cornerbracket", "Corner cube bracket", "Three-way corner block that joins panels at a box corner.", "corner bracket cube three way box joint", "#1d4ed8",
		[]Param{num("s", "Size", 20, 6, 200, 1, "mm"), num("hole", "Screw hole Ø", 3.4, 1, 20, 0.1, "mm"), boolp("countersink", "Countersunk", true)},
		func(p Params) (*Mesh, error) {
			s, h := p.F("s", 20), p.F("hole", 3.4)
			m := boxAt(V(0, 0, 0), V(s, s, s))
			cs := func(axis int) *Mesh {
				c := cylAt(0, 0, h, -1, s+1, 16)
				if p.B("countersink", true) {
					c = Union(c, coneAt(h, h*2.2, s-h*0.6, s+0.01, 16))
				}
				switch axis {
				case 0:
					return c.Transformed(Translate(s/2, s/2, 0))
				case 1:
					return c.Transformed(Translate(s/2, 0, s/2).Mul(RotX(-90)))
				}
				return c.Transformed(Translate(0, s/2, s/2).Mul(RotY(90)))
			}
			return Subtract(Subtract(Subtract(m, cs(0)), cs(1)), cs(2)), nil
		})

	mech("mendplate", "Mending plate", "Flat repair plate — straight, L or T shaped — with countersunk holes.", "mending plate repair flat bracket t plate l plate", "#2563eb",
		[]Param{sel("shape", "Shape", "Straight", "Straight", "L", "T", "Cross"), num("len", "Arm length", 60, 10, 500, 1, "mm"), num("w", "Width", 15, 4, 100, 1, "mm"), num("t", "Thickness", 3, 0.6, 20, 0.2, "mm"), num("hole", "Hole Ø", 4, 1, 20, 0.1, "mm")},
		func(p Params) (*Mesh, error) {
			l, w, t, hd := p.F("len", 60), p.F("w", 15), p.F("t", 3), p.F("hole", 4)
			arms := [][4]float64{{-l / 2, -w / 2, l / 2, w / 2}}
			pts := [][2]float64{{-l/2 + w/2, 0}, {l/2 - w/2, 0}, {-l / 6, 0}, {l / 6, 0}}
			switch p.S("shape", "Straight") {
			case "L":
				arms = [][4]float64{{0, -w / 2, l, w / 2}, {0, -w / 2, w, l}}
				pts = [][2]float64{{l - w/2, 0}, {l * 0.5, 0}, {w / 2, l - w/2}, {w / 2, l * 0.5}, {w / 2, 0}}
			case "T":
				arms = [][4]float64{{-l / 2, -w / 2, l / 2, w / 2}, {-w / 2, -w / 2, w / 2, l * 0.8}}
				pts = [][2]float64{{-l/2 + w/2, 0}, {l/2 - w/2, 0}, {0, 0}, {0, l*0.8 - w/2}, {0, l * 0.4}}
			case "Cross":
				arms = [][4]float64{{-l / 2, -w / 2, l / 2, w / 2}, {-w / 2, -l / 2, w / 2, l / 2}}
				pts = [][2]float64{{-l/2 + w/2, 0}, {l/2 - w/2, 0}, {0, -l/2 + w/2}, {0, l/2 - w/2}}
			}
			var ms []*Mesh
			for _, a := range arms {
				ms = append(ms, roundedBlock(a[2]-a[0], a[3]-a[1], t, w*0.2, 0, 0, false, false, 4).Transformed(Translate((a[0]+a[2])/2, (a[1]+a[3])/2, 0)))
			}
			m := UnionAll(ms)
			var cut []*Mesh
			for _, q := range pts {
				cut = append(cut, Union(cylAt(q[0], q[1], hd, -1, t+1, 16), coneAt(hd, hd*2, t-hd*0.5, t+0.01, 16).Transformed(Translate(q[0], q[1], 0))))
			}
			return Subtract(m, UnionAll(cut)), nil
		})

	mech("shelfbracket", "Shelf bracket", "Triangular bracket with a diagonal brace for shelves and overhangs.", "shelf bracket support triangle brace wall", "#1e3a8a",
		[]Param{num("a", "Wall leg", 80, 20, 500, 1, "mm"), num("b", "Shelf leg", 80, 20, 500, 1, "mm"), num("w", "Width", 20, 5, 100, 1, "mm"), num("t", "Thickness", 5, 1.5, 30, 0.5, "mm"), num("hole", "Screw hole Ø", 4.5, 2, 15, 0.1, "mm")},
		func(p Params) (*Mesh, error) {
			a, b, w, t := p.F("a", 80), p.F("b", 80), p.F("w", 20), p.F("t", 5)
			outer := Loop{{0, 0}, {b, 0}, {b, t}, {t * 2, t}, {t, t * 2}, {t, a}, {0, a}}
			brace := Loop{{t - 0.01, a * 0.75}, {t - 0.01, a*0.75 - t*1.6}, {b*0.75 - t*1.6, t - 0.01}, {b * 0.75, t - 0.01}}
			m := Union(prismY(outer, w), prismY(brace, w))
			m = Subtract(m, cylAt(b*0.5, 0, p.F("hole", 4.5), -1, t+1, 16))
			m = Subtract(m, cylAt(b*0.9, 0, p.F("hole", 4.5), -1, t+1, 16))
			for _, z := range []float64{a * 0.5, a * 0.9} {
				m = Subtract(m, along(cylAt(0, 0, p.F("hole", 4.5), -1, t+1, 16), 0).Transformed(Translate(0, 0, z)))
			}
			return m, nil
		})

	mech("dinclip", "DIN rail clip", "Clip-on mount for standard 35 mm DIN rail, with a flat face for your part.", "din rail clip 35mm mount electrical", "#4d7c0f",
		[]Param{num("w", "Width", 12, 4, 100, 1, "mm"), num("plate", "Mounting face height", 45, 36, 150, 1, "mm")},
		func(p Params) (*Mesh, error) {
			w, ph := p.F("w", 12), p.F("plate", 45)
			plate := boxAt(V(-w/2, -ph/2, 0), V(w/2, ph/2, 3))
			top := prismY(Loop{{0, 0}, {0, 2.5}, {-17.6, 2.5}, {-17.6, 5}, {-19.2, 5}, {-19.2, 0}}, w).Transformed(RotZ(90).Mul(RotX(0)))
			_ = top
			hookA := boxAt(V(-w/2, 16, -1.3), V(w/2, 19.5, 0.01))
			hookA = Union(hookA, boxAt(V(-w/2, 17.6, -2.8), V(w/2, 19.5, -1.29)))
			lipB := boxAt(V(-w/2, -19.5, -1.3), V(w/2, -16.5, 0.01))
			lipB = Union(lipB, prismY(Loop{{-17.6, -1.29}, {-19.5, -1.29}, {-19.5, -2.8}, {-18.5, -2.8}}, w).Transformed(RotZ(90)))
			return UnionAll([]*Mesh{plate, hookA, lipB}).Transformed(Translate(0, 0, 2.8)), nil
		})

	mech("extrusion", "Aluminium extrusion profile", "2020 / 3030 / 4040 T-slot extrusion profile (for mock-ups or printed frames).", "2020 3030 4040 extrusion profile t-slot v-slot aluminium frame", "#a1a1aa",
		[]Param{sel("profile", "Profile", "2020", "2020", "3030", "4040"), num("len", "Length", 100, 5, 2000, 1, "mm")},
		func(p Params) (*Mesh, error) {
			k := map[string]float64{"2020": 1, "3030": 1.5, "4040": 2}[p.S("profile", "2020")]
			if k == 0 {
				k = 1
			}
			s := 20 * k
			l := p.F("len", 100)
			m := boxAt(V(-s/2, -s/2, 0), V(s/2, s/2, l))
			slot := func(rot float64) *Mesh {
				pr := Loop{{-3 * k, s/2 + 1}, {-3 * k, s/2 - 1.8*k}, {-5.5 * k, s/2 - 1.8*k}, {-3 * k, s/2 - 6*k}, {3 * k, s/2 - 6*k}, {5.5 * k, s/2 - 1.8*k}, {3 * k, s/2 - 1.8*k}, {3 * k, s/2 + 1}}
				return extrudeAt(Shape{Outer: pr}, -1, l+1).Transformed(RotZ(rot))
			}
			cut := UnionAll([]*Mesh{slot(0), slot(90), slot(180), slot(270), cylAt(0, 0, 4.2*k, -1, l+1, 24)})
			return Subtract(m, cut), nil
		})

	mech("handle", "D-handle / pull", "Drawer or lid pull handle with two screw bosses.", "handle pull drawer door grip d handle", "#57534e",
		[]Param{num("span", "Screw spacing", 64, 20, 400, 1, "mm"), num("h", "Height", 25, 10, 100, 1, "mm"), num("d", "Grip thickness", 9, 4, 30, 0.5, "mm"), num("hole", "Screw hole Ø", 3.2, 1.5, 8, 0.1, "mm")},
		func(p Params) (*Mesh, error) {
			sp, h, d := p.F("span", 64), p.F("h", 25), p.F("d", 9)
			r := math.Min(h-d/2, sp/2) * 0.8
			pts := []Vec3{V(-sp/2, 0, d*0.3), V(-sp/2, 0, h-d/2-r)}
			pts = append(pts, arcPts(-sp/2+r, h-d/2-r, r, 180, 90, 10, false, 0)[1:]...)
			pts = append(pts, arcPts(sp/2-r, h-d/2-r, r, 90, 0, 10, false, 0)...)
			pts = append(pts, V(sp/2, 0, d*0.3))
			m := tubePath(pts, d, 20)
			m = Union(m, cylAt(-sp/2, 0, d*1.3, 0, d*0.6, 24))
			m = Union(m, cylAt(sp/2, 0, d*1.3, 0, d*0.6, 24))
			return holesThrough(m, [][2]float64{{-sp / 2, 0}, {sp / 2, 0}}, p.F("hole", 3.2), -1, h*0.5, 16), nil
		})

	mech("starknob", "Star knob", "Star-shaped clamping knob with a captive hex nut or bolt head pocket.", "star knob clamping lobe knob wing", "#b91c1c",
		[]Param{num("d", "Diameter", 35, 10, 200, 1, "mm"), num("lobes", "Lobes", 5, 3, 12, 1, ""), num("h", "Height", 14, 4, 80, 1, "mm"), sel("size", "Nut size", "M6", metricList()...)},
		func(p Params) (*Mesh, error) {
			d, n, h := p.F("d", 35), p.I("lobes", 5), p.F("h", 14)
			var l Loop
			segs := n * 24
			for i := 0; i < segs; i++ {
				a := 2 * math.Pi * float64(i) / float64(segs)
				r := d/2*0.72 + d/2*0.28*math.Pow(math.Abs(math.Cos(float64(n)*a/2)), 0.6)
				l = append(l, P2{r * math.Cos(a), r * math.Sin(a)})
			}
			body := Extrude(Shape{Outer: l}, h, 0, 1, 1)
			cap := RevolveProfile([]P2{{0, -0.01}, {d, -0.01}, {d, h * 0.6}, {d * 0.25, h + 0.01}, {0, h + 0.01}}, p.Segs(d))
			body = Intersect(body, cap)
			m := getMetric(p)
			body = Subtract(body, hexPrism(m.NutAF+0.4, -1, m.NutH+0.5))
			return Subtract(body, cylAt(0, 0, m.Normal+0.2, -1, h*0.85, p.Segs(m.D))), nil
		})

	mech("strapslot", "Strap loop / buckle", "Slot loop for straps and webbing, with an optional tension bar.", "strap loop buckle webbing velcro slot bag", "#9a3412",
		[]Param{num("strap", "Strap width", 25, 5, 100, 1, "mm"), num("t", "Thickness", 3, 1, 15, 0.5, "mm"), boolp("bar", "Centre bar (slide adjuster)", true)},
		func(p Params) (*Mesh, error) {
			s, t := p.F("strap", 25)+1, p.F("t", 3)
			out := RoundedRect(s+2*4, 14, 3, 6)
			h1 := moveLoop(RoundedRect(s, 3.5, 1, 4), 0, -3)
			h2 := moveLoop(RoundedRect(s, 3.5, 1, 4), 0, 3)
			holes := []Loop{h1, h2}
			if !p.B("bar", true) {
				holes = []Loop{RoundedRect(s, 9.5, 1, 4)}
			}
			return Extrude(Shape{Outer: out, Holes: holes}, t, 0, 1, 1), nil
		})

	mech("peghook", "Pegboard hook", "Hook that clips into standard pegboard holes (25.4 mm / 1\" grid).", "pegboard hook peg board tool holder", "#b45309",
		[]Param{num("reach", "Hook reach", 50, 10, 300, 1, "mm"), num("d", "Wire Ø", 5, 3, 7, 0.5, "mm")},
		func(p Params) (*Mesh, error) {
			r, d := p.F("reach", 50), p.F("d", 5)
			// back plate against the board, a bent top pin and a straight bottom pin into the holes
			back := boxAt(V(0, -d*0.8, -d), V(3, d*0.8, 25.4+d))
			top := tubePath([]Vec3{V(1.5, 0, 25.4), V(-4, 0, 25.4), V(-6, 0, 26.5), V(-6.5, 0, 31)}, d*0.9, 16)
			bottom := tubePath([]Vec3{V(1.5, 0, 0), V(-6, 0, 0)}, d*0.9, 16)
			arm := []Vec3{V(1.5, 0, 0), V(r, 0, 0)}
			arm = append(arm, arcPts(r, d*1.5, d*1.5, -90, 30, 10, false, 0)[1:]...)
			m := UnionAll([]*Mesh{back, top, bottom, tubePath(arm, d, 16)})
			return m.Transformed(Translate(0, 0, d)), nil
		})

	mech("shook", "S-hook", "S-shaped hook for hanging things from rails and rods.", "s hook hanger rail", "#78350f",
		[]Param{num("size", "Hook size", 30, 8, 200, 1, "mm"), num("d", "Thickness", 4, 1.5, 20, 0.5, "mm")},
		func(p Params) (*Mesh, error) {
			s, d := p.F("size", 30), p.F("d", 4)
			r := s / 4
			pts := arcPts(0, r, r, -30, 180, 18, false, 0)
			pts = append(pts, arcPts(-2*r, r, r, 0, -210, 18, false, 0)[1:]...)
			// flatten into the XY plane so it prints lying down
			for i := range pts {
				pts[i] = V(pts[i].X, pts[i].Z, d/2)
			}
			return tubePath(pts, d, 14), nil
		})

	mech("flexure", "Flexure hinge", "Compliant hinge: two blocks joined by a thin flexible web — no assembly, no wear.", "flexure compliant hinge flexible joint living hinge", "#0f766e",
		[]Param{num("w", "Width", 20, 3, 200, 1, "mm"), num("block", "Block length", 15, 3, 200, 1, "mm"), num("t", "Thickness", 6, 1, 50, 0.5, "mm"), num("web", "Web thickness", 0.8, 0.3, 5, 0.1, "mm"), num("gap", "Web length", 6, 1, 50, 0.5, "mm")},
		func(p Params) (*Mesh, error) {
			w, b, t, web, g := p.F("w", 20), p.F("block", 15), p.F("t", 6), p.F("web", 0.8), p.F("gap", 6)
			a := boxAt(V(-g/2-b, -w/2, 0), V(-g/2, w/2, t))
			c := boxAt(V(g/2, -w/2, 0), V(g/2+b, w/2, t))
			webm := boxAt(V(-g/2-0.5, -w/2, 0), V(g/2+0.5, w/2, web))
			return UnionAll([]*Mesh{a, c, webm}), nil
		})

	mech("spiralspring", "Flat spiral spring", "Clock-style flat spiral (torsion) spring printed in one piece.", "spiral spring flat torsion clock spring compliant", "#6d28d9",
		[]Param{num("d", "Outer Ø", 40, 10, 300, 1, "mm"), num("turns", "Turns", 4, 1, 20, 0.5, ""), num("strip", "Strip width", 1.2, 0.4, 10, 0.1, "mm"), num("h", "Height", 5, 1, 50, 0.5, "mm"), num("hub", "Hub Ø", 10, 2, 100, 0.5, "mm")},
		func(p Params) (*Mesh, error) {
			d, turns, sw, h, hub := p.F("d", 40), p.F("turns", 4), p.F("strip", 1.2), p.F("h", 5), p.F("hub", 10)
			r0, r1 := hub/2-sw/2, d/2-sw
			n := int(turns * 60)
			var pts []Vec3
			for i := 0; i <= n; i++ {
				t := float64(i) / float64(n)
				a := 2 * math.Pi * turns * t
				r := r0 + (r1-r0)*t
				pts = append(pts, V(r*math.Cos(a), r*math.Sin(a), 0))
			}
			// rectangular strip: sweep a flat rectangle along the spiral
			var frames []Frame
			for i := range pts {
				var tg Vec3
				if i+1 < len(pts) {
					tg = pts[i+1].Sub(pts[i])
				} else {
					tg = pts[i].Sub(pts[i-1])
				}
				u := V(0, 0, 1).Cross(tg).Norm()
				frames = append(frames, Frame{pts[i], u, V(0, 0, 1)})
			}
			sp := Sweep(Shape{Outer: Loop{{-sw / 2, 0}, {sw / 2, 0}, {sw / 2, h}, {-sw / 2, h}}}, frames, false)
			m := Union(sp, ring(hub, hub*0.4, 0, h, p.Segs(hub)))
			m = Union(m, ring(d, d-2*sw, 0, h, p.Segs(d)))
			return m, nil
		})

	mech("hosebarb", "Hose barb", "Barbed hose connector — straight, with optional panel thread or flange.", "hose barb fitting tube connector tubing", "#0369a1",
		[]Param{num("hose", "Hose inner Ø", 6, 2, 50, 0.5, "mm"), num("len", "Barb length", 15, 5, 100, 1, "mm"), sel("end", "Other end", "Barb", "Barb", "Flange", "Thread (M10/M12…)"), num("wall", "Wall", 1.6, 1.2, 5, 0.1, "mm")},
		func(p Params) (*Mesh, error) {
			hd, l, w := p.F("hose", 6), p.F("len", 15), p.F("wall", 1.6)
			barb := func() *Mesh {
				pts := []P2{{0, 0}, {hd * 0.45, 0}}
				nb := int(math.Max(2, l/(hd*0.6)))
				for i := 0; i < nb; i++ {
					z := l * float64(i) / float64(nb)
					pts = append(pts, P2{hd * 0.45, z}, P2{hd * 0.58, z + l/float64(nb)*0.7})
				}
				pts = append(pts, P2{hd * 0.45, l}, P2{0, l})
				return RevolveProfile(pts, p.Segs(hd*1.2))
			}
			top := barb().Transformed(Translate(0, 0, 6))
			mid := hexPrism(hd*1.4, 0, 6.01)
			var bottom *Mesh
			switch p.S("end", "Barb") {
			case "Flange":
				bottom = cylAt(0, 0, hd*3, 0, 2, p.Segs(hd*3))
			case "Thread (M10/M12…)":
				td := math.Max(hd+2*w+2, 8)
				bottom = Thread(ThreadSpec{Major: td, Pitch: StandardPitch("Metric fine", "ISO metric 60°", td), Profile: "ISO metric 60°", Starts: 1, Chamfer: true}, -8, 0.01, p.Q())
			default:
				bottom = barb().Transformed(RotX(180))
			}
			m := UnionAll([]*Mesh{top, mid, bottom})
			lo, hi := m.Bounds()
			return Subtract(m, cylAt(0, 0, math.Max(0.8, hd*0.9-2*w), lo.Z-1, hi.Z+1, p.Segs(hd))), nil
		})

	mech("funnel", "Funnel", "Funnel with a spout — sized for filling bottles or resin tanks.", "funnel cone spout pour", "#0284c7",
		[]Param{num("d", "Top Ø", 60, 10, 400, 1, "mm"), num("spout", "Spout Ø", 10, 2, 100, 0.5, "mm"), num("h", "Cone height", 40, 5, 400, 1, "mm"), num("sl", "Spout length", 20, 0, 200, 1, "mm"), num("wall", "Wall", 1.6, 0.6, 5, 0.1, "mm")},
		func(p Params) (*Mesh, error) {
			d, sd, h, sl, w := p.F("d", 60), p.F("spout", 10), p.F("h", 40), p.F("sl", 20), p.F("wall", 1.6)
			pr := Loop{{sd/2 - w, 0}, {sd / 2, 0}, {sd / 2, sl}, {d / 2, sl + h}, {d / 2, sl + h + 3}, {d/2 - w, sl + h + 3}, {d/2 - w, sl + h + w*0.3}, {sd/2 - w, sl + w}}
			return Revolve(Shape{Outer: pr}, 360, p.Segs(d)), nil
		})

	mech("dovetailrail", "Dovetail rail", "Sliding dovetail rail — fits the Dovetail slot hole with the same width.", "dovetail rail slide joint key", "#7e22ce",
		[]Param{num("len", "Length", 40, 5, 1000, 1, "mm"), num("w", "Narrow width", 8, 2, 100, 0.5, "mm"), num("h", "Height", 4, 1, 50, 0.5, "mm"), num("angle", "Angle", 60, 45, 80, 1, "°")},
		func(p Params) (*Mesh, error) {
			return dovetailBody(p.F("len", 40), p.F("w", 8), p.F("h", 4), p.F("angle", 60), false), nil
		})

	mech("jar", "Screw-top jar", "Container with a threaded neck — pair with the Jar lid of the same diameter.", "jar container screw top bottle pot box threaded", "#0891b2",
		[]Param{num("d", "Neck Ø", 50, 15, 300, 1, "mm"), num("h", "Height", 50, 10, 500, 1, "mm"), num("wall", "Wall", 2, 0.8, 10, 0.2, "mm"), num("pitch", "Thread pitch", 4, 1.5, 10, 0.5, "mm")},
		func(p Params) (*Mesh, error) {
			d, h, w, pt := p.F("d", 50), p.F("h", 50), p.F("wall", 2), p.F("pitch", 4)
			neckH := pt * 2.5
			body := cylAt(0, 0, d+2*pt*0.5, 0, h-neckH, p.Segs(d))
			neck := Thread(ThreadSpec{Major: d - 0.175, Pitch: pt, Profile: "Trapezoidal 30°", Starts: 2, Chamfer: true, Axial: -0.175}, h-neckH-0.01, h, p.Q())
			m := Union(body, neck)
			inner := cylAt(0, 0, d-2*pt*0.5-2*w, w, h+1, p.Segs(d))
			res := Subtract(m, inner)
			res.Info = "Neck thread: trapezoidal, 2 starts, pitch " + ff(pt) + " mm, nominal Ø " + ff(d) + " mm — matches a Jar lid with the same settings."
			return res, nil
		})

	mech("jarlid", "Jar lid", "Threaded screw-on lid for the Screw-top jar (same diameter and pitch).", "jar lid cap screw cap bottle cap threaded", "#155e75",
		[]Param{num("d", "Neck Ø", 50, 15, 300, 1, "mm"), num("wall", "Wall", 2, 0.8, 10, 0.2, "mm"), num("pitch", "Thread pitch", 4, 1.5, 10, 0.5, "mm"), num("grip", "Grip ridges", 40, 0, 200, 1, "")},
		func(p Params) (*Mesh, error) {
			d, w, pt := p.F("d", 50), p.F("wall", 2), p.F("pitch", 4)
			depth := pt*2.5 + 0.6
			od := d + 2*w + pt
			var outer Loop
			if g := p.I("grip", 40); g > 0 {
				outer = knurlLoop(od, 0.8, g, p.Segs(od))
			} else {
				outer = Circle(od/2, p.Segs(od))
			}
			cap := Extrude(Shape{Outer: outer}, depth+w, 0, 1, 1)
			cut := Thread(ThreadSpec{Major: d + 0.175, Pitch: pt, Profile: "Trapezoidal 30°", Starts: 2, Axial: 0.175}, w, depth+w+0.5, p.Q())
			// lead-in chamfer removes the paper-thin start of the thread ridge at the opening
			ld0 := d + 0.175 - pt - 0.6 // starts inside the ridge crest so no flat step is left
			lh := (d + 0.6 - ld0) / 2   // 45° lead-in
			lead := coneAt(ld0, d+0.6, depth+w-lh, depth+w+0.01, p.Segs(d))
			cut = Union(cut, Union(lead, cylAt(0, 0, d+0.6, depth+w, depth+w+1, p.Segs(d))))
			return Subtract(cap, cut), nil
		})

	mech("propeller", "Propeller", "Twisted-blade propeller for boats, fans and drones (prints upright).", "propeller prop fan blade boat drone", "#dc2626",
		[]Param{num("d", "Diameter", 80, 10, 500, 1, "mm"), num("blades", "Blades", 3, 2, 8, 1, ""), num("pitch", "Blade twist", 35, 0, 80, 1, "°"), num("h", "Hub height", 12, 3, 80, 1, "mm"), bore(3)},
		func(p Params) (*Mesh, error) {
			d, n, tw, h := p.F("d", 80), p.I("blades", 3), p.F("pitch", 35), p.F("h", 12)
			hub := d * 0.15
			var ms []*Mesh
			ms = append(ms, cylAt(0, 0, hub, 0, h, p.Segs(hub)))
			for k := 0; k < n; k++ {
				var frames []Frame
				for i := 0; i <= 10; i++ {
					t := float64(i) / 10
					r := hub*0.4 + (d/2-hub*0.4)*t
					a := (tw + 15*(1-t)) * math.Pi / 180
					chord := d * 0.12 * (1 - 0.6*t*t)
					_ = chord
					frames = append(frames, Frame{V(r, 0, h/2), V(0, math.Cos(a)*(1-0.6*t*t), math.Sin(a)*(1-0.6*t*t)), V(0, -math.Sin(a), math.Cos(a))})
				}
				blade := Sweep(Shape{Outer: Loop{{-d * 0.06, -0.9}, {d * 0.06, -0.9}, {d * 0.06, 0.9}, {-d * 0.06, 0.9}}}, frames, false)
				ms = append(ms, blade.Transformed(RotZ(360*float64(k)/float64(n))))
			}
			return boreFor(UnionAll(ms), p.F("bore", 3), -1, h+1, p.Segs(8)), nil
		})

	mech("impeller", "Fan impeller", "Centrifugal (blower) impeller with curved blades on a base disc.", "impeller blower centrifugal fan pump", "#ea580c",
		[]Param{num("d", "Diameter", 50, 10, 400, 1, "mm"), num("blades", "Blades", 8, 3, 32, 1, ""), num("h", "Blade height", 10, 2, 100, 1, "mm"), bore(3)},
		func(p Params) (*Mesh, error) {
			d, n, h := p.F("d", 50), p.I("blades", 8), p.F("h", 10)
			base := cylAt(0, 0, d, 0, 1.6, p.Segs(d))
			var ms []*Mesh
			ms = append(ms, base, cylAt(0, 0, d*0.22, 0, h+1.6, p.Segs(d)))
			for k := 0; k < n; k++ {
				var l Loop
				var back Loop
				for i := 0; i <= 10; i++ {
					t := float64(i) / 10
					r := d*0.12 + (d/2-d*0.12)*t
					a := t * 0.6
					l = append(l, P2{r * math.Cos(a), r * math.Sin(a)})
					ab := a + 1.8/r // 1.8 mm thick blades all the way along
					back = append(back, P2{r * math.Cos(ab), r * math.Sin(ab)})
				}
				for i := len(back) - 1; i >= 0; i-- {
					l = append(l, back[i])
				}
				ms = append(ms, extrudeAt(Shape{Outer: l}, 1.59, h+1.6).Transformed(RotZ(360*float64(k)/float64(n))))
			}
			m := UnionAll(ms)
			return boreFor(m, p.F("bore", 3), -1, h+3, p.Segs(8)), nil
		})

	mech("chainlink", "Chain link", "Oval chain link — print several interlocked with Pattern, or as a keyring.", "chain link oval ring keyring", "#52525b",
		[]Param{num("l", "Length", 24, 6, 200, 1, "mm"), num("w", "Width", 14, 4, 150, 1, "mm"), num("d", "Thickness", 3.5, 1, 20, 0.5, "mm")},
		func(p Params) (*Mesh, error) {
			l, w, d := p.F("l", 24), p.F("w", 14), p.F("d", 3.5)
			r := w/2 - d/2
			s := l/2 - w/2
			var pts []Vec3
			pts = append(pts, arcPts(s, 0, r, -90, 90, 14, true, d/2)...)
			pts = append(pts, arcPts(-s, 0, r, 90, 270, 14, true, d/2)...)
			pts = append(pts, pts[0])
			return tubePath(pts, d, 14), nil
		})

	mech("geneva", "Geneva drive wheel", "Slotted Geneva wheel for intermittent (step-by-step) motion.", "geneva drive maltese cross intermittent indexing", "#4338ca",
		[]Param{num("d", "Wheel Ø", 50, 15, 300, 1, "mm"), num("slots", "Slots", 4, 3, 12, 1, ""), num("h", "Thickness", 5, 1, 50, 0.5, "mm"), num("pin", "Drive pin Ø", 4, 1, 20, 0.5, "mm"), bore(5)},
		func(p Params) (*Mesh, error) {
			d, n, h, pin := p.F("d", 50), p.I("slots", 4), p.F("h", 5), p.F("pin", 4)+0.4
			R := d / 2
			c := R / math.Cos(math.Pi/float64(n)) // distance to driver centre
			body := cylAt(0, 0, d, 0, h, p.Segs(d))
			var cuts []*Mesh
			for k := 0; k < n; k++ {
				a := 360 * float64(k) / float64(n)
				slotLen := R + c - c*math.Cos(math.Pi/float64(n)) - R*0.0
				_ = slotLen
				cuts = append(cuts, boxAt(V(R*0.35, -pin/2, -1), V(R+1, pin/2, h+1)).Transformed(RotZ(a)))
				// clearance arcs for the driver's locking disc
				b := a + 180/float64(n)
				cuts = append(cuts, cylAt(c*math.Cos(b*math.Pi/180), c*math.Sin(b*math.Pi/180), (c-R*0.75)*2, -1, h+1, p.Segs(d)))
			}
			m := Subtract(body, UnionAll(cuts))
			return boreFor(m, p.F("bore", 5), -1, h+1, p.Segs(8)), nil
		})

	mech("pipetee", "Tube connector", "Joint for building frames from tubes or dowels: elbow, tee, cross or 3-way corner.", "tube connector pipe joint tee elbow cross corner frame dowel", "#0d9488",
		[]Param{sel("kind", "Shape", "Tee", "Elbow", "Tee", "Cross", "3-way corner", "Straight"), num("tube", "Tube Ø", 16, 3, 100, 0.5, "mm"), num("wall", "Wall", 2.5, 1, 15, 0.5, "mm"), num("depth", "Socket depth", 15, 5, 100, 1, "mm"), num("tol", "Fit clearance", 0.3, 0, 2, 0.05, "mm"),
			boolp("hollow", "Hollow (flow goes through)", true)},
		func(p Params) (*Mesh, error) {
			td, w, dp := p.F("tube", 16)+p.F("tol", 0.3), p.F("wall", 2.5), p.F("depth", 15)
			od := td + 2*w
			arm := func(dir Vec3) *Mesh {
				c := ring(od, td, od*0.3, od*0.5+dp, p.Segs(od))
				c = Union(c, cylAt(0, 0, od, 0, od*0.31, p.Segs(od)))
				R := rotBetweenG(V(0, 0, 1), dir)
				return c.Transformed(R)
			}
			var dirs []Vec3
			switch p.S("kind", "Tee") {
			case "Elbow":
				dirs = []Vec3{V(1, 0, 0), V(0, 1, 0)}
			case "Cross":
				dirs = []Vec3{V(1, 0, 0), V(-1, 0, 0), V(0, 1, 0), V(0, -1, 0)}
			case "3-way corner":
				dirs = []Vec3{V(1, 0, 0), V(0, 1, 0), V(0, 0, 1)}
			case "Straight":
				dirs = []Vec3{V(0, 0, 1), V(0, 0, -1)}
			default:
				dirs = []Vec3{V(1, 0, 0), V(-1, 0, 0), V(0, 1, 0)}
			}
			var ms []*Mesh
			// spherical-ish core
			ms = append(ms, boxAt(V(-od/2, -od/2, -od/2), V(od/2, od/2, od/2)))
			for _, d := range dirs {
				ms = append(ms, arm(d))
			}
			m := UnionAll(ms)
			if p.B("hollow", true) {
				// open the core so air / water / cables pass between the tubes
				var bores []*Mesh
				bd := math.Max(1, td-2*w)
				for _, d := range dirs {
					bores = append(bores, cylAt(0, 0, bd, -0.01, od, p.Segs(od)).Transformed(rotBetweenG(V(0, 0, 1), d)))
				}
				bores = append(bores, sphereD(bd, p.Segs(od)))
				m = Subtract(m, UnionAll(bores))
			}
			lo, _ := m.Bounds()
			return m.Transformed(Translate(0, 0, -lo.Z)), nil
		})
}

// arcLoop: points on a circle from a0 to a1 (degrees), CCW
func arcLoop(cx, cy, r, a0, a1 float64) Loop {
	var l Loop
	n := int(math.Max(6, math.Abs(a1-a0)/6))
	for i := 0; i <= n; i++ {
		a := (a0 + (a1-a0)*float64(i)/float64(n)) * math.Pi / 180
		l = append(l, P2{cx + r*math.Cos(a), cy + r*math.Sin(a)})
	}
	return l
}

func arcLoopRev(cx, cy, r, a0, a1 float64) Loop { return arcLoop(cx, cy, r, a0, a1).Reversed() }

// rotBetweenG is the rotation turning a onto b (Go side).
func rotBetweenG(a, b Vec3) Mat4 {
	a, b = a.Norm(), b.Norm()
	c := a.Dot(b)
	if c > 0.99999 {
		return Identity()
	}
	if c < -0.99999 {
		return RotX(180)
	}
	v := a.Cross(b)
	k := 1 / (1 + c)
	return Mat4{
		1 - (v.Y*v.Y+v.Z*v.Z)*k, -v.Z + v.X*v.Y*k, v.Y + v.X*v.Z*k, 0,
		v.Z + v.X*v.Y*k, 1 - (v.X*v.X+v.Z*v.Z)*k, -v.X + v.Y*v.Z*k, 0,
		-v.Y + v.X*v.Z*k, v.X + v.Y*v.Z*k, 1 - (v.X*v.X+v.Y*v.Y)*k, 0,
		0, 0, 0, 1,
	}
}
