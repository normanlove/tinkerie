//go:build windows

// Tunie — platform_windows.go: Windows-only pieces (app window, folders,
// SD-card drives, COM ports, error box). Uses only the Go standard library
// and DLLs that ship with every copy of Windows.
// Part of the nrmluvdev maker suite. Created by nrmluvdev.
package tunie

import (
	"tinkerie/shared/appwindow"

	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"syscall"
	"unsafe"
)

var (
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	user32               = syscall.NewLazyDLL("user32.dll")
	procGetLogicalDrives = kernel32.NewProc("GetLogicalDrives")
	procGetDriveTypeW    = kernel32.NewProc("GetDriveTypeW")
	procGetVolumeInfoW   = kernel32.NewProc("GetVolumeInformationW")
	procQueryDosDeviceW  = kernel32.NewProc("QueryDosDeviceW")
	procMessageBoxW      = user32.NewProc("MessageBoxW")
	procGetCommState     = kernel32.NewProc("GetCommState")
	procSetCommState     = kernel32.NewProc("SetCommState")
	procSetCommTimeouts  = kernel32.NewProc("SetCommTimeouts")
	procPurgeComm        = kernel32.NewProc("PurgeComm")
)

const createNoWindow = 0x08000000

func osName() string { return "windows" }

func fatal(msg string) {
	t, _ := syscall.UTF16PtrFromString(msg)
	c, _ := syscall.UTF16PtrFromString("Tunie")
	procMessageBoxW.Call(0, uintptr(unsafe.Pointer(t)), uintptr(unsafe.Pointer(c)), 0x10)
	os.Exit(1)
}

func findBrowser() string {
	var cands []string
	for _, env := range []string{"ProgramFiles(x86)", "ProgramFiles", "LOCALAPPDATA"} {
		base := os.Getenv(env)
		if base == "" {
			continue
		}
		cands = append(cands,
			filepath.Join(base, "Microsoft", "Edge", "Application", "msedge.exe"),
			filepath.Join(base, "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(base, "BraveSoftware", "Brave-Browser", "Application", "brave.exe"),
		)
	}
	for _, c := range cands {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c
		}
	}
	return ""
}

// openAppWindow shows the interface in a frameless "app mode" browser window
// with its own profile, so it looks and behaves like a normal desktop app.
func openAppWindow(url, dataDir string) error {
	if b := findBrowser(); b != "" {
		cmd := exec.Command(b, append(appwindow.Maximised(filepath.Join(dataDir, "window")),
			"--app="+url,
			"--user-data-dir="+filepath.Join(dataDir, "window"),
			"--no-first-run",
			"--no-default-browser-check",
			"--disable-features=Translate,msEdgeSidebarV2,msHubApps",
		)...)
		raise := appwindow.WatchNewWindow() // so the window opens in front of the launcher
		if err := cmd.Start(); err == nil {
			raise()
			go cmd.Wait()
			return nil
		}
	}
	return openURL(url)
}

func openURL(u string) error {
	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", u)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNoWindow}
	return cmd.Start()
}

func openFolder(p string) error {
	if st, err := os.Stat(p); err == nil && !st.IsDir() {
		return exec.Command("explorer", "/select,", p).Start()
	}
	if err := os.MkdirAll(p, 0o755); err != nil {
		return err
	}
	return exec.Command("explorer", p).Start()
}

type Drive struct {
	Path      string `json:"path"`
	Label     string `json:"label"`
	Removable bool   `json:"removable"`
}

func listDrives() []Drive {
	mask, _, _ := procGetLogicalDrives.Call()
	var out []Drive
	for i := 0; i < 26; i++ {
		if mask&(1<<uint(i)) == 0 {
			continue
		}
		root := string(rune('A'+i)) + `:\`
		p, _ := syscall.UTF16PtrFromString(root)
		t, _, _ := procGetDriveTypeW.Call(uintptr(unsafe.Pointer(p)))
		if t != 2 { // DRIVE_REMOVABLE
			continue
		}
		name := make([]uint16, 261)
		r, _, _ := procGetVolumeInfoW.Call(uintptr(unsafe.Pointer(p)), uintptr(unsafe.Pointer(&name[0])), 261, 0, 0, 0, 0, 0)
		if r == 0 {
			continue // no card inserted
		}
		out = append(out, Drive{Path: root, Label: syscall.UTF16ToString(name), Removable: true})
	}
	return out
}

var reCom = regexp.MustCompile(`^COM(\d+)$`)

func listSerialPorts() []string {
	buf := make([]uint16, 65536)
	n, _, _ := procQueryDosDeviceW.Call(0, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	var ports []string
	start := 0
	for i := 0; i < int(n); i++ {
		if buf[i] == 0 {
			if i > start {
				s := syscall.UTF16ToString(buf[start:i])
				if reCom.MatchString(s) {
					ports = append(ports, s)
				}
			}
			start = i + 1
		}
	}
	sort.Slice(ports, func(a, b int) bool {
		na, _ := strconv.Atoi(reCom.FindStringSubmatch(ports[a])[1])
		nb, _ := strconv.Atoi(reCom.FindStringSubmatch(ports[b])[1])
		return na < nb
	})
	return ports
}

// ---------------------------------------------------------------- serial port

type dcb struct {
	DCBlength  uint32
	BaudRate   uint32
	Flags      uint32
	wReserved  uint16
	XonLim     uint16
	XoffLim    uint16
	ByteSize   byte
	Parity     byte
	StopBits   byte
	XonChar    byte
	XoffChar   byte
	ErrorChar  byte
	EofChar    byte
	EvtChar    byte
	wReserved1 uint16
}

type commTimeouts struct {
	ReadIntervalTimeout         uint32
	ReadTotalTimeoutMultiplier  uint32
	ReadTotalTimeoutConstant    uint32
	WriteTotalTimeoutMultiplier uint32
	WriteTotalTimeoutConstant   uint32
}

type winSerial struct{ h syscall.Handle }

func openSerial(name string, baud int) (serialPort, error) {
	p, err := syscall.UTF16PtrFromString(`\\.\` + name)
	if err != nil {
		return nil, err
	}
	h, err := syscall.CreateFile(p, syscall.GENERIC_READ|syscall.GENERIC_WRITE, 0, nil, syscall.OPEN_EXISTING, 0, 0)
	if err != nil {
		if errors.Is(err, syscall.ERROR_ACCESS_DENIED) {
			return nil, errors.New(name + " is in use by another program (close Cura, Pronterface, OctoPrint, Arduino…)")
		}
		return nil, errors.New("couldn't open " + name + ": " + err.Error())
	}
	var d dcb
	d.DCBlength = uint32(unsafe.Sizeof(d))
	if r, _, e := procGetCommState.Call(uintptr(h), uintptr(unsafe.Pointer(&d))); r == 0 {
		syscall.CloseHandle(h)
		return nil, e
	}
	d.BaudRate = uint32(baud)
	d.ByteSize = 8
	d.Parity = 0
	d.StopBits = 0
	d.Flags = 0x0001 | (1 << 4) | (1 << 12) // binary, DTR on, RTS on
	if r, _, e := procSetCommState.Call(uintptr(h), uintptr(unsafe.Pointer(&d))); r == 0 {
		syscall.CloseHandle(h)
		return nil, e
	}
	t := commTimeouts{ReadIntervalTimeout: 0xFFFFFFFF, ReadTotalTimeoutMultiplier: 0xFFFFFFFF, ReadTotalTimeoutConstant: 200, WriteTotalTimeoutConstant: 3000}
	if r, _, e := procSetCommTimeouts.Call(uintptr(h), uintptr(unsafe.Pointer(&t))); r == 0 {
		syscall.CloseHandle(h)
		return nil, e
	}
	procPurgeComm.Call(uintptr(h), 0x0008|0x0004)
	return &winSerial{h: h}, nil
}

func (s *winSerial) Read(b []byte) (int, error) {
	var n uint32
	err := syscall.ReadFile(s.h, b, &n, nil)
	return int(n), err
}

func (s *winSerial) Write(b []byte) (int, error) {
	var n uint32
	err := syscall.WriteFile(s.h, b, &n, nil)
	return int(n), err
}

func (s *winSerial) Close() error { return syscall.CloseHandle(s.h) }
