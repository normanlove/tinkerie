package appwindow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMaximised(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "Default"), 0o755)
	p := filepath.Join(dir, "Default", "Preferences")
	os.WriteFile(p, []byte(`{"browser":{"window_placement":{"maximized":false,"left":10},"app_window_placement":{"_crx_x":{"maximized":false}}},"other":1}`), 0o600)
	args := Maximised(dir)
	if args[0] != "--start-maximized" {
		t.Fatal(args)
	}
	b, _ := os.ReadFile(p)
	if strings.Contains(string(b), `"maximized":false`) || !strings.Contains(string(b), `"other":1`) {
		t.Fatalf("%s", b)
	}
	Maximised(filepath.Join(dir, "missing")) // no profile yet: nothing to do
}
