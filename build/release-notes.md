> ⚠️ **Tinkerie is in alpha.** Everything works, but it's still being tested on real hardware, so expect rough edges and changes. Please [report](https://github.com/normanlove/tinkerie/issues) anything that doesn't work.

Free maker tools by **nrmluvdev**: Wirie, Makie, Tunie, Flashie, Scopie, Meterie, Probie and Codie in one program.

### Which file?
- **Windows 10/11:** `Tinkerie-{VERSION}-windows.exe` (just run it), or `Tinkerie-{VERSION}-windows.zip` if your connection blocks .exe downloads (it also has the source)
- **Mac:** `…-mac-apple-silicon.zip` (M1 or newer) or `…-mac-intel.zip`
- **Linux PC:** `…-linux-x86_64.tar.gz`
- **Raspberry Pi:** `…-pi-64bit.tar.gz` (64-bit Raspberry Pi OS) or `…-pi-32bit.tar.gz`

Each package has a *READ ME FIRST.txt*.

### New in {VERSION}
- **Tinkerie for Android** (Android 5.0 and newer: phones, tablets and Chromebooks). The same apps, each in its own window. Plug boards, scopes and analysers in with a USB OTG adapter: Flashie flashes boards (including UF2 boards like the Pico, and SD cards in a USB reader), Probie and Scopie capture, and the serial apps work with Arduino and other native-USB boards, CH340, CP210x, FTDI and PL2303 adapters. Files the apps save appear in the Files app under *Tinkerie*. Compiling Arduino code still needs a computer.

### Also in recent alpha builds
- **0.12.x-alpha:** Keebie diode layouts (including on show, off the keys), a Parts view, a movable controller, switch pins and LED facing.
- **0.11.0-alpha:** Tracie became Probie, with an Explain tab that explains captures in plain words; Codie went back to explaining code.
- **0.10.0-alpha:** Codie explained captures from Scopie, Tracie and Meterie (that's now Probie's Explain tab).
- **0.9.1-alpha:** Keebie fixes: key spacing, widening keys, much faster redraws.
- **0.9.0-alpha:** Flashie's Keyboard tab makes and flashes firmware for Keebie keyboards; Makie makes keyboard cases with a printed switch plate; Keebie knobs and OLED screens; version numbers in every download's name.
- **0.8.0-alpha, new in Wirie: Keebie, custom keyboard PCBs.** Lay out the keys, pick switches and any common controller (or a bare chip with all its circuitry), and Keebie builds the PCB, QMK/ZMK files and a switch plate.
- **0.7.0-alpha, new app: Codie, the code explainer**, in 20+ languages, with a Game logic tab and a Game school.
- 0.6.1-alpha: the version in the Windows file details now matches the alpha numbering.
- **0.6.0-alpha, new version numbers:** while Tinkerie is in alpha, it and every app are numbered below 1.0. Apps: Makie and Scopie 0.9, Wirie 0.8, Flashie 0.7.1, Meterie 0.6, Tunie and Tracie 0.5.
- **Flashie:** ESP32 Marauder and Bruce were removed from the firmware library.
- Wirie's auto-router draws ground (GND) tracks too, and checks that the ground fill really reaches each pin.
- Every app opens maximised.
- Flashie's Write image tab: OS settings for Raspberry Pi, Windows 10/11 installer USB drives, Linux ISOs and .zst images.

Licensed under GPL-3.0-or-later.

Windows code signing is being set up (until then Windows may warn about an unrecognised app): free code signing provided by [SignPath.io](https://signpath.io), certificate by [SignPath Foundation](https://signpath.org). See the [code signing policy](https://tinkerie.net/code-signing.html) and [privacy statement](https://tinkerie.net/code-signing.html#privacy).
