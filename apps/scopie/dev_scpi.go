// Bench oscilloscopes controlled with SCPI text commands over USB or LAN.
//
// Supported command dialects:
//   - Rigol: DS1000Z (DS1054Z...), DS2000, MSO5000, DHO800/900/1000/4000
//   - Keysight / Agilent InfiniiVision: DSOX1000/2000/3000/4000, DSO-X, MSO-X
//   - Tektronix: TBS1000/B/C, TBS2000, TDS1000/2000 series
//   - Siglent: SDS1000X-E, SDS2000X-E, SDS1000X/X+ (classic command set)
//
// LAN needs nothing at all: type the scope's IP address in the Connect
// window (VXI-11 is tried first, then raw sockets on 5025 and 5555).
// USB works through an installed VISA library (NI-VISA, Keysight IO -
// often already present if the scope's own software was installed), or
// directly over USBTMC once the scope has the WinUSB driver.
//
// Bench scopes send roughly 3-30 waveforms per second; they cannot match a
// streaming USB scope's update rate.
package scopie

import (
	"encoding/binary"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Vendors whose USB instruments are worth asking "*IDN?".
var tmVendors = map[uint16]bool{
	0x1AB1: true,               // Rigol
	0xF4EC: true, 0xF4ED: true, // Siglent
	0x0957: true, 0x2A8D: true, // Agilent / Keysight
	0x0699: true, // Tektronix
	0x5345: true, // OWON
	0x049F: true, // Hameg/R&S older
	0x0AAD: true, // Rohde & Schwarz
	0x05FF: true, // LeCroy
	0x3121: true, // Hantek (DSO2000/4000 USBTMC)
}

func scpiDialect(idn string) string {
	vendor := strings.ToUpper(strings.TrimSpace(strings.Split(idn, ",")[0]))
	switch {
	case strings.Contains(vendor, "RIGOL"):
		return "rigol"
	case strings.Contains(vendor, "SIGLENT"):
		return "siglent"
	case strings.Contains(vendor, "KEYSIGHT"), strings.Contains(vendor, "AGILENT"):
		return "keysight"
	case strings.Contains(vendor, "TEKTRONIX"):
		return "tek"
	case strings.Contains(vendor, "OWON"), strings.Contains(vendor, "LILLIPUT"):
		return "owon"
	}
	return ""
}

func probeIDN(l link) string {
	s, err := lQuery(l, "*IDN?", 1500)
	if err != nil || !strings.Contains(s, ",") {
		return ""
	}
	return s
}

// linkMaker recreates a link of the kind found during the scan.
type linkMaker func() link

func scanSCPI(lan []string, notes *[]string) []*Candidate {
	var out []*Candidate
	seen := map[string]bool{}
	add := func(mk linkMaker, name, idn, via string) {
		key := strings.ToUpper(idn)
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, scpiCandidate(mk, name, idn, via))
	}
	// 1. an installed VISA library (it owns its USB devices)
	for _, r := range visaFindUSB() {
		r := r
		mk := func() link { return &visaLink{rsrc: r} }
		l := mk()
		if l.Open() == nil {
			if idn := probeIDN(l); idn != "" {
				add(mk, r, idn, "USB, VISA")
			}
			l.Close()
		}
	}
	// 2. USBTMC over WinUSB
	for _, u := range listUSB() {
		if !u.WinUSB || u.Path == "" || !tmVendors[u.VID] {
			continue
		}
		u := u
		t := &usbtmcLink{u: u}
		if t.Open() != nil {
			continue
		}
		_, _, class, sub := t.d.Endpoints()
		idn := ""
		if class == 0xFE && sub == 0x03 {
			idn = probeIDN(t)
		}
		t.Close()
		if idn != "" {
			add(func() link { return &usbtmcLink{u: u} }, t.Name(), idn, "USB")
		}
	}
	// 3. network addresses the user typed
	for _, addr := range lan {
		addr = strings.TrimSpace(addr)
		if addr == "" {
			continue
		}
		l, idn, err := openNetLink(addr, probeIDN)
		if err != nil {
			out = append(out, &Candidate{Label: addr + " (no answer)", Family: "Bench scope", Ready: false,
				Detail: "Check the address, the network cable and that LAN control is enabled on the scope."})
			continue
		}
		var mk linkMaker
		switch v := l.(type) {
		case *vxiLink:
			host := v.host
			mk = func() link { return &vxiLink{host: host} }
		case *rawLink:
			host, port := v.host, v.port
			mk = func() link { return &rawLink{host: host, port: port} }
		}
		name := l.Name()
		l.Close()
		add(mk, name, idn, "LAN")
	}
	return out
}

func scpiCandidate(mk linkMaker, name, idn, via string) *Candidate {
	parts := append(strings.Split(idn, ","), "", "", "")
	label := strings.TrimSpace(titleCase(strings.TrimSpace(parts[0])) + " " + strings.TrimSpace(parts[1]))
	d := scpiDialect(idn)
	switch d {
	case "owon":
		return &Candidate{Label: label, Family: "OWON", Detail: via + ", SCPI", Ready: true,
			Factory: func() (Device, error) { return newOwon(mk(), idn), nil }}
	case "":
		return &Candidate{Label: label, Family: "Bench scope", Ready: false,
			Detail: "This make of scope is not supported yet."}
	}
	return &Candidate{Label: label, Family: "Bench scope", Detail: fmt.Sprintf("%s, SCPI (%s)", via, name), Ready: true,
		Factory: func() (Device, error) { return newSCPIScope(mk(), name, idn, d), nil }}
}

// ---------------------------------------------------------------- scope

type scpiFetch struct {
	codes  []float32
	k, z   float64
	dt, t0 float64
	rng    CodeRange
}

type dialect interface {
	hdivs() int
	setup()
	apply(p *Panel, n int, enabled []bool) error
	run(single bool) error
	stop() error
	status() (string, error) // trig | wait | auto | stop
	fetch(c int) (*scpiFetch, error)
}

type scpiScope struct {
	l       link
	name    string
	idn     string
	model   string
	d       dialect
	info    *DeviceInfo
	panel   *Panel
	running bool
	single  bool
	enabled []bool
	tStart  time.Time
}

func newSCPIScope(l link, name, idn, dial string) *scpiScope {
	parts := append(strings.Split(idn, ","), "", "", "")
	s := &scpiScope{l: l, name: name, idn: idn, model: strings.TrimSpace(parts[1]), enabled: []bool{true, true, true, true}}
	switch dial {
	case "rigol":
		s.d = newRigol(s)
	case "keysight":
		s.d = &keysight{s: s}
	case "tek":
		s.d = &tektronix{s: s, pre: "WFMOutpre"}
	default:
		s.d = newSiglent(s)
	}
	nch := 2
	if m := regexp.MustCompile(`\d{3,4}`).FindString(s.model); m != "" && m[len(m)-1] == '4' {
		nch = 4
	}
	info := frameInfo(strings.TrimSpace(titleCase(strings.TrimSpace(parts[0]))+" "+s.model), nch)
	info.HDivs = s.d.hdivs()
	info.Notes = fmt.Sprintf("%s\nConnected through %s.\nBench scopes update a few to about 30 times per second.", idn, name)
	s.info = info
	return s
}

func (s *scpiScope) Info() *DeviceInfo { return s.info }

func (s *scpiScope) Open(progress func(string)) error {
	if err := s.l.Open(); err != nil {
		return devErr("Could not open %s:\n%v", s.name, err)
	}
	s.d.setup()
	return nil
}

func (s *scpiScope) Close() {
	if s.l != nil {
		_ = s.d.stop()
		s.l.Close()
	}
}

func (s *scpiScope) w(cmd string) error {
	if err := lWrite(s.l, cmd); err != nil {
		return devErr("Lost contact with the scope (%v).", err)
	}
	return nil
}

func (s *scpiScope) tryW(cmd string) { _ = lWrite(s.l, cmd) }

func (s *scpiScope) q(cmd string) (string, error) {
	r, err := lQuery(s.l, cmd, 5000)
	if err != nil {
		return "", devErr("Lost contact with the scope (%v).", err)
	}
	return r, nil
}

func (s *scpiScope) qBlock(cmd string) ([]byte, error) {
	if err := s.w(cmd); err != nil {
		return nil, err
	}
	b, err := lReadBlock(s.l, 8000)
	if err != nil {
		if _, ok := err.(*DeviceError); ok {
			return nil, err
		}
		return nil, devErr("Lost contact with the scope while reading data (%v).", err)
	}
	return b, nil
}

func (s *scpiScope) ApplyPanel(p *Panel) error {
	cp := *p
	s.panel = &cp
	n := s.info.Channels
	for c := 0; c < n; c++ {
		s.enabled[c] = p.Ch[c].Enabled || c == p.TrigSource
	}
	if err := s.d.apply(&cp, n, s.enabled); err != nil {
		return err
	}
	if s.running {
		return s.d.run(s.single)
	}
	return nil
}

func (s *scpiScope) Start(single bool) error {
	s.running, s.single = true, single
	s.tStart = time.Now()
	return s.d.run(single)
}

func (s *scpiScope) Stop() error {
	s.running = false
	return s.d.stop()
}

func (s *scpiScope) ReadFrame(timeout float64) (*FrameData, error) {
	if !s.running || s.panel == nil {
		sleepS(timeout)
		return nil, nil
	}
	st, err := s.d.status()
	if err != nil {
		return nil, err
	}
	triggered := false
	if s.single {
		if st != "stop" || time.Since(s.tStart) < 200*time.Millisecond {
			sleepS(timeout)
			return nil, nil
		}
		s.running = false
		triggered = true
	} else {
		if st == "wait" {
			sleepS(timeout)
			return nil, nil
		}
		triggered = st == "trig"
	}
	var chans []int
	for c := 0; c < s.info.Channels; c++ {
		if s.enabled[c] {
			chans = append(chans, c)
		}
	}
	// with several channels, stop the scope so they all come from the same capture
	coherent := len(chans) > 1 && !s.single
	if coherent {
		if err := s.d.stop(); err != nil {
			return nil, err
		}
	}
	fd := newFrameData()
	gotDT := false
	var ferr error
	for _, c := range chans {
		f, err := s.d.fetch(c)
		if err != nil {
			ferr = err
			break
		}
		if f == nil || len(f.codes) == 0 {
			continue
		}
		fd.Codes[c], fd.Scales[c], fd.Ranges[c] = f.codes, Scale{f.k, f.z}, f.rng
		if !gotDT {
			fd.DT, fd.TFirst, gotDT = f.dt, f.t0, true
		}
	}
	if coherent && s.running {
		if err := s.d.run(false); err != nil && ferr == nil {
			ferr = err
		}
	}
	if ferr != nil {
		return nil, ferr
	}
	if !gotDT || fd.DT <= 0 {
		return nil, nil
	}
	fd.Triggered = triggered
	return fd, nil
}

// num reads a number from a reply like "1.0E-3", "TDIV 1.00E-03S" or "200mV".
var numRe = regexp.MustCompile(`[-+]?\d*\.?\d+(?:[eE][-+]?\d+)?`)

func num(s string, def float64) float64 {
	f := strings.Fields(strings.TrimSpace(s))
	if len(f) > 0 {
		last := strings.TrimRight(f[len(f)-1], "VvSs")
		if v, err := strconv.ParseFloat(last, 64); err == nil {
			return v
		}
	}
	if m := numRe.FindString(s); m != "" {
		if v, err := strconv.ParseFloat(m, 64); err == nil {
			return v
		}
	}
	return def
}

func fmtG(v float64) string { return strconv.FormatFloat(v, 'g', -1, 64) }

func preamble(s string) ([]float64, error) {
	parts := strings.Split(s, ",")
	if len(parts) < 10 {
		return nil, devErr("The scope sent an unexpected waveform description.")
	}
	out := make([]float64, len(parts))
	for i, p := range parts {
		out[i], _ = strconv.ParseFloat(strings.TrimSpace(p), 64)
	}
	return out, nil
}

func bytesU8(b []byte) []float32 {
	out := make([]float32, len(b))
	for i, v := range b {
		out[i] = float32(v)
	}
	return out
}

func bytesI8(b []byte) []float32 {
	out := make([]float32, len(b))
	for i, v := range b {
		out[i] = float32(int8(v))
	}
	return out
}

func wordsU16(b []byte) []float32 {
	out := make([]float32, len(b)/2)
	for i := range out {
		out[i] = float32(binary.LittleEndian.Uint16(b[2*i:]))
	}
	return out
}

func writeAll(s *scpiScope, cmds ...string) error {
	for _, c := range cmds {
		if err := s.w(c); err != nil {
			return err
		}
	}
	return nil
}

func onOff(b bool) string {
	if b {
		return "ON"
	}
	return "OFF"
}

// ================================================================ Rigol

type rigol struct {
	s    *scpiScope
	word bool
	hd   int
}

func newRigol(s *scpiScope) *rigol {
	m := strings.ToUpper(s.model)
	r := &rigol{s: s, word: strings.HasPrefix(m, "DHO"), hd: 12}
	switch {
	case strings.HasPrefix(m, "DS2") || strings.HasPrefix(m, "MSO2"):
		r.hd = 14
	case strings.HasPrefix(m, "DHO") || strings.HasPrefix(m, "MSO5") || strings.HasPrefix(m, "DS7") ||
		strings.HasPrefix(m, "MSO7") || strings.HasPrefix(m, "MSO8"):
		r.hd = 10
	}
	return r
}

func (r *rigol) hdivs() int { return r.hd }

func (r *rigol) form() string {
	if r.word {
		return ":WAV:FORM WORD"
	}
	return ":WAV:FORM BYTE"
}

func (r *rigol) setup() {
	r.s.tryW(":WAV:MODE NORM")
	r.s.tryW(r.form())
	r.s.tryW(":TRIG:MODE EDGE")
}

func (r *rigol) apply(p *Panel, n int, enabled []bool) error {
	s := r.s
	for c := 0; c < n; c++ {
		cp := p.Ch[c]
		if err := s.w(fmt.Sprintf(":CHAN%d:DISP %s", c+1, onOff(enabled[c]))); err != nil {
			return err
		}
		if !enabled[c] {
			continue
		}
		s.tryW(fmt.Sprintf(":CHAN%d:PROB 1", c+1))
		if err := writeAll(s, fmt.Sprintf(":CHAN%d:SCAL %s", c+1, fmtG(cp.VDiv)),
			fmt.Sprintf(":CHAN%d:OFFS %s", c+1, fmtG(cp.Pos*cp.VDiv)),
			fmt.Sprintf(":CHAN%d:COUP %s", c+1, cp.Coupling)); err != nil {
			return err
		}
	}
	slope := map[string]string{"rising": "POS", "falling": "NEG", "both": "RFAL"}[p.TrigEdge]
	sweep := "NORM"
	if p.TrigMode == "auto" {
		sweep = "AUTO"
	}
	return writeAll(s, ":TIM:MAIN:SCAL "+fmtG(p.TDiv), ":TIM:MAIN:OFFS "+fmtG(-p.TrigX*p.TDiv),
		fmt.Sprintf(":TRIG:EDG:SOUR CHAN%d", p.TrigSource+1), ":TRIG:EDG:SLOP "+slope,
		":TRIG:EDG:LEV "+fmtG(p.TrigLevel), ":TRIG:SWE "+sweep)
}

func (r *rigol) run(single bool) error {
	if single {
		return r.s.w(":SING")
	}
	return r.s.w(":RUN")
}

func (r *rigol) stop() error { return r.s.w(":STOP") }

func (r *rigol) status() (string, error) {
	st, err := r.s.q(":TRIG:STAT?")
	if err != nil {
		return "", err
	}
	switch strings.ToUpper(st) {
	case "TD":
		return "trig", nil
	case "WAIT", "RUN":
		return "wait", nil
	case "STOP":
		return "stop", nil
	}
	return "auto", nil
}

func (r *rigol) fetch(c int) (*scpiFetch, error) {
	s := r.s
	if err := writeAll(s, fmt.Sprintf(":WAV:SOUR CHAN%d", c+1), ":WAV:MODE NORM", r.form()); err != nil {
		return nil, err
	}
	ps, err := s.q(":WAV:PRE?")
	if err != nil {
		return nil, err
	}
	pre, err := preamble(ps)
	if err != nil {
		return nil, err
	}
	data, err := s.qBlock(":WAV:DATA?")
	if err != nil {
		return nil, err
	}
	xinc, xorig, xref, yinc, yorig, yref := pre[4], pre[5], pre[6], pre[7], pre[8], pre[9]
	f := &scpiFetch{k: yinc, z: yorig + yref, dt: xinc, t0: xorig - xref*xinc}
	if r.word {
		f.codes, f.rng = wordsU16(data), CodeRange{0, 65535}
	} else {
		f.codes, f.rng = bytesU8(data), CodeRange{0, 255}
	}
	return f, nil
}

// ================================================================ Keysight

type keysight struct{ s *scpiScope }

func (k *keysight) hdivs() int { return 10 }

func (k *keysight) setup() {
	for _, c := range []string{":WAVeform:FORMat BYTE", ":WAVeform:UNSigned 1", ":WAVeform:POINts:MODE NORMal", ":TRIGger:MODE EDGE"} {
		k.s.tryW(c)
	}
}

func (k *keysight) apply(p *Panel, n int, enabled []bool) error {
	s := k.s
	for c := 0; c < n; c++ {
		cp := p.Ch[c]
		if err := s.w(fmt.Sprintf(":CHANnel%d:DISPlay %d", c+1, boolInt(enabled[c]))); err != nil {
			return err
		}
		if !enabled[c] {
			continue
		}
		s.tryW(fmt.Sprintf(":CHANnel%d:PROBe 1", c+1))
		if err := writeAll(s, fmt.Sprintf(":CHANnel%d:SCALe %s", c+1, fmtG(cp.VDiv)),
			fmt.Sprintf(":CHANnel%d:OFFSet %s", c+1, fmtG(-cp.Pos*cp.VDiv)),
			fmt.Sprintf(":CHANnel%d:COUPling %s", c+1, cp.Coupling)); err != nil {
			return err
		}
	}
	slope := map[string]string{"rising": "POSitive", "falling": "NEGative", "both": "EITHer"}[p.TrigEdge]
	sweep := "NORMal"
	if p.TrigMode == "auto" {
		sweep = "AUTO"
	}
	return writeAll(s, ":TIMebase:SCALe "+fmtG(p.TDiv), ":TIMebase:POSition "+fmtG(-p.TrigX*p.TDiv),
		fmt.Sprintf(":TRIGger:EDGE:SOURce CHANnel%d", p.TrigSource+1), ":TRIGger:EDGE:SLOPe "+slope,
		":TRIGger:EDGE:LEVel "+fmtG(p.TrigLevel), ":TRIGger:SWEep "+sweep)
}

func (k *keysight) run(single bool) error {
	if single {
		return k.s.w(":SINGle")
	}
	return k.s.w(":RUN")
}

func (k *keysight) stop() error { return k.s.w(":STOP") }

func (k *keysight) status() (string, error) {
	op, err := k.s.q(":OPERegister:CONDition?")
	if err != nil {
		return "", err
	}
	ter, err := k.s.q(":TER?")
	if err != nil {
		return "", err
	}
	if int(num(op, 0))&8 == 0 {
		return "stop", nil
	}
	if int(num(ter, 0)) != 0 {
		return "trig", nil
	}
	sweep, err := k.s.q(":TRIGger:SWEep?")
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(strings.ToUpper(sweep), "AUTO") {
		return "auto", nil
	}
	return "wait", nil
}

func (k *keysight) fetch(c int) (*scpiFetch, error) {
	s := k.s
	if err := writeAll(s, fmt.Sprintf(":WAVeform:SOURce CHANnel%d", c+1), ":WAVeform:FORMat BYTE",
		":WAVeform:UNSigned 1", ":WAVeform:POINts 1000"); err != nil {
		return nil, err
	}
	ps, err := s.q(":WAVeform:PREamble?")
	if err != nil {
		return nil, err
	}
	pre, err := preamble(ps)
	if err != nil {
		return nil, err
	}
	data, err := s.qBlock(":WAVeform:DATA?")
	if err != nil {
		return nil, err
	}
	xinc, xorig, xref, yinc, yorig, yref := pre[4], pre[5], pre[6], pre[7], pre[8], pre[9]
	if yinc == 0 {
		yinc = 1
	}
	return &scpiFetch{codes: bytesU8(data), k: yinc, z: yref - yorig/yinc, dt: xinc, t0: xorig - xref*xinc,
		rng: CodeRange{0, 255}}, nil
}

// ================================================================ Tektronix

type tektronix struct {
	s   *scpiScope
	pre string
}

func (t *tektronix) hdivs() int { return 10 }

func (t *tektronix) setup() {
	s := t.s
	for _, c := range []string{"HEADer OFF", "VERBose ON", "DATa:ENCdg RIBinary", "DATa:WIDth 1", "TRIGger:MAIn:TYPe EDGE"} {
		s.tryW(c)
	}
	if _, err := s.q("WFMOutpre:YMUlt?"); err != nil {
		t.pre = "WFMPre"
		s.tryW("*CLS")
	}
}

func (t *tektronix) apply(p *Panel, n int, enabled []bool) error {
	s := t.s
	for c := 0; c < n; c++ {
		cp := p.Ch[c]
		if err := s.w(fmt.Sprintf("SELect:CH%d %s", c+1, onOff(enabled[c]))); err != nil {
			return err
		}
		if !enabled[c] {
			continue
		}
		s.tryW(fmt.Sprintf("CH%d:PRObe 1", c+1))
		if err := writeAll(s, fmt.Sprintf("CH%d:SCAle %s", c+1, fmtG(cp.VDiv)),
			fmt.Sprintf("CH%d:POSition %s", c+1, fmtG(cp.Pos)),
			fmt.Sprintf("CH%d:COUPling %s", c+1, cp.Coupling)); err != nil {
			return err
		}
	}
	if err := s.w("HORizontal:MAIn:SCAle " + fmtG(p.TDiv)); err != nil {
		return err
	}
	s.tryW("HORizontal:MAIn:POSition " + fmtG(-p.TrigX*p.TDiv))
	slope := "RISe"
	if p.TrigEdge == "falling" {
		slope = "FALL"
	}
	mode := "NORMal"
	if p.TrigMode == "auto" {
		mode = "AUTO"
	}
	return writeAll(s, fmt.Sprintf("TRIGger:MAIn:EDGE:SOUrce CH%d", p.TrigSource+1), "TRIGger:MAIn:EDGE:SLOpe "+slope,
		"TRIGger:MAIn:LEVel "+fmtG(p.TrigLevel), "TRIGger:MAIn:MODe "+mode)
}

func (t *tektronix) run(single bool) error {
	after := "RUNSTop"
	if single {
		after = "SEQuence"
	}
	return writeAll(t.s, "ACQuire:STOPAfter "+after, "ACQuire:STATE RUN")
}

func (t *tektronix) stop() error { return t.s.w("ACQuire:STATE STOP") }

func (t *tektronix) status() (string, error) {
	a, err := t.s.q("ACQuire:STATE?")
	if err != nil {
		return "", err
	}
	if int(num(a, 0)) == 0 {
		return "stop", nil
	}
	st, err := t.s.q("TRIGger:STATE?")
	if err != nil {
		return "", err
	}
	st = strings.ToUpper(st)
	switch {
	case strings.HasPrefix(st, "TRIG"), strings.HasPrefix(st, "SAVE"):
		return "trig", nil
	case strings.HasPrefix(st, "AUTO"):
		return "auto", nil
	}
	return "wait", nil
}

func (t *tektronix) fetch(c int) (*scpiFetch, error) {
	s := t.s
	if err := writeAll(s, fmt.Sprintf("DATa:SOUrce CH%d", c+1), "DATa:STARt 1", "DATa:STOP 1000000"); err != nil {
		return nil, err
	}
	vals := map[string]float64{}
	for _, k := range []struct {
		name string
		def  float64
	}{{"XINcr", 0}, {"PT_Off", 0}, {"XZEro", 0}, {"YMUlt", 1}, {"YZEro", 0}, {"YOFf", 0}} {
		r, err := s.q(t.pre + ":" + k.name + "?")
		if err != nil {
			return nil, err
		}
		vals[k.name] = num(r, k.def)
	}
	data, err := s.qBlock("CURVe?")
	if err != nil {
		return nil, err
	}
	ymult := vals["YMUlt"]
	if ymult == 0 {
		ymult = 1
	}
	return &scpiFetch{codes: bytesI8(data), k: ymult, z: vals["YOFf"] - vals["YZEro"]/ymult,
		dt: vals["XINcr"], t0: vals["XZEro"] - vals["PT_Off"]*vals["XINcr"], rng: CodeRange{-128, 127}}, nil
}

// ================================================================ Siglent

type siglent struct {
	s  *scpiScope
	hd int
	p  *Panel
}

const siglentPoints = 1400

func newSiglent(s *scpiScope) *siglent {
	g := &siglent{s: s, hd: 14}
	if !strings.Contains(strings.ToUpper(s.model), "X") {
		g.hd = 18
	}
	return g
}

func (g *siglent) hdivs() int { return g.hd }
func (g *siglent) setup()     { g.s.tryW("CHDR OFF") }

func (g *siglent) apply(p *Panel, n int, enabled []bool) error {
	s := g.s
	g.p = p
	for c := 0; c < n; c++ {
		cp := p.Ch[c]
		if err := s.w(fmt.Sprintf("C%d:TRA %s", c+1, onOff(enabled[c]))); err != nil {
			return err
		}
		if !enabled[c] {
			continue
		}
		s.tryW(fmt.Sprintf("C%d:ATTN 1", c+1))
		cpl := "D1M"
		if cp.Coupling == "AC" {
			cpl = "A1M"
		}
		if err := writeAll(s, fmt.Sprintf("C%d:VDIV %sV", c+1, fmtG(cp.VDiv)),
			fmt.Sprintf("C%d:OFST %sV", c+1, fmtG(cp.Pos*cp.VDiv)), fmt.Sprintf("C%d:CPL %s", c+1, cpl)); err != nil {
			return err
		}
	}
	if err := s.w("TDIV " + fmtG(p.TDiv) + "S"); err != nil {
		return err
	}
	s.tryW("TRDL " + fmtG(-p.TrigX*p.TDiv) + "S")
	src := fmt.Sprintf("C%d", p.TrigSource+1)
	slope := map[string]string{"rising": "POS", "falling": "NEG", "both": "WINDOW"}[p.TrigEdge]
	mode := "NORM"
	if p.TrigMode == "auto" {
		mode = "AUTO"
	}
	return writeAll(s, "TRSE EDGE,SR,"+src+",HT,OFF", src+":TRSL "+slope, src+":TRLV "+fmtG(p.TrigLevel)+"V", "TRMD "+mode)
}

func (g *siglent) run(single bool) error {
	s := g.s
	switch {
	case single:
		if err := s.w("TRMD SINGLE"); err != nil {
			return err
		}
	case g.p == nil || g.p.TrigMode == "auto":
		if err := s.w("TRMD AUTO"); err != nil {
			return err
		}
	default:
		if err := s.w("TRMD NORM"); err != nil {
			return err
		}
	}
	s.tryW("ARM")
	return nil
}

func (g *siglent) stop() error { return g.s.w("STOP") }

func (g *siglent) status() (string, error) {
	st, err := g.s.q("SAST?")
	if err != nil {
		return "", err
	}
	st = strings.ToUpper(st)
	switch {
	case strings.Contains(st, "STOP"):
		return "stop", nil
	case strings.Contains(st, "TRIG"):
		return "trig", nil
	case strings.Contains(st, "AUTO"):
		return "auto", nil
	}
	return "wait", nil
}

func (g *siglent) fetch(c int) (*scpiFetch, error) {
	s := g.s
	qn := func(cmd string, def float64) (float64, error) {
		r, err := s.q(cmd)
		if err != nil {
			return 0, err
		}
		return num(r, def), nil
	}
	sara, err := qn("SARA?", 1e6)
	if err != nil {
		return nil, err
	}
	tdiv, err := qn("TDIV?", 1e-3)
	if err != nil {
		return nil, err
	}
	sp := maxInt(1, int(sara*tdiv*float64(g.hd)/siglentPoints))
	if err := s.w(fmt.Sprintf("WFSU SP,%d,NP,%d,FP,0", sp, siglentPoints)); err != nil {
		return nil, err
	}
	vdiv, err := qn(fmt.Sprintf("C%d:VDIV?", c+1), 1)
	if err != nil {
		return nil, err
	}
	ofst, err := qn(fmt.Sprintf("C%d:OFST?", c+1), 0)
	if err != nil {
		return nil, err
	}
	delay := 0.0
	if r, err := s.q("TRDL?"); err == nil {
		delay = num(r, 0)
	}
	data, err := s.qBlock(fmt.Sprintf("C%d:WF? DAT2", c+1))
	if err != nil {
		return nil, err
	}
	if n := len(data); n >= 2 && data[n-1] == 10 && data[n-2] == 10 {
		data = data[:n-2]
	}
	k := vdiv / 25
	return &scpiFetch{codes: bytesI8(data), k: k, z: ofst / k, dt: float64(sp) / sara,
		t0: -(tdiv * float64(g.hd) / 2) - delay, rng: CodeRange{-128, 127}}, nil
}
