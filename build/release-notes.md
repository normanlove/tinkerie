> ⚠️ **Tinkerie is in alpha.** Everything works, but it's still being tested on real hardware, so expect rough edges and changes. Please [report](https://github.com/normanlove/tinkerie/issues) anything that doesn't work.

Free maker tools by **nrmluvdev**: Wirie, Makie, Tunie, Flashie, Scopie, Meterie, Probie, Codie and Fixie in one program.

### Which file?
- **Windows 10/11:** `Tinkerie-{VERSION}-windows.exe` (just run it), or `Tinkerie-{VERSION}-windows.zip` if your connection blocks .exe downloads (it also has the source)
- **Mac:** `…-mac-apple-silicon.zip` (M1 or newer) or `…-mac-intel.zip`
- **Linux PC:** `…-linux-x86_64.tar.gz`
- **Raspberry Pi:** `…-pi-64bit.tar.gz` (64-bit Raspberry Pi OS) or `…-pi-32bit.tar.gz`

Each package has a *READ ME FIRST.txt*.

<!-- release notes for 0.33.0-alpha: package.sh checks this line, so update the notes with every version -->
### New in {VERSION}
- **Install Tinkerie, and keep it up to date, from the launcher.** Click **Install & updates** at the bottom of the launcher:
  - **Install on this computer** puts Tinkerie in your Start menu (Windows) or applications menu (Linux, Raspberry Pi) with a shortcut for every app. No administrator rights or password are needed. On Windows you can add a desktop shortcut too, and remove it again from *Settings ▸ Apps*.
  - **Check for updates** asks GitHub whether there's a newer Tinkerie, and **Update now** downloads it, checks it against the SHA-256 GitHub publishes for it, and swaps it in (Windows, Linux and Raspberry Pi; on a Mac it opens the download page). Apps that are open keep running until you close them.
  - Tinkerie only asks GitHub when you press the button, unless you tick *Check for updates when Tinkerie starts* (at most once a day). Nothing about you or your computer is sent.
  - Running Tinkerie straight from the download still works exactly as before.
- Also in 0.32.1: Tinkerie web keeps every tab, app and person's work apart.

Licensed under GPL-3.0-or-later.

Windows code signing is being set up (until then Windows may warn about an unrecognised app): free code signing provided by [SignPath.io](https://signpath.io), certificate by [SignPath Foundation](https://signpath.org). See the [code signing policy](https://tinkerie.net/code-signing.html) and [privacy statement](https://tinkerie.net/code-signing.html#privacy).
