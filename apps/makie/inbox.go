package makie

// Opening files given on the command line ("Open with → Makie", or dragging
// a file onto Makie.exe). The first Makie reads them into an inbox the
// editor collects; a second Makie hands its files to the one already running.

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type inboxItem struct {
	Name  string     `json:"name"`
	Kind  string     `json:"kind"` // wirie, project, mesh, error
	Wirie *wirieResp `json:"wirie,omitempty"`
	Text  string     `json:"text,omitempty"`
	Data  string     `json:"data,omitempty"` // base64 (meshes)
	Error string     `json:"error,omitempty"`
}

var (
	inboxMu sync.Mutex
	inbox   []inboxItem
)

func classify(path string, b []byte) inboxItem {
	name := filepath.Base(path)
	low := strings.ToLower(name)
	it := inboxItem{Name: name}
	switch {
	case strings.HasSuffix(low, ".stl") || strings.HasSuffix(low, ".obj"):
		it.Kind, it.Data = "mesh", base64.StdEncoding.EncodeToString(b)
	case strings.HasSuffix(low, ".makie.json") || (strings.HasSuffix(low, ".json") && bytes.Contains(b[:min(len(b), 4096)], []byte("wirie-case"))):
		res, err := wirieResult(b, path)
		if err != nil {
			it.Kind, it.Error = "error", err.Error()
		} else {
			it.Kind, it.Wirie = "wirie", res
		}
	case strings.HasSuffix(low, ".makie") || strings.HasSuffix(low, ".smithers") || strings.HasSuffix(low, ".partsmith") || strings.HasSuffix(low, ".json"):
		it.Kind, it.Text = "project", string(b)
	default:
		it.Kind, it.Error = "error", "Makie can't open this kind of file."
	}
	return it
}

func queueFiles(paths []string) {
	for _, p := range paths {
		p = strings.Trim(p, `"`)
		if p == "" {
			continue
		}
		var it inboxItem
		fi, err := os.Stat(p)
		switch {
		case err != nil:
			it = inboxItem{Name: filepath.Base(p), Kind: "error", Error: "File not found: " + p}
		case fi.IsDir() || fi.Size() > 256<<20:
			it = inboxItem{Name: filepath.Base(p), Kind: "error", Error: "Makie can't open this."}
		default:
			b, err := os.ReadFile(p)
			if err != nil {
				it = inboxItem{Name: filepath.Base(p), Kind: "error", Error: err.Error()}
			} else {
				it = classify(p, b)
			}
		}
		inboxMu.Lock()
		inbox = append(inbox, it)
		inboxMu.Unlock()
	}
}

// a secret shared through the settings folder, so only another Makie can
// hand files over
func inboxTokenPath() string {
	d, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(d, appName, "instance.key")
}

var inboxToken string

func makeInboxToken() {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	inboxToken = hex.EncodeToString(b)
	if p := inboxTokenPath(); p != "" {
		_ = os.MkdirAll(filepath.Dir(p), 0o755)
		_ = os.WriteFile(p, []byte(inboxToken), 0o600)
	}
}

// handOver sends files to a Makie that is already running.
func handOver(addr string, paths []string) bool {
	tok, err := os.ReadFile(inboxTokenPath())
	if err != nil {
		return false
	}
	abs := make([]string, 0, len(paths))
	for _, p := range paths {
		if a, err := filepath.Abs(strings.Trim(p, `"`)); err == nil {
			abs = append(abs, a)
		}
	}
	body, _ := json.Marshal(map[string]any{"token": string(tok), "paths": abs})
	c := http.Client{Timeout: 5 * time.Second}
	r, err := c.Post("http://"+addr+"/api/inbox/add", "application/json", bytes.NewReader(body))
	if err != nil {
		return false
	}
	r.Body.Close()
	return r.StatusCode == 200
}

func registerInboxAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/inbox", func(w http.ResponseWriter, r *http.Request) {
		inboxMu.Lock()
		items := inbox
		inbox = nil
		inboxMu.Unlock()
		if items == nil {
			items = []inboxItem{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(items)
	})
	mux.HandleFunc("/api/inbox/add", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Token string   `json:"token"`
			Paths []string `json:"paths"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil || req.Token == "" || req.Token != inboxToken {
			http.Error(w, "forbidden", 403)
			return
		}
		log.Printf("opening %v", req.Paths)
		queueFiles(req.Paths)
		w.WriteHeader(200)
	})
}
