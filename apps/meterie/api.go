package meterie

// api.go: local suite API so Wirie's Calcie (and other suite apps) can pull
// live readings from Meterie. Listens on 127.0.0.1 only.

import (
	"encoding/json"
	"fmt"
	"math"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type apiReading struct {
	ID      int      `json:"id"`
	Name    string   `json:"name"`
	Source  string   `json:"source"`
	Value   *float64 `json:"value"` // base SI units; null when overloaded
	Unit    string   `json:"unit"`
	Prefix  string   `json:"prefix"`
	Display string   `json:"display"`
	Text    string   `json:"text"` // human readable, e.g. "4.70 kΩ"
	Mode    string   `json:"mode"`
	Flags   string   `json:"flags"`
	OL      bool     `json:"ol"`
	Time    string   `json:"time"`
	UnixMs  int64    `json:"unix_ms"`
	Main    bool     `json:"main"`
}

type apiSnapshot struct {
	Readings map[int]apiReading
	Main     int
	seq      int
}

var (
	apiSubsMu sync.Mutex
	apiSubs   = map[chan []byte]bool{}
	liveFileT time.Time
)

func suiteDir() string {
	d, err := os.UserConfigDir()
	if err != nil {
		d = "."
	}
	return filepath.Join(d, "nrmluvdev", "suite")
}

func (a *App) updateAPI(ch *Channel) {
	if !a.settings.APIEnabled {
		return
	}
	r := ch.Last
	ar := apiReading{ID: ch.ID, Name: ch.Name, Source: ch.Source, Unit: r.Unit, Prefix: r.Prefix, Display: r.Display,
		Mode: r.ModeLabel(), Flags: flagNames(r.Flags), OL: r.IsOL(), Time: r.T.Format(time.RFC3339Nano), UnixMs: r.T.UnixMilli(), Main: ch.ID == a.mainChan}
	if !r.IsOL() {
		v := ch.DisplayValue(r.Value)
		ar.Value = &v
		ar.Text = FormatSI(v, r.Unit, 5)
	} else {
		ar.Text = "OL " + r.Prefix + r.Unit
	}
	a.apiMu.Lock()
	if a.apiState.Readings == nil {
		a.apiState.Readings = map[int]apiReading{}
	}
	a.apiState.Readings[ch.ID] = ar
	a.apiState.Main = a.mainChan
	a.apiState.seq++
	snap := a.snapshotJSONLocked()
	a.apiMu.Unlock()
	// stream to SSE subscribers
	b, _ := json.Marshal(ar)
	apiSubsMu.Lock()
	for c := range apiSubs {
		select {
		case c <- b:
		default:
		}
	}
	apiSubsMu.Unlock()
	if time.Since(liveFileT) > 500*time.Millisecond {
		liveFileT = time.Now()
		go func() {
			os.MkdirAll(suiteDir(), 0o755)
			tmp := filepath.Join(suiteDir(), "meterie-live.json.tmp")
			if os.WriteFile(tmp, snap, 0o644) == nil {
				os.Rename(tmp, filepath.Join(suiteDir(), "meterie-live.json"))
			}
		}()
	}
}

func (a *App) snapshotJSONLocked() []byte {
	type out struct {
		App      string       `json:"app"`
		Version  string       `json:"version"`
		Updated  string       `json:"updated"`
		Main     int          `json:"main"`
		Readings []apiReading `json:"readings"`
	}
	o := out{App: AppName, Version: AppVersion, Updated: time.Now().Format(time.RFC3339Nano), Main: a.apiState.Main}
	for _, r := range a.apiState.Readings {
		o.Readings = append(o.Readings, r)
	}
	sortReadings(o.Readings)
	b, _ := json.MarshalIndent(o, "", "  ")
	return b
}

func sortReadings(rs []apiReading) {
	for i := 1; i < len(rs); i++ {
		for j := i; j > 0 && rs[j].ID < rs[j-1].ID; j-- {
			rs[j], rs[j-1] = rs[j-1], rs[j]
		}
	}
}

func (a *App) startAPI() {
	if a.apiStarted || !a.settings.APIEnabled {
		return
	}
	a.apiStarted = true
	mux := http.NewServeMux()
	cors := func(w http.ResponseWriter) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Cache-Control", "no-store")
	}
	mux.HandleFunc("/api/v1/ping", func(w http.ResponseWriter, r *http.Request) {
		cors(w)
		json.NewEncoder(w).Encode(map[string]string{"app": AppName, "version": AppVersion, "author": AppAuthor})
	})
	mux.HandleFunc("/api/v1/readings", func(w http.ResponseWriter, r *http.Request) {
		cors(w)
		w.Header().Set("Content-Type", "application/json")
		a.apiMu.Lock()
		b := a.snapshotJSONLocked()
		a.apiMu.Unlock()
		w.Write(b)
	})
	mux.HandleFunc("/api/v1/reading", func(w http.ResponseWriter, r *http.Request) {
		cors(w)
		w.Header().Set("Content-Type", "application/json")
		a.apiMu.Lock()
		id := a.apiState.Main
		if s := r.URL.Query().Get("channel"); s != "" {
			if n, err := strconv.Atoi(strings.TrimPrefix(strings.ToLower(s), "ch")); err == nil {
				id = n
			}
		}
		rd, ok := a.apiState.Readings[id]
		a.apiMu.Unlock()
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "no reading for that channel yet"})
			return
		}
		json.NewEncoder(w).Encode(rd)
	})
	// Plain text value for the simplest possible integration: "4700" (base units)
	mux.HandleFunc("/api/v1/value", func(w http.ResponseWriter, r *http.Request) {
		cors(w)
		a.apiMu.Lock()
		rd, ok := a.apiState.Readings[a.apiState.Main]
		a.apiMu.Unlock()
		if !ok || rd.Value == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		fmt.Fprintf(w, "%s %s", FormatValue(*rd.Value), rd.Unit)
	})
	mux.HandleFunc("/api/v1/stream", func(w http.ResponseWriter, r *http.Request) {
		cors(w)
		fl, ok := w.(http.Flusher)
		if !ok {
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		ch := make(chan []byte, 32)
		apiSubsMu.Lock()
		apiSubs[ch] = true
		apiSubsMu.Unlock()
		defer func() {
			apiSubsMu.Lock()
			delete(apiSubs, ch)
			apiSubsMu.Unlock()
		}()
		for {
			select {
			case <-r.Context().Done():
				return
			case b := <-ch:
				fmt.Fprintf(w, "data: %s\n\n", b)
				fl.Flush()
			case <-time.After(15 * time.Second):
				fmt.Fprint(w, ": keep-alive\n\n")
				fl.Flush()
			}
		}
	})
	go func() {
		port := a.settings.APIPort
		ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err != nil {
			ln, err = net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				return
			}
		}
		actual := ln.Addr().(*net.TCPAddr).Port
		os.MkdirAll(suiteDir(), 0o755)
		disc, _ := json.MarshalIndent(map[string]any{"app": AppName, "version": AppVersion, "pid": os.Getpid(), "port": actual,
			"readings": fmt.Sprintf("http://127.0.0.1:%d/api/v1/readings", actual), "started": time.Now().Format(time.RFC3339)}, "", "  ")
		os.WriteFile(filepath.Join(suiteDir(), "meterie.json"), disc, 0o644)
		http.Serve(ln, mux)
	}()
}

func (a *App) shutdown() {
	for _, d := range a.devices {
		d.Stop()
	}
	a.log.Stop(a, "app closed")
	a.cam.Stop()
	platformStopSound()
	a.saveSettings()
	os.Remove(filepath.Join(suiteDir(), "meterie.json"))
}

var _ = math.Pi
