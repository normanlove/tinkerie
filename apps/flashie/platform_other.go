//go:build !windows && !linux && !darwin

// Flashie — placeholders for systems other than Windows, Linux and macOS.
// Made by nrmluvdev.

package flashie

import (
	"errors"
	"os/exec"
)

func openSerial(name string, baud int) (Port, error) {
	return nil, errors.New("real serial ports are only implemented on Windows in this stage (try FLASHIE_DEMO=1)")
}

func openSerialEx(name string, baud int, even bool) (Port, error) { return openSerial(name, baud) }

func listPorts() []PortInfo { return nil }

func listUF2Drives() []uf2Drive { return nil }

func launchWindow(url string) error { return exec.Command("xdg-open", url).Start() }

func showError(title, msg string) { println(title + ": " + msg) }
