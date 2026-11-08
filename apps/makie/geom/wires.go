package geom

import (
	"fmt"
	"math"
)

// ===================== Wire & cable management =====================

func init() {
	hole := func(typ, name, desc, tags string, params []Param, gen func(p Params) (*Mesh, error)) {
		register(&ShapeDef{Type: typ, Name: name, Category: "Holes", Hole: true, Desc: desc, Tags: tags + " wire cable", Color: "#64748b", Params: params, Gen: gen})
	}
	solid := func(typ, name, desc, tags string, params []Param, gen func(p Params) (*Mesh, error)) {
		register(&ShapeDef{Type: typ, Name: name, Category: "Mechanical", Desc: desc, Tags: tags + " wire cable", Color: "#64748b", Params: params, Gen: gen})
	}

	// ---- cut-outs ----
	hole("wirechannel", "Wire channel", "Groove for running wires inside a part — straight, round a corner or a U-turn, with an optional ledge for a snap-on cover.",
		"wire channel groove route trench",
		[]Param{num("w", "Width", 6, 1, 100, 0.5, "mm"), num("depth", "Depth", 5, 0.5, 100, 0.5, "mm"), num("len", "Length", 40, 2, 1000, 1, "mm"),
			sel("shape", "Path", "Straight", "Straight", "Corner (90°)", "U-turn", "S-bend"), num("r", "Bend radius", 10, 0, 200, 0.5, "mm").show("shape!=Straight"),
			boolp("ledge", "Ledge for a cover", false)},
		func(p Params) (*Mesh, error) {
			w, d, L := p.F("w", 6), p.F("depth", 5), p.F("len", 40)
			r := math.Max(p.F("r", 10), w/2+0.5)
			var pts []Vec3
			line := func(a, b Vec3) {
				n := int(math.Max(2, math.Ceil(b.Sub(a).Len()/4)))
				for i := 0; i <= n; i++ {
					q := a.Add(b.Sub(a).Mul(float64(i) / float64(n)))
					if len(pts) > 0 && pts[len(pts)-1].Sub(q).Len() < 1e-6 {
						continue
					}
					pts = append(pts, q)
				}
			}
			arc := func(cx, cy, a0, a1 float64) {
				n := int(math.Max(8, math.Abs(a1-a0)*r/2))
				for i := 0; i <= n; i++ {
					a := a0 + (a1-a0)*float64(i)/float64(n)
					q := V(cx+r*math.Cos(a), cy+r*math.Sin(a), 0)
					if len(pts) > 0 && pts[len(pts)-1].Sub(q).Len() < 1e-6 {
						continue
					}
					pts = append(pts, q)
				}
			}
			switch p.S("shape", "Straight") {
			case "Corner (90°)":
				line(V(-L/2, 0, 0), V(0, 0, 0))
				arc(0, r, -math.Pi/2, 0)
				line(V(r, r, 0), V(r, r+L/2, 0))
			case "U-turn":
				line(V(-L/2, 0, 0), V(0, 0, 0))
				arc(0, r, -math.Pi/2, math.Pi/2)
				line(V(0, 2*r, 0), V(-L/2, 2*r, 0))
			case "S-bend":
				line(V(-L/2, 0, 0), V(0, 0, 0))
				arc(0, r, -math.Pi/2, 0)
				arc(2*r, r, math.Pi, math.Pi/2)
				line(V(2*r, 2*r, 0), V(2*r+L/2, 2*r, 0))
			default:
				line(V(-L/2, 0, 0), V(L/2, 0, 0))
			}
			sweep := func(hw, z0, z1 float64) *Mesh {
				var fr []Frame
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
					u := V(0, 0, 1).Cross(t.Norm()).Norm()
					fr = append(fr, Frame{pts[i], u, V(0, 0, 1)})
				}
				// round bottom so wires lie neatly and nothing snags
				var l Loop
				for i := 0; i <= 12; i++ {
					a := math.Pi + math.Pi*float64(i)/12
					l = append(l, P2{hw * math.Cos(a), z0 + hw + hw*math.Sin(a)})
				}
				l = append(l, P2{hw, z1}, P2{-hw, z1})
				return Sweep(Shape{Outer: l}, fr, false)
			}
			m := sweep(w/2, -d, over)
			if p.B("ledge", false) {
				m = Union(m, sweep(w/2+1.2, -1.6, over))
			}
			return m, nil
		})
	hole("cablenotch", "Cable notch", "U-shaped notch in an edge so a cable can leave a box or lid.", "cable notch exit slot edge",
		[]Param{num("d", "Cable Ø", 5, 1, 60, 0.5, "mm"), num("depth", "Wall thickness", 3, 0.5, 50, 0.5, "mm")},
		func(p Params) (*Mesh, error) {
			d := p.F("d", 5) + 0.6
			l := moveLoop(stadium(d+20, d, p.Segs(d)), 10, 0)
			return extrudeAt(Shape{Outer: rotLoop(l, 90)}, -p.F("depth", 3)-over, over).Transformed(Translate(0, 0, 0)), nil
		})
	hole("wirepass", "Plug pass-through", "Hole big enough to push a plug through, with a slot that leaves just a snug fit round the cable.", "plug cable pass through usb hole slot",
		[]Param{num("plug", "Plug width", 12, 3, 60, 0.5, "mm"), num("plugh", "Plug height", 7, 2, 60, 0.5, "mm"), num("cable", "Cable Ø", 4, 1, 30, 0.5, "mm"), num("depth", "Depth", 3, 0.5, 50, 0.5, "mm")},
		func(p Params) (*Mesh, error) {
			pw, ph, c, d := p.F("plug", 12)+0.8, p.F("plugh", 7)+0.8, p.F("cable", 4)+0.4, p.F("depth", 3)
			plug := extrudeAt(Shape{Outer: RoundedRect(pw, ph, math.Min(ph, pw)/4, 24)}, -d, over).Transformed(Translate(0, ph/2+c, 0))
			slot := extrudeAt(Shape{Outer: moveLoop(stadium(ph/2+c, c, 16), 0, 0)}, -d, over)
			slot = extrudeAt(Shape{Outer: rotLoop(moveLoop(stadium(ph/2+c+c/2, c, 16), (ph/2+c)/2-c/4, 0), 90)}, -d, over)
			return Union(plug, slot), nil
		})

	// ---- solids ----
	solid("grommet", "Cable grommet", "Flexible-petal grommet that clips into a round hole and grips the cable.", "grommet bushing cable hole",
		[]Param{num("hole", "Panel hole Ø", 16, 6, 80, 0.5, "mm"), num("panel", "Panel thickness", 3, 0.6, 10, 0.2, "mm"), num("cable", "Cable Ø", 6, 1, 60, 0.5, "mm"), num("petals", "Petals", 8, 4, 16, 1, "")},
		func(p Params) (*Mesh, error) {
			hd, pt, cd := p.F("hole", 16), p.F("panel", 3), p.F("cable", 6)
			if cd > hd-2*MinWall-1 {
				return nil, fmt.Errorf("cable is too big for that hole")
			}
			segs := p.Segs(hd * 1.4)
			flange := ring(hd+6, hd-2*MinWall, 0, 1.6, segs)
			body := ring(hd-0.2, hd-0.2-2*MinWall, 1.59, 1.6+pt, segs)
			lip := coneAt(hd+1.2, hd-0.2, 1.6+pt, 1.6+pt+1.2, segs)
			lip = Subtract(lip, cylAt(0, 0, hd-0.2-2*MinWall, 0, 10+pt, segs))
			// petals closing in on the cable
			pet := Subtract(cylAt(0, 0, hd-2*MinWall+0.01, 0, 1.6, segs), cylAt(0, 0, cd*0.85, -1, 3, segs))
			n := p.I("petals", 8)
			var slits []*Mesh
			for i := 0; i < n; i++ {
				slits = append(slits, boxAt(V(0, -0.4, -1), V(hd, 0.4, 3)).Transformed(RotZ(360*float64(i)/float64(n))))
			}
			pet = Subtract(pet, UnionAll(slits))
			// split the barbed ring so it can squeeze into the hole
			split := boxAt(V(-0.6, hd/2-3, 1.7), V(0.6, hd, 10+pt))
			m := Subtract(UnionAll([]*Mesh{flange, body, lip}), split)
			return Union(m, pet), nil
		})
	solid("raceway", "Cable raceway", "Square trunking with a snap-on lid — screw it to a wall or desk.", "raceway trunking duct cable cover",
		[]Param{num("w", "Inside width", 16, 5, 100, 0.5, "mm"), num("h", "Inside height", 10, 4, 100, 0.5, "mm"), num("len", "Length", 100, 10, 1000, 1, "mm"), sel("screw", "Screw holes", "M3", "None", "M3", "M4"), sel("part", "Show", "Both", "Both", "Channel", "Lid")},
		func(p Params) (*Mesh, error) {
			w, h, L := p.F("w", 16), p.F("h", 10), p.F("len", 100)
			t := 1.6
			chanM := func() *Mesh {
				out := shapeX(Shape{Outer: rectL(-w/2-t, 0, w/2+t, h+t)}, 0, L)
				in := shapeX(Shape{Outer: rectL(-w/2, t, w/2, h+t+1)}, -1, L+1)
				// little ridges near the top for the lid to snap behind
				ridges := Union(shapeX(Shape{Outer: Loop{{-w / 2, h + t - 1.5}, {-w/2 + 0.6, h + t - 1}, {-w / 2, h + t - 0.5}}}, 0, L), shapeX(Shape{Outer: Loop{{w / 2, h + t - 1.5}, {w / 2, h + t - 0.5}, {w/2 - 0.6, h + t - 1}}}, 0, L))
				m := Union(Subtract(out, in), ridges)
				if sz := p.S("screw", "M3"); sz != "None" {
					hd := metrics[sz].Normal + 0.2
					var holes []*Mesh
					for x := 15.0; x < L-5; x += 60 {
						holes = append(holes, cylAt(x, 0, hd, -1, t+1, 20), coneAt(hd, hd+3.4, t-1.7, t+0.01, 20).Transformed(Translate(x, 0, 0)))
					}
					m = Subtract(m, UnionAll(holes))
				}
				return m
			}
			lid := func() *Mesh {
				top := shapeX(Shape{Outer: rectL(-w/2-t, 0, w/2+t, t)}, 0, L)
				legs := shapeX(Shape{Outer: Loop{{-w/2 + 0.15, -2.2}, {-w/2 + 0.15 + 1.2, -2.2}, {-w/2 + 0.15 + 1.2, 0.01}, {-w/2 + 0.15, 0.01}}}, 0, L)
				legs = Union(legs, shapeX(Shape{Outer: Loop{{w/2 - 0.15 - 1.2, -2.2}, {w/2 - 0.15, -2.2}, {w/2 - 0.15, 0.01}, {w/2 - 0.15 - 1.2, 0.01}}}, 0, L))
				hook := Union(shapeX(Shape{Outer: Loop{{-w/2 + 0.15, -2.2}, {-w/2 + 0.15, -1.5}, {-w/2 - 0.45, -1.85}}}, 0, L), shapeX(Shape{Outer: Loop{{w/2 - 0.15, -2.2}, {w/2 + 0.45, -1.85}, {w/2 - 0.15, -1.5}}}, 0, L))
				return Union(top, Union(legs, hook)).Transformed(Translate(0, 0, 2.2))
			}
			switch p.S("part", "Both") {
			case "Channel":
				return chanM(), nil
			case "Lid":
				return lid(), nil
			}
			return Union(chanM(), lid().Transformed(Translate(0, w+2*t+6, 0))), nil
		})
	solid("cablecomb", "Cable comb", "Keeps a bundle of wires neat and parallel.", "cable comb organiser loom wire",
		[]Param{num("n", "Wires", 6, 2, 30, 1, ""), num("d", "Wire Ø", 3, 0.5, 20, 0.1, "mm"), num("t", "Thickness", 4, 2, 20, 0.5, "mm")},
		func(p Params) (*Mesh, error) {
			n, d, t := p.I("n", 6), p.F("d", 3), p.F("t", 4)
			pitch := d + 1.6
			W := float64(n)*pitch + 1.6
			H := d*2 + 3
			m := extrudeAt(Shape{Outer: RoundedRect(W, H, 1.5, 12)}, 0, t)
			var cuts []*Mesh
			for i := 0; i < n; i++ {
				x := -W/2 + 1.6 + pitch*(float64(i)+0.5) - 0.8
				cuts = append(cuts, cylAt(x, 0, d+0.2, -1, t+1, 20), boxAt(V(x-d*0.35, 0, -1), V(x+d*0.35, H, t+1)))
			}
			return Subtract(m, UnionAll(cuts)), nil
		})
	solid("strainrelief", "Cable strain relief", "Screw-down clamp that stops a cable being pulled out of your project.", "strain relief cable clamp anchor",
		[]Param{num("d", "Cable Ø", 5, 1, 30, 0.5, "mm"), sel("screw", "Screws", "M3", "M2.5", "M3", "M4")},
		func(p Params) (*Mesh, error) {
			d := p.F("d", 5)
			hd := metrics[p.S("screw", "M3")].Normal + 0.2
			e := holeEdge(hd)
			W := d + 4*e + 2
			H := d*0.75 + 2.4
			wd := math.Max(8, 2*e)
			m := boxAt(V(-W/2, -wd/2, 0), V(W/2, wd/2, H))
			cable := along(cylAt(0, 0, d*0.92, -wd, wd, 24), 1).Transformed(Translate(0, 0, 0))
			// small teeth that bite the jacket
			var cuts []*Mesh
			cuts = append(cuts, cable)
			for _, x := range []float64{-W/2 + e, W/2 - e} {
				cuts = append(cuts, cylAt(x, 0, hd, -1, H+1, 20))
			}
			m = Subtract(m, UnionAll(cuts))
			for _, y := range []float64{-wd / 4, wd / 4} {
				m = Union(m, Intersect(boxAt(V(-d/2, y-0.65, 0), V(d/2, y+0.65, d*0.46)), boxAt(V(-W/2, -wd/2, 0), V(W/2, wd/2, H))))
			}
			return m, nil
		})
	solid("dragchain", "Cable drag chain link", "Links clip together into a chain that guides cables on moving machines (like a 3D printer bed). Print as many as you need.",
		"drag chain energy chain cable carrier link",
		[]Param{num("w", "Inside width", 15, 5, 80, 0.5, "mm"), num("h", "Inside height", 10, 5, 60, 0.5, "mm")},
		func(p Params) (*Mesh, error) {
			w, h := p.F("w", 15), p.F("h", 10)
			t := 2.0
			H := h + 2*t
			L := H + 4 // link pitch
			pinD := math.Max(3, H*0.3)
			gap := 0.3
			yi := w/2 + t/2         // inner (male) plates
			yo := w/2 + 1.5*t + gap // outer (female) plates
			plate := func(x0, x1, y float64) *Mesh {
				return extrudeAt(Shape{Outer: moveLoop(stadium(x1-x0+H, H, 24), (x0+x1)/2, 0)}, -t/2, t/2).Transformed(RotX(90)).Transformed(Translate(0, y, H/2))
			}
			var parts []*Mesh
			for _, sg := range []float64{-1, 1} {
				parts = append(parts, plate(L*0.5, L, sg*yi), plate(0, L*0.5, sg*yo))
				// step joining the two halves
				parts = append(parts, boxAt(V(L*0.5-2, math.Min(sg*yi, sg*yo)-t/2, 0), V(L*0.5+2, math.Max(sg*yi, sg*yo)+t/2, H)))
				// pin on the outside of the male end
				pin := along(cylAt(0, 0, pinD, 0, t+gap+t-0.2, 20), 1)
				if sg < 0 {
					pin = pin.Transformed(RotZ(180))
				}
				parts = append(parts, pin.Transformed(Translate(L, sg*(yi+t/2), H/2)))
			}
			// cross bars top and bottom
			parts = append(parts, boxAt(V(L*0.3, -yi-t/2, 0), V(L*0.7, yi+t/2, t)), boxAt(V(L*0.3, -yi-t/2, H-t), V(L*0.7, yi+t/2, H)))
			m := UnionAll(parts)
			m = Subtract(m, along(cylAt(0, 0, pinD+0.4, -yo-t, yo+t, 20), 1).Transformed(Translate(0, 0, H/2)))
			m.Info = "Print several; snap each link's pin into the next link's holes."
			return m, nil
		})
}
