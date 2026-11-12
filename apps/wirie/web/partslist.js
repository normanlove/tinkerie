// Wirie parts list: everything to buy for the board on the PCB tab, and for a
// Keebie keyboard (every board of it: both halves of a split), including the
// things that aren't on the PCB (switches, keycaps, standoffs, screws, cables).
// Created by nrmluvdev.

// rows: { group, qty, part, detail, refs }
function pcbPartsRows() {
  const rows = [], groups = new Map(), holes = new Map();
  for (const c of S.comps) {
    const fp = compFP(c); if (!fp) continue;
    if (/^mh_/.test(fp.id)) { const k = fp.value; holes.set(k, (holes.get(k) || 0) + 1); continue; }
    if (fp.cat === "Mechanical" || /^kb_stab_/.test(fp.id)) continue;
    const k = fp.id + "|" + c.value;
    if (!groups.has(k)) groups.set(k, { fp, value: c.value, refs: [] });
    groups.get(k).refs.push(c.ref);
  }
  const num = (a, b) => a.localeCompare(b, undefined, { numeric: true });
  for (const g of [...groups.values()].sort((a, b) => num(a.refs[0], b.refs[0]))) {
    const tht = g.fp.pads.some(p => p.drill), smd = g.fp.pads.some(p => !p.drill);
    rows.push({ group: "On the board", qty: g.refs.length, part: g.value || g.fp.value || g.fp.name, detail: g.fp.name + (smd && !tht ? " · surface-mount" : ""), refs: g.refs.sort(num).join(", ") });
  }
  for (const [size, n] of holes) {
    const m = String(size).match(/M\d(?:\.\d)?/), sz = m ? m[0] : String(size);
    rows.push({ group: "Building it", qty: n, part: `${sz} standoffs`, detail: "Brass or nylon, height to suit your case", refs: "" });
    rows.push({ group: "Building it", qty: n * 2, part: `${sz} screws`, detail: "Two per standoff (one each side)", refs: "" });
  }
  return rows;
}

function keebPartsRows() {
  const K = KBS(); if (!K.keys.length) return [];
  const G = kbGeom(), rows = [], add = (group, qty, part, detail = "") => { if (qty > 0) rows.push({ group, qty, part, detail, refs: "" }); };
  const boards = G.boards.filter(Boolean), places = boards.flatMap(b => b.places);
  const keys = places.filter(p => p.k.kind !== "knob"), knobs = places.filter(p => p.k.kind === "knob");
  const sw = G.sw, hs = /_hs$/.test(sw.id);
  add("Keys", keys.length, `${sw.name.split(",")[0]} switches`, sw.fam === "choc" ? "Kailh Choc v1" : sw.fam === "alps" ? "Alps / Matias" : "Any MX-style switch");
  if (hs) add("Keys", keys.length, sw.fam === "choc" ? "Kailh Choc hot-swap sockets" : "Kailh MX hot-swap sockets", "Soldered to the back of the board");
  const caps = {}; for (const p of keys) { const k = p.k, s = k.shape === "iso" ? "ISO enter" : k.h > k.w ? `${k.h}u vertical` : `${k.w}u`; caps[s] = (caps[s] || 0) + 1; }
  const prof = KB_PROFILES.find(p => p.id === K.profile);
  for (const [s, n] of Object.entries(caps).sort((a, b) => b[1] - a[1])) add("Keys", n, `${s} keycaps`, prof ? prof.name + " profile" : "");
  const stabs = {}; for (const p of keys) { const st = kbStab(K, G, p); if (st) { const u = st.units >= 6 ? "6.25u+" : st.units >= 3 ? st.units + "u" : "2u"; stabs[u] = (stabs[u] || 0) + 1; } }
  for (const [u, n] of Object.entries(stabs)) add("Keys", n, `${u} stabilisers`, "PCB-mount (screw-in) stabilisers");
  add("Keys", knobs.length, "EC11 rotary encoders", "With push switch; 20mm shaft");
  add("Keys", knobs.length, "Knob caps", "To fit a 6mm D-shaft");
  // electronics on every board
  const tht = K.diode === "d_do35";
  add("Electronics", places.length, tht ? "1N4148 diodes" : "1N4148W diodes", tht ? "Through-hole, DO-35" : "Surface-mount, SOD-123");
  if (K.rgb) add("Electronics", keys.length, "SK6812 MINI-E RGB LEDs", "Surface-mount, under each key");
  if (K.rgb && K.ledCap) add("Electronics", keys.length, "100nF capacitors", "One beside each LED");
  const plan = new Map();
  for (const b of boards) for (const p of b.plan.parts) {
    const fp = LIB_BY_ID[p.fp]; if (!fp) continue;
    const k = p.fixed === "mcu" ? "mcu" : p.fp + "|" + p.value;
    if (!plan.has(k)) plan.set(k, { fp, value: p.value, n: 0, mcu: p.fixed === "mcu" });
    plan.get(k).n++;
  }
  for (const g of plan.values()) {
    if (g.mcu) { add("Electronics", g.n, G.mcu.name, G.mcu.bare ? "The chip: order the board with factory assembly" : g.fp.castellatedOf ? "Soldered flat by its castellated edges: no headers needed" : "Plus pin headers or low-profile sockets to mount it"); continue; }
    const smd = g.fp.pads.every(p => !p.drill);
    add("Electronics", g.n, g.value || g.fp.value, g.fp.name + (smd ? " · surface-mount" : ""));
  }
  // building it
  const holes = boards.reduce((n, b) => n + b.holes.length, 0), so = kbStandoff(K);
  add("Building it", boards.length, "Keyboard PCBs", boards.length > 1 ? "One for each half" : "Gerbers from the PCB tab's Export");
  add("Building it", boards.length, "Switch plates (optional)", sw.fam === "choc" ? "1.2mm, from Plate (SVG/DXF)" : "1.5mm, from Plate (SVG/DXF)");
  add("Building it", holes, `${so.id} standoffs, ${K.soH || 5}mm`, "Between the PCB and the plate or case");
  add("Building it", holes * 2, `${so.id} screws, 4mm`, "Two per standoff");
  if (K.split.mode === "two" && !G.mcu.wireless) add("Building it", 1, "TRRS cable (3.5mm, 4-pole)", "Joins the two halves");
  add("Building it", 1, G.mcu.wireless ? "USB cable (for charging and setting up)" : "USB cable", "To fit the controller's socket");
  if (G.mcu.wireless) add("Building it", boards.length, "LiPo battery (3.7V, 110–300mAh)", "With a JST-PH 2-pin plug");
  add("Building it", 4 * boards.length, "Rubber feet", "Stick-on bumpers");
  return rows;
}

function partsRowsCSV(rows) {
  const out = [["Section", "Quantity", "Part", "Details", "References"], ...rows.map(r => [r.group, r.qty, r.part, r.detail, r.refs])];
  return out.map(r => r.map(v => `"${String(v).replace(/"/g, '""')}"`).join(",")).join("\r\n") + "\r\n";
}
function partsRowsText(title, rows) {
  let t = title + "\n", g = null;
  for (const r of rows) { if (r.group !== g) { g = r.group; t += `\n${g}\n`; } t += `  ${String(r.qty).padStart(3)} × ${r.part}${r.detail ? " (" + r.detail + ")" : ""}${r.refs ? ": " + r.refs : ""}\n`; }
  return t + "\nMade with Wirie (Tinkerie) by nrmluvdev\n";
}

// the dialog: the keyboard's list (all boards) and/or the PCB tab's list
function partsListDialog(preferKeeb) {
  const hasKeeb = !!(S.keeb && S.keeb.keys && S.keeb.keys.length), hasPcb = S.comps.length > 0;
  if (!hasKeeb && !hasPcb) { toast("Add some parts first: there's nothing to list yet.", "warn"); return; }
  let which = preferKeeb && hasKeeb ? "keeb" : hasPcb ? "pcb" : "keeb";
  const m = modal(`<h3>Parts list</h3><div id="pl_tabs" class="seg" style="margin:6px 0 4px">${hasKeeb ? `<button data-w="keeb" data-help="Keyboard parts list">Whole keyboard${KBS().split.mode === "two" ? " (both halves)" : ""}</button>` : ""}${hasPcb ? `<button data-w="pcb" data-help="Board parts list">This board (PCB tab)</button>` : ""}</div>
    <p class="dim" id="pl_note"></p><div class="plist-wrap" id="pl_body"></div>
    <div style="display:flex;gap:8px;justify-content:flex-end;flex-wrap:wrap"><button class="btn" data-close>Close</button><button class="btn" id="pl_copy" data-help="Copy parts list">Copy as text</button><button class="btn primary" id="pl_csv" data-help="Download parts list">Download CSV</button></div>`, "wide");
  let rows = [];
  const draw = () => {
    rows = which === "keeb" ? keebPartsRows() : pcbPartsRows();
    m.querySelectorAll("#pl_tabs button").forEach(b => b.classList.toggle("on", b.dataset.w === which));
    m.querySelector("#pl_note").textContent = which === "keeb" ? "Everything to buy for this keyboard, worked out from its design: the parts on every board, plus switches, keycaps, standoffs, screws and cables." : "Every part on the board in the PCB tab, grouped by value, with their references.";
    let g = null, h = `<table class="plist"><tr><th>Qty</th><th>Part</th><th>Details</th>${which === "pcb" ? "<th>References</th>" : ""}</tr>`;
    for (const r of rows) { if (r.group !== g) { g = r.group; h += `<tr class="grp"><td colspan="4">${esc(g)}</td></tr>`; } h += `<tr><td class="q">${r.qty}</td><td>${esc(r.part)}</td><td class="r">${esc(r.detail)}</td>${which === "pcb" ? `<td class="r">${esc(r.refs)}</td>` : ""}</tr>`; }
    m.querySelector("#pl_body").innerHTML = h + "</table>";
  };
  m.querySelectorAll("#pl_tabs button").forEach(b => b.onclick = () => { which = b.dataset.w; draw(); });
  const name = () => (which === "keeb" ? KBS().name || "keyboard" : S.name || "board");
  m.querySelector("#pl_csv").onclick = () => { downloadFile(name() + "-parts.csv", partsRowsCSV(rows), "text/csv"); toast("Parts list downloaded"); };
  m.querySelector("#pl_copy").onclick = async () => { try { await navigator.clipboard.writeText(partsRowsText("Parts list: " + name(), rows)); toast("Parts list copied", "good"); } catch (e) { downloadFile(name() + "-parts.txt", partsRowsText("Parts list: " + name(), rows), "text/plain"); toast("Saved as a text file instead", "warn"); } };
  draw();
}
