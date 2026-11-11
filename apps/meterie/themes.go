package meterie

// themes.go: Meterie's colour themes, chosen on the Settings page: the same set
// as the rest of Tinkerie's Look & accessibility panel (Dark and nine more dark
// themes, seven light ones, high contrast and colour-blind safe, dark and light),
// and reduced motion. Created by nrmluvdev.

import "math"

type themeDef struct {
	name string
	base string  // "dark", "light", "black", "contrast", "contrastl"
	h, s float64 // tint for backgrounds, borders and text (hue in degrees); s < 0 means none
	cvd  bool    // colour-blind safe: greens become blue, reds orange
}

// The first four keep the numbers saved by earlier versions.
var themes = []themeDef{
	{"Dark", "dark", 0, -1, false}, {"Light", "light", 0, -1, false}, {"High contrast", "contrast", 0, -1, false}, {"Colour-blind safe", "dark", 0, -1, true},
	{"Midnight", "dark", 225, 0.42, false}, {"Ocean", "dark", 198, 0.5, false}, {"Forest", "dark", 145, 0.32, false}, {"Plum", "dark", 285, 0.3, false},
	{"Ember", "dark", 12, 0.32, false}, {"Mocha", "dark", 30, 0.22, false}, {"Slate", "dark", 210, 0.16, false}, {"Graphite", "dark", 0, 0, false},
	{"True black", "black", 0, -1, false},
	{"Paper", "light", 45, 0.12, false}, {"Sepia", "light", 36, 0.45, false}, {"Sky", "light", 205, 0.55, false}, {"Mint", "light", 155, 0.4, false},
	{"Rose", "light", 340, 0.45, false}, {"Lavender", "light", 265, 0.4, false},
	{"High contrast light", "contrastl", 0, -1, false}, {"Colour-blind safe light", "light", 0, -1, true},
}

func themeNamesList() []string {
	out := make([]string, len(themes))
	for i, t := range themes {
		out[i] = t.name
	}
	return out
}

// the original dark palette, kept so switching back restores it exactly
var thDark = th
var channelPaletteDark = append([]Color(nil), channelPalette...)

// Okabe-Ito colours: told apart with every common kind of colour blindness
var channelPaletteCVD = []Color{hexColor(0x56B4E9), hexColor(0xE69F00), hexColor(0xF0E442), hexColor(0xCC79A7), hexColor(0x009E73), hexColor(0xD55E00), hexColor(0x0072B2), hexColor(0xFFFFFF)}

// reduceMotion makes animations jump straight to where they end.
var reduceMotion bool

const (
	roleBg = iota
	roleLine
	roleText
	roleAccent
)

func rgbToHSL(c Color) (h, s, l float64) {
	r, g, b := float64(c.R)/255, float64(c.G)/255, float64(c.B)/255
	mx, mn := math.Max(r, math.Max(g, b)), math.Min(r, math.Min(g, b))
	l = (mx + mn) / 2
	if mx == mn {
		return 0, 0, l
	}
	d := mx - mn
	if l > 0.5 {
		s = d / (2 - mx - mn)
	} else {
		s = d / (mx + mn)
	}
	switch mx {
	case r:
		h = (g - b) / d
		if g < b {
			h += 6
		}
	case g:
		h = (b-r)/d + 2
	default:
		h = (r-g)/d + 4
	}
	return h / 6, s, l
}

func hslToColor(h, s, l float64, a uint8) Color {
	if s == 0 {
		v := uint8(math.Round(l * 255))
		return Color{v, v, v, a}
	}
	q := l + s - l*s
	if l < 0.5 {
		q = l * (1 + s)
	}
	p := 2*l - q
	f := func(t float64) uint8 {
		t = math.Mod(t+1, 1)
		var v float64
		switch {
		case t < 1.0/6:
			v = p + (q-p)*6*t
		case t < 0.5:
			v = q
		case t < 2.0/3:
			v = p + (q-p)*(2.0/3-t)*6
		default:
			v = p
		}
		return uint8(math.Round(math.Max(0, math.Min(1, v)) * 255))
	}
	return Color{f(h + 1.0/3), f(h), f(h - 1.0/3), a}
}

func clampF(x, a, b float64) float64 { return math.Max(a, math.Min(b, x)) }

// mapThemeColor re-colours one dark-palette colour for a theme (the same rules
// as look.js uses for the other apps).
func mapThemeColor(c Color, role int, t themeDef) Color {
	h, s, l := rgbToHSL(c)
	neutral := s < 0.28 || l < 0.1 || l > 0.94
	surface := neutral || (role == roleBg && l < 0.22 && t.base != "dark")
	switch t.base {
	case "light":
		if surface {
			switch role {
			case roleBg:
				l = clampF(0.94+(l-0.04)*0.45, 0.9, 1)
			case roleLine:
				l = clampF(0.95-l*0.5, 0.62, 0.9)
			default:
				if l < 0.25 {
					l = 0.98 // dark text on a bright button stays readable on the darker one
				} else {
					l = clampF(1-l*1.03, 0.06, 0.58)
				}
			}
			s = math.Min(s, 0.25)
		} else {
			l = math.Min(l, 0.44)
		}
	case "black":
		if surface && role == roleBg {
			if l < 0.16 {
				l *= 0.15
			} else {
				l *= 0.8
			}
			s = 0
		}
	case "contrast", "contrastl":
		dark := t.base == "contrast"
		if surface {
			s = 0
			switch role {
			case roleBg:
				if dark {
					l = map[bool]float64{true: 0, false: 0.1}[l < 0.2]
				} else {
					l = map[bool]float64{true: 1, false: 0.9}[l < 0.2]
				}
			case roleLine:
				l = map[bool]float64{true: 0.8, false: 0.15}[dark]
			default:
				l = map[bool]float64{true: 1, false: 0}[dark]
			}
		} else {
			s = 1
			if dark {
				l = clampF(l, 0.55, 0.68)
			} else {
				l = clampF(l, 0.22, 0.34)
			}
		}
	}
	if t.s >= 0 && surface {
		h = t.h / 360
		switch role {
		case roleBg:
			s = t.s
			if t.base == "dark" && t.s > 0 {
				l = math.Min(0.4, l*1.2+0.015) // a touch lighter, so the tint shows
			}
		case roleLine:
			s = t.s * 0.75
		default:
			s = t.s * 0.3
		}
	}
	if t.cvd && !neutral {
		deg := h * 360
		if deg >= 70 && deg < 175 {
			h, s = 205.0/360, math.Max(s, 0.6)
		} else if deg >= 330 || deg < 18 {
			h = 30.0 / 360
			l = clampF(l, 0.45, 0.62)
		}
	}
	return hslToColor(h, s, l, c.A)
}

func (a *App) applyTheme() {
	if a.settings.Theme < 0 || a.settings.Theme >= len(themes) {
		a.settings.Theme = 0
	}
	t := themes[a.settings.Theme]
	th = thDark
	copy(channelPalette, channelPaletteDark)
	switch a.settings.Theme {
	case 0:
	case 1: // light: hand-tuned
		th.Bg, th.Bg2, th.Rail, th.Panel = hexColor(0xF3F5F8), hexColor(0xEBEEF3), hexColor(0xE6EAF0), hexColor(0xFFFFFF)
		th.Card, th.Card2, th.Hover, th.Border, th.BorderHi = hexColor(0xFFFFFF), hexColor(0xF4F6F9), hexColor(0xE7ECF3), hexColor(0xD5DBE4), hexColor(0xB4BFCE)
		th.Text, th.Sub, th.Faint = hexColor(0x12161D), hexColor(0x4F5A6B), hexColor(0x8591A2)
		th.Good, th.Warn, th.Bad, th.Info = hexColor(0x15803D), hexColor(0xB45309), hexColor(0xDC2626), hexColor(0x2563EB)
		th.LCD, th.LCDOn, th.LCDGhost, th.Grid = hexColor(0xDDE8EA), hexColor(0x0B3B4A), hexColor(0xC9D6D9), hexColor(0xE1E6ED)
		th.Accent2 = hexColor(0x6D5DFC)
	case 2: // high contrast: hand-tuned
		th.Bg, th.Bg2, th.Rail, th.Panel = hexColor(0x000000), hexColor(0x000000), hexColor(0x000000), hexColor(0x000000)
		th.Card, th.Card2, th.Hover, th.Border, th.BorderHi = hexColor(0x000000), hexColor(0x0A0A0A), hexColor(0x1F1F1F), hexColor(0xFFFFFF), hexColor(0xFFD400)
		th.Text, th.Sub, th.Faint = hexColor(0xFFFFFF), hexColor(0xFFFFFF), hexColor(0xD8D8D8)
		th.Good, th.Warn, th.Bad, th.Info = hexColor(0x3DFF8C), hexColor(0xFFD400), hexColor(0xFF6B6B), hexColor(0x6CB8FF)
		th.LCD, th.LCDOn, th.LCDGhost, th.Grid = hexColor(0x000000), hexColor(0xFFFFFF), hexColor(0x161616), hexColor(0x5A5A5A)
		th.Accent2 = hexColor(0x00E5FF)
	default: // every other theme: the dark palette re-coloured by role
		bg := []*Color{&th.Bg, &th.Bg2, &th.Rail, &th.Panel, &th.Card, &th.Card2, &th.Hover, &th.LCD, &th.LCDGhost}
		line := []*Color{&th.Border, &th.BorderHi, &th.Grid}
		text := []*Color{&th.Text, &th.Sub, &th.Faint, &th.LCDOn}
		acc := []*Color{&th.Accent2, &th.Good, &th.Warn, &th.Bad, &th.Info}
		for role, set := range [][]*Color{bg, line, text, acc} {
			for _, c := range set {
				*c = mapThemeColor(*c, role, t)
			}
		}
		if t.cvd {
			th.Good, th.Warn, th.Bad = hexColor(0x56B4E9), hexColor(0xF0E442), hexColor(0xE69F00)
			if t.base == "light" {
				th.Good, th.Warn, th.Bad = hexColor(0x0072B2), hexColor(0x9A7B00), hexColor(0xC26A00)
			}
			copy(channelPalette, channelPaletteCVD)
		}
	}
	a.applyAccent()
	a.bg = nil // repaint the background in the new colours
	reduceMotion = a.settings.ReduceMotion
}

// themeIsLight: light themes need darker accents to read on white.
func themeIsLight(i int) bool {
	return i >= 0 && i < len(themes) && (themes[i].base == "light" || themes[i].base == "contrastl")
}
