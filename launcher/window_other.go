//go:build !windows

package launcher

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
)

// openWindow shows the launcher in a Chromium "app" window if one is installed,
// else in the default browser.
func openWindow(url, dataDir string) {
	args := []string{"--app=" + url, "--user-data-dir=" + filepath.Join(dataDir, "window"), "--window-size=980,640", "--no-first-run"}
	if runtime.GOOS == "darwin" {
		for _, app := range []string{"Google Chrome", "Microsoft Edge", "Brave Browser", "Chromium"} {
			if exec.Command("open", append([]string{"-na", app, "--args"}, args...)...).Run() == nil {
				return
			}
		}
		exec.Command("open", url).Start()
		return
	}
	for _, b := range []string{"chromium", "chromium-browser", "google-chrome", "microsoft-edge", "brave-browser"} {
		if p, err := exec.LookPath(b); err == nil && exec.Command(p, args...).Start() == nil {
			return
		}
	}
	exec.Command("xdg-open", url).Start()
}

// detach lets a started app outlive the launcher.
func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
