//go:build windows

package makie

// Lets Windows offer Makie in "Open with" for Wirie boards (.makie.json),
// STL / OBJ models and Makie projects. Only the current user's settings are
// touched, and existing default apps are left alone (Makie only becomes the
// default for .makie, its own project files, when nothing else has it).

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

func reg(args ...string) error {
	c := exec.Command("reg", args...)
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return c.Run()
}

func registerFileTypes() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	d, err := os.UserConfigDir()
	if err != nil {
		return
	}
	mark := filepath.Join(d, appName, "filetypes.txt")
	if b, err := os.ReadFile(mark); err == nil && strings.TrimSpace(string(b)) == exe+"|"+version {
		return // already done for this copy of Makie
	}
	cmd := `"` + exe + `" --app makie "%1"` // Makie runs inside Tinkerie.exe
	icon := `"` + exe + `",0`
	const cls = `HKCU\Software\Classes\`
	for _, p := range []struct{ id, desc string }{{"Makie.WirieBoard", "Wirie board (open in Makie)"}, {"Makie.Project", "Makie project"}, {"Makie.Model", "3D model (open in Makie)"}} {
		_ = reg("add", cls+p.id, "/ve", "/d", p.desc, "/f")
		_ = reg("add", cls+p.id+`\DefaultIcon`, "/ve", "/d", icon, "/f")
		_ = reg("add", cls+p.id+`\shell\open\command`, "/ve", "/d", cmd, "/f")
	}
	for ext, id := range map[string]string{".json": "Makie.WirieBoard", ".makie": "Makie.Project", ".stl": "Makie.Model", ".obj": "Makie.Model"} {
		_ = reg("add", cls+ext+`\OpenWithProgids`, "/v", id, "/t", "REG_NONE", "/f")
	}
	if reg("query", cls+".makie", "/ve") != nil {
		_ = reg("add", cls+".makie", "/ve", "/d", "Makie.Project", "/f")
	}
	app := cls + `Applications\` + filepath.Base(exe)
	_ = reg("add", app, "/v", "FriendlyAppName", "/d", appName, "/f")
	_ = reg("add", app+`\shell\open\command`, "/ve", "/d", cmd, "/f")
	for _, ext := range []string{".json", ".makie", ".stl", ".obj"} {
		_ = reg("add", app+`\SupportedTypes`, "/v", ext, "/d", "", "/f")
	}
	_ = os.MkdirAll(filepath.Dir(mark), 0o755)
	_ = os.WriteFile(mark, []byte(exe+"|"+version), 0o644)
}
