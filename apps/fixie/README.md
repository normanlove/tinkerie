# Fixie 0.2.1-alpha — repair guide

Part of **Tinkerie**, the free maker tools by **nrmluvdev**.

Fixie takes you step by step through finding what's wrong with a broken electronic device. Describe what you're fixing and say what it's doing, and Fixie asks you to look, measure and test one thing at a time, easiest first. Each step says what to do and why, which tools to use, and what a good and a bad result look like. Each answer leads to the next check, until Fixie reaches the likely fault, how hard it is to fix, and the parts and tools you'll need.

Fixie never uses the internet. Repairs are saved on this computer.

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

The guide is data: `web/guide.json`, built by `guide.py` (`python3 apps/fixie/guide.py apps/fixie/web/guide.json`). Steps have a title, why, what to do (lines can depend on the device's tags), tools, signs, an optional measurement, a question and answers that lead to other steps; results have the cause, fix, parts and difficulty. Lines in a step (and in a fix) can depend on the device: `iff("motor&mains&!printer", …)` (`|` is or, `&` is and, `!` is not). `fixie_test.go` follows every answer for every kind of device and every combination of power and features, and checks each path ends at a result with no loops, that no step or fix is ever empty for the device in front of it, that every tool, sign and tag it names exists, and that the refusal list works. `TestRelevance` lists steps each kind of device must never be shown (a 3D printer never gets the hobby-board upload steps, a cordless tool never gets USB charging checks, and so on).

Fixie runs on port 47825 (see PORTING.md) and keeps its repair log in `nrmluvdev/Fixie/state.json` in the system's settings folder.

Created by nrmluvdev.
