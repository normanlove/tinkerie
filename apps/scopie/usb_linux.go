//go:build linux

// USB scopes on Linux and Raspberry Pi OS, through usbfs (shared/usb).
package scopie

import (
	"fmt"
	"os/exec"
	"tinkerie/shared/droid"
)

// usbDriverHint says what to do when a device can't be used yet.
func usbDriverHint(u usbInfo) string {
	return fmt.Sprintf("USB %04X:%04X: Linux needs permission first. Run install-usb.sh in Tinkerie's folder once, then replug the scope.", u.VID, u.PID)
}

func shellExecute(verb, file, args, dir string) error {
	return fmt.Errorf("not needed on Linux")
}

func openURL(u string) {
	if droid.Active() {
		droid.Browse(u)
		return
	}
	_ = exec.Command("xdg-open", u).Start()
}

func openFolder(dir string) {
	if droid.Active() {
		droid.ShowFolder(dir)
		return
	}
	openURL(dir)
}
