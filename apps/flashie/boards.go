// Flashie — board catalogue and USB VID/PID → "likely board" guesses.
// Made by nrmluvdev.

package flashie

import "fmt"

// Families
const (
	famESP   = "esp"
	famRP2   = "rp2"
	famAVR   = "avr"
	famSTM32 = "stm32"
)

// Board is an entry in the manual-override dropdown.
type Board struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Family     string `json:"family"`
	Chip       string `json:"chip"`
	Group      string `json:"group"`
	Protocol   string `json:"protocol"`
	Bauds      []int  `json:"bauds,omitempty"`
	BootOffset uint32 `json:"bootOffset"`
	FlashMB    int    `json:"flashMB,omitempty"`
	MaxSketch  int    `json:"maxSketch,omitempty"`
	Note       string `json:"note,omitempty"`
	CYD        string `json:"cyd,omitempty"` // key into the CYD database
}

var espBoot = map[string]uint32{"ESP32": 0x1000, "ESP32-S2": 0x1000, "ESP8266": 0x0}

func espB(id, name, chip string, mb int, note string) Board {
	group := chip
	return Board{ID: id, Name: name, Family: famESP, Chip: chip, Group: group, Protocol: "esp", BootOffset: espBoot[chip], FlashMB: mb, Note: note}
}

func avrB(id, name, chip, proto string, bauds []int, maxSketch int, note string) Board {
	return Board{ID: id, Name: name, Family: famAVR, Chip: chip, Group: "Arduino (AVR)", Protocol: proto, Bauds: bauds, MaxSketch: maxSketch, Note: note}
}

func rpB(id, name, chip, note string) Board {
	return Board{ID: id, Name: name, Family: famRP2, Chip: chip, Group: "Raspberry Pi Pico (RP2040 / RP2350)", Protocol: "uf2", Note: note}
}

func stmB(id, name, chip string, kb int, note string) Board {
	return Board{ID: id, Name: name, Family: famSTM32, Chip: chip, Group: "STM32 (serial bootloader)", Protocol: "stm32", Bauds: []int{115200}, FlashMB: 0, MaxSketch: kb * 1024, Note: note}
}

const (
	noteCAM      = "ESP32-CAM has no auto-reset: connect GPIO0 to GND, press RESET, then flash. Remove the jumper afterwards."
	noteNativeS2 = "Native USB: hold BOOT (0), tap RST, let go — a new COM port appears. Pick that one."
	noteNative   = "Built-in USB. If flashing won't start, hold BOOT, tap RESET, then let go of BOOT."
	noteTDeck    = "T-Deck: hold the trackball down while switching on (or pressing RST) to enter download mode if auto-reset fails."
	note8266Adp  = "ESP-01 on an adapter: GPIO0 must be held to GND while powering up (use a programmer with a PROG switch)."
	notePico     = "Hold BOOTSEL while plugging in — a drive called RPI-RP2 appears. Flashie can also reboot a running Pico into BOOTSEL for you (works with C/C++ SDK and Arduino firmware)."
	notePico2    = "Hold BOOTSEL while plugging in — a drive called RP2350 appears. Flashie can also reboot a running Pico 2 for you."
	noteLeo      = "Flashie double-resets the board into its bootloader for you (1200-baud touch). If it fails, double-tap RESET and press Flash straight away."
	noteProMini  = "Pro Mini has no USB: use an FTDI/CH340 adapter with its DTR pin wired to the Pro Mini's DTR/GRN pin for auto-reset."
	noteSTM      = "Wire a USB-serial adapter: TX → PA10, RX → PA9, GND → GND. Set BOOT0 to 1, press RESET, then flash. Set BOOT0 back to 0 afterwards."
	noteBlack    = "Wire a USB-serial adapter: TX → PA10, RX → PA9, GND → GND. Hold BOOT0, tap NRST, let go of BOOT0, then flash."
)

var boards = []Board{
	// ESP32
	espB("esp32-devkitc", "ESP32 DevKitC (WROOM-32)", "ESP32", 4, ""),
	espB("esp32-devkitc-wrover", "ESP32 DevKitC (WROVER, with PSRAM)", "ESP32", 4, ""),
	espB("esp32-devkit", "ESP32 DevKit V1 (DOIT)", "ESP32", 4, ""),
	espB("nodemcu-32s", "NodeMCU-32S", "ESP32", 4, ""),
	espB("wemos-d1-mini32", "Wemos / LOLIN D1 Mini ESP32", "ESP32", 4, ""),
	espB("lolin32-lite", "LOLIN32 Lite", "ESP32", 4, ""),
	espB("ttgo-t-display", "LilyGO TTGO T-Display", "ESP32", 4, ""),
	espB("ttgo-t-beam", "LilyGO T-Beam", "ESP32", 4, ""),
	espB("m5stack-core", "M5Stack Core / Basic / Gray", "ESP32", 16, ""),
	espB("m5stack-core2", "M5Stack Core2", "ESP32", 16, ""),
	espB("m5stickc-plus", "M5StickC Plus", "ESP32", 4, ""),
	espB("m5atom-lite", "M5Stack ATOM Lite / Matrix", "ESP32", 4, ""),
	espB("esp32-cam", "ESP32-CAM (AI-Thinker)", "ESP32", 4, noteCAM),
	espB("esp32-generic", "Generic ESP32 board", "ESP32", 4, ""),
	// ESP32-S2
	espB("esp32-s2-mini", "LOLIN S2 Mini", "ESP32-S2", 4, noteNativeS2),
	espB("esp32-s2-saola", "ESP32-S2 Saola-1", "ESP32-S2", 4, ""),
	espB("esp32-s2-generic", "Generic ESP32-S2 board", "ESP32-S2", 4, ""),
	// ESP32-S3
	espB("esp32-s3-devkitc", "ESP32-S3 DevKitC-1", "ESP32-S3", 8, ""),
	espB("xiao-s3", "Seeed XIAO ESP32-S3", "ESP32-S3", 8, noteNative),
	espB("t-display-s3", "LilyGO T-Display S3", "ESP32-S3", 16, noteNative),
	espB("t-deck", "LilyGO T-Deck / T-Deck Plus", "ESP32-S3", 16, noteTDeck),
	espB("m5-atoms3", "M5Stack AtomS3", "ESP32-S3", 8, noteNative),
	espB("m5-cores3", "M5Stack CoreS3", "ESP32-S3", 16, noteNative),
	espB("s3-zero", "Waveshare ESP32-S3-Zero", "ESP32-S3", 4, noteNative),
	espB("nano-esp32", "Arduino Nano ESP32", "ESP32-S3", 16, noteNative),
	espB("heltec-v3", "Heltec WiFi LoRa 32 V3", "ESP32-S3", 8, ""),
	espB("esp32-s3-generic", "Generic ESP32-S3 board", "ESP32-S3", 8, ""),
	// ESP32-C3
	espB("esp32-c3-devkitm", "ESP32-C3 DevKitM-1", "ESP32-C3", 4, ""),
	espB("c3-supermini", "ESP32-C3 SuperMini", "ESP32-C3", 4, noteNative),
	espB("xiao-c3", "Seeed XIAO ESP32-C3", "ESP32-C3", 4, noteNative),
	espB("m5stamp-c3", "M5Stamp C3", "ESP32-C3", 4, ""),
	espB("esp32-c3-generic", "Generic ESP32-C3 board", "ESP32-C3", 4, ""),
	// ESP32-C6
	espB("esp32-c6-devkitc", "ESP32-C6 DevKitC-1", "ESP32-C6", 8, ""),
	espB("xiao-c6", "Seeed XIAO ESP32-C6", "ESP32-C6", 4, noteNative),
	espB("esp32-c6-generic", "Generic ESP32-C6 board", "ESP32-C6", 4, ""),
	// ESP32-H2
	espB("esp32-h2-devkitm", "ESP32-H2 DevKitM-1", "ESP32-H2", 4, ""),
	espB("esp32-h2-generic", "Generic ESP32-H2 board", "ESP32-H2", 4, ""),
	// ESP8266
	espB("nodemcu-v2", "NodeMCU v2 (ESP8266, CP2102)", "ESP8266", 4, ""),
	espB("nodemcu-v3", "NodeMCU v3 / LoLin (ESP8266, CH340)", "ESP8266", 4, ""),
	espB("d1-mini", "Wemos / LOLIN D1 Mini (ESP8266)", "ESP8266", 4, ""),
	espB("esp01", "ESP-01 / ESP-01S on an adapter", "ESP8266", 1, note8266Adp),
	espB("esp8266-generic", "Generic ESP8266 board", "ESP8266", 4, ""),
	// Raspberry Pi
	rpB("pico", "Raspberry Pi Pico", "RP2040", notePico),
	rpB("pico-w", "Raspberry Pi Pico W", "RP2040", notePico),
	rpB("pico2", "Raspberry Pi Pico 2", "RP2350", notePico2),
	rpB("pico2-w", "Raspberry Pi Pico 2 W", "RP2350", notePico2),
	rpB("rp2040-zero", "Waveshare RP2040-Zero / clones", "RP2040", "Hold BOOT while plugging in (or hold BOOT and tap RESET)."),
	rpB("xiao-rp2040", "Seeed XIAO RP2040", "RP2040", "Hold B (BOOT) while plugging in."),
	rpB("rp2040-generic", "Generic RP2040 board (Pico clone)", "RP2040", notePico),
	rpB("rp2350-generic", "Generic RP2350 board", "RP2350", notePico2),
	// Arduino AVR
	avrB("uno", "Arduino Uno (R3)", "ATmega328P", "stk500v1", []int{115200}, 32256, ""),
	avrB("uno-ch340", "Uno clone (CH340)", "ATmega328P", "stk500v1", []int{115200, 57600}, 32256, ""),
	avrB("nano", "Arduino Nano", "ATmega328P", "stk500v1", []int{115200, 57600}, 30720, "Flashie tries the new (115200) and old (57600) Nano bootloaders automatically."),
	avrB("nano-old", "Arduino Nano (old bootloader)", "ATmega328P", "stk500v1", []int{57600, 115200}, 30720, ""),
	avrB("nano-168", "Arduino Nano (ATmega168)", "ATmega168", "stk500v1", []int{19200, 57600}, 14336, ""),
	avrB("pro-mini-5v", "Pro Mini 5V / 16 MHz", "ATmega328P", "stk500v1", []int{57600, 115200}, 30720, noteProMini),
	avrB("pro-mini-3v3", "Pro Mini 3.3V / 8 MHz", "ATmega328P", "stk500v1", []int{57600}, 30720, noteProMini),
	avrB("mega2560", "Arduino Mega 2560", "ATmega2560", "stk500v2", []int{115200}, 253952, ""),
	avrB("mega2560-ch340", "Mega 2560 clone (CH340)", "ATmega2560", "stk500v2", []int{115200}, 253952, ""),
	avrB("leonardo", "Arduino Leonardo", "ATmega32U4", "avr109", []int{57600}, 28672, noteLeo),
	avrB("micro", "Arduino Micro", "ATmega32U4", "avr109", []int{57600}, 28672, noteLeo),
	avrB("pro-micro", "SparkFun Pro Micro (32U4)", "ATmega32U4", "avr109", []int{57600}, 28672, noteLeo),
	// STM32
	stmB("bluepill", "Blue Pill (STM32F103C8)", "STM32F103", 64, noteSTM),
	stmB("bluepill-cb", "Blue Pill (STM32F103CB, 128 KB)", "STM32F103", 128, noteSTM),
	stmB("blackpill-f401", "Black Pill (STM32F401CC)", "STM32F401", 256, noteBlack),
	stmB("blackpill-f411", "Black Pill (STM32F411CE)", "STM32F411", 512, noteBlack),
	stmB("stm32-generic", "Generic STM32 (UART bootloader)", "STM32", 0, noteSTM),
}

func boardByID(id string) *Board {
	for i := range boards {
		if boards[i].ID == id {
			return &boards[i]
		}
	}
	return nil
}

// PortInfo describes something Flashie can flash: a COM port or a UF2 drive.
type PortInfo struct {
	Name     string     `json:"name"`
	Kind     string     `json:"kind"` // serial | drive
	Device   string     `json:"device"`
	USB      bool       `json:"usb"`
	VID      uint16     `json:"vid"`
	PID      uint16     `json:"pid"`
	Friendly string     `json:"friendly"`
	Guess    BoardGuess `json:"guess"`
	drive    *uf2Drive
}

// BoardGuess is our best guess at what's on the other end.
type BoardGuess struct {
	Label      string   `json:"label"`
	BoardID    string   `json:"boardId"`
	Family     string   `json:"family"`   // set only when we're fairly sure
	Families   []string `json:"families"` // what it could be, most likely first
	Chip       string   `json:"chip"`
	USBChip    string   `json:"usbChip"`
	Confidence string   `json:"confidence"` // high | medium | low | none
	Likely     bool     `json:"likely"`
	NativeUSB  bool     `json:"nativeUsb"`
	JTAG       bool     `json:"jtag"`
	Touch1200  bool     `json:"touch1200"` // needs a 1200-baud touch to reach the bootloader
	Hint       string   `json:"hint,omitempty"`
}

func sure(fam string) []string { return []string{fam} }

var (
	bridgeFams = []string{famESP, famAVR, famSTM32}
)

func guessBoard(p PortInfo) BoardGuess {
	if !p.USB {
		return BoardGuess{Label: "Built-in or Bluetooth serial port", Confidence: "none",
			Hint: "Not a USB device — probably not your board."}
	}
	id := fmt.Sprintf("%04X:%04X", p.VID, p.PID)
	switch id {
	// ---- Espressif native USB
	case "303A:1001":
		return BoardGuess{Label: "ESP32-S3 / C3 / C6 / H2 with native USB", BoardID: "esp32-s3-generic", Family: famESP, Families: sure(famESP),
			USBChip: "Built-in USB-Serial/JTAG", Confidence: "high", Likely: true, NativeUSB: true, JTAG: true,
			Hint: "Use “Identify chip” to find out exactly which one (also covers XIAO, T-Display S3 and T-Deck)."}
	case "303A:0002":
		return BoardGuess{Label: "ESP32-S2 in download mode", BoardID: "esp32-s2-generic", Family: famESP, Families: sure(famESP), Chip: "ESP32-S2",
			USBChip: "Built-in USB (ROM)", Confidence: "high", Likely: true, NativeUSB: true}
	case "303A:80C2", "303A:80C3", "303A:8143", "303A:81DF":
		return BoardGuess{Label: "Espressif native-USB board (running firmware)", BoardID: "esp32-s3-generic", Family: famESP, Families: sure(famESP),
			USBChip: "Built-in USB (TinyUSB)", Confidence: "medium", Likely: true, NativeUSB: true, Hint: "If flashing won't start, hold BOOT and tap RESET first."}
	case "2341:0070", "2341:0071":
		return BoardGuess{Label: "Arduino Nano ESP32", BoardID: "nano-esp32", Family: famESP, Families: sure(famESP), Chip: "ESP32-S3",
			USBChip: "Built-in USB", Confidence: "high", Likely: true, NativeUSB: true}
	// ---- Arduino AVR
	case "2341:0043", "2341:0001", "2A03:0043", "2341:0243", "2341:006D":
		return BoardGuess{Label: "Arduino Uno", BoardID: "uno", Family: famAVR, Families: sure(famAVR), Chip: "ATmega328P", USBChip: "ATmega16U2", Confidence: "high", Likely: true}
	case "2341:0010", "2341:0042", "2A03:0010", "2A03:0042", "2341:0242":
		return BoardGuess{Label: "Arduino Mega 2560", BoardID: "mega2560", Family: famAVR, Families: sure(famAVR), Chip: "ATmega2560", USBChip: "ATmega16U2", Confidence: "high", Likely: true}
	case "2341:8036", "2A03:8036", "2341:0036", "2A03:0036":
		return BoardGuess{Label: "Arduino Leonardo", BoardID: "leonardo", Family: famAVR, Families: sure(famAVR), Chip: "ATmega32U4", USBChip: "Built-in USB",
			Confidence: "high", Likely: true, NativeUSB: true, Touch1200: p.PID&0x8000 != 0}
	case "2341:8037", "2A03:8037", "2341:0037", "2A03:0037":
		return BoardGuess{Label: "Arduino Micro", BoardID: "micro", Family: famAVR, Families: sure(famAVR), Chip: "ATmega32U4", USBChip: "Built-in USB",
			Confidence: "high", Likely: true, NativeUSB: true, Touch1200: p.PID&0x8000 != 0}
	case "1B4F:9206", "1B4F:9205", "1B4F:9204", "1B4F:9203":
		return BoardGuess{Label: "SparkFun Pro Micro", BoardID: "pro-micro", Family: famAVR, Families: sure(famAVR), Chip: "ATmega32U4", USBChip: "Built-in USB",
			Confidence: "high", Likely: true, NativeUSB: true, Touch1200: p.PID == 0x9206 || p.PID == 0x9204}
	// ---- USB-serial bridge chips (could be anything)
	case "10C4:EA60":
		return BoardGuess{Label: "ESP32 / ESP8266 board (CP2102)", BoardID: "", Families: []string{famESP, famSTM32, famAVR}, USBChip: "Silicon Labs CP210x",
			Confidence: "medium", Likely: true, Hint: "CP2102 is used on ESP32 DevKits, NodeMCU v2 and M5Stack. Press “Identify chip” and Flashie will work out which."}
	case "1A86:7523":
		return BoardGuess{Label: "ESP32, ESP8266 or Arduino clone (CH340)", BoardID: "", Families: []string{famESP, famAVR, famSTM32}, USBChip: "WCH CH340",
			Confidence: "low", Likely: true, Hint: "CH340 is on CYDs, D1 Minis, NodeMCU v3 and Nano/Uno/Mega clones. Press “Identify chip” and Flashie will work out which."}
	case "1A86:55D4", "1A86:55D3", "1A86:55D2":
		return BoardGuess{Label: "ESP32 board (CH9102 / CH343)", BoardID: "", Families: []string{famESP, famAVR, famSTM32}, USBChip: "WCH CH9102/CH343",
			Confidence: "medium", Likely: true, Hint: "Used on newer DevKits, M5Stack Core2, T-Display and CYDs. “Identify chip” confirms it."}
	case "0403:6001", "0403:6015":
		return BoardGuess{Label: "FTDI adapter — Pro Mini, old Nano, ESP32-CAM or STM32", BoardID: "", Families: []string{famAVR, famESP, famSTM32}, USBChip: "FTDI FT232",
			Confidence: "low", Likely: true, Hint: "Pick your board below, or press “Identify chip”."}
	case "0403:6010", "0403:6014":
		return BoardGuess{Label: "ESP-Prog / ESP-WROVER-KIT", BoardID: "esp32-generic", Family: famESP, Families: sure(famESP), Chip: "ESP32", USBChip: "FTDI FT2232",
			Confidence: "medium", Likely: true, Hint: "Pick the higher-numbered of the two COM ports."}
	case "067B:2303", "067B:23A3":
		return BoardGuess{Label: "USB-serial adapter (Prolific)", Families: bridgeFams, USBChip: "Prolific PL2303", Confidence: "low", Likely: true}
	}
	switch p.VID {
	case 0x303A:
		return BoardGuess{Label: "Espressif device", BoardID: "esp32-s3-generic", Family: famESP, Families: sure(famESP), USBChip: "Espressif USB", Confidence: "medium", Likely: true, NativeUSB: true}
	case 0x2E8A:
		chip := "RP2040"
		if p.PID == 0x0009 || p.PID == 0x000F || p.PID == 0x0010 {
			chip = "RP2350"
		}
		bid := "pico"
		if chip == "RP2350" {
			bid = "pico2"
		}
		return BoardGuess{Label: "Raspberry Pi " + chip + " (running firmware)", BoardID: bid, Family: famRP2, Families: sure(famRP2), Chip: chip,
			USBChip: "Built-in USB", Confidence: "high", Likely: true, NativeUSB: true,
			Hint: "Flashie will try to reboot it into BOOTSEL mode. If that doesn't work (e.g. MicroPython), hold BOOTSEL while plugging it in."}
	case 0x239A:
		return BoardGuess{Label: "Adafruit board (ESP32-S2/S3 or RP2040)", BoardID: "", Families: []string{famESP, famRP2}, USBChip: "Built-in USB",
			Confidence: "low", Likely: true, NativeUSB: true, Hint: "Pick your board below. RP2040 Feathers: hold BOOT and tap RESET to get the drive."}
	case 0x2341, 0x2A03:
		return BoardGuess{Label: "Arduino board", BoardID: "", Families: sure(famAVR), USBChip: "Arduino", Confidence: "low", Likely: true,
			Hint: "Not one Flashie recognises by ID — pick it from the list."}
	case 0x1A86:
		return BoardGuess{Label: "Board with a WCH USB chip", Families: bridgeFams, USBChip: "WCH", Confidence: "low", Likely: true}
	case 0x10C4:
		return BoardGuess{Label: "Board with a Silicon Labs USB chip", Families: bridgeFams, USBChip: "Silicon Labs", Confidence: "low", Likely: true}
	case 0x0403:
		return BoardGuess{Label: "FTDI USB-serial device", Families: bridgeFams, USBChip: "FTDI", Confidence: "low", Likely: true}
	case 0x0483:
		return BoardGuess{Label: "STMicroelectronics device", Families: sure(famSTM32), USBChip: "ST", Confidence: "low", Likely: true,
			Hint: "Flashie's STM32 support uses the UART bootloader via a USB-serial adapter (USB DFU comes later)."}
	}
	return BoardGuess{Label: "Unknown USB serial device", Families: bridgeFams, Confidence: "low", Likely: true,
		Hint: "Pick your board from the list if you know it, or press “Identify chip”."}
}

func guessDrive(d uf2Drive) BoardGuess {
	chip := d.chipName()
	g := BoardGuess{Family: famRP2, Families: sure(famRP2), Chip: chip, USBChip: "UF2 drive " + d.Letter, Confidence: "high", Likely: true, NativeUSB: true}
	switch chip {
	case "RP2040":
		g.Label, g.BoardID = "Raspberry Pi Pico / RP2040 in BOOTSEL mode", "pico"
	case "RP2350":
		g.Label, g.BoardID = "Raspberry Pi Pico 2 / RP2350 in BOOTSEL mode", "pico2"
	default:
		g.Label = "UF2 bootloader drive (" + chip + ")"
		g.Hint = "Not a Raspberry Pi chip — Flashie will copy .uf2 files to it as-is."
	}
	return g
}
