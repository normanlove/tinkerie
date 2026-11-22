package launcher

// Installing Tinkerie on this computer, and updating it from GitHub.
//
// Tinkerie runs from wherever it was downloaded. The launcher offers to
// install it for the current user (no administrator rights): Windows gets
// Start-menu shortcuts for every app, an optional desktop shortcut and an
// entry in Settings > Apps; Linux and Raspberry Pi OS get a menu entry per app
// (self_windows.go, self_other.go).
//
// Updates: only when the person asks (the "Check for updates" button), or once
// a day if they turned on automatic checks, the launcher reads the list of
// releases from GitHub. If a newer one has a file for this system it can
// download it, check it against the SHA-256 digest GitHub lists for it, and
// put it in place of the running program. Nothing is sent about the person or
// the computer: it's an ordinary request for the public release list.

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
	"tinkerie/shared/droid"
)

// Where releases come from (tests point these at a local server).
var (
	releasesURL = "https://api.github.com/repos/normanlove/tinkerie/releases?per_page=20"
	releasePage = "https://github.com/normanlove/tinkerie/releases"
)

// selfExe is this program's own path, found once at start: after an update
// has replaced the file, the running program still knows where it lives.
var selfExe = func() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	if p, err := filepath.EvalSymlinks(exe); err == nil {
		exe = p
	}
	return exe
}()

// ---------------------------------------------------------------- settings

type selfSettings struct {
	AutoCheck       bool       `json:"autoCheck"`                 // check for updates once a day
	LastCheck       time.Time  `json:"lastCheck"`                 // when the last check worked
	Latest          *ghRelease `json:"latest,omitempty"`          // what it found
	NoInstallPrompt bool       `json:"noInstallPrompt,omitempty"` // "Not now" on the install banner
}

var settingsMu sync.Mutex

func settingsFile() string { return filepath.Join(dataDir(), "self.json") }

func loadSettings() selfSettings {
	var s selfSettings
	if b, err := os.ReadFile(settingsFile()); err == nil {
		json.Unmarshal(b, &s)
	}
	return s
}

func saveSettings(s selfSettings) {
	b, _ := json.MarshalIndent(s, "", "  ")
	tmp := settingsFile() + ".tmp"
	if os.WriteFile(tmp, b, 0o644) == nil {
		os.Rename(tmp, settingsFile())
	}
}

// ---------------------------------------------------------------- versions

type version struct {
	n   [3]int
	pre []string // after "-": "alpha", "beta.2"…; none for a full release
	ok  bool
}

func parseVersion(s string) version {
	s = strings.TrimPrefix(strings.TrimSpace(s), "v")
	var v version
	main, pre, hasPre := strings.Cut(s, "-")
	if i := strings.IndexByte(main, '+'); i >= 0 {
		main = main[:i]
	}
	parts := strings.Split(main, ".")
	if len(parts) == 0 || len(parts) > 3 {
		return v
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return v
		}
		v.n[i] = n
	}
	if hasPre {
		pre, _, _ = strings.Cut(pre, "+")
		if pre == "" {
			return v
		}
		v.pre = strings.Split(pre, ".")
	}
	v.ok = true
	return v
}

// compareVersions returns -1, 0 or 1 as a is older than, the same as, or newer
// than b, by the semantic versioning rules (0.33.0-alpha < 0.33.0-beta < 0.33.0).
func compareVersions(a, b string) int {
	va, vb := parseVersion(a), parseVersion(b)
	for i := range va.n {
		if va.n[i] != vb.n[i] {
			return sign(va.n[i] - vb.n[i])
		}
	}
	switch {
	case len(va.pre) == 0 && len(vb.pre) == 0:
		return 0
	case len(va.pre) == 0:
		return 1
	case len(vb.pre) == 0:
		return -1
	}
	for i := 0; i < len(va.pre) && i < len(vb.pre); i++ {
		x, y := va.pre[i], vb.pre[i]
		nx, ex := strconv.Atoi(x)
		ny, ey := strconv.Atoi(y)
		switch {
		case ex == nil && ey == nil:
			if nx != ny {
				return sign(nx - ny)
			}
		case ex == nil: // numbers come before words
			return -1
		case ey == nil:
			return 1
		case x != y:
			return sign(strings.Compare(x, y))
		}
	}
	return sign(len(va.pre) - len(vb.pre))
}

func sign(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	}
	return 0
}

// ---------------------------------------------------------------- releases

type ghAsset struct {
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	Digest string `json:"digest"` // "sha256:<hex>"
	URL    string `json:"browser_download_url"`
}

type ghRelease struct {
	Tag        string    `json:"tag_name"`
	Draft      bool      `json:"draft"`
	Prerelease bool      `json:"prerelease"`
	Page       string    `json:"html_url"`
	Assets     []ghAsset `json:"assets"`
}

func (r *ghRelease) Version() string { return strings.TrimPrefix(r.Tag, "v") }

// pickLatest finds the newest usable release. Someone on an alpha or beta also
// gets alphas and betas; someone on a full release only gets full releases.
func pickLatest(rels []ghRelease, current string) *ghRelease {
	wantPre := len(parseVersion(current).pre) > 0
	var best *ghRelease
	for i := range rels {
		r := &rels[i]
		if r.Draft || !parseVersion(r.Version()).ok || (r.Prerelease && !wantPre) {
			continue
		}
		if best == nil || compareVersions(r.Version(), best.Version()) > 0 {
			best = r
		}
	}
	return best
}

func fetchReleases(ctx context.Context) ([]ghRelease, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", releasesURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "Tinkerie/"+Version)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("couldn't reach GitHub (%v)", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub answered %s", resp.Status)
	}
	var rels []ghRelease
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&rels); err != nil {
		return nil, fmt.Errorf("GitHub's answer wasn't readable (%v)", err)
	}
	return rels, nil
}

// assetFor names the release file for a system, and whether it's a .tar.gz
// holding the program (Linux, Pi) or the program itself (Windows).
func assetFor(ver, goos, goarch string) (name string, packed bool) {
	switch goos + "/" + goarch {
	case "windows/amd64":
		return "Tinkerie-" + ver + "-windows.exe", false
	case "linux/amd64":
		return "Tinkerie-" + ver + "-linux-x86_64.tar.gz", true
	case "linux/arm64":
		return "Tinkerie-" + ver + "-pi-64bit.tar.gz", true
	case "linux/arm":
		return "Tinkerie-" + ver + "-pi-32bit.tar.gz", true
	}
	return "", false
}

// canReplaceSelf says whether this copy can update itself, and if not, why.
func canReplaceSelf() (bool, string) {
	switch {
	case droid.Active() || runtime.GOOS == "android":
		return false, "On Android, download the new .apk from the release page and open it."
	case runtime.GOOS == "darwin":
		return false, "On a Mac, download the new version from the release page and put it in place of Tinkerie in Applications."
	case selfExe == "":
		return false, "Tinkerie couldn't find its own program file."
	}
	if n, _ := assetFor("0", runtime.GOOS, runtime.GOARCH); n == "" {
		return false, "There's no ready-made Tinkerie for this system: build it from the source."
	}
	if !writable(filepath.Dir(selfExe)) {
		return false, "Tinkerie can't change its own folder (" + filepath.Dir(selfExe) + "). Install it, or download the new version from the release page."
	}
	return true, ""
}

func writable(dir string) bool {
	f, err := os.CreateTemp(dir, ".tinkerie-check-*")
	if err != nil {
		return false
	}
	f.Close()
	os.Remove(f.Name())
	return true
}

// ---------------------------------------------------------------- updating

// updateState is what the page shows while an update downloads.
type updateState struct {
	State string `json:"state"` // "", "downloading", "checking", "done", "error"
	Got   int64  `json:"got"`
	Total int64  `json:"total"`
	Msg   string `json:"msg,omitempty"`
	To    string `json:"to,omitempty"` // the version it updated to
}

var (
	updMu sync.Mutex
	upd   updateState
)

func setUpd(f func(*updateState)) { updMu.Lock(); f(&upd); updMu.Unlock() }

// applyUpdate downloads rel's file for this system, checks it, and puts the
// new program in place of exe. Running apps keep running the old one until
// they're closed.
func applyUpdate(ctx context.Context, rel *ghRelease, exe string) error {
	name, packed := assetFor(rel.Version(), runtime.GOOS, runtime.GOARCH)
	var a *ghAsset
	for i := range rel.Assets {
		if rel.Assets[i].Name == name {
			a = &rel.Assets[i]
		}
	}
	if a == nil {
		return fmt.Errorf("Tinkerie %s has no file for this system (%s)", rel.Version(), name)
	}
	want, ok := strings.CutPrefix(a.Digest, "sha256:")
	if !ok || len(want) != 64 {
		return fmt.Errorf("GitHub doesn't list a SHA-256 for %s, so it can't be checked: download it from the release page instead", name)
	}
	if a.Size <= 0 || a.Size > 1<<30 {
		return fmt.Errorf("%s has an unexpected size (%d bytes)", name, a.Size)
	}
	dir := filepath.Dir(exe)
	dl, err := os.CreateTemp(dir, ".tinkerie-download-*")
	if err != nil {
		return fmt.Errorf("can't save the download next to Tinkerie: %v", err)
	}
	defer os.Remove(dl.Name())
	defer dl.Close()

	setUpd(func(u *updateState) { *u = updateState{State: "downloading", Total: a.Size} })
	req, err := http.NewRequestWithContext(ctx, "GET", a.URL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Tinkerie/"+Version)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("the download didn't start (%v)", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("the download failed: %s", resp.Status)
	}
	h := sha256.New()
	body := io.LimitReader(resp.Body, a.Size+1)
	var got int64
	buf := make([]byte, 256<<10)
	for {
		n, rerr := body.Read(buf)
		if n > 0 {
			if _, err := dl.Write(buf[:n]); err != nil {
				return fmt.Errorf("couldn't save the download: %v", err)
			}
			h.Write(buf[:n])
			got += int64(n)
			setUpd(func(u *updateState) { u.Got = got })
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return fmt.Errorf("the download stopped (%v)", rerr)
		}
	}
	setUpd(func(u *updateState) { u.State = "checking" })
	if got != a.Size {
		return fmt.Errorf("the download is the wrong size (%d bytes, expected %d)", got, a.Size)
	}
	if sum := hex.EncodeToString(h.Sum(nil)); !strings.EqualFold(sum, want) {
		return errors.New("the download doesn't match the SHA-256 GitHub lists for it, so it wasn't used")
	}

	next := exe + ".new"
	os.Remove(next)
	if packed {
		if _, err := dl.Seek(0, io.SeekStart); err != nil {
			return err
		}
		if err := extractProgram(dl, next); err != nil {
			return err
		}
	} else {
		dl.Close()
		if err := os.Rename(dl.Name(), next); err != nil {
			return err
		}
	}
	if err := looksRunnable(next); err != nil {
		os.Remove(next)
		return err
	}
	if err := replaceExe(exe, next); err != nil {
		os.Remove(next)
		return fmt.Errorf("couldn't put the new version in place: %v", err)
	}
	log.Printf("updated %s to %s", exe, rel.Version())
	return nil
}

// extractProgram copies the "tinkerie" program out of a release .tar.gz
// (Tinkerie-<version>-<system>/tinkerie) into dst.
func extractProgram(r io.Reader, dst string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("the download isn't a .tar.gz: %v", err)
	}
	tr := tar.NewReader(gz)
	for {
		hd, err := tr.Next()
		if err == io.EOF {
			return errors.New("the download has no tinkerie program in it")
		}
		if err != nil {
			return fmt.Errorf("the download is damaged: %v", err)
		}
		p := path.Clean(hd.Name)
		if hd.Typeflag != tar.TypeReg || path.Base(p) != "tinkerie" || strings.Count(p, "/") != 1 || strings.HasPrefix(p, "..") {
			continue
		}
		f, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o755)
		if err != nil {
			return err
		}
		_, err = io.Copy(f, io.LimitReader(tr, 1<<30))
		if cerr := f.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			os.Remove(dst)
			return err
		}
		return nil
	}
}

// looksRunnable checks the new file is a program for this kind of system.
func looksRunnable(p string) error {
	f, err := os.Open(p)
	if err != nil {
		return err
	}
	defer f.Close()
	head := make([]byte, 4)
	if _, err := io.ReadFull(f, head); err != nil {
		return errors.New("the new program is empty")
	}
	want := []byte("\x7fELF")
	if runtime.GOOS == "windows" {
		want = []byte("MZ")
	}
	if !bytes.HasPrefix(head, want) {
		return errors.New("the download isn't a program for this system")
	}
	return nil
}

// pngToICO wraps a PNG in an .ico file (Windows Vista and later read these).
func pngToICO(png []byte) []byte {
	if len(png) < 24 || string(png[1:4]) != "PNG" {
		return nil
	}
	w, h := binary.BigEndian.Uint32(png[16:20]), binary.BigEndian.Uint32(png[20:24])
	dim := func(n uint32) byte {
		if n >= 256 {
			return 0 // 0 means 256
		}
		return byte(n)
	}
	b := make([]byte, 22, 22+len(png))
	binary.LittleEndian.PutUint16(b[2:], 1) // icon
	binary.LittleEndian.PutUint16(b[4:], 1) // one image
	b[6], b[7] = dim(w), dim(h)
	binary.LittleEndian.PutUint16(b[10:], 1)  // planes
	binary.LittleEndian.PutUint16(b[12:], 32) // bits per pixel
	binary.LittleEndian.PutUint32(b[14:], uint32(len(png)))
	binary.LittleEndian.PutUint32(b[18:], 22)
	return append(b, png...)
}

// copyFile copies src to dst (made fresh), as a program.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// cleanUpOld removes what an earlier update left behind (on Windows the old
// program can only go once nothing runs it any more).
func cleanUpOld() {
	if selfExe == "" {
		return
	}
	os.Remove(selfExe + ".old")
	os.Remove(selfExe + ".new")
	if m, _ := filepath.Glob(filepath.Join(filepath.Dir(selfExe), ".tinkerie-download-*")); len(m) > 0 {
		for _, p := range m {
			if st, err := os.Stat(p); err == nil && time.Since(st.ModTime()) > time.Hour {
				os.Remove(p)
			}
		}
	}
}

// ---------------------------------------------------------------- installing

// installMark is kept in the install folder: which version set up the
// shortcuts, and the person's choices, so a newer version can redo them.
type installMark struct {
	Version string `json:"version"`
	Desktop bool   `json:"desktop"`
}

func readMark(dir string) (installMark, bool) {
	var m installMark
	b, err := os.ReadFile(filepath.Join(dir, "installed.json"))
	if err != nil || json.Unmarshal(b, &m) != nil {
		return m, false
	}
	return m, true
}

func writeMark(dir string, m installMark) error {
	b, _ := json.MarshalIndent(m, "", "  ")
	return os.WriteFile(filepath.Join(dir, "installed.json"), b, 0o644)
}

func samePath(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	a, b = filepath.Clean(a), filepath.Clean(b)
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// isInstalled says whether this copy is the installed one.
func isInstalled() bool {
	ok, _ := installSupported()
	return ok && samePath(selfExe, installedExe())
}

// writeIcons puts the launcher's icons in the install folder for shortcuts.
func writeIcons(dir string) error {
	if err := os.MkdirAll(filepath.Join(dir, "icons"), 0o755); err != nil {
		return err
	}
	b, err := fs.ReadFile(webFS, "web/tinkerie.png")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "tinkerie.png"), b, 0o644); err != nil {
		return err
	}
	for _, a := range Apps {
		b, err := fs.ReadFile(webFS, "web/icons/"+a.Icon)
		if err != nil {
			continue
		}
		if err := os.WriteFile(filepath.Join(dir, "icons", a.Icon), b, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// install copies this program into the install folder and makes shortcuts.
// It returns a note for the person if something only partly worked.
func install(desktop bool) (note string, err error) {
	if ok, why := installSupported(); !ok {
		return "", errors.New(why)
	}
	dst := installedExe()
	dir := filepath.Dir(dst)
	if samePath(selfExe, dst) {
		return "", errors.New("this copy is already the installed one")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	next := dst + ".new"
	os.Remove(next)
	if err := copyFile(selfExe, next); err != nil {
		return "", fmt.Errorf("couldn't copy Tinkerie to %s: %v", dir, err)
	}
	if _, err := os.Stat(dst); err == nil {
		err = replaceExe(dst, next) // an older installed copy may be running
	} else {
		err = os.Rename(next, dst)
	}
	if err != nil {
		os.Remove(next)
		return "", fmt.Errorf("couldn't put Tinkerie in %s: %v", dir, err)
	}
	if err := writeIcons(dir); err != nil {
		return "", err
	}
	m := installMark{Version: Version, Desktop: desktop}
	note = register(dst, m)
	if err := writeMark(dir, m); err != nil {
		return note, err
	}
	log.Printf("installed in %s", dir)
	return note, nil
}

// refreshInstall redoes the shortcuts when a newer version first starts from
// the install folder (after an update, or when an app was added).
func refreshInstall() {
	if !isInstalled() {
		return
	}
	dir := filepath.Dir(selfExe)
	m, ok := readMark(dir)
	if ok && m.Version == Version {
		return
	}
	m.Version = Version
	writeIcons(dir)
	if note := register(selfExe, m); note != "" {
		log.Println("refreshing shortcuts:", note)
	}
	writeMark(dir, m)
}

// Uninstall removes the installed copy, its shortcuts and its entries. The
// person's own files and settings are kept. With ask, it asks first (Windows
// shows a dialog; "Tinkerie --uninstall" from Settings > Apps comes here).
func Uninstall(ask bool) {
	if ok, why := installSupported(); !ok {
		tell(why)
		return
	}
	if !installPresent() {
		tell("Tinkerie isn't installed on this computer, so there's nothing to remove.")
		return
	}
	if ask && !confirm("Remove Tinkerie from this computer?\n\nClose any Tinkerie apps first. Your designs, files and settings are kept.") {
		return
	}
	if err := uninstall(); err != nil {
		tell("Tinkerie couldn't be fully removed: " + err.Error())
		return
	}
	if ask {
		tell("Tinkerie has been removed. Your files and settings are still in Documents.")
	}
}

func installPresent() bool {
	_, err := os.Stat(installedExe())
	return err == nil
}

// ---------------------------------------------------------------- the page's API

// selfInfo is what /api/self tells the page. It never uses the network.
type selfInfo struct {
	Version          string     `json:"version"`
	Installed        bool       `json:"installed"`        // this copy is the installed one
	InstallOK        bool       `json:"installOK"`        // installing is possible here
	InstallWhy       string     `json:"installWhy"`       // why not
	InstallDir       string     `json:"installDir"`       // where it goes
	InstalledVersion string     `json:"installedVersion"` // another copy already installed
	Running          string     `json:"running"`          // the folder this copy runs from
	Desktop          bool       `json:"desktopOption"`    // offer a desktop shortcut
	Prompt           bool       `json:"prompt"`           // show the install banner
	AutoCheck        bool       `json:"autoCheck"`
	Update           *updateMsg `json:"update,omitempty"` // the last check's result
}

type updateMsg struct {
	Current  string    `json:"current"`
	Latest   string    `json:"latest"`
	Newer    bool      `json:"newer"`
	Page     string    `json:"page"`
	CanApply bool      `json:"canApply"`
	Why      string    `json:"why,omitempty"`
	Checked  time.Time `json:"checked"`
}

func updateMsgFor(rel *ghRelease, checked time.Time) *updateMsg {
	m := &updateMsg{Current: Version, Page: releasePage, Checked: checked}
	if rel == nil {
		m.Latest = Version
		return m
	}
	m.Latest = rel.Version()
	m.Newer = compareVersions(rel.Version(), Version) > 0
	if strings.HasPrefix(rel.Page, releasePage+"/") {
		m.Page = rel.Page
	}
	if m.Newer {
		m.CanApply, m.Why = canReplaceSelf()
		if m.CanApply {
			if n, _ := assetFor(rel.Version(), runtime.GOOS, runtime.GOARCH); !hasAsset(rel, n) {
				m.CanApply, m.Why = false, "This release has no file for this system yet."
			}
		}
	}
	return m
}

func hasAsset(rel *ghRelease, name string) bool {
	for _, a := range rel.Assets {
		if a.Name == name {
			return true
		}
	}
	return false
}

func getSelfInfo() selfInfo {
	settingsMu.Lock()
	s := loadSettings()
	settingsMu.Unlock()
	ok, why := installSupported()
	in := selfInfo{Version: Version, InstallOK: ok, InstallWhy: why, AutoCheck: s.AutoCheck, Desktop: runtime.GOOS == "windows"}
	if selfExe != "" {
		in.Running = filepath.Dir(selfExe)
	}
	if ok {
		in.InstallDir = filepath.Dir(installedExe())
		in.Installed = isInstalled()
		if !in.Installed && installPresent() {
			if m, ok := readMark(in.InstallDir); ok {
				in.InstalledVersion = m.Version
			} else {
				in.InstalledVersion = "?"
			}
		}
		in.Prompt = !in.Installed && !s.NoInstallPrompt && in.InstalledVersion == ""
	}
	if s.Latest != nil {
		in.Update = updateMsgFor(s.Latest, s.LastCheck)
	}
	return in
}

// checkForUpdate asks GitHub for the newest release (if auto, only once a day).
func checkForUpdate(ctx context.Context, auto bool) (*updateMsg, error) {
	settingsMu.Lock()
	s := loadSettings()
	settingsMu.Unlock()
	if auto {
		if !s.AutoCheck {
			return nil, nil
		}
		if time.Since(s.LastCheck) < 20*time.Hour {
			return updateMsgFor(s.Latest, s.LastCheck), nil
		}
	}
	rels, err := fetchReleases(ctx)
	if err != nil {
		return nil, err
	}
	rel := pickLatest(rels, Version)
	now := time.Now()
	settingsMu.Lock()
	s = loadSettings()
	s.LastCheck, s.Latest = now, rel
	saveSettings(s)
	settingsMu.Unlock()
	return updateMsgFor(rel, now), nil
}

// selfRoutes adds the install and update API to the launcher's server.
// post checks a request really comes from the launcher's own page.
func selfRoutes(mux *http.ServeMux, post func(http.ResponseWriter, *http.Request) bool, quit func()) {
	reply := func(w http.ResponseWriter, v any) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(v)
	}
	fail := func(w http.ResponseWriter, code int, msg string) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(map[string]string{"error": msg})
	}
	mux.HandleFunc("/api/self", func(w http.ResponseWriter, r *http.Request) {
		reply(w, getSelfInfo())
	})
	mux.HandleFunc("/api/self/settings", func(w http.ResponseWriter, r *http.Request) {
		if !post(w, r) {
			return
		}
		var req struct {
			AutoCheck       *bool `json:"autoCheck"`
			NoInstallPrompt *bool `json:"noInstallPrompt"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		settingsMu.Lock()
		s := loadSettings()
		if req.AutoCheck != nil {
			s.AutoCheck = *req.AutoCheck
		}
		if req.NoInstallPrompt != nil {
			s.NoInstallPrompt = *req.NoInstallPrompt
		}
		saveSettings(s)
		settingsMu.Unlock()
		reply(w, getSelfInfo())
	})
	mux.HandleFunc("/api/self/install", func(w http.ResponseWriter, r *http.Request) {
		if !post(w, r) {
			return
		}
		var req struct{ Desktop bool }
		json.NewDecoder(r.Body).Decode(&req)
		note, err := install(req.Desktop)
		if err != nil {
			fail(w, 500, err.Error())
			return
		}
		reply(w, map[string]any{"ok": true, "note": note, "dir": filepath.Dir(installedExe())})
	})
	mux.HandleFunc("/api/self/uninstall", func(w http.ResponseWriter, r *http.Request) {
		if !post(w, r) {
			return
		}
		if !isInstalled() && !installPresent() {
			fail(w, 409, "Tinkerie isn't installed.")
			return
		}
		if err := uninstall(); err != nil {
			fail(w, 500, err.Error())
			return
		}
		reply(w, map[string]any{"ok": true})
		if isInstalledPath(selfExe) {
			go func() { time.Sleep(300 * time.Millisecond); quit() }()
		}
	})
	// open: start another copy's launcher and close this one: "installed" (the
	// installed copy) or "self" (this program again, e.g. after an update).
	mux.HandleFunc("/api/self/open", func(w http.ResponseWriter, r *http.Request) {
		if !post(w, r) {
			return
		}
		var req struct{ Which string }
		json.NewDecoder(r.Body).Decode(&req)
		exe := selfExe
		if req.Which == "installed" {
			exe = installedExe()
		}
		if exe == "" {
			fail(w, 500, "Tinkerie couldn't find its program.")
			return
		}
		cmd := launcherCommand(exe)
		if err := cmd.Start(); err != nil {
			fail(w, 500, err.Error())
			return
		}
		go cmd.Wait()
		reply(w, map[string]any{"ok": true})
		go func() { time.Sleep(300 * time.Millisecond); quit() }()
	})
	mux.HandleFunc("/api/self/page", func(w http.ResponseWriter, r *http.Request) {
		if !post(w, r) {
			return
		}
		u := releasePage
		settingsMu.Lock()
		if s := loadSettings(); s.Latest != nil && strings.HasPrefix(s.Latest.Page, releasePage+"/") {
			u = s.Latest.Page
		}
		settingsMu.Unlock()
		openURL(u)
		reply(w, map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/update/check", func(w http.ResponseWriter, r *http.Request) {
		if !post(w, r) {
			return
		}
		var req struct{ Auto bool }
		json.NewDecoder(r.Body).Decode(&req)
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		m, err := checkForUpdate(ctx, req.Auto)
		if err != nil {
			fail(w, 502, err.Error())
			return
		}
		reply(w, map[string]any{"update": m})
	})
	mux.HandleFunc("/api/update/apply", func(w http.ResponseWriter, r *http.Request) {
		if !post(w, r) {
			return
		}
		if ok, why := canReplaceSelf(); !ok {
			fail(w, 409, why)
			return
		}
		settingsMu.Lock()
		rel := loadSettings().Latest
		settingsMu.Unlock()
		if rel == nil || compareVersions(rel.Version(), Version) <= 0 {
			fail(w, 409, "There's no newer version to install: check for updates first.")
			return
		}
		updMu.Lock()
		busy := upd.State == "downloading" || upd.State == "checking"
		done := upd.State == "done"
		if !busy && !done {
			upd = updateState{State: "downloading"}
		}
		updMu.Unlock()
		if busy {
			fail(w, 409, "An update is already downloading.")
			return
		}
		if done {
			fail(w, 409, "Tinkerie is already updated: restart it to use the new version.")
			return
		}
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
			defer cancel()
			err := applyUpdate(ctx, rel, selfExe)
			setUpd(func(u *updateState) {
				if err != nil {
					u.State, u.Msg = "error", err.Error()
				} else {
					u.State, u.To = "done", rel.Version()
				}
			})
		}()
		reply(w, map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/update/status", func(w http.ResponseWriter, r *http.Request) {
		updMu.Lock()
		u := upd
		updMu.Unlock()
		reply(w, u)
	})
}

func isInstalledPath(p string) bool {
	ok, _ := installSupported()
	return ok && samePath(p, installedExe())
}
