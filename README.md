# Tinkerie

A free suite of tools for DIY hobbyists and makers, in one program. Designed and created by **nrmluvdev**.

> **Tinkerie is in alpha.** Everything works, but it's still being tested on real hardware, so expect rough edges and changes between versions. Please [report anything odd](https://github.com/normanlove/tinkerie/issues).

Design stripboard circuits, model 3D-printed parts, flash microcontrollers, write SD cards and USB installers, and use your PC as an oscilloscope, multimeter, serial monitor or logic analyser — all from one small program that needs nothing installed. Runs on Windows, Mac, Linux and Raspberry Pi.

## Download

**Website:** https://normanlove.github.io/tinkerie/ (screenshots and a guide for every app)

Get the newest version from the **[Releases page](https://github.com/normanlove/tinkerie/releases)** (alpha versions are marked "Pre-release"):

| Your computer | Download |
|---|---|
| Windows 10 or 11 (64-bit) | `Tinkerie.exe` on its own, or `Tinkerie-…-windows.zip` (with the source) |
| Mac with Apple silicon (M1 or newer) | `Tinkerie-…-mac-apple-silicon.zip` |
| Mac with an Intel processor | `Tinkerie-…-mac-intel.zip` |
| Linux PC (64-bit Intel/AMD) | `Tinkerie-…-linux-x86_64.tar.gz` |
| Raspberry Pi OS 64-bit (Pi 3, 4, 5) | `Tinkerie-…-pi-64bit.tar.gz` |
| Raspberry Pi OS 32-bit (Pi 2, 3, 4, 5, Zero 2 W) | `Tinkerie-…-pi-32bit.tar.gz` |

Each package has a **READ ME FIRST.txt** that explains how to start it and what works on that system. Windows may warn about an unrecognised app the first time: choose *More info* then *Run anyway*. Macs need the one-time *Open Anyway* step in System Settings ▸ Privacy & Security.

## Using it

Double-click `Tinkerie.exe` to open the launcher and pick an app. Each app opens in its own window and process, so you can run several at once. They keep running after you close the launcher.

| App | What it does | Version |
|---|---|---|
| **Wirie** | Beginner-friendly circuit board designer (Boardie stripboard, Calcie calculators, Pinnie, Powerie) | 1.2.1-alpha |
| **Makie** | 3D-printing part designer. Builds cases for Wirie boards | 1.5.1-alpha |
| **Tunie** | 3D printer tuning and calibration. Shares fit clearances with Makie | 1.0.0-alpha |
| **Flashie** | Firmware flasher for microcontroller boards (ESP32, Pico, Arduino, STM32), plus a "Write image" tab that puts Raspberry Pi OS (with your Wi-Fi, user and SSH set up), Linux, or a Windows 10/11 installer onto an SD card or USB drive | 0.7.0-alpha |
| **Scopie** | Oscilloscope program (Hantek 6022BE/BL and others, plus network scopes and demo) | 3.0.0-alpha |
| **Meterie** | PC multimeter: HoldPeak HP-90EPC, UNI-T, OWON, Brymen, Fluke and more | 1.1.0-alpha |
| **Tracie** | Serial monitor and logic analyser | 1.0.0-alpha |

Every app's own guide is its `README.md` in `apps\<name>\`.

## Teaching mode

Every app explains itself. Point at any button, setting or panel and a card shows what it does, in plain language. Press **F2** to turn the explanations off or back on. Each app remembers your choice, and Meterie also has the switch in Settings. Scopie has its own help bar under the scope screen, which is always on.

## Starting an app directly

```
Tinkerie.exe --app makie              start Makie without the launcher
Tinkerie.exe --app makie part.makie   app arguments go after the name
```

A copy of `Tinkerie.exe` renamed to an app's name (`Makie.exe`, `Scopie.exe`…) starts that app directly. That's handy for desktop shortcuts, and on Linux/macOS a symlink named `makie` does the same.

Makie adds itself to Windows "Open with" for `.makie` projects, Wirie board files (`.makie.json`), STL and OBJ, as `Tinkerie.exe --app makie "file"`.

## Building

Go 1.24 or newer, standard library only, no cgo. From this folder:

```
build.bat             Windows: builds Tinkerie.exe
./build.sh            builds for the computer you're on (Mac, Linux or Raspberry Pi)
./build.sh pi         or name one: windows, mac, mac-intel, linux, pi, pi32, all
go test ./...         runs every app's tests (Makie's geometry tests take about 12 minutes)
```

Mac builds come out as `dist/<name>/Tinkerie.app`, Linux and Pi builds as `dist/<name>/tinkerie` with an `install.sh` that adds Tinkerie to the applications menu.

## Platforms

| | Windows | Linux, Raspberry Pi | Mac |
|---|---|---|---|
| Launcher, Wirie, Makie, Tunie, Flashie | everything | everything | everything |
| Scopie | everything | everything (USB scopes: run `install-usb.sh` once) | no USB scopes yet |
| Tracie | everything | everything (logic analyser: run `install-usb.sh` once) | serial monitor; no logic analyser yet |
| Meterie | everything | USB-serial meters (like the HoldPeak HP-90EPC) | USB-serial meters |

Meterie's HID-cable meters, Bluetooth meters and Camera mode are Windows-only for now. On a Mac, USB scopes and logic analysers need Apple's IOKit, which these pure-Go builds don't use.

For other platforms and how the source is laid out, see [PORTING.md](PORTING.md).

## Licence

Tinkerie is free software: you can use, share and change it under the terms of the **GNU General Public License, version 3 or later** ([LICENSE](LICENSE)). If you share a changed version, you must share its source code under the same licence. It comes with no warranty.

Copyright © 2026 nrmluvdev.

Parts made by others keep their own licences, listed with them: Go's zstd decoder (`shared/zstd/LICENSE`, BSD), the fonts in Makie and Meterie (`apps/makie/LICENSE-fonts.txt`, `apps/meterie/web/fonts/`), and Scopie's firmware, Zadig and adapted protocol code (`apps/scopie/THIRD-PARTY-NOTICES.md`).

## Reporting problems

Found a bug, or a board, scope or meter that doesn't work? Open an [issue](https://github.com/normanlove/tinkerie/issues) and say which app, which system, and what happened.
