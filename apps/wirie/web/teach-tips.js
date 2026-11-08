// Wirie teaching-mode explanations, shown by teach.js. Created by nrmluvdev.
window.TEACH = {
  app: "Wirie",
  intro: "Point at any button, box or tab in Wirie to learn what it does.",
  tips: {
    // ------------------------------------------------------------- top bar
    "PCB": ["PCB designer", "Design a real circuit board that a factory can make: place parts, connect their pins, let Wirie route the copper tracks, check it, then export the files to order it."],
    "Boardie": ["Boardie: stripboard and perfboard", "Plan the same circuit on stripboard or perfboard instead, for building by hand. Boardie places the parts, works out which copper strips to cut and where wire links go, and prints a build plan."],
    "Calcie": ["Calcie: electronics calculators", "Everyday electronics maths: Ohm's law, resistor colour codes, the right resistor for an LED, voltage dividers, 555 timers, battery life, wire and track widths and more."],
    "Pinnie": ["Pinnie: interactive pinouts", "Pin maps for popular boards and chips (Pico, ESP32, Arduino, the Raspberry Pi header…). Point at a pin to see what it can do (PWM, analogue input, I²C, SPI, UART) and any gotchas."],
    "Powerie": ["Powerie: battery power planner", "Works out how long a battery will last. Add each part's current while awake and asleep, how often it wakes up, and the regulator, and Powerie estimates the runtime."],
    "Untitled board|#projname|Project name": ["Project name", "The name of this board. It's used for the saved project file and the files you export."],
    "New": ["New project", "Starts a fresh, empty board. Save your current one first if you want to keep it."],
    "Open": ["Open a project", "Opens a board you saved before. Projects live in Documents\\Wirie\\Projects."],
    "Save": ["Save", "Saves this board so you can come back to it later. Wirie also restores your last session automatically."],
    "Export": ["Export", "Makes the files you need to get your board built: Gerber and drill files (zipped, ready to upload to JLCPCB, PCBWay, OSH Park and others), a parts list (BOM), and case data for Makie."],
    "Help": ["Help", "Shortcuts and how-to guides for Wirie."],
    "About Wirie": ["About Wirie", "Version information and credits."],

    // ------------------------------------------------------- parts library
    "Search parts — pico, oled, 10k, header|#libq": ["Search parts", "Type what you're looking for (a board name like pico, a part like oled, or a value like 10k) to find it in the library of over 700 parts."],
    "My Parts": ["My Parts", "Parts you've made yourself. They're kept in Documents\\Wirie\\Parts and work just like library parts."],
    "Create a part": ["Create a part", "Make the footprint (the pattern of copper pads) for a part that isn't in the library: start from a template such as a pin header or chip package, or place pads by hand."],
    "Check pin positions against your real module": ["Check the pin positions", "The pin layout of this module varies between sellers. Compare it with the board you actually have before ordering your PCB."],

    // ------------------------------------------------------------ tools
    "Select & move": ["Select & move (V)", "Click a part to select it, drag to move it. R rotates the selected part and F flips it to the back of the board."],
    "Connect pins": ["Connect pins (W)", "Tell Wirie which pins should be joined: click one pin, then another. These connections (called nets) are what the auto-router turns into copper tracks."],
    "Draw track": ["Draw track (T)", "Draw a copper track by hand, click by click. Useful for important wires you want to place yourself before auto-routing the rest."],
    "Via": ["Via (I)", "A via is a small plated hole that carries a track from one side of the board to the other."],
    "Text": ["Text (X)", "Add text to the board's silkscreen (the white printing): labels, your name, a version number."],
    "Measure": ["Measure (M)", "Click two points to measure the distance between them, in millimetres."],
    "Undo": ["Undo (Ctrl+Z)", "Takes back your last change."],
    "Redo": ["Redo (Ctrl+Y)", "Puts back a change you undid."],
    "Fit to screen": ["Fit to screen (Home)", "Zooms so the whole board fits in the window."],

    // --------------------------------------------------------- canvas bar
    "Top": ["Top copper side", "Tracks you draw go on the top copper layer. A 2-layer board has copper on both sides, so tracks can cross by using different sides."],
    "Bottom": ["Bottom copper side", "Tracks you draw go on the bottom copper layer, the back of the board."],
    "Active copper side": ["Active copper side (L)", "Which copper layer new tracks go on. Press L to switch."],
    "Design": ["Design view", "The working view: clear colours for each copper side, pads and nets, made for editing."],
    "Real": ["Realistic view", "Shows the board the way it will look when it arrives from the factory: solder mask colour, silkscreen and gold pads."],
    "Flip": ["Flip view", "Looks at the back of the board, as if you turned it over in your hand."],
    "Auto-route": ["Auto-route", "Lets Wirie draw the copper tracks for every connection you've made, using both sides of the board and vias where needed. Power tracks are made wider automatically."],
    "Remove auto-routed tracks": ["Remove auto-routed tracks", "Deletes the tracks the auto-router drew, keeping any you drew by hand, so you can move parts and route again."],
    "Check|Run check": ["Design check", "Checks the board before you order it: tracks too close together, shorts between different nets, connections that aren't routed yet, and parts too near the board edge."],

    // ------------------------------------------------------- getting started
    "Getting started": ["Getting started", "A checklist that walks you through your first board, one step at a time. Each step has a button that takes you straight there."],
    "Templates|Board templates": ["Board templates", "Ready-made board shapes with the right size and mounting holes: a Raspberry Pi HAT, a Pico carrier, an ESP32 board and more."],
    "Connect": ["Connect", "Joins pins together. Pick the Connect pins tool, or use Smart connect on a module to match pins by name."],

    // ------------------------------------------------------- right panel
    "Properties": ["Properties", "Settings for whatever is selected. With nothing selected, these are settings for the whole board."],
    "Nets": ["Nets", "Every connection in your circuit. A net is a group of pins that should all be joined, like GND or 3V3."],
    "Width": ["Board width", "How wide the board is, in millimetres."],
    "Height": ["Board height", "How tall the board is, in millimetres."],
    "Corner radius": ["Corner radius", "How rounded the board's corners are. 0 gives sharp corners; 2–3 mm looks neat and is kinder to fingers."],
    "Shape": ["Board shape", "The outline of the board: a rectangle, or another shape."],
    "Signal track": ["Signal track width", "How wide ordinary tracks are. 0.25–0.3 mm is fine for signals; every factory can make it."],
    "Power track": ["Power track width", "Tracks carrying power and ground are made wider so they can carry more current without heating up. 0.5–1 mm suits most hobby projects."],
    "Clearance": ["Clearance", "The smallest gap allowed between copper that shouldn't touch. 0.2 mm is safe for every common PCB factory."],
    "Via size / drill": ["Via size / drill", "The outer size of each via's copper ring, and the size of the hole drilled through it."],
    "Top side": ["Ground fill on top", "Fills the empty parts of the top copper layer with the fill net (usually GND). Less routing, less electrical noise."],
    "Bottom side": ["Ground fill on the bottom", "Fills the empty parts of the bottom copper layer with the fill net (usually GND). Most boards fill the bottom."],
    "Fill net": ["Fill net", "Which connection the copper fill belongs to. Almost always GND (ground)."],
    "0.5|0.635|1|1.27|2.54": ["Grid", "The snapping grid for placing parts, in millimetres. 2.54 mm (0.1 inch) matches pin headers and breadboards; 1.27 mm is half of that."],
    "parts": ["Parts", "How many parts are on the board."],
    "tracks": ["Tracks", "How many copper track segments the board has."],
    "vias": ["Vias", "How many vias (holes that join the two copper sides) the board has."],
  },
};
