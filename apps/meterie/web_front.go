//go:build linux || darwin

package meterie

// web_front.go: Meterie's window on Linux, Raspberry Pi OS and macOS.
//
// Meterie renders every frame with its own renderer (exactly as on Windows)
// into a pixel buffer; changed 64×64 tiles go to a browser "app" window over a
// local connection. Text is drawn into the same buffer from glyphs the window
// renders with the embedded Inter and DejaVu Sans Mono fonts and sends back,
// so overlays and dialogs cover text just as on Windows. Measuring text uses
// those fonts' exact advance widths (fonts/metrics.json).

import (
	"bytes"
	"compress/flate"
	"embed"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image/png"
	"io"
	"io/fs"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"tinkerie/shared/appwindow"
)

//go:embed web
var webFS embed.FS

// front is the running window, for the platform functions (sound, clipboard).
var front *webFront

// ---------------------------------------------------------------- glyphs

type glyph struct {
	w, h int
	a    []byte // alpha mask
}

type glyphNeed struct {
	K    string  `json:"k"`
	CP   int     `json:"cp"`
	Px   float64 `json:"px"`
	Face string  `json:"face"`
	Icon bool    `json:"icon"`
	W    int     `json:"w"`
	H    int     `json:"h"`
	BX   float64 `json:"bx"`
	BY   float64 `json:"by"`
}

type glyphCache struct {
	mu     sync.Mutex
	g      map[string]*glyph
	need   map[string]glyphNeed // wanted, not yet asked for
	asked  map[string]time.Time
	adv    map[string]map[rune]float64
	asc    float64
	desc   float64
	loaded bool
}

func newGlyphCache() *glyphCache {
	gc := &glyphCache{g: map[string]*glyph{}, need: map[string]glyphNeed{}, asked: map[string]time.Time{}, adv: map[string]map[rune]float64{}}
	b, err := webFS.ReadFile("web/fonts/metrics.json")
	if err != nil {
		return gc
	}
	var m struct {
		Asc, Desc float64
		Faces     map[string]map[string]float64
	}
	if json.Unmarshal(b, &m) != nil {
		return gc
	}
	gc.asc, gc.desc = m.Asc, m.Desc
	for face, t := range m.Faces {
		gc.adv[face] = map[rune]float64{}
		for k, v := range t {
			if n, err := strconv.Atoi(k); err == nil {
				gc.adv[face][rune(n)] = v
			}
		}
	}
	gc.loaded = true
	return gc
}

const glyphPad = 2

// webText is Meterie's TextEngine on Linux and macOS.
type webText struct {
	gc   *glyphCache
	c    *Canvas
	clip Rect
}

func (gc *glyphCache) forCanvas(c *Canvas) *webText {
	return &webText{gc: gc, c: c, clip: Rect{0, 0, c.W, c.H}}
}

func face(f Font) string {
	switch {
	case f.Mono:
		return "m"
	case f.Weight >= 650:
		return "b"
	case f.Weight >= 550:
		return "s"
	}
	return "r"
}

func isIcon(f Font, r rune) bool { return f.Icon || (r >= 0xE000 && r <= 0xF8FF) }

func (t *webText) advance(f Font, fc string, r rune) float64 {
	if isIcon(f, r) {
		return f.Size
	}
	a, ok := t.gc.adv[fc][r]
	if !ok {
		a = t.gc.adv[fc]['?']
		if a == 0 {
			a = 0.6
		}
	}
	return a * f.Size
}

func (t *webText) Measure(s string, f Font) int {
	fc := face(f)
	w := 0.0
	for _, r := range s {
		w += t.advance(f, fc, r)
	}
	return int(w + 0.5)
}

func (t *webText) LineHeight(f Font) int { return int(f.Size*1.30 + 0.5) }

func (t *webText) SetClip(r Rect) { t.clip = r }

func (t *webText) Draw(s string, x, y int, f Font, col Color) {
	if col.A == 0 {
		return
	}
	fc := face(f)
	q := int(math.Round(f.Size * 4)) // quarter-pixel font sizes
	px := float64(q) / 4
	lh := int(px*1.30 + 0.5)
	pen := 0.0
	t.gc.mu.Lock()
	defer t.gc.mu.Unlock()
	for _, r := range s {
		adv := t.advance(f, fc, r)
		if r != ' ' && r != ' ' {
			icon := isIcon(f, r)
			k := fmt.Sprintf("%d|%d|%s|%t", r, q, fc, icon)
			if g := t.gc.g[k]; g != nil {
				t.blend(g, x+int(math.Round(pen))-glyphPad, y-glyphPad, col)
			} else if _, ok := t.gc.need[k]; !ok {
				n := glyphNeed{K: k, CP: int(r), Px: px, Face: fc, Icon: icon, H: lh + 2*glyphPad, BX: glyphPad}
				if icon {
					n.W = int(math.Ceil(px)) + 2*glyphPad
					n.BY = glyphPad + float64(lh)/2 - px/2 // top of the icon square
				} else {
					n.W = int(math.Ceil(adv)) + 3*glyphPad
					n.BY = glyphPad + math.Round(px*(t.gc.asc+(1.30-t.gc.asc-t.gc.desc)/2)) // baseline
				}
				t.gc.need[k] = n
			}
		}
		pen += adv
	}
}

// blend draws a glyph mask in col, clipped to the text clip and the canvas.
func (t *webText) blend(g *glyph, gx, gy int, col Color) {
	c := t.c
	cl := t.clip.Intersect(c.clip).Intersect(Rect{0, 0, c.W, c.H})
	x0, y0 := max(gx, cl.X), max(gy, cl.Y)
	x1, y1 := min(gx+g.w, cl.X+cl.W), min(gy+g.h, cl.Y+cl.H)
	ca := int(col.A)
	for y := y0; y < y1; y++ {
		row := (y-gy)*g.w - gx
		off := y*c.Stride + x0*4
		for x := x0; x < x1; x, off = x+1, off+4 {
			m := int(g.a[row+x])
			if m == 0 {
				continue
			}
			a := m * ca / 255
			p := c.Pix[off : off+4 : off+4]
			p[0] = uint8(int(p[0]) + (int(col.B)-int(p[0]))*a/255)
			p[1] = uint8(int(p[1]) + (int(col.G)-int(p[1]))*a/255)
			p[2] = uint8(int(p[2]) + (int(col.R)-int(p[2]))*a/255)
			p[3] = 255
		}
	}
}

// takeNeeds returns glyphs to ask the window for (again after 3 s if lost).
func (gc *glyphCache) takeNeeds() []glyphNeed {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	var out []glyphNeed
	now := time.Now()
	for k, n := range gc.need {
		if t, ok := gc.asked[k]; ok && now.Sub(t) < 3*time.Second {
			continue
		}
		gc.asked[k] = now
		out = append(out, n)
		if len(out) >= 400 {
			break
		}
	}
	return out
}

// addGlyphs stores glyphs uploaded by the window: [u16 keyLen][u16 w][u16 h][key][w*h alpha]...
func (gc *glyphCache) addGlyphs(b []byte) int {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	n := 0
	for len(b) >= 6 {
		kl, w, h := int(binary.LittleEndian.Uint16(b)), int(binary.LittleEndian.Uint16(b[2:])), int(binary.LittleEndian.Uint16(b[4:]))
		if len(b) < 6+kl+w*h {
			break
		}
		k := string(b[6 : 6+kl])
		a := append([]byte(nil), b[6+kl:6+kl+w*h]...)
		b = b[6+kl+w*h:]
		gc.g[k] = &glyph{w: w, h: h, a: a}
		delete(gc.need, k)
		delete(gc.asked, k)
		n++
	}
	return n
}

// ----------------------------------------------------------------- window

type webFront struct {
	app  *App
	text *glyphCache

	mu       sync.Mutex
	in       Input
	w, h     int
	dpr      float64
	c        *Canvas
	prev     []byte   // the last frame, for finding changed tiles
	seq      uint64   // frame number
	tileSeq  []uint64 // per tile: the frame it last changed in
	sizeSeq  uint64   // frame of the last resize: older windows need everything
	changed  chan struct{}
	wantDraw bool
	lastDraw time.Time
	lastPoll time.Time
	polled   bool
	upLater  bool // a button release that arrived before its press was drawn
	credit   bool // the window is waiting for a newer frame than it has

	// METERIE_FRONT_STATS=1 logs frame rate, data sent and time per frame.
	stats                  bool
	nFrames, nSent, nBytes int
	tRender, tEncode       time.Duration
	statsAt                time.Time

	clipOut   *string
	sound     *soundCmd
	stopSnd   bool
	sounds    map[int][]byte
	nextSound int
}

type soundCmd struct {
	ID   int  `json:"id"`
	Loop bool `json:"loop"`
}

func (f *webFront) playSound(wav []byte, loop bool) {
	f.nextSound++
	f.sounds = map[int][]byte{f.nextSound: wav} // keep only the newest
	f.sound = &soundCmd{f.nextSound, loop}
}

func (f *webFront) stopSound()            { f.stopSnd = true; f.sound = nil }
func (f *webFront) setClipboard(s string) { f.clipOut = &s }

// runWeb starts Meterie with its window in a browser app window.
func runWeb() {
	app := NewApp()
	f := &webFront{app: app, text: newGlyphCache(), dpr: 1, changed: make(chan struct{}), sounds: map[int][]byte{}}
	front = f
	f.stats, f.statsAt = os.Getenv("METERIE_FRONT_STATS") == "1", time.Now()
	app.ui.Clipboard = platformClipboardGet
	app.ui.SetClipboard = platformClipboardSet
	wakeFn = func() {
		f.mu.Lock()
		f.wantDraw = true
		f.mu.Unlock()
	}
	f.resize(1440, 900, 1)
	app.startAPI()
	if len(os.Args) > 1 {
		p := os.Args[1]
		if strings.HasSuffix(strings.ToLower(p), ".meterie") {
			if err := app.loadSession(p); err == nil {
				app.sessionPath = p
			}
		} else if strings.HasSuffix(strings.ToLower(p), ".csv") {
			app.importTimeline(p)
		}
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	web, _ := fs.Sub(webFS, "web")
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(web)))
	mux.HandleFunc("/icon.png", f.serveIcon)
	mux.HandleFunc("/frame", f.serveFrame)
	mux.HandleFunc("/input", f.serveInput)
	mux.HandleFunc("/glyphs", func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(io.LimitReader(r.Body, 64<<20))
		if f.text.addGlyphs(b) > 0 {
			f.mu.Lock()
			f.wantDraw = true
			f.mu.Unlock()
		}
	})
	mux.HandleFunc("/sound", func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(r.URL.Query().Get("id"))
		f.mu.Lock()
		b := f.sounds[id]
		f.mu.Unlock()
		w.Header().Set("Content-Type", "audio/wav")
		w.Write(b)
	})
	go http.Serve(ln, guardLocal(mux))
	url := "http://" + ln.Addr().String() + "/"
	log.Println("Meterie window at", url)
	base, _ := os.UserConfigDir()
	appwindow.Open(url, filepath.Join(base, "nrmluvdev", "Meterie", "window"))

	f.loop()
	app.shutdown()
}

// guardLocal refuses requests made by other web pages (a page can't set a
// custom header on a cross-site request without a CORS preflight, which we
// never allow), except plain page loads.
func guardLocal(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if o := r.Header.Get("Origin"); o != "" && !strings.HasPrefix(o, "http://"+r.Host) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func (f *webFront) resize(w, h int, dpr float64) {
	f.w, f.h, f.dpr = w, h, dpr
	f.c = NewCanvas(w, h)
	f.prev = nil
	f.tileSeq = make([]uint64, ((w+tileSize-1)/tileSize)*((h+tileSize-1)/tileSize))
	f.sizeSeq = f.seq + 1
	f.wantDraw = true
}

func (f *webFront) wantsFrame() bool {
	inboxMu.Lock()
	n := len(inbox)
	inboxMu.Unlock()
	a := f.app
	if n > 0 || a.ui.Animating || a.ui.teachIntro > 0 {
		return true
	}
	return (a.ui.Ambient || a.cam.running || a.log.Active()) && time.Since(f.lastDraw) > 40*time.Millisecond
}

// loop renders frames as needed and quits once the window has been closed.
func (f *webFront) loop() {
	t := time.NewTicker(15 * time.Millisecond)
	defer t.Stop()
	start := time.Now()
	for range t.C {
		f.mu.Lock()
		if f.polled && time.Since(f.lastPoll) > 8*time.Second || !f.polled && time.Since(start) > 90*time.Second {
			f.mu.Unlock()
			return
		}
		// Render when there's something to show and the window is ready for it
		// (at least 10 times a second regardless, so everything keeps ticking).
		if (f.wantDraw || f.wantsFrame()) && (f.credit || time.Since(f.lastDraw) > 100*time.Millisecond) {
			f.wantDraw = false
			f.render()
		}
		f.mu.Unlock()
	}
}

// render draws one frame and works out which tiles changed (f.mu held).
func (f *webFront) render() {
	a := f.app
	us := a.settings.UIScale
	if us <= 0 {
		us = 1
	}
	a.ui.S = f.dpr * us
	f.c.ResetClip()
	t0 := time.Now()
	a.Frame(f.c, f.text.forCanvas(f.c), f.in)
	f.lastDraw = time.Now()
	f.credit = false
	f.tRender += f.lastDraw.Sub(t0)
	f.nFrames++
	f.in.Pressed, f.in.Released, f.in.RPressed, f.in.Dbl = false, false, false, false
	f.in.Wheel = 0
	f.in.Keys = f.in.Keys[:0]
	f.in.Chars = f.in.Chars[:0]
	if f.upLater { // deliver a quick click's release in the next frame, as Windows does
		f.upLater = false
		f.in.Down, f.in.Released = false, true
		f.wantDraw = true
	}
	if a.ui.Animating {
		f.wantDraw = true
	}
	f.text.mu.Lock()
	if len(f.text.need) > 0 {
		f.wantDraw = true // redraw once the window has sent the glyphs
	}
	f.text.mu.Unlock()

	t1 := time.Now()
	changed := f.markChangedTiles()
	f.tEncode += time.Since(t1)
	if f.stats && time.Since(f.statsAt) > 5*time.Second {
		d := time.Since(f.statsAt).Seconds()
		log.Printf("front: %.1f frames/s, %.1f sent/s, %.0f KB/s, render %.1f ms, diff %.1f ms per frame",
			float64(f.nFrames)/d, float64(f.nSent)/d, float64(f.nBytes)/d/1024,
			f.tRender.Seconds()*1000/float64(max(f.nFrames, 1)), f.tEncode.Seconds()*1000/float64(max(f.nFrames, 1)))
		f.nFrames, f.nSent, f.nBytes, f.tRender, f.tEncode, f.statsAt = 0, 0, 0, 0, 0, time.Now()
	}
	if !changed {
		return // nothing changed on screen
	}
	close(f.changed)
	f.changed = make(chan struct{})
}

const tileSize = 64

// markChangedTiles compares the new frame with the last one and stamps every
// changed tile with the next frame number. It reports whether anything changed.
func (f *webFront) markChangedTiles() bool {
	c := f.c
	cols := (c.W + tileSize - 1) / tileSize
	next := f.seq + 1
	any := false
	for ty, row := 0, 0; ty < c.H; ty, row = ty+tileSize, row+1 {
		for tx, col := 0, 0; tx < c.W; tx, col = tx+tileSize, col+1 {
			w, h := min(tileSize, c.W-tx), min(tileSize, c.H-ty)
			if f.prev == nil || tileChanged(c, f.prev, tx, ty, w, h) {
				f.tileSeq[row*cols+col] = next
				any = true
			}
		}
	}
	if f.prev == nil {
		f.prev = make([]byte, len(c.Pix))
	}
	copy(f.prev, c.Pix)
	if any {
		f.seq = next
	}
	return any
}

// encodeSince packs every tile that changed after frame since, as
// [u16 x][u16 y][u16 w][u16 h][RGBA...], deflate-compressed (f.mu held).
func (f *webFront) encodeSince(since uint64) []byte {
	c := f.c
	if since < f.sizeSeq {
		since = 0 // the window has an older size: send everything
	}
	cols := (c.W + tileSize - 1) / tileSize
	var raw bytes.Buffer
	for ty, row := 0, 0; ty < c.H; ty, row = ty+tileSize, row+1 {
		for tx, col := 0, 0; tx < c.W; tx, col = tx+tileSize, col+1 {
			if since != 0 && f.tileSeq[row*cols+col] <= since {
				continue
			}
			w, h := min(tileSize, c.W-tx), min(tileSize, c.H-ty)
			var hd [8]byte
			binary.LittleEndian.PutUint16(hd[0:], uint16(tx))
			binary.LittleEndian.PutUint16(hd[2:], uint16(ty))
			binary.LittleEndian.PutUint16(hd[4:], uint16(w))
			binary.LittleEndian.PutUint16(hd[6:], uint16(h))
			raw.Write(hd[:])
			for y := ty; y < ty+h; y++ {
				line := c.Pix[y*c.Stride+tx*4 : y*c.Stride+(tx+w)*4]
				for i := 0; i < len(line); i += 4 { // BGRA -> RGBA
					raw.Write([]byte{line[i+2], line[i+1], line[i], 255})
				}
			}
		}
	}
	if raw.Len() == 0 {
		return nil
	}
	var out bytes.Buffer
	zw, _ := flate.NewWriter(&out, flate.BestSpeed)
	zw.Write(raw.Bytes())
	zw.Close()
	return out.Bytes()
}

func tileChanged(c *Canvas, prev []byte, tx, ty, w, h int) bool {
	for y := ty; y < ty+h; y++ {
		o := y*c.Stride + tx*4
		if !bytes.Equal(c.Pix[o:o+w*4], prev[o:o+w*4]) {
			return true
		}
	}
	return false
}

// serveFrame answers the window's long poll: changed tiles since its last
// frame (or everything if it fell behind), plus glyph requests and commands.
func (f *webFront) serveFrame(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	since, _ := strconv.ParseUint(q.Get("since"), 10, 64)
	cw, _ := strconv.Atoi(q.Get("w"))
	ch, _ := strconv.Atoi(q.Get("h"))
	dpr, _ := strconv.ParseFloat(q.Get("dpr"), 64)
	f.mu.Lock()
	f.lastPoll, f.polled = time.Now(), true
	if cw > 0 && ch > 0 && dpr > 0 && (cw != f.w || ch != f.h || dpr != f.dpr) && cw <= 8192 && ch <= 8192 {
		f.resize(cw, ch, dpr)
		since = 0
	}
	wait := f.changed
	ready := f.seq > since
	f.mu.Unlock()
	if !ready {
		f.mu.Lock()
		f.credit = true // one frame, please
		f.mu.Unlock()
		select {
		case <-wait:
		case <-time.After(5 * time.Second):
		case <-r.Context().Done():
			return
		}
	}

	f.mu.Lock()
	f.lastPoll = time.Now()
	var tiles []byte
	if f.seq != since {
		tiles = f.encodeSince(since)
	}
	m := map[string]any{"seq": f.seq, "w": f.w, "h": f.h, "cursor": f.app.ui.Cursor, "title": "Meterie"}
	if f.clipOut != nil {
		m["clip"] = *f.clipOut
		f.clipOut = nil
	}
	if f.stopSnd {
		m["stopSound"] = true
		f.stopSnd = false
	}
	if f.sound != nil {
		m["sound"] = f.sound
		f.sound = nil
	}
	f.nSent++
	f.nBytes += len(tiles)
	f.mu.Unlock()
	if need := f.text.takeNeeds(); len(need) > 0 {
		m["need"] = need
	}
	js, _ := json.Marshal(m)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", "no-store")
	var hd [4]byte
	binary.LittleEndian.PutUint32(hd[:], uint32(len(js)))
	w.Write(hd[:])
	w.Write(js)
	w.Write(tiles)
}

var keyNames = map[string]Key{
	"Back": KBack, "Delete": KDelete, "Left": KLeft, "Right": KRight, "Up": KUp, "Down": KDown, "Home": KHome,
	"End": KEnd, "Enter": KEnter, "Escape": KEscape, "Tab": KTab, "Space": KSpace, "A": KA, "C": KC, "V": KV,
	"X": KX, "S": KS, "O": KO, "N": KN, "Z": KZ, "F1": KF1, "H": KH, "R": KR, "M": KM, "Plus": KPlus,
	"Minus": KMinus, "0": K0, "F2": KF2,
}

// serveInput applies a batch of mouse and keyboard events from the window.
func (f *webFront) serveInput(w http.ResponseWriter, r *http.Request) {
	var evs []struct {
		T     string  `json:"t"`
		X     int     `json:"x"`
		Y     int     `json:"y"`
		B     int     `json:"b"`
		D     float64 `json:"d"`
		K     string  `json:"k"`
		C     string  `json:"c"`
		S     string  `json:"s"`
		Ctrl  bool    `json:"ctrl"`
		Shift bool    `json:"shift"`
	}
	if json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&evs) != nil {
		http.Error(w, "bad input", 400)
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	in := &f.in
	for _, e := range evs {
		switch e.T {
		case "move":
			in.MX, in.MY, in.HasMouse = e.X, e.Y, true
		case "down":
			in.MX, in.MY, in.HasMouse = e.X, e.Y, true
			in.Ctrl, in.Shift = e.Ctrl, e.Shift
			if e.B == 2 {
				in.RDown, in.RPressed = true, true
			} else if e.B == 0 {
				in.Down, in.Pressed = true, true
			}
		case "up":
			if e.B == 2 {
				in.RDown = false
			} else if e.B == 0 && in.Down {
				if in.Pressed {
					f.upLater = true // the press hasn't been drawn yet
				} else {
					in.Down, in.Released = false, true
				}
			}
		case "dbl":
			in.Dbl = true
		case "wheel":
			in.Wheel += e.D
			in.MX, in.MY = e.X, e.Y
		case "leave":
			in.HasMouse = false
		case "key":
			in.Ctrl, in.Shift = e.Ctrl, e.Shift
			if k, ok := keyNames[e.K]; ok {
				in.Keys = append(in.Keys, k)
			}
		case "char":
			in.Chars = append(in.Chars, []rune(e.C)...)
		case "mods":
			in.Ctrl, in.Shift = e.Ctrl, e.Shift
		case "paste":
			clip = e.S
		case "resize":
		}
	}
	f.wantDraw = true
}

func (f *webFront) serveIcon(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/png")
	png.Encode(w, renderLogoImage(128))
}
