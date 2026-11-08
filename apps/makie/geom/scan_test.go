package geom

import (
	"testing"
	"time"
)

func TestScanKnownBoards(t *testing.T) {
	for _, name := range []string{"Raspberry Pi 4 Model B", "0.96\" OLED (SSD1306, 128×64)", "HC-SR04 ultrasonic sensor", "Raspberry Pi Pico / Pico W / Pico 2", "1.28\" round TFT (GC9A01)", "CYD 2.8\" (ESP32-2432S028R)"} {
		ref := boardDB[name]
		model := FitMesh(ref, "Board model (fit check)", Params{"_q": 1.0})
		// pretend it was imported somewhere random
		model = model.Transformed(Translate(37, -12, 5))
		start := time.Now()
		r := ScanModel(model, "scan")
		el := time.Since(start)
		if !r.OK {
			t.Errorf("%s: scan failed %v", name, r.Report)
			continue
		}
		s := r.Spec
		t.Logf("%s (%v)\n   want %.1f×%.1f holes=%d window=%v circles=%d ports=%d\n   got  %.1f×%.1f T=%.2f holes=%d window=%v circles=%d ports=%d round=%v origin=%v\n   %v",
			name, el.Round(time.Millisecond), ref.W, ref.D, len(ref.Holes), ref.Window != nil, len(ref.Circles), len(ref.Ports),
			s.W, s.D, s.T, len(s.Holes), s.Window != nil, len(s.Circles), len(s.Ports), s.Round, r.Origin, r.Report)
		// and the parts made from the scan must build
		for _, mode := range fitModes {
			p := Params{"_q": 1.0}
			if m := FitMesh(s, mode, p); m == nil || m.Volume() <= 0 {
				t.Errorf("%s: %s from scan is empty", name, mode)
			}
		}
	}
}
