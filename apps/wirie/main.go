// Wirie — a beginner-friendly circuit board designer.
// Created by nrmluvdev. Part of the nrmluvdev maker tool suite (with Smithers).
//
// Architecture: a single self-contained executable. The whole editor UI
// (HTML/CSS/JS in ./web) is embedded into the binary and served on a local
// loopback port; the app opens it in a chromeless "app mode" window
// (Edge/Chrome), falling back to the default browser. Only the Go standard
// library is used, so porting is just a matter of GOOS/GOARCH plus the small
// launcher_*.go file for that platform.
package wirie

import (
	"archive/zip"
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	appName    = "Wirie"
	appVersion = "0.13.0-alpha"
	appAuthor  = "nrmluvdev"
	basePort   = 47820 // Tinkerie ports: see PORTING.md
)

//go:embed web/*
var webFS embed.FS

var (
	lastPing   time.Time
	pingMu     sync.Mutex
	everPinged bool
	dataDir    string
)

func touch() {
	pingMu.Lock()
	lastPing = time.Now()
	everPinged = true
	pingMu.Unlock()
}

// documentsDir finds the user's Documents folder (handles OneDrive redirection).
func documentsDir() string {
	home, _ := os.UserHomeDir()
	cands := []string{}
	if od := os.Getenv("OneDrive"); od != "" {
		cands = append(cands, filepath.Join(od, "Documents"))
	}
	cands = append(cands, filepath.Join(home, "Documents"))
	for _, c := range cands {
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			return c
		}
	}
	return home
}

var safeName = regexp.MustCompile(`[^A-Za-z0-9 _\-\.\(\)]+`)

func cleanName(n string) string {
	n = strings.TrimSpace(safeName.ReplaceAllString(n, "_"))
	n = strings.Trim(n, ". ")
	if n == "" {
		n = "Untitled"
	}
	if len(n) > 80 {
		n = n[:80]
	}
	return n
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, code int, msg string) {
	w.WriteHeader(code)
	writeJSON(w, map[string]string{"error": msg})
}

// only accept requests from our own page (blocks other websites poking the API)
func localOnly(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if o := r.Header.Get("Origin"); o != "" && !strings.HasPrefix(o, "http://127.0.0.1:") && !strings.HasPrefix(o, "http://localhost:") {
			fail(w, 403, "forbidden")
			return
		}
		if r.Method == http.MethodPost && !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			fail(w, 415, "json only")
			return
		}
		next(w, r)
	}
}

func projectsDir() string { return filepath.Join(dataDir, "Projects") }
func exportsDir() string  { return filepath.Join(dataDir, "Exports") }

func handleList(w http.ResponseWriter, r *http.Request) {
	type item struct {
		Name     string `json:"name"`
		Modified string `json:"modified"`
		Size     int64  `json:"size"`
	}
	items := []item{}
	ents, _ := os.ReadDir(projectsDir())
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".wirie") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		items = append(items, item{strings.TrimSuffix(e.Name(), filepath.Ext(e.Name())), info.ModTime().Format("2006-01-02 15:04"), info.Size()})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Modified > items[j].Modified })
	writeJSON(w, map[string]any{"items": items, "folder": projectsDir()})
}

func handleLoad(w http.ResponseWriter, r *http.Request) {
	name := cleanName(r.URL.Query().Get("name"))
	b, err := os.ReadFile(filepath.Join(projectsDir(), name+".wirie"))
	if err != nil {
		fail(w, 404, "not found")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

type saveReq struct {
	Name string          `json:"name"`
	Data json.RawMessage `json:"data"`
}

func handleSave(w http.ResponseWriter, r *http.Request) {
	var req saveReq
	if err := json.NewDecoder(io.LimitReader(r.Body, 64<<20)).Decode(&req); err != nil {
		fail(w, 400, "bad request")
		return
	}
	name := cleanName(req.Name)
	os.MkdirAll(projectsDir(), 0o755)
	p := filepath.Join(projectsDir(), name+".wirie")
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, req.Data, 0o644); err != nil {
		fail(w, 500, err.Error())
		return
	}
	os.Rename(tmp, p)
	writeJSON(w, map[string]string{"path": p, "name": name})
}

func handleAutosave(w http.ResponseWriter, r *http.Request) {
	p := filepath.Join(dataDir, "autosave.wirie")
	if r.Method == http.MethodGet {
		b, err := os.ReadFile(p)
		if err != nil {
			fail(w, 404, "none")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
		return
	}
	b, err := io.ReadAll(io.LimitReader(r.Body, 64<<20))
	if err != nil {
		fail(w, 400, "bad")
		return
	}
	os.WriteFile(p, b, 0o644)
	writeJSON(w, map[string]bool{"ok": true})
}

type exportReq struct {
	Name  string            `json:"name"`
	Files map[string]string `json:"files"`
}

func handleExport(w http.ResponseWriter, r *http.Request) {
	var req exportReq
	if err := json.NewDecoder(io.LimitReader(r.Body, 128<<20)).Decode(&req); err != nil {
		fail(w, 400, "bad request")
		return
	}
	name := cleanName(req.Name)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	keys := make([]string, 0, len(req.Files))
	for k := range req.Files {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		f, err := zw.CreateHeader(&zip.FileHeader{Name: cleanName(k), Method: zip.Deflate, Modified: time.Now()})
		if err != nil {
			continue
		}
		f.Write([]byte(req.Files[k]))
	}
	zw.SetComment("Made with " + appName + " by " + appAuthor)
	zw.Close()
	os.MkdirAll(exportsDir(), 0o755)
	p := filepath.Join(exportsDir(), name+"-gerbers.zip")
	if err := os.WriteFile(p, buf.Bytes(), 0o644); err != nil {
		fail(w, 500, err.Error())
		return
	}
	writeJSON(w, map[string]string{"path": p})
}

type fileReq struct {
	Name    string `json:"name"`
	Ext     string `json:"ext"`
	Content string `json:"content"`
}

// handleSaveFile writes a single export file (e.g. the Makie case file) into Exports.
func handleSaveFile(w http.ResponseWriter, r *http.Request) {
	var req fileReq
	if err := json.NewDecoder(io.LimitReader(r.Body, 64<<20)).Decode(&req); err != nil {
		fail(w, 400, "bad request")
		return
	}
	allowed := map[string]bool{".makie.json": true, ".keebie.json": true, ".svg": true, ".csv": true}
	if !allowed[req.Ext] {
		fail(w, 400, "file type not allowed")
		return
	}
	os.MkdirAll(exportsDir(), 0o755)
	p := filepath.Join(exportsDir(), cleanName(req.Name)+req.Ext)
	if err := os.WriteFile(p, []byte(req.Content), 0o644); err != nil {
		fail(w, 500, err.Error())
		return
	}
	writeJSON(w, map[string]string{"path": p})
}

func partsDir() string { return filepath.Join(dataDir, "Parts") }

var safeID = regexp.MustCompile(`^[A-Za-z0-9_\-]{1,80}$`)

// user-made parts: Documents\Wirie\Parts\<id>.wpart.json
func handleParts(w http.ResponseWriter, r *http.Request) {
	os.MkdirAll(partsDir(), 0o755)
	if r.Method == http.MethodGet {
		items := []json.RawMessage{}
		ents, _ := os.ReadDir(partsDir())
		for _, e := range ents {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".wpart.json") {
				continue
			}
			if b, err := os.ReadFile(filepath.Join(partsDir(), e.Name())); err == nil && json.Valid(b) {
				items = append(items, b)
			}
		}
		writeJSON(w, map[string]any{"items": items, "folder": partsDir()})
		return
	}
	var req struct {
		ID     string          `json:"id"`
		Def    json.RawMessage `json:"def"`
		Delete bool            `json:"delete"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 8<<20)).Decode(&req); err != nil || !safeID.MatchString(req.ID) {
		fail(w, 400, "bad request")
		return
	}
	p := filepath.Join(partsDir(), req.ID+".wpart.json")
	if req.Delete {
		os.Remove(p)
		writeJSON(w, map[string]bool{"ok": true})
		return
	}
	if !json.Valid(req.Def) {
		fail(w, 400, "bad part")
		return
	}
	if err := os.WriteFile(p, req.Def, 0o644); err != nil {
		fail(w, 500, err.Error())
		return
	}
	writeJSON(w, map[string]string{"path": p})
}

func handleReveal(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&req)
	p := filepath.Clean(req.Path)
	// only reveal inside our own data folder
	if rel, err := filepath.Rel(dataDir, p); err != nil || strings.HasPrefix(rel, "..") {
		p = dataDir
	}
	revealInFolder(p)
	writeJSON(w, map[string]bool{"ok": true})
}

// handleOpenIn hands an export to another Tinkerie app: a case file to Makie,
// or a Keebie keyboard to Flashie's Keyboard tab. Only files in Exports.
func handleOpenIn(w http.ResponseWriter, r *http.Request) {
	var req struct {
		App  string `json:"app"`
		Path string `json:"path"`
	}
	json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&req)
	p := filepath.Clean(req.Path)
	rel, err := filepath.Rel(exportsDir(), p)
	if err != nil || strings.HasPrefix(rel, "..") {
		fail(w, 400, "only exported files can be opened")
		return
	}
	var args []string
	switch {
	case req.App == "makie" && strings.HasSuffix(p, ".makie.json"):
		args = []string{"--app", "makie", p}
	case req.App == "flashie" && strings.HasSuffix(p, ".keebie.json"):
		args = []string{"--app", "flashie", "--keyboard", p}
	default:
		fail(w, 400, "can't open that file there")
		return
	}
	exe, err := os.Executable()
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	cmd := exec.Command(exe, args...)
	cmd.Dir = filepath.Dir(exe)
	if err := cmd.Start(); err != nil {
		fail(w, 500, "couldn't start "+req.App+": "+err.Error())
		return
	}
	go cmd.Wait()
	writeJSON(w, map[string]string{"path": p})
}

func Main() {
	dataDir = filepath.Join(documentsDir(), appName)
	os.MkdirAll(projectsDir(), 0o755)
	os.MkdirAll(exportsDir(), 0o755)
	os.MkdirAll(partsDir(), 0o755)
	if lf, err := os.OpenFile(filepath.Join(dataDir, "wirie.log"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644); err == nil {
		log.SetOutput(lf)
	}
	log.Printf("%s %s by %s starting", appName, appVersion, appAuthor)

	// single instance: if Wirie is already running, just open another window on it
	cl := http.Client{Timeout: 400 * time.Millisecond}
	if resp, err := cl.Get(fmt.Sprintf("http://127.0.0.1:%d/api/ping", basePort)); err == nil {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if strings.Contains(string(b), `"wirie"`) {
			openWindow(fmt.Sprintf("http://127.0.0.1:%d/", basePort))
			return
		}
	}

	var ln net.Listener
	var err error
	// Only Wirie's own port, then any free one: the ports next to it belong to
	// other Tinkerie apps (see PORTING.md).
	ln, err = net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", basePort))
	if err != nil {
		ln, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			log.Fatal(err)
		}
	}
	url := "http://" + ln.Addr().String() + "/"

	sub, _ := fs.Sub(webFS, "web")
	mux := http.NewServeMux()
	static := http.FileServer(http.FS(sub))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		static.ServeHTTP(w, r)
	})
	mux.HandleFunc("/api/ping", func(w http.ResponseWriter, r *http.Request) {
		touch()
		writeJSON(w, map[string]string{"app": "wirie", "version": appVersion, "author": appAuthor, "folder": dataDir})
	})
	mux.HandleFunc("/api/bye", func(w http.ResponseWriter, r *http.Request) {
		// page closed or reloading: give it a few seconds to come back
		pingMu.Lock()
		lastPing = time.Now().Add(-55 * time.Second)
		pingMu.Unlock()
		w.WriteHeader(204)
	})
	mux.HandleFunc("/api/list", localOnly(handleList))
	mux.HandleFunc("/api/load", localOnly(handleLoad))
	mux.HandleFunc("/api/save", localOnly(handleSave))
	mux.HandleFunc("/api/autosave", localOnly(handleAutosave))
	mux.HandleFunc("/api/export", localOnly(handleExport))
	mux.HandleFunc("/api/reveal", localOnly(handleReveal))
	mux.HandleFunc("/api/savefile", localOnly(handleSaveFile))
	mux.HandleFunc("/api/openin", localOnly(handleOpenIn))
	mux.HandleFunc("/api/parts", localOnly(handleParts))

	// watchdog: quit once the editor window has been gone for a while
	go func() {
		start := time.Now()
		for range time.Tick(2 * time.Second) {
			pingMu.Lock()
			idle := time.Since(lastPing)
			ever := everPinged
			pingMu.Unlock()
			if os.Getenv("WIRIE_STAY") != "" {
				continue
			}
			if (ever && idle > 60*time.Second) || (!ever && time.Since(start) > 3*time.Minute) {
				log.Printf("window closed, exiting")
				os.Exit(0)
			}
		}
	}()

	if os.Getenv("WIRIE_NO_WINDOW") == "" {
		go func() { time.Sleep(150 * time.Millisecond); openWindow(url) }()
	}
	log.Printf("serving on %s", url)
	fmt.Println("Wirie running at", url)
	log.Fatal(http.Serve(ln, mux))
}
