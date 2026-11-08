//go:build windows

package scopie

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

var (
	modSetupapi = syscall.NewLazyDLL("setupapi.dll")
	modAdvapi   = syscall.NewLazyDLL("advapi32.dll")
	modKernel   = syscall.NewLazyDLL("kernel32.dll")
	modShell32  = syscall.NewLazyDLL("shell32.dll")
	modWinusb   = syscall.NewLazyDLL("winusb.dll")

	procSetupDiGetClassDevsW              = modSetupapi.NewProc("SetupDiGetClassDevsW")
	procSetupDiEnumDeviceInfo             = modSetupapi.NewProc("SetupDiEnumDeviceInfo")
	procSetupDiGetDeviceRegistryPropertyW = modSetupapi.NewProc("SetupDiGetDeviceRegistryPropertyW")
	procSetupDiGetDeviceInstanceIdW       = modSetupapi.NewProc("SetupDiGetDeviceInstanceIdW")
	procSetupDiOpenDevRegKey              = modSetupapi.NewProc("SetupDiOpenDevRegKey")
	procSetupDiDestroyDeviceInfoList      = modSetupapi.NewProc("SetupDiDestroyDeviceInfoList")
	procSetupDiEnumDeviceInterfaces       = modSetupapi.NewProc("SetupDiEnumDeviceInterfaces")
	procSetupDiGetDeviceInterfaceDetailW  = modSetupapi.NewProc("SetupDiGetDeviceInterfaceDetailW")
	procShellExecuteW                     = modShell32.NewProc("ShellExecuteW")
	procCreateEventW                      = modKernel.NewProc("CreateEventW")
	procResetEvent                        = modKernel.NewProc("ResetEvent")
	procLoadLibraryExW                    = modKernel.NewProc("LoadLibraryExW")

	procWinUsbInitialize             = modWinusb.NewProc("WinUsb_Initialize")
	procWinUsbFree                   = modWinusb.NewProc("WinUsb_Free")
	procWinUsbControlTransfer        = modWinusb.NewProc("WinUsb_ControlTransfer")
	procWinUsbReadPipe               = modWinusb.NewProc("WinUsb_ReadPipe")
	procWinUsbWritePipe              = modWinusb.NewProc("WinUsb_WritePipe")
	procWinUsbAbortPipe              = modWinusb.NewProc("WinUsb_AbortPipe")
	procWinUsbResetPipe              = modWinusb.NewProc("WinUsb_ResetPipe")
	procWinUsbFlushPipe              = modWinusb.NewProc("WinUsb_FlushPipe")
	procWinUsbSetPipePolicy          = modWinusb.NewProc("WinUsb_SetPipePolicy")
	procWinUsbGetOverlappedResult    = modWinusb.NewProc("WinUsb_GetOverlappedResult")
	procWinUsbQueryInterfaceSettings = modWinusb.NewProc("WinUsb_QueryInterfaceSettings")
	procWinUsbQueryPipe              = modWinusb.NewProc("WinUsb_QueryPipe")
)

type winGUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

var (
	guidDevIfaceUSBDev = winGUID{0xA5DCBF10, 0x6530, 0x11D2, [8]byte{0x90, 0x1F, 0x00, 0xC0, 0x4F, 0xB9, 0x51, 0xED}}
	invalidHandleValue = ^uintptr(0)
)

const (
	digcfPresent         = 0x02
	digcfAllClasses      = 0x04
	digcfDeviceInterface = 0x10
	spdrpDeviceDesc      = 0x00
	spdrpHardwareID      = 0x01
	spdrpService         = 0x04
	spdrpFriendlyName    = 0x0C
	dicsFlagGlobal       = 1
	diregDev             = 1
	keyRead              = 0x20019

	fileFlagOverlapped  = 0x40000000
	errIOPending        = syscall.Errno(997)
	errOpAborted        = syscall.Errno(995)
	pipeTransferTimeout = 0x03
)

type spDevinfoData struct {
	CbSize    uint32
	ClassGuid winGUID
	DevInst   uint32
	Reserved  uintptr
}

type spDeviceInterfaceData struct {
	CbSize             uint32
	InterfaceClassGuid winGUID
	Flags              uint32
	Reserved           uintptr
}

func devRegString(set uintptr, di *spDevinfoData, prop uint32) string {
	buf := make([]uint16, 1024)
	var regType, need uint32
	r, _, _ := procSetupDiGetDeviceRegistryPropertyW.Call(set, uintptr(unsafe.Pointer(di)), uintptr(prop),
		uintptr(unsafe.Pointer(&regType)), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)*2), uintptr(unsafe.Pointer(&need)))
	if r == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf) // REG_MULTI_SZ: the first string
}

func devInstanceID(set uintptr, di *spDevinfoData) string {
	buf := make([]uint16, 512)
	var need uint32
	r, _, _ := procSetupDiGetDeviceInstanceIdW.Call(set, uintptr(unsafe.Pointer(di)), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)), uintptr(unsafe.Pointer(&need)))
	if r == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf)
}

// regMultiSz reads a REG_SZ or REG_MULTI_SZ value as a list of strings.
func regMultiSz(hk syscall.Handle, name string) []string {
	n, _ := syscall.UTF16PtrFromString(name)
	buf := make([]uint16, 2048)
	l := uint32(len(buf) * 2)
	var typ uint32
	if err := syscall.RegQueryValueEx(hk, n, nil, &typ, (*byte)(unsafe.Pointer(&buf[0])), &l); err != nil {
		return nil
	}
	var out []string
	start := 0
	for i := 0; i < int(l/2) && i < len(buf); i++ {
		if buf[i] == 0 {
			if i > start {
				out = append(out, syscall.UTF16ToString(buf[start:i]))
			}
			start = i + 1
		}
	}
	return out
}

func parseGUID(s string) (winGUID, bool) {
	s = strings.Trim(strings.TrimSpace(s), "{}")
	p := strings.Split(s, "-")
	if len(p) != 5 || len(p[0]) != 8 || len(p[1]) != 4 || len(p[2]) != 4 || len(p[3]) != 4 || len(p[4]) != 12 {
		return winGUID{}, false
	}
	var g winGUID
	a, e1 := strconv.ParseUint(p[0], 16, 32)
	b, e2 := strconv.ParseUint(p[1], 16, 16)
	c, e3 := strconv.ParseUint(p[2], 16, 16)
	if e1 != nil || e2 != nil || e3 != nil {
		return g, false
	}
	g.Data1, g.Data2, g.Data3 = uint32(a), uint16(b), uint16(c)
	rest := p[3] + p[4]
	for i := 0; i < 8; i++ {
		v, err := strconv.ParseUint(rest[2*i:2*i+2], 16, 8)
		if err != nil {
			return g, false
		}
		g.Data4[i] = byte(v)
	}
	return g, true
}

func hwField(u, key string) string {
	i := strings.Index(u, key)
	if i < 0 {
		return ""
	}
	j := i + len(key)
	k := j
	for k < len(u) && u[k] != '&' && u[k] != '\\' {
		k++
	}
	return u[j:k]
}

// interfacePaths maps device instance -> device path for one interface class GUID.
func interfacePaths(g winGUID) map[uint32]string {
	out := map[uint32]string{}
	set, _, _ := procSetupDiGetClassDevsW.Call(uintptr(unsafe.Pointer(&g)), 0, 0, digcfPresent|digcfDeviceInterface)
	if set == invalidHandleValue {
		return out
	}
	defer procSetupDiDestroyDeviceInfoList.Call(set)
	for i := uint32(0); ; i++ {
		var id spDeviceInterfaceData
		id.CbSize = uint32(unsafe.Sizeof(id))
		r, _, _ := procSetupDiEnumDeviceInterfaces.Call(set, 0, uintptr(unsafe.Pointer(&g)), uintptr(i), uintptr(unsafe.Pointer(&id)))
		if r == 0 {
			break
		}
		var need uint32
		procSetupDiGetDeviceInterfaceDetailW.Call(set, uintptr(unsafe.Pointer(&id)), 0, 0, uintptr(unsafe.Pointer(&need)), 0)
		if need < 8 || need > 8192 {
			continue
		}
		buf := make([]byte, need+8)
		cb := uint32(8) // sizeof(SP_DEVICE_INTERFACE_DETAIL_DATA_W) on 64-bit
		if unsafe.Sizeof(uintptr(0)) == 4 {
			cb = 6
		}
		*(*uint32)(unsafe.Pointer(&buf[0])) = cb
		var di spDevinfoData
		di.CbSize = uint32(unsafe.Sizeof(di))
		r, _, _ = procSetupDiGetDeviceInterfaceDetailW.Call(set, uintptr(unsafe.Pointer(&id)), uintptr(unsafe.Pointer(&buf[0])), uintptr(need), 0, uintptr(unsafe.Pointer(&di)))
		if r == 0 {
			continue
		}
		u16 := unsafe.Slice((*uint16)(unsafe.Pointer(&buf[4])), (need-4)/2)
		out[di.DevInst] = syscall.UTF16ToString(u16)
	}
	return out
}

// listUSBDevices lists every USB device and interface Windows knows about,
// with the path to open it when the WinUSB driver is attached.
func listUSBDevices() []usbInfo {
	set, _, _ := procSetupDiGetClassDevsW.Call(0, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("USB"))), 0, digcfPresent|digcfAllClasses)
	if set == invalidHandleValue {
		return nil
	}
	defer procSetupDiDestroyDeviceInfoList.Call(set)
	guidPaths := map[winGUID]map[uint32]string{}
	var generic map[uint32]string
	var out []usbInfo
	for i := uint32(0); ; i++ {
		var di spDevinfoData
		di.CbSize = uint32(unsafe.Sizeof(di))
		r, _, _ := procSetupDiEnumDeviceInfo.Call(set, uintptr(i), uintptr(unsafe.Pointer(&di)))
		if r == 0 {
			break
		}
		hw := strings.ToUpper(devRegString(set, &di, spdrpHardwareID))
		vid, e1 := strconv.ParseUint(hwField(hw, "VID_"), 16, 16)
		pid, e2 := strconv.ParseUint(hwField(hw, "PID_"), 16, 16)
		if e1 != nil || e2 != nil {
			continue
		}
		u := usbInfo{VID: uint16(vid), PID: uint16(pid), MI: -1}
		if rev, err := strconv.ParseUint(hwField(hw, "REV_"), 16, 16); err == nil {
			u.BCD = uint16(rev)
		}
		if mi, err := strconv.ParseUint(hwField(hw, "MI_"), 16, 8); err == nil {
			u.MI = int(mi)
		}
		u.Service = devRegString(set, &di, spdrpService)
		u.Desc = devRegString(set, &di, spdrpFriendlyName)
		if u.Desc == "" {
			u.Desc = devRegString(set, &di, spdrpDeviceDesc)
		}
		inst := devInstanceID(set, &di)
		u.Key = inst
		if parts := strings.Split(inst, `\`); len(parts) == 3 && u.MI < 0 && !strings.Contains(parts[2], "&") {
			u.Serial = parts[2]
		}
		if strings.EqualFold(u.Service, "WinUSB") {
			u.WinUSB = true
			hk, _, _ := procSetupDiOpenDevRegKey.Call(set, uintptr(unsafe.Pointer(&di)), dicsFlagGlobal, 0, diregDev, keyRead)
			if hk != invalidHandleValue && hk != 0 {
				guids := regMultiSz(syscall.Handle(hk), "DeviceInterfaceGUIDs")
				guids = append(guids, regMultiSz(syscall.Handle(hk), "DeviceInterfaceGUID")...)
				syscall.RegCloseKey(syscall.Handle(hk))
				for _, gs := range guids {
					g, ok := parseGUID(gs)
					if !ok {
						continue
					}
					m, done := guidPaths[g]
					if !done {
						m = interfacePaths(g)
						guidPaths[g] = m
					}
					if p := m[di.DevInst]; p != "" {
						u.Path = p
						break
					}
				}
			}
			if u.Path == "" && u.MI < 0 {
				if generic == nil {
					generic = interfacePaths(guidDevIfaceUSBDev)
				}
				u.Path = generic[di.DevInst]
			}
		}
		out = append(out, u)
	}
	return out
}

// ---------------------------------------------------------------- WinUSB device

type winUSB struct {
	h       syscall.Handle
	w       uintptr
	path    string
	in, out []uint8
	class   uint8
	sub     uint8
}

type usbIfaceDesc struct {
	BLength, BDescriptorType, BInterfaceNumber, BAlternateSetting, BNumEndpoints uint8
	BInterfaceClass, BInterfaceSubClass, BInterfaceProtocol, IInterface          uint8
}

type winusbPipeInfo struct {
	PipeType  int32
	PipeID    uint8
	_         uint8
	MaxPacket uint16
	Interval  uint8
	_         [3]uint8
}

func openUSBDevice(path string) (usbDevice, error) {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	h, err := syscall.CreateFile(p, syscall.GENERIC_READ|syscall.GENERIC_WRITE,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE, nil, syscall.OPEN_EXISTING,
		syscall.FILE_ATTRIBUTE_NORMAL|fileFlagOverlapped, 0)
	if err != nil {
		return nil, fmt.Errorf("open USB device: %w", err)
	}
	var w uintptr
	if r, _, e := procWinUsbInitialize.Call(uintptr(h), uintptr(unsafe.Pointer(&w))); r == 0 {
		syscall.CloseHandle(h)
		return nil, &driverError{fmt.Sprintf("WinUSB driver not attached (%v)", e)}
	}
	d := &winUSB{h: h, w: w, path: path}
	to := uint32(2000)
	procWinUsbSetPipePolicy.Call(w, 0, pipeTransferTimeout, 4, uintptr(unsafe.Pointer(&to)))
	var desc usbIfaceDesc
	if r, _, _ := procWinUsbQueryInterfaceSettings.Call(w, 0, uintptr(unsafe.Pointer(&desc))); r != 0 {
		d.class, d.sub = desc.BInterfaceClass, desc.BInterfaceSubClass
		for i := uint8(0); i < desc.BNumEndpoints; i++ {
			var pi winusbPipeInfo
			if r, _, _ := procWinUsbQueryPipe.Call(w, 0, uintptr(i), uintptr(unsafe.Pointer(&pi))); r == 0 {
				continue
			}
			if pi.PipeType != 2 { // UsbdPipeTypeBulk
				continue
			}
			if pi.PipeID&0x80 != 0 {
				d.in = append(d.in, pi.PipeID)
			} else {
				d.out = append(d.out, pi.PipeID)
			}
		}
	}
	return d, nil
}

func (d *winUSB) Endpoints() ([]uint8, []uint8, uint8, uint8) { return d.in, d.out, d.class, d.sub }

func (d *winUSB) Close() {
	if d.w != 0 {
		procWinUsbFree.Call(d.w)
		d.w = 0
	}
	if d.h != 0 {
		syscall.CloseHandle(d.h)
		d.h = 0
	}
}

func (d *winUSB) Control(rt, req uint8, val, idx uint16, data []byte) (int, error) {
	if d.w == 0 {
		return 0, errors.New("device closed")
	}
	setup := uint64(rt) | uint64(req)<<8 | uint64(val)<<16 | uint64(idx)<<32 | uint64(len(data))<<48
	var ptr uintptr
	if len(data) > 0 {
		ptr = uintptr(unsafe.Pointer(&data[0]))
	}
	var n uint32
	r, _, e := procWinUsbControlTransfer.Call(d.w, uintptr(setup), ptr, uintptr(len(data)), uintptr(unsafe.Pointer(&n)), 0)
	if r == 0 {
		return int(n), fmt.Errorf("control transfer 0x%02x: %v", req, e)
	}
	return int(n), nil
}

func (d *winUSB) ResetPipe(ep uint8) {
	procWinUsbAbortPipe.Call(d.w, uintptr(ep))
	procWinUsbFlushPipe.Call(d.w, uintptr(ep))
	procWinUsbResetPipe.Call(d.w, uintptr(ep))
}

// xfer runs one overlapped bulk transfer with a timeout.
func (d *winUSB) xfer(read bool, ep uint8, buf []byte, timeoutMs int) (int, error) {
	if d.w == 0 {
		return 0, errors.New("device closed")
	}
	if len(buf) == 0 {
		return 0, nil
	}
	ev, _, e := procCreateEventW.Call(0, 1, 0, 0)
	if ev == 0 {
		return 0, fmt.Errorf("CreateEvent: %v", e)
	}
	defer syscall.CloseHandle(syscall.Handle(ev))
	ov := &syscall.Overlapped{HEvent: syscall.Handle(ev)}
	proc := procWinUsbWritePipe
	if read {
		proc = procWinUsbReadPipe
	}
	r, _, e := proc.Call(d.w, uintptr(ep), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)), 0, uintptr(unsafe.Pointer(ov)))
	if r == 0 && e != errIOPending {
		return 0, fmt.Errorf("bulk transfer: %v", e)
	}
	if timeoutMs <= 0 {
		timeoutMs = 2000
	}
	timedOut := false
	if w, _ := syscall.WaitForSingleObject(ov.HEvent, uint32(timeoutMs)); w != syscall.WAIT_OBJECT_0 {
		procWinUsbAbortPipe.Call(d.w, uintptr(ep))
		timedOut = true
	}
	var n uint32
	r, _, e = procWinUsbGetOverlappedResult.Call(d.w, uintptr(unsafe.Pointer(ov)), uintptr(unsafe.Pointer(&n)), 1)
	if timedOut {
		return int(n), errors.New("timed out")
	}
	if r == 0 {
		return int(n), fmt.Errorf("bulk transfer failed: %v", e)
	}
	return int(n), nil
}

func (d *winUSB) BulkWrite(ep uint8, data []byte, timeoutMs int) (int, error) {
	return d.xfer(false, ep, data, timeoutMs)
}

func (d *winUSB) BulkRead(ep uint8, buf []byte, timeoutMs int) (int, error) {
	return d.xfer(true, ep, buf, timeoutMs)
}

type winXfer struct {
	buf     []byte
	ov      *syscall.Overlapped
	pending bool
}

func (d *winUSB) Stream(ep uint8, size, count int, stop <-chan struct{}, fn func([]byte) bool) (err error) {
	xs := make([]*winXfer, count)
	for i := range xs {
		ev, _, e := procCreateEventW.Call(0, 1, 0, 0)
		if ev == 0 {
			return fmt.Errorf("CreateEvent: %v", e)
		}
		xs[i] = &winXfer{buf: make([]byte, size), ov: &syscall.Overlapped{HEvent: syscall.Handle(ev)}}
	}
	defer func() {
		procWinUsbAbortPipe.Call(d.w, uintptr(ep))
		for _, x := range xs {
			if x.pending {
				var n uint32
				procWinUsbGetOverlappedResult.Call(d.w, uintptr(unsafe.Pointer(x.ov)), uintptr(unsafe.Pointer(&n)), 1)
			}
			syscall.CloseHandle(x.ov.HEvent)
		}
	}()
	submit := func(x *winXfer) error {
		ev := x.ov.HEvent
		*x.ov = syscall.Overlapped{HEvent: ev}
		procResetEvent.Call(uintptr(ev))
		r, _, e := procWinUsbReadPipe.Call(d.w, uintptr(ep), uintptr(unsafe.Pointer(&x.buf[0])), uintptr(len(x.buf)), 0, uintptr(unsafe.Pointer(x.ov)))
		if r == 0 && e != errIOPending {
			return fmt.Errorf("bulk read: %v", e)
		}
		x.pending = true
		return nil
	}
	for _, x := range xs {
		if err := submit(x); err != nil {
			return err
		}
	}
	for i := 0; ; i = (i + 1) % count {
		x := xs[i]
		for {
			ev, _ := syscall.WaitForSingleObject(x.ov.HEvent, 100)
			if ev == syscall.WAIT_OBJECT_0 {
				break
			}
			select {
			case <-stop:
				return nil
			default:
			}
		}
		var n uint32
		r, _, e := procWinUsbGetOverlappedResult.Call(d.w, uintptr(unsafe.Pointer(x.ov)), uintptr(unsafe.Pointer(&n)), 0)
		x.pending = false
		if r == 0 {
			if e == errOpAborted {
				return nil
			}
			return fmt.Errorf("bulk read failed: %v", e)
		}
		if n > 0 && !fn(x.buf[:n]) {
			return nil
		}
		select {
		case <-stop:
			return nil
		default:
		}
		if err := submit(x); err != nil {
			return err
		}
	}
}

// ---------------------------------------------------------------- misc Windows helpers

func shellExecute(verb, file, args, dir string) error {
	v, _ := syscall.UTF16PtrFromString(verb)
	f, _ := syscall.UTF16PtrFromString(file)
	var a, d *uint16
	if args != "" {
		a, _ = syscall.UTF16PtrFromString(args)
	}
	if dir != "" {
		d, _ = syscall.UTF16PtrFromString(dir)
	}
	r, _, _ := procShellExecuteW.Call(0, uintptr(unsafe.Pointer(v)), uintptr(unsafe.Pointer(f)), uintptr(unsafe.Pointer(a)), uintptr(unsafe.Pointer(d)), 1)
	if r <= 32 {
		if r == 5 || r == 1223 {
			return errors.New("permission was not given")
		}
		return fmt.Errorf("could not start %s (code %d)", file, r)
	}
	return nil
}

func openFolder(dir string) { _ = shellExecute("open", dir, "", "") }
func openURL(url string)    { _ = shellExecute("open", url, "", "") }

// loadDLL loads a DLL by full path so its own folder is searched for the DLLs it needs.
func loadDLL(path string) (*syscall.DLL, error) {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	const loadWithAlteredSearchPath = 0x8
	h, _, e := procLoadLibraryExW.Call(uintptr(unsafe.Pointer(p)), 0, loadWithAlteredSearchPath)
	if h == 0 {
		return nil, fmt.Errorf("%s: %v", path, e)
	}
	return &syscall.DLL{Name: path, Handle: syscall.Handle(h)}, nil
}

// usbDriverHint says what to do when a device can't be used yet.
func usbDriverHint(u usbInfo) string {
	return fmt.Sprintf("USB %04X:%04X: needs the WinUSB driver. Click 'Install USB driver…'.", u.VID, u.PID)
}
