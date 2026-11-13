# Fixie 0.4.0-alpha — repair guide

Part of **Tinkerie**, the free maker tools by **nrmluvdev**.

Fixie takes you step by step through finding what's wrong with a broken electronic device, down to the failed part, so old tech can be repaired and reused instead of becoming e-waste. It also troubleshoots circuits you've built yourself. Describe what you're fixing and say what it's doing, and Fixie asks you to look, measure and test one thing at a time, easiest first. Each step says what to do and why, which tools to use, and what a good and a bad result look like. Each answer leads to the next check, until Fixie reaches the likely fault, how hard it is to fix, and the parts and tools you'll need.

Fixie never uses the internet. Repairs are saved on this computer.

## The guides

- **Quick checks**: for any device. Describe it (how it's powered, what it has) and say what it's doing; Fixie goes through the easy checks first.
- **Board level**: component-level fault finding for repairing old tech. Pick a section of the board, or start from the beginning: power input and protection, mains power supply (cold tests, then a dim bulb tester), regulators and rails (finding a shorted part by injecting current), battery and charging, clock and reset, processor and memory (boot messages, corrupted firmware), data buses (decoding I2C, SPI and UART to see who doesn't answer), audio (following a test tone), video (sync and levels), outputs and drivers, inputs and keyboard matrices, and connectors and tracks. Results say what's worth salvaging when a board can't be saved.
- **My build**: troubleshooting your own projects: power and current draw, parts fitted backwards, regulator pinouts, wiring against your design, code that won't run or keeps resetting, I2C/SPI/serial parts that don't answer, signals, noise and decoupling, motors, servos and steppers, and addressable LEDs.
- **Parts tests**: how to test 25 kinds of part (out of the circuit, in it, and powered), what to replace them with, and whether they're worth saving from scrap.

## Pictures and reference

Board level, My build and Parts tests steps show drawings (click one to see it bigger): what each part looks like with its pins labelled, where the meter probes, scope probe, logic analyser or serial adapter connect, and what the signal should look like next to a faulty one (crystals, switching nodes, I2C ACK/NAK and pull-up edges, serial bytes, reset lines, composite and VGA video, PWM, servo pulses, brownouts, noise, clipping, and what a dim bulb tester's bulb tells you). There are 55 in `web/pics.js`, drawn as SVG and coloured by the theme.

Reference cards (on the Parts tests tab, and from a step's Reference chips): a resistor colour-code calculator, SMD resistor marking decoder (3- and 4-digit, R notation, EIA-96), capacitor marking decoder, common pinouts (78xx, 79xx, LM317, AMS1117, TL431, common transistors and MOSFETs, PC817, 555, dual op-amps, logic chips, USB, VGA), what board labels mean, finding pin 1, common rail voltages, and Ohm's law, LED resistor and regulator heat calculators. They're in `web/refs.js`.

## Working with the rest of Tinkerie

Steps say which Tinkerie app to use and how to set it up, with a button that opens it: Meterie (your multimeter), Scopie (your oscilloscope), Probie (logic analyser and serial monitor), Flashie (firmware) and Wirie (your design). Where a step needs a reading, **Use Meterie's reading**, **Use Scopie's frequency** (or peak-to-peak, minimum…) on a chosen channel, or **Use Probie's frequency/duty cycle** on a chosen logic channel fills it in, converted to the step's units, and Fixie judges it. Fixie reads Meterie through its suite API (port 47823), Scopie through its status stream (47831) and Probie through its logic measurement (47822); the apps must be open and measuring.

## Describing the device

Fixie has no list of products to choose from, so it works for almost anything electronic. You describe it instead:

- **What is it?** In your own words ("Bluetooth speaker", "LED desk lamp").
- **Closest kind of thing**: a shortcut that fills in the usual answers below: Computer or laptop, Phone or tablet, Games console or controller, TV or display, Speaker/headphones/amp/radio, Toy or gadget, Cordless tool or battery pack, Lights and LEDs, Electronics project or board, Keyboard/mouse/remote, 3D printer or machine with motors, Household gadget, or Something else.
- **How is it powered?** Mains, adapter, USB, swappable batteries, built-in rechargeable battery (all that apply).
- **What does it have?** Screen, sound, buttons, touchscreen, wireless, USB, motor, cooling fan, LED lighting, a hobby board you program yourself (Arduino, ESP32, Pico…), a low-voltage heater (3D printer hot end or bed).

Only the symptoms and checks that fit are offered: no plug-fuse check on a phone, no picture checks without a screen. The symptoms: completely dead; turns on then off; won't charge; no picture; no sound; buttons or touch; motors and fans; low-voltage heaters; USB, Wi-Fi and Bluetooth; code won't upload; LEDs; overheating or burning; water damage; faults that come and go; trips the electricity.

## What Fixie won't help with

For safety, Fixie gives **no repair steps at all** for these. It checks the name you type (and refuses anything you say heats things up from the mains), shows why, and tells you to take it to the maker or a qualified repairer, or recycle it:

- Microwave ovens
- Anything that heats from the mains: ovens, cookers, hobs, toasters, kettles, air fryers, coffee machines, heaters, irons, hair dryers and straighteners, electric blankets, soldering irons…
- Large appliances: washing machines, dishwashers, dryers, fridges, freezers, air conditioners
- High-voltage devices: CRT TVs and monitors, camera flashes, bug zappers, stun guns, neon signs, laser cutters, welders…
- Anything wired into the house, and gas appliances
- Big batteries: e-bikes, e-scooters, hoverboards, electric cars, car batteries, home batteries and inverters
- Medical and life-safety devices: pacemakers, CPAP machines, hearing aids, smoke and carbon monoxide alarms…

The full word list is in `guide.py`; `fixie_test.go` checks it refuses these and doesn't refuse ordinary devices by mistake. Repairs saved by Fixie 0.1 for kettles, toasters, irons, hair dryers, washing machines and microwaves now show the same refusal.

## The tabs

- **Repair**: the guide. Each step has *What to do*, *Tools* (click one to see how to use it), *What to look for* (good and faulty), *What it looks like* (pictures of failed parts), and a box for your reading where a measurement is involved. Fixie compares the reading with what it should be (a charger against its label, a rail against its voltage, a heating element against its voltage and power) and highlights the answer it points to. Answer with a click or the number keys; go back with Alt+Left or by clicking any earlier step in *Your path*. Notes are kept with each step.
- **Tools**: 26 tools, from a multimeter to a hot air station, with what each is for, roughly what it costs and how to use it. Tick the ones you have and each step marks them.
- **What to look for**: good and faulty parts side by side: bulging capacitors, burnt parts, corrosion, cracked solder joints, blown fuses, swollen batteries, damaged connectors, and more.
- **Safety**: what Fixie won't help with, then mains and big capacitors, lithium batteries, soldering, static and chemicals.
- **My repairs**: every repair you start, saved as you go, with Open, Report and Delete.

## Safety built in

- Dangerous devices are refused outright (see above).
- Steps inside a device that runs from the mains or has a lithium battery have a safety check to tick before the answers unlock.
- Steps that measure with the power on carry a warning.
- Each fix is marked Easy, Medium, Hard or Pro, and says when a repair shop is the better choice.

## Reports

**Save report** saves the repair as a Markdown text file: every step, your answers, readings and notes, and the fault and fix it reached.

## For developers

The guide is data: `web/guide.json`, built by `guide.py` (Quick checks, devices, tools, signs, safety) and `guide_more.py` (Board level, My build, Parts tests) (`python3 apps/fixie/guide.py apps/fixie/web/guide.json`). Steps have a title, why, what to do (lines can depend on the device's tags), tools, signs, an optional measurement, a question and answers that lead to other steps; results have the cause, fix, parts and difficulty. Lines in a step (and in a fix) can depend on the device: `iff("motor&mains&!printer", …)` (`|` is or, `&` is and, `!` is not). `fixie_test.go` follows every answer from every symptom, board section and build problem, for every kind of device and every combination of power and features, and checks each path ends at a result with no loops, that no step or fix is ever empty for the device in front of it, that every tool, sign, tag, picture and reference card it names exists, and that the refusal list works. `TestWording` reads every word a person can see (titles, explanations, instructions, questions, answers, readings, causes and fixes) for every kind of device and every combination of power and features, and fails if text mentions something the device doesn't have: a battery, a screen, a fan, USB, mains, a phone, a printer and so on. Tool and sign chips on a step are filtered the same way. `TestRelevance` lists steps each kind of device must never be shown (a 3D printer never gets the hobby-board upload steps, a cordless tool never gets USB charging checks, and so on).

Fixie runs on port 47825 (see PORTING.md) and keeps its repair log in `nrmluvdev/Fixie/state.json` in the system's settings folder.

Created by nrmluvdev.
