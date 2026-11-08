package flashie

// Windows installer ISOs. Unlike Linux ISOs these aren't made to be copied
// straight onto a USB drive, so Flashie does what Rufus and Microsoft's
// Media Creation Tool do: it formats the drive and copies the installer's
// files onto it, for PCs that start in UEFI mode (every PC that can run
// Windows 11).
//
//   - install.wim under 4 GB: one FAT32 partition with everything.
//   - install.wim 4 GB or bigger (too big for FAT32): a FAT32 "boot"
//     partition with everything except the sources folder, plus
//     sources\boot.wim, and an exFAT partition with the whole installer.
//     The PC starts from the FAT32 one; Windows Setup then finds
//     install.wim on the exFAT one.
//
// Linux ISOs (Ubuntu, Fedora, Debian, Mint…) are "hybrid" images and are
// written as they are, like .img files. Made by nrmluvdev.

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"tinkerie/shared/exfat"
	"tinkerie/shared/fat"
	"tinkerie/shared/udf"
)

type winInfo struct {
	Label  string `json:"label"`
	Arch   string `json:"arch"`   // x64, arm64, x86
	Total  int64  `json:"total"`  // bytes of files
	Files  int    `json:"files"`  // how many
	Layout string `json:"layout"` // "FAT32" or "FAT32 + exFAT"
	Need   int64  `json:"need"`   // smallest drive it fits on
}

const mib = int64(1) << 20

// fat32Limit is the biggest file FAT32 holds (a variable so tests can use
// small files).
var fat32Limit = int64(1)<<32 - 1

type winPlan struct {
	files  []udf.File
	label  string
	big    bool  // a file too big for FAT32
	p1Size int64 // FAT32 partition, bytes (whole drive if !big)
	need   int64
}

func lowerPath(p string) string { return strings.ToLower(p) }

// onBoot reports whether a file goes on the FAT32 boot partition when the
// installer is split: everything except sources\, plus sources\boot.wim.
func onBoot(f udf.File) bool {
	p := lowerPath(f.Path)
	return !(p == "sources" || strings.HasPrefix(p, "sources/")) || p == "sources" || p == "sources/boot.wim"
}

func roundUp(n, to int64) int64 { return (n + to - 1) / to * to }

func planWindows(label string, files []udf.File) (*winPlan, *winInfo, error) {
	have := map[string]bool{}
	pl := &winPlan{files: files, label: label}
	var total, bootBytes, all int64
	n := 0
	for _, f := range files {
		have[lowerPath(f.Path)] = true
		if f.Dir {
			all += 32 << 10
			bootBytes += 32 << 10
			continue
		}
		n++
		total += f.Size
		all += roundUp(f.Size, 128<<10)
		if onBoot(f) {
			bootBytes += roundUp(f.Size, 32<<10)
		}
		if f.Size > fat32Limit {
			pl.big = true
		}
	}
	if !have["sources/boot.wim"] || !(have["sources/install.wim"] || have["sources/install.esd"] || have["sources/install.swm"]) {
		return nil, nil, nil // not a Windows installer
	}
	info := &winInfo{Label: label, Total: total, Files: n, Layout: "FAT32"}
	switch {
	case have["efi/boot/bootx64.efi"]:
		info.Arch = "x64"
	case have["efi/boot/bootaa64.efi"]:
		info.Arch = "arm64"
	case have["efi/boot/bootia32.efi"]:
		info.Arch = "x86"
	default:
		return nil, nil, errors.New("this Windows ISO can't start on UEFI PCs (it has no efi\\boot folder), so Flashie can't make a USB installer from it")
	}
	if pl.big {
		info.Layout = "FAT32 + exFAT"
		pl.p1Size = max(512*mib, roundUp(bootBytes+bootBytes/10+64*mib, mib))
		pl.need = mib + pl.p1Size + roundUp(all+all/20+64*mib, mib)
	} else {
		pl.need = mib + max(300*mib, roundUp(all+all/20+64*mib, mib)) // FAT32 needs 65525 clusters
	}
	info.Need = pl.need
	return pl, info, nil
}

// sniffWindows looks for a Windows installer inside a raw .iso.
func sniffWindows(path string) (*winInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if !udf.IsUDF(f) {
		return nil, nil
	}
	label, files, err := udf.Read(f)
	if err != nil {
		return nil, nil // some other kind of disc image: write it as it is
	}
	_, info, err := planWindows(label, files)
	return info, err
}

// isoNote warns about ISOs that aren't made to start from a USB drive.
func isoNote(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	b := make([]byte, 6)
	if _, err := f.ReadAt(b, 16*2048); err != nil || string(b[1:6]) != "CD001" {
		return ""
	}
	mbr := make([]byte, 1024)
	f.ReadAt(mbr, 0)
	if mbr[510] == 0x55 && mbr[511] == 0xAA || string(mbr[512:520]) == "EFI PART" {
		return "" // a hybrid ISO: fine on a USB drive
	}
	return "This ISO is made for CDs and DVDs only, so a computer may not start from it on a USB drive."
}

// ------------------------------------------------------------- writing ---

func chs(lba int64) [3]byte {
	c, rem := lba/(255*63), lba%(255*63)
	h, s := rem/63, rem%63+1
	if c > 1023 {
		c, h, s = 1023, 254, 63
	}
	return [3]byte{byte(h), byte(s) | byte(c>>8)<<6, byte(c)}
}

// mbr makes a partition table. parts: start and size in bytes, type.
func mbr(parts [][3]int64) []byte {
	b := alignedBuf(pageSize)
	clear(b)
	b[0], b[1] = 0xCD, 0x18 // not bootable on BIOS: int 18h, try the next device
	rand.Read(b[440:444])   // disk signature
	for i, p := range parts {
		e := b[446+16*i:]
		if i == 0 {
			e[0] = 0x80
		}
		s, n := p[0]/512, p[1]/512
		a, z := chs(s), chs(s+n-1)
		copy(e[1:4], a[:])
		e[4] = byte(p[2])
		copy(e[5:8], z[:])
		binary.LittleEndian.PutUint32(e[8:], uint32(s))
		binary.LittleEndian.PutUint32(e[12:], uint32(n))
	}
	b[510], b[511] = 0x55, 0xAA
	return b
}

type copyJob struct {
	file udf.File
	dst  []int64
	sum  []byte
}

func runWriteWindows(spec writeSpec, d Disk, report func(progress) bool) error {
	iso, err := os.Open(spec.Source.Path)
	if err != nil {
		return err
	}
	defer iso.Close()
	label, files, err := udf.Read(iso)
	if err != nil {
		return fmt.Errorf("couldn't read the Windows ISO: %v", err)
	}
	pl, _, err := planWindows(label, files)
	if err != nil {
		return err
	}
	if pl == nil {
		return errors.New("this isn't a Windows installer ISO")
	}
	if pl.need > d.Size {
		return fmt.Errorf("this Windows installer needs a drive of at least %s — this one is %s", imgBytes(pl.need), imgBytes(d.Size))
	}
	if d.Size/512 >= 1<<32 {
		return errors.New("drives bigger than 2 TB can't be used for a Windows installer")
	}
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
	// Blank the start and the end (old partition tables) first.
	blank := alignedBuf(int(mib))
	clear(blank)
	if _, err := rd.WriteAt(blank, 0); err != nil {
		return diskErr("write", err)
	}
	if d.Size > 16*mib {
		if _, err := rd.WriteAt(blank, (d.Size-mib)/4096*4096); err != nil {
			return diskErr("write", err)
		}
	}

	report(progress{Phase: "prepare", Text: "Making the file system…"})
	pd := newPageDev(rd, d.Size)
	end := d.Size / mib * mib
	var parts [][3]int64
	jobs := map[string]*copyJob{}
	addDst := func(f udf.File, off int64) {
		j := jobs[f.Path]
		if j == nil {
			j = &copyJob{file: f}
			jobs[f.Path] = j
		}
		if off >= 0 {
			j.dst = append(j.dst, off)
		}
	}
	fat1 := mib
	p1 := min(end-mib, 32<<30) // FAT32 beyond 32 GB only makes the FAT huge
	if pl.big {
		p1 = pl.p1Size
	}
	lab := strings.ToUpper(pl.label)
	if lab == "" {
		lab = "WINDOWS"
	}
	ser := make([]byte, 4)
	rand.Read(ser)
	fs, err := fat.Format32(pd, fat1, p1, lab, binary.LittleEndian.Uint32(ser))
	if err != nil {
		return err
	}
	parts = append(parts, [3]int64{fat1, p1, 0x0C})
	for _, f := range pl.files {
		if pl.big && !onBoot(f) {
			continue
		}
		if f.Dir {
			if err := fs.MkdirAll(f.Path); err != nil {
				return fmt.Errorf("couldn't make %s: %v", f.Path, err)
			}
			continue
		}
		off, err := fs.CreateContig(f.Path, f.Size)
		if err != nil {
			return fmt.Errorf("couldn't make %s: %v", f.Path, err)
		}
		addDst(f, off)
	}
	if err := fs.Flush(); err != nil {
		return err
	}
	if pl.big {
		p2 := fat1 + p1
		var list []exfat.File
		for _, f := range pl.files {
			list = append(list, exfat.File{Path: f.Path, Size: f.Size, Dir: f.Dir})
		}
		offs, err := exfat.Build(pd, p2, end-p2, pl.label, list, time.Now())
		if err != nil {
			return err
		}
		for _, f := range pl.files {
			if !f.Dir {
				o, ok := offs[f.Path]
				if !ok {
					o = -1
				}
				addDst(f, o)
			}
		}
		parts = append(parts, [3]int64{p2, end - p2, 0x07})
	}
	if err := pd.Flush(); err != nil {
		return err
	}

	// Copy the files, in the order they lie in the ISO.
	var list []*copyJob
	var total int64
	for _, j := range jobs {
		if len(j.dst) > 0 && j.file.Size > 0 {
			list = append(list, j)
			total += j.file.Size * int64(len(j.dst))
		}
	}
	sort.Slice(list, func(a, b int) bool { return list[a].file.Extents[0].Off < list[b].file.Extents[0].Off })
	buf := alignedBuf(blockSize)
	var done int64
	last := time.Time{}
	for _, j := range list {
		h := sha256.New()
		src := &extentReader{r: iso, ext: j.file.Extents}
		for pos := int64(0); pos < j.file.Size; {
			n := min(int64(blockSize), j.file.Size-pos)
			chunk := buf[:n]
			if _, err := io.ReadFull(src, chunk); err != nil {
				return fmt.Errorf("couldn't read the ISO: %v", err)
			}
			h.Write(chunk)
			pad := int(roundUp(n, pageSize)) // within the file's last cluster
			clear(buf[n:pad])
			for _, dst := range j.dst {
				if _, err := rd.WriteAt(buf[:pad], dst+pos); err != nil {
					return diskErr("write", err)
				}
			}
			pos += n
			done += n * int64(len(j.dst))
			if time.Since(last) > 200*time.Millisecond {
				last = time.Now()
				if report(progress{Phase: "write", Done: done, Total: total, Written: done, Text: "Copying the Windows installer…"}) {
					return errImgCancelled
				}
			}
		}
		j.sum = h.Sum(nil)
	}
	report(progress{Phase: "write", Done: total, Total: total, Written: total, Text: "Finishing writing…"})
	if err := rd.Sync(); err != nil {
		return diskErr("finish writing to", err)
	}
	table := mbr(parts)
	if _, err := rd.WriteAt(table, 0); err != nil {
		return diskErr("write", err)
	}
	if err := rd.Sync(); err != nil {
		return diskErr("finish writing to", err)
	}

	if spec.Verify {
		if c, ok := rd.(interface{ dropCache() }); ok {
			c.dropCache()
		}
		report(progress{Phase: "verify", Total: total, Written: total, Text: "Checking…"})
		if err := pd.Verify(); err != nil {
			return err
		}
		got := alignedBuf(pageSize)
		if _, err := rd.ReadAt(got, 0); err != nil || !bytes.Equal(got, table) {
			return errors.New("checking failed: the drive didn't keep its partition table. The drive may be faulty — try another one")
		}
		var vdone int64
		for _, j := range list {
			for _, dst := range j.dst {
				h := sha256.New()
				for pos := int64(0); pos < j.file.Size; {
					n := min(int64(blockSize), j.file.Size-pos)
					rn := int(roundUp(n, pageSize))
					if _, err := rd.ReadAt(buf[:rn], dst+pos); err != nil && err != io.EOF {
						return diskErr("read back", err)
					}
					h.Write(buf[:n])
					pos += n
					vdone += n
					if time.Since(last) > 200*time.Millisecond {
						last = time.Now()
						if report(progress{Phase: "verify", Done: vdone, Total: total, Written: total, Text: "Checking…"}) {
							return errors.New("check cancelled — the installer was written but not checked")
						}
					}
				}
				if !bytes.Equal(h.Sum(nil), j.sum) {
					return fmt.Errorf("checking failed: %s on the drive doesn't match the ISO. The drive may be faulty or fake — try another one", j.file.Path)
				}
			}
		}
	}
	report(progress{Phase: "finish", Done: 1, Total: 1, Written: total, Text: "Tidying up…"})
	closed = true
	rd.Close()
	finishDisk(d, spec.Eject)
	return nil
}

// extentReader reads a file from its pieces in the ISO.
type extentReader struct {
	r   io.ReaderAt
	ext []udf.Extent
	i   int
	at  int64
}

func (e *extentReader) Read(p []byte) (int, error) {
	n := 0
	for n < len(p) && e.i < len(e.ext) {
		x := e.ext[e.i]
		k := int(min(int64(len(p)-n), x.Len-e.at))
		if x.Zero {
			clear(p[n : n+k])
		} else if _, err := e.r.ReadAt(p[n:n+k], x.Off+e.at); err != nil {
			return n, err
		}
		n += k
		e.at += int64(k)
		if e.at == x.Len {
			e.i, e.at = e.i+1, 0
		}
	}
	if n == 0 {
		return 0, io.EOF
	}
	return n, nil
}
