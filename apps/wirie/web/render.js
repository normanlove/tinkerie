// Wirie PCB renderer — design view + realistic board preview — by nrmluvdev
"use strict";

const VIEW = { zoom: 8, ox: 200, oy: 100, mode: "design", flip: false, show: { top: true, bottom: true, silk: true, rats: true, parts: true, labels: true } };
const COL = {
  bg: "#0c1016", dot: "#1c2533", board: "#17271f", edge: "#f0c36d",
  top: "#ef6a43", bottom: "#4c8dff", pad: "#e2b04f", hole: "#0c1016", via: "#cfb36a",
  silk: "#f4f1ea", silkB: "#a9b8ff", rats: "#ffd166", sel: "#3fe0d0", hover: "#ffffff",
};
function w2s(x, y) { const fx = VIEW.flip ? S.board.w - x : x; return [fx * VIEW.zoom + VIEW.ox, y * VIEW.zoom + VIEW.oy]; }
function s2w(x, y) { let wx = (x - VIEW.ox) / VIEW.zoom; if (VIEW.flip) wx = S.board.w - wx; return [wx, (y - VIEW.oy) / VIEW.zoom]; }

function pathPoly(ctx, pts) { ctx.beginPath(); pts.forEach((p, i) => (i ? ctx.lineTo(p[0], p[1]) : ctx.moveTo(p[0], p[1]))); ctx.closePath(); }
function rrect(ctx, x, y, w, h, r) { r = Math.min(r, w / 2, h / 2); ctx.beginPath(); ctx.moveTo(x + r, y); ctx.arcTo(x + w, y, x + w, y + h, r); ctx.arcTo(x + w, y + h, x, y + h, r); ctx.arcTo(x, y + h, x, y, r); ctx.arcTo(x, y, x + w, y, r); ctx.closePath(); }
function padPath(ctx, p, grow = 0) {
  const w = p.w + 2 * grow, h = p.h + 2 * grow;
  if (p.ang) { ctx.save(); ctx.translate(p.x, p.y); ctx.rotate(p.ang); if (p.shape === "rect") { ctx.beginPath(); ctx.rect(-w / 2, -h / 2, w, h); } else rrect(ctx, -w / 2, -h / 2, w, h, Math.min(w, h) / 2); ctx.restore(); return; }
  if (p.shape === "rect") { ctx.beginPath(); ctx.rect(p.x - w / 2, p.y - h / 2, w, h); }
  else rrect(ctx, p.x - w / 2, p.y - h / 2, w, h, Math.min(w, h) / 2);
}
function drillPath(ctx, p) { if (p.slot) { const [w, h] = p.slot; rrect(ctx, p.x - w / 2, p.y - h / 2, w, h, Math.min(w, h) / 2); } else { ctx.beginPath(); ctx.arc(p.x, p.y, p.drill / 2, 0, 7); } }
function itemPath(ctx, it, grow = 0) {
  if (it.kind === "pad") { padPath(ctx, it.ref, grow); ctx.fill(); }
  else if (it.kind === "via") { ctx.beginPath(); ctx.arc(it.ref.x, it.ref.y, it.ref.d / 2 + grow, 0, 7); ctx.fill(); }
  else { const s = it.shape; ctx.lineWidth = 2 * (s.r + grow); ctx.beginPath(); ctx.moveTo(s.ax, s.ay); ctx.lineTo(s.bx, s.by); ctx.stroke(); }
}

let pourCanvas = null;
function drawPour(ctx, layer, color, alpha, W, H, dpr) {
  const A = analyze(); const pn = pourActive(layer); if (!pn) return;
  if (!pourCanvas) pourCanvas = document.createElement("canvas");
  if (pourCanvas.width !== W || pourCanvas.height !== H) { pourCanvas.width = W; pourCanvas.height = H; }
  const p = pourCanvas.getContext("2d");
  p.setTransform(1, 0, 0, 1, 0, 0); p.clearRect(0, 0, W, H);
  setWorld(p, dpr);
  p.fillStyle = color; pathBoard(p, S.board.edgeClear); p.fill();
  p.globalCompositeOperation = "destination-out"; p.fillStyle = "#000"; p.strokeStyle = "#000"; p.lineCap = "round";
  for (const it of A.items) if (it.layers.includes(layer) && (it.cnet !== pn || it.kind === "pad")) itemPath(p, it, S.board.pourClear);
  p.globalCompositeOperation = "source-over"; p.strokeStyle = color; p.lineWidth = SPOKE_W; p.lineCap = "butt";
  for (const it of A.items) if (it.layers.includes(layer) && it.kind === "pad" && it.cnet === pn) for (const s of thermalSpokes(it.ref)) { p.beginPath(); p.moveTo(s[0], s[1]); p.lineTo(s[2], s[3]); p.stroke(); }
  p.globalCompositeOperation = "destination-out"; p.lineCap = "round";
  for (const it of A.items) if (it.layers.includes(layer) && it.cnet !== pn) itemPath(p, it, S.board.pourClear);
  for (const h of A.holes) { p.beginPath(); p.arc(h.x, h.y, h.d / 2 + S.board.pourClear, 0, 7); p.fill(); }
  p.globalCompositeOperation = "source-over";
  ctx.save(); ctx.setTransform(1, 0, 0, 1, 0, 0); ctx.globalAlpha = alpha; ctx.drawImage(pourCanvas, 0, 0); ctx.restore();
}
function setWorld(ctx, dpr) {
  const z = VIEW.zoom * dpr;
  if (VIEW.flip) ctx.setTransform(-z, 0, 0, z, (VIEW.ox + S.board.w * VIEW.zoom) * dpr, VIEW.oy * dpr);
  else ctx.setTransform(z, 0, 0, z, VIEW.ox * dpr, VIEW.oy * dpr);
}

function drawPCB(cv) {
  const ctx = cv.getContext("2d"), dpr = window.devicePixelRatio || 1;
  const W = cv.width, H = cv.height;
  const A = analyze(), B = S.board, real = VIEW.mode === "real";
  ctx.setTransform(1, 0, 0, 1, 0, 0);
  // background
  const bgG = ctx.createRadialGradient(W * 0.5, H * 0.45, 0, W * 0.5, H * 0.5, Math.max(W, H) * 0.75);
  bgG.addColorStop(0, real ? "#1a1f29" : "#111722"); bgG.addColorStop(1, real ? "#0b0d12" : "#0a0d13");
  ctx.fillStyle = bgG; ctx.fillRect(0, 0, W, H);
  setWorld(ctx, dpr);
  const px = 1 / VIEW.zoom; // one screen pixel in mm
  // grid dots
  if (!real) {
    let g = B.grid; while (g * VIEW.zoom < 9 && g < 1000) g *= 2;
    const [ax, ay] = s2w(0, 0), [bx, by] = s2w(W / dpr, H / dpr);
    ctx.fillStyle = COL.dot;
    const r = 1.1 * px;
    for (let x = Math.floor(Math.min(ax, bx) / g) * g; x <= Math.max(ax, bx); x += g) for (let y = Math.floor(ay / g) * g; y <= by; y += g) ctx.fillRect(x - r / 2, y - r / 2, r, r);
  }
  // board
    ctx.save();
  ctx.shadowColor = "rgba(0,0,0,0.55)"; ctx.shadowBlur = 30 * dpr; ctx.shadowOffsetY = 8 * dpr;
  ctx.fillStyle = real ? "#1d6b3c" : COL.board; pathBoard(ctx, 0); ctx.fill();
  ctx.restore();
  if (real) { const gr = ctx.createLinearGradient(0, 0, B.w, B.h); gr.addColorStop(0, "rgba(255,255,255,0.07)"); gr.addColorStop(1, "rgba(0,0,0,0.12)"); ctx.fillStyle = gr; pathBoard(ctx, 0); ctx.fill(); }
  ctx.lineCap = "round"; ctx.lineJoin = "round";
  const viewTop = !VIEW.flip;
  const layersOrder = viewTop ? ["bottom", "top"] : ["top", "bottom"];
  const active = UI.layer;
  if (!real) {
    for (const L of (active === "top" ? ["bottom", "top"] : ["top", "bottom"])) {
      if (!VIEW.show[L]) continue;
      const col = L === "top" ? COL.top : COL.bottom, dim = L !== active;
      drawPour(ctx, L, col, dim ? 0.24 : 0.34, W, H, dpr);
      setWorld(ctx, dpr);
      ctx.globalAlpha = dim ? 0.42 : 0.95;
      ctx.strokeStyle = col; ctx.fillStyle = col;
      for (const it of A.items) if (it.kind === "seg" && it.layers[0] === L) { ctx.strokeStyle = hl(it) ? "#fff" : col; itemPath(ctx, it); }
      for (const it of A.items) if (it.kind === "pad" && !it.ref.drill && it.layers[0] === L) { ctx.fillStyle = hl(it) ? "#fff" : col; itemPath(ctx, it); }
      ctx.globalAlpha = 1;
    }
  } else {
    const L = viewTop ? "top" : "bottom";
    drawPour(ctx, L, "#2e8a52", 1, W, H, dpr); setWorld(ctx, dpr);
    ctx.strokeStyle = "#2b8650"; ctx.fillStyle = "#2b8650";
    for (const it of A.items) if (it.kind === "seg" && it.layers[0] === L) itemPath(ctx, it);
    // faint tracks of other side show through
    ctx.globalAlpha = 0.18; ctx.strokeStyle = "#0e3d22";
    for (const it of A.items) if (it.kind === "seg" && it.layers[0] !== L) itemPath(ctx, it);
    ctx.globalAlpha = 1;
  }
  // THT pads & vias
  const vis = real ? (viewTop ? "top" : "bottom") : null;
  for (const it of A.items) {
    if (it.kind === "pad") {
      const p = it.ref;
      if (!p.drill && real && it.layers[0] !== vis) continue;
      if (!p.drill && !real) continue;
      if (p.drill && !real && !VIEW.show.top && !VIEW.show.bottom) continue;
      const g = ctx.createLinearGradient(p.x - p.w / 2, p.y - p.h / 2, p.x + p.w / 2, p.y + p.h / 2);
      g.addColorStop(0, hl(it) ? "#fff7d6" : "#f6d27a"); g.addColorStop(1, hl(it) ? "#ffe9a8" : (real ? "#b98a2e" : "#c08f35"));
      ctx.fillStyle = g; padPath(ctx, p); ctx.fill();
      if (p.drill) { ctx.fillStyle = real ? "#0b0b0b" : COL.hole; drillPath(ctx, p); ctx.fill(); }
    } else if (it.kind === "via") {
      const v = it.ref;
      ctx.fillStyle = real ? "#2f8a52" : (hl(it) ? "#fff" : COL.via); ctx.beginPath(); ctx.arc(v.x, v.y, v.d / 2, 0, 7); ctx.fill();
      ctx.fillStyle = real ? "#0b0b0b" : COL.hole; ctx.beginPath(); ctx.arc(v.x, v.y, v.drill / 2, 0, 7); ctx.fill();
    }
  }
  // mounting holes
  for (const h of A.holes) { ctx.fillStyle = "#06080b"; ctx.beginPath(); ctx.arc(h.x, h.y, h.d / 2, 0, 7); ctx.fill(); ctx.strokeStyle = "rgba(255,255,255,0.15)"; ctx.lineWidth = px; ctx.stroke(); }
  // silkscreen
  if (VIEW.show.silk) {
    const sl = real ? [viewTop ? "top" : "bottom"] : (active === "top" ? ["bottom", "top"] : ["top", "bottom"]);
    for (const L of sl) {
      ctx.strokeStyle = real ? "#f7f7f2" : (L === "top" ? COL.silk : COL.silkB);
      ctx.globalAlpha = real ? 0.95 : (L === active ? 0.85 : 0.35);
      for (const l of silkLines(L)) { ctx.lineWidth = l.w; ctx.beginPath(); l.pts.forEach((p, i) => (i ? ctx.lineTo(p[0], p[1]) : ctx.moveTo(p[0], p[1]))); ctx.stroke(); }
    }
    ctx.globalAlpha = 1;
  }
  // board edge
  ctx.strokeStyle = real ? "rgba(0,0,0,0.5)" : COL.edge; ctx.lineWidth = real ? 0.25 : 1.5 * px; pathBoard(ctx, 0); ctx.stroke();
  // realistic parts
  if (real && VIEW.show.parts) for (const c of S.comps) if ((c.side || "top") === (viewTop ? "top" : "bottom")) drawPartBody(ctx, c, px);
  // design-mode bodies (very faint) + selection
  if (!real) {
    for (const c of S.comps) {
      const fp = compFP(c); if (!fp) continue;
      const sel = UI.sel.has(c.id);
      if (fp.body.module || sel || UI.hoverComp === c.id) {
        const r = bodyRect(c);
        ctx.fillStyle = sel ? "rgba(63,224,208,0.10)" : fp.body.module ? "rgba(120,170,255,0.06)" : "rgba(255,255,255,0.04)";
        rrect(ctx, r.x0, r.y0, r.x1 - r.x0, r.y1 - r.y0, 0.8); ctx.fill();
        if (fp.body.module && VIEW.show.labels && VIEW.zoom > 3) {
          ctx.save(); ctx.translate((r.x0 + r.x1) / 2, (r.y0 + r.y1) / 2); if (VIEW.flip) ctx.scale(-1, 1);
          const vert = (r.y1 - r.y0) > (r.x1 - r.x0) * 1.3; if (vert) ctx.rotate(-Math.PI / 2);
          ctx.fillStyle = "rgba(170,200,255,0.17)"; ctx.font = `600 ${Math.min(2.4, (vert ? r.y1 - r.y0 : r.x1 - r.x0) / 10)}px Segoe UI, system-ui, sans-serif`; ctx.textAlign = "center"; ctx.textBaseline = "middle";
          ctx.fillText(fp.value || fp.name, 0, 0); ctx.restore();
        }
      }
      if (sel) { const b = compBox(c); ctx.strokeStyle = COL.sel; ctx.lineWidth = 1.5 * px; ctx.setLineDash([4 * px, 3 * px]); ctx.strokeRect(b.x0 - 0.4, b.y0 - 0.4, b.x1 - b.x0 + 0.8, b.y1 - b.y0 + 0.8); ctx.setLineDash([]); }
    }
    const rh = typeof rotHandle === "function" && rotHandle();
    if (rh) { // drag this round handle to turn the part by hand
      ctx.strokeStyle = COL.sel; ctx.fillStyle = "#0e1a20"; ctx.lineWidth = 1.5 * px;
      ctx.beginPath(); ctx.moveTo(rh.x, rh.top); ctx.lineTo(rh.x, rh.y + 6 * px); ctx.stroke();
      ctx.beginPath(); ctx.arc(rh.x, rh.y, 6 * px, 0, 7); ctx.fill(); ctx.stroke();
      ctx.beginPath(); ctx.arc(rh.x, rh.y, 3 * px, -2.6, 1.6); ctx.stroke();
      if (UI.rotShow) { ctx.save(); ctx.translate(UI.rotShow.x, UI.rotShow.y); if (VIEW.flip) ctx.scale(-1, 1); ctx.fillStyle = COL.sel; ctx.font = `600 ${12 * px}px Segoe UI, system-ui, sans-serif`; ctx.fillText(UI.rotShow.a + "°", 12 * px, -10 * px); ctx.restore(); }
    }
    for (const t of S.tracks) if (UI.sel.has(t.id)) { ctx.strokeStyle = COL.sel; ctx.globalAlpha = 0.5; ctx.lineWidth = t.w + 6 * px; ctx.beginPath(); t.pts.forEach((p, i) => (i ? ctx.lineTo(p[0], p[1]) : ctx.moveTo(p[0], p[1]))); ctx.stroke(); ctx.globalAlpha = 1; }
    for (const v of S.vias) if (UI.sel.has(v.id)) { ctx.strokeStyle = COL.sel; ctx.lineWidth = 2 * px; ctx.beginPath(); ctx.arc(v.x, v.y, v.d / 2 + 0.3, 0, 7); ctx.stroke(); }
    for (const t of S.texts) if (UI.sel.has(t.id)) { const w = textStrokes(t.text, t.size).width; ctx.strokeStyle = COL.sel; ctx.lineWidth = 1.5 * px; ctx.setLineDash([4 * px, 3 * px]); ctx.save(); ctx.translate(t.x, t.y); ctx.rotate((t.rot || 0) * Math.PI / 180); ctx.strokeRect(-w / 2 - 0.5, -t.size / 2 - 0.5, w + 1, t.size + 1); ctx.restore(); ctx.setLineDash([]); }
    // pin labels when zoomed in
    if (VIEW.show.labels && VIEW.zoom > 7) {
      ctx.textAlign = "center"; ctx.textBaseline = "middle";
      for (const p of A.pads) {
        const fp = compFP(p.comp); if (!fp.pinLabels && VIEW.zoom < 16) continue;
        const fs = Math.min(p.w, p.h) * (p.name.length > 5 ? 0.27 : p.name.length > 3 ? 0.33 : p.name.length > 2 ? 0.4 : 0.5);
        if (fs * VIEW.zoom < 5) continue;
        ctx.save(); ctx.translate(p.x, p.y); if (VIEW.flip) ctx.scale(-1, 1);
        ctx.font = `700 ${fs}px Segoe UI, system-ui, sans-serif`; ctx.lineJoin = "round"; ctx.lineWidth = fs * 0.32; ctx.strokeStyle = "rgba(10,8,2,0.9)"; ctx.strokeText(p.name, 0, 0.03); ctx.fillStyle = "#fff8e6"; ctx.fillText(p.name, 0, 0.03); ctx.restore();
      }
    }
    // ratsnest
    if (VIEW.show.rats) {
      ctx.lineWidth = 1.3 * px; ctx.setLineDash([5 * px, 4 * px]);
      for (const r of A.rats) { ctx.strokeStyle = r.net.color; ctx.globalAlpha = UI.hoverNet && UI.hoverNet !== r.net ? 0.25 : 0.95; ctx.beginPath(); ctx.moveTo(r.a.x, r.a.y); ctx.lineTo(r.b.x, r.b.y); ctx.stroke(); }
      ctx.setLineDash([]); ctx.globalAlpha = 1;
    }
    // DRC markers
    if (UI.drcShow) for (const d of UI.drc) if (d.sev === "err") { ctx.strokeStyle = "#ff4d6d"; ctx.lineWidth = 2 * px; ctx.beginPath(); ctx.arc(d.x, d.y, 10 * px, 0, 7); ctx.stroke(); }
    if (UI.focus) { ctx.strokeStyle = "#ff4d6d"; ctx.lineWidth = 2.5 * px; ctx.beginPath(); ctx.arc(UI.focus.x, UI.focus.y, (14 + 4 * Math.sin(performance.now() / 120)) * px, 0, 7); ctx.stroke(); }
  }
  drawOverlay(ctx, px);
}
function hl(it) { const n = UI.hoverNet; return n && it.cnet === n; }

// ---------- realistic part bodies ----------
const BAND = ["#111", "#7a4a1a", "#d22", "#f07a10", "#f3d21b", "#1f9d3a", "#1d55d6", "#8a3ad6", "#888", "#f5f5f5"];
function resistorBands(value) {
  const ohms = parseEng(value); if (!(ohms > 0)) return null;
  let e = Math.floor(Math.log10(ohms)) - 1; let sig = Math.round(ohms / Math.pow(10, e)); if (sig >= 100) { sig = Math.round(sig / 10); e++; }
  if (e < -2 || e > 9) return null;
  return [BAND[Math.floor(sig / 10)], BAND[sig % 10], e === -1 ? "#c9a227" : e === -2 ? "#c0c0c0" : BAND[e], "#c9a227"];
}
function drawPartBody(ctx, c, px) {
  const fp = compFP(c); if (!fp) return; const b = fp.body;
  ctx.save(); ctx.translate(c.x, c.y); ctx.rotate(c.rot * Math.PI / 180); if (c.side === "bottom") ctx.scale(-1, 1);
  const x0 = b.x - b.w / 2, y0 = b.y - b.h / 2;
  ctx.shadowColor = "rgba(0,0,0,0.45)"; ctx.shadowBlur = 6 / px / 12; ctx.shadowOffsetY = 0.5;
  const id = fp.id;
  if (b.module) {
    rrect(ctx, x0, y0, b.w, b.h, 1); ctx.fillStyle = b.col; ctx.fill(); ctx.shadowColor = "transparent";
    ctx.strokeStyle = "rgba(0,0,0,0.55)"; ctx.lineWidth = 0.35; ctx.stroke(); rrect(ctx, x0 + 0.35, y0 + 0.35, b.w - 0.7, b.h - 0.7, 0.8); ctx.strokeStyle = "rgba(255,255,255,0.10)"; ctx.lineWidth = 0.2; ctx.stroke();
    const g = ctx.createLinearGradient(x0, y0, x0 + b.w, y0 + b.h); g.addColorStop(0, "rgba(255,255,255,0.12)"); g.addColorStop(1, "rgba(0,0,0,0.25)"); ctx.fillStyle = g; rrect(ctx, x0, y0, b.w, b.h, 1); ctx.fill();
    for (const p of fp.pads) { ctx.fillStyle = "#d9b25a"; ctx.beginPath(); ctx.arc(p.x, p.y, 0.8, 0, 7); ctx.fill(); ctx.fillStyle = "#2a2a2a"; ctx.beginPath(); ctx.arc(p.x, p.y, 0.45, 0, 7); ctx.fill(); }
    if (b.usb) { const uy = b.usb === "top" ? y0 - 0.8 : y0 + b.h - 4.2; ctx.fillStyle = "#c9ccd1"; rrect(ctx, b.x - 4, uy, 8, 5, 0.6); ctx.fill(); ctx.fillStyle = "#6b7078"; rrect(ctx, b.x - 3, uy + 1.2, 6, 2.5, 0.4); ctx.fill(); }
    if (/pico|esp32|nodemcu|d1mini|xiao|c3|zero|teensy|bluepill|nano|pro/.test(id)) { const cw = Math.min(b.w, b.h) * 0.42; ctx.fillStyle = "#141414"; rrect(ctx, b.x - cw / 2, b.y - cw / 2, cw, cw, 0.4); ctx.fill(); ctx.fillStyle = "rgba(255,255,255,0.18)"; ctx.font = `600 ${cw / 5}px Segoe UI, sans-serif`; ctx.textAlign = "center"; ctx.textBaseline = "middle"; ctx.fillText(fp.value.split(" ")[0].slice(0, 8), b.x, b.y); }
    if (/esp32dev|nodemcu|d1mini|esp01|picow|pico2w/.test(id)) { const sy = b.usb === "bottom" || id === "esp01" ? y0 + 1 : y0 + b.h - 9; ctx.fillStyle = "#c8ccd2"; rrect(ctx, b.x - b.w * 0.36, sy, b.w * 0.72, 8, 0.4); ctx.fill(); ctx.fillStyle = "rgba(0,0,0,0.25)"; ctx.font = `600 1.4px Segoe UI, sans-serif`; ctx.textAlign = "center"; ctx.fillText("WiFi", b.x, sy + 4); }
    ctx.fillStyle = "rgba(255,255,255,0.85)"; ctx.font = `700 ${Math.min(2.2, b.w / 9, b.h / 3.2)}px Segoe UI, sans-serif`; ctx.textAlign = "center"; ctx.textBaseline = "middle";
    const label = fp.value.length > 22 ? fp.value.slice(0, 22) : fp.value;
    ctx.save(); const vert = b.h > b.w * 1.3; ctx.translate(b.x, b.y + (vert ? b.h * 0.3 : b.h * 0.28)); if (vert) ctx.rotate(-Math.PI / 2); ctx.fillText(label, vert ? 0 : 0, 0); ctx.restore();
  } else if (id.startsWith("r_axial") || id === "l_axial") {
    const p1 = fp.pads[0], p2 = fp.pads[1];
    ctx.strokeStyle = "#c9c9c9"; ctx.lineWidth = 0.5; ctx.beginPath(); ctx.moveTo(p1.x, 0); ctx.lineTo(p2.x, 0); ctx.stroke();
    rrect(ctx, x0, y0, b.w, b.h, b.h / 2); ctx.fillStyle = b.col; ctx.fill(); ctx.shadowColor = "transparent";
    const bands = id === "l_axial" ? null : resistorBands(c.value);
    if (bands) bands.forEach((col, i) => { ctx.fillStyle = col; ctx.fillRect(x0 + b.w * (0.18 + i * 0.16) + (i === 3 ? b.w * 0.1 : 0), y0, b.w * 0.08, b.h); });
    const g = ctx.createLinearGradient(0, y0, 0, y0 + b.h); g.addColorStop(0, "rgba(255,255,255,0.35)"); g.addColorStop(0.5, "rgba(255,255,255,0)"); g.addColorStop(1, "rgba(0,0,0,0.25)"); ctx.fillStyle = g; rrect(ctx, x0, y0, b.w, b.h, b.h / 2); ctx.fill();
  } else if (b.round) {
    ctx.beginPath(); ctx.arc(b.x, b.y, b.w / 2, 0, 7);
    const isLed = id.startsWith("led");
    const ledCol = /blue/i.test(c.value) ? "#3b82f6" : /green/i.test(c.value) ? "#22c55e" : /yellow/i.test(c.value) ? "#facc15" : /white/i.test(c.value) ? "#f8fafc" : /rgb/i.test(c.value) ? "#e2e8f0" : "#ef4444";
    const g = ctx.createRadialGradient(b.x - b.w * 0.15, b.y - b.w * 0.15, 0, b.x, b.y, b.w / 2);
    g.addColorStop(0, isLed ? "#fff" : "#4a6fd6"); g.addColorStop(0.35, isLed ? ledCol : b.col); g.addColorStop(1, isLed ? shade(ledCol) : "#0d1f4a");
    ctx.fillStyle = g; ctx.fill(); ctx.shadowColor = "transparent";
    if (id.startsWith("c_elec")) { ctx.fillStyle = "rgba(220,220,220,0.8)"; ctx.beginPath(); ctx.moveTo(b.x, b.y); ctx.arc(b.x, b.y, b.w / 2, Math.PI * 0.6, Math.PI * 1.0); ctx.fill(); ctx.strokeStyle = "rgba(255,255,255,0.4)"; ctx.lineWidth = 0.15; ctx.beginPath(); ctx.moveTo(b.x - b.w * 0.2, b.y); ctx.lineTo(b.x + b.w * 0.2, b.y); ctx.moveTo(b.x, b.y - b.w * 0.2); ctx.lineTo(b.x, b.y + b.w * 0.2); ctx.stroke(); }
  } else {
    const isHdr = /^(hdr|idc|jst|term)/.test(id);
    rrect(ctx, x0, y0, b.w, b.h, isHdr ? 0.3 : 0.4); ctx.fillStyle = b.col === "#0000" ? "rgba(0,0,0,0)" : b.col; ctx.fill(); ctx.shadowColor = "transparent";
    const g = ctx.createLinearGradient(x0, y0, x0, y0 + b.h); g.addColorStop(0, "rgba(255,255,255,0.14)"); g.addColorStop(1, "rgba(0,0,0,0.25)"); ctx.fillStyle = g; rrect(ctx, x0, y0, b.w, b.h, 0.4); ctx.fill();
    if (isHdr) for (const p of fp.pads) { ctx.fillStyle = "#d8b65c"; ctx.fillRect(p.x - 0.32, p.y - 0.32, 0.64, 0.64); }
    if (b.notch) { ctx.fillStyle = "rgba(255,255,255,0.12)"; ctx.beginPath(); ctx.arc(b.x, y0, Math.min(b.w, b.h) * 0.12, 0, Math.PI); ctx.fill(); ctx.fillStyle = "rgba(255,255,255,0.75)"; ctx.font = `600 ${Math.min(1.6, b.w / 5)}px Segoe UI, sans-serif`; ctx.textAlign = "center"; ctx.textBaseline = "middle"; ctx.save(); ctx.translate(b.x, b.y); if (b.h > b.w) ctx.rotate(Math.PI / 2); ctx.fillText(c.value.slice(0, 12), 0, 0); ctx.restore(); }
    if (b.btn) { ctx.fillStyle = "#151515"; ctx.beginPath(); ctx.arc(b.x, b.y, b.w * 0.3, 0, 7); ctx.fill(); ctx.strokeStyle = "rgba(255,255,255,0.15)"; ctx.lineWidth = 0.15; ctx.stroke(); }
    if (b.dshape) { ctx.fillStyle = "rgba(255,255,255,0.7)"; ctx.font = `600 0.9px Segoe UI, sans-serif`; ctx.textAlign = "center"; ctx.textBaseline = "middle"; ctx.fillText(c.value.slice(0, 7), b.x, b.y); }
    if (b.stripe) { ctx.fillStyle = "#ddd"; ctx.fillRect(x0 + b.w * 0.12, y0, b.w * 0.1, b.h); }
    if (id.startsWith("term")) for (const p of fp.pads) { ctx.fillStyle = "#c0c4ca"; ctx.beginPath(); ctx.arc(p.x, p.y, 1.3, 0, 7); ctx.fill(); ctx.strokeStyle = "#555"; ctx.lineWidth = 0.3; ctx.beginPath(); ctx.moveTo(p.x - 0.8, p.y); ctx.lineTo(p.x + 0.8, p.y); ctx.stroke(); }
    if (id === "ws2812b") { ctx.fillStyle = "#fafafa"; ctx.beginPath(); ctx.arc(0, 0, 1.8, 0, 7); ctx.fill(); ctx.fillStyle = "#f0c36d"; ctx.fillRect(-0.4, -0.4, 0.8, 0.8); }
  }
  ctx.restore();
}
function shade(hex) { const n = parseInt(hex.slice(1), 16); const r = (n >> 16) * 0.45, g = ((n >> 8) & 255) * 0.45, b = (n & 255) * 0.45; return `rgb(${r | 0},${g | 0},${b | 0})`; }

// small footprint preview for library cards
function drawFPPreview(cv, fp) {
  const ctx = cv.getContext("2d"), W = cv.width, H = cv.height;
  ctx.clearRect(0, 0, W, H);
  let x0 = 1e9, y0 = 1e9, x1 = -1e9, y1 = -1e9;
  const b = fp.body; x0 = b.x - b.w / 2; x1 = b.x + b.w / 2; y0 = b.y - b.h / 2; y1 = b.y + b.h / 2;
  for (const p of fp.pads) { x0 = Math.min(x0, p.x - p.w / 2); x1 = Math.max(x1, p.x + p.w / 2); y0 = Math.min(y0, p.y - p.h / 2); y1 = Math.max(y1, p.y + p.h / 2); }
  for (const h of fp.holes) { x0 = Math.min(x0, h.x - h.d); x1 = Math.max(x1, h.x + h.d); y0 = Math.min(y0, h.y - h.d); y1 = Math.max(y1, h.y + h.d); }
  const sc = Math.min((W - 8) / (x1 - x0 || 1), (H - 8) / (y1 - y0 || 1), 40);
  ctx.setTransform(sc, 0, 0, sc, W / 2 - (x0 + x1) / 2 * sc, H / 2 - (y0 + y1) / 2 * sc);
  const fake = { x: 0, y: 0, rot: 0, side: "top", fp: fp.id, value: fp.value };
  const px = 1 / sc;
  drawPartBody(ctx, fake, px * 12);
  if (!b.module) for (const p of fp.pads) { ctx.fillStyle = p.drill ? "#e2b04f" : p.opp ? "#4c8dff" : "#ef6a43"; ctx.globalAlpha = b.module ? 0 : 0.95; padPath(ctx, p); ctx.fill(); if (p.drill) { ctx.fillStyle = "#11161f"; drillPath(ctx, p); ctx.fill(); } }
  ctx.globalAlpha = 1;
  for (const h of fp.holes) { ctx.strokeStyle = "#9aa"; ctx.lineWidth = 2 * px; ctx.beginPath(); ctx.arc(h.x, h.y, h.d / 2, 0, 7); ctx.stroke(); }
}
