package geom

// BSP-tree constructive solid geometry (after Evan Wallace's csg.js, MIT),
// with bounding-box culling so only polygons near the other solid are clipped.

const csgEps = 1e-5

type plane struct {
	n Vec3
	w float64
}

type poly struct {
	v []Vec3
	p plane
}

func planeFrom(a, b, c Vec3) (plane, bool) {
	n := b.Sub(a).Cross(c.Sub(a))
	l := n.Len()
	if l < 1e-12 {
		return plane{}, false
	}
	n = n.Mul(1 / l)
	return plane{n, n.Dot(a)}, true
}

func (p plane) flip() plane { return plane{p.n.Neg(), -p.w} }

func (pl poly) flip() poly {
	n := len(pl.v)
	v := make([]Vec3, n)
	for i := range pl.v {
		v[i] = pl.v[n-1-i]
	}
	return poly{v, pl.p.flip()}
}

const (
	coplanar = 0
	front    = 1
	back     = 2
	spanning = 3
)

func (p plane) split(pl poly, cf, cb, f, b *[]poly) {
	ptype := 0
	types := make([]int, len(pl.v))
	for i, v := range pl.v {
		t := p.n.Dot(v) - p.w
		tp := coplanar
		if t < -csgEps {
			tp = back
		} else if t > csgEps {
			tp = front
		}
		ptype |= tp
		types[i] = tp
	}
	switch ptype {
	case coplanar:
		if p.n.Dot(pl.p.n) > 0 {
			*cf = append(*cf, pl)
		} else {
			*cb = append(*cb, pl)
		}
	case front:
		*f = append(*f, pl)
	case back:
		*b = append(*b, pl)
	default:
		var fv, bv []Vec3
		n := len(pl.v)
		for i := 0; i < n; i++ {
			j := (i + 1) % n
			ti, tj := types[i], types[j]
			vi, vj := pl.v[i], pl.v[j]
			if ti != back {
				fv = append(fv, vi)
			}
			if ti != front {
				bv = append(bv, vi)
			}
			if (ti | tj) == spanning {
				t := (p.w - p.n.Dot(vi)) / p.n.Dot(vj.Sub(vi))
				v := vi.Lerp(vj, t)
				fv = append(fv, v)
				bv = append(bv, v)
			}
		}
		if len(fv) >= 3 {
			*f = append(*f, poly{fv, pl.p})
		}
		if len(bv) >= 3 {
			*b = append(*b, poly{bv, pl.p})
		}
	}
}

type bspNode struct {
	pl          *plane
	front, back *bspNode
	polys       []poly
}

func newBSP(polys []poly) *bspNode {
	n := &bspNode{}
	n.build(polys)
	return n
}

func (n *bspNode) invert() {
	// iterative to avoid deep recursion
	stack := []*bspNode{n}
	for len(stack) > 0 {
		c := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for i := range c.polys {
			c.polys[i] = c.polys[i].flip()
		}
		if c.pl != nil {
			f := c.pl.flip()
			c.pl = &f
		}
		c.front, c.back = c.back, c.front
		if c.front != nil {
			stack = append(stack, c.front)
		}
		if c.back != nil {
			stack = append(stack, c.back)
		}
	}
}

func (n *bspNode) clipPolygons(polys []poly) []poly {
	if n.pl == nil {
		out := make([]poly, len(polys))
		copy(out, polys)
		return out
	}
	var f, b []poly
	for _, p := range polys {
		n.pl.split(p, &f, &b, &f, &b)
	}
	if n.front != nil {
		f = n.front.clipPolygons(f)
	}
	if n.back != nil {
		b = n.back.clipPolygons(b)
	} else {
		b = nil
	}
	return append(f, b...)
}

func (n *bspNode) clipTo(o *bspNode) {
	stack := []*bspNode{n}
	for len(stack) > 0 {
		c := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		c.polys = o.clipPolygons(c.polys)
		if c.front != nil {
			stack = append(stack, c.front)
		}
		if c.back != nil {
			stack = append(stack, c.back)
		}
	}
}

func (n *bspNode) all() []poly {
	var out []poly
	stack := []*bspNode{n}
	for len(stack) > 0 {
		c := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		out = append(out, c.polys...)
		if c.front != nil {
			stack = append(stack, c.front)
		}
		if c.back != nil {
			stack = append(stack, c.back)
		}
	}
	return out
}

func (n *bspNode) build(polys []poly) {
	type job struct {
		n *bspNode
		p []poly
	}
	stack := []job{{n, polys}}
	for len(stack) > 0 {
		j := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if len(j.p) == 0 {
			continue
		}
		c := j.n
		if c.pl == nil {
			// choose a splitter: sample a few candidates, prefer fewest spans
			c.pl = pickSplitter(j.p)
		}
		var f, b []poly
		for _, p := range j.p {
			c.pl.split(p, &c.polys, &c.polys, &f, &b)
		}
		if len(f) > 0 {
			if c.front == nil {
				c.front = &bspNode{}
			}
			stack = append(stack, job{c.front, f})
		}
		if len(b) > 0 {
			if c.back == nil {
				c.back = &bspNode{}
			}
			stack = append(stack, job{c.back, b})
		}
	}
}

func pickSplitter(ps []poly) *plane {
	if len(ps) < 24 {
		p := ps[0].p
		return &p
	}
	best := ps[0].p
	bestScore := 1 << 30
	step := len(ps) / 7
	for k := 0; k < 7; k++ {
		cand := ps[k*step].p
		spans, fr, bk := 0, 0, 0
		// evaluate on a sample
		sstep := len(ps)/64 + 1
		for i := 0; i < len(ps); i += sstep {
			t := 0
			for _, v := range ps[i].v {
				d := cand.n.Dot(v) - cand.w
				if d < -csgEps {
					t |= back
				} else if d > csgEps {
					t |= front
				}
			}
			switch t {
			case spanning:
				spans++
			case front:
				fr++
			case back:
				bk++
			}
		}
		bal := fr - bk
		if bal < 0 {
			bal = -bal
		}
		score := spans*8 + bal
		if score < bestScore {
			bestScore = score
			best = cand
		}
	}
	return &best
}

// ---- mesh <-> polygons ----

func toPolys(m *Mesh) []poly {
	out := make([]poly, 0, len(m.Tris))
	for _, t := range m.Tris {
		p, ok := planeFrom(t[0], t[1], t[2])
		if !ok {
			continue
		}
		out = append(out, poly{[]Vec3{t[0], t[1], t[2]}, p})
	}
	return out
}

func fromPolys(ps []poly) *Mesh {
	m := &Mesh{}
	for _, p := range ps {
		for i := 1; i+1 < len(p.v); i++ {
			m.Add(p.v[0], p.v[i], p.v[i+1])
		}
	}
	return m
}

type box3 struct{ lo, hi Vec3 }

func boundsOf(ps []poly) box3 {
	if len(ps) == 0 {
		return box3{}
	}
	b := box3{ps[0].v[0], ps[0].v[0]}
	for _, p := range ps {
		for _, v := range p.v {
			b.lo, b.hi = b.lo.Min(v), b.hi.Max(v)
		}
	}
	return b
}

func (b box3) overlaps(o box3) bool {
	e := 1e-4
	return b.lo.X <= o.hi.X+e && b.hi.X >= o.lo.X-e &&
		b.lo.Y <= o.hi.Y+e && b.hi.Y >= o.lo.Y-e &&
		b.lo.Z <= o.hi.Z+e && b.hi.Z >= o.lo.Z-e
}

func polyBox(p poly) box3 {
	b := box3{p.v[0], p.v[0]}
	for _, v := range p.v[1:] {
		b.lo, b.hi = b.lo.Min(v), b.hi.Max(v)
	}
	return b
}

// partition splits polygons into those touching box b and those clearly outside it.
func partition(ps []poly, b box3) (in, out []poly) {
	for _, p := range ps {
		if polyBox(p).overlaps(b) {
			in = append(in, p)
		} else {
			out = append(out, p)
		}
	}
	return
}

func flipAll(ps []poly) []poly {
	out := make([]poly, len(ps))
	for i, p := range ps {
		out[i] = p.flip()
	}
	return out
}

// Union of two closed meshes.
func Union(a, b *Mesh) *Mesh {
	if a == nil || len(a.Tris) == 0 {
		return b.Clone()
	}
	if b == nil || len(b.Tris) == 0 {
		return a.Clone()
	}
	A, B := toPolys(a), toPolys(b)
	ba, bb := boundsOf(A), boundsOf(B)
	if !ba.overlaps(bb) {
		m := a.Clone()
		m.Append(b)
		return m
	}
	Ain, Aout := partition(A, bb)
	Bin, Bout := partition(B, ba)
	at, bt := newBSP(A), newBSP(B)
	ra := bt.clipPolygons(Ain)
	rb := at.clipPolygons(Bin)
	rb = flipAll(at.clipPolygons(flipAll(rb)))
	res := append(append(append(ra, rb...), Aout...), Bout...)
	return fromPolys(res)
}

// Subtract b from a.
func Subtract(a, b *Mesh) *Mesh {
	if a == nil || len(a.Tris) == 0 {
		return &Mesh{}
	}
	if b == nil || len(b.Tris) == 0 {
		return a.Clone()
	}
	A, B := toPolys(a), toPolys(b)
	ba, bb := boundsOf(A), boundsOf(B)
	if !ba.overlaps(bb) {
		return a.Clone()
	}
	Ain, Aout := partition(A, bb)
	Bin, _ := partition(B, ba)
	bt := newBSP(B)
	ait := newBSP(A)
	ait.invert()
	// A parts outside B
	ra := flipAll(bt.clipPolygons(flipAll(Ain)))
	// keep only the B parts inside A, removing faces coplanar with A
	rb := ait.clipPolygons(Bin)
	rb = flipAll(ait.clipPolygons(flipAll(rb)))
	res := append(append(ra, flipAll(rb)...), Aout...)
	return fromPolys(res)
}

// Intersect keeps only the volume common to both.
func Intersect(a, b *Mesh) *Mesh {
	if a == nil || b == nil || len(a.Tris) == 0 || len(b.Tris) == 0 {
		return &Mesh{}
	}
	A, B := toPolys(a), toPolys(b)
	ba, bb := boundsOf(A), boundsOf(B)
	if !ba.overlaps(bb) {
		return &Mesh{}
	}
	Ain, _ := partition(A, bb)
	Bin, _ := partition(B, ba)
	aInv := newBSP(A)
	aInv.invert()
	bInv := newBSP(B)
	bInv.invert()
	b1 := aInv.clipPolygons(Bin)          // B inside A
	a1 := bInv.clipPolygons(flipAll(Ain)) // A inside B (flipped)
	b2 := aInv.clipPolygons(flipAll(b1))  // drop coplanar duplicates
	return fromPolys(append(flipAll(a1), flipAll(b2)...))
}

// UnionAll merges many meshes, combining disjoint ones cheaply.
func UnionAll(ms []*Mesh) *Mesh {
	var res *Mesh
	for _, m := range ms {
		if m == nil || len(m.Tris) == 0 {
			continue
		}
		if res == nil {
			res = m.Clone()
			continue
		}
		res = Union(res, m)
	}
	if res == nil {
		return &Mesh{}
	}
	return res
}
