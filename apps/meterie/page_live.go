package meterie

import (
	"math"
	"time"
)

var graphWindows = []struct {
	label string
	secs  float64
}{{"10 s", 10}, {"1 min", 60}, {"5 min", 300}, {"30 min", 1800}, {"2 h", 7200}, {"All", -1}}

var statsWindows = []string{"Visible", "Selection", "Last 10 s", "All"}

func (a *App) pageLive(r Rect) {
	u := a.ui
	gap := u.Px(16)
	topH := clampi(int(float64(r.H)*0.47), u.Px(300), u.Px(420))
	top, rest := r.CutTop(topH)
	statsW := clampi(r.W/4, u.Px(290), u.Px(360))
	readR, statsR := top.CutLeft(top.W - statsW - gap)
	statsR.X += gap
	statsR.W -= gap
	ch := a.mainChannel()
	u.TeachZone("readout", readR)
	u.TeachZone("stats", statsR)
	a.DrawReadout(readR, ch)
	a.drawStatsCard(statsR, ch)

	strip := Rect{rest.X, rest.Y + gap, rest.W, u.Px(66)}
	u.TeachZone("strip", strip)
	a.drawChannelStrip(strip)
	gr := Rect{rest.X, strip.Bottom() + gap, rest.W, rest.Bottom() - strip.Bottom() - gap}
	u.TeachZone("graph", gr)
	a.drawGraphCard(gr)
	if a.settings.ShowWelcome && a.modal == "" && len(a.devices) == 0 && a.frameCount > 2 {
		a.modal = "welcome"
	}
}

func (a *App) statsRange(ch *Channel) (int64, int64) {
	now := time.Now().UnixNano()
	switch a.statsWin {
	case 0:
		return a.graph.Range(now)
	case 1:
		if a.graph.SelOn {
			s0, s1 := a.graph.Sel0, a.graph.Sel1
			if s1 < s0 {
				s0, s1 = s1, s0
			}
			return s0, s1
		}
		return a.graph.Range(now)
	case 2:
		return now - 10e9, now
	}
	return math.MinInt64, math.MaxInt64
}

func (a *App) drawStatsCard(r Rect, ch *Channel) {
	u := a.ui
	in := u.Card(r, "Statistics")
	u.Segmented("stats#win", Rect{in.X, in.Y, in.W, u.Px(30)}, statsWindows, &a.statsWin)
	y := in.Y + u.Px(42)
	if ch == nil || len(ch.Samples) == 0 {
		u.Wrap("Statistics appear here once readings arrive. Pick a window above: what's visible on the graph, a Shift+drag selection, the last 10 seconds, or everything.", in.X, y, in.W, u.F(12.5), th.Faint)
		return
	}
	t0, t1 := a.statsRange(ch)
	st := ComputeStats(ch.Samples, t0, t1, ch.CurrentUnit())
	unit := ch.UnitLabel()
	if ch.RelOn {
		st.Min -= ch.RelValue
		st.Max -= ch.RelValue
		st.Mean -= ch.RelValue
		st.First -= ch.RelValue
		st.Last -= ch.RelValue
	}
	rows := []struct{ k, v string }{
		{"Minimum", FormatSI(st.Min, unit, 5)},
		{"Maximum", FormatSI(st.Max, unit, 5)},
		{"Average", FormatSI(st.Mean, unit, 5)},
		{"Peak-to-peak", FormatSI(st.PkPk, unit, 4)},
		{"Std deviation", FormatSI(st.Std, unit, 3)},
		{"Samples", itoa(st.N) + "  over " + fmtDur(time.Duration(st.T1-st.T0))},
	}
	if st.N == 0 {
		rows = rows[:0]
		u.Text("No samples in this window", Rect{in.X, y, in.W, u.Px(20)}, u.F(12.5), th.Faint, AlignLeft)
	}
	switch ch.Last.Q {
	case QCurrent:
		rows = append(rows, struct{ k, v string }{"Charge", FormatSI(st.Integral*1000, "mAh", 4)})
	case QPower:
		rows = append(rows, struct{ k, v string }{"Energy", FormatSI(st.Integral, "Wh", 4)})
	}
	rh := u.Px(25)
	for i, rw := range rows {
		ry := y + i*rh
		if ry+rh > in.Bottom()-u.Px(44) {
			break
		}
		u.Text(rw.k, Rect{in.X, ry, in.W / 2, rh}, u.F(12.5), th.Sub, AlignLeft)
		u.Text(rw.v, Rect{in.X + in.W/2 - u.Px(20), ry, in.W/2 + u.Px(20), rh}, u.FB(13), th.Text, AlignRight)
		if i < len(rows)-1 {
			u.C.HLine(in.X, in.Right(), ry+rh-1, th.Border.WithA(0.5))
		}
	}
	// actions
	by := in.Bottom() - u.Px(34)
	bw := (in.W - u.Px(16)) / 3
	relLabel := "REL Δ"
	if ch.RelOn {
		relLabel = "REL off"
	}
	if u.Button("st#rel", Rect{in.X, by, bw, u.Px(34)}, "", relLabel, map[bool]int{true: BtnPrimary, false: BtnSecondary}[ch.RelOn]) {
		if ch.RelOn {
			ch.RelOn = false
		} else if ch.HasLast && !ch.Last.IsOL() {
			ch.RelOn, ch.RelValue = true, ch.Last.Value
			u.Toast("Relative mode: zeroed at "+FormatSI(ch.RelValue, ch.Last.Unit, 5), 0)
		}
	}
	if u.Button("st#copy", Rect{in.X + bw + u.Px(8), by, bw, u.Px(34)}, icCopy, "Copy", BtnSecondary) {
		if ch.HasLast {
			platformClipboardSet(FormatSI(ch.DisplayValue(ch.Last.Value), ch.Last.Unit, 6))
			u.Toast("Copied "+FormatSI(ch.DisplayValue(ch.Last.Value), ch.Last.Unit, 6), 1)
		}
	}
	if u.Button("st#clear", Rect{in.X + 2*(bw+u.Px(8)), by, bw, u.Px(34)}, icDelete, "Clear", BtnSecondary) {
		ch.Samples = ch.Samples[:0]
		if ch.Math != nil {
			ch.Math.Reset()
		}
		u.Toast("Cleared "+ch.Name+"'s history", 0)
	}
}

func (a *App) drawChannelStrip(r Rect) {
	u := a.ui
	c := u.C
	x := r.X
	main := a.mainChannel()
	manual := main != nil && main.Kind == KindManual
	stripR := r
	if manual {
		stripR.W = r.W - u.Px(470)
	}
	u.PushClip(stripR)
	for _, ch := range a.channels {
		w := u.Px(232)
		cr := Rect{x, r.Y, w, r.H}
		sel := main == ch
		id := "chan#" + itoa(ch.ID)
		h, clicked := u.interact(id, cr)
		hv := u.Anim(id+"#h", b2f(h), 14)
		sv := u.Anim(id+"#s", b2f(sel), 14)
		col := chanColor(ch.Color)
		c.FillRoundRect(cr, u.Pf(12), lerpColor(th.Card, th.Card2, hv))
		c.StrokeRoundRect(cr, u.Pf(12), u.Pf(1), lerpColor(th.Border, col.WithA(0.8), sv))
		c.FillRoundRect(Rect{cr.X + u.Px(10), cr.Y + u.Px(12), u.Px(4), cr.H - u.Px(24)}, u.Pf(2), col)
		u.Text(ch.Name, Rect{cr.X + u.Px(24), cr.Y + u.Px(9), w - u.Px(70), u.Px(18)}, u.FB(12.5), th.Text, AlignLeft)
		val := "—"
		if ch.HasLast {
			if ch.Last.IsOL() {
				val = "OL " + ch.Last.Prefix + ch.Last.Unit
			} else {
				val = FormatSI(ch.DisplayValue(ch.Last.Value), ch.Last.Unit, 5)
			}
			if ch.RelOn {
				val = "Δ " + val
			}
		}
		u.Text(val, Rect{cr.X + u.Px(24), cr.Y + u.Px(29), w - u.Px(70), u.Px(26)}, u.FH(17), col.Lighten(0.2), AlignLeft)
		// visibility toggle
		er := Rect{cr.Right() - u.Px(40), cr.Y + u.Px(17), u.Px(30), u.Px(30)}
		eye := icView
		if !ch.Visible {
			eye = icRemove
		}
		if u.IconButton(id+"#eye", er, eye, "Show/hide on graph", ch.Visible) {
			ch.Visible = !ch.Visible
		} else if clicked {
			a.mainChan = ch.ID
		}
		x += w + u.Px(10)
	}
	if len(a.channels) == 0 {
		u.Text("Channels appear here — every meter, math channel and camera feed gets its own colour on the shared timeline.", r, u.F(13), th.Faint, AlignLeft)
	}
	u.PopClip()
	if manual {
		a.drawManualBar(Rect{r.Right() - u.Px(460), r.Y, u.Px(460), r.H}, main)
	}
}

func (a *App) drawManualBar(r Rect, ch *Channel) {
	u := a.ui
	u.C.FillRoundRect(r, u.Pf(12), th.Card)
	u.C.StrokeRoundRect(r, u.Pf(12), u.Pf(1), th.Accent.WithA(0.5))
	in := r.InsetXY(u.Px(12), u.Px(14))
	opts := make([]string, len(manualUnits))
	for i, m := range manualUnits {
		opts[i] = m.label
	}
	vr := Rect{in.X, in.Y, u.Px(150), in.H}
	_, enter := u.TextField("manual#v", vr, &a.manual.value, "Value e.g. 4k7")
	u.Dropdown("manual#u", Rect{vr.Right() + u.Px(8), in.Y, u.Px(110), in.H}, opts, &a.manual.unit)
	nr := Rect{vr.Right() + u.Px(126), in.Y, in.W - u.Px(126) - vr.W - u.Px(80), in.H}
	_, enter2 := u.TextField("manual#n", nr, &a.manual.note, "Note")
	if u.Button("manual#add", Rect{in.Right() - u.Px(72), in.Y, u.Px(72), in.H}, icAdd, "Add", BtnPrimary) || enter || enter2 {
		if a.submitManual(ch) {
			u.focus = "manual#v"
		}
	}
}

func (a *App) drawGraphCard(r Rect) {
	u := a.ui
	c := u.C
	c.Shadow(r, u.Pf(14), u.Pf(16), u.Pf(4), RGBA(0, 0, 0, 90))
	c.FillRoundRect(r, u.Pf(14), th.Card)
	c.StrokeRoundRect(r, u.Pf(14), u.Pf(1), th.Border)
	tb := Rect{r.X + u.Px(16), r.Y + u.Px(12), r.W - u.Px(32), u.Px(32)}
	g := a.graph
	sel := -1
	for i, w := range graphWindows {
		if g.Follow && (w.secs == g.Window || (w.secs < 0 && g.Window > 1e8)) {
			sel = i
		}
	}
	segW := min(u.Px(420), tb.W/2)
	selCopy := sel
	if selCopy < 0 {
		selCopy = 0
	}
	if u.Segmented("graph#win", Rect{tb.X, tb.Y, segW, tb.H}, labelsOf(), &selCopy) || (sel < 0 && false) {
		w := graphWindows[selCopy].secs
		if w < 0 {
			w = 1e9 // effectively everything
			if ch := a.mainChannel(); ch != nil && len(ch.Samples) > 0 {
				w = math.Max(10, float64(time.Now().UnixNano()-ch.Samples[0].T)/1e9*1.02)
			}
		}
		g.Window, g.Follow, g.Paused = w, true, false
	}
	x := tb.Right()
	ib := func(id, icon, tip string, on bool) bool {
		x -= tb.H
		ok := u.IconButton(id, Rect{x, tb.Y, tb.H, tb.H}, icon, tip, on)
		x -= u.Px(6)
		return ok
	}
	if ib("graph#png", icPhoto, "Export graph as PNG", false) {
		a.exportGraphPNG()
	}
	if ib("graph#marker", icFlag, "Add marker now (Ctrl+M)\nRight-click the graph to mark any moment", false) {
		a.openMarkerDialog(time.Now().UnixNano())
	}
	if ib("graph#fit", icFit, "Back to live view with auto-scale (or double-click the graph)", false) {
		g.Follow, g.AutoY, g.Paused, g.SelOn = true, true, false, false
	}
	if ib("graph#auto", icSync, "Auto-scale the Y axis", g.AutoY) {
		g.AutoY = !g.AutoY
	}
	pIcon, pTip := icPause, "Pause the graph (Space)"
	if g.Paused {
		pIcon, pTip = icPlay, "Resume (Space)"
	}
	if ib("graph#pause", pIcon, pTip, g.Paused) {
		g.Paused = !g.Paused
		g.pauseAt = time.Now().UnixNano()
	}
	hint := "Wheel zoom · drag pan · Shift+drag select · Ctrl+wheel Y zoom · right-click marker · double-click live"
	hr := Rect{tb.X + segW + u.Px(16), tb.Y, x - tb.X - segW - u.Px(24), tb.H}
	if hr.W > u.Px(120) {
		u.Text(hint, hr, u.F(11.5), th.Faint, AlignLeft)
	}
	if !g.Follow {
		u.Chip(tb.X+segW+u.Px(16), tb.Y-u.Px(16), "HISTORY VIEW", th.Warn, th.Warn.WithA(0.15))
	}
	gr := Rect{r.X + u.Px(6), tb.Bottom() + u.Px(6), r.W - u.Px(12), r.H - tb.H - u.Px(24)}
	a.DrawGraph(gr, g, a.channels, true)
}

func labelsOf() []string {
	out := make([]string, len(graphWindows))
	for i, w := range graphWindows {
		out[i] = w.label
	}
	return out
}
