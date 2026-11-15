//go:build js && wasm

// The web version of Probie: its own Main, run inside the page.
package main

import (
	"tinkerie/apps/probie"
	"tinkerie/webapp/webhost"
)

func main() {
	webhost.Run(webhost.App{Name: "probie", Main: probie.Main, Addr: "127.0.0.1:47822",
		Args: []string{"-nowindow"}, Env: map[string]string{}})
}
