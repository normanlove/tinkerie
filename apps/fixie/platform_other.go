//go:build !windows

// Fixie on Linux, Raspberry Pi OS and macOS: the window comes from
// shared/appwindow. Part of the nrmluvdev maker suite. Created by nrmluvdev.
package fixie

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"tinkerie/shared/appwindow"
)

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}

func openAppWindow(url, profile string) error { return appwindow.Open(url, profile) }

// detach lets an app started for a step outlive Fixie.
func detach(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true} }
