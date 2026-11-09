> ⚠️ **Tinkerie is in alpha.** Everything works, but it's still being tested on real hardware, so expect rough edges and changes. Please [report](https://github.com/normanlove/tinkerie/issues) anything that doesn't work.

Free maker tools by **nrmluvdev**: Wirie, Makie, Tunie, Flashie, Scopie, Meterie and Tracie in one program.

### Which file?
- **Windows 10/11:** `Tinkerie.exe` (just run it), or `Tinkerie-{VERSION}-windows.zip` if your connection blocks .exe downloads (it also has the source)
- **Mac:** `…-mac-apple-silicon.zip` (M1 or newer) or `…-mac-intel.zip`
- **Linux PC:** `…-linux-x86_64.tar.gz`
- **Raspberry Pi:** `…-pi-64bit.tar.gz` (64-bit Raspberry Pi OS) or `…-pi-32bit.tar.gz`

Each package has a *READ ME FIRST.txt*.

### New in {VERSION}
- **Tidy-up release:** the version in the Windows file details (Properties › Details) and in a few examples and notes now matches the pre-1.0 alpha numbering. No changes to how the apps work.

### Also in recent alpha builds
- **0.6.0-alpha, new version numbers:** while Tinkerie is in alpha, it and every app are numbered below 1.0. Apps: Makie and Scopie 0.9, Wirie 0.8, Flashie 0.7.1, Meterie 0.6, Tunie and Tracie 0.5.
- **Flashie:** ESP32 Marauder and Bruce were removed from the firmware library.
- Wirie's auto-router draws ground (GND) tracks too, and checks that the ground fill really reaches each pin.
- Every app opens maximised.
- Flashie's Write image tab: OS settings for Raspberry Pi, Windows 10/11 installer USB drives, Linux ISOs and .zst images.

Licensed under GPL-3.0-or-later.

Windows code signing is being set up (until then Windows may warn about an unrecognised app): free code signing provided by [SignPath.io](https://signpath.io), certificate by [SignPath Foundation](https://signpath.org). See the [code signing policy](https://tinkerie.net/code-signing.html) and [privacy statement](https://tinkerie.net/code-signing.html#privacy).
