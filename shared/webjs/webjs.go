//go:build js && wasm

// Package webjs lets Tinkerie's Go code use the browser's devices (Web Serial,
// WebUSB, WebHID) through tkdev.js in the web version's page. Calls wait for
// the browser's promise, so they must run on an ordinary goroutine, never
// inside a JavaScript callback. Created by nrmluvdev for Tinkerie.
package webjs

import (
	"errors"
	"syscall/js"
	"time"
)

// ErrTimeout is returned when Await gives up waiting.
var ErrTimeout = errors.New("timed out")

// ErrNoBrowserSupport means the browser has no such device API (not Chrome or Edge).
var ErrNoBrowserSupport = errors.New("this browser can't use devices: open Tinkerie web in Chrome or Edge on a computer")

// Dev returns tkdev.js's object for kind ("serial", "usb", "hid"), or an error
// when the page or browser doesn't have it.
func Dev(kind string) (js.Value, error) {
	d := js.Global().Get("tkDev")
	if d.IsUndefined() {
		return js.Undefined(), ErrNoBrowserSupport
	}
	k := d.Get(kind)
	if !k.Call("supported").Bool() {
		return js.Undefined(), ErrNoBrowserSupport
	}
	return k, nil
}

// Call calls dev[kind].method(args...) and waits for its promise.
func Call(kind, method string, args ...any) (js.Value, error) {
	return CallTimeout(0, kind, method, args...)
}

// CallTimeout is Call, giving up after d (0 = wait for ever).
func CallTimeout(d time.Duration, kind, method string, args ...any) (v js.Value, err error) {
	k, err := Dev(kind)
	if err != nil {
		return js.Undefined(), err
	}
	defer func() { // a synchronous throw
		if r := recover(); r != nil {
			err = jsErr(r)
		}
	}()
	return Await(k.Call(method, args...), d)
}

// Await waits for a promise (or returns a plain value straight away).
func Await(p js.Value, d time.Duration) (js.Value, error) {
	if p.Type() != js.TypeObject || p.Get("then").Type() != js.TypeFunction {
		return p, nil
	}
	type res struct {
		v   js.Value
		err error
	}
	ch := make(chan res, 1)
	ok := js.FuncOf(func(_ js.Value, a []js.Value) any {
		v := js.Undefined()
		if len(a) > 0 {
			v = a[0]
		}
		ch <- res{v: v}
		return nil
	})
	bad := js.FuncOf(func(_ js.Value, a []js.Value) any {
		msg := "failed"
		if len(a) > 0 {
			msg = errText(a[0])
		}
		ch <- res{err: errors.New(msg)}
		return nil
	})
	p.Call("then", ok, bad)
	release := func() { ok.Release(); bad.Release() }
	if d <= 0 {
		r := <-ch
		release()
		return r.v, r.err
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case r := <-ch:
		release()
		return r.v, r.err
	case <-t.C:
		go func() { <-ch; release() }() // let the promise finish
		return js.Undefined(), ErrTimeout
	}
}

func errText(v js.Value) string {
	if v.Type() == js.TypeObject && v.Get("message").Type() == js.TypeString {
		return v.Get("message").String()
	}
	return v.String()
}

func jsErr(r any) error {
	if e, ok := r.(js.Error); ok {
		return errors.New(errText(e.Value))
	}
	if e, ok := r.(error); ok {
		return e
	}
	return errors.New("browser error")
}

// Bytes copies a Uint8Array into Go.
func Bytes(v js.Value) []byte {
	if v.IsNull() || v.IsUndefined() {
		return nil
	}
	b := make([]byte, v.Get("length").Int())
	js.CopyBytesToGo(b, v)
	return b
}

// JSBytes copies b into a new Uint8Array.
func JSBytes(b []byte) js.Value {
	u := js.Global().Get("Uint8Array").New(len(b))
	js.CopyBytesToJS(u, b)
	return u
}

// Object builds a plain JavaScript object.
func Object(kv map[string]any) js.Value {
	o := js.Global().Get("Object").New()
	for k, v := range kv {
		o.Set(k, v)
	}
	return o
}
