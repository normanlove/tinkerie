// Codie tests: every example must be recognised as the right language and
// (nearly) every line must get a real explanation. Created by nrmluvdev.
package codie

import (
	"os"
	"strings"
	"testing"
)

func allExamples() []Sample {
	s := Samples()
	for _, l := range lessons {
		for _, e := range l.Examples {
			s = append(s, Sample{"lesson " + l.ID, e.Label, e.Lang, e.Code})
		}
	}
	return s
}

func TestExamplesDetectAndExplain(t *testing.T) {
	verbose := os.Getenv("CODIE_DUMP") != ""
	for _, s := range allExamples() {
		r := Analyse(s.Code, "")
		if r.Lang.ID != s.Lang {
			t.Errorf("%s: detected %s, want %s (others %v)", s.Name, r.Lang.ID, s.Lang, r.Lang.Others)
		}
		r = Analyse(s.Code, s.Lang)
		unknown := 0
		for _, ln := range r.Lines {
			if !ln.Known {
				unknown++
				if verbose {
					t.Logf("%s line %d: %q", s.Name, ln.N, strings.TrimSpace(ln.Text))
				}
			}
		}
		if code := r.Stats.Code; code > 0 && unknown*10 > code {
			t.Errorf("%s: %d of %d code lines unexplained", s.Name, unknown, code)
		}
	}
}

func TestBlocksBalance(t *testing.T) {
	for _, s := range allExamples() {
		r := Analyse(s.Code, s.Lang)
		for _, o := range r.Outline {
			if o.End < o.Start {
				t.Errorf("%s: %s %q ends (%d) before it starts (%d)", s.Name, o.Kind, o.Name, o.End, o.Start)
			}
		}
	}
}

func TestWindowsGo(t *testing.T) {
	var code string
	for _, s := range Samples() {
		if strings.Contains(s.Name, "message box") {
			code = s.Code
		}
	}
	r := Analyse(code, "")
	if r.Lang.Dialect != "Windows Go" {
		t.Fatalf("dialect %q", r.Lang.Dialect)
	}
	want := map[string]string{"//go:build": "only compiled", "NewProc(\"MessageBoxW": "message box", "procMessageBoxW.Call": "Windows function", "NewLazyDLL(\"user32": "DLL"}
	for _, ln := range r.Lines {
		for k, w := range want {
			if !strings.Contains(ln.Text, k) {
				continue
			}
			txt := ln.Parts[0].What + ln.Parts[0].Why
			if !strings.Contains(txt, w) {
				t.Errorf("line %d: %q lacks %q", ln.N, txt, w)
			}
		}
	}
}

func TestGameEngines(t *testing.T) {
	cases := map[string]string{"Unity: player controller (C#)": "unity", "Godot: coin pickup (GDScript)": "godot", "PICO-8: collect the stars": "pico8", "Roblox: kill brick and coins (Luau)": "roblox", "Commodore 64: sprite and joystick": "c64", "ZX Spectrum: dodge game": "zx", "BBC Micro: move a dot": "bbc", "Unreal: rotating pickup (C++)": "unreal"}
	for _, s := range Samples() {
		want, ok := cases[s.Name]
		if !ok {
			continue
		}
		r := Analyse(s.Code, "")
		if r.Game.Engine == nil || r.Game.Engine.ID != want {
			got := "none"
			if r.Game.Engine != nil {
				got = r.Game.Engine.ID
			}
			t.Errorf("%s: engine %s, want %s", s.Name, got, want)
		}
		if !r.Game.IsGame {
			t.Errorf("%s: not seen as a game", s.Name)
		}
	}
}

func TestShow(t *testing.T) {
	name := os.Getenv("CODIE_SHOW")
	if name == "" {
		t.Skip()
	}
	for _, s := range allExamples() {
		if !strings.Contains(s.Name, name) {
			continue
		}
		r := Analyse(s.Code, "")
		t.Logf("== %s: %s %s %.2f | %v", s.Name, r.Lang.ID, r.Lang.Dialect, r.Lang.Confidence, r.Summary)
		for _, ln := range r.Lines {
			for _, p := range ln.Parts {
				t.Logf("%3d %s%-8s %s | %s", ln.N, strings.Repeat(" ", ln.Depth), p.Kind, p.What, p.Why)
			}
			for _, n := range ln.Notes {
				t.Logf("      note: %s", n)
			}
		}
		for _, o := range r.Outline {
			t.Logf("outline %s %s %d-%d d%d", o.Kind, o.Name, o.Start, o.End, o.Depth)
		}
		for _, w := range r.Warnings {
			t.Logf("warn %d %s", w.Line, w.Text)
		}
		if r.Game.Engine != nil {
			t.Logf("engine %s frame %+v", r.Game.Engine.Name, r.Game.Engine.Frame)
		}
		for _, h := range r.Game.Hits {
			t.Logf("game %s %v", h.ID, h.Lines)
		}
		break
	}
}

// Any code read as any language must never crash Codie.
func TestNoPanicsAcrossLanguages(t *testing.T) {
	odd := []string{"", "\n\n", "}}}}", "end end end", "((((", "\"unfinished", "/* never closed", "'''\nnot closed", "NEXT\nWEND\nfi", "\x00\x01\x02", strings.Repeat("if (x) {\n", 300)}
	codes := odd
	for _, s := range allExamples() {
		codes = append(codes, s.Code)
	}
	for _, l := range LangList() {
		for _, c := range codes {
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("%s panicked on %q: %v", l.ID, clip(c, 40), r)
					}
				}()
				Analyse(c, l.ID)
			}()
		}
	}
	for _, c := range odd {
		Analyse(c, "")
	}
}
