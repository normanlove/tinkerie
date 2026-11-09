# Flashie — firmware flasher

Part of a free maker tool suite. Made by **nrmluvdev**.

Flashie is a single-file Windows app for flashing microcontroller boards. It speaks every bootloader protocol itself, so it needs no esptool, avrdude, stm32flash, Python or extra DLLs. You only need the board's normal USB-serial driver.

## What it can flash

| Family | Boards (examples) | How | File |
|---|---|---|---|
| ESP32, S2, S3, C3, C6, H2 | DevKitC (WROOM/WROVER), CYD, XIAO, M5Stack, LilyGO T-Display / T-Deck, Nano ESP32 | ESP ROM bootloader over serial, compressed and MD5-verified | `.bin` (or `.hex`) |
| ESP8266 | NodeMCU v2/v3, Wemos D1 Mini, ESP-01 | ESP ROM bootloader at 115200, with each block checksummed | `.bin` |
| RP2040 / RP2350 | Pico, Pico W, Pico 2, Pico 2 W, RP2040-Zero, XIAO RP2040 | Copies the file to the UF2 drive. Flashie can reboot a running Pico into BOOTSEL itself | `.uf2` (a `.bin` is converted for you) |
| Arduino AVR | Uno, Nano (new and old bootloader), Pro Mini, Mega 2560, Leonardo, Micro, Pro Micro | STK500 v1 (Optiboot), STK500 v2 (Mega), AVR109 (Leonardo, via the 1200-baud double-reset), then read back to verify | `.hex` |
| CYD (Cheap Yellow Display) | 20+ Sunton/Guition models from 1.28" round to 7" — see below | ESP ROM bootloader | `.bin` |
| STM32 | Blue Pill (F103), Black Pill (F401/F411), most STM32s | ST UART bootloader (AN3155) at 8E1: mass erase, write, read back to verify, then start | `.bin` or `.hex` |

## Firmware library

Step 2 has two tabs: **Firmware library** and **My own file**. The library is filtered to the board you picked or identified, and shows builds made for that exact board (CYD builds, for example) first. Each entry shows its version and release date, a description, the boards it supports, links to the project and release page, every file it will write with its address, and how each file is verified.

| Firmware | Where Flashie gets it | Checksum |
|---|---|---|
| MicroPython | micropython.org download pages (latest stable, generic builds) | not published — fingerprinted in cache |
| CircuitPython | downloads.circuitpython.org (version from the GitHub release) | not published — fingerprinted in cache |
| ESPHome (ready to adopt) | esphome/firmware GitHub release (`*.factory.bin`) | GitHub SHA-256 |
| WLED | wled/WLED GitHub release, plus WLED's v4 bootloader on classic ESP32 | GitHub SHA-256 (app) |
| Tasmota | ota.tasmota.com release folder (`*.factory.bin`) | not published — fingerprinted in cache |
| Meshtastic | meshtastic/firmware GitHub release zip → your board's `factory.bin` / `.uf2` | GitHub SHA-256 (zip) + CRC-32 |
| NerdMiner v2 (incl. CYD builds) | BitMaker-hub/NerdMiner_v2 `bin/<board>/0x…_*.bin` | git blob SHA-1 |
| Grbl / Grbl-Mega | gnea/grbl and gnea/grbl-Mega release `.hex` | GitHub SHA-256 |
| Klipper, Marlin | no ready-made binaries — Flashie explains how to build them, then you flash the result with “My own file” | — |

Downloads go to `%LOCALAPPDATA%\nrmluvdev\Flashie\firmware`. Every cached file's SHA-256 is checked again before each reuse. If a file doesn't match the checksum its source published, Flashie refuses to flash it. Flashie also reads the chip ID in every ESP image header and won't flash firmware built for a different chip (for example S3 firmware onto a classic ESP32).

## Using it

1. **Pick board**: Flashie lists COM ports and Pico BOOTSEL drives, with a guess at what each one is based on its USB ID. Some ports use a generic USB chip (CH340, CP2102, FTDI). For those, press **Identify chip**: Flashie tries the ESP, Arduino and STM32 bootloaders and reports which one answered. You can also choose your board from **Board type**.
2. **Pick firmware**: choose or drop a file. Flashie recognises the type (ESP app, bootloader or merged image, UF2, Intel HEX, STM32 vector table, …) and suggests where it goes. It warns you if the file is for a different chip.
3. **Flash**: you get progress, plain-English status messages and verification, and the board restarts with the new firmware.

## Firmware Creator (the **Create** tab)

A drag-and-drop block editor, Scratch-style, that writes a real program for the board you picked on the Flash tab.

- **Blocks**: start, forever, repeat, repeat-while, if / if-else, waits; pins (on/off, toggle, read, analog %, PWM brightness); the built-in LED (Flashie knows where it is on each board, including active-low and NeoPixel LEDs); NeoPixel strips; the CYD's RGB LED; buttons; variables; maths and logic; print to the console; append to a file; Wi-Fi connect, IP address and a tiny web server; DHT11/DHT22, I²C setup/scan/register read; servos; SSD1306 OLEDs; and CYD screens (fill, text, boxes, touch x/y).
- **Split view**: blocks on the left, the generated code on the right, updated as you build. Hover a block to highlight its lines; hover a line to highlight its block. **Type code** switches to a syntax-highlighted editor so you can write your own code. Typed code doesn't go back into blocks.
- **Teaching mode**: hover any block, palette entry, line of code or pin to get a plain-English explanation in the side panel. Each line also says which block made it.
- **Pin pickers** use the selected board's real pinout and sort the pins into four groups: *good to use*, *use with care*, *used by the board*, and *not suitable for this*. They cover boot-strapping pins, input-only pins (ESP32 34–39), flash/PSRAM/USB/serial pins, ADC2-with-Wi-Fi, PWM-capable Arduino pins, and every pin the selected CYD's display, touch panel and SD card use. If a block uses a bad pin, the block gets a ⚠ and the problem is listed under the canvas.
- **Upload & run**: Flashie checks for MicroPython on the board. If it's missing, Flashie offers to install the official build from the firmware library. On a Pico it flashes the BOOTSEL drive, then waits for the new COM port. Flashie then copies `main.py`, plus any driver it needs (`flashie_cyd.py`, `flashie_oled.py`), soft-reboots the board and streams the output to the console. You can type into the REPL from the console, and Ctrl-C/Stop interrupts the program.
- **Arduino Uno / Nano / Mega / Leonardo / Pro Mini / Micro**: these boards are too small for MicroPython, so the same blocks become Arduino C++ (`setup()` + `loop()`). The first compile downloads the official `arduino-cli` from downloads.arduino.cc, along with the AVR core and the libraries the blocks need (Servo, DHT, NeoPixel, SSD1306/GFX), into `%LOCALAPPDATA%\nrmluvdev\Flashie\arduino`. Compiling then flashes with Flashie's own STK500/AVR109 code. Wi-Fi, file and CYD-screen blocks are greyed out, with an explanation.
- **Templates**: Blink, Button + LED, Wi-Fi web server (ON/OFF links), CYD hello + touch drawing, and a temperature logger. Each template is built for the current board's LED, boot button and free pins.
- **Projects**: work autosaves in the app. *File* menu → save the code (`main.py` / `.ino`), save or open a `.flashie` project, tidy the blocks, or clear everything.

Supported in the Creator: ESP32, S2, S3, C3, C6, ESP8266, RP2040/RP2350 (MicroPython), and AVR Arduinos (C++). STM32 isn't supported. CYD screen blocks work on the SPI CYDs (2.4", 2.8" in every version, 3.2", 3.5", 1.9"). The parallel-bus and RGB-panel CYDs (2.2", 4.0"–7") and the round GC9A01 aren't supported by the MicroPython driver.

## Cheap Yellow Display (CYD) database

Choose your CYD from the **Cheap Yellow Display** group in Board type. The model number is printed on the back of the board. Flashie then shows:

- a spec card with the display controller and bus, the touch controller, the chip/module, flash/PSRAM and the USB connector
- the full pinout, grouped by display, touch, SD card and extras
- **Copy pins as #defines**, which copies the pins as generic `CYD_…` defines plus a ready-made TFT_eSPI `User_Setup.h` block for the SPI-display boards
- firmware matching: when you pick a .bin, Flashie looks for CYD model names (`2432S028R`, `CYD2USB`, `8048S070C`, …) and driver names inside the file and in its filename. It warns you if the firmware is for a different CYD or revision, and offers one click to switch boards.

| Size | Models | Display | Touch |
|---|---|---|---|
| 1.28" round | 2424S012C (ESP32-C3) | GC9A01A SPI | CST816S |
| 1.9" | 1732S019C (S3) | ST7789 SPI | GT911 |
| 2.2" | 2432S022C | ST7789 8-bit parallel | CST816S |
| 2.4" | 2432S024R / C | ILI9341 SPI | XPT2046 / CST820 |
| 2.8" | 2432S028R (micro-USB), Rv2 (USB-C), Rv3 "CYD2USB" (two USB ports) | ILI9341 / ILI9341 / ST7789 SPI | XPT2046 (own SPI bus) |
| 3.2" | 2432S032R / C | ST7789 SPI | XPT2046 / GT911 |
| 3.5" | 3248S035R / C | ST7796 SPI | XPT2046 / GT911 |
| 4.0" square | 4848S040C (S3) | ST7701 RGB | GT911 |
| 4.3" | 4827S043R / C, 8048S043R / C (S3), JC4827W543C | ST7262 RGB / NV3041A QSPI | XPT2046 / GT911 |
| 5" | 8048S050R / C (S3) | ST7262 RGB | XPT2046 / GT911 |
| 7" | 8048S070R / C (S3) | ST7262 RGB | XPT2046 / GT911 |

Most pin data comes from rzeldent/platformio-espressif32-sunton. The RGB-panel boards were also cross-checked against Arduino_GFX, ESPHome and LovyanGFX configs, which name the R and B pins the opposite way round from esp32-smartdisplay. Each board lists where its data came from. The JC4827W543C's touch pins aren't included because they couldn't be verified.

## Board-specific tips

- **ESP32 won't connect**: hold BOOT, tap RESET/EN, let go of BOOT, then press Flash.
- **ESP32-CAM / ESP-01**: hold GPIO0 to GND while resetting or powering up.
- **Pico**: hold BOOTSEL while plugging it in, or let Flashie reboot it. Flashie can reboot boards running C/C++ SDK or Arduino firmware, but not boards running MicroPython.
- **Old Nano clones**: Flashie tries 115200 and 57600 automatically. "Nano (old bootloader)" tries 57600 first.
- **Pro Mini**: connect a USB-serial adapter, wiring DTR to DTR/GRN so Flashie can reset the board automatically.
- **Leonardo / Micro**: Flashie double-resets the board for you. If that fails, double-tap RESET and press Flash straight away.
- **STM32**: wire the adapter TX → PA10, RX → PA9 and GND → GND, set BOOT0 = 1 and press RESET. Set BOOT0 back to 0 afterwards. Read-protected chips have to be unlocked with ST's tools first.

## Building from source

Go 1.22 or newer, using the standard library only:

```
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -H windowsgui" -o Flashie.exe .
```

`go test .` runs every protocol against a simulated bootloader. `FLASHIE_DEMO=1` starts the full UI with simulated boards and an offline copy of the firmware sources: an ESP32, an ESP8266, an S3, an Uno, a Mega, a Leonardo, an STM32 on an FTDI adapter, a running Pico and a Pico 2 drive.

## Source layout

| File | What it does |
|---|---|
| `esp.go` | ESP32/ESP8266 ROM protocol: SLIP framing, sync, resets, chip detection, compressed/plain flashing, MD5 |
| `avr.go` | STK500 v1, STK500 v2 and AVR109, plus the AVR chip table |
| `stm32.go` | STM32 UART bootloader (AN3155) |
| `uf2.go` / `drives_windows.go` | Finding UF2 drives and copying to them; .bin/.hex → UF2 conversion |
| `fwfile.go` | Intel HEX parser and UF2 reader/writer |
| `jobs.go` | Device list, universal Identify, and sending each flash job to the right protocol |
| `library.go` | Firmware library catalogue, release lookups (GitHub API, micropython.org, CircuitPython, Tasmota), downloads, cache and checksums |
| `libflash.go` | Flashing a library build (multi-part ESP, UF2, AVR) |
| `repl.go` | MicroPython raw-REPL: detect, upload files, run, live console session |
| `arduino.go` / `exec_*.go` | arduino-cli download, private build folder, compile + upload for the Creator |
| `creator.js` / `creator.css` | The Firmware Creator: blocks, code generators, pin database, teaching text, run flow |
| `mpy/flashie_cyd.py` / `mpy/flashie_oled.py` | Small MicroPython drivers copied to the board (CYD SPI screens + XPT2046/GT911/CST816 touch; SSD1306) |
| `sim_mpy.go` | Simulated MicroPython board for tests and demo mode |
| `mocklib.go` | Offline stand-in for the firmware sources (tests and demo mode) |
| `cyd.go` | CYD database (display/touch/pinout) and firmware ↔ CYD matching |
| `boards.go` | Board catalogue (~90 boards incl. CYDs) and USB VID/PID → board guesses |
| `link.go` | Byte-stream helpers |
| `serial_windows.go` / `ports_windows.go` | Win32 serial (8N1/8E1, DTR/RTS) and the COM port + VID/PID list from the registry |
| `main.go` / `window_windows.go` | Local server, progress events, app window and lifetime |
| `sim.go` / `sim_more.go` | Simulated bootloaders for tests and demo mode |
| `ui.html` | The whole interface (embedded in the .exe) |

## Write image

The **💾 Write image** tab puts a whole operating system onto an SD card or USB drive, like Raspberry Pi Imager: choose an OS from Raspberry Pi's official list (or your own .img/.iso, plain or in .zip/.xz/.zst/.gz/.bz2), choose a removable drive, press Write. Flashie downloads and checks the image, asks for your password, writes it, and reads it back to check it. Your computer's own disk is never listed.

- **⚙ OS settings** set up Raspberry Pi OS (and Ubuntu for Raspberry Pi) before its first start: hostname, username and password, Wi-Fi, SSH (password or public keys), time zone and keyboard. Newer images get cloud-init files, older ones a one-time `firstrun.sh`, the same as Raspberry Pi Imager. Passwords aren't saved, and the login password goes on the card only as a hash.
- **Windows 10/11 installer ISOs** become a proper USB installer, like Rufus: Flashie formats the drive (FAT32, plus exFAT when `install.wim` is over 4 GB) and copies the files, for PCs that start in UEFI mode.
- **Linux ISOs** (Ubuntu, Fedora, Debian, Mint…) are written as they are. CD-only ISOs get a warning.
