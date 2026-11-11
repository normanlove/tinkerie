//go:build windows

// Windows-only pieces: the app window, folders, message boxes and the USB
// driver helpers (Zadig and the signed Hantek 6022 driver). Everything uses
// the Go standard library and DLLs that ship with every copy of Windows.
package scopie

import (
	"tinkerie/shared/appwindow"

	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

var (
	modUser32                = syscall.NewLazyDLL("user32.dll")
	procMessageBoxW          = modUser32.NewProc("MessageBoxW")
	procSetCurrentProcessID  = modShell32.NewProc("SetCurrentProcessExplicitAppUserModelID")
	procSHGetKnownFolderPath = modShell32.NewProc("SHGetKnownFolderPath")
	modOle32                 = syscall.NewLazyDLL("ole32.dll")
	procCoTaskMemFree        = modOle32.NewProc("CoTaskMemFree")
)

const createNoWindow = 0x08000000

func osName() string { return "windows" }

func fatal(msg string) {
	t, _ := syscall.UTF16PtrFromString(msg)
	c, _ := syscall.UTF16PtrFromString(appName)
	procMessageBoxW.Call(0, uintptr(unsafe.Pointer(t)), uintptr(unsafe.Pointer(c)), 0x10)
	os.Exit(1)
}

func setAppID() {
	id, _ := syscall.UTF16PtrFromString("nrmluvdev.Scopie")
	procSetCurrentProcessID.Call(uintptr(unsafe.Pointer(id)))
}

// appDataDir is %APPDATA%\nrmluvdev\Scopie (settings, imported firmware, logs).
func appDataDir() string {
	base := os.Getenv("APPDATA")
	if base == "" {
		base, _ = os.UserHomeDir()
	}
	d := filepath.Join(base, "nrmluvdev", "Scopie")
	_ = os.MkdirAll(d, 0o755)
	return d
}

var folderDocuments = winGUID{0xFDD39AD0, 0x238F, 0x46AF, [8]byte{0xAD, 0xB4, 0x6C, 0x85, 0x48, 0x03, 0x69, 0xC7}}

// exportDir is Documents\Scopie (screenshots, CSV files, setups).
func exportDir() string {
	var p *uint16
	base := ""
	if r, _, _ := procSHGetKnownFolderPath.Call(uintptr(unsafe.Pointer(&folderDocuments)), 0, 0, uintptr(unsafe.Pointer(&p))); r == 0 && p != nil {
		base = utf16At(p)
		procCoTaskMemFree.Call(uintptr(unsafe.Pointer(p)))
	}
	if base == "" {
		h, _ := os.UserHomeDir()
		base = filepath.Join(h, "Documents")
	}
	return filepath.Join(base, "Scopie")
}

// utf16At reads a NUL-terminated UTF-16 string that Windows allocated.
func utf16At(p *uint16) string {
	var n int
	for ptr := unsafe.Pointer(p); *(*uint16)(ptr) != 0 && n < 32768; n++ {
		ptr = unsafe.Add(ptr, 2)
	}
	return syscall.UTF16ToString(unsafe.Slice(p, n))
}

// legacyFirmwareDir is where the Python version kept imported firmware.
func legacyFirmwareDir() string {
	if b := os.Getenv("LOCALAPPDATA"); b != "" {
		return filepath.Join(b, "Scopie", "firmware")
	}
	return ""
}

func findBrowser() string {
	var cands []string
	for _, env := range []string{"ProgramFiles(x86)", "ProgramFiles", "LOCALAPPDATA"} {
		base := os.Getenv(env)
		if base == "" {
			continue
		}
		cands = append(cands,
			filepath.Join(base, "Microsoft", "Edge", "Application", "msedge.exe"),
			filepath.Join(base, "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(base, "BraveSoftware", "Brave-Browser", "Application", "brave.exe"),
		)
	}
	for _, c := range cands {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c
		}
	}
	return ""
}

// openAppWindow shows the interface in a frameless "app mode" browser window
// with its own profile, so it looks and behaves like a normal desktop app.
func openAppWindow(url, dataDir string) error {
	if b := findBrowser(); b != "" {
		cmd := exec.Command(b, append(appwindow.Maximised(filepath.Join(dataDir, "window")),
			"--app="+url,
			"--user-data-dir="+filepath.Join(dataDir, "window"),
			"--no-first-run",
			"--no-default-browser-check",
			"--disable-features=Translate,msEdgeSidebarV2,msHubApps",
			"--disable-renderer-backgrounding",
			"--disable-background-timer-throttling",
		)...)
		raise := appwindow.WatchNewWindow() // so the window opens in front of the launcher
		if err := cmd.Start(); err == nil {
			raise()
			go cmd.Wait()
			return nil
		}
	}
	openURL(url)
	return nil
}

// ---------------------------------------------------------------- USB drivers

func driverDir() string {
	d := filepath.Join(appDataDir(), "drivers")
	_ = os.MkdirAll(d, 0o755)
	return d
}

// installZadig unpacks Zadig (GPL-3.0, shipped unmodified with its licence)
// and starts it; Windows asks for permission first.
func installZadig() error {
	dir := driverDir()
	for _, name := range []string{"zadig.exe", "zadig.ini", "ZADIG-LICENSE.txt"} {
		b, err := driverFS.ReadFile("drivers/" + name)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, name), b, 0o644); err != nil {
			return err
		}
	}
	return shellExecute("open", filepath.Join(dir, "zadig.exe"), "", dir)
}

// install6022Driver installs the signed WinUSB driver packages for the
// Hantek 6022BE/BL (before and after firmware loading) with Windows' own
// pnputil, as administrator.
func install6022Driver() error {
	dir := filepath.Join(driverDir(), "inf")
	_ = os.MkdirAll(dir, 0o755)
	ents, err := driverFS.ReadDir("drivers/inf")
	if err != nil {
		return err
	}
	for _, e := range ents {
		b, err := driverFS.ReadFile("drivers/inf/" + e.Name())
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, e.Name()), b, 0o644); err != nil {
			return err
		}
	}
	// a small batch file next to the .inf files avoids quoting trouble with spaces in paths
	var b strings.Builder
	b.WriteString("@echo off\r\n")
	for _, f := range []string{"Hantek_6022BE_loader.inf", "Hantek_6022BE_openht.inf", "Hantek_6022BL_loader.inf", "Hantek_6022BL_openht.inf"} {
		b.WriteString(`pnputil /add-driver "%~dp0` + f + `" /install` + "\r\n")
	}
	bat := filepath.Join(dir, "install-6022-drivers.cmd")
	if err := os.WriteFile(bat, []byte(b.String()), 0o644); err != nil {
		return err
	}
	ps := "Start-Process -FilePath '" + strings.ReplaceAll(bat, "'", "''") + "' -Verb RunAs -Wait -WindowStyle Hidden"
	cmd := exec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command", ps)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNoWindow}
	if err := cmd.Run(); err != nil {
		return errors.New("Windows didn't install the driver (permission was not given?). Use 'Install USB driver (Zadig)' instead.")
	}
	return nil
}
