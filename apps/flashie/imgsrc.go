package flashie

// Image sources for the "Write image" tab: .img/.iso as they are, or packed
// in .xz, .zst, .gz, .bz2 or .zip. Detected by their first bytes, not the name.

import (
	"archive/zip"
	"bufio"
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"

	"tinkerie/shared/xz"
	"tinkerie/shared/zstd"
)

type imageInfo struct {
	Path   string `json:"path"`
	Name   string `json:"name"`
	Kind   string `json:"kind"`   // "raw", "xz", "zstd", "gzip", "bzip2", "zip"
	Size   int64  `json:"size"`   // written size if known in advance, else 0
	Packed int64  `json:"packed"` // file size on disk
	Inner  string `json:"inner,omitempty"`
	// A Windows installer ISO: copied file by file onto a new file system.
	Windows *winInfo `json:"windows,omitempty"`
	Note    string   `json:"note,omitempty"` // a warning to show
}

func sniffImage(path string) (imageInfo, error) {
	info := imageInfo{Path: path, Name: filepath.Base(path)}
	f, err := os.Open(path)
	if err != nil {
		return info, fmt.Errorf("couldn't open the image: %v", err)
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return info, err
	}
	if st.IsDir() {
		return info, errors.New("that's a folder, not an image file")
	}
	info.Packed = st.Size()
	head := make([]byte, 8)
	n, _ := io.ReadFull(f, head)
	head = head[:n]
	switch {
	case xz.IsXZ(head):
		info.Kind = "xz"
	case bytes.HasPrefix(head, []byte{0x1F, 0x8B}):
		info.Kind = "gzip"
	case bytes.HasPrefix(head, []byte("BZh")):
		info.Kind = "bzip2"
	case bytes.HasPrefix(head, []byte("PK\x03\x04")):
		info.Kind = "zip"
		zr, err := zip.NewReader(f, st.Size())
		if err != nil {
			return info, fmt.Errorf("the zip file is damaged: %v", err)
		}
		e := pickZipEntry(zr.File)
		if e == nil {
			return info, errors.New("the zip file has no image inside")
		}
		info.Inner, info.Size = e.Name, int64(e.UncompressedSize64)
	case zstd.IsZstd(head):
		info.Kind = "zstd"
	case bytes.HasPrefix(head, []byte("7z\xBC\xAF\x27\x1C")):
		return info, errors.New(".7z files aren't supported — unpack it first")
	default:
		info.Kind = "raw"
		info.Size = st.Size()
		if info.Size == 0 {
			return info, errors.New("the file is empty")
		}
		w, err := sniffWindows(path)
		if err != nil {
			return info, err
		}
		if w != nil {
			info.Windows, info.Size = w, 0
		} else {
			info.Note = isoNote(path)
		}
	}
	if info.Windows == nil && info.Kind != "raw" && strings.HasSuffix(strings.ToLower(info.Inner+info.Name), ".iso") {
		info.Note = "If this is a Windows installer, unpack the .iso first so Flashie can make a proper USB installer from it."
	}
	return info, nil
}

func pickZipEntry(files []*zip.File) *zip.File {
	var cands []*zip.File
	for _, f := range files {
		if f.FileInfo().IsDir() || strings.HasPrefix(filepath.Base(f.Name), ".") || strings.HasPrefix(f.Name, "__MACOSX") {
			continue
		}
		cands = append(cands, f)
	}
	sort.Slice(cands, func(i, j int) bool { return cands[i].UncompressedSize64 > cands[j].UncompressedSize64 })
	for _, f := range cands {
		switch strings.ToLower(filepath.Ext(f.Name)) {
		case ".img", ".iso", ".raw", ".bin", ".hddimg", ".sdcard", ".wic":
			return f
		}
	}
	if len(cands) > 0 {
		return cands[0]
	}
	return nil
}

type multiCloser struct {
	io.Reader
	close []func() error
}

func (m *multiCloser) Close() error {
	var first error
	for i := len(m.close) - 1; i >= 0; i-- {
		if err := m.close[i](); err != nil && first == nil {
			first = err
		}
	}
	return first
}

// openImage returns the image's raw bytes, unpacking as it reads.
// read, if not nil, counts the packed bytes read so far (for progress).
func openImage(info imageInfo, read *atomic.Int64) (io.ReadCloser, error) {
	f, err := os.Open(info.Path)
	if err != nil {
		return nil, err
	}
	mc := &multiCloser{close: []func() error{f.Close}}
	br := bufio.NewReaderSize(&countingFile{f, read}, 1<<20)
	switch info.Kind {
	case "raw":
		mc.Reader = br
	case "xz":
		r, err := xz.NewReader(br)
		if err != nil {
			f.Close()
			return nil, err
		}
		mc.Reader = r
	case "zstd":
		mc.Reader = zstd.NewReader(br)
	case "gzip":
		r, err := gzip.NewReader(br)
		if err != nil {
			f.Close()
			return nil, fmt.Errorf("the .gz file is damaged: %v", err)
		}
		mc.Reader = r
	case "bzip2":
		mc.Reader = bzip2.NewReader(br)
	case "zip":
		st, _ := f.Stat()
		zr, err := zip.NewReader(&countingFile{f, read}, st.Size())
		if err != nil {
			f.Close()
			return nil, err
		}
		e := pickZipEntry(zr.File)
		if e == nil {
			f.Close()
			return nil, errors.New("the zip file has no image inside")
		}
		r, err := e.Open()
		if err != nil {
			f.Close()
			return nil, err
		}
		mc.Reader, mc.close = r, append(mc.close, r.Close)
	default:
		f.Close()
		return nil, errors.New("unknown image type")
	}
	return mc, nil
}

type countingFile struct {
	f *os.File
	n *atomic.Int64
}

func (c *countingFile) Read(p []byte) (int, error) {
	n, err := c.f.Read(p)
	if c.n != nil {
		c.n.Add(int64(n))
	}
	return n, err
}

func (c *countingFile) ReadAt(p []byte, off int64) (int, error) {
	n, err := c.f.ReadAt(p, off)
	if c.n != nil {
		c.n.Add(int64(n))
	}
	return n, err
}
