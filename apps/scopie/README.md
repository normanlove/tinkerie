# Scopie — a beginner-friendly program for many oscilloscopes

**Created by nrmluvdev** · part of the free nrmluvdev maker suite (Wirie · Scopie · Makie · Flashie · Tunie · Tracie)

Scopie turns your PC into the screen and controls of an oscilloscope. It started as a front end
for the Hantek 6022BE and now supports many common USB, bench and handheld scopes — and every
button, menu and on-screen marker explains itself: hover over anything and a plain-English
explanation appears in the help bar under the screen. Press **F1** for the beginner's guide.

This version is a complete rewrite in Go of the old Python/Qt Scopie: one `Scopie.exe`, nothing to
install, no Python, no DLLs, no runtime.

## Running it

Double-click `Scopie.exe`. It opens in its own window (Microsoft Edge "app mode", which ships
with Windows 10/11). Close the window and Scopie exits.

If exactly one scope is plugged in, Scopie connects to it by itself. Otherwise use
**Connect…** (Ctrl+K), or **Demo mode** to learn everything with simulated signals.

| Where | What |
|---|---|
| `%APPDATA%\nrmluvdev\Scopie\` | settings, imported firmware, `scopie.log` |
| `Documents\Scopie\` | screenshots, CSV exports, saved setups |

Command-line options: `--demo` (start in demo mode), `--no-browser` (don't open a window; browse
to the printed address yourself), `--port 47831` (local port; Scopie only listens on 127.0.0.1).

## Supported oscilloscopes

| Family | Models | Connection | Notes |
|---|---|---|---|
| Hantek 6022 family | DSO-6022BE, 6022BL, 6021; Voltcraft DSO-2020; Sainsmart/BUUDAI DDS120; Instrustar ISDS-205B; YiXingDianZi MDSO | USB | Streams raw samples and triggers in software: hundreds to thousands of waveforms per second. Open firmware is built in and loaded automatically. ISDS-205B / MDSO: import their firmware once from an OpenHantek6022 download. |
| Bench scopes (SCPI) | Rigol DS1000Z/DS2000/MSO5000/DHO; Keysight/Agilent InfiniiVision; Tektronix TBS/TDS; Siglent SDS1000X-E/2000X-E/1000X | USB or LAN | LAN needs no driver: type the IP address in the Connect window (VXI-11, or raw sockets on 5025/5555). USB works through NI-VISA / Keysight IO if installed, or directly over USBTMC with WinUSB. |
| OWON | HDS200/HDS300 handhelds, SDS/XDS/TAO bench scopes | USB or LAN | |
| Hantek handhelds | 2C42, 2C72, 2D42, 2D72 | USB | Experimental. |
| PicoScope | 2000 (2204A/2205A), 2000A/B, 3000A/B/D, 4000A, 5000A/B/D | USB | Needs Pico's free PicoSDK (we may not bundle it). |
| Old Hantek DSO | DSO-2090, 2150, 2250, 5200, 5200A | USB | Experimental. Needs Hantek's firmware, extracted from their driver with *Import firmware…*. |
| Demo mode | 2-channel USB scope, 4-channel bench scope | — | 13 simulated signals. |

**USB drivers on Windows:** *Device ▸ Install USB driver…* (or the button in the Connect window).
The Hantek 6022BE/BL get a one-step signed WinUSB driver; every other USB scope gets WinUSB with
Zadig, which is built in. Hantek 6022 and DSO-2xxx scopes get a new USB ID once their firmware is
loaded: after connecting once, run Zadig again for the new entry.

## Features

- **Acquisition:** Run/Stop, Single, Auto setup, Default setup, roll mode on streaming scopes.
- **Channels (up to 4):** volts/div, ×1/×10/×100 probe, DC/AC coupling, position, invert, clipping warning.
- **Trigger:** Auto/Normal, source, rising/falling/either edge, a level line you drag, 50 % button, holdoff and noise reject on streaming scopes.
- **Display:** lines or dots, sin(x)/x interpolation, averaging, colour-graded persistence (with beam-style edges), XY mode.
- **Analysis:** 13 automatic measurements plus phase and delay, time and voltage cursors, FFT with 5 windows (log or linear axis, zoom and pan), math channel (A+B, A−B, A×B).
- **Files:** CSV export and import (drop a CSV on the screen), PNG screenshots, save/load setups. Settings are remembered.
- **Speed:** an acquisition goroutine separate from the 60 fps display; with persistence or averaging every captured waveform is used. The status bar shows wfm/s next to fps.

Keyboard: **Space** run/stop · **S** single · **A** auto setup · **F1** guide · **Ctrl+K** connect ·
**Ctrl+O** open CSV · **Ctrl+S** export CSV · **Ctrl+P** screenshot

**Safety:** on USB and bench scopes the probe ground is connected to earth / your computer's
ground. Never connect it to mains or anything mains-powered without proper isolation.

## Building from source

Needs Go 1.22+ and nothing else (standard library only).

```
build.bat            (on Windows)
./build.sh           (on Linux/macOS — cross-compiles Scopie.exe)
go test ./...        (DSP checked against the original numpy code, engine, SCPI/VXI-11 with fake scopes)
```

The same source builds on Linux and macOS (`go build -o scopie .`) for porting: demo mode, network
bench scopes and CSV files work there today; USB needs a libusb/usbfs back end in `usb_other.go`.

## Source map

| File | Purpose |
|---|---|
| `main.go` | Local server, API, app window, single-instance hand-over, shutdown when the window closes |
| `engine.go` | Acquisition goroutine: ring buffer, software trigger, frames, persistence/averaging data |
| `view.go` | Display side: volts → screen, averaging, persistence images, measurements, FFT, auto setup, CSV |
| `dsp.go` | Triggers, measurements, FFT (any length), sin(x)/x interpolation |
| `settings.go` | Every front-panel setting, saved between sessions |
| `devices.go` | Driver interfaces and the scan registry |
| `dev_fx2.go` · `dev_hantekdso.go` · `dev_handheld.go` · `dev_owon.go` · `dev_scpi.go` · `dev_pico.go` · `dev_sim.go` | The drivers |
| `link.go` | Instrument links: VXI-11, raw TCP, USBTMC, USB bulk |
| `usb.go` / `usb_windows.go` | USB device list, WinUSB control/bulk/streaming, Intel-HEX and FX2 firmware upload |
| `visa_windows.go` · `pico_windows.go` | Optional NI-VISA and PicoSDK libraries, used when installed |
| `platform_windows.go` / `platform_other.go` | Window, folders, USB driver installers |
| `web/` | The interface (HTML/CSS/JS, embedded into the exe) |
| `firmware/` · `drivers/` | Built-in 6022 firmware, Zadig and the signed 6022 driver |
| `tools/mkres` | Writes the Windows icon + version resource |

## Licence

Free software under the GNU General Public License, version 3 or later — see `LICENSE` and
`THIRD-PARTY-NOTICES.md` (also in *Help ▸ Licences*). The built-in Zadig and firmware come with
their source in `third_party/`.

Made with care by **nrmluvdev**.

## Explain in Codie

- **Explain in Codie** (toolbar, or File menu): saves the waveform and opens it in Codie, which explains each channel in plain words.
