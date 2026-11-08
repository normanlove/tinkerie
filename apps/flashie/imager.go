package flashie

// "Write image" tab: puts an OS image (Raspberry Pi OS and friends, or any
// .img) onto an SD card or USB drive, like Raspberry Pi Imager. The OS list
// is Raspberry Pi's own catalogue, the one Raspberry Pi Imager uses.

import (
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const osListURL = "https://downloads.raspberrypi.com/os_list_imagingutility_v4.json"

// imgHTTP has no overall timeout: images are large.
var imgHTTP = &http.Client{}

func sleepMS(n int) { time.Sleep(time.Duration(n) * time.Millisecond) }

func imagesDir() string {
	d := filepath.Join(filepath.Dir(cacheRoot()), "images")
	os.MkdirAll(d, 0o755)
	return d
}

// ------------------------------------------------------------ test drive ---

// FLASHIE_IMG_TESTDISK=/path/file:SIZE_MB adds a pretend drive backed by a
// file, for trying the writer out (and for the tests) without a real card.
func testDiskSpec() (string, int64) {
	v := os.Getenv("FLASHIE_IMG_TESTDISK")
	if v == "" {
		return "", 0
	}
	size := int64(64)
	if i := strings.LastIndex(v, ":"); i > 1 { // keep C:\ paths whole
		if n, err := strconv.ParseInt(v[i+1:], 10, 64); err == nil {
			v, size = v[:i], n
		}
	}
	return v, size << 20
}

type fileDisk struct{ *os.File }

func testDiskFile(d Disk) (rawDisk, bool) {
	p, size := testDiskSpec()
	if p == "" || d.ID != p {
		return nil, false
	}
	f, err := os.OpenFile(p, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return nil, false
	}
	if st, _ := f.Stat(); st != nil && st.Size() < size {
		f.Truncate(size)
	}
	return fileDisk{f}, true
}

func listDisks() ([]Disk, error) {
	ds, err := listDisksOS()
	if p, size := testDiskSpec(); p != "" {
		ds = append(ds, Disk{ID: p, Name: "Test drive (a file)", Size: size, Bus: "Test"})
		err = nil
	}
	if ds == nil {
		ds = []Disk{}
	}
	return ds, err
}

// ------------------------------------------------------------ catalogue ---

type flexInt int64

func (f *flexInt) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "" || s == "null" {
		return nil
	}
	n, err := strconv.ParseInt(s, 10, 64)
	*f = flexInt(n)
	return err
}

type osItem struct {
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	Icon         string   `json:"icon,omitempty"`
	URL          string   `json:"url,omitempty"`
	ReleaseDate  string   `json:"release_date,omitempty"`
	ExtractSize  flexInt  `json:"extract_size,omitempty"`
	ExtractSHA   string   `json:"extract_sha256,omitempty"`
	DownloadSize flexInt  `json:"image_download_size,omitempty"`
	DownloadSHA  string   `json:"image_download_sha256,omitempty"`
	Devices      []string `json:"devices,omitempty"`
	Subitems     []osItem `json:"subitems,omitempty"`
	SubitemsURL  string   `json:"subitems_url,omitempty"`
	Website      string   `json:"website,omitempty"`
	InitFormat   string   `json:"init_format,omitempty"` // how the OS takes customisation: "systemd", "cloudinit", "cloudinit-rpi"
}

type osDevice struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags"`
	Default     bool     `json:"default,omitempty"`
	Icon        string   `json:"icon,omitempty"`
}

type osList struct {
	Imager struct {
		Devices []osDevice `json:"devices"`
	} `json:"imager"`
	OSList []osItem `json:"os_list"`
}

// tidy drops entries Flashie can't write (Imager's built-in "Erase" and
// "Use custom" actions).
func tidy(items []osItem) []osItem {
	out := items[:0:0]
	for _, it := range items {
		if strings.HasPrefix(it.URL, "internal://") || it.URL == "" && len(it.Subitems) == 0 && it.SubitemsURL == "" {
			continue
		}
		it.Subitems = tidy(it.Subitems)
		if it.URL == "" && it.SubitemsURL == "" && len(it.Subitems) == 0 {
			continue
		}
		out = append(out, it)
	}
	return out
}

func fetchOSList(u string) (osList, bool, error) {
	var l osList
	if u == "" {
		u = osListURL
	}
	pu, err := url.Parse(u)
	if err != nil || pu.Scheme != "https" {
		return l, false, errors.New("the OS list address must be https")
	}
	key := sha1.Sum([]byte(u))
	cache := filepath.Join(imagesDir(), "oslist-"+hex.EncodeToString(key[:6])+".json")
	var body []byte
	cached := false
	resp, err := httpGet(u, "application/json")
	if err == nil {
		if resp.StatusCode == 200 {
			body, err = io.ReadAll(io.LimitReader(resp.Body, 16<<20))
		} else {
			err = fmt.Errorf("the OS list server said %s", resp.Status)
		}
		resp.Body.Close()
	}
	if err != nil || json.Unmarshal(body, &l) != nil {
		b, cerr := os.ReadFile(cache) // offline: use the last copy
		if cerr != nil {
			if err == nil {
				err = errors.New("the OS list couldn't be read")
			}
			return l, false, err
		}
		body, cached = b, true
		l = osList{}
		if err := json.Unmarshal(body, &l); err != nil {
			return l, false, err
		}
	} else {
		os.WriteFile(cache, body, 0o644)
	}
	l.OSList = tidy(l.OSList)
	return l, cached, nil
}

// ------------------------------------------------------------- download ---

func safeName(s string) string {
	s = path.Base(s)
	return strings.Map(func(r rune) rune {
		if r < 32 || strings.ContainsRune(`<>:"/\|?*`, r) {
			return '_'
		}
		return r
	}, s)
}

// download fetches a catalogue image into the cache (resuming a partial
// download), checking its SHA-256. It returns the cached file's path.
func download(it osItem, cancel *atomic.Bool, tell func(done, total int64)) (string, error) {
	pu, err := url.Parse(it.URL)
	if err != nil || pu.Scheme != "https" {
		return "", errors.New("that image's address isn't https")
	}
	key := strings.ToLower(it.DownloadSHA)
	if len(key) > 16 {
		key = key[:16]
	}
	if key == "" {
		k := sha1.Sum([]byte(it.URL))
		key = hex.EncodeToString(k[:8])
	}
	final := filepath.Join(imagesDir(), key+"-"+safeName(pu.Path))
	if st, err := os.Stat(final); err == nil && (it.DownloadSize == 0 || st.Size() == int64(it.DownloadSize)) {
		return final, nil // already downloaded and checked
	}
	part := final + ".part"
	h := sha256.New()
	var have int64
	if f, err := os.Open(part); err == nil {
		have, _ = io.Copy(h, f)
		f.Close()
	}
	req, _ := http.NewRequest("GET", it.URL, nil)
	req.Header.Set("User-Agent", userAgent)
	if have > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", have))
	}
	resp, err := imgHTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("couldn't reach %s — check your internet connection (%v)", pu.Host, err)
	}
	defer resp.Body.Close()
	flags := os.O_WRONLY | os.O_CREATE | os.O_APPEND
	switch {
	case resp.StatusCode == 206 && have > 0:
	case resp.StatusCode == 200:
		have, h = 0, sha256.New()
		flags |= os.O_TRUNC
	case resp.StatusCode == 416 && have > 0 && have == int64(it.DownloadSize):
		resp.Body.Close()
	default:
		return "", fmt.Errorf("the download server said %s", resp.Status)
	}
	total := int64(it.DownloadSize)
	if total == 0 && resp.ContentLength > 0 {
		total = have + resp.ContentLength
	}
	out, err := os.OpenFile(part, flags, 0o644)
	if err != nil {
		return "", err
	}
	buf := make([]byte, 1<<20)
	done := have
	last := time.Time{}
	for resp.StatusCode != 416 {
		if cancel.Load() {
			out.Close()
			return "", errors.New("download cancelled (Flashie will carry on from here next time)")
		}
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, err := out.Write(buf[:n]); err != nil {
				out.Close()
				return "", fmt.Errorf("couldn't save the download — is your disk full? (%v)", err)
			}
			h.Write(buf[:n])
			done += int64(n)
			if time.Since(last) > 200*time.Millisecond {
				last = time.Now()
				tell(done, total)
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			out.Close()
			return "", fmt.Errorf("the download stopped (%v) — press Write again to carry on", rerr)
		}
	}
	if err := out.Close(); err != nil {
		return "", err
	}
	tell(done, total)
	if it.DownloadSHA != "" && !strings.EqualFold(hex.EncodeToString(h.Sum(nil)), it.DownloadSHA) {
		os.Remove(part)
		return "", errors.New("the download was damaged (its checksum didn't match) — press Write to try again")
	}
	if err := os.Rename(part, final); err != nil {
		return "", err
	}
	return final, nil
}

// ----------------------------------------------------------------- jobs ---

type imgJob struct {
	mu       sync.Mutex
	token    string
	spec     writeSpec
	fetched  bool
	lastSeen time.Time
	cancel   atomic.Bool
	final    chan progress
}

var (
	imgMu    sync.Mutex
	imgCur   *imgJob
	selfBase string // http://127.0.0.1:PORT/
)

func emitImg(p progress) {
	b, _ := json.Marshal(p)
	m := map[string]any{}
	json.Unmarshal(b, &m)
	m["type"] = "img"
	events.send(m)
}

type writeReq struct {
	Path     string     `json:"path"`
	Item     *osItem    `json:"item"`
	Disk     string     `json:"disk"`
	DiskSize int64      `json:"diskSize"`
	Verify   bool       `json:"verify"`
	Eject    bool       `json:"eject"`
	Custom   *customise `json:"custom"`
}

func handleImgWrite(w http.ResponseWriter, r *http.Request) {
	var req writeReq
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		jsonErr(w, "bad request")
		return
	}
	d, err := findDisk(req.Disk)
	if err != nil {
		jsonErr(w, err.Error())
		return
	}
	if d.Size != req.DiskSize {
		jsonErr(w, "the drive changed since you chose it — choose it again")
		return
	}
	if req.Item == nil && req.Path == "" {
		jsonErr(w, "choose an OS or an image file first")
		return
	}
	custom, err := req.Custom.prepare()
	if err != nil {
		jsonErr(w, "OS customisation: "+err.Error())
		return
	}
	req.Custom = custom
	if !busy.CompareAndSwap(false, true) {
		jsonErr(w, "Flashie is busy with another job")
		return
	}
	job := &imgJob{final: make(chan progress, 1)}
	imgMu.Lock()
	imgCur = job
	imgMu.Unlock()
	go func() {
		defer busy.Store(false)
		warn, err := runImgJob(job, req, d)
		p := progress{Phase: "finish", Finished: true, OK: err == nil, Warn: warn}
		if err != nil {
			p.Err = err.Error()
		}
		emitImg(p)
		imgMu.Lock()
		if imgCur == job {
			imgCur = nil
		}
		imgMu.Unlock()
	}()
	writeJSON(w, map[string]any{"ok": true})
}

func runImgJob(job *imgJob, req writeReq, d Disk) (string, error) {
	spec := writeSpec{Device: d.ID, Verify: req.Verify, Eject: req.Eject, Custom: req.Custom}
	src := req.Path
	if req.Item != nil {
		it := *req.Item
		if it.ExtractSize > 0 && int64(it.ExtractSize) > d.Size {
			return "", fmt.Errorf("%s needs a drive of at least %s — this one is %s", it.Name, imgBytes(int64(it.ExtractSize)), imgBytes(d.Size))
		}
		emitImg(progress{Phase: "download", Text: "Downloading " + it.Name + "…"})
		p, err := download(it, &job.cancel, func(done, total int64) {
			emitImg(progress{Phase: "download", Done: done, Total: total, Text: "Downloading " + it.Name + "…"})
		})
		if err != nil {
			return "", err
		}
		src = p
		if spec.Custom != nil {
			spec.Custom.Format = it.InitFormat
		}
		spec.Size, spec.SHA256 = int64(it.ExtractSize), it.ExtractSHA
	}
	info, err := sniffImage(src)
	if err != nil {
		return "", err
	}
	spec.Source = info
	if info.Windows != nil {
		spec.Layout, spec.Custom = "windows", nil
		if need := info.Windows.Need; need > d.Size {
			return "", fmt.Errorf("this Windows installer needs a drive of at least %s — this one is %s", imgBytes(need), imgBytes(d.Size))
		}
	}
	if spec.Size == 0 {
		spec.Size = info.Size
	}
	if job.cancel.Load() {
		return "", errors.New("cancelled")
	}
	if canWriteDirect(d) {
		return runWrite(spec, func(p progress) bool { emitImg(p); return job.cancel.Load() })
	}
	return runViaHelper(job, spec)
}

func runViaHelper(job *imgJob, spec writeSpec) (string, error) {
	tok := make([]byte, 24)
	rand.Read(tok)
	job.mu.Lock()
	job.token, job.spec = hex.EncodeToString(tok), spec
	job.mu.Unlock()
	emitImg(progress{Phase: "prepare", Text: "Asking for permission to write the drive…"})
	exited, err := elevate(selfBase, job.token)
	if err != nil {
		return "", err
	}
	start := time.Now()
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		select {
		case p := <-job.final:
			if !p.OK {
				return "", errors.New(p.Err)
			}
			return p.Warn, nil
		case err := <-exited:
			exited = nil
			select { // its last message may still be on the way
			case p := <-job.final:
				if !p.OK {
					return "", errors.New(p.Err)
				}
				return p.Warn, nil
			case <-time.After(3 * time.Second):
			}
			job.mu.Lock()
			fetched := job.fetched
			job.mu.Unlock()
			if err != nil && !fetched {
				return "", err
			}
			return "", errors.New("the writer stopped unexpectedly — check the drive and try again")
		case <-tick.C:
			job.mu.Lock()
			fetched, seen := job.fetched, job.lastSeen
			job.mu.Unlock()
			if !fetched && time.Since(start) > 3*time.Minute {
				return "", errors.New("the permission prompt wasn't answered")
			}
			if fetched && time.Since(seen) > 60*time.Second {
				return "", errors.New("the writer stopped responding — check the drive and try again")
			}
		}
	}
}

func helperJob(r *http.Request) *imgJob {
	imgMu.Lock()
	j := imgCur
	imgMu.Unlock()
	if j == nil {
		return nil
	}
	j.mu.Lock()
	ok := j.token != "" && subtle.ConstantTimeCompare([]byte(j.token), []byte(r.URL.Query().Get("token"))) == 1
	j.mu.Unlock()
	if !ok {
		return nil
	}
	return j
}

// --------------------------------------------------------------- routes ---

func imgRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/imager.js", func(w http.ResponseWriter, r *http.Request) {
		b, _ := assets.ReadFile("imager.js")
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Write(b)
	})
	mux.HandleFunc("/api/img/disks", func(w http.ResponseWriter, r *http.Request) {
		ds, err := listDisks()
		res := map[string]any{"disks": ds}
		if err != nil {
			res["error"] = err.Error()
		}
		writeJSON(w, res)
	})
	mux.HandleFunc("/api/img/oslist", func(w http.ResponseWriter, r *http.Request) {
		l, cached, err := fetchOSList(r.URL.Query().Get("url"))
		if err != nil {
			jsonErr(w, err.Error())
			return
		}
		writeJSON(w, map[string]any{"devices": l.Imager.Devices, "items": l.OSList, "cached": cached})
	})
	mux.HandleFunc("/api/img/inspect", func(w http.ResponseWriter, r *http.Request) {
		info, err := sniffImage(r.URL.Query().Get("path"))
		if err != nil {
			jsonErr(w, err.Error())
			return
		}
		writeJSON(w, info)
	})
	mux.HandleFunc("/api/img/pick", post(func(w http.ResponseWriter, r *http.Request) {
		p, err := pickImageFile()
		if errors.Is(err, errNoPicker) {
			writeJSON(w, map[string]any{"fallback": true})
			return
		}
		if err != nil || p == "" {
			writeJSON(w, map[string]any{"none": true})
			return
		}
		info, err := sniffImage(p)
		if err != nil {
			jsonErr(w, err.Error())
			return
		}
		writeJSON(w, info)
	}))
	mux.HandleFunc("/api/img/upload", post(func(w http.ResponseWriter, r *http.Request) {
		dir := filepath.Join(imagesDir(), "uploads")
		os.MkdirAll(dir, 0o755)
		p := filepath.Join(dir, safeName(r.URL.Query().Get("name")))
		f, err := os.Create(p)
		if err != nil {
			jsonErr(w, err.Error())
			return
		}
		_, err = io.Copy(f, r.Body)
		if cerr := f.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			os.Remove(p)
			jsonErr(w, "couldn't copy the file in: "+err.Error())
			return
		}
		info, err := sniffImage(p)
		if err != nil {
			jsonErr(w, err.Error())
			return
		}
		writeJSON(w, info)
	}))
	mux.HandleFunc("/api/img/write", post(handleImgWrite))
	mux.HandleFunc("/api/img/cancel", post(func(w http.ResponseWriter, r *http.Request) {
		imgMu.Lock()
		if imgCur != nil {
			imgCur.cancel.Store(true)
		}
		imgMu.Unlock()
		writeJSON(w, map[string]any{"ok": true})
	}))
	mux.HandleFunc("/api/img/cache", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			if busy.Load() {
				jsonErr(w, "wait for the current job to finish")
				return
			}
			ents, _ := os.ReadDir(imagesDir())
			for _, e := range ents {
				if !strings.HasPrefix(e.Name(), "oslist-") {
					os.RemoveAll(filepath.Join(imagesDir(), e.Name()))
				}
			}
		}
		var n, sz int64
		filepath.Walk(imagesDir(), func(p string, fi os.FileInfo, err error) error {
			if err == nil && !fi.IsDir() && !strings.HasPrefix(fi.Name(), "oslist-") {
				n++
				sz += fi.Size()
			}
			return nil
		})
		writeJSON(w, map[string]any{"files": n, "bytes": sz, "dir": imagesDir()})
	})
	// The elevated helper's two calls, guarded by the one-off token.
	mux.HandleFunc("/api/img/helper/job", func(w http.ResponseWriter, r *http.Request) {
		j := helperJob(r)
		if j == nil {
			http.Error(w, "no", 403)
			return
		}
		j.mu.Lock()
		defer j.mu.Unlock()
		if j.fetched {
			http.Error(w, "no", 403)
			return
		}
		j.fetched, j.lastSeen = true, time.Now()
		writeJSON(w, j.spec)
	})
	mux.HandleFunc("/api/img/helper/progress", func(w http.ResponseWriter, r *http.Request) {
		j := helperJob(r)
		if j == nil || r.Method != http.MethodPost {
			http.Error(w, "no", 403)
			return
		}
		var p progress
		if json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&p) != nil {
			http.Error(w, "bad", 400)
			return
		}
		j.mu.Lock()
		j.lastSeen = time.Now()
		j.mu.Unlock()
		if p.Finished {
			select {
			case j.final <- p:
			default:
			}
		} else {
			emitImg(p)
		}
		writeJSON(w, map[string]any{"cancel": j.cancel.Load()})
	})
}

var errNoPicker = errors.New("no native file picker")
