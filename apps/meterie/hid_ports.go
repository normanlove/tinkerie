//go:build windows || linux || darwin

package meterie

// hid_ports.go: the USB-HID meter cables, the same on every system: UNI-T
// UT-D04 (CH9325 / HE2325U), Silicon Labs CP2110, Victor and Brymen BU-86X.
// Each system supplies hidHandle (Windows: hid.dll, Linux: /dev/hidraw,
// macOS: IOKit) with Windows' report layout: an input report starts with its
// report ID (0 when the device doesn't number its reports), and writes and
// feature reports start with the report ID too.

import "fmt"

type hidHandle interface {
	readReport(timeoutMs uint32) ([]byte, error) // nil on timeout
	writeReport(b []byte) error
	setFeature(b []byte) error
	defaultLens(out, feat int) // report lengths to pad to, if the system doesn't know
	forceFeatLen(n int)
	close()
}

// UNI-T UT-D04 (CH9325 / HE2325U): a feature report sets the baud rate; input
// reports carry up to 7 UART bytes: [reportID, 0xF0|count, data...].
type ch9325Port struct {
	d     hidHandle
	mask7 bool
}

func platformOpenCH9325(path string, cfg SerialCfg) (Port, error) {
	d, err := openHIDDev(path)
	if err != nil {
		return nil, err
	}
	b := cfg.Baud
	fr := []byte{0x00, byte(b), byte(b >> 8), byte(b >> 16), byte(b >> 24), 0x03}
	if err := d.setFeature(fr); err != nil {
		d.forceFeatLen(6)
		if err2 := d.setFeature(fr); err2 != nil {
			d.close()
			return nil, fmt.Errorf("can't configure the UT-D04 cable: %v", err)
		}
	}
	return &ch9325Port{d: d, mask7: cfg.DataBits == 7}, nil
}

func (u *ch9325Port) Read(p []byte) (int, error) {
	rep, err := u.d.readReport(150)
	if err != nil || rep == nil || len(rep) < 2 {
		return 0, err
	}
	cnt := int(rep[1] & 0x07)
	if cnt > len(rep)-2 {
		cnt = len(rep) - 2
	}
	n := copy(p, rep[2:2+cnt])
	if u.mask7 {
		for i := 0; i < n; i++ {
			p[i] &= 0x7F
		}
	}
	return n, nil
}
func (u *ch9325Port) Write(p []byte) (int, error) { return len(p), nil } // receive-only cable
func (u *ch9325Port) Close() error                { u.d.close(); return nil }

// Silicon Labs CP2110 HID-to-UART (AN434). Data reports: report ID = byte count.
type cp2110Port struct{ d hidHandle }

func platformOpenCP2110(path string, cfg SerialCfg) (Port, error) {
	d, err := openHIDDev(path)
	if err != nil {
		return nil, err
	}
	d.defaultLens(64, 64)
	if err := d.setFeature([]byte{0x41, 0x01}); err != nil {
		d.close()
		return nil, fmt.Errorf("can't enable the CP2110 UART: %v", err)
	}
	b := cfg.Baud
	if b == 0 {
		b = 9600
	}
	parity := byte(0)
	switch cfg.Parity {
	case 'O':
		parity = 1
	case 'E':
		parity = 2
	}
	bits := byte(3)
	if cfg.DataBits == 7 {
		bits = 2
	}
	stop := byte(0)
	if cfg.StopBits == 2 {
		stop = 1
	}
	if err := d.setFeature([]byte{0x50, byte(b >> 24), byte(b >> 16), byte(b >> 8), byte(b), parity, 0, bits, stop}); err != nil {
		d.close()
		return nil, fmt.Errorf("can't configure the CP2110 UART: %v", err)
	}
	d.setFeature([]byte{0x43, 0x03}) // purge FIFOs
	return &cp2110Port{d: d}, nil
}

func (c *cp2110Port) Read(p []byte) (int, error) {
	rep, err := c.d.readReport(150)
	if err != nil || rep == nil || len(rep) < 2 {
		return 0, err
	}
	n := int(rep[0])
	if n < 1 || n > 63 {
		return 0, nil
	}
	if n > len(rep)-1 {
		n = len(rep) - 1
	}
	return copy(p, rep[1:1+n]), nil
}

func (c *cp2110Port) Write(p []byte) (int, error) {
	sent := 0
	for len(p) > 0 {
		n := min(len(p), 63)
		rep := append([]byte{byte(n)}, p[:n]...)
		if err := c.d.writeReport(rep); err != nil {
			return sent, err
		}
		sent += n
		p = p[n:]
	}
	return sent, nil
}
func (c *cp2110Port) Close() error { c.d.close(); return nil }

// Raw HID frames (Victor cable): each Read returns one report's payload.
type hidRawPort struct{ d hidHandle }

func platformOpenHIDRaw(path string) (Port, error) {
	d, err := openHIDDev(path)
	if err != nil {
		return nil, err
	}
	return &hidRawPort{d: d}, nil
}

func (h *hidRawPort) Read(p []byte) (int, error) {
	rep, err := h.d.readReport(200)
	if err != nil || len(rep) < 2 {
		return 0, err
	}
	return copy(p, rep[1:]), nil
}
func (h *hidRawPort) Write(p []byte) (int, error) { return len(p), nil }
func (h *hidRawPort) Close() error                { h.d.close(); return nil }

// Brymen BU-86X: plain HID reports. Writes go out as an output report whose
// first byte is the report ID (0), exactly like hidapi's hid_write; reads
// return each input report's payload without the report ID.
type bu86xPort struct{ d hidHandle }

func platformOpenBU86X(path string) (Port, error) {
	d, err := openHIDDev(path)
	if err != nil {
		return nil, err
	}
	d.defaultLens(9, 0)
	return &bu86xPort{d: d}, nil
}

func (b *bu86xPort) Read(p []byte) (int, error) {
	rep, err := b.d.readReport(200)
	if err != nil || len(rep) < 2 {
		return 0, err
	}
	return copy(p, rep[1:]), nil
}

// Write expects hidapi-style data: [reportID, payload...].
func (b *bu86xPort) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if err := b.d.writeReport(p); err != nil {
		return 0, err
	}
	return len(p), nil
}
func (b *bu86xPort) Close() error { b.d.close(); return nil }

func classifyPort(p *PortInfo) {
	if u := lookupUSB(p.VID, p.PID, p.Kind); u != nil {
		p.Meters = u.Meters
		switch u.Kind {
		case TrSerial:
			p.Hint = u.Label + " — press Detect to identify the meter"
		case TrCH9325:
			p.Hint = u.Label + " — UT61E / UT61B/C/D / UT60E"
		case TrCP2110:
			p.Hint = u.Label + " — connects directly"
		case TrVictor:
			p.Hint = u.Label + " — experimental"
		}
		if p.Name == "" {
			p.Name = u.Label
		}
		return
	}
	if p.Kind == TrSerial {
		p.Hint = "Serial port — press Detect to try the known meter protocols"
	}
}
