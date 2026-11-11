# Wirie

**A free, beginner-friendly circuit board designer — created by nrmluvdev.**
Part of the nrmluvdev maker tool suite.

Wirie has these tabs:

| Tab | What it does |
|---|---|
| **PCB** | Design a real 2-layer circuit board: place parts, connect pins, auto-route, check, export Gerbers. |
| **Boardie** | Plan the same circuit on stripboard or perfboard: auto-place, auto track-cuts, auto wire links, printable build plan. |
| **Pinnie** | Interactive pinouts for 26 boards & chips (Pico, ESP32/S3/C3, ESP8266, Arduino, Pi header, ATmega, ATtiny, Teensy, STM32…) — hover a pin to see PWM/ADC/I²C/SPI/UART/touch and gotchas, filter by function, place on your PCB. |
| **Powerie** | Battery power budget planner: battery & regulator presets, each part's awake/sleep current, wake-up schedule, regulator losses and dropout, runtime estimate, breakdown chart and tips. Import parts from your PCB. Saved with the project. |
| **Keebie** | Custom mechanical keyboard PCBs: lay out the keys (presets, grids, KLE import, any key size, ISO Enter, angles, splits, arcs, rotary knobs, a small OLED screen), pick a switch (MX/Choc, soldered or hot-swap, Alps) and a controller, and Keebie builds the whole board: switch footprints, diodes, the matrix, the controller and the support circuitry it needs, per-key RGB, stabilisers and mounting holes. Also writes QMK/ZMK firmware files and a switch plate (SVG/DXF), sends the board to Makie for a printed case and to Flashie for ready-made firmware. |
| **Calcie** | Everyday electronics maths: Ohm's law, resistor colours, E-series, LED resistor, divider, 555, RC/LC, battery life, wire gauge, PCB trace width, series/parallel, capacitor codes. |

## Features
- 700+ parts (plus your own — see Part Creator): Raspberry Pi Pico / Pico W / Pico 2, Arduino Nano / Pro Micro / Pro Mini, ESP32 DevKit (30 & 38 pin), ESP32-C3 SuperMini, NodeMCU, Wemos D1 Mini, Seeed XIAO, Teensy 4.0, STM32 Blue Pill, ESP-01, Raspberry Pi 40-pin GPIO header, ~45 sensor/display/radio modules, pin headers 1x1–1x40 and 2x2–2x40, JST, screw terminals, passives (THT + SMD), LEDs, transistors, regulators, DIP/SOIC/TSSOP/TQFP chips, switches, mounting holes.
- Every module pin is labelled (GP4, SDA, 3V3…), and **Smart connect** wires a sensor to your microcontroller in one click (I²C, SPI, UART, power, GPIO).
- Board templates: Raspberry Pi HAT & Zero pHAT (header and holes in the right place), Pico / Nano / ESP32 carriers, D1 Mini shield, round boards.
- Two-layer A* **auto-router** with rip-up & retry, wider power tracks, vias.
- Turn parts to **any angle**: type it in the Angle box, drag the round handle above a selected part (15° steps, Shift for 1°), or R / Shift+R for 90° / 15°.
- **Auto-fix** in the Check tab: redraws tracks that are too close or shorted, moves parts that clash or sit off the board, widens thin tracks and routes what's left. *Auto-fix mode* does it every time the check runs. One Ctrl+Z undoes it.
- Ground fill with thermal reliefs, design-rule check, realistic board preview, flip view.
- Export: Gerber RS-274X X2 (copper, mask, silk, paste, outline), Excellon drills, BOM and pick-and-place CSV — zipped and ready for JLCPCB / PCBWay / OSH Park / Aisler.
- 1:1 printouts for home etching, PNG board picture.
- Projects are saved in `Documents\Wirie\Projects`, exports in `Documents\Wirie\Exports`, and your last session auto-restores.

## More parts
Displays & LED matrices (OLED/TFT/e-paper/LCD/7-seg/WS2812 panels), a big surface-mount range (0201–2512, SOT/SOD, SOIC/SSOP/TSSOP/MSOP, QFN, LQFP, DPAK/D2PAK…), named chips, crystals & oscillators, FFC/FPC (incl. Raspberry Pi camera/display), PCIe card-edge fingers, ISP/SWD/JTAG ribbon headers, USB-C/USB-A/DC/RJ45/D-sub/HDMI connectors, keyboard switches, encoders, sensors, comms and motor modules, and a **Retro & Consoles** section (cartridge edges & slots, controller ports, Z80/6502/SID/SN76489/EPROM/SRAM/74xx and more). Parts marked **!** should be measured against your real part before ordering.

## Keebie (custom keyboards)
- **Layout:** start from a preset (macropad, numpad, 40% ortho, 60% ANSI/ISO, 65%, 75%, TKL, full size, Alice, Corne, Lily58, Sweep), a rows × columns grid, a number of keys, or a [KLE](https://www.keyboard-layout-editor.com) paste. Drag keys, add any size (1u–7u, tall keys, ISO Enter, stepped Caps), rotate them, and set the row/column by hand if you like.
- **Shape:** an outline that hugs the keys, a rectangle or your own corners; an arc, an angled Alice-style split or two separate halves joined by TRRS.
- **Knobs and a screen:** any key can be a rotary knob (EC11 encoder with push switch; its press joins the matrix, its two turn pins get controller pins of their own). A 0.91" (128 × 32) or 0.96" (128 × 64) SSD1306 OLED can be added next to the controller, wired to its I2C pins.
- **Switches and caps:** MX or Kailh Choc (soldered or hot-swap sockets), Alps; standard or custom key spacing; keycap profiles (Cherry, OEM, SA, DSA, XDA, MT3, KAT…). **Switch pins** (MX): *5-pin holes* fit both 5-pin (PCB-mount) and 3-pin switches; *3-pin only* leaves out the peg holes for switches held by a plate. **LED facing**: *south* (towards you, the modern choice that suits Cherry-profile keycaps) or *north* (the classic Cherry way); it turns every switch footprint so the LED, its hole and the hot-swap sockets move together. The switch footprint can also be turned 0/90/180/270° globally or per key for custom-height and side-mounted builds.
- **Controllers:** Pro Micro, Elite-C, RP2040 Pro Micro boards (KB2040, Sea-Picro, Elite-Pi…), Raspberry Pi Pico, RP2040-Zero, XIAO RP2040, WeAct Black Pill, nice!nano and XIAO nRF52840 (wireless, with battery and switch), or bare chips: ATmega32U4, RP2040, STM32F072, STM32F411. Bare chips get their USB-C port (fuse and CC resistors), crystal, regulator, decoupling, QSPI flash, and reset/boot buttons added automatically.
- **Diodes, LEDs and parts:** every key gets a diode (and, with per-key RGB, an SK6812 MINI-E LED). *Diode layout* picks how they sit: upright beside the switch (right or left), flat below or above it, or at 45° in the corner; or **on show**, off the keys altogether in a bank on the front: a straight row, a zig-zag, a wave, an arch, a chevron or slanted at 45° above the keys, or a row, zig-zag or smile below them. Patterns that change column by column (alternating sides, herringbone, staggered) suit surface-mount diodes; long through-hole diodes go beside the switch. *Part spacing* (tight, comfortable, roomy) sets how far apart diodes, rows and the controller's parts sit. The board grows to fit the bank and the controller moves out of its way; each diode then has a longer track to its key, so auto-routing takes a little longer. For the layouts round each switch, Keebie checks each spot clears the switch's pads and holes and the LED, and uses the next-best spot if not. *Diodes on* puts them on the back (hidden) or the front (on show). An optional 100nF capacitor can sit beside each LED. The **Parts** view above the keyboard draws every diode, LED and capacitor where the build will put them; LEDs (and, by default, diodes) are on the back of the board, so press **Bottom** on the PCB tab (or the link after building) to see them there.
- **Through-hole by default:** 1N4148 (DO-35) diodes, a 6×6mm reset button, an SS12D00 power switch, 3mm LEDs and leaded resistors. Per-key RGB LEDs and bare-chip circuitry stay surface-mount (there's no through-hole equivalent).
- **Status lights:** a power light, and an activity light on a spare pin that flashes as keys are sent (Flashie's firmware, or a small C file with the QMK files).
- **Standoffs:** M2, M2.5 or M3 at a chosen height, placed for you or by hand (click the board to add one, click one to remove it).
- **Screens:** 0.91in, 0.96in, 1.3in and 2.42in OLEDs and a 16×2 LCD on I2C; the nice!view and colour TFTs (ST7789 1.3/1.54/1.9in, GC9A01 round, ST7735 1.8in) on SPI pins of their own.
- **Place on their own:** the screen, status lights and reset button can each have their own block, dragged anywhere on the board (the outline grows to reach it, with a bridge so the board stays one piece).
- **Parts list:** everything to buy for every board of the keyboard; the Export window's Parts list does the same for any board.
- **USB:** controller footprints show which end the USB socket is, on the board and in silkscreen.
- **The controller:** click it to select it, then drag it, nudge it with the arrow keys or the Move boxes, and turn it (R turns 15°, or ↺/↻ 90°); the board outline follows. *Controller's parts* lays its support parts out packed tightly, in neat rows of alike parts, or in one tidy line. Keebie warns if the controller ends up over the keys.
- **New keyboard / Close keyboard** start a fresh keyboard (from a layout or from nothing) or close the current one. Both clear the board built from it on the PCB tab too; Ctrl+Z undoes it straight after.
- **Build the PCB** places everything on the PCB tab with named nets; Auto-route and Check work as usual. Firmware: QMK `keyboard.json` + `keymap.json`, or a ZMK overlay for wireless boards.
- **Case in Makie** saves the board's case file (with the switch plate data) and opens it in Makie, which makes an open-top case, a printed switch plate and feet for a typing angle.
- **Firmware in Flashie** saves a `.keebie.json` (matrix pins, knobs, screen, LEDs and layout) to `Documents\Wirie\Exports` and opens Flashie's Keyboard tab, where you set the keymap and Flashie compiles and flashes Arduino firmware (ATmega32U4, RP2040 and STM32 controllers).
- Limits: fine-pitch bare chips (RP2040, STM32) usually need a few tracks finished by hand after auto-routing; split boards are built one half at a time; stabiliser plate cut-outs are approximate; parts marked **!** (TRRS jack, bare chips) should be checked against the datasheet.

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
  - `route.js` auto-router · `fix.js` design-check auto-fix · `export.js` Gerber/drill/BOM/SVG · `render.js` canvas renderer
  - `strip.js` Boardie · `calc.js` Calcie · `keebie*.js` Keebie (layout, controllers, footprints) · `ui.js` the interface · `font.js` silkscreen stroke font

Environment switches for development: `WIRIE_NO_WINDOW=1` (don't open a window), `WIRIE_STAY=1` (don't auto-quit).

## A note on footprints
Parts marked **!** in the library (some modules and connectors) vary between manufacturers. Always measure your actual module, or do a 1:1 paper print, before ordering boards.

---
Wirie © 2026 nrmluvdev
