//go:build windows

package fixie

import (
	"context"
	"os/exec"
	"syscall"
)

func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
}

func logsAvailable() (bool, string) {
	return true, "Windows' event logs: the newest critical, error and warning events from System, and errors from Application"
}

// readSystemLogs reads the event logs with wevtutil (part of Windows), newest
// first, as XML the window understands.
func readSystemLogs(ctx context.Context) []logSource {
	return []logSource{
		source(ctx, "System events (critical, errors and warnings)", "wevtutil", "qe", "System",
			"/q:*[System[(Level=1 or Level=2 or Level=3)]]", "/c:600", "/rd:true", "/f:RenderedXml"),
		source(ctx, "Application events (critical and errors)", "wevtutil", "qe", "Application",
			"/q:*[System[(Level=1 or Level=2)]]", "/c:300", "/rd:true", "/f:RenderedXml"),
	}
}
