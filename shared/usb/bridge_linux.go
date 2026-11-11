//go:build linux

package usb

// bridge_linux.go: USB on Android. Apps can't read /sys/bus/usb or open
// /dev/bus/usb themselves there, so the Tinkerie Android app lists devices
// with Android's USB manager and, once the person allows it, opens the
// device and passes its file descriptor over. usbfs works the same on it.

import (
	"errors"
	"syscall"
	"time"

	"tinkerie/shared/droid"
)

func bridgeList() []Info {
	var r struct {
		Devices []Info `json:"devices"`
	}
	if _, err := droid.Call("usblist", nil, &r, 10*time.Second); err != nil {
		return nil
	}
	return r.Devices
}

func bridgeOpen(path string) (int, error) {
	fd, err := droid.Call("usbopen", map[string]any{"path": path}, nil, 2*time.Minute)
	if err != nil {
		return -1, err
	}
	if fd < 0 {
		return -1, errors.New("the Tinkerie app didn't hand over the USB device")
	}
	syscall.CloseOnExec(fd)
	return fd, nil
}
