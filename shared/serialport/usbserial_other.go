//go:build !linux

package serialport

import (
	"errors"
	"time"
)

// USB serial over USB is only needed on Android (which counts as Linux).
type usbPort struct{ dtr, rts bool }

const usbPrefix = "usb:"

func openUSB(name string, cfg Config) (*Port, error) {
	return nil, errors.New("USB serial over USB is only used on Android")
}
func usbList() []Info                                          { return nil }
func (u *usbPort) read(b []byte, d time.Duration) (int, error) { return 0, ErrUnsupported }
func (u *usbPort) write(b []byte) (int, error)                 { return 0, ErrUnsupported }
func (u *usbPort) configure(c Config) error                    { return ErrUnsupported }
func (u *usbPort) lines() error                                { return ErrUnsupported }
func (u *usbPort) purge() error                                { return ErrUnsupported }
func (u *usbPort) close() error                                { return nil }
