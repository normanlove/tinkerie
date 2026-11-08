//go:build !linux

package usb

// Device is unavailable off Linux; Open always fails.
type Device struct{ Info Info }

// List finds nothing off Linux.
func List() []Info { return nil }

// Open fails off Linux.
func Open(path string, iface int) (*Device, error) { return nil, ErrUnsupported }

func (d *Device) Interface() Interface { return Interface{} }
func (d *Device) Control(rt, req uint8, val, idx uint16, data []byte, timeoutMs int) (int, error) {
	return 0, ErrUnsupported
}
func (d *Device) BulkRead(ep uint8, buf []byte, timeoutMs int) (int, error) { return 0, ErrUnsupported }
func (d *Device) BulkWrite(ep uint8, data []byte, timeoutMs int) (int, error) {
	return 0, ErrUnsupported
}
func (d *Device) ResetPipe(ep uint8) {}
func (d *Device) Close()             {}
func (d *Device) Stream(ep uint8, size, count int, stop <-chan struct{}, fn func([]byte) bool) error {
	return ErrUnsupported
}
