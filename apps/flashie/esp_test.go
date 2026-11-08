package flashie

import (
	"bytes"
	"math/rand"
	"testing"
)

func testImage(n int) []byte {
	r := rand.New(rand.NewSource(1))
	img := make([]byte, n)
	r.Read(img[:n/2]) // half random, half compressible
	for i := n / 2; i < n; i++ {
		img[i] = byte(i % 7)
	}
	img[0] = 0xE9
	return img
}

func runESPFlash(t *testing.T, chipID uint32, compress bool, size int, offset uint32) {
	sim := newSimChip(chipID, 0)
	l := NewLoader(sim, false, false)
	if err := l.Connect(); err != nil {
		t.Fatalf("connect: %v", err)
	}
	info, err := l.DetectChip()
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	t.Logf("chip=%s mac=%s", info.Chip, info.MAC)
	if err := l.ChangeBaud(460800); err != nil {
		t.Fatal(err)
	}
	img := testImage(size)
	last := 0
	err = l.Flash(img, FlashOptions{Offset: offset, FlashSize: 4 << 20, Compress: compress,
		Progress: func(d, tot int) { last = d }})
	if err != nil {
		t.Fatalf("flash: %v", err)
	}
	if last < len(img) {
		t.Fatalf("progress ended at %d/%d", last, len(img))
	}
	if !bytes.Equal(sim.flash[offset:int(offset)+size], img) {
		t.Fatal("flash contents differ")
	}
}

func TestESP32Compressed(t *testing.T)   { runESPFlash(t, 0, true, 300_001, 0x1000) }
func TestESP32Plain(t *testing.T)        { runESPFlash(t, 0, false, 70_003, 0x10000) }
func TestESP32S3Compressed(t *testing.T) { runESPFlash(t, 9, true, 1_200_000, 0x0) }

func TestDetectMAC(t *testing.T) {
	sim := newSimChip(0, 0)
	l := NewLoader(sim, false, false)
	if err := l.Connect(); err != nil {
		t.Fatal(err)
	}
	info, err := l.DetectChip()
	if err != nil {
		t.Fatal(err)
	}
	if info.Chip != "ESP32" || info.MAC != "24:62:AB:12:CD:34" {
		t.Fatalf("got %+v", info)
	}
}

func TestSlipRoundTrip(t *testing.T) {
	in := []byte{1, 0xC0, 2, 0xDB, 3, 0xC0, 0xDB}
	sim := newSimChip(0, 0)
	l := NewLoader(sim, false, false)
	sim.out.Write(slipEncode(in))
	got, err := l.readPacket(100e6)
	if err != nil || !bytes.Equal(got, in) {
		t.Fatalf("got %v %v", got, err)
	}
}
