package launcher

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want int
	}{
		{"0.32.1-alpha", "0.32.1-alpha", 0},
		{"v0.32.1-alpha", "0.32.1-alpha", 0},
		{"0.33.0-alpha", "0.32.1-alpha", 1},
		{"0.32.10-alpha", "0.32.9-alpha", 1},
		{"1.0.0", "0.99.99", 1},
		{"0.33.0", "0.33.0-alpha", 1},
		{"0.33.0-beta", "0.33.0-alpha", 1},
		{"0.33.0-alpha.2", "0.33.0-alpha", 1},
		{"0.33.0-alpha.10", "0.33.0-alpha.9", 1},
		{"0.33.0-rc.1", "0.33.0-beta.5", 1},
		{"0.32.0-alpha", "0.32.1-alpha", -1},
	} {
		if got := compareVersions(c.a, c.b); got != c.want {
			t.Errorf("compareVersions(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
		if got := compareVersions(c.b, c.a); got != -c.want {
			t.Errorf("compareVersions(%q, %q) = %d, want %d", c.b, c.a, got, -c.want)
		}
	}
	for _, bad := range []string{"", "next", "1.2.3.4", "1.x", "1.2-"} {
		if parseVersion(bad).ok {
			t.Errorf("parseVersion(%q) should fail", bad)
		}
	}
}

func TestPickLatest(t *testing.T) {
	rels := []ghRelease{
		{Tag: "v0.32.1-alpha", Prerelease: true},
		{Tag: "v0.34.0-alpha", Draft: true}, // drafts never count
		{Tag: "v0.33.0-alpha", Prerelease: true},
		{Tag: "nightly", Prerelease: true},
		{Tag: "v0.31.0"},
	}
	if r := pickLatest(rels, "0.32.1-alpha"); r == nil || r.Version() != "0.33.0-alpha" {
		t.Errorf("alpha user: got %+v, want 0.33.0-alpha", r)
	}
	// Someone on a full release isn't offered alphas.
	if r := pickLatest(rels, "0.30.0"); r == nil || r.Version() != "0.31.0" {
		t.Errorf("full-release user: got %+v, want 0.31.0", r)
	}
	if r := pickLatest(nil, "0.30.0"); r != nil {
		t.Errorf("no releases: got %+v", r)
	}
}

// Every system with a ready-made build has a file in the release, named the
// way package.sh and the release workflow name them.
func TestAssetNames(t *testing.T) {
	for sys, want := range map[string]string{
		"windows/amd64": "Tinkerie-1.2.3-windows.exe",
		"linux/amd64":   "Tinkerie-1.2.3-linux-x86_64.tar.gz",
		"linux/arm64":   "Tinkerie-1.2.3-pi-64bit.tar.gz",
		"linux/arm":     "Tinkerie-1.2.3-pi-32bit.tar.gz",
		"darwin/arm64":  "",
		"windows/arm64": "",
	} {
		os_, arch, _ := strings.Cut(sys, "/")
		if got, _ := assetFor("1.2.3", os_, arch); got != want {
			t.Errorf("%s: %q, want %q", sys, got, want)
		}
	}
	b, err := os.ReadFile("../package.sh")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"linux-x86_64", "pi-64bit", "pi-32bit"} {
		if !strings.Contains(string(b), p) {
			t.Errorf("package.sh no longer makes %s", p)
		}
	}
	wf, err := os.ReadFile("../.github/workflows/release.yml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(wf), `"Tinkerie-$v-windows.exe"`) {
		t.Error("the release workflow no longer publishes Tinkerie-<version>-windows.exe")
	}
}

func TestPNGToICO(t *testing.T) {
	png, err := webFS.ReadFile("web/icons/scopie.png")
	if err != nil {
		t.Fatal(err)
	}
	ico := pngToICO(png)
	if len(ico) != 22+len(png) || ico[2] != 1 || ico[4] != 1 || ico[6] != 0 || !bytes.Equal(ico[22:], png) {
		t.Errorf("bad .ico header % x", ico[:22])
	}
	if pngToICO([]byte("<svg/>")) != nil {
		t.Error("an SVG isn't a PNG")
	}
}

// fakeRelease serves a release list and one file, like GitHub does.
func fakeRelease(t *testing.T, ver string, file []byte, digest string) *httptest.Server {
	t.Helper()
	name, _ := assetFor(ver, runtime.GOOS, runtime.GOARCH)
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/releases":
			json.NewEncoder(w).Encode([]ghRelease{{
				Tag: "v" + ver, Prerelease: true, Page: releasePage + "/tag/v" + ver,
				Assets: []ghAsset{{Name: name, Size: int64(len(file)), Digest: digest, URL: srv.URL + "/dl/" + name}},
			}})
		case "/dl/" + name:
			w.Write(file)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	old := releasesURL
	releasesURL = srv.URL + "/releases"
	t.Cleanup(func() { releasesURL = old })
	return srv
}

// releaseFile makes what the release holds for this system: the program on
// its own (Windows) or in a .tar.gz with the rest of the package (Linux).
func releaseFile(t *testing.T, ver string, prog []byte) []byte {
	if runtime.GOOS == "windows" {
		return prog
	}
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	tw := tar.NewWriter(gz)
	top := "Tinkerie-" + ver + "-linux-x86_64/"
	for _, f := range []struct {
		name string
		body []byte
	}{{top + "install.sh", []byte("#!/bin/sh\n")}, {top + "source/tinkerie", []byte("not this one")}, {top + "tinkerie", prog}} {
		tw.WriteHeader(&tar.Header{Name: f.name, Mode: 0o755, Size: int64(len(f.body)), Typeflag: tar.TypeReg})
		tw.Write(f.body)
	}
	tw.Close()
	gz.Close()
	return b.Bytes()
}

func sha(b []byte) string { s := sha256.Sum256(b); return "sha256:" + hex.EncodeToString(s[:]) }

func progHead() string {
	if runtime.GOOS == "windows" {
		return "MZ"
	}
	return "\x7fELF"
}

func TestUpdateEndToEnd(t *testing.T) {
	if n, _ := assetFor("0", runtime.GOOS, runtime.GOARCH); n == "" {
		t.Skip("no release file for this system")
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("APPDATA", t.TempDir())
	dir := t.TempDir()
	exe := filepath.Join(dir, "tinkerie")
	os.WriteFile(exe, []byte(progHead()+"old program"), 0o755)
	newProg := []byte(progHead() + "new program")
	file := releaseFile(t, "9.0.0-alpha", newProg)
	fakeRelease(t, "9.0.0-alpha", file, sha(file))

	m, err := checkForUpdate(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if !m.Newer || m.Latest != "9.0.0-alpha" || !strings.HasSuffix(m.Page, "/tag/v9.0.0-alpha") {
		t.Fatalf("check: %+v", m)
	}
	// The result is kept, so the page shows it without asking again.
	if s := loadSettings(); s.Latest == nil || s.Latest.Version() != "9.0.0-alpha" {
		t.Fatalf("not saved: %+v", s)
	}
	if err := applyUpdate(context.Background(), loadSettings().Latest, exe); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(exe)
	if !bytes.Equal(got, newProg) {
		t.Fatalf("program is %q, want the new one", got)
	}
	if st, _ := os.Stat(exe); runtime.GOOS != "windows" && st.Mode().Perm()&0o100 == 0 {
		t.Errorf("new program isn't executable: %v", st.Mode())
	}
	if left, _ := filepath.Glob(filepath.Join(dir, ".tinkerie-download-*")); len(left) > 0 {
		t.Errorf("download left behind: %v", left)
	}
}

func TestUpdateRefusesBadDownloads(t *testing.T) {
	if n, _ := assetFor("0", runtime.GOOS, runtime.GOARCH); n == "" {
		t.Skip("no release file for this system")
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("APPDATA", t.TempDir())
	good := releaseFile(t, "9.0.0-alpha", []byte(progHead()+"new"))
	for _, c := range []struct {
		name   string
		file   []byte
		digest string
		want   string
	}{
		{"tampered", append([]byte(nil), good...), sha(append(append([]byte(nil), good...), 'x')), "doesn't match the SHA-256"},
		{"no digest", good, "", "doesn't list a SHA-256"},
		{"not a program", releaseFile(t, "9.0.0-alpha", []byte("#!/bin/sh\nrm -rf ~\n")), "", "isn't a program"},
	} {
		t.Run(c.name, func(t *testing.T) {
			if c.digest == "" && c.want != "doesn't list a SHA-256" {
				c.digest = sha(c.file)
			}
			dir := t.TempDir()
			exe := filepath.Join(dir, "tinkerie")
			old := []byte(progHead() + "old program")
			os.WriteFile(exe, old, 0o755)
			fakeRelease(t, "9.0.0-alpha", c.file, c.digest)
			if _, err := checkForUpdate(context.Background(), false); err != nil {
				t.Fatal(err)
			}
			err := applyUpdate(context.Background(), loadSettings().Latest, exe)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("error %v, want %q", err, c.want)
			}
			if got, _ := os.ReadFile(exe); !bytes.Equal(got, old) {
				t.Error("the old program was changed")
			}
			if left, _ := filepath.Glob(filepath.Join(dir, "*")); len(left) != 1 {
				t.Errorf("files left behind: %v", left)
			}
		})
	}
}

func TestAutoCheckOnlyWhenAllowed(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("APPDATA", t.TempDir())
	asked := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { asked++; w.Write([]byte("[]")) }))
	defer srv.Close()
	old := releasesURL
	releasesURL = srv.URL
	defer func() { releasesURL = old }()

	if m, err := checkForUpdate(context.Background(), true); m != nil || err != nil || asked != 0 {
		t.Fatalf("automatic check with it turned off asked GitHub (%d, %v, %v)", asked, m, err)
	}
	s := loadSettings()
	s.AutoCheck = true
	saveSettings(s)
	checkForUpdate(context.Background(), true)
	checkForUpdate(context.Background(), true) // the same day: no second request
	if asked != 1 {
		t.Fatalf("asked GitHub %d times, want 1", asked)
	}
	checkForUpdate(context.Background(), false) // the button always asks
	if asked != 2 {
		t.Fatalf("asked GitHub %d times, want 2", asked)
	}
}

// Only the launcher's own page may change anything.
func TestSelfAPIGuard(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("APPDATA", t.TempDir())
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()
	host := strings.TrimPrefix(srv.URL, "http://")
	post := func(w http.ResponseWriter, r *http.Request) bool {
		if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, "use POST with JSON", http.StatusMethodNotAllowed)
			return false
		}
		if r.Host != host {
			http.Error(w, "wrong host", http.StatusForbidden)
			return false
		}
		return true
	}
	selfRoutes(mux, post, func() { t.Error("quit called") })

	send := func(ctype, hostHeader string) int {
		req, _ := http.NewRequest("POST", srv.URL+"/api/self/settings", strings.NewReader(`{"autoCheck":true}`))
		req.Header.Set("Content-Type", ctype)
		if hostHeader != "" {
			req.Host = hostHeader
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return resp.StatusCode
	}
	if c := send("text/plain", ""); c != 405 {
		t.Errorf("form-style post: %d", c)
	}
	if c := send("application/json", "evil.example:80"); c != 403 {
		t.Errorf("another host name: %d", c)
	}
	if loadSettings().AutoCheck {
		t.Fatal("a refused request changed the settings")
	}
	if c := send("application/json", ""); c != 200 || !loadSettings().AutoCheck {
		t.Errorf("the page's own request: %d", c)
	}
}
