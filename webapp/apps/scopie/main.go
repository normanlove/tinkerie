//go:build js && wasm

// The web version of Scopie: its own Main, run inside the page.
package main

import (
	"tinkerie/apps/scopie"
	"tinkerie/webapp/webhost"
)

func main() {
	webhost.Run(webhost.App{Name: "scopie", Main: scopie.Main, Addr: "127.0.0.1:47831",
		Args: []string{"-no-browser"}, Env: map[string]string{}})
}
