//go:build js && wasm

// The web version of Codie: its own Main, run inside the page.
package main

import (
	"tinkerie/apps/codie"
	"tinkerie/webapp/webhost"
)

func main() {
	webhost.Run(webhost.App{Name: "codie", Main: codie.Main, Addr: "127.0.0.1:47824",
		Args: []string{"-no-browser"}, Env: map[string]string{"CODIE_NO_WINDOW": "1"}})
}
