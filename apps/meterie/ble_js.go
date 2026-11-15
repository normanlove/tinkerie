//go:build js && wasm

package meterie

// ble_js.go: Bluetooth meters aren't in the web version yet (the desktop app
// has them).

import "errors"

var errBLEWeb = errors.New("Bluetooth meters need the desktop app: the web version can't use them yet")

func platformBLEScan(on bool) error                              { return errBLEWeb }
func platformBLEResults() []PortInfo                             { return nil }
func platformOpenBLE(addr uint64, svc, chr string) (Port, error) { return nil, errBLEWeb }
