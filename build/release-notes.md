> ⚠️ **Tinkerie is in alpha.** Everything works, but it's still being tested on real hardware, so expect rough edges and changes. Please [report](https://github.com/normanlove/tinkerie/issues) anything that doesn't work.

Free maker tools by **nrmluvdev**: Wirie, Makie, Tunie, Flashie, Scopie, Meterie and Tracie in one program.

### Which file?
- **Windows 10/11:** `Tinkerie.exe` (just run it), or `Tinkerie-{VERSION}-windows.zip` if your connection blocks .exe downloads (it also has the source)
- **Mac:** `…-mac-apple-silicon.zip` (M1 or newer) or `…-mac-intel.zip`
- **Linux PC:** `…-linux-x86_64.tar.gz`
- **Raspberry Pi:** `…-pi-64bit.tar.gz` (64-bit Raspberry Pi OS) or `…-pi-32bit.tar.gz`

Each package has a *READ ME FIRST.txt*.

### New in {VERSION}
- **Alpha:** every version number now ends in "-alpha" while Tinkerie is tested on real hardware.
- **Wirie:** the auto-router now draws ground (GND) tracks too, so every connection is visible on the board (the ground fill stays as extra ground). It also checks that the ground fill really reaches each pin, instead of assuming it does, and routes any pin the fill can't reach. A new *Route the fill net with tracks too* switch (on by default) is under Ground fill.

### Earlier
- 1.5.1: every app opens maximised.
- 1.5.0: Flashie's Write image tab gained OS settings for Raspberry Pi, Windows 10/11 installer USB drives, Linux ISOs and .zst images.

Licensed under GPL-3.0-or-later.
