package geom

import (
	"fmt"
	"math"
	"sort"
)

// Slice cuts the mesh with a horizontal plane and returns closed 2D loops.
func Slice(m *Mesh, z float64) []Loop {
	z += 1.37e-5 // avoid vertices lying exactly on the plane
	type seg struct{ a, b P2 }
	var segs []seg
	for _, t := range m.Tris {
		d0, d1, d2 := t[0].Z-z, t[1].Z-z, t[2].Z-z
		if (d0 > 0 && d1 > 0 && d2 > 0) || (d0 < 0 && d1 < 0 && d2 < 0) {
			continue
		}
		var pts []P2
		edge := func(p, q Vec3, dp, dq float64) {
			if (dp > 0) != (dq > 0) {
				s := dp / (dp - dq)
				pts = append(pts, P2{p.X + (q.X-p.X)*s, p.Y + (q.Y-p.Y)*s})
			}
		}
		edge(t[0], t[1], d0, d1)
		edge(t[1], t[2], d1, d2)
		edge(t[2], t[0], d2, d0)
		if len(pts) == 2 {
			segs = append(segs, seg{pts[0], pts[1]})
		}
	}
	type key struct{ x, y int64 }
	k := func(p P2) key { return key{int64(math.Round(p.X * 2e3)), int64(math.Round(p.Y * 2e3))} }
	adj := map[key][]int{}
	for i, s := range segs {
		adj[k(s.a)] = append(adj[k(s.a)], i)
		adj[k(s.b)] = append(adj[k(s.b)], i)
	}
	used := make([]bool, len(segs))
	var loops []Loop
	for i := range segs {
		if used[i] {
			continue
		}
		used[i] = true
		l := Loop{segs[i].a, segs[i].b}
		cur := segs[i].b
		for steps := 0; steps < len(segs); steps++ {
			next := -1
			for _, j := range adj[k(cur)] {
				if !used[j] {
					next = j
					break
				}
			}
			if next < 0 {
				break
			}
			used[next] = true
			s := segs[next]
			if k(s.a) == k(cur) {
				cur = s.b
			} else {
				cur = s.a
			}
			if k(cur) == k(l[0]) {
				break
			}
			l = append(l, cur)
		}
		if len(l) >= 3 {
			loops = append(loops, l)
		}
	}
	return loops
}

func loopBounds(l Loop) (float64, float64, float64, float64) {
	x0, y0, x1, y1 := math.MaxFloat64, math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64
	for _, p := range l {
		x0, y0, x1, y1 = math.Min(x0, p.X), math.Min(y0, p.Y), math.Max(x1, p.X), math.Max(y1, p.Y)
	}
	return x0, y0, x1, y1
}

func perimeter(l Loop) float64 {
	s := 0.0
	for i := range l {
		j := (i + 1) % len(l)
		s += math.Hypot(l[j].X-l[i].X, l[j].Y-l[i].Y)
	}
	return s
}

func circularity(l Loop) float64 {
	p := perimeter(l)
	if p == 0 {
		return 0
	}
	return 4 * math.Pi * math.Abs(l.Area()) / (p * p)
}

func centroid(l Loop) P2 {
	x0, y0, x1, y1 := loopBounds(l)
	return P2{(x0 + x1) / 2, (y0 + y1) / 2}
}

// ScanResult is returned to the editor.
type ScanResult struct {
	Spec   *BoardSpec `json:"spec"`
	Origin [3]float64 `json:"origin"` // world position of the board centre (bottom surface)
	Report []string   `json:"report"`
	OK     bool       `json:"ok"`
}

// ScanModel studies an imported model (lying flat, parts facing up) and describes it
// as a BoardSpec: outline, mounting holes, screen window, connectors and heights.
func ScanModel(m *Mesh, name string) ScanResult {
	lo, hi := m.Bounds()
	H := hi.Z - lo.Z
	res := ScanResult{}
	if len(m.Tris) == 0 || H <= 0 {
		res.Report = []string{"The model is empty."}
		return res
	}
	n := 70
	step := H / float64(n)
	type level struct {
		z      float64
		shapes []Shape
		area   float64 // largest outer area
		best   int
	}
	levels := make([]level, 0, n)
	for i := 0; i < n; i++ {
		z := lo.Z + step*(float64(i)+0.5)
		sh := ShapesFromLoops(Slice(m, z))
		lv := level{z: z, shapes: sh, best: -1}
		for k, s := range sh {
			if a := s.Outer.Area(); a > lv.area {
				lv.area, lv.best = a, k
			}
		}
		levels = append(levels, lv)
	}
	// the board is the widest layer in the lower part of the model
	bi := 0
	for i, lv := range levels {
		if float64(i) > float64(n)*0.75 {
			break
		}
		if lv.area > levels[bi].area*1.02 {
			bi = i
		}
	}
	if levels[bi].best < 0 {
		res.Report = []string{"Could not find a flat board in this model. Lay it flat (board horizontal, parts facing up) and scan again."}
		return res
	}
	maxA := levels[bi].area
	i0, i1 := bi, bi
	for i0 > 0 && levels[i0-1].area > maxA*0.9 {
		i0--
	}
	for i1 < n-1 && levels[i1+1].area > maxA*0.9 {
		i1++
	}
	zb0 := math.Max(lo.Z, levels[i0].z-step/2)
	zb1 := math.Min(hi.Z, levels[i1].z+step/2)
	board := levels[bi].shapes[levels[bi].best]
	x0, y0, x1, y1 := loopBounds(board.Outer)
	cx, cy := (x0+x1)/2, (y0+y1)/2
	W, D := x1-x0, y1-y0
	T := math.Max(0.4, zb1-zb0)
	if T > H*0.7 {
		// a solid object rather than a board: treat its full height as the "board"
		zb0, zb1, T = lo.Z, hi.Z, H
	}
	b := &BoardSpec{Name: name, Family: "Scanned", W: round3(W), D: round3(D), T: round3(T), Above: round3(hi.Z - zb1), Below: round3(zb0 - lo.Z)}
	ratio := math.Abs(board.Outer.Area()) / (W * D)
	if math.Abs(W-D) < 0.05*W && ratio > 0.74 && ratio < 0.83 && circularity(board.Outer) > 0.85 {
		b.Round = true
	}
	for _, p := range board.Outer {
		b.Outline = append(b.Outline, P2{round3(p.X - cx), round3(p.Y - cy)})
	}
	if len(b.Outline) > 600 { // keep the saved spec small
		var thin []P2
		k := len(b.Outline) / 600
		for i := 0; i < len(b.Outline); i += k + 1 {
			thin = append(thin, b.Outline[i])
		}
		b.Outline = thin
	}
	// mounting holes: round holes through the board layer
	for _, h := range board.Holes {
		a := math.Abs(h.Area())
		d := 2 * math.Sqrt(a/math.Pi)
		if circularity(h) > 0.75 && d >= 1 && d <= 8.5 {
			c := centroid(h)
			b.Holes = append(b.Holes, BHole{round3(c.X - cx), round3(c.Y - cy), round3(d)})
		}
	}
	boardArea := maxA
	// top feature (screen glass / sensor domes): the highest level with a big flat area
	if hi.Z-zb1 > 0.3 {
		for i := n - 1; i > i1; i-- {
			lv := levels[i]
			var rounds []BWindow
			var big *BWindow
			for _, s := range lv.shapes {
				a := s.Outer.Area()
				lx0, ly0, lx1, ly1 := loopBounds(s.Outer)
				c := P2{(lx0+lx1)/2 - cx, (ly0+ly1)/2 - cy}
				if a > boardArea*0.12 && a/((lx1-lx0)*(ly1-ly0)) > 0.8 {
					w := BWindow{round3(c.X), round3(c.Y), round3(lx1 - lx0), round3(ly1 - ly0), round3(lv.z - zb1 + step/2), false}
					if circularity(s.Outer) > 0.85 {
						w.Round = true
					}
					if big == nil || w.W*w.D > big.W*big.D {
						big = &w
					}
				} else if a > boardArea*0.03 && circularity(s.Outer) > 0.85 {
					rounds = append(rounds, BWindow{round3(c.X), round3(c.Y), round3(lx1 - lx0), round3(lx1 - lx0), round3(lv.z - zb1 + step/2), true})
				}
			}
			if big != nil {
				b.Window = big
				break
			}
			if len(rounds) > 0 {
				b.Circles = rounds
				break
			}
		}
	}
	// connectors: anything above the board that reaches (or overhangs) an edge
	type hit struct {
		edge   string
		s0, s1 float64
		z0, z1 float64
	}
	var hits []hit
	ex0, ey0, ex1, ey1 := x0-cx, y0-cy, x1-cx, y1-cy
	near := 1.2
	for i := i1 + 1; i < n; i++ {
		lv := levels[i]
		zr := lv.z - zb1
		for _, s := range lv.shapes {
			a := s.Outer.Area()
			if a > boardArea*0.25 {
				continue
			}
			lx0, ly0, lx1, ly1 := loopBounds(s.Outer)
			lx0, lx1, ly0, ly1 = lx0-cx, lx1-cx, ly0-cy, ly1-cy
			add := func(e string, s0, s1 float64) {
				hits = append(hits, hit{e, s0, s1, zr - step/2, zr + step/2})
			}
			if lx0 <= ex0+near {
				add("-x", ly0, ly1)
			}
			if lx1 >= ex1-near {
				add("+x", ly0, ly1)
			}
			if ly0 <= ey0+near {
				add("-y", lx0, lx1)
			}
			if ly1 >= ey1-near {
				add("+y", lx0, lx1)
			}
		}
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].edge != hits[j].edge {
			return hits[i].edge < hits[j].edge
		}
		return hits[i].s0 < hits[j].s0
	})
	var merged []hit
	for _, h := range hits {
		ok := false
		for k := range merged {
			g := &merged[k]
			if g.edge == h.edge && h.s0 <= g.s1+0.6 && h.s1 >= g.s0-0.6 {
				g.s0, g.s1 = math.Min(g.s0, h.s0), math.Max(g.s1, h.s1)
				g.z0, g.z1 = math.Min(g.z0, h.z0), math.Max(g.z1, h.z1)
				ok = true
				break
			}
		}
		if !ok {
			merged = append(merged, h)
		}
	}
	for _, g := range merged {
		edgeLen := W
		if g.edge == "-x" || g.edge == "+x" {
			edgeLen = D
		}
		if g.s1-g.s0 < 2.5 || g.s1-g.s0 > edgeLen*0.9 || g.z1-g.z0 < 0.8 {
			continue
		}
		b.Ports = append(b.Ports, BPort{g.edge, round3((g.s0 + g.s1) / 2), round3(g.s1 - g.s0), round3(math.Max(0, g.z0)), round3(g.z1), "Connector"})
	}
	res.Spec = b
	res.Origin = [3]float64{cx, cy, zb0}
	res.OK = true
	shape := "rectangular"
	if b.Round {
		shape = "round"
	}
	res.Report = append(res.Report, fmt.Sprintf("Board: %s, %.1f × %.1f mm, %.1f mm thick", shape, b.W, b.D, b.T))
	if len(b.Holes) > 0 {
		res.Report = append(res.Report, fmt.Sprintf("%d mounting hole(s), Ø %.1f mm", len(b.Holes), b.Holes[0].D))
	} else {
		res.Report = append(res.Report, "No mounting holes found — a snap-in tray will hold it")
	}
	res.Report = append(res.Report, fmt.Sprintf("Tallest part %.1f mm above the board, %.1f mm underneath", b.Above, b.Below))
	if b.Window != nil {
		res.Report = append(res.Report, fmt.Sprintf("Screen / window on top: %.1f × %.1f mm", b.Window.W, b.Window.D))
	}
	if len(b.Circles) > 0 {
		res.Report = append(res.Report, fmt.Sprintf("%d round sensor / knob opening(s) on top", len(b.Circles)))
	}
	if len(b.Ports) > 0 {
		names := map[string]string{"-x": "left", "+x": "right", "-y": "front", "+y": "back"}
		s := fmt.Sprintf("%d connector opening(s):", len(b.Ports))
		for _, p := range b.Ports {
			s += fmt.Sprintf(" %s edge %.1f mm wide;", names[p.Edge], p.W)
		}
		res.Report = append(res.Report, s)
	}
	return res
}

func round3(x float64) float64 { return math.Round(x*1000) / 1000 }
