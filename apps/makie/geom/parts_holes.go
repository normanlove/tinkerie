package geom

import "math"

// All holes have their origin on the surface they are placed on and cut downward (-Z),
// overshooting 0.6 mm above so cuts are always clean.
const over = 0.6

func init() {
	hole := func(typ, name, desc, tags string, fixed bool, params []Param, gen func(p Params) (*Mesh, error)) {
		register(&ShapeDef{Type: typ, Name: name, Category: "Holes", Hole: true, Fixed: fixed, Desc: desc, Tags: tags, Color: "#64748b", Params: params, Gen: gen})
	}
	depth := func(def float64) Param { return num("depth", "Depth", def, 0.5, 500, 0.5, "mm") }

	hole("keyhole", "Keyhole hanger", "Keyhole slot for hanging a part on a screw head.", "keyhole hang wall mount slot",
		true, []Param{num("head", "Screw head Ø", 8, 3, 30, 0.5, "mm"), num("shank", "Screw shank Ø", 4, 1.5, 15, 0.5, "mm"), num("slide", "Slide length", 10, 3, 60, 0.5, "mm"), depth(6)},
		func(p Params) (*Mesh, error) {
			hd, sd, sl, d := p.F("head", 8)+0.6, p.F("shank", 4)+0.5, p.F("slide", 10), p.F("depth", 6)
			big := cylAt(0, 0, hd, -d, over, p.Segs(hd))
			slot := extrudeAt(Shape{Outer: moveLoop(stadium(sl+sd, sd, p.Segs(sd)), sl/2, 0)}, -d*0.45, over)
			under := extrudeAt(Shape{Outer: moveLoop(stadium(sl+hd, hd, p.Segs(hd)), sl/2, 0)}, -d, -d*0.45+0.01)
			return UnionAll([]*Mesh{big, slot, under}), nil
		})

	hole("teardrop", "Teardrop hole", "Horizontal hole with a pointed top so it prints without supports. Lay it sideways.", "teardrop horizontal hole support free",
		false, []Param{num("d", "Diameter", 6, 0.5, 200, 0.5, "mm"), depth(20)},
		func(p Params) (*Mesh, error) {
			d, l := p.F("d", 6), p.F("depth", 20)
			r := d / 2
			// pointed top (local +Y is "up" when placed on a side face)
			out := Loop{{0, r * math.Sqrt2}}
			n := p.Segs(d)
			for i := 0; i <= n; i++ {
				a := (135 + 270*float64(i)/float64(n)) * math.Pi / 180
				out = append(out, P2{r * math.Cos(a), r * math.Sin(a)})
			}
			return extrudeAt(Shape{Outer: out}, -l, over), nil
		})

	hole("shafthole", "Shaft hole (D / keyed / hex)", "Hole shaped to grip a motor or knob shaft so it can't spin.", "d shaft flat keyway double d hex shaft motor",
		false, []Param{sel("kind", "Shape", "D-flat", "D-flat", "Double-D", "Keyway", "Hex", "Square", "Spline"), num("d", "Shaft Ø", 5, 1, 100, 0.1, "mm"), num("tol", "Tolerance", 0.15, 0, 1, 0.05, "mm"), depth(10)},
		func(p Params) (*Mesh, error) {
			d := p.F("d", 5) + p.F("tol", 0.15)
			n := p.Segs(d)
			var l Loop
			switch p.S("kind", "D-flat") {
			case "Hex":
				l = NGon(d/math.Sqrt(3), 6, 0)
			case "Square":
				l = NGon(d/math.Sqrt2, 4, 45)
			case "Spline":
				l = knurlLoop(d, d*0.1, 12, n)
			default:
				l = Circle(d/2, n)
				for i := range l {
					switch p.S("kind", "") {
					case "D-flat":
						l[i].X = math.Min(l[i].X, d/2-d*0.1)
					case "Double-D":
						l[i].X = math.Max(-(d/2 - d*0.1), math.Min(l[i].X, d/2-d*0.1))
					}
				}
				if p.S("kind", "") == "Keyway" {
					m := extrudeAt(Shape{Outer: l}, -p.F("depth", 10), over)
					return Union(m, boxAt(V(0, -d*0.13, -p.F("depth", 10)), V(d*0.62, d*0.13, over))), nil
				}
			}
			return extrudeAt(Shape{Outer: l}, -p.F("depth", 10), over), nil
		})

	hole("ventslots", "Vent slots", "A row of rounded slots for airflow.", "vent slots grille airflow cooling",
		false, []Param{num("count", "Slots", 6, 1, 60, 1, ""), num("len", "Slot length", 20, 3, 300, 1, "mm"), num("w", "Slot width", 2.5, 0.8, 20, 0.1, "mm"), num("gap", "Gap", 2.5, 0.8, 20, 0.1, "mm"), depth(3)},
		func(p Params) (*Mesh, error) {
			n, l, w, g, d := p.I("count", 6), p.F("len", 20), p.F("w", 2.5), p.F("gap", 2.5), p.F("depth", 3)
			var ls []Loop
			for i := 0; i < n; i++ {
				y := (float64(i) - float64(n-1)/2) * (w + g)
				ls = append(ls, moveLoop(stadium(l, w, 12), 0, y))
			}
			m := &Mesh{}
			for _, s := range ShapesFromLoops(ls) {
				m.Append(extrudeAt(s, -d, over))
			}
			return m, nil
		})

	hole("holegrid", "Hole grid", "A grid of round holes — speaker grilles, sieves, perforated panels.", "hole grid perforated speaker grille sieve array",
		false, []Param{num("nx", "Columns", 6, 1, 60, 1, ""), num("ny", "Rows", 4, 1, 60, 1, ""), num("d", "Hole Ø", 3, 0.5, 50, 0.1, "mm"), num("pitch", "Spacing", 5, 1, 100, 0.5, "mm"), boolp("stagger", "Staggered rows", true), depth(3)},
		func(p Params) (*Mesh, error) {
			nx, ny, d, s, dep := p.I("nx", 6), p.I("ny", 4), p.F("d", 3), p.F("pitch", 5), p.F("depth", 3)
			m := &Mesh{}
			for j := 0; j < ny; j++ {
				off := 0.0
				if p.B("stagger", true) && j%2 == 1 {
					off = s / 2
				}
				for i := 0; i < nx; i++ {
					m.Append(cylAt((float64(i)-float64(nx-1)/2)*s+off, (float64(j)-float64(ny-1)/2)*s*0.9, d, -dep, over, p.Segs(d)))
				}
			}
			return m, nil
		})

	hole("sqnuttrap", "Square nut slot", "Slot to slide a square nut in from the side, with the bolt hole through it.", "square nut slot trap captive",
		true, []Param{sel("size", "Size", "M3", metricList()...), num("slot", "Slot length", 10, 2, 100, 0.5, "mm"), num("boltDepth", "Bolt hole depth", 8, 0, 200, 0.5, "mm"), num("tol", "Tolerance", 0.25, 0, 1, 0.05, "mm")},
		func(p Params) (*Mesh, error) {
			m := getMetric(p)
			t := p.F("tol", 0.25)
			s := m.SqAF + 2*t
			nz := -m.D * 1.2
			pocket := boxAt(V(-s/2, -s/2, nz-m.SqH-t), V(p.F("slot", 10), s/2, nz))
			bolt := cylAt(0, 0, m.Normal+t, nz-m.SqH-p.F("boltDepth", 8), over, p.Segs(m.D))
			return Union(pocket, bolt), nil
		})

	hole("tslot", "T-slot channel", "T-shaped channel for sliding bolt heads, T-nuts or rails.", "t slot channel rail track bolt head",
		false, []Param{num("len", "Length", 40, 5, 1000, 1, "mm"), num("neck", "Neck width", 6, 1, 50, 0.5, "mm"), num("head", "Head width", 11, 2, 80, 0.5, "mm"), num("neckH", "Neck depth", 3, 0.5, 30, 0.5, "mm"), num("headH", "Head depth", 4, 0.5, 30, 0.5, "mm")},
		func(p Params) (*Mesh, error) {
			l, nw, hw, nh, hh := p.F("len", 40), p.F("neck", 6), p.F("head", 11), p.F("neckH", 3), p.F("headH", 4)
			pr := Loop{{-nw / 2, over}, {-nw / 2, -nh}, {-hw / 2, -nh}, {-hw / 2, -nh - hh}, {hw / 2, -nh - hh}, {hw / 2, -nh}, {nw / 2, -nh}, {nw / 2, over}}
			return prismY(pr, l), nil
		})

	hole("dovetail", "Dovetail slot", "Dovetail groove that a matching Dovetail rail slides into.", "dovetail slot groove joint slide",
		false, []Param{num("len", "Length", 40, 5, 1000, 1, "mm"), num("w", "Top width", 8, 2, 100, 0.5, "mm"), num("h", "Depth", 4, 1, 50, 0.5, "mm"), num("angle", "Angle", 60, 45, 80, 1, "°"), num("tol", "Clearance", 0.2, 0, 1, 0.05, "mm")},
		func(p Params) (*Mesh, error) {
			return dovetailBody(p.F("len", 40), p.F("w", 8)+2*p.F("tol", 0.2), p.F("h", 4)+p.F("tol", 0.2), p.F("angle", 60), true), nil
		})

	hole("cablehole", "Cable pass-through", "Round cable hole with a side slot so a plugged cable can be pushed in.", "cable hole pass through grommet wire slot",
		false, []Param{num("d", "Hole Ø", 8, 2, 60, 0.5, "mm"), num("cable", "Cable Ø", 4, 1, 30, 0.5, "mm"), num("depth", "Depth", 3, 0.5, 100, 0.5, "mm")},
		func(p Params) (*Mesh, error) {
			d, c, dep := p.F("d", 8), p.F("cable", 4), p.F("depth", 3)
			return Union(cylAt(0, 0, d, -dep, over, p.Segs(d)), boxAt(V(0, -c*0.45, -dep), V(d*1.5, c*0.45, over))), nil
		})

	ports := map[string][3]float64{"USB-C": {9.2, 3.4, 1.6}, "USB-A": {13.2, 5.9, 0.5}, "Micro-USB": {8, 3.2, 1}, "Mini-USB": {7.8, 4.2, 0.8}, "HDMI": {15.2, 6, 0.8}, "Micro-HDMI": {6.8, 3.4, 0.6}, "RJ45 Ethernet": {16.5, 14, 0.6}, "SD card": {12.5, 2.5, 0.5}, "Micro-SD": {11.5, 1.8, 0.4}, "3.5 mm audio": {6.2, 6.2, 3.1}}
	pnames := []string{"USB-C", "USB-A", "Micro-USB", "Mini-USB", "HDMI", "Micro-HDMI", "RJ45 Ethernet", "SD card", "Micro-SD", "3.5 mm audio"}
	hole("portcut", "Connector cutout", "Correctly sized panel cutout for common connectors (USB-C, USB-A, HDMI, RJ45, SD…).", "usb c usb a hdmi rj45 ethernet port cutout connector sd card audio jack",
		true, []Param{sel("port", "Connector", "USB-C", pnames...), num("tol", "Extra clearance", 0.4, 0, 3, 0.1, "mm"), depth(3)},
		func(p Params) (*Mesh, error) {
			d := ports[p.S("port", "USB-C")]
			if d[0] == 0 {
				d = ports["USB-C"]
			}
			t := p.F("tol", 0.4)
			// cutouts are drilled straight in: X = width, Y = height
			return extrudeAt(Shape{Outer: RoundedRect(d[0]+t, d[1]+t, d[2]+t/2, 6)}, -p.F("depth", 3), over), nil
		})

	hole("ledhole", "LED / button hole", "Hole for a 3 / 5 mm LED or round push button, with a recess for the rim.", "led hole button push light indicator",
		true, []Param{sel("kind", "For", "5 mm LED", "3 mm LED", "5 mm LED", "12 mm push button", "16 mm push button", "22 mm push button"), depth(4)},
		func(p Params) (*Mesh, error) {
			v := map[string][2]float64{"3 mm LED": {3.1, 4}, "5 mm LED": {5.1, 6}, "12 mm push button": {12.2, 14}, "16 mm push button": {16.2, 19}, "22 mm push button": {22.3, 26}}[p.S("kind", "5 mm LED")]
			if v[0] == 0 {
				v = [2]float64{5.1, 6}
			}
			dep := p.F("depth", 4)
			return RevolveProfile([]P2{{0, -dep}, {v[0] / 2, -dep}, {v[0] / 2, -dep * 0.4}, {v[1] / 2, -dep * 0.4}, {v[1] / 2, over}, {0, over}}, p.Segs(v[1])), nil
		})

	hole("switchhole", "Panel switch hole", "Round panel-mount hole with an anti-rotation key flat (toggle switches, pots, DC jacks).", "toggle switch potentiometer pot panel mount dc jack barrel",
		false, []Param{sel("kind", "For", "Potentiometer (7 mm)", "Toggle switch (6 mm)", "Toggle switch (12 mm)", "Potentiometer (7 mm)", "DC barrel jack (8 mm)", "GX12 connector (12 mm)", "Custom"), num("d", "Diameter", 7, 1, 60, 0.1, "mm").show("kind=Custom"), depth(3)},
		func(p Params) (*Mesh, error) {
			d := map[string]float64{"Toggle switch (6 mm)": 6.2, "Toggle switch (12 mm)": 12.2, "Potentiometer (7 mm)": 7.2, "DC barrel jack (8 mm)": 8.1, "GX12 connector (12 mm)": 12.2}[p.S("kind", "")]
			if d == 0 {
				d = p.F("d", 7) + 0.2
			}
			l := Circle(d/2, p.Segs(d))
			for i := range l {
				l[i].Y = math.Min(l[i].Y, d/2-d*0.07)
			}
			return extrudeAt(Shape{Outer: l}, -p.F("depth", 3), over), nil
		})

	hole("fanmount", "Fan mount", "Airflow opening plus the 4 screw holes for a standard PC fan.", "fan mount 40mm 60mm 80mm 120mm cooling",
		true, []Param{sel("fan", "Fan size", "40 mm", "25 mm", "30 mm", "40 mm", "50 mm", "60 mm", "80 mm", "92 mm", "120 mm"), sel("grill", "Opening", "Open circle", "Open circle", "Rings", "Spokes"), depth(3)},
		func(p Params) (*Mesh, error) {
			fs := map[string][2]float64{"25 mm": {25, 20}, "30 mm": {30, 24}, "40 mm": {40, 32}, "50 mm": {50, 40}, "60 mm": {60, 50}, "80 mm": {80, 71.5}, "92 mm": {92, 82.5}, "120 mm": {120, 105}}
			f := fs[p.S("fan", "40 mm")]
			if f[0] == 0 {
				f = fs["40 mm"]
			}
			dep := p.F("depth", 3)
			od := f[0] * 0.95
			m := cylAt(0, 0, od, -dep, over, p.Segs(od))
			if g := p.S("grill", ""); g != "Open circle" {
				var bars []*Mesh
				if g == "Rings" {
					for r := od * 0.18; r < od/2; r += od * 0.14 {
						bars = append(bars, ring(2*r+1.6, 2*r, -dep-1, over+1, p.Segs(od)))
					}
				} else {
					for k := 0; k < 4; k++ {
						bars = append(bars, boxAt(V(-od/2-1, -0.9, -dep-1), V(od/2+1, 0.9, over+1)).Transformed(RotZ(45*float64(k))))
					}
				}
				bars = append(bars, cylAt(0, 0, od*0.25, -dep-1, over+1, p.Segs(od)))
				m = Subtract(m, UnionAll(bars))
			}
			s := f[1] / 2
			hs := &Mesh{}
			sd := 3.3
			if f[0] <= 30 {
				sd = 2.7
			} else if f[0] >= 80 {
				sd = 4.4
			}
			for _, q := range [][2]float64{{s, s}, {-s, s}, {s, -s}, {-s, -s}} {
				hs.Append(cylAt(q[0], q[1], sd, -dep, over, p.Segs(sd)))
			}
			return Union(m, hs), nil
		})

	hole("nemamount", "Stepper motor mount", "Centre boss hole and 4 screw holes for NEMA 8/11/14/17/23 stepper motors.", "nema 17 stepper motor mount nema17 nema23",
		true, []Param{sel("nema", "Motor", "NEMA 17", "NEMA 8", "NEMA 11", "NEMA 14", "NEMA 17", "NEMA 23"), boolp("slots", "Slotted holes (belt tension)", false), depth(5)},
		func(p Params) (*Mesh, error) {
			nm := map[string][3]float64{"NEMA 8": {16, 15.4, 2.2}, "NEMA 11": {22, 23, 2.7}, "NEMA 14": {22, 26, 3.4}, "NEMA 17": {22.4, 31, 3.4}, "NEMA 23": {38.5, 47.14, 5.2}}[p.S("nema", "NEMA 17")]
			if nm[0] == 0 {
				nm = [3]float64{22.4, 31, 3.4}
			}
			dep := p.F("depth", 5)
			m := cylAt(0, 0, nm[0], -dep, over, p.Segs(nm[0]))
			s := nm[1] / 2
			for _, q := range [][2]float64{{s, s}, {-s, s}, {s, -s}, {-s, -s}} {
				if p.B("slots", false) {
					m.Append(extrudeAt(Shape{Outer: moveLoop(stadium(nm[2]+6, nm[2], 12), q[0], q[1])}, -dep, over))
				} else {
					m.Append(cylAt(q[0], q[1], nm[2], -dep, over, p.Segs(nm[2])))
				}
			}
			return m, nil
		})

	hole("boltcircle", "Bolt circle", "Holes evenly spaced on a circle — flanges, wheels, lids.", "bolt circle pcd pattern flange holes",
		false, []Param{num("count", "Holes", 6, 2, 72, 1, ""), num("pcd", "Circle Ø", 40, 2, 1000, 1, "mm"), num("d", "Hole Ø", 3.4, 0.5, 50, 0.1, "mm"), num("start", "Start angle", 0, -360, 360, 5, "°"), depth(5)},
		func(p Params) (*Mesh, error) {
			n, r, d, dep := p.I("count", 6), p.F("pcd", 40)/2, p.F("d", 3.4), p.F("depth", 5)
			m := &Mesh{}
			for i := 0; i < n; i++ {
				a := (p.F("start", 0) + 360*float64(i)/float64(n)) * math.Pi / 180
				m.Append(cylAt(r*math.Cos(a), r*math.Sin(a), d, -dep, over, p.Segs(d)))
			}
			return m, nil
		})

	hole("pocket", "Rounded pocket", "Rectangular recess with rounded corners and an optional rounded floor — trays, inlays, lids.", "pocket recess tray inlay cavity",
		false, []Param{num("w", "Width", 30, 1, 1000, 1, "mm"), num("d", "Length", 20, 1, 1000, 1, "mm"), num("depth", "Depth", 5, 0.2, 500, 0.5, "mm"), num("r", "Corner radius", 3, 0, 200, 0.5, "mm"), boolp("scoop", "Rounded floor (easy to grab from)", false)},
		func(p Params) (*Mesh, error) {
			w, d, dep, r := p.F("w", 30), p.F("d", 20), p.F("depth", 5), p.F("r", 3)
			if p.B("scoop", false) {
				return roundedBlock(w, d, dep+over, r, 0, math.Min(dep*0.6, math.Min(w, d)/2-0.1), false, false, 6).Transformed(Translate(0, 0, -dep)), nil
			}
			return roundedBlock(w, d, dep+over, r, 0, 0, false, false, 6).Transformed(Translate(0, 0, -dep)), nil
		})

	hole("oringgroove", "O-ring groove", "Gland groove for an O-ring seal (face seal).", "o-ring groove seal gland gasket waterproof",
		false, []Param{num("id", "O-ring inner Ø", 20, 2, 500, 0.5, "mm"), num("cs", "Cord thickness", 2, 0.5, 10, 0.1, "mm")},
		func(p Params) (*Mesh, error) {
			id, cs := p.F("id", 20), p.F("cs", 2)
			w, dep := cs*1.35, cs*0.75
			return ring(id+2*w, id, -dep, over, p.Segs(id+2*w)), nil
		})

	hole("edgeround", "Edge rounder", "Place on an outside edge and group it to round (fillet) or chamfer that edge.", "fillet round edge chamfer cutter bevel",
		false, []Param{sel("kind", "Edge", "Round (fillet)", "Round (fillet)", "Chamfer"), num("r", "Size", 3, 0.2, 100, 0.2, "mm"), num("len", "Edge length", 40, 1, 2000, 1, "mm")},
		func(p Params) (*Mesh, error) {
			r, l := p.F("r", 3), p.F("len", 40)
			// cutter sits in the corner at the origin, edge along Y, solid in -X / -Z
			if p.S("kind", "") == "Chamfer" {
				return prismY(Loop{{-r, over}, {over, over}, {over, -r}}, l+2*over), nil
			}
			pr := Loop{{over, over}, {over, -r}, {0, -r}}
			for i := 0; i <= 12; i++ {
				a := (90 * float64(i) / 12) * math.Pi / 180
				pr = append(pr, P2{-r + r*math.Cos(a), -r + r*math.Sin(a)})
			}
			pr = append(pr, P2{-r, over})
			return prismY(pr.Clean(), l+2*over), nil
		})

	hole("snapsocket", "Snap-fit socket", "Rectangular hole with a catch ledge for the Snap-fit clip.", "snap fit socket catch latch receiver",
		false, []Param{num("w", "Clip width", 6, 1, 100, 0.5, "mm"), num("t", "Clip thickness", 1.6, 0.6, 10, 0.1, "mm"), num("hook", "Hook depth", 1.2, 0.2, 10, 0.1, "mm"), num("depth", "Wall thickness", 3, 1, 50, 0.5, "mm")},
		func(p Params) (*Mesh, error) {
			w, t, hk, dep := p.F("w", 6)+0.4, p.F("t", 1.6)+0.3, p.F("hook", 1.2)+0.2, p.F("depth", 3)
			slot := boxAt(V(0, -w/2, -dep-1), V(t, w/2, over))
			catch := boxAt(V(t-0.01, -w/2, -dep-1), V(t+hk, w/2, -dep+0.01))
			return Union(slot, catch), nil
		})

	hole("livinghinge", "Living hinge cuts", "Staggered slot pattern that makes a flat plate bend like fabric.", "living hinge flex lattice bend kerf",
		false, []Param{num("w", "Width (bend line length)", 40, 5, 500, 1, "mm"), num("l", "Hinge zone", 20, 3, 300, 1, "mm"), num("slot", "Slot width", 0.8, 0.3, 5, 0.1, "mm"), num("pitch", "Spacing", 2, 0.8, 10, 0.1, "mm"), depth(3)},
		func(p Params) (*Mesh, error) {
			w, l, s, pt, dep := p.F("w", 40), p.F("l", 20), p.F("slot", 0.8), p.F("pitch", 2), p.F("depth", 3)
			m := &Mesh{}
			seg := w / 3
			row := 0
			for x := -l / 2; x <= l/2+1e-6; x += pt {
				if row%2 == 0 {
					for _, y := range []float64{-seg, seg} {
						m.Append(boxAt(V(x-s/2, y-seg*0.45, -dep), V(x+s/2, y+seg*0.45, over)))
					}
				} else {
					m.Append(boxAt(V(x-s/2, -seg*0.45, -dep), V(x+s/2, seg*0.45, over)))
					m.Append(boxAt(V(x-s/2, w/2-seg*0.3, -dep), V(x+s/2, w/2+1, over)))
					m.Append(boxAt(V(x-s/2, -w/2-1, -dep), V(x+s/2, -w/2+seg*0.3, over)))
				}
				row++
			}
			return m, nil
		})

	hole("ballsocket", "Ball socket", "Spherical cavity with a narrow mouth so a Ball joint snaps in and swivels.", "ball socket joint snap swivel",
		false, []Param{num("d", "Ball Ø", 10, 2, 100, 0.5, "mm"), num("tol", "Clearance", 0.3, 0, 2, 0.05, "mm"), num("mouth", "Mouth (% of ball)", 85, 50, 100, 1, "%")},
		func(p Params) (*Mesh, error) {
			d := p.F("d", 10) + p.F("tol", 0.3)
			r := d / 2
			mouth := r * p.F("mouth", 85) / 100
			zc := -r * 1.05
			n := p.Segs(d) / 2
			pts := []P2{}
			for i := 0; i <= n; i++ {
				a := -math.Pi/2 + math.Pi*float64(i)/float64(n)
				x, z := r*math.Cos(a), zc+r*math.Sin(a)
				if x < mouth && z > zc {
					break
				}
				pts = append(pts, P2{x, z})
			}
			pts[0].X = 0
			pts = append(pts, P2{mouth, over}, P2{0, over})
			return RevolveProfile(pts, p.Segs(d)), nil
		})

	hole("fingergrip", "Finger scoop", "Oval scoop so a lid, drawer or tray is easy to grab.", "finger grip scoop notch pull lid",
		false, []Param{num("w", "Width", 20, 5, 200, 1, "mm"), num("l", "Length", 12, 3, 200, 1, "mm"), num("depth", "Depth", 4, 0.5, 50, 0.5, "mm")},
		func(p Params) (*Mesh, error) {
			w, l, dep := p.F("w", 20), p.F("l", 12), p.F("depth", 4)
			n := p.Segs(w) / 2
			var pts []P2
			for i := 0; i <= n; i++ {
				a := -math.Pi / 2 * (1 - float64(i)/float64(n))
				pts = append(pts, P2{0.5 * math.Cos(a), dep * math.Sin(a)})
			}
			pts = append([]P2{{0, -dep}}, pts...)
			pts = append(pts, P2{0.5, over}, P2{0, over})
			return RevolveProfile(pts, p.Segs(w)).Transformed(Scale(w, l, 1)), nil
		})

	hole("wirechannel", "Wire channel", "Rounded-bottom trough for routing wires along or inside a part.", "wire channel cable trough route groove",
		false, []Param{num("len", "Length", 50, 2, 2000, 1, "mm"), num("w", "Width", 5, 0.5, 100, 0.5, "mm"), num("depth", "Depth", 4, 0.5, 100, 0.5, "mm")},
		func(p Params) (*Mesh, error) {
			l, w, d := p.F("len", 50), p.F("w", 5), p.F("depth", 4)
			pr := Loop{{-w / 2, over}, {-w / 2, -d + w/2}}
			for i := 1; i < 12; i++ {
				a := math.Pi + math.Pi*float64(i)/12
				pr = append(pr, P2{w / 2 * math.Cos(a), -d + w/2 + w/2*math.Sin(a)})
			}
			pr = append(pr, P2{w / 2, -d + w/2}, P2{w / 2, over})
			return Sweep(Shape{Outer: pr}, []Frame{{V(0, -l/2, 0), V(1, 0, 0), V(0, 0, 1)}, {V(0, l/2, 0), V(1, 0, 0), V(0, 0, 1)}}, false).Transformed(RotZ(90)), nil
		})

	hole("taperhole", "Tapered hole", "Conical hole — press fits, funnels, pipe tapers and drain holes.", "taper cone hole conical press fit drain",
		false, []Param{num("top", "Top Ø", 10, 0.5, 500, 0.5, "mm"), num("bottom", "Bottom Ø", 6, 0, 500, 0.5, "mm"), depth(10)},
		func(p Params) (*Mesh, error) {
			t, b, d := p.F("top", 10), p.F("bottom", 6), p.F("depth", 10)
			tt := t + (t-b)/d*over
			return RevolveProfile([]P2{{0, -d}, {b / 2, -d}, {tt / 2, over}, {0, over}}, p.Segs(t)), nil
		})
}

// dovetailBody makes a dovetail (rail if hole=false, slot if hole=true) along Y.
func dovetailBody(l, w, h, angle float64, hole bool) *Mesh {
	inset := h / math.Tan(angle*math.Pi/180)
	top := 0.0
	if hole {
		top = over
		// slot: narrow at the surface, wide at the bottom
		return prismY(Loop{{-w / 2, top}, {-w/2 - inset, -h}, {w/2 + inset, -h}, {w / 2, top}}, l)
	}
	return prismY(Loop{{-w/2 - inset, h}, {-w / 2, 0}, {w / 2, 0}, {w/2 + inset, h}}, l)
}
