package flashie

import (
	"bytes"
	"crypto/md5"
	"strings"
	"testing"
	"time"
)

func withMock(t *testing.T) *mockLib {
	m := newMockLib()
	m.prepare()
	oldGH, oldM, oldC, oldT, oldR, oldCache := ghAPI, mpyBase, cpyBase, tasBase, urlRewrite, cacheRoot
	useMock(m)
	dir := t.TempDir()
	cacheRoot = func() string { return dir }
	metaMu.Lock()
	metaCache = map[string]struct {
		at time.Time
		v  []byte
	}{}
	metaMu.Unlock()
	t.Cleanup(func() {
		m.srv.Close()
		ghAPI, mpyBase, cpyBase, tasBase, urlRewrite, cacheRoot = oldGH, oldM, oldC, oldT, oldR, oldCache
	})
	return m
}

func TestPartitionTable(t *testing.T) {
	b, err := buildPartitionTable("min_spiffs")
	if err != nil || len(b) != 0xC00 || b[0] != 0xAA || b[1] != 0x50 {
		t.Fatalf("%v %d", err, len(b))
	}
	// app0 entry
	if b[2*32+2] != 0 || b[2*32+3] != 0x10 || le(b[2*32+4:]) != 0x10000 || le(b[2*32+8:]) != 0x1E0000 || string(bytes.TrimRight(b[2*32+12:2*32+28], "\x00")) != "app0" {
		t.Fatalf("bad app0 entry % x", b[64:96])
	}
	md := b[6*32:]
	sum := md5.Sum(b[:6*32])
	if md[0] != 0xEB || md[1] != 0xEB || !bytes.Equal(md[16:32], sum[:]) {
		t.Fatal("bad md5 entry")
	}
}

func le(b []byte) uint32 { return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24 }

func TestLibraryFiltering(t *testing.T) {
	names := func(board, chip string) []string {
		var out []string
		for _, e := range libraryFor(board, chip) {
			if e.Available {
				out = append(out, e.ID)
			}
		}
		return out
	}
	cyd := strings.Join(names("cyd-2432s028r", "ESP32"), ",")
	for _, want := range []string{"nerdminer", "micropython", "wled", "tasmota", "esphome"} {
		if !strings.Contains(cyd, want) {
			t.Errorf("CYD list missing %s: %s", want, cyd)
		}
	}
	if !strings.HasPrefix(cyd, "nerdminer") {
		t.Errorf("board-specific builds should sort first: %s", cyd)
	}
	uno := strings.Join(names("uno", "ATmega328P"), ",")
	if !strings.Contains(uno, "grbl") || strings.Contains(uno, "wled") {
		t.Errorf("uno: %s", uno)
	}
	pico := strings.Join(names("pico", "RP2040"), ",")
	if !strings.Contains(pico, "micropython") || !strings.Contains(pico, "circuitpython") || !strings.Contains(pico, "klipper") {
		t.Errorf("pico: %s", pico)
	}
}

// testItems adds two library entries that exist only in the tests, served by
// the mock library: an app-only image (Flashie adds the partition table) and
// a merged image written at 0x0.
func testItems(t *testing.T) {
	old := library
	library = append(append([]LibItem{}, library...),
		LibItem{ID: "apponly", Name: "App-only test", Category: "Fun", Targets: []LibTarget{
			boardsT(Source{Kind: "gh-asset", Repo: "tinkerie-test/app-only", Asset: `_cyd_2432S028\.bin$`, Offset: 0x10000, AppOnly: true}, cydR...)}},
		LibItem{ID: "merged", Name: "Merged test", Category: "Fun", Targets: []LibTarget{
			boardsT(Source{Kind: "gh-asset", Repo: "tinkerie-test/merged", Asset: `(?i)^merged-demo-CYD-2432S028\.bin$`, Offset: 0}, cydR...)}})
	t.Cleanup(func() { library = old })
}

func TestResolveAll(t *testing.T) {
	withMock(t)
	testItems(t)
	cases := []struct {
		id, board, chip string
		parts           int
	}{
		{"micropython", "esp32-devkitc", "ESP32", 1}, {"micropython", "pico2", "RP2350", 1}, {"micropython", "esp32-s3-devkitc", "ESP32-S3", 1},
		{"circuitpython", "pico-w", "RP2040", 1}, {"esphome", "esp32-devkitc", "ESP32", 1}, {"esphome", "nodemcu-v2", "ESP8266", 1},
		{"wled", "esp32-devkitc", "ESP32", 2}, {"tasmota", "esp32-c3-devkitm", "ESP32-C3", 1},
		{"meshtastic", "t-deck", "ESP32-S3", 1}, {"meshtastic", "pico", "RP2040", 1},
		{"apponly", "cyd-2432s028r", "ESP32", 3}, {"merged", "cyd-2432s028r", "ESP32", 1}, {"nerdminer", "cyd-2432s028r", "ESP32", 4}, {"nerdminer", "t-display-s3", "ESP32-S3", 4},
		{"grbl", "nano", "ATmega328P", 1},
	}
	for _, c := range cases {
		res, err := resolveItem(c.id, c.board, c.chip)
		if err != nil {
			t.Errorf("%s/%s: %v", c.id, c.board, err)
			continue
		}
		if len(res.Parts) != c.parts || res.Version == "" {
			t.Errorf("%s/%s: %d parts, version %q: %+v", c.id, c.board, len(res.Parts), res.Version, res.Parts)
		}
		t.Logf("%-13s %-18s %-12s %s | %s", c.id, c.board, res.Version, res.Parts[len(res.Parts)-1].Name, res.Verified)
	}
	// NerdMiner must pick the right folder per revision.
	r1, _ := resolveItem("nerdminer", "cyd-2432s028r", "ESP32")
	r3, _ := resolveItem("nerdminer", "cyd-2432s028r-v3", "ESP32")
	if strings.Contains(r1.Parts[0].Name, "ST7789") || !strings.Contains(r1.Parts[0].Name, "2432S028R/") {
		t.Errorf("R picked %s", r1.Parts[0].Name)
	}
	_ = r3
	if _, err := resolveItem("klipper", "pico", "RP2040"); err == nil {
		t.Error("klipper should be build-only")
	}
	if _, err := resolveItem("wled", "pico", "RP2040"); err == nil {
		t.Error("wled shouldn't resolve for pico")
	}
}

func TestDownloadVerifyAndCache(t *testing.T) {
	m := withMock(t)
	testItems(t)
	res, err := resolveItem("apponly", "cyd-2432s028r", "ESP32")
	if err != nil {
		t.Fatal(err)
	}
	r := testRep(t)
	imgs, err := downloadParts(res, r)
	if err != nil {
		t.Fatal(err)
	}
	if len(imgs) != 3 || imgs[0].Offset != 0x8000 || imgs[1].Offset != 0xE000 || imgs[2].Offset != 0x10000 {
		t.Fatalf("%+v", imgs)
	}
	hits := m.hits["/dl/tinkerie-test/app-only/app_only_demo_cyd_2432S028.bin"]
	// Second time comes from cache.
	if _, err := downloadParts(res, r); err != nil {
		t.Fatal(err)
	}
	if m.hits["/dl/tinkerie-test/app-only/app_only_demo_cyd_2432S028.bin"] != hits {
		t.Fatal("cache not used")
	}
	res2, _ := resolveItem("apponly", "cyd-2432s028r", "ESP32")
	if !res2.Parts[2].Cached {
		t.Fatal("resolve should report cached")
	}
	// Corrupt the published file: download must be refused.
	res3, _ := resolveItem("merged", "cyd-2432s028r", "ESP32")
	p := "/dl/tinkerie-test/merged/merged-demo-CYD-2432S028.bin"
	m.files[p] = append([]byte{}, m.files[p]...)
	m.files[p][5000] ^= 0xFF
	if _, err := downloadParts(res3, r); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("expected checksum mismatch, got %v", err)
	}
	// Git-hash path (NerdMiner) also rejects tampering.
	res4, _ := resolveItem("nerdminer", "cyd-2432s028r", "ESP32")
	fp := "/raw/nerdminer/bin/ESP32-2432S028R/0x10000_firmware.bin"
	m.files[fp] = append([]byte{}, m.files[fp]...)
	m.files[fp][100] ^= 1
	if _, err := downloadParts(res4, r); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("expected git checksum mismatch, got %v", err)
	}
}

func TestLibraryFlashEndToEnd(t *testing.T) {
	withMock(t)
	testItems(t)
	demoMode = true
	resetDemo()
	defer func() { demoMode = false }()
	cases := []struct{ item, port, board, chip string }{
		{"apponly", "COM3", "cyd-2432s028r", "ESP32"},
		{"wled", "COM3", "esp32-devkitc", "ESP32"},
		{"micropython", "COM3", "esp32-devkitc", "ESP32"},
		{"nerdminer", "COM3", "cyd-2432s028r", "ESP32"},
		{"esphome", "COM4", "nodemcu-v2", "ESP8266"},
		{"micropython", "COM7", "esp32-s3-devkitc", "ESP32-S3"},
		{"grbl", "COM5", "uno", "ATmega328P"},
		{"micropython", "P:", "pico2", "RP2350"},
		{"meshtastic", "COM10", "pico", "RP2040"},
	}
	for _, c := range cases {
		msg, err := runLibFlash(libFlashReq{Item: c.item, Port: c.port, Board: boardByID(c.board), BoardID: c.board, Chip: c.chip, Baud: 460800, Compress: true}, testRep(t))
		if err != nil {
			t.Fatalf("%s on %s: %v", c.item, c.port, err)
		}
		t.Logf("%s → %s", c.item, msg)
	}
	// Wrong chip: S3 firmware onto the ESP32 sim must be refused before writing.
	_, err := runLibFlash(libFlashReq{Item: "micropython", Port: "COM3", BoardID: "esp32-s3-devkitc", Chip: "ESP32-S3", Baud: 460800}, testRep(t))
	if err == nil || !strings.Contains(err.Error(), "built for an ESP32-S3") {
		t.Fatalf("expected chip mismatch refusal, got %v", err)
	}
}
