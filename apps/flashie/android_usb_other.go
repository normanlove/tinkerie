//go:build !linux

package flashie

import "errors"

const usbDrivePrefix = "usbdrive:"

// Drives over USB are only used on Android (which counts as Linux).
func copyUF2USB(d uf2Drive, data []byte, r *reporter) error {
	return errors.New("this drive can only be written on Android")
}

func androidUF2Drives() []uf2Drive { return nil }
