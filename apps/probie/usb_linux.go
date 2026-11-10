package probie

// USB logic analysers (Cypress FX2 / fx2lafw) on Linux and Raspberry Pi OS,
// through usbfs (shared/usb).

import "tinkerie/shared/usb"

func listUSBDevices() []usbInfo {
	var out []usbInfo
	// Every device: Probie picks analysers by ID, and opening one without
	// permission explains how to fix it (install-usb.sh).
	for _, d := range usb.List() {
		out = append(out, usbInfo{Path: d.Path, VID: d.VID, PID: d.PID})
	}
	return out
}

type linuxUSB struct{ d *usb.Device }

func openUSBDevice(path string) (usbDevice, error) {
	d, err := usb.Open(path, 0)
	if err != nil {
		return nil, err
	}
	return &linuxUSB{d: d}, nil
}

func (u *linuxUSB) Control(rt, req uint8, val, idx uint16, data []byte) (int, error) {
	return u.d.Control(rt, req, val, idx, data, 2000)
}
func (u *linuxUSB) Stream(ep uint8, size, count int, stop <-chan struct{}, fn func([]byte) bool) error {
	return u.d.Stream(ep, size, count, stop, fn)
}
func (u *linuxUSB) ResetPipe(ep uint8) { u.d.ResetPipe(ep) }
func (u *linuxUSB) Close()             { u.d.Close() }
