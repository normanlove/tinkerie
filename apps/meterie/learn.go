package meterie

// learn.go: "What am I measuring?" — a beginner guide to each mode, with a
// drawn meter showing which jacks to use and how to connect, plus safety.

import (
	"math"
	"time"
)

type lesson struct {
	title, symbol, what, analogy, connect, dial, tips string
	red                                               int // jack index for red lead: 0=10A, 1=mA/µA, 2=COM, 3=VΩHz
	circuit                                           int // 0 parallel (across), 1 series (in line), 2 part on its own, 3 thermocouple
	q                                                 Quantity
	ac                                                bool
}

var lessons = []lesson{
	{title: "DC voltage", symbol: "DC V", q: QVoltage,
		what:    "The 'push' that makes current flow, from a steady source like a battery, USB port or bench supply.",
		analogy: "Like water pressure in a pipe: the difference in pressure between two points.",
		connect: "Connect ACROSS the thing you're measuring (in parallel) — red probe to the + side, black to − or ground. The circuit stays powered; nothing is disconnected.",
		dial:    "Turn the dial to V with the straight line (DC, often drawn as a solid line over a dashed one). On manual-range meters start at the highest range and work down.",
		tips:    "A minus sign just means the probes are the other way round. Reading the voltage of a battery with no load says little about its health — use the internal-resistance wizard for that.",
		red:     3, circuit: 0},
	{title: "AC voltage", symbol: "AC V", q: QVoltage, ac: true,
		what:    "Voltage that keeps reversing direction, like mains (230 V / 120 V) or a transformer's output. Meters show the RMS value.",
		analogy: "Like a tide going in and out — the meter tells you its 'effective' strength.",
		connect: "Across the source, exactly like DC voltage. Polarity doesn't matter.",
		dial:    "V with the wavy line (AC). 'True RMS' meters stay accurate on non-sine waveforms (dimmers, inverters).",
		tips:    "Mains is genuinely dangerous: only measure it if you are trained, your meter and leads are CAT III rated or better, and you keep fingers behind the probe guards. Low-voltage transformer outputs are a safe place to practise.",
		red:     3, circuit: 0},
	{title: "Current (amps)", symbol: "A", q: QCurrent,
		what:    "How much charge flows through a wire each second — what actually does the work and heats things up.",
		analogy: "Like the flow rate of water through a pipe, so the meter has to be IN the pipe.",
		connect: "Break the circuit and put the meter IN SERIES, so all the current flows through it. Power off first, open the circuit, connect the meter into the gap, then power up.",
		dial:    "A / mA / µA. Move the red lead to the mA or 10A jack: use 10A if you're not sure how much current flows, and only use mA for small currents (it's fused at a few hundred mA).",
		tips:    "NEVER put the meter across a battery or supply while it's in current mode — that's a short circuit through the meter and blows its fuse (or worse). Move the red lead back to V/Ω when you're done!",
		red:     0, circuit: 1},
	{title: "Resistance (ohms)", symbol: "Ω", q: QResistance,
		what:    "How strongly a part resists current. Resistors are marked in Ω, kΩ and MΩ.",
		analogy: "Like how narrow a pipe is — narrower pipe, less flow for the same pressure.",
		connect: "Measure the part ON ITS OWN with the power OFF. Take at least one leg out of the circuit, or other parts in parallel will change the reading.",
		dial:    "Ω. 'OL' means the resistance is higher than the range (or an open circuit).",
		tips:    "Don't hold both resistor legs with your fingers — your body (~100 kΩ+) adds a parallel path. Use the sorter wizard to measure lots of resistors quickly.",
		red:     3, circuit: 2},
	{title: "Continuity", symbol: "·)))", q: QContinuity,
		what:    "A quick 'is it connected?' test: the meter beeps when there's a low-resistance path.",
		analogy: "Like checking a hose isn't blocked by blowing through it.",
		connect: "Power OFF. Touch the probes to both ends of a wire, fuse, switch or PCB track.",
		dial:    "The speaker/sound-wave symbol. Meterie can beep on the PC too (Alarms page).",
		tips:    "Great for finding broken wires and solder bridges. A beep only means 'low resistance' — usually under about 30–50 Ω.",
		red:     3, circuit: 2},
	{title: "Diode test", symbol: "→|", q: QDiode,
		what:    "Pushes a small current through a diode or LED and shows the forward voltage it needs to conduct.",
		analogy: "Like a one-way valve: it opens one way (shows a voltage) and blocks the other (shows OL).",
		connect: "Power OFF. Red to the anode, black to the cathode (the striped end / LED's flat side, shorter leg). Swap to check it blocks.",
		dial:    "The diode symbol (a triangle pointing at a bar). Silicon diodes read ~0.5–0.7 V, Schottky ~0.2–0.4 V, LEDs 1.6–3.3 V.",
		tips:    "OL both ways = open (dead) diode; ~0 V both ways = shorted. Use the LED wizard to pick a series resistor.",
		red:     3, circuit: 2},
	{title: "Capacitance", symbol: "-||-", q: QCapacitance,
		what:    "How much charge a capacitor can store, in nF, µF or mF.",
		analogy: "Like the size of a water tank that fills and empties.",
		connect: "Power OFF and DISCHARGE the capacitor first (large ones can hold a dangerous charge). Measure it out of circuit; for electrolytics, red to +.",
		dial:    "The capacitor symbol -||-. Big capacitors take a few seconds to read.",
		tips:    "Small values (pF) are affected by lead capacitance — use the REL Δ button with the probes open to zero it out.",
		red:     3, circuit: 2},
	{title: "Frequency", symbol: "Hz", q: QFrequency,
		what:    "How many times per second a signal repeats: 50/60 Hz mains, PWM, oscillators.",
		analogy: "Like counting waves hitting a beach each second.",
		connect: "Across the signal like a voltage (black to ground, red to the signal).",
		dial:    "Hz (often a button on the V or Hz position). Duty cycle (%) shows how long the signal is 'on'.",
		tips:    "Very small or noisy signals may not trigger the counter — an oscilloscope (Scopie!) shows what's really going on.",
		red:     3, circuit: 0},
	{title: "Temperature", symbol: "°C", q: QTemperature,
		what:    "Reads a K-type thermocouple probe plugged into the meter.",
		analogy: "Like a thermometer on a wire — the tip is the sensing point.",
		connect: "Plug the thermocouple's plug into COM and the V/Ω jack (or the meter's dedicated socket), observing + and −. Touch the tip to the thing you're measuring.",
		dial:    "°C / °F.",
		tips:    "Never touch the thermocouple to anything live. The cheap bead probes are fine to ~250 °C; use a proper probe for soldering-iron tips.",
		red:     3, circuit: 3},
}

func (a *App) pageLearn(r Rect) {
	u := a.ui
	gap := u.Px(16)
	listW := clampi(r.W*22/100, u.Px(210), u.Px(260))
	list, rest := r.CutLeft(listW)
	rest.X += gap
	rest.W -= gap
	// mode list
	lin := u.Card(list, "Modes")
	y := lin.Y
	for i, l := range lessons {
		ir := Rect{lin.X, y, lin.W, u.Px(40)}
		if u.ListItem("learn#"+itoa(i), ir, a.learnSel == i) {
			a.learnSel = i
		}
		sr := Rect{ir.X + u.Px(8), ir.Y + u.Px(7), u.Px(42), ir.H - u.Px(14)}
		u.C.FillRoundRect(sr, u.Pf(6), th.Accent.WithA(0.14))
		u.Text(l.symbol, sr, u.FB(11.5), th.AccentHi, AlignCenter)
		u.Text(l.title, Rect{ir.X + u.Px(58), ir.Y, ir.W - u.Px(64), ir.H}, u.FB(13), th.Text, AlignLeft)
		y += u.Px(44)
	}
	y += u.Px(6)
	if ch := a.mainChannel(); ch != nil && ch.HasLast {
		if u.Button("learn#mine", Rect{lin.X, y, lin.W, u.Px(36)}, icHelp, "Explain my meter's mode", BtnSecondary) {
			for i, l := range lessons {
				if l.q == ch.Last.Q && (l.q != QVoltage || l.ac == ch.Last.Flags.Has(FAC)) {
					a.learnSel = i
				}
			}
		}
	}
	// lesson + safety
	safetyH := u.Px(206)
	lr := Rect{rest.X, rest.Y, rest.W, rest.H - safetyH - gap}
	sr := Rect{rest.X, lr.Bottom() + gap, rest.W, safetyH}
	l := lessons[clampi(a.learnSel, 0, len(lessons)-1)]
	in := u.Card(lr, "")
	illusW := clampi(in.W*42/100, u.Px(300), u.Px(440))
	txt, ill := in.CutLeft(in.W - illusW - gap)
	ill.X += gap
	ill.W -= gap
	off := u.BeginScroll("learn#txt", txt)
	ty := txt.Y - off
	u.T.Draw(l.title, txt.X, ty, u.FH(24), th.Text)
	u.Chip(txt.X+u.T.Measure(l.title, u.FH(24))+u.Px(12), ty+u.Px(8), l.symbol, th.AccentHi, th.Accent.WithA(0.15))
	ty += u.Px(42)
	ty += u.Wrap(l.what, txt.X, ty, txt.W-u.Px(10), u.F(14.5), th.Text) + u.Px(8)
	ty += u.Wrap("Think of it like this: "+l.analogy, txt.X, ty, txt.W-u.Px(10), u.F(13), th.Sub) + u.Px(16)
	sect := func(icon, title, body string) {
		u.Text(icon, Rect{txt.X, ty, u.Px(20), u.Px(20)}, u.FI(14), th.Accent, AlignLeft)
		u.T.Draw(title, txt.X+u.Px(26), ty, u.FB(13.5), th.Text)
		ty += u.Px(22)
		ty += u.Wrap(body, txt.X+u.Px(26), ty, txt.W-u.Px(36), u.F(13), th.Sub) + u.Px(14)
	}
	sect(icComp, "How to connect", l.connect)
	sect(icSettings, "Dial & jacks", l.dial)
	sect(icBulb, "Good to know", l.tips)
	u.EndScroll("learn#txt", txt, ty+off-txt.Y)
	a.drawMeterIllustration(ill, l)

	// safety card
	sin := u.Card(sr, "")
	u.C.FillRoundRect(sr, u.Pf(14), hexColor(0x1F1611))
	u.C.StrokeRoundRect(sr, u.Pf(14), u.Pf(1), th.Warn.WithA(0.45))
	u.Text(icWarn, Rect{sin.X, sin.Y, u.Px(26), u.Px(26)}, u.FI(18), th.Warn, AlignLeft)
	u.T.Draw("Safety first: mains and high voltage can kill", sin.X+u.Px(34), sin.Y+u.Px(1), u.FH(16), hexColor(0xFFD8A8))
	colW := (sin.W - u.Px(24)) / 2
	y1 := sin.Y + u.Px(34)
	rules := "• Beginners: stick to batteries and low-voltage projects (under 30 V AC / 60 V DC).\n• Select the mode and jacks BEFORE touching the circuit. Never measure voltage with the red lead in the A/mA jack.\n• Inspect leads for cracks; keep fingers behind the guards; one hand away from the circuit.\n• Capacitors in power supplies can stay charged after unplugging — discharge them safely."
	u.Wrap(rules, sin.X, y1, colW, u.F(12.5), hexColor(0xF3D9C2))
	cx := sin.X + colW + u.Px(24)
	hh := u.Wrap("CAT ratings (printed by the jacks) — use a rating at least as high as where you measure:", cx, y1, sin.Right()-cx, u.FB(12), hexColor(0xFFD8A8))
	cats := [][2]string{
		{"CAT I", "Electronics not connected to mains, battery projects"},
		{"CAT II", "Plug-in appliances and wall sockets"},
		{"CAT III", "Fuse boards, fixed wiring, hard-wired machines"},
		{"CAT IV", "Supply origin: meters, service entrance, overhead lines"},
	}
	cy := y1 + hh + u.Px(4)
	for _, c := range cats {
		u.Chip(cx, cy, c[0], hexColor(0x1F1611), th.Warn)
		u.T.Draw(c[1], cx+u.Px(72), cy+u.Px(1), u.F(12.5), hexColor(0xF3D9C2))
		cy += u.Px(26)
	}
}

// drawMeterIllustration draws a stylised meter with the right jacks lit and a small circuit.
func (a *App) drawMeterIllustration(r Rect, l lesson) {
	u := a.ui
	c := u.C
	s := u.S
	t := float64(time.Now().UnixMilli()) / 1000
	u.Ambient = true
	mw := math.Min(float64(r.W)*0.62, 230*s)
	mh := mw * 1.55
	mx := float64(r.X) + (float64(r.W)-mw)/2
	my := float64(r.Y) + 6*s
	// body
	c.Shadow(Rect{int(mx), int(my), int(mw), int(mh)}, mw*0.12, 18*s, 6*s, RGBA(0, 0, 0, 120))
	c.FillRoundRectF(mx, my, mw, mh, mw*0.12, hexColor(0xE9A23B))
	c.FillRoundRectF(mx+mw*0.06, my+mw*0.06, mw*0.88, mh-mw*0.12, mw*0.09, hexColor(0x1E242D))
	// screen
	sx, sy, sw, sh := mx+mw*0.14, my+mw*0.14, mw*0.72, mw*0.36
	c.FillRoundRectF(sx, sy, sw, sh, 6*s, hexColor(0xB9C7B0))
	disp := map[Quantity]string{QVoltage: "12.03", QCurrent: "0.250", QResistance: "4.70", QContinuity: "0.6", QDiode: "0.612", QCapacitance: "100.2", QFrequency: "50.00", QTemperature: "24"}[l.q]
	dr := Rect{int(sx + sw*0.06), int(sy + sh*0.22), int(sw * 0.88), int(sh * 0.58)}
	u.DrawLCDDigits("illus", dr, disp, 4, hexColor(0x1D2620), hexColor(0xAFBDA6), false)
	u.T.Draw(l.symbol, int(sx+sw*0.06), int(sy+sh*0.04), u.FB(10), hexColor(0x1D2620))
	// dial
	dcx, dcy, dr2 := mx+mw/2, my+mw*0.82, mw*0.2
	c.FillCircle(dcx, dcy, dr2*1.15, hexColor(0x2A313B))
	c.FillCircle(dcx, dcy, dr2, hexColor(0x3A424E))
	ang := map[Quantity]float64{QVoltage: -2.4, QCurrent: 0.9, QResistance: -1.0, QContinuity: -0.6, QDiode: -0.2, QCapacitance: 0.3, QFrequency: -1.6, QTemperature: 2.0}[l.q]
	if l.ac {
		ang = -1.9
	}
	c.Line(dcx, dcy, dcx+math.Cos(ang)*dr2*0.85, dcy+math.Sin(ang)*dr2*0.85, 5*s, th.Text)
	for i := 0; i < 12; i++ {
		aa := -math.Pi + float64(i)*math.Pi*2/12
		c.FillCircle(dcx+math.Cos(aa)*dr2*1.4, dcy+math.Sin(aa)*dr2*1.4, 1.6*s, th.Sub)
	}
	// jacks
	labels := []string{"10A", "mA µA", "COM", "VΩHz"}
	jy := my + mh - mw*0.2
	pulse := 0.5 + 0.5*math.Sin(t*4)
	var jx [4]float64
	for i := 0; i < 4; i++ {
		jx[i] = mx + mw*(0.17+float64(i)*0.22)
		jr := mw * 0.065
		active := i == l.red || i == 2
		ring := hexColor(0x4A525E)
		if active {
			ring = th.Text
			if i == 2 {
				ring = hexColor(0x9AA4B2)
			} else {
				ring = hexColor(0xFF5A5A)
			}
			c.FillCircle(jx[i], jy, jr*(1.6+0.25*pulse), ring.WithA(0.22))
		}
		c.FillCircle(jx[i], jy, jr, ring)
		c.FillCircle(jx[i], jy, jr*0.55, hexColor(0x0B0E13))
		f := u.FB(9.5)
		w := u.T.Measure(labels[i], f)
		u.T.Draw(labels[i], int(jx[i])-w/2, int(jy-jr-15*s), f, lerpColor(th.Faint, th.Text, b2f(active)))
	}
	// leads
	cy := float64(r.Bottom()) - 70*s
	redX, blkX := float64(r.X)+float64(r.W)*0.72, float64(r.X)+float64(r.W)*0.28
	if l.circuit == 1 {
		redX, blkX = float64(r.X)+float64(r.W)*0.6, float64(r.X)+float64(r.W)*0.4
	}
	lead := func(x0, y0, x1, y1 float64, col Color) {
		var pts []Pt
		for i := 0; i <= 20; i++ {
			tt := float64(i) / 20
			x := x0 + (x1-x0)*tt
			y := y0 + (y1-y0)*tt + math.Sin(tt*math.Pi)*28*s
			pts = append(pts, Pt{x, y})
		}
		c.Polyline(pts, 4*s, col)
	}
	lead(jx[l.red], jy, redX, cy-18*s, hexColor(0xE5484D))
	lead(jx[2], jy, blkX, cy-18*s, hexColor(0x2B2F36).Lighten(0.15))
	// circuit drawing
	cc := th.Sub
	switch l.circuit {
	case 0, 3: // across a source / component
		bx := float64(r.X) + float64(r.W)*0.2
		bw := float64(r.W) * 0.6
		c.StrokeRoundRect(Rect{int(bx), int(cy), int(bw), int(44 * s)}, 8*s, 2*s, cc)
		u.T.Draw("battery / circuit", int(bx+bw/2)-u.T.Measure("battery / circuit", u.F(11))/2, int(cy+14*s), u.F(11), th.Sub)
		c.FillCircle(redX, cy, 5*s, hexColor(0xE5484D))
		c.FillCircle(blkX, cy, 5*s, hexColor(0x9AA4B2))
		u.T.Draw("+", int(redX)+u.Px(8), int(cy)-u.Px(20), u.FB(12), th.Text)
		u.T.Draw("−", int(blkX)-u.Px(16), int(cy)-u.Px(20), u.FB(12), th.Text)
		u.T.Draw("meter goes ACROSS (in parallel)", r.X, r.Bottom()-u.Px(16), u.FB(11.5), th.AccentHi)
	case 1: // series
		lx := float64(r.X) + float64(r.W)*0.12
		rx := float64(r.X) + float64(r.W)*0.88
		c.Line(lx, cy, blkX, cy, 2*s, cc)
		c.Line(redX, cy, rx, cy, 2*s, cc)
		c.Line(lx, cy, lx, cy+40*s, 2*s, cc)
		c.Line(rx, cy, rx, cy+40*s, 2*s, cc)
		c.Line(lx, cy+40*s, rx, cy+40*s, 2*s, cc)
		c.FillRoundRectF(float64(r.CX())-28*s, cy+30*s, 56*s, 20*s, 4*s, th.Card2)
		u.T.Draw("load", r.CX()-u.Px(13), int(cy+33*s), u.F(11), th.Sub)
		c.FillCircle(redX, cy, 5*s, hexColor(0xE5484D))
		c.FillCircle(blkX, cy, 5*s, hexColor(0x9AA4B2))
		u.T.Draw("circuit opened — meter IN SERIES", r.X, r.Bottom()-u.Px(16), u.FB(11.5), th.AccentHi)
	case 2: // part on its own
		c.FillRoundRectF(float64(r.CX())-34*s, cy-4*s, 68*s, 16*s, 6*s, hexColor(0xD9B58C))
		for i := 0; i < 3; i++ {
			c.FillRect(Rect{r.CX() - u.Px(18) + i*u.Px(12), int(cy - 4*s), u.Px(4), int(16 * s)}, []Color{hexColor(0x7A4A1E), hexColor(0x9B3DE5), hexColor(0xE5484D)}[i])
		}
		c.Line(blkX, cy+4*s, float64(r.CX())-34*s, cy+4*s, 2*s, cc)
		c.Line(float64(r.CX())+34*s, cy+4*s, redX, cy+4*s, 2*s, cc)
		c.FillCircle(redX, cy+4*s, 5*s, hexColor(0xE5484D))
		c.FillCircle(blkX, cy+4*s, 5*s, hexColor(0x9AA4B2))
		u.T.Draw("power OFF — part out of circuit", r.X, r.Bottom()-u.Px(16), u.FB(11.5), th.AccentHi)
	}
}
