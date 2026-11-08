//go:build windows

package tracie

import (
	"strings"
	"syscall"
	"unsafe"
)

var (
	modSetupapi = syscall.NewLazyDLL("setupapi.dll")
	modAdvapi   = syscall.NewLazyDLL("advapi32.dll")
	modKernel   = syscall.NewLazyDLL("kernel32.dll")
	modShell32  = syscall.NewLazyDLL("shell32.dll")

	procSetupDiGetClassDevsW              = modSetupapi.NewProc("SetupDiGetClassDevsW")
	procSetupDiEnumDeviceInfo             = modSetupapi.NewProc("SetupDiEnumDeviceInfo")
	procSetupDiGetDeviceRegistryPropertyW = modSetupapi.NewProc("SetupDiGetDeviceRegistryPropertyW")
	procSetupDiOpenDevRegKey              = modSetupapi.NewProc("SetupDiOpenDevRegKey")
	procSetupDiDestroyDeviceInfoList      = modSetupapi.NewProc("SetupDiDestroyDeviceInfoList")
	procSetupDiEnumDeviceInterfaces       = modSetupapi.NewProc("SetupDiEnumDeviceInterfaces")
	procSetupDiGetDeviceInterfaceDetailW  = modSetupapi.NewProc("SetupDiGetDeviceInterfaceDetailW")
	procRegEnumValueW                     = modAdvapi.NewProc("RegEnumValueW")
	procShellExecuteW                     = modShell32.NewProc("ShellExecuteW")
)

type winGUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

var (
	guidDevClassPorts  = winGUID{0x4D36E978, 0xE325, 0x11CE, [8]byte{0xBF, 0xC1, 0x08, 0x00, 0x2B, 0xE1, 0x03, 0x18}}
	guidDevIfaceUSBDev = winGUID{0xA5DCBF10, 0x6530, 0x11D2, [8]byte{0x90, 0x1F, 0x00, 0xC0, 0x4F, 0xB9, 0x51, 0xED}}
	invalidHandleValue = ^uintptr(0)
)

const (
	digcfPresent         = 0x02
	digcfDeviceInterface = 0x10
	spdrpHardwareID      = 0x01
	spdrpFriendlyName    = 0x0C
	spdrpDeviceDesc      = 0x00
	dicsFlagGlobal       = 1
	diregDev             = 1
	keyRead              = 0x20019
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
	// REG_MULTI_SZ: keep only the first string, which is what we want.
	return syscall.UTF16ToString(buf)
}

func parseVIDPID(s string) (vid, pid string) {
	u := strings.ToUpper(s)
	if i := strings.Index(u, "VID_"); i >= 0 && i+8 <= len(u) {
		vid = u[i+4 : i+8]
	}
	if i := strings.Index(u, "PID_"); i >= 0 && i+8 <= len(u) {
		pid = u[i+4 : i+8]
	}
	return
}

// listPorts enumerates COM ports with friendly names and USB IDs.
func listPorts() []PortInfo {
	seen := map[string]bool{}
	var out []PortInfo

	set, _, _ := procSetupDiGetClassDevsW.Call(uintptr(unsafe.Pointer(&guidDevClassPorts)), 0, 0, digcfPresent)
	if set != invalidHandleValue {
		for i := uint32(0); ; i++ {
			var di spDevinfoData
			di.CbSize = uint32(unsafe.Sizeof(di))
			r, _, _ := procSetupDiEnumDeviceInfo.Call(set, uintptr(i), uintptr(unsafe.Pointer(&di)))
			if r == 0 {
				break
			}
			name := ""
			hk, _, _ := procSetupDiOpenDevRegKey.Call(set, uintptr(unsafe.Pointer(&di)), dicsFlagGlobal, 0, diregDev, keyRead)
			if hk != invalidHandleValue && hk != 0 {
				name = regQueryString(syscall.Handle(hk), "PortName")
				syscall.RegCloseKey(syscall.Handle(hk))
			}
			if !strings.HasPrefix(strings.ToUpper(name), "COM") {
				continue
			}
			desc := devRegString(set, &di, spdrpFriendlyName)
			if desc == "" {
				desc = devRegString(set, &di, spdrpDeviceDesc)
			}
			desc = strings.TrimSpace(strings.Replace(desc, "("+name+")", "", 1))
			hw := devRegString(set, &di, spdrpHardwareID)
			vid, pid := parseVIDPID(hw)
			out = append(out, PortInfo{Name: name, Desc: desc, VID: vid, PID: pid, Chip: chipFromIDs(vid, pid)})
			seen[strings.ToUpper(name)] = true
		}
		procSetupDiDestroyDeviceInfoList.Call(set)
	}

	// Anything else the serial driver stack knows about (virtual ports etc).
	var hk syscall.Handle
	key, _ := syscall.UTF16PtrFromString(`HARDWARE\DEVICEMAP\SERIALCOMM`)
	if syscall.RegOpenKeyEx(syscall.HKEY_LOCAL_MACHINE, key, 0, syscall.KEY_READ, &hk) == nil {
		for i := uint32(0); ; i++ {
			nameBuf := make([]uint16, 256)
			nameLen := uint32(len(nameBuf))
			dataBuf := make([]uint16, 256)
			dataLen := uint32(len(dataBuf) * 2)
			var typ uint32
			r, _, _ := procRegEnumValueW.Call(uintptr(hk), uintptr(i), uintptr(unsafe.Pointer(&nameBuf[0])), uintptr(unsafe.Pointer(&nameLen)),
				0, uintptr(unsafe.Pointer(&typ)), uintptr(unsafe.Pointer(&dataBuf[0])), uintptr(unsafe.Pointer(&dataLen)))
			if r != 0 {
				break
			}
			port := syscall.UTF16ToString(dataBuf)
			if port == "" || seen[strings.ToUpper(port)] {
				continue
			}
			dev := syscall.UTF16ToString(nameBuf[:nameLen])
			out = append(out, PortInfo{Name: port, Desc: strings.TrimPrefix(dev, `\Device\`)})
			seen[strings.ToUpper(port)] = true
		}
		syscall.RegCloseKey(hk)
	}
	return out
}

func regQueryString(hk syscall.Handle, name string) string {
	n, _ := syscall.UTF16PtrFromString(name)
	buf := make([]uint16, 256)
	l := uint32(len(buf) * 2)
	var typ uint32
	if err := syscall.RegQueryValueEx(hk, n, nil, &typ, (*byte)(unsafe.Pointer(&buf[0])), &l); err != nil {
		return ""
	}
	return syscall.UTF16ToString(buf)
}

// usbDevicePaths returns the device-interface paths of every present USB device.
func usbDevicePaths() []string {
	set, _, _ := procSetupDiGetClassDevsW.Call(uintptr(unsafe.Pointer(&guidDevIfaceUSBDev)), 0, 0, digcfPresent|digcfDeviceInterface)
	if set == invalidHandleValue {
		return nil
	}
	defer procSetupDiDestroyDeviceInfoList.Call(set)
	var out []string
	for i := uint32(0); ; i++ {
		var id spDeviceInterfaceData
		id.CbSize = uint32(unsafe.Sizeof(id))
		r, _, _ := procSetupDiEnumDeviceInterfaces.Call(set, 0, uintptr(unsafe.Pointer(&guidDevIfaceUSBDev)), uintptr(i), uintptr(unsafe.Pointer(&id)))
		if r == 0 {
			break
		}
		var need uint32
		procSetupDiGetDeviceInterfaceDetailW.Call(set, uintptr(unsafe.Pointer(&id)), 0, 0, uintptr(unsafe.Pointer(&need)), 0)
		if need < 8 || need > 4096 {
			continue
		}
		buf := make([]byte, need+8)
		cb := uint32(8) // sizeof(SP_DEVICE_INTERFACE_DETAIL_DATA_W) on 64-bit
		if unsafe.Sizeof(uintptr(0)) == 4 {
			cb = 6
		}
		*(*uint32)(unsafe.Pointer(&buf[0])) = cb
		r, _, _ = procSetupDiGetDeviceInterfaceDetailW.Call(set, uintptr(unsafe.Pointer(&id)), uintptr(unsafe.Pointer(&buf[0])), uintptr(need), 0, 0)
		if r == 0 {
			continue
		}
		u16 := unsafe.Slice((*uint16)(unsafe.Pointer(&buf[4])), (need-4)/2)
		out = append(out, syscall.UTF16ToString(u16))
	}
	return out
}

func openFolder(dir string) {
	verb, _ := syscall.UTF16PtrFromString("open")
	p, _ := syscall.UTF16PtrFromString(dir)
	procShellExecuteW.Call(0, uintptr(unsafe.Pointer(verb)), uintptr(unsafe.Pointer(p)), 0, 0, 1)
}
