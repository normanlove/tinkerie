//go:build js

package fixie

import (
	"fmt"
	"os"
	"os/exec"
)

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, msg)
}

// In a web page there's no window to open: the page is the window.
func openAppWindow(url, profile string) error { return nil }

func detach(cmd *exec.Cmd) {}
