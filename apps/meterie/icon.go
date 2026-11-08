package meterie

import "image"

// renderLogoImage renders the Meterie logo with real transparency by drawing it on
// black and on white and recovering alpha from the difference.
func renderLogoImage(sz int) *image.NRGBA {
	cb := NewCanvas(sz, sz)
	cw := NewCanvas(sz, sz)
	cb.Clear(RGB(0, 0, 0))
	cw.Clear(RGB(255, 255, 255))
	pad := float64(sz) * 0.02
	drawLogo(cb, pad, pad, float64(sz)-2*pad)
	drawLogo(cw, pad, pad, float64(sz)-2*pad)
	img := image.NewNRGBA(image.Rect(0, 0, sz, sz))
	for i := 0; i < sz*sz; i++ {
		alpha := 255 - (float64(cw.Pix[i*4+1]) - float64(cb.Pix[i*4+1]))
		alpha = clampf(alpha, 0, 255)
		img.Pix[i*4+3] = uint8(alpha)
		if alpha > 0 {
			img.Pix[i*4+0] = uint8(clampf(float64(cb.Pix[i*4+2])*255/alpha, 0, 255))
			img.Pix[i*4+1] = uint8(clampf(float64(cb.Pix[i*4+1])*255/alpha, 0, 255))
			img.Pix[i*4+2] = uint8(clampf(float64(cb.Pix[i*4+0])*255/alpha, 0, 255))
		}
	}
	return img
}
