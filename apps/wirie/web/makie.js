// Wirie → Makie case-data export (format "wirie-case" v1) — by nrmluvdev
// Describes the board and every part in 3D so Makie can generate enclosures,
// mounts and brackets automatically. Spec: MAKIE_FORMAT.md
"use strict";

const BOARD_THICKNESS = 1.6;
const USB_C_MODULES = /^(xiao|esp32c3sm|rp2040zero|usbc_bo)$/;

// physical info for one part: height, what kind of thing it is, openings & lid features (all in footprint-local mm, y-down)
function caseInfo(fp, c) {
  const id = fp.id, b = fp.body, cat = fp.cat;
  const info = { kind: "component", h: partHeight(fp), edge: null, lid: [] };
  const bodyW = b.w, bodyH = b.h;
  const raise = c && c.raise !== undefined ? c.raise : null;
  const mod = (h, kind = "module") => { info.kind = kind; info.h = h; info.raise = raise ?? 2.5; };
  if (b.module) {
    if (cat === "Microcontrollers") { mod(4.5); if (b.usb) info.edge = { type: USB_C_MODULES.test(id) ? "usb-c" : "usb-micro", local: [0, b.usb === "top" ? -1 : 1], w: 12, h: 7, zOff: 1.6 + 1.6, at: b.usb === "top" ? [b.x, b.y - b.h / 2] : [b.x, b.y + b.h / 2] }; }
    else if (/oled|tft|lcd|epaper|nokia|max7219|tm1637|ht16k33|matrix/.test(id)) { mod(4.5, "display"); info.lid.push({ type: "display-window", shape: "rect", x: b.x, y: b.y + (fp.pads[0].y < b.y ? 2 : -2), w: Math.max(4, bodyW - 4), h: Math.max(4, bodyH - 8), approx: true }); }
    else if (id === "hcsr04") { mod(15, "sensor"); info.lid.push({ type: "sensor-hole", shape: "circle", x: b.x - 13, y: b.y, d: 16.5 }, { type: "sensor-hole", shape: "circle", x: b.x + 13, y: b.y, d: 16.5 }); }
    else if (id === "pir") { mod(25, "sensor"); info.lid.push({ type: "sensor-hole", shape: "circle", x: b.x, y: b.y, d: 23.5 }); }
    else if (/relay/.test(id)) { mod(19, "module"); }
    else if (/^(bh1750|vl53l0x|ldrmod|ir_rx|apds|veml|max30102)/.test(id)) { mod(3.5, "sensor"); info.lid.push({ type: "sensor-window", shape: "circle", x: b.x, y: b.y, d: 5, approx: true }); }
    else if (/encoder|joystick/.test(id)) { mod(id === "joystick" ? 32 : 20, "control"); info.lid.push({ type: "knob", shape: "circle", x: b.x, y: b.y + (id === "joystick" ? 0 : 2), d: id === "joystick" ? 26 : 8 }); }
    else if (/stepstick/.test(id)) mod(14);
    else mod(4);
  } else if (/^hdr|^idc/.test(id)) { info.kind = "connector"; info.h = id.startsWith("idc") ? 9 : 8.5; }
  else if (/^jstxh/.test(id)) { info.kind = "connector"; info.h = 9.6; info.edge = { type: "cable", w: bodyW + 2, h: 8, auto: true }; }
  else if (/^jstph/.test(id)) { info.kind = "connector"; info.h = 6; info.edge = { type: "cable", w: bodyW + 2, h: 6, auto: true }; }
  else if (/^term/.test(id)) { info.kind = "connector"; info.h = 10; info.edge = { type: "wire-entry", w: bodyW + 1, h: 6, auto: true }; }
  else if (/usbc_bo|musb_bo/.test(id)) { info.kind = "connector"; info.h = 4.5; info.edge = { type: id === "usbc_bo" ? "usb-c" : "usb-micro", local: [0, -1], w: 12, h: 7, zOff: 2.5, at: [b.x, b.y - b.h / 2] }; }
  else if (/^sw6x6/.test(id)) { info.kind = "button"; info.h = 5; info.lid.push({ type: "button", shape: "circle", x: 0, y: 0, d: 4 }); }
  else if (/^sw12x12/.test(id)) { info.kind = "button"; info.h = 7.3; info.lid.push({ type: "button", shape: "circle", x: 0, y: 0, d: 8 }); }
  else if (/^sw_slide/.test(id)) { info.kind = "switch"; info.h = 5; info.edge = { type: "slide-switch", w: 10, h: 5, auto: true }; }
  else if (/^dip_sw/.test(id)) { info.kind = "switch"; info.h = 5; }
  else if (/^led(3|5)$/.test(id)) { const d = id === "led3" ? 3 : 5; info.kind = "led"; info.h = d === 3 ? 5.3 : 8.6; info.lid.push({ type: "led", shape: "circle", x: 0, y: 0, d: d + 0.3 }); }
  else if (/^led_rgb5/.test(id)) { info.kind = "led"; info.h = 8.6; info.lid.push({ type: "led", shape: "circle", x: b.x, y: b.y, d: 5.3 }); }
  else if (/^led_|ws2812/.test(id)) { info.kind = "led"; info.h = 1.2; info.lid.push({ type: "light-pipe", shape: "circle", x: 0, y: 0, d: 3 }); }
  else if (/^c_elec_/.test(id)) { const d = bodyW; info.h = d <= 6.3 ? 11.5 : d <= 8 ? 12 : d <= 10 ? 16.5 : 20.5; }
  else if (/^c_disc/.test(id)) info.h = 6;
  else if (/^r_axial|l_axial|d_do/.test(id)) info.h = 3;
  else if (id === "r_vert") info.h = 8;
  else if (/^to92|ds18b20/.test(id)) info.h = 5;
  else if (/^reg7805|reglm317|q_to220/.test(id)) info.h = 19;
  else if (/^dip|atmega|attiny|ne555|lm358|hc595|uln2003|l293d|pc817|cd4017|mcp23017/.test(id)) info.h = 4.5;
  else if (id === "relay_srd") info.h = 15.6;
  else if (id === "buzzer12") { info.h = 9.5; info.lid.push({ type: "vent", shape: "circle", x: 0, y: 0, d: 8 }); }
  else if (id === "xtal_hc49") info.h = 4;
  else if (/^pot16/.test(id)) { info.kind = "control"; info.h = 25; info.lid.push({ type: "knob", shape: "circle", x: b.x, y: b.y, d: 7.5 }); }
  else if (/^rv09/.test(id)) { info.kind = "control"; info.h = 18; info.lid.push({ type: "knob", shape: "circle", x: b.x, y: b.y, d: 7 }); }
  else if (/^trim3296/.test(id)) info.h = 10;
  else if (cat === "Mechanical") { info.kind = "mechanical"; info.h = 0; }
  else if (/^pi40/.test(id)) { info.kind = "connector"; info.h = 8.5; }
  // fall back to the broad opening table for everything else in the library
  if (!info.lid.length || !info.edge) for (const o of partOpenings(fp)) {
    if (o.face === "side" && !info.edge) { info.kind = info.kind === "component" ? (o.kind || "connector") : info.kind; info.edge = { type: o.kind || connectorType(id), w: Math.max(bodyW, bodyH) > 20 ? Math.min(bodyW, bodyH) + 2 : Math.max(bodyW, bodyH) + 2, h: o.kind === "card-edge" ? BOARD_THICKNESS + 0.6 : info.h + 1, auto: true, zOff: o.kind === "card-edge" ? -BOARD_THICKNESS / 2 : undefined }; }
    else if (o.face === "top" && !info.lid.length) info.lid.push(o.shape === "circle" ? { type: lidType(id, fp), shape: "circle", x: o.x, y: o.y, d: o.diameter } : { type: lidType(id, fp), shape: "rect", x: o.x, y: o.y, w: o.width, h: o.height, approx: fp.cat === "Displays & LEDs" });
  }
  if (info.kind === "component") info.kind = kindOf(id, fp);
  // generic data-driven overrides (used by the expanded library & user parts)
  if (fp.case) Object.assign(info, fp.case);
  if (c && c.caseH) info.h = c.caseH;
  return info;
}

function connectorType(id) {
  const m = [[/usbc/, "usb-c"], [/musb/, "usb-micro"], [/usba/, "usb-a"], [/dcjack/, "dc-jack"], [/jack35/, "audio-3.5mm"], [/rj45/, "rj45"], [/rca/, "rca"], [/minidin/, "mini-din"], [/de9|de15|db25/, "d-sub"], [/hdmi/, "hdmi"], [/sd_socket/, "sd-card"], [/xt30|xt60/, "power-xt"], [/atx/, "atx"]];
  for (const [re, t] of m) if (re.test(id)) return t; return "connector";
}
function lidType(id, fp) { if (/^sw|mx_|choc|arcade/.test(id)) return "button"; if (/^led|ws2812|sk6812/.test(id)) return /^led(3|5)|led_rgb5/.test(id) ? "led" : "light-pipe"; if (/pot|ec11|rv09|toggle/.test(id)) return "knob"; if (/slide/.test(id)) return "switch"; if (/buzzer|sound|mic|inmp/.test(id)) return "sound-holes"; if (fp.cat === "Displays & LEDs") return "display-window"; return "sensor-window"; }
function kindOf(id, fp) { const c = fp.cat; if (/Connectors|Ribbon|Headers/.test(c)) return "connector"; if (/Switches/.test(c)) return "button"; if (/Displays/.test(c)) return "display"; if (/Mechanical/.test(c)) return "mechanical"; if (/Microcontrollers/.test(c)) return "module"; return "component"; }

function makieExport(opts = {}) {
  const B = S.board, A = analyze();
  const Y = y => +(B.h - y).toFixed(3), R = v => +(+v).toFixed(3);
  const rotOut = r => (360 - r) % 360; // Wirie rotates clockwise on screen; export is CCW (Y-up)
  const xf = (c, x, y) => { const [wx, wy] = compXform(c, x, y); return [R(wx), Y(wy)]; };
  const dirXf = (c, v) => { const lx = c.side === "bottom" ? -v[0] : v[0]; const [rx, ry] = rotPt(lx, v[1], c.rot); return [R(rx), R(-ry)]; };
  const outline = boardPolygon(0).map(([x, y]) => [R(x), Y(y)]).reverse(); // CCW in Y-up
  const holes = [];
  for (const h of A.holes) holes.push({ x: R(h.x), y: Y(h.y), d: R(h.d), plated: false, ref: h.comp.ref });
  for (const p of A.pads) if (p.drill >= 2 && /^(MH|1)$/.test(p.name) && compFP(p.comp).cat === "Mechanical") holes.push({ x: R(p.x), y: Y(p.y), d: R(p.drill), plated: true, ref: p.comp.ref });
  const screw = d => (d <= 2.4 ? "M2" : d <= 2.9 ? "M2.5" : d <= 3.6 ? "M3" : d <= 4.6 ? "M4" : "M5");
  const parts = [], openings = [], lid = [];
  let maxTop = 0, maxBot = 0;
  // nearest board edge for a point / direction
  const faceOf = (x, y, dir) => {
    if (dir) return Math.abs(dir[0]) > Math.abs(dir[1]) ? (dir[0] > 0 ? "right" : "left") : (dir[1] > 0 ? "back" : "front");
    const d = { left: x, right: B.w - x, front: B.h - y, back: y };
    return Object.entries(d).sort((a, b) => a[1] - b[1])[0][0];
  };
  for (const c of S.comps) {
    const fp = compFP(c); if (!fp) continue;
    const ci = caseInfo(fp, c);
    const bb = compBox(c), br = bodyRect(c);
    const top = (c.side || "top") === "top";
    const raise = ci.raise || 0;
    const z0 = top ? BOARD_THICKNESS + raise : -raise, z1 = top ? z0 + ci.h : z0 - ci.h;
    if (top) maxTop = Math.max(maxTop, z1 - BOARD_THICKNESS); else maxBot = Math.max(maxBot, -z1);
    const [cx, cy] = xf(c, fp.body.x, fp.body.y);
    parts.push({
      ref: c.ref, value: c.value, name: fp.name, footprint: fp.id, category: fp.cat, kind: ci.kind, side: top ? "top" : "bottom",
      x: cx, y: cy, rotation: rotOut(c.rot),
      box: { x0: R(bb.x0), y0: Y(bb.y1), x1: R(bb.x1), y1: Y(bb.y0), z0: R(Math.min(z0, z1)), z1: R(Math.max(z0, z1)) },
      body: { x0: R(br.x0), y0: Y(br.y1), x1: R(br.x1), y1: Y(br.y0) },
      height: R(ci.h), raisedBy: R(raise), pins: fp.pads.length, approxHeight: !fp.case || !fp.case.h,
    });
    if (ci.edge) {
      const e = ci.edge;
      let at, dir;
      if (e.local) { at = xf(c, ...e.at); dir = dirXf(c, e.local); }
      else { at = [R((br.x0 + br.x1) / 2), Y((br.y0 + br.y1) / 2)]; }
      const face = faceOf(at[0], at[1], dir);
      const n = { left: [-1, 0], right: [1, 0], front: [0, -1], back: [0, 1] }[face];
      // project the opening onto that board edge
      const ex = face === "left" ? 0 : face === "right" ? B.w : at[0], ey = face === "front" ? 0 : face === "back" ? B.h : at[1];
      const zc = top ? BOARD_THICKNESS + raise + (e.zOff ?? ci.h / 2) : -(raise + (e.zOff ?? ci.h / 2));
      openings.push({ ref: c.ref, type: e.type, face, x: R(ex), y: R(ey), z: R(zc), width: R(e.w), height: R(e.h), partEdge: at, normal: n, distanceFromEdge: R((ex - at[0]) * n[0] + (ey - at[1]) * n[1]) });
    }
    for (const L of ci.lid) {
      const [lx, ly] = xf(c, L.x, L.y);
      const f = { ref: c.ref, type: L.type, shape: L.shape, x: lx, y: ly, side: top ? "top" : "bottom", partTop: R(Math.abs(z1)), approx: !!L.approx };
      if (L.shape === "circle") f.d = R(L.d); else { const sw = c.rot % 180 !== 0; f.w = R(sw ? L.h : L.w); f.h = R(sw ? L.w : L.h); }
      lid.push(f);
    }
  }
  const out = {
    format: "wirie-case", version: 1,
    generator: { app: APP.name, version: APP.version, author: APP.author, exported: new Date().toISOString() },
    project: S.name, units: "mm",
    coordinates: "Origin at the board's bottom-left corner. +X right, +Y toward the back (up on screen), +Z up. Board bottom face is z=0, top face is z=thickness. Rotations are degrees counter-clockwise looking down on the top.",
    board: { width: R(B.w), height: R(B.h), thickness: BOARD_THICKNESS, shape: B.round ? "round" : "rect", cornerRadius: R(B.round ? Math.min(B.w, B.h) / 2 : B.r || 0), outline },
    mountingHoles: holes.map(h => Object.assign(h, { screw: screw(h.d) })),
    parts, openings, lidFeatures: lid,
    envelope: { maxPartHeightTop: R(maxTop), maxPartHeightBottom: R(maxBot), suggestedStandoff: R(Math.max(3, maxBot + 2)), suggestedLidClearance: R(maxTop + 2), suggestedWall: 2.0, suggestedFit: 0.3 },
  };
  if (opts.textures !== false) {
    out.textures = { pixelsPerMm: 8, top: renderBoardPNG(false, 8), bottom: renderBoardPNG(true, 8) };
  }
  return out;
}

function renderBoardPNG(flip, ppm) {
  const B = S.board;
  const c = document.createElement("canvas"); c.width = Math.round(B.w * ppm); c.height = Math.round(B.h * ppm);
  const keep = { ...VIEW, show: { ...VIEW.show } }; const keepSel = UI.sel; UI.sel = new Set();
  Object.assign(VIEW, { zoom: ppm, ox: 0, oy: 0, mode: "real", flip }); VIEW.show = { ...VIEW.show, parts: false };
  const d = window.devicePixelRatio; Object.defineProperty(window, "devicePixelRatio", { value: 1, configurable: true });
  try { drawPCB(c); } finally { Object.defineProperty(window, "devicePixelRatio", { value: d, configurable: true }); Object.assign(VIEW, keep); UI.sel = keepSel; UI.dirty = true; }
  // keep only the board (transparent outside the outline)
  const ctx = c.getContext("2d"); ctx.setTransform(ppm, 0, 0, ppm, 0, 0); ctx.globalCompositeOperation = "destination-in";
  ctx.beginPath(); for (const bp of boardPolys(0)) { (flip ? bp.map(([x, y]) => [B.w - x, y]) : bp).forEach((q, i) => (i ? ctx.lineTo(q[0], q[1]) : ctx.moveTo(q[0], q[1]))); ctx.closePath(); } ctx.fill();
  return c.toDataURL("image/png");
}
