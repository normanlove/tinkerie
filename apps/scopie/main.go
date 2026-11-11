// Scopie — a beginner-friendly program for many USB, bench and handheld
// oscilloscopes. Part of the free nrmluvdev maker suite.
// Created by nrmluvdev.
//
// Scopie is a single executable: a small local web server (bound to
// 127.0.0.1 only) with the whole interface embedded, shown in its own app
// window (Microsoft Edge / Chrome "app mode"). It closes itself when the
// window is closed.
//
// Copyright (C) 2026 nrmluvdev. Free software under the GNU General Public
// License, version 3 or later: see LICENSE and THIRD-PARTY-NOTICES.md.
package scopie

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
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"tinkerie/shared/suite"
)

//go:embed web
var webFS embed.FS

//go:embed LICENSE THIRD-PARTY-NOTICES.md
var docFS embed.FS

const (
	appName     = "Scopie"
	appVersion  = "0.11.2-alpha"
	appAuthor   = "nrmluvdev"
	defaultPort = 47831
)

type App struct {
	mu        sync.Mutex
	clients   int
	ever      bool
	lastGone  time.Time
	started   time.Time
	url       string
	host      string
	noBrowser bool
	hub       *Hub
	eng       *Engine
	view      *View
	scanMu    sync.Mutex
}

func Main() {
	port := flag.Int("port", defaultPort, "local port for the interface")
	noBrowser := flag.Bool("no-browser", false, "don't open a window (use your own browser)")
	demo := flag.Bool("demo", false, "start in demo mode")
	flag.Parse()

	setAppID()
	dataDir := appDataDir()
	setupLog(dataDir)

	// Already running? Bring the existing window forward and exit.
	if handOver(*port) {
		return
	}
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *port))
	if err != nil {
		if ln, err = net.Listen("tcp", "127.0.0.1:0"); err != nil {
			fatal("Scopie could not start its local server:\n" + err.Error())
		}
	}
	hub := newHub()
	eng := newEngine()
	eng.Start()
	a := &App{started: time.Now(), noBrowser: *noBrowser, hub: hub, eng: eng, host: ln.Addr().String()}
	a.view = newView(eng, hub, newSettingsStore())
	a.url = "http://" + a.host + "/"
	if *demo {
		a.view.startupDone = true
		for _, c := range scanSim(nil, nil) {
			a.view.Connect(c)
			break
		}
	}
	log.Printf("Scopie %s listening on %s", appVersion, a.url)

	srv := &http.Server{Handler: a.routes(), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("server: %v", err)
		}
	}()
	go a.statusLoop()
	go a.analysisLoop()
	if !*noBrowser {
		if err := openAppWindow(a.url, dataDir); err != nil {
			log.Printf("window: %v", err)
		}
	} else {
		fmt.Println("Scopie is running at", a.url)
	}
	a.waitForExit()
	a.view.store.flush()
	eng.Quit()
	log.Printf("Scopie closed")
}

func setupLog(dir string) {
	p := filepath.Join(dir, "scopie.log")
	if st, err := os.Stat(p); err == nil && st.Size() > 1<<20 {
		_ = os.Remove(p)
	}
	if f, err := os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644); err == nil {
		log.SetOutput(io.MultiWriter(f, os.Stderr))
	}
}

// handOver asks an already-running Scopie to show itself; true if it did.
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
	req.Header.Set("X-Scopie", "1")
	if r, err := c.Do(req); err == nil {
		r.Body.Close()
	}
	return true
}

// waitForExit returns once the window has been closed for a few seconds.
func (a *App) waitForExit() {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for range t.C {
		if a.noBrowser {
			continue
		}
		a.mu.Lock()
		n, ever, gone := a.clients, a.ever, a.lastGone
		a.mu.Unlock()
		if ever && n == 0 && time.Since(gone) > 6*time.Second {
			return
		}
		if !ever && time.Since(a.started) > 10*time.Minute {
			return
		}
	}
}

// analysisLoop keeps measurements and the spectrum up to date in the background.
func (a *App) analysisLoop() {
	t := time.NewTicker(50 * time.Millisecond)
	defer t.Stop()
	for i := 0; ; i++ {
		<-t.C
		a.mu.Lock()
		n := a.clients
		a.mu.Unlock()
		if n == 0 {
			continue
		}
		a.view.updateFFT()
		if i%3 == 0 {
			a.view.updateMeasurements()
		}
	}
}

func (a *App) statusLoop() {
	t := time.NewTicker(200 * time.Millisecond)
	defer t.Stop()
	for range t.C {
		a.mu.Lock()
		n := a.clients
		a.mu.Unlock()
		if n > 0 {
			a.hub.publish("status", a.view.status())
		}
	}
}

// ---------------------------------------------------------------- HTTP

// guard blocks other websites from talking to Scopie: requests must target
// our own host name, and API calls must carry a custom header (which
// browsers won't send cross-origin without a CORS approval we never give).
func (a *App) guard(next http.Handler) http.Handler {
	port := a.host[strings.LastIndex(a.host, ":"):]
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h := r.Host; h != a.host && h != "localhost"+port && h != "127.0.0.1"+port {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") && r.URL.Path != "/api/ping" && r.URL.Path != "/api/events" {
			if r.Header.Get("X-Scopie") != "1" {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

type candJSON struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Family string `json:"family"`
	Detail string `json:"detail"`
	Ready  bool   `json:"ready"`
}

func candList(cs []*Candidate) []candJSON {
	out := []candJSON{}
	for _, c := range cs {
		out = append(out, candJSON{c.ID, c.Label, c.Family, c.Detail, c.Ready && c.Factory != nil})
	}
	return out
}

func splitLAN(s string) []string {
	var out []string
	for _, p := range strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ';' || r == ' ' }) {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (a *App) scan(lan string) ([]*Candidate, []string) {
	a.scanMu.Lock()
	defer a.scanMu.Unlock()
	cs, notes := scanAll(splitLAN(lan))
	if notes == nil {
		notes = []string{}
	}
	return cs, notes
}

var safeName = regexp.MustCompile(`[^A-Za-z0-9 ._()\-]+`)

// saveExport writes a file into Documents\Scopie and returns its path.
func saveExport(name string, data []byte) (string, error) {
	dir := exportDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name = strings.TrimSpace(safeName.ReplaceAllString(filepath.Base(name), "_"))
	if name == "" || name == "." {
		name = "scopie-" + time.Now().Format("20060102-150405")
	}
	p := filepath.Join(dir, name)
	ext := filepath.Ext(name)
	for i := 2; ; i++ {
		if _, err := os.Stat(p); err != nil {
			break
		}
		p = filepath.Join(dir, fmt.Sprintf("%s (%d)%s", strings.TrimSuffix(name, ext), i, ext))
	}
	return p, os.WriteFile(p, data, 0o644)
}

func (a *App) routes() http.Handler {
	mux := http.NewServeMux()
	sub, _ := fs.Sub(webFS, "web")
	files := http.FileServer(http.FS(sub))
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		files.ServeHTTP(w, r)
	}))
	ok := func(w http.ResponseWriter) { writeJSON(w, map[string]bool{"ok": true}) }
	fail := func(w http.ResponseWriter, err error) { writeErr(w, 400, err.Error()) }

	mux.HandleFunc("/api/ping", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"app": appName, "version": appVersion})
	})
	mux.HandleFunc("/api/activate", func(w http.ResponseWriter, r *http.Request) {
		a.mu.Lock()
		n := a.clients
		a.mu.Unlock()
		a.hub.publish("activate", map[string]any{})
		if n == 0 {
			_ = openAppWindow(a.url, appDataDir())
		}
		ok(w)
	})
	mux.HandleFunc("/api/events", func(w http.ResponseWriter, r *http.Request) {
		fl, isFl := w.(http.Flusher)
		if !isFl {
			http.Error(w, "streaming unsupported", 500)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-store")
		ch := a.hub.subscribe()
		a.mu.Lock()
		a.clients++
		a.ever = true
		a.mu.Unlock()
		defer func() {
			a.hub.unsubscribe(ch)
			a.mu.Lock()
			a.clients--
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
			case m, open := <-ch:
				if !open {
					return
				}
				fmt.Fprintf(w, "event: %s\ndata: %s\n\n", m.event, m.data)
				fl.Flush()
			case <-ka.C:
				fmt.Fprint(w, ": keep-alive\n\n")
				fl.Flush()
			}
		}
	})

	mux.HandleFunc("/api/info", func(w http.ResponseWriter, r *http.Request) {
		var fams []string
		for _, m := range driverModules {
			fams = append(fams, m.family)
		}
		writeJSON(w, map[string]any{
			"app": appName, "version": appVersion, "author": appAuthor, "os": osName(),
			"dataDir": appDataDir(), "exportDir": exportDir(),
			"measurements": measurements, "demoSignals": demoSignals, "fftWindows": fftWindows,
			"mathScales": mathScales, "avgOpts": avgOpts, "persistOpts": persistOpts,
			"families": fams, "device": a.view.InfoPayload(), "settings": a.view.Settings(),
		})
	})

	mux.HandleFunc("/api/settings", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			writeJSON(w, a.view.Settings())
			return
		}
		var s Settings
		if err := readJSON(r, &s); err != nil {
			fail(w, err)
			return
		}
		writeJSON(w, a.view.SetSettings(s))
	})

	mux.HandleFunc("/api/frame", func(w http.ResponseWriter, r *http.Request) {
		have, _ := strconv.ParseInt(r.URL.Query().Get("have"), 10, 64)
		b := a.view.Frame(have)
		w.Header().Set("Cache-Control", "no-store")
		if b == nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(b)
	})

	mux.HandleFunc("/api/run", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Run *bool `json:"run"`
		}
		_ = readJSON(r, &req)
		writeJSON(w, map[string]bool{"running": a.view.ToggleRun(req.Run)})
	})
	mux.HandleFunc("/api/single", func(w http.ResponseWriter, r *http.Request) { a.view.Single(); ok(w) })
	mux.HandleFunc("/api/clear", func(w http.ResponseWriter, r *http.Request) { a.view.Clear(); ok(w) })
	mux.HandleFunc("/api/default", func(w http.ResponseWriter, r *http.Request) { a.view.DefaultSetup(); ok(w) })
	mux.HandleFunc("/api/trig50", func(w http.ResponseWriter, r *http.Request) { a.view.Trigger50(); ok(w) })
	mux.HandleFunc("/api/autoset", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"message": a.view.Autoset()})
	})

	mux.HandleFunc("/api/scan", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			LAN string `json:"lan"`
		}
		_ = readJSON(r, &req)
		cs, notes := a.scan(req.LAN)
		writeJSON(w, map[string]any{"candidates": candList(cs), "notes": notes})
	})

	// startup: look for scopes; connect automatically if exactly one is found
	mux.HandleFunc("/api/startup", func(w http.ResponseWriter, r *http.Request) {
		a.view.mu.Lock()
		done := a.view.startupDone || a.view.connected
		a.view.startupDone = true
		lan, last := a.view.s.LAN, a.view.s.LastDevice
		a.view.mu.Unlock()
		if done {
			writeJSON(w, map[string]any{"action": "none"})
			return
		}
		cs, notes := a.scan(lan)
		var real []*Candidate
		for _, c := range cs {
			if c.Family != "Demo mode" && c.Ready && c.Factory != nil {
				real = append(real, c)
			}
		}
		res := map[string]any{"candidates": candList(cs), "notes": notes}
		switch {
		case len(real) == 1:
			a.view.Connect(real[0])
			res["action"], res["label"] = "connected", real[0].Label
		case len(real) > 1:
			res["action"] = "choose"
			for _, c := range real {
				if c.Label == last {
					a.view.Connect(c)
					res["action"], res["label"] = "connected", c.Label
					break
				}
			}
		default:
			res["action"] = "noscope"
		}
		writeJSON(w, res)
	})

	mux.HandleFunc("/api/connect", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID   string `json:"id"`
			Demo string `json:"demo"`
		}
		_ = readJSON(r, &req)
		var c *Candidate
		if req.Demo != "" {
			sims := scanSim(nil, nil)
			c = sims[0]
			if req.Demo == "bench" {
				c = sims[1]
			}
		} else {
			c = candidateByID(req.ID)
		}
		if c == nil || c.Factory == nil {
			fail(w, errors.New("that scope is no longer in the list - search again"))
			return
		}
		a.view.Connect(c)
		ok(w)
	})
	mux.HandleFunc("/api/disconnect", func(w http.ResponseWriter, r *http.Request) { a.view.Disconnect(); ok(w) })

	mux.HandleFunc("/api/import-firmware", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name string `json:"name"`
			Data string `json:"data"`
		}
		if err := readJSON(r, &req); err != nil {
			fail(w, err)
			return
		}
		data, err := base64.StdEncoding.DecodeString(req.Data)
		if err != nil {
			fail(w, err)
			return
		}
		saved, err := importFirmware(req.Name, data)
		if err != nil {
			fail(w, err)
			return
		}
		writeJSON(w, map[string]any{"saved": saved})
	})

	mux.HandleFunc("/api/driver", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Kind string `json:"kind"`
		}
		_ = readJSON(r, &req)
		var err error
		switch req.Kind {
		case "zadig":
			err = installZadig()
		case "6022":
			err = install6022Driver()
		case "pico":
			openURL(picoSDKURL)
		default:
			err = errors.New("unknown driver")
		}
		if err != nil {
			fail(w, err)
			return
		}
		ok(w)
	})

	mux.HandleFunc("/api/open-csv", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name string `json:"name"`
			Text string `json:"text"`
		}
		if err := readJSON(r, &req); err != nil {
			fail(w, err)
			return
		}
		if err := a.view.OpenCSV(req.Name, []byte(req.Text)); err != nil {
			fail(w, err)
			return
		}
		ok(w)
	})

	mux.HandleFunc("/api/export-csv", func(w http.ResponseWriter, r *http.Request) {
		b, n, err := a.view.ExportCSV()
		if err != nil {
			fail(w, err)
			return
		}
		p, err := saveExport("scope-"+time.Now().Format("20060102-150405")+".csv", b)
		if err != nil {
			fail(w, err)
			return
		}
		writeJSON(w, map[string]any{"path": p, "samples": n})
	})

	// Explain in Probie: save the waveform, then open it in Probie's Explain tab
	mux.HandleFunc("/api/explain-probie", func(w http.ResponseWriter, r *http.Request) {
		b, _, err := a.view.ExportCSV()
		if err != nil {
			fail(w, err)
			return
		}
		p, err := saveExport("scope-"+time.Now().Format("20060102-150405")+".csv", b)
		if err != nil {
			fail(w, err)
			return
		}
		if err := suite.OpenWith("probie", p); err != nil {
			fail(w, err)
			return
		}
		writeJSON(w, map[string]any{"path": p})
	})

	mux.HandleFunc("/api/save-file", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name   string `json:"name"`
			Data   string `json:"data"`
			Base64 bool   `json:"base64"`
		}
		if err := readJSON(r, &req); err != nil {
			fail(w, err)
			return
		}
		data := []byte(req.Data)
		if req.Base64 {
			var err error
			if data, err = base64.StdEncoding.DecodeString(req.Data); err != nil {
				fail(w, err)
				return
			}
		}
		p, err := saveExport(req.Name, data)
		if err != nil {
			fail(w, err)
			return
		}
		writeJSON(w, map[string]string{"path": p})
	})

	mux.HandleFunc("/api/load-setup", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Text string `json:"text"`
		}
		if err := readJSON(r, &req); err != nil {
			fail(w, err)
			return
		}
		if err := a.view.LoadSetup([]byte(req.Text)); err != nil {
			fail(w, err)
			return
		}
		ok(w)
	})

	mux.HandleFunc("/api/open-folder", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Which string `json:"which"`
		}
		_ = readJSON(r, &req)
		dir := exportDir()
		if req.Which == "data" {
			dir = appDataDir()
		}
		_ = os.MkdirAll(dir, 0o755)
		openFolder(dir)
		ok(w)
	})

	mux.HandleFunc("/api/open-url", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			URL string `json:"url"`
		}
		_ = readJSON(r, &req)
		if !strings.HasPrefix(req.URL, "https://") && !strings.HasPrefix(req.URL, "http://") {
			fail(w, errors.New("only web links can be opened"))
			return
		}
		openURL(req.URL)
		ok(w)
	})

	mux.HandleFunc("/api/doc", func(w http.ResponseWriter, r *http.Request) {
		name := "THIRD-PARTY-NOTICES.md"
		if r.URL.Query().Get("name") == "license" {
			name = "LICENSE"
		}
		b, _ := docFS.ReadFile(name)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write(b)
	})

	return a.guard(mux)
}
