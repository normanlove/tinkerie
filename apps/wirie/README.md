# Wirie

**A free, beginner-friendly circuit board designer — created by nrmluvdev.**
Part of the nrmluvdev maker tool suite.

Wirie has three tabs:

| Tab | What it does |
|---|---|
| **PCB** | Design a real 2-layer circuit board: place parts, connect pins, auto-route, check, export Gerbers. |
| **Boardie** | Plan the same circuit on stripboard or perfboard: auto-place, auto track-cuts, auto wire links, printable build plan. |
| **Pinnie** | Interactive pinouts for 26 boards & chips (Pico, ESP32/S3/C3, ESP8266, Arduino, Pi header, ATmega, ATtiny, Teensy, STM32…) — hover a pin to see PWM/ADC/I²C/SPI/UART/touch and gotchas, filter by function, place on your PCB. |
| **Powerie** | Battery power budget planner: battery & regulator presets, each part's awake/sleep current, wake-up schedule, regulator losses and dropout, runtime estimate, breakdown chart and tips. Import parts from your PCB. Saved with the project. |
| **Calcie** | Everyday electronics maths: Ohm's law, resistor colours, E-series, LED resistor, divider, 555, RC/LC, battery life, wire gauge, PCB trace width, series/parallel, capacitor codes. |

## Features
- 700+ parts (plus your own — see Part Creator): Raspberry Pi Pico / Pico W / Pico 2, Arduino Nano / Pro Micro / Pro Mini, ESP32 DevKit (30 & 38 pin), ESP32-C3 SuperMini, NodeMCU, Wemos D1 Mini, Seeed XIAO, Teensy 4.0, STM32 Blue Pill, ESP-01, Raspberry Pi 40-pin GPIO header, ~45 sensor/display/radio modules, pin headers 1x1–1x40 and 2x2–2x40, JST, screw terminals, passives (THT + SMD), LEDs, transistors, regulators, DIP/SOIC/TSSOP/TQFP chips, switches, mounting holes.
- Every module pin is labelled (GP4, SDA, 3V3…), and **Smart connect** wires a sensor to your microcontroller in one click (I²C, SPI, UART, power, GPIO).
- Board templates: Raspberry Pi HAT & Zero pHAT (header and holes in the right place), Pico / Nano / ESP32 carriers, D1 Mini shield, round boards.
- Two-layer A* **auto-router** with rip-up & retry, wider power tracks, vias.
- Ground fill with thermal reliefs, design-rule check, realistic board preview, flip view.
- Export: Gerber RS-274X X2 (copper, mask, silk, paste, outline), Excellon drills, BOM and pick-and-place CSV — zipped and ready for JLCPCB / PCBWay / OSH Park / Aisler.
- 1:1 printouts for home etching, PNG board picture.
- Projects are saved in `Documents\Wirie\Projects`, exports in `Documents\Wirie\Exports`, and your last session auto-restores.

## More parts
Displays & LED matrices (OLED/TFT/e-paper/LCD/7-seg/WS2812 panels), a big surface-mount range (0201–2512, SOT/SOD, SOIC/SSOP/TSSOP/MSOP, QFN, LQFP, DPAK/D2PAK…), named chips, crystals & oscillators, FFC/FPC (incl. Raspberry Pi camera/display), PCIe card-edge fingers, ISP/SWD/JTAG ribbon headers, USB-C/USB-A/DC/RJ45/D-sub/HDMI connectors, keyboard switches, encoders, sensors, comms and motor modules, and a **Retro & Consoles** section (cartridge edges & slots, controller ports, Z80/6502/SID/SN76489/EPROM/SRAM/74xx and more). Parts marked **!** should be measured against your real part before ordering.

## Part Creator
Library → My Parts → **Create a part** (or Ctrl+Shift+P). Build from a template (pin rows/modules, DIP/SOIC/SSOP/TSSOP/QFP/QFN chips, 2-pin parts) or place pads by hand, name every pin, set the body, and give it a height and case opening for Makie. Parts are saved to `Documents\Wirie\Parts\*.wpart.json`, can be exported/imported, and are embedded in any project that uses them.

## Makie case data
Export → "Case data for Makie" writes `Documents\Wirie\Exports\<project>.makie.json` (a copy without textures is also inside the Gerber ZIP). The format is documented in **MAKIE_FORMAT.md**, and **example.makie.json** is a sample.

## Building
Only the Go standard library is used (Go 1.22+).

```
# Windows (the shipped build) — no console window
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w -H windowsgui" -o Wirie.exe .

# macOS / Linux (porting) — launcher_other.go handles the window
go build -o wirie .
```

The Windows icon + version info (Company / Copyright: nrmluvdev) live in `rsrc_windows_amd64.syso`.
Regenerate it after changing the logo with `go run tools/mkrsrc.go` (icons come from `assets/icon*.png`).

## How it's put together
- `main.go` — tiny local server (127.0.0.1 only), save/load/export API, single-instance, auto-quit when the window closes.
- `launcher_windows.go` / `launcher_other.go` — opens the editor as a chromeless app window (Edge/Chrome `--app`), falling back to the default browser. This is the only platform-specific code.
- `web/` — the whole editor, embedded into the binary:
  - `lib.js` parts library & board templates · `core.js` project model, nets, connectivity, DRC
  - `route.js` auto-router · `export.js` Gerber/drill/BOM/SVG · `render.js` canvas renderer
  - `strip.js` Boardie · `calc.js` Calcie · `ui.js` the interface · `font.js` silkscreen stroke font

Environment switches for development: `WIRIE_NO_WINDOW=1` (don't open a window), `WIRIE_STAY=1` (don't auto-quit).

## A note on footprints
Parts marked **!** in the library (some modules and connectors) vary between manufacturers. Always measure your actual module, or do a 1:1 paper print, before ordering boards.

---
Wirie © 2026 nrmluvdev
