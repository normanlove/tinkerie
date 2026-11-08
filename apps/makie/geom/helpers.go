package geom

import "math"

// ---- small construction helpers shared by the part library ----

// ring makes a tube (annulus) between z0 and z1.
func ring(od, id, z0, z1 float64, segs int) *Mesh {
	if id <= 0.01 {
		return cylAt(0, 0, od, z0, z1, segs)
	}
	return Revolve(Shape{Outer: Loop{{id / 2, z0}, {od / 2, z0}, {od / 2, z1}, {id / 2, z1}}}, 360, segs)
}

// cone between z0 (diameter d0) and z1 (diameter d1).
func coneAt(d0, d1, z0, z1 float64, segs int) *Mesh {
	return RevolveProfile([]P2{{0, z0}, {d0 / 2, z0}, {d1 / 2, z1}, {0, z1}}, segs)
}

// prismY extrudes a profile drawn in the X–Z plane symmetrically along Y.
func prismY(l Loop, width float64) *Mesh {
	return Sweep(Shape{Outer: l}, []Frame{{V(0, -width/2, 0), V(1, 0, 0), V(0, 0, 1)}, {V(0, width/2, 0), V(1, 0, 0), V(0, 0, 1)}}, false)
}

// shapeY extrudes a shape with holes drawn in X–Z along Y.
func shapeY(s Shape, width float64) *Mesh {
	return Sweep(s, []Frame{{V(0, -width/2, 0), V(1, 0, 0), V(0, 0, 1)}, {V(0, width/2, 0), V(1, 0, 0), V(0, 0, 1)}}, false)
}

// extrudeAt extrudes a 2D shape between z0 and z1.
func extrudeAt(s Shape, z0, z1 float64) *Mesh {
	return Extrude(s, z1-z0, 0, 1, 1).Transformed(Translate(0, 0, z0))
}

// along rotates a Z-axis mesh so it lies along X (axis=0) or Y (axis=1).
func along(m *Mesh, axis int) *Mesh {
	switch axis {
	case 0:
		return m.Transformed(RotY(90))
	case 1:
		return m.Transformed(RotX(-90))
	}
	return m
}

// tubePath sweeps a circle of diameter d along a polyline (rounded joints come from dense points).
func tubePath(pts []Vec3, d float64, segs int) *Mesh {
	if len(pts) < 2 {
		return &Mesh{}
	}
	prof := Shape{Outer: Circle(d/2, segs)}
	closed := len(pts) > 3 && pts[0].Sub(pts[len(pts)-1]).Len() < 1e-6
	if closed {
		pts = pts[:len(pts)-1]
		// drop interior duplicates (arcs that share an end point)
		var q []Vec3
		for i, p := range pts {
			if i > 0 && p.Sub(q[len(q)-1]).Len() < 1e-6 {
				continue
			}
			q = append(q, p)
		}
		pts = q
	}
	var frames []Frame
	var prevU Vec3
	for i := range pts {
		var t Vec3
		switch {
		case closed:
			t = pts[(i+1)%len(pts)].Sub(pts[(i+len(pts)-1)%len(pts)])
		case i == 0:
			t = pts[1].Sub(pts[0])
		case i == len(pts)-1:
			t = pts[i].Sub(pts[i-1])
		default:
			t = pts[i+1].Sub(pts[i-1])
		}
		t = t.Norm()
		var u Vec3
		if i == 0 {
			ref := V(0, 0, 1)
			if math.Abs(t.Z) > 0.9 {
				ref = V(1, 0, 0)
			}
			u = ref.Sub(t.Mul(ref.Dot(t))).Norm()
		} else {
			// parallel transport of the previous frame
			u = prevU.Sub(t.Mul(prevU.Dot(t))).Norm()
		}
		prevU = u
		v := t.Cross(u)
		frames = append(frames, Frame{pts[i], u, v})
	}
	return Sweep(prof, frames, closed)
}

// arcPts returns points on an arc in the XZ plane (or XY when flat=true).
func arcPts(cx, cy, r, a0, a1 float64, n int, flat bool, z float64) []Vec3 {
	var out []Vec3
	for i := 0; i <= n; i++ {
		a := (a0 + (a1-a0)*float64(i)/float64(n)) * math.Pi / 180
		if flat {
			out = append(out, V(cx+r*math.Cos(a), cy+r*math.Sin(a), z))
		} else {
			out = append(out, V(cx+r*math.Cos(a), 0, cy+r*math.Sin(a)))
		}
	}
	return out
}

// knurlLoop is a circle with n rounded lobes (for grips).
func knurlLoop(d, depth float64, n, segs int) Loop {
	var l Loop
	if segs < n*6 {
		segs = n * 6
	}
	for i := 0; i < segs; i++ {
		a := 2 * math.Pi * float64(i) / float64(segs)
		r := d/2 - depth*(0.5-0.5*math.Cos(float64(n)*a))
		l = append(l, P2{r * math.Cos(a), r * math.Sin(a)})
	}
	return l
}

// moveLoop translates a 2D loop.
func moveLoop(l Loop, dx, dy float64) Loop {
	o := make(Loop, len(l))
	for i, p := range l {
		o[i] = P2{p.X + dx, p.Y + dy}
	}
	return o
}

// stadium is a slot outline of length l (centre to centre + diameter) and width w.
func stadium(l, w float64, segs int) Loop {
	r := w / 2
	l = math.Max(l, w+0.01)
	var lp Loop
	n := segs / 2
	if n < 4 {
		n = 4
	}
	for i := 0; i <= n; i++ {
		a := -math.Pi/2 + math.Pi*float64(i)/float64(n)
		lp = append(lp, P2{l/2 - r + r*math.Cos(a), r * math.Sin(a)})
	}
	for i := 0; i <= n; i++ {
		a := math.Pi/2 + math.Pi*float64(i)/float64(n)
		lp = append(lp, P2{-l/2 + r + r*math.Cos(a), r * math.Sin(a)})
	}
	return lp
}

// holesThrough subtracts vertical cylinders at the given XY points.
func holesThrough(m *Mesh, pts [][2]float64, d, z0, z1 float64, segs int) *Mesh {
	var hs []*Mesh
	for _, p := range pts {
		hs = append(hs, cylAt(p[0], p[1], d, z0, z1, segs))
	}
	if len(hs) == 0 {
		return m
	}
	return Subtract(m, UnionAll(hs))
}

// sizesWith returns the metric size list (used by most fasteners).
func metricList() []string { return metricSizes() }

// shapeX extrudes a shape drawn in the Y–Z plane along X, from x0 to x1.
func shapeX(s Shape, x0, x1 float64) *Mesh {
	return Sweep(s, []Frame{{V(x0, 0, 0), V(0, 1, 0), V(0, 0, 1)}, {V(x1, 0, 0), V(0, 1, 0), V(0, 0, 1)}}, false)
}

// rectL is an axis-aligned rectangle loop.
func rectL(x0, y0, x1, y1 float64) Loop { return Loop{{x0, y0}, {x1, y0}, {x1, y1}, {x0, y1}} }

// rotLoop rotates a loop about the origin.
func rotLoop(l Loop, deg float64) Loop {
	a := deg * math.Pi / 180
	c, sn := math.Cos(a), math.Sin(a)
	out := make(Loop, len(l))
	for i, p := range l {
		out[i] = P2{p.X*c - p.Y*sn, p.X*sn + p.Y*c}
	}
	return out
}
