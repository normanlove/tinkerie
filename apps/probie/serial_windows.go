//go:build windows

package probie

import (
	"errors"
	"fmt"
	"sync"
	"syscall"
	"unsafe"
)

var (
	procSetupComm          = modKernel.NewProc("SetupComm")
	procGetCommState       = modKernel.NewProc("GetCommState")
	procSetCommState       = modKernel.NewProc("SetCommState")
	procSetCommTimeouts    = modKernel.NewProc("SetCommTimeouts")
	procEscapeCommFunction = modKernel.NewProc("EscapeCommFunction")
	procPurgeComm          = modKernel.NewProc("PurgeComm")
	procCancelIoEx         = modKernel.NewProc("CancelIoEx")
	procClearCommError     = modKernel.NewProc("ClearCommError")
)

type winDCB struct {
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

type winCommTimeouts struct {
	ReadIntervalTimeout         uint32
	ReadTotalTimeoutMultiplier  uint32
	ReadTotalTimeoutConstant    uint32
	WriteTotalTimeoutMultiplier uint32
	WriteTotalTimeoutConstant   uint32
}

const (
	escSetRTS = 3
	escClrRTS = 4
	escSetDTR = 5
	escClrDTR = 6
)

type winPort struct {
	h      syscall.Handle
	mu     sync.Mutex
	closed bool
}

func openPort(name string, cfg PortConfig) (Port, error) {
	path, err := syscall.UTF16PtrFromString(`\\.\` + name)
	if err != nil {
		return nil, err
	}
	h, err := syscall.CreateFile(path, syscall.GENERIC_READ|syscall.GENERIC_WRITE, 0, nil, syscall.OPEN_EXISTING, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", name, err)
	}
	fail := func(what string, e error) (Port, error) {
		syscall.CloseHandle(h)
		return nil, fmt.Errorf("%s on %s: %v", what, name, e)
	}
	procSetupComm.Call(uintptr(h), 1<<16, 1<<16)

	var d winDCB
	d.DCBlength = uint32(unsafe.Sizeof(d))
	if r, _, e := procGetCommState.Call(uintptr(h), uintptr(unsafe.Pointer(&d))); r == 0 {
		return fail("GetCommState", e)
	}
	d.BaudRate = uint32(cfg.Baud)
	flags := uint32(1) // fBinary
	switch cfg.Parity {
	case "odd":
		d.Parity = 1
		flags |= 1 << 1
	case "even":
		d.Parity = 2
		flags |= 1 << 1
	case "mark":
		d.Parity = 3
		flags |= 1 << 1
	case "space":
		d.Parity = 4
		flags |= 1 << 1
	default:
		d.Parity = 0
	}
	if cfg.DTR {
		flags |= 1 << 4 // DTR_CONTROL_ENABLE
	}
	if cfg.RTS {
		flags |= 1 << 12 // RTS_CONTROL_ENABLE
	}
	d.Flags = flags
	d.ByteSize = byte(clampInt(cfg.DataBits, 5, 8))
	switch cfg.StopBits {
	case 1.5:
		d.StopBits = 1
	case 2:
		d.StopBits = 2
	default:
		d.StopBits = 0
	}
	d.XonLim, d.XoffLim, d.XonChar, d.XoffChar = 2048, 512, 0x11, 0x13
	if r, _, e := procSetCommState.Call(uintptr(h), uintptr(unsafe.Pointer(&d))); r == 0 {
		return fail("baud rate / format not supported", e)
	}
	// Return as soon as anything arrives, otherwise wake every 50 ms.
	to := winCommTimeouts{0xFFFFFFFF, 0xFFFFFFFF, 50, 0, 2000}
	if r, _, e := procSetCommTimeouts.Call(uintptr(h), uintptr(unsafe.Pointer(&to))); r == 0 {
		return fail("SetCommTimeouts", e)
	}
	procPurgeComm.Call(uintptr(h), 0x0008|0x0004)
	return &winPort{h: h}, nil
}

func (p *winPort) Read(b []byte) (int, error) {
	for {
		p.mu.Lock()
		closed := p.closed
		p.mu.Unlock()
		if closed {
			return 0, errors.New("closed")
		}
		var n uint32
		err := syscall.ReadFile(p.h, b, &n, nil)
		if err != nil {
			// Clear line errors (framing/overrun) so the port keeps working.
			var errs uint32
			procClearCommError.Call(uintptr(p.h), uintptr(unsafe.Pointer(&errs)), 0)
			if err == syscall.Errno(995) { // operation aborted
				return 0, errors.New("closed")
			}
			return int(n), err
		}
		if n > 0 {
			return int(n), nil
		}
	}
}

func (p *winPort) Write(b []byte) (int, error) {
	var n uint32
	err := syscall.WriteFile(p.h, b, &n, nil)
	return int(n), err
}

func (p *winPort) esc(f uintptr) error {
	if r, _, e := procEscapeCommFunction.Call(uintptr(p.h), f); r == 0 {
		return e
	}
	return nil
}

func (p *winPort) SetDTR(v bool) error {
	if v {
		return p.esc(escSetDTR)
	}
	return p.esc(escClrDTR)
}

func (p *winPort) SetRTS(v bool) error {
	if v {
		return p.esc(escSetRTS)
	}
	return p.esc(escClrRTS)
}

func (p *winPort) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	p.mu.Unlock()
	procCancelIoEx.Call(uintptr(p.h), 0)
	return syscall.CloseHandle(p.h)
}
