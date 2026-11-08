package flashie

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

var (
	modk32            = syscall.NewLazyDLL("kernel32.dll")
	procDeviceIoCtl   = modk32.NewProc("DeviceIoControl")
	procFindFirstVol  = modk32.NewProc("FindFirstVolumeW")
	procFindNextVol   = modk32.NewProc("FindNextVolumeW")
	procFindVolClose  = modk32.NewProc("FindVolumeClose")
	procVolPathNames  = modk32.NewProc("GetVolumePathNamesForVolumeNameW")
	procGetWinDir     = modk32.NewProc("GetWindowsDirectoryW")
	modshell32        = syscall.NewLazyDLL("shell32.dll")
	procShellExecuteW = modshell32.NewProc("ShellExecuteW")
)

const (
	ioctlStorageQueryProperty = 0x2D1400
	ioctlDiskGeometryEx       = 0x700A0
	ioctlStorageDeviceNumber  = 0x2D1080
	ioctlVolumeDiskExtents    = 0x560000
	ioctlDiskUpdateProps      = 0x70140
	fsctlLockVolume           = 0x90018
	fsctlDismountVolume       = 0x90020
	fsctlAllowExtendedIO      = 0x90083
	fileFlagNoBuffering       = 0x20000000
	fileFlagWriteThrough      = 0x80000000
)

func ioctl(h syscall.Handle, code uint32, in []byte, out []byte) (int, error) {
	var ret uint32
	var ip, op unsafe.Pointer
	if len(in) > 0 {
		ip = unsafe.Pointer(&in[0])
	}
	if len(out) > 0 {
		op = unsafe.Pointer(&out[0])
	}
	r, _, e := procDeviceIoCtl.Call(uintptr(h), uintptr(code), uintptr(ip), uintptr(len(in)), uintptr(op), uintptr(len(out)), uintptr(unsafe.Pointer(&ret)), 0)
	if r == 0 {
		return 0, e
	}
	return int(ret), nil
}

func openDev(path string, access uint32, flags uint32) (syscall.Handle, error) {
	p, _ := syscall.UTF16PtrFromString(path)
	return syscall.CreateFile(p, access, syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE, nil, syscall.OPEN_EXISTING, flags, 0)
}

func cstrAt(b []byte, off uint32) string {
	if off == 0 || int(off) >= len(b) {
		return ""
	}
	e := int(off)
	for e < len(b) && b[e] != 0 {
		e++
	}
	return strings.TrimSpace(string(b[off:e]))
}

type volInfo struct {
	guid  string // \\?\Volume{...}\
	disks []uint32
	paths []string
}

func volumes() []volInfo {
	var out []volInfo
	buf := make([]uint16, 300)
	h, _, _ := procFindFirstVol.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if syscall.Handle(h) == syscall.InvalidHandle {
		return nil
	}
	defer procFindVolClose.Call(h)
	for {
		name := syscall.UTF16ToString(buf)
		v := volInfo{guid: name}
		if vh, err := openDev(strings.TrimSuffix(name, `\`), 0, 0); err == nil {
			ext := make([]byte, 8+24*16)
			if n, err := ioctl(vh, ioctlVolumeDiskExtents, nil, ext); err == nil && n >= 8 {
				cnt := binary.LittleEndian.Uint32(ext)
				for i := uint32(0); i < cnt && 8+24*(i+1) <= uint32(n); i++ {
					v.disks = append(v.disks, binary.LittleEndian.Uint32(ext[8+24*i:]))
				}
			}
			syscall.CloseHandle(vh)
		}
		pn := make([]uint16, 1024)
		var ret uint32
		np, _ := syscall.UTF16PtrFromString(name)
		if r, _, _ := procVolPathNames.Call(uintptr(unsafe.Pointer(np)), uintptr(unsafe.Pointer(&pn[0])), uintptr(len(pn)), uintptr(unsafe.Pointer(&ret))); r != 0 {
			start := 0
			for i, c := range pn[:ret] {
				if c == 0 {
					if i > start {
						v.paths = append(v.paths, strings.TrimSuffix(syscall.UTF16ToString(pn[start:i]), `\`))
					}
					start = i + 1
				}
			}
		}
		out = append(out, v)
		if r, _, _ := procFindNextVol.Call(h, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf))); r == 0 {
			break
		}
	}
	return out
}

func systemDiskNumbers() map[uint32]bool {
	out := map[uint32]bool{}
	buf := make([]uint16, 260)
	n, _, _ := procGetWinDir.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	letters := []string{"C"}
	if n >= 2 {
		letters = append(letters, strings.ToUpper(syscall.UTF16ToString(buf[:1])))
	}
	for _, l := range letters {
		h, err := openDev(`\\.\`+l+":", 0, 0)
		if err != nil {
			continue
		}
		ext := make([]byte, 8+24*16)
		if n, err := ioctl(h, ioctlVolumeDiskExtents, nil, ext); err == nil && n >= 8 {
			for i := uint32(0); i < binary.LittleEndian.Uint32(ext) && 8+24*(i+1) <= uint32(n); i++ {
				out[binary.LittleEndian.Uint32(ext[8+24*i:])] = true
			}
		}
		syscall.CloseHandle(h)
	}
	return out
}

func listDisksOS() ([]Disk, error) {
	sys := systemDiskNumbers()
	vols := volumes()
	var out []Disk
	for i := uint32(0); i < 64; i++ {
		if sys[i] {
			continue
		}
		path := fmt.Sprintf(`\\.\PhysicalDrive%d`, i)
		h, err := openDev(path, 0, 0)
		if err != nil {
			continue
		}
		q := make([]byte, 12) // StorageDeviceProperty, PropertyStandardQuery
		desc := make([]byte, 1024)
		n, err := ioctl(h, ioctlStorageQueryProperty, q, desc)
		geo := make([]byte, 256)
		gn, gerr := ioctl(h, ioctlDiskGeometryEx, nil, geo)
		syscall.CloseHandle(h)
		if err != nil || n < 32 || gerr != nil || gn < 32 {
			continue // no media in the reader, or not a disk
		}
		removable := desc[10] != 0
		bus := binary.LittleEndian.Uint32(desc[28:])
		var b string
		switch bus {
		case 7:
			b = "USB"
		case 0xC, 0xD:
			b = "SD card"
		default:
			if !removable {
				continue
			}
			b = "Removable"
		}
		size := int64(binary.LittleEndian.Uint64(geo[24:]))
		if size == 0 {
			continue
		}
		name := strings.TrimSpace(cstrAt(desc[:n], binary.LittleEndian.Uint32(desc[12:])) + " " + cstrAt(desc[:n], binary.LittleEndian.Uint32(desc[16:])))
		if name == "" {
			name = b + " drive"
		}
		d := Disk{ID: path, Name: name, Size: size, Bus: b}
		for _, v := range vols {
			for _, dn := range v.disks {
				if dn == i {
					d.Mounts = append(d.Mounts, v.paths...)
				}
			}
		}
		out = append(out, d)
	}
	return out, nil
}

type winDisk struct {
	*os.File
	locks []syscall.Handle
}

func (w *winDisk) Close() error {
	err := w.File.Close()
	for _, h := range w.locks {
		syscall.CloseHandle(h)
	}
	return err
}

func diskNumber(id string) (uint32, error) {
	var n uint32
	_, err := fmt.Sscanf(id, `\\.\PhysicalDrive%d`, &n)
	return n, err
}

func openDiskForWrite(d Disk) (rawDisk, error) {
	if t, ok := testDiskFile(d); ok {
		return t, nil
	}
	num, err := diskNumber(d.ID)
	if err != nil {
		return nil, err
	}
	w := &winDisk{}
	// Lock and dismount every volume on the drive, and keep them locked
	// while writing, or Windows refuses writes inside them.
	for _, v := range volumes() {
		on := false
		for _, dn := range v.disks {
			on = on || dn == num
		}
		if !on {
			continue
		}
		vh, err := openDev(strings.TrimSuffix(v.guid, `\`), syscall.GENERIC_READ|syscall.GENERIC_WRITE, 0)
		if err != nil {
			continue
		}
		locked := false
		for try := 0; try < 20 && !locked; try++ {
			_, err := ioctl(vh, fsctlLockVolume, nil, nil)
			locked = err == nil
			if !locked {
				sleepMS(250)
			}
		}
		if !locked {
			syscall.CloseHandle(vh)
			for _, h := range w.locks {
				syscall.CloseHandle(h)
			}
			return nil, fmt.Errorf("the drive is in use (%s) — close any Explorer windows or programs using it and try again", strings.Join(v.paths, ", "))
		}
		ioctl(vh, fsctlDismountVolume, nil, nil)
		w.locks = append(w.locks, vh)
	}
	h, err := openDev(d.ID, syscall.GENERIC_READ|syscall.GENERIC_WRITE, fileFlagNoBuffering|fileFlagWriteThrough)
	if err != nil {
		for _, l := range w.locks {
			syscall.CloseHandle(l)
		}
		if errors.Is(err, syscall.ERROR_ACCESS_DENIED) {
			return nil, errors.New("Windows wouldn't let Flashie open the drive — if it's an SD card, slide its lock switch up, away from LOCK")
		}
		return nil, fmt.Errorf("couldn't open the drive: %v", err)
	}
	ioctl(h, fsctlAllowExtendedIO, nil, nil)
	w.File = os.NewFile(uintptr(h), d.ID)
	return w, nil
}

func finishDisk(d Disk, eject bool) {
	if _, ok := testDiskFile(d); ok {
		return
	}
	if h, err := openDev(d.ID, syscall.GENERIC_READ|syscall.GENERIC_WRITE, 0); err == nil {
		ioctl(h, ioctlDiskUpdateProps, nil, nil) // let Windows see the new partitions
		syscall.CloseHandle(h)
	}
}

func canWriteDirect(d Disk) bool {
	if _, ok := testDiskFile(d); ok {
		return true
	}
	h, err := openDev(d.ID, syscall.GENERIC_READ|syscall.GENERIC_WRITE, 0)
	if err != nil {
		return false
	}
	syscall.CloseHandle(h)
	return true
}

func elevate(base, token string) (chan error, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	verb, _ := syscall.UTF16PtrFromString("runas")
	file, _ := syscall.UTF16PtrFromString(exe)
	params, _ := syscall.UTF16PtrFromString(`--app flashie --img-helper "` + base + `" ` + token)
	r, _, _ := procShellExecuteW.Call(0, uintptr(unsafe.Pointer(verb)), uintptr(unsafe.Pointer(file)), uintptr(unsafe.Pointer(params)), 0, 0)
	if r <= 32 {
		if r == 5 {
			return nil, errors.New("the administrator prompt was cancelled")
		}
		return nil, fmt.Errorf("couldn't start the writer (error %d)", r)
	}
	return nil, nil // no process handle; Flashie watches the helper's check-ins instead
}
