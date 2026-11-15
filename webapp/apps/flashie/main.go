//go:build js && wasm

// The web version of Flashie: its own Main, run inside the page.
package main

import (
	"tinkerie/apps/flashie"
	"tinkerie/webapp/webhost"
)

func main() {
	webhost.Run(webhost.App{Name: "flashie", Main: flashie.Main, Addr: "127.0.0.1:47826",
		Args: []string{}, Env: map[string]string{"FLASHIE_ADDR": "127.0.0.1:47826", "FLASHIE_NOWINDOW": "1"}})
}
