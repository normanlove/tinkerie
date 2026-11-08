package geom

import (
	"encoding/json"
	"sort"
)

// BoardSpec describes a circuit board, screen or module so mounts, cases, lids and
// bezels can be generated for it. The board is centred on X/Y; Z=0 is the board's
// bottom surface. X is the board's long side.
type BoardSpec struct {
	Name    string    `json:"name"`
	W       float64   `json:"w"`     // X size
	D       float64   `json:"d"`     // Y size
	T       float64   `json:"t"`     // board thickness
	Round   bool      `json:"round"` // round board (rings, round displays)
	Holes   []BHole   `json:"holes"`
	Above   float64   `json:"above"`  // tallest part above the top surface
	Below   float64   `json:"below"`  // pins / parts under the board
	Window  *BWindow  `json:"window"` // screen or sensor window (for lids / bezels)
	Ports   []BPort   `json:"ports"`  // connectors at the edges (for case cut-outs)
	Cut     *BWindow  `json:"cut"`    // body that passes through the mount (servos, motors)
	Circles []BWindow `json:"circles"`
	Outline []P2      `json:"outline,omitempty"` // exact footprint (from scanned models)
	Note    string    `json:"note,omitempty"`
	Family  string    `json:"family"`
}

type BHole struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	D float64 `json:"d"`
}

// BWindow is a rectangle (or circle when Round) at X/Y, H = height above the board top.
type BWindow struct {
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	W     float64 `json:"w"`
	D     float64 `json:"d"`
	H     float64 `json:"h"`
	Round bool    `json:"round"`
}

// BPort is a connector on an edge: Edge is "+x", "-x", "+y" or "-y"; Pos is the centre
// along that edge (from board centre); Z0..Z1 above the board's top surface.
type BPort struct {
	Edge           string
	Pos, W, Z0, Z1 float64
	Name           string
}

func corners(w, d, inset, hd float64) []BHole {
	x, y := w/2-inset, d/2-inset
	return []BHole{{-x, -y, hd}, {x, -y, hd}, {x, y, hd}, {-x, y, hd}}
}

// holes from corner-origin coordinates (as in datasheets)
func fromCorner(w, d, hd float64, pts ...[2]float64) []BHole {
	var o []BHole
	for _, p := range pts {
		o = append(o, BHole{p[0] - w/2, p[1] - d/2, hd})
	}
	return o
}

var boardDB = map[string]*BoardSpec{}
var boardOrder []string

func addBoard(b *BoardSpec) {
	if b.T == 0 {
		b.T = 1.6
	}
	boardDB[b.Name] = b
	boardOrder = append(boardOrder, b.Name)
}

func usb(edge string, pos float64, kind string) BPort {
	switch kind {
	case "C":
		return BPort{edge, pos, 9.2, 0, 3.4, "USB-C"}
	case "B":
		return BPort{edge, pos, 12.5, 0, 11, "USB-B"}
	case "mini":
		return BPort{edge, pos, 8, 0, 4.2, "Mini-USB"}
	}
	return BPort{edge, pos, 8, 0, 3.2, "Micro-USB"}
}

func init() {
	// ---------------- Raspberry Pi ----------------
	piHoles := fromCorner(85, 56, 2.75, [2]float64{3.5, 3.5}, [2]float64{61.5, 3.5}, [2]float64{3.5, 52.5}, [2]float64{61.5, 52.5})
	addBoard(&BoardSpec{Family: "Raspberry Pi", Name: "Raspberry Pi 5", W: 85, D: 56, T: 1.4, Holes: piHoles, Above: 16, Below: 2.5,
		Ports: []BPort{usb("-y", -31.3, "C"), {"-y", -16.5, 7.5, 0, 3.6, "micro-HDMI"}, {"-y", -3, 7.5, 0, 3.6, "micro-HDMI"}, {"+x", -17.75, 16, 0, 13.6, "Ethernet"}, {"+x", 1.1, 14.5, 0, 16, "USB"}, {"+x", 19, 14.5, 0, 16, "USB"}}})
	addBoard(&BoardSpec{Family: "Raspberry Pi", Name: "Raspberry Pi 4 Model B", W: 85, D: 56, T: 1.4, Holes: piHoles, Above: 16, Below: 2.5,
		Ports: []BPort{usb("-y", -31.3, "C"), {"-y", -16.5, 7.5, 0, 3.6, "micro-HDMI"}, {"-y", -3, 7.5, 0, 3.6, "micro-HDMI"}, {"-y", 11, 7, 0, 6, "Audio"}, {"+x", -19, 14.5, 0, 16, "USB"}, {"+x", -1, 14.5, 0, 16, "USB"}, {"+x", 17.75, 16, 0, 13.6, "Ethernet"}}})
	addBoard(&BoardSpec{Family: "Raspberry Pi", Name: "Raspberry Pi 3 Model B / B+", W: 85, D: 56, T: 1.4, Holes: piHoles, Above: 16, Below: 2.5,
		Ports: []BPort{usb("-y", -31.9, "micro"), {"-y", -10.5, 15.5, 0, 6.5, "HDMI"}, {"-y", 11, 7, 0, 6, "Audio"}, {"+x", -17.75, 16, 0, 13.6, "Ethernet"}, {"+x", 1.1, 14.5, 0, 16, "USB"}, {"+x", 19, 14.5, 0, 16, "USB"}}})
	addBoard(&BoardSpec{Family: "Raspberry Pi", Name: "Raspberry Pi 3 Model A+", W: 65, D: 56, T: 1.4, Holes: fromCorner(65, 56, 2.75, [2]float64{3.5, 3.5}, [2]float64{61.5, 3.5}, [2]float64{3.5, 52.5}, [2]float64{61.5, 52.5}), Above: 8, Below: 2.5,
		Ports: []BPort{usb("-y", -21.9, "micro"), {"-y", -0.5, 15.5, 0, 6.5, "HDMI"}, {"+x", 3, 14.5, 0, 7.5, "USB"}}})
	addBoard(&BoardSpec{Family: "Raspberry Pi", Name: "Raspberry Pi Zero / Zero W / Zero 2 W", W: 65, D: 30, T: 1.4, Holes: fromCorner(65, 30, 2.75, [2]float64{3.5, 3.5}, [2]float64{61.5, 3.5}, [2]float64{3.5, 26.5}, [2]float64{61.5, 26.5}), Above: 5, Below: 1.5,
		Ports: []BPort{{"-y", -20.1, 11.5, 0, 3.6, "mini-HDMI"}, usb("-y", 8.9, "micro"), usb("-y", 21.5, "micro"), {"+x", 0, 17, 0, 1.6, "Camera"}}})
	addBoard(&BoardSpec{Family: "Raspberry Pi", Name: "Raspberry Pi Pico / Pico W / Pico 2", W: 51, D: 21, T: 1.0, Holes: fromCorner(51, 21, 2.1, [2]float64{2, 4.8}, [2]float64{2, 16.2}, [2]float64{49, 4.8}, [2]float64{49, 16.2}), Above: 3.5, Below: 2.5,
		Ports: []BPort{usb("-x", 0, "micro")}})
	addBoard(&BoardSpec{Family: "Raspberry Pi", Name: "Raspberry Pi Compute Module 4", W: 55, D: 40, T: 1.2, Holes: corners(55, 40, 3.5, 2.75), Above: 3, Below: 2.7})
	addBoard(&BoardSpec{Family: "Raspberry Pi", Name: "Raspberry Pi Camera Module (v2 / v3)", W: 25, D: 24, T: 1, Holes: []BHole{{-10.5, 10, 2.2}, {10.5, 10, 2.2}, {-10.5, -2.5, 2.2}, {10.5, -2.5, 2.2}}, Above: 5, Below: 1,
		Window: &BWindow{0, 2.5, 8.5, 8.5, 5, false}, Ports: []BPort{{"-y", 0, 17, 0, 2, "Ribbon"}}})

	// ---------------- Arduino ----------------
	unoHoles := fromCorner(68.6, 53.3, 3.2, [2]float64{13.97, 2.54}, [2]float64{15.24, 50.8}, [2]float64{66.04, 7.62}, [2]float64{66.04, 35.56})
	addBoard(&BoardSpec{Family: "Arduino", Name: "Arduino Uno (R3 / R4) / Leonardo", W: 68.6, D: 53.3, Holes: unoHoles, Above: 12, Below: 2.5,
		Ports: []BPort{{"-x", 11.5, 12.5, 0, 11, "USB-B"}, {"-x", -19, 9.5, 0, 11, "DC jack"}}})
	addBoard(&BoardSpec{Family: "Arduino", Name: "Arduino Mega 2560", W: 101.6, D: 53.3, Holes: fromCorner(101.6, 53.3, 3.2, [2]float64{13.97, 2.54}, [2]float64{15.24, 50.8}, [2]float64{66.04, 7.62}, [2]float64{66.04, 35.56}, [2]float64{90.17, 50.8}, [2]float64{96.52, 2.54}), Above: 12, Below: 2.5,
		Ports: []BPort{{"-x", 11.5, 12.5, 0, 11, "USB-B"}, {"-x", -19, 9.5, 0, 11, "DC jack"}}})
	addBoard(&BoardSpec{Family: "Arduino", Name: "Arduino Nano / Nano Every / Nano ESP32", W: 45, D: 18, Holes: corners(45, 18, 1.3, 1.7), Above: 4.5, Below: 8, Ports: []BPort{usb("-x", 0, "mini")}})
	addBoard(&BoardSpec{Family: "Arduino", Name: "Arduino Micro", W: 48, D: 18, Above: 4, Below: 8, Ports: []BPort{usb("-x", 0, "micro")}})
	addBoard(&BoardSpec{Family: "Arduino", Name: "Arduino Pro Mini", W: 33, D: 18, Above: 2.5, Below: 8})
	addBoard(&BoardSpec{Family: "Arduino", Name: "Arduino MKR series", W: 61.5, D: 25, Holes: corners(61.5, 25, 2.5, 2.2), Above: 4, Below: 8, Ports: []BPort{usb("-x", 0, "micro")}})

	// ---------------- ESP32 / ESP8266 / other MCUs ----------------
	addBoard(&BoardSpec{Family: "ESP & other microcontrollers", Name: "ESP32 DevKit V1 (30-pin)", W: 51.5, D: 28.3, Above: 4.5, Below: 8.5, Ports: []BPort{usb("-x", 0, "micro")}})
	addBoard(&BoardSpec{Family: "ESP & other microcontrollers", Name: "ESP32 DevKitC (38-pin)", W: 55, D: 28, Above: 4.5, Below: 8.5, Ports: []BPort{usb("-x", 0, "micro")}})
	addBoard(&BoardSpec{Family: "ESP & other microcontrollers", Name: "ESP32-S3 DevKitC-1", W: 62.7, D: 25.4, Above: 4.5, Below: 8.5, Ports: []BPort{usb("-x", -6, "C"), usb("-x", 6, "C")}})
	addBoard(&BoardSpec{Family: "ESP & other microcontrollers", Name: "ESP32-CAM", W: 40, D: 27, Above: 6, Below: 8.5, Window: &BWindow{-5, 0, 9, 9, 6, false}})
	addBoard(&BoardSpec{Family: "ESP & other microcontrollers", Name: "ESP32-C3 / S3 SuperMini", W: 22.5, D: 18, Above: 3.3, Below: 8, Ports: []BPort{usb("-x", 0, "C")}})
	addBoard(&BoardSpec{Family: "ESP & other microcontrollers", Name: "NodeMCU ESP8266 (v2, Amica)", W: 48, D: 26, Holes: corners(48, 26, 2.5, 3), Above: 4.5, Below: 8.5, Ports: []BPort{usb("-x", 0, "micro")}})
	addBoard(&BoardSpec{Family: "ESP & other microcontrollers", Name: "NodeMCU ESP8266 (v3, LoLin)", W: 58, D: 31, Holes: corners(58, 31, 2.5, 3), Above: 4.5, Below: 8.5, Ports: []BPort{usb("-x", 0, "micro")}})
	addBoard(&BoardSpec{Family: "ESP & other microcontrollers", Name: "Wemos D1 mini", W: 34.2, D: 25.6, Above: 4.5, Below: 8, Ports: []BPort{usb("-x", 3, "micro")}})
	addBoard(&BoardSpec{Family: "ESP & other microcontrollers", Name: "Seeed XIAO (all versions)", W: 21, D: 17.8, Above: 3.3, Below: 2, Ports: []BPort{usb("-x", 0, "C")}})
	addBoard(&BoardSpec{Family: "ESP & other microcontrollers", Name: "Adafruit Feather", W: 50.8, D: 22.86, Holes: corners(50.8, 22.86, 2.54, 2.5), Above: 4.5, Below: 8, Ports: []BPort{usb("-x", 0, "micro")}})
	addBoard(&BoardSpec{Family: "ESP & other microcontrollers", Name: "Adafruit QT Py", W: 21.8, D: 17.8, Above: 3.3, Below: 2, Ports: []BPort{usb("-x", 0, "C")}})
	addBoard(&BoardSpec{Family: "ESP & other microcontrollers", Name: "RP2040-Zero", W: 23.5, D: 18, Above: 3.3, Below: 2, Ports: []BPort{usb("-x", 0, "C")}})
	addBoard(&BoardSpec{Family: "ESP & other microcontrollers", Name: "STM32 Blue Pill / Black Pill", W: 53, D: 22.5, Above: 5, Below: 8.5, Ports: []BPort{usb("-x", 0, "micro")}})
	addBoard(&BoardSpec{Family: "ESP & other microcontrollers", Name: "Teensy 4.0", W: 35.6, D: 17.8, Above: 3, Below: 8, Ports: []BPort{usb("-x", 0, "micro")}})
	addBoard(&BoardSpec{Family: "ESP & other microcontrollers", Name: "Teensy 4.1", W: 61, D: 17.8, Above: 4, Below: 8, Ports: []BPort{usb("-x", 0, "micro"), {"+x", 0, 12, 0, 2, "SD card"}}})
	addBoard(&BoardSpec{Family: "ESP & other microcontrollers", Name: "BBC micro:bit", W: 51.6, D: 42, T: 1.2, Above: 5, Below: 3, Ports: []BPort{usb("+y", -14, "micro")}})

	// ---------------- screens ----------------
	addBoard(&BoardSpec{Family: "Screens", Name: "0.96\" OLED (SSD1306, 128×64)", W: 27.3, D: 27.8, T: 1.2, Holes: corners(27.3, 27.8, 2, 2.1), Above: 3, Below: 9, Window: &BWindow{0, 1.5, 23, 12, 1.6, false}})
	addBoard(&BoardSpec{Family: "Screens", Name: "1.3\" OLED (SH1106, 128×64)", W: 35.4, D: 33.5, T: 1.2, Holes: corners(35.4, 33.5, 2.1, 3), Above: 3, Below: 9, Window: &BWindow{0, 1.5, 30, 16, 1.6, false}})
	addBoard(&BoardSpec{Family: "Screens", Name: "0.91\" OLED (128×32)", W: 38, D: 12, T: 1.2, Above: 2.5, Below: 9, Window: &BWindow{-2, 0, 24, 7, 1.5, false}})
	addBoard(&BoardSpec{Family: "Screens", Name: "1.8\" TFT (ST7735)", W: 34.5, D: 56.5, Holes: corners(34.5, 56.5, 2.5, 3), Above: 3, Below: 9, Window: &BWindow{0, -2, 29, 37, 2.6, false}})
	addBoard(&BoardSpec{Family: "Screens", Name: "2.4\" TFT SPI (ILI9341)", W: 77, D: 42.7, Holes: corners(77, 42.7, 2.5, 3), Above: 4, Below: 9, Window: &BWindow{-4, 0, 50, 37, 3.6, false}})
	addBoard(&BoardSpec{Family: "Screens", Name: "2.8\" TFT SPI (ILI9341)", W: 86, D: 50, Holes: corners(86, 50, 2.5, 3), Above: 4, Below: 9, Window: &BWindow{-4, 0, 59, 45, 3.6, false}})
	addBoard(&BoardSpec{Family: "Screens", Name: "3.5\" TFT SPI (ILI9488)", W: 98, D: 55, Holes: corners(98, 55, 2.5, 3), Above: 4, Below: 9, Window: &BWindow{-5, 0, 75, 50, 3.8, false}})
	addBoard(&BoardSpec{Family: "Screens", Name: "CYD 2.8\" (ESP32-2432S028R)", W: 86, D: 50, Holes: corners(86, 50, 3, 3.2), Above: 4.2, Below: 6, Window: &BWindow{-3, 0, 59, 45, 4.2, false}, Ports: []BPort{usb("+x", -8, "micro")}, Note: "Cheap Yellow Display"})
	addBoard(&BoardSpec{Family: "Screens", Name: "CYD 3.5\" (ESP32-3248S035)", W: 98, D: 55, Holes: corners(98, 55, 3, 3.2), Above: 4.5, Below: 6, Window: &BWindow{-2, 0, 75, 50, 4.5, false}, Ports: []BPort{usb("+x", -8, "micro")}})
	addBoard(&BoardSpec{Family: "Screens", Name: "1.28\" round TFT (GC9A01)", W: 38, D: 38, Round: true, Above: 3, Below: 9, Window: &BWindow{0, 0, 32.4, 32.4, 2, true}})
	addBoard(&BoardSpec{Family: "Screens", Name: "16×2 character LCD (HD44780)", W: 80, D: 36, Holes: corners(80, 36, 2.5, 3), Above: 9.5, Below: 9, Window: &BWindow{0, 0, 71.5, 24.5, 9.5, false}})
	addBoard(&BoardSpec{Family: "Screens", Name: "20×4 character LCD (HD44780)", W: 98, D: 60, Holes: corners(98, 60, 2.5, 3.2), Above: 9.5, Below: 9, Window: &BWindow{0, 0, 77, 26, 9.5, false}})
	addBoard(&BoardSpec{Family: "Screens", Name: "Nokia 5110 LCD", W: 45, D: 45, Holes: corners(45, 45, 2.5, 2.5), Above: 4, Below: 9, Window: &BWindow{0, 2, 39, 30, 4, false}})
	addBoard(&BoardSpec{Family: "Screens", Name: "2.9\" e-paper module", W: 89.5, D: 38, Holes: corners(89.5, 38, 2.5, 2.5), Above: 1.2, Below: 9, Window: &BWindow{0, 0, 67, 29.5, 1.2, false}})
	addBoard(&BoardSpec{Family: "Screens", Name: "7-segment 4-digit (TM1637)", W: 42, D: 24, Holes: corners(42, 24, 2.2, 2.2), Above: 7, Below: 9, Window: &BWindow{0, 0, 30.5, 14.5, 7, false}})

	// ---------------- LEDs ----------------
	addBoard(&BoardSpec{Family: "LED modules", Name: "WS2812 LED ring (12 LEDs, 37 mm)", W: 37, D: 37, Round: true, Above: 1.7, Below: 1, Window: &BWindow{0, 0, 37, 37, 1.7, true}})
	addBoard(&BoardSpec{Family: "LED modules", Name: "WS2812 LED ring (16 LEDs, 44.5 mm)", W: 44.5, D: 44.5, Round: true, Above: 1.7, Below: 1, Window: &BWindow{0, 0, 44.5, 44.5, 1.7, true}})
	addBoard(&BoardSpec{Family: "LED modules", Name: "WS2812 LED ring (24 LEDs, 66 mm)", W: 66, D: 66, Round: true, Above: 1.7, Below: 1, Window: &BWindow{0, 0, 66, 66, 1.7, true}})
	addBoard(&BoardSpec{Family: "LED modules", Name: "WS2812 LED stick (8 LEDs)", W: 51, D: 10, Above: 1.7, Below: 1, Window: &BWindow{0, 0, 49, 6, 1.7, false}})
	addBoard(&BoardSpec{Family: "LED modules", Name: "WS2812 8×8 LED matrix panel", W: 65, D: 65, Above: 1.7, Below: 1, Window: &BWindow{0, 0, 64, 64, 1.7, false}})
	addBoard(&BoardSpec{Family: "LED modules", Name: "MAX7219 8×8 LED matrix", W: 32, D: 32, Above: 9, Below: 12, Window: &BWindow{0, 0, 32, 32, 9, false}})
	addBoard(&BoardSpec{Family: "LED modules", Name: "MAX7219 4-in-1 LED matrix", W: 128, D: 32, Holes: corners(128, 32, 2.5, 3), Above: 9, Below: 12, Window: &BWindow{0, 0, 128, 32, 9, false}})

	// ---------------- sensors & modules ----------------
	addBoard(&BoardSpec{Family: "Sensors & modules", Name: "HC-SR04 ultrasonic sensor", W: 45, D: 20, Holes: corners(45, 20, 2, 2), Above: 12, Below: 9, Circles: []BWindow{{-13, 0, 16.5, 16.5, 12, true}, {13, 0, 16.5, 16.5, 12, true}}})
	addBoard(&BoardSpec{Family: "Sensors & modules", Name: "HC-SR501 PIR motion sensor", W: 32.3, D: 24.3, Holes: []BHole{{-14, 0, 2.1}, {14, 0, 2.1}}, Above: 13, Below: 6, Circles: []BWindow{{0, 0, 23.2, 23.2, 13, true}}})
	addBoard(&BoardSpec{Family: "Sensors & modules", Name: "Relay module (1 channel)", W: 50, D: 26, Holes: corners(50, 26, 2.5, 3), Above: 19, Below: 3})
	addBoard(&BoardSpec{Family: "Sensors & modules", Name: "Relay module (4 channel)", W: 75, D: 55, Holes: corners(75, 55, 3, 3.1), Above: 19, Below: 3})
	addBoard(&BoardSpec{Family: "Sensors & modules", Name: "L298N motor driver", W: 43, D: 43, Holes: corners(43, 43, 3, 3.1), Above: 27, Below: 3})
	addBoard(&BoardSpec{Family: "Sensors & modules", Name: "LM2596 buck converter", W: 43, D: 21, Holes: []BHole{{-15, 7.6, 3}, {15, -7.6, 3}}, Above: 14, Below: 3})
	addBoard(&BoardSpec{Family: "Sensors & modules", Name: "MP1584 mini buck converter", W: 22, D: 17, Above: 4, Below: 1})
	addBoard(&BoardSpec{Family: "Sensors & modules", Name: "TP4056 USB-C charger", W: 26, D: 17, Above: 3.5, Below: 1, Ports: []BPort{usb("-x", 0, "C")}})
	addBoard(&BoardSpec{Family: "Sensors & modules", Name: "MPU6050 (GY-521)", W: 21, D: 16, Holes: []BHole{{-7.6, 5, 3}, {7.6, 5, 3}}, Above: 2, Below: 9})
	addBoard(&BoardSpec{Family: "Sensors & modules", Name: "Micro-SD card module", W: 42, D: 24, Holes: corners(42, 24, 2.5, 3), Above: 3, Below: 9, Ports: []BPort{{"+x", 0, 15, 0, 3, "SD card"}}})
	addBoard(&BoardSpec{Family: "Sensors & modules", Name: "Joystick module (KY-023)", W: 34, D: 26, Holes: []BHole{{-13.3, -10.2, 3.2}, {13.3, -10.2, 3.2}, {-13.3, 10.2, 3.2}, {13.3, 10.2, 3.2}}, Above: 32, Below: 3, Circles: []BWindow{{0, 0, 26, 26, 16, true}}})
	addBoard(&BoardSpec{Family: "Sensors & modules", Name: "Rotary encoder (KY-040)", W: 26, D: 19, Above: 27, Below: 3, Circles: []BWindow{{0, 0, 7.5, 7.5, 27, true}}})
	addBoard(&BoardSpec{Family: "Sensors & modules", Name: "DHT22 temperature / humidity sensor", W: 15.1, D: 25, Above: 7.7, Below: 9, Window: &BWindow{0, 2, 15.1, 20, 7.7, false}})

	// ---------------- motors (mount by tabs, body passes through) ----------------
	addBoard(&BoardSpec{Family: "Motors", Name: "SG90 / MG90S micro servo", W: 32.3, D: 12.2, T: 2.5, Holes: []BHole{{-13.8, 0, 2}, {13.8, 0, 2}}, Above: 12, Below: 16, Cut: &BWindow{0, 0, 23.2, 12.6, 0, false}, Circles: []BWindow{{-5.9, 0, 12, 12, 12, true}}})
	addBoard(&BoardSpec{Family: "Motors", Name: "MG996R / standard servo", W: 54.5, D: 20, T: 2.5, Holes: []BHole{{-24.4, -5, 4.4}, {-24.4, 5, 4.4}, {24.4, -5, 4.4}, {24.4, 5, 4.4}}, Above: 11, Below: 26.5, Cut: &BWindow{0, 0, 40.9, 20.4, 0, false}, Circles: []BWindow{{-10, 0, 14, 14, 11, true}}})
	addBoard(&BoardSpec{Family: "Motors", Name: "28BYJ-48 stepper motor", W: 42, D: 28, T: 0.8, Holes: []BHole{{-17.5, 0, 4.2}, {17.5, 0, 4.2}}, Above: 10, Below: 19, Cut: &BWindow{0, 0, 28.5, 28.5, 0, true}, Circles: []BWindow{{0, 8, 9.5, 9.5, 10, true}}})
	addBoard(&BoardSpec{Family: "Motors", Name: "N20 gear motor", W: 24, D: 12, T: 10, Above: 0, Below: 0, Cut: &BWindow{0, 0, 24.4, 12.4, 0, false}})
}

// BoardNames lists boards grouped by family (used for the dropdown; "Family — Name").
func BoardNames() []string {
	fam := map[string][]string{}
	var order []string
	for _, n := range boardOrder {
		f := boardDB[n].Family
		if _, ok := fam[f]; !ok {
			order = append(order, f)
		}
		fam[f] = append(fam[f], n)
	}
	var out []string
	for _, f := range order {
		for _, n := range fam[f] {
			out = append(out, f+" — "+n)
		}
	}
	out = append(out, "Custom — Custom board (type sizes)")
	return out
}

func boardFromParams(p Params) *BoardSpec {
	// scanned or user-supplied spec as JSON
	if js := p.S("spec", ""); js != "" {
		var b BoardSpec
		if json.Unmarshal([]byte(js), &b) == nil && b.W > 0 {
			return &b
		}
	}
	name := p.S("board", "")
	if i := indexDash(name); i >= 0 {
		name = name[i+len(" — "):]
	}
	if b, ok := boardDB[name]; ok {
		return b
	}
	// custom board
	w, d := p.F("cw", 60), p.F("cd", 40)
	b := &BoardSpec{Name: "Custom board", W: w, D: d, T: p.F("ct", 1.6), Above: p.F("cabove", 10), Below: p.F("cbelow", 3)}
	if n := p.I("choles", 4); n > 0 {
		hs := corners(w, d, p.F("cinset", 3.5), p.F("chd", 3.2))
		if n == 2 {
			hs = []BHole{hs[0], hs[2]}
		}
		b.Holes = hs
	}
	return b
}

func indexDash(s string) int {
	for i := 0; i+len(" — ") <= len(s); i++ {
		if s[i:i+len(" — ")] == " — " {
			return i
		}
	}
	return -1
}

var _ = sort.Strings
