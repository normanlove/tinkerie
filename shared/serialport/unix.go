//go:build linux || darwin

package serialport

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

const (
	tiocmDTR = 0x002
	tiocmRTS = 0x004
)

func ioctl(fd int, req uintptr, arg unsafe.Pointer) error {
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), req, uintptr(arg)); e != 0 {
		return e
	}
	return nil
}

func open(name string, cfg Config) (*Port, error) {
	// Non-blocking so the runtime poller can wait on it; O_NOCTTY so a board
	// can't become this program's controlling terminal.
	fd, err := syscall.Open(name, syscall.O_RDWR|syscall.O_NOCTTY|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
	if err != nil {
		if err == syscall.EACCES {
			return nil, fmt.Errorf("no permission to open %s: %s", name, permissionHint)
		}
		if err == syscall.EBUSY {
			return nil, fmt.Errorf("%s is in use by another program", name)
		}
		return nil, fmt.Errorf("couldn't open %s: %v", name, err)
	}
	_ = ioctl(fd, tiocexcl, nil) // keep other programs out while we use it
	if err := configure(fd, cfg); err != nil {
		syscall.Close(fd)
		return nil, err
	}
	setModemBit(fd, tiocmDTR, cfg.DTR)
	setModemBit(fd, tiocmRTS, cfg.RTS)
	flush(fd)
	f := os.NewFile(uintptr(fd), name) // pollable, because the fd is non-blocking
	return &Port{f: f, fd: fd, cfg: cfg}, nil
}

func setModemBit(fd int, bit int32, on bool) error {
	req := uintptr(tiocmbic)
	if on {
		req = tiocmbis
	}
	b := bit
	return ioctl(fd, req, unsafe.Pointer(&b))
}

// rawFlags clears line editing and translation (like cfmakeraw) and sets the
// character size, parity and stop bits.
func rawCflag(cflag uint64, cfg Config) (uint64, error) {
	cflag &^= tCSIZE | tPARENB | tPARODD | tCSTOPB | tCRTSCTS | tCMSPAR
	switch cfg.DataBits {
	case 5:
		cflag |= tCS5
	case 6:
		cflag |= tCS6
	case 7:
		cflag |= tCS7
	case 8:
		cflag |= tCS8
	default:
		return 0, fmt.Errorf("%d data bits isn't possible", cfg.DataBits)
	}
	switch cfg.Parity {
	case 'N', 'n':
	case 'E', 'e':
		cflag |= tPARENB
	case 'O', 'o':
		cflag |= tPARENB | tPARODD
	case 'M', 'm', 'S', 's':
		if tCMSPAR == 0 {
			return 0, fmt.Errorf("mark and space parity aren't available on this system")
		}
		cflag |= tPARENB | tCMSPAR
		if cfg.Parity == 'M' || cfg.Parity == 'm' {
			cflag |= tPARODD
		}
	default:
		return 0, fmt.Errorf("unknown parity %q", cfg.Parity)
	}
	if cfg.StopBits > 1 {
		cflag |= tCSTOPB
	}
	return cflag | tCREAD | tCLOCAL, nil
}

const rawIflagClear = tIGNBRK | tBRKINT | tPARMRK | tISTRIP | tINLCR | tIGNCR | tICRNL | tIXON | tIXOFF | tIXANY
const rawLflagClear = tECHO | tECHONL | tICANON | tISIG | tIEXTEN
