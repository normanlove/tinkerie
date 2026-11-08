//go:build windows

package wirie

import (
	"tinkerie/shared/appwindow"

	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// openWindow opens the editor in a chromeless Edge/Chrome "app" window with its
// own profile (so it behaves like a native window), else the default browser.
func openWindow(url string) {
	cands := []string{}
	for _, env := range []string{"ProgramFiles(x86)", "ProgramFiles", "LOCALAPPDATA"} {
		base := os.Getenv(env)
		if base == "" {
			continue
		}
		cands = append(cands,
			filepath.Join(base, `Microsoft\Edge\Application\msedge.exe`),
			filepath.Join(base, `Google\Chrome\Application\chrome.exe`),
			filepath.Join(base, `BraveSoftware\Brave-Browser\Application\brave.exe`),
		)
	}
	profile := filepath.Join(os.Getenv("LOCALAPPDATA"), "Wirie", "window")
	os.MkdirAll(profile, 0o755)
	for _, c := range cands {
		if _, err := os.Stat(c); err == nil {
			cmd := exec.Command(c, append(appwindow.Maximised(profile), "--app="+url, "--user-data-dir="+profile, "--no-first-run", "--no-default-browser-check", "--disable-features=Translate")...)
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			if cmd.Start() == nil {
				return
			}
		}
	}
	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	cmd.Start()
}

func revealInFolder(p string) {
	st, err := os.Stat(p)
	var cmd *exec.Cmd
	if err == nil && !st.IsDir() {
		cmd = exec.Command("explorer", "/select,", p)
	} else {
		cmd = exec.Command("explorer", p)
	}
	cmd.Start()
}
