package geom

import (
	"encoding/binary"
	"math"
)

// Tri is one triangle, counter-clockwise when seen from outside.
type Tri [3]Vec3

// Mesh is a triangle soup. Closed meshes are expected for solids.
type Mesh struct {
	Tris  []Tri
	Info  string // human-readable notes (e.g. thread specification)
	Ports []Port // connection points (optional)
}

// Port is an optional connection point of a part: a position, a direction
// and whether it is an input or an output.
type Port struct {
	P  [3]float64 `json:"p"`
	D  [3]float64 `json:"d"`
	In bool       `json:"in"`
}

func (m *Mesh) Add(a, b, c Vec3) {
	// drop degenerate triangles (zero area)
	if b.Sub(a).Cross(c.Sub(a)).Len() < 1e-10 {
		return
	}
	m.Tris = append(m.Tris, Tri{a, b, c})
}

func (m *Mesh) AddQuad(a, b, c, d Vec3) { m.Add(a, b, c); m.Add(a, c, d) }

func (m *Mesh) Append(o *Mesh) {
	if o != nil {
		m.Tris = append(m.Tris, o.Tris...)
	}
}

func (m *Mesh) Clone() *Mesh {
	n := &Mesh{Tris: make([]Tri, len(m.Tris))}
	copy(n.Tris, m.Tris)
	return n
}

// Transformed returns a copy with the matrix applied; winding is fixed for mirrors.
func (m *Mesh) Transformed(t Mat4) *Mesh {
	n := &Mesh{Tris: make([]Tri, len(m.Tris))}
	flip := t.Det3() < 0
	for i, tr := range m.Tris {
		a, b, c := t.Apply(tr[0]), t.Apply(tr[1]), t.Apply(tr[2])
		if flip {
			b, c = c, b
		}
		n.Tris[i] = Tri{a, b, c}
	}
	return n
}

func (m *Mesh) Flip() {
	for i := range m.Tris {
		m.Tris[i][1], m.Tris[i][2] = m.Tris[i][2], m.Tris[i][1]
	}
}

// Volume returns the signed volume (positive for outward-facing closed meshes).
func (m *Mesh) Volume() float64 {
	var v float64
	for _, t := range m.Tris {
		v += t[0].Dot(t[1].Cross(t[2]))
	}
	return v / 6
}

func (m *Mesh) Area() float64 {
	var a float64
	for _, t := range m.Tris {
		a += t[1].Sub(t[0]).Cross(t[2].Sub(t[0])).Len() / 2
	}
	return a
}

// Orient makes the mesh face outward (positive volume).
func (m *Mesh) Orient() *Mesh {
	if m.Volume() < 0 {
		m.Flip()
	}
	return m
}

func (m *Mesh) Bounds() (Vec3, Vec3) {
	if len(m.Tris) == 0 {
		return Vec3{}, Vec3{}
	}
	lo, hi := m.Tris[0][0], m.Tris[0][0]
	for _, t := range m.Tris {
		for _, p := range t {
			lo, hi = lo.Min(p), hi.Max(p)
		}
	}
	return lo, hi
}

// Encode produces the binary format the editor consumes:
// uint32 triCount, uint32 edgeVertCount, then float32 positions (tri*9),
// float32 normals (tri*9, creased smooth normals), float32 edge line vertices.
func (m *Mesh) Encode(creaseDeg float64) []byte {
	n := len(m.Tris)
	normals := m.creasedNormals(creaseDeg)
	edges := m.featureEdges(28)
	buf := make([]byte, 8+n*9*4*2+len(edges)*3*4)
	binary.LittleEndian.PutUint32(buf[0:], uint32(n))
	binary.LittleEndian.PutUint32(buf[4:], uint32(len(edges)))
	off := 8
	put := func(f float64) {
		binary.LittleEndian.PutUint32(buf[off:], math.Float32bits(float32(f)))
		off += 4
	}
	for _, t := range m.Tris {
		for _, p := range t {
			put(p.X)
			put(p.Y)
			put(p.Z)
		}
	}
	for _, nn := range normals {
		put(nn.X)
		put(nn.Y)
		put(nn.Z)
	}
	for _, e := range edges {
		put(e.X)
		put(e.Y)
		put(e.Z)
	}
	return buf
}

type vkey struct{ x, y, z int64 }

func keyOf(p Vec3) vkey {
	const q = 1e4 // 0.1 micron
	return vkey{int64(math.Round(p.X * q)), int64(math.Round(p.Y * q)), int64(math.Round(p.Z * q))}
}

func faceNormal(t Tri) Vec3 { return t[1].Sub(t[0]).Cross(t[2].Sub(t[0])).Norm() }

func (m *Mesh) creasedNormals(creaseDeg float64) []Vec3 {
	cosT := math.Cos(creaseDeg * math.Pi / 180)
	fn := make([]Vec3, len(m.Tris))
	area := make([]float64, len(m.Tris))
	adj := map[vkey][]int{}
	for i, t := range m.Tris {
		c := t[1].Sub(t[0]).Cross(t[2].Sub(t[0]))
		area[i] = c.Len()
		fn[i] = c.Norm()
		for _, p := range t {
			k := keyOf(p)
			adj[k] = append(adj[k], i)
		}
	}
	out := make([]Vec3, len(m.Tris)*3)
	for i, t := range m.Tris {
		for j, p := range t {
			var s Vec3
			for _, f := range adj[keyOf(p)] {
				if fn[f].Dot(fn[i]) >= cosT {
					s = s.Add(fn[f].Mul(area[f]))
				}
			}
			s = s.Norm()
			if s.Len() == 0 {
				s = fn[i]
			}
			out[i*3+j] = s
		}
	}
	return out
}

type ekey struct{ a, b vkey }

func lessKey(a, b vkey) bool {
	if a.x != b.x {
		return a.x < b.x
	}
	if a.y != b.y {
		return a.y < b.y
	}
	return a.z < b.z
}

// featureEdges returns line segments (pairs of points) where faces meet at a sharp angle
// or where the mesh has open boundaries.
func (m *Mesh) featureEdges(angleDeg float64) []Vec3 {
	cosT := math.Cos(angleDeg * math.Pi / 180)
	type rec struct {
		p, q    Vec3
		normals []Vec3
	}
	edges := map[ekey]*rec{}
	order := []ekey{}
	for _, t := range m.Tris {
		n := faceNormal(t)
		for i := 0; i < 3; i++ {
			p, q := t[i], t[(i+1)%3]
			kp, kq := keyOf(p), keyOf(q)
			if kp == kq {
				continue
			}
			k := ekey{kp, kq}
			if lessKey(kq, kp) {
				k = ekey{kq, kp}
			}
			r, ok := edges[k]
			if !ok {
				r = &rec{p: p, q: q}
				edges[k] = r
				order = append(order, k)
			}
			r.normals = append(r.normals, n)
		}
	}
	var out []Vec3
	for _, k := range order {
		r := edges[k]
		sharp := len(r.normals) != 2
		if !sharp && r.normals[0].Dot(r.normals[1]) < cosT {
			sharp = true
		}
		if sharp {
			out = append(out, r.p, r.q)
		}
		if len(out) > 400000 {
			break
		}
	}
	return out
}

// DecodeRaw reads little-endian float32 triangle positions (9 floats per triangle).
func DecodeRaw(b []byte) *Mesh {
	n := len(b) / 36
	m := &Mesh{Tris: make([]Tri, 0, n)}
	f := func(i int) float64 { return float64(math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))) }
	for t := 0; t < n; t++ {
		var tr Tri
		for v := 0; v < 3; v++ {
			i := t*9 + v*3
			tr[v] = Vec3{f(i), f(i + 1), f(i + 2)}
		}
		m.Tris = append(m.Tris, tr)
	}
	return m
}

// EncodeRaw writes float32 triangle positions.
func (m *Mesh) EncodeRaw() []byte {
	buf := make([]byte, len(m.Tris)*36)
	off := 0
	for _, t := range m.Tris {
		for _, p := range t {
			for _, f := range []float64{p.X, p.Y, p.Z} {
				binary.LittleEndian.PutUint32(buf[off:], math.Float32bits(float32(f)))
				off += 4
			}
		}
	}
	return buf
}
