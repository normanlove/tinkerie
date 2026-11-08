//go:build windows

package meterie

// serial_windows.go: COM ports and USB-HID meter cables (UNI-T UT-D04
// CH9325/HE2325U, Silicon Labs CP2110, Victor) via kernel32/setupapi/hid.dll.

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

var (
	setupapi = sysDLL("setupapi.dll")
	hidDLL   = sysDLL("hid.dll")
	advapi32 = sysDLL("advapi32.dll")

	pGetCommState        = kernel32.NewProc("GetCommState")
	pSetCommState        = kernel32.NewProc("SetCommState")
	pSetCommTimeouts     = kernel32.NewProc("SetCommTimeouts")
	pEscapeCommFunction  = kernel32.NewProc("EscapeCommFunction")
	pPurgeComm           = kernel32.NewProc("PurgeComm")
	pCreateEventW        = kernel32.NewProc("CreateEventW")
	pGetOverlappedResult = kernel32.NewProc("GetOverlappedResult")
	pCancelIoEx          = kernel32.NewProc("CancelIoEx")

	pSetupDiGetClassDevsW              = setupapi.NewProc("SetupDiGetClassDevsW")
	pSetupDiEnumDeviceInfo             = setupapi.NewProc("SetupDiEnumDeviceInfo")
	pSetupDiGetDeviceRegistryPropertyW = setupapi.NewProc("SetupDiGetDeviceRegistryPropertyW")
	pSetupDiOpenDevRegKey              = setupapi.NewProc("SetupDiOpenDevRegKey")
	pSetupDiEnumDeviceInterfaces       = setupapi.NewProc("SetupDiEnumDeviceInterfaces")
	pSetupDiGetDeviceInterfaceDetailW  = setupapi.NewProc("SetupDiGetDeviceInterfaceDetailW")
	pSetupDiDestroyDeviceInfoList      = setupapi.NewProc("SetupDiDestroyDeviceInfoList")

	pHidD_GetHidGuid         = hidDLL.NewProc("HidD_GetHidGuid")
	pHidD_GetAttributes      = hidDLL.NewProc("HidD_GetAttributes")
	pHidD_GetProductString   = hidDLL.NewProc("HidD_GetProductString")
	pHidD_SetFeature         = hidDLL.NewProc("HidD_SetFeature")
	pHidD_GetPreparsedData   = hidDLL.NewProc("HidD_GetPreparsedData")
	pHidD_FreePreparsedData  = hidDLL.NewProc("HidD_FreePreparsedData")
	pHidP_GetCaps            = hidDLL.NewProc("HidP_GetCaps")
	pHidD_SetNumInputBuffers = hidDLL.NewProc("HidD_SetNumInputBuffers")

	pRegEnumValueW = advapi32.NewProc("RegEnumValueW")
)

const (
	invalidHandle = ^uintptr(0)
	errIOPending  = syscall.Errno(997)
	waitTimeout   = 0x102
)

// ---------------------------------------------------------------------------
// Serial

type dcbT struct {
	DCBlength  uint32
	BaudRate   uint32
	Flags      uint32
	wReserved  uint16
	XonLim     uint16
	XoffLim    uint16
	ByteSize   byte
	Parity     byte
	StopBits   byte
	XonChar    byte
	XoffChar   byte
	ErrorChar  byte
	EofChar    byte
	EvtChar    byte
	wReserved1 uint16
}

type commTimeouts struct {
	ReadIntervalTimeout, ReadTotalTimeoutMultiplier, ReadTotalTimeoutConstant uint32
	WriteTotalTimeoutMultiplier, WriteTotalTimeoutConstant                    uint32
}

type serialPort struct{ h syscall.Handle }

func platformOpenSerial(name string, cfg SerialCfg) (Port, error) {
	path := name
	if !strings.HasPrefix(path, `\\.\`) {
		path = `\\.\` + name
	}
	h, err := syscall.CreateFile(u16(path), syscall.GENERIC_READ|syscall.GENERIC_WRITE, 0, nil, syscall.OPEN_EXISTING, 0, 0)
	if err != nil {
		if err == syscall.ERROR_ACCESS_DENIED {
			return nil, fmt.Errorf("%s is in use by another program — close the meter's own software", name)
		}
		if err == syscall.ERROR_FILE_NOT_FOUND {
			return nil, fmt.Errorf("%s isn't there any more — is the cable plugged in?", name)
		}
		return nil, fmt.Errorf("can't open %s: %v", name, err)
	}
	var d dcbT
	d.DCBlength = uint32(unsafe.Sizeof(d))
	if r, _, e := pGetCommState.Call(uintptr(h), uintptr(unsafe.Pointer(&d))); r == 0 {
		syscall.CloseHandle(h)
		return nil, fmt.Errorf("GetCommState: %v", e)
	}
	d.BaudRate = uint32(cfg.Baud)
	if cfg.DataBits == 0 {
		cfg.DataBits = 8
	}
	d.ByteSize = byte(cfg.DataBits)
	switch cfg.Parity {
	case 'O':
		d.Parity = 1
	case 'E':
		d.Parity = 2
	default:
		d.Parity = 0
	}
	if cfg.StopBits == 2 {
		d.StopBits = 2
	} else {
		d.StopBits = 0
	}
	flags := uint32(1) // fBinary
	if d.Parity != 0 {
		flags |= 1 << 1
	}
	if cfg.DTR {
		flags |= 1 << 4
	}
	if cfg.RTS {
		flags |= 1 << 12
	}
	d.Flags = flags
	if r, _, e := pSetCommState.Call(uintptr(h), uintptr(unsafe.Pointer(&d))); r == 0 {
		syscall.CloseHandle(h)
		return nil, fmt.Errorf("can't set %s to %s: %v", name, cfg, e)
	}
	// Optically isolated cables are often powered from DTR (on) with RTS (off).
	if cfg.DTR {
		pEscapeCommFunction.Call(uintptr(h), 5)
	} else {
		pEscapeCommFunction.Call(uintptr(h), 6)
	}
	if cfg.RTS {
		pEscapeCommFunction.Call(uintptr(h), 3)
	} else {
		pEscapeCommFunction.Call(uintptr(h), 4)
	}
	to := commTimeouts{ReadIntervalTimeout: 0xFFFFFFFF, ReadTotalTimeoutMultiplier: 0xFFFFFFFF, ReadTotalTimeoutConstant: 150, WriteTotalTimeoutConstant: 500}
	pSetCommTimeouts.Call(uintptr(h), uintptr(unsafe.Pointer(&to)))
	pPurgeComm.Call(uintptr(h), 0x0004|0x0008)
	return &serialPort{h: h}, nil
}

func (s *serialPort) Read(p []byte) (int, error) {
	var n uint32
	if err := syscall.ReadFile(s.h, p, &n, nil); err != nil {
		return 0, fmt.Errorf("serial read failed (cable unplugged?): %v", err)
	}
	return int(n), nil
}

func (s *serialPort) Write(p []byte) (int, error) {
	var n uint32
	err := syscall.WriteFile(s.h, p, &n, nil)
	return int(n), err
}

func (s *serialPort) Close() error { return syscall.CloseHandle(s.h) }

// ---------------------------------------------------------------------------
// HID

type hidAttributes struct {
	Size          uint32
	VendorID      uint16
	ProductID     uint16
	VersionNumber uint16
	_             uint16
}

type hidpCaps struct {
	Usage, UsagePage                              uint16
	InputReportByteLength, OutputReportByteLength uint16
	FeatureReportByteLength                       uint16
	Reserved                                      [17]uint16
	Counts                                        [10]uint16
}

type hidDev struct {
	h                      syscall.Handle
	inLen, outLen, featLen int
	rov, wov               syscall.Overlapped
	rbuf                   []byte
}

func createEvent() syscall.Handle {
	r, _, _ := pCreateEventW.Call(0, 1, 0, 0)
	return syscall.Handle(r)
}

func openHID(path string) (*hidDev, error) {
	h, err := syscall.CreateFile(u16(path), syscall.GENERIC_READ|syscall.GENERIC_WRITE, syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE, nil, syscall.OPEN_EXISTING, syscall.FILE_FLAG_OVERLAPPED, 0)
	if err != nil {
		return nil, fmt.Errorf("can't open the USB cable (%v) — is the meter's own software running?", err)
	}
	d := &hidDev{h: h}
	var pp uintptr
	if r, _, _ := pHidD_GetPreparsedData.Call(uintptr(h), uintptr(unsafe.Pointer(&pp))); r != 0 {
		var caps hidpCaps
		pHidP_GetCaps.Call(pp, uintptr(unsafe.Pointer(&caps)))
		pHidD_FreePreparsedData.Call(pp)
		d.inLen, d.outLen, d.featLen = int(caps.InputReportByteLength), int(caps.OutputReportByteLength), int(caps.FeatureReportByteLength)
	}
	if d.inLen == 0 {
		d.inLen = 65
	}
	pHidD_SetNumInputBuffers.Call(uintptr(h), 64)
	d.rov.HEvent = createEvent()
	d.wov.HEvent = createEvent()
	d.rbuf = make([]byte, d.inLen)
	return d, nil
}

func (d *hidDev) close() {
	pCancelIoEx.Call(uintptr(d.h), 0)
	syscall.CloseHandle(d.h)
	syscall.CloseHandle(d.rov.HEvent)
	syscall.CloseHandle(d.wov.HEvent)
}

// readReport returns one input report (report ID at [0]) or nil on timeout.
func (d *hidDev) readReport(timeoutMs uint32) ([]byte, error) {
	ev := d.rov.HEvent
	d.rov = syscall.Overlapped{HEvent: ev}
	var n uint32
	err := syscall.ReadFile(d.h, d.rbuf, &n, &d.rov)
	if err != nil && err != errIOPending {
		return nil, fmt.Errorf("USB read failed (cable unplugged?): %v", err)
	}
	if err == errIOPending {
		w, _ := syscall.WaitForSingleObject(ev, timeoutMs)
		if w == waitTimeout {
			pCancelIoEx.Call(uintptr(d.h), uintptr(unsafe.Pointer(&d.rov)))
			pGetOverlappedResult.Call(uintptr(d.h), uintptr(unsafe.Pointer(&d.rov)), uintptr(unsafe.Pointer(&n)), 1)
			if n == 0 {
				return nil, nil
			}
		} else {
			r, _, e := pGetOverlappedResult.Call(uintptr(d.h), uintptr(unsafe.Pointer(&d.rov)), uintptr(unsafe.Pointer(&n)), 0)
			if r == 0 {
				return nil, fmt.Errorf("USB read: %v", e)
			}
		}
	}
	out := make([]byte, n)
	copy(out, d.rbuf[:n])
	return out, nil
}

func (d *hidDev) writeReport(b []byte) error {
	buf := make([]byte, max(d.outLen, len(b)))
	copy(buf, b)
	ev := d.wov.HEvent
	d.wov = syscall.Overlapped{HEvent: ev}
	var n uint32
	err := syscall.WriteFile(d.h, buf, &n, &d.wov)
	if err != nil && err != errIOPending {
		return fmt.Errorf("USB write: %v", err)
	}
	if err == errIOPending {
		if w, _ := syscall.WaitForSingleObject(ev, 1000); w == waitTimeout {
			pCancelIoEx.Call(uintptr(d.h), uintptr(unsafe.Pointer(&d.wov)))
			return fmt.Errorf("USB write timed out")
		}
		pGetOverlappedResult.Call(uintptr(d.h), uintptr(unsafe.Pointer(&d.wov)), uintptr(unsafe.Pointer(&n)), 0)
	}
	return nil
}

func (d *hidDev) setFeature(b []byte) error {
	buf := make([]byte, max(d.featLen, len(b)))
	copy(buf, b)
	if r, _, e := pHidD_SetFeature.Call(uintptr(d.h), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf))); r == 0 {
		return fmt.Errorf("HidD_SetFeature: %v", e)
	}
	return nil
}

// UNI-T UT-D04 (CH9325 / HE2325U): a feature report sets the baud rate; input
// reports carry up to 7 UART bytes: [reportID, 0xF0|count, data...].
type ch9325Port struct {
	d     *hidDev
	mask7 bool
}

func platformOpenCH9325(path string, cfg SerialCfg) (Port, error) {
	d, err := openHID(path)
	if err != nil {
		return nil, err
	}
	b := cfg.Baud
	fr := []byte{0x00, byte(b), byte(b >> 8), byte(b >> 16), byte(b >> 24), 0x03}
	if err := d.setFeature(fr); err != nil {
		d.featLen = 6
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
type cp2110Port struct{ d *hidDev }

func platformOpenCP2110(path string, cfg SerialCfg) (Port, error) {
	d, err := openHID(path)
	if err != nil {
		return nil, err
	}
	if d.featLen == 0 {
		d.featLen = 64
	}
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
type hidRawPort struct{ d *hidDev }

func platformOpenHIDRaw(path string) (Port, error) {
	d, err := openHID(path)
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
type bu86xPort struct{ d *hidDev }

func platformOpenBU86X(path string) (Port, error) {
	d, err := openHID(path)
	if err != nil {
		return nil, err
	}
	if d.outLen == 0 {
		d.outLen = 9
	}
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

// ---------------------------------------------------------------------------
// Enumeration

type spDevinfoData struct {
	cbSize    uint32
	ClassGuid syscall.GUID
	DevInst   uint32
	Reserved  uintptr
}

type spDeviceInterfaceData struct {
	cbSize             uint32
	InterfaceClassGuid syscall.GUID
	Flags              uint32
	Reserved           uintptr
}

var vidpidRe = regexp.MustCompile(`(?i)VID_([0-9A-F]{4}).*PID_([0-9A-F]{4})`)

func parseVIDPID(s string) (uint16, uint16) {
	m := vidpidRe.FindStringSubmatch(s)
	if m == nil {
		return 0, 0
	}
	v, _ := strconv.ParseUint(m[1], 16, 16)
	p, _ := strconv.ParseUint(m[2], 16, 16)
	return uint16(v), uint16(p)
}

func devProp(h uintptr, did *spDevinfoData, prop uint32) string {
	buf := make([]uint16, 1024)
	var typ, req uint32
	r, _, _ := pSetupDiGetDeviceRegistryPropertyW.Call(h, uintptr(unsafe.Pointer(did)), uintptr(prop), uintptr(unsafe.Pointer(&typ)), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)*2), uintptr(unsafe.Pointer(&req)))
	if r == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf)
}

func regString(k syscall.Handle, name string) string {
	buf := make([]uint16, 256)
	l := uint32(len(buf) * 2)
	var typ uint32
	if syscall.RegQueryValueEx(k, u16(name), nil, &typ, (*byte)(unsafe.Pointer(&buf[0])), &l) != nil {
		return ""
	}
	return syscall.UTF16ToString(buf)
}

func listSerialPorts() []PortInfo {
	seen := map[string]bool{}
	var out []PortInfo
	guidPorts := syscall.GUID{Data1: 0x4D36E978, Data2: 0xE325, Data3: 0x11CE, Data4: [8]byte{0xBF, 0xC1, 0x08, 0x00, 0x2B, 0xE1, 0x03, 0x18}}
	h, _, _ := pSetupDiGetClassDevsW.Call(uintptr(unsafe.Pointer(&guidPorts)), 0, 0, 0x2)
	if h != invalidHandle {
		for i := uint32(0); ; i++ {
			var did spDevinfoData
			did.cbSize = uint32(unsafe.Sizeof(did))
			if r, _, _ := pSetupDiEnumDeviceInfo.Call(h, uintptr(i), uintptr(unsafe.Pointer(&did))); r == 0 {
				break
			}
			friendly := devProp(h, &did, 0x0C)
			hwid := devProp(h, &did, 0x01)
			key, _, _ := pSetupDiOpenDevRegKey.Call(h, uintptr(unsafe.Pointer(&did)), 1, 0, 1, syscall.KEY_READ)
			port := ""
			if key != invalidHandle && key != 0 {
				port = regString(syscall.Handle(key), "PortName")
				syscall.RegCloseKey(syscall.Handle(key))
			}
			if !strings.HasPrefix(strings.ToUpper(port), "COM") {
				continue
			}
			vid, pid := parseVIDPID(hwid)
			pi := PortInfo{Kind: TrSerial, Path: port, Name: friendly, VID: vid, PID: pid}
			classifyPort(&pi)
			out = append(out, pi)
			seen[strings.ToUpper(port)] = true
		}
		pSetupDiDestroyDeviceInfoList.Call(h)
	}
	var k syscall.Handle
	if syscall.RegOpenKeyEx(syscall.HKEY_LOCAL_MACHINE, u16(`HARDWARE\DEVICEMAP\SERIALCOMM`), 0, syscall.KEY_READ, &k) == nil {
		for i := uint32(0); ; i++ {
			name := make([]uint16, 256)
			nameLen := uint32(len(name))
			data := make([]uint16, 64)
			dataLen := uint32(len(data) * 2)
			var typ uint32
			r, _, _ := pRegEnumValueW.Call(uintptr(k), uintptr(i), uintptr(unsafe.Pointer(&name[0])), uintptr(unsafe.Pointer(&nameLen)), 0, uintptr(unsafe.Pointer(&typ)), uintptr(unsafe.Pointer(&data[0])), uintptr(unsafe.Pointer(&dataLen)))
			if r != 0 {
				break
			}
			port := syscall.UTF16ToString(data)
			if port == "" || seen[strings.ToUpper(port)] {
				continue
			}
			pi := PortInfo{Kind: TrSerial, Path: port, Name: port}
			classifyPort(&pi)
			out = append(out, pi)
		}
		syscall.RegCloseKey(k)
	}
	return out
}

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

func listHID() []PortInfo {
	var out []PortInfo
	var guid syscall.GUID
	pHidD_GetHidGuid.Call(uintptr(unsafe.Pointer(&guid)))
	h, _, _ := pSetupDiGetClassDevsW.Call(uintptr(unsafe.Pointer(&guid)), 0, 0, 0x12)
	if h == invalidHandle {
		return nil
	}
	defer pSetupDiDestroyDeviceInfoList.Call(h)
	for i := uint32(0); ; i++ {
		var ifd spDeviceInterfaceData
		ifd.cbSize = uint32(unsafe.Sizeof(ifd))
		if r, _, _ := pSetupDiEnumDeviceInterfaces.Call(h, 0, uintptr(unsafe.Pointer(&guid)), uintptr(i), uintptr(unsafe.Pointer(&ifd))); r == 0 {
			break
		}
		var req uint32
		pSetupDiGetDeviceInterfaceDetailW.Call(h, uintptr(unsafe.Pointer(&ifd)), 0, 0, uintptr(unsafe.Pointer(&req)), 0)
		if req < 8 || req > 4096 {
			continue
		}
		buf := make([]byte, req+8)
		*(*uint32)(unsafe.Pointer(&buf[0])) = 8 // cbSize on 64-bit
		if r, _, _ := pSetupDiGetDeviceInterfaceDetailW.Call(h, uintptr(unsafe.Pointer(&ifd)), uintptr(unsafe.Pointer(&buf[0])), uintptr(req), 0, 0); r == 0 {
			continue
		}
		path := syscall.UTF16ToString(unsafe.Slice((*uint16)(unsafe.Pointer(&buf[4])), (req-4)/2))
		vid, pid := parseVIDPID(path)
		var kind Transport = -1
		for _, k := range []Transport{TrCH9325, TrCP2110, TrVictor, TrBU86X} {
			if u := lookupUSB(vid, pid, k); u != nil && u.Kind == k {
				kind = k
			}
		}
		if kind < 0 {
			continue
		}
		name := ""
		if hh, err := syscall.CreateFile(u16(path), 0, syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE, nil, syscall.OPEN_EXISTING, 0, 0); err == nil {
			sb := make([]uint16, 128)
			if r, _, _ := pHidD_GetProductString.Call(uintptr(hh), uintptr(unsafe.Pointer(&sb[0])), uintptr(len(sb)*2)); r != 0 {
				name = syscall.UTF16ToString(sb)
			}
			syscall.CloseHandle(hh)
		}
		pi := PortInfo{Kind: kind, Path: path, Name: name, VID: vid, PID: pid}
		classifyPort(&pi)
		if name != "" {
			pi.Name = name + " (" + fmt.Sprintf("%04X:%04X", vid, pid) + ")"
		}
		// CP2110 exposes one HID interface; CH9325 sometimes two collections — keep the first
		dup := false
		for _, o := range out {
			if o.VID == vid && o.PID == pid && strings.EqualFold(hidInstance(o.Path), hidInstance(path)) {
				dup = true
			}
		}
		if !dup {
			out = append(out, pi)
		}
	}
	return out
}

// hidInstance returns the device-instance part of a HID interface path.
func hidInstance(p string) string {
	parts := strings.Split(p, "#")
	if len(parts) > 2 {
		return parts[2]
	}
	return p
}

func platformListPorts() []PortInfo {
	return append(listHID(), listSerialPorts()...)
}
