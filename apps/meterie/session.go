package meterie

// session.go: .meterie sessions, CSV / suite-timeline export & import, PNG export.

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type sessChannel struct {
	ID       int       `json:"id"`
	Name     string    `json:"name"`
	Color    int       `json:"color"`
	Kind     int       `json:"kind"`
	Source   string    `json:"source"`
	Visible  bool      `json:"visible"`
	RelOn    bool      `json:"relOn"`
	RelValue float64   `json:"relValue"`
	Units    []unitKey `json:"units"`
	T        []int64   `json:"t"`
	V        []float64 `json:"v"` // NaN encoded as null via pointer-free trick (see below)
	OL       []int     `json:"ol,omitempty"`
	U        []uint8   `json:"u"`
	F        []uint32  `json:"f"`
	Math     *MathDef  `json:"math,omitempty"`
	MathExpr string    `json:"mathExpr,omitempty"`
}

type sessFile struct {
	App      string        `json:"app"`
	Version  string        `json:"version"`
	Author   string        `json:"author"`
	Saved    time.Time     `json:"saved"`
	Channels []sessChannel `json:"channels"`
	Markers  []Marker      `json:"markers"`
	Alarms   []*Alarm      `json:"alarms"`
	MainChan int           `json:"main"`
}

// MathDef JSON (exported fields only are encoded; ast is rebuilt).
func (m *MathDef) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{"preset": m.Preset, "a": m.A, "b": m.B, "expr": m.Expr, "unit": m.Unit, "acc": m.acc})
}
func (m *MathDef) UnmarshalJSON(b []byte) error {
	var v struct {
		Preset MathPreset `json:"preset"`
		A, B   int
		Expr   string  `json:"expr"`
		Unit   string  `json:"unit"`
		Acc    float64 `json:"acc"`
	}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	m.Preset, m.A, m.B, m.Expr, m.Unit, m.acc = v.Preset, v.A, v.B, v.Expr, v.Unit, v.Acc
	return m.Prepare()
}

func (a *App) buildSession() sessFile {
	sf := sessFile{App: AppName, Version: AppVersion, Author: AppAuthor, Saved: time.Now(), Markers: a.markers, Alarms: a.alarms, MainChan: a.mainChan}
	for _, ch := range a.channels {
		sc := sessChannel{ID: ch.ID, Name: ch.Name, Color: ch.Color, Kind: int(ch.Kind), Source: ch.Source, Visible: ch.Visible,
			RelOn: ch.RelOn, RelValue: ch.RelValue, Units: ch.Units, Math: ch.Math}
		for i, s := range ch.Samples {
			sc.T = append(sc.T, s.T)
			if math.IsNaN(s.V) {
				sc.V = append(sc.V, 0)
				sc.OL = append(sc.OL, i)
			} else {
				sc.V = append(sc.V, s.V)
			}
			sc.U = append(sc.U, s.U)
			sc.F = append(sc.F, uint32(s.F))
		}
		sf.Channels = append(sf.Channels, sc)
	}
	return sf
}

func (a *App) saveSession(path string) error {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	enc := json.NewEncoder(zw)
	if err := enc.Encode(a.buildSession()); err != nil {
		return err
	}
	zw.Close()
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

func (a *App) saveSessionDialog(forceNew bool) {
	p := a.sessionPath
	if p == "" || forceNew {
		def := filepath.Join(a.dataDir(), "session-"+time.Now().Format("2006-01-02_15-04")+".meterie")
		var ok bool
		p, ok = platformSaveDialog("Save Meterie session", def, "Meterie session (*.meterie)", "meterie")
		if !ok {
			return
		}
	}
	os.MkdirAll(filepath.Dir(p), 0o755)
	if err := a.saveSession(p); err != nil {
		a.ui.Toast("Save failed: "+err.Error(), 3)
		return
	}
	a.sessionPath = p
	a.ui.Toast("Session saved — "+filepath.Base(p), 1)
}

func (a *App) openSessionDialog() {
	p, ok := platformOpenDialog("Open Meterie session", "Meterie sessions & timelines (*.meterie;*.csv)", "*.meterie;*.csv;*.nst.csv")
	if !ok {
		return
	}
	if strings.HasSuffix(strings.ToLower(p), ".csv") {
		n, err := a.importTimeline(p)
		if err != nil {
			a.ui.Toast("Import failed: "+err.Error(), 3)
		} else {
			a.ui.Toast(fmt.Sprintf("Imported %d channel(s) from %s", n, filepath.Base(p)), 1)
			a.page = PageLive
		}
		return
	}
	if err := a.loadSession(p); err != nil {
		a.ui.Toast("Open failed: "+err.Error(), 3)
		return
	}
	a.sessionPath = p
	a.page = PageLive
	a.ui.Toast("Opened "+filepath.Base(p), 1)
}

func (a *App) loadSession(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	var r io.Reader = bufio.NewReader(f)
	br := r.(*bufio.Reader)
	if hdr, _ := br.Peek(2); len(hdr) == 2 && hdr[0] == 0x1f && hdr[1] == 0x8b {
		zr, err := gzip.NewReader(br)
		if err != nil {
			return err
		}
		r = zr
	}
	var sf sessFile
	if err := json.NewDecoder(r).Decode(&sf); err != nil {
		return err
	}
	if sf.App != AppName {
		return fmt.Errorf("not a Meterie session")
	}
	a.newSession()
	maxID := 0
	for _, sc := range sf.Channels {
		ch := &Channel{ID: sc.ID, Name: sc.Name, Color: sc.Color, Kind: ChannelKind(sc.Kind), Source: sc.Source + " (saved)",
			Visible: sc.Visible, RelOn: sc.RelOn, RelValue: sc.RelValue, Units: sc.Units, Math: sc.Math}
		if ch.Kind == KindMeter || ch.Kind == KindCamera || ch.Kind == KindManual {
			ch.Kind = KindImport
		}
		ol := map[int]bool{}
		for _, i := range sc.OL {
			ol[i] = true
		}
		for i := range sc.T {
			v := sc.V[i]
			if ol[i] {
				v = math.NaN()
			}
			s := Sample{T: sc.T[i], V: v}
			if i < len(sc.U) {
				s.U = sc.U[i]
			}
			if i < len(sc.F) {
				s.F = Flags(sc.F[i])
			}
			ch.Samples = append(ch.Samples, s)
		}
		if n := len(ch.Samples); n > 0 && len(ch.Units) > 0 {
			s := ch.Samples[n-1]
			uk := ch.Units[min(int(s.U), len(ch.Units)-1)]
			ch.Last = Reading{T: time.Unix(0, s.T), Value: s.V, Q: uk.Q, Unit: uk.Unit, Flags: s.F}
			num, pu := SplitSI(s.V, uk.Unit, 5)
			ch.Last.Display, ch.Last.Prefix = num, strings.TrimSuffix(pu, uk.Unit)
			if math.IsNaN(s.V) {
				ch.Last.Display = "OL"
			}
			ch.HasLast = true
		}
		a.channels = append(a.channels, ch)
		maxID = max(maxID, ch.ID)
	}
	a.nextChanID = maxID + 1
	a.markers = sf.Markers
	a.alarms = sf.Alarms
	a.mainChan = sf.MainChan
	// show the whole recording
	a.fitAll()
	return nil
}

func (a *App) fitAll() {
	var t0, t1 int64 = math.MaxInt64, math.MinInt64
	for _, ch := range a.channels {
		if len(ch.Samples) > 0 {
			t0 = min(t0, ch.Samples[0].T)
			t1 = max(t1, ch.Samples[len(ch.Samples)-1].T)
		}
	}
	if t0 < t1 {
		pad := (t1 - t0) / 30
		a.graph.Follow = false
		a.graph.T0, a.graph.T1 = t0-pad, t1+pad
		a.graph.AutoY = true
	}
}

func (a *App) newSession() {
	for _, d := range a.devices {
		d.Stop()
	}
	a.log.Stop(a, "new session")
	a.devices = nil
	a.channels = nil
	a.markers = nil
	a.alarms = nil
	a.mainChan = 0
	a.nextChanID = 1
	a.sessionPath = ""
	a.graph = NewGraphView()
	a.demoSecond = map[int]int{}
	a.cam.Stop()
	a.cam.feed = false
	a.cam.chanID = 0
}

// ---------------------------------------------------------------------------
// Suite timeline format (shared with Scopie, Serialie / Logicie …)

const timelineMagic = "# nrmluv-timeline v1"

func flagNames(f Flags) string {
	var p []string
	for _, x := range []struct {
		f Flags
		n string
	}{{FAC, "AC"}, {FDC, "DC"}, {FAuto, "AUTO"}, {FManual, "MANUAL"}, {FHold, "HOLD"}, {FRel, "REL"}, {FMin, "MIN"}, {FMax, "MAX"},
		{FLowBat, "LOWBAT"}, {FDiode, "DIODE"}, {FBeep, "BEEP"}, {FOL, "OL"}, {FHV, "HV"}} {
		if f&x.f != 0 {
			p = append(p, x.n)
		}
	}
	return strings.Join(p, "|")
}

func (a *App) writeTimeline(path string, t0, t1 int64) error {
	var b strings.Builder
	start := int64(math.MaxInt64)
	for _, ch := range a.channels {
		for _, s := range ch.Samples {
			if s.T >= t0 && s.T <= t1 {
				start = min(start, s.T)
				break
			}
		}
	}
	if start == math.MaxInt64 {
		start = time.Now().UnixNano()
	}
	b.WriteString(timelineMagic + "\n")
	fmt.Fprintf(&b, "# app=%s version=%s author=%s exported=%s\n", AppName, AppVersion, AppAuthor, time.Now().UTC().Format(time.RFC3339))
	fmt.Fprintf(&b, "# t0_unix_ns=%d\n", start)
	for _, ch := range a.channels {
		fmt.Fprintf(&b, "# channel id=%d name=%s unit=%s kind=%s source=%s\n", ch.ID, tlEsc(ch.Name), tlEsc(ch.UnitLabel()), []string{"meter", "math", "import", "manual", "camera"}[ch.Kind], tlEsc(ch.Source))
	}
	for _, m := range a.markers {
		if m.T >= t0 && m.T <= t1 {
			fmt.Fprintf(&b, "# marker t_unix_ns=%d text=%s\n", m.T, tlEsc(m.Text))
		}
	}
	b.WriteString("t_unix_ns,t_rel_s,channel_id,channel,value,unit,flags\n")
	type row struct {
		t   int64
		txt string
	}
	var rows []row
	for _, ch := range a.channels {
		for _, s := range ch.Samples {
			if s.T < t0 || s.T > t1 {
				continue
			}
			unit := ""
			if int(s.U) < len(ch.Units) {
				unit = ch.Units[s.U].Unit
			}
			v := FormatValue(s.V)
			rows = append(rows, row{s.T, fmt.Sprintf("%d,%.6f,%d,%s,%s,%s,%s\n", s.T, float64(s.T-start)/1e9, ch.ID, csvEsc(ch.Name), v, unit, flagNames(s.F))})
		}
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].t < rows[j].t })
	for _, r := range rows {
		b.WriteString(r.txt)
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func tlEsc(s string) string { return strings.ReplaceAll(strings.ReplaceAll(s, " ", "_"), "=", "-") }

// writeCSV exports every reading in a spreadsheet-friendly long format.
func (a *App) writeCSV(path string, t0, t1 int64) error {
	var b strings.Builder
	b.WriteString("timestamp,unix_ms,channel,source,value,unit,mode,flags\n")
	type row struct {
		t   int64
		txt string
	}
	var rows []row
	for _, ch := range a.channels {
		for _, s := range ch.Samples {
			if s.T < t0 || s.T > t1 {
				continue
			}
			unit, mode := "", ""
			if int(s.U) < len(ch.Units) {
				uk := ch.Units[s.U]
				unit = uk.Unit
				mode = Reading{Q: uk.Q, Flags: s.F}.ModeLabel()
			}
			tm := time.Unix(0, s.T)
			v := FormatValue(ch.DisplayValue(s.V))
			rows = append(rows, row{s.T, fmt.Sprintf("%s,%d,%s,%s,%s,%s,%s,%s\n", tm.Format("2006-01-02 15:04:05.000"), s.T/1e6,
				csvEsc(ch.Name), csvEsc(ch.Source), v, unit, csvEsc(mode), flagNames(s.F))})
		}
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].t < rows[j].t })
	for _, r := range rows {
		b.WriteString(r.txt)
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

// importTimeline reads a nrmluv-timeline CSV (from any suite app) as new channels.
func (a *App) importTimeline(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	first := true
	idMap := map[string]*Channel{}
	app := "file"
	n := 0
	header := false
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if first {
			first = false
			if !strings.HasPrefix(line, timelineMagic) {
				return 0, fmt.Errorf("not a nrmluv-timeline file (first line must be %q)", timelineMagic)
			}
			continue
		}
		if strings.HasPrefix(line, "# app=") {
			app = strings.Fields(strings.TrimPrefix(line, "# app="))[0]
			continue
		}
		if strings.HasPrefix(line, "# marker ") {
			kv := parseKV(strings.TrimPrefix(line, "# marker "))
			t, _ := strconv.ParseInt(kv["t_unix_ns"], 10, 64)
			a.markers = append(a.markers, Marker{T: t, Text: strings.ReplaceAll(kv["text"], "_", " "), Color: 4})
			continue
		}
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		if !header {
			header = true
			continue
		}
		cols := strings.Split(line, ",")
		if len(cols) < 6 {
			continue
		}
		t, err := strconv.ParseInt(cols[0], 10, 64)
		if err != nil {
			continue
		}
		key := cols[2] + "/" + cols[3]
		ch := idMap[key]
		if ch == nil {
			ch = a.addChannel(strings.Trim(cols[3], "\"")+" ("+app+")", KindImport, "Imported from "+filepath.Base(path), 0)
			idMap[key] = ch
			n++
		}
		v, err := strconv.ParseFloat(cols[4], 64)
		var fl Flags
		if err != nil {
			v = math.NaN()
			fl |= FOL
		}
		if len(cols) > 6 {
			for _, p := range strings.Split(cols[6], "|") {
				switch p {
				case "AC":
					fl |= FAC
				case "DC":
					fl |= FDC
				}
			}
		}
		q := quantityForUnit(cols[5])
		r := Reading{T: time.Unix(0, t), Value: v, Q: q, Unit: cols[5], Flags: fl}
		num, pu := SplitSI(v, cols[5], 5)
		r.Display, r.Prefix = num, strings.TrimSuffix(pu, cols[5])
		ch.Add(r)
	}
	if n == 0 {
		return 0, fmt.Errorf("no samples found")
	}
	a.fitAll()
	return n, sc.Err()
}

func parseKV(s string) map[string]string {
	out := map[string]string{}
	for _, f := range strings.Fields(s) {
		if i := strings.IndexByte(f, '='); i > 0 {
			out[f[:i]] = f[i+1:]
		}
	}
	return out
}

func quantityForUnit(u string) Quantity {
	for q := QNone; q < qCount; q++ {
		if q.BaseUnit() == u && u != "" {
			return q
		}
	}
	return QUnitless
}

// ---------------------------------------------------------------------------
// PNG export of the graph (rendered off-screen at high resolution)

func (a *App) exportGraphPNG() {
	def := filepath.Join(a.dataDir(), "meterie-graph-"+time.Now().Format("2006-01-02_15-04-05")+".png")
	p, ok := platformSaveDialog("Export graph as PNG", def, "PNG image (*.png)", "png")
	if !ok {
		return
	}
	if err := a.renderGraphPNG(p, 1920, 1080); err != nil {
		a.ui.Toast("Export failed: "+err.Error(), 3)
		return
	}
	a.ui.Toast("Graph saved — "+filepath.Base(p), 1)
}

func (a *App) renderGraphPNG(path string, w, h int) error {
	c, t, done := platformOffscreen(w, h)
	if c == nil {
		return fmt.Errorf("off-screen rendering unavailable")
	}
	defer done()
	u := a.ui
	oc, ot, os_, oin := u.C, u.T, u.S, u.In
	u.C, u.T, u.S = c, t, 1.5
	u.In = Input{}
	c.Clear(th.Bg)
	t.SetClip(Rect{0, 0, w, h})
	title := "Meterie — " + time.Now().Format("2 Jan 2006 15:04")
	u.T.Draw(title, u.Px(28), u.Px(18), u.FH(20), th.Text)
	u.T.Draw("by "+AppAuthor, u.Px(28), u.Px(50), u.F(12), th.Sub)
	g := *a.graph
	a.DrawGraph(Rect{u.Px(10), u.Px(80), w - u.Px(20), h - u.Px(100)}, &g, a.channels, false)
	u.C, u.T, u.S, u.In = oc, ot, os_, oin
	img := c.ToImage(Rect{0, 0, w, h})
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// ---------------------------------------------------------------------------
// Export modal

var exportRanges = []string{"Everything", "Visible on graph", "Selection"}

func (a *App) drawExportModal() {
	u := a.ui
	cr, closed := u.Modal("m#export", u.Px(620), u.Px(470), "Export & import")
	if closed {
		a.modal = ""
	}
	y := cr.Y
	u.T.Draw("Range", cr.X, y+u.Px(9), u.FB(12), th.Sub)
	u.Segmented("exp#range", Rect{cr.X + u.Px(70), y, cr.W - u.Px(70), u.Px(34)}, exportRanges, &a.expRange)
	y += u.Px(50)
	t0, t1 := int64(math.MinInt64), int64(math.MaxInt64)
	switch a.expRange {
	case 1:
		t0, t1 = a.graph.Range(time.Now().UnixNano())
	case 2:
		if a.graph.SelOn {
			t0, t1 = min(a.graph.Sel0, a.graph.Sel1), max(a.graph.Sel0, a.graph.Sel1)
		}
	}
	item := func(id, icon, title, desc string) bool {
		r := Rect{cr.X, y, cr.W, u.Px(62)}
		cl := u.ListItem(id, r, false)
		u.C.FillRoundRect(Rect{r.X + u.Px(10), r.Y + u.Px(11), u.Px(40), u.Px(40)}, u.Pf(10), th.Accent.WithA(0.14))
		u.Text(icon, Rect{r.X + u.Px(10), r.Y + u.Px(11), u.Px(40), u.Px(40)}, u.FI(16), th.AccentHi, AlignCenter)
		u.T.Draw(title, r.X+u.Px(62), r.Y+u.Px(11), u.FB(14), th.Text)
		u.T.Draw(desc, r.X+u.Px(62), r.Y+u.Px(33), u.F(12), th.Sub)
		y += u.Px(66)
		return cl
	}
	stamp := time.Now().Format("2006-01-02_15-04")
	if item("exp#csv", icList, "CSV spreadsheet", "Every reading: timestamp, channel, value, unit, mode (Excel / LibreOffice)") {
		if p, ok := platformSaveDialog("Export CSV", filepath.Join(a.dataDir(), "meterie-"+stamp+".csv"), "CSV files (*.csv)", "csv"); ok {
			a.reportExport(a.writeCSV(p, t0, t1), p)
		}
	}
	if item("exp#tl", icSync, "Suite timeline (.nst.csv)", "Shared nrmluv-timeline format — lines up with Scopie, Serialie and Logicie captures") {
		if p, ok := platformSaveDialog("Export suite timeline", filepath.Join(a.dataDir(), "meterie-"+stamp+".nst.csv"), "Suite timeline (*.nst.csv)", "nst.csv"); ok {
			a.reportExport(a.writeTimeline(p, t0, t1), p)
		}
	}
	if item("exp#png", icPhoto, "Graph image (PNG)", "1920×1080 picture of the current graph view, with legend and markers") {
		a.modal = ""
		a.exportGraphPNG()
	}
	if item("exp#sess", icSave, "Meterie session (.meterie)", "Everything — channels, data, markers, math and alarms — to reopen later") {
		a.modal = ""
		a.saveSessionDialog(true)
	}
	if item("exp#imp", icOpen, "Import a suite timeline / open a session", "Overlay a Scopie / Serialie / Logicie capture on Meterie's timeline") {
		a.modal = ""
		a.openSessionDialog()
	}
	u.EndModal()
}

func (a *App) reportExport(err error, p string) {
	if err != nil {
		a.ui.Toast("Export failed: "+err.Error(), 3)
		return
	}
	a.ui.Toast("Saved "+filepath.Base(p), 1)
	a.modal = ""
}
