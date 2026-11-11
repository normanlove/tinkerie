//go:build linux

package dbus

import (
	"bytes"
	"testing"
)

// Round trip: what we send decodes back to the same values.
func TestRoundTrip(t *testing.T) {
	var e encoder
	args := []any{"hello", Variant{"s", "le"}, map[string]Variant{"Transport": {"s", "le"}, "RSSI": {"n", int16(-60)}}, []byte{1, 2, 3}, true, uint32(7)}
	if err := e.values("sva{sv}aybu", args); err != nil {
		t.Fatal(err)
	}
	msg := header(4, 9, e.b.Len(), "/org/bluez/hci0", "org.freedesktop.DBus.Properties", "PropertiesChanged", "", "sva{sv}aybu")
	msg = append(msg, e.b.Bytes()...)
	m, err := readMessage(bytes.NewReader(msg))
	if err != nil {
		t.Fatal(err)
	}
	if m.Path != "/org/bluez/hci0" || m.Member != "PropertiesChanged" || m.Type != 4 || m.Serial != 9 {
		t.Fatalf("header: %+v", m)
	}
	if m.Body[0] != "hello" || m.Body[1] != "le" || m.Body[4] != true || m.Body[5] != uint64(7) {
		t.Fatalf("body: %#v", m.Body)
	}
	d := m.Body[2].(map[string]any)
	if d["Transport"] != "le" || d["RSSI"] != int64(-60) {
		t.Fatalf("dict: %#v", d)
	}
	if !bytes.Equal(m.Body[3].([]byte), []byte{1, 2, 3}) {
		t.Fatalf("bytes: %#v", m.Body[3])
	}
}

// Talks to the real system bus when there is one (skipped otherwise).
func TestSystemBus(t *testing.T) {
	c, err := SystemBus()
	if err != nil {
		t.Skip("no system bus:", err)
	}
	defer c.Close()
	r, err := c.Call("org.freedesktop.DBus", "/org/freedesktop/DBus", "org.freedesktop.DBus", "ListNames", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%d names on the bus", len(r[0].([]any)))
}
