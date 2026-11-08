//go:build !windows && !linux && !darwin

package flashie

import "errors"

var errNoDisks = errors.New("writing drives isn't supported on this system yet")

func listDisksOS() ([]Disk, error) { return nil, errNoDisks }
func openDiskForWrite(d Disk) (rawDisk, error) {
	if t, ok := testDiskFile(d); ok {
		return t, nil
	}
	return nil, errNoDisks
}
func finishDisk(d Disk, eject bool)                  {}
func canWriteDirect(d Disk) bool                     { _, ok := testDiskFile(d); return ok }
func elevate(base, token string) (chan error, error) { return nil, errNoDisks }
func pickImageFile() (string, error)                 { return "", errNoPicker }
