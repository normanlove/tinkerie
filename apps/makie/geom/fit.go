package geom

import (
	"fmt"
	"math"
	"strings"
)

var fitModes = []string{"Mount plate", "Standoffs only", "Snap-in tray", "Slide-in rails", "Slide-in case", "Slide-in case end cap", "Case", "Case lid", "Front bezel", "Board model (fit check)", "Holder / cradle"}

var screwSizes = []string{"Auto (from board)", "M2", "M2.5", "M3", "M4"}
var fixings = []string{"Self-tapping screw", "Heat-set insert", "Through bolt + nut", "Snap pins (no screws)"}

func screwFor(p Params, hd float64) metric {
	s := p.S("screw", "Auto (from board)")
	if m, ok := metrics[s]; ok {
		return m
	}
	switch {
	case hd <= 2.3:
		return metrics["M2"]
	case hd <= 2.85:
		return metrics["M2.5"]
	case hd <= 3.7:
		return metrics["M3"]
	}
	return metrics["M4"]
}

func boardParams(defMode string, modes []string) []Param {
	ps := []Param{
		{Key: "board", Label: "Board / module", Type: "select", Def: "Raspberry Pi — Raspberry Pi 4 Model B", Options: BoardNames(), Help: "Typical published dimensions. Clone boards vary a little — check yours with calipers, use Custom, or import a 3D model and use “Make parts for this model”."},
		{Key: "spec", Label: "Scanned model", Type: "hidden", Def: ""},
	}
	if len(modes) > 0 {
		ps = append(ps, sel("mode", "Make", defMode, modes...))
	}
	ps = append(ps,
		num("cw", "Board width (X)", 60, 5, 500, 0.5, "mm").show("board=Custom — Custom board (type sizes)"),
		num("cd", "Board depth (Y)", 40, 5, 500, 0.5, "mm").show("board=Custom — Custom board (type sizes)"),
		num("ct", "Board thickness", 1.6, 0.4, 20, 0.1, "mm").show("board=Custom — Custom board (type sizes)"),
		num("choles", "Mounting holes (0, 2 or 4)", 4, 0, 4, 2, "").show("board=Custom — Custom board (type sizes)"),
		num("chd", "Hole Ø", 3.2, 1, 8, 0.1, "mm").show("board=Custom — Custom board (type sizes)"),
		num("cinset", "Hole distance from edges", 3.5, 1, 50, 0.5, "mm").show("board=Custom — Custom board (type sizes)"),
		num("cabove", "Tallest part above", 10, 0, 200, 0.5, "mm").show("board=Custom — Custom board (type sizes)"),
		num("cbelow", "Pins below", 3, 0, 50, 0.5, "mm").show("board=Custom — Custom board (type sizes)"),
		sel("screw", "Screw size", "Auto (from board)", screwSizes...).help("M2 / M2.5 / M3 / M4. Auto picks the size that fits the board's holes."),
		sel("fix", "Fixing", "Self-tapping screw", fixings...),
		num("standoff", "Standoff height (0 = auto)", 0, 0, 100, 0.5, "mm").help("Auto leaves room for pins and solder joints under the board."),
		num("tol", "Fit clearance", 0.4, 0, 3, 0.1, "mm"),
		num("wall", "Wall thickness", 2, 0.8, 10, 0.2, "mm"),
		num("margin", "Plate margin", 4, 0, 50, 0.5, "mm"),
		boolp("ports", "Cut openings for connectors", true),
		num("extra", "Extra case height", 2, 0, 100, 0.5, "mm"),
	)
	return ps
}

func standoffHeight(b *BoardSpec, p Params) float64 {
	if h := p.F("standoff", 0); h > 0 {
		return h
	}
	return math.Max(3, b.Below+1)
}

// standoff with the chosen fixing; base at z0, top at z1
func standoffAt(x, y float64, m metric, holeD float64, fix string, z0, z1, boardT float64, segs int) (*Mesh, *Mesh) {
	od := math.Max(holeD*2.2, m.D*2.4)
	floorZ := z0 + 1.6 // solid floor under blind holes
	if z1-z0 < 4 {
		floorZ = z0 - 1
	}
	var hole *Mesh
	switch fix {
	case "Heat-set insert":
		od = math.Max(od, bossOD(m.InsertD))
		hole = cylAt(x, y, m.InsertD, math.Max(floorZ, z1-m.InsertL-0.5), z1+1, segs)
	case "Through bolt + nut":
		od = math.Max(od, bossOD(m.Normal+0.2))
		hole = cylAt(x, y, m.Normal+0.2, z0-50, z1+1, segs)
	case "Snap pins (no screws)":
		// the two halves of a snap pin are meant to flex; keep them as thick as the hole allows
		pd := holeD - 0.35
		gap := math.Max(0.5, math.Min(0.9, pd*0.22))
		pin := Union(cylAt(x, y, pd, z1-0.01, z1+boardT+0.2, segs), coneAt(holeD+0.5, holeD-0.6, 0, 1.2, segs).Transformed(Translate(x, y, z1+boardT+0.2)))
		slot := boxAt(V(x-od, y-gap/2, z1+0.3), V(x+od, y+gap/2, z1+boardT+2))
		return Union(cylAt(x, y, od, z0, z1, segs), Subtract(pin, slot)), nil
	default:
		od = math.Max(od, bossOD(m.D*0.85))
		hole = cylAt(x, y, m.D*0.85, floorZ, z1+1, segs)
	}
	return cylAt(x, y, od, z0, z1, segs), hole
}

func outline2D(b *BoardSpec, grow float64, segs int) Loop {
	if b.Round {
		return Circle(b.W/2+grow, segs)
	}
	return RoundedRect(b.W+2*grow, b.D+2*grow, math.Max(0.5, grow+1), 4)
}

func plateMesh(b *BoardSpec, grow, z0, z1 float64, segs int) *Mesh {
	return extrudeAt(Shape{Outer: outline2D(b, grow, segs)}, z0, z1)
}

// FitMesh builds one kind of part for a board spec.
func FitMesh(b *BoardSpec, mode string, p Params) *Mesh {
	segs := p.Segs(12)
	tol, wall, margin := p.F("tol", 0.4), p.F("wall", 2), p.F("margin", 4)
	fix := p.S("fix", "Self-tapping screw")
	hs := standoffHeight(b, p)
	floor := 2.0
	m := screwFor(p, 2.7)
	if len(b.Holes) > 0 {
		m = screwFor(p, b.Holes[0].D)
	}
	addStandoffs := func(base *Mesh, z0, z1 float64, mirror bool) *Mesh {
		var solids, holes []*Mesh
		for _, h := range b.Holes {
			x := h.X
			if mirror {
				x = -x
			}
			s, c := standoffAt(x, h.Y, m, h.D, fix, z0, z1, b.T, segs)
			solids = append(solids, s)
			if c != nil {
				holes = append(holes, c)
			}
		}
		out := base
		for _, s := range solids {
			out = Union(out, s)
		}
		if len(holes) > 0 {
			out = Subtract(out, UnionAll(holes))
		}
		if fix == "Through bolt + nut" {
			for _, h := range b.Holes {
				x := h.X
				if mirror {
					x = -x
				}
				out = Subtract(out, hexPrism(m.NutAF+0.4, -1, m.NutH+0.3).Transformed(Translate(x, h.Y, 0)))
			}
		}
		return out
	}
	// supports for boards without holes: ledges in the corners and a snap lip
	trayWalls := func(z0, zTop float64) *Mesh {
		in := outline2D(b, tol, segs)
		out := outline2D(b, tol+math.Min(wall, 1.6), segs)
		w := extrudeAt(Shape{Outer: out, Holes: []Loop{in}}, z0, zTop)
		var ledge *Mesh
		if b.Round {
			ledge = extrudeAt(Shape{Outer: outline2D(b, tol, segs), Holes: []Loop{outline2D(b, tol-2.5, segs)}}, z0, z0+hs)
		} else {
			cw := math.Min(6, math.Min(b.W, b.D)/4)
			X, Y := b.W/2+tol, b.D/2+tol
			for _, sx := range []float64{-1, 1} {
				for _, sy := range []float64{-1, 1} {
					x0, x1 := -X, -X+cw
					if sx > 0 {
						x0, x1 = X-cw, X
					}
					y0, y1 := -Y, -Y+cw
					if sy > 0 {
						y0, y1 = Y-cw, Y
					}
					c := boxAt(V(x0, y0, z0), V(x1, y1, z0+hs))
					if ledge == nil {
						ledge = c
					} else {
						ledge = Union(ledge, c)
					}
				}
			}
		}
		res := Union(w, ledge)
		// snap lips over the board edge on the long sides
		lipZ := z0 + hs + b.T + tol*0.5
		if !b.Round && lipZ+0.9 < zTop {
			lw := b.W * 0.3
			for _, sy := range []float64{-1, 1} {
				y0 := sy * (b.D/2 + tol)
				lip := prismY(Loop{{0, 0}, {0, 1.2}, {0.9, 0.6}}, lw).Transformed(Translate(0, y0, lipZ).Mul(RotZ(-90 * sy)))
				res = Union(res, lip)
			}
		}
		return res
	}
	portCuts := func(zBoardTop, outerGrow float64) *Mesh {
		cut := &Mesh{}
		if !p.B("ports", true) {
			return cut
		}
		for _, pt := range b.Ports {
			w := pt.W + 1.2
			z0, z1 := zBoardTop+pt.Z0-0.6, zBoardTop+pt.Z1+0.6
			var c *Mesh
			switch pt.Edge {
			case "-x", "+x":
				c = roundedBlock(outerGrow*2+4, w, z1-z0, 0, 0, 0, false, false, 1)
				x := b.W/2 + outerGrow/2
				if pt.Edge == "-x" {
					x = -x
				}
				c = c.Transformed(Translate(x, pt.Pos, z0))
			default:
				c = roundedBlock(w, outerGrow*2+4, z1-z0, 0, 0, 0, false, false, 1)
				y := b.D/2 + outerGrow/2
				if pt.Edge == "-y" {
					y = -y
				}
				c = c.Transformed(Translate(pt.Pos, y, z0))
			}
			cut.Append(c)
		}
		return cut
	}
	windowCuts := func(z0, z1 float64, mirror bool) *Mesh {
		ws := []BWindow{}
		if b.Window != nil {
			ws = append(ws, *b.Window)
		}
		ws = append(ws, b.Circles...)
		cut := &Mesh{}
		for _, w := range ws {
			x := w.X
			if mirror {
				x = -x
			}
			if w.Round {
				cut.Append(cylAt(x, w.Y, w.W+0.4, z0, z1, p.Segs(w.W)))
			} else {
				cut.Append(roundedBlock(w.W+0.4, w.D+0.4, z1-z0, 0.6, 0, 0, false, false, 3).Transformed(Translate(x, w.Y, z0)))
			}
		}
		return cut
	}
	wallGap := tol + 0.6 // room around the board inside a case
	switch mode {
	case "Standoffs only":
		if len(b.Holes) == 0 {
			return trayWalls(0, hs+b.T+1.5)
		}
		base := &Mesh{}
		return addStandoffs(base, 0, hs, false)

	case "Slide-in rails", "Slide-in case", "Slide-in case end cap":
		if b.Round {
			base := plateMesh(b, tol+1.6+margin*0.25, 0, floor, segs)
			return Union(base, trayWalls(floor-0.01, floor+hs+b.T+1.6))
		}
		// the board slides in from the +X end along grooves in two side rails
		W, D, T := b.W, b.D, b.T
		zb := floor + hs               // underside of the board
		grip := 1.2                    // how far the rails reach over the board edge
		gt := T + tol                  // groove height
		rw := 2.4                      // rail wall beyond the groove
		y0, y1 := D/2-grip, D/2+tol+rw // rail extent (one side, +Y)
		ztop := zb + gt + 1.6
		stopX := -W/2 - tol
		rails := func(xEnd float64, withStop bool) (*Mesh, *Mesh) {
			var solid, cut []*Mesh
			for _, sg := range []float64{-1, 1} {
				ya, yb := sg*y0, sg*y1
				if sg < 0 {
					ya, yb = yb, ya
				}
				solid = append(solid, boxAt(V(stopX-2.4, ya, 0), V(xEnd, yb, ztop)))
				ga, gb := sg*(D/2-grip-0.01), sg*(D/2+tol)
				if sg < 0 {
					ga, gb = gb, ga
				}
				cut = append(cut, boxAt(V(stopX, ga, zb-tol/2), V(xEnd+1, gb, zb-tol/2+gt)))
				// lead-in flare at the open end
				cut = append(cut, boxAt(V(xEnd-2, ga, zb-tol/2-0.4), V(xEnd+1, gb, zb-tol/2+gt+0.4)))
				// small bump that clicks behind the board when it is fully in
				solid = append(solid, cylAt(W/2+tol+0.8, sg*(D/2-0.1), 1.6, zb-tol/2-0.01, zb-tol/2+gt+0.01, 12))
			}
			if withStop {
				solid = append(solid, boxAt(V(stopX-2.4, -y1, 0), V(stopX, y1, ztop)))
			}
			return UnionAll(solid), UnionAll(cut)
		}
		if mode == "Slide-in rails" {
			base := boxAt(V(stopX-2.4, -y1, 0), V(W/2+tol+2, y1, floor))
			s, c := rails(W/2+tol+2, true)
			m := Subtract(Union(base, s), c)
			m.Info = "Slide the board in from the open end until it clicks."
			return m
		}
		// case: box around the rails, open at +X, with a separate end cap
		ow, oh := y1+wall, ztop+b.Above+p.F("extra", 2)+wall
		xa, xb := stopX-2.4-wall, W/2+tol+2
		if mode == "Slide-in case end cap" {
			t := 2.0
			plug := 3.0
			capM := boxAt(V(-oh/2, -ow, 0), V(oh/2, ow, t))
			lip := Subtract(boxAt(V(-oh/2+wall+tol, -ow+wall+tol, t-0.01), V(oh/2-wall-tol, ow-wall-tol, t+plug)), boxAt(V(-oh/2+wall+tol+1.6, -ow+wall+tol+1.6, t-1), V(oh/2-wall-tol-1.6, ow-wall-tol-1.6, t+plug+1)))
			m := Union(capM, lip)
			m.Info = "Press-fit end cap for the slide-in case (print it flat)."
			return m
		}
		box := Subtract(boxAt(V(xa, -ow, 0), V(xb, ow, oh)), boxAt(V(xa+wall, -ow+wall, floor), V(xb+1, ow-wall, oh-wall)))
		s, c := rails(xb, true)
		m := Subtract(Union(box, s), c)
		if pc := portCuts(zb+T, wallGap+wall); len(pc.Tris) > 0 {
			m = Subtract(m, pc)
		}
		m.Info = "Board slides in from the open end; close it with the Slide-in case end cap."
		return m

	case "Snap-in tray":
		base := plateMesh(b, tol+1.6+margin*0.25, 0, floor, segs)
		return Union(base, trayWalls(floor-0.01, floor+hs+b.T+1.6))

	case "Case", "Case lid":
		inner := wallGap
		innerH := hs + b.T + b.Above + p.F("extra", 2)
		total := floor + innerH
		if mode == "Case lid" {
			t := 2.0
			lip := 2.0
			lid := plateMesh(b, inner+wall, 0, t, segs)
			ring := extrudeAt(Shape{Outer: outline2D(b, inner-tol, segs), Holes: []Loop{outline2D(b, inner-tol-1.4, segs)}}, t-0.01, t+lip)
			lid = Union(lid, ring)
			// lid is printed face-down: window positions are mirrored in X
			if b.Window != nil || len(b.Circles) > 0 {
				lid = Subtract(lid, windowCuts(-1, t+lip+1, true))
			}
			return lid
		}
		outer := plateMesh(b, inner+wall, 0, total, segs)
		cavity := plateMesh(b, inner, floor, total+1, segs)
		c := Subtract(outer, cavity)
		if b.Cut != nil {
			c = Subtract(c, cutBody(b, -1, floor+1, segs))
		}
		if len(b.Holes) > 0 {
			c = addStandoffs(c, floor-0.01, floor+hs, false)
		} else {
			c = Union(c, trayWalls(floor-0.01, floor+hs+b.T+1.2))
		}
		if pc := portCuts(floor+hs+b.T, inner+wall); len(pc.Tris) > 0 {
			c = Subtract(c, pc)
		}
		return c

	case "Front bezel":
		t := 2.0
		bz := plateMesh(b, wallGap+wall+margin*0.5, 0, t, segs)
		gh := 1.6
		if b.Window != nil {
			gh = b.Window.H
		}
		// locating rim around the module
		rim := extrudeAt(Shape{Outer: outline2D(b, wallGap+wall*0.8, segs), Holes: []Loop{outline2D(b, wallGap, segs)}}, t-0.01, t+gh+b.T)
		bz = Union(bz, rim)
		if len(b.Holes) > 0 {
			bz = addStandoffs(bz, t-0.01, t+gh, true)
		}
		return Subtract(bz, windowCuts(-1, t+0.01, true))

	case "Board model (fit check)":
		pcb := plateMesh(b, 0, 0, b.T, segs)
		if len(b.Holes) > 0 {
			pcb = holesThrough(pcb, holePts(b), b.Holes[0].D, -1, b.T+1, segs)
		}
		parts := []*Mesh{pcb}
		if b.Window != nil {
			w := b.Window
			if w.Round {
				parts = append(parts, cylAt(w.X, w.Y, w.W, b.T-0.01, b.T+w.H, p.Segs(w.W)))
			} else {
				parts = append(parts, boxAt(V(w.X-w.W/2, w.Y-w.D/2, b.T-0.01), V(w.X+w.W/2, w.Y+w.D/2, b.T+w.H)))
			}
		}
		for _, c := range b.Circles {
			parts = append(parts, cylAt(c.X, c.Y, c.W, b.T-0.01, b.T+c.H, p.Segs(c.W)))
		}
		for _, pt := range b.Ports {
			var lo, hi Vec3
			depth := 8.0
			switch pt.Edge {
			case "-x":
				lo, hi = V(-b.W/2-1, pt.Pos-pt.W/2, b.T+pt.Z0), V(-b.W/2+depth, pt.Pos+pt.W/2, b.T+pt.Z1)
			case "+x":
				lo, hi = V(b.W/2-depth, pt.Pos-pt.W/2, b.T+pt.Z0), V(b.W/2+1, pt.Pos+pt.W/2, b.T+pt.Z1)
			case "-y":
				lo, hi = V(pt.Pos-pt.W/2, -b.D/2-1, b.T+pt.Z0), V(pt.Pos+pt.W/2, -b.D/2+depth, b.T+pt.Z1)
			default:
				lo, hi = V(pt.Pos-pt.W/2, b.D/2-depth, b.T+pt.Z0), V(pt.Pos+pt.W/2, b.D/2+1, b.T+pt.Z1)
			}
			parts = append(parts, boxAt(lo, hi))
		}
		if b.Cut != nil {
			parts = append(parts, cutBody(b, -b.Below, 0.01, segs))
		}
		res := parts[0]
		for _, q := range parts[1:] {
			res = Union(res, q)
		}
		return res

	case "Holder / cradle":
		depth := math.Max(b.T+1, math.Min(b.T+b.Above, 12))
		block := plateMesh(b, tol+wall+1, 0, floor+depth, segs)
		var pocket *Mesh
		if len(b.Outline) >= 3 {
			pocket = extrudeAt(Shape{Outer: offsetLoop(Loop(b.Outline), tol)}, floor, floor+depth+1)
		} else {
			pocket = plateMesh(b, tol, floor, floor+depth+1, segs)
		}
		block = Subtract(block, pocket)
		// finger notches on both long sides to lift it out
		for _, sy := range []float64{-1, 1} {
			block = Subtract(block, cylAt(0, sy*(b.D/2+tol+wall*0.5), math.Min(16, b.W*0.4), floor+depth*0.35, floor+depth+1, 24))
		}
		return block

	case "Mounting holes":
		dep := p.F("depth", 6)
		hm := &Mesh{}
		for _, h := range b.Holes {
			d := m.Normal + 0.2
			switch fix {
			case "Self-tapping screw":
				d = m.D * 0.85
			case "Heat-set insert":
				d = m.InsertD
			}
			hm.Append(cylAt(h.X, h.Y, d, -dep, 0.6, segs))
		}
		if b.Cut != nil {
			hm = Union(hm, cutBody(b, -dep, 0.6, segs))
		}
		// panel openings for screens, sensors and knob shafts
		if wc := windowCuts(-dep, 0.6, false); len(wc.Tris) > 0 {
			hm = Union(hm, wc)
		}
		if len(hm.Tris) == 0 {
			// nothing to screw into: a recess the board drops into
			hm = plateMesh(b, tol, -b.T-0.3, 0.6, segs)
		}
		return hm
	}
	// Mount plate (default)
	plate := plateMesh(b, margin, 0, floor, segs)
	if len(b.Holes) == 0 {
		return Union(plate, trayWalls(floor-0.01, floor+hs+b.T+1.5))
	}
	if b.Cut != nil {
		plate = Subtract(plate, cutBody(b, -1, floor+1, segs))
	}
	return addStandoffs(plate, floor-0.01, floor+hs, false)
}

func cutBody(b *BoardSpec, z0, z1 float64, segs int) *Mesh {
	c := b.Cut
	if c.Round {
		return cylAt(c.X, c.Y, c.W+0.4, z0, z1, segs*2)
	}
	return boxAt(V(c.X-c.W/2-0.2, c.Y-c.D/2-0.2, z0), V(c.X+c.W/2+0.2, c.Y+c.D/2+0.2, z1))
}

func holePts(b *BoardSpec) [][2]float64 {
	var o [][2]float64
	for _, h := range b.Holes {
		o = append(o, [2]float64{h.X, h.Y})
	}
	return o
}

// offsetLoop grows a polygon outward by d (miter join, fine for gentle shapes).
func offsetLoop(l Loop, d float64) Loop {
	if l.Area() < 0 {
		l = l.Reversed()
	}
	n := len(l)
	out := make(Loop, n)
	for i := range l {
		a, b, c := l[(i+n-1)%n], l[i], l[(i+1)%n]
		e1 := V(b.X-a.X, b.Y-a.Y, 0).Norm()
		e2 := V(c.X-b.X, c.Y-b.Y, 0).Norm()
		n1, n2 := V(e1.Y, -e1.X, 0), V(e2.Y, -e2.X, 0)
		nm := n1.Add(n2).Norm()
		k := nm.Dot(n1)
		if k < 0.3 {
			k = 0.3
		}
		out[i] = P2{b.X + nm.X*d/k, b.Y + nm.Y*d/k}
	}
	return out
}

func boardInfo(b *BoardSpec, p Params) string {
	m := screwFor(p, 2.7)
	if len(b.Holes) > 0 {
		m = screwFor(p, b.Holes[0].D)
	}
	s := fmt.Sprintf("%s\nBoard %.1f × %.1f × %.1f mm · tallest part %.1f mm · under-side %.1f mm", b.Name, b.W, b.D, b.T, b.Above, b.Below)
	if len(b.Holes) > 0 {
		s += fmt.Sprintf("\n%d mounting holes Ø%.1f → %s screws", len(b.Holes), b.Holes[0].D, metricName(m))
	} else {
		s += "\nNo mounting holes → held by a snap-in tray with corner ledges and clips"
	}
	if len(b.Ports) > 0 {
		var ns []string
		for _, pt := range b.Ports {
			ns = append(ns, pt.Name)
		}
		s += "\nConnector openings: " + strings.Join(ns, ", ")
	}
	if b.Window != nil {
		s += fmt.Sprintf("\nScreen / window %.1f × %.1f mm", b.Window.W, b.Window.D)
	}
	return s
}

func metricName(m metric) string {
	for k, v := range metrics {
		if v.D == m.D {
			return k
		}
	}
	return "M3"
}

func init() {
	reg := func(typ, name, desc, mode, color string, hole bool) {
		modes := fitModes
		params := boardParams(mode, modes)
		if hole {
			params = boardParams(mode, nil)
			params = append(params, num("depth", "Hole depth", 6, 1, 200, 0.5, "mm"))
		}
		cat := "Electronics"
		if hole {
			cat = "Holes"
		}
		register(&ShapeDef{Type: typ, Name: name, Category: cat, Desc: desc, Hole: hole, Fixed: true, Color: color,
			Tags:   "raspberry pi arduino esp32 esp8266 pico nano oled tft lcd cyd screen display led ring ws2812 relay servo sensor module mount case enclosure bezel standoff board pcb " + name,
			Params: params,
			Gen: func(p Params) (*Mesh, error) {
				b := boardFromParams(p)
				md := mode
				if !hole {
					md = p.S("mode", mode)
				}
				r := FitMesh(b, md, p)
				r.Info = boardInfo(b, p)
				return r, nil
			}})
	}
	reg("boardmount", "Board mount", "Base plate with standoffs for a Raspberry Pi, Arduino, ESP32, screen or module (or a snap-in tray if the board has no holes).", "Mount plate", "#16a34a", false)
	reg("boardcase", "Board case", "Enclosure sized for the board with standoffs and openings for its connectors. Add a Case lid with the same board.", "Case", "#334155", false)
	reg("boardlid", "Board case lid", "Lid for the Board case, with a window for screens and sensors. Printed face down.", "Case lid", "#475569", false)
	reg("bezel", "Screen / module bezel", "Front frame with a window that holds a screen, LED ring or sensor from behind.", "Front bezel", "#0f172a", false)
	reg("boardmodel", "Board model", "Simple stand-in model of the board with its connectors — use it to check fit in your design.", "Board model (fit check)", "#15803d", false)
	reg("pcbmount", "Board mounting holes", "Mounting-hole pattern (and body cut-out for servos/motors) for any board in the library.", "Mounting holes", "#64748b", true)

	sizes := []string{"M2", "M2.5", "M3", "M4"}
	register(&ShapeDef{Type: "screwboss", Name: "Screw boss", Category: "Electronics", Color: "#22c55e", Fixed: true,
		Desc: "Mounting point for M2 / M2.5 / M3 / M4 screws with a flared base and optional ribs — place it on a floor or wall.",
		Tags: "screw boss mounting point standoff post m2 m3 m4 insert",
		Params: []Param{sel("size", "Screw", "M3", sizes...), sel("fix", "Fixing", "Heat-set insert", "Self-tapping screw", "Heat-set insert", "Through bolt + nut"),
			num("h", "Height", 8, 1, 200, 0.5, "mm"), boolp("ribs", "Support ribs", true)},
		Gen: func(p Params) (*Mesh, error) {
			m := getMetric(p)
			h := p.F("h", 8)
			fix := p.S("fix", "Heat-set insert")
			od := math.Max(m.D*2.4, bossOD(m.InsertD))
			if fix == "Through bolt + nut" {
				od = math.Max(od, hexCorner(m.NutAF+0.4)+2*screwWall(m.Normal))
			}
			s, c := standoffAt(0, 0, m, m.D+0.2, fix, 0, h, 1.6, p.Segs(m.D*3))
			s = Union(s, cylAt(0, 0, od, 0, h, p.Segs(od)))
			// flared foot with a short upright edge (no knife edge at the bottom)
			s = Union(s, cylAt(0, 0, od+3, 0, 0.8, p.Segs(od+3)))
			s = Union(s, coneAt(od+3, od, 0.79, 2.3, p.Segs(od+3)))
			if p.B("ribs", true) {
				for k := 0; k < 4; k++ {
					rib := prismY(Loop{{od/2 - 0.3, 0}, {od/2 + 3, 0}, {od/2 + 3, 0.8}, {od/2 - 0.3, h * 0.7}}, 1.6).Transformed(RotZ(45 + 90*float64(k)))
					s = Union(s, rib)
				}
			}
			if c != nil {
				s = Subtract(s, c)
			}
			if p.S("fix", "") == "Through bolt + nut" {
				s = Subtract(s, hexPrism(m.NutAF+0.4, -1, m.NutH+0.3))
			}
			return s, nil
		}})

	register(&ShapeDef{Type: "mountear", Name: "Mounting ear / tab", Category: "Electronics", Color: "#22c55e",
		Desc: "Screw tab that sticks out from the side of a case or part so it can be screwed to a wall or panel.", Tags: "mounting ear tab flange lug wall mount m3 m4",
		Params: []Param{sel("size", "Screw", "M3", sizes...), num("w", "Width", 14, 5, 100, 0.5, "mm"), num("t", "Thickness", 3, 1, 20, 0.5, "mm"), num("reach", "Reach", 10, 4, 100, 0.5, "mm"), boolp("gusset", "Gusset", true), boolp("csk", "Countersunk", true)},
		Gen: func(p Params) (*Mesh, error) {
			m := getMetric(p)
			w, t, r := p.F("w", 14), p.F("t", 3), p.F("reach", 10)
			l := append(Loop{{0, -w / 2}}, arcLoop(r, 0, w/2, -90, 90)...)
			l = append(l, P2{0, w / 2})
			ear := Extrude(Shape{Outer: l}, t, 0, 1, 1)
			if p.B("gusset", true) {
				ear = Union(ear, prismY(Loop{{-0.01, t - 0.01}, {r * 0.45, t - 0.01}, {-0.01, t + r*0.5}}, 1.6))
			}
			hole := cylAt(r, 0, m.Normal+0.2, -1, t+1, p.Segs(m.D))
			if p.B("csk", true) {
				hole = Union(hole, coneAt(m.Normal, m.CskD+0.4, t-(m.CskD-m.Normal)/2, t+0.01, p.Segs(m.CskD)).Transformed(Translate(r, 0, 0)))
			}
			return Subtract(ear, hole), nil
		}})

	register(&ShapeDef{Type: "cornerpost", Name: "Lid screw corner post", Category: "Electronics", Color: "#22c55e", Fixed: true,
		Desc: "Quarter-round post that sits in an enclosure corner and takes the lid screw (insert or self-tapping).", Tags: "corner post lid screw enclosure insert boss m2 m3 m4",
		Params: []Param{sel("size", "Screw", "M3", sizes...), sel("fix", "Fixing", "Heat-set insert", "Self-tapping screw", "Heat-set insert"), num("h", "Height", 20, 3, 300, 0.5, "mm"), num("r", "Size (grows to fit the screw)", 7, 4, 30, 0.5, "mm")},
		Gen: func(p Params) (*Mesh, error) {
			m := getMetric(p)
			h, r := p.F("h", 20), p.F("r", 7)
			d := m.D * 0.85
			if p.S("fix", "") == "Heat-set insert" {
				d = m.InsertD
			}
			// The hole sits at (a, a) with the same wall all round it: a square corner block
			// with the inner corner rounded about the hole. The post grows if the screw needs it.
			ro := math.Max(r/2, d/2+screwWall(d))
			a := ro
			l := append(Loop{{0, 0}}, arcLoop(a, a, ro, -90, 180)...)
			body := Extrude(Shape{Outer: l}, h, 0, 1, 1)
			depth := math.Min(h-1.6, math.Max(m.InsertL+1, 8))
			body = Subtract(body, cylAt(a, a, d, h-depth, h+1, p.Segs(d)))
			body.Info = fmt.Sprintf("Post %.1f mm across · %.1f mm of wall all round the %.1f mm hole.", 2*ro, ro-d/2, d)
			return body, nil
		}})

	register(&ShapeDef{Type: "pcbpin", Name: "PCB snap pin", Category: "Electronics", Color: "#22c55e",
		Desc: "Standoff with a split snap pin on top — push a board's mounting hole over it, no screws needed.", Tags: "pcb snap pin clip standoff push",
		Params: []Param{num("hole", "Board hole Ø", 2.75, 1.5, 6, 0.05, "mm"), num("h", "Standoff height", 5, 1, 50, 0.5, "mm"), num("t", "Board thickness", 1.6, 0.4, 5, 0.1, "mm")},
		Gen: func(p Params) (*Mesh, error) {
			s, _ := standoffAt(0, 0, metrics["M2.5"], p.F("hole", 2.75), "Snap pins (no screws)", 0, p.F("h", 5), p.F("t", 1.6), p.Segs(6))
			return s, nil
		}})

	register(&ShapeDef{Type: "filhinge", Name: "Filament hinge", Category: "Mechanical", Color: "#7c3aed",
		Desc: "Hinge whose axle is a piece of 1.75 mm (or 2.85 mm) 3D-printer filament. Print flat, push the filament through, trim and melt the ends.",
		Tags: "filament hinge pin axle 1.75 knuckle lid door",
		Params: []Param{
			num("w", "Length", 40, 10, 500, 1, "mm"), num("leaf", "Leaf width", 15, 4, 200, 1, "mm"), num("t", "Leaf thickness", 2.5, 1, 20, 0.2, "mm"),
			num("knuckles", "Knuckles", 5, 3, 31, 2, ""), sel("fil", "Filament", "1.75 mm", "1.75 mm", "2.85 mm"),
			sel("fit", "Pin fit", "Free turning", "Press fit (stiff)", "Free turning", "Loose"),
			num("gap", "Knuckle gap", 0.3, 0.1, 1.5, 0.05, "mm"),
			sel("layout", "Print as", "One piece (assembled, flat)", "One piece (assembled, flat)", "Two separate leaves"),
			sel("size", "Screw holes", "M3", "None", "M2", "M2.5", "M3", "M4"),
		},
		Gen: func(p Params) (*Mesh, error) {
			w, leaf, t := p.F("w", 40), p.F("leaf", 15), p.F("t", 2.5)
			n := p.I("knuckles", 5)
			if n%2 == 0 {
				n++
			}
			fd := 1.75
			if p.S("fil", "") == "2.85 mm" {
				fd = 2.85
			}
			hole := fd + map[string]float64{"Press fit (stiff)": 0.05, "Free turning": 0.3, "Loose": 0.5}[p.S("fit", "Free turning")]
			kd := math.Max(2*t, hole+2.6)
			gap := p.F("gap", 0.3)
			seg := w / float64(n)
			zc := kd / 2
			segs := p.Segs(kd)
			cylX := func(x0, x1, d float64) *Mesh {
				return RevolveProfile([]P2{{0, 0}, {d / 2, 0}, {d / 2, x1 - x0}, {0, x1 - x0}}, segs).Transformed(Translate(x0, 0, zc).Mul(RotY(90)))
			}
			sep := p.S("layout", "") == "Two separate leaves"
			var A, B []*Mesh
			A = append(A, boxAt(V(-w/2, kd/2+gap, 0), V(w/2, kd/2+gap+leaf, t)))
			B = append(B, boxAt(V(-w/2, -kd/2-gap-leaf, 0), V(w/2, -kd/2-gap, t)))
			for i := 0; i < n; i++ {
				x0 := -w/2 + float64(i)*seg + gap/2
				x1 := x0 + seg - gap
				if i == 0 {
					x0 -= gap / 2
				}
				if i == n-1 {
					x1 += gap / 2
				}
				k := cylX(x0, x1, kd)
				if i%2 == 0 {
					A = append(A, k, boxAt(V(x0, 0, 0), V(x1, kd/2+gap+0.5, t)))
				} else {
					B = append(B, k, boxAt(V(x0, -kd/2-gap-0.5, 0), V(x1, 0, t)))
				}
			}
			a, b := UnionAll(A), UnionAll(B)
			axle := cylX(-w/2-1, w/2+1, hole)
			// chamfered entry so the filament is easy to push in
			entry := RevolveProfile([]P2{{0, 0}, {hole/2 + 0.6, 0}, {hole / 2, 0.6}, {0, 0.6}}, segs).Transformed(Translate(-w/2-0.01, 0, zc).Mul(RotY(90)))
			a, b = Subtract(a, Union(axle, entry)), Subtract(b, axle)
			if mt, ok := metrics[p.S("size", "M3")]; ok {
				for _, x := range []float64{-w / 3, w / 3} {
					cs := Union(cylAt(x, 0, mt.Normal+0.2, -1, t+1, 16), coneAt(mt.Normal, mt.CskD+0.3, t-(mt.CskD-mt.Normal)/2, t+0.01, 16).Transformed(Translate(x, 0, 0)))
					a = Subtract(a, cs.Transformed(Translate(0, kd/2+gap+leaf*0.6, 0)))
					b = Subtract(b, cs.Transformed(Translate(0, -kd/2-gap-leaf*0.6, 0)))
				}
			}
			if sep {
				b = b.Transformed(Translate(0, -6, 0))
			}
			m := a.Clone()
			m.Append(b)
			m.Info = fmt.Sprintf("Axle: %.2f mm filament in a %.2f mm hole · knuckle Ø %.1f mm · cut the filament to %.0f mm", fd, hole, kd, w+2)
			return m, nil
		}})
}
