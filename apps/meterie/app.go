package meterie

// app.go: application state, frame loop, navigation and data flow.

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	PageLive = iota
	PageMeters
	PageLog
	PageAlarms
	PageMath
	PageWizards
	PageCamera
	PageLearn
	PageSettings
	pageCount
)

var pageInfo = []struct{ icon, name, sub string }{
	{icHome, "Live", "Readout, graph and statistics"},
	{icUSB, "Meters", "Connect, detect and manage meters"},
	{icRecord, "Logging", "Record readings to CSV with triggers"},
	{icWarn, "Alarms", "High/low limits and continuity beep"},
	{icCalc, "Math", "Power, resistance, energy and formulas"},
	{icBulb, "Wizards", "Guided tests for batteries, parts, LEDs, PSUs"},
	{icCamera, "Camera", "Read any meter's display with a webcam"},
	{icHelp, "What am I measuring?", "Beginner guide and safety"},
	{icSettings, "Settings", "Preferences, suite link and about"},
}

// Settings persisted between runs.
type Settings struct {
	Accent        int     `json:"accent"`
	SoundOn       bool    `json:"sound"`
	ContBeep      bool    `json:"contBeep"`
	ContThreshold float64 `json:"contThreshold"`
	APIEnabled    bool    `json:"api"`
	APIPort       int     `json:"apiPort"`
	LastMeter     string  `json:"lastMeter"`
	LastPort      string  `json:"lastPort"`
	DataDir       string  `json:"dataDir"`
	MaxSamples    int     `json:"maxSamples"`
	ShowWelcome   bool    `json:"welcome"`
	TeachOff      bool    `json:"teachOff"` // teaching mode is on unless switched off
	UIScale       float64 `json:"uiScale"`
	Theme         int     `json:"theme"`        // 0 dark, 1 light, 2 high contrast, 3 colour-blind safe
	ReduceMotion  bool    `json:"reduceMotion"` // animations jump straight to the end
}

func defaultSettings() Settings {
	return Settings{SoundOn: true, ContBeep: true, ContThreshold: 30, APIEnabled: true, APIPort: 47823, MaxSamples: 2_000_000, ShowWelcome: true, UIScale: 1}
}

var accentChoices = []struct {
	name    string
	acc, hi Color
}{
	{"Cyan", hexColor(0x3CC8F0), hexColor(0x7BDCF7)},
	{"Violet", hexColor(0x8B7CFF), hexColor(0xB1A8FF)},
	{"Amber", hexColor(0xFBBF24), hexColor(0xFCD34D)},
	{"Mint", hexColor(0x34D399), hexColor(0x6EE7B7)},
	{"Rose", hexColor(0xF472B6), hexColor(0xF9A8D4)},
}

// App is the whole application.
type App struct {
	ui       *UI
	page     int
	settings Settings

	channels   []*Channel
	nextChanID int
	devices    []*Device
	nextDevID  int
	mainChan   int
	markers    []Marker
	graph      *GraphView
	statsWin   int

	log    LogState
	alarms []*Alarm

	// pages
	mp       MetersPage
	cam      CameraState
	wiz      WizardState
	math     MathPage
	learnSel int

	modal       string
	modalMarker int64
	modalText   string

	sessionPath string
	lastFrame   time.Time
	sound       soundState
	frameCount  int

	apiMu      sync.Mutex
	apiState   apiSnapshot
	apiStarted bool

	manual     ManualState
	quit       bool
	demoSecond map[int]int
	expRange   int
	bg         *Canvas
	bgAccent   Color
}

func NewApp() *App {
	a := &App{ui: NewUI(), graph: NewGraphView(), settings: defaultSettings(), nextChanID: 1, nextDevID: 1, demoSecond: map[int]int{}}
	a.loadSettings()
	a.ui.Teach = !a.settings.TeachOff
	if a.ui.Teach {
		a.ui.teachIntro = 4
	}
	a.applyTheme()
	a.mp.init()
	a.cam.init()
	a.wiz.which = -1
	a.math.b = 1
	return a
}

func (a *App) applyAccent() {
	i := clampi(a.settings.Accent, 0, len(accentChoices)-1)
	th.Accent, th.AccentHi = accentChoices[i].acc, accentChoices[i].hi
	switch a.settings.Theme {
	case 1: // darker accents, so they read on white
		th.Accent, th.AccentHi = th.Accent.Darken(0.38), th.Accent.Darken(0.22)
	case 2:
		th.Accent, th.AccentHi = hexColor(0xFFD400), hexColor(0xFFE866)
	}
}

func configDir() string {
	d, err := os.UserConfigDir()
	if err != nil || d == "" {
		d = "."
	}
	return filepath.Join(d, "nrmluvdev", "Meterie")
}

func (a *App) loadSettings() {
	b, err := os.ReadFile(filepath.Join(configDir(), "settings.json"))
	if err == nil {
		s := defaultSettings()
		if json.Unmarshal(b, &s) == nil {
			a.settings = s
		}
	}
	// 47820 was the old default, which clashed with Wirie (Tinkerie ports: see PORTING.md).
	if a.settings.APIPort == 0 || a.settings.APIPort == 47820 {
		a.settings.APIPort = 47823
	}
	if a.settings.ContThreshold <= 0 {
		a.settings.ContThreshold = 30
	}
	if a.settings.MaxSamples < 10000 {
		a.settings.MaxSamples = 2_000_000
	}
}

func (a *App) saveSettings() {
	os.MkdirAll(configDir(), 0o755)
	b, _ := json.MarshalIndent(a.settings, "", "  ")
	os.WriteFile(filepath.Join(configDir(), "settings.json"), b, 0o644)
}

// ---------------------------------------------------------------------------
// Channels & devices

func (a *App) addChannel(name string, kind ChannelKind, source string, dev int) *Channel {
	ch := &Channel{ID: a.nextChanID, Name: name, Kind: kind, Source: source, DeviceID: dev, Visible: true}
	ch.Color = (a.nextChanID - 1) % len(channelPalette)
	a.nextChanID++
	a.channels = append(a.channels, ch)
	if a.mainChan == 0 {
		a.mainChan = ch.ID
	}
	return ch
}

func (a *App) chanByID(id int) *Channel {
	for _, c := range a.channels {
		if c.ID == id {
			return c
		}
	}
	return nil
}

func (a *App) devByID(id int) *Device {
	for _, d := range a.devices {
		if d.ID == id {
			return d
		}
	}
	return nil
}

func (a *App) mainChannel() *Channel {
	if c := a.chanByID(a.mainChan); c != nil {
		return c
	}
	if len(a.channels) > 0 {
		a.mainChan = a.channels[0].ID
		return a.channels[0]
	}
	return nil
}

// connectDevice creates a device + channel and starts it.
func (a *App) connectDevice(m *MeterModel, kind Transport, path, label string, cfg SerialCfg, bleAddr uint64) *Device {
	d := &Device{ID: a.nextDevID, Model: m, Kind: kind, Path: path, Label: label, Cfg: cfg, BLEAddr: bleAddr}
	a.nextDevID++
	name := "CH" + itoa(a.nextChanID)
	src := label
	if m != nil {
		src = m.Name()
		if path != "" && kind != TrBLE {
			src += " · " + shortPath(path)
		}
	}
	ch := a.addChannel(name, KindMeter, src, d.ID)
	d.ChannelID = ch.ID
	a.devices = append(a.devices, d)
	d.Start()
	if m != nil {
		a.settings.LastMeter = m.ID
		a.settings.LastPort = path
	}
	return d
}

func shortPath(p string) string {
	if strings.HasPrefix(p, `\\?\`) || len(p) > 24 {
		return "USB"
	}
	return p
}

// addDemo starts a simulated meter.
func (a *App) addDemo(sc DemoScenario) {
	d := &Device{ID: a.nextDevID, Kind: TrDemo, Label: "Demo meter", demo: &DemoSim{Scenario: sc}}
	a.nextDevID++
	ch := a.addChannel("CH"+itoa(a.nextChanID), KindMeter, "Demo · "+demoScenarioNames[sc], d.ID)
	d.ChannelID = ch.ID
	if sc == DemoSolar {
		ch.Name = "Panel V"
		ch2 := a.addChannel("Panel I", KindMeter, "Demo · "+demoScenarioNames[sc], d.ID)
		d.demo.hold = false
		a.demoSecond[d.ID] = ch2.ID
	}
	a.devices = append(a.devices, d)
	d.Start()
}

func (a *App) removeDevice(d *Device) {
	d.Stop()
	for i, x := range a.devices {
		if x == d {
			a.devices = append(a.devices[:i], a.devices[i+1:]...)
			break
		}
	}
}

func (a *App) removeChannel(id int) {
	for i, c := range a.channels {
		if c.ID == id {
			if c.DeviceID != 0 {
				if d := a.devByID(c.DeviceID); d != nil {
					// stop the device if no other channel uses it
					others := 0
					for _, o := range a.channels {
						if o.DeviceID == d.ID && o.ID != id {
							others++
						}
					}
					if others == 0 {
						a.removeDevice(d)
					}
				}
			}
			a.channels = append(a.channels[:i], a.channels[i+1:]...)
			break
		}
	}
	// remove dependent math channels' references
	if a.mainChan == id {
		a.mainChan = 0
	}
	for _, al := range a.alarms {
		if al.Chan == id {
			al.Enabled = false
		}
	}
}

// processInbox moves readings from device goroutines into channels.
func (a *App) processInbox() bool {
	items := drainInbox()
	if len(items) == 0 {
		return false
	}
	changed := map[int]bool{}
	for _, it := range items {
		d := a.devByID(it.dev)
		var ch *Channel
		if d != nil {
			cid := d.ChannelID
			if it.ch == -1 {
				cid = a.demoSecond[d.ID]
			} else if it.ch > 0 {
				cid = it.ch
			}
			ch = a.chanByID(cid)
		} else if it.ch > 0 {
			ch = a.chanByID(it.ch)
		}
		if ch == nil {
			continue
		}
		a.ingest(ch, it.r)
		changed[ch.ID] = true
	}
	a.updateMath(changed)
	return true
}

// ingest stores a reading and runs alarms/logging hooks.
func (a *App) ingest(ch *Channel, r Reading) {
	if r.T.IsZero() {
		r.T = time.Now()
	}
	ch.Add(r)
	if len(ch.Samples) > a.settings.MaxSamples {
		drop := len(ch.Samples) - a.settings.MaxSamples + a.settings.MaxSamples/10
		ch.Samples = append(ch.Samples[:0:0], ch.Samples[drop:]...)
	}
	a.checkAlarms(ch)
	a.log.onReading(a, ch, r)
	a.wiz.onReading(a, ch, r)
	a.updateAPI(ch)
}

func (a *App) updateMath(changed map[int]bool) {
	if len(changed) == 0 {
		return
	}
	for _, mc := range a.channels {
		if mc.Kind != KindMath || mc.Math == nil {
			continue
		}
		hit := false
		for _, id := range mc.Math.Inputs() {
			if changed[id] {
				hit = true
			}
		}
		if !hit {
			continue
		}
		vals := map[int]float64{}
		now := time.Now()
		var flags Flags
		for _, c := range a.channels {
			if c.HasLast && !c.Last.IsOL() && now.Sub(c.Last.T) < 5*time.Second {
				vals[c.ID] = c.DisplayValue(c.Last.Value)
				if changed[c.ID] {
					flags |= c.Last.Flags & (FAC | FDC)
				}
			}
		}
		v, ok := mc.Math.Eval(vals, now.UnixNano())
		if !ok {
			continue
		}
		q, unit := mc.Math.Quantity()
		r := Reading{T: now, Value: v, Q: q, Unit: unit, Counts: 0}
		if mc.Math.Preset == MathCharge && strings.EqualFold(unit, "Ah") {
			// show charge in mAh for readability
		}
		num, pu := SplitSI(v, unit, 5)
		r.Display, r.Prefix = num, strings.TrimSuffix(pu, unit)
		r.Flags = flags
		a.ingest(mc, r)
	}
}

// ---------------------------------------------------------------------------
// Frame

// Frame draws one frame. Returns true if another frame is wanted soon (animating).
func (a *App) Frame(c *Canvas, t TextEngine, in Input) bool {
	now := time.Now()
	dt := 1.0 / 60
	if !a.lastFrame.IsZero() {
		dt = now.Sub(a.lastFrame).Seconds()
	}
	a.lastFrame = now
	a.frameCount++
	a.processInbox()
	a.log.tick(a)
	a.updateSound()
	u := a.ui
	u.Begin(c, t, in, dt)
	a.handleShortcuts()

	// background with a subtle accent glow, cached per size/accent
	if a.bg == nil || a.bg.W != c.W || a.bg.H != c.H || a.bgAccent != th.Accent {
		a.bg = NewCanvas(c.W, c.H)
		a.bg.Clear(th.Bg)
		a.bg.RadialGlow(float64(c.W)*0.72, -float64(c.H)*0.1, float64(c.W)*0.6, th.Accent, 0.05)
		a.bgAccent = th.Accent
	}
	c.CopyFrom(a.bg)
	railW := u.Px(232)
	rail, main := Rect{0, 0, c.W, c.H}.CutLeft(railW)
	a.drawRail(rail)
	a.drawPage(main)
	a.drawModals()
	u.End()
	return u.Animating
}

func (a *App) handleShortcuts() {
	u := a.ui
	if u.focus != "" {
		return
	}
	for _, k := range u.In.Keys {
		switch {
		case u.In.Ctrl && k == KS:
			a.saveSessionDialog(false)
		case u.In.Ctrl && k == KO:
			a.openSessionDialog()
		case k == KSpace:
			a.graph.Paused = !a.graph.Paused
			a.graph.pauseAt = time.Now().UnixNano()
		case u.In.Ctrl && k == KM:
			a.openMarkerDialog(time.Now().UnixNano())
		case k == KF1:
			a.page = PageLearn
		case k == KF2:
			u.ToggleTeach()
			a.settings.TeachOff = !u.Teach
			a.saveSettings()
		}
	}
}

func (a *App) drawRail(r Rect) {
	u := a.ui
	c := u.C
	c.FillRect(r, th.Rail)
	c.VLine(r.Right()-1, 0, r.H, th.Border)
	// logo
	lx, ly := r.X+u.Px(22), r.Y+u.Px(24)
	drawLogo(c, float64(lx), float64(ly), u.Pf(34))
	u.T.Draw("Meterie", lx+u.Px(46), ly-u.Px(1), u.FH(20), th.Text)
	u.T.Draw("by "+AppAuthor, lx+u.Px(46), ly+u.Px(23), u.F(11.5), th.Sub)

	y := ly + u.Px(64)
	itemH := u.Px(42)
	// sliding active pill
	pos := u.Anim("rail#pos", float64(a.page), 14)
	pill := Rect{r.X + u.Px(12), y + int(pos*float64(itemH)), r.W - u.Px(24), itemH - u.Px(4)}
	c.FillRoundRect(pill, u.Pf(10), th.Accent.WithA(0.13))
	c.FillRoundRect(Rect{pill.X, pill.Y + u.Px(10), u.Px(3), pill.H - u.Px(20)}, u.Pf(2), th.Accent)
	for i, p := range pageInfo {
		ir := Rect{r.X + u.Px(12), y + i*itemH, r.W - u.Px(24), itemH - u.Px(4)}
		id := "rail" + itoa(i)
		h, cl := u.interact(id, ir)
		u.noteLabel(id, p.name)
		if cl {
			a.page = i
		}
		hv := u.Anim(id+"#h", b2f(h && a.page != i), 14)
		if hv > 0.01 {
			c.FillRoundRect(ir, u.Pf(10), th.Hover.WithA(0.6*hv))
		}
		col := lerpColor(th.Sub, th.Text, math.Max(hv, b2f(a.page == i)))
		icol := col
		if a.page == i {
			icol = th.AccentHi
		}
		u.Text(p.icon, Rect{ir.X + u.Px(14), ir.Y, u.Px(20), ir.H}, u.FI(15), icol, AlignCenter)
		u.Text(p.name, Rect{ir.X + u.Px(44), ir.Y, ir.W - u.Px(50), ir.H}, u.FB(13.5), col, AlignLeft)
		// badges
		badge := ""
		switch i {
		case PageMeters:
			n := 0
			for _, d := range a.devices {
				if _, _, conn, _, _, _, _ := d.Snapshot(); conn {
					n++
				}
			}
			if n > 0 {
				badge = itoa(n)
			}
		case PageLog:
			if a.log.Active() {
				badge = "REC"
			}
		case PageAlarms:
			if a.anyAlarmActive() {
				badge = "!"
			}
		}
		if badge != "" {
			bc := th.Accent
			if badge == "REC" || badge == "!" {
				bc = th.Bad
			}
			f := u.FB(10.5)
			bw := u.T.Measure(badge, f) + u.Px(12)
			br := Rect{ir.Right() - bw - u.Px(8), ir.CY() - u.Px(9), bw, u.Px(18)}
			c.FillRoundRect(br, u.Pf(9), bc.WithA(0.2))
			u.Text(badge, br, f, bc.Lighten(0.3), AlignCenter)
		}
	}
	// footer: session + suite
	fy := r.Bottom() - u.Px(92)
	c.HLine(r.X+u.Px(16), r.Right()-u.Px(16), fy, th.Border)
	sess := "Unsaved session"
	if a.sessionPath != "" {
		sess = filepath.Base(a.sessionPath)
	}
	u.Text(sess, Rect{r.X + u.Px(18), fy + u.Px(10), r.W - u.Px(36), u.Px(18)}, u.F(12), th.Sub, AlignLeft)
	bw := (r.W - u.Px(36) - u.Px(8)) / 3
	bx := r.X + u.Px(18)
	by := fy + u.Px(34)
	if u.IconButton("rail#new", Rect{bx, by, bw, u.Px(32)}, icAdd, "New session", false) {
		a.newSession()
	}
	if u.IconButton("rail#open", Rect{bx + bw + u.Px(4), by, bw, u.Px(32)}, icOpen, "Open session (Ctrl+O)", false) {
		a.openSessionDialog()
	}
	if u.IconButton("rail#save", Rect{bx + 2*(bw+u.Px(4)), by, bw, u.Px(32)}, icSave, "Save session (Ctrl+S)", false) {
		a.saveSessionDialog(false)
	}
}

// drawLogo draws the Meterie mark: a rounded meter body with a gauge needle.
func drawLogo(c *Canvas, x, y, s float64) {
	c.FillRoundRectF(x, y, s, s, s*0.28, th.Accent.Darken(0.55))
	c.FillRoundRectF(x+s*0.08, y+s*0.08, s*0.84, s*0.84, s*0.22, hexColor(0x0D1A22))
	c.Arc(x+s/2, y+s*0.62, s*0.28, s*0.09, math.Pi*1.1, math.Pi*1.9, th.Accent)
	c.Line(x+s/2, y+s*0.62, x+s*0.68, y+s*0.36, s*0.07, th.AccentHi)
	c.FillCircle(x+s/2, y+s*0.62, s*0.07, th.Text)
}

func (a *App) drawPage(r Rect) {
	u := a.ui
	// page header
	pad := u.Px(28)
	hdr := Rect{r.X + pad, r.Y + u.Px(20), r.W - 2*pad, u.Px(52)}
	pi := pageInfo[a.page]
	u.T.Draw(pi.name, hdr.X, hdr.Y, u.FH(24), th.Text)
	u.T.Draw(pi.sub, hdr.X, hdr.Y+u.Px(32), u.F(13), th.Sub)
	a.drawHeaderActions(Rect{hdr.X + hdr.W/2, hdr.Y + u.Px(4), hdr.W / 2, u.Px(36)})
	body := Rect{r.X + pad, hdr.Bottom() + u.Px(14), r.W - 2*pad, r.H - hdr.H - u.Px(14) - u.Px(20) - u.Px(20)}
	// page transition
	t := u.Anim("page#"+itoa(a.page), 1, 12)
	if t < 0.999 {
		off := int((1 - easeOut(t)) * u.Pf(16))
		body.Y += off
	}
	switch a.page {
	case PageLive:
		a.pageLive(body)
	case PageMeters:
		a.pageMeters(body)
	case PageLog:
		a.pageLog(body)
	case PageAlarms:
		a.pageAlarms(body)
	case PageMath:
		a.pageMath(body)
	case PageWizards:
		a.pageWizards(body)
	case PageCamera:
		a.pageCamera(body)
	case PageLearn:
		a.pageLearn(body)
	case PageSettings:
		a.pageSettings(body)
	}
	// reset other page anims so transitions replay
	for i := 0; i < pageCount; i++ {
		if i != a.page {
			u.Anim("page#"+itoa(i), 0, 1000)
		}
	}
}

func (a *App) drawHeaderActions(r Rect) {
	u := a.ui
	x := r.Right()
	bh := r.H
	add := func(id, icon, label string, style int, w int) bool {
		x -= w
		ok := u.Button(id, Rect{x, r.Y, w, bh}, icon, label, style)
		x -= u.Px(8)
		return ok
	}
	if a.log.Active() {
		if add("hdr#stoplog", icStop, "Stop logging", BtnDanger, u.Px(140)) {
			a.log.Stop(a, "stopped by you")
		}
	}
	if add("hdr#export", icSave, "Export", BtnSecondary, u.Px(104)) {
		a.modal = "export"
	}
	if add("hdr#marker", icFlag, "Marker", BtnSecondary, u.Px(104)) {
		a.openMarkerDialog(time.Now().UnixNano())
	}
	if a.page != PageMeters && len(a.devices) == 0 {
		if add("hdr#connect", icAdd, "Connect meter", BtnPrimary, u.Px(156)) {
			a.page = PageMeters
		}
	}
}

func (a *App) openMarkerDialog(t int64) {
	a.modal = "marker"
	a.modalMarker = t
	a.modalText = "Marker " + itoa(len(a.markers)+1)
	a.ui.focus = "modal#markertext"
	a.ui.selAll = true
	a.ui.caret = len([]rune(a.modalText))
}

func (a *App) drawModals() {
	u := a.ui
	switch a.modal {
	case "marker":
		cr, closed := u.Modal("m#marker", u.Px(460), u.Px(250), "Add a marker")
		u.T.Draw("Label shown on the graph at "+time.Unix(0, a.modalMarker).Format("15:04:05"), cr.X, cr.Y, u.F(13), th.Sub)
		_, enter := u.TextField("modal#markertext", Rect{cr.X, cr.Y + u.Px(28), cr.W, u.Px(38)}, &a.modalText, "e.g. load connected")
		by := cr.Bottom() - u.Px(38)
		if u.Button("m#add", Rect{cr.Right() - u.Px(140), by, u.Px(140), u.Px(38)}, icFlag, "Add marker", BtnPrimary) || enter {
			col := 1
			if ch := a.mainChannel(); ch != nil {
				col = ch.Color + 1
			}
			a.markers = append(a.markers, Marker{T: a.modalMarker, Text: a.modalText, Color: col})
			sort.Slice(a.markers, func(i, j int) bool { return a.markers[i].T < a.markers[j].T })
			a.modal = ""
			u.focus = ""
			u.Toast("Marker added", 1)
		}
		if u.Button("m#cancel", Rect{cr.Right() - u.Px(250), by, u.Px(100), u.Px(38)}, "", "Cancel", BtnGhost) || closed {
			a.modal = ""
			u.focus = ""
		}
		u.EndModal()
	case "export":
		a.drawExportModal()
	case "welcome":
		a.drawWelcomeModal()
	}
	if a.modal != "" {
		u.modalOpen = true
	}
}

func (a *App) drawWelcomeModal() {
	u := a.ui
	cr, closed := u.Modal("m#welcome", u.Px(620), u.Px(430), "Welcome to Meterie")
	y := cr.Y
	y += u.Wrap("Meterie turns your PC into a big, friendly multimeter display: live graphs, logging, alarms and guided tests for the meter you already own.", cr.X, y, cr.W, u.F(14), th.Text)
	y += u.Px(14)
	steps := []string{
		"1.  Plug in your meter's USB/Bluetooth link and open the Meters page — Meterie detects most cables automatically.",
		"2.  No PC link on your meter? Use Camera mode to read its display, or type readings in by hand.",
		"3.  Just exploring? Start a demo meter to try every feature without hardware.",
	}
	for _, s := range steps {
		y += u.Wrap(s, cr.X, y, cr.W, u.F(13.5), th.Sub) + u.Px(6)
	}
	y += u.Px(8)
	warn := Rect{cr.X, y, cr.W, u.Px(58)}
	u.C.FillRoundRect(warn, u.Pf(10), hexColor(0x2A1A12))
	u.Text(icWarn, Rect{warn.X + u.Px(10), warn.Y, u.Px(24), warn.H}, u.FI(16), th.Warn, AlignCenter)
	u.Wrap("Never measure mains or high voltage unless you are trained and your meter and leads are CAT-rated for it. See 'What am I measuring?' for the safety basics.", warn.X+u.Px(42), warn.Y+u.Px(9), warn.W-u.Px(52), u.F(12.5), hexColor(0xFFD8A8))
	by := cr.Bottom() - u.Px(40)
	if u.Button("w#demo", Rect{cr.X, by, u.Px(170), u.Px(40)}, icPlay, "Start demo meter", BtnSecondary) {
		a.addDemo(DemoBattery)
		a.modal = ""
		a.page = PageLive
	}
	if u.Button("w#connect", Rect{cr.Right() - u.Px(170), by, u.Px(170), u.Px(40)}, icUSB, "Connect a meter", BtnPrimary) {
		a.modal = ""
		a.page = PageMeters
	}
	if closed {
		a.modal = ""
	}
	a.settings.ShowWelcome = false
	u.EndModal()
}

// ---------------------------------------------------------------------------
// Alarms

type Alarm struct {
	Chan    int     `json:"chan"`
	UseHigh bool    `json:"useHigh"`
	High    float64 `json:"high"`
	UseLow  bool    `json:"useLow"`
	Low     float64 `json:"low"`
	Enabled bool    `json:"enabled"`
	Sound   bool    `json:"sound"`
	active  bool
	acked   bool
	since   time.Time
	lastVal float64
}

func (a *App) checkAlarms(ch *Channel) {
	for _, al := range a.alarms {
		if !al.Enabled || al.Chan != ch.ID || !ch.HasLast {
			continue
		}
		v := ch.DisplayValue(ch.Last.Value)
		hit := false
		if ch.Last.IsOL() {
			hit = al.UseHigh
		} else {
			if al.UseHigh && v > al.High {
				hit = true
			}
			if al.UseLow && v < al.Low {
				hit = true
			}
		}
		al.lastVal = v
		if hit && !al.active {
			al.active, al.acked, al.since = true, false, time.Now()
			a.ui.Toast("Alarm: "+ch.Name+" is "+FormatSI(v, ch.Last.Unit, 5), 3)
		} else if !hit && al.active {
			al.active = false
		}
	}
}

func (a *App) channelAlarmActive(ch *Channel) bool {
	if ch == nil {
		return false
	}
	for _, al := range a.alarms {
		if al.Enabled && al.Chan == ch.ID && al.active {
			return true
		}
	}
	return false
}

func (a *App) anyAlarmActive() bool {
	for _, al := range a.alarms {
		if al.Enabled && al.active {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Sound

type soundState struct {
	playing int // 0 none, 1 beep, 2 alarm
}

func (a *App) updateSound() {
	want := 0
	if a.settings.SoundOn {
		for _, al := range a.alarms {
			if al.Enabled && al.active && al.Sound && !al.acked {
				want = 2
			}
		}
		if want == 0 && a.settings.ContBeep {
			for _, ch := range a.channels {
				if ch.HasLast && ch.Last.Q == QContinuity && !ch.Last.IsOL() && ch.Last.Value < a.settings.ContThreshold &&
					time.Since(ch.Last.T) < 1500*time.Millisecond {
					want = 1
				}
			}
		}
	}
	if want != a.sound.playing {
		a.sound.playing = want
		switch want {
		case 0:
			platformStopSound()
		case 1:
			platformPlaySound(beepWAV(), true)
		case 2:
			platformPlaySound(alarmWAV(), true)
		}
	}
}

var (
	beepOnce, alarmOnce sync.Once
	beepData, alarmData []byte
)

func beepWAV() []byte {
	beepOnce.Do(func() { beepData = makeWAV(0.5, func(t float64) float64 { return 0.35 * math.Sin(2*math.Pi*2700*t) }) })
	return beepData
}

var shortOnce sync.Once
var shortData []byte

func shortBeepWAV() []byte {
	shortOnce.Do(func() {
		shortData = makeWAV(0.09, func(t float64) float64 { return 0.35 * math.Sin(2*math.Pi*2700*t) })
	})
	return shortData
}

func alarmWAV() []byte {
	alarmOnce.Do(func() {
		alarmData = makeWAV(1.0, func(t float64) float64 {
			f := 880.0
			if math.Mod(t, 0.5) > 0.25 {
				f = 660
			}
			env := 1.0
			if math.Mod(t, 0.25) > 0.21 {
				env = 0
			}
			return 0.4 * env * (math.Sin(2*math.Pi*f*t) + 0.3*math.Sin(2*math.Pi*2*f*t))
		})
	})
	return alarmData
}

// makeWAV renders a mono 16-bit PCM WAV.
func makeWAV(dur float64, fn func(t float64) float64) []byte {
	rate := 44100
	n := int(dur * float64(rate))
	data := make([]byte, 44+n*2)
	le32 := func(o int, v uint32) {
		data[o], data[o+1], data[o+2], data[o+3] = byte(v), byte(v>>8), byte(v>>16), byte(v>>24)
	}
	le16 := func(o int, v uint16) { data[o], data[o+1] = byte(v), byte(v>>8) }
	copy(data[0:], "RIFF")
	le32(4, uint32(36+n*2))
	copy(data[8:], "WAVEfmt ")
	le32(16, 16)
	le16(20, 1)
	le16(22, 1)
	le32(24, uint32(rate))
	le32(28, uint32(rate*2))
	le16(32, 2)
	le16(34, 16)
	copy(data[36:], "data")
	le32(40, uint32(n*2))
	for i := 0; i < n; i++ {
		v := fn(float64(i) / float64(rate))
		s := int16(clampf(v, -1, 1) * 32000)
		le16(44+i*2, uint16(s))
	}
	return data
}

// ---------------------------------------------------------------------------
// Manual entry

type ManualState struct {
	value string
	unit  int
	note  string
}

var manualUnits = []struct {
	label string
	q     Quantity
	unit  string
	f     Flags
}{
	{"V DC", QVoltage, "V", FDC}, {"V AC", QVoltage, "V", FAC}, {"A DC", QCurrent, "A", FDC}, {"A AC", QCurrent, "A", FAC},
	{"Ω", QResistance, "Ω", 0}, {"F", QCapacitance, "F", 0}, {"Hz", QFrequency, "Hz", 0}, {"°C", QTemperature, "°C", 0},
	{"%", QDuty, "%", 0}, {"W", QPower, "W", 0}, {"(no unit)", QUnitless, "", 0},
}

func (a *App) addManualChannel() *Channel {
	d := &Device{ID: a.nextDevID, Kind: TrManual, Label: "Manual entry"}
	a.nextDevID++
	ch := a.addChannel("Manual "+itoa(a.nextChanID), KindManual, "Typed in by hand", d.ID)
	d.ChannelID = ch.ID
	a.devices = append(a.devices, d)
	d.Start()
	return ch
}

func (a *App) submitManual(ch *Channel) bool {
	v, err := parseNumberSI(a.manual.value)
	if err != nil {
		a.ui.Toast("Type a number, e.g. 4.7k or 12.03", 2)
		return false
	}
	mu := manualUnits[clampi(a.manual.unit, 0, len(manualUnits)-1)]
	r := Reading{T: time.Now(), Value: v, Q: mu.q, Unit: mu.unit, Flags: mu.f, Note: a.manual.note}
	num, pu := SplitSI(v, mu.unit, 5)
	r.Display, r.Prefix = num, strings.TrimSuffix(pu, mu.unit)
	a.ingest(ch, r)
	a.manual.value = ""
	return true
}
