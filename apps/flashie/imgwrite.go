package flashie

// The image writer core, shared by every platform. It runs either inside
// Flashie (when Flashie may already open the drive) or in a short-lived
// helper: the same program relaunched with administrator rights, which
// fetches its job from Flashie and reports progress back over localhost.

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"
)

// Disk is a drive the image can go on. Only removable USB/SD drives that
// don't hold the running system are ever listed.
type Disk struct {
	ID     string   `json:"id"`   // /dev/sdb, /dev/disk4, \\.\PhysicalDrive2
	Name   string   `json:"name"` // "SanDisk Ultra"
	Size   int64    `json:"size"`
	Bus    string   `json:"bus"`    // "USB", "SD card"
	Mounts []string `json:"mounts"` // where its partitions are mounted (E:, /media/pi/boot)
}

// rawDisk is an open drive. Offsets and lengths are multiples of 512 and
// buffers are 4 KiB aligned, which raw devices on every platform accept.
type rawDisk interface {
	WriteAt(p []byte, off int64) (int, error)
	ReadAt(p []byte, off int64) (int, error)
	Sync() error
	Close() error
}

type writeSpec struct {
	Device string     `json:"device"`
	Source imageInfo  `json:"source"`
	Size   int64      `json:"size"`   // unpacked size, if known
	SHA256 string     `json:"sha256"` // published hash of the unpacked image, if any
	Verify bool       `json:"verify"`
	Eject  bool       `json:"eject"`
	Custom *customise `json:"custom,omitempty"` // OS customisation, already checked and hashed
	Layout string     `json:"layout,omitempty"` // "" = copy the image as it is; "windows" = Windows installer
}

type progress struct {
	Phase    string `json:"phase"` // prepare, write, verify, finish
	Done     int64  `json:"done"`
	Total    int64  `json:"total"`
	Written  int64  `json:"written"`
	Text     string `json:"text,omitempty"`
	Err      string `json:"err,omitempty"`
	Finished bool   `json:"finished,omitempty"`
	OK       bool   `json:"ok,omitempty"`
	Warn     string `json:"warn,omitempty"` // finished, but with something to tell the user
}

var errImgCancelled = errors.New("cancelled — the drive is only partly written, so write it again before using it")

const blockSize = 4 << 20

func alignedBuf(n int) []byte {
	b := make([]byte, n+4096)
	off := int(uintptr(unsafe.Pointer(&b[0])) & 4095)
	if off != 0 {
		off = 4096 - off
	}
	return b[off : off+n : off+n]
}

func findDisk(id string) (Disk, error) {
	disks, err := listDisks()
	if err != nil {
		return Disk{}, err
	}
	for _, d := range disks {
		if d.ID == id {
			return d, nil
		}
	}
	return Disk{}, errors.New("that drive isn't plugged in any more, or isn't a removable drive")
}

// padLen rounds n up to whole 4 KiB (or at least 512-byte) sectors without
// running past the end of the disk.
func padLen(n int, off, size int64) int {
	for _, a := range []int{4096, 512} {
		p := (n + a - 1) / a * a
		if off+int64(p) <= size {
			return p
		}
	}
	return n
}

type block struct {
	buf []byte
	n   int
	err error
}

// runWrite writes the image, verifies it and applies any OS customisation.
// report returns true to cancel. warn is a note for the user on success.
func runWrite(spec writeSpec, report func(progress) bool) (warn string, err error) {
	d, err := findDisk(spec.Device)
	if err != nil {
		return "", err
	}
	if spec.Layout == "windows" {
		return "", runWriteWindows(spec, d, report)
	}
	err = runWriteRaw(spec, d, report, &warn)
	return warn, err
}

func runWriteRaw(spec writeSpec, d Disk, report func(progress) bool, warn *string) (err error) {
	if spec.Size > d.Size {
		return fmt.Errorf("the image (%s) is bigger than the drive (%s)", imgBytes(spec.Size), imgBytes(d.Size))
	}
	var packed atomic.Int64
	src, err := openImage(spec.Source, &packed)
	if err != nil {
		return err
	}
	defer src.Close()

	report(progress{Phase: "prepare", Text: "Getting the drive ready…"})
	rd, err := openDiskForWrite(d)
	if err != nil {
		return err
	}
	closed := false
	defer func() {
		if !closed {
			rd.Close()
		}
	}()

	// Reader goroutine: unpacking runs alongside writing.
	pool := make(chan []byte, 3)
	for i := 0; i < 3; i++ {
		pool <- alignedBuf(blockSize)
	}
	blocks := make(chan block, 2)
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		defer close(blocks)
		for {
			var buf []byte
			select {
			case buf = <-pool:
			case <-stop:
				return
			}
			n, err := io.ReadFull(src, buf)
			if err == io.ErrUnexpectedEOF {
				err = io.EOF
			}
			select {
			case blocks <- block{buf, n, err}:
			case <-stop:
				return
			}
			if err != nil {
				return
			}
		}
	}()

	total, unit := spec.Size, "out"
	if total <= 0 {
		total, unit = spec.Source.Packed, "in"
	}
	h := sha256.New()
	first := alignedBuf(blockSize)
	firstLen := 0
	var off int64
	last := time.Time{}
	tell := func(force bool) bool {
		if !force && time.Since(last) < 200*time.Millisecond {
			return false
		}
		last = time.Now()
		done := off
		if unit == "in" {
			done = packed.Load()
		}
		return report(progress{Phase: "write", Done: done, Total: total, Written: off, Text: "Writing…"})
	}
	for b := range blocks {
		if b.n > 0 {
			if off+int64(b.n) > d.Size {
				return fmt.Errorf("the image is bigger than the drive (%s)", imgBytes(d.Size))
			}
			h.Write(b.buf[:b.n])
			n := padLen(b.n, off, d.Size)
			clear(b.buf[b.n:n])
			if off == 0 {
				// Hold the first block (partition table) back until the end, and
				// blank it now, so a half-written drive never looks bootable and
				// the computer doesn't try to open its partitions mid-write.
				copy(first, b.buf[:n])
				firstLen = n
				clear(b.buf[:n])
				if d.Size > 16<<20 { // also blank a stale backup partition table
					tail := alignedBuf(1 << 20)
					if _, err := rd.WriteAt(tail, (d.Size-(1<<20))/4096*4096); err != nil {
						return diskErr("write", err)
					}
				}
			}
			if _, err := rd.WriteAt(b.buf[:n], off); err != nil {
				return diskErr("write", err)
			}
			off += int64(b.n)
		}
		pool <- b.buf
		if b.err == io.EOF {
			break
		}
		if b.err != nil {
			if strings.HasPrefix(b.err.Error(), "xz:") || strings.HasPrefix(b.err.Error(), "zstd") || strings.Contains(b.err.Error(), "gzip") || strings.Contains(b.err.Error(), "bzip2") || strings.Contains(b.err.Error(), "zip") {
				return fmt.Errorf("the image file is damaged (%v)", b.err)
			}
			return fmt.Errorf("couldn't read the image: %v", b.err)
		}
		if tell(false) {
			return errImgCancelled
		}
	}
	if off == 0 {
		return errors.New("the image is empty")
	}
	report(progress{Phase: "write", Done: total, Total: total, Written: off, Text: "Finishing writing…"})
	if err := rd.Sync(); err != nil {
		return diskErr("finish writing to", err)
	}
	if _, err := rd.WriteAt(first[:firstLen], 0); err != nil {
		return diskErr("write", err)
	}
	if err := rd.Sync(); err != nil {
		return diskErr("finish writing to", err)
	}
	sum := h.Sum(nil)
	if spec.SHA256 != "" && !strings.EqualFold(hex.EncodeToString(sum), spec.SHA256) {
		return errors.New("the unpacked image doesn't match its published checksum — the download may be damaged. Clear the image cache and try again")
	}

	if spec.Verify {
		if c, ok := rd.(interface{ dropCache() }); ok {
			c.dropCache()
		}
		vh := sha256.New()
		buf := alignedBuf(blockSize)
		report(progress{Phase: "verify", Total: off, Written: off, Text: "Checking…"})
		for pos := int64(0); pos < off; {
			n := blockSize
			if int64(n) > off-pos {
				n = int(off - pos)
			}
			rn := padLen(n, pos, d.Size)
			if _, err := rd.ReadAt(buf[:rn], pos); err != nil && err != io.EOF {
				return diskErr("read back", err)
			}
			vh.Write(buf[:n])
			pos += int64(n)
			if time.Since(last) > 200*time.Millisecond {
				last = time.Now()
				if report(progress{Phase: "verify", Done: pos, Total: off, Written: off, Text: "Checking…"}) {
					return errors.New("check cancelled — the image was written but not checked")
				}
			}
		}
		if !bytes.Equal(vh.Sum(nil), sum) {
			return errors.New("checking failed: what's on the drive doesn't match the image. The card may be faulty or fake — try another one")
		}
	}
	if spec.Custom != nil {
		report(progress{Phase: "customise", Text: "Putting your OS settings on the drive…"})
		pd := newPageDev(rd, d.Size)
		how, err := applyCustom(pd, first[:firstLen], spec.Custom)
		switch {
		case errors.Is(err, errNotPiImage):
			*warn = err.Error()
		case err != nil:
			return fmt.Errorf("the image was written, but the OS settings couldn't be added: %v", err)
		default:
			if err := pd.Flush(); err != nil {
				return err
			}
			if err := rd.Sync(); err != nil {
				return diskErr("finish writing to", err)
			}
			if spec.Verify {
				if err := pd.Verify(); err != nil {
					return err
				}
			}
			*warn = "Your OS settings were added (" + how + "). They're applied the first time it starts, which takes a minute or two longer than usual."
		}
	}
	report(progress{Phase: "finish", Done: 1, Total: 1, Written: off, Text: "Tidying up…"})
	closed = true
	rd.Close()
	finishDisk(d, spec.Eject)
	return nil
}

func diskErr(what string, err error) error {
	return fmt.Errorf("couldn't %s the drive: %v. Check it's plugged in properly and its lock switch (on SD cards) isn't set to LOCK", what, err)
}

func imgBytes(n int64) string {
	f := float64(n)
	for _, u := range []string{"bytes", "KB", "MB", "GB", "TB"} {
		if f < 1000 || u == "TB" {
			if u == "bytes" {
				return fmt.Sprintf("%d bytes", n)
			}
			return fmt.Sprintf("%.1f %s", f, u)
		}
		f /= 1000
	}
	return ""
}

// ------------------------------------------------------- elevated helper ---

// runImgHelper is the entry point of the elevated helper:
//
//	tinkerie --app flashie --img-helper http://127.0.0.1:PORT/ TOKEN
func runImgHelper(base, token string) int {
	cl := &http.Client{Timeout: 20 * time.Second}
	resp, err := cl.Get(base + "api/img/helper/job?token=" + token)
	if err != nil {
		return 3
	}
	var spec writeSpec
	err = json.NewDecoder(resp.Body).Decode(&spec)
	resp.Body.Close()
	if err != nil || resp.StatusCode != 200 {
		return 3
	}
	post := func(p progress) bool {
		b, _ := json.Marshal(p)
		r, err := cl.Post(base+"api/img/helper/progress?token="+token, "application/json", bytes.NewReader(b))
		if err != nil {
			return false // Flashie has gone: finish the job anyway
		}
		defer r.Body.Close()
		var a struct{ Cancel bool }
		json.NewDecoder(r.Body).Decode(&a)
		return a.Cancel
	}
	warn, err := runWrite(spec, post)
	fin := progress{Phase: "finish", Finished: true, OK: err == nil, Warn: warn}
	if err != nil {
		fin.Err = err.Error()
	}
	for i := 0; i < 3; i++ { // make sure the result arrives
		b, _ := json.Marshal(fin)
		if r, e := cl.Post(base+"api/img/helper/progress?token="+token, "application/json", bytes.NewReader(b)); e == nil {
			r.Body.Close()
			break
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		return 1
	}
	return 0
}
