// mkres draws Tunie's app icon and writes:
//   - web/icon.png (256 px, used as the window/favicon)
//   - rsrc_windows_amd64.syso (Windows icon resource linked into Tunie.exe)
//
// Run from the project root:  go run ./tools/mkres
// Part of the nrmluvdev maker suite. Created by nrmluvdev.
package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
)

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

// coverage of the icon shape at a point (0..1 coordinates)
func shade(x, y float64) (r, g, b, a float64) {
	// rounded square
	const rad = 0.22
	dx := math.Max(math.Abs(x-0.5)-(0.5-0.04-rad), 0)
	dy := math.Max(math.Abs(y-0.5)-(0.5-0.04-rad), 0)
	if math.Hypot(dx, dy) > rad {
		return 0, 0, 0, 0
	}
	t := clamp01((x + y) / 2)
	// violet -> cyan
	r = lerp(124, 34, t)
	g = lerp(92, 211, t)
	b = lerp(255, 238, t)
	// soft top highlight
	if y < 0.5 {
		h := (0.5 - y) * 0.25
		r, g, b = lerp(r, 255, h), lerp(g, 255, h), lerp(b, 255, h)
	}
	a = 1
	// damped wave glyph
	for _, p := range wave {
		if math.Hypot(x-p[0], y-p[1]) < 0.034 {
			return 255, 255, 255, 1
		}
	}
	// end dot
	if math.Hypot(x-0.84, y-0.54) < 0.05 {
		return 255, 255, 255, 1
	}
	return
}

var wave = func() [][2]float64 {
	var pts [][2]float64
	for i := 0; i <= 600; i++ {
		t := float64(i) / 600
		amp := 0.2 * math.Exp(-2.2*t)
		pts = append(pts, [2]float64{0.17 + 0.67*t, 0.54 - amp*math.Sin(t*math.Pi*4.2)})
	}
	return pts
}()

func clamp01(v float64) float64 { return math.Max(0, math.Min(1, v)) }

func render(size int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	const ss = 4
	for py := 0; py < size; py++ {
		for px := 0; px < size; px++ {
			var R, G, B, A float64
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					x := (float64(px) + (float64(sx)+0.5)/ss) / float64(size)
					y := (float64(py) + (float64(sy)+0.5)/ss) / float64(size)
					r, g, b, a := shade(x, y)
					R += r * a
					G += g * a
					B += b * a
					A += a
				}
			}
			n := float64(ss * ss)
			if A > 0 {
				img.SetNRGBA(px, py, color.NRGBA{uint8(R / A), uint8(G / A), uint8(B / A), uint8(255 * A / n)})
			}
		}
	}
	return img
}

func pngBytes(img image.Image) []byte {
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		log.Fatal(err)
	}
	return b.Bytes()
}

// ---------------------------------------------------------------- COFF .syso

func main() {
	sizes := []int{256, 64, 48, 32, 16}
	var icons [][]byte
	for _, s := range sizes {
		icons = append(icons, pngBytes(render(s)))
	}
	if err := os.WriteFile("web/icon.png", icons[0], 0o644); err != nil {
		log.Fatal(err)
	}

	// group icon directory
	var grp bytes.Buffer
	w := func(v any) { _ = binary.Write(&grp, binary.LittleEndian, v) }
	w(uint16(0))
	w(uint16(1))
	w(uint16(len(sizes)))
	for i, s := range sizes {
		dim := uint8(s)
		if s >= 256 {
			dim = 0
		}
		w(dim)
		w(dim)
		w(uint8(0))
		w(uint8(0))
		w(uint16(1))
		w(uint16(32))
		w(uint32(len(icons[i])))
		w(uint16(i + 1))
	}

	// Resource tree layout:
	// root dir (2 id entries: RT_ICON=3, RT_GROUP_ICON=14)
	//   type dir RT_ICON: n entries (ids 1..n) -> lang dirs
	//   type dir RT_GROUP_ICON: 1 entry (id 1) -> lang dir
	//   lang dirs: 1 entry each (0x0409) -> data entry
	//   data entries (16 bytes each)
	//   data blobs
	n := len(sizes)
	type blob struct{ data []byte }
	var blobs []blob
	for _, ic := range icons {
		blobs = append(blobs, blob{ic})
	}
	blobs = append(blobs, blob{grp.Bytes()})

	dirSize := func(entries int) int { return 16 + 8*entries }
	off := 0
	rootOff := off
	off += dirSize(2)
	iconTypeOff := off
	off += dirSize(n)
	grpTypeOff := off
	off += dirSize(1)
	langOffs := make([]int, n+1)
	for i := range langOffs {
		langOffs[i] = off
		off += dirSize(1)
	}
	dataEntryOffs := make([]int, n+1)
	for i := range dataEntryOffs {
		dataEntryOffs[i] = off
		off += 16
	}
	blobOffs := make([]int, n+1)
	for i := range blobs {
		off = (off + 7) &^ 7
		blobOffs[i] = off
		off += len(blobs[i].data)
	}
	total := (off + 7) &^ 7
	sec := make([]byte, total)
	put16 := func(o int, v uint16) { binary.LittleEndian.PutUint16(sec[o:], v) }
	put32 := func(o int, v uint32) { binary.LittleEndian.PutUint32(sec[o:], v) }
	dir := func(o, ids int) { put16(o+14, uint16(ids)) }
	entry := func(dirOff, idx int, id uint32, target int, sub bool) {
		e := dirOff + 16 + 8*idx
		put32(e, id)
		v := uint32(target)
		if sub {
			v |= 0x80000000
		}
		put32(e+4, v)
	}
	dir(rootOff, 2)
	entry(rootOff, 0, 3, iconTypeOff, true)
	entry(rootOff, 1, 14, grpTypeOff, true)
	dir(iconTypeOff, n)
	for i := 0; i < n; i++ {
		entry(iconTypeOff, i, uint32(i+1), langOffs[i], true)
	}
	dir(grpTypeOff, 1)
	entry(grpTypeOff, 0, 1, langOffs[n], true)
	var relocs []int
	for i := 0; i <= n; i++ {
		dir(langOffs[i], 1)
		entry(langOffs[i], 0, 0x0409, dataEntryOffs[i], false)
		de := dataEntryOffs[i]
		put32(de, uint32(blobOffs[i])) // RVA, fixed up by relocation
		put32(de+4, uint32(len(blobs[i].data)))
		relocs = append(relocs, de)
		copy(sec[blobOffs[i]:], blobs[i].data)
	}

	var out bytes.Buffer
	o := func(v any) { _ = binary.Write(&out, binary.LittleEndian, v) }
	const fileHdr, secHdr = 20, 40
	rawPtr := fileHdr + secHdr
	relPtr := rawPtr + len(sec)
	symPtr := relPtr + 10*len(relocs)
	// file header
	o(uint16(0x8664))
	o(uint16(1))
	o(uint32(0))
	o(uint32(symPtr))
	o(uint32(1))
	o(uint16(0))
	o(uint16(0))
	// section header
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
	// symbol table: .rsrc section symbol
	out.Write([]byte(".rsrc\x00\x00\x00"))
	o(uint32(0))
	o(int16(1))
	o(uint16(0))
	o(uint8(3))
	o(uint8(0))
	o(uint32(4)) // empty string table
	if err := os.WriteFile("rsrc_windows_amd64.syso", out.Bytes(), 0o644); err != nil {
		log.Fatal(err)
	}
	log.Printf("wrote web/icon.png and rsrc_windows_amd64.syso (%d bytes)", out.Len())
}
