//go:build js && wasm

// The web version of Meterie: its own Main, run inside the page.
package main

import (
	"tinkerie/apps/meterie"
	"tinkerie/webapp/webhost"
)

func main() {
	webhost.Run(webhost.App{Name: "meterie", Main: meterie.Main, Addr: "127.0.0.1:47827",
		Args: []string{}, Env: map[string]string{"METERIE_ADDR": "127.0.0.1:47827"}})
}
