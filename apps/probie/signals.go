// Probie — Signals: reading captures from the rest of the Tinkerie suite.
// Scopie waveforms (CSV), Probie logic captures (VCD) and serial logs (CSV),
// Meterie timelines (.nst.csv, the suite's shared format) and plain CSV.
// Part of the Tinkerie maker suite. Created by nrmluvdev.

package probie

import (
	"bufio"
	"bytes"
	"errors"
	"math"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// MaxSignal is the largest capture Probie reads.
const MaxSignal = 96 << 20

// Kinds of channel.
const (
	KindAnalog  = "analog"  // regularly sampled voltage (a scope)
	KindDigital = "digital" // edges: 0/1 (a logic analyser)
	KindMeter   = "meter"   // slow readings (a multimeter)
)

// SigChan is one channel of a capture. Times are in seconds from the start.
type SigChan struct {
	Name string
	Unit string
	Kind string
	T    []float64
	V    []float64
}

// SerialLine is one line of a serial log.
type SerialLine struct {
	T    float64
	Dir  string
	Text string
}

// SigCapture is a loaded file.
type SigCapture struct {
	File    string
	Source  string // "Scopie waveform", "Probie logic capture", …
	App     string // Scopie, Probie, Meterie or ""
	Chans   []*SigChan
	Serial  []SerialLine
	Markers []Marker
	Notes   []string
}

// Marker is an annotation from the file.
type Marker struct {
	T    float64
	Text string
}

// IsSignalFile guesses from the name and the first bytes whether a file is a
// capture rather than code.
func IsSignalFile(name string, head []byte) bool {
	low := strings.ToLower(name)
	if strings.HasSuffix(low, ".vcd") || strings.HasSuffix(low, ".nst.csv") {
		return true
	}
	h := string(head)
	if strings.HasPrefix(h, "# nrmluv-timeline") || strings.HasPrefix(strings.TrimSpace(h), "$date") || strings.HasPrefix(strings.TrimSpace(h), "$timescale") {
		return true
	}
	if strings.HasSuffix(low, ".csv") || strings.HasSuffix(low, ".tsv") || strings.HasSuffix(low, ".txt") {
		first := strings.ToLower(strings.SplitN(h, "\n", 2)[0])
		if strings.HasPrefix(first, "time") || strings.HasPrefix(first, "timestamp,elapsed_s,direction") || strings.HasPrefix(first, "t,") || strings.HasPrefix(first, "x,") {
			return true
		}
		// mostly numbers?
		lines := strings.Split(h, "\n")
		num := 0
		for i, ln := range lines {
			if i > 20 {
				break
			}
			if _, err := strconv.ParseFloat(strings.TrimSpace(strings.Split(strings.Split(ln, ",")[0], ";")[0]), 64); err == nil {
				num++
			}
		}
		return num >= 5
	}
	return false
}

// LoadCapture reads any supported capture.
func LoadCapture(name string, b []byte) (*SigCapture, error) {
	b = bytes.TrimPrefix(b, []byte("\xEF\xBB\xBF"))
	if len(b) == 0 {
		return nil, errors.New("the file is empty")
	}
	s := string(b)
	var c *SigCapture
	var err error
	switch {
	case strings.HasPrefix(s, "# nrmluv-timeline"):
		c, err = parseTimeline(s)
	case strings.Contains(s[:min(len(s), 4000)], "$enddefinitions") || strings.HasPrefix(strings.TrimSpace(s), "$"):
		c, err = parseVCD(s)
	case strings.HasPrefix(strings.ToLower(s), "timestamp,elapsed_s,direction,line"):
		c, err = parseSerialLog(s)
	default:
		c, err = parseCSV(s)
	}
	if err != nil {
		return nil, err
	}
	c.File = filepath.Base(name)
	if c.App == "" {
		low := strings.ToLower(c.File)
		switch {
		case strings.HasPrefix(low, "scope-"):
			c.App, c.Source = "Scopie", "Scopie waveform"
		case strings.HasPrefix(low, "probie"), strings.HasPrefix(low, "tracie"):
			c.App = "Probie"
		case strings.HasPrefix(low, "meterie"):
			c.App = "Meterie"
		}
	}
	for _, ch := range c.Chans {
		sortChan(ch)
	}
	if len(c.Chans) == 0 && len(c.Serial) == 0 {
		return nil, errors.New("Probie couldn't find any channels in this file")
	}
	return c, nil
}

func sortChan(ch *SigChan) {
	if sort.Float64sAreSorted(ch.T) {
		return
	}
	idx := make([]int, len(ch.T))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool { return ch.T[idx[a]] < ch.T[idx[b]] })
	t, v := make([]float64, len(idx)), make([]float64, len(idx))
	for i, k := range idx {
		t[i], v[i] = ch.T[k], ch.V[k]
	}
	ch.T, ch.V = t, v
}

// ---------------------------------------------------------------- nrmluv-timeline

func metaFields(line string) map[string]string {
	out := map[string]string{}
	for _, f := range strings.Fields(strings.TrimPrefix(line, "#")) {
		if k, v, ok := strings.Cut(f, "="); ok {
			out[k] = strings.ReplaceAll(v, "_", " ")
		}
	}
	return out
}

func parseTimeline(s string) (*SigCapture, error) {
	c := &SigCapture{Source: "Suite timeline"}
	byID := map[string]*SigChan{}
	var t0 int64
	var hdr []string
	sc := bufio.NewScanner(strings.NewReader(s))
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	var rawT []int64
	var rawCh []*SigChan
	var rawV []float64
	for sc.Scan() {
		ln := strings.TrimSpace(sc.Text())
		if ln == "" {
			continue
		}
		if strings.HasPrefix(ln, "#") {
			m := metaFields(ln)
			switch {
			case m["app"] != "" && strings.Contains(ln, "version="):
				c.App = m["app"]
			case strings.Contains(ln, "t0_unix_ns="):
				t0, _ = strconv.ParseInt(m["t0_unix_ns"], 10, 64)
			case strings.HasPrefix(ln, "# channel"):
				ch := &SigChan{Name: m["name"], Unit: m["unit"], Kind: KindMeter}
				switch strings.ToLower(m["kind"]) {
				case "logic", "digital":
					ch.Kind = KindDigital
				case "scope", "analog":
					ch.Kind = KindAnalog
				}
				byID[m["id"]] = ch
				c.Chans = append(c.Chans, ch)
			case strings.HasPrefix(ln, "# marker"):
				ts, _ := strconv.ParseInt(m["t_unix_ns"], 10, 64)
				c.Markers = append(c.Markers, Marker{T: float64(ts), Text: m["text"]})
			}
			continue
		}
		cells := strings.Split(ln, ",")
		if hdr == nil {
			hdr = cells
			continue
		}
		col := func(name string) string {
			for i, h := range hdr {
				if h == name && i < len(cells) {
					return cells[i]
				}
			}
			return ""
		}
		ts, err := strconv.ParseInt(col("t_unix_ns"), 10, 64)
		if err != nil {
			continue
		}
		v, err := strconv.ParseFloat(col("value"), 64)
		if err != nil {
			v = math.NaN() // OL
		}
		ch := byID[col("channel_id")]
		if ch == nil {
			ch = &SigChan{Name: col("channel"), Unit: col("unit"), Kind: KindMeter}
			byID[col("channel_id")] = ch
			c.Chans = append(c.Chans, ch)
		}
		rawT, rawCh, rawV = append(rawT, ts), append(rawCh, ch), append(rawV, v)
	}
	if t0 == 0 && len(rawT) > 0 {
		t0 = rawT[0]
		for _, t := range rawT {
			if t < t0 {
				t0 = t
			}
		}
	}
	for i, ts := range rawT {
		ch := rawCh[i]
		ch.T = append(ch.T, float64(ts-t0)/1e9)
		ch.V = append(ch.V, rawV[i])
	}
	for i := range c.Markers {
		c.Markers[i].T = (c.Markers[i].T - float64(t0)) / 1e9
	}
	var keep []*SigChan
	for _, ch := range c.Chans {
		if len(ch.T) > 0 {
			keep = append(keep, ch)
		}
	}
	c.Chans = keep
	if c.App != "" {
		c.Source = c.App + " timeline"
	}
	return c, nil
}

// ---------------------------------------------------------------- VCD

func parseVCD(s string) (*SigCapture, error) {
	c := &SigCapture{Source: "Logic capture (VCD)"}
	scale := 1e-9
	type vv struct {
		ch    *SigChan
		width int
	}
	ids := map[string][]vv{}
	var now float64
	tokens := strings.Fields(s)
	inDefs := true
	for i := 0; i < len(tokens); i++ {
		tk := tokens[i]
		if inDefs {
			switch tk {
			case "$version":
				var v []string
				for i++; i < len(tokens) && tokens[i] != "$end"; i++ {
					v = append(v, tokens[i])
				}
				ver := strings.Join(v, " ")
				if strings.Contains(ver, "Probie") || strings.Contains(ver, "Tracie") {
					c.App, c.Source = "Probie", "Probie logic capture"
				}
			case "$timescale":
				var v []string
				for i++; i < len(tokens) && tokens[i] != "$end"; i++ {
					v = append(v, tokens[i])
				}
				scale = parseTimescale(strings.Join(v, ""))
			case "$var":
				// $var wire 1 ! name $end
				if i+4 < len(tokens) {
					w, _ := strconv.Atoi(tokens[i+2])
					id, name := tokens[i+3], tokens[i+4]
					ch := &SigChan{Name: name, Kind: KindDigital}
					if w > 1 {
						ch.Kind = KindAnalog // a bus: shown as its value
					}
					ids[id] = append(ids[id], vv{ch, w})
					c.Chans = append(c.Chans, ch)
				}
				for i < len(tokens) && tokens[i] != "$end" {
					i++
				}
			case "$enddefinitions":
				inDefs = false
				for i < len(tokens) && tokens[i] != "$end" {
					i++
				}
			case "$date", "$comment", "$scope", "$upscope":
				for i < len(tokens) && tokens[i] != "$end" {
					i++
				}
			}
			continue
		}
		if tk == "" {
			continue
		}
		switch tk[0] {
		case '#':
			t, err := strconv.ParseFloat(tk[1:], 64)
			if err == nil {
				now = t * scale
			}
		case '$':
			// $dumpvars / $end etc
		case '0', '1', 'x', 'X', 'z', 'Z':
			id := tk[1:]
			v := 0.0
			if tk[0] == '1' {
				v = 1
			}
			for _, x := range ids[id] {
				addPoint(x.ch, now, v)
			}
		case 'b', 'B', 'r', 'R':
			if i+1 < len(tokens) {
				val := tk[1:]
				id := tokens[i+1]
				i++
				var v float64
				if tk[0] == 'b' || tk[0] == 'B' {
					n, _ := strconv.ParseUint(strings.Map(func(r rune) rune {
						if r == '1' {
							return '1'
						}
						return '0'
					}, val), 2, 64)
					v = float64(n)
				} else {
					v, _ = strconv.ParseFloat(val, 64)
				}
				for _, x := range ids[id] {
					addPoint(x.ch, now, v)
				}
			}
		}
	}
	// drop channels that never change and never had a value
	var keep []*SigChan
	for _, ch := range c.Chans {
		if len(ch.T) > 0 {
			keep = append(keep, ch)
		}
	}
	c.Chans = keep
	// the final timestamp marks the end of the capture
	if now > 0 {
		for _, ch := range c.Chans {
			if ch.T[len(ch.T)-1] < now {
				ch.T = append(ch.T, now)
				ch.V = append(ch.V, ch.V[len(ch.V)-1])
			}
		}
	}
	return c, nil
}

func addPoint(ch *SigChan, t, v float64) {
	n := len(ch.T)
	if n > 0 && ch.T[n-1] == t {
		ch.V[n-1] = v
		return
	}
	if n > 0 && ch.V[n-1] == v && ch.Kind == KindDigital {
		return // no change
	}
	ch.T = append(ch.T, t)
	ch.V = append(ch.V, v)
}

func parseTimescale(s string) float64 {
	s = strings.TrimSpace(s)
	i := 0
	for i < len(s) && (s[i] >= '0' && s[i] <= '9') {
		i++
	}
	n, _ := strconv.ParseFloat(s[:i], 64)
	if n == 0 {
		n = 1
	}
	unit := map[string]float64{"s": 1, "ms": 1e-3, "us": 1e-6, "ns": 1e-9, "ps": 1e-12, "fs": 1e-15}[strings.TrimSpace(s[i:])]
	if unit == 0 {
		unit = 1e-9
	}
	return n * unit
}

// ---------------------------------------------------------------- serial log

func parseSerialLog(s string) (*SigCapture, error) {
	c := &SigCapture{App: "Probie", Source: "Probie serial log"}
	sc := bufio.NewScanner(strings.NewReader(s))
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	first := true
	var nums []*SigChan
	named := false
	for sc.Scan() {
		if first {
			first = false
			continue
		}
		cells := splitCSV(sc.Text())
		if len(cells) < 4 {
			continue
		}
		t, err := strconv.ParseFloat(cells[1], 64)
		if err != nil {
			continue
		}
		c.Serial = append(c.Serial, SerialLine{T: t, Dir: cells[2], Text: cells[3]})
		if !named && len(cells) > 4 && cells[4] != "" {
			// name the value columns after the labels in the text (temp=21.5 hum=40)
			if m := labelRe.FindAllStringSubmatch(cells[3], -1); len(m) > 0 {
				named = true
				for k, g := range m {
					if k >= 16 {
						break
					}
					for len(nums) <= k {
						nums = append(nums, &SigChan{Name: "value" + strconv.Itoa(len(nums)+1), Kind: KindMeter})
					}
					nums[k].Name, nums[k].Unit = g[1], unitFromLabel(g[1])
				}
			}
		}
		for k := 4; k < len(cells) && k < 20; k++ {
			if cells[k] == "" {
				continue
			}
			v, err := strconv.ParseFloat(cells[k], 64)
			if err != nil {
				continue
			}
			for len(nums) <= k-4 {
				nums = append(nums, &SigChan{Name: "value" + strconv.Itoa(len(nums)+1), Kind: KindMeter})
			}
			nums[k-4].T = append(nums[k-4].T, t)
			nums[k-4].V = append(nums[k-4].V, v)
		}
	}
	for _, ch := range nums {
		if len(ch.T) >= 3 {
			c.Chans = append(c.Chans, ch)
		}
	}
	return c, nil
}

// splitCSV splits one CSV line, honouring quotes.
func splitCSV(line string) []string {
	var out []string
	var b strings.Builder
	q := false
	for i := 0; i < len(line); i++ {
		ch := line[i]
		switch {
		case q && ch == '"' && i+1 < len(line) && line[i+1] == '"':
			b.WriteByte('"')
			i++
		case ch == '"':
			q = !q
		case ch == ',' && !q:
			out = append(out, b.String())
			b.Reset()
		default:
			b.WriteByte(ch)
		}
	}
	return append(out, b.String())
}

// ---------------------------------------------------------------- plain CSV (Scopie, scopes, spreadsheets)

func parseCSV(s string) (*SigCapture, error) {
	c := &SigCapture{Source: "Waveform (CSV)"}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	lines := strings.Split(s, "\n")
	delim := ","
	sample := strings.Join(lines[:min(20, len(lines))], "\n")
	if strings.Count(sample, ";") > strings.Count(sample, ",") {
		delim = ";"
	} else if strings.Count(sample, "\t") > strings.Count(sample, ",") {
		delim = "\t"
	}
	var names []string
	var rows [][]float64
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln == "" || strings.HasPrefix(ln, "#") {
			continue
		}
		cells := strings.Split(ln, delim)
		vals := make([]float64, 0, len(cells))
		ok := true
		for _, cell := range cells {
			v, err := strconv.ParseFloat(strings.TrimSpace(strings.Trim(cell, `"`)), 64)
			if err != nil {
				ok = false
				break
			}
			vals = append(vals, v)
		}
		if !ok {
			if len(rows) == 0 {
				names = nil
				for _, cell := range cells {
					names = append(names, strings.TrimSpace(strings.Trim(cell, `"`)))
				}
			}
			continue
		}
		if len(vals) >= 2 {
			rows = append(rows, vals)
		}
	}
	if len(rows) < 4 {
		return nil, errors.New("Probie couldn't find a time column and at least one column of numbers")
	}
	nc := len(rows[0])
	if len(names) > 0 && strings.HasPrefix(strings.ToLower(names[0]), "time_s") {
		c.App, c.Source = "Scopie", "Scopie waveform"
	}
	tScale := 1.0
	if len(names) > 0 {
		h := strings.ToLower(names[0])
		switch {
		case strings.Contains(h, "(ms)") || strings.HasSuffix(h, "_ms"):
			tScale = 1e-3
		case strings.Contains(h, "(us)") || strings.Contains(h, "(µs)") || strings.HasSuffix(h, "_us"):
			tScale = 1e-6
		case strings.Contains(h, "(ns)") || strings.HasSuffix(h, "_ns"):
			tScale = 1e-9
		}
	}
	t0 := rows[0][0] * tScale
	for k := 1; k < nc; k++ {
		ch := &SigChan{Name: "CH" + strconv.Itoa(k), Unit: "V", Kind: KindAnalog}
		if k < len(names) && names[k] != "" {
			ch.Name = names[k]
			n := strings.ToLower(names[k])
			switch {
			case strings.HasSuffix(n, "_v") || strings.Contains(n, "(v)"):
				ch.Name = strings.TrimSuffix(strings.TrimSuffix(names[k], "_V"), "_v")
			case strings.Contains(n, "(a)") || strings.HasSuffix(n, "_a"):
				ch.Unit = "A"
			case strings.Contains(n, "°c") || strings.Contains(n, "temp"):
				ch.Unit = "°C"
			}
			ch.Name = strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(ch.Name, "(V)"), "(v)"))
			if strings.HasPrefix(strings.ToLower(ch.Name), "ch") {
				ch.Name = strings.ToUpper(ch.Name)
			}
		}
		for _, r := range rows {
			if k < len(r) {
				ch.T = append(ch.T, r[0]*tScale-t0)
				ch.V = append(ch.V, r[k])
			}
		}
		// a column of only 0s and 1s is a logic channel
		digital := true
		for _, v := range ch.V {
			if v != 0 && v != 1 {
				digital = false
				break
			}
		}
		if digital {
			ch.Kind, ch.Unit = KindDigital, ""
			ch.T, ch.V = compressEdges(ch.T, ch.V)
		} else if len(ch.T) > 2 && (ch.T[len(ch.T)-1]-ch.T[0])/float64(len(ch.T)) > 0.05 {
			ch.Kind = KindMeter // slow readings, not a waveform
		}
		c.Chans = append(c.Chans, ch)
	}
	return c, nil
}

func compressEdges(t, v []float64) ([]float64, []float64) {
	var ot, ov []float64
	for i := range t {
		if i == 0 || v[i] != v[i-1] {
			ot, ov = append(ot, t[i]), append(ov, v[i])
		}
	}
	if len(t) > 0 && ot[len(ot)-1] != t[len(t)-1] {
		ot, ov = append(ot, t[len(t)-1]), append(ov, v[len(v)-1])
	}
	return ot, ov
}

var labelRe = regexp.MustCompile(`([A-Za-z][A-Za-z0-9_ ]{0,15}?)\s*[=:]\s*-?[0-9.]+`)

func unitFromLabel(l string) string {
	l = strings.ToLower(l)
	switch {
	case strings.Contains(l, "temp"):
		return "°C"
	case strings.Contains(l, "hum") || strings.Contains(l, "%"):
		return "%"
	case strings.Contains(l, "volt") || l == "v" || strings.HasSuffix(l, "_v"):
		return "V"
	case strings.Contains(l, "curr") || l == "a" || l == "ma":
		return "A"
	}
	return ""
}
