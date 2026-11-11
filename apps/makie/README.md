# Makie

*Designed and created by **nrmluvdev**.*

A simple-to-use but powerful 3D part designer for 3D printing. Written in Go
(geometry engine + app server, no dependencies) with a built-in WebGL editor.
One self-contained executable — nothing to install.

## Running it

| Platform | File | How |
|---|---|---|
| Windows 10/11 (x64) | `Makie.exe` | Double-click. Opens in its own window (Edge app mode). |
| Windows on ARM | `Makie-arm64.exe` | Double-click. |
| macOS Apple silicon / Intel | `Makie-macos-arm64` / `-macos-intel` | `chmod +x` then run (right-click → Open the first time). |
| Linux x64 | `Makie-linux` | `chmod +x Makie-linux && ./Makie-linux` |

Makie opens an app window using Edge/Chrome/Brave if installed, otherwise your
default browser at `http://127.0.0.1:47811`. It closes itself a few minutes after
the window is closed (or use **File → Quit**). Your work autosaves and is restored
next time. Options: `-port 47811`, `-nobrowser`, `-idle 4m` (0 = never quit).

## The idea (Tinkercad-style)

1. **Add shapes** — click a shape in the library, then click on the bed or on any
   face of a part (or drag it in). Parts dropped on a face stand on that face.
2. **Holes** — any shape can be a *Hole*. **Group** holes with solids (Ctrl+G, or
   drag one part onto another in the Parts list) and the holes are cut out.
   Groups can be ungrouped and edited at any time (double-click a group).
3. **Export** — STL / 3MF / OBJ, watertight and in millimetres.

## Features

**Shapes:** box (rounded/chamfered edges), cylinder, cone, sphere, dome, torus,
tube, wedge, pyramid, polygon prism (with twist), star, capsule, rounded plate,
3D text (4 fonts), and custom sketches you draw in 2D and extrude (twist/taper)
or revolve.

*Smart drilling:* screw holes placed on a face go all the way through.

**Threads:** on cylinders, tubes (outside, inside or both), threaded holes,
bolts, nuts and rods. Profiles ISO metric 60°, trapezoidal 30°, square,
buttress; metric coarse/fine or custom pitch; 1–8 starts; right/left hand; fit
clearance (radial and flank); chamfered ends. A threaded solid and a threaded
hole of the same diameter fit together. The panel shows the exact major,
pitch and minor diameters, pitch, lead, depth, flank and helix angle.
"Make matching screw / threaded hole" creates the other half in one click.

**Hardware (33):** bolts (socket, hex, button, countersunk, pan, grub), hex /
square / wing / acorn / flange / coupling / knurled nuts, washers, spring lock
washer, threaded rod, threaded post, standoff, thumb screw, eye bolt, U-bolt,
screw hook, push rivet, dowel pin, shoulder bolt, carriage bolt, heat-set insert,
clevis pin, circlip, linear bearing (LM8UU…), ball bearing (608…), wall plug,
roll pin, hex key, cage nut, flanged bushing, levelling foot, extrusion T-nut.

**Holes (35):** screw clearance, self-tapping, threaded hole/bore, nut trap,
square-nut slot, heat-set insert, magnet pocket, bearing seat, slot, zip-tie
channel, keyhole hanger, teardrop (support-free), D/double-D/keyway/hex/spline
shaft hole, vent slots, hole grid, T-slot, dovetail slot, cable pass-through,
connector cutouts (USB-C/A, micro/mini USB, HDMI, RJ45, SD, audio), LED/button
hole, panel switch/pot hole, fan mount (25–120 mm), NEMA stepper mount, bolt
circle, PCB mounting (Raspberry Pi, Arduino), rounded pocket, O-ring groove,
edge rounder (fillet/chamfer cutter), snap-fit socket, living-hinge cuts, ball
socket, finger scoop, wire channel, tapered hole.

**Mechanical (50):** spur gears (straight, helical, herringbone), bevel gear,
internal ring gear, worm, rack, GT2 timing pulley, V/rope pulley, chain sprocket,
ratchet + pawl, cam (eccentric, heart, pear), Geneva drive, crank arm, linkage
bar, shaft coupler (rigid / flexible), universal-joint yoke, pillow block,
wheel with tread, ball joint, coil spring, flat spiral spring, flexure hinge,
print-in-place hinge, snap-fit clip, knurled knob, star knob, D-handle, L-bracket,
corner cube bracket, mending plates, shelf bracket, DIN-rail clip, 2020/3030/4040
extrusion, project box + lid, screw-top jar + lid, pipe bend, tube connectors,
hose barb, funnel, pipe clip, cable clip, strap loop, pegboard hook, S-hook,
chain link, dovetail rail, propeller, fan impeller, honeycomb panel.

**Electronics (70 boards & modules):** pick a board and make a *mount plate*
(standoffs), *standoffs only*, *snap-in tray* (for boards without holes),
*case* (with openings for its connectors), *case lid* (with a screen / sensor
window), *front bezel*, *mounting-hole pattern*, *holder* or a *fit-check model*.
Choose M2 / M2.5 / M3 / M4 screws and self-tapping, heat-set insert, bolt + nut
or snap-pin fixing. Covers Raspberry Pi 5/4/3/Zero/Pico/CM4/camera, Arduino
Uno/Mega/Nano/Micro/Pro Mini/MKR, ESP32 (DevKit V1/C, S3, CAM, C3/S3 SuperMini),
ESP8266 (NodeMCU, D1 mini), XIAO, Feather, QT Py, RP2040-Zero, STM32 pills,
Teensy, micro:bit, OLEDs (0.91", 0.96", 1.3"), TFTs (1.8"–3.5"), CYD 2.8" / 3.5",
round GC9A01, 16×2 / 20×4 LCDs, Nokia 5110, e-paper, 7-segment, WS2812 rings /
sticks / panels, MAX7219 matrices, HC-SR04, PIR, relays, L298N, buck converters,
TP4056, MPU6050, SD module, joystick, encoder, DHT22, SG90/MG996R servos,
28BYJ-48 and N20 motors — plus a Custom board. Also screw bosses, mounting ears,
lid corner posts and PCB snap pins for M2–M4. Dimensions are typical published
values; clone boards vary, so check yours or scan a model of it.

**Make parts for any model:** import an STL/OBJ (e.g. a new screen you modelled),
select it and press *Make parts for this model*. Makie slices it to find the
board outline, round mounting holes, a screen or sensor window on top,
connectors at the edges and the heights above/below, then creates fitted
mounts, a case with port openings, a lid with a window, a bezel, a hole pattern
or a cradle — placed around the model so you can check the fit, and fully
editable afterwards. Works on groups you built in Makie too.

**Filament hinge:** hinge whose axle is a length of 1.75 or 2.85 mm filament —
prints flat in one piece (or as two leaves), with press / free / loose fits.

**Editing:** move/rotate/resize gizmos with grid & angle snap, drag on the bed,
typed values with maths (`25.4*2`), smart snapping guides to other parts' edges
and centres, align & distribute, **auto-align**, **snap together** (click a face
on each part: flat faces mate flush and centred, round faces become concentric),
mirror, duplicate, copy/paste, patterns (row, grid, circle), drop to bed,
lay flat on a face, split a part in two, screw holes at corners, undo/redo,
lock/hide, measure tool, section view, overhang highlight, X-ray, view cube,
orthographic view.

**Motion mode:** rotation and slide joints with axes picked from faces, chained
joints (parts riding on moving parts), driven joints with automatic gear,
rack-and-pinion and lead-screw ratios, play/scrub, and **live collision detection**: parts are solid, so a moving part stops
the moment it touches another (shown in red, with the parts named) — just like the
real machine would jam. Parts that already overlap before anything moves are flagged
as jammed. *Check whole movement* runs through the full range in fine steps. **Print layout** lays every part out separately
on the bed (auto-orient to the largest flat face, or pick a face) and exports it
— your assembly is never changed.

**AMS / multi-material:** up to 16 filament slots; give each part a slot.
Export *3MF — multi-colour, one object* (Bambu Studio / OrcaSlicer read the
filament per part), *3MF — separate objects*, or *one STL per filament (zip)*
for any slicer. Parts without a slot use the closest filament colour.

**Tinkercad-style resizing:** drag the white handles on the selection box —
corners resize width and depth, the side squares one direction, the round top
handle the height (Shift keeps proportions, Alt turns off snapping). Click any
of the size labels (or the height above the bed) and type a new number.

**Strength check (structural integrity):** every part that takes a screw, bolt
or insert keeps at least 1.6 mm of material round the hole (more for bigger
screws) and at least 1.2 mm of wall elsewhere. A wall-thickness analyser checks
the whole library automatically, and *Print check* now finds thin walls in your
own designs and pins them in the view.

**Printer / build plate:** pick your printer from ~70 common models (Bambu,
Prusa, Creality, Elegoo, Anycubic, Voron, Qidi, Sovol …) or type a custom size —
click the printer chip at the bottom-left of the view. The plate is see-through
from underneath so you can look at the bottom of your parts.

**Gears:** spur, helical, herringbone, rack, bevel, crown, worm, internal ring,
compound, sector, intermittent, lantern pinion, planetary set, cycloidal drive,
**nautilus** and **elliptical** (non-circular) gears, ratchet with a flexible or
pivoting pawl. The **gear ratio calculator** finds one- and two-stage gear
trains for any ratio and adds them already meshed and linked in Motion.
**Make matching part…** (right-click or the inspector) makes the mating gear,
ring, rack, worm wheel, bevel, nut/bolt, bayonet half — or a cut-out that fits
any part — positioned and linked.

**Print-friendly hardware:** printable bolts and nuts with big coarse threads,
thick washers, bayonet mounts, and springs that work in plastic — flat and coil
torsion springs, zig-zag, diamond compression and leaf springs.

**Hoses & tube fittings:** hollow couplers, elbows (any angle), tees, Y
splitters, crosses, 3/5/6-way hubs, reducers, adapters, manifolds, swept bends,
end caps, panel bulkheads with printed nuts, a quarter-turn valve, hose clamps
and holder clips — each end can be a hose barb, push-in socket or spigot.

**Wires:** wire channels (straight, corner, U, S with an optional cover ledge),
cable notches, plug pass-throughs, grommets, raceway with snap lid, cable combs,
strain reliefs and drag-chain links.

**Patterns & text:** Voronoi, organic Voronoi, honeycomb, circles, squares,
diamonds, triangles, bricks, slots, waves, stars and radial patterns — as a
cut-out to lighten any part, a pattern panel, or a raised surface texture.
34 open-licence fonts for embossed or engraved text.

**IKEA SKÅDIS:** hooks, shelves, bins, tool holders, connector plates, cable
holders, rod holders and phone stands that hang on SKÅDIS pegboards.

**Board mounts:** besides plates, standoffs, trays, cases, lids and bezels,
boards can now slide into **slide-in rails** or a **slide-in case** with an end cap.

**Wirie boards:** Makie reads the `.makie.json` case files exported by
Wirie, nrmluvdev's PCB designer (format `wirie-case`, version 1). Press the
**Wirie** button on the toolbar (or Ctrl+Shift+O), use File → Open, drag the
file onto the window, pick it from the recent list in the Wirie menu, or
right-click it in Explorer → *Open with* → Makie. Makie watches `Documents\Wirie\Exports` (and
OneDrive's Documents) and offers new exports by themselves. The board is
shown in 3D: the outline extruded to its thickness, Wirie's pictures on the
top and bottom, every part as a coloured block and real mounting holes. It
is a locked *reference object*, so it isn't exported unless you switch
*Include in exports* on. Click a part to see its reference, value and
height, and to change its height, how far it is raised, and the size of its
lid or wall opening.

*Make a case for this board* opens a live preview with settings taken from
the board: wall, floor, fit gap, standoff height, lid clearance, corner
radius, lid style (screw-down, snap-fit or slide-in), screw size, self-tapping
or heat-set inserts, vent slots, text or a logo on the lid, and feet: indents
for stick-on rubber feet (8 × 2, 10 × 3, 12.7 × 3.5, 6.4 × 1.9 or 20 × 5 mm),
printed feet, or none. Keyboard cases keep taller printed feet at the back for
the typing angle, with a rubber-foot indent under each when you pick rubber feet. It builds a
base tray that follows the board outline, standoffs at every mounting hole,
chamfered wall openings for every connector at the right height, room in
the wall for anything that sticks out past the board edge (a module's USB end,
a USB-C socket, a button or an antenna: a pocket open at the top so the board
drops in, with a filler on the lid to close it), lid holes
for buttons, LEDs, screens, knobs and sensors, sound and vent grilles, and
floor holes for parts underneath. Extras: button extenders, light pipes and
a display bezel. A clash check runs after every change and turns any part
the case would hit red. The case is ordinary Makie geometry (a base group
and a lid group) that you can keep editing. *Export for printing* saves the
base, the lid (flipped) and the extras as STL, 3MF or zips, laid out flat
and keeping their AMS filament slots. The file format is described in
`docs/MAKIE_FORMAT.md` and the design in `docs/WIRIE.md`.

**Keyboards:** boards made with Keebie (Wirie's keyboard designer) carry a
`keyboard` block, and *Make a case* then builds a keyboard case instead: an
open-top tray that follows the board outline, with standoffs for the PCB
screws and openings for USB and TRRS, a printed switch plate (1.5 mm for MX,
1.2 mm for Choc) with a cut-out for every switch, knob and stabiliser, room
for the controller, screen and other tall parts, and holes over the PCB
screws, plus two rear feet for a typing angle. Settings: plate or no plate,
plate thickness, wall, floor, fit, standoff height, wall height above the
plate, typing angle, corner radius and screws. Custom (non-rectangular)
outlines are grown with a distance field so stepped and curved keyboards
keep a clean wall.

**Example:** *Show me an example* builds a working hand-cranked 9:1 gearbox —
press Play in Motion to watch it turn.

**Files:** projects (`.makie`, includes imported meshes; older `.smithers` files still open), STL/OBJ import,
print check (size, volume, weight estimate, bed fit, floating parts, overhangs,
watertightness, thin walls, loose holes), printer presets.

## Building from source

Requires Go 1.22+ only.

```
go build -o Makie .                                   # this machine
GOOS=windows GOARCH=amd64 go build -ldflags "-H windowsgui -s -w" -o Makie.exe .
GOOS=darwin  GOARCH=arm64 go build -ldflags "-s -w" -o Makie-macos-arm64 .
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o Makie-linux .
go test ./geom/                                           # geometry tests
```

`rsrc_windows_*.syso` holds the Windows icon, version info and manifest
(regenerate with `python3 tools/mksyso.py tools/icon .`).

Code map: `geom/` (BSP CSG, T-junction repair, shapes, threads, hardware,
mechanical parts, text, file formats), `main.go` (server + API),
`web/js/` (renderer, viewport, UI, motion, alignment, sketch editor).

Fonts: DejaVu (Bitstream Vera / DejaVu licence, free to redistribute).

---
Makie © nrmluvdev.
