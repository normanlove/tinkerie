// Flashie — flashing a firmware-library build: resolve, download (cached and
// verified), then hand the files to the right protocol. Made by nrmluvdev.

package flashie

import (
	"bytes"
	"errors"
	"fmt"
)

type libFlashReq struct {
	Item     string
	Port     string
	Board    *Board
	BoardID  string
	Chip     string
	Hint     *ChipInfo
	Baud     int
	FlashMB  int
	Compress bool
}

// downloadParts fetches every part of a resolved build into memory.
func downloadParts(res *Resolved, r *reporter) ([]espImage, error) {
	var imgs []espImage
	var appSize int
	for _, p := range res.Parts {
		if p.Kind == "download" && p.Offset == 0x10000 {
			appSize = int(p.Size)
		}
	}
	for _, p := range res.Parts {
		switch p.Kind {
		case "download":
			b, err := fetchPart(p, res.Version, r)
			if err != nil {
				return nil, err
			}
			if p.Offset == 0x10000 {
				appSize = len(b)
			}
			imgs = append(imgs, espImage{Name: shortName(p.Name), Offset: p.Offset, Data: b})
		}
	}
	// Generated parts go in after we know the real app size.
	var out []espImage
	for _, p := range res.Parts {
		switch p.Kind {
		case "ptable":
			table := p.Table
			if appSize > int(appSlotSize(table)) {
				if appSize > int(appSlotSize("huge_app")) {
					return nil, fmt.Errorf("the app is %s — too big for a 4 MB board's partition layout", humanBytes(appSize))
				}
				table = "huge_app"
				r.logf("App is %s, so using the Huge APP partition layout", humanBytes(appSize))
			}
			b, err := buildPartitionTable(table)
			if err != nil {
				return nil, err
			}
			out = append(out, espImage{Name: "partitions (" + table + ")", Offset: p.Offset, Data: b})
		case "fill":
			out = append(out, espImage{Name: "otadata (cleared)", Offset: p.Offset, Data: bytes.Repeat([]byte{0xFF}, p.FillLen)})
		}
	}
	return append(out, imgs...), nil
}

func runLibFlash(req libFlashReq, r *reporter) (msg string, err error) {
	pi, found := findDevice(req.Port)
	if !found {
		return "", fmt.Errorf("%s isn't connected any more — was the board unplugged?", req.Port)
	}
	stopSession()
	defer func() {
		if demoMode && req.Item == "micropython" && err == nil {
			demoInstallMPY(req.Port, req.Chip)
		}
	}()
	res, err := resolveItem(req.Item, req.BoardID, req.Chip)
	if err != nil {
		return "", err
	}
	r.logf("%s %s for %s: %d part(s). %s", res.Name, res.Version, req.Chip, len(res.Parts), res.Verified)
	imgs, err := downloadParts(res, r)
	if err != nil {
		return "", err
	}
	label := res.Name + " " + res.Version
	switch res.Format {
	case "esp":
		if pi.Kind == "drive" {
			return "", errors.New("that's a UF2 drive — pick the board's COM port for ESP firmware")
		}
		freq := flashReq{Port: req.Port, Board: req.Board, Baud: req.Baud, FlashMB: req.FlashMB, Compress: req.Compress}
		msg, err := flashESPImages(pi, sortImages(imgs), freq, r)
		if err != nil {
			return "", err
		}
		return label + ": " + msg, nil
	case "uf2":
		msg, err := flashRP2Job(pi, flashReq{Port: req.Port, Data: imgs[0].Data, FileName: imgs[0].Name}, r)
		if err != nil {
			return "", err
		}
		return label + ": " + msg, nil
	case "hex":
		pl := planFor(pi, req.Board, req.Hint)
		if pl == nil || pl.Family != famAVR {
			pl = &plan{Family: famAVR, Proto: "stk500v1", Bauds: []int{115200, 57600}, Board: req.Board}
		}
		msg, err := flashAVRJob(pi, pl, flashReq{Port: req.Port, Data: imgs[0].Data, FileName: imgs[0].Name}, r)
		if err != nil {
			return "", err
		}
		return label + ": " + msg, nil
	}
	return "", errors.New("unsupported firmware format")
}

func sortImages(imgs []espImage) []espImage {
	out := append([]espImage{}, imgs...)
	for i := 1; i < len(out); i++ {
		for k := i; k > 0 && out[k].Offset < out[k-1].Offset; k-- {
			out[k], out[k-1] = out[k-1], out[k]
		}
	}
	return out
}
