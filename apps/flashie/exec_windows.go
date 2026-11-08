//go:build windows

package flashie

import (
	"os/exec"
	"syscall"
)

// hideWindow stops a console window flashing up for child processes.
func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
}
