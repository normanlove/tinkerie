package usb

import (
	"testing"
	"unsafe"
)

// The same checks for 32-bit Linux (older Raspberry Pi OS), from the header's
// layout with 4-byte pointers.
func TestABI32(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) != 4 {
		t.Skip("32-bit values")
	}
	for name, c := range map[string][2]uintptr{
		"CONTROL": {reqControl, 0xc0105500}, "BULK": {reqBulk, 0xc0105502},
		"SUBMITURB": {reqSubmitURB, 0x802c550a}, "REAPURB": {reqReapURB, 0x4004550c}, "IOCTL": {reqIoctl, 0xc00c5512},
	} {
		if c[0] != c[1] {
			t.Errorf("USBDEVFS_%s = %#x, want %#x", name, c[0], c[1])
		}
	}
	var u urb
	if unsafe.Offsetof(u.Buffer) != 12 || unsafe.Offsetof(u.UserContext) != 40 || unsafe.Sizeof(u) != 44 {
		t.Errorf("urb layout: buffer %d ctx %d size %d", unsafe.Offsetof(u.Buffer), unsafe.Offsetof(u.UserContext), unsafe.Sizeof(u))
	}
}
