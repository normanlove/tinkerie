// Flashie — simulated AVR (STK500 v1/v2, AVR109) and STM32 bootloaders,
// plus the demo-mode device set. Used by tests and FLASHIE_DEMO=1.
// Made by nrmluvdev.

package flashie

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// streamSim is a byte-oriented fake device.
type streamSim struct {
	mu       sync.Mutex
	out      bytes.Buffer
	in       []byte
	baud     int
	wantBaud int // 0 = any
	dtr      bool
	active   bool // bootloader is listening
	needDTR  bool // only becomes active after a DTR reset
	parse    func() bool
	delay    time.Duration
	flash    []byte
	sig      [3]byte
	addr     int
	// STM32 state
	stage  int
	cmd    byte
	sAddr  uint32
	inited bool
	seq    byte
}

func (s *streamSim) emit(b ...byte) { s.out.Write(b) }

func (s *streamSim) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.active || (s.wantBaud != 0 && s.baud != s.wantBaud) {
		return len(p), nil
	}
	s.in = append(s.in, p...)
	for len(s.in) > 0 && s.parse() {
	}
	return len(p), nil
}

func (s *streamSim) ReadTimeout(p []byte, d time.Duration) (int, error) {
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

func (s *streamSim) SetBaud(b int) error {
	s.mu.Lock()
	s.baud = b
	s.in = nil
	s.mu.Unlock()
	return nil
}
func (s *streamSim) SetDTR(on bool) error {
	s.mu.Lock()
	if on && !s.dtr && s.needDTR {
		s.active, s.in, s.stage = true, nil, 0
	}
	s.dtr = on
	s.mu.Unlock()
	return nil
}
func (s *streamSim) SetRTS(bool) error { return nil }
func (s *streamSim) Purge() error      { s.mu.Lock(); s.out.Reset(); s.mu.Unlock(); return nil }
func (s *streamSim) Close() error      { return nil }

func (s *streamSim) take(n int) []byte { b := s.in[:n]; s.in = s.in[n:]; return b }

// ---- STK500 v1 (Optiboot) ----

func newSimSTK1(baud int, sig [3]byte, flash int) *streamSim {
	s := &streamSim{baud: baud, wantBaud: baud, needDTR: true, sig: sig, flash: bytes.Repeat([]byte{0xFF}, flash)}
	s.parse = func() bool {
		need := map[byte]int{0x30: 2, 0x75: 2, 0x50: 2, 0x51: 2, 0x55: 4, 0x74: 5}
		c := s.in[0]
		n, ok := need[c]
		if c == 0x64 {
			if len(s.in) < 4 {
				return false
			}
			n, ok = 5+(int(s.in[1])<<8|int(s.in[2])), true
		}
		if !ok {
			s.take(1)
			return true
		}
		if len(s.in) < n {
			return false
		}
		b := s.take(n)
		if b[n-1] != 0x20 {
			s.emit(0x15)
			return true
		}
		switch c {
		case 0x75:
			s.emit(0x14, s.sig[0], s.sig[1], s.sig[2], 0x10)
		case 0x55:
			s.addr = (int(b[1]) | int(b[2])<<8) * 2
			s.emit(0x14, 0x10)
		case 0x64:
			copy(s.flash[s.addr:], b[4:n-1])
			time.Sleep(s.delay)
			s.emit(0x14, 0x10)
		case 0x74:
			ln := int(b[1])<<8 | int(b[2])
			s.emit(0x14)
			s.emit(s.flash[s.addr : s.addr+ln]...)
			s.emit(0x10)
		default:
			s.emit(0x14, 0x10)
		}
		return true
	}
	return s
}

// ---- STK500 v2 (Wiring / Mega) ----

func newSimSTK2(sig [3]byte, flash int) *streamSim {
	s := &streamSim{baud: 115200, wantBaud: 115200, needDTR: true, sig: sig, flash: bytes.Repeat([]byte{0xFF}, flash)}
	s.parse = func() bool {
		if s.in[0] != 0x1B {
			s.take(1)
			return true
		}
		if len(s.in) < 5 {
			return false
		}
		n := int(s.in[2])<<8 | int(s.in[3])
		if len(s.in) < 6+n {
			return false
		}
		f := s.take(6 + n)
		seq, body := f[1], f[5:5+n]
		var resp []byte
		switch body[0] {
		case 0x01:
			resp = append([]byte{0x01, 0x00, 8}, "AVRISP_2"...)
		case 0x1B:
			resp = []byte{0x1B, 0x00, s.sig[body[4]], 0x00}
		case 0x06:
			w := binary.BigEndian.Uint32(body[1:]) & 0x7FFFFFFF
			s.addr = int(w) * 2
			resp = []byte{0x06, 0x00}
		case 0x13:
			ln := int(body[1])<<8 | int(body[2])
			copy(s.flash[s.addr:], body[10:10+ln])
			time.Sleep(s.delay)
			resp = []byte{0x13, 0x00}
		case 0x14:
			ln := int(body[1])<<8 | int(body[2])
			resp = append([]byte{0x14, 0x00}, s.flash[s.addr:s.addr+ln]...)
			resp = append(resp, 0x00)
		default:
			resp = []byte{body[0], 0x00}
		}
		msg := []byte{0x1B, seq, byte(len(resp) >> 8), byte(len(resp)), 0x0E}
		msg = append(msg, resp...)
		var cs byte
		for _, v := range msg {
			cs ^= v
		}
		s.emit(append(msg, cs)...)
		return true
	}
	return s
}

// ---- AVR109 (Caterina) ----

func newSimAVR109(active bool) *streamSim {
	s := &streamSim{active: active, sig: [3]byte{0x1E, 0x95, 0x87}, flash: bytes.Repeat([]byte{0xFF}, 32768)}
	s.parse = func() bool {
		c := s.in[0]
		switch c {
		case 'S':
			s.take(1)
			s.emit([]byte("CATERIN")...)
		case 'b':
			s.take(1)
			s.emit('Y', 0, 128)
		case 's':
			s.take(1)
			s.emit(s.sig[2], s.sig[1], s.sig[0])
		case 'P', 'L', 'E':
			s.take(1)
			s.emit('\r')
		case 'e':
			s.take(1)
			for i := range s.flash[:28672] {
				s.flash[i] = 0xFF
			}
			s.emit('\r')
		case 'A':
			if len(s.in) < 3 {
				return false
			}
			b := s.take(3)
			s.addr = (int(b[1])<<8 | int(b[2])) * 2
			s.emit('\r')
		case 'B':
			if len(s.in) < 4 {
				return false
			}
			n := int(s.in[1])<<8 | int(s.in[2])
			if len(s.in) < 4+n {
				return false
			}
			b := s.take(4 + n)
			copy(s.flash[s.addr:], b[4:])
			s.addr += n
			time.Sleep(s.delay)
			s.emit('\r')
		case 'g':
			if len(s.in) < 4 {
				return false
			}
			b := s.take(4)
			n := int(b[1])<<8 | int(b[2])
			s.emit(s.flash[s.addr : s.addr+n]...)
			s.addr += n
		default:
			s.take(1)
			s.emit('?')
		}
		return true
	}
	return s
}

// ---- STM32 UART bootloader ----

func newSimSTM32() *streamSim {
	s := &streamSim{active: true, flash: bytes.Repeat([]byte{0xFF}, 128*1024)}
	mem := func(addr uint32, n int) []byte {
		switch {
		case addr >= stmFlashBase && addr < stmFlashBase+uint32(len(s.flash)):
			o := addr - stmFlashBase
			return s.flash[o : int(o)+n]
		case addr == 0x1FFFF7E0:
			return []byte{64, 0}
		}
		return make([]byte, n)
	}
	s.parse = func() bool {
		if !s.inited {
			if s.take(1)[0] == 0x7F {
				s.inited = true
				s.emit(stmACK)
			}
			return true
		}
		switch s.stage {
		case 0:
			if len(s.in) < 2 {
				return false
			}
			b := s.take(2)
			if b[0]^b[1] != 0xFF {
				s.emit(stmNACK)
				return true
			}
			s.cmd = b[0]
			s.emit(stmACK)
			switch s.cmd {
			case 0x00:
				cmds := []byte{0x00, 0x01, 0x02, 0x11, 0x21, 0x31, 0x43, 0x63, 0x73, 0x82, 0x92}
				s.emit(byte(len(cmds)), 0x22)
				s.emit(cmds...)
				s.emit(stmACK)
			case 0x02:
				s.emit(1, 0x04, 0x10, stmACK)
			case 0x11, 0x31, 0x21, 0x43:
				s.stage = 1
			}
		case 1:
			if s.cmd == 0x43 {
				if len(s.in) < 2 {
					return false
				}
				s.take(2)
				for i := range s.flash {
					s.flash[i] = 0xFF
				}
				time.Sleep(s.delay * 20)
				s.emit(stmACK)
				s.stage = 0
				return true
			}
			if len(s.in) < 5 {
				return false
			}
			b := s.take(5)
			s.sAddr = binary.BigEndian.Uint32(b)
			s.emit(stmACK)
			if s.cmd == 0x21 {
				s.stage, s.inited = 0, false
				return true
			}
			s.stage = 2
		case 2:
			if s.cmd == 0x11 {
				if len(s.in) < 2 {
					return false
				}
				b := s.take(2)
				s.emit(stmACK)
				s.emit(mem(s.sAddr, int(b[0])+1)...)
			} else {
				if len(s.in) < 1 {
					return false
				}
				n := int(s.in[0]) + 1
				if len(s.in) < n+2 {
					return false
				}
				b := s.take(n + 2)
				if xorAll(b[:n+1]) != b[n+1] {
					s.emit(stmNACK)
				} else {
					copy(mem(s.sAddr, n), b[1:n+1])
					time.Sleep(s.delay)
					s.emit(stmACK)
				}
			}
			s.stage = 0
		}
		return true
	}
	return s
}

// ------------------------------------------------------------ demo mode ---

var (
	demoMu     sync.Mutex
	demoDevs   = map[string]Port{}
	demoPicoUp bool
	demoDirs   = map[string]string{}
)

func demoPorts() []PortInfo {
	ps := []PortInfo{
		{Name: "COM3", USB: true, VID: 0x1A86, PID: 0x7523, Friendly: "USB-SERIAL CH340 (COM3)"},
		{Name: "COM4", USB: true, VID: 0x10C4, PID: 0xEA60, Friendly: "Silicon Labs CP210x USB to UART Bridge (COM4)"},
		{Name: "COM5", USB: true, VID: 0x2341, PID: 0x0043, Friendly: "Arduino Uno (COM5)"},
		{Name: "COM6", USB: true, VID: 0x2A03, PID: 0x0042, Friendly: "Arduino Mega 2560 (COM6)"},
		{Name: "COM7", USB: true, VID: 0x303A, PID: 0x1001, Friendly: "USB JTAG/serial debug unit (COM7)"},
		{Name: "COM8", USB: true, VID: 0x2341, PID: 0x8036, Friendly: "Arduino Leonardo (COM8)"},
		{Name: "COM9", USB: true, VID: 0x0403, PID: 0x6001, Friendly: "USB Serial Port (COM9)"},
		{Name: "COM10", USB: true, VID: 0x2E8A, PID: 0x000A, Friendly: "USB Serial Device (COM10)"},
		{Name: "COM1", Friendly: "Communications Port (COM1)"},
	}
	for i := range ps {
		ps[i].Guess = guessBoard(ps[i])
	}
	demoMu.Lock()
	ps = append(ps, demoExtra...)
	demoMu.Unlock()
	return ps
}

func demoOpen(name string) (Port, error) {
	demoMu.Lock()
	defer demoMu.Unlock()
	if p, ok := demoDevs[name]; ok {
		return p, nil
	}
	var p Port
	switch name {
	case "COM3":
		p = newSimChip(0, 2*time.Millisecond)
	case "COM4":
		p = newSimChip(esp8266SimID, time.Millisecond)
	case "COM7":
		p = newSimChip(9, 2*time.Millisecond)
	case "COM5":
		s := newSimSTK1(115200, [3]byte{0x1E, 0x95, 0x0F}, 32768)
		s.delay = 4 * time.Millisecond
		p = s
	case "COM6":
		s := newSimSTK2([3]byte{0x1E, 0x98, 0x01}, 262144)
		s.delay = 6 * time.Millisecond
		p = s
	case "COM8":
		s := newSimAVR109(false)
		s.delay = 4 * time.Millisecond
		p = s
	case "COM9":
		s := newSimSTM32()
		s.delay = 3 * time.Millisecond
		p = s
	default:
		p = &streamSim{parse: func() bool { return false }}
	}
	demoDevs[name] = p
	return p, nil
}

func demoTouch(name string) {
	p, _ := demoOpen(name)
	demoMu.Lock()
	defer demoMu.Unlock()
	if s, ok := p.(*streamSim); ok && name == "COM8" {
		s.mu.Lock()
		s.active = true
		s.mu.Unlock()
	}
	if name == "COM10" {
		demoPicoUp = true
	}
}

func demoDrive(letter, info string) uf2Drive {
	dir, ok := demoDirs[letter]
	if !ok {
		dir, _ = os.MkdirTemp("", "flashie-uf2-")
		os.WriteFile(filepath.Join(dir, "INFO_UF2.TXT"), []byte(info), 0o644)
		demoDirs[letter] = dir
	}
	d := parseInfoUF2(dir, info)
	d.Letter, d.Label = letter, map[bool]string{true: "RP2350", false: "RPI-RP2"}[d.isRP2350()]
	return d
}

func demoDrives() []uf2Drive {
	demoMu.Lock()
	defer demoMu.Unlock()
	out := []uf2Drive{demoDrive("P:", "UF2 Bootloader v1.0\nModel: Raspberry Pi RP2350\nBoard-ID: RP2350\n")}
	if demoPicoUp {
		out = append(out, demoDrive("Q:", "UF2 Bootloader v3.0\nModel: Raspberry Pi RP2\nBoard-ID: RPI-RP2\n"))
	}
	return out
}

// resetDemo puts every simulated device back to its power-on state.
func resetDemo() {
	demoMu.Lock()
	defer demoMu.Unlock()
	demoDevs = map[string]Port{}
	demoPicoUp = false
	demoMPYPorts = map[string]*simMPY{}
	demoExtra = nil
}
