// Flashie — ESP32 firmware flasher.
// Part of nrmluvdev's free maker tool suite. Made by nrmluvdev.
//
// Native flashing, no external tools:
//   ESP32 family + ESP8266 (ROM bootloader), RP2040/RP2350 (UF2 drive),
//   Arduino AVR (STK500 v1/v2, AVR109) and STM32 (UART bootloader).
// Three-step "Pick board → Pick firmware → Flash" UI.
//
// Build (Windows, single .exe, no console window):
//   GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w -H windowsgui" -o Flashie.exe .

package flashie

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const appVersion = "0.7.1-alpha"

//go:embed ui.html
var uiHTML []byte

//go:embed creator.js creator.css imager.js mpy/*.py teach.js teach-tips.js
var assets embed.FS

var demoMode = os.Getenv("FLASHIE_DEMO") == "1"

// ------------------------------------------------------------ events ---

type hub struct {
	mu       sync.Mutex
	clients  map[chan []byte]struct{}
	ever     bool
	lastDrop time.Time
}

var events = &hub{clients: map[chan []byte]struct{}{}}

func (h *hub) send(v map[string]any) {
	b, _ := json.Marshal(v)
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients {
		select {
		case c <- b:
		default: // slow client: drop
		}
	}
}

func (h *hub) count() int { h.mu.Lock(); defer h.mu.Unlock(); return len(h.clients) }

func emitLog(s string) { events.send(map[string]any{"type": "log", "text": s}) }
func emitStatus(s string) {
	events.send(map[string]any{"type": "status", "text": s})
	emitLog("» " + s)
}
func emitProgress(d, t int)      { events.send(map[string]any{"type": "progress", "done": d, "total": t}) }
func emitDone(ok bool, m string) { events.send(map[string]any{"type": "done", "ok": ok, "text": m}) }

func handleEvents(w http.ResponseWriter, r *http.Request) {
	fl, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", 500)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	c := make(chan []byte, 256)
	events.mu.Lock()
	events.clients[c] = struct{}{}
	events.ever = true
	events.mu.Unlock()
	defer func() {
		events.mu.Lock()
		delete(events.clients, c)
		events.lastDrop = time.Now()
		events.mu.Unlock()
	}()
	fmt.Fprintf(w, ": hello\n\n")
	fl.Flush()
	ping := time.NewTicker(15 * time.Second)
	defer ping.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case b := <-c:
			fmt.Fprintf(w, "data: %s\n\n", b)
			fl.Flush()
		case <-ping.C:
			fmt.Fprintf(w, ": ping\n\n")
			fl.Flush()
		}
	}
}

// --------------------------------------------------------------- jobs ---

var (
	busy      atomic.Bool
	cancelReq atomic.Bool
)

func getPorts() []PortInfo {
	if demoMode {
		return demoPorts()
	}
	return listPorts()
}

func uiReporter() *reporter {
	return &reporter{Log: emitLog, Status: emitStatus, Cancel: cancelReq.Load,
		Progress: func() func(int, int) {
			var last time.Time
			return func(d, t int) {
				if time.Since(last) > 50*time.Millisecond || d == t {
					last = time.Now()
					emitProgress(d, t)
				}
			}
		}()}
}

func handleIdentify(w http.ResponseWriter, r *http.Request) {
	var req struct{ Port, Board string }
	json.NewDecoder(r.Body).Decode(&req)
	if req.Port == "" {
		jsonErr(w, "Pick a board first.")
		return
	}
	if !busy.CompareAndSwap(false, true) {
		jsonErr(w, "Flashie is busy with another job — wait for it to finish.")
		return
	}
	defer busy.Store(false)
	pi, ok := findDevice(req.Port)
	if !ok {
		jsonErr(w, req.Port+" isn't connected any more.")
		return
	}
	stopSession()
	rep := uiReporter()
	info, err := identify(pi, boardByID(req.Board), rep)
	if err != nil {
		emitStatus("Couldn't identify the board.")
		jsonErr(w, friendly(err))
		return
	}
	emitLog("» Identified: " + info.Chip)
	writeJSON(w, info)
}

func handleFlash(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	req := flashReq{Port: q.Get("port"), Board: boardByID(q.Get("board")), FileName: q.Get("name"), Compress: q.Get("compress") != "0"}
	if req.Port == "" {
		jsonErr(w, "Pick a board first.")
		return
	}
	if s := q.Get("offset"); s != "" {
		off, err := parseHex(s)
		if err != nil {
			jsonErr(w, "That address doesn't look right. Use hex like 0x1000.")
			return
		}
		req.Offset, req.HasOff = off, true
	}
	req.Baud, _ = strconv.Atoi(q.Get("baud"))
	req.FlashMB, _ = strconv.Atoi(q.Get("flashmb"))
	if fam := q.Get("hfamily"); fam != "" {
		hb, _ := strconv.Atoi(q.Get("hbaud"))
		req.Hint = &ChipInfo{Family: fam, Proto: q.Get("hproto"), Baud: hb, BoardID: q.Get("hboard")}
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, 65<<20))
	if err != nil || len(data) == 0 {
		jsonErr(w, "The firmware file couldn't be read.")
		return
	}
	if len(data) > 64<<20 {
		jsonErr(w, "That file is far too big for a microcontroller (64 MB max).")
		return
	}
	req.Data = data
	if !busy.CompareAndSwap(false, true) {
		jsonErr(w, "Flashie is busy with another job — wait for it to finish.")
		return
	}
	cancelReq.Store(false)
	writeJSON(w, map[string]any{"started": true})
	go func() {
		defer busy.Store(false)
		defer func() {
			if e := recover(); e != nil {
				emitDone(false, fmt.Sprintf("Something went wrong inside Flashie: %v", e))
			}
		}()
		msg, err := runFlash(req, uiReporter())
		switch {
		case errors.Is(err, errCancelled):
			emitDone(false, "Cancelled. The firmware on the board is now incomplete — flash again before using it.")
		case err != nil:
			emitDone(false, friendly(err))
		default:
			emitDone(true, msg)
		}
	}()
}

func handleLibFlash(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Item, Port, Board, Chip string
		Baud, FlashMB           int
		Compress                bool
		HFamily, HProto, HBoard string
		HBaud                   int
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Item == "" || in.Port == "" {
		jsonErr(w, "Pick a board and a firmware first.")
		return
	}
	req := libFlashReq{Item: in.Item, Port: in.Port, Board: boardByID(in.Board), BoardID: in.Board, Chip: in.Chip,
		Baud: in.Baud, FlashMB: in.FlashMB, Compress: in.Compress}
	if in.HFamily != "" {
		req.Hint = &ChipInfo{Family: in.HFamily, Proto: in.HProto, Baud: in.HBaud, BoardID: in.HBoard}
	}
	if !busy.CompareAndSwap(false, true) {
		jsonErr(w, "Flashie is busy with another job — wait for it to finish.")
		return
	}
	cancelReq.Store(false)
	writeJSON(w, map[string]any{"started": true})
	go func() {
		defer busy.Store(false)
		defer func() {
			if e := recover(); e != nil {
				emitDone(false, fmt.Sprintf("Something went wrong inside Flashie: %v", e))
			}
		}()
		msg, err := runLibFlash(req, uiReporter())
		switch {
		case errors.Is(err, errCancelled):
			emitDone(false, "Cancelled. If flashing had started, the board's firmware is incomplete — flash again before using it.")
		case err != nil:
			emitDone(false, friendly(err))
		default:
			emitDone(true, msg)
		}
	}()
}

func friendly(err error) string {
	s := err.Error()
	if strings.Contains(s, errTimeout.Error()) {
		return "The board stopped answering (" + s + "). Try a slower speed under Advanced, a shorter/better USB cable, or a different USB port."
	}
	if len(s) > 0 {
		s = strings.ToUpper(s[:1]) + s[1:]
	}
	if !strings.HasSuffix(s, ".") {
		s += "."
	}
	return s
}

func parseHex(s string) (uint32, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	base := 10
	if strings.HasPrefix(s, "0x") {
		s, base = s[2:], 16
	}
	v, err := strconv.ParseUint(s, base, 32)
	return uint32(v), err
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func jsonErr(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(400)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// ---------------------------------------------------------------- main ---

func Main() {
	if len(os.Args) >= 4 && os.Args[1] == "--img-helper" { // elevated image writer
		os.Exit(runImgHelper(os.Args[2], os.Args[3]))
	}
	if demoMode {
		startDemoLibrary()
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Write(uiHTML)
	})
	mux.HandleFunc("/creator.js", func(w http.ResponseWriter, r *http.Request) {
		b, _ := assets.ReadFile("creator.js")
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Write(b)
	})
	for _, f := range []string{"teach.js", "teach-tips.js"} { // teaching mode
		f := f
		mux.HandleFunc("/"+f, func(w http.ResponseWriter, r *http.Request) {
			b, _ := assets.ReadFile(f)
			w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")
			w.Write(b)
		})
	}
	mux.HandleFunc("/creator.css", func(w http.ResponseWriter, r *http.Request) {
		b, _ := assets.ReadFile("creator.css")
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Write(b)
	})
	mux.HandleFunc("/api/creator/drivers", func(w http.ResponseWriter, r *http.Request) {
		out := map[string]string{}
		ents, _ := assets.ReadDir("mpy")
		for _, e := range ents {
			b, _ := assets.ReadFile("mpy/" + e.Name())
			out[e.Name()] = string(b)
		}
		writeJSON(w, out)
	})
	mux.HandleFunc("/api/creator/check", post(handleCreatorCheck))
	mux.HandleFunc("/api/creator/run", post(handleCreatorRun))
	mux.HandleFunc("/api/creator/stop", post(handleCreatorStop))
	mux.HandleFunc("/api/creator/send", post(handleCreatorSend))
	mux.HandleFunc("/api/creator/disconnect", post(handleCreatorDisconnect))
	mux.HandleFunc("/api/creator/monitor", post(handleCreatorMonitor))
	mux.HandleFunc("/api/creator/arduino", post(handleCreatorArduino))
	imgRoutes(mux)
	mux.HandleFunc("/api/info", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"version": appVersion, "boards": boards, "cyds": cyds, "demo": demoMode})
	})
	mux.HandleFunc("/api/ports", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, getDevices())
	})
	mux.HandleFunc("/api/identify", post(handleIdentify))
	mux.HandleFunc("/api/flash", post(handleFlash))
	mux.HandleFunc("/api/cancel", post(func(w http.ResponseWriter, r *http.Request) {
		cancelReq.Store(true)
		writeJSON(w, map[string]any{"ok": true})
	}))
	mux.HandleFunc("/api/events", handleEvents)
	mux.HandleFunc("/api/library", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		n, sz := cacheInfo()
		writeJSON(w, map[string]any{"items": libraryFor(q.Get("board"), q.Get("chip")), "cacheFiles": n, "cacheBytes": sz, "cacheDir": cacheRoot()})
	})
	mux.HandleFunc("/api/library/resolve", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		res, err := resolveItem(q.Get("id"), q.Get("board"), q.Get("chip"))
		if err != nil {
			jsonErr(w, friendly(err))
			return
		}
		writeJSON(w, res)
	})
	mux.HandleFunc("/api/library/flash", post(handleLibFlash))
	mux.HandleFunc("/api/library/clearcache", post(func(w http.ResponseWriter, r *http.Request) {
		cacheClear()
		writeJSON(w, map[string]any{"ok": true})
	}))
	mux.HandleFunc("/api/cydmatch", post(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(io.LimitReader(r.Body, 65<<20))
		writeJSON(w, matchCYDFirmware(r.URL.Query().Get("name"), data))
	}))

	addr := "127.0.0.1:0"
	if v := os.Getenv("FLASHIE_ADDR"); v != "" {
		addr = v
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		showError("Flashie", "Couldn't start: "+err.Error())
		os.Exit(1)
	}
	url := "http://" + ln.Addr().String() + "/"
	selfBase = url
	go func() {
		if os.Getenv("FLASHIE_NOWINDOW") == "1" {
			println("Flashie listening on " + url)
			return
		}
		if err := launchWindow(url); err != nil {
			showError("Flashie", "Couldn't open the Flashie window. Open this address in your browser:\n\n"+url)
		}
	}()
	go watchdog()
	http.Serve(ln, mux)
}

func post(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", 405)
			return
		}
		// Only accept requests from our own page (blocks drive-by sites).
		if o := r.Header.Get("Origin"); o != "" && !strings.HasPrefix(o, "http://"+r.Host) {
			http.Error(w, "forbidden", 403)
			return
		}
		h(w, r)
	}
}

// watchdog quits once the window is closed (no event stream for a while).
func watchdog() {
	if os.Getenv("FLASHIE_NOWINDOW") == "1" {
		return
	}
	started := time.Now()
	for range time.Tick(time.Second) {
		events.mu.Lock()
		n, ever, drop := len(events.clients), events.ever, events.lastDrop
		events.mu.Unlock()
		if busy.Load() {
			continue // never quit mid-flash
		}
		if !ever && time.Since(started) > 3*time.Minute {
			os.Exit(0)
		}
		if ever && n == 0 && time.Since(drop) > 6*time.Second {
			os.Exit(0)
		}
	}
}
