//go:build !windows

package meterie

// main_other.go: Meterie on Linux and macOS opens its window (web_front.go).
// -preview outdir renders every page to PNGs instead (a development aid).

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"time"
)

func Main() {
	out := flag.String("preview", "", "render preview PNGs of every page into this folder")
	w := flag.Int("w", 1440, "width")
	h := flag.Int("h", 900, "height")
	ico := flag.String("icon", "", "write the app icon (multi-size .ico) to this path")
	flag.Parse()
	if *ico != "" {
		if err := writeICO(*ico); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		return
	}
	if *out == "" {
		runWeb() // the real app: Meterie in a browser app window
		return
	}
	os.MkdirAll(*out, 0o755)
	a := NewApp()
	a.settings.ShowWelcome = false
	a.settings.APIEnabled = false
	seedPreview(a)
	for p := 0; p < pageCount; p++ {
		a.page = p
		var tx *previewText
		var c *Canvas
		for f := 0; f < 40; f++ { // let animations settle
			c = NewCanvas(*w, *h)
			tx = &previewText{}
			in := Input{MX: -100, MY: -100}
			if p == PageLive && f == 39 {
				in = Input{MX: *w * 70 / 100, MY: *h * 82 / 100, HasMouse: true}
			}
			a.lastFrame = time.Now().Add(-50 * time.Millisecond)
			a.Frame(c, tx, in)
		}
		name := fmt.Sprintf("%02d_%s", p, pageInfo[p].name)
		writePreview(filepath.Join(*out, name), c, tx)
	}
	// extra: wizard screens and the welcome modal
	extra := []struct {
		name string
		set  func()
	}{
		{"wiz_battery", func() { a.page, a.wiz.which = PageWizards, 0 }},
		{"wiz_sorter", func() { a.page, a.wiz.which = PageWizards, 2 }},
		{"wiz_load", func() { a.page, a.wiz.which = PageWizards, 4 }},
		{"meters_models", func() { a.page, a.mp.tab = PageMeters, 1 }},
		{"meters_ble", func() { a.page, a.mp.tab = PageMeters, 2 }},
		{"export_modal", func() { a.page, a.modal = PageLive, "export" }},
	}
	for _, e := range extra {
		e.set()
		var tx *previewText
		var c *Canvas
		for f := 0; f < 40; f++ {
			c = NewCanvas(*w, *h)
			tx = &previewText{}
			a.lastFrame = time.Now().Add(-50 * time.Millisecond)
			a.Frame(c, tx, Input{MX: -100, MY: -100})
		}
		writePreview(filepath.Join(*out, e.name), c, tx)
		a.modal = ""
	}
}

func writePreview(base string, c *Canvas, t *previewText) {
	f, _ := os.Create(base + ".png")
	png.Encode(f, c.ToImage(Rect{0, 0, c.W, c.H}))
	f.Close()
	b, _ := json.Marshal(t.ops)
	os.WriteFile(base+".json", b, 0o644)
}

// seedPreview fills channels with simulated history.
func seedPreview(a *App) {
	now := time.Now()
	mk := func(sc DemoScenario, name string) (*Channel, *DemoSim) {
		d := &Device{ID: a.nextDevID, Kind: TrDemo, Label: "Demo", demo: &DemoSim{Scenario: sc}}
		a.nextDevID++
		ch := a.addChannel(name, KindMeter, "HoldPeak HP-90EPC · COM5", d.ID)
		d.ChannelID = ch.ID
		a.devices = append(a.devices, d)
		d.setStatus("Live", nil, true)
		return ch, d.demo
	}
	ch, sim := mk(DemoBattery, "CH1")
	sim.t0 = now.Add(-110 * time.Second)
	sim.rng = newRand()
	for t := now.Add(-110 * time.Second); t.Before(now); t = t.Add(250 * time.Millisecond) {
		for _, r := range sim.next(t) {
			r.Counts = 4000
			r.Display = fmt.Sprintf("%.3f", r.Value)
			a.ingest(ch, r)
		}
	}
	ch2, sim2 := mk(DemoMotor, "CH2")
	ch2.Source = "UNI-T UT61E+ · USB"
	sim2.t0 = now.Add(-110 * time.Second)
	sim2.rng = newRand()
	for t := now.Add(-110 * time.Second); t.Before(now); t = t.Add(300 * time.Millisecond) {
		for _, r := range sim2.next(t) {
			a.ingest(ch2, r)
		}
	}
	pm := a.addChannel("Power", KindMath, "Power  P = A × B", 0)
	pm.Math = &MathDef{Preset: MathPower, A: 1, B: 2}
	for i := range ch.Samples {
		s := ch.Samples[i]
		v2, ok := sampleAt(ch2, s.T, 0)
		if ok && i%2 == 0 {
			r := Reading{T: time.Unix(0, s.T), Value: s.V * v2, Q: QPower, Unit: "W"}
			num, pu := SplitSI(r.Value, "W", 4)
			r.Display, r.Prefix = num, pu[:len(pu)-1]
			a.ingest(pm, r)
		}
	}
	pm.Visible = false
	a.markers = append(a.markers, Marker{T: now.Add(-70 * time.Second).UnixNano(), Text: "load on", Color: 1})
	a.alarms = append(a.alarms, &Alarm{Chan: 1, UseLow: true, Low: 3.6, UseHigh: true, High: 4.25, Enabled: true, Sound: true})
	a.graph.Window = 120
	a.wiz.soParts = []sortedPart{{now, 4712, 4700, 0.25, "PASS"}, {now, 2251, 2200, 2.3, "PASS"}, {now, 1062, 1000, 6.2, "HIGH"}, {now, 98.1, 100, -1.9, "PASS"}}
	a.wiz.ltPts = []loadPt{{21.4, 0.02}, {20.9, 0.15}, {20.1, 0.3}, {18.6, 0.45}, {16.5, 0.52}, {12, 0.56}, {4, 0.58}}
	previewPorts = []PortInfo{
		{Kind: TrSerial, Path: "COM5", Name: "USB-SERIAL CH340 (COM5)", VID: 0x1A86, PID: 0x7523, Hint: "CH340 USB-serial — press Detect to identify the meter"},
		{Kind: TrCP2110, Path: `\\?\hid#vid_10c4&pid_ea80`, Name: "UNI-T UT61E+ / UT161 (CP2110)", Hint: "UNI-T UT61E+ family — connects directly"},
	}
	previewBLE = []PortInfo{{Kind: TrBLE, Path: "A4:C1:38:12:34:56", Name: "BDM", RSSI: -58}, {Kind: TrBLE, Path: "FC:58:FA:3C:94:16", Name: "Bluetooth DMM", RSSI: -71}}
	_ = math.Pi
}

// writeICO writes a PNG-compressed multi-resolution .ico file.
func writeICO(path string) error {
	sizes := []int{16, 20, 24, 32, 40, 48, 64, 128, 256}
	var imgs [][]byte
	for _, sz := range sizes {
		var b bytes.Buffer
		png.Encode(&b, renderLogoImage(sz))
		imgs = append(imgs, b.Bytes())
	}
	var out bytes.Buffer
	binary.Write(&out, binary.LittleEndian, [3]uint16{0, 1, uint16(len(sizes))})
	off := 6 + 16*len(sizes)
	for i, sz := range sizes {
		w := byte(sz)
		if sz >= 256 {
			w = 0
		}
		out.Write([]byte{w, w, 0, 0})
		binary.Write(&out, binary.LittleEndian, uint16(1))
		binary.Write(&out, binary.LittleEndian, uint16(32))
		binary.Write(&out, binary.LittleEndian, uint32(len(imgs[i])))
		binary.Write(&out, binary.LittleEndian, uint32(off))
		off += len(imgs[i])
	}
	for _, b := range imgs {
		out.Write(b)
	}
	return os.WriteFile(path, out.Bytes(), 0o644)
}
