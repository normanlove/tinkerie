// Flashie — native AVR bootloader protocols for Arduino boards:
//   * STK500 v1  (Optiboot / ATmegaBOOT — Uno, Nano, Pro Mini)
//   * STK500 v2  (Wiring bootloader — Mega 2560 / 1280)
//   * AVR109     (Caterina — Leonardo, Micro, Pro Micro)
// Made by nrmluvdev.

package flashie

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"time"
)

// reporter carries UI callbacks into the protocol code.
type reporter struct {
	Log      func(string)
	Status   func(string)
	Progress func(done, total int)
	Cancel   func() bool
}

func (r *reporter) logf(f string, a ...any) {
	if r.Log != nil {
		r.Log(fmt.Sprintf(f, a...))
	}
}
func (r *reporter) status(s string) {
	if r.Status != nil {
		r.Status(s)
	}
}
func (r *reporter) progress(d, t int) {
	if r.Progress != nil {
		r.Progress(d, t)
	}
}
func (r *reporter) cancelled() bool { return r.Cancel != nil && r.Cancel() }

type avrMCU struct {
	Name  string
	Sig   [3]byte
	Flash int
	Page  int
}

var avrMCUs = []avrMCU{
	{"ATmega328P", [3]byte{0x1E, 0x95, 0x0F}, 32768, 128},
	{"ATmega328", [3]byte{0x1E, 0x95, 0x14}, 32768, 128},
	{"ATmega328PB", [3]byte{0x1E, 0x95, 0x16}, 32768, 128},
	{"ATmega168", [3]byte{0x1E, 0x94, 0x06}, 16384, 128},
	{"ATmega168PA", [3]byte{0x1E, 0x94, 0x0B}, 16384, 128},
	{"ATmega8", [3]byte{0x1E, 0x93, 0x07}, 8192, 64},
	{"ATmega2560", [3]byte{0x1E, 0x98, 0x01}, 262144, 256},
	{"ATmega1280", [3]byte{0x1E, 0x97, 0x03}, 131072, 256},
	{"ATmega32U4", [3]byte{0x1E, 0x95, 0x87}, 32768, 128},
	{"ATmega1284P", [3]byte{0x1E, 0x97, 0x05}, 131072, 256},
	{"ATmega644P", [3]byte{0x1E, 0x96, 0x0A}, 65536, 256},
}

func mcuBySig(sig []byte) *avrMCU {
	for i := range avrMCUs {
		if bytes.Equal(avrMCUs[i].Sig[:], sig) {
			return &avrMCUs[i]
		}
	}
	return nil
}

// bootloaderReserve is how much flash each bootloader keeps for itself.
var bootloaderReserve = map[string]int{"stk500v1": 512, "stk500v1-old": 2048, "stk500v2": 8192, "avr109": 4096}

// avrReset pulses DTR/RTS so the auto-reset capacitor resets the chip.
func avrReset(p Port) {
	p.SetDTR(false)
	p.SetRTS(false)
	time.Sleep(250 * time.Millisecond)
	p.SetDTR(true)
	p.SetRTS(true)
	time.Sleep(50 * time.Millisecond)
}

// avrSession is one connected bootloader.
type avrSession interface {
	Signature() ([]byte, error)
	Program(img []byte, page int, r *reporter) error
	Verify(img []byte, page int, r *reporter) error
	Leave() error
	Proto() string
}

// ---------------------------------------------------------- STK500 v1 ---

type stk1 struct{ k *link }

const (
	stkOK      = 0x10
	stkInSync  = 0x14
	stkCrcEOP  = 0x20
	stkGetSync = 0x30
)

func connectSTK1(p Port) (*stk1, error) {
	s := &stk1{k: newLink(p)}
	avrReset(p)
	s.k.drain(30 * time.Millisecond)
	for i := 0; i < 12; i++ {
		s.k.write(stkGetSync, stkCrcEOP)
		b, err := s.k.readN(2, 120*time.Millisecond)
		if err == nil && b[0] == stkInSync && b[1] == stkOK {
			s.k.drain(30 * time.Millisecond)
			// Confirm sync once more after draining duplicates.
			s.k.write(stkGetSync, stkCrcEOP)
			if b, err := s.k.readN(2, 300*time.Millisecond); err == nil && b[0] == stkInSync && b[1] == stkOK {
				return s, nil
			}
		}
		s.k.buf = nil
	}
	return nil, errTimeout
}

func (s *stk1) cmd(what string, n int, timeout time.Duration, b ...byte) ([]byte, error) {
	if err := s.k.write(append(b, stkCrcEOP)...); err != nil {
		return nil, err
	}
	c, err := s.k.readByte(timeout)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", what, err)
	}
	if c != stkInSync {
		return nil, fmt.Errorf("%s: the bootloader lost sync (got 0x%02X)", what, c)
	}
	data, err := s.k.readN(n, timeout)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", what, err)
	}
	c, err = s.k.readByte(timeout)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", what, err)
	}
	if c != stkOK {
		return nil, fmt.Errorf("%s: the bootloader reported a failure (0x%02X)", what, c)
	}
	return data, nil
}

func (s *stk1) Proto() string { return "stk500v1" }

func (s *stk1) Signature() ([]byte, error) {
	return s.cmd("read signature", 3, time.Second, 0x75)
}

func (s *stk1) load(byteAddr int) error {
	w := byteAddr / 2
	_, err := s.cmd("set address", 0, time.Second, 0x55, byte(w), byte(w>>8))
	return err
}

func (s *stk1) Program(img []byte, page int, r *reporter) error {
	if _, err := s.cmd("enter programming mode", 0, time.Second, 0x50); err != nil {
		return err
	}
	for a := 0; a < len(img); a += page {
		if r.cancelled() {
			return errCancelled
		}
		chunk := img[a:min(a+page, len(img))]
		if err := s.load(a); err != nil {
			return err
		}
		pkt := append([]byte{0x64, byte(len(chunk) >> 8), byte(len(chunk)), 'F'}, chunk...)
		if _, err := s.cmd(fmt.Sprintf("write page at 0x%04X", a), 0, 2*time.Second, pkt...); err != nil {
			return err
		}
		r.progress(a+len(chunk), len(img))
	}
	return nil
}

func (s *stk1) Verify(img []byte, page int, r *reporter) error {
	for a := 0; a < len(img); a += page {
		chunk := img[a:min(a+page, len(img))]
		if err := s.load(a); err != nil {
			return err
		}
		got, err := s.cmd("read back", len(chunk), 2*time.Second, 0x74, byte(len(chunk)>>8), byte(len(chunk)), 'F')
		if err != nil {
			return err
		}
		if !bytes.Equal(got, chunk) {
			return fmt.Errorf("verification failed at 0x%04X — the chip doesn't hold what we wrote", a)
		}
		r.progress(a+len(chunk), len(img))
	}
	return nil
}

func (s *stk1) Leave() error {
	_, err := s.cmd("leave programming mode", 0, time.Second, 0x51)
	return err
}

// ---------------------------------------------------------- STK500 v2 ---

type stk2 struct {
	k   *link
	seq byte
}

func connectSTK2(p Port) (*stk2, error) {
	s := &stk2{k: newLink(p), seq: 1}
	avrReset(p)
	s.k.drain(30 * time.Millisecond)
	for i := 0; i < 8; i++ {
		resp, err := s.xfer([]byte{0x01}, 200*time.Millisecond)
		if err == nil && len(resp) >= 2 && resp[0] == 0x01 && resp[1] == 0x00 {
			return s, nil
		}
		s.k.buf = nil
	}
	return nil, errTimeout
}

func (s *stk2) xfer(body []byte, timeout time.Duration) ([]byte, error) {
	msg := []byte{0x1B, s.seq, byte(len(body) >> 8), byte(len(body)), 0x0E}
	msg = append(msg, body...)
	var cs byte
	for _, b := range msg {
		cs ^= b
	}
	msg = append(msg, cs)
	if err := s.k.write(msg...); err != nil {
		return nil, err
	}
	deadline := time.Now().Add(timeout)
	for {
		b, err := s.k.readByte(time.Until(deadline))
		if err != nil {
			return nil, err
		}
		if b != 0x1B {
			continue
		}
		h, err := s.k.readN(4, time.Until(deadline))
		if err != nil {
			return nil, err
		}
		if h[3] != 0x0E {
			continue
		}
		n := int(h[1])<<8 | int(h[2])
		rest, err := s.k.readN(n+1, time.Until(deadline))
		if err != nil {
			return nil, err
		}
		c := byte(0x1B)
		for _, v := range h {
			c ^= v
		}
		for _, v := range rest[:n] {
			c ^= v
		}
		if c != rest[n] {
			return nil, errors.New("STK500v2 checksum error — noisy connection")
		}
		if h[0] != s.seq {
			continue
		}
		s.seq++
		return rest[:n], nil
	}
}

func (s *stk2) cmd(what string, body []byte, timeout time.Duration) ([]byte, error) {
	r, err := s.xfer(body, timeout)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", what, err)
	}
	if len(r) < 2 || r[0] != body[0] {
		return nil, fmt.Errorf("%s: unexpected reply", what)
	}
	if r[1] != 0x00 {
		return nil, fmt.Errorf("%s: the bootloader reported a failure (0x%02X)", what, r[1])
	}
	return r, nil
}

func (s *stk2) Proto() string { return "stk500v2" }

func (s *stk2) Signature() ([]byte, error) {
	sig := make([]byte, 3)
	for i := 0; i < 3; i++ {
		r, err := s.cmd("read signature", []byte{0x1B, 0x04, 0x30, 0x00, byte(i), 0x00}, time.Second)
		if err != nil {
			return nil, err
		}
		if len(r) < 3 {
			return nil, errors.New("read signature: short reply")
		}
		sig[i] = r[2]
	}
	return sig, nil
}

func (s *stk2) load(byteAddr int) error {
	w := uint32(byteAddr/2) | 0x80000000
	_, err := s.cmd("set address", []byte{0x06, byte(w >> 24), byte(w >> 16), byte(w >> 8), byte(w)}, time.Second)
	return err
}

func (s *stk2) Program(img []byte, page int, r *reporter) error {
	if _, err := s.cmd("enter programming mode", []byte{0x10, 200, 100, 25, 32, 0, 0x53, 3, 0xAC, 0x53, 0, 0}, time.Second); err != nil {
		return err
	}
	for a := 0; a < len(img); a += page {
		if r.cancelled() {
			return errCancelled
		}
		chunk := img[a:min(a+page, len(img))]
		if err := s.load(a); err != nil {
			return err
		}
		body := append([]byte{0x13, byte(len(chunk) >> 8), byte(len(chunk)), 0xC1, 0x0A, 0x40, 0x4C, 0x20, 0x00, 0x00}, chunk...)
		if _, err := s.cmd(fmt.Sprintf("write page at 0x%05X", a), body, 2*time.Second); err != nil {
			return err
		}
		r.progress(a+len(chunk), len(img))
	}
	return nil
}

func (s *stk2) Verify(img []byte, page int, r *reporter) error {
	for a := 0; a < len(img); a += page {
		chunk := img[a:min(a+page, len(img))]
		if err := s.load(a); err != nil {
			return err
		}
		resp, err := s.cmd("read back", []byte{0x14, byte(len(chunk) >> 8), byte(len(chunk)), 0x20}, 2*time.Second)
		if err != nil {
			return err
		}
		if len(resp) < 2+len(chunk) || !bytes.Equal(resp[2:2+len(chunk)], chunk) {
			return fmt.Errorf("verification failed at 0x%05X — the chip doesn't hold what we wrote", a)
		}
		r.progress(a+len(chunk), len(img))
	}
	return nil
}

func (s *stk2) Leave() error {
	_, err := s.cmd("leave programming mode", []byte{0x11, 1, 1}, time.Second)
	return err
}

// --------------------------------------------------------------- AVR109 ---

type avr109 struct {
	k     *link
	block int
}

func connectAVR109(p Port) (*avr109, error) {
	s := &avr109{k: newLink(p), block: 128}
	s.k.drain(50 * time.Millisecond)
	for i := 0; i < 6; i++ {
		s.k.write('S')
		id, err := s.k.readN(7, 400*time.Millisecond)
		if err == nil && len(strings.TrimSpace(string(id))) > 0 {
			if b, err := s.ask("block support", 3, 'b'); err == nil && b[0] == 'Y' {
				if sz := int(b[1])<<8 | int(b[2]); sz > 0 && sz <= 1024 {
					s.block = sz
				}
			}
			return s, nil
		}
		s.k.drain(100 * time.Millisecond)
	}
	return nil, errTimeout
}

func (s *avr109) ask(what string, n int, b ...byte) ([]byte, error) {
	if err := s.k.write(b...); err != nil {
		return nil, err
	}
	r, err := s.k.readN(n, 3*time.Second)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", what, err)
	}
	return r, nil
}

func (s *avr109) cr(what string, timeout time.Duration, b ...byte) error {
	if err := s.k.write(b...); err != nil {
		return err
	}
	c, err := s.k.readByte(timeout)
	if err != nil {
		return fmt.Errorf("%s: %w", what, err)
	}
	return expect(c, '\r', what)
}

func (s *avr109) Proto() string { return "avr109" }

func (s *avr109) Signature() ([]byte, error) {
	r, err := s.ask("read signature", 3, 's')
	if err != nil {
		return nil, err
	}
	return []byte{r[2], r[1], r[0]}, nil
}

func (s *avr109) Program(img []byte, _ int, r *reporter) error {
	if err := s.cr("enter programming mode", time.Second, 'P'); err != nil {
		return err
	}
	r.status("Erasing the old sketch…")
	if err := s.cr("erase", 15*time.Second, 'e'); err != nil {
		return err
	}
	r.status("Writing firmware…")
	if err := s.cr("set address", time.Second, 'A', 0, 0); err != nil {
		return err
	}
	for a := 0; a < len(img); a += s.block {
		if r.cancelled() {
			return errCancelled
		}
		chunk := img[a:min(a+s.block, len(img))]
		if len(chunk)%2 == 1 {
			chunk = append(append([]byte{}, chunk...), 0xFF)
		}
		pkt := append([]byte{'B', byte(len(chunk) >> 8), byte(len(chunk)), 'F'}, chunk...)
		if err := s.cr(fmt.Sprintf("write block at 0x%04X", a), 3*time.Second, pkt...); err != nil {
			return err
		}
		r.progress(a+len(chunk), len(img))
	}
	return nil
}

func (s *avr109) Verify(img []byte, _ int, r *reporter) error {
	if err := s.cr("set address", time.Second, 'A', 0, 0); err != nil {
		return err
	}
	for a := 0; a < len(img); a += s.block {
		chunk := img[a:min(a+s.block, len(img))]
		n := len(chunk) + len(chunk)%2
		got, err := s.ask("read back", n, 'g', byte(n>>8), byte(n), 'F')
		if err != nil {
			return err
		}
		if !bytes.Equal(got[:len(chunk)], chunk) {
			return fmt.Errorf("verification failed at 0x%04X — the chip doesn't hold what we wrote", a)
		}
		r.progress(a+len(chunk), len(img))
	}
	return nil
}

func (s *avr109) Leave() error {
	s.cr("leave programming mode", time.Second, 'L')
	return s.cr("exit bootloader", time.Second, 'E')
}

// --------------------------------------------------------- shared flow ---

// flashAVR writes img (starting at address 0) through an open session.
func flashAVR(sess avrSession, mcu *avrMCU, img []byte, maxSketch int, r *reporter) error {
	if len(img) > maxSketch {
		if len(img) >= mcu.Flash-1024 {
			return fmt.Errorf("this HEX file includes a bootloader (%s). Use the normal sketch .hex, not the “with_bootloader” one", humanBytes(len(img)))
		}
		return fmt.Errorf("the sketch is %s but this board only has room for %s", humanBytes(len(img)), humanBytes(maxSketch))
	}
	// Pad to a whole page so every write is a full page.
	if rem := len(img) % mcu.Page; rem != 0 {
		img = append(append([]byte{}, img...), bytes.Repeat([]byte{0xFF}, mcu.Page-rem)...)
	}
	r.status("Writing firmware…")
	if err := sess.Program(img, mcu.Page, r); err != nil {
		return err
	}
	r.status("Reading it back to check every byte…")
	if err := sess.Verify(img, mcu.Page, r); err != nil {
		return err
	}
	r.status("Starting your sketch…")
	if err := sess.Leave(); err != nil {
		r.logf("Leave programming mode: %v (harmless — the board restarts on its own)", err)
	}
	return nil
}
