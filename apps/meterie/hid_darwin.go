//go:build darwin && !cgo

package meterie

import "errors"

// Filled in by hid_darwin_iokit.go when built with cgo.
var errHIDMac = errors.New("this meter's USB cable needs the Mac build of Tinkerie")

func openHIDDev(path string) (hidHandle, error) { return nil, errHIDMac }
func listHID() []PortInfo                       { return nil }
