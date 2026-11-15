//go:build js && wasm

package meterie

// hid_js.go: USB-HID meter cables in the web version, through Chrome and
// Edge's WebHID (tkdev.js). A cable appears once it's been picked with the
// page's Connect button. Reports keep Windows' layout: report ID first.

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"tinkerie/shared/webjs"
)

type webHID struct {
	h              int
	outLen, featLn int
}

func openHIDDev(path string) (hidHandle, error) {
	id, err := strconv.Atoi(strings.TrimPrefix(path, "webhid:"))
	if err != nil {
		return nil, errors.New("not a USB cable name: " + path)
	}
	h, err := webjs.CallTimeout(10*time.Second, "hid", "open", id)
	if err != nil {
		return nil, err
	}
	return &webHID{h: h.Int()}, nil
}

func (d *webHID) readReport(timeoutMs uint32) ([]byte, error) {
	v, err := webjs.Call("hid", "read", d.h, int(timeoutMs))
	if err != nil {
		return nil, fmt.Errorf("USB read failed (cable unplugged?): %v", err)
	}
	if v.IsNull() {
		return nil, nil
	}
	return webjs.Bytes(v), nil
}

func pad(b []byte, n int) []byte {
	buf := make([]byte, max(n, len(b)))
	copy(buf, b)
	return buf
}

func (d *webHID) writeReport(b []byte) error {
	if _, err := webjs.CallTimeout(time.Second, "hid", "write", d.h, webjs.JSBytes(pad(b, d.outLen))); err != nil {
		return fmt.Errorf("USB write: %v", err)
	}
	return nil
}

func (d *webHID) setFeature(b []byte) error {
	if _, err := webjs.CallTimeout(2*time.Second, "hid", "feature", d.h, webjs.JSBytes(pad(b, d.featLn))); err != nil {
		return fmt.Errorf("set feature report: %v", err)
	}
	return nil
}

func (d *webHID) defaultLens(out, feat int) {
	if d.outLen == 0 {
		d.outLen = out
	}
	if d.featLn == 0 {
		d.featLn = feat
	}
}
func (d *webHID) forceFeatLen(n int) { d.featLn = n }
func (d *webHID) close()             { webjs.CallTimeout(3*time.Second, "hid", "close", d.h) }

// listHID finds known meter cables among the USB-HID devices this page may use.
func listHID() []PortInfo {
	v, err := webjs.CallTimeout(3*time.Second, "hid", "list")
	if err != nil {
		return nil
	}
	var out []PortInfo
	for i := 0; i < v.Length(); i++ {
		d := v.Index(i)
		vid, pid, name := uint16(d.Get("vid").Int()), uint16(d.Get("pid").Int()), d.Get("name").String()
		var kind Transport = -1
		for _, k := range []Transport{TrCH9325, TrCP2110, TrVictor, TrBU86X} {
			if u := lookupUSB(vid, pid, k); u != nil && u.Kind == k {
				kind = k
			}
		}
		if kind < 0 {
			continue
		}
		pi := PortInfo{Kind: kind, Path: "webhid:" + strconv.Itoa(d.Get("id").Int()), Name: name, VID: vid, PID: pid}
		classifyPort(&pi)
		if name != "" {
			pi.Name = fmt.Sprintf("%s (%04X:%04X)", name, vid, pid)
		}
		out = append(out, pi)
	}
	return out
}
