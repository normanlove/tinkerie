// Flashie — firmware library: a catalogue of popular firmware, resolved on
// demand from each project's official release channel, downloaded into a
// local cache and verified against published checksums.
// Made by nrmluvdev.

package flashie

import (
	"archive/zip"
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// Endpoints (variables so tests and demo mode can point them at a mock).
var (
	ghAPI   = "https://api.github.com"
	mpyBase = "https://micropython.org"
	cpyBase = "https://downloads.circuitpython.org"
	tasBase = "http://ota.tasmota.com"
)

const userAgent = "Flashie/0.4 (firmware flasher by nrmluvdev)"

// ---------------------------------------------------------------- model ---

// LibPart is one file to write (or a generated/erased region).
type LibPart struct {
	Kind     string `json:"kind"` // download | ptable | fill
	Name     string `json:"name"`
	URL      string `json:"url,omitempty"`
	Offset   uint32 `json:"offset"`
	Size     int64  `json:"size,omitempty"`
	SHA256   string `json:"sha256,omitempty"`  // published by the source
	GitSHA1  string `json:"gitSha1,omitempty"` // git blob id (repo files)
	Entry    string `json:"-"`                 // regex: file inside a zip
	Table    string `json:"table,omitempty"`   // ptable: min_spiffs | huge_app
	FillLen  int    `json:"fillLen,omitempty"`
	Checksum string `json:"checksum"` // human summary of how it's verified
	Cached   bool   `json:"cached"`
}

type Source struct {
	Kind        string    // gh-asset | gh-zip | gh-tree | url | mpy | cpy | info
	Repo        string    // owner/name
	Asset       string    // regex for the release asset
	Entry       string    // regex for the file inside a zip asset
	Dir         string    // gh-tree: folder holding per-board subfolders
	Want, Not   []string  // gh-tree: tokens a folder name must / mustn't contain
	URL         string    // url: direct download
	Board       string    // mpy / cpy board name
	Offset      uint32    // where the main file goes
	Format      string    // bin | uf2 | hex
	Extra       []LibPart // fixed extra parts (e.g. a bootloader)
	AppOnly     bool      // app image only: add partition table + clear otadata
	VersionRepo string    // url: where to read the version from
}

type LibTarget struct {
	Boards []string // board IDs this build is for (takes priority)
	Chips  []string // or any board with one of these chips
	Src    Source
	Note   string
}

type LibItem struct {
	ID       string      `json:"id"`
	Name     string      `json:"name"`
	Category string      `json:"category"`
	Summary  string      `json:"summary"`
	About    string      `json:"about"`
	Homepage string      `json:"homepage"`
	License  string      `json:"license,omitempty"`
	Tags     []string    `json:"tags,omitempty"`
	Build    string      `json:"build,omitempty"` // set for build-it-yourself entries
	Warn     string      `json:"warn,omitempty"`
	Targets  []LibTarget `json:"-"`
}

func chipsT(src Source, chips ...string) LibTarget { return LibTarget{Chips: chips, Src: src} }
func boardsT(src Source, boards ...string) LibTarget {
	return LibTarget{Boards: boards, Src: src}
}

var cydR = []string{"cyd-2432s028r", "cyd-2432s028r-v2"}

// ------------------------------------------------------------- catalogue ---

var library = []LibItem{
	{ID: "micropython", Name: "MicroPython", Category: "Programming", Homepage: "https://micropython.org/download/", License: "MIT",
		Summary: "Python 3 for microcontrollers, with an interactive prompt over USB.",
		About:   "The official generic builds from micropython.org. Connect with Thonny or mpremote after flashing.",
		Tags:    []string{"Python", "Official builds"},
		Targets: []LibTarget{
			boardsT(Source{Kind: "mpy", Board: "RPI_PICO_W", Format: "uf2"}, "pico-w"),
			boardsT(Source{Kind: "mpy", Board: "RPI_PICO2_W", Format: "uf2"}, "pico2-w"),
			boardsT(Source{Kind: "mpy", Board: "RPI_PICO2", Format: "uf2"}, "pico2", "rp2350-generic"),
			boardsT(Source{Kind: "mpy", Board: "RPI_PICO", Format: "uf2"}, "pico", "rp2040-zero", "xiao-rp2040", "rp2040-generic"),
			chipsT(Source{Kind: "mpy", Board: "ESP32_GENERIC", Format: "bin", Offset: 0x1000}, "ESP32"),
			chipsT(Source{Kind: "mpy", Board: "ESP32_GENERIC_S2", Format: "bin", Offset: 0x1000}, "ESP32-S2"),
			chipsT(Source{Kind: "mpy", Board: "ESP32_GENERIC_S3", Format: "bin", Offset: 0x0}, "ESP32-S3"),
			chipsT(Source{Kind: "mpy", Board: "ESP32_GENERIC_C3", Format: "bin", Offset: 0x0}, "ESP32-C3"),
			chipsT(Source{Kind: "mpy", Board: "ESP32_GENERIC_C6", Format: "bin", Offset: 0x0}, "ESP32-C6"),
			chipsT(Source{Kind: "mpy", Board: "ESP8266_GENERIC", Format: "bin", Offset: 0x0}, "ESP8266"),
		}},
	{ID: "circuitpython", Name: "CircuitPython", Category: "Programming", Homepage: "https://circuitpython.org/downloads", License: "MIT",
		Summary: "Adafruit's beginner-friendly Python: your board shows up as a USB drive for code.",
		About:   "Official per-board builds from downloads.circuitpython.org (US English). The board must be listed here — CircuitPython builds are board-specific.",
		Tags:    []string{"Python", "Board-specific"},
		Targets: []LibTarget{
			boardsT(Source{Kind: "cpy", Board: "raspberry_pi_pico", Format: "uf2"}, "pico", "rp2040-generic"),
			boardsT(Source{Kind: "cpy", Board: "raspberry_pi_pico_w", Format: "uf2"}, "pico-w"),
			boardsT(Source{Kind: "cpy", Board: "raspberry_pi_pico2", Format: "uf2"}, "pico2", "rp2350-generic"),
			boardsT(Source{Kind: "cpy", Board: "raspberry_pi_pico2_w", Format: "uf2"}, "pico2-w"),
			boardsT(Source{Kind: "cpy", Board: "waveshare_rp2040_zero", Format: "uf2"}, "rp2040-zero"),
			boardsT(Source{Kind: "cpy", Board: "seeed_xiao_rp2040", Format: "uf2"}, "xiao-rp2040"),
			boardsT(Source{Kind: "cpy", Board: "espressif_esp32s3_devkitc_1_n8r8", Format: "bin", Offset: 0}, "esp32-s3-devkitc"),
			boardsT(Source{Kind: "cpy", Board: "lolin_s2_mini", Format: "bin", Offset: 0}, "esp32-s2-mini"),
			boardsT(Source{Kind: "cpy", Board: "espressif_esp32c3_devkitm_1_n4", Format: "bin", Offset: 0}, "esp32-c3-devkitm"),
		}},
	{ID: "esphome", Name: "ESPHome (ready to adopt)", Category: "Smart home", Homepage: "https://esphome.io", License: "GPL-3.0 / MIT",
		Summary: "Starter firmware that joins Wi-Fi and lets Home Assistant adopt and configure the board.",
		About:   "The same “ESPHome Web” starter firmware web.esphome.io installs. After flashing, set up Wi-Fi from Home Assistant or the ESPHome dashboard.",
		Tags:    []string{"Home Assistant", "Official builds"},
		Targets: []LibTarget{
			chipsT(Source{Kind: "gh-asset", Repo: "esphome/firmware", Asset: `(?i)^esphome-web-esp32(-[0-9][^-]*)?\.factory\.bin$`, Offset: 0}, "ESP32"),
			chipsT(Source{Kind: "gh-asset", Repo: "esphome/firmware", Asset: `(?i)^esphome-web-esp32s2(-[0-9][^-]*)?\.factory\.bin$`, Offset: 0}, "ESP32-S2"),
			chipsT(Source{Kind: "gh-asset", Repo: "esphome/firmware", Asset: `(?i)^esphome-web-esp32s3(-[0-9][^-]*)?\.factory\.bin$`, Offset: 0}, "ESP32-S3"),
			chipsT(Source{Kind: "gh-asset", Repo: "esphome/firmware", Asset: `(?i)^esphome-web-esp32c3(-[0-9][^-]*)?\.factory\.bin$`, Offset: 0}, "ESP32-C3"),
			chipsT(Source{Kind: "gh-asset", Repo: "esphome/firmware", Asset: `(?i)^esphome-web-esp32c6(-[0-9][^-]*)?\.factory\.bin$`, Offset: 0}, "ESP32-C6"),
			chipsT(Source{Kind: "gh-asset", Repo: "esphome/firmware", Asset: `(?i)^esphome-web-esp8266(-[0-9][^-]*)?\.factory\.bin$`, Offset: 0}, "ESP8266"),
		}},
	{ID: "wled", Name: "WLED", Category: "Lighting", Homepage: "https://kno.wled.ge", License: "EUPL-1.2",
		Summary: "Control WS2812B / NeoPixel LED strips over Wi-Fi with 100+ effects and a phone app.",
		About:   "Official release binaries from GitHub. On classic ESP32, Flashie also writes WLED's v4 bootloader at 0x0 first, as the WLED docs instruct.",
		Tags:    []string{"LEDs", "Official builds"},
		Targets: []LibTarget{
			chipsT(Source{Kind: "gh-asset", Repo: "wled/WLED", Asset: `^WLED_[0-9][^_]*_ESP32\.bin$`, Offset: 0x10000,
				Extra: []LibPart{{Kind: "download", Name: "esp32_bootloader_v4.bin", URL: "https://github.com/Aircoookie/WLED/releases/download/v0.13.1/esp32_bootloader_v4.bin", Offset: 0}}}, "ESP32"),
			chipsT(Source{Kind: "gh-asset", Repo: "wled/WLED", Asset: `^WLED_[0-9][^_]*_ESP8266\.bin$`, Offset: 0}, "ESP8266"),
		}},
	{ID: "tasmota", Name: "Tasmota", Category: "Smart home", Homepage: "https://tasmota.github.io/docs/", License: "GPL-3.0",
		Summary: "Local-control firmware for smart plugs, relays and sensors — MQTT, web UI, rules.",
		About:   "The standard release build from ota.tasmota.com (the “factory” image on ESP32, which includes the bootloader).",
		Tags:    []string{"MQTT", "Official builds"},
		Targets: []LibTarget{
			chipsT(Source{Kind: "url", URL: "/tasmota32/release/tasmota32.factory.bin", Offset: 0, VersionRepo: "arendst/Tasmota"}, "ESP32"),
			chipsT(Source{Kind: "url", URL: "/tasmota32/release/tasmota32s2.factory.bin", Offset: 0, VersionRepo: "arendst/Tasmota"}, "ESP32-S2"),
			chipsT(Source{Kind: "url", URL: "/tasmota32/release/tasmota32s3.factory.bin", Offset: 0, VersionRepo: "arendst/Tasmota"}, "ESP32-S3"),
			chipsT(Source{Kind: "url", URL: "/tasmota32/release/tasmota32c3.factory.bin", Offset: 0, VersionRepo: "arendst/Tasmota"}, "ESP32-C3"),
			chipsT(Source{Kind: "url", URL: "/tasmota/release/tasmota.bin", Offset: 0, VersionRepo: "arendst/Tasmota"}, "ESP8266"),
		}},
	{ID: "meshtastic", Name: "Meshtastic", Category: "Radio", Homepage: "https://meshtastic.org", License: "GPL-3.0",
		Summary: "Off-grid LoRa mesh messaging — pair with the phone app and text without phone signal.",
		About:   "Official release from GitHub. Needs a board with a LoRa radio. Flashie picks your board's “factory” image out of the release archive.",
		Tags:    []string{"LoRa", "Official builds"},
		Warn:    "Flashing Meshtastic resets the node's settings — note your channel URL first.",
		Targets: []LibTarget{
			boardsT(Source{Kind: "gh-zip", Repo: "meshtastic/firmware", Asset: `^firmware-esp32s3-[0-9].*\.zip$`, Entry: `^firmware-t-deck-[0-9][^/]*\.factory\.bin$`, Offset: 0}, "t-deck"),
			boardsT(Source{Kind: "gh-zip", Repo: "meshtastic/firmware", Asset: `^firmware-esp32s3-[0-9].*\.zip$`, Entry: `^firmware-heltec-v3-[0-9][^/]*\.factory\.bin$`, Offset: 0}, "heltec-v3"),
			boardsT(Source{Kind: "gh-zip", Repo: "meshtastic/firmware", Asset: `^firmware-esp32-[0-9].*\.zip$`, Entry: `^firmware-tbeam-[0-9][^/]*\.factory\.bin$`, Offset: 0}, "ttgo-t-beam"),
			boardsT(Source{Kind: "gh-zip", Repo: "meshtastic/firmware", Asset: `^firmware-rp2040-[0-9].*\.zip$`, Entry: `^firmware-rpipico-[0-9][^/]*\.uf2$`, Format: "uf2"}, "pico"),
		}},
	{ID: "nerdminer", Name: "NerdMiner v2", Category: "Fun", Homepage: "https://github.com/BitMaker-hub/NerdMiner_v2", License: "MIT",
		Summary: "A tiny solo Bitcoin “lottery” miner with a stats screen — popular on CYDs and T-Display S3.",
		About:   "Official per-board binaries from the project's bin folder. Each file's name gives its flash address, and Flashie checks each one against GitHub's git hash.",
		Tags:    []string{"CYD build", "Bitcoin"},
		Targets: []LibTarget{
			boardsT(Source{Kind: "gh-tree", Repo: "BitMaker-hub/NerdMiner_v2", Dir: "bin", Want: []string{"2432s028"}, Not: []string{"7789", "st7789", "2usb"}}, cydR...),
			boardsT(Source{Kind: "gh-tree", Repo: "BitMaker-hub/NerdMiner_v2", Dir: "bin", Want: []string{"2432s028"}}, "cyd-2432s028r-v3"),
			boardsT(Source{Kind: "gh-tree", Repo: "BitMaker-hub/NerdMiner_v2", Dir: "bin", Want: []string{"display", "s3"}, Not: []string{"amoled", "pro", "touch"}}, "t-display-s3"),
			boardsT(Source{Kind: "gh-tree", Repo: "BitMaker-hub/NerdMiner_v2", Dir: "bin", Want: []string{"c3", "super", "mini"}}, "c3-supermini"),
		}},
	{ID: "grbl", Name: "Grbl 1.1", Category: "CNC & 3D printing", Homepage: "https://github.com/gnea/grbl", License: "GPL-3.0",
		Summary: "The classic G-code CNC / laser controller for Arduino Uno and Nano (e.g. with a CNC shield).",
		About:   "The official pre-compiled .hex from the gnea/grbl release. It resets your Grbl $ settings to defaults.",
		Tags:    []string{"CNC", "Laser", "Official builds"},
		Targets: []LibTarget{
			boardsT(Source{Kind: "gh-asset", Repo: "gnea/grbl", Asset: `(?i)^grbl_v.*\.hex$`, Format: "hex"}, "uno", "uno-ch340", "nano", "nano-old", "pro-mini-5v"),
		}},
	{ID: "grbl-mega", Name: "Grbl-Mega", Category: "CNC & 3D printing", Homepage: "https://github.com/gnea/grbl-Mega", License: "GPL-3.0",
		Summary: "Grbl for the Arduino Mega 2560 — more I/O for larger machines and RAMPS-style boards.",
		About:   "The official pre-compiled .hex from the gnea/grbl-Mega release, if that release includes one.",
		Tags:    []string{"CNC", "Official builds"},
		Targets: []LibTarget{
			boardsT(Source{Kind: "gh-asset", Repo: "gnea/grbl-Mega", Asset: `(?i)\.hex$`, Format: "hex"}, "mega2560", "mega2560-ch340"),
		}},
	{ID: "klipper", Name: "Klipper (MCU firmware)", Category: "CNC & 3D printing", Homepage: "https://www.klipper3d.org/Installation.html", License: "GPL-3.0",
		Summary: "3D-printer firmware where a Raspberry Pi does the maths and the board just drives motors.",
		About:   "Klipper doesn't publish ready-made binaries — the MCU firmware is compiled for your exact printer board with `make menuconfig` on your Klipper host.",
		Build:   "On your Klipper host: cd ~/klipper && make menuconfig (pick your MCU) && make. Copy out/klipper.uf2 (RP2040), out/klipper.elf.hex (AVR) or out/klipper.bin (STM32) to this PC, then flash it with “Use my own file”.",
		Tags:    []string{"3D printing", "Build it yourself"},
		Targets: []LibTarget{
			{Chips: []string{"RP2040", "RP2350", "ATmega2560", "ATmega1280", "ATmega328P", "STM32F103", "STM32F401", "STM32F411", "STM32"}, Src: Source{Kind: "info"}},
		}},
	{ID: "marlin", Name: "Marlin", Category: "CNC & 3D printing", Homepage: "https://marlinfw.org/docs/basics/install.html", License: "GPL-3.0",
		Summary: "The most widely used 3D-printer firmware — runs on Mega 2560 + RAMPS and many STM32 boards.",
		About:   "Marlin has no universal ready-made binaries: you set your printer's options in Configuration.h and build it (PlatformIO / VS Code).",
		Build:   "Build Marlin with PlatformIO for your board. AVR boards (Mega 2560 / RAMPS) produce firmware.hex; STM32 boards produce firmware.bin. Flash the file here with “Use my own file”.",
		Tags:    []string{"3D printing", "Build it yourself"},
		Targets: []LibTarget{
			{Chips: []string{"ATmega2560", "ATmega1280", "STM32F103", "STM32F401", "STM32F411", "STM32"}, Src: Source{Kind: "info"}},
		}},
}

func libItem(id string) *LibItem {
	for i := range library {
		if library[i].ID == id {
			return &library[i]
		}
	}
	return nil
}

// pickTarget finds the build for a board (by ID first, then by chip).
func (it *LibItem) pickTarget(boardID, chip string) *LibTarget {
	for i := range it.Targets {
		for _, b := range it.Targets[i].Boards {
			if b == boardID {
				return &it.Targets[i]
			}
		}
	}
	for i := range it.Targets {
		for _, c := range it.Targets[i].Chips {
			if strings.EqualFold(c, chip) {
				return &it.Targets[i]
			}
		}
	}
	return nil
}

// supportList is a short, human list of what an item runs on.
func (it *LibItem) supportList() []string {
	seen := map[string]bool{}
	var out []string
	add := func(s string) {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	for _, t := range it.Targets {
		for _, c := range t.Chips {
			add(c)
		}
		for _, b := range t.Boards {
			if bd := boardByID(b); bd != nil {
				add(bd.Name)
			}
		}
	}
	return out
}

// LibListEntry is the catalogue as the UI sees it for one board.
type LibListEntry struct {
	LibItem
	Supports  []string `json:"supports"`
	Available bool     `json:"available"`
	TargetFor string   `json:"targetFor,omitempty"`
}

func libraryFor(boardID, chip string) []LibListEntry {
	var out []LibListEntry
	for _, it := range library {
		e := LibListEntry{LibItem: it, Supports: it.supportList()}
		if t := it.pickTarget(boardID, chip); t != nil {
			e.Available = true
			if len(t.Boards) > 0 {
				e.TargetFor = "board"
			} else {
				e.TargetFor = "chip"
			}
		}
		out = append(out, e)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Available != out[j].Available {
			return out[i].Available
		}
		// board-specific builds (e.g. CYD) first
		return out[i].TargetFor == "board" && out[j].TargetFor != "board"
	})
	return out
}

// --------------------------------------------------------------- network ---

var httpClient = &http.Client{Timeout: 5 * time.Minute}

func httpGet(url string, accept string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("couldn't reach %s — check your internet connection (%v)", hostOf(url), err)
	}
	if resp.StatusCode != 200 {
		resp.Body.Close()
		if resp.StatusCode == 403 && strings.Contains(url, "api.github.com") {
			return nil, errors.New("GitHub's hourly limit for anonymous requests has been reached — try again in a while")
		}
		return nil, fmt.Errorf("%s answered %s for %s", hostOf(url), resp.Status, url)
	}
	return resp, nil
}

func hostOf(u string) string {
	u = strings.TrimPrefix(strings.TrimPrefix(u, "https://"), "http://")
	if i := strings.IndexByte(u, '/'); i >= 0 {
		return u[:i]
	}
	return u
}

func getJSON(url string, v any) error {
	resp, err := httpGet(url, "application/vnd.github+json")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(v)
}

type ghAsset struct {
	Name   string `json:"name"`
	URL    string `json:"browser_download_url"`
	Size   int64  `json:"size"`
	Digest string `json:"digest"`
}

type ghRelease struct {
	Tag       string    `json:"tag_name"`
	Name      string    `json:"name"`
	HTMLURL   string    `json:"html_url"`
	Published string    `json:"published_at"`
	Body      string    `json:"body"`
	Assets    []ghAsset `json:"assets"`
}

type ghContent struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Type        string `json:"type"`
	SHA         string `json:"sha"`
	Size        int64  `json:"size"`
	DownloadURL string `json:"download_url"`
}

var (
	metaMu    sync.Mutex
	metaCache = map[string]struct {
		at time.Time
		v  []byte
	}{}
)

// cachedJSON memoises API answers for 15 minutes (GitHub allows 60 anonymous calls an hour).
func cachedJSON(url string, v any) error {
	metaMu.Lock()
	c, ok := metaCache[url]
	metaMu.Unlock()
	if ok && time.Since(c.at) < 15*time.Minute {
		return json.Unmarshal(c.v, v)
	}
	var raw json.RawMessage
	if err := getJSON(url, &raw); err != nil {
		return err
	}
	metaMu.Lock()
	metaCache[url] = struct {
		at time.Time
		v  []byte
	}{time.Now(), raw}
	metaMu.Unlock()
	return json.Unmarshal(raw, v)
}

func latestRelease(repo string) (*ghRelease, error) {
	var r ghRelease
	if err := cachedJSON(ghAPI+"/repos/"+repo+"/releases/latest", &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// --------------------------------------------------------------- resolve ---

// Resolved is a concrete, downloadable build for one board.
type Resolved struct {
	ItemID     string    `json:"itemId"`
	Name       string    `json:"name"`
	Version    string    `json:"version"`
	Date       string    `json:"date,omitempty"`
	ReleaseURL string    `json:"releaseUrl,omitempty"`
	Notes      string    `json:"notes,omitempty"`
	Family     string    `json:"family"`
	Format     string    `json:"format"` // esp | uf2 | hex | bin
	Parts      []LibPart `json:"parts"`
	Verified   string    `json:"verified"` // summary of checksum coverage
	Note       string    `json:"note,omitempty"`
}

func shortNotes(s string) string {
	s = strings.TrimSpace(regexp.MustCompile(`(?m)^#+\s*`).ReplaceAllString(s, ""))
	if len(s) > 600 {
		s = s[:600] + "…"
	}
	return s
}

func digestHex(d string) string {
	if strings.HasPrefix(d, "sha256:") {
		return strings.ToLower(strings.TrimPrefix(d, "sha256:"))
	}
	return ""
}

func checksumText(p LibPart) string {
	switch {
	case p.Kind == "ptable":
		return "Generated by Flashie"
	case p.Kind == "fill":
		return "Erased region"
	case p.SHA256 != "":
		return "SHA-256 published by GitHub"
	case p.GitSHA1 != "":
		return "Git hash from GitHub"
	}
	return "No checksum published"
}

func resolveItem(id, boardID, chip string) (*Resolved, error) {
	it := libItem(id)
	if it == nil {
		return nil, errors.New("unknown firmware")
	}
	t := it.pickTarget(boardID, chip)
	if t == nil {
		return nil, fmt.Errorf("%s has no build for this board", it.Name)
	}
	src := t.Src
	res := &Resolved{ItemID: id, Name: it.Name, Note: t.Note}
	switch {
	case src.Format == "uf2":
		res.Family, res.Format = famRP2, "uf2"
	case src.Format == "hex":
		res.Family, res.Format = famAVR, "hex"
	default:
		res.Family, res.Format = famESP, "esp"
	}
	main := LibPart{Kind: "download", Offset: src.Offset}

	switch src.Kind {
	case "info":
		return nil, errors.New(it.Name + " has to be built for your machine — see the instructions")

	case "gh-asset", "gh-zip":
		rel, err := latestRelease(src.Repo)
		if err != nil {
			return nil, err
		}
		res.Version, res.Date, res.ReleaseURL, res.Notes = rel.Tag, dateOnly(rel.Published), rel.HTMLURL, shortNotes(rel.Body)
		re := regexp.MustCompile(src.Asset)
		var found *ghAsset
		for i := range rel.Assets {
			if re.MatchString(rel.Assets[i].Name) {
				found = &rel.Assets[i]
				break
			}
		}
		if found == nil {
			return nil, fmt.Errorf("the latest %s release (%s) doesn't include a build for this board — see %s", it.Name, rel.Tag, rel.HTMLURL)
		}
		main.Name, main.URL, main.Size, main.SHA256 = found.Name, found.URL, found.Size, digestHex(found.Digest)
		if src.Kind == "gh-zip" {
			main.Entry = src.Entry
			main.Name = found.Name + " → " + strings.NewReplacer(`[0-9][^/]*`, "…", `\.`, ".", "^", "", "$", "").Replace(src.Entry)
		}

	case "gh-tree":
		rel, err := latestRelease(src.Repo)
		if err == nil {
			res.Version, res.Date, res.ReleaseURL, res.Notes = rel.Tag, dateOnly(rel.Published), rel.HTMLURL, shortNotes(rel.Body)
		}
		var dirs []ghContent
		if err := cachedJSON(ghAPI+"/repos/"+src.Repo+"/contents/"+src.Dir, &dirs); err != nil {
			return nil, err
		}
		var pick *ghContent
		for i := range dirs {
			if dirs[i].Type == "dir" && tokenMatch(dirs[i].Name, src.Want, src.Not) {
				pick = &dirs[i]
				break
			}
		}
		if pick == nil {
			return nil, fmt.Errorf("couldn't find a %s build folder for this board in %s/%s", it.Name, src.Repo, src.Dir)
		}
		var files []ghContent
		if err := cachedJSON(ghAPI+"/repos/"+src.Repo+"/contents/"+pick.Path, &files); err != nil {
			return nil, err
		}
		offRe := regexp.MustCompile(`(?i)^0x([0-9a-f]+)_.*\.bin$`)
		for _, f := range files {
			m := offRe.FindStringSubmatch(f.Name)
			if f.Type != "file" || m == nil {
				continue
			}
			var off uint32
			fmt.Sscanf(m[1], "%x", &off)
			res.Parts = append(res.Parts, LibPart{Kind: "download", Name: pick.Name + "/" + f.Name, URL: f.DownloadURL, Offset: off, Size: f.Size, GitSHA1: f.SHA})
		}
		if len(res.Parts) == 0 {
			return nil, fmt.Errorf("the %s folder has no flashable files", pick.Name)
		}
		if res.Version == "" {
			res.Version = "latest (" + pick.Name + ")"
		}
		sort.Slice(res.Parts, func(i, j int) bool { return res.Parts[i].Offset < res.Parts[j].Offset })
		finishResolved(res)
		return res, nil

	case "mpy":
		url, ver, date, err := micropythonLatest(src.Board, src.Format)
		if err != nil {
			return nil, err
		}
		main.Name, main.URL = url[strings.LastIndex(url, "/")+1:], url
		res.Version, res.Date = ver, date
		res.ReleaseURL = mpyBase + "/download/" + src.Board + "/"

	case "cpy":
		rel, err := latestRelease("adafruit/circuitpython")
		if err != nil {
			return nil, err
		}
		ver := strings.TrimPrefix(rel.Tag, "v")
		ext := map[string]string{"uf2": "uf2"}[src.Format]
		if ext == "" {
			ext = "bin"
		}
		name := fmt.Sprintf("adafruit-circuitpython-%s-en_US-%s.%s", src.Board, ver, ext)
		main.Name, main.URL = name, fmt.Sprintf("%s/bin/%s/en_US/%s", cpyBase, src.Board, name)
		res.Version, res.Date, res.ReleaseURL = ver, dateOnly(rel.Published), "https://circuitpython.org/board/"+src.Board+"/"

	case "url":
		u := src.URL
		if strings.HasPrefix(u, "/") {
			u = tasBase + u
		}
		main.Name, main.URL = u[strings.LastIndex(u, "/")+1:], u
		if src.VersionRepo != "" {
			if rel, err := latestRelease(src.VersionRepo); err == nil {
				res.Version, res.Date, res.ReleaseURL, res.Notes = rel.Tag, dateOnly(rel.Published), rel.HTMLURL, shortNotes(rel.Body)
			}
		}
		if res.Version == "" {
			res.Version = "latest release"
		}
	}

	res.Parts = append(res.Parts, src.Extra...)
	if src.AppOnly {
		res.Parts = append(res.Parts,
			LibPart{Kind: "ptable", Name: "Partition table (Minimal SPIFFS)", Offset: 0x8000, Table: "min_spiffs"},
			LibPart{Kind: "fill", Name: "Clear OTA data (boot the new app)", Offset: 0xE000, FillLen: 0x2000})
	}
	res.Parts = append(res.Parts, main)
	sort.SliceStable(res.Parts, func(i, j int) bool { return res.Parts[i].Offset < res.Parts[j].Offset })
	finishResolved(res)
	return res, nil
}

func finishResolved(res *Resolved) {
	n, have := 0, 0
	for i := range res.Parts {
		p := &res.Parts[i]
		p.Checksum = checksumText(*p)
		if p.Kind == "download" {
			n++
			if p.SHA256 != "" || p.GitSHA1 != "" {
				have++
			}
			if _, ok := cacheLookup(cacheKey(p.URL, res.Version), p.SHA256); ok {
				p.Cached = true
			}
		}
	}
	switch {
	case n == 0:
		res.Verified = "Nothing to download"
	case have == n:
		res.Verified = "Every file is checked against a checksum published by the source"
	case have > 0:
		res.Verified = fmt.Sprintf("%d of %d files have published checksums; the others are checked for completeness and fingerprinted in the cache", have, n)
	default:
		res.Verified = "The source doesn't publish checksums — Flashie checks the download is complete and fingerprints it (SHA-256) in its cache"
	}
}

func dateOnly(s string) string {
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}

func tokenMatch(name string, want, not []string) bool {
	n := strings.ToLower(regexp.MustCompile(`[^a-zA-Z0-9]+`).ReplaceAllString(name, ""))
	for _, w := range want {
		if !strings.Contains(n, strings.ToLower(w)) {
			return false
		}
	}
	for _, x := range not {
		if strings.Contains(n, strings.ToLower(x)) {
			return false
		}
	}
	return true
}

// micropythonLatest reads the board's download page and returns the
// newest stable (non-preview) firmware link.
func micropythonLatest(board, ext string) (url, ver, date string, err error) {
	resp, err := httpGet(mpyBase+"/download/"+board+"/", "text/html")
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()
	page, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	re := regexp.MustCompile(`/resources/firmware/` + regexp.QuoteMeta(board) + `-(\d{8})-v(\d+\.\d+(?:\.\d+)?)\.` + ext + `\b`)
	m := re.FindSubmatch(page)
	if m == nil {
		return "", "", "", fmt.Errorf("couldn't find a stable %s download on micropython.org", board)
	}
	d := string(m[1])
	return mpyBase + string(m[0]), "v" + string(m[2]), d[:4] + "-" + d[4:6] + "-" + d[6:], nil
}

// ---------------------------------------------------------------- cache ---

func cacheDir() string {
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	d := filepath.Join(base, "nrmluvdev", "Flashie", "firmware")
	os.MkdirAll(d, 0o755)
	return d
}

var cacheRoot = cacheDir

type cacheEntry struct {
	File   string    `json:"file"`
	SHA256 string    `json:"sha256"`
	Size   int64     `json:"size"`
	At     time.Time `json:"at"`
	URL    string    `json:"url"`
}

var cacheMu sync.Mutex

func cacheKey(url, version string) string {
	h := sha256.Sum256([]byte(url + "|" + version))
	return hex.EncodeToString(h[:12])
}

func loadIndex() map[string]cacheEntry {
	idx := map[string]cacheEntry{}
	b, err := os.ReadFile(filepath.Join(cacheRoot(), "index.json"))
	if err == nil {
		json.Unmarshal(b, &idx)
	}
	return idx
}

func saveIndex(idx map[string]cacheEntry) {
	b, _ := json.MarshalIndent(idx, "", " ")
	os.WriteFile(filepath.Join(cacheRoot(), "index.json"), b, 0o644)
}

// cacheLookup returns cached bytes if present, intact (fingerprint
// matches) and, when the source publishes one, matching its checksum.
func cacheLookup(key, wantSHA string) ([]byte, bool) {
	cacheMu.Lock()
	e, ok := loadIndex()[key]
	cacheMu.Unlock()
	if !ok {
		return nil, false
	}
	b, err := os.ReadFile(filepath.Join(cacheRoot(), e.File))
	if err != nil {
		return nil, false
	}
	sum := sha256.Sum256(b)
	got := hex.EncodeToString(sum[:])
	if got != e.SHA256 || (wantSHA != "" && got != wantSHA) {
		return nil, false
	}
	return b, true
}

func cacheStore(key, url string, data []byte) {
	sum := sha256.Sum256(data)
	e := cacheEntry{File: key + ".bin", SHA256: hex.EncodeToString(sum[:]), Size: int64(len(data)), At: time.Now(), URL: url}
	if os.WriteFile(filepath.Join(cacheRoot(), e.File), data, 0o644) != nil {
		return
	}
	cacheMu.Lock()
	idx := loadIndex()
	idx[key] = e
	saveIndex(idx)
	cacheMu.Unlock()
}

func cacheInfo() (files int, bytes int64) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	for _, e := range loadIndex() {
		files++
		bytes += e.Size
	}
	return
}

func cacheClear() error {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	ents, _ := os.ReadDir(cacheRoot())
	for _, e := range ents {
		os.Remove(filepath.Join(cacheRoot(), e.Name()))
	}
	return nil
}

// -------------------------------------------------------------- download ---

func gitBlobSHA1(b []byte) string {
	h := sha1.New()
	fmt.Fprintf(h, "blob %d\x00", len(b))
	h.Write(b)
	return hex.EncodeToString(h.Sum(nil))
}

// fetchPart downloads (or takes from cache) one part and verifies it.
func fetchPart(p LibPart, version string, r *reporter) ([]byte, error) {
	key := cacheKey(p.URL, version)
	data, hit := cacheLookup(key, p.SHA256)
	if hit {
		r.logf("Using cached %s (%s)", p.Name, humanBytes(len(data)))
	} else {
		r.status(fmt.Sprintf("Downloading %s…", shortName(p.Name)))
		u := p.URL
		if urlRewrite != nil {
			u = urlRewrite(u)
		}
		resp, err := httpGet(u, "")
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		total := resp.ContentLength
		if total <= 0 {
			total = p.Size
		}
		var buf bytes.Buffer
		chunk := make([]byte, 64*1024)
		for {
			if r.cancelled() {
				return nil, errCancelled
			}
			n, err := resp.Body.Read(chunk)
			buf.Write(chunk[:n])
			if total > 0 {
				r.progress(buf.Len(), int(total))
			}
			if buf.Len() > 64<<20 {
				return nil, errors.New("download is unexpectedly large — stopped")
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("download of %s was interrupted: %v", p.Name, err)
			}
		}
		data = buf.Bytes()
		if total > 0 && int64(len(data)) != total {
			return nil, fmt.Errorf("download of %s is incomplete (%s of %s)", p.Name, humanBytes(len(data)), humanBytes(int(total)))
		}
		if p.SHA256 != "" {
			sum := sha256.Sum256(data)
			if hex.EncodeToString(sum[:]) != p.SHA256 {
				return nil, fmt.Errorf("checksum mismatch for %s — the download doesn't match what GitHub published, so Flashie won't flash it", p.Name)
			}
			r.logf("SHA-256 verified for %s", p.Name)
		}
		if p.GitSHA1 != "" {
			if gitBlobSHA1(data) != p.GitSHA1 {
				return nil, fmt.Errorf("checksum mismatch for %s — the download doesn't match the repository, so Flashie won't flash it", p.Name)
			}
			r.logf("Git hash verified for %s", p.Name)
		}
		cacheStore(key, p.URL, data)
	}
	if p.Entry != "" {
		return extractZip(data, p.Entry, r)
	}
	return data, nil
}

func shortName(s string) string {
	if i := strings.LastIndex(s, "/"); i >= 0 && i < len(s)-1 {
		return s[i+1:]
	}
	return s
}

func extractZip(data []byte, entry string, r *reporter) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("the release archive is damaged: %v", err)
	}
	re := regexp.MustCompile(entry)
	var pick *zip.File
	for _, f := range zr.File {
		base := f.Name[strings.LastIndex(f.Name, "/")+1:]
		if re.MatchString(base) {
			pick = f
			break
		}
	}
	if pick == nil {
		var names []string
		for _, f := range zr.File {
			names = append(names, f.Name)
		}
		r.logf("Archive contents: %s", strings.Join(names, ", "))
		return nil, errors.New("this release archive doesn't contain a build for your board (see Technical details for its contents)")
	}
	rc, err := pick.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	b, err := io.ReadAll(io.LimitReader(rc, 64<<20))
	if err != nil {
		return nil, fmt.Errorf("couldn't unpack %s (CRC check failed?): %v", pick.Name, err)
	}
	r.logf("Unpacked %s (%s, CRC-32 checked)", pick.Name, humanBytes(len(b)))
	return b, nil
}

// ------------------------------------------------------- partition tables ---

type ptEntry struct {
	label        string
	typ, sub     byte
	offset, size uint32
}

var partitionTables = map[string][]ptEntry{
	// Arduino-ESP32 "min_spiffs.csv" (Minimal SPIFFS, 1.9 MB APP with OTA).
	"min_spiffs": {
		{"nvs", 1, 0x02, 0x9000, 0x5000}, {"otadata", 1, 0x00, 0xE000, 0x2000},
		{"app0", 0, 0x10, 0x10000, 0x1E0000}, {"app1", 0, 0x11, 0x1F0000, 0x1E0000},
		{"spiffs", 1, 0x82, 0x3D0000, 0x20000}, {"coredump", 1, 0x03, 0x3F0000, 0x10000},
	},
	// Arduino-ESP32 "huge_app.csv" (3 MB APP, no OTA).
	"huge_app": {
		{"nvs", 1, 0x02, 0x9000, 0x5000}, {"otadata", 1, 0x00, 0xE000, 0x2000},
		{"app0", 0, 0x10, 0x10000, 0x300000},
		{"spiffs", 1, 0x82, 0x310000, 0xE0000}, {"coredump", 1, 0x03, 0x3F0000, 0x10000},
	},
}

// buildPartitionTable makes the binary form ESP-IDF's gen_esp32part.py produces.
func buildPartitionTable(name string) ([]byte, error) {
	ents, ok := partitionTables[name]
	if !ok {
		return nil, fmt.Errorf("unknown partition table %q", name)
	}
	var body []byte
	for _, e := range ents {
		b := make([]byte, 32)
		b[0], b[1], b[2], b[3] = 0xAA, 0x50, e.typ, e.sub
		putLE32(b[4:], e.offset)
		putLE32(b[8:], e.size)
		copy(b[12:28], e.label)
		body = append(body, b...)
	}
	sum := md5Sum(body)
	md := make([]byte, 32)
	md[0], md[1] = 0xEB, 0xEB
	for i := 2; i < 16; i++ {
		md[i] = 0xFF
	}
	copy(md[16:], sum[:])
	body = append(body, md...)
	for len(body) < 0xC00 {
		body = append(body, 0xFF)
	}
	return body, nil
}

func appSlotSize(table string) uint32 {
	for _, e := range partitionTables[table] {
		if e.typ == 0 {
			return e.size
		}
	}
	return 0
}

func putLE32(b []byte, v uint32) {
	b[0], b[1], b[2], b[3] = byte(v), byte(v>>8), byte(v>>16), byte(v>>24)
}

func md5Sum(b []byte) [16]byte { return md5.Sum(b) }
