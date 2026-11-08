// Flashie — Cheap Yellow Display (CYD) board database.
// Every CYD's display driver, touch controller and pinout, so firmware can
// be matched to the exact board. Made by nrmluvdev.
//
// Sources (checked November 2026):
//   - rzeldent/platformio-espressif32-sunton board JSON files (pin defines)
//   - Arduino_GFX / ESPHome / LovyanGFX configs for the RGB-panel boards
//     (homeding.github.io, forum.squareline.io, wiki.ncot.uk)
//   - witnessmenow/ESP32-Cheap-Yellow-Display, atomic14, ropg/lvgl_cyd
//   - openHASP issue #713 + straga/micropython_lcd for the JC4827W543

package flashie

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

type CYDPin struct {
	Func string `json:"f"`
	GPIO int    `json:"g"`
}

type CYDGroup struct {
	Name string   `json:"name"`
	Bus  string   `json:"bus,omitempty"`
	Pins []CYDPin `json:"pins"`
	Note string   `json:"note,omitempty"`
}

type CYD struct {
	ID         string     `json:"id"`
	Model      string     `json:"model"` // as printed on the PCB
	Size       string     `json:"size"`
	Res        string     `json:"res"`
	Chip       string     `json:"chip"`
	Module     string     `json:"module"`
	FlashMB    int        `json:"flashMB"`
	PSRAMMB    int        `json:"psramMB,omitempty"`
	USB        string     `json:"usb"`
	Display    string     `json:"display"`
	DisplayBus string     `json:"displayBus"`
	ColorOrder string     `json:"colorOrder,omitempty"`
	Touch      string     `json:"touch"`
	TouchBus   string     `json:"touchBus,omitempty"`
	Audio      string     `json:"audio,omitempty"`
	Extras     []string   `json:"extras,omitempty"`
	Groups     []CYDGroup `json:"groups"`
	Notes      []string   `json:"notes,omitempty"`
	Tokens     []string   `json:"-"`      // strings that identify firmware built for this board
	Family     string     `json:"family"` // shared token core, e.g. 2432S028
	Source     string     `json:"source"`
	TFTeSPI    string     `json:"tftDriver,omitempty"` // TFT_eSPI driver define, SPI boards only
	Verified   string     `json:"verified"`            // what has been cross-checked
}

func pins(kv ...any) []CYDPin {
	var out []CYDPin
	for i := 0; i+1 < len(kv); i += 2 {
		out = append(out, CYDPin{kv[i].(string), kv[i+1].(int)})
	}
	return out
}

const (
	srcSunton = "rzeldent/platformio-espressif32-sunton"
	srcRGB    = "Arduino_GFX / ESPHome / LovyanGFX configs, cross-checked with " + srcSunton
)

// Shared blocks for the ESP32-WROOM CYD family.
var (
	grpTF32  = CYDGroup{Name: "SD card", Bus: "SPI (VSPI)", Pins: pins("CS", 5, "MOSI", 23, "SCK", 18, "MISO", 19)}
	grpMisc  = CYDGroup{Name: "On-board extras", Pins: pins("RGB LED red", 4, "RGB LED green", 16, "RGB LED blue", 17, "Light sensor (LDR)", 34, "Speaker", 26)}
	grpTFS3  = CYDGroup{Name: "SD card", Bus: "SPI", Pins: pins("CS", 10, "MOSI", 11, "SCK", 12, "MISO", 13)}
	extras32 = []string{"microSD", "RGB LED", "Light sensor", "Speaker amp (FM8002A)"}
)

func spiDisplay(ic string, bl int, miso int) CYDGroup {
	p := pins("MOSI", 13, "SCK", 14, "CS", 15, "DC", 2, "Backlight", bl)
	if miso >= 0 {
		p = append(pins("MISO", miso), p...)
	}
	return CYDGroup{Name: "Display — " + ic, Bus: "SPI (HSPI)", Pins: p}
}

func rgb565(de, vs, hs, pclk int, r, g, b []int, bl int, note string) CYDGroup {
	p := pins("DE", de, "VSYNC", vs, "HSYNC", hs, "PCLK", pclk)
	for i, v := range r {
		p = append(p, CYDPin{fmt.Sprintf("R%d", i), v})
	}
	for i, v := range g {
		p = append(p, CYDPin{fmt.Sprintf("G%d", i), v})
	}
	for i, v := range b {
		p = append(p, CYDPin{fmt.Sprintf("B%d", i), v})
	}
	p = append(p, CYDPin{"Backlight", bl})
	return CYDGroup{Name: "Display — ST7262", Bus: "16-bit parallel RGB (RGB565)", Pins: p, Note: note}
}

var (
	rgbS043R = []int{45, 48, 47, 21, 14}
	rgbS043G = []int{5, 6, 7, 15, 16, 4}
	rgbS043B = []int{8, 3, 46, 9, 1}
	rgbS070R = []int{14, 21, 47, 48, 45}
	rgbS070G = []int{9, 46, 3, 8, 16, 1}
	rgbS070B = []int{15, 7, 6, 5, 4}

	noteRGBOrder = "Pin names follow Arduino_GFX / ESPHome (R = 45,48,…). esp32-smartdisplay configures the panel as BGR and so labels R and B the other way round — same wires."
)

var (
	touchXPTsep    = CYDGroup{Name: "Touch — XPT2046 (resistive)", Bus: "SPI (own bus, VSPI)", Pins: pins("MOSI", 32, "MISO", 39, "SCK", 25, "CS", 33, "IRQ", 36)}
	touchXPTshared = CYDGroup{Name: "Touch — XPT2046 (resistive)", Bus: "SPI (shares the display bus 13/12/14)", Pins: pins("CS", 33, "IRQ", 36)}
	touchGT911_32  = CYDGroup{Name: "Touch — GT911 (capacitive)", Bus: "I²C, address 0x5D", Pins: pins("SDA", 33, "SCL", 32, "RST", 25, "INT", 21)}
	touchGT911S3   = CYDGroup{Name: "Touch — GT911 (capacitive)", Bus: "I²C, address 0x5D", Pins: pins("SDA", 19, "SCL", 20, "RST", 38, "INT", 18)}
	touchXPTS3     = CYDGroup{Name: "Touch — XPT2046 (resistive)", Bus: "SPI (shares the SD card bus)", Pins: pins("MOSI", 11, "MISO", 13, "SCK", 12, "CS", 38, "IRQ", 18)}
)

var cyds = []CYD{
	// ------------------------------------------------------------ 1.28" round
	{ID: "2424s012c", Model: "ESP32-2424S012C", Size: `1.28" round`, Res: "240×240", Chip: "ESP32-C3", Module: "ESP32-C3-MINI-1U", FlashMB: 4, USB: "USB-C",
		Display: "GC9A01A", DisplayBus: "SPI", Touch: "CST816S (capacitive)", TouchBus: "I²C",
		Groups: []CYDGroup{
			{Name: "Display — GC9A01A", Bus: "SPI", Pins: pins("MOSI", 7, "SCK", 6, "CS", 10, "DC", 2, "Backlight", 3)},
			{Name: "Touch — CST816S (capacitive)", Bus: "I²C", Pins: pins("SDA", 4, "SCL", 5, "RST", 1, "INT", 0)},
		},
		Notes:  []string{"The 2424S012N is the same board without touch."},
		Tokens: []string{"2424S012"}, Family: "2424S012", Source: srcSunton, Verified: "Pins from the Sunton board definition"},

	// ------------------------------------------------------------ 1.9"
	{ID: "1732s019c", Model: "ESP32-1732S019C", Size: `1.9"`, Res: "170×320", Chip: "ESP32-S3", Module: "ESP32-S3-WROOM-1-N16R8", FlashMB: 16, PSRAMMB: 8, USB: "USB-C",
		Display: "ST7789", DisplayBus: "SPI", Touch: "GT911 (capacitive)", TouchBus: "I²C",
		Groups: []CYDGroup{
			{Name: "Display — ST7789", Bus: "SPI", Pins: pins("MOSI", 13, "SCK", 12, "CS", 10, "DC", 11, "RST", 1, "Backlight", 14),
				Note: "The 170-pixel panel sits 35 pixels in on the ST7789's 240-wide memory (column offset 35)."},
			{Name: "Touch — GT911 (capacitive)", Bus: "I²C", Pins: pins("SDA", 9, "SCL", 46, "RST", 3, "INT", 8)},
		},
		Notes:  []string{"The 1732S019N is the same board without touch."},
		Tokens: []string{"1732S019"}, Family: "1732S019", Source: srcSunton, TFTeSPI: "ST7789_DRIVER", Verified: "Pins from the Sunton board definition"},

	// ------------------------------------------------------------ 2.2"
	{ID: "2432s022c", Model: "ESP32-2432S022C", Size: `2.2"`, Res: "240×320", Chip: "ESP32", Module: "ESP32-WROOM-32", FlashMB: 4, USB: "USB-C",
		Display: "ST7789", DisplayBus: "8-bit parallel (i80)", Touch: "CST816S (capacitive)", TouchBus: "I²C", Audio: "FM8002A",
		Groups: []CYDGroup{
			{Name: "Display — ST7789", Bus: "8-bit parallel (Intel 8080)", Pins: pins("CS", 17, "DC", 16, "WR", 4, "RD", 2,
				"Data 0", 15, "Data 1", 13, "Data 2", 12, "Data 3", 14, "Data 4", 27, "Data 5", 25, "Data 6", 33, "Data 7", 32, "Backlight", 0),
				Note: "Data lines in bus order as listed by esp32-smartdisplay (it numbers them D8–D15)."},
			{Name: "Touch — CST816S (capacitive)", Bus: "I²C", Pins: pins("SDA", 21, "SCL", 22), Note: "Reset and interrupt aren't wired to the ESP32."},
			grpTF32,
			{Name: "On-board extras", Pins: pins("Speaker", 26)},
		},
		Notes:  []string{"Parallel display — TFT_eSPI's usual CYD SPI setup won't work on this one.", "The 2432S022N is the same board without touch."},
		Tokens: []string{"2432S022"}, Family: "2432S022", Source: srcSunton, Verified: "Pins from the Sunton board definition"},

	// ------------------------------------------------------------ 2.4"
	{ID: "2432s024r", Model: "ESP32-2432S024R", Size: `2.4"`, Res: "240×320", Chip: "ESP32", Module: "ESP32-WROOM-32", FlashMB: 4, USB: "Micro-USB",
		Display: "ILI9341", DisplayBus: "SPI", ColorOrder: "BGR", Touch: "XPT2046 (resistive)", TouchBus: "SPI (shared)", Audio: "FM8002A", Extras: extras32,
		Groups: []CYDGroup{spiDisplay("ILI9341", 27, 12), touchXPTshared, grpTF32, grpMisc},
		Notes:  []string{"Backlight is GPIO 27 on the 2.4\" (GPIO 21 on the 2.8\")."},
		Tokens: []string{"2432S024R", "2432S024"}, Family: "2432S024", Source: srcSunton, TFTeSPI: "ILI9341_2_DRIVER", Verified: "Pins from the Sunton board definition"},
	{ID: "2432s024c", Model: "ESP32-2432S024C", Size: `2.4"`, Res: "240×320", Chip: "ESP32", Module: "ESP32-WROOM-32", FlashMB: 4, USB: "Micro-USB",
		Display: "ILI9341", DisplayBus: "SPI", ColorOrder: "BGR", Touch: "CST820 (capacitive)", TouchBus: "I²C", Audio: "FM8002A", Extras: extras32,
		Groups: []CYDGroup{spiDisplay("ILI9341", 27, 12),
			{Name: "Touch — CST820 (capacitive)", Bus: "I²C", Pins: pins("SDA", 33, "SCL", 32, "RST", 25, "INT", 21), Note: "CST820 is drop-in compatible with CST816S drivers."},
			grpTF32, grpMisc},
		Notes:  []string{"The 2432S024N is the same board without touch."},
		Tokens: []string{"2432S024C", "2432S024"}, Family: "2432S024", Source: srcSunton, TFTeSPI: "ILI9341_2_DRIVER", Verified: "Pins from the Sunton board definition"},

	// ------------------------------------------------------------ 2.8" — the CYD
	{ID: "2432s028r", Model: "ESP32-2432S028R", Size: `2.8"`, Res: "240×320", Chip: "ESP32", Module: "ESP32-WROOM-32", FlashMB: 4, USB: "Micro-USB (CH340)",
		Display: "ILI9341", DisplayBus: "SPI", ColorOrder: "BGR", Touch: "XPT2046 (resistive)", TouchBus: "SPI (own bus)", Audio: "FM8002A", Extras: extras32,
		Groups: []CYDGroup{spiDisplay("ILI9341", 21, 12), touchXPTsep, grpTF32, grpMisc},
		Notes: []string{"This is the original “Cheap Yellow Display”.",
			"Touch is on its own SPI bus — code written for a shared-bus XPT2046 won't see touches."},
		Tokens: []string{"2432S028R", "2432S028", "CYD"}, Family: "2432S028", Source: srcSunton + ", witnessmenow/ESP32-Cheap-Yellow-Display",
		TFTeSPI: "ILI9341_2_DRIVER", Verified: "Pins from the Sunton definition, matching the witnessmenow CYD guide and Random Nerd Tutorials"},
	{ID: "2432s028r-v2", Model: "ESP32-2432S028R (USB-C, v2)", Size: `2.8"`, Res: "240×320", Chip: "ESP32", Module: "ESP32-WROOM-32", FlashMB: 4, USB: "USB-C",
		Display: "ILI9341", DisplayBus: "SPI", ColorOrder: "BGR", Touch: "XPT2046 (resistive)", TouchBus: "SPI (own bus)", Audio: "FM8002A", Extras: extras32,
		Groups: []CYDGroup{spiDisplay("ILI9341", 21, 12), touchXPTsep, grpTF32, grpMisc},
		Notes: []string{"Same pins as the micro-USB CYD, but the panel needs a different init sequence — firmware for the original may show washed-out or wrong colours.",
			"Pick firmware built for “2432S028Rv2” / USB-C CYD where offered."},
		Tokens: []string{"2432S028RV2", "2432S028R_V2", "2432S028", "CYD"}, Family: "2432S028", Source: srcSunton,
		TFTeSPI: "ILI9341_2_DRIVER", Verified: "Pins from the Sunton board definition"},
	{ID: "2432s028r-v3", Model: "ESP32-2432S028R (2 USB, v3 / “CYD2USB”)", Size: `2.8"`, Res: "240×320", Chip: "ESP32", Module: "ESP32-WROOM-32", FlashMB: 4, USB: "USB-C + Micro-USB",
		Display: "ST7789", DisplayBus: "SPI", ColorOrder: "RGB", Touch: "XPT2046 (resistive)", TouchBus: "SPI (own bus)", Audio: "FM8002A", Extras: extras32,
		Groups: []CYDGroup{spiDisplay("ST7789", 21, -1), touchXPTsep, grpTF32, grpMisc},
		Notes: []string{"Uses an ST7789 instead of the ILI9341: needs colour inversion on, SPI mode 3 and no MISO.",
			"Its USB-C port has no CC resistors, so USB-C-to-C cables won't power it — use a USB-A-to-C cable or the micro-USB port.",
			"Boxes marked “7789” are usually this version."},
		Tokens: []string{"CYD2USB", "2432S028RV3", "2432S028R_V3", "2432S028", "CYD"}, Family: "2432S028", Source: srcSunton + ", atomic14, ropg/lvgl_cyd",
		TFTeSPI: "ST7789_DRIVER", Verified: "Pins from the Sunton definition; ST7789 + inversion confirmed by atomic14 and ropg"},

	// ------------------------------------------------------------ 3.2"
	{ID: "2432s032r", Model: "ESP32-2432S032R", Size: `3.2"`, Res: "240×320", Chip: "ESP32", Module: "ESP32-WROOM-32", FlashMB: 4, USB: "Micro-USB",
		Display: "ST7789", DisplayBus: "SPI", ColorOrder: "RGB", Touch: "XPT2046 (resistive)", TouchBus: "SPI (shared)", Audio: "FM8002A",
		Groups: []CYDGroup{spiDisplay("ST7789", 27, 12), touchXPTshared},
		Notes:  []string{"The 2432S032N is the same board without touch."},
		Tokens: []string{"2432S032R", "2432S032"}, Family: "2432S032", Source: srcSunton, TFTeSPI: "ST7789_DRIVER", Verified: "Display and touch pins from the Sunton board definition"},
	{ID: "2432s032c", Model: "ESP32-2432S032C", Size: `3.2"`, Res: "240×320", Chip: "ESP32", Module: "ESP32-WROOM-32", FlashMB: 4, USB: "Micro-USB",
		Display: "ST7789", DisplayBus: "SPI", ColorOrder: "RGB", Touch: "GT911 (capacitive)", TouchBus: "I²C", Audio: "FM8002A", Extras: extras32,
		Groups: []CYDGroup{spiDisplay("ST7789", 27, 12), touchGT911_32, grpTF32, grpMisc},
		Tokens: []string{"2432S032C", "2432S032"}, Family: "2432S032", Source: srcSunton, TFTeSPI: "ST7789_DRIVER", Verified: "Pins from the Sunton board definition"},

	// ------------------------------------------------------------ 3.5"
	{ID: "3248s035r", Model: "ESP32-3248S035R", Size: `3.5"`, Res: "320×480", Chip: "ESP32", Module: "ESP32-WROOM-32", FlashMB: 4, USB: "Micro-USB",
		Display: "ST7796", DisplayBus: "SPI", ColorOrder: "BGR", Touch: "XPT2046 (resistive)", TouchBus: "SPI (shared)", Audio: "FM8002A", Extras: extras32,
		Groups: []CYDGroup{spiDisplay("ST7796", 27, 12), touchXPTshared, grpTF32, grpMisc},
		Notes:  []string{"Touch shares the display's SPI bus (unlike the 2.8\")."},
		Tokens: []string{"3248S035R", "3248S035"}, Family: "3248S035", Source: srcSunton, TFTeSPI: "ST7796_DRIVER", Verified: "Pins from the Sunton board definition"},
	{ID: "3248s035c", Model: "ESP32-3248S035C", Size: `3.5"`, Res: "320×480", Chip: "ESP32", Module: "ESP32-WROOM-32", FlashMB: 4, USB: "Micro-USB",
		Display: "ST7796", DisplayBus: "SPI", ColorOrder: "BGR", Touch: "GT911 (capacitive)", TouchBus: "I²C", Audio: "FM8002A", Extras: extras32,
		Groups: []CYDGroup{spiDisplay("ST7796", 27, 12), touchGT911_32, grpTF32, grpMisc},
		Tokens: []string{"3248S035C", "3248S035"}, Family: "3248S035", Source: srcSunton, TFTeSPI: "ST7796_DRIVER", Verified: "Pins from the Sunton board definition"},

	// ------------------------------------------------------------ 4.0" square
	{ID: "4848s040c", Model: "ESP32-4848S040C", Size: `4.0" square`, Res: "480×480", Chip: "ESP32-S3", Module: "ESP32-S3-WROOM-1-N16R8", FlashMB: 16, PSRAMMB: 8, USB: "USB-C",
		Display: "ST7701", DisplayBus: "16-bit parallel RGB + 3-wire SPI init", ColorOrder: "BGR", Touch: "GT911 (capacitive)", TouchBus: "I²C", Audio: "NS4168 (I²S)",
		Extras: []string{"Relays (optional)", "I²S audio"},
		Groups: []CYDGroup{
			{Name: "Display — ST7701 init", Bus: "3-wire SPI", Pins: pins("CS", 39, "SCK", 48, "SDA", 47)},
			{Name: "Display — ST7701 RGB", Bus: "16-bit parallel RGB", Pins: pins("HSYNC", 16, "VSYNC", 17, "DE", 18, "PCLK", 21,
				"R0", 11, "R1", 12, "R2", 13, "R3", 14, "R4", 0, "G0", 8, "G1", 20, "G2", 3, "G3", 46, "G4", 9, "G5", 10,
				"B0", 4, "B1", 5, "B2", 6, "B3", 7, "B4", 15, "Backlight", 38),
				Note: "Pin names as in esp32-smartdisplay (BGR panel) — other libraries may label R and B the other way round."},
			{Name: "Touch — GT911 (capacitive)", Bus: "I²C", Pins: pins("SDA", 19, "SCL", 45), Note: "Reset and interrupt aren't wired."},
			{Name: "Relay", Pins: pins("Relay 1", 40)},
		},
		Notes:  []string{"Relays and I²S audio share IO1, IO2 and IO40 — for audio, move the 0 Ω links R25/R26/R27 to R21/R22/R23."},
		Tokens: []string{"4848S040"}, Family: "4848S040", Source: srcSunton, Verified: "Pins from the Sunton board definition"},

	// ------------------------------------------------------------ 4.3"
	{ID: "4827s043c", Model: "ESP32-4827S043C", Size: `4.3"`, Res: "480×272", Chip: "ESP32-S3", Module: "ESP32-S3-WROOM-1-N16R8", FlashMB: 16, PSRAMMB: 8, USB: "USB-C",
		Display: "ST7262 (RGB panel)", DisplayBus: "16-bit parallel RGB", Touch: "GT911 (capacitive)", TouchBus: "I²C",
		Groups: []CYDGroup{rgb565(40, 41, 39, 42, rgbS043R, rgbS043G, rgbS043B, 2, noteRGBOrder+" Pixel clock about 8 MHz."), touchGT911S3, grpTFS3},
		Tokens: []string{"4827S043C", "4827S043"}, Family: "4827S043", Source: srcRGB, Verified: "RGB pins agree across Sunton, Arduino_GFX and LovyanGFX configs"},
	{ID: "4827s043r", Model: "ESP32-4827S043R", Size: `4.3"`, Res: "480×272", Chip: "ESP32-S3", Module: "ESP32-S3-WROOM-1-N16R8", FlashMB: 16, PSRAMMB: 8, USB: "USB-C",
		Display: "ST7262 (RGB panel)", DisplayBus: "16-bit parallel RGB", Touch: "XPT2046 (resistive)", TouchBus: "SPI",
		Groups: []CYDGroup{rgb565(40, 41, 39, 42, rgbS043R, rgbS043G, rgbS043B, 2, noteRGBOrder), touchXPTS3, grpTFS3},
		Tokens: []string{"4827S043R", "4827S043"}, Family: "4827S043", Source: srcRGB, Verified: "RGB pins agree across sources; touch from the Sunton definition"},
	{ID: "jc4827w543c", Model: "JC4827W543C (Guition)", Size: `4.3"`, Res: "480×272", Chip: "ESP32-S3", Module: "ESP32-S3-WROOM-1-N4R8", FlashMB: 4, PSRAMMB: 8, USB: "USB-C",
		Display: "NV3041A", DisplayBus: "QSPI (4-line)", Touch: "GT911 (capacitive)", TouchBus: "I²C",
		Groups: []CYDGroup{
			{Name: "Display — NV3041A", Bus: "QSPI", Pins: pins("CS", 45, "SCK", 47, "D0", 21, "D1", 48, "D2", 40, "D3", 39, "Backlight", 1)},
		},
		Notes:  []string{"Guition-made, often sold alongside Sunton CYDs. Touch pin numbers aren't listed because Flashie couldn't verify them."},
		Tokens: []string{"JC4827W543", "4827W543"}, Family: "JC4827W543", Source: "openHASP issue #713, straga/micropython_lcd", Verified: "Display pins agree across two independent sources"},
	{ID: "8048s043c", Model: "ESP32-8048S043C", Size: `4.3"`, Res: "800×480", Chip: "ESP32-S3", Module: "ESP32-S3-WROOM-1-N16R8", FlashMB: 16, PSRAMMB: 8, USB: "USB-C",
		Display: "ST7262 (RGB panel, IPS)", DisplayBus: "16-bit parallel RGB", Touch: "GT911 (capacitive)", TouchBus: "I²C",
		Groups: []CYDGroup{rgb565(40, 41, 39, 42, rgbS043R, rgbS043G, rgbS043B, 2, noteRGBOrder+" Pixel clock 12.5–16 MHz."), touchGT911S3, grpTFS3},
		Notes:  []string{"Check the label on the back — a near-identical 480×272 board (4827S043) exists.", "The 8048S043N is the same board without touch."},
		Tokens: []string{"8048S043C", "8048S043"}, Family: "8048S043", Source: srcRGB + ", homeding.github.io", Verified: "RGB pins agree across Sunton, Arduino_GFX, ESPHome and homeding"},
	{ID: "8048s043r", Model: "ESP32-8048S043R", Size: `4.3"`, Res: "800×480", Chip: "ESP32-S3", Module: "ESP32-S3-WROOM-1-N16R8", FlashMB: 16, PSRAMMB: 8, USB: "USB-C",
		Display: "ST7262 (RGB panel, IPS)", DisplayBus: "16-bit parallel RGB", Touch: "XPT2046 (resistive)", TouchBus: "SPI",
		Groups: []CYDGroup{rgb565(40, 41, 39, 42, rgbS043R, rgbS043G, rgbS043B, 2, noteRGBOrder), touchXPTS3, grpTFS3},
		Tokens: []string{"8048S043R", "8048S043"}, Family: "8048S043", Source: srcRGB, Verified: "RGB pins agree across sources; touch from the Sunton definition"},

	// ------------------------------------------------------------ 5"
	{ID: "8048s050c", Model: "ESP32-8048S050C", Size: `5.0"`, Res: "800×480", Chip: "ESP32-S3", Module: "ESP32-S3-WROOM-1-N16R8", FlashMB: 16, PSRAMMB: 8, USB: "USB-C",
		Display: "ST7262 (RGB panel)", DisplayBus: "16-bit parallel RGB", Touch: "GT911 (capacitive)", TouchBus: "I²C",
		Groups: []CYDGroup{rgb565(40, 41, 39, 42, rgbS043R, rgbS043G, rgbS043B, 2, noteRGBOrder+" Pixel clock 16 MHz."), touchGT911S3, grpTFS3},
		Notes:  []string{"Same wiring as the 4.3\" 8048S043 — only the panel size differs.", "The 8048S050N is the same board without touch."},
		Tokens: []string{"8048S050C", "8048S050"}, Family: "8048S050", Source: srcRGB, Verified: "Pins from the Sunton definition, same as the 8048S043"},
	{ID: "8048s050r", Model: "ESP32-8048S050R", Size: `5.0"`, Res: "800×480", Chip: "ESP32-S3", Module: "ESP32-S3-WROOM-1-N16R8", FlashMB: 16, PSRAMMB: 8, USB: "USB-C",
		Display: "ST7262 (RGB panel)", DisplayBus: "16-bit parallel RGB", Touch: "XPT2046 (resistive)", TouchBus: "SPI",
		Groups: []CYDGroup{rgb565(40, 41, 39, 42, rgbS043R, rgbS043G, rgbS043B, 2, noteRGBOrder), touchXPTS3, grpTFS3},
		Tokens: []string{"8048S050R", "8048S050"}, Family: "8048S050", Source: srcRGB, Verified: "Pins from the Sunton board definition"},

	// ------------------------------------------------------------ 7"
	{ID: "8048s070c", Model: "ESP32-8048S070C", Size: `7.0"`, Res: "800×480", Chip: "ESP32-S3", Module: "ESP32-S3-WROOM-1-N16R8", FlashMB: 16, PSRAMMB: 8, USB: "USB-C",
		Display: "ST7262 (RGB panel)", DisplayBus: "16-bit parallel RGB", Touch: "GT911 (capacitive)", TouchBus: "I²C",
		Groups: []CYDGroup{rgb565(41, 40, 39, 42, rgbS070R, rgbS070G, rgbS070B, 2, noteRGBOrder+" Pixel clock about 12.5 MHz."), touchGT911S3, grpTFS3},
		Notes: []string{"Wired differently from the 4.3\"/5\" boards — DE and VSYNC are swapped and the colour pins are different. Firmware for those won't display properly here.",
			"Board revisions v1.1 and v1.3 exist and differ in their colour wiring; if colours look wrong, try firmware for the other revision.",
			"The 8048S070N is the same board without touch."},
		Tokens: []string{"8048S070C", "8048S070"}, Family: "8048S070", Source: srcRGB + ", wiki.ncot.uk",
		Verified: "RGB pins agree across Sunton, Arduino_GFX and the ncot.uk wiki"},
	{ID: "8048s070r", Model: "ESP32-8048S070R", Size: `7.0"`, Res: "800×480", Chip: "ESP32-S3", Module: "ESP32-S3-WROOM-1-N16R8", FlashMB: 16, PSRAMMB: 8, USB: "USB-C",
		Display: "ST7262 (RGB panel)", DisplayBus: "16-bit parallel RGB", Touch: "XPT2046 (resistive)", TouchBus: "SPI",
		Groups: []CYDGroup{rgb565(41, 40, 39, 42, rgbS070R, rgbS070G, rgbS070B, 2, noteRGBOrder), touchXPTS3, grpTFS3},
		Tokens: []string{"8048S070R", "8048S070"}, Family: "8048S070", Source: srcRGB, Verified: "RGB pins agree across sources; touch from the Sunton definition"},
}

func cydByID(id string) *CYD {
	for i := range cyds {
		if cyds[i].ID == id {
			return &cyds[i]
		}
	}
	return nil
}

func init() {
	// Every CYD becomes a selectable board in the main list.
	for _, c := range cyds {
		n := c.Model
		if !strings.Contains(n, c.Size) {
			n = fmt.Sprintf("%s %s · %s", c.Size, strings.TrimPrefix(c.Model, "ESP32-"), c.Res)
		}
		b := espB("cyd-"+c.ID, n, c.Chip, c.FlashMB, "")
		b.Group = "Cheap Yellow Display (CYD)"
		b.CYD = c.ID
		boards = append(boards, b)
	}
}

// ------------------------------------------------------ firmware matching ---

// CYDHit is a reason to believe a firmware file targets a particular CYD.
type CYDHit struct {
	ID     string `json:"id,omitempty"`
	Family string `json:"family,omitempty"`
	Token  string `json:"token"`
	Where  string `json:"where"` // filename | firmware
	Exact  bool   `json:"exact"`
}

// CYDMatch summarises what a firmware file says about the board it's for.
type CYDMatch struct {
	Hits     []CYDHit `json:"hits"`
	Families []string `json:"families"` // distinct board families mentioned
	Drivers  []string `json:"drivers"`  // display/touch driver names seen
}

var driverTokens = []string{"ILI9341", "ST7789", "ST7796", "ST7262", "ST7701", "GC9A01", "NV3041A", "XPT2046", "GT911", "CST816", "CST820"}

func normTok(s string) []byte {
	return bytes.ToUpper([]byte(strings.NewReplacer("-", "", "_", "", " ", "").Replace(s)))
}

// matchCYDFirmware looks for CYD model names and driver names in a file's
// name and in readable strings inside it.
func matchCYDFirmware(name string, data []byte) CYDMatch {
	m := CYDMatch{Hits: []CYDHit{}, Families: []string{}, Drivers: []string{}}
	fname := normTok(name)
	// Collapse the firmware to upper case with - and _ removed, so
	// "esp32-2432s028r" and "ESP32_2432S028R" both match.
	body := make([]byte, 0, len(data))
	for _, c := range data {
		switch {
		case c == '-' || c == '_':
		case c >= 'a' && c <= 'z':
			body = append(body, c-32)
		default:
			body = append(body, c)
		}
	}
	seenFam := map[string]bool{}
	seenHit := map[string]bool{}
	add := func(h CYDHit) {
		k := h.ID + "|" + h.Family + "|" + h.Where
		if seenHit[k] {
			return
		}
		seenHit[k] = true
		m.Hits = append(m.Hits, h)
		if h.Family != "" && !seenFam[h.Family] {
			seenFam[h.Family] = true
			m.Families = append(m.Families, h.Family)
		}
	}
	for _, c := range cyds {
		for _, t := range c.Tokens {
			if t == "CYD" {
				continue // too generic to identify a model on its own
			}
			nt := normTok(t)
			exact := t != c.Family
			for _, src := range []struct {
				where string
				b     []byte
			}{{"filename", fname}, {"firmware", body}} {
				if bytes.Contains(src.b, nt) {
					h := CYDHit{Family: c.Family, Token: t, Where: src.where, Exact: exact}
					if exact {
						h.ID = c.ID
					}
					add(h)
				}
			}
		}
	}
	// "CYD" on its own in the filename usually means the classic 2.8".
	if len(m.Families) == 0 && bytes.Contains(fname, []byte("CYD")) {
		add(CYDHit{Family: "2432S028", Token: "CYD", Where: "filename"})
	}
	for _, d := range driverTokens {
		if bytes.Contains(body, []byte(d)) || bytes.Contains(fname, []byte(d)) {
			m.Drivers = append(m.Drivers, d)
		}
	}
	// Drop a model-level hit when a more specific one covers it
	// (e.g. "2432S028R" inside "2432S028RV3").
	var kept []CYDHit
	for _, h := range m.Hits {
		shadowed := false
		if h.Exact {
			for _, o := range m.Hits {
				if o.Exact && o.Where == h.Where && o.ID != h.ID && len(o.Token) > len(h.Token) &&
					bytes.Contains(normTok(o.Token), normTok(h.Token)) {
					shadowed = true
				}
			}
		}
		if !shadowed {
			kept = append(kept, h)
		}
	}
	m.Hits = append([]CYDHit{}, kept...)
	sort.Strings(m.Families)
	return m
}
