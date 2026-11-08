package flashie

import (
	"runtime"
	"syscall"
	"unsafe"
)

var procGetOpenFileNameW = syscall.NewLazyDLL("comdlg32.dll").NewProc("GetOpenFileNameW")

type openFileNameW struct {
	lStructSize       uint32
	hwndOwner         uintptr
	hInstance         uintptr
	lpstrFilter       *uint16
	lpstrCustomFilter *uint16
	nMaxCustFilter    uint32
	nFilterIndex      uint32
	lpstrFile         *uint16
	nMaxFile          uint32
	lpstrFileTitle    *uint16
	nMaxFileTitle     uint32
	lpstrInitialDir   *uint16
	lpstrTitle        *uint16
	flags             uint32
	nFileOffset       uint16
	nFileExtension    uint16
	lpstrDefExt       *uint16
	lCustData         uintptr
	lpfnHook          uintptr
	lpTemplateName    *uint16
	pvReserved        uintptr
	dwReserved        uint32
	flagsEx           uint32
}

func pickImageFile() (string, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	filter := syscall.StringToUTF16("Disk images (*.img, *.iso, *.xz, *.gz, *.bz2, *.zip)\x00*.img;*.iso;*.raw;*.wic;*.xz;*.gz;*.bz2;*.zip\x00All files\x00*.*\x00\x00")
	title, _ := syscall.UTF16PtrFromString("Choose an image to write")
	buf := make([]uint16, 4096)
	ofn := openFileNameW{
		lpstrFilter: &filter[0],
		lpstrFile:   &buf[0],
		nMaxFile:    uint32(len(buf)),
		lpstrTitle:  title,
		flags:       0x00001000 | 0x00000800 | 0x00080000 | 0x00000008, // FILEMUSTEXIST|PATHMUSTEXIST|EXPLORER|NOCHANGEDIR
	}
	ofn.lStructSize = uint32(unsafe.Sizeof(ofn))
	if r, _, _ := procGetOpenFileNameW.Call(uintptr(unsafe.Pointer(&ofn))); r == 0 {
		return "", nil // cancelled
	}
	return syscall.UTF16ToString(buf), nil
}
