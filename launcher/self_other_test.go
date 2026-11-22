//go:build !windows

package launcher

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestInstallAndUninstallLinux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux only")
	}
	home := t.TempDir()
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, "data"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	src := filepath.Join(t.TempDir(), "tinkerie")
	os.WriteFile(src, []byte("\x7fELF program"), 0o755)
	old := selfExe
	selfExe = src
	defer func() { selfExe = old }()

	in := getSelfInfo()
	if !in.InstallOK || in.Installed || !in.Prompt || in.InstallDir != filepath.Join(home, "data", "tinkerie") {
		t.Fatalf("before installing: %+v", in)
	}
	// An entry made by build/install.sh is replaced; other programs' are left alone.
	menu := filepath.Join(home, "data", "applications")
	os.MkdirAll(menu, 0o755)
	os.WriteFile(filepath.Join(menu, "tinkerie.desktop"), []byte("[Desktop Entry]\nExec=/old/tinkerie\n"), 0o644)
	os.WriteFile(filepath.Join(menu, "tinkerie-something.desktop"), []byte("[Desktop Entry]\nExec=other\n"), 0o644)

	if note, err := install(false); err != nil || note != "" {
		t.Fatalf("install: %v %q", err, note)
	}
	exe := filepath.Join(home, "data", "tinkerie", "tinkerie")
	if b, _ := os.ReadFile(exe); string(b) != "\x7fELF program" {
		t.Fatalf("installed program is %q", b)
	}
	main, _ := os.ReadFile(filepath.Join(menu, "tinkerie.desktop"))
	if !strings.Contains(string(main), `Exec="`+exe+`"`+"\n") || !strings.Contains(string(main), madeBy) {
		t.Errorf("launcher entry:\n%s", main)
	}
	for _, a := range Apps {
		b, err := os.ReadFile(filepath.Join(menu, "tinkerie-"+a.Name+".desktop"))
		if err != nil {
			t.Errorf("no menu entry for %s", a.Name)
			continue
		}
		if !strings.Contains(string(b), `Exec="`+exe+`" --app `+a.Name+"\n") || !strings.Contains(string(b), "Name="+a.Title+"\n") {
			t.Errorf("%s entry:\n%s", a.Name, b)
		}
		icon := filepath.Join(home, "data", "tinkerie", "icons", a.Icon)
		if _, err := os.Stat(icon); err != nil || !strings.Contains(string(b), "Icon="+icon+"\n") {
			t.Errorf("%s icon missing: %v", a.Name, err)
		}
	}
	if m, ok := readMark(filepath.Dir(exe)); !ok || m.Version != Version {
		t.Errorf("mark: %+v", m)
	}

	// The installed copy knows it's installed, and a stale mark makes it redo
	// the menu entries (as after an update).
	selfExe = exe
	if in := getSelfInfo(); !in.Installed || in.Prompt {
		t.Fatalf("after installing: %+v", in)
	}
	writeMark(filepath.Dir(exe), installMark{Version: "0.1.0"})
	os.Remove(filepath.Join(menu, "tinkerie-wirie.desktop"))
	refreshInstall()
	if _, err := os.Stat(filepath.Join(menu, "tinkerie-wirie.desktop")); err != nil {
		t.Error("refreshInstall didn't redo the menu entries")
	}

	// A portable copy sees the installed one.
	selfExe = src
	if in := getSelfInfo(); in.InstalledVersion != Version || in.Prompt {
		t.Errorf("portable copy with one installed: %+v", in)
	}

	if err := uninstall(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Dir(exe)); !os.IsNotExist(err) {
		t.Error("install folder still there")
	}
	left, _ := filepath.Glob(filepath.Join(menu, "*"))
	if len(left) != 1 || filepath.Base(left[0]) != "tinkerie-something.desktop" {
		t.Errorf("menu after uninstall: %v (only the other program's entry should be left)", left)
	}
}

func TestDesktopExecQuoting(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Linux menu entries")
	}
	if got := desktopExec(`/home/a b/$x"100%`); got != `"/home/a b/\$x\"100%%"` {
		t.Errorf("got %s", got)
	}
}
