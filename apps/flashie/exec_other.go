//go:build !windows

package flashie

import "os/exec"

func hideWindow(cmd *exec.Cmd) {}
