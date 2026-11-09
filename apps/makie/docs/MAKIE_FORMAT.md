# Wirie → Makie case data — `wirie-case` format, version 1

Wirie (by nrmluvdev) exports a **`.makie.json`** file that describes a circuit board in 3D, so Makie can build an enclosure, lid, standoffs, brackets or mounts for it automatically.

- **Where it's saved:** `Documents\Wirie\Exports\<project>.makie.json` (Export → "Case data for Makie"). A copy without textures is also inside every Gerber ZIP.
- **Encoding:** UTF-8 JSON. All lengths are in **millimetres**.
- **Identifying the file:** check `format === "wirie-case"` and `version` (an integer). New fields may be added within version 1, so ignore keys you don't know. A breaking change bumps `version`.

## Coordinate system
```
        +Y (back of the board)
         ^
         |   board seen from ABOVE
         |
 (0,0) --+-----> +X (right)
 bottom-left corner of the board
 +Z = up. Board bottom face is z = 0, top face is z = board.thickness (1.6).
 Rotations: degrees, counter-clockwise, looking down on the top side.
```
Parts on the **bottom** side have negative z values (they hang below z = 0).

## Top-level object
| key | type | meaning |
|---|---|---|
| `format` | `"wirie-case"` | file type tag |
| `version` | int | format version (currently 1) |
| `generator` | `{app, version, author, exported}` | e.g. `{"app":"Wirie","version":"1.1.0","author":"nrmluvdev","exported":"2026-11-06T22:40:00Z"}` |
| `project` | string | project name |
| `units` | `"mm"` | always millimetres |
| `coordinates` | string | human-readable reminder of the coordinate system |
| `board` | object | board shape (below) |
| `mountingHoles` | array | screw holes (below) |
| `parts` | array | every component as a 3D box (below) |
| `openings` | array | holes needed in the case **walls** (USB, power jacks, cables…) |
| `lidFeatures` | array | holes needed in the **lid / top face** (screens, buttons, LEDs, knobs, sensors) |
| `envelope` | object | overall heights + suggested case settings |
| `textures` | object, optional | PNG pictures of the board for realistic rendering |

### `board`
```json
{ "width": 80, "height": 60, "thickness": 1.6, "shape": "rect", "cornerRadius": 3,
  "outline": [[3,60],[2.4,59.9], ... ] }
```
- `shape`: `"rect"` or `"round"`. For `round`, the diameter is `width` and `cornerRadius = width/2`.
- `outline`: closed polygon (last point joins the first), **counter-clockwise**, with rounded corners already turned into points. Extrude it by `thickness` to get the board solid. It's the true board shape, so use it for the case's inner cavity and pocket.

### `mountingHoles[]`
```json
{ "x": 4, "y": 56, "d": 3.2, "plated": false, "ref": "H1", "screw": "M3" }
```
- `d` is the drilled hole diameter.
- `screw` is the suggested screw size (`M2`, `M2.5`, `M3`, `M4`, `M5`).
- Put standoffs or bosses here. A good boss outer diameter is `d * 2.2`.

### `parts[]`
```json
{ "ref": "U1", "value": "Pico", "name": "Raspberry Pi Pico", "footprint": "pico",
  "category": "Microcontrollers", "kind": "module", "side": "top",
  "x": 22, "y": 30, "rotation": 0,
  "box":  { "x0": 11.6, "y0": 4.5, "x1": 32.4, "y1": 55.5, "z0": 4.1, "z1": 8.6 },
  "body": { "x0": 11.5, "y0": 4.5, "x1": 32.5, "y1": 55.5 },
  "height": 4.5, "raisedBy": 2.5, "pins": 40, "approxHeight": true }
```
- `box` is the full 3D bounding box (axis-aligned, world coordinates) from `z0` to `z1`, including pads. It's the space the part occupies, so the case must not intrude into it.
- `body` is the plastic/board outline in plan view.
- `height` is the part's own height. `raisedBy` is the gap under it (modules soldered on header pins sit about 2.5 mm up). For top parts, `z0 = thickness + raisedBy` and `z1 = z0 + height`.
- `kind` is one of: `module`, `display`, `sensor`, `control`, `connector`, `button`, `switch`, `led`, `mechanical`, `component`.
- `approxHeight: true` means Wirie estimated the height from the part type. The user can override it in Wirie (Properties → Case / 3D), and the override is exported.

### `openings[]` — cut-outs in the side walls
```json
{ "ref": "U1", "type": "usb-micro", "face": "back", "x": 22, "y": 60, "z": 7.3,
  "width": 12, "height": 7, "partEdge": [22, 55.5], "normal": [0, 1], "distanceFromEdge": 4.5 }
```
- `face` is the wall the opening goes in: `left` (x = 0), `right` (x = width), `front` (y = 0) or `back` (y = height).
- `x, y` is the centre of the opening, already projected onto that board edge. `z` is the vertical centre.
- `width` × `height` is the suggested hole size, with plug clearance included. A USB plug overmould needs about 12 × 7.
- `partEdge` is where the connector itself ends. `distanceFromEdge` is the gap from the connector to the board edge: positive means it sits back from the edge (make a tunnel, or warn the user), negative means it overhangs the edge by that much (the wall must clear it).
- `normal` is a unit vector pointing out through the wall.
- `type` values include: `usb-c`, `usb-micro`, `usb-a`, `dc-jack`, `audio-3.5mm`, `rj45`, `rca`, `mini-din`, `d-sub`, `hdmi`, `sd-card`, `power-xt`, `atx`, `cable`, `wire-entry`, `slide-switch`, `card-edge`, `connector`. Treat any unknown type as a plain rectangular cut-out.

### `lidFeatures[]` — cut-outs in the lid / top face
```json
{ "ref": "SW1", "type": "button", "shape": "circle", "x": 42, "y": 10, "d": 4, "side": "top", "partTop": 6.6, "approx": false }
{ "ref": "M1", "type": "display-window", "shape": "rect", "x": 60, "y": 31.7, "w": 23.3, "h": 19.8, "side": "top", "partTop": 8.6, "approx": true }
```
- `shape` is `circle` (uses `d`) or `rect` (uses `w`, `h`, axis-aligned).
- `partTop` is the z of the top of that part. Use it to size button extensions or caps (lid underside z − `partTop`), light pipes, or to recess the lid around a screen.
- `side` is `top`, or `bottom` for parts mounted underneath (cut the floor instead).
- `type` values: `button`, `led`, `light-pipe`, `display-window`, `knob`, `switch`, `sensor-window`, `sensor-hole`, `sound-holes`, `vent`.
- `approx: true` marks windows Wirie guessed from the module size (mainly screens). Show these to the user so they can adjust.

### `envelope`
```json
{ "maxPartHeightTop": 8.6, "maxPartHeightBottom": 0, "suggestedStandoff": 3,
  "suggestedLidClearance": 10.6, "suggestedWall": 2.0, "suggestedFit": 0.3 }
```
- Heights are measured from the board surface on each side.
- `suggestedStandoff` is the board height above the case floor.
- `suggestedLidClearance` is the space needed above the board's top face.
- `suggestedFit` is the gap between the board and the case walls.

### `textures` (optional)
```json
{ "pixelsPerMm": 8, "top": "data:image/png;base64,...", "bottom": "data:image/png;base64,..." }
```
These are realistic renders of the board (solder mask, copper, silkscreen) with a transparent background outside the outline. `top` is seen from above. `bottom` is seen from **below**, so it is mirrored left-to-right compared with the top. Image size is `width*ppm × height*ppm` pixels. Map them onto the board's top and bottom faces.

## Building a basic case
1. **Inner cavity:** the board outline offset outwards by `suggestedFit`.
2. **Floor:** at `z = -(suggestedStandoff + floorThickness)`. Add a standoff at each mounting hole, from the floor up to `z = 0`, with a screw hole of the right size for `screw`.
3. **Walls:** from the floor up to `z = thickness + suggestedLidClearance`, thickness `suggestedWall`.
4. **Wall cut-outs:** cut every `openings[]` entry through its `face` wall, centred at (`x`/`y`, `z`).
5. **Lid:** cut every `lidFeatures[]` entry with `side == "top"`. Cut entries with `side == "bottom"` through the floor.
6. **Clash check:** make sure no case geometry enters any `parts[].box`.

Example files: open `example.makie.json` alongside this spec.

---
Format by nrmluvdev · Wirie 0.8
