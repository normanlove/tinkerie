//go:build linux

package meterie

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"tinkerie/shared/dbus"
)

// A stand-in for BlueZ on a private bus: a meter that's found by a scan,
// connects, and sends one reading.
func TestBlueZ(t *testing.T) {
	daemon, err := exec.LookPath("dbus-daemon")
	if err != nil {
		t.Skip("no dbus-daemon")
	}
	dir := t.TempDir()
	sock := filepath.Join(dir, "bus")
	conf := filepath.Join(dir, "bus.conf")
	os.WriteFile(conf, []byte(`<!DOCTYPE busconfig PUBLIC "-//freedesktop//DTD D-Bus Bus Configuration 1.0//EN" "http://www.freedesktop.org/standards/dbus/1.0/busconfig.dtd">
<busconfig><type>system</type><listen>unix:path=`+sock+`</listen><auth>EXTERNAL</auth>
<policy context="default"><allow user="*"/><allow own="*"/><allow send_destination="*"/><allow receive_sender="*"/></policy></busconfig>`), 0o644)
	cmd := exec.Command(daemon, "--config-file="+conf, "--nofork")
	if err := cmd.Start(); err != nil {
		t.Skip("dbus-daemon won't start:", err)
	}
	defer cmd.Process.Kill()
	for i := 0; i < 50; i++ {
		if _, err := os.Stat(sock); err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Setenv("DBUS_SYSTEM_BUS_ADDRESS", "unix:path="+sock)

	fake, err := dbus.SystemBus()
	if err != nil {
		t.Fatal(err)
	}
	defer fake.Close()
	if err := fake.RequestName("org.bluez"); err != nil {
		t.Fatal(err)
	}
	const dev = "/org/bluez/hci0/dev_AA_BB_CC_DD_EE_01"
	const svc, chr = dev + "/service0010", dev + "/service0010/char0014"
	connected := false
	objects := func() map[string]any {
		o := map[string]any{
			"/org/bluez/hci0": map[string]any{"org.bluez.Adapter1": map[string]dbus.Variant{"Powered": {Sig: "b", Value: true}}},
			dev: map[string]any{"org.bluez.Device1": map[string]dbus.Variant{
				"Address": {Sig: "s", Value: "AA:BB:CC:DD:EE:01"}, "Name": {Sig: "s", Value: "Test meter"}, "RSSI": {Sig: "n", Value: int16(-55)}}},
		}
		if connected {
			o[svc] = map[string]any{"org.bluez.GattService1": map[string]dbus.Variant{
				"UUID": {Sig: "s", Value: "0000fff0-0000-1000-8000-00805f9b34fb"}, "Device": {Sig: "o", Value: dev}}}
			o[chr] = map[string]any{"org.bluez.GattCharacteristic1": map[string]dbus.Variant{
				"UUID": {Sig: "s", Value: "0000fff4-0000-1000-8000-00805f9b34fb"}, "Service": {Sig: "o", Value: svc}, "Flags": {Sig: "as", Value: []string{"notify"}}}}
		}
		return o
	}
	go func() {
		for m := range fake.Calls() {
			switch m.Member {
			case "GetManagedObjects":
				fake.Reply(m, "a{oa{sa{sv}}}", objects())
			case "Get":
				fake.Reply(m, "v", dbus.Variant{Sig: "b", Value: true})
			case "Connect":
				connected = true
				fake.Reply(m, "")
			case "StartNotify":
				fake.Reply(m, "")
				fake.Emit(chr, "org.freedesktop.DBus.Properties", "PropertiesChanged", "sa{sv}as",
					"org.bluez.GattCharacteristic1", map[string]dbus.Variant{"Value": {Sig: "ay", Value: []byte{0xAB, 0xCD, 0x01}}}, []string{})
			default:
				fake.Reply(m, "")
			}
		}
	}()

	if err := platformBLEScan(true); err != nil {
		t.Fatal("scan:", err)
	}
	var found []PortInfo
	for i := 0; i < 30 && len(found) == 0; i++ {
		time.Sleep(100 * time.Millisecond)
		found = platformBLEResults()
	}
	platformBLEScan(false)
	if len(found) != 1 || found[0].Name != "Test meter" || found[0].RSSI != -55 || found[0].BLEAddr != 0xAABBCCDDEE01 {
		t.Fatalf("scan found %+v", found)
	}
	p, err := platformOpenBLE(found[0].BLEAddr, "fff0", "fff4")
	if err != nil {
		t.Fatal("connect:", err)
	}
	defer p.Close()
	b := make([]byte, 16)
	var n int
	for i := 0; i < 20 && n == 0; i++ {
		n, _ = p.Read(b)
	}
	if !bytes.Equal(b[:n], []byte{0xAB, 0xCD, 0x01}) {
		t.Fatalf("reading: % x", b[:n])
	}
}
