package exfat

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf16"
)

// Lookup finds a file on an exFAT volume at base and returns where its data
// starts on the device and its size. Files must be contiguous (as Build
// makes them). Used to check a written drive.
func Lookup(dev Device, base int64, path string) (int64, int64, error) {
	b := make([]byte, 512)
	if _, err := dev.ReadAt(b, base); err != nil {
		return 0, 0, err
	}
	if string(b[3:11]) != "EXFAT   " {
		return 0, 0, errors.New("not exFAT")
	}
	heap, root := int64(le.Uint32(b[88:])), le.Uint32(b[96:])
	cs := int64(512) << b[109]
	off := func(c uint32) int64 { return base + heap*512 + int64(c-2)*cs }
	fatAt := base + int64(le.Uint32(b[80:]))*512
	readChain := func(c uint32, n int64) ([]byte, error) {
		var out []byte
		for k := int64(0); k < (n+cs-1)/cs && k < 1<<16; k++ {
			buf := make([]byte, cs)
			if _, err := dev.ReadAt(buf, off(c)); err != nil {
				return nil, err
			}
			out = append(out, buf...)
			e := make([]byte, 4)
			if _, err := dev.ReadAt(e, fatAt+int64(c)*4); err != nil {
				return nil, err
			}
			c = le.Uint32(e)
		}
		return out, nil
	}
	dir, err := readChain(root, cs)
	if err != nil {
		return 0, 0, err
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for depth, want := range parts {
		found := false
		for i := 0; i+32 <= len(dir); i += 32 {
			if dir[i] == 0 {
				break
			}
			if dir[i] != 0x85 {
				continue
			}
			n := int(dir[i+1])
			if i+(n+1)*32 > len(dir) {
				break
			}
			st := dir[i+32:]
			var name []uint16
			for k := 0; k < n-1; k++ {
				for j := 0; j < 15; j++ {
					name = append(name, le.Uint16(st[32+k*32+2+j*2:]))
				}
			}
			name = name[:min(len(name), int(st[3]))]
			if !strings.EqualFold(string(utf16.Decode(name)), want) {
				continue
			}
			first, size := le.Uint32(st[20:]), int64(le.Uint64(st[24:]))
			if depth == len(parts)-1 {
				if size == 0 {
					return -1, 0, nil
				}
				return off(first), size, nil
			}
			if dir, err = readChain(first, size); err != nil {
				return 0, 0, err
			}
			found = true
			break
		}
		if !found {
			return 0, 0, fmt.Errorf("%s: not found", path)
		}
	}
	return 0, 0, fmt.Errorf("%s: not found", path)
}
