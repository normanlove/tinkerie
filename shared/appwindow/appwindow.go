// Package appwindow opens a Tinkerie app's page in a chromeless browser
// "app" window on macOS and Linux (including Raspberry Pi OS), falling back to
// the default browser. Windows apps start the browser with their own code,
// but every app uses Maximised so all windows open maximised.
// Created by nrmluvdev for Tinkerie.
package appwindow

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Open shows url in its own, maximised window. profile is a folder for the
// window's browser profile (so the app keeps its own settings).
func Open(url, profile string) error {
	args := append([]string{"--app=" + url, "--user-data-dir=" + profile,
		"--no-first-run", "--no-default-browser-check"}, Maximised(profile)...)
	if runtime.GOOS == "darwin" {
		for _, app := range []string{"Google Chrome", "Microsoft Edge", "Brave Browser", "Chromium"} {
			if !exists("/Applications/"+app+".app") && !exists(filepath.Join(os.Getenv("HOME"), "Applications", app+".app")) {
				continue
			}
			if exec.Command("open", append([]string{"-na", app, "--args"}, args...)...).Run() == nil {
				return nil
			}
		}
		return exec.Command("open", url).Start()
	}
	for _, b := range []string{"chromium", "chromium-browser", "google-chrome", "google-chrome-stable", "microsoft-edge", "brave-browser"} {
		if p, err := exec.LookPath(b); err == nil {
			cmd := exec.Command(p, args...)
			if cmd.Start() == nil {
				go cmd.Wait()
				return nil
			}
		}
	}
	return exec.Command("xdg-open", url).Start()
}

// Maximised returns the browser switches that open an app window maximised.
// The browser remembers each profile's last window size and would bring it
// back, so the profile's saved window is marked maximised too. On a Mac,
// where Chrome may ignore --start-maximized, the window is also asked for
// more than the screen, which the browser shrinks to fill the screen.
func Maximised(profile string) []string {
	markMaximised(profile)
	if runtime.GOOS == "darwin" {
		return []string{"--start-maximized", "--window-position=0,0", "--window-size=10000,10000"}
	}
	return []string{"--start-maximized"}
}

func markMaximised(profile string) {
	if profile == "" {
		return
	}
	p := filepath.Join(profile, "Default", "Preferences")
	b, err := os.ReadFile(p)
	if err != nil {
		return
	}
	var prefs map[string]any
	if json.Unmarshal(b, &prefs) != nil {
		return
	}
	br, ok := prefs["browser"].(map[string]any)
	if !ok {
		return
	}
	if wp, ok := br["window_placement"].(map[string]any); ok {
		wp["maximized"] = true
	}
	if aw, ok := br["app_window_placement"].(map[string]any); ok {
		for _, v := range aw {
			if m, ok := v.(map[string]any); ok {
				m["maximized"] = true
			}
		}
	}
	if out, err := json.Marshal(prefs); err == nil {
		os.WriteFile(p, out, 0o600)
	}
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
