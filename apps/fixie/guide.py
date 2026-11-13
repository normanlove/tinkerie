# Builds apps/fixie/web/guide.json, Fixie's repair guide: run "python3 apps/fixie/guide.py apps/fixie/web/guide.json"
# after changing it (fixie_test.go checks every path through the guide). Created by nrmluvdev.
import json, sys

# ---------------------------------------------------------------- describing the device
# Fixie doesn't use a list of products: you describe what you're fixing (how it's
# powered and what it has), and the steps follow from that. Categories are only
# shortcuts that fill in the usual answers, which you can change.
power = [
  ("mains", "Plugs straight into the wall", "Its power supply is inside it."),
  ("adapter", "A separate adapter or power brick", "A plug-in adapter with a round or special plug."),
  ("usbpow", "Powered by USB", "From a USB charger, computer or power bank."),
  ("battery", "Batteries you can swap", "AA, AAA, 9V, coin cells…"),
  ("lipo", "A built-in rechargeable battery", "Charges up; the battery is inside."),
]
power = [dict(id=i, name=n, hint=h) for i, n, h in power]
features = [
  ("screen", "A screen or display"), ("audio", "Speakers or sound"), ("buttons", "Buttons, keys or switches"),
  ("touch", "A touchscreen"), ("wireless", "Wi-Fi or Bluetooth"), ("usb", "Connects to a computer by USB"),
  ("motor", "A motor"), ("fan", "A cooling fan"), ("lights", "LED lighting (strip, string or lamp)"),
  ("mcu", "A hobby board you program yourself (Arduino, ESP32, Pico…)"),
  ("heater", "A low-voltage heater (3D printer hot end or bed)"),
  ("heatmains", "Heats things up from the mains (toaster, kettle, iron, heater…)"),
]
features = [dict(id=i, name=n) for i, n in features]
# tags worked out from the answers: all of "all", none of "none"
derived = [dict(tag="hv", all=["mains"]), dict(tag="laptop", all=["pc", "lipo"]), dict(tag="desktop", all=["pc", "mains"], none=["lipo"])]

C = [
  ("computer", "Computer or laptop", "pc", "mains", "screen audio usb wireless fan buttons",
   "Laptops usually open from the bottom: remove every screw (some hide under rubber feet or stickers), then prise the base off with a plastic pick, and unplug the battery before touching the board. Desktop PCs: the side panel comes off, but never open the power supply box itself."),
  ("phone", "Phone or tablet", "phone", "lipo", "screen audio touch buttons wireless usb",
   "Phones and tablets are glued shut. Warm the edge to about 80°C (a hair dryer or heat pad), lift with a suction cup and slide plastic picks round. Disconnect the battery connector first, before anything else."),
  ("console", "Games console or controller", "console", "mains", "usb wireless fan audio buttons",
   "Consoles and controllers use security screws (often Torx or tri-point), some under stickers or grips, then clips. Unplug the battery in controllers first; a console's built-in power supply holds mains voltage."),
  ("screen", "TV, monitor or display", "tv", "mains", "screen audio buttons",
   "Lay it face down on something soft. The back comes off with screws and clips. The power board holds mains voltage and big capacitors: unplug it and leave it a few minutes before touching anything."),
  ("audio", "Speaker, headphones, amp or radio", "sound", "lipo", "audio wireless buttons usb",
   "Small speakers and earbuds are usually glued or welded: prise the seam gently with a thin plastic pick and mind the battery wires. Amplifiers have big capacitors that stay charged after unplugging."),
  ("toy", "Toy or gadget", "toy", "battery", "buttons motor audio",
   "Usually small Phillips screws: check under the battery cover and stickers. Some snap apart with a plastic pick round the seam."),
  ("tool", "Cordless tool or battery pack", "powertool", "lipo", "motor buttons",
   "Screws round the body. Battery packs hold a lot of energy even when 'flat': never short the terminals, and keep them away from metal tools."),
  ("lights", "Lights and LEDs", "lamp", "adapter", "lights",
   "LED strips need no opening. Lamps with a built-in driver run from the mains: unplug first."),
  ("project", "Electronics project or board", "devboard", "usbpow", "mcu usb",
   "Nothing to open: unplug USB and anything else powering it before changing wiring."),
  ("input", "Keyboard, mouse or remote", "input", "usbpow", "buttons usb",
   "Keycaps pull off with a keycap puller. Mice have screws under the gliding pads; remotes snap apart at the seam."),
  ("machine", "3D printer or machine with motors", "printer", "mains", "motor fan screen usb heater",
   "Switch off and unplug at the wall. The power supply has mains terminals, often with no cover: keep fingers and tools away from them."),
  ("household", "Household gadget (fan, vacuum, shaver…)", "gadget", "mains", "motor buttons",
   "Screws, often hidden under feet or labels. Unplug before opening; mains motors are wired straight to the mains."),
  ("other", "Something else", "", "", "", None),
]
categories = [dict(id=i, name=n, kind=k, power=p.split(), has=h.split(), open=o) for i, n, k, p, h, o in C]

# ---------------------------------------------------------------- what Fixie won't help with
# Checked against whatever the person types as the device's name, and against
# their answers (anything that heats from the mains). There is no way round it.
danger = [
  dict(id="microwave", t="Microwave ovens", why="A microwave's high-voltage capacitor can hold a lethal charge long after it's unplugged, and its magnetron runs at over 2000 volts.",
       words=["microwave", "magnetron"]),
  dict(id="heat", t="Anything that heats from the mains", why="Ovens, toasters, kettles, heaters, irons and the like pass large mains currents through bare elements. A mistake can mean a fatal shock or a house fire.",
       words=["oven", "toaster", "toastie maker", "sandwich maker", "waffle maker", "cooker", "hob", "stove", "grill", "air fryer", "airfryer", "deep fryer", "fryer", "kettle",
              "coffee machine", "coffee maker", "espresso", "slow cooker", "rice cooker", "pressure cooker", "instant pot", "bread maker", "breadmaker", "food warmer", "hot plate", "hotplate",
              "induction hob", "induction cooker", "heater", "space heater", "radiator", "electric fire", "storage heater", "immersion", "electric shower", "boiler", "hair dryer", "hairdryer", "blow dryer",
              "hair straightener", "straighteners", "curling iron", "curling tongs", "hot brush", "steam iron", "clothes iron", "flat iron", "ironing", "garment steamer", "steamer",
              "electric blanket", "heated blanket", "heat gun", "soldering iron", "soldering station", "hot air station", "glue gun", "sous vide", "dehydrator", "vape", "e-cigarette", "ecig", "e-cig", "popcorn maker",
              "electric grill", "panini", "griddle", "heated blanket", "heated towel rail", "towel rail", "plate warmer", "bottle warmer"]),
  dict(id="white", t="Large household appliances", why="Washing machines, dishwashers, dryers, fridges and freezers combine mains power with water, heavy moving parts or pressurised refrigerant.",
       words=["washing machine", "washer", "dishwasher", "tumble dryer", "dryer", "fridge", "freezer", "refrigerator", "air conditioner", "air conditioning", "aircon", "heat pump", "dehumidifier", "ice maker", "wine cooler"]),
  dict(id="hv", t="High-voltage devices", why="These store or make thousands of volts: enough to kill, even when switched off.",
       words=["crt", "cathode ray", "tube tv", "tube television", "camera flash", "flashgun", "flash gun", "disposable camera", "neon", "bug zapper", "fly zapper", "insect killer",
              "taser", "stun gun", "plasma ball", "tesla coil", "ignition coil", "hid ballast", "xenon", "x-ray", "xray", "electric fence", "fence energiser", "ozone generator", "laser cutter", "laser engraver", "welder", "spot welder"]),
  dict(id="house", t="Anything wired into the house, or gas", why="Fixed wiring and gas work must be done by a qualified electrician or gas engineer, by law in many countries.",
       words=["fuse box", "consumer unit", "light switch", "wall socket", "mains socket", "plug socket", "house wiring", "ceiling light", "light fitting", "doorbell transformer", "extractor fan", "gas boiler", "gas hob", "gas fire", "gas cooker", "gas heater", "thermostat wiring"]),
  dict(id="bigbatt", t="Big batteries", why="E-bike, scooter and car batteries hold enough energy to start fires that are very hard to put out; car and mobility batteries can explode or leak acid.",
       words=["e-bike", "ebike", "electric bike", "e-scooter", "escooter", "electric scooter", "hoverboard", "electric car", "ev", "car battery", "mobility scooter", "power station", "powerwall",
              "home battery", "solar inverter", "inverter", "ups", "uninterruptible", "golf cart", "electric motorbike"]),
  dict(id="life", t="Medical and life-safety devices", why="If one of these fails after a home repair, someone could be hurt. They must be repaired or replaced by the maker.",
       words=["pacemaker", "defibrillator", "cpap", "oxygen concentrator", "insulin pump", "hearing aid", "medical", "nebuliser", "nebulizer", "smoke alarm", "smoke detector",
              "carbon monoxide", "co alarm", "co detector", "fire alarm", "stairlift", "hoist"]),
]
# old saved repairs (from before Fixie asked about the device) and what they become
legacy = {
  "phone": ["phone", "lipo", "screen audio touch buttons wireless usb"], "laptop": ["computer", "lipo adapter", "screen audio buttons touch usb wireless fan"],
  "desktop": ["computer", "mains", "usb audio fan"], "console": ["console", "mains", "usb wireless fan audio"], "controller": ["console", "lipo usb", "buttons wireless audio"],
  "remote": ["input", "battery", "buttons"], "tv": ["screen", "mains", "screen audio buttons lights"], "audio": ["audio", "mains", "audio buttons"],
  "earbuds": ["audio", "lipo usb", "audio wireless buttons"], "motorhome": ["household", "mains", "motor buttons"], "powertool": ["tool", "lipo", "motor buttons"],
  "lights": ["lights", "adapter", "lights"], "diy": ["project", "usbpow", "usb mcu"], "printer3d": ["machine", "mains", "motor heater fan screen usb"],
  "toy": ["toy", "battery", "buttons motor audio"], "keyboard": ["input", "usbpow", "usb buttons wireless"],
  "heater": ["other", "mains", "heatmains"], "washer": ["other", "mains", "motor heatmains"], "microwave": ["other", "mains", "heatmains"],
}
legacy = {k: dict(cat=c, power=p.split(), has=h.split()) for k, (c, p, h) in legacy.items()}

# ---------------------------------------------------------------- symptoms
S = [
  ("dead", "Completely dead", "No lights, no sound, nothing happens.", "", "dead.1"),
  ("shutsoff", "Turns on, then switches off or restarts", "Starts, then cuts out, keeps rebooting, or switches off under load.", "", "so.1"),
  ("charge", "Won't charge, or the battery runs out fast", "The charging light or icon doesn't come on, or it's flat in no time.", "lipo", "ch.1"),
  ("display", "No picture, or the screen is dim, lined or garbled", "It sounds or feels like it's on, but the screen is wrong.", "screen", "dp.1"),
  ("sound", "No sound, or it hums, crackles or distorts", "Speakers silent, one side out, or sound is noisy.", "audio", "snd.1"),
  ("buttons", "Buttons, keys or touch don't respond", "Something you press doesn't do anything, or does the wrong thing.", "buttons|touch", "bt.1"),
  ("motor", "A motor or fan won't spin, or is noisy", "Hums, grinds, or doesn't turn.", "motor|fan", "m.1"),
  ("heat", "Its heater doesn't heat, or won't stop", "A 3D printer hot end or bed stays cold, overheats, or shows a heating error.", "heater", "h.1"),
  ("connect", "Won't connect: USB, Wi-Fi or Bluetooth", "The computer doesn't see it, or it won't pair.", "usb|wireless", "cn.0"),
  ("upload", "The computer can't see the board, or code won't upload", "No COM port, upload errors, or it won't run your code.", "mcu&usb|mcu&usbpow", "up.1"),
  ("lights", "LEDs flicker, part is out, or colours are wrong", "Some LEDs dark, wrong colours, dim at one end.", "lights", "l.1"),
  ("hot", "Gets hot, smells of burning, or smoked", "Too hot to touch, a burning smell, smoke or a bang.", "", "hot.1"),
  ("water", "Got wet, or had a drink spilled on it", "Rain, a spill, dropped in water, or condensation.", "", "w.1"),
  ("intermittent", "Works sometimes, cuts out when moved", "Comes and goes, works when you hold the cable just so.", "", "int.1"),
  ("trips", "Trips the electricity or blows its fuse", "The house breaker or RCD trips, or the plug fuse blows.", "mains", "tr.1"),
]
symptoms = [dict(id=i, name=n, desc=d, **({"for": f} if f else {}), start=s) for i, n, d, f, s in S]

# ---------------------------------------------------------------- steps
steps = {}
def step(id, t, why, do, q, a, tools=(), good=None, bad=None, open=False, live=False, only=None, skip=None, measure=None, signs=()):
    s = dict(t=t, why=why, do=do, q=q, a=[dict(zip(("t", "go"), x[:2]), **({"only": x[2]} if len(x) > 2 else {})) for x in a])
    if tools: s["tools"] = list(tools)
    if good: s["good"] = good
    if bad: s["bad"] = bad
    if open: s["open"] = True
    if live: s["live"] = True
    if only: s["only"] = only; s["skip"] = skip
    if measure: s["measure"] = measure
    if signs: s["signs"] = list(signs)
    assert id not in steps, id
    steps[id] = s
def iff(cond, text): return {"if": cond, "t": text}

def result(id, t, cause, fix, level, parts=(), tools=(), next=None, pro=None):
    r = dict(result=True, t=t, cause=cause, fix=fix, level=level)
    if parts: r["parts"] = list(parts)
    if tools: r["tools"] = list(tools)
    if next: r["next"] = next
    if pro: r["pro"] = pro
    assert id not in steps, id
    steps[id] = r

# ---- completely dead
step("dead.1", "Rule out the power source", "Most 'dead' devices aren't: whatever powers them is the problem far more often than the device itself.", [
    iff("mains|adapter", "Plug it into a different wall socket that you know works (try a lamp in it first). Skip extension leads for now."),
    iff("adapter", "Try a different adapter with the same voltage, plug and polarity, if you have one."),
    iff("usbpow|usb", "Try a different USB charger and cable that you know work with another device."),
    iff("lipo&!usbpow&!usb&!adapter&!powertool", "Try a different charger that you know works."),
    iff("powertool", "Try a different battery pack that you know is charged, and charge this pack on its charger: the charger's light shows whether it's taking charge."),
    iff("usbpow", "Try a different USB port, ideally one on the back of a computer or a USB wall charger rather than a hub."),
    iff("battery", "Put in fresh batteries, checking each one faces the right way (+ to +)."),
    iff("lipo&!powertool", "Leave it on the charger for at least 30 minutes: a completely flat battery may show nothing at first."),
  ], "Does it come to life with power you know is good?",
  [("Yes, it works now", "r.supply"), ("No, still dead", "dead.2")],
  tools=["cable"], good="It powers up, so the device is fine and the supply was the problem.", bad="Still nothing: carry on.")

step("dead.2", "Check the batteries and their contacts", "Leaking batteries leave a crust that stops the current, and springs lose their grip.", [
    "Take the batteries out and look at the springs and contacts with a torch.",
    "Green, blue or white crust is corrosion from a leaking battery.",
    "Measure each battery with a multimeter on DC volts (the 20V range, or auto). A fresh AA or AAA reads about 1.5–1.6V; under 1.2V is flat.",
  ], "Were the contacts crusty, or the batteries flat?",
  [("Yes", "r.contacts"), ("No, they look clean and the batteries are good", "dead.4")],
  tools=["meter", "torch"], only="battery", skip="dead.pack", signs=["corrosion"],
  good="Shiny metal contacts, firm springs, and each cell above 1.3V.", bad="Crust on the contacts, a weak spring, or a cell below 1.2V.",
  measure=dict(label="Lowest battery voltage", unit="V", bands=[dict(lt=1.2, a=0, say="Flat: that battery needs replacing."), dict(lt=1.3, a=0, say="Nearly flat."), dict(a=1, say="Good.", ok=True)]))

step("dead.pack", "Measure the battery pack", "Tool packs have protection that switches the pack off when it's run right down or a cell is faulty.", [
    "Set the multimeter to DC volts and measure across the pack's main + and − terminals.",
    "An 18V (20V max) pack reads about 18–20.5V charged, and about 15–16V flat. 12V packs: about 10.5–12.5V. 36V packs: about 32–41V.",
    "A pack that reads 0V or very low has shut itself off: try it on its charger; some chargers can't wake a pack that's gone too low.",
  ], "What does the pack read?",
  [("About right", "dead.4"), ("Low, 0V, or it won't charge", "ch.4")],
  tools=["meter"], only="powertool", skip="dead.3",
  good="Within the range for its voltage.", bad="0V, or far below the 'flat' figure.")

step("dead.3", "Measure what the charger or adapter gives out", "A dead adapter looks exactly like a dead device. Measuring it takes a minute.", [
    "Read the label on the adapter: it says what it should give out, such as 'Output: 12V ⎓ 2A' or '5V 3A'.",
    iff("adapter", "Set the multimeter to DC volts. For a round barrel plug, put one probe inside the hole (centre pin) and the other on the outside sleeve. The small symbol on the label shows which is + (usually the centre)."),
    iff("usbpow|usb", "For USB, plug a USB power meter between the charger and the device: it shows the voltage and the current. Without one, use a USB breakout board."),
    "An adapter with nothing plugged in can read up to about 10% high: that's normal.",
  ], "Is the output close to what the label says?",
  [("Yes", "dead.3b"), ("No: nothing, much lower, or jumping about", "r.adapter")],
  tools=["meter", "usbmeter"], only="adapter|usbpow|lipo&usb", skip="dead.4",
  good="Within about 10% of the label (5V: 4.75–5.25V; 12V: about 11–13V).", bad="0V, far too low, or a reading that keeps changing.",
  measure=dict(label="Voltage you measured", unit="V", near=dict(label="Voltage on the label", lo=0.9, hi=1.15, ok=0, bad=1)))

step("dead.3b", "Watch whether it takes any current", "A device that draws current is getting power and doing something with it. One that draws nothing isn't connected.", [
    "With the USB power meter (or a bench power supply's current reading) in line, plug the device in and watch the current (A) for a minute.",
    iff("phone", "Then try a forced restart: on an iPhone press volume up, volume down, then hold the side button for 15 seconds; on most Android phones hold power and volume down for 20 seconds."),
    iff("laptop", "Then unplug the charger and hold the power button for 30 seconds, and try again. Some laptops have a small battery-reset hole underneath."),
    iff("lipo&buttons&!phone&!laptop", "Then hold the power button for 15–30 seconds; some gadgets also have a small reset hole."),
    iff("lipo&!buttons&!phone&!laptop", "Then look for a small reset hole, and press it with a paper clip."),
  ], "What does the meter show?",
  [("0.00 A: nothing at all", "dead.port"), ("It draws some current", "dead.flat", "lipo"), ("It draws some current", "dead.4", "!lipo")],
  tools=["usbmeter"], only="usbpow|usb&lipo", skip="dead.4",
  measure=dict(label="Current it draws", unit="A", bands=[dict(lt=0.02, a=0, say="Nothing: it isn't connecting."), dict(a=1, say="It's taking power.", ok=True)]))

step("dead.port", "Inspect and clean the port", "Lint and dirt build up in ports until the plug can't reach the contacts, and sockets work loose.", [
    "Shine a torch into the port and look with a magnifier or your phone's camera zoomed in.",
    "Scrape out lint with a wooden toothpick or a plastic pick. Never use metal: the pins short easily.",
    "Check the plug clicks firmly home and doesn't wobble. Look for bent or missing pins.",
  ], "What did you find?",
  [("Lint or dirt came out, and it works now", "r.lint"), ("The port wobbles or pins are damaged", "r.port"), ("The port is clean and solid", "dead.4")],
  tools=["torch", "magnifier", "spudger"], signs=["connector"])

step("dead.flat", "Give the battery time", "Lithium batteries that have gone below their safe voltage charge very slowly at first, and some devices show nothing until they've recovered.", [
    "Leave it on its own charger for an hour.",
    "Then do the forced restart again.",
  ], "Did it start?", [("Yes", "r.flat"), ("No", "dead.4")])

step("dead.4", "Check the switch, and any fuse or lead", "These are the simplest parts on the power path, and the most likely to fail.", [
    iff("mains", "Unplug it. In the UK, plugs have a fuse: open the plug (or the fuse drawer) and test the fuse with the multimeter on continuity (the beep setting). A beep means it's good; no beep means it's blown."),
    iff("mains", "Check the lead along its length, especially where it goes into the plug and the device: cuts, kinks, scorch marks or a loose cord grip."),
    iff("mains|adapter", "Unplugged, test each wire of the lead end to end on continuity, bending the lead as you go. The beep should stay steady."),
    "Make sure the on/off switch clicks and, if you can reach its contacts, that it beeps on continuity when switched on.",
  ], "What did you find?",
  [("A blown fuse", "dead.fuse", "mains"), ("A damaged lead", "r.cable", "mains|adapter"), ("A faulty switch", "r.button"), ("All fine", "dead.5")],
  tools=["meter", "screwdrivers"], signs=["fuse"],
  good="The fuse and each wire beep steadily (close to 0Ω).", bad="No beep (OL on the meter), or a beep that cuts in and out as you bend the lead.")

step("dead.fuse", "Replace the fuse once", "A fuse blows for a reason. If a new one blows straight away, something inside is shorted and must be found first.", [
    "Fit a new fuse of exactly the same rating (it's printed on it: 3A, 5A, 13A…). Never fit a bigger fuse, and never bridge it with wire or foil.",
    "Plug in and switch on.",
  ], "What happened?",
  [("It works", "r.fuseok"), ("The new fuse blew too", "tr.2")], tools=["fuses"], signs=["fuse"])

step("dead.5", "Open it up and look inside", "Faults that kill a device usually leave marks you can see: burnt parts, bulging capacitors or corrosion.", [
    "Take a photo before every step, and keep screws in order (an ice-cube tray or a magnetic mat works well).",
    iff("lipo", "Disconnect the battery first: lever its connector straight up with a plastic tool."),
    iff("hv", "Wait a few minutes after unplugging, then check the big capacitors are discharged (see Safety) before touching anything."),
    "Look over every part of the board under a bright light, with a magnifier. Sniff too: burnt parts have a sharp smell.",
  ], "What did you see?",
  [("Burnt or blackened parts", "r.burnt"), ("Bulging or leaking capacitors", "r.caps"), ("Swollen battery", "r.swollen", "lipo"), ("Green or white crust (corrosion)", "w.3"), ("A loose or unplugged cable", "r.connector"), ("Nothing obvious", "dead.6")],
  tools=["screwdrivers", "spudger", "torch", "magnifier", "esd"], open=True, signs=["caps", "burnt", "corrosion", "battery", "connector"])

step("dead.6", "Test the fuse inside", "Devices that run from the mains have a fuse on the power board, near where the mains comes in.", [
    "Unplugged. Find the fuse near the mains input: a glass tube or a small ceramic or black part marked F1 (or F with a number).",
    "Test it on continuity.",
  ], "Is the fuse inside good?",
  [("It beeps: good", "dead.7"), ("No beep: blown", "r.psufuse")],
  tools=["meter"], open=True, only="mains&!desktop", skip="dead.7", signs=["fuse"],
  good="A steady beep.", bad="No beep, or the glass is blackened inside.")

step("dead.7", "Measure the power rails", "Boards turn the input into a few fixed voltages ('rails'), such as 5V, 3.3V and 1.8V. A missing rail points straight at the fault.", [
    "Find test points or parts marked 5V, 3V3, VCC or VBAT, or the regulators (small 3- or 5-legged parts near the power input). A board diagram or service manual helps a lot.",
    "Put the black probe on ground: a metal shield, the metal shell of a socket, or a screw hole.",
    iff("mains", "Never use the mains side of a power supply as ground: that 'ground' is live."),
    iff("pc", "On a PC power supply you can check it on its own: link the green wire to a black one on the 24-pin plug and it starts. Yellow should be 12V, red 5V, orange 3.3V, purple 5V standby."),
    "Switch on and measure each rail with the red probe. Keep one hand away from the board.",
  ], "Are the voltages there?",
  [("All present and right", "dead.9"), ("One is missing or much too low", "dead.8"), ("Nothing at all from the supply", "r.psu")],
  tools=["meter", "probes"], open=True, live=True,
  good="Each rail within about 5% of its value.", bad="0V, or a rail that's well below what it should be (pulled down by a short).",
  measure=dict(label="Voltage on the rail", unit="V", near=dict(label="What the rail should be", lo=0.93, hi=1.07, ok=0, bad=1)))

step("dead.8", "Find the short", "A rail that's missing or pulled low is often shorted to ground by one failed part, which gets warm when power goes into it.", [
    "Unplug. Measure the resistance from the rail to ground. Under about 5Ω usually means a short (a busy rail with a big chip can read 10–100Ω: compare with a similar board if you can).",
    "Feed the rail from a bench power supply set to its voltage, with the current limit low (start at 0.5A). The shorted part warms up.",
    "Find the warm part: a thermal camera, a drop of isopropyl alcohol that evaporates first, freeze spray that melts first, or carefully with a fingertip.",
    "The usual suspects: tiny ceramic capacitors, protection diodes, and regulators.",
  ], "Did you find a part that heats up?",
  [("Yes", "r.short"), ("It's shorted, but nothing warms up", "r.shortpro"), ("It isn't shorted", "r.regulator")],
  tools=["meter", "benchpsu", "hotspot"], open=True,
  measure=dict(label="Resistance, rail to ground", unit="Ω", bands=[dict(lt=5, a=0, say="That's a short."), dict(a=2, say="Not shorted.", ok=True)]))

step("dead.9", "Check it's being switched on", "With power present, the next step is whether it's being told to start.", [
    iff("buttons", "Test the power button on continuity: it should beep only while pressed."),
    iff("mcu", "Development boards: press reset, and check the power LED. An on-board LED that blinks shows its code is running."),
    iff("!buttons&!mcu", "Check anything that switches it on (a switch, a sensor, a remote) is working."),
    iff("desktop", "On a desktop, short the two 'PWR SW' pins on the motherboard with a screwdriver tip to bypass the case button."),
    iff("desktop", "Take out all but one memory stick, unplug extra drives and cards, and clear the BIOS settings (the CMOS jumper or the coin battery out for a minute)."),
    iff("laptop", "Unplug the battery inside and try on the charger alone; take out and refit the memory if it isn't soldered in."),
    "Listen for beeps and watch for blinking lights: they're often a code for the fault.",
  ], "What happened?",
  [("The power switch doesn't close when pressed", "r.button", "buttons"), ("It started once the switch was bypassed", "r.button", "desktop"), ("It started after reseating the memory or cards", "r.ram", "pc"), ("It beeps or flashes a pattern", "r.codes"), ("Still nothing", "r.board")],
  tools=["meter"], open=True)

# ---- shuts off
step("so.1", "Notice exactly when it switches off", "When it cuts out says a lot about why.", [
    "Use it until it cuts out, a few times, and notice what's happening each time.",
  ], "When does it switch off?",
  [("On battery, but fine on the charger", "ch.4", "lipo"), ("Once the batteries aren't brand new", "dead.2", "battery"), ("After it's been on a while, or when warm", "so.2"), ("When it's working hard (busy, heavy load, loud)", "so.3"), ("Straight away, a second after starting", "so.4"), ("At random", "int.1")])

step("so.2", "Check the cooling", "Chips switch themselves off to protect themselves when they overheat. Dust and dried thermal paste are the usual reasons.", [
    "Check the fan spins when it's on, and air comes out of the vents.",
    "Look into the vents with a torch: grey felt across the heatsink fins blocks the air completely.",
    iff("pc", "Free tools like HWMonitor (Windows) show the temperatures. Over about 95°C for a processor is too hot."),
    iff("pc|console", "Thermal paste dries out after four or five years."),
    iff("printer", "3D printers: check the power supply's and mainboard's fans spin and their vents are clear; a hot mainboard's stepper drivers shut down."),
  ], "What did you find?",
  [("Dust or fluff clogging it", "r.cooling"), ("The fan doesn't spin", "m.1"), ("It's clean and the fan works", "so.3")],
  tools=["torch", "screwdrivers"], only="fan", skip="so.3", signs=["dust"])

step("so.3", "Check the power holds up under load", "A tired or undersized power source can't keep up when the device works hard.", [
    iff("usbpow|usb", "Measure the supply while it's working hard: a USB power meter in line shows the voltage dropping."),
    iff("battery|lipo&!usbpow&!usb", "Measure the battery while it's working hard, with the meter across its terminals."),
    iff("mains&!usbpow&!usb&!battery&!lipo", "Measure the power supply's low-voltage output on the board while it's working hard: never the mains side."),
    iff("adapter&!mains&!usbpow&!usb&!battery&!lipo", "Measure the power input on the board while it's working hard."),
    iff("laptop", "Check the adapter's watts match the one it came with. A lower-power adapter makes laptops shut off or slow down."),
    iff("pc", "Check the power supply's watts are enough for the graphics card (its maker says what's recommended)."),
  ], "Does the voltage drop a lot when it's working?",
  [("Yes, by more than about 10%", "r.weaksupply"), ("No, it holds steady", "so.4")],
  tools=["meter", "usbmeter"])

step("so.4", "Look for failing parts", "These two are the classic reasons for a start-and-stop loop.", [
    iff("lipo", "Disconnect the battery and look at it: it should be completely flat. Puffy, bent or rounded means swollen."),
    "Look at the round capacitors on the board: their tops should be flat. Domed or leaking tops mean they've failed.",
  ], "What did you see?",
  [("A swollen battery", "r.swollen", "lipo"), ("Bulging or leaking capacitors", "r.caps"), ("A part that's very hot", "dead.8"), ("None of these", "r.board")],
  tools=["screwdrivers", "spudger", "torch"], open=True, signs=["caps", "battery"])

# ---- charging
step("ch.1", "Try a known-good charger", "Cables fail far more often than devices. A cheap or worn cable may charge some devices and not others.", [
    iff("!powertool", "Try a different cable, and a different charger, that you know work."),
    iff("phone|sound", "If it can charge on a charging pad, try that: if the pad works and the cable doesn't, the port is the problem."),
    iff("powertool", "Try the pack on another charger, and another pack on this charger. A flashing fault light on the charger usually means the pack is too hot, too cold, or faulty."),
  ], "What happened?",
  [("It charges with another charger, cable or pack", "r.supply"), ("The pad works, the cable doesn't", "dead.port", "phone|sound"), ("Still won't charge", "ch.2")],
  tools=["cable"])

step("ch.2", "Measure the charging current", "How much current it takes tells you which part of the charging path has stopped.", [
    "Plug a USB power meter between the charger and the device and watch the current for a minute.",
    "Normal charging is usually 0.5–3A, slowing right down near full.",
  ], "What does it show?",
  [("0.00 A", "dead.port"), ("Below 0.3 A and stays there", "ch.3"), ("0.3 A or more, but the battery doesn't fill", "ch.4")],
  tools=["usbmeter"], only="usb|usbpow", skip="ch.4",
  measure=dict(label="Charging current", unit="A", bands=[dict(lt=0.02, a=0, say="Nothing is flowing."), dict(lt=0.3, a=1, say="Very low."), dict(a=2, say="It's charging at a normal rate.", ok=True)]))

step("ch.3", "Check the port's contacts", "A worn port makes poor contact, so it only charges slowly, or only at an angle.", [
    "Clean the port as in the port step (wooden toothpick, torch, no metal).",
    "Gently hold the plug at different angles while watching the current.",
  ], "What happened?",
  [("Lint came out and it charges now", "r.lint"), ("It charges only at an angle", "r.port"), ("No change", "ch.4")],
  tools=["torch", "spudger", "usbmeter"], signs=["connector"])

step("ch.4", "Check the battery's health", "Batteries wear out: after a few hundred charges they hold much less.", [
    iff("phone", "iPhone: Settings, Battery, Battery Health. Android: many phones show it under Settings, Battery or Device care, or use an app like AccuBattery."),
    iff("laptop", "Windows: run 'powercfg /batteryreport' in a command prompt and compare 'Design capacity' with 'Full charge capacity'. macOS: System Information, Power, shows the cycle count and condition."),
    iff("powertool", "Measure the pack's voltage: an 18V pack reads about 18–20.5V when charged. Packs that read very low (under about 14V) may have shut themselves off; one weak cell group is common."),
    iff("lipo", "Look at the battery itself: any puffiness at all means it's swollen."),
  ], "How is the battery?",
  [("Worn: much less than it should hold", "r.battery"), ("Swollen", "r.swollen"), ("Healthy", "ch.5")],
  tools=["meter"], signs=["battery"], only="lipo", skip="r.battery")

step("ch.5", "Check the charging circuit", "If the battery is fine and power reaches the device, the charging chip or its battery sensor is next.", [
    iff("!powertool", "Measure the voltage on the battery's connector while it's charging: a single lithium cell slowly rises towards 4.2V (some to 4.35V)."),
    iff("!powertool", "Many batteries have a third wire for a temperature sensor. If it's broken or missing, the charger refuses to charge."),
    iff("powertool", "Measure the pack's voltage while it's on the charger: it should rise steadily. Chargers read the pack's temperature and cells through the small contacts between the main terminals: clean them and check none are bent."),
  ], "Does the battery voltage rise while charging?",
  [("Yes, it rises", "r.gauge"), ("No", "r.chargeic")],
  tools=["meter"], open=True, live=True)

# ---- display
step("dp.1", "Is it the screen, or what feeds it?", "First find out whether the device is running and only the screen is out.", [
    iff("pc|console", "Connect another monitor or TV by HDMI or DisplayPort."),
    iff("printer", "3D printers: check the printer still moves and heats when you send it a job from a computer or SD card: if it does, only the screen is out."),
    iff("phone", "Ring it from another phone, or plug it into a computer: if it rings, vibrates or shows up, it's on."),
    iff("tv", "Press Input or Source and try every input, and try a different HDMI cable and port. Check the TV's menu appears with nothing plugged in."),
    "Listen for sounds and watch for lights that show it's running.",
  ], "What did you find?",
  [("It works on another monitor", "r.monitor", "desktop"), ("It works on another screen, or is clearly on", "dp.2", "!desktop"), ("Nothing on any screen", "dp.6", "pc"), ("Can't tell", "dp.2", "!desktop")])

step("dp.2", "Shine a torch at the screen", "LCD screens need a backlight. If only the backlight is out, the picture is still there, just in the dark.", [
    "With it on, shine a bright torch at the screen from close up, at an angle, in a dark room.",
  ], "What do you see?",
  [("A faint picture", "dp.3"), ("Nothing at all", "dp.4"), ("A picture with lines, blotches or wrong colours", "dp.5")],
  tools=["torch"],
  good="Nothing faint means the backlight isn't the (only) problem.", bad="A faint picture under the torch means the backlight has failed.")

step("dp.3", "Check the backlight", "Backlights fail on their own: one dead LED can put out a whole row, and many boards have a small backlight fuse.", [
    iff("tv", "TVs: the LEDs are strips behind the panel. Lift the panel out carefully (it's fragile) and test each LED with an LED tester, or a meter's diode test (it may only glow faintly)."),
    iff("laptop|phone", "Look for a small fuse near the display connector on the board, and check it on continuity."),
    iff("tv", "Burnt spots on the LEDs' lenses show the failed ones."),
    iff("!tv&!laptop&!phone", "Small screens have their backlight in the screen module itself: check its cable and the backlight supply pins (often marked LED+ and LED−, or BL) on the board."),
  ], "Did you find a dead LED or a blown fuse?",
  [("Yes", "r.backlight"), ("No", "r.backlightdriver")],
  tools=["meter", "ledtester", "spudger"], open=True, signs=["burnt"])

step("dp.4", "Reseat the display cable", "Display cables are thin and come loose after a knock.", [
    iff("lipo", "Disconnect the battery first."),
    iff("printer", "3D printer screens join the mainboard by one or two ribbon cables (often EXP1 and EXP2): check both are pushed fully home, the right way round and in the right sockets."),
    "Find the display connector, open its latch (a flip-up bar or a sliding lock), pull the cable out, look at the contacts and refit it straight and fully home.",
    "Look along the cable for creases, tears or burn marks.",
  ], "Did that help?",
  [("Yes", "r.connector"), ("No", "r.panel")],
  tools=["spudger", "magnifier"], open=True, signs=["connector"])

step("dp.5", "Read the pattern", "The shape of a screen fault tells you which part has failed.", [
    "Press gently along the edges of the screen and see if lines change.",
  ], "What does it look like?",
  [("Lines change when I press the edge", "r.bond"), ("Cracks, ink blots or spreading patches", "r.panel"), ("Half the screen or all the colours are wrong (TV)", "r.tcon", "tv"), ("Lines that don't change", "dp.4")])

step("dp.6", "Check it starts properly", "No picture anywhere means the computer isn't finishing its start-up.", [
    "Listen for beeps and watch the lights for a pattern.",
    iff("desktop", "Reseat the memory and the graphics card. Try one memory stick at a time, in the slot the motherboard's manual says."),
    iff("laptop", "Reseat the memory if it's not soldered in."),
    "Clear the BIOS settings (the CMOS jumper, or take the coin battery out for a minute).",
  ], "Does it start now?",
  [("Yes", "r.ram"), ("It beeps or flashes a pattern", "r.codes"), ("No", "dead.7")],
  tools=["screwdrivers", "esd"], open=True)

# ---- sound
step("snd.1", "Rule out settings and sources", "Most sound faults are a setting, or a cable.", [
    "Check the volume and mute, and that the right output is chosen (many devices switch outputs on their own).",
    iff("wireless", "Turn Bluetooth off on your other phones, tablets and computers nearby: it may be connected to one of them."),
    "Try a different source and cable.",
    "If it has a headphone socket, plug headphones in and out a few times: a stuck switch inside it mutes the speakers.",
  ], "Is there sound now?",
  [("Yes", "r.settings"), ("No", "snd.2")], tools=["cable"])

step("snd.2", "Test the speaker", "Speakers fail on their own: the fine coil wire breaks, or the cone tears.", [
    "If it has a headphone socket and headphones work but the speaker doesn't, it's the speaker or its wiring.",
    "Unplug or unsolder one speaker wire, then measure the speaker's resistance. A '4Ω' or '8Ω' speaker reads a bit less, about 3–7Ω.",
    "Push the cone very gently: it should move smoothly with no scraping.",
  ], "What did the speaker read?",
  [("Open (OL) or 0Ω", "r.speaker"), ("A few ohms: it looks fine", "snd.3")],
  tools=["meter"], open=True,
  measure=dict(label="Speaker resistance", unit="Ω", bands=[dict(lt=1, a=0, say="Shorted."), dict(lt=40, a=1, say="Normal for a speaker.", ok=True), dict(a=0, say="Open: the coil is broken.")]))

step("snd.3", "Listen to the fault", "Each noise has a usual cause.", [
    "Listen with the volume low and turn it up slowly. Move the knobs and switches.",
  ], "What do you hear?",
  [("Crackles when I move a knob or switch", "r.pot"), ("A steady hum or buzz", "r.hum"), ("Silence on one or both sides", "snd.4"), ("Distortion, worse when loud", "r.distort")])

step("snd.4", "Follow the sound through the circuit", "Find the last point where the sound is still there.", [
    "With it on and the volume halfway, touch the input of the amplifier chip or stage with a screwdriver tip held by its handle: a hum in the speaker means everything from there on works.",
    "With an oscilloscope (Scopie), follow a test tone from the input to the output and see where it stops.",
    "Measure the amplifier chip's supply pins.",
    iff("sound&mains", "Amplifiers have a protection relay that clicks on a few seconds after switching on. If it clicks off again, the output has a fault."),
  ], "What did you find?",
  [("A hum appears when I touch the amp's input", "r.preamp"), ("No hum even at the amplifier", "r.amp"), ("The relay clicks off", "r.amp", "sound&mains")],
  tools=["scope", "meter"], open=True, live=True)

# ---- buttons
step("bt.1", "Clean the button", "Dirt, sticky drinks and skin oil stop buttons making contact.", [
    "Switch it off and disconnect its power.",
    iff("battery", "Take out the batteries."),
    iff("input", "For a keyboard, pull off the keycap with a keycap puller."),
    "Clean round the button with isopropyl alcohol on a cotton bud, and blow out crumbs.",
    "Press the button 20–30 times to work the cleaner in, then let it dry for 10 minutes.",
    iff("phone|laptop|pc", "Many devices have a hardware test in their diagnostics or a test app: use it to see whether the button registers at all."),
  ], "Does it work now?", [("Yes", "r.clean"), ("No", "bt.2")], tools=["ipa", "brush"])

step("bt.2", "Check the touch controls", "Touch problems are often dirt, a cover over the surface, the power supply or the software.", [
    iff("screen", "Take off the case and screen protector, clean the screen, and restart."),
    iff("!screen", "Clean the touch surface, take off any cover or sticker over it, and restart."),
    iff("lipo", "Try it with the charger unplugged: a poor charger can make touch go wild."),
    iff("screen", "If the screen was replaced recently, cheap replacements often have poor touch."),
  ], "What happened?",
  [("Fine now", "r.clean"), ("It only misbehaves on the charger", "r.weaksupply", "lipo"), ("Touch is still dead or jumpy", "r.digitizer")],
  only="touch", skip="bt.3")

step("bt.3", "Test the switch with a meter", "Find out whether the switch itself closes.", [
    "With all power disconnected, put the meter on continuity across the switch's legs.",
    "It should beep only while you press it. Small square 4-leg switches have their legs joined in pairs, so try across corners.",
  ], "Does it beep when pressed, and not when released?",
  [("Yes", "bt.4"), ("No, or it beeps all the time", "r.button")],
  tools=["meter"], open=True)

step("bt.4", "Follow the track", "If the switch is fine, the connection from it is broken.", [
    "Follow the copper track or flex cable from the switch towards the chip, testing continuity along it.",
    "Look closely for a cracked track, corrosion or a torn flex cable.",
  ], "Did you find a break?",
  [("Yes", "r.trace"), ("No", "r.board")],
  tools=["meter", "magnifier"], open=True, signs=["trace", "corrosion"])

# ---- motor
step("m.1", "Turn it by hand", "A motor that can't turn freely can't start, and may hum or blow its fuse.", [
    "Unplug it. Turn the shaft, blades or wheel by hand.",
    "Feel for stiffness and grinding, and look for hair, string or fluff wound round the shaft.",
  ], "How does it turn?",
  [("Stiff, jammed or grinding", "r.bearing"), ("Freely", "m.2")], signs=["dust"])

step("m.2", "Check power reaches the motor", "Find out whether it's the motor, or what drives it.", [
    iff("fan", "Cooling fans: measure across the fan plug's red and black wires (usually 12V or 5V) while the device is on."),
    iff("motor&!mains&!powertool", "Low-voltage motors: measure across the motor's two terminals with its switch on."),
    iff("powertool", "Measure across the motor's two terminals with the trigger pressed."),
    iff("motor&mains&!printer", "Mains motors: only measure live if you're trained in mains work. Otherwise answer 'No power' and test the switch unplugged."),
    iff("printer", "3D printer motors are driven by the mainboard: move the axis from the printer's menu and feel whether the motor locks (holds firm) when the printer is on."),
  ], "Does power reach the motor?",
  [("Yes, but it doesn't turn", "m.3"), ("No power at the motor", "m.4")],
  tools=["meter", "probes"], open=True, live=True)

step("m.3", "Test the motor", "Brushes wear down, windings break, and fans need their start capacitor.", [
    iff("motor&!printer", "Unplugged, measure the resistance across the motor: a few ohms is normal. Open (OL) means a broken winding or worn brushes."),
    iff("printer", "Stepper motors: unplug the motor and measure across each pair of wires: two pairs read a few ohms (the two coils); open means a broken coil. Swap it with another axis's motor to see whether the fault moves."),
    iff("mains&motor&!printer|powertool", "Drills, blenders, vacuums and mixers have two carbon brushes, often behind round caps. Under about 5mm long, they're worn out."),
    iff("mains&motor&gadget", "Pedestal and ceiling fans that hum but don't spin need a new start capacitor (a small box marked µF)."),
    iff("fan", "For a cooling fan, try a spare fan of the same size and voltage."),
  ], "What did you find?",
  [("Worn brushes", "r.brushes", "mains&motor&!printer|powertool"), ("It hums but doesn't start (fan)", "r.runcap", "mains&motor&gadget"), ("A broken coil or winding", "r.motor", "motor"), ("The fan is dead", "r.motor", "fan"), ("The motor's fine: the fault moved with the driver", "r.stepper", "printer")],
  tools=["meter"], open=True)

step("m.4", "Check what feeds the motor", "Something before the motor has stopped passing power.", [
    iff("motor&!printer", "Unplugged, test the on/off switch and each speed setting on continuity."),
    iff("motor&mains&!printer", "Mains motors often have a thermal fuse or cut-out taped into the windings. It should beep."),
    iff("fan", "Cooling fans: check the fan's plug is fully on its header, and that the fan isn't turned off in the device's settings."),
    iff("pc&fan", "On a computer, check the fan isn't set to off or 0% in the BIOS fan settings."),
    iff("printer", "3D printers: check the motor's cable at both ends and the end-stop switches; a stepper driver that's failed or too hot stops its motor."),
    iff("powertool", "Trigger switches in cordless tools wear out: test the trigger at full press."),
  ], "What did you find?",
  [("A faulty switch", "r.button", "motor&!printer"), ("A blown thermal fuse", "r.thermalfuse", "motor&mains&!printer"), ("A loose plug or cable", "r.connector"), ("A failed stepper driver", "r.stepper", "printer"), ("All fine", "r.board")],
  tools=["meter"], open=True)

# ---- heating
step("h.1", "Check the obvious first", "Heaters are watched by safety checks that stop them when something isn't right.", [
    iff("printer", "3D printers show an error (thermal runaway, MINTEMP, MAXTEMP, heating failed). Note exactly what it says."),
    "Check the heater's and sensor's plugs are pushed fully home and not scorched.",
  ], "What's happening?",
  [("It shows an error message", "h.4", "printer"), ("It won't stop heating", "r.relay"), ("It stays cold", "h.2")], signs=["melted"])

step("h.2", "Test the heating element", "The element is just a wire with a known resistance, so a meter tells you if it's broken.", [
    "Switch off, unplug the heater's wires from the board, and measure across them.",
    "The expected resistance is the voltage squared divided by the power: a 24V 40W heater is 24×24÷40 ≈ 14Ω. Fixie works it out below.",
    "Also measure from each terminal to the metal block or frame: that must read open (OL).",
  ], "What does the element read?",
  [("Open (OL): broken", "r.element"), ("About what it should", "h.3"), ("A low reading to the metal block", "r.elementearth")],
  tools=["meter"], open=True,
  measure=dict(label="Element resistance", unit="Ω", calc="element"))

step("h.3", "Check power reaches the heater", "The heater is fine, so either the board isn't switching it on or the connection to it is bad.", [
    "Plug the heater back in and set it to heat.",
    "Measure across the heater's terminals on the board: it should show the supply voltage (often 12V or 24V) while heating.",
    "Look at the board's heater fuse (a small blade or surface-mount fuse near the power input) and check it on continuity, switched off.",
  ], "What did you find?",
  [("Full voltage, but the heater stays cold", "r.connector"), ("No voltage at the heater terminals", "r.heatdriver"), ("A blown fuse on the board", "r.heatdriver")],
  tools=["meter"], open=True, live=True)

step("h.4", "Read the error", "3D printer errors say exactly which part of the heating loop has failed.", [
    "MINTEMP means the temperature sensor reads impossibly cold: an open sensor wire. MAXTEMP means impossibly hot: a shorted sensor.",
    "A common sensor is a 100k thermistor: unplug it and measure about 100kΩ at room temperature (it falls as it warms).",
    "'Heating failed' or 'thermal runaway' means it didn't heat as expected: a loose sensor or a failing heater.",
    iff("fan", "A cooling fan pointed at the nozzle can cause it too: fit a fan duct or a silicone sock."),
    "Check the crimped connectors on the heater and sensor wires: they work loose and burn.",
  ], "Which error?",
  [("MINTEMP or MAXTEMP", "r.thermistor"), ("Heating failed or thermal runaway", "h.2")],
  tools=["meter"], signs=["melted"])

# ---- connect
step("cn.0", "Which connection won't work?", "USB and wireless faults have different causes.", [
    "Think about which connection is failing.",
  ], "Which is it?",
  [("USB to a computer", "cn.1"), ("Wi-Fi or Bluetooth", "cn.3")], only="usb&wireless", skip="cn.1")

step("cn.1", "Try another cable, port and computer", "Many USB cables only carry power, with no data wires inside.", [
    "Use a cable you know transfers data (one you've used to copy files with).",
    "Try a different port, directly on the computer rather than a hub, and if you can, a different computer.",
  ], "Does it connect now?",
  [("Yes", "r.datacable"), ("No", "cn.2")], tools=["cable"], only="usb|usbpow", skip="cn.3")

step("cn.2", "See whether the computer notices it at all", "Whether it appears, and how, points to the cause.", [
    "Windows: open Device Manager and plug it in. Watch for something new appearing, or an 'Unknown USB device' with a yellow mark.",
    "Mac: System Information, USB. Linux: run 'lsusb', or 'sudo dmesg -w' and plug it in.",
  ], "What happened?",
  [("Nothing appears at all", "dead.port"), ("It appears with an error (like 'device descriptor request failed')", "r.usbdata"), ("It appears, but the app can't use it", "r.driver")])

step("cn.3", "Wireless checks", "Pairing gets muddled, and antenna leads come loose.", [
    "Forget the device on both sides and pair again. Switch both off and on.",
    "Move closer, away from microwaves and other 2.4GHz gadgets.",
    iff("laptop|console|pc", "Inside, Wi-Fi cards have small antenna leads that snap onto the card: they pop off after a repair."),
  ], "What happened?",
  [("It connects now", "r.settings"), ("I found a loose antenna lead", "r.connector", "pc|console"), ("Still nothing", "r.board")],
  signs=["connector"])

# ---- upload
step("up.1", "Use a data cable, and look for the port", "The commonest cause by far is a charge-only USB cable.", [
    "Use a cable you know transfers data.",
    "Plug the board in and look in Device Manager (Windows), System Information (Mac) or 'ls /dev/tty*' (Linux) for a new COM or serial port.",
    "Flashie and Probie list the ports too.",
  ], "Does a port appear?",
  [("Yes", "up.3"), ("No", "up.2")], tools=["cable"])

step("up.2", "Check the driver and the board's power", "Cheaper boards use a USB-serial chip that needs a driver on some computers.", [
    "Look at the small chip near the USB socket: CH340, CP2102 or FT232 each have a free driver from their maker.",
    "Boards with USB built into the main chip (Pico, ESP32-S3, Pro Micro) need no driver.",
    "Check the power LED. If it's off, measure 5V and 3.3V on the board's pins.",
  ], "What did you find?",
  [("The driver fixed it", "r.driver"), ("The power LED is off", "dead.7"), ("The LED is on, but there's still no port", "up.4")],
  tools=["meter"])

step("up.3", "Put it into upload (bootloader) mode", "Boards need to be in a special mode to accept new code. Most switch on their own, but not always.", [
    "ESP32: hold BOOT, tap EN (or RST), let go of BOOT, then upload.",
    "Raspberry Pi Pico: hold BOOTSEL while plugging it in: a drive called RPI-RP2 appears. Drag a .uf2 file onto it, or use Flashie.",
    "Pro Micro and Leonardo: tap reset twice quickly, then upload within 8 seconds.",
    "Close anything else using the port (a serial monitor, Probie, Cura).",
    "Check the right board and port are chosen in your software.",
  ], "Does it upload now?",
  [("Yes", "r.bootmode"), ("No", "up.4")])

step("up.4", "Check what's wired to the pins", "Some pins decide how the chip starts. Something connected to them can stop uploads.", [
    "Unplug everything from the board except USB, and try again.",
    "ESP32 start-up pins: GPIO 0, 2, 5, 12 and 15. ESP8266: GPIO 0, 2 and 15. Arduino Uno: D0 and D1 are the USB serial pins.",
  ], "Does it work with nothing else connected?",
  [("Yes", "r.strap"), ("No", "r.mcu")])

# ---- lights
step("l.1", "Find where it goes wrong", "LED strips are a chain: a fault affects everything after it.", [
    "Switch on and look along the strip.",
  ], "What do you see?",
  [("Everything after one point is out or wrong", "r.ledjoint"), ("Flickers, or dim and yellowish at the far end", "r.voltdrop"), ("One colour is missing everywhere", "r.ledchannel"), ("Nothing lights at all", "dead.3")])

# ---- hot
step("hot.1", "Make it safe first", "Heat, smoke and burning smells are warnings. Stop using it before checking anything else.", [
    "Unplug it and don't switch it on again until you know why.",
    iff("lipo", "If a lithium battery is swelling, hissing or smoking, move it outside or onto a stone or metal surface away from anything that can burn. If it catches fire, call the fire service."),
  ], "What happened?",
  [("A swollen or hissing battery", "r.swollen", "lipo"), ("It smelt of burning, smoked or banged", "hot.2"), ("It's just hotter than it used to be", "so.2", "fan"), ("It's just hotter than it used to be", "so.4", "!fan")], signs=["battery"])

step("hot.2", "Find what burnt", "Burnt parts leave dark marks, split cases and a smell that tells you where to look.", [
    "Open it and look for discoloured board, split or charred parts, and melted plastic. Follow your nose.",
    iff("mains", "In mains power supplies, small box-shaped 'X' or 'Y' capacitors near the mains input often crack and smoke as they age."),
  ], "What did you find?",
  [("A cracked or burnt box capacitor near the mains input", "r.x2", "mains"), ("A melted plug or connector", "r.melt"), ("A burnt part on the board", "r.burnt"), ("Nothing I can see", "r.board")],
  tools=["torch", "magnifier"], open=True, signs=["burnt", "melted", "caps"])

# ---- water
step("w.1", "Switch it off and disconnect the power", "Water itself does little harm. Electricity flowing through it is what causes corrosion and shorts.", [
    "Switch it off and unplug it. Don't try to see if it still works.",
    iff("battery", "Take out the batteries."),
    "Skip the bag of rice: it doesn't dry the inside, and adds dust. Opening it up and cleaning it is what works.",
    "Sugary, salty or milky drinks corrode much faster than water, so act within hours.",
  ], "Done?", [("Done", "w.2")])

step("w.2", "Open it up", "Liquid trapped inside keeps corroding it: it needs opening up to dry and clean.", [
    iff("lipo", "Open it and unplug the battery connector before anything else."),
    "Open it up, and take out any boards you can reach easily, so both sides can dry.",
  ], "Done?", [("Done", "w.3")], tools=["screwdrivers", "spudger"], open=True)

step("w.3", "Clean off the corrosion", "Isopropyl alcohol shifts the residue and dries quickly.", [
    "Scrub every wet area with 99% isopropyl alcohol and a soft brush (a toothbrush is fine), especially connectors and the pins of chips.",
    "An ultrasonic cleaner does a better job.",
    iff("audio", "Take speakers, microphones and buzzers out before an ultrasonic clean."),
    "Let it dry for at least a day somewhere warm.",
    "Look under a magnifier for parts or tracks that have been eaten away.",
  ], "How does it look?",
  [("Clean, dry and intact", "w.4"), ("Parts or tracks are eaten away", "r.corrosion")],
  tools=["ipa", "brush", "magnifier"], open=True, signs=["corrosion"])

step("w.4", "Put it back together and test", "Test carefully: a short that's left can do more damage.", [
    "If you can, power it from a bench power supply with a low current limit first.",
    "Test everything it does, one thing at a time.",
  ], "How is it?",
  [("Everything works", "r.water"), ("Some things don't work", "r.corrosion"), ("It's dead", "dead.7")],
  tools=["benchpsu"])

# ---- intermittent
step("int.1", "Find what sets it off", "If you can make the fault happen when you want, you can find it.", [
    iff("mains|adapter|usbpow|usb", "With it on, wiggle the cable at each end."),
    iff("battery", "With it on, press gently on the battery cover and move the batteries in their springs."),
    "Tap gently round the case and board with a plastic tool.",
    "Warm it with a hair dryer, then cool it with freeze spray.",
  ], "What sets it off?",
  [("Moving the lead or plug", "int.2", "mains|adapter|usbpow|usb"), ("Moving it, or pressing on the battery cover", "r.contacts", "battery"), ("Tapping the board or case", "int.3"), ("Heat or cold", "int.4"), ("I can't make it happen", "int.5")],
  tools=["spudger", "hotspot"])

step("int.2", "Test the lead and socket", "Leads break inside next to the plugs, where they bend the most.", [
    "Unplugged, set the meter to continuity and bend the lead along its length while it beeps: if the beep cuts out, that's the break.",
    "Check the socket on the device: wobbly sockets have cracked solder joints underneath.",
  ], "Where's the fault?",
  [("The lead", "r.cable"), ("The socket", "r.joint")],
  tools=["meter"], signs=["joint", "connector"])

step("int.3", "Look for cracked solder joints", "Heat and knocks crack the joints on big, heavy or hot parts.", [
    "Look with a magnifier at joints of sockets, transformers, regulators, heatsinks and big capacitors.",
    "A crack is a thin dark ring round the pin, or a dull, grainy joint.",
  ], "What did you find?",
  [("Cracked joints", "r.joint"), ("A loose connector", "r.connector"), ("Nothing", "int.4")],
  tools=["magnifier", "torch"], open=True, signs=["joint", "connector"])

step("int.4", "Warm and freeze one part at a time", "Faulty parts often fail only when hot or cold.", [
    "With it running, spray freeze spray onto one part at a time, or warm one area with a hair dryer.",
    "When the fault comes and goes as you treat a single part, you've found it.",
  ], "Did you find the part?",
  [("Yes", "r.thermalpart"), ("No", "int.5")], tools=["hotspot"], open=True, live=True)

step("int.5", "Keep a log", "Faults that come and go need a record to show the pattern.", [
    "Write down each time it happens: how long it had been on, how warm, and what it was doing. Use the notes box below.",
    "Meterie can log the supply voltage over hours and show any dips.",
    iff("mcu|printer", "Probie can record what the board prints over its serial connection while it runs: errors often show there."),
  ], "What did the log show?",
  [("The supply dips when it fails", "r.weaksupply"), ("No pattern yet", "r.board")], tools=["meter"])

# ---- trips
step("tr.1", "Which protection is tripping?", "Overcurrent and earth leakage have different causes.", [
    "Look at your fuse box. An RCD (the switch with a 'T' test button) means current is leaking to earth. A breaker (MCB) or a blown plug fuse means too much current: a short.",
  ], "What trips?",
  [("The RCD", "tr.3"), ("A breaker or fuse, the moment it's switched on", "tr.2"), ("Only after it's been on a while, or when damp", "tr.3")])

step("tr.2", "Test for a short across the mains input", "A short inside the power supply makes the fuse blow straight away.", [
    "Unplugged, with its power switch on, measure resistance between the live and neutral pins of the plug.",
    iff("motor", "Motors read a few ohms to a few hundred ohms: that's normal."),
    "Electronics with a power supply should read high: thousands of ohms, often climbing as its capacitors fill.",
    "Inside, look for a split or burnt disc (a varistor or 'MOV') and burnt X-capacitors near the mains input.",
    "Test the bridge rectifier (four diodes, or a square block with ~ + − marks) and the switching transistor on the meter's diode test.",
  ], "What did you find?",
  [("A split varistor or burnt X-capacitor", "r.mov"), ("A shorted rectifier or transistor", "r.psufuse"), ("No short", "tr.3")],
  tools=["meter"], open=True, signs=["burnt", "caps"])

step("tr.3", "Test the insulation to earth", "Leakage to the earth wire usually comes from a failing part, a damaged lead or water inside.", [
    "Unplugged, with its switch on, measure from live and neutral (one at a time) to the earth pin.",
    "A multimeter should read open (OL) or many megohms. An insulation tester at 500V is the proper test.",
  ], "What does it read?",
  [("Low resistance to earth", "r.elementearth"), ("Open: fine", "r.pro")],
  tools=["meter", "insulation"],
  measure=dict(label="Resistance to earth", unit="MΩ", bands=[dict(lt=1, a=0, say="Too low: that's a leak to earth."), dict(a=1, say="Fine for a multimeter test.", ok=True)]))

# ---------------------------------------------------------------- results
result("r.supply", "The power source was the problem", "Whatever powers it was at fault, not the device itself.",
  ["Keep using the power that works.", iff("mains", "If a wall socket was at fault, have it checked by an electrician."), iff("adapter|usbpow|usb|lipo&!powertool", "Replace the faulty charger, adapter or lead with one that matches: same voltage, same plug, the same or more current, and from a reputable maker."), iff("battery", "Use fresh batteries of the right type."), iff("powertool", "Replace the faulty pack or charger with one made for the tool.")], "Easy", parts=["Charger or lead"])
result("r.contacts", "Flat batteries or corroded contacts", "Batteries had run down, or leaked and left a crust on the contacts.",
  ["Wear gloves and remove any leaking batteries.", "Dab the crust with vinegar or lemon juice (it neutralises the leak from ordinary alkaline batteries), then clean with isopropyl alcohol.", "Rub contacts bright with a fibreglass pen or fine sandpaper, and bend weak springs out a little.", "If the crust has crept up the wires, resolder or replace them."], "Easy", parts=["Fresh batteries"], tools=["ipa", "brush"])
result("r.adapter", "The adapter or charger has failed", "It isn't giving out the voltage on its label.",
  ["Replace it with one of the same voltage, the same plug size and polarity (the + and − symbol on the label), and at least the same current (A).", "Don't open sealed adapters: they hold mains voltage."], "Easy", parts=["Matching adapter"])
result("r.lint", "Lint or dirt in the port", "Packed fluff or dirt stopped the plug going all the way in.",
  ["Keep the port clean with a wooden toothpick now and then.", iff("phone|sound|input", "A dust plug helps if you carry it in a pocket or bag.")], "Easy")
result("r.port", "A worn or damaged charging port", "The contacts have worn or bent, or the port has come loose from the board.",
  [iff("phone", "On many phones and tablets the port is on a separate small board or flex cable that unplugs: order the part for your exact model."), "If it's soldered to the main board, it needs hot air, flux and care to replace (or a repair shop).", "A wobbly socket often just needs its joints resoldering."], "Medium", parts=["Charging port or charge board"], tools=["iron", "hotair", "flux"])
result("r.flat", "A deeply flat battery", "The battery had run down below its safe level and needed time to recover.",
  ["Check the battery's health (see the charging guide): if it's below about 80%, a new battery is worth it.", "Avoid leaving it flat for weeks: lithium batteries are damaged by it."], "Easy")
result("r.cable", "A broken lead", "The wire has broken inside, usually where it bends next to a plug.",
  ["Replace the lead if it unplugs.", "For a fixed lead, cut it off a little past the break and refit it, with the cord grip clamping the outer sheath. Or fit a new lead of the same rating.", iff("mains", "On mains leads, brown goes to live (L), blue to neutral (N), green-and-yellow to earth (E).")], "Medium", parts=["Lead", "Plug"], tools=["screwdrivers"])
result("r.fuseok", "The fuse had blown", "A single surge or a moment's overload.",
  ["Keep an eye on it. If the new fuse blows too, something is shorted inside.", "Always use the same rating."], "Easy", parts=["Fuse of the same rating"], next="tr.2")
result("r.burnt", "A burnt part", "A part has overheated and failed, often because something after it shorted.",
  ["Note its markings (resistor colour bands, a code printed on it) before removing it. A photo of a working board, or the service manual, helps if it's unreadable.", "Before fitting a new one, look for why it burnt: test what it feeds for a short.", "Scrape out any blackened board around it: charred board conducts."], "Hard", parts=["The burnt part, by its markings"], tools=["iron", "wick", "flux", "meter"], next="dead.8")
result("r.caps", "Failed capacitors", "Electrolytic capacitors dry out with age and heat, and then bulge or leak.",
  ["Replace each one with the same capacitance (µF), the same or higher voltage, and 105°C rating.", "In power supplies, use 'low-ESR' types.", "Fit them the right way round: the stripe on the side is the negative leg.", "Clean up any leaked electrolyte with isopropyl alcohol: it corrodes."], "Medium", parts=["Capacitors of matching value"], tools=["iron", "wick", "flux"])
result("r.connector", "A loose connector", "A cable had worked loose or wasn't fully seated.",
  ["Reseat it fully, with its latch closed.", "If the contacts look dull or green, clean them with isopropyl alcohol first."], "Easy", tools=["spudger"])
result("r.psufuse", "A short in the power supply", "The fuse inside blew because a part in the power supply shorted: commonly the bridge rectifier, the switching transistor (MOSFET) or the controller chip.",
  ["Don't just fit a new fuse: it will blow again, and can take more parts with it.", "On the meter's diode test, check the rectifier diodes and the transistor for a short (close to 0V both ways).", "Replace the shorted parts, then power it through a 'dim bulb tester' (a lamp in series) so a remaining fault can't do more damage.", "Often it's easier and safer to replace the whole power supply board, by the part number on its sticker."], "Hard", parts=["Power supply board, or the shorted parts"], pro="Mains power supplies are dangerous to work on. If you're not experienced, swap the whole board or take it to a repair shop.")
result("r.psu", "The power supply has failed", "The power supply isn't giving out anything.",
  [iff("desktop", "Replace the power supply unit; any good one with enough watts and the right plugs will do."),
   iff("mains&!desktop", "The power supply board can usually be bought by the part number on its sticker."),
   iff("adapter|usbpow|usb", "Power reaches the device but its own power circuit makes nothing: check the input socket, any fuse just inside it, and the first regulator."),
   iff("battery|lipo", "Check the battery's connector and the wires from it, then the power switch and the first regulator after it."),
   "If you're not confident working on power circuits, a repair shop can test it."], "Medium", parts=["Power supply or board"])
result("r.short", "A shorted part", "One part has failed short and was pulling the whole rail down.",
  ["Remove the warm part. Tiny ceramic capacitors fail like this often, and many boards still work without one of them for testing, but always fit a new one of the same value.", "Measure the rail again: the short should be gone."], "Hard", parts=["The shorted part"], tools=["hotair", "iron", "flux", "magnifier"])
result("r.shortpro", "A short under a chip", "The short is probably under a large chip, or inside the board.",
  ["Feed a little more current (carefully, 1–2A at the rail's voltage) to warm the faulty part enough to find.", "Otherwise this is a job for a board-level repair shop."], "Pro", pro="Board-level repair needs a microscope, hot air and experience.")
result("r.regulator", "A regulator isn't working", "The rail isn't shorted, so whatever makes it isn't doing its job.",
  ["Measure the regulator's input: if the input is there and the output isn't, the regulator has failed (or its enable pin isn't being switched on).", "Replace the regulator with the same part (its code is printed on it)."], "Hard", parts=["Regulator"], tools=["iron", "hotair"])
result("r.button", "A faulty switch", "The switch contacts have worn or got dirty.",
  ["Spray contact cleaner into it and work it many times.", "Replace it with the same type: small tactile switches are cheap and sold by size (6×6mm, 12×12mm…).", iff("mains", "Mains switches must be replaced with one rated for the same voltage and current.")], "Medium", parts=["Switch"], tools=["iron", "wick", "contact"])
result("r.codes", "It's reporting a fault code", "The device is running and telling you what's wrong.",
  ["Count the beeps or flashes and look up the pattern in the manual or the maker's support site.", iff("pc", "Memory and graphics faults are the commonest: reseat or swap the part.")], "Easy")
result("r.board", "A fault on the main board", "The simple checks are clear, so the fault is in the board's own circuits.",
  ["Search for the board's part number with the fault: others have often found the cause.", "With Scopie, check the crystal or clock is running. With Probie, watch the debug output many boards print at start-up.", "A donor board from a broken device of the same model is often the cheapest fix.", "Or take it to a board-level repair shop."], "Pro")
result("r.cooling", "Overheating", "Dust or dried thermal paste stops the heat getting out.",
  ["Blow dust out with compressed air, holding the fan still so it doesn't over-spin.", iff("pc|console", "Clean off old thermal paste with isopropyl alcohol and apply a pea-sized blob of new paste."), iff("pc|console", "Replace thermal pads with pads of the same thickness."), iff("!pc&!console", "Replace a fan that's slow or noisy with one of the same size and voltage.")], "Medium", parts=["Thermal paste", "Thermal pads"], tools=["screwdrivers", "ipa"])
result("r.weaksupply", "Not enough power", "Its power source can't keep up when the device works hard.",
  [iff("mains|adapter|usbpow|usb", "Use an adapter or power supply with at least the rated watts."), iff("battery|lipo", "Replace worn batteries."), "Check for thin or long cables that drop the voltage."], "Easy", parts=["Adapter, power supply or battery"])
result("r.swollen", "A swollen lithium battery", "Gas has built up inside: it's failing and a fire risk.",
  ["Stop charging and using it now.", "Remove it carefully: never puncture, bend or crush it. Soften the adhesive with isopropyl alcohol and use plastic tools only.", "Store it in a metal box or on a stone floor, away from anything that burns, until you can recycle it.", "Take it to a battery recycling point (most supermarkets and tips): never in household waste.", "Fit a new battery from a reputable source."], "Medium", parts=["Replacement battery"], tools=["spudger", "ipa"], pro="If it's very swollen or the case is cracked, many repair shops will remove it safely.")
result("r.battery", "A worn-out battery", "The battery has lost much of what it can hold.",
  [iff("!powertool", "Fit a new battery from a reputable maker."), iff("powertool", "Fit a new pack from the tool's maker or a reputable brand. Rebuilding packs needs a spot welder and care."), iff("phone|laptop", "Then charge it fully, and use it until it switches off once, so the device learns its new capacity.")], "Medium", parts=["Battery"], tools=["screwdrivers", "spudger"])
result("r.gauge", "It is charging: the battery reading is wrong", "The battery is healthy and takes charge, so the charge figure or light is what's wrong.",
  [iff("phone|laptop", "Charge it fully, use it until it switches off, then charge it fully again without stopping: this resets the battery reading."), iff("!phone&!laptop", "Charge it fully, then use it until it switches off, once or twice: many gadgets relearn their battery this way."), "If the light or reading still looks wrong, the battery's gauge chip or the charge light may have failed: a new battery is often the easiest fix."], "Easy")
result("r.chargeic", "The charging circuit has failed", "Power reaches the device but doesn't go into the battery.",
  ["Check the battery's temperature-sensor wire first.", "Next is the charging chip near the port: a board-level repair.", iff("laptop", "On laptops the charging circuit is sometimes on the separate DC-in board.")], "Hard", pro="Charging chips are tiny and under shields: a board-level repair shop can do this.")
result("r.backlight", "Dead backlight LEDs or fuse", "Backlight LEDs fail on their own, and one failed LED can put out a whole row; some boards have a backlight fuse.",
  [iff("tv", "Replace the whole set of strips with a kit for your model: the others will follow the first."), iff("laptop|phone", "Replace the fuse with the same value (bridge it only to test, then fit a proper one)."), iff("!tv&!laptop&!phone", "Replace the screen module: small screens have the backlight built in.")], "Medium", parts=["Backlight strips or fuse"], tools=["ledtester", "iron"])
result("r.backlightdriver", "The backlight driver has failed", "The circuit that powers the backlight isn't working.",
  [iff("tv", "The driver is often on the power board: replace the board by its part number."), iff("!tv", "Check the backlight supply on the board (often marked LED+ or BL); a missing supply is a board-level repair.")], "Hard", parts=["Power board"])
result("r.panel", "The screen itself has failed", "The screen panel is damaged.",
  ["Replace the screen with one for your exact model.", iff("phone", "For phones, buy a complete assembly with the frame if you can."), iff("tv", "TV panels usually cost too much to replace: compare with buying a new set.")], "Medium", parts=["Screen or display assembly"], tools=["spudger", "heatpad"])
result("r.digitizer", "The touch layer has failed", "The touch sensor itself has failed, or its cable.",
  ["Reseat the touch sensor's ribbon cable first.", iff("screen", "On most touchscreens the touch layer is part of the screen: replace the screen assembly for your exact model."), iff("!screen", "Replace the touch pad or sensor for your exact model.")], "Medium", parts=["Screen assembly"], tools=["spudger", "heatpad"])
result("r.bond", "The screen's ribbon or bonding has failed", "Lines that change when pressed come from where the ribbon cable is bonded to the glass.",
  ["This can't be fixed reliably at home. Replace the screen."], "Medium", parts=["Screen"])
result("r.tcon", "The T-con board has failed", "The T-con board turns the picture signal into what the panel needs.",
  ["T-con boards are cheap and swap with two ribbon cables: buy one by the part number printed on it.", "Check its fuse first (often a small surface-mount fuse near the power input)."], "Easy", parts=["T-con board"], tools=["screwdrivers"])
result("r.monitor", "The monitor is the problem", "The computer works on another monitor, so its own screen is fine.",
  ["Start a new repair for the monitor itself (the closest kind is 'TV, monitor or display').", "Check its cable and input setting first."], "Easy")
result("r.stepper", "A failed stepper driver", "The motor is fine, but the driver chip that powers it has failed or overheats.",
  ["On boards with plug-in drivers, swap it for a spare of the same type.", "On boards with soldered drivers, the mainboard usually needs replacing.", "Check the driver's cooling, and its current setting (Vref) if it's adjustable."], "Medium", parts=["Stepper driver or mainboard"])
result("r.ram", "Memory or a card wasn't seated", "A loose memory stick or card stops a computer starting.",
  ["Reseat it firmly until the clips close. If one stick causes the fault, replace it."], "Easy", tools=["esd"])
result("r.settings", "A setting or connection", "Nothing was broken.", ["Keep a note of the setting that fixed it."], "Easy")
result("r.speaker", "A broken speaker", "The speaker's coil has broken or shorted.",
  ["Replace the speaker with one of the same impedance (Ω), size and power, or more power."], "Easy", parts=["Speaker"], tools=["iron"])
result("r.pot", "A dirty volume knob or switch", "Dust and oxidation inside the control make it crackle.",
  ["Spray contact cleaner into the gap in the control's body, then turn it end to end 20 times.", "If it's worn through, replace it with the same value (printed on it, like 'B10K')."], "Easy", parts=["Contact cleaner", "Potentiometer"], tools=["contact"])
result("r.hum", "Hum", "Usually old smoothing capacitors in the power supply, or a ground loop between connected equipment.",
  ["Unplug other equipment and try again: if the hum stops, it's a ground loop (a ground-loop isolator fixes it).", "If it hums on its own, replace the big smoothing capacitors in the power supply."], "Medium", parts=["Capacitors"], tools=["iron"])
result("r.distort", "Distortion", "A damaged speaker cone, or the amplifier running out of power.",
  ["Check the speaker cone for tears and a scraping coil.", "Check the amplifier's supply voltage holds steady when loud."], "Medium", parts=["Speaker"])
result("r.preamp", "The fault is before the amplifier", "The amplifier works, so the signal is lost earlier: input socket, volume control, source switch or preamp.",
  ["Follow the signal back from the amplifier to the input to find where it disappears (Scopie helps)."], "Hard", tools=["scope"])
result("r.amp", "The amplifier has failed", "The amplifier chip or output transistors have failed.",
  ["Check its supply voltage first.", "Replace the amplifier chip (its code is printed on it), or the output transistors in pairs."], "Hard", parts=["Amplifier chip or transistors"], tools=["iron", "wick"])
result("r.clean", "Dirt was the problem", "Cleaning brought the contacts back.", ["Done. Keep drinks away next time."], "Easy")
result("r.trace", "A broken track", "A copper track has cracked or corroded through.",
  ["Scrape back the coating either side of the break with a knife until the copper shines.", "Solder a fine wire (30 AWG) across the break, or follow the track to the next solder point.", "Cover it with UV solder mask or nail varnish."], "Medium", tools=["iron", "flux", "magnifier"])
result("r.bearing", "A jammed or worn bearing", "Hair and dirt wrap round the shaft, and bearings dry out.",
  ["Unplug it. Clear any hair or fluff from the shaft.", "Clean the bearing and add a drop of light machine oil (not WD-40, it dries out).", "If it still grinds, replace the bearing or the motor."], "Easy", parts=["Bearing or motor"], tools=["screwdrivers"])
result("r.brushes", "Worn carbon brushes", "The brushes press on the motor's spinning part and wear down over time.",
  ["Replace both brushes with the right size for your motor (measure them, or search the model number).", "Run it gently for a minute so they bed in."], "Easy", parts=["Carbon brushes"], tools=["screwdrivers"])
result("r.motor", "The motor has failed", "A winding has broken, or the motor is worn out.",
  ["Replace the motor (or fan) with the same type, voltage and size."], "Medium", parts=["Motor or fan"])
result("r.runcap", "A failed start capacitor", "Fans need a capacitor to start turning: without it they only hum.",
  ["Replace it with a motor run capacitor of the same µF value and at least the same voltage (marked on it, often 450V AC)."], "Easy", parts=["Run capacitor"], tools=["screwdrivers"])
result("r.thermalfuse", "A blown thermal fuse", "It blew because something got too hot.",
  ["Find out why first: a blocked vent, a seized motor or a stuck thermostat.", "Replace it with the same temperature and current rating. Crimp it, or solder quickly with a heatsink clip: heat from the iron can blow a new one."], "Medium", parts=["Thermal fuse"], tools=["iron", "crimper"])
result("r.relay", "The heater is stuck on", "The part that switches the heater (a MOSFET on the board, or a relay) has failed shorted, so the heater gets full power all the time.",
  ["Switch it off at the wall now, and don't use it until it's fixed: a heater stuck on is a fire risk.", "Replace the heater's MOSFET or the control board. Many 3D printer owners add an external MOSFET board.", "Make sure the firmware's thermal runaway protection is turned on."], "Hard", parts=["MOSFET, relay or control board"], pro="If you're not sure, a heater stuck on is worth taking to someone who is.")
result("r.element", "A broken heating element", "The element's wire has burnt through.",
  ["Replace the heater with one of the same voltage and power (it's printed on it or in the machine's manual)."], "Medium", parts=["Heating element"])
result("r.elementearth", "Leakage to earth", "Something is leaking current to the metal frame or earth: dangerous.",
  ["Don't use it.", "Replace the faulty part, or the damaged lead.", "Have it safety-tested (a PAT test) before using it again."], "Pro", pro="Earth faults are a shock risk: a qualified electrician or appliance repairer should test it.")
result("r.heatdriver", "The board isn't powering the heater", "The heater's fuse on the board has blown, or its switching MOSFET has failed.",
  ["If the fuse blew, look for why first: a pinched heater wire touching the frame, or a shorted heater.", "Replace the fuse with the same rating.", "A failed MOSFET can be replaced on the board, or an external MOSFET board added; otherwise replace the mainboard."], "Hard", parts=["Fuse", "MOSFET or MOSFET board"], tools=["meter", "iron"])
result("r.thermistor", "A faulty temperature sensor", "The sensor wire is broken or shorted, or the sensor has failed.",
  ["Check the wire along its length and the plug on the board.", "Replace the thermistor with the same type (the printer's firmware expects a particular one, often a 100k NTC 3950)."], "Easy", parts=["Thermistor"], tools=["screwdrivers"])
result("r.datacable", "A charge-only cable or a bad port", "The cable has no data wires, or the port was faulty.", ["Label your charge-only cables so they don't get mixed up."], "Easy")
result("r.usbdata", "A USB data fault", "It has power, but the two data lines (D+ and D−) aren't getting through cleanly.",
  ["Try another cable and port first.", "Inside, check the USB socket's joints and the small protection chip next to it."], "Medium", tools=["iron", "magnifier"])
result("r.driver", "A missing driver", "The computer can see it but doesn't know what to do with it.",
  ["Install the driver or app from the device's maker.", iff("mcu|printer", "For boards with a USB-serial chip: WCH for CH340, Silicon Labs for CP210x, FTDI for FT232."), "Restart the computer, then try again."], "Easy")
result("r.bootmode", "It needed upload mode", "The board wasn't switching into upload mode on its own.",
  ["Use the button sequence each time, or fit a 10µF capacitor between EN and GND on ESP32 boards that never switch on their own."], "Easy")
result("r.strap", "Something on a start-up pin", "Something connected to a start-up pin stopped the chip entering upload mode.",
  ["Move that wiring to a different pin, or disconnect it while uploading."], "Easy")
result("r.mcu", "The board has failed", "The chip or its regulator has failed.",
  ["Check its 3.3V or 5V supply for a short.", "Development boards are cheap: replacing it is usually best."], "Easy", parts=["Replacement board"])
result("r.ledjoint", "A bad joint or LED in the strip", "The connection, or the first LED where the fault starts, has failed.",
  ["Press on the joint: if it flickers back, resolder it.", "Otherwise cut out that segment at the copper pads and solder in a new piece."], "Easy", parts=["LED strip"], tools=["iron"])
result("r.voltdrop", "Voltage drop along the strip", "Long strips lose voltage along their thin copper.",
  ["Feed power in at both ends, and every 2–5 metres along long runs.", "Use thicker wire to the strip, and a supply with enough current."], "Easy", parts=["Wire"])
result("r.ledchannel", "One colour channel has failed", "The controller's driver for that colour, or a wire, has failed.",
  ["Check the wire for that colour from the controller. Swap two colour wires to see whether the fault moves.", "Replace the controller if the fault stays with its output."], "Easy", parts=["LED controller"])
result("r.x2", "A failed suppression capacitor", "The X or Y capacitor across the mains has aged and burnt.",
  ["Replace it only with an X2-rated (or Y-rated, if that's what it was) capacitor of the same value. Never an ordinary capacitor."], "Medium", parts=["X2 or Y capacitor"], tools=["iron"])
result("r.melt", "A melted connector", "A poor contact got hot and melted the plastic.",
  ["Replace both halves of the connector, and the wire up to clean copper.", "Use a connector rated for the current."], "Medium", parts=["Connector"], tools=["crimper", "iron"])
result("r.mov", "A blown surge protector part", "The varistor (MOV) or X-capacitor shorted after a surge.",
  ["Replace it with the same type and voltage, and check the fuse.", "If there was a big surge, check other parts too."], "Hard", parts=["Varistor", "Fuse"], tools=["iron"])
result("r.water", "Dried out and working", "The clean-up worked.", ["Keep an eye on it for a few weeks: corrosion can carry on in places you couldn't reach."], "Easy")
result("r.corrosion", "Corrosion damage", "The liquid ate away parts or connections.",
  ["Replace damaged connectors and flex cables.", "Repair eaten tracks with fine wire.", "Replace small parts that have gone (you'll need the board diagram)."], "Hard", tools=["iron", "hotair", "magnifier"])
result("r.joint", "Cracked solder joints", "Heat and movement have cracked the solder.",
  ["Add a little flux and reflow each joint with a hot iron (about 350°C for leaded solder, 380°C for lead-free) and a bit of fresh solder.", "For old lead-free joints, adding some leaded solder helps it flow."], "Medium", tools=["iron", "solder", "flux"])
result("r.thermalpart", "A part that fails with temperature", "You found the part that misbehaves when hot or cold.",
  ["Replace it. Reflow its joints first if it's large: often it's the joint, not the part."], "Medium", tools=["iron", "hotair"])
result("r.pro", "Time for a qualified repairer", "This fault needs mains-safety training or special equipment.",
  ["Have it tested by an appliance repairer or electrician (a PAT test checks earth and insulation).", "Note down everything you've checked so far: it saves them time."], "Pro")

# ---------------------------------------------------------------- tools
T = [
  ("meter", "Multimeter", True, "£15–£40", "Measures voltage, resistance and continuity: the main tool for finding faults.",
   ["DC volts (V⎓): black probe on ground, red on the point to measure. Use auto-range, or a range above what you expect.",
    "Continuity (the beep or speaker symbol): beeps when two points are connected. Only on unplugged circuits.",
    "Resistance (Ω): unplugged only. Parts still on the board may read lower, because of what's around them.",
    "Diode test: a good diode reads about 0.5–0.7V one way and open (OL) the other. Close to 0V both ways is a short."],
   "Meterie shows your meter on the computer and logs it over time.", "meterie"),
  ("screwdrivers", "Precision screwdriver set", True, "£10–£30", "Small devices use tiny and special screws.",
   ["Get a set with Phillips PH000–PH1, Torx T3–T10 (including security Torx with a hole), pentalobe for iPhones, and tri-point for consoles.", "Push down firmly while turning: stripped screws are the hardest thing to recover from."], None, None),
  ("spudger", "Plastic pry tools and picks", True, "£5", "Open clips and glued seams without scratching or shorting anything.",
   ["Use plastic for anything near the battery or board.", "Thin guitar-pick-style picks slide into glued seams."], None, None),
  ("torch", "Bright torch", True, "Any", "Inspection needs strong, angled light.",
   ["Shine it low and from the side to show cracks, bulges and corrosion."], None, None),
  ("magnifier", "Magnifier or microscope", True, "£5–£60", "Cracked joints and tiny parts are invisible without magnification.",
   ["A phone camera zoomed in works well to start.", "A USB microscope or a 10× loupe is a big step up."], None, None),
  ("cable", "Known-good cables and chargers", True, "Already in your drawer", "To rule out the supply quickly.",
   ["Keep one charger and cable you know work, labelled, just for testing.", "Label charge-only USB cables."], None, None),
  ("ipa", "Isopropyl alcohol (99%)", True, "£5–£10", "Cleans flux, corrosion and sticky residue, and evaporates quickly.",
   ["Use 99% rather than 70% (which has more water).", "It's flammable: use it unplugged, in a ventilated room."], None, None),
  ("brush", "Soft brush (an old toothbrush)", True, "Free", "Scrubs corrosion and dust off boards.", ["Use with isopropyl alcohol."], None, None),
  ("iron", "Soldering iron", True, "£20–£80", "To replace parts and repair joints.",
   ["A temperature-controlled iron with a few tips (a chisel and a fine point) covers most jobs.", "Keep the tip clean and tinned (coated in solder).", "Work in a ventilated room."], None, None),
  ("solder", "Solder", True, "£8", "Joins parts.", ["0.5–0.8mm leaded solder (60/40 or 63/37) is the easiest to use; lead-free needs a hotter iron.", "Wash your hands after handling leaded solder."], None, None),
  ("flux", "Flux", True, "£5–£10", "Makes solder flow and joints form properly.", ["A flux pen or no-clean gel flux. Clean it off afterwards with isopropyl alcohol."], None, None),
  ("wick", "Desoldering wick or pump", True, "£5–£15", "Removes solder so parts can come out.", ["Press the wick onto the joint with the iron; the solder soaks into it.", "A pump sucks molten solder up in one go."], None, None),
  ("esd", "Anti-static wrist strap", False, "£5", "Static from your body can damage chips.", ["Clip it to the metal of the case or an earthed point.", "Work on a mat, not carpet."], None, None),
  ("usbmeter", "USB power meter", False, "£8–£25", "Shows voltage and current through a USB cable: the quickest charging test.",
   ["Plug it between the charger and the device.", "USB-C models also show the fast-charging protocol."], None, None),
  ("benchpsu", "Bench power supply", False, "£40–£100", "Powers a board at a set voltage with a current limit, so a fault can't do more damage.",
   ["Set the voltage first, with the current limit low, before connecting.", "Watch the current: a jump to the limit means a short."], None, None),
  ("hotspot", "Hot-spot finder", False, "£0–£250", "Finds the part that's getting hot.",
   ["Thermal camera: the clearest.", "Isopropyl alcohol: wipe a thin film on the board; it evaporates first from the hot part.", "Freeze spray: frost melts first on the hot part, and it can bring back faults that only happen when warm.", "A careful fingertip works on low-voltage boards."], None, None),
  ("probes", "Fine test probes and clips", False, "£5–£15", "Reach tiny test points without slipping and shorting things.",
   ["Needle-point probes or grabber clips. A slipped probe can short two pins and cause damage."], None, None),
  ("scope", "Oscilloscope", False, "£50–£400", "Shows signals that change too fast for a meter: clocks, audio, data.",
   ["Use it to check a crystal is running, follow an audio signal, or look at ripple on a power rail."], "Scopie runs USB and network scopes.", "scopie"),
  ("hotair", "Hot air station", False, "£40–£120", "Removes and fits surface-mount parts and connectors.",
   ["Use plenty of flux, and protect nearby plastic parts with Kapton tape or foil."], None, None),
  ("heatpad", "Heat pad or hair dryer", False, "£0–£30", "Softens the glue on phones and tablets.", ["About 80°C for a few minutes, then a suction cup and picks."], None, None),
  ("ledtester", "LED backlight tester", False, "£15–£30", "Lights each LED in a TV backlight strip, which a meter can't.", ["Touch its probes across one LED at a time."], None, None),
  ("contact", "Contact cleaner spray", False, "£6", "Cleans switches, knobs and connectors.", ["Use a cleaner made for electronics, not WD-40."], None, None),
  ("fuses", "Spare fuses", False, "£3", "Plug fuses (3A, 5A, 13A) and glass fuses.", ["Always replace with the same rating and type (fast or slow-blow)."], None, None),
  ("crimper", "Crimp tool", False, "£15–£30", "For connectors and thermal fuses that shouldn't be soldered.", ["Use the right size of crimp for the wire."], None, None),
  ("insulation", "Insulation tester or PAT tester", False, "£80+ (or a repair shop)", "Tests mains appliances at high voltage for leakage to earth.", ["A multimeter test is a rough check only: a PAT test is the proper one."], None, None),
  ("discharge", "Capacitor discharge tool", False, "£5 (DIY)", "Safely empties big capacitors in mains power supplies.",
   ["A 1–10kΩ resistor rated 5W or more, with insulated leads and clips. Hold it across the capacitor's legs for several seconds.", "Then measure: under about 5V is safe to touch.", "Never short a big capacitor with a screwdriver."], None, None),
]
tools = [dict(id=i, name=n, starter=s, cost=c, why=w, how=h, **({"app": a, "appWhy": aw} if a else {})) for i, n, s, c, w, h, aw, a in T]
# which devices a step's tool chip is shown for (the Tools tab lists them all)
toolFor = {"usbmeter": "usb|usbpow", "ledtester": "tv", "heatpad": "phone|sound", "insulation": "mains", "discharge": "hv", "fuses": "mains|adapter", "crimper": "mains|heater|printer|powertool"}
for t in tools:
    if t["id"] in toolFor: t["for"] = toolFor[t["id"]]

# ---------------------------------------------------------------- signs
G = [
  ("caps", "Bulging or leaking capacitors", "Round electrolytic capacitors should have flat tops. Domed tops, crusty brown deposits, or a capacitor pushed up off the board mean it's failed.", "Restart loops, no power, hum, a picture that takes a while to appear.", "Replace with the same µF, same or higher voltage, 105°C."),
  ("burnt", "Burnt parts", "Brown or black marks on the board or a part, cracked part bodies, a sharp burnt smell.", "The part failed, often because something after it shorted.", "Replace it, and find out what made it burn."),
  ("corrosion", "Corrosion", "Green, blue or white crust on contacts and pins; dull or eaten-away copper.", "Liquid got in, or a battery leaked.", "Clean with isopropyl alcohol and a brush; replace parts that have been eaten away."),
  ("joint", "Cracked solder joints", "A thin dark ring round a pin, or a dull, grainy joint that's lifted from the pad. Common on sockets and heavy or hot parts.", "Faults that come and go when it's moved or warms up.", "Reflow with flux and a little fresh solder."),
  ("fuse", "A blown fuse", "Glass fuses: a broken wire inside, or blackened or silvered glass. Others: no beep on continuity.", "Too much current flowed: a surge, or a short.", "Same rating only. If the new one blows, find the short first."),
  ("battery", "A swollen battery", "A lithium battery that is puffy, rounded or bent instead of flat. A case or screen that is pushed apart.", "A failing battery: a fire risk.", "Stop charging it. Remove carefully and recycle."),
  ("connector", "Loose or damaged connectors", "A plug that wobbles, bent or missing pins, cracked joints under the socket, a ribbon cable out of its latch.", "Intermittent faults, no charging, missing picture or sound.", "Clean, reseat, resolder, or replace the socket."),
  ("melted", "Melted plastic", "Deformed or browned connectors, a melted plug, discoloured wire insulation.", "A poor connection or too much current made heat.", "Replace both halves of the connector and the damaged wire."),
  ("trace", "A broken track", "A thin crack across a copper line, often near screw holes, edges or corrosion.", "Something stopped working completely after a drop or a spill.", "Bridge it with fine wire."),
  ("dust", "Dust and hair", "Grey felt across heatsink fins, fluff round fans, hair wound round motor shafts.", "Overheating, shutdowns, a noisy or slow fan or motor.", "Blow out with compressed air, clean the shaft."),
]
signs = [dict(id=i, name=n, look=l, means=m, fix=f) for i, n, l, m, f in G]
signFor = {"battery": "lipo|powertool", "dust": "fan|motor"}
for g_ in signs:
    if g_["id"] in signFor: g_["for"] = signFor[g_["id"]]

# ---------------------------------------------------------------- safety
safety = [
  dict(id="mains", t="Mains electricity", level="danger", points=[
    "Unplug it, and keep the plug where you can see it. Switching off isn't enough.",
    "Power supplies have big capacitors that stay charged after unplugging. Wait a few minutes, then measure across them: under about 5V is safe. Use a discharge resistor if they hold charge.",
    "On the mains side of a power supply, 'ground' is live. Never clip a probe there with it plugged in.",
    "If you must measure with it on, keep one hand in your pocket and use probes with shrouded tips.",
    "Never fit a bigger fuse, and never bridge a fuse with wire or foil.",
    "After a mains repair, check the earth wire is connected and nothing bare can be touched. A PAT test is the proper check."]),
  dict(id="lithium", t="Lithium batteries", level="warn", points=[
    "Disconnect the battery before anything else when you open a device.",
    "Never puncture, bend or crush one. Use plastic tools near it.",
    "Swollen batteries must not be charged. Store them away from anything that burns and recycle them.",
    "Never short the terminals: big currents flow and they get very hot.",
    "If one catches fire, get away from it and call the fire service."]),
  dict(id="solder", t="Soldering", level="warn", points=[
    "The iron is about 350°C: always put it back in its stand.",
    "Work in a ventilated room, or use a fume extractor.",
    "Wear glasses: solder can spit.",
    "Wash your hands after using leaded solder, and don't eat at the bench."]),
  dict(id="static", t="Static", level="info", points=[
    "Touch something earthed, or wear a wrist strap, before handling boards.",
    "Hold boards by their edges. Keep them in anti-static bags."]),
  dict(id="chem", t="Chemicals", level="info", points=[
    "Isopropyl alcohol and freeze spray are flammable: use them with the device unplugged and away from the iron.",
    "Battery leakage is caustic: wear gloves."]),
  dict(id="when", t="When to stop", level="info", points=[
    "If you're unsure what a step involves, stop and ask, or take it to a repair shop.",
    "If it's under warranty, opening it may void that: check first.",
    "If parts would cost more than a replacement, a repair shop or recycling may be the better choice."]),
]

guide = dict(version=2, power=power, features=features, derived=derived, categories=categories, danger=danger, legacy=legacy, symptoms=symptoms, steps=steps, tools=tools, signs=signs, safety=safety)
json.dump(guide, open(sys.argv[1], "w"), ensure_ascii=False, indent=1)
print(len(steps), "steps", sum(1 for s in steps.values() if s.get("result")), "results")
