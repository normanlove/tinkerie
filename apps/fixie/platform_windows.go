//go:build windows

// Fixie on Windows: the window is Edge, Chrome or Brave in app mode, and
// errors show in a message box. Only the standard library and DLLs that come
// with Windows. Part of the nrmluvdev maker suite. Created by nrmluvdev.
package fixie

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"

	"tinkerie/shared/appwindow"
)

var procMessageBoxW = syscall.NewLazyDLL("user32.dll").NewProc("MessageBoxW")

func fatal(msg string) {
	t, _ := syscall.UTF16PtrFromString(msg)
	c, _ := syscall.UTF16PtrFromString("Fixie")
	procMessageBoxW.Call(0, uintptr(unsafe.Pointer(t)), uintptr(unsafe.Pointer(c)), 0x10)
	os.Exit(1)
}

func findBrowser() string {
	for _, env := range []string{"ProgramFiles(x86)", "ProgramFiles", "LOCALAPPDATA"} {
		base := os.Getenv(env)
		if base == "" {
			continue
		}
		for _, c := range []string{
			filepath.Join(base, "Microsoft", "Edge", "Application", "msedge.exe"),
			filepath.Join(base, "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(base, "BraveSoftware", "Brave-Browser", "Application", "brave.exe"),
		} {
			if st, err := os.Stat(c); err == nil && !st.IsDir() {
				return c
			}
		}
	}
	return ""
}

func openAppWindow(url, profile string) error {
	if b := findBrowser(); b != "" {
		cmd := exec.Command(b, append(appwindow.Maximised(profile),
			"--app="+url,
			"--user-data-dir="+profile,
			"--no-first-run",
			"--no-default-browser-check",
			"--disable-features=Translate,msEdgeSidebarV2,msHubApps",
		)...)
		raise := appwindow.WatchNewWindow() // so the window opens in front of the launcher
		if err := cmd.Start(); err == nil {
			raise()
			go cmd.Wait()
			return nil
		}
	}
	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
	return cmd.Start()
}
