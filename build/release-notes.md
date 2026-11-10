> ⚠️ **Tinkerie is in alpha.** Everything works, but it's still being tested on real hardware, so expect rough edges and changes. Please [report](https://github.com/normanlove/tinkerie/issues) anything that doesn't work.

Free maker tools by **nrmluvdev**: Wirie, Makie, Tunie, Flashie, Scopie, Meterie, Tracie and Codie in one program.

### Which file?
- **Windows 10/11:** `Tinkerie.exe` (just run it), or `Tinkerie-{VERSION}-windows.zip` if your connection blocks .exe downloads (it also has the source)
- **Mac:** `…-mac-apple-silicon.zip` (M1 or newer) or `…-mac-intel.zip`
- **Linux PC:** `…-linux-x86_64.tar.gz`
- **Raspberry Pi:** `…-pi-64bit.tar.gz` (64-bit Raspberry Pi OS) or `…-pi-32bit.tar.gz`

Each package has a *READ ME FIRST.txt*.

### New in {VERSION}
- **New in Wirie: Keebie, custom keyboard PCBs.** Lay out your keys (presets from macropads to full size, grids, KLE import, every key size, ISO Enter, any angle), shape the board (hug the keys, rectangle or your own corners, wave rows, arcs, Alice-style angled splits or two halves with TRRS) and press Build: Keebie places the switches, diodes, matrix, stabilisers, per-key RGB LEDs and mounting holes on the PCB.
- **Every common keyboard controller:** Pro Micro, Elite-C, RP2040 Pro Micro boards, Pico, RP2040-Zero, XIAO RP2040, Black Pill, nice!nano and XIAO nRF52840, or bare ATmega32U4, RP2040 and STM32 chips with all their support circuitry (USB-C, fuse, crystal, regulator, flash, reset and boot buttons) added for you.
- **Switches and caps:** MX and Choc, soldered or hot-swap, and Alps; keycap profiles; switch footprints can be turned per key for custom-height builds.
- **Firmware and plate:** Keebie writes QMK (or ZMK for wireless) files with your matrix pins, and the switch plate as SVG or DXF.
- Wirie now supports custom board outlines, cut-outs and parts at any angle.

### Also in recent alpha builds
- **0.7.0-alpha, new app: Codie, the code explainer.** Paste code (or open a file) and Codie explains it line by line, in 20+ languages including Windows Go and 8-bit BASIC, with a Game logic tab and a Game school.
- 0.6.1-alpha: the version in the Windows file details now matches the alpha numbering.
- **0.6.0-alpha, new version numbers:** while Tinkerie is in alpha, it and every app are numbered below 1.0. Apps: Makie and Scopie 0.9, Wirie 0.8, Flashie 0.7.1, Meterie 0.6, Tunie and Tracie 0.5.
- **Flashie:** ESP32 Marauder and Bruce were removed from the firmware library.
- Wirie's auto-router draws ground (GND) tracks too, and checks that the ground fill really reaches each pin.
- Every app opens maximised.
- Flashie's Write image tab: OS settings for Raspberry Pi, Windows 10/11 installer USB drives, Linux ISOs and .zst images.

Licensed under GPL-3.0-or-later.

Windows code signing is being set up (until then Windows may warn about an unrecognised app): free code signing provided by [SignPath.io](https://signpath.io), certificate by [SignPath Foundation](https://signpath.org). See the [code signing policy](https://tinkerie.net/code-signing.html) and [privacy statement](https://tinkerie.net/code-signing.html#privacy).
