// Flashie — STM32 system-memory UART bootloader (ST AN3155), native Go.
// Works with Blue Pill / Black Pill and most STM32s via a USB-serial
// adapter on USART1 (PA9 = TX, PA10 = RX), 8E1, BOOT0 = 1.
// Made by nrmluvdev.

package flashie

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"time"
)

const (
	stmACK  = 0x79
	stmNACK = 0x1F

	stmFlashBase = 0x08000000
)

var stmChips = map[uint16]string{
	0x410: "STM32F103 (medium density — Blue Pill)", 0x412: "STM32F10x (low density)", 0x414: "STM32F10x (high density)",
	0x430: "STM32F10x (XL density)", 0x418: "STM32F105/107", 0x420: "STM32F100 (value line)", 0x428: "STM32F100 (high density)",
	0x413: "STM32F405/407/415/417", 0x419: "STM32F42x/43x", 0x423: "STM32F401xB/C (Black Pill)", 0x433: "STM32F401xD/E (Black Pill)",
	0x431: "STM32F411 (Black Pill)", 0x458: "STM32F410", 0x441: "STM32F412", 0x463: "STM32F413/423", 0x421: "STM32F446",
	0x440: "STM32F030x8/F05x", 0x444: "STM32F03x", 0x445: "STM32F04x", 0x448: "STM32F07x", 0x442: "STM32F09x",
	0x422: "STM32F30x", 0x432: "STM32F37x", 0x438: "STM32F334", 0x439: "STM32F301/302",
	0x460: "STM32G07x/08x", 0x466: "STM32G03x/04x", 0x468: "STM32G431/441", 0x469: "STM32G47x/48x",
	0x415: "STM32L47x/48x", 0x435: "STM32L43x/44x", 0x464: "STM32L41x/42x", 0x417: "STM32L0x3", 0x447: "STM32L07x",
	0x449: "STM32F74x/75x", 0x452: "STM32F72x/73x", 0x451: "STM32F76x/77x", 0x450: "STM32H74x/75x", 0x483: "STM32H72x/73x",
	0x497: "STM32WLE5", 0x495: "STM32WB55",
}

// Where each family keeps its "flash size in KB" register.
func stmFlashSizeReg(pid uint16) uint32 {
	switch pid {
	case 0x410, 0x412, 0x414, 0x430, 0x418, 0x420, 0x428:
		return 0x1FFFF7E0
	case 0x413, 0x419, 0x423, 0x433, 0x431, 0x458, 0x441, 0x463, 0x421:
		return 0x1FFF7A22
	case 0x440, 0x444, 0x445, 0x448, 0x442, 0x422, 0x432, 0x438, 0x439:
		return 0x1FFFF7CC
	}
	return 0
}

type stm32 struct {
	k        *link
	cmds     []byte
	extErase bool
	pid      uint16
	version  byte
}

// connectSTM32 wakes the bootloader (0x7F autobaud) and reads its info.
func connectSTM32(p Port, r *reporter) (*stm32, error) {
	s := &stm32{k: newLink(p)}
	s.k.drain(30 * time.Millisecond)
	ok := false
	for i := 0; i < 6 && !ok; i++ {
		s.k.write(0x7F)
		b, err := s.k.readByte(300 * time.Millisecond)
		// NACK means "already autobauded" — that's fine too.
		if err == nil && (b == stmACK || b == stmNACK) {
			ok = true
		}
	}
	if !ok {
		return nil, errors.New("the STM32 bootloader didn't answer. Set BOOT0 to 1 (jumper or button), press RESET, and check the wiring: adapter TX → PA10, RX → PA9, GND → GND")
	}
	s.k.drain(20 * time.Millisecond)
	// GET: version and supported commands.
	if err := s.send(0x00); err != nil {
		return nil, fmt.Errorf("get commands: %w", err)
	}
	n, err := s.k.readByte(time.Second)
	if err != nil {
		return nil, err
	}
	info, err := s.k.readN(int(n)+1, time.Second)
	if err != nil {
		return nil, err
	}
	if err := s.ack("get commands", time.Second); err != nil {
		return nil, err
	}
	s.version, s.cmds = info[0], info[1:]
	s.extErase = bytes.IndexByte(s.cmds, 0x44) >= 0
	// GET ID.
	if err := s.send(0x02); err != nil {
		return nil, fmt.Errorf("get chip ID: %w", err)
	}
	n, err = s.k.readByte(time.Second)
	if err != nil {
		return nil, err
	}
	id, err := s.k.readN(int(n)+1, time.Second)
	if err != nil {
		return nil, err
	}
	if err := s.ack("get chip ID", time.Second); err != nil {
		return nil, err
	}
	s.pid = binary.BigEndian.Uint16(id[len(id)-2:])
	r.logf("STM32 bootloader v%d.%d, chip ID 0x%03X, extended erase: %v", s.version>>4, s.version&0xF, s.pid, s.extErase)
	return s, nil
}

func (s *stm32) chipName() string {
	if n, ok := stmChips[s.pid]; ok {
		return n
	}
	return fmt.Sprintf("STM32 (chip ID 0x%03X)", s.pid)
}

func (s *stm32) ack(what string, timeout time.Duration) error {
	b, err := s.k.readByte(timeout)
	if err != nil {
		return fmt.Errorf("%s: %w", what, err)
	}
	if b == stmNACK {
		return fmt.Errorf("%s: the chip refused (NACK)", what)
	}
	return expect(b, stmACK, what)
}

func (s *stm32) send(cmd byte) error {
	if err := s.k.write(cmd, cmd^0xFF); err != nil {
		return err
	}
	return s.ack(fmt.Sprintf("command 0x%02X", cmd), time.Second)
}

func xorAll(b []byte) byte {
	var x byte
	for _, v := range b {
		x ^= v
	}
	return x
}

func (s *stm32) sendAddr(addr uint32) error {
	a := []byte{byte(addr >> 24), byte(addr >> 16), byte(addr >> 8), byte(addr)}
	if err := s.k.write(append(a, xorAll(a))...); err != nil {
		return err
	}
	return s.ack(fmt.Sprintf("address 0x%08X", addr), time.Second)
}

func (s *stm32) read(addr uint32, n int) ([]byte, error) {
	if err := s.send(0x11); err != nil {
		if bytes.IndexByte(s.cmds, 0x11) < 0 {
			return nil, err
		}
		return nil, fmt.Errorf("the chip is read-protected (%v)", err)
	}
	if err := s.sendAddr(addr); err != nil {
		return nil, err
	}
	if err := s.k.write(byte(n-1), byte(n-1)^0xFF); err != nil {
		return nil, err
	}
	if err := s.ack("read length", time.Second); err != nil {
		return nil, err
	}
	return s.k.readN(n, 2*time.Second)
}

func (s *stm32) flashSizeKB() int {
	reg := stmFlashSizeReg(s.pid)
	if reg == 0 {
		return 0
	}
	b, err := s.read(reg, 2)
	if err != nil {
		s.k.drain(50 * time.Millisecond)
		return 0
	}
	return int(binary.LittleEndian.Uint16(b))
}

func (s *stm32) erase(r *reporter) error {
	r.status("Erasing the chip (STM32 mass erase can take up to 30 seconds)…")
	if s.extErase {
		if err := s.send(0x44); err != nil {
			return err
		}
		if err := s.k.write(0xFF, 0xFF, 0x00); err != nil {
			return err
		}
	} else {
		if err := s.send(0x43); err != nil {
			return err
		}
		if err := s.k.write(0xFF, 0x00); err != nil {
			return err
		}
	}
	if err := s.ack("erase", 60*time.Second); err != nil {
		return fmt.Errorf("%w — if the chip is read-protected it needs unprotecting with ST's tools first", err)
	}
	return nil
}

func (s *stm32) write(addr uint32, data []byte) error {
	if err := s.send(0x31); err != nil {
		return fmt.Errorf("write: %w (is the chip write- or read-protected?)", err)
	}
	if err := s.sendAddr(addr); err != nil {
		return err
	}
	pkt := append([]byte{byte(len(data) - 1)}, data...)
	pkt = append(pkt, xorAll(pkt))
	if err := s.k.write(pkt...); err != nil {
		return err
	}
	return s.ack(fmt.Sprintf("write at 0x%08X", addr), 3*time.Second)
}

func (s *stm32) Go(addr uint32) error {
	if err := s.send(0x21); err != nil {
		return err
	}
	return s.sendAddr(addr)
}

func flashSTM32(s *stm32, base uint32, img []byte, r *reporter) error {
	if base < stmFlashBase || base >= stmFlashBase+0x1000000 {
		return fmt.Errorf("address 0x%08X isn't in STM32 flash (it starts at 0x08000000)", base)
	}
	if kb := s.flashSizeKB(); kb > 0 {
		r.logf("Flash size register says %d KB", kb)
		end := int(base-stmFlashBase) + len(img)
		// Many Blue Pill F103C8s really have 128 KB, so only warn there.
		if end > kb*1024 && !(s.pid == 0x410 && end <= 128*1024) {
			return fmt.Errorf("the firmware (%s) is bigger than this chip's %d KB of flash", humanBytes(len(img)), kb)
		}
		if end > kb*1024 {
			r.logf("Firmware is bigger than the reported 64 KB — trying anyway (most Blue Pills have 128 KB)")
		}
	}
	if rem := len(img) % 4; rem != 0 {
		img = append(append([]byte{}, img...), bytes.Repeat([]byte{0xFF}, 4-rem)...)
	}
	if err := s.erase(r); err != nil {
		return err
	}
	r.status("Writing firmware…")
	const chunk = 256
	for off := 0; off < len(img); off += chunk {
		if r.cancelled() {
			return errCancelled
		}
		blk := img[off:min(off+chunk, len(img))]
		if allFF(blk) {
			r.progress(off+len(blk), len(img))
			continue // erased flash is already 0xFF
		}
		if err := s.write(base+uint32(off), blk); err != nil {
			return err
		}
		r.progress(off+len(blk), len(img))
	}
	r.status("Reading it back to check every byte…")
	for off := 0; off < len(img); off += chunk {
		blk := img[off:min(off+chunk, len(img))]
		got, err := s.read(base+uint32(off), len(blk))
		if err != nil {
			return err
		}
		if !bytes.Equal(got, blk) {
			return fmt.Errorf("verification failed at 0x%08X — the chip doesn't hold what we wrote", base+uint32(off))
		}
		r.progress(off+len(blk), len(img))
	}
	return nil
}

func allFF(b []byte) bool {
	for _, v := range b {
		if v != 0xFF {
			return false
		}
	}
	return true
}
