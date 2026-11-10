package main

import (
	"reflect"
	"testing"
	"tinkerie/launcher"
)

func TestPickApp(t *testing.T) {
	cases := []struct {
		args []string
		app  string
		rest []string
	}{
		{[]string{`C:\Tinkerie.exe`}, "", []string{}},
		{[]string{`C:\Tinkerie.exe`, "--app", "makie", "part.makie"}, "makie", []string{"part.makie"}},
		{[]string{"tinkerie", "--app=Scopie"}, "scopie", []string{}},
		{[]string{"tinkerie", "-app", "tunie", "--from", "makie"}, "tunie", []string{"--from", "makie"}},
		{[]string{`C:\Tools\Meterie.exe`, "session.json"}, "meterie", []string{"session.json"}},
		{[]string{"/usr/local/bin/wirie"}, "wirie", []string{}},
		{[]string{"tinkerie", "--app"}, "", []string{"--app"}},
	}
	for _, c := range cases {
		app, rest := pickApp(c.args)
		if app != c.app || !reflect.DeepEqual(rest, c.rest) {
			t.Errorf("pickApp(%q) = %q %q, want %q %q", c.args, app, rest, c.app, c.rest)
		}
	}
}

func TestEveryLauncherAppRuns(t *testing.T) {
	for _, n := range launcher.Names() {
		if mains[n] == nil {
			t.Errorf("launcher tile %q has no app", n)
		}
	}
	// old names kept so earlier shortcuts still work
	aliases := map[string]bool{"tracie": true}
	n := 0
	for k := range mains {
		if !aliases[k] {
			n++
		}
	}
	if n != len(launcher.Names()) {
		t.Errorf("%d apps but %d launcher tiles", n, len(launcher.Names()))
	}
}
