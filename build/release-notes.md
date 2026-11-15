> ⚠️ **Tinkerie is in alpha.** Everything works, but it's still being tested on real hardware, so expect rough edges and changes. Please [report](https://github.com/normanlove/tinkerie/issues) anything that doesn't work.

Free maker tools by **nrmluvdev**: Wirie, Makie, Tunie, Flashie, Scopie, Meterie, Probie, Codie and Fixie in one program.

### Which file?
- **Windows 10/11:** `Tinkerie-{VERSION}-windows.exe` (just run it), or `Tinkerie-{VERSION}-windows.zip` if your connection blocks .exe downloads (it also has the source)
- **Mac:** `…-mac-apple-silicon.zip` (M1 or newer) or `…-mac-intel.zip`
- **Linux PC:** `…-linux-x86_64.tar.gz`
- **Raspberry Pi:** `…-pi-64bit.tar.gz` (64-bit Raspberry Pi OS) or `…-pi-32bit.tar.gz`

Each package has a *READ ME FIRST.txt*.

<!-- release notes for 0.32.1-alpha: package.sh checks this line, so update the notes with every version -->
### New in {VERSION}
- **Tinkerie web keeps everyone's work apart.** There are no accounts and no server: each person's work stays in their own browser. Now each tab and each app keeps to itself too:
  - every open tab gets its own random id, claimed while it's open, so a duplicated tab can never talk to another tab's app;
  - each app has its own part of the browser's storage and cookies (as the desktop apps do), so apps never see or overwrite each other's settings or autosaves;
  - opening the same app in a second tab shows a warning, since both save to the same files;
  - for shared computers: **Clear my data** in each app's Files, and **Clear all my Tinkerie web data** on the Tinkerie web page (files, settings and device permissions).
- Also in 0.32.0: Fixie explains error logs from Windows, macOS, Linux, Raspberry Pi OS and Android, with no AI and no internet.

Licensed under GPL-3.0-or-later.

Windows code signing is being set up (until then Windows may warn about an unrecognised app): free code signing provided by [SignPath.io](https://signpath.io), certificate by [SignPath Foundation](https://signpath.org). See the [code signing policy](https://tinkerie.net/code-signing.html) and [privacy statement](https://tinkerie.net/code-signing.html#privacy).
