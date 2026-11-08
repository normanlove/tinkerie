//go:build !windows && !linux

// USB access is implemented for Windows (WinUSB). On other systems the source
// builds and runs (demo mode, network bench scopes, CSV files); USB scopes
// need a libusb/usbfs back end here before they work.
package scopie

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
)

func listUSBDevices() []usbInfo { return nil }

func openUSBDevice(path string) (usbDevice, error) {
	return nil, errors.New("USB scopes are only supported on Windows in this version")
}

func shellExecute(verb, file, args, dir string) error {
	return errors.New("not supported on " + runtime.GOOS)
}

func openURL(u string) {
	if runtime.GOOS == "darwin" {
		_ = exec.Command("open", u).Start()
		return
	}
	_ = exec.Command("xdg-open", u).Start()
}

func openFolder(dir string) { openURL(dir) }

// usbDriverHint says what to do when a device can't be used yet.
func usbDriverHint(u usbInfo) string {
	return fmt.Sprintf("USB %04X:%04X: USB scopes aren't supported on this system yet.", u.VID, u.PID)
}
