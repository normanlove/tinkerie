//go:build js && wasm

// The web version of Fixie: its own Main, run inside the page.
package main

import (
	"tinkerie/apps/fixie"
	"tinkerie/webapp/webhost"
)

func main() {
	webhost.Run(webhost.App{Name: "fixie", Main: fixie.Main, Addr: "127.0.0.1:47825",
		Args: []string{"-no-browser"}, Env: map[string]string{"FIXIE_NO_WINDOW": "1"}})
}
