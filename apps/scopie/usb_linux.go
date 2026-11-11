//go:build linux

// USB scopes on Linux and Raspberry Pi OS, through usbfs (shared/usb).
package scopie

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"tinkerie/shared/droid"

	"tinkerie/shared/usb"
)

// listUSBDevices lists every USB device once. Path is the device node plus the
// interface Scopie should claim ("/dev/bus/usb/001/005#0"): a USB-TMC or
// vendor interface if the device has several.
func listUSBDevices() []usbInfo {
	var out []usbInfo
	for _, d := range usb.List() {
		pick, mi := 0, -1
		if len(d.Interfaces) > 1 {
			mi = d.Interfaces[0].Number
			for _, it := range d.Interfaces {
				if it.Class == 0xFE && it.Sub == 3 { // USB-TMC
					mi = it.Number
					break
				}
				if it.Class == 0xFF && mi == d.Interfaces[0].Number {
					mi = it.Number
				}
			}
			pick = mi
		} else if len(d.Interfaces) == 1 {
			pick = d.Interfaces[0].Number
		}
		desc := strings.TrimSpace(d.Manufacturer + " " + d.Product)
		out = append(out, usbInfo{
			Path: d.Path + "#" + strconv.Itoa(pick), VID: d.VID, PID: d.PID, BCD: d.BCD, MI: mi,
			Service: "usbfs", Desc: desc, Serial: d.Serial,
			WinUSB: d.Accessible, // "ready to use without installing anything"
			Key:    d.SysPath,
		})
	}
	return out
}

type linuxUSB struct {
	d *usb.Device
}

func openUSBDevice(path string) (usbDevice, error) {
	node, ifs, _ := strings.Cut(path, "#")
	iface, _ := strconv.Atoi(ifs)
	d, err := usb.Open(node, iface)
	if err != nil {
		return nil, err
	}
	return &linuxUSB{d: d}, nil
}

func (u *linuxUSB) Control(rt, req uint8, val, idx uint16, data []byte) (int, error) {
	return u.d.Control(rt, req, val, idx, data, 2000)
}
func (u *linuxUSB) BulkWrite(ep uint8, data []byte, timeoutMs int) (int, error) {
	return u.d.BulkWrite(ep, data, timeoutMs)
}
func (u *linuxUSB) BulkRead(ep uint8, buf []byte, timeoutMs int) (int, error) {
	return u.d.BulkRead(ep, buf, timeoutMs)
}
func (u *linuxUSB) Stream(ep uint8, size, count int, stop <-chan struct{}, fn func([]byte) bool) error {
	return u.d.Stream(ep, size, count, stop, fn)
}
func (u *linuxUSB) ResetPipe(ep uint8) { u.d.ResetPipe(ep) }
func (u *linuxUSB) Endpoints() ([]uint8, []uint8, uint8, uint8) {
	it := u.d.Interface()
	return it.BulkIn, it.BulkOut, it.Class, it.Sub
}
func (u *linuxUSB) Close() { u.d.Close() }

// usbDriverHint says what to do when a device can't be used yet.
func usbDriverHint(u usbInfo) string {
	return fmt.Sprintf("USB %04X:%04X: Linux needs permission first. Run install-usb.sh in Tinkerie's folder once, then replug the scope.", u.VID, u.PID)
}

func shellExecute(verb, file, args, dir string) error {
	return fmt.Errorf("not needed on Linux")
}

func openURL(u string) {
	if droid.Active() {
		droid.Browse(u)
		return
	}
	_ = exec.Command("xdg-open", u).Start()
}

func openFolder(dir string) {
	if droid.Active() {
		droid.ShowFolder(dir)
		return
	}
	openURL(dir)
}
