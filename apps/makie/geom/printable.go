package geom

import (
	"fmt"
	"math"
)

// ===================== Hardware designed to be 3D printed =====================
// Big, coarse, rounded threads and springs whose shapes print well in PLA/PETG.

// coarse printable pitch for a diameter
func printPitch(d float64) float64 { return math.Max(1.5, math.Min(6, math.Round(d/4.5*2)/2)) }

func printHead(kind string, af, h float64, segs int) *Mesh {
	switch kind {
	case "Knurled":
		return Extrude(Shape{Outer: knurlLoop(af*1.25, 1.2, int(af*1.25*math.Pi/3), segs*2)}, h, 0, 1, 1)
	case "Wing":
		hub := cylAt(0, 0, af*0.9, 0, h, segs)
		wing := Extrude(Shape{Outer: stadium(af*2.6, h*0.9, 16)}, 3.2, 0, 1, 1).Transformed(RotX(90)).Transformed(Translate(0, 1.6, h*0.45))
		return Union(hub, Union(wing, boxAt(V(-af*1.3+h*0.45, -1.6, 0), V(af*1.3-h*0.45, 1.6, h*0.45))))
	case "Round (slotted)":
		return coneAt(af*1.2, af*1.0, 0, h, segs)
	default: // hex, with chamfered top edge
		return Intersect(Extrude(Shape{Outer: NGon(af/math.Cos(math.Pi/6)/2, 6, 30)}, h, 0, 1, 1), coneAt(af/math.Cos(math.Pi/6)+h*1.4, af*0.92, 0, h+0.01, segs))
	}
}

func init() {
	hw := func(typ, name, desc, tags string, params []Param, gen func(p Params) (*Mesh, error)) {
		register(&ShapeDef{Type: typ, Name: name, Category: "Hardware", Desc: desc, Tags: tags + " printable printed 3d print", Color: "#f59e0b", Params: params, Gen: gen})
	}
	pthread := func(p Params) (d, pitch, clr float64) {
		d = p.F("d", 12)
		pitch = p.F("pitch", 0)
		clr = p.F("clr", 0.3)
		if pitch <= 0 {
			pitch = printPitch(d)
			// square teeth are half the pitch wide: keep them printable
			if p.S("profile", "") == "Square" {
				pitch = math.Max(pitch, 2*(MinWall+clr)+0.2)
			}
		}
		return d, pitch, clr
	}
	tparams := []Param{
		num("d", "Diameter", 12, 6, 80, 0.5, "mm"),
		num("pitch", "Pitch", 0, 0, 10, 0.25, "mm").help("0 = automatic coarse pitch that prints cleanly"),
		sel("profile", "Thread shape", "Trapezoidal 30°", "Trapezoidal 30°", "Square", "Buttress"),
		num("clr", "Clearance", 0.3, 0, 1, 0.05, "mm").help("Gap between bolt and nut. 0.3 suits most printers; raise it if they're too tight."),
	}
	heads := sel("head", "Head", "Hex", "Hex", "Knurled", "Wing", "Round (slotted)")

	hw("pbolt", "Printable bolt", "Bolt with a big coarse thread that prints cleanly — pair it with a Printable nut of the same diameter.", "bolt screw coarse thread",
		append(append([]Param{}, tparams...), num("len", "Thread length", 25, 5, 300, 1, "mm"), heads),
		func(p Params) (*Mesh, error) {
			d, pitch, clr := pthread(p)
			L := p.F("len", 25)
			af := d * 1.6
			hh := math.Max(4, d*0.6)
			head := printHead(p.S("head", "Hex"), af, hh, p.Segs(af))
			thr := Thread(ThreadSpec{Major: d - clr, Pitch: pitch, Profile: p.S("profile", "Trapezoidal 30°"), Starts: 1, Chamfer: true, Axial: -clr / 2}, hh-0.01, hh+L, p.Q())
			m := Union(head, thr)
			if p.S("head", "Hex") == "Round (slotted)" {
				m = Subtract(m, boxAt(V(-af, -0.9-d*0.06, -1), V(af, 0.9+d*0.06, hh*0.45)))
			}
			m.Info = fmt.Sprintf("%.1f mm × %.2g mm pitch %s thread. Print standing on its head.", d, pitch, p.S("profile", "Trapezoidal 30°"))
			return m, nil
		})
	hw("pnut", "Printable nut", "Nut for the Printable bolt (same diameter, pitch and thread shape).", "nut coarse thread",
		append(append([]Param{}, tparams...), sel("head", "Shape", "Hex", "Hex", "Knurled", "Wing")),
		func(p Params) (*Mesh, error) {
			d, pitch, clr := pthread(p)
			af := d * 1.6
			h := math.Max(5, d*0.8)
			body := printHead(p.S("head", "Hex"), af, h, p.Segs(af))
			cut := Thread(ThreadSpec{Major: d + clr, Pitch: pitch, Profile: p.S("profile", "Trapezoidal 30°"), Starts: 1, Axial: clr / 2}, -1, h+1, p.Q())
			minor := d + clr - pitch*0.9
			cs := (d + clr + 0.8 - minor) / 2
			cut = UnionAll([]*Mesh{cut, coneAt(d+clr+0.8, minor, -0.01, cs, p.Segs(d)), coneAt(minor, d+clr+0.8, h-cs, h+0.01, p.Segs(d))})
			if p.S("head", "Hex") == "Wing" {
				body = Union(body, cylAt(0, 0, d+2*screwWall(d), 0, h, p.Segs(d)))
			}
			return Subtract(body, cut), nil
		})
	hw("pwasher", "Printable washer", "Thick washer that's strong when printed (spreads the load of a printed bolt).", "washer spacer",
		[]Param{num("d", "Bolt Ø", 12, 2, 80, 0.5, "mm"), num("t", "Thickness", 2.4, 1.2, 10, 0.2, "mm")},
		func(p Params) (*Mesh, error) {
			d := p.F("d", 12)
			return ring(d*2.1+2, d+0.5, 0, p.F("t", 2.4), p.Segs(d*2)), nil
		})

	// ---------- printable springs ----------
	spring := func(typ, name, desc string, params []Param, gen func(p Params) (*Mesh, error)) {
		register(&ShapeDef{Type: typ, Name: name, Category: "Hardware", Desc: desc, Tags: "spring printable flex torsion compression", Color: "#8b5cf6", Params: params, Gen: gen})
	}
	spring("ptorsion", "Torsion spring (flat)", "Flat spiral spring with a hub and an arm — prints flat, twists back. Great for lids, triggers and clips. PETG springs best.",
		[]Param{num("d", "Outer Ø", 40, 15, 200, 1, "mm"), num("turns", "Turns", 2.5, 1, 8, 0.25, ""), num("strip", "Strip width", 1.6, 1.2, 6, 0.1, "mm"), num("h", "Thickness", 6, 2, 40, 0.5, "mm"), num("shaft", "Shaft Ø", 5, 1.75, 40, 0.25, "mm"), num("arm", "Arm length", 20, 0, 200, 1, "mm")},
		func(p Params) (*Mesh, error) {
			d, turns, sw, h, shaft := p.F("d", 40), p.F("turns", 2.5), math.Max(1.2, p.F("strip", 1.6)), p.F("h", 6), p.F("shaft", 5)
			hub := shaft + 2*screwWall(shaft)
			r0, r1 := hub/2-sw/2, d/2-sw
			if r1 < r0+2*sw*turns {
				return nil, fmt.Errorf("not enough room for %g turns — make it bigger or use fewer turns", turns)
			}
			n := int(turns * 72)
			var frames []Frame
			var last Vec3
			for i := 0; i <= n; i++ {
				t := float64(i) / float64(n)
				a := 2 * math.Pi * turns * t
				r := r0 + (r1-r0)*t
				pt := V(r*math.Cos(a), r*math.Sin(a), 0)
				tg := V(-math.Sin(a)*r+(r1-r0)/(2*math.Pi*turns)*math.Cos(a), math.Cos(a)*r+(r1-r0)/(2*math.Pi*turns)*math.Sin(a), 0)
				frames = append(frames, Frame{pt, V(0, 0, 1).Cross(tg).Norm(), V(0, 0, 1)})
				last = pt
			}
			sp := Sweep(Shape{Outer: Loop{{-sw / 2, 0}, {sw / 2, 0}, {sw / 2, h}, {-sw / 2, h}}}, frames, false)
			hubM := Subtract(cylAt(0, 0, hub, 0, h, p.Segs(hub)), Subtract(cylAt(0, 0, shaft, -1, h+1, p.Segs(shaft)), boxAt(V(shaft/2-shaft*0.15, -shaft, -2), V(shaft, shaft, h+2))))
			ea := 2 * math.Pi * turns
			tip := cylAt(last.X, last.Y, sw*3, 0, h, 16)
			m := UnionAll([]*Mesh{sp, hubM, tip})
			if arm := p.F("arm", 20); arm > 0 {
				dir := V(-math.Sin(ea), math.Cos(ea), 0)
				out := V(math.Cos(ea), math.Sin(ea), 0)
				a0 := last.Add(out.Mul(-sw * 0.5))
				a1 := a0.Add(dir.Mul(arm)).Add(out.Mul(sw * 2))
				w := sw * 2.5
				l := Loop{{a0.X - out.X*w/2, a0.Y - out.Y*w/2}, {a1.X - out.X*w/2, a1.Y - out.Y*w/2}, {a1.X + out.X*w/2, a1.Y + out.Y*w/2}, {a0.X + out.X*w/2, a0.Y + out.Y*w/2}}
				m = Union(m, Union(extrudeAt(Shape{Outer: l}, 0, h), cylAt(a1.X, a1.Y, w*1.4, 0, h, 16)))
			}
			m.Info = "The hub fits a D-shaped shaft. Turn the arm to wind it up."
			return m, nil
		})
	spring("ptorsioncoil", "Torsion spring (coil)", "Coil torsion spring with two legs, made of square 'wire'. Print upright in PETG or TPU, gap ≥ 0.6 mm.",
		[]Param{num("d", "Coil outer Ø", 16, 6, 100, 0.5, "mm"), num("coils", "Coils", 4, 1, 20, 0.5, ""), num("wire", "Wire size", 1.6, 1.2, 6, 0.1, "mm"), num("gap", "Gap between coils", 0.8, 0.4, 5, 0.1, "mm"), num("leg", "Leg length", 15, 2, 200, 1, "mm")},
		func(p Params) (*Mesh, error) {
			d, coils, w, gap, leg := p.F("d", 16), p.F("coils", 4), math.Max(1.2, p.F("wire", 1.6)), p.F("gap", 0.8), p.F("leg", 15)
			r := d/2 - w/2
			pitch := w + gap
			n := int(coils * 60)
			var pts []Vec3
			// first leg: straight out tangentially
			pts = append(pts, V(r, -leg, w/2))
			for i := 0; i <= n; i++ {
				a := 2 * math.Pi * coils * float64(i) / float64(n)
				pts = append(pts, V(r*math.Cos(a), r*math.Sin(a), w/2+pitch*coils*float64(i)/float64(n)))
			}
			ea := 2 * math.Pi * coils
			e := pts[len(pts)-1]
			tg := V(-math.Sin(ea), math.Cos(ea), 0)
			pts = append(pts, e.Add(tg.Mul(leg)))
			var frames []Frame
			for i := range pts {
				var t Vec3
				switch {
				case i == 0:
					t = pts[1].Sub(pts[0])
				case i == len(pts)-1:
					t = pts[i].Sub(pts[i-1])
				default:
					t = pts[i+1].Sub(pts[i-1])
				}
				t = t.Norm()
				u := V(0, 0, 1).Cross(t).Norm()
				frames = append(frames, Frame{pts[i], u, t.Cross(u)})
			}
			m := Sweep(Shape{Outer: Loop{{-w / 2, -w / 2}, {w / 2, -w / 2}, {w / 2, w / 2}, {-w / 2, w / 2}}}, frames, false)
			m.Info = fmt.Sprintf("Fits on a %.1f mm pin.", d-2*w-0.5)
			return m, nil
		})
	spring("pzigzag", "Zig-zag spring", "Flat serpentine spring — stretches and squashes in the plane. Prints flat, no supports.",
		[]Param{num("len", "Length", 40, 10, 300, 1, "mm"), num("w", "Width", 12, 4, 100, 0.5, "mm"), num("n", "Folds", 6, 2, 40, 1, ""), num("strip", "Strip width", 1.2, 1.0, 5, 0.1, "mm"), num("h", "Thickness", 5, 1, 40, 0.5, "mm"), boolp("ends", "Mounting tabs", true)},
		func(p Params) (*Mesh, error) {
			L, W, n, sw, h := p.F("len", 40), p.F("w", 12), p.I("n", 6), math.Max(1.0, p.F("strip", 1.2)), p.F("h", 5)
			step := L / float64(n)
			if step < 2*sw+0.6 {
				return nil, fmt.Errorf("too many folds for that length — the strips would touch")
			}
			var ms []*Mesh
			for i := 0; i <= n; i++ {
				x := -L/2 + step*float64(i)
				ms = append(ms, boxAt(V(x-sw/2, -W/2+sw/2, 0), V(x+sw/2, W/2-sw/2, h)))
				if i < n {
					y := W/2 - sw
					if i%2 == 1 {
						y = -W / 2
					}
					ms = append(ms, boxAt(V(x-sw/2, y, 0), V(x+step+sw/2, y+sw, h)))
				}
			}
			if p.B("ends", true) {
				for _, s := range []float64{-1, 1} {
					x := s * (L/2 + 3)
					ms = append(ms, Subtract(boxAt(V(x-3.5, -W/2, 0), V(x+3.5, W/2, h)), cylAt(x, 0, 3.4, -1, h+1, 20)))
				}
			}
			return UnionAll(ms), nil
		})
	spring("pdiamond", "Diamond compression spring", "Stack of diamond cells that squashes like a coil spring — prints flat on its side without supports.",
		[]Param{num("len", "Length (free)", 30, 8, 300, 1, "mm"), num("w", "Width", 12, 5, 100, 0.5, "mm"), num("cells", "Cells", 4, 1, 30, 1, ""), num("strip", "Strip width", 1.2, 1.0, 4, 0.1, "mm"), num("h", "Depth", 8, 2, 60, 0.5, "mm")},
		func(p Params) (*Mesh, error) {
			L, W, n, sw, h := p.F("len", 30), p.F("w", 12), p.I("cells", 4), math.Max(1.0, p.F("strip", 1.2)), p.F("h", 8)
			cl := L / float64(n)
			var ms []*Mesh
			plate := 2.0
			ms = append(ms, boxAt(V(-W/2, 0, 0), V(W/2, plate, h)), boxAt(V(-W/2, L+plate, 0), V(W/2, L+2*plate, h)))
			for i := 0; i < n; i++ {
				y0 := plate + cl*float64(i)
				outer := Loop{{0, y0}, {W / 2, y0 + cl/2}, {0, y0 + cl}, {-W / 2, y0 + cl/2}}
				in := W/2 - sw*1.4
				ic := cl/2 - sw*1.4*cl/W
				if in < 0.5 || ic < 0.5 {
					return nil, fmt.Errorf("cells too small for that strip width")
				}
				inner := Loop{{0, y0 + cl/2 - ic}, {in, y0 + cl/2}, {0, y0 + cl/2 + ic}, {-in, y0 + cl/2}}
				ms = append(ms, extrudeAt(Shape{Outer: outer, Holes: []Loop{inner}}, 0, h))
				if i < n-1 {
					ms = append(ms, boxAt(V(-sw, y0+cl-0.5, 0), V(sw, y0+cl+0.5, h)))
				}
			}
			ms = append(ms, boxAt(V(-sw, plate-0.5, 0), V(sw, plate+0.5, h)), boxAt(V(-sw, L+plate-0.5, 0), V(sw, L+plate+0.5, h)))
			return UnionAll(ms), nil
		})
	spring("pleaf", "Leaf spring", "Curved strip spring (bow) with mounting holes — for latches, buttons and clips.",
		[]Param{num("len", "Length", 50, 10, 300, 1, "mm"), num("bow", "Bow height", 6, 0, 50, 0.5, "mm"), num("t", "Thickness", 1.6, 1.2, 6, 0.1, "mm"), num("w", "Width", 8, 3, 60, 0.5, "mm"), sel("screw", "Screw holes", "M3", "None", "M2", "M2.5", "M3", "M4")},
		func(p Params) (*Mesh, error) {
			L, bow, t, w := p.F("len", 50), p.F("bow", 6), math.Max(1.2, p.F("t", 1.6)), p.F("w", 8)
			var top, bot Loop
			n := 40
			for i := 0; i <= n; i++ {
				x := -L/2 + L*float64(i)/float64(n)
				z := bow * (1 - math.Pow(2*x/L, 2))
				top = append(top, P2{x, z + t})
				bot = append(bot, P2{x, z})
			}
			l := append(bot, top.Reversed()...)
			m := prismY(l, w)
			if sz := p.S("screw", "M3"); sz != "None" {
				mt := metrics[sz]
				hd := mt.Normal + 0.2
				e := holeEdge(hd)
				for _, s := range []float64{-1, 1} {
					x := s * (L/2 + e - 0.5)
					tab := Subtract(boxAt(V(x-e, -math.Max(w, 2*e)/2, 0), V(x+e, math.Max(w, 2*e)/2, t+0.6)), cylAt(x, 0, hd, -1, t+2, 20))
					m = Union(m, tab)
				}
			}
			return m, nil
		})
	hw("pbayonet", "Bayonet mount", "Push and twist to lock — a strong, quick printed connector (like a camera lens). Print both halves.", "bayonet twist lock quick release connector",
		[]Param{num("d", "Diameter", 30, 10, 200, 1, "mm"), num("lugs", "Lugs", 3, 2, 6, 1, ""), num("clr", "Clearance", 0.3, 0.1, 1, 0.05, "mm"), sel("part", "Show", "Both", "Both", "Male", "Female")},
		func(p Params) (*Mesh, error) {
			d, n, clr := p.F("d", 30), p.I("lugs", 3), p.F("clr", 0.3)
			segs := p.Segs(d * 1.5)
			lugH, lugW := 3.0, 360.0/float64(n)*0.32
			wall := math.Max(2.4, d*0.08)
			male := func() *Mesh {
				m := Union(cylAt(0, 0, d+2*wall+4, 0, 3, segs), ring(d, d-2*wall, 2.99, 3+lugH+3, segs))
				for i := 0; i < n; i++ {
					a0 := 360 * float64(i) / float64(n)
					lug := extrudeAt(Shape{Outer: append(arcLoop(0, 0, d/2+1.6, a0, a0+lugW), arcLoopRev(0, 0, d/2-0.5, a0, a0+lugW)...)}, 3+lugH*0.5, 3+lugH*0.5+lugH*0.8)
					m = Union(m, lug)
				}
				return m
			}
			female := func() *Mesh {
				od := d + 2*1.6 + 2*clr + 2*wall + 2
				body := Union(ring(od, d+2*clr, 0, lugH*2+4, segs), ring(od+4, d-2*wall, 0, 2.4, segs))
				var cuts []*Mesh
				for i := 0; i < n; i++ {
					a0 := 360 * float64(i) / float64(n)
					// entry slot down from the top, then a turn slot round the side
					entry := extrudeAt(Shape{Outer: append(arcLoop(0, 0, d/2+1.6+clr+0.2, a0-1, a0+lugW+1), arcLoopRev(0, 0, d/2, a0-1, a0+lugW+1)...)}, 2.4+lugH*0.2, lugH*2+5)
					turn := extrudeAt(Shape{Outer: append(arcLoop(0, 0, d/2+1.6+clr+0.2, a0-1, a0+lugW*2.2), arcLoopRev(0, 0, d/2, a0-1, a0+lugW*2.2)...)}, 2.4+lugH*0.2, 2.4+lugH*0.2+lugH*0.8+2*clr)
					cuts = append(cuts, entry, turn)
				}
				return Subtract(body, UnionAll(cuts))
			}
			switch p.S("part", "Both") {
			case "Male":
				return male(), nil
			case "Female":
				return female(), nil
			}
			return Union(male(), female().Transformed(Translate(d*1.6+12, 0, 0))), nil
		})
}
