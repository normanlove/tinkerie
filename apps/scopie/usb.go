// USB helpers shared by the drivers: device lists, Intel-HEX firmware and
// uploading firmware into Cypress FX2 based scopes.
package scopie

import (
	"bufio"
	"bytes"
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf16"
)

//go:embed firmware/*.hex
var firmwareFS embed.FS

// usbInfo describes one USB device (or one interface of a composite device).
type usbInfo struct {
	Path     string // device path to open, "" if no usable driver is attached
	VID, PID uint16
	BCD      uint16
	MI       int    // interface number of a composite device, -1 otherwise
	Service  string // Windows driver service (WinUSB, usbccgp, NI-VISA ...)
	Desc     string
	Serial   string
	WinUSB   bool
	Key      string // stable identity for de-duplicating
}

type usbDevice interface {
	Control(rt, req uint8, val, idx uint16, data []byte) (int, error)
	BulkWrite(ep uint8, data []byte, timeoutMs int) (int, error)
	BulkRead(ep uint8, buf []byte, timeoutMs int) (int, error)
	// Stream keeps `count` bulk reads of `size` bytes queued on ep and hands
	// each completed buffer to fn (which must copy what it keeps).
	Stream(ep uint8, size, count int, stop <-chan struct{}, fn func([]byte) bool) error
	ResetPipe(ep uint8)
	// Endpoints returns the bulk IN and OUT endpoints and the interface class.
	Endpoints() (in, out []uint8, class, subclass uint8)
	Close()
}

// driverError means the device is present but the WinUSB driver isn't attached.
type driverError struct{ msg string }

func (e *driverError) Error() string { return e.msg }

const (
	usbVendorOut = 0x40
	usbVendorIn  = 0xC0
	usbStdIn     = 0x80
	usbClassOut  = 0x21
	usbClassIn   = 0xA1
)

var accessHelp = "• This scope needs the WinUSB driver: click 'Install USB driver…' in the Connect " +
	"window (or run Zadig) and choose WinUSB for it.\n" +
	"• Close any other oscilloscope program that may be using it."

// ---------------------------------------------------------------- device list cache

var usbCacheMu sync.Mutex
var usbCache []usbInfo
var usbCacheT time.Time

func listUSB() []usbInfo {
	usbCacheMu.Lock()
	defer usbCacheMu.Unlock()
	if usbCache != nil && time.Since(usbCacheT) < 1500*time.Millisecond {
		return usbCache
	}
	usbCache = listUSBDevices()
	if usbCache == nil {
		usbCache = []usbInfo{}
	}
	usbCacheT = time.Now()
	return usbCache
}

func invalidateUSBCache() {
	usbCacheMu.Lock()
	usbCache = nil
	usbCacheMu.Unlock()
}

func findUSB(match func(u usbInfo) bool) *usbInfo {
	invalidateUSBCache()
	for _, u := range listUSB() {
		if match(u) {
			u := u
			return &u
		}
	}
	return nil
}

func openUSB(u *usbInfo) (usbDevice, error) {
	if u.Path == "" || !u.WinUSB {
		svc := u.Service
		if svc == "" {
			svc = "none"
		}
		return nil, devErr("The scope was found but Windows has the wrong driver for it (%s).\n\n%s", svc, accessHelp)
	}
	d, err := openUSBDevice(u.Path)
	if err != nil {
		var de *driverError
		if errors.As(err, &de) {
			return nil, devErr("The scope was found but could not be opened (%v).\n\n%s", err, accessHelp)
		}
		return nil, devErr("The scope was found but could not be opened (%v).\n\n%s", err, accessHelp)
	}
	return d, nil
}

func waitFor[T any](seconds float64, step time.Duration, fn func() *T) *T {
	deadline := time.Now().Add(time.Duration(seconds * 1e9))
	for time.Now().Before(deadline) {
		if r := fn(); r != nil {
			return r
		}
		time.Sleep(step)
	}
	return nil
}

// usbStrings reads the manufacturer and product strings.
func usbStrings(d usbDevice) (manuf, product, serial string) {
	dd := make([]byte, 18)
	n, err := d.Control(usbStdIn, 6, 0x0100, 0, dd)
	if err != nil || n < 17 {
		return "", "", ""
	}
	return usbString(d, dd[14]), usbString(d, dd[15]), usbString(d, dd[16])
}

func usbString(d usbDevice, idx uint8) string {
	if idx == 0 {
		return ""
	}
	buf := make([]byte, 255)
	n, err := d.Control(usbStdIn, 6, 0x0300|uint16(idx), 0x0409, buf)
	if err != nil || n < 2 {
		return ""
	}
	u := make([]uint16, 0, n/2)
	for i := 2; i+1 < n; i += 2 {
		u = append(u, uint16(buf[i])|uint16(buf[i+1])<<8)
	}
	return strings.TrimSpace(string(utf16.Decode(u)))
}

// ---------------------------------------------------------------- Intel HEX + FX2

type hexRecord struct {
	Addr int
	Data []byte
}

func parseIHex(b []byte) ([]hexRecord, error) {
	var recs []hexRecord
	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		ln := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(ln, ":") {
			continue
		}
		if len(ln) < 11 || (len(ln)-1)%2 != 0 {
			return nil, errors.New("bad hex record")
		}
		raw := make([]byte, (len(ln)-1)/2)
		for i := range raw {
			v, err := strconv.ParseUint(ln[1+2*i:3+2*i], 16, 8)
			if err != nil {
				return nil, errors.New("bad hex record")
			}
			raw[i] = byte(v)
		}
		cnt := int(raw[0])
		if len(raw) < 5+cnt {
			return nil, errors.New("short hex record")
		}
		addr := int(raw[1])<<8 | int(raw[2])
		switch raw[3] {
		case 0:
			recs = append(recs, hexRecord{addr, append([]byte(nil), raw[4:4+cnt]...)})
		case 1:
			return recs, nil
		}
	}
	if len(recs) == 0 {
		return nil, errors.New("no data records")
	}
	return recs, nil
}

// mergeRecords joins adjacent records into chunks of at most max bytes.
func mergeRecords(recs []hexRecord, max int) []hexRecord {
	var out []hexRecord
	for _, r := range recs {
		for off := 0; off < len(r.Data); {
			n := minInt(max, len(r.Data)-off)
			piece := r.Data[off : off+n]
			addr := r.Addr + off
			if l := len(out); l > 0 && out[l-1].Addr+len(out[l-1].Data) == addr && len(out[l-1].Data)+n <= max {
				out[l-1].Data = append(out[l-1].Data, piece...)
			} else {
				out = append(out, hexRecord{addr, append([]byte(nil), piece...)})
			}
			off += n
		}
	}
	return out
}

// fx2Upload loads firmware into FX2 RAM with the built-in 0xA0 vendor request.
func fx2Upload(d usbDevice, recs []hexRecord) error {
	const cpucs = 0xE600
	if _, err := d.Control(usbVendorOut, 0xA0, cpucs, 0, []byte{1}); err != nil { // hold CPU in reset
		return devErr("Uploading firmware to the scope failed: %v", err)
	}
	for _, r := range mergeRecords(recs, 1024) {
		if _, err := d.Control(usbVendorOut, 0xA0, uint16(r.Addr), 0, r.Data); err != nil {
			return devErr("Uploading firmware to the scope failed: %v", err)
		}
	}
	_, _ = d.Control(usbVendorOut, 0xA0, cpucs, 0, []byte{0}) // run; the device re-enumerates
	return nil
}

// Firmware we may not bundle (no published source); users import these themselves.
var userFirmware = map[string]string{
	"isds205b-firmware.hex": "Instrustar ISDS-205B",
	"mdso-firmware.hex":     "YiXingDianZi MDSO",
}

func userFirmwareDir() string {
	d := filepath.Join(appDataDir(), "firmware")
	_ = os.MkdirAll(d, 0o755)
	return d
}

// firmwareBytes finds bundled firmware, or firmware the user imported.
func firmwareBytes(name string) ([]byte, error) {
	if b, err := firmwareFS.ReadFile("firmware/" + name); err == nil {
		return b, nil
	}
	for _, d := range []string{userFirmwareDir(), legacyFirmwareDir()} {
		if d == "" {
			continue
		}
		if b, err := os.ReadFile(filepath.Join(d, name)); err == nil {
			return b, nil
		}
	}
	return nil, devErr("Firmware file '%s' is missing. Use 'Import firmware…' in the Connect window to add it.", name)
}

func haveFirmware(name string) bool {
	_, err := firmwareBytes(name)
	return err == nil
}

// importFirmware stores a user-supplied firmware or driver file. It returns
// the scopes it was saved for.
func importFirmware(name string, data []byte) ([]string, error) {
	low := strings.ToLower(filepath.Base(name))
	if strings.HasSuffix(low, ".hex") {
		model, ok := userFirmware[low]
		if !ok {
			var names []string
			for k := range userFirmware {
				names = append(names, k)
			}
			return nil, devErr("Unknown firmware file. Expected one of: %s", strings.Join(names, ", "))
		}
		if _, err := parseIHex(data); err != nil {
			return nil, devErr("That file is not valid Intel-HEX firmware (%v).", err)
		}
		if err := os.WriteFile(filepath.Join(userFirmwareDir(), low), data, 0o644); err != nil {
			return nil, err
		}
		return []string{model}, nil
	}
	return extractHantekFirmware(name, data)
}

func hex4(v uint16) string { return fmt.Sprintf("%04X", v) }
