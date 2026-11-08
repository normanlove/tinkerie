// Package fat reads, edits and creates FAT16/FAT32 file systems on a raw
// device, in pure Go. Flashie uses it to put OS customisation files on a
// Raspberry Pi boot partition, and to format Windows installer USB drives.
//
// Long file names (VFAT) are read and written. New files are given
// contiguous clusters where possible, so a big file is one extent that can
// be written straight to the drive (CreateContig).
//
// Part of Tinkerie, by nrmluvdev.
package fat

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf16"
)

// Device is the drive (or image file) holding the file system. Offsets are
// bytes from the start of the drive.
type Device interface {
	io.ReaderAt
	io.WriterAt
}

// FS is an open FAT16 or FAT32 file system.
type FS struct {
	dev         Device
	base        int64 // partition start on the device, bytes
	bps         int64 // bytes per sector
	clusterSize int64
	rsvd        int64 // reserved sectors
	nfats       int64
	fatSz       int64 // sectors per FAT
	fat32       bool
	rootStart   int64 // FAT16: root directory region, bytes from base
	rootSize    int64
	rootClus    uint32
	dataStart   int64 // bytes from base
	nclus       uint32
	fsinfo      int64 // FAT32 FSInfo sector, 0 if none
	fat         []uint32
	dirty       map[int64]bool // FAT sectors to write
	hint        uint32
	Now         func() time.Time
}

var le = binary.LittleEndian

const (
	attrRO    = 0x01
	attrHid   = 0x02
	attrSys   = 0x04
	attrVol   = 0x08
	attrDir   = 0x10
	attrArc   = 0x20
	attrLFN   = 0x0F
	eoc32     = 0x0FFFFFFF
	maxChain  = 1 << 28
	entrySize = 32
)

// Open reads the FAT file system starting at byte offset base.
func Open(dev Device, base int64) (*FS, error) {
	bs := make([]byte, 512)
	if _, err := dev.ReadAt(bs, base); err != nil {
		return nil, err
	}
	if bs[510] != 0x55 || bs[511] != 0xAA {
		return nil, errors.New("not a FAT file system (no boot signature)")
	}
	f := &FS{dev: dev, base: base, dirty: map[int64]bool{}, Now: time.Now}
	f.bps = int64(le.Uint16(bs[11:]))
	spc := int64(bs[13])
	f.rsvd = int64(le.Uint16(bs[14:]))
	f.nfats = int64(bs[16])
	rootEnt := int64(le.Uint16(bs[17:]))
	tot := int64(le.Uint16(bs[19:]))
	if tot == 0 {
		tot = int64(le.Uint32(bs[32:]))
	}
	f.fatSz = int64(le.Uint16(bs[22:]))
	if f.fatSz == 0 {
		f.fatSz = int64(le.Uint32(bs[36:]))
	}
	switch f.bps {
	case 512, 1024, 2048, 4096:
	default:
		return nil, errors.New("not a FAT file system (bad sector size)")
	}
	if spc == 0 || spc&(spc-1) != 0 || f.nfats == 0 || f.rsvd == 0 || f.fatSz == 0 {
		return nil, errors.New("not a FAT file system")
	}
	f.clusterSize = f.bps * spc
	rootSecs := (rootEnt*entrySize + f.bps - 1) / f.bps
	dataSec := tot - f.rsvd - f.nfats*f.fatSz - rootSecs
	if dataSec <= 0 {
		return nil, errors.New("not a FAT file system (bad size)")
	}
	n := dataSec / spc
	switch {
	case n < 4085:
		return nil, errors.New("FAT12 file systems aren't supported")
	case n < 65525:
		f.rootStart = (f.rsvd + f.nfats*f.fatSz) * f.bps
		f.rootSize = rootSecs * f.bps
	default:
		f.fat32 = true
		f.rootClus = le.Uint32(bs[44:])
		if fi := int64(le.Uint16(bs[48:])); fi > 0 && fi < f.rsvd {
			f.fsinfo = fi
		}
	}
	f.nclus = uint32(n)
	f.dataStart = (f.rsvd+f.nfats*f.fatSz)*f.bps + rootSecs*f.bps
	ent := int64(2)
	if f.fat32 {
		ent = 4
	}
	if (int64(f.nclus)+2)*ent > f.fatSz*f.bps {
		return nil, errors.New("damaged FAT file system (FAT too small)")
	}
	raw := make([]byte, (int64(f.nclus)+2)*ent)
	if _, err := dev.ReadAt(raw, base+f.rsvd*f.bps); err != nil {
		return nil, err
	}
	f.fat = make([]uint32, f.nclus+2)
	for i := range f.fat {
		if f.fat32 {
			f.fat[i] = le.Uint32(raw[i*4:])
		} else {
			v := uint32(le.Uint16(raw[i*2:]))
			if v >= 0xFFF7 {
				v |= 0x0FFF0000 // same meaning as the FAT32 values
			}
			f.fat[i] = v
		}
	}
	if f.fat32 && (f.rootClus < 2 || f.rootClus >= f.nclus+2) {
		return nil, errors.New("damaged FAT32 file system (bad root directory)")
	}
	f.hint = 2
	return f, nil
}

// FAT32 reports whether the file system is FAT32 (not FAT16).
func (f *FS) FAT32() bool { return f.fat32 }

// ClusterSize is the allocation unit in bytes.
func (f *FS) ClusterSize() int64 { return f.clusterSize }

func (f *FS) next(c uint32) uint32 { return f.fat[c] & 0x0FFFFFFF }

func isEnd(v uint32) bool { return v >= 0x0FFFFFF8 || v < 2 }

func (f *FS) set(c, v uint32) {
	f.fat[c] = f.fat[c]&0xF0000000 | v&0x0FFFFFFF
	ent := int64(2)
	if f.fat32 {
		ent = 4
	}
	f.dirty[int64(c)*ent/f.bps] = true
}

func (f *FS) clusterOff(c uint32) int64 {
	return f.base + f.dataStart + int64(c-2)*f.clusterSize
}

func (f *FS) chain(c uint32) ([]uint32, error) {
	var out []uint32
	for c >= 2 && !isEnd(c) {
		if c >= f.nclus+2 || len(out) > int(f.nclus) {
			return nil, errors.New("damaged FAT file system (bad cluster chain)")
		}
		out = append(out, c)
		c = f.next(c)
	}
	return out, nil
}

// alloc finds n free clusters, contiguous if possible (or only if contig),
// links them into a chain and returns them.
func (f *FS) alloc(n int64, contig bool) ([]uint32, error) {
	if n == 0 {
		return nil, nil
	}
	top := f.nclus + 2
	run := func(from uint32) uint32 {
		var start, have uint32
		for c := from; c < top; c++ {
			if f.fat[c]&0x0FFFFFFF != 0 {
				have = 0
				continue
			}
			if have == 0 {
				start = c
			}
			have++
			if int64(have) == n {
				return start
			}
		}
		return 0
	}
	var cl []uint32
	if s := run(f.hint); s != 0 {
		for i := uint32(0); int64(i) < n; i++ {
			cl = append(cl, s+i)
		}
	} else if s := run(2); s != 0 {
		for i := uint32(0); int64(i) < n; i++ {
			cl = append(cl, s+i)
		}
	} else if !contig {
		for c := uint32(2); c < top && int64(len(cl)) < n; c++ {
			if f.fat[c]&0x0FFFFFFF == 0 {
				cl = append(cl, c)
			}
		}
		if int64(len(cl)) < n {
			return nil, errors.New("the drive is full")
		}
	} else {
		return nil, errors.New("the drive is full (or has no space in one piece)")
	}
	for i, c := range cl {
		if i+1 < len(cl) {
			f.set(c, cl[i+1])
		} else {
			f.set(c, eoc32)
		}
	}
	f.hint = cl[len(cl)-1] + 1
	return cl, nil
}

func (f *FS) free(c uint32) error {
	cl, err := f.chain(c)
	if err != nil {
		return err
	}
	for _, x := range cl {
		f.set(x, 0)
	}
	if len(cl) > 0 && cl[0] < f.hint {
		f.hint = cl[0]
	}
	return nil
}

// FreeBytes is the space left.
func (f *FS) FreeBytes() int64 {
	var n int64
	for c := uint32(2); c < f.nclus+2; c++ {
		if f.fat[c]&0x0FFFFFFF == 0 {
			n++
		}
	}
	return n * f.clusterSize
}

// ----------------------------------------------------------- directories

type dir struct {
	first uint32  // 0 = the FAT16 root region
	data  []byte  // the whole directory
	offs  []int64 // device offset of each cluster (or the root region)
	chunk int64
}

type entry struct {
	name    string
	short   [11]byte
	attr    byte
	cluster uint32
	size    uint32
	idx     int // index of the short entry
	first   int // index of its first long-name entry
}

func (f *FS) openDir(c uint32) (*dir, error) {
	d := &dir{first: c}
	if c == 0 && !f.fat32 {
		d.data = make([]byte, f.rootSize)
		d.offs = []int64{f.base + f.rootStart}
		d.chunk = f.rootSize
		_, err := f.dev.ReadAt(d.data, d.offs[0])
		return d, err
	}
	if c == 0 {
		c = f.rootClus
		d.first = c
	}
	cl, err := f.chain(c)
	if err != nil {
		return nil, err
	}
	if len(cl) == 0 || len(cl) > 65536*entrySize/int(f.clusterSize)+1 {
		return nil, errors.New("damaged directory")
	}
	d.chunk = f.clusterSize
	d.data = make([]byte, int64(len(cl))*f.clusterSize)
	for i, x := range cl {
		o := f.clusterOff(x)
		d.offs = append(d.offs, o)
		if _, err := f.dev.ReadAt(d.data[int64(i)*f.clusterSize:int64(i+1)*f.clusterSize], o); err != nil {
			return nil, err
		}
	}
	return d, nil
}

// writeBack writes entries [from, to) of the directory to the device.
func (f *FS) writeBack(d *dir, from, to int) error {
	for i := int64(from) * entrySize / d.chunk; i <= int64(to*entrySize-1)/d.chunk && i < int64(len(d.offs)); i++ {
		if _, err := f.dev.WriteAt(d.data[i*d.chunk:(i+1)*d.chunk], d.offs[i]); err != nil {
			return err
		}
	}
	return nil
}

func lfnSum(s []byte) byte {
	var sum byte
	for _, c := range s[:11] {
		sum = (sum&1)<<7 + sum>>1 + c
	}
	return sum
}

var lfnPos = []int{1, 3, 5, 7, 9, 14, 16, 18, 20, 22, 24, 28, 30}

func (d *dir) entries() []entry {
	var out []entry
	var lfn []uint16
	lfnSumV, lfnFirst, want := byte(0), 0, 0
	for i := 0; i*entrySize < len(d.data); i++ {
		e := d.data[i*entrySize : (i+1)*entrySize]
		if e[0] == 0 {
			break
		}
		if e[0] == 0xE5 {
			lfn, want = nil, 0
			continue
		}
		if e[11]&0x3F == attrLFN {
			ord := int(e[0] & 0x3F)
			if e[0]&0x40 != 0 {
				lfn, want, lfnSumV, lfnFirst = make([]uint16, ord*13), ord, e[13], i
			}
			if want == 0 || ord != want || e[13] != lfnSumV || ord == 0 {
				lfn, want = nil, 0
				continue
			}
			for k, p := range lfnPos {
				lfn[(ord-1)*13+k] = le.Uint16(e[p:])
			}
			want--
			if want == 0 {
				want = -1 // complete: the short entry comes next
			}
			continue
		}
		en := entry{attr: e[11], cluster: uint32(le.Uint16(e[20:]))<<16 | uint32(le.Uint16(e[26:])), size: le.Uint32(e[28:]), idx: i, first: i}
		copy(en.short[:], e[:11])
		if want == -1 && lfnSum(e) == lfnSumV {
			n := 0
			for n < len(lfn) && lfn[n] != 0 {
				n++
			}
			en.name, en.first = string(utf16.Decode(lfn[:n])), lfnFirst
		} else {
			en.name = shortToName(e[:11], e[12])
		}
		lfn, want = nil, 0
		if en.attr&attrVol != 0 {
			continue
		}
		out = append(out, en)
	}
	return out
}

func shortToName(s []byte, nt byte) string {
	b := strings.TrimRight(string(s[:8]), " ")
	x := strings.TrimRight(string(s[8:11]), " ")
	if len(b) > 0 && b[0] == 0x05 {
		b = "\xE5" + b[1:]
	}
	if nt&0x08 != 0 {
		b = strings.ToLower(b)
	}
	if nt&0x10 != 0 {
		x = strings.ToLower(x)
	}
	if x != "" {
		return b + "." + x
	}
	return b
}

func (d *dir) find(name string) (entry, bool) {
	for _, e := range d.entries() {
		if strings.EqualFold(e.name, name) || strings.EqualFold(shortToName(e.short[:], 0), name) {
			return e, true
		}
	}
	return entry{}, false
}

func splitPath(p string) []string {
	var out []string
	for _, s := range strings.Split(strings.ReplaceAll(p, "\\", "/"), "/") {
		if s != "" && s != "." {
			out = append(out, s)
		}
	}
	return out
}

// walk returns the directory holding path's last element.
func (f *FS) walk(parts []string, create bool) (*dir, error) {
	d, err := f.openDir(0)
	if err != nil {
		return nil, err
	}
	for _, p := range parts {
		e, ok := d.find(p)
		if !ok {
			if !create {
				return nil, fmt.Errorf("%s: not found", p)
			}
			if e, err = f.mkdirIn(d, p); err != nil {
				return nil, err
			}
		} else if e.attr&attrDir == 0 {
			return nil, fmt.Errorf("%s isn't a folder", p)
		}
		if d, err = f.openDir(e.cluster); err != nil {
			return nil, err
		}
	}
	return d, nil
}

// Exists reports whether path is in the file system.
func (f *FS) Exists(path string) bool {
	parts := splitPath(path)
	if len(parts) == 0 {
		return true
	}
	d, err := f.walk(parts[:len(parts)-1], false)
	if err != nil {
		return false
	}
	_, ok := d.find(parts[len(parts)-1])
	return ok
}

// ReadFile returns a file's contents.
func (f *FS) ReadFile(path string) ([]byte, error) {
	parts := splitPath(path)
	if len(parts) == 0 {
		return nil, errors.New("no file name")
	}
	d, err := f.walk(parts[:len(parts)-1], false)
	if err != nil {
		return nil, err
	}
	e, ok := d.find(parts[len(parts)-1])
	if !ok {
		return nil, fmt.Errorf("%s: not found", path)
	}
	if e.attr&attrDir != 0 {
		return nil, fmt.Errorf("%s is a folder", path)
	}
	cl, err := f.chain(e.cluster)
	if err != nil {
		return nil, err
	}
	if int64(len(cl))*f.clusterSize < int64(e.size) {
		return nil, errors.New("damaged file (shorter than its size)")
	}
	out := make([]byte, e.size)
	for i, c := range cl {
		lo := int64(i) * f.clusterSize
		if lo >= int64(e.size) {
			break
		}
		hi := min(lo+f.clusterSize, int64(e.size))
		if _, err := f.dev.ReadAt(out[lo:hi], f.clusterOff(c)); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// List returns the names in a folder ("" for the top).
func (f *FS) List(path string) ([]string, error) {
	d, err := f.walk(splitPath(path), false)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range d.entries() {
		if e.name != "." && e.name != ".." {
			out = append(out, e.name)
		}
	}
	return out, nil
}

// --------------------------------------------------------- creating names

const shortOK = "$%'-_@~`!(){}^#&"

func validShortChar(r rune) bool {
	return r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune(shortOK, r)
}

// shortFor works out the 8.3 name. If the long name fits 8.3 exactly
// (allowing all-lowercase parts, flagged in NTRes as Windows does), no long
// name entries are needed.
func shortFor(name string, taken func([11]byte) bool) (s [11]byte, nt byte, needLFN bool) {
	for i := range s {
		s[i] = ' '
	}
	base, ext := name, ""
	if i := strings.LastIndexByte(name, '.'); i > 0 {
		base, ext = name[:i], name[i+1:]
	}
	fits := func(p string, max int) (bool, bool) { // fits, lower
		if len(p) == 0 && max == 8 || len(p) > max {
			return false, false
		}
		up, lo := false, false
		for _, r := range p {
			switch {
			case r >= 'a' && r <= 'z':
				lo = true
			case r >= 'A' && r <= 'Z':
				up = true
			case !validShortChar(r):
				return false, false
			}
		}
		return !(up && lo), lo
	}
	bf, bl := fits(base, 8)
	ef, el := fits(ext, 3)
	if bf && ef && name != "." && name != ".." {
		copy(s[:8], strings.ToUpper(base))
		copy(s[8:], strings.ToUpper(ext))
		if bl {
			nt |= 0x08
		}
		if el {
			nt |= 0x10
		}
		if !taken(s) {
			return s, nt, false
		}
	}
	clean := func(p string, max int) string {
		var b strings.Builder
		for _, r := range strings.ToUpper(p) {
			if r == ' ' || r == '.' {
				continue
			}
			if !validShortChar(r) {
				r = '_'
			}
			b.WriteRune(r)
			if b.Len() >= max {
				break
			}
		}
		return b.String()
	}
	if i := strings.LastIndexByte(name, '.'); i > 0 {
		base, ext = name[:i], name[i+1:]
	} else {
		base, ext = name, ""
	}
	cb, ce := clean(base, 8), clean(ext, 3)
	if cb == "" {
		cb = "FILE"
	}
	for n := 1; n < 1000000; n++ {
		tail := fmt.Sprintf("~%d", n)
		b := cb
		if len(b) > 8-len(tail) {
			b = b[:8-len(tail)]
		}
		for i := range s {
			s[i] = ' '
		}
		copy(s[:8], b+tail)
		copy(s[8:], ce)
		if !taken(s) {
			return s, 0, true
		}
	}
	return s, 0, true
}

func dosTime(t time.Time) (uint16, uint16) {
	if t.Year() < 1980 {
		t = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	d := uint16((t.Year()-1980)<<9 | int(t.Month())<<5 | t.Day())
	tm := uint16(t.Hour()<<11 | t.Minute()<<5 | t.Second()/2)
	return d, tm
}

func (f *FS) makeEntries(d *dir, name string, attr byte, cluster, size uint32) ([][]byte, error) {
	if name == "" || len(name) > 255 || strings.ContainsAny(name, `/\:*?"<>|`) {
		return nil, fmt.Errorf("%q can't be used as a file name", name)
	}
	ents := d.entries()
	taken := func(s [11]byte) bool {
		for _, e := range ents {
			if e.short == s {
				return true
			}
		}
		return false
	}
	short, nt, needLFN := shortFor(name, taken)
	se := make([]byte, entrySize)
	copy(se, short[:])
	if se[0] == 0xE5 {
		se[0] = 0x05
	}
	se[11], se[12] = attr, nt
	dd, tt := dosTime(f.Now())
	le.PutUint16(se[14:], tt)
	le.PutUint16(se[16:], dd)
	le.PutUint16(se[18:], dd)
	le.PutUint16(se[20:], uint16(cluster>>16))
	le.PutUint16(se[22:], tt)
	le.PutUint16(se[24:], dd)
	le.PutUint16(se[26:], uint16(cluster))
	le.PutUint32(se[28:], size)
	var out [][]byte
	if needLFN {
		u := utf16.Encode([]rune(name))
		n := (len(u) + 12) / 13
		sum := lfnSum(se)
		for ord := n; ord >= 1; ord-- {
			e := make([]byte, entrySize)
			e[0] = byte(ord)
			if ord == n {
				e[0] |= 0x40
			}
			e[11], e[13] = attrLFN, sum
			for k, p := range lfnPos {
				i := (ord-1)*13 + k
				var c uint16 = 0xFFFF
				if i < len(u) {
					c = u[i]
				} else if i == len(u) {
					c = 0
				}
				le.PutUint16(e[p:], c)
			}
			out = append(out, e)
		}
	}
	return append(out, se), nil
}

// addEntries puts the entries into free slots, growing the directory if
// needed. It returns the index of the last (short) entry.
func (f *FS) addEntries(d *dir, es [][]byte) (int, error) {
	n := len(es)
	slots := len(d.data) / entrySize
	start, run := -1, 0
	for i := 0; i < slots; i++ {
		b := d.data[i*entrySize]
		if b == 0 || b == 0xE5 {
			if run == 0 {
				start = i
			}
			run++
			if run == n {
				break
			}
		} else {
			run = 0
		}
	}
	if run < n {
		if d.first == 0 && !f.fat32 {
			return 0, errors.New("the top folder of the drive is full")
		}
		need := int64(n-run)*entrySize + f.clusterSize - 1
		cl, err := f.alloc(need/f.clusterSize, false)
		if err != nil {
			return 0, err
		}
		last, _ := f.chain(d.first)
		f.set(last[len(last)-1], cl[0])
		for _, c := range cl {
			z := make([]byte, f.clusterSize)
			if _, err := f.dev.WriteAt(z, f.clusterOff(c)); err != nil {
				return 0, err
			}
			d.offs = append(d.offs, f.clusterOff(c))
			d.data = append(d.data, z...)
		}
		if run == 0 {
			start = slots
		}
	}
	for i, e := range es {
		copy(d.data[(start+i)*entrySize:], e)
	}
	return start + n - 1, f.writeBack(d, start, start+n)
}

func (f *FS) mkdirIn(d *dir, name string) (entry, error) {
	cl, err := f.alloc(1, false)
	if err != nil {
		return entry{}, err
	}
	z := make([]byte, f.clusterSize)
	dd, tt := dosTime(f.Now())
	dot := func(at int, n string, c uint32) {
		e := z[at:]
		copy(e, "           ")
		copy(e, n)
		e[11] = attrDir
		le.PutUint16(e[14:], tt)
		le.PutUint16(e[16:], dd)
		le.PutUint16(e[18:], dd)
		le.PutUint16(e[20:], uint16(c>>16))
		le.PutUint16(e[22:], tt)
		le.PutUint16(e[24:], dd)
		le.PutUint16(e[26:], uint16(c))
	}
	parent := d.first
	if f.fat32 && parent == f.rootClus {
		parent = 0 // ".." of a top-level folder points at 0
	}
	dot(0, ".", cl[0])
	dot(32, "..", parent)
	if _, err := f.dev.WriteAt(z, f.clusterOff(cl[0])); err != nil {
		return entry{}, err
	}
	es, err := f.makeEntries(d, name, attrDir, cl[0], 0)
	if err != nil {
		return entry{}, err
	}
	if _, err := f.addEntries(d, es); err != nil {
		return entry{}, err
	}
	return entry{name: name, attr: attrDir, cluster: cl[0]}, nil
}

// MkdirAll makes a folder and any missing parents.
func (f *FS) MkdirAll(path string) error {
	_, err := f.walk(splitPath(path), true)
	return err
}

func (f *FS) place(path string, size int64, contig bool) (*dir, entry, []uint32, error) {
	parts := splitPath(path)
	if len(parts) == 0 {
		return nil, entry{}, nil, errors.New("no file name")
	}
	if size >= 1<<32 {
		return nil, entry{}, nil, fmt.Errorf("%s is 4 GB or bigger, too big for FAT32", parts[len(parts)-1])
	}
	d, err := f.walk(parts[:len(parts)-1], true)
	if err != nil {
		return nil, entry{}, nil, err
	}
	name := parts[len(parts)-1]
	old, exists := d.find(name)
	if exists {
		if old.attr&attrDir != 0 {
			return nil, entry{}, nil, fmt.Errorf("%s is a folder", path)
		}
		if err := f.free(old.cluster); err != nil {
			return nil, entry{}, nil, err
		}
	}
	cl, err := f.alloc((size+f.clusterSize-1)/f.clusterSize, contig)
	if err != nil {
		return nil, entry{}, nil, err
	}
	var first uint32
	if len(cl) > 0 {
		first = cl[0]
	}
	if exists {
		e := d.data[old.idx*entrySize:]
		le.PutUint16(e[20:], uint16(first>>16))
		le.PutUint16(e[26:], uint16(first))
		le.PutUint32(e[28:], uint32(size))
		dd, tt := dosTime(f.Now())
		le.PutUint16(e[22:], tt)
		le.PutUint16(e[24:], dd)
		e[11] |= attrArc
		return d, old, cl, f.writeBack(d, old.idx, old.idx+1)
	}
	es, err := f.makeEntries(d, name, attrArc, first, uint32(size))
	if err != nil {
		return nil, entry{}, nil, err
	}
	idx, err := f.addEntries(d, es)
	return d, entry{name: name, idx: idx, cluster: first, size: uint32(size)}, cl, err
}

// WriteFile creates or replaces a file.
func (f *FS) WriteFile(path string, data []byte) error {
	_, _, cl, err := f.place(path, int64(len(data)), false)
	if err != nil {
		return err
	}
	for i, c := range cl {
		buf := make([]byte, f.clusterSize)
		copy(buf, data[min(int64(i)*f.clusterSize, int64(len(data))):])
		if _, err := f.dev.WriteAt(buf, f.clusterOff(c)); err != nil {
			return err
		}
	}
	return nil
}

// CreateContig creates a file of the given size in one contiguous run of
// clusters and returns where its data starts on the device (-1 if empty).
// The caller writes the data there itself; the space after the end of the
// file up to the cluster boundary is the caller's to fill (with zeros).
func (f *FS) CreateContig(path string, size int64) (int64, error) {
	_, _, cl, err := f.place(path, size, true)
	if err != nil || len(cl) == 0 {
		return -1, err
	}
	return f.clusterOff(cl[0]), nil
}

// Flush writes the changed parts of the FAT (to every copy) and FAT32's
// free-space note.
func (f *FS) Flush() error {
	ent := int64(2)
	if f.fat32 {
		ent = 4
	}
	per := f.bps / ent
	for s := range f.dirty {
		buf := make([]byte, f.bps)
		if _, err := f.dev.ReadAt(buf, f.base+(f.rsvd+s)*f.bps); err != nil {
			return err
		}
		for i := int64(0); i < per; i++ {
			c := s*per + i
			if c >= int64(len(f.fat)) {
				break
			}
			if f.fat32 {
				le.PutUint32(buf[i*4:], f.fat[c])
			} else {
				le.PutUint16(buf[i*2:], uint16(f.fat[c]))
			}
		}
		for k := int64(0); k < f.nfats; k++ {
			if _, err := f.dev.WriteAt(buf, f.base+(f.rsvd+k*f.fatSz+s)*f.bps); err != nil {
				return err
			}
		}
	}
	f.dirty = map[int64]bool{}
	if f.fat32 && f.fsinfo > 0 {
		fi := make([]byte, 512)
		if _, err := f.dev.ReadAt(fi, f.base+f.fsinfo*f.bps); err != nil {
			return err
		}
		if le.Uint32(fi) == 0x41615252 && le.Uint32(fi[484:]) == 0x61417272 {
			le.PutUint32(fi[488:], uint32(f.FreeBytes()/f.clusterSize))
			le.PutUint32(fi[492:], f.hint)
			if _, err := f.dev.WriteAt(fi, f.base+f.fsinfo*f.bps); err != nil {
				return err
			}
		}
	}
	return nil
}

// -------------------------------------------------------------- formatting

// Format32 makes an empty FAT32 file system on the partition at base (bytes)
// of size bytes, with 512-byte sectors. The data area starts on a 1 MiB
// boundary of the partition, so clusters are aligned for flash drives.
func Format32(dev Device, base, size int64, label string, serial uint32) (*FS, error) {
	const bps = 512
	tot := size / bps
	var spc int64
	switch {
	case size <= 8<<30:
		spc = 8
	case size <= 16<<30:
		spc = 16
	case size <= 32<<30:
		spc = 32
	default:
		spc = 64
	}
	if tot >= 1<<32 {
		return nil, errors.New("the partition is too big for FAT32 (2 TB at most)")
	}
	fatSz := ((tot/spc+2)*4 + bps - 1) / bps
	rsvd := (32+2*fatSz+2047)/2048*2048 - 2*fatSz
	nclus := (tot - rsvd - 2*fatSz) / spc
	if nclus < 65525 {
		return nil, errors.New("the partition is too small for FAT32")
	}
	if nclus >= 0x0FFFFFF5 {
		return nil, errors.New("the partition is too big for FAT32")
	}
	bs := make([]byte, bps)
	copy(bs, []byte{0xEB, 0x58, 0x90})
	copy(bs[3:], "MSWIN4.1")
	le.PutUint16(bs[11:], bps)
	bs[13] = byte(spc)
	le.PutUint16(bs[14:], uint16(rsvd))
	bs[16] = 2
	bs[21] = 0xF8
	le.PutUint16(bs[24:], 63)
	le.PutUint16(bs[26:], 255)
	le.PutUint32(bs[28:], uint32(base/bps))
	le.PutUint32(bs[32:], uint32(tot))
	le.PutUint32(bs[36:], uint32(fatSz))
	le.PutUint32(bs[44:], 2)
	le.PutUint16(bs[48:], 1)
	le.PutUint16(bs[50:], 6)
	bs[64], bs[66] = 0x80, 0x29
	le.PutUint32(bs[67:], serial)
	lab := volLabel(label)
	copy(bs[71:], lab[:])
	copy(bs[82:], "FAT32   ")
	copy(bs[90:], []byte{0xCD, 0x18}) // not bootable on BIOS: int 18h, try the next device
	bs[510], bs[511] = 0x55, 0xAA
	fi := make([]byte, bps)
	le.PutUint32(fi, 0x41615252)
	le.PutUint32(fi[484:], 0x61417272)
	le.PutUint32(fi[488:], uint32(nclus-1))
	le.PutUint32(fi[492:], 3)
	le.PutUint32(fi[508:], 0xAA550000)
	// Reserved area, FATs and the root cluster start out blank.
	zero := make([]byte, 1<<20)
	clearTo := (rsvd + 2*fatSz + spc) * bps
	for off := int64(0); off < clearTo; off += int64(len(zero)) {
		n := min(int64(len(zero)), clearTo-off)
		if _, err := dev.WriteAt(zero[:n], base+off); err != nil {
			return nil, err
		}
	}
	for _, at := range []int64{0, 6} {
		if _, err := dev.WriteAt(bs, base+at*bps); err != nil {
			return nil, err
		}
		if _, err := dev.WriteAt(fi, base+(at+1)*bps); err != nil {
			return nil, err
		}
	}
	// Sector 2 (and its copy at 8) is the rest of the boot area: signature only.
	s2 := make([]byte, bps)
	s2[510], s2[511] = 0x55, 0xAA
	dev.WriteAt(s2, base+2*bps)
	dev.WriteAt(s2, base+8*bps)
	f0 := make([]byte, 12)
	le.PutUint32(f0, 0x0FFFFFF8)
	le.PutUint32(f0[4:], 0x0FFFFFFF)
	le.PutUint32(f0[8:], 0x0FFFFFFF) // root directory: one cluster
	for k := int64(0); k < 2; k++ {
		if _, err := dev.WriteAt(f0, base+(rsvd+k*fatSz)*bps); err != nil {
			return nil, err
		}
	}
	f, err := Open(dev, base)
	if err != nil {
		return nil, err
	}
	if label != "" {
		d, err := f.openDir(0)
		if err != nil {
			return nil, err
		}
		e := make([]byte, entrySize)
		copy(e, lab[:])
		e[11] = attrVol
		dd, tt := dosTime(f.Now())
		le.PutUint16(e[22:], tt)
		le.PutUint16(e[24:], dd)
		if _, err := f.addEntries(d, [][]byte{e}); err != nil {
			return nil, err
		}
	}
	f.hint = 3
	return f, nil
}

func volLabel(s string) [11]byte {
	var b [11]byte
	for i := range b {
		b[i] = ' '
	}
	if s == "" {
		copy(b[:], "NO NAME")
		return b
	}
	n := 0
	for _, r := range strings.ToUpper(s) {
		if n == 11 {
			break
		}
		if !validShortChar(r) && r != ' ' {
			r = '_'
		}
		b[n] = byte(r)
		n++
	}
	return b
}
