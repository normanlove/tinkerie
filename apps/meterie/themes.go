package meterie

// themes.go: Meterie's colour themes (Dark, Light, High contrast, Colour-blind
// safe) and reduced motion, chosen on the Settings page. The same four themes
// as the rest of Tinkerie's Look & accessibility panel. Created by nrmluvdev.

var themeNames = []string{"Dark", "Light", "High contrast", "Colour-blind safe"}

// the original dark palette, kept so switching back restores it exactly
var thDark = th
var channelPaletteDark = append([]Color(nil), channelPalette...)

// Okabe-Ito colours: told apart with every common kind of colour blindness
var channelPaletteCVD = []Color{hexColor(0x56B4E9), hexColor(0xE69F00), hexColor(0xF0E442), hexColor(0xCC79A7), hexColor(0x009E73), hexColor(0xD55E00), hexColor(0x0072B2), hexColor(0xFFFFFF)}

// reduceMotion makes animations jump straight to where they end.
var reduceMotion bool

func (a *App) applyTheme() {
	th = thDark
	copy(channelPalette, channelPaletteDark)
	switch a.settings.Theme {
	case 1: // light
		th.Bg, th.Bg2, th.Rail, th.Panel = hexColor(0xF3F5F8), hexColor(0xEBEEF3), hexColor(0xE6EAF0), hexColor(0xFFFFFF)
		th.Card, th.Card2, th.Hover, th.Border, th.BorderHi = hexColor(0xFFFFFF), hexColor(0xF4F6F9), hexColor(0xE7ECF3), hexColor(0xD5DBE4), hexColor(0xB4BFCE)
		th.Text, th.Sub, th.Faint = hexColor(0x12161D), hexColor(0x4F5A6B), hexColor(0x8591A2)
		th.Good, th.Warn, th.Bad, th.Info = hexColor(0x15803D), hexColor(0xB45309), hexColor(0xDC2626), hexColor(0x2563EB)
		th.LCD, th.LCDOn, th.LCDGhost, th.Grid = hexColor(0xDDE8EA), hexColor(0x0B3B4A), hexColor(0xC9D6D9), hexColor(0xE1E6ED)
		th.Accent2 = hexColor(0x6D5DFC)
	case 2: // high contrast
		th.Bg, th.Bg2, th.Rail, th.Panel = hexColor(0x000000), hexColor(0x000000), hexColor(0x000000), hexColor(0x000000)
		th.Card, th.Card2, th.Hover, th.Border, th.BorderHi = hexColor(0x000000), hexColor(0x0A0A0A), hexColor(0x1F1F1F), hexColor(0xFFFFFF), hexColor(0xFFD400)
		th.Text, th.Sub, th.Faint = hexColor(0xFFFFFF), hexColor(0xFFFFFF), hexColor(0xD8D8D8)
		th.Good, th.Warn, th.Bad, th.Info = hexColor(0x3DFF8C), hexColor(0xFFD400), hexColor(0xFF6B6B), hexColor(0x6CB8FF)
		th.LCD, th.LCDOn, th.LCDGhost, th.Grid = hexColor(0x000000), hexColor(0xFFFFFF), hexColor(0x161616), hexColor(0x5A5A5A)
		th.Accent2 = hexColor(0x00E5FF)
	case 3: // colour-blind safe: no red/green pairs
		th.Good, th.Warn, th.Bad, th.Info = hexColor(0x56B4E9), hexColor(0xF0E442), hexColor(0xE69F00), hexColor(0x5AA9E6)
		copy(channelPalette, channelPaletteCVD)
	}
	a.applyAccent()
	a.bg = nil // repaint the background in the new colours
	reduceMotion = a.settings.ReduceMotion
}
