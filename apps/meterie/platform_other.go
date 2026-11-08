//go:build !windows && !linux && !darwin

package meterie

// platform_other.go: stubs for systems other than Windows, Linux and macOS.

import (
	"errors"
	"fmt"
)

var errNotWindows = errors.New("hardware access is only implemented in the Windows build")

func platformOpenSerial(path string, cfg SerialCfg) (Port, error) { return nil, errNotWindows }
func platformOpenCH9325(path string, cfg SerialCfg) (Port, error) { return nil, errNotWindows }
func platformOpenCP2110(path string, cfg SerialCfg) (Port, error) { return nil, errNotWindows }
func platformOpenHIDRaw(path string) (Port, error)                { return nil, errNotWindows }
func platformOpenBU86X(path string) (Port, error)                 { return nil, errNotWindows }
func platformOpenBLE(addr uint64, svc, chr string) (Port, error)  { return nil, errNotWindows }
func platformListPorts() []PortInfo                               { return previewPorts }
func platformBLEScan(on bool) error                               { return nil }
func platformBLEResults() []PortInfo                              { return previewBLE }
func platformListCameras() []string                               { return []string{"Integrated Webcam"} }
func platformOpenCamera(i int) (FrameSource, error)               { return nil, errNotWindows }
func platformPlaySound(wav []byte, loop bool)                     {}
func platformStopSound()                                          {}
func platformReveal(path string)                                  {}

var clip string

func platformClipboardSet(s string) { clip = s }
func platformClipboardGet() string  { return clip }
func platformSaveDialog(title, def, filter, ext string) (string, bool) {
	return "", false
}
func platformOpenDialog(title, filter, pattern string) (string, bool) { return "", false }

func platformOffscreen(w, h int) (*Canvas, TextEngine, func()) {
	return NewCanvas(w, h), &previewText{}, func() {}
}

func runWeb() { fmt.Println("Meterie's window isn't available on this system yet.") }
