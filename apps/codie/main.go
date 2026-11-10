// Codie — code explainer.
// Part of the free nrmluvdev maker suite Tinkerie. Created by nrmluvdev.
//
// Paste code (or open a file) and Codie explains it line by line: what each
// part does and why it's there, with an outline, the ideas it uses, warnings
// about common mistakes, and a game-development view that explains game loops
// and game logic across Unity, Unreal, Godot, LÖVE, PICO-8, Roblox, pygame,
// browser games and the BASIC of older machines.
//
// Like the other Tinkerie apps, Codie is a small local web server (127.0.0.1
// only) with its interface embedded, shown in its own app window. All the
// analysis happens on this computer; no code is sent anywhere.
package codie

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
	appName     = "Codie"
	appVersion  = "0.3.0-alpha"
	appAuthor   = "nrmluvdev"
	defaultPort = 47824
	maxFile     = 2 << 20
)

type openFile struct {
	Name string `json:"name"`
	Code string `json:"code"`
	Seq  int64  `json:"seq"`
}

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
	file     openFile
}

// Main runs Codie.
func Main() {
	port := flag.Int("port", defaultPort, "local port for the interface")
	noBrowser := flag.Bool("no-browser", false, "don't open a window (use your own browser)")
	flag.Parse()

	dir := dataDir()
	setupLog(dir)
	var file openFile
	if flag.NArg() > 0 {
		file = readFile(flag.Arg(0))
	}
	if handOver(*port, file) {
		return
	}
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *port))
	if err != nil {
		if ln, err = net.Listen("tcp", "127.0.0.1:0"); err != nil {
			fatal("Codie could not start its local server:\n" + err.Error())
		}
	}
	s := &server{started: time.Now(), host: ln.Addr().String(), dataDir: dir, file: file,
		stay: *noBrowser || os.Getenv("CODIE_NO_WINDOW") != ""}
	s.url = "http://" + s.host + "/"
	log.Printf("Codie %s listening on %s", appVersion, s.url)
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
	log.Printf("Codie closed")
}

func dataDir() string {
	base, err := os.UserConfigDir()
	if err != nil {
		base = os.TempDir()
	}
	d := filepath.Join(base, "nrmluvdev", "Codie")
	_ = os.MkdirAll(d, 0o755)
	return d
}

func setupLog(dir string) {
	p := filepath.Join(dir, "codie.log")
	if st, err := os.Stat(p); err == nil && st.Size() > 1<<20 {
		_ = os.Remove(p)
	}
	if f, err := os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644); err == nil {
		log.SetOutput(io.MultiWriter(f, os.Stderr))
	}
}

func readFile(p string) openFile {
	f, err := os.Open(p)
	if err != nil {
		return openFile{}
	}
	defer f.Close()
	b, _ := io.ReadAll(io.LimitReader(f, maxFile))
	return openFile{Name: filepath.Base(p), Code: string(b), Seq: time.Now().UnixNano()}
}

// handOver gives a file to a Codie that's already open; true if it took it.
func handOver(port int, f openFile) bool {
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
	b, _ := json.Marshal(f)
	req, _ := http.NewRequest("POST", base+"/api/activate", strings.NewReader(string(b)))
	req.Header.Set("X-Codie", "1")
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
		if strings.HasPrefix(r.URL.Path, "/api/") && r.URL.Path != "/api/ping" && r.URL.Path != "/api/events" && r.Header.Get("X-Codie") != "1" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

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
		s.mu.Lock()
		f := s.file
		s.mu.Unlock()
		writeJSON(w, map[string]any{"app": appName, "version": appVersion, "author": appAuthor,
			"langs": LangList(), "samples": Samples(), "file": f})
	})
	mux.HandleFunc("/api/lessons", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, lessons)
	})
	mux.HandleFunc("/api/analyse", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Code string `json:"code"`
			Lang string `json:"lang"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, MaxCode+4096)).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		writeJSON(w, Analyse(req.Code, req.Lang))
	})
	mux.HandleFunc("/api/activate", func(w http.ResponseWriter, r *http.Request) {
		var f openFile
		_ = json.NewDecoder(io.LimitReader(r.Body, maxFile*2)).Decode(&f)
		s.mu.Lock()
		if f.Code != "" {
			f.Seq = time.Now().UnixNano()
			s.file = f
		}
		n := s.clients
		s.mu.Unlock()
		if n == 0 && !s.stay {
			_ = openAppWindow(s.url, filepath.Join(s.dataDir, "window"))
		}
		writeJSON(w, map[string]bool{"ok": true})
	})
	// The window keeps this stream open; when it closes, Codie quits. It also
	// tells the window about files opened while it was running.
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
		seq := s.file.Seq
		s.mu.Unlock()
		defer func() {
			s.mu.Lock()
			s.clients--
			s.lastGone = time.Now()
			s.mu.Unlock()
		}()
		fmt.Fprint(w, "event: hello\ndata: {}\n\n")
		fl.Flush()
		t := time.NewTicker(time.Second)
		defer t.Stop()
		beat := 0
		for {
			select {
			case <-r.Context().Done():
				return
			case <-t.C:
				s.mu.Lock()
				f := s.file
				s.mu.Unlock()
				if f.Seq != seq {
					seq = f.Seq
					b, _ := json.Marshal(f)
					fmt.Fprintf(w, "event: file\ndata: %s\n\n", b)
					fl.Flush()
				}
				if beat++; beat%15 == 0 {
					fmt.Fprint(w, ": keep-alive\n\n")
					fl.Flush()
				}
			}
		}
	})
	return mux
}
