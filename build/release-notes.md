> ⚠️ **Tinkerie is in alpha.** Everything works, but it's still being tested on real hardware, so expect rough edges and changes. Please [report](https://github.com/normanlove/tinkerie/issues) anything that doesn't work.

Free maker tools by **nrmluvdev**: Wirie, Makie, Tunie, Flashie, Scopie, Meterie, Probie, Codie and Fixie in one program.

### Which file?
- **Windows 10/11:** `Tinkerie-{VERSION}-windows.exe` (just run it), or `Tinkerie-{VERSION}-windows.zip` if your connection blocks .exe downloads (it also has the source)
- **Mac:** `…-mac-apple-silicon.zip` (M1 or newer) or `…-mac-intel.zip`
- **Linux PC:** `…-linux-x86_64.tar.gz`
- **Raspberry Pi:** `…-pi-64bit.tar.gz` (64-bit Raspberry Pi OS) or `…-pi-32bit.tar.gz`

Each package has a *READ ME FIRST.txt*.

<!-- release notes for 0.30.0-alpha: package.sh checks this line, so update the notes with every version -->
### New in {VERSION}
- **Fixie shows you what to look for.** Board level, My build and Parts tests steps now have drawings (click to enlarge): 55 of them.
  - **The parts**, with their pins labelled: colour-banded and SMD resistors, electrolytic and ceramic capacitors, diodes, LEDs, bridge rectifiers, transistors, MOSFETs, regulators (with their different pinouts side by side), buck converters, crystals, logic chips and microcontrollers (finding pin 1), op-amps, 555s, optocouplers, fuses, relays, buttons, lithium cells, character LCDs, barrel plugs and USB.
  - **How to connect**: measuring a voltage, diode tests, resistance with a lifted leg, continuity, current in series, a scope probe on a crystal, a logic analyser on I2C, a serial adapter for boot messages, injecting current to find a short, following a test tone, keyboard matrix scanning, MOSFET drivers, breadboard rails, I2C pull-ups, decoupling, LED strips, the inside of a mains power supply.
  - **Good and faulty signals side by side**: crystals, switching regulators, I2C ACK and NAK, weak pull-ups, serial bytes, reset lines, composite video and VGA sync, PWM, servo pulses, brownouts, noise and clipping.
- **Reference cards and calculators** on the Parts tests tab and from each step: resistor colour codes, SMD resistor markings (including EIA-96), capacitor markings, common pinouts, what board labels (R, C, Q, U, Y…) mean, finding pin 1, rail voltages, and Ohm's law, LED resistor and regulator heat calculators.

Licensed under GPL-3.0-or-later.

Windows code signing is being set up (until then Windows may warn about an unrecognised app): free code signing provided by [SignPath.io](https://signpath.io), certificate by [SignPath Foundation](https://signpath.org). See the [code signing policy](https://tinkerie.net/code-signing.html) and [privacy statement](https://tinkerie.net/code-signing.html#privacy).
