package serialport

import (
	"bytes"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

// newPTY returns the controller side of a pseudo-terminal and the path of its
// other side, which behaves like a serial port.
func newPTY(t *testing.T) (*os.File, string) {
	m, err := os.OpenFile("/dev/ptmx", os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		t.Skip("no pseudo-terminals here:", err)
	}
	t.Cleanup(func() { m.Close() })
	unlock := int32(0)
	if err := ioctl(int(m.Fd()), 0x40045431, unsafe.Pointer(&unlock)); err != nil { // TIOCSPTLCK
		t.Fatal(err)
	}
	var n uint32
	if err := ioctl(int(m.Fd()), 0x80045430, unsafe.Pointer(&n)); err != nil { // TIOCGPTN
		t.Fatal(err)
	}
	return m, "/dev/pts/" + strconv.Itoa(int(n))
}

func stty(t *testing.T, p *Port) string {
	cmd := exec.Command("stty", "-a")
	cmd.Stdin = p.f
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func TestPortOnPTY(t *testing.T) {
	ctl, path := newPTY(t)
	p, err := Open(path, Config{Baud: 115200, Parity: 'E'})
	if err != nil {
		t.Fatal(err)
	}

	s := stty(t, p)
	// (Pseudo-terminals always force 8 data bits and no parity, so those are
	// checked in TestCflag instead.)
	for _, want := range []string{"speed 115200", "-icanon", "-echo", "-opost", "-icrnl", "-ixon"} {
		if !strings.Contains(s, want) {
			t.Errorf("after Open, settings lack %q:\n%s", want, s)
		}
	}

	// Data from the other end arrives.
	ctl.Write([]byte("hello"))
	buf := make([]byte, 64)
	n, err := p.ReadTimeout(buf, time.Second)
	if err != nil || string(buf[:n]) != "hello" {
		t.Fatalf("ReadTimeout = %q, %v", buf[:n], err)
	}

	// Nothing to read: returns 0, nil after about the timeout.
	t0 := time.Now()
	n, err = p.ReadTimeout(buf, 150*time.Millisecond)
	if n != 0 || err != nil || time.Since(t0) < 120*time.Millisecond || time.Since(t0) > time.Second {
		t.Errorf("empty ReadTimeout = %d, %v after %v", n, err, time.Since(t0))
	}

	// Writes reach the other end unchanged (raw mode: no \n -> \r\n).
	p.Write([]byte("a\nb"))
	got := make([]byte, 16)
	ctl.SetReadDeadline(time.Now().Add(time.Second))
	m, _ := ctl.Read(got)
	if !bytes.Equal(got[:m], []byte("a\nb")) {
		t.Errorf("other end got %q", got[:m])
	}

	// Odd speeds work too (ESP boot messages use 74880).
	if err := p.SetBaud(74880); err != nil {
		t.Fatal(err)
	}
	var tm termios2
	if err := ioctl(p.fd, tcgets2, unsafe.Pointer(&tm)); err != nil || tm.Ospeed != 74880 || tm.Cflag&tCBAUD != tBOTHER {
		t.Errorf("after SetBaud(74880): ospeed %d, cflag %o, %v", tm.Ospeed, tm.Cflag, err)
	}

	// Close wakes a blocked Read.
	done := make(chan error, 1)
	go func() { _, err := p.Read(buf); done <- err }()
	time.Sleep(100 * time.Millisecond)
	p.Close()
	select {
	case err := <-done:
		if err == nil {
			t.Error("Read after Close returned no error")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Close didn't wake Read")
	}
}

func TestCflag(t *testing.T) {
	for _, c := range []struct {
		cfg      Config
		set, clr uint64
	}{
		{Config{DataBits: 8, Parity: 'N', StopBits: 1}, tCS8 | tCREAD | tCLOCAL, tPARENB | tCSTOPB | tCRTSCTS},
		{Config{DataBits: 8, Parity: 'E', StopBits: 1}, tCS8 | tPARENB, tPARODD | tCSTOPB},
		{Config{DataBits: 7, Parity: 'O', StopBits: 2}, tCS7 | tPARENB | tPARODD | tCSTOPB, 0},
		{Config{DataBits: 8, Parity: 'M', StopBits: 1.5}, tCS8 | tPARENB | tCMSPAR | tPARODD | tCSTOPB, 0},
		{Config{DataBits: 8, Parity: 'S', StopBits: 1}, tCS8 | tPARENB | tCMSPAR, tPARODD},
	} {
		got, err := rawCflag(tCRTSCTS|tCSIZE|tPARODD, c.cfg) // start dirty: old bits must go
		if err != nil {
			t.Errorf("%+v: %v", c.cfg, err)
			continue
		}
		if got&tCSIZE != c.set&tCSIZE || got&(c.set&^tCSIZE) != c.set&^tCSIZE || got&c.clr != 0 {
			t.Errorf("%+v: cflag %o", c.cfg, got)
		}
	}
	if _, err := rawCflag(0, Config{DataBits: 9, Parity: 'N'}); err == nil {
		t.Error("9 data bits should fail")
	}
}

func TestStandardSpeedCode(t *testing.T) {
	_, path := newPTY(t)
	p, err := Open(path, Config{Baud: 9600})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	if s := stty(t, p); !strings.Contains(s, "speed 9600") {
		t.Errorf("9600 should use its classic code:\n%s", s)
	}
}
