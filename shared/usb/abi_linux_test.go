package usb

import (
	"runtime"
	"testing"
	"unsafe"
)

// The ioctl numbers and layouts must match linux/usbdevice_fs.h exactly. These
// are the values a C compiler gives for that header on 64-bit Linux.
func TestABI64(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		t.Skip("64-bit values")
	}
	for name, c := range map[string][2]uintptr{
		"CONTROL": {reqControl, 0xc0185500}, "BULK": {reqBulk, 0xc0185502},
		"SUBMITURB": {reqSubmitURB, 0x8038550a}, "DISCARDURB": {reqDiscardURB, 0x550b},
		"REAPURB": {reqReapURB, 0x4008550c}, "CLAIMINTERFACE": {reqClaim, 0x8004550f},
		"RELEASEINTERFACE": {reqRelease, 0x80045510}, "IOCTL": {reqIoctl, 0xc0105512},
		"CLEAR_HALT": {reqClearHalt, 0x80045515}, "DISCONNECT": {reqDisconnect, 0x5516},
		"CONNECT": {reqConnect, 0x5517},
	} {
		if c[0] != c[1] {
			t.Errorf("USBDEVFS_%s = %#x, want %#x", name, c[0], c[1])
		}
	}
	var u urb
	for name, c := range map[string][2]uintptr{
		"status": {unsafe.Offsetof(u.Status), 4}, "flags": {unsafe.Offsetof(u.Flags), 8},
		"buffer": {unsafe.Offsetof(u.Buffer), 16}, "buffer_length": {unsafe.Offsetof(u.BufferLength), 24},
		"actual_length": {unsafe.Offsetof(u.ActualLength), 28}, "usercontext": {unsafe.Offsetof(u.UserContext), 48},
	} {
		if c[0] != c[1] {
			t.Errorf("urb.%s at %d, want %d", name, c[0], c[1])
		}
	}
	var c ctrlTransfer
	if unsafe.Offsetof(c.Timeout) != 8 || unsafe.Offsetof(c.Data) != 16 {
		t.Errorf("ctrltransfer layout: timeout %d data %d", unsafe.Offsetof(c.Timeout), unsafe.Offsetof(c.Data))
	}
	_ = runtime.GOARCH
}

func TestListWithoutUSB(t *testing.T) {
	_ = List() // must not fail when sysfs has no USB bus (as in containers)
}
