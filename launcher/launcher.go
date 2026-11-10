// Package launcher is Tinkerie's start window: one tile per app. Picking a tile
// starts that app as its own process (the same program with --app NAME).
//
// The window is a small local web page shown in a chromeless browser window,
// like the apps themselves. The launcher exits a few seconds after its window
// is closed; apps it started keep running. Created by nrmluvdev.
package launcher

import (
	"embed"
	"encoding/json"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// Version is the suite version shown in the launcher.
const Version = "0.12.1-alpha"

// App describes one tile in the launcher.
type App struct {
	Name    string `json:"name"`           // command name: tinkerie --app <name>
	Title   string `json:"title"`          // display name
	About   string `json:"about"`          // one line
	Version string `json:"version"`        // app version
	Icon    string `json:"icon"`           // file in web/icons
	Note    string `json:"note,omitempty"` // what is limited on this system
	Off     bool   `json:"off,omitempty"`  // can't run on this system yet
}

// Apps is the suite, in launcher order.
var Apps = []App{
	{"wirie", "Wirie", "Circuit board designer, with Keebie keyboards, Boardie, Calcie, Pinnie and Powerie", "0.11.1-alpha", "wirie.png", "", false},
	{"makie", "Makie", "3D-printing part designer. Builds cases for Wirie boards and keyboards", "0.11.0-alpha", "makie.png", "", false},
	{"tunie", "Tunie", "3D printer tuning and calibration", "0.5.0-alpha", "tunie.png", "", false},
	{"flashie", "Flashie", "Firmware flasher for ESP32, Pico, Arduino and STM32, keyboard firmware maker, and SD card / USB image writer", "0.8.0-alpha", "flashie.svg", "", false},
	{"scopie", "Scopie", "Oscilloscope: Hantek 6022, network scopes and more", "0.9.1-alpha", "scopie.png", "", false},
	{"meterie", "Meterie", "PC multimeter: HoldPeak, UNI-T, OWON, Brymen, Fluke…", "0.6.1-alpha", "meterie.png", "", false},
	{"probie", "Probie", "Serial monitor and logic analyser that explains captures in plain words: its own, and Scopie's and Meterie's", "0.6.0-alpha", "probie.svg", "", false},
	{"codie", "Codie", "Code explainer: what each line of code does and why", "0.3.0-alpha", "codie.svg", "", false},
}

// What each app can't do yet on this system (see PORTING.md).
var limits = map[string]map[string]string{
	"darwin": {
		"scopie":  "USB scopes: not on Mac yet",
		"probie":  "Logic analyser: not on Mac yet",
		"meterie": "USB-serial meters only on Mac",
	},
	"linux": {
		"meterie": "USB-serial meters only on Linux",
	},
}

// forThisSystem returns the app list with notes for what doesn't work here.
func forThisSystem() []App {
	out := append([]App(nil), Apps...)
	for i, a := range out {
		out[i].Note = limits[runtime.GOOS][a.Name]
	}
	return out
}

// Names lists the app command names.
func Names() []string {
	var n []string
	for _, a := range Apps {
		n = append(n, a.Name)
	}
	return n
}

//go:embed web
var webFS embed.FS

// Run shows the launcher and returns once its window has been closed.
func Run() {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	// The page keeps one /api/live connection open. When the window closes the
	// browser drops it straight away; a minimised window keeps it open.
	var mu sync.Mutex
	live, opened, lastGone := 0, false, time.Now()

	web, _ := fs.Sub(webFS, "web")
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(web)))
	mux.HandleFunc("/api/apps", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"version": Version, "author": "nrmluvdev", "apps": forThisSystem()})
	})
	mux.HandleFunc("/api/live", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		live, opened = live+1, true
		mu.Unlock()
		defer func() { mu.Lock(); live, lastGone = live-1, time.Now(); mu.Unlock() }()
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		f, _ := w.(http.Flusher)
		t := time.NewTicker(20 * time.Second)
		defer t.Stop()
		for {
			if _, err := io.WriteString(w, ": alive\n\n"); err != nil {
				return
			}
			if f != nil {
				f.Flush()
			}
			select {
			case <-r.Context().Done():
				return
			case <-t.C:
			}
		}
	})
	mux.HandleFunc("/api/launch", func(w http.ResponseWriter, r *http.Request) {
		// Only the launcher page may start apps: a POST with a JSON body, which
		// other web pages can't send to this port without a CORS preflight.
		if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, "use POST with JSON", http.StatusMethodNotAllowed)
			return
		}
		var req struct{ App string }
		if json.NewDecoder(r.Body).Decode(&req) != nil || !known(req.App) {
			http.Error(w, "unknown app", http.StatusBadRequest)
			return
		}
		for _, a := range forThisSystem() {
			if a.Name == req.App && a.Off {
				http.Error(w, a.Title+": "+a.Note, http.StatusConflict)
				return
			}
		}
		if err := start(req.App); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write([]byte(`{"ok":true}`))
	})
	go http.Serve(ln, mux)

	url := "http://" + ln.Addr().String() + "/"
	log.Println("Tinkerie launcher at", url)
	openWindow(url, dataDir())

	// Quit once the window has been gone for 3 s (a reload reconnects sooner),
	// or after a minute if the window never opened.
	began := time.Now()
	for {
		time.Sleep(500 * time.Millisecond)
		mu.Lock()
		gone := opened && live == 0 && time.Since(lastGone) > 3*time.Second
		never := !opened && time.Since(began) > time.Minute
		mu.Unlock()
		if gone || never {
			return
		}
	}
}

func known(name string) bool {
	for _, a := range Apps {
		if a.Name == name {
			return true
		}
	}
	return false
}

// start runs one app as a separate process of this same program.
func start(name string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, "--app", name)
	cmd.Dir = filepath.Dir(exe)
	detach(cmd)
	if err := cmd.Start(); err != nil {
		return err
	}
	log.Printf("started %s (pid %d)", name, cmd.Process.Pid)
	go cmd.Wait()
	return nil
}

// dataDir holds the launcher window's browser profile.
func dataDir() string {
	d, err := os.UserConfigDir()
	if err != nil {
		d = os.TempDir()
	}
	d = filepath.Join(d, "Tinkerie")
	os.MkdirAll(d, 0o755)
	return d
}
