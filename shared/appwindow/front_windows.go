//go:build windows

package appwindow

// front_windows.go: bring a newly opened app window to the front.
//
// Windows only lets the program you're using bring windows forward. An app
// started by the launcher is a background program, so its browser window would
// open behind the launcher. WatchNewWindow notes the browser windows already
// open; the function it returns waits for the new one and brings it forward.
// Created by nrmluvdev for Tinkerie.

import (
	"runtime"
	"syscall"
	"time"
	"unsafe"
)

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	kernel32                = syscall.NewLazyDLL("kernel32.dll")
	pEnumWindows            = user32.NewProc("EnumWindows")
	pIsWindowVisible        = user32.NewProc("IsWindowVisible")
	pGetClassNameW          = user32.NewProc("GetClassNameW")
	pGetWindow              = user32.NewProc("GetWindow")
	pGetForegroundWindow    = user32.NewProc("GetForegroundWindow")
	pSetForegroundWindow    = user32.NewProc("SetForegroundWindow")
	pBringWindowToTop       = user32.NewProc("BringWindowToTop")
	pShowWindow             = user32.NewProc("ShowWindow")
	pIsIconic               = user32.NewProc("IsIconic")
	pSetWindowPos           = user32.NewProc("SetWindowPos")
	pGetWindowThreadProcess = user32.NewProc("GetWindowThreadProcessId")
	pAttachThreadInput      = user32.NewProc("AttachThreadInput")
	pKeybdEvent             = user32.NewProc("keybd_event")
	pAllowSetForeground     = user32.NewProc("AllowSetForegroundWindow")
	pGetCurrentThreadId     = kernel32.NewProc("GetCurrentThreadId")
)

// browserWindows lists the visible top-level Chrome/Edge/Brave windows.
func browserWindows() map[uintptr]bool {
	out := map[uintptr]bool{}
	cb := syscall.NewCallback(func(h, _ uintptr) uintptr {
		if v, _, _ := pIsWindowVisible.Call(h); v == 0 {
			return 1
		}
		if owner, _, _ := pGetWindow.Call(h, 4); owner != 0 { // GW_OWNER: skip pop-ups
			return 1
		}
		var cls [64]uint16
		pGetClassNameW.Call(h, uintptr(unsafe.Pointer(&cls[0])), 64)
		if syscall.UTF16ToString(cls[:]) == "Chrome_WidgetWin_1" {
			out[h] = true
		}
		return 1
	})
	pEnumWindows.Call(cb, 0)
	return out
}

// WatchNewWindow notes the browser windows open now. Call the function it
// returns once the app's browser has started: it waits (in the background) for
// the new window and brings it to the front.
func WatchNewWindow() func() {
	pAllowSetForeground.Call(^uintptr(0)) // ASFW_ANY: let the browser come forward if we may
	before := browserWindows()
	return func() {
		go func() {
			deadline := time.Now().Add(20 * time.Second)
			for time.Now().Before(deadline) {
				time.Sleep(150 * time.Millisecond)
				for h := range browserWindows() {
					if !before[h] {
						time.Sleep(250 * time.Millisecond) // let it finish appearing
						ForceForeground(h)
						return
					}
				}
			}
		}()
	}
}

// ForceForeground brings window h to the front and gives it the keyboard,
// even when this program isn't the one in front.
func ForceForeground(h uintptr) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if ic, _, _ := pIsIconic.Call(h); ic != 0 {
		pShowWindow.Call(h, 9) // SW_RESTORE
	}
	fg, _, _ := pGetForegroundWindow.Call()
	if fg == h {
		return
	}
	// borrow the front window's input for a moment: then Windows lets us switch
	me, _, _ := pGetCurrentThreadId.Call()
	front, _, _ := pGetWindowThreadProcess.Call(fg, 0)
	if front != 0 && front != me {
		pAttachThreadInput.Call(me, front, 1)
		defer pAttachThreadInput.Call(me, front, 0)
	}
	pBringWindowToTop.Call(h)
	pSetForegroundWindow.Call(h)
	if now, _, _ := pGetForegroundWindow.Call(); now != h {
		// a tap of the Alt key also counts as you using the computer
		pKeybdEvent.Call(0x12, 0, 0, 0)
		pKeybdEvent.Call(0x12, 0, 2, 0) // KEYEVENTF_KEYUP
		pSetForegroundWindow.Call(h)
	}
	if now, _, _ := pGetForegroundWindow.Call(); now != h {
		// still not allowed: at least put it on top of the others
		const swpNoMove, swpNoSize, swpShow = 0x2, 0x1, 0x40
		pSetWindowPos.Call(h, ^uintptr(0), 0, 0, 0, 0, swpNoMove|swpNoSize|swpShow) // HWND_TOPMOST
		pSetWindowPos.Call(h, ^uintptr(1), 0, 0, 0, 0, swpNoMove|swpNoSize|swpShow) // HWND_NOTOPMOST
	}
}
