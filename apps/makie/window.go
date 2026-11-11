package makie

import (
	"tinkerie/shared/appwindow"
	"tinkerie/shared/droid"

	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// openWindow opens the editor in a chromeless app window if a Chromium-based
// browser is installed (Edge ships with every Windows 10/11), else the default browser.
func openWindow(url string) {
	if droid.Active() {
		droid.OpenWindow(url, "Makie")
		return
	}
	dataDir := ""
	if d, err := os.UserConfigDir(); err == nil {
		dataDir = filepath.Join(d, "Makie", "window")
	}
	appArgs := append([]string{"--app=" + url, "--no-first-run", "--no-default-browser-check"}, appwindow.Maximised(dataDir)...)
	if dataDir != "" {
		appArgs = append(appArgs, "--user-data-dir="+dataDir)
	}
	for _, b := range browserCandidates() {
		if p, err := exec.LookPath(b); err == nil {
			if err := exec.Command(p, appArgs...).Start(); err == nil {
				log.Println("opened app window with", p)
				return
			}
		}
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		log.Println("could not open a browser:", err, "— open", url, "yourself")
	}
}

func browserCandidates() []string {
	switch runtime.GOOS {
	case "windows":
		var c []string
		for _, env := range []string{"ProgramFiles(x86)", "ProgramFiles", "LocalAppData"} {
			base := os.Getenv(env)
			if base == "" {
				continue
			}
			c = append(c,
				filepath.Join(base, `Microsoft\Edge\Application\msedge.exe`),
				filepath.Join(base, `Google\Chrome\Application\chrome.exe`),
				filepath.Join(base, `BraveSoftware\Brave-Browser\Application\brave.exe`),
			)
		}
		return c
	case "darwin":
		return []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			"/Applications/Brave Browser.app/Contents/MacOS/Brave Browser",
		}
	default:
		return []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "microsoft-edge", "brave-browser"}
	}
}
