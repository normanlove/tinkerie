package geom

import (
	"fmt"
	"math"
)

// ===================== Hoses & tube fittings =====================
// Every fitting is hollow all the way through and is built from "arms" that share one
// set of end types, so a network can mix barbs, push-in sockets and spigots freely.

var hoseEnds = []string{"Hose barb", "Push-in socket", "Spigot (goes inside)"}

func sphereD(d float64, segs int) *Mesh {
	var prof []P2
	n := segs / 2
	if n < 8 {
		n = 8
	}
	for i := 0; i <= n; i++ {
		a := -math.Pi/2 + math.Pi*float64(i)/float64(n)
		prof = append(prof, P2{d / 2 * math.Cos(a), d / 2 * math.Sin(a)})
	}
	return RevolveProfile(prof, segs)
}

type hoseEnd struct {
	kind string
	size float64 // barb: hose inner Ø · socket: tube outer Ø · spigot: tube inner Ø
	tol  float64
	wall float64
	len  float64
	segs int
}

// outer diameter of the body next to the hub, and the flow bore
func (e hoseEnd) od() float64 {
	switch e.kind {
	case "Push-in socket":
		return e.size + e.tol + 2*e.wall
	default:
		return math.Max(e.size*1.15, e.bore()+2*e.wall)
	}
}

func (e hoseEnd) bore() float64 {
	switch e.kind {
	case "Push-in socket":
		return math.Max(1, e.size*0.66) // a little under a typical tube's inside
	case "Spigot (goes inside)":
		return math.Max(1, e.size-e.tol-2*e.wall)
	default: // barb: the shank is 0.85 × hose ID
		return math.Max(1, e.size*0.85-2*e.wall)
	}
}

// arm along +Z starting at z0 (inside the hub) — returns solid and its bore
func (e hoseEnd) arm(z0 float64) (*Mesh, *Mesh) {
	od := e.od()
	b := e.bore()
	segs := e.segs
	var m *Mesh
	L := e.len
	switch e.kind {
	case "Push-in socket":
		if L <= 0 {
			L = math.Max(10, e.size*0.9)
		}
		m = cylAt(0, 0, od, 0, z0+L, segs)
		sock := e.size + e.tol
		cut := Union(cylAt(0, 0, sock, z0, z0+L+1, segs), coneAt(sock, sock+1.2, z0+L-0.6, z0+L+0.01, segs)) // lead-in
		return Subtract(m, cut), cylAt(0, 0, b, -0.01, z0+1, segs)
	case "Spigot (goes inside)":
		if L <= 0 {
			L = math.Max(10, e.size*1.2)
		}
		d := e.size - e.tol
		m = Union(cylAt(0, 0, od, 0, z0, segs), cylAt(0, 0, d, z0-0.01, z0+L-1, segs))
		m = Union(m, coneAt(d, d-1.2, z0+L-1.01, z0+L, segs))
		return m, cylAt(0, 0, b, -0.01, z0+L+1, segs)
	default: // hose barb
		hd := e.size
		bl := math.Max(3, hd*0.55)
		n := 3
		if L > 0 {
			n = int(math.Max(1, math.Round(L/bl)))
		}
		shank := hd * 0.85
		m = cylAt(0, 0, od, 0, z0, segs)
		z := z0 - 0.01
		for i := 0; i < n; i++ {
			// each barb: a sharp step out to 1.12 × hose ID facing the hub, then a ramp down
			// towards the tip, so the hose slides on easily but won't pull off
			m = Union(m, coneAt(hd*1.12, shank, z, z+bl*0.85, segs))
			m = Union(m, cylAt(0, 0, shank, z+bl*0.85-0.01, z+bl, segs))
			z += bl
		}
		return m, cylAt(0, 0, b, -0.01, z+1, segs)
	}
}

type fitArm struct {
	dir Vec3
	end hoseEnd
}

// buildFitting puts arms round a hub (sphere sized to the biggest arm) and bores it through.
func buildFitting(arms []fitArm, segs int) *Mesh {
	maxOD := 0.0
	maxB := 0.0
	for _, a := range arms {
		maxOD = math.Max(maxOD, a.end.od())
		maxB = math.Max(maxB, a.end.bore())
	}
	hub := maxOD / 2
	var solids, bores []*Mesh
	if len(arms) > 2 || (len(arms) == 2 && arms[0].dir.Dot(arms[1].dir) > -0.99) {
		solids = append(solids, sphereD(maxOD, segs))
		bores = append(bores, sphereD(maxB, segs))
	}
	for _, a := range arms {
		s, b := a.end.arm(hub)
		R := rotBetweenG(V(0, 0, 1), a.dir.Norm())
		solids = append(solids, s.Transformed(R))
		bores = append(bores, b.Transformed(R))
	}
	return Subtract(UnionAll(solids), UnionAll(bores))
}

func hoseParams(extra ...Param) []Param {
	ps := []Param{
		sel("end", "Ends", "Hose barb", hoseEnds...).help("Hose barb: push a soft hose over it. Push-in socket: a rigid tube slides in. Spigot: slides inside a tube or hose."),
		num("size", "Hose / tube size", 8, 2, 100, 0.5, "mm").help("Barb: the hose's inside Ø. Socket: the tube's outside Ø. Spigot: the tube's inside Ø."),
		num("wall", "Wall", 1.6, 1.2, 6, 0.1, "mm"),
		num("tol", "Fit clearance", 0.25, 0, 1.5, 0.05, "mm").show("end!=Hose barb"),
	}
	return append(ps, extra...)
}

func endFrom(p Params, kindKey, sizeKey string) hoseEnd {
	return hoseEnd{kind: p.S(kindKey, "Hose barb"), size: p.F(sizeKey, 8), tol: p.F("tol", 0.25), wall: math.Max(MinWall, p.F("wall", 1.6)), segs: p.Segs(p.F(sizeKey, 8) * 1.4)}
}

func init() {
	hose := func(typ, name, desc string, params []Param, gen func(p Params) (*Mesh, error)) {
		register(&ShapeDef{Type: typ, Name: name, Category: "Hoses", Desc: desc, Tags: "hose tube pipe fitting connector plumbing air water", Color: "#0ea5e9", Params: params, Gen: gen})
	}
	simple := func(typ, name, desc string, dirs func(p Params) []Vec3, extra ...Param) {
		hose(typ, name, desc, hoseParams(extra...), func(p Params) (*Mesh, error) {
			e := endFrom(p, "end", "size")
			var arms []fitArm
			for _, d := range dirs(p) {
				arms = append(arms, fitArm{d, e})
			}
			return buildFitting(arms, e.segs), nil
		})
	}
	X, Y, Z := V(1, 0, 0), V(0, 1, 0), V(0, 0, 1)
	simple("hf_coupler", "Straight coupler", "Joins two hoses or tubes end to end.", func(p Params) []Vec3 { return []Vec3{Z, Z.Neg()} })
	simple("hf_elbow", "Elbow", "Turns a corner — any angle.", func(p Params) []Vec3 {
		a := p.F("angle", 90) * math.Pi / 180
		return []Vec3{X.Neg(), V(math.Cos(a), math.Sin(a), 0)}
	}, num("angle", "Turn", 90, 15, 165, 5, "°"))
	simple("hf_tee", "Tee", "One hose in, two out (or the other way round).", func(p Params) []Vec3 { return []Vec3{X, X.Neg(), Y} })
	simple("hf_y", "Y splitter", "Splits a flow gently in two — smoother than a tee.", func(p Params) []Vec3 {
		return []Vec3{X.Neg(), V(math.Cos(0.5), math.Sin(0.5), 0), V(math.Cos(0.5), -math.Sin(0.5), 0)}
	})
	simple("hf_cross", "Cross", "Four-way junction.", func(p Params) []Vec3 { return []Vec3{X, X.Neg(), Y, Y.Neg()} })
	simple("hf_corner", "3-way corner", "Three arms at right angles — for frames and manifolds.", func(p Params) []Vec3 { return []Vec3{X, Y, Z} })
	simple("hf_5way", "5-way hub", "Four arms round a hub plus one up.", func(p Params) []Vec3 { return []Vec3{X, X.Neg(), Y, Y.Neg(), Z} })
	simple("hf_6way", "6-way hub", "Arms in every direction.", func(p Params) []Vec3 { return []Vec3{X, X.Neg(), Y, Y.Neg(), Z, Z.Neg()} })

	hose("hf_adapter", "Adapter", "Changes one kind of end to another — barb to socket, socket to spigot…", hoseParams(
		sel("end2", "Other end", "Push-in socket", hoseEnds...), num("size2", "Other end size", 8, 2, 100, 0.5, "mm")),
		func(p Params) (*Mesh, error) {
			a, b := endFrom(p, "end", "size"), endFrom(p, "end2", "size2")
			return buildFitting([]fitArm{{Z.Neg(), a}, {Z, b}}, a.segs), nil
		})
	hose("hf_reducer", "Reducer", "Joins a big hose to a small one.", hoseParams(num("size2", "Small end size", 5, 2, 100, 0.5, "mm")),
		func(p Params) (*Mesh, error) {
			a, b := endFrom(p, "end", "size"), endFrom(p, "end", "size2")
			return buildFitting([]fitArm{{Z.Neg(), a}, {Z, b}}, a.segs), nil
		})
	hose("hf_redtee", "Reducing tee", "Tee with a smaller side branch.", hoseParams(num("size2", "Branch size", 5, 2, 100, 0.5, "mm")),
		func(p Params) (*Mesh, error) {
			a, b := endFrom(p, "end", "size"), endFrom(p, "end", "size2")
			return buildFitting([]fitArm{{X, a}, {X.Neg(), a}, {Y, b}}, a.segs), nil
		})
	hose("hf_manifold", "Manifold", "One inlet feeding a row of outlets.", hoseParams(num("n", "Outlets", 4, 2, 12, 1, ""), num("size2", "Outlet size", 5, 2, 100, 0.5, "mm"), num("pitch", "Outlet spacing", 0, 0, 200, 1, "mm").help("0 = as close as they fit")),
		func(p Params) (*Mesh, error) {
			a, b := endFrom(p, "end", "size"), endFrom(p, "end", "size2")
			n := p.I("n", 4)
			pitch := p.F("pitch", 0)
			if pitch < b.od()+3 {
				pitch = b.od() + 3
			}
			body := a.od()
			if body < b.bore()+2*MinScrewWall {
				body = b.bore() + 2*MinScrewWall
			}
			L := pitch * float64(n)
			bar := along(cylAt(0, 0, body, -pitch/2, L-pitch/2, a.segs), 0)
			bore := along(cylAt(0, 0, math.Min(a.bore(), body-2*MinWall), -pitch/2+MinWall, L+5, a.segs), 0)
			// inlet on the -X end
			ia, ib := a.arm(0)
			inlet := ia.Transformed(rotBetweenG(Z, X.Neg())).Transformed(Translate(-pitch/2, 0, 0))
			inB := ib.Transformed(rotBetweenG(Z, X.Neg())).Transformed(Translate(-pitch/2, 0, 0))
			solids := []*Mesh{bar, inlet}
			bores := []*Mesh{bore, inB}
			for i := 0; i < n; i++ {
				x := pitch * float64(i)
				s, bb := b.arm(body / 2)
				solids = append(solids, s.Transformed(Translate(x, 0, 0)))
				bores = append(bores, bb.Transformed(Translate(x, 0, 0)))
			}
			// closed far end
			m := Subtract(UnionAll(solids), UnionAll(bores))
			return Union(m, along(cylAt(0, 0, body, L-pitch/2-MinWall-0.01, L-pitch/2, a.segs), 0)), nil
		})
	hose("hf_bend", "Swept bend", "Long smooth bend — less flow loss than an elbow, and prints without supports standing up.", hoseParams(num("angle", "Bend", 90, 10, 180, 5, "°"), num("R", "Bend radius", 0, 0, 500, 1, "mm").help("0 = 2 × the tube size")),
		func(p Params) (*Mesh, error) {
			e := endFrom(p, "end", "size")
			od, b := e.od(), e.bore()
			R := p.F("R", 0)
			if R < od*0.75+1 {
				R = math.Max(2*e.size, od*0.75+1)
			}
			ang := p.F("angle", 90)
			ring := Shape{Outer: moveLoop(Circle(od/2, e.segs), R, 0), Holes: []Loop{moveLoop(Circle(b/2, e.segs), R, 0)}}
			bend := Revolve(ring, ang, int(math.Max(24, ang/4)))
			a := ang * math.Pi / 180
			s1, b1 := e.arm(0.01)
			end1 := Subtract(s1, b1).Transformed(rotBetweenG(Z, Y.Neg())).Transformed(Translate(R, 0, 0))
			end2 := Subtract(s1, b1).Transformed(rotBetweenG(Z, V(-math.Sin(a), math.Cos(a), 0))).Transformed(Translate(R*math.Cos(a), R*math.Sin(a), 0))
			return UnionAll([]*Mesh{bend, end1, end2}), nil
		})
	hose("hf_cap", "End cap", "Closes off a hose or tube end.", hoseParams(),
		func(p Params) (*Mesh, error) {
			e := endFrom(p, "end", "size")
			s, b := e.arm(MinScrewWall)
			return Subtract(s, Subtract(b, cylAt(0, 0, e.od()+2, -2, MinScrewWall, e.segs))), nil
		})
	hose("hf_bulkhead", "Panel bulkhead", "Passes a hose through a wall or box side, held by a printed nut.",
		hoseParams(num("panel", "Panel thickness", 3, 0.5, 20, 0.5, "mm"), sel("part", "Show", "Fitting", "Fitting", "Nut", "Both")),
		func(p Params) (*Mesh, error) {
			e := endFrom(p, "end", "size")
			th := math.Max(e.od()+4, 10)
			pitch := 2.0
			pl := p.F("panel", 3)
			tl := pl + 8
			flange := th + 8
			b := e.bore()
			mk := func() *Mesh {
				body := Union(cylAt(0, 0, flange, 0, 3, e.segs*2), Thread(ThreadSpec{Major: th, Pitch: pitch, Profile: "Trapezoidal 30°", Starts: 1, Axial: -0.15, Chamfer: true}, 2.99, 3+tl, p.Q()))
				s1, b1 := e.arm(0.01)
				top := Subtract(s1, b1).Transformed(Translate(0, 0, 3+tl-0.02))
				bot := Subtract(s1, b1).Transformed(RotX(180))
				m := Union(Union(body, top), bot)
				return Subtract(m, cylAt(0, 0, b, -100, 100, e.segs))
			}
			nut := func() *Mesh {
				cut := Thread(ThreadSpec{Major: th + 0.6, Pitch: pitch, Profile: "Trapezoidal 30°", Starts: 1, Axial: 0.2}, -1, 7, p.Q())
				// 45° countersinks on both faces remove the feather-edge start of the thread
				minor := th + 0.6 - pitch - 0.4
				cs := (th + 1.4 - minor) / 2
				cut = UnionAll([]*Mesh{cut, coneAt(th+1.4, minor, -0.01, cs, e.segs*2), coneAt(minor, th+1.4, 6-cs, 6.01, e.segs*2)})
				return Subtract(cylAt(0, 0, flange, 0, 6, 6), cut)
			}
			switch p.S("part", "Fitting") {
			case "Nut":
				return nut(), nil
			case "Both":
				return Union(mk(), nut().Transformed(Translate(flange+6, 0, 0))), nil
			}
			m := mk()
			m.Info = fmt.Sprintf("Drill the panel %.0f mm. Print the nut too (Show → Nut).", math.Ceil(th+1))
			return m, nil
		})
	hose("hf_valve", "Tap / valve", "Quarter-turn plug valve: turn the handle to open or close. Print the body and the plug separately.",
		hoseParams(sel("part", "Show", "Assembled", "Assembled", "Body", "Plug")),
		func(p Params) (*Mesh, error) {
			e := endFrom(p, "end", "size")
			b := e.bore()
			cd0, cd1 := b+2*MinScrewWall+2, b+2*MinScrewWall+5 // tapered plug (top wider)
			bodyD := cd1 + 2*2.4
			bh := cd0 + 6
			plug := func(clear float64) *Mesh {
				pl := coneAt(cd0-clear, cd1-clear, -bh/2, bh/2, e.segs*2)
				pl = Union(pl, cylAt(0, 0, cd1-clear, bh/2-0.01, bh/2+3, e.segs*2))
				handle := boxAt(V(-2, -(bodyD/2+10), bh/2+3), V(2, bodyD/2+10, bh/2+10))
				pl = Union(pl, handle)
				pl = Union(pl, cylAt(0, 0, cd0*0.6, -bh/2-4, -bh/2+0.01, 20))
				pl = Union(pl, coneAt(cd0*0.6, cd0*0.6+1.2, -bh/2-4, -bh/2-3, 20)) // clip-in lip
				return Subtract(pl, along(cylAt(0, 0, b, -50, 50, e.segs), 1))
			}
			body := func() *Mesh {
				s := cylAt(0, 0, bodyD, -bh/2, bh/2, e.segs*2)
				var bores []*Mesh
				for _, d := range []Vec3{X, X.Neg()} {
					a, bb := e.arm(bodyD/2 - 1)
					R := rotBetweenG(Z, d)
					s = Union(s, a.Transformed(R))
					bores = append(bores, bb.Transformed(R))
				}
				s = Subtract(s, UnionAll(bores))
				s = Subtract(s, coneAt(cd0, cd1, -bh/2-0.01, bh/2+0.01, e.segs*2))
				return Subtract(s, cylAt(0, 0, cd0*0.6+0.4, -bh/2-1, -bh/2+1, 20))
			}
			switch p.S("part", "Assembled") {
			case "Body":
				return body().Transformed(Translate(0, 0, bh/2)), nil
			case "Plug":
				return plug(0.3).Transformed(RotX(180)).Transformed(Translate(0, 0, bh/2+10)), nil
			}
			m := Union(body(), plug(0.3))
			m.Info = "Handle along the pipe = open, across = closed. A little grease helps it seal."
			return m, nil
		})
	hose("hf_clamp", "Hose clamp", "Screw-tightened clamp to seal a hose onto a barb.", []Param{num("size", "Hose outside Ø", 12, 4, 80, 0.5, "mm"), sel("screw", "Screw", "M3", "M2.5", "M3", "M4")},
		func(p Params) (*Mesh, error) {
			hd := p.F("size", 12)
			mt := metrics[p.S("screw", "M3")]
			w := math.Max(math.Max(6, hd*0.4), 2*holeEdge(mt.Normal+0.2))
			t := 2.4
			r := hd/2 - 0.3 // slight squeeze
			band := Subtract(cylAt(0, 0, 2*(r+t), 0, w, p.Segs(hd*2)), cylAt(0, 0, 2*r, -1, w+1, p.Segs(hd*2)))
			gap := 2.0
			band = Subtract(band, boxAt(V(r-1, -gap/2, -1), V(r+t+1, gap/2, w+1)))
			hdia := mt.Normal + 0.2
			e := holeEdge(hdia)
			ear := 2 * e
			ears := Union(boxAt(V(r, gap/2, 0), V(r+t+ear, gap/2+3.2, w)), boxAt(V(r, -gap/2-3.2, 0), V(r+t+ear, -gap/2, w)))
			m := Union(band, ears)
			hole := along(cylAt(0, 0, hdia, -20, 20, 20), 1).Transformed(Translate(r+t+ear-e, 0, w/2))
			return Subtract(m, hole), nil
		})
	hose("hf_holder", "Hose holder clip", "Screw-down clip that holds one or more hoses along a wall.", []Param{num("size", "Hose outside Ø", 10, 3, 60, 0.5, "mm"), num("n", "Hoses", 1, 1, 8, 1, ""), sel("screw", "Screw", "M3", "M3", "M4")},
		func(p Params) (*Mesh, error) {
			hd := p.F("size", 10)
			n := p.I("n", 1)
			mt := metrics[p.S("screw", "M3")]
			hdia := mt.Normal + 0.2
			e := holeEdge(hdia)
			t := 2.0
			w := 8.0
			pitch := hd + 2*t + 1
			L := float64(n)*pitch + 4*e
			base := boxAt(V(-L/2, -w/2, 0), V(L/2, w/2, t))
			var parts []*Mesh
			parts = append(parts, base)
			var cuts []*Mesh
			for i := 0; i < n; i++ {
				x := -float64(n)*pitch/2 + pitch*(float64(i)+0.5)
				c := along(Subtract(cylAt(0, 0, hd+2*t, -w/2, w/2, p.Segs(hd*2)), cylAt(0, 0, hd-0.3, -w, w, p.Segs(hd*2))), 1).Transformed(Translate(x, 0, hd/2+t-0.3))
				parts = append(parts, c)
				// opening at the top (snap in) — 80 % of the hose Ø
				cuts = append(cuts, boxAt(V(x-hd*0.4, -w, hd/2+t), V(x+hd*0.4, w, hd+3*t)))
			}
			for _, x := range []float64{-L/2 + e, L/2 - e} {
				cuts = append(cuts, cylAt(x, 0, hdia, -1, t+1, 20), coneAt(hdia, hdia+3, t-1.2, t+0.01, 20).Transformed(Translate(x, 0, 0)))
			}
			return Subtract(UnionAll(parts), UnionAll(cuts)), nil
		})
}
