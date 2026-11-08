package udf

import (
	"bytes"
	"math/rand"
	"sort"
	"testing"

	"tinkerie/shared/udf/udftest"
)

func rnd(n int) []byte {
	b := make([]byte, n)
	rand.New(rand.NewSource(int64(n))).Read(b)
	return b
}

type N = udftest.Node

const partStart = 300

func TestReadUDF(t *testing.T) {
	want := map[string][]byte{
		"bootmgr":                rnd(100000),
		"autorun.inf":            []byte("[AutoRun]\nicon=setup.exe\n"),
		"efi/boot/bootx64.efi":   rnd(30000),
		"sources/install.wim":    rnd(3 << 20),
		"sources/boot.wim":       rnd(700000),
		"sources/empty.txt":      nil,
		"sources/ünïcode ☃.txt":  []byte("snow"),
		"sources/extended.entry": rnd(5000),
	}
	root := &N{Dir: true, Kids: []*N{
		{Name: "bootmgr", Data: want["bootmgr"]},
		{Name: "autorun.inf", Data: want["autorun.inf"], Kind: 2},
		{Name: "efi", Dir: true, Kids: []*N{{Name: "boot", Dir: true, Kids: []*N{{Name: "bootx64.efi", Data: want["efi/boot/bootx64.efi"]}}}}},
		{Name: "sources", Dir: true, Kids: []*N{
			{Name: "install.wim", Data: want["sources/install.wim"], Split: 5},
			{Name: "boot.wim", Data: want["sources/boot.wim"]},
			{Name: "empty.txt"},
			{Name: "ünïcode ☃.txt", Data: want["sources/ünïcode ☃.txt"]},
			{Name: "extended.entry", Data: want["sources/extended.entry"], Kind: 1},
		}},
	}}
	img := udftest.Build(root)
	r := bytes.NewReader(img)
	if !IsUDF(r) || IsUDF(bytes.NewReader(make([]byte, 1<<20))) {
		t.Fatal("IsUDF")
	}
	label, files, err := Read(r)
	if err != nil {
		t.Fatal(err)
	}
	if label != "CCCOMA_X64FRE_EN-GB_DV9" {
		t.Fatalf("label %q", label)
	}
	var paths []string
	for _, f := range files {
		paths = append(paths, f.Path)
		if f.Dir {
			continue
		}
		var got []byte
		for _, e := range f.Extents {
			got = append(got, img[e.Off:e.Off+e.Len]...)
		}
		if w, ok := want[f.Path]; !ok || !bytes.Equal(got, w) || f.Size != int64(len(w)) {
			t.Fatalf("%s: wrong contents (%d bytes)", f.Path, len(got))
		}
		if f.Path == "sources/install.wim" && len(f.Extents) < 5 {
			t.Fatalf("install.wim has %d extents", len(f.Extents))
		}
	}
	sort.Strings(paths)
	if len(paths) != len(want)+3 { // + the efi, efi/boot and sources folders
		t.Fatalf("%v", paths)
	}
	// Damage a tag: Read must fail, not crash.
	bad := append([]byte(nil), img...)
	bad[256*sector+4]++
	if _, _, err := Read(bytes.NewReader(bad)); err == nil {
		t.Fatal("damaged anchor accepted")
	}
	for i := 0; i < 200; i++ { // random damage anywhere in the metadata
		b := append([]byte(nil), img...)
		rr := rand.New(rand.NewSource(int64(i)))
		for k := 0; k < 20; k++ {
			b[partStart*sector+rr.Intn(40*sector)] = byte(rr.Intn(256))
		}
		Read(bytes.NewReader(b))
	}
}
