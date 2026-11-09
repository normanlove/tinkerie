// mkres turns Scopie's icon (tools/mkres/icon.ico, PNG images inside) into:
//   - web/icon.png and web/favicon.ico (window and page icon)
//   - rsrc_windows_amd64.syso (icon + version info linked into Scopie.exe)
//
// Run from the project root:  go run ./tools/mkres
// Part of the nrmluvdev maker suite. Created by nrmluvdev.
package main

import (
	"bytes"
	"encoding/binary"
	"log"
	"os"
	"unicode/utf16"
)

const version = "0.9.0.0"

type icon struct {
	w, h int
	data []byte
}

func readICO(b []byte) []icon {
	n := int(binary.LittleEndian.Uint16(b[4:]))
	var out []icon
	for i := 0; i < n; i++ {
		e := b[6+16*i:]
		w, h := int(e[0]), int(e[1])
		if w == 0 {
			w = 256
		}
		if h == 0 {
			h = 256
		}
		size := int(binary.LittleEndian.Uint32(e[8:]))
		off := int(binary.LittleEndian.Uint32(e[12:]))
		out = append(out, icon{w, h, b[off : off+size]})
	}
	return out
}

// ---------------------------------------------------------------- version info

type vbuf struct{ bytes.Buffer }

func (v *vbuf) align() {
	for v.Len()%4 != 0 {
		v.WriteByte(0)
	}
}

func wstr(s string) []byte {
	var b bytes.Buffer
	for _, r := range utf16.Encode([]rune(s)) {
		_ = binary.Write(&b, binary.LittleEndian, r)
	}
	b.Write([]byte{0, 0})
	return b.Bytes()
}

// block writes one VS_VERSIONINFO-style node: wLength, wValueLength, wType, key, value, children.
func block(key string, value []byte, valueLen int, typ uint16, children ...[]byte) []byte {
	var v vbuf
	v.Write([]byte{0, 0})
	_ = binary.Write(&v, binary.LittleEndian, uint16(valueLen))
	_ = binary.Write(&v, binary.LittleEndian, typ)
	v.Write(wstr(key))
	v.align()
	v.Write(value)
	v.align()
	for _, c := range children {
		v.Write(c)
		v.align()
	}
	b := v.Bytes()
	binary.LittleEndian.PutUint16(b, uint16(len(b)))
	return b
}

func versionInfo() []byte {
	var fixed vbuf
	w := func(x uint32) { _ = binary.Write(&fixed, binary.LittleEndian, x) }
	w(0xFEEF04BD)
	w(0x00010000)
	w(3<<16 | 0) // 3.0
	w(0<<16 | 0) // .0.0
	w(3<<16 | 0)
	w(0<<16 | 0)
	w(0x3F)
	w(0)
	w(0x40004) // VOS_NT_WINDOWS32
	w(1)       // VFT_APP
	w(0)
	w(0)
	w(0)
	str := func(k, v string) []byte {
		val := wstr(v)
		return block(k, val, len(val)/2, 1)
	}
	strings := block("040904B0", nil, 0, 1,
		str("CompanyName", "nrmluvdev"),
		str("FileDescription", "Scopie - oscilloscope program"),
		str("FileVersion", version),
		str("InternalName", "Scopie"),
		str("LegalCopyright", "Copyright (C) 2026 nrmluvdev - GPL-3.0-or-later"),
		str("OriginalFilename", "Scopie.exe"),
		str("ProductName", "Scopie"),
		str("ProductVersion", version))
	sfi := block("StringFileInfo", nil, 0, 1, strings)
	trans := []byte{0x09, 0x04, 0xB0, 0x04}
	vfi := block("VarFileInfo", nil, 0, 1, block("Translation", trans, 4, 0))
	return block("VS_VERSION_INFO", fixed.Bytes(), fixed.Len(), 0, sfi, vfi)
}

// ---------------------------------------------------------------- COFF .syso

type res struct {
	typ, id uint32
	data    []byte
}

func main() {
	ico, err := os.ReadFile("tools/mkres/icon.ico")
	if err != nil {
		log.Fatal(err)
	}
	icons := readICO(ico)
	if err := os.WriteFile("web/favicon.ico", ico, 0o644); err != nil {
		log.Fatal(err)
	}
	for _, ic := range icons {
		if ic.w == 256 {
			if err := os.WriteFile("web/icon.png", ic.data, 0o644); err != nil {
				log.Fatal(err)
			}
		}
	}
	var grp bytes.Buffer
	gw := func(v any) { _ = binary.Write(&grp, binary.LittleEndian, v) }
	gw(uint16(0))
	gw(uint16(1))
	gw(uint16(len(icons)))
	var rs []res
	for i, ic := range icons {
		dim := func(x int) uint8 {
			if x >= 256 {
				return 0
			}
			return uint8(x)
		}
		gw(dim(ic.w))
		gw(dim(ic.h))
		gw(uint8(0))
		gw(uint8(0))
		gw(uint16(1))
		gw(uint16(32))
		gw(uint32(len(ic.data)))
		gw(uint16(i + 1))
		rs = append(rs, res{3, uint32(i + 1), ic.data})
	}
	rs = append(rs, res{14, 1, grp.Bytes()}, res{16, 1, versionInfo()})

	// group by type: RT_ICON (3), RT_GROUP_ICON (14), RT_VERSION (16)
	types := []uint32{3, 14, 16}
	byType := map[uint32][]int{}
	for i, r := range rs {
		byType[r.typ] = append(byType[r.typ], i)
	}
	dirSize := func(n int) int { return 16 + 8*n }
	off := dirSize(len(types))
	typeOff := map[uint32]int{}
	for _, t := range types {
		typeOff[t] = off
		off += dirSize(len(byType[t]))
	}
	langOff := make([]int, len(rs))
	for i := range rs {
		langOff[i] = off
		off += dirSize(1)
	}
	dataEntry := make([]int, len(rs))
	for i := range rs {
		dataEntry[i] = off
		off += 16
	}
	blobOff := make([]int, len(rs))
	for i, r := range rs {
		off = (off + 7) &^ 7
		blobOff[i] = off
		off += len(r.data)
	}
	sec := make([]byte, (off+7)&^7)
	put16 := func(o int, v uint16) { binary.LittleEndian.PutUint16(sec[o:], v) }
	put32 := func(o int, v uint32) { binary.LittleEndian.PutUint32(sec[o:], v) }
	entry := func(dir, idx int, id uint32, target int, sub bool) {
		e := dir + 16 + 8*idx
		put32(e, id)
		v := uint32(target)
		if sub {
			v |= 0x80000000
		}
		put32(e+4, v)
	}
	put16(14, uint16(len(types)))
	for i, t := range types {
		entry(0, i, t, typeOff[t], true)
		put16(typeOff[t]+14, uint16(len(byType[t])))
		for j, ri := range byType[t] {
			entry(typeOff[t], j, rs[ri].id, langOff[ri], true)
		}
	}
	var relocs []int
	for i, r := range rs {
		put16(langOff[i]+14, 1)
		entry(langOff[i], 0, 0x0409, dataEntry[i], false)
		put32(dataEntry[i], uint32(blobOff[i]))
		put32(dataEntry[i]+4, uint32(len(r.data)))
		relocs = append(relocs, dataEntry[i])
		copy(sec[blobOff[i]:], r.data)
	}

	var out bytes.Buffer
	o := func(v any) { _ = binary.Write(&out, binary.LittleEndian, v) }
	rawPtr := 20 + 40
	relPtr := rawPtr + len(sec)
	symPtr := relPtr + 10*len(relocs)
	o(uint16(0x8664))
	o(uint16(1))
	o(uint32(0))
	o(uint32(symPtr))
	o(uint32(1))
	o(uint16(0))
	o(uint16(0))
	out.Write([]byte(".rsrc\x00\x00\x00"))
	o(uint32(0))
	o(uint32(0))
	o(uint32(len(sec)))
	o(uint32(rawPtr))
	o(uint32(relPtr))
	o(uint32(0))
	o(uint16(len(relocs)))
	o(uint16(0))
	o(uint32(0x40000040))
	out.Write(sec)
	for _, r := range relocs {
		o(uint32(r))
		o(uint32(0))
		o(uint16(3)) // IMAGE_REL_AMD64_ADDR32NB
	}
	out.Write([]byte(".rsrc\x00\x00\x00"))
	o(uint32(0))
	o(int16(1))
	o(uint16(0))
	o(uint8(3))
	o(uint8(0))
	o(uint32(4))
	if err := os.WriteFile("rsrc_windows_amd64.syso", out.Bytes(), 0o644); err != nil {
		log.Fatal(err)
	}
	log.Printf("wrote web/icon.png, web/favicon.ico and rsrc_windows_amd64.syso (%d bytes)", out.Len())
}
