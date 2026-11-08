// Package exfat creates an exFAT file system holding a known set of files,
// in pure Go, following Microsoft's exFAT specification. Flashie uses it for
// the second partition of a Windows installer USB drive, where install.wim
// is too big for FAT32 (4 GB limit). Windows Setup reads exFAT.
//
// Build lays everything out at once: every folder and file gets one
// contiguous run of clusters. The caller then writes each file's data at
// the offset Build returns.
//
// Part of Tinkerie, by nrmluvdev.
package exfat

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf16"
)

// Device is the drive; offsets are bytes from its start.
type Device interface {
	io.ReaderAt
	io.WriterAt
}

// File is one file (or folder) to put on the volume. Parent folders are
// made automatically.
type File struct {
	Path string
	Size int64
	Dir  bool
}

var le = binary.LittleEndian

const (
	bps       = 512
	entrySize = 32
)

type node struct {
	name    string
	dir     bool
	size    int64 // file size, or folder size in bytes (whole clusters)
	kids    []*node
	cluster uint32
	nclus   uint32
	path    string
}

// Build makes the file system on the partition at base (bytes) of size
// bytes and returns where each file's data starts on the device.
func Build(dev Device, base, size int64, label string, files []File, now time.Time) (map[string]int64, error) {
	volSec := size / bps
	var shift uint
	switch {
	case size <= 256<<20:
		shift = 3 // 4 KiB
	case size <= 32<<30:
		shift = 6 // 32 KiB
	default:
		shift = 8 // 128 KiB
	}
	spc := int64(1) << shift
	cs := spc * bps
	const fatOff = 2048 // 1 MiB: flash-friendly alignment
	est := (volSec - fatOff) / spc
	fatLen := ((est+2)*4 + bps - 1) / bps
	heap := (fatOff + fatLen + 2047) / 2048 * 2048
	if volSec <= heap+spc*16 {
		return nil, errors.New("the partition is too small for exFAT")
	}
	ccount := (volSec - heap) / spc
	if ccount > 0xFFFFFFF5 {
		return nil, errors.New("the partition is too big")
	}

	// The folder tree.
	root := &node{dir: true}
	byPath := map[string]*node{"": root}
	var get func(p string) (*node, error)
	get = func(p string) (*node, error) {
		if n, ok := byPath[strings.ToLower(p)]; ok {
			if !n.dir {
				return nil, fmt.Errorf("%s is a file, not a folder", p)
			}
			return n, nil
		}
		parent, name := "", p
		if i := strings.LastIndexByte(p, '/'); i >= 0 {
			parent, name = p[:i], p[i+1:]
		}
		pn, err := get(parent)
		if err != nil {
			return nil, err
		}
		n := &node{name: name, dir: true, path: p}
		pn.kids = append(pn.kids, n)
		byPath[strings.ToLower(p)] = n
		return n, nil
	}
	for _, f := range files {
		p := strings.Trim(strings.ReplaceAll(f.Path, "\\", "/"), "/")
		if p == "" {
			continue
		}
		for _, part := range strings.Split(p, "/") {
			if err := checkName(part); err != nil {
				return nil, err
			}
		}
		if f.Dir {
			if _, err := get(p); err != nil {
				return nil, err
			}
			continue
		}
		if _, dup := byPath[strings.ToLower(p)]; dup {
			return nil, fmt.Errorf("%s is listed twice", p)
		}
		parent, name := "", p
		if i := strings.LastIndexByte(p, '/'); i >= 0 {
			parent, name = p[:i], p[i+1:]
		}
		pn, err := get(parent)
		if err != nil {
			return nil, err
		}
		n := &node{name: name, size: f.Size, path: p}
		pn.kids = append(pn.kids, n)
		byPath[strings.ToLower(p)] = n
	}

	up := upcaseTable()
	upData := make([]byte, len(up)*2)
	for i, u := range up {
		le.PutUint16(upData[i*2:], u)
	}
	var upSum uint32
	for _, b := range upData {
		upSum = (upSum>>1 | upSum<<31) + uint32(b)
	}

	// Allocation: bitmap, up-case table, root, then folders, then files.
	next := uint32(2)
	clusters := func(bytes int64) uint32 { return uint32((bytes + cs - 1) / cs) }
	take := func(n uint32) (uint32, error) {
		if n == 0 {
			return 0, nil
		}
		if int64(next-2)+int64(n) > ccount {
			return 0, errors.New("the files don't fit on the partition")
		}
		c := next
		next += n
		return c, nil
	}
	bmLen := (ccount + 7) / 8
	bmClus, _ := take(clusters(bmLen))
	upClus, _ := take(clusters(int64(len(upData))))
	var dirs []*node
	var walk func(n *node)
	walk = func(n *node) {
		sort.Slice(n.kids, func(i, j int) bool { return n.kids[i].name < n.kids[j].name })
		ents := 0
		if n == root {
			ents = 3 // volume label, bitmap, up-case table
		}
		for _, k := range n.kids {
			ents += 2 + (len(utf16.Encode([]rune(k.name)))+14)/15
		}
		n.size = max(cs, (int64(ents)*entrySize+cs-1)/cs*cs)
		dirs = append(dirs, n)
		for _, k := range n.kids {
			if k.dir {
				walk(k)
			}
		}
	}
	walk(root)
	for _, d := range dirs {
		d.nclus = clusters(d.size)
		var err error
		if d.cluster, err = take(d.nclus); err != nil {
			return nil, err
		}
	}
	out := map[string]int64{}
	off := func(c uint32) int64 { return base + heap*bps + int64(c-2)*cs }
	var fileList []*node
	for _, n := range byPath {
		if !n.dir {
			fileList = append(fileList, n)
		}
	}
	sort.Slice(fileList, func(i, j int) bool { return fileList[i].path < fileList[j].path })
	for _, n := range fileList {
		n.nclus = clusters(n.size)
		var err error
		if n.cluster, err = take(n.nclus); err != nil {
			return nil, err
		}
		if n.size > 0 {
			out[n.path] = off(n.cluster)
		}
	}
	used := int64(next - 2)

	// Boot region (and its backup).
	boot := make([]byte, 12*bps)
	b := boot[:bps]
	copy(b, []byte{0xEB, 0x76, 0x90})
	copy(b[3:], "EXFAT   ")
	le.PutUint64(b[64:], uint64(base/bps))
	le.PutUint64(b[72:], uint64(volSec))
	le.PutUint32(b[80:], fatOff)
	le.PutUint32(b[84:], uint32(fatLen))
	le.PutUint32(b[88:], uint32(heap))
	le.PutUint32(b[92:], uint32(ccount))
	le.PutUint32(b[96:], root.cluster)
	le.PutUint32(b[100:], uint32(now.UnixNano())^0x6e726d6c) // serial
	le.PutUint16(b[104:], 0x0100)
	b[108], b[109], b[110], b[111] = 9, byte(shift), 1, 0x80
	b[112] = byte(used * 100 / ccount)
	for i := 120; i < 510; i++ {
		b[i] = 0xF4 // halt: not bootable on BIOS
	}
	b[510], b[511] = 0x55, 0xAA
	for s := 1; s <= 8; s++ { // extended boot sectors
		le.PutUint32(boot[s*bps+508:], 0xAA550000)
	}
	var sum uint32
	for i := 0; i < 11*bps; i++ {
		if i == 106 || i == 107 || i == 112 {
			continue
		}
		sum = (sum>>1 | sum<<31) + uint32(boot[i])
	}
	for i := 0; i < bps/4; i++ {
		le.PutUint32(boot[11*bps+i*4:], sum)
	}
	write := func(p []byte, at int64) error {
		_, err := dev.WriteAt(p, at)
		return err
	}
	if err := write(boot, base); err != nil {
		return nil, err
	}
	if err := write(boot, base+12*bps); err != nil {
		return nil, err
	}

	// FAT: a chain for everything allocated.
	fat := make([]byte, fatLen*bps)
	le.PutUint32(fat, 0xFFFFFFF8)
	le.PutUint32(fat[4:], 0xFFFFFFFF)
	chain := func(c, n uint32) {
		for i := uint32(0); i < n; i++ {
			v := c + i + 1
			if i == n-1 {
				v = 0xFFFFFFFF
			}
			le.PutUint32(fat[(c+i)*4:], v)
		}
	}
	chain(bmClus, clusters(bmLen))
	chain(upClus, clusters(int64(len(upData))))
	for _, d := range dirs {
		chain(d.cluster, d.nclus)
	}
	for _, n := range fileList {
		if n.nclus > 0 {
			chain(n.cluster, n.nclus)
		}
	}
	// Blank the gap between the boot region and the FAT too.
	if err := write(make([]byte, (fatOff-24)*bps), base+24*bps); err != nil {
		return nil, err
	}
	if err := write(fat, base+fatOff*bps); err != nil {
		return nil, err
	}

	// Allocation bitmap.
	bm := make([]byte, int64(clusters(bmLen))*cs)
	for i := int64(0); i < used; i++ {
		bm[i/8] |= 1 << (i % 8)
	}
	if err := write(bm, off(bmClus)); err != nil {
		return nil, err
	}
	upBuf := make([]byte, int64(clusters(int64(len(upData))))*cs)
	copy(upBuf, upData)
	if err := write(upBuf, off(upClus)); err != nil {
		return nil, err
	}

	// Folders.
	ts := timestamp(now)
	for _, d := range dirs {
		buf := make([]byte, d.size)
		p := 0
		if d == root {
			lab := utf16.Encode([]rune(label))
			if len(lab) > 11 {
				lab = lab[:11]
			}
			e := buf[p:]
			e[0], e[1] = 0x83, byte(len(lab))
			for i, u := range lab {
				le.PutUint16(e[2+i*2:], u)
			}
			p += entrySize
			e = buf[p:]
			e[0] = 0x81
			le.PutUint32(e[20:], bmClus)
			le.PutUint64(e[24:], uint64(bmLen))
			p += entrySize
			e = buf[p:]
			e[0] = 0x82
			le.PutUint32(e[4:], upSum)
			le.PutUint32(e[20:], upClus)
			le.PutUint64(e[24:], uint64(len(upData)))
			p += entrySize
		}
		for _, k := range d.kids {
			set := entrySet(k, up, ts)
			copy(buf[p:], set)
			p += len(set)
		}
		if err := write(buf, off(d.cluster)); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func checkName(n string) error {
	if n == "" || n == "." || n == ".." || len(utf16.Encode([]rune(n))) > 255 {
		return fmt.Errorf("%q can't be used as a file name", n)
	}
	for _, r := range n {
		if r < 32 || strings.ContainsRune(`"*/:<>?\|`, r) {
			return fmt.Errorf("%q can't be used as a file name", n)
		}
	}
	return nil
}

// timestamp is exFAT's packed local date and time (here: UTC).
func timestamp(t time.Time) uint32 {
	t = t.UTC()
	if t.Year() < 1980 {
		t = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	return uint32(t.Year()-1980)<<25 | uint32(t.Month())<<21 | uint32(t.Day())<<16 |
		uint32(t.Hour())<<11 | uint32(t.Minute())<<5 | uint32(t.Second()/2)
}

func entrySet(n *node, up []uint16, ts uint32) []byte {
	name := utf16.Encode([]rune(n.name))
	nNames := (len(name) + 14) / 15
	set := make([]byte, (2+nNames)*entrySize)
	f := set
	f[0], f[1] = 0x85, byte(1+nNames)
	attr := uint16(0x20)
	if n.dir {
		attr = 0x10
	}
	le.PutUint16(f[4:], attr)
	le.PutUint32(f[8:], ts)
	le.PutUint32(f[12:], ts)
	le.PutUint32(f[16:], ts)
	f[22], f[23], f[24] = 0x80, 0x80, 0x80 // UTC offsets valid, +0
	s := set[entrySize:]
	s[0] = 0xC0
	s[1] = 1 // allocation possible; FAT chain valid
	s[3] = byte(len(name))
	var h uint16
	for _, u := range name {
		if int(u) < len(up) {
			u = up[u]
		}
		h = (h>>1 | h<<15) + u&0xFF
		h = (h>>1 | h<<15) + u>>8
	}
	le.PutUint16(s[4:], h)
	le.PutUint64(s[8:], uint64(n.size))
	if n.nclus > 0 {
		le.PutUint32(s[20:], n.cluster)
	}
	le.PutUint64(s[24:], uint64(n.size))
	for i := 0; i < nNames; i++ {
		e := set[(2+i)*entrySize:]
		e[0] = 0xC1
		for k := 0; k < 15 && i*15+k < len(name); k++ {
			le.PutUint16(e[2+k*2:], name[i*15+k])
		}
	}
	var cs uint16
	for i, b := range set {
		if i == 2 || i == 3 {
			continue
		}
		cs = (cs>>1 | cs<<15) + uint16(b)
	}
	le.PutUint16(f[2:], cs)
	return set
}

// upcaseTable maps every BMP character to its upper case (a full,
// uncompressed table, which the specification allows).
func upcaseTable() []uint16 {
	t := make([]uint16, 0x10000)
	for i := range t {
		r := rune(i)
		u := unicode.ToUpper(r)
		if u > 0xFFFF || (i >= 0xD800 && i < 0xE000) {
			u = r
		}
		t[i] = uint16(u)
	}
	return t
}
