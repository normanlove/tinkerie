# Fixie's component-level guides: Board level (finding the failed part in old
# tech, to repair and reuse it instead of throwing it away), My build (your own
# projects), and Parts tests (how to test each kind of part). Run by guide.py,
# which provides step(), result(), iff(), steps and tools. Created by nrmluvdev.
#
# apps=[(app, how)] puts an "Open <app>" button on a step with what to set up
# there. measure["src"] lets the step fill its reading from another app:
#   "meterie"             Meterie's live meter reading
#   "scopie:<m>"          a Scopie measurement (freq, vpp, mean, rms, duty…), CH1 by default
#   "probie:freq|duty"    a Probie logic channel's frequency or duty cycle

# ---------------------------------------------------------------- more tools
tools += [
  dict(id="logic", name="Logic analyser", starter=False, cost="£10–£30",
       why="Records several digital signals at once and decodes them: I2C, SPI, UART, keyboard scans, sync pulses.",
       how=["Clip ground first, then one channel per signal.", "Set the sample rate at least 4× the fastest signal (a 400kHz I2C bus needs 2MHz or more).", "Add a decoder for the bus, then read the bytes it shows: addresses, ACK or NAK, data."],
       app="probie", appWhy="Probie runs cheap USB logic analysers, decodes the common buses and explains captures in plain words."),
  dict(id="esr", name="Component tester or ESR meter", starter=False, cost="£15–£40",
       why="Identifies and measures transistors, diodes and capacitors, and measures capacitors' ESR, which a multimeter can't.",
       how=["Discharge capacitors before testing them.", "Most testers work out an unknown part's pins for you: clip all three leads and press test.", "High ESR (compared with the chart on the tester) means a dried-out capacitor even when its µF reads fine."]),
  dict(id="dimbulb", name="Dim bulb tester", starter=False, cost="£10 (DIY, in a proper enclosure)",
       why="A filament lamp in series with a mains device limits the current, so a fault can't blow parts or fuses while you test.",
       how=["Build it only in a closed, earthed enclosure, or buy one.", "Use an old-style filament bulb (40–100W), not LED.", "Bright and stays bright: still a short. Flashes then dims: it starts normally."], **{"for": "mains"}),
  dict(id="signal", name="Signal source", starter=False, cost="Free–£30",
       why="A known test tone or signal to feed into a circuit, so you can follow it through with a scope.",
       how=["A phone's headphone output playing a 1kHz test tone (free apps) works for audio.", "A cheap function generator, or a microcontroller's PWM pin, gives square waves.", "Keep the level low: about 0.5V for audio inputs."]),
  dict(id="serial", name="USB-serial adapter", starter=False, cost="£3–£8",
       why="Lets a computer read the debug messages many boards print at start-up, and talk to serial devices.",
       how=["Connect its GND to the board's ground, its RX to the board's TX.", "Check the board's voltage first: most need a 3.3V adapter; 5V can damage them.", "Try common speeds: 115200, 9600, 57600, 74880 (ESP8266 boot messages)."],
       app="probie", appWhy="Probie's Serial tab shows and explains what the board prints."),
  dict(id="programmer", name="Chip programmer", starter=False, cost="£5–£20",
       why="Reads, backs up and rewrites memory chips and microcontrollers when a corrupted program is the fault.",
       how=["Always read and save a backup before writing anything.", "SPI flash chips: a CH341A programmer with a clip (check the chip's voltage: many are 1.8V or 3.3V).", "Microcontrollers: the maker's programmer (ST-Link, USBasp, a Pico running debugprobe)."],
       app="flashie", appWhy="Flashie writes firmware to ESP32, Pico, Arduino and STM32 boards."),
]

# ---------------------------------------------------------------- parts tests
# Each part: what it is, how it fails, how to test it (out of the circuit, in
# the circuit, powered), what to buy instead, and whether to save it from scrap.
def part(id, name, group, what, fails, tests, replace, salvage=None):
    return dict(id=id, name=name, group=group, what=what, fails=fails, tests=tests, replace=replace, **({"salvage": salvage} if salvage else {}))
def test(t, do, good, bad, measure=None, app=None):
    d = dict(t=t, do=do, good=good, bad=bad)
    if measure: d["measure"] = measure
    if app: d["app"] = dict(app=app[0], how=app[1])
    return d
M = "Set Meterie to show your meter, then press the button here to fill the reading in."

parts = [
  part("res", "Resistor", "Passive",
       "Small cylinders with coloured bands (through-hole), or black chips marked with a number code like 103 (10kΩ) or 4701 (4.7kΩ).",
       ["Goes open (infinite resistance) or high after overheating: often looks burnt or cracked.", "Rarely shorts. High-value resistors (over 100kΩ) in power supplies drift high with age."],
       [test("Out of the circuit (most accurate)", ["Lift one leg (or remove it) so nothing else is in parallel.", "Meter on resistance (Ω). Touch the probes to each end, keeping fingers off the metal."],
             "Within its tolerance of the marked value (usually 1% or 5%).", "Open (OL), or far from the marked value.",
             measure=dict(label="Resistance you measured", unit="Ω", src="meterie", near=dict(label="Marked value", unit="Ω", lo=0.95, hi=1.05))),
        test("In the circuit", ["Power off, capacitors discharged. Measure across it in place."],
             "Equal to or lower than the marked value (other parts in parallel can only lower it).", "Higher than the marked value: it's definitely faulty.")],
       "Same value, same power rating or higher (size is a clue: 1/4W is about 6mm long), same tolerance.", "Not worth it: resistors are cheap. Keep a kit instead."),
  part("pot", "Potentiometer or trimmer", "Passive",
       "A knob or slider (volume, tone), or a small trimmer adjusted with a screwdriver. Three pins: two ends and the wiper.",
       ["Dirty or worn track: crackles, jumps or dead spots as it turns.", "Open track: no change over part of its travel."],
       [test("Sweep the wiper", ["Power off. Meter on resistance between the middle pin and one end.", "Turn it slowly from one end to the other while watching the reading."],
             "The reading changes smoothly from near 0 to the full value.", "Jumps, flickers to OL, or doesn't change.", app=("meterie", "Meterie's graph shows a jump or dead spot as you turn it much better than the display can."))],
       "Same value and taper (a letter: A is log for volume, B is linear), same shaft and pin spacing. Try contact cleaner first.", "Yes, if they're smooth: knobs, pots and switches from old audio gear are useful."),
  part("ecap", "Electrolytic capacitor", "Passive",
       "A can with a stripe marking the negative leg, rated in µF and volts. Common in power supplies and audio.",
       ["Dries out with age and heat: capacitance drops and ESR rises, often with no visible sign.", "Bulges at the top, leaks brown crust, or vents. Can short."],
       [test("Look first", ["Tops should be flat, with no crust on the board under them."], "Flat top, clean base.", "Domed top, crust, or pushed up off the board."),
        test("ESR (in the circuit, power off)", ["Discharge it, then measure with an ESR meter or component tester."], "ESR at or below the chart for its µF and voltage (often under 1Ω for 100µF+).", "ESR several times the chart value, even if the µF reads fine."),
        test("Capacitance (out of the circuit)", ["Discharge it. Meter on capacitance, red probe to the + leg."], "Within about 20% of the marked value.", "Well below the marked value, or reading as a short.",
             measure=dict(label="Capacitance you measured", unit="µF", src="meterie", near=dict(label="Marked value", unit="µF", lo=0.8, hi=1.25)))],
       "Same µF, the same or higher voltage, 105°C, and low-ESR in power supplies. Fit it the right way round.", "No: old electrolytics are the part most likely to fail. Replace rather than reuse."),
  part("ccap", "Ceramic or film capacitor", "Passive",
       "Small beige or brown chips (surface-mount), or blue/yellow blobs and boxes. Used for decoupling and filtering.",
       ["Tiny ceramic chips crack and short, pulling a power rail down: the most common cause of a shorted rail.", "Film capacitors across the mains (X2) crack and burn with age."],
       [test("Find a shorted one", ["Power off. Resistance across it: a short reads near 0Ω, but so does everything else on that rail.", "If the rail is shorted, inject a little current into it (see Board level: find the short): the shorted capacitor gets warm."],
             "Not shorted, and stays cool when current is injected.", "Short, and the warm spot is on it.")],
       "Same capacitance and voltage or higher; for X2 or Y capacitors across the mains, only the same safety rating.", None),
  part("diode", "Diode", "Semiconductors",
       "A small cylinder with a band at one end (the cathode), or a surface-mount part marked with a line.",
       ["Shorts (reads close to 0V both ways), especially rectifiers after a surge.", "Goes open (OL both ways). Schottky diodes leak when hot."],
       [test("Diode test (in or out of the circuit, power off)", ["Meter on diode test. Red probe on the end without the band, black on the band.", "Then swap the probes."],
             "About 0.5–0.7V one way (0.15–0.4V for Schottky), OL the other way.", "Close to 0V both ways (shorted) or OL both ways (open).",
             measure=dict(label="Forward reading", unit="V", src="meterie", bands=[dict(lt=0.08, a=1, say="Shorted."), dict(lt=0.9, a=0, say="Normal forward drop.", ok=True), dict(a=1, say="Too high, or open.")]))],
       "Same type (rectifier, fast, Schottky, signal) with the same or higher current and voltage rating.", None),
  part("zener", "Zener diode", "Semiconductors",
       "Looks like a diode, but sets a voltage: marked with it (like 5V1, or a code).",
       ["Shorts after being overloaded; or the voltage drifts."],
       [test("Diode test", ["As for a diode: one way only should conduct."], "About 0.7V one way, OL the other (a meter can't show the zener voltage).", "Shorted both ways."),
        test("Zener voltage (powered, in the circuit)", ["Measure across it with the circuit running."], "Close to its marked voltage.", "Far from it, or 0V.", measure=dict(label="Voltage across it", unit="V", src="meterie", near=dict(label="Its marked voltage", unit="V", lo=0.93, hi=1.07)))],
       "Same voltage and power rating.", None),
  part("led", "LED", "Semiconductors",
       "Light-emitting diode: the longer leg (or the side without the flat) is +.",
       ["Goes open, dims, or one colour of an RGB LED dies. In strips, one dead LED can stop the rest."],
       [test("Diode test", ["Meter on diode test, red to + (the longer leg). Many meters light the LED faintly."], "It glows faintly, or reads about 1.6–3.2V one way.", "OL both ways, or no glow with a meter that lights others.")],
       "Same colour, size and forward voltage. Keep the resistor that sets its current.", "Yes: through-hole LEDs from old boards are fine to reuse after a quick test."),
  part("bridge", "Bridge rectifier", "Semiconductors",
       "Four diodes in one block, marked ~ ~ + −, or four separate diodes in a square near a power input.",
       ["One or more diodes short after a surge, blowing the fuse."],
       [test("Test all four diodes", ["Diode test from each ~ pin to + (red on ~), and from − to each ~ (red on −).", "Then reverse each one."], "About 0.5V forward on all four, OL in reverse.", "Any one near 0V both ways (shorted).")],
       "Same current and voltage rating or higher.", None),
  part("bjt", "Transistor (BJT)", "Semiconductors",
       "Three-legged black part (TO-92) or a surface-mount chip marked with a code: NPN or PNP, pins collector, base, emitter.",
       ["Collector-emitter short (the most common), or an open junction."],
       [test("Two diode tests (power off)", ["Find the pinout from its number (look it up; a component tester works it out for you).", "NPN: red on base, black on collector, then on emitter: both about 0.6V. Reverse: OL. PNP: the same with black on the base.", "Collector to emitter: OL both ways."],
             "Both base junctions about 0.5–0.7V one way and OL the other; collector-emitter OL.", "Any junction near 0V, or collector-emitter shorted.")],
       "Same part number, or one with the same type (NPN/PNP), pinout, and at least the same voltage, current and gain.", None),
  part("mosfet", "MOSFET", "Semiconductors",
       "A power transistor (TO-220, DPAK, or small SOT-23): pins gate, drain, source. Switches motors, heaters, LEDs and power supplies.",
       ["Drain-source short (stuck on), often with a cracked case or a burnt mark.", "Gate damaged by static or overvoltage."],
       [test("Diode test (out of the circuit is best)", ["Short all three legs briefly to discharge the gate.", "N-channel: red on source, black on drain: about 0.4–0.7V (the built-in diode). Reverse: OL.", "Gate to the other two: OL both ways.", "To check it switches: touch red to gate and black to source for a moment, then measure drain-source again: it should now conduct."],
             "Body diode reads normally, no short, gate OL, and it switches on when the gate is charged.", "Drain-source near 0V both ways, or the gate isn't OL.")],
       "Same part, or with the same or higher voltage and current, the same or lower on-resistance, a logic-level gate if it's driven from a microcontroller, and the same pinout.", None),
  part("reg", "Linear voltage regulator", "Power",
       "Three-legged part (78xx, LM1117, AMS1117, LM317) or small surface-mount chip turning a higher voltage into a fixed one.",
       ["Output missing or wrong, overheating, or shorted input to output. Pinouts differ between types: wrong orientation is common in builds."],
       [test("Powered: input, output, ground", ["Look up the pinout from its number (78xx is IN-GND-OUT; AMS1117 is GND-OUT-IN).", "Measure input to ground and output to ground with the circuit on."],
             "Input at least 1–2V above the output (less for LDOs), output at its rated voltage.", "Input fine but output missing or low: the regulator, or something shorting the output. Unplug the load to tell which.",
             measure=dict(label="Output voltage", unit="V", src="meterie", near=dict(label="Its rated output", unit="V", lo=0.95, hi=1.05)))],
       "Same output voltage, pinout and current, and the same or lower dropout. Check the output capacitor the datasheet asks for.", "Yes: 78xx and LM317 regulators are worth keeping."),
  part("buck", "Switching regulator (buck or boost)", "Power",
       "A chip with an inductor (a small coil, often a black block marked 4R7, 100…) and a diode nearby; turns one DC voltage into another efficiently.",
       ["No output (failed chip, open inductor, enable pin held off), or output shorted.", "Dried input or output capacitors make it whine, drop out or reset loads."],
       [test("Powered checks", ["Input voltage present at the chip.", "Enable (EN) pin high (above about 1.2V; check its datasheet).", "Output at its rated voltage."], "All present and right.", "Input present but no output, with enable high."),
        test("Switching node (with a scope)", ["Scope probe on the pad between the chip and the inductor, ground clip short."], "A square wave of about the input voltage, at a few hundred kHz to a few MHz.", "Flat line: the chip isn't switching.", app=("scopie", "Set the timebase to about 1µs/div and trigger on CH1's rising edge; turn on the Frequency and Peak-to-peak measurements."))],
       "Same chip; or a ready-made buck module of the right voltage and current as a bodge.", "Yes: buck and boost modules from dead gadgets are handy for projects."),
  part("crystal", "Crystal or resonator", "Timing",
       "A metal can (HC-49), or a small surface-mount box, marked with its frequency (8.000, 16.000, 32.768…). Usually next to the processor with two small capacitors.",
       ["Cracked by a drop; stops oscillating; or the circuit's capacitors fail. Without a clock, a processor does nothing."],
       [test("Oscillating? (powered, with a scope)", ["Use a ×10 probe (×1 probes load it enough to stop it).", "Probe the crystal's output pin (often marked XOUT, XTAL2 or OSC_OUT), not the input."],
             "A sine or rounded square wave at the marked frequency.", "Flat line, or a frequency far from the marking.",
             measure=dict(label="Frequency", unit="MHz", src="scopie:freq", near=dict(label="Marked frequency", unit="MHz", lo=0.995, hi=1.005)),
             app=("scopie", "×10 probe, timebase so a few cycles fill the screen (about 50ns/div for 16MHz), AC or DC coupling, turn on Frequency."))],
       "Same frequency and type; keep the two load capacitors.", None),
  part("ic", "Logic chip (74xx, 4000)", "Chips",
       "Black chips with 14 or 16 pins and numbers like 74HC04, 74LS138, CD4017.",
       ["Inputs or outputs damaged by static or overvoltage: an output stuck high or low."],
       [test("Powered: power, inputs, outputs", ["Find the pinout in its datasheet. Check VCC (usually pin 14 or 16) and GND (pin 7 or 8).", "Watch an input and the output it drives: does the output follow the logic it should?"],
             "Power present, and outputs follow their inputs as the datasheet says.", "An output stuck while its inputs change, or a chip that gets hot.",
             app=("probie", "Clip one channel to each input and the output, capture, and compare: Probie shows which signals change together."))],
       "Same part number; the family letters matter (HC, HCT, LS have different input levels).", "Yes, in sockets: old 74xx and 4000 chips are useful for projects and learning."),
  part("mcu", "Microcontroller or processor", "Chips",
       "The main chip that runs the device's program: an Atmel/Microchip, STM32, ESP, or a custom chip.",
       ["Rarely the first thing to fail. Usually it's missing power, clock or reset, or its program (in flash) is corrupted."],
       [test("Is it alive? (powered)", ["Every power pin at the right voltage (VCC, VDD, VDDA, AVCC).", "Clock running (see crystal).", "Reset pin at its 'run' level (usually high).", "Activity on its pins, or messages on its serial pins."],
             "Power, clock and reset all right, and pins toggling.", "All three right but no activity at all: a corrupt program or a failed chip.",
             app=("probie", "Capture the serial TX pin (Serial tab at 115200 first), or several I/O pins on the Logic tab to see whether it's doing anything."))],
       "Same chip, programmed with the device's firmware (from a backup or the maker): usually a job for the maker or a specialist.", "Development boards: yes. Chips soldered on old boards: rarely worth removing."),
  part("opamp", "Op-amp", "Chips",
       "8-pin chips like TL072, NE5532, LM358: in audio, sensor and filter circuits.",
       ["Output stuck at one supply rail, or noisy. Audio op-amps fail after a supply fault."],
       [test("Powered checks", ["Supply pins (often 8 and 4) at the expected voltages.", "With no signal, outputs (pins 1 and 7 on dual op-amps) usually sit in the middle: 0V on a ± supply, half the supply on a single one."],
             "Outputs at their resting level, and a test tone passes through.", "An output stuck at a supply rail.",
             app=("scopie", "Feed a test tone in and compare the input and output pins on CH1 and CH2."))],
       "Same part, or a pin-compatible op-amp that works on the same supply.", None),
  part("opto", "Optocoupler", "Chips",
       "A 4- or 6-pin chip (PC817 and similar) that passes a signal across an isolation gap with an LED and a sensor. In power supplies it carries the voltage feedback.",
       ["The LED inside wears out: a power supply's output drifts high or it cycles."],
       [test("Two tests (out of the circuit)", ["Diode test across the LED pins (1 and 2): about 1.1V one way.", "Light the LED through a 1kΩ resistor from 5V and measure the output pins (4 to 3) on resistance: it should drop sharply."],
             "LED conducts one way, and the output switches when the LED is lit.", "LED open, or the output doesn't change.")],
       "Same part (the current transfer ratio matters in power supplies).", None),
  part("fuse", "Fuse or polyfuse", "Protection",
       "Glass or ceramic tubes, small surface-mount fuses (marked F), or green/yellow polyfuses (resettable).",
       ["Blows because something after it drew too much current. Polyfuses go high-resistance and recover when cool."],
       [test("Continuity (power off)", ["Meter on continuity or low ohms across it."], "A beep: under 1Ω (polyfuses: a few ohms at most).", "OL: blown.", measure=dict(label="Resistance", unit="Ω", src="meterie", bands=[dict(lt=5, a=0, say="Good.", ok=True), dict(a=1, say="Blown or tripped.")]))],
       "Same current, voltage and speed (fast or slow-blow). Never bigger. Find why it blew before powering up.", None),
  part("tvs", "Protection diode, TVS or varistor", "Protection",
       "Small diodes or discs right at an input (USB, DC jack, mains) that clamp spikes.",
       ["Shorts after a spike to protect everything else: the input then reads as a dead short."],
       [test("Short check (power off)", ["Diode test or resistance across it. Lift one end if the reading could be something else."], "Not shorted (a TVS reads like a diode or OL).", "Close to 0Ω both ways.")],
       "Same type and voltage. A device often still works with it removed, but put a new one in before using it for real.", None),
  part("relay", "Relay", "Switching",
       "A box (often blue or black) that clicks: a coil moves a switch. Pins: coil, common, normally open (NO), normally closed (NC).",
       ["Coil open; contacts burnt, so they don't pass current even when they click; contacts welded shut."],
       [test("Coil and contacts", ["Coil resistance: tens to hundreds of ohms (the rating is on the case).", "Common to NC: should beep; common to NO: OL.", "Power the coil at its rated voltage: it clicks and the two swap. Check the contacts carry current, not just beep."],
             "Coil reads normally, it clicks, and the contacts swap cleanly.", "Coil OL, no click, or contacts that don't swap or stay closed.")],
       "Same coil voltage, the same or higher contact rating, same footprint.", "Yes: relays from old appliances and boards are handy, after a test."),
  part("switch", "Switch or button", "Switching",
       "Tactile buttons, slide and toggle switches, microswitches, keyboard switches.",
       ["Dirty or worn contacts: works sometimes, or bounces. Broken internal spring."],
       [test("Continuity while pressing", ["Power off. Continuity across the switch (small 4-leg buttons: try across the corners)."], "Beeps only while pressed, crisply.", "Never beeps, beeps all the time, or crackles.", app=("meterie", "Meterie's continuity mode with the graph shows a crackly contact as a ragged line."))],
       "Same size and action; contact cleaner first.", "Yes: switches, buttons and keyboard switches are worth keeping."),
  part("conn", "Connector, cable or ribbon", "Wiring",
       "Plugs and sockets, wire harnesses, and flat ribbon or flex cables.",
       ["Broken wire inside the sheath, cracked flex cable, corroded or loose contacts, cracked solder joints on the socket."],
       [test("End-to-end continuity", ["Power off. Continuity from each pin at one end to the matching pin at the other.", "Flex the cable while it beeps: the beep must not break.", "Check neighbouring pins aren't joined."], "Every wire beeps, steadily, and no two are joined.", "A wire that's OL or cuts in and out.", app=("meterie", "Meterie's continuity beep and graph catch a break that only opens when the cable bends."))],
       "Same connector type and pitch; ribbon cables by width, pitch and length.", "Yes: cables, connectors and ribbon cables are some of the most reusable parts."),
  part("speaker", "Speaker or buzzer", "Output",
       "Speakers (marked 4Ω or 8Ω), piezo buzzers and magnetic buzzers.",
       ["Open voice coil, torn cone, rubbing coil (buzzes)."],
       [test("Resistance (out of the circuit)", ["Unplug one wire. Meter on resistance."], "A speaker reads a bit under its marking (8Ω reads about 6–7.5Ω); a magnetic buzzer tens of ohms.", "OL (open coil) or near 0.", measure=dict(label="Resistance", unit="Ω", src="meterie", bands=[dict(lt=1, a=1, say="Shorted."), dict(lt=40, a=0, say="Normal for a speaker.", ok=True), dict(a=1, say="Open coil.")]))],
       "Same impedance (Ω), same or higher power, and a size that fits.", "Yes: speakers from old TVs and toys are fine for projects."),
  part("motor", "DC motor", "Output",
       "Small brushed motors in toys, fans, drives and tools.",
       ["Worn brushes, open winding, seized bearing, a shorted winding that draws too much current."],
       [test("Resistance and spin", ["Power off. Resistance across its terminals while turning it slowly by hand."], "A few ohms that stays fairly steady as it turns.", "OL at some positions (worn brushes or a dead segment) or 0Ω (shorted)."),
        test("Current draw (powered)", ["Run it from a bench supply at its voltage and watch the current with no load."], "Small and steady current that drops once it's up to speed.", "High current and little speed: a shorted winding or tight bearing.")],
       "Same voltage, size, shaft and speed.", "Yes: motors, gears and fans from old printers and toys are classic project parts."),
  part("liion", "Lithium-ion cell (18650 and others)", "Power",
       "Cylindrical cells (18650, 21700) in laptop and tool packs, or flat pouch cells.",
       ["Loses capacity with age; self-discharges when faulty; swells (pouch cells). A cell left below 2.5V can be damaged inside."],
       [test("Voltage, then capacity", ["Measure each cell: 3.0–4.2V is the normal range.", "Below 2.5V: don't charge it unless you have a charger with a recovery mode; below 1.5V, recycle it.", "Charge fully, wait a week, and measure again: a cell that's dropped more than 0.1V self-discharges and must not be used.", "A capacity tester shows what's left: below about 70% of its rating isn't worth keeping."],
             "3.0–4.2V, holds its charge, and decent capacity.", "Below 2.5V, a big drop over a week, swelling, or very low capacity.",
             measure=dict(label="Cell voltage", unit="V", src="meterie", bands=[dict(lt=1.5, a=1, say="Recycle it."), dict(lt=2.5, a=1, say="Over-discharged: only with a recovery charger, if at all."), dict(lt=4.25, a=0, say="Normal range.", ok=True), dict(a=1, say="Over-charged: don't use it.")]))],
       "Same chemistry and size, from a reputable maker; matched cells in packs.", "Yes, with care: good cells from laptop packs power many projects. Always use a protection board (BMS)."),
  part("display", "LCD or display module", "Output",
       "Character LCDs (16×2), small graphic LCDs and OLEDs, LED segment displays.",
       ["No backlight, missing segments or lines (connections), contrast set wrong, or a dead driver chip."],
       [test("Powered checks", ["Power pins at the right voltage.", "Character LCDs: turn the contrast trimmer slowly: a row of blocks should appear.", "Backlight pins (A and K): about 3–4V across a working backlight.", "Data pins: watch them while the device writes to the display."],
             "Blocks appear with contrast, backlight lights, data pins toggle.", "Nothing with power and contrast right: the module; data pins quiet: the device isn't writing.",
             app=("probie", "Capture the display's data or I2C/SPI pins to see whether the device is sending to it."))],
       "Same model; many character LCDs are interchangeable.", "Yes: displays from old gadgets are great for projects, after checking the pinout."),
]

# ---------------------------------------------------------------- board level (repairing old tech)
board = dict(sections=[
  dict(id="start", name="Start from the beginning", desc="Not sure where the fault is? Fixie goes round the board in order: look, power in, rails, clock and reset, then the rest.", start="b.vis.1"),
  dict(id="power", name="Power input and protection", desc="DC jack or input, fuse, reverse-polarity diode, protection parts.", start="b.in.1"),
  dict(id="smps", name="Mains power supply", desc="The part of the board that turns mains into low voltage. Cold tests first, then a safe powered test.", start="b.smps.1", **{"for": "mains"}),
  dict(id="rails", name="Regulators and power rails", desc="The fixed voltages (5V, 3.3V, 1.8V…) the chips run on.", start="b.reg.1"),
  dict(id="battery", name="Battery and charging", desc="Cells, protection board, charging chip.", start="b.bat.1", **{"for": "lipo|battery"}),
  dict(id="clock", name="Clock and reset", desc="Crystals and oscillators, reset and enable lines: without them nothing runs.", start="b.clk.1"),
  dict(id="cpu", name="Processor and memory", desc="Is the main chip running its program? Debug messages, activity, corrupted firmware.", start="b.cpu.1"),
  dict(id="bus", name="Data buses (I2C, SPI, UART)", desc="Chips talking to each other: decode the traffic and see who isn't answering.", start="b.bus.1"),
  dict(id="audio", name="Audio path", desc="Follow a sound from input to speaker and find where it stops.", start="b.aud.1"),
  dict(id="video", name="Video path", desc="Sync and picture signals: composite, VGA, and panels.", start="b.vid.1"),
  dict(id="outputs", name="Outputs and drivers", desc="Transistors, MOSFETs, relays and driver chips that switch motors, LEDs and loads.", start="b.out.1"),
  dict(id="inputs", name="Buttons, keyboards and sensors", desc="Switch matrices, pull-ups, encoders and sensors.", start="b.inp.1"),
  dict(id="wiring", name="Connectors, cables and tracks", desc="Breaks between parts: cables, flex, vias and corroded tracks.", start="b.con.1"),
])

# -- start: visual
step("b.vis.1", "Look over the whole board", "Many faults are visible: a few minutes with a light and magnifier saves hours of measuring.", [
    iff("hv", "Unplug it, wait a few minutes, and check the big capacitors are discharged before touching anything."),
    iff("lipo", "Disconnect the battery first."),
    "Photograph both sides of the board: useful later, and for asking for help.",
    "Look under a bright light, at an angle, with a magnifier: burnt or cracked parts, bulging capacitors, corrosion, cracked joints, damaged tracks, missing parts.",
    "Note the part numbers of the main chips: their datasheets show the pins to measure.",
  ], "What did you see?",
  [("Burnt, cracked or bulging parts", "r.burnt"), ("Corrosion", "w.3"), ("Cracked solder joints", "r.joint"), ("A loose or damaged cable", "b.con.1"), ("Nothing obvious", "b.in.1")],
  tools=["torch", "magnifier", "screwdrivers", "spudger"], open=True, signs=["caps", "burnt", "corrosion", "joint", "connector"])

# -- power input
step("b.in.1", "Measure the power coming in", "Start at the input: everything else depends on it.", [
    iff("mains", "For a mains board, measure the low-voltage outputs of its power supply instead (never the mains side). If they're missing, go to the Mains power supply section."),
    iff("!mains", "Power it as normal. Measure the voltage on the input connector, black probe on its ground."),
  ], "Is the input voltage there?",
  [("Yes", "b.in.2"), ("No, it's missing at the connector", "r.supply", "!mains"), ("No: a mains board's supply gives nothing", "b.smps.1", "mains")],
  tools=["meter"], live=True, apps=[("meterie", "DC volts. Watch the graph while you wiggle the plug: a dip shows a loose socket.")],
  measure=dict(label="Input voltage", unit="V", src="meterie", near=dict(label="What it should be", unit="V", lo=0.9, hi=1.15, ok=0, bad=1)))
step("b.in.2", "Follow it past the protection", "Just inside the input there's usually a fuse, a diode against reversed power, and a clamp. Any can stop or short the power.", [
    "Measure on the far side of each part in turn: the fuse (or polyfuse), then the diode.",
    "A diode drops about 0.3–0.7V; a fuse should drop almost nothing.",
  ], "Where does the voltage stop?",
  [("After the fuse: it's blown or tripped", "b.in.3"), ("After the diode: it's open", "r.diodeopen"), ("It gets through, but the voltage is pulled low", "b.reg.3"), ("It all gets through", "b.reg.1")],
  tools=["meter"], live=True, signs=["fuse"], apps=[("meterie", "DC volts; move the red probe along the path, keeping the black on ground.")])
step("b.in.3", "Why did the fuse blow?", "A fuse blows because something after it drew too much: find that before fitting a new one.", [
    "Power off. Measure resistance from the far side of the fuse to ground.",
    "Under about 10Ω: something is shorted. Check the protection diode or TVS and the input capacitors first: they're the commonest culprits.",
  ], "What did you find?",
  [("A shorted protection diode or TVS", "r.tvs"), ("A shorted capacitor", "r.short"), ("Shorted, but I can't find where", "b.reg.3"), ("Not shorted", "r.fuseonly")],
  tools=["meter"], open=True,
  measure=dict(label="Resistance to ground", unit="Ω", src="meterie", bands=[dict(lt=10, a=2, say="That's a short."), dict(a=3, say="Not shorted.", ok=True)]))
result("r.diodeopen", "An open protection diode", "The diode against reversed power has failed open.",
  ["Replace it with the same type, or one with the same or higher current and the same or lower voltage drop (a Schottky in, Schottky out)."], "Medium", parts=["Diode"], tools=["iron"])
result("r.tvs", "A shorted protection part", "A TVS diode or protection diode shorted to protect the rest from a spike or reversed power.",
  ["Remove it: if the short goes, that was it.", "Fit a new one of the same type and voltage before using it for real: it's what protects the rest.", "Fit a new fuse of the same rating."], "Medium", parts=["TVS or diode", "Fuse"], tools=["iron", "hotair"])
result("r.fuseonly", "Just a blown fuse", "Nothing is shorted now: probably a one-off surge, or a fault that cleared.",
  ["Fit the same rating of fuse, and power it from a current-limited supply (or dim bulb tester for mains) the first time.", "If it blows again, the fault comes on under power: measure the rails as it starts."], "Easy", parts=["Fuse"])

# -- mains SMPS (cold tests, then a protected powered test)
step("b.smps.1", "Cold tests on the power supply", "Most mains power supply faults show up with the power off: start here, it's safer.", [
    "Unplugged. Wait, then measure across the big capacitor: below about 5V before you touch anything. Discharge it with a resistor if not.",
    "Fuse: continuity. Bridge rectifier: four diode tests. The big switching MOSFET: drain-source not shorted.",
    "Look for the startup resistors (high values like 470k–2M feeding the controller chip): they drift open with age, a common cause of a supply that's completely dead.",
  ], "What did you find?",
  [("A blown fuse and a shorted MOSFET or rectifier", "r.psufuse"), ("A startup resistor open or high", "r.startres"), ("Bulging or leaking capacitors", "r.caps"), ("All fine cold", "b.smps.2")],
  tools=["meter", "discharge", "esr"], open=True, signs=["fuse", "caps", "burnt"])
step("b.smps.2", "Check the output side", "Faults on the low-voltage side make a supply shut itself down: it ticks, chirps or cycles.", [
    "Power off. Diode-test the output rectifier diodes (often a dual Schottky on a heatsink): a short is common.",
    "ESR-test the output capacitors: dried ones cause cycling and noise.",
    "Check each output for a short to ground.",
  ], "What did you find?",
  [("A shorted output diode", "r.schottky"), ("High-ESR output capacitors", "r.caps"), ("A shorted output (the fault is in what it feeds)", "b.reg.3"), ("All fine", "b.smps.3")],
  tools=["meter", "esr"], open=True, signs=["caps"])
step("b.smps.3", "A protected powered test", "With the cold tests passed, power it through a dim bulb tester so a remaining fault can't do damage.", [
    "Connect it through a dim bulb tester. Keep hands off the board; use clip leads on the low-voltage outputs only.",
    "Watch the bulb: a flash then dim is normal; bright and staying bright means a short is still there.",
    "Measure the outputs (low-voltage side only), and listen for ticking.",
  ], "What happens?",
  [("Outputs are right", "b.reg.1"), ("Bulb stays bright", "r.psufuse"), ("It ticks or the outputs pulse", "r.feedback"), ("Bulb dims but no output", "r.startres")],
  tools=["dimbulb", "meter"], live=True, apps=[("meterie", "DC volts on the low-voltage output, with the graph running: pulsing outputs show as a saw-tooth.")])
result("r.startres", "Startup resistors or controller supply", "The controller chip never gets enough voltage to start, usually because its high-value startup resistors have drifted open, or its supply capacitor has dried out.",
  ["Measure the startup resistors out of the circuit and replace any that read high or open.", "Replace the small electrolytic on the controller's supply pin (VCC/VDD): a very common fault.", "If it still won't start, replace the controller chip."], "Hard", parts=["Resistors", "Small electrolytic"], tools=["iron", "meter", "esr"],
  pro="This is on the mains side of the board: work only with it unplugged and the big capacitor discharged.")
result("r.schottky", "A shorted output diode", "The rectifier on the low-voltage side has failed short, so the supply shuts down.",
  ["Replace it with the same part (or the same voltage and current, Schottky for Schottky).", "Check the output capacitors too: they often failed first."], "Medium", parts=["Schottky diode"], tools=["iron"])
result("r.feedback", "The supply's feedback or output stage", "It starts, but cycles: the voltage feedback (optocoupler and TL431) or a dried output capacitor makes it shut down and restart.",
  ["Replace the output capacitors if not done.", "Test the optocoupler and the TL431 reference (often a 3-leg part near the opto).", "Check nothing on the outputs draws too much: disconnect the load and try again."], "Hard", parts=["Capacitors", "Optocoupler", "TL431"], tools=["esr", "meter", "iron"])

# -- regulators and rails
step("b.reg.1", "Measure every rail", "Each chip runs from a rail. A missing or low rail points straight at the fault.", [
    "Find the rails: labels like 5V, 3V3, 1V8, VCC, test points, or the regulators' outputs (look up their pinouts).",
    "Measure each with the device on, black probe on a solid ground.",
  ], "Are all the rails right?",
  [("All right", "b.clk.1"), ("One is missing (0V)", "b.reg.2"), ("One is low", "b.reg.3"), ("One is too high", "r.regulator")],
  tools=["meter", "probes"], live=True, apps=[("meterie", "DC volts. Use the min/max hold to catch a rail that dips at start-up.")],
  measure=dict(label="Rail voltage", unit="V", src="meterie", near=dict(label="What the rail should be", unit="V", lo=0.95, hi=1.05, ok=0, bad=2)))
step("b.reg.2", "Find why a rail is missing", "Missing means the regulator isn't making it, or it's switched off.", [
    "Measure the regulator's input: if that's missing, go back a step along the power path.",
    "Switching regulators: is the enable pin high? Some rails are switched on by the processor: they stay off if it isn't running.",
    "Power off and measure the rail to ground: a short makes the regulator shut down.",
  ], "What did you find?",
  [("No input to the regulator", "r.noinput"), ("Enable pin is low", "b.cpu.1"), ("The rail is shorted to ground", "b.reg.3"), ("Input and enable fine, no output", "r.regulator")],
  tools=["meter"], live=True)
step("b.reg.3", "Find the short", "A shorted part pulls a rail down and gets warm when power is pushed into it.", [
    "Power off. Measure the rail to ground: compare with a similar rail; a few ohms or less is a short.",
    "Feed the rail from a bench power supply at its voltage, current limit at 0.5–1A.",
    "Find the warm part: thermal camera, a film of isopropyl alcohol that dries first, or freeze spray that melts first. Small ceramic capacitors are the usual cause.",
  ], "Did you find it?",
  [("Yes, a part gets warm", "r.short"), ("Shorted but nothing warms", "r.shortpro"), ("It's not shorted after all", "r.regulator")],
  tools=["meter", "benchpsu", "hotspot"], open=True,
  measure=dict(label="Rail to ground", unit="Ω", src="meterie", bands=[dict(lt=5, a=0, say="Short."), dict(a=2, say="Not shorted.", ok=True)]))

# -- battery and charging
step("b.bat.1", "Measure the battery and its protection", "Battery packs have a protection board that cuts the output when a cell goes too low or too high.", [
    "Measure the pack at its output, then each cell (or cell group) directly.",
    "Single cells: 3.0–4.2V is normal. A pack that reads 0V at its output with healthy cells has its protection tripped.",
  ], "What did you find?",
  [("Cells fine, output 0V", "r.bmstrip"), ("A cell or group much lower than the rest", "r.cellbad"), ("All cells very low", "b.bat.2"), ("All fine", "b.bat.2")],
  tools=["meter"], apps=[("meterie", "DC volts on each cell in turn; note them in the step's notes.")],
  measure=dict(label="Lowest cell voltage", unit="V", src="meterie", bands=[dict(lt=2.5, a=1, say="Over-discharged."), dict(lt=4.25, a=3, say="Normal range.", ok=True), dict(a=1, say="Too high.")]))
step("b.bat.2", "Check the charging circuit", "If the cells are fine, the charge chip or its sensing is next.", [
    "With the charger connected, measure the charger's input on the board, then the battery voltage: it should slowly rise.",
    "Look up the charge chip: check its status pin (it often drives the charge LED) and its temperature-sense pin (a missing thermistor stops charging).",
  ], "What did you find?",
  [("Input present, battery doesn't rise", "r.chargeic"), ("Temperature sense open", "r.chargeic"), ("It charges", "r.gauge")],
  tools=["meter"], live=True)
result("r.bmstrip", "The pack's protection has tripped", "The protection board cut the output after the pack went too low, or saw a short.",
  ["Connect the proper charger: many wake up a tripped pack.", "If not, the board may need a brief connection to a charger at the pack's own voltage; only if you know Li-ion safety.", "If cells are below 2.5V, see Parts tests: lithium-ion cell."], "Medium", tools=["meter"],
  salvage="If the pack is done for, healthy cells can be reused in projects with a new protection board: test each one (Parts tests).")
result("r.cellbad", "One failing cell", "One cell or group has lost capacity or self-discharges, and drags the whole pack down.",
  ["Replace the pack, or rebuild it with matched new cells (a spot welder is needed: never solder directly to cells)."], "Hard", parts=["Matched cells", "Nickel strip"],
  salvage="The other cells may be good: test each one before reusing it.")

# -- clock and reset
step("b.clk.1", "Is the clock running?", "A processor does nothing without its clock.", [
    "Find the crystal or oscillator next to the main chip (a metal can or a small box marked with a frequency).",
    "Use a ×10 probe on the output side (XOUT, XTAL2, OSC_OUT): ×1 probes can stop it.",
  ], "What does the scope show?",
  [("A clean wave at the marked frequency", "b.clk.2"), ("Flat line", "r.crystal"), ("A wave, but the wrong frequency", "r.crystal")],
  tools=["scope"], live=True, apps=[("scopie", "×10 probe, timebase so a few cycles fill the screen, trigger on CH1, turn on Frequency.")],
  measure=dict(label="Frequency", unit="MHz", src="scopie:freq", near=dict(label="Marked frequency", unit="MHz", lo=0.995, hi=1.005, ok=0, bad=2)))
step("b.clk.2", "Is reset released?", "If the reset or enable line is held, the chip waits forever. If it pulses, the chip keeps restarting.", [
    "Find the reset pin (RESET, RST, NRST, often with a small capacitor and pull-up, or a 3-pin supervisor chip).",
    "Measure it with the device on: most chips run when it's high.",
    "Watch it over a few seconds with a scope or logic analyser: repeated pulses mean something keeps resetting it (a watchdog, or a supply dipping).",
  ], "What does reset do?",
  [("High and steady (running)", "b.cpu.1"), ("Held low", "r.reset"), ("Pulses every so often", "b.cpu.2")],
  tools=["meter", "scope", "logic"], live=True, apps=[("probie", "Logic tab: one channel on the reset pin, capture several seconds at a low sample rate (10kHz) to see any pulses."), ("scopie", "Slow timebase (500ms/div), roll mode, CH1 on reset.")])
result("r.crystal", "The clock isn't running", "The crystal, its two small capacitors, or the chip's oscillator has failed.",
  ["Reflow the crystal's joints (they crack in dropped devices).", "Replace the crystal with the same frequency and type, and check the two load capacitors.", "If it still doesn't run, the processor's oscillator may have failed."], "Medium", parts=["Crystal", "Load capacitors"], tools=["iron", "scope"])
result("r.reset", "Reset held low", "The reset supervisor or its pull-up holds the chip in reset, usually because it sees a low supply, or the supervisor has failed.",
  ["Check the supply the supervisor watches is at its full voltage.", "Check the pull-up resistor and the reset capacitor (a shorted capacitor holds reset low).", "Replace the supervisor chip if its input is fine but its output is low."], "Medium", parts=["Supervisor chip", "Capacitor"], tools=["meter", "iron"])

# -- processor and memory
step("b.cpu.1", "Is the processor doing anything?", "With power, clock and reset right, look for signs of life.", [
    "Find a serial TX pin or debug header (often 3–4 pins marked TX, RX, GND, or unlabeled pads near the chip).",
    "Connect a USB-serial adapter (TX of the board to RX of the adapter, grounds together) at the board's voltage, and power up.",
    "Or capture several of the chip's pins with a logic analyser: any switching shows it's running.",
  ], "What did you see?",
  [("Boot messages that stop with an error", "r.bootlog"), ("Boot messages that repeat", "b.cpu.2"), ("Pins switching, but no messages", "b.bus.1"), ("Nothing at all", "r.firmware")],
  tools=["serial", "logic"], live=True, apps=[("probie", "Serial tab: pick the adapter's port, try 115200 then 9600; Explain describes errors in plain words. Logic tab: 4–8 channels on the chip's pins, 1MHz sample rate.")])
step("b.cpu.2", "Find why it keeps restarting", "A boot loop is usually power dipping when something switches on, or a watchdog firing.", [
    "Watch the main rail with a scope (or Meterie's min/max) during start-up: a dip at the moment it restarts points at the supply.",
    "Read the boot messages: many chips print the reset reason (brownout, watchdog).",
  ], "What's the cause?",
  [("The rail dips", "r.weakrail"), ("A watchdog or crash message", "r.firmware"), ("Can't tell", "r.board")],
  tools=["scope", "serial"], live=True, apps=[("scopie", "CH1 on the main rail, single trigger on a falling edge set just below the rail voltage."), ("probie", "Serial tab: the reset reason is usually near the start of the boot messages.")])
result("r.bootlog", "The boot messages show the fault", "The processor runs, and is telling you where it stops.",
  ["Note the last lines before it stops: a missing part on a bus, a memory test failure, a storage error.", "Search for that message with the device's model: others often know it.", "Use Probie's Explain to turn the error into plain words."], "Medium", tools=["serial"])
result("r.firmware", "Corrupted or missing program", "Power, clock and reset are fine, but the program in flash memory is damaged, or the chip has failed.",
  ["Back up whatever is on the flash chip first with a programmer.", "Reflash with the maker's firmware or a backup from an identical device.", "If it still doesn't run, the processor (or the flash chip) has failed."], "Hard", tools=["programmer"],
  salvage="If it can't be revived, the rest of the board often isn't: but displays, motors, connectors and power parts usually are.")
result("r.weakrail", "The supply dips under load", "The rail sags when something starts, and the processor resets: dried capacitors or a weak regulator.",
  ["Replace electrolytic capacitors on that rail and at the regulator.", "Check the regulator's input holds up too.", "Add bulk capacitance near whatever draws the surge, if it's a design weakness."], "Medium", parts=["Capacitors"], tools=["esr", "iron"])

# -- buses
step("b.bus.1", "Capture the bus", "Chips talk over a few shared wires. Decoding the traffic shows which part isn't answering.", [
    "Identify the bus: I2C has SDA and SCL (two wires with pull-up resistors); SPI has CLK, MOSI, MISO and a chip-select per device; UART has TX and RX.",
    "Clip the logic analyser's ground to the board's ground and one channel to each wire.",
    "Capture during start-up, when the processor talks to everything.",
  ], "What does the decoder show?",
  [("I2C: an address gets NAK (no answer)", "b.bus.2"), ("I2C: lines stay low, or no clean edges", "b.bus.3"), ("SPI: data out, but nothing comes back", "r.spidev"), ("UART: garbage characters", "r.baud"), ("All the traffic looks normal", "b.out.1")],
  tools=["logic"], live=True, apps=[("probie", "Logic tab: add the I2C, SPI or UART decoder; sample rate at least 4× the bus speed (2MHz for 400kHz I2C). Explain summarises who answered.")])
step("b.bus.2", "Why doesn't that chip answer?", "A NAK means nobody answered at that address: the chip has no power, isn't connected, or has failed.", [
    "Find the chip with that address (its datasheet lists it).",
    "Check its power pins and ground.",
    "Check continuity from SDA and SCL on the processor to the same pins on the chip.",
  ], "What did you find?",
  [("No power at the chip", "r.chippower"), ("A break in SDA or SCL", "r.trace"), ("Power and wiring fine", "r.busdev")],
  tools=["meter", "logic"])
step("b.bus.3", "Check the pull-ups", "I2C lines are pulled high by resistors and pulled low by the chips. A shorted chip or missing pull-up stops the whole bus.", [
    "With the device on and idle, both SDA and SCL should read close to the supply voltage.",
    "If one is stuck low, power off and disconnect or lift chips one at a time until it frees: the last one removed is the culprit.",
    "Slow, rounded rising edges on a scope mean the pull-ups are too weak (or missing).",
  ], "What did you find?",
  [("A chip holds the line low", "r.busdev"), ("Pull-up resistor missing or open", "r.pullup"), ("Lines idle high, edges are clean", "r.board")],
  tools=["meter", "scope"], live=True, apps=[("scopie", "CH1 on SDA, CH2 on SCL, about 2µs/div: edges should rise quickly and square.")],
  measure=dict(label="SDA or SCL when idle", unit="V", src="meterie", near=dict(label="Supply voltage", unit="V", lo=0.85, hi=1.05, ok=2, bad=0)))
result("r.chippower", "The chip has no power", "Its supply pin is dead, so it can't answer.",
  ["Follow its supply pin back to the rail it comes from: a broken track, a missing ferrite bead or resistor in series, or a rail that's switched off.", "If the whole rail is missing, go to the Regulators and rails section."], "Medium", tools=["meter"])
result("r.nocontrol", "Nothing is telling it to switch", "The control signal never changes, so the fault is before the driver.",
  ["Check continuity from the processor's pin to the driver's input (resistors in between too).", "If the processor isn't running at all, go to the Processor and memory section."], "Medium", tools=["meter", "logic"])
result("r.noinput", "The regulator gets no input", "Power stops somewhere between the input and this regulator.",
  ["Follow its input pin back towards the power input, measuring at each part: a fuse, a diode, a switch, a ferrite or a broken track.", "The Power input and protection section goes through the usual parts."], "Medium", tools=["meter"])
result("r.busdev", "A chip on the bus has failed", "It doesn't answer, or holds the bus.",
  ["Reflow its joints first.", "Replace it with the same part (and the same address variant)."], "Hard", parts=["The chip"], tools=["hotair", "iron"])
result("r.pullup", "Missing or open pull-up", "Without its pull-up, the line can't return high.",
  ["Replace the resistor (usually 2.2k–10k to the supply)."], "Medium", parts=["Resistor"], tools=["iron"])
result("r.spidev", "An SPI chip doesn't answer", "Its chip-select, power, or the chip itself.",
  ["Check its chip-select line goes low when it's addressed, and its power.", "Check MISO continuity back to the processor.", "Replace the chip (memory chips: back up and reprogram)."], "Hard", tools=["logic", "programmer"])
result("r.baud", "Serial speed or level is wrong", "Garbled characters mean the speed doesn't match, or the voltage levels don't (RS-232 vs logic level).",
  ["Measure one bit's width in the capture: 1 ÷ width = the baud rate.", "RS-232 swings to ±5–12V; logic-level serial is 0–3.3V or 0–5V: a level converter (MAX232 type) may have failed."], "Medium", tools=["logic"])

# -- audio
step("b.aud.1", "Feed in a test tone", "Following a known signal through the circuit shows exactly where it stops.", [
    "Play a 1kHz tone into the input (a phone app works), at a low volume.",
    "Put the scope's ground on the board's audio ground.",
    "Start at the input socket, then follow the signal: each coupling capacitor, each op-amp or transistor stage, the volume control, the amplifier chip, then the speaker.",
  ], "Where does the tone stop?",
  [("At a capacitor (there on one side, not the other)", "r.couplecap"), ("At an op-amp or transistor stage", "b.aud.2"), ("At the amplifier chip", "b.aud.3"), ("It reaches the speaker terminals", "r.speaker")],
  tools=["scope", "signal"], live=True, apps=[("scopie", "CH1, about 500µs/div, AC coupling, Peak-to-peak on. Move the probe along the path.")],
  measure=dict(label="Signal at this point", unit="mV", src="scopie:vpp", bands=[dict(lt=5, a=1, say="Practically nothing: the signal stops before here."), dict(a=3, say="Signal present.", ok=True)]))
step("b.aud.2", "Check the stage's power and bias", "Every amplifying stage needs its supply, and sits at a resting voltage with no signal.", [
    "Measure the op-amp's supply pins, or the transistor's collector and emitter.",
    "With no signal, op-amp outputs rest mid-way (0V on a ± supply, half the supply on a single one).",
  ], "What did you find?",
  [("Supply missing", "b.reg.1"), ("Output stuck at a supply rail", "r.opamp"), ("Supply and bias right", "r.couplecap")],
  tools=["meter"], live=True)
step("b.aud.3", "Check the amplifier chip", "Amplifier chips need power, and many have a mute or standby pin the processor controls.", [
    "Supply pins at the right voltage.",
    "Find the MUTE, STBY or SD pin in the datasheet: is it in the 'play' state?",
    "With no signal, measure DC across the speaker terminals: more than about 50mV means a faulty output stage.",
  ], "What did you find?",
  [("Muted or in standby", "r.mute"), ("DC at the speaker", "r.amp"), ("Supply missing", "b.reg.1"), ("All right, still no sound", "r.amp")],
  tools=["meter"], live=True)
result("r.couplecap", "A failed coupling capacitor", "Small electrolytics between stages dry out and stop passing the signal: very common in old audio.",
  ["Replace the electrolytics in the signal path (a 'recap'), same µF and voltage or higher.", "Film capacitors are a good replacement for small values."], "Medium", parts=["Capacitors"], tools=["iron", "esr"])
result("r.opamp", "A failed op-amp stage", "The op-amp's output is stuck.",
  ["Replace the op-amp with the same or a pin-compatible part.", "Check its supply rails are both present first."], "Medium", parts=["Op-amp"], tools=["iron"])
result("r.mute", "The amplifier is muted", "The mute or standby pin keeps it silent: usually the processor holding it, or a fault the processor detected.",
  ["Follow the mute pin back to what drives it.", "Check the processor is running (Processor and memory).", "Briefly tying it to the 'play' level shows whether the amplifier works."], "Medium", tools=["meter", "logic"])

# -- video
step("b.vid.1", "Look for sync", "Every video signal carries timing pulses. If they're there, the video circuit is running.", [
    "Composite video (yellow socket): about 1V peak to peak, with sync pulses every 64µs (15.6–15.7kHz).",
    "VGA: separate horizontal sync (about 31.5kHz for 640×480) and vertical sync (50–75Hz), on pins 13 and 14.",
    "Probe the output socket first, then back towards the video chip.",
  ], "What do you see?",
  [("Sync is there at the right rate", "b.vid.2"), ("No sync at all", "b.vid.3"), ("Sync at an odd rate", "r.vidclock")],
  tools=["scope", "logic"], live=True, apps=[("scopie", "Composite: CH1, 20µs/div, trigger on the falling sync edge. VGA: 10µs/div on HSYNC."), ("probie", "VGA or digital sync: one channel each on HSYNC and VSYNC; the frequency shows in the measurement.")],
  measure=dict(label="Horizontal sync rate", unit="kHz", src="scopie:freq", bands=[dict(lt=14, a=2, say="Unusual rate."), dict(lt=17, a=0, say="TV rate (composite).", ok=True), dict(lt=30, a=2, say="Unusual rate."), dict(lt=82, a=0, say="A normal VGA rate.", ok=True), dict(a=2, say="Unusual rate.")]))
step("b.vid.2", "Check the picture levels", "With sync right, look at the picture part of the signal.", [
    "Composite: the picture rides above the sync, up to about 0.7V above black.",
    "VGA: each of R, G and B swings 0 to 0.7V; a missing colour channel means that colour's output stage.",
    "Check the 75Ω terminations and output resistors in the path.",
  ], "What did you find?",
  [("One colour or the picture part is missing", "r.vidout"), ("Levels right: the fault's in the screen or cable", "b.con.1")],
  tools=["scope"], live=True, apps=[("scopie", "CH1 on each colour pin in turn at 10µs/div; turn on Peak-to-peak.")])
step("b.vid.3", "Check the video chip's supply and clock", "No sync means the video chip isn't running.", [
    "Measure its supply pins, and look for its own clock (often a separate crystal: 14.318MHz, 17.734MHz, 25.175MHz…).",
    "Check it's out of reset or enabled.",
  ], "What did you find?",
  [("Supply missing", "b.reg.1"), ("Its clock is dead", "r.crystal"), ("Supply and clock fine", "r.vidout")],
  tools=["meter", "scope"], live=True)
result("r.vidclock", "Video timing is wrong", "The video chip's clock is off, or it's set to a mode the display can't show.",
  ["Check the video crystal's frequency.", "Some devices switch PAL/NTSC with a jumper or setting: check it."], "Medium", tools=["scope"])
result("r.vidout", "A failed video output stage", "The output transistor, buffer chip, or a resistor in the path has failed.",
  ["Follow the missing signal back to the first point it's present, and test the parts between (Parts tests).", "Replace the failed transistor, resistor or buffer."], "Medium", tools=["scope", "meter", "iron"])

# -- outputs and drivers
step("b.out.1", "Is the output being told to switch?", "Every driven output has a control signal. Check that first: it tells you whether the fault is before or after the driver.", [
    "Find the driver (a transistor, MOSFET, driver chip, or relay) for the thing that doesn't work.",
    "Watch its control input (transistor base, MOSFET gate, chip's IN pin) while you ask the device to switch it.",
  ], "Does the control signal change?",
  [("Yes, but the output doesn't follow", "b.out.2"), ("No, the control stays still", "r.nocontrol")],
  tools=["meter", "logic"], live=True, apps=[("probie", "One channel on the control pin; PWM outputs show their frequency and duty in the measurement.")])
step("b.out.2", "Test the driver", "The control works, so the driver or the load has failed.", [
    "Power off. Test the transistor or MOSFET (Parts tests); test the relay's coil and contacts.",
    "Check the flyback diode across motors and relay coils: when it fails, the driver fails with it.",
    "Test the load itself (motor, LED, lamp) on its own.",
  ], "What failed?",
  [("The transistor or MOSFET", "r.driver2"), ("The relay", "r.relay2"), ("The load itself", "r.motor"), ("Nothing found", "r.board")],
  tools=["meter", "esr"], open=True)
result("r.driver2", "A failed driver transistor or MOSFET", "It's shorted (output stuck on) or open (never switches).",
  ["Replace it with the same part or an equivalent (Parts tests: transistor, MOSFET).", "Replace the flyback diode too if there is one.", "Find why it failed: a shorted motor or a missing diode kills the new one."], "Medium", parts=["Transistor or MOSFET", "Flyback diode"], tools=["iron"])
result("r.relay2", "A failed relay", "The coil is open, or the contacts are burnt or welded.",
  ["Replace it with the same coil voltage and contact rating."], "Medium", parts=["Relay"], tools=["iron"], salvage="Working relays from scrapped boards are worth keeping: test them first (Parts tests).")

# -- inputs
step("b.inp.1", "Watch the inputs", "Buttons and keyboards are read by scanning; sensors give a voltage or a digital signal.", [
    "Single buttons: measure the input pin while pressing: it should swing between its two levels (often pulled high, pressed = low).",
    "Keyboard matrices: the processor pulses each row in turn and reads the columns. Capture a row and a column while holding a key.",
    "Analog sensors and pots: measure the voltage at the processor's input as you change them.",
  ], "What did you find?",
  [("The input never changes", "b.inp.2"), ("It changes but the device ignores it", "r.board"), ("Rows aren't being scanned", "b.cpu.1")],
  tools=["meter", "logic"], live=True, apps=[("probie", "Logic tab: one channel per row and column, 100kHz sample rate, capture while holding a key: the pressed key joins its row and column.")])
step("b.inp.2", "Test the switch and its path", "The switch, its pull-up, or the track to the processor.", [
    "Power off. Continuity across the switch while pressing.",
    "Check the pull-up resistor (often 10k).",
    "Continuity from the switch to the processor's pin.",
  ], "What failed?",
  [("The switch", "r.button"), ("The pull-up", "r.pullup"), ("A track", "r.trace")],
  tools=["meter"], open=True)

# -- wiring
step("b.con.1", "Beep out the connections", "Breaks in cables, flex, tracks and vias stop signals and power reaching where they're needed.", [
    "Power off. Continuity from each pin of the connector to where it goes on each board.",
    "Flex cables: look for creases and cracks; bend them gently while beeping.",
    "Vias (small plated holes) corrode through after a spill: beep from one side of the board to the other.",
  ], "What did you find?",
  [("A broken wire or flex", "r.cable"), ("A broken track or via", "r.trace"), ("Corroded contacts", "r.connector"), ("All connected", "r.board")],
  tools=["meter", "magnifier"], open=True, apps=[("meterie", "Continuity with the graph: a break that only opens when bent shows as a gap.")], signs=["connector", "trace", "corrosion"])

# ---------------------------------------------------------------- my build (your own projects)
build = dict(problems=[
  dict(id="power", name="Nothing happens, or it won't power up", desc="No LEDs, no activity, or the supply cuts out.", start="u.pw.1"),
  dict(id="hot", name="A part gets hot, or smelt of burning", desc="Something's warm, hot, or let out the magic smoke.", start="u.hot.1"),
  dict(id="code", name="The code doesn't run, or it keeps resetting", desc="Uploads, but does nothing, freezes, or restarts.", start="u.code.1", **{"for": "mcu"}),
  dict(id="upload", name="It won't upload, or the computer can't see it", desc="No port, upload errors.", start="up.1", **{"for": "mcu&usb|mcu&usbpow"}),
  dict(id="bus", name="An I2C, SPI or serial part doesn't respond", desc="Sensor not found, display blank, garbled serial.", start="u.bus.1", **{"for": "mcu"}),
  dict(id="wiring", name="I think something's wired wrong", desc="Check your build against your design, connection by connection.", start="u.wire.1"),
  dict(id="signal", name="A signal isn't right (PWM, clock, audio, video)", desc="Wrong frequency, wrong level, noisy, missing.", start="u.sig.1"),
  dict(id="noise", name="It glitches when a motor, relay or LEDs switch", desc="Resets, flicker or random readings when something turns on.", start="u.noise.1"),
  dict(id="motor", name="A motor, servo or stepper misbehaves", desc="Won't move, jitters, buzzes or skips.", start="u.mot.1", **{"for": "motor"}),
  dict(id="leds", name="Addressable LEDs (WS2812, NeoPixel) misbehave", desc="Wrong colours, flicker, only the first few work.", start="u.led.1", **{"for": "lights"}),
])

step("u.pw.1", "Measure the supply at your board", "Check the voltage where it's used, not just at the supply.", [
    "Power it on. Measure across the supply pins of your main board or chip (VCC/5V/3V3 to GND).",
    "On a breadboard, measure the power rails at both ends: long breadboards split their rails in the middle.",
  ], "What did you find?",
  [("The right voltage", "u.pw.3"), ("Nothing", "u.pw.2"), ("Much lower than it should be", "u.pw.4")],
  tools=["meter"], live=True, apps=[("meterie", "DC volts; the min/max hold catches a supply that drops when it starts.")],
  measure=dict(label="Supply at the board", unit="V", src="meterie", near=dict(label="What it should be", unit="V", lo=0.93, hi=1.07, ok=0, bad=2)))
step("u.pw.2", "Follow the power from the source", "Find where it stops.", [
    "Measure at the supply itself, then at each point along the way: connector, switch, fuse, regulator in and out.",
    "Breadboards: is the rail you're using actually connected? Wire jumper across the gap in split rails.",
    "Regulators: check the pinout. A 7805 is IN-GND-OUT; an AMS1117 is GND-OUT-IN: fitting one like the other is a very common mistake.",
  ], "Where does it stop?",
  [("At the supply: it gives nothing", "r.supply"), ("At a regulator: wrong pinout or way round", "r.pinout"), ("At a connector, switch or wire", "r.cable")],
  tools=["meter"], live=True)
step("u.pw.3", "Measure the current it draws", "Too much current means a short or a part in backwards; none at all means something's open.", [
    "Power it from a bench supply with the current limit set low (100–200mA to start), or put the meter in series on its current range.",
    "Compare with what you'd expect: a bare dev board draws 20–100mA.",
  ], "What does it draw?",
  [("Hits the current limit or far too much", "u.hot.1"), ("Almost nothing", "u.wire.1"), ("About what I'd expect", "u.code.1")],
  tools=["benchpsu", "meter"], live=True,
  measure=dict(label="Current drawn", unit="mA", src="meterie", bands=[dict(lt=1, a=1, say="Almost nothing: something isn't connected."), dict(lt=500, a=2, say="Plausible for a small build: compare with what you expected.", ok=True), dict(a=0, say="A lot: look for a short or a reversed part.")]))
step("u.pw.4", "Why is the voltage low?", "Low voltage at the board means the supply can't keep up, or something pulls it down.", [
    "Disconnect parts of the circuit one at a time (motors, LEDs, modules) and watch the voltage.",
    "Long, thin wires drop voltage under load: measure at both ends of each.",
    "USB ports give 500mA (USB 2) to 900mA (USB 3): motors and LED strips need their own supply.",
  ], "What did you find?",
  [("One part pulls it down", "u.hot.1"), ("Thin or long wires", "r.weaksupply"), ("The supply can't give enough current", "r.weaksupply")],
  tools=["meter"], live=True)
result("r.pinout", "A part fitted the wrong way, or the wrong pinout", "Regulators, transistors and chips with the same package often have different pinouts.",
  ["Check the datasheet's pinout against your wiring, with the part's flat face or notch matched to the drawing.", "Test the part after a power-up the wrong way: regulators and chips often survive, but not always (Parts tests)."], "Easy", tools=["meter"])

step("u.hot.1", "Find the hot part, carefully", "A hot part is either shorted, fitted backwards, or working too hard.", [
    "Power off at once if anything smells or smokes.",
    "Power it from a current-limited supply and feel for warmth with the back of a finger, or use a thermal camera.",
    "Check orientation of everything with a polarity: electrolytic capacitors, diodes, LEDs, regulators, chips (pin 1 dot or notch).",
  ], "What's hot?",
  [("A part fitted backwards", "r.pinout"), ("A regulator, but it's fitted right", "u.hot.2"), ("A chip or module", "u.wire.1"), ("A transistor or MOSFET driving a load", "u.hot.3")],
  tools=["benchpsu", "hotspot"], live=True)
step("u.hot.2", "Work out the regulator's heat", "A linear regulator burns off the difference as heat: (input − output) × current.", [
    "Measure its input and output voltage, and the current through it.",
    "Example: 12V in, 5V out at 200mA is (12 − 5) × 0.2 = 1.4W: too much for a small regulator without a heatsink (a bare TO-220 manages about 1W; SOT-223 well under 1W).",
  ], "Is it working too hard?",
  [("Yes, more than it can shed", "r.reghot"), ("No, but it still overheats", "u.wire.1")],
  tools=["meter"], live=True)
step("u.hot.3", "Check the driver is switching fully", "A transistor or MOSFET that's only half on gets hot.", [
    "Measure the gate (MOSFET) or base current (transistor) drive: a MOSFET driven from 3.3V needs a 'logic-level' type.",
    "Measure the voltage across it while on: more than a few tenths of a volt at real current means it isn't fully on.",
    "Motors and relays need a flyback diode across them.",
  ], "What did you find?",
  [("Not a logic-level MOSFET", "r.logiclevel"), ("No flyback diode", "r.flyback"), ("It's switching fully", "r.heatsink")],
  tools=["meter", "scope"], live=True)
result("r.reghot", "The regulator is working too hard", "Too big a drop at too much current.",
  ["Use a buck converter module instead: they waste very little.", "Or lower the input voltage, add a heatsink, or split the load."], "Easy", parts=["Buck converter module"])
result("r.logiclevel", "The MOSFET isn't fully switched on", "Standard MOSFETs need about 10V on the gate; 3.3V or 5V leaves them half on.",
  ["Use a logic-level MOSFET (look for RDS(on) quoted at VGS = 2.5V or 4.5V), or drive the gate through a transistor or driver."], "Easy", parts=["Logic-level MOSFET"])
result("r.flyback", "Missing flyback diode", "Motors and relay coils kick back a high voltage spike when switched off.",
  ["Fit a diode (1N4148 for small relays, 1N400x or a Schottky for motors) across the coil or motor, band to the positive side."], "Easy", parts=["Diode"])
result("r.heatsink", "It needs a heatsink", "It's switching properly but carries enough current to heat up.",
  ["Add a heatsink, or use a part with a lower on-resistance."], "Easy", parts=["Heatsink"])

step("u.code.1", "Prove the board runs at all", "Before your code, check the board can run the simplest thing.", [
    "Upload a blink sketch (or the board's 'hello world') and nothing else.",
    "Add a print at the very start of your program, and open the serial monitor at the matching speed.",
  ], "What happens?",
  [("Blink works, my code doesn't", "u.code.2"), ("It resets over and over", "u.code.3"), ("Even blink doesn't run", "up.3")],
  tools=["serial"], apps=[("probie", "Serial tab at your sketch's speed: prints show how far it gets; Explain describes crash messages.")])
step("u.code.2", "Find where your code stops", "Print messages narrow it down.", [
    "Add a print before and after each part of your setup: the last message tells you where it stops.",
    "Common stops: waiting for a sensor that isn't there (I2C begin), a library expecting different pins, a loop that never ends.",
  ], "Where does it stop?",
  [("At a sensor or module", "u.bus.1"), ("At pin setup or a library", "u.wire.1"), ("It crashes with a message", "r.crash")],
  tools=["serial"], apps=[("probie", "Serial tab: watch the prints as it starts.")])
step("u.code.3", "Find why it resets", "A reset loop is a power dip (brownout), a crash, or a watchdog.", [
    "Read the first lines it prints after each restart: ESP32 and ESP8266 print the reset reason (brownout, panic, watchdog).",
    "Watch the supply with a scope while it resets: a dip when a motor, radio or LEDs start is a brownout.",
  ], "What's the reason?",
  [("Brownout, or the supply dips", "u.noise.1"), ("A crash or panic", "r.crash"), ("Watchdog", "r.watchdog")],
  tools=["serial", "scope"], live=True, apps=[("scopie", "CH1 on the 3.3V or 5V pin, single trigger on a falling edge just below the rail."), ("probie", "Serial tab at 115200 (74880 for ESP8266 boot messages).")])
result("r.crash", "Your program crashes", "A crash message (panic, exception, stack trace) shows where.",
  ["Decode the backtrace (the ESP exception decoder turns addresses into line numbers).", "Common causes: an array written past its end, a null pointer, too little stack for a big local array, a library used before begin()."], "Medium")
result("r.watchdog", "The watchdog resets it", "A loop runs too long without letting the system do its own work.",
  ["Add a short delay or yield() in long loops; never wait forever for a sensor.", "On ESP32, avoid long blocking work in callbacks and interrupts."], "Easy")

step("u.bus.1", "Check the module is there", "Before the data, check the module answers at all.", [
    "I2C: run an I2C scanner sketch. It lists every address that answers.",
    "Check the module's power and ground, and that SDA and SCL go to the right pins for your board (they differ between boards).",
  ], "What did the scanner find?",
  [("Nothing at all", "u.bus.2"), ("A different address from my code", "r.i2caddr"), ("The right address, but readings are wrong", "u.bus.3"), ("It's SPI or serial, not I2C", "u.bus.4")],
  tools=["serial", "logic"], apps=[("probie", "Logic tab: SDA and SCL on two channels with the I2C decoder; NAK after the address means nothing answered.")])
step("u.bus.2", "Why doesn't anything answer?", "Usually wiring, pull-ups or voltage.", [
    "SDA and SCL swapped is the commonest mistake.",
    "I2C needs pull-up resistors (2.2k–10k to the supply). Many modules include them; a bare chip doesn't.",
    "Measure SDA and SCL with everything idle: both should sit near the supply voltage.",
    "A 5V module on a 3.3V board may need a level shifter.",
  ], "What did you find?",
  [("Swapped or wrong pins", "r.pinout"), ("No pull-ups", "r.pullup"), ("Voltage mismatch", "r.levels"), ("All right", "r.busdev")],
  tools=["meter", "logic"], live=True,
  measure=dict(label="SDA or SCL when idle", unit="V", src="meterie", near=dict(label="Supply voltage", unit="V", lo=0.85, hi=1.05, ok=3, bad=1)))
step("u.bus.3", "Look at the data", "It answers, so look at what it's saying.", [
    "Capture a reading with the logic analyser and the bus decoder: compare the bytes with the datasheet.",
    "Check your code reads the right registers, in the right order, and converts them as the datasheet says (byte order, signed values, scale).",
  ], "What did you find?",
  [("The data looks right: my conversion is wrong", "r.convert"), ("The data is wrong or noisy", "u.noise.1")],
  tools=["logic"], apps=[("probie", "Logic tab with the I2C or SPI decoder; Explain lists each read and write.")])
step("u.bus.4", "Check SPI or serial", "SPI needs the right mode and chip-select; serial needs matching speed and crossed wires.", [
    "SPI: check chip-select goes low around each transfer, and the clock mode (0–3) matches the datasheet.",
    "Serial: TX goes to RX and RX to TX; both sides at the same speed and voltage.",
  ], "What did you find?",
  [("Chip-select or mode wrong", "r.spidev"), ("TX and RX not crossed", "r.pinout"), ("Speed mismatch: garbled text", "r.baud")],
  tools=["logic"], apps=[("probie", "Logic tab: SPI decoder (set the mode) or UART decoder; it can measure the bit width to find the baud rate.")])
result("r.i2caddr", "The address in your code is wrong", "Many modules have two possible addresses, set by a pin or solder jumper.",
  ["Use the address the scanner found, or change the module's address pin to match your code."], "Easy")
result("r.levels", "Voltage levels don't match", "3.3V boards can be damaged by 5V signals, and may not read 3.3V as high on 5V parts.",
  ["Use a level shifter, or run the module at 3.3V if it supports it."], "Easy", parts=["Level shifter"])
result("r.convert", "The data is right, the conversion isn't", "The module sends good data; the code reads it wrongly.",
  ["Check byte order (high byte first?), signed values, and the scale factor in the datasheet.", "Try the library's example sketch to compare."], "Easy")

step("u.wire.1", "Check your build against your design", "Go through every connection, one at a time, with the power off.", [
    "With your schematic or Wirie design open, beep out each connection from part to part, ticking them off.",
    "Then check neighbouring pins on each chip and header aren't shorted together.",
    "Check pin numbering: board labels (D4, GPIO2) aren't the same as chip pin numbers.",
  ], "What did you find?",
  [("A missing connection", "r.cable"), ("A short between pins", "r.solderbridge"), ("A wire on the wrong pin", "r.pinout"), ("All correct", "u.sig.1")],
  tools=["meter"], open=True, apps=[("wirie", "Open your design; follow each net in the PCB tab and beep it out."), ("meterie", "Continuity beep, so you can keep your eyes on the board.")])
result("r.solderbridge", "A solder bridge or short", "Solder or a stray strand joins two pins.",
  ["Remove it with solder wick and flux, then check again with the meter.", "Look under a magnifier at every fine-pitch joint."], "Easy", tools=["wick", "flux", "magnifier"])

step("u.sig.1", "Look at the signal", "A scope or logic analyser shows exactly what the signal does, not what you hope it does.", [
    "Ground clip to your circuit's ground, probe on the signal.",
    "Check it's there, its voltage levels, its frequency or timing, and its shape (clean edges, no ringing or noise).",
  ], "What's wrong with it?",
  [("It's missing", "r.sigmissing"), ("Wrong frequency or timing", "u.sig.2"), ("Wrong voltage level", "r.levels"), ("Noisy or ringing", "u.noise.1"), ("Audio: distorted or clipped", "r.clipping")],
  tools=["scope", "logic"], live=True, apps=[("scopie", "Autoset, then turn on Frequency, Peak-to-peak and Duty cycle."), ("probie", "Digital signals: the measurement shows frequency and duty cycle.")],
  measure=dict(label="Frequency", unit="Hz", src="scopie:freq", near=dict(label="Frequency you expect", unit="Hz", lo=0.98, hi=1.02, bad=1)))
step("u.sig.2", "Check the timing", "Wrong frequency usually comes from settings, not hardware.", [
    "PWM: check the frequency and resolution settings in your code (servos want 50Hz, 1–2ms pulses).",
    "Clocks: the board's clock setting in your software (8MHz vs 16MHz boards run everything at the wrong speed).",
    "Video: sync timing must match the standard exactly.",
  ], "Fixed?", [("Yes", "r.settings"), ("No", "r.board")], tools=["logic"], apps=[("probie", "Measure the pulse width and frequency on the Logic tab.")])
result("r.sigmissing", "The signal isn't being made", "Nothing appears where the signal should be.",
  ["Check the pin in your code is set as an output, and is the pin you've wired (board labels differ from chip numbers).", "Probe right at the chip's pin: if it's there, follow the wire; if not, it's the code or pin setup.", "Beep out the wiring against your design (I think something's wired wrong)."], "Easy", tools=["scope", "logic"])
result("r.clipping", "The audio is clipping", "The signal is bigger than the stage can handle, or the stage isn't biased in the middle.",
  ["Lower the input level, or the gain.", "Check the stage's resting voltage is about half its supply (single-supply circuits).", "Check coupling capacitors are the right way round."], "Easy")

step("u.noise.1", "Watch the supply when the load switches", "Motors, relays, radios and LED strips draw surges that pull the supply down or put spikes on it.", [
    "Scope on your board's supply pin, trigger on a falling edge just below the supply, then switch the load on.",
    "Check each chip has a 100nF capacitor right next to its power pins, and there's a bigger one (100–1000µF) where the load's power enters.",
    "Motors and relays: is there a flyback diode? Do they share a thin ground wire with your logic?",
  ], "What did you find?",
  [("The supply dips or spikes", "r.decouple"), ("No flyback diode", "r.flyback"), ("Shared, thin ground", "r.ground"), ("The supply is clean", "r.pickup")],
  tools=["scope"], live=True, apps=[("scopie", "CH1 on the supply, single trigger, 1ms/div to start; zoom in on the dip.")],
  measure=dict(label="Lowest supply voltage", unit="V", src="scopie:vmin", near=dict(label="Normal supply voltage", unit="V", lo=0.9, hi=1.1, ok=3, bad=0)))
result("r.pickup", "Noise picked up on the signal wires", "The supply is clean, so interference gets in on the signals themselves.",
  ["Keep signal wires short and away from motor and mains wires; twist each signal with a ground wire.", "Add pull-up or pull-down resistors to inputs that float.", "Filter slow signals (an RC filter) and average sensor readings in code."], "Easy")
result("r.decouple", "Not enough decoupling", "The supply sags or spikes when the load switches.",
  ["Add 100nF ceramic capacitors at every chip's power pins.", "Add bulk capacitance (100–1000µF) where the load's power comes in.", "Give motors and LED strips their own supply, joined to your logic only at ground."], "Easy", parts=["Capacitors"])
result("r.ground", "Ground shared with a heavy load", "Load current through a shared ground wire shifts your logic's ground.",
  ["Run the load's ground straight back to the supply, and join logic ground at one point.", "Use thicker wire for load current."], "Easy")

step("u.mot.1", "Check the motor's supply and driver", "Motors need far more current than a board pin can give: always through a driver.", [
    "Measure the motor supply while it tries to move: does it sag?",
    "Driver boards: check enable pins and that its logic supply is there.",
    "Steppers: find the two coil pairs (each pair reads a few ohms; different pairs read OL) and wire each pair to one driver output.",
    "Servos: they need 50Hz pulses, 1–2ms wide, and their own 5–6V supply.",
  ], "What did you find?",
  [("Supply sags", "r.weaksupply"), ("Stepper coils wired wrong", "r.pinout"), ("Servo signal wrong", "u.sig.2"), ("All right", "u.mot.2")],
  tools=["meter", "logic"], live=True, apps=[("probie", "Servo signal: the measurement should show about 50Hz and 5–10% duty.")],
  measure=dict(label="Servo pulse frequency", unit="Hz", src="probie:freq", near=dict(label="Expected (50Hz for servos)", unit="Hz", lo=0.9, hi=1.1, ok=3, bad=2)))
step("u.mot.2", "Check the driver's settings", "Stepper drivers have a current limit; too low skips, too high overheats.", [
    "Set the driver's current limit (Vref) for your motor's rated current, from the driver's datasheet.",
    "Check microstepping jumpers match your code.",
    "Too fast acceleration makes steppers stall: start slower.",
  ], "Fixed?", [("Yes", "r.settings"), ("No", "r.stepper")], tools=["meter"])

step("u.led.1", "Check data level, resistor and power", "Addressable LEDs are fussy about their data signal and their supply.", [
    "Data voltage: a 3.3V board driving 5V LEDs often works badly. Use a level shifter (74AHCT125), or power the first LED a little lower.",
    "Fit a 300–500Ω resistor in the data line at the first LED, and a 1000µF capacitor across the supply.",
    "Join the LED supply's ground to the board's ground.",
    "Check the colour order (RGB vs GRB) and LED type in your code.",
  ], "What did you find?",
  [("Data level too low", "r.levels"), ("Colours swapped", "r.settings"), ("Only the first few work", "r.ledjoint"), ("Flicker or wrong colours at the far end", "r.voltdrop")],
  tools=["scope"], live=True, apps=[("scopie", "CH1 on the data line at the first LED, 1µs/div: pulses should be clean and reach near the LED supply voltage.")])

# ---------------------------------------------------------------- pictures and reference cards
# pics: drawings from web/pics.js; refs: reference cards from web/refs.js
stepPics = {
  "b.vis.1": (["ic", "mcu"], ["labels", "pin1"]),
  "b.in.1": (["meter-dcv", "dcjack"], []),
  "b.in.2": (["meter-dcv", "diode", "fuse"], ["labels"]),
  "b.in.3": (["inject-short", "ccap"], []),
  "b.smps.1": (["smps-map", "bridge", "meter-diode", "mosfet"], ["pinouts"]),
  "b.smps.2": (["smps-map", "meter-diode", "ecap"], []),
  "b.smps.3": (["bulb", "smps-map"], []),
  "b.reg.1": (["meter-dcv", "reg"], ["rails", "pinouts"]),
  "b.reg.2": (["buck", "wave-switching", "reg"], ["pinouts"]),
  "b.reg.3": (["inject-short", "ccap"], []),
  "b.bat.1": (["cell-pack", "liion"], []),
  "b.bat.2": (["cell-pack"], []),
  "b.clk.1": (["scope-probe", "wave-crystal", "crystal"], []),
  "b.clk.2": (["wave-reset", "mcu"], ["pin1"]),
  "b.cpu.1": (["serial-hookup", "wave-uart", "logic-i2c"], []),
  "b.cpu.2": (["wave-brownout", "serial-hookup"], []),
  "b.bus.1": (["logic-i2c", "wave-i2c", "wave-uart"], []),
  "b.bus.2": (["wave-i2c", "i2c-pullups"], []),
  "b.bus.3": (["wave-i2c-edges", "i2c-pullups"], []),
  "b.aud.1": (["signal-trace", "wave-tone"], []),
  "b.aud.2": (["opamp", "wave-tone"], ["pinouts"]),
  "b.aud.3": (["meter-dcv"], []),
  "b.vid.1": (["wave-composite", "wave-vga"], ["pinouts"]),
  "b.vid.2": (["wave-composite", "wave-vga"], []),
  "b.vid.3": (["crystal", "wave-crystal"], []),
  "b.out.1": (["driver-mosfet", "wave-pwm"], []),
  "b.out.2": (["mosfet", "relay", "meter-diode"], ["pinouts"]),
  "b.inp.1": (["key-matrix", "switch"], []),
  "b.inp.2": (["switch", "meter-continuity"], []),
  "b.con.1": (["meter-continuity", "usbpins"], []),
  "u.pw.1": (["meter-dcv", "breadboard"], ["rails"]),
  "u.pw.2": (["reg", "breadboard", "dcjack"], ["pinouts"]),
  "u.pw.3": (["meter-current"], ["ohm"]),
  "u.pw.4": (["meter-dcv", "usbpins"], ["ohm"]),
  "u.hot.1": (["ecap", "diode", "led", "ic"], ["pin1", "pinouts"]),
  "u.hot.2": (["reg"], ["ohm"]),
  "u.hot.3": (["driver-mosfet", "mosfet"], ["pinouts"]),
  "u.code.3": (["wave-brownout", "serial-hookup"], []),
  "u.bus.1": (["logic-i2c", "wave-i2c"], []),
  "u.bus.2": (["i2c-pullups", "wave-i2c-edges"], []),
  "u.bus.3": (["wave-i2c"], []),
  "u.bus.4": (["serial-hookup", "wave-uart"], []),
  "u.wire.1": (["meter-continuity", "breadboard"], ["pin1", "pinouts"]),
  "u.sig.1": (["wave-noise", "wave-pwm", "wave-tone"], []),
  "u.sig.2": (["wave-servo", "wave-pwm"], []),
  "u.noise.1": (["wave-brownout", "decoupling", "driver-mosfet"], []),
  "u.mot.1": (["wave-servo", "driver-mosfet"], []),
  "u.led.1": (["ws2812-hookup"], ["ohm"]),
  "up.1": (["usbpins"], []),
}
for sid, (pics, refs) in stepPics.items():
    assert sid in steps, sid
    if pics: steps[sid]["pics"] = pics
    if refs: steps[sid]["refs"] = refs
partPics = {"res": "res", "ecap": "ecap", "ccap": "ccap", "diode": "diode", "zener": "diode", "led": "led", "bridge": "bridge", "bjt": "bjt", "mosfet": "mosfet",
            "reg": "reg", "buck": "buck", "crystal": "crystal", "ic": "ic", "mcu": "mcu", "opamp": "opamp", "opto": "opto", "fuse": "fuse", "relay": "relay",
            "switch": "switch", "conn": "usbpins", "liion": "liion", "display": "lcd1602", "tvs": "ccap"}
testPics = {("res", 0): "meter-res", ("diode", 0): "meter-diode", ("led", 0): "meter-diode", ("bridge", 0): "meter-diode", ("bjt", 0): "meter-diode",
            ("mosfet", 0): "meter-diode", ("reg", 0): "meter-dcv", ("buck", 1): "wave-switching", ("crystal", 0): "wave-crystal", ("fuse", 0): "meter-continuity",
            ("switch", 0): "meter-continuity", ("conn", 0): "meter-continuity", ("ccap", 0): "inject-short", ("mcu", 0): "serial-hookup", ("opamp", 0): "wave-tone", ("motor", 1): "meter-current"}
partRefs = {"res": ["colour", "smd"], "ccap": ["cap"], "ecap": ["cap"], "bjt": ["pinouts"], "mosfet": ["pinouts"], "reg": ["pinouts"], "opamp": ["pinouts"], "opto": ["pinouts"], "ic": ["pin1", "pinouts"], "mcu": ["pin1"], "led": ["ohm"]}
for p in parts:
    if p["id"] in partPics: p["pic"] = partPics[p["id"]]
    if p["id"] in partRefs: p["refs"] = partRefs[p["id"]]
    for i, tt in enumerate(p["tests"]):
        if (p["id"], i) in testPics: tt["pic"] = testPics[(p["id"], i)]
assert all(k[0] in {p["id"] for p in parts} for k in testPics)
