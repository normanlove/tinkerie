// Wirie core: project model, geometry, nets & connectivity — by nrmluvdev
"use strict";

const APP = { name: "Wirie", version: "0.15.0-alpha", author: "nrmluvdev" };

function newProject() {
  return {
    app: "wirie", version: 1, name: "Untitled board",
    board: { w: 60, h: 40, r: 2, round: false, track: 0.3, power: 0.6, clear: 0.2, viaD: 0.8, viaDrill: 0.4, pourTop: false, pourBot: true, pourNet: "GND", pourRoute: true, pourClear: 0.35, grid: 1.27, edgeClear: 0.4 },
    comps: [], conns: [], tracks: [], vias: [], texts: [], nextId: 1,
    strip: { kind: "strip", cols: 40, rows: 26, place: {}, cuts: [], links: [], autoCuts: [], autoLinks: [] },
  };
}

let S = newProject();
const uid = () => "o" + (S.nextId++).toString(36);

// ---------------- undo ----------------
const UNDO = { stack: [], redo: [], last: null };
function snapshot() { return JSON.stringify(S); }
function resetHistory() { UNDO.stack = []; UNDO.redo = []; UNDO.last = snapshot(); }
function commit(label) {
  const s = snapshot();
  if (s === UNDO.last) return;
  if (UNDO.last) UNDO.stack.push(UNDO.last);
  if (UNDO.stack.length > 200) UNDO.stack.shift();
  UNDO.last = s; UNDO.redo = [];
  invalidate();
  if (typeof onCommit === "function") onCommit(label);
}
function undo() { if (!UNDO.stack.length) return; UNDO.redo.push(UNDO.last); UNDO.last = UNDO.stack.pop(); S = JSON.parse(UNDO.last); invalidate(); onCommit && onCommit("undo"); }
function redo() { if (!UNDO.redo.length) return; UNDO.stack.push(UNDO.last); UNDO.last = UNDO.redo.pop(); S = JSON.parse(UNDO.last); invalidate(); onCommit && onCommit("redo"); }

// ---------------- geometry helpers ----------------
const snapTo = (v, g) => Math.round(v / g) * g;
const dist = (ax, ay, bx, by) => Math.hypot(ax - bx, ay - by);
function rotPt(x, y, rot) {
  const r = ((rot % 360) + 360) % 360;
  switch (r) { case 0: return [x, y]; case 90: return [-y, x]; case 180: return [-x, -y]; case 270: return [y, -x]; }
  const a = r * Math.PI / 180, c = Math.cos(a), s = Math.sin(a); // any angle (keyboards: splayed and wavy keys)
  return [x * c - y * s, x * s + y * c];
}
function compXform(c, x, y) { const lx = c.side === "bottom" ? -x : x; const [rx, ry] = rotPt(lx, y, c.rot); return [c.x + rx, c.y + ry]; }
function compFP(c) { return LIB_BY_ID[c.fp]; }

// world pads of a component (cached by invalidate)
let CACHE = null;
function invalidate() { CACHE = null; if (typeof stripInvalidate === "function") stripInvalidate(); if (typeof UI !== "undefined") UI.dirty = true; }
function padsOf(c) {
  const fp = compFP(c); if (!fp) return [];
  const rr = ((c.rot % 360) + 360) % 360, q = Math.round(rr / 90) * 90;
  const odd = Math.abs(rr - q) > 1e-6; // not a multiple of 90°: pads keep their own angle
  const sw = (odd ? q : rr) % 180 !== 0;
  const ang = odd ? (c.side === "bottom" ? -1 : 1) * (rr - q) * Math.PI / 180 : 0;
  return fp.pads.map(p => {
    const [x, y] = compXform(c, p.x, p.y);
    const side = c.side || "top", other = side === "top" ? "bottom" : "top";
    return { key: c.id + ":" + p.n, comp: c, fpPad: p, n: p.n, name: p.name, func: p.func, x, y, w: sw ? p.h : p.w, h: sw ? p.w : p.h, shape: p.shape, drill: p.drill,
      slot: p.slot ? (sw ? [p.slot[1], p.slot[0]] : p.slot) : null,
      layers: p.drill ? ["top", "bottom"] : [p.opp ? other : side], ang };
  });
}
function holesOf(c) { const fp = compFP(c); if (!fp) return []; return fp.holes.map(h => { const [x, y] = compXform(c, h.x, h.y); return { x, y, d: h.d, comp: c }; }); }
function compBox(c) {
  const fp = compFP(c); if (!fp) return { x0: c.x - 1, y0: c.y - 1, x1: c.x + 1, y1: c.y + 1 };
  const b = fp.body; const pts = [[b.x - b.w / 2, b.y - b.h / 2], [b.x + b.w / 2, b.y + b.h / 2], [b.x - b.w / 2, b.y + b.h / 2], [b.x + b.w / 2, b.y - b.h / 2]];
  for (const p of fp.pads) { pts.push([p.x - p.w / 2, p.y - p.h / 2], [p.x + p.w / 2, p.y + p.h / 2], [p.x - p.w / 2, p.y + p.h / 2], [p.x + p.w / 2, p.y - p.h / 2]); }
  let x0 = 1e9, y0 = 1e9, x1 = -1e9, y1 = -1e9;
  for (const [x, y] of pts) { const [wx, wy] = compXform(c, x, y); x0 = Math.min(x0, wx); x1 = Math.max(x1, wx); y0 = Math.min(y0, wy); y1 = Math.max(y1, wy); }
  return { x0, y0, x1, y1 };
}
function bodyRect(c) {
  const b = compFP(c).body; const P = [[-1, -1], [1, 1], [-1, 1], [1, -1]].map(([sx, sy]) => compXform(c, b.x + sx * b.w / 2, b.y + sy * b.h / 2));
  return { x0: Math.min(...P.map(p => p[0])), y0: Math.min(...P.map(p => p[1])), x1: Math.max(...P.map(p => p[0])), y1: Math.max(...P.map(p => p[1])) };
}

// segment / shape distances. Shapes: {t:'cap', ax,ay,bx,by,r} or {t:'rect', cx,cy,hw,hh}
function segSegDist(ax, ay, bx, by, cx, cy, dx, dy) {
  if (segIntersect(ax, ay, bx, by, cx, cy, dx, dy)) return 0;
  return Math.min(ptSegDist(ax, ay, cx, cy, dx, dy), ptSegDist(bx, by, cx, cy, dx, dy), ptSegDist(cx, cy, ax, ay, bx, by), ptSegDist(dx, dy, ax, ay, bx, by));
}
function ptSegDist(px, py, ax, ay, bx, by) {
  const vx = bx - ax, vy = by - ay, l2 = vx * vx + vy * vy;
  let t = l2 ? ((px - ax) * vx + (py - ay) * vy) / l2 : 0; t = Math.max(0, Math.min(1, t));
  return Math.hypot(px - ax - t * vx, py - ay - t * vy);
}
function segIntersect(ax, ay, bx, by, cx, cy, dx, dy) {
  const o = (px, py, qx, qy, rx, ry) => Math.sign((qx - px) * (ry - py) - (qy - py) * (rx - px));
  const o1 = o(ax, ay, bx, by, cx, cy), o2 = o(ax, ay, bx, by, dx, dy), o3 = o(cx, cy, dx, dy, ax, ay), o4 = o(cx, cy, dx, dy, bx, by);
  return o1 !== o2 && o3 !== o4 && o1 * o2 <= 0 && o3 * o4 <= 0 && !(o1 === 0 && o2 === 0);
}
function ptRectDist(px, py, r) { const dx = Math.max(Math.abs(px - r.cx) - r.hw, 0), dy = Math.max(Math.abs(py - r.cy) - r.hh, 0); return Math.hypot(dx, dy); }
function segRectDist(ax, ay, bx, by, r) {
  const x0 = r.cx - r.hw, x1 = r.cx + r.hw, y0 = r.cy - r.hh, y1 = r.cy + r.hh;
  if ((ax >= x0 && ax <= x1 && ay >= y0 && ay <= y1) || (bx >= x0 && bx <= x1 && by >= y0 && by <= y1)) return 0;
  const edges = [[x0, y0, x1, y0], [x1, y0, x1, y1], [x1, y1, x0, y1], [x0, y1, x0, y0]];
  let m = 1e9;
  for (const e of edges) m = Math.min(m, segSegDist(ax, ay, bx, by, e[0], e[1], e[2], e[3]));
  return m;
}
function shapeDist(A, B) {
  if (A.t === "poly" || B.t === "poly") { // turned pads: distance from the polygon's edges
    if (B.t === "poly" && A.t !== "poly") return shapeDist(B, A);
    const P = A.pts, rect2poly = r => [[r.cx - r.hw, r.cy - r.hh], [r.cx + r.hw, r.cy - r.hh], [r.cx + r.hw, r.cy + r.hh], [r.cx - r.hw, r.cy + r.hh]];
    if (B.t === "cap" && ptInPoly(B.ax, B.ay, P)) return -1;
    const Q = B.t === "poly" ? B.pts : B.t === "rect" ? rect2poly(B) : null;
    if (Q && (ptInPoly(Q[0][0], Q[0][1], P) || ptInPoly(P[0][0], P[0][1], Q))) return -1;
    let m = 1e9;
    for (let i = 0; i < P.length; i++) { const a = P[i], b = P[(i + 1) % P.length];
      if (B.t === "cap") m = Math.min(m, segSegDist(a[0], a[1], b[0], b[1], B.ax, B.ay, B.bx, B.by) - B.r);
      else for (let j = 0; j < Q.length; j++) { const c = Q[j], d = Q[(j + 1) % Q.length]; m = Math.min(m, segSegDist(a[0], a[1], b[0], b[1], c[0], c[1], d[0], d[1])); } }
    return m <= 0 ? -1 : m;
  }
  if (A.t === "cap" && B.t === "cap") return segSegDist(A.ax, A.ay, A.bx, A.by, B.ax, B.ay, B.bx, B.by) - A.r - B.r;
  if (A.t === "rect" && B.t === "rect") { const dx = Math.max(Math.abs(A.cx - B.cx) - A.hw - B.hw, 0), dy = Math.max(Math.abs(A.cy - B.cy) - A.hh - B.hh, 0); const d = Math.hypot(dx, dy); return d === 0 ? -1 : d; }
  if (A.t === "rect") return shapeDist(B, A);
  return segRectDist(A.ax, A.ay, A.bx, A.by, B) - A.r;
}
// corners of a pad turned by a non-90° angle
function padCorners(p, grow = 0) { const w = p.w / 2 + grow, h = p.h / 2 + grow, c = Math.cos(p.ang), s = Math.sin(p.ang); return [[-w, -h], [w, -h], [w, h], [-w, h]].map(([x, y]) => [p.x + x * c - y * s, p.y + x * s + y * c]); }
function padShape(p) {
  if (p.ang && p.shape === "rect") return { t: "poly", pts: padCorners(p) }; // turned rectangle
  if (p.ang) { const r = Math.min(p.w, p.h) / 2, L = (Math.max(p.w, p.h) - 2 * r) / 2, a = p.ang + (p.h > p.w ? Math.PI / 2 : 0); return { t: "cap", ax: p.x - L * Math.cos(a), ay: p.y - L * Math.sin(a), bx: p.x + L * Math.cos(a), by: p.y + L * Math.sin(a), r }; }
  if (p.shape === "rect") return { t: "rect", cx: p.x, cy: p.y, hw: p.w / 2, hh: p.h / 2 };
  const r = Math.min(p.w, p.h) / 2, ex = (p.w - 2 * r) / 2, ey = (p.h - 2 * r) / 2;
  return { t: "cap", ax: p.x - ex, ay: p.y - ey, bx: p.x + ex, by: p.y + ey, r };
}
function shapeBox(s) { if (s.t === "poly") { const xs = s.pts.map(q => q[0]), ys = s.pts.map(q => q[1]); return [Math.min(...xs), Math.min(...ys), Math.max(...xs), Math.max(...ys)]; } return s.t === "rect" ? [s.cx - s.hw, s.cy - s.hh, s.cx + s.hw, s.cy + s.hh] : [Math.min(s.ax, s.bx) - s.r, Math.min(s.ay, s.by) - s.r, Math.max(s.ax, s.bx) + s.r, Math.max(s.ay, s.by) + s.r]; }
function ptInShape(x, y, s, tol = 0) { if (s.t === "poly") return ptInPoly(x, y, s.pts) || ptPolyEdgeDist(x, y, s.pts) <= tol; return s.t === "rect" ? ptRectDist(x, y, s) <= tol : ptSegDist(x, y, s.ax, s.ay, s.bx, s.by) <= s.r + tol; }

// ---------------- nets ----------------
class UF { constructor() { this.p = new Map(); } find(a) { let p = this.p.get(a); if (p === undefined) { this.p.set(a, a); return a; } if (p === a) return a; const r = this.find(p); this.p.set(a, r); return r; } union(a, b) { const ra = this.find(a), rb = this.find(b); if (ra !== rb) this.p.set(ra, rb); } }

const NET_COLORS = ["#4fd1c5", "#f6ad55", "#b794f4", "#68d391", "#fc8181", "#63b3ed", "#f687b3", "#faf089", "#9ae6b4", "#fbd38d", "#90cdf4", "#d6bcfa"];
function netPriority(name) {
  const n = name.toUpperCase();
  if (/^(GND|VSS|0V)$/.test(n)) return [0, "GND"];
  if (/^(3V3|3\.3V|3V3_OUT|\+3V3|VDD_3V3)$/.test(n)) return [1, "3V3"];
  if (/^(5V|\+5V|VBUS|VCC5)$/.test(n)) return [2, "5V"];
  if (/^(VIN|RAW|VSYS|VBAT|VMOT|12V)$/.test(n)) return [3, n];
  if (/^(VCC|VDD|V\+|\+)$/.test(n)) return [4, "VCC"];
  return [9, null];
}

// same-name pins that are joined inside the part itself (module GND pins, tactile switch legs)
function internallyLinked(p) { const fp = compFP(p.comp); return !!(fp.body.module || /^(sw6x6|sw12x12|sw6x6smd|sw4x4smd)$/.test(fp.id) || /^(SHIELD|MP|MH)$/.test(p.name)); }

// Build everything derived: pads, nets, copper connectivity, ratsnest.
// pourLink joins the fill-net items that the copper fill on one layer really
// reaches. The fill is traced on a 0.15 mm grid, drawn exactly as the board
// shows and exports it (clearances, thermal spokes, holes), so a ground pin
// cut off by other tracks is NOT counted as connected and gets routed.
const POUR_RES = 0.15;
let pourGrid = null;
function pourLink(layer, items, holes, isPour, cu) {
  const B = S.board, res = POUR_RES;
  const W = Math.ceil(B.w / res) + 2, H = Math.ceil(B.h / res) + 2;
  if (typeof document === "undefined" || W * H > 4e6) { // no canvas (or a huge board): trust the fill
    let anchor = null;
    for (const it of items) if (it.layers.includes(layer) && isPour(it)) { if (anchor === null) anchor = it.id; else cu.union(anchor, it.id); }
    return;
  }
  if (!pourGrid) pourGrid = document.createElement("canvas");
  if (pourGrid.width !== W || pourGrid.height !== H) { pourGrid.width = W; pourGrid.height = H; }
  const p = pourGrid.getContext("2d", { willReadFrequently: true });
  p.setTransform(1, 0, 0, 1, 0, 0); p.clearRect(0, 0, W, H);
  p.setTransform(1 / res, 0, 0, 1 / res, 0, 0);
  const on = items.filter(it => it.layers.includes(layer));
  p.fillStyle = "#fff"; pathBoard(p, B.edgeClear); p.fill();
  p.globalCompositeOperation = "destination-out"; p.fillStyle = "#000"; p.strokeStyle = "#000"; p.lineCap = "round";
  for (const it of on) if (!isPour(it) || it.kind === "pad") itemPath(p, it, B.pourClear);
  p.globalCompositeOperation = "source-over"; p.strokeStyle = "#fff"; p.lineWidth = SPOKE_W; p.lineCap = "butt";
  for (const it of on) if (it.kind === "pad" && isPour(it)) for (const sp of thermalSpokes(it.ref)) { p.beginPath(); p.moveTo(sp[0], sp[1]); p.lineTo(sp[2], sp[3]); p.stroke(); }
  p.globalCompositeOperation = "destination-out"; p.lineCap = "round";
  for (const it of on) if (!isPour(it)) itemPath(p, it, B.pourClear);
  for (const h of holes) { p.beginPath(); p.arc(h.x, h.y, h.d / 2 + B.pourClear, 0, 7); p.fill(); }
  p.globalCompositeOperation = "source-over";
  // Label the separate pieces of fill.
  const px = p.getImageData(0, 0, W, H).data;
  const lab = new Int32Array(W * H), stack = new Int32Array(W * H);
  let next = 0;
  for (let i = 0; i < W * H; i++) {
    if (lab[i] || px[i * 4 + 3] < 128) continue;
    next++; let sp = 0; stack[sp++] = i; lab[i] = next;
    while (sp) {
      const k = stack[--sp], x = k % W;
      if (x > 0 && !lab[k - 1] && px[(k - 1) * 4 + 3] >= 128) { lab[k - 1] = next; stack[sp++] = k - 1; }
      if (x < W - 1 && !lab[k + 1] && px[(k + 1) * 4 + 3] >= 128) { lab[k + 1] = next; stack[sp++] = k + 1; }
      if (k >= W && !lab[k - W] && px[(k - W) * 4 + 3] >= 128) { lab[k - W] = next; stack[sp++] = k - W; }
      if (k < W * (H - 1) && !lab[k + W] && px[(k + W) * 4 + 3] >= 128) { lab[k + W] = next; stack[sp++] = k + W; }
    }
  }
  const at = (x, y) => { const i = Math.round(x / res), j = Math.round(y / res); return i < 0 || j < 0 || i >= W || j >= H ? 0 : lab[j * W + i]; };
  // Which piece of fill touches each fill-net item: pads through their
  // spokes, tracks and vias where they lie (the fill flows round them).
  const first = {};
  const join = (l, it) => { if (!l) return; if (first[l] === undefined) first[l] = it.id; else cu.union(first[l], it.id); };
  for (const it of on) {
    if (!isPour(it)) continue;
    if (it.kind === "pad") {
      const cx = it.ref.x, cy = it.ref.y; // sample each spoke from the pad outwards
      for (const sp of thermalSpokes(it.ref)) for (const [ex, ey] of [[sp[0], sp[1]], [sp[2], sp[3]]])
        for (let f = 0.5; f <= 1.0001; f += 0.05) join(at(cx + (ex - cx) * f, cy + (ey - cy) * f), it);
    } else {
      const s = it.shape;
      for (let f = 0; f <= 1.0001; f += 0.25) {
        const x = s.ax + (s.bx - s.ax) * f, y = s.ay + (s.by - s.ay) * f, r = s.r + res;
        join(at(x, y), it); join(at(x + r, y), it); join(at(x - r, y), it); join(at(x, y + r), it); join(at(x, y - r), it);
      }
    }
  }
}

function analyze() {
  if (CACHE) return CACHE;
  const pads = [], padByKey = {}, holes = [];
  for (const c of S.comps) { for (const p of padsOf(c)) { pads.push(p); padByKey[p.key] = p; } holes.push(...holesOf(c)); }
  // netlist union-find over pad keys
  const nu = new UF();
  for (const p of pads) nu.find(p.key);
  for (const cn of S.conns) if (padByKey[cn.a] && padByKey[cn.b]) nu.union(cn.a, cn.b);
  // pads with the same name on one part are joined internally (e.g. many GND pins on a Pico)
  const byCompName = {};
  for (const p of pads) { if (internallyLinked(p)) { const k = p.comp.id + "|" + p.name; if (byCompName[k]) nu.union(byCompName[k], p.key); else byCompName[k] = p.key; } }
  // collect net groups (only pads that are part of some connection or internally linked group count)
  const connected = new Set(); for (const cn of S.conns) { connected.add(nu.find(cn.a)); }
  const groups = {};
  for (const p of pads) { const r = nu.find(p.key); (groups[r] = groups[r] || []).push(p); }
  const nets = []; const netOfPad = {};
  let auto = 1;
  for (const r in groups) {
    const ps = groups[r];
    if (!connected.has(r)) continue;
    let best = [9, null];
    for (const p of ps) { const pr = netPriority(p.name); if (pr[0] < best[0]) best = pr; }
    let name = best[1];
    if (S.netLabels) for (const p of ps) { const lb = S.netLabels[p.key]; if (lb && (!name || netPriority(lb)[0] < 9)) { name = lb; break; } }
    if (!name) { const bus = ps.map(p => p.func).find(f => /^(SDA|SCL|MOSI|MISO|SCK)$/.test(f || "")); const nonBus = ps.some(p => p.func && !/^(SDA|SCL|MOSI|MISO|SCK|SIG|ADC|CS)$/.test(p.func)); if (bus && !nonBus) name = bus; }
    if (!name) { const lead = ps.slice().sort((a, b) => (compFP(b.comp).mcu ? 1 : 0) - (compFP(a.comp).mcu ? 1 : 0))[0]; name = (compFP(lead.comp).pinLabels ? lead.name : lead.comp.ref + "_" + lead.name); }
    nets.push({ root: r, name, pads: ps, idx: 0 });
  }
  // de-duplicate names
  const seen = {};
  nets.sort((a, b) => netPriority(a.name)[0] - netPriority(b.name)[0] || a.name.localeCompare(b.name));
  for (const n of nets) { if (seen[n.name]) { seen[n.name]++; n.name = n.name + "_" + seen[n.name]; } else seen[n.name] = 1; }
  nets.forEach((n, i) => { n.idx = i; n.color = n.name === "GND" ? "#a0aec0" : n.name === "5V" ? "#fc8181" : n.name === "3V3" ? "#f6ad55" : NET_COLORS[i % NET_COLORS.length]; for (const p of n.pads) netOfPad[p.key] = n; });
  const netByName = {}; for (const n of nets) netByName[n.name] = n;

  // copper items for connectivity
  const items = [];
  for (const p of pads) items.push({ kind: "pad", ref: p, shape: padShape(p), layers: p.layers, net: netOfPad[p.key] || null });
  for (const t of S.tracks) for (let i = 0; i < t.pts.length - 1; i++) { const a = t.pts[i], b = t.pts[i + 1]; items.push({ kind: "seg", ref: t, i, shape: { t: "cap", ax: a[0], ay: a[1], bx: b[0], by: b[1], r: t.w / 2 }, layers: [t.layer], net: null }); }
  for (const v of S.vias) items.push({ kind: "via", ref: v, shape: { t: "cap", ax: v.x, ay: v.y, bx: v.x, by: v.y, r: v.d / 2 }, layers: ["top", "bottom"], net: null });
  items.forEach((it, i) => { it.id = i; it.box = shapeBox(it.shape); });
  // sweep for overlaps
  const cu = new UF(); items.forEach(it => cu.find(it.id));
  const order = items.slice().sort((a, b) => a.box[0] - b.box[0]);
  const pairs = []; // near pairs for DRC
  const maxGap = Math.max(S.board.clear, S.board.pourClear) + 0.05;
  for (let i = 0; i < order.length; i++) {
    const A = order[i];
    for (let j = i + 1; j < order.length; j++) {
      const B = order[j];
      if (B.box[0] > A.box[2] + maxGap) break;
      if (B.box[1] > A.box[3] + maxGap || B.box[3] < A.box[1] - maxGap) continue;
      if (!A.layers.some(l => B.layers.includes(l))) continue;
      if (A.kind === "pad" && B.kind === "pad" && A.ref.comp === B.ref.comp && A.ref.name === B.ref.name) { continue; }
      const d = shapeDist(A.shape, B.shape);
      if (d <= 0.001) cu.union(A.id, B.id); else if (d < maxGap) pairs.push([A, B, d]);
    }
  }
  // internal links (same-name pins of a module) count as connected
  for (const k in byCompName) { /* handled through netlist; copper islands joined below */ }
  const padItem = {}; items.forEach(it => { if (it.kind === "pad") padItem[it.ref.key] = it; });
  for (const p of pads) { const k = p.comp.id + "|" + p.name; if (byCompName[k] && byCompName[k] !== p.key) cu.union(padItem[p.key].id, padItem[byCompName[k]].id); }
  // pour joins all same-net items on the pour layer
  const pourLayers = []; if (S.board.pourTop) pourLayers.push("top"); if (S.board.pourBot) pourLayers.push("bottom");
  // cluster net assignment
  const clusterNets = {};
  for (const it of items) if (it.net) { const r = cu.find(it.id); (clusterNets[r] = clusterNets[r] || new Set()).add(it.net); }
  const pourNet = netByName[S.board.pourNet];
  // While the auto-router works with "route the fill net too" on, the fill
  // doesn't count, so ground gets real tracks (the fill stays as a backup).
  const routingThroughFill = typeof ROUTER !== "undefined" && ROUTER.noPour;
  if (pourNet && pourLayers.length && !routingThroughFill) {
    const isPour = it => { const ns = clusterNets[cu.find(it.id)]; return !!ns && ns.size === 1 && ns.has(pourNet); };
    for (const layer of pourLayers) pourLink(layer, items, holes, isPour, cu);
  }
  // recompute after pour merge
  const cNets = {};
  for (const it of items) if (it.net) { const r = cu.find(it.id); (cNets[r] = cNets[r] || new Set()).add(it.net); }
  const shorts = [];
  for (const it of items) { const ns = cNets[cu.find(it.id)]; it.cnet = ns && ns.size === 1 ? [...ns][0] : null; it.short = ns && ns.size > 1; }
  const shortSeen = new Set();
  for (const r in cNets) if (cNets[r].size > 1) { const names = [...cNets[r]].map(n => n.name).sort().join(" + "); if (!shortSeen.has(names)) { shortSeen.add(names); const it = items.find(x => String(cu.find(x.id)) === r && x.net); shorts.push({ names, x: it ? it.ref.x ?? it.shape.ax : 0, y: it ? it.ref.y ?? it.shape.ay : 0 }); } }
  // ratsnest per net
  const rats = []; let unrouted = 0, routedNets = 0;
  for (const n of nets) {
    const isl = {};
    for (const p of n.pads) { const r = cu.find(padItem[p.key].id); (isl[r] = isl[r] || []).push(p); }
    const groupsL = Object.values(isl);
    n.islands = groupsL.length;
    if (groupsL.length <= 1) { routedNets++; continue; }
    unrouted += groupsL.length - 1;
    // Prim MST between islands using closest pad pairs
    const inT = [groupsL[0]]; const rest = groupsL.slice(1);
    while (rest.length) {
      let best = null;
      for (const a of inT) for (let j = 0; j < rest.length; j++) for (const pa of a) for (const pb of rest[j]) { const d = dist(pa.x, pa.y, pb.x, pb.y); if (!best || d < best.d) best = { d, j, pa, pb }; }
      rats.push({ net: n, a: best.pa, b: best.pb });
      inT.push(rest[best.j]); rest.splice(best.j, 1);
    }
  }
  // net of each track
  const trackNet = {}, viaNet = {};
  for (const it of items) { if (it.kind === "seg" && it.cnet) trackNet[it.ref.id] = it.cnet; if (it.kind === "via" && it.cnet) viaNet[it.ref.id] = it.cnet; }
  CACHE = { pads, padByKey, holes, nets, netByName, netOfPad, items, pairs, rats, unrouted, routedNets, shorts, trackNet, viaNet, cu };
  return CACHE;
}

// ---------------- DRC ----------------
function runDRC() {
  const A = analyze(), B = S.board, out = [];
  for (const [a, b, d] of A.pairs) {
    if (a.cnet && b.cnet && a.cnet === b.cnet) continue;
    if (!a.cnet && !b.cnet && a.kind === "pad" && b.kind === "pad") { if (d < 0.1) out.push({ sev: "warn", code: "close", a, b, d, msg: `Pads very close: ${a.ref.comp.ref}.${a.ref.name} / ${b.ref.comp.ref}.${b.ref.name}`, x: a.ref.x, y: a.ref.y }); continue; }
    if (d < B.clear - 0.005) { const p = a.kind === "pad" ? a.ref : b.kind === "pad" ? b.ref : null; const x = p ? p.x : (a.shape.ax + a.shape.bx) / 2, y = p ? p.y : (a.shape.ay + a.shape.by) / 2; out.push({ sev: "err", code: "clear", a, b, d, msg: `Too close (${d.toFixed(2)}mm < ${B.clear}mm): ${desc(a)} ↔ ${desc(b)}`, x, y }); }
  }
  for (const s of A.shorts) out.push({ sev: "err", code: "short", msg: `Short circuit between nets ${s.names}`, x: s.x, y: s.y });
  for (const r of A.rats) out.push({ sev: "warn", code: "rat", msg: `Not routed yet: ${r.net.name} (${r.a.comp.ref}.${r.a.name} → ${r.b.comp.ref}.${r.b.name})`, x: (r.a.x + r.b.x) / 2, y: (r.a.y + r.b.y) / 2 });
  // board edge
  const custom = (B.polys && B.polys.length) || (B.cutouts && B.cutouts.length);
  const edgeBad = (it, e) => { // custom outlines: test the copper's real shape, not its bounding box
    const s = it.shape, nc = it.kind === "pad" && compFP(it.ref.comp).cutout; // reverse-mount LEDs sit in their cut-out on purpose
    if (s.t === "poly") return s.pts.some(([x, y]) => !insideBoard(x, y, e, nc));
    if (s.t === "rect") return [[-1, -1], [1, -1], [1, 1], [-1, 1]].some(([a, b]) => !insideBoard(s.cx + a * s.hw, s.cy + b * s.hh, e, nc));
    const L = Math.hypot(s.bx - s.ax, s.by - s.ay), n = Math.max(1, Math.ceil(L / 0.4));
    for (let i = 0; i <= n; i++) if (!insideBoard(s.ax + (s.bx - s.ax) * i / n, s.ay + (s.by - s.ay) * i / n, e + s.r - 0.02)) return true;
    return false;
  };
  for (const it of A.items) { const bx = it.box; const e = B.edgeClear; if (custom ? edgeBad(it, e) : (!insideBoard(bx[0] + 0, bx[1], e) || !insideBoard(bx[2], bx[3], e) || !insideBoard(bx[0], bx[3], e) || !insideBoard(bx[2], bx[1], e))) out.push({ sev: it.kind === "pad" ? "err" : "warn", code: "edge", it, msg: `Copper too close to / outside the board edge: ${desc(it)}`, x: (bx[0] + bx[2]) / 2, y: (bx[1] + bx[3]) / 2 }); }
  for (const h of A.holes) for (const it of A.items) { if (Math.abs(it.box[0] - h.x) > 20) continue; const d = shapeDist(it.shape, { t: "cap", ax: h.x, ay: h.y, bx: h.x, by: h.y, r: h.d / 2 }); if (d < B.clear) out.push({ sev: "err", code: "hole", it, h, d, msg: `Copper too close to mounting hole: ${desc(it)}`, x: h.x, y: h.y }); }
  // overlapping parts (pads of different parts overlapping)
  const ver = {};
  for (const c of S.comps) { const fp = compFP(c); if (!fp) out.push({ sev: "err", code: "unknown", msg: `Unknown part ${c.ref}`, x: c.x, y: c.y }); else if (fp.verify) (ver[fp.id] = ver[fp.id] || []).push(c); }
  for (const cs of Object.values(ver)) { const c = cs[0], fp = compFP(c); out.push({ sev: "info", code: "verify", msg: `${cs.length > 3 ? cs.slice(0, 3).map(q => q.ref).join(", ") + ` and ${cs.length - 3} more` : cs.map(q => q.ref).join(", ")} (${fp.name}): check pin positions against your actual part before ordering`, x: c.x, y: c.y }); }
  for (const t of S.tracks) if (t.w < 0.15) out.push({ sev: "warn", code: "thin", t, msg: `Track thinner than 0.15mm may be hard to manufacture`, x: t.pts[0][0], y: t.pts[0][1] });
  const sev = { err: 0, warn: 1, info: 2 }; out.sort((a, b) => sev[a.sev] - sev[b.sev]);
  return out;
}
function desc(it) { if (it.kind === "pad") return `${it.ref.comp.ref} pin ${it.ref.name}`; if (it.kind === "via") return "via"; return `track${it.cnet ? " (" + it.cnet.name + ")" : ""}`; }
// Custom outlines (keyboards, split boards): S.board.polys is a list of
// polygons in mm (y down, all coordinates >= 0); B.w/B.h are their bounds.
function polyArea(pts) { let a = 0; for (let i = 0; i < pts.length; i++) { const p = pts[i], q = pts[(i + 1) % pts.length]; a += p[0] * q[1] - q[0] * p[1]; } return a / 2; }
function ptInPoly(x, y, pts) { let inside = false; for (let i = 0, j = pts.length - 1; i < pts.length; j = i++) { const [xi, yi] = pts[i], [xj, yj] = pts[j]; if ((yi > y) !== (yj > y) && x < (xj - xi) * (y - yi) / (yj - yi) + xi) inside = !inside; } return inside; }
function ptPolyEdgeDist(x, y, pts) { let d = 1e9; for (let i = 0; i < pts.length; i++) { const p = pts[i], q = pts[(i + 1) % pts.length]; d = Math.min(d, ptSegDist(x, y, p[0], p[1], q[0], q[1])); } return d; }
// move every edge inwards by d mm (negative = outwards)
function offsetPoly(pts, d) {
  if (!d) return pts.map(p => p.slice());
  const s = polyArea(pts) > 0 ? 1 : -1, n = pts.length, N = [];
  for (let i = 0; i < n; i++) { const a = pts[i], b = pts[(i + 1) % n]; const L = Math.hypot(b[0] - a[0], b[1] - a[1]) || 1; N.push([-(b[1] - a[1]) / L * s, (b[0] - a[0]) / L * s]); }
  return pts.map((p, i) => { const n1 = N[(i - 1 + n) % n], n2 = N[i]; const k = Math.min(3, 1 / Math.max(0.2, (1 + n1[0] * n2[0] + n1[1] * n2[1]) / 2)) / 2; return [p[0] + (n1[0] + n2[0]) * k * d, p[1] + (n1[1] + n2[1]) * k * d]; });
}
function boardPolys(inset = 0) { const B = S.board; return B.polys && B.polys.length ? B.polys.map(p => offsetPoly(p, inset)) : [boardPolygon(inset)]; }
// board shape as a canvas path; cut-outs (e.g. for reverse-mount LEDs) are wound the other way so they stay empty
function pathBoard(ctx, inset = 0) {
  ctx.beginPath();
  const add = pts => { pts.forEach((p, i) => (i ? ctx.lineTo(p[0], p[1]) : ctx.moveTo(p[0], p[1]))); ctx.closePath(); };
  for (const pts of boardPolys(inset)) add(polyArea(pts) > 0 ? pts : pts.slice().reverse());
  for (const c of S.board.cutouts || []) { const g = offsetPoly(c, -inset); add(polyArea(g) > 0 ? g.slice().reverse() : g); }
}

function insideBoard(x, y, m = 0, noCut = false) {
  const B = S.board;
  if (!noCut && B.cutouts && B.cutouts.length && B.cutouts.some(c => ptInPoly(x, y, c) || ptPolyEdgeDist(x, y, c) < m)) return false;
  if (B.polys && B.polys.length) return B.polys.some(pts => ptInPoly(x, y, pts) && ptPolyEdgeDist(x, y, pts) >= m - 1e-6);
  if (B.round) { const r = Math.min(B.w, B.h) / 2; return dist(x, y, B.w / 2, B.h / 2) <= r - m; }
  if (x < m - 1e-6 || y < m - 1e-6 || x > B.w - m + 1e-6 || y > B.h - m + 1e-6) return false;
  const r = B.r || 0;
  if (r <= m) return true;
  const cx = x < r ? r : x > B.w - r ? B.w - r : null, cy = y < r ? r : y > B.h - r ? B.h - r : null;
  if (cx === null || cy === null) return true;
  return dist(x, y, cx, cy) <= r - m + 1e-6;
}
// polygon for the board outline (optionally inset), y-down, mm
function boardPolygon(inset = 0) {
  const B = S.board, pts = [];
  if (B.polys && B.polys.length) return offsetPoly(B.polys[0], inset);
  if (B.round) { const R = Math.min(B.w, B.h) / 2 - inset; for (let i = 0; i < 96; i++) { const a = i / 96 * Math.PI * 2; pts.push([B.w / 2 + R * Math.cos(a), B.h / 2 + R * Math.sin(a)]); } return pts; }
  const r = Math.max(0, (B.r || 0) - inset), x0 = inset, y0 = inset, x1 = B.w - inset, y1 = B.h - inset;
  if (r < 0.01) return [[x0, y0], [x1, y0], [x1, y1], [x0, y1]];
  const arc = (cx, cy, a0) => { for (let i = 0; i <= 8; i++) { const a = a0 + i / 8 * Math.PI / 2; pts.push([cx + r * Math.cos(a), cy + r * Math.sin(a)]); } };
  arc(x1 - r, y0 + r, -Math.PI / 2); arc(x1 - r, y1 - r, 0); arc(x0 + r, y1 - r, Math.PI / 2); arc(x0 + r, y0 + r, Math.PI);
  return pts;
}

// ---------------- reference designators ----------------
function nextRef(prefix) {
  let n = 1; const used = new Set(S.comps.map(c => c.ref));
  while (used.has(prefix + n)) n++;
  return prefix + n;
}
function addComp(fpId, x, y, o = {}) {
  const fp = LIB_BY_ID[fpId]; if (!fp) return null;
  const c = { id: uid(), fp: fpId, ref: o.ref || nextRef(fp.prefix), value: o.value || fp.value, x, y, rot: o.rot || 0, side: o.side || "top", silkPins: false, hideRef: fp.cat === "Mechanical" };
  S.comps.push(c); invalidate();
  return c;
}
function removeComp(id) {
  S.comps = S.comps.filter(c => c.id !== id);
  S.conns = S.conns.filter(cn => !cn.a.startsWith(id + ":") && !cn.b.startsWith(id + ":"));
  delete S.strip.place[id];
  invalidate();
}
function addConn(a, b) {
  if (a === b) return false;
  const A = analyze();
  if (A.netOfPad[a] && A.netOfPad[a] === A.netOfPad[b]) return false;
  S.conns.push({ a, b }); invalidate(); return true;
}
function disconnectPad(key) { S.conns = S.conns.filter(cn => cn.a !== key && cn.b !== key); invalidate(); }
