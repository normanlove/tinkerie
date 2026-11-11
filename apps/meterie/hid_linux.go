//go:build linux

package meterie

// hid_linux.go: USB-HID meter cables on Linux and Raspberry Pi OS through
// /dev/hidraw*. install-usb.sh lets the logged-in user open them.

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

type hidrawDev struct {
	f              *os.File
	numbered       bool // the device puts a report ID in front of its reports
	outLen, featLn int
}

// hidraw ioctls (linux/hidraw.h)
func hidIOC(dir, nr, size uintptr) uintptr { return dir<<30 | size<<16 | 'H'<<8 | nr }

const iocRead, iocWrite = 2, 1

func openHIDDev(path string) (hidHandle, error) {
	f, err := os.OpenFile(path, os.O_RDWR|syscall.O_NONBLOCK, 0)
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			return nil, fmt.Errorf("no permission for the USB cable: run install-usb.sh once (from Tinkerie's folder), then unplug and replug it")
		}
		return nil, fmt.Errorf("can't open the USB cable: %v", err)
	}
	d := &hidrawDev{f: f}
	d.numbered = reportDescNumbered(f)
	return d, nil
}

// reportDescNumbered reads the report descriptor and looks for Report ID items.
func reportDescNumbered(f *os.File) bool {
	var size int32
	rc, err := f.SyscallConn()
	if err != nil {
		return false
	}
	var desc struct {
		Size  uint32
		Value [4096]byte
	}
	rc.Control(func(fd uintptr) {
		if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, fd, hidIOC(iocRead, 0x01, 4), uintptr(unsafe.Pointer(&size))); e != 0 {
			return
		}
		desc.Size = uint32(min(size, 4096))
		syscall.Syscall(syscall.SYS_IOCTL, fd, hidIOC(iocRead, 0x02, unsafe.Sizeof(desc)), uintptr(unsafe.Pointer(&desc)))
	})
	return descHasReportID(desc.Value[:desc.Size])
}

// descHasReportID walks HID report descriptor items for a Report ID (0x85).
func descHasReportID(b []byte) bool {
	for i := 0; i < len(b); {
		p := b[i]
		if p == 0xFE { // long item
			if i+1 >= len(b) {
				break
			}
			i += 3 + int(b[i+1])
			continue
		}
		n := []int{0, 1, 2, 4}[p&3]
		if p&0xFC == 0x84 {
			return true
		}
		i += 1 + n
	}
	return false
}

func (d *hidrawDev) readReport(timeoutMs uint32) ([]byte, error) {
	buf := make([]byte, 256)
	d.f.SetReadDeadline(time.Now().Add(time.Duration(timeoutMs) * time.Millisecond))
	n, err := d.f.Read(buf)
	if err != nil {
		if errors.Is(err, os.ErrDeadlineExceeded) {
			return nil, nil
		}
		return nil, fmt.Errorf("USB read failed (cable unplugged?): %v", err)
	}
	if !d.numbered { // match Windows: report ID 0 in front
		return append([]byte{0}, buf[:n]...), nil
	}
	return buf[:n], nil
}

func (d *hidrawDev) writeReport(b []byte) error {
	buf := make([]byte, max(d.outLen, len(b)))
	copy(buf, b)
	d.f.SetWriteDeadline(time.Now().Add(time.Second))
	if _, err := d.f.Write(buf); err != nil {
		return fmt.Errorf("USB write: %v", err)
	}
	return nil
}

func (d *hidrawDev) setFeature(b []byte) error {
	buf := make([]byte, max(d.featLn, len(b)))
	copy(buf, b)
	rc, err := d.f.SyscallConn()
	if err != nil {
		return err
	}
	var e syscall.Errno
	rc.Control(func(fd uintptr) {
		_, _, e = syscall.Syscall(syscall.SYS_IOCTL, fd, hidIOC(iocWrite|iocRead, 0x06, uintptr(len(buf))), uintptr(unsafe.Pointer(&buf[0])))
	})
	if e != 0 {
		return fmt.Errorf("set feature report: %v", e)
	}
	return nil
}

func (d *hidrawDev) defaultLens(out, feat int) {
	if d.outLen == 0 {
		d.outLen = out
	}
	if d.featLn == 0 {
		d.featLn = feat
	}
}
func (d *hidrawDev) forceFeatLen(n int) { d.featLn = n }
func (d *hidrawDev) close()             { d.f.Close() }

// listHID finds known meter cables among /sys/class/hidraw.
func listHID() []PortInfo {
	var out []PortInfo
	nodes, _ := filepath.Glob("/sys/class/hidraw/hidraw*")
	for _, n := range nodes {
		f, err := os.Open(filepath.Join(n, "device", "uevent"))
		if err != nil {
			continue
		}
		var vid, pid uint16
		name := ""
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			k, v, _ := strings.Cut(sc.Text(), "=")
			switch k {
			case "HID_ID": // bus:vendor:product, e.g. 0003:00001A86:0000E008
				if p := strings.Split(v, ":"); len(p) == 3 {
					a, _ := strconv.ParseUint(p[1], 16, 32)
					b, _ := strconv.ParseUint(p[2], 16, 32)
					vid, pid = uint16(a), uint16(b)
				}
			case "HID_NAME":
				name = v
			}
		}
		f.Close()
		var kind Transport = -1
		for _, k := range []Transport{TrCH9325, TrCP2110, TrVictor, TrBU86X} {
			if u := lookupUSB(vid, pid, k); u != nil && u.Kind == k {
				kind = k
			}
		}
		if kind < 0 {
			continue
		}
		pi := PortInfo{Kind: kind, Path: "/dev/" + filepath.Base(n), Name: name, VID: vid, PID: pid}
		classifyPort(&pi)
		if name != "" {
			pi.Name = fmt.Sprintf("%s (%04X:%04X)", name, vid, pid)
		}
		out = append(out, pi)
	}
	return out
}
