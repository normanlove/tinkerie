// Flashie — UF2 "drag-copy" flashing for RP2040 / RP2350 (and any other
// board with a UF2 bootloader drive). Made by nrmluvdev.

package flashie

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type uf2Drive struct {
	Root    string // e.g. `E:\`
	Letter  string // e.g. `E:`
	Label   string
	Model   string
	BoardID string
	Info    string
}

func (d uf2Drive) isRP2350() bool {
	return strings.Contains(strings.ToUpper(d.Info), "RP2350")
}
func (d uf2Drive) isRP2040() bool {
	u := strings.ToUpper(d.Info)
	return strings.Contains(u, "RPI-RP2") || strings.Contains(u, "RP2040") || (strings.Contains(u, "RASPBERRY PI RP2") && !strings.Contains(u, "RP2350"))
}

func (d uf2Drive) chipName() string {
	switch {
	case d.isRP2350():
		return "RP2350"
	case d.isRP2040():
		return "RP2040"
	}
	if d.Model != "" {
		return d.Model
	}
	return "UF2 bootloader"
}

func parseInfoUF2(root, text string) uf2Drive {
	d := uf2Drive{Root: root, Info: text}
	for _, line := range strings.Split(text, "\n") {
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		switch strings.TrimSpace(strings.ToLower(k)) {
		case "model":
			d.Model = strings.TrimSpace(v)
		case "board-id":
			d.BoardID = strings.TrimSpace(v)
		}
	}
	return d
}

// findUF2Drives looks for mounted UF2 bootloader drives.
func findUF2Drives() []uf2Drive {
	if demoMode {
		return demoDrives()
	}
	return listUF2Drives()
}

func waitForUF2Drive(before map[string]bool, timeout time.Duration) (*uf2Drive, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		for _, d := range findUF2Drives() {
			if !before[d.Root] {
				return &d, nil
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
	return nil, errTimeout
}

// prepareUF2 turns whatever file we were given into UF2 for this drive.
func prepareUF2(name string, data []byte, d uf2Drive, r *reporter) ([]byte, error) {
	switch sniffFormat(name, data) {
	case fmtUF2:
		info, err := inspectUF2(data)
		if err != nil {
			return nil, err
		}
		r.logf("UF2: %d blocks, %s, family: %s", info.Blocks, humanBytes(info.Bytes), info.familyText())
		rp2040File := info.has(famRP2040)
		rp2350File := info.has(famRP2350ARM) || info.has(famRP2350RV) || info.has(famRP2350NS) || info.has(famRP2350Abs)
		if d.isRP2350() && rp2040File && !rp2350File {
			return nil, errors.New("this UF2 was built for RP2040 (Pico / Pico W) but the board is an RP2350 (Pico 2). Download the Pico 2 version of the firmware")
		}
		if d.isRP2040() && rp2350File && !rp2040File {
			return nil, errors.New("this UF2 was built for RP2350 (Pico 2) but the board is an RP2040 (Pico / Pico W). Download the Pico version of the firmware")
		}
		return data, nil
	case fmtHex:
		img, err := parseIntelHex(data)
		if err != nil {
			return nil, err
		}
		fam := uint32(famRP2040)
		if d.isRP2350() {
			fam = famRP2350ARM
		}
		r.logf("Converted HEX at 0x%08X to UF2", img.Base)
		return binToUF2(img.Data, img.Base, fam), nil
	default:
		if !d.isRP2040() && !d.isRP2350() {
			return nil, errors.New("this UF2 drive isn't a Raspberry Pi chip, so Flashie can't convert a .bin for it — use a .uf2 file")
		}
		fam := uint32(famRP2040)
		if d.isRP2350() {
			fam = famRP2350ARM
		}
		r.logf("Converted .bin to UF2 at 0x10000000 (%s)", uf2FamilyNames[fam])
		return binToUF2(data, 0x10000000, fam), nil
	}
}

func copyUF2(d uf2Drive, data []byte, r *reporter) error {
	dst := filepath.Join(d.Root, "FLASHIE.UF2")
	f, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("couldn't write to the %s drive: %v", d.Letter, err)
	}
	r.status("Copying firmware to the board…")
	const chunk = 32 * 1024
	written := 0
	for written < len(data) {
		if r.cancelled() {
			f.Close()
			os.Remove(dst)
			return errCancelled
		}
		end := min(written+chunk, len(data))
		n, err := f.Write(data[written:end])
		written += n
		r.progress(written, len(data))
		if err != nil {
			f.Close()
			if written >= len(data) {
				break // board rebooted as the last block landed
			}
			return fmt.Errorf("copy failed after %s: %v", humanBytes(written), err)
		}
	}
	r.status("Waiting for the board to finish and restart…")
	if err := f.Sync(); err != nil {
		r.logf("Flush: %v (normal if the board already rebooted)", err)
	}
	f.Close()
	if demoMode {
		os.Remove(dst)
		time.Sleep(600 * time.Millisecond)
		return nil
	}
	// The drive disappears when the chip reboots into the new firmware.
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(filepath.Join(d.Root, "INFO_UF2.TXT")); err != nil {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	r.logf("Drive is still mounted after 10 s")
	return errors.New("the file copied, but the board didn't restart. The firmware may be for a different chip, or the file was rejected")
}
