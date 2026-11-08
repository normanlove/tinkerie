//go:build windows

package meterie

// win_windows.go: the native Win32 window, GDI text engine, input, timers,
// clipboard, dialogs and sound — pure Go via system DLLs (no cgo, no extra DLLs).

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"
)

func init() { runtime.LockOSThread() }

func sysDLL(name string) *syscall.LazyDLL {
	root := os.Getenv("SystemRoot")
	if root == "" {
		root = `C:\Windows`
	}
	return syscall.NewLazyDLL(filepath.Join(root, "System32", name))
}

var (
	user32   = sysDLL("user32.dll")
	gdi32    = sysDLL("gdi32.dll")
	kernel32 = sysDLL("kernel32.dll")
	winmm    = sysDLL("winmm.dll")
	dwmapi   = sysDLL("dwmapi.dll")
	comdlg32 = sysDLL("comdlg32.dll")
	shell32  = sysDLL("shell32.dll")
	ole32    = sysDLL("ole32.dll")

	pRegisterClassExW     = user32.NewProc("RegisterClassExW")
	pCreateWindowExW      = user32.NewProc("CreateWindowExW")
	pDefWindowProcW       = user32.NewProc("DefWindowProcW")
	pGetMessageW          = user32.NewProc("GetMessageW")
	pTranslateMessage     = user32.NewProc("TranslateMessage")
	pDispatchMessageW     = user32.NewProc("DispatchMessageW")
	pPostQuitMessage      = user32.NewProc("PostQuitMessage")
	pPostMessageW         = user32.NewProc("PostMessageW")
	pSendMessageW         = user32.NewProc("SendMessageW")
	pShowWindow           = user32.NewProc("ShowWindow")
	pUpdateWindow         = user32.NewProc("UpdateWindow")
	pInvalidateRect       = user32.NewProc("InvalidateRect")
	pBeginPaint           = user32.NewProc("BeginPaint")
	pEndPaint             = user32.NewProc("EndPaint")
	pGetClientRect        = user32.NewProc("GetClientRect")
	pSetTimer             = user32.NewProc("SetTimer")
	pLoadCursorW          = user32.NewProc("LoadCursorW")
	pSetCursor            = user32.NewProc("SetCursor")
	pSetCapture           = user32.NewProc("SetCapture")
	pReleaseCapture       = user32.NewProc("ReleaseCapture")
	pScreenToClient       = user32.NewProc("ScreenToClient")
	pGetKeyState          = user32.NewProc("GetKeyState")
	pGetDpiForWindow      = user32.NewProc("GetDpiForWindow")
	pSetDpiAwarenessCtx   = user32.NewProc("SetProcessDpiAwarenessContext")
	pSetProcessDPIAware   = user32.NewProc("SetProcessDPIAware")
	pSetWindowPos         = user32.NewProc("SetWindowPos")
	pMessageBoxW          = user32.NewProc("MessageBoxW")
	pOpenClipboard        = user32.NewProc("OpenClipboard")
	pCloseClipboard       = user32.NewProc("CloseClipboard")
	pEmptyClipboard       = user32.NewProc("EmptyClipboard")
	pSetClipboardData     = user32.NewProc("SetClipboardData")
	pGetClipboardData     = user32.NewProc("GetClipboardData")
	pCreateIconIndirect   = user32.NewProc("CreateIconIndirect")
	pTrackMouseEvent      = user32.NewProc("TrackMouseEvent")
	pSystemParametersInfo = user32.NewProc("SystemParametersInfoW")
	pGetDC                = user32.NewProc("GetDC")
	pReleaseDC            = user32.NewProc("ReleaseDC")
	pDestroyWindow        = user32.NewProc("DestroyWindow")

	pCreateCompatibleDC   = gdi32.NewProc("CreateCompatibleDC")
	pCreateDIBSection     = gdi32.NewProc("CreateDIBSection")
	pSelectObject         = gdi32.NewProc("SelectObject")
	pDeleteObject         = gdi32.NewProc("DeleteObject")
	pDeleteDC             = gdi32.NewProc("DeleteDC")
	pBitBlt               = gdi32.NewProc("BitBlt")
	pCreateFontW          = gdi32.NewProc("CreateFontW")
	pSetTextColor         = gdi32.NewProc("SetTextColor")
	pSetBkMode            = gdi32.NewProc("SetBkMode")
	pExtTextOutW          = gdi32.NewProc("ExtTextOutW")
	pGetTextExtentPoint32 = gdi32.NewProc("GetTextExtentPoint32W")
	pGetTextMetricsW      = gdi32.NewProc("GetTextMetricsW")
	pGetTextFaceW         = gdi32.NewProc("GetTextFaceW")
	pGdiFlush             = gdi32.NewProc("GdiFlush")
	pCreateBitmap         = gdi32.NewProc("CreateBitmap")

	pGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
	pGlobalAlloc      = kernel32.NewProc("GlobalAlloc")
	pGlobalLock       = kernel32.NewProc("GlobalLock")
	pGlobalUnlock     = kernel32.NewProc("GlobalUnlock")

	pPlaySoundW = winmm.NewProc("PlaySoundW")

	pDwmSetWindowAttribute = dwmapi.NewProc("DwmSetWindowAttribute")

	pGetSaveFileNameW = comdlg32.NewProc("GetSaveFileNameW")
	pGetOpenFileNameW = comdlg32.NewProc("GetOpenFileNameW")
	pShellExecuteW    = shell32.NewProc("ShellExecuteW")
	pCoTaskMemFree    = ole32.NewProc("CoTaskMemFree")
)

const (
	wmDestroy     = 0x0002
	wmSize        = 0x0005
	wmPaint       = 0x000F
	wmClose       = 0x0010
	wmEraseBkgnd  = 0x0014
	wmSetCursor   = 0x0020
	wmGetMinMax   = 0x0024
	wmKeyDown     = 0x0100
	wmChar        = 0x0102
	wmSysKeyDown  = 0x0104
	wmTimer       = 0x0113
	wmMouseMove   = 0x0200
	wmLButtonDown = 0x0201
	wmLButtonUp   = 0x0202
	wmLButtonDbl  = 0x0203
	wmRButtonDown = 0x0204
	wmRButtonUp   = 0x0205
	wmMouseWheel  = 0x020A
	wmMouseLeave  = 0x02A3
	wmDpiChanged  = 0x02E0
	wmSetIcon     = 0x0080
	wmApp         = 0x8000
	wmActivate    = 0x0006
	wmKillFocus   = 0x0008
	wmCaptureChg  = 0x0215
)

type wndClassEx struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSm     uintptr
}

type msgT struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      [2]int32
	Private uint32
}

type rectT struct{ Left, Top, Right, Bottom int32 }

type paintStruct struct {
	Hdc         uintptr
	Erase       int32
	RcPaint     rectT
	Restore     int32
	IncUpdate   int32
	RgbReserved [32]byte
}

type bitmapInfoHeader struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

type textMetric struct {
	Height, Ascent, Descent, InternalLeading, ExternalLeading int32
	AveCharWidth, MaxCharWidth, Weight, Overhang              int32
	DigitizedAspectX, DigitizedAspectY                        int32
	FirstChar, LastChar, DefaultChar, BreakChar               uint16
	Italic, Underlined, StruckOut, PitchAndFamily, CharSet    byte
}

func u16(s string) *uint16 { p, _ := syscall.UTF16PtrFromString(s); return p }

// ---------------------------------------------------------------------------
// DIB backbuffer

type dibBuf struct {
	dc, bmp, old uintptr
	w, h         int
	pix          []byte
}

func newDIB(w, h int) *dibBuf {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	d := &dibBuf{w: w, h: h}
	d.dc, _, _ = pCreateCompatibleDC.Call(0)
	bi := bitmapInfoHeader{Size: uint32(unsafe.Sizeof(bitmapInfoHeader{})), Width: int32(w), Height: -int32(h), Planes: 1, BitCount: 32}
	var bits uintptr
	d.bmp, _, _ = pCreateDIBSection.Call(d.dc, uintptr(unsafe.Pointer(&bi)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if d.bmp == 0 || bits == 0 {
		return nil
	}
	d.old, _, _ = pSelectObject.Call(d.dc, d.bmp)
	d.pix = unsafe.Slice((*byte)(unsafe.Pointer(bits)), w*h*4)
	pSetBkMode.Call(d.dc, 1) // TRANSPARENT
	return d
}

func (d *dibBuf) free() {
	if d == nil {
		return
	}
	pSelectObject.Call(d.dc, d.old)
	pDeleteObject.Call(d.bmp)
	pDeleteDC.Call(d.dc)
}

// ---------------------------------------------------------------------------
// GDI text engine

type fontKey struct {
	size   int
	weight int
	mono   bool
	icon   bool
}

type gdiFonts struct {
	fonts    map[fontKey]uintptr
	lh       map[fontKey]int
	iconFace string
	uiFace   string
}

var fontCache = &gdiFonts{fonts: map[fontKey]uintptr{}, lh: map[fontKey]int{}}

type gdiText struct {
	dc   uintptr
	clip rectT
	cur  uintptr
}

func (fc *gdiFonts) get(dc uintptr, f Font) (uintptr, fontKey) {
	k := fontKey{int(f.Size*4 + 0.5), f.Weight, f.Mono, f.Icon}
	if h, ok := fc.fonts[k]; ok {
		return h, k
	}
	if fc.iconFace == "" {
		fc.iconFace = pickFace(dc, []string{"Segoe Fluent Icons", "Segoe MDL2 Assets"})
		fc.uiFace = pickFace(dc, []string{"Segoe UI Variable Text", "Segoe UI"})
	}
	face := fc.uiFace
	quality := uintptr(5) // CLEARTYPE_QUALITY
	if f.Mono {
		face = "Consolas"
	}
	if f.Icon {
		face = fc.iconFace
		quality = 4 // ANTIALIASED
	}
	w := f.Weight
	if w == 0 {
		w = 400
	}
	h, _, _ := pCreateFontW.Call(uintptr(-int32(float64(k.size)/4+0.5)), 0, 0, 0, uintptr(w), 0, 0, 0, 1, 0, 0, quality, 0, uintptr(unsafe.Pointer(u16(face))))
	fc.fonts[k] = h
	return h, k
}

func pickFace(dc uintptr, faces []string) string {
	for _, f := range faces {
		h, _, _ := pCreateFontW.Call(uintptr(-20&0xFFFFFFFF), 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, 4, 0, uintptr(unsafe.Pointer(u16(f))))
		old, _, _ := pSelectObject.Call(dc, h)
		buf := make([]uint16, 64)
		pGetTextFaceW.Call(dc, 64, uintptr(unsafe.Pointer(&buf[0])))
		pSelectObject.Call(dc, old)
		pDeleteObject.Call(h)
		if strings.EqualFold(syscall.UTF16ToString(buf), f) {
			return f
		}
	}
	return faces[len(faces)-1]
}

func (t *gdiText) sel(f Font) fontKey {
	h, k := fontCache.get(t.dc, f)
	if t.cur != h {
		pSelectObject.Call(t.dc, h)
		t.cur = h
	}
	return k
}

func (t *gdiText) Measure(s string, f Font) int {
	if s == "" {
		return 0
	}
	t.sel(f)
	w := utf16.Encode([]rune(s))
	var sz [2]int32
	pGetTextExtentPoint32.Call(t.dc, uintptr(unsafe.Pointer(&w[0])), uintptr(len(w)), uintptr(unsafe.Pointer(&sz)))
	return int(sz[0])
}

func (t *gdiText) LineHeight(f Font) int {
	k := t.sel(f)
	if v, ok := fontCache.lh[k]; ok {
		return v
	}
	var tm textMetric
	pGetTextMetricsW.Call(t.dc, uintptr(unsafe.Pointer(&tm)))
	fontCache.lh[k] = int(tm.Height)
	return int(tm.Height)
}

func (t *gdiText) Draw(s string, x, y int, f Font, col Color) {
	if s == "" || col.A == 0 || t.clip.Right <= t.clip.Left || t.clip.Bottom <= t.clip.Top {
		return
	}
	t.sel(f)
	// GDI has no per-text alpha: pre-blend translucent text toward the panel colour.
	if col.A < 255 {
		col = lerpColor(th.Card, Color{col.R, col.G, col.B, 255}, float64(col.A)/255)
	}
	pSetTextColor.Call(t.dc, uintptr(uint32(col.R)|uint32(col.G)<<8|uint32(col.B)<<16))
	w := utf16.Encode([]rune(s))
	pExtTextOutW.Call(t.dc, uintptr(int32(x)), uintptr(int32(y)), 4, uintptr(unsafe.Pointer(&t.clip)), uintptr(unsafe.Pointer(&w[0])), uintptr(len(w)), 0)
	pGdiFlush.Call()
}

func (t *gdiText) SetClip(r Rect) {
	t.clip = rectT{int32(r.X), int32(r.Y), int32(r.Right()), int32(r.Bottom())}
}

// ---------------------------------------------------------------------------
// Window

var win struct {
	hwnd     uintptr
	dib      *dibBuf
	text     *gdiText
	app      *App
	in       Input
	scale    float64
	dpi      int
	wantDraw bool
	lastDraw time.Time
	cursors  map[int]uintptr
	tracking bool
	minW     int
	minH     int
}

func runWindows() {
	if r, _, _ := pSetDpiAwarenessCtx.Call(^uintptr(3)); r == 0 { // DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 = -4
		pSetProcessDPIAware.Call()
	}
	app := NewApp()
	win.app = app
	wakeFn = func() {
		if win.hwnd != 0 {
			pPostMessageW.Call(win.hwnd, wmApp+1, 0, 0)
		}
	}
	app.ui.Clipboard = platformClipboardGet
	app.ui.SetClipboard = platformClipboardSet
	win.cursors = map[int]uintptr{}
	for k, id := range map[int]uintptr{CurArrow: 32512, CurHand: 32649, CurIBeam: 32513, CurMove: 32646, CurSizeWE: 32644, CurCross: 32515} {
		h, _, _ := pLoadCursorW.Call(0, id)
		win.cursors[k] = h
	}
	hinst, _, _ := pGetModuleHandleW.Call(0)
	cls := u16("MeterieWindow")
	wc := wndClassEx{Style: 0x0008 | 0x0002 | 0x0001, WndProc: syscall.NewCallback(wndProc), Instance: hinst, Cursor: win.cursors[CurArrow], ClassName: cls}
	wc.Size = uint32(unsafe.Sizeof(wc))
	big, small := makeAppIcons()
	wc.Icon, wc.IconSm = big, small
	pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))

	// initial size: 1440x920 logical, clamped to the work area
	var wa rectT
	pSystemParametersInfo.Call(0x0030, 0, uintptr(unsafe.Pointer(&wa)), 0) // SPI_GETWORKAREA
	sysDPI := 96
	if dc, _, _ := pGetDC.Call(0); dc != 0 {
		pReleaseDC.Call(0, dc)
	}
	_ = sysDPI
	w, h := 1440, 920
	ww, wh := int(wa.Right-wa.Left), int(wa.Bottom-wa.Top)
	if ww > 0 && w > ww-40 {
		w = ww - 40
	}
	if wh > 0 && h > wh-40 {
		h = wh - 40
	}
	x := int(wa.Left) + (ww-w)/2
	y := int(wa.Top) + (wh-h)/2
	hwnd, _, _ := pCreateWindowExW.Call(0, uintptr(unsafe.Pointer(cls)), uintptr(unsafe.Pointer(u16("Meterie — PC multimeter by nrmluvdev"))),
		0x00CF0000|0x02000000, uintptr(x), uintptr(y), uintptr(w), uintptr(h), 0, 0, hinst, 0) // WS_OVERLAPPEDWINDOW|WS_CLIPCHILDREN
	if hwnd == 0 {
		fatalBox("Meterie could not create its window.")
	}
	win.hwnd = hwnd
	updateDPI()
	// rescale the window to the monitor DPI on first show
	if win.dpi != 96 {
		sw, sh := int(float64(w)*float64(win.dpi)/96), int(float64(h)*float64(win.dpi)/96)
		if ww > 0 {
			sw, sh = min(sw, ww-20), min(sh, wh-20)
		}
		pSetWindowPos.Call(hwnd, 0, uintptr(int(wa.Left)+(ww-sw)/2), uintptr(int(wa.Top)+(wh-sh)/2), uintptr(sw), uintptr(sh), 0x0004)
	}
	dark := int32(1)
	pDwmSetWindowAttribute.Call(hwnd, 20, uintptr(unsafe.Pointer(&dark)), 4) // immersive dark mode
	cap := uint32(th.Rail.R) | uint32(th.Rail.G)<<8 | uint32(th.Rail.B)<<16
	pDwmSetWindowAttribute.Call(hwnd, 35, uintptr(unsafe.Pointer(&cap)), 4) // caption colour (Win11)
	txt := uint32(0xE8EEF6)
	txt = (txt>>16)&0xFF | txt&0xFF00 | (txt&0xFF)<<16
	pDwmSetWindowAttribute.Call(hwnd, 36, uintptr(unsafe.Pointer(&txt)), 4)
	pSendMessageW.Call(hwnd, wmSetIcon, 1, big)
	pSendMessageW.Call(hwnd, wmSetIcon, 0, small)
	win.wantDraw = true
	pShowWindow.Call(hwnd, 3) // SW_SHOWMAXIMIZED: every Tinkerie app opens maximised
	renderFrame()
	pUpdateWindow.Call(hwnd)
	pSetTimer.Call(hwnd, 1, 15, 0)
	app.startAPI()
	if len(os.Args) > 1 {
		p := os.Args[1]
		if strings.HasSuffix(strings.ToLower(p), ".meterie") {
			if err := app.loadSession(p); err == nil {
				app.sessionPath = p
			}
		} else if strings.HasSuffix(strings.ToLower(p), ".csv") {
			app.importTimeline(p)
		}
	}
	var m msgT
	for {
		r, _, _ := pGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		pDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
	app.shutdown()
}

func updateDPI() {
	d, _, _ := pGetDpiForWindow.Call(win.hwnd)
	if d == 0 {
		d = 96
	}
	win.dpi = int(d)
	us := win.app.settings.UIScale
	if us <= 0 {
		us = 1
	}
	win.scale = float64(d) / 96 * us
	win.app.ui.S = win.scale
}

func clientSize() (int, int) {
	var r rectT
	pGetClientRect.Call(win.hwnd, uintptr(unsafe.Pointer(&r)))
	return int(r.Right - r.Left), int(r.Bottom - r.Top)
}

func invalidate() { pInvalidateRect.Call(win.hwnd, 0, 0) }

func lparamXY(lp uintptr) (int, int) { return int(int16(lp & 0xFFFF)), int(int16((lp >> 16) & 0xFFFF)) }

func keyDown(vk int32) bool {
	r, _, _ := pGetKeyState.Call(uintptr(vk))
	return int16(r) < 0
}

func mapKey(vk uintptr) Key {
	switch vk {
	case 0x08:
		return KBack
	case 0x2E:
		return KDelete
	case 0x25:
		return KLeft
	case 0x27:
		return KRight
	case 0x26:
		return KUp
	case 0x28:
		return KDown
	case 0x24:
		return KHome
	case 0x23:
		return KEnd
	case 0x0D:
		return KEnter
	case 0x1B:
		return KEscape
	case 0x09:
		return KTab
	case 0x20:
		return KSpace
	case 0x70:
		return KF1
	case 0x71:
		return KF2
	case 'A':
		return KA
	case 'C':
		return KC
	case 'V':
		return KV
	case 'X':
		return KX
	case 'S':
		return KS
	case 'O':
		return KO
	case 'N':
		return KN
	case 'Z':
		return KZ
	case 'H':
		return KH
	case 'R':
		return KR
	case 'M':
		return KM
	}
	return KNone
}

func wndProc(hwnd, msg, wp, lp uintptr) uintptr {
	in := &win.in
	switch msg {
	case wmPaint:
		var ps paintStruct
		hdc, _, _ := pBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		if win.dib != nil {
			pBitBlt.Call(hdc, 0, 0, uintptr(win.dib.w), uintptr(win.dib.h), win.dib.dc, 0, 0, 0x00CC0020)
		}
		pEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		return 0
	case wmEraseBkgnd:
		return 1
	case wmSize:
		w, h := clientSize()
		if w > 0 && h > 0 && (win.dib == nil || win.dib.w != w || win.dib.h != h) {
			win.dib.free()
			win.dib = newDIB(w, h)
			if win.dib != nil {
				win.text = &gdiText{dc: win.dib.dc}
			}
		}
		renderFrame()
		invalidate()
		return 0
	case wmTimer:
		if win.wantDraw || win.app.wantsFrame() {
			win.wantDraw = false
			renderFrame()
			invalidate()
		}
		return 0
	case wmApp + 1:
		win.wantDraw = true
		return 0
	case wmMouseMove:
		in.MX, in.MY = lparamXY(lp)
		in.HasMouse = true
		if !win.tracking {
			tme := struct {
				Size, Flags uint32
				Hwnd        uintptr
				Hover       uint32
			}{Flags: 0x2, Hwnd: hwnd}
			tme.Size = uint32(unsafe.Sizeof(tme))
			pTrackMouseEvent.Call(uintptr(unsafe.Pointer(&tme)))
			win.tracking = true
		}
		win.wantDraw = true
		return 0
	case wmMouseLeave:
		win.tracking = false
		if !in.Down {
			in.HasMouse = false
		}
		win.wantDraw = true
		return 0
	case wmLButtonDown, wmLButtonDbl:
		in.MX, in.MY = lparamXY(lp)
		in.Down, in.Pressed, in.HasMouse = true, true, true
		if msg == wmLButtonDbl {
			in.Dbl = true
		}
		pSetCapture.Call(hwnd)
		win.wantDraw = true
		return 0
	case wmLButtonUp:
		in.MX, in.MY = lparamXY(lp)
		in.Down, in.Released = false, true
		pReleaseCapture.Call()
		win.wantDraw = true
		return 0
	case wmCaptureChg:
		if in.Down {
			in.Down, in.Released = false, true
		}
		return 0
	case wmRButtonDown:
		in.MX, in.MY = lparamXY(lp)
		in.RPressed = true
		win.wantDraw = true
		return 0
	case wmMouseWheel:
		in.Wheel += float64(int16(wp>>16)) / 120
		win.wantDraw = true
		return 0
	case wmKeyDown, wmSysKeyDown:
		if k := mapKey(wp); k != KNone {
			in.Keys = append(in.Keys, k)
			win.wantDraw = true
		}
		if msg == wmSysKeyDown {
			break
		}
		return 0
	case wmChar:
		if wp >= 32 && wp != 127 {
			in.Chars = append(in.Chars, rune(wp))
			win.wantDraw = true
		}
		return 0
	case wmSetCursor:
		if lp&0xFFFF == 1 { // HTCLIENT
			pSetCursor.Call(win.cursors[win.app.ui.Cursor])
			return 1
		}
	case wmGetMinMax:
		mmi := (*[10]int32)(unsafe.Pointer(lp))
		mmi[6] = int32(980 * win.scale) // ptMinTrackSize.x
		mmi[7] = int32(640 * win.scale)
		return 0
	case wmDpiChanged:
		updateDPI()
		r := (*rectT)(unsafe.Pointer(lp))
		pSetWindowPos.Call(hwnd, 0, uintptr(r.Left), uintptr(r.Top), uintptr(r.Right-r.Left), uintptr(r.Bottom-r.Top), 0x0004|0x0010)
		return 0
	case wmClose:
		if win.app.log.Active() {
			r, _, _ := pMessageBoxW.Call(hwnd, uintptr(unsafe.Pointer(u16("Logging is still running. Stop logging and quit Meterie?"))), uintptr(unsafe.Pointer(u16("Meterie"))), 0x4|0x30)
			if r != 6 {
				return 0
			}
		}
		pDestroyWindow.Call(hwnd)
		return 0
	case wmDestroy:
		pPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := pDefWindowProcW.Call(hwnd, msg, wp, lp)
	return r
}

func (a *App) wantsFrame() bool {
	inboxMu.Lock()
	n := len(inbox)
	inboxMu.Unlock()
	if n > 0 || a.ui.Animating {
		return true
	}
	// slow ambient animation (live pulse, recording dot, camera) at ~25 fps
	if (a.ui.Ambient || a.cam.running || a.log.Active()) && time.Since(win.lastDraw) > 40*time.Millisecond {
		return true
	}
	return false
}

var inFrame bool

func renderFrame() {
	// Modal dialogs opened from inside a frame run a nested message loop; never re-enter.
	if win.dib == nil || inFrame {
		return
	}
	inFrame = true
	defer func() { inFrame = false }()
	win.in.Ctrl = keyDown(0x11)
	win.in.Shift = keyDown(0x10)
	us := win.app.settings.UIScale
	if us <= 0 {
		us = 1
	}
	win.app.ui.S = float64(win.dpi) / 96 * us
	c := WrapCanvas(win.dib.pix, win.dib.w, win.dib.h)
	in := win.in
	win.app.Frame(c, win.text, in)
	pGdiFlush.Call()
	win.lastDraw = time.Now()
	// reset one-shot input
	win.in.Pressed, win.in.Released, win.in.RPressed, win.in.Dbl = false, false, false, false
	win.in.Wheel = 0
	win.in.Keys = win.in.Keys[:0]
	win.in.Chars = win.in.Chars[:0]
	pSetCursor.Call(win.cursors[win.app.ui.Cursor])
	if win.app.ui.Animating {
		win.wantDraw = true
	}
}

// ---------------------------------------------------------------------------
// Icons (rendered from the logo; alpha recovered by drawing on black and white)

func makeAppIcons() (uintptr, uintptr) {
	return makeIcon(48), makeIcon(16)
}

func renderLogoRGBA(sz int) []byte {
	img := renderLogoImage(sz)
	out := make([]byte, sz*sz*4) // BGRA, straight alpha
	for i := 0; i < sz*sz; i++ {
		out[i*4+0], out[i*4+1], out[i*4+2], out[i*4+3] = img.Pix[i*4+2], img.Pix[i*4+1], img.Pix[i*4+0], img.Pix[i*4+3]
	}
	return out
}

func makeIcon(sz int) uintptr {
	px := renderLogoRGBA(sz)
	bi := bitmapInfoHeader{Size: 40, Width: int32(sz), Height: -int32(sz), Planes: 1, BitCount: 32}
	dc, _, _ := pGetDC.Call(0)
	var bits uintptr
	color, _, _ := pCreateDIBSection.Call(dc, uintptr(unsafe.Pointer(&bi)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
	pReleaseDC.Call(0, dc)
	if color == 0 {
		return 0
	}
	copy(unsafe.Slice((*byte)(unsafe.Pointer(bits)), sz*sz*4), px)
	mask, _, _ := pCreateBitmap.Call(uintptr(sz), uintptr(sz), 1, 1, 0)
	ii := struct {
		FIcon    int32
		XHot     uint32
		YHot     uint32
		HbmMask  uintptr
		HbmColor uintptr
	}{FIcon: 1, HbmMask: mask, HbmColor: color}
	h, _, _ := pCreateIconIndirect.Call(uintptr(unsafe.Pointer(&ii)))
	return h
}

// ---------------------------------------------------------------------------
// Platform services used by the app

func platformClipboardSet(s string) {
	if r, _, _ := pOpenClipboard.Call(win.hwnd); r == 0 {
		return
	}
	defer pCloseClipboard.Call()
	pEmptyClipboard.Call()
	w := utf16.Encode([]rune(s + "\x00"))
	h, _, _ := pGlobalAlloc.Call(0x0002, uintptr(len(w)*2))
	if h == 0 {
		return
	}
	p, _, _ := pGlobalLock.Call(h)
	copy(unsafe.Slice((*uint16)(unsafe.Pointer(p)), len(w)), w)
	pGlobalUnlock.Call(h)
	pSetClipboardData.Call(13, h) // CF_UNICODETEXT
}

func platformClipboardGet() string {
	if r, _, _ := pOpenClipboard.Call(win.hwnd); r == 0 {
		return ""
	}
	defer pCloseClipboard.Call()
	h, _, _ := pGetClipboardData.Call(13)
	if h == 0 {
		return ""
	}
	p, _, _ := pGlobalLock.Call(h)
	if p == 0 {
		return ""
	}
	defer pGlobalUnlock.Call(h)
	var out []uint16
	for i := 0; i < 1<<16; i++ {
		c := *(*uint16)(unsafe.Pointer(p + uintptr(i*2)))
		if c == 0 {
			break
		}
		out = append(out, c)
	}
	return string(utf16.Decode(out))
}

const sndAsync, sndNoDefault, sndMemory, sndLoop = 0x1, 0x2, 0x4, 0x8

var playing []byte // keep the buffer alive while PlaySound uses it

func platformPlaySound(wav []byte, loop bool) {
	playing = wav
	fl := uintptr(sndAsync | sndNoDefault | sndMemory)
	if loop {
		fl |= sndLoop
	}
	pPlaySoundW.Call(uintptr(unsafe.Pointer(&playing[0])), 0, fl)
}

func platformStopSound() { pPlaySoundW.Call(0, 0, 0) }

func platformReveal(path string) {
	if fi, err := os.Stat(path); err == nil && fi.IsDir() {
		exec.Command("explorer", path).Start()
		return
	}
	exec.Command("explorer", "/select,", path).Start()
}

func fatalBox(msg string) {
	pMessageBoxW.Call(0, uintptr(unsafe.Pointer(u16(msg))), uintptr(unsafe.Pointer(u16("Meterie"))), 0x10)
	os.Exit(1)
}

func platformOffscreen(w, h int) (*Canvas, TextEngine, func()) {
	d := newDIB(w, h)
	if d == nil {
		return nil, nil, func() {}
	}
	t := &gdiText{dc: d.dc}
	t.SetClip(Rect{0, 0, w, h})
	return WrapCanvas(d.pix, w, h), t, func() { d.free() }
}

// ---------------------------------------------------------------------------
// Common file dialogs

type openFileName struct {
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
	Flags             uint32
	nFileOffset       uint16
	nFileExtension    uint16
	lpstrDefExt       *uint16
	lCustData         uintptr
	lpfnHook          uintptr
	lpTemplateName    *uint16
	pvReserved        uintptr
	dwReserved        uint32
	FlagsEx           uint32
}

func filterUTF16(s string) *uint16 {
	u := utf16.Encode([]rune(s + "\x00\x00"))
	return &u[0]
}

func fileDialog(save bool, title, def, filter, pattern, ext string) (string, bool) {
	file := make([]uint16, 4096)
	if def != "" {
		copy(file, utf16.Encode([]rune(filepath.Base(def))))
	}
	dir := filepath.Dir(def)
	if def == "" {
		dir, _ = os.UserHomeDir()
	}
	os.MkdirAll(dir, 0o755)
	var de *uint16
	if ext != "" {
		de = u16(ext)
	}
	ofn := openFileName{hwndOwner: win.hwnd, lpstrFilter: filterUTF16(filter + "\x00" + pattern + "\x00All files (*.*)\x00*.*"),
		nFilterIndex: 1, lpstrFile: &file[0], nMaxFile: uint32(len(file)), lpstrInitialDir: u16(dir), lpstrTitle: u16(title), lpstrDefExt: de}
	ofn.lStructSize = uint32(unsafe.Sizeof(ofn))
	var r uintptr
	if save {
		ofn.Flags = 0x2 | 0x800 | 0x8 | 0x80000
		r, _, _ = pGetSaveFileNameW.Call(uintptr(unsafe.Pointer(&ofn)))
	} else {
		ofn.Flags = 0x1000 | 0x800 | 0x8 | 0x80000
		r, _, _ = pGetOpenFileNameW.Call(uintptr(unsafe.Pointer(&ofn)))
	}
	// the modal dialog swallowed our button-up; resync input
	win.in.Down = false
	win.app.ui.active = ""
	if r == 0 {
		return "", false
	}
	return syscall.UTF16ToString(file), true
}

func platformSaveDialog(title, def, filter, ext string) (string, bool) {
	return fileDialog(true, title, def, filter, "*."+ext, ext)
}

func platformOpenDialog(title, filter, pattern string) (string, bool) {
	h, _ := os.UserHomeDir()
	return fileDialog(false, title, filepath.Join(h, "Documents", "Meterie", ""), filter, pattern, "")
}
