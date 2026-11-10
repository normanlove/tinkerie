// Keebie — custom keyboard PCB designer inside Wirie.
// Lay out keys (any size, any angle), shape the board (fit, hug, custom,
// arc, split and splayed), pick a switch type and controller, add knobs and a
// screen, and Keebie builds the PCB: switches, diodes, stabilisers, per-key RGB, the
// controller and all its support circuitry, matrix wiring, outline and
// mounting holes. Also exports a switch plate, QMK/ZMK files and KLE layouts.
// Part of Wirie, in the Tinkerie maker suite — by nrmluvdev.
"use strict";

const KV = { zoom: 0, vx: 0, vy: 0, sel: new Set(), drag: null, view: "legend", geomKey: "", geom: null, built: false, tool: "select" };
let kbUid = 1;
const kbId = () => "k" + (Date.now() % 1e6).toString(36) + (kbUid++).toString(36);

// ---------------------------------------------------------------- model
function kbDefault() {
  return { v: 1, name: "My keyboard", keys: kbFromPreset("ansi60"), preset: "ansi60", sw: "mx_hs", profile: "cherry", spacing: 0, fpRot: 0,
    outline: { mode: "hug", margin: 2, radius: 2, custom: null }, holes: 6, arc: 0,
    split: { mode: "none", at: 7.5, gap: 30, angle: 0 }, mcu: "promicro", ctrlPos: "right", diode: "d_sod123", dir: "COL2ROW", rgb: false, oled: "none", half: 0,
    dstyle: "beside", dside: "bottom", ledCap: false, pack: "compact", ctrlOff: [] };
}
function KBS() { if (!S.keeb) S.keeb = kbDefault(); return S.keeb; }

function kbParseRows(rows) {
  const keys = []; let y = 0;
  for (const row of rows) {
    const spec = typeof row === "string" ? { k: row } : row;
    if (spec.y !== undefined) y = spec.y;
    let x = 0;
    for (const tok of spec.k.split(/\s+/).filter(Boolean)) {
      if (tok[0] === "+" && tok.length > 1 && !isNaN(+tok.slice(1))) { x += +tok.slice(1); continue; }
      if (tok[0] === "@" && tok.length > 1 && !isNaN(+tok.slice(1))) { x = +tok.slice(1); continue; }
      const [lab, w = "1", h = "1", shape = ""] = tok.split("|");
      const k = { id: kbId(), x, y, w: +w, h: +h, r: 0, label: lab === "_" ? "" : lab.replace(/~/g, " "), shape, fr: 0, row: null, col: null, stab: "auto" };
      if (shape === "iso") k.x += 0.25; // ISO enter: the switch sits under the lower, narrower part
      keys.push(k); x += +w + (shape === "iso" ? 0.25 : 0);
    }
    y += 1;
  }
  return keys;
}
function kbErgo(o) {
  const cols = o.cols, rows = o.rows, st = o.stagger, left = [];
  const LBL = rows === 4 ? [KB_ERGO_LABELS[3], ...KB_ERGO_LABELS.slice(0, 3)] : KB_ERGO_LABELS, LBR = rows === 4 ? [KB_ERGO_LABELS_R[3], ...KB_ERGO_LABELS_R.slice(0, 3)] : KB_ERGO_LABELS_R;
  for (let r = 0; r < rows; r++) for (let c = 0; c < cols; c++) left.push({ x: c, y: r + (st[c] || 0), w: 1, h: 1, r: 0, label: (LBL[r] || [])[c + (6 - cols)] || "", rr: r, cc: c });
  const tl = ["Alt", "Lower", "Space", "Ctrl"].slice(0, o.thumbs), tr = ["Enter", "Raise", "Bksp", "Alt"].slice(0, o.thumbs);
  const base = rows + (st[cols - 1] || 0) * 0.5 + 0.25;
  for (let t = 0; t < o.thumbs; t++) {
    const tall = t === o.thumbs - 1 && o.thumbs === 3;
    left.push({ x: cols - o.thumbs + t + 0.6 + t * 0.08, y: base + t * t * 0.12 - (tall ? 0.25 : 0), w: 1, h: tall ? 1.5 : 1, r: t * 7, label: tl[t], thumb: t });
  }
  const Lmax = Math.max(...left.map(k => k.x + k.w)), W = Lmax + 2.5, keys = [];
  for (const k of left) keys.push({ x: k.x, y: k.y, w: k.w, h: k.h, r: k.r, label: k.label });
  for (const k of left) {
    const label = k.thumb !== undefined ? tr[k.thumb] : (LBR[k.rr] || [])[cols - 1 - k.cc + (6 - cols)] || "";
    keys.push({ x: +(W + (Lmax - (k.x + k.w))).toFixed(3), y: k.y, w: k.w, h: k.h, r: -k.r, label });
  }
  return { keys: keys.map(k => Object.assign({ id: kbId(), shape: "", fr: 0, row: null, col: null, stab: "none" }, k)), at: +(Lmax + 1.25).toFixed(2) };
}
function kbFromPreset(id) {
  const p = KB_PRESETS.find(q => q.id === id); if (!p) return [];
  if (p.ergo) return kbErgo(p.ergo).keys;
  return kbParseRows(p.rows);
}
function kbLoadPreset(id) {
  const K = KBS(), p = KB_PRESETS.find(q => q.id === id); if (!p) return;
  K.preset = id; K.name = p.name.replace(/\s*\(.*\)/, "");
  if (p.ergo) { const e = kbErgo(p.ergo); K.keys = e.keys; K.split = { mode: "two", at: e.at, gap: 30, angle: 0 }; K.ctrlPos = "side"; }
  else { K.keys = kbParseRows(p.rows); K.split = p.split ? { ...p.split } : { mode: "none", at: 7.5, gap: 30, angle: 0 }; if (K.ctrlPos === "side") K.ctrlPos = p.split ? "centre" : "right"; if (p.split) K.ctrlPos = "centre"; }
  K.outline.custom = null; KV.sel.clear(); KV.zoom = 0;
}
function kbGrid(rows, cols) { const keys = []; for (let r = 0; r < rows; r++) for (let c = 0; c < cols; c++) keys.push({ id: kbId(), x: c, y: r, w: 1, h: 1, r: 0, label: "", shape: "", fr: 0, row: null, col: null, stab: "auto" }); return keys; }

// ---------------------------------------------------------------- KLE import / export
function kbFromKLE(text) {
  const t = text.trim().replace(/,\s*$/, "");
  let data; try { data = JSON.parse(t); if (Array.isArray(data) && data.length && !Array.isArray(data[0]) && typeof data[0] !== "object") data = [data]; } catch { data = JSON.parse("[" + t + "]"); }
  if (Array.isArray(data) && data.every(r => !Array.isArray(r)) && data.some(r => typeof r === "string")) data = [data];
  const keys = []; let r = 0, rx = 0, ry = 0, y = 0, x = 0;
  for (const row of data) {
    if (!Array.isArray(row)) continue;
    let w = 1, h = 1, iso = false;
    for (const item of row) {
      if (typeof item === "object" && item) {
        if (item.r !== undefined) r = item.r;
        if (item.rx !== undefined) { rx = item.rx; x = rx; y = ry; }
        if (item.ry !== undefined) { ry = item.ry; x = rx; y = ry; }
        if (item.x) x += item.x; if (item.y) y += item.y;
        if (item.w) w = item.w; if (item.h) h = item.h;
        if (item.w2 && item.h2 && (item.h || 1) >= 2 && item.w2 > (item.w || 1)) iso = true;
        continue;
      }
      // KLE rotates about (rx, ry); Keebie rotates each key about its own centre
      const a = r * Math.PI / 180, c0x = x + w / 2, c0y = y + h / 2;
      const cx = rx + (c0x - rx) * Math.cos(a) - (c0y - ry) * Math.sin(a), cy = ry + (c0x - rx) * Math.sin(a) + (c0y - ry) * Math.cos(a);
      const label = String(item).split("\n").filter(Boolean).pop() || "";
      keys.push({ id: kbId(), x: +(cx - w / 2).toFixed(4), y: +(cy - h / 2).toFixed(4), w, h, r, label: label.replace(/<[^>]+>/g, "").slice(0, 12), shape: iso ? "iso" : "", fr: 0, row: null, col: null, stab: "auto" });
      x += w; w = 1; h = 1; iso = false;
    }
    y += 1; x = rx;
  }
  if (!keys.length) throw new Error("no keys found");
  const mx = Math.min(...keys.map(k => k.x)), my = Math.min(...keys.map(k => k.y));
  for (const k of keys) { k.x = +(k.x - mx).toFixed(4); k.y = +(k.y - my).toFixed(4); }
  return keys;
}
function kbToKLE() {
  const K = KBS(), rows = [];
  for (const k of K.keys) {
    const cx = k.x + k.w / 2, cy = k.y + k.h / 2;
    const p = { r: k.r || 0, rx: +cx.toFixed(4), ry: +cy.toFixed(4), x: -k.w / 2, y: -k.h / 2 };
    if (k.w !== 1) p.w = k.w; if (k.h !== 1) p.h = k.h;
    if (k.shape === "iso") Object.assign(p, { w2: 1.5, h2: 1, x2: -0.25 });
    rows.push([p, k.label || ""]);
  }
  return JSON.stringify([{ name: K.name, author: "Made with Keebie in Wirie (Tinkerie, by nrmluvdev)" }, ...rows], null, 1);
}

// ---------------------------------------------------------------- geometry
function kbSwitch(K) { return KB_SWITCHES.find(s => s.id === K.sw) || KB_SWITCHES[0]; }
function kbPitch(K) { const s = kbSwitch(K); return K.spacing > 0 ? [K.spacing, K.spacing * s.pitch[1] / s.pitch[0]] : s.pitch; }
function kbMcu(K) { return KB_MCU[K.mcu] || KB_MCUS[0]; }

// where every key ends up, in mm, after arc and split
function kbPlace(K) {
  const [sx, sy] = kbPitch(K), out = [];
  if (!K.keys.length) return out;
  const xs = K.keys.map(k => k.x + k.w / 2), mid = (Math.min(...xs) + Math.max(...xs)) / 2;
  const maxY = Math.max(...K.keys.map(k => k.y + k.h));
  const sp = K.split, split = sp.mode !== "none";
  for (const k of K.keys) {
    const bcx = k.x + k.w / 2, bcy = k.y + k.h / 2;
    let dy = 0, ang = k.r || 0;
    if (K.arc) { const d = bcx - mid; dy -= d * d / (2 * K.arc); ang += Math.atan(-d / K.arc * sy / sx) * 180 / Math.PI; }
    let cx = bcx * sx, cy = (bcy + dy) * sy;
    const half = split && bcx >= sp.at ? 1 : 0;
    if (split && (sp.mode === "angled" || sp.mode === "two")) {
      const px = sp.at * sx, py = maxY * sy;
      if (half) cx += sp.gap || 0;
      const a = (half ? 1 : -1) * (sp.angle || 0), ox = half ? px + (sp.gap || 0) : px;
      if (a) { const [rx, ry] = rotPt(cx - ox, cy - py, a); cx = ox + rx; cy = py + ry; ang += a; }
    }
    out.push({ k, cx, cy, ang, half, w: k.w, h: k.h, bcx, bcy });
  }
  return out;
}
function kbCorners(p, sx, sy, grow = 0) {
  const hw = p.w * sx / 2 + grow, hh = p.h * sy / 2 + grow;
  return [[-hw, -hh], [hw, -hh], [hw, hh], [-hw, hh]].map(([x, y]) => { const [rx, ry] = rotPt(x, y, p.ang); return [p.cx + rx, p.cy + ry]; });
}
function kbISOPoly(p, sx, sy, grow = 0) { // ISO enter cap outline
  const hw = p.w * sx / 2, hh = p.h * sy / 2, ext = 0.25 * sx;
  return [[-hw - ext - grow, -hh - grow], [hw + grow, -hh - grow], [hw + grow, hh + grow], [-hw - grow, hh + grow], [-hw - grow, 0], [-hw - ext - grow, 0]].map(([x, y]) => { const [rx, ry] = rotPt(x, y, p.ang); return [p.cx + rx, p.cy + ry]; });
}

// matrix: rows by height, columns by order within the row; folded if it needs too many pins
function kbMatrix(K, places) {
  const mcu = kbMcu(K), out = new Map();
  const halves = K.split.mode === "two" ? [0, 1] : [0];
  let R = 0, C = 0;
  for (const h of halves) {
    const ps = places.filter(p => p.half === h || K.split.mode !== "two").slice().sort((a, b) => a.bcy - b.bcy || a.bcx - b.bcx);
    const rows = []; let cur = null;
    for (const p of ps) { if (!cur || p.bcy - cur.y > 0.55) { cur = { y: p.bcy, ps: [] }; rows.push(cur); } cur.ps.push(p); }
    let asg = [];
    rows.forEach((row, r) => row.ps.sort((a, b) => a.bcx - b.bcx).forEach((p, c) => asg.push([p, r, c])));
    let nr = rows.length, nc = Math.max(0, ...rows.map(r => r.ps.length));
    const avail = kbFreePins(K).length;
    // fold: split the columns into two halves stacked as extra rows while it saves pins
    while (nr + nc > avail && nc > 2 && nr * 2 + Math.ceil(nc / 2) < nr + nc) { const hc = Math.ceil(nc / 2); asg = asg.map(([p, r, c]) => [p, c >= hc ? r + nr : r, c % hc]); nr *= 2; nc = hc; }
    for (const [p, r, c] of asg) out.set(p.k.id, [p.k.row ?? r, p.k.col ?? c]);
    for (const p of ps) { const [r, c] = out.get(p.k.id); R = Math.max(R, r + 1); C = Math.max(C, c + 1); }
  }
  // duplicates (from manual overrides)
  const seen = {}, dup = [];
  for (const p of places) { const m = out.get(p.k.id); const key = (K.split.mode === "two" ? p.half : 0) + ":" + m; if (seen[key]) dup.push(p.k); seen[key] = 1; }
  return { pos: out, rows: R, cols: C, dup };
}
// pins the matrix may use, once the split link, LEDs, screen and knobs have theirs
function kbBasePins(K) {
  const m = kbMcu(K), split = K.split.mode === "two" && !m.wireless;
  return m.pins.filter(p => !(split && p === m.serial) && !(K.rgb && p === m.led));
}
function kbKnobCount(K) {
  const ks = K.keys.filter(k => k.kind === "knob"); if (K.split.mode !== "two") return ks.length;
  const r = ks.filter(k => k.x + k.w / 2 >= K.split.at).length; return Math.max(r, ks.length - r);
}
function kbPinPlan(K) {
  const m = kbMcu(K); let pins = kbBasePins(K);
  const scr = K.oled && K.oled !== "none";
  const i2c = scr ? (m.ard ? m.ard.i2c : m.id === "nicenano" ? ["D2", "D3"] : ["D4", "D5"]) : null;
  if (i2c) pins = pins.filter(p => !i2c.includes(p));
  const enc = []; for (let i = 0; i < kbKnobCount(K) && pins.length >= 2; i++) enc.push(pins.splice(pins.length - 2, 2));
  return { pins, i2c, enc };
}
function kbFreePins(K) { return kbPinPlan(K).pins; }
// keys in wiring order (snakes through the matrix rows; the LED chain follows it)
function kbOrd(G, b) { const M = G.matrix; return b.places.slice().sort((p, q) => { const a = M.pos.get(p.k.id), c = M.pos.get(q.k.id); return a[0] - c[0] || (a[0] % 2 ? c[1] - a[1] : a[1] - c[1]); }); }

// ---------------------------------------------------------------- controller block
// Plans the controller area in local coordinates (0,0 = area top-left) by
// recording the parts and nets the controller needs, then packing them.
// the controller plan only depends on a few settings: remember it between redraws
const KB_PLAN_CACHE = new Map();
function kbPlanCtrl(K, half) {
  const key = JSON.stringify([K.mcu, K.outline.margin, K.rgb, K.oled, K.split.mode, half || 0, kbKnobCount(K), kbPinPlan(K), K.pack || "compact"]);
  let plan = KB_PLAN_CACHE.get(key);
  if (!plan) { plan = kbPlanCtrl0(K, half); if (KB_PLAN_CACHE.size > 30) KB_PLAN_CACHE.clear(); KB_PLAN_CACHE.set(key, plan); }
  // hand back a copy (callers move the parts), keeping the nets pointing at the copied parts
  const map = new Map(plan.parts.map(q => [q, Object.assign({}, q)]));
  return Object.assign({}, plan, { parts: plan.parts.map(q => map.get(q)), nets: plan.nets.map(([n, pins]) => [n, pins.map(pin => Array.isArray(pin) && map.has(pin[0]) ? [map.get(pin[0]), ...pin.slice(1)] : pin)]) });
}
function kbPlanCtrl0(K, half) {
  const mcu = kbMcu(K), parts = [], nets = [], margin = K.outline.margin;
  const B = {
    add(fp, value, ref) { const p = { fp, value, ref: ref || null, x: 0, y: 0, rot: 0, side: "top", id: "p" + parts.length, fixed: false }; parts.push(p); return p; },
    net(name, ...pins) { nets.push([name, pins]); },
    usb() {
      const j = B.add("usbc16", "USB-C", "J_USB"); j.fixed = "usb";
      const f = B.add("kb_fuse1206", "500mA PTC", "F1"), r1 = B.add("r_0805", "5.1k"), r2 = B.add("r_0805", "5.1k");
      B.net("VBUS", [j, "VBUS"], [f, "1"]); B.net("5V", [f, "2"]); B.net("GND", [j, "GND"], [j, "SHIELD"], [r1, "2"], [r2, "2"]);
      B.net("CC1", [j, "CC1"], [r1, "1"]); B.net("CC2", [j, "CC2"], [r2, "1"]);
      return { dp: [[j, "DP1"], [j, "DP2"]], dn: [[j, "DN1"], [j, "DN2"]] };
    },
  };
  const U = B.add(mcu.fp, mcu.bare ? mcu.name.replace(" chip", "") : LIB_BY_ID[mcu.fp].value, "U1"); U.fixed = "mcu";
  let vcc = mcu.pwr;
  if (mcu.bare) vcc = kbSupport(mcu, B, U).vcc;
  else {
    if (mcu.reset) { const s = B.add("sw4x4smd", "Reset", "SW_RST"); B.net("RESET", [U, mcu.reset[0]], [s, "A"]); B.net("GND", [U, mcu.reset[1]], [s, "B"]); }
    if (mcu.id === "nicenano") { const j = B.add("jstph2", "Battery", "J_BAT"), s = B.add("slide_smd", "Power", "SW_PWR"); B.net("BAT+", [j, "1"], [s, "A"]); B.net("RAW", [s, "COM"], [U, "RAW"]); B.net("GND", [j, "2"], [U, "GND"]); }
  }
  const split = K.split.mode === "two" && !mcu.wireless;
  if (split) { const t = B.add("kb_trrs", "TRRS", "J_TRRS"); t.fixed = "trrs"; B.net("GND", [t, "SLEEVE"], [U, /^stm32/.test(mcu.id) ? "VSS" : "GND"]); B.net("SERIAL", [t, "TIP"], [U, mcu.serial]); B.net(mcu.bare ? "5V" : "VCC_LINK", [t, "RING1"], ...(mcu.bare ? [] : [[U, mcu.logic]])); }
  const scr = KB_SCREENS.find(x => x.id === K.oled);
  if (scr && scr.w) {
    const pp = kbPinPlan(K), o = B.add(scr.id, `OLED ${scr.w}x${scr.h}`, "DISP1");
    B.net("SDA", [o, "SDA"], [U, pp.i2c[0]]); B.net("SCL", [o, "SCL"], [U, pp.i2c[1]]); B.net("GND", [o, "GND"]);
    if (mcu.bare) B.net(vcc, [o, "VCC"]); else B.net("OLED_VCC", [o, "VCC"], [U, mcu.logic]);
  }
  if (K.rgb) { const c = B.add("c_0805", "10uF"); B.net(mcu.bare ? "5V" : "LED_5V", [c, "1"], ...(mcu.bare ? [] : [[U, mcu.pwr]])); B.net("GND", [c, "2"]); }
  // ---- pack
  const fpBox = (p, rot) => { const c = { fp: p.fp, x: 0, y: 0, rot, side: "top" }; return compBox(c); };
  const placed = [];
  const W0 = mcu.bare ? 40 : Math.max(LIB_BY_ID[mcu.fp].body.w + 16, split ? 36 : 28);
  const W = Math.max(W0, scr && scr.w ? LIB_BY_ID[scr.id].body.w + 3 : 0);
  const put = (p, x, y, rot) => { p.x = x; p.y = y; p.rot = rot; const b = fpBox(p, rot); placed.push({ x0: b.x0 + x - 0.8, y0: b.y0 + y - 0.8, x1: b.x1 + x + 0.8, y1: b.y1 + y + 0.8 }); };
  const usb = parts.find(p => p.fixed === "usb"), trrs = parts.find(p => p.fixed === "trrs");
  if (usb) put(usb, W / 2, -margin + 2.6, 180);
  if (mcu.bare) { const b = LIB_BY_ID[mcu.fp].body; put(U, W / 2, 9 + b.h / 2 + 2, 0); }
  else { const b = LIB_BY_ID[mcu.fp].body; put(U, W / 2, -margin + 1 + b.h / 2, 0); }
  if (trrs) { const xr = half === 0 ? W - 4.5 : 4.5; put(trrs, xr, -margin + 0.65, 0); }
  const rest = parts.filter(p => !p.fixed).sort((a, b) => { const A = fpBox(a, 0), Bx = fpBox(b, 0); return (Bx.x1 - Bx.x0) * (Bx.y1 - Bx.y0) - (A.x1 - A.x0) * (A.y1 - A.y0); });
  const hits = (r) => placed.some(q => r.x0 < q.x1 && r.x1 > q.x0 && r.y0 < q.y1 && r.y1 > q.y0);
  if (K.pack === "rows" || K.pack === "line") {
    // tidy: the support parts in rows under the controller, alike parts together, all facing the same way
    const rot = K.pack === "line" ? 90 : 0;
    rest.sort((a, b) => a.fp.localeCompare(b.fp) || String(a.value).localeCompare(String(b.value)));
    const dims = rest.map(p => { const b = fpBox(p, rot); return [b.x1 - b.x0, b.y1 - b.y0]; });
    const cw = Math.max(0, ...dims.map(d => d[0])) + 1.4, ch = Math.max(0, ...dims.map(d => d[1])) + 1.4;
    const perRow = Math.max(1, Math.floor((W - 1) / cw));
    let y = Math.max(0, ...placed.filter(q => q.y1 > 0).map(q => q.y1)) + 0.8;
    for (let i = 0; i < rest.length; i += perRow) {
      const row = rest.slice(i, i + perRow), x0 = (W - row.length * cw) / 2;
      row.forEach((p, j) => { const b = fpBox(p, rot); put(p, x0 + j * cw + cw / 2 - (b.x0 + b.x1) / 2, y + ch / 2 - (b.y0 + b.y1) / 2, rot); });
      y += ch;
    }
  } else for (const p of rest) {
    let done = false;
    for (let y = 0.5; y < 200 && !done; y += 0.5) for (let x = 0.5; x < W && !done; x += 0.5) for (const rot of [0, 90]) {
      const b = fpBox(p, rot), r = { x0: x, y0: y, x1: x + (b.x1 - b.x0), y1: y + (b.y1 - b.y0) };
      if (r.x1 > W - 0.3) continue;
      if (hits({ x0: r.x0 - 0.8, y0: r.y0 - 0.8, x1: r.x1 + 0.8, y1: r.y1 + 0.8 })) continue;
      put(p, x - b.x0, y - b.y0, rot); done = true; break;
    }
  }
  const H = Math.max(10, ...placed.map(q => q.y1)) + 0.5;
  return { W, H, parts, nets, vcc, mcu };
}

// ---------------------------------------------------------------- outline
function kbRectPoly(x0, y0, x1, y1, r) {
  r = Math.max(0, Math.min(r, (x1 - x0) / 2, (y1 - y0) / 2)); if (r < 0.05) return [[x0, y0], [x1, y0], [x1, y1], [x0, y1]];
  const pts = [], arc = (cx, cy, a0) => { for (let i = 0; i <= 6; i++) { const a = a0 + i / 6 * Math.PI / 2; pts.push([cx + r * Math.cos(a), cy + r * Math.sin(a)]); } };
  arc(x1 - r, y0 + r, -Math.PI / 2); arc(x1 - r, y1 - r, 0); arc(x0 + r, y1 - r, Math.PI / 2); arc(x0 + r, y0 + r, Math.PI);
  return pts;
}
function kbSimplify(pts, tol) {
  if (pts.length < 4) return pts;
  const keep = new Array(pts.length).fill(false); keep[0] = keep[pts.length - 1] = true;
  const st = [[0, pts.length - 1]];
  while (st.length) { const [a, b] = st.pop(); let md = 0, mi = -1; for (let i = a + 1; i < b; i++) { const d = ptSegDist(pts[i][0], pts[i][1], pts[a][0], pts[a][1], pts[b][0], pts[b][1]); if (d > md) { md = d; mi = i; } } if (md > tol) { keep[mi] = true; st.push([a, mi], [mi, b]); } }
  return pts.filter((_, i) => keep[i]);
}
// round every corner with radius r (cut back along both edges, then a curve through the corner)
function kbFillet(pts, r) {
  const n = pts.length, o = [];
  for (let i = 0; i < n; i++) {
    const p = pts[i], a = pts[(i - 1 + n) % n], b = pts[(i + 1) % n];
    const la = Math.hypot(a[0] - p[0], a[1] - p[1]), lb = Math.hypot(b[0] - p[0], b[1] - p[1]);
    const d = Math.min(r, la * 0.45, lb * 0.45);
    const turn = Math.PI - Math.acos(Math.max(-1, Math.min(1, ((a[0] - p[0]) * (b[0] - p[0]) + (a[1] - p[1]) * (b[1] - p[1])) / (la * lb || 1))));
    if (d < 0.05 || turn < 0.26) { o.push(p); continue; } // nearly straight: leave it
    const s = [p[0] + (a[0] - p[0]) / la * d, p[1] + (a[1] - p[1]) / la * d], e = [p[0] + (b[0] - p[0]) / lb * d, p[1] + (b[1] - p[1]) / lb * d];
    const steps = Math.max(2, Math.min(6, Math.round(turn / 0.3)));
    for (let k = 0; k <= steps; k++) { const t = k / steps, u = 1 - t; o.push([u * u * s[0] + 2 * u * t * p[0] + t * t * e[0], u * u * s[1] + 2 * u * t * p[1] + t * t * e[1]]); }
  }
  return o;
}
// trace the outside of a set of shapes (rotated key rectangles + the controller area)
function kbHug(shapes, radius) {
  let x0 = 1e9, y0 = 1e9, x1 = -1e9, y1 = -1e9;
  for (const s of shapes) for (const [x, y] of s) { x0 = Math.min(x0, x); y0 = Math.min(y0, y); x1 = Math.max(x1, x); y1 = Math.max(y1, y); }
  x0 -= 2; y0 -= 2; x1 += 2; y1 += 2;
  const res = Math.max(0.5, Math.sqrt((x1 - x0) * (y1 - y0) / 3e6)); // keep the grid a sensible size for huge layouts
  const W = Math.ceil((x1 - x0) / res) + 1, H = Math.ceil((y1 - y0) / res) + 1, g = new Uint8Array(W * H);
  for (const s of shapes) {
    let a = 1e9, b = 1e9, c = -1e9, d = -1e9; for (const [x, y] of s) { a = Math.min(a, x); b = Math.min(b, y); c = Math.max(c, x); d = Math.max(d, y); }
    for (let j = Math.floor((b - y0) / res); j <= Math.ceil((d - y0) / res); j++) for (let i = Math.floor((a - x0) / res); i <= Math.ceil((c - x0) / res); i++) if (i >= 0 && j >= 0 && i < W && j < H && ptInPoly(x0 + (i + 0.5) * res, y0 + (j + 0.5) * res, s)) g[j * W + i] = 1;
  }
  // close small notches between keys (a morphological close of ~1mm)
  const close = (src, grow) => { const o = new Uint8Array(W * H); for (let j = 0; j < H; j++) for (let i = 0; i < W; i++) { let v = grow ? 0 : 1; for (let dj = -2; dj <= 2 && (grow ? !v : v); dj++) for (let di = -2; di <= 2; di++) { const q = (j + dj) * W + i + di, inside = i + di >= 0 && j + dj >= 0 && i + di < W && j + dj < H && src[q]; if (grow && inside) { v = 1; break; } if (!grow && !inside) { v = 0; break; } } o[j * W + i] = v; } return o; };
  const G = close(close(g, true), false);
  // label components, trace the outer boundary of each
  const lab = new Int32Array(W * H); let n = 0; const polys = [];
  for (let s = 0; s < W * H; s++) {
    if (!G[s] || lab[s]) continue; n++;
    const st = [s]; lab[s] = n; let cnt = 0;
    while (st.length) { const q = st.pop(); cnt++; const i = q % W, j = (q - i) / W; for (const [di, dj] of [[1, 0], [-1, 0], [0, 1], [0, -1]]) { const ii = i + di, jj = j + dj; if (ii < 0 || jj < 0 || ii >= W || jj >= H) continue; const r = jj * W + ii; if (G[r] && !lab[r]) { lab[r] = n; st.push(r); } } }
    if (cnt < 40) continue;
    const next = new Map(), on = (i, j) => i >= 0 && j >= 0 && i < W && j < H && lab[j * W + i] === n;
    for (let j = 0; j < H; j++) for (let i = 0; i < W; i++) {
      if (!on(i, j)) continue;
      if (!on(i, j - 1)) next.set(i + "," + j, [i + 1, j]);
      if (!on(i + 1, j)) next.set((i + 1) + "," + j, [i + 1, j + 1]);
      if (!on(i, j + 1)) next.set((i + 1) + "," + (j + 1), [i, j + 1]);
      if (!on(i - 1, j)) next.set(i + "," + (j + 1), [i, j]);
    }
    let best = null; const used = new Set();
    for (const start of next.keys()) {
      if (used.has(start)) continue;
      const loop = []; let k = start, guard = 0;
      while (!used.has(k) && guard++ < 1e6) { used.add(k); const [a, b] = k.split(",").map(Number); loop.push([x0 + a * res, y0 + b * res]); const nx = next.get(k); if (!nx) break; k = nx[0] + "," + nx[1]; }
      if (!best || Math.abs(polyArea(loop)) > Math.abs(polyArea(best))) best = loop;
    }
    if (best) { let p = kbSimplify([...best, best[0]], 0.7).slice(0, -1); if (radius > 0.1) p = kbFillet(p, radius); polys.push(p); }
  }
  return polys;
}

function kbHull(pts) {
  const P = pts.slice().sort((a, b) => a[0] - b[0] || a[1] - b[1]), cr = (o, a, b) => (a[0] - o[0]) * (b[1] - o[1]) - (a[1] - o[1]) * (b[0] - o[0]);
  const lo = [], up = [];
  for (const p of P) { while (lo.length >= 2 && cr(lo[lo.length - 2], lo[lo.length - 1], p) <= 0) lo.pop(); lo.push(p); }
  for (const p of P.slice().reverse()) { while (up.length >= 2 && cr(up[up.length - 2], up[up.length - 1], p) <= 0) up.pop(); up.push(p); }
  return lo.slice(0, -1).concat(up.slice(0, -1));
}
// everything the editor and the builder need, cached until the design changes
// the controller's area: a W × H rectangle at (x, y), turned by rot about its centre
function kbArea(x, y, w, h, rot) {
  const cx = x + w / 2, cy = y + h / 2;
  const at = (lx, ly) => { const [rx, ry] = rotPt(lx - w / 2, ly - h / 2, rot); return [cx + rx, cy + ry]; };
  const poly = (g = 0) => [[-g, -g], [w + g, -g], [w + g, h + g], [-g, h + g]].map(([a, b]) => at(a, b));
  const P = poly(0);
  return { x, y, w, h, rot, cx, cy, at, poly, bb: [Math.min(...P.map(q => q[0])), Math.min(...P.map(q => q[1])), Math.max(...P.map(q => q[0])), Math.max(...P.map(q => q[1]))] };
}
function kbGeom(force) {
  const K = KBS(); const key = JSON.stringify(K);
  if (!force && KV.geom && KV.geomKey === key) return KV.geom;
  if (!force && KV.drag && !KV.drag.ctrl && KV.geom) { // while dragging: move the keys, keep the slow parts (outline, holes) until the drop
    const G0 = KV.geom, places = kbPlace(K);
    const G = Object.assign({}, G0, { places, boards: G0.boards.map(b => b && Object.assign({}, b, { places: places.filter(p => K.split.mode !== "two" || p.half === b.half), polys: K.outline.mode === "custom" && K.outline.custom && K.outline.custom[b.half] ? [K.outline.custom[b.half]] : b.polys })) });
    G.matrix = G0.matrix; KV.geom = G; KV.geomKey = key; return G;
  }
  const [sx, sy] = kbPitch(K), sw = kbSwitch(K), m = K.outline.margin, places = kbPlace(K);
  const boards = K.split.mode === "two" ? [0, 1] : [0];
  const G = { sx, sy, places, boards: [], matrix: kbMatrix(K, places), mcu: kbMcu(K), sw };
  for (const h of boards) {
    const ps = places.filter(p => K.split.mode !== "two" || p.half === h);
    if (!ps.length) { G.boards.push(null); continue; }
    let bx0 = 1e9, by0 = 1e9, bx1 = -1e9; for (const p of ps) for (const [x, y] of kbCorners(p, sx, sy)) { bx0 = Math.min(bx0, x); bx1 = Math.max(bx1, x); by0 = Math.min(by0, y); }
    const plan = kbPlanCtrl(K, h);
    const two = K.split.mode === "two";
    const pos = K.ctrlPos === "side" ? (two && h === 1 ? "sideL" : "sideR") : two ? (K.ctrlPos === "centre" ? "centre" : h === 0 ? "right" : "left") : K.ctrlPos;
    const keyPolys = ps.map(p => kbCorners(p, sx, sy));
    let ax, ay;
    if (pos === "sideR" || pos === "sideL") {
      ay = by0; let ex = pos === "sideR" ? -1e9 : 1e9;
      for (const poly of keyPolys) for (const [x, y] of poly) if (y >= ay - 2 && y <= ay + plan.H + 2) ex = pos === "sideR" ? Math.max(ex, x) : Math.min(ex, x);
      if (Math.abs(ex) > 1e8) ex = pos === "sideR" ? bx1 : bx0;
      ax = pos === "sideR" ? ex + 1.5 : ex - plan.W - 1.5;
    }
    else {
      ax = pos === "left" ? bx0 : pos === "centre" ? (bx0 + bx1) / 2 - plan.W / 2 : bx1 - plan.W;
      // sit just above the keys that are underneath the controller
      let top = 1e9; for (const poly of keyPolys) for (const [x, y] of poly) if (x >= ax - 2 && x <= ax + plan.W + 2) top = Math.min(top, y);
      ay = (top < 1e9 ? top : by0) - plan.H - 1;
    }
    const overl = () => keyPolys.some(poly => poly.some(([x, y]) => x > ax - 1 && x < ax + plan.W + 1 && y > ay - 1 && y < ay + plan.H + 1) || ptInPoly(ax + plan.W / 2, ay + plan.H, poly) || ptInPoly(ax, ay + plan.H, poly) || ptInPoly(ax + plan.W, ay + plan.H, poly));
    for (let i = 0; i < 200 && overl(); i++) { if (pos === "sideR") ax += 1; else if (pos === "sideL") ax -= 1; else ay -= 1; }
    // moved or turned by hand (drag the controller, or its panel)
    const off = (K.ctrlOff || [])[h] || {};
    ax += off.x || 0; ay += off.y || 0;
    const area = kbArea(ax, ay, plan.W, plan.H, off.rot || 0);
    const areaPoly = g => area.poly(g);
    let polys;
    const custom = K.outline.mode === "custom" && K.outline.custom && K.outline.custom[h];
    if (custom) polys = [custom.map(p => p.slice())];
    else if (K.outline.mode === "rect") {
      let x0 = Math.min(bx0, area.bb[0]), x1 = Math.max(bx1, area.bb[2]), y0 = Math.min(by0, area.bb[1]), y1 = area.bb[3];
      for (const p of ps) for (const [, y] of kbCorners(p, sx, sy)) y1 = Math.max(y1, y);
      polys = [kbRectPoly(x0 - m, y0 - m, x1 + m, y1 + m, K.outline.radius)];
    } else {
      const shapes = ps.map(p => p.k.shape === "iso" ? kbISOPoly(p, sx, sy, m) : kbCorners(p, sx, sy, m));
      shapes.push(areaPoly(m));
      // one board with angled halves: fill the space between the halves (and up to the controller)
      if (K.split.mode === "angled") {
        const near = ps.filter(p => Math.abs(p.bcx - K.split.at) < 1.4).flatMap(p => kbCorners(p, sx, sy, m));
        near.push(...areaPoly(m));
        if (near.length > 2) shapes.push(kbHull(near));
      }
      polys = kbHug(shapes, K.outline.radius);
    }
    G.boards.push({ half: h, places: ps, plan, area, polys, holes: [] });
  }
  for (const b of G.boards) if (b) b.holes = KV.drag ? (KV.geom?.boards?.[b.half]?.holes || []) : kbHoles(K, G, b);
  KV.geom = G; KV.geomKey = key;
  return G;
}

// local features (pads, holes, diode, LED) around one key, for mounting-hole checks
function kbKeyFeatures(K, G, p) {
  const knob = p.k.kind === "knob", fp = LIB_BY_ID[knob ? "ec11" : G.sw.fp], f = [];
  const rot = p.ang + (K.fpRot || 0) + (p.k.fr || 0);
  const T = (x, y) => { const [rx, ry] = rotPt(x, y, rot); return [p.cx + rx, p.cy + ry]; };
  for (const q of fp.pads) f.push([...T(q.x, q.y), Math.max(q.w, q.h) / 2 + 0.3]);
  for (const h of fp.holes) f.push([...T(h.x, h.y), h.d / 2]);
  const sp = kbSpotsFor(K, G, p), dl = K.diode === "d_do35" ? 4.7 : 2.4;
  for (const t of [-1, -0.5, 0, 0.5, 1]) { const [ox, oy] = rotPt(t * dl, 0, sp.dio.rot); f.push([...T(sp.dio.x + ox, sp.dio.y + oy), 1.2]); }
  if (sp.led) f.push([...T(0, 5.05), 3.2]);
  if (sp.cap) f.push([...T(sp.cap.x, sp.cap.y), 1.6]);
  const st = kbStab(K, G, p);
  if (st) { const s = stabSpacing(st.units); for (const [x, y, d] of [[-s / 2, -6.985, 3.05], [s / 2, -6.985, 3.05], [-s / 2, 8.255, 3.988], [s / 2, 8.255, 3.988]]) { const [rx, ry] = rotPt(x, y, st.rot); f.push([p.cx + rx, p.cy + ry, d / 2]); } }
  return f;
}
function kbStab(K, G, p) {
  const k = p.k; if (k.stab === "none" || !G.sw.stabs || k.kind === "knob") return null;
  const units = Math.max(k.w, k.h); if (units < 2) return null;
  return { units, rot: p.ang + (k.h > k.w ? 90 : 0) };
}
// ---- where each key's diode, LED and LED capacitor go
// Spots are in the switch footprint's own frame (the frame the build turns by
// the key angle and footprint turn). Each layout style lists the spots it
// would like, best first; a spot is used when it clears the switch's pads and
// holes, the LED and its cut-out, and stays inside the key's own square.
const KB_DSTYLES = [
  { id: "beside", name: "Beside the switch, upright (right)", tries: [[7.6, 3.8, 90], [7.6, 2.6, 90], [7.6, 5.0, 90], [-7.6, 3.8, 90]] },
  { id: "left", name: "Beside the switch, upright (left)", tries: [[-7.6, 3.8, 90], [-7.6, 2.6, 90], [-7.6, 5.0, 90], [7.6, 3.8, 90]] },
  { id: "below", name: "Below the switch, lying flat", tries: [[0, "low", 0], [2, "low", 0], [-2, "low", 0], [4, "low", 0]] },
  { id: "above", name: "Above the switch, lying flat", tries: [[0, "high", 0], [1.5, "high", 0], [-1.5, "high", 0], [3, "high", 0], [-3, "high", 0]] },
  { id: "corner", name: "In the corner, at 45°", tries: [[6.3, 6.3, 45], [6.0, 6.6, 45], [-6.3, 6.3, -45], [6.3, -6.6, -45]] },
];
const KB_PACKS = [{ id: "compact", name: "Packed tightly round the controller" }, { id: "rows", name: "Neat rows, grouped by part" }, { id: "line", name: "One tidy line, all upright" }];
function kbSpotsFor(K, G, p) {
  const knob = p.k.kind === "knob", swfp = LIB_BY_ID[knob ? "ec11" : G.sw.fp], [sx, sy] = [G.sx, G.sy];
  const halfX = p.k.w * sx / 2 - 0.4, halfY = p.k.h * sy / 2 - 0.4;
  const obs = []; // [x0, y0, x1, y1] boxes and [cx, cy, r] circles, in the footprint frame
  for (const q of swfp.pads) obs.push({ b: [q.x - q.w / 2, q.y - q.h / 2, q.x + q.w / 2, q.y + q.h / 2] });
  for (const h of swfp.holes || []) obs.push({ c: [h.x, h.y, h.d / 2] });
  const led = K.rgb && !knob ? { x: 0, y: 5.05, rot: 0 } : null;
  if (led) obs.push({ b: [-3.25, 3.6, 3.25, 6.5] });
  // a part's corners at (x, y, rot), with a little clearance
  const shape = (fpId, x, y, rot) => {
    const fp = LIB_BY_ID[fpId]; let x0 = -fp.body.w / 2, x1 = fp.body.w / 2, y0 = -fp.body.h / 2, y1 = fp.body.h / 2;
    for (const q of fp.pads) { x0 = Math.min(x0, q.x - q.w / 2); x1 = Math.max(x1, q.x + q.w / 2); y0 = Math.min(y0, q.y - q.h / 2); y1 = Math.max(y1, q.y + q.h / 2); }
    const c = 0.35, pts = [];
    for (let i = 0; i <= 6; i++) for (let j = 0; j <= 3; j++) { const [rx, ry] = rotPt(x0 - c + (x1 - x0 + 2 * c) * i / 6, y0 - c + (y1 - y0 + 2 * c) * j / 3, rot); pts.push([x + rx, y + ry]); }
    return pts;
  };
  const clear = pts => pts.every(([x, y]) => Math.abs(x) <= halfX && Math.abs(y) <= halfY && !obs.some(o => o.b ? x > o.b[0] && x < o.b[2] && y > o.b[1] && y < o.b[3] : Math.hypot(x - o.c[0], y - o.c[1]) < o.c[2]));
  const st = KB_DSTYLES.find(s => s.id === K.dstyle) || KB_DSTYLES[0];
  const dfp = K.diode, dl = LIB_BY_ID[dfp].pads.reduce((m, q) => Math.max(m, Math.abs(q.x) + q.w / 2), 0);
  const low = led ? 6.5 + 1.25 : (knob ? 8.4 : 5.6), high = -(Math.max(...swfp.pads.map(q => -(q.y - q.h / 2)), 0) + 1.25);
  let dio = null;
  for (const [x, y0, rot] of [...st.tries, ...KB_DSTYLES[0].tries]) {
    const y = y0 === "low" ? Math.min(low, halfY - 1) : y0 === "high" ? Math.max(high, -halfY + 1) : y0;
    const xx = rot === 0 && Math.abs(x) + dl > halfX ? Math.sign(x) * (halfX - dl) : x;
    const pts = shape(dfp, xx, y, rot);
    if (clear(pts)) { dio = { x: xx, y, rot }; break; }
  }
  if (!dio) dio = { x: 7.6, y: 3.8, rot: 90 }; // the classic spot, even if it's tight
  const dpts = shape(dfp, dio.x, dio.y, dio.rot), dbox = [Math.min(...dpts.map(q => q[0])), Math.min(...dpts.map(q => q[1])), Math.max(...dpts.map(q => q[0])), Math.max(...dpts.map(q => q[1]))];
  obs.push({ b: dbox });
  let cap = null;
  if (led && K.ledCap) for (const [x, y, rot] of [[4.6, 5.05, 90], [-4.6, 5.05, 90], [0, 7.6, 0], [0, 2.7, 0], [5.8, 5.05, 90], [-5.8, 5.05, 90]]) if (clear(shape("c_0805", x, y, rot))) { cap = { x, y, rot }; break; }
  return { dio, led, cap };
}
function kbHoles(K, G, b) {
  const n = K.holes | 0; if (!n || !b.polys.length) return [];
  const feats = []; for (const p of b.places) feats.push(...kbKeyFeatures(K, G, p));
  const A = b.area; let x0 = 1e9, y0 = 1e9, x1 = -1e9, y1 = -1e9;
  for (const poly of b.polys) for (const [x, y] of poly) { x0 = Math.min(x0, x); y0 = Math.min(y0, y); x1 = Math.max(x1, x); y1 = Math.max(y1, y); }
  const cands = [], r = 1.1 + 1.8;
  // features bucketed in a 10 mm grid, so each spot only checks its neighbours
  const CELL = 10, grid = new Map(), cell = (x, y) => Math.floor(x / CELL) + "," + Math.floor(y / CELL);
  for (const f of feats) { const reach = r + f[2]; for (let gx = Math.floor((f[0] - reach) / CELL); gx <= Math.floor((f[0] + reach) / CELL); gx++) for (let gy = Math.floor((f[1] - reach) / CELL); gy <= Math.floor((f[1] + reach) / CELL); gy++) { const k = gx + "," + gy; if (!grid.has(k)) grid.set(k, []); grid.get(k).push(f); } }
  const pb = b.polys.map(pl => { let a0 = 1e9, b0 = 1e9, a1 = -1e9, b1 = -1e9; for (const [x, y] of pl) { a0 = Math.min(a0, x); b0 = Math.min(b0, y); a1 = Math.max(a1, x); b1 = Math.max(b1, y); } return [a0, b0, a1, b1]; });
  const inBoard = (x, y) => b.polys.some((pl, i) => x > pb[i][0] + 3 && x < pb[i][2] - 3 && y > pb[i][1] + 3 && y < pb[i][3] - 3 && ptInPoly(x, y, pl) && !nearEdge(x, y, pl, 3.2));
  function nearEdge(x, y, pl, lim) { // is (x, y) within lim of the outline? (exits early)
    for (let i = 0, n = pl.length; i < n; i++) {
      const [ax, ay] = pl[i], [bx, by] = pl[(i + 1) % n];
      if (x < Math.min(ax, bx) - lim || x > Math.max(ax, bx) + lim || y < Math.min(ay, by) - lim || y > Math.max(ay, by) + lim) continue;
      const dx = bx - ax, dy = by - ay, l2 = dx * dx + dy * dy, t = l2 ? Math.max(0, Math.min(1, ((x - ax) * dx + (y - ay) * dy) / l2)) : 0;
      if (Math.hypot(x - ax - t * dx, y - ay - t * dy) < lim) return true;
    }
    return false;
  }
  for (let y = y0 + 3; y < y1 - 3; y += 1.6) for (let x = x0 + 3; x < x1 - 3; x += 1.6) {
    if (A && (A.rot ? x > A.bb[0] - 6 && x < A.bb[2] + 6 && y > A.bb[1] - 6 && y < A.bb[3] + 6 : x > A.x - 3 && x < A.x + A.w + 3 && y > A.y - 6 && y < A.y + A.h + 3)) continue;
    let ok = true; for (const f of grid.get(cell(x, y)) || []) { const dx = x - f[0], dy = y - f[1], lim = r + f[2]; if (dx * dx + dy * dy < lim * lim) { ok = false; break; } }
    if (!ok || !inBoard(x, y)) continue;
    cands.push([x, y]);
  }
  if (!cands.length) return [];
  const chosen = []; const corner = [[x0, y0], [x1, y0], [x1, y1], [x0, y1]];
  for (const c of corner) { if (chosen.length >= n) break; let best = null, bd = 1e9; for (const p of cands) { const d = Math.hypot(p[0] - c[0], p[1] - c[1]); if (d < bd && chosen.every(q => Math.hypot(q[0] - p[0], q[1] - p[1]) > 25)) { bd = d; best = p; } } if (best) chosen.push(best); }
  while (chosen.length < n) { let best = null, bd = 0; for (const p of cands) { const d = Math.min(...chosen.map(q => Math.hypot(q[0] - p[0], q[1] - p[1]))); if (d > bd) { bd = d; best = p; } } if (!best || bd < 20) break; chosen.push(best); }
  return chosen;
}

// ---------------------------------------------------------------- build the PCB
function kbBuild(half) {
  const K = KBS();
  if (!K.keys.length) { toast("Add some keys first: pick a layout, make a grid or type a number of keys.", "warn"); return; }
  const G = kbGeom(true), b = G.boards[half || 0];
  if (!b) { toast("That half has no keys.", "warn"); return; }
  const mcu = G.mcu, sw = G.sw, M = G.matrix;
  const pins = kbFreePins(K);
  const R = M.rows, C = M.cols;
  if (R + C > pins.length) { toast(`The ${mcu.name} has ${pins.length} free pins but this matrix needs ${R + C}. Pick a controller with more pins.`, "bad"); return; }
  // shift so the board starts at (0, 0)
  let ox = 1e9, oy = 1e9, ex = -1e9, ey = -1e9; for (const pl of b.polys) for (const [x, y] of pl) { ox = Math.min(ox, x); oy = Math.min(oy, y); ex = Math.max(ex, x); ey = Math.max(ey, y); }
  const X = x => x - ox, Y = y => y - oy;
  const keepName = S.name;
  const fresh = newProject();
  S.comps = []; S.conns = []; S.tracks = []; S.vias = []; S.texts = []; S.netLabels = {};
  Object.assign(S.board, { w: ex - ox, h: ey - oy, r: 0, round: false, polys: b.polys.map(pl => pl.map(([x, y]) => [X(x), Y(y)])), cutouts: [], grid: 0.5,
    track: mcu.bare ? 0.2 : 0.25, power: 0.5, clear: mcu.bare ? 0.15 : 0.2, pourBot: true, pourTop: false, pourNet: "GND", edgeClear: 0.4 });
  S.name = keepName && keepName !== fresh.name && !K.built ? keepName : (K.name || "Keyboard") + (K.split.mode === "two" ? (half ? " (right)" : " (left)") : "");
  const nets = {}; const pinKey = (c, name) => { const fp = compFP(c); const p = fp.pads.find(q => q.name === name) || fp.pads.find(q => q.n === name); return p ? c.id + ":" + p.n : null; };
  const net = (name, c, pin) => { const k = pinKey(c, pin); if (!k) { console.warn("no pin", c.fp, pin); return; } (nets[name] = nets[name] || []).push(k); };
  const pfx = "";
  // ---- keys
  const ord = kbOrd(G, b);
  let n = 0, stabN = 0, nDio = 0, nCap = 0; const leds = [], knobs = [];
  for (const p of ord) {
    n++;
    const knob = p.k.kind === "knob";
    const rot = knob ? p.ang : p.ang + (K.fpRot || 0) + (p.k.fr || 0);
    const T = (x, y) => { const [rx, ry] = rotPt(x, y, rot); return [X(p.cx + rx), Y(p.cy + ry)]; };
    const s = knob ? addComp("ec11", X(p.cx), Y(p.cy), { ref: "ENC" + (knobs.length + 1), value: "EC11 knob", rot })
      : addComp(sw.fp, X(p.cx), Y(p.cy), { ref: "SW" + n, value: (p.k.label || sw.fam.toUpperCase()) + (p.k.w !== 1 || p.k.h !== 1 ? ` ${Math.max(p.k.w, p.k.h)}u` : ""), rot });
    if (knob) knobs.push(s); else s.kbTurn = (K.fpRot || 0) + (p.k.fr || 0);
    const P1 = knob ? "S1" : "1", P2 = knob ? "S2" : "2";
    const sp = kbSpotsFor(K, G, p);
    const [dx, dy] = T(sp.dio.x, sp.dio.y);
    const d = addComp(K.diode, dx, dy, { ref: "D" + n, value: K.diode === "d_do35" ? "1N4148" : "1N4148W", rot: rot + sp.dio.rot, side: K.dside === "top" ? "top" : "bottom" });
    nDio++;
    const [r, c] = M.pos.get(p.k.id);
    const ROW = `${pfx}ROW${r}`, COL = `${pfx}COL${c}`;
    if (K.dir === "COL2ROW") { net(COL, s, P1); net(`K${n}`, s, P2); net(`K${n}`, d, "A"); net(ROW, d, "K"); }
    else { net(ROW, s, P1); net(`K${n}`, s, P2); net(`K${n}`, d, "A"); net(COL, d, "K"); }
    if (knob) continue;
    const st = kbStab(K, G, p);
    if (st) { stabN++; const u = st.units >= 8 ? "8" : st.units >= 7 ? "7" : st.units >= 6.25 ? "6.25" : st.units >= 6 ? "6" : st.units >= 3 ? "3" : "2"; addComp("kb_stab_" + u.replace(".", "_"), X(p.cx), Y(p.cy), { ref: "ST" + stabN, rot: st.rot }); }
    if (K.rgb) {
      const [lx, ly] = T(0, 5.05);
      const L = addComp("sk6812mini", lx, ly, { ref: "LED" + n, value: "SK6812MINI-E", rot, side: "bottom" });
      leds.push(L);
      if (sp.cap) { const [cx, cy] = T(sp.cap.x, sp.cap.y); const C = addComp("c_0805", cx, cy, { ref: "C" + (100 + n), value: "100nF", rot: rot + sp.cap.rot, side: "bottom" }); if (C) { nCap++; L.kbCap = C; } }
      const [qa, qb] = [[-1.7, -1.4], [1.7, 1.4]]; const cut = [[qa[0], qa[1]], [qb[0], qa[1]], [qb[0], qb[1]], [qa[0], qb[1]]].map(([x, y]) => T(x, 5.05 + y));
      S.board.cutouts.push(cut);
    }
  }
  // ---- controller and its circuitry
  const plan = b.plan, area = b.area, made = new Map();
  for (const p of plan.parts) {
    const [wx, wy] = area.at(p.x, p.y);
    const c = addComp(p.fp, X(wx), Y(wy), { ref: p.ref || undefined, value: p.value, rot: (p.rot + (area.rot || 0)) % 360, side: p.side });
    if (c) made.set(p.id, c);
  }
  for (const [name, list] of plan.nets) for (const [p, pin] of list) { const c = made.get(p.id); if (c) net(name, c, pin); }
  const U = made.get(plan.parts.find(p => p.fixed === "mcu").id);
  for (let r = 0; r < R; r++) net(`${pfx}ROW${r}`, U, pins[r]);
  for (let c = 0; c < C; c++) net(`${pfx}COL${c}`, U, pins[R + c]);
  // power names for modules: join the module's own pins to the friendly net names
  if (!mcu.bare) { net("GND", U, "GND"); }
  const PP = kbPinPlan(K);
  knobs.forEach((e, i) => { const pr = PP.enc[i]; if (!pr) return; net(`ENC${i + 1}_A`, e, "A"); net(`ENC${i + 1}_A`, U, pr[0]); net(`ENC${i + 1}_B`, e, "B"); net(`ENC${i + 1}_B`, U, pr[1]); net("GND", e, "C"); });
  if (K.rgb && leds.length) {
    const p5 = mcu.bare ? "5V" : "LED_5V";
    net("LED_DATA0", U, mcu.led);
    leds.forEach((L, i) => { net(`LED_DATA${i}`, L, "DIN"); net(`LED_DATA${i + 1}`, L, "DOUT"); net(p5, L, "VDD"); net("GND", L, "GND"); if (L.kbCap) { net(p5, L.kbCap, "1"); net("GND", L.kbCap, "2"); delete L.kbCap; } });
  }
  // ---- mounting holes
  for (const [hx, hy] of b.holes) addComp("mh_M2", X(hx), Y(hy), { ref: nextRef("H") });
  // ---- nets → connections and names
  for (const [name, keys] of Object.entries(nets)) {
    const ks = [...new Set(keys)];
    for (let i = 1; i < ks.length; i++) S.conns.push({ a: ks[i - 1], b: ks[i] });
    if (!/^K\d+$/.test(name) && !/^LED_DATA\d+$/.test(name) || name === "LED_DATA0") S.netLabels[ks[0]] = name.replace(/^LED_DATA0$/, "LED_DATA");
  }
  const [tx, ty] = area.at(2, area.h - 1.5);
  S.texts.push({ id: uid(), x: X(tx), y: Y(ty), text: `${(K.name || "KEYBOARD").toUpperCase().slice(0, 24)}`, size: 1.5, layer: "top", rot: area.rot || 0 });
  S.texts.push({ id: uid(), x: X(tx), y: Y(ty), text: "KEEBIE BY NRMLUVDEV", size: 1.2, layer: "bottom", rot: area.rot || 0 });
  K.built = { at: Date.now(), half: half || 0 };
  UI.sel = new Set(); invalidate(); commit("keebie build");
  switchMode("pcb"); fitView();
  const under = [K.dside !== "top" ? "diodes" : "", leds.length ? "LEDs" : "", nCap ? "capacitors" : ""].filter(Boolean);
  const msg = `Built <b>${n - knobs.length}</b> keys${knobs.length ? ` and ${knobs.length} knob${knobs.length > 1 ? "s" : ""}` : ""} with <b>${nDio}</b> diodes${leds.length ? `, <b>${leds.length}</b> RGB LEDs` : ""}${nCap ? ` and <b>${nCap}</b> LED capacitors` : ""} on a ${(ex - ox).toFixed(0)} × ${(ey - oy).toFixed(0)} mm board with the ${esc(mcu.name)}.${under.length ? ` The ${under.join(", ").replace(/, ([^,]*)$/, " and $1")} are on the back: <a href="#" class="kb-under">show the back</a>.` : ""} Press <b>Auto-route</b> to draw the tracks.`;
  toast(`<span>${msg}</span>`);
  const a = $$("#toasts .kb-under").pop(); if (a) a.onclick = e => { e.preventDefault(); setLayer("bottom"); };
}

// ---------------------------------------------------------------- plate & firmware exports
function kbPlateShapes() {
  const K = KBS(), G = kbGeom(), sw = G.sw, half = K.split.mode === "two" ? (K.half || 0) : 0, b = G.boards[half];
  const cuts = [];
  for (const p of b.places) {
    const rot = p.ang + (K.fpRot || 0) + (p.k.fr || 0), cw = sw.cut, ch = sw.cutH || sw.cut;
    cuts.push([[-cw / 2, -ch / 2], [cw / 2, -ch / 2], [cw / 2, ch / 2], [-cw / 2, ch / 2]].map(([x, y]) => { const [rx, ry] = rotPt(x, y, rot); return [p.cx + rx, p.cy + ry]; }));
    const st = kbStab(K, G, p);
    if (st) { const s = stabSpacing(st.units); for (const sx of [-s / 2, s / 2]) cuts.push([[-3.375, -6.2], [3.375, -6.2], [3.375, 7.8], [-3.375, 7.8]].map(([x, y]) => { const [rx, ry] = rotPt(sx + x, y, st.rot); return [p.cx + rx, p.cy + ry]; })); }
  }
  return { outline: b.polys, cuts, holes: b.holes };
}
function kbPlateSVG() {
  const P = kbPlateShapes(); let x0 = 1e9, y0 = 1e9, x1 = -1e9, y1 = -1e9;
  for (const pl of P.outline) for (const [x, y] of pl) { x0 = Math.min(x0, x); y0 = Math.min(y0, y); x1 = Math.max(x1, x); y1 = Math.max(y1, y); }
  const pt = pl => pl.map(([x, y]) => `${(x - x0).toFixed(3)},${(y - y0).toFixed(3)}`).join(" ");
  const W = (x1 - x0).toFixed(3), H = (y1 - y0).toFixed(3);
  return `<svg xmlns="http://www.w3.org/2000/svg" width="${W}mm" height="${H}mm" viewBox="0 0 ${W} ${H}"><!-- Switch plate made with Keebie (Wirie, Tinkerie) by nrmluvdev. Stabiliser cut-outs are approximate: check them against your stabilisers. -->\n<g fill="none" stroke="#000" stroke-width="0.1">\n${P.outline.map(pl => `<polygon points="${pt(pl)}"/>`).join("\n")}\n${P.cuts.map(pl => `<polygon points="${pt(pl)}"/>`).join("\n")}\n${P.holes.map(([x, y]) => `<circle cx="${(x - x0).toFixed(3)}" cy="${(y - y0).toFixed(3)}" r="1.1"/>`).join("\n")}\n</g></svg>`;
}
function kbPlateDXF() {
  const P = kbPlateShapes(); let x0 = 1e9, y1 = -1e9; for (const pl of P.outline) for (const [x, y] of pl) { x0 = Math.min(x0, x); y1 = Math.max(y1, y); }
  const L = ["0", "SECTION", "2", "ENTITIES"];
  const line = (a, b) => L.push("0", "LINE", "8", "PLATE", "10", (a[0] - x0).toFixed(4), "20", (y1 - a[1]).toFixed(4), "11", (b[0] - x0).toFixed(4), "21", (y1 - b[1]).toFixed(4));
  for (const pl of [...P.outline, ...P.cuts]) for (let i = 0; i < pl.length; i++) line(pl[i], pl[(i + 1) % pl.length]);
  for (const [x, y] of P.holes) L.push("0", "CIRCLE", "8", "PLATE", "10", (x - x0).toFixed(4), "20", (y1 - y).toFixed(4), "40", "1.1");
  L.push("0", "ENDSEC", "0", "EOF");
  return L.join("\n");
}
// layout positions in key units for firmware files (after arc/split), top-left at 0
function kbFirmwareLayout(K, G) {
  const [sx, sy] = [G.sx, G.sy], ps = G.places;
  const minX = Math.min(...ps.map(p => p.cx / sx - p.w / 2)), minY = Math.min(...ps.map(p => p.cy / sy - p.h / 2));
  return ps.map(p => { const x = p.cx / sx - p.w / 2 - minX, y = p.cy / sy - p.h / 2 - minY; const o = { x: +x.toFixed(3), y: +y.toFixed(3) }; if (p.w !== 1) o.w = p.w; if (p.h !== 1) o.h = p.h; if (Math.abs(p.ang) > 0.01) Object.assign(o, { r: +p.ang.toFixed(2), rx: +(x + p.w / 2).toFixed(3), ry: +(y + p.h / 2).toFixed(3) }); return { p, o }; });
}
function kbMatrixFor(K, G, p) { const [r, c] = G.matrix.pos.get(p.k.id); return [K.split.mode === "two" && p.half ? r + G.matrix.rows : r, c]; }
function kbQMK() {
  const K = KBS(), G = kbGeom(), m = G.mcu, M = G.matrix, pins = kbFreePins(K);
  const q = m.qmk; const name = (K.name || "keyboard").toLowerCase().replace(/[^a-z0-9]+/g, "_").replace(/^_|_$/g, "") || "keyboard";
  const lay = kbFirmwareLayout(K, G).sort((a, b) => a.o.y - b.o.y || a.o.x - b.o.x);
  const kb = { manufacturer: "Your name", keyboard_name: K.name || "Keyboard", maintainer: "your-github-name", url: "", usb: { vid: "0xFEED", pid: "0x" + (Math.abs([...name].reduce((h, c) => h * 31 + c.charCodeAt(0) | 0, 7)) % 65536).toString(16).padStart(4, "0").toUpperCase(), device_version: "1.0.0" } };
  if (q.board && /^(promicro|elite_c)$/.test(q.board)) kb.development_board = q.board; else { kb.processor = q.processor; kb.bootloader = q.bootloader; if (q.board) kb.board = q.board; }
  kb.diode_direction = K.dir;
  kb.matrix_pins = { rows: pins.slice(0, M.rows).map(q.map), cols: pins.slice(M.rows, M.rows + M.cols).map(q.map) };
  kb.features = { bootmagic: true, extrakey: true, mousekey: true, nkro: true };
  if (K.split.mode === "two") kb.split = { enabled: true, serial: { pin: q.map(m.serial) } };
  const PP = kbPinPlan(K);
  if (PP.enc.length) { kb.features.encoder = true; kb.encoder = { rotary: PP.enc.map(([a, b]) => ({ pin_a: q.map(a), pin_b: q.map(b) })) }; }
  if (PP.i2c) kb.features.oled = true;
  if (K.rgb) {
    kb.features.rgb_matrix = true; kb.ws2812 = { pin: q.map(m.led) };
    const xs = lay.map(l => l.o.x), ys = lay.map(l => l.o.y), mx = Math.max(...xs) || 1, my = Math.max(...ys) || 1;
    const ord = G.places.slice().sort((a, b) => { const A = kbMatrixFor(K, G, a), B = kbMatrixFor(K, G, b); return A[0] - B[0] || (A[0] % 2 ? B[1] - A[1] : A[1] - B[1]); });
    kb.rgb_matrix = { driver: "ws2812", layout: ord.map(p => { const l = lay.find(z => z.p === p); return { matrix: kbMatrixFor(K, G, p), x: Math.round(l.o.x / mx * 224), y: Math.round(l.o.y / my * 64), flags: 4 }; }) };
    if (K.split.mode === "two") kb.rgb_matrix.split_count = [G.places.filter(p => !p.half).length, G.places.filter(p => p.half).length];
  }
  kb.layouts = { LAYOUT: { layout: lay.map(l => Object.assign({ label: l.p.k.label || "", matrix: kbMatrixFor(K, G, l.p) }, l.o)) } };
  const keymap = { keyboard: name, keymap: "default", layout: "LAYOUT", layers: [lay.map(l => kbKeycode(l.p.k.label, l.p.half === 1 || l.o.x > 9))] };
  return { kb, keymap, name, note: q.convert ? `Build with: qmk compile -kb ${name} -km default -e CONVERT_TO=${q.convert}` : `Build with: qmk compile -kb ${name} -km default` };
}
function kbZMK() {
  const K = KBS(), G = kbGeom(), m = G.mcu, M = G.matrix, pins = kbFreePins(K), z = m.zmk;
  const rows = pins.slice(0, M.rows), cols = pins.slice(M.rows, M.rows + M.cols), split = K.split.mode === "two";
  const R = split ? M.rows * 2 : M.rows, c2r = K.dir === "COL2ROW";
  const lay = kbFirmwareLayout(K, G).sort((a, b) => a.o.y - b.o.y || a.o.x - b.o.x);
  const gl = (list, flags) => list.map((p, i) => `            ${i ? "," : "="} <${z.node} ${z.map(p)} ${flags}>`).join("\n");
  const inF = "(GPIO_ACTIVE_HIGH | GPIO_PULL_DOWN)", outF = "GPIO_ACTIVE_HIGH";
  return `/*
 * ${K.name}: ZMK matrix for the ${m.name}
 * Made with Keebie (Wirie, Tinkerie) by nrmluvdev.
 * Put this in boards/shields/<name>/<name>.overlay${split ? ". Both halves use the same pins; give the right half its own overlay with col-offset if you change the columns" : ""}.
 */
#include <dt-bindings/zmk/matrix_transform.h>

/ {
    chosen {
        zmk,kscan = &kscan0;
        zmk,matrix-transform = &default_transform;
    };

    default_transform: keymap_transform_0 {
        compatible = "zmk,matrix-transform";
        columns = <${M.cols}>;
        rows = <${R}>;
        map = <
${lay.map(l => { const [r, c] = kbMatrixFor(K, G, l.p); return `            RC(${r},${c})`; }).join("\n")}
        >;
    };

    kscan0: kscan {
        compatible = "zmk,kscan-gpio-matrix";
        diode-direction = "${c2r ? "col2row" : "row2col"}";
        row-gpios
${gl(rows, c2r ? inF : outF)}
            ;
        col-gpios
${gl(cols, c2r ? outF : inF)}
            ;
    };
};
`;
}

// ---------------------------------------------------------------- hand-off to Flashie and Makie
// Everything Flashie needs to write Arduino firmware for this keyboard.
function kbFlashieExport() {
  const K = KBS(), G = kbGeom(), m = G.mcu, M = G.matrix, PP = kbPinPlan(K), a = m.ard;
  const pin = pad => pad ? { pad, pin: a ? a.pin(pad) : null } : null;
  const lay = kbFirmwareLayout(K, G);
  const knobs = G.boards[0] ? kbOrd(G, G.boards[0]).filter(p => p.k.kind === "knob") : [];
  const scr = KB_SCREENS.find(x => x.id === K.oled);
  return {
    format: "keebie-keyboard", version: 1,
    generator: { app: APP.name, version: APP.version, author: APP.author, tool: "Keebie", exported: new Date().toISOString() },
    name: K.name || "Keyboard", switch: kbSwitch(K).fam,
    mcu: { id: m.id, name: m.name, core: a ? a.core : null, fqbn: a ? a.fqbn : null, flash: a ? a.flash : null, note: a && a.note || "", volt: m.volt, wireless: !!m.wireless, bare: !!m.bare },
    split: K.split.mode, diode: K.dir,
    rows: PP.pins.slice(0, M.rows).map(pin), cols: PP.pins.slice(M.rows, M.rows + M.cols).map(pin),
    keys: lay.map(({ p, o }) => { const [row, col] = kbMatrixFor(K, G, p); return { id: p.k.id, label: p.k.label || "", x: o.x, y: o.y, w: p.w, h: p.h, r: o.r || 0, rx: o.rx, ry: o.ry, row, col, half: p.half, kind: p.k.kind || "key", qmk: kbKeycode(p.k.label, p.half === 1 || o.x > 9) }; }),
    knobs: knobs.map((p, i) => PP.enc[i] && { key: p.k.id, a: pin(PP.enc[i][0]), b: pin(PP.enc[i][1]) }).filter(Boolean),
    screen: scr && scr.w ? { type: scr.id, name: scr.name, w: scr.w, h: scr.h, sda: pin(PP.i2c[0]), scl: pin(PP.i2c[1]) } : null,
    rgb: K.rgb ? { pin: pin(m.led), count: (G.boards[0] ? G.boards[0].places : G.places).filter(p => p.k.kind !== "knob").length } : null,
  };
}
async function kbSaveExport(ext, content) {
  const r = await api("/api/savefile", { name: KBS().name || "keyboard", ext, content });
  return r.path;
}
async function kbOpenIn(app, path, what) {
  try { await api("/api/openin", { app, path }); toast(`<span>Opening ${what} in <b>${app === "makie" ? "Makie" : "Flashie"}</b>…</span>`, "good"); }
  catch (e) { toast(`<span>Saved to <b>${esc(path)}</b>. Open it in ${app === "makie" ? "Makie" : "Flashie"} (${esc(e.message)}).</span>`, "warn"); }
}
async function kbToMakie() {
  const K = KBS();
  if (!K.built || !S.comps.some(c => /^kb_|^mx_switch|^choc_switch|^ec11$/.test(c.fp))) { toast("Build the PCB first (Build the PCB), then Makie can make a case around it.", "warn"); return; }
  try { const path = await kbSaveExport(".makie.json", JSON.stringify(makieExport({ textures: true }))); kbOpenIn("makie", path, "the case"); }
  catch (e) { downloadFile((K.name || "keyboard") + ".makie.json", JSON.stringify(makieExport()), "application/json"); toast("Saved as a download: open it in Makie.", "warn"); }
}
async function kbToFlashie() {
  const K = KBS(), m = kbMcu(K);
  if (m.wireless) { toast(`The ${esc(m.name)} runs ZMK: use QMK / ZMK files instead.`, "warn"); return; }
  if (K.split.mode === "two") toast("Flashie makes firmware for one-piece keyboards. For split halves, use QMK / ZMK files.", "warn");
  const data = JSON.stringify(kbFlashieExport(), null, 1);
  try { const path = await kbSaveExport(".keebie.json", data); kbOpenIn("flashie", path, "the keymap"); }
  catch (e) { downloadFile((K.name || "keyboard") + ".keebie.json", data, "application/json"); toast("Saved as a download: open it in Flashie's Keyboard tab.", "warn"); }
}

// ---------------------------------------------------------------- the page
function buildKeebie(el) {
  el.innerHTML = `
  <div class="kb">
    <aside class="kb-side" id="kbSide"></aside>
    <div class="kb-main">
      <div class="kb-bar">
        <div class="seg" id="kbAdd" data-help="Add key"><span class="kb-lbl">Add key</span>${KB_SIZES.map(([w, h]) => `<button data-w="${w}" data-h="${h}" title="Add a ${w}${h !== 1 ? "×" + h : ""}u key">${h !== 1 ? `${h}u↕` : w + "u"}</button>`).join("")}<button data-iso="1" title="Add an ISO enter key">ISO⏎</button><button data-knob="1" title="Add a rotary knob (EC11 encoder)">Knob</button></div>
        <span class="grow"></span>
        <div class="seg" id="kbView" data-help="Key view"><button data-v="legend" class="on" title="Show the key labels">Labels</button><button data-v="matrix" title="Show each key's row and column in the matrix">Matrix</button><button data-v="fp" title="Show the parts on each key: solder points, diodes, LEDs">Parts</button></div>
        <button class="btn ghost icon" id="kbFit" data-tip="Fit to window">${ico("fit")}</button>
      </div>
      <div class="kb-stage" id="kbStage"><svg id="kbSvg" tabindex="0"></svg><div class="kb-hint" id="kbHint"></div>
        <div class="kb-empty" id="kbEmpty" hidden><b>No keyboard open</b><p>Start one: press <b>New keyboard</b>, pick a layout on the left, make a grid, or import a KLE layout.</p><button class="btn primary" id="kbEmptyNew" data-help="New keyboard">${ico("keyboard")}New keyboard</button></div></div>
    </div>
    <aside class="kb-props" id="kbProps"></aside>
  </div>`;
  $("#kbAdd").onclick = e => { const b = e.target.closest("button"); if (!b) return; if (b.dataset.knob) return kbAddKey(1, 1, "", "knob"); kbAddKey(+b.dataset.w || 1.25, +b.dataset.h || 2, b.dataset.iso ? "iso" : ""); };
  $$("#kbView button").forEach(b => b.onclick = () => { KV.view = b.dataset.v; $$("#kbView button").forEach(x => x.classList.toggle("on", x === b)); kbDraw(); });
  $("#kbFit").onclick = () => { KV.zoom = 0; kbDraw(); };
  $("#kbEmptyNew").onclick = () => kbNewDialog(false);
  const svg = $("#kbSvg");
  svg.addEventListener("pointerdown", kbDown); svg.addEventListener("pointermove", kbMove); svg.addEventListener("pointerup", kbUp);
  svg.addEventListener("wheel", e => { e.preventDefault(); const [x, y] = kbPt(e); const f = Math.pow(1.0015, e.deltaY); KV.zoom *= f; KV.vx = x - (x - KV.vx) * f; KV.vy = y - (y - KV.vy) * f; kbDraw(); }, { passive: false });
  svg.addEventListener("dblclick", kbDbl); svg.addEventListener("contextmenu", e => { e.preventDefault(); kbCtx(e); });
  new ResizeObserver(() => kbDraw()).observe($("#kbStage"));
}
// switch the keyboard view to Parts, so a change to the diodes shows straight away
function kbShowParts() { KV.view = "fp"; $$("#kbView button").forEach(x => x.classList.toggle("on", x.dataset.v === "fp")); }
// New keyboard / Close keyboard. Either one clears the keyboard design and, if
// the PCB tab holds the board built from it, that board too (Ctrl+Z undoes it).
function kbNewDialog(close) {
  const K = KBS(), built = !!K.built && S.comps.length > 0, has = K.keys.length > 0 || built;
  const go = preset => {
    const name = S.name;
    if (built) { const fresh = newProject(); for (const k of Object.keys(fresh)) S[k] = fresh[k]; S.netLabels = {}; }
    S.keeb = kbDefault();
    if (close || preset === "") { S.keeb.keys = []; S.keeb.preset = ""; }
    else if (preset) kbLoadPreset(preset);
    if (!built) S.name = name;
    KV.sel.clear(); KV.csel = null; KV.zoom = 0; KV.geom = null; KV.view = "legend";
    $$("#kbView button").forEach(x => x.classList.toggle("on", x.dataset.v === "legend"));
    invalidate(); commit(close ? "close keyboard" : "new keyboard");
    if (typeof fitView === "function") fitView();
    if (typeof refreshPanels === "function") refreshPanels();
    kbPage();
    toast(close ? "Keyboard closed. Ctrl+Z brings it back." : "New keyboard started.");
  };
  const presets = `<label class="fld"><span>Start from</span><select id="kn_p"><option value="ansi60">${esc((KB_PRESETS.find(p => p.id === "ansi60") || {}).name || "60%")}</option>${KB_PRESETS.filter(p => p.id !== "ansi60").map(p => `<option value="${esc(p.id)}">${esc(p.name)}</option>`).join("")}<option value="">Nothing: I'll add keys myself</option></select></label>`;
  if (!has && !close) { const m = modal(`<h3>New keyboard</h3>${presets}<div class="mfoot"><button class="btn ghost" data-close>Cancel</button><button class="btn primary" id="kn_go">Start</button></div>`, "small"); m.querySelector("#kn_go").onclick = () => { const v = m.querySelector("#kn_p").value; closeModal(); go(v); }; return; }
  const what = `<b>${esc(K.name || "this keyboard")}</b>${built ? " and the board built from it on the PCB tab" : ""}`;
  const m = modal(`<h3>${close ? "Close this keyboard?" : "Start a new keyboard?"}</h3><p class="dim">This clears ${what}. Anything you haven't saved will be lost (Ctrl+Z undoes it straight after).</p>${close ? "" : presets}
    <div class="mfoot"><button class="btn ghost" data-close>Cancel</button><button class="btn" id="kn_save">${ico("save")}Save first</button><button class="btn primary" id="kn_go">${close ? "Close keyboard" : "New keyboard"}</button></div>`, "small");
  m.querySelector("#kn_go").onclick = () => { const p = m.querySelector("#kn_p"); const v = p ? p.value : null; closeModal(); go(v); };
  m.querySelector("#kn_save").onclick = () => { closeModal(); saveProject(); };
}
function kbPage() { KBS(); kbSide(); kbDraw(); kbPropsPanel(); }
function kbChanged(label) { KV.geom = null; commit(label || "keebie"); kbDraw(); kbPropsPanel(); kbStats(); }

// ---- settings panel
function kbSide() {
  const K = KBS(), el = $("#kbSide"); if (!el) return;
  const opt = (list, cur, f = x => [x.id, x.name]) => list.map(x => { const [v, t] = f(x); return `<option value="${esc(v)}" ${v === cur ? "selected" : ""}>${esc(t)}</option>`; }).join("");
  const groups = [...new Set(KB_MCUS.map(m => m.group))];
  const sel = (id, label, html, help) => `<label class="fld" ${help ? `data-help="${help}"` : ""}><span>${label}</span><select id="${id}">${html}</select></label>`;
  const nm = (id, label, v, unit, help, step = "any") => `<label class="fld" ${help ? `data-help="${help}"` : ""}><span>${label}</span><div class="kb-inp"><input id="${id}" type="number" step="${step}" value="${v}">${unit ? `<em>${unit}</em>` : ""}</div></label>`;
  el.innerHTML = `
    <div class="kb-head"><h2>${ico("keyboard")}Keebie</h2><p>Design a custom keyboard, then let Keebie build the PCB.</p>
      <div class="kb-row"><button class="btn sm" id="kb_new" data-help="New keyboard">${ico("new")}New keyboard</button><button class="btn sm" id="kb_close" data-help="Close keyboard" ${K.keys.length || K.built ? "" : "disabled"}>${ico("close")}Close keyboard</button></div></div>
    <section><h4>1 · Layout</h4>
      <label class="fld" data-help="Keyboard name"><span>Name</span><div class="kb-inp"><input id="kb_name" value="${esc(K.name)}" spellcheck="false"></div></label>
      ${sel("kb_preset", "Start from a layout", `<option value="">Choose a layout…</option>${opt(KB_PRESETS, "")}`, "Start from a layout")}
      <div class="kb-row"><label class="fld" data-help="Grid rows"><span>Rows</span><div class="kb-inp"><input id="kb_gr" type="number" min="1" max="12" value="4"></div></label><label class="fld" data-help="Grid columns"><span>Columns</span><div class="kb-inp"><input id="kb_gc" type="number" min="1" max="24" value="12"></div></label><button class="btn sm" id="kb_grid" data-help="Make grid">Make grid</button></div>
      <div class="kb-row"><label class="fld" data-help="Number of keys"><span>Or just a number of keys</span><div class="kb-inp"><input id="kb_n" type="number" min="1" max="140" value="${K.keys.length}"></div></label><button class="btn sm" id="kb_nk" data-help="Make keys">Make</button></div>
      <div class="kb-row"><button class="btn sm" id="kb_kle_in" data-help="Import KLE">Import KLE…</button><button class="btn sm" id="kb_kle_out" data-help="Export KLE">Export KLE</button></div>
    </section>
    <section><h4>2 · Switches &amp; keycaps</h4>
      ${sel("kb_sw", "Switches", opt(KB_SWITCHES, K.sw), "Switch type")}
      <small class="kb-note">${esc(kbSwitch(K).note)}</small>
      <label class="fld" data-help="Key spacing"><span>Key spacing (centre to centre)</span><div class="kb-inp"><input id="kb_sp" type="number" min="15" max="25" step="0.05" value="${K.spacing > 0 ? K.spacing : kbSwitch(K).pitch[0]}"><em>mm</em></div></label>
      <small class="kb-note">Standard for ${esc(kbSwitch(K).name.split(",")[0])}: ${kbSwitch(K).pitch.join(" × ")} mm${K.spacing > 0 ? ` · <a href="#" id="kb_sp0">use standard</a>` : ""}</small>
      ${sel("kb_prof", "Keycap profile", opt(KB_PROFILES, K.profile, p => [p.id, `${p.name} · ${p.h}`]), "Keycap profile")}
      <small class="kb-note" id="kb_profnote"></small>
      ${sel("kb_fr", "Turn every switch footprint", [0, 90, 180, 270].map(a => `<option value="${a}" ${a === (K.fpRot || 0) ? "selected" : ""}>${a === 0 ? "0° · pins at the top (south-facing LED)" : a === 180 ? "180° · pins at the bottom (north-facing LED)" : a + "°"}</option>`).join(""), "Turn every switch footprint")}
    </section>
    <section><h4>3 · Shape</h4>
      ${sel("kb_ol", "Board outline", opt([{ id: "hug", name: "Hug the keys" }, { id: "rect", name: "Rectangle" }, { id: "custom", name: "Custom (drag the corners)" }], K.outline.mode), "Board outline")}
      <div class="kb-row">${nm("kb_mg", "Edge margin", K.outline.margin, "mm", "Edge margin")}${nm("kb_rad", "Rounding", K.outline.radius, "mm", "Rounding")}</div>
      ${nm("kb_holes", "Mounting holes (M2)", K.holes, "", "Mounting holes", 1)}
      ${nm("kb_arc", "Arc (bend radius, 0 = flat)", K.arc, "u", "Arc", 1)}
    </section>
    <section><h4>4 · Split</h4>
      ${sel("kb_split", "Split", opt([{ id: "none", name: "One piece" }, { id: "angled", name: "One board, angled halves (Alice-style)" }, { id: "two", name: "Two separate halves" }], K.split.mode), "Split")}
      <div id="kb_splitopts" ${K.split.mode === "none" ? "hidden" : ""}>
        <div class="kb-row">${nm("kb_sat", "Split after", K.split.at, "u", "Split after", 0.25)}${nm("kb_sang", "Splay angle", K.split.angle, "°", "Splay angle", 1)}</div>
        ${nm("kb_sgap", "Gap between halves", K.split.gap, "mm", "Gap between halves", 1)}
      </div>
    </section>
    <section><h4>5 · Electronics</h4>
      <label class="fld" data-help="Controller"><span>Controller</span><select id="kb_mcu">${groups.map(g => `<optgroup label="${esc(g)}">${opt(KB_MCUS.filter(m => m.group === g), K.mcu)}</optgroup>`).join("")}</select></label>
      <small class="kb-note">${esc(kbMcu(K).desc)} <b>Firmware:</b> ${esc(kbMcu(K).fw)}.</small>
      ${sel("kb_pos", "Controller position", opt([{ id: "left", name: "Top left" }, { id: "centre", name: "Top centre" }, { id: "right", name: "Top right" }, { id: "side", name: K.split.mode === "two" ? "Beside the inner column" : "Beside the keys (right)" }], K.ctrlPos), "Controller position")}
      ${sel("kb_dio", "Diodes", opt([{ id: "d_sod123", name: "SOD-123 surface-mount (1N4148W)" }, { id: "d_do35", name: "Through-hole (1N4148)" }], K.diode), "Diodes")}
      ${sel("kb_dstyle", "Diode layout", opt(KB_DSTYLES, K.dstyle || "beside"), "Diode layout")}
      ${sel("kb_dside", "Diodes on", opt([{ id: "bottom", name: "The back (hidden under the keys)" }, { id: "top", name: "The front (on show, for a clear case)" }], K.dside || "bottom"), "Diodes on")}
      ${sel("kb_dir", "Diode direction", opt([{ id: "COL2ROW", name: "COL2ROW (most common)" }, { id: "ROW2COL", name: "ROW2COL" }], K.dir), "Diode direction")}
      <label class="kb-check" data-help="Per-key RGB"><input type="checkbox" id="kb_rgb" ${K.rgb ? "checked" : ""}> Per-key RGB LEDs (SK6812 MINI-E)</label>
      <label class="kb-check" data-help="LED capacitors" ${K.rgb ? "" : 'style="opacity:.5"'}><input type="checkbox" id="kb_lcap" ${K.ledCap ? "checked" : ""} ${K.rgb ? "" : "disabled"}> A 100nF capacitor beside each LED</label>
      ${sel("kb_pack", "Controller's parts", opt(KB_PACKS, K.pack || "compact"), "Controller parts layout")}
      <small class="kb-note">Click the controller on the keyboard to drag it or turn it. Use <b>Parts</b> above the keyboard to see every diode and LED.</small>
      ${sel("kb_oled", "Screen", opt(KB_SCREENS, K.oled || "none"), "Screen")}
      <small class="kb-note">Knobs: pick a key and set its <b>Type</b> to <i>Rotary knob</i>, or press <b>Knob</b> above the keyboard.</small>
    </section>
    <section class="kb-stats" id="kbStats"></section>
    <section><h4>6 · Make it</h4>
      ${K.split.mode === "two" ? sel("kb_half", "Half to build", opt([{ id: "0", name: "Left half" }, { id: "1", name: "Right half" }], String(K.half || 0)), "Half to build") : ""}
      <button class="btn primary wide" id="kb_build" data-help="Build the PCB">${ico("magic")}Build the PCB</button>
      <div class="kb-row" style="margin-top:8px"><button class="btn sm" id="kb_plate" data-help="Switch plate">Plate (SVG)</button><button class="btn sm" id="kb_dxf" data-help="Switch plate DXF">Plate (DXF)</button></div>
      <div class="kb-row"><button class="btn sm" id="kb_makie" data-help="Case in Makie">${ico("box")}Case in Makie</button><button class="btn sm" id="kb_flashie" data-help="Firmware in Flashie">${ico("chip")}Firmware in Flashie</button></div>
      <button class="btn sm wide" id="kb_fw" data-help="Firmware files">QMK / ZMK files…</button>
    </section>`;
  const on = (id, ev, f) => { const e = $("#" + id); if (e) e[ev] = f; };
  const val = id => $("#" + id).value, numv = id => parseFloat($("#" + id).value);
  on("kb_name", "onchange", () => { K.name = val("kb_name").trim() || "Keyboard"; kbChanged("name"); });
  on("kb_preset", "onchange", () => { const v = val("kb_preset"); if (!v) return; kbLoadPreset(v); kbSide(); kbChanged("layout"); KV.zoom = 0; kbDraw(); });
  on("kb_grid", "onclick", () => { K.keys = kbGrid(Math.max(1, Math.min(12, numv("kb_gr") | 0)), Math.max(1, Math.min(24, numv("kb_gc") | 0))); K.split.mode = "none"; K.outline.custom = null; KV.sel.clear(); KV.zoom = 0; kbSide(); kbChanged("grid"); });
  on("kb_nk", "onclick", () => { const n = Math.max(1, Math.min(140, numv("kb_n") | 0)); const c = Math.ceil(Math.sqrt(n * 2.2)); K.keys = kbGrid(Math.ceil(n / c), c).slice(0, n); K.split.mode = "none"; K.outline.custom = null; KV.sel.clear(); KV.zoom = 0; kbSide(); kbChanged("keys"); });
  on("kb_kle_in", "onclick", kbImportDialog);
  on("kb_kle_out", "onclick", () => downloadFile((K.name || "keyboard") + "-kle.json", kbToKLE(), "application/json"));
  on("kb_sw", "onchange", () => { K.sw = val("kb_sw"); const fam = kbSwitch(K).fam; if ((KB_PROFILES.find(p => p.id === K.profile) || {}).fam !== fam) K.profile = (KB_PROFILES.find(p => p.fam === fam) || KB_PROFILES[0]).id; kbSide(); kbChanged("switch"); });
  on("kb_sp", "onchange", () => {
    const std = kbSwitch(K).pitch[0], v = numv("kb_sp");
    K.spacing = !(v > 0) || Math.abs(v - std) < 0.001 ? 0 : Math.max(15, Math.min(25, v));
    kbSide(); kbChanged("spacing");
  });
  on("kb_sp0", "onclick", e => { e.preventDefault(); K.spacing = 0; kbSide(); kbChanged("spacing"); });
  on("kb_prof", "onchange", () => { K.profile = val("kb_prof"); kbSide(); kbChanged("profile"); });
  on("kb_fr", "onchange", () => { K.fpRot = +val("kb_fr"); kbChanged("footprint turn"); });
  on("kb_ol", "onchange", () => { const v = val("kb_ol"); if (v === "custom" && !K.outline.custom) { const G = kbGeom(); K.outline.custom = G.boards.map(b => b && b.polys[0] ? kbSimplify([...b.polys[0], b.polys[0][0]], 0.8).slice(0, -1).map(([x, y]) => [+x.toFixed(2), +y.toFixed(2)]) : null); } K.outline.mode = v; kbChanged("outline"); toast(v === "custom" ? "Drag the corner dots to shape the board. Double-click an edge to add a corner, right-click a corner to remove it." : "Outline updated"); });
  on("kb_mg", "onchange", () => { K.outline.margin = Math.max(0, Math.min(20, numv("kb_mg") || 0)); kbChanged("margin"); });
  on("kb_rad", "onchange", () => { K.outline.radius = Math.max(0, Math.min(30, numv("kb_rad") || 0)); kbChanged("rounding"); });
  on("kb_holes", "onchange", () => { K.holes = Math.max(0, Math.min(16, numv("kb_holes") | 0)); kbChanged("holes"); });
  on("kb_arc", "onchange", () => { const v = numv("kb_arc") || 0; K.arc = Math.abs(v) < 3 ? 0 : Math.max(-400, Math.min(400, v)); kbChanged("arc"); });
  on("kb_split", "onchange", () => { K.split.mode = val("kb_split"); if (K.split.mode !== "none" && (K.split.at <= 0 || !K.keys.some(k => k.x + k.w / 2 >= K.split.at))) { const xs = K.keys.map(k => k.x + k.w / 2); K.split.at = +((Math.min(...xs) + Math.max(...xs)) / 2).toFixed(2); } if (K.split.mode === "angled" && !K.split.angle) K.split.angle = 10; if (K.split.mode === "two" && !K.split.gap) K.split.gap = 30; K.outline.custom = null; kbSide(); kbChanged("split"); });
  on("kb_sat", "onchange", () => { K.split.at = numv("kb_sat") || 0; kbChanged("split"); });
  on("kb_sang", "onchange", () => { K.split.angle = Math.max(-45, Math.min(45, numv("kb_sang") || 0)); kbChanged("splay"); });
  on("kb_sgap", "onchange", () => { K.split.gap = Math.max(0, Math.min(400, numv("kb_sgap") || 0)); kbChanged("gap"); });
  on("kb_mcu", "onchange", () => { K.mcu = val("kb_mcu"); kbSide(); kbChanged("controller"); });
  on("kb_pos", "onchange", () => { K.ctrlPos = val("kb_pos"); K.ctrlOff = []; kbChanged("controller position"); });
  on("kb_dio", "onchange", () => { K.diode = val("kb_dio"); kbShowParts(); kbChanged("diodes"); });
  on("kb_dir", "onchange", () => { K.dir = val("kb_dir"); kbChanged("diode direction"); });
  on("kb_rgb", "onchange", () => { K.rgb = $("#kb_rgb").checked; kbSide(); kbChanged("rgb"); });
  on("kb_lcap", "onchange", () => { K.ledCap = $("#kb_lcap").checked; kbChanged("LED capacitors"); });
  on("kb_dstyle", "onchange", () => { K.dstyle = val("kb_dstyle"); kbShowParts(); kbChanged("diode layout"); });
  on("kb_dside", "onchange", () => { K.dside = val("kb_dside"); kbShowParts(); kbChanged("diode side"); });
  on("kb_pack", "onchange", () => { K.pack = val("kb_pack"); kbChanged("controller parts"); });
  on("kb_new", "onclick", () => kbNewDialog(false));
  on("kb_close", "onclick", () => kbNewDialog(true));
  on("kb_oled", "onchange", () => { K.oled = val("kb_oled"); kbChanged("screen"); });
  on("kb_makie", "onclick", kbToMakie);
  on("kb_flashie", "onclick", kbToFlashie);
  on("kb_half", "onchange", () => { K.half = +val("kb_half"); kbChanged("half"); });
  on("kb_build", "onclick", () => {
    const go = () => kbBuild(K.split.mode === "two" ? K.half || 0 : 0);
    if (S.comps.length && !(S.keeb && S.keeb.built)) { const m = modal(`<h3>Replace the current board?</h3><p class="dim">Building the keyboard replaces everything on the PCB tab (${S.comps.length} parts). Save your current board first if you want to keep it.</p><div class="row btns" style="display:flex;gap:8px;justify-content:flex-end;margin-top:14px"><button class="btn" data-close>Cancel</button><button class="btn primary" id="kb_go">Build the keyboard</button></div>`); m.querySelector("#kb_go").onclick = () => { closeModal(); go(); }; }
    else go();
  });
  on("kb_plate", "onclick", () => downloadFile((K.name || "keyboard") + "-plate.svg", kbPlateSVG(), "image/svg+xml"));
  on("kb_dxf", "onclick", () => downloadFile((K.name || "keyboard") + "-plate.dxf", kbPlateDXF(), "application/dxf"));
  on("kb_fw", "onclick", kbFirmwareDialog);
  const pr = KB_PROFILES.find(p => p.id === K.profile); const fam = kbSwitch(K).fam;
  if ($("#kb_profnote")) $("#kb_profnote").innerHTML = pr ? esc(pr.note) + (pr.fam !== fam ? ` <b class="kb-warn">These caps don't fit ${fam === "choc" ? "Choc" : fam === "alps" ? "Alps" : "MX-style"} switches.</b>` : "") : "";
  kbStats();
}
function kbStats() {
  const el = $("#kbStats"); if (!el) return;
  const K = KBS(), G = kbGeom(), M = G.matrix, PP = kbPinPlan(K), base = kbBasePins(K), extra = PP.enc.length * 2 + (PP.i2c ? 2 : 0), pins = PP.pins, need = M.rows + M.cols;
  const nKnob = kbKnobCount(K);
  const sizes = {}; for (const k of K.keys.filter(q => q.kind !== "knob")) { const s = k.shape === "iso" ? "ISO enter" : (k.h > k.w ? `${k.h}u vertical` : `${k.w}u`); sizes[s] = (sizes[s] || 0) + 1; }
  const stabs = G.places.filter(p => kbStab(K, G, p)).length;
  const dims = G.boards.filter(Boolean).map(b => { let x0 = 1e9, y0 = 1e9, x1 = -1e9, y1 = -1e9; for (const pl of b.polys) for (const [x, y] of pl) { x0 = Math.min(x0, x); y0 = Math.min(y0, y); x1 = Math.max(x1, x); y1 = Math.max(y1, y); } return `${(x1 - x0).toFixed(0)} × ${(y1 - y0).toFixed(0)} mm`; });
  const warn = [];
  if (PP.enc.length < nKnob) warn.push(`Not enough free pins for ${nKnob} knob${nKnob > 1 ? "s" : ""}: each knob needs 2 pins of its own.`);
  if (need > pins.length) warn.push(`The matrix needs ${need} pins but the ${G.mcu.name} has ${pins.length} free. Try: ${KB_MCUS.filter(m => m.pins.length - 2 >= need).map(m => m.name.split(" (")[0]).slice(0, 3).join(", ") || "a split layout"}.`);
  if (M.dup.length) warn.push(`${M.dup.length} key(s) share a matrix position with another key: check the row/column overrides.`);
  if (G.boards.some(b => b && !b.polys.length)) warn.push("Couldn't make an outline: try a bigger margin.");
  if (G.boards.some(b => b && b.area && b.places.some(p => { const kc = kbCorners(p, G.sx, G.sy, -0.5), ap = b.area.poly(-0.3); return kc.some(([x, y]) => ptInPoly(x, y, ap)) || ap.some(([x, y]) => ptInPoly(x, y, kc)); }))) warn.push("The controller overlaps some keys: drag it clear, or press Put it back.");
  if (K.rgb && G.mcu.volt === 3.3) warn.push("3.3V controller driving 5V LEDs: usually fine with SK6812, but keep the data track short.");
  if (G.mcu.bare) warn.push(`Bare chips have fine-pitch pins: order the board with factory assembly (PCBA) for the chip. Auto-route does the keys; tracks right at the chip's ${LIB_BY_ID[G.mcu.fp].pads[1] && Math.abs(LIB_BY_ID[G.mcu.fp].pads[1].y - LIB_BY_ID[G.mcu.fp].pads[0].y).toFixed(1)}mm-pitch pins often need finishing by hand, so run Check before ordering.`);
  el.innerHTML = `<h4>Your keyboard</h4><div class="kb-outs">
    <div><span>Keys</span><b>${K.keys.length}</b></div><div><span>Matrix</span><b>${M.rows} × ${M.cols}</b></div>
    <div class="${need > pins.length || PP.enc.length < nKnob ? "bad" : ""}"><span>Pins used</span><b>${need + extra} / ${base.length}</b></div><div><span>Board${dims.length > 1 ? "s" : ""}</span><b>${dims.join(" + ") || "—"}</b></div></div>
    <div class="kb-bom"><b>Keycaps:</b> ${Object.entries(sizes).sort((a, b) => b[1] - a[1]).map(([s, n]) => `${n}× ${s}`).join(", ")}${stabs ? `<br><b>Stabilisers:</b> ${stabs}` : ""}<br><b>Diodes:</b> ${K.keys.length} (${K.dside === "top" ? "front" : "back"})${K.rgb ? ` · <b>LEDs:</b> ${K.keys.length - K.keys.filter(k => k.kind === "knob").length} (back)` : ""}${K.rgb && K.ledCap ? ` · <b>LED capacitors:</b> ${G.places.filter(p => kbSpotsFor(K, G, p).cap).length}` : ""}${nKnob ? `<br><b>Knobs:</b> ${K.keys.filter(k => k.kind === "knob").length} × EC11 rotary encoder` : ""}${PP.i2c ? `<br><b>Screen:</b> ${esc(KB_SCREENS.find(x => x.id === K.oled).name)} on ${PP.i2c.join(" / ")}` : ""}</div>
    ${warn.map(w => `<p class="kb-warn">${esc(w)}</p>`).join("")}`;
}

// ---- the controller's hand placement (one per half)
function kbCtrlOff(K, h) { if (!Array.isArray(K.ctrlOff)) K.ctrlOff = []; return K.ctrlOff[h] || (K.ctrlOff[h] = { x: 0, y: 0, rot: 0 }); }
function kbCtrlPanel(el, K, h) {
  const off = kbCtrlOff(K, h), mcu = kbMcu(K);
  el.innerHTML = `<h4>Controller${K.split.mode === "two" ? (h ? " (right half)" : " (left half)") : ""}</h4>
    <p class="dim">${esc(mcu.name)}${mcu.bare ? " and its circuitry" : ""}. Drag it to move it, or use the boxes below.</p>
    <div class="kb-row"><label class="fld" data-help="Controller nudge X"><span>Move across</span><div class="kb-inp"><input id="kc_x" type="number" step="0.5" value="${off.x || 0}"><em>mm</em></div></label><label class="fld" data-help="Controller nudge Y"><span>Move down</span><div class="kb-inp"><input id="kc_y" type="number" step="0.5" value="${off.y || 0}"><em>mm</em></div></label></div>
    <label class="fld" data-help="Controller angle"><span>Angle</span><div class="kb-inp"><input id="kc_r" type="number" step="15" value="${off.rot || 0}"><em>°</em></div></label>
    <div class="kb-row"><button class="btn sm" id="kc_l" data-help="Turn controller left">↺ 90°</button><button class="btn sm" id="kc_rr" data-help="Turn controller right">↻ 90°</button></div>
    <button class="btn sm wide" id="kc_0" data-help="Reset controller">Put it back where Keebie chose</button>
    ${K.outline.mode === "custom" ? `<p class="kb-warn">Your outline is custom: drag its corners so the board still covers the controller.</p>` : ""}
    <ul class="kb-help"><li>Drag to move (<kbd>Alt</kbd> for fine steps)</li><li><kbd>Arrows</kbd> move 1 mm (<kbd>Shift</kbd> 5 mm)</li><li><kbd>R</kbd> / <kbd>Shift R</kbd> turn 15°</li><li><kbd>Esc</kbd> deselect</li></ul>`;
  const num = id => parseFloat($("#" + id).value) || 0;
  const set = f => { f(); kbChanged("controller"); };
  $("#kc_x").onchange = () => set(() => off.x = Math.max(-300, Math.min(300, num("kc_x"))));
  $("#kc_y").onchange = () => set(() => off.y = Math.max(-300, Math.min(300, num("kc_y"))));
  $("#kc_r").onchange = () => set(() => off.rot = ((num("kc_r") % 360) + 360) % 360);
  $("#kc_l").onclick = () => set(() => off.rot = (((off.rot || 0) - 90) % 360 + 360) % 360);
  $("#kc_rr").onclick = () => set(() => off.rot = ((off.rot || 0) + 90) % 360);
  $("#kc_0").onclick = () => set(() => { off.x = 0; off.y = 0; off.rot = 0; });
}

// ---- key properties panel
function kbPropsPanel() {
  const el = $("#kbProps"); if (!el) return;
  const K = KBS(), sel = K.keys.filter(k => KV.sel.has(k.id));
  if (!sel.length && KV.csel != null && kbGeom().boards[KV.csel]) { kbCtrlPanel(el, K, KV.csel); return; }
  if (!sel.length) {
    el.innerHTML = `<h4>Keys</h4><p class="dim">Click a key to change it; Shift-click or drag a box to pick several.</p>
      <ul class="kb-help"><li><kbd>Arrows</kbd> move ¼ key (<kbd>Shift</kbd> 1 key)</li><li><kbd>R</kbd> / <kbd>Shift R</kbd> turn 5°</li><li><kbd>F</kbd> turn the switch footprint 90°</li><li><kbd>Ctrl D</kbd> duplicate · <kbd>Del</kbd> delete</li><li><kbd>Ctrl A</kbd> select all · wheel to zoom, right-drag to pan</li></ul>
      <p class="dim">Click the controller to move or turn it.</p>`;
    return;
  }
  const one = sel[0], same = f => sel.every(k => k[f] === one[f]) ? one[f] : "";
  const G = kbGeom(), mp = sel.length === 1 ? G.matrix.pos.get(one.id) : null;
  const sizeOpts = [1, 1.25, 1.5, 1.75, 2, 2.25, 2.5, 2.75, 3, 4, 4.5, 5.5, 6, 6.25, 7, 8, 10];
  el.innerHTML = `<h4>${sel.length === 1 ? "Key" : sel.length + " keys"}</h4>
    <label class="fld" data-help="Key label"><span>Label</span><div class="kb-inp"><input id="kp_label" value="${esc(same("label"))}" spellcheck="false"></div></label>
    <div class="kb-row"><label class="fld" data-help="Key width"><span>Width</span><select id="kp_w">${!same("w") ? '<option value="">mixed</option>' : ""}${sizeOpts.map(s => `<option value="${s}" ${same("w") === s ? "selected" : ""}>${s}u</option>`).join("")}</select></label>
    <label class="fld" data-help="Key height"><span>Height</span><select id="kp_h">${!same("h") ? '<option value="">mixed</option>' : ""}${[1, 1.5, 2].map(s => `<option value="${s}" ${same("h") === s ? "selected" : ""}>${s}u</option>`).join("")}</select></label></div>
    <label class="fld" data-help="Key type"><span>Type</span><select id="kp_kind">${!sel.every(k => (k.kind || "") === (one.kind || "")) ? '<option value="">mixed</option>' : ""}${[["key", "Key switch"], ["knob", "Rotary knob (EC11)"]].map(([v, t]) => `<option value="${v}" ${sel.every(k => (k.kind || "key") === v) ? "selected" : ""}>${t}</option>`).join("")}</select></label>
    <label class="fld" data-help="Key shape"><span>Shape</span><select id="kp_shape">${[["", "Normal"], ["iso", "ISO enter"], ["stepped", "Stepped (Caps Lock)"]].map(([v, t]) => `<option value="${v}" ${same("shape") === v ? "selected" : ""}>${t}</option>`).join("")}</select></label>
    <div class="kb-row"><label class="fld" data-help="Key angle"><span>Angle</span><div class="kb-inp"><input id="kp_r" type="number" step="1" value="${same("r") === "" ? "" : same("r")}"><em>°</em></div></label>
    <label class="fld" data-help="Switch footprint turn"><span>Solder points</span><select id="kp_fr">${!sel.every(k => (k.fr || 0) === (one.fr || 0)) ? '<option value="">mixed</option>' : ""}${[0, 90, 180, 270].map(a => `<option value="${a}" ${(one.fr || 0) === a && sel.every(k => (k.fr || 0) === a) ? "selected" : ""}>+${a}°</option>`).join("")}</select></label></div>
    ${sel.length === 1 ? `<div class="kb-row"><label class="fld" data-help="Matrix row"><span>Row ${one.row == null ? `(auto: ${mp ? mp[0] : "?"})` : ""}</span><div class="kb-inp"><input id="kp_row" type="number" min="0" value="${one.row ?? ""}" placeholder="auto"></div></label><label class="fld" data-help="Matrix column"><span>Column ${one.col == null ? `(auto: ${mp ? mp[1] : "?"})` : ""}</span><div class="kb-inp"><input id="kp_col" type="number" min="0" value="${one.col ?? ""}" placeholder="auto"></div></label></div>
      <div class="kb-row"><label class="fld"><span>X</span><div class="kb-inp"><input id="kp_x" type="number" step="0.25" value="${one.x}"><em>u</em></div></label><label class="fld"><span>Y</span><div class="kb-inp"><input id="kp_y" type="number" step="0.25" value="${one.y}"><em>u</em></div></label></div>` : ""}
    ${G.sw.stabs && !sel.every(k => k.kind === "knob") ? `<label class="fld" data-help="Stabiliser"><span>Stabiliser</span><select id="kp_stab">${[["auto", "Automatic (2u and longer)"], ["none", "None"]].map(([v, t]) => `<option value="${v}" ${same("stab") === v ? "selected" : ""}>${t}</option>`).join("")}</select></label>` : ""}
    <div class="kb-row"><button class="btn sm" id="kp_dup" data-help="Duplicate keys">Duplicate</button><button class="btn sm danger" id="kp_del" data-help="Delete keys">Delete</button></div>`;
  const ch = (id, f) => { const e = $("#" + id); if (e) e.onchange = () => { f(e.value); kbChanged("key"); }; };
  ch("kp_label", v => sel.forEach(k => k.label = v));
  ch("kp_kind", v => v && sel.forEach(k => { if (v === "knob") { k.kind = "knob"; k.w = 1; k.h = 1; k.shape = ""; if (!k.label) k.label = "Knob"; } else delete k.kind; }));
  ch("kp_w", v => v && kbResizeW(sel, +v));
  ch("kp_h", v => v && sel.forEach(k => k.h = +v));
  ch("kp_shape", v => sel.forEach(k => { k.shape = v; if (v === "iso") { kbResizeW([k], 1.25); k.h = 2; } if (v === "stepped") kbResizeW([k], Math.max(k.w, 1.75)); }));
  ch("kp_r", v => v !== "" && sel.forEach(k => k.r = +v || 0));
  ch("kp_fr", v => v !== "" && sel.forEach(k => k.fr = +v));
  ch("kp_row", v => one.row = v === "" ? null : Math.max(0, v | 0));
  ch("kp_col", v => one.col = v === "" ? null : Math.max(0, v | 0));
  ch("kp_x", v => { if (v !== "" && isFinite(+v)) one.x = Math.max(-10, Math.min(60, +v)); }); ch("kp_y", v => { if (v !== "" && isFinite(+v)) one.y = Math.max(-10, Math.min(40, +v)); });
  ch("kp_stab", v => sel.forEach(k => k.stab = v));
  const d = $("#kp_dup"); if (d) d.onclick = kbDuplicate;
  const x = $("#kp_del"); if (x) x.onclick = kbDelete;
}

// ---- editing actions
// change key widths, sliding the keys to their right along the row so nothing overlaps
function kbResizeW(keys, w) {
  const K = KBS(), G = kbGeom(), pos = G.matrix.pos, half = new Map(G.places.map(p => [p.k.id, p.half]));
  const sameRow = (a, b) => Math.abs(a.y - b.y) < 0.01 || (pos.get(a.id) && pos.get(b.id) && pos.get(a.id)[0] === pos.get(b.id)[0] && Math.abs(a.y - b.y) < 0.75);
  for (const k of [...keys].sort((a, b) => b.x - a.x)) {
    const d = w - k.w; if (!d) continue;
    const edge = k.x + k.w - 0.001;
    if (!k.r) for (const o of K.keys) if (o !== k && !o.r && half.get(o.id) === half.get(k.id) && o.x >= edge && sameRow(o, k)) o.x = Math.round((o.x + d) * 1e4) / 1e4;
    k.w = w;
  }
}
function kbAddKey(w, h, shape, kind) {
  const K = KBS(); let y = K.keys.length ? Math.max(...K.keys.map(k => k.y + k.h)) + 0.25 : 0;
  const k = { id: kbId(), x: 0, y, w: shape === "iso" ? 1.25 : w, h: shape === "iso" ? 2 : h, r: 0, label: "", shape, fr: 0, row: null, col: null, stab: "auto" };
  if (kind) { k.kind = kind; k.label = "Knob"; }
  K.keys.push(k); KV.sel = new Set([k.id]); kbChanged("add key");
}
function kbDuplicate() { const K = KBS(); const add = K.keys.filter(k => KV.sel.has(k.id)).map(k => ({ ...k, id: kbId(), y: k.y + 1.25, row: null, col: null })); K.keys.push(...add); KV.sel = new Set(add.map(k => k.id)); kbChanged("duplicate"); }
function kbDelete() { const K = KBS(); K.keys = K.keys.filter(k => !KV.sel.has(k.id)); KV.sel.clear(); kbChanged("delete keys"); }
function kbKey(e) {
  if (UI.mode !== "keeb") return false;
  const K = KBS(), k = e.key.toLowerCase(), ctrl = e.ctrlKey || e.metaKey, sel = K.keys.filter(q => KV.sel.has(q.id));
  if (ctrl && k === "a") { e.preventDefault(); KV.sel = new Set(K.keys.map(q => q.id)); kbDraw(); kbPropsPanel(); return true; }
  if (ctrl && k === "d") { e.preventDefault(); if (sel.length) kbDuplicate(); return true; }
  if (ctrl) return false;
  if (KV.csel != null && !sel.length) {
    const off = kbCtrlOff(K, KV.csel), st = e.shiftKey ? 5 : 1, mv = { arrowleft: [-st, 0], arrowright: [st, 0], arrowup: [0, -st], arrowdown: [0, st] }[k];
    if (mv) { e.preventDefault(); off.x = +((off.x || 0) + mv[0]).toFixed(2); off.y = +((off.y || 0) + mv[1]).toFixed(2); kbChanged("move controller"); return true; }
    if (k === "r") { e.preventDefault(); off.rot = (((off.rot || 0) + (e.shiftKey ? -15 : 15)) % 360 + 360) % 360; kbChanged("turn controller"); return true; }
    if (e.key === "Escape") { KV.csel = null; kbDraw(); kbPropsPanel(); return true; }
  }
  if ((e.key === "Delete" || e.key === "Backspace") && sel.length) { e.preventDefault(); kbDelete(); return true; }
  if (e.key === "Escape") { KV.sel.clear(); kbDraw(); kbPropsPanel(); return true; }
  const step = e.shiftKey ? 1 : 0.25, mv = { arrowleft: [-step, 0], arrowright: [step, 0], arrowup: [0, -step], arrowdown: [0, step] }[k];
  if (mv && sel.length) { e.preventDefault(); sel.forEach(q => { q.x = +(q.x + mv[0]).toFixed(4); q.y = +(q.y + mv[1]).toFixed(4); }); kbChanged("move keys"); return true; }
  if (k === "r" && sel.length) { e.preventDefault(); sel.forEach(q => q.r = ((q.r || 0) + (e.shiftKey ? -5 : 5))); kbChanged("rotate keys"); return true; }
  if (k === "f" && sel.length) { e.preventDefault(); sel.forEach(q => q.fr = ((q.fr || 0) + 90) % 360); kbChanged("turn footprint"); return true; }
  return false;
}

// ---- dialogs
function kbImportDialog() {
  const m = modal(`<h3>Import a layout</h3><p class="dim">Paste the raw data from <b>keyboard-layout-editor.com</b> (Raw data tab), a KLE .json file, or a QMK <code>info.json</code> / <code>keyboard.json</code>.</p>
    <label class="fld"><textarea id="kb_kle" rows="12" placeholder='["Esc","1","2"],[{"w":1.5},"Tab","Q"]'></textarea></label>
    <div style="display:flex;gap:8px;justify-content:flex-end"><button class="btn" data-close>Cancel</button><button class="btn primary" id="kb_imp">Import</button></div>`, "kbw");
  m.querySelector("#kb_imp").onclick = () => {
    const t = m.querySelector("#kb_kle").value.trim(); const K = KBS();
    try {
      let keys;
      const j = (() => { try { return JSON.parse(t); } catch { return null; } })();
      if (j && !Array.isArray(j) && j.layouts) {
        const L = Object.values(j.layouts)[0].layout;
        keys = L.map(q => { const w = q.w || 1, h = q.h || 1; let cx = q.x + w / 2, cy = q.y + h / 2; if (q.r) { const [rx, ry] = [q.rx ?? 0, q.ry ?? 0]; const a = q.r * Math.PI / 180; const dx = cx - rx, dy = cy - ry; cx = rx + dx * Math.cos(a) - dy * Math.sin(a); cy = ry + dx * Math.sin(a) + dy * Math.cos(a); } return { id: kbId(), x: cx - w / 2, y: cy - h / 2, w, h, r: q.r || 0, label: q.label || "", shape: "", fr: 0, row: q.matrix ? q.matrix[0] : null, col: q.matrix ? q.matrix[1] : null, stab: "auto" }; });
        if (j.keyboard_name) K.name = j.keyboard_name;
      } else keys = kbFromKLE(t);
      K.keys = keys; K.split.mode = "none"; K.outline.custom = null; KV.sel.clear(); KV.zoom = 0; closeModal(); kbSide(); kbChanged("import"); toast(`Imported ${keys.length} keys`);
    } catch (e) { toast("Couldn't read that layout: " + esc(e.message), "bad"); }
  };
}
function kbFirmwareDialog() {
  const K = KBS(), m = kbMcu(K);
  if (m.zmk) {
    const txt = kbZMK();
    const d = modal(`<h3>ZMK firmware files</h3><p class="dim">The ${esc(m.name)} runs <b>ZMK</b>. This overlay describes your matrix; add a keymap with the ZMK docs' "new shield" guide.</p><label class="fld"><textarea rows="14" readonly>${esc(txt)}</textarea></label><div style="display:flex;gap:8px;justify-content:flex-end"><button class="btn" data-close>Close</button><button class="btn primary" id="fw_dl">Download .overlay</button></div>`, "kbw");
    d.querySelector("#fw_dl").onclick = () => downloadFile((K.name || "keyboard").toLowerCase().replace(/[^a-z0-9]+/g, "_") + ".overlay", txt, "text/plain");
    return;
  }
  const q = kbQMK(), kb = JSON.stringify(q.kb, null, 2), km = JSON.stringify(q.keymap, null, 2);
  const d = modal(`<h3>QMK firmware files</h3><p class="dim">Put <code>keyboard.json</code> in <code>qmk_firmware/keyboards/${esc(q.name)}/</code> and <code>keymap.json</code> in <code>keymaps/default/</code>. ${esc(q.note)}. Vial needs a few extra files (see the Vial docs).</p>
    <label class="fld"><span>keyboard.json</span><textarea rows="10" readonly>${esc(kb)}</textarea></label><div style="display:flex;gap:8px;justify-content:flex-end"><button class="btn" data-close>Close</button><button class="btn" id="fw_km">Download keymap.json</button><button class="btn primary" id="fw_kb">Download keyboard.json</button></div>`, "kbw");
  d.querySelector("#fw_kb").onclick = () => downloadFile("keyboard.json", kb, "application/json");
  d.querySelector("#fw_km").onclick = () => downloadFile("keymap.json", km, "application/json");
}

// ---------------------------------------------------------------- drawing
function kbPt(e) { const svg = $("#kbSvg"), r = svg.getBoundingClientRect(); return [KV.vx + (e.clientX - r.left) * KV.zoom, KV.vy + (e.clientY - r.top) * KV.zoom]; }
function kbFit(G) {
  const svg = $("#kbSvg"), r = svg.getBoundingClientRect(); if (!r.width) return;
  let x0 = 1e9, y0 = 1e9, x1 = -1e9, y1 = -1e9;
  for (const b of G.boards) if (b) for (const pl of b.polys) for (const [x, y] of pl) { x0 = Math.min(x0, x); y0 = Math.min(y0, y); x1 = Math.max(x1, x); y1 = Math.max(y1, y); }
  for (const p of G.places) { x0 = Math.min(x0, p.cx - 20); x1 = Math.max(x1, p.cx + 20); y0 = Math.min(y0, p.cy - 20); y1 = Math.max(y1, p.cy + 20); }
  if (x0 > x1) { x0 = 0; y0 = 0; x1 = 200; y1 = 100; }
  KV.zoom = Math.max((x1 - x0 + 30) / r.width, (y1 - y0 + 30) / r.height);
  KV.vx = (x0 + x1) / 2 - r.width * KV.zoom / 2; KV.vy = (y0 + y1) / 2 - r.height * KV.zoom / 2;
}
// a small drawing of a part (diode, LED, capacitor) at a spot in the footprint frame
function kbPartSVG(fpId, sp, cls) {
  const fp = LIB_BY_ID[fpId], b = fp.body;
  let g = `<rect x="${-b.w / 2}" y="${-b.h / 2}" width="${b.w}" height="${b.h}" class="${cls}"/>`;
  for (const q of fp.pads) g += q.drill ? `<circle cx="${q.x}" cy="${q.y}" r="${q.w / 2}" class="pp"/>` : `<rect x="${q.x - q.w / 2}" y="${q.y - q.h / 2}" width="${q.w}" height="${q.h}" class="pp"/>`;
  if (b.stripe) g += `<rect x="${-b.w / 2 + 0.2}" y="${-b.h / 2}" width="0.45" height="${b.h}" class="st"/>`;
  return `<g transform="translate(${sp.x},${sp.y}) rotate(${sp.rot})">${g}</g>`;
}
function kbCapColour(k, prof) {
  const mod = (k.label || "").length > 1 && !/^F\d+$/.test(k.label) || k.w > 1.6 || k.h > 1;
  return mod ? ["#3a4354", "#4a5569", "#d6dde9"] : ["#cfd6e0", "#e9eef5", "#222a36"];
}
function kbDraw() {
  const svg = $("#kbSvg"); if (!svg || UI.mode !== "keeb") return;
  const K = KBS(), G = kbGeom(), r = svg.getBoundingClientRect(); if (!r.width) return;
  if (!KV.zoom) kbFit(G);
  svg.setAttribute("viewBox", `${KV.vx} ${KV.vy} ${r.width * KV.zoom} ${r.height * KV.zoom}`);
  const { sx, sy } = G, o = [], px = KV.zoom;
  const path = pl => "M" + pl.map(p => p[0].toFixed(2) + "," + p[1].toFixed(2)).join("L") + "Z";
  // grid
  o.push(`<defs><pattern id="kbg" width="${sx}" height="${sy}" patternUnits="userSpaceOnUse"><path d="M${sx} 0H0V${sy}" fill="none" stroke="rgba(255,255,255,.035)" stroke-width="${px}"/></pattern></defs><rect x="${KV.vx}" y="${KV.vy}" width="${r.width * px}" height="${r.height * px}" fill="url(#kbg)"/>`);
  for (const b of G.boards) {
    if (!b) continue;
    for (const pl of b.polys) o.push(`<path d="${path(pl)}" class="kb-board"/>`);
    const A = b.area, csel = KV.csel === b.half;
    o.push(`<g data-ctrl="${b.half}" class="kb-ctrl ${csel ? "sel" : ""}" transform="rotate(${A.rot || 0} ${A.cx} ${A.cy})"><rect x="${A.x}" y="${A.y}" width="${A.w}" height="${A.h}" rx="1.5" class="kb-area"/>`);
    for (const p of b.plan.parts) { const fp = LIB_BY_ID[p.fp]; if (!fp) continue; const bb = compBox({ fp: p.fp, x: A.x + p.x, y: A.y + p.y, rot: p.rot, side: "top" }); o.push(`<rect x="${bb.x0}" y="${bb.y0}" width="${bb.x1 - bb.x0}" height="${bb.y1 - bb.y0}" rx="0.4" class="kb-part ${p.fixed ? "big" : ""}"/>`); }
    if (csel) o.push(`<rect x="${A.x - 1}" y="${A.y - 1}" width="${A.w + 2}" height="${A.h + 2}" rx="2" class="kb-selr" stroke-width="${2 * px}"/>`);
    o.push(`</g>`);
    o.push(`<text x="${A.cx}" y="${A.bb[1] - 1.2 - K.outline.margin}" class="kb-areat" font-size="${Math.max(2.4, 11 * px)}">${esc(G.mcu.name.split(" (")[0])}${G.mcu.bare ? " + circuitry" : ""}${A.rot ? ` · ${Math.round(A.rot)}°` : ""}</text>`);
    for (const [x, y] of b.holes) o.push(`<circle cx="${x}" cy="${y}" r="1.1" class="kb-hole"/>`);
    if (K.outline.mode === "custom" && K.outline.custom && K.outline.custom[b.half]) K.outline.custom[b.half].forEach(([x, y], i) => o.push(`<circle cx="${x}" cy="${y}" r="${Math.max(1, 6 * px)}" class="kb-vtx" data-vtx="${b.half}:${i}"/>`));
  }
  const prof = KB_PROFILES.find(p => p.id === K.profile);
  const fp = LIB_BY_ID[G.sw.fp];
  for (const p of G.places) {
    const k = p.k, sel = KV.sel.has(k.id), [c0, c1, tc] = kbCapColour(k, prof);
    const W = k.w * sx, H = k.h * sy, g = 0.6;
    let cap, top;
    if (k.kind === "knob") { const R = Math.min(W, H) / 2 - 1.2; cap = `<circle r="${R}" fill="#2b313c" stroke="#5c6678" stroke-width="0.6" class="kb-cap"/>`; top = `<circle r="${R * 0.72}" fill="#3b4352"/><line x1="0" y1="${-R * 0.72}" x2="0" y2="${-R * 0.25}" stroke="#ef8a3c" stroke-width="1" stroke-linecap="round"/>`; }
    else if (k.shape === "iso") { const ext = 0.25 * sx; cap = `<path d="M${-W / 2 - ext + g},${-H / 2 + g}H${W / 2 - g}V${H / 2 - g}H${-W / 2 + g}V0H${-W / 2 - ext + g}Z" fill="${c0}" class="kb-cap"/>`; top = `<path d="M${-W / 2 - ext + 2.4},${-H / 2 + 1.6}H${W / 2 - 2.4}V${H / 2 - 3.2}H${-W / 2 + 2.4}V${-1.6}H${-W / 2 - ext + 2.4}Z" fill="${c1}"/>`; }
    else { cap = `<rect x="${-W / 2 + g}" y="${-H / 2 + g}" width="${W - 2 * g}" height="${H - 2 * g}" rx="1.6" fill="${c0}" class="kb-cap"/>`; top = `<rect x="${-W / 2 + 2.4}" y="${-H / 2 + 1.6}" width="${W - 4.8}" height="${H - 4.8}" rx="1.4" fill="${c1}"/>`; }
    if (k.shape === "stepped") top += `<rect x="${-W / 2 + 2.4}" y="${-H / 2 + 1.6}" width="${(k.w - 1.25) * sx - 2}" height="${H - 4.8}" rx="1" fill="${c0}" opacity=".6"/>`;
    let lab = "";
    if (KV.view === "matrix") { const m = G.matrix.pos.get(k.id); lab = `<text class="kb-lab mono" fill="${tc}" font-size="3.2">${m ? (K.split.mode === "two" && p.half ? m[0] + G.matrix.rows : m[0]) + "," + m[1] : "?"}</text>`; }
    else if (KV.view === "legend" && k.kind === "knob") lab = "";
    else if (KV.view === "legend") lab = `<text class="kb-lab" fill="${tc}" font-size="${(k.label || "").length > 3 ? 2.6 : 3.4}">${esc(k.label || "")}</text>`;
    let fpart = "";
    if (KV.view === "fp" && k.kind === "knob") { const e = LIB_BY_ID.ec11; fpart = `<g class="kb-fp">${e.pads.map(q => `<circle cx="${q.x}" cy="${q.y}" r="${Math.max(q.w || q.d || 1.8, q.h || 0) / 2}" class="p"/>`).join("")}${kbPartSVG(K.diode, kbSpotsFor(K, G, p).dio, "d" + (K.dside === "top" ? " top" : ""))}</g>`; }
    else if (KV.view === "fp" && fp) {
      const fr = (K.fpRot || 0) + (k.fr || 0), sp = kbSpotsFor(K, G, p);
      fpart = `<g transform="rotate(${fr})" class="kb-fp">${(fp.holes || []).map(h => `<circle cx="${h.x}" cy="${h.y}" r="${h.d / 2}" class="h"/>`).join("")}${fp.pads.map(q => q.drill ? `<circle cx="${q.x}" cy="${q.y}" r="${q.w / 2}" class="p"/>` : `<rect x="${q.x - q.w / 2}" y="${q.y - q.h / 2}" width="${q.w}" height="${q.h}" class="p s"/>`).join("")}${kbPartSVG(K.diode, sp.dio, "d" + (K.dside === "top" ? " top" : ""))}${sp.led ? kbPartSVG("sk6812mini", sp.led, "l") : ""}${sp.cap ? kbPartSVG("c_0805", sp.cap, "c") : ""}</g>`;
    }
    o.push(`<g transform="translate(${p.cx.toFixed(3)},${p.cy.toFixed(3)}) rotate(${p.ang.toFixed(3)})" data-key="${k.id}" class="kb-key ${sel ? "sel" : ""}">${cap}${top}${fpart}${lab}${sel ? `<rect x="${-W / 2}" y="${-H / 2}" width="${W}" height="${H}" rx="2" class="kb-selr" stroke-width="${2 * px}"/>` : ""}</g>`);
  }
  if (K.split.mode !== "none") { const x = K.split.at * sx, ys = G.places.map(p => p.cy); o.push(`<line x1="${x}" x2="${x}" y1="${Math.min(...ys) - 30}" y2="${Math.max(...ys) + 30}" class="kb-split" stroke-width="${1.5 * px}"/>`); }
  if (KV.box) { const b = KV.box; o.push(`<rect x="${Math.min(b[0], b[2])}" y="${Math.min(b[1], b[3])}" width="${Math.abs(b[2] - b[0])}" height="${Math.abs(b[3] - b[1])}" class="kb-box" stroke-width="${px}"/>`); }
  svg.innerHTML = o.join("");
  const hint = $("#kbHint"); if (hint) hint.textContent = K.keys.length ? `${K.keys.length} keys · ${G.sw.name} · ${kbMcu(K).name}${KV.view === "fp" ? ` · diodes ${K.dside === "top" ? "on top" : "underneath"} (${(KB_DSTYLES.find(d => d.id === K.dstyle) || KB_DSTYLES[0]).name.toLowerCase()})` : ""}` : "";
  const em = $("#kbEmpty"); if (em) em.hidden = K.keys.length > 0;
}

// ---- pointer handling
function kbDown(e) {
  const svg = $("#kbSvg"); svg.focus(); svg.setPointerCapture(e.pointerId);
  const [x, y] = kbPt(e), K = KBS();
  if (e.button === 2 || e.button === 1) { KV.drag = { pan: true, sx: e.clientX, sy: e.clientY, vx: KV.vx, vy: KV.vy, moved: false }; return; }
  const v = e.target.closest("[data-vtx]");
  if (v) { const [h, i] = v.dataset.vtx.split(":").map(Number); KV.drag = { vtx: [h, i], moved: false }; return; }
  const g = e.target.closest("[data-key]");
  const cg = !g && e.target.closest("[data-ctrl]");
  if (cg) {
    const h = +cg.dataset.ctrl; KV.sel.clear(); KV.csel = h;
    const off = kbCtrlOff(K, h);
    KV.drag = { ctrl: true, h, x0: x, y0: y, ox: off.x || 0, oy: off.y || 0, moved: false };
    kbDraw(); kbPropsPanel(); return;
  }
  if (g || !e.shiftKey) KV.csel = null;
  if (g) {
    const id = g.dataset.key;
    if (e.shiftKey) { KV.sel.has(id) ? KV.sel.delete(id) : KV.sel.add(id); }
    else if (!KV.sel.has(id)) KV.sel = new Set([id]);
    const start = K.keys.filter(k => KV.sel.has(k.id)).map(k => [k, k.x, k.y]);
    KV.drag = { keys: start, x0: x, y0: y, moved: false };
    kbDraw(); kbPropsPanel(); return;
  }
  if (!e.shiftKey) KV.sel.clear();
  KV.drag = { box: true, x0: x, y0: y }; KV.box = [x, y, x, y]; kbDraw(); kbPropsPanel();
}
function kbMove(e) {
  const d = KV.drag; if (!d) return;
  if (d.pan) { KV.vx = d.vx - (e.clientX - d.sx) * KV.zoom; KV.vy = d.vy - (e.clientY - d.sy) * KV.zoom; d.moved = true; kbDraw(); return; }
  const [x, y] = kbPt(e), K = KBS(), G = KV.geom || kbGeom();
  if (d.vtx) { const [h, i] = d.vtx; K.outline.custom[h][i] = [+x.toFixed(2), +y.toFixed(2)]; d.moved = true; KV.geom = null; kbDraw(); return; }
  if (d.ctrl) {
    const snap = e.altKey ? 0.05 : 0.5, off = kbCtrlOff(K, d.h);
    const nx = Math.round((d.ox + x - d.x0) / snap) * snap, ny = Math.round((d.oy + y - d.y0) / snap) * snap;
    if (nx !== off.x || ny !== off.y) { off.x = +nx.toFixed(2); off.y = +ny.toFixed(2); d.moved = true; KV.geom = null; kbDraw(); }
    return;
  }
  if (d.keys) {
    const snap = e.altKey ? 0.01 : 0.25;
    let dx = (x - d.x0) / G.sx, dy = (y - d.y0) / G.sy; dx = Math.round(dx / snap) * snap; dy = Math.round(dy / snap) * snap;
    if (dx || dy || d.moved) { d.moved = true; for (const [k, kx, ky] of d.keys) { k.x = +(kx + dx).toFixed(4); k.y = +(ky + dy).toFixed(4); } KV.geom = null; kbDraw(); }
    return;
  }
  if (d.box) { KV.box = [d.x0, d.y0, x, y]; const [a, b, c, f] = [Math.min(d.x0, x), Math.min(d.y0, y), Math.max(d.x0, x), Math.max(d.y0, y)]; for (const p of G.places) if (p.cx > a && p.cx < c && p.cy > b && p.cy < f) KV.sel.add(p.k.id); kbDraw(); }
}
function kbUp() {
  const d = KV.drag; KV.drag = null; KV.box = null;
  if (d && (d.keys || d.vtx || d.ctrl) && d.moved) { kbChanged(d.ctrl ? "move controller" : d.vtx ? "outline corner" : "move keys"); return; }
  kbDraw(); kbPropsPanel();
}
function kbDbl(e) {
  const K = KBS(); if (K.outline.mode !== "custom" || !K.outline.custom) return;
  const [x, y] = kbPt(e); let best = null;
  K.outline.custom.forEach((pl, h) => pl && pl.forEach((a, i) => { const b = pl[(i + 1) % pl.length]; const dd = ptSegDist(x, y, a[0], a[1], b[0], b[1]); if (!best || dd < best.d) best = { d: dd, h, i }; }));
  if (best && best.d < 6) { K.outline.custom[best.h].splice(best.i + 1, 0, [+x.toFixed(2), +y.toFixed(2)]); kbChanged("add corner"); }
}
function kbCtx(e) {
  const v = e.target.closest("[data-vtx]"); if (!v) return; const K = KBS(); const [h, i] = v.dataset.vtx.split(":").map(Number);
  if (K.outline.custom[h].length > 3) { K.outline.custom[h].splice(i, 1); kbChanged("remove corner"); }
}
