// Package udf lists the files in a UDF disc image (.iso), in pure Go.
// Windows installer ISOs keep their files in UDF; their ISO 9660 part only
// holds a README. Flashie uses this to copy a Windows installer onto a USB
// drive. Reading follows ECMA-167 / OSTA UDF 1.02 to 2.01 (type 1
// partition maps); metadata partitions (UDF 2.5+, Blu-ray) aren't read.
//
// Part of Tinkerie, by nrmluvdev.
package udf

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf16"
)

// Extent is a piece of a file: Len bytes at byte offset Off in the image,
// or Len zero bytes if Zero.
type Extent struct {
	Off  int64
	Len  int64
	Zero bool
}

// File is a file or folder in the image. Path uses forward slashes.
type File struct {
	Path    string
	Dir     bool
	Size    int64
	Extents []Extent
}

const sector = 2048

var le = binary.LittleEndian

type image struct {
	r       io.ReaderAt
	bs      int64 // logical block size
	part    int64 // partition start, in blocks
	partLen int64
	count   int
}

// IsUDF reports whether the image has a UDF volume recognition sequence.
func IsUDF(r io.ReaderAt) bool {
	b := make([]byte, 6)
	for s := int64(16); s < 64; s++ {
		if _, err := r.ReadAt(b, s*sector); err != nil {
			return false
		}
		switch string(b[1:6]) {
		case "NSR02", "NSR03":
			return true
		case "BEA01", "CD001", "CDW02", "BOOT2", "TEA01":
		default:
			if b[0] == 0 && string(b[1:6]) == "\x00\x00\x00\x00\x00" {
				return false
			}
		}
	}
	return false
}

func (im *image) read(off int64, n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := im.r.ReadAt(b, off)
	if err == io.EOF {
		err = errors.New("the image ends too soon")
	}
	return b, err
}

// tag checks a descriptor tag and returns its identifier.
func tag(b []byte) (uint16, bool) {
	if len(b) < 16 {
		return 0, false
	}
	var sum byte
	for i := 0; i < 16; i++ {
		if i != 4 {
			sum += b[i]
		}
	}
	return le.Uint16(b), sum == b[4]
}

// Read lists every file and folder, returning the volume name too.
func Read(r io.ReaderAt) (string, []File, error) {
	if !IsUDF(r) {
		return "", nil, errors.New("not a UDF image")
	}
	im := &image{r: r}
	avdp, err := im.read(256*sector, 512)
	if err != nil {
		return "", nil, err
	}
	if id, ok := tag(avdp); !ok || id != 2 {
		return "", nil, errors.New("UDF anchor not found")
	}
	vdsLen, vdsLoc := int64(le.Uint32(avdp[16:])), int64(le.Uint32(avdp[20:]))
	var fsd [16]byte
	gotPD, gotLVD := false, false
	label := ""
	for i := int64(0); i < vdsLen/sector && i < 64; i++ {
		d, err := im.read((vdsLoc+i)*sector, sector)
		if err != nil {
			return "", nil, err
		}
		id, ok := tag(d)
		if !ok {
			continue
		}
		switch id {
		case 1: // primary volume descriptor
			label = dstring(d[24 : 24+32])
		case 5: // partition descriptor
			im.part, im.partLen = int64(le.Uint32(d[188:])), int64(le.Uint32(d[192:]))
			gotPD = true
		case 6: // logical volume descriptor
			im.bs = int64(le.Uint32(d[212:]))
			copy(fsd[:], d[248:264])
			if l := dstring(d[84 : 84+128]); l != "" {
				label = l
			}
			nmaps := le.Uint32(d[268:])
			if nmaps > 0 && d[440] != 1 {
				return "", nil, errors.New("this UDF version (with a metadata partition) isn't supported")
			}
			gotLVD = true
		case 8: // terminating descriptor
			i = vdsLen
		}
	}
	if !gotPD || !gotLVD || im.bs != sector {
		return "", nil, errors.New("UDF volume descriptors not found")
	}
	fsdLoc := int64(le.Uint32(fsd[4:]))
	f, err := im.read(im.blockOff(fsdLoc), sector)
	if err != nil {
		return "", nil, err
	}
	if id, ok := tag(f); !ok || id != 256 {
		return "", nil, errors.New("UDF file set descriptor not found")
	}
	var out []File
	err = im.walk(int64(le.Uint32(f[404:])), "", 0, &out)
	return label, out, err
}

func (im *image) blockOff(lbn int64) int64 { return (im.part + lbn) * im.bs }

// dstring decodes an OSTA compressed Unicode d-string (last byte = length).
func dstring(b []byte) string {
	n := int(b[len(b)-1])
	if n == 0 || n > len(b)-1 {
		return ""
	}
	return strings.TrimRight(dchars(b[:n]), " \x00")
}

func dchars(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	switch b[0] {
	case 8:
		r := make([]rune, len(b)-1)
		for i, c := range b[1:] {
			r[i] = rune(c)
		}
		return string(r)
	case 16:
		u := make([]uint16, (len(b)-1)/2)
		for i := range u {
			u[i] = binary.BigEndian.Uint16(b[1+2*i:])
		}
		return string(utf16.Decode(u))
	}
	return ""
}

// entry reads a (extended) file entry: whether it's a folder, its size and
// where its data is.
func (im *image) entry(lbn int64) (bool, int64, []Extent, error) {
	off := im.blockOff(lbn)
	b, err := im.read(off, int(im.bs))
	if err != nil {
		return false, 0, nil, err
	}
	id, ok := tag(b)
	var lea, lad, base int
	switch {
	case ok && id == 261:
		lea, lad, base = int(le.Uint32(b[168:])), int(le.Uint32(b[172:])), 176
	case ok && id == 266:
		lea, lad, base = int(le.Uint32(b[208:])), int(le.Uint32(b[212:])), 216
	default:
		return false, 0, nil, fmt.Errorf("bad UDF file entry at block %d", lbn)
	}
	ftype := b[27]
	flags := le.Uint16(b[34:])
	size := int64(le.Uint64(b[56:]))
	if base+lea+lad > len(b) || lea < 0 || lad < 0 {
		return false, 0, nil, errors.New("bad UDF file entry")
	}
	ads := b[base+lea : base+lea+lad]
	adOff := off + int64(base+lea)
	var ext []Extent
	switch flags & 7 {
	case 3: // data inside the entry
		ext = []Extent{{Off: adOff, Len: int64(lad)}}
	case 0, 1: // short_ad, long_ad
		step := 8
		if flags&7 == 1 {
			step = 16
		}
		for guard := 0; guard < 4096; guard++ {
			cont := false
			for i := 0; i+step <= len(ads); i += step {
				l := le.Uint32(ads[i:])
				kind, n := l>>30, int64(l&0x3FFFFFFF)
				pos := int64(le.Uint32(ads[i+4:]))
				if n == 0 {
					break
				}
				if kind == 3 { // the list carries on in another block
					nb, err := im.read(im.blockOff(pos), int(min(n, im.bs)))
					if err != nil {
						return false, 0, nil, err
					}
					if id, ok := tag(nb); !ok || id != 258 {
						return false, 0, nil, errors.New("bad UDF allocation extent")
					}
					la := int(le.Uint32(nb[20:]))
					if 24+la > len(nb) {
						return false, 0, nil, errors.New("bad UDF allocation extent")
					}
					ads, cont = nb[24:24+la], true
					break
				}
				if kind == 0 && pos+(n+im.bs-1)/im.bs > im.partLen {
					return false, 0, nil, errors.New("UDF file data lies outside the partition")
				}
				ext = append(ext, Extent{Off: im.blockOff(pos), Len: n, Zero: kind != 0})
			}
			if !cont {
				break
			}
		}
	default:
		return false, 0, nil, errors.New("unsupported UDF allocation type")
	}
	// Trim to the file's size.
	var have int64
	for i := range ext {
		if have+ext[i].Len >= size {
			ext[i].Len = size - have
			ext = ext[:i+1]
			have = size
			break
		}
		have += ext[i].Len
	}
	if have < size {
		return false, 0, nil, errors.New("UDF file is shorter than its size")
	}
	if size == 0 {
		ext = nil
	}
	return ftype == 4, size, ext, nil
}

func (im *image) readAll(ext []Extent, size int64) ([]byte, error) {
	if size > 64<<20 {
		return nil, errors.New("UDF folder too big")
	}
	out := make([]byte, 0, size)
	for _, e := range ext {
		if e.Zero {
			out = append(out, make([]byte, e.Len)...)
			continue
		}
		b, err := im.read(e.Off, int(e.Len))
		if err != nil {
			return nil, err
		}
		out = append(out, b...)
	}
	return out, nil
}

func (im *image) walk(lbn int64, path string, depth int, out *[]File) error {
	if depth > 32 {
		return errors.New("UDF folders nested too deeply")
	}
	dir, size, ext, err := im.entry(lbn)
	if err != nil {
		return err
	}
	if !dir {
		return errors.New("UDF root isn't a folder")
	}
	b, err := im.readAll(ext, size)
	if err != nil {
		return err
	}
	for p := 0; p+38 <= len(b); {
		id, ok := tag(b[p:])
		if !ok || id != 257 {
			return errors.New("bad UDF folder entry")
		}
		chars := b[p+18]
		lfi := int(b[p+19])
		icb := int64(le.Uint32(b[p+24:]))
		liu := int(le.Uint16(b[p+36:]))
		end := p + 38 + liu + lfi
		if end > len(b) {
			return errors.New("bad UDF folder entry")
		}
		name := dchars(b[p+38+liu : end])
		p = (end + 3) &^ 3
		if chars&0x08 != 0 || chars&0x04 != 0 || name == "" { // parent, deleted
			continue
		}
		if strings.ContainsAny(name, "/\\\x00") || name == "." || name == ".." {
			return fmt.Errorf("unsafe file name %q in the image", name)
		}
		im.count++
		if im.count > 100000 {
			return errors.New("too many files in the image")
		}
		full := name
		if path != "" {
			full = path + "/" + name
		}
		if chars&0x02 != 0 {
			*out = append(*out, File{Path: full, Dir: true})
			if err := im.walk(icb, full, depth+1, out); err != nil {
				return err
			}
			continue
		}
		isDir, sz, ex, err := im.entry(icb)
		if err != nil {
			return err
		}
		if isDir {
			return errors.New("bad UDF folder entry")
		}
		*out = append(*out, File{Path: full, Size: sz, Extents: ex})
	}
	return nil
}
