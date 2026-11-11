//go:build linux

package usb

// msc.go: USB drives (card readers, flash drives, UF2 bootloaders) spoken to
// directly with the USB mass-storage "bulk-only" protocol and SCSI commands.
// Android doesn't let apps reach drives as block devices, but it does hand
// over the USB device, so Flashie can still write disk images and UF2
// firmware there. Created by nrmluvdev for Tinkerie.

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"
)

// IsMassStorage reports whether a device has a bulk-only mass-storage interface.
func (in Info) IsMassStorage() (Interface, bool) {
	for _, it := range in.Interfaces {
		if it.Class == 8 && it.Sub == 6 && len(it.BulkIn) > 0 && len(it.BulkOut) > 0 {
			return it, true
		}
	}
	return Interface{}, false
}

// Drive is an open USB drive.
type Drive struct {
	d          *Device
	in, out    uint8
	lun        uint8
	tag        uint32
	BlockSize  int
	Blocks     int64
	Vendor     string
	ProductStr string
	mu         sync.Mutex
}

// OpenDrive opens a USB drive and reads its size.
func OpenDrive(info Info) (*Drive, error) {
	it, ok := info.IsMassStorage()
	if !ok {
		return nil, errors.New("this USB device isn't a drive")
	}
	d, err := Open(info.Path, it.Number)
	if err != nil {
		return nil, err
	}
	dr := &Drive{d: d, in: it.BulkIn[0], out: it.BulkOut[0]}
	inq := make([]byte, 36)
	if _, err := dr.cmd([]byte{0x12, 0, 0, 0, 36, 0}, inq, true, 5000); err == nil {
		dr.Vendor, dr.ProductStr = trimASCII(inq[8:16]), trimASCII(inq[16:32])
	}
	// a card reader with no card, or one still waking up, says "not ready" for a while
	var lastErr error
	for i := 0; i < 20; i++ {
		if _, lastErr = dr.cmd([]byte{0, 0, 0, 0, 0, 0}, nil, false, 3000); lastErr == nil {
			break
		}
		dr.cmd([]byte{0x03, 0, 0, 0, 18, 0}, make([]byte, 18), true, 3000) // request sense clears the condition
		time.Sleep(150 * time.Millisecond)
	}
	cap := make([]byte, 8)
	if _, err := dr.cmd([]byte{0x25, 0, 0, 0, 0, 0, 0, 0, 0, 0}, cap, true, 5000); err != nil {
		d.Close()
		if lastErr != nil {
			return nil, fmt.Errorf("the drive isn't ready (is a card in the reader?): %v", lastErr)
		}
		return nil, fmt.Errorf("couldn't read the drive's size: %v", err)
	}
	last, bs := binary.BigEndian.Uint32(cap), binary.BigEndian.Uint32(cap[4:])
	if last == 0xFFFFFFFF { // over 2 TB: READ CAPACITY(16)
		c16 := make([]byte, 32)
		if _, err := dr.cmd([]byte{0x9E, 0x10, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 32, 0, 0}, c16, true, 5000); err == nil {
			dr.Blocks = int64(binary.BigEndian.Uint64(c16)) + 1
			bs = binary.BigEndian.Uint32(c16[8:])
		}
	} else {
		dr.Blocks = int64(last) + 1
	}
	if bs == 0 || bs > 4096 {
		bs = 512
	}
	dr.BlockSize = int(bs)
	return dr, nil
}

func trimASCII(b []byte) string {
	s := string(b)
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == 0) {
		s = s[:len(s)-1]
	}
	return s
}

// Size is the drive's size in bytes.
func (dr *Drive) Size() int64 { return dr.Blocks * int64(dr.BlockSize) }

// cmd runs one SCSI command: command block wrapper, data, status wrapper.
func (dr *Drive) cmd(cdb []byte, data []byte, in bool, timeoutMs int) (int, error) {
	dr.mu.Lock()
	defer dr.mu.Unlock()
	dr.tag++
	cbw := make([]byte, 31)
	binary.LittleEndian.PutUint32(cbw, 0x43425355) // "USBC"
	binary.LittleEndian.PutUint32(cbw[4:], dr.tag)
	binary.LittleEndian.PutUint32(cbw[8:], uint32(len(data)))
	if in {
		cbw[12] = 0x80
	}
	cbw[13] = dr.lun
	cbw[14] = byte(len(cdb))
	copy(cbw[15:], cdb)
	if _, err := dr.d.BulkWrite(dr.out, cbw, timeoutMs); err != nil {
		dr.recover()
		return 0, err
	}
	n := 0
	if len(data) > 0 {
		// in 16 KB pieces: older Android kernels refuse bigger usbfs transfers
		var err error
		for n < len(data) {
			piece := data[n:min(n+16384, len(data))]
			var k int
			if in {
				k, err = dr.d.BulkRead(dr.in, piece, timeoutMs)
			} else {
				k, err = dr.d.BulkWrite(dr.out, piece, timeoutMs)
			}
			n += k
			if err == nil && k == 0 {
				err = errors.New("the drive stopped answering")
			}
			if err != nil || (in && k < len(piece)) {
				break // a short read ends the data stage
			}
		}
		if err != nil {
			dr.d.ResetPipe(dr.in)
			dr.d.ResetPipe(dr.out)
		}
	}
	csw := make([]byte, 13)
	var err error
	for i := 0; i < 2; i++ {
		if _, err = dr.d.BulkRead(dr.in, csw, timeoutMs); err == nil {
			break
		}
		dr.d.ResetPipe(dr.in)
	}
	if err != nil {
		dr.recover()
		return n, err
	}
	if binary.LittleEndian.Uint32(csw) != 0x53425355 || binary.LittleEndian.Uint32(csw[4:]) != dr.tag {
		dr.recover()
		return n, errors.New("the drive answered out of step")
	}
	switch csw[12] {
	case 0:
		return n, nil
	case 1:
		return n, errors.New("the drive reported an error")
	default:
		dr.recover()
		return n, errors.New("the drive needs resetting")
	}
}

// recover is the bulk-only reset: class reset, then clear both endpoints.
func (dr *Drive) recover() {
	dr.d.Control(0x21, 0xFF, 0, uint16(dr.d.iface), nil, 2000)
	dr.d.ResetPipe(dr.in)
	dr.d.ResetPipe(dr.out)
}

const mscChunk = 64 << 10

// ReadAt reads whole blocks.
func (dr *Drive) ReadAt(p []byte, off int64) (int, error) {
	return dr.rw(p, off, false)
}

// WriteAt writes whole blocks.
func (dr *Drive) WriteAt(p []byte, off int64) (int, error) {
	return dr.rw(p, off, true)
}

func (dr *Drive) rw(p []byte, off int64, write bool) (int, error) {
	bs := int64(dr.BlockSize)
	if off%bs != 0 || int64(len(p))%bs != 0 {
		return 0, fmt.Errorf("drive access must be in %d-byte blocks", bs)
	}
	done := 0
	for done < len(p) {
		n := min(len(p)-done, mscChunk)
		lba := (off + int64(done)) / bs
		cnt := n / int(bs)
		var cdb []byte
		if lba > 0xFFFFFFFF {
			op := byte(0x88)
			if write {
				op = 0x8A
			}
			cdb = make([]byte, 16)
			cdb[0] = op
			binary.BigEndian.PutUint64(cdb[2:], uint64(lba))
			binary.BigEndian.PutUint32(cdb[10:], uint32(cnt))
		} else {
			op := byte(0x28)
			if write {
				op = 0x2A
			}
			cdb = make([]byte, 10)
			cdb[0] = op
			binary.BigEndian.PutUint32(cdb[2:], uint32(lba))
			binary.BigEndian.PutUint16(cdb[7:], uint16(cnt))
		}
		var err error
		for try := 0; try < 3; try++ {
			if _, err = dr.cmd(cdb, p[done:done+n], !write, 20000); err == nil {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		if err != nil {
			return done, err
		}
		done += n
	}
	return done, nil
}

// Sync asks the drive to write out its cache.
func (dr *Drive) Sync() error {
	_, err := dr.cmd([]byte{0x35, 0, 0, 0, 0, 0, 0, 0, 0, 0}, nil, false, 60000)
	return err
}

// Close lets the drive go.
func (dr *Drive) Close() error {
	dr.d.Close()
	return nil
}

// Eject asks the drive to stop and eject the medium where it can.
func (dr *Drive) Eject() {
	dr.cmd([]byte{0x1B, 0, 0, 0, 0x02, 0}, nil, false, 5000)
}
