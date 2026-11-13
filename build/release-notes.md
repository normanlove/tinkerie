> ⚠️ **Tinkerie is in alpha.** Everything works, but it's still being tested on real hardware, so expect rough edges and changes. Please [report](https://github.com/normanlove/tinkerie/issues) anything that doesn't work.

Free maker tools by **nrmluvdev**: Wirie, Makie, Tunie, Flashie, Scopie, Meterie, Probie, Codie and Fixie in one program.

### Which file?
- **Windows 10/11:** `Tinkerie-{VERSION}-windows.exe` (just run it), or `Tinkerie-{VERSION}-windows.zip` if your connection blocks .exe downloads (it also has the source)
- **Mac:** `…-mac-apple-silicon.zip` (M1 or newer) or `…-mac-intel.zip`
- **Linux PC:** `…-linux-x86_64.tar.gz`
- **Raspberry Pi:** `…-pi-64bit.tar.gz` (64-bit Raspberry Pi OS) or `…-pi-32bit.tar.gz`

Each package has a *READ ME FIRST.txt*.

<!-- release notes for 0.28.1-alpha: package.sh checks this line, so update the notes with every version -->
### New in {VERSION}
- **Fixie's steps fit the device far more closely.** Every step and fix was checked against every kind of device that can reach it, and about 30 mismatches fixed. 3D printers no longer get the hobby-board upload steps (they get stepper motor, driver, screen-cable and heater checks instead); "a programmable board" now means a hobby board you program yourself. Cordless tools get battery pack and charger checks instead of USB ones. PC fans no longer get carbon-brush or start-capacitor advice, desktop PCs aren't sent into their sealed power supply, and boards without a battery aren't told to wait for it to charge. Fixes now only show the lines that apply to your device, and Fixie asks how it's powered before going on.
- **New tests**: no step or fix can ever be empty for a device, and a list of steps each kind of device must never be shown is checked on every build.

Licensed under GPL-3.0-or-later.

Windows code signing is being set up (until then Windows may warn about an unrecognised app): free code signing provided by [SignPath.io](https://signpath.io), certificate by [SignPath Foundation](https://signpath.org). See the [code signing policy](https://tinkerie.net/code-signing.html) and [privacy statement](https://tinkerie.net/code-signing.html#privacy).
