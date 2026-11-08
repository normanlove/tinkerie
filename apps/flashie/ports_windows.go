//go:build windows

// Flashie — COM port discovery with USB VID/PID, read from the registry.
// Made by nrmluvdev.

package flashie

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

var (
	advapi32         = syscall.NewLazyDLL("advapi32.dll")
	procRegEnumValue = advapi32.NewProc("RegEnumValueW")
)

func regOpen(root syscall.Handle, path string) (syscall.Handle, error) {
	var h syscall.Handle
	p, _ := syscall.UTF16PtrFromString(path)
	err := syscall.RegOpenKeyEx(root, p, 0, syscall.KEY_READ, &h)
	return h, err
}

func regSubkeys(h syscall.Handle) []string {
	var out []string
	for i := uint32(0); ; i++ {
		buf := make([]uint16, 256)
		n := uint32(len(buf))
		if err := syscall.RegEnumKeyEx(h, i, &buf[0], &n, nil, nil, nil, nil); err != nil {
			break
		}
		out = append(out, syscall.UTF16ToString(buf[:n]))
	}
	return out
}

func regString(h syscall.Handle, name string) string {
	np, _ := syscall.UTF16PtrFromString(name)
	var typ uint32
	buf := make([]uint16, 512)
	n := uint32(len(buf) * 2)
	if err := syscall.RegQueryValueEx(h, np, nil, &typ, (*byte)(unsafe.Pointer(&buf[0])), &n); err != nil {
		return ""
	}
	if typ != syscall.REG_SZ && typ != syscall.REG_EXPAND_SZ {
		return ""
	}
	return syscall.UTF16ToString(buf[:n/2])
}

// activeComPorts lists ports Windows currently considers present.
func activeComPorts() map[string]string {
	out := map[string]string{}
	h, err := regOpen(syscall.HKEY_LOCAL_MACHINE, `HARDWARE\DEVICEMAP\SERIALCOMM`)
	if err != nil {
		return out
	}
	defer syscall.RegCloseKey(h)
	for i := uint32(0); ; i++ {
		name := make([]uint16, 256)
		nameLen := uint32(len(name))
		data := make([]uint16, 256)
		dataLen := uint32(len(data) * 2)
		var typ uint32
		r, _, _ := procRegEnumValue.Call(uintptr(h), uintptr(i),
			uintptr(unsafe.Pointer(&name[0])), uintptr(unsafe.Pointer(&nameLen)), 0,
			uintptr(unsafe.Pointer(&typ)), uintptr(unsafe.Pointer(&data[0])), uintptr(unsafe.Pointer(&dataLen)))
		if r != 0 {
			break
		}
		com := syscall.UTF16ToString(data[:dataLen/2])
		out[strings.ToUpper(com)] = syscall.UTF16ToString(name[:nameLen])
	}
	return out
}

var reVID = regexp.MustCompile(`(?i)VID_([0-9A-F]{4})`)
var rePID = regexp.MustCompile(`(?i)PID_([0-9A-F]{4})`)

type usbInfo struct {
	vid, pid uint16
	friendly string
	serial   string
}

// scanUSB walks Enum\USB and Enum\FTDIBUS for devices that own a COM port.
func scanUSB() map[string]usbInfo {
	out := map[string]usbInfo{}
	for _, root := range []string{`SYSTEM\CurrentControlSet\Enum\USB`, `SYSTEM\CurrentControlSet\Enum\FTDIBUS`} {
		rh, err := regOpen(syscall.HKEY_LOCAL_MACHINE, root)
		if err != nil {
			continue
		}
		for _, dev := range regSubkeys(rh) {
			vm, pm := reVID.FindStringSubmatch(dev), rePID.FindStringSubmatch(dev)
			if vm == nil || pm == nil {
				continue
			}
			vid, _ := strconv.ParseUint(vm[1], 16, 16)
			pid, _ := strconv.ParseUint(pm[1], 16, 16)
			dh, err := regOpen(rh, dev)
			if err != nil {
				continue
			}
			for _, inst := range regSubkeys(dh) {
				ih, err := regOpen(dh, inst)
				if err != nil {
					continue
				}
				friendly := regString(ih, "FriendlyName")
				// FTDIBUS nests one level deeper (…\0000\Device Parameters).
				candidates := []string{`Device Parameters`}
				for _, sub := range regSubkeys(ih) {
					if sub != "Device Parameters" && sub != "Properties" && sub != "LogConf" && sub != "Control" {
						candidates = append(candidates, sub+`\Device Parameters`)
					}
				}
				for _, c := range candidates {
					ph, err := regOpen(ih, c)
					if err != nil {
						continue
					}
					if port := strings.ToUpper(regString(ph, "PortName")); strings.HasPrefix(port, "COM") {
						out[port] = usbInfo{vid: uint16(vid), pid: uint16(pid), friendly: friendly, serial: inst}
					}
					syscall.RegCloseKey(ph)
				}
				syscall.RegCloseKey(ih)
			}
			syscall.RegCloseKey(dh)
		}
		syscall.RegCloseKey(rh)
	}
	return out
}

func listPorts() []PortInfo {
	active := activeComPorts()
	usb := scanUSB()
	var out []PortInfo
	for com, dev := range active {
		pi := PortInfo{Name: com, Device: dev}
		if u, ok := usb[com]; ok {
			pi.VID, pi.PID, pi.Friendly = u.vid, u.pid, u.friendly
			pi.USB = true
		}
		pi.Guess = guessBoard(pi)
		out = append(out, pi)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].USB != out[j].USB {
			return out[i].USB
		}
		return comNum(out[i].Name) < comNum(out[j].Name)
	})
	return out
}

func comNum(s string) int {
	n, _ := strconv.Atoi(strings.TrimPrefix(strings.ToUpper(s), "COM"))
	return n
}
