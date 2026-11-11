//go:build linux

package usb

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"tinkerie/shared/droid"
	"unsafe"
)

// ---------------------------------------------------------------- usbfs ABI

// Structs from linux/usbdevice_fs.h. Pointers are uintptr, so the sizes (and
// with them the ioctl numbers) come out right on 64-bit and 32-bit ARM alike.
type ctrlTransfer struct {
	RequestType, Request uint8
	Value, Index, Length uint16
	Timeout              uint32 // ms
	Data                 uintptr
}

type bulkTransfer struct {
	Ep, Len, Timeout uint32
	Data             uintptr
}

type urb struct {
	Type         uint8
	Endpoint     uint8
	Status       int32
	Flags        uint32
	Buffer       uintptr
	BufferLength int32
	ActualLength int32
	StartFrame   int32
	NumPackets   int32
	ErrorCount   int32
	Signr        uint32
	UserContext  uintptr
}

type usbIoctl struct {
	Ifno, IoctlCode int32
	Data            uintptr
}

func ioc(dir, nr, size uintptr) uintptr { return dir<<30 | size<<16 | 'U'<<8 | nr }

const (
	iocNone, iocWrite, iocRead = 0, 1, 2
	urbTypeBulk                = 3
)

var (
	reqControl    = ioc(iocRead|iocWrite, 0, unsafe.Sizeof(ctrlTransfer{}))
	reqBulk       = ioc(iocRead|iocWrite, 2, unsafe.Sizeof(bulkTransfer{}))
	reqSubmitURB  = ioc(iocRead, 10, unsafe.Sizeof(urb{}))
	reqDiscardURB = ioc(iocNone, 11, 0)
	reqReapURB    = ioc(iocWrite, 12, unsafe.Sizeof(uintptr(0)))
	reqClaim      = ioc(iocRead, 15, 4)
	reqRelease    = ioc(iocRead, 16, 4)
	reqIoctl      = ioc(iocRead|iocWrite, 18, unsafe.Sizeof(usbIoctl{}))
	reqClearHalt  = ioc(iocRead, 21, 4)
	reqDisconnect = ioc(iocNone, 22, 0) // detach the kernel driver (used inside reqIoctl)
	reqConnect    = ioc(iocNone, 23, 0) // give the interface back to the kernel driver
)

// ioctl retries when a signal (Go's scheduler uses them) interrupts the call.
func ioctl(fd int, req, arg uintptr) error {
	for {
		_, _, e := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), req, arg)
		if e == syscall.EINTR {
			continue
		}
		if e != 0 {
			return e
		}
		return nil
	}
}

// ------------------------------------------------------------------ listing

// List returns the USB devices on this computer, from sysfs.
func List() []Info {
	if droid.Active() {
		return bridgeList()
	}
	dirs, _ := filepath.Glob("/sys/bus/usb/devices/*")
	var out []Info
	for _, d := range dirs {
		name := filepath.Base(d)
		if strings.Contains(name, ":") || strings.HasPrefix(name, "usb") {
			continue // interfaces and root hubs
		}
		bus, _ := strconv.Atoi(read(d, "busnum"))
		dev, _ := strconv.Atoi(read(d, "devnum"))
		if bus == 0 || dev == 0 {
			continue
		}
		in := Info{
			Path:    fmt.Sprintf("/dev/bus/usb/%03d/%03d", bus, dev),
			SysPath: d,
			VID:     hex16(read(d, "idVendor")), PID: hex16(read(d, "idProduct")), BCD: hex16(read(d, "bcdDevice")),
			Manufacturer: read(d, "manufacturer"), Product: read(d, "product"), Serial: read(d, "serial"),
		}
		ifs, _ := filepath.Glob(d + "/" + name + ":*")
		for _, f := range ifs {
			it := Interface{Number: int(hex16(read(f, "bInterfaceNumber"))),
				Class: uint8(hex16(read(f, "bInterfaceClass"))), Sub: uint8(hex16(read(f, "bInterfaceSubClass")))}
			eps, _ := filepath.Glob(f + "/ep_*")
			for _, e := range eps {
				if read(e, "type") != "Bulk" {
					continue
				}
				a := uint8(hex16(read(e, "bEndpointAddress")))
				if a&0x80 != 0 {
					it.BulkIn = append(it.BulkIn, a)
				} else {
					it.BulkOut = append(it.BulkOut, a)
				}
			}
			in.Interfaces = append(in.Interfaces, it)
		}
		sort.Slice(in.Interfaces, func(i, j int) bool { return in.Interfaces[i].Number < in.Interfaces[j].Number })
		in.Accessible = syscall.Access(in.Path, 6 /* R_OK|W_OK */) == nil
		out = append(out, in)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

func read(dir, f string) string {
	b, err := os.ReadFile(filepath.Join(dir, f))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func hex16(s string) uint16 {
	n, _ := strconv.ParseUint(s, 16, 16)
	return uint16(n)
}

// ------------------------------------------------------------------- device

// Device is an open USB device with one claimed interface.
type Device struct {
	fd    int
	iface int
	extra []int
	Info  Info
	mu    sync.Mutex
}

// Open opens a device (Info.Path) and claims an interface, detaching any
// kernel driver that had it.
func Open(path string, iface int) (*Device, error) {
	var fd int
	var err error
	if droid.Active() {
		fd, err = bridgeOpen(path) // Android asks the person, then hands over the device
	} else {
		fd, err = syscall.Open(path, syscall.O_RDWR|syscall.O_CLOEXEC, 0)
	}
	if err != nil {
		if err == syscall.EACCES || err == syscall.EPERM {
			return nil, ErrNoPermission
		}
		if droid.Active() {
			return nil, err
		}
		return nil, fmt.Errorf("couldn't open USB device %s: %v", path, err)
	}
	d := &Device{fd: fd, iface: iface}
	for _, in := range List() {
		if in.Path == path {
			d.Info = in
		}
	}
	dc := usbIoctl{Ifno: int32(iface), IoctlCode: int32(reqDisconnect)}
	_ = ioctl(fd, reqIoctl, uintptr(unsafe.Pointer(&dc))) // ENODATA: no driver attached, which is fine
	n := uint32(iface)
	if err := ioctl(fd, reqClaim, uintptr(unsafe.Pointer(&n))); err != nil {
		syscall.Close(fd)
		if err == syscall.EBUSY {
			return nil, errors.New("this USB device is in use by another program")
		}
		return nil, fmt.Errorf("couldn't claim the USB interface: %v", err)
	}
	return d, nil
}

// ClaimExtra claims another interface of an open device too (a USB serial
// adapter's control interface, say), detaching any kernel driver from it.
func (d *Device) ClaimExtra(iface int) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if iface == d.iface {
		return nil
	}
	dc := usbIoctl{Ifno: int32(iface), IoctlCode: int32(reqDisconnect)}
	_ = ioctl(d.fd, reqIoctl, uintptr(unsafe.Pointer(&dc)))
	n := uint32(iface)
	if err := ioctl(d.fd, reqClaim, uintptr(unsafe.Pointer(&n))); err != nil {
		return fmt.Errorf("couldn't claim USB interface %d: %v", iface, err)
	}
	d.extra = append(d.extra, iface)
	return nil
}

// Interface returns the claimed interface's details.
func (d *Device) Interface() Interface {
	for _, it := range d.Info.Interfaces {
		if it.Number == d.iface {
			return it
		}
	}
	return Interface{Number: d.iface}
}

// Control does a control transfer. The direction comes from rt's top bit.
func (d *Device) Control(rt, req uint8, val, idx uint16, data []byte, timeoutMs int) (int, error) {
	c := ctrlTransfer{RequestType: rt, Request: req, Value: val, Index: idx, Length: uint16(len(data)), Timeout: uint32(timeoutMs)}
	if len(data) > 0 {
		c.Data = uintptr(unsafe.Pointer(&data[0]))
	}
	n, _, e := syscall.Syscall(syscall.SYS_IOCTL, uintptr(d.fd), reqControl, uintptr(unsafe.Pointer(&c)))
	for e == syscall.EINTR {
		n, _, e = syscall.Syscall(syscall.SYS_IOCTL, uintptr(d.fd), reqControl, uintptr(unsafe.Pointer(&c)))
	}
	runtime.KeepAlive(data)
	if e != 0 {
		return 0, fmt.Errorf("USB control transfer: %v", e)
	}
	return int(n), nil
}

func (d *Device) bulk(ep uint8, data []byte, timeoutMs int) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}
	b := bulkTransfer{Ep: uint32(ep), Len: uint32(len(data)), Timeout: uint32(timeoutMs), Data: uintptr(unsafe.Pointer(&data[0]))}
	n, _, e := syscall.Syscall(syscall.SYS_IOCTL, uintptr(d.fd), reqBulk, uintptr(unsafe.Pointer(&b)))
	for e == syscall.EINTR {
		n, _, e = syscall.Syscall(syscall.SYS_IOCTL, uintptr(d.fd), reqBulk, uintptr(unsafe.Pointer(&b)))
	}
	runtime.KeepAlive(data)
	if e == syscall.ETIMEDOUT {
		return 0, nil
	}
	if e != 0 {
		return 0, fmt.Errorf("USB bulk transfer: %v", e)
	}
	return int(n), nil
}

// BulkRead reads from an IN endpoint. It returns (0, nil) on timeout.
func (d *Device) BulkRead(ep uint8, buf []byte, timeoutMs int) (int, error) {
	return d.bulk(ep|0x80, buf, timeoutMs)
}

// BulkWrite writes to an OUT endpoint.
func (d *Device) BulkWrite(ep uint8, data []byte, timeoutMs int) (int, error) {
	return d.bulk(ep&0x7F, data, timeoutMs)
}

// ResetPipe clears a stalled endpoint.
func (d *Device) ResetPipe(ep uint8) {
	n := uint32(ep)
	_ = ioctl(d.fd, reqClearHalt, uintptr(unsafe.Pointer(&n)))
}

// Stream keeps count bulk reads of size bytes queued on an IN endpoint, so a
// fast device never waits for the PC, and calls fn with each completed read in
// order until fn returns false or stop is closed.
func (d *Device) Stream(ep uint8, size, count int, stop <-chan struct{}, fn func([]byte) bool) error {
	ep |= 0x80
	// Buffers and URBs live outside the Go heap: the kernel writes to them
	// while no Go code is looking.
	us := int(unsafe.Sizeof(urb{}))
	mem, err := syscall.Mmap(-1, 0, count*(size+us), syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_ANON|syscall.MAP_PRIVATE)
	if err != nil {
		return fmt.Errorf("USB stream buffers: %v", err)
	}
	defer syscall.Munmap(mem)
	urbs := make([]*urb, count)
	bufs := make([][]byte, count)
	for i := range urbs {
		urbs[i] = (*urb)(unsafe.Pointer(&mem[count*size+i*us]))
		bufs[i] = mem[i*size : (i+1)*size : (i+1)*size]
	}
	var mu sync.Mutex
	pending := make([]bool, count)
	stopping := false
	submit := func(i int) error {
		*urbs[i] = urb{Type: urbTypeBulk, Endpoint: ep, Buffer: uintptr(unsafe.Pointer(&bufs[i][0])), BufferLength: int32(size), UserContext: uintptr(i)}
		if err := ioctl(d.fd, reqSubmitURB, uintptr(unsafe.Pointer(urbs[i]))); err != nil {
			if err == syscall.ENOMEM {
				return fmt.Errorf("USB stream: the system's USB buffer limit is too small (raise /sys/module/usbcore/parameters/usbfs_memory_mb)")
			}
			return fmt.Errorf("USB stream: %v", err)
		}
		pending[i] = true
		return nil
	}
	cancel := func() { // caller holds mu
		stopping = true
		for i, p := range pending {
			if p {
				_ = ioctl(d.fd, reqDiscardURB, uintptr(unsafe.Pointer(urbs[i])))
			}
		}
	}
	mu.Lock()
	var firstErr error
	for i := range urbs {
		if err := submit(i); err != nil {
			firstErr = err
			cancel()
			break
		}
	}
	mu.Unlock()

	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-stop:
			mu.Lock()
			cancel()
			mu.Unlock()
		case <-done:
		}
	}()

	for {
		mu.Lock()
		left := 0
		for _, p := range pending {
			if p {
				left++
			}
		}
		mu.Unlock()
		if left == 0 {
			return firstErr
		}
		var got uintptr
		if err := ioctl(d.fd, reqReapURB, uintptr(unsafe.Pointer(&got))); err != nil {
			mu.Lock()
			if firstErr == nil && !stopping {
				firstErr = fmt.Errorf("USB stream: %v", err)
			}
			cancel()
			mu.Unlock()
			if err == syscall.ENODEV {
				return errors.New("the USB device was unplugged")
			}
			continue
		}
		u := (*urb)(unsafe.Pointer(got))
		i := int(u.UserContext)
		mu.Lock()
		pending[i] = false
		st, n, quit := u.Status, int(u.ActualLength), stopping
		mu.Unlock()
		if quit {
			continue // draining after stop
		}
		if st != 0 {
			mu.Lock()
			if firstErr == nil {
				firstErr = fmt.Errorf("USB stream read failed (status %d)", st)
			}
			cancel()
			mu.Unlock()
			continue
		}
		if n > 0 && !fn(bufs[i][:n]) {
			mu.Lock()
			cancel()
			mu.Unlock()
			continue
		}
		mu.Lock()
		if !stopping {
			if err := submit(i); err != nil {
				firstErr = err
				cancel()
			}
		}
		mu.Unlock()
	}
}

// Close releases the interface, gives it back to its kernel driver if it had
// one, and closes the device.
func (d *Device) Close() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.fd < 0 {
		return
	}
	for _, i := range append([]int{d.iface}, d.extra...) {
		n := uint32(i)
		_ = ioctl(d.fd, reqRelease, uintptr(unsafe.Pointer(&n)))
		c := usbIoctl{Ifno: int32(i), IoctlCode: int32(reqConnect)}
		_ = ioctl(d.fd, reqIoctl, uintptr(unsafe.Pointer(&c)))
	}
	syscall.Close(d.fd)
	d.fd = -1
}
