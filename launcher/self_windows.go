//go:build windows

package launcher

// Installing on Windows: for the current user only, so no administrator
// rights are needed. The program goes in %LOCALAPPDATA%\Programs\Tinkerie,
// shortcuts in the Start menu (a Tinkerie folder with the launcher and every
// app) and, if wanted, on the desktop, and an entry in Settings > Apps that
// runs "Tinkerie.exe --uninstall".

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

const uninstallKey = `HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\Tinkerie`

func installSupported() (bool, string) {
	if os.Getenv("LOCALAPPDATA") == "" || os.Getenv("APPDATA") == "" {
		return false, "Windows didn't say where programs go for this user (LOCALAPPDATA)."
	}
	return true, ""
}

func installedExe() string {
	return filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Tinkerie", "Tinkerie.exe")
}

func startMenuDir() string {
	return filepath.Join(os.Getenv("APPDATA"), `Microsoft\Windows\Start Menu\Programs\Tinkerie`)
}

func hidden(name string, args ...string) *exec.Cmd {
	c := exec.Command(name, args...)
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return c
}

func reg(args ...string) error { return hidden("reg", args...).Run() }

// psQuote makes s a PowerShell single-quoted string.
func psQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

func powershell(script string) error {
	u := utf16.Encode([]rune(script))
	b := make([]byte, 2*len(u))
	for i, c := range u {
		binary.LittleEndian.PutUint16(b[2*i:], c)
	}
	out, err := hidden("powershell", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass",
		"-EncodedCommand", base64.StdEncoding.EncodeToString(b)).CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if len(msg) > 300 {
			msg = msg[:300] + "…"
		}
		return fmt.Errorf("%v %s", err, msg)
	}
	return nil
}

// register makes the shortcuts and the Settings > Apps entry for the installed
// exe. It returns a note if something didn't work (the program still does).
func register(exe string, m installMark) string {
	dir := filepath.Dir(exe)
	var notes []string

	// One .ico per app that has a PNG icon; the others use Tinkerie's own.
	icons := map[string]string{}
	for _, a := range Apps {
		if !strings.HasSuffix(a.Icon, ".png") {
			continue
		}
		b, err := fs.ReadFile(webFS, "web/icons/"+a.Icon)
		if ico := pngToICO(b); err == nil && ico != nil {
			p := filepath.Join(dir, "icons", a.Name+".ico")
			if os.WriteFile(p, ico, 0o644) == nil {
				icons[a.Name] = p
			}
		}
	}

	// Shortcuts through Windows' own scripting object.
	menu := startMenuDir()
	var sb strings.Builder
	keep := []string{psQuote("Tinkerie.lnk")}
	sb.WriteString("$ErrorActionPreference='Stop'\n$s=New-Object -ComObject WScript.Shell\n")
	sb.WriteString("New-Item -ItemType Directory -Force -Path " + psQuote(menu) + " | Out-Null\n")
	link := func(where, args, icon, about string) {
		fmt.Fprintf(&sb, "$l=$s.CreateShortcut(%s)\n$l.TargetPath=%s\n$l.Arguments=%s\n$l.WorkingDirectory=%s\n$l.IconLocation=%s\n$l.Description=%s\n$l.Save()\n",
			where, psQuote(exe), psQuote(args), psQuote(dir), psQuote(icon), psQuote(about))
	}
	link(psQuote(filepath.Join(menu, "Tinkerie.lnk")), "", exe+",0", "Free tools for makers by nrmluvdev")
	for _, a := range Apps {
		if a.Off {
			continue
		}
		icon := exe + ",0"
		if p, ok := icons[a.Name]; ok {
			icon = p + ",0"
		}
		link(psQuote(filepath.Join(menu, a.Title+".lnk")), "--app "+a.Name, icon, a.About)
		keep = append(keep, psQuote(a.Title+".lnk"))
	}
	// Only once the new shortcuts are made: remove ones for apps that are gone or renamed.
	sb.WriteString("Get-ChildItem -LiteralPath " + psQuote(menu) + " -Filter *.lnk | Where-Object { @(" + strings.Join(keep, ",") +
		") -notcontains $_.Name } | Remove-Item -Force\n")
	desk := "(Join-Path ([Environment]::GetFolderPath('Desktop')) 'Tinkerie.lnk')"
	if m.Desktop {
		link(desk, "", exe+",0", "Free tools for makers by nrmluvdev")
	} else {
		sb.WriteString("Remove-Item -Force -ErrorAction SilentlyContinue -LiteralPath " + desk + "\n")
	}
	if err := powershell(sb.String()); err != nil {
		notes = append(notes, "the shortcuts couldn't be made ("+err.Error()+"); start Tinkerie from "+dir)
	}

	// Settings > Apps.
	size := int64(0)
	if st, err := os.Stat(exe); err == nil {
		size = st.Size() / 1024
	}
	q := `"` + exe + `"`
	for _, v := range [][3]string{
		{"DisplayName", "REG_SZ", "Tinkerie"},
		{"DisplayVersion", "REG_SZ", Version},
		{"Publisher", "REG_SZ", "nrmluvdev"},
		{"DisplayIcon", "REG_SZ", exe + ",0"},
		{"InstallLocation", "REG_SZ", dir},
		{"UninstallString", "REG_SZ", q + " --uninstall"},
		{"QuietUninstallString", "REG_SZ", q + " --uninstall --quiet"},
		{"URLInfoAbout", "REG_SZ", "https://tinkerie.net/"},
		{"HelpLink", "REG_SZ", "https://github.com/normanlove/tinkerie/issues"},
		{"NoModify", "REG_DWORD", "1"},
		{"NoRepair", "REG_DWORD", "1"},
		{"EstimatedSize", "REG_DWORD", fmt.Sprint(size)},
	} {
		if err := reg("add", uninstallKey, "/v", v[0], "/t", v[1], "/d", v[2], "/f"); err != nil {
			notes = append(notes, "Tinkerie couldn't add itself to Settings ▸ Apps")
			break
		}
	}
	return strings.Join(notes, "; ")
}

func uninstall() error {
	exe := installedExe()
	dir := filepath.Dir(exe)
	desk := "(Join-Path ([Environment]::GetFolderPath('Desktop')) 'Tinkerie.lnk')"
	powershell("Remove-Item -Force -ErrorAction SilentlyContinue -LiteralPath " + desk)
	os.RemoveAll(startMenuDir())
	reg("delete", uninstallKey, "/f")
	unregisterMakie(exe)

	// The program can't delete itself while it runs: a hidden cmd waits a
	// moment and removes the folder once this one (and the apps) have quit.
	cmd := exec.Command(os.Getenv("ComSpec"))
	if cmd.Path == "" {
		cmd = exec.Command("cmd.exe")
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true,
		CmdLine: `/d /c ping -n 4 127.0.0.1 >nul & rmdir /s /q "` + dir + `"`}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("couldn't remove %s: %v", dir, err)
	}
	return nil
}

// unregisterMakie removes the "Open with" entries Makie made for itself
// (apps/makie/assoc_windows.go), when they point at the installed copy.
func unregisterMakie(exe string) {
	const cls = `HKCU\Software\Classes\`
	out, err := hidden("reg", "query", cls+`Makie.Project\shell\open\command`, "/ve").Output()
	if err != nil || !strings.Contains(strings.ToLower(string(out)), strings.ToLower(exe)) {
		return
	}
	for _, id := range []string{"Makie.WirieBoard", "Makie.Project", "Makie.Model"} {
		reg("delete", cls+id, "/f")
	}
	for ext, id := range map[string]string{".json": "Makie.WirieBoard", ".makie": "Makie.Project", ".stl": "Makie.Model", ".obj": "Makie.Model"} {
		reg("delete", cls+ext+`\OpenWithProgids`, "/v", id, "/f")
	}
	if out, err := hidden("reg", "query", cls+".makie", "/ve").Output(); err == nil && strings.Contains(string(out), "Makie.Project") {
		reg("delete", cls+".makie", "/ve", "/f")
	}
	reg("delete", cls+`Applications\`+filepath.Base(exe), "/f")
	if d, err := os.UserConfigDir(); err == nil {
		os.Remove(filepath.Join(d, "Makie", "filetypes.txt")) // so a later copy registers again
	}
}

// replaceExe puts next in place of exe. Windows won't overwrite a program
// that's running, but it will rename it: the old one becomes exe.old and is
// deleted the next time Tinkerie starts.
func replaceExe(exe, next string) error {
	old := exe + ".old"
	os.Remove(old)
	if _, err := os.Stat(old); err == nil {
		// Still running from an earlier update: give it a name of its own.
		for i := 2; i < 100; i++ {
			if alt := fmt.Sprintf("%s.old%d", exe, i); os.Remove(alt) == nil || !exists(alt) {
				old = alt
				break
			}
		}
	}
	if err := os.Rename(exe, old); err != nil {
		return err
	}
	if err := os.Rename(next, exe); err != nil {
		os.Rename(old, exe)
		return err
	}
	return nil
}

func exists(p string) bool { _, err := os.Stat(p); return err == nil }

func openURL(u string) {
	hidden("rundll32", "url.dll,FileProtocolHandler", u).Start()
}

var (
	user32      = syscall.NewLazyDLL("user32.dll")
	messageBoxW = user32.NewProc("MessageBoxW")
)

func messageBox(text string, flags uintptr) uintptr {
	t, _ := syscall.UTF16PtrFromString(text)
	c, _ := syscall.UTF16PtrFromString("Tinkerie")
	r, _, _ := messageBoxW.Call(0, uintptr(unsafe.Pointer(t)), uintptr(unsafe.Pointer(c)), flags)
	return r
}

var quiet bool // --uninstall --quiet: no dialogs

func tell(msg string) {
	if !quiet {
		messageBox(msg, 0x40) // MB_ICONINFORMATION
	}
}

func confirm(msg string) bool {
	if quiet {
		return true
	}
	return messageBox(msg, 0x4|0x20) == 6 // MB_YESNO|MB_ICONQUESTION -> IDYES
}

// SetQuiet turns the dialogs off (for "--uninstall --quiet").
func SetQuiet() { quiet = true }
