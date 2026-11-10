// Codie — Signals: example captures (made in code, so they're small and
// exact), shown in the Examples menu in Signals mode and used by the tests.
// Part of the Tinkerie maker suite. Created by nrmluvdev.

package codie

import (
	"fmt"
	"math"
	"strings"
)

// SigDemo is an example capture.
type SigDemo struct {
	File  string `json:"file"`
	Label string `json:"label"`
	App   string `json:"app"`
}

// SigDemos lists the example captures in menu order.
var SigDemos = []SigDemo{
	{"scope-20261110-190000.csv", "Scopie: a 440 Hz tone and a PWM signal", "Scopie"},
	{"tracie_uart_i2c.vcd", "Tracie: serial text, an I2C OLED and a servo", "Tracie"},
	{"tracie_spi.vcd", "Tracie: SPI flash chip reading its ID", "Tracie"},
	{"tracie_ws2812.vcd", "Tracie: three NeoPixel LEDs", "Tracie"},
	{"tracie_ir.vcd", "Tracie: an infrared remote button", "Tracie"},
	{"serial_2026-11-10_19-00-00.csv", "Tracie: serial log from an ESP32 with GPS", "Tracie"},
	{"meterie-2026-11-10_19-30.nst.csv", "Meterie: a battery-powered gadget for an hour", "Meterie"},
}


type lane struct {
	name string
	t    []float64
	v    []int
}

func (l *lane) set(t float64, v int) {
	if len(l.v) > 0 && l.v[len(l.v)-1] == v {
		return
	}
	l.t, l.v = append(l.t, t), append(l.v, v)
}

func writeVCD(lanes []*lane, end float64) string {
	var b strings.Builder
	b.WriteString("$date today $end\n$version Tracie 0.3.0-alpha by nrmluvdev $end\n$timescale 1 ns $end\n$scope module tracie $end\n")
	ids := "!\"#$%&'("
	for i, l := range lanes {
		fmt.Fprintf(&b, "$var wire 1 %c %s $end\n", ids[i], l.name)
	}
	b.WriteString("$upscope $end\n$enddefinitions $end\n#0\n$dumpvars\n")
	type ev struct {
		t float64
		i int
		v int
	}
	var evs []ev
	for i, l := range lanes {
		fmt.Fprintf(&b, "%d%c\n", l.v[0], ids[i])
		for k := 1; k < len(l.t); k++ {
			evs = append(evs, ev{l.t[k], i, l.v[k]})
		}
	}
	b.WriteString("$end\n")
	// sort
	for a := 1; a < len(evs); a++ {
		for c := a; c > 0 && evs[c].t < evs[c-1].t; c-- {
			evs[c], evs[c-1] = evs[c-1], evs[c]
		}
	}
	last := int64(-1)
	for _, e := range evs {
		ns := int64(math.Round(e.t * 1e9))
		if ns != last {
			fmt.Fprintf(&b, "#%d\n", ns)
			last = ns
		}
		fmt.Fprintf(&b, "%d%c\n", e.v, ids[e.i])
	}
	fmt.Fprintf(&b, "#%d\n", int64(end*1e9))
	return b.String()
}

func uartLane(name string, baud float64, start float64, text string) (*lane, float64) {
	l := &lane{name: name}
	l.set(0, 1)
	bit := 1 / baud
	t := start
	for _, c := range []byte(text) {
		l.set(t, 0)
		for k := 0; k < 8; k++ {
			l.set(t+bit*float64(k+1), int(c>>k)&1)
		}
		l.set(t+9*bit, 1)
		t += 10*bit + bit*0.5
	}
	return l, t
}

func i2cLanes(start float64, addr byte, data []byte, read bool) (*lane, *lane, float64) {
	scl, sda := &lane{name: "SCL"}, &lane{name: "SDA"}
	scl.set(0, 1)
	sda.set(0, 1)
	q := 2.5e-6 // 100 kHz: quarter of 10 µs
	t := start
	sda.set(t, 0) // START
	t += q
	scl.set(t, 0)
	sendBit := func(b int) {
		t += q
		sda.set(t, b)
		t += q
		scl.set(t, 1)
		t += 2 * q
		scl.set(t, 0)
	}
	ab := addr << 1
	if read {
		ab |= 1
	}
	all := append([]byte{ab}, data...)
	for _, by := range all {
		for k := 7; k >= 0; k-- {
			sendBit(int(by>>k) & 1)
		}
		sendBit(0) // ACK
	}
	t += q
	sda.set(t, 0)
	t += q
	scl.set(t, 1)
	t += q
	sda.set(t, 1) // STOP
	return scl, sda, t
}

func spiLanes(start float64, mosi, miso []byte) []*lane {
	ck, mo, mi, cs := &lane{name: "CH0"}, &lane{name: "CH1"}, &lane{name: "CH2"}, &lane{name: "CH3"}
	for _, l := range []*lane{ck, mo, mi} {
		l.set(0, 0)
	}
	cs.set(0, 1)
	h := 0.5e-6
	t := start
	cs.set(t, 0)
	t += h
	for i := range mosi {
		for k := 7; k >= 0; k-- {
			mo.set(t, int(mosi[i]>>k)&1)
			mi.set(t, int(miso[i]>>k)&1)
			t += h
			ck.set(t, 1)
			t += h
			ck.set(t, 0)
		}
	}
	t += h
	cs.set(t, 1)
	return []*lane{ck, mo, mi, cs}
}

func ws2812Lane(start float64, rgb [][3]byte) *lane {
	l := &lane{name: "CH0"}
	l.set(0, 0)
	t := start
	for _, c := range rgb {
		for _, by := range []byte{c[1], c[0], c[2]} {
			for k := 7; k >= 0; k-- {
				hi := 0.35e-6
				if by>>k&1 == 1 {
					hi = 0.7e-6
				}
				l.set(t, 1)
				l.set(t+hi, 0)
				t += 1.25e-6
			}
		}
	}
	return l
}

func necLane(start float64, addr, cmd byte) *lane {
	l := &lane{name: "IR"}
	l.set(0, 1)
	t := start
	l.set(t, 0)
	t += 9e-3
	l.set(t, 1)
	t += 4.5e-3
	v := uint32(addr) | uint32(^addr)<<8 | uint32(cmd)<<16 | uint32(^cmd)<<24
	for k := 0; k < 32; k++ {
		l.set(t, 0)
		t += 562e-6
		l.set(t, 1)
		if v>>k&1 == 1 {
			t += 1687e-6
		} else {
			t += 562e-6
		}
	}
	l.set(t, 0)
	t += 562e-6
	l.set(t, 1)
	return l
}

func servoLane(pw float64, n int) *lane {
	l := &lane{name: "CH2"}
	l.set(0, 0)
	for k := 0; k < n; k++ {
		t := 0.001 + float64(k)*0.02
		l.set(t, 1)
		l.set(t+pw, 0)
	}
	return l
}

func scopeCSV() string {
	var b strings.Builder
	b.WriteString("time_s,ch1_V,ch2_V\n")
	n := 20000
	dt := 1e-6 // 20 ms
	for i := 0; i < n; i++ {
		t := float64(i) * dt
		s := 1.65 + 1.5*math.Sin(2*math.Pi*440*t) + 0.01*math.Sin(float64(i)*1.7)
		// 3.3 V PWM at 1 kHz, 25%
		p := 0.02
		if math.Mod(t, 1e-3) < 0.25e-3 {
			p = 3.28
		}
		fmt.Fprintf(&b, "%.9g,%.5f,%.5f\n", t, s, p)
	}
	return b.String()
}

func meterTimeline() string {
	var b strings.Builder
	b.WriteString("# nrmluv-timeline v1\n# app=Meterie version=0.6.0-alpha author=nrmluvdev exported=2026-11-10T19:30:00Z\n# t0_unix_ns=1794079800000000000\n")
	b.WriteString("# channel id=1 name=CH1 unit=V kind=meter source=UT61E\n# channel id=2 name=CH2 unit=A kind=meter source=UT61E\n")
	b.WriteString("t_unix_ns,t_rel_s,channel_id,channel,value,unit,flags\n")
	t0 := int64(1794079800000000000)
	for k := 0; k < 3600; k += 10 {
		v := 4.1 - 0.6*float64(k)/3600
		a := 0.012
		if (k/10)%6 == 0 {
			a = 0.180
		}
		fmt.Fprintf(&b, "%d,%.3f,1,CH1,%.4f,V,DC\n", t0+int64(k)*1e9, float64(k), v)
		fmt.Fprintf(&b, "%d,%.3f,2,CH2,%.4f,A,DC\n", t0+int64(k)*1e9+5e8, float64(k)+0.5, a)
	}
	return b.String()
}

func serialLog() string {
	lines := []string{"ets Jun  8 2016 00:22:57", "rst:0x1 (POWERON_RESET),boot:0x13 (SPI_FAST_FLASH_BOOT)", "Connecting to WiFi...", "WiFi connected, IP 192.168.1.42", "temp=21.5 hum=40", "temp=21.6 hum=41", "temp=21.8 hum=41",
		"$GPGGA,123519,4807.038,N,01131.000,E,1,08,0.9,545.4,M,46.9,M,,*47", "Brownout detector was triggered", "rst:0xc (SW_CPU_RESET),boot:0x13", "temp=21.9 hum=42"}
	var b strings.Builder
	b.WriteString("timestamp,elapsed_s,direction,line,value1,value2\n")
	for i, l := range lines {
		vals := ""
		if strings.HasPrefix(l, "temp=") {
			var a, h float64
			fmt.Sscanf(l, "temp=%f hum=%f", &a, &h)
			vals = fmt.Sprintf("%g,%g", a, h)
		} else {
			vals = ","
		}
		fmt.Fprintf(&b, "2026-11-10T19:00:%02d,%.3f,RX,\"%s\",%s\n", i, float64(i)*1.5, strings.ReplaceAll(l, `"`, `""`), vals)
	}
	return b.String()
}

func demoCaptures() map[string]string {
	out := map[string]string{}
	// UART + I2C + servo in one Tracie capture
	tx, _ := uartLane("CH0", 115200, 0.0002, "Hello, Codie!\r\n")
	scl, sda, _ := i2cLanes(0.0005, 0x3C, []byte{0x00, 0xAF}, false)
	scl2, sda2, _ := i2cLanes(0.0015, 0x76, nil, false)
	for k := range scl2.t {
		if k > 0 {
			scl.set(scl2.t[k], scl2.v[k])
		}
	}
	for k := range sda2.t {
		if k > 0 {
			sda.set(sda2.t[k], sda2.v[k])
		}
	}
	out["tracie_uart_i2c.vcd"] = writeVCD([]*lane{tx, scl, sda, servoLane(1.5e-3, 6)}, 0.12)
	out["tracie_spi.vcd"] = writeVCD(spiLanes(0.0001, []byte{0x9F, 0, 0, 0}, []byte{0xFF, 0xEF, 0x40, 0x18}), 0.0002)
	out["tracie_ws2812.vcd"] = writeVCD([]*lane{ws2812Lane(0.0001, [][3]byte{{255, 0, 0}, {0, 255, 0}, {0, 0, 255}})}, 0.0005)
	out["tracie_ir.vcd"] = writeVCD([]*lane{necLane(0.001, 0x00, 0x45)}, 0.09)
	out["scope-20261110-190000.csv"] = scopeCSV()
	out["meterie-2026-11-10_19-30.nst.csv"] = meterTimeline()
	out["serial_2026-11-10_19-00-00.csv"] = serialLog()
	return out
}

