//go:build js && wasm

package serialport

// web_js.go: serial ports in the web version, through Chrome and Edge's Web
// Serial (tkdev.js). A port appears once the person has picked it with the
// page's Connect button; ports are named "web:N".
// Created by nrmluvdev for Tinkerie.

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"tinkerie/shared/webjs"
)

const usbPrefix = "web:"

const (
	tiocmDTR = 0x002
	tiocmRTS = 0x004
)

func open(name string, cfg Config) (*Port, error) {
	return nil, errors.New("in the web version, pick the port with the Connect button at the top of the page")
}
func configure(fd int, cfg Config) error           { return ErrUnsupported }
func setModemBit(fd int, bit int32, on bool) error { return ErrUnsupported }
func flush(fd int) error                           { return ErrUnsupported }
func list() []Info                                 { return usbList() }

// knownChips names common USB serial chips by vendor ID.
var knownChips = map[uint16]string{0x1A86: "WCH CH340", 0x10C4: "Silicon Labs CP210x", 0x0403: "FTDI", 0x067B: "Prolific PL2303",
	0x2341: "Arduino", 0x2A03: "Arduino", 0x2E8A: "Raspberry Pi", 0x303A: "Espressif", 0x0483: "STMicroelectronics", 0x239A: "Adafruit", 0x1B4F: "SparkFun", 0x2886: "Seeed"}

func usbList() []Info {
	v, err := webjs.CallTimeout(3*time.Second, "serial", "list")
	if err != nil {
		return nil
	}
	var out []Info
	for i := 0; i < v.Length(); i++ {
		p := v.Index(i)
		id, vid, pid := p.Get("id").Int(), uint16(p.Get("vid").Int()), uint16(p.Get("pid").Int())
		name := "Serial port " + strconv.Itoa(id)
		in := Info{Path: usbPrefix + strconv.Itoa(id), Name: name, VID: vid, PID: pid, USB: vid != 0}
		if vid != 0 {
			in.Description = fmt.Sprintf("USB serial %04X:%04X", vid, pid)
			in.Manufacturer = knownChips[vid]
			in.Product = fmt.Sprintf("(%04X:%04X)", vid, pid)
		} else {
			in.Description = "serial port"
		}
		out = append(out, in)
	}
	return out
}

type usbPort struct {
	h        int
	dtr, rts bool
}

func opts(cfg Config) map[string]any {
	par := map[byte]string{'N': "none", 'E': "even", 'O': "odd"}[cfg.Parity]
	stop := 1
	if cfg.StopBits > 1 {
		stop = 2
	}
	return map[string]any{"baud": cfg.Baud, "dataBits": cfg.DataBits, "parity": par, "stopBits": stop, "dtr": cfg.DTR, "rts": cfg.RTS}
}

func openUSB(name string, cfg Config) (*Port, error) {
	id, err := strconv.Atoi(strings.TrimPrefix(name, usbPrefix))
	if err != nil {
		return nil, errors.New("not a serial port name: " + name)
	}
	if cfg.Parity == 'M' || cfg.Parity == 'S' {
		return nil, errors.New("browsers can't use mark or space parity: use the desktop app for this device")
	}
	if cfg.DataBits < 7 {
		return nil, errors.New("browsers can only use 7 or 8 data bits: use the desktop app for this device")
	}
	h, err := webjs.CallTimeout(10*time.Second, "serial", "open", id, webjs.Object(opts(cfg)))
	if err != nil {
		return nil, err
	}
	return &Port{u: &usbPort{h: h.Int(), dtr: cfg.DTR, rts: cfg.RTS}, cfg: cfg, fd: -1}, nil
}

func (u *usbPort) read(b []byte, d time.Duration) (int, error) {
	ms := int(d / time.Millisecond)
	if d > 0 && ms == 0 {
		ms = 1
	}
	v, err := webjs.Call("serial", "read", u.h, len(b), ms)
	if err != nil {
		return 0, err
	}
	n := v.Get("length").Int()
	if n == 0 {
		if d == 0 {
			return u.read(b, 0)
		}
		return 0, nil
	}
	got := webjs.Bytes(v)
	return copy(b, got), nil
}

func (u *usbPort) write(b []byte) (int, error) {
	v, err := webjs.CallTimeout(10*time.Second, "serial", "write", u.h, webjs.JSBytes(b))
	if err != nil {
		return 0, err
	}
	return v.Int(), nil
}

func (u *usbPort) configure(cfg Config) error {
	if cfg.Parity == 'M' || cfg.Parity == 'S' {
		return errors.New("browsers can't use mark or space parity")
	}
	o := opts(cfg)
	o["dtr"], o["rts"] = u.dtr, u.rts
	_, err := webjs.CallTimeout(10*time.Second, "serial", "reconfigure", u.h, webjs.Object(o))
	return err
}

func (u *usbPort) lines() error {
	_, err := webjs.CallTimeout(3*time.Second, "serial", "signals", u.h, u.dtr, u.rts)
	return err
}

func (u *usbPort) purge() error {
	_, err := webjs.Call("serial", "purge", u.h)
	return err
}

func (u *usbPort) close() error {
	_, err := webjs.CallTimeout(5*time.Second, "serial", "close", u.h)
	return err
}
