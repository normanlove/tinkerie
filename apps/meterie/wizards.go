package meterie

// wizards.go: guided tests — battery capacity, internal resistance,
// resistor/capacitor sorting, LED forward voltage, solar/PSU load test.

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type sortedPart struct {
	T       time.Time
	Value   float64
	Nominal float64
	Dev     float64 // %
	Bin     string
}

type loadPt struct{ V, I float64 }

type WizardState struct {
	which int // -1 list

	// battery capacity
	bcV, bcI          int // channel indices
	bcUseR            bool
	bcR, bcCutoff     float64
	bcNominal         float64
	bcRunning, bcDone bool
	bcStart           time.Time
	bcLastT           time.Time
	bcmAh, bcWh       float64
	bcBelow           time.Time
	bcVstart, bcVnow  float64
	bcInow            float64

	// internal resistance
	irStep    int
	irVoc     float64
	irVl      float64
	irR       float64
	irResults []float64

	// sorter
	soType   int // 0 R 1 C
	soSeries int
	soTol    float64
	soParts  []sortedPart
	soStable time.Time
	soRef    float64
	soArmed  bool
	soAuto   bool

	// LED
	ledSupply, ledmA float64
	ledHist          []float64

	// load test
	ltV, ltI int
	ltUseR   bool
	ltR      float64
	ltMode   int // 0 solar 1 psu
	ltPts    []loadPt
	ltAuto   bool
	ltStable time.Time
	ltLast   float64
}

var wizardInfo = []struct{ icon, title, desc string }{
	{icBolt, "Battery capacity", "Discharge a battery through a load and measure its real mAh and Wh, stopping at a safe cut-off voltage."},
	{icComp, "Internal resistance", "Compare open-circuit and loaded voltage to work out a cell's internal resistance — the best sign of a tired battery."},
	{icList, "Resistor / capacitor sorter", "Touch parts to the probes one after another; Meterie captures each, finds its E-series value and bins it by tolerance."},
	{icBulb, "LED forward voltage", "Measure an LED in diode mode, guess its colour and calculate the right series resistor for your supply."},
	{icChart, "Solar panel / PSU load test", "Step through loads and plot the I–V and power curves: find a panel's maximum power point or a supply's regulation."},
}

func (w *WizardState) defaults() {
	if w.bcR == 0 {
		w.bcR, w.bcCutoff, w.bcNominal = 4.7, 3.0, 2500
	}
	if w.irR == 0 {
		w.irR = 4.7
	}
	if w.soTol == 0 {
		w.soTol, w.soSeries, w.soAuto = 5, 2, true
	}
	if w.ledSupply == 0 {
		w.ledSupply, w.ledmA = 5, 10
	}
	if w.ltR == 0 {
		w.ltR, w.ltI = 10, 1
	}
}

func (a *App) chanAt(idx int) *Channel {
	if idx >= 0 && idx < len(a.channels) {
		return a.channels[idx]
	}
	return nil
}

func liveVal(ch *Channel) (float64, bool) {
	if ch == nil || !ch.HasLast || ch.Last.IsOL() || time.Since(ch.Last.T) > 5*time.Second {
		return 0, false
	}
	return ch.DisplayValue(ch.Last.Value), true
}

// onReading drives the automatic parts of the wizards.
func (w *WizardState) onReading(a *App, ch *Channel, r Reading) {
	switch w.which {
	case 0:
		if !w.bcRunning || ch != a.chanAt(w.bcV) {
			return
		}
		v, ok := liveVal(ch)
		if !ok {
			return
		}
		i := 0.0
		if w.bcUseR {
			if w.bcR > 0 {
				i = v / w.bcR
			}
		} else if ic, ok := liveVal(a.chanAt(w.bcI)); ok {
			i = math.Abs(ic)
		}
		now := r.T
		if !w.bcLastT.IsZero() {
			dt := now.Sub(w.bcLastT).Hours()
			if dt > 0 && dt < 1.0/60 {
				w.bcmAh += i * 1000 * dt
				w.bcWh += v * i * dt
			}
		}
		w.bcLastT, w.bcVnow, w.bcInow = now, v, i
		if v < w.bcCutoff {
			if w.bcBelow.IsZero() {
				w.bcBelow = now
			} else if now.Sub(w.bcBelow) > 5*time.Second {
				w.bcRunning, w.bcDone = false, true
				a.markers = append(a.markers, Marker{T: now.UnixNano(), Text: fmt.Sprintf("cut-off %.0f mAh", w.bcmAh), Color: 7})
				a.ui.Toast(fmt.Sprintf("Battery test finished: %.0f mAh, %.2f Wh", w.bcmAh, w.bcWh), 1)
				if a.settings.SoundOn {
					platformPlaySound(alarmWAV(), false)
				}
			}
		} else {
			w.bcBelow = time.Time{}
		}
	case 2:
		if ch != a.mainChannel() {
			return
		}
		w.sorterFeed(a, ch, r)
	case 4:
		if !w.ltAuto || ch != a.chanAt(w.ltV) {
			return
		}
		v, ok := liveVal(ch)
		if !ok {
			return
		}
		if math.Abs(v-w.ltLast) > math.Max(0.003*math.Abs(v), 0.002) {
			w.ltLast = v
			w.ltStable = r.T
		} else if !w.ltStable.IsZero() && r.T.Sub(w.ltStable) > 1500*time.Millisecond {
			w.ltCapture(a)
			w.ltStable = time.Time{}
		}
	}
}

func (w *WizardState) sorterFeed(a *App, ch *Channel, r Reading) {
	wantQ := QResistance
	if w.soType == 1 {
		wantQ = QCapacitance
	}
	if r.IsOL() || (wantQ == QCapacitance && r.Value < 5e-12) {
		w.soArmed = true
		w.soStable = time.Time{}
		return
	}
	if r.Q != wantQ || !w.soArmed {
		return
	}
	v := r.Value
	if w.soStable.IsZero() || math.Abs(v-w.soRef) > math.Abs(w.soRef)*0.004 {
		w.soStable, w.soRef = r.T, v
		return
	}
	if r.T.Sub(w.soStable) < 900*time.Millisecond {
		return
	}
	series := []string{"E6", "E12", "E24", "E96"}[w.soSeries]
	nom := NearestE(v, series)
	dev := (v - nom) / nom * 100
	bin := "PASS"
	if dev > w.soTol {
		bin = "HIGH"
	} else if dev < -w.soTol {
		bin = "LOW"
	}
	w.soParts = append(w.soParts, sortedPart{T: r.T, Value: v, Nominal: nom, Dev: dev, Bin: bin})
	w.soArmed = false
	w.soStable = time.Time{}
	kind := 1
	if bin != "PASS" {
		kind = 2
	}
	unit := "Ω"
	if w.soType == 1 {
		unit = "F"
	}
	a.ui.Toast(fmt.Sprintf("%s  %s  (%+.1f%%)", bin, FormatSI(nom, unit, 3), dev), kind)
	if a.settings.SoundOn {
		platformPlaySound(shortBeepWAV(), false)
	}
}

func (w *WizardState) ltCapture(a *App) {
	v, ok := liveVal(a.chanAt(w.ltV))
	if !ok {
		a.ui.Toast("No live voltage reading", 2)
		return
	}
	i := 0.0
	if w.ltUseR {
		i = v / math.Max(w.ltR, 1e-9)
	} else if iv, ok := liveVal(a.chanAt(w.ltI)); ok {
		i = math.Abs(iv)
	} else {
		a.ui.Toast("No live current reading", 2)
		return
	}
	w.ltPts = append(w.ltPts, loadPt{v, i})
	a.ui.Toast(fmt.Sprintf("Point %d: %s, %s, %s", len(w.ltPts), FormatSI(v, "V", 4), FormatSI(i, "A", 4), FormatSI(v*i, "W", 4)), 1)
}

func (a *App) pageWizards(r Rect) {
	u := a.ui
	w := &a.wiz
	w.defaults()
	if w.which < 0 || w.which >= len(wizardInfo) {
		w.which = -1
	}
	if w.which < 0 {
		cols := 3
		if r.W < u.Px(900) {
			cols = 2
		}
		gap := u.Px(16)
		cw := (r.W - gap*(cols-1)) / cols
		ch := u.Px(190)
		for i, wi := range wizardInfo {
			cr := Rect{r.X + (i%cols)*(cw+gap), r.Y + (i/cols)*(ch+gap), cw, ch}
			id := "wz#" + itoa(i)
			h, cl := u.interact(id, cr)
			hv := u.Anim(id+"#h", b2f(h), 12)
			cr2 := cr
			cr2.Y -= int(hv * u.Pf(3))
			u.C.Shadow(cr2, u.Pf(14), u.Pf(16+8*hv), u.Pf(4+4*hv), RGBA(0, 0, 0, uint8(90+40*hv)))
			u.C.FillRoundRect(cr2, u.Pf(14), lerpColor(th.Card, th.Card2, hv))
			u.C.StrokeRoundRect(cr2, u.Pf(14), u.Pf(1), lerpColor(th.Border, th.Accent.WithA(0.7), hv))
			ic := Rect{cr2.X + u.Px(20), cr2.Y + u.Px(20), u.Px(44), u.Px(44)}
			u.C.FillRoundRect(ic, u.Pf(12), th.Accent.WithA(0.15))
			u.Text(wi.icon, ic, u.FI(20), th.AccentHi, AlignCenter)
			u.T.Draw(wi.title, cr2.X+u.Px(20), cr2.Y+u.Px(78), u.FH(16), th.Text)
			u.Wrap(wi.desc, cr2.X+u.Px(20), cr2.Y+u.Px(106), cr2.W-u.Px(40), u.F(12.5), th.Sub)
			if cl {
				w.which = i
			}
		}
		return
	}
	// header with back button
	if u.Button("wz#back", Rect{r.X, r.Y, u.Px(110), u.Px(34)}, icLeft, "Wizards", BtnGhost) {
		w.which = -1
		return
	}
	u.T.Draw(wizardInfo[w.which].title, r.X+u.Px(126), r.Y+u.Px(4), u.FH(18), th.Text)
	body := Rect{r.X, r.Y + u.Px(50), r.W, r.H - u.Px(50)}
	switch w.which {
	case 0:
		a.wizBattery(body)
	case 1:
		a.wizIR(body)
	case 2:
		a.wizSorter(body)
	case 3:
		a.wizLED(body)
	case 4:
		a.wizLoad(body)
	}
}

func (a *App) safetyNote(x, y, w int, text string) int {
	u := a.ui
	h := u.WrapHeight(text, w-u.Px(52), u.F(12.5)) + u.Px(18)
	r := Rect{x, y, w, h}
	u.C.FillRoundRect(r, u.Pf(10), hexColor(0x2A1E12))
	u.Text(icWarn, Rect{r.X + u.Px(10), r.Y, u.Px(24), r.H}, u.FI(15), th.Warn, AlignCenter)
	u.Wrap(text, r.X+u.Px(42), r.Y+u.Px(9), r.W-u.Px(52), u.F(12.5), hexColor(0xFFD8A8))
	return h
}

func (a *App) wizBattery(r Rect) {
	u := a.ui
	w := &a.wiz
	gap := u.Px(16)
	left, right := r.CutLeft(r.W*42/100 - gap/2)
	right.X += gap
	right.W -= gap
	in := u.Card(left, "Setup")
	y := in.Y
	names, _ := a.channelNames()
	lab := func(s string) { u.T.Draw(s, in.X, y, u.FB(12), th.Sub); y += u.Px(20) }
	lab("Voltage channel (meter across the battery)")
	u.Dropdown("bc#v", Rect{in.X, y, in.W, u.Px(36)}, names, &w.bcV)
	y += u.Px(48)
	u.Toggle("bc#user", Rect{in.X, y, in.W, u.Px(30)}, &w.bcUseR, "No ammeter — work current out from the load resistor")
	y += u.Px(40)
	if w.bcUseR {
		lab("Load resistor")
		u.NumberField("bc#r", Rect{in.X, y, u.Px(160), u.Px(36)}, &w.bcR, "Ω")
	} else {
		lab("Current channel (meter in series)")
		u.Dropdown("bc#i", Rect{in.X, y, in.W, u.Px(36)}, names, &w.bcI)
	}
	y += u.Px(48)
	lab("Cut-off voltage")
	u.NumberField("bc#cut", Rect{in.X, y, u.Px(160), u.Px(36)}, &w.bcCutoff, "V")
	u.Text("Li-ion 3.0 V · NiMH 1.0 V · Alkaline 0.9 V · Lead-acid 10.5 V (12 V)", Rect{in.X + u.Px(172), y, in.W - u.Px(172), u.Px(36)}, u.F(11.5), th.Faint, AlignLeft)
	y += u.Px(48)
	lab("Rated capacity (optional)")
	u.NumberField("bc#nom", Rect{in.X, y, u.Px(160), u.Px(36)}, &w.bcNominal, "mAh")
	y += u.Px(52)
	y += a.safetyNote(in.X, y, in.W, "Use a load resistor rated for the power (P = V²/R — a 4.7 Ω on a 4.2 V cell is ~3.8 W, it gets hot). Never discharge Li-ion below its rated minimum, and don't leave the test unattended.") + u.Px(12)
	if !w.bcRunning {
		if u.ButtonEx("bc#start", Rect{in.X, y, u.Px(190), u.Px(42)}, icPlay, "Start discharge", BtnPrimary, len(a.channels) > 0) {
			w.bcRunning, w.bcDone = true, false
			w.bcStart, w.bcLastT, w.bcBelow = time.Now(), time.Time{}, time.Time{}
			w.bcmAh, w.bcWh = 0, 0
			if v, ok := liveVal(a.chanAt(w.bcV)); ok {
				w.bcVstart = v
			}
			a.markers = append(a.markers, Marker{T: time.Now().UnixNano(), Text: "discharge start", Color: 6})
		}
	} else if u.Button("bc#stop", Rect{in.X, y, u.Px(190), u.Px(42)}, icStop, "Stop test", BtnDanger) {
		w.bcRunning, w.bcDone = false, true
	}
	// results
	res := u.Card(right, "Results")
	ry := res.Y
	big := func(label, val string, col Color) {
		u.T.Draw(label, res.X, ry, u.F(12.5), th.Sub)
		u.T.Draw(val, res.X, ry+u.Px(18), u.FH(28), col)
		ry += u.Px(64)
	}
	state := "Ready"
	if w.bcRunning {
		state = "Discharging…"
		u.Ambient = true
	} else if w.bcDone {
		state = "Finished"
	}
	u.Chip(res.Right()-u.Px(110), res.Y-u.Px(30), strings.ToUpper(state), th.AccentHi, th.Accent.WithA(0.15))
	half := res.W / 2
	y0 := ry
	big("Capacity", fmt.Sprintf("%.0f mAh", w.bcmAh), th.AccentHi)
	big("Energy", fmt.Sprintf("%.3f Wh", w.bcWh), th.Text)
	big("Elapsed", fmtDur(time.Since(w.bcStart)*time.Duration(b2i(w.bcRunning || w.bcDone))), th.Text)
	ry = y0
	res.X += half
	big("Voltage now", FormatSI(w.bcVnow, "V", 4), th.Text)
	big("Current now", FormatSI(w.bcInow, "A", 4), th.Text)
	pct := "—"
	if w.bcNominal > 0 {
		pct = fmt.Sprintf("%.0f %%", w.bcmAh/w.bcNominal*100)
	}
	big("Of rated capacity", pct, th.Good)
	res.X -= half
	ry += u.Px(6)
	if w.bcNominal > 0 {
		bar := Rect{res.X, ry, res.W, u.Px(12)}
		u.C.FillRoundRect(bar, u.Pf(6), th.Bg2)
		f := clamp01(w.bcmAh / w.bcNominal)
		u.C.FillRoundRect(Rect{bar.X, bar.Y, int(float64(bar.W) * f), bar.H}, u.Pf(6), th.Good)
		ry += u.Px(28)
	}
	u.Wrap("The graph on the Live page shows the full discharge curve; markers are added at the start and at cut-off. Tip: add an Energy math channel too if you have both a voltmeter and an ammeter.", res.X, ry, res.W, u.F(12.5), th.Faint)
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (a *App) wizIR(r Rect) {
	u := a.ui
	w := &a.wiz
	in := u.Card(Rect{r.X, r.Y, r.W * 55 / 100, r.H}, "Steps")
	y := in.Y
	ch := a.mainChannel()
	v, ok := liveVal(ch)
	steps := []string{"Measure the open-circuit voltage", "Connect the load and measure again", "Result"}
	for i, s := range steps {
		col := th.Faint
		if i == w.irStep {
			col = th.AccentHi
		} else if i < w.irStep {
			col = th.Good
		}
		u.C.FillCircle(float64(in.X+u.Px(12)), float64(y+u.Px(12)), u.Pf(11), col.WithA(0.2))
		u.Text(itoa(i+1), Rect{in.X, y, u.Px(24), u.Px(24)}, u.FB(12), col, AlignCenter)
		u.T.Draw(s, in.X+u.Px(34), y+u.Px(2), u.FB(13.5), lerpColor(th.Sub, th.Text, b2f(i == w.irStep)))
		y += u.Px(34)
	}
	y += u.Px(10)
	live := "no live reading on the main channel"
	if ok {
		live = "Live: " + FormatSI(v, ch.Last.Unit, 5)
	}
	u.T.Draw(live, in.X, y, u.FH(18), th.Text)
	y += u.Px(40)
	switch w.irStep {
	case 0:
		y += u.Wrap("Put the meter on DC volts across the battery with nothing else connected. Wait for the reading to settle, then capture it.", in.X, y, in.W, u.F(13), th.Sub) + u.Px(12)
		if u.ButtonEx("ir#c1", Rect{in.X, y, u.Px(220), u.Px(42)}, icCheck, "Capture open-circuit V", BtnPrimary, ok) {
			w.irVoc, w.irStep = v, 1
		}
	case 1:
		y += u.Wrap("Keep the meter across the battery and connect the load resistor. Capture within a couple of seconds (before the cell sags further).", in.X, y, in.W, u.F(13), th.Sub) + u.Px(12)
		u.T.Draw("Load resistor", in.X, y+u.Px(9), u.F(13), th.Text)
		u.NumberField("ir#r", Rect{in.X + u.Px(110), y, u.Px(150), u.Px(36)}, &w.irR, "Ω")
		y += u.Px(50)
		if u.ButtonEx("ir#c2", Rect{in.X, y, u.Px(220), u.Px(42)}, icCheck, "Capture loaded V", BtnPrimary, ok) {
			w.irVl, w.irStep = v, 2
			if i := w.irVl / w.irR; i > 0 {
				w.irResults = append(w.irResults, (w.irVoc-w.irVl)/i)
			}
		}
	case 2:
		i := w.irVl / math.Max(w.irR, 1e-9)
		rint := (w.irVoc - w.irVl) / math.Max(i, 1e-12)
		u.T.Draw("Internal resistance", in.X, y, u.F(13), th.Sub)
		u.T.Draw(FormatSI(rint, "Ω", 3), in.X, y+u.Px(20), u.FH(34), th.AccentHi)
		y += u.Px(74)
		y += u.Wrap(fmt.Sprintf("Open-circuit %s · loaded %s · load current %s (I = V/R) · sag %s", FormatSI(w.irVoc, "V", 4), FormatSI(w.irVl, "V", 4), FormatSI(i, "A", 3), FormatSI(w.irVoc-w.irVl, "V", 3)), in.X, y, in.W, u.F(12.5), th.Sub) + u.Px(14)
		if u.Button("ir#again", Rect{in.X, y, u.Px(180), u.Px(40)}, icRefresh, "Measure again", BtnSecondary) {
			w.irStep = 0
		}
	}
	tr := Rect{r.X + r.W*55/100 + u.Px(16), r.Y, r.W*45/100 - u.Px(16), r.H}
	tin := u.Card(tr, "Typical values (fresh cells)")
	ty := tin.Y
	for _, row := range [][2]string{{"Li-ion 18650 (good)", "20 – 60 mΩ"}, {"Li-ion 18650 (worn)", "> 120 mΩ"}, {"AA alkaline", "150 – 300 mΩ"}, {"AA NiMH", "20 – 50 mΩ"}, {"CR2032 coin", "10 – 40 Ω"}, {"12 V lead-acid", "5 – 20 mΩ"}} {
		u.Text(row[0], Rect{tin.X, ty, tin.W / 2, u.Px(26)}, u.F(13), th.Sub, AlignLeft)
		u.Text(row[1], Rect{tin.X + tin.W/2, ty, tin.W / 2, u.Px(26)}, u.FB(13), th.Text, AlignRight)
		ty += u.Px(30)
	}
	ty += u.Px(8)
	ty += u.Wrap("A hobby meter can't resolve milliohms directly, so this two-step method is more accurate for low resistances. Probe the battery terminals directly (not the load wires) and use a load that draws a meaningful current.", tin.X, ty, tin.W, u.F(12), th.Faint) + u.Px(10)
	if len(w.irResults) > 1 {
		s := "History: "
		for _, v := range w.irResults {
			s += FormatSI(v, "Ω", 3) + "  "
		}
		u.Wrap(s, tin.X, ty, tin.W, u.F(12), th.Sub)
	}
}

func (a *App) wizSorter(r Rect) {
	u := a.ui
	w := &a.wiz
	gap := u.Px(16)
	left, right := r.CutLeft(r.W*40/100 - gap/2)
	right.X += gap
	right.W -= gap
	in := u.Card(left, "Setup")
	y := in.Y
	u.Segmented("so#type", Rect{in.X, y, in.W, u.Px(34)}, []string{"Resistors (Ω)", "Capacitors (F)"}, &w.soType)
	y += u.Px(48)
	u.T.Draw("Value series", in.X, y, u.FB(12), th.Sub)
	y += u.Px(20)
	u.Segmented("so#ser", Rect{in.X, y, in.W, u.Px(32)}, []string{"E6", "E12", "E24", "E96"}, &w.soSeries)
	y += u.Px(46)
	u.T.Draw("Tolerance ±", in.X, y+u.Px(9), u.F(13), th.Text)
	u.NumberField("so#tol", Rect{in.X + u.Px(100), y, u.Px(120), u.Px(36)}, &w.soTol, "%")
	y += u.Px(52)
	y += u.Wrap("Put the main channel's meter on Ω (or capacitance). Touch the probes to each part; when the reading is steady Meterie captures it and beeps. Lift the probes (OL) before the next part.", in.X, y, in.W, u.F(12.5), th.Sub) + u.Px(12)
	ch := a.mainChannel()
	st := "Waiting for an open reading (lift the probes)"
	if w.soArmed {
		st = "Ready — touch the next part"
	}
	if v, ok := liveVal(ch); ok && w.soArmed {
		st = "Measuring " + FormatSI(v, ch.Last.Unit, 4) + " — hold steady"
	}
	u.C.FillRoundRect(Rect{in.X, y, in.W, u.Px(40)}, u.Pf(10), th.Bg2)
	u.Text(st, Rect{in.X + u.Px(12), y, in.W - u.Px(24), u.Px(40)}, u.FB(13), th.AccentHi, AlignLeft)
	y += u.Px(54)
	bw := (in.W - u.Px(8)) / 2
	if u.Button("so#clear", Rect{in.X, y, bw, u.Px(38)}, icDelete, "Clear list", BtnSecondary) {
		w.soParts = nil
	}
	if u.ButtonEx("so#export", Rect{in.X + bw + u.Px(8), y, bw, u.Px(38)}, icSave, "Export CSV", BtnSecondary, len(w.soParts) > 0) {
		a.exportSorterCSV()
	}
	// results: bins + list
	res := u.Card(right, fmt.Sprintf("Parts (%d)", len(w.soParts)))
	bins := []string{"PASS", "LOW", "HIGH"}
	cnt := map[string]int{}
	for _, p := range w.soParts {
		cnt[p.Bin]++
	}
	bx := res.X
	for _, b := range bins {
		col := map[string]Color{"PASS": th.Good, "LOW": th.Warn, "HIGH": th.Bad}[b]
		br := Rect{bx, res.Y, (res.W - u.Px(16)) / 3, u.Px(64)}
		u.C.FillRoundRect(br, u.Pf(10), col.WithA(0.12))
		u.T.Draw(b, br.X+u.Px(12), br.Y+u.Px(8), u.FB(11.5), col)
		u.T.Draw(itoa(cnt[b]), br.X+u.Px(12), br.Y+u.Px(26), u.FH(24), th.Text)
		bx += br.W + u.Px(8)
	}
	// histogram of deviation
	hy := res.Y + u.Px(78)
	hr := Rect{res.X, hy, res.W, u.Px(70)}
	u.C.FillRoundRect(hr, u.Pf(8), th.Bg2)
	nb := 21
	hist := make([]int, nb)
	span := math.Max(w.soTol*2, 1)
	for _, p := range w.soParts {
		i := int((p.Dev/span + 1) / 2 * float64(nb))
		hist[clampi(i, 0, nb-1)]++
	}
	mx := 1
	for _, h := range hist {
		mx = max(mx, h)
	}
	bwid := float64(hr.W-u.Px(16)) / float64(nb)
	for i, h := range hist {
		dev := (float64(i)+0.5)/float64(nb)*2*span - span
		col := th.Good
		if math.Abs(dev) > w.soTol {
			col = th.Bad
		}
		bh := float64(hr.H-u.Px(16)) * float64(h) / float64(mx)
		u.C.FillRoundRectF(float64(hr.X+u.Px(8))+float64(i)*bwid+1, float64(hr.Bottom()-u.Px(8))-bh, bwid-2, bh, u.Pf(2), col.WithA(0.8))
	}
	u.C.VLine(hr.CX(), hr.Y+u.Px(4), hr.Bottom()-u.Px(4), th.Text.WithA(0.3))
	ly := hy + u.Px(84)
	unit := "Ω"
	if w.soType == 1 {
		unit = "F"
	}
	lr := Rect{res.X, ly, res.W, res.Bottom() - ly}
	off := u.BeginScroll("so#list", lr)
	yy := ly - off
	for i := len(w.soParts) - 1; i >= 0; i-- {
		p := w.soParts[i]
		col := map[string]Color{"PASS": th.Good, "LOW": th.Warn, "HIGH": th.Bad}[p.Bin]
		row := Rect{res.X, yy, res.W - u.Px(8), u.Px(30)}
		u.Text("#"+itoa(i+1), Rect{row.X, row.Y, u.Px(40), row.H}, u.F(12), th.Faint, AlignLeft)
		u.Text(FormatSI(p.Value, unit, 4), Rect{row.X + u.Px(44), row.Y, u.Px(110), row.H}, u.FB(13), th.Text, AlignLeft)
		u.Text("→ "+FormatSI(p.Nominal, unit, 3), Rect{row.X + u.Px(160), row.Y, u.Px(110), row.H}, u.F(13), th.Sub, AlignLeft)
		u.Text(fmt.Sprintf("%+.2f %%", p.Dev), Rect{row.X + u.Px(280), row.Y, u.Px(90), row.H}, u.F(13), th.Text, AlignLeft)
		u.Chip(row.Right()-u.Px(70), row.Y+u.Px(5), p.Bin, col, col.WithA(0.15))
		yy += u.Px(32)
	}
	u.EndScroll("so#list", lr, yy+off-ly)
}

func (a *App) exportSorterCSV() {
	p, ok := platformSaveDialog("Export sorted parts", filepath.Join(a.dataDir(), "meterie-parts.csv"), "CSV files (*.csv)", "csv")
	if !ok {
		return
	}
	var b strings.Builder
	b.WriteString("index,time,measured,nominal,deviation_percent,bin\n")
	for i, s := range a.wiz.soParts {
		fmt.Fprintf(&b, "%d,%s,%s,%s,%.3f,%s\n", i+1, s.T.Format("15:04:05"), FormatValue(s.Value), FormatValue(s.Nominal), s.Dev, s.Bin)
	}
	if err := os.WriteFile(p, []byte(b.String()), 0o644); err != nil {
		a.ui.Toast("Export failed: "+err.Error(), 3)
		return
	}
	a.ui.Toast("Saved "+filepath.Base(p), 1)
}

func ledGuess(vf float64) (string, Color) {
	switch {
	case vf < 1.0:
		return "Not an LED? (very low Vf — maybe a Schottky or a short)", th.Faint
	case vf < 1.55:
		return "Infrared", hexColor(0x9B5DE5)
	case vf < 2.0:
		return "Red", hexColor(0xFF4D4D)
	case vf < 2.15:
		return "Orange / amber", hexColor(0xFF9F1C)
	case vf < 2.4:
		return "Yellow or yellow-green", hexColor(0xFFE45E)
	case vf < 2.9:
		return "Green (older) or pure green", hexColor(0x3DDC84)
	case vf < 3.6:
		return "Blue, white, true green or UV", hexColor(0x7FB3FF)
	}
	return "High Vf — several LEDs in series, or not lit by the meter", th.Faint
}

func (a *App) wizLED(r Rect) {
	u := a.ui
	w := &a.wiz
	gap := u.Px(16)
	left, right := r.CutLeft(r.W/2 - gap/2)
	right.X += gap
	right.W -= gap
	in := u.Card(left, "Measure")
	y := in.Y
	ch := a.mainChannel()
	v, ok := liveVal(ch)
	isDiode := ok && ch.Last.Q == QDiode
	y += u.Wrap("Set the meter to diode test (the triangle-and-bar symbol). Red probe on the LED's anode (longer leg), black on the cathode (flat side). Many meters make the LED glow faintly.", in.X, y, in.W, u.F(13), th.Sub) + u.Px(12)
	vfTxt := "—"
	if ok {
		vfTxt = FormatSI(v, "V", 4)
	}
	u.T.Draw("Forward voltage", in.X, y, u.F(12.5), th.Sub)
	u.T.Draw(vfTxt, in.X, y+u.Px(20), u.FH(36), th.AccentHi)
	y += u.Px(78)
	if ok && !isDiode {
		y += u.Wrap("The main channel isn't in diode mode — the value may not be a forward voltage.", in.X, y, in.W, u.F(12.5), th.Warn) + u.Px(8)
	}
	if ok {
		name, col := ledGuess(v)
		u.C.FillCircle(float64(in.X+u.Px(14)), float64(y+u.Px(14)), u.Pf(12), col)
		u.C.RadialGlow(float64(in.X+u.Px(14)), float64(y+u.Px(14)), u.Pf(30), col, 0.25)
		u.T.Draw("Probably: "+name, in.X+u.Px(38), y+u.Px(4), u.FB(14), th.Text)
		y += u.Px(44)
	}
	if u.ButtonEx("led#cap", Rect{in.X, y, u.Px(170), u.Px(40)}, icCheck, "Save to list", BtnSecondary, ok) {
		w.ledHist = append(w.ledHist, v)
	}
	y += u.Px(54)
	u.Wrap("Note: a meter's diode test uses ~1 mA, so at 10–20 mA the real Vf is usually 0.1–0.3 V higher. The resistor maths on the right uses your measured value plus 0.15 V for that.", in.X, y, in.W, u.F(12), th.Faint)
	// calculator
	cin := u.Card(right, "Series resistor")
	cy := cin.Y
	u.T.Draw("Supply voltage", cin.X, cy+u.Px(9), u.F(13), th.Text)
	u.NumberField("led#vs", Rect{cin.X + u.Px(140), cy, u.Px(140), u.Px(36)}, &w.ledSupply, "V")
	cy += u.Px(46)
	u.T.Draw("LED current", cin.X, cy+u.Px(9), u.F(13), th.Text)
	u.NumberField("led#ma", Rect{cin.X + u.Px(140), cy, u.Px(140), u.Px(36)}, &w.ledmA, "mA")
	cy += u.Px(56)
	if ok {
		vf := v + 0.15
		i := w.ledmA / 1000
		if w.ledSupply <= vf || i <= 0 {
			u.Wrap("The supply must be higher than the LED's forward voltage.", cin.X, cy, cin.W, u.F(13), th.Warn)
		} else {
			rr := (w.ledSupply - vf) / i
			re := NearestEAbove(rr, "E12")
			p := i * i * re
			ia := (w.ledSupply - vf) / re
			watt := "1/8 W"
			switch {
			case p > 0.5:
				watt = "1 W or more"
			case p > 0.25:
				watt = "1/2 W"
			case p > 0.125:
				watt = "1/4 W"
			}
			u.T.Draw("Use a", cin.X, cy, u.F(12.5), th.Sub)
			u.T.Draw(FormatSI(re, "Ω", 3)+" resistor", cin.X, cy+u.Px(18), u.FH(30), th.AccentHi)
			cy += u.Px(70)
			cy += u.Wrap(fmt.Sprintf("Exact value %s → next E12 value up %s. The LED will draw about %.1f mA and the resistor dissipates %s — a %s part is fine.", FormatSI(rr, "Ω", 3), FormatSI(re, "Ω", 3), ia*1000, FormatSI(p, "W", 2), watt), cin.X, cy, cin.W, u.F(12.5), th.Sub) + u.Px(10)
		}
	} else {
		cy += u.Wrap("Measure an LED first.", cin.X, cy, cin.W, u.F(13), th.Faint)
	}
	if len(w.ledHist) > 0 {
		cy += u.Px(10)
		u.T.Draw("SAVED", cin.X, cy, u.FB(11), th.Faint)
		cy += u.Px(20)
		for i := len(w.ledHist) - 1; i >= 0 && cy < cin.Bottom()-u.Px(24); i-- {
			n, col := ledGuess(w.ledHist[i])
			u.C.FillCircle(float64(cin.X+u.Px(7)), float64(cy+u.Px(10)), u.Pf(5), col)
			u.T.Draw(FormatSI(w.ledHist[i], "V", 4)+"   "+n, cin.X+u.Px(20), cy+u.Px(2), u.F(12.5), th.Text)
			cy += u.Px(24)
		}
	}
}

func (a *App) wizLoad(r Rect) {
	u := a.ui
	w := &a.wiz
	gap := u.Px(16)
	left, right := r.CutLeft(r.W*36/100 - gap/2)
	right.X += gap
	right.W -= gap
	in := u.Card(left, "Setup")
	y := in.Y
	u.Segmented("lt#mode", Rect{in.X, y, in.W, u.Px(34)}, []string{"Solar panel", "Power supply"}, &w.ltMode)
	y += u.Px(48)
	names, _ := a.channelNames()
	u.T.Draw("Voltage channel", in.X, y, u.FB(12), th.Sub)
	y += u.Px(20)
	u.Dropdown("lt#v", Rect{in.X, y, in.W, u.Px(36)}, names, &w.ltV)
	y += u.Px(46)
	u.Toggle("lt#user", Rect{in.X, y, in.W, u.Px(30)}, &w.ltUseR, "No ammeter — I'll enter each load resistor")
	y += u.Px(40)
	if w.ltUseR {
		u.T.Draw("Current load resistor", in.X, y, u.FB(12), th.Sub)
		y += u.Px(20)
		u.NumberField("lt#r", Rect{in.X, y, u.Px(160), u.Px(36)}, &w.ltR, "Ω")
	} else {
		u.T.Draw("Current channel", in.X, y, u.FB(12), th.Sub)
		y += u.Px(20)
		u.Dropdown("lt#i", Rect{in.X, y, in.W, u.Px(36)}, names, &w.ltI)
	}
	y += u.Px(48)
	u.Toggle("lt#auto", Rect{in.X, y, in.W, u.Px(30)}, &w.ltAuto, "Auto-capture when the reading settles")
	y += u.Px(42)
	bw := (in.W - u.Px(8)) / 2
	if u.Button("lt#cap", Rect{in.X, y, bw, u.Px(40)}, icAdd, "Capture point", BtnPrimary) {
		w.ltCapture(a)
	}
	if u.Button("lt#clr", Rect{in.X + bw + u.Px(8), y, bw, u.Px(40)}, icDelete, "Clear", BtnSecondary) {
		w.ltPts = nil
	}
	y += u.Px(54)
	tip := "Solar: start with no load (open circuit), then add loads step by step down to a short through the ammeter. Keep the sun steady!"
	if w.ltMode == 1 {
		tip = "PSU: start unloaded, then add load in steps up to the rated current. Watch the supply's temperature."
	}
	y += u.Wrap(tip, in.X, y, in.W, u.F(12.5), th.Sub) + u.Px(10)
	a.safetyNote(in.X, y, in.W, "Load resistors get hot: P = V × I. Use parts rated well above the power and never short a mains-powered supply.")

	res := u.Card(right, "Curves")
	pts := append([]loadPt(nil), w.ltPts...)
	sort.Slice(pts, func(i, j int) bool { return pts[i].V < pts[j].V })
	chartR := Rect{res.X, res.Y, res.W, res.H - u.Px(110)}
	a.drawIVChart(chartR, pts)
	sy := chartR.Bottom() + u.Px(14)
	if len(pts) == 0 {
		u.Wrap("Captured points appear here as an I–V curve (current vs voltage) with the power curve on top.", res.X, sy, res.W, u.F(13), th.Faint)
		return
	}
	var voc, isc, pmax, vmp, imp float64
	for _, p := range pts {
		voc = math.Max(voc, p.V)
		isc = math.Max(isc, p.I)
		if p.V*p.I > pmax {
			pmax, vmp, imp = p.V*p.I, p.V, p.I
		}
	}
	kv := []struct{ k, v string }{}
	if w.ltMode == 0 {
		ff := 0.0
		if voc*isc > 0 {
			ff = pmax / (voc * isc) * 100
		}
		kv = append(kv, struct{ k, v string }{"Max power", FormatSI(pmax, "W", 4) + " at " + FormatSI(vmp, "V", 4) + " / " + FormatSI(imp, "A", 3)},
			struct{ k, v string }{"Voc · Isc", FormatSI(voc, "V", 4) + " · " + FormatSI(isc, "A", 3)},
			struct{ k, v string }{"Fill factor", fmt.Sprintf("%.0f %%", ff)})
	} else {
		vnl := pts[len(pts)-1].V
		vfl := vnl
		ifl := 0.0
		for _, p := range w.ltPts {
			if p.I > ifl {
				ifl, vfl = p.I, p.V
			}
		}
		reg := 0.0
		if vfl > 0 {
			reg = (vnl - vfl) / vfl * 100
		}
		kv = append(kv, struct{ k, v string }{"Load regulation", fmt.Sprintf("%.2f %%", reg) + "  (" + FormatSI(vnl, "V", 4) + " → " + FormatSI(vfl, "V", 4) + ")"},
			struct{ k, v string }{"Max load tested", FormatSI(ifl, "A", 3) + " · " + FormatSI(vfl*ifl, "W", 3)},
			struct{ k, v string }{"Points", itoa(len(pts))})
	}
	for i, x := range kv {
		yy := sy + i*u.Px(28)
		u.Text(x.k, Rect{res.X, yy, res.W / 3, u.Px(26)}, u.F(13), th.Sub, AlignLeft)
		u.Text(x.v, Rect{res.X + res.W/3, yy, res.W * 2 / 3, u.Px(26)}, u.FB(13.5), th.Text, AlignRight)
	}
}

func (a *App) drawIVChart(r Rect, pts []loadPt) {
	u := a.ui
	c := u.C
	plot := Rect{r.X + u.Px(56), r.Y + u.Px(10), r.W - u.Px(56) - u.Px(56), r.H - u.Px(36)}
	c.FillRoundRect(plot.Inset(-u.Px(2)), u.Pf(8), th.Bg2)
	var vmax, imax, pmax float64
	for _, p := range pts {
		vmax = math.Max(vmax, p.V)
		imax = math.Max(imax, p.I)
		pmax = math.Max(pmax, p.V*p.I)
	}
	if vmax == 0 {
		vmax = 1
	}
	if imax == 0 {
		imax = 1
	}
	if pmax == 0 {
		pmax = 1
	}
	vmax *= 1.08
	imax *= 1.12
	pmax *= 1.12
	f := u.F(11)
	for i := 0; i <= 4; i++ {
		y := plot.Bottom() - plot.H*i/4
		c.HLine(plot.X, plot.Right(), y, th.Grid)
		l := FormatSI(imax*float64(i)/4, "A", 3)
		u.T.Draw(l, plot.X-u.T.Measure(l, f)-u.Px(6), y-u.Px(7), f, chanColor(1))
		l2 := FormatSI(pmax*float64(i)/4, "W", 3)
		u.T.Draw(l2, plot.Right()+u.Px(6), y-u.Px(7), f, chanColor(2))
		x := plot.X + plot.W*i/4
		lv := FormatSI(vmax*float64(i)/4, "V", 3)
		u.T.Draw(lv, x-u.T.Measure(lv, f)/2, plot.Bottom()+u.Px(6), f, th.Sub)
	}
	xy := func(v, y, ymax float64) Pt {
		return Pt{float64(plot.X) + v/vmax*float64(plot.W), float64(plot.Bottom()) - y/ymax*float64(plot.H)}
	}
	var iv, pv []Pt
	for _, p := range pts {
		iv = append(iv, xy(p.V, p.I, imax))
		pv = append(pv, xy(p.V, p.V*p.I, pmax))
	}
	c.Polyline(iv, u.Pf(2), chanColor(1))
	c.Polyline(pv, u.Pf(2), chanColor(2))
	best := -1
	for i, p := range pts {
		c.FillCircle(iv[i].X, iv[i].Y, u.Pf(3.5), chanColor(1))
		c.FillCircle(pv[i].X, pv[i].Y, u.Pf(3), chanColor(2))
		if best < 0 || p.V*p.I > pts[best].V*pts[best].I {
			best = i
		}
	}
	if best >= 0 {
		c.StrokeCircle(pv[best].X, pv[best].Y, u.Pf(8), u.Pf(2), th.Good)
		u.T.Draw("MPP", int(pv[best].X)+u.Px(10), int(pv[best].Y)-u.Px(20), u.FB(11), th.Good)
	}
	c.FillCircle(float64(plot.X+u.Px(14)), float64(plot.Y+u.Px(13)), u.Pf(4), chanColor(1))
	u.T.Draw("Current", plot.X+u.Px(22), plot.Y+u.Px(6), u.FB(11), chanColor(1))
	c.FillCircle(float64(plot.X+u.Px(94)), float64(plot.Y+u.Px(13)), u.Pf(4), chanColor(2))
	u.T.Draw("Power", plot.X+u.Px(102), plot.Y+u.Px(6), u.FB(11), chanColor(2))
}
