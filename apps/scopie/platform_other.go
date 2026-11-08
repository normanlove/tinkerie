//go:build !windows

// Linux / macOS versions of the platform pieces, so the source builds and
// runs everywhere for porting (demo mode, network bench scopes, CSV files).
package scopie

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"tinkerie/shared/appwindow"
)

func osName() string { return runtime.GOOS }

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}

func setAppID() {}

func appDataDir() string {
	base, err := os.UserConfigDir()
	if err != nil {
		base = os.TempDir()
	}
	d := filepath.Join(base, "nrmluvdev", "Scopie")
	_ = os.MkdirAll(d, 0o755)
	return d
}

func exportDir() string {
	h, _ := os.UserHomeDir()
	return filepath.Join(h, "Documents", "Scopie")
}

func legacyFirmwareDir() string {
	h, _ := os.UserHomeDir()
	if runtime.GOOS == "darwin" {
		return filepath.Join(h, "Library", "Application Support", "Scopie", "firmware")
	}
	return filepath.Join(h, ".local", "share", "scopie", "firmware")
}

func openAppWindow(url, dataDir string) error {
	return appwindow.Open(url, filepath.Join(dataDir, "window"))
}

func installZadig() error {
	return errors.New("Zadig is only needed on Windows")
}

func install6022Driver() error {
	return errors.New("the signed driver is only needed on Windows")
}
