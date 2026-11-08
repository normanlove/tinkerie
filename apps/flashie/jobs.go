// Flashie — device discovery, universal "Identify", and the flash dispatcher
// that routes each job to the right protocol. Made by nrmluvdev.

package flashie

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// --------------------------------------------------------------- devices ---

func getDevices() []PortInfo {
	var out []PortInfo
	for _, p := range getPorts() {
		p.Kind = "serial"
		out = append(out, p)
	}
	for _, d := range findUF2Drives() {
		d := d
		label := d.Label
		if label == "" {
			label = "UF2 drive"
		}
		out = append(out, PortInfo{Name: d.Letter, Kind: "drive", Device: d.Root, USB: true, VID: 0x2E8A,
			Friendly: label + " (" + d.Letter + ")", Guess: guessDrive(d), drive: &d})
	}
	if out == nil {
		out = []PortInfo{}
	}
	return out
}

func findDevice(name string) (PortInfo, bool) {
	for _, p := range getDevices() {
		if strings.EqualFold(p.Name, name) {
			return p, true
		}
	}
	return PortInfo{Name: name, Kind: "serial"}, false
}

func openPortEx(name string, baud int, even bool) (Port, error) {
	if demoMode {
		return demoOpen(name)
	}
	return openSerialEx(name, baud, even)
}

// touch1200 does the "open at 1200 baud and close" trick that reboots
// Leonardo/Micro and Pico (SDK / Arduino firmware) into their bootloaders.
func touch1200(name string, r *reporter) {
	r.logf("1200-baud touch on %s", name)
	if demoMode {
		demoTouch(name)
		return
	}
	p, err := openSerialEx(name, 1200, false)
	if err != nil {
		r.logf("Touch: %v", err)
		return
	}
	p.SetDTR(false)
	time.Sleep(100 * time.Millisecond)
	p.Close()
}

// waitForSerial waits for a new COM port to appear (a bootloader after a
// touch). Falls back to the original port if it comes back.
func waitForSerial(before map[string]bool, old string, timeout time.Duration) (string, error) {
	if demoMode {
		time.Sleep(400 * time.Millisecond)
		return old, nil
	}
	deadline := time.Now().Add(timeout)
	start := time.Now()
	for time.Now().Before(deadline) {
		cur := map[string]bool{}
		for _, p := range listPorts() {
			cur[p.Name] = true
			if !before[p.Name] {
				time.Sleep(300 * time.Millisecond) // let the driver settle
				return p.Name, nil
			}
		}
		if time.Since(start) > 2500*time.Millisecond && cur[old] {
			return old, nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return "", errTimeout
}

func portSet() map[string]bool {
	m := map[string]bool{}
	for _, p := range getPorts() {
		m[p.Name] = true
	}
	return m
}

func driveSet() map[string]bool {
	m := map[string]bool{}
	for _, d := range findUF2Drives() {
		m[d.Root] = true
	}
	return m
}

// ------------------------------------------------------------------ plan ---

// plan says how a particular device will be flashed.
type plan struct {
	Family string
	Proto  string
	Bauds  []int
	Board  *Board
}

func planFor(pi PortInfo, b *Board, hint *ChipInfo) *plan {
	if pi.Kind == "drive" {
		return &plan{Family: famRP2, Proto: "uf2", Board: b}
	}
	if b != nil {
		return &plan{Family: b.Family, Proto: b.Protocol, Bauds: b.Bauds, Board: b}
	}
	if hint != nil && hint.Family != "" {
		p := &plan{Family: hint.Family, Proto: hint.Proto}
		if hint.Baud > 0 {
			p.Bauds = []int{hint.Baud}
		}
		if hint.BoardID != "" {
			p.Board = boardByID(hint.BoardID)
		}
		return p
	}
	if pi.Guess.Family != "" {
		gb := boardByID(pi.Guess.BoardID)
		if gb != nil {
			return &plan{Family: gb.Family, Proto: gb.Protocol, Bauds: gb.Bauds, Board: gb}
		}
		return &plan{Family: pi.Guess.Family, Proto: map[string]string{famESP: "esp", famRP2: "uf2", famSTM32: "stm32"}[pi.Guess.Family]}
	}
	return nil // needs probing
}

// -------------------------------------------------------------- identify ---

// identify works out what's connected, trying each likely protocol.
func identify(pi PortInfo, b *Board, r *reporter) (*ChipInfo, error) {
	if pi.Kind == "drive" {
		d := pi.drive
		info := &ChipInfo{Family: famRP2, Proto: "uf2", Chip: d.chipName(), Supported: true, Port: pi.Name, BoardID: pi.Guess.BoardID,
			Details: [][2]string{{"Mode", "BOOTSEL (UF2 drive " + d.Letter + ")"}}}
		if d.Model != "" {
			info.Details = append(info.Details, [2]string{"Model", d.Model})
		}
		if d.BoardID != "" {
			info.Details = append(info.Details, [2]string{"Board-ID", d.BoardID})
		}
		return info, nil
	}
	var fams []string
	quick := true
	switch {
	case b != nil:
		fams, quick = []string{b.Family}, false
	case pi.Guess.Family != "":
		fams, quick = []string{pi.Guess.Family}, false
	case len(pi.Guess.Families) > 0:
		fams = pi.Guess.Families
	default:
		fams = bridgeFams
	}
	var errs []string
	for _, fam := range fams {
		var info *ChipInfo
		var err error
		switch fam {
		case famESP:
			r.status("Checking for an ESP32 / ESP8266…")
			info, err = identifyESP(pi, quick, r)
		case famAVR:
			r.status("Checking for an Arduino (AVR) bootloader…")
			info, err = identifyAVR(pi, b, r)
		case famSTM32:
			r.status("Checking for an STM32 bootloader…")
			info, err = identifySTM32(pi, b, r)
		case famRP2:
			chip := pi.Guess.Chip
			if b != nil {
				chip = b.Chip
			}
			return &ChipInfo{Family: famRP2, Proto: "uf2", Chip: chip, Supported: true, Port: pi.Name, BoardID: pi.Guess.BoardID,
				Details: [][2]string{{"Mode", "Running firmware"}},
				Note:    "When you flash, Flashie reboots it into BOOTSEL mode. If that doesn't work, hold BOOTSEL while plugging it in."}, nil
		}
		if err == nil {
			info.Port = pi.Name
			return info, nil
		}
		r.logf("%s probe: %v", fam, err)
		errs = append(errs, err.Error())
		if errors.Is(err, errPortBusy) {
			return nil, err
		}
	}
	if len(fams) == 1 {
		return nil, errors.New(errs[0])
	}
	return nil, errors.New("Flashie couldn't get a reply from the board. If it's an ESP32, hold BOOT and tap RESET; an Arduino should answer straight away; an STM32 needs BOOT0 = 1 and a reset. Or pick your board from the list")
}

var errPortBusy = errors.New("port busy")

func wrapOpen(err error) error {
	if err != nil && strings.Contains(err.Error(), "busy") {
		return fmt.Errorf("%w: %v", errPortBusy, err)
	}
	return err
}

func identifyESP(pi PortInfo, quick bool, r *reporter) (*ChipInfo, error) {
	p, err := openPortEx(pi.Name, 115200, false)
	if err != nil {
		return nil, wrapOpen(err)
	}
	defer p.Close()
	l := NewLoader(p, pi.Guess.JTAG, pi.Guess.NativeUSB && !pi.Guess.JTAG)
	l.Log, l.Status = func(s string) { r.logf("%s", s) }, func(string) {}
	if quick {
		l.Attempts = 2
	}
	if err := l.Connect(); err != nil {
		return nil, err
	}
	info, err := l.DetectChip()
	l.HardReset()
	if err != nil {
		return nil, err
	}
	if info.MAC != "" {
		info.Details = append(info.Details, [2]string{"MAC", info.MAC})
	}
	info.Details = append(info.Details, [2]string{"Bootloader at", fmt.Sprintf("0x%X", info.BootOffset)})
	if info.Chip == "ESP8266" {
		info.Note = "ESP8266 flashes at 115200 baud (its ROM can't go faster), so big files take a minute or two."
	}
	return info, nil
}

type avrTry struct {
	proto string
	baud  int
}

func avrTries(b *Board, pi PortInfo) []avrTry {
	if b == nil {
		if gb := boardByID(pi.Guess.BoardID); gb != nil && gb.Family == famAVR {
			b = gb
		}
	}
	if b != nil {
		var t []avrTry
		for _, bd := range b.Bauds {
			t = append(t, avrTry{b.Protocol, bd})
		}
		return t
	}
	return []avrTry{{"stk500v1", 115200}, {"stk500v1", 57600}, {"stk500v2", 115200}}
}

// openAVR connects to whichever AVR bootloader answers.
func openAVR(pi PortInfo, b *Board, r *reporter) (avrSession, Port, avrTry, error) {
	tries := avrTries(b, pi)
	if tries[0].proto == "avr109" {
		name := pi.Name
		if pi.Guess.Touch1200 || (b != nil && pi.Guess.Family == "") {
			r.status("Restarting the board into its bootloader…")
			before := portSet()
			delete(before, pi.Name)
			touch1200(pi.Name, r)
			n, err := waitForSerial(before, pi.Name, 8*time.Second)
			if err != nil {
				return nil, nil, avrTry{}, errors.New("the board didn't come back in bootloader mode. Double-tap its RESET button and press Flash straight away")
			}
			name = n
			r.logf("Bootloader port: %s", name)
		}
		p, err := openPortEx(name, 57600, false)
		if err != nil {
			return nil, nil, avrTry{}, wrapOpen(err)
		}
		s, err := connectAVR109(p)
		if err != nil {
			p.Close()
			return nil, nil, avrTry{}, errors.New("the Leonardo-style bootloader didn't answer. Double-tap RESET and press Flash within 8 seconds")
		}
		return s, p, avrTry{"avr109", 57600}, nil
	}
	p, err := openPortEx(pi.Name, tries[0].baud, false)
	if err != nil {
		return nil, nil, avrTry{}, wrapOpen(err)
	}
	s, t, err := avrAuto(p, tries, r)
	if err != nil {
		p.Close()
		return nil, nil, avrTry{}, err
	}
	return s, p, t, nil
}

// avrAuto tries each protocol/baud combination on an open port.
func avrAuto(p Port, tries []avrTry, r *reporter) (avrSession, avrTry, error) {
	for _, t := range tries {
		r.logf("Trying %s at %d baud", t.proto, t.baud)
		p.SetBaud(t.baud)
		switch t.proto {
		case "stk500v1":
			if s, e := connectSTK1(p); e == nil {
				return s, t, nil
			}
		case "stk500v2":
			if s, e := connectSTK2(p); e == nil {
				return s, t, nil
			}
		}
	}
	return nil, avrTry{}, errors.New("no Arduino bootloader answered. Check you picked the right board (old Nanos need “Nano (old bootloader)”) and that nothing else is using the port")
}

func describeAVR(s avrSession, t avrTry, r *reporter) (*ChipInfo, *avrMCU, error) {
	sig, err := s.Signature()
	if err != nil {
		return nil, nil, err
	}
	r.logf("AVR signature %02X %02X %02X", sig[0], sig[1], sig[2])
	mcu := mcuBySig(sig)
	name := fmt.Sprintf("AVR (signature %02X%02X%02X)", sig[0], sig[1], sig[2])
	if mcu != nil {
		name = mcu.Name
	}
	pn := map[string]string{"stk500v1": "STK500 v1 (Optiboot)", "stk500v2": "STK500 v2 (Wiring)", "avr109": "AVR109 (Caterina)"}[t.proto]
	info := &ChipInfo{Family: famAVR, Proto: t.proto, Baud: t.baud, Chip: name, Supported: mcu != nil,
		Details: [][2]string{{"Bootloader", pn}, {"Speed", fmt.Sprintf("%d baud", t.baud)}}}
	if mcu != nil {
		info.Details = append(info.Details, [2]string{"Flash", humanBytes(mcu.Flash)})
	}
	return info, mcu, nil
}

func identifyAVR(pi PortInfo, b *Board, r *reporter) (*ChipInfo, error) {
	s, p, t, err := openAVR(pi, b, r)
	if err != nil {
		return nil, err
	}
	defer p.Close()
	info, _, err := describeAVR(s, t, r)
	s.Leave()
	if err != nil {
		return nil, err
	}
	if b != nil {
		info.BoardID = b.ID
	} else {
		info.BoardID = avrBoardFor(info, pi)
	}
	return info, nil
}

func avrBoardFor(info *ChipInfo, pi PortInfo) string {
	if pi.Guess.BoardID != "" {
		return pi.Guess.BoardID
	}
	switch {
	case info.Proto == "stk500v2":
		return "mega2560-ch340"
	case info.Proto == "avr109":
		return "leonardo"
	case strings.HasPrefix(info.Chip, "ATmega168"):
		return "nano-168"
	case info.Baud == 57600:
		return "nano-old"
	}
	return "uno-ch340"
}

func identifySTM32(pi PortInfo, b *Board, r *reporter) (*ChipInfo, error) {
	p, err := openPortEx(pi.Name, 115200, true)
	if err != nil {
		return nil, wrapOpen(err)
	}
	defer p.Close()
	s, err := connectSTM32(p, r)
	if err != nil {
		return nil, err
	}
	info := &ChipInfo{Family: famSTM32, Proto: "stm32", Baud: 115200, Chip: s.chipName(), ChipID: uint32(s.pid), Supported: true,
		Details: [][2]string{{"Bootloader", fmt.Sprintf("v%d.%d", s.version>>4, s.version&0xF)}}}
	if kb := s.flashSizeKB(); kb > 0 {
		info.Details = append(info.Details, [2]string{"Flash", fmt.Sprintf("%d KB", kb)})
	}
	if b != nil {
		info.BoardID = b.ID
	}
	return info, nil
}

// ----------------------------------------------------------------- flash ---

type flashReq struct {
	Port     string
	Board    *Board
	Hint     *ChipInfo
	Data     []byte
	FileName string
	Offset   uint32
	HasOff   bool
	Baud     int
	FlashMB  int
	Compress bool
}

// runFlash routes a job to the right protocol. Returns a success message.
func runFlash(req flashReq, r *reporter) (string, error) {
	stopSession()
	pi, found := findDevice(req.Port)
	if !found {
		return "", fmt.Errorf("%s isn't connected any more — was the board unplugged?", req.Port)
	}
	pl := planFor(pi, req.Board, req.Hint)
	if pl == nil {
		r.status("Working out what kind of board this is…")
		info, err := identify(pi, nil, r)
		if err != nil {
			return "", err
		}
		events.send(map[string]any{"type": "chip", "chip": info})
		pl = planFor(pi, nil, info)
	}
	r.logf("Plan: family=%s protocol=%s", pl.Family, pl.Proto)
	switch pl.Family {
	case famESP:
		return flashESPJob(pi, req, r)
	case famAVR:
		return flashAVRJob(pi, pl, req, r)
	case famSTM32:
		return flashSTM32Job(pi, pl, req, r)
	case famRP2:
		return flashRP2Job(pi, req, r)
	}
	return "", errors.New("Flashie doesn't know how to flash this board")
}

// espImage is one region to write in an ESP flashing session.
type espImage struct {
	Name   string
	Offset uint32
	Data   []byte
}

func flashESPJob(pi PortInfo, req flashReq, r *reporter) (string, error) {
	data, offset := req.Data, req.Offset
	switch sniffFormat(req.FileName, data) {
	case fmtUF2:
		return "", errors.New("that's a UF2 file (for Raspberry Pi Pico and similar). ESP boards need a .bin file")
	case fmtHex:
		img, err := parseIntelHex(data)
		if err != nil {
			return "", err
		}
		data, offset = img.Data, img.Base
		r.logf("HEX file: writing %s at 0x%X", humanBytes(len(data)), offset)
	}
	return flashESPImages(pi, []espImage{{Name: req.FileName, Offset: offset, Data: data}}, req, r)
}

// imageChipMismatch checks an ESP image header's chip ID against the chip.
func imageChipMismatch(img espImage, chip *ChipInfo) string {
	d := img.Data
	if chip.Chip == "ESP8266" || len(d) < 16 || d[0] != 0xE9 {
		return ""
	}
	id := uint32(d[12]) | uint32(d[13])<<8
	if c := chipByID(id); c != nil && c.Name != chip.Chip && id != 0xFFFF {
		return fmt.Sprintf("%s is built for an %s, but this board is an %s — flashing it wouldn't boot, so Flashie stopped", img.Name, c.Name, chip.Chip)
	}
	return ""
}

func flashESPImages(pi PortInfo, imgs []espImage, req flashReq, r *reporter) (string, error) {
	start := time.Now()
	r.status(fmt.Sprintf("Opening %s…", pi.Name))
	p, err := openPortEx(pi.Name, 115200, false)
	if err != nil {
		return "", err
	}
	defer p.Close()
	l := NewLoader(p, pi.Guess.JTAG, pi.Guess.NativeUSB && !pi.Guess.JTAG)
	l.Log, l.Status = func(s string) { r.logf("%s", s) }, r.status
	if err := l.Connect(); err != nil {
		return "", err
	}
	info, err := l.DetectChip()
	if err != nil {
		return "", err
	}
	info.Port = pi.Name
	events.send(map[string]any{"type": "chip", "chip": info})
	r.status("Connected to an " + info.Chip)
	if !info.Supported {
		return "", errors.New(info.Chip + " isn't supported yet")
	}
	for _, im := range imgs {
		if msg := imageChipMismatch(im, info); msg != "" {
			l.HardReset()
			return "", errors.New(msg)
		}
	}
	if req.Board != nil && req.Board.Chip != info.Chip {
		r.logf("Note: you picked a %s board but the chip says %s — going with the chip.", req.Board.Chip, info.Chip)
	}
	if req.Baud > 115200 {
		r.status(fmt.Sprintf("Switching to a faster link (%d baud)…", req.Baud))
		if err := l.ChangeBaud(req.Baud); err != nil {
			return "", err
		}
	}
	flashMB := req.FlashMB
	if flashMB <= 0 {
		flashMB = 4
	}
	total, done := 0, 0
	for _, im := range imgs {
		total += len(im.Data)
	}
	for i, im := range imgs {
		if len(imgs) > 1 {
			r.logf("Part %d/%d: %s → 0x%X (%s)", i+1, len(imgs), im.Name, im.Offset, humanBytes(len(im.Data)))
		}
		base := done
		err = l.Flash(im.Data, FlashOptions{Offset: im.Offset, FlashSize: uint32(flashMB) << 20, Compress: req.Compress,
			Progress: func(d, _ int) { r.progress(base+d, total) }, Cancel: r.Cancel})
		if err != nil {
			if errors.Is(err, errCancelled) {
				l.HardReset()
			}
			if len(imgs) > 1 {
				err = fmt.Errorf("%s: %w", im.Name, err)
			}
			return "", err
		}
		done += len(im.Data)
	}
	r.status("Restarting your board into the new firmware…")
	l.HardReset()
	check := "verified"
	if info.Chip == "ESP8266" {
		check = "checksummed"
	}
	if len(imgs) == 1 {
		return fmt.Sprintf("Flashed %s to %s at 0x%X in %.1fs and %s. Your board is running the new firmware.",
			humanBytes(total), info.Chip, imgs[0].Offset, time.Since(start).Seconds(), check), nil
	}
	return fmt.Sprintf("Flashed %d parts (%s) to %s in %.1fs and %s. Your board is running the new firmware.",
		len(imgs), humanBytes(total), info.Chip, time.Since(start).Seconds(), check), nil
}

func flashAVRJob(pi PortInfo, pl *plan, req flashReq, r *reporter) (string, error) {
	var img []byte
	switch sniffFormat(req.FileName, req.Data) {
	case fmtUF2:
		return "", errors.New("that's a UF2 file (for Pico-style boards). Arduino boards need the sketch's .hex file")
	case fmtHex:
		m, err := parseIntelHex(req.Data)
		if err != nil {
			return "", err
		}
		if m.Base != 0 {
			if m.Base >= 0x7000 {
				return "", errors.New("this HEX file only contains a bootloader. Use the sketch .hex instead")
			}
			m.Data = append(make([]byte, m.Base), m.Data...)
			for i := 0; i < int(m.Base); i++ {
				m.Data[i] = 0xFF
			}
		}
		img = m.Data
	default:
		img = req.Data
		r.logf("Raw .bin: writing from address 0")
	}
	start := time.Now()
	b := pl.Board
	if b == nil && pl.Proto != "" {
		b = &Board{Family: famAVR, Protocol: pl.Proto, Bauds: pl.Bauds}
		if len(b.Bauds) == 0 {
			b.Bauds = []int{115200, 57600}
		}
	}
	r.status("Resetting the board into its bootloader…")
	s, p, t, err := openAVR(pi, b, r)
	if err != nil {
		return "", err
	}
	defer p.Close()
	info, mcu, err := describeAVR(s, t, r)
	if err != nil {
		return "", err
	}
	info.Port = pi.Name
	events.send(map[string]any{"type": "chip", "chip": info})
	if mcu == nil {
		return "", fmt.Errorf("Flashie doesn't know this chip (%s) yet", info.Chip)
	}
	r.status("Connected to an " + mcu.Name)
	if pl.Board != nil && pl.Board.Chip != "" && pl.Board.Chip != mcu.Name && !(pl.Board.Chip == "ATmega328P" && mcu.Name == "ATmega328") {
		return "", fmt.Errorf("you picked %s (%s) but the chip is an %s. Pick the right board and try again", pl.Board.Name, pl.Board.Chip, mcu.Name)
	}
	max := mcu.Flash - bootloaderReserve[t.proto]
	if pl.Board != nil && pl.Board.MaxSketch > 0 {
		max = pl.Board.MaxSketch
	}
	if err := flashAVR(s, mcu, img, max, r); err != nil {
		return "", err
	}
	return fmt.Sprintf("Uploaded %s to the %s in %.1fs and verified. Your sketch is running.", humanBytes(len(img)), mcu.Name, time.Since(start).Seconds()), nil
}

func flashSTM32Job(pi PortInfo, pl *plan, req flashReq, r *reporter) (string, error) {
	var img []byte
	base := uint32(stmFlashBase)
	switch sniffFormat(req.FileName, req.Data) {
	case fmtUF2:
		return "", errors.New("that's a UF2 file — STM32 serial flashing needs a .bin or .hex")
	case fmtHex:
		m, err := parseIntelHex(req.Data)
		if err != nil {
			return "", err
		}
		img, base = m.Data, m.Base
		if base < stmFlashBase && base+uint32(len(img)) <= 0x00100000 {
			base += stmFlashBase // HEX linked at the 0x0 alias
		}
	default:
		img = req.Data
		if req.HasOff {
			base = req.Offset
			if base < stmFlashBase {
				base += stmFlashBase
			}
		}
	}
	baud := req.Baud
	if baud <= 0 || baud > 115200 {
		baud = 115200
	}
	start := time.Now()
	r.status(fmt.Sprintf("Opening %s (8E1, %d baud)…", pi.Name, baud))
	p, err := openPortEx(pi.Name, baud, true)
	if err != nil {
		return "", err
	}
	defer p.Close()
	r.status("Talking to the STM32 bootloader…")
	s, err := connectSTM32(p, r)
	if err != nil {
		return "", err
	}
	events.send(map[string]any{"type": "chip", "chip": &ChipInfo{Family: famSTM32, Proto: "stm32", Chip: s.chipName(), Supported: true, Port: pi.Name}})
	r.status("Connected to an " + s.chipName())
	if err := flashSTM32(s, base, img, r); err != nil {
		return "", err
	}
	r.status("Starting your firmware…")
	if err := s.Go(stmFlashBase); err != nil {
		r.logf("Go command: %v", err)
	}
	return fmt.Sprintf("Flashed %s to the %s at 0x%08X in %.1fs and verified. It's running now — set BOOT0 back to 0 so it boots your firmware next time.",
		humanBytes(len(img)), s.chipName(), base, time.Since(start).Seconds()), nil
}

func flashRP2Job(pi PortInfo, req flashReq, r *reporter) (string, error) {
	start := time.Now()
	var d *uf2Drive
	if pi.Kind == "drive" {
		d = pi.drive
	} else {
		r.status("Rebooting the board into BOOTSEL mode…")
		before := driveSet()
		touch1200(pi.Name, r)
		got, err := waitForUF2Drive(before, 12*time.Second)
		if err != nil {
			return "", errors.New("the board didn't switch to BOOTSEL mode (MicroPython and some firmware ignore the request). Unplug it, hold BOOTSEL, plug it back in, then pick the new drive")
		}
		d = got
		r.logf("BOOTSEL drive appeared at %s", d.Letter)
	}
	data, err := prepareUF2(req.FileName, req.Data, *d, r)
	if err != nil {
		return "", err
	}
	events.send(map[string]any{"type": "chip", "chip": &ChipInfo{Family: famRP2, Proto: "uf2", Chip: d.chipName(), Supported: true, Port: pi.Name}})
	if err := copyUF2(*d, data, r); err != nil {
		return "", err
	}
	return fmt.Sprintf("Copied %s to the %s in %.1fs. The board has restarted with the new firmware.",
		humanBytes(len(data)), d.chipName(), time.Since(start).Seconds()), nil
}
