// Package droid connects Tinkerie's Go apps to the Android app that runs
// them. On Android every app still serves its page on 127.0.0.1, but the
// things a desktop does for it (app windows, file pickers, opening links and
// folders, USB permission) are done by the Android side, which listens on an
// abstract Unix socket named in TINKERIE_BRIDGE.
//
// Each call is one connection: a JSON request line, then a JSON reply line.
// A reply can carry one file descriptor (an opened USB device). Only processes
// running as the Android app's own user may connect; the Java side checks.
//
// Everywhere else Active is false and nothing here is used.
// Created by nrmluvdev for Tinkerie.
package droid

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func init() {
	// Go on Android uses UTC unless told otherwise; the app passes the phone's zone
	if tz := os.Getenv("TINKERIE_TZ"); tz != "" && Active() {
		if loc, err := time.LoadLocation(tz); err == nil {
			time.Local = loc
		}
	}
}

// Active reports whether this process was started by the Android app.
func Active() bool { return os.Getenv("TINKERIE_BRIDGE") != "" }

// App is this process's app name ("launcher", "wirie", …).
func App() string {
	for i, a := range os.Args {
		if a == "--app" && i+1 < len(os.Args) {
			return os.Args[i+1]
		}
		if strings.HasPrefix(a, "--app=") {
			return strings.TrimPrefix(a, "--app=")
		}
	}
	return "launcher"
}

// OpenWindow shows url (this app's page) in its own Android window.
func OpenWindow(url, title string) error {
	_, err := Call("window", map[string]any{"url": url, "title": title}, nil, 10*time.Second)
	return err
}

// Browse opens a web page (or a file) in the phone's browser or viewer.
func Browse(url string) error {
	_, err := Call("browse", map[string]any{"url": url}, nil, 10*time.Second)
	return err
}

// ShowFolder opens the Files app at a folder inside Tinkerie's documents.
func ShowFolder(path string) error {
	if st, err := os.Stat(path); err == nil && !st.IsDir() {
		path = filepath.Dir(path)
	}
	_, err := Call("folder", map[string]any{"path": path}, nil, 10*time.Second)
	return err
}

// Toast shows a short message.
func Toast(text string) { Call("toast", map[string]any{"text": text}, nil, 5*time.Second) }

// ErrCancelled is returned when the person closes a picker without choosing.
var ErrCancelled = errors.New("cancelled")

// PickFile lets the person choose a file; the Android side copies it to a
// private folder and returns that path. exts like ".uf2" narrow the choice
// where the phone's picker can.
func PickFile(title string, exts []string) (string, error) {
	var r struct {
		Path string `json:"path"`
	}
	if _, err := Call("pick", map[string]any{"title": title, "exts": exts}, &r, 30*time.Minute); err != nil {
		return "", err
	}
	if r.Path == "" {
		return "", ErrCancelled
	}
	return r.Path, nil
}

// DocsDir is where an app keeps the files people make. On Android it's
// inside Tinkerie's own storage, shown in the Files app as "Tinkerie".
func DocsDir(app string) string {
	home, _ := os.UserHomeDir()
	d := filepath.Join(home, "Documents", app)
	os.MkdirAll(d, 0o755)
	return d
}

// SavePath is where a "Save as…" lands on Android: the app's documents
// folder, with a number added if the name is taken.
func SavePath(app, name string) string {
	d := DocsDir(app)
	name = filepath.Base(name)
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	p := filepath.Join(d, name)
	for i := 2; i < 1000; i++ {
		if _, err := os.Stat(p); err != nil {
			break
		}
		p = filepath.Join(d, fmt.Sprintf("%s (%d)%s", stem, i, ext))
	}
	return p
}

// ErrNeedsComputer explains features Android can't offer.
func ErrNeedsComputer(what string) error {
	return fmt.Errorf("%s needs Tinkerie on a computer (Windows, Mac, Linux or Raspberry Pi): Android can't run the tools it uses", what)
}
