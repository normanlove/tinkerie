package flashie

// pageDev lets the FAT and exFAT code read and write any byte range of a raw
// drive, which only takes whole, aligned 4 KiB pieces. Changes are kept in
// memory until Flush, then written (and later checked) in large runs.

import (
	"bytes"
	"errors"
	"sort"
)

const pageSize = 4096

type pageDev struct {
	rd    rawDisk
	size  int64
	pages map[int64][]byte
	dirty map[int64]bool
	saved map[int64]bool // written by Flush, to check in Verify
}

func newPageDev(rd rawDisk, size int64) *pageDev {
	return &pageDev{rd: rd, size: size, pages: map[int64][]byte{}, dirty: map[int64]bool{}, saved: map[int64]bool{}}
}

func (p *pageDev) plen(i int64) int {
	return int(min(pageSize, p.size-i*pageSize))
}

func (p *pageDev) page(i int64, whole bool) ([]byte, error) {
	if b, ok := p.pages[i]; ok {
		return b, nil
	}
	if i < 0 || i*pageSize >= p.size {
		return nil, errors.New("past the end of the drive")
	}
	b := alignedBuf(pageSize)[:p.plen(i)]
	if !whole {
		if _, err := p.rd.ReadAt(b, i*pageSize); err != nil {
			return nil, diskErr("read", err)
		}
	}
	p.pages[i] = b
	return b, nil
}

func (p *pageDev) ReadAt(b []byte, off int64) (int, error) {
	n := 0
	for n < len(b) {
		i, o := (off+int64(n))/pageSize, int((off+int64(n))%pageSize)
		pg, err := p.page(i, false)
		if err != nil {
			return n, err
		}
		if o >= len(pg) {
			return n, errors.New("past the end of the drive")
		}
		n += copy(b[n:], pg[o:])
	}
	return n, nil
}

func (p *pageDev) WriteAt(b []byte, off int64) (int, error) {
	n := 0
	for n < len(b) {
		i, o := (off+int64(n))/pageSize, int((off+int64(n))%pageSize)
		whole := o == 0 && len(b)-n >= p.plen(i)
		pg, err := p.page(i, whole)
		if err != nil {
			return n, err
		}
		if o >= len(pg) {
			return n, errors.New("past the end of the drive")
		}
		n += copy(pg[o:], b[n:])
		p.dirty[i] = true
	}
	return n, nil
}

// runs groups pages into runs of consecutive pages, up to 4 MiB each.
func runs(set map[int64]bool) [][2]int64 {
	idx := make([]int64, 0, len(set))
	for i := range set {
		idx = append(idx, i)
	}
	sort.Slice(idx, func(a, b int) bool { return idx[a] < idx[b] })
	var out [][2]int64
	for _, i := range idx {
		if k := len(out) - 1; k >= 0 && out[k][1] == i && out[k][1]-out[k][0] < blockSize/pageSize {
			out[k][1]++
		} else {
			out = append(out, [2]int64{i, i + 1})
		}
	}
	return out
}

func (p *pageDev) gather(r [2]int64, buf []byte) []byte {
	n := 0
	for i := r[0]; i < r[1]; i++ {
		n += copy(buf[n:], p.pages[i])
	}
	return buf[:n]
}

// Flush writes every changed page.
func (p *pageDev) Flush() error {
	buf := alignedBuf(blockSize)
	for _, r := range runs(p.dirty) {
		if _, err := p.rd.WriteAt(p.gather(r, buf), r[0]*pageSize); err != nil {
			return diskErr("write", err)
		}
		for i := r[0]; i < r[1]; i++ {
			p.saved[i] = true
		}
	}
	p.dirty = map[int64]bool{}
	return nil
}

// Verify reads back everything Flush wrote and compares it.
func (p *pageDev) Verify() error {
	buf, got := alignedBuf(blockSize), alignedBuf(blockSize)
	for _, r := range runs(p.saved) {
		want := p.gather(r, buf)
		g := got[:len(want)]
		if _, err := p.rd.ReadAt(g, r[0]*pageSize); err != nil {
			return diskErr("read back", err)
		}
		if !bytes.Equal(g, want) {
			return errors.New("checking failed: the drive didn't keep what was written. The card may be faulty or fake — try another one")
		}
	}
	return nil
}
