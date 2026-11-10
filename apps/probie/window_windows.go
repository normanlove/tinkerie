//go:build windows

package probie

import (
	"tinkerie/shared/appwindow"

	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"
)

var procMessageBoxW = syscall.NewLazyDLL("user32.dll").NewProc("MessageBoxW")

// openWindow shows the UI in a chromeless Edge (or Chrome) app window with
// its own profile so it feels like a native app; falls back to the default
// browser.
func openWindow(url, dataDir string) {
	var cands []string
	for _, env := range []string{"ProgramFiles(x86)", "ProgramFiles", "LOCALAPPDATA"} {
		base := os.Getenv(env)
		if base == "" {
			continue
		}
		cands = append(cands,
			filepath.Join(base, `Microsoft\Edge\Application\msedge.exe`),
			filepath.Join(base, `Google\Chrome\Application\chrome.exe`),
			filepath.Join(base, `BraveSoftware\Brave-Browser\Application\brave.exe`))
	}
	for _, c := range cands {
		if _, err := os.Stat(c); err != nil {
			continue
		}
		cmd := exec.Command(c, append(appwindow.Maximised(filepath.Join(dataDir, "window")),
			"--app="+url,
			"--user-data-dir="+filepath.Join(dataDir, "window"),
			"--no-first-run", "--no-default-browser-check",
			"--disable-features=Translate,msEdgeSidebarV2,msHubApps",
			"--disable-background-timer-throttling")...)
		if err := cmd.Start(); err == nil {
			return
		}
	}
	verb, _ := syscall.UTF16PtrFromString("open")
	u, _ := syscall.UTF16PtrFromString(url)
	procShellExecuteW.Call(0, uintptr(unsafe.Pointer(verb)), uintptr(unsafe.Pointer(u)), 0, 0, 1)
}

func fatal(msg string) {
	t, _ := syscall.UTF16PtrFromString("Probie")
	m, _ := syscall.UTF16PtrFromString(msg)
	procMessageBoxW.Call(0, uintptr(unsafe.Pointer(m)), uintptr(unsafe.Pointer(t)), 0x10)
	os.Exit(1)
}
