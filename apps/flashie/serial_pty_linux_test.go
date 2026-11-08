package flashie

import (
	"bytes"
	"os"
	"strconv"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"tinkerie/shared/serialport"
)

// ptyPair opens a pseudo-terminal: the controller end and the path of the end
// that behaves like a serial port.
func ptyPair(t *testing.T) (*os.File, string) {
	m, err := os.OpenFile("/dev/ptmx", os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		t.Skip("no pseudo-terminals here:", err)
	}
	t.Cleanup(func() { m.Close() })
	unlock := int32(0)
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, m.Fd(), 0x40045431, uintptr(unsafe.Pointer(&unlock))); e != 0 {
		t.Fatal(e)
	}
	var n uint32
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, m.Fd(), 0x80045430, uintptr(unsafe.Pointer(&n))); e != 0 {
		t.Fatal(e)
	}
	return m, "/dev/pts/" + strconv.Itoa(int(n))
}

// linePort is the real serial port, except that DTR and RTS (which a
// pseudo-terminal doesn't have) go straight to the simulated chip.
type linePort struct {
	*serialport.Port
	sim *simChip
}

func (p linePort) SetDTR(on bool) error { return p.sim.SetDTR(on) }
func (p linePort) SetRTS(on bool) error { return p.sim.SetRTS(on) }

// TestESP32FlashOverRealSerial runs the whole ESP32 flashing protocol through
// the Linux/macOS serial code, with a simulated chip on the other end.
func TestESP32FlashOverRealSerial(t *testing.T) {
	ctl, path := ptyPair(t)
	sim := newSimChip(0, 0)

	// Wire the simulated chip to the controller end of the pseudo-terminal.
	stop := make(chan struct{})
	defer close(stop)
	go func() { // PC -> chip
		buf := make([]byte, 4096)
		for {
			n, err := ctl.Read(buf)
			if n > 0 {
				sim.Write(append([]byte(nil), buf[:n]...))
			}
			if err != nil {
				return
			}
		}
	}()
	go func() { // chip -> PC
		buf := make([]byte, 4096)
		for {
			select {
			case <-stop:
				return
			default:
			}
			if n, _ := sim.ReadTimeout(buf, 20*time.Millisecond); n > 0 {
				ctl.Write(buf[:n])
			}
		}
	}()

	sp, err := openSerial(path, 115200)
	if err != nil {
		t.Fatal(err)
	}
	defer sp.Close()
	l := NewLoader(linePort{sp.(*serialport.Port), sim}, false, false)
	if err := l.Connect(); err != nil {
		t.Fatalf("connect: %v", err)
	}
	info, err := l.DetectChip()
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if err := l.ChangeBaud(460800); err != nil {
		t.Fatal(err)
	}
	img := testImage(200_003)
	if err := l.Flash(img, FlashOptions{Offset: 0x10000, FlashSize: 4 << 20, Compress: true}); err != nil {
		t.Fatalf("flash: %v", err)
	}
	if !bytes.Equal(sim.flash[0x10000:0x10000+len(img)], img) {
		t.Fatal("flash contents differ")
	}
	t.Logf("flashed %d bytes to a simulated %s over %s", len(img), info.Chip, path)
}
