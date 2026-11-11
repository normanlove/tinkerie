//go:build linux || darwin

package flashie

import (
	"os/exec"
	"runtime"
	"strings"
	"tinkerie/shared/droid"
)

func pickImageFile() (string, error) {
	if droid.Active() {
		return droid.PickFile("Choose an image to write", []string{".img", ".iso", ".raw", ".wic", ".xz", ".gz", ".bz2", ".zip", ".zst"})
	}
	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		cmd = exec.Command("osascript", "-e", `POSIX path of (choose file with prompt "Choose an image to write")`)
	} else if p, err := exec.LookPath("zenity"); err == nil {
		cmd = exec.Command(p, "--file-selection", "--title=Choose an image to write",
			"--file-filter=Disk images | *.img *.iso *.raw *.wic *.xz *.gz *.bz2 *.zip *.IMG *.ISO", "--file-filter=All files | *")
	} else if p, err := exec.LookPath("kdialog"); err == nil {
		cmd = exec.Command(p, "--getopenfilename", ".", "*.img *.iso *.raw *.wic *.xz *.gz *.bz2 *.zip|Disk images")
	} else {
		return "", errNoPicker
	}
	out, err := cmd.Output()
	if err != nil {
		return "", nil // cancelled
	}
	return strings.TrimRight(string(out), "\r\n"), nil
}
