// PicoScope USB oscilloscopes, using Pico Technology's own driver libraries.
//
// Supported driver families (block mode, hardware trigger):
//   - ps2000:  PicoScope 2204A, 2205A and older 2000 models
//   - ps2000a: PicoScope 2206B/2207B/2208B, 2405A/2406B/2407B/2408B, MSO models
//   - ps3000a: PicoScope 3000 A/B/D series
//   - ps4000a: PicoScope 4000 A series (first 4 channels)
//   - ps5000a: PicoScope 5000 A/B/D series (8-bit mode)
//
// Pico's licence does not allow other programs to bundle these libraries, so
// the free 'PicoSDK' must be installed from picotech.com. The function
// signatures follow Pico's own ISC-licensed picosdk-python-wrappers.
package scopie

import (
	"fmt"
	"math"
	"strings"
	"time"
	"unicode"
)

const (
	picoVID        = 0x0CE9
	picoOK         = 0
	picoPowerNC    = 0x11A
	picoUSB3OnUSB2 = 0x11E
	picoSDKURL     = "https://www.picotech.com/downloads"
	picoPoints     = 2000
)

var picoRangesA = []float64{0.01, 0.02, 0.05, 0.1, 0.2, 0.5, 1, 2, 5, 10, 20, 50} // enum 0..11
var picoRanges2000 = []struct {
	id int
	v  float64
}{{1, 0.02}, {2, 0.05}, {3, 0.1}, {4, 0.2}, {5, 0.5}, {6, 1}, {7, 2}, {8, 5}, {9, 10}, {10, 20}}
var picoFamilies = []string{"ps2000a", "ps3000a", "ps4000a", "ps5000a"}

// picoLib is a loaded Pico driver library (see pico_windows.go).
type picoLib interface {
	call(name string, args ...uintptr) (uintptr, error)
}

func scanPico(lan []string, notes *[]string) []*Candidate {
	onUSB := false
	for _, u := range listUSB() {
		if u.VID == picoVID {
			onUSB = true
		}
	}
	if !picoSDKInstalled() {
		if onUSB {
			*notes = append(*notes, "A PicoScope is plugged in, but Pico's free PicoSDK drivers are not installed. "+
				"Click 'Get PicoScope drivers…', install PicoSDK and search again.")
		}
		return nil
	}
	var out []*Candidate
	for _, fam := range picoFamilies {
		lib := loadPico(fam)
		if lib == nil {
			continue
		}
		var count int16
		serials := make([]byte, 512)
		ln := int16(512)
		st, err := lib.call(fam+"EnumerateUnits", ptr(&count), ptr(&serials[0]), ptr(&ln))
		if err != nil || uint32(st) != picoOK || count == 0 {
			continue
		}
		for _, s := range strings.Split(goStr(serials), ",") {
			s := strings.TrimSpace(s)
			if s == "" {
				continue
			}
			fam := fam
			out = append(out, &Candidate{Label: fmt.Sprintf("PicoScope (%s series) %s", fam[2:], s), Family: "PicoScope",
				Detail: "USB, PicoSDK", Ready: true, Factory: func() (Device, error) { return newPicoA(fam, s), nil }})
		}
	}
	if onUSB && len(out) == 0 && loadPico("ps2000") != nil {
		out = append(out, &Candidate{Label: "PicoScope 2000 (2204A / 2205A)", Family: "PicoScope", Detail: "USB, PicoSDK",
			Ready: true, Factory: func() (Device, error) { return newPico2000(), nil }})
	}
	return out
}

func picoSDKInstalled() bool {
	for _, n := range append(append([]string(nil), picoFamilies...), "ps2000") {
		if loadPico(n) != nil {
			return true
		}
	}
	return false
}

func picoModelChannels(model string) int {
	var digits []rune
	for _, r := range model {
		if unicode.IsDigit(r) {
			digits = append(digits, r)
		}
	}
	if len(digits) >= 2 && strings.ContainsRune("248", digits[1]) {
		return minInt(4, int(digits[1]-'0'))
	}
	return 2
}

func picoInfo(name string, ch int) *DeviceInfo {
	info := frameInfo(name, ch)
	info.VDivs = stepsBetween(0.002, 5, vdiv125)
	info.TDivs = stepsBetween(5e-9, 50, tdiv125)
	info.Notes = name + " - connected through Pico's PicoSDK driver."
	return info
}

// ================================================================ A-API (2000a/3000a/4000a/5000a)

type picoA struct {
	fam, serial string
	lib         picoLib
	handle      int16
	info        *DeviceInfo
	oversample  bool
	maxADC      int
	panel       *Panel
	running     bool
	single      bool
	armed       bool
	tArm        time.Time
	chRange     [maxCh]float64
	offsets     [maxCh]float64
	timebase    uint32
	dt          float64
	n, pre      int
}

func newPicoA(fam, serial string) *picoA {
	return &picoA{fam: fam, serial: serial, lib: loadPico(fam), info: picoInfo("PicoScope "+serial, 2),
		oversample: fam == "ps2000a" || fam == "ps3000a"}
}

func (p *picoA) Info() *DeviceInfo { return p.info }

func (p *picoA) f(name string, args ...uintptr) (uint32, error) {
	r, err := p.lib.call(p.fam+name, args...)
	return uint32(r), err
}

func (p *picoA) check(name, what string, args ...uintptr) error {
	st, err := p.f(name, args...)
	if err != nil {
		return devErr("PicoScope driver problem while %s: %v", what, err)
	}
	if st != picoOK {
		return devErr("PicoScope error 0x%X while %s.", st, what)
	}
	return nil
}

func (p *picoA) Open(progress func(string)) error {
	if p.lib == nil {
		return devErr("Pico's %s driver is not installed (get PicoSDK from %s).", p.fam, picoSDKURL)
	}
	if progress != nil {
		progress("Opening the PicoScope (this takes a few seconds)…")
	}
	ser := cstr(p.serial)
	var st uint32
	var err error
	if p.fam == "ps5000a" {
		st, err = p.f("OpenUnit", ptr(&p.handle), ptr(&ser[0]), 0) // 8-bit resolution
	} else {
		st, err = p.f("OpenUnit", ptr(&p.handle), ptr(&ser[0]))
	}
	if err != nil {
		return devErr("PicoScope driver problem: %v", err)
	}
	if st == picoPowerNC || st == picoUSB3OnUSB2 {
		if s2, err := p.f("ChangePowerSource", uintptr(p.handle), uintptr(st)); err == nil {
			st = s2
		}
	}
	if st != picoOK {
		return devErr("PicoScope error 0x%X while opening the scope.", st)
	}
	info := make([]byte, 64)
	var req int16
	model := ""
	if s, err := p.f("GetUnitInfo", uintptr(p.handle), ptr(&info[0]), 64, ptr(&req), 3); err == nil && s == picoOK {
		model = goStr(info)
	}
	var mx int16
	if err := p.check("MaximumValue", "reading the ADC range", uintptr(p.handle), ptr(&mx)); err != nil {
		return err
	}
	p.maxADC = int(mx)
	if p.maxADC == 0 {
		p.maxADC = 32512
	}
	name := model
	if name == "" {
		name = p.serial
	}
	p.info = picoInfo("PicoScope "+name, picoModelChannels(model))
	if p.maxADC >= 40000 {
		p.info.Bits = 16
	}
	return nil
}

func (p *picoA) Close() {
	if p.lib != nil && p.handle > 0 {
		p.f("Stop", uintptr(p.handle))
		p.f("CloseUnit", uintptr(p.handle))
		p.handle = 0
	}
}

func (p *picoA) Start(single bool) error {
	p.running, p.single, p.armed = true, single, false
	return nil
}

func (p *picoA) Stop() error {
	p.running, p.armed = false, false
	p.f("Stop", uintptr(p.handle))
	return nil
}

func pickRange(ranges []float64, need float64) (int, float64) {
	for i, v := range ranges {
		if v >= need*0.999 {
			return i, v
		}
	}
	return len(ranges) - 1, ranges[len(ranges)-1]
}

func (p *picoA) ApplyPanel(pp *Panel) error {
	cp := *pp
	p.panel = &cp
	p.f("Stop", uintptr(p.handle))
	p.armed = false
	p.chRange, p.offsets = [maxCh]float64{}, [maxCh]float64{}
	for c := 0; c < p.info.Channels; c++ {
		ch := cp.Ch[c]
		on := ch.Enabled || c == cp.TrigSource
		idx, v := pickRange(picoRangesA, 4*ch.VDiv)
		offset := ch.Pos * ch.VDiv
		dc := boolInt(ch.Coupling == "DC")
		st, err := p.f("SetChannel", uintptr(p.handle), uintptr(c), uintptr(boolInt(on)), uintptr(dc), uintptr(idx), f32(offset))
		if err == nil && st != picoOK { // the offset is outside what this range allows
			offset = 0
			st, err = p.f("SetChannel", uintptr(p.handle), uintptr(c), uintptr(boolInt(on)), uintptr(dc), uintptr(idx), f32(0))
		}
		if err != nil || st != picoOK {
			return devErr("PicoScope error 0x%X while setting up channel %d.", st, c+1)
		}
		if on {
			p.chRange[c], p.offsets[c] = v, offset
		}
	}
	hd := float64(p.info.HDivs)
	want := cp.TDiv * hd / picoPoints
	tb, dt, err := p.findTimebase(want)
	if err != nil {
		return err
	}
	p.timebase, p.dt = tb, dt
	p.n = maxInt(100, int(math.Round(cp.TDiv*hd/p.dt)))
	p.pre = int(float64(p.n) * (cp.TrigX + hd/2) / hd)
	src := cp.TrigSource
	rv := p.chRange[src]
	if rv == 0 {
		rv = 1
	}
	thr := int(clampF((cp.TrigLevel+p.offsets[src])/rv*float64(p.maxADC), -float64(p.maxADC), float64(p.maxADC)))
	dir := map[string]int{"rising": 2, "falling": 3, "both": 4}[cp.TrigEdge]
	auto := 0
	if cp.TrigMode != "normal" {
		auto = maxInt(20, int(float64(p.n)*p.dt*3000))
	}
	return p.check("SetSimpleTrigger", "setting the trigger", uintptr(p.handle), 1, uintptr(src),
		uintptr(int16(thr)), uintptr(dir), 0, uintptr(int16(auto)))
}

func (p *picoA) interval(tb uint32) (float64, bool) {
	var ns float32
	var mx int32
	var st uint32
	var err error
	if p.oversample {
		st, err = p.f("GetTimebase2", uintptr(p.handle), uintptr(tb), picoPoints, ptr(&ns), 1, ptr(&mx), 0)
	} else {
		st, err = p.f("GetTimebase2", uintptr(p.handle), uintptr(tb), picoPoints, ptr(&ns), ptr(&mx), 0)
	}
	if err != nil || st != picoOK {
		return 0, false
	}
	return float64(ns) * 1e-9, true
}

func (p *picoA) findTimebase(want float64) (uint32, float64, error) {
	tb := uint32(0)
	// the fastest timebases can be unavailable with several channels on: skip them
	for tb < 64 {
		if _, ok := p.interval(tb); ok {
			break
		}
		tb++
	}
	lo, hi := tb, tb
	for hi < 1<<30 {
		if dt, ok := p.interval(hi); ok && dt >= want {
			break
		}
		hi = hi*2 + 1
	}
	for lo < hi { // binary search: smallest timebase with dt >= want
		mid := (lo + hi) / 2
		if dt, ok := p.interval(mid); ok && dt >= want {
			hi = mid
		} else {
			lo = mid + 1
		}
	}
	bestTB, bestDT, found := uint32(0), 0.0, false
	for _, cand := range []int64{int64(lo) - 1, int64(lo)} {
		if cand < 0 {
			continue
		}
		if dt, ok := p.interval(uint32(cand)); ok && dt > 0 {
			if !found || math.Abs(math.Log(dt/want)) < math.Abs(math.Log(bestDT/want)) {
				bestTB, bestDT, found = uint32(cand), dt, true
			}
		}
	}
	if !found {
		return 0, 0, devErr("The PicoScope could not find a suitable sample rate.")
	}
	return bestTB, bestDT, nil
}

func (p *picoA) ReadFrame(timeout float64) (*FrameData, error) {
	if !p.running || p.panel == nil {
		sleepS(timeout)
		return nil, nil
	}
	if !p.armed {
		var indisposed int32
		var err error
		if p.oversample {
			err = p.check("RunBlock", "starting a capture", uintptr(p.handle), uintptr(p.pre), uintptr(p.n-p.pre),
				uintptr(p.timebase), 1, ptr(&indisposed), 0, 0, 0)
		} else {
			err = p.check("RunBlock", "starting a capture", uintptr(p.handle), uintptr(p.pre), uintptr(p.n-p.pre),
				uintptr(p.timebase), ptr(&indisposed), 0, 0, 0)
		}
		if err != nil {
			return nil, err
		}
		p.armed, p.tArm = true, time.Now()
	}
	deadline := time.Now().Add(time.Duration(timeout * 1e9))
	for {
		var ready int16
		if err := p.check("IsReady", "waiting for data", uintptr(p.handle), ptr(&ready)); err != nil {
			return nil, err
		}
		if ready != 0 {
			break
		}
		if time.Now().After(deadline) {
			return nil, nil
		}
		time.Sleep(2 * time.Millisecond)
	}
	p.armed = false
	bufs := map[int][]int16{}
	for c := 0; c < p.info.Channels; c++ {
		if p.chRange[c] == 0 {
			continue
		}
		b := make([]int16, p.n)
		bufs[c] = b
		if err := p.check("SetDataBuffer", "preparing the data buffer", uintptr(p.handle), uintptr(c), ptr(&b[0]), uintptr(p.n), 0, 0); err != nil {
			return nil, err
		}
	}
	n := uint32(p.n)
	var overflow int16
	if err := p.check("GetValues", "reading the data", uintptr(p.handle), 0, ptr(&n), 1, 0, 0, ptr(&overflow)); err != nil {
		return nil, err
	}
	fd := newFrameData()
	for c, b := range bufs {
		k := p.chRange[c] / float64(p.maxADC)
		codes := make([]float32, minInt(int(n), len(b)))
		for i := range codes {
			codes[i] = float32(b[i])
		}
		fd.Codes[c], fd.Scales[c] = codes, Scale{k, p.offsets[c] / k}
		fd.Ranges[c] = CodeRange{-float64(p.maxADC), float64(p.maxADC)}
	}
	// a capture that took about as long as the auto-trigger timeout was not triggered
	auto := float64(maxInt(20, int(float64(p.n)*p.dt*3000)))
	fd.Triggered = !(p.panel.TrigMode == "auto" && time.Since(p.tArm).Seconds()*1000 >= auto*0.95)
	if p.single {
		p.running = false
	}
	fd.DT, fd.TFirst = p.dt, -float64(p.pre)*p.dt
	return fd, nil
}

// ================================================================ ps2000

type pico2000 struct {
	lib      picoLib
	handle   int16
	info     *DeviceInfo
	panel    *Panel
	running  bool
	single   bool
	armed    bool
	tArm     time.Time
	chRange  [2]float64
	timebase int
	dt       float64
	n, pre   int
}

const pico2000Max = 32767

func newPico2000() *pico2000 {
	return &pico2000{lib: loadPico("ps2000"), info: picoInfo("PicoScope 2000", 2)}
}

func (p *pico2000) Info() *DeviceInfo { return p.info }

func (p *pico2000) fn(name string, args ...uintptr) int16 {
	r, err := p.lib.call("ps2000_"+name, args...)
	if err != nil {
		return 0
	}
	return int16(r)
}

func (p *pico2000) Open(progress func(string)) error {
	if p.lib == nil {
		return devErr("Pico's ps2000 driver is not installed (get PicoSDK from %s).", picoSDKURL)
	}
	if progress != nil {
		progress("Opening the PicoScope (this takes a few seconds)…")
	}
	h := p.fn("open_unit")
	if h <= 0 {
		return devErr("No PicoScope 2000 could be opened (is PicoScope software running?).")
	}
	p.handle = h
	info := make([]byte, 32)
	if p.fn("get_unit_info", uintptr(h), ptr(&info[0]), 32, 3) > 0 {
		p.info.Name = "PicoScope " + goStr(info)
	}
	p.info.VDivs = stepsBetween(0.005, 5, vdiv125)
	return nil
}

func (p *pico2000) Close() {
	if p.handle > 0 {
		p.fn("stop", uintptr(p.handle))
		p.fn("close_unit", uintptr(p.handle))
		p.handle = 0
	}
}

func (p *pico2000) Start(single bool) error {
	p.running, p.single, p.armed = true, single, false
	return nil
}

func (p *pico2000) Stop() error {
	p.running, p.armed = false, false
	p.fn("stop", uintptr(p.handle))
	return nil
}

func (p *pico2000) ApplyPanel(pp *Panel) error {
	cp := *pp
	p.panel = &cp
	p.fn("stop", uintptr(p.handle))
	p.armed = false
	p.chRange = [2]float64{}
	for c := 0; c < 2; c++ {
		ch := cp.Ch[c]
		on := ch.Enabled || c == cp.TrigSource
		need := (4 + math.Abs(ch.Pos)) * ch.VDiv
		r := picoRanges2000[len(picoRanges2000)-1]
		for _, x := range picoRanges2000 {
			if x.v >= need*0.999 {
				r = x
				break
			}
		}
		p.fn("set_channel", uintptr(p.handle), uintptr(c), uintptr(boolInt(on)), uintptr(boolInt(ch.Coupling == "DC")), uintptr(r.id))
		if on {
			p.chRange[c] = r.v
		}
	}
	hd := float64(p.info.HDivs)
	want := cp.TDiv * hd / picoPoints
	best, bestDT := -1, 0.0
	for tb := 0; tb < 24; tb++ {
		var interval int32
		var units int16
		var mx int32
		if p.fn("get_timebase", uintptr(p.handle), uintptr(tb), picoPoints, ptr(&interval), ptr(&units), 1, ptr(&mx)) != 0 {
			dt := float64(interval) * 1e-9
			if dt > 0 && (best < 0 || math.Abs(math.Log(dt/want)) < math.Abs(math.Log(bestDT/want))) {
				best, bestDT = tb, dt
			}
		}
	}
	if best < 0 {
		return devErr("The PicoScope could not find a suitable sample rate.")
	}
	p.timebase, p.dt = best, bestDT
	p.n = clampInt(int(math.Round(cp.TDiv*hd/p.dt)), 100, 8000)
	frac := (cp.TrigX + hd/2) / hd
	p.pre = int(float64(p.n) * frac)
	src := minInt(cp.TrigSource, 1)
	rv := p.chRange[src]
	if rv == 0 {
		rv = 1
	}
	thr := int(clampF(cp.TrigLevel/rv*pico2000Max, -pico2000Max, pico2000Max))
	dir := boolInt(cp.TrigEdge == "falling")
	auto := 0
	if cp.TrigMode != "normal" {
		auto = maxInt(20, int(float64(p.n)*p.dt*3000))
	}
	p.fn("set_trigger", uintptr(p.handle), uintptr(src), uintptr(int16(thr)), uintptr(dir),
		uintptr(int16(-frac*100)), uintptr(int16(auto)))
	return nil
}

func (p *pico2000) ReadFrame(timeout float64) (*FrameData, error) {
	if !p.running || p.panel == nil {
		sleepS(timeout)
		return nil, nil
	}
	if !p.armed {
		var indisposed int32
		if p.fn("run_block", uintptr(p.handle), uintptr(p.n), uintptr(p.timebase), 1, ptr(&indisposed)) == 0 {
			return nil, devErr("The PicoScope refused to start a capture.")
		}
		p.armed, p.tArm = true, time.Now()
	}
	deadline := time.Now().Add(time.Duration(timeout * 1e9))
	for p.fn("ready", uintptr(p.handle)) == 0 {
		if time.Now().After(deadline) {
			return nil, nil
		}
		time.Sleep(2 * time.Millisecond)
	}
	p.armed = false
	a := make([]int16, p.n)
	b := make([]int16, p.n)
	var overflow int16
	r, err := p.lib.call("ps2000_get_values", uintptr(p.handle), ptr(&a[0]), ptr(&b[0]), 0, 0, ptr(&overflow), uintptr(p.n))
	got := int(int32(r))
	if err != nil || got <= 0 {
		return nil, nil
	}
	fd := newFrameData()
	for c, buf := range [][]int16{a, b} {
		if p.chRange[c] == 0 {
			continue
		}
		codes := make([]float32, minInt(got, len(buf)))
		for i := range codes {
			codes[i] = float32(buf[i])
		}
		fd.Codes[c], fd.Scales[c] = codes, Scale{p.chRange[c] / pico2000Max, 0}
		fd.Ranges[c] = CodeRange{-pico2000Max, pico2000Max}
	}
	auto := float64(maxInt(20, int(float64(p.n)*p.dt*3000)))
	fd.Triggered = !(p.panel.TrigMode == "auto" && time.Since(p.tArm).Seconds()*1000 >= auto*0.95)
	if p.single {
		p.running = false
	}
	fd.DT, fd.TFirst = p.dt, -float64(p.pre)*p.dt
	return fd, nil
}
