> ⚠️ **Tinkerie is in alpha.** Everything works, but it's still being tested on real hardware, so expect rough edges and changes. Please [report](https://github.com/normanlove/tinkerie/issues) anything that doesn't work.

Free maker tools by **nrmluvdev**: Wirie, Makie, Tunie, Flashie, Scopie, Meterie, Probie, Codie and Fixie in one program.

### Which file?
- **Windows 10/11:** `Tinkerie-{VERSION}-windows.exe` (just run it), or `Tinkerie-{VERSION}-windows.zip` if your connection blocks .exe downloads (it also has the source)
- **Mac:** `…-mac-apple-silicon.zip` (M1 or newer) or `…-mac-intel.zip`
- **Linux PC:** `…-linux-x86_64.tar.gz`
- **Raspberry Pi:** `…-pi-64bit.tar.gz` (64-bit Raspberry Pi OS) or `…-pi-32bit.tar.gz`

Each package has a *READ ME FIRST.txt*.

<!-- release notes for 0.30.1-alpha: package.sh checks this line, so update the notes with every version -->
### New in {VERSION}
- **Fixie checked against every device you can describe.** All three guides were walked for every kind of device with every combination of power and features (952,320 descriptions), checking that every path ends at a fix, no step is empty, and no word or picture mentions something the device doesn't have. Over 100 mismatches were fixed: steps now fit builds without a hobby board, battery checks only appear for battery devices, motor and flyback advice only where there's a motor, and the audio, video and keyboard sections assume the device has them.
- **Pictures match their steps.** Each drawing is checked against what its step says; a few that didn't fit were moved or removed (a TVS diode no longer shows a capacitor).

Licensed under GPL-3.0-or-later.

Windows code signing is being set up (until then Windows may warn about an unrecognised app): free code signing provided by [SignPath.io](https://signpath.io), certificate by [SignPath Foundation](https://signpath.org). See the [code signing policy](https://tinkerie.net/code-signing.html) and [privacy statement](https://tinkerie.net/code-signing.html#privacy).
