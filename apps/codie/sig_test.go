package codie

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSignals(t *testing.T) {
	want := map[string][]string{
		"tracie_uart_i2c.vcd":              {"UART", "115200", "Hello, Codie!", "I2C bus", "0x3C", "SSD1306", "0x76", "servo", "90°"},
		"tracie_spi.vcd":                   {"SPI bus", "9F", "JEDEC", "EF 40 18"},
		"tracie_ws2812.vcd":                {"WS2812", "3 LEDs", "red", "green", "blue"},
		"tracie_ir.vcd":                    {"infrared", "command 0x45"},
		"scope-20261110-190000.csv":        {"sine wave", "440 Hz", "A4", "PWM", "25%", "3.3 V logic"},
		"meterie-2026-11-10_19-30.nst.csv": {"lithium-ion", "discharging", "sleeps", "Power and energy"},
		"serial_2026-11-10_19-00-00.csv":   {"ESP32/ESP8266 boot", "brownout", "GPS", "48.11730", "Wi-Fi"},
	}
	out := os.Getenv("CODIE_SIG_OUT")
	for name, body := range demoCaptures() {
		if out != "" {
			os.WriteFile(filepath.Join(out, name), []byte(body), 0o644)
		}
		if !IsSignalFile(name, []byte(body[:min(4096, len(body))])) {
			t.Errorf("%s: not recognised as a capture", name)
		}
		c, err := LoadCapture(name, []byte(body))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		r := AnalyseCapture(c)
		var all strings.Builder
		for _, cd := range r.Cards {
			all.WriteString(cd.Title + " " + cd.What + " " + cd.Why + " ")
			for _, f := range cd.Facts {
				all.WriteString(f[0] + " " + f[1] + " ")
			}
		}
		for _, f := range r.Decoded {
			all.WriteString(f.Text + " " + f.Note + " ")
		}
		txt := all.String()
		for _, w := range want[name] {
			if !strings.Contains(txt, w) {
				t.Errorf("%s: missing %q", name, w)
			}
		}
		if os.Getenv("CODIE_SIG_SHOW") == name {
			for _, cd := range r.Cards {
				t.Logf("[%s] %s\n  %s\n  %s\n  %v", cd.Tag, cd.Title, cd.What, cd.Why, cd.Facts)
			}
			for _, w := range r.Warnings {
				t.Logf("WARN %s: %s", w.Title, w.Text)
			}
		}
	}
	// code is still code
	if IsSignalFile("main.py", []byte("import os\nprint('hi')\n")) {
		t.Error("python file taken for a capture")
	}
}
