// Older Hantek USB scopes: DSO-2090, DSO-2150, DSO-2250, DSO-5200, DSO-5200A
// (also sold as Voltcraft DSO-2090 and similar).
//
// These need Hantek's own firmware loaded after every power-up. That firmware
// belongs to Hantek and may not be redistributed, so it is extracted from the
// driver file that comes with Hantek's software (e.g. DSO2090X861.SYS) using
// 'Import firmware…' in the Connect window. The extractor is a port of
// sigrok-util's sigrok-fwextract-hantek-dso (GPL-3.0).
//
// Protocol reference: OpenHantek (legacy, GPL) and libsigrok's hantek-dso
// driver (GPL-3.0). This driver is EXPERIMENTAL.
package scopie

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	hdEPOut, hdEPIn                        = 0x02, 0x86
	hdCtrlValue, hdCtrlBegin, hdCtrlOffset = 0xA2, 0xB3, 0xB4
	hdCtrlRelays                           = 0xB5
	hdRecord                               = 10240
)

var hdBegin = []byte{0x0F, 0x03, 0x03, 0x03, 0x68, 0xAC, 0xFE, 0x00, 0x01, 0x00}

type hdGain struct {
	id   int
	full float64 // volts across the 8 vertical divisions
}

var hdGain2x = []hdGain{{0, 0.08}, {1, 0.16}, {2, 0.40}, {0, 0.80}, {1, 1.60}, {2, 4.00}, {0, 8.00}, {1, 16.0}, {2, 40.0}}
var hdGain2250 = []hdGain{{0, 0.08}, {2, 0.16}, {3, 0.40}, {0, 0.80}, {2, 1.60}, {3, 4.00}, {0, 8.00}, {2, 16.0}, {3, 40.0}}
var hdGain5200 = []hdGain{{1, 0.16}, {0, 0.40}, {0, 0.80}, {1, 1.60}, {0, 4.00}, {0, 8.00}, {1, 16.0}, {0, 40.0}, {0, 80.0}}
var hdLimit5200 = []float64{368, 454, 908, 368, 454, 908, 368, 454, 908}

type hdModel struct {
	name, key string
	loaders   [][2]uint16
	pid       uint16
	kind      string // "2090" | "2250" | "5200": which command set
	base      float64
	gain      []hdGain
	ready     byte // capture-state value meaning "data ready"
	bits      int
	limits    []float64
	maxDown   int
}

var hdModels = []hdModel{
	{"Hantek DSO-2090", "2090", [][2]uint16{{0x04B4, 0x2090}, {0x04B4, 0x8613}}, 0x2090, "2090", 50e6, hdGain2x, 2, 8, nil, 131072},
	{"Hantek DSO-2150", "2150", [][2]uint16{{0x04B4, 0x2150}}, 0x2150, "2090", 50e6, hdGain2x, 2, 8, nil, 131072},
	{"Hantek DSO-2250", "2250", [][2]uint16{{0x04B4, 0x2250}}, 0x2250, "2250", 100e6, hdGain2250, 3, 8, nil, 65536},
	{"Hantek DSO-5200", "5200", [][2]uint16{{0x04B4, 0x5200}}, 0x5200, "5200", 100e6, hdGain5200, 7, 10, hdLimit5200, 131072},
	{"Hantek DSO-5200A", "5200A", [][2]uint16{{0x04B4, 0x520A}}, 0x520A, "5200", 100e6, hdGain5200, 7, 10, hdLimit5200, 131072},
}

func (m *hdModel) limit(g int) float64 {
	if m.limits == nil {
		return 255
	}
	return m.limits[g]
}

func (m *hdModel) isLoader(u usbInfo) bool {
	for _, l := range m.loaders {
		if u.VID == l[0] && u.PID == l[1] {
			return true
		}
	}
	return false
}

func (m *hdModel) isLoaded(u usbInfo) bool { return u.VID == 0x04B5 && u.PID == m.pid }

func hdFirmwareFile(m *hdModel) string {
	return filepath.Join(userFirmwareDir(), "hantek-dso-"+m.key+".fw")
}

// hdFirmware reads the extracted firmware (also where the Python version kept it).
func hdFirmware(m *hdModel) ([]byte, error) {
	b, err := os.ReadFile(hdFirmwareFile(m))
	if err != nil {
		if d := legacyFirmwareDir(); d != "" {
			if b2, err2 := os.ReadFile(filepath.Join(filepath.Dir(d), "hantek-dso-"+m.key+".fw")); err2 == nil {
				return b2, nil
			}
		}
	}
	return b, err
}

func scanHantekDSO(lan []string, notes *[]string) []*Candidate {
	var out []*Candidate
	for _, u := range listUSB() {
		for i := range hdModels {
			m := &hdModels[i]
			loaded := m.isLoaded(u)
			if !loaded && !m.isLoader(u) {
				continue
			}
			_, err := hdFirmware(m)
			haveFW := loaded || err == nil
			c := &Candidate{Label: m.name, Family: "Hantek DSO-2xxx/5xxx", Ready: haveFW && u.WinUSB}
			switch {
			case !u.WinUSB:
				c.Detail = usbDriverHint(u)
			case !haveFW:
				c.Detail = "Needs firmware: use 'Import firmware…' with Hantek's driver file"
			default:
				c.Detail = "USB (experimental)"
			}
			c.Factory = func() (Device, error) { return newHantekDSO(m), nil }
			out = append(out, c)
			break
		}
	}
	return out
}

// ---------------------------------------------------------------- firmware extraction

type peSection struct{ vaddr, rptr, rsize uint32 }

// peSymbols reads the COFF symbol table of a Windows driver (.sys).
func peSymbols(data []byte) (map[string][2]int64, []peSection, error) {
	bad := devErr("That file is not a Windows driver (.sys).")
	if len(data) < 0x40 || string(data[:2]) != "MZ" {
		return nil, nil, bad
	}
	pe := int(binary.LittleEndian.Uint32(data[0x3C:]))
	if pe+24 > len(data) || string(data[pe:pe+4]) != "PE\x00\x00" {
		return nil, nil, bad
	}
	nsec := int(binary.LittleEndian.Uint16(data[pe+6:]))
	symtab := int(binary.LittleEndian.Uint32(data[pe+12:]))
	nsym := int(binary.LittleEndian.Uint32(data[pe+16:]))
	opt := int(binary.LittleEndian.Uint16(data[pe+20:]))
	secOff := pe + 24 + opt
	var secs []peSection
	for i := 0; i < nsec; i++ {
		o := secOff + 40*i
		if o+24 > len(data) {
			break
		}
		secs = append(secs, peSection{binary.LittleEndian.Uint32(data[o+12:]), binary.LittleEndian.Uint32(data[o+20:]), binary.LittleEndian.Uint32(data[o+16:])})
	}
	if symtab == 0 || nsym == 0 || symtab+nsym*18 > len(data) {
		return nil, nil, devErr("This driver file has no symbol table; use the 32-bit (x86) driver file, e.g. DSO2090X861.SYS.")
	}
	strtab := symtab + nsym*18
	syms := map[string][2]int64{}
	for i := 0; i < nsym; {
		o := symtab + i*18
		raw := data[o : o+8]
		value := int64(binary.LittleEndian.Uint32(data[o+8:]))
		secnum := int64(int16(binary.LittleEndian.Uint16(data[o+12:])))
		naux := int(data[o+17])
		var name string
		if binary.LittleEndian.Uint32(raw) == 0 {
			so := strtab + int(binary.LittleEndian.Uint32(raw[4:]))
			if so < len(data) {
				end := bytes.IndexByte(data[so:], 0)
				if end >= 0 {
					name = string(data[so : so+end])
				}
			}
		} else {
			name = strings.TrimRight(string(raw), "\x00")
		}
		syms[name] = [2]int64{value, secnum}
		i += 1 + naux
	}
	return syms, secs, nil
}

// unsparse decodes the driver's 22-byte records: len(2) addr(2) type(1) data(16) pad(1).
func unsparse(data []byte, start int) []byte {
	blob := make([]byte, 0x4000)
	maxAddr := 0
	for p := start; p+22 <= len(data) && data[p+4] == 0; p += 22 {
		n := int(binary.LittleEndian.Uint16(data[p:]))
		addr := int(binary.LittleEndian.Uint16(data[p+2:]))
		if n == 0 || n > 16 {
			break
		}
		if addr+n <= 0x4000 { // FX2 RAM only; higher addresses are registers
			copy(blob[addr:], data[p+5:p+5+n])
			maxAddr = maxInt(maxAddr, addr+n)
		}
	}
	return blob[:maxAddr]
}

// extractHantekFirmware pulls the FX2 firmware out of Hantek's driver (.sys,
// or a .zip containing it) and stores it for the matching model(s).
func extractHantekFirmware(name string, data []byte) ([]string, error) {
	type file struct {
		name string
		data []byte
	}
	var files []file
	if zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data))); err == nil {
		for _, f := range zr.File {
			if strings.HasSuffix(strings.ToLower(f.Name), "1.sys") {
				rc, err := f.Open()
				if err != nil {
					continue
				}
				b, _ := io.ReadAll(io.LimitReader(rc, 16<<20))
				rc.Close()
				files = append(files, file{filepath.Base(f.Name), b})
			}
		}
		if len(files) == 0 {
			return nil, devErr("No Hantek driver file (…1.sys) was found in that archive.")
		}
	} else {
		files = append(files, file{filepath.Base(name), data})
	}
	var saved, errs []string
	for _, f := range files {
		err := func() error {
			syms, secs, err := peSymbols(f.data)
			if err != nil {
				return err
			}
			fw, ok := syms["_firmware"]
			if !ok {
				return devErr("no firmware inside")
			}
			value, secnum := fw[0], fw[1]
			var vaddr, rptr int64
			if secnum > 0 && int(secnum) <= len(secs) {
				s := secs[secnum-1]
				vaddr, rptr = int64(s.vaddr), int64(s.rptr)
			}
			// sigrok treats the symbol value as a file offset; some files use section offsets.
			var blob []byte
			for _, start := range []int64{value, rptr + value, rptr + value - vaddr} {
				if start >= 0 && start < int64(len(f.data)) {
					blob = unsparse(f.data, int(start))
					if len(blob) > 1000 {
						break
					}
				}
			}
			if len(blob) < 1000 {
				return devErr("the firmware could not be decoded")
			}
			low := strings.ToLower(f.name)
			matched := false
			for i := range hdModels {
				m := &hdModels[i]
				tag := "dso" + strings.ToLower(m.key)
				if strings.HasPrefix(low, tag) && !(m.key == "5200" && strings.HasPrefix(low, "dso5200a")) {
					if err := os.WriteFile(hdFirmwareFile(m), blob, 0o644); err != nil {
						return err
					}
					saved = append(saved, m.name)
					matched = true
				}
			}
			if !matched {
				return devErr("the file name doesn't say which scope it is for (expected e.g. DSO2090X861.SYS)")
			}
			return nil
		}()
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", f.name, err))
		}
	}
	if len(saved) == 0 {
		return nil, devErr("No firmware could be extracted.\n%s", strings.Join(errs, "\n"))
	}
	return saved, nil
}

// ---------------------------------------------------------------- device

type hantekDSO struct {
	m       *hdModel
	info    *DeviceInfo
	d       usbDevice
	vdivs   []float64
	panel   *Panel
	running bool
	single  bool
	state   string
	limits  [2][9][2]float64
	gid     [2]int
	offReal [2]float64
	rate    float64
	pre     int
	tArm    time.Time
	forced  bool
}

func newHantekDSO(m *hdModel) *hantekDSO {
	s := &hantekDSO{m: m, state: "idle"}
	for _, g := range m.gain {
		s.vdivs = append(s.vdivs, g.full/8)
	}
	info := frameInfo(m.name, 2)
	info.VDivs = s.vdivs
	info.TDivs = []float64{1e-6, 2e-6, 5e-6, 10e-6, 20e-6, 50e-6, 100e-6, 200e-6, 500e-6,
		1e-3, 2e-3, 5e-3, 10e-3, 20e-3, 50e-3, 100e-3, 200e-3, 500e-3}
	info.Bits = m.bits
	info.Notes = m.name + " - experimental driver based on OpenHantek."
	s.info = info
	for c := 0; c < 2; c++ {
		for g := 0; g < 9; g++ {
			s.limits[c][g] = [2]float64{0, 0xFFFF}
		}
	}
	return s
}

func (s *hantekDSO) Info() *DeviceInfo { return s.info }

func (s *hantekDSO) find(loaded bool) *usbInfo {
	return findUSB(func(u usbInfo) bool {
		if !u.WinUSB {
			return false
		}
		if loaded {
			return s.m.isLoaded(u)
		}
		return s.m.isLoader(u)
	})
}

func (s *hantekDSO) Open(progress func(string)) error {
	u := s.find(true)
	if u == nil {
		u = s.find(false)
		if u == nil {
			return devErr("The %s is no longer connected (or doesn't have the WinUSB driver).", s.m.name)
		}
		blob, err := hdFirmware(s.m)
		if err != nil {
			return devErr("The %s needs Hantek's firmware, which cannot be shipped with this program.\n\n"+
				"In the Connect window click 'Import firmware…' and choose the driver file from Hantek's "+
				"software (for example DSO2090X861.SYS, or the driver .zip downloaded from hantek.com).", s.m.name)
		}
		if progress != nil {
			progress("Loading firmware into the scope…")
		}
		d, err := openUSB(u)
		if err != nil {
			return err
		}
		err = fx2Upload(d, []hexRecord{{0, blob}})
		d.Close()
		if err != nil {
			return err
		}
		time.Sleep(1500 * time.Millisecond)
		u = waitFor(15, 250*time.Millisecond, func() *usbInfo { return s.find(true) })
		if u == nil {
			return devErr("The firmware was loaded but the scope did not come back. The scope now has "+
				"USB ID 04B5:%04X, which also needs the WinUSB driver.", s.m.pid)
		}
	}
	d, err := openUSB(u)
	if err != nil {
		return err
	}
	s.d = d
	s.readLimits()
	return nil
}

func (s *hantekDSO) Close() {
	if s.d != nil {
		s.d.Close()
		s.d = nil
	}
}

func (s *hantekDSO) readLimits() {
	raw := make([]byte, 72)
	n, err := s.d.Control(usbVendorIn, hdCtrlValue, 0x08, 0, raw)
	if err != nil || n < 72 {
		return
	}
	for ch := 0; ch < 2; ch++ {
		for g := 0; g < 9; g++ {
			o := (ch*9 + g) * 4
			lo := float64(binary.BigEndian.Uint16(raw[o:]))
			hi := float64(binary.BigEndian.Uint16(raw[o+2:]))
			if hi > lo {
				s.limits[ch][g] = [2]float64{lo, hi}
			}
		}
	}
}

func (s *hantekDSO) bulk(data []byte) error {
	if _, err := s.d.Control(usbVendorOut, hdCtrlBegin, 0, 0, append([]byte(nil), hdBegin...)); err != nil {
		return devErr("Lost contact with the scope (%v).", err)
	}
	if _, err := s.d.BulkWrite(hdEPOut, data, 1000); err != nil {
		return devErr("Lost contact with the scope (%v).", err)
	}
	return nil
}

func (s *hantekDSO) ctrl(req uint8, data []byte) error {
	if _, err := s.d.Control(usbVendorOut, req, 0, 0, data); err != nil {
		return devErr("Lost contact with the scope (%v).", err)
	}
	return nil
}

func (s *hantekDSO) downsampler(want float64) (int, float64) {
	m := s.m
	d := m.base / want
	if d < 1 {
		return 0, m.base
	}
	var di int
	if m.kind == "2090" {
		switch {
		case d < 1.5:
			di = 1
		case d < 3.5:
			di = 2
		case d <= 5:
			di = 5
		default:
			di = maxInt(6, int(math.Round(d/2))*2)
		}
	} else {
		di = maxInt(1, int(math.Round(d)))
	}
	di = minInt(di, m.maxDown)
	return di, m.base / float64(di)
}

func (s *hantekDSO) ApplyPanel(p *Panel) error {
	cp := *p
	s.panel = &cp
	m := s.m
	for c := 0; c < 2; c++ {
		s.gid[c] = nearestIndex(s.vdivs, p.Ch[c].VDiv)
	}
	// vertical position -> hardware offset (0..1 of the ADC window, 0.5 = centre)
	var offs [2]int
	for c := 0; c < 2; c++ {
		frac := clampF(0.5+p.Ch[c].Pos/8, 0, 1)
		lim := s.limits[c][s.gid[c]]
		val := int(frac*(lim[1]-lim[0]) + lim[0] + 0.5)
		s.offReal[c] = (float64(val) - lim[0]) / (lim[1] - lim[0])
		offs[c] = val
	}
	src := minInt(p.TrigSource, 1)
	gstep := m.gain[s.gid[src]].full
	tlo, thi := 0.0, float64(0xFD)
	if m.bits > 8 {
		tlo, thi = s.limits[src][s.gid[src]][0], s.limits[src][s.gid[src]][1]
	}
	tval := int((s.offReal[src]+p.TrigLevel/gstep)*(thi-tlo) + 0.5 + tlo)
	tval = clampInt(tval, int(tlo), int(thi))
	off := []byte{byte(offs[0] >> 8), byte(offs[0]), byte(offs[1] >> 8), byte(offs[1]), byte(tval >> 8), byte(tval)}
	if err := s.ctrl(hdCtrlOffset, append(off, make([]byte, 11)...)); err != nil {
		return err
	}
	pick := func(cond bool, a, b byte) byte {
		if cond {
			return a
		}
		return b
	}
	relays := []byte{0x00,
		pick(s.gid[0] < 3, 0xFB, 0x04), pick(s.gid[0] < 6, 0xF7, 0x08), pick(p.Ch[0].Coupling == "DC", 0xFD, 0x02),
		pick(s.gid[1] < 3, 0xDF, 0x20), pick(s.gid[1] < 6, 0xBF, 0x40), pick(p.Ch[1].Coupling == "DC", 0xEF, 0x10),
		0x01}
	if err := s.ctrl(hdCtrlRelays, append(relays, make([]byte, 9)...)); err != nil {
		return err
	}
	gbits := byte(m.gain[s.gid[0]].id&3) | byte(m.gain[s.gid[1]].id&3)<<2
	if err := s.bulk([]byte{0x07, 0x00, gbits, 0, 0, 0, 0, 0}); err != nil {
		return err
	}
	// timebase: hdRecord samples should cover the screen
	hd := float64(s.info.HDivs)
	down, rate := s.downsampler(hdRecord / (p.TDiv * hd))
	s.rate = rate
	s.pre = int(hdRecord * (p.TrigX + hd/2) / hd)
	slope := 0
	if p.TrigEdge == "falling" {
		slope = 1
	}
	var err error
	switch m.kind {
	case "2090":
		var sid, dval, dmode int
		switch {
		case down == 0:
			sid, dval, dmode = 1, 0, 0
		case down <= 2:
			sid, dval, dmode = down, 0, 0
		case down == 5:
			sid, dval, dmode = 3, 0xFFFF, 0
		default:
			sid, dval, dmode = 0, (0x10001-(down>>1))&0xFFFF, 1
		}
		b2 := byte(((1 - src) & 3) | (1 << 2) | ((sid & 3) << 5) | (dmode << 7))
		b3 := byte(2 | (slope << 3)) // both channels, normal rate
		tpos := 0x7FFFF - hdRecord + s.pre
		err = s.bulk([]byte{0x01, 0, b2, b3, byte(dval), byte(dval >> 8), byte(tpos), byte(tpos >> 8), 0, 0, byte(tpos >> 16), 0})
	case "2250":
		dmode, dval := 0, 0
		if down > 1 {
			dmode, dval = 1, (0x10001-down)&0xFFFF
		}
		pre := 0x7FFFF - hdRecord + s.pre
		post := 0x7FFFF - s.pre
		for _, cmd := range [][]byte{
			{0x0D, 0, 1, 0}, // record length id 1
			{0x0B, 0, 2, 0}, // both channels
			{0x0C, 0, byte((2 + src) | (slope << 2)), 0, 0, 0, 0, 0},
			{0x0E, 0, byte(dmode << 1), 0, byte(dval), byte(dval >> 8), 0, 0},
			{0x0F, 0, byte(post), byte(post >> 8), byte(post >> 16), 0, byte(pre), byte(pre >> 8), byte(pre >> 16), 0, 0, 0},
		} {
			if err = s.bulk(cmd); err != nil {
				break
			}
		}
	default: // 5200 / 5200A
		d := maxInt(down, 1)
		slow := maxInt((d-3)/2, 0)
		fast := d - slow*2
		sval := 0
		if slow != 0 {
			sval = 0xFFFF - slow
		}
		pre := (0xFFFF - hdRecord + s.pre) & 0xFFFF
		post := (0xFFFF - s.pre) & 0xFFFF
		etsr := 1 | (2 << 1) | (((1 - src) & 3) << 3) | ((slope & 3) << 5)
		for _, cmd := range [][]byte{
			{0x0C, 0, byte(sval), byte(sval >> 8), byte(4 - fast), 0},
			{0x0D, 0, byte(pre), byte(pre >> 8), 0x07, 0xFF, byte(post), byte(post >> 8), 0x07 | (1 << 3), 0xFF},
			{0x0E, 0, byte(etsr), 0, 0x02, 0, 0, 0},
		} {
			if err = s.bulk(cmd); err != nil {
				break
			}
		}
	}
	s.state = "idle"
	return err
}

func (s *hantekDSO) Start(single bool) error {
	s.running, s.single, s.state = true, single, "idle"
	return nil
}

func (s *hantekDSO) Stop() error {
	s.running, s.state = false, "idle"
	return nil
}

func (s *hantekDSO) captureState() (byte, int, error) {
	if err := s.bulk([]byte{0x06, 0}); err != nil {
		return 0, 0, err
	}
	r := make([]byte, 512)
	n, err := s.d.BulkRead(hdEPIn, r, 500)
	if err != nil {
		return 0, 0, devErr("Lost contact with the scope (%v).", err)
	}
	if n < 4 {
		return 0, 0, nil
	}
	tp := int(r[2]) | int(r[3])<<8 | int(r[1])<<16
	bit := 1
	for i := 0; i < 24; i++ { // each set bit inverts all lower bits (OpenHantek)
		if tp&bit != 0 {
			tp ^= bit - 1
		}
		bit <<= 1
	}
	return r[0], tp, nil
}

func (s *hantekDSO) ReadFrame(timeout float64) (*FrameData, error) {
	if !s.running || s.panel == nil {
		sleepS(timeout)
		return nil, nil
	}
	now := time.Now()
	if s.state == "idle" {
		if err := s.bulk([]byte{0x03, 0}); err != nil { // start sampling
			return nil, err
		}
		if err := s.bulk([]byte{0x04, 0}); err != nil { // enable trigger
			return nil, err
		}
		s.state, s.tArm, s.forced = "armed", now, false
		time.Sleep(5 * time.Millisecond)
		return nil, nil
	}
	state, tpoint, err := s.captureState()
	if err != nil {
		return nil, err
	}
	if state != s.m.ready {
		frameTime := hdRecord / s.rate
		if s.panel.TrigMode == "auto" && !s.single && !s.forced && now.Sub(s.tArm).Seconds() > math.Max(0.1, 3*frameTime) {
			if err := s.bulk([]byte{0x02, 0}); err != nil { // force a trigger
				return nil, err
			}
			s.forced = true
		}
		sleepS(math.Min(timeout, 0.01))
		return nil, nil
	}
	wide := s.m.bits > 8
	nbytes := hdRecord * 2
	if wide {
		nbytes *= 2
	}
	if err := s.bulk([]byte{0x05, 0}); err != nil {
		return nil, err
	}
	raw := make([]byte, 0, nbytes)
	buf := make([]byte, 512*32)
	for len(raw) < nbytes {
		k, err := s.d.BulkRead(hdEPIn, buf[:minInt(len(buf), nbytes-len(raw))], 1000)
		if err != nil {
			return nil, devErr("Lost contact while reading data (%v).", err)
		}
		if k == 0 {
			break
		}
		raw = append(raw, buf[:k]...)
	}
	s.state = "idle"
	total := hdRecord * 2
	if len(raw) < total {
		return nil, nil
	}
	start := (tpoint * 2) % total
	fd := newFrameData()
	for c := 0; c < 2; c++ {
		gid := s.gid[c]
		limit := s.m.limit(gid)
		gstep := s.m.gain[gid].full
		codes := make([]float32, hdRecord)
		for i := range codes {
			idx := (start + i*2) % total
			low := int(raw[(idx+1-c)%total])
			if wide && len(raw) >= 2*total {
				high := (int(raw[total+idx]) << (8 - c*2)) & 0x0300
				codes[i] = float32(low + high)
			} else {
				codes[i] = float32(low)
			}
		}
		if wide && len(raw) >= 2*total {
			fd.Ranges[c] = CodeRange{0, 1023}
		} else {
			fd.Ranges[c] = CodeRange{0, 255}
		}
		fd.Codes[c] = codes
		fd.Scales[c] = Scale{gstep / limit, s.offReal[c] * limit}
	}
	if s.single {
		s.running = false
	}
	dt := 1 / s.rate
	fd.DT, fd.TFirst, fd.Triggered = dt, -float64(s.pre)*dt, !s.forced
	return fd, nil
}
