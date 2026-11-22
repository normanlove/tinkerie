//go:build !windows

package launcher

// Installing on Linux and Raspberry Pi OS: for the current user only. The
// program goes in ~/.local/share/tinkerie (or $XDG_DATA_HOME/tinkerie), with a
// menu entry for the launcher and one for each app in ~/.local/share/applications.
// On a Mac, Tinkerie.app is installed by dragging it into Applications, and on
// Android by the .apk, so the launcher doesn't offer it there.

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"tinkerie/shared/droid"
)

func installSupported() (bool, string) {
	switch {
	case droid.Active() || runtime.GOOS == "android":
		return false, "On Android, Tinkerie is installed from its .apk."
	case runtime.GOOS == "darwin":
		return false, "On a Mac, drag Tinkerie into the Applications folder to install it."
	case runtime.GOOS != "linux":
		return false, "Installing isn't set up for this system yet."
	case dataHome() == "":
		return false, "Tinkerie couldn't find your home folder."
	}
	return true, ""
}

func dataHome() string {
	if d := os.Getenv("XDG_DATA_HOME"); filepath.IsAbs(d) {
		return d
	}
	if h, err := os.UserHomeDir(); err == nil && h != "" {
		return filepath.Join(h, ".local", "share")
	}
	return ""
}

func installedExe() string { return filepath.Join(dataHome(), "tinkerie", "tinkerie") }

func menuDir() string { return filepath.Join(dataHome(), "applications") }

// The line that marks a menu entry as one Tinkerie made (so only those are removed).
const madeBy = "X-Tinkerie-Installed=true"

// desktopExec quotes a path for a .desktop file's Exec line.
func desktopExec(p string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "`", "\\`", `$`, `\$`, `%`, `%%`)
	return `"` + r.Replace(p) + `"`
}

func oneLine(s string) string { return strings.NewReplacer("\n", " ", "\r", " ").Replace(s) }

func register(exe string, m installMark) string {
	dir := filepath.Dir(exe)
	menu := menuDir()
	if err := os.MkdirAll(menu, 0o755); err != nil {
		return "the menu entries couldn't be made (" + err.Error() + ")"
	}
	want := map[string]string{}
	entry := func(name, comment, args, icon string) string {
		run := desktopExec(exe)
		if args != "" {
			run += " " + args
		}
		return "[Desktop Entry]\nType=Application\nName=" + oneLine(name) + "\nComment=" + oneLine(comment) +
			"\nExec=" + run + "\nIcon=" + icon + "\nTerminal=false\nCategories=Development;Electronics;Engineering;\n" + madeBy + "\n"
	}
	// Same file name as build/install.sh's entry, so this one replaces it.
	want["tinkerie.desktop"] = entry("Tinkerie", "Free maker tools by nrmluvdev: "+strings.Join(titles(), ", "), "", filepath.Join(dir, "tinkerie.png"))
	for _, a := range Apps {
		if a.Off {
			continue
		}
		want["tinkerie-"+a.Name+".desktop"] = entry(a.Title, a.About, "--app "+a.Name, filepath.Join(dir, "icons", a.Icon))
	}
	// Remove entries for apps that are gone or renamed.
	old, _ := filepath.Glob(filepath.Join(menu, "tinkerie-*.desktop"))
	for _, p := range old {
		if _, keep := want[filepath.Base(p)]; !keep && madeByUs(p) {
			os.Remove(p)
		}
	}
	var failed error
	for name, body := range want {
		if err := os.WriteFile(filepath.Join(menu, name), []byte(body), 0o755); err != nil {
			failed = err
		}
	}
	if p, err := exec.LookPath("update-desktop-database"); err == nil {
		exec.Command(p, menu).Run()
	}
	if failed != nil {
		return "some menu entries couldn't be made (" + failed.Error() + ")"
	}
	return ""
}

func titles() []string {
	var t []string
	for _, a := range Apps {
		t = append(t, a.Title)
	}
	return t
}

func madeByUs(p string) bool {
	f, err := os.Open(p)
	if err != nil {
		return false
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if strings.TrimSpace(sc.Text()) == madeBy {
			return true
		}
	}
	return false
}

func uninstall() error {
	menu := menuDir()
	entries, _ := filepath.Glob(filepath.Join(menu, "tinkerie*.desktop"))
	for _, p := range entries {
		if madeByUs(p) {
			os.Remove(p)
		}
	}
	if p, err := exec.LookPath("update-desktop-database"); err == nil {
		exec.Command(p, menu).Run()
	}
	dir := filepath.Dir(installedExe())
	if filepath.Base(dir) != "tinkerie" { // never remove anything but our own folder
		return errors.New("unexpected install folder " + dir)
	}
	// A running program can be deleted on Linux; apps already open keep running.
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("couldn't remove %s: %v", dir, err)
	}
	return nil
}

// replaceExe puts next in place of exe in one step. Programs already running
// keep the old file open, so they carry on until they're closed.
func replaceExe(exe, next string) error {
	if st, err := os.Stat(exe); err == nil {
		os.Chmod(next, st.Mode().Perm()|0o100)
	}
	return os.Rename(next, exe)
}

func openURL(u string) {
	if droid.Active() {
		droid.Browse(u)
		return
	}
	if runtime.GOOS == "darwin" {
		exec.Command("open", u).Start()
		return
	}
	exec.Command("xdg-open", u).Start()
}

func tell(msg string)         { fmt.Println(msg) }
func confirm(msg string) bool { return true } // "tinkerie --uninstall" in a terminal means it

// SetQuiet is for "--uninstall --quiet" (only Windows shows dialogs).
func SetQuiet() {}
