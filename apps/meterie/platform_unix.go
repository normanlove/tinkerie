//go:build linux || darwin

package meterie

// platform_unix.go: Meterie's platform layer on Linux, Raspberry Pi OS and
// macOS. Serial meters use shared/serialport; sound and the clipboard go
// through the window (web_front.go); file dialogs use the system's own
// (osascript on macOS, zenity or kdialog on Linux).

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
	"tinkerie/shared/droid"

	"tinkerie/shared/serialport"
)


type unixSerial struct{ p *serialport.Port }

// Read waits up to 150 ms for data, like the Windows port, so the reader loop
// can notice when it should stop.
func (s unixSerial) Read(b []byte) (int, error)  { return s.p.ReadTimeout(b, 150*time.Millisecond) }
func (s unixSerial) Write(b []byte) (int, error) { return s.p.Write(b) }
func (s unixSerial) Close() error                { return s.p.Close() }

func platformOpenSerial(path string, cfg SerialCfg) (Port, error) {
	p, err := serialport.Open(path, serialport.Config{Baud: cfg.Baud, DataBits: cfg.DataBits, Parity: cfg.Parity,
		StopBits: float64(cfg.StopBits), DTR: cfg.DTR, RTS: cfg.RTS})
	if err != nil {
		return nil, err
	}
	return unixSerial{p}, nil
}

func platformListPorts() []PortInfo {
	if previewPorts != nil { // the preview renderer's pretend ports
		return previewPorts
	}
	out := listHID()
	for _, in := range serialport.List() {
		name := strings.TrimSpace(in.Manufacturer + " " + in.Product)
		if name == "" {
			name = in.Description
		}
		if name == "" {
			name = in.Name
		}
		out = append(out, PortInfo{Kind: TrSerial, Path: in.Path, Name: name, VID: in.VID, PID: in.PID})
	}
	return out
}

func platformPlaySound(wav []byte, loop bool) {
	if front != nil {
		front.playSound(wav, loop)
	}
}

func platformStopSound() {
	if front != nil {
		front.stopSound()
	}
}

func platformReveal(path string) {
	if droid.Active() {
		droid.ShowFolder(path)
		return
	}
	if runtime.GOOS == "darwin" {
		_ = exec.Command("open", "-R", path).Start()
		return
	}
	_ = exec.Command("xdg-open", filepath.Dir(path)).Start()
}

var clip string

func platformClipboardSet(s string) {
	clip = s
	if front != nil {
		front.setClipboard(s)
	}
}

func platformClipboardGet() string { return clip }

// ------------------------------------------------------------- file dialogs

var patRe = regexp.MustCompile(`\*\.[A-Za-z0-9.]+`)

func platformSaveDialog(title, def, filter, ext string) (string, bool) {
	if droid.Active() { // saves go to Files › Tinkerie › Meterie
		if ext != "" && !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		if ext != "" && !strings.HasSuffix(strings.ToLower(def), strings.ToLower(ext)) {
			def += ext
		}
		return droid.SavePath("Meterie", def), true
	}
	dir := defaultDialogDir()
	if strings.ContainsRune(def, '/') { // a full suggested path
		dir, def = filepath.Dir(def), filepath.Base(def)
	}
	if ext != "" && !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	if runtime.GOOS == "darwin" {
		out, err := exec.Command("osascript", "-e",
			`POSIX path of (choose file name with prompt `+asQuote(title)+` default name `+asQuote(def)+` default location (POSIX file `+asQuote(dir)+`))`).Output()
		return dialogResult(out, err, ext)
	}
	if p, err := exec.LookPath("zenity"); err == nil {
		out, err := exec.Command(p, "--file-selection", "--save", "--confirm-overwrite", "--title="+title, "--filename="+filepath.Join(dir, def)).Output()
		return dialogResult(out, err, ext)
	}
	if p, err := exec.LookPath("kdialog"); err == nil {
		out, err := exec.Command(p, "--getsavefilename", filepath.Join(dir, def), strings.Join(patRe.FindAllString(filter, -1), " ")).Output()
		return dialogResult(out, err, ext)
	}
	// No dialog program: save under the suggested name and folder.
	os.MkdirAll(dir, 0o755)
	return filepath.Join(dir, def), true
}

func platformOpenDialog(title, filter, pattern string) (string, bool) {
	if droid.Active() {
		p, err := droid.PickFile(title, patRe.FindAllString(pattern+" "+filter, -1))
		return p, err == nil
	}
	dir := defaultDialogDir()
	if runtime.GOOS == "darwin" {
		out, err := exec.Command("osascript", "-e",
			`POSIX path of (choose file with prompt `+asQuote(title)+` default location (POSIX file `+asQuote(dir)+`))`).Output()
		return dialogResult(out, err, "")
	}
	pats := patRe.FindAllString(pattern+" "+filter, -1)
	if p, err := exec.LookPath("zenity"); err == nil {
		args := []string{"--file-selection", "--title=" + title, "--filename=" + dir + "/"}
		if len(pats) > 0 {
			args = append(args, "--file-filter="+strings.Join(pats, " "), "--file-filter=All files | *")
		}
		out, err := exec.Command(p, args...).Output()
		return dialogResult(out, err, "")
	}
	if p, err := exec.LookPath("kdialog"); err == nil {
		out, err := exec.Command(p, "--getopenfilename", dir, strings.Join(pats, " ")).Output()
		return dialogResult(out, err, "")
	}
	return "", false
}

func defaultDialogDir() string {
	h, _ := os.UserHomeDir()
	d := filepath.Join(h, "Documents", "Meterie")
	if _, err := os.Stat(d); err != nil {
		d = filepath.Join(h, "Documents")
	}
	return d
}

func dialogResult(out []byte, err error, ext string) (string, bool) {
	p := strings.TrimSpace(string(out))
	if err != nil || p == "" {
		return "", false
	}
	if ext != "" && !strings.HasSuffix(strings.ToLower(p), strings.ToLower(ext)) && filepath.Ext(p) == "" {
		p += ext
	}
	return p, true
}

// asQuote makes an AppleScript string literal.
func asQuote(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

func platformOffscreen(w, h int) (*Canvas, TextEngine, func()) {
	c := NewCanvas(w, h)
	if front != nil {
		return c, front.text.forCanvas(c), func() {}
	}
	return c, &previewText{}, func() {}
}
