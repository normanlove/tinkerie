//go:build ignore

// mkrsrc builds rsrc_windows_amd64.syso: the Windows icon + version info
// (Company / Copyright = nrmluvdev) that the Go linker embeds into Wirie.exe.
// Usage (from the repo root):  go run tools/mkrsrc.go
// Needs only the Go standard library. Icons come from assets/icon*.png.
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"unicode/utf16"
)

const (
	appName = "Wirie"
	version = "1.2.0.0"
	author  = "nrmluvdev"
)

type res struct {
	typ, id uint16
	data    []byte
}

func le(v any) []byte { var b bytes.Buffer; binary.Write(&b, binary.LittleEndian, v); return b.Bytes() }
func pad4(b []byte) []byte {
	for len(b)%4 != 0 {
		b = append(b, 0)
	}
	return b
}
func wstr(s string) []byte {
	u := utf16.Encode([]rune(s))
	u = append(u, 0)
	return le(u)
}

// version-info node
func node(key string, value []byte, valueWords int, typ uint16, children ...[]byte) []byte {
	b := make([]byte, 6)
	b = append(b, wstr(key)...)
	b = pad4(b)
	b = append(b, value...)
	for _, c := range children {
		b = pad4(b)
		b = append(b, c...)
	}
	binary.LittleEndian.PutUint16(b[0:], uint16(len(b)))
	binary.LittleEndian.PutUint16(b[2:], uint16(valueWords))
	binary.LittleEndian.PutUint16(b[4:], typ)
	return b
}
func str(k, v string) []byte { w := wstr(v); return node(k, w, len(w)/2, 1) }

func versionInfo() []byte {
	ms, ls := uint32(1<<16|2), uint32(0<<16|0)
	fixed := le([]uint32{0xFEEF04BD, 0x00010000, ms, ls, ms, ls, 0x3F, 0, 0x40004, 1, 0, 0, 0})
	table := node("040904B0", nil, 0, 1,
		str("CompanyName", author),
		str("FileDescription", appName+" — beginner-friendly circuit board designer by "+author),
		str("FileVersion", version),
		str("InternalName", appName),
		str("LegalCopyright", "© 2026 "+author),
		str("OriginalFilename", appName+".exe"),
		str("ProductName", appName),
		str("ProductVersion", version),
		str("Comments", "Made by "+author+". Part of the "+author+" maker tool suite."),
	)
	sfi := node("StringFileInfo", nil, 0, 1, table)
	vfi := node("VarFileInfo", nil, 0, 1, node("Translation", le([]uint16{0x0409, 0x04B0}), 4, 0))
	return node("VS_VERSION_INFO", fixed, len(fixed), 0, sfi, vfi)
}

func main() {
	sizes := []int{16, 24, 32, 48, 64, 128, 256}
	var resources []res
	var grp bytes.Buffer
	binary.Write(&grp, binary.LittleEndian, []uint16{0, 1, uint16(len(sizes))})
	for i, s := range sizes {
		png, err := os.ReadFile(fmt.Sprintf("assets/icon%d.png", s))
		if err != nil {
			panic(err)
		}
		resources = append(resources, res{3, uint16(i + 1), png})
		wh := byte(s)
		if s >= 256 {
			wh = 0
		}
		grp.Write([]byte{wh, wh, 0, 0})
		binary.Write(&grp, binary.LittleEndian, uint16(1))
		binary.Write(&grp, binary.LittleEndian, uint16(32))
		binary.Write(&grp, binary.LittleEndian, uint32(len(png)))
		binary.Write(&grp, binary.LittleEndian, uint16(i+1))
	}
	resources = append(resources, res{14, 1, grp.Bytes()}, res{16, 1, versionInfo()})

	// group by type (types must be sorted ascending)
	types := []uint16{3, 14, 16}
	byType := map[uint16][]res{}
	for _, r := range resources {
		byType[r.typ] = append(byType[r.typ], r)
	}
	dirSize := func(n int) int { return 16 + 8*n }
	// layout: root dir, type dirs, lang dirs, data entries, data
	off := dirSize(len(types))
	typeOff := map[uint16]int{}
	for _, t := range types {
		typeOff[t] = off
		off += dirSize(len(byType[t]))
	}
	langOff := map[[2]uint16]int{}
	for _, t := range types {
		for _, r := range byType[t] {
			langOff[[2]uint16{t, r.id}] = off
			off += dirSize(1)
		}
	}
	entryOff := map[[2]uint16]int{}
	for _, t := range types {
		for _, r := range byType[t] {
			entryOff[[2]uint16{t, r.id}] = off
			off += 16
		}
	}
	dataOff := map[[2]uint16]int{}
	for _, t := range types {
		for _, r := range byType[t] {
			for off%8 != 0 {
				off++
			}
			dataOff[[2]uint16{t, r.id}] = off
			off += len(r.data)
		}
	}
	sec := make([]byte, off)
	dirHdr := func(at, n int) { binary.LittleEndian.PutUint16(sec[at+14:], uint16(n)) }
	dirEnt := func(at int, id uint32, target uint32) {
		binary.LittleEndian.PutUint32(sec[at:], id)
		binary.LittleEndian.PutUint32(sec[at+4:], target)
	}
	dirHdr(0, len(types))
	for i, t := range types {
		dirEnt(16+8*i, uint32(t), 0x80000000|uint32(typeOff[t]))
		dirHdr(typeOff[t], len(byType[t]))
		for j, r := range byType[t] {
			k := [2]uint16{t, r.id}
			dirEnt(typeOff[t]+16+8*j, uint32(r.id), 0x80000000|uint32(langOff[k]))
			dirHdr(langOff[k], 1)
			dirEnt(langOff[k]+16, 0x0409, uint32(entryOff[k]))
			binary.LittleEndian.PutUint32(sec[entryOff[k]:], uint32(dataOff[k])) // RVA (relocated)
			binary.LittleEndian.PutUint32(sec[entryOff[k]+4:], uint32(len(r.data)))
			copy(sec[dataOff[k]:], r.data)
		}
	}
	// relocations: one ADDR32NB per data entry
	var relocs bytes.Buffer
	n := 0
	for _, t := range types {
		for _, r := range byType[t] {
			binary.Write(&relocs, binary.LittleEndian, uint32(entryOff[[2]uint16{t, r.id}]))
			binary.Write(&relocs, binary.LittleEndian, uint32(0)) // symbol 0 = .rsrc
			binary.Write(&relocs, binary.LittleEndian, uint16(3)) // IMAGE_REL_AMD64_ADDR32NB
			n++
		}
	}
	const fileHdr, secHdr = 20, 40
	rawPtr := fileHdr + secHdr
	relPtr := rawPtr + len(sec)
	symPtr := relPtr + relocs.Len()
	var out bytes.Buffer
	binary.Write(&out, binary.LittleEndian, uint16(0x8664))
	binary.Write(&out, binary.LittleEndian, uint16(1))
	binary.Write(&out, binary.LittleEndian, uint32(0))
	binary.Write(&out, binary.LittleEndian, uint32(symPtr))
	binary.Write(&out, binary.LittleEndian, uint32(1))
	binary.Write(&out, binary.LittleEndian, uint16(0))
	binary.Write(&out, binary.LittleEndian, uint16(0))
	out.Write([]byte(".rsrc\x00\x00\x00"))
	binary.Write(&out, binary.LittleEndian, []uint32{0, 0, uint32(len(sec)), uint32(rawPtr), uint32(relPtr), 0})
	binary.Write(&out, binary.LittleEndian, uint16(n))
	binary.Write(&out, binary.LittleEndian, uint16(0))
	binary.Write(&out, binary.LittleEndian, uint32(0x40000040))
	out.Write(sec)
	out.Write(relocs.Bytes())
	out.Write([]byte(".rsrc\x00\x00\x00"))
	binary.Write(&out, binary.LittleEndian, uint32(0))
	binary.Write(&out, binary.LittleEndian, int16(1))
	binary.Write(&out, binary.LittleEndian, uint16(0))
	out.Write([]byte{3, 0})
	binary.Write(&out, binary.LittleEndian, uint32(4))
	if err := os.WriteFile("rsrc_windows_amd64.syso", out.Bytes(), 0o644); err != nil {
		panic(err)
	}
	fmt.Printf("wrote rsrc_windows_amd64.syso (%d resources, %d bytes)\n", len(resources), out.Len())
}
