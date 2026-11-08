package geom

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// NamedMesh is an exported object.
type NamedMesh struct {
	Name     string
	Color    string
	Extruder int // 1-based filament / AMS slot
	Mesh     *Mesh
}

// WriteSTL writes one binary STL containing all meshes.
func WriteSTL(objs []NamedMesh) []byte {
	n := 0
	for _, o := range objs {
		n += len(o.Mesh.Tris)
	}
	buf := make([]byte, 84+n*50)
	copy(buf, []byte("Makie by nrmluvdev - binary STL (mm)"))
	binary.LittleEndian.PutUint32(buf[80:], uint32(n))
	off := 84
	put := func(f float64) {
		binary.LittleEndian.PutUint32(buf[off:], math.Float32bits(float32(f)))
		off += 4
	}
	for _, o := range objs {
		for _, t := range o.Mesh.Tris {
			nn := faceNormal(t)
			put(nn.X)
			put(nn.Y)
			put(nn.Z)
			for _, p := range t {
				put(p.X)
				put(p.Y)
				put(p.Z)
			}
			off += 2
		}
	}
	return buf
}

type indexed struct {
	verts []Vec3
	faces [][3]int
}

func index(m *Mesh) indexed {
	idx := map[vkey]int{}
	var out indexed
	for _, t := range m.Tris {
		var f [3]int
		for i, p := range t {
			k := keyOf(p)
			j, ok := idx[k]
			if !ok {
				j = len(out.verts)
				idx[k] = j
				out.verts = append(out.verts, p)
			}
			f[i] = j
		}
		if f[0] != f[1] && f[1] != f[2] && f[0] != f[2] {
			out.faces = append(out.faces, f)
		}
	}
	return out
}

func ff(f float64) string { return strconv.FormatFloat(f, 'f', 5, 64) }

// WriteOBJ writes a Wavefront OBJ with one object per part.
func WriteOBJ(objs []NamedMesh) []byte {
	var b bytes.Buffer
	b.WriteString("# Makie by nrmluvdev - OBJ export (units: mm)\n")
	base := 1
	for i, o := range objs {
		ix := index(o.Mesh)
		name := o.Name
		if name == "" {
			name = fmt.Sprintf("part_%d", i+1)
		}
		fmt.Fprintf(&b, "o %s\n", strings.ReplaceAll(name, " ", "_"))
		for _, v := range ix.verts {
			fmt.Fprintf(&b, "v %s %s %s\n", ff(v.X), ff(v.Y), ff(v.Z))
		}
		for _, f := range ix.faces {
			fmt.Fprintf(&b, "f %d %d %d\n", f[0]+base, f[1]+base, f[2]+base)
		}
		base += len(ix.verts)
	}
	return b.Bytes()
}

func xmlEsc(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;")
	return r.Replace(s)
}

// Write3MF writes a 3MF package with colours and filament (extruder) assignments.
// assembled=true makes one multi-part object (what AMS / multi-material slicers
// expect for multi-colour prints); otherwise every part is its own object.
// Bambu Studio and OrcaSlicer read Metadata/model_settings.config; PrusaSlicer and
// others read the standard 3MF materials and objects.
func Write3MF(objs []NamedMesh, assembled bool) ([]byte, error) {
	var out bytes.Buffer
	z := zip.NewWriter(&out)
	w, _ := z.Create("[Content_Types].xml")
	w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="model" ContentType="application/vnd.ms-package.3dmanufacturing-3dmodel+xml"/><Default Extension="config" ContentType="text/xml"/></Types>`))
	w, _ = z.Create("_rels/.rels")
	w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Target="/3D/3dmodel.model" Id="rel0" Type="http://schemas.microsoft.com/3dmanufacturing/2013/01/3dmodel"/></Relationships>`))
	w, _ = z.Create("3D/3dmodel.model")
	bw := bufio.NewWriter(w)
	bw.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<model unit="millimeter" xml:lang="en-US" xmlns="http://schemas.microsoft.com/3dmanufacturing/core/2015/02" xmlns:m="http://schemas.microsoft.com/3dmanufacturing/material/2015/02">
<metadata name="Application">Makie by nrmluvdev</metadata>
<resources>
<basematerials id="1">
`)
	for _, o := range objs {
		c := o.Color
		if len(c) != 7 {
			c = "#3B82F6"
		}
		fmt.Fprintf(bw, `<base name="%s" displaycolor="%sFF"/>`+"\n", xmlEsc(o.Name), strings.ToUpper(c))
	}
	bw.WriteString("</basematerials>\n")
	for i, o := range objs {
		ix := index(o.Mesh)
		fmt.Fprintf(bw, `<object id="%d" type="model" name="%s" pid="1" pindex="%d"><mesh><vertices>`+"\n", i+2, xmlEsc(o.Name), i)
		for _, v := range ix.verts {
			fmt.Fprintf(bw, `<vertex x="%s" y="%s" z="%s"/>`+"\n", ff(v.X), ff(v.Y), ff(v.Z))
		}
		bw.WriteString("</vertices><triangles>\n")
		for _, f := range ix.faces {
			fmt.Fprintf(bw, `<triangle v1="%d" v2="%d" v3="%d"/>`+"\n", f[0], f[1], f[2])
		}
		bw.WriteString("</triangles></mesh></object>\n")
	}
	asm := len(objs) + 2
	if assembled {
		fmt.Fprintf(bw, `<object id="%d" type="model" name="Assembly"><components>`+"\n", asm)
		for i := range objs {
			fmt.Fprintf(bw, `<component objectid="%d"/>`+"\n", i+2)
		}
		bw.WriteString("</components></object>\n")
	}
	bw.WriteString("</resources>\n<build>\n")
	if assembled {
		fmt.Fprintf(bw, `<item objectid="%d"/>`+"\n", asm)
	} else {
		for i := range objs {
			fmt.Fprintf(bw, `<item objectid="%d"/>`+"\n", i+2)
		}
	}
	bw.WriteString("</build>\n</model>\n")
	if err := bw.Flush(); err != nil {
		return nil, err
	}
	// slicer settings: which filament (AMS slot / extruder) each part uses
	cw, _ := z.Create("Metadata/model_settings.config")
	var cfg strings.Builder
	cfg.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n<config>\n")
	ext := func(o NamedMesh) int {
		if o.Extruder < 1 {
			return 1
		}
		return o.Extruder
	}
	if assembled {
		fmt.Fprintf(&cfg, "  <object id=\"%d\">\n    <metadata key=\"name\" value=\"Assembly\"/>\n    <metadata key=\"extruder\" value=\"%d\"/>\n", asm, ext(objs[0]))
		for i, o := range objs {
			fmt.Fprintf(&cfg, "    <part id=\"%d\" subtype=\"normal_part\">\n      <metadata key=\"name\" value=\"%s\"/>\n      <metadata key=\"extruder\" value=\"%d\"/>\n    </part>\n", i+2, xmlEsc(o.Name), ext(o))
		}
		cfg.WriteString("  </object>\n")
	} else {
		for i, o := range objs {
			fmt.Fprintf(&cfg, "  <object id=\"%d\">\n    <metadata key=\"name\" value=\"%s\"/>\n    <metadata key=\"extruder\" value=\"%d\"/>\n    <part id=\"%d\" subtype=\"normal_part\">\n      <metadata key=\"extruder\" value=\"%d\"/>\n    </part>\n  </object>\n", i+2, xmlEsc(o.Name), ext(o), i+2, ext(o))
		}
	}
	cfg.WriteString("</config>\n")
	cw.Write([]byte(cfg.String()))
	if err := z.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// WriteSTLPerFilament zips one STL per filament slot (works with every slicer:
// load all files at once as parts of one object and assign filaments).
func WriteSTLPerFilament(objs []NamedMesh, base string) ([]byte, error) {
	groups := map[int][]NamedMesh{}
	var order []int
	for _, o := range objs {
		e := o.Extruder
		if e < 1 {
			e = 1
		}
		if _, ok := groups[e]; !ok {
			order = append(order, e)
		}
		groups[e] = append(groups[e], o)
	}
	var out bytes.Buffer
	z := zip.NewWriter(&out)
	for _, e := range order {
		all := &Mesh{}
		for _, o := range groups[e] {
			all.Append(o.Mesh)
		}
		w, err := z.Create(fmt.Sprintf("%s - filament %d.stl", base, e))
		if err != nil {
			return nil, err
		}
		w.Write(WriteSTL([]NamedMesh{{Mesh: all}}))
	}
	if err := z.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// WriteSTLZip zips one STL per part.
func WriteSTLZip(objs []NamedMesh) ([]byte, error) {
	var out bytes.Buffer
	z := zip.NewWriter(&out)
	used := map[string]int{}
	for _, o := range objs {
		name := o.Name
		if name == "" {
			name = "part"
		}
		used[name]++
		if used[name] > 1 {
			name = fmt.Sprintf("%s %d", name, used[name])
		}
		w, err := z.Create(name + ".stl")
		if err != nil {
			return nil, err
		}
		w.Write(WriteSTL([]NamedMesh{o}))
	}
	if err := z.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// ReadSTL parses binary or ASCII STL.
func ReadSTL(b []byte) (*Mesh, error) {
	if len(b) >= 84 {
		n := binary.LittleEndian.Uint32(b[80:])
		if int64(84)+int64(n)*50 == int64(len(b)) {
			m := &Mesh{Tris: make([]Tri, 0, n)}
			f := func(o int) float64 { return float64(math.Float32frombits(binary.LittleEndian.Uint32(b[o:]))) }
			for i := 0; i < int(n); i++ {
				o := 84 + i*50 + 12
				var t Tri
				for v := 0; v < 3; v++ {
					t[v] = Vec3{f(o + v*12), f(o + v*12 + 4), f(o + v*12 + 8)}
				}
				m.Tris = append(m.Tris, t)
			}
			return m, nil
		}
	}
	// ASCII
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	m := &Mesh{}
	var cur []Vec3
	for sc.Scan() {
		fs := strings.Fields(sc.Text())
		if len(fs) == 4 && strings.EqualFold(fs[0], "vertex") {
			x, _ := strconv.ParseFloat(fs[1], 64)
			y, _ := strconv.ParseFloat(fs[2], 64)
			zz, _ := strconv.ParseFloat(fs[3], 64)
			cur = append(cur, Vec3{x, y, zz})
			if len(cur) == 3 {
				m.Tris = append(m.Tris, Tri{cur[0], cur[1], cur[2]})
				cur = cur[:0]
			}
		}
	}
	if len(m.Tris) == 0 {
		return nil, errors.New("no triangles found — is this an STL file?")
	}
	return m, nil
}

// ReadOBJ parses the vertices and faces of an OBJ file.
func ReadOBJ(b []byte) (*Mesh, error) {
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	var vs []Vec3
	m := &Mesh{}
	for sc.Scan() {
		fs := strings.Fields(sc.Text())
		if len(fs) == 0 {
			continue
		}
		switch fs[0] {
		case "v":
			if len(fs) >= 4 {
				x, _ := strconv.ParseFloat(fs[1], 64)
				y, _ := strconv.ParseFloat(fs[2], 64)
				z, _ := strconv.ParseFloat(fs[3], 64)
				vs = append(vs, Vec3{x, y, z})
			}
		case "f":
			var ids []int
			for _, t := range fs[1:] {
				s := strings.Split(t, "/")[0]
				i, err := strconv.Atoi(s)
				if err != nil {
					continue
				}
				if i < 0 {
					i = len(vs) + i
				} else {
					i--
				}
				if i >= 0 && i < len(vs) {
					ids = append(ids, i)
				}
			}
			for k := 1; k+1 < len(ids); k++ {
				m.Add(vs[ids[0]], vs[ids[k]], vs[ids[k+1]])
			}
		}
	}
	if len(m.Tris) == 0 {
		return nil, errors.New("no faces found in OBJ file")
	}
	return m, nil
}

// Stats summarises a mesh for the print check.
type Stats struct {
	Volume    float64    `json:"volume"`
	Area      float64    `json:"area"`
	Tris      int        `json:"tris"`
	Min       [3]float64 `json:"min"`
	Max       [3]float64 `json:"max"`
	OpenEdges int        `json:"openEdges"`
	Overhang  float64    `json:"overhangArea"`
}

func MeshStats(m *Mesh) Stats {
	lo, hi := m.Bounds()
	s := Stats{Volume: math.Abs(m.Volume()), Area: m.Area(), Tris: len(m.Tris), Min: [3]float64{lo.X, lo.Y, lo.Z}, Max: [3]float64{hi.X, hi.Y, hi.Z}}
	cnt := map[ekey]int{}
	for _, t := range m.Tris {
		n := faceNormal(t)
		if n.Z < -0.7072 && math.Min(t[0].Z, math.Min(t[1].Z, t[2].Z)) > lo.Z+0.05 {
			s.Overhang += t[1].Sub(t[0]).Cross(t[2].Sub(t[0])).Len() / 2
		}
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
	for _, c := range cnt {
		if c%2 == 1 {
			s.OpenEdges++
		}
	}
	return s
}
