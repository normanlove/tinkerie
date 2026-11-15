//go:build js && wasm

package usb

// web_js.go: USB devices in the web version, through Chrome and Edge's WebUSB
// (tkdev.js). A device appears once the person has picked it with the page's
// Connect button; devices are named "web:N".
// Created by nrmluvdev for Tinkerie.

import (
	"errors"
	"strconv"
	"strings"
	"sync"
	"syscall/js"
	"time"

	"tinkerie/shared/webjs"
)

// Device is an open USB device with one claimed interface.
type Device struct {
	h     int
	iface int
	Info  Info
	mu    sync.Mutex
}

// List returns the USB devices this page may use.
func List() []Info {
	v, err := webjs.CallTimeout(3*time.Second, "usb", "list")
	if err != nil {
		return nil
	}
	var out []Info
	for i := 0; i < v.Length(); i++ {
		d := v.Index(i)
		in := Info{Path: "web:" + strconv.Itoa(d.Get("id").Int()), SysPath: "web:" + strconv.Itoa(d.Get("id").Int()),
			VID: uint16(d.Get("vid").Int()), PID: uint16(d.Get("pid").Int()), BCD: uint16(d.Get("bcd").Int()),
			Manufacturer: d.Get("manufacturer").String(), Product: d.Get("product").String(), Serial: d.Get("serial").String(), Accessible: true}
		its := d.Get("interfaces")
		for j := 0; j < its.Length(); j++ {
			it := its.Index(j)
			in.Interfaces = append(in.Interfaces, Interface{Number: it.Get("number").Int(), Class: uint8(it.Get("cls").Int()), Sub: uint8(it.Get("sub").Int()),
				BulkIn: eps(it.Get("bulkIn")), BulkOut: eps(it.Get("bulkOut"))})
		}
		out = append(out, in)
	}
	return out
}

func eps(a js.Value) []uint8 {
	var out []uint8
	for i := 0; i < a.Length(); i++ {
		out = append(out, uint8(a.Index(i).Int()))
	}
	return out
}

// Open opens a device and claims an interface.
func Open(path string, iface int) (*Device, error) {
	id, err := strconv.Atoi(strings.TrimPrefix(path, "web:"))
	if err != nil {
		return nil, errors.New("not a USB device name: " + path)
	}
	h, err := webjs.CallTimeout(15*time.Second, "usb", "open", id, iface)
	if err != nil {
		return nil, err
	}
	d := &Device{h: h.Int(), iface: iface}
	for _, in := range List() {
		if in.Path == path {
			d.Info = in
		}
	}
	return d, nil
}

// ClaimExtra claims another interface of an open device too.
func (d *Device) ClaimExtra(iface int) error {
	_, err := webjs.CallTimeout(5*time.Second, "usb", "claim", d.h, iface)
	return err
}

// Interface returns the claimed interface's details.
func (d *Device) Interface() Interface {
	for _, it := range d.Info.Interfaces {
		if it.Number == d.iface {
			return it
		}
	}
	return Interface{Number: d.iface}
}

func ms(timeoutMs int) time.Duration {
	if timeoutMs <= 0 {
		return 0
	}
	return time.Duration(timeoutMs) * time.Millisecond
}

// Control does a control transfer. The direction comes from rt's top bit.
func (d *Device) Control(rt, req uint8, val, idx uint16, data []byte, timeoutMs int) (int, error) {
	if rt&0x80 != 0 {
		v, err := webjs.CallTimeout(ms(timeoutMs), "usb", "controlIn", d.h, int(rt), int(req), int(val), int(idx), len(data))
		if err != nil {
			return 0, err
		}
		return copy(data, webjs.Bytes(v)), nil
	}
	v, err := webjs.CallTimeout(ms(timeoutMs), "usb", "controlOut", d.h, int(rt), int(req), int(val), int(idx), webjs.JSBytes(data))
	if err != nil {
		return 0, err
	}
	return v.Int(), nil
}

// BulkRead reads from an IN endpoint, waiting at most timeoutMs.
func (d *Device) BulkRead(ep uint8, buf []byte, timeoutMs int) (int, error) {
	v, err := webjs.Call("usb", "bulkIn", d.h, int(ep|0x80), len(buf), timeoutMs)
	if err != nil {
		return 0, err
	}
	if v.IsNull() {
		return 0, errors.New("USB read timed out")
	}
	return copy(buf, webjs.Bytes(v)), nil
}

// BulkWrite writes to an OUT endpoint.
func (d *Device) BulkWrite(ep uint8, data []byte, timeoutMs int) (int, error) {
	v, err := webjs.CallTimeout(ms(timeoutMs), "usb", "bulkOut", d.h, int(ep&0x7f), webjs.JSBytes(data))
	if err != nil {
		return 0, err
	}
	return v.Int(), nil
}

// ResetPipe clears a stalled endpoint.
func (d *Device) ResetPipe(ep uint8) { webjs.CallTimeout(2*time.Second, "usb", "clearHalt", d.h, int(ep)) }

// Stream keeps count reads of size bytes queued on an IN endpoint and calls
// fn with each one in order until fn returns false or stop is closed.
func (d *Device) Stream(ep uint8, size, count int, stop <-chan struct{}, fn func([]byte) bool) error {
	ep |= 0x80
	if _, err := webjs.Call("usb", "streamStart", d.h, int(ep), size, count); err != nil {
		return err
	}
	defer webjs.Call("usb", "streamStop", d.h, int(ep))
	for {
		select {
		case <-stop:
			return nil
		default:
		}
		v, err := webjs.Call("usb", "streamNext", d.h, int(ep), 200)
		if err != nil {
			return err
		}
		if v.IsNull() {
			continue
		}
		if !fn(webjs.Bytes(v)) {
			return nil
		}
	}
}

// Close releases the device.
func (d *Device) Close() { webjs.CallTimeout(5*time.Second, "usb", "close", d.h) }
