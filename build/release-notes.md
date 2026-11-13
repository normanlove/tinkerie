> ⚠️ **Tinkerie is in alpha.** Everything works, but it's still being tested on real hardware, so expect rough edges and changes. Please [report](https://github.com/normanlove/tinkerie/issues) anything that doesn't work.

Free maker tools by **nrmluvdev**: Wirie, Makie, Tunie, Flashie, Scopie, Meterie, Probie, Codie and Fixie in one program.

### Which file?
- **Windows 10/11:** `Tinkerie-{VERSION}-windows.exe` (just run it), or `Tinkerie-{VERSION}-windows.zip` if your connection blocks .exe downloads (it also has the source)
- **Mac:** `…-mac-apple-silicon.zip` (M1 or newer) or `…-mac-intel.zip`
- **Linux PC:** `…-linux-x86_64.tar.gz`
- **Raspberry Pi:** `…-pi-64bit.tar.gz` (64-bit Raspberry Pi OS) or `…-pi-32bit.tar.gz`

Each package has a *READ ME FIRST.txt*.

<!-- release notes for 0.28.2-alpha: package.sh checks this line, so update the notes with every version -->
### New in {VERSION}
- **Fixie: a deeper check of every word.** A new automatic check reads everything a person can see (titles, explanations, instructions, questions, answers, readings, causes and fixes) for every kind of device and every combination of power and features (about 240,000), and flags text that mentions something the device doesn't have. It found about 70 real mismatches, all fixed: for example charger and battery advice for mains-only devices, "power button" steps for devices without one, phone and laptop wording on other devices, USB checks for adapter-powered ones, TV and laptop backlight wording on every screen, mains warnings on low-voltage devices, and the upload symptom offered for boards with no USB.
- **Where answers lead**: 5 fixes. "It starts now" after reseating memory no longer says the switch is faulty; a healthy battery that does charge now points to a wrong battery reading (not a worn battery); a board with no power LED gets its power measured before hunting for a short; "Neither" became "None of these" on a four-way question; devices without a fan that run hotter go to a look for failing parts.
- **Tool and picture chips** on each step only show what fits the device (no USB power meter for a mains adapter, no swollen-battery picture for a device with no battery).
- The wording check runs on every build, so these can't come back.

Licensed under GPL-3.0-or-later.

Windows code signing is being set up (until then Windows may warn about an unrecognised app): free code signing provided by [SignPath.io](https://signpath.io), certificate by [SignPath Foundation](https://signpath.org). See the [code signing policy](https://tinkerie.net/code-signing.html) and [privacy statement](https://tinkerie.net/code-signing.html#privacy).
