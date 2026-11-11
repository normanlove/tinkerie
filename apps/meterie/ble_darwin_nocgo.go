//go:build darwin && !cgo

package meterie

import "errors"

// Built without cgo (not on a Mac), the Mac app has no Bluetooth.
var errBLEOnly = errors.New("Bluetooth meters need the Mac build of Tinkerie")

func platformOpenBLE(addr uint64, svc, chr string) (Port, error) { return nil, errBLEOnly }
func platformBLEScan(on bool) error {
	if on {
		return errBLEOnly
	}
	return nil
}
func platformBLEResults() []PortInfo { return previewBLE }
