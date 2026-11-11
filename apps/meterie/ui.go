package meterie

// ui.go: a small immediate-mode UI toolkit with animated widgets, drawn with
// the software canvas + platform text engine. Dark, rounded, suite styling.

import (
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Font selects a typeface.
type Font struct {
	Size   float64 // px at current scale (already scaled)
	Weight int     // 400 regular, 600 semibold, 700 bold
	Mono   bool
	Icon   bool
}

// TextEngine draws text into the canvas pixels.
type TextEngine interface {
	Measure(s string, f Font) int
	LineHeight(f Font) int
	Draw(s string, x, y int, f Font, col Color)
	SetClip(r Rect)
}

// Theme colours.
var th = struct {
	Bg, Bg2, Rail, Panel, Card, Card2, Hover, Border, BorderHi Color
	Text, Sub, Faint                                           Color
	Accent, AccentHi, Accent2, Good, Warn, Bad, Info           Color
	LCD, LCDOn, LCDGhost, Grid                                 Color
}{
	Bg: hexColor(0x0B0E13), Bg2: hexColor(0x0F131A), Rail: hexColor(0x0D1117), Panel: hexColor(0x12171F),
	Card: hexColor(0x161C26), Card2: hexColor(0x1C2430), Hover: hexColor(0x232C3A), Border: hexColor(0x253041), BorderHi: hexColor(0x34425A),
	Text: hexColor(0xE8EEF6), Sub: hexColor(0x93A1B5), Faint: hexColor(0x5B6779),
	Accent: hexColor(0x3CC8F0), AccentHi: hexColor(0x7BDCF7), Accent2: hexColor(0x8B7CFF), Good: hexColor(0x34D399),
	Warn: hexColor(0xFBBF24), Bad: hexColor(0xF87171), Info: hexColor(0x60A5FA),
	LCD: hexColor(0x0A1016), LCDOn: hexColor(0x7FE7FF), LCDGhost: hexColor(0x13212B), Grid: hexColor(0x1D2633),
}

// Icon glyphs (Segoe Fluent Icons / Segoe MDL2 Assets share code points).
const (
	icHome     = ""
	icSettings = ""
	icPlay     = ""
	icPause    = ""
	icStop     = ""
	icRecord   = ""
	icSave     = ""
	icOpen     = ""
	icAdd      = ""
	icClose    = ""
	icRemove   = ""
	icDelete   = ""
	icRefresh  = ""
	icCamera   = ""
	icBT       = ""
	icUSB      = ""
	icWarn     = ""
	icInfo     = ""
	icBolt     = ""
	icChart    = ""
	icCalc     = ""
	icBulb     = ""
	icFlag     = ""
	icEdit     = ""
	icList     = ""
	icCopy     = ""
	icZoomIn   = ""
	icZoomOut  = ""
	icFit      = ""
	icCheck    = ""
	icDown     = ""
	icUp       = ""
	icLeft     = ""
	icRight    = ""
	icVolume   = ""
	icMute     = ""
	icHelp     = ""
	icDiag     = ""
	icFolder   = ""
	icClock    = ""
	icWorld    = ""
	icPhoto    = ""
	icSearch   = ""
	icSync     = ""
	icTag      = ""
	icView     = ""
	icComp     = ""
)

// Key codes delivered by the platform layer.
type Key int

const (
	KNone Key = iota
	KBack
	KDelete
	KLeft
	KRight
	KUp
	KDown
	KHome
	KEnd
	KEnter
	KEscape
	KTab
	KSpace
	KA
	KC
	KV
	KX
	KS
	KO
	KN
	KZ
	KF1
	KH
	KR
	KM
	KPlus
	KMinus
	K0
	KF2
)

// Input is the per-frame input state.
type Input struct {
	MX, MY                      int
	Down, RDown                 bool
	Pressed, Released, RPressed bool
	Dbl                         bool
	Wheel                       float64
	Keys                        []Key
	Chars                       []rune
	Ctrl, Shift                 bool
	HasMouse                    bool
}

type anim struct {
	v, target float64
	seen      bool
}

type scrollState struct {
	off, target, content float64
}

// Cursor kinds.
const (
	CurArrow = iota
	CurHand
	CurIBeam
	CurMove
	CurSizeWE
	CurCross
)

// UI is the immediate-mode context.
type UI struct {
	C  *Canvas
	T  TextEngine
	S  float64 // DPI scale
	In Input

	hot, active, focus string
	prevFocus          string
	anims              map[string]*anim
	scrolls            map[string]*scrollState
	dt                 float64
	now                time.Time
	Animating          bool
	Ambient            bool // slow continuous animation (pulses); rendered at a lower frame rate
	Cursor             int
	blocked            bool // input currently captured by a popup/modal
	overlay            []func()
	popupID            string
	popupRect          Rect
	popupOwner         Rect
	modalOpen          bool
	caret              int
	selAll             bool
	caretBlink         float64
	tooltip            string
	tipX, tipY         int
	tipHover           string
	tipT               float64
	toasts             []toast
	editBuf            map[string]string
	Clipboard          func() string
	Rects              map[string]Rect // optional: widget rects recorded for tests
	SetClipboard       func(string)

	Teach      bool // teaching mode: explain whatever is under the mouse (teach.go)
	teachZone  string
	teachLabel string
	teachIntro float64
}

type toast struct {
	msg  string
	kind int
	t    float64
}

func NewUI() *UI {
	return &UI{anims: map[string]*anim{}, scrolls: map[string]*scrollState{}, editBuf: map[string]string{}, S: 1}
}

// Px scales a design pixel value.
func (u *UI) Px(v float64) int     { return int(math.Round(v * u.S)) }
func (u *UI) Pf(v float64) float64 { return v * u.S }

// Fonts.
func (u *UI) F(size float64) Font  { return Font{Size: size * u.S, Weight: 400} }
func (u *UI) FB(size float64) Font { return Font{Size: size * u.S, Weight: 600} }
func (u *UI) FH(size float64) Font { return Font{Size: size * u.S, Weight: 700} }
func (u *UI) FM(size float64) Font { return Font{Size: size * u.S, Weight: 400, Mono: true} }
func (u *UI) FI(size float64) Font { return Font{Size: size * u.S, Icon: true} }

// Begin starts a frame.
func (u *UI) Begin(c *Canvas, t TextEngine, in Input, dt float64) {
	u.C, u.T, u.In = c, t, in
	u.dt = math.Min(dt, 0.1)
	u.now = time.Now()
	u.Animating = false
	u.Ambient = false
	u.Cursor = CurArrow
	u.hot = ""
	u.overlay = u.overlay[:0]
	u.blocked = false
	if u.popupID != "" && u.popupRect.Contains(in.MX, in.MY) {
		u.blocked = true // main pass shouldn't react under an open popup
	}
	if u.modalOpen {
		u.blocked = true
	}
	u.modalOpen = false
	for _, a := range u.anims {
		a.seen = false
	}
	u.caretBlink += u.dt
	if u.focus != "" {
		u.Ambient = true // caret blink
	}
	u.tooltip = ""
	u.teachZone, u.teachLabel = "", ""
	c.ResetClip()
	t.SetClip(Rect{0, 0, c.W, c.H})
}

// End draws overlays, toasts and tooltips.
func (u *UI) End() {
	// overlays capture input
	u.blocked = false
	ov := u.overlay
	u.overlay = nil
	for _, f := range ov {
		f()
	}
	u.drawToasts()
	if !u.drawTeach() {
		u.drawTooltip()
	}
	if u.In.Pressed && u.popupID != "" && !u.popupRect.Contains(u.In.MX, u.In.MY) && !u.popupOwner.Contains(u.In.MX, u.In.MY) {
		u.popupID = ""
	}
	if u.In.Released {
		u.active = ""
	}
	if u.In.Pressed && u.hot == "" && u.focus != "" {
		u.focus = ""
	}
	for k, a := range u.anims {
		if !a.seen {
			delete(u.anims, k)
		}
	}
}

// Anim eases toward target and returns the current value.
func (u *UI) Anim(id string, target, speed float64) float64 {
	a := u.anims[id]
	if a == nil {
		a = &anim{v: target, target: target}
		u.anims[id] = a
	}
	a.seen = true
	a.target = target
	if reduceMotion {
		a.v = target
		return a.v
	}
	k := 1 - math.Exp(-speed*u.dt)
	a.v += (target - a.v) * k
	if math.Abs(a.v-target) > 0.002 {
		u.Animating = true
	} else {
		a.v = target
	}
	return a.v
}

// Hover returns whether the mouse is over r and input isn't blocked.
func (u *UI) Hover(r Rect) bool {
	return !u.blocked && u.In.HasMouse && r.Contains(u.In.MX, u.In.MY) && u.C.Clip().Contains(u.In.MX, u.In.MY)
}

// interact handles press/release behaviour. Returns (hovered, clicked).
func (u *UI) interact(id string, r Rect) (bool, bool) {
	if u.Rects != nil {
		u.Rects[id] = r
	}
	h := u.Hover(r)
	if h {
		u.hot = id
		u.Cursor = CurHand
		if u.In.Pressed {
			u.active = id
		}
	}
	clicked := h && u.In.Released && u.active == id
	return h, clicked
}

// Toast shows a transient message. kind: 0 info, 1 good, 2 warn, 3 bad.
func (u *UI) Toast(msg string, kind int) {
	u.toasts = append(u.toasts, toast{msg: msg, kind: kind, t: 0})
	if len(u.toasts) > 4 {
		u.toasts = u.toasts[1:]
	}
}

func (u *UI) drawToasts() {
	if len(u.toasts) == 0 {
		return
	}
	u.Animating = true
	y := u.C.H - u.Px(24)
	keep := u.toasts[:0]
	for i := len(u.toasts) - 1; i >= 0; i-- {
		t := &u.toasts[i]
		t.t += u.dt
		life := 3.6
		if t.t > life {
			continue
		}
		in := clamp01(t.t / 0.25)
		out := clamp01((life - t.t) / 0.4)
		a := math.Min(in, out)
		f := u.FB(13)
		w := u.T.Measure(t.msg, f) + u.Px(56)
		h := u.Px(42)
		x := u.C.W - w - u.Px(24) + int((1-easeOut(in))*float64(u.Px(40)))
		y -= h
		r := Rect{x, y, w, h}
		col := []Color{th.Accent, th.Good, th.Warn, th.Bad}[clampi(t.kind, 0, 3)]
		u.C.Shadow(r, u.Pf(10), u.Pf(18), u.Pf(6), RGBA(0, 0, 0, uint8(140*a)))
		u.C.FillRoundRect(r, u.Pf(10), lerpColor(th.Bg, th.Card2, a))
		u.C.StrokeRoundRect(r, u.Pf(10), u.Pf(1), lerpColor(th.Bg, col.Darken(0.35), a))
		u.C.FillCircle(float64(x+u.Px(20)), float64(y+h/2), u.Pf(4), col.WithA(a))
		u.T.Draw(t.msg, x+u.Px(34), y+(h-u.T.LineHeight(f))/2, f, lerpColor(th.Bg, th.Text, a))
		y -= u.Px(8)
		_ = i
	}
	for _, t := range u.toasts {
		if t.t <= 3.6 {
			keep = append(keep, t)
		}
	}
	u.toasts = keep
}

func easeOut(t float64) float64   { t = clamp01(t); return 1 - (1-t)*(1-t)*(1-t) }
func easeInOut(t float64) float64 { t = clamp01(t); return t * t * (3 - 2*t) }

// Tip sets a tooltip for a hovered rect.
func (u *UI) Tip(id string, r Rect, text string) {
	if text == "" || !u.Hover(r) {
		return
	}
	if u.tipHover != id {
		u.tipHover = id
		u.tipT = 0
	}
	u.tipT += u.dt
	u.tooltip = text
	u.tipX, u.tipY = u.In.MX, r.Bottom()+u.Px(6)
	if u.tipT < 0.45 {
		u.Animating = true
	}
}

func (u *UI) drawTooltip() {
	if u.tooltip == "" || u.tipT < 0.45 {
		if u.tooltip == "" {
			u.tipHover = ""
		}
		return
	}
	f := u.F(12)
	lines := strings.Split(u.tooltip, "\n")
	w := 0
	for _, l := range lines {
		w = max(w, u.T.Measure(l, f))
	}
	lh := u.T.LineHeight(f)
	r := Rect{u.tipX - w/2 - u.Px(10), u.tipY, w + u.Px(20), lh*len(lines) + u.Px(12)}
	if r.X < 4 {
		r.X = 4
	}
	if r.Right() > u.C.W-4 {
		r.X = u.C.W - 4 - r.W
	}
	if r.Bottom() > u.C.H-4 {
		r.Y = u.tipY - r.H - u.Px(40)
	}
	u.C.Shadow(r, u.Pf(6), u.Pf(10), u.Pf(3), RGBA(0, 0, 0, 120))
	u.C.FillRoundRect(r, u.Pf(6), th.Hover)
	u.C.StrokeRoundRect(r, u.Pf(6), 1, th.BorderHi)
	for i, l := range lines {
		u.T.Draw(l, r.X+u.Px(10), r.Y+u.Px(6)+i*lh, f, th.Text)
	}
}

// Clip helpers keep canvas and text clip in sync.
func (u *UI) PushClip(r Rect) {
	u.C.PushClip(r)
	u.T.SetClip(u.C.Clip())
}
func (u *UI) PopClip() {
	u.C.PopClip()
	u.T.SetClip(u.C.Clip())
}

// ---------------------------------------------------------------------------
// Text helpers

const (
	AlignLeft = iota
	AlignCenter
	AlignRight
)

// Text draws a single line aligned inside r (vertically centred). Ellipsises if needed.
func (u *UI) Text(s string, r Rect, f Font, col Color, align int) {
	w := u.T.Measure(s, f)
	if w > r.W && r.W > 0 && !f.Icon {
		s = u.ellipsis(s, r.W, f)
		w = u.T.Measure(s, f)
	}
	lh := u.T.LineHeight(f)
	y := r.Y + (r.H-lh)/2
	x := r.X
	switch align {
	case AlignCenter:
		x = r.X + (r.W-w)/2
	case AlignRight:
		x = r.Right() - w
	}
	u.T.Draw(s, x, y, f, col)
}

// TextAt draws at an exact position (top-left).
func (u *UI) TextAt(s string, x, y int, f Font, col Color) int {
	u.T.Draw(s, x, y, f, col)
	return u.T.Measure(s, f)
}

func (u *UI) ellipsis(s string, w int, f Font) string {
	if w <= 0 {
		return ""
	}
	rs := []rune(s)
	for len(rs) > 0 && u.T.Measure(string(rs)+"…", f) > w {
		rs = rs[:len(rs)-1]
	}
	return string(rs) + "…"
}

// Wrap draws word-wrapped text and returns the height used.
func (u *UI) Wrap(s string, x, y, w int, f Font, col Color) int {
	lines := u.WrapLines(s, w, f)
	lh := u.T.LineHeight(f) + u.Px(3)
	for i, l := range lines {
		u.T.Draw(l, x, y+i*lh, f, col)
	}
	return len(lines) * lh
}

func (u *UI) WrapHeight(s string, w int, f Font) int {
	return len(u.WrapLines(s, w, f)) * (u.T.LineHeight(f) + u.Px(3))
}

func (u *UI) WrapLines(s string, w int, f Font) []string {
	var out []string
	for _, para := range strings.Split(s, "\n") {
		words := strings.Fields(para)
		if len(words) == 0 {
			out = append(out, "")
			continue
		}
		line := words[0]
		for _, wd := range words[1:] {
			t := line + " " + wd
			if u.T.Measure(t, f) > w {
				out = append(out, line)
				line = wd
			} else {
				line = t
			}
		}
		out = append(out, line)
	}
	return out
}

// ---------------------------------------------------------------------------
// Containers

// Card draws a rounded panel with optional title.
func (u *UI) Card(r Rect, title string) Rect {
	u.C.Shadow(r, u.Pf(14), u.Pf(16), u.Pf(4), RGBA(0, 0, 0, 90))
	u.C.FillRoundRect(r, u.Pf(14), th.Card)
	u.C.StrokeRoundRect(r, u.Pf(14), u.Pf(1), th.Border)
	inner := r.InsetXY(u.Px(18), u.Px(16))
	if title != "" {
		u.Text(strings.ToUpper(title), Rect{inner.X, inner.Y, inner.W, u.Px(18)}, u.FB(11.5), th.Sub, AlignLeft)
		inner.Y += u.Px(28)
		inner.H -= u.Px(28)
	}
	return inner
}

// Panel draws a flat inner panel.
func (u *UI) Panel(r Rect, col Color) {
	u.C.FillRoundRect(r, u.Pf(10), col)
}

// SectionTitle draws a title + optional subtitle; returns height.
func (u *UI) SectionTitle(x, y, w int, title, sub string) int {
	u.T.Draw(title, x, y, u.FH(22), th.Text)
	h := u.T.LineHeight(u.FH(22))
	if sub != "" {
		h += u.Px(4)
		h += u.Wrap(sub, x, y+h, w, u.F(13.5), th.Sub)
	}
	return h + u.Px(14)
}

// ---------------------------------------------------------------------------
// Buttons

const (
	BtnPrimary = iota
	BtnSecondary
	BtnGhost
	BtnDanger
	BtnGood
)

// Button with optional icon. Returns true when clicked.
func (u *UI) Button(id string, r Rect, icon, label string, style int) bool {
	return u.ButtonEx(id, r, icon, label, style, true)
}

func (u *UI) ButtonEx(id string, r Rect, icon, label string, style int, enabled bool) bool {
	h, clicked := u.interact(id, r)
	defer u.noteLabel(id, label)
	if !enabled {
		h, clicked = false, false
		if u.hot == id {
			u.Cursor = CurArrow
		}
	}
	hv := u.Anim(id+"#h", b2f(h), 14)
	pr := u.Anim(id+"#p", b2f(h && u.active == id && u.In.Down), 22)
	var bg, fg, bd Color
	switch style {
	case BtnPrimary:
		bg = lerpColor(th.Accent, th.AccentHi, hv*0.5)
		fg = hexColor(0x061018)
		bd = bg
	case BtnDanger:
		bg = lerpColor(hexColor(0x3A1C22), hexColor(0x4E2128), hv)
		fg = hexColor(0xFFB4B4)
		bd = lerpColor(hexColor(0x5A2A31), th.Bad, hv*0.6)
	case BtnGood:
		bg = lerpColor(hexColor(0x15372B), hexColor(0x1B4535), hv)
		fg = hexColor(0x9FF5D2)
		bd = lerpColor(hexColor(0x1F5240), th.Good, hv*0.6)
	case BtnGhost:
		bg = th.Hover.WithA(hv)
		fg = lerpColor(th.Sub, th.Text, hv)
		bd = Color{}
	default:
		bg = lerpColor(th.Card2, th.Hover, hv)
		fg = th.Text
		bd = lerpColor(th.Border, th.BorderHi, hv)
	}
	if !enabled {
		bg = th.Card2.WithA(0.5)
		fg = th.Faint
		bd = th.Border.WithA(0.5)
	}
	rr := r
	if pr > 0.01 {
		d := int(pr * u.Pf(1.5))
		rr = r.Inset(d)
	}
	rad := u.Pf(9)
	if style == BtnPrimary && enabled {
		u.C.Shadow(rr, rad, u.Pf(12), u.Pf(3), th.Accent.WithA(0.18+0.2*hv))
	}
	u.C.FillRoundRect(rr, rad, bg)
	if bd.A > 0 {
		u.C.StrokeRoundRect(rr, rad, u.Pf(1), bd)
	}
	fi, fl := u.FI(15), u.FB(13.5)
	tw := 0
	if label != "" {
		tw = u.T.Measure(label, fl)
	}
	iw := 0
	if icon != "" {
		iw = u.T.Measure(icon, fi)
	}
	gap := 0
	if iw > 0 && tw > 0 {
		gap = u.Px(8)
	}
	total := iw + gap + tw
	x := rr.X + (rr.W-total)/2
	if icon != "" {
		u.Text(icon, Rect{x, rr.Y, iw, rr.H}, fi, fg, AlignLeft)
	}
	if label != "" {
		u.Text(label, Rect{x + iw + gap, rr.Y, rr.W - (x - rr.X) - iw - gap, rr.H}, fl, fg, AlignLeft)
	}
	return clicked
}

// IconButton is a square ghost button with a tooltip.
func (u *UI) IconButton(id string, r Rect, icon, tip string, on bool) bool {
	h, clicked := u.interact(id, r)
	hv := u.Anim(id+"#h", b2f(h), 14)
	ov := u.Anim(id+"#o", b2f(on), 12)
	bg := lerpColor(th.Card2.WithA(0), th.Hover, hv)
	if ov > 0.01 {
		bg = lerpColor(bg, th.Accent.WithA(0.22), ov)
	}
	u.C.FillRoundRect(r, u.Pf(8), bg)
	if ov > 0.01 {
		u.C.StrokeRoundRect(r, u.Pf(8), u.Pf(1), th.Accent.WithA(0.55*ov))
	}
	fg := lerpColor(th.Sub, th.Text, hv)
	fg = lerpColor(fg, th.AccentHi, ov)
	u.Text(icon, r, u.FI(15), fg, AlignCenter)
	u.Tip(id, r, tip)
	return clicked
}

// Toggle is an animated switch with a label to its right. Returns true if changed.
func (u *UI) Toggle(id string, r Rect, v *bool, label string) bool {
	defer u.noteLabel(id, label)
	h, clicked := u.interact(id, r)
	if clicked {
		*v = !*v
	}
	t := u.Anim(id+"#t", b2f(*v), 16)
	hv := u.Anim(id+"#h", b2f(h), 14)
	sw := Rect{r.X, r.Y + (r.H-u.Px(22))/2, u.Px(40), u.Px(22)}
	track := lerpColor(th.Card2, th.Accent, t)
	track = lerpColor(track, track.Lighten(0.12), hv)
	u.C.FillRoundRect(sw, u.Pf(11), track)
	u.C.StrokeRoundRect(sw, u.Pf(11), u.Pf(1), lerpColor(th.BorderHi, th.Accent, t))
	kx := float64(sw.X) + u.Pf(11) + t*u.Pf(18)
	u.C.FillCircle(kx, float64(sw.CY()), u.Pf(8), lerpColor(th.Sub, hexColor(0x061018), t))
	if label != "" {
		u.Text(label, Rect{sw.Right() + u.Px(10), r.Y, r.W - sw.W - u.Px(10), r.H}, u.F(13.5), th.Text, AlignLeft)
	}
	return clicked
}

// Checkbox with label.
func (u *UI) Checkbox(id string, r Rect, v *bool, label string) bool {
	defer u.noteLabel(id, label)
	h, clicked := u.interact(id, r)
	if clicked {
		*v = !*v
	}
	t := u.Anim(id+"#t", b2f(*v), 18)
	hv := u.Anim(id+"#h", b2f(h), 14)
	b := Rect{r.X, r.Y + (r.H-u.Px(18))/2, u.Px(18), u.Px(18)}
	u.C.FillRoundRect(b, u.Pf(5), lerpColor(lerpColor(th.Card2, th.Hover, hv), th.Accent, t))
	u.C.StrokeRoundRect(b, u.Pf(5), u.Pf(1), lerpColor(th.BorderHi, th.Accent, t))
	if t > 0.05 {
		x, y, s := float64(b.X), float64(b.Y), u.Pf(1)
		c := hexColor(0x061018).WithA(t)
		u.C.Line(x+4.5*s, y+9.5*s, x+7.5*s, y+12.5*s, 2*s, c)
		u.C.Line(x+7.5*s, y+12.5*s, x+13.5*s, y+5.5*s, 2*s, c)
	}
	if label != "" {
		u.Text(label, Rect{b.Right() + u.Px(9), r.Y, r.W - b.W - u.Px(9), r.H}, u.F(13.5), th.Text, AlignLeft)
	}
	return clicked
}

// Segmented control. Returns true when the selection changed.
func (u *UI) Segmented(id string, r Rect, opts []string, sel *int) bool {
	if len(opts) == 0 {
		return false
	}
	changed := false
	u.C.FillRoundRect(r, u.Pf(9), th.Bg2)
	u.C.StrokeRoundRect(r, u.Pf(9), u.Pf(1), th.Border)
	pad := u.Px(3)
	inner := r.Inset(pad)
	segW := float64(inner.W) / float64(len(opts))
	pos := u.Anim(id+"#pos", float64(*sel), 16)
	hl := Rect{inner.X + int(pos*segW), inner.Y, int(segW), inner.H}
	u.C.FillRoundRect(hl, u.Pf(7), th.Hover)
	u.C.StrokeRoundRect(hl, u.Pf(7), u.Pf(1), th.BorderHi)
	for i, o := range opts {
		sr := Rect{inner.X + int(float64(i)*segW), inner.Y, int(segW), inner.H}
		sid := id + "#" + itoa(i)
		h, clicked := u.interact(sid, sr)
		if clicked && *sel != i {
			*sel = i
			changed = true
		}
		hv := u.Anim(sid+"#h", b2f(h), 14)
		col := lerpColor(th.Sub, th.Text, math.Max(hv*0.7, b2f(*sel == i)))
		u.Text(o, sr, u.FB(12.5), col, AlignCenter)
	}
	return changed
}

// Dropdown: returns true when changed.
func (u *UI) Dropdown(id string, r Rect, opts []string, sel *int) bool {
	h, clicked := u.interact(id, r)
	open := u.popupID == id
	if clicked {
		if open {
			u.popupID = ""
		} else {
			u.popupID = id
		}
		open = !open
	}
	hv := u.Anim(id+"#h", b2f(h || open), 14)
	u.C.FillRoundRect(r, u.Pf(9), lerpColor(th.Card2, th.Hover, hv))
	u.C.StrokeRoundRect(r, u.Pf(9), u.Pf(1), lerpColor(th.Border, th.Accent.WithA(0.8), b2f(open)*0.8+hv*0.2))
	label := ""
	if *sel >= 0 && *sel < len(opts) {
		label = opts[*sel]
	}
	u.Text(label, Rect{r.X + u.Px(12), r.Y, r.W - u.Px(40), r.H}, u.F(13.5), th.Text, AlignLeft)
	rot := u.Anim(id+"#rot", b2f(open), 14)
	ic := icDown
	if rot > 0.5 {
		ic = icUp
	}
	u.Text(ic, Rect{r.Right() - u.Px(30), r.Y, u.Px(20), r.H}, u.FI(11), th.Sub, AlignCenter)
	changed := false
	if open {
		itemH := u.Px(32)
		maxVis := 10
		n := len(opts)
		vis := min(n, maxVis)
		pr := Rect{r.X, r.Bottom() + u.Px(4), max(r.W, u.Px(180)), vis*itemH + u.Px(8)}
		if pr.Bottom() > u.C.H-u.Px(8) {
			pr.Y = r.Y - pr.H - u.Px(4)
		}
		u.popupRect = pr
		u.popupOwner = r
		selv := *sel
		u.overlay = append(u.overlay, func() {
			prog := u.Anim(id+"#open", 1, 18)
			ph := int(float64(pr.H) * easeOut(prog))
			dr := Rect{pr.X, pr.Y, pr.W, ph}
			u.C.Shadow(dr, u.Pf(10), u.Pf(18), u.Pf(6), RGBA(0, 0, 0, 160))
			u.C.FillRoundRect(dr, u.Pf(10), th.Card2)
			u.C.StrokeRoundRect(dr, u.Pf(10), u.Pf(1), th.BorderHi)
			u.PushClip(dr.Inset(u.Px(4)))
			sid := id + "#list"
			ss := u.scroll(sid)
			total := float64(n * itemH)
			viewH := float64(vis * itemH)
			if u.Hover(pr) && u.In.Wheel != 0 {
				ss.target -= u.In.Wheel * float64(itemH) * 1.5
			}
			ss.target = clampf(ss.target, 0, math.Max(0, total-viewH))
			ss.off += (ss.target - ss.off) * (1 - math.Exp(-18*u.dt))
			if math.Abs(ss.off-ss.target) > 0.5 {
				u.Animating = true
			}
			for i, o := range opts {
				ir := Rect{pr.X + u.Px(4), pr.Y + u.Px(4) + i*itemH - int(ss.off), pr.W - u.Px(8), itemH}
				if ir.Bottom() < pr.Y || ir.Y > pr.Bottom() {
					continue
				}
				iid := id + "#i" + itoa(i)
				hh, cl := u.interact(iid, ir)
				hv := u.Anim(iid+"#h", b2f(hh), 16)
				if i == selv {
					u.C.FillRoundRect(ir, u.Pf(7), th.Accent.WithA(0.16))
				}
				if hv > 0.01 {
					u.C.FillRoundRect(ir, u.Pf(7), th.Hover.WithA(hv))
				}
				col := th.Text
				if i == selv {
					col = th.AccentHi
				}
				u.Text(o, Rect{ir.X + u.Px(10), ir.Y, ir.W - u.Px(20), ir.H}, u.F(13.5), col, AlignLeft)
				if cl {
					*sel = i
					changed = true
					u.popupID = ""
				}
			}
			u.PopClip()
		})
	} else {
		u.Anim(id+"#open", 0, 100)
	}
	return changed
}

func (u *UI) scroll(id string) *scrollState {
	s := u.scrolls[id]
	if s == nil {
		s = &scrollState{}
		u.scrolls[id] = s
	}
	return s
}

// TextField edits *s. Returns (changed, enterPressed).
func (u *UI) TextField(id string, r Rect, s *string, placeholder string) (bool, bool) {
	h, clicked := u.interact(id, r)
	if h {
		u.Cursor = CurIBeam
	}
	focused := u.focus == id
	if clicked && !focused {
		u.focus = id
		focused = true
		u.caret = utf8.RuneCountInString(*s)
		u.selAll = true
		u.caretBlink = 0
	}
	changed, enter := false, false
	if focused {
		rs := []rune(*s)
		if u.caret > len(rs) {
			u.caret = len(rs)
		}
		for _, k := range u.In.Keys {
			switch k {
			case KBack:
				if u.selAll {
					rs, u.caret = nil, 0
				} else if u.caret > 0 {
					rs = append(rs[:u.caret-1], rs[u.caret:]...)
					u.caret--
				}
				changed = true
			case KDelete:
				if u.selAll {
					rs, u.caret = nil, 0
				} else if u.caret < len(rs) {
					rs = append(rs[:u.caret], rs[u.caret+1:]...)
				}
				changed = true
			case KLeft:
				if u.caret > 0 {
					u.caret--
				}
			case KRight:
				if u.caret < len(rs) {
					u.caret++
				}
			case KHome:
				u.caret = 0
			case KEnd:
				u.caret = len(rs)
			case KEnter:
				enter = true
			case KEscape, KTab:
				u.focus = ""
			case KA:
				if u.In.Ctrl {
					u.selAll = true
					continue
				}
			case KC:
				if u.In.Ctrl && u.SetClipboard != nil {
					u.SetClipboard(string(rs))
				}
			case KV:
				if u.In.Ctrl && u.Clipboard != nil {
					p := []rune(strings.ReplaceAll(strings.ReplaceAll(u.Clipboard(), "\r", ""), "\n", " "))
					if u.selAll {
						rs, u.caret = nil, 0
					}
					rs = append(rs[:u.caret], append(p, rs[u.caret:]...)...)
					u.caret += len(p)
					changed = true
				}
			}
			switch k { // only editing / caret keys drop the "select all" state
			case KBack, KDelete, KLeft, KRight, KHome, KEnd, KV:
				u.selAll = false
			}
			u.caretBlink = 0
		}
		for _, c := range u.In.Chars {
			if c < 32 || c == 127 {
				continue
			}
			if u.selAll {
				rs, u.caret = nil, 0
				u.selAll = false
			}
			rs = append(rs[:u.caret], append([]rune{c}, rs[u.caret:]...)...)
			u.caret++
			changed = true
			u.caretBlink = 0
		}
		*s = string(rs)
	}
	fv := u.Anim(id+"#f", b2f(focused), 16)
	hv := u.Anim(id+"#h", b2f(h), 14)
	u.C.FillRoundRect(r, u.Pf(9), lerpColor(th.Bg2, th.Bg2.Lighten(0.03), hv))
	u.C.StrokeRoundRect(r, u.Pf(9), u.Pf(1), lerpColor(lerpColor(th.Border, th.BorderHi, hv), th.Accent, fv))
	if fv > 0.01 {
		u.C.StrokeRoundRectF(float64(r.X)-u.Pf(2), float64(r.Y)-u.Pf(2), float64(r.W)+u.Pf(4), float64(r.H)+u.Pf(4), u.Pf(11), u.Pf(2), th.Accent.WithA(0.25*fv))
	}
	f := u.F(13.5)
	tr := Rect{r.X + u.Px(11), r.Y, r.W - u.Px(22), r.H}
	u.PushClip(tr)
	if *s == "" && !focused {
		u.Text(placeholder, tr, f, th.Faint, AlignLeft)
	} else {
		lh := u.T.LineHeight(f)
		ty := r.Y + (r.H-lh)/2
		rs := []rune(*s)
		cx := u.T.Measure(string(rs[:min(u.caret, len(rs))]), f)
		shift := 0
		if focused && cx > tr.W-u.Px(4) {
			shift = cx - tr.W + u.Px(4)
		}
		if focused && u.selAll && len(rs) > 0 {
			w := u.T.Measure(*s, f)
			u.C.FillRect(Rect{tr.X - shift, ty, w, lh}, th.Accent.WithA(0.3))
		}
		u.T.Draw(*s, tr.X-shift, ty, f, th.Text)
		if focused && math.Mod(u.caretBlink, 1.0) < 0.6 {
			u.C.FillRect(Rect{tr.X + cx - shift, ty + u.Px(1), max(1, u.Px(1.5)), lh - u.Px(2)}, th.AccentHi)
		}
	}
	u.PopClip()
	return changed, enter
}

// NumberField edits a float with a label suffix. Returns true when committed.
func (u *UI) NumberField(id string, r Rect, v *float64, unit string) bool {
	buf, ok := u.editBuf[id]
	if !ok || u.focus != id {
		buf = trimFloat(*v)
	}
	changed, enter := u.TextField(id, r, &buf, "")
	u.editBuf[id] = buf
	committed := false
	if changed || enter {
		if f, err := parseNumberSI(buf); err == nil {
			*v = f
			committed = true
		}
	}
	if u.focus != id {
		delete(u.editBuf, id)
	}
	if unit != "" {
		f := u.F(12.5)
		w := u.T.Measure(unit, f)
		u.Text(unit, Rect{r.Right() - w - u.Px(11), r.Y, w, r.H}, f, th.Sub, AlignLeft)
	}
	return committed
}

func trimFloat(v float64) string {
	if math.IsNaN(v) {
		return ""
	}
	a := math.Abs(v)
	if a != 0 && (a < 1e-4 || a >= 1e12) {
		return strconv.FormatFloat(v, 'g', 6, 64)
	}
	s := strconv.FormatFloat(v, 'f', 6, 64)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	}
	if s == "-0" {
		s = "0"
	}
	return s
}

// Slider (horizontal). Returns true while changing.
func (u *UI) Slider(id string, r Rect, v *float64, lo, hi float64) bool {
	h := u.Hover(r)
	if h {
		u.hot = id
		u.Cursor = CurHand
		if u.In.Pressed {
			u.active = id
		}
	}
	changed := false
	if u.active == id && u.In.Down {
		t := clamp01(float64(u.In.MX-r.X) / float64(max(1, r.W)))
		nv := lo + t*(hi-lo)
		if nv != *v {
			*v = nv
			changed = true
		}
	}
	t := clamp01((*v - lo) / (hi - lo))
	hv := u.Anim(id+"#h", b2f(h || u.active == id), 14)
	ty := float64(r.CY())
	u.C.FillRoundRectF(float64(r.X), ty-u.Pf(3), float64(r.W), u.Pf(6), u.Pf(3), th.Card2)
	u.C.FillRoundRectF(float64(r.X), ty-u.Pf(3), float64(r.W)*t, u.Pf(6), u.Pf(3), th.Accent)
	kx := float64(r.X) + float64(r.W)*t
	u.C.FillCircle(kx, ty, u.Pf(8+2*hv), th.Accent.WithA(0.25*hv))
	u.C.FillCircle(kx, ty, u.Pf(7), th.Text)
	return changed
}

// Chip draws a pill badge; returns width.
func (u *UI) Chip(x, y int, text string, fg, bg Color) int {
	f := u.FB(11)
	w := u.T.Measure(text, f) + u.Px(16)
	h := u.Px(20)
	r := Rect{x, y, w, h}
	u.C.FillRoundRect(r, u.Pf(10), bg)
	u.Text(text, r, f, fg, AlignCenter)
	return w
}

// ListItem is a hoverable row; returns clicked.
func (u *UI) ListItem(id string, r Rect, selected bool) bool {
	h, cl := u.interact(id, r)
	hv := u.Anim(id+"#h", b2f(h), 14)
	sv := u.Anim(id+"#s", b2f(selected), 14)
	if sv > 0.01 {
		u.C.FillRoundRect(r, u.Pf(9), th.Accent.WithA(0.13*sv))
		u.C.StrokeRoundRect(r, u.Pf(9), u.Pf(1), th.Accent.WithA(0.45*sv))
	}
	if hv > 0.01 {
		u.C.FillRoundRect(r, u.Pf(9), th.Hover.WithA(0.7*hv))
	}
	return cl
}

// BeginScroll starts a vertically scrolling region. Returns the y offset to apply.
func (u *UI) BeginScroll(id string, r Rect) int {
	ss := u.scroll(id)
	if u.Hover(r) && u.In.Wheel != 0 {
		ss.target -= u.In.Wheel * u.Pf(60)
	}
	maxOff := math.Max(0, ss.content-float64(r.H))
	ss.target = clampf(ss.target, 0, maxOff)
	ss.off += (ss.target - ss.off) * (1 - math.Exp(-16*u.dt))
	if math.Abs(ss.off-ss.target) > 0.5 {
		u.Animating = true
	} else {
		ss.off = ss.target
	}
	u.PushClip(r)
	return int(ss.off)
}

// EndScroll records the content height and draws a slim scrollbar.
func (u *UI) EndScroll(id string, r Rect, contentH int) {
	u.PopClip()
	ss := u.scroll(id)
	ss.content = float64(contentH)
	if contentH > r.H {
		frac := float64(r.H) / float64(contentH)
		bh := math.Max(u.Pf(30), float64(r.H)*frac)
		pos := ss.off / math.Max(1, float64(contentH-r.H))
		by := float64(r.Y) + pos*(float64(r.H)-bh)
		hv := u.Anim(id+"#sb", b2f(u.Hover(r)), 10)
		u.C.FillRoundRectF(float64(r.Right())-u.Pf(6), by, u.Pf(4), bh, u.Pf(2), th.BorderHi.WithA(0.4+0.5*hv))
	}
}

// Modal draws a centred dialog; returns the content rect. Call ui.CloseModal to dismiss.
func (u *UI) Modal(id string, w, h int, title string) (Rect, bool) {
	u.modalOpen = true
	t := u.Anim(id+"#m", 1, 14)
	u.C.FillRect(Rect{0, 0, u.C.W, u.C.H}, RGBA(3, 6, 10, uint8(170*t)))
	r := Rect{(u.C.W - w) / 2, (u.C.H-h)/2 + int((1-easeOut(t))*u.Pf(24)), w, h}
	u.C.Shadow(r, u.Pf(16), u.Pf(30), u.Pf(10), RGBA(0, 0, 0, 180))
	u.C.FillRoundRect(r, u.Pf(16), th.Card)
	u.C.StrokeRoundRect(r, u.Pf(16), u.Pf(1), th.BorderHi)
	u.T.Draw(title, r.X+u.Px(24), r.Y+u.Px(20), u.FH(18), th.Text)
	u.blocked = false
	closeR := Rect{r.Right() - u.Px(44), r.Y + u.Px(14), u.Px(30), u.Px(30)}
	closed := u.IconButton(id+"#x", closeR, icClose, "Close", false)
	for _, k := range u.In.Keys {
		if k == KEscape {
			closed = true
		}
	}
	return Rect{r.X + u.Px(24), r.Y + u.Px(62), r.W - u.Px(48), r.H - u.Px(86)}, closed
}

// EndModal must be called after drawing modal content.
func (u *UI) EndModal() {
	u.blocked = true
}

func (u *UI) CloseModal() { u.modalOpen = false }

func b2f(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
