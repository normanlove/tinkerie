// Codie — Signals: the glossary, and explaining serial logs (the text a
// board prints over USB), such as boot messages, GPS sentences and errors.
// Part of the Tinkerie maker suite. Created by nrmluvdev.

package codie

import (
	"fmt"
	"strconv"
	"strings"
)

var sigGlossary = map[string]string{
	"Sample rate":               "How many times a second the scope or logic analyser measures the signal. To see a signal's shape you need at least 10 samples per cycle; to catch its edges, faster still.",
	"Peak to peak":              "The difference between the highest and the lowest point of a signal: how big the swing is.",
	"DC":                        "Direct current: a level that stays put, like a battery or a power supply rail. Small wobbles on top of it are ripple and noise.",
	"Ripple":                    "A small repeating wobble on a supply. From a mains supply it's at 50/60 or 100/120 Hz; from a switching regulator it's much faster. Too much can upset sensitive chips.",
	"Frequency":                 "How many times a second something repeats, in hertz (Hz). The time for one repeat is the period: period = 1 ÷ frequency.",
	"Sine wave":                 "The smooth wave that mains electricity, sound and radio are made of. Its RMS value is the peak divided by √2.",
	"Aliasing":                  "When a signal is sampled too slowly, it can look like a slower, different signal. The fix is to sample faster.",
	"Time constant":             "How quickly a resistor-capacitor (RC) circuit charges: it gets 63% of the way in R × C seconds and is practically finished after 5 R × C.",
	"Noise":                     "Random, unwanted wobble on a signal. Short wires, good grounding, shielding and filtering reduce it.",
	"Logic levels":              "Digital signals are LOW (near 0 V) or HIGH (near the supply: 3.3 V or 5 V). A chip reads anything above about 70% of its supply as HIGH, and below about 30% as LOW.",
	"Rise time":                 "How long an edge takes to go from 10% to 90% of the way up. Slow edges come from long wires, capacitance or weak pull-up resistors.",
	"Overshoot and ringing":     "An edge that shoots past its level and wobbles before settling. A little is normal; a lot can cause false clocks. The scope probe's long ground lead often makes it look worse than it is.",
	"Duty cycle":                "The fraction of each cycle a signal is HIGH. 50% is a square wave; 10% is a short blip; 90% is nearly always on.",
	"PWM":                       "Pulse-width modulation: switching fully on and off quickly and changing how long it's on, to dim LEDs, set motor speed or make an 'analog' voltage from a digital pin.",
	"Servo signal":              "A pulse every 20 ms (50 Hz) whose width sets a servo's angle: about 1 ms for one end, 1.5 ms for the middle and 2 ms for the other end.",
	"Clock":                     "A steady square wave that paces other circuits: each tick tells a chip when to take the next step or read the next bit.",
	"UART serial":               "The simplest serial link: TX of one device to RX of the other, no clock. Each byte is a LOW start bit, 8 data bits (lowest first) and a HIGH stop bit. Used by Serial.print(), GPS and Bluetooth modules.",
	"Baud rate":                 "The speed of a serial link in bits per second. 9600 and 115200 are the most common. Both ends must use the same baud rate or the text turns into rubbish.",
	"I2C":                       "A two-wire bus (SCL clock, SDA data) that connects many chips. Each has a 7-bit address; the controller says START, the address, read or write, then the data. Both wires need pull-up resistors.",
	"I2C address":               "The number a chip answers to on an I2C bus, such as 0x3C for most small OLED displays or 0x76 for a BME280. Some chips let you pick between two or more with a pin.",
	"ACK and NACK":              "After every I2C byte, the receiver pulls SDA LOW to say 'got it' (ACK). If it doesn't (NACK), nobody heard, or the controller is saying 'that's the last byte I want'.",
	"SPI":                       "A fast bus with a clock (SCK), data out (MOSI), data in (MISO) and a chip select (CS) per chip. Data goes both ways at once, one bit per clock edge.",
	"Chip select":               "The SPI wire that goes LOW to choose which chip listens. Each chip on the bus has its own.",
	"WS2812 addressable LEDs":   "LEDs with a tiny chip inside (NeoPixels). One data wire carries 24 bits per LED (green, red, blue); each LED keeps its own colour and passes the rest down the strip.",
	"Infrared remotes (NEC)":    "TV-style remotes flash an IR LED at 38 kHz. An IR receiver turns that into pulses; the NEC code sends an address (which device) and a command (which button).",
	"1-Wire":                    "A bus that needs just one data wire plus ground, with a 4.7 kΩ pull-up. Every device has a unique 64-bit ID. The DS18B20 temperature sensor is the best-known 1-Wire part.",
	"Voltage":                   "The 'push' that drives current through a circuit, in volts (V). Measured between two points: usually a signal and ground.",
	"Current draw":              "How much current a circuit uses, in amps (A) or milliamps (mA). Battery life is roughly the battery's capacity (mAh) divided by the average current (mA).",
	"Power and energy":          "Power (watts) is voltage × current: how hard something is working right now. Energy (watt-hours) is power over time: how much it used.",
	"Serial log":                "Text a board sends to the computer over USB (or a UART), one line at a time. It's how most projects report what they're doing and print errors.",
	"NMEA GPS sentences":        "The text GPS modules send: lines starting with $GP, $GN or $GL. GGA has the position, height and number of satellites; RMC has the time, date, position and speed.",
	"Brownout":                  "When the supply voltage dips too low for a moment, the chip resets itself to stay safe. Usually a weak USB cable or power supply, or a big load like Wi-Fi or a motor starting.",
	"Watchdog reset":            "A timer that restarts the chip if the program gets stuck. On an ESP32 it often means a loop that never waits (add delay() or yield()).",
}

// ---------------------------------------------------------------- serial logs

func (x *sigCtx) explainSerialLog() {
	L := x.c.Serial
	n := len(L)
	dur := L[n-1].T - L[0].T
	x.concept("Serial log", -1)
	dirs := map[string]int{}
	for _, l := range L {
		dirs[strings.ToUpper(l.Dir)]++
	}
	rate := 0.0
	if dur > 0 {
		rate = float64(n) / dur
	}
	facts := [][2]string{{"Lines", fmt.Sprintf("%d over %s", n, humanDur(dur))}, {"Lines per second", fmt.Sprintf("%.1f", rate)}}
	var dl []string
	for k, v := range dirs {
		dl = append(dl, fmt.Sprintf("%s %d", k, v))
	}
	if len(dl) > 0 {
		facts = append(facts, [2]string{"Direction", strings.Join(dl, ", ")})
	}
	what := fmt.Sprintf("**%d lines** of text from the serial port over %s.", n, humanDur(dur))
	why := "RX lines came from the board; TX lines were typed or sent to it. Codie looks for well-known messages below."
	x.card(SigCard{Ch: -1, Tag: "serial", Title: "Serial log", What: what, Why: why, Facts: facts, T0: L[0].T, T1: L[n-1].T})

	type hit struct {
		title, what, why string
		warn             bool
		t                float64
		count            int
	}
	hits := map[string]*hit{}
	var order []string
	add := func(key, title, what, why string, warn bool, t float64) {
		if h := hits[key]; h != nil {
			h.count++
			return
		}
		hits[key] = &hit{title, what, why, warn, t, 1}
		order = append(order, key)
	}
	var gps []string
	for _, l := range L {
		s := strings.TrimSpace(l.Text)
		low := strings.ToLower(s)
		switch {
		case strings.Contains(s, "rst:0x"):
			reason := ""
			if i := strings.Index(s, "("); i > 0 && strings.Contains(s, "rst:") {
				reason = strings.Trim(s[i:], "() ,")
				if j := strings.Index(reason, ")"); j > 0 {
					reason = reason[:j]
				}
			}
			why := "The ESP32 or ESP8266 just started (or restarted). This line is printed by its built-in ROM before your program runs."
			switch {
			case strings.Contains(reason, "POWERON"):
				why += " POWERON_RESET means it was just powered up: normal."
			case strings.Contains(reason, "BROWNOUT"):
				why += " BROWNOUT means the supply voltage dipped too low."
			case strings.Contains(reason, "WDT"):
				why += " A WDT reset means the watchdog restarted it because the program got stuck."
			case strings.Contains(reason, "SW_CPU") || strings.Contains(reason, "SW_RESET"):
				why += " A software reset: the program restarted itself (often after a crash)."
			}
			add("boot:"+reason, "ESP boot: "+strings.ToLower(strings.ReplaceAll(strings.TrimSuffix(reason, "_RESET"), "_", " ")), "An **ESP32/ESP8266 boot message**: `"+trunc(s, 80)+"`.", why, strings.Contains(reason, "BROWNOUT") || strings.Contains(reason, "WDT"), l.T)
		case strings.Contains(low, "brownout"):
			x.concept("Brownout", -1)
			add("brownout", "Brownout", "The board reported a **brownout**: `"+trunc(s, 80)+"`.", sigGlossary["Brownout"]+" Try a shorter or better USB cable, a stronger supply, or a capacitor (100–470 µF) across the supply.", true, l.T)
		case strings.Contains(s, "Guru Meditation") || strings.Contains(s, "Backtrace:") || strings.Contains(low, "panic"):
			add("crash", "Crash", "The program **crashed**: `"+trunc(s, 90)+"`.", "This is an ESP32 crash report. The Backtrace addresses point to the code that failed; the line before usually names the problem (LoadProhibited often means using an empty pointer, StoreProhibited writing through one). The board then restarts.", true, l.T)
		case strings.Contains(low, "task_wdt") || strings.Contains(low, "watchdog"):
			x.concept("Watchdog reset", -1)
			add("wdt", "Watchdog", "The **watchdog** fired: `"+trunc(s, 80)+"`.", sigGlossary["Watchdog reset"], true, l.T)
		case strings.HasPrefix(s, "Traceback (most recent call last)") || strings.HasPrefix(s, "Traceback"):
			add("pytrace", "Python error", "A **MicroPython/CircuitPython error** (Traceback).", "The lines after it show the file and line where it happened, and the last line names the error (NameError, OSError and so on).", true, l.T)
		case strings.HasPrefix(s, ">>>") || strings.Contains(s, "MicroPython v"):
			add("repl", "MicroPython", "This board runs **MicroPython** (its >>> prompt or banner appears).", "The >>> prompt is the REPL: type Python and it runs straight away on the board.", false, l.T)
		case strings.HasPrefix(s, "$G") && len(s) > 6 && strings.Contains(s, ","):
			x.concept("NMEA GPS sentences", -1)
			add("nmea", "GPS module", "These are **GPS sentences (NMEA)**.", sigGlossary["NMEA GPS sentences"], false, l.T)
			if g := nmeaFix(s); g != "" {
				gps = append(gps, g)
			}
		case strings.HasPrefix(s, "AT") && (len(s) == 2 || s[2] == '+' || s[2] == '\r'):
			add("at", "AT commands", "**AT commands** were sent: `"+trunc(s, 60)+"`.", "AT commands control modems and wireless modules (ESP-01 Wi-Fi, HC-05 Bluetooth, SIM800 GSM). The module answers OK or ERROR.", false, l.T)
		case strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}"):
			add("json", "JSON", "Some lines are **JSON** data: `"+trunc(s, 60)+"`.", "JSON is a common way for a board to send structured readings that a program on the computer (or a web page) can read easily.", false, l.T)
		case strings.Contains(low, "error") || strings.Contains(low, "failed") || strings.Contains(low, "not found"):
			add("err:"+trunc(low, 30), "Error message", "An **error message**: `"+trunc(s, 90)+"`.", "The program reported a problem. The words usually say what: a sensor not found (check its wiring and address), a Wi-Fi connection failing (check the name and password) and so on.", true, l.T)
		case strings.Contains(low, "wifi") && (strings.Contains(low, "connect") || strings.Contains(low, "ip")):
			add("wifi", "Wi-Fi", "Wi-Fi messages: `"+trunc(s, 80)+"`.", "The board is joining a Wi-Fi network. Once it prints an IP address it's connected.", false, l.T)
		}
	}
	for _, k := range order {
		h := hits[k]
		w := h.what
		if h.count > 1 {
			w += fmt.Sprintf(" (seen %d times)", h.count)
		}
		if k == "nmea" && len(gps) > 0 {
			w += " The latest fix: " + gps[len(gps)-1] + "."
		}
		x.card(SigCard{Ch: -1, Tag: "serial", Title: h.title, What: w, Why: h.why, T0: h.t, T1: h.t})
		if h.warn {
			x.warn(SigWarn{Title: h.title, Text: strings.ReplaceAll(strings.ReplaceAll(w, "**", ""), "`", "") + " " + h.why, Ch: -1, T0: h.t, T1: h.t})
		}
	}
	// numbers in each line become channels too (Tracie's value columns)
	if len(x.c.Chans) > 0 {
		x.card(SigCard{Ch: -1, Tag: "serial", Title: "Numbers in the lines", What: fmt.Sprintf("Tracie found numbers in the lines and kept them as **%d value column%s**, explained below as readings over time.", len(x.c.Chans), map[bool]string{true: "", false: "s"}[len(x.c.Chans) == 1]), Why: "Boards often print readings like 'temp=23.5 hum=41': each number becomes a line on the chart.", T0: L[0].T, T1: L[n-1].T})
	}
}

func trunc(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// nmeaFix describes a GGA or RMC sentence.
func nmeaFix(s string) string {
	f := strings.Split(strings.SplitN(s, "*", 2)[0], ",")
	if len(f) < 7 {
		return ""
	}
	typ := f[0][3:]
	coord := func(v, h string, deg int) string {
		if len(v) < deg+2 {
			return ""
		}
		d, _ := strconv.Atoi(v[:deg])
		m, _ := strconv.ParseFloat(v[deg:], 64)
		x := float64(d) + m/60
		if h == "S" || h == "W" {
			x = -x
		}
		return fmt.Sprintf("%.5f", x)
	}
	switch typ {
	case "GGA":
		if len(f) < 10 || f[6] == "0" || f[2] == "" {
			return "no fix yet (still looking for satellites)"
		}
		return fmt.Sprintf("%s, %s with %s satellites, %s m above sea level", coord(f[2], f[3], 2), coord(f[4], f[5], 3), strings.TrimLeft(f[7], "0"), f[9])
	case "RMC":
		if len(f) < 8 || f[2] != "A" {
			return "no fix yet (still looking for satellites)"
		}
		kn, _ := strconv.ParseFloat(f[7], 64)
		return fmt.Sprintf("%s, %s moving at %.1f km/h", coord(f[3], f[4], 2), coord(f[5], f[6], 3), kn*1.852)
	}
	return ""
}
