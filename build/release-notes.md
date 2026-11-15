> ⚠️ **Tinkerie is in alpha.** Everything works, but it's still being tested on real hardware, so expect rough edges and changes. Please [report](https://github.com/normanlove/tinkerie/issues) anything that doesn't work.

Free maker tools by **nrmluvdev**: Wirie, Makie, Tunie, Flashie, Scopie, Meterie, Probie, Codie and Fixie in one program.

### Which file?
- **Windows 10/11:** `Tinkerie-{VERSION}-windows.exe` (just run it), or `Tinkerie-{VERSION}-windows.zip` if your connection blocks .exe downloads (it also has the source)
- **Mac:** `…-mac-apple-silicon.zip` (M1 or newer) or `…-mac-intel.zip`
- **Linux PC:** `…-linux-x86_64.tar.gz`
- **Raspberry Pi:** `…-pi-64bit.tar.gz` (64-bit Raspberry Pi OS) or `…-pi-32bit.tar.gz`

Each package has a *READ ME FIRST.txt*.

<!-- release notes for 0.32.0-alpha: package.sh checks this line, so update the notes with every version -->
### New in {VERSION}
- **Fixie explains error logs (Fixie 0.5.0).** A new *Error logs* tab reads logs from Windows, macOS, Linux, Raspberry Pi OS, ChromeOS and Android, and lists the error messages in them, most serious first, with what each means and what to do: failing drives, bad memory, overheating, under-voltage, blue screen stop codes, kernel panics, crashing apps, USB and Wi-Fi faults, and the scary-looking ones that are harmless.
  - Open, paste or drop a log (Event Viewer's XML, text or CSV, PowerShell, journalctl, dmesg, macOS crash and panic reports, logcat), or in the desktop app press *Scan this computer* to read its own logs.
  - No AI and no internet: 166 known messages, written by hand, matched on your computer. Lines that look like errors but aren't in the list are shown too.
  - In Tinkerie web too, for logs you open or paste.

Licensed under GPL-3.0-or-later.

Windows code signing is being set up (until then Windows may warn about an unrecognised app): free code signing provided by [SignPath.io](https://signpath.io), certificate by [SignPath Foundation](https://signpath.org). See the [code signing policy](https://tinkerie.net/code-signing.html) and [privacy statement](https://tinkerie.net/code-signing.html#privacy).
