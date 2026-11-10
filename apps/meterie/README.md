# Meterie — PC multimeter app

**Made by nrmluvdev** · part of the free hobbyist tool suite (alongside Wirie, Scopie, Makie, Flashie, Tunie …)

Meterie turns your PC into a big, friendly multimeter display: a premium bench-meter style
readout, live graphs, logging, alarms, math channels and guided tests — for the meter you
already own. It is a single Windows `.exe` (no installer, no DLLs, no runtimes), written in
pure Go using only Windows' own system libraries.

---

## Quick start

1. Run `Meterie.exe`. (Windows SmartScreen may warn about an unsigned app — *More info → Run anyway*.)
2. Plug in your meter's USB / IR cable, switch the meter on and enable its PC link
   (usually a long press on a button marked USB, RS232 or PC until the symbol shows on the LCD).
3. Open **Meters → Auto-detect → Scan**, then **Detect & connect**.
4. No hardware? **Meters → Demo & manual → Start demo** lets you try every feature.

### HoldPeak HP-90EPC

Fully supported (Fortune FS9721 protocol, 2400 baud). Connect the supplied USB cable — if
Windows doesn't create a COM port, install HoldPeak's USB driver — turn the meter on and press
its USB/PC button until the USB/RS232 icon appears. Then either pick *HoldPeak HP-90EPC*
under **Meters → Choose model**, or use Auto-detect and set *Exact model → HoldPeak HP-90EPC*
on the meter's card (auto-detect only knows "an FS9721 meter"). Volts, amps, ohms, continuity,
diode, capacitance, frequency, duty cycle and temperature (°C, per sigrok's HP-90EPC entry) are
all decoded, plus HOLD, REL, AUTO, AC/DC and low-battery.

---

## Supported meters

Every USB/RS-232 protocol below was checked against the decoder source of the
[sigrok](https://sigrok.org) project (or the community reference for Bluetooth meters) and has
unit tests with real or byte-exact frames.

| Status | Meters |
|---|---|
| **Protocol verified** | **HoldPeak HP-90EPC** · TekPower TP4000ZC · Digitek DT4000ZC · Tenma 72-7745 · UNI-T UT60A/UT60E · V&A VA18B · Mastech MS8250B · PeakTech 3330/3415 · Voltcraft VC-820/VC-840 · any FS9721/FS9711 4000-count meter |
| **Protocol verified** | UNI-T UT61B / UT61C / UT61D (UT-D04 USB or RS-232) · Voltcraft VC-830 · any FS9922 6000-count meter |
| **Protocol verified** | UNI-T UT61E (ES51922) · UNI-T UT60G / Tenma 72-7750 · ISO-TECH IDM103N (ES519xx 11-byte) |
| **Protocol verified** | UNI-T UT61E+ / UT61B+ / UT61D+ / UT161B/D/E and **UNI-T UT181A** (CP2110 USB — auto-detect tells them apart) |
| **Protocol verified** | **UNI-T UT71A/B/C/D/E**, UT804 · Voltcraft VC-920/940/960 · Tenma 72-7730/72-7732/72-9380A |
| **Protocol verified** | **Brymen BM869s / BM867s / BM859s** (BU-86X USB cable, main display) · **Brymen BM25x** (BM257s …, IR cable) |
| **Protocol verified** | Victor 70C / 86C / 86D USB (de-scrambled exactly like sigrok) |
| **Protocol verified** | Fluke 287 / 289 (IR cable, 115200) · Fluke 187 / 189 · Fluke 87-IV / 89-IV |
| **Protocol verified** | Mastech MAS-345, Metex M-3850, Voltcraft ME/M-36xx (14-byte ASCII, 600/1200 baud) |
| **Community protocol** | OWON Bluetooth: B35T+, B41T+, OW18E, OW16 (6-byte BLE protocol) and the original B35T |
| **Community protocol** | **ANENG / ZOYI / BSIDE Bluetooth** ("Bluetooth DMM"): AN9002, V05B, ST207, AN999S, ZT-300AB, ZT-5B, ZT-5BQ, ZT-5566S/SE, BABATools AD900 |
| **Experimental** | OWON XDM1041/1241/2041 bench meters (SCPI) |
| **Any meter** | **Camera mode** reads 7-segment displays with a webcam, a phone (IP camera app) or a photo |
| **Any meter** | **Manual entry** for logging readings by hand |

*Not supported over USB/Bluetooth yet* (use Camera mode): UNI-T UT171 series (packet layout
differs from the UT181A and couldn't be verified) and Brymen BM235 / BM23x (no documented PC
protocol — if you own Brymen's IR cable, try the *Brymen BM25x* entry).

**Auto-detect picks a protocol family; pick your exact model afterwards.** Many meters share a
chip (e.g. the HP-90EPC, TP4000ZC and UT60E all speak FS9721). The meter's card on the
**Meters** page has an *Exact model* menu — choose yours so model-specific details such as the
temperature symbol decode correctly. Meterie remembers it for next time.

Generic BLE-to-serial meter bridges can be used with **Bluetooth → Protocol → BLE-serial FS9721 / FS9922**.

---

## Features

**Live** — big seven-segment readout with ghost segments and glow, mirrored annunciators
(AUTO/MANUAL, AC/DC, HOLD, REL Δ, MIN/MAX, PEAK, HV ⚡, beep, diode, low battery) and an
animated bar graph. Statistics (min, max, average, peak-to-peak, std dev, sample count, charge
in mAh for current, energy in Wh for power) over *what's visible*, a *Shift+drag selection*,
the *last 10 s* or *everything*. **REL Δ** zeroes a channel (e.g. test-lead resistance).

**Graph** — every meter, math and camera channel on one shared timeline with its own colour;
two labelled Y axes when units differ.

| Action | How |
|---|---|
| Zoom time | mouse wheel |
| Zoom Y | Ctrl + wheel |
| Pan | drag |
| Select a range for statistics / export | Shift + drag |
| Add a marker / annotation | right-click the graph, Ctrl+M, or the Marker button |
| Back to live view | double-click, or the fit button |
| Pause / resume | Space |

**Logging** — CSV with timestamps; every reading or a fixed interval (0.1 s … 5 min);
start immediately or when a channel drops below / rises above a value; stop manually, after a
duration, after N rows, or on a threshold. Excel-friendly columns per channel.

**Alarms** — high/low limits per channel with sound and a flashing readout; **continuity beep
on the PC** (threshold adjustable).

**Math channels** — Power (V×I), Resistance (V÷I), Energy (∫V·I dt in Wh), Charge (∫I dt in mAh),
difference, sum, ratio, or a custom formula like `(ch1 - ch2) / 0.1`.

**Wizards** — battery capacity / discharge test (with cut-off), battery internal resistance,
resistor & capacitor sorter (E6/E12/E24/E96, tolerance bins, CSV export), LED forward voltage
with colour guess and series-resistor calculator, solar panel / power-supply load test with
I–V and power curves (maximum power point, fill factor, load regulation).

**Camera mode** — webcam (Media Foundation), phone/IP camera (JPEG snapshot or MJPEG URL, e.g.
the *IP Webcam* Android app's `http://phone:8080/shot.jpg`), or a photo. Drag the box around
the digits, choose LCD/LED, set the digit count — or type what the display shows and press
**Auto-fit** to calibrate slant, thickness and threshold automatically. Readings are only
accepted when two frames agree.

**What am I measuring?** — a beginner guide for every mode (DC/AC volts, amps, ohms,
continuity, diode, capacitance, frequency, temperature) showing which jacks to use and how to
connect, with mains/high-voltage safety and CAT ratings explained.

**Export & sessions** — CSV, suite timeline (`.nst.csv`), 1920×1080 PNG of the graph, and
`.meterie` sessions that reopen with all channels, markers, math and alarms. Double-clicking a
`.meterie` file (once associated) or passing it on the command line opens it.

Keyboard: **Ctrl+S** save session · **Ctrl+O** open/import · **Ctrl+M** marker · **Space** pause
graph · **F1** beginner guide.

---

## Suite integration

* **Wirie / Calcie** can pull a live reading from `http://127.0.0.1:47823/api/v1/reading`
  (or read `%APPDATA%\nrmluvdev\suite\meterie-live.json`). The *Copy* button on the Live page
  also puts the value on the clipboard.
* **Scopie, Serialie / Logicie**: Meterie exports and imports the shared **nrmluv-timeline**
  CSV format with absolute nanosecond timestamps, so captures line up on one timeline.

Full details: [`SUITE_FORMAT.md`](SUITE_FORMAT.md).

---

## Files & settings

* Settings: `%APPDATA%\nrmluvdev\Meterie\settings.json`
* Logs & exports default to `Documents\Meterie` (changeable in Settings)
* Crash reports (if any): `%APPDATA%\nrmluvdev\Meterie\crash-*.txt`

## Building from source

See [`BUILD.md`](BUILD.md). Short version (Go 1.22+):

```
build.bat            (on Windows)
./build.sh           (on Linux/macOS, cross-compiles the Windows exe)
```

## Safety

Meterie displays what your meter measures — it adds no protection. Never measure mains or
high voltage unless you are trained and your meter and leads carry an adequate CAT rating.

## Credits

Meterie by **nrmluvdev**. Protocol knowledge thanks to the sigrok project's multimeter notes,
the OWON / UNI-T / Aneng reverse-engineering community (DeanCording, ljakob, mwuertinger,
ludwich66 and others) and Fluke's remote interface specification.

## Explain in Codie

- **Explain in Codie** (Export): saves the readings as a suite timeline and opens them in Codie, which explains batteries, current patterns, power and energy in plain words.
