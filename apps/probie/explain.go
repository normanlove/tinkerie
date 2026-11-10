// Probie — the Explain tab: captures from Probie itself, Scopie and Meterie
// explained in plain words. Other apps hand a capture over with
// `--app probie <file>`; a Probie that's already open takes it and shows it.
// Part of the Tinkerie maker suite. Created by nrmluvdev.
package probie

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// pending is a capture to show on the Explain tab (from the command line or
// handed over by another app).
type pending struct {
	mu   sync.Mutex
	path string
	seq  int64
}

func (p *pending) set(path string) {
	p.mu.Lock()
	p.path, p.seq = path, time.Now().UnixNano()
	p.mu.Unlock()
}

func (p *pending) get() (string, int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.path, p.seq
}

func tokenFile(dataDir string) string { return filepath.Join(dataDir, "session.json") }

// saveSession lets a second launch find the running Probie and its token.
func saveSession(dataDir string, port int, token string) {
	b, _ := json.Marshal(map[string]any{"port": port, "token": token})
	_ = os.WriteFile(tokenFile(dataDir), b, 0o600)
}

// handOver gives a capture to a Probie that's already open; true if it took it.
func handOver(dataDir, path string) bool {
	b, err := os.ReadFile(tokenFile(dataDir))
	if err != nil {
		return false
	}
	var s struct {
		Port  int    `json:"port"`
		Token string `json:"token"`
	}
	if json.Unmarshal(b, &s) != nil || s.Port == 0 || !pingExisting(s.Port) {
		return false
	}
	abs, _ := filepath.Abs(path)
	req, _ := http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:%d/api/explain/activate?path=%s", s.Port, urlEsc(abs)), nil)
	req.Header.Set("X-Probie-Token", s.Token)
	c := &http.Client{Timeout: 2 * time.Second}
	r, err := c.Do(req)
	if err != nil {
		return false
	}
	r.Body.Close()
	return r.StatusCode == 200
}

func urlEsc(s string) string {
	var b strings.Builder
	for _, c := range []byte(s) {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.IndexByte("-_.~/", c) >= 0 {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}

// migrateFromTracie moves settings from the app's old name the first time
// Probie runs.
func migrateFromTracie(dataDir string) {
	if _, err := os.Stat(dataDir); err == nil {
		return
	}
	old := filepath.Join(filepath.Dir(dataDir), "Tracie")
	if st, err := os.Stat(old); err == nil && st.IsDir() {
		_ = os.Rename(old, dataDir)
	}
}

func (a *App) explainRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/explain/info", func(w http.ResponseWriter, r *http.Request) {
		p, seq := a.pending.get()
		writeJSON(w, map[string]any{"demos": SigDemos, "path": p, "seq": seq})
	})
	mux.HandleFunc("/api/explain/activate", func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Query().Get("path")
		if p == "" {
			writeErr(w, 400, "no path")
			return
		}
		a.pending.set(p)
		a.hub.publish("explainFile", map[string]any{"path": p, "name": filepath.Base(p)})
		a.clientsMu.Lock()
		n := a.clients
		a.clientsMu.Unlock()
		if n == 0 {
			openWindow(fmt.Sprintf("http://127.0.0.1:%d/", a.port), a.dataDir)
		}
		writeJSON(w, map[string]bool{"ok": true})
	})
	mux.HandleFunc("/api/explain/open", func(w http.ResponseWriter, r *http.Request) {
		writeSignal(w, r.URL.Query().Get("path"))
	})
	mux.HandleFunc("/api/explain/analyse", func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(io.LimitReader(r.Body, MaxSignal))
		if err != nil {
			writeErr(w, 400, err.Error())
			return
		}
		c, err := LoadCapture(r.URL.Query().Get("name"), b)
		if err != nil {
			writeJSON(w, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, AnalyseCapture(c))
	})
	mux.HandleFunc("/api/explain/demo", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		body, ok := demoCaptures()[name]
		if !ok {
			writeJSON(w, map[string]string{"error": "no such example"})
			return
		}
		c, err := LoadCapture(name, []byte(body))
		if err != nil {
			writeJSON(w, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, AnalyseCapture(c))
	})
	mux.HandleFunc("/api/explain/recent", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, RecentCaptures())
	})
}

func writeSignal(w http.ResponseWriter, p string) {
	if p == "" {
		writeJSON(w, map[string]string{"error": "no capture to open"})
		return
	}
	st, err := os.Stat(p)
	if err != nil {
		writeJSON(w, map[string]string{"error": "Probie couldn't open " + filepath.Base(p) + ": " + err.Error()})
		return
	}
	if st.Size() > MaxSignal {
		writeJSON(w, map[string]string{"error": fmt.Sprintf("%s is %d MB: Probie explains captures up to %d MB. Export a shorter piece.", filepath.Base(p), st.Size()>>20, MaxSignal>>20)})
		return
	}
	b, err := os.ReadFile(p)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	c, err := LoadCapture(p, b)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, AnalyseCapture(c))
}

// RecentCapture is a capture saved by Probie, Scopie or Meterie.
type RecentCapture struct {
	Path string `json:"path"`
	Name string `json:"name"`
	App  string `json:"app"`
	Time int64  `json:"time"`
	Size int64  `json:"size"`
}

// RecentCaptures lists the newest captures in the suite's folders
// (including Documents/Tracie, from before Probie had its new name).
func RecentCaptures() []RecentCapture {
	home, _ := os.UserHomeDir()
	dirs := [][2]string{{"Probie", "Probie"}, {"Probie", "Tracie"}, {"Scopie", "Scopie"}, {"Meterie", "Meterie"}}
	var out []RecentCapture
	for _, d0 := range dirs {
		for _, root := range []string{filepath.Join(home, "Documents"), filepath.Join(home, "OneDrive", "Documents")} {
			d := filepath.Join(root, d0[1])
			ents, _ := os.ReadDir(d)
			for _, e := range ents {
				low := strings.ToLower(e.Name())
				if e.IsDir() || !(strings.HasSuffix(low, ".csv") || strings.HasSuffix(low, ".vcd")) {
					continue
				}
				info, err := e.Info()
				if err != nil {
					continue
				}
				out = append(out, RecentCapture{Path: filepath.Join(d, e.Name()), Name: e.Name(), App: d0[0], Time: info.ModTime().Unix(), Size: info.Size()})
			}
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Time > out[b].Time })
	if len(out) > 40 {
		out = out[:40]
	}
	return out
}
