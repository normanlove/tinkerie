# Codie 0.1.0-alpha — code explainer

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

## Opening files

- Open file…, or drag a file onto the code box.
- `Tinkerie --app codie path/to/file.py` opens a file directly; if Codie is already open, the file goes to that window.

## How it works

`lang.go` guesses the language from telltale patterns. `lex.go` separates each line's code from its comments and text values, so keywords inside strings don't confuse it. The rules in `rules_*.go` explain statements by pattern, per language family; `generic.go` explains assignments and calls that no rule knows, wording them from the names used. `api_*.go` hold the well-known functions in a simple table format. `game.go` holds the engines and game concepts, `lessons.go` the Game school, `concepts.go` the glossary and warnings. Blocks are tracked by braces, indentation or keywords (end, NEXT, WEND, fi…) to build the outline and explain closing lines.

Port 47824 (any free port if it's busy). Data and log: the system config folder, `nrmluvdev/Codie`.

## Tests

```
go test ./apps/codie/
```

Every example and Game school sample must be detected as its own language and have at least 90% of its lines explained by a real rule. `CODIE_SHOW="name"` with `-run Show -v` prints the full analysis of an example.

Licensed under GPL-3.0-or-later. Made by nrmluvdev.
