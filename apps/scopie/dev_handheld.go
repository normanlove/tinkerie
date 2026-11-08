// Hantek handheld 3-in-1 scopes: 2C42, 2C72, 2D42, 2D72 (USB 0483:2D42).
//
// Protocol reference: the open-source 'Hantek' tool by lucaoli (GPL-3.0) and
// its Rust port 'hanteker' by hkoosha (GPL-3.0). Every command is a 10-byte
// packet: [0x00, 0x0A, function lo, function hi, command, value x4, 0x00].
//
// This driver is EXPERIMENTAL: the time scale of the captured samples was
// inferred from the reference tools and has not been verified on all models.
package scopie

import (
	"time"
)

const (
	hhVID, hhPID    = 0x0483, 0x2D42
	hhEPOut, hhEPIn = 0x02, 0x81

	hhFuncScope, hhFuncCapture, hhFuncScreen = 0x0000, 0x0100, 0x0003
	hhStartStop, hhTimeScale                 = 0x0C, 0x0E
	hhTrigSource, hhTrigSlope, hhTrigMode    = 0x10, 0x11, 0x12
	hhTrigLevel, hhStartRecv                 = 0x14, 0x16

	hhCentre, hhPerDiv = 130.0, 25.25 // screen spans codes 29..231 over 8 divisions
	hhSamplesPerDiv    = 100
	hhPoints           = 1000
)

var (
	hhEnable   = [2]byte{0x00, 0x06}
	hhCoupling = [2]byte{0x01, 0x07}
	hhProbe    = [2]byte{0x02, 0x08}
	hhScale    = [2]byte{0x04, 0x0A}
	hhOffset   = [2]byte{0x05, 0x0B}
	hhVDivs    = []float64{0.01, 0.02, 0.05, 0.1, 0.2, 0.5, 1, 2, 5, 10}
	hhTDivs    = []float64{5e-9, 10e-9, 20e-9, 50e-9, 100e-9, 200e-9, 500e-9, 1e-6, 2e-6, 5e-6, 10e-6, 20e-6,
		50e-6, 100e-6, 200e-6, 500e-6, 1e-3, 2e-3, 5e-3, 10e-3, 20e-3, 50e-3, 100e-3, 200e-3,
		500e-3, 1, 2, 5, 10, 20, 50, 100, 200, 500}
)

func scanHandheld(lan []string, notes *[]string) []*Candidate {
	var out []*Candidate
	for _, u := range listUSB() {
		if u.VID != hhVID || u.PID != hhPID {
			continue
		}
		u := u
		c := &Candidate{Label: "Hantek 2C42 / 2C72 / 2D42 / 2D72", Family: "Hantek handheld",
			Detail: "USB (experimental)", Ready: u.WinUSB,
			Factory: func() (Device, error) { return newHandheld(u), nil }}
		if !u.WinUSB {
			c.Detail = usbDriverHint(u)
		}
		out = append(out, c)
	}
	return out
}

type handheld struct {
	u       usbInfo
	d       usbDevice
	info    *DeviceInfo
	panel   *Panel
	running bool
	single  bool
}

func newHandheld(u usbInfo) *handheld {
	info := frameInfo("Hantek handheld (2C42/2C72/2D42/2D72)", 2)
	info.VDivs, info.TDivs = hhVDivs, hhTDivs
	info.Notes = "Experimental driver. The handheld sends about 5-20 waveforms per second."
	return &handheld{u: u, info: info}
}

func (h *handheld) Info() *DeviceInfo { return h.info }

func (h *handheld) Open(progress func(string)) error {
	d, err := openUSB(&h.u)
	if err != nil {
		return err
	}
	h.d = d
	if _, product, _ := usbStrings(d); product != "" {
		h.info.Name = "Hantek " + product
	}
	return h.cmd(hhFuncScreen, 0, 0) // switch the handheld to scope mode
}

func (h *handheld) Close() {
	if h.d != nil {
		h.d.Close()
		h.d = nil
	}
}

func (h *handheld) cmd(fn int, cmd byte, v ...int) error {
	pkt := []byte{0x00, 0x0A, byte(fn), byte(fn >> 8), cmd, 0, 0, 0, 0, 0x00}
	for i := 0; i < len(v) && i < 4; i++ {
		pkt[5+i] = byte(v[i])
	}
	if _, err := h.d.BulkWrite(hhEPOut, pkt, 1000); err != nil {
		return devErr("Lost contact with the handheld (%v).", err)
	}
	return nil
}

func (h *handheld) ApplyPanel(p *Panel) error {
	cp := *p
	h.panel = &cp
	src := minInt(p.TrigSource, 1)
	for c := 0; c < 2; c++ {
		ch := p.Ch[c]
		steps := []struct {
			c byte
			v int
		}{
			{hhEnable[c], boolInt(ch.Enabled || c == src)},
			{hhCoupling[c], boolInt(ch.Coupling == "DC")},
			{hhProbe[c], 0},
			{hhScale[c], nearestIndex(hhVDivs, ch.VDiv)},
			{hhOffset[c], clampInt(int(100+25*ch.Pos+0.5), 0, 200)},
		}
		for _, s := range steps {
			if err := h.cmd(hhFuncScope, s.c, s.v); err != nil {
				return err
			}
		}
	}
	slope := map[string]int{"rising": 0, "falling": 1, "both": 2}[p.TrigEdge]
	mode := 1
	if p.TrigMode == "auto" {
		mode = 0
	}
	sp := p.Ch[src]
	lvl := clampInt(int(100+25*(p.TrigLevel/sp.VDiv+sp.Pos)+0.5), 0, 200)
	for _, s := range [][2]int{{hhTimeScale, nearestIndex(hhTDivs, p.TDiv)}, {hhTrigSource, src},
		{hhTrigSlope, slope}, {hhTrigMode, mode}, {hhTrigLevel, lvl}} {
		if err := h.cmd(hhFuncScope, byte(s[0]), s[1]); err != nil {
			return err
		}
	}
	return nil
}

func (h *handheld) Start(single bool) error {
	h.running, h.single = true, single
	if single {
		if err := h.cmd(hhFuncScope, hhTrigMode, 2); err != nil {
			return err
		}
	}
	return h.cmd(hhFuncScope, hhStartStop, 1)
}

func (h *handheld) Stop() error {
	h.running = false
	if h.d == nil {
		return nil
	}
	return h.cmd(hhFuncScope, hhStartStop, 0)
}

func (h *handheld) ReadFrame(timeout float64) (*FrameData, error) {
	if !h.running || h.panel == nil {
		sleepS(timeout)
		return nil, nil
	}
	p := h.panel
	var chans []int
	for c := 0; c < 2; c++ {
		if p.Ch[c].Enabled || c == p.TrigSource {
			chans = append(chans, c)
		}
	}
	nch := len(chans)
	total := hhPoints * nch
	half := total / 2
	req := []byte{0x00, 0x0A, byte(hhFuncCapture & 0xFF), byte(hhFuncCapture >> 8), hhStartRecv,
		byte(half), byte(half >> 8), byte(half), byte(half >> 8), 0x00}
	buf := make([]byte, 0, total)
	chunk := make([]byte, 64)
	deadline := time.Now().Add(3 * time.Second)
	for len(buf) < total {
		if time.Now().After(deadline) {
			return nil, devErr("The handheld stopped sending data.")
		}
		if _, err := h.d.BulkWrite(hhEPOut, req, 1000); err != nil {
			return nil, devErr("Lost contact with the handheld (%v).", err)
		}
		n, err := h.d.BulkRead(hhEPIn, chunk[:minInt(64, total-len(buf))], 1000)
		if err != nil {
			return nil, devErr("Lost contact with the handheld (%v).", err)
		}
		buf = append(buf, chunk[:n]...)
	}
	fd := newFrameData()
	for i, c := range chans {
		codes := make([]float32, hhPoints)
		for k := range codes {
			codes[k] = float32(buf[k*nch+i])
		}
		vd := hhVDivs[nearestIndex(hhVDivs, p.Ch[c].VDiv)]
		fd.Codes[c] = codes
		fd.Scales[c] = Scale{vd / hhPerDiv, hhCentre + p.Ch[c].Pos*hhPerDiv}
	}
	dt := hhTDivs[nearestIndex(hhTDivs, p.TDiv)] / hhSamplesPerDiv
	if h.single {
		h.running = false
	}
	fd.DT, fd.TFirst = dt, -hhPoints/2*dt
	fd.Triggered = p.TrigMode == "normal" || h.single
	return fd, nil
}
