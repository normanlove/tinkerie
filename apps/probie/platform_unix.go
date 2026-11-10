//go:build linux || darwin

package probie

// Probie on Linux, Raspberry Pi OS and macOS: real serial ports through
// shared/serialport. The USB logic analyser is in usb_linux.go (usbfs); on
// macOS it would need IOKit, so usb_darwin.go reports that clearly.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"tinkerie/shared/appwindow"
	"tinkerie/shared/serialport"
)

func listPorts() []PortInfo {
	var out []PortInfo
	for _, in := range serialport.List() {
		pi := PortInfo{Name: in.Path, Desc: strings.TrimSpace(in.Manufacturer + " " + in.Product)}
		if pi.Desc == "" {
			pi.Desc = in.Description
		}
		if in.USB && in.VID != 0 {
			pi.VID, pi.PID = fmt.Sprintf("%04X", in.VID), fmt.Sprintf("%04X", in.PID)
			pi.Chip = chipFromIDs(pi.VID, pi.PID)
		}
		out = append(out, pi)
	}
	return out
}

func openPort(name string, cfg PortConfig) (Port, error) {
	parity := map[string]byte{"": 'N', "none": 'N', "odd": 'O', "even": 'E', "mark": 'M', "space": 'S'}[strings.ToLower(cfg.Parity)]
	if parity == 0 {
		return nil, fmt.Errorf("unknown parity %q", cfg.Parity)
	}
	p, err := serialport.Open(name, serialport.Config{
		Baud: cfg.Baud, DataBits: cfg.DataBits, Parity: parity, StopBits: cfg.StopBits, DTR: cfg.DTR, RTS: cfg.RTS,
	})
	if err != nil {
		return nil, err
	}
	return p, nil
}

func openFolder(dir string) {
	if runtime.GOOS == "darwin" {
		_ = exec.Command("open", dir).Start()
		return
	}
	_ = exec.Command("xdg-open", dir).Start()
}

func openWindow(url, dataDir string) {
	_ = appwindow.Open(url, filepath.Join(dataDir, "window"))
}

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, "Probie: "+msg)
	os.Exit(1)
}
