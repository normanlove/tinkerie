package scopie

// libusb-win32 devices (Zadig's other common driver) are opened through
// shared/libusb0; usbInfo.Path names them "libusb0|VID|PID|n".

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"tinkerie/shared/libusb0"
)

func loadLibusb0() error { return libusb0.Available() }

func libusb0Path(vid, pid uint16, n int) string {
	return fmt.Sprintf("libusb0|%04X|%04X|%d", vid, pid, n)
}

func openLibusb0(path string) (usbDevice, error) {
	if err := libusb0.Available(); err != nil {
		return nil, &driverError{"the libusb-win32 driver is attached but libusb0.dll couldn't be loaded (" + err.Error() + ")"}
	}
	f := strings.Split(path, "|")
	if len(f) != 4 {
		return nil, errors.New("bad libusb-win32 device name")
	}
	vid, _ := strconv.ParseUint(f[1], 16, 16)
	pid, _ := strconv.ParseUint(f[2], 16, 16)
	n, _ := strconv.Atoi(f[3])
	d, err := libusb0.Open(uint16(vid), uint16(pid), n)
	if err != nil {
		return nil, err
	}
	return d, nil
}
