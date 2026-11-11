//go:build !windows

// Tunie — platform_other.go: Linux / macOS versions of the platform pieces,
// so the source builds everywhere for porting. Serial uses the device file
// directly (configure the baud rate with `stty` beforehand if needed).
// Part of the nrmluvdev maker suite. Created by nrmluvdev.
package tunie

import (
	"tinkerie/shared/appwindow"
	"tinkerie/shared/droid"
	"tinkerie/shared/serialport"

	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

func osName() string { return runtime.GOOS }

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}

func openAppWindow(url, dataDir string) error {
	if droid.Active() {
		return droid.OpenWindow(url, "Tunie")
	}
	for _, b := range []string{"microsoft-edge", "google-chrome", "chromium", "chromium-browser"} {
		if p, err := exec.LookPath(b); err == nil {
			cmd := exec.Command(p, append(appwindow.Maximised(filepath.Join(dataDir, "window")), "--app="+url, "--user-data-dir="+filepath.Join(dataDir, "window"), "--no-first-run")...)
			if cmd.Start() == nil {
				go cmd.Wait()
				return nil
			}
		}
	}
	return openURL(url)
}

func openURL(u string) error {
	if droid.Active() {
		return droid.Browse(u)
	}
	if runtime.GOOS == "darwin" {
		return exec.Command("open", u).Start()
	}
	return exec.Command("xdg-open", u).Start()
}

func openFolder(p string) error {
	if droid.Active() {
		return droid.ShowFolder(p)
	}
	if st, err := os.Stat(p); err == nil && !st.IsDir() {
		p = filepath.Dir(p)
	}
	_ = os.MkdirAll(p, 0o755)
	return openURL(p)
}

type Drive struct {
	Path      string `json:"path"`
	Label     string `json:"label"`
	Removable bool   `json:"removable"`
}

func listDrives() []Drive {
	if droid.Active() {
		return nil // Android keeps drives to itself
	}
	var out []Drive
	for _, base := range []string{"/media", "/run/media", "/Volumes"} {
		ents, _ := os.ReadDir(base)
		for _, e := range ents {
			p := filepath.Join(base, e.Name())
			if sub, err := os.ReadDir(p); err == nil && base != "/Volumes" {
				for _, s := range sub {
					if s.IsDir() {
						out = append(out, Drive{Path: filepath.Join(p, s.Name()), Label: s.Name(), Removable: true})
					}
				}
				continue
			}
			out = append(out, Drive{Path: p, Label: e.Name(), Removable: true})
		}
	}
	return out
}

func listSerialPorts() []string {
	if droid.Active() {
		var out []string
		for _, p := range serialport.List() {
			out = append(out, p.Path)
		}
		return out
	}
	var out []string
	// Linux, Raspberry Pi and macOS. On a Mac only the cu.* devices: opening a
	// tty.* device waits for a carrier signal and can hang.
	for _, pat := range []string{"/dev/ttyUSB*", "/dev/ttyACM*", "/dev/cu.usb*", "/dev/cu.wchusbserial*", "/dev/cu.SLAB_USBtoUART*"} {
		m, _ := filepath.Glob(pat)
		out = append(out, m...)
	}
	sort.Strings(out)
	return out
}

func openSerial(name string, baud int) (serialPort, error) {
	if droid.Active() {
		return serialport.Open(name, serialport.Config{Baud: baud})
	}
	if !strings.HasPrefix(name, "/dev/") {
		name = "/dev/" + name
	}
	f, err := os.OpenFile(name, os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	// Set the speed on our open handle (stty works on its standard input on both
	// Linux and macOS), so the settings last while the port stays open.
	cmd := exec.Command("stty", fmt.Sprint(baud), "raw", "-echo")
	cmd.Stdin = f
	if out, err := cmd.CombinedOutput(); err != nil {
		f.Close()
		return nil, fmt.Errorf("couldn't set %s to %d baud: %v %s", name, baud, err, strings.TrimSpace(string(out)))
	}
	return f, nil
}
