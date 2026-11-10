// Codie — rules_win.go: Windows Go — loading Windows DLLs and calling the
// Windows API from Go with syscall and golang.org/x/sys/windows.
// Part of the free nrmluvdev maker suite. Created by nrmluvdev.
package codie

import (
	"strings"
)

var winDLLs = map[string]string{
	"kernel32.dll":  "Windows' core: files, drives, memory, processes, time, serial ports",
	"user32.dll":    "windows, message boxes, the keyboard and mouse",
	"gdi32.dll":     "classic Windows drawing: pens, brushes, fonts, bitmaps",
	"shell32.dll":   "Explorer features: opening files, special folders, tray icons",
	"advapi32.dll":  "security, the registry and services",
	"ole32.dll":     "COM, the system Windows components use to talk to each other",
	"winmm.dll":     "older multimedia: sounds, MIDI, joysticks and high-precision timers",
	"xinput1_4.dll": "Xbox-style game controllers (XInput)",
	"comdlg32.dll":  "standard Open and Save dialogs",
	"dwmapi.dll":    "the desktop window manager: dark title bars and window effects",
	"setupapi.dll":  "listing hardware devices (USB, COM ports)",
	"ntdll.dll":     "the lowest-level Windows functions",
	"ws2_32.dll":    "network sockets",
	"version.dll":   "reading version information from files",
}

var winFuncs = map[string]string{
	"MessageBoxW":           "shows a message box with a title, text and buttons",
	"GetLogicalDrives":      "gives a bit mask of the drive letters in use (bit 0 = A:, bit 2 = C:…)",
	"GetDriveTypeW":         "says whether a drive is removable, fixed, a network share or a CD",
	"GetVolumeInformationW": "reads a drive's label and file system name",
	"GetDiskFreeSpaceExW":   "reads a drive's size and free space",
	"CreateFileW":           "opens a file, drive or device and gives back a handle",
	"ReadFile":              "reads bytes through a handle", "WriteFile": "writes bytes through a handle", "CloseHandle": "closes a handle",
	"DeviceIoControl":  "sends a control code to a device driver",
	"GetAsyncKeyState": "says whether a key is held down right now, even if another window has focus",
	"GetKeyState":      "says whether a key is down (or toggled, like Caps Lock)",
	"FindWindowW":      "finds a window by its class name or title", "ShowWindow": "shows, hides, minimises or maximises a window",
	"SetWindowTextW": "changes a window's title", "GetForegroundWindow": "gets the window the user is using",
	"SetForegroundWindow": "brings a window to the front",
	"ShellExecuteW":       "opens a file, folder or web address with its usual program",
	"GetModuleHandleW":    "gets a handle to a loaded module (often the program itself)",
	"LoadLibraryW":        "loads a DLL", "GetProcAddress": "finds a function inside a DLL",
	"Sleep": "pauses for a number of milliseconds", "GetTickCount64": "gives the milliseconds since Windows started",
	"QueryPerformanceCounter":   "reads a very precise timer, used to measure frame times",
	"QueryPerformanceFrequency": "says how many ticks the precise timer makes per second",
	"timeBeginPeriod":           "makes Windows' timers more precise (for example 1 ms), which steadies game loops",
	"timeEndPeriod":             "undoes timeBeginPeriod",
	"Beep":                      "plays a beep of a frequency and length", "MessageBeep": "plays a standard Windows sound",
	"PlaySoundW":       "plays a .wav sound",
	"GetSystemMetrics": "reads screen sizes and other system measurements",
	"SetConsoleTitleW": "changes the console window's title", "GetConsoleWindow": "gets the console window's handle",
	"AllocConsole": "opens a console window for a GUI program", "FreeConsole": "detaches from the console window",
	"GetCommState": "reads a serial port's settings", "SetCommState": "changes a serial port's speed and format",
	"SetCommTimeouts": "sets how long serial reads and writes wait", "PurgeComm": "throws away data waiting in a serial port",
	"QueryDosDeviceW": "lists device names, such as COM ports",
	"CreateWindowExW": "creates a window", "RegisterClassExW": "registers a window class (its look and message handler)",
	"DefWindowProcW": "does the default handling of a window message",
	"GetMessageW":    "waits for the next window message", "PeekMessageW": "checks for a window message without waiting",
	"TranslateMessage": "turns key presses into character messages", "DispatchMessageW": "sends a message to the window's handler",
	"PostQuitMessage":      "asks the message loop to end",
	"SHGetKnownFolderPath": "gets a special folder's path, such as Documents",
	"GetUserNameW":         "gets the signed-in user's name", "GetComputerNameW": "gets the computer's name",
	"XInputGetState": "reads an Xbox controller's buttons and sticks",
	"XInputSetState": "makes an Xbox controller vibrate",
	"GetCursorPos":   "reads the mouse pointer's position on screen", "SetCursorPos": "moves the mouse pointer",
	"mciSendStringW":        "controls media playback with text commands (play an MP3, open the CD tray)",
	"DwmSetWindowAttribute": "changes window effects such as a dark title bar",
	"SetupDiGetClassDevsW":  "lists devices of a type, such as all USB serial ports",
	"GetLastError":          "gives the error code of the last failed call",
	"GetCurrentProcessId":   "gives this program's process ID",
	"OpenProcess":           "opens another running program by its ID",
	"TerminateProcess":      "ends another program",
	"RegOpenKeyExW":         "opens a registry key", "RegQueryValueExW": "reads a registry value",
	"GetWindowRect": "reads a window's position and size", "MoveWindow": "moves and resizes a window",
	"SetTimer": "asks Windows to send timer messages at an interval",
}

func winFuncWhy(name string) string {
	switch {
	case strings.HasSuffix(name, "W"):
		return " The W at the end means the 'wide' version that uses UTF-16 text, so every language and emoji works."
	case strings.HasSuffix(name, "A"):
		return " The A at the end means the old 'ANSI' text version; the W version is usually better."
	}
	return ""
}

func init() {
	R("go", `^({I})?\s*:?=?\s*(?:syscall|windows)\.(NewLazyDLL|NewLazySystemDLL|MustLoadDLL|LoadDLL)\(({S})\)$`, "var", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("windows-go")
		dll := ""
		if len(x.strs) > 0 {
			dll = strings.ToLower(x.strs[len(x.strs)-1])
		}
		w := "Prepares the Windows DLL " + code(dll)
		if d, ok := winDLLs[dll]; ok {
			w += " (" + d + ")"
		}
		if m[1] != "" {
			w += " and keeps it in " + code(m[1])
		}
		why := "A DLL is a library of ready-made Windows functions. "
		switch m[2] {
		case "NewLazyDLL":
			why += "Lazy means it's only loaded the first time one of its functions is used, so a missing DLL doesn't stop the program starting. This is plain standard-library Go: no cgo or C compiler needed."
		case "NewLazySystemDLL":
			why += "It is only ever loaded from the Windows system folder, which stops a fake DLL next to the program being used instead."
		default:
			why += "It's loaded straight away; Must… versions panic if it's missing."
		}
		return w + ".", why
	}).setsVar(1).first()
	R("go", `^({I})?\s*:?=?\s*({D})\.(?:NewProc|MustFindProc|FindProc)\(({S})\)$`, "var", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("windows-go")
		fn := ""
		if len(x.strs) > 0 {
			fn = x.strs[len(x.strs)-1]
		}
		w := "Looks up the Windows function " + code(fn) + " in " + code(m[2])
		if d, ok := winFuncs[fn]; ok {
			w += ": it " + d
		}
		if m[1] != "" {
			w += ". It's kept in " + code(m[1]) + " so the code can call it later with " + code(m[1]+".Call(...)")
		}
		return w + ".", "Each Windows function is found by its exact name inside its DLL." + winFuncWhy(fn) + " Microsoft's documentation for " + fn + " lists the arguments it expects, in order."
	}).setsVar(1).first()
	R("go", `^(?:([\w, ]+?)\s*:?=\s*)?({I})\.Call\((.*)\)$`, "call", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("windows-go")
		args := splitTop(m[3], ',')
		w := "Calls the Windows function in " + code(m[2])
		if len(args) > 0 && strings.TrimSpace(m[3]) != "" {
			w += " with " + plural(len(args), "argument")
		}
		if m[1] != "" {
			w += ", keeping the results in " + code(m[1])
		}
		w += "."
		why := "Every argument is passed as a uintptr (a plain number); pointers to text or structures are converted with uintptr(unsafe.Pointer(...)). Call gives back the result, a second result and the Windows error, which is only meaningful if the result says the call failed."
		if strings.Contains(m[3], "unsafe.Pointer") {
			x.tag("memory")
		}
		return w, why
	}).first()
}
