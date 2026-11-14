//go:build js && wasm

// Package webhost runs a Tinkerie app inside a web page.
//
// The app's own Main runs unchanged: it starts its usual local web server,
// but on Go's in-page network (net_fake.go), not a real port. The page's
// service worker forwards the app frame's requests here through tkFetch,
// which sends them to that server and streams the answer back, so the app's
// pages, API calls and live event streams all work as on a desktop.
package webhost

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"os"
	"strings"
	"syscall/js"
	"time"
)

// App describes how to start one app for the web.
type App struct {
	Name string            // "fixie"
	Main func()            // the app's Main
	Addr string            // where its server listens, "127.0.0.1:47825"
	Args []string          // command-line flags, e.g. -no-browser
	Env  map[string]string // environment, e.g. FIXIE_NO_WINDOW=1
}

// Run starts the app and serves the page's requests until the page closes.
func Run(a App) {
	os.Setenv("HOME", "/home/web")
	os.Setenv("XDG_CONFIG_HOME", "/home/web/.config")
	os.Setenv("XDG_CACHE_HOME", "/home/web/.cache")
	os.Setenv("XDG_DATA_HOME", "/home/web/.local/share")
	os.Setenv("TINKERIE_WEB", "1")
	for k, v := range a.Env {
		os.Setenv(k, v)
	}
	for _, d := range []string{"/home/web/.config", "/home/web/.cache", "/home/web/.local/share", "/tmp"} {
		os.MkdirAll(d, 0o755)
	}
	os.Args = append([]string{"tinkerie"}, a.Args...)
	go a.Main()

	dial := func(ctx context.Context, _, _ string) (net.Conn, error) {
		var d net.Dialer
		return d.DialContext(ctx, "tcp", a.Addr)
	}
	client := &http.Client{Transport: &http.Transport{DialContext: dial, DisableCompression: true, MaxIdleConnsPerHost: 16}}

	js.Global().Set("tkFetch", js.FuncOf(func(this js.Value, args []js.Value) any {
		// tkFetch(method, path, headers, body, onHead, onChunk, onEnd)
		method, path, hdrs := args[0].String(), args[1].String(), args[2]
		var body []byte
		if b := args[3]; !b.IsNull() && !b.IsUndefined() {
			body = make([]byte, b.Get("length").Int())
			js.CopyBytesToGo(body, b)
		}
		onHead, onChunk, onEnd := args[4], args[5], args[6]
		ctx, cancel := context.WithCancel(context.Background())
		go serve(ctx, client, a.Addr, method, path, hdrs, body, onHead, onChunk, onEnd)
		var stop js.Func
		stop = js.FuncOf(func(js.Value, []js.Value) any { cancel(); stop.Release(); return nil })
		return stop // the page calls this when the frame stops listening
	}))

	// Wait until the app is listening, then tell the page.
	for i := 0; ; i++ {
		c, err := net.Dial("tcp", a.Addr)
		if err == nil {
			c.Close()
			break
		}
		if i > 600 {
			js.Global().Call("tkFailed", "the app didn't start: "+err.Error())
			select {}
		}
		time.Sleep(50 * time.Millisecond)
	}
	js.Global().Call("tkReady")
	select {}
}

func serve(ctx context.Context, client *http.Client, addr, method, path string, hdrs js.Value, body []byte, onHead, onChunk, onEnd js.Value) {
	req, err := http.NewRequestWithContext(ctx, method, "http://"+addr+path, bytes.NewReader(body))
	if err != nil {
		onEnd.Invoke(err.Error())
		return
	}
	keys := js.Global().Get("Object").Call("keys", hdrs)
	for i := 0; i < keys.Length(); i++ {
		k := keys.Index(i).String()
		switch strings.ToLower(k) {
		case "host", "origin", "referer", "cookie", "accept-encoding", "content-length":
			continue
		}
		req.Header.Set(k, hdrs.Get(k).String())
	}
	if method != "GET" && method != "HEAD" {
		req.Header.Set("Origin", "http://"+addr) // the apps accept only their own origin
	}
	resp, err := client.Do(req)
	if err != nil {
		onEnd.Invoke(err.Error())
		return
	}
	defer resp.Body.Close()
	h := js.Global().Get("Object").New()
	for k, v := range resp.Header {
		h.Set(k, strings.Join(v, ", "))
	}
	onHead.Invoke(resp.StatusCode, h)
	buf := make([]byte, 64<<10)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			u := js.Global().Get("Uint8Array").New(n)
			js.CopyBytesToJS(u, buf[:n])
			onChunk.Invoke(u)
		}
		if err != nil {
			break
		}
	}
	onEnd.Invoke(js.Null())
}
