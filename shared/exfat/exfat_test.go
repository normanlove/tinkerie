package exfat

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

type memDev struct{ b []byte }

func (m *memDev) ReadAt(p []byte, off int64) (int, error)  { return copy(p, m.b[off:]), nil }
func (m *memDev) WriteAt(p []byte, off int64) (int, error) { return copy(m.b[off:], p), nil }

// read is an independent reader, written from the specification's rules
// (it reads by following FAT chains, which Build never needs to).
func read(t *testing.T, img []byte, base int64) map[string][]byte {
	t.Helper()
	v := img[base:]
	u32 := func(b []byte) uint32 { return binary.LittleEndian.Uint32(b) }
	if string(v[3:11]) != "EXFAT   " || v[510] != 0x55 || v[511] != 0xAA {
		t.Fatal("not exFAT")
	}
	for i := 11; i < 64; i++ {
		if v[i] != 0 {
			t.Fatal("MustBeZero area not zero")
		}
	}
	if binary.LittleEndian.Uint64(v[64:]) != uint64(base/512) {
		t.Fatal("PartitionOffset")
	}
	var sum uint32
	for i := 0; i < 11*512; i++ {
		if i != 106 && i != 107 && i != 112 {
			if sum&1 != 0 {
				sum = 0x80000000 + sum>>1 + uint32(v[i])
			} else {
				sum = sum>>1 + uint32(v[i])
			}
		}
	}
	for i := 0; i < 128; i++ {
		if u32(v[11*512+i*4:]) != sum {
			t.Fatal("boot checksum wrong")
		}
	}
	if !bytes.Equal(v[:12*512], v[12*512:24*512]) {
		t.Fatal("backup boot region differs")
	}
	for s := 1; s <= 8; s++ {
		if u32(v[s*512+508:]) != 0xAA550000 {
			t.Fatal("extended boot signature")
		}
	}
	fatOff, fatLen, heap, cc, rootC := int64(u32(v[80:])), int64(u32(v[84:])), int64(u32(v[88:])), int64(u32(v[92:])), u32(v[96:])
	if fatOff < 24 || heap < fatOff+fatLen || (cc+2)*4 > fatLen*512 || heap*512+cc<<v[109]*512 > int64(binary.LittleEndian.Uint64(v[72:]))*512 {
		t.Fatal("layout fields inconsistent")
	}
	cs := int64(512) << v[109]
	fat := v[fatOff*512:]
	if u32(fat) != 0xFFFFFFF8 || u32(fat[4:]) != 0xFFFFFFFF {
		t.Fatal("FAT media entries")
	}
	owner := map[uint32]string{}
	chain := func(c uint32, n int64, who string) []byte {
		var out []byte
		for k := int64(0); k < (n+cs-1)/cs; k++ {
			if c < 2 || int64(c) >= cc+2 {
				t.Fatalf("%s: bad cluster %d", who, c)
			}
			if owner[c] != "" {
				t.Fatalf("%s: cluster %d also used by %s", who, c, owner[c])
			}
			owner[c] = who
			o := heap*512 + int64(c-2)*cs
			out = append(out, v[o:o+cs]...)
			c = u32(fat[c*4:])
		}
		if n > 0 && c != 0xFFFFFFFF {
			t.Fatalf("%s: chain doesn't end (%x)", who, c)
		}
		return out[:n]
	}
	var bitmap []byte
	var up []uint16
	files := map[string][]byte{}
	var walk func(d []byte, path string)
	walk = func(d []byte, path string) {
		for i := 0; i < len(d)/32; i++ {
			e := d[i*32 : i*32+32]
			switch e[0] {
			case 0:
				return
			case 0x83:
			case 0x81:
				bitmap = chain(u32(e[20:]), int64(binary.LittleEndian.Uint64(e[24:])), "bitmap")
			case 0x82:
				data := chain(u32(e[20:]), int64(binary.LittleEndian.Uint64(e[24:])), "upcase")
				var s uint32
				for _, b := range data {
					s = (s>>1 | s<<31) + uint32(b)
				}
				if s != u32(e[4:]) {
					t.Fatal("up-case checksum")
				}
				up = make([]uint16, len(data)/2)
				for k := range up {
					up[k] = binary.LittleEndian.Uint16(data[k*2:])
				}
			case 0x85:
				n := int(e[1])
				set := d[i*32 : (i+1+n)*32]
				var s uint16
				for k, b := range set {
					if k != 2 && k != 3 {
						s = (s>>1 | s<<15) + uint16(b)
					}
				}
				if s != binary.LittleEndian.Uint16(e[2:]) {
					t.Fatal("entry set checksum")
				}
				st := set[32:]
				if st[0] != 0xC0 {
					t.Fatal("no stream extension")
				}
				nl := int(st[3])
				var name []uint16
				for k := 0; k < n-1; k++ {
					fe := set[(2+k)*32:]
					if fe[0] != 0xC1 {
						t.Fatal("no file name entry")
					}
					for j := 0; j < 15; j++ {
						name = append(name, binary.LittleEndian.Uint16(fe[2+j*2:]))
					}
				}
				name = name[:nl]
				var h uint16
				for _, c := range name {
					c = up[c]
					h = (h>>1 | h<<15) + c&0xFF
					h = (h>>1 | h<<15) + c>>8
				}
				if h != binary.LittleEndian.Uint16(st[4:]) {
					t.Fatal("name hash")
				}
				full := path + "/" + string(utf16.Decode(name))
				size := int64(binary.LittleEndian.Uint64(st[24:]))
				if int64(binary.LittleEndian.Uint64(st[8:])) != size {
					t.Fatal("ValidDataLength")
				}
				data := chain(u32(st[20:]), size, full)
				if binary.LittleEndian.Uint16(e[4:])&0x10 != 0 {
					walk(data, full)
				} else {
					files[full[1:]] = data
				}
				i += n
			default:
				t.Fatalf("unexpected entry type %x", e[0])
			}
		}
	}
	walk(chain(rootC, cs, "root"), "")
	if bitmap == nil || up == nil {
		t.Fatal("no bitmap or up-case table")
	}
	for c := int64(2); c < cc+2; c++ {
		set := bitmap[(c-2)/8]&(1<<((c-2)%8)) != 0
		if set != (owner[uint32(c)] != "") {
			t.Fatalf("bitmap wrong for cluster %d", c)
		}
	}
	return files
}

func TestBuild(t *testing.T) {
	const base, size = 1 << 20, 600 << 20
	dev := &memDev{make([]byte, base+size)}
	r := rand.New(rand.NewSource(3))
	want := map[string][]byte{}
	var list []File
	add := func(p string, n int) {
		b := make([]byte, n)
		r.Read(b)
		want[p] = b
		list = append(list, File{Path: p, Size: int64(n)})
	}
	add("bootmgr", 400000)
	add("sources/install.wim", 300<<20)
	add("sources/boot.wim", 20<<20)
	add("efi/boot/bootx64.efi", 2<<20)
	add("Sources/empty.cfg", 0) // same folder, other case
	add("a very long file name that needs more than one name entry, really.txt", 100)
	add("ünïcödé ☃.txt", 5)
	for i := 0; i < 2000; i++ { // a folder bigger than one cluster
		add(fmt.Sprintf("sources/en-gb/file%04d.mui", i), 10+i%300)
	}
	list = append(list, File{Path: "support/empty folder", Dir: true})
	offs, err := Build(dev, base, size, "CCCOMA_X64FRE", list, time.Date(2026, 11, 8, 15, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	for p, o := range offs {
		if o%4096 != 0 {
			t.Fatalf("%s not 4 KiB aligned", p)
		}
		copy(dev.b[o:], want[p])
	}
	got := read(t, dev.b, base)
	if len(got) != len(want) {
		t.Fatalf("%d files, want %d", len(got), len(want))
	}
	for p, b := range want {
		g, ok := got[p]
		if !ok {
			g = got[strings.Replace(p, "Sources", "sources", 1)]
		}
		if !bytes.Equal(g, b) {
			t.Fatalf("%s differs", p)
		}
	}
	for _, p := range []string{"bootmgr", "SOURCES/install.wim", "sources/en-gb/file1999.mui"} {
		o, n, err := Lookup(dev, base, p)
		w := want[strings.Replace(p, "SOURCES", "sources", 1)]
		if err != nil || n != int64(len(w)) || !bytes.Equal(dev.b[o:o+n], w) {
			t.Fatalf("Lookup %s: %v", p, err)
		}
	}
	if _, _, err := Lookup(dev, base, "nope"); err == nil {
		t.Fatal("Lookup found a missing file")
	}
	if _, err := Build(dev, base, 64<<20, "x", []File{{Path: "big", Size: 100 << 20}}, time.Now()); err == nil {
		t.Fatal("too-big file accepted")
	}
	if _, err := Build(dev, base, size, "x", []File{{Path: "a:b", Size: 1}}, time.Now()); err == nil {
		t.Fatal("bad name accepted")
	}
}
