// OWON oscilloscopes that understand OWON's SCPI dialect.
//
// Covers the popular HDS200 / HDS300 handheld 3-in-1 scopes (HDS242, HDS272,
// HDS2102, HDS2202, HDS307S ...) and SDS / XDS / TAO bench scopes.
//
// The handhelds appear on USB as vendor 5345 with two bulk endpoints (not
// USBTMC), so we talk to them directly. Models that are proper USBTMC
// instruments, or are reached over LAN, are found by the SCPI scan instead.
//
// Protocol references: OWON 'HDS200 Series SCPI Protocol' and the open-source
// projects owon-hds200-capture (linux4life798) and owon-scope-web (MIT).
package scopie

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const owonVID = 0x5345

func scanOwon(lan []string, notes *[]string) []*Candidate {
	var out []*Candidate
	for _, u := range listUSB() {
		if u.VID != owonVID || isUSBTMCInfo(u) {
			continue // USBTMC models are found by the SCPI scan
		}
		u := u
		label := strings.TrimSpace(strings.Replace(u.Desc, "Oscilloscope", "", 1))
		if label == "" || strings.EqualFold(label, "WinUSB Device") || strings.HasPrefix(label, "USB ") {
			label = "OWON scope"
		}
		c := &Candidate{Label: label, Family: "OWON", Detail: "USB", Ready: u.WinUSB,
			Factory: func() (Device, error) { return newOwon(&bulkLink{u: u}, ""), nil }}
		if !u.WinUSB {
			c.Detail = usbDriverHint(u)
		}
		out = append(out, c)
	}
	return out
}

// isUSBTMCInfo guesses from the Windows device list whether an OWON entry is a
// USBTMC instrument (those are listed under the 'USB Test and Measurement' class).
func isUSBTMCInfo(u usbInfo) bool {
	s := strings.ToLower(u.Service + " " + u.Desc)
	return strings.Contains(s, "usbtmc") || strings.Contains(s, "test and measurement")
}

// ---------------------------------------------------------------- helpers

var owonSIRe = regexp.MustCompile(`^\s*([-+]?[\d.]+)\s*([numkMµu]?)`)

// owonSI parses '200mV' -> 0.2 ; '1.0ms' -> 0.001
func owonSI(text any) (float64, bool) {
	s := fmt.Sprint(text)
	m := owonSIRe.FindStringSubmatch(s)
	if m == nil {
		return 0, false
	}
	v, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, false
	}
	mult := map[string]float64{"": 1, "n": 1e-9, "u": 1e-6, "µ": 1e-6, "m": 1e-3, "k": 1e3, "M": 1e6}[m[2]]
	return v * mult, true
}

// owonFmt formats 0.2 -> '200mv' ; 0.001 -> '1ms' (the forms OWON accepts).
func owonFmt(v float64, unit string) string {
	for _, sp := range []struct {
		s float64
		p string
	}{{1, ""}, {1e-3, "m"}, {1e-6, "u"}, {1e-9, "n"}} {
		if math.Abs(v) >= sp.s*0.999 {
			return strconv.FormatFloat(v/sp.s, 'g', -1, 64) + sp.p + unit
		}
	}
	return strconv.FormatFloat(v*1e9, 'g', -1, 64) + "n" + unit
}

// ci looks up a key case-insensitively in a JSON object.
func ci(d any, keys ...string) any {
	m, ok := d.(map[string]any)
	if !ok {
		return nil
	}
	for _, k := range keys {
		for mk, v := range m {
			if strings.EqualFold(mk, k) {
				return v
			}
		}
	}
	return nil
}

func ciStr(d any, keys ...string) string {
	v := ci(d, keys...)
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

// ---------------------------------------------------------------- device

type owonScope struct {
	l       link
	idn     string
	info    *DeviceInfo
	panel   *Panel
	running bool
	single  bool
	t0      time.Time
}

func newOwon(l link, idn string) *owonScope {
	info := frameInfo("OWON oscilloscope", 2)
	info.VDivs = stepsBetween(0.01, 10, vdiv125)
	info.TDivs = stepsBetween(5e-9, 50, tdiv125)
	info.TriggerBothEdges = false
	return &owonScope{l: l, idn: idn, info: info}
}

func (o *owonScope) Info() *DeviceInfo { return o.info }

func (o *owonScope) w(cmd string) error {
	if err := lWrite(o.l, cmd); err != nil {
		return devErr("Lost contact with the scope (%v).", err)
	}
	return nil
}

func (o *owonScope) q(cmd string) (string, error) {
	s, err := lQuery(o.l, cmd, 2000)
	if err != nil {
		return "", devErr("The scope did not answer (%v).", err)
	}
	return s, nil
}

func (o *owonScope) qLen4(cmd string) ([]byte, error) {
	if err := o.w(cmd); err != nil {
		return nil, err
	}
	return lReadLen4(o.l, 3000)
}

func (o *owonScope) Open(progress func(string)) error {
	if err := o.l.Open(); err != nil {
		return devErr("Could not open the OWON scope:\n%v", err)
	}
	if o.idn == "" {
		idn, err := o.q("*IDN?")
		if err != nil {
			return err
		}
		o.idn = idn
	}
	parts := append(strings.Split(o.idn, ","), "", "", "")
	model := strings.TrimSpace(parts[1])
	if model == "" {
		model = "oscilloscope"
	}
	o.info.Name = "OWON " + model
	if head, err := o.head(); err == nil {
		if chans, ok := ci(head, "CHANNEL").([]any); ok && len(chans) > 0 {
			o.info.Channels = clampInt(len(chans), 1, 4)
		}
	} else if m := regexp.MustCompile(`\d{3,4}`).FindString(model); m != "" && m[len(m)-1] == '4' {
		o.info.Channels = 4
	}
	o.info.Notes = o.idn + "\nOWON scopes send the on-screen waveform about 5-15 times per second."
	return nil
}

func (o *owonScope) Close() {
	if o.l != nil {
		_ = lWrite(o.l, ":RUNning RUN")
		o.l.Close()
	}
}

func (o *owonScope) head() (any, error) {
	raw, err := o.qLen4(":DATa:WAVe:SCReen:HEAD?")
	if err != nil {
		return nil, err
	}
	text := strings.Trim(string(raw), "\x00 \r\n")
	if i := strings.Index(text, "{"); i >= 0 {
		text = text[i:]
	}
	var v any
	if err := json.Unmarshal([]byte(text), &v); err != nil {
		return nil, devErr("The scope sent an unreadable status reply.")
	}
	return v, nil
}

func (o *owonScope) ApplyPanel(p *Panel) error {
	cp := *p
	o.panel = &cp
	var cmds []string
	for c := 0; c < o.info.Channels; c++ {
		ch := p.Ch[c]
		on := ch.Enabled || c == p.TrigSource
		n := c + 1
		if !on {
			cmds = append(cmds, fmt.Sprintf(":CH%d:DISPlay OFF", n))
			continue
		}
		cmds = append(cmds, fmt.Sprintf(":CH%d:DISPlay ON", n), fmt.Sprintf(":CH%d:PROBe 1X", n),
			fmt.Sprintf(":CH%d:SCALe %s", n, owonFmt(ch.VDiv, "v")), fmt.Sprintf(":CH%d:COUPling %s", n, ch.Coupling),
			fmt.Sprintf(":CH%d:OFFSet %.2f", n, ch.Pos))
	}
	src := fmt.Sprintf("CH%d", p.TrigSource+1)
	slope := "RISE"
	if p.TrigEdge == "falling" {
		slope = "FALL"
	}
	lvl := "0mv"
	if p.TrigLevel != 0 {
		lvl = owonFmt(p.TrigLevel, "v")
	}
	sweep := "NORMal"
	if p.TrigMode == "auto" {
		sweep = "AUTO"
	}
	// two spellings exist across OWON models; the scope ignores the one it doesn't know
	cmds = append(cmds, ":HORIzontal:SCALe "+owonFmt(p.TDiv, "s"),
		":TRIGger:SINGle:SOURce "+src, ":TRIGger:SINGle:EDGe:SOURce "+src,
		":TRIGger:SINGle:EDGe "+slope, ":TRIGger:SINGle:EDGe:SLOPe "+slope,
		":TRIGger:SINGle:EDGe:LEVel "+lvl, ":TRIGger:SINGle:SWEep "+sweep)
	for _, c := range cmds {
		if err := o.w(c); err != nil {
			return err
		}
	}
	time.Sleep(50 * time.Millisecond)
	return nil
}

func (o *owonScope) Start(single bool) error {
	o.running, o.single = true, single
	if single {
		if err := o.w(":TRIGger:SINGle:SWEep SINGle"); err != nil {
			return err
		}
	}
	o.t0 = time.Now()
	return o.w(":RUNning RUN")
}

func (o *owonScope) Stop() error {
	o.running = false
	return o.w(":RUNning STOP")
}

func (o *owonScope) ReadFrame(timeout float64) (*FrameData, error) {
	if !o.running || o.panel == nil {
		sleepS(timeout)
		return nil, nil
	}
	head, err := o.head()
	if err != nil {
		return nil, err
	}
	status := strings.ToUpper(ciStr(head, "RUNSTATUS"))
	if o.single {
		if status != "STOP" || time.Since(o.t0) < 300*time.Millisecond {
			sleepS(timeout)
			return nil, nil
		}
		o.running = false
	} else if strings.HasPrefix(status, "READ") && o.panel.TrigMode == "normal" {
		sleepS(timeout)
		return nil, nil
	}
	triggered := strings.HasPrefix(status, "TRIG") || o.single
	tdiv, ok := owonSI(ci(ci(head, "TIMEBASE"), "SCALE"))
	if !ok || tdiv <= 0 {
		tdiv = o.panel.TDiv
	}
	fd := newFrameData()
	nPts := 0
	chans, _ := ci(head, "CHANNEL").([]any)
	for _, ch := range chans {
		name := strings.ToUpper(ciStr(ch, "NAME"))
		disp := strings.ToUpper(ciStr(ch, "DISPLAY"))
		if !strings.HasPrefix(name, "CH") || (disp != "" && disp != "ON") {
			continue
		}
		c, err := strconv.Atoi(name[2:])
		if err != nil || c < 1 || c > 4 {
			continue
		}
		c--
		data, err := o.qLen4(":DATa:WAVe:SCReen:" + name + "?")
		if err != nil {
			return nil, err
		}
		nExp := 0
		if v := ci(ci(head, "SAMPLE"), "DATALEN", "FULLSCREEN"); v != nil {
			f, _ := strconv.ParseFloat(fmt.Sprint(v), 64)
			nExp = int(f)
		}
		var codes []float32
		perDiv, rng := 25.0, CodeRange{-128, 127}
		if nExp > 0 && len(data) >= 2*nExp {
			codes = make([]float32, nExp)
			for i := range codes {
				codes[i] = float32(int16(binary.LittleEndian.Uint16(data[2*i:])))
			}
			perDiv, rng = 25*256, CodeRange{-32768, 32767}
		} else {
			codes = make([]float32, len(data))
			for i, b := range data {
				codes[i] = float32(int8(b))
			}
		}
		vdiv, ok := owonSI(ci(ch, "SCALE"))
		if !ok || vdiv <= 0 {
			vdiv = o.panel.Ch[c].VDiv
		}
		offset, _ := strconv.ParseFloat(ciStr(ch, "OFFSET"), 64)
		if perDiv > 25 {
			offset *= 256
		}
		fd.Codes[c], fd.Scales[c], fd.Ranges[c] = codes, Scale{vdiv / perDiv, offset}, rng
		nPts = len(codes)
	}
	if nPts == 0 {
		return nil, nil
	}
	span := tdiv * float64(o.info.HDivs)
	fd.DT, fd.TFirst, fd.Triggered = span/float64(nPts), -span/2, triggered
	return fd, nil
}
