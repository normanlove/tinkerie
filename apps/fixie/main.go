// Fixie — repair guide.
// Part of the free nrmluvdev maker suite Tinkerie. Created by nrmluvdev.
//
// Fixie takes you step by step through finding what's wrong with a broken
// electronic device: pick the device and what it's doing, and Fixie asks you
// to look, measure and test one thing at a time, telling you which tools to
// use and what a good and a bad result looks like, until it reaches the
// likely fault and how to fix it. It also has a tools guide, a picture guide
// to the signs of a faulty part, safety advice, and a log of your repairs.
//
// Like the other Tinkerie apps, Fixie is a small local web server (127.0.0.1
// only) with its interface embedded, shown in its own app window. The guide
// is in web/guide.json; repairs are saved in this computer's settings folder.
package fixie

import (
	"embed"
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
	appName     = "Fixie"
	appVersion  = "0.2.1-alpha"
	appAuthor   = "nrmluvdev"
	defaultPort = 47825 // Tinkerie ports: see PORTING.md
	maxRepairs  = 4 << 20
)

type server struct {
	mu       sync.Mutex
	clients  int
	ever     bool
	lastGone time.Time
	started  time.Time
	host     string
	url      string
	dataDir  string
	stay     bool
}

// Main runs Fixie.
func Main() {
	port := flag.Int("port", defaultPort, "local port for the interface")
	noBrowser := flag.Bool("no-browser", false, "don't open a window (use your own browser)")
	flag.Parse()

	dir := dataDir()
	setupLog(dir)
	if handOver(*port) {
		return
	}
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *port))
	if err != nil {
		if ln, err = net.Listen("tcp", "127.0.0.1:0"); err != nil {
			fatal("Fixie could not start its local server:\n" + err.Error())
		}
	}
	s := &server{started: time.Now(), host: ln.Addr().String(), dataDir: dir,
		stay: *noBrowser || os.Getenv("FIXIE_NO_WINDOW") != ""}
	s.url = "http://" + s.host + "/"
	log.Printf("Fixie %s listening on %s", appVersion, s.url)
	srv := &http.Server{Handler: s.guard(s.routes()), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("server: %v", err)
		}
	}()
	if !s.stay {
		if err := openAppWindow(s.url, filepath.Join(dir, "window")); err != nil {
			log.Printf("window: %v", err)
		}
	}
	s.waitForExit()
	log.Printf("Fixie closed")
}

func dataDir() string {
	base, err := os.UserConfigDir()
	if err != nil {
		base = os.TempDir()
	}
	d := filepath.Join(base, "nrmluvdev", "Fixie")
	_ = os.MkdirAll(d, 0o755)
	return d
}

func setupLog(dir string) {
	p := filepath.Join(dir, "fixie.log")
	if st, err := os.Stat(p); err == nil && st.Size() > 1<<20 {
		_ = os.Remove(p)
	}
	if f, err := os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644); err == nil {
		log.SetOutput(io.MultiWriter(f, os.Stderr))
	}
}

// handOver asks a Fixie that's already open to show its window; true if one did.
func handOver(port int) bool {
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
	req, _ := http.NewRequest("POST", base+"/api/activate", strings.NewReader("{}"))
	req.Header.Set("X-Fixie", "1")
	if r, err := c.Do(req); err == nil {
		r.Body.Close()
	}
	return true
}

// waitForExit returns once the window has been gone for a few seconds.
func (s *server) waitForExit() {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for range t.C {
		s.mu.Lock()
		n, ever, gone := s.clients, s.ever, s.lastGone
		s.mu.Unlock()
		if s.stay {
			continue
		}
		if ever && n == 0 && time.Since(gone) > 6*time.Second {
			return
		}
		if !ever && time.Since(s.started) > 3*time.Minute {
			return
		}
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(v)
}

// guard keeps other websites out: the Host must be ours, and API calls must
// carry a custom header that browsers won't send cross-site.
func (s *server) guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		port := s.host[strings.LastIndex(s.host, ":"):]
		if h := r.Host; h != s.host && h != "localhost"+port && h != "127.0.0.1"+port {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") && r.URL.Path != "/api/ping" && r.URL.Path != "/api/events" && r.Header.Get("X-Fixie") != "1" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// repairsPath is where the repair log and tool list are kept (one JSON file).
func (s *server) repairsPath() string { return filepath.Join(s.dataDir, "state.json") }

func (s *server) routes() http.Handler {
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
	mux.HandleFunc("/api/info", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"app": appName, "version": appVersion, "author": appAuthor})
	})
	// the repair log and the tools you have: the window sends it all whenever it changes
	mux.HandleFunc("/api/state", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		if r.Method == http.MethodPost {
			b, err := io.ReadAll(io.LimitReader(r.Body, maxRepairs+1))
			if err != nil || len(b) > maxRepairs || !json.Valid(b) {
				http.Error(w, "bad state", 400)
				return
			}
			tmp := s.repairsPath() + ".tmp"
			if err := os.WriteFile(tmp, b, 0o644); err == nil {
				err = os.Rename(tmp, s.repairsPath())
				if err != nil {
					log.Printf("repairs: %v", err)
				}
			}
			writeJSON(w, map[string]bool{"ok": true})
			return
		}
		b, err := os.ReadFile(s.repairsPath())
		if err != nil || !json.Valid(b) {
			b = []byte("[]")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(b)
	})
	mux.HandleFunc("/api/activate", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		n := s.clients
		s.mu.Unlock()
		if n == 0 && !s.stay {
			_ = openAppWindow(s.url, filepath.Join(s.dataDir, "window"))
		}
		writeJSON(w, map[string]bool{"ok": true})
	})
	// The window keeps this stream open; when it closes, Fixie quits.
	mux.HandleFunc("/api/events", func(w http.ResponseWriter, r *http.Request) {
		fl, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", 500)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-store")
		s.mu.Lock()
		s.clients++
		s.ever = true
		s.mu.Unlock()
		defer func() {
			s.mu.Lock()
			s.clients--
			s.lastGone = time.Now()
			s.mu.Unlock()
		}()
		fmt.Fprint(w, "event: hello\ndata: {}\n\n")
		fl.Flush()
		t := time.NewTicker(15 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-r.Context().Done():
				return
			case <-t.C:
				fmt.Fprint(w, ": keep-alive\n\n")
				fl.Flush()
			}
		}
	})
	return mux
}
