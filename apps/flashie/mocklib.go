// Flashie — an in-process stand-in for GitHub, micropython.org,
// downloads.circuitpython.org and ota.tasmota.com. Used by the tests and by
// demo mode (FLASHIE_DEMO=1) so the library can be tried offline.
// Made by nrmluvdev.

package flashie

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
)

// urlRewrite lets demo/tests redirect absolute URLs (e.g. github.com).
var urlRewrite func(string) string

type mockLib struct {
	srv    *httptest.Server
	mu     sync.Mutex
	files  map[string][]byte // path → content
	digest map[string]string // asset name → published digest (can be wrong on purpose)
	hits   map[string]int
}

func fakeImage(name string, size int) []byte {
	h := fnv.New64a()
	h.Write([]byte(name))
	r := rand.New(rand.NewSource(int64(h.Sum64())))
	b := make([]byte, size)
	r.Read(b)
	n := strings.ToLower(name)
	chip := uint16(0)
	switch {
	case strings.Contains(n, "8266") || strings.HasSuffix(n, "tasmota.bin"):
		b[0] = 0xE9
		b[12], b[13] = 0xAA, 0xBB // ESP8266 images have no chip field
		return b
	case strings.Contains(n, "s3") || strings.Contains(n, "t-deck") || strings.Contains(n, "heltec"):
		chip = 9
	case strings.Contains(n, "c3"):
		chip = 5
	case strings.Contains(n, "s2"):
		chip = 2
	case strings.Contains(n, "c6"):
		chip = 13
	}
	b[0] = 0xE9
	binary.LittleEndian.PutUint16(b[12:], chip)
	return b
}

func newMockLib() *mockLib {
	m := &mockLib{files: map[string][]byte{}, digest: map[string]string{}, hits: map[string]int{}}
	m.srv = httptest.NewServer(http.HandlerFunc(m.serve))
	return m
}

func (m *mockLib) put(path string, b []byte) {
	m.files[path] = b
	sum := sha256.Sum256(b)
	m.digest[path[strings.LastIndex(path, "/")+1:]] = "sha256:" + hex.EncodeToString(sum[:])
}

func (m *mockLib) releases() map[string][]string {
	return map[string][]string{
		"esphome/firmware": {"esphome-web-esp32-2026.10.1.factory.bin", "esphome-web-esp32s3-2026.10.1.factory.bin",
			"esphome-web-esp32c3-2026.10.1.factory.bin", "esphome-web-esp8266-2026.10.1.factory.bin", "esphome-web-esp32-2026.10.1.ota.bin"},
		"wled/WLED":                 {"WLED_0.15.3_ESP32.bin", "WLED_0.15.3_ESP8266.bin", "WLED_0.15.3_ESP32-S3_8MB.bin"},
		"arendst/Tasmota":           {},
		"adafruit/circuitpython":    {},
		"BitMaker-hub/NerdMiner_v2": {},
		"meshtastic/firmware":       {"firmware-esp32s3-2.7.15.567b8ea.zip", "firmware-esp32-2.7.15.567b8ea.zip", "firmware-rp2040-2.7.15.567b8ea.zip", "firmware-2.7.15.567b8ea.json"},
		"tinkerie-test/app-only":    {"app_only_demo_cyd_2432S028.bin"},
		"tinkerie-test/merged":      {"merged-demo-CYD-2432S028.bin"},
		"gnea/grbl":                 {"grbl_v1.1h.20190825.hex"},
	}
}

var mockTags = map[string]string{
	"esphome/firmware": "2026.10.1", "wled/WLED": "v0.15.3", "arendst/Tasmota": "v15.1.0", "adafruit/circuitpython": "10.0.3",
	"BitMaker-hub/NerdMiner_v2": "v1.8.3", "meshtastic/firmware": "v2.7.15.567b8ea", "tinkerie-test/app-only": "v1.0.0",
	"tinkerie-test/merged": "1.0", "gnea/grbl": "v1.1h.20190825",
}

// prepare builds every file the mock serves.
func (m *mockLib) prepare() {
	for repo, assets := range m.releases() {
		for _, a := range assets {
			var b []byte
			switch {
			case strings.HasSuffix(a, ".zip"):
				b = mockZip(a)
			case strings.HasSuffix(a, ".hex"):
				b = makeHexFile(fakeImage(a, 30000)[1:], 0)
			case strings.Contains(a, "app_only"):
				b = fakeImage(a, 1_400_000)
			default:
				b = fakeImage(a, 260_000)
			}
			m.put("/dl/"+repo+"/"+a, b)
		}
	}
	for _, bd := range []string{"ESP32_GENERIC", "ESP32_GENERIC_S3", "ESP32_GENERIC_C3", "ESP32_GENERIC_S2", "ESP32_GENERIC_C6", "ESP8266_GENERIC"} {
		m.put("/resources/firmware/"+bd+"-20260406-v1.28.0.bin", fakeImage(bd, 1_600_000))
	}
	for _, bd := range []string{"RPI_PICO", "RPI_PICO_W", "RPI_PICO2", "RPI_PICO2_W"} {
		fam := uint32(famRP2040)
		if strings.Contains(bd, "PICO2") {
			fam = famRP2350ARM
		}
		m.put("/resources/firmware/"+bd+"-20260406-v1.28.0.uf2", binToUF2(fakeImage(bd, 600_000), 0x10000000, fam))
	}
	for _, f := range []string{"tasmota32", "tasmota32s3", "tasmota32c3", "tasmota32s2"} {
		m.put("/tasmota32/release/"+f+".factory.bin", fakeImage(f, 1_900_000))
	}
	m.put("/tasmota/release/tasmota.bin", fakeImage("tasmota.bin", 650_000))
	for _, bd := range []string{"raspberry_pi_pico", "raspberry_pi_pico_w", "raspberry_pi_pico2", "raspberry_pi_pico2_w"} {
		fam := uint32(famRP2040)
		if strings.Contains(bd, "pico2") {
			fam = famRP2350ARM
		}
		name := fmt.Sprintf("adafruit-circuitpython-%s-en_US-10.0.3.uf2", bd)
		m.put("/bin/"+bd+"/en_US/"+name, binToUF2(fakeImage(bd, 900_000), 0x10000000, fam))
	}
	for _, bd := range []string{"espressif_esp32s3_devkitc_1_n8r8", "lolin_s2_mini", "espressif_esp32c3_devkitm_1_n4"} {
		name := fmt.Sprintf("adafruit-circuitpython-%s-en_US-10.0.3.bin", bd)
		m.put("/bin/"+bd+"/en_US/"+name, fakeImage(bd, 1_500_000))
	}
	// NerdMiner bin folders
	for _, dir := range []string{"ESP32-2432S028R", "ESP32-2432S028R-ST7789", "TTGO-T-Display-S3", "ESP32-C3-super-mini"} {
		for _, f := range []string{"0x1000_bootloader.bin", "0x8000_partitions.bin", "0xe000_boot_app0.bin", "0x10000_firmware.bin"} {
			n := dir + "/" + f
			var b []byte
			if strings.HasPrefix(f, "0x10000") {
				b = fakeImage(dir, 1_100_000)
			} else if strings.Contains(f, "partitions") {
				b, _ = buildPartitionTable("min_spiffs")
			} else {
				b = fakeImage(dir+f, 20_000)
			}
			if strings.Contains(f, "bootloader") && strings.Contains(dir, "S3") {
				b = fakeImage("s3 bootloader", 20_000)
			}
			if strings.Contains(f, "bootloader") && strings.Contains(dir, "C3") {
				b = fakeImage("c3 bootloader", 20_000)
			}
			m.put("/raw/nerdminer/bin/"+n, b)
		}
	}
	// WLED's v4 bootloader lives on github.com (rewritten in demo/tests)
	m.put("/gh/Aircoookie/WLED/releases/download/v0.13.1/esp32_bootloader_v4.bin", fakeImage("esp32_bootloader_v4", 0x10000))
}

func mockZip(name string) []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	ver := "2.7.15.567b8ea"
	var entries []string
	switch {
	case strings.Contains(name, "esp32s3"):
		entries = []string{"firmware-t-deck-" + ver + ".factory.bin", "firmware-t-deck-" + ver + ".bin", "firmware-heltec-v3-" + ver + ".factory.bin", "firmware-heltec-v3-" + ver + ".bin"}
	case strings.Contains(name, "esp32-"):
		entries = []string{"firmware-tbeam-" + ver + ".factory.bin", "firmware-tbeam-" + ver + ".bin"}
	case strings.Contains(name, "rp2040"):
		entries = []string{"firmware-rpipico-" + ver + ".uf2", "firmware-rpipicow-" + ver + ".uf2"}
	}
	for _, e := range entries {
		w, _ := zw.Create(e)
		if strings.HasSuffix(e, ".uf2") {
			w.Write(binToUF2(fakeImage(e, 500_000), 0x10000000, famRP2040))
		} else {
			w.Write(fakeImage(e, 900_000))
		}
	}
	zw.Close()
	return buf.Bytes()
}

func makeHexFile(b []byte, base int) []byte {
	var sb strings.Builder
	for i := 0; i < len(b); i += 16 {
		n := min(16, len(b)-i)
		a := base + i
		line := []byte{byte(n), byte(a >> 8), byte(a), 0}
		line = append(line, b[i:i+n]...)
		var s byte
		for _, v := range line {
			s += v
		}
		line = append(line, -s)
		sb.WriteString(":" + strings.ToUpper(hex.EncodeToString(line)) + "\n")
	}
	sb.WriteString(":00000001FF\n")
	return []byte(sb.String())
}

func (m *mockLib) serve(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	m.hits[r.URL.Path]++
	m.mu.Unlock()
	p := r.URL.Path
	base := m.srv.URL
	switch {
	case strings.HasPrefix(p, "/repos/") && strings.HasSuffix(p, "/releases/latest"):
		repo := strings.TrimSuffix(strings.TrimPrefix(p, "/repos/"), "/releases/latest")
		tag, ok := mockTags[repo]
		if !ok {
			http.NotFound(w, r)
			return
		}
		rel := ghRelease{Tag: tag, Name: repo + " " + tag, HTMLURL: "https://github.com/" + repo + "/releases/tag/" + tag,
			Published: "2026-11-01T12:00:00Z", Body: "## Highlights\n- Demo release served by Flashie's offline mock\n- Real releases come from GitHub"}
		for _, a := range m.releases()[repo] {
			fp := "/dl/" + repo + "/" + a
			rel.Assets = append(rel.Assets, ghAsset{Name: a, URL: base + fp, Size: int64(len(m.files[fp])), Digest: m.digest[a]})
		}
		json.NewEncoder(w).Encode(rel)
	case p == "/repos/BitMaker-hub/NerdMiner_v2/contents/bin":
		var out []ghContent
		for _, d := range []string{"ESP32-2432S028R", "ESP32-2432S028R-ST7789", "TTGO-T-Display-S3", "ESP32-C3-super-mini"} {
			out = append(out, ghContent{Name: d, Path: "bin/" + d, Type: "dir"})
		}
		json.NewEncoder(w).Encode(out)
	case strings.HasPrefix(p, "/repos/BitMaker-hub/NerdMiner_v2/contents/bin/"):
		dir := strings.TrimPrefix(p, "/repos/BitMaker-hub/NerdMiner_v2/contents/bin/")
		var out []ghContent
		for _, f := range []string{"0x1000_bootloader.bin", "0x8000_partitions.bin", "0xe000_boot_app0.bin", "0x10000_firmware.bin"} {
			fp := "/raw/nerdminer/bin/" + dir + "/" + f
			out = append(out, ghContent{Name: f, Path: "bin/" + dir + "/" + f, Type: "file", SHA: gitBlobSHA1(m.files[fp]), Size: int64(len(m.files[fp])), DownloadURL: base + fp})
		}
		out = append(out, ghContent{Name: "README.md", Type: "file"})
		json.NewEncoder(w).Encode(out)
	case strings.HasPrefix(p, "/download/"):
		board := strings.Trim(strings.TrimPrefix(p, "/download/"), "/")
		ext := "bin"
		if strings.HasPrefix(board, "RPI_") {
			ext = "uf2"
		}
		fmt.Fprintf(w, `<html><h2>Firmware</h2><a href="/resources/firmware/%[1]s-20260715-v1.29.0-preview.627.g8250ff1c11.%[2]s">preview</a>
<a href="/resources/firmware/%[1]s-SPIRAM-20260406-v1.28.0.%[2]s">spiram</a>
<a href="/resources/firmware/%[1]s-20260406-v1.28.0.%[2]s">v1.28.0</a>
<a href="/resources/firmware/%[1]s-20251209-v1.27.0.%[2]s">v1.27.0</a></html>`, board, ext)
	default:
		b, ok := m.files[p]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Length", fmt.Sprint(len(b)))
		w.Write(b)
	}
}

// useMock points every library endpoint at the mock.
func useMock(m *mockLib) {
	ghAPI, mpyBase, cpyBase, tasBase = m.srv.URL, m.srv.URL, m.srv.URL, m.srv.URL
	urlRewrite = func(u string) string {
		return strings.Replace(u, "https://github.com/", m.srv.URL+"/gh/", 1)
	}
	dir, _ := os.MkdirTemp("", "flashie-cache-")
	cacheRoot = func() string { return dir }
}

func startDemoLibrary() {
	m := newMockLib()
	m.prepare()
	useMock(m)
}
