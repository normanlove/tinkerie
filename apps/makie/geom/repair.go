package geom

import (
	"math"
	"sort"
)

// FixTJunctions splits triangles whose edges have other vertices lying on them,
// which BSP CSG leaves behind. The result is watertight for slicers.
func (m *Mesh) FixTJunctions() *Mesh {
	if len(m.Tris) == 0 {
		return m
	}
	a := m.fixOnly()
	oa := openEdgeCount(a)
	if oa == 0 {
		return a
	}
	best, bo := a, oa
	// BSP splits of curved parts can leave points a fraction of a micron
	// apart; try welding a little more coarsely until the mesh closes
	for _, eps := range []float64{weldEps, 2e-4, 8e-4} {
		b := m.Weld(eps).fixOnly()
		if ob := openEdgeCount(b); ob < bo {
			best, bo = b, ob
			if ob == 0 {
				break
			}
		}
	}
	if bo > 0 {
		// last resort: the gaps left are tiny triangles BSP lost; fill any
		// hole whose rim is shorter than half a millimetre
		for _, eps := range []float64{2e-4, 8e-4} {
			b := m.Weld(eps).fixOnly().closeTinyHoles(0.5)
			if ob := openEdgeCount(b); ob < bo {
				best, bo = b, ob
				if ob == 0 {
					break
				}
			}
		}
	}
	return best
}

// closeTinyHoles fills small open loops with triangle fans.
func (m *Mesh) closeTinyHoles(maxPerim float64) *Mesh {
	type dk struct{ a, b vkey }
	cnt := map[dk]int{}
	pos := map[vkey]Vec3{}
	for _, t := range m.Tris {
		for i := 0; i < 3; i++ {
			a, b := keyOf(t[i]), keyOf(t[(i+1)%3])
			pos[a] = t[i]
			cnt[dk{a, b}]++
			cnt[dk{b, a}]--
		}
	}
	// unmatched directed edges a→b (more a→b than b→a)
	next := map[vkey][]vkey{}
	for k, c := range cnt {
		for ; c > 0; c-- {
			next[k.a] = append(next[k.a], k.b)
		}
	}
	if len(next) == 0 {
		return m
	}
	out := &Mesh{Tris: append([]Tri(nil), m.Tris...), Info: m.Info, Ports: m.Ports}
	for start := range next {
		for len(next[start]) > 0 {
			loop := []vkey{start}
			cur := start
			ok := false
			for steps := 0; steps < 64; steps++ {
				nx := next[cur]
				if len(nx) == 0 {
					break
				}
				n := nx[0]
				next[cur] = nx[1:]
				if n == start {
					ok = true
					break
				}
				loop = append(loop, n)
				cur = n
			}
			if !ok || len(loop) < 3 {
				continue
			}
			per := 0.0
			for i := range loop {
				per += pos[loop[i]].Sub(pos[loop[(i+1)%len(loop)]]).Len()
			}
			if per > maxPerim {
				continue
			}
			// the existing faces run a→b along the rim, so the patch runs b→a
			for i := 1; i+1 < len(loop); i++ {
				out.Tris = append(out.Tris, Tri{pos[loop[0]], pos[loop[i+1]], pos[loop[i]]})
			}
		}
	}
	return out
}

func openEdgeCount(m *Mesh) int {
	cnt := map[ekey]int{}
	for _, t := range m.Tris {
		for i := 0; i < 3; i++ {
			a, b := keyOf(t[i]), keyOf(t[(i+1)%3])
			if a == b {
				continue
			}
			if lessKey(b, a) {
				a, b = b, a
			}
			cnt[ekey{a, b}]++
		}
	}
	n := 0
	for _, c := range cnt {
		if c%2 == 1 {
			n++
		}
	}
	return n
}

var weldEps = 2e-5

func (m *Mesh) fixOnly() *Mesh {
	// directed edge counts
	type dk struct{ a, b vkey }
	cnt := map[dk]int{}
	for _, t := range m.Tris {
		for i := 0; i < 3; i++ {
			cnt[dk{keyOf(t[i]), keyOf(t[(i+1)%3])}]++
		}
	}
	// unique vertices
	uniq := map[vkey]Vec3{}
	for _, t := range m.Tris {
		for _, p := range t {
			uniq[keyOf(p)] = p
		}
	}
	lo, hi := m.Bounds()
	diag := hi.Sub(lo).Len()
	cell := math.Max(diag/64, 0.05)
	type ck struct{ x, y, z int }
	cellOf := func(p Vec3) ck {
		return ck{int(math.Floor(p.X / cell)), int(math.Floor(p.Y / cell)), int(math.Floor(p.Z / cell))}
	}
	grid := map[ck][]Vec3{}
	for _, p := range uniq {
		c := cellOf(p)
		grid[c] = append(grid[c], p)
	}
	onEdge := func(a, b Vec3) []Vec3 {
		ab := b.Sub(a)
		L2 := ab.Dot(ab)
		if L2 < 1e-14 {
			return nil
		}
		c0, c1 := cellOf(a.Min(b)), cellOf(a.Max(b))
		if (c1.x-c0.x+1)*(c1.y-c0.y+1)*(c1.z-c0.z+1) > 20000 {
			return nil
		}
		type hit struct {
			t float64
			p Vec3
		}
		var hs []hit
		for x := c0.x; x <= c1.x; x++ {
			for y := c0.y; y <= c1.y; y++ {
				for z := c0.z; z <= c1.z; z++ {
					for _, p := range grid[ck{x, y, z}] {
						t := p.Sub(a).Dot(ab) / L2
						if t <= 1e-6 || t >= 1-1e-6 {
							continue
						}
						d := a.Add(ab.Mul(t)).Sub(p).Len()
						if d < 2e-5 {
							hs = append(hs, hit{t, p})
						}
					}
				}
			}
		}
		sort.Slice(hs, func(i, j int) bool { return hs[i].t < hs[j].t })
		out := make([]Vec3, len(hs))
		for i, h := range hs {
			out[i] = h.p
		}
		return out
	}
	res := &Mesh{Tris: make([]Tri, 0, len(m.Tris))}
	for _, t := range m.Tris {
		var poly []Vec3
		split := false
		for i := 0; i < 3; i++ {
			a, b := t[i], t[(i+1)%3]
			poly = append(poly, a)
			ka, kb := keyOf(a), keyOf(b)
			if cnt[dk{kb, ka}] > 0 { // matched edge, nothing to do
				continue
			}
			mid := onEdge(a, b)
			if len(mid) > 0 {
				split = true
				poly = append(poly, mid...)
			}
		}
		if !split {
			res.Tris = append(res.Tris, t)
			continue
		}
		c := t[0].Add(t[1]).Add(t[2]).Mul(1.0 / 3)
		for i := range poly {
			res.Add(c, poly[i], poly[(i+1)%len(poly)])
		}
	}
	return res
}

// Weld snaps vertices closer than eps to one shared position and drops
// triangles that collapse, so nearly-identical BSP split points line up.
func (m *Mesh) Weld(eps float64) *Mesh {
	type ck struct{ x, y, z int64 }
	cell := eps * 4
	cof := func(p Vec3) ck {
		return ck{int64(math.Floor(p.X / cell)), int64(math.Floor(p.Y / cell)), int64(math.Floor(p.Z / cell))}
	}
	grid := map[ck][]Vec3{}
	canon := func(p Vec3) Vec3 {
		c := cof(p)
		for dx := int64(-1); dx <= 1; dx++ {
			for dy := int64(-1); dy <= 1; dy++ {
				for dz := int64(-1); dz <= 1; dz++ {
					for _, q := range grid[ck{c.x + dx, c.y + dy, c.z + dz}] {
						if q.Sub(p).Len() <= eps {
							return q
						}
					}
				}
			}
		}
		grid[c] = append(grid[c], p)
		return p
	}
	out := &Mesh{Tris: make([]Tri, 0, len(m.Tris))}
	for _, t := range m.Tris {
		a, b, c := canon(t[0]), canon(t[1]), canon(t[2])
		if a == b || b == c || a == c {
			continue
		}
		out.Tris = append(out.Tris, Tri{a, b, c})
	}
	return out
}
