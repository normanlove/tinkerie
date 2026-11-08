package xz

import (
	"bytes"
	"errors"
	"io"
	"math/rand"
	"os/exec"
	"testing"
)

func xzTool(t *testing.T, data []byte, args ...string) []byte {
	t.Helper()
	if _, err := exec.LookPath("xz"); err != nil {
		t.Skip("xz tool not installed")
	}
	cmd := exec.Command("xz", append([]string{"-c", "-T1"}, args...)...)
	cmd.Stdin = bytes.NewReader(data)
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func sample(n int, seed int64) []byte {
	r := rand.New(rand.NewSource(seed))
	words := []string{"raspberry ", "pi ", "image ", "boot ", "\x00\x00\x00\x00", "linux ", "kernel ", "\n"}
	var b bytes.Buffer
	for b.Len() < n {
		switch r.Intn(4) {
		case 0: // random bytes, compress poorly
			k := r.Intn(300)
			for i := 0; i < k; i++ {
				b.WriteByte(byte(r.Intn(256)))
			}
		case 1: // long zero run, like empty partitions
			b.Write(make([]byte, r.Intn(20000)))
		default:
			b.WriteString(words[r.Intn(len(words))])
		}
	}
	return b.Bytes()[:n]
}

func decode(t *testing.T, comp []byte) ([]byte, error) {
	t.Helper()
	r, err := NewReader(bytes.NewReader(comp))
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	buf := make([]byte, 7777) // odd size to exercise partial reads
	_, err = io.CopyBuffer(&out, struct{ io.Reader }{r}, buf)
	return out.Bytes(), err
}

func TestAgainstXZTool(t *testing.T) {
	data := sample(9<<20, 1)
	cases := [][]string{
		{"-0"}, {"-6"}, {"-9e"}, {"--check=crc32"}, {"--check=none"}, {"--check=sha256"},
		{"--lzma2=preset=6,lc=0,lp=2,pb=0"}, {"--lzma2=preset=1,lc=4,lp=0,pb=4"},
		{"--block-size=1MiB"}, {"--format=xz", "--lzma2=dict=64KiB"},
	}
	for _, c := range cases {
		comp := xzTool(t, data, c...)
		got, err := decode(t, comp)
		if err != nil {
			t.Fatalf("%v: %v", c, err)
		}
		if !bytes.Equal(got, data) {
			t.Fatalf("%v: output differs (got %d bytes, want %d)", c, len(got), len(data))
		}
	}
}

func TestIncompressibleAndEmpty(t *testing.T) {
	rnd := make([]byte, 3<<20)
	rand.New(rand.NewSource(2)).Read(rnd)
	for _, d := range [][]byte{rnd, {}, []byte("x")} {
		got, err := decode(t, xzTool(t, d))
		if err != nil || !bytes.Equal(got, d) {
			t.Fatalf("len %d: err=%v equal=%v", len(d), err, bytes.Equal(got, d))
		}
	}
}

func TestMultiStreamAndPadding(t *testing.T) {
	a, b := sample(100000, 3), sample(50000, 4)
	comp := append(xzTool(t, a), make([]byte, 8)...)
	comp = append(comp, xzTool(t, b, "--check=sha256")...)
	got, err := decode(t, comp)
	if err != nil || !bytes.Equal(got, append(append([]byte{}, a...), b...)) {
		t.Fatalf("err=%v", err)
	}
	if _, err := decode(t, append(xzTool(t, a), 0, 0, 0)); err == nil {
		t.Fatal("bad padding accepted")
	}
}

func TestCorruption(t *testing.T) {
	data := sample(400000, 5)
	comp := xzTool(t, data)
	bad := 0
	r := rand.New(rand.NewSource(6))
	for i := 0; i < 200; i++ {
		c := append([]byte{}, comp...)
		c[12+r.Intn(len(c)-12)] ^= byte(1 << r.Intn(8))
		got, err := decode(t, c)
		if err == nil && !bytes.Equal(got, data) {
			t.Fatal("corruption went unnoticed")
		}
		if err != nil {
			bad++
		}
	}
	if bad < 190 {
		t.Fatalf("only %d of 200 corruptions detected", bad)
	}
	if _, err := decode(t, comp[:len(comp)/2]); err == nil {
		t.Fatal("truncated file accepted")
	}
	if _, err := NewReader(bytes.NewReader([]byte("hello world!"))); !errors.Is(err, ErrFormat) {
		t.Fatal("non-xz accepted")
	}
}

func TestBCJRejected(t *testing.T) {
	comp := xzTool(t, sample(10000, 7), "--x86", "--lzma2")
	if _, err := decode(t, comp); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("got %v", err)
	}
}

func BenchmarkDecode(b *testing.B) {
	t := &testing.T{}
	data := sample(32<<20, 8)
	cmd := exec.Command("xz", "-c", "-6", "-T1")
	cmd.Stdin = bytes.NewReader(data)
	comp, err := cmd.Output()
	if err != nil {
		b.Skip(err)
	}
	_ = t
	b.SetBytes(int64(len(data)))
	for i := 0; i < b.N; i++ {
		r, _ := NewReader(bytes.NewReader(comp))
		io.Copy(io.Discard, r)
	}
}
