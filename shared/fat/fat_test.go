package fat

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"
	"unicode/utf16"
)

// memDev is a drive in memory.
type memDev struct{ b []byte }

func (m *memDev) ReadAt(p []byte, off int64) (int, error)  { return copy(p, m.b[off:]), nil }
func (m *memDev) WriteAt(p []byte, off int64) (int, error) { return copy(m.b[off:], p), nil }

var fixed = func() time.Time { return time.Date(2026, 11, 8, 15, 30, 10, 0, time.UTC) }

// check is a small fsck: it walks the whole file system from the raw bytes
// (not through FS) and checks the rules a real fsck would.
func check(t *testing.T, dev *memDev, base int64) map[string][]byte {
	t.Helper()
	bs := dev.b[base:]
	if bs[510] != 0x55 || bs[511] != 0xAA {
		t.Fatal("no boot signature")
	}
	bps := int64(binary.LittleEndian.Uint16(bs[11:]))
	spc := int64(bs[13])
	rsvd := int64(binary.LittleEndian.Uint16(bs[14:]))
	nf := int64(bs[16])
	rootEnt := int64(binary.LittleEndian.Uint16(bs[17:]))
	tot := int64(binary.LittleEndian.Uint16(bs[19:]))
	if tot == 0 {
		tot = int64(binary.LittleEndian.Uint32(bs[32:]))
	}
	fsz := int64(binary.LittleEndian.Uint16(bs[22:]))
	if fsz == 0 {
		fsz = int64(binary.LittleEndian.Uint32(bs[36:]))
	}
	rootSecs := (rootEnt*32 + bps - 1) / bps
	data := (rsvd + nf*fsz + rootSecs) * bps
	ncl := (tot - rsvd - nf*fsz - rootSecs) / spc
	f32 := ncl >= 65525
	cs := bps * spc
	fat := bs[rsvd*bps : (rsvd+fsz)*bps]
	for k := int64(1); k < nf; k++ {
		if !bytes.Equal(fat, bs[(rsvd+k*fsz)*bps:(rsvd+(k+1)*fsz)*bps]) {
			t.Fatalf("FAT copy %d differs", k)
		}
	}
	get := func(c int64) int64 {
		if f32 {
			return int64(binary.LittleEndian.Uint32(fat[c*4:]) & 0x0FFFFFFF)
		}
		v := int64(binary.LittleEndian.Uint16(fat[c*2:]))
		if v >= 0xFFF8 {
			v = 0x0FFFFFFF
		}
		return v
	}
	if f32 {
		if binary.LittleEndian.Uint32(fat) != 0x0FFFFFF8 || binary.LittleEndian.Uint32(fat[4:])&0x0FFFFFFF != 0x0FFFFFFF {
			t.Fatal("FAT32 media entries wrong")
		}
	}
	used := map[int64]string{}
	chain := func(c int64, who string) []int64 {
		var out []int64
		for c >= 2 && c < 0x0FFFFFF8 {
			if c >= ncl+2 {
				t.Fatalf("%s: cluster %d out of range", who, c)
			}
			if o, ok := used[c]; ok {
				t.Fatalf("%s: cluster %d also used by %s", who, c, o)
			}
			used[c] = who
			out = append(out, c)
			c = get(c)
		}
		if c != 0x0FFFFFFF && c < 0x0FFFFFF8 && len(out) > 0 {
			t.Fatalf("%s: chain ends badly (%x)", who, c)
		}
		return out
	}
	coff := func(c int64) int64 { return base + data + (c-2)*cs }
	files := map[string][]byte{}
	var walk func(raw []byte, path string, self, parent int64)
	walk = func(raw []byte, path string, self, parent int64) {
		var lfn []uint16
		var sum byte
		seenEnd := false
		names := map[string]bool{}
		for i := 0; i < len(raw)/32; i++ {
			e := raw[i*32 : i*32+32]
			if seenEnd {
				if e[0] != 0 {
					t.Fatalf("%s: entry after the end marker", path)
				}
				continue
			}
			if e[0] == 0 {
				seenEnd = true
				continue
			}
			if e[0] == 0xE5 {
				lfn = nil
				continue
			}
			if e[11] == 0x0F {
				if e[0]&0x40 != 0 {
					lfn = make([]uint16, int(e[0]&0x3F)*13)
					sum = e[13]
				}
				if lfn == nil || e[13] != sum {
					t.Fatalf("%s: stray long-name entry", path)
				}
				o := int(e[0]&0x3F) - 1
				for k, p := range lfnPos {
					lfn[o*13+k] = binary.LittleEndian.Uint16(e[p:])
				}
				continue
			}
			name := shortToName(e[:11], e[12])
			if lfn != nil {
				if lfnSum(e) != sum {
					t.Fatalf("%s: long name checksum wrong for %q", path, name)
				}
				n := 0
				for n < len(lfn) && lfn[n] != 0 {
					n++
				}
				for k := n + 1; k < len(lfn); k++ {
					if lfn[k] != 0xFFFF {
						t.Fatalf("%s: long name padding wrong", path)
					}
				}
				name = string(utf16.Decode(lfn[:n]))
				lfn = nil
			}
			if e[11]&0x08 != 0 {
				continue
			}
			c := int64(binary.LittleEndian.Uint16(e[20:]))<<16 | int64(binary.LittleEndian.Uint16(e[26:]))
			if name == "." || name == ".." {
				want := self
				if name == ".." {
					want = parent
				}
				if c != want {
					t.Fatalf("%s: %q points at %d, want %d", path, name, c, want)
				}
				continue
			}
			if names[strings.ToLower(name)] {
				t.Fatalf("%s: %q twice", path, name)
			}
			names[strings.ToLower(name)] = true
			full := path + "/" + name
			cl := chain(c, full)
			if e[11]&0x10 != 0 {
				var d []byte
				for _, x := range cl {
					d = append(d, dev.b[coff(x):coff(x)+cs]...)
				}
				if !bytes.HasPrefix(d, []byte(".          ")) {
					t.Fatalf("%s: no . entry", full)
				}
				p := c
				if path == "" {
					p = 0
				} else {
					p = self
				}
				walk(d, full, c, p)
				continue
			}
			size := int64(binary.LittleEndian.Uint32(e[28:]))
			if int64(len(cl)) != (size+cs-1)/cs {
				t.Fatalf("%s: %d clusters for %d bytes", full, len(cl), size)
			}
			var b []byte
			for _, x := range cl {
				b = append(b, dev.b[coff(x):coff(x)+cs]...)
			}
			files[full[1:]] = b[:size]
		}
	}
	if f32 {
		root := int64(binary.LittleEndian.Uint32(bs[44:]))
		var d []byte
		for _, x := range chain(root, "/") {
			d = append(d, dev.b[coff(x):coff(x)+cs]...)
		}
		walk(d, "", 0, 0)
		fi := bs[bps:]
		if binary.LittleEndian.Uint32(fi) == 0x41615252 {
			free := int64(0)
			for c := int64(2); c < ncl+2; c++ {
				if get(c) == 0 {
					free++
				}
			}
			if int64(binary.LittleEndian.Uint32(fi[488:])) != free {
				t.Fatalf("FSInfo free count %d, really %d", binary.LittleEndian.Uint32(fi[488:]), free)
			}
		}
	} else {
		walk(bs[(rsvd+nf*fsz)*bps:data], "", 0, 0)
	}
	for c := int64(2); c < ncl+2; c++ {
		if get(c) != 0 && used[c] == "" {
			t.Fatalf("cluster %d is marked used but nobody owns it", c)
		}
	}
	return files
}

// mkfat16 makes a FAT16 file system by hand, like an old mkfs would.
func mkfat16(dev *memDev, base, size int64) {
	b := dev.b[base:]
	copy(b, []byte{0xEB, 0x3C, 0x90})
	copy(b[3:], "mkfs.fat")
	binary.LittleEndian.PutUint16(b[11:], 512)
	b[13] = 4
	binary.LittleEndian.PutUint16(b[14:], 4)
	b[16] = 2
	binary.LittleEndian.PutUint16(b[17:], 512)
	binary.LittleEndian.PutUint32(b[32:], uint32(size/512))
	b[21] = 0xF8
	binary.LittleEndian.PutUint16(b[22:], 128)
	b[510], b[511] = 0x55, 0xAA
	for k := int64(0); k < 2; k++ {
		f := b[(4+k*128)*512:]
		binary.LittleEndian.PutUint16(f, 0xFFF8)
		binary.LittleEndian.PutUint16(f[2:], 0xFFFF)
	}
}

func TestFormat32AndFiles(t *testing.T) {
	const base, size = 1 << 20, 300 << 20
	dev := &memDev{make([]byte, base+size)}
	f, err := Format32(dev, base, size, "Win 11 x64", 0x1234)
	if err != nil {
		t.Fatal(err)
	}
	f.Now = fixed
	if !f.FAT32() || (f.dataStart%(1<<20)) != 0 {
		t.Fatalf("data area at %d", f.dataStart)
	}
	r := rand.New(rand.NewSource(1))
	want := map[string][]byte{}
	put := func(p string, n int) {
		b := make([]byte, n)
		r.Read(b)
		if err := f.WriteFile(p, b); err != nil {
			t.Fatal(p, err)
		}
		want[p] = b
	}
	put("bootmgr", 400000)
	put("efi/boot/bootx64.efi", 2000000)
	put("efi/microsoft/boot/fonts/segoe_slboot.ttf", 12345)
	put("README.TXT", 10)
	put("cmdline.txt", 70)
	put("autorun.inf", 0)
	// Lots of names in one folder: the folder must grow past one cluster.
	for i := 0; i < 300; i++ {
		put(fmt.Sprintf("sources/a long file name number %03d.dll", i), 100+i)
	}
	off, err := f.CreateContig("sources/boot.wim", 50<<20)
	if err != nil {
		t.Fatal(err)
	}
	wim := make([]byte, 50<<20)
	r.Read(wim)
	copy(dev.b[off:], wim)
	want["sources/boot.wim"] = wim
	put("bootmgr", 1234) // replace with a smaller one: old clusters freed
	if err := f.Flush(); err != nil {
		t.Fatal(err)
	}
	got := check(t, dev, base)
	if len(got) != len(want) {
		t.Fatalf("%d files, want %d", len(got), len(want))
	}
	for p, b := range want {
		if !bytes.Equal(got[p], b) {
			t.Fatalf("%s differs", p)
		}
	}
	// And FS reads back what it wrote, after reopening.
	g, err := Open(dev, base)
	if err != nil {
		t.Fatal(err)
	}
	for p, b := range want {
		rb, err := g.ReadFile(strings.ToUpper(p))
		if err != nil || !bytes.Equal(rb, b) {
			t.Fatalf("ReadFile %s: %v", p, err)
		}
	}
	// 8.3 names: lowercase ones use the NT flags, no long-name entries.
	d, _ := g.openDir(0)
	for _, e := range d.entries() {
		if e.name == "cmdline.txt" && e.first != e.idx {
			t.Fatal("cmdline.txt got long-name entries")
		}
		if e.name == "README.TXT" && e.first != e.idx {
			t.Fatal("README.TXT got long-name entries")
		}
	}
	if _, err := g.CreateContig("big.wim", 5<<30); err == nil {
		t.Fatal("4 GB file accepted")
	}
}

func TestEditFAT16(t *testing.T) {
	const base, size = 4 << 20, 64 << 20
	dev := &memDev{make([]byte, base+size)}
	mkfat16(dev, base, size)
	f, err := Open(dev, base)
	if err != nil {
		t.Fatal(err)
	}
	if f.FAT32() {
		t.Fatal("want FAT16")
	}
	f.Now = fixed
	f.WriteFile("cmdline.txt", []byte("console=serial0,115200 root=PARTUUID=1234-02 rootwait\n"))
	f.WriteFile("config.txt", []byte("dtparam=audio=on\n"))
	f.WriteFile("overlays/README", []byte("x"))
	f.Flush()
	check(t, dev, base)
	g, _ := Open(dev, base)
	g.Now = fixed
	c, _ := g.ReadFile("CMDLINE.TXT")
	g.WriteFile("cmdline.txt", append(bytes.TrimSpace(c), []byte(" systemd.run=/boot/firstrun.sh\n")...))
	g.WriteFile("firstrun.sh", []byte("#!/bin/bash\n"))
	g.WriteFile("user-data", []byte("#cloud-config\n"))
	g.WriteFile("network-config", bytes.Repeat([]byte("x"), 5000))
	g.Flush()
	files := check(t, dev, base)
	if !strings.HasSuffix(string(files["cmdline.txt"]), "firstrun.sh\n") || len(files["network-config"]) != 5000 || string(files["firstrun.sh"]) != "#!/bin/bash\n" {
		t.Fatalf("%q", files)
	}
	if !g.Exists("USER-DATA") || g.Exists("meta-data") {
		t.Fatal("Exists")
	}
	ls, _ := g.List("")
	if strings.Join(ls, ",") != "cmdline.txt,config.txt,overlays,firstrun.sh,user-data,network-config" {
		t.Fatalf("list %v", ls)
	}
}

func TestShortNames(t *testing.T) {
	none := func([11]byte) bool { return false }
	for _, c := range []struct {
		in, short string
		nt        byte
		lfn       bool
	}{
		{"README.TXT", "README  TXT", 0, false},
		{"cmdline.txt", "CMDLINE TXT", 0x18, false},
		{"Config.txt", "CONFIG~1TXT", 0, true},
		{"user-data", "USER-D~1   ", 0, true},
		{"bootx64.efi", "BOOTX64 EFI", 0x18, false},
		{"segoe_slboot.ttf", "SEGOE_~1TTF", 0, true},
		{"a b.c.d", "ABC~1   D  ", 0, true},
	} {
		s, nt, lfn := shortFor(c.in, none)
		if string(s[:]) != c.short || nt != c.nt || lfn != c.lfn {
			t.Errorf("%s: %q %x %v", c.in, s, nt, lfn)
		}
	}
}
