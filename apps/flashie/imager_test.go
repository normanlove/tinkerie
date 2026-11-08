package flashie

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func fakeDiskImage(n int) []byte {
	b := make([]byte, n)
	r := rand.New(rand.NewSource(int64(n)))
	for i := 0; i < n; i += 4096 { // half random, half zeros, like a real image
		if (i/4096)%2 == 0 {
			r.Read(b[i:min(n, i+4096)])
		}
	}
	copy(b[510:], []byte{0x55, 0xAA})
	return b
}

func sha(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }

// writeVariants saves the image as raw, gz, bz2, xz and zip.
func writeVariants(t *testing.T, dir string, img []byte) map[string]string {
	t.Helper()
	out := map[string]string{}
	raw := filepath.Join(dir, "os.img")
	os.WriteFile(raw, img, 0o644)
	out["raw"] = raw
	var gz bytes.Buffer
	w := gzip.NewWriter(&gz)
	w.Write(img)
	w.Close()
	out["gzip"] = filepath.Join(dir, "os.img.gz")
	os.WriteFile(out["gzip"], gz.Bytes(), 0o644)
	var zb bytes.Buffer
	zw := zip.NewWriter(&zb)
	zw.Create("README.txt")
	f, _ := zw.Create("images/os.img")
	f.Write(img)
	zw.Close()
	out["zip"] = filepath.Join(dir, "os.zip")
	os.WriteFile(out["zip"], zb.Bytes(), 0o644)
	for _, tool := range []struct{ kind, cmd, ext string }{{"xz", "xz", ".xz"}, {"bzip2", "bzip2", ".bz2"}} {
		if _, err := exec.LookPath(tool.cmd); err != nil {
			continue
		}
		c := exec.Command(tool.cmd, "-c", "-k", raw)
		b, err := c.Output()
		if err != nil {
			t.Fatal(err)
		}
		p := raw + tool.ext
		os.WriteFile(p, b, 0o644)
		out[tool.kind] = p
	}
	return out
}

func useTestDisk(t *testing.T, mb int) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "card.bin")
	old := []byte("OLD CARD CONTENTS ")
	os.WriteFile(p, bytes.Repeat(old, (mb<<20)/len(old)), 0o644)
	t.Setenv("FLASHIE_IMG_TESTDISK", p+":"+itoa(mb))
	return p
}

func itoa(n int) string { return strconv.Itoa(n) }

type atomicBool struct{ b atomic.Bool }

func TestWriteEveryFormat(t *testing.T) {
	img := fakeDiskImage(9<<20 + 1536) // not a whole number of blocks
	vs := writeVariants(t, t.TempDir(), img)
	for kind, p := range vs {
		disk := useTestDisk(t, 32)
		info, err := sniffImage(p)
		if err != nil || info.Kind != kind {
			t.Fatalf("%s: sniff %+v %v", kind, info, err)
		}
		var phases []string
		_, err = runWrite(writeSpec{Device: disk, Source: info, Size: info.Size, SHA256: sha(img), Verify: true},
			func(pr progress) bool {
				if len(phases) == 0 || phases[len(phases)-1] != pr.Phase {
					phases = append(phases, pr.Phase)
				}
				return false
			})
		if err != nil {
			t.Fatalf("%s: %v", kind, err)
		}
		got, _ := os.ReadFile(disk)
		if !bytes.Equal(got[:len(img)], img) {
			t.Fatalf("%s: drive contents differ", kind)
		}
		if !bytes.Equal(got[len(img):len(img)+512], make([]byte, 512)) {
			t.Fatalf("%s: padding after the image isn't blank", kind)
		}
		if !bytes.Equal(got[len(got)-(1<<20):], make([]byte, 1<<20)) {
			t.Fatalf("%s: old backup partition table area not blanked", kind)
		}
		if strings.Join(phases, ",") != "prepare,write,verify,finish" {
			t.Fatalf("%s: phases %v", kind, phases)
		}
	}
}

func TestWriteRefusals(t *testing.T) {
	dir := t.TempDir()
	img := fakeDiskImage(6 << 20)
	vs := writeVariants(t, dir, img)
	disk := useTestDisk(t, 4)
	info, _ := sniffImage(vs["gzip"])
	if _, err := runWrite(writeSpec{Device: disk, Source: info}, func(progress) bool { return false }); err == nil || !strings.Contains(err.Error(), "bigger than the drive") {
		t.Fatalf("too-big image: %v", err)
	}
	disk = useTestDisk(t, 16)
	if _, err := runWrite(writeSpec{Device: disk, Source: info, SHA256: strings.Repeat("0", 64)}, func(progress) bool { return false }); err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("checksum mismatch: %v", err)
	}
	if _, err := runWrite(writeSpec{Device: "/dev/sda", Source: info}, func(progress) bool { return false }); err == nil || !strings.Contains(err.Error(), "isn't plugged in") {
		t.Fatalf("unknown drive accepted: %v", err)
	}
	// Damaged xz: flip a byte in the middle.
	if p, ok := vs["xz"]; ok {
		b, _ := os.ReadFile(p)
		b[len(b)/2] ^= 0x40
		os.WriteFile(p, b, 0o644)
		xi, _ := sniffImage(p)
		if _, err := runWrite(writeSpec{Device: disk, Source: xi}, func(progress) bool { return false }); err == nil || !strings.Contains(err.Error(), "damaged") {
			t.Fatalf("damaged xz: %v", err)
		}
	}
	// Cancel part way; the start of the drive must not hold the new image.
	ri, _ := sniffImage(vs["raw"])
	n := 0
	_, err := runWrite(writeSpec{Device: disk, Source: ri}, func(p progress) bool { n++; return p.Phase == "write" })
	if err != errImgCancelled {
		t.Fatalf("cancel: %v", err)
	}
	got, _ := os.ReadFile(disk)
	if bytes.Equal(got[:4096], img[:4096]) {
		t.Fatal("a cancelled write left a bootable start")
	}
	if _, err := sniffImage(dir); err == nil {
		t.Fatal("folder accepted as an image")
	}
}

func TestPartOfAndPad(t *testing.T) {
	for _, c := range []struct {
		dev, disk string
		want      bool
	}{{"/dev/sdb1", "/dev/sdb", true}, {"/dev/sdb", "/dev/sdb", true}, {"/dev/sdbc1", "/dev/sdb", false},
		{"/dev/mmcblk0p2", "/dev/mmcblk0", true}, {"/dev/mmcblk01", "/dev/mmcblk0", true}, {"/dev/sda1", "/dev/sdb", false}} {
		if partOf(c.dev, c.disk) != c.want {
			t.Errorf("partOf(%s, %s)", c.dev, c.disk)
		}
	}
	if padLen(1000, 0, 1<<20) != 4096 || padLen(1000, (1<<20)-1024, 1<<20) != 1024 {
		t.Error("padLen")
	}
}

func TestPlist(t *testing.T) {
	v, err := parsePlist([]byte(`<?xml version="1.0"?><plist version="1.0"><dict><key>WholeDisks</key><array><string>disk4</string></array>
	<key>Internal</key><false/><key>TotalSize</key><integer>31914983424</integer><key>APFSPhysicalStores</key><array><dict><key>APFSPhysicalStore</key><string>disk0s2</string></dict></array></dict></plist>`))
	m, _ := v.(map[string]any)
	if err != nil || m["TotalSize"] != int64(31914983424) || m["Internal"] != false || m["WholeDisks"].([]any)[0] != "disk4" {
		t.Fatalf("%v %#v", err, v)
	}
}

// TestCatalogueDownloadAndHelper runs the whole path a user takes: OS list
// from a server, download (resumed) with checksum, then a write done by the
// "elevated helper" talking to Flashie's own endpoints.
func TestCatalogueDownloadAndHelper(t *testing.T) {
	if _, err := exec.LookPath("xz"); err != nil {
		t.Skip("xz not installed")
	}
	dir := t.TempDir()
	cacheRoot = func() string { d := filepath.Join(dir, "firmware"); os.MkdirAll(d, 0o755); return d }
	defer func() { cacheRoot = cacheDir }()
	img := fakeDiskImage(5 << 20)
	packed := writeVariants(t, t.TempDir(), img)["xz"]
	xzb, _ := os.ReadFile(packed)
	var srv *httptest.Server
	cut := true
	srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/os_list.json":
			json.NewEncoder(w).Encode(map[string]any{
				"imager": map[string]any{"devices": []any{map[string]any{"name": "Raspberry Pi 5", "tags": []string{"pi5-64bit"}}}},
				"os_list": []any{
					map[string]any{"name": "Erase", "url": "internal://format"},
					map[string]any{"name": "Other", "subitems": []any{map[string]any{
						"name": "Test OS", "url": srv.URL + "/test.img.xz", "devices": []string{"pi5-64bit"},
						"extract_size": itoa(len(img)), "extract_sha256": sha(img),
						"image_download_size": len(xzb), "image_download_sha256": sha(xzb)}}},
				}})
		case "/test.img.xz":
			if r.Header.Get("Range") == "" && cut { // first try: connection drops half way
				cut = false
				w.Header().Set("Content-Length", itoa(len(xzb)))
				w.Write(xzb[:len(xzb)/2])
				hj, _ := w.(http.Hijacker)
				c, _, _ := hj.Hijack()
				c.Close()
				return
			}
			http.ServeContent(w, r, "test.img.xz", time.Now(), bytes.NewReader(xzb))
		}
	}))
	defer srv.Close()
	oldC, oldI := httpClient, imgHTTP
	httpClient, imgHTTP = srv.Client(), srv.Client()
	defer func() { httpClient, imgHTTP = oldC, oldI }()

	l, _, err := fetchOSList(srv.URL + "/os_list.json")
	if err != nil || len(l.OSList) != 1 || l.OSList[0].Subitems[0].Name != "Test OS" || int(l.OSList[0].Subitems[0].ExtractSize) != len(img) {
		t.Fatalf("catalogue: %v %+v", err, l)
	}
	it := l.OSList[0].Subitems[0]
	var cancel = new(atomicBool)
	if _, err := download(it, &cancel.b, func(int64, int64) {}); err == nil {
		t.Fatal("cut-off download reported success")
	}
	p, err := download(it, &cancel.b, func(int64, int64) {})
	if err != nil {
		t.Fatalf("resumed download: %v", err)
	}

	// Flashie's endpoints, with a job waiting for its helper.
	mux := http.NewServeMux()
	imgRoutes(mux)
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	go http.Serve(ln, mux)
	base := "http://" + ln.Addr().String() + "/"
	disk := useTestDisk(t, 16)
	info, _ := sniffImage(p)
	job := &imgJob{final: make(chan progress, 1), token: "secret-token",
		spec: writeSpec{Device: disk, Source: info, Size: int64(it.ExtractSize), SHA256: it.ExtractSHA, Verify: true}}
	imgMu.Lock()
	imgCur = job
	imgMu.Unlock()
	defer func() { imgMu.Lock(); imgCur = nil; imgMu.Unlock() }()

	if code := runImgHelper(base, "wrong-token"); code == 0 {
		t.Fatal("helper accepted a bad token")
	}
	if code := runImgHelper(base, "secret-token"); code != 0 {
		t.Fatalf("helper exit %d", code)
	}
	select {
	case f := <-job.final:
		if !f.OK {
			t.Fatalf("helper failed: %s", f.Err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no result from the helper")
	}
	got, _ := os.ReadFile(disk)
	if !bytes.Equal(got[:len(img)], img) {
		t.Fatal("drive contents differ")
	}
	if code := runImgHelper(base, "secret-token"); code == 0 {
		t.Fatal("job could be fetched twice")
	}
	if r, _ := http.Get(base + "api/img/disks"); r != nil {
		var d struct{ Disks []Disk }
		json.NewDecoder(r.Body).Decode(&d)
		r.Body.Close()
		found := false
		for _, x := range d.Disks {
			found = found || x.ID == disk
		}
		if !found {
			t.Fatal("test drive not listed")
		}
	}
	_ = io.Discard
}

// .zst images: there's no zstd packer in Go, so this uses a file made by the
// real zstd library (shared/zstd/testdata) as the image.
func TestWriteZstd(t *testing.T) {
	p := filepath.Join("..", "..", "shared", "zstd", "testdata", "long27.zst")
	wb, _ := os.ReadFile(filepath.Join("..", "..", "shared", "zstd", "testdata", "plain.sha256"))
	want := strings.Fields(string(wb))
	info, err := sniffImage(p)
	if err != nil || info.Kind != "zstd" {
		t.Fatalf("sniff %+v %v", info, err)
	}
	disk := useTestDisk(t, 32)
	if _, err := runWrite(writeSpec{Device: disk, Source: info, SHA256: want[0], Verify: true}, func(progress) bool { return false }); err != nil {
		t.Fatal(err)
	}
	n, _ := strconv.Atoi(want[1])
	got, _ := os.ReadFile(disk)
	if sha(got[:n]) != want[0] {
		t.Fatal("drive contents differ")
	}
	bad, _ := sniffImage(filepath.Join("..", "..", "shared", "zstd", "testdata", "damaged.zst"))
	_, err = runWrite(writeSpec{Device: useTestDisk(t, 32), Source: bad, Verify: true}, func(progress) bool { return false })
	if err == nil || !strings.Contains(err.Error(), "damaged") {
		t.Fatalf("damaged file: %v", err)
	}
}
