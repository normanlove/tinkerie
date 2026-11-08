//go:build linux || darwin

// Flashie on Linux, Raspberry Pi OS and macOS: serial ports through
// shared/serialport, Pico BOOTSEL drives from the mounted volumes, and the
// window through shared/appwindow. Made by nrmluvdev.

package flashie

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"tinkerie/shared/appwindow"
	"tinkerie/shared/serialport"
)

func openSerial(name string, baud int) (Port, error) { return openSerialEx(name, baud, false) }

// openSerialEx opens a port at 8N1, or 8E1 for the STM32 bootloader. DTR and
// RTS start low, as on Windows, so boards aren't held in reset.
func openSerialEx(name string, baud int, evenParity bool) (Port, error) {
	cfg := serialport.Config{Baud: baud, Parity: 'N'}
	if evenParity {
		cfg.Parity = 'E'
	}
	p, err := serialport.Open(name, cfg)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func listPorts() []PortInfo {
	var out []PortInfo
	for _, in := range serialport.List() {
		pi := PortInfo{Name: in.Path, Kind: "serial", Device: in.Path, USB: in.USB, VID: in.VID, PID: in.PID}
		pi.Friendly = strings.TrimSpace(in.Manufacturer + " " + in.Product)
		if pi.Friendly == "" {
			pi.Friendly = in.Description
		}
		if pi.Friendly == "" {
			pi.Friendly = in.Name
		}
		pi.Guess = guessBoard(pi)
		out = append(out, pi)
	}
	return out
}

// uf2Mounts are where a Pico in BOOTSEL mode appears as a drive.
func uf2Mounts() []string {
	if runtime.GOOS == "darwin" {
		m, _ := filepath.Glob("/Volumes/*")
		return m
	}
	var out []string
	for _, pat := range []string{"/media/*/*", "/run/media/*/*", "/media/*", "/mnt/*"} {
		m, _ := filepath.Glob(pat)
		out = append(out, m...)
	}
	return out
}

func listUF2Drives() []uf2Drive {
	var out []uf2Drive
	seen := map[string]bool{}
	for _, dir := range uf2Mounts() {
		b, err := os.ReadFile(filepath.Join(dir, "INFO_UF2.TXT"))
		if err != nil || len(b) > 4096 || seen[dir] {
			continue
		}
		seen[dir] = true
		d := parseInfoUF2(dir, string(b))
		d.Label = filepath.Base(dir)
		d.Letter = d.Label // the name shown in the UI
		out = append(out, d)
	}
	return out
}

func launchWindow(url string) error {
	base, err := os.UserConfigDir()
	if err != nil {
		base = os.TempDir()
	}
	profile := filepath.Join(base, "nrmluvdev", "Flashie", "window")
	os.MkdirAll(profile, 0o755)
	return appwindow.Open(url, profile)
}

func showError(title, msg string) { fmt.Fprintln(os.Stderr, title+": "+msg) }
