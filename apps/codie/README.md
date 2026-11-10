# Codie 0.2.0-alpha — code and signal explainer

Part of **Tinkerie**, the free maker tools by **nrmluvdev**.

Paste code (or open a file) and Codie explains it line by line: **what** each part does and **why** it's there, in plain English. It's like the explanations in Flashie's Create tab, but for code you already have, in the languages people actually meet.

Everything happens on your computer. Codie reads code; it never runs it and never uploads it.

## What it reads

| Group | Languages |
|---|---|
| Everyday | Python, JavaScript, TypeScript, C, C++, C#, Java, Go, Rust, Kotlin, Swift, PHP, Ruby, SQL, HTML, CSS |
| Windows Go | `//go:build windows` files, `syscall.NewLazyDLL` / `NewProc` / `Call`, `unsafe.Pointer` and `uintptr`, UTF-16 text, `golang.org/x/sys/windows`, the registry and services, with what common Windows functions (MessageBoxW, CreateFileW, GetAsyncKeyState…) do |
| Scripts | Bash, PowerShell, Windows batch (.bat/.cmd) |
| Game languages | Lua (LÖVE, PICO-8, Roblox Luau, Solar2D), GDScript (Godot), GameMaker GML |
| Older machines | BASIC: Commodore 64, ZX Spectrum, BBC Micro, Apple II, QBasic/QB64, Visual Basic. POKE and PEEK addresses are explained (border colour, sprites, joystick, SID sound…) |

Codie works out the language and its flavour (Unity, Unreal, Arduino, Windows Go, C64 BASIC…) by itself, or you can pick it from the Language menu.

## The tabs

- **Explain**: a card for every line, or every statement on a BASIC line. Click a card to jump to the code; click a line of code to find its card. Names in your code are used too: `updatePlayer()` "probably updates the player", `score` "probably keeps the player's points".
- **Outline**: functions, classes, loops and checks, nested as in the code.
- **Concepts**: the ideas the code uses (variables, loops, functions, classes, errors, Windows Go…), each explained, with the lines that use them.
- **Game logic**: for game code, the engine or machine it's for, how that engine runs one frame (in order, marking the steps this code uses), engine tips, and where the code handles the game loop, delta time, input, movement, gravity, collisions, spawning, timers, randomness, score, states, animation and sound. Recognises Unity, Unreal, Godot, LÖVE, PICO-8, Roblox, Solar2D, pygame, Phaser, HTML canvas, raylib, SDL, MonoGame, GameMaker, Ebitengine, Bevy, SpriteKit, Arduino and the BASIC of the C64, ZX Spectrum, BBC Micro, Apple II and QBasic/QB64.
- **Warnings**: common slips and risky habits, each with a tip (`=` instead of `==`, ignored Go errors, `uintptr` kept in a variable, `GetComponent` every frame, `while true` without a wait in Roblox, `DELETE` without `WHERE`…).
- **Game school**: twelve short lessons (the game loop, delta time, input, gravity and jumping, collisions, game states, spawning, timers, randomness, score and saving, animation, games on 8-bit machines). Each has working examples in several languages; **Explain this example** opens one in the editor.

**Save report** writes the whole explanation as a Markdown file. **F2** (or the **?** button) turns teaching mode on and off.

## Signals mode

Codie also explains **captures** from the rest of Tinkerie, the same way it explains code: a card for the capture, each channel and each protocol or event, saying what it is, why it matters and the numbers behind it.

| From | File | How to send it |
|---|---|---|
| Scopie | waveform `.csv` (time + volts) | **Explain in Codie** (toolbar or File menu) |
| Tracie | logic capture `.vcd` | Export → **Explain in Codie** |
| Tracie | serial log `.csv` | Switch on Log to CSV, then **Explain in Codie** |
| Meterie | suite timeline `.nst.csv` | Export → **Explain in Codie** |

Plain CSV files (a time column and number columns) and VCD files from other tools work too. **From the suite** lists recent captures in `Documents\Scopie`, `Documents\Tracie` and `Documents\Meterie`.

What Codie recognises:
- **Scope channels:** steady DC levels (with what they probably are: 3.3 V rail, USB 5 V, a Li-ion cell…), ripple and its likely source, sine, triangle and sawtooth waves (with the musical note in the audio range, and mains at 50/60 Hz), one-off changes with their RC time constant, and signals switching between two levels, which are then treated as logic (below) with their logic family, rise time and overshoot.
- **Logic channels:** clocks and square waves, PWM (with a changing duty cycle), servo signals (with the angle), UART serial (baud rate found automatically, decoded to text), I2C (address, read/write, ACK/NACK, data, with the common chip at each address), SPI (clock, chip select, MOSI and MISO, flash commands and JEDEC IDs), WS2812/NeoPixel LED colours, NEC infrared remote codes and 1-Wire (DS18B20 temperatures). Channel names such as SCL, SDA, SCK, MOSI, CS and TX are used when they're there.
- **Meter readings:** trends, batteries charging and discharging (Li-ion, AA, lead-acid, USB), sleep/wake current patterns with battery-life estimates, power and energy when a voltage and a current are logged together, temperature, sudden changes and over-range readings.
- **Serial logs:** ESP32/ESP8266 boot and reset reasons, crashes and Backtraces, brownouts, watchdog resets, Python tracebacks, MicroPython, Wi-Fi, NMEA GPS fixes, AT commands, JSON and error messages, plus the numbers in each line as readings over time.
- **Warnings:** too few samples per cycle, clipping, slow edges, overshoot, 5 V signals that can harm 3.3 V boards, weak HIGH levels, ripple, framing errors, I2C devices that didn't answer, brownouts and crashes.

The capture view draws every channel; click a card to see where it happens, click the capture to find the card that explains it, scroll to zoom and drag to move. **Decoded** lists every decoded byte; **Save report** writes it all as Markdown. Captures up to 96 MB.

## Opening files

- Open file…, or drag a file onto the code box.
- `Tinkerie --app codie path/to/file.py` opens a file directly; if Codie is already open, the file goes to that window.

## How it works (signals)

`signals.go` reads the file formats, `sig_analyse.go` measures analog and meter channels and writes the explanations, `sig_decode.go` turns channels into edges and decodes the protocols, `sig_text.go` holds the glossary and the serial-log rules, and `sig_demo.go` builds the example captures in code (they're also the tests).

## How it works

`lang.go` guesses the language from telltale patterns. `lex.go` separates each line's code from its comments and text values, so keywords inside strings don't confuse it. The rules in `rules_*.go` explain statements by pattern, per language family; `generic.go` explains assignments and calls that no rule knows, wording them from the names used. `api_*.go` hold the well-known functions in a simple table format. `game.go` holds the engines and game concepts, `lessons.go` the Game school, `concepts.go` the glossary and warnings. Blocks are tracked by braces, indentation or keywords (end, NEXT, WEND, fi…) to build the outline and explain closing lines.

Port 47824 (any free port if it's busy). Data and log: the system config folder, `nrmluvdev/Codie`.

## Tests

```
go test ./apps/codie/
```

Every example and Game school sample must be detected as its own language and have at least 90% of its lines explained by a real rule. `CODIE_SHOW="name"` with `-run Show -v` prints the full analysis of an example.

Licensed under GPL-3.0-or-later. Made by nrmluvdev.
