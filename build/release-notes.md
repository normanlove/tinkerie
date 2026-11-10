> ⚠️ **Tinkerie is in alpha.** Everything works, but it's still being tested on real hardware, so expect rough edges and changes. Please [report](https://github.com/normanlove/tinkerie/issues) anything that doesn't work.

Free maker tools by **nrmluvdev**: Wirie, Makie, Tunie, Flashie, Scopie, Meterie, Probie and Codie in one program.

### Which file?
- **Windows 10/11:** `Tinkerie-{VERSION}-windows.exe` (just run it), or `Tinkerie-{VERSION}-windows.zip` if your connection blocks .exe downloads (it also has the source)
- **Mac:** `…-mac-apple-silicon.zip` (M1 or newer) or `…-mac-intel.zip`
- **Linux PC:** `…-linux-x86_64.tar.gz`
- **Raspberry Pi:** `…-pi-64bit.tar.gz` (64-bit Raspberry Pi OS) or `…-pi-32bit.tar.gz`

Each package has a *READ ME FIRST.txt*.

### New in {VERSION}
- **Keebie: switch pins and LED facing.** Choose 5-pin holes (fit 5-pin and 3-pin MX switches) or 3-pin only for plate builds, and south- or north-facing LEDs.
- **Keebie shows its diodes and LEDs.** They were always added, but on the back of the board where they were easy to miss. A new **Parts** view shows every diode, LED and capacitor on the keyboard before you build, and the build message says how many went on and links straight to the back of the board.
- **Diode layouts:** upright beside each switch (right or left), flat below or above it, or at 45° in the corner, on the back or on show at the front. Every spot is checked against the switch's pads, holes and LED. Optional 100nF capacitor beside each LED.
- **Move and turn the controller:** drag it, nudge it with the arrow keys, turn it with R or the 90° buttons; the board outline follows. Its support parts can be packed tightly, set out in neat rows, or put in one tidy line.
- **New keyboard and Close keyboard** buttons in Keebie.

### Also in recent alpha builds
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
