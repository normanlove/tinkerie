//go:build !windows

package wirie

import (
	"tinkerie/shared/appwindow"
	"tinkerie/shared/droid"

	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Porting notes: macOS uses `open`, Linux uses xdg-open. Chrome/Chromium app
// mode is tried first so the editor gets its own window.
func openWindow(url string) {
	if droid.Active() {
		droid.OpenWindow(url, "Wirie")
		return
	}
	profile := filepath.Join(os.TempDir(), "wirie-window")
	if runtime.GOOS == "darwin" {
		for _, app := range []string{"Google Chrome", "Microsoft Edge", "Brave Browser"} {
			if exec.Command("open", append([]string{"-na", app, "--args", "--app=" + url, "--user-data-dir=" + profile}, appwindow.Maximised(profile)...)...).Run() == nil {
				return
			}
		}
		exec.Command("open", url).Start()
		return
	}
	for _, b := range []string{"google-chrome", "chromium", "chromium-browser", "microsoft-edge", "brave-browser"} {
		if p, err := exec.LookPath(b); err == nil {
			if exec.Command(p, append(appwindow.Maximised(profile), "--app="+url, "--user-data-dir="+profile)...).Start() == nil {
				return
			}
		}
	}
	exec.Command("xdg-open", url).Start()
}

func revealInFolder(p string) {
	if droid.Active() {
		droid.ShowFolder(p)
		return
	}
	if st, err := os.Stat(p); err == nil && !st.IsDir() {
		p = filepath.Dir(p)
	}
	if runtime.GOOS == "darwin" {
		exec.Command("open", p).Start()
		return
	}
	exec.Command("xdg-open", p).Start()
}
