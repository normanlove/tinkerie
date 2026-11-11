//go:build !windows && !linux

// Opening links and folders on macOS and other systems. USB access lives in
// usb_usbfs.go (Linux, macOS) and usb_windows.go.
package scopie

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
)

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
	if runtime.GOOS == "darwin" {
		return fmt.Sprintf("USB %04X:%04X: another app or driver is using it. Quit the scope's own software, then replug it.", u.VID, u.PID)
	}
	return fmt.Sprintf("USB %04X:%04X: USB scopes aren't supported on this system yet.", u.VID, u.PID)
}
