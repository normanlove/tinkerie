package serialport

import "testing"

// A trimmed `ioreg -r -c IOUSBHostDevice -l -w0` from a Mac with a CH340 board
// and a Pico, in the format ioreg prints.
const ioregSample = `+-o USB Serial@01100000  <class IOUSBHostDevice, id 0x100000a1b, registered, matched, active, busy 0 (13 ms), retain 30>
  | {
  |   "idProduct" = 29987
  |   "USB Product Name" = "USB Serial"
  |   "USB Vendor Name" = "QinHeng Electronics"
  |   "idVendor" = 6790
  | }
  |
  +-o IOUSBHostInterface@0  <class IOUSBHostInterface, id 0x100000a1f, registered, matched, active, busy 0 (2 ms), retain 7>
    | {
    |   "bInterfaceClass" = 255
    | }
    |
    +-o AppleUSBCHCOM  <class AppleUSBCHCOM, id 0x100000a25, registered, matched, active, busy 0 (1 ms), retain 8>
      +-o IOSerialBSDClient  <class IOSerialBSDClient, id 0x100000a2c, registered, matched, active, busy 0 (0 ms), retain 6>
          {
            "IOCalloutDevice" = "/dev/cu.usbserial-110"
            "IODialinDevice" = "/dev/tty.usbserial-110"
          }

+-o Board in FS mode@02100000  <class IOUSBHostDevice, id 0x100000b01, registered, matched, active, busy 0 (9 ms), retain 28>
  | {
  |   "idProduct" = 10
  |   "kUSBSerialNumberString" = "E6614C311B7A2C2B"
  |   "USB Product Name" = "Board in FS mode"
  |   "idVendor" = 11914
  | }
  |
  +-o IOUSBHostInterface@0  <class IOUSBHostInterface, id 0x100000b05, registered, matched, active, busy 0 (2 ms), retain 7>
    +-o AppleUSBACMData  <class AppleUSBACMData, id 0x100000b0a, registered, matched, active, busy 0 (1 ms), retain 8>
      +-o IOSerialBSDClient  <class IOSerialBSDClient, id 0x100000b0f, registered, matched, active, busy 0 (0 ms), retain 6>
          {
            "IOCalloutDevice" = "/dev/cu.usbmodem2101"
          }
`

func TestParseIoreg(t *testing.T) {
	m := parseIoreg(ioregSample)
	ch := m["/dev/cu.usbserial-110"]
	if ch.VID != 0x1A86 || ch.PID != 0x7523 || ch.Manufacturer != "QinHeng Electronics" || !ch.USB {
		t.Errorf("CH340: %+v", ch)
	}
	pico := m["/dev/cu.usbmodem2101"]
	if pico.VID != 0x2E8A || pico.PID != 0x000A || pico.Serial != "E6614C311B7A2C2B" || pico.Product != "Board in FS mode" {
		t.Errorf("Pico: %+v", pico)
	}
	if len(m) != 2 {
		t.Errorf("found %d ports, want 2", len(m))
	}
}
