package geom

// Keyboard cases for boards made with Keebie (Wirie's keyboard designer):
// an open-top tray that the PCB screws into, a printed switch plate with a
// cut-out for every switch, knob and stabiliser, and optional rear feet for
// a typing angle. Made by nrmluvdev.

import (
	"fmt"
	"math"
	"sort"
)

// WirieKeyboard is the "keyboard" block of a wirie-case file.
type WirieKeyboard struct {
	Switch         string       `json:"switch"`         // mx | choc | alps
	PlateTop       float64      `json:"plateTop"`       // plate top, above the board's top face
	PlateThickness float64      `json:"plateThickness"` // usual plate thickness for the switch
	Hotswap        bool         `json:"hotswap,omitempty"`
	Cutouts        []WirieKbCut `json:"cutouts"`
}

// WirieKbCut is one rectangular plate cut-out (switch, knob or stabiliser).
type WirieKbCut struct {
	Ref      string  `json:"ref,omitempty"`
	Kind     string  `json:"kind"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	W        float64 `json:"w"`
	H        float64 `json:"h"`
	Rotation float64 `json:"rotation"`
}

func (c *WirieCase) isKeyboard() bool { return c.Keyboard != nil && len(c.Keyboard.Cutouts) > 0 }

func (kb *WirieKeyboard) plateT() float64 {
	if kb.PlateThickness > 0 {
		return kb.PlateThickness
	}
	return 1.5
}

func kbCaseParams(c *WirieCase) []Param {
	e, kb := c.Envelope, c.Keyboard
	screws := []string{"From board", "M2", "M2.5", "M3", "M4", "M5"}
	return append([]Param{
		sel("plate", "Switch plate", "Printed plate", "Printed plate", "No plate").help("The plate holds the switches straight and takes the typing force. Print it flat. Choose No plate for a PCB-mount build (switches soldered or pushed into the PCB only)."),
		num("plateT", "Plate thickness", round3(kb.plateT()), 1, 5, 0.1, "mm").help("MX switch clips are made for 1.5 mm; Choc plates are 1.2 mm. A thicker plate is stiffer but the switches may not clip in.").show("plate=Printed plate"),
		num("wall", "Wall thickness", 2.4, 1.2, 10, 0.2, "mm"),
		num("floor", "Floor thickness", 2, 1.2, 10, 0.2, "mm"),
		num("fit", "Fit gap around the board", 0.5, 0, 3, 0.05, "mm"),
		num("standoff", "Standoff height", round3(math.Max(3, e.MaxPartHeightBottom+1)), 1, 40, 0.5, "mm").help("Space between the case floor and the bottom of the PCB. Hot-swap sockets and diodes hang under the board."),
		num("rim", "Wall above the plate", 0, 0, 12, 0.5, "mm").help("0 makes the walls flush with the plate. Higher walls hide the switch sides (a 'high profile' case)."),
		num("tilt", "Typing angle", 6, 0, 12, 0.5, "°").help("Feet under the back tilt the keyboard towards you. 0 lays it flat.").show("feet!=No feet"),
		num("radius", "Corner radius (-1 = follow the board)", -1, -1, 50, 0.5, "mm"),
		sel("screw", "Screw size", "From board", screws...),
		sel("fixing", "Screws go into", "Self-tapping screw", caseFixings...),
	}, feetParams(kbFeetDefault)...)
}

// keyboards sit on rubber feet by default (in indents, so they stay put)
const kbFeetDefault = "Rubber feet in indents"

// heights of a keyboard case, in the board frame
type kbZs struct{ floorBot, floorTop, plateBot, plateTop, wallTop float64 }

func (c *WirieCase) kbZ(o CaseOpts, p Params) kbZs {
	kb, t := c.Keyboard, c.Board.Thickness
	var z kbZs
	z.plateTop = t + kb.PlateTop
	z.plateBot = z.plateTop - clampF(p.F("plateT", kb.plateT()), 0.8, 5)
	z.floorTop = -o.Standoff
	z.floorBot = z.floorTop - o.Floor
	z.wallTop = z.plateTop + clampF(p.F("rim", 0), 0, 12)
	return z
}

func (c *WirieCase) kbOuter(o CaseOpts) (outer, cavity Loop) {
	r := o.Radius
	if r < 0 {
		r = c.Board.CornerRadius + o.Fit + o.Wall
	}
	return c.outlineAt(o.Fit+o.Wall, r, o.segs), c.outlineAt(o.Fit, math.Max(0, r-o.Wall), o.segs)
}

// kbTrayMesh: floor and walls up to the plate (open top).
func (c *WirieCase) kbTrayMesh(o CaseOpts, p Params) *Mesh {
	z := c.kbZ(o, p)
	outer, cavity := c.kbOuter(o)
	return Subtract(extrudeAt(Shape{Outer: outer}, z.floorBot, z.wallTop), extrudeAt(Shape{Outer: cavity}, z.floorTop, z.wallTop+1))
}

func cutLoop(x, y, w, h, rot float64) Loop {
	return moveLoop(rotLoop(rectL(-w/2, -h/2, w/2, h/2), rot), x, y)
}

// loopsTouch: do two closed loops overlap (or nearly)?
func loopsTouch(a, b Loop) bool {
	ax0, ay0, ax1, ay1 := loopBox(a)
	bx0, by0, bx1, by1 := loopBox(b)
	if ax1 < bx0-0.3 || bx1 < ax0-0.3 || ay1 < by0-0.3 || by1 < ay0-0.3 {
		return false
	}
	for _, q := range a {
		if b.Contains(q) {
			return true
		}
	}
	for _, q := range b {
		if a.Contains(q) {
			return true
		}
	}
	for i := range a {
		for j := range b {
			if segIntersect(a[i], a[(i+1)%len(a)], b[j], b[(j+1)%len(b)]) || segDist(a[i], b[j], b[(j+1)%len(b)]) < 0.3 {
				return true
			}
		}
	}
	return false
}

func loopBox(l Loop) (x0, y0, x1, y1 float64) {
	x0, y0, x1, y1 = math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, q := range l {
		x0, y0, x1, y1 = math.Min(x0, q.X), math.Min(y0, q.Y), math.Max(x1, q.X), math.Max(y1, q.Y)
	}
	return
}

func segDist(p, a, b P2) float64 {
	dx, dy := b.X-a.X, b.Y-a.Y
	l2 := dx*dx + dy*dy
	t := 0.0
	if l2 > 0 {
		t = clampF(((p.X-a.X)*dx+(p.Y-a.Y)*dy)/l2, 0, 1)
	}
	return math.Hypot(p.X-a.X-t*dx, p.Y-a.Y-t*dy)
}

func loopEdgeDist(l Loop, q P2) float64 {
	d := math.Inf(1)
	for i := range l {
		d = math.Min(d, segDist(q, l[i], l[(i+1)%len(l)]))
	}
	return d
}

// hull2 is the convex hull (CCW) of a set of points.
func hull2(pts []P2) Loop {
	ps := append([]P2(nil), pts...)
	sort.Slice(ps, func(i, j int) bool { return ps[i].X < ps[j].X || ps[i].X == ps[j].X && ps[i].Y < ps[j].Y })
	var h []P2
	for pass := 0; pass < 2; pass++ {
		start := len(h)
		for _, q := range ps {
			for len(h) >= start+2 && cross2(h[len(h)-2], h[len(h)-1], q) <= 0 {
				h = h[:len(h)-1]
			}
			h = append(h, q)
		}
		h = h[:len(h)-1]
		for i, j := 0, len(ps)-1; i < j; i, j = i+1, j-1 {
			ps[i], ps[j] = ps[j], ps[i]
		}
	}
	return Loop(h)
}

// kbPlateHoles works out the plate's holes: switch, knob and stabiliser
// cut-outs, clearance for tall parts and screw access. Holes that overlap are
// merged (their convex hull); holes that reach the plate's edge are returned
// separately so they can be cut as notches.
func (c *WirieCase) kbPlateHoles(o CaseOpts, z kbZs, outer Loop) (inner, edge []Loop) {
	var holes []Loop
	kbRefs := map[string]bool{}
	for _, k := range c.Keyboard.Cutouts {
		if k.W > 0.5 && k.H > 0.5 {
			holes = append(holes, cutLoop(k.X, k.Y, k.W, k.H, k.Rotation))
			kbRefs[k.Ref] = true
		}
	}
	for _, pt := range c.Parts {
		if kbRefs[pt.Ref] || pt.Side == "bottom" || !partSolid(pt) || pt.Box.Z1 < z.plateBot-0.3 {
			continue
		}
		b := pt.Box
		holes = append(holes, rectL(b.X0-1, b.Y0-1, b.X1+1, b.Y1+1))
	}
	m := metrics[c.screwName(o.Screw)]
	// screw access, only where it doesn't run into a cut-out (merging the two
	// would leave odd wedges; the screw is reachable through the cut-out then)
	for _, h := range c.Holes {
		d := math.Max(m.ButtonD+1.2, h.D+1.5)
		ring := Circle(d/2, 20).translated(h.X, h.Y)
		free := true
		for _, o := range holes {
			if loopsTouch(ring, o) {
				free = false
				break
			}
		}
		if free {
			holes = append(holes, ring)
		}
	}
	// merge overlapping holes until nothing changes
	for changed := true; changed; {
		changed = false
		for i := 0; i < len(holes) && !changed; i++ {
			for j := i + 1; j < len(holes); j++ {
				if loopsTouch(holes[i], holes[j]) {
					holes[i] = hull2(append(append([]P2(nil), holes[i]...), holes[j]...))
					holes = append(holes[:j], holes[j+1:]...)
					changed = true
					break
				}
			}
		}
	}
	for _, h := range holes {
		in := true
		for _, q := range h {
			if !outer.Contains(q) || loopEdgeDist(outer, q) < 2.5 {
				in = false
				break
			}
		}
		if in {
			inner = append(inner, h)
		} else {
			edge = append(edge, h)
		}
	}
	return
}

func (c *WirieCase) kbPlateOutline(o CaseOpts) Loop {
	g := o.Fit - 0.3
	r := o.Radius
	if r < 0 {
		r = c.Board.CornerRadius + g
	}
	return c.outlineAt(g, math.Max(0, r-o.Wall-0.3), o.segs)
}

// kbPlateMesh: the switch plate, in place on top of the switches.
func (c *WirieCase) kbPlateMesh(o CaseOpts, p Params) *Mesh {
	z := c.kbZ(o, p)
	outer := c.kbPlateOutline(o)
	inner, edge := c.kbPlateHoles(o, z, outer)
	if len(edge) > 0 { // holes at the edge: trace the plate outline without them
		var more []Loop
		outer, more = rasterShape(outer, edge, 0)
		inner = append(inner, more...)
	}
	return slabExtrude(Shape{Outer: outer, Holes: inner}, z.plateBot, z.plateTop)
}

// slabExtrude extrudes a shape with many holes. The caps are cut into
// horizontal slabs at every vertex height, so rows of identical holes (a
// switch plate) triangulate cleanly and the mesh stays watertight.
func slabExtrude(s Shape, z0, z1 float64) *Mesh {
	s = s.Normalize()
	loops := append([]Loop{s.Outer}, s.Holes...)
	var ys []float64
	for _, l := range loops {
		for _, q := range l {
			ys = append(ys, q.Y)
		}
	}
	sort.Float64s(ys)
	lv := ys[:0]
	for _, y := range ys {
		if len(lv) == 0 || y-lv[len(lv)-1] > 1e-7 {
			lv = append(lv, y)
		}
	}
	level := func(y float64) int { // index of the level at y
		i := sort.SearchFloat64s(lv, y-1e-7)
		if i >= len(lv) {
			i = len(lv) - 1
		}
		return i
	}
	// split every edge where it crosses a level, and snap vertices onto their level
	ref := make([]Loop, len(loops))
	levelPts := make([][]float64, len(lv))
	for li, l := range loops {
		var out Loop
		n := len(l)
		for i := 0; i < n; i++ {
			a, b := l[i], l[(i+1)%n]
			ka, kb := level(a.Y), level(b.Y)
			out = append(out, P2{a.X, lv[ka]})
			step := 1
			if kb < ka {
				step = -1
			}
			for k := ka + step; k != kb && kb != ka; k += step {
				t := (lv[k] - a.Y) / (b.Y - a.Y)
				out = append(out, P2{a.X + t*(b.X-a.X), lv[k]})
			}
		}
		ref[li] = out
		for _, q := range out {
			k := level(q.Y)
			levelPts[k] = append(levelPts[k], q.X)
		}
	}
	for k := range levelPts {
		sort.Float64s(levelPts[k])
	}
	m := &Mesh{}
	for _, l := range ref {
		n := len(l)
		for i := 0; i < n; i++ {
			a, b := l[i], l[(i+1)%n]
			m.AddQuad(V(a.X, a.Y, z0), V(b.X, b.Y, z0), V(b.X, b.Y, z1), V(a.X, a.Y, z1))
		}
	}
	type sedge struct{ xb, xt float64 }
	slabs := make([][]sedge, len(lv))
	for _, l := range ref {
		n := len(l)
		for i := 0; i < n; i++ {
			a, b := l[i], l[(i+1)%n]
			ka, kb := level(a.Y), level(b.Y)
			if ka == kb {
				continue
			}
			if ka > kb {
				a, b, ka = b, a, kb
			}
			slabs[ka] = append(slabs[ka], sedge{a.X, b.X})
		}
	}
	between := func(xs []float64, lo, hi float64) []float64 {
		var o []float64
		for _, x := range xs {
			if x > lo+1e-7 && x < hi-1e-7 {
				o = append(o, x)
			}
		}
		return o
	}
	for k, es := range slabs {
		if len(es) < 2 {
			continue
		}
		sort.Slice(es, func(i, j int) bool { return es[i].xb+es[i].xt < es[j].xb+es[j].xt })
		yb, yt := lv[k], lv[k+1]
		for i := 0; i+1 < len(es); i += 2 {
			L, R := es[i], es[i+1]
			poly := []P2{{L.xb, yb}}
			for _, x := range between(levelPts[k], L.xb, R.xb) {
				poly = append(poly, P2{x, yb})
			}
			poly = append(poly, P2{R.xb, yb}, P2{R.xt, yt})
			top := between(levelPts[k+1], L.xt, R.xt)
			for j := len(top) - 1; j >= 0; j-- {
				poly = append(poly, P2{top[j], yt})
			}
			poly = append(poly, P2{L.xt, yt})
			var pp []P2
			for _, q := range poly {
				if len(pp) == 0 || !samePt(pp[len(pp)-1], q) {
					pp = append(pp, q)
				}
			}
			if len(pp) > 1 && samePt(pp[0], pp[len(pp)-1]) {
				pp = pp[:len(pp)-1]
			}
			if len(pp) < 3 {
				continue
			}
			var cx, cy float64
			for _, q := range pp {
				cx += q.X
				cy += q.Y
			}
			cx, cy = cx/float64(len(pp)), cy/float64(len(pp))
			for j := range pp {
				a, b := pp[j], pp[(j+1)%len(pp)]
				m.Add(V(cx, cy, z1), V(a.X, a.Y, z1), V(b.X, b.Y, z1))
				m.Add(V(cx, cy, z0), V(b.X, b.Y, z0), V(a.X, a.Y, z0))
			}
		}
	}
	return m.Orient()
}

// kbWallHit finds where a ray from (x, y) along n leaves the board outline.
func (c *WirieCase) kbWallHit(x, y float64, n [2]float64) (float64, float64) {
	l := c.loop()
	best := math.Inf(1)
	for i := range l {
		a, b := l[i], l[(i+1)%len(l)]
		ex, ey := b.X-a.X, b.Y-a.Y
		den := n[0]*ey - n[1]*ex
		if math.Abs(den) < 1e-9 {
			continue
		}
		t := ((a.X-x)*ey - (a.Y-y)*ex) / den
		u := ((a.X-x)*n[1] - (a.Y-y)*n[0]) / den
		if t > -3 && u >= 0 && u <= 1 && t < best {
			best = t
		}
	}
	if math.IsInf(best, 1) {
		return x, y
	}
	return x + n[0]*best, y + n[1]*best
}

// buildKeyboardCase makes the keyboard case nodes: base (tray, standoffs,
// openings, feet) and the plate.
func (c *WirieCase) buildKeyboardCase(p Params) ([]*Node, CaseInfo) {
	o := caseOpts(c, p)
	z := c.kbZ(o, p)
	var info CaseInfo
	info.LidStyle = "Keyboard"
	bj := c.Compact()
	shared := func() Params {
		q := Params{"board": bj}
		for _, k := range []string{"wall", "floor", "fit", "radius", "plateT", "rim", "screw"} {
			if v, ok := p[k]; ok {
				q[k] = v
			}
		}
		q["standoff"] = o.Standoff
		return q
	}
	f3 := func(x, y, zz float64) [3]float64 { return [3]float64{round3(x), round3(y), round3(zz)} }
	zero := [3]float64{}
	base := &Node{Name: "Keyboard case — " + c.Project, Type: "group", Scale: [3]float64{1, 1, 1}, Wirie: "base", Color: "#475569"}
	plate := &Node{Name: "Switch plate — " + c.Project, Type: "group", Scale: [3]float64{1, 1, 1}, Wirie: "lid", Color: "#cbd5e1"}
	base.Children = append(base.Children, newNode("Tray", "wcase_kbtray", shared(), zero, zero, false, ""))
	usePlate := p.S("plate", "Printed plate") != "No plate"
	if usePlate {
		plate.Children = append(plate.Children, newNode("Plate", "wcase_kbplate", shared(), zero, zero, false, ""))
	}

	// standoffs and screws (screws go in from the top, through the PCB)
	m := metrics[c.screwName(o.Screw)]
	pilot := round3(m.D * 0.85)
	insert := m.InsertD > 0 && o.Fixing == "Heat-set insert"
	holeBot := z.floorBot + 1.2
	for _, h := range c.Holes {
		od := round3(math.Max(h.D*2.2, bossOD(math.Max(pilot, h.D))))
		if insert {
			od = math.Max(od, round3(bossOD(m.InsertD)))
		}
		flare := math.Min(1.5, o.Standoff*0.5)
		if gap := c.edgeDist(h.X, h.Y, o.Fit) - od/2; gap < flare+0.6 {
			flare = math.Max(0, gap-0.6)
		}
		base.Children = append(base.Children, newNode("Standoff "+h.Ref, "wcase_standoff", Params{"od": od, "h": round3(o.Standoff), "flare": round3(flare)}, f3(h.X, h.Y, z.floorTop), zero, false, ""))
		if insert {
			base.Children = append(base.Children, newNode("Insert hole "+h.Ref, "cylinder", Params{"d": m.InsertD, "h": round3(m.InsertL + 0.4)}, f3(h.X, h.Y, -m.InsertL-0.3), zero, true, ""))
		} else {
			base.Children = append(base.Children, newNode("Screw hole "+h.Ref, "cylinder", Params{"d": pilot, "h": round3(-holeBot + 0.2)}, f3(h.X, h.Y, holeBot), zero, true, ""))
		}
	}
	if len(c.Holes) == 0 {
		info.Warnings = append(info.Warnings, "The board has no mounting holes, so it only rests on the walls' fit. Add mounting holes in Keebie (Mounting holes) to screw it down.")
	} else {
		t := c.Board.Thickness
		lo, hi, grip := t+1.2*m.D, t-holeBot, t+0.8*m.D
		if insert {
			lo, hi, grip = t+0.6*m.InsertL, t+m.InsertL+0.2, t+0.4*m.InsertL
		}
		L := 0.0
		for _, s := range stdScrewLengths {
			if s >= lo && s <= hi {
				L = s
				break
			}
		}
		for i := len(stdScrewLengths) - 1; L == 0 && i >= 0; i-- {
			if s := stdScrewLengths[i]; s <= hi && s >= grip {
				L = s
			}
		}
		if L > 0 {
			info.Lines = append(info.Lines, fmt.Sprintf("PCB screws: %d × %s × %.0f mm pan head, put in from the top before the plate goes on.", len(c.Holes), c.screwName(o.Screw), L))
		}
		if usePlate {
			info.Lines = append(info.Lines, "The plate has holes over the PCB screws (or a switch cut-out next to them), so you can reach them without taking it apart.")
		}
	}
	if insert {
		info.Lines = append(info.Lines, fmt.Sprintf("Heat-set inserts: %d × %s.", len(c.Holes), c.screwName(o.Screw)))
	}

	// wall openings (USB, TRRS…): find the real wall along the opening's direction
	for _, op := range c.Openings {
		n := op.Normal
		if n == [2]float64{} {
			n = faceNormal2(op.Face)
		}
		sx, sy := op.PartEdge[0], op.PartEdge[1]
		if sx == 0 && sy == 0 {
			sx, sy = op.X, op.Y
		}
		ex, ey := c.kbWallHit(sx-n[0]*2, sy-n[1]*2, n)
		inset := o.Fit + o.Wall/2
		cx, cy := ex+n[0]*inset, ey+n[1]*inset
		w, h, zc := op.Width, op.Height, op.Z
		name := fmt.Sprintf("Opening %s (%s)", op.Ref, op.Type)
		if zc+h/2 > z.wallTop-0.8 { // reaches the top: make it a notch
			lo := zc - h/2
			h = z.wallTop + 1 - lo
			zc = lo + h/2
		}
		pr := Params{"w": round3(w), "h": round3(h), "din": round3(o.Wall/2 + 0.8), "dout": round3(o.Wall/2 + 0.01), "cham": round3(math.Min(0.6, o.Wall/2-0.1)), "r": round3(math.Min(1, math.Min(w, h)/4)), "circle": op.Type == "audio-3.5mm"}
		base.Children = append(base.Children, newNode(name, "wcase_cutout", pr, f3(cx, cy, zc), wallRot(n), true, ""))
	}

	// typing angle: two feet under the back
	outer, _ := c.kbOuter(o)
	x0, y0, x1, y1 := loopBox(outer)
	var avoid []P2
	for _, h := range c.Holes {
		avoid = append(avoid, P2{h.X, h.Y})
	}
	tilt := clampF(p.F("tilt", 6), 0, 15)
	if p.S("feet", kbFeetDefault) == "No feet" {
		if tilt > 0.2 {
			info.Lines = append(info.Lines, "No feet, so the keyboard lies flat (the typing angle needs feet at the back).")
		}
		tilt = 0
	}
	// the front feet (or, lying flat, all of them); the back ones raise it to the typing angle
	which := ""
	if tilt > 0.2 {
		which = "front"
	}
	lift := addFeet(base, outer, z.floorBot, o.Floor, p, avoid, which, kbFeetDefault, &info)
	if tilt > 0.2 {
		depth := y1 - y0
		fh := round3(math.Max(1, depth*math.Tan(tilt*math.Pi/180)) + lift)
		var feet []P2
		for _, fx := range []float64{x0 + 12, x1 - 12} {
			q := P2{fx, y1 - 8}
			for i := 0; i < 40 && !(outer.Contains(q) && loopEdgeDist(outer, q) > 6.5); i++ {
				q = P2{q.X + ((x0+x1)/2-q.X)*0.08, q.Y - 1}
			}
			feet = append(feet, q)
		}
		for i, q := range feet {
			base.Children = append(base.Children, newNode(fmt.Sprintf("Rear foot %d", i+1), "cylinder", Params{"d": 12.0, "h": fh + 0.5}, f3(q.X, q.Y, z.floorBot-fh), zero, false, ""))
			rubberIndentUnder(base, q, z.floorBot-fh, p, fmt.Sprintf("Rubber foot indent (rear %d)", i+1), kbFeetDefault)
		}
		info.Lines = append(info.Lines, fmt.Sprintf("Typing angle about %.1f°: two %.1f mm feet under the back.", tilt, fh))
	}

	info.Outer = [3]float64{round3(x1 - x0), round3(y1 - y0), round3(z.wallTop - z.floorBot)}
	sw := map[string]string{"mx": "MX-style", "choc": "Choc", "alps": "Alps"}[c.Keyboard.Switch]
	nsw, nknob := 0, 0
	for _, k := range c.Keyboard.Cutouts {
		switch k.Kind {
		case "switch":
			nsw++
		case "knob":
			nknob++
		}
	}
	head := fmt.Sprintf("Keyboard case %.0f × %.0f × %.0f mm, open top.", info.Outer[0], info.Outer[1], info.Outer[2])
	lines := []string{head}
	if usePlate {
		pl := fmt.Sprintf("Switch plate %.1f mm thick with %d %s switch cut-outs", z.plateTop-z.plateBot, nsw, sw)
		if nknob > 0 {
			pl += fmt.Sprintf(" and %d knob hole%s", nknob, map[bool]string{true: "s", false: ""}[nknob > 1])
		}
		lines = append(lines, pl+". Print it flat, then clip the switches in before soldering or pushing them into the sockets.")
	}
	info.Lines = append(lines, info.Lines...)
	if c.Keyboard.Switch == "alps" {
		info.Warnings = append(info.Warnings, "Alps plates are usually metal: check the cut-out size fits your switches before printing the whole plate.")
	}
	return []*Node{base, plate}, info
}

// ---------------- robust outline offset for custom (non-rectangular) boards

// rasterOffset grows (or shrinks, g < 0) a polygon by g using a distance
// field, so concave shapes don't fold over themselves.
func rasterOffset(l Loop, g float64) Loop { o, _ := rasterShape(l, nil, g); return o }

// rasterShape traces the outline of l minus the cuts, grown by g. It returns
// the outer outline and any holes left inside it.
func rasterShape(l Loop, cuts []Loop, g float64) (Loop, []Loop) {
	x0, y0, x1, y1 := loopBox(l)
	res := 0.2
	pad := math.Abs(g) + 2
	W := int((x1-x0+2*pad)/res) + 2
	H := int((y1-y0+2*pad)/res) + 2
	if W*H > 6_000_000 { // very big boards: coarser grid
		res = math.Sqrt((x1 - x0 + 2*pad) * (y1 - y0 + 2*pad) / 6_000_000)
		W = int((x1-x0+2*pad)/res) + 2
		H = int((y1-y0+2*pad)/res) + 2
	}
	ox, oy := x0-pad, y0-pad
	inside := make([]bool, W*H)
	// scanline fill
	for j := 0; j < H; j++ {
		y := oy + (float64(j)+0.5)*res
		var xs []float64
		for i := range l {
			a, b := l[i], l[(i+1)%len(l)]
			if (a.Y <= y) != (b.Y <= y) {
				xs = append(xs, a.X+(y-a.Y)/(b.Y-a.Y)*(b.X-a.X))
			}
		}
		sort.Float64s(xs)
		for k := 0; k+1 < len(xs); k += 2 {
			i0 := int(math.Ceil((xs[k]-ox)/res - 0.5))
			i1 := int(math.Floor((xs[k+1]-ox)/res - 0.5))
			for i := max(0, i0); i <= min(W-1, i1); i++ {
				inside[j*W+i] = true
			}
		}
	}
	for _, cl := range cuts {
		for j := 0; j < H; j++ {
			for i := 0; i < W; i++ {
				if inside[j*W+i] && cl.Contains(P2{ox + (float64(i)+0.5)*res, oy + (float64(j)+0.5)*res}) {
					inside[j*W+i] = false
				}
			}
		}
	}
	if len(cuts) > 0 { // open up slivers thinner than 2 mm (they'd break off)
		const r = 1.0
		dIn := edt(inside, W, H, false)
		for k := range inside {
			inside[k] = inside[k] && (dIn[k]-0.5)*res > r
		}
		g += r
	}
	// signed distance (in cells): + outside, - inside
	dOut := edt(inside, W, H, true) // distance from outside cells to the nearest inside cell
	dIn := edt(inside, W, H, false) // distance from inside cells to the nearest outside cell
	f := make([]float64, W*H)
	gc := g / res
	for k := range f {
		if inside[k] {
			f[k] = -(dIn[k] - 0.5) - gc
		} else {
			f[k] = (dOut[k] - 0.5) - gc
		}
	}
	loops := marchLoops(f, W, H)
	conv := func(loop []P2) Loop {
		out := make(Loop, len(loop))
		for i, q := range loop {
			out[i] = P2{ox + (q.X+0.5)*res, oy + (q.Y+0.5)*res}
		}
		return simplifyLoop(out, 0.04)
	}
	best, bestA := -1, 0.0
	for i, lp := range loops {
		if a := math.Abs(Loop(lp).Area()); a > bestA {
			best, bestA = i, a
		}
	}
	if best < 0 {
		return l, nil
	}
	outer := conv(loops[best])
	sgn := Loop(loops[best]).Area() > 0
	var holes []Loop
	for i, lp := range loops {
		if i == best || (Loop(lp).Area() > 0) == sgn || math.Abs(Loop(lp).Area())*res*res < 1 {
			continue
		}
		holes = append(holes, conv(lp))
	}
	if outer.Area() < 0 {
		outer = outer.Reversed()
	}
	return outer, holes
}

// edt: Euclidean distance (in cells) from each cell to the nearest cell whose
// inside value equals target (Felzenszwalb & Huttenlocher, two passes).
func edt(inside []bool, W, H int, target bool) []float64 {
	const inf = 1e20
	d := make([]float64, W*H)
	for k, v := range inside {
		if v == target {
			d[k] = 0
		} else {
			d[k] = inf
		}
	}
	n := max(W, H)
	fbuf, dbuf := make([]float64, n), make([]float64, n)
	v, zz := make([]int, n), make([]float64, n+1)
	dt1 := func(cnt int) {
		k := 0
		v[0] = 0
		zz[0], zz[1] = -inf, inf
		for q := 1; q < cnt; q++ {
			s := ((fbuf[q] + float64(q*q)) - (fbuf[v[k]] + float64(v[k]*v[k]))) / float64(2*q-2*v[k])
			for s <= zz[k] {
				k--
				s = ((fbuf[q] + float64(q*q)) - (fbuf[v[k]] + float64(v[k]*v[k]))) / float64(2*q-2*v[k])
			}
			k++
			v[k] = q
			zz[k], zz[k+1] = s, inf
		}
		k = 0
		for q := 0; q < cnt; q++ {
			for zz[k+1] < float64(q) {
				k++
			}
			dq := float64(q - v[k])
			dbuf[q] = dq*dq + fbuf[v[k]]
		}
	}
	for x := 0; x < W; x++ {
		for y := 0; y < H; y++ {
			fbuf[y] = d[y*W+x]
		}
		dt1(H)
		for y := 0; y < H; y++ {
			d[y*W+x] = dbuf[y]
		}
	}
	for y := 0; y < H; y++ {
		copy(fbuf[:W], d[y*W:y*W+W])
		dt1(W)
		copy(d[y*W:y*W+W], dbuf[:W])
	}
	for k := range d {
		d[k] = math.Sqrt(d[k])
	}
	return d
}

// marchLoops traces the f = 0 contour (marching squares) and returns every
// closed loop, in cell coordinates (outlines one way round, holes the other).
func marchLoops(f []float64, W, H int) [][]P2 {
	type ek struct{ x, y, dir int } // edge: horizontal (dir 0) from (x,y)-(x+1,y), vertical (dir 1) (x,y)-(x,y+1)
	pt := func(e ek) P2 {
		a := f[e.y*W+e.x]
		var b float64
		if e.dir == 0 {
			b = f[e.y*W+e.x+1]
		} else {
			b = f[(e.y+1)*W+e.x]
		}
		t := 0.5
		if a != b {
			t = clampF(a/(a-b), 0, 1)
		}
		if e.dir == 0 {
			return P2{float64(e.x) + t, float64(e.y)}
		}
		return P2{float64(e.x), float64(e.y) + t}
	}
	next := map[ek]ek{}
	for y := 0; y+1 < H; y++ {
		for x := 0; x+1 < W; x++ {
			a, b, cc, d := f[y*W+x] < 0, f[y*W+x+1] < 0, f[(y+1)*W+x+1] < 0, f[(y+1)*W+x] < 0
			idx := 0
			if a {
				idx |= 1
			}
			if b {
				idx |= 2
			}
			if cc {
				idx |= 4
			}
			if d {
				idx |= 8
			}
			top, right, bot, left := ek{x, y, 0}, ek{x + 1, y, 1}, ek{x, y + 1, 0}, ek{x, y, 1}
			// segments oriented with "inside" on the left
			switch idx {
			case 1:
				next[left] = top
			case 2:
				next[top] = right
			case 3:
				next[left] = right
			case 4:
				next[right] = bot
			case 5:
				next[left] = top
				next[right] = bot
			case 6:
				next[top] = bot
			case 7:
				next[left] = bot
			case 8:
				next[bot] = left
			case 9:
				next[bot] = top
			case 10:
				next[top] = right
				next[bot] = left
			case 11:
				next[bot] = right
			case 12:
				next[right] = left
			case 13:
				next[right] = top
			case 14:
				next[top] = left
			}
		}
	}
	var all [][]P2
	seen := map[ek]bool{}
	for s := range next {
		if seen[s] {
			continue
		}
		var loop []P2
		e := s
		for i := 0; i < 4*W*H; i++ {
			if seen[e] {
				break
			}
			seen[e] = true
			loop = append(loop, pt(e))
			n, ok := next[e]
			if !ok {
				break
			}
			e = n
		}
		if len(loop) > 2 {
			all = append(all, loop)
		}
	}
	return all
}

// simplifyLoop removes points closer than tol to the line through their
// neighbours (Douglas–Peucker on a closed loop).
func simplifyLoop(l Loop, tol float64) Loop {
	if len(l) < 8 {
		return l
	}
	// split at the two farthest-apart points
	far := 0
	for i := range l {
		if math.Hypot(l[i].X-l[0].X, l[i].Y-l[0].Y) > math.Hypot(l[far].X-l[0].X, l[far].Y-l[0].Y) {
			far = i
		}
	}
	var dp func(pts []P2) []P2
	dp = func(pts []P2) []P2 {
		if len(pts) < 3 {
			return pts
		}
		a, b := pts[0], pts[len(pts)-1]
		idx, md := 0, 0.0
		for i := 1; i < len(pts)-1; i++ {
			if d := segDist(pts[i], a, b); d > md {
				idx, md = i, d
			}
		}
		if md <= tol {
			return []P2{a, b}
		}
		l1 := dp(pts[:idx+1])
		l2 := dp(pts[idx:])
		return append(l1[:len(l1)-1], l2...)
	}
	first := dp(append([]P2(nil), l[:far+1]...))
	second := dp(append(append([]P2(nil), l[far:]...), l[0]))
	out := append(first[:len(first)-1], second[:len(second)-1]...)
	return Loop(out)
}

func init() {
	kbShape := func(typ, name string, gen func(c *WirieCase, o CaseOpts, p Params) *Mesh) {
		register(&ShapeDef{Type: typ, Name: name, Category: "Wirie", Hidden: true, Fixed: true, Color: "#64748b",
			Desc: "Part of a keyboard case made for a Keebie board.", Params: wirieParams(),
			Gen: func(p Params) (*Mesh, error) {
				c, err := wirieFromParams(p)
				if err != nil {
					return nil, err
				}
				if !c.isKeyboard() {
					return nil, fmt.Errorf("this board has no keyboard data")
				}
				return gen(c, caseOpts(c, p), p), nil
			}})
	}
	kbShape("wcase_kbtray", "Keyboard case tray", func(c *WirieCase, o CaseOpts, p Params) *Mesh { return c.kbTrayMesh(o, p) })
	kbShape("wcase_kbplate", "Switch plate", func(c *WirieCase, o CaseOpts, p Params) *Mesh { return c.kbPlateMesh(o, p) })
}
