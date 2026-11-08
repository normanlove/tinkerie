package meterie

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// CSV logging with start/stop triggers

var logIntervals = []struct {
	label string
	secs  float64
}{{"Every reading", 0}, {"0.1 s", 0.1}, {"0.2 s", 0.2}, {"0.5 s", 0.5}, {"1 s", 1}, {"2 s", 2}, {"5 s", 5}, {"10 s", 10}, {"30 s", 30}, {"1 min", 60}, {"5 min", 300}}

var startModes = []string{"Immediately", "When a channel drops below…", "When a channel rises above…"}
var stopModes = []string{"When I press Stop", "After a duration", "After N rows", "When a channel drops below…", "When a channel rises above…"}

type LogState struct {
	Path      string
	Interval  int
	StartMode int
	StartChan int
	StartVal  float64
	StopMode  int
	StopChan  int
	StopVal   float64
	StopMins  float64
	StopRows  float64
	Exclude   map[int]bool

	state     int // 0 idle, 1 armed, 2 recording
	f         *os.File
	w         *bufio.Writer
	rows      int
	started   time.Time
	armedAt   time.Time
	nextRow   time.Time
	lastFlush time.Time
	cols      []int
	err       string
	lastStop  string
	prevVals  map[int]float64
}

func (l *LogState) Active() bool { return l.state != 0 }

func (l *LogState) defaults() {
	if l.Exclude == nil {
		l.Exclude = map[int]bool{}
	}
	if l.Interval == 0 && l.Path == "" {
		l.Interval = 4
	}
	if l.StopMins == 0 {
		l.StopMins = 10
	}
	if l.StopRows == 0 {
		l.StopRows = 1000
	}
}

func defaultDataDir() string {
	h, _ := os.UserHomeDir()
	d := filepath.Join(h, "Documents", "Meterie")
	return d
}

func (a *App) dataDir() string {
	if a.settings.DataDir != "" {
		return a.settings.DataDir
	}
	return defaultDataDir()
}

func (l *LogState) Start(a *App) {
	l.defaults()
	if l.Path == "" {
		os.MkdirAll(a.dataDir(), 0o755)
		l.Path = filepath.Join(a.dataDir(), "meterie-"+time.Now().Format("2006-01-02_15-04-05")+".csv")
	}
	os.MkdirAll(filepath.Dir(l.Path), 0o755)
	f, err := os.Create(l.Path)
	if err != nil {
		l.err = err.Error()
		a.ui.Toast("Can't create log file: "+err.Error(), 3)
		return
	}
	l.f, l.w, l.rows, l.err = f, bufio.NewWriter(f), 0, ""
	l.cols = l.cols[:0]
	for _, ch := range a.channels {
		if !l.Exclude[ch.ID] {
			l.cols = append(l.cols, ch.ID)
		}
	}
	fmt.Fprintf(l.w, "# Meterie %s by %s — log started %s\n", AppVersion, AppAuthor, time.Now().Format(time.RFC3339))
	hdr := []string{"timestamp", "unix_ms", "elapsed_s"}
	for _, id := range l.cols {
		ch := a.chanByID(id)
		n := csvEsc(ch.Name + " (" + ch.Source + ")")
		hdr = append(hdr, n, csvEsc(ch.Name+" unit"), csvEsc(ch.Name+" mode"))
	}
	l.w.WriteString(strings.Join(hdr, ",") + "\n")
	l.prevVals = map[int]float64{}
	l.armedAt = time.Now()
	if l.StartMode == 0 {
		l.begin(a)
	} else {
		l.state = 1
		a.ui.Toast("Logging armed — waiting for the start trigger", 0)
	}
}

func (l *LogState) begin(a *App) {
	l.state = 2
	l.started = time.Now()
	l.nextRow = l.started
	a.ui.Toast("Logging to "+filepath.Base(l.Path), 1)
	if !a.markersHasNear(l.started.UnixNano()) {
		a.markers = append(a.markers, Marker{T: l.started.UnixNano(), Text: "REC", Color: 7})
	}
}

func (a *App) markersHasNear(t int64) bool {
	for _, m := range a.markers {
		if math.Abs(float64(m.T-t)) < 5e8 {
			return true
		}
	}
	return false
}

func (l *LogState) Stop(a *App, reason string) {
	if l.state == 0 {
		return
	}
	if l.w != nil {
		l.w.Flush()
	}
	if l.f != nil {
		l.f.Close()
	}
	if l.state == 2 {
		a.markers = append(a.markers, Marker{T: time.Now().UnixNano(), Text: "stop", Color: 7})
	}
	l.lastStop = fmt.Sprintf("%d rows saved to %s (%s)", l.rows, filepath.Base(l.Path), reason)
	a.ui.Toast("Logging stopped — "+itoa(l.rows)+" rows", 1)
	l.state, l.f, l.w = 0, nil, nil
	l.Path = ""
}

func csvEsc(s string) string {
	if strings.ContainsAny(s, ",\"\n") {
		return "\"" + strings.ReplaceAll(s, "\"", "\"\"") + "\""
	}
	return s
}

func (l *LogState) writeRow(a *App, now time.Time, only int) {
	if l.w == nil {
		return
	}
	cells := []string{now.Format("2006-01-02 15:04:05.000"), itoa64(now.UnixMilli()), fmt.Sprintf("%.3f", now.Sub(l.started).Seconds())}
	for _, id := range l.cols {
		ch := a.chanByID(id)
		if ch == nil || !ch.HasLast || (only != 0 && id != only) || now.Sub(ch.Last.T) > 10*time.Second {
			cells = append(cells, "", "", "")
			continue
		}
		v := ch.DisplayValue(ch.Last.Value)
		val := FormatValue(v)
		if ch.Last.IsOL() {
			val = "OL"
		}
		cells = append(cells, val, ch.Last.Unit, csvEsc(ch.Last.ModeLabel()))
	}
	l.w.WriteString(strings.Join(cells, ",") + "\n")
	l.rows++
	if time.Since(l.lastFlush) > time.Second {
		l.w.Flush()
		l.lastFlush = time.Now()
	}
}

func itoa64(v int64) string { return fmt.Sprintf("%d", v) }

// trigger evaluates a crossing condition; mode 1 below, 2 above (or stop modes 3/4).
func (l *LogState) crossed(a *App, chID int, below bool, thr float64) bool {
	ch := a.chanByID(chID)
	if ch == nil || !ch.HasLast || ch.Last.IsOL() {
		return false
	}
	v := ch.DisplayValue(ch.Last.Value)
	if below {
		return v < thr
	}
	return v > thr
}

func (l *LogState) onReading(a *App, ch *Channel, r Reading) {
	if l.state == 2 && logIntervals[l.Interval].secs == 0 {
		for _, id := range l.cols {
			if id == ch.ID {
				l.writeRow(a, r.T, ch.ID)
			}
		}
	}
}

func (l *LogState) tick(a *App) {
	switch l.state {
	case 1:
		if l.crossed(a, l.StartChan, l.StartMode == 1, l.StartVal) {
			l.begin(a)
		}
	case 2:
		now := time.Now()
		if iv := logIntervals[l.Interval].secs; iv > 0 && !now.Before(l.nextRow) {
			l.writeRow(a, now, 0)
			l.nextRow = l.nextRow.Add(time.Duration(iv * 1e9))
			if l.nextRow.Before(now) {
				l.nextRow = now.Add(time.Duration(iv * 1e9))
			}
		}
		switch l.StopMode {
		case 1:
			if now.Sub(l.started) >= time.Duration(l.StopMins*60e9) {
				l.Stop(a, "duration reached")
			}
		case 2:
			if l.rows >= int(l.StopRows) {
				l.Stop(a, "row count reached")
			}
		case 3, 4:
			if l.crossed(a, l.StopChan, l.StopMode == 3, l.StopVal) {
				l.Stop(a, "stop trigger")
			}
		}
	}
}

func (a *App) channelNames() ([]string, []int) {
	var names []string
	var ids []int
	for _, ch := range a.channels {
		names = append(names, ch.Name+" · "+ch.Source)
		ids = append(ids, ch.ID)
	}
	if len(names) == 0 {
		names = []string{"(no channels)"}
		ids = []int{0}
	}
	return names, ids
}

func idxOf(ids []int, id int) int {
	for i, x := range ids {
		if x == id {
			return i
		}
	}
	return 0
}

func (a *App) pageLog(r Rect) {
	u := a.ui
	l := &a.log
	l.defaults()
	gap := u.Px(16)
	left, right := r.CutLeft(r.W*58/100 - gap/2)
	right.X += gap
	right.W -= gap
	in := u.Card(left, "CSV logging")
	y := in.Y
	names, ids := a.channelNames()
	lab := func(s string) {
		u.T.Draw(s, in.X, y, u.FB(12), th.Sub)
		y += u.Px(20)
	}
	disabled := l.Active()
	lab("File")
	path := l.Path
	if path == "" {
		path = filepath.Join(a.dataDir(), "meterie-<date-time>.csv")
	}
	pr := Rect{in.X, y, in.W - u.Px(120), u.Px(36)}
	u.C.FillRoundRect(pr, u.Pf(9), th.Bg2)
	u.Text(path, pr.InsetXY(u.Px(11), 0), u.F(12.5), th.Text, AlignLeft)
	if u.ButtonEx("log#choose", Rect{pr.Right() + u.Px(8), y, u.Px(112), u.Px(36)}, icFolder, "Choose…", BtnSecondary, !disabled) {
		if p, ok := platformSaveDialog("Save log as", filepath.Join(a.dataDir(), "meterie-log.csv"), "CSV files (*.csv)", "csv"); ok {
			l.Path = p
		}
	}
	y += u.Px(50)
	lab("Sample interval")
	iv := make([]string, len(logIntervals))
	for i, x := range logIntervals {
		iv[i] = x.label
	}
	u.Dropdown("log#iv", Rect{in.X, y, u.Px(220), u.Px(36)}, iv, &l.Interval)
	u.Text("One row per interval with every channel's latest value (or a row per reading).", Rect{in.X + u.Px(234), y, in.W - u.Px(234), u.Px(36)}, u.F(12), th.Faint, AlignLeft)
	y += u.Px(52)
	lab("Start")
	u.Dropdown("log#start", Rect{in.X, y, u.Px(260), u.Px(36)}, startModes, &l.StartMode)
	if l.StartMode > 0 {
		ci := idxOf(ids, l.StartChan)
		if u.Dropdown("log#startch", Rect{in.X + u.Px(270), y, u.Px(200), u.Px(36)}, names, &ci) || l.StartChan == 0 {
			l.StartChan = ids[ci]
		}
		u.NumberField("log#startv", Rect{in.X + u.Px(480), y, min(u.Px(150), in.W-u.Px(480)), u.Px(36)}, &l.StartVal, unitOf(a, l.StartChan))
	}
	y += u.Px(52)
	lab("Stop")
	u.Dropdown("log#stop", Rect{in.X, y, u.Px(260), u.Px(36)}, stopModes, &l.StopMode)
	switch l.StopMode {
	case 1:
		u.NumberField("log#mins", Rect{in.X + u.Px(270), y, u.Px(150), u.Px(36)}, &l.StopMins, "min")
	case 2:
		u.NumberField("log#rows", Rect{in.X + u.Px(270), y, u.Px(150), u.Px(36)}, &l.StopRows, "rows")
	case 3, 4:
		ci := idxOf(ids, l.StopChan)
		if u.Dropdown("log#stopch", Rect{in.X + u.Px(270), y, u.Px(200), u.Px(36)}, names, &ci) || l.StopChan == 0 {
			l.StopChan = ids[ci]
		}
		u.NumberField("log#stopv", Rect{in.X + u.Px(480), y, min(u.Px(150), in.W-u.Px(480)), u.Px(36)}, &l.StopVal, unitOf(a, l.StopChan))
	}
	y += u.Px(52)
	lab("Channels to log")
	x := in.X
	for _, ch := range a.channels {
		on := !l.Exclude[ch.ID]
		w := u.T.Measure(ch.Name, u.F(13.5)) + u.Px(40)
		if x+w > in.Right() {
			x = in.X
			y += u.Px(30)
		}
		if u.Checkbox("log#ch"+itoa(ch.ID), Rect{x, y, w, u.Px(26)}, &on, ch.Name) && !disabled {
			l.Exclude[ch.ID] = !on
		}
		x += w + u.Px(12)
	}
	if len(a.channels) == 0 {
		u.T.Draw("Connect a meter first.", in.X, y+u.Px(4), u.F(13), th.Faint)
	}
	y += u.Px(44)
	// start/stop
	if !l.Active() {
		if u.ButtonEx("log#go", Rect{in.X, y, u.Px(200), u.Px(44)}, icRecord, "Start logging", BtnDanger, len(a.channels) > 0) {
			l.Start(a)
		}
	} else {
		if u.Button("log#stopbtn", Rect{in.X, y, u.Px(200), u.Px(44)}, icStop, "Stop logging", BtnSecondary) {
			l.Stop(a, "stopped by you")
		}
	}
	// status card
	sin := u.Card(right, "Status")
	sy := sin.Y
	state := []string{"Idle", "Armed — waiting for trigger", "Recording"}[l.state]
	col := []Color{th.Faint, th.Warn, th.Bad}[l.state]
	pulse := 0.5 + 0.5*math.Sin(float64(time.Now().UnixMilli())/300)
	if l.state == 2 {
		u.C.FillCircle(float64(sin.X+u.Px(8)), float64(sy+u.Px(12)), u.Pf(6), col.WithA(0.5+0.5*pulse))
		u.Ambient = true
	} else {
		u.C.FillCircle(float64(sin.X+u.Px(8)), float64(sy+u.Px(12)), u.Pf(6), col)
	}
	u.T.Draw(state, sin.X+u.Px(24), sy, u.FH(17), th.Text)
	sy += u.Px(40)
	kv := func(k, v string) {
		u.Text(k, Rect{sin.X, sy, sin.W / 2, u.Px(24)}, u.F(13), th.Sub, AlignLeft)
		u.Text(v, Rect{sin.X + sin.W/3, sy, sin.W * 2 / 3, u.Px(24)}, u.FB(13), th.Text, AlignRight)
		sy += u.Px(30)
	}
	if l.state == 2 {
		kv("Rows written", itoa(l.rows))
		kv("Running for", fmtDur(time.Since(l.started)))
		kv("File", filepath.Base(l.Path))
		if st, err := os.Stat(l.Path); err == nil {
			kv("Size", fmt.Sprintf("%.1f kB", float64(st.Size())/1024))
		}
	} else if l.state == 1 {
		kv("Armed for", fmtDur(time.Since(l.armedAt)))
		kv("Trigger", startModes[l.StartMode]+" "+FormatSI(l.StartVal, unitOf(a, l.StartChan), 4))
	}
	if l.lastStop != "" {
		sy += u.Px(6)
		sy += u.Wrap("Last log: "+l.lastStop, sin.X, sy, sin.W, u.F(12.5), th.Sub)
	}
	if l.err != "" {
		sy += u.Wrap(l.err, sin.X, sy, sin.W, u.F(12.5), th.Bad)
	}
	sy += u.Px(20)
	u.Wrap("Tip: everything on the timeline is kept in memory too — use Export to save all of it as CSV, a suite timeline (for Scopie / Serialie / Logicie), or a .meterie session you can reopen later.", sin.X, sy, sin.W, u.F(12.5), th.Faint)
	if u.Button("log#open", Rect{sin.X, sin.Bottom() - u.Px(40), u.Px(180), u.Px(38)}, icFolder, "Open log folder", BtnSecondary) {
		os.MkdirAll(a.dataDir(), 0o755)
		platformReveal(a.dataDir())
	}
}

func unitOf(a *App, chID int) string {
	if ch := a.chanByID(chID); ch != nil {
		return ch.UnitLabel()
	}
	return ""
}

// ---------------------------------------------------------------------------
// Alarms page

func (a *App) pageAlarms(r Rect) {
	u := a.ui
	gap := u.Px(16)
	left, right := r.CutLeft(r.W*62/100 - gap/2)
	right.X += gap
	right.W -= gap
	in := u.Card(left, "Limits")
	names, ids := a.channelNames()
	y := in.Y
	if len(a.alarms) == 0 {
		y += u.Wrap("Add an alarm to get a loud PC alert and a flashing readout when a reading goes above or below your limits — handy for battery cut-offs, overheating or a supply drifting.", in.X, y, in.W, u.F(13), th.Sub) + u.Px(10)
	}
	for i := 0; i < len(a.alarms); i++ {
		al := a.alarms[i]
		id := "al#" + itoa(i)
		rowH := u.Px(98)
		row := Rect{in.X, y, in.W, rowH}
		bg := th.Card2
		if al.active {
			pulse := 0.5 + 0.5*math.Sin(float64(time.Now().UnixMilli())/180)
			bg = lerpColor(th.Card2, hexColor(0x4A1C22), 0.4+0.4*pulse)
			u.Ambient = true
		}
		u.C.FillRoundRect(row, u.Pf(10), bg)
		ix := row.X + u.Px(12)
		ci := idxOf(ids, al.Chan)
		if u.Dropdown(id+"ch", Rect{ix, row.Y + u.Px(10), u.Px(220), u.Px(34)}, names, &ci) {
			al.Chan = ids[ci]
		}
		unit := unitOf(a, al.Chan)
		u.Checkbox(id+"hi", Rect{ix + u.Px(232), row.Y + u.Px(14), u.Px(70), u.Px(26)}, &al.UseHigh, "High")
		u.NumberField(id+"hv", Rect{ix + u.Px(306), row.Y + u.Px(10), u.Px(130), u.Px(34)}, &al.High, unit)
		u.Checkbox(id+"lo", Rect{ix + u.Px(232), row.Y + u.Px(56), u.Px(70), u.Px(26)}, &al.UseLow, "Low")
		u.NumberField(id+"lv", Rect{ix + u.Px(306), row.Y + u.Px(52), u.Px(130), u.Px(34)}, &al.Low, unit)
		u.Toggle(id+"en", Rect{ix, row.Y + u.Px(56), u.Px(100), u.Px(30)}, &al.Enabled, "On")
		u.Toggle(id+"snd", Rect{ix + u.Px(104), row.Y + u.Px(56), u.Px(110), u.Px(30)}, &al.Sound, "Sound")
		st := "OK"
		sc := th.Good
		if !al.Enabled {
			st, sc = "Off", th.Faint
		} else if al.active {
			st, sc = "ALARM", th.Bad
		}
		sx := row.Right() - u.Px(150)
		u.Text(st, Rect{sx, row.Y + u.Px(10), u.Px(100), u.Px(30)}, u.FH(15), sc, AlignLeft)
		if al.active && !al.acked {
			if u.Button(id+"ack", Rect{sx, row.Y + u.Px(52), u.Px(100), u.Px(34)}, icMute, "Silence", BtnDanger) {
				al.acked = true
			}
		}
		if u.IconButton(id+"rm", Rect{row.Right() - u.Px(42), row.Y + u.Px(10), u.Px(30), u.Px(30)}, icDelete, "Remove alarm", false) {
			a.alarms = append(a.alarms[:i], a.alarms[i+1:]...)
			i--
		}
		y += rowH + u.Px(10)
	}
	if u.ButtonEx("al#add", Rect{in.X, y + u.Px(4), u.Px(180), u.Px(40)}, icAdd, "Add alarm", BtnPrimary, len(a.channels) > 0) {
		ch := a.mainChannel()
		al := &Alarm{Chan: ch.ID, Enabled: true, Sound: true, UseHigh: true}
		if ch.HasLast && !ch.Last.IsOL() {
			al.High = NearestE(math.Abs(ch.Last.Value)*1.2+1e-12, "E6")
			al.Low = NearestE(math.Abs(ch.Last.Value)*0.8+1e-12, "E6")
		}
		a.alarms = append(a.alarms, al)
	}
	sin := u.Card(right, "Sound & continuity")
	sy := sin.Y
	if u.Toggle("snd#master", Rect{sin.X, sy, sin.W, u.Px(30)}, &a.settings.SoundOn, "PC sounds enabled") {
		a.saveSettings()
	}
	sy += u.Px(40)
	if u.Toggle("snd#cont", Rect{sin.X, sy, sin.W, u.Px(30)}, &a.settings.ContBeep, "Continuity beep on the PC") {
		a.saveSettings()
	}
	sy += u.Px(40)
	sy += u.Wrap("When a meter is in continuity mode and reads below the threshold, the PC beeps too — so you can keep your eyes on the probes.", sin.X, sy, sin.W, u.F(12.5), th.Sub) + u.Px(8)
	u.T.Draw("Beep below", sin.X, sy+u.Px(9), u.F(13), th.Text)
	if u.NumberField("snd#thr", Rect{sin.X + u.Px(100), sy, u.Px(140), u.Px(36)}, &a.settings.ContThreshold, "Ω") {
		a.saveSettings()
	}
	sy += u.Px(56)
	if u.Button("snd#t1", Rect{sin.X, sy, u.Px(130), u.Px(36)}, icVolume, "Test beep", BtnSecondary) {
		platformPlaySound(beepWAV(), false)
		a.sound.playing = -1
	}
	if u.Button("snd#t2", Rect{sin.X + u.Px(140), sy, u.Px(130), u.Px(36)}, icWarn, "Test alarm", BtnSecondary) {
		platformPlaySound(alarmWAV(), false)
		a.sound.playing = -1
	}
}

// ---------------------------------------------------------------------------
// Math page

type MathPage struct {
	preset int
	a, b   int
	expr   string
	unit   string
	name   string
	err    string
}

func (a *App) pageMath(r Rect) {
	u := a.ui
	mp := &a.math
	gap := u.Px(16)
	left, right := r.CutLeft(r.W*55/100 - gap/2)
	right.X += gap
	right.W -= gap
	in := u.Card(left, "New math channel")
	y := in.Y
	y += u.Wrap("Combine meters on the shared timeline. Connect a voltmeter and an ammeter and pick Power to see watts live — or integrate to get Wh and mAh for battery and charger tests.", in.X, y, in.W, u.F(13), th.Sub) + u.Px(12)
	u.T.Draw("Type", in.X, y, u.FB(12), th.Sub)
	y += u.Px(20)
	u.Dropdown("math#preset", Rect{in.X, y, in.W, u.Px(36)}, mathPresetNames, &mp.preset)
	y += u.Px(50)
	names, ids := a.channelNames()
	if MathPreset(mp.preset) != MathCustom {
		u.T.Draw("A", in.X, y+u.Px(9), u.FH(14), th.Accent)
		u.Dropdown("math#a", Rect{in.X + u.Px(26), y, in.W/2 - u.Px(34), u.Px(36)}, names, &mp.a)
		if MathPreset(mp.preset) != MathCharge {
			u.T.Draw("B", in.X+in.W/2+u.Px(6), y+u.Px(9), u.FH(14), th.Accent)
			u.Dropdown("math#b", Rect{in.X + in.W/2 + u.Px(30), y, in.W/2 - u.Px(30), u.Px(36)}, names, &mp.b)
		}
		y += u.Px(50)
	} else {
		u.T.Draw("Formula (use ch1, ch2 … for channels)", in.X, y, u.FB(12), th.Sub)
		y += u.Px(20)
		u.TextField("math#expr", Rect{in.X, y, in.W, u.Px(36)}, &mp.expr, "e.g. (ch1 - ch2) / 0.1")
		y += u.Px(46)
		u.T.Draw("Unit", in.X, y+u.Px(9), u.FB(12), th.Sub)
		u.TextField("math#unit", Rect{in.X + u.Px(60), y, u.Px(120), u.Px(36)}, &mp.unit, "A")
		y += u.Px(50)
		y += u.Wrap("Functions: abs sqrt min max pow log10 ln exp sin cos · constants pi e · operators + − × ÷ ^", in.X, y, in.W, u.F(12), th.Faint) + u.Px(8)
	}
	u.T.Draw("Name", in.X, y+u.Px(9), u.FB(12), th.Sub)
	u.TextField("math#name", Rect{in.X + u.Px(60), y, in.W - u.Px(60), u.Px(36)}, &mp.name, mathPresetNames[mp.preset])
	y += u.Px(54)
	if mp.err != "" {
		y += u.Wrap(mp.err, in.X, y, in.W, u.F(12.5), th.Bad) + u.Px(6)
	}
	if u.ButtonEx("math#add", Rect{in.X, y, u.Px(200), u.Px(42)}, icAdd, "Add math channel", BtnPrimary, len(a.channels) > 0) {
		def := &MathDef{Preset: MathPreset(mp.preset), A: ids[clampi(mp.a, 0, len(ids)-1)], B: ids[clampi(mp.b, 0, len(ids)-1)], Expr: mp.expr, Unit: mp.unit}
		if err := def.Prepare(); err != nil {
			mp.err = err.Error()
		} else {
			mp.err = ""
			name := mp.name
			if name == "" {
				name = strings.Fields(mathPresetNames[mp.preset])[0]
			}
			ch := a.addChannel(name, KindMath, mathPresetNames[mp.preset], 0)
			ch.Math = def
			u.Toast("Math channel "+name+" added", 1)
			mp.name = ""
		}
	}
	// existing math channels
	lin := u.Card(right, "Math channels")
	ly := lin.Y
	n := 0
	for _, ch := range a.channels {
		if ch.Kind != KindMath {
			continue
		}
		n++
		row := Rect{lin.X, ly, lin.W, u.Px(74)}
		u.C.FillRoundRect(row, u.Pf(10), th.Card2)
		u.C.FillRoundRect(Rect{row.X, row.Y, u.Px(4), row.H}, u.Pf(2), chanColor(ch.Color))
		u.Text(ch.Name, Rect{row.X + u.Px(16), row.Y + u.Px(10), row.W - u.Px(120), u.Px(20)}, u.FB(13.5), th.Text, AlignLeft)
		desc := mathPresetNames[ch.Math.Preset]
		if ch.Math.Preset == MathCustom {
			desc = ch.Math.Expr
		} else {
			desc += "   (A = " + chName(a, ch.Math.A)
			if ch.Math.Preset != MathCharge {
				desc += ", B = " + chName(a, ch.Math.B)
			}
			desc += ")"
		}
		u.Text(desc, Rect{row.X + u.Px(16), row.Y + u.Px(32), row.W - u.Px(120), u.Px(18)}, u.F(12), th.Sub, AlignLeft)
		val := "—"
		if ch.HasLast {
			val = FormatSI(ch.Last.Value, ch.Last.Unit, 5)
			if ch.Last.Unit == "Ah" {
				val = FormatSI(ch.Last.Value*1000, "mAh", 5)
			}
		}
		u.Text(val, Rect{row.X + u.Px(16), row.Y + u.Px(50), row.W - u.Px(120), u.Px(18)}, u.FB(13), chanColor(ch.Color), AlignLeft)
		if ch.Math.Preset == MathEnergy || ch.Math.Preset == MathCharge {
			if u.IconButton("mc#rs"+itoa(ch.ID), Rect{row.Right() - u.Px(76), row.Y + u.Px(20), u.Px(30), u.Px(30)}, icRefresh, "Reset integrator to zero", false) {
				ch.Math.Reset()
			}
		}
		if u.IconButton("mc#rm"+itoa(ch.ID), Rect{row.Right() - u.Px(40), row.Y + u.Px(20), u.Px(30), u.Px(30)}, icDelete, "Remove", false) {
			a.removeChannel(ch.ID)
			break
		}
		ly += u.Px(82)
	}
	if n == 0 {
		u.Wrap("No math channels yet.", lin.X, ly, lin.W, u.F(13), th.Faint)
	}
}

func chName(a *App, id int) string {
	if c := a.chanByID(id); c != nil {
		return c.Name
	}
	return "?"
}

// ---------------------------------------------------------------------------
// Settings page

func (a *App) pageSettings(r Rect) {
	u := a.ui
	gap := u.Px(16)
	left, right := r.CutLeft(r.W/2 - gap/2)
	right.X += gap
	right.W -= gap
	in := u.Card(left, "Appearance & data")
	y := in.Y
	u.T.Draw("Accent colour", in.X, y, u.FB(12), th.Sub)
	y += u.Px(24)
	for i, ac := range accentChoices {
		cr := Rect{in.X + i*u.Px(52), y, u.Px(40), u.Px(40)}
		id := "acc#" + itoa(i)
		h, cl := u.interact(id, cr)
		hv := u.Anim(id+"#h", b2f(h), 14)
		u.C.FillCircle(float64(cr.CX()), float64(cr.CY()), u.Pf(14+2*hv), ac.acc)
		if a.settings.Accent == i {
			u.C.StrokeCircle(float64(cr.CX()), float64(cr.CY()), u.Pf(19), u.Pf(2), th.Text)
		}
		u.Tip(id, cr, ac.name)
		if cl {
			a.settings.Accent = i
			a.applyAccent()
			a.saveSettings()
		}
	}
	y += u.Px(58)
	u.T.Draw("Interface size", in.X, y, u.FB(12), th.Sub)
	y += u.Px(22)
	scales := []string{"90%", "100%", "110%", "125%", "150%"}
	vals := []float64{0.9, 1, 1.1, 1.25, 1.5}
	si := 1
	for i, v := range vals {
		if math.Abs(v-a.settings.UIScale) < 0.01 {
			si = i
		}
	}
	if u.Segmented("set#scale", Rect{in.X, y, min(in.W, u.Px(400)), u.Px(32)}, scales, &si) {
		a.settings.UIScale = vals[si]
		a.saveSettings()
	}
	y += u.Px(50)
	u.T.Draw("Data folder (logs, exports)", in.X, y, u.FB(12), th.Sub)
	y += u.Px(22)
	dr := Rect{in.X, y, in.W - u.Px(120), u.Px(36)}
	u.C.FillRoundRect(dr, u.Pf(9), th.Bg2)
	u.Text(a.dataDir(), dr.InsetXY(u.Px(11), 0), u.F(12.5), th.Text, AlignLeft)
	if u.Button("set#open", Rect{dr.Right() + u.Px(8), y, u.Px(112), u.Px(36)}, icFolder, "Open", BtnSecondary) {
		os.MkdirAll(a.dataDir(), 0o755)
		platformReveal(a.dataDir())
	}
	y += u.Px(52)
	u.T.Draw("Samples kept per channel", in.X, y+u.Px(9), u.F(13), th.Text)
	ms := float64(a.settings.MaxSamples)
	if u.NumberField("set#max", Rect{in.X + u.Px(220), y, u.Px(160), u.Px(36)}, &ms, "") {
		a.settings.MaxSamples = max(10000, int(ms))
		a.saveSettings()
	}
	y += u.Px(54)
	if u.Toggle("set#welcome", Rect{in.X, y, in.W, u.Px(30)}, &a.settings.ShowWelcome, "Show the welcome tips on start") {
		a.saveSettings()
	}
	y += u.Px(38)
	teach := u.Teach
	if u.Toggle("set#teach", Rect{in.X, y, in.W, u.Px(30)}, &teach, "Teaching mode: explain whatever I point at (F2)") {
		u.ToggleTeach()
		a.settings.TeachOff = !u.Teach
		a.saveSettings()
	}

	sin := u.Card(Rect{right.X, right.Y, right.W, right.H*52/100 - gap/2}, "Suite integration")
	sy := sin.Y
	if u.Toggle("set#api", Rect{sin.X, sy, sin.W, u.Px(30)}, &a.settings.APIEnabled, "Share live readings with other suite apps") {
		a.saveSettings()
		if a.settings.APIEnabled {
			a.startAPI()
		}
	}
	sy += u.Px(40)
	url := fmt.Sprintf("http://127.0.0.1:%d/api/v1/readings", a.settings.APIPort)
	sy += u.Wrap("Wirie's Calcie calculators (and any suite app) can pull the current value from:", sin.X, sy, sin.W, u.F(12.5), th.Sub) + u.Px(6)
	ur := Rect{sin.X, sy, sin.W - u.Px(46), u.Px(34)}
	u.C.FillRoundRect(ur, u.Pf(8), th.Bg2)
	u.Text(url, ur.InsetXY(u.Px(10), 0), u.FM(12), th.AccentHi, AlignLeft)
	if u.IconButton("set#copyurl", Rect{ur.Right() + u.Px(8), sy, u.Px(34), u.Px(34)}, icCopy, "Copy URL", false) {
		platformClipboardSet(url)
		u.Toast("Copied", 1)
	}
	sy += u.Px(44)
	sy += u.Wrap("The latest readings are also written to "+filepath.Join(suiteDir(), "meterie-live.json")+" so apps can read them without networking. Timeline exports use the shared nrmluv-timeline format so captures from Scopie, Serialie and Logicie line up.", sin.X, sy, sin.W, u.F(12), th.Faint)

	ab := u.Card(Rect{right.X, right.Y + right.H*52/100 + gap/2, right.W, right.H*48/100 - gap/2}, "About")
	ay := ab.Y
	drawLogo(u.C, float64(ab.X), float64(ay), u.Pf(44))
	u.T.Draw("Meterie "+AppVersion, ab.X+u.Px(58), ay, u.FH(18), th.Text)
	u.T.Draw("Made by "+AppAuthor+" · part of the free hobbyist tool suite", ab.X+u.Px(58), ay+u.Px(26), u.F(12.5), th.Sub)
	ay += u.Px(60)
	ay += u.Wrap("Free for hobbyists. Protocol knowledge thanks to the sigrok project, the OWON/UNI-T/Aneng reverse-engineering community (DeanCording, ljakob, mwuertinger, ludwich66 and others) and the Fluke remote interface spec.", ab.X, ay, ab.W, u.F(12), th.Faint)
}
