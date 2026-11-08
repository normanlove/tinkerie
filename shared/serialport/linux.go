//go:build linux

package serialport

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

const permissionHint = "add yourself to the dialout group (sudo usermod -aG dialout $USER), then log out and back in"

// Linux termios flags (asm-generic/termbits.h; the same on x86, ARM and ARM64).
const (
	tIGNBRK  = 0o1
	tBRKINT  = 0o2
	tPARMRK  = 0o10
	tISTRIP  = 0o40
	tINLCR   = 0o100
	tIGNCR   = 0o200
	tICRNL   = 0o400
	tIXON    = 0o2000
	tIXANY   = 0o4000
	tIXOFF   = 0o10000
	tOPOST   = 0o1
	tCBAUD   = 0o10017
	tBOTHER  = 0o10000
	tCSIZE   = 0o60
	tCS5     = 0o0
	tCS6     = 0o20
	tCS7     = 0o40
	tCS8     = 0o60
	tCSTOPB  = 0o100
	tCREAD   = 0o200
	tPARENB  = 0o400
	tPARODD  = 0o1000
	tCLOCAL  = 0o4000
	tCIBAUD  = 0o2003600000
	tIBSHIFT = 16
	tCRTSCTS = 0o20000000000
	tCMSPAR  = 0o10000000000
	tISIG    = 0o1
	tICANON  = 0o2
	tECHO    = 0o10
	tECHONL  = 0o100
	tIEXTEN  = 0o100000
	tVTIME   = 5
	tVMIN    = 6

	tcgets2   = 0x802C542A // _IOR('T', 0x2A, struct termios2)
	tcsets2   = 0x402C542B // _IOW('T', 0x2B, struct termios2)
	tcflsh    = 0x540B
	tcioflush = 2 // TCIOFLUSH
	tiocexcl  = 0x540C
	tiocmbis  = 0x5416
	tiocmbic  = 0x5417
)

var stdBaud = map[int]uint32{
	300: syscall.B300, 600: syscall.B600, 1200: syscall.B1200, 1800: syscall.B1800, 2400: syscall.B2400,
	4800: syscall.B4800, 9600: syscall.B9600, 19200: syscall.B19200, 38400: syscall.B38400,
	57600: syscall.B57600, 115200: syscall.B115200, 230400: syscall.B230400, 460800: syscall.B460800,
	500000: syscall.B500000, 576000: syscall.B576000, 921600: syscall.B921600, 1000000: syscall.B1000000,
	1152000: syscall.B1152000, 1500000: syscall.B1500000, 2000000: syscall.B2000000,
	2500000: syscall.B2500000, 3000000: syscall.B3000000, 3500000: syscall.B3500000, 4000000: syscall.B4000000,
}

// termios2 lets Linux use any baud rate (BOTHER), not just the standard ones.
type termios2 struct {
	Iflag, Oflag, Cflag, Lflag uint32
	Line                       uint8
	Cc                         [19]uint8
	Ispeed, Ospeed             uint32
}

func configure(fd int, cfg Config) error {
	var t termios2
	if err := ioctl(fd, tcgets2, unsafe.Pointer(&t)); err != nil {
		return err
	}
	t.Iflag &^= rawIflagClear
	t.Oflag &^= tOPOST
	t.Lflag &^= rawLflagClear
	c, err := rawCflag(uint64(t.Cflag), cfg)
	if err != nil {
		return err
	}
	c &^= tCBAUD | tCIBAUD
	if code, ok := stdBaud[cfg.Baud]; ok {
		c |= uint64(code) // standard rates keep their classic code, for old-style readers
	} else {
		c |= tBOTHER | tBOTHER<<tIBSHIFT // any other rate
	}
	t.Cflag = uint32(c)
	t.Ispeed, t.Ospeed = uint32(cfg.Baud), uint32(cfg.Baud)
	t.Cc[tVMIN], t.Cc[tVTIME] = 1, 0
	return ioctl(fd, tcsets2, unsafe.Pointer(&t))
}

func flush(fd int) error {
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), tcflsh, tcioflush); e != 0 {
		return e
	}
	return nil
}

// list finds USB serial adapters and boards (/dev/ttyUSB*, /dev/ttyACM*) and
// the Raspberry Pi's GPIO serial port, with USB details from sysfs.
func list() []Info {
	var out []Info
	seen := map[string]bool{}
	for _, pat := range []string{"/dev/ttyUSB*", "/dev/ttyACM*"} {
		m, _ := filepath.Glob(pat)
		for _, p := range m {
			out = append(out, describe(p))
			seen[p] = true
		}
	}
	// Raspberry Pi GPIO header serial (pins 8 and 10).
	if p, err := filepath.EvalSymlinks("/dev/serial0"); err == nil && !seen[p] {
		out = append(out, Info{Path: "/dev/serial0", Name: "serial0", Description: "Raspberry Pi GPIO serial (pins 8 and 10)"})
	}
	return out
}

func describe(path string) Info {
	in := Info{Path: path, Name: filepath.Base(path)}
	dev, err := filepath.EvalSymlinks("/sys/class/tty/" + in.Name + "/device")
	if err != nil {
		return in
	}
	// Walk up from the tty to the USB device, which has idVendor.
	for d := dev; d != "/" && d != "."; d = filepath.Dir(d) {
		vid := readSys(d, "idVendor")
		if vid == "" {
			continue
		}
		v, _ := strconv.ParseUint(vid, 16, 16)
		p, _ := strconv.ParseUint(readSys(d, "idProduct"), 16, 16)
		in.USB, in.VID, in.PID = true, uint16(v), uint16(p)
		in.Manufacturer, in.Product, in.Serial = readSys(d, "manufacturer"), readSys(d, "product"), readSys(d, "serial")
		break
	}
	return in
}

func readSys(dir, name string) string {
	b, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}
