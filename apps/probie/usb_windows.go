//go:build windows

package probie

import (
	"fmt"
	"strconv"
	"syscall"
	"unsafe"
)

var (
	modWinusb = syscall.NewLazyDLL("winusb.dll")

	procWinUsbInitialize          = modWinusb.NewProc("WinUsb_Initialize")
	procWinUsbFree                = modWinusb.NewProc("WinUsb_Free")
	procWinUsbControlTransfer     = modWinusb.NewProc("WinUsb_ControlTransfer")
	procWinUsbReadPipe            = modWinusb.NewProc("WinUsb_ReadPipe")
	procWinUsbAbortPipe           = modWinusb.NewProc("WinUsb_AbortPipe")
	procWinUsbResetPipe           = modWinusb.NewProc("WinUsb_ResetPipe")
	procWinUsbFlushPipe           = modWinusb.NewProc("WinUsb_FlushPipe")
	procWinUsbSetPipePolicy       = modWinusb.NewProc("WinUsb_SetPipePolicy")
	procWinUsbGetOverlappedResult = modWinusb.NewProc("WinUsb_GetOverlappedResult")

	procCreateEventW = modKernel.NewProc("CreateEventW")
	procResetEvent   = modKernel.NewProc("ResetEvent")
)

const (
	fileFlagOverlapped  = 0x40000000
	errIOPending        = syscall.Errno(997)
	pipeTransferTimeout = 0x03
)

type winUSB struct {
	h    syscall.Handle
	w    uintptr
	path string
}

func listUSBDevices() []usbInfo {
	var out []usbInfo
	for _, p := range usbDevicePaths() {
		v, pi := parseVIDPID(p)
		vid, e1 := strconv.ParseUint(v, 16, 16)
		pid, e2 := strconv.ParseUint(pi, 16, 16)
		if e1 != nil || e2 != nil {
			continue
		}
		out = append(out, usbInfo{Path: p, VID: uint16(vid), PID: uint16(pid)})
	}
	return out
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
	to := uint32(1500)
	procWinUsbSetPipePolicy.Call(w, 0, pipeTransferTimeout, 4, uintptr(unsafe.Pointer(&to)))
	return d, nil
}

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

type winXfer struct {
	buf     []byte
	ov      *syscall.Overlapped
	pending bool
}

// Stream keeps `count` bulk reads of `size` bytes queued on endpoint ep and
// hands each completed buffer to fn (which must copy what it keeps).
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
			if e == syscall.Errno(995) {
				return nil
			}
			return fmt.Errorf("bulk read failed: %v", e)
		}
		if !fn(x.buf[:n]) {
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
