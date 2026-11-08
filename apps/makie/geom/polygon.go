package geom

import (
	"math"
	"sort"
)

// P2 is a 2D point.
type P2 struct{ X, Y float64 }

// Loop is a closed 2D polyline (no repeated end point).
type Loop []P2

func (l Loop) Area() float64 {
	var a float64
	for i := range l {
		j := (i + 1) % len(l)
		a += l[i].X*l[j].Y - l[j].X*l[i].Y
	}
	return a / 2
}

func (l Loop) Reversed() Loop {
	r := make(Loop, len(l))
	for i := range l {
		r[i] = l[len(l)-1-i]
	}
	return r
}

// Clean removes consecutive duplicates and collinear points.
func (l Loop) Clean() Loop {
	var out Loop
	for i, p := range l {
		q := l[(i+1)%len(l)]
		if math.Hypot(p.X-q.X, p.Y-q.Y) > 1e-7 {
			out = append(out, p)
		}
	}
	if len(out) < 3 {
		return out
	}
	var res Loop
	for i := range out {
		a := out[(i+len(out)-1)%len(out)]
		b := out[i]
		c := out[(i+1)%len(out)]
		cr := (b.X-a.X)*(c.Y-a.Y) - (b.Y-a.Y)*(c.X-a.X)
		if math.Abs(cr) > 1e-10 {
			res = append(res, b)
		}
	}
	return res
}

func (l Loop) Contains(p P2) bool {
	in := false
	for i, j := 0, len(l)-1; i < len(l); j, i = i, i+1 {
		a, b := l[i], l[j]
		if (a.Y > p.Y) != (b.Y > p.Y) && p.X < (b.X-a.X)*(p.Y-a.Y)/(b.Y-a.Y)+a.X {
			in = !in
		}
	}
	return in
}

// Shape is an outer boundary (CCW) plus holes (CW).
type Shape struct {
	Outer Loop
	Holes []Loop
}

// Normalize enforces CCW outer and CW holes.
func (s Shape) Normalize() Shape {
	o := s.Outer.Clean()
	if o.Area() < 0 {
		o = o.Reversed()
	}
	var hs []Loop
	for _, h := range s.Holes {
		h = h.Clean()
		if len(h) < 3 {
			continue
		}
		if h.Area() > 0 {
			h = h.Reversed()
		}
		hs = append(hs, h)
	}
	return Shape{o, hs}
}

// ShapesFromLoops groups arbitrary loops into shapes using even/odd nesting.
func ShapesFromLoops(loops []Loop) []Shape {
	type L struct {
		l     Loop
		area  float64
		depth int
		par   int
	}
	var ls []L
	for _, l := range loops {
		l = l.Clean()
		if len(l) < 3 || math.Abs(l.Area()) < 1e-9 {
			continue
		}
		ls = append(ls, L{l: l, area: math.Abs(l.Area()), par: -1})
	}
	sort.Slice(ls, func(i, j int) bool { return ls[i].area > ls[j].area })
	for i := range ls {
		// find smallest containing loop
		probe := ls[i].l[0]
		for j := i - 1; j >= 0; j-- {
			if ls[j].l.Contains(probe) {
				ls[i].par = j
				ls[i].depth = ls[j].depth + 1
				break
			}
		}
	}
	idx := map[int]int{}
	var out []Shape
	for i, l := range ls {
		if l.depth%2 == 0 {
			idx[i] = len(out)
			out = append(out, Shape{Outer: l.l})
		}
	}
	for _, l := range ls {
		if l.depth%2 == 1 {
			if k, ok := idx[l.par]; ok {
				out[k].Holes = append(out[k].Holes, l.l)
			}
		}
	}
	for i := range out {
		out[i] = out[i].Normalize()
	}
	return out
}

// Triangulate returns triangles (indices into the returned point list) for a shape with holes.
func Triangulate(s Shape) ([]P2, [][3]int) {
	s = s.Normalize()
	if len(s.Outer) < 3 {
		return nil, nil
	}
	pts := append([]P2{}, s.Outer...)
	ring := make([]int, len(s.Outer))
	for i := range ring {
		ring[i] = i
	}
	// bridge holes, rightmost first
	holes := append([]Loop{}, s.Holes...)
	sort.Slice(holes, func(i, j int) bool { return maxX(holes[i]) > maxX(holes[j]) })
	for _, h := range holes {
		base := len(pts)
		pts = append(pts, h...)
		// rightmost hole vertex
		hi := 0
		for i := range h {
			if h[i].X > h[hi].X {
				hi = i
			}
		}
		M := h[hi]
		bridge := findBridge(pts, ring, M)
		if bridge < 0 {
			continue
		}
		// splice: ring[..bridge], hole from hi around back to hi, ring[bridge], ...
		var nr []int
		nr = append(nr, ring[:bridge+1]...)
		for k := 0; k <= len(h); k++ {
			nr = append(nr, base+(hi+k)%len(h))
		}
		nr = append(nr, ring[bridge])
		nr = append(nr, ring[bridge+1:]...)
		ring = nr
	}
	return pts, earClip(pts, ring)
}

func maxX(l Loop) float64 {
	m := -math.MaxFloat64
	for _, p := range l {
		m = math.Max(m, p.X)
	}
	return m
}

func cross2(a, b, c P2) float64 { return (b.X-a.X)*(c.Y-a.Y) - (b.Y-a.Y)*(c.X-a.X) }

func segIntersect(a, b, c, d P2) bool {
	d1 := cross2(c, d, a)
	d2 := cross2(c, d, b)
	d3 := cross2(a, b, c)
	d4 := cross2(a, b, d)
	return ((d1 > 1e-12 && d2 < -1e-12) || (d1 < -1e-12 && d2 > 1e-12)) &&
		((d3 > 1e-12 && d4 < -1e-12) || (d3 < -1e-12 && d4 > 1e-12))
}

// findBridge returns the ring position of a vertex visible from M.
func findBridge(pts []P2, ring []int, M P2) int {
	type cand struct {
		pos int
		d   float64
	}
	var cs []cand
	for i, idx := range ring {
		p := pts[idx]
		if p.X < M.X-1e-9 {
			continue
		}
		cs = append(cs, cand{i, (p.X-M.X)*(p.X-M.X) + (p.Y-M.Y)*(p.Y-M.Y)})
	}
	if len(cs) == 0 {
		for i, idx := range ring {
			p := pts[idx]
			cs = append(cs, cand{i, (p.X-M.X)*(p.X-M.X) + (p.Y-M.Y)*(p.Y-M.Y)})
		}
	}
	sort.Slice(cs, func(i, j int) bool { return cs[i].d < cs[j].d })
	for _, c := range cs {
		p := pts[ring[c.pos]]
		ok := true
		for i := range ring {
			a := pts[ring[i]]
			b := pts[ring[(i+1)%len(ring)]]
			if segIntersect(M, p, a, b) {
				ok = false
				break
			}
		}
		if ok {
			return c.pos
		}
	}
	if len(cs) > 0 {
		return cs[0].pos
	}
	return -1
}

func pointInTri(p, a, b, c P2) bool {
	return cross2(a, b, p) >= -1e-12 && cross2(b, c, p) >= -1e-12 && cross2(c, a, p) >= -1e-12
}

func earClip(pts []P2, ring []int) [][3]int {
	var out [][3]int
	idx := append([]int{}, ring...)
	guard := 0
	for len(idx) > 3 && guard < 100000 {
		guard++
		n := len(idx)
		found := false
		for i := 0; i < n; i++ {
			ia, ib, ic := idx[(i+n-1)%n], idx[i], idx[(i+1)%n]
			a, b, c := pts[ia], pts[ib], pts[ic]
			cr := cross2(a, b, c)
			if cr <= 1e-14 {
				if math.Abs(cr) <= 1e-14 && (ia == ic || samePt(a, c) || samePt(a, b) || samePt(b, c)) {
					// degenerate: drop vertex
					idx = append(idx[:i], idx[i+1:]...)
					found = true
					break
				}
				continue
			}
			ear := true
			for j := 0; j < n; j++ {
				ij := idx[j]
				if ij == ia || ij == ib || ij == ic {
					continue
				}
				p := pts[ij]
				if samePt(p, a) || samePt(p, b) || samePt(p, c) {
					continue
				}
				if pointInTri(p, a, b, c) {
					ear = false
					break
				}
			}
			if ear {
				out = append(out, [3]int{ia, ib, ic})
				idx = append(idx[:i], idx[i+1:]...)
				found = true
				break
			}
		}
		if !found {
			// fall back: clip the most convex vertex to guarantee progress
			best, bv := 0, -math.MaxFloat64
			for i := 0; i < n; i++ {
				ia, ib, ic := idx[(i+n-1)%n], idx[i], idx[(i+1)%n]
				cr := cross2(pts[ia], pts[ib], pts[ic])
				if cr > bv {
					bv, best = cr, i
				}
			}
			ia, ib, ic := idx[(best+n-1)%n], idx[best], idx[(best+1)%n]
			if bv > 0 {
				out = append(out, [3]int{ia, ib, ic})
			}
			idx = append(idx[:best], idx[best+1:]...)
		}
	}
	if len(idx) == 3 && cross2(pts[idx[0]], pts[idx[1]], pts[idx[2]]) > 1e-14 {
		out = append(out, [3]int{idx[0], idx[1], idx[2]})
	}
	return out
}

func samePt(a, b P2) bool { return math.Abs(a.X-b.X) < 1e-9 && math.Abs(a.Y-b.Y) < 1e-9 }

// Frame maps 2D profile coordinates into 3D: O + u*U + v*V.
type Frame struct{ O, U, Vv Vec3 }

func (f Frame) At(p P2) Vec3 { return f.O.Add(f.U.Mul(p.X)).Add(f.Vv.Mul(p.Y)) }

// Sweep moves a 2D shape through frames. closed=true joins last frame to first (no caps).
func Sweep(s Shape, frames []Frame, closed bool) *Mesh {
	s = s.Normalize()
	m := &Mesh{}
	if len(s.Outer) < 3 || len(frames) < 2 {
		return m
	}
	loops := append([]Loop{s.Outer}, s.Holes...)
	nf := len(frames)
	segs := nf - 1
	if closed {
		segs = nf
	}
	for _, l := range loops {
		n := len(l)
		for k := 0; k < segs; k++ {
			f0, f1 := frames[k], frames[(k+1)%nf]
			for i := 0; i < n; i++ {
				p, q := l[i], l[(i+1)%n]
				a, b := f0.At(p), f0.At(q)
				c, d := f1.At(q), f1.At(p)
				m.AddQuad(a, b, c, d)
			}
		}
	}
	if !closed {
		pts, tris := Triangulate(s)
		f0, f1 := frames[0], frames[nf-1]
		for _, t := range tris {
			m.Add(f0.At(pts[t[0]]), f0.At(pts[t[2]]), f0.At(pts[t[1]]))
			m.Add(f1.At(pts[t[0]]), f1.At(pts[t[1]]), f1.At(pts[t[2]]))
		}
	}
	return m.Orient()
}

// Extrude a shape along Z with optional twist (degrees) and top scale.
func Extrude(s Shape, h, twistDeg, topScale float64, layers int) *Mesh {
	if layers < 1 {
		layers = 1
	}
	if twistDeg == 0 && topScale == 1 {
		layers = 1
	}
	frames := make([]Frame, layers+1)
	for i := 0; i <= layers; i++ {
		t := float64(i) / float64(layers)
		a := twistDeg * t * math.Pi / 180
		sc := 1 + (topScale-1)*t
		sn, cs := math.Sincos(a)
		frames[i] = Frame{V(0, 0, h*t), V(cs*sc, sn*sc, 0), V(-sn*sc, cs*sc, 0)}
	}
	return Sweep(s, frames, false)
}

// Revolve a profile (x = radius, y = height) around the Z axis.
func Revolve(s Shape, angleDeg float64, segs int) *Mesh {
	full := angleDeg >= 359.999
	if segs < 3 {
		segs = 3
	}
	n := int(math.Ceil(float64(segs) * angleDeg / 360))
	if n < 1 {
		n = 1
	}
	var frames []Frame
	count := n + 1
	if full {
		count = n
	}
	for i := 0; i < count; i++ {
		a := angleDeg * math.Pi / 180 * float64(i) / float64(n)
		sn, cs := math.Sincos(a)
		frames = append(frames, Frame{Vec3{}, V(cs, sn, 0), V(0, 0, 1)})
	}
	return Sweep(s, frames, full)
}

// RevolveProfile revolves an open polyline from the axis to the axis (r,z points).
func RevolveProfile(pts []P2, segs int) *Mesh {
	return Revolve(Shape{Outer: Loop(pts)}, 360, segs)
}

// Circle returns a polygon approximating a circle.
func Circle(r float64, n int) Loop {
	l := make(Loop, n)
	for i := range l {
		a := 2 * math.Pi * float64(i) / float64(n)
		l[i] = P2{r * math.Cos(a), r * math.Sin(a)}
	}
	return l
}

func NGon(r float64, n int, rotDeg float64) Loop {
	l := make(Loop, n)
	for i := range l {
		a := 2*math.Pi*float64(i)/float64(n) + rotDeg*math.Pi/180
		l[i] = P2{r * math.Cos(a), r * math.Sin(a)}
	}
	return l
}

// RoundedRect returns a rectangle centred on the origin with rounded corners.
func RoundedRect(w, d, r float64, segs int) Loop {
	r = math.Max(0, math.Min(r, math.Min(w, d)/2-1e-4))
	if r < 1e-4 {
		return Loop{{-w / 2, -d / 2}, {w / 2, -d / 2}, {w / 2, d / 2}, {-w / 2, d / 2}}
	}
	var l Loop
	cs := [][3]float64{{w/2 - r, -d/2 + r, -90}, {w/2 - r, d/2 - r, 0}, {-w/2 + r, d/2 - r, 90}, {-w/2 + r, -d/2 + r, 180}}
	for _, c := range cs {
		for i := 0; i <= segs; i++ {
			a := (c[2] + 90*float64(i)/float64(segs)) * math.Pi / 180
			l = append(l, P2{c[0] + r*math.Cos(a), c[1] + r*math.Sin(a)})
		}
	}
	return l
}

// SmoothLoop subdivides a polygon with a Catmull-Rom spline.
func SmoothLoop(l Loop, per int, closed bool) Loop {
	n := len(l)
	if n < 3 || per < 2 {
		return l
	}
	get := func(i int) P2 {
		if closed {
			return l[(i%n+n)%n]
		}
		if i < 0 {
			i = 0
		}
		if i >= n {
			i = n - 1
		}
		return l[i]
	}
	var out Loop
	last := n
	if !closed {
		last = n - 1
	}
	for i := 0; i < last; i++ {
		p0, p1, p2, p3 := get(i-1), get(i), get(i+1), get(i+2)
		for k := 0; k < per; k++ {
			t := float64(k) / float64(per)
			t2, t3 := t*t, t*t*t
			x := 0.5 * (2*p1.X + (-p0.X+p2.X)*t + (2*p0.X-5*p1.X+4*p2.X-p3.X)*t2 + (-p0.X+3*p1.X-3*p2.X+p3.X)*t3)
			y := 0.5 * (2*p1.Y + (-p0.Y+p2.Y)*t + (2*p0.Y-5*p1.Y+4*p2.Y-p3.Y)*t2 + (-p0.Y+3*p1.Y-3*p2.Y+p3.Y)*t3)
			out = append(out, P2{x, y})
		}
	}
	if !closed {
		out = append(out, l[n-1])
	}
	return out
}
