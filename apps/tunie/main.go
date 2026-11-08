// Tunie — 3D printer calibration helper
// Part of the free nrmluvdev maker suite (Wirie, Scopie, Makie, Flashie, Tunie).
// Created by nrmluvdev.
//
// Tunie is a single executable: a small local web server (bound to
// 127.0.0.1 only) with the whole interface embedded, shown in its own
// app window (Microsoft Edge / Chrome "app mode"). It closes itself when
// the window is closed.
package tunie

import (
	"embed"
	"encoding/base64"
	"encoding/json"
	"errors"
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
	appName     = "Tunie"
	appVersion  = "1.0.0"
	appAuthor   = "nrmluvdev"
	defaultPort = 47821
)

// LaunchInfo carries command-line context, e.g. when Makie opens Tunie.
type LaunchInfo struct {
	From     string `json:"from"`
	Printer  string `json:"printer"`
	Filament string `json:"filament"`
	Page     string `json:"page"`
	Seq      int64  `json:"seq"`
}

type App struct {
	mu        sync.Mutex
	launch    LaunchInfo
	clients   map[chan string]bool
	ever      bool
	lastGone  time.Time
	started   time.Time
	url       string
	noBrowser bool
	store     *Store
	conn      *ConnManager
	host      string
}

func Main() {
	port := flag.Int("port", defaultPort, "local port for the interface")
	noBrowser := flag.Bool("no-browser", false, "don't open a window (use your own browser)")
	from := flag.String("from", "", "which app opened Tunie (e.g. makie)")
	printer := flag.String("printer", "", "printer profile name to select")
	filament := flag.String("filament", "", "filament name to select")
	page := flag.String("page", "", "page or test to open (e.g. tolerance)")
	flag.Parse()

	launch := LaunchInfo{From: *from, Printer: *printer, Filament: *filament, Page: *page, Seq: time.Now().UnixNano()}

	store, err := NewStore()
	if err != nil {
		fatal("Tunie could not create its data folder:\n" + err.Error())
	}
	setupLog(store.dataDir)

	// Already running? Hand over to the existing window and exit.
	if handOver(*port, launch) {
		return
	}

	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *port))
	if err != nil {
		ln, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			fatal("Tunie could not start its local server:\n" + err.Error())
		}
	}
	app := &App{
		launch:    launch,
		clients:   map[chan string]bool{},
		started:   time.Now(),
		noBrowser: *noBrowser,
		store:     store,
		host:      ln.Addr().String(),
	}
	app.conn = NewConnManager()
	app.url = "http://" + app.host + "/"
	log.Printf("Tunie %s listening on %s", appVersion, app.url)
	_ = os.WriteFile(filepath.Join(store.dataDir, "tunie.port"), []byte(app.host), 0o644)

	srv := &http.Server{Handler: app.routes(), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("server: %v", err)
		}
	}()
	if !*noBrowser {
		if err := openAppWindow(app.url, store.dataDir); err != nil {
			log.Printf("window: %v", err)
		}
	}
	app.waitForExit()
	app.conn.Close()
	_ = os.Remove(filepath.Join(store.dataDir, "tunie.port"))
	log.Printf("Tunie closed")
}

func setupLog(dir string) {
	p := filepath.Join(dir, "tunie.log")
	if st, err := os.Stat(p); err == nil && st.Size() > 1<<20 {
		_ = os.Remove(p)
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err == nil {
		log.SetOutput(io.MultiWriter(f, os.Stderr))
	}
}

// handOver asks an already-running Tunie to show itself; returns true if it did.
func handOver(port int, l LaunchInfo) bool {
	c := &http.Client{Timeout: 700 * time.Millisecond}
	base := fmt.Sprintf("http://127.0.0.1:%d", port)
	resp, err := c.Get(base + "/api/ping")
	if err != nil {
		return false
	}
	var p map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&p)
	resp.Body.Close()
	if p["app"] != appName {
		return false
	}
	b, _ := json.Marshal(l)
	req, _ := http.NewRequest("POST", base+"/api/activate", strings.NewReader(string(b)))
	req.Header.Set("X-Tunie", "1")
	req.Header.Set("Content-Type", "application/json")
	if r2, err := c.Do(req); err == nil {
		r2.Body.Close()
	}
	return true
}

// waitForExit returns once the window has been closed for a few seconds.
func (a *App) waitForExit() {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for range t.C {
		a.mu.Lock()
		n, ever, gone := len(a.clients), a.ever, a.lastGone
		a.mu.Unlock()
		if a.noBrowser {
			continue
		}
		if ever && n == 0 && time.Since(gone) > 6*time.Second && !a.conn.Busy() {
			return
		}
		if !ever && time.Since(a.started) > 10*time.Minute {
			return
		}
	}
}

func (a *App) broadcast(event, data string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	msg := "event: " + event + "\ndata: " + data + "\n\n"
	for c := range a.clients {
		select {
		case c <- msg:
		default:
		}
	}
}

// ---------------------------------------------------------------- HTTP

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func readJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(io.LimitReader(r.Body, 64<<20)).Decode(v)
}

// guard blocks other websites from talking to Tunie: requests must target
// our own host name and API calls must carry a custom header (which browsers
// will not send cross-origin without a CORS approval we never give).
func (a *App) guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := r.Host
		port := a.host[strings.LastIndex(a.host, ":"):]
		if h != a.host && h != "localhost"+port && h != "127.0.0.1"+port {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") && r.URL.Path != "/api/ping" && r.URL.Path != "/api/events" {
			if r.Header.Get("X-Tunie") != "1" {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (a *App) routes() http.Handler {
	mux := http.NewServeMux()
	sub, _ := fs.Sub(webFS, "web")
	files := http.FileServer(http.FS(sub))
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		files.ServeHTTP(w, r)
	}))

	mux.HandleFunc("/api/ping", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"app": appName, "version": appVersion})
	})

	mux.HandleFunc("/api/activate", func(w http.ResponseWriter, r *http.Request) {
		var l LaunchInfo
		_ = readJSON(r, &l)
		l.Seq = time.Now().UnixNano()
		a.mu.Lock()
		a.launch = l
		n := len(a.clients)
		a.mu.Unlock()
		b, _ := json.Marshal(l)
		a.broadcast("activate", string(b))
		if n == 0 {
			_ = openAppWindow(a.url, a.store.dataDir)
		}
		writeJSON(w, map[string]bool{"ok": true})
	})

	mux.HandleFunc("/api/events", func(w http.ResponseWriter, r *http.Request) {
		fl, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", 500)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-store")
		ch := make(chan string, 16)
		a.mu.Lock()
		a.clients[ch] = true
		a.ever = true
		a.mu.Unlock()
		defer func() {
			a.mu.Lock()
			delete(a.clients, ch)
			a.lastGone = time.Now()
			a.mu.Unlock()
		}()
		fmt.Fprint(w, "event: hello\ndata: {}\n\n")
		fl.Flush()
		ka := time.NewTicker(15 * time.Second)
		defer ka.Stop()
		for {
			select {
			case <-r.Context().Done():
				return
			case m := <-ch:
				fmt.Fprint(w, m)
				fl.Flush()
			case <-ka.C:
				fmt.Fprint(w, ": keep-alive\n\n")
				fl.Flush()
			}
		}
	})

	mux.HandleFunc("/api/info", func(w http.ResponseWriter, r *http.Request) {
		a.mu.Lock()
		l := a.launch
		a.mu.Unlock()
		writeJSON(w, map[string]any{
			"app": appName, "version": appVersion, "author": appAuthor,
			"dataDir": a.store.dataDir, "exportDir": a.store.exportDir, "suiteFile": a.store.suiteFile(),
			"launch": l, "os": osName(),
		})
	})

	mux.HandleFunc("/api/data", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-store")
			_, _ = w.Write(a.store.Load())
		case "PUT", "POST":
			b, err := io.ReadAll(io.LimitReader(r.Body, 64<<20))
			if err != nil {
				writeErr(w, 400, err)
				return
			}
			if !json.Valid(b) {
				writeErr(w, 400, errors.New("invalid JSON"))
				return
			}
			if err := a.store.Save(b); err != nil {
				writeErr(w, 500, err)
				return
			}
			writeJSON(w, map[string]bool{"ok": true})
		}
	})

	mux.HandleFunc("/api/suite/active", func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(io.LimitReader(r.Body, 4<<20))
		if err != nil || !json.Valid(b) {
			writeErr(w, 400, errors.New("invalid JSON"))
			return
		}
		if err := a.store.WriteSuite(b); err != nil {
			writeErr(w, 500, err)
			return
		}
		writeJSON(w, map[string]string{"path": a.store.suiteFile()})
	})

	mux.HandleFunc("/api/generate", func(w http.ResponseWriter, r *http.Request) {
		var req GenRequest
		if err := readJSON(r, &req); err != nil {
			writeErr(w, 400, err)
			return
		}
		res, err := Generate(req)
		if err != nil {
			writeErr(w, 422, err)
			return
		}
		writeJSON(w, res)
	})

	mux.HandleFunc("/api/save-file", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name     string `json:"name"`
			Content  string `json:"content"`
			Encoding string `json:"encoding"`
			Dir      string `json:"dir"`
		}
		if err := readJSON(r, &req); err != nil {
			writeErr(w, 400, err)
			return
		}
		data := []byte(req.Content)
		if req.Encoding == "base64" {
			d, err := base64.StdEncoding.DecodeString(req.Content)
			if err != nil {
				writeErr(w, 400, err)
				return
			}
			data = d
		}
		p, err := a.store.SaveExport(req.Dir, req.Name, data)
		if err != nil {
			writeErr(w, 500, err)
			return
		}
		writeJSON(w, map[string]string{"path": p})
	})

	mux.HandleFunc("/api/drives", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, listDrives())
	})

	mux.HandleFunc("/api/open-folder", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Path string `json:"path"`
		}
		_ = readJSON(r, &req)
		if req.Path == "" {
			req.Path = a.store.exportDir
		}
		if err := openFolder(req.Path); err != nil {
			writeErr(w, 500, err)
			return
		}
		writeJSON(w, map[string]bool{"ok": true})
	})

	mux.HandleFunc("/api/open-url", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			URL string `json:"url"`
		}
		_ = readJSON(r, &req)
		if !strings.HasPrefix(req.URL, "https://") && !strings.HasPrefix(req.URL, "http://") {
			writeErr(w, 400, errors.New("only web links can be opened"))
			return
		}
		_ = openURL(req.URL)
		writeJSON(w, map[string]bool{"ok": true})
	})

	a.conn.routes(mux)
	return a.guard(mux)
}
