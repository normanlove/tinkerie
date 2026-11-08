// Package geom contains the Makie geometry kernel: meshes, matrices,
// constructive solid geometry and every shape generator.
package geom

import "math"

// Vec3 is a 3D vector in millimetres.
type Vec3 struct{ X, Y, Z float64 }

func V(x, y, z float64) Vec3               { return Vec3{x, y, z} }
func (a Vec3) Add(b Vec3) Vec3             { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }
func (a Vec3) Sub(b Vec3) Vec3             { return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func (a Vec3) Mul(s float64) Vec3          { return Vec3{a.X * s, a.Y * s, a.Z * s} }
func (a Vec3) Dot(b Vec3) float64          { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }
func (a Vec3) Neg() Vec3                   { return Vec3{-a.X, -a.Y, -a.Z} }
func (a Vec3) Len() float64                { return math.Sqrt(a.Dot(a)) }
func (a Vec3) Lerp(b Vec3, t float64) Vec3 { return a.Add(b.Sub(a).Mul(t)) }
func (a Vec3) Cross(b Vec3) Vec3 {
	return Vec3{a.Y*b.Z - a.Z*b.Y, a.Z*b.X - a.X*b.Z, a.X*b.Y - a.Y*b.X}
}
func (a Vec3) Norm() Vec3 {
	l := a.Len()
	if l < 1e-15 {
		return Vec3{}
	}
	return a.Mul(1 / l)
}
func (a Vec3) Min(b Vec3) Vec3 {
	return Vec3{math.Min(a.X, b.X), math.Min(a.Y, b.Y), math.Min(a.Z, b.Z)}
}
func (a Vec3) Max(b Vec3) Vec3 {
	return Vec3{math.Max(a.X, b.X), math.Max(a.Y, b.Y), math.Max(a.Z, b.Z)}
}

// Mat4 is a row-major affine 4x4 matrix. Points are column vectors.
type Mat4 [16]float64

func Identity() Mat4 { return Mat4{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1} }

func (m Mat4) Mul(n Mat4) Mat4 {
	var r Mat4
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			var s float64
			for k := 0; k < 4; k++ {
				s += m[i*4+k] * n[k*4+j]
			}
			r[i*4+j] = s
		}
	}
	return r
}

func (m Mat4) Apply(p Vec3) Vec3 {
	return Vec3{
		m[0]*p.X + m[1]*p.Y + m[2]*p.Z + m[3],
		m[4]*p.X + m[5]*p.Y + m[6]*p.Z + m[7],
		m[8]*p.X + m[9]*p.Y + m[10]*p.Z + m[11],
	}
}

func (m Mat4) Det3() float64 {
	return m[0]*(m[5]*m[10]-m[6]*m[9]) - m[1]*(m[4]*m[10]-m[6]*m[8]) + m[2]*(m[4]*m[9]-m[5]*m[8])
}

func Translate(x, y, z float64) Mat4 {
	m := Identity()
	m[3], m[7], m[11] = x, y, z
	return m
}

func Scale(x, y, z float64) Mat4 {
	m := Identity()
	m[0], m[5], m[10] = x, y, z
	return m
}

func RotX(deg float64) Mat4 {
	s, c := math.Sincos(deg * math.Pi / 180)
	return Mat4{1, 0, 0, 0, 0, c, -s, 0, 0, s, c, 0, 0, 0, 0, 1}
}
func RotY(deg float64) Mat4 {
	s, c := math.Sincos(deg * math.Pi / 180)
	return Mat4{c, 0, s, 0, 0, 1, 0, 0, -s, 0, c, 0, 0, 0, 0, 1}
}
func RotZ(deg float64) Mat4 {
	s, c := math.Sincos(deg * math.Pi / 180)
	return Mat4{c, -s, 0, 0, s, c, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
}

// TRS builds T * Rz * Ry * Rx * S. The editor uses exactly the same order.
func TRS(pos, rotDeg, scale [3]float64) Mat4 {
	sx, sy, sz := scale[0], scale[1], scale[2]
	if sx == 0 {
		sx = 1
	}
	if sy == 0 {
		sy = 1
	}
	if sz == 0 {
		sz = 1
	}
	return Translate(pos[0], pos[1], pos[2]).Mul(RotZ(rotDeg[2])).Mul(RotY(rotDeg[1])).Mul(RotX(rotDeg[0])).Mul(Scale(sx, sy, sz))
}
