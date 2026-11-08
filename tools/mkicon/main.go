// mkicon draws the Tinkerie app icon and writes:
//   - build/tinkerie.ico          (Windows icon, PNG images inside)
//   - launcher/web/tinkerie.png   (256 px, used by the launcher window)
//   - build/tinkerie.icns         (macOS app icon)
//   - build/tinkerie.png          (512 px, Linux desktop icon)
//
// Run from the repository root:  go run ./tools/mkicon
// Standard library only. Created by nrmluvdev.
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

// shape coverage at (x, y) in 0..1 icon coordinates: 0 outside, 1 inside.
type shape func(x, y float64) bool

func roundRect(x0, y0, x1, y1, r float64) shape {
	return func(x, y float64) bool {
		if x < x0 || x > x1 || y < y0 || y > y1 {
			return false
		}
		cx := math.Max(x0+r, math.Min(x, x1-r))
		cy := math.Max(y0+r, math.Min(y, y1-r))
		return (x-cx)*(x-cx)+(y-cy)*(y-cy) <= r*r
	}
}

func circle(cx, cy, r float64) shape {
	return func(x, y float64) bool { return (x-cx)*(x-cx)+(y-cy)*(y-cy) <= r*r }
}

func union(s ...shape) shape {
	return func(x, y float64) bool {
		for _, f := range s {
			if f(x, y) {
				return true
			}
		}
		return false
	}
}

func lerp(a, b uint8, t float64) float64 { return float64(a) + (float64(b)-float64(a))*t }

func draw(size int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	bg := roundRect(0.04, 0.04, 0.96, 0.96, 0.22)
	// A bold "T" whose stem ends in a spark: the tinkerer's mark.
	glyph := union(
		roundRect(0.24, 0.26, 0.76, 0.40, 0.07), // bar
		roundRect(0.43, 0.26, 0.57, 0.66, 0.07), // stem
		circle(0.50, 0.76, 0.075),               // spark
	)
	from := color.NRGBA{0x22, 0xd3, 0xee, 255} // teal
	to := color.NRGBA{0x8b, 0x5c, 0xf6, 255}   // violet
	const ss = 4                               // 4x4 supersampling for smooth edges
	for py := 0; py < size; py++ {
		for px := 0; px < size; px++ {
			var r, g, b, a float64
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					x := (float64(px) + (float64(sx)+0.5)/ss) / float64(size)
					y := (float64(py) + (float64(sy)+0.5)/ss) / float64(size)
					if !bg(x, y) {
						continue
					}
					if glyph(x, y) {
						r, g, b = r+255, g+255, b+255
					} else {
						t := (x + y) / 2
						r += lerp(from.R, to.R, t)
						g += lerp(from.G, to.G, t)
						b += lerp(from.B, to.B, t)
					}
					a++
				}
			}
			if a > 0 {
				img.SetNRGBA(px, py, color.NRGBA{uint8(r / a), uint8(g / a), uint8(b / a), uint8(255 * a / ss / ss)})
			}
		}
	}
	return img
}

func main() {
	sizes := []int{256, 128, 64, 48, 32, 24, 16}
	var pngs [][]byte
	for _, s := range sizes {
		var buf bytes.Buffer
		if err := png.Encode(&buf, draw(s)); err != nil {
			log.Fatal(err)
		}
		pngs = append(pngs, buf.Bytes())
	}
	var ico bytes.Buffer
	binary.Write(&ico, binary.LittleEndian, [3]uint16{0, 1, uint16(len(sizes))})
	offset := 6 + 16*len(sizes)
	for i, s := range sizes {
		b := uint8(s)
		if s >= 256 {
			b = 0
		}
		binary.Write(&ico, binary.LittleEndian, struct {
			W, H, Colors, Reserved uint8
			Planes, Bits           uint16
			Size, Offset           uint32
		}{b, b, 0, 0, 1, 32, uint32(len(pngs[i])), uint32(offset)})
		offset += len(pngs[i])
	}
	for _, p := range pngs {
		ico.Write(p)
	}
	must(os.MkdirAll("build", 0o755))
	must(os.WriteFile("build/tinkerie.ico", ico.Bytes(), 0o644))
	must(os.WriteFile("launcher/web/tinkerie.png", pngs[0], 0o644))

	// macOS .icns: PNG images tagged by size (ic07 128, ic08 256, ic09 512, ic10 1024).
	var icns, body bytes.Buffer
	for _, e := range []struct {
		tag  string
		size int
	}{{"ic07", 128}, {"ic08", 256}, {"ic09", 512}, {"ic10", 1024}} {
		var buf bytes.Buffer
		must(png.Encode(&buf, draw(e.size)))
		body.WriteString(e.tag)
		binary.Write(&body, binary.BigEndian, uint32(8+buf.Len()))
		body.Write(buf.Bytes())
		if e.size == 512 {
			must(os.WriteFile("build/tinkerie.png", buf.Bytes(), 0o644))
		}
	}
	icns.WriteString("icns")
	binary.Write(&icns, binary.BigEndian, uint32(8+body.Len()))
	icns.Write(body.Bytes())
	must(os.WriteFile("build/tinkerie.icns", icns.Bytes(), 0o644))
	log.Println("wrote build/tinkerie.ico, .icns, .png and launcher/web/tinkerie.png")
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
