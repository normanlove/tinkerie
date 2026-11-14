//go:build js && wasm

// The web version of Makie: its own Main, run inside the page.
package main

import (
	"tinkerie/apps/makie"
	"tinkerie/webapp/webhost"
)

func main() {
	webhost.Run(webhost.App{Name: "makie", Main: makie.Main, Addr: "127.0.0.1:47811",
		Args: []string{"-nobrowser", "-idle", "0"}, Env: map[string]string{}})
}
