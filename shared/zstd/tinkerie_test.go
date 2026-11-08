package zstd

// Tinkerie's tests: files made by the real zstd library (testdata/make.py,
// Python 3.14's compression.zstd), including a match more than 8 MiB back,
// which Go's original 8 MiB window cap couldn't unpack.

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

func want(t *testing.T) (string, int64) {
	b, err := os.ReadFile("testdata/plain.sha256")
	if err != nil {
		t.Fatal(err)
	}
	var h string
	var n int64
	fmt.Sscan(string(b), &h, &n)
	return h, n
}

func unpack(name string) (string, int64, error) {
	f, err := os.Open("testdata/" + name)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, NewReader(f))
	return hex.EncodeToString(h.Sum(nil)), n, err
}

func TestRealZstdFiles(t *testing.T) {
	wh, wn := want(t)
	for _, name := range []string{"level19.zst", "long27.zst", "nosize.zst", "multi.zst"} {
		h, n, err := unpack(name)
		if err != nil || h != wh || n != wn {
			t.Errorf("%s: %d bytes, %v, hash ok %v", name, n, err, h == wh)
		}
	}
}

func TestBadZstdFiles(t *testing.T) {
	for name, msg := range map[string]string{
		"damaged.zst":    "",
		"truncated.zst":  "",
		"hugewindow.zst": "window",
	} {
		_, _, err := unpack(name)
		if err == nil || !strings.Contains(err.Error(), msg) {
			t.Errorf("%s: got %v", name, err)
		}
	}
}

func TestIsZstd(t *testing.T) {
	b, _ := os.ReadFile("testdata/long27.zst")
	if !IsZstd(b) || IsZstd([]byte("PK\x03\x04")) || IsZstd(nil) {
		t.Fatal("IsZstd")
	}
	if !bytes.HasPrefix(b, []byte{0x28, 0xB5, 0x2F, 0xFD}) {
		t.Fatal("magic")
	}
}

// long27.zst really does reach further back than Go's old 8 MiB cap.
func TestLongWindowNeeded(t *testing.T) {
	old := maxWindow
	maxWindow = 8 << 20
	defer func() { maxWindow = old }()
	if _, _, err := unpack("long27.zst"); err == nil || !strings.Contains(err.Error(), "window") {
		t.Fatalf("expected a window error, got %v", err)
	}
}
