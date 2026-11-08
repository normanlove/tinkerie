// Makie teaching-mode explanations, shown by teach.js. Shapes in the library
// already explain themselves through their own descriptions. Created by nrmluvdev.
window.TEACH = {
  app: "Makie",
  intro: "Point at any tool, shape or setting in Makie to learn what it does.",
  tips: {
    // ------------------------------------------------------------- toolbar
    "File": ["File", "Start a new design, open or save a Makie project (.makie), and import STL or OBJ models made in other programs."],
    "Export|Export for 3D printing": ["Export", "Saves your design as a file for your slicer (the program that prepares it for your 3D printer): STL, 3MF or OBJ, in millimetres and watertight."],
    "Wirie": ["Wirie boards", "Opens a circuit board you designed in Wirie, so Makie can build a case, lid, mounts or standoffs that fit it exactly."],
    "Undo": ["Undo (Ctrl+Z)", "Takes back your last change."],
    "Redo": ["Redo (Ctrl+Y)", "Puts back a change you undid."],
    "Move|Move tool": ["Move (W)", "Drag a part to slide it across the bed. The coloured arrows move it along one direction only; the green one lifts it up or down."],
    "Rotate": ["Rotate (E)", "Turn the selected part. Drag one of the coloured rings to spin it around that axis; it snaps to neat angles."],
    "Resize": ["Resize (R)", "Change the part's size by dragging its handles. You can also type exact sizes in the panel on the right."],
    "Group|Group selected": ["Group (Ctrl+G)", "Joins the selected parts into one. Any part set as a Hole is cut out of the solids it's grouped with. This is how you make holes, slots and pockets."],
    "Ungroup": ["Ungroup (Ctrl+Shift+G)", "Splits a group back into its separate parts so you can change them."],
    "Hole|Switch between solid and hole": ["Solid or hole (H)", "Turns the selected part into a Hole (shown striped). A hole does nothing on its own: group it with a solid and it's cut out of it."],
    "Align": ["Align", "Lines parts up with each other: left edges, centres, tops and so on. Auto-align and snap-together join parts face to face."],
    "Mirror": ["Mirror (M)", "Flips the part like a reflection, left to right, front to back or top to bottom. Handy for making left and right versions of a part."],
    "Duplicate": ["Duplicate (Ctrl+D)", "Makes a copy of the selected parts, right next to the originals."],
    "Pattern": ["Pattern", "Makes many copies at once: in a row, in a grid, or around a circle. Great for screw holes, vents and teeth."],
    "Drop": ["Drop (D)", "Lowers the part until it sits on the bed or on the part below it, so nothing floats in mid-air."],
    "Delete": ["Delete (Del)", "Removes the selected parts."],
    "Measure": ["Measure (Q)", "Click two points to measure the distance between them, in millimetres."],
    "View": ["View", "Ways to look inside and check your design: a section view that slices through it, overhang highlighting (steep areas that may need supports), and X-ray to see hidden parts."],
    "Print check": ["Print check", "Checks your design before you print: does it fit on your printer's bed, are any parts floating, are walls too thin, are there steep overhangs that need supports, and roughly how much filament it uses."],
    "Settings": ["Settings", "Your printer and its bed size, the grid, and how smooth curved surfaces are when exported (quality)."],
    "Help|Shortcuts and tips": ["Help (?)", "All the keyboard and mouse shortcuts, plus tips for designing parts that print well."],

    // ------------------------------------------------------------- modes
    "Design|Build and edit your parts": ["Design mode", "Where you build parts: add shapes, move and resize them, make holes and group them together."],
    "Motion|Add joints and test how parts move together": ["Motion mode", "Add hinges and slides between parts and test how they move: gears turning gears, lids opening, and whether anything jams."],
    "Print layout": ["Print layout", "Lays every part out flat on the bed, ready to print, with each one turned to its best side so it needs fewer supports."],

    // ------------------------------------------------------- shape library
    "Search shapes, screws, holes": ["Search shapes", "Type to find a shape: a screw size, a hinge, a gear, a box, a cable clip… The library has hundreds of ready-made parts."],
    "All": ["All shapes", "Shows every shape in the library."],
    "Basic": ["Basic shapes", "Simple building blocks: boxes, cylinders, spheres, cones and more. Most designs start with these."],
    "Holes": ["Ready-made holes", "Holes that are already the right size: screw holes, nut traps, magnet pockets, cable notches, fan mounts and more. Group one with a solid to cut it out."],
    "Hardware": ["Hardware", "Screws, nuts, washers, inserts and bearings, at real metric sizes, so you can check your design fits the parts you'll buy."],
    "Mechanical": ["Mechanical parts", "Hinges, brackets, clips, springs, knobs and handles you can print and use."],
    "Gears": ["Gears", "Spur, bevel, planetary and other gears that mesh correctly. Two gears with the same module fit together."],
    "Hoses": ["Hoses and pipes", "Hose barbs, elbows, adapters, clamps and fittings for air and water."],
    "Pegboard": ["Pegboard", "Hooks and holders that clip into standard pegboard, for organising tools."],
    "Electronics": ["Electronics", "Cases, mounts and standoffs for Raspberry Pi, Arduino, ESP32 and other boards, plus cut-outs for connectors and buttons."],
    "Sketch|Custom shape": ["Sketch", "Draw any outline in 2D, then pull it up into a solid or spin it around to make round things like vases and knobs."],

    // --------------------------------------------------------------- scene
    "Front|Back|Left|Right|Top|Bottom|Front view|Back view|Left view|Right view|Top view|Bottom view": ["View cube", "Click a side of the cube to look at your design from that side. Drag empty space to orbit around it."],
    "P|Toggle perspective / orthographic": ["Perspective or flat view (P)", "Perspective looks natural, with far things smaller. Orthographic (flat) keeps sizes equal everywhere, which is easier for lining things up."],
    "Nothing selected": ["Part settings", "Click a part to see and change its settings here: exact size, position, rounding, and options special to that shape."],
    "Parts": ["Parts list", "Every part in your design. Drag a part onto another to group them, or drag it out of a group to ungroup it."],
    "Snap|Change snap grid": ["Snap", "Parts move and resize in steps of this size, so they line up neatly. Hold Alt while dragging to move freely."],
    "Show me an example": ["Example", "Opens a small finished design so you can see how it was made."],
    "Open a Wirie board": ["Open a Wirie board", "Opens a board exported from Wirie (Export → Case data for Makie) so you can design a case or mount for it."],
    "Start with an empty bed": ["Empty bed", "Starts a new design with nothing on the bed."],
  },
};
