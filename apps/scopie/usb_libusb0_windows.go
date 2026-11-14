// libusb-win32 (libusb0.sys) support on Windows.
//
// Zadig, OpenHantek's older instructions and many older scope programs put
// the libusb-win32 driver on Hantek 6022 scopes instead of WinUSB. Those
// devices can't be opened through winusb.dll, so Scopie talks to them through
// libusb0.dll, which Zadig installs into System32 along with the driver.
// No cgo: the DLL is called through syscall like the rest of Scopie.
package scopie

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

var (
	lu0Once sync.Once
	lu0     *syscall.DLL
	lu0Err  error
	lu0Mu   sync.Mutex // the DLL's device list is global
	lu0P    = map[string]*syscall.Proc{}
)

var lu0Names = []string{
	"usb_init", "usb_find_busses", "usb_find_devices", "usb_get_busses", "usb_open", "usb_close",
	"usb_set_configuration", "usb_claim_interface", "usb_release_interface", "usb_control_msg",
	"usb_bulk_read", "usb_bulk_write", "usb_clear_halt", "usb_strerror",
	"usb_bulk_setup_async", "usb_submit_async", "usb_reap_async_nocancel", "usb_cancel_async", "usb_free_async",
}

func loadLibusb0() error {
	lu0Once.Do(func() {
		lu0, lu0Err = syscall.LoadDLL("libusb0.dll")
		if lu0Err != nil {
			return
		}
		for _, n := range lu0Names {
			p, err := lu0.FindProc(n)
			if err != nil {
				lu0Err = fmt.Errorf("libusb0.dll has no %s", n)
				return
			}
			lu0P[n] = p
		}
		lu0P["usb_init"].Call()
	})
	return lu0Err
}

func lu0Call(name string, a ...uintptr) int {
	r, _, _ := lu0P[name].Call(a...)
	return int(int32(r))
}

func lu0Error(code int) error {
	r, _, _ := lu0P["usb_strerror"].Call()
	msg := ""
	if r != 0 {
		msg = cString(r)
	}
	if msg == "" {
		msg = "error " + strconv.Itoa(code)
	}
	return errors.New("libusb-win32: " + msg)
}

func cString(p uintptr) string {
	var b []byte
	for i := 0; i < 1024; i++ {
		c := *(*byte)(cmem(p + uintptr(i)))
		if c == 0 {
			break
		}
		b = append(b, c)
	}
	return string(b)
}

// Offsets into libusb-win32's structs (lusb0_usb.h, LIBUSB_PATH_MAX = 512).
// Only fields before any packing-sensitive member are read.
const (
	lu0PathMax       = 512
	lu0BusDevices    = 16 + lu0PathMax     // struct usb_bus.devices
	lu0DevDescriptor = 16 + lu0PathMax + 8 // struct usb_device.descriptor
)

// libusb0Path is how a libusb-win32 device is named in usbInfo.Path.
func libusb0Path(vid, pid uint16, n int) string {
	return fmt.Sprintf("libusb0|%04X|%04X|%d", vid, pid, n)
}

// findLibusb0 returns the nth device with this VID:PID in libusb0's list.
func findLibusb0(vid, pid uint16, n int) uintptr {
	lu0Call("usb_find_busses")
	lu0Call("usb_find_devices")
	seen := 0
	for bus := uintptr(lu0Call0("usb_get_busses")); bus != 0; bus = *(*uintptr)(cmem(bus)) {
		for dev := *(*uintptr)(cmem(bus + lu0BusDevices)); dev != 0; dev = *(*uintptr)(cmem(dev)) {
			d := dev + lu0DevDescriptor
			if *(*uint16)(cmem(d + 8)) == vid && *(*uint16)(cmem(d + 10)) == pid {
				if seen == n {
					return dev
				}
				seen++
			}
		}
	}
	return 0
}

// cmem turns an address inside libusb0.dll's own memory into a pointer.
func cmem(p uintptr) unsafe.Pointer { return unsafe.Add(nil, p) }

func lu0Call0(name string) uintptr { r, _, _ := lu0P[name].Call(); return r }

type lu0Dev struct {
	h          uintptr
	in, out    []uint8
	class, sub uint8
}

func openLibusb0(path string) (usbDevice, error) {
	if err := loadLibusb0(); err != nil {
		return nil, &driverError{"the libusb-win32 driver is attached but libusb0.dll couldn't be loaded (" + err.Error() + ")"}
	}
	f := strings.Split(path, "|")
	if len(f) != 4 {
		return nil, errors.New("bad libusb-win32 device name")
	}
	vid, _ := strconv.ParseUint(f[1], 16, 16)
	pid, _ := strconv.ParseUint(f[2], 16, 16)
	n, _ := strconv.Atoi(f[3])
	lu0Mu.Lock()
	defer lu0Mu.Unlock()
	dev := findLibusb0(uint16(vid), uint16(pid), n)
	if dev == 0 {
		return nil, fmt.Errorf("libusb-win32 doesn't list USB %s:%s", f[1], f[2])
	}
	h := uintptr(lu0Call0Arg("usb_open", dev))
	if h == 0 {
		return nil, lu0Error(-1)
	}
	d := &lu0Dev{h: h}
	if r := lu0Call("usb_set_configuration", h, 1); r < 0 {
		// Not fatal: already configured devices may refuse a repeat.
		_ = r
	}
	if r := lu0Call("usb_claim_interface", h, 0); r < 0 {
		err := lu0Error(r)
		lu0Call("usb_close", h)
		return nil, fmt.Errorf("%v — is another oscilloscope program using the scope?", err)
	}
	d.readEndpoints()
	return d, nil
}

func lu0Call0Arg(name string, a uintptr) uintptr { r, _, _ := lu0P[name].Call(a); return r }

// readEndpoints reads the configuration descriptor for interface 0's bulk endpoints.
func (d *lu0Dev) readEndpoints() {
	buf := make([]byte, 512)
	n, err := d.Control(usbStdIn, 6, 0x0200, 0, buf)
	if err != nil || n < 9 {
		return
	}
	buf = buf[:n]
	inIface0 := false
	for i := 0; i+2 <= len(buf) && buf[i] >= 2; i += int(buf[i]) {
		l, t := int(buf[i]), buf[i+1]
		if i+l > len(buf) {
			break
		}
		switch t {
		case 4: // interface
			inIface0 = l >= 9 && buf[i+2] == 0 && buf[i+3] == 0
			if inIface0 {
				d.class, d.sub = buf[i+5], buf[i+6]
			}
		case 5: // endpoint
			if inIface0 && l >= 7 && buf[i+3]&3 == 2 {
				if ep := buf[i+2]; ep&0x80 != 0 {
					d.in = append(d.in, ep)
				} else {
					d.out = append(d.out, ep)
				}
			}
		}
	}
}

func (d *lu0Dev) Endpoints() ([]uint8, []uint8, uint8, uint8) { return d.in, d.out, d.class, d.sub }

func (d *lu0Dev) Close() {
	if d.h != 0 {
		lu0Call("usb_release_interface", d.h, 0)
		lu0Call("usb_close", d.h)
		d.h = 0
	}
}

func bufPtr(b []byte) uintptr {
	if len(b) == 0 {
		return 0
	}
	return uintptr(unsafe.Pointer(&b[0]))
}

func (d *lu0Dev) Control(rt, req uint8, val, idx uint16, data []byte) (int, error) {
	if d.h == 0 {
		return 0, errors.New("device closed")
	}
	r := lu0Call("usb_control_msg", d.h, uintptr(rt), uintptr(req), uintptr(val), uintptr(idx), bufPtr(data), uintptr(len(data)), 2000)
	if r < 0 {
		return 0, fmt.Errorf("control transfer 0x%02x: %v", req, lu0Error(r))
	}
	return r, nil
}

func (d *lu0Dev) ResetPipe(ep uint8) {
	if d.h != 0 {
		lu0Call("usb_clear_halt", d.h, uintptr(ep))
	}
}

func (d *lu0Dev) bulk(name string, ep uint8, buf []byte, timeoutMs int) (int, error) {
	if d.h == 0 {
		return 0, errors.New("device closed")
	}
	if len(buf) == 0 {
		return 0, nil
	}
	if timeoutMs <= 0 {
		timeoutMs = 2000
	}
	r := lu0Call(name, d.h, uintptr(ep), bufPtr(buf), uintptr(len(buf)), uintptr(timeoutMs))
	if r < 0 {
		if r == -116 { // ETIMEDOUT
			return 0, errors.New("timed out")
		}
		return 0, fmt.Errorf("bulk transfer: %v", lu0Error(r))
	}
	return r, nil
}

func (d *lu0Dev) BulkWrite(ep uint8, data []byte, timeoutMs int) (int, error) {
	return d.bulk("usb_bulk_write", ep, data, timeoutMs)
}

func (d *lu0Dev) BulkRead(ep uint8, buf []byte, timeoutMs int) (int, error) {
	return d.bulk("usb_bulk_read", ep, buf, timeoutMs)
}

// Stream keeps count reads queued with libusb-win32's async calls.
func (d *lu0Dev) Stream(ep uint8, size, count int, stop <-chan struct{}, fn func([]byte) bool) error {
	type xfer struct {
		ctx uintptr
		buf []byte
	}
	xs := make([]*xfer, count)
	defer func() {
		for _, x := range xs {
			if x != nil && x.ctx != 0 {
				lu0Call("usb_cancel_async", x.ctx)
				lu0Call("usb_free_async", uintptr(unsafe.Pointer(&x.ctx)))
			}
		}
	}()
	for i := range xs {
		x := &xfer{buf: make([]byte, size)}
		xs[i] = x
		if r := lu0Call("usb_bulk_setup_async", d.h, uintptr(unsafe.Pointer(&x.ctx)), uintptr(ep)); r < 0 {
			return fmt.Errorf("bulk read setup: %v", lu0Error(r))
		}
		if r := lu0Call("usb_submit_async", x.ctx, bufPtr(x.buf), uintptr(size)); r < 0 {
			return fmt.Errorf("bulk read: %v", lu0Error(r))
		}
	}
	for i := 0; ; i = (i + 1) % count {
		x := xs[i]
		var n int
		for {
			n = lu0Call("usb_reap_async_nocancel", x.ctx, 100)
			if n != -116 { // not a timeout: done or failed
				break
			}
			select {
			case <-stop:
				return nil
			default:
			}
		}
		if n < 0 {
			select {
			case <-stop:
				return nil
			default:
			}
			return fmt.Errorf("bulk read failed: %v", lu0Error(n))
		}
		if n > 0 && !fn(x.buf[:n]) {
			return nil
		}
		select {
		case <-stop:
			return nil
		default:
		}
		if r := lu0Call("usb_submit_async", x.ctx, bufPtr(x.buf), uintptr(size)); r < 0 {
			return fmt.Errorf("bulk read: %v", lu0Error(r))
		}
	}
}
