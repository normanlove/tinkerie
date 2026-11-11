> ⚠️ **Tinkerie is in alpha.** Everything works, but it's still being tested on real hardware, so expect rough edges and changes. Please [report](https://github.com/normanlove/tinkerie/issues) anything that doesn't work.

Free maker tools by **nrmluvdev**: Wirie, Makie, Tunie, Flashie, Scopie, Meterie, Probie and Codie in one program.

### Which file?
- **Windows 10/11:** `Tinkerie-{VERSION}-windows.exe` (just run it), or `Tinkerie-{VERSION}-windows.zip` if your connection blocks .exe downloads (it also has the source)
- **Mac:** `…-mac-apple-silicon.zip` (M1 or newer) or `…-mac-intel.zip`
- **Linux PC:** `…-linux-x86_64.tar.gz`
- **Raspberry Pi:** `…-pi-64bit.tar.gz` (64-bit Raspberry Pi OS) or `…-pi-32bit.tar.gz`

Each package has a *READ ME FIRST.txt*.

<!-- release notes for 0.19.0-alpha: package.sh checks this line, so update the notes with every version -->
### New in {VERSION}
- **21 colour themes in every app:** ten dark (Dark, Midnight, Ocean, Forest, Plum, Ember, Mocha, Slate, Graphite, True black), seven light (Light, Paper, Sepia, Sky, Mint, Rose, Lavender), and High contrast and Colour-blind safe, each in dark and light.
- **Accent colour:** keep the app's own, or pick one of eleven (blue, cyan, teal, green, lime, amber, orange, red, pink, purple, indigo) to replace its main colour; warning and status colours stay put.
- **Meterie** has all 21 themes on its Settings page.
- Text on bright buttons stays readable in the light and high-contrast themes.

### Also in recent alpha builds
- **0.18.0-alpha:** Look & accessibility in every app: a Look button (or Ctrl+,) with Dark, Light, High contrast and Colour-blind safe themes, text and controls size (90–150%), touch-friendly controls, reduced motion, an extra-strong focus outline, easier-to-read text and underlined links. Meterie has the themes and reduced motion on its Settings page. Touchscreens: on drawing areas (Wirie's board, Keebie, Scopie, Probie, Makie and more) drag with a finger, press and hold for the right-click menu, pinch to zoom and use two fingers to pan. Bigger buttons when touch is on. Tidier on smaller screens: Wirie's mode tabs become icons, and Makie's and Scopie's toolbars wrap instead of running off the edge; side panels narrow on tablets. Teaching mode says it's on the first time only, instead of every time an app opens. Icon-only buttons now have names for screen readers.
- **0.17.0-alpha:** Keebie screens: many more to choose from: 0.91in, 0.96in, 1.3in and 2.42in OLEDs, a 16×2 character LCD, the nice!view, and 1.3in, 1.54in and 1.9in colour TFTs, a 1.28in round TFT and a 1.8in TFT. SPI screens get pins of their own; Keebie warns when a screen and controller don't suit each other (3.3V screens on a 5V controller). Place parts on their own: the screen, status lights and reset button can leave the controller's block and be dragged anywhere on the keyboard; the board outline grows to reach them. Flashie makes firmware for every new screen (SSD1306/SSD1309, SH1106, Sharp memory LCD, ST7789, ST7735, GC9A01 and I2C character LCDs), redrawing only when something changes.
- **0.16.0-alpha:** Keebie: part spacing (tight, comfortable, roomy) so diode rows don't crowd; new diode patterns (zig-zag, wave, arch, chevron, slant, smile, alternating, herringbone, staggered); the "little map" layout was removed. Through-hole by default: 1N4148 diodes, 6×6mm reset button, SS12D00 power switch, 3mm LEDs and leaded resistors, picked from the most common parts. Power and activity lights, knobs and a screen (now also 1.3in) on any keyboard, with a one-click switch to a controller with enough pins. Standoffs (M2, M2.5 or M3) placed for you or by hand; controllers show which end their USB socket is; a parts list for the whole keyboard (every board) and for any PCB.
- **0.15.0-alpha:** Wirie: turn parts to any angle: type it in, drag the round handle above a part, or Shift+R for 15°. Auto-fix in the design check: redraws clashing tracks, moves parts that clash or sit off the board, widens thin tracks and routes the rest. Auto-fix mode does it every time the check runs.
- **0.14.0-alpha:** Mac and Linux catch up with Windows: Meterie's HID-cable meters, Bluetooth meters and Camera mode on Mac, Linux and Raspberry Pi; USB scopes, logic analysers and PicoScope on Mac; Camera mode on Android.
- **0.13.0-alpha:** Tinkerie for Android (5.0 and newer), with USB boards, scopes and analysers over USB OTG.
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
