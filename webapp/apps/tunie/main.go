//go:build js && wasm

// The web version of Tunie: its own Main, run inside the page.
package main

import (
	"tinkerie/apps/tunie"
	"tinkerie/webapp/webhost"
)

func main() {
	webhost.Run(webhost.App{Name: "tunie", Main: tunie.Main, Addr: "127.0.0.1:47821",
		Args: []string{"-no-browser"}, Env: map[string]string{}})
}
