> ⚠️ **Tinkerie is in alpha.** Everything works, but it's still being tested on real hardware, so expect rough edges and changes. Please [report](https://github.com/normanlove/tinkerie/issues) anything that doesn't work.

Free maker tools by **nrmluvdev**: Wirie, Makie, Tunie, Flashie, Scopie, Meterie, Probie and Codie in one program.

### Which file?
- **Windows 10/11:** `Tinkerie-{VERSION}-windows.exe` (just run it), or `Tinkerie-{VERSION}-windows.zip` if your connection blocks .exe downloads (it also has the source)
- **Mac:** `…-mac-apple-silicon.zip` (M1 or newer) or `…-mac-intel.zip`
- **Linux PC:** `…-linux-x86_64.tar.gz`
- **Raspberry Pi:** `…-pi-64bit.tar.gz` (64-bit Raspberry Pi OS) or `…-pi-32bit.tar.gz`

Each package has a *READ ME FIRST.txt*.

<!-- release notes for 0.26.0-alpha: package.sh checks this line, so update the notes with every version -->
### New in {VERSION}
- **Wirie: castellated modules soldered flat.** Boards with half-holes along their edges (Raspberry Pi Pico, Pico W, Pico 2 and 2 W, Waveshare RP2040-Zero, Seeed XIAO, ESP32-C3 SuperMini) now have a surface-mount version: select the part and choose *Mounting: Soldered flat (castellated)*. Its pins become edge pads, it sits low with no headers, and every connection stays as it was (re-route to redraw the tracks). Keebie has the same option for the controller, and the parts list, Pinnie, Powerie and Makie's case height all follow it.

Licensed under GPL-3.0-or-later.

Windows code signing is being set up (until then Windows may warn about an unrecognised app): free code signing provided by [SignPath.io](https://signpath.io), certificate by [SignPath Foundation](https://signpath.org). See the [code signing policy](https://tinkerie.net/code-signing.html) and [privacy statement](https://tinkerie.net/code-signing.html#privacy).
