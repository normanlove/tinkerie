// Flashie — ESP32 firmware flasher
// Part of nrmluvdev's free maker tool suite. Made by nrmluvdev.
//
// esp.go — a native Go implementation of the Espressif ROM bootloader
// serial protocol (the same protocol esptool.py speaks), talking straight
// to the chip's mask-ROM download mode. No stub loader, no external tools.

package flashie

import (
	"bytes"
	"compress/zlib"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Port is the minimal serial interface the loader needs. The Windows
// implementation lives in serial_windows.go; tests use a simulated chip.
type Port interface {
	Write(p []byte) (int, error)
	// ReadTimeout reads whatever is available, waiting at most d for the
	// first byte. Returns (0, nil) on timeout.
	ReadTimeout(p []byte, d time.Duration) (int, error)
	SetBaud(baud int) error
	SetDTR(on bool) error
	SetRTS(on bool) error
	Purge() error
	Close() error
}

// ROM loader command opcodes.
const (
	cmdFlashBegin      = 0x02
	cmdFlashData       = 0x03
	cmdFlashEnd        = 0x04
	cmdMemBegin        = 0x05
	cmdMemEnd          = 0x06
	cmdMemData         = 0x07
	cmdSync            = 0x08
	cmdWriteReg        = 0x09
	cmdReadReg         = 0x0A
	cmdSpiSetParams    = 0x0B
	cmdSpiAttach       = 0x0D
	cmdChangeBaud      = 0x0F
	cmdFlashDeflBegin  = 0x10
	cmdFlashDeflData   = 0x11
	cmdFlashDeflEnd    = 0x12
	cmdSpiFlashMD5     = 0x13
	cmdGetSecurityInfo = 0x14
)

const (
	romWriteBlock   = 0x400 // ROM loader flash block size (1 KB)
	checksumSeed    = 0xEF
	defaultTimeout  = 3 * time.Second
	chipMagicReg    = 0x40001000
	eraseTimeoutMB  = 30 * time.Second
	writeTimeoutMB  = 40 * time.Second
	md5TimeoutMB    = 8 * time.Second
	syncAttempts    = 6
	connectAttempts = 6
)

var errTimeout = errors.New("timed out waiting for the board")

// Chip describes a member of the ESP32 family.
type Chip struct {
	Name       string
	ID         uint32   // chip_id reported by GET_SECURITY_INFO / image header
	Magic      []uint32 // value of the chip-detect magic register
	BootOffset uint32   // where the 2nd-stage bootloader lives in flash
	EncParam   bool     // ROM's FLASH_BEGIN takes an extra "encrypted" word
	MacReg     uint32   // eFuse MAC base (0 = unknown)
	Supported  bool
}

var chips = []Chip{
	{Name: "ESP32", ID: 0, Magic: []uint32{0x00f01d83}, BootOffset: 0x1000, EncParam: false, MacReg: 0, Supported: true},
	{Name: "ESP32-S2", ID: 2, Magic: []uint32{0x000007c6}, BootOffset: 0x1000, EncParam: true, MacReg: 0x3F41A044, Supported: true},
	{Name: "ESP32-C3", ID: 5, Magic: []uint32{0x6921506f, 0x1b31506f, 0x4881606f, 0x4361606f}, BootOffset: 0x0, EncParam: true, MacReg: 0x60008844, Supported: true},
	{Name: "ESP32-S3", ID: 9, Magic: []uint32{0x00000009}, BootOffset: 0x0, EncParam: true, MacReg: 0x60007044, Supported: true},
	{Name: "ESP32-C2", ID: 12, Magic: []uint32{0x6f51306f, 0x7c41a06f}, BootOffset: 0x0, EncParam: true, MacReg: 0x60008840, Supported: true},
	{Name: "ESP32-C6", ID: 13, Magic: []uint32{0x2ce0806f}, BootOffset: 0x0, EncParam: true, MacReg: 0x600B0844, Supported: true},
	{Name: "ESP32-H2", ID: 16, Magic: []uint32{0xd7b73e80}, BootOffset: 0x0, EncParam: true, MacReg: 0x600B0844, Supported: true},
	{Name: "ESP32-P4", ID: 18, BootOffset: 0x2000, EncParam: true, Supported: true},
	{Name: "ESP32-C5", ID: 23, BootOffset: 0x2000, EncParam: true, Supported: true},
	{Name: "ESP8266", ID: 0xFFFF, Magic: []uint32{0xfff0c101}, BootOffset: 0x0, EncParam: false, Supported: true},
}

func chipByID(id uint32) *Chip {
	for i := range chips {
		if chips[i].ID == id && chips[i].Name != "ESP8266" {
			return &chips[i]
		}
	}
	return nil
}

func chipByMagic(m uint32) *Chip {
	for i := range chips {
		for _, v := range chips[i].Magic {
			if v == m {
				return &chips[i]
			}
		}
	}
	return nil
}

// ChipInfo is what Identify reports back to the UI.
type ChipInfo struct {
	Family     string      `json:"family"`
	Chip       string      `json:"chip"`
	ChipID     uint32      `json:"chipId"`
	MAC        string      `json:"mac,omitempty"`
	BootOffset uint32      `json:"bootOffset"`
	Supported  bool        `json:"supported"`
	Proto      string      `json:"proto"`
	Baud       int         `json:"baud,omitempty"`
	Port       string      `json:"port,omitempty"`
	BoardID    string      `json:"boardId,omitempty"`
	Details    [][2]string `json:"details,omitempty"`
	Note       string      `json:"note,omitempty"`
}

// Loader drives one ROM bootloader session over a Port.
type Loader struct {
	p       Port
	rx      []byte
	chip    *Chip
	usbJTAG bool // native USB-Serial/JTAG peripheral (S3/C3/C6/H2)
	usbOTG  bool // native USB OTG CDC (S2) — baud changes are meaningless
	// Attempts is how many reset/sync rounds Connect tries (0 = default).
	Attempts int
	Log      func(string)
	Status   func(string)
}

func NewLoader(p Port, usbJTAG, usbOTG bool) *Loader {
	return &Loader{p: p, usbJTAG: usbJTAG, usbOTG: usbOTG, Log: func(string) {}, Status: func(string) {}}
}

func (l *Loader) logf(f string, a ...any) { l.Log(fmt.Sprintf(f, a...)) }

// ---------------------------------------------------------------- SLIP ---

func slipEncode(pkt []byte) []byte {
	out := make([]byte, 0, len(pkt)+8)
	out = append(out, 0xC0)
	for _, b := range pkt {
		switch b {
		case 0xC0:
			out = append(out, 0xDB, 0xDC)
		case 0xDB:
			out = append(out, 0xDB, 0xDD)
		default:
			out = append(out, b)
		}
	}
	return append(out, 0xC0)
}

func (l *Loader) writeAll(b []byte) error {
	for len(b) > 0 {
		n, err := l.p.Write(b)
		if err != nil {
			return err
		}
		b = b[n:]
	}
	return nil
}

// readPacket returns the next complete SLIP frame.
func (l *Loader) readPacket(timeout time.Duration) ([]byte, error) {
	deadline := time.Now().Add(timeout)
	var pkt []byte
	in, esc := false, false
	buf := make([]byte, 4096)
	for {
		for len(l.rx) > 0 {
			b := l.rx[0]
			l.rx = l.rx[1:]
			if !in {
				if b == 0xC0 {
					in = true
					pkt = pkt[:0]
				}
				continue
			}
			if esc {
				esc = false
				switch b {
				case 0xDC:
					pkt = append(pkt, 0xC0)
				case 0xDD:
					pkt = append(pkt, 0xDB)
				default: // corrupt escape — drop frame, resync
					in = false
				}
				continue
			}
			switch b {
			case 0xDB:
				esc = true
			case 0xC0:
				if len(pkt) == 0 {
					continue // back-to-back delimiters
				}
				return pkt, nil
			default:
				pkt = append(pkt, b)
			}
		}
		rem := time.Until(deadline)
		if rem <= 0 {
			return nil, errTimeout
		}
		if rem > 100*time.Millisecond {
			rem = 100 * time.Millisecond
		}
		n, err := l.p.ReadTimeout(buf, rem)
		if err != nil {
			return nil, err
		}
		l.rx = append(l.rx, buf[:n]...)
	}
}

func checksum(data []byte) uint32 {
	c := uint32(checksumSeed)
	for _, b := range data {
		c ^= uint32(b)
	}
	return c
}

// command sends one request and waits for the matching response.
func (l *Loader) command(op byte, data []byte, chk uint32, timeout time.Duration) (uint32, []byte, error) {
	pkt := make([]byte, 8+len(data))
	pkt[0] = 0x00
	pkt[1] = op
	binary.LittleEndian.PutUint16(pkt[2:], uint16(len(data)))
	binary.LittleEndian.PutUint32(pkt[4:], chk)
	copy(pkt[8:], data)
	if err := l.writeAll(slipEncode(pkt)); err != nil {
		return 0, nil, err
	}
	deadline := time.Now().Add(timeout)
	for {
		r, err := l.readPacket(time.Until(deadline))
		if err != nil {
			return 0, nil, err
		}
		if len(r) < 8 || r[0] != 0x01 || r[1] != op {
			continue // stale or unrelated frame
		}
		return binary.LittleEndian.Uint32(r[4:]), r[8:], nil
	}
}

var romErrors = map[byte]string{
	0x05: "the board didn't understand the request",
	0x06: "the board couldn't carry out the request",
	0x07: "data got corrupted on the way (bad checksum) — try a slower speed or a better USB cable",
	0x08: "writing to flash failed",
	0x09: "reading from flash failed",
	0x0A: "flash read length error",
	0x0B: "the compressed data couldn't be unpacked — try turning compression off",
}

// check sends a command and validates the status bytes. payload is the
// number of data bytes that precede the status bytes in a successful reply
// (this lets us cope with both 2- and 4-byte status trailers).
func (l *Loader) check(what string, op byte, data []byte, chk uint32, timeout time.Duration, payload int) (uint32, []byte, error) {
	val, body, err := l.command(op, data, chk, timeout)
	if err != nil {
		return 0, nil, fmt.Errorf("%s: %w", what, err)
	}
	idx := payload
	if len(body) < idx+2 {
		idx = 0
	}
	if len(body) < 2 {
		return 0, nil, fmt.Errorf("%s: the board sent a reply that was too short", what)
	}
	if body[idx] != 0 {
		msg, ok := romErrors[body[idx+1]]
		if !ok {
			msg = fmt.Sprintf("error code 0x%02X", body[idx+1])
		}
		return 0, nil, fmt.Errorf("%s: %s", what, msg)
	}
	if payload > 0 && len(body) >= payload {
		return val, body[:payload], nil
	}
	return val, nil, nil
}

func le32s(vals ...uint32) []byte {
	b := make([]byte, 4*len(vals))
	for i, v := range vals {
		binary.LittleEndian.PutUint32(b[i*4:], v)
	}
	return b
}

func timeoutPerMB(per time.Duration, size int) time.Duration {
	t := time.Duration(float64(per) * float64(size) / 1e6)
	if t < defaultTimeout {
		return defaultTimeout
	}
	return t
}

// ------------------------------------------------------ reset + connect ---

func (l *Loader) sleep(d time.Duration) { time.Sleep(d) }

// classicReset drives the standard two-transistor auto-reset circuit
// found on almost every dev board (DTR → GPIO0, RTS → EN).
func (l *Loader) classicReset(delay time.Duration) {
	l.p.SetDTR(false)
	l.p.SetRTS(true) // EN low: hold in reset
	l.sleep(100 * time.Millisecond)
	l.p.SetDTR(true) // GPIO0 low
	l.p.SetRTS(false)
	l.sleep(delay)
	l.p.SetDTR(false) // release GPIO0
}

// usbJTAGReset is the sequence for the built-in USB-Serial/JTAG peripheral.
func (l *Loader) usbJTAGReset() {
	l.p.SetRTS(false)
	l.p.SetDTR(false)
	l.sleep(100 * time.Millisecond)
	l.p.SetDTR(true)
	l.p.SetRTS(false)
	l.sleep(100 * time.Millisecond)
	l.p.SetRTS(true)
	l.p.SetDTR(false)
	l.p.SetRTS(true)
	l.sleep(100 * time.Millisecond)
	l.p.SetDTR(false)
	l.p.SetRTS(false)
}

// HardReset restarts the chip into the freshly flashed firmware.
func (l *Loader) HardReset() {
	l.p.SetDTR(false)
	l.p.SetRTS(true)
	if l.usbJTAG {
		l.sleep(200 * time.Millisecond)
	} else {
		l.sleep(100 * time.Millisecond)
	}
	l.p.SetRTS(false)
}

func (l *Loader) sync() error {
	data := append([]byte{0x07, 0x07, 0x12, 0x20}, bytes.Repeat([]byte{0x55}, 32)...)
	for i := 0; i < syncAttempts; i++ {
		_, _, err := l.command(cmdSync, data, 0, 120*time.Millisecond)
		if err == nil {
			// ROM answers SYNC several times; swallow the extras.
			for k := 0; k < 8; k++ {
				if _, err := l.readPacket(40 * time.Millisecond); err != nil {
					break
				}
			}
			return nil
		}
	}
	return errTimeout
}

// Connect resets the board into download mode and syncs with the ROM.
func (l *Loader) Connect() error {
	l.Status("Putting the board into download mode…")
	tries := connectAttempts
	if l.Attempts > 0 {
		tries = l.Attempts
	}
	for attempt := 0; attempt < tries; attempt++ {
		switch {
		case l.usbJTAG && attempt%2 == 0:
			l.logf("Reset attempt %d: USB-Serial/JTAG sequence", attempt+1)
			l.usbJTAGReset()
		case attempt%2 == 0:
			l.logf("Reset attempt %d: classic DTR/RTS sequence (50 ms)", attempt+1)
			l.classicReset(50 * time.Millisecond)
		default:
			l.logf("Reset attempt %d: classic DTR/RTS sequence (550 ms, for slow boards)", attempt+1)
			l.classicReset(550 * time.Millisecond)
		}
		l.sleep(50 * time.Millisecond)
		l.p.Purge()
		l.rx = nil
		if err := l.sync(); err == nil {
			l.logf("Synced with ROM bootloader")
			return nil
		}
		if attempt == 2 {
			l.Status("Still trying… if this keeps failing, hold BOOT, tap RESET/EN, then let go of BOOT.")
		}
	}
	return errors.New("couldn't get the board into download mode. Hold the BOOT button, tap RESET (EN), release BOOT, then press Flash again. Also check it's a data USB cable, not charge-only")
}

func (l *Loader) ReadReg(addr uint32) (uint32, error) {
	v, _, err := l.check("read register", cmdReadReg, le32s(addr), 0, defaultTimeout, 0)
	return v, err
}

// DetectChip identifies which ESP32 variant we're talking to.
func (l *Loader) DetectChip() (*ChipInfo, error) {
	var c *Chip
	// Newer ROMs report a chip ID directly.
	if _, body, err := l.check("security info", cmdGetSecurityInfo, nil, 0, 600*time.Millisecond, 20); err == nil && len(body) >= 20 {
		id := binary.LittleEndian.Uint32(body[12:16])
		c = chipByID(id)
		l.logf("GET_SECURITY_INFO chip_id=%d", id)
	} else {
		l.logf("GET_SECURITY_INFO not available (%v) — using magic register", err)
		l.rx = nil
		l.p.Purge()
	}
	if c == nil {
		m, err := l.ReadReg(chipMagicReg)
		if err != nil {
			return nil, err
		}
		l.logf("Chip magic register = 0x%08X", m)
		c = chipByMagic(m)
		if c == nil {
			return nil, fmt.Errorf("found an Espressif bootloader but didn't recognise the chip (magic 0x%08X)", m)
		}
	}
	l.chip = c
	info := &ChipInfo{Family: famESP, Proto: "esp", Chip: c.Name, ChipID: c.ID, BootOffset: c.BootOffset, Supported: c.Supported}
	if mac, err := l.readMAC(); err == nil {
		info.MAC = mac
	}
	return info, nil
}

func (l *Loader) readMAC() (string, error) {
	if l.chip == nil {
		return "", errors.New("no chip")
	}
	var hi, lo uint32
	var err error
	switch {
	case l.chip.Name == "ESP32":
		if hi, err = l.ReadReg(0x3FF5A008); err != nil {
			return "", err
		}
		if lo, err = l.ReadReg(0x3FF5A004); err != nil {
			return "", err
		}
	case l.chip.MacReg != 0:
		if lo, err = l.ReadReg(l.chip.MacReg); err != nil {
			return "", err
		}
		if hi, err = l.ReadReg(l.chip.MacReg + 4); err != nil {
			return "", err
		}
	default:
		return "", errors.New("MAC location unknown")
	}
	b := make([]byte, 8)
	binary.BigEndian.PutUint32(b[0:], hi)
	binary.BigEndian.PutUint32(b[4:], lo)
	parts := make([]string, 6)
	for i, v := range b[2:8] {
		parts[i] = fmt.Sprintf("%02X", v)
	}
	return strings.Join(parts, ":"), nil
}

// ChangeBaud switches both ends to a faster link.
func (l *Loader) is8266() bool { return l.chip != nil && l.chip.Name == "ESP8266" }

func (l *Loader) ChangeBaud(baud int) error {
	if l.is8266() {
		l.logf("ESP8266 ROM can't change speed — staying at 115200 baud")
		return nil
	}
	if l.usbJTAG || l.usbOTG {
		l.logf("Native USB link — speed is fixed by USB, skipping baud change")
		return nil
	}
	if _, _, err := l.check("change speed", cmdChangeBaud, le32s(uint32(baud), 0), 0, defaultTimeout, 0); err != nil {
		return err
	}
	if err := l.p.SetBaud(baud); err != nil {
		return err
	}
	l.sleep(60 * time.Millisecond)
	l.p.Purge()
	l.rx = nil
	return nil
}

// FlashOptions controls a write.
type FlashOptions struct {
	Offset    uint32
	FlashSize uint32
	Compress  bool
	Progress  func(done, total int)
	Cancel    func() bool
}

var errCancelled = errors.New("cancelled")

// Flash writes image to flash at opts.Offset and verifies it via MD5.
func (l *Loader) Flash(image []byte, opts FlashOptions) error {
	if l.chip == nil {
		return errors.New("chip not detected")
	}
	if !l.chip.Supported {
		return fmt.Errorf("%s isn't supported by Flashie yet", l.chip.Name)
	}
	if opts.Offset%0x1000 != 0 {
		return fmt.Errorf("offset 0x%X isn't on a 4 KB boundary (it should end in 000)", opts.Offset)
	}
	if opts.FlashSize == 0 {
		opts.FlashSize = 4 << 20
	}
	if uint64(opts.Offset)+uint64(len(image)) > uint64(opts.FlashSize) {
		return fmt.Errorf("the firmware (%s) doesn't fit at 0x%X in a %d MB flash chip", humanBytes(len(image)), opts.Offset, opts.FlashSize>>20)
	}
	// Pad to a 4-byte boundary with erased-flash 0xFF bytes.
	if r := len(image) % 4; r != 0 {
		image = append(append([]byte{}, image...), bytes.Repeat([]byte{0xFF}, 4-r)...)
	}

	l.Status("Getting the flash chip ready…")
	if l.is8266() {
		// ESP8266 ROM has no SPI_ATTACH, no compression and no MD5.
		if _, _, err := l.check("set flash parameters", cmdSpiSetParams,
			le32s(0, opts.FlashSize, 64*1024, 4*1024, 256, 0xFFFF), 0, defaultTimeout, 0); err != nil {
			l.logf("SPI_SET_PARAMS: %v (ignored on ESP8266)", err)
		}
		if err := l.flashPlain(image, opts); err != nil {
			return err
		}
		l.logf("ESP8266 ROM can't checksum flash, so the write isn't read back (each block is still checksummed in transit)")
		return nil
	}
	if _, _, err := l.check("attach flash", cmdSpiAttach, make([]byte, 8), 0, defaultTimeout, 0); err != nil {
		return err
	}
	if _, _, err := l.check("set flash parameters", cmdSpiSetParams,
		le32s(0, opts.FlashSize, 64*1024, 4*1024, 256, 0xFFFF), 0, defaultTimeout, 0); err != nil {
		return err
	}

	var err error
	if opts.Compress {
		err = l.flashDeflate(image, opts)
	} else {
		err = l.flashPlain(image, opts)
	}
	if err != nil {
		return err
	}

	l.Status("Double-checking every byte was written correctly…")
	want := md5.Sum(image)
	_, body, err := l.check("verify", cmdSpiFlashMD5, le32s(opts.Offset, uint32(len(image)), 0, 0), 0,
		timeoutPerMB(md5TimeoutMB, len(image)), 32)
	if err != nil {
		return err
	}
	got := strings.ToLower(string(body))
	if len(body) == 16 { // stub-style raw digest, just in case
		got = hex.EncodeToString(body)
	}
	if got != hex.EncodeToString(want[:]) {
		return fmt.Errorf("verification failed — what's on the chip doesn't match the file (got %s, expected %x). Try again at a slower speed", got, want)
	}
	l.logf("MD5 verified: %s", got)
	return nil
}

func (l *Loader) beginParams(size, blocks, offset uint32) []byte {
	p := le32s(size, blocks, romWriteBlock, offset)
	if l.chip.EncParam {
		p = append(p, le32s(0)...)
	}
	return p
}

func (l *Loader) flashDeflate(image []byte, opts FlashOptions) error {
	var zb bytes.Buffer
	zw, _ := zlib.NewWriterLevel(&zb, zlib.BestCompression)
	zw.Write(image)
	zw.Close()
	comp := zb.Bytes()
	blocks := (len(comp) + romWriteBlock - 1) / romWriteBlock
	eraseBlocks := (len(image) + romWriteBlock - 1) / romWriteBlock
	writeSize := eraseBlocks * romWriteBlock
	l.logf("Compressed %s → %s (%.0f%%)", humanBytes(len(image)), humanBytes(len(comp)), 100*float64(len(comp))/float64(len(image)))

	l.Status("Erasing the area we're about to write (big files take a few seconds)…")
	if _, _, err := l.check("erase", cmdFlashDeflBegin, l.beginParams(uint32(writeSize), uint32(blocks), opts.Offset), 0,
		timeoutPerMB(eraseTimeoutMB, writeSize), 0); err != nil {
		return err
	}
	l.Status("Writing firmware…")
	ratio := float64(len(image)) / float64(len(comp))
	blockTimeout := timeoutPerMB(writeTimeoutMB, int(ratio*romWriteBlock))
	if blockTimeout < 5*time.Second {
		blockTimeout = 5 * time.Second
	}
	for seq := 0; seq < blocks; seq++ {
		if opts.Cancel != nil && opts.Cancel() {
			return errCancelled
		}
		start := seq * romWriteBlock
		end := start + romWriteBlock
		if end > len(comp) {
			end = len(comp)
		}
		blk := comp[start:end]
		payload := append(le32s(uint32(len(blk)), uint32(seq), 0, 0), blk...)
		if err := l.dataBlock(cmdFlashDeflData, payload, blk, blockTimeout, seq); err != nil {
			return err
		}
		if opts.Progress != nil {
			done := int(float64(end) * ratio)
			if done > len(image) || seq == blocks-1 {
				done = len(image)
			}
			opts.Progress(done, len(image))
		}
	}
	return nil
}

func (l *Loader) flashPlain(image []byte, opts FlashOptions) error {
	blocks := (len(image) + romWriteBlock - 1) / romWriteBlock
	eraseSize := uint32(len(image))
	if l.is8266() {
		eraseSize = esp8266EraseSize(opts.Offset, uint32(len(image)))
	}
	l.Status("Erasing the area we're about to write (big files take a few seconds)…")
	if _, _, err := l.check("erase", cmdFlashBegin, l.beginParams(eraseSize, uint32(blocks), opts.Offset), 0,
		timeoutPerMB(eraseTimeoutMB, len(image)), 0); err != nil {
		return err
	}
	l.Status("Writing firmware…")
	for seq := 0; seq < blocks; seq++ {
		if opts.Cancel != nil && opts.Cancel() {
			return errCancelled
		}
		start := seq * romWriteBlock
		end := start + romWriteBlock
		var blk []byte
		if end <= len(image) {
			blk = image[start:end]
		} else {
			blk = append(append([]byte{}, image[start:]...), bytes.Repeat([]byte{0xFF}, end-len(image))...)
			end = len(image)
		}
		payload := append(le32s(uint32(len(blk)), uint32(seq), 0, 0), blk...)
		if err := l.dataBlock(cmdFlashData, payload, blk, timeoutPerMB(writeTimeoutMB, romWriteBlock), seq); err != nil {
			return err
		}
		if opts.Progress != nil {
			opts.Progress(end, len(image))
		}
	}
	return nil
}

// dataBlock sends one block, retrying a couple of times on timeouts.
func (l *Loader) dataBlock(op byte, payload, blk []byte, timeout time.Duration, seq int) error {
	var err error
	for try := 0; try < 3; try++ {
		_, _, err = l.check(fmt.Sprintf("write block %d", seq), op, payload, checksum(blk), timeout, 0)
		if err == nil {
			return nil
		}
		if !errors.Is(err, errTimeout) {
			return err
		}
		l.logf("Block %d timed out, retrying (%d/3)", seq, try+1)
	}
	return err
}

func humanBytes(n int) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.2f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d bytes", n)
}

// esp8266EraseSize works around the ESP8266 ROM's erase-size bug
// (it erases roughly double what it's asked to), as esptool does.
func esp8266EraseSize(offset, size uint32) uint32 {
	const sectorsPerBlock, sector = 16, 4096
	num := (size + sector - 1) / sector
	start := offset / sector
	head := sectorsPerBlock - start%sectorsPerBlock
	if num < head {
		head = num
	}
	if num < 2*head {
		return (num + 1) / 2 * sector
	}
	return (num - head) * sector
}
