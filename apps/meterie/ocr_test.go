package meterie

import (
	"image"
	"testing"
)

// renderSeg draws a 7-seg number like a real LCD (dark on light) and returns a grey image.
func renderSeg(text string, digits int, slant float64) *image.Gray {
	c := NewCanvas(digits*70+80, 140)
	c.Clear(RGB(200, 210, 190))
	cells, dps, neg := lcdCells(text, digits)
	_ = neg
	h, w, t := 100.0, 52.0, 11.5
	for i := 0; i < digits; i++ {
		x := 40 + float64(i)*70
		polys := segPolys(x, 20, w, h, t, slant)
		m := charSegs[cells[i]]
		for s := 0; s < 7; s++ {
			if m&(1<<uint(s)) != 0 {
				c.FillConvex(polys[s], RGB(30, 35, 30), 0, 0)
			}
		}
		if dps[i] {
			c.FillCircle(x+w+9, 20+h-6, 6, RGB(30, 35, 30))
		}
	}
	img := c.ToImage(Rect{0, 0, c.W, c.H})
	g := image.NewGray(img.Bounds())
	for i := 0; i < len(g.Pix); i++ {
		g.Pix[i] = img.Pix[i*4+1]
	}
	return g
}

func TestOCR(t *testing.T) {
	for _, tc := range []struct {
		text  string
		slant float64
	}{{"12.34", 0}, {"0.987", 0}, {"5678", 0}, {"20.05", 8}} {
		g := renderSeg(tc.text, 4, tc.slant)
		W, H := float64(g.Rect.Dx()), float64(g.Rect.Dy())
		p := OCRParams{Crop: [4]float64{40 / W, 20 / H, (40 + 4*70) / W, 120 / H}, Digits: 4, Thick: 0.22, Invert: true, DP: true}
		p.Skew = tc.slant / 69
		txt, ok, _ := OCR(g, p)
		if !ok || txt != tc.text {
			q, found := OCRCalibrate(g, p, tc.text)
			txt2, _, _ := OCR(g, q)
			t.Logf("%q: direct=%q calibrated=%q found=%v skew=%.2f thick=%.2f", tc.text, txt, txt2, found, q.Skew, q.Thick)
			if !found {
				t.Errorf("OCR failed for %q", tc.text)
			}
		}
	}
}
