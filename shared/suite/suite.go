// Package suite lets one Tinkerie app hand a file to another, by starting
// Tinkerie again with --app <name> <file> (the receiving app takes over if
// it's already open). Made by nrmluvdev.
package suite

import (
	"os"
	"os/exec"
	"path/filepath"
)

// OpenWith starts another Tinkerie app with the given arguments.
func OpenWith(app string, args ...string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, append([]string{"--app", app}, args...)...)
	cmd.Dir = filepath.Dir(exe)
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait()
	return nil
}
