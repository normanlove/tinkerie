package geom

import (
	"fmt"
	"math"
)

// ===================== More gears =====================

// gear-family parts get their own library section
var gearTypes = map[string]bool{"gear": true, "rack": true, "bevelgear": true, "ringgear": true, "worm": true, "sprocket": true, "ratchet": true, "geneva": true, "gt2pulley": true}

// ---------- non-circular gears (nautilus, elliptical) ----------
// Teeth are laid along any closed pitch curve: a rack-like tooth every π·m of arc,
// pushed out along the curve's normal. mate=true shifts the teeth by half a tooth.
func ncGearOutline(pitch []P2, teeth int, mate bool, toothed func(i int) bool) (Loop, float64) {
	n := len(pitch)
	acc := make([]float64, n+1)
	for i := 0; i < n; i++ {
		a, b := pitch[i], pitch[(i+1)%n]
		acc[i+1] = acc[i] + math.Hypot(b.X-a.X, b.Y-a.Y)
	}
	L := acc[n]
	var tl float64 // length that carries teeth
	for i := 0; i < n; i++ {
		if toothed == nil || toothed(i) {
			tl += acc[i+1] - acc[i]
		}
	}
	m := tl / (math.Pi * float64(teeth))
	pp := math.Pi * m
	var out Loop
	steps := teeth * 28
	_ = L
	// walk the toothed length uniformly
	s := 0.0
	for i := 0; i < n; i++ {
		a, b := pitch[i], pitch[(i+1)%n]
		seg := acc[i+1] - acc[i]
		if seg < 1e-9 {
			continue
		}
		tx, ty := (b.X-a.X)/seg, (b.Y-a.Y)/seg
		nx, ny := ty, -tx // outward for a CCW curve
		k := int(math.Max(1, math.Ceil(seg/(tl/float64(steps)))))
		for j := 0; j < k; j++ {
			f := float64(j) / float64(k)
			px, py := a.X+(b.X-a.X)*f, a.Y+(b.Y-a.Y)*f
			off := 0.0
			if toothed == nil || toothed(i) {
				u := (s + seg*f) / pp
				if mate {
					u += 0.5
				}
				off = m * math.Max(-1.25, math.Min(1.0, 1.7*math.Cos(2*math.Pi*u)))
			}
			out = append(out, P2{px + nx*off, py + ny*off})
		}
		if toothed == nil || toothed(i) {
			s += seg
		}
	}
	return out, m
}

func init() {
	gear := func(typ, name, desc, tags string, params []Param, gen func(p Params) (*Mesh, error)) {
		register(&ShapeDef{Type: typ, Name: name, Category: "Gears", Desc: desc, Tags: tags + " gear", Color: "#0ea5e9", Params: params, Gen: gen})
	}
	boreP := func(def float64) Param { return num("bore", "Bore Ø", def, 0, 200, 0.1, "mm") }

	gear("nautilus", "Nautilus gear", "Spiral gear — print two: the speed rises and falls every turn. Two identical gears mesh with centres (min + max radius) apart.",
		"nautilus spiral non-circular variable ratio",
		[]Param{num("rmin", "Smallest radius", 15, 5, 200, 0.5, "mm"), num("rmax", "Largest radius", 30, 8, 300, 0.5, "mm"), num("teeth", "Teeth", 30, 8, 200, 1, ""), num("h", "Thickness", 6, 1, 100, 0.5, "mm"), boreP(5), boolp("mate", "Mating half (shift teeth)", false)},
		func(p Params) (*Mesh, error) {
			r0, r1 := p.F("rmin", 15), p.F("rmax", 30)
			if r1 <= r0*1.1 {
				return nil, fmt.Errorf("the largest radius must be bigger than the smallest")
			}
			k := math.Log(r1/r0) / (2 * math.Pi)
			var pc []P2
			n := 360
			end := 2*math.Pi - 2.5*(r1-r0)/r1*0.35 // stop short of the step so the lip isn't a knife edge
			for i := 0; i <= n; i++ {
				a := end * float64(i) / float64(n)
				r := r0 * math.Exp(k*a)
				pc = append(pc, P2{r * math.Cos(a), r * math.Sin(a)})
			}
			// the step back from the big end to the small end carries no teeth
			l, m := ncGearOutline(pc, p.I("teeth", 30), p.B("mate", false), func(i int) bool { return i < n })
			g := Extrude(Shape{Outer: l}, p.F("h", 6), 0, 1, 1)
			g = boreFor(g, p.F("bore", 5), -1, p.F("h", 6)+1, p.Segs(8))
			g.Info = fmt.Sprintf("Module ≈ %.2f mm. Mesh two of them %.1f mm apart (tick “Mating half” on one).", m, r0+r1)
			return g, nil
		})
	gear("ellgear", "Elliptical gear", "Oval gear turning about a focus — makes a smooth speed-up / slow-down. Two identical gears mesh 2 × semi-major axis apart.",
		"elliptical oval non-circular variable speed",
		[]Param{num("a", "Semi-major axis", 25, 8, 200, 0.5, "mm"), num("e", "Eccentricity", 0.3, 0.05, 0.6, 0.01, ""), num("teeth", "Teeth", 30, 10, 200, 1, ""), num("h", "Thickness", 6, 1, 100, 0.5, "mm"), boreP(5), boolp("mate", "Mating half (shift teeth)", false)},
		func(p Params) (*Mesh, error) {
			a, e := p.F("a", 25), p.F("e", 0.3)
			var pc []P2
			n := 360
			for i := 0; i < n; i++ {
				t := 2 * math.Pi * float64(i) / float64(n)
				r := a * (1 - e*e) / (1 + e*math.Cos(t))
				pc = append(pc, P2{r * math.Cos(t), r * math.Sin(t)})
			}
			l, m := ncGearOutline(pc, p.I("teeth", 30), p.B("mate", false), nil)
			g := Extrude(Shape{Outer: l}, p.F("h", 6), 0, 1, 1)
			g = boreFor(g, p.F("bore", 5), -1, p.F("h", 6)+1, p.Segs(8))
			g.Info = fmt.Sprintf("Module ≈ %.2f mm. Mesh two of them %.1f mm apart, the second turned 180° (tick “Mating half”).", m, 2*a)
			return g, nil
		})

	gear("lantern", "Lantern pinion", "Pins held between two discs — the clockmaker's pinion. Very easy to print and meshes with a spur gear of the same module.",
		"lantern pinion clock trundle pin",
		[]Param{num("module", "Module", 2, 0.5, 10, 0.1, "mm"), num("teeth", "Pins", 8, 5, 40, 1, ""), num("h", "Pin length", 8, 2, 100, 0.5, "mm"), num("flange", "Flange thickness", 2, 1.2, 10, 0.2, "mm"), boreP(5)},
		func(p Params) (*Mesh, error) {
			m, z, h, f := p.F("module", 2), p.I("teeth", 8), p.F("h", 8), p.F("flange", 2)
			rp := m * float64(z) / 2
			pd := math.Max(1.6, 1.6*m)
			od := 2*rp + pd + 2*m
			g := Union(cylAt(0, 0, od, 0, f, p.Segs(od)), cylAt(0, 0, od, f+h, 2*f+h, p.Segs(od)))
			for i := 0; i < z; i++ {
				a := 2 * math.Pi * float64(i) / float64(z)
				g = Union(g, cylAt(rp*math.Cos(a), rp*math.Sin(a), pd, f-0.01, f+h+0.01, 20))
			}
			hub := 2*(rp-pd/2) - 1
			if hub > 0 {
				g = Union(g, cylAt(0, 0, math.Min(hub, p.F("bore", 5)+2*screwWall(p.F("bore", 5))), f-0.01, f+h+0.01, p.Segs(hub)))
			}
			return boreFor(g, p.F("bore", 5), -1, 2*f+h+1, p.Segs(8)), nil
		})

	gear("crowngear", "Crown gear", "Face gear with teeth on its top — turns motion 90° with an ordinary spur gear of the same module.",
		"crown face contrate right angle",
		[]Param{num("module", "Module", 1.5, 0.5, 8, 0.1, "mm"), num("teeth", "Teeth", 40, 12, 200, 1, ""), num("face", "Tooth length", 8, 2, 60, 0.5, "mm"), num("base", "Disc thickness", 4, 1.2, 40, 0.2, "mm"), boreP(6)},
		func(p Params) (*Mesh, error) {
			m, z, fw, base := p.F("module", 1.5), p.I("teeth", 40), p.F("face", 8), p.F("base", 4)
			rp := m * float64(z) / 2
			ri, ro := rp-fw/2, rp+fw/2
			if ri < 2 {
				return nil, fmt.Errorf("teeth are too long for that gear — shorten them")
			}
			disk := cylAt(0, 0, 2*ro, 0, base, p.Segs(2*ro))
			var teeth []*Mesh
			tn := math.Tan(20 * math.Pi / 180)
			for i := 0; i < z; i++ {
				c := 2 * math.Pi * float64(i) / float64(z)
				ring := func(r float64) []Vec3 {
					p := math.Pi * m * r / rp
					hw := p / 4
					add, ded := m, 1.25*m
					loc := [][2]float64{{-hw - ded*tn, 0}, {hw + ded*tn, 0}, {hw - add*tn, ded + add}, {-hw + add*tn, ded + add}}
					var out []Vec3
					for _, q := range loc {
						a := c + q[0]/r
						out = append(out, V(r*math.Cos(a), r*math.Sin(a), base-0.01+q[1]))
					}
					return out
				}
				teeth = append(teeth, Loft([][]Vec3{ring(ri), ring(ro)}, true, true))
			}
			g := Union(disk, UnionAll(teeth))
			return boreFor(g, p.F("bore", 6), -1, base+3*m, p.Segs(8)), nil
		})

	gear("compoundgear", "Compound gear", "Two gears on one hub — the building block of a gear train (multiplies the ratio).",
		"compound double cluster gear train reduction",
		[]Param{num("module", "Module", 1.5, 0.3, 10, 0.1, "mm"), num("teeth", "Big gear teeth", 36, 8, 200, 1, ""), num("teeth2", "Small gear teeth", 12, 6, 200, 1, ""), num("h", "Big gear thickness", 5, 1, 100, 0.5, "mm"), num("h2", "Small gear thickness", 6, 1, 100, 0.5, "mm"), boreP(5)},
		func(p Params) (*Mesh, error) {
			m := p.F("module", 1.5)
			h, h2 := p.F("h", 5), p.F("h2", 6)
			a := Extrude(Shape{Outer: GearOutline(m, p.I("teeth", 36), 20, 0.15, p.Q())}, h, 0, 1, 1)
			b := Extrude(Shape{Outer: GearOutline(m, p.I("teeth2", 12), 20, 0.15, p.Q())}, h2, 0, 1, 1).Transformed(Translate(0, 0, h-0.01))
			g := boreFor(Union(a, b), p.F("bore", 5), -1, h+h2+1, p.Segs(8))
			g.Info = fmt.Sprintf("Ratio %d:%d = %.3g : 1", p.I("teeth", 36), p.I("teeth2", 12), float64(p.I("teeth", 36))/float64(p.I("teeth2", 12)))
			return g, nil
		})
	gear("sectorgear", "Sector gear", "Just a slice of a gear on an arm — for levers, steering and wipers.",
		"sector segment partial quadrant gear lever",
		[]Param{num("module", "Module", 1.5, 0.3, 10, 0.1, "mm"), num("teeth", "Teeth (full circle)", 40, 12, 300, 1, ""), num("angle", "Sector angle", 90, 15, 300, 5, "°"), num("h", "Thickness", 5, 1, 100, 0.5, "mm"), boreP(5)},
		func(p Params) (*Mesh, error) {
			m, z := p.F("module", 1.5), p.I("teeth", 40)
			ang := p.F("angle", 90)
			rp := m * float64(z) / 2
			h := p.F("h", 5)
			g := Extrude(Shape{Outer: GearOutline(m, z, 20, 0.15, p.Q())}, h, 0, 1, 1)
			pie := append(Loop{{0, 0}}, arcLoop(0, 0, rp+3*m, 90-ang/2, 90+ang/2)...)
			g = Intersect(g, extrudeAt(Shape{Outer: pie}, -1, h+1))
			bore := p.F("bore", 5)
			hub := math.Max(bore+2*screwWall(bore), 10)
			g = Union(g, cylAt(0, 0, hub, 0, h, p.Segs(hub)))
			return boreFor(g, bore, -1, h+1, p.Segs(8)), nil
		})
	gear("intermittent", "Intermittent gear", "A spur gear with some teeth missing — the driven gear moves, then pauses, every turn.",
		"intermittent mutilated partial dwell indexing",
		[]Param{num("module", "Module", 1.5, 0.3, 10, 0.1, "mm"), num("teeth", "Teeth (full circle)", 24, 8, 200, 1, ""), num("keep", "Teeth kept", 6, 1, 199, 1, ""), num("h", "Thickness", 6, 1, 100, 0.5, "mm"), boreP(5)},
		func(p Params) (*Mesh, error) {
			m, z := p.F("module", 1.5), p.I("teeth", 24)
			keep := math.Min(p.F("keep", 6), float64(z-1))
			l := GearOutline(m, z, 20, 0.15, p.Q())
			rf := m*float64(z)/2 - 1.25*m
			lim := 2 * math.Pi * (keep - 0.5) / float64(z)
			for i, q := range l {
				a := math.Atan2(q.Y, q.X)
				if a < -math.Pi/float64(z) {
					a += 2 * math.Pi
				}
				if a > lim {
					r := math.Min(math.Hypot(q.X, q.Y), rf)
					l[i] = P2{r * math.Cos(a), r * math.Sin(a)}
				}
			}
			g := Extrude(Shape{Outer: l.Clean()}, p.F("h", 6), 0, 1, 1)
			return boreFor(g, p.F("bore", 5), -1, p.F("h", 6)+1, p.Segs(8)), nil
		})

	gear("cycloidal", "Cycloidal drive", "Cycloidal reducer: a lobed disc rolling inside a ring of pins. Huge reduction (pins − 1 : 1) in a flat package.",
		"cycloidal reducer drive disc pins high ratio robot",
		[]Param{num("R", "Pin circle radius", 30, 10, 200, 0.5, "mm"), num("pins", "Pins", 12, 6, 60, 1, ""), num("pin", "Pin Ø", 5, 2, 20, 0.5, "mm"), num("E", "Eccentricity", 1.2, 0.3, 5, 0.1, "mm"), num("h", "Thickness", 6, 2, 60, 0.5, "mm"), num("bore", "Centre bore (bearing)", 16, 3, 100, 0.5, "mm"), sel("part", "Show", "Both", "Both", "Disc", "Pin ring")},
		func(p Params) (*Mesh, error) {
			R, N, Rr, E := p.F("R", 30), p.I("pins", 12), p.F("pin", 5)/2, p.F("E", 1.2)
			h := p.F("h", 6)
			if E >= Rr*0.9 || R/(E*float64(N)) <= 1 {
				return nil, fmt.Errorf("eccentricity too big for those pins")
			}
			disc := func() *Mesh {
				var l Loop
				n := N * 40
				for i := 0; i < n; i++ {
					t := 2 * math.Pi * float64(i) / float64(n)
					psi := math.Atan2(math.Sin(float64(1-N)*t), R/(E*float64(N))-math.Cos(float64(1-N)*t))
					x := R*math.Cos(t) - Rr*math.Cos(t+psi) - E*math.Cos(float64(N)*t)
					y := -R*math.Sin(t) + Rr*math.Sin(t+psi) + E*math.Sin(float64(N)*t)
					l = append(l, P2{x, y})
				}
				s := Shape{Outer: l}
				g := Extrude(s, h, 0, 1, 1)
				g = boreFor(g, p.F("bore", 16), -1, h+1, p.Segs(p.F("bore", 16)))
				// output pin holes
				oh := (R - Rr - p.F("bore", 16)/2) / 2
				orad := p.F("bore", 16)/2 + oh + 1
				for k := 0; k < 4; k++ {
					a := math.Pi / 2 * float64(k)
					hd := math.Max(3, oh*0.9) + 2*E
					if orad+hd/2 < R-Rr-E-MinScrewWall {
						g = Subtract(g, cylAt(orad*math.Cos(a), orad*math.Sin(a), hd, -1, h+1, 24))
					}
				}
				return g
			}
			ring := func() *Mesh {
				od := 2*(R+Rr) + 2*4
				base := ring(od, 2*(R-Rr)-2, 0, 2.4, p.Segs(od))
				g := Union(base, ring(od, 2*(R+Rr)+0.6, 2.39, 2.4+h+0.4, p.Segs(od)))
				for i := 0; i < N; i++ {
					a := 2 * math.Pi * float64(i) / float64(N)
					g = Union(g, cylAt(R*math.Cos(a), R*math.Sin(a), 2*Rr, 2.39, 2.4+h+0.4, 20))
				}
				return g
			}
			switch p.S("part", "Both") {
			case "Disc":
				return disc(), nil
			case "Pin ring":
				return ring(), nil
			}
			m := Union(ring(), disc().Transformed(Translate(E, 0, 2.6)))
			m.Info = fmt.Sprintf("Reduction %d : 1 (disc has %d lobes). Drive the centre bore with an eccentric cam of %.1f mm offset.", N-1, N-1, E)
			return m, nil
		})

	gear("planetary", "Planetary gear set", "Sun, planets and ring that fit together — a compact high-ratio gearbox. Printed as separate parts in their meshing positions.",
		"planetary epicyclic sun planet ring gearbox",
		[]Param{num("module", "Module", 1.5, 0.5, 6, 0.1, "mm"), num("sun", "Sun teeth", 12, 6, 80, 1, ""), num("planet", "Planet teeth", 18, 6, 80, 1, ""), num("n", "Planets", 3, 2, 8, 1, ""), num("h", "Thickness", 8, 2, 60, 0.5, "mm"), num("bore", "Sun bore", 5, 0, 50, 0.1, "mm"), num("pbore", "Planet bore", 3.4, 0, 30, 0.1, "mm")},
		func(p Params) (*Mesh, error) {
			m, zs, zp, n := p.F("module", 1.5), p.I("sun", 12), p.I("planet", 18), p.I("n", 3)
			zr := zs + 2*zp
			if (zs+zr)%n != 0 {
				return nil, fmt.Errorf("planets won't space evenly: (sun + ring) = %d must divide by %d — try %d sun teeth", zs+zr, n, zs+1)
			}
			h := p.F("h", 8)
			bl := 0.2
			sunL := GearOutline(m, zs, 20, bl, p.Q())
			sun := boreFor(Extrude(Shape{Outer: sunL}, h, 0, 1, 1), p.F("bore", 5), -1, h+1, 24)
			parts := []*Mesh{sun}
			rc := m * float64(zs+zp) / 2
			planetL := GearOutline(m, zp, 20, bl, p.Q())
			rot := func(l Loop, a float64, dx, dy float64) Loop {
				c, s := math.Cos(a), math.Sin(a)
				out := make(Loop, len(l))
				for i, q := range l {
					out[i] = P2{q.X*c - q.Y*s + dx, q.X*s + q.Y*c + dy}
				}
				return out
			}
			ringIn := GearOutline(m, zr, 20, -0.3, p.Q())
			var planets []Loop
			for k := 0; k < n; k++ {
				th := 2 * math.Pi * float64(k) / float64(n)
				cx, cy := rc*math.Cos(th), rc*math.Sin(th)
				// pick the planet rotation that clears the sun best
				best, bestA := math.Inf(1), 0.0
				for s := 0; s < 40; s++ {
					a := 2 * math.Pi / float64(zp) * float64(s) / 40
					pl := rot(planetL, a, cx, cy)
					bad := 0.0
					for _, q := range pl {
						if sunL.Contains(q) {
							bad++
						}
					}
					if bad < best {
						best, bestA = bad, a
					}
				}
				pl := rot(planetL, bestA, cx, cy)
				planets = append(planets, pl)
				pm := Extrude(Shape{Outer: pl}, h, 0, 1, 1)
				parts = append(parts, boreFor(pm.Transformed(Translate(-cx, -cy, 0)), p.F("pbore", 3.4), -1, h+1, 20).Transformed(Translate(cx, cy, 0)))
			}
			// ring rotation that clears the first planet
			best, bestA := math.Inf(1), 0.0
			for s := 0; s < 60; s++ {
				a := 2 * math.Pi / float64(zr) * float64(s) / 60
				rl := rot(ringIn, a, 0, 0)
				bad := 0.0
				for _, q := range planets[0] {
					if !rl.Contains(q) {
						bad++
					}
				}
				if bad < best {
					best, bestA = bad, a
				}
			}
			od := m*float64(zr) + 2*m + 2*4
			ringM := Extrude(Shape{Outer: Circle(od/2, p.Segs(od)), Holes: []Loop{rot(ringIn, bestA, 0, 0)}}, h, 0, 1, 1)
			parts = append(parts, ringM)
			g := UnionAll(parts)
			g.Info = fmt.Sprintf("Ring %d teeth. Ring fixed, sun in, carrier out: %.3g : 1.", zr, 1+float64(zr)/float64(zs))
			return g, nil
		})

	// ---------- ratchet (replaces the old one) ----------
	gear("ratchet", "Ratchet wheel + pawl", "Saw-tooth wheel that only turns one way, with a pawl. Pick a springy one-piece pawl or a pivoting pawl.",
		"ratchet pawl one way clutch freewheel",
		[]Param{num("d", "Wheel Ø", 40, 15, 300, 1, "mm"), num("teeth", "Teeth", 16, 6, 60, 1, ""), num("h", "Thickness", 6, 2, 50, 0.5, "mm"), boreP(5), sel("pawl", "Pawl", "Flexible (one piece)", "None", "Flexible (one piece)", "Pivoting")},
		func(p Params) (*Mesh, error) {
			d, z, h := p.F("d", 40), p.I("teeth", 16), p.F("h", 6)
			r1 := d / 2
			pitch := math.Pi * d / float64(z)
			depth := math.Max(1.2, math.Min(pitch*0.45, d*0.12))
			r0 := r1 - depth
			var l Loop
			for i := 0; i < z; i++ {
				a0 := 2 * math.Pi * float64(i) / float64(z)
				a1 := 2 * math.Pi * float64(i+1) / float64(z)
				// sloped back: straight line from the tooth tip down to the root
				p0 := P2{r1 * math.Cos(a0), r1 * math.Sin(a0)}
				p1 := P2{r0 * math.Cos(a1), r0 * math.Sin(a1)}
				for k := 0; k < 5; k++ {
					t := float64(k) / 5
					l = append(l, P2{p0.X + (p1.X-p0.X)*t, p0.Y + (p1.Y-p0.Y)*t})
				}
				l = append(l, p1) // then the radial face back up to the next tip
			}
			wheel := boreFor(Extrude(Shape{Outer: l}, h, 0, 1, 1), p.F("bore", 5), -1, h+1, 24)
			kind := p.S("pawl", "Flexible (one piece)")
			if kind == "None" {
				return wheel, nil
			}
			// pawl tip sits in the valley at the top (90°), pushing tangentially
			ta := math.Pi/2 - 0.02
			tip := P2{(r0 + 0.35) * math.Cos(ta), (r0 + 0.35) * math.Sin(ta)}
			L := d * 0.55
			pw := math.Max(4, depth*1.6)
			piv := P2{tip.X + L, tip.Y + depth + pw/2}
			// lever body (a slanted bar) with a pointed tip that matches the radial face
			body := Loop{{tip.X, tip.Y}, {tip.X + depth*0.9, tip.Y + depth + 0.1}, {piv.X, piv.Y + pw/2}, {piv.X, piv.Y - pw/2}, {tip.X + depth*1.4, tip.Y - 0.2}}
			pawl := extrudeAt(Shape{Outer: body}, 0, h)
			hole := 3.4
			hub := hole + 2*screwWall(hole)
			pawl = Union(pawl, cylAt(piv.X, piv.Y, hub, 0, h, 24))
			if kind == "Flexible (one piece)" {
				// a thin curved spring arm runs back over the wheel to a screw-down foot;
				// the pawl bends at it, so no separate pivot or spring is needed
				foot := P2{piv.X + 6, piv.Y + 10}
				var pts []Vec3
				for i := 0; i <= 20; i++ {
					t := float64(i) / 20
					x := piv.X + (foot.X-piv.X)*t + 4*math.Sin(math.Pi*t)
					y := piv.Y + (foot.Y-piv.Y)*t
					pts = append(pts, V(x, y, 0))
				}
				var frames []Frame
				for i := range pts {
					var t Vec3
					if i+1 < len(pts) {
						t = pts[i+1].Sub(pts[i])
					} else {
						t = pts[i].Sub(pts[i-1])
					}
					frames = append(frames, Frame{pts[i], V(0, 0, 1).Cross(t).Norm(), V(0, 0, 1)})
				}
				spring := Sweep(Shape{Outer: Loop{{-0.7, 0}, {0.7, 0}, {0.7, h}, {-0.7, h}}}, frames, false)
				fw := 2 * holeEdge(hole)
				footM := Subtract(boxAt(V(foot.X-fw/2, foot.Y, 0), V(foot.X+fw/2, foot.Y+fw, h)), cylAt(foot.X, foot.Y+fw/2, hole, -1, h+1, 20))
				pawl = UnionAll([]*Mesh{pawl, spring, footM})
			} else {
				pawl = Subtract(pawl, cylAt(piv.X, piv.Y, hole, -1, h+1, 20))
			}
			m := Union(wheel, pawl)
			m.Info = "Turns anticlockwise freely, locks clockwise. Screw the pawl's hub (and foot) down beside the wheel."
			return m, nil
		})
}
