//go:build windows

// Flashie — serial port access via the Win32 API (kernel32), no cgo, no DLLs.
// Made by nrmluvdev.

package flashie

import (
	"errors"
	"fmt"
	"syscall"
	"time"
	"unsafe"
)

var (
	kernel32            = syscall.NewLazyDLL("kernel32.dll")
	procGetCommState    = kernel32.NewProc("GetCommState")
	procSetCommState    = kernel32.NewProc("SetCommState")
	procSetCommTimeouts = kernel32.NewProc("SetCommTimeouts")
	procEscapeCommFunc  = kernel32.NewProc("EscapeCommFunction")
	procPurgeComm       = kernel32.NewProc("PurgeComm")
	procSetupComm       = kernel32.NewProc("SetupComm")
	procClearCommError  = kernel32.NewProc("ClearCommError")
)

type dcb struct {
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
	ReadIntervalTimeout         uint32
	ReadTotalTimeoutMultiplier  uint32
	ReadTotalTimeoutConstant    uint32
	WriteTotalTimeoutMultiplier uint32
	WriteTotalTimeoutConstant   uint32
}

const (
	setRTS = 3
	clrRTS = 4
	setDTR = 5
	clrDTR = 6

	purgeTxClear = 0x0004
	purgeRxClear = 0x0008

	maxDWORD = 0xFFFFFFFF
)

type winPort struct {
	h      syscall.Handle
	name   string
	parity bool
}

func openSerial(name string, baud int) (Port, error) { return openSerialEx(name, baud, false) }

// openSerialEx opens a port as 8N1, or 8E1 when evenParity is set (STM32).
func openSerialEx(name string, baud int, evenParity bool) (Port, error) {
	path, err := syscall.UTF16PtrFromString(`\\.\` + name)
	if err != nil {
		return nil, err
	}
	h, err := syscall.CreateFile(path, syscall.GENERIC_READ|syscall.GENERIC_WRITE, 0, nil,
		syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		var en syscall.Errno
		if errors.As(err, &en) {
			switch en {
			case 5: // ERROR_ACCESS_DENIED
				return nil, fmt.Errorf("%s is busy — another program has it open (Arduino IDE serial monitor, PuTTY, Cura…). Close it and try again", name)
			case 2, 3: // not found
				return nil, fmt.Errorf("%s has disappeared — was the board unplugged?", name)
			}
		}
		return nil, fmt.Errorf("couldn't open %s: %v", name, err)
	}
	p := &winPort{h: h, name: name, parity: evenParity}
	procSetupComm.Call(uintptr(h), 64*1024, 64*1024)
	if err := p.configure(baud); err != nil {
		syscall.CloseHandle(h)
		return nil, err
	}
	to := commTimeouts{
		ReadIntervalTimeout:        maxDWORD,
		ReadTotalTimeoutMultiplier: maxDWORD,
		ReadTotalTimeoutConstant:   20, // return as soon as bytes arrive, or after 20 ms
		WriteTotalTimeoutConstant:  5000,
	}
	if r, _, e := procSetCommTimeouts.Call(uintptr(h), uintptr(unsafe.Pointer(&to))); r == 0 {
		syscall.CloseHandle(h)
		return nil, fmt.Errorf("SetCommTimeouts: %v", e)
	}
	p.SetDTR(false)
	p.SetRTS(false)
	p.Purge()
	return p, nil
}

func (p *winPort) configure(baud int) error {
	var d dcb
	d.DCBlength = uint32(unsafe.Sizeof(d))
	if r, _, e := procGetCommState.Call(uintptr(p.h), uintptr(unsafe.Pointer(&d))); r == 0 {
		return fmt.Errorf("GetCommState: %v", e)
	}
	d.BaudRate = uint32(baud)
	d.ByteSize = 8
	d.Parity = 0   // NOPARITY
	d.StopBits = 0 // ONESTOPBIT
	// fBinary=1; DTR/RTS control disabled (we drive them by hand); no flow control.
	d.Flags = 0x0001
	if p.parity {
		d.Parity = 2      // EVENPARITY
		d.Flags |= 0x0002 // fParity
	}
	if r, _, e := procSetCommState.Call(uintptr(p.h), uintptr(unsafe.Pointer(&d))); r == 0 {
		return fmt.Errorf("couldn't set %s to %d baud: %v", p.name, baud, e)
	}
	return nil
}

func (p *winPort) Write(b []byte) (int, error) {
	var n uint32
	err := syscall.WriteFile(p.h, b, &n, nil)
	if err != nil {
		return int(n), fmt.Errorf("write to %s failed (board unplugged?): %v", p.name, err)
	}
	if n == 0 && len(b) > 0 {
		return 0, fmt.Errorf("write to %s timed out", p.name)
	}
	return int(n), nil
}

func (p *winPort) ReadTimeout(b []byte, d time.Duration) (int, error) {
	deadline := time.Now().Add(d)
	for {
		var n uint32
		if err := syscall.ReadFile(p.h, b, &n, nil); err != nil {
			procClearCommError.Call(uintptr(p.h), 0, 0)
			return 0, fmt.Errorf("read from %s failed (board unplugged?): %v", p.name, err)
		}
		if n > 0 || time.Now().After(deadline) {
			return int(n), nil
		}
	}
}

func (p *winPort) SetBaud(baud int) error { return p.configure(baud) }

func (p *winPort) esc(fn uintptr) error {
	if r, _, e := procEscapeCommFunc.Call(uintptr(p.h), fn); r == 0 {
		return e
	}
	return nil
}

func (p *winPort) SetDTR(on bool) error {
	if on {
		return p.esc(setDTR)
	}
	return p.esc(clrDTR)
}

func (p *winPort) SetRTS(on bool) error {
	if on {
		return p.esc(setRTS)
	}
	return p.esc(clrRTS)
}

func (p *winPort) Purge() error {
	procPurgeComm.Call(uintptr(p.h), purgeRxClear|purgeTxClear)
	return nil
}

func (p *winPort) Close() error { return syscall.CloseHandle(p.h) }
