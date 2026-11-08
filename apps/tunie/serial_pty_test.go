//go:build linux

package tunie

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestOpenSerialSetsBaud opens a pseudo-terminal like a printer's USB port and
// checks the speed is set while it stays open. TUNIE_TEST_PTY names the device.
func TestOpenSerialSetsBaud(t *testing.T) {
	dev := os.Getenv("TUNIE_TEST_PTY")
	if dev == "" {
		t.Skip("set TUNIE_TEST_PTY to a pseudo-terminal to run this test")
	}
	p, err := openSerial(dev, 115200)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	cmd := exec.Command("stty", "-a")
	cmd.Stdin = p.(*os.File)
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "speed 115200") || !strings.Contains(string(out), "-echo") {
		t.Errorf("port settings after open: %s", out)
	}
}
