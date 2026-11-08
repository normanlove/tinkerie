// Hantek 6022 family: Cypress-FX2 based USB scopes that stream raw samples.
//
// Supported: Hantek DSO-6022BE, DSO-6022BL, DSO-6021, Voltcraft DSO-2020,
// Sainsmart/BUUDAI DDS120, Instrustar ISDS-205B and YiXingDianZi MDSO.
//
// Each needs open-source firmware uploaded after every power-up (it lives in
// RAM). The 6022BE/BL, 6021 and DDS120 firmware is built into Scopie (from
// Hantek6022API, LGPL-2.1+; source in third_party/source). The ISDS-205B and
// MDSO firmware has no published source, so it is not bundled: users import
// isds205b-firmware.hex / mdso-firmware.hex from an OpenHantek6022 download.
// Protocol reference: OpenHantek6022 (GPL-3.0) and Hantek6022API.
package scopie

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

const (
	reqGainCh1 = 0xE0
	reqGainCh2 = 0xE1
	reqRate    = 0xE2
	reqStart   = 0xE3
	reqNumCh   = 0xE4
	reqACDC    = 0xE5
	reqCalFreq = 0xE6
	reqEEPROM  = 0xA2
	fx2EPIn    = 0x86
)

var rates6022 = []RateID{{102, 20e3}, {104, 40e3}, {105, 50e3}, {110, 100e3}, {120, 200e3}, {140, 400e3},
	{150, 500e3}, {1, 1e6}, {2, 2e6}, {4, 4e6}, {5, 5e6}, {8, 8e6}, {10, 10e6},
	{12, 12e6}, {15, 15e6}, {24, 24e6}, {30, 30e6}}
var ratesDDS = []RateID{{1, 1e6}, {2, 2e6}, {8, 8e6}, {11, 10e6}, {15, 15e6}, {24, 24e6}, {30, 30e6}}

var cal6022 = []float64{32, 50, 100, 200, 500, 1000, 2000, 5000, 10000, 20000, 50000, 100000}
var calDDS = []float64{50, 100, 200, 500, 1000, 2000, 5000, 10000, 20000}

type fx2Model struct {
	name          string
	loaderV       uint16
	loaderP       uint16
	fwV, fwP      uint16
	fwBCD         int // -1: any
	firmware      string
	countsPerVolt map[int]float64
	rates         []RateID
	max1, max2    float64
	eeprom        bool
	hwAC, acdcMod bool
	cal           []float64
}

var cpv6022 = map[int]float64{1: 25, 2: 50, 5: 125, 10: 250}

var fx2Models = []fx2Model{
	{"Hantek DSO-6022BE", 0x04B4, 0x6022, 0x04B5, 0x6022, 0x0210, "dso6022be-firmware.hex", cpv6022, rates6022, 30e6, 15e6, true, false, true, cal6022},
	{"Voltcraft DSO-2020", 0x04B4, 0x2020, 0x04B5, 0x6022, 0x0210, "dso6022be-firmware.hex", cpv6022, rates6022, 30e6, 15e6, true, false, true, cal6022},
	{"Hantek DSO-6022BL", 0x04B4, 0x602A, 0x04B5, 0x602A, 0x0210, "dso6022bl-firmware.hex", cpv6022, rates6022, 30e6, 15e6, true, false, true, cal6022},
	{"Hantek DSO-6021", 0x04B4, 0x6021, 0x04B5, 0x6021, 0x0210, "dso6021-firmware.hex", cpv6022, rates6022, 30e6, 15e6, true, true, false, cal6022},
	{"Sainsmart / BUUDAI DDS120", 0x8102, 0x8102, 0x04B5, 0x0120, -1, "dds120-firmware.hex",
		map[int]float64{1: 21, 2: 43, 5: 96, 10: 200, 20: 400}, ratesDDS, 30e6, 15e6, false, true, false, calDDS},
	{"Instrustar ISDS-205B", 0xD4A2, 0x5661, 0x1D50, 0x608E, 0x0005, "isds205b-firmware.hex",
		map[int]float64{1: 21.78, 2: 37.48, 5: 85.36, 10: 1330}, ratesDDS, 30e6, 15e6, false, true, false, calDDS},
	{"YiXingDianZi MDSO", 0xD4A2, 0x5660, 0x1D50, 0x608E, 0x0001, "mdso-firmware.hex", cpv6022, ratesDDS, 30e6, 15e6, false, false, false, calDDS},
}

func (m *fx2Model) isLoader(u usbInfo) bool { return u.VID == m.loaderV && u.PID == m.loaderP }
func (m *fx2Model) isLoaded(u usbInfo, strict bool) bool {
	if u.VID != m.fwV || u.PID != m.fwP {
		return false
	}
	if m.fwBCD < 0 || int(u.BCD) == m.fwBCD {
		return true
	}
	return !strict && m.fwV == 0x04B5
}

// pick prefers the entry that has the WinUSB driver attached.
func preferWinUSB(us []usbInfo) []usbInfo {
	sort.SliceStable(us, func(i, j int) bool { return us[i].WinUSB && !us[j].WinUSB })
	return us
}

func scanFX2(lan []string, notes *[]string) []*Candidate {
	var out []*Candidate
	seen := map[string]bool{}
	for _, u := range preferWinUSB(append([]usbInfo(nil), listUSB()...)) {
		for i := range fx2Models {
			m := &fx2Models[i]
			fresh := m.isLoader(u)
			if !fresh && !m.isLoaded(u, false) {
				continue
			}
			key := fmt.Sprintf("%04x:%04x:%s", u.VID, u.PID, u.Serial)
			if u.Serial == "" {
				key = u.Key
			}
			if seen[key] {
				break
			}
			seen[key] = true
			c := &Candidate{Label: m.name, Family: "Hantek 6022 family", Ready: true}
			haveFW := !fresh || haveFirmware(m.firmware)
			switch {
			case !u.WinUSB:
				c.Ready = false
				c.Detail = usbDriverHint(u)
			case !haveFW:
				c.Ready = false
				c.Detail = fmt.Sprintf("Needs firmware: click 'Import firmware…' and choose %s (included in the OpenHantek6022 download, folder 'firmware').", m.firmware)
			case fresh:
				c.Detail = "USB, firmware will be loaded"
			default:
				c.Detail = "USB, ready"
			}
			mm := m
			c.Factory = func() (Device, error) { return newFX2Scope(mm), nil }
			out = append(out, c)
			break
		}
	}
	return out
}

type fx2Scope struct {
	m      *fx2Model
	info   *DeviceInfo
	d      usbDevice
	calOff [2]map[int]float64
	calMul [2]map[int]float64
	stop   chan struct{}
	wg     sync.WaitGroup
	mu     sync.Mutex
}

func newFX2Scope(m *fx2Model) *fx2Scope {
	info := defaultInfo(m.name)
	for _, r := range m.rates {
		info.SampleRates = append(info.SampleRates, r.Rate)
	}
	info.MaxRate1, info.MaxRateMulti = m.max1, m.max2
	info.HWAC, info.ACDCModOption = m.hwAC, m.acdcMod
	info.CalOutput, info.CalFreqs = true, m.cal
	info.VDivs = []float64{0.01, 0.02, 0.05, 0.1, 0.2, 0.5, 1, 2, 5}
	info.Notes = fmt.Sprintf("%s: max %g MS/s with two channels, %g MS/s with one.", m.name, m.max2/1e6, m.max1/1e6)
	s := &fx2Scope{m: m, info: info}
	for c := 0; c < 2; c++ {
		s.calOff[c] = map[int]float64{}
		s.calMul[c] = map[int]float64{}
		for g := range m.countsPerVolt {
			s.calOff[c][g] = 0
			s.calMul[c][g] = 1
		}
	}
	return s
}

func (s *fx2Scope) Info() *DeviceInfo { return s.info }

func (s *fx2Scope) find(loaded bool) *usbInfo {
	return findUSB(func(u usbInfo) bool {
		if !u.WinUSB {
			return false
		}
		if loaded {
			return s.m.isLoaded(u, true)
		}
		return s.m.isLoader(u)
	})
}

func (s *fx2Scope) Open(progress func(string)) error {
	m := s.m
	u := s.find(true)
	if u == nil {
		u = s.find(false)
		if u == nil { // loaded, but with older firmware: load ours over it
			u = findUSB(func(x usbInfo) bool { return x.WinUSB && x.VID == m.fwV && x.PID == m.fwP })
		}
		if u == nil {
			if findUSB(func(x usbInfo) bool { return m.isLoader(x) || m.isLoaded(x, false) }) != nil {
				return devErr("The %s is plugged in but doesn't have the WinUSB driver yet.\n\n%s", m.name, accessHelp)
			}
			return devErr("The %s is no longer connected.", m.name)
		}
		if progress != nil {
			progress("Loading firmware into the scope…")
		}
		fw, err := firmwareBytes(m.firmware)
		if err != nil {
			return err
		}
		recs, err := parseIHex(fw)
		if err != nil {
			return devErr("Firmware %s is damaged: %v", m.firmware, err)
		}
		d, err := openUSB(u)
		if err != nil {
			return err
		}
		err = fx2Upload(d, recs)
		d.Close()
		if err != nil {
			return err
		}
		time.Sleep(time.Second) // the scope disconnects and comes back as a new device
		u = waitFor(15, 250*time.Millisecond, func() *usbInfo { return s.find(true) })
		if u == nil {
			return devErr("The firmware was loaded but the scope did not come back.\n\n"+
				"After loading firmware the scope has a new USB ID (%04X:%04X), which also needs the "+
				"WinUSB driver. Click 'Install USB driver…' in the Connect window, then unplug and re-plug the scope.", m.fwV, m.fwP)
		}
	}
	d, err := openUSB(u)
	if err != nil {
		return err
	}
	s.d = d
	if progress != nil {
		progress("Reading calibration…")
	}
	if m.eeprom {
		s.readCalibration()
	}
	if _, _, serial := usbStrings(d); serial != "" {
		s.info.Name = fmt.Sprintf("%s (%s)", m.name, serial)
	}
	return nil
}

func (s *fx2Scope) readCalibration() {
	cal := make([]byte, 80)
	n, err := s.d.Control(usbVendorIn, reqEEPROM, 0x08, 0, cal)
	if err != nil || n < 64 {
		return
	}
	idx := map[int][2]int{10: {0, 1}, 5: {6, 7}, 2: {8, 9}, 1: {14, 15}}
	for g, ii := range idx {
		s.calOff[0][g] = float64(cal[ii[0]]) - 128
		s.calOff[1][g] = float64(cal[ii[1]]) - 128
	}
	// optional fine offsets and gain corrections written by calibration tools
	fine := map[int][2]int{10: {48, 49}, 5: {54, 55}, 2: {56, 57}, 1: {62, 63}}
	gain := map[int][2]int{10: {32, 33}, 5: {38, 39}, 2: {40, 41}, 1: {46, 47}}
	for _, g := range []int{1, 2, 5, 10} {
		for ch := 0; ch < 2; ch++ {
			if b := cal[fine[g][ch]]; b != 0 && b != 255 {
				s.calOff[ch][g] += (float64(b) - 128) / 250
			}
			if b := cal[gain[g][ch]]; b != 0 && b != 255 {
				s.calMul[ch][g] *= 1 + (float64(b)-128)/500
			}
		}
	}
}

func (s *fx2Scope) Close() {
	s.StopStream()
	if s.d != nil {
		s.d.Close()
		s.d = nil
	}
}

func (s *fx2Scope) Gains() []Gain {
	var out []Gain
	for g, cpv := range s.m.countsPerVolt {
		out = append(out, Gain{g, 128 / cpv})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (s *fx2Scope) Scale(ch, gain int) Scale {
	cpv := s.m.countsPerVolt[gain]
	mul := s.calMul[ch][gain]
	if mul == 0 {
		mul = 1
	}
	return Scale{mul / cpv, 128 + s.calOff[ch][gain]}
}

func (s *fx2Scope) RateIDs(nch int) []RateID {
	mx := s.m.max2
	if nch == 1 {
		mx = s.m.max1
	}
	var out []RateID
	for _, r := range s.m.rates {
		if r.Rate <= mx {
			out = append(out, r)
		}
	}
	return out
}

func (s *fx2Scope) ctl(req uint8, value int) error {
	_, err := s.d.Control(usbVendorOut, req, 0, 0, []byte{byte(value)})
	return err
}

func (s *fx2Scope) Configure(rid, nch int, gains [2]int, dc [2]bool, mod bool, calFreq float64) (int, error) {
	if s.d == nil {
		return 0, devErr("The scope is not connected.")
	}
	acdc := 0x11
	if s.m.hwAC || mod {
		acdc = 0
		if dc[0] {
			acdc |= 0x01
		}
		if dc[1] {
			acdc |= 0x10
		}
	}
	for _, kv := range [][2]int{{reqNumCh, nch}, {reqRate, rid}, {reqGainCh1, gains[0]}, {reqGainCh2, gains[1]},
		{reqACDC, acdc}, {reqCalFreq, calByte(calFreq)}} {
		if err := s.ctl(uint8(kv[0]), kv[1]); err != nil {
			return 0, devErr("The scope stopped responding (%v).", err)
		}
	}
	return nch, nil
}

func (s *fx2Scope) StartStream(chunk int, fn func([]byte), fail func(error)) error {
	s.StopStream()
	if err := s.ctl(reqStart, 1); err != nil {
		return devErr("The scope stopped responding (%v).", err)
	}
	stop := make(chan struct{})
	s.mu.Lock()
	s.stop = stop
	s.mu.Unlock()
	d := s.d
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		err := d.Stream(fx2EPIn, chunk, 8, stop, func(b []byte) bool { fn(b); return true })
		if err != nil {
			select {
			case <-stop:
			default:
				fail(devErr("USB error: %v. Was the scope unplugged?", err))
			}
		}
	}()
	return nil
}

func (s *fx2Scope) StopStream() {
	s.mu.Lock()
	stop := s.stop
	s.stop = nil
	s.mu.Unlock()
	if stop == nil {
		return
	}
	if s.d != nil {
		_ = s.ctl(reqStart, 0)
	}
	close(stop)
	s.wg.Wait()
}

// calByte encodes the calibration-output frequency the way the firmware expects.
func calByte(f float64) int {
	f = clampF(f, 32, 100000)
	if f < 1000 {
		return int((f+5)/10) + 100
	}
	if f < 5750 {
		return minInt(int((f+50)/100)+200, 255)
	}
	return int((f + 500) / 1000)
}
