package xz

import "io"

// ---------------------------------------------------------------- range coder

type rangeDec struct {
	buf  []byte
	p    int
	rng  uint32
	code uint32
}

func (r *rangeDec) init(b []byte) bool {
	if len(b) < 5 || b[0] != 0 {
		return false
	}
	r.buf, r.p = b, 5
	r.rng = 0xFFFFFFFF
	r.code = uint32(b[1])<<24 | uint32(b[2])<<16 | uint32(b[3])<<8 | uint32(b[4])
	return true
}

func (r *rangeDec) next() uint32 {
	if r.p < len(r.buf) {
		b := r.buf[r.p]
		r.p++
		return uint32(b)
	}
	r.p++ // reading past the end marks the chunk as corrupt
	return 0
}

func (r *rangeDec) bit(p *uint16) uint32 {
	bound := (r.rng >> 11) * uint32(*p)
	var b uint32
	if r.code < bound {
		r.rng = bound
		*p += (2048 - *p) >> 5
	} else {
		r.rng -= bound
		r.code -= bound
		*p -= *p >> 5
		b = 1
	}
	if r.rng < 1<<24 {
		r.rng <<= 8
		r.code = r.code<<8 | r.next()
	}
	return b
}

func (r *rangeDec) direct(n uint32) uint32 {
	var res uint32
	for ; n > 0; n-- {
		r.rng >>= 1
		r.code -= r.rng
		t := 0 - (r.code >> 31)
		r.code += r.rng & t
		res = res<<1 + (t + 1)
		if r.rng < 1<<24 {
			r.rng <<= 8
			r.code = r.code<<8 | r.next()
		}
	}
	return res
}

func (r *rangeDec) tree(probs []uint16, bits uint) uint32 {
	m := uint32(1)
	for i := uint(0); i < bits; i++ {
		m = m<<1 | r.bit(&probs[m])
	}
	return m - 1<<bits
}

func (r *rangeDec) revTree(probs []uint16, bits uint) uint32 {
	m, sym := uint32(1), uint32(0)
	for i := uint(0); i < bits; i++ {
		b := r.bit(&probs[m])
		m = m<<1 | b
		sym |= b << i
	}
	return sym
}

// ---------------------------------------------------------------- LZMA state

const probInit = 1024

type lenDec struct {
	choice, choice2 uint16
	low, mid        [16][8]uint16
	high            [256]uint16
}

func (l *lenDec) reset() {
	l.choice, l.choice2 = probInit, probInit
	for i := range l.low {
		for j := range l.low[i] {
			l.low[i][j], l.mid[i][j] = probInit, probInit
		}
	}
	for i := range l.high {
		l.high[i] = probInit
	}
}

func (l *lenDec) decode(r *rangeDec, ps uint32) uint32 {
	if r.bit(&l.choice) == 0 {
		return r.tree(l.low[ps][:], 3)
	}
	if r.bit(&l.choice2) == 0 {
		return 8 + r.tree(l.mid[ps][:], 3)
	}
	return 16 + r.tree(l.high[:], 8)
}

type lzma2 struct {
	in        *countReader
	dictSize  uint32
	dict      []byte
	pos       int   // write position in dict
	total     int64 // bytes written since the last dictionary reset
	started   bool  // a dictionary reset has happened
	haveProp  bool
	needState bool

	lc, lp, pb uint
	state      uint32
	rep        [4]uint32

	isMatch    [12 << 4]uint16
	isRep      [12]uint16
	isRepG0    [12]uint16
	isRepG1    [12]uint16
	isRepG2    [12]uint16
	isRep0Long [12 << 4]uint16
	posSlot    [4][64]uint16
	posDec     [115]uint16
	align      [16]uint16
	lit        []uint16
	mlen, rlen lenDec

	rc   rangeDec
	cbuf []byte
	out2 []byte
}

func newLZMA2(dict uint32) *lzma2 {
	n := int(dict)
	if n < 4<<20 {
		n = 4 << 20 // at least two maximum-size chunks
	}
	return &lzma2{dictSize: dict, dict: make([]byte, n), cbuf: make([]byte, 0, 1<<16)}
}

func (z *lzma2) reset() {
	z.pos, z.total, z.started, z.haveProp, z.needState = 0, 0, false, false, false
}

func (z *lzma2) resetState() {
	z.state = 0
	z.rep = [4]uint32{}
	for _, a := range [][]uint16{z.isMatch[:], z.isRep[:], z.isRepG0[:], z.isRepG1[:], z.isRepG2[:], z.isRep0Long[:], z.posDec[:], z.align[:], z.lit} {
		for i := range a {
			a[i] = probInit
		}
	}
	for i := range z.posSlot {
		for j := range z.posSlot[i] {
			z.posSlot[i][j] = probInit
		}
	}
	z.mlen.reset()
	z.rlen.reset()
}

func (z *lzma2) setProps(b byte) error {
	if b >= 9*5*5 {
		return ErrCorrupt
	}
	z.lc = uint(b % 9)
	b /= 9
	z.lp = uint(b % 5)
	z.pb = uint(b / 5)
	if z.lc+z.lp > 4 || z.pb > 4 {
		return ErrCorrupt
	}
	n := 0x300 << (z.lc + z.lp)
	if cap(z.lit) < n {
		z.lit = make([]uint16, n)
	}
	z.lit = z.lit[:n]
	z.haveProp = true
	return nil
}

func (z *lzma2) byteAt(dist uint32) byte { // dist >= 1
	i := z.pos - int(dist)
	if i < 0 {
		i += len(z.dict)
	}
	return z.dict[i]
}

func (z *lzma2) put(b byte) {
	z.dict[z.pos] = b
	z.pos++
	if z.pos == len(z.dict) {
		z.pos = 0
	}
	z.total++
}

// result returns the n bytes just written, as one or two slices.
func (z *lzma2) result(n int) []byte {
	start := z.pos - n
	if start >= 0 {
		z.out2 = nil
		return z.dict[start:z.pos]
	}
	// Wrapped: hand out the tail now, the head on the following call.
	z.out2 = z.dict[:z.pos]
	return z.dict[len(z.dict)+start:]
}

// chunk decodes one LZMA2 chunk. done is true at the end-of-data marker.
func (z *lzma2) chunk() (out []byte, done bool, err error) {
	if z.out2 != nil {
		o := z.out2
		z.out2 = nil
		return o, false, nil
	}
	ctl, err := z.in.ReadByte()
	if err != nil {
		return nil, false, ErrCorrupt
	}
	if ctl == 0 {
		return nil, true, nil
	}
	if ctl == 1 || ctl == 2 {
		var h [2]byte
		if _, err := z.in.Read(h[:]); err != nil {
			return nil, false, ErrCorrupt
		}
		n := int(h[0])<<8 | int(h[1]) + 1
		if ctl == 1 {
			z.pos, z.total, z.started = 0, 0, true
		} else if !z.started {
			return nil, false, ErrCorrupt
		}
		for left := n; left > 0; {
			k := left
			if k > len(z.dict)-z.pos {
				k = len(z.dict) - z.pos
			}
			if _, err := io.ReadFull(z.in, z.dict[z.pos:z.pos+k]); err != nil {
				return nil, false, ErrCorrupt
			}
			z.pos += k
			if z.pos == len(z.dict) {
				z.pos = 0
			}
			left -= k
		}
		z.total += int64(n)
		z.needState = true
		return z.result(n), false, nil
	}
	if ctl < 0x80 {
		return nil, false, ErrCorrupt
	}
	var h [4]byte
	if _, err := z.in.Read(h[:]); err != nil {
		return nil, false, ErrCorrupt
	}
	usize := int(ctl&0x1F)<<16 | int(h[0])<<8 | int(h[1]) + 1
	csize := int(h[2])<<8 | int(h[3]) + 1
	mode := (ctl >> 5) & 3
	if mode == 3 {
		z.pos, z.total, z.started = 0, 0, true
	} else if !z.started {
		return nil, false, ErrCorrupt
	}
	if mode >= 2 {
		p, err := z.in.ReadByte()
		if err != nil {
			return nil, false, ErrCorrupt
		}
		if err := z.setProps(p); err != nil {
			return nil, false, err
		}
	} else if !z.haveProp {
		return nil, false, ErrCorrupt
	}
	if mode >= 1 {
		z.resetState()
		z.needState = false
	}
	if cap(z.cbuf) < csize {
		z.cbuf = make([]byte, csize)
	}
	z.cbuf = z.cbuf[:csize]
	if _, err := io.ReadFull(z.in, z.cbuf); err != nil {
		return nil, false, ErrCorrupt
	}
	if !z.rc.init(z.cbuf) {
		return nil, false, ErrCorrupt
	}
	if err := z.decode(usize); err != nil {
		return nil, false, err
	}
	if z.rc.p > len(z.cbuf) {
		return nil, false, ErrCorrupt
	}
	return z.result(usize), false, nil
}

func (z *lzma2) decode(n int) error {
	r := &z.rc
	pbMask := uint32(1)<<z.pb - 1
	lpMask := uint32(1)<<z.lp - 1
	end := z.total + int64(n)
	for z.total < end {
		ps := uint32(z.total) & pbMask
		st := z.state
		if r.bit(&z.isMatch[st<<4+ps]) == 0 {
			// literal
			var prev uint32
			if z.total > 0 {
				prev = uint32(z.byteAt(1))
			}
			ls := (uint32(z.total)&lpMask)<<z.lc + prev>>(8-z.lc)
			probs := z.lit[0x300*ls : 0x300*ls+0x300]
			sym := uint32(1)
			if st >= 7 {
				mb := uint32(z.byteAt(z.rep[0] + 1))
				for sym < 0x100 {
					mbit := (mb >> 7) & 1
					mb <<= 1
					b := r.bit(&probs[(1+mbit)<<8+sym])
					sym = sym<<1 | b
					if mbit != b {
						break
					}
				}
			}
			for sym < 0x100 {
				sym = sym<<1 | r.bit(&probs[sym])
			}
			z.put(byte(sym))
			switch {
			case st < 4:
				z.state = 0
			case st < 10:
				z.state = st - 3
			default:
				z.state = st - 6
			}
			continue
		}
		var ln uint32
		if r.bit(&z.isRep[st]) != 0 {
			if z.total == 0 {
				return ErrCorrupt
			}
			if r.bit(&z.isRepG0[st]) == 0 {
				if r.bit(&z.isRep0Long[st<<4+ps]) == 0 {
					if st < 7 {
						z.state = 9
					} else {
						z.state = 11
					}
					z.put(z.byteAt(z.rep[0] + 1))
					continue
				}
			} else {
				var d uint32
				if r.bit(&z.isRepG1[st]) == 0 {
					d = z.rep[1]
				} else {
					if r.bit(&z.isRepG2[st]) == 0 {
						d = z.rep[2]
					} else {
						d = z.rep[3]
						z.rep[3] = z.rep[2]
					}
					z.rep[2] = z.rep[1]
				}
				z.rep[1] = z.rep[0]
				z.rep[0] = d
			}
			ln = z.rlen.decode(r, ps)
			if st < 7 {
				z.state = 8
			} else {
				z.state = 11
			}
		} else {
			z.rep[3], z.rep[2], z.rep[1] = z.rep[2], z.rep[1], z.rep[0]
			ln = z.mlen.decode(r, ps)
			if st < 7 {
				z.state = 7
			} else {
				z.state = 10
			}
			lst := ln
			if lst > 3 {
				lst = 3
			}
			slot := r.tree(z.posSlot[lst][:], 6)
			var dist uint32
			if slot < 4 {
				dist = slot
			} else {
				nd := slot>>1 - 1
				dist = (2 | slot&1) << nd
				if slot < 14 {
					dist += r.revTree(z.posDec[dist-slot:], uint(nd))
				} else {
					dist += r.direct(nd-4) << 4
					dist += r.revTree(z.align[:], 4)
				}
			}
			if dist == 0xFFFFFFFF {
				return ErrCorrupt // end marker isn't allowed inside LZMA2
			}
			z.rep[0] = dist
		}
		ln += 2
		dist := z.rep[0] + 1
		if int64(dist) > z.total || dist > uint32(len(z.dict)) || dist > z.dictSize {
			return ErrCorrupt
		}
		if z.total+int64(ln) > end {
			return ErrCorrupt // a match can't run past the chunk
		}
		z.copyMatch(int(dist), int(ln))
	}
	return nil
}

func (z *lzma2) copyMatch(dist, n int) {
	src := z.pos - dist
	if src < 0 {
		src += len(z.dict)
	}
	z.total += int64(n)
	// Fast path: no wrap on either side and no overlap.
	if dist >= n && src+n <= len(z.dict) && z.pos+n <= len(z.dict) {
		copy(z.dict[z.pos:z.pos+n], z.dict[src:src+n])
		z.pos += n
		if z.pos == len(z.dict) {
			z.pos = 0
		}
		return
	}
	d := z.dict
	for ; n > 0; n-- {
		d[z.pos] = d[src]
		z.pos++
		if z.pos == len(d) {
			z.pos = 0
		}
		src++
		if src == len(d) {
			src = 0
		}
	}
}
