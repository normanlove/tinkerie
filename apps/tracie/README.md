# Tracie

**Serial monitor + logic analyser on one shared timeline.**
Part of the free nrmluvdev hobbyist suite: Wirie · Scopie · Makie · **Tracie**.

Made by **nrmluvdev**. Free for everyone.

---

## Run it

Unzip anywhere and double-click **Tracie.exe**. It opens in its own window and needs no install.
(Windows may show a SmartScreen "unrecognised app" prompt because the exe isn't code-signed: click *More info → Run anyway*.)

Settings live in `%APPDATA%\Tracie`, and CSV logs are saved to `Documents\Tracie`.

## Serial tab

* COM port list shows each board's USB chip (Arduino, ESP32, Pico, CH340, CP210x, FTDI…). It refreshes when you plug a board in.
* Baud picker (9600 to 921600 or any custom rate), plus data bits, parity and stop bits.
* DTR/RTS toggles and a **Reset board** button for ESP32 and Arduino auto-reset.
* Auto-reconnect when native-USB boards (ESP32-S3, Pico) re-enumerate after a reset.
* Text view with millisecond timestamps, ESP-IDF ANSI colours, search filter and command history (↑/↓).
* Hex view: offset, direction and ASCII side by side.
* **Live plotter.** Prints like `temp:21.5,hum:40` or `1.2 3.4 5.6` are graphed automatically. Series can share one axis or be stacked with their own scales. Pause, 5 s to 5 min windows, hover readout.
* **Quick command buttons.** Add with **+**, right-click to edit. Escapes like `\n`, `\r` and `\x1B` work.
* **CSV logging** with timestamp, elapsed time, direction, raw line and every number parsed into its own column.
* Built-in **DEMO** board so you can try everything without hardware.

## Logic tab

* Supports cheap 24 MHz 8-channel "Saleae-compatible" analysers (Cypress FX2, VID 0925 PID 3881, among others) via the open-source **fx2lafw** firmware.
* 20 kHz to 24 MHz sample rates, captures of up to 100 M samples.
* Per-channel triggers (high, low, rising, falling, any edge), pre-trigger %, Repeat mode.
* Decoders for **UART** (baud, bits, parity, stop, inverted), **I²C** (start/restart/stop, address R/W, ACK/NACK) and **SPI** (modes 0–3, MSB/LSB, 4–32-bit words, CS polarity).
* Zoom with the mouse wheel and pan by dragging. A/B cursors show Δt and 1/Δt. Measurements give frequency, period, duty and edge count.
* Searchable decoded-data list: click a row to jump to it.
* Export as VCD (PulseView/GTKWave), decoded CSV, or PNG screenshot.
* **Demo signal generator** with UART, I²C, SPI and PWM on all 8 channels.

### One-time analyser setup
1. **Driver.** Run Zadig (zadig.akeo.ie), choose *Options → List All Devices*, select the analyser, pick **WinUSB**, then *Install/Replace Driver*.
2. **Firmware.** If PulseView is installed, Tracie finds the firmware automatically. Otherwise get `fx2lafw-saleae-logic.fw` (from sigrok's `sigrok-firmware-fx2lafw` package) and drop it on the *Firmware needed* box in the Logic sidebar.
3. Press **Capture** (or Space). The firmware uploads on the first capture.

## Timeline tab
Every serial line, plotted value, logic channel and decoded byte is timestamped on the same PC clock and shown on one zoomable timeline. Logic captures also show a **serial overlay** lane. Alignment is within a few milliseconds (USB latency).

## Building from source
Install Go 1.22+ and run `build.bat` inside `source\` (or `go build -ldflags "-H windowsgui" .`).
It's pure Go with no cgo and no third-party modules. The UI is HTML/CSS/JS embedded in the exe from `web/`.

| File | What it does |
|---|---|
| `main.go` | app startup, local UI server, window, events |
| `serial.go` | serial session, CSV logging, demo board |
| `serial_windows.go` | Win32 COM port driver |
| `win_setupapi_windows.go` | port and USB enumeration (SetupAPI) |
| `usb_windows.go` | WinUSB control and bulk streaming |
| `fx2.go` | fx2lafw firmware upload and acquisition protocol |
| `logic.go` | capture engine, trigger, edge index, VCD export |
| `decode.go` | UART / I²C / SPI decoders |
| `demo.go` | demo signal generator |
| `platform_other.go` | stubs for future macOS/Linux ports (demo devices work) |
| `web/` | the UI |
| `tools/` | scripts that regenerate the exe icon resource |

fx2lafw firmware © the sigrok project (GPL). It is not bundled; Tracie loads it from your sigrok/PulseView install or from a file you provide.
