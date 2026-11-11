//go:build linux

package flashie

// android_usb.go: drives on Android. Android doesn't let apps reach USB
// drives as files or block devices, so Flashie talks to them directly over
// USB (shared/usb's mass-storage driver):
//   - a board in UF2 bootloader mode (Raspberry Pi Pico and friends) gets its
//     UF2 blocks written straight to the drive; the bootloader picks out
//     every UF2 block written to it, wherever it lands, so no file system work
//     is needed.
//   - SD card readers and USB sticks show up in the Imager as drives.

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"tinkerie/shared/usb"
)

const usbDrivePrefix = "usbdrive:"

// UF2 bootloaders Flashie recognises by their USB IDs.
func uf2Boot(in usb.Info) (info string, ok bool) {
	if _, msc := in.IsMassStorage(); !msc {
		return "", false
	}
	switch {
	case in.VID == 0x2E8A && in.PID == 0x0003:
		return "UF2 Bootloader\nModel: Raspberry Pi RP2\nBoard-ID: RPI-RP2", true
	case in.VID == 0x2E8A && in.PID == 0x000F:
		return "UF2 Bootloader\nModel: Raspberry Pi RP2350\nBoard-ID: RP2350", true
	case strings.Contains(strings.ToUpper(in.Product+in.Manufacturer), "UF2"):
		return "UF2 Bootloader\nModel: " + in.Product, true
	}
	return "", false
}

func androidUF2Drives() []uf2Drive {
	var out []uf2Drive
	for _, in := range usb.List() {
		text, ok := uf2Boot(in)
		if !ok {
			continue
		}
		d := parseInfoUF2(usbDrivePrefix+in.Path, text)
		d.Label = "RPI-RP2"
		if d.isRP2350() {
			d.Label = "RP2350"
		} else if !d.isRP2040() {
			d.Label = "UF2 drive"
		}
		d.Letter = d.Label
		out = append(out, d)
	}
	return out
}

func usbInfo(path string) (usb.Info, bool) {
	for _, in := range usb.List() {
		if in.Path == path {
			return in, true
		}
	}
	return usb.Info{}, false
}

// copyUF2USB writes the UF2 file's blocks into the drive's data area.
func copyUF2USB(d uf2Drive, data []byte, r *reporter) error {
	path := strings.TrimPrefix(d.Root, usbDrivePrefix)
	in, ok := usbInfo(path)
	if !ok {
		return errors.New("the board's drive has gone: hold BOOTSEL and plug it in again")
	}
	r.status("Asking Android for the board's drive…")
	dr, err := usb.OpenDrive(in)
	if err != nil {
		return fmt.Errorf("couldn't open the %s drive: %v", d.Letter, err)
	}
	defer dr.Close()
	if dr.BlockSize != 512 {
		return fmt.Errorf("the drive uses %d-byte blocks; UF2 needs 512", dr.BlockSize)
	}
	// put the blocks where a file would go: after the boot sector, FATs and root folder
	start := int64(1024)
	boot := make([]byte, 512)
	if _, err := dr.ReadAt(boot, 0); err == nil && boot[510] == 0x55 && boot[511] == 0xAA {
		bps := int64(binary.LittleEndian.Uint16(boot[11:]))
		res := int64(binary.LittleEndian.Uint16(boot[14:]))
		fats := int64(boot[16])
		roots := int64(binary.LittleEndian.Uint16(boot[17:]))
		fatSz := int64(binary.LittleEndian.Uint16(boot[22:]))
		if bps == 512 && fats > 0 && fatSz > 0 {
			start = res + fats*fatSz + (roots*32+511)/512 + int64(boot[13])*2 // leave the first two clusters alone
		}
	}
	if total := start + int64(len(data)/512); total > dr.Blocks {
		start = max(dr.Blocks-int64(len(data)/512), 1)
	}
	if len(data)%512 != 0 {
		data = append(data, make([]byte, 512-len(data)%512)...)
	}
	r.status("Copying firmware to the board…")
	const chunk = 32 * 512
	for off := 0; off < len(data); off += chunk {
		if r.cancelled() {
			return errCancelled
		}
		end := min(off+chunk, len(data))
		if _, err := dr.WriteAt(data[off:end], (start*512)+int64(off)); err != nil {
			if end >= len(data) {
				break // the board restarted as the last block landed
			}
			return fmt.Errorf("copy failed after %s: %v", humanBytes(off), err)
		}
		r.progress(end, len(data))
	}
	dr.Sync()
	r.status("Waiting for the board to finish and restart…")
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if _, ok := usbInfo(path); !ok {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return errors.New("the firmware copied, but the board didn't restart. It may be for a different chip")
}

// ------------------------------------------------------------- Imager drives

type diskInfo struct {
	size            int64
	vendor, product string
}

var (
	diskMu    sync.Mutex
	diskSizes = map[string]diskInfo{}
	diskAsked = map[string]bool{}
)

func androidDisks() ([]Disk, error) {
	var out []Disk
	for _, in := range usb.List() {
		if _, ok := in.IsMassStorage(); !ok {
			continue
		}
		if _, boot := uf2Boot(in); boot {
			continue
		}
		diskMu.Lock()
		info, known := diskSizes[in.Path]
		asked := diskAsked[in.Path]
		diskAsked[in.Path] = true
		diskMu.Unlock()
		if !known {
			if !asked { // ask once: Android shows its "allow access" box, then we read the size
				go func(in usb.Info) {
					if dr, err := usb.OpenDrive(in); err == nil {
						diskMu.Lock()
						diskSizes[in.Path] = diskInfo{dr.Size(), dr.Vendor, dr.ProductStr}
						diskMu.Unlock()
						dr.Close()
					} else {
						diskMu.Lock()
						delete(diskAsked, in.Path) // try again on the next look
						diskMu.Unlock()
					}
				}(in)
			}
			continue
		}
		name := strings.TrimSpace(in.Manufacturer + " " + in.Product)
		if name == "" {
			name = strings.TrimSpace(info.vendor + " " + info.product)
		}
		if name == "" {
			name = "USB drive"
		}
		out = append(out, Disk{ID: usbDrivePrefix + in.Path, Name: name, Size: info.size, Bus: "USB"})
	}
	// forget drives that were unplugged
	diskMu.Lock()
	for p := range diskSizes {
		if _, ok := usbInfo(p); !ok {
			delete(diskSizes, p)
			delete(diskAsked, p)
		}
	}
	diskMu.Unlock()
	return out, nil
}

type usbRawDisk struct{ *usb.Drive }

func (u usbRawDisk) Sync() error  { return u.Drive.Sync() }
func (u usbRawDisk) Close() error { return u.Drive.Close() }

func androidOpenDisk(d Disk) (rawDisk, error) {
	in, ok := usbInfo(strings.TrimPrefix(d.ID, usbDrivePrefix))
	if !ok {
		return nil, errors.New("the drive was unplugged")
	}
	dr, err := usb.OpenDrive(in)
	if err != nil {
		return nil, err
	}
	if dr.BlockSize != 512 {
		dr.Close()
		return nil, fmt.Errorf("the drive uses %d-byte blocks, which Flashie can't write", dr.BlockSize)
	}
	return usbRawDisk{dr}, nil
}

func androidFinishDisk(d Disk, eject bool) {
	diskMu.Lock()
	delete(diskSizes, strings.TrimPrefix(d.ID, usbDrivePrefix))
	delete(diskAsked, strings.TrimPrefix(d.ID, usbDrivePrefix))
	diskMu.Unlock()
}
