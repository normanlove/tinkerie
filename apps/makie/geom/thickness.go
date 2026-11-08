package geom

import (
	"math"
	"math/rand"
	"sort"
)

// ---------- BVH for fast ray casting ----------

type bvhNode struct {
	lo, hi      Vec3
	left, right int // child indices (-1 = leaf)
	start, n    int // triangle range for leaves
}

type BVH struct {
	tris  []Tri
	nodes []bvhNode
}

func NewBVH(m *Mesh) *BVH {
	b := &BVH{tris: append([]Tri(nil), m.Tris...)}
	if len(b.tris) > 0 {
		b.build(0, len(b.tris))
	}
	return b
}

func triBox(t Tri) (Vec3, Vec3) {
	return t[0].Min(t[1]).Min(t[2]), t[0].Max(t[1]).Max(t[2])
}

func (b *BVH) build(s, e int) int {
	lo, hi := triBox(b.tris[s])
	for i := s + 1; i < e; i++ {
		l, h := triBox(b.tris[i])
		lo, hi = lo.Min(l), hi.Max(h)
	}
	idx := len(b.nodes)
	b.nodes = append(b.nodes, bvhNode{lo: lo, hi: hi, left: -1, right: -1, start: s, n: e - s})
	if e-s <= 6 {
		return idx
	}
	ext := hi.Sub(lo)
	axis := 0
	if ext.Y > ext.X && ext.Y >= ext.Z {
		axis = 1
	} else if ext.Z > ext.X && ext.Z > ext.Y {
		axis = 2
	}
	c := func(t Tri) float64 {
		p := t[0].Add(t[1]).Add(t[2])
		return [3]float64{p.X, p.Y, p.Z}[axis]
	}
	sub := b.tris[s:e]
	sort.Slice(sub, func(i, j int) bool { return c(sub[i]) < c(sub[j]) })
	mid := (s + e) / 2
	l := b.build(s, mid)
	r := b.build(mid, e)
	b.nodes[idx].left, b.nodes[idx].right = l, r
	return idx
}

func rayBoxHit(o, inv, lo, hi Vec3, tmax float64) bool {
	t0, t1 := 0.0, tmax
	for a := 0; a < 3; a++ {
		oa := [3]float64{o.X, o.Y, o.Z}[a]
		ia := [3]float64{inv.X, inv.Y, inv.Z}[a]
		la := [3]float64{lo.X, lo.Y, lo.Z}[a]
		ha := [3]float64{hi.X, hi.Y, hi.Z}[a]
		ta, tb := (la-oa)*ia, (ha-oa)*ia
		if ta > tb {
			ta, tb = tb, ta
		}
		t0, t1 = math.Max(t0, ta), math.Min(t1, tb)
		if t0 > t1 {
			return false
		}
	}
	return true
}

// Ray returns the nearest hit distance and the hit triangle's normal (or -1).
func (b *BVH) Ray(o, d Vec3, tmin float64) (float64, Vec3) {
	if len(b.nodes) == 0 {
		return -1, Vec3{}
	}
	inv := Vec3{1 / nz(d.X), 1 / nz(d.Y), 1 / nz(d.Z)}
	best := math.MaxFloat64
	var bn Vec3
	stack := []int{0}
	for len(stack) > 0 {
		ni := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		nd := &b.nodes[ni]
		if !rayBoxHit(o, inv, nd.lo, nd.hi, best) {
			continue
		}
		if nd.left < 0 {
			for i := nd.start; i < nd.start+nd.n; i++ {
				t := b.tris[i]
				if h := rayTriangle(o, d, t); h > tmin && h < best {
					best = h
					bn = faceNormal(t)
				}
			}
			continue
		}
		stack = append(stack, nd.left, nd.right)
	}
	if best == math.MaxFloat64 {
		return -1, Vec3{}
	}
	return best, bn
}

func nz(x float64) float64 {
	if math.Abs(x) < 1e-12 {
		return 1e-12
	}
	return x
}

func rayTriangle(o, d Vec3, t Tri) float64 {
	e1, e2 := t[1].Sub(t[0]), t[2].Sub(t[0])
	p := d.Cross(e2)
	det := e1.Dot(p)
	if math.Abs(det) < 1e-14 {
		return -1
	}
	inv := 1 / det
	s := o.Sub(t[0])
	u := s.Dot(p) * inv
	if u < 0 || u > 1 {
		return -1
	}
	q := s.Cross(e1)
	v := d.Dot(q) * inv
	if v < 0 || u+v > 1 {
		return -1
	}
	return e2.Dot(q) * inv
}

// ---------- wall thickness ----------

// ThinSpot is a place where the part is thinner than the recommended wall.
type ThinSpot struct {
	At        [3]float64 `json:"at"`
	Thickness float64    `json:"thickness"`
}

// Thickness summarises how thick the material is all over a part.
type Thickness struct {
	Min         float64    `json:"min"`      // thinnest wall found (robust: 2nd percentile)
	ThinArea    float64    `json:"thinArea"` // surface area thinner than Warn
	CritArea    float64    `json:"critArea"` // surface area thinner than Crit
	TotalArea   float64    `json:"totalArea"`
	Spots       []ThinSpot `json:"spots"` // worst places (clustered)
	Warn, Crit  float64    `json:"-"`
	Samples     int        `json:"samples"`
	PerTriangle []float32  `json:"-"` // thickness per triangle (for the heat-map)
}

// WallThickness casts rays inward from points all over the surface and measures how far
// it is to the other side. Walls around screw holes, near edges and thin plates all show up.
func WallThickness(m *Mesh, samples int, warn, crit float64, perTri bool) Thickness {
	res := Thickness{Warn: warn, Crit: crit, Min: math.Inf(1)}
	if len(m.Tris) == 0 {
		return res
	}
	bvh := NewBVH(m)
	areas := make([]float64, len(m.Tris))
	total := 0.0
	for i, t := range m.Tris {
		areas[i] = t[1].Sub(t[0]).Cross(t[2].Sub(t[0])).Len() / 2
		total += areas[i]
	}
	res.TotalArea = total
	// shape-diameter style: a cone of rays into the material, median distance.
	// Single rays near convex corners exit early; the median ignores that.
	measure := func(t Tri, p Vec3) float64 {
		n := faceNormal(t)
		if n.Len() == 0 {
			return -1
		}
		d := n.Neg()
		u := d.Cross(V(0, 0, 1))
		if u.Len() < 0.1 {
			u = d.Cross(V(1, 0, 0))
		}
		u = u.Norm()
		w := d.Cross(u)
		var hs []float64
		dirs := []Vec3{d}
		for k := 0; k < 8; k++ {
			a := float64(k) * math.Pi / 4
			c, s := math.Cos(a), math.Sin(a)
			dirs = append(dirs, d.Mul(math.Cos(0.44)).Add(u.Mul(c*math.Sin(0.44))).Add(w.Mul(s*math.Sin(0.44))).Norm())
		}
		for i, dir := range dirs {
			h, hn := bvh.Ray(p.Add(d.Mul(2e-4)), dir, 1e-4)
			// must leave the material through a back face that is roughly parallel to this one:
			// that is a *wall*. Faces meeting at an angle (chamfers, gear teeth, cone tips) are
			// tapered edges, not walls, and are not counted.
			if h < 0 || hn.Dot(d) < 0.94 {
				if i == 0 {
					return -1
				}
				continue
			}
			if i > 0 {
				h *= math.Cos(0.44) // project onto the wall normal
			}
			hs = append(hs, h)
		}
		if len(hs) < 5 {
			return -1
		}
		sort.Float64s(hs)
		return hs[len(hs)/2]
	}
	rng := rand.New(rand.NewSource(7))
	// area-weighted sampling
	cum := make([]float64, len(areas))
	acc := 0.0
	for i, a := range areas {
		acc += a
		cum[i] = acc
	}
	var vals []float64
	type pt struct {
		p Vec3
		t float64
	}
	var thin []pt
	for s := 0; s < samples; s++ {
		r := rng.Float64() * acc
		i := sort.SearchFloat64s(cum, r)
		if i >= len(m.Tris) {
			i = len(m.Tris) - 1
		}
		t := m.Tris[i]
		u, v := rng.Float64(), rng.Float64()
		if u+v > 1 {
			u, v = 1-u, 1-v
		}
		p := t[0].Add(t[1].Sub(t[0]).Mul(u)).Add(t[2].Sub(t[0]).Mul(v))
		th := measure(t, p)
		if th < 0 {
			continue
		}
		vals = append(vals, th)
		w := acc / float64(samples)
		if th < warn {
			res.ThinArea += w
			thin = append(thin, pt{p, th})
		}
		if th < crit {
			res.CritArea += w
		}
	}
	res.Samples = len(vals)
	if len(vals) > 0 {
		sort.Float64s(vals)
		res.Min = vals[int(float64(len(vals))*0.02)]
	}
	// cluster thin points into spots (worst first)
	sort.Slice(thin, func(i, j int) bool { return thin[i].t < thin[j].t })
	for _, q := range thin {
		near := false
		for _, s := range res.Spots {
			if math.Hypot(math.Hypot(s.At[0]-q.p.X, s.At[1]-q.p.Y), s.At[2]-q.p.Z) < 4 {
				near = true
				break
			}
		}
		if !near && len(res.Spots) < 12 {
			res.Spots = append(res.Spots, ThinSpot{[3]float64{q.p.X, q.p.Y, q.p.Z}, round3(q.t)})
		}
	}
	if perTri {
		res.PerTriangle = make([]float32, len(m.Tris))
		for i, t := range m.Tris {
			c := t[0].Add(t[1]).Add(t[2]).Mul(1.0 / 3)
			th := measure(t, c)
			if th < 0 {
				th = 99
			}
			res.PerTriangle[i] = float32(th)
		}
	}
	return res
}
