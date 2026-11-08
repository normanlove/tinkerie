package serialport

import (
	"testing"
	"unsafe"
)

// The ioctl request numbers encode the struct size, so it must match the kernel's.
func TestTermios2Size(t *testing.T) {
	if s := unsafe.Sizeof(termios2{}); s != 44 {
		t.Errorf("termios2 is %d bytes, want 44", s)
	}
}
