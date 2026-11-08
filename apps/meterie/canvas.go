package meterie

// canvas.go: a small anti-aliased software rasterizer drawing into a BGRA
// buffer. On Windows the buffer is a GDI DIB section so text can be drawn
// into the very same pixels with ClearType.

import (
	"image"
	"image/color"
	"math"
)

// Color is non-premultiplied RGBA.
type Color struct{ R, G, B, A uint8 }

func RGB(r, g, b uint8) Color         { return Color{r, g, b, 255} }
func RGBA(r, g, b, a uint8) Color     { return Color{r, g, b, a} }
func (c Color) WithA(a float64) Color { return Color{c.R, c.G, c.B, uint8(clamp01(a) * float64(c.A))} }

func hexColor(v uint32) Color { return RGB(uint8(v>>16), uint8(v>>8), uint8(v)) }

func lerpColor(a, b Color, t float64) Color {
	t = clamp01(t)
	return Color{
		uint8(float64(a.R) + (float64(b.R)-float64(a.R))*t),
		uint8(float64(a.G) + (float64(b.G)-float64(a.G))*t),
		uint8(float64(a.B) + (float64(b.B)-float64(a.B))*t),
		uint8(float64(a.A) + (float64(b.A)-float64(a.A))*t),
	}
}

func (c Color) Lighten(t float64) Color { return lerpColor(c, Color{255, 255, 255, c.A}, t) }
func (c Color) Darken(t float64) Color  { return lerpColor(c, Color{0, 0, 0, c.A}, t) }

func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

func clampf(x, lo, hi float64) float64 {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

func clampi(x, lo, hi int) int {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

// Rect is an integer rectangle (x, y, w, h).
type Rect struct{ X, Y, W, H int }

func R(x, y, w, h int) Rect { return Rect{x, y, w, h} }

func (r Rect) Contains(x, y int) bool { return x >= r.X && y >= r.Y && x < r.X+r.W && y < r.Y+r.H }
func (r Rect) Inset(d int) Rect       { return Rect{r.X + d, r.Y + d, r.W - 2*d, r.H - 2*d} }
func (r Rect) InsetXY(dx, dy int) Rect {
	return Rect{r.X + dx, r.Y + dy, r.W - 2*dx, r.H - 2*dy}
}
func (r Rect) Right() int  { return r.X + r.W }
func (r Rect) Bottom() int { return r.Y + r.H }
func (r Rect) CX() int     { return r.X + r.W/2 }
func (r Rect) CY() int     { return r.Y + r.H/2 }
func (r Rect) Intersect(o Rect) Rect {
	x0, y0 := max(r.X, o.X), max(r.Y, o.Y)
	x1, y1 := min(r.Right(), o.Right()), min(r.Bottom(), o.Bottom())
	if x1 < x0 {
		x1 = x0
	}
	if y1 < y0 {
		y1 = y0
	}
	return Rect{x0, y0, x1 - x0, y1 - y0}
}
func (r Rect) Empty() bool { return r.W <= 0 || r.H <= 0 }

// SplitH cuts off a left part of width w.
func (r Rect) CutLeft(w int) (Rect, Rect) {
	if w > r.W {
		w = r.W
	}
	return Rect{r.X, r.Y, w, r.H}, Rect{r.X + w, r.Y, r.W - w, r.H}
}
func (r Rect) CutTop(h int) (Rect, Rect) {
	if h > r.H {
		h = r.H
	}
	return Rect{r.X, r.Y, r.W, h}, Rect{r.X, r.Y + h, r.W, r.H - h}
}
func (r Rect) CutRight(w int) (Rect, Rect) {
	if w > r.W {
		w = r.W
	}
	return Rect{r.Right() - w, r.Y, w, r.H}, Rect{r.X, r.Y, r.W - w, r.H}
}
func (r Rect) CutBottom(h int) (Rect, Rect) {
	if h > r.H {
		h = r.H
	}
	return Rect{r.X, r.Bottom() - h, r.W, h}, Rect{r.X, r.Y, r.W, r.H - h}
}

// Canvas is a BGRA pixel buffer with a clip rectangle.
type Canvas struct {
	W, H   int
	Stride int
	Pix    []byte // B,G,R,A
	clip   Rect
	clips  []Rect
}

func NewCanvas(w, h int) *Canvas {
	c := &Canvas{W: w, H: h, Stride: w * 4, Pix: make([]byte, w*h*4)}
	c.clip = Rect{0, 0, w, h}
	return c
}

// WrapCanvas uses existing memory (DIB section bits).
func WrapCanvas(pix []byte, w, h int) *Canvas {
	c := &Canvas{W: w, H: h, Stride: w * 4, Pix: pix}
	c.clip = Rect{0, 0, w, h}
	return c
}

func (c *Canvas) Clip() Rect { return c.clip }
func (c *Canvas) PushClip(r Rect) {
	c.clips = append(c.clips, c.clip)
	c.clip = c.clip.Intersect(r)
}
func (c *Canvas) PopClip() {
	if n := len(c.clips); n > 0 {
		c.clip = c.clips[n-1]
		c.clips = c.clips[:n-1]
	}
}
func (c *Canvas) ResetClip() { c.clip = Rect{0, 0, c.W, c.H}; c.clips = c.clips[:0] }

// blend one pixel with coverage a (0..1) of colour col (integer maths for speed).
func (c *Canvas) blend(x, y int, col Color, a float64) {
	if a <= 0 {
		return
	}
	i := y*c.Stride + x*4
	al := uint32(a*float64(col.A) + 0.5)
	if al >= 255 {
		c.Pix[i], c.Pix[i+1], c.Pix[i+2], c.Pix[i+3] = col.B, col.G, col.R, 255
		return
	}
	if al == 0 {
		return
	}
	inv := 255 - al
	p := c.Pix[i : i+4 : i+4]
	p[0] = uint8((uint32(col.B)*al + uint32(p[0])*inv + 127) / 255)
	p[1] = uint8((uint32(col.G)*al + uint32(p[1])*inv + 127) / 255)
	p[2] = uint8((uint32(col.R)*al + uint32(p[2])*inv + 127) / 255)
	p[3] = 255
}

// CopyFrom copies an identically sized canvas (used for the cached background).
func (c *Canvas) CopyFrom(o *Canvas) {
	if o != nil && o.W == c.W && o.H == c.H {
		copy(c.Pix, o.Pix)
	}
}

func (c *Canvas) Clear(col Color) {
	for y := 0; y < c.H; y++ {
		row := c.Pix[y*c.Stride : y*c.Stride+c.W*4]
		for x := 0; x < len(row); x += 4 {
			row[x], row[x+1], row[x+2], row[x+3] = col.B, col.G, col.R, 255
		}
	}
}

// FillRect fills an axis aligned integer rectangle.
func (c *Canvas) FillRect(r Rect, col Color) {
	r = r.Intersect(c.clip)
	if r.Empty() || col.A == 0 {
		return
	}
	if col.A == 255 {
		for y := r.Y; y < r.Bottom(); y++ {
			i := y*c.Stride + r.X*4
			for x := 0; x < r.W; x++ {
				c.Pix[i], c.Pix[i+1], c.Pix[i+2], c.Pix[i+3] = col.B, col.G, col.R, 255
				i += 4
			}
		}
		return
	}
	for y := r.Y; y < r.Bottom(); y++ {
		for x := r.X; x < r.Right(); x++ {
			c.blend(x, y, col, 1)
		}
	}
}

// sdRoundBox: signed distance from point (px,py) to a rounded box centred at (cx,cy).
func sdRoundBox(px, py, cx, cy, hw, hh, r float64) float64 {
	qx := math.Abs(px-cx) - hw + r
	qy := math.Abs(py-cy) - hh + r
	ox, oy := math.Max(qx, 0), math.Max(qy, 0)
	return math.Hypot(ox, oy) + math.Min(math.Max(qx, qy), 0) - r
}

// FillRoundRectF draws an anti-aliased rounded rectangle (float coordinates).
// Pixels in the straight middle band of each row share one coverage value, so
// the distance field is only evaluated near the corners.
func (c *Canvas) FillRoundRectF(x, y, w, h, rad float64, col Color) {
	if w <= 0 || h <= 0 || col.A == 0 {
		return
	}
	rad = math.Min(rad, math.Min(w, h)/2)
	cx, cy, hw, hh := x+w/2, y+h/2, w/2, h/2
	x0 := max(int(math.Floor(x)), c.clip.X)
	y0 := max(int(math.Floor(y)), c.clip.Y)
	x1 := min(int(math.Ceil(x+w)), c.clip.Right())
	y1 := min(int(math.Ceil(y+h)), c.clip.Bottom())
	bandL := int(math.Ceil(x + rad + 1))
	bandR := int(math.Floor(x + w - rad - 1))
	for py := y0; py < y1; py++ {
		fy := float64(py) + 0.5
		midCov := clamp01(0.5 - sdRoundBox(cx, fy, cx, cy, hw, hh, rad))
		edgeL := max(bandL, x0)
		edgeR := min(bandR, x1)
		// left corner/edge pixels
		for px := x0; px < min(edgeL, x1); px++ {
			d := sdRoundBox(float64(px)+0.5, fy, cx, cy, hw, hh, rad)
			if d < 0.5 {
				c.blend(px, py, col, clamp01(0.5-d))
			}
		}
		// middle band: constant coverage
		if midCov > 0 && edgeR > edgeL {
			if midCov >= 0.999 && col.A == 255 {
				i := py*c.Stride + edgeL*4
				for px := edgeL; px < edgeR; px++ {
					c.Pix[i], c.Pix[i+1], c.Pix[i+2], c.Pix[i+3] = col.B, col.G, col.R, 255
					i += 4
				}
			} else {
				for px := edgeL; px < edgeR; px++ {
					c.blend(px, py, col, midCov)
				}
			}
		}
		// right corner/edge pixels
		for px := max(edgeR, edgeL, x0); px < x1; px++ {
			d := sdRoundBox(float64(px)+0.5, fy, cx, cy, hw, hh, rad)
			if d < 0.5 {
				c.blend(px, py, col, clamp01(0.5-d))
			}
		}
	}
}

func (c *Canvas) FillRoundRect(r Rect, rad float64, col Color) {
	c.FillRoundRectF(float64(r.X), float64(r.Y), float64(r.W), float64(r.H), rad, col)
}

// StrokeRoundRectF draws a rounded rectangle outline of width lw.
func (c *Canvas) StrokeRoundRectF(x, y, w, h, rad, lw float64, col Color) {
	if w <= 0 || h <= 0 {
		return
	}
	rad = math.Min(rad, math.Min(w, h)/2)
	ix, iy, iw, ih := x+lw/2, y+lw/2, w-lw, h-lw
	cx, cy, hw, hh := ix+iw/2, iy+ih/2, iw/2, ih/2
	ir := math.Max(rad-lw/2, 0)
	x0 := max(int(math.Floor(x)), c.clip.X)
	y0 := max(int(math.Floor(y)), c.clip.Y)
	x1 := min(int(math.Ceil(x+w)), c.clip.Right())
	y1 := min(int(math.Ceil(y+h)), c.clip.Bottom())
	bandL := int(math.Ceil(x + rad + lw + 1))
	bandR := int(math.Floor(x + w - rad - lw - 1))
	for py := y0; py < y1; py++ {
		fy := float64(py) + 0.5
		midCov := clamp01(0.5 - (math.Abs(sdRoundBox(cx, fy, cx, cy, hw, hh, ir)) - lw/2))
		edgeL := max(bandL, x0)
		edgeR := min(bandR, x1)
		for px := x0; px < min(edgeL, x1); px++ {
			d := math.Abs(sdRoundBox(float64(px)+0.5, fy, cx, cy, hw, hh, ir)) - lw/2
			if d < 0.5 {
				c.blend(px, py, col, clamp01(0.5-d))
			}
		}
		if midCov > 0 {
			for px := edgeL; px < edgeR; px++ {
				c.blend(px, py, col, midCov)
			}
		}
		for px := max(edgeR, edgeL, x0); px < x1; px++ {
			d := math.Abs(sdRoundBox(float64(px)+0.5, fy, cx, cy, hw, hh, ir)) - lw/2
			if d < 0.5 {
				c.blend(px, py, col, clamp01(0.5-d))
			}
		}
	}
}

func (c *Canvas) StrokeRoundRect(r Rect, rad, lw float64, col Color) {
	c.StrokeRoundRectF(float64(r.X), float64(r.Y), float64(r.W), float64(r.H), rad, lw, col)
}

// Shadow draws a soft drop shadow outside a rounded rect.
func (c *Canvas) Shadow(r Rect, rad, blur float64, dy float64, col Color) {
	x, y, w, h := float64(r.X), float64(r.Y)+dy, float64(r.W), float64(r.H)
	cx, cy, hw, hh := x+w/2, y+h/2, w/2, h/2
	rad = math.Min(rad, math.Min(w, h)/2)
	x0 := max(int(x-blur), c.clip.X)
	y0 := max(int(y-blur), c.clip.Y)
	x1 := min(int(x+w+blur)+1, c.clip.Right())
	y1 := min(int(y+h+blur)+1, c.clip.Bottom())
	for py := y0; py < y1; py++ {
		fy := float64(py) + 0.5
		inner := fy > y+rad && fy < y+h-rad
		for px := x0; px < x1; px++ {
			fx := float64(px) + 0.5
			if inner && fx > x+2 && fx < x+w-2 {
				if j := int(x+w-2) - 1; j > px {
					px = j
				}
				continue
			}
			d := sdRoundBox(fx, fy, cx, cy, hw, hh, rad)
			if d <= -1 {
				continue
			}
			t := clamp01(d / blur)
			a := (1 - t) * (1 - t)
			c.blend(px, py, col, a)
		}
	}
}

// FillCircle draws an anti-aliased disc.
func (c *Canvas) FillCircle(cx, cy, r float64, col Color) {
	x0 := max(int(cx-r-1), c.clip.X)
	y0 := max(int(cy-r-1), c.clip.Y)
	x1 := min(int(cx+r+2), c.clip.Right())
	y1 := min(int(cy+r+2), c.clip.Bottom())
	for py := y0; py < y1; py++ {
		for px := x0; px < x1; px++ {
			d := math.Hypot(float64(px)+0.5-cx, float64(py)+0.5-cy) - r
			if d < 0.5 {
				c.blend(px, py, col, clamp01(0.5-d))
			}
		}
	}
}

// StrokeCircle draws a ring.
func (c *Canvas) StrokeCircle(cx, cy, r, lw float64, col Color) {
	x0 := max(int(cx-r-lw-1), c.clip.X)
	y0 := max(int(cy-r-lw-1), c.clip.Y)
	x1 := min(int(cx+r+lw+2), c.clip.Right())
	y1 := min(int(cy+r+lw+2), c.clip.Bottom())
	for py := y0; py < y1; py++ {
		for px := x0; px < x1; px++ {
			d := math.Abs(math.Hypot(float64(px)+0.5-cx, float64(py)+0.5-cy)-r) - lw/2
			if d < 0.5 {
				c.blend(px, py, col, clamp01(0.5-d))
			}
		}
	}
}

// Arc draws a thick arc from angle a0 to a1 (radians, 0 = +x, clockwise on screen).
func (c *Canvas) Arc(cx, cy, r, lw, a0, a1 float64, col Color) {
	if a1 < a0 {
		a0, a1 = a1, a0
	}
	x0 := max(int(cx-r-lw-1), c.clip.X)
	y0 := max(int(cy-r-lw-1), c.clip.Y)
	x1 := min(int(cx+r+lw+2), c.clip.Right())
	y1 := min(int(cy+r+lw+2), c.clip.Bottom())
	mid := (a0 + a1) / 2
	half := (a1 - a0) / 2
	for py := y0; py < y1; py++ {
		for px := x0; px < x1; px++ {
			fx, fy := float64(px)+0.5-cx, float64(py)+0.5-cy
			ang := math.Atan2(fy, fx)
			da := math.Remainder(ang-mid, 2*math.Pi)
			var d float64
			if math.Abs(da) <= half {
				d = math.Abs(math.Hypot(fx, fy)-r) - lw/2
			} else {
				// round caps
				e0x, e0y := cx+r*math.Cos(a0), cy+r*math.Sin(a0)
				e1x, e1y := cx+r*math.Cos(a1), cy+r*math.Sin(a1)
				px2, py2 := float64(px)+0.5, float64(py)+0.5
				d = math.Min(math.Hypot(px2-e0x, py2-e0y), math.Hypot(px2-e1x, py2-e1y)) - lw/2
			}
			if d < 0.5 {
				c.blend(px, py, col, clamp01(0.5-d))
			}
		}
	}
}

// Line draws an anti-aliased thick line with round caps.
func (c *Canvas) Line(ax, ay, bx, by, lw float64, col Color) {
	hw := lw / 2
	x0 := max(int(math.Min(ax, bx)-hw-1), c.clip.X)
	y0 := max(int(math.Min(ay, by)-hw-1), c.clip.Y)
	x1 := min(int(math.Max(ax, bx)+hw+2), c.clip.Right())
	y1 := min(int(math.Max(ay, by)+hw+2), c.clip.Bottom())
	dx, dy := bx-ax, by-ay
	l2 := dx*dx + dy*dy
	for py := y0; py < y1; py++ {
		for px := x0; px < x1; px++ {
			fx, fy := float64(px)+0.5, float64(py)+0.5
			t := 0.0
			if l2 > 0 {
				t = clamp01(((fx-ax)*dx + (fy-ay)*dy) / l2)
			}
			d := math.Hypot(fx-(ax+t*dx), fy-(ay+t*dy)) - hw
			if d < 0.5 {
				c.blend(px, py, col, clamp01(0.5-d))
			}
		}
	}
}

// HLine / VLine: crisp 1px-ish lines.
func (c *Canvas) HLine(x0, x1, y int, col Color) { c.FillRect(Rect{x0, y, x1 - x0, 1}, col) }
func (c *Canvas) VLine(x, y0, y1 int, col Color) { c.FillRect(Rect{x, y0, 1, y1 - y0}, col) }

// DashedHLine draws a dashed horizontal line.
func (c *Canvas) DashedHLine(x0, x1, y, dash, gap int, col Color) {
	for x := x0; x < x1; x += dash + gap {
		c.FillRect(Rect{x, y, min(dash, x1-x), 1}, col)
	}
}
func (c *Canvas) DashedVLine(x, y0, y1, dash, gap int, col Color) {
	for y := y0; y < y1; y += dash + gap {
		c.FillRect(Rect{x, y, 1, min(dash, y1-y)}, col)
	}
}

// Pt is a float point.
type Pt struct{ X, Y float64 }

// FillConvex fills a convex polygon with anti-aliasing (SDF approach).
// glow > 0 adds a soft halo of that radius at alpha glowA.
func (c *Canvas) FillConvex(pts []Pt, col Color, glow float64, glowA float64) {
	n := len(pts)
	if n < 3 {
		return
	}
	minx, miny, maxx, maxy := pts[0].X, pts[0].Y, pts[0].X, pts[0].Y
	for _, p := range pts {
		minx, miny = math.Min(minx, p.X), math.Min(miny, p.Y)
		maxx, maxy = math.Max(maxx, p.X), math.Max(maxy, p.Y)
	}
	// orientation
	area := 0.0
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		area += pts[i].X*pts[j].Y - pts[j].X*pts[i].Y
	}
	sign := 1.0
	if area < 0 {
		sign = -1
	}
	type edge struct{ nx, ny, d float64 }
	edges := make([]edge, n)
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		ex, ey := pts[j].X-pts[i].X, pts[j].Y-pts[i].Y
		l := math.Hypot(ex, ey)
		if l == 0 {
			l = 1
		}
		// outward normal
		nx, ny := ey/l*sign, -ex/l*sign
		edges[i] = edge{nx, ny, nx*pts[i].X + ny*pts[i].Y}
	}
	pad := 1.0 + glow
	x0 := max(int(minx-pad), c.clip.X)
	y0 := max(int(miny-pad), c.clip.Y)
	x1 := min(int(maxx+pad)+1, c.clip.Right())
	y1 := min(int(maxy+pad)+1, c.clip.Bottom())
	for py := y0; py < y1; py++ {
		fy := float64(py) + 0.5
		for px := x0; px < x1; px++ {
			fx := float64(px) + 0.5
			d := -1e9
			for _, e := range edges {
				v := e.nx*fx + e.ny*fy - e.d
				if v > d {
					d = v
				}
			}
			if d < 0.5 {
				c.blend(px, py, col, clamp01(0.5-d))
			} else if glow > 0 && d < glow {
				t := 1 - d/glow
				c.blend(px, py, col, glowA*t*t)
			}
		}
	}
}

// Polyline draws connected thick segments.
func (c *Canvas) Polyline(pts []Pt, lw float64, col Color) {
	for i := 1; i < len(pts); i++ {
		c.Line(pts[i-1].X, pts[i-1].Y, pts[i].X, pts[i].Y, lw, col)
	}
}

// VGradient fills r with a vertical gradient.
func (c *Canvas) VGradient(r Rect, top, bot Color) {
	rr := r.Intersect(c.clip)
	for y := rr.Y; y < rr.Bottom(); y++ {
		t := float64(y-r.Y) / math.Max(1, float64(r.H-1))
		col := lerpColor(top, bot, t)
		for x := rr.X; x < rr.Right(); x++ {
			c.blend(x, y, col, 1)
		}
	}
}

// RadialGlow adds a soft coloured glow centred at (cx,cy).
func (c *Canvas) RadialGlow(cx, cy, r float64, col Color, a float64) {
	x0 := max(int(cx-r), c.clip.X)
	y0 := max(int(cy-r), c.clip.Y)
	x1 := min(int(cx+r)+1, c.clip.Right())
	y1 := min(int(cy+r)+1, c.clip.Bottom())
	for py := y0; py < y1; py++ {
		for px := x0; px < x1; px++ {
			d := math.Hypot(float64(px)-cx, float64(py)-cy) / r
			if d < 1 {
				t := 1 - d
				c.blend(px, py, col, a*t*t)
			}
		}
	}
}

// FillAreaUnder fills from a polyline down to baseline y with a vertical fade.
func (c *Canvas) FillAreaUnder(pts []Pt, base float64, col Color, a0, a1 float64) {
	if len(pts) < 2 {
		return
	}
	for i := 1; i < len(pts); i++ {
		p, q := pts[i-1], pts[i]
		xa, xb := int(math.Floor(p.X)), int(math.Ceil(q.X))
		for x := max(xa, c.clip.X); x < min(xb, c.clip.Right()); x++ {
			t := 0.0
			if q.X != p.X {
				t = clamp01((float64(x) + 0.5 - p.X) / (q.X - p.X))
			}
			yy := p.Y + (q.Y-p.Y)*t
			top := int(math.Min(yy, base))
			bot := int(math.Max(yy, base))
			h := float64(bot - top)
			for y := max(top, c.clip.Y); y < min(bot, c.clip.Bottom()); y++ {
				f := 0.0
				if h > 0 {
					f = float64(y-top) / h
				}
				c.blend(x, y, col, a0+(a1-a0)*f)
			}
		}
	}
}

// ToImage copies the canvas (or a sub-rectangle) into an RGBA image.
func (c *Canvas) ToImage(r Rect) *image.RGBA {
	r = r.Intersect(Rect{0, 0, c.W, c.H})
	img := image.NewRGBA(image.Rect(0, 0, r.W, r.H))
	for y := 0; y < r.H; y++ {
		for x := 0; x < r.W; x++ {
			i := (r.Y+y)*c.Stride + (r.X+x)*4
			img.SetRGBA(x, y, color.RGBA{c.Pix[i+2], c.Pix[i+1], c.Pix[i], 255})
		}
	}
	return img
}

// DrawImage blits an image scaled into dst (nearest/bilinear-lite).
func (c *Canvas) DrawImage(img image.Image, dst Rect) {
	if img == nil || dst.Empty() {
		return
	}
	b := img.Bounds()
	sw, sh := b.Dx(), b.Dy()
	if sw == 0 || sh == 0 {
		return
	}
	rr := dst.Intersect(c.clip)
	if g, ok := img.(*image.Gray); ok {
		for y := rr.Y; y < rr.Bottom(); y++ {
			sy := b.Min.Y + (y-dst.Y)*sh/dst.H
			for x := rr.X; x < rr.Right(); x++ {
				sx := b.Min.X + (x-dst.X)*sw/dst.W
				v := g.Pix[(sy-b.Min.Y)*g.Stride+(sx-b.Min.X)]
				i := y*c.Stride + x*4
				c.Pix[i], c.Pix[i+1], c.Pix[i+2], c.Pix[i+3] = v, v, v, 255
			}
		}
		return
	}
	if rgba, ok := img.(*image.RGBA); ok {
		for y := rr.Y; y < rr.Bottom(); y++ {
			sy := (y - dst.Y) * sh / dst.H
			for x := rr.X; x < rr.Right(); x++ {
				sx := (x - dst.X) * sw / dst.W
				j := sy*rgba.Stride + sx*4
				i := y*c.Stride + x*4
				c.Pix[i], c.Pix[i+1], c.Pix[i+2], c.Pix[i+3] = rgba.Pix[j+2], rgba.Pix[j+1], rgba.Pix[j], 255
			}
		}
		return
	}
	for y := rr.Y; y < rr.Bottom(); y++ {
		sy := b.Min.Y + (y-dst.Y)*sh/dst.H
		for x := rr.X; x < rr.Right(); x++ {
			sx := b.Min.X + (x-dst.X)*sw/dst.W
			r, g, bb, _ := img.At(sx, sy).RGBA()
			i := y*c.Stride + x*4
			c.Pix[i], c.Pix[i+1], c.Pix[i+2], c.Pix[i+3] = uint8(bb>>8), uint8(g>>8), uint8(r>>8), 255
		}
	}
}
