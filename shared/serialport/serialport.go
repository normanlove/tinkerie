// Package serialport talks to serial ports on Linux (including Raspberry Pi)
// and macOS, in pure Go with no cgo. Windows apps use their own Win32 code.
//
// Ports are opened non-blocking and handed to Go's runtime poller, so reads can
// time out (ReadTimeout) and Close wakes up a blocked Read straight away.
// Created by nrmluvdev for Tinkerie.
package serialport

import (
	"errors"
	"os"
	"sort"
	"strings"
	"time"
)

// Config describes how to set up a port. Zero values mean 8N1.
type Config struct {
	Baud     int
	DataBits int     // 5–8, default 8
	Parity   byte    // 'N' (default), 'E', 'O', 'M' (mark), 'S' (space)
	StopBits float64 // 1 (default), 1.5 or 2; 1.5 is treated as 2
	DTR, RTS bool    // control line levels right after opening
}

// Info describes a port found by List.
type Info struct {
	Path         string // /dev/ttyUSB0, /dev/cu.usbserial-110
	Name         string // ttyUSB0, cu.usbserial-110
	USB          bool
	VID, PID     uint16
	Manufacturer string
	Product      string
	Serial       string
	Description  string // e.g. "Raspberry Pi GPIO serial" for non-USB ports
}

// Port is an open serial port.
type Port struct {
	f   *os.File
	fd  int
	cfg Config
}

// ErrUnsupported is returned on systems this package doesn't cover.
var ErrUnsupported = errors.New("serial ports aren't supported on this system")

// Open opens and configures a port. name may be a full path or a bare name
// like ttyUSB0.
func Open(name string, cfg Config) (*Port, error) {
	if !strings.HasPrefix(name, "/") {
		name = "/dev/" + name
	}
	if cfg.DataBits == 0 {
		cfg.DataBits = 8
	}
	if cfg.Parity == 0 {
		cfg.Parity = 'N'
	}
	if cfg.StopBits == 0 {
		cfg.StopBits = 1
	}
	return open(name, cfg)
}

// Read waits for data and returns what has arrived. It returns an error once
// the port is closed or unplugged.
func (p *Port) Read(b []byte) (int, error) {
	if err := p.f.SetReadDeadline(time.Time{}); err != nil {
		return 0, err
	}
	for {
		n, err := p.f.Read(b)
		if n > 0 || err != nil {
			return n, err
		}
	}
}

// ReadTimeout reads whatever is available, waiting at most d for the first
// byte. It returns (0, nil) on timeout.
func (p *Port) ReadTimeout(b []byte, d time.Duration) (int, error) {
	if err := p.f.SetReadDeadline(time.Now().Add(d)); err != nil {
		return 0, err
	}
	n, err := p.f.Read(b)
	if errors.Is(err, os.ErrDeadlineExceeded) {
		return n, nil
	}
	if n == 0 && err == nil {
		return 0, nil
	}
	return n, err
}

// Write sends b, waiting until the port has accepted all of it.
func (p *Port) Write(b []byte) (int, error) { return p.f.Write(b) }

// SetBaud changes the speed, keeping the other settings.
func (p *Port) SetBaud(baud int) error {
	c := p.cfg
	c.Baud = baud
	if err := configure(p.fd, c); err != nil {
		return err
	}
	p.cfg = c
	return nil
}

// SetDTR sets the DTR control line.
func (p *Port) SetDTR(on bool) error { return setModemBit(p.fd, tiocmDTR, on) }

// SetRTS sets the RTS control line.
func (p *Port) SetRTS(on bool) error { return setModemBit(p.fd, tiocmRTS, on) }

// Purge throws away anything waiting to be read or sent.
func (p *Port) Purge() error { return flush(p.fd) }

// Close closes the port, waking any Read in progress.
func (p *Port) Close() error { return p.f.Close() }

// List returns the serial ports on this computer, USB ports first.
func List() []Info {
	out := list()
	sort.Slice(out, func(i, j int) bool {
		if out[i].USB != out[j].USB {
			return out[i].USB
		}
		return out[i].Path < out[j].Path
	})
	return out
}
