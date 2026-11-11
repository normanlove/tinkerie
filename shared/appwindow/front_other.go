//go:build !windows

package appwindow

// front_other.go: bring a newly opened app window to the front on Linux and
// Raspberry Pi OS (macOS's "open" already brings new windows forward).
// Uses wmctrl or xdotool when they're installed (X11); without them the
// desktop decides. Created by nrmluvdev for Tinkerie.

import (
	"os/exec"
	"runtime"
	"strings"
	"time"
	"tinkerie/shared/droid"
)

func listWindows() (map[string]bool, string) {
	out := map[string]bool{}
	if p, err := exec.LookPath("wmctrl"); err == nil {
		b, _ := exec.Command(p, "-l").Output()
		for _, l := range strings.Split(string(b), "\n") {
			if f := strings.Fields(l); len(f) > 0 {
				out[f[0]] = true
			}
		}
		return out, "wmctrl"
	}
	if p, err := exec.LookPath("xdotool"); err == nil {
		b, _ := exec.Command(p, "search", "--onlyvisible", "--class", "chrom|Chrom|brave|edge").Output()
		for _, f := range strings.Fields(string(b)) {
			out[f] = true
		}
		return out, "xdotool"
	}
	return nil, ""
}

// WatchNewWindow notes the windows open now; the function it returns waits for
// the app's new window and brings it to the front.
func WatchNewWindow() func() {
	if runtime.GOOS != "linux" || droid.Active() {
		return func() {}
	}
	before, tool := listWindows()
	if tool == "" {
		return func() {}
	}
	return func() {
		go func() {
			for end := time.Now().Add(20 * time.Second); time.Now().Before(end); {
				time.Sleep(200 * time.Millisecond)
				now, _ := listWindows()
				for id := range now {
					if before[id] {
						continue
					}
					time.Sleep(250 * time.Millisecond)
					if tool == "wmctrl" {
						exec.Command("wmctrl", "-i", "-a", id).Run()
					} else {
						exec.Command("xdotool", "windowactivate", id).Run()
					}
					return
				}
			}
		}()
	}
}
