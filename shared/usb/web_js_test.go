//go:build js && wasm

package usb

// Runs with Node:  GOOS=js GOARCH=wasm go test -exec="$(go env GOROOT)/lib/wasm/go_js_wasm_exec" ./shared/usb
// A pretend WebUSB device checks the browser back end (tkdev.js and web_js.go).

import (
	"bytes"
	"os"
	"syscall/js"
	"testing"
)

const mockUSB = `
(() => {
  const ep = (n, dir, type) => ({ endpointNumber: n, direction: dir, type });
  const alt = { interfaceClass: 0xff, interfaceSubclass: 0, interfaceProtocol: 0, endpoints: [ep(2, 'in', 'bulk'), ep(1, 'out', 'bulk')] };
  let echo = [], sent = 0;
  const dev = {
    vendorId: 0x04b4, productId: 0x6022, deviceVersionMajor: 1, deviceVersionMinor: 0, deviceVersionSubminor: 0,
    manufacturerName: 'Hantek', productName: 'DSO-6022BE', serialNumber: '',
    configuration: null, configurations: [{ configurationValue: 1, interfaces: [{ interfaceNumber: 0, alternates: [alt] }] }],
    opened: false,
    async open() { this.opened = true; },
    async selectConfiguration() { this.configuration = this.configurations[0]; },
    async claimInterface(i) { globalThis.__claimed = i; },
    async releaseInterface() {}, async close() { this.opened = false; },
    async controlTransferIn(s, len) { globalThis.__setup = s; return { status: 'ok', data: new DataView(new Uint8Array([1, 2, 3, 4, 5, 6, 7, 8]).slice(0, len).buffer) }; },
    async controlTransferOut(s, data) { globalThis.__setup = s; return { status: 'ok', bytesWritten: data.length }; },
    async transferOut(n, data) { echo.push(new Uint8Array(data)); return { status: 'ok', bytesWritten: data.length }; },
    async transferIn(n, len) {
      if (globalThis.__stream) { await new Promise((r) => setTimeout(r, 2)); const b = new Uint8Array(len).fill(sent++ & 255); return { status: 'ok', data: new DataView(b.buffer) }; }
      for (let i = 0; i < 50 && !echo.length; i++) await new Promise((r) => setTimeout(r, 10));
      if (!echo.length) return new Promise(() => {});
      return { status: 'ok', data: new DataView(echo.shift().buffer) };
    },
    async clearHalt() {},
  };
  const usb = new EventTarget();
  usb.getDevices = async () => [dev];
  Object.defineProperty(globalThis, 'navigator', { value: { usb }, configurable: true });
})();`

func loadDev(t *testing.T, mock string) {
	src, err := os.ReadFile("../../webapp/static/tkdev.js")
	if err != nil {
		t.Fatal(err)
	}
	js.Global().Call("eval", mock)
	js.Global().Call("eval", string(src))
}

func TestWebUSB(t *testing.T) {
	loadDev(t, mockUSB)
	l := List()
	if len(l) != 1 || l[0].VID != 0x04b4 || l[0].PID != 0x6022 || len(l[0].Interfaces) != 1 ||
		!bytes.Equal(l[0].Interfaces[0].BulkIn, []byte{0x82}) || !bytes.Equal(l[0].Interfaces[0].BulkOut, []byte{0x01}) {
		t.Fatalf("list: %+v", l)
	}
	d, err := Open(l[0].Path, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	buf := make([]byte, 4)
	if n, err := d.Control(0xC0, 0xA2, 1, 0, buf, 1000); err != nil || n != 4 || buf[3] != 4 {
		t.Fatalf("control in: %d %v %v", n, err, buf)
	}
	if s := js.Global().Get("__setup"); s.Get("requestType").String() != "vendor" || s.Get("recipient").String() != "device" || s.Get("request").Int() != 0xA2 {
		t.Fatalf("setup: %v", s)
	}
	if n, err := d.Control(0x41, 0xE0, 0, 0, []byte{9, 9}, 1000); err != nil || n != 2 {
		t.Fatalf("control out: %d %v", n, err)
	}
	if js.Global().Get("__setup").Get("recipient").String() != "interface" {
		t.Fatal("recipient")
	}
	if _, err := d.BulkWrite(0x01, []byte("ping"), 1000); err != nil {
		t.Fatal(err)
	}
	in := make([]byte, 64)
	if n, err := d.BulkRead(0x82, in, 1000); err != nil || string(in[:n]) != "ping" {
		t.Fatalf("bulk read: %q %v", in[:n], err)
	}
	// a read that times out leaves its transfer waiting; the next read gets its data
	if _, err := d.BulkRead(0x82, in, 100); err == nil {
		t.Fatal("expected a timeout")
	}
	d.BulkWrite(0x01, []byte("late"), 1000)
	if n, err := d.BulkRead(0x82, in, 1000); err != nil || string(in[:n]) != "late" {
		t.Fatalf("after timeout: %q %v", in[:n], err)
	}
	js.Global().Set("__stream", true)
	got, stop := 0, make(chan struct{})
	err = d.Stream(0x82, 512, 4, stop, func(b []byte) bool { got++; return got < 20 })
	if err != nil || got != 20 {
		t.Fatalf("stream: %d %v", got, err)
	}
}
