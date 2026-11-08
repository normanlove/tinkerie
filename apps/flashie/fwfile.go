// Flashie — firmware file formats: raw .bin, Intel HEX and UF2.
// Made by nrmluvdev.

package flashie

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
)

type fwFormat int

const (
	fmtBin fwFormat = iota
	fmtHex
	fmtUF2
)

func (f fwFormat) String() string {
	return [...]string{"binary (.bin)", "Intel HEX (.hex)", "UF2"}[f]
}

func sniffFormat(name string, data []byte) fwFormat {
	if len(data) >= 512 && binary.LittleEndian.Uint32(data) == uf2Magic0 && binary.LittleEndian.Uint32(data[4:]) == uf2Magic1 {
		return fmtUF2
	}
	trim := bytes.TrimLeft(data, " \t\r\n\xef\xbb\xbf")
	if len(trim) > 10 && trim[0] == ':' && isHexText(trim[:min(len(trim), 600)]) {
		return fmtHex
	}
	if strings.HasSuffix(strings.ToLower(name), ".hex") && len(trim) > 0 && trim[0] == ':' {
		return fmtHex
	}
	return fmtBin
}

func isHexText(b []byte) bool {
	for _, c := range b {
		switch {
		case c == ':' || c == '\r' || c == '\n':
		case c >= '0' && c <= '9', c >= 'A' && c <= 'F', c >= 'a' && c <= 'f':
		default:
			return false
		}
	}
	return true
}

// memImage is a contiguous chunk of memory starting at Base.
type memImage struct {
	Base uint32
	Data []byte
}

// parseIntelHex returns the file as one contiguous image (gaps = 0xFF).
func parseIntelHex(text []byte) (*memImage, error) {
	type seg struct {
		addr uint32
		data []byte
	}
	var segs []seg
	var upper uint32
	for ln, raw := range strings.Split(string(text), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if line[0] != ':' {
			return nil, fmt.Errorf("HEX line %d doesn't start with ':'", ln+1)
		}
		b, err := hex.DecodeString(line[1:])
		if err != nil || len(b) < 5 {
			return nil, fmt.Errorf("HEX line %d is damaged", ln+1)
		}
		n := int(b[0])
		if len(b) != n+5 {
			return nil, fmt.Errorf("HEX line %d has the wrong length", ln+1)
		}
		var sum byte
		for _, v := range b {
			sum += v
		}
		if sum != 0 {
			return nil, fmt.Errorf("HEX line %d has a bad checksum — the file is corrupted", ln+1)
		}
		addr := uint32(b[1])<<8 | uint32(b[2])
		data := b[4 : 4+n]
		switch b[3] {
		case 0x00:
			segs = append(segs, seg{upper + addr, append([]byte{}, data...)})
		case 0x01:
			goto done
		case 0x02:
			upper = (uint32(data[0])<<8 | uint32(data[1])) << 4
		case 0x04:
			upper = (uint32(data[0])<<8 | uint32(data[1])) << 16
		}
	}
done:
	if len(segs) == 0 {
		return nil, errors.New("the HEX file contains no data")
	}
	sort.Slice(segs, func(i, j int) bool { return segs[i].addr < segs[j].addr })
	lo, hi := segs[0].addr, uint32(0)
	for _, s := range segs {
		if e := s.addr + uint32(len(s.data)); e > hi {
			hi = e
		}
	}
	if hi-lo > 64<<20 {
		return nil, errors.New("the HEX file spans more than 64 MB — it has data at very different addresses")
	}
	img := bytes.Repeat([]byte{0xFF}, int(hi-lo))
	for _, s := range segs {
		copy(img[s.addr-lo:], s.data)
	}
	return &memImage{Base: lo, Data: img}, nil
}

// ------------------------------------------------------------------ UF2 ---

const (
	uf2Magic0   = 0x0A324655
	uf2Magic1   = 0x9E5D5157
	uf2MagicEnd = 0x0AB16F30
	uf2FlagFam  = 0x00002000

	famRP2040    = 0xe48bff56
	famRP2350Abs = 0xe48bff57
	famRP2350Dat = 0xe48bff58
	famRP2350ARM = 0xe48bff59
	famRP2350RV  = 0xe48bff5a
	famRP2350NS  = 0xe48bff5b
)

var uf2FamilyNames = map[uint32]string{
	famRP2040: "RP2040", famRP2350Abs: "RP2350 (absolute)", famRP2350Dat: "RP2350 (data)",
	famRP2350ARM: "RP2350 (ARM)", famRP2350RV: "RP2350 (RISC-V)", famRP2350NS: "RP2350 (ARM non-secure)",
	0x1c5f21b0: "ESP32", 0xbfdd4eee: "ESP32-S2", 0xc47e5767: "ESP32-S3", 0xd42ba06c: "ESP32-C3",
	0xada52840: "nRF52840", 0x68ed2b88: "SAMD21", 0x55114460: "SAMD51", 0x5ee21072: "STM32F1", 0x57755a57: "STM32F4",
}

type uf2Info struct {
	Blocks   int
	Families []uint32
	Bytes    int
}

func inspectUF2(data []byte) (*uf2Info, error) {
	if len(data)%512 != 0 {
		return nil, errors.New("the UF2 file is truncated (size isn't a multiple of 512 bytes)")
	}
	info := &uf2Info{Blocks: len(data) / 512}
	seen := map[uint32]bool{}
	for i := 0; i < len(data); i += 512 {
		b := data[i : i+512]
		if binary.LittleEndian.Uint32(b) != uf2Magic0 || binary.LittleEndian.Uint32(b[4:]) != uf2Magic1 ||
			binary.LittleEndian.Uint32(b[508:]) != uf2MagicEnd {
			return nil, fmt.Errorf("UF2 block %d is damaged", i/512)
		}
		flags := binary.LittleEndian.Uint32(b[8:])
		info.Bytes += int(binary.LittleEndian.Uint32(b[16:]))
		if flags&uf2FlagFam != 0 {
			f := binary.LittleEndian.Uint32(b[28:])
			if !seen[f] {
				seen[f] = true
				info.Families = append(info.Families, f)
			}
		}
	}
	return info, nil
}

func (u *uf2Info) has(f uint32) bool {
	for _, v := range u.Families {
		if v == f {
			return true
		}
	}
	return false
}

func (u *uf2Info) familyText() string {
	var s []string
	for _, f := range u.Families {
		if n, ok := uf2FamilyNames[f]; ok {
			s = append(s, n)
		} else {
			s = append(s, fmt.Sprintf("0x%08X", f))
		}
	}
	if len(s) == 0 {
		return "no family tag"
	}
	return strings.Join(s, ", ")
}

// binToUF2 wraps a raw image (loaded at base) into UF2 blocks.
func binToUF2(data []byte, base, family uint32) []byte {
	const payload = 256
	n := (len(data) + payload - 1) / payload
	out := make([]byte, 0, n*512)
	for i := 0; i < n; i++ {
		b := make([]byte, 512)
		binary.LittleEndian.PutUint32(b[0:], uf2Magic0)
		binary.LittleEndian.PutUint32(b[4:], uf2Magic1)
		binary.LittleEndian.PutUint32(b[8:], uf2FlagFam)
		binary.LittleEndian.PutUint32(b[12:], base+uint32(i*payload))
		binary.LittleEndian.PutUint32(b[16:], payload)
		binary.LittleEndian.PutUint32(b[20:], uint32(i))
		binary.LittleEndian.PutUint32(b[24:], uint32(n))
		binary.LittleEndian.PutUint32(b[28:], family)
		end := (i + 1) * payload
		if end > len(data) {
			end = len(data)
		}
		chunk := data[i*payload : end]
		copy(b[32:], chunk)
		for j := 32 + len(chunk); j < 32+payload; j++ {
			b[j] = 0xFF
		}
		binary.LittleEndian.PutUint32(b[508:], uf2MagicEnd)
		out = append(out, b...)
	}
	return out
}
