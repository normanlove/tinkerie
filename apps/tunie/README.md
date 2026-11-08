# Tunie — 3D printer calibration helper

**Created by nrmluvdev** · part of the free nrmluvdev maker suite (Wirie · Scopie · Makie · Flashie · Tunie)

Tunie walks you through calibrating a 3D printer: **print this → measure this → type the number here → here's your new setting.**
Every test generates its own G-code (no slicer needed), explains in plain English what it checks, why it matters,
what good looks like and how to measure it, then turns your measurements into the exact command or config line to apply.

## Running it

Double-click `Tunie.exe`. It opens in its own window (Microsoft Edge "app mode", which ships with Windows 10/11).
Nothing to install, no DLLs, no runtime. Close the window and Tunie exits.

Your data lives in `%APPDATA%\nrmluvdev\Tunie\` and exports go to `Documents\Tunie\`.

Command-line options:

| Option | What it does |
|---|---|
| `--from makie` | Shows that Tunie was opened from another suite app |
| `--printer "Name"` / `--filament "Brand Type"` | Selects that printer / filament |
| `--page tolerance` | Opens a test (`pid`, `esteps`, `flow`, `temp`, `retraction`, `pa`, `maxflow`, `shaper`, `firstlayer`, `dims`, `overhang`, `bridge`, `tolerance`, `stringing`) or page (`mesh`, `belt`, `export`, …) |
| `--no-browser` | Don't open a window; browse to the printed address yourself |
| `--port 47821` | Local port (Tunie only listens on 127.0.0.1) |

## What's inside

- **Recommended-order checklist** per printer: PID → E-steps → flow → temperature → retraction → pressure/linear advance → input shaping → dimensional accuracy.
- **Tests:** PID (hotend & bed), E-steps / rotation distance, flow (single-wall cube or flow-step patches), temperature tower (labelled bands with bridge and stringing gap),
  retraction tower (distance or speed), pressure/linear advance (line pattern or tower), max volumetric flow, input shaper / ringing tower (+ Klipper accelerometer guide),
  first-layer patches, XYZ cube, multi-size accuracy frame with skew, overhang, bridging, tolerance / fit (peg in hole), stringing.
- **Bed mesh viewer:** reads Marlin (`M420 V`, `G29 T`, UBL), Klipper (`BED_MESH_OUTPUT`, Moonraker) and RRF `heightmap.csv`; coloured 3D heightmap; screw-turn suggestions.
- **Belt tension helper:** listens through the PC microphone, shows the belt's frequency, compares belts.
- **Printers:** 40 presets (Creality, Prusa, Bambu Lab, Voron, Sovol, Elegoo, Anycubic, FLSUN, Artillery, deltas, tiny DIY) plus fully custom profiles; rectangular or round beds.
- **Firmware:** Marlin, Klipper, RepRapFirmware, Prusa (MK3 and Buddy), Bambu Lab — correct commands for each.
- **Connections (optional):** USB serial (Marlin/Prusa/RRF), Moonraker (Klipper), OctoPrint. Otherwise save `.gcode` to the SD card.
- **Results:** history per printer and filament, reloadable profiles, summary card PNG, PrusaSlicer / OrcaSlicer / Bambu Studio presets, Cura settings list, firmware command batch / printer.cfg snippet.
- **Suite link:** the active profile (fit clearances, shrinkage, line width…) is written to `%APPDATA%\nrmluvdev\Suite\tunie-active.json` for Makie. See `docs/MAKIE_INTEGRATION.md`.

## Building from source

Needs Go 1.22+ and nothing else (standard library only).

```
build.bat            (on Windows)
./build.sh           (on Linux/macOS — cross-compiles Tunie.exe)
```

or by hand:

```
go run ./tools/mkres                                        # (optional) redraw the icon + Windows resource
GOOS=windows GOARCH=amd64 go build -ldflags "-H windowsgui -s -w" -o Tunie.exe .
go test ./...                                               # G-code, parser and connection tests
```

The same source builds on Linux and macOS (`go build .`) for porting; serial there opens `/dev/tty*` directly.

## Source map

| File | Purpose |
|---|---|
| `main.go` | Local server, app window, single-instance hand-over, shutdown when the window closes |
| `gcode.go` | G-code writer: moves, extrusion maths, retraction, shapes, start/end per firmware |
| `font.go` | 5×7 pixel font for flat and embossed labels |
| `tests.go` | Every calibration test's generator |
| `conn.go` | USB serial protocol, Moonraker and OctoPrint clients, print streaming |
| `store.go` | Saving data, exports, suite file |
| `platform_windows.go` / `platform_other.go` | Window, folders, SD cards, COM ports (Win32 via syscall) |
| `web/` | The interface (HTML/CSS/JS, embedded into the exe) |
| `tools/mkres` | Draws the icon and writes the Windows icon resource |

Free to use. Made with care by **nrmluvdev**.
