package serialport

import (
	"strconv"
	"strings"
)

// parseIoreg reads `ioreg -l` output: USB devices carry idVendor/idProduct, and
// the serial ports below them carry IOCalloutDevice (/dev/cu.*).
func parseIoreg(s string) map[string]Info {
	type node struct {
		depth int
		info  Info
	}
	var stack []node
	out := map[string]Info{}
	for _, line := range strings.Split(s, "\n") {
		if i := strings.Index(line, "+-o "); i >= 0 {
			for len(stack) > 0 && stack[len(stack)-1].depth >= i {
				stack = stack[:len(stack)-1]
			}
			stack = append(stack, node{depth: i})
			continue
		}
		if len(stack) == 0 {
			continue
		}
		k, v, ok := strings.Cut(strings.TrimLeft(line, " |"), " = ")
		if !ok {
			continue
		}
		k, v = strings.Trim(strings.TrimSpace(k), `"`), strings.Trim(strings.TrimSpace(v), `"`)
		cur := &stack[len(stack)-1].info
		switch k {
		case "idVendor":
			n, _ := strconv.ParseUint(v, 10, 16)
			cur.VID, cur.USB = uint16(n), true
		case "idProduct":
			n, _ := strconv.ParseUint(v, 10, 16)
			cur.PID = uint16(n)
		case "USB Vendor Name", "kUSBVendorString":
			cur.Manufacturer = v
		case "USB Product Name", "kUSBProductString":
			cur.Product = v
		case "USB Serial Number", "kUSBSerialNumberString":
			cur.Serial = v
		case "IOCalloutDevice":
			for j := len(stack) - 1; j >= 0; j-- {
				if stack[j].info.USB {
					out[v] = stack[j].info
					break
				}
			}
		}
	}
	return out
}
