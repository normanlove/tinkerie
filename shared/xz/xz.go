// Package xz is a small streaming .xz decoder (LZMA2 filter only), written
// for Flashie's image writer. Raspberry Pi OS and most SBC images ship as
// .img.xz. Standard library only.
//
// Supported: multiple streams and stream padding, CRC32 / CRC64 / SHA-256 /
// no check, block header sizes, and index verification. Not supported:
// BCJ and delta filters (image files don't use them); they give a clear error.
package xz

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"hash/crc32"
	"hash/crc64"
	"io"
)

var (
	ErrFormat      = errors.New("xz: not an xz file")
	ErrCorrupt     = errors.New("xz: data is corrupt")
	ErrCheck       = errors.New("xz: checksum mismatch (the download is damaged)")
	ErrUnsupported = errors.New("xz: uses a filter this decoder doesn't support")
)

var headerMagic = []byte{0xFD, '7', 'z', 'X', 'Z', 0x00}

var crc64Table = crc64.MakeTable(crc64.ECMA)

// IsXZ reports whether b starts with the xz magic bytes.
func IsXZ(b []byte) bool { return bytes.HasPrefix(b, headerMagic) }

// MaxDict is the largest dictionary the decoder will allocate.
var MaxDict uint32 = 1 << 30

type countReader struct {
	r *bufio.Reader
	n int64
}

func (c *countReader) ReadByte() (byte, error) {
	b, err := c.r.ReadByte()
	if err == nil {
		c.n++
	}
	return b, err
}

func (c *countReader) Read(p []byte) (int, error) {
	n, err := io.ReadFull(c.r, p)
	c.n += int64(n)
	if err == io.ErrUnexpectedEOF {
		err = ErrCorrupt
	}
	return n, err
}

type record struct{ unpadded, uncomp uint64 }

// Reader decompresses an xz stream.
type Reader struct {
	in      *countReader
	err     error
	lz      *lzma2
	out     []byte // pending output (points into the dictionary)
	inBlock bool

	check    byte
	checkLen int
	h        hash.Hash
	h32      uint32
	h64      uint64

	blockStart int64 // input offset where the block header started
	hdrSize    int64
	dataStart  int64
	wantComp   int64 // -1 if unknown
	wantUncomp int64
	uncomp     int64
	records    []record
	streams    int
}

// NewReader returns a Reader. The stream header is checked straight away.
func NewReader(r io.Reader) (*Reader, error) {
	z := &Reader{in: &countReader{r: bufio.NewReaderSize(r, 1<<16)}}
	if err := z.streamHeader(true); err != nil {
		return nil, err
	}
	return z, nil
}

func (z *Reader) streamHeader(first bool) error {
	var h [12]byte
	if _, err := z.in.Read(h[:]); err != nil {
		if first {
			return ErrFormat
		}
		return ErrCorrupt
	}
	if !bytes.Equal(h[:6], headerMagic) {
		if first {
			return ErrFormat
		}
		return ErrCorrupt
	}
	if crc32.ChecksumIEEE(h[6:8]) != binary.LittleEndian.Uint32(h[8:]) || h[6] != 0 || h[7] > 0x0F {
		return ErrCorrupt
	}
	z.check = h[7]
	switch z.check {
	case 0:
		z.checkLen = 0
	case 1:
		z.checkLen = 4
	case 4:
		z.checkLen = 8
	case 0x0A:
		z.checkLen = 32
	default:
		// Unknown check types still have a defined size; we skip them.
		z.checkLen = []int{0, 4, 4, 4, 8, 8, 8, 16, 16, 16, 32, 32, 32, 64, 64, 64}[z.check]
	}
	z.records = z.records[:0]
	z.streams++
	return nil
}

func (z *Reader) readVLI() (uint64, error) {
	var v uint64
	for i := 0; i < 9; i++ {
		b, err := z.in.ReadByte()
		if err != nil {
			return 0, ErrCorrupt
		}
		v |= uint64(b&0x7F) << (7 * i)
		if b&0x80 == 0 {
			if b == 0 && i > 0 {
				return 0, ErrCorrupt
			}
			return v, nil
		}
	}
	return 0, ErrCorrupt
}

func vliFrom(b []byte) (uint64, int, error) {
	var v uint64
	for i := 0; i < 9 && i < len(b); i++ {
		v |= uint64(b[i]&0x7F) << (7 * i)
		if b[i]&0x80 == 0 {
			return v, i + 1, nil
		}
	}
	return 0, 0, ErrCorrupt
}

// startBlock is called with the first header byte already read (non-zero).
func (z *Reader) startBlock(sizeByte byte) error {
	z.blockStart = z.in.n - 1
	size := (int(sizeByte) + 1) * 4
	hdr := make([]byte, size)
	hdr[0] = sizeByte
	if _, err := z.in.Read(hdr[1:]); err != nil {
		return ErrCorrupt
	}
	if crc32.ChecksumIEEE(hdr[:size-4]) != binary.LittleEndian.Uint32(hdr[size-4:]) {
		return ErrCorrupt
	}
	flags := hdr[1]
	if flags&0x3C != 0 {
		return ErrUnsupported
	}
	p := hdr[2 : size-4]
	z.wantComp, z.wantUncomp = -1, -1
	if flags&0x40 != 0 {
		v, n, err := vliFrom(p)
		if err != nil || v == 0 {
			return ErrCorrupt
		}
		z.wantComp, p = int64(v), p[n:]
	}
	if flags&0x80 != 0 {
		v, n, err := vliFrom(p)
		if err != nil {
			return ErrCorrupt
		}
		z.wantUncomp, p = int64(v), p[n:]
	}
	nf := int(flags&3) + 1
	var dictProp byte
	for i := 0; i < nf; i++ {
		id, n, err := vliFrom(p)
		if err != nil {
			return ErrCorrupt
		}
		p = p[n:]
		psz, n, err := vliFrom(p)
		if err != nil || int(psz) > len(p)-n {
			return ErrCorrupt
		}
		props := p[n : n+int(psz)]
		p = p[n+int(psz):]
		if i < nf-1 || id != 0x21 {
			return fmt.Errorf("%w (filter id 0x%x)", ErrUnsupported, id)
		}
		if len(props) != 1 || props[0] > 40 {
			return ErrCorrupt
		}
		dictProp = props[0]
	}
	for _, b := range p {
		if b != 0 {
			return ErrCorrupt
		}
	}
	dict := uint32(0xFFFFFFFF)
	if dictProp < 40 {
		dict = (2 | uint32(dictProp&1)) << (dictProp/2 + 11)
	}
	if dict > MaxDict {
		return fmt.Errorf("xz: dictionary too large (%d MiB)", dict>>20)
	}
	if z.lz == nil || z.lz.dictSize < dict {
		z.lz = newLZMA2(dict)
	} else {
		z.lz.reset()
	}
	z.lz.in = z.in
	z.hdrSize = int64(size)
	z.dataStart = z.in.n
	z.uncomp = 0
	switch z.check {
	case 0x0A:
		z.h = sha256.New()
	default:
		z.h = nil
	}
	z.h32, z.h64 = 0, 0
	z.inBlock = true
	return nil
}

func (z *Reader) endBlock() error {
	comp := z.in.n - z.dataStart
	if z.wantComp >= 0 && comp != z.wantComp {
		return ErrCorrupt
	}
	if z.wantUncomp >= 0 && z.uncomp != z.wantUncomp {
		return ErrCorrupt
	}
	for pad := (4 - comp%4) % 4; pad > 0; pad-- {
		b, err := z.in.ReadByte()
		if err != nil || b != 0 {
			return ErrCorrupt
		}
	}
	sum := make([]byte, z.checkLen)
	if _, err := z.in.Read(sum); err != nil {
		return ErrCorrupt
	}
	switch z.check {
	case 1:
		if binary.LittleEndian.Uint32(sum) != z.h32 {
			return ErrCheck
		}
	case 4:
		if binary.LittleEndian.Uint64(sum) != z.h64 {
			return ErrCheck
		}
	case 0x0A:
		if !bytes.Equal(sum, z.h.Sum(nil)) {
			return ErrCheck
		}
	}
	z.records = append(z.records, record{uint64(z.hdrSize + comp + int64(z.checkLen)), uint64(z.uncomp)})
	z.inBlock = false
	return nil
}

// index reads the index (indicator byte already read) and the stream footer.
func (z *Reader) index() error {
	start := z.in.n - 1
	crc := crc32.NewIEEE()
	crc.Write([]byte{0})
	rd := func() (uint64, error) {
		var v uint64
		for i := 0; i < 9; i++ {
			b, err := z.in.ReadByte()
			if err != nil {
				return 0, ErrCorrupt
			}
			crc.Write([]byte{b})
			v |= uint64(b&0x7F) << (7 * i)
			if b&0x80 == 0 {
				return v, nil
			}
		}
		return 0, ErrCorrupt
	}
	n, err := rd()
	if err != nil || n != uint64(len(z.records)) {
		return ErrCorrupt
	}
	for _, r := range z.records {
		u, err := rd()
		if err != nil || u != r.unpadded {
			return ErrCorrupt
		}
		c, err := rd()
		if err != nil || c != r.uncomp {
			return ErrCorrupt
		}
	}
	for (z.in.n-start)%4 != 0 {
		b, err := z.in.ReadByte()
		if err != nil || b != 0 {
			return ErrCorrupt
		}
		crc.Write([]byte{b})
	}
	var s [4]byte
	if _, err := z.in.Read(s[:]); err != nil || binary.LittleEndian.Uint32(s[:]) != crc.Sum32() {
		return ErrCorrupt
	}
	indexSize := z.in.n - start
	var f [12]byte
	if _, err := z.in.Read(f[:]); err != nil {
		return ErrCorrupt
	}
	if f[10] != 'Y' || f[11] != 'Z' || crc32.ChecksumIEEE(f[4:10]) != binary.LittleEndian.Uint32(f[:4]) {
		return ErrCorrupt
	}
	if (int64(binary.LittleEndian.Uint32(f[4:8]))+1)*4 != indexSize || f[8] != 0 || f[9] != z.check {
		return ErrCorrupt
	}
	return nil
}

// next advances to the next block, or another stream, or EOF.
func (z *Reader) next() error {
	for {
		b, err := z.in.ReadByte()
		if err != nil {
			return ErrCorrupt
		}
		if b != 0 {
			return z.startBlock(b)
		}
		if err := z.index(); err != nil {
			return err
		}
		// Stream padding, then maybe another stream.
		pad := 0
		for {
			p, err := z.in.r.Peek(1)
			if err != nil {
				if pad%4 != 0 {
					return ErrCorrupt
				}
				return io.EOF
			}
			if p[0] != 0 {
				break
			}
			z.in.ReadByte()
			pad++
		}
		if pad%4 != 0 {
			return ErrCorrupt
		}
		if err := z.streamHeader(false); err != nil {
			return err
		}
	}
}

func (z *Reader) Read(p []byte) (int, error) {
	for len(z.out) == 0 {
		if z.err != nil {
			return 0, z.err
		}
		if !z.inBlock {
			if err := z.next(); err != nil {
				z.err = err
				continue
			}
		}
		out, done, err := z.lz.chunk()
		if err != nil {
			z.err = err
			continue
		}
		if len(out) > 0 {
			z.uncomp += int64(len(out))
			switch z.check {
			case 1:
				z.h32 = crc32.Update(z.h32, crc32.IEEETable, out)
			case 4:
				z.h64 = crc64.Update(z.h64, crc64Table, out)
			case 0x0A:
				z.h.Write(out)
			}
			z.out = out
		}
		if done {
			if err := z.endBlock(); err != nil {
				z.err = err
			}
		}
	}
	n := copy(p, z.out)
	z.out = z.out[n:]
	return n, nil
}
