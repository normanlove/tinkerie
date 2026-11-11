//go:build linux

package serialport

// usbserial.go: USB serial adapters driven directly over USB, for Android,
// where apps can't open /dev/ttyUSB* or /dev/ttyACM*. Covers what makers
// plug in: boards with native USB serial (CDC-ACM: Arduino Uno/Leonardo/R4,
// Raspberry Pi Pico, ESP32-S2/S3/C3/C6, STM32, CH9102/CH343), and the common
// adapter chips (WCH CH340/CH341, Silicon Labs CP210x, FTDI FT232/FT2232/FT-X,
// Prolific PL2303). The control requests follow the chips' Linux drivers.
// Created by nrmluvdev for Tinkerie.

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"tinkerie/shared/usb"
)

const usbPrefix = "usb:"

type chip int

const (
	chipNone chip = iota
	chipCDC
	chipCH34x
	chipCP210x
	chipFTDI
	chipPL2303
)

var chipNames = map[chip]string{chipCDC: "USB serial", chipCH34x: "CH340", chipCP210x: "CP210x", chipFTDI: "FTDI", chipPL2303: "PL2303"}

// usbChip works out which driver a device needs and which interfaces it uses.
func usbChip(in usb.Info) (c chip, data, comm int) {
	// native USB serial (CDC-ACM): a communications interface and a data one
	comm = -1
	for _, it := range in.Interfaces {
		if it.Class == 2 && it.Sub == 2 && comm < 0 {
			comm = it.Number
		}
	}
	for _, it := range in.Interfaces {
		if it.Class == 10 && len(it.BulkIn) > 0 && len(it.BulkOut) > 0 {
			if comm < 0 {
				comm = it.Number
			}
			return chipCDC, it.Number, comm
		}
	}
	first := func() int {
		for _, it := range in.Interfaces {
			if len(it.BulkIn) > 0 && len(it.BulkOut) > 0 {
				return it.Number
			}
		}
		return 0
	}
	switch in.VID {
	case 0x1A86:
		switch in.PID {
		case 0x7523, 0x5523, 0x7522, 0xE523:
			return chipCH34x, first(), -1
		}
	case 0x10C4:
		switch in.PID {
		case 0xEA60, 0xEA61, 0xEA63, 0xEA70, 0xEA71, 0xEA80:
			return chipCP210x, first(), -1
		}
	case 0x0403:
		switch in.PID {
		case 0x6001, 0x6010, 0x6011, 0x6014, 0x6015:
			return chipFTDI, first(), -1
		}
	case 0x067B:
		switch in.PID {
		case 0x2303:
			return chipPL2303, first(), -1
		}
	}
	return chipNone, 0, -1
}

// usbList lists USB serial devices as ports named "usb:<device path>".
func usbList() []Info {
	var out []Info
	for _, d := range usb.List() {
		c, _, _ := usbChip(d)
		if c == chipNone {
			continue
		}
		name := d.Product
		if name == "" {
			name = chipNames[c]
		}
		desc := chipNames[c]
		if c == chipCDC {
			desc = "native USB serial"
		}
		out = append(out, Info{Path: usbPrefix + d.Path, Name: name, USB: true, VID: d.VID, PID: d.PID,
			Manufacturer: d.Manufacturer, Product: d.Product, Serial: d.Serial, Description: name + " (" + desc + ")"})
	}
	return out
}

type usbPort struct {
	d         *usb.Device
	c         chip
	iface     int
	comm      int
	in, out   uint8
	pkt       int // FTDI: bytes per USB packet, each starting with 2 status bytes
	ftdiH     bool
	ftdiIdx   uint16
	ftdiMulti bool
	dtr, rts  bool

	mu      sync.Mutex
	buf     []byte
	ready   chan struct{}
	closed  chan struct{}
	readErr error
	once    sync.Once
}

func openUSB(name string, cfg Config) (*Port, error) {
	path := strings.TrimPrefix(name, usbPrefix)
	var info usb.Info
	found := false
	for _, d := range usb.List() {
		if d.Path == path {
			info, found = d, true
		}
	}
	if !found {
		return nil, errors.New("that USB serial device isn't plugged in any more")
	}
	c, data, comm := usbChip(info)
	if c == chipNone {
		return nil, errors.New("this USB device isn't a serial adapter Tinkerie knows")
	}
	d, err := usb.Open(path, data)
	if err != nil {
		return nil, err
	}
	u := &usbPort{d: d, c: c, iface: data, comm: comm, ready: make(chan struct{}, 1), closed: make(chan struct{})}
	if c == chipCDC && comm >= 0 && comm != data {
		if err := d.ClaimExtra(comm); err != nil {
			d.Close()
			return nil, err
		}
	}
	for _, it := range info.Interfaces {
		if it.Number == data {
			u.in, u.out = it.BulkIn[0], it.BulkOut[0]
		}
	}
	if c == chipFTDI {
		u.ftdiH = info.BCD == 0x0700 || info.BCD == 0x0800 || info.BCD == 0x0900
		u.pkt = 64
		if u.ftdiH {
			u.pkt = 512
		}
		if info.PID == 0x6010 || info.PID == 0x6011 { // FT2232 / FT4232: ports are numbered from 1
			u.ftdiIdx = uint16(data + 1)
			u.ftdiMulti = true
		}
	}
	u.dtr, u.rts = cfg.DTR, cfg.RTS
	if err := u.init(cfg); err != nil {
		d.Close()
		return nil, fmt.Errorf("couldn't set up the %s adapter: %v", chipNames[c], err)
	}
	go u.reader()
	return &Port{u: u, cfg: cfg, fd: -1}, nil
}

func (u *usbPort) ctrlOut(rt, req uint8, val, idx uint16, data []byte) error {
	_, err := u.d.Control(rt, req, val, idx, data, 2000)
	return err
}

func (u *usbPort) ctrlIn(rt, req uint8, val, idx uint16, n int) ([]byte, error) {
	b := make([]byte, n)
	k, err := u.d.Control(rt, req, val, idx, b, 2000)
	return b[:max(k, 0)], err
}

func (u *usbPort) init(cfg Config) error {
	switch u.c {
	case chipCH34x:
		u.ctrlIn(0xC0, 0x5F, 0, 0, 2)
		if err := u.ctrlOut(0x40, 0xA1, 0, 0, nil); err != nil {
			return err
		}
		u.ch34xBaud(9600)
		u.ctrlIn(0xC0, 0x95, 0x2518, 0, 2)
		u.ctrlOut(0x40, 0x9A, 0x2518, 0xC3, nil)
		u.ctrlIn(0xC0, 0x95, 0x0706, 0, 2)
		u.ctrlOut(0x40, 0xA1, 0x501F, 0xD90A, nil)
	case chipCP210x:
		if err := u.ctrlOut(0x41, 0x00, 1, uint16(u.iface), nil); err != nil { // enable the UART
			return err
		}
	case chipFTDI:
		u.ctrlOut(0x40, 0, 0, u.ftdiIdx, nil) // reset
		u.ctrlOut(0x40, 2, 0, u.ftdiIdx, nil) // no flow control
		u.ctrlOut(0x40, 9, 1, u.ftdiIdx, nil) // 1 ms latency: answers come back quickly
	case chipPL2303:
		vin := func(v uint16) { u.ctrlIn(0xC0, 0x01, v, 0, 1) }
		vout := func(v, i uint16) { u.ctrlOut(0x40, 0x01, v, i, nil) }
		vin(0x8484)
		vout(0x0404, 0)
		vin(0x8484)
		vin(0x8383)
		vin(0x8484)
		vout(0x0404, 1)
		vin(0x8484)
		vin(0x8383)
		vout(0, 1)
		vout(1, 0)
		vout(2, 0x44)
	}
	if err := u.configure(cfg); err != nil {
		return err
	}
	return u.lines()
}

func (u *usbPort) ch34xBaud(baud int) error {
	var factor, divisor int64
	if baud == 921600 {
		divisor, factor = 7, 0xF300
	} else {
		factor, divisor = 1532620800/int64(baud), 3
		for factor > 0xFFF0 && divisor > 0 {
			factor >>= 3
			divisor--
		}
		if factor > 0xFFF0 {
			return fmt.Errorf("the CH340 can't run at %d baud", baud)
		}
		factor = 0x10000 - factor
	}
	divisor |= 0x80
	if err := u.ctrlOut(0x40, 0x9A, 0x1312, uint16((factor&0xFF00)|divisor), nil); err != nil {
		return err
	}
	return u.ctrlOut(0x40, 0x9A, 0x0F2C, uint16(factor&0xFF), nil)
}

func ftdiDivisor(baud int, h bool) (val, idx uint16) {
	frac := [8]uint32{0, 3, 2, 4, 1, 5, 6, 7}
	var div uint32
	if h && baud >= 1200 {
		d3 := uint32((8*120000000/10 + baud/2) / baud) // 12 MHz clock
		div = d3>>3 | frac[d3&7]<<14 | 0x20000
		if div == 1|0x20000 {
			div = 0 | 0x20000
		} else if div == 0x4001|0x20000 {
			div = 1 | 0x20000
		}
	} else {
		d3 := uint32((24000000 + baud/2) / baud) // 3 MHz clock, in eighths
		div = d3>>3 | frac[d3&7]<<14
		if div == 1 {
			div = 0
		} else if div == 0x4001 {
			div = 1
		}
	}
	return uint16(div), uint16(div >> 16)
}

func (u *usbPort) configure(cfg Config) error {
	parity := map[byte]int{'N': 0, 'O': 1, 'E': 2, 'M': 3, 'S': 4}[cfg.Parity]
	stop := 0
	if cfg.StopBits >= 1.5 {
		stop = 2
	}
	switch u.c {
	case chipCDC, chipPL2303:
		lc := make([]byte, 7)
		binary.LittleEndian.PutUint32(lc, uint32(cfg.Baud))
		lc[4], lc[5], lc[6] = byte(stop), byte(parity), byte(cfg.DataBits)
		idx := uint16(0)
		if u.c == chipCDC {
			idx = uint16(u.comm)
		}
		return u.ctrlOut(0x21, 0x20, 0, idx, lc)
	case chipCH34x:
		if err := u.ch34xBaud(cfg.Baud); err != nil {
			return err
		}
		lcr := 0xC0 | (cfg.DataBits-5)&3
		if parity > 0 {
			lcr |= 0x08 | (parity-1)<<4
		}
		if stop == 2 {
			lcr |= 0x04
		}
		return u.ctrlOut(0x40, 0x9A, 0x2518, uint16(lcr), nil)
	case chipCP210x:
		b := make([]byte, 4)
		binary.LittleEndian.PutUint32(b, uint32(cfg.Baud))
		if err := u.ctrlOut(0x41, 0x1E, 0, uint16(u.iface), b); err != nil {
			return err
		}
		return u.ctrlOut(0x41, 0x03, uint16(cfg.DataBits)<<8|uint16(parity)<<4|uint16(stop), uint16(u.iface), nil)
	case chipFTDI:
		val, hi := ftdiDivisor(cfg.Baud, u.ftdiH)
		idx := hi // FT232R and FT-X: the divisor's top bits go in the index
		if u.ftdiMulti || u.ftdiH {
			idx = hi<<8 | u.ftdiIdx
		}
		if err := u.ctrlOut(0x40, 3, val, idx, nil); err != nil {
			return err
		}
		return u.ctrlOut(0x40, 4, uint16(cfg.DataBits)|uint16(parity)<<8|uint16(stop)<<11, u.ftdiIdx, nil)
	}
	return nil
}

// lines sets DTR and RTS.
func (u *usbPort) lines() error {
	b := func(on bool, v uint16) uint16 {
		if on {
			return v
		}
		return 0
	}
	switch u.c {
	case chipCDC, chipPL2303:
		idx := uint16(0)
		if u.c == chipCDC {
			idx = uint16(u.comm)
		}
		return u.ctrlOut(0x21, 0x22, b(u.dtr, 1)|b(u.rts, 2), idx, nil)
	case chipCH34x:
		return u.ctrlOut(0x40, 0xA4, ^(b(u.dtr, 0x20) | b(u.rts, 0x40)), 0, nil)
	case chipCP210x:
		return u.ctrlOut(0x41, 0x07, 0x0300|b(u.dtr, 1)|b(u.rts, 2), uint16(u.iface), nil)
	case chipFTDI:
		if err := u.ctrlOut(0x40, 1, 0x0100|b(u.dtr, 1), u.ftdiIdx, nil); err != nil {
			return err
		}
		return u.ctrlOut(0x40, 1, 0x0200|b(u.rts, 2), u.ftdiIdx, nil)
	}
	return nil
}

// reader keeps reading from the device into a buffer, so Read and
// ReadTimeout can wait on it.
func (u *usbPort) reader() {
	buf := make([]byte, 16384)
	for {
		select {
		case <-u.closed:
			return
		default:
		}
		n, err := u.d.BulkRead(u.in, buf, 200)
		if err != nil {
			u.mu.Lock()
			u.readErr = errors.New("the USB serial device was unplugged")
			u.mu.Unlock()
			u.wake()
			return
		}
		if n == 0 {
			continue
		}
		data := buf[:n]
		if u.c == chipFTDI { // drop the 2 status bytes at the start of every packet
			var clean []byte
			for i := 0; i < len(data); i += u.pkt {
				end := min(i+u.pkt, len(data))
				if end-i > 2 {
					clean = append(clean, data[i+2:end]...)
				}
			}
			data = clean
			if len(data) == 0 {
				continue
			}
		}
		u.mu.Lock()
		if len(u.buf) < 4<<20 { // drop data nobody reads rather than grow forever
			u.buf = append(u.buf, data...)
		}
		u.mu.Unlock()
		u.wake()
	}
}

func (u *usbPort) wake() {
	select {
	case u.ready <- struct{}{}:
	default:
	}
}

func (u *usbPort) read(b []byte, d time.Duration) (int, error) {
	var deadline <-chan time.Time
	if d > 0 {
		t := time.NewTimer(d)
		defer t.Stop()
		deadline = t.C
	}
	for {
		u.mu.Lock()
		if len(u.buf) > 0 {
			n := copy(b, u.buf)
			u.buf = u.buf[n:]
			u.mu.Unlock()
			return n, nil
		}
		err := u.readErr
		u.mu.Unlock()
		if err != nil {
			return 0, err
		}
		select {
		case <-u.ready:
		case <-u.closed:
			return 0, errors.New("port closed")
		case <-deadline:
			return 0, nil
		}
	}
}

func (u *usbPort) write(b []byte) (int, error) {
	done := 0
	for done < len(b) {
		n, err := u.d.BulkWrite(u.out, b[done:min(done+16384, len(b))], 5000)
		if err != nil {
			return done, err
		}
		if n == 0 {
			return done, errors.New("the USB serial device stopped taking data")
		}
		done += n
	}
	return done, nil
}

func (u *usbPort) purge() error {
	u.mu.Lock()
	u.buf = nil
	u.mu.Unlock()
	switch u.c {
	case chipCP210x:
		return u.ctrlOut(0x41, 0x12, 0x0F, uint16(u.iface), nil)
	case chipFTDI:
		u.ctrlOut(0x40, 0, 1, u.ftdiIdx, nil)
		return u.ctrlOut(0x40, 0, 2, u.ftdiIdx, nil)
	}
	return nil
}

func (u *usbPort) close() error {
	u.once.Do(func() {
		close(u.closed)
		time.Sleep(250 * time.Millisecond) // let the reader's last transfer time out
		u.d.Close()
	})
	return nil
}
