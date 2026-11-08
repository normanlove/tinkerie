package meterie

// ocr.go: seven-segment display reader for Camera mode.

import (
	"image"
	"math"
	"strconv"
	"strings"
)

// OCRParams configures the reader.
type OCRParams struct {
	Crop   [4]float64 // x0,y0,x1,y1 normalized 0..1
	Digits int
	Skew   float64 // italic slant correction (fraction of cell width per cell height)
	Thresh float64 // 0 = automatic (Otsu), else 0..1
	Thick  float64 // segment thickness as fraction of cell width
	Invert bool    // true: dark segments on a light background (LCD)
	DP     bool    // detect decimal points
}

// OCRDebug holds per-cell sampling info for the overlay.
type OCRDebug struct {
	Cells  [][8]float64 // per cell, per segment (a..g, dp) on-fraction
	Masks  []byte
	Thresh uint8
}

// Sample points are relative to one digit pitch: the digit itself spans about
// 0..0.78 of the pitch and the decimal point sits in the gap after it.
var segSample = [8][4]float64{ // cx, cy, half-w, half-h (fractions of the pitch) — thickness fills the zeros
	{0.39, 0.06, 0.16, 0}, // a
	{0.71, 0.28, 0, 0.11}, // b
	{0.71, 0.72, 0, 0.11}, // c
	{0.39, 0.94, 0.16, 0}, // d
	{0.07, 0.72, 0, 0.11}, // e
	{0.07, 0.28, 0, 0.11}, // f
	{0.39, 0.50, 0.16, 0}, // g
	{0.88, 0.93, 0, 0},    // decimal point
}

const ocrCellW, ocrCellH = 40, 64

// normalize crops, de-skews and scales the display region to digits*40 x 64.
func ocrNormalize(img *image.Gray, p OCRParams) *image.Gray {
	b := img.Bounds()
	W, H := float64(b.Dx()), float64(b.Dy())
	x0, y0 := p.Crop[0]*W, p.Crop[1]*H
	x1, y1 := p.Crop[2]*W, p.Crop[3]*H
	if x1-x0 < 4 || y1-y0 < 4 {
		return nil
	}
	n := max(1, p.Digits)
	ow, oh := n*ocrCellW, ocrCellH
	out := image.NewGray(image.Rect(0, 0, ow, oh))
	sx := (x1 - x0) / float64(ow)
	sy := (y1 - y0) / float64(oh)
	for y := 0; y < oh; y++ {
		shift := p.Skew * float64(ocrCellW) * (float64(oh-y) / float64(oh)) // italic: top shifted right
		for x := 0; x < ow; x++ {
			fx := x0 + (float64(x)+shift)*sx
			fy := y0 + float64(y)*sy
			out.Pix[y*out.Stride+x] = bilinear(img, fx, fy)
		}
	}
	return out
}

func bilinear(img *image.Gray, x, y float64) uint8 {
	b := img.Bounds()
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	xi, yi := int(x), int(y)
	if xi >= b.Dx()-1 {
		xi = b.Dx() - 2
	}
	if yi >= b.Dy()-1 {
		yi = b.Dy() - 2
	}
	if xi < 0 || yi < 0 {
		return 0
	}
	fx, fy := x-float64(xi), y-float64(yi)
	p := func(px, py int) float64 { return float64(img.Pix[(py)*img.Stride+px]) }
	v := p(xi, yi)*(1-fx)*(1-fy) + p(xi+1, yi)*fx*(1-fy) + p(xi, yi+1)*(1-fx)*fy + p(xi+1, yi+1)*fx*fy
	return uint8(v)
}

func otsu(g *image.Gray) uint8 {
	var hist [256]int
	for _, v := range g.Pix {
		hist[v]++
	}
	total := len(g.Pix)
	sum := 0.0
	for i := 0; i < 256; i++ {
		sum += float64(i * hist[i])
	}
	var sumB, wB float64
	best, bestT := 0.0, 128
	for t := 0; t < 256; t++ {
		wB += float64(hist[t])
		if wB == 0 {
			continue
		}
		wF := float64(total) - wB
		if wF == 0 {
			break
		}
		sumB += float64(t * hist[t])
		mB, mF := sumB/wB, (sum-sumB)/wF
		v := wB * wF * (mB - mF) * (mB - mF)
		if v > best {
			best, bestT = v, t
		}
	}
	return uint8(bestT)
}

// OCR reads the display. Returns the text (e.g. "-12.34"), whether all cells decoded, and debug info.
func OCR(img *image.Gray, p OCRParams) (string, bool, OCRDebug) {
	var dbg OCRDebug
	if img == nil {
		return "", false, dbg
	}
	g := ocrNormalize(img, p)
	if g == nil {
		return "", false, dbg
	}
	th := uint8(clampf(p.Thresh*255, 1, 254))
	if p.Thresh <= 0 {
		th = otsu(g)
	}
	dbg.Thresh = th
	on := func(v uint8) bool {
		if p.Invert {
			return v < th
		}
		return v > th
	}
	thick := clampf(p.Thick, 0.06, 0.4)
	frac := func(cx, cy, hw, hh float64) float64 {
		x0, x1 := int((cx-hw)*ocrCellW), int(math.Ceil((cx+hw)*ocrCellW))
		y0, y1 := int((cy-hh)*ocrCellH), int(math.Ceil((cy+hh)*ocrCellH))
		n, k := 0, 0
		for y := y0; y <= y1; y++ {
			for x := x0; x <= x1; x++ {
				if x < 0 || y < 0 || x >= g.Rect.Dx() || y >= g.Rect.Dy() {
					continue
				}
				n++
				if on(g.Pix[y*g.Stride+x]) {
					k++
				}
			}
		}
		if n == 0 {
			return 0
		}
		return float64(k) / float64(n)
	}
	var sb strings.Builder
	ok := true
	seenDigit := false
	for i := 0; i < p.Digits; i++ {
		var cell [8]float64
		var mask byte
		for s := 0; s < 8; s++ {
			ss := segSample[s]
			hw, hh := ss[2], ss[3]
			if hw == 0 {
				hw = thick * 0.45
			}
			if hh == 0 {
				hh = thick * 0.45 * ocrCellW / ocrCellH
			}
			if s == 7 {
				hw, hh = thick*0.4, thick*0.4*ocrCellW/ocrCellH
			}
			f := frac(float64(i)+ss[0], ss[1], hw, hh)
			cell[s] = f
			if s < 7 && f > 0.45 {
				mask |= 1 << uint(s)
			}
		}
		dbg.Cells = append(dbg.Cells, cell)
		dbg.Masks = append(dbg.Masks, mask)
		ch, good := segToChar(mask)
		if !good {
			ch = '?'
			ok = false
		}
		if ch == ' ' && seenDigit {
			ok = false
		}
		if ch != ' ' {
			seenDigit = true
		}
		if ch != ' ' {
			sb.WriteByte(ch)
		}
		if p.DP && i < p.Digits-1 && cell[7] > 0.5 && seenDigit {
			sb.WriteByte('.')
		}
	}
	text := sb.String()
	if text == "" {
		ok = false
	}
	return text, ok, dbg
}

// OCRValue parses OCR text into a number (handles "OL", "-", leading/trailing junk).
func OCRValue(text string) (float64, bool, bool) {
	t := strings.TrimSpace(text)
	if strings.Contains(t, "L") && (strings.Contains(t, "0") || strings.Contains(t, "O")) {
		return math.NaN(), true, true // overload
	}
	v, err := strconv.ParseFloat(t, 64)
	if err != nil {
		return 0, false, false
	}
	return v, true, false
}

// OCRCalibrate searches skew/threshold/thickness so the reader returns want.
func OCRCalibrate(img *image.Gray, p OCRParams, want string) (OCRParams, bool) {
	want = strings.ReplaceAll(strings.TrimSpace(want), " ", "")
	if !p.DP {
		want = strings.ReplaceAll(want, ".", "")
	}
	best := p
	bestScore := -1
	for _, inv := range []bool{p.Invert, !p.Invert} {
		for sk := -0.3; sk <= 0.301; sk += 0.05 {
			for _, tk := range []float64{0.14, 0.18, 0.22, 0.26, 0.3} {
				for _, tr := range []float64{0, 0.3, 0.4, 0.5, 0.6, 0.7} {
					q := p
					q.Invert, q.Skew, q.Thick, q.Thresh = inv, sk, tk, tr
					txt, _, _ := OCR(img, q)
					score := matchScore(txt, want)
					if score > bestScore {
						best, bestScore = q, score
					}
					if txt == want {
						return q, true
					}
				}
			}
		}
	}
	return best, false
}

func matchScore(a, b string) int {
	s := 0
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[len(a)-1-i] == b[len(b)-1-i] {
			s++
		}
	}
	if len(a) == len(b) {
		s++
	}
	return s
}
