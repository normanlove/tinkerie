# Tinkerie

A free suite of tools for DIY hobbyists and makers, in one program. Designed and created by **nrmluvdev**.

Double-click `Tinkerie.exe` to open the launcher and pick an app. Each app opens in its own window and process, so you can run several at once. They keep running after you close the launcher.

| App | What it does | Version |
|---|---|---|
| **Wirie** | Beginner-friendly circuit board designer (Boardie stripboard, Calcie calculators, Pinnie, Powerie) | 1.2.0 |
| **Makie** | 3D-printing part designer. Builds cases for Wirie boards | 1.5.1 |
| **Tunie** | 3D printer tuning and calibration. Shares fit clearances with Makie | 1.0.0 |
| **Flashie** | Firmware flasher for microcontroller boards (ESP32, Pico, Arduino, STM32), plus a "Write image" tab that puts Raspberry Pi OS (with your Wi-Fi, user and SSH set up), Linux, or a Windows 10/11 installer onto an SD card or USB drive | 0.7.0 |
| **Scopie** | Oscilloscope program (Hantek 6022BE/BL and others, plus network scopes and demo) | 3.0.0 |
| **Meterie** | PC multimeter: HoldPeak HP-90EPC, UNI-T, OWON, Brymen, Fluke and more | 1.1.0 |
| **Tracie** | Serial monitor and logic analyser | 1.0.0 |

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
