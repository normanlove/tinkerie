// Package usb gives Tinkerie apps raw USB access: on Linux (including Raspberry
// Pi OS) and Android through the kernel's usbfs in pure Go, and on macOS
// through IOKit (darwin.go, the one place that needs cgo). Windows apps use
// WinUSB directly.
//
// Devices are listed from sysfs and opened as /dev/bus/usb/BBB/DDD. A normal
// user needs permission for that: install-usb.sh adds a udev rule for the
// scopes and analysers Tinkerie supports. Created by nrmluvdev for Tinkerie.
package usb

import "errors"

// Info describes a device found by List.
type Info struct {
	Path         string // /dev/bus/usb/001/005
	SysPath      string // /sys/bus/usb/devices/1-1.2, stable while plugged in
	VID, PID     uint16
	BCD          uint16
	Manufacturer string
	Product      string
	Serial       string
	Interfaces   []Interface
	Accessible   bool // this user may open it
}

// Interface is one USB interface of a device (first alternate setting).
type Interface struct {
	Number          int
	Class, Sub      uint8
	BulkIn, BulkOut []uint8 // bulk endpoint addresses
}

// ErrUnsupported is returned on systems this package doesn't cover.
var ErrUnsupported = errors.New("USB devices aren't supported on this system yet")

// ErrNoPermission explains how to allow access on Linux.
var ErrNoPermission = errors.New("no permission to use this USB device: run install-usb.sh (in Tinkerie's folder) once, then unplug and replug it")
