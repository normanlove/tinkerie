package probie

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
)

// Cheap "Saleae Logic" clones are a Cypress FX2 with no useful firmware of
// their own. We load sigrok's open-source fx2lafw firmware into RAM, after
// which the device streams one byte (8 channels) per sample on bulk EP2.

type usbInfo struct {
	Path string
	VID  uint16
	PID  uint16
}

type usbDevice interface {
	Control(rt, req uint8, val, idx uint16, data []byte) (int, error)
	Stream(ep uint8, size, count int, stop <-chan struct{}, fn func([]byte) bool) error
	ResetPipe(ep uint8)
	Close()
}

type driverError struct{ msg string }

func (e *driverError) Error() string { return e.msg }

type fx2Profile struct {
	VID, PID uint16
	Name     string
	FW       string
}

var fx2Profiles = []fx2Profile{
	{0x0925, 0x3881, "Saleae-compatible 24 MHz 8CH", "fx2lafw-saleae-logic.fw"},
	{0x1d50, 0x608c, "sigrok FX2 LA (8ch)", "fx2lafw-sigrok-fx2-8ch.fw"},
	{0x04b4, 0x8613, "Cypress FX2 (blank EEPROM)", "fx2lafw-cypress-fx2.fw"},
	{0x08a9, 0x0014, "CWAV USBee AX", "fx2lafw-cwav-usbeeax.fw"},
	{0x08a9, 0x0009, "CWAV USBee SX", "fx2lafw-cwav-usbeesx.fw"},
}

func fx2ProfileFor(vid, pid uint16) *fx2Profile {
	for i := range fx2Profiles {
		if fx2Profiles[i].VID == vid && fx2Profiles[i].PID == pid {
			return &fx2Profiles[i]
		}
	}
	return nil
}

const (
	fx2CmdGetFWVersion = 0xb0
	fx2CmdStart        = 0xb1
	fx2CmdGetRevID     = 0xb2
	fx2DataEP          = 0x82
	usbVendorOut       = 0x40
	usbVendorIn        = 0xC0
	usbStdIn           = 0x80
)

// fx2SampleRates lists every rate the firmware can hit exactly.
var fx2SampleRates = []float64{20e3, 25e3, 50e3, 100e3, 200e3, 250e3, 500e3, 1e6, 2e6, 3e6, 4e6, 6e6, 8e6, 12e6, 16e6, 24e6}

func usbStrings(d usbDevice) (manuf, product string) {
	dd := make([]byte, 18)
	n, err := d.Control(usbStdIn, 6, 0x0100, 0, dd)
	if err != nil || n < 16 {
		return "", ""
	}
	return usbString(d, dd[14]), usbString(d, dd[15])
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
	return string(utf16.Decode(u))
}

// firmwareDirs are searched for fx2lafw *.fw files, including where
// PulseView installs them.
func firmwareDirs(dataDir string) []string {
	var dirs []string
	if exe, err := os.Executable(); err == nil {
		d := filepath.Dir(exe)
		dirs = append(dirs, d, filepath.Join(d, "firmware"))
	}
	dirs = append(dirs, filepath.Join(dataDir, "firmware"))
	for _, env := range []string{"ProgramFiles", "ProgramFiles(x86)", "ProgramW6432"} {
		if b := os.Getenv(env); b != "" {
			dirs = append(dirs,
				filepath.Join(b, "sigrok", "PulseView", "share", "sigrok-firmware"),
				filepath.Join(b, "sigrok", "sigrok-cli", "share", "sigrok-firmware"))
		}
	}
	if b := os.Getenv("LOCALAPPDATA"); b != "" {
		dirs = append(dirs, filepath.Join(b, "sigrok-firmware"))
	}
	dirs = append(dirs, "/usr/share/sigrok-firmware", "/usr/local/share/sigrok-firmware")
	return dirs
}

func findFirmware(dataDir, name string) (string, []byte) {
	base := strings.TrimSuffix(name, ".fw")
	for _, d := range firmwareDirs(dataDir) {
		for _, n := range []string{name, base + ".ihx", base + ".hex"} {
			p := filepath.Join(d, n)
			if b, err := os.ReadFile(p); err == nil && len(b) > 0 {
				img, err := firmwareImage(b)
				if err == nil {
					return p, img
				}
				log.Printf("firmware %s: %v", p, err)
			}
		}
	}
	return "", nil
}

// firmwareImage accepts a raw .fw binary or an Intel-HEX .ihx file.
func firmwareImage(b []byte) ([]byte, error) {
	if len(b) == 0 || b[0] != ':' {
		return b, nil
	}
	img := make([]byte, 0, 16384)
	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		ln := strings.TrimSpace(sc.Text())
		if len(ln) < 11 || ln[0] != ':' {
			continue
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
		addr := int(raw[1])<<8 | int(raw[2])
		typ := raw[3]
		if len(raw) < 5+cnt {
			return nil, errors.New("short hex record")
		}
		switch typ {
		case 0:
			if addr+cnt > len(img) {
				img = append(img, make([]byte, addr+cnt-len(img))...)
			}
			copy(img[addr:], raw[4:4+cnt])
		case 1:
			return img, nil
		}
	}
	return img, nil
}

func fx2UploadFirmware(d usbDevice, img []byte) error {
	if _, err := d.Control(usbVendorOut, 0xA0, 0xE600, 0, []byte{1}); err != nil {
		return fmt.Errorf("halt CPU: %w", err)
	}
	for off := 0; off < len(img); off += 4096 {
		end := off + 4096
		if end > len(img) {
			end = len(img)
		}
		chunk := append([]byte(nil), img[off:end]...)
		if _, err := d.Control(usbVendorOut, 0xA0, uint16(off), 0, chunk); err != nil {
			return fmt.Errorf("write firmware @%04x: %w", off, err)
		}
	}
	_, _ = d.Control(usbVendorOut, 0xA0, 0xE600, 0, []byte{0}) // device disconnects here
	return nil
}

type fx2Probe struct {
	info     usbInfo
	profile  *fx2Profile
	status   string // ready, firmware, nofirmware, driver, error
	detail   string
	fwLoaded bool
}

func probeFX2(info usbInfo, dataDir string) fx2Probe {
	p := fx2Probe{info: info, profile: fx2ProfileFor(info.VID, info.PID)}
	d, err := openUSBDevice(info.Path)
	if err != nil {
		var de *driverError
		if errors.As(err, &de) {
			p.status = "driver"
			p.detail = "Needs the WinUSB driver — run Zadig once (see Help)."
		} else {
			p.status = "error"
			p.detail = err.Error()
		}
		return p
	}
	defer d.Close()
	m, prod := usbStrings(d)
	if strings.EqualFold(m, "sigrok") && strings.EqualFold(prod, "fx2lafw") {
		p.fwLoaded = true
		p.status = "ready"
		fw := make([]byte, 2)
		if n, err := d.Control(usbVendorIn, fx2CmdGetFWVersion, 0, 0, fw); err == nil && n == 2 {
			p.detail = fmt.Sprintf("fx2lafw firmware v%d.%d running", fw[0], fw[1])
		} else {
			p.detail = "fx2lafw firmware running"
		}
		return p
	}
	if path, _ := findFirmware(dataDir, p.profile.FW); path != "" {
		p.status = "ready"
		p.detail = "Firmware loads automatically on first capture"
	} else {
		p.status = "nofirmware"
		p.detail = "Needs " + p.profile.FW + " — drop it on the Firmware box (see Help)."
	}
	return p
}

// fx2Open returns an opened device that is running fx2lafw, uploading the
// firmware and waiting for re-enumeration when needed.
func fx2Open(path, dataDir string, status func(string)) (usbDevice, error) {
	var info *usbInfo
	for _, u := range listUSBDevices() {
		if u.Path == path {
			u := u
			info = &u
		}
	}
	if info == nil {
		return nil, errors.New("analyser not found — is it plugged in?")
	}
	prof := fx2ProfileFor(info.VID, info.PID)
	if prof == nil {
		return nil, errors.New("not a supported FX2 analyser")
	}
	d, err := openUSBDevice(path)
	if err != nil {
		return nil, err
	}
	m, prod := usbStrings(d)
	if strings.EqualFold(m, "sigrok") && strings.EqualFold(prod, "fx2lafw") {
		return d, nil
	}
	fwPath, img := findFirmware(dataDir, prof.FW)
	if img == nil {
		d.Close()
		return nil, fmt.Errorf("firmware %s not found — drop it on the Firmware box in the Logic sidebar", prof.FW)
	}
	status("Uploading firmware…")
	log.Printf("uploading %s (%d bytes)", fwPath, len(img))
	err = fx2UploadFirmware(d, img)
	d.Close()
	if err != nil {
		return nil, err
	}
	status("Waiting for analyser to restart…")
	deadline := time.Now().Add(6 * time.Second)
	time.Sleep(700 * time.Millisecond)
	for time.Now().Before(deadline) {
		for _, u := range listUSBDevices() {
			if u.VID != info.VID || u.PID != info.PID {
				continue
			}
			d2, err := openUSBDevice(u.Path)
			if err != nil {
				continue
			}
			m, prod := usbStrings(d2)
			if strings.EqualFold(m, "sigrok") && strings.EqualFold(prod, "fx2lafw") {
				return d2, nil
			}
			d2.Close()
		}
		time.Sleep(300 * time.Millisecond)
	}
	return nil, errors.New("analyser didn't come back after the firmware upload — unplug it, plug it back in and try again")
}

func fx2StartAcq(d usbDevice, rate float64) error {
	r := int(rate)
	if r <= 0 {
		return errors.New("bad sample rate")
	}
	var flags byte
	delay := 0
	if 48000000%r == 0 {
		flags = 1 << 6 // 48 MHz clock
		delay = 48000000/r - 1
		if delay > 6*256 {
			delay = 0
		}
	}
	if delay == 0 && 30000000%r == 0 {
		flags = 0 // 30 MHz clock
		delay = 30000000/r - 1
	}
	if delay <= 0 && r != 24000000 && r != 48000000 {
		return fmt.Errorf("sample rate %d not supported", r)
	}
	_, err := d.Control(usbVendorOut, fx2CmdStart, 0, 0, []byte{flags, byte(delay >> 8), byte(delay)})
	return err
}
