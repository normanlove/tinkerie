//go:build darwin

package serialport

import (
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"unsafe"
)

const permissionHint = "check that no other program (like the Arduino IDE) has it open"

// macOS termios flags (sys/termios.h).
const (
	tIGNBRK  = 0x1
	tBRKINT  = 0x2
	tPARMRK  = 0x8
	tISTRIP  = 0x20
	tINLCR   = 0x40
	tIGNCR   = 0x80
	tICRNL   = 0x100
	tIXON    = 0x200
	tIXOFF   = 0x400
	tIXANY   = 0x800
	tOPOST   = 0x1
	tCSIZE   = 0x300
	tCS5     = 0x0
	tCS6     = 0x100
	tCS7     = 0x200
	tCS8     = 0x300
	tCSTOPB  = 0x400
	tCREAD   = 0x800
	tPARENB  = 0x1000
	tPARODD  = 0x2000
	tCLOCAL  = 0x8000
	tCRTSCTS = 0x30000 // CCTS_OFLOW | CRTS_IFLOW
	tCMSPAR  = 0       // macOS has no mark/space parity
	tECHO    = 0x8
	tECHONL  = 0x10
	tISIG    = 0x80
	tICANON  = 0x100
	tIEXTEN  = 0x400
	tVMIN    = 16
	tVTIME   = 17

	tiocgeta    = 0x40487413
	tiocseta    = 0x80487414
	tiocflush   = 0x80047410
	tiocexcl    = 0x2000740d
	tiocmbis    = 0x8004746c
	tiocmbic    = 0x8004746b
	iossiospeed = 0x80085402 // _IOW('T', 2, speed_t): any baud rate
)

// termios matches struct termios on 64-bit macOS.
type termios struct {
	Iflag, Oflag, Cflag, Lflag uint64
	Cc                         [20]uint8
	_                          [4]byte
	Ispeed, Ospeed             uint64
}

func configure(fd int, cfg Config) error {
	var t termios
	if err := ioctl(fd, tiocgeta, unsafe.Pointer(&t)); err != nil {
		return err
	}
	t.Iflag &^= rawIflagClear
	t.Oflag &^= tOPOST
	t.Lflag &^= rawLflagClear
	c, err := rawCflag(t.Cflag, cfg)
	if err != nil {
		return err
	}
	t.Cflag = c
	t.Cc[tVMIN], t.Cc[tVTIME] = 1, 0
	// Standard rates go straight into termios; anything else is set afterwards
	// with IOSSIOSPEED, which macOS drivers accept for any rate.
	std := cfg.Baud <= 230400 && strings.Contains(" 300 600 1200 2400 4800 9600 19200 38400 57600 115200 230400 ", " "+strconv.Itoa(cfg.Baud)+" ")
	speed := uint64(9600)
	if std {
		speed = uint64(cfg.Baud)
	}
	t.Ispeed, t.Ospeed = speed, speed
	if err := ioctl(fd, tiocseta, unsafe.Pointer(&t)); err != nil {
		return err
	}
	if !std {
		s := uint64(cfg.Baud)
		return ioctl(fd, iossiospeed, unsafe.Pointer(&s))
	}
	return nil
}

func flush(fd int) error {
	which := int32(3) // FREAD | FWRITE
	return ioctl(fd, tiocflush, unsafe.Pointer(&which))
}

// list finds /dev/cu.* ports (never tty.*: opening those waits for a carrier
// signal), with USB details from ioreg.
func list() []Info {
	m, _ := filepath.Glob("/dev/cu.*")
	usb := ioregPorts()
	var out []Info
	for _, p := range m {
		name := filepath.Base(p)
		if strings.Contains(name, "Bluetooth-Incoming-Port") || strings.Contains(name, "debug-console") || strings.Contains(name, "wlan-debug") {
			continue
		}
		in := Info{Path: p, Name: name}
		if u, ok := usb[p]; ok {
			u.Path, u.Name = p, name
			in = u
		} else if strings.Contains(name, "usb") || strings.Contains(name, "SLAB") || strings.Contains(name, "wch") {
			in.USB = true
		}
		out = append(out, in)
	}
	return out
}

func ioregPorts() map[string]Info {
	b, err := exec.Command("ioreg", "-r", "-c", "IOUSBHostDevice", "-l", "-w0").Output()
	if err != nil {
		return nil
	}
	return parseIoreg(string(b))
}
