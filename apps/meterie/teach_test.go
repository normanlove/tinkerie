//go:build !windows

package meterie

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// hover moves the mouse over a widget or rect and returns the teaching card's title and text.
func (d *uiDriver) hover(r Rect) (string, string) {
	in := Input{MX: r.CX(), MY: r.CY(), HasMouse: true}
	d.frame(in)
	d.frame(in)
	return d.a.ui.teachTip()
}

func TestTeachingMode(t *testing.T) {
	d := newDriver(t)
	d.a.ui.Teach = true
	d.frame(Input{MX: -1, MY: -1})

	for id, want := range map[string]string{"rail1": "Meters", "graph#auto": "Auto-scale", "stats#win#1": "Statistics window"} {
		r, ok := d.a.ui.Rects[id]
		if !ok {
			t.Fatalf("widget %q not on screen", id)
		}
		if title, text := d.hover(r); title != want || text == "" {
			t.Errorf("hovering %s: got %q / %q, want %q", id, title, text, want)
		}
	}

	// An area that isn't a widget: the big readout, top-left of the Live page.
	if title, _ := d.hover(Rect{d.a.ui.Rects["rail0"].Right() + 60, 140, 10, 10}); title != "Readout" {
		t.Errorf("hovering the readout: got %q, want Readout", title)
	}

	// F2 turns it off and on again.
	d.frame(Input{MX: -1, MY: -1, Keys: []Key{KF2}})
	if d.a.ui.Teach || !d.a.settings.TeachOff {
		t.Errorf("F2 should turn teaching mode off")
	}
	d.frame(Input{MX: -1, MY: -1, Keys: []Key{KF2}})
	if !d.a.ui.Teach {
		t.Errorf("F2 should turn teaching mode back on")
	}
}

// TestTeachingCoverage lists every widget on every page that has neither a
// teaching tip nor a tooltip, so new controls don't go unexplained.
func TestTeachingCoverage(t *testing.T) {
	d := newDriver(t)
	d.a.ui.Teach = true
	var missing []string
	seen := map[string]bool{}
	for p := range pageInfo {
		d.a.page = p
		d.a.ui.Rects = map[string]Rect{}
		d.frame(Input{MX: -1, MY: -1})
		for id, r := range d.a.ui.Rects {
			base := strings.TrimRight(id, "0123456789")
			if seen[base] || strings.HasPrefix(id, "rail") || strings.Contains(id, "#opt") || strings.HasSuffix(id, "#pop") {
				continue
			}
			seen[base] = true
			if title, _ := d.hover(r); title == "" {
				missing = append(missing, pageInfo[p].name+": "+id)
			}
			d.frame(Input{MX: -1, MY: -1})
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Logf("%d widgets without an explanation:\n  %s", len(missing), strings.Join(missing, "\n  "))
	}
}

// TestOldAPIPortMoves checks the suite link leaves 47820, which is Wirie's port.
func TestOldAPIPortMoves(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	if err := os.MkdirAll(configDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir(), "settings.json"), []byte(`{"apiPort":47820,"api":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	a := NewApp()
	if a.settings.APIPort != 47823 {
		t.Errorf("suite link port = %d, want 47823", a.settings.APIPort)
	}
}
