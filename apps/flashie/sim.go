// Flashie — a simulated ESP32 ROM bootloader.
// Used by the unit tests and by demo mode (set FLASHIE_DEMO=1) so the whole
// flow can be tried without a board plugged in. Made by nrmluvdev.

package flashie

import (
	"bytes"
	"compress/zlib"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"io"
	"sync"
	"time"
)

const esp8266SimID = 0xFFFF

type simChip struct {
	mu        sync.Mutex
	out       bytes.Buffer // bytes waiting for the host to read
	in        []byte       // partial inbound SLIP data
	dtr, rts  bool
	download  bool
	chipID    uint32 // 0 = classic ESP32 (no GET_SECURITY_INFO)
	flash     []byte
	deflBuf   []byte
	deflOff   uint32
	deflN     uint32
	deflSeen  uint32
	writeOff  uint32
	blockWait time.Duration
	baud      int
}

func newSimChip(chipID uint32, blockWait time.Duration) *simChip {
	f := bytes.Repeat([]byte{0xFF}, 16<<20)
	return &simChip{chipID: chipID, flash: f, blockWait: blockWait, baud: 115200}
}

func (s *simChip) SetDTR(on bool) error { s.mu.Lock(); s.dtr = on; s.mu.Unlock(); return nil }
func (s *simChip) SetRTS(on bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.rts && !on { // EN released: chip boots, GPIO0 = !DTR
		s.download = s.dtr
		s.out.WriteString("ets Jun  8 2016 00:22:57\r\nrst:0x1 (POWERON_RESET),boot:0x3 (DOWNLOAD_BOOT(UART0/UART1/SDIO_REI_REO_V2))\r\nwaiting for download\r\n")
	}
	s.rts = on
	return nil
}
func (s *simChip) SetBaud(b int) error { s.baud = b; return nil }
func (s *simChip) Purge() error        { s.mu.Lock(); s.out.Reset(); s.mu.Unlock(); return nil }
func (s *simChip) Close() error        { return nil }

func (s *simChip) ReadTimeout(p []byte, d time.Duration) (int, error) {
	deadline := time.Now().Add(d)
	for {
		s.mu.Lock()
		if s.out.Len() > 0 {
			n, _ := s.out.Read(p)
			s.mu.Unlock()
			return n, nil
		}
		s.mu.Unlock()
		if time.Now().After(deadline) {
			return 0, nil
		}
		time.Sleep(2 * time.Millisecond)
	}
}

func (s *simChip) Write(p []byte) (int, error) {
	s.mu.Lock()
	s.in = append(s.in, p...)
	var frames [][]byte
	for {
		start := bytes.IndexByte(s.in, 0xC0)
		if start < 0 {
			s.in = nil
			break
		}
		end := bytes.IndexByte(s.in[start+1:], 0xC0)
		if end < 0 {
			s.in = s.in[start:]
			break
		}
		raw := s.in[start+1 : start+1+end]
		s.in = s.in[start+2+end:]
		if len(raw) == 0 {
			continue
		}
		raw = bytes.ReplaceAll(raw, []byte{0xDB, 0xDC}, []byte{0xC0})
		raw = bytes.ReplaceAll(raw, []byte{0xDB, 0xDD}, []byte{0xDB})
		frames = append(frames, raw)
	}
	s.mu.Unlock()
	for _, f := range frames {
		s.handle(f)
	}
	return len(p), nil
}

func (s *simChip) reply(op byte, val uint32, data []byte, ok bool, errCode byte) {
	status := []byte{0, 0, 0, 0}
	if !ok {
		status = []byte{1, errCode, 0, 0}
	}
	if s.chipID == esp8266SimID {
		status = status[:2]
	}
	body := append(append([]byte{}, data...), status...)
	h := make([]byte, 8)
	h[0], h[1] = 1, op
	binary.LittleEndian.PutUint16(h[2:], uint16(len(body)))
	binary.LittleEndian.PutUint32(h[4:], val)
	s.mu.Lock()
	s.out.Write(slipEncode(append(h, body...)))
	s.mu.Unlock()
}

func (s *simChip) handle(f []byte) {
	if len(f) < 8 || f[0] != 0 {
		return
	}
	s.mu.Lock()
	dl := s.download
	s.mu.Unlock()
	if !dl {
		return // running user firmware: ignores the protocol
	}
	op := f[1]
	data := f[8:]
	if s.chipID == esp8266SimID {
		switch op {
		case cmdGetSecurityInfo, cmdSpiAttach, cmdFlashDeflBegin, cmdFlashDeflData, cmdSpiFlashMD5, cmdChangeBaud:
			s.reply(op, 0, nil, false, 0x05)
			return
		}
	}
	u32 := func(i int) uint32 { return binary.LittleEndian.Uint32(data[i*4:]) }
	switch op {
	case cmdSync:
		for i := 0; i < 4; i++ {
			s.reply(op, 0x20120707, nil, true, 0)
		}
	case cmdGetSecurityInfo:
		if s.chipID == 0 {
			s.reply(op, 0, nil, false, 0x05)
			return
		}
		b := make([]byte, 20)
		binary.LittleEndian.PutUint32(b[12:], s.chipID)
		s.reply(op, 0, b, true, 0)
	case cmdReadReg:
		var v uint32
		switch u32(0) {
		case chipMagicReg:
			v = 0x00f01d83
			if s.chipID == esp8266SimID {
				v = 0xfff0c101
			} else if c := chipByID(s.chipID); c != nil && len(c.Magic) > 0 && s.chipID != 0 {
				v = c.Magic[0]
			}
		case 0x3FF5A008:
			v = 0x00002462
		case 0x3FF5A004:
			v = 0xAB12CD34
		}
		s.reply(op, v, nil, true, 0)
	case cmdSpiAttach, cmdSpiSetParams, cmdChangeBaud:
		s.reply(op, 0, nil, true, 0)
	case cmdFlashBegin:
		s.writeOff = u32(3)
		s.reply(op, 0, nil, true, 0)
	case cmdFlashData:
		blk := data[16:]
		if checksum(blk) != binary.LittleEndian.Uint32(f[4:]) {
			s.reply(op, 0, nil, false, 0x07)
			return
		}
		seq := u32(1)
		copy(s.flash[s.writeOff+seq*romWriteBlock:], blk)
		time.Sleep(s.blockWait)
		s.reply(op, 0, nil, true, 0)
	case cmdFlashDeflBegin:
		s.deflOff, s.deflN, s.deflSeen, s.deflBuf = u32(3), u32(1), 0, nil
		s.reply(op, 0, nil, true, 0)
	case cmdFlashDeflData:
		blk := data[16:]
		if checksum(blk) != binary.LittleEndian.Uint32(f[4:]) {
			s.reply(op, 0, nil, false, 0x07)
			return
		}
		s.deflBuf = append(s.deflBuf, blk...)
		s.deflSeen++
		if s.deflSeen == s.deflN {
			zr, err := zlib.NewReader(bytes.NewReader(s.deflBuf))
			if err != nil {
				s.reply(op, 0, nil, false, 0x0B)
				return
			}
			raw, err := io.ReadAll(zr)
			if err != nil {
				s.reply(op, 0, nil, false, 0x0B)
				return
			}
			copy(s.flash[s.deflOff:], raw)
		}
		time.Sleep(s.blockWait)
		s.reply(op, 0, nil, true, 0)
	case cmdSpiFlashMD5:
		addr, size := u32(0), u32(1)
		sum := md5.Sum(s.flash[addr : addr+size])
		s.reply(op, 0, []byte(hex.EncodeToString(sum[:])), true, 0)
	default:
		s.reply(op, 0, nil, false, 0x05)
	}
}
