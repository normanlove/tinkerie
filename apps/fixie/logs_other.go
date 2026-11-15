//go:build !windows && !linux && !darwin

package fixie

import (
	"context"
	"os/exec"
	"runtime"
)

func hideWindow(cmd *exec.Cmd) {}

func logsAvailable() (bool, string) {
	if runtime.GOOS == "js" {
		return false, "A web page can't read your computer's logs: save one (see How to get a log) and open it here"
	}
	return false, "Reading this system's logs isn't supported yet: open a saved log instead"
}

func readSystemLogs(ctx context.Context) []logSource { return nil }
