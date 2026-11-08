//go:build windows

// Flashie — find UF2 bootloader drives (RPI-RP2, RP2350, …). Made by nrmluvdev.

package flashie

import (
	"os"
	"syscall"
	"unsafe"
)

var (
	procGetLogicalDrives = kernel32.NewProc("GetLogicalDrives")
	procGetDriveType     = kernel32.NewProc("GetDriveTypeW")
	procGetVolumeInfo    = kernel32.NewProc("GetVolumeInformationW")
	procSetErrorMode     = kernel32.NewProc("SetErrorMode")
)

func init() {
	// Never pop up "There is no disk in the drive" dialogs while scanning.
	procSetErrorMode.Call(0x0001 | 0x8000) // SEM_FAILCRITICALERRORS | SEM_NOOPENFILEERRORBOX
}

func listUF2Drives() []uf2Drive {
	mask, _, _ := procGetLogicalDrives.Call()
	var out []uf2Drive
	for i := 2; i < 26; i++ { // skip A: and B:
		if mask&(1<<uint(i)) == 0 {
			continue
		}
		letter := string(rune('A'+i)) + ":"
		root := letter + `\`
		rp, _ := syscall.UTF16PtrFromString(root)
		t, _, _ := procGetDriveType.Call(uintptr(unsafe.Pointer(rp)))
		if t != 2 && t != 3 { // DRIVE_REMOVABLE, DRIVE_FIXED (some hosts mount UF2 as fixed)
			continue
		}
		label := make([]uint16, 64)
		r, _, _ := procGetVolumeInfo.Call(uintptr(unsafe.Pointer(rp)), uintptr(unsafe.Pointer(&label[0])), 64, 0, 0, 0, 0, 0)
		if r == 0 {
			continue
		}
		b, err := os.ReadFile(root + "INFO_UF2.TXT")
		if err != nil || len(b) > 4096 {
			continue
		}
		d := parseInfoUF2(root, string(b))
		d.Letter = letter
		d.Label = syscall.UTF16ToString(label)
		out = append(out, d)
	}
	return out
}
