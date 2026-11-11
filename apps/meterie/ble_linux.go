//go:build linux

package meterie

// ble_linux.go: Bluetooth LE meters on Linux and Raspberry Pi OS through
// BlueZ, the system's Bluetooth service, over D-Bus (shared/dbus): scan,
// connect, and take the meter's readings from a notifying characteristic.
// Android has no BlueZ, so there Bluetooth meters aren't offered.

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"tinkerie/shared/dbus"
	"tinkerie/shared/droid"
)

const bluezName = "org.bluez"

var bz struct {
	mu       sync.Mutex
	conn     *dbus.Conn
	adapter  string
	scanning bool
	stop     chan struct{}
	results  map[uint64]*PortInfo
}

func fmtBTAddr(a uint64) string {
	return fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X", byte(a>>40), byte(a>>32), byte(a>>24), byte(a>>16), byte(a>>8), byte(a))
}

func parseBTAddr(s string) uint64 {
	var a uint64
	for _, p := range strings.Split(s, ":") {
		var b uint64
		fmt.Sscanf(p, "%x", &b)
		a = a<<8 | b
	}
	return a
}

// bleUUIDFull turns "fff4" or a full UUID into BlueZ's lower-case form.
func bleUUIDFull(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	switch len(s) {
	case 4:
		return "0000" + s + "-0000-1000-8000-00805f9b34fb"
	case 8:
		return s + "-0000-1000-8000-00805f9b34fb"
	}
	return s
}

// bluez connects to BlueZ and finds the Bluetooth adapter.
func bluez() (*dbus.Conn, string, error) {
	bz.mu.Lock()
	defer bz.mu.Unlock()
	if droid.Active() {
		return nil, "", errors.New("Bluetooth meters aren't available in the Android app yet")
	}
	if bz.conn == nil {
		c, err := dbus.SystemBus()
		if err != nil {
			return nil, "", fmt.Errorf("Bluetooth isn't available: %v", err)
		}
		bz.conn = c
	}
	objs, err := managedObjects(bz.conn)
	if err != nil {
		bz.conn.Close()
		bz.conn = nil
		if strings.Contains(err.Error(), "ServiceUnknown") || strings.Contains(err.Error(), "org.bluez") {
			return nil, "", errors.New("Bluetooth isn't running on this computer (install and start BlueZ, the bluetooth service)")
		}
		return nil, "", err
	}
	bz.adapter = ""
	paths := make([]string, 0, len(objs))
	for p := range objs {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		if _, ok := objs[p]["org.bluez.Adapter1"]; ok {
			bz.adapter = p
			if props := objs[p]["org.bluez.Adapter1"]; props["Powered"] != true {
				bz.conn.Call(bluezName, p, "org.freedesktop.DBus.Properties", "Set", "ssv", "org.bluez.Adapter1", "Powered", dbus.Variant{Sig: "b", Value: true})
			}
			break
		}
	}
	if bz.adapter == "" {
		return nil, "", errors.New("no Bluetooth adapter found: is Bluetooth switched on?")
	}
	return bz.conn, bz.adapter, nil
}

// managedObjects returns path → interface → properties for all of BlueZ.
func managedObjects(c *dbus.Conn) (map[string]map[string]map[string]any, error) {
	r, err := c.CallTimeout(10*time.Second, bluezName, "/", "org.freedesktop.DBus.ObjectManager", "GetManagedObjects", "")
	if err != nil {
		return nil, err
	}
	out := map[string]map[string]map[string]any{}
	if len(r) == 0 {
		return out, nil
	}
	objs, _ := r[0].(map[string]any)
	for p, v := range objs {
		ifs, _ := v.(map[string]any)
		m := map[string]map[string]any{}
		for name, props := range ifs {
			pm, _ := props.(map[string]any)
			m[name] = pm
		}
		out[p] = m
	}
	return out, nil
}

func platformBLEScan(on bool) error {
	c, adapter, err := bluez()
	if err != nil {
		return err
	}
	bz.mu.Lock()
	defer bz.mu.Unlock()
	if !on {
		if bz.scanning {
			close(bz.stop)
			bz.scanning = false
			c.Call(bluezName, adapter, "org.bluez.Adapter1", "StopDiscovery", "")
		}
		return nil
	}
	if bz.scanning {
		return nil
	}
	if bz.results == nil {
		bz.results = map[uint64]*PortInfo{}
	}
	c.Call(bluezName, adapter, "org.bluez.Adapter1", "SetDiscoveryFilter", "a{sv}", map[string]dbus.Variant{"Transport": {Sig: "s", Value: "le"}})
	if _, err := c.Call(bluezName, adapter, "org.bluez.Adapter1", "StartDiscovery", ""); err != nil && !strings.Contains(err.Error(), "InProgress") {
		return fmt.Errorf("couldn't start a Bluetooth scan: %v", err)
	}
	bz.scanning = true
	bz.stop = make(chan struct{})
	go scanLoop(c, adapter, bz.stop)
	return nil
}

// scanLoop reads the devices BlueZ has seen, once a second.
func scanLoop(c *dbus.Conn, adapter string, stop chan struct{}) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		if objs, err := managedObjects(c); err == nil {
			bz.mu.Lock()
			for p, ifs := range objs {
				d, ok := ifs["org.bluez.Device1"]
				if !ok || !strings.HasPrefix(p, adapter+"/") {
					continue
				}
				addrS, _ := d["Address"].(string)
				addr := parseBTAddr(addrS)
				rssi, hasRSSI := d["RSSI"].(int64)
				if !hasRSSI && bz.results[addr] == nil {
					continue // not seen in this scan
				}
				pi := bz.results[addr]
				if pi == nil {
					pi = &PortInfo{Kind: TrBLE, Path: addrS, BLEAddr: addr}
					bz.results[addr] = pi
				}
				if n, _ := d["Name"].(string); n != "" {
					pi.Name = n
				} else if a, _ := d["Alias"].(string); a != "" && pi.Name == "" && !strings.Contains(a, "-") {
					pi.Name = a
				}
				if hasRSSI {
					pi.RSSI = int(rssi)
				}
			}
			bz.mu.Unlock()
			appWake()
		}
		select {
		case <-stop:
			return
		case <-t.C:
		}
	}
}

func platformBLEResults() []PortInfo {
	bz.mu.Lock()
	defer bz.mu.Unlock()
	var out []PortInfo
	for _, p := range bz.results {
		if p.Name == "" && bleNameMatch(p.Name) == nil && p.RSSI < -85 {
			continue
		}
		out = append(out, *p)
	}
	return out
}

// ------------------------------------------------------------------ GATT

type blePort struct {
	c         *dbus.Conn
	dev, char string
	frames    chan []byte
	closed    chan struct{}
	once      sync.Once
}

func platformOpenBLE(addr uint64, svc, chr string) (Port, error) {
	if addr == 0 {
		return nil, errors.New("no Bluetooth address: scan again")
	}
	c, adapter, err := bluez()
	if err != nil {
		return nil, err
	}
	dev := adapter + "/dev_" + strings.ReplaceAll(fmtBTAddr(addr), ":", "_")
	objs, err := managedObjects(c)
	if err != nil {
		return nil, err
	}
	if _, ok := objs[dev]; !ok { // BlueZ forgot it: look for it again
		c.Call(bluezName, adapter, "org.bluez.Adapter1", "StartDiscovery", "")
		for i := 0; i < 30; i++ {
			time.Sleep(500 * time.Millisecond)
			if objs, _ = managedObjects(c); objs[dev] != nil {
				break
			}
		}
		bz.mu.Lock()
		scanning := bz.scanning
		bz.mu.Unlock()
		if !scanning {
			c.Call(bluezName, adapter, "org.bluez.Adapter1", "StopDiscovery", "")
		}
		if objs[dev] == nil {
			return nil, errors.New("the meter isn't in range: switch its Bluetooth on and scan again")
		}
	}
	if _, err := c.CallTimeout(40*time.Second, bluezName, dev, "org.bluez.Device1", "Connect", ""); err != nil && !strings.Contains(err.Error(), "AlreadyConnected") {
		return nil, fmt.Errorf("couldn't connect to the meter: %v", err)
	}
	// wait until BlueZ has read the meter's services
	for i := 0; i < 40; i++ {
		if r, err := c.Call(bluezName, dev, "org.freedesktop.DBus.Properties", "Get", "ss", "org.bluez.Device1", "ServicesResolved"); err == nil && len(r) > 0 && r[0] == true {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	objs, _ = managedObjects(c)
	char := findChar(objs, dev, svc, chr)
	if char == "" {
		c.Call(bluezName, dev, "org.bluez.Device1", "Disconnect", "")
		return nil, errors.New("the meter doesn't offer the expected data service: is it the right model?")
	}
	p := &blePort{c: c, dev: dev, char: char, frames: make(chan []byte, 256), closed: make(chan struct{})}
	sigs := c.Signals()
	c.AddMatch("type='signal',sender='org.bluez',interface='org.freedesktop.DBus.Properties',member='PropertiesChanged',path='" + char + "'")
	c.AddMatch("type='signal',sender='org.bluez',interface='org.freedesktop.DBus.Properties',member='PropertiesChanged',path='" + dev + "'")
	go p.listen(sigs)
	if _, err := c.Call(bluezName, char, "org.bluez.GattCharacteristic1", "StartNotify", ""); err != nil {
		p.Close()
		return nil, fmt.Errorf("the meter wouldn't send readings: %v", err)
	}
	return p, nil
}

// findChar picks the characteristic to listen to: the model's, or the first
// one that notifies (inside the model's service when it names one).
func findChar(objs map[string]map[string]map[string]any, dev, svc, chr string) string {
	services := map[string]string{} // path → UUID
	for p, ifs := range objs {
		if s, ok := ifs["org.bluez.GattService1"]; ok && strings.HasPrefix(p, dev+"/") {
			u, _ := s["UUID"].(string)
			services[p] = strings.ToLower(u)
		}
	}
	var paths []string
	for p := range objs {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	best := ""
	for _, p := range paths {
		c, ok := objs[p]["org.bluez.GattCharacteristic1"]
		if !ok || !strings.HasPrefix(p, dev+"/") {
			continue
		}
		sp, _ := c["Service"].(string)
		if svc != "" && services[sp] != bleUUIDFull(svc) {
			continue
		}
		u, _ := c["UUID"].(string)
		if chr != "" {
			if strings.ToLower(u) == bleUUIDFull(chr) {
				return p
			}
			continue
		}
		flags, _ := c["Flags"].([]any)
		for _, f := range flags {
			if f == "notify" || f == "indicate" {
				if best == "" {
					best = p
				}
			}
		}
	}
	return best
}

func (p *blePort) listen(sigs <-chan *dbus.Message) {
	for {
		select {
		case <-p.closed:
			return
		case m := <-sigs:
			if len(m.Body) < 2 {
				continue
			}
			changed, _ := m.Body[1].(map[string]any)
			switch m.Path {
			case p.char:
				if v, ok := changed["Value"].([]byte); ok && len(v) > 0 {
					select {
					case p.frames <- v:
					default:
					}
				}
			case p.dev:
				if c, ok := changed["Connected"].(bool); ok && !c {
					p.Close()
					return
				}
			}
		}
	}
}

func (p *blePort) Read(b []byte) (int, error) {
	select {
	case f := <-p.frames:
		return copy(b, f), nil
	case <-p.closed:
		return 0, errors.New("Bluetooth connection closed")
	case <-time.After(200 * time.Millisecond):
		return 0, nil
	}
}

func (p *blePort) Write(b []byte) (int, error) { return len(b), nil }

func (p *blePort) Close() error {
	p.once.Do(func() {
		close(p.closed)
		p.c.Call(bluezName, p.char, "org.bluez.GattCharacteristic1", "StopNotify", "")
		p.c.Call(bluezName, p.dev, "org.bluez.Device1", "Disconnect", "")
	})
	return nil
}
