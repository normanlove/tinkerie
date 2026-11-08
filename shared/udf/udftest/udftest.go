// Package udftest builds small UDF images for tests, laid out the way
// Microsoft's CDIMAGE lays out a Windows ISO. Part of Tinkerie, by nrmluvdev.
package udftest

import "encoding/binary"

const sector = 2048

// mkudf builds a small UDF image the way CDIMAGE lays out a Windows ISO:
// recognition sequence, anchor at 256, one partition, file entries with
// short allocation descriptors. Some files use the less common forms
// (extended entries, data inside the entry, many extents).
type mk struct {
	img  []byte
	next int64 // next free block in the partition
}

const partStart = 300

func (m *mk) blk(n int64) []byte {
	need := (partStart + n + 1) * sector
	for int64(len(m.img)) < need {
		m.img = append(m.img, make([]byte, sector)...)
	}
	return m.img[(partStart+n)*sector : (partStart+n+1)*sector]
}

func (m *mk) alloc(n int64) int64 {
	s := m.next
	m.next += n
	m.blk(m.next)
	return s
}

func settag(b []byte, id uint16, loc uint32) {
	binary.LittleEndian.PutUint16(b, id)
	binary.LittleEndian.PutUint16(b[2:], 2)
	binary.LittleEndian.PutUint32(b[12:], loc)
	var s byte
	for i := 0; i < 16; i++ {
		if i != 4 {
			s += b[i]
		}
	}
	b[4] = s
}

type ext struct{ off, n int64 }

func (m *mk) data(d []byte) []ext {
	n := (int64(len(d)) + sector - 1) / sector
	at := m.alloc(n)
	copy(m.img[(partStart+at)*sector:], d)
	return []ext{{off: at, n: int64(len(d))}}
}

// fe writes a file entry; kind: 0 normal, 1 extended, 2 embedded.
func (m *mk) fe(dir bool, d []byte, kind int, split int) int64 {
	lbn := m.alloc(1)
	var ads []byte
	flags := uint16(0)
	if kind == 2 {
		ads, flags = d, 3
	} else {
		ex := m.data(d)
		if split > 1 && len(d) > 0 { // cut into several extents, in reverse order on disc
			ex = nil
			step := (len(d)/split + sector - 1) / sector * sector
			var parts [][]byte
			for i := 0; i < len(d); i += step {
				parts = append(parts, d[i:min(len(d), i+step)])
			}
			locs := make([]int64, len(parts))
			for i := len(parts) - 1; i >= 0; i-- {
				locs[i] = m.data(parts[i])[0].off
			}
			for i := range parts {
				ex = append(ex, ext{off: locs[i], n: int64(len(parts[i]))})
			}
		}
		if len(d) == 0 {
			ex = nil
		}
		for _, e := range ex {
			a := make([]byte, 8)
			binary.LittleEndian.PutUint32(a, uint32(e.n))
			binary.LittleEndian.PutUint32(a[4:], uint32(e.off))
			ads = append(ads, a...)
		}
	}
	b := m.blk(lbn)
	ft := byte(5)
	if dir {
		ft = 4
	}
	b[27] = ft
	binary.LittleEndian.PutUint16(b[34:], flags)
	binary.LittleEndian.PutUint64(b[56:], uint64(len(d)))
	id, base := uint16(261), 176
	if kind == 1 {
		id, base = 266, 216
		binary.LittleEndian.PutUint32(b[212:], uint32(len(ads)))
	} else {
		binary.LittleEndian.PutUint32(b[172:], uint32(len(ads)))
	}
	copy(b[base:], ads)
	settag(b, id, uint32(lbn))
	return lbn
}

func fid(name string, icb int64, chars byte) []byte {
	var nb []byte
	if name != "" {
		nb = append([]byte{8}, name...)
		for _, r := range name {
			if r > 255 {
				nb = []byte{16}
				for _, u := range []rune(name) {
					nb = append(nb, byte(u>>8), byte(u))
				}
				break
			}
		}
	}
	b := make([]byte, (38+len(nb)+3)&^3)
	b[18], b[19] = chars, byte(len(nb))
	binary.LittleEndian.PutUint32(b[20:], sector)
	binary.LittleEndian.PutUint32(b[24:], uint32(icb))
	copy(b[38:], nb)
	settag(b, 257, 0)
	return b
}

// Node is a file or folder for Build.
type Node struct {
	Name  string
	Data  []byte
	Kind  int // 0 file entry, 1 extended file entry, 2 data inside the entry
	Split int // cut the data into this many extents
	Kids  []*Node
	Dir   bool
}

func (m *mk) put(n *Node) int64 {
	if !n.Dir {
		return m.fe(false, n.Data, n.Kind, n.Split)
	}
	d := fid("", 0, 0x0A) // parent
	for _, k := range n.Kids {
		c := byte(0)
		if k.Dir {
			c = 2
		}
		d = append(d, fid(k.Name, m.put(k), c)...)
	}
	d = append(d, fid("deleted.txt", 0, 0x04)...)
	return m.fe(true, d, 0, 0)
}

// Build makes the image.
func Build(root *Node) []byte {
	m := &mk{img: make([]byte, 300*sector)}
	copy(m.img[16*sector:], "\x00BEA01\x01")
	copy(m.img[17*sector:], "\x00NSR02\x01")
	copy(m.img[18*sector:], "\x00TEA01\x01")
	fsd := m.alloc(1)
	rootICB := m.put(root)
	f := m.blk(fsd)
	binary.LittleEndian.PutUint32(f[400:], sector)
	binary.LittleEndian.PutUint32(f[404:], uint32(rootICB))
	settag(f, 256, uint32(fsd))
	pvd := m.img[32*sector:]
	copy(pvd[24:], "\x08CCCOMA_X64FRE_EN-GB_DV9")
	pvd[24+31] = 24
	settag(pvd, 1, 32)
	pd := m.img[33*sector:]
	binary.LittleEndian.PutUint32(pd[188:], partStart)
	binary.LittleEndian.PutUint32(pd[192:], uint32(m.next+10))
	settag(pd, 5, 33)
	lvd := m.img[34*sector:]
	binary.LittleEndian.PutUint32(lvd[212:], sector)
	binary.LittleEndian.PutUint32(lvd[248:], sector)
	binary.LittleEndian.PutUint32(lvd[252:], uint32(fsd))
	binary.LittleEndian.PutUint32(lvd[268:], 1)
	lvd[440], lvd[441] = 1, 6
	settag(lvd, 6, 34)
	settag(m.img[35*sector:], 8, 35)
	av := m.img[256*sector:]
	binary.LittleEndian.PutUint32(av[16:], 4*sector)
	binary.LittleEndian.PutUint32(av[20:], 32)
	settag(av, 2, 256)
	m.blk(m.next + 10)
	return m.img
}
