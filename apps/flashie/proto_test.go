package flashie

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func randImg(n int, seed int64) []byte {
	b := make([]byte, n)
	rand.New(rand.NewSource(seed)).Read(b)
	return b
}

func testRep(t *testing.T) *reporter {
	return &reporter{Log: func(s string) { t.Log(s) }}
}

func TestESP8266(t *testing.T) {
	sim := newSimChip(esp8266SimID, 0)
	l := NewLoader(sim, false, false)
	if err := l.Connect(); err != nil {
		t.Fatal(err)
	}
	info, err := l.DetectChip()
	if err != nil || info.Chip != "ESP8266" {
		t.Fatalf("%v %+v", err, info)
	}
	if err := l.ChangeBaud(921600); err != nil {
		t.Fatal(err)
	}
	img := randImg(150_003, 3)
	if err := l.Flash(img, FlashOptions{Offset: 0, FlashSize: 4 << 20, Compress: true}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sim.flash[:len(img)], img) {
		t.Fatal("mismatch")
	}
}

func TestESP8266EraseSize(t *testing.T) {
	// Values from esptool's get_erase_size.
	if got := esp8266EraseSize(0, 0x100000); got != 0xF0000 {
		t.Fatalf("got %x", got)
	}
	if got := esp8266EraseSize(0, 4096); got != 4096 {
		t.Fatalf("got %x", got)
	}
}

func TestSTK500v1(t *testing.T) {
	for _, baud := range []int{115200, 57600} {
		sim := newSimSTK1(baud, [3]byte{0x1E, 0x95, 0x0F}, 32768)
		s, tr, err := avrAuto(sim, avrTries(nil, PortInfo{}), testRep(t))
		if err != nil {
			t.Fatalf("baud %d: %v", baud, err)
		}
		info, mcu, err := describeAVR(s, tr, testRep(t))
		if err != nil || mcu.Name != "ATmega328P" || info.Baud != baud {
			t.Fatalf("%v %+v", err, info)
		}
		img := randImg(9001, 4)
		if err := flashAVR(s, mcu, img, 32256, testRep(t)); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(sim.flash[:len(img)], img) {
			t.Fatal("mismatch")
		}
	}
}

func TestSTK500v2(t *testing.T) {
	sim := newSimSTK2([3]byte{0x1E, 0x98, 0x01}, 262144)
	s, tr, err := avrAuto(sim, avrTries(nil, PortInfo{}), testRep(t))
	if err != nil {
		t.Fatal(err)
	}
	info, mcu, err := describeAVR(s, tr, testRep(t))
	if err != nil || mcu.Name != "ATmega2560" || info.Proto != "stk500v2" {
		t.Fatalf("%v %+v", err, info)
	}
	img := randImg(140_000, 5) // crosses the 64K-word boundary
	if err := flashAVR(s, mcu, img, 253952, testRep(t)); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sim.flash[:len(img)], img) {
		t.Fatal("mismatch")
	}
}

func TestAVR109(t *testing.T) {
	sim := newSimAVR109(true)
	s, err := connectAVR109(sim)
	if err != nil {
		t.Fatal(err)
	}
	sig, _ := s.Signature()
	mcu := mcuBySig(sig)
	if mcu == nil || mcu.Name != "ATmega32U4" {
		t.Fatalf("sig %x", sig)
	}
	img := randImg(20001, 6)
	if err := flashAVR(s, mcu, img, 28672, testRep(t)); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sim.flash[:len(img)], img) {
		t.Fatal("mismatch")
	}
}

func TestAVRBootloaderHexRejected(t *testing.T) {
	sim := newSimAVR109(true)
	s, _ := connectAVR109(sim)
	mcu := mcuBySig([]byte{0x1E, 0x95, 0x87})
	err := flashAVR(s, mcu, make([]byte, 32000), 28672, testRep(t))
	if err == nil || !strings.Contains(err.Error(), "bootloader") {
		t.Fatalf("got %v", err)
	}
}

func TestSTM32(t *testing.T) {
	sim := newSimSTM32()
	s, err := connectSTM32(sim, testRep(t))
	if err != nil {
		t.Fatal(err)
	}
	if s.pid != 0x410 || s.flashSizeKB() != 64 {
		t.Fatalf("pid %x", s.pid)
	}
	img := randImg(50_001, 7)
	copy(img[1000:], bytes.Repeat([]byte{0xFF}, 600)) // exercise the skip-blank path
	if err := flashSTM32(s, stmFlashBase, img, testRep(t)); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sim.flash[:len(img)], img) {
		t.Fatal("mismatch")
	}
	if err := s.Go(stmFlashBase); err != nil {
		t.Fatal(err)
	}
}

func TestIntelHex(t *testing.T) {
	src := ":020000040800F2\n:10000000000102030405060708090A0B0C0D0E0F78\n:04001000DEADBEEFB4\n:00000001FF\n"
	m, err := parseIntelHex([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if m.Base != 0x08000000 || len(m.Data) != 20 || m.Data[16] != 0xDE {
		t.Fatalf("%x %d % x", m.Base, len(m.Data), m.Data)
	}
	if sniffFormat("a.hex", []byte(src)) != fmtHex {
		t.Fatal("sniff")
	}
	if _, err := parseIntelHex([]byte(":0400100000ADBEEFB4\n")); err == nil {
		t.Fatal("expected checksum error")
	}
}

func TestUF2RoundTripAndCopy(t *testing.T) {
	bin := randImg(70_000, 8)
	u := binToUF2(bin, 0x10000000, famRP2040)
	info, err := inspectUF2(u)
	if err != nil || info.Blocks != 274 || !info.has(famRP2040) {
		t.Fatalf("%v %+v", err, info)
	}
	if sniffFormat("x.uf2", u) != fmtUF2 {
		t.Fatal("sniff")
	}
	dir := t.TempDir()
	d := parseInfoUF2(dir, "UF2 Bootloader v1.0\nModel: Raspberry Pi RP2350\nBoard-ID: RP2350\n")
	if !d.isRP2350() || d.isRP2040() {
		t.Fatal("rp2350 detect")
	}
	if _, err := prepareUF2("x.uf2", u, d, testRep(t)); err == nil {
		t.Fatal("expected RP2040-on-RP2350 rejection")
	}
	out, err := prepareUF2("fw.bin", bin, d, testRep(t))
	if err != nil {
		t.Fatal(err)
	}
	demoMode = true
	defer func() { demoMode = false }()
	if err := copyUF2(d, out, testRep(t)); err != nil {
		t.Fatal(err)
	}
	_ = os.Remove(filepath.Join(dir, "FLASHIE.UF2"))
}

func TestGuesses(t *testing.T) {
	for _, c := range []struct {
		vid, pid uint16
		fam      string
	}{{0x2341, 0x0043, famAVR}, {0x2E8A, 0x000A, famRP2}, {0x303A, 0x1001, famESP}, {0x2341, 0x8036, famAVR}} {
		g := guessBoard(PortInfo{USB: true, VID: c.vid, PID: c.pid})
		if g.Family != c.fam || boardByID(g.BoardID) == nil {
			t.Errorf("%04X:%04X → %+v", c.vid, c.pid, g)
		}
	}
	ids := map[string]bool{}
	for _, b := range boards {
		if ids[b.ID] {
			t.Errorf("duplicate board id %s", b.ID)
		}
		ids[b.ID] = true
	}
	fmt.Println(len(boards), "boards")
}

func TestDemoEndToEnd(t *testing.T) {
	demoMode = true
	resetDemo()
	defer func() { demoMode = false }()
	for _, c := range []struct{ port, file string }{
		{"COM3", "fw.bin"}, {"COM4", "fw.bin"}, {"COM5", "sketch.hex"}, {"COM6", "sketch.hex"},
		{"COM8", "sketch.hex"}, {"COM9", "fw.bin"}, {"COM10", "fw.uf2"}, {"P:", "fw.uf2"},
	} {
		var data []byte
		switch {
		case strings.HasSuffix(c.file, ".hex"):
			data = makeHex(randImg(3000, 9), 0)
		case c.port == "COM10":
			data = binToUF2(randImg(5000, 9), 0x10000000, famRP2040)
		case c.port == "P:":
			data = binToUF2(randImg(5000, 9), 0x10000000, famRP2350ARM)
		default:
			data = randImg(40000, 9)
		}
		pi, _ := findDevice(c.port)
		var hint *ChipInfo
		if pi.Guess.Family == "" { // ambiguous bridge chip: identify first, like the UI does
			info, err := identify(pi, nil, testRep(t))
			if err != nil {
				t.Fatalf("%s identify: %v", c.port, err)
			}
			hint = info
		}
		req := flashReq{Port: c.port, Data: data, FileName: c.file, Baud: 460800, Compress: true, Hint: hint}
		if c.port == "COM3" || c.port == "COM4" {
			req.Offset, req.HasOff = 0x10000, true
		}
		start := time.Now()
		msg, err := runFlash(req, testRep(t))
		if err != nil {
			t.Fatalf("%s: %v", c.port, err)
		}
		t.Logf("%s (%.1fs): %s", c.port, time.Since(start).Seconds(), msg)
	}
}

func makeHex(b []byte, base int) []byte {
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
		sb.WriteString(":" + strings.ToUpper(fmt.Sprintf("%x", line)) + "\n")
	}
	sb.WriteString(":00000001FF\n")
	return []byte(sb.String())
}

func TestCYDDatabase(t *testing.T) {
	ids := map[string]bool{}
	for _, c := range cyds {
		if ids[c.ID] {
			t.Errorf("duplicate CYD %s", c.ID)
		}
		ids[c.ID] = true
		if boardByID("cyd-"+c.ID) == nil {
			t.Errorf("no board entry for %s", c.ID)
		}
		used := map[int]string{}
		for _, g := range c.Groups {
			for _, p := range g.Pins {
				if p.GPIO < 0 || p.GPIO > 48 {
					t.Errorf("%s: bad GPIO %d", c.ID, p.GPIO)
				}
				// A pin may be shared by buses (SPI) but never by two display data lines.
				if prev, ok := used[p.GPIO]; ok && strings.HasPrefix(g.Name, "Display") && strings.HasPrefix(prev, "Display") {
					t.Errorf("%s: GPIO %d used twice in display (%s)", c.ID, p.GPIO, p.Func)
				}
				used[p.GPIO] = g.Name
			}
		}
	}
	t.Logf("%d CYD boards", len(cyds))
}

func TestCYDMatch(t *testing.T) {
	m := matchCYDFirmware("MyApp_v1.2_CYD_2432S028R.bin", []byte("xx esp32-2432s028r yy"))
	if len(m.Families) != 1 || m.Families[0] != "2432S028" || m.Hits[0].ID != "2432s028r" {
		t.Fatalf("%+v", m)
	}
	m = matchCYDFirmware("fw.bin", []byte("....ESP32_2432S028Rv3 ...ST7789...XPT2046"))
	var ids []string
	for _, h := range m.Hits {
		if h.ID != "" {
			ids = append(ids, h.ID)
		}
	}
	if len(ids) != 1 || ids[0] != "2432s028r-v3" || len(m.Drivers) != 2 {
		t.Fatalf("%v %+v", ids, m)
	}
	m = matchCYDFirmware("openhasp_8048s070c.bin", nil)
	if len(m.Families) != 1 || m.Families[0] != "8048S070" {
		t.Fatalf("%+v", m)
	}
	m = matchCYDFirmware("blink.bin", []byte("nothing here"))
	if len(m.Hits) != 0 {
		t.Fatalf("%+v", m)
	}
}
