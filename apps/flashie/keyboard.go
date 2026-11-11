// Flashie — Keyboard tab: makes Arduino firmware for keyboards designed in
// Keebie (Wirie), compiles it with arduino-cli for AVR (ATmega32U4), RP2040
// or STM32, and flashes it. Made by nrmluvdev.

package flashie

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// kbInitial is a .keebie.json passed with --keyboard (Wirie's "Firmware in Flashie").
var kbInitial string

type kbCoreSpec struct {
	core, url, ext string
	libs           []string
}

var kbCores = map[string]kbCoreSpec{
	"avr":    {"arduino:avr", "", ".hex", []string{"HID-Project"}},
	"rp2040": {"rp2040:rp2040", "https://github.com/earlephilhower/arduino-pico/releases/download/global/package_rp2040_index.json", ".uf2", nil},
	"stm32":  {"STMicroelectronics:stm32", "https://github.com/stm32duino/BoardManagerFiles/raw/main/package_stmicroelectronics_index.json", ".bin", nil},
}

var kbLibAllow = map[string]bool{"HID-Project": true, "Adafruit SSD1306": true, "Adafruit GFX Library": true, "Adafruit NeoPixel": true}

var fqbnOK = regexp.MustCompile(`^[A-Za-z0-9_.-]+:[A-Za-z0-9_.-]+:[A-Za-z0-9_.-]+(:[A-Za-z0-9_.,=-]+)?$`)

var kbLast struct {
	sync.Mutex
	data []byte
	name string
}

// keyboardBuild compiles a keyboard sketch and returns the firmware file.
func keyboardBuild(core, fqbn, code string, libs []string, r *reporter) ([]byte, string, error) {
	spec, ok := kbCores[core]
	if !ok {
		return nil, "", errors.New("Flashie can't build firmware for this controller (it needs an AVR, RP2040 or STM32 chip)")
	}
	if !fqbnOK.MatchString(fqbn) || !strings.HasPrefix(fqbn, spec.core+":") {
		return nil, "", errors.New("the keyboard file names an unknown board")
	}
	name := "keebie_fw.ino" + spec.ext
	if demoMode {
		r.status("Compiling the keyboard firmware (demo: pretending — real builds use arduino-cli)…")
		emitConsole("$ arduino-cli compile --fqbn "+fqbn+" keebie_fw\n", "build")
		time.Sleep(500 * time.Millisecond)
		if !strings.Contains(code, "void setup()") {
			return nil, "", errors.New("keebie_fw.ino: error: 'setup' was not declared")
		}
		img := fakeImage("keyboard", 14000)
		switch core {
		case "avr":
			return makeHexFile(img, 0), name, nil
		case "rp2040":
			return binToUF2(img, 0x10000000, famRP2040), name, nil
		}
		return img, name, nil
	}
	exe, err := ensureArduinoCLI(r)
	if err != nil {
		return nil, "", err
	}
	home := arduinoHome()
	extra := []string{}
	if spec.url != "" {
		extra = []string{"--additional-urls", spec.url}
	}
	marker := filepath.Join(home, strings.ReplaceAll(spec.core, ":", "_")+"-core.ok")
	if _, err := os.Stat(marker); err != nil {
		r.status("Setting up the " + map[string]string{"avr": "Arduino AVR", "rp2040": "RP2040", "stm32": "STM32"}[core] + " compiler (one-time download, a few minutes)…")
		if err := runCLI(exe, r, append([]string{"core", "update-index"}, extra...)...); err != nil {
			return nil, "", err
		}
		if err := runCLI(exe, r, append([]string{"core", "install", spec.core}, extra...)...); err != nil {
			return nil, "", err
		}
		os.WriteFile(marker, []byte(time.Now().Format(time.RFC3339)), 0o644)
	}
	for _, l := range append(append([]string{}, spec.libs...), libs...) {
		if !kbLibAllow[l] {
			continue
		}
		lm := filepath.Join(home, "lib-"+strings.ReplaceAll(l, " ", "_")+".ok")
		if _, err := os.Stat(lm); err == nil {
			continue
		}
		r.status("Installing the " + l + " library…")
		if err := runCLI(exe, r, "lib", "install", l); err != nil {
			return nil, "", err
		}
		os.WriteFile(lm, nil, 0o644)
	}
	sketch := filepath.Join(home, "sketches", "keebie_fw")
	os.MkdirAll(sketch, 0o755)
	if err := os.WriteFile(filepath.Join(sketch, "keebie_fw.ino"), []byte(code), 0o644); err != nil {
		return nil, "", err
	}
	outDir := filepath.Join(home, "build-keyboard")
	os.RemoveAll(outDir)
	r.status("Compiling the keyboard firmware…")
	if err := runCLI(exe, r, append([]string{"compile", "--fqbn", fqbn, "--output-dir", outDir, sketch}, extra...)...); err != nil {
		return nil, "", err
	}
	b, err := os.ReadFile(filepath.Join(outDir, name))
	return b, name, err
}

func kbReporter() *reporter {
	return &reporter{
		Log: func(s string) { emitConsole(s+"\n", "info") },
		Status: func(s string) {
			events.send(map[string]any{"type": "kbstatus", "text": s})
			emitConsole("» "+s+"\n", "info")
		},
		Progress: func() func(int, int) {
			var last time.Time
			return func(d, t int) {
				if time.Since(last) > 80*time.Millisecond || d == t {
					last = time.Now()
					events.send(map[string]any{"type": "kbprogress", "done": d, "total": t})
				}
			}
		}(),
		Cancel: cancelReq.Load,
	}
}

func kbDone(ok bool, msg string) {
	events.send(map[string]any{"type": "kbdone", "ok": ok, "text": msg})
}

func handleKeyboardBuild(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Core, Fqbn, Code, Port, Board, Name string
		Libs                                []string
		Flash                               bool
	}
	if err := decodeJSON(r, &in); err != nil || in.Code == "" {
		jsonErr(w, "Nothing to compile.")
		return
	}
	if !busy.CompareAndSwap(false, true) {
		jsonErr(w, "Flashie is busy with another job — wait for it to finish.")
		return
	}
	cancelReq.Store(false)
	writeJSON(w, map[string]any{"started": true})
	go func() {
		defer busy.Store(false)
		rep := kbReporter()
		data, name, err := keyboardBuild(in.Core, in.Fqbn, in.Code, in.Libs, rep)
		if err != nil {
			kbDone(false, friendly(err))
			return
		}
		base := safeFileName(in.Name)
		if base == "" {
			base = "keyboard"
		}
		kbLast.Lock()
		kbLast.data, kbLast.name = data, base+"-firmware"+filepath.Ext(name)
		kbLast.Unlock()
		if !in.Flash || in.Port == "" {
			kbDone(true, fmt.Sprintf("Firmware built (%s). Save it, or pick your keyboard's port and press Make & flash.", humanBytes(len(data))))
			return
		}
		rep.status("Writing the firmware to your keyboard…")
		msg, err := runFlash(flashReq{Port: in.Port, Board: boardByID(in.Board), Data: data, FileName: name}, rep)
		if err != nil {
			kbDone(false, friendly(err))
			return
		}
		kbDone(true, msg+" Your keyboard restarts with the new keymap.")
	}()
}

var unsafeName = regexp.MustCompile(`[^A-Za-z0-9 _.-]+`)

func safeFileName(s string) string {
	s = strings.TrimSpace(unsafeName.ReplaceAllString(s, ""))
	if len(s) > 60 {
		s = s[:60]
	}
	return s
}

// keyboardDirs: where Wirie keeps its exports (Documents\Wirie\Exports).
func keyboardDirs() []string {
	var roots []string
	if h, err := os.UserHomeDir(); err == nil {
		roots = append(roots, filepath.Join(h, "Documents"), filepath.Join(h, "OneDrive", "Documents"))
	}
	for _, env := range []string{"OneDrive", "OneDriveConsumer", "OneDriveCommercial", "USERPROFILE"} {
		if v := os.Getenv(env); v != "" {
			roots = append(roots, filepath.Join(v, "Documents"))
		}
	}
	seen := map[string]bool{}
	var out []string
	for _, r := range roots {
		d := filepath.Join(r, "Wirie", "Exports")
		k := strings.ToLower(filepath.Clean(d))
		if !seen[k] {
			seen[k] = true
			out = append(out, d)
		}
	}
	return out
}

type kbFile struct {
	Path     string `json:"path"`
	Name     string `json:"name"`
	Modified int64  `json:"modified"`
}

func listKeyboards() []kbFile {
	out := []kbFile{}
	for _, d := range keyboardDirs() {
		ents, _ := os.ReadDir(d)
		for _, e := range ents {
			if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".keebie.json") {
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			out = append(out, kbFile{Path: filepath.Join(d, e.Name()), Name: strings.TrimSuffix(e.Name(), ".keebie.json"), Modified: info.ModTime().Unix()})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Modified > out[j].Modified })
	return out
}

func keymapDir() string {
	d := filepath.Join(filepath.Dir(cacheRoot()), "keymaps")
	os.MkdirAll(d, 0o755)
	return d
}

func registerKeyboardAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/keyboard/list", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"files": listKeyboards(), "initial": kbInitial})
	})
	mux.HandleFunc("/api/keyboard/open", func(w http.ResponseWriter, r *http.Request) {
		p := filepath.Clean(r.URL.Query().Get("path"))
		if !strings.HasSuffix(strings.ToLower(p), ".keebie.json") {
			jsonErr(w, "That isn't a Keebie keyboard file (.keebie.json).")
			return
		}
		b, err := os.ReadFile(p)
		if err != nil {
			jsonErr(w, "Couldn't open "+filepath.Base(p)+": "+err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	})
	// keymaps live in Flashie's own folder, one per keyboard name
	mux.HandleFunc("/api/keyboard/keymap", func(w http.ResponseWriter, r *http.Request) {
		name := safeFileName(r.URL.Query().Get("name"))
		if name == "" {
			jsonErr(w, "no keyboard name")
			return
		}
		p := filepath.Join(keymapDir(), name+".keymap.json")
		if r.Method == http.MethodPost {
			if o := r.Header.Get("Origin"); o != "" && !strings.HasPrefix(o, "http://"+r.Host) {
				http.Error(w, "forbidden", 403)
				return
			}
			b := make([]byte, 0, 4096)
			buf := make([]byte, 32*1024)
			for len(b) < 4<<20 {
				n, err := r.Body.Read(buf)
				b = append(b, buf[:n]...)
				if err != nil {
					break
				}
			}
			if err := os.WriteFile(p, b, 0o644); err != nil {
				jsonErr(w, err.Error())
				return
			}
			writeJSON(w, map[string]any{"ok": true})
			return
		}
		b, err := os.ReadFile(p)
		if err != nil {
			writeJSON(w, nil)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	})
	mux.HandleFunc("/api/keyboard/build", post(handleKeyboardBuild))
	mux.HandleFunc("/api/keyboard/firmware", func(w http.ResponseWriter, r *http.Request) {
		kbLast.Lock()
		data, name := kbLast.data, kbLast.name
		kbLast.Unlock()
		if data == nil {
			http.Error(w, "nothing built yet", 404)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
		w.Write(data)
	})
}
