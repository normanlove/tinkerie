// Wirie manufacturing export: Gerber RS-274X X2, Excellon drill, BOM, CPL — by nrmluvdev
"use strict";

class Gerber {
  constructor(func, polarityNeg) { this.func = func; this.aps = new Map(); this.body = []; this.next = 10; this.cur = null; }
  X(v) { return Math.round(v * 1e6); }
  Y(v) { return Math.round((S.board.h - v) * 1e6); }
  ap(shape, w, h) {
    w = Math.max(0.01, +w.toFixed(4)); h = Math.max(0.01, +(h ?? w).toFixed(4));
    let key;
    if (shape === "circle" || (shape === "oval" && Math.abs(w - h) < 1e-4)) key = `C,${w.toFixed(4)}`;
    else if (shape === "rect") key = `R,${w.toFixed(4)}X${h.toFixed(4)}`;
    else key = `O,${w.toFixed(4)}X${h.toFixed(4)}`;
    if (!this.aps.has(key)) this.aps.set(key, this.next++);
    return this.aps.get(key);
  }
  use(d) { if (this.cur !== d) { this.body.push(`D${d}*`); this.cur = d; } }
  flash(x, y, shape, w, h) { this.use(this.ap(shape, w, h)); this.body.push(`X${this.X(x)}Y${this.Y(y)}D03*`); }
  poly(pts, w) { if (pts.length < 2) return; this.use(this.ap("circle", w)); this.body.push(`X${this.X(pts[0][0])}Y${this.Y(pts[0][1])}D02*`); for (let i = 1; i < pts.length; i++) this.body.push(`X${this.X(pts[i][0])}Y${this.Y(pts[i][1])}D01*`); }
  region(pts) { this.body.push("G36*"); this.body.push(`X${this.X(pts[0][0])}Y${this.Y(pts[0][1])}D02*`); for (let i = 1; i < pts.length; i++) this.body.push(`X${this.X(pts[i][0])}Y${this.Y(pts[i][1])}D01*`); this.body.push(`X${this.X(pts[0][0])}Y${this.Y(pts[0][1])}D01*`); this.body.push("G37*"); }
  polarity(dark) { this.body.push(dark ? "%LPD*%" : "%LPC*%"); }
  toString() {
    const h = [`G04 Made with ${APP.name} ${APP.version} by ${APP.author}*`, `%TF.GenerationSoftware,${APP.author},${APP.name},${APP.version}*%`, `%TF.CreationDate,${new Date().toISOString().slice(0, 19)}*%`, `%TF.ProjectId,${(S.name || "board").replace(/[^A-Za-z0-9_-]/g, "_")},0,1*%`, `%TF.FileFunction,${this.func}*%`, "%FSLAX46Y46*%", "%MOMM*%", "%LPD*%", "G01*"];
    for (const [k, d] of this.aps) { const [t, p] = k.split(","); h.push(`%ADD${d}${t},${p}*%`); }
    return [...h, ...this.body, "M02*", ""].join("\n");
  }
}

function copperItemsOn(layer) {
  const A = analyze(); const out = [];
  for (const it of A.items) if (it.layers.includes(layer)) out.push(it);
  return out;
}
function emitItem(gb, it, grow) {
  if (it.kind === "pad") { const p = it.ref; if (p.ang) { if (p.shape === "rect") gb.region(padCorners(p, grow)); else { const s = padShape(p); gb.poly([[s.ax, s.ay], [s.bx, s.by]], Math.min(p.w, p.h) + 2 * grow); } } else gb.flash(p.x, p.y, p.shape, p.w + 2 * grow, p.h + 2 * grow); }
  else if (it.kind === "via") gb.flash(it.ref.x, it.ref.y, "circle", it.ref.d + 2 * grow);
  else { const s = it.shape; gb.poly([[s.ax, s.ay], [s.bx, s.by]], s.r * 2 + 2 * grow); }
}
// thermal-relief spokes for pads that join the ground fill (much easier to hand-solder)
const SPOKE_W = 0.5;
function thermalSpokes(p) { const L = Math.max(p.w, p.h) / 2 + S.board.pourClear + 0.3; return [[p.x - L, p.y, p.x + L, p.y], [p.x, p.y - L, p.x, p.y + L]]; }
function pourActive(layer) { const A = analyze(); return ((layer === "top" && S.board.pourTop) || (layer === "bottom" && S.board.pourBot)) && A.netByName[S.board.pourNet]; }

function gerberCopper(layer) {
  const gb = new Gerber(layer === "top" ? "Copper,L1,Top" : "Copper,L2,Bot");
  const A = analyze(); const items = copperItemsOn(layer);
  const pn = pourActive(layer);
  if (pn) {
    for (const bp of boardPolys(S.board.edgeClear)) gb.region(bp);
    gb.polarity(false);
    for (const c of S.board.cutouts || []) gb.region(offsetPoly(c, -S.board.edgeClear));
    for (const it of items) if (it.cnet !== pn || it.kind === "pad") emitItem(gb, it, S.board.pourClear);
    for (const h of A.holes) gb.flash(h.x, h.y, "circle", h.d + 2 * S.board.pourClear);
    gb.polarity(true);
    for (const it of items) if (it.kind === "pad" && it.cnet === pn) for (const sp of thermalSpokes(it.ref)) gb.poly([[sp[0], sp[1]], [sp[2], sp[3]]], SPOKE_W);
    gb.polarity(false);
    for (const it of items) if (it.cnet !== pn) emitItem(gb, it, S.board.pourClear);
    for (const h of A.holes) gb.flash(h.x, h.y, "circle", h.d + 2 * S.board.pourClear);
    gb.polarity(true);
  }
  for (const it of items) emitItem(gb, it, 0);
  return gb.toString();
}
function gerberMask(layer) {
  const gb = new Gerber(layer === "top" ? "Soldermask,Top" : "Soldermask,Bot");
  for (const it of copperItemsOn(layer)) if (it.kind === "pad") emitItem(gb, it, 0.05);
  for (const h of analyze().holes) gb.flash(h.x, h.y, "circle", h.d + 0.1);
  return gb.toString();
}
function gerberPaste(layer) {
  const gb = new Gerber(layer === "top" ? "Paste,Top" : "Paste,Bot");
  for (const it of copperItemsOn(layer)) if (it.kind === "pad" && !it.ref.drill) emitItem(gb, it, 0);
  return gb.toString();
}
// silkscreen geometry as polylines, shared by canvas + gerber + svg
function silkLines(layer) {
  const out = []; // {pts, w}
  const W = 0.15;
  for (const c of S.comps) {
    if ((c.side || "top") !== layer) continue;
    const fp = compFP(c); if (!fp) continue;
    const b = fp.body; const tf = pts => pts.map(([x, y]) => compXform(c, x, y));
    const x0 = b.x - b.w / 2, x1 = b.x + b.w / 2, y0 = b.y - b.h / 2, y1 = b.y + b.h / 2;
    if (fp.silk === "box") out.push({ pts: tf([[x0, y0], [x1, y0], [x1, y1], [x0, y1], [x0, y0]]), w: W });
    else if (fp.silk === "sides") { out.push({ pts: tf([[x0 + 0.2, y0 - 0.15], [x1 - 0.2, y0 - 0.15]]), w: W }); out.push({ pts: tf([[x0 + 0.2, y1 + 0.15], [x1 - 0.2, y1 + 0.15]]), w: W }); }
    else if (fp.silk === "circle") { const r = b.w / 2 + 0.15, pts = []; for (let i = 0; i <= 48; i++) { const a = i / 48 * Math.PI * 2; pts.push([b.x + r * Math.cos(a), b.y + r * Math.sin(a)]); } out.push({ pts: tf(pts), w: W }); }
    else if (fp.silk === "holes") for (const h of fp.holes) { const r = h.d / 2 + 0.6, pts = []; for (let i = 0; i <= 32; i++) { const a = i / 32 * Math.PI * 2; pts.push([h.x + r * Math.cos(a), h.y + r * Math.sin(a)]); } out.push({ pts: tf(pts), w: W }); }
    // pin-1 / polarity marker
    const p1 = fp.pads[0];
    if (p1 && fp.pads.length > 2 && fp.silk !== "none" && !fp.pinLabels) { const m = 0.5; out.push({ pts: tf([[x0 - m, p1.y - 0.6], [x0 - m, p1.y + 0.6]]), w: 0.2 }); }
    if (b.stripe) out.push({ pts: tf([[x0 + b.w * 0.18, y0 + 0.1], [x0 + b.w * 0.18, y1 - 0.1]]), w: 0.25 });
    // which end the USB socket is: an arrow pointing out of that edge and "USB"
    if (b.module && b.usb) {
      const s = b.usb === "bottom" ? -1 : 1, e = b.usb === "bottom" ? y1 : y0, cx = b.x;
      out.push({ pts: tf([[cx - 1.2, e + s * 1.9], [cx, e + s * 0.7], [cx + 1.2, e + s * 1.9]]), w: 0.2 });
      const ts = textStrokes("USB", 1.1, "c", false);
      for (const l of ts.lines) out.push({ pts: tf(l.map(([x, y]) => [cx + x, e + s * 3.3 + y])), w: 0.15 });
    }
    if (fp.id.startsWith("c_elec") || fp.id === "c_tant_a") { const p = fp.pads[0]; out.push({ pts: tf([[p.x - 1.6, p.y - 1.6], [p.x - 0.6, p.y - 1.6]]), w: W }); out.push({ pts: tf([[p.x - 1.1, p.y - 2.1], [p.x - 1.1, p.y - 1.1]]), w: W }); }
    // reference designator
    if (!c.hideRef) {
      const bx = compBox(c); const size = c.refSize || 1.0;
      const ts = textStrokes(c.ref, size, "c", layer === "bottom");
      const tx = c.refX !== undefined ? c.x + c.refX : (bx.x0 + bx.x1) / 2, ty = c.refY !== undefined ? c.y + c.refY : bx.y0 - size * 0.8;
      for (const l of ts.lines) out.push({ pts: l.map(([x, y]) => [tx + x, ty + y]), w: Math.max(0.12, size * 0.13) });
    }
    if (c.silkPins && fp.pinLabels) {
      for (const p of padsOf(c)) {
        const size = 0.8; const local = fp.pads.find(q => q.n === p.n);
        const outward = local.x < 0 ? -1 : 1; const horiz = c.rot % 180 === 0;
        const ts = textStrokes(p.name, size, horiz ? (outward * (c.side === "bottom" ? -1 : 1) * (c.rot === 180 ? -1 : 1) < 0 ? "r" : "l") : "c", layer === "bottom");
        const [ox, oy] = rotPt((c.side === "bottom" ? -outward : outward) * 1.4, 0, c.rot);
        for (const l of ts.lines) out.push({ pts: l.map(([x, y]) => [p.x + ox + x, p.y + oy + y]), w: 0.12 });
      }
    }
  }
  for (const t of S.texts) {
    if (t.layer !== layer) continue;
    const ts = textStrokes(t.text, t.size || 1.5, "c", layer === "bottom");
    for (const l of ts.lines) out.push({ pts: l.map(([x, y]) => { const [rx, ry] = rotPt(x, y, t.rot || 0); return [t.x + rx, t.y + ry]; }), w: Math.max(0.12, (t.size || 1.5) * 0.13) });
  }
  return out;
}
function gerberSilk(layer) {
  const gb = new Gerber(layer === "top" ? "Legend,Top" : "Legend,Bot");
  for (const l of silkLines(layer)) gb.poly(l.pts, l.w);
  return gb.toString();
}
function gerberOutline() {
  const gb = new Gerber("Profile,NP");
  for (const p of [...boardPolys(0), ...(S.board.cutouts || [])]) gb.poly([...p, p[0]], 0.1);
  return gb.toString();
}
function excellon(plated) {
  const A = analyze(); const holes = [];
  if (plated) { for (const p of A.pads) if (p.drill) holes.push(p.slot ? [p.x, p.y, Math.min(...p.slot), p.slot] : [p.x, p.y, p.drill]); for (const v of S.vias) holes.push([v.x, v.y, v.drill]); }
  else for (const h of A.holes) holes.push([h.x, h.y, h.d]);
  const tools = [...new Set(holes.map(h => h[2].toFixed(3)))].sort((a, b) => a - b);
  const L = ["M48", `; DRILL file ${APP.name} ${APP.version} by ${APP.author}`, `; ${plated ? "PLATED" : "NON-PLATED"} holes`, "; FORMAT={-:-/ absolute / metric / decimal}", "FMAT,2", "METRIC"];
  tools.forEach((t, i) => L.push(`T${i + 1}C${t}`));
  L.push("%", "G90", "G05");
  const Y = y => (S.board.h - y).toFixed(3);
  tools.forEach((t, i) => { L.push(`T${i + 1}`); for (const h of holes) if (h[2].toFixed(3) === t) {
    if (h[3]) { const [w, hh] = h[3], d = (Math.max(w, hh) - Math.min(w, hh)) / 2; const [ax, ay, bx, by] = w > hh ? [h[0] - d, h[1], h[0] + d, h[1]] : [h[0], h[1] - d, h[0], h[1] + d]; L.push(`X${ax.toFixed(3)}Y${Y(ay)}G85X${bx.toFixed(3)}Y${Y(by)}`); }
    else L.push(`X${h[0].toFixed(3)}Y${Y(h[1])}`);
  } });
  L.push("T0", "M30", "");
  return L.join("\n");
}
function bomCSV() {
  const groups = {};
  for (const c of S.comps) { const fp = compFP(c); if (!fp || fp.cat === "Mechanical") continue; const k = fp.id + "|" + c.value; (groups[k] = groups[k] || { fp, value: c.value, refs: [] }).refs.push(c.ref); }
  const rows = [["Comment", "Designator", "Footprint", "Quantity"]];
  for (const k of Object.keys(groups).sort()) { const g = groups[k]; rows.push([g.value, g.refs.sort((a, b) => a.localeCompare(b, undefined, { numeric: true })).join(","), g.fp.name, g.refs.length]); }
  return rows.map(r => r.map(v => `"${String(v).replace(/"/g, '""')}"`).join(",")).join("\r\n") + "\r\n";
}
function cplCSV() {
  const rows = [["Designator", "Mid X", "Mid Y", "Layer", "Rotation"]];
  for (const c of S.comps) { const fp = compFP(c); if (!fp || !fp.pads.some(p => !p.drill)) continue; rows.push([c.ref, c.x.toFixed(3) + "mm", (S.board.h - c.y).toFixed(3) + "mm", c.side === "bottom" ? "Bottom" : "Top", (360 - c.rot) % 360]); }
  return rows.map(r => r.join(",")).join("\r\n") + "\r\n";
}
function buildManufacturingFiles() {
  const base = (S.name || "board").replace(/[^A-Za-z0-9_-]+/g, "_");
  const f = {};
  f[`${base}-F_Cu.gtl`] = gerberCopper("top");
  f[`${base}-B_Cu.gbl`] = gerberCopper("bottom");
  f[`${base}-F_Mask.gts`] = gerberMask("top");
  f[`${base}-B_Mask.gbs`] = gerberMask("bottom");
  f[`${base}-F_Silkscreen.gto`] = gerberSilk("top");
  f[`${base}-B_Silkscreen.gbo`] = gerberSilk("bottom");
  f[`${base}-F_Paste.gtp`] = gerberPaste("top");
  f[`${base}-B_Paste.gbp`] = gerberPaste("bottom");
  f[`${base}-Edge_Cuts.gm1`] = gerberOutline();
  f[`${base}-PTH.drl`] = excellon(true);
  f[`${base}-NPTH.drl`] = excellon(false);
  f[`${base}-BOM.csv`] = bomCSV();
  f[`${base}-CPL.csv`] = cplCSV();
  f[`${base}.makie.json`] = JSON.stringify(makieExport({ textures: false }), null, 1);
  f["README.txt"] = [
    `${S.name} — manufacturing files`, `Made with ${APP.name} ${APP.version} by ${APP.author}`, "",
    `Board size: ${S.board.w} x ${S.board.h} mm, 2 layers, 1.6mm FR4 recommended`,
    "", "Upload this whole ZIP to your PCB maker (JLCPCB, PCBWay, OSH Park, Aisler...).",
    "Files:", "  *.gtl / *.gbl  top / bottom copper", "  *.gts / *.gbs  top / bottom solder mask", "  *.gto / *.gbo  top / bottom silkscreen", "  *.gtp / *.gbp  solder paste (only needed for stencils)", "  *.gm1          board outline", "  *-PTH.drl      plated holes & vias", "  *-NPTH.drl     unplated mounting holes", "  *-BOM.csv / *-CPL.csv  parts list & placement (for assembly services)", "  *.makie.json   3D board description for Makie (case designer) — not needed by the factory", "",
  ].join("\r\n");
  return f;
}

// ---------- 1:1 SVG for printing (home etching / paper fit test) ----------
function svgLayer(layer, mirror, what = "copper") {
  const B = S.board, A = analyze();
  const pad = 5, W = B.w + 2 * pad, H = B.h + 2 * pad;
  const tf = mirror ? `translate(${W - pad},${pad}) scale(-1,1)` : `translate(${pad},${pad})`;
  const parts = [];
  const shapeSVG = (it, grow, fill) => {
    if (it.kind === "pad") { const p = it.ref; const w = p.w + 2 * grow, h = p.h + 2 * grow; const rr = p.shape === "rect" ? 0 : Math.min(w, h) / 2; return `<rect${p.ang ? ` transform="rotate(${(p.ang * 180 / Math.PI).toFixed(3)} ${p.x.toFixed(3)} ${p.y.toFixed(3)})"` : ""} x="${(p.x - w / 2).toFixed(3)}" y="${(p.y - h / 2).toFixed(3)}" width="${w.toFixed(3)}" height="${h.toFixed(3)}" rx="${rr.toFixed(3)}" fill="${fill}"/>`; }
    if (it.kind === "via") return `<circle cx="${it.ref.x}" cy="${it.ref.y}" r="${it.ref.d / 2 + grow}" fill="${fill}"/>`;
    const s = it.shape; return `<line x1="${s.ax}" y1="${s.ay}" x2="${s.bx}" y2="${s.by}" stroke="${fill}" stroke-width="${2 * s.r + 2 * grow}" stroke-linecap="round"/>`;
  };
  const poly = p => p.map(q => q.map(v => v.toFixed(3)).join(",")).join(" ");
  if (what === "copper") {
    const items = copperItemsOn(layer); const pn = pourActive(layer);
    if (pn) {
      parts.push(`<mask id="pm"><rect x="-10" y="-10" width="${W + 20}" height="${H + 20}" fill="black"/>${boardPolys(B.edgeClear).map(bp => `<polygon points="${poly(bp)}" fill="white"/>`).join("")}${(B.cutouts || []).map(c => `<polygon points="${poly(offsetPoly(c, -B.edgeClear))}" fill="black"/>`).join("")}${items.filter(i => i.cnet !== pn || i.kind === "pad").map(i => shapeSVG(i, B.pourClear, "black")).join("")}${items.filter(i => i.kind === "pad" && i.cnet === pn).flatMap(i => thermalSpokes(i.ref)).map(s => `<line x1="${s[0]}" y1="${s[1]}" x2="${s[2]}" y2="${s[3]}" stroke="white" stroke-width="${SPOKE_W}"/>`).join("")}${A.holes.map(h => `<circle cx="${h.x}" cy="${h.y}" r="${h.d / 2 + B.pourClear}" fill="black"/>`).join("")}</mask>`);
      parts.push(`<rect x="-10" y="-10" width="${W + 20}" height="${H + 20}" fill="#000" mask="url(#pm)"/>`);
    }
    for (const it of items) parts.push(shapeSVG(it, 0, "#000"));
    // drill centres help when drilling by hand
    for (const p of A.pads) if (p.drill) parts.push(`<circle cx="${p.x}" cy="${p.y}" r="${Math.min(0.35, p.drill / 3)}" fill="#fff"/>`);
    for (const v of S.vias) parts.push(`<circle cx="${v.x}" cy="${v.y}" r="0.2" fill="#fff"/>`);
  } else {
    for (const l of silkLines(layer)) parts.push(`<polyline points="${poly(l.pts)}" fill="none" stroke="#000" stroke-width="${l.w}" stroke-linecap="round" stroke-linejoin="round"/>`);
  }
  for (const bp of [...boardPolys(0), ...(B.cutouts || [])]) parts.push(`<polygon points="${poly(bp)}" fill="none" stroke="#000" stroke-width="0.1"/>`);
  for (const h of A.holes) parts.push(`<circle cx="${h.x}" cy="${h.y}" r="${h.d / 2}" fill="none" stroke="#000" stroke-width="0.1"/>`);
  return `<svg xmlns="http://www.w3.org/2000/svg" width="${W}mm" height="${H}mm" viewBox="0 0 ${W} ${H}"><g transform="${tf}">${parts.join("")}</g></svg>`;
}
