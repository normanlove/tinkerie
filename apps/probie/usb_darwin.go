//go:build darwin && !cgo

package probie

import "errors"

// USB logic analysers on macOS would need IOKit, which means cgo.
func listUSBDevices() []usbInfo { return nil }

func openUSBDevice(path string) (usbDevice, error) {
	return nil, errors.New("the USB logic analyser isn't supported on macOS yet (it works on Windows, Linux and Raspberry Pi)")
}
