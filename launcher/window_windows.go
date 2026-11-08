//go:build windows

package launcher

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// openWindow shows the launcher in a chromeless Edge/Chrome "app" window with
// its own profile, else in the default browser.
func openWindow(url, dataDir string) {
	profile := filepath.Join(dataDir, "window")
	for _, env := range []string{"ProgramFiles(x86)", "ProgramFiles", "LOCALAPPDATA"} {
		base := os.Getenv(env)
		if base == "" {
			continue
		}
		for _, b := range []string{`Microsoft\Edge\Application\msedge.exe`, `Google\Chrome\Application\chrome.exe`, `BraveSoftware\Brave-Browser\Application\brave.exe`} {
			p := filepath.Join(base, b)
			if _, err := os.Stat(p); err != nil {
				continue
			}
			cmd := exec.Command(p, "--app="+url, "--user-data-dir="+profile, "--window-size=980,640",
				"--no-first-run", "--no-default-browser-check", "--disable-features=Translate")
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

// detach lets a started app outlive the launcher.
func detach(cmd *exec.Cmd) {
	const createNewProcessGroup = 0x00000200
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNewProcessGroup}
}
