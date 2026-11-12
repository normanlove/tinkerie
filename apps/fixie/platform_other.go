//go:build !windows

// Fixie on Linux, Raspberry Pi OS and macOS: the window comes from
// shared/appwindow. Part of the nrmluvdev maker suite. Created by nrmluvdev.
package fixie

import (
	"fmt"
	"os"

	"tinkerie/shared/appwindow"
)

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}

func openAppWindow(url, profile string) error { return appwindow.Open(url, profile) }
