//go:build !windows && !linux && !darwin

package tracie

import (
	"tinkerie/shared/appwindow"

	"errors"
	"os/exec"
	"runtime"
)

// Non-Windows builds are for development and future ports: the demo
// devices work, real hardware access is Windows-only for now.

func listPorts() []PortInfo { return nil }

func openPort(name string, cfg PortConfig) (Port, error) {
	return nil, errors.New("real serial ports are only supported on Windows in this build")
}

func listUSBDevices() []usbInfo { return nil }

func openUSBDevice(path string) (usbDevice, error) {
	return nil, errors.New("USB access is only supported on Windows in this build")
}

func openFolder(dir string) {
	if runtime.GOOS == "darwin" {
		_ = exec.Command("open", dir).Start()
		return
	}
	_ = exec.Command("xdg-open", dir).Start()
}

func openWindow(url, dataDir string) {
	for _, b := range []string{"google-chrome", "chromium", "chromium-browser", "microsoft-edge"} {
		if p, err := exec.LookPath(b); err == nil {
			_ = exec.Command(p, append(appwindow.Maximised(dataDir+"/window"), "--app="+url, "--user-data-dir="+dataDir+"/window")...).Start()
			return
		}
	}
	_ = exec.Command("xdg-open", url).Start()
}

func fatal(msg string) { panic(msg) }
