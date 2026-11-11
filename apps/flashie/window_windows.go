//go:build windows

// Flashie — opens the UI in a chromeless app window using the Microsoft Edge
// engine that ships with Windows 10/11. Made by nrmluvdev.

package flashie

import (
	"tinkerie/shared/appwindow"

	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"
)

func findBrowser() string {
	var cands []string
	for _, env := range []string{"ProgramFiles(x86)", "ProgramFiles", "LOCALAPPDATA"} {
		if base := os.Getenv(env); base != "" {
			cands = append(cands,
				filepath.Join(base, `Microsoft\Edge\Application\msedge.exe`),
				filepath.Join(base, `Google\Chrome\Application\chrome.exe`),
				filepath.Join(base, `BraveSoftware\Brave-Browser\Application\brave.exe`))
		}
	}
	for _, c := range cands {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c
		}
	}
	return ""
}

func launchWindow(url string) error {
	if b := findBrowser(); b != "" {
		profile := filepath.Join(os.Getenv("LOCALAPPDATA"), "nrmluvdev", "Flashie", "window")
		os.MkdirAll(profile, 0o755)
		cmd := exec.Command(b, append(appwindow.Maximised(profile),
			"--app="+url,
			"--user-data-dir="+profile,
			"--no-first-run",
			"--no-default-browser-check",
			"--disable-extensions",
			"--disable-sync",
			"--disable-features=Translate,msEdgeSidebarV2,msHubApps",
		)...)
		raise := appwindow.WatchNewWindow() // so the window opens in front of the launcher
		if err := cmd.Start(); err == nil {
			raise()
			return nil
		}
	}
	// Fallback: default browser.
	return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
}

func showError(title, msg string) {
	user32 := syscall.NewLazyDLL("user32.dll")
	t, _ := syscall.UTF16PtrFromString(title)
	m, _ := syscall.UTF16PtrFromString(msg)
	user32.NewProc("MessageBoxW").Call(0, uintptr(unsafe.Pointer(m)), uintptr(unsafe.Pointer(t)), 0x10)
}
