// mksyso writes a COFF resource object (.syso) containing the app icon, a
// DPI-aware manifest and version info, which the Go linker embeds into the .exe.
//
// Usage (from the repository root):
//
//	go run ./tools/mksyso -ico build/tinkerie.ico -out rsrc_windows_amd64.syso -version 0.15.0-alpha
package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"unicode/utf16"
)

const manifest = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<assembly xmlns="urn:schemas-microsoft-com:asm.v1" manifestVersion="1.0">
  <assemblyIdentity type="win32" name="nrmluvdev.Tinkerie" version="VERSION4" processorArchitecture="amd64"/>
  <description>Tinkerie - maker tool suite by nrmluvdev</description>
  <compatibility xmlns="urn:schemas-microsoft-com:compatibility.v1">
    <application>
      <supportedOS Id="{8e0f7a12-bfb3-4fe8-b9a5-48fd50a15a9a}"/>
    </application>
  </compatibility>
  <application xmlns="urn:schemas-microsoft-com:asm.v3">
    <windowsSettings>
      <dpiAware xmlns="http://schemas.microsoft.com/SMI/2005/WindowsSettings">true/pm</dpiAware>
      <dpiAwareness xmlns="http://schemas.microsoft.com/SMI/2016/WindowsSettings">PerMonitorV2</dpiAwareness>
    </windowsSettings>
  </application>
  <trustInfo xmlns="urn:schemas-microsoft-com:asm.v3">
    <security><requestedPrivileges><requestedExecutionLevel level="asInvoker" uiAccess="false"/></requestedPrivileges></security>
  </trustInfo>
</assembly>
`

type res struct {
	typ, id uint32
	data    []byte
}

func main() {
	icoPath := flag.String("ico", "build/tinkerie.ico", "icon file")
	out := flag.String("out", "rsrc_windows_amd64.syso", "output .syso")
	ver := flag.String("version", "0.1.0", "version (e.g. 0.6.0-alpha)")
	flag.Parse()
	ico, err := os.ReadFile(*icoPath)
	if err != nil {
		fail(err)
	}
	var rs []res
	// --- icons
	n := int(binary.LittleEndian.Uint16(ico[4:]))
	var grp bytes.Buffer
	binary.Write(&grp, binary.LittleEndian, [3]uint16{0, 1, uint16(n)})
	for i := 0; i < n; i++ {
		e := ico[6+16*i:]
		size := binary.LittleEndian.Uint32(e[8:])
		off := binary.LittleEndian.Uint32(e[12:])
		id := uint32(i + 1)
		rs = append(rs, res{3, id, ico[off : off+size]})
		grp.Write(e[:12]) // w,h,colors,reserved,planes,bitcount,bytesInRes
		binary.Write(&grp, binary.LittleEndian, uint16(id))
	}
	rs = append(rs, res{14, 1, grp.Bytes()})
	rs = append(rs, res{16, 1, versionInfo(*ver)})
	var va, vb, vc int
	fmt.Sscanf(*ver, "%d.%d.%d", &va, &vb, &vc) // "0.6.0-alpha" -> 0.6.0
	rs = append(rs, res{24, 1, []byte(strings.Replace(manifest, "VERSION4", fmt.Sprintf("%d.%d.%d.0", va, vb, vc), 1))})
	if err := os.WriteFile(*out, coff(rs), 0o644); err != nil {
		fail(err)
	}
	fmt.Println("wrote", *out)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

// ---------------------------------------------------------------------------
// VS_VERSIONINFO

func utf16z(s string) []byte {
	u := utf16.Encode([]rune(s))
	b := make([]byte, 0, len(u)*2+2)
	for _, c := range u {
		b = append(b, byte(c), byte(c>>8))
	}
	return append(b, 0, 0)
}

func pad4(b []byte) []byte {
	for len(b)%4 != 0 {
		b = append(b, 0)
	}
	return b
}

func vnode(key string, typ uint16, value []byte, valueLen uint16, children ...[]byte) []byte {
	b := []byte{0, 0, byte(valueLen), byte(valueLen >> 8), byte(typ), byte(typ >> 8)}
	b = append(b, utf16z(key)...)
	b = pad4(b)
	b = append(b, value...)
	for _, c := range children {
		b = pad4(b)
		b = append(b, c...)
	}
	binary.LittleEndian.PutUint16(b, uint16(len(b)))
	return b
}

func vstring(k, v string) []byte {
	val := utf16z(v)
	return vnode(k, 1, val, uint16(len(val)/2))
}

func versionInfo(ver string) []byte {
	var a, b, c int
	fmt.Sscanf(ver, "%d.%d.%d", &a, &b, &c)
	ms := uint32(a)<<16 | uint32(b)
	ls := uint32(c) << 16
	var fixed bytes.Buffer
	var flags uint32
	if strings.Contains(ver, "-") {
		flags = 0x2 // VS_FF_PRERELEASE: an alpha or beta build
	}
	binary.Write(&fixed, binary.LittleEndian, []uint32{0xFEEF04BD, 0x00010000, ms, ls, ms, ls, 0x3F, flags, 0x40004, 1, 0, 0, 0})
	strs := vnode("040904B0", 1, nil, 0,
		vstring("CompanyName", "nrmluvdev"),
		vstring("FileDescription", "Tinkerie - maker tool suite"),
		vstring("FileVersion", ver),
		vstring("InternalName", "Tinkerie"),
		vstring("LegalCopyright", "© nrmluvdev - free for hobbyists"),
		vstring("OriginalFilename", "Tinkerie.exe"),
		vstring("ProductName", "Tinkerie"), // must match the SignPath artifact configuration
		vstring("ProductVersion", ver),
	)
	sfi := vnode("StringFileInfo", 1, nil, 0, strs)
	trans := []byte{0x09, 0x04, 0xB0, 0x04}
	vfi := vnode("VarFileInfo", 1, nil, 0, vnode("Translation", 0, trans, 4))
	return vnode("VS_VERSION_INFO", 0, fixed.Bytes(), uint16(fixed.Len()), sfi, vfi)
}

// ---------------------------------------------------------------------------
// COFF object with a .rsrc section

func coff(rs []res) []byte {
	sort.Slice(rs, func(i, j int) bool {
		if rs[i].typ != rs[j].typ {
			return rs[i].typ < rs[j].typ
		}
		return rs[i].id < rs[j].id
	})
	// group by type
	var types []uint32
	byType := map[uint32][]res{}
	for _, r := range rs {
		if len(byType[r.typ]) == 0 {
			types = append(types, r.typ)
		}
		byType[r.typ] = append(byType[r.typ], r)
	}
	dirSize := func(entries int) int { return 16 + 8*entries }
	// layout: root dir, type dirs, name dirs, lang dirs, data entries, data
	off := dirSize(len(types))
	typeDirOff := map[uint32]int{}
	for _, t := range types {
		typeDirOff[t] = off
		off += dirSize(len(byType[t]))
	}
	nameDirOff := map[[2]uint32]int{}
	for _, t := range types {
		for _, r := range byType[t] {
			nameDirOff[[2]uint32{t, r.id}] = off
			off += dirSize(1)
		}
	}
	entryOff := map[[2]uint32]int{}
	for _, t := range types {
		for _, r := range byType[t] {
			entryOff[[2]uint32{t, r.id}] = off
			off += 16
		}
	}
	dataOff := map[[2]uint32]int{}
	for _, t := range types {
		for _, r := range byType[t] {
			off = (off + 7) &^ 7
			dataOff[[2]uint32{t, r.id}] = off
			off += len(r.data)
		}
	}
	sec := make([]byte, (off+7)&^7)
	le := binary.LittleEndian
	writeDir := func(at, n int) {
		le.PutUint16(sec[at+14:], uint16(n)) // NumberOfIdEntries
	}
	writeDir(0, len(types))
	for i, t := range types {
		e := 16 + 8*i
		le.PutUint32(sec[e:], t)
		le.PutUint32(sec[e+4:], 0x80000000|uint32(typeDirOff[t]))
		writeDir(typeDirOff[t], len(byType[t]))
		for j, r := range byType[t] {
			k := [2]uint32{t, r.id}
			e2 := typeDirOff[t] + 16 + 8*j
			le.PutUint32(sec[e2:], r.id)
			le.PutUint32(sec[e2+4:], 0x80000000|uint32(nameDirOff[k]))
			writeDir(nameDirOff[k], 1)
			e3 := nameDirOff[k] + 16
			le.PutUint32(sec[e3:], 0x0409)
			le.PutUint32(sec[e3+4:], uint32(entryOff[k]))
			de := entryOff[k]
			le.PutUint32(sec[de:], uint32(dataOff[k])) // relocated to an RVA by the linker
			le.PutUint32(sec[de+4:], uint32(len(r.data)))
			copy(sec[dataOff[k]:], r.data)
		}
	}
	// relocations: one per data entry
	var relocs bytes.Buffer
	nrel := 0
	for _, t := range types {
		for _, r := range byType[t] {
			binary.Write(&relocs, le, uint32(entryOff[[2]uint32{t, r.id}]))
			binary.Write(&relocs, le, uint32(0)) // symbol 0 (.rsrc)
			binary.Write(&relocs, le, uint16(3)) // IMAGE_REL_AMD64_ADDR32NB
			nrel++
		}
	}
	const hdr = 20 + 40
	rawPtr := hdr
	relPtr := rawPtr + len(sec)
	symPtr := relPtr + relocs.Len()
	var b bytes.Buffer
	binary.Write(&b, le, uint16(0x8664))
	binary.Write(&b, le, uint16(1))
	binary.Write(&b, le, uint32(0))
	binary.Write(&b, le, uint32(symPtr))
	binary.Write(&b, le, uint32(1))
	binary.Write(&b, le, uint16(0))
	binary.Write(&b, le, uint16(0))
	b.WriteString(".rsrc\x00\x00\x00")
	binary.Write(&b, le, uint32(0))
	binary.Write(&b, le, uint32(0))
	binary.Write(&b, le, uint32(len(sec)))
	binary.Write(&b, le, uint32(rawPtr))
	binary.Write(&b, le, uint32(relPtr))
	binary.Write(&b, le, uint32(0))
	binary.Write(&b, le, uint16(nrel))
	binary.Write(&b, le, uint16(0))
	binary.Write(&b, le, uint32(0x40000040))
	b.Write(sec)
	b.Write(relocs.Bytes())
	b.WriteString(".rsrc\x00\x00\x00")
	binary.Write(&b, le, uint32(0))
	binary.Write(&b, le, int16(1))
	binary.Write(&b, le, uint16(0))
	b.WriteByte(3)
	b.WriteByte(0)
	binary.Write(&b, le, uint32(4)) // empty string table
	return b.Bytes()
}
