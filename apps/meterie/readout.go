package meterie

// readout.go: the big "premium bench meter" readout — seven-segment digits
// with ghost segments and glow, LCD annunciators mirrored from the meter,
// unit, and an analog-style bar graph.

import (
	"math"
	"strings"
	"time"
)

// segPolys returns the 7 segment polygons for one digit cell.
func segPolys(x, y, w, h, t, slant float64) [7][]Pt {
	g := t * 0.28 // gap between segments
	xl, xr := t/2, w-t/2
	yt, ym, yb := t/2, h/2, h-t/2
	sk := func(px, py float64) Pt { return Pt{x + px + slant*(1-py/h), y + py} }
	hseg := func(x0, x1, yy float64) []Pt {
		return []Pt{sk(x0, yy), sk(x0+t/2, yy-t/2), sk(x1-t/2, yy-t/2), sk(x1, yy), sk(x1-t/2, yy+t/2), sk(x0+t/2, yy+t/2)}
	}
	vseg := func(xx, y0, y1 float64) []Pt {
		return []Pt{sk(xx, y0), sk(xx+t/2, y0+t/2), sk(xx+t/2, y1-t/2), sk(xx, y1), sk(xx-t/2, y1-t/2), sk(xx-t/2, y0+t/2)}
	}
	return [7][]Pt{
		hseg(xl+g, xr-g, yt),
		vseg(xr, yt+g, ym-g),
		vseg(xr, ym+g, yb-g),
		hseg(xl+g, xr-g, yb),
		vseg(xl, ym+g, yb-g),
		vseg(xl, yt+g, ym-g),
		hseg(xl+g, xr-g, ym),
	}
}

// lcdCells converts a display string into cells (char + decimal point flag), right aligned to n cells.
func lcdCells(disp string, n int) (cells []byte, dps []bool, neg bool) {
	disp = strings.TrimSpace(disp)
	if strings.HasPrefix(disp, "-") {
		neg = true
		disp = disp[1:]
	}
	if disp == "OL" {
		disp = "0.L"
	}
	for _, c := range []byte(disp) {
		if c == '.' {
			if len(dps) > 0 {
				dps[len(dps)-1] = true
			}
			continue
		}
		cells = append(cells, c)
		dps = append(dps, false)
	}
	for len(cells) < n {
		cells = append([]byte{' '}, cells...)
		dps = append([]bool{false}, dps...)
	}
	if len(cells) > n {
		cells, dps = cells[len(cells)-n:], dps[len(dps)-n:]
	}
	return
}

func countsDigits(counts int) int {
	switch {
	case counts <= 0:
		return 4
	case counts <= 9999:
		return 4
	case counts <= 99999:
		return 5
	default:
		return 6
	}
}

// DrawLCDDigits draws a seven-segment number into r. Returns the x where digits end.
func (u *UI) DrawLCDDigits(id string, r Rect, disp string, nDigits int, on, ghost Color, glow bool) int {
	cells, dps, neg := lcdCells(disp, nDigits)
	h := float64(r.H)
	t := h * 0.115
	w := h * 0.52
	gap := h * 0.17
	slant := h * 0.085
	signW := w * 0.55
	total := signW + gap*0.6 + float64(nDigits)*w + float64(nDigits-1)*gap
	x := float64(r.X) + (float64(r.W)-total)/2
	if x < float64(r.X) {
		x = float64(r.X)
	}
	y := float64(r.Y)
	// minus sign
	sy := y + h/2
	sx0, sx1 := x+slant*0.5, x+signW+slant*0.5
	sign := []Pt{{sx0, sy}, {sx0 + t/2, sy - t/2}, {sx1 - t/2, sy - t/2}, {sx1, sy}, {sx1 - t/2, sy + t/2}, {sx0 + t/2, sy + t/2}}
	sa := u.Anim(id+"#neg", b2f(neg), 20)
	u.C.FillConvex(sign, ghost, 0, 0)
	if sa > 0.02 {
		gl := 0.0
		if glow {
			gl = t * 0.9
		}
		u.C.FillConvex(sign, on.WithA(sa), gl, 0.22*sa)
	}
	x += signW + gap*0.6
	for i := 0; i < nDigits; i++ {
		cx := x + float64(i)*(w+gap)
		polys := segPolys(cx, y, w, h, t, slant)
		mask := charSegs[cells[i]]
		for s := 0; s < 7; s++ {
			lit := mask&(1<<uint(s)) != 0
			a := u.Anim(id+"#"+itoa(i)+"_"+itoa(s), b2f(lit), 24)
			u.C.FillConvex(polys[s], ghost, 0, 0)
			if a > 0.02 {
				gl := 0.0
				if glow {
					gl = t * 0.9
				}
				u.C.FillConvex(polys[s], on.WithA(a), gl, 0.2*a)
			}
		}
		// decimal point (after this digit)
		if i < nDigits-1 || dps[i] {
			dx := cx + w + gap*0.5 - t*0.15
			dy := y + h - t*0.55
			da := u.Anim(id+"#dp"+itoa(i), b2f(dps[i]), 20)
			u.C.FillCircle(dx, dy, t*0.55, ghost)
			if da > 0.02 {
				u.C.FillCircle(dx, dy, t*0.55, on.WithA(da))
				if glow {
					u.C.RadialGlow(dx, dy, t*1.6, on, 0.18*da)
				}
			}
		}
	}
	return int(x + float64(nDigits)*(w+gap))
}

// annunciator list in LCD order
var annunciators = []struct {
	label string
	flag  Flags
}{
	{"AUTO", FAuto}, {"MANUAL", FManual}, {"AC", FAC}, {"DC", FDC}, {"HOLD", FHold}, {"REL Δ", FRel},
	{"MIN", FMin}, {"MAX", FMax}, {"PEAK", FPeakMax | FPeakMin},
}

// DrawReadout renders the main readout card for a channel.
func (a *App) DrawReadout(r Rect, ch *Channel) {
	u := a.ui
	alarm := a.channelAlarmActive(ch)
	inner := u.Card(r, "")
	_ = inner
	// header row
	hdr := Rect{r.X + u.Px(22), r.Y + u.Px(16), r.W - u.Px(44), u.Px(26)}
	var rd Reading
	has := ch != nil && ch.HasLast
	if has {
		rd = ch.Last
	}
	mode := "NO METER"
	if has {
		mode = rd.ModeLabel()
	} else if ch != nil {
		mode = "WAITING FOR DATA"
	}
	live := ch != nil && has && time.Since(rd.T) < 3*time.Second
	pulse := 0.5 + 0.5*math.Sin(float64(time.Now().UnixMilli())/1000*2*math.Pi*0.8)
	if live {
		u.Ambient = true
	}
	dotCol := th.Faint
	if live {
		dotCol = th.Good
	}
	if alarm {
		dotCol = th.Bad
	}
	u.C.FillCircle(float64(hdr.X+u.Px(5)), float64(hdr.CY()), u.Pf(4.5), dotCol)
	if live {
		u.C.FillCircle(float64(hdr.X+u.Px(5)), float64(hdr.CY()), u.Pf(4.5+5*pulse), dotCol.WithA(0.25*(1-pulse)))
	}
	u.Text(mode, Rect{hdr.X + u.Px(18), hdr.Y, hdr.W / 2, hdr.H}, u.FH(14), th.Text, AlignLeft)
	name := "Connect a meter on the Meters page"
	if ch != nil {
		name = ch.Name
		if ch.Source != "" {
			name += "  ·  " + ch.Source
		}
	}
	u.Text(name, Rect{hdr.X + hdr.W/2, hdr.Y, hdr.W / 2, hdr.H}, u.F(12.5), th.Sub, AlignRight)

	// LCD panel
	lcd := Rect{r.X + u.Px(18), hdr.Bottom() + u.Px(10), r.W - u.Px(36), r.H - u.Px(26) - hdr.H - u.Px(18)}
	onCol := th.LCDOn
	if has && rd.Flags.Has(FHold) {
		onCol = hexColor(0xFFD27A)
	}
	if alarm {
		f := 0.6 + 0.4*pulse
		onCol = lerpColor(hexColor(0xFF8080), hexColor(0xFFC2C2), f)
		u.Ambient = true
	}
	if ch != nil {
		cc := chanColor(ch.Color)
		if !alarm && !(has && rd.Flags.Has(FHold)) {
			onCol = lerpColor(cc, hexColor(0xFFFFFF), 0.28)
		}
	}
	ghost := lerpColor(th.LCD, onCol, 0.055)
	u.C.FillRoundRect(lcd, u.Pf(12), th.LCD)
	u.C.VGradient(Rect{lcd.X + u.Px(2), lcd.Y + u.Px(2), lcd.W - u.Px(4), lcd.H / 2}, RGBA(255, 255, 255, 6), RGBA(255, 255, 255, 0))
	u.C.StrokeRoundRect(lcd, u.Pf(12), u.Pf(1), lerpColor(th.Border, onCol, 0.12))
	if alarm {
		u.C.StrokeRoundRect(lcd.Inset(-u.Px(2)), u.Pf(14), u.Pf(2), th.Bad.WithA(0.4+0.4*pulse))
	}

	// annunciators
	ay := lcd.Y + u.Px(12)
	ax := lcd.X + u.Px(18)
	af := u.FB(11)
	for _, an := range annunciators {
		lit := has && rd.Flags&an.flag != 0
		if an.flag == FManual && has && !rd.Flags.Has(FManual) {
			continue
		}
		if an.flag == FAuto && has && rd.Flags.Has(FManual) {
			continue
		}
		col := lerpColor(ghost.Lighten(0.04), onCol, u.Anim("ann"+an.label, b2f(lit), 14))
		w := u.T.Measure(an.label, af)
		u.T.Draw(an.label, ax, ay, af, col)
		ax += w + u.Px(14)
	}
	// right-side icons: HV bolt, beep, diode, battery
	rx := lcd.Right() - u.Px(18)
	batLit := has && rd.Flags.Has(FLowBat)
	bt := u.Anim("ann_bat", b2f(batLit), 12)
	bw, bh := u.Px(22), u.Px(11)
	bx, by := rx-bw, ay+u.Px(1)
	bcol := lerpColor(ghost.Lighten(0.04), th.Warn, bt)
	u.C.StrokeRoundRect(Rect{bx, by, bw - u.Px(3), bh}, u.Pf(2), u.Pf(1.4), bcol)
	u.C.FillRect(Rect{bx + bw - u.Px(3), by + bh/4, u.Px(2), bh / 2}, bcol)
	u.C.FillRect(Rect{bx + u.Px(2), by + u.Px(2), u.Px(4), bh - u.Px(4)}, bcol)
	rx = bx - u.Px(12)
	for _, ic := range []struct {
		glyph string
		flag  Flags
		col   Color
	}{{icBolt, FHV, th.Bad}, {icVolume, FBeep, onCol}} {
		lit := has && rd.Flags.Has(ic.flag)
		col := lerpColor(ghost.Lighten(0.04), ic.col, u.Anim("ann_"+ic.glyph, b2f(lit), 14))
		w := u.T.Measure(ic.glyph, u.FI(13))
		u.T.Draw(ic.glyph, rx-w, ay-u.Px(1), u.FI(13), col)
		rx -= w + u.Px(12)
	}
	// diode symbol (drawn)
	{
		lit := has && rd.Flags.Has(FDiode)
		col := lerpColor(ghost.Lighten(0.04), onCol, u.Anim("ann_diode", b2f(lit), 14))
		s := u.Pf(1)
		dx, dy := float64(rx)-14*s, float64(ay)+6*s
		u.C.FillConvex([]Pt{{dx, dy - 5*s}, {dx, dy + 5*s}, {dx + 8*s, dy}}, col, 0, 0)
		u.C.FillRect(Rect{int(dx + 8*s), int(dy - 5*s), max(1, int(1.6*s)), int(10 * s)}, col)
		u.C.Line(dx-4*s, dy, dx+12*s, dy, 1.2*s, col)
	}

	// digits + unit
	nd := 4
	if has {
		nd = countsDigits(rd.Counts)
	}
	disp := "    "
	unitTxt := ""
	if has {
		disp = rd.Display
		unitTxt = rd.Prefix + rd.Unit
		if ch.RelOn && !rd.IsOL() {
			v := ch.DisplayValue(rd.Value)
			num, pu := SplitSI(v, rd.Unit, nd)
			disp, unitTxt = num, pu
		}
		if rd.Q == QHFE {
			unitTxt = "hFE"
		}
	}
	digitsArea := Rect{lcd.X + u.Px(18), ay + u.Px(26), lcd.W - u.Px(36), lcd.H - u.Px(26) - u.Px(80)}
	unitW := u.Px(130)
	dh := min(digitsArea.H, int(float64(digitsArea.W-unitW)/(float64(nd)*0.69+0.45)))
	if dh < u.Px(30) {
		dh = u.Px(30)
	}
	dr := Rect{digitsArea.X, digitsArea.Y + (digitsArea.H-dh)/2, digitsArea.W - unitW, dh}
	endX := u.DrawLCDDigits("main", dr, disp, nd, onCol, ghost, true)
	uf := u.FH(math.Max(20, float64(dh)/u.S*0.42))
	ux := min(endX+u.Px(14), dr.Right()+u.Px(10))
	uy := dr.Bottom() - u.T.LineHeight(uf) + u.Px(4)
	u.T.Draw(unitTxt, ux, uy, uf, onCol)
	if has && (rd.Q == QVoltage || rd.Q == QCurrent) {
		acdc := ""
		switch {
		case rd.Flags.Has(FAC) && rd.Flags.Has(FDC):
			acdc = "AC+DC"
		case rd.Flags.Has(FAC):
			acdc = "AC"
		case rd.Flags.Has(FDC):
			acdc = "DC"
		}
		u.T.Draw(acdc, ux+u.Px(2), dr.Y+u.Px(4), u.FB(13), onCol.WithA(0.75))
	}

	// bar graph
	barR := Rect{lcd.X + u.Px(22), lcd.Bottom() - u.Px(58), lcd.W - u.Px(44), u.Px(14)}
	frac := 0.0
	if has && !rd.IsOL() && rd.Counts > 0 {
		cnt := 0.0
		for _, c := range rd.Display {
			if c >= '0' && c <= '9' {
				cnt = cnt*10 + float64(c-'0')
			}
		}
		frac = clamp01(cnt / float64(rd.Counts))
	} else if has && rd.IsOL() {
		frac = 1
	}
	if ch != nil {
		k := 1 - math.Exp(-10*u.dt)
		ch.barAnim += (frac - ch.barAnim) * k
		if math.Abs(ch.barAnim-frac) > 0.002 {
			u.Animating = true
		}
		frac = ch.barAnim
	}
	nseg := 50
	sw := float64(barR.W) / float64(nseg)
	for i := 0; i < nseg; i++ {
		x0 := float64(barR.X) + float64(i)*sw
		segOn := float64(i)/float64(nseg) < frac
		col := ghost
		if segOn {
			col = onCol
		}
		hgt := float64(barR.H)
		if i%5 != 0 {
			hgt *= 0.72
		}
		u.C.FillRoundRectF(x0+1, float64(barR.Bottom())-hgt, sw-2.5, hgt, u.Pf(1.5), col)
	}
	// scale labels
	if has && rd.Counts > 0 {
		lf := u.F(10.5)
		for i := 0; i <= 5; i++ {
			lx := barR.X + barR.W*i/5
			lbl := itoa(rd.Counts * i / 5)
			if rd.Counts >= 10000 {
				lbl = itoa(rd.Counts*i/5/1000) + "k"
			}
			w := u.T.Measure(lbl, lf)
			u.T.Draw(lbl, lx-w/2, barR.Bottom()+u.Px(4), lf, ghost.Lighten(0.25))
		}
	}
	// footer line: secondary info
	foot := Rect{lcd.X + u.Px(18), lcd.Bottom() - u.Px(26), lcd.W - u.Px(36), u.Px(20)}
	_ = foot
	info := ""
	if has {
		if ch.RelOn {
			info = "Δ relative to " + FormatSI(ch.RelValue, rd.Unit, 5)
		} else if !rd.IsOL() {
			info = "= " + FormatSI(rd.Value, rd.Unit, 6)
		} else {
			info = "Overload — the value is above the selected range"
		}
		age := time.Since(rd.T)
		if age > 3*time.Second {
			info += "   ·   last update " + fmtDur(age) + " ago"
		}
	}
	u.Text(info, Rect{foot.X, foot.Y + u.Px(4), foot.W, foot.H}, u.F(12), ghost.Lighten(0.35), AlignLeft)
}
