// Tracie — serial monitor + logic analyser on one shared timeline.
// Part of the free nrmluvdev hobbyist tool suite (Wirie · Scopie · Makie · Tracie).
//
// Copyright (c) 2026 nrmluvdev. Free to use and share.
//
// Architecture: a single Go binary that serves an embedded web UI on
// 127.0.0.1 and opens it in a chromeless Edge/Chrome "app" window. All
// hardware access (Win32 serial, WinUSB for fx2lafw analysers) lives in Go;
// the UI talks to it over a small token-protected JSON/SSE API.
package tracie

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

//go:embed web
var webFS embed.FS

const (
	appName     = "Tracie"
	appVersion  = "1.0.0"
	appAuthor   = "nrmluvdev"
	defaultPort = 47822 // Tinkerie ports: see PORTING.md
)

func Main() {
	noWindow := flag.Bool("nowindow", false, "don't open a window (development)")
	port := flag.Int("port", defaultPort, "local port for the UI")
	flag.Parse()

	dataDir := appDataDir()
	_ = os.MkdirAll(dataDir, 0o755)
	if f, err := os.OpenFile(filepath.Join(dataDir, "tracie.log"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644); err == nil {
		log.SetOutput(io.MultiWriter(f, os.Stderr))
	}
	log.Printf("%s %s by %s starting", appName, appVersion, appAuthor)

	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *port))
	if err != nil {
		// Probably already running: just open another window onto it.
		if pingExisting(*port) {
			log.Printf("already running, opening window")
			openWindow(fmt.Sprintf("http://127.0.0.1:%d/", *port), dataDir)
			return
		}
		ln, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			fatal("Could not start the local UI server: " + err.Error())
		}
	}
	actualPort := ln.Addr().(*net.TCPAddr).Port

	app := newApp(dataDir, actualPort)
	srv := &http.Server{Handler: app.routes(), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Printf("server: %v", err)
		}
	}()

	url := fmt.Sprintf("http://127.0.0.1:%d/", actualPort)
	log.Printf("UI at %s", url)
	if !*noWindow {
		openWindow(url, dataDir)
		app.watchClients()
	} else {
		select {}
	}
	app.shutdown()
}

func pingExisting(port int) bool {
	c := http.Client{Timeout: 800 * time.Millisecond}
	r, err := c.Get(fmt.Sprintf("http://127.0.0.1:%d/api/ping", port))
	if err != nil {
		return false
	}
	defer r.Body.Close()
	b, _ := io.ReadAll(r.Body)
	return strings.Contains(string(b), "tracie")
}

func appDataDir() string {
	if d, err := os.UserConfigDir(); err == nil {
		return filepath.Join(d, appName)
	}
	return filepath.Join(os.TempDir(), appName)
}

func logsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}
	d := filepath.Join(home, "Documents", appName)
	_ = os.MkdirAll(d, 0o755)
	return d
}

// ---------------------------------------------------------------- app

type App struct {
	dataDir string
	port    int
	token   string

	hub    *Hub
	serial *SerialManager
	logic  *LogicManager

	cfgMu sync.Mutex

	clientsMu   sync.Mutex
	clients     int
	everClient  bool
	lastClientT time.Time
}

func newApp(dataDir string, port int) *App {
	tok := make([]byte, 16)
	_, _ = rand.Read(tok)
	a := &App{dataDir: dataDir, port: port, token: hex.EncodeToString(tok), hub: newHub()}
	a.serial = newSerialManager(a.hub)
	a.logic = newLogicManager(a.hub, dataDir, a.serial)
	a.lastClientT = time.Now()
	return a
}

func (a *App) shutdown() {
	a.serial.Close()
	a.logic.Stop()
	log.Printf("bye")
}

// watchClients blocks until the UI window has gone away.
func (a *App) watchClients() {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	start := time.Now()
	for range t.C {
		a.clientsMu.Lock()
		n, ever, last := a.clients, a.everClient, a.lastClientT
		a.clientsMu.Unlock()
		if n > 0 {
			continue
		}
		if ever && time.Since(last) > 6*time.Second {
			return
		}
		if !ever && time.Since(start) > 3*time.Minute {
			return
		}
	}
}

func (a *App) clientDelta(d int) {
	a.clientsMu.Lock()
	a.clients += d
	if d > 0 {
		a.everClient = true
	}
	a.lastClientT = time.Now()
	a.clientsMu.Unlock()
}

// ---------------------------------------------------------------- HTTP

func (a *App) routes() http.Handler {
	mux := http.NewServeMux()
	sub, _ := fs.Sub(webFS, "web")
	static := http.FileServer(http.FS(sub))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			b, err := fs.ReadFile(sub, "index.html")
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			html := strings.Replace(string(b), "%%TOKEN%%", a.token, 1)
			html = strings.ReplaceAll(html, "%%VERSION%%", appVersion)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")
			_, _ = w.Write([]byte(html))
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		static.ServeHTTP(w, r)
	})
	mux.HandleFunc("/api/ping", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"app":"tracie"}`))
	})

	api := http.NewServeMux()
	api.HandleFunc("/api/events", a.handleEvents)
	api.HandleFunc("/api/config", a.handleConfig)
	api.HandleFunc("/api/openfolder", a.handleOpenFolder)
	a.serial.routes(api)
	a.logic.routes(api)
	mux.Handle("/api/", a.guard(api))
	return mux
}

// guard blocks DNS-rebinding and cross-site requests: the Host must be
// loopback and every call must carry the per-launch token.
func (a *App) guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		if host != "127.0.0.1" && host != "localhost" {
			http.Error(w, "forbidden host", http.StatusForbidden)
			return
		}
		tok := r.Header.Get("X-Tracie-Token")
		if tok == "" {
			tok = r.URL.Query().Get("token")
		}
		if tok != a.token {
			http.Error(w, "bad token", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *App) handleEvents(w http.ResponseWriter, r *http.Request) {
	fl, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "no streaming", 500)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Accel-Buffering", "no")

	a.clientDelta(1)
	defer a.clientDelta(-1)

	ch := a.hub.subscribe()
	defer a.hub.unsubscribe(ch)

	fmt.Fprintf(w, "event: hello\ndata: {\"version\":%q}\n\n", appVersion)
	fmt.Fprintf(w, "event: serialState\ndata: %s\n\n", mustJSON(a.serial.State()))
	fmt.Fprintf(w, "event: logicState\ndata: %s\n\n", mustJSON(a.logic.State()))
	fl.Flush()

	ping := time.NewTicker(10 * time.Second)
	defer ping.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case m, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", m.event, m.data)
			// Drain whatever else is queued before flushing.
		drain:
			for i := 0; i < 64; i++ {
				select {
				case m2, ok2 := <-ch:
					if !ok2 {
						return
					}
					fmt.Fprintf(w, "event: %s\ndata: %s\n\n", m2.event, m2.data)
				default:
					break drain
				}
			}
			fl.Flush()
		case <-ping.C:
			fmt.Fprint(w, ": ping\n\n")
			fl.Flush()
		}
	}
}

func (a *App) handleConfig(w http.ResponseWriter, r *http.Request) {
	path := filepath.Join(a.dataDir, "config.json")
	a.cfgMu.Lock()
	defer a.cfgMu.Unlock()
	switch r.Method {
	case http.MethodGet:
		b, err := os.ReadFile(path)
		if err != nil {
			b = []byte("{}")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	case http.MethodPut, http.MethodPost:
		b, err := io.ReadAll(io.LimitReader(r.Body, 4<<20))
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		tmp := path + ".tmp"
		if err := os.WriteFile(tmp, b, 0o644); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		_ = os.Rename(tmp, path)
		writeJSON(w, map[string]any{"ok": true})
	default:
		http.Error(w, "method", 405)
	}
}

func (a *App) handleOpenFolder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Which string `json:"which"`
	}
	_ = readJSON(r, &req)
	dir := logsDir()
	switch req.Which {
	case "firmware":
		dir = filepath.Join(a.dataDir, "firmware")
		_ = os.MkdirAll(dir, 0o755)
	case "data":
		dir = a.dataDir
	}
	openFolder(dir)
	writeJSON(w, map[string]any{"ok": true, "path": dir})
}
