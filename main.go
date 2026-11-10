// Tinkerie: the nrmluvdev maker tool suite in one program.
//
//	Tinkerie                      opens the launcher
//	Tinkerie --app makie [args]   runs one app (the launcher starts apps this way)
//
// A copy or link of the program named after an app (makie.exe, ./wirie) runs that
// app directly. Each app runs in its own process, so several can be open at once.
//
// Designed and created by nrmluvdev.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tinkerie/apps/codie"
	"tinkerie/apps/flashie"
	"tinkerie/apps/makie"
	"tinkerie/apps/meterie"
	"tinkerie/apps/scopie"
	"tinkerie/apps/tracie"
	"tinkerie/apps/tunie"
	"tinkerie/apps/wirie"
	"tinkerie/launcher"
)

var mains = map[string]func(){
	"wirie":   wirie.Main,
	"scopie":  scopie.Main,
	"makie":   makie.Main,
	"flashie": flashie.Main,
	"tunie":   tunie.Main,
	"tracie":  tracie.Main,
	"meterie": meterie.Main,
	"codie":   codie.Main,
}

func main() {
	name, rest := pickApp(os.Args)
	if name == "" {
		launcher.Run()
		return
	}
	run, ok := mains[name]
	if !ok {
		fmt.Fprintf(os.Stderr, "Tinkerie: unknown app %q (try one of: %s)\n", name, strings.Join(launcher.Names(), ", "))
		os.Exit(2)
	}
	// The app sees its own arguments only, as if it were a program of its own.
	os.Args = append([]string{os.Args[0]}, rest...)
	run()
}

// pickApp finds which app to run from "--app NAME", "--app=NAME" or the
// program's own name, and returns the arguments meant for that app.
func pickApp(args []string) (string, []string) {
	rest := args[1:]
	if len(rest) > 0 {
		a := rest[0]
		for _, p := range []string{"--app=", "-app="} {
			if strings.HasPrefix(a, p) {
				return strings.ToLower(a[len(p):]), rest[1:]
			}
		}
		if (a == "--app" || a == "-app") && len(rest) > 1 {
			return strings.ToLower(rest[1]), rest[2:]
		}
	}
	base := args[0][strings.LastIndexAny(args[0], `/\`)+1:] // either separator, on any OS
	base = strings.ToLower(strings.TrimSuffix(base, filepath.Ext(base)))
	if _, ok := mains[base]; ok {
		return base, rest
	}
	return "", rest
}
