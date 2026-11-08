# Wirie boards in Makie — design notes

How Makie turns a Wirie `.makie.json` file (format `wirie-case`, version 1,
see `MAKIE_FORMAT.md`) into a reference board and a printable case.

## Where things live

| Piece | File | Job |
|---|---|---|
| Format, validation | `geom/wirie.go` (`ParseWirie`, `normalize`) | Reads the file, checks `format` / `version`, ignores unknown keys, fills in a missing outline or envelope, gives friendly errors. |
| Board model | `geom/wirie.go` (`wirieboard` shape, `BoardMesh`) | The outline extruded by the board thickness with real mounting holes, then one 12-triangle block per part in file order, so the editor can colour and pick each one. |
| Case builder | `geom/wirie.go` (`BuildCase`) | Turns the board and the case settings into ordinary editable node trees: a base group, a lid group and loose extras. |
| Case shapes | `geom/wirie.go` (`wcase_*` shapes) | Tray, lid plate, standoff, lid post, chamfered cut-out, hole grille, button extender / light pipe, bezel, logo. They are hidden from the shape library. |
| Clash check | `geom/wirie.go` (`ClashCheck`) | Builds every case node, moves it into the board frame, and tests each triangle against every part box (separating-axis test, plus an inside test). |
| HTTP | `wirie_api.go` | `/api/wirie/parse`, `/recent`, `/open`, `/case` and `/clash`. |
| Editor | `web/js/wirie.js` | Import, drag-and-drop, watching the export folder, drawing the board (pictures and part colours), the board inspector, the live case panel and the print-ready export. |

## Frames

Everything Wirie sends is in the **board frame**: origin at the board's
bottom-left corner, board underside at z = 0. The board node's transform
places that frame in the scene. The case is built in the board frame, and
each top-level case node is then given `boardWorld × local`, so the case
always lands on the board wherever the board is. The clash check undoes the
board transform (`InverseRigid`) before comparing with the part boxes.

## Case heights (board frame)

```
zLidTop  ─────────────  lid top            = zTop + lid thickness
zTop     ─────────────  wall top           = board thickness + lid clearance
              (slide-in: walls go up to zLidTop + 1.2 to hold the lid)
0..t     ═════════════  board
zFloorTop ────────────  floor top          = −standoff
zFloorBot ────────────  floor underside    = −(standoff + floor)
```

The standoff is raised to clear the parts underneath, and the lid clearance
is raised to clear the tallest part on top. A warning is shown whenever a
setting the user typed had to be raised.

## Lid styles

* **Screw-down**: posts under the lid press the board onto the standoffs.
  One long screw per mounting hole goes through the lid, the post, the board
  and into the standoff (self-tapping, or into a heat-set insert). Makie works
  out a standard screw length.
* **Snap-fit**: a lip under the lid fits inside the walls. Small ramps on the
  lip click into matching grooves in the walls.
* **Slide-in** (rectangular boards only): grooves along the walls, with the
  right-hand end wall cut down so the lid slides in from that end.

Connectors that stick out past the board edge get a wall slot open to the
top, so the board can be lowered in. The lid has a tab that fills the slot
down to just above the plug.

## Editor flow

1. Import (File menu, drag-and-drop, the recent list or the new-export
   banner) → `/api/wirie/parse` → a locked `wirieboard` node with
   `reference: true`. Its pictures are stored once in `S.textures` and
   saved with the project.
2. Clicking the board picks a part: the picked triangle's index, minus the
   slab's triangles, divided by 12. Part edits rewrite `params.board` and
   keep lid features and openings in step.
3. *Make a case*: the panel's form comes from `caseParams` (with defaults
   from the envelope). Every change waits 0.4 s, then rebuilds the case
   through `/api/wirie/case` and runs `/api/wirie/clash`.
4. The export lays out the base, the lid (turned over), the button
   extenders (upright) and the bezel (flat) in a row on the bed, then sends
   them through the normal exporter, so AMS filament slots are kept.
   *Print layout* turns the lid over too (`printRot`). Reference objects are
   left out of exports, print checks and the layout unless *Include in
   exports* is on.
