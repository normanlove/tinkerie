package meterie

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type MetersPage struct {
	tab      int
	mu       sync.Mutex
	ports    []PortInfo
	scanning bool
	scanned  bool
	probe    map[string]string
	search   string
	modelSel int
	trSel    int
	portSel  int
	baudSel  int
	bleOn    bool
	bleSel   int
	bleModel int
	demoSel  int
	showRaw  map[int]bool
	devModel map[int]*int // device ID -> dropdown index into modelSiblings (exact-model refiner)
	pending  []func(a *App)
}

func (m *MetersPage) init() {
	m.probe = map[string]string{}
	m.showRaw = map[int]bool{}
	m.devModel = map[int]*int{}
	m.modelSel = 0
}

var baudChoices = []int{0, 600, 1200, 2400, 4800, 9600, 19200, 38400, 57600, 115200}

func baudLabels() []string {
	out := []string{"Default"}
	for _, b := range baudChoices[1:] {
		out = append(out, itoa(b)+" baud")
	}
	return out
}

func (a *App) scanPorts() {
	m := &a.mp
	m.mu.Lock()
	if m.scanning {
		m.mu.Unlock()
		return
	}
	m.scanning = true
	m.mu.Unlock()
	go func() {
		ps := platformListPorts()
		m.mu.Lock()
		m.ports = ps
		m.scanning = false
		m.scanned = true
		m.mu.Unlock()
		appWake()
	}()
}

func (a *App) runPending() {
	m := &a.mp
	m.mu.Lock()
	p := m.pending
	m.pending = nil
	m.mu.Unlock()
	for _, f := range p {
		f(a)
	}
}

func (m *MetersPage) later(f func(a *App)) {
	m.mu.Lock()
	m.pending = append(m.pending, f)
	m.mu.Unlock()
	appWake()
}

func (m *MetersPage) setProbe(key, s string) {
	m.mu.Lock()
	m.probe[key] = s
	m.mu.Unlock()
	appWake()
}

// autoConnect detects which meter is on a port and connects it.
func (a *App) autoConnect(p PortInfo) {
	m := &a.mp
	key := p.Path
	switch p.Kind {
	case TrBU86X:
		a.connectDevice(meterByID("brymen-bm86x"), TrBU86X, p.Path, p.Name, SerialCfg{}, 0)
		a.ui.Toast("Connected Brymen BM86x (BU-86X cable)", 1)
		a.page = PageLive
		return
	case TrVictor:
		a.connectDevice(meterByID("victor-86c"), TrVictor, p.Path, p.Name, SerialCfg{}, 0)
		a.page = PageLive
		return
	}
	m.setProbe(key, "Detecting…")
	go func() {
		var mm *MeterModel
		var cfg SerialCfg
		var err error
		prog := func(s string) { m.setProbe(key, s) }
		if p.Kind == TrCH9325 {
			mm, cfg, err = ProbeHIDUART(p.Path, prog)
		} else if p.Kind == TrCP2110 {
			mm, cfg, err = ProbeCP2110(p.Path, prog)
		} else {
			mm, cfg, err = ProbeSerial(p.Path, prog)
		}
		if err != nil {
			m.setProbe(key, "Failed — "+err.Error())
			return
		}
		kind := p.Kind
		m.later(func(a *App) {
			// prefer the exact model you used before if it speaks the same protocol (e.g. HoldPeak HP-90EPC)
			if lm := meterByID(a.settings.LastMeter); lm != nil && lm.Proto == mm.Proto && lm.Has(kind) {
				mm = lm
			}
			m.setProbe(key, "Found "+mm.Name())
			a.connectDevice(mm, kind, p.Path, p.Name, cfg, 0)
			a.ui.Toast("Connected "+mm.Name()+" on "+shortPath(p.Path), 1)
			a.page = PageLive
		})
	}()
}

func (a *App) pageMeters(r Rect) {
	u := a.ui
	a.runPending()
	if !a.mp.scanned && !a.mp.scanning {
		a.scanPorts()
	}
	gap := u.Px(16)
	// connected devices column
	leftW := clampi(r.W*38/100, u.Px(340), u.Px(520))
	left, right := r.CutLeft(leftW)
	right.X += gap
	right.W -= gap
	a.drawConnected(left)
	a.drawAddMeter(right)
}

func (a *App) drawConnected(r Rect) {
	u := a.ui
	in := u.Card(r, "Connected")
	if len(a.devices) == 0 {
		y := in.Y + u.Px(10)
		u.Text(icUSB, Rect{in.X, y, in.W, u.Px(40)}, u.FI(30), th.Faint, AlignCenter)
		y += u.Px(52)
		u.Wrap("Nothing connected yet. Use Auto-detect on the right, pick your meter model, or start a demo meter.", in.X+u.Px(10), y, in.W-u.Px(20), u.F(13), th.Sub)
		return
	}
	off := u.BeginScroll("conn#scroll", in)
	y := in.Y - off
	for _, d := range append([]*Device(nil), a.devices...) {
		status, errS, conn, rx, frames, raw, last := d.Snapshot()
		showRaw := a.mp.showRaw[d.ID]
		h := u.Px(112)
		sibs := modelSiblings(d)
		if len(sibs) > 1 {
			h += u.Px(44)
		}
		if errS != "" {
			h += u.WrapHeight(errS, in.W-u.Px(28), u.F(12)) + u.Px(4)
		}
		if showRaw {
			h += u.Px(44)
		}
		cr := Rect{in.X, y, in.W - u.Px(8), h}
		u.C.FillRoundRect(cr, u.Pf(11), th.Card2)
		ch := a.chanByID(d.ChannelID)
		col := th.Accent
		if ch != nil {
			col = chanColor(ch.Color)
		}
		u.C.FillRoundRect(Rect{cr.X, cr.Y, u.Px(4), cr.H}, u.Pf(2), col)
		icon := map[Transport]string{TrSerial: icUSB, TrCH9325: icUSB, TrCP2110: icUSB, TrVictor: icUSB, TrBU86X: icUSB, TrBLE: icBT, TrDemo: icPlay, TrManual: icEdit, TrCamera: icCamera}[d.Kind]
		u.Text(icon, Rect{cr.X + u.Px(14), cr.Y + u.Px(12), u.Px(22), u.Px(22)}, u.FI(15), col, AlignCenter)
		title := d.Label
		if d.Model != nil {
			title = d.Model.Name()
		}
		if d.Kind == TrDemo && d.demo != nil {
			title = "Demo · " + demoScenarioNames[d.demo.Scenario]
		}
		u.Text(title, Rect{cr.X + u.Px(44), cr.Y + u.Px(10), cr.W - u.Px(150), u.Px(22)}, u.FB(14), th.Text, AlignLeft)
		sub := d.Kind.String()
		if d.Path != "" && d.Kind != TrBLE {
			sub += " · " + shortPath(d.Path)
		}
		if d.Kind == TrSerial || d.Kind == TrCH9325 || d.Kind == TrCP2110 {
			sub += " · " + d.Cfg.String()
		}
		if ch != nil {
			sub += " → " + ch.Name
		}
		u.Text(sub, Rect{cr.X + u.Px(44), cr.Y + u.Px(32), cr.W - u.Px(60), u.Px(18)}, u.F(12), th.Sub, AlignLeft)
		sc := th.Faint
		switch {
		case strings.HasPrefix(status, "Live"):
			sc = th.Good
		case errS != "":
			sc = th.Bad
		case conn:
			sc = th.Warn
		}
		u.C.FillCircle(float64(cr.X+u.Px(50)), float64(cr.Y+u.Px(65)), u.Pf(4), sc)
		stat := status
		if d.Kind != TrManual && d.Kind != TrDemo {
			stat += fmt.Sprintf("  ·  %d bytes, %d readings", rx, frames)
			if !last.IsZero() {
				stat += "  ·  " + fmtDur(time.Since(last)) + " ago"
			}
		}
		u.Text(stat, Rect{cr.X + u.Px(62), cr.Y + u.Px(56), cr.W - u.Px(80), u.Px(18)}, u.F(12), th.Text, AlignLeft)
		ey := cr.Y + u.Px(80)
		if len(sibs) > 1 {
			cur := 0
			for i, sm := range sibs {
				if sm == d.Model {
					cur = i
				}
			}
			sel := a.mp.devModel[d.ID]
			if sel == nil {
				sel = new(int)
				*sel = cur
				a.mp.devModel[d.ID] = sel
			}
			if *sel != cur && *sel >= 0 && *sel < len(sibs) { // picked in the dropdown last frame
				a.reconnectAs(d, sibs[*sel])
				u.Toast("Now decoding as "+sibs[*sel].Name(), 1)
			}
			*sel = indexOfModel(sibs, d.Model)
			names := make([]string, len(sibs))
			for i, sm := range sibs {
				names[i] = sm.Name()
			}
			u.T.Draw("Exact model", cr.X+u.Px(14), ey+u.Px(8), u.FB(12), th.Sub)
			u.Dropdown("dev#mdl"+itoa(d.ID), Rect{cr.X + u.Px(104), ey, min(cr.W-u.Px(118), u.Px(330)), u.Px(32)}, names, sel)
			ey += u.Px(44)
		}
		if errS != "" {
			ey += u.Wrap(errS, cr.X+u.Px(14), ey, cr.W-u.Px(28), u.F(12), th.Bad.Lighten(0.2)) + u.Px(4)
		}
		if showRaw {
			hex := ""
			for i, b := range raw {
				if i >= 32 {
					break
				}
				hex += fmt.Sprintf("%02X ", b)
			}
			if hex == "" {
				hex = "(no bytes yet)"
			}
			pr := Rect{cr.X + u.Px(14), ey, cr.W - u.Px(28), u.Px(36)}
			u.C.FillRoundRect(pr, u.Pf(6), th.Bg2)
			u.Text(hex, pr.InsetXY(u.Px(8), 0), u.FM(11.5), th.AccentHi, AlignLeft)
			ey += u.Px(44)
		}
		// actions
		bx := cr.Right() - u.Px(40)
		if u.IconButton("dev#rm"+itoa(d.ID), Rect{bx, cr.Y + u.Px(8), u.Px(30), u.Px(30)}, icDelete, "Disconnect and remove", false) {
			a.removeChannel(d.ChannelID)
			if d2 := a.demoSecond[d.ID]; d2 != 0 {
				a.removeChannel(d2)
			}
			a.removeDevice(d)
		}
		if d.Kind != TrManual && d.Kind != TrDemo {
			if u.IconButton("dev#raw"+itoa(d.ID), Rect{bx - u.Px(36), cr.Y + u.Px(8), u.Px(30), u.Px(30)}, icDiag, "Show raw bytes (diagnostics)", showRaw) {
				a.mp.showRaw[d.ID] = !showRaw
			}
		}
		_ = ey
		y += h + u.Px(10)
	}
	u.EndScroll("conn#scroll", in, y+off-in.Y)
}

var meterTabs = []string{"Auto-detect", "Choose model", "Bluetooth", "Demo & manual"}

func (a *App) drawAddMeter(r Rect) {
	u := a.ui
	in := u.Card(r, "Add a meter")
	u.Segmented("mt#tabs", Rect{in.X, in.Y, min(in.W, u.Px(560)), u.Px(34)}, meterTabs, &a.mp.tab)
	body := Rect{in.X, in.Y + u.Px(50), in.W, in.H - u.Px(50)}
	switch a.mp.tab {
	case 0:
		a.tabAutodetect(body)
	case 1:
		a.tabModels(body)
	case 2:
		a.tabBLE(body)
	case 3:
		a.tabDemo(body)
	}
}

func (a *App) tabAutodetect(r Rect) {
	u := a.ui
	m := &a.mp
	m.mu.Lock()
	ports := append([]PortInfo(nil), m.ports...)
	scanning := m.scanning
	probe := map[string]string{}
	for k, v := range m.probe {
		probe[k] = v
	}
	m.mu.Unlock()
	u.Wrap("Plug in your meter's USB cable, switch the meter on and enable its PC link, then press Scan. Meterie recognises UNI-T, Victor and CP2110 cables directly, and listens to plain USB-serial ports to work out which protocol your meter speaks.", r.X, r.Y, r.W-u.Px(150), u.F(13), th.Sub)
	label := "Scan"
	if scanning {
		label = "Scanning…"
		u.Ambient = true
	}
	if u.ButtonEx("ad#scan", Rect{r.Right() - u.Px(130), r.Y, u.Px(130), u.Px(36)}, icRefresh, label, BtnPrimary, !scanning) {
		a.scanPorts()
	}
	y := r.Y + u.Px(70)
	if len(ports) == 0 && !scanning {
		u.C.FillRoundRect(Rect{r.X, y, r.W, u.Px(84)}, u.Pf(10), th.Bg2)
		u.Wrap("No USB meter cables or COM ports found. Check the cable, install its driver (CH340, CP210x or PL2303 for most USB meters) — or try Bluetooth, Camera mode, or a demo meter.", r.X+u.Px(16), y+u.Px(14), r.W-u.Px(32), u.F(13), th.Faint)
		return
	}
	lr := Rect{r.X, y, r.W, r.Bottom() - y}
	off := u.BeginScroll("ad#list", lr)
	yy := y - off
	for i, p := range ports {
		rowH := u.Px(70)
		row := Rect{r.X, yy, r.W - u.Px(8), rowH}
		u.C.FillRoundRect(row, u.Pf(10), th.Card2)
		icon := icUSB
		u.Text(icon, Rect{row.X + u.Px(10), row.Y, u.Px(30), rowH}, u.FI(17), th.Accent, AlignCenter)
		name := p.Name
		if name == "" {
			name = p.Path
		}
		u.Text(name, Rect{row.X + u.Px(48), row.Y + u.Px(12), row.W - u.Px(260), u.Px(20)}, u.FB(13.5), th.Text, AlignLeft)
		hint := p.Hint
		if s, ok := probe[p.Path]; ok {
			hint = s
		}
		hc := th.Sub
		if strings.HasPrefix(hint, "Found") {
			hc = th.Good
		} else if strings.HasPrefix(hint, "Failed") {
			hc = th.Bad
		}
		u.Text(hint, Rect{row.X + u.Px(48), row.Y + u.Px(36), row.W - u.Px(260), u.Px(18)}, u.F(12), hc, AlignLeft)
		busy := strings.HasSuffix(probe[p.Path], "…")
		if busy {
			u.Ambient = true
		}
		if u.ButtonEx("ad#go"+itoa(i), Rect{row.Right() - u.Px(190), row.Y + u.Px(16), u.Px(176), u.Px(38)}, icSearch, "Detect & connect", BtnPrimary, !busy) {
			a.autoConnect(p)
		}
		yy += rowH + u.Px(8)
	}
	yy += u.Px(8)
	u.Wrap("Tip: if detection fails, use 'Choose model' — you can also force the baud rate there. Close the meter's own software first: only one program can use a COM port at a time.", r.X, yy, r.W-u.Px(8), u.F(12), th.Faint)
	yy += u.Px(40)
	u.EndScroll("ad#list", lr, yy+off-y)
}

func (a *App) filteredModels() []*MeterModel {
	q := strings.ToLower(strings.TrimSpace(a.mp.search))
	var out []*MeterModel
	for _, m := range meterCatalog {
		if q == "" || strings.Contains(strings.ToLower(m.Name()+" "+m.Also), q) {
			out = append(out, m)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Featured != out[j].Featured {
			return out[i].Featured
		}
		return false
	})
	return out
}

func (a *App) tabModels(r Rect) {
	u := a.ui
	m := &a.mp
	listW := clampi(r.W*45/100, u.Px(260), u.Px(380))
	lr, dr := r.CutLeft(listW)
	dr.X += u.Px(16)
	dr.W -= u.Px(16)
	u.TextField("mdl#search", Rect{lr.X, lr.Y, lr.W, u.Px(36)}, &m.search, "Search brand or model…")
	models := a.filteredModels()
	listR := Rect{lr.X, lr.Y + u.Px(46), lr.W, lr.H - u.Px(46)}
	off := u.BeginScroll("mdl#list", listR)
	y := listR.Y - off
	if m.modelSel >= len(models) {
		m.modelSel = 0
	}
	for i, mm := range models {
		ir := Rect{listR.X, y, listR.W - u.Px(8), u.Px(52)}
		if u.ListItem("mdl#"+mm.ID, ir, i == m.modelSel) {
			m.modelSel = i
			m.trSel, m.portSel, m.baudSel = 0, 0, 0
		}
		u.Text(mm.Name(), Rect{ir.X + u.Px(12), ir.Y + u.Px(8), ir.W - u.Px(24), u.Px(18)}, u.FB(13), th.Text, AlignLeft)
		sub := mm.Also
		if sub == "" {
			sub = strings.ToUpper(mm.Proto)
		}
		u.Text(sub, Rect{ir.X + u.Px(12), ir.Y + u.Px(28), ir.W - u.Px(24), u.Px(16)}, u.F(11.5), th.Sub, AlignLeft)
		if mm.Featured {
			u.C.FillCircle(float64(ir.Right()-u.Px(14)), float64(ir.Y+u.Px(16)), u.Pf(3), th.Accent)
		}
		y += u.Px(56)
	}
	// unsupported notes
	y += u.Px(10)
	u.T.Draw("NOT YET SUPPORTED OVER USB/BT", listR.X, y, u.FB(11), th.Faint)
	y += u.Px(22)
	for _, n := range unsupportedNotes {
		u.T.Draw(n.Name, listR.X, y, u.FB(12), th.Sub)
		y += u.Px(18)
		y += u.Wrap(n.Why, listR.X, y, listR.W-u.Px(12), u.F(11.5), th.Faint) + u.Px(8)
	}
	u.EndScroll("mdl#list", listR, y+off-listR.Y)

	if len(models) == 0 {
		u.Text("No match", dr, u.F(13), th.Faint, AlignCenter)
		return
	}
	mm := models[m.modelSel]
	y = dr.Y
	u.T.Draw(mm.Name(), dr.X, y, u.FH(20), th.Text)
	y += u.Px(32)
	sc := th.Good
	if mm.Support == SupCommunity {
		sc = th.Info
	} else if mm.Support == SupExperimental {
		sc = th.Warn
	}
	x := dr.X
	x += u.Chip(x, y, mm.Support, sc.Lighten(0.2), sc.WithA(0.15)) + u.Px(6)
	x += u.Chip(x, y, strings.ToUpper(mm.Proto), th.Sub, th.Card2) + u.Px(6)
	if mm.Counts > 0 {
		u.Chip(x, y, itoa(mm.Counts)+" counts", th.Sub, th.Card2)
	}
	y += u.Px(34)
	y += u.Wrap(mm.Connect, dr.X, y, dr.W, u.F(13), th.Sub) + u.Px(16)
	// transport
	trs := mm.Transports
	trNames := make([]string, len(trs))
	for i, t := range trs {
		trNames[i] = t.String()
	}
	if m.trSel >= len(trs) {
		m.trSel = 0
	}
	if len(trs) > 1 {
		u.T.Draw("Connection", dr.X, y, u.FB(12), th.Sub)
		y += u.Px(20)
		u.Segmented("mdl#tr", Rect{dr.X, y, dr.W, u.Px(32)}, trNames, &m.trSel)
		y += u.Px(44)
	}
	tr := trs[m.trSel]
	m.mu.Lock()
	var ports []PortInfo
	for _, p := range m.ports {
		if p.Kind == tr || (tr == TrSerial && p.Kind == TrSerial) {
			ports = append(ports, p)
		}
	}
	m.mu.Unlock()
	if tr == TrBLE {
		u.Wrap("This meter connects over Bluetooth — use the Bluetooth tab to scan for it.", dr.X, y, dr.W, u.F(13), th.Text)
		y += u.Px(30)
		if u.Button("mdl#toble", Rect{dr.X, y, u.Px(200), u.Px(38)}, icBT, "Go to Bluetooth", BtnPrimary) {
			m.tab = 2
		}
		return
	}
	u.T.Draw("Port", dr.X, y, u.FB(12), th.Sub)
	y += u.Px(20)
	names := []string{}
	for _, p := range ports {
		n := p.Name
		if tr == TrSerial && !strings.Contains(n, p.Path) {
			n = p.Path + " — " + n
		}
		names = append(names, n)
	}
	if len(names) == 0 {
		names = []string{"(no matching port found — rescan)"}
	}
	if m.portSel >= len(names) {
		m.portSel = 0
	}
	pr := Rect{dr.X, y, dr.W - u.Px(46), u.Px(36)}
	u.Dropdown("mdl#port", pr, names, &m.portSel)
	if u.IconButton("mdl#rescan", Rect{pr.Right() + u.Px(8), y, u.Px(36), u.Px(36)}, icRefresh, "Rescan ports", false) {
		a.scanPorts()
	}
	y += u.Px(48)
	if tr == TrSerial || tr == TrCH9325 {
		u.T.Draw("Baud rate (only change if the default doesn't work)", dr.X, y, u.FB(12), th.Sub)
		y += u.Px(20)
		u.Dropdown("mdl#baud", Rect{dr.X, y, u.Px(200), u.Px(36)}, baudLabels(), &m.baudSel)
		u.Text("Default: "+mm.Serial.String(), Rect{dr.X + u.Px(214), y, dr.W - u.Px(214), u.Px(36)}, u.F(12), th.Faint, AlignLeft)
		y += u.Px(52)
	}
	ok := len(ports) > 0
	if u.ButtonEx("mdl#connect", Rect{dr.X, y, u.Px(200), u.Px(42)}, icPlay, "Connect", BtnPrimary, ok) {
		p := ports[m.portSel]
		cfg := mm.Serial
		if m.baudSel > 0 {
			cfg.Baud = baudChoices[m.baudSel]
		}
		a.connectDevice(mm, tr, p.Path, p.Name, cfg, 0)
		u.Toast("Connecting "+mm.Name()+"…", 0)
		a.page = PageLive
	}
}

func (a *App) tabBLE(r Rect) {
	u := a.ui
	m := &a.mp
	u.Wrap("Switch on your meter's Bluetooth. Meterie talks to it directly with Windows' built-in Bluetooth LE — don't pair it in Windows Settings first.", r.X, r.Y, r.W-u.Px(170), u.F(13), th.Sub)
	label := "Start scan"
	if m.bleOn {
		label = "Stop scan"
		u.Ambient = true
	}
	if u.Button("ble#scan", Rect{r.Right() - u.Px(150), r.Y, u.Px(150), u.Px(36)}, icBT, label, map[bool]int{true: BtnSecondary, false: BtnPrimary}[m.bleOn]) {
		m.bleOn = !m.bleOn
		if err := platformBLEScan(m.bleOn); err != nil {
			m.bleOn = false
			u.Toast("Bluetooth: "+err.Error(), 3)
		}
	}
	y := r.Y + u.Px(64)
	res := platformBLEResults()
	sort.Slice(res, func(i, j int) bool {
		mi, mj := bleNameMatch(res[i].Name) != nil, bleNameMatch(res[j].Name) != nil
		if mi != mj {
			return mi
		}
		return res[i].RSSI > res[j].RSSI
	})
	if len(res) == 0 {
		msg := "Press Start scan. Meters usually advertise as BDM, B35T, B41T, OW18E …"
		if m.bleOn {
			msg = "Scanning… make sure the meter's Bluetooth icon is on."
		}
		u.C.FillRoundRect(Rect{r.X, y, r.W, u.Px(60)}, u.Pf(10), th.Bg2)
		u.Text(msg, Rect{r.X + u.Px(16), y, r.W - u.Px(32), u.Px(60)}, u.F(13), th.Faint, AlignLeft)
		y += u.Px(76)
	}
	protoOpts := []string{"Auto (by name)", "OWON 6-byte (B35T+/B41T+/OW18E)", "BLE-serial FS9721 stream", "BLE-serial FS9922 stream", "ANENG / ZOYI / BSIDE (Bluetooth DMM)", "Raw (diagnostics only)"}
	for i, p := range res {
		if i >= 8 {
			break
		}
		row := Rect{r.X, y, r.W, u.Px(62)}
		u.C.FillRoundRect(row, u.Pf(10), th.Card2)
		name := p.Name
		if name == "" {
			name = "(unnamed device)"
		}
		mm := bleNameMatch(p.Name)
		u.Text(icBT, Rect{row.X + u.Px(10), row.Y, u.Px(30), row.H}, u.FI(16), th.Accent, AlignCenter)
		u.Text(name, Rect{row.X + u.Px(48), row.Y + u.Px(10), row.W / 2, u.Px(20)}, u.FB(13.5), th.Text, AlignLeft)
		sub := fmt.Sprintf("%s · %d dBm", p.Path, p.RSSI)
		if mm != nil {
			sub = "Looks like " + mm.Name() + " · " + sub
		}
		u.Text(sub, Rect{row.X + u.Px(48), row.Y + u.Px(34), row.W/2 + u.Px(60), u.Px(18)}, u.F(12), th.Sub, AlignLeft)
		if u.Button("ble#c"+itoa(i), Rect{row.Right() - u.Px(126), row.Y + u.Px(12), u.Px(112), u.Px(38)}, icPlay, "Connect", BtnPrimary) {
			model := mm
			switch m.bleModel {
			case 1:
				model = meterByID("owon-ble")
			case 2:
				model = cloneBLE(meterByID("generic-fs9721"))
			case 3:
				model = cloneBLE(meterByID("generic-fs9922"))
			case 4:
				model = meterByID("zoyi-ble")
			case 5:
				model = &MeterModel{ID: "raw-ble", Brand: "Bluetooth", Model: "raw", Proto: "raw", Transports: []Transport{TrBLE}}
			}
			if model == nil {
				model = &MeterModel{ID: "raw-ble", Brand: "Bluetooth", Model: name + " (raw)", Proto: "raw", Transports: []Transport{TrBLE}}
				u.Toast("Unknown meter — connecting in raw mode; see the Meters page diagnostics", 2)
			}
			platformBLEScan(false)
			m.bleOn = false
			a.connectDevice(model, TrBLE, p.Path, name, SerialCfg{}, p.BLEAddr)
			a.page = PageLive
		}
		y += u.Px(70)
	}
	y += u.Px(6)
	u.T.Draw("Protocol", r.X, y+u.Px(9), u.FB(12), th.Sub)
	u.Dropdown("ble#proto", Rect{r.X + u.Px(80), y, u.Px(320), u.Px(36)}, protoOpts, &m.bleModel)
	y += u.Px(52)
	u.Wrap("Auto picks the protocol from the meter's name: OWON meters advertise as 'BDM', ANENG / ZOYI / BSIDE meters as 'Bluetooth DMM'. Raw mode shows the bytes on the Meters page diagnostics.", r.X, y, r.W, u.F(12), th.Faint)
}

// modelSiblings lists catalogue models that speak the same protocol over the same
// transport, so an auto-detected "generic FS9721" can be refined to e.g. HoldPeak HP-90EPC.
func modelSiblings(d *Device) []*MeterModel {
	if d.Model == nil || d.Kind == TrDemo || d.Kind == TrManual || d.Kind == TrCamera {
		return nil
	}
	var out []*MeterModel
	for _, m := range meterCatalog {
		if m.Proto == d.Model.Proto && m.Has(d.Kind) {
			out = append(out, m)
		}
	}
	found := false
	for _, m := range out {
		if m == d.Model {
			found = true
		}
	}
	if !found {
		out = append([]*MeterModel{d.Model}, out...)
	}
	return out
}

func indexOfModel(list []*MeterModel, m *MeterModel) int {
	for i, x := range list {
		if x == m {
			return i
		}
	}
	return 0
}

// reconnectAs restarts a device with a different (same-protocol) model.
func (a *App) reconnectAs(d *Device, m *MeterModel) {
	d.Stop()
	d.Model = m
	if ch := a.chanByID(d.ChannelID); ch != nil {
		ch.Source = m.Name()
		if d.Path != "" && d.Kind != TrBLE {
			ch.Source += " · " + shortPath(d.Path)
		}
	}
	a.settings.LastMeter = m.ID
	a.saveSettings()
	d.Start()
}

func cloneBLE(m *MeterModel) *MeterModel {
	c := *m
	c.Model += " over BLE serial"
	c.BLEService, c.BLEChar = "", "" // auto: first notify characteristic
	return &c
}

func (a *App) tabDemo(r Rect) {
	u := a.ui
	m := &a.mp
	half := (r.W - u.Px(16)) / 2
	d := Rect{r.X, r.Y, half, r.H}
	u.T.Draw("Demo meter", d.X, d.Y, u.FH(16), th.Text)
	y := d.Y + u.Px(28)
	y += u.Wrap("A simulated meter so you can explore graphs, logging, alarms, math and the wizards without hardware.", d.X, y, d.W, u.F(13), th.Sub) + u.Px(12)
	for i, n := range demoScenarioNames {
		ir := Rect{d.X, y, d.W, u.Px(34)}
		if u.ListItem("demo#"+itoa(i), ir, m.demoSel == i) {
			m.demoSel = i
		}
		u.Text(n, ir.InsetXY(u.Px(12), 0), u.F(13), th.Text, AlignLeft)
		y += u.Px(36)
	}
	y += u.Px(8)
	if u.Button("demo#start", Rect{d.X, y, u.Px(190), u.Px(40)}, icPlay, "Start demo", BtnPrimary) {
		a.addDemo(DemoScenario(m.demoSel))
		a.page = PageLive
	}
	mr := Rect{r.X + half + u.Px(16), r.Y, half, r.H}
	u.T.Draw("Manual entry", mr.X, mr.Y, u.FH(16), th.Text)
	y = mr.Y + u.Px(28)
	y += u.Wrap("Log readings by hand from any meter — type the value, pick the unit and press Enter. They go on the timeline like any other channel.", mr.X, y, mr.W, u.F(13), th.Sub) + u.Px(12)
	if u.Button("man#add", Rect{mr.X, y, u.Px(220), u.Px(40)}, icEdit, "Add manual channel", BtnSecondary) {
		ch := a.addManualChannel()
		a.mainChan = ch.ID
		a.page = PageLive
		u.focus = "manual#v"
	}
	y += u.Px(66)
	u.T.Draw("Camera mode", mr.X, y, u.FH(16), th.Text)
	y += u.Px(28)
	y += u.Wrap("No PC link at all? Point a webcam (or a phone running an IP-camera app) at the display and Meterie reads the 7-segment digits.", mr.X, y, mr.W, u.F(13), th.Sub) + u.Px(12)
	if u.Button("cam#go", Rect{mr.X, y, u.Px(220), u.Px(40)}, icCamera, "Open Camera mode", BtnSecondary) {
		a.page = PageCamera
	}
}
