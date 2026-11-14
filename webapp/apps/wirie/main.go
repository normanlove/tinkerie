//go:build js && wasm

// The web version of Wirie: its own Main, run inside the page.
package main

import (
	"tinkerie/apps/wirie"
	"tinkerie/webapp/webhost"
)

func main() {
	webhost.Run(webhost.App{Name: "wirie", Main: wirie.Main, Addr: "127.0.0.1:47820",
		Args: []string{}, Env: map[string]string{"WIRIE_NO_WINDOW": "1", "WIRIE_STAY": "1"}})
}
