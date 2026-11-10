> ⚠️ **Tinkerie is in alpha.** Everything works, but it's still being tested on real hardware, so expect rough edges and changes. Please [report](https://github.com/normanlove/tinkerie/issues) anything that doesn't work.

Free maker tools by **nrmluvdev**: Wirie, Makie, Tunie, Flashie, Scopie, Meterie, Tracie and Codie in one program.

### Which file?
- **Windows 10/11:** `Tinkerie-{VERSION}-windows.exe` (just run it), or `Tinkerie-{VERSION}-windows.zip` if your connection blocks .exe downloads (it also has the source)
- **Mac:** `…-mac-apple-silicon.zip` (M1 or newer) or `…-mac-intel.zip`
- **Linux PC:** `…-linux-x86_64.tar.gz`
- **Raspberry Pi:** `…-pi-64bit.tar.gz` (64-bit Raspberry Pi OS) or `…-pi-32bit.tar.gz`

Each package has a *READ ME FIRST.txt*.

### New in {VERSION}
- **Downloads now carry their version number**, including the Windows program: `Tinkerie-{VERSION}-windows.exe`.
- **Flashie makes keyboard firmware (new Keyboard tab).** Send a keyboard from Wirie's Keebie tab, choose what every key does (letters, shortcuts, media keys, text, up to 4 layers), what each knob does when turned and pressed, and what the screen and lights show. Flashie writes the firmware, compiles it with arduino-cli and flashes it to ATmega32U4 (Pro Micro), RP2040 and STM32 keyboards.
- **Makie makes keyboard cases.** Press Case in Makie in Keebie: Makie builds an open-top case that follows your keyboard's outline, a printed switch plate with a cut-out for every switch, knob and stabiliser, and feet for a typing angle.
- **Keebie:** keys can now be rotary knobs (EC11), and a small OLED screen can be added. The wave option was removed (it didn't handle spacebars well).

### Also in recent alpha builds
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
