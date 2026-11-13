> ⚠️ **Tinkerie is in alpha.** Everything works, but it's still being tested on real hardware, so expect rough edges and changes. Please [report](https://github.com/normanlove/tinkerie/issues) anything that doesn't work.

Free maker tools by **nrmluvdev**: Wirie, Makie, Tunie, Flashie, Scopie, Meterie, Probie, Codie and Fixie in one program.

### Which file?
- **Windows 10/11:** `Tinkerie-{VERSION}-windows.exe` (just run it), or `Tinkerie-{VERSION}-windows.zip` if your connection blocks .exe downloads (it also has the source)
- **Mac:** `…-mac-apple-silicon.zip` (M1 or newer) or `…-mac-intel.zip`
- **Linux PC:** `…-linux-x86_64.tar.gz`
- **Raspberry Pi:** `…-pi-64bit.tar.gz` (64-bit Raspberry Pi OS) or `…-pi-32bit.tar.gz`

Each package has a *READ ME FIRST.txt*.

<!-- release notes for 0.29.0-alpha: package.sh checks this line, so update the notes with every version -->
### New in {VERSION}
- **Fixie goes down to component level**, for repairing and reusing old tech instead of throwing it away, and for your own builds. Three guides now: **Quick checks** (the describe-any-device guide), **Board level** and **My build**, plus a **Parts tests** tab.
- **Board level**: pick a section of the board or start from the beginning: power input and protection, mains power supply (cold tests, then a dim bulb tester), regulators and rails (find a shorted part by injecting current), battery and charging, clock and reset, processor and memory (boot messages, corrupted firmware), data buses (decode I2C, SPI and UART to see which chip doesn't answer), audio (follow a test tone to where it stops), video (sync and levels), outputs and drivers, inputs and keyboard matrices, connectors and tracks. Salvage notes say what's worth keeping when a board can't be saved.
- **My build**: power and current draw, parts fitted backwards, regulator pinouts, wiring against your design, code that won't run or resets, I2C/SPI/serial parts that don't answer, signals, noise and decoupling, motors, servos and steppers, addressable LEDs.
- **Parts tests**: 25 kinds of part, from resistors and capacitors to MOSFETs, crystals, microcontrollers, relays and lithium cells: how they fail, how to test them out of the circuit, in it and powered, what to replace them with, and whether they're worth saving from scrap.
- **Works with the rest of Tinkerie**: steps say which app to use and how to set it up, and open it for you (Meterie, Scopie, Probie, Flashie, Wirie). A reading can be filled straight from Meterie's meter, a Scopie measurement on a chosen channel, or a Probie logic channel's frequency or duty cycle, converted to the step's units and judged.

Licensed under GPL-3.0-or-later.

Windows code signing is being set up (until then Windows may warn about an unrecognised app): free code signing provided by [SignPath.io](https://signpath.io), certificate by [SignPath Foundation](https://signpath.org). See the [code signing policy](https://tinkerie.net/code-signing.html) and [privacy statement](https://tinkerie.net/code-signing.html#privacy).
