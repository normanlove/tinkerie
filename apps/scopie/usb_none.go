//go:build !windows && !linux && !(darwin && cgo)

// Systems without a USB back end: demo mode, network bench scopes and CSV
// files still work.
package scopie

import "errors"

func listUSBDevices() []usbInfo { return nil }

func openUSBDevice(path string) (usbDevice, error) {
	return nil, errors.New("USB scopes aren't supported in this build")
}
