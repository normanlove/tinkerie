// Tunie teaching-mode explanations, shown by teach.js. Created by nrmluvdev.
window.TEACH = {
  app: "Tunie",
  intro: "Point at any test, setting or page in Tunie to learn what it means.",
  tips: {
    // --------------------------------------------------------------- pages
    "Home": ["Home", "Your printer at a glance: which test to do next, your progress, and the settings Tunie has worked out so far."],
    "Tests|All tests": ["Tests", "Every calibration test Tunie can run. Each one prints a small test piece, tells you what to measure, and works out the new setting for you."],
    "Bed mesh": ["Bed mesh", "Shows how flat your print bed really is as a 3D map, and tells you which levelling screws to turn and by how much."],
    "Belt tension": ["Belt tension", "Checks your X and Y belts by the note they make when plucked, like tuning a guitar string, using your microphone. Belts that are too loose or tight spoil prints."],
    "Printers": ["Printers", "Your printers and their settings: bed size, firmware, nozzle size and more. Tunie keeps separate results for each printer."],
    "Filaments": ["Filaments", "Your spools, and what Tunie has learned about each one on the active printer: best temperature, flow, retraction and so on."],
    "History": ["History", "Every test you've run and its result, so you can see what changed and go back if needed."],
    "Export & share|Export / share": ["Export & share", "Saves your tuned settings as slicer profiles (Cura, PrusaSlicer, OrcaSlicer…) or as commands to send to the printer."],
    "Connection": ["Connection", "Optional: connect Tunie to your printer over USB, OctoPrint or Moonraker so it can send tests and read values directly. Without it, you save files to an SD card instead."],
    "Bambu Lab notes": ["Bambu Lab printers", "Bambu printers calibrate a lot by themselves. This page explains which Tunie tests still help them and how to print Tunie's files on a Bambu printer."],
    "About": ["About Tunie", "Version information and credits."],

    // ---------------------------------------------------------- top bar
    "Active printer": ["Active printer", "The printer you're tuning now. Tests and results are saved separately for each printer."],
    "Active filament": ["Active filament", "The filament loaded right now. Temperature, flow and retraction are different for each filament, so Tunie keeps results per filament."],
    "Not connected|Connected": ["Printer connection", "Whether Tunie is connected to your printer. You don't need a connection: every test can also be saved as a G-code file for an SD card."],
    "Marlin|Klipper|RepRapFirmware|Bambu": ["Firmware", "The software running on your printer. It decides which commands Tunie uses, for example Marlin's Linear Advance versus Klipper's Pressure Advance."],

    // ------------------------------------------------------------ tests
    "Start PID tuning|PID tuning": ["PID tuning", "Teaches the printer how to heat up quickly and then hold the nozzle and bed temperatures steady, without swinging up and down. Do this first: every other test needs steady temperatures."],
    "E-steps / rotation distance": ["E-steps / rotation distance", "Makes sure the extruder pushes exactly as much filament as asked. You mark the filament, ask for 100 mm, and measure what really went in."],
    "Flow rate|Flow": ["Flow rate", "Fine-tunes how much plastic comes out so walls are exactly as thick as planned. Too much gives blobby, oversized parts; too little gives weak gaps."],
    "Temperature": ["Temperature", "Prints a tower with a different temperature on each level, so you can pick the one with the best surface, bridges and layer strength for this filament."],
    "Retraction": ["Retraction", "Pulls the filament back a little before the nozzle travels, so it doesn't ooze and leave strings. Too much can cause gaps or jams."],
    "Pressure / linear advance|Linear advance|Pressure advance": ["Pressure / Linear advance", "Lets the printer push a little extra plastic when speeding up and a little less when slowing down, giving sharp corners and even lines at any speed."],
    "Max flow|Max volumetric flow": ["Max volumetric flow", "The most plastic your hotend can melt per second. Printing faster than this gives weak, under-extruded parts, so slicers use it to cap your speed."],
    "First layer": ["First layer", "Checks the gap between nozzle and bed (Z offset) and whether the bed is level. A good first layer is the base of every good print."],
    "Fit for Makie|Sliding fit|Tolerance": ["Fit and tolerance", "Prints pegs and holes with different gaps to find how much clearance your printer needs for parts to slide, snap or press together. Makie uses the result automatically."],
    "XY compensation": ["XY compensation", "Corrects holes printing too small and pegs too big, which happens because plastic spreads a little. Slicers can shrink or grow outlines by this amount."],
    "Mark as done": ["Mark as done", "Ticks this step off your list without running the test, for example if you've already tuned it another way."],
    "Recommended order": ["Recommended order", "The best order to tune a new printer: each test builds on the ones before it, so work from top to bottom."],
    "Active profile": ["Active profile", "The settings Tunie has worked out for this printer and filament so far. A dash means that test hasn't been done yet."],

    // ----------------------------------------------------- first-run setup
    "Printer": ["Printer", "Pick your printer model. Tunie fills in its bed size, firmware and sensible starting settings."],
    "Filament brand": ["Filament brand", "Who made your filament. Pick Generic if it isn't listed; you can tune it either way."],
    "First filament": ["Filament type", "What your filament is made of: PLA, PETG, ABS, TPU… Each needs different temperatures."],
    "Let's go": ["Start", "Saves your printer and filament and takes you to the home page."],
  },
};
