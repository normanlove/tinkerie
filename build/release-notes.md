> ⚠️ **Tinkerie is in alpha.** Everything works, but it's still being tested on real hardware, so expect rough edges and changes. Please [report](https://github.com/normanlove/tinkerie/issues) anything that doesn't work.

Free maker tools by **nrmluvdev**: Wirie, Makie, Tunie, Flashie, Scopie, Meterie, Probie, Codie and Fixie in one program.

### Which file?
- **Windows 10/11:** `Tinkerie-{VERSION}-windows.exe` (just run it), or `Tinkerie-{VERSION}-windows.zip` if your connection blocks .exe downloads (it also has the source)
- **Mac:** `…-mac-apple-silicon.zip` (M1 or newer) or `…-mac-intel.zip`
- **Linux PC:** `…-linux-x86_64.tar.gz`
- **Raspberry Pi:** `…-pi-64bit.tar.gz` (64-bit Raspberry Pi OS) or `…-pi-32bit.tar.gz`

Each package has a *READ ME FIRST.txt*.

<!-- release notes for 0.30.3-alpha: package.sh checks this line, so update the notes with every version -->
### New in {VERSION}
A check of every app that talks to USB devices, after the Hantek 6022BE driver fix in 0.30.2:
- **Probie's logic analysers work with the libusb-win32 driver.** On Windows, Probie only accepted analysers with WinUSB, so one set up for PulseView with Zadig's libusb-win32 option was reported as needing a driver. It now falls back to libusb-win32 (the same code Scopie uses, now shared), so these analysers work as they are.
- **Meterie explains missing meter cables.** On Windows, a UNI-T, Victor, Brymen or USB-serial meter cable whose driver was replaced by Zadig (sigrok's guides do this) used to vanish from Meterie's list. It's now listed, in red, with the driver it has and how to give it back to Windows.
- **Probie says when a COM port is busy**, like the other apps do, instead of "Access is denied".
- **Scopie's Hantek DSO-2xxx/5xxx message** says libusb-win32 works as well as WinUSB.
- **A new test** checks that the Linux and Raspberry Pi USB permission rule (install-usb.sh) covers every scope, logic analyser and HID meter cable the apps support. It does today.

Licensed under GPL-3.0-or-later.

Windows code signing is being set up (until then Windows may warn about an unrecognised app): free code signing provided by [SignPath.io](https://signpath.io), certificate by [SignPath Foundation](https://signpath.org). See the [code signing policy](https://tinkerie.net/code-signing.html) and [privacy statement](https://tinkerie.net/code-signing.html#privacy).
