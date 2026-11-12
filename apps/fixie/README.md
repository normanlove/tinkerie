# Fixie 0.1.0-alpha — repair guide

Part of **Tinkerie**, the free maker tools by **nrmluvdev**.

Fixie takes you step by step through finding what's wrong with a broken electronic device. Pick what you're fixing and what it's doing, and Fixie asks you to look, measure and test one thing at a time, easiest first. Each step says what to do and why, which tools to use, and what a good and a bad result look like. Each answer leads to the next check, until Fixie reaches the likely fault, how hard it is to fix, and the parts and tools you'll need.

Fixie never uses the internet. Repairs are saved on this computer.

## What it covers

| | |
|---|---|
| Devices | Phones and tablets, laptops, desktop PCs, games consoles, controllers, remotes, keyboards and mice, TVs and monitors, amplifiers and radios, headphones and Bluetooth speakers, kettles/toasters/irons/hair dryers, fans/blenders/vacuums, washing machines and fridges, cordless tools and battery packs, LED lamps and strips, Arduino/ESP32/Pico projects, 3D printers, battery toys, and microwaves (outside checks only) |
| Symptoms | Completely dead; turns on then off; won't charge; no picture; no sound; buttons or touch; motors and fans; heating; USB, Wi-Fi and Bluetooth; code won't upload; LEDs; overheating or burning; water damage; faults that come and go; trips the electricity |

Each device has tags (how it's powered, whether it has a screen, battery, motor or heater, whether it holds mains voltage), and steps that don't apply are skipped: no plug-fuse check on a phone, no battery check on a kettle.

## The tabs

- **Repair**: the guide. Each step has *What to do*, *Tools* (click one to see how to use it), *What to look for* (good and faulty), *What it looks like* (pictures of failed parts), and a box for your reading where a measurement is involved. Fixie compares the reading with what it should be (a charger against its label, a rail against its voltage, a heating element against its voltage and power) and highlights the answer it points to. Answer with a click or the number keys; go back with Alt+Left or by clicking any earlier step in *Your path*. Notes are kept with each step.
- **Tools**: 26 tools, from a multimeter to a hot air station, with what each is for, roughly what it costs and how to use it. Tick the ones you have and each step marks them.
- **What to look for**: good and faulty parts side by side: bulging capacitors, burnt parts, corrosion, cracked solder joints, blown fuses, swollen batteries, damaged connectors, and more.
- **Safety**: mains and big capacitors, things never to open at home (microwaves, CRTs, camera flashes), lithium batteries, soldering, static and chemicals.
- **My repairs**: every repair you start, saved as you go, with Open, Report and Delete.

## Safety built in

- Steps inside a device that runs from the mains or has a lithium battery have a safety check to tick before the answers unlock.
- Steps that measure with the power on carry a warning.
- Microwave ovens only get the checks you can do from outside; any step that would open one leads to a qualified repairer instead.
- Each fix is marked Easy, Medium, Hard or Pro, and says when a repair shop is the better choice.

## Reports

**Save report** saves the repair as a Markdown text file: every step, your answers, readings and notes, and the fault and fix it reached.

## For developers

The guide is data: `web/guide.json`, built by `guide.py` (`python3 apps/fixie/guide.py apps/fixie/web/guide.json`). Steps have a title, why, what to do (lines can depend on the device's tags), tools, signs, an optional measurement, a question and answers that lead to other steps; results have the cause, fix, parts and difficulty. `fixie_test.go` follows every answer for every device and symptom and checks each path ends at a result with no loops, and that every tool, sign and tag it names exists.

Fixie runs on port 47825 (see PORTING.md) and keeps its repair log in `nrmluvdev/Fixie/state.json` in the system's settings folder.

Created by nrmluvdev.
