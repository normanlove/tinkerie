package meterie

// graph.go: the live multi-channel timeline graph.

import (
	"math"
	"sort"
	"strconv"
	"time"
)

// Marker is an annotation on the shared timeline.
type Marker struct {
	T     int64  `json:"t"`
	Text  string `json:"text"`
	Color int    `json:"color"`
}

// GraphView holds zoom / pan state.
type GraphView struct {
	Follow   bool
	Window   float64 // seconds shown when following
	T0, T1   int64   // explicit range when not following
	AutoY    bool
	YMin     float64
	YMax     float64
	SelOn    bool
	Sel0     int64
	Sel1     int64
	Paused   bool
	pauseAt  int64
	drag     int // 0 none, 1 pan, 2 select
	dragX    int
	dragY    int
	dragT0   int64
	dragT1   int64
	dragY0   float64
	dragY1   float64
	hoverT   int64
	hoverOn  bool
	lastAuto [2][2]float64
}

func NewGraphView() *GraphView { return &GraphView{Follow: true, Window: 60, AutoY: true} }

func (g *GraphView) Range(now int64) (int64, int64) {
	if g.Follow {
		end := now
		if g.Paused {
			end = g.pauseAt
		}
		return end - int64(g.Window*1e9), end
	}
	return g.T0, g.T1
}

// niceStep returns a 1-2-5 step for a span divided into ~n ticks.
func niceStep(span float64, n int) float64 {
	if span <= 0 || math.IsNaN(span) || math.IsInf(span, 0) {
		return 1
	}
	raw := span / float64(n)
	mag := math.Pow(10, math.Floor(math.Log10(raw)))
	for _, m := range []float64{1, 2, 2.5, 5, 10} {
		if raw <= m*mag {
			return m * mag
		}
	}
	return 10 * mag
}

var timeSteps = []float64{0.1, 0.2, 0.5, 1, 2, 5, 10, 15, 30, 60, 120, 300, 600, 900, 1800, 3600, 7200, 14400, 21600, 43200, 86400}

func timeStep(span float64, n int) float64 {
	raw := span / float64(n)
	for _, s := range timeSteps {
		if s >= raw {
			return s
		}
	}
	return 86400
}

type axisGroup struct {
	key      string
	unit     string
	min, max float64
	chans    []*Channel
}

type colAgg struct {
	min, max, sum float64
	n             int
	last          float64
}

// DrawGraph draws the graph into r and handles interaction.
func (a *App) DrawGraph(r Rect, g *GraphView, chans []*Channel, interactive bool) {
	u := a.ui
	c := u.C
	now := time.Now().UnixNano()
	t0, t1 := g.Range(now)
	if t1 <= t0 {
		t1 = t0 + 1e9
	}
	// layout
	fAx := u.F(11)
	plot := Rect{r.X + u.Px(64), r.Y + u.Px(14), r.W - u.Px(64) - u.Px(64), r.H - u.Px(14) - u.Px(30)}
	if plot.W < 50 || plot.H < 40 {
		return
	}
	c.FillRoundRect(plot.Inset(-u.Px(2)), u.Pf(8), th.Bg2)

	// visible channels & unit groups
	var vis []*Channel
	for _, ch := range chans {
		if ch.Visible && len(ch.Samples) > 0 {
			vis = append(vis, ch)
		}
	}
	groups := []*axisGroup{}
	gidx := map[string]int{}
	for _, ch := range vis {
		uk := ch.Units[ch.CurrentUnit()]
		key := uk.Unit
		if uk.AC {
			key += "~"
		}
		i, ok := gidx[key]
		if !ok {
			i = len(groups)
			gidx[key] = i
			groups = append(groups, &axisGroup{key: key, unit: uk.Unit, min: math.Inf(1), max: math.Inf(-1)})
		}
		groups[i].chans = append(groups[i].chans, ch)
	}
	// aggregate per pixel column per channel
	cols := plot.W
	type series struct {
		ch  *Channel
		agg []colAgg
		gi  int
		u   uint8
	}
	var ss []series
	for gi, grp := range groups {
		for _, ch := range grp.chans {
			cu := ch.CurrentUnit()
			agg := make([]colAgg, cols)
			// find start index by binary search
			i0 := sort.Search(len(ch.Samples), func(i int) bool { return ch.Samples[i].T >= t0 })
			if i0 > 0 {
				i0--
			}
			for i := i0; i < len(ch.Samples); i++ {
				s := ch.Samples[i]
				if s.T > t1 {
					// include one point past the edge for continuity
					break
				}
				if s.U != cu || math.IsNaN(s.V) {
					continue
				}
				x := int(float64(s.T-t0) / float64(t1-t0) * float64(cols))
				if x < 0 || x >= cols {
					continue
				}
				v := ch.DisplayValue(s.V)
				ag := &agg[x]
				if ag.n == 0 {
					ag.min, ag.max = v, v
				} else {
					ag.min, ag.max = math.Min(ag.min, v), math.Max(ag.max, v)
				}
				ag.sum += v
				ag.n++
				ag.last = v
				grp.min = math.Min(grp.min, v)
				grp.max = math.Max(grp.max, v)
			}
			ss = append(ss, series{ch, agg, gi, cu})
		}
	}
	// y ranges
	yr := func(gi int) (float64, float64) {
		grp := groups[gi]
		lo, hi := grp.min, grp.max
		if math.IsInf(lo, 0) {
			lo, hi = 0, 1
		}
		if gi == 0 && !g.AutoY {
			return g.YMin, g.YMax
		}
		span := hi - lo
		if span < math.Abs(hi)*1e-4+1e-12 {
			span = math.Max(math.Abs(hi)*0.02, 1e-9)
			lo -= span / 2
			hi += span / 2
		} else {
			lo -= span * 0.08
			hi += span * 0.08
		}
		// smooth autoscale
		if gi < 2 {
			k := 1 - math.Exp(-8*u.dt)
			p := &g.lastAuto[gi]
			if p[0] == 0 && p[1] == 0 || math.IsNaN(p[0]) {
				p[0], p[1] = lo, hi
			}
			p[0] += (lo - p[0]) * k
			p[1] += (hi - p[1]) * k
			if math.Abs(p[0]-lo) > (hi-lo)*0.002 || math.Abs(p[1]-hi) > (hi-lo)*0.002 {
				u.Animating = true
			}
			return p[0], p[1]
		}
		return lo, hi
	}
	ranges := make([][2]float64, len(groups))
	for i := range groups {
		lo, hi := yr(i)
		ranges[i] = [2]float64{lo, hi}
	}
	if len(groups) > 0 && g.AutoY {
		g.YMin, g.YMax = ranges[0][0], ranges[0][1]
	}
	yOf := func(gi int, v float64) float64 {
		lo, hi := ranges[gi][0], ranges[gi][1]
		return float64(plot.Bottom()) - (v-lo)/(hi-lo)*float64(plot.H)
	}
	xOf := func(t int64) float64 { return float64(plot.X) + float64(t-t0)/float64(t1-t0)*float64(plot.W) }
	tOf := func(x int) int64 { return t0 + int64(float64(x-plot.X)/float64(plot.W)*float64(t1-t0)) }

	// grid + y labels (left axis = group 0, right axis = group 1)
	if len(groups) > 0 {
		lo, hi := ranges[0][0], ranges[0][1]
		step := niceStep(hi-lo, max(2, plot.H/u.Px(48)))
		for v := math.Ceil(lo/step) * step; v <= hi; v += step {
			y := int(yOf(0, v))
			c.HLine(plot.X, plot.Right(), y, th.Grid)
			lbl := FormatSI(v, groups[0].unit, 4)
			w := u.T.Measure(lbl, fAx)
			u.T.Draw(lbl, plot.X-w-u.Px(8), y-u.T.LineHeight(fAx)/2, fAx, th.Sub)
		}
		if lo < 0 && hi > 0 {
			c.HLine(plot.X, plot.Right(), int(yOf(0, 0)), th.BorderHi)
		}
	} else {
		for i := 1; i < 5; i++ {
			c.HLine(plot.X, plot.Right(), plot.Y+plot.H*i/5, th.Grid)
		}
	}
	if len(groups) > 1 {
		lo, hi := ranges[1][0], ranges[1][1]
		step := niceStep(hi-lo, max(2, plot.H/u.Px(48)))
		col := chanColor(groups[1].chans[0].Color)
		for v := math.Ceil(lo/step) * step; v <= hi; v += step {
			y := int(yOf(1, v))
			lbl := FormatSI(v, groups[1].unit, 4)
			u.T.Draw(lbl, plot.Right()+u.Px(8), y-u.T.LineHeight(fAx)/2, fAx, col.WithA(0.85))
		}
	}
	// time grid
	span := float64(t1-t0) / 1e9
	ts := timeStep(span, max(2, plot.W/u.Px(110)))
	first := math.Ceil(float64(t0)/1e9/ts) * ts
	for tt := first; tt <= float64(t1)/1e9; tt += ts {
		x := int(xOf(int64(tt * 1e9)))
		c.VLine(x, plot.Y, plot.Bottom(), th.Grid)
		tm := time.Unix(0, int64(tt*1e9))
		lbl := tm.Format("15:04:05")
		if ts < 1 {
			lbl = tm.Format("15:04:05.0")
		}
		w := u.T.Measure(lbl, fAx)
		u.T.Draw(lbl, x-w/2, plot.Bottom()+u.Px(8), fAx, th.Sub)
	}
	// selection
	if g.SelOn {
		x0, x1 := xOf(g.Sel0), xOf(g.Sel1)
		if x1 < x0 {
			x0, x1 = x1, x0
		}
		sr := Rect{int(x0), plot.Y, int(x1 - x0), plot.H}.Intersect(plot)
		c.FillRect(sr, th.Accent2.WithA(0.12))
		c.VLine(sr.X, plot.Y, plot.Bottom(), th.Accent2.WithA(0.6))
		c.VLine(sr.Right(), plot.Y, plot.Bottom(), th.Accent2.WithA(0.6))
	}
	// series
	u.PushClip(plot)
	for si, s := range ss {
		col := chanColor(s.ch.Color)
		var pts []Pt
		flush := func() {
			if len(pts) == 0 {
				return
			}
			if si == 0 && len(pts) > 1 {
				c.FillAreaUnder(pts, float64(plot.Bottom()), col, 0.16, 0.0)
			}
			if len(pts) == 1 {
				c.FillCircle(pts[0].X, pts[0].Y, u.Pf(2), col)
			} else {
				c.Polyline(pts, u.Pf(1.8), col)
			}
			pts = pts[:0]
		}
		gap := 0
		for x := 0; x < cols; x++ {
			ag := s.agg[x]
			if ag.n == 0 {
				gap++
				if gap > max(8, cols/20) {
					flush()
				}
				continue
			}
			gap = 0
			px := float64(plot.X + x)
			if ag.n > 1 && ag.max != ag.min {
				y0, y1 := yOf(s.gi, ag.max), yOf(s.gi, ag.min)
				c.FillRect(Rect{plot.X + x, int(y0), 1, max(1, int(y1-y0))}, col.WithA(0.35))
			}
			pts = append(pts, Pt{px, yOf(s.gi, ag.sum/float64(ag.n))})
		}
		flush()
		// latest value dot
		if len(pts) == 0 && s.ch.HasLast {
			last := s.ch.Samples[len(s.ch.Samples)-1]
			if last.T >= t0 && last.T <= t1 && !math.IsNaN(last.V) {
				c.FillCircle(xOf(last.T), yOf(s.gi, s.ch.DisplayValue(last.V)), u.Pf(3), col)
			}
		}
	}
	// markers
	for _, m := range a.markers {
		if m.T < t0 || m.T > t1 {
			continue
		}
		x := int(xOf(m.T))
		col := chanColor(m.Color)
		c.DashedVLine(x, plot.Y, plot.Bottom(), u.Px(4), u.Px(3), col.WithA(0.8))
		f := u.FB(11)
		w := u.T.Measure(m.Text, f) + u.Px(12)
		fr := Rect{x + u.Px(3), plot.Y + u.Px(4), w, u.Px(18)}
		c.FillRoundRect(fr, u.Pf(5), col.WithA(0.9))
		u.Text(m.Text, fr, f, hexColor(0x081018), AlignCenter)
	}
	u.PopClip()
	c.StrokeRoundRect(plot.Inset(-u.Px(2)), u.Pf(8), u.Pf(1), th.Border)

	// legend (top-left)
	lx, ly := plot.X+u.Px(10), plot.Y+u.Px(8)
	for _, s := range ss {
		col := chanColor(s.ch.Color)
		lbl := s.ch.Name
		if s.ch.RelOn {
			lbl += " (Δ)"
		}
		f := u.FB(11.5)
		w := u.T.Measure(lbl, f)
		br := Rect{lx, ly, w + u.Px(26), u.Px(20)}
		c.FillRoundRect(br, u.Pf(6), th.Card.WithA(0.85))
		c.FillCircle(float64(lx+u.Px(10)), float64(br.CY()), u.Pf(3.5), col)
		u.T.Draw(lbl, lx+u.Px(19), ly+(br.H-u.T.LineHeight(f))/2, f, th.Text)
		lx += br.W + u.Px(6)
	}
	if len(ss) == 0 {
		u.Text("No data yet — connect a meter or start the demo meter", plot, u.F(14), th.Faint, AlignCenter)
	}

	if !interactive {
		return
	}
	// interaction
	hov := u.Hover(plot)
	g.hoverOn = hov
	id := "graph"
	if hov {
		u.hot = id
		u.Cursor = CurCross
		if u.In.Wheel != 0 {
			if u.In.Ctrl && len(groups) > 0 {
				// zoom Y around cursor
				lo, hi := ranges[0][0], ranges[0][1]
				cy := lo + (float64(plot.Bottom()-u.In.MY)/float64(plot.H))*(hi-lo)
				f := math.Pow(0.85, u.In.Wheel)
				g.AutoY = false
				g.YMin, g.YMax = cy-(cy-lo)*f, cy+(hi-cy)*f
			} else {
				ct := tOf(u.In.MX)
				f := math.Pow(0.8, u.In.Wheel)
				nt0 := ct - int64(float64(ct-t0)*f)
				nt1 := ct + int64(float64(t1-ct)*f)
				if nt1-nt0 < 2e8 {
					nt1 = nt0 + 2e8
				}
				if g.Follow && !u.In.Shift && nt1 >= now-int64(1e9) {
					g.Window = float64(nt1-nt0) / 1e9
				} else {
					g.Follow = false
					g.T0, g.T1 = nt0, nt1
				}
			}
		}
		if u.In.Pressed {
			u.active = id
			g.dragX, g.dragY = u.In.MX, u.In.MY
			g.dragT0, g.dragT1 = t0, t1
			if len(groups) > 0 {
				g.dragY0, g.dragY1 = ranges[0][0], ranges[0][1]
			}
			if u.In.Shift {
				g.drag = 2
				g.SelOn = true
				g.Sel0 = tOf(u.In.MX)
				g.Sel1 = g.Sel0
			} else {
				g.drag = 1
			}
		}
		if u.In.Dbl {
			g.Follow, g.AutoY, g.Paused = true, true, false
			g.SelOn = false
		}
		if u.In.RPressed {
			a.openMarkerDialog(tOf(u.In.MX))
		}
	}
	if u.active == id && u.In.Down {
		switch g.drag {
		case 1:
			dx := u.In.MX - g.dragX
			if abs(dx) > 2 || abs(u.In.MY-g.dragY) > 2 {
				dt := int64(float64(dx) / float64(plot.W) * float64(g.dragT1-g.dragT0))
				g.Follow = false
				g.T0, g.T1 = g.dragT0-dt, g.dragT1-dt
				if !g.AutoY {
					dy := float64(u.In.MY-g.dragY) / float64(plot.H) * (g.dragY1 - g.dragY0)
					g.YMin, g.YMax = g.dragY0+dy, g.dragY1+dy
				}
				u.Cursor = CurMove
			}
		case 2:
			g.Sel1 = tOf(clampi(u.In.MX, plot.X, plot.Right()))
		}
	}
	if u.In.Released && g.drag != 0 {
		if g.drag == 2 && abs(int(xOf(g.Sel1)-xOf(g.Sel0))) < 3 {
			g.SelOn = false
		}
		g.drag = 0
	}
	// crosshair
	if hov && g.drag == 0 {
		x := u.In.MX
		c.VLine(x, plot.Y, plot.Bottom(), th.Text.WithA(0.25))
		ht := tOf(x)
		g.hoverT = ht
		type row struct {
			col  Color
			text string
		}
		rows := []row{{th.Sub, time.Unix(0, ht).Format("15:04:05.000")}}
		for _, s := range ss {
			v, ok := sampleAt(s.ch, ht, s.u)
			if !ok {
				continue
			}
			v = s.ch.DisplayValue(v)
			rows = append(rows, row{chanColor(s.ch.Color), s.ch.Name + "  " + FormatSI(v, s.ch.Units[s.u].Unit, 5)})
			c.FillCircle(float64(x), yOf(s.gi, v), u.Pf(3.5), chanColor(s.ch.Color))
			c.StrokeCircle(float64(x), yOf(s.gi, v), u.Pf(5), u.Pf(1), th.Bg)
		}
		f := u.F(12)
		w := 0
		for _, rw := range rows {
			w = max(w, u.T.Measure(rw.text, f))
		}
		lh := u.T.LineHeight(f) + u.Px(2)
		bx := x + u.Px(14)
		if bx+w+u.Px(20) > plot.Right() {
			bx = x - w - u.Px(34)
		}
		br := Rect{bx, u.In.MY + u.Px(10), w + u.Px(20), lh*len(rows) + u.Px(12)}
		if br.Bottom() > plot.Bottom() {
			br.Y = plot.Bottom() - br.H
		}
		c.Shadow(br, u.Pf(8), u.Pf(10), u.Pf(3), RGBA(0, 0, 0, 120))
		c.FillRoundRect(br, u.Pf(8), th.Card2)
		c.StrokeRoundRect(br, u.Pf(8), u.Pf(1), th.BorderHi)
		for i, rw := range rows {
			u.T.Draw(rw.text, br.X+u.Px(10), br.Y+u.Px(6)+i*lh, f, rw.col)
		}
		u.Ambient = true
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// sampleAt returns the channel value at time t (nearest sample in the same unit).
func sampleAt(ch *Channel, t int64, unit uint8) (float64, bool) {
	n := len(ch.Samples)
	if n == 0 {
		return 0, false
	}
	i := sort.Search(n, func(i int) bool { return ch.Samples[i].T >= t })
	best := -1
	bestD := int64(math.MaxInt64)
	for _, j := range []int{i - 1, i} {
		if j < 0 || j >= n {
			continue
		}
		s := ch.Samples[j]
		if s.U != unit || math.IsNaN(s.V) {
			continue
		}
		d := s.T - t
		if d < 0 {
			d = -d
		}
		if d < bestD {
			best, bestD = j, d
		}
	}
	if best < 0 || bestD > 5e9 {
		return 0, false
	}
	return ch.Samples[best].V, true
}

func fmtSeconds(s float64) string {
	if s < 60 {
		return strconv.FormatFloat(s, 'f', -1, 64) + " s"
	}
	if s < 3600 {
		return strconv.FormatFloat(s/60, 'f', -1, 64) + " min"
	}
	return strconv.FormatFloat(s/3600, 'f', -1, 64) + " h"
}
