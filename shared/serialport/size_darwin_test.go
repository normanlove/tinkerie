package serialport

import (
	"testing"
	"unsafe"
)

// The ioctl request numbers encode the struct size, so it must match the kernel's.
func TestTermiosSize(t *testing.T) {
	if s := unsafe.Sizeof(termios{}); s != 72 {
		t.Errorf("termios is %d bytes, want 72", s)
	}
}
