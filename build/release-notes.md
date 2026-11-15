> ⚠️ **Tinkerie is in alpha.** Everything works, but it's still being tested on real hardware, so expect rough edges and changes. Please [report](https://github.com/normanlove/tinkerie/issues) anything that doesn't work.

Free maker tools by **nrmluvdev**: Wirie, Makie, Tunie, Flashie, Scopie, Meterie, Probie, Codie and Fixie in one program.

### Which file?
- **Windows 10/11:** `Tinkerie-{VERSION}-windows.exe` (just run it), or `Tinkerie-{VERSION}-windows.zip` if your connection blocks .exe downloads (it also has the source)
- **Mac:** `…-mac-apple-silicon.zip` (M1 or newer) or `…-mac-intel.zip`
- **Linux PC:** `…-linux-x86_64.tar.gz`
- **Raspberry Pi:** `…-pi-64bit.tar.gz` (64-bit Raspberry Pi OS) or `…-pi-32bit.tar.gz`

Each package has a *READ ME FIRST.txt*.

<!-- release notes for 0.31.0-alpha: package.sh checks this line, so update the notes with every version -->
### New in {VERSION}
- **Tinkerie web: every app in your browser.** [tinkerie.net/web](https://tinkerie.net/web/) runs Wirie, Makie, Tunie, Flashie, Scopie, Meterie, Probie, Codie and Fixie in Chrome or Edge with nothing to download. It's the same code as the desktop apps, running in the page on your own computer, and what you make is kept in your browser.
  - **Devices work too**: flash ESP32, Arduino and STM32 boards, print to a USB printer, use Hantek 6022 and other USB scopes, fx2lafw logic analysers, serial monitors and meter cables, through the page's *Connect device* button.
  - **Each app says what's different**: its *Web vs desktop* button (and its page on the website) lists what works the same, what works differently and what needs the desktop app (writing SD cards, compiling Arduino code, network scopes and printers, Bluetooth meters).
  - Every update to the apps goes to the web versions automatically.
  - `Tinkerie-web-{VERSION}.tar.gz` puts it on your own web server; `install-pi.sh` sets it up on a Raspberry Pi.
- **Report a bug or ask for a feature** from [the website](https://tinkerie.net/feedback.html), or with the *Report a bug* button in each web app: a short form that opens a ready-filled GitHub issue, labelled with its app.

Licensed under GPL-3.0-or-later.

Windows code signing is being set up (until then Windows may warn about an unrecognised app): free code signing provided by [SignPath.io](https://signpath.io), certificate by [SignPath Foundation](https://signpath.org). See the [code signing policy](https://tinkerie.net/code-signing.html) and [privacy statement](https://tinkerie.net/code-signing.html#privacy).
