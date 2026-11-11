// Wirie editor UI — by nrmluvdev
"use strict";

const UI = {
  mode: "pcb", tool: "select", layer: "top", sel: new Set(), hoverPad: null, hoverComp: null, hoverNet: null,
  autoFix: (() => { try { return !!localStorage.getItem("wirie.autofix"); } catch (_) { return false; } })(), mouseW: [0, 0], mouseS: [0, 0], drag: null, wireFrom: null, track: null, placing: null, drc: [], drcShow: false, drcStale: true,
  focus: null, box: null, measure: null, dirty: true, space: false, exported: false, guideOpen: true, lastSaved: null,
};
const $ = s => document.querySelector(s);
const $$ = s => [...document.querySelectorAll(s)];
const cv = () => $("#cv");

// ---------- icons ----------
const IC = {
  select: '<path d="M5 3l14 8-6 2-2 6z"/>', wire: '<circle cx="5" cy="6" r="2.2"/><circle cx="19" cy="18" r="2.2"/><path d="M7 7c6 1 4 9 10 10" stroke-dasharray="2.5 2.5"/>',
  track: '<path d="M3 18h5l6-8h7"/><circle cx="3" cy="18" r="1.6"/><circle cx="21" cy="10" r="1.6"/>', via: '<circle cx="12" cy="12" r="6"/><circle cx="12" cy="12" r="2"/>',
  text: '<path d="M5 6V4h14v2M12 4v16M9 20h6"/>', measure: '<path d="M3 17L17 3l4 4L7 21z"/><path d="M7 13l2 2M10 10l2 2M13 7l2 2"/>',
  route: '<path d="M4 20V9a3 3 0 0 1 3-3h6"/><path d="M11 3l3 3-3 3"/><path d="M20 4v11a3 3 0 0 1-3 3h-6"/><path d="M13 21l-3-3 3-3"/>', pour: '<path d="M12 3s6 6.5 6 11a6 6 0 0 1-12 0c0-4.5 6-11 6-11z"/>',
  box: '<path d="M21 8l-9-5-9 5v8l9 5 9-5z"/><path d="M3 8l9 5 9-5M12 13v8"/>',
  check: '<path d="M9 12l2 2 4-4"/><path d="M12 3l8 3v6c0 5-3.5 8-8 9-4.5-1-8-4-8-9V6z"/>', layers: '<path d="M12 3l9 5-9 5-9-5z"/><path d="M3 13l9 5 9-5"/>',
  eye: '<path d="M2 12s4-7 10-7 10 7 10 7-4 7-10 7S2 12 2 12z"/><circle cx="12" cy="12" r="3"/>', fit: '<path d="M4 9V4h5M20 9V4h-5M4 15v5h5M20 15v5h-5"/>',
  undo: '<path d="M9 14L4 9l5-5"/><path d="M4 9h11a5 5 0 0 1 0 10h-3"/>', redo: '<path d="M15 14l5-5-5-5"/><path d="M20 9H9a5 5 0 0 0 0 10h3"/>',
  new: '<path d="M14 3H6a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9z"/><path d="M14 3v6h6M12 12v6M9 15h6"/>', open: '<path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v1H3z"/><path d="M3 10h18l-2 9H5z"/>',
  save: '<path d="M5 3h11l5 5v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2z"/><path d="M7 3v5h8V3M7 21v-7h10v7"/>', export: '<path d="M12 15V3M7 8l5-5 5 5"/><path d="M5 15v4a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2v-4"/>',
  help: '<circle cx="12" cy="12" r="9"/><path d="M9.5 9a2.5 2.5 0 1 1 3.5 2.3c-.8.4-1 1-1 1.7M12 17h.01"/>', rotate: '<path d="M20 12a8 8 0 1 1-2.3-5.7"/><path d="M20 4v5h-5"/>', flip: '<path d="M12 3v18"/><path d="M8 7L3 12l5 5zM16 7l5 5-5 5z"/>',
  trash: '<path d="M4 7h16M9 7V4h6v3M6 7l1 13h10l1-13"/>', magic: '<path d="M15 4V2M15 10V8M11 6h2M17 6h2M4 20L14 10"/><path d="M19 13l1 2 2 1-2 1-1 2-1-2-2-1 2-1z"/>', cut: '<circle cx="6" cy="6" r="3"/><circle cx="6" cy="18" r="3"/><path d="M8.5 7.5L20 18M8.5 16.5L20 6"/>',
  link: '<path d="M4 18c0-8 16-8 16 0"/><circle cx="4" cy="18" r="2"/><circle cx="20" cy="18" r="2"/>', print: '<path d="M6 9V3h12v6"/><rect x="3" y="9" width="18" height="8" rx="2"/><path d="M6 14h12v7H6z"/>', place: '<rect x="4" y="4" width="16" height="16" rx="2"/><path d="M9 4v16M15 4v16M4 9h16M4 15h16"/>',
  info: '<circle cx="12" cy="12" r="9"/><path d="M12 11v6M12 7h.01"/>', chip: '<rect x="6" y="6" width="12" height="12" rx="1.5"/><path d="M9 2v4M15 2v4M9 18v4M15 18v4M2 9h4M2 15h4M18 9h4M18 15h4"/>', search: '<circle cx="11" cy="11" r="7"/><path d="M20 20l-4-4"/>',
  board: '<rect x="3" y="5" width="18" height="14" rx="2"/><circle cx="7" cy="9" r="1"/><circle cx="17" cy="15" r="1"/><path d="M7 9h5l2 3h3v3"/>', strip: '<rect x="3" y="4" width="18" height="16" rx="2"/><path d="M3 9h18M3 14h18"/><path d="M10 7l2 4M14 7l-2 4" stroke-width="1.4"/>', calc: '<rect x="5" y="3" width="14" height="18" rx="2"/><path d="M8 7h8M8 12h2M12 12h2M16 12h0M8 16h2M12 16h2M16 16v0"/>',
  bolt: '<path d="M13 2L4 14h7l-1 8 9-12h-7z"/>', keyboard: '<rect x="2" y="6" width="20" height="12" rx="2"/><path d="M6 10h.01M10 10h.01M14 10h.01M18 10h.01M7 14h10"/>',
  star: '<path d="M12 3l2.6 5.6 6.1.7-4.5 4.2 1.2 6L12 16.8 6.6 19.5l1.2-6L3.3 9.3l6.1-.7z"/>', close: '<path d="M6 6l12 12M18 6L6 18"/>', folder: '<path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>',
};
const ico = (n, cls = "") => `<svg class="ic ${cls}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">${IC[n]}</svg>`;
const LOGO = `<svg viewBox="0 0 64 64" class="logo"><defs><linearGradient id="lg1" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#ffb26b"/><stop offset="1" stop-color="#e2683c"/></linearGradient><linearGradient id="lg2" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#5ff2e2"/><stop offset="1" stop-color="#2aa8c9"/></linearGradient></defs><rect x="4" y="4" width="56" height="56" rx="14" fill="#141a24" stroke="#2a3443" stroke-width="2"/><path d="M14 22h10l8 10h8l8-10" fill="none" stroke="url(#lg1)" stroke-width="4.5" stroke-linecap="round" stroke-linejoin="round"/><path d="M14 42h8l6-6h8l6 6h8" fill="none" stroke="url(#lg2)" stroke-width="4.5" stroke-linecap="round" stroke-linejoin="round"/><circle cx="14" cy="22" r="4" fill="#ffd38f"/><circle cx="50" cy="42" r="4" fill="#9ff7ee"/></svg>`;

// ---------- toast / modal ----------
function toast(msg, kind = "") { const box = $("#toasts"); while (box.children.length >= 3) box.firstElementChild.remove(); const t = document.createElement("div"); t.className = "toast " + kind; t.innerHTML = msg; $("#toasts").appendChild(t); requestAnimationFrame(() => t.classList.add("in")); setTimeout(() => { t.classList.remove("in"); setTimeout(() => t.remove(), 300); }, 3800); }
function modal(html, cls = "") {
  closeModal();
  const m = document.createElement("div"); m.className = "modal-wrap"; m.innerHTML = `<div class="modal ${cls}">${html}</div>`;
  m.addEventListener("mousedown", e => { if (e.target === m) closeModal(); });
  document.body.appendChild(m); requestAnimationFrame(() => m.classList.add("in"));
  m.querySelectorAll("[data-close]").forEach(b => b.onclick = closeModal);
  return m.querySelector(".modal");
}
function closeModal() { $$(".modal-wrap").forEach(m => m.remove()); }
function askText(title, label, value, ok) {
  const m = modal(`<h2>${title}</h2><label class="fld"><span>${label}</span><div class="inp"><input id="ask_in" value="${esc(value)}"></div></label><div class="mfoot"><button class="btn ghost" data-close>Cancel</button><button class="btn primary" id="ask_ok">OK</button></div>`, "small");
  const inp = m.querySelector("#ask_in"); inp.focus(); inp.select();
  const go = () => { const v = inp.value.trim(); closeModal(); if (v) ok(v); };
  m.querySelector("#ask_ok").onclick = go; inp.onkeydown = e => { if (e.key === "Enter") go(); if (e.key === "Escape") closeModal(); };
}

// ---------- API ----------
async function api(path, body) {
  const o = body !== undefined ? { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) } : {};
  const r = await fetch(path, o); if (!r.ok) throw new Error((await r.json().catch(() => ({}))).error || r.statusText); return r.json();
}
let autosaveT = null;
function onCommit() {
  UI.drcStale = true; UI.dirty = true;
  clearTimeout(autosaveT); autosaveT = setTimeout(() => { embedUserParts(); api("/api/autosave", S).catch(() => {}); }, 1200);
  refreshPanels();
}

// ---------- view ----------
function fitView() {
  const c = cv(), w = c.clientWidth, h = c.clientHeight;
  if (UI.mode === "strip") { const [bw, bh] = stripSize(); SV.zoom = Math.min((w - 120) / bw, (h - 120) / bh); SV.ox = (w - bw * SV.zoom) / 2; SV.oy = (h - bh * SV.zoom) / 2; }
  else { const B = S.board; VIEW.zoom = Math.max(1, Math.min((w - 160) / B.w, (h - 160) / B.h)); VIEW.ox = (w - B.w * VIEW.zoom) / 2; VIEW.oy = (h - B.h * VIEW.zoom) / 2 + 10; }
  UI.dirty = true;
}
function zoomAt(sx, sy, f) {
  if (UI.mode === "strip") { const z = Math.min(80, Math.max(2, SV.zoom * f)); f = z / SV.zoom; SV.ox = sx - (sx - SV.ox) * f; SV.oy = sy - (sy - SV.oy) * f; SV.zoom = z; }
  else { const z = Math.min(160, Math.max(0.8, VIEW.zoom * f)); f = z / VIEW.zoom; VIEW.ox = sx - (sx - VIEW.ox) * f; VIEW.oy = sy - (sy - VIEW.oy) * f; VIEW.zoom = z; }
  UI.dirty = true;
}
function resize() { const c = cv(), dpr = window.devicePixelRatio || 1; c.width = Math.round(c.clientWidth * dpr); c.height = Math.round(c.clientHeight * dpr); UI.dirty = true; }
function frame() {
  if (UI.dirty || UI.focus) { UI.dirty = false; if (UI.mode === "strip") drawStrip(cv()); else if (UI.mode === "pcb") drawPCB(cv()); }
  requestAnimationFrame(frame);
}

// ---------- hit testing ----------
function hitPCB(wx, wy) {
  const A = analyze(); const tol = 3 / VIEW.zoom;
  let best = null;
  for (const p of A.pads) { if (!p.drill && !p.layers.includes(UI.layer) && VIEW.mode !== "real") continue; if (ptInShape(wx, wy, padShape(p), tol)) { const d = dist(wx, wy, p.x, p.y); if (!best || d < best.d) best = { kind: "pad", obj: p, d }; } }
  if (best) return best;
  for (const v of S.vias) if (dist(wx, wy, v.x, v.y) < v.d / 2 + tol) return { kind: "via", obj: v };
  for (const L of [UI.layer, UI.layer === "top" ? "bottom" : "top"]) for (const t of S.tracks) { if (t.layer !== L || !VIEW.show[L]) continue; for (let i = 0; i < t.pts.length - 1; i++) if (ptSegDist(wx, wy, t.pts[i][0], t.pts[i][1], t.pts[i + 1][0], t.pts[i + 1][1]) < t.w / 2 + tol) return { kind: "track", obj: t, seg: i }; }
  for (const t of S.texts) { const w = textStrokes(t.text, t.size).width; const vert = (t.rot || 0) % 180; const hw = (vert ? t.size : w) / 2 + 0.5, hh = (vert ? w : t.size) / 2 + 0.5; if (Math.abs(wx - t.x) < hw && Math.abs(wy - t.y) < hh) return { kind: "text", obj: t }; }
  let cbest = null;
  for (const c of S.comps) { const b = compBox(c); if (wx >= b.x0 - tol && wx <= b.x1 + tol && wy >= b.y0 - tol && wy <= b.y1 + tol) { const a = (b.x1 - b.x0) * (b.y1 - b.y0); if (!cbest || a < cbest.a) cbest = { kind: "comp", obj: c, a }; } }
  return cbest;
}
function snapPt(wx, wy) { const g = S.board.grid; return [+snapTo(wx, g).toFixed(4), +snapTo(wy, g).toFixed(4)]; }

// ---------- tools ----------
const TOOLS_PCB = [
  ["select", "Select & move", "V", "Click parts to select. Drag to move. R rotates 90° (Shift+R 15°), or drag the round handle to any angle. F flips to the back."],
  ["wire", "Connect pins", "W", "Click a pin, then another pin, to say they should be connected. Yellow dashed lines show what still needs a track."],
  ["track", "Draw track", "T", "Click a pin to start a copper track, click to bend, click a pin to finish. V adds a via and swaps side. Esc cancels."],
  ["via", "Via", "I", "Click to drop a via (a plated hole that joins top and bottom copper)."],
  ["text", "Text", "X", "Click to write text on the silkscreen."],
  ["measure", "Measure", "M", "Drag to measure a distance."],
];
const TOOLS_STRIP = [
  ["select", "Select & move", "V", "Drag parts to move them. R rotates. Delete takes a part off the stripboard."],
  ["wire", "Connect pins", "W", "Click a pin, then another pin, to add a connection to the circuit."],
  ["cut", "Cut track", "C", "Click between two holes to cut (or un-cut) the copper strip there."],
  ["link", "Wire link", "L", "Click a hole, then another hole, to add a wire link. Click a link + Delete to remove it."],
];
function setTool(t) {
  UI.tool = t; UI.wireFrom = null; UI.track = null; UI.measure = null; SV.tool = t; SV.linkStart = null;
  $$("#tools [data-tool]").forEach(b => b.classList.toggle("on", b.dataset.tool === t));
  const list = UI.mode === "strip" ? TOOLS_STRIP : TOOLS_PCB; const d = list.find(x => x[0] === t);
  $("#hint").innerHTML = d ? `<b>${d[1]}</b> — ${d[3]}` : "";
  cv().style.cursor = t === "select" ? "default" : "crosshair";
  UI.dirty = true;
}
function buildTools() {
  const list = UI.mode === "strip" ? TOOLS_STRIP : TOOLS_PCB;
  $("#tools").innerHTML = list.map(([id, name, key]) => `<button data-tool="${id}" class="tb" data-tip="${name} (${key})">${ico(id === "cut" ? "cut" : id === "link" ? "link" : id)}</button>`).join("") + `<div class="sep"></div><button class="tb" id="tb_undo" data-tip="Undo (Ctrl+Z)">${ico("undo")}</button><button class="tb" id="tb_redo" data-tip="Redo (Ctrl+Y)">${ico("redo")}</button><button class="tb" id="tb_fit" data-tip="Fit to screen (Home)">${ico("fit")}</button>`;
  $$("#tools [data-tool]").forEach(b => b.onclick = () => setTool(b.dataset.tool));
  $("#tb_undo").onclick = () => { undo(); }; $("#tb_redo").onclick = () => { redo(); }; $("#tb_fit").onclick = fitView;
  if (!list.some(x => x[0] === UI.tool)) UI.tool = "select";
  setTool(UI.tool);
}

// ---------- overlay drawing (called from drawPCB) ----------
function trackPreview(from, to) {
  const dx = to[0] - from[0], dy = to[1] - from[1];
  if (Math.abs(dx) < 1e-6 || Math.abs(dy) < 1e-6 || Math.abs(Math.abs(dx) - Math.abs(dy)) < 1e-6) return [to];
  if (Math.abs(dx) > Math.abs(dy)) return [[to[0] - Math.sign(dx) * Math.abs(dy), from[1]], to];
  return [[from[0], to[1] - Math.sign(dy) * Math.abs(dx)], to];
}
function trackTarget() { const p = UI.hoverPad; if (p) return [p.x, p.y]; return snapPt(...UI.mouseW); }
function drawOverlay(ctx, px) {
  const A = analyze();
  if (UI.placing) {
    const fp = LIB_BY_ID[UI.placing.fp]; const [x, y] = snapPt(...UI.mouseW);
    const ghost = { id: "ghost", fp: fp.id, x, y, rot: UI.placing.rot, side: UI.placing.side, value: fp.value };
    ctx.globalAlpha = 0.55; drawPartBody(ctx, ghost, px * 10); ctx.globalAlpha = 0.9;
    for (const p of padsOf(ghost)) { ctx.fillStyle = "#3fe0d0"; padPath(ctx, p); ctx.fill(); }
    ctx.globalAlpha = 1;
  }
  if (UI.hoverPad && (UI.tool === "wire" || UI.tool === "track")) { const p = UI.hoverPad; ctx.strokeStyle = "#3fe0d0"; ctx.lineWidth = 2 * px; padPath(ctx, p, 0.25); ctx.stroke(); }
  if (UI.wireFrom && UI.mode === "pcb") {
    const p = UI.wireFrom; const t = UI.hoverPad || { x: UI.mouseW[0], y: UI.mouseW[1] };
    ctx.strokeStyle = "#ffd166"; ctx.lineWidth = 2 * px; ctx.setLineDash([6 * px, 4 * px]); ctx.beginPath(); ctx.moveTo(p.x, p.y); ctx.lineTo(t.x, t.y); ctx.stroke(); ctx.setLineDash([]);
    ctx.strokeStyle = "#ffd166"; padPath(ctx, p, 0.25); ctx.stroke();
  }
  if (UI.track) {
    const T = UI.track; const pts = [...T.pts, ...trackPreview(T.pts[T.pts.length - 1], trackTarget())];
    ctx.strokeStyle = T.layer === "top" ? COL.top : COL.bottom; ctx.globalAlpha = 0.85; ctx.lineWidth = T.w; ctx.lineCap = "round"; ctx.lineJoin = "round";
    ctx.beginPath(); pts.forEach((p, i) => (i ? ctx.lineTo(p[0], p[1]) : ctx.moveTo(p[0], p[1]))); ctx.stroke(); ctx.globalAlpha = 1;
    ctx.strokeStyle = "#fff"; ctx.lineWidth = px; ctx.setLineDash([3 * px, 3 * px]); ctx.beginPath(); pts.forEach((p, i) => (i ? ctx.lineTo(p[0], p[1]) : ctx.moveTo(p[0], p[1]))); ctx.stroke(); ctx.setLineDash([]);
  }
  if (UI.tool === "via" && !UI.placing) { const [x, y] = snapPt(...UI.mouseW); ctx.strokeStyle = "#3fe0d0"; ctx.lineWidth = 1.5 * px; ctx.beginPath(); ctx.arc(x, y, S.board.viaD / 2, 0, 7); ctx.stroke(); }
  if (UI.box) { const b = UI.box; ctx.fillStyle = "rgba(63,224,208,0.08)"; ctx.strokeStyle = "rgba(63,224,208,0.8)"; ctx.lineWidth = px; ctx.fillRect(b.x0, b.y0, b.x1 - b.x0, b.y1 - b.y0); ctx.strokeRect(b.x0, b.y0, b.x1 - b.x0, b.y1 - b.y0); }
  if (UI.measure) {
    const m = UI.measure; ctx.strokeStyle = "#ffd166"; ctx.lineWidth = 1.5 * px; ctx.beginPath(); ctx.moveTo(m.a[0], m.a[1]); ctx.lineTo(m.b[0], m.b[1]); ctx.stroke();
    const d = dist(m.a[0], m.a[1], m.b[0], m.b[1]);
    ctx.save(); ctx.translate((m.a[0] + m.b[0]) / 2, (m.a[1] + m.b[1]) / 2 - 8 * px); if (VIEW.flip) ctx.scale(-1, 1); ctx.font = `600 ${12 * px}px Segoe UI, sans-serif`; ctx.textAlign = "center"; ctx.fillStyle = "#ffd166";
    ctx.fillText(`${d.toFixed(2)} mm  (${(d / 25.4 * 1000).toFixed(0)} mil)  Δx ${Math.abs(m.b[0] - m.a[0]).toFixed(2)}  Δy ${Math.abs(m.b[1] - m.a[1]).toFixed(2)}`, 0, 0); ctx.restore();
  }
}

// ---------- mouse ----------
function canvasXY(e) { const r = cv().getBoundingClientRect(); return [e.clientX - r.left, e.clientY - r.top]; }
function onDown(e) {
  const [sx, sy] = canvasXY(e); cv().focus();
  if (e.button === 1 || e.button === 2 || (e.button === 0 && UI.space)) { UI.drag = { pan: true, sx, sy, ox: UI.mode === "strip" ? SV.ox : VIEW.ox, oy: UI.mode === "strip" ? SV.oy : VIEW.oy, moved: false, btn: e.button }; return; }
  if (e.button !== 0) return;
  if (UI.mode === "strip") return stripDown(e, sx, sy);
  const [wx, wy] = s2w(sx, sy); UI.mouseW = [wx, wy];
  if (UI.placing) {
    const [x, y] = snapPt(wx, wy); const c = addComp(UI.placing.fp, x, y, { rot: UI.placing.rot, side: UI.placing.side });
    UI.sel = new Set([c.id]); commit("place");
    const fp = compFP(c);
    if (fp.pinLabels && !fp.mcu && S.comps.some(q => compFP(q).mcu)) toast(`${ico("magic")} Tip: click <b>Smart connect</b> on the right to wire ${c.ref} to your microcontroller automatically.`);
    if (!e.shiftKey) UI.placing = null;
    return;
  }
  const rh = UI.tool === "select" && rotHandle();
  if (rh && dist(wx, wy, rh.x, rh.y) < 10 / VIEW.zoom) {
    const c = rh.c, pts = pinTrackPts([c]); pts.base = c.rot;
    UI.drag = { rot: true, c, pts, rot0: c.rot, a0: Math.atan2(wy - c.y, wx - c.x), moved: false }; return;
  }
  const hit = hitPCB(wx, wy);
  switch (UI.tool) {
    case "select": {
      if (hit && hit.kind === "pad") { hit.obj = hit.obj.comp; hit.kind = "comp"; }
      if (hit) {
        const id = hit.obj.id;
        if (e.shiftKey || e.ctrlKey) { UI.sel.has(id) ? UI.sel.delete(id) : UI.sel.add(id); }
        else if (!UI.sel.has(id)) UI.sel = new Set([id]);
        // start dragging selected comps/texts/vias
        const items = [];
        for (const c of S.comps) if (UI.sel.has(c.id)) items.push({ o: c, x: c.x, y: c.y });
        for (const t of S.texts) if (UI.sel.has(t.id)) items.push({ o: t, x: t.x, y: t.y });
        for (const v of S.vias) if (UI.sel.has(v.id)) items.push({ o: v, x: v.x, y: v.y });
        const tpts = [];
        const selPads = S.comps.filter(c => UI.sel.has(c.id)).flatMap(padsOf);
        for (const t of S.tracks) t.pts.forEach((p, i) => { if (UI.sel.has(t.id) || selPads.some(pd => ptInShape(p[0], p[1], padShape(pd), 0.01)) || S.vias.some(v => UI.sel.has(v.id) && dist(v.x, v.y, p[0], p[1]) < 0.01)) tpts.push({ p, x: p[0], y: p[1] }); });
        UI.drag = { move: true, wx, wy, items, tpts, moved: false };
      } else { if (!e.shiftKey) UI.sel.clear(); UI.drag = { boxStart: [wx, wy] }; }
      refreshPanels(); break;
    }
    case "wire": {
      if (hit && hit.kind === "pad") {
        if (!UI.wireFrom) UI.wireFrom = hit.obj;
        else if (hit.obj.key !== UI.wireFrom.key) {
          const a = UI.wireFrom, b = hit.obj;
          if (addConn(a.key, b.key)) { commit("connect"); const n = analyze().netOfPad[a.key]; toast(`Connected <b>${a.comp.ref}.${a.name}</b> → <b>${b.comp.ref}.${b.name}</b>${n ? ` <span class="pill" style="--c:${n.color}">${n.name}</span>` : ""}`); }
          else toast("Those pins are already connected.");
          UI.wireFrom = e.shiftKey ? b : null;
        }
      } else UI.wireFrom = null;
      break;
    }
    case "track": {
      if (!UI.track) {
        let start = null, layer = UI.layer;
        if (hit && hit.kind === "pad") { start = [hit.obj.x, hit.obj.y]; if (!hit.obj.drill) layer = hit.obj.layers[0]; }
        else if (hit && hit.kind === "via") start = [hit.obj.x, hit.obj.y];
        else if (hit && hit.kind === "track") { const t = hit.obj; const a = t.pts[hit.seg], b = t.pts[hit.seg + 1]; start = closestOnSeg(wx, wy, a, b); layer = t.layer; }
        else start = snapPt(wx, wy);
        if (layer !== UI.layer) setLayer(layer);
        const A = analyze(); let net = hit && hit.kind === "pad" ? A.netOfPad[hit.obj.key] : hit && hit.kind === "track" ? A.trackNet[hit.obj.id] : hit && hit.kind === "via" ? A.viaNet[hit.obj.id] : null;
        UI.track = { layer, w: net && isPowerNet(net.name) ? S.board.power : S.board.track, pts: [start], net };
      } else {
        const T = UI.track; const add = trackPreview(T.pts[T.pts.length - 1], trackTarget());
        T.pts.push(...add);
        const end = hit && (hit.kind === "pad" || hit.kind === "via" || (hit.kind === "track" && hit.obj.layer === T.layer));
        if (end || e.detail >= 2) finishTrack();
      }
      break;
    }
    case "via": { const [x, y] = snapPt(wx, wy); S.vias.push({ id: uid(), x, y, d: S.board.viaD, drill: S.board.viaDrill }); commit("via"); break; }
    case "text": { const [x, y] = snapPt(wx, wy); askText("Add silkscreen text", "Text", S.name || "My board", v => { const t = { id: uid(), x, y, text: v, size: 1.5, layer: UI.layer, rot: 0 }; S.texts.push(t); UI.sel = new Set([t.id]); commit("text"); }); break; }
    case "measure": { UI.measure = { a: [wx, wy], b: [wx, wy], live: true }; if (UI.hoverPad) UI.measure.a = [UI.hoverPad.x, UI.hoverPad.y]; break; }
  }
  UI.dirty = true;
}
function closestOnSeg(px, py, a, b) { const vx = b[0] - a[0], vy = b[1] - a[1], l2 = vx * vx + vy * vy; let t = l2 ? ((px - a[0]) * vx + (py - a[1]) * vy) / l2 : 0; t = Math.max(0, Math.min(1, t)); return [a[0] + t * vx, a[1] + t * vy]; }
function finishTrack() {
  const T = UI.track; if (!T) return;
  const pts = simplify(T.pts);
  if (pts.length >= 2) { S.tracks.push({ id: uid(), layer: T.layer, w: T.w, pts }); commit("track"); }
  UI.track = null; UI.dirty = true;
}
function onMove(e) {
  const [sx, sy] = canvasXY(e); UI.mouseS = [sx, sy];
  if (UI.drag && UI.drag.pan) { const d = UI.drag; const tgt = UI.mode === "strip" ? SV : VIEW; tgt.ox = d.ox + sx - d.sx; tgt.oy = d.oy + sy - d.sy; if (Math.abs(sx - d.sx) + Math.abs(sy - d.sy) > 3) d.moved = true; UI.dirty = true; return; }
  if (UI.mode === "strip") return stripMove(e, sx, sy);
  if (UI.mode !== "pcb") return;
  const [wx, wy] = s2w(sx, sy); UI.mouseW = [wx, wy];
  $("#coords").textContent = `X ${wx.toFixed(2)}  Y ${wy.toFixed(2)} mm`;
  if (UI.drag && UI.drag.move) {
    const d = UI.drag; const g = S.board.grid;
    let dx = wx - d.wx, dy = wy - d.wy;
    if (d.items.length) { const a = d.items[0]; dx = snapTo(a.x + dx, g) - a.x; dy = snapTo(a.y + dy, g) - a.y; }
    if (Math.abs(dx) + Math.abs(dy) > 0) d.moved = true;
    for (const it of d.items) { it.o.x = +(it.x + dx).toFixed(4); it.o.y = +(it.y + dy).toFixed(4); }
    for (const tp of d.tpts) { tp.p[0] = +(tp.x + dx).toFixed(4); tp.p[1] = +(tp.y + dy).toFixed(4); }
    invalidate(); return;
  }
  if (UI.drag && UI.drag.rot) {
    const d = UI.drag, c = d.c;
    let r = d.rot0 + (Math.atan2(wy - c.y, wx - c.x) - d.a0) * 180 / Math.PI * (VIEW.flip ? -1 : 1);
    r = e.shiftKey ? Math.round(r) : Math.round(r / 15) * 15; // 15° steps; hold Shift for 1°
    if (normRot(r) !== c.rot) d.moved = true;
    turnComp(c, r, d.pts); UI.rotShow = { x: wx, y: wy, a: c.rot };
    $("#coords").textContent = `${c.ref} at ${c.rot}°`; return;
  }
  if (UI.drag && UI.drag.boxStart) { const [ax, ay] = UI.drag.boxStart; UI.box = { x0: Math.min(ax, wx), y0: Math.min(ay, wy), x1: Math.max(ax, wx), y1: Math.max(ay, wy) }; UI.dirty = true; return; }
  if (UI.measure && UI.measure.live) { UI.measure.b = UI.hoverPad ? [UI.hoverPad.x, UI.hoverPad.y] : [wx, wy]; }
  // hover
  const rh = UI.tool === "select" && rotHandle();
  cv().style.cursor = rh && dist(wx, wy, rh.x, rh.y) < 10 / VIEW.zoom ? "grab" : "";
  const hit = hitPCB(wx, wy); const A = analyze();
  const hp = hit && hit.kind === "pad" ? hit.obj : null;
  UI.hoverPad = hp; UI.hoverComp = hit ? (hit.kind === "comp" ? hit.obj.id : hp ? hp.comp.id : null) : null;
  UI.hoverNet = hp ? A.netOfPad[hp.key] || null : hit && hit.kind === "track" ? A.trackNet[hit.obj.id] || null : hit && hit.kind === "via" ? A.viaNet[hit.obj.id] || null : null;
  showTip(e, hit, A);
  UI.dirty = true;
}
function showTip(e, hit, A) {
  const tip = $("#tip");
  if (!hit || UI.drag) { tip.classList.remove("on"); return; }
  let h = "";
  if (hit.kind === "pad") { const p = hit.obj, n = A.netOfPad[p.key]; const fp = compFP(p.comp); h = `<b>${p.comp.ref} · ${esc(p.name)}</b>${p.func && p.func !== "SIG" ? `<span class="tag">${p.func.replace(/^PWR\d?$/, "power")}</span>` : ""}<br><span class="dim">${esc(fp.name)} · pin ${p.n}${p.drill ? "" : " · SMD"}</span>${n ? `<br><span class="pill" style="--c:${n.color}">${esc(n.name)}</span> <span class="dim">${n.pads.length} pins${n.islands > 1 ? " · not fully routed" : " · routed ✓"}</span>` : `<br><span class="dim">not connected</span>`}`; }
  else if (hit.kind === "comp") { const c = hit.obj, fp = compFP(c); h = `<b>${c.ref}</b> ${esc(c.value)}<br><span class="dim">${esc(fp.name)}</span>`; }
  else if (hit.kind === "track") { const n = A.trackNet[hit.obj.id]; h = `<b>Track</b> ${hit.obj.w}mm · ${hit.obj.layer}${n ? `<br><span class="pill" style="--c:${n.color}">${esc(n.name)}</span>` : ""}`; }
  else if (hit.kind === "via") h = `<b>Via</b> ${hit.obj.d}mm / ${hit.obj.drill}mm drill`;
  else if (hit.kind === "text") h = `<b>Text</b> “${esc(hit.obj.text)}”`;
  tip.innerHTML = h; tip.classList.add("on");
  const r = $("#center").getBoundingClientRect();
  tip.style.left = Math.min(e.clientX - r.left + 16, r.width - 260) + "px"; tip.style.top = (e.clientY - r.top + 18) + "px";
}
function onUp(e) {
  const d = UI.drag; UI.drag = null;
  if (!d) return;
  if (d.pan) { if (d.btn === 2 && !d.moved) onContext(e); return; }
  if (UI.mode === "strip") return stripUp(e);
  if (d.move) { if (d.moved) commit("move"); refreshPanels(); }
  if (d.rot) { UI.rotShow = null; if (d.moved) commit("rotate"); refreshPanels(); }
  if (d.boxStart && UI.box) {
    const b = UI.box;
    for (const c of S.comps) { const r = compBox(c); if (r.x0 >= b.x0 && r.x1 <= b.x1 && r.y0 >= b.y0 && r.y1 <= b.y1) UI.sel.add(c.id); }
    for (const t of S.tracks) if (t.pts.every(p => p[0] >= b.x0 && p[0] <= b.x1 && p[1] >= b.y0 && p[1] <= b.y1)) UI.sel.add(t.id);
    for (const v of S.vias) if (v.x >= b.x0 && v.x <= b.x1 && v.y >= b.y0 && v.y <= b.y1) UI.sel.add(v.id);
    UI.box = null; refreshPanels();
  }
  if (UI.measure) UI.measure.live = false;
  UI.dirty = true;
}
function onContext(e) {
  if (UI.mode !== "pcb") return;
  const [sx, sy] = canvasXY(e); const [wx, wy] = s2w(sx, sy); const hit = hitPCB(wx, wy);
  if (UI.track) { finishTrack(); return; }
  if (UI.placing) { UI.placing.rot = (UI.placing.rot + 90) % 360; UI.dirty = true; return; }
  if (!hit) return;
  const items = [];
  if (hit.kind === "pad") {
    const p = hit.obj; const n = analyze().netOfPad[p.key];
    items.push([ico("wire") + "Connect from this pin", () => { setTool("wire"); UI.wireFrom = p; }]);
    items.push([ico("track") + "Draw track from here", () => { setTool("track"); UI.track = { layer: p.drill ? UI.layer : p.layers[0], w: n && isPowerNet(n.name) ? S.board.power : S.board.track, pts: [[p.x, p.y]], net: n }; }]);
    if (n) { items.push([ico("close") + `Disconnect ${esc(p.comp.ref)}.${esc(p.name)}`, () => { disconnectPad(p.key); commit("disconnect"); }]); items.push([ico("route") + `Auto-route net ${esc(n.name)}`, () => runAutoroute([n.name])]); }
    hit.obj = p.comp; hit.kind = "comp";
  }
  if (hit.kind === "comp") {
    const c = hit.obj; UI.sel = new Set([c.id]); refreshPanels();
    items.push([ico("rotate") + "Rotate (R)", () => rotateSel()]); items.push([ico("flip") + "Flip to other side (F)", () => flipSel()]);
    if (compFP(c).pinLabels) items.push([ico("magic") + "Smart connect…", () => smartConnect(c)]);
    items.push([ico("trash") + "Delete", () => deleteSel()]);
  } else if (hit.kind === "track" || hit.kind === "via" || hit.kind === "text") { UI.sel = new Set([hit.obj.id]); refreshPanels(); items.push([ico("trash") + "Delete", () => deleteSel()]); }
  showMenu(e.clientX, e.clientY, items);
}
function showMenu(x, y, items) {
  $$(".ctx").forEach(m => m.remove()); if (!items.length) return;
  const m = document.createElement("div"); m.className = "ctx"; m.innerHTML = items.map((it, i) => `<button data-i="${i}">${it[0]}</button>`).join("");
  document.body.appendChild(m); m.style.left = Math.min(x, innerWidth - 240) + "px"; m.style.top = Math.min(y, innerHeight - items.length * 36 - 16) + "px";
  m.querySelectorAll("button").forEach(b => b.onclick = () => { m.remove(); items[+b.dataset.i][1](); });
  setTimeout(() => document.addEventListener("mousedown", function h(ev) { if (!m.contains(ev.target)) { m.remove(); document.removeEventListener("mousedown", h); } }), 0);
}

// ---------- stripboard interaction ----------
function stripDown(e, sx, sy) {
  const [wx, wy] = ss2w(sx, sy); const h = stripHit(wx, wy); const st = S.strip;
  if (SV.placing) {
    const c = S.comps.find(q => q.id === SV.placing.id); const [x, y] = snapStripPos(c.fp, wx, wy, SV.placing.rot);
    st.place[c.id] = { x, y, rot: SV.placing.rot }; SV.placing = null; commit("strip place"); refreshPanels(); return;
  }
  if (UI.tool === "select") {
    if (h.link) { SV.sel = h.link; UI.dirty = true; return; }
    const comp = h.pad ? S.comps.find(q => q.id === h.pad.comp.id) : h.comp;
    if (comp) { SV.sel = comp.id; const p = st.place[comp.id]; UI.drag = { strip: true, id: comp.id, wx, wy, x: p.x, y: p.y, moved: false }; }
    else SV.sel = null;
  } else if (UI.tool === "cut") {
    if (h.gap[0] >= 0 && h.gap[0] < st.cols - 1 && h.r >= 0 && h.r < st.rows) {
      const k = h.gap.join(",");
      if (st.cuts.includes(k)) st.cuts = st.cuts.filter(x => x !== k);
      else if (st.autoCuts.includes(k)) st.autoCuts = st.autoCuts.filter(x => x !== k);
      else st.cuts.push(k);
      commit("cut");
    }
  } else if (UI.tool === "link") {
    if (!h.inBoard) { SV.linkStart = null; return; }
    if (!SV.linkStart) SV.linkStart = [h.c, h.r];
    else { if (SV.linkStart[0] !== h.c || SV.linkStart[1] !== h.r) { st.links.push({ a: SV.linkStart, b: [h.c, h.r] }); commit("link"); } SV.linkStart = e.shiftKey ? [h.c, h.r] : null; }
  } else if (UI.tool === "wire") {
    if (h.pad) { const pk = h.pad.key; if (!UI.wireFrom) UI.wireFrom = h.pad; else { if (addConn(UI.wireFrom.key, pk)) { commit("connect"); toast(`Connected <b>${UI.wireFrom.comp.ref}.${UI.wireFrom.name}</b> → <b>${h.pad.comp.ref}.${h.pad.name}</b>. Press <b>Do it all</b> to update cuts & links.`); } UI.wireFrom = null; } }
    else UI.wireFrom = null;
  }
  UI.dirty = true; refreshPanels();
}
function stripMove(e, sx, sy) {
  const [wx, wy] = ss2w(sx, sy); UI.mouseW = [wx, wy];
  const st = S.strip;
  const c = Math.round((wx - SM) / SP), r = Math.round((wy - SM) / SP);
  $("#coords").textContent = c >= 0 && r >= 0 && c < st.cols && r < st.rows ? `Hole ${rowName(r)}${c + 1}` : "";
  if (UI.drag && UI.drag.strip) {
    const d = UI.drag; const comp = S.comps.find(q => q.id === d.id); const p = st.place[d.id];
    const [x, y] = snapStripPos(comp.fp, d.x + (wx - d.wx) + (rotPt(compFP(comp).pads[0].x, compFP(comp).pads[0].y, p.rot)[0]) * 0, d.y + (wy - d.wy), p.rot);
    if (x !== p.x || y !== p.y) { p.x = x; p.y = y; d.moved = true; stripInvalidate(); }
    UI.dirty = true; return;
  }
  if (SV.placing) { SV.placing.x = wx; SV.placing.y = wy; const comp = S.comps.find(q => q.id === SV.placing.id); [SV.placing.x, SV.placing.y] = snapStripPos(comp.fp, wx, wy, SV.placing.rot); }
  const h = stripHit(wx, wy);
  SV.hover = h.inBoard || UI.tool === "cut" ? { c: UI.tool === "cut" ? h.gap[0] : h.c, r: h.r } : null;
  const tip = $("#tip");
  if (h.pad) { const n = analyze().netOfPad[h.pad.key]; tip.innerHTML = `<b>${h.pad.comp.ref} · ${esc(h.pad.name)}</b> <span class="dim">hole ${rowName(h.pad.r)}${h.pad.c + 1}</span>${n ? `<br><span class="pill" style="--c:${n.color}">${esc(n.name)}</span>` : "<br><span class='dim'>not connected</span>"}`; tip.classList.add("on"); const rr = $("#center").getBoundingClientRect(); tip.style.left = (e.clientX - rr.left + 16) + "px"; tip.style.top = (e.clientY - rr.top + 18) + "px"; }
  else tip.classList.remove("on");
  UI.dirty = true;
}
function stripUp() { const d = UI.drag; UI.drag = null; if (d && d.strip && d.moved) commit("strip move"); }

// ---------- edit actions ----------
function setLayer(l) { UI.layer = l; $$("#layerSw button").forEach(b => b.classList.toggle("on", b.dataset.l === l)); UI.dirty = true; }
function rotateSel() {
  if (UI.mode === "strip") { if (SV.placing) { SV.placing.rot = (SV.placing.rot + 90) % 360; } else if (typeof SV.sel === "string" && S.strip.place[SV.sel]) { const p = S.strip.place[SV.sel]; const c = S.comps.find(q => q.id === SV.sel); const p1 = compFP(c).pads[0]; const [ax, ay] = rotPt(p1.x, p1.y, p.rot); p.rot = (p.rot + 90) % 360; [p.x, p.y] = snapStripPos(c.fp, p.x + ax - rotPt(p1.x, p1.y, p.rot)[0], p.y + ay - rotPt(p1.x, p1.y, p.rot)[1], p.rot); commit("strip rotate"); } UI.dirty = true; return; }
  const step = arguments[0] || 90;
  if (UI.placing) { UI.placing.rot = normRot(UI.placing.rot + step); UI.dirty = true; return; }
  let n = 0; for (const c of S.comps) if (UI.sel.has(c.id)) { turnComp(c, c.rot + step); n++; }
  for (const t of S.texts) if (UI.sel.has(t.id)) { t.rot = normRot((t.rot || 0) + step); n++; }
  if (n) commit("rotate");
}
// any angle, kept between 0 and 360 and to a tenth of a degree
function normRot(r) { r = Math.round((((r % 360) + 360) % 360) * 10) / 10; return r >= 360 ? 0 : r; }
// track ends sitting on these parts' pins (they turn with the part)
function pinTrackPts(comps) {
  const shapes = comps.flatMap(padsOf).map(padShape), out = [];
  for (const t of S.tracks) for (const p of t.pts) if (shapes.some(s => ptInShape(p[0], p[1], s, 0.01))) out.push({ p, x: p[0], y: p[1] });
  return out;
}
// turn a part to an angle about its origin; tracks on its pins follow
function turnComp(c, rot, pts) {
  rot = normRot(rot); const d = rot - c.rot; if (!d && !pts) return;
  pts = pts || pinTrackPts([c]); const base = pts.base ?? c.rot; pts.base = base;
  for (const q of pts) { const [x, y] = rotPt(q.x - c.x, q.y - c.y, rot - base); q.p[0] = +(c.x + x).toFixed(4); q.p[1] = +(c.y + y).toFixed(4); }
  c.rot = rot; invalidate();
}
// the round rotate handle above a single selected part
function rotHandle() {
  if (UI.mode !== "pcb" || UI.sel.size !== 1 || UI.placing) return null;
  const c = S.comps.find(q => UI.sel.has(q.id)); if (!c || !compFP(c)) return null;
  const b = compBox(c); return { c, x: (b.x0 + b.x1) / 2, y: b.y0 - 0.4 - 22 / VIEW.zoom, top: b.y0 - 0.4 };
}
function flipSel() {
  if (UI.placing) { UI.placing.side = UI.placing.side === "top" ? "bottom" : "top"; UI.dirty = true; return; }
  let n = 0; for (const c of S.comps) if (UI.sel.has(c.id)) { c.side = c.side === "bottom" ? "top" : "bottom"; n++; }
  for (const t of S.texts) if (UI.sel.has(t.id)) { t.layer = t.layer === "bottom" ? "top" : "bottom"; n++; }
  if (n) { commit("flip"); toast("Flipped to the " + (S.comps.find(c => UI.sel.has(c.id))?.side || "other") + " side."); }
}
function deleteSel() {
  if (UI.mode === "strip") {
    if (SV.sel && typeof SV.sel === "object") { S.strip.links = S.strip.links.filter(l => l !== SV.sel); S.strip.autoLinks = S.strip.autoLinks.filter(l => l !== SV.sel); SV.sel = null; commit("unlink"); }
    else if (SV.sel) { delete S.strip.place[SV.sel]; SV.sel = null; commit("unplace"); }
    return;
  }
  if (!UI.sel.size) return;
  for (const id of [...UI.sel]) if (S.comps.some(c => c.id === id)) removeComp(id);
  S.tracks = S.tracks.filter(t => !UI.sel.has(t.id)); S.vias = S.vias.filter(v => !UI.sel.has(v.id)); S.texts = S.texts.filter(t => !UI.sel.has(t.id));
  UI.sel.clear(); commit("delete");
}
function duplicateSel() {
  const ids = []; for (const c of S.comps.filter(c => UI.sel.has(c.id))) { const n = addComp(c.fp, c.x + 5.08, c.y + 5.08, { rot: c.rot, side: c.side, value: c.value }); ids.push(n.id); }
  if (ids.length) { UI.sel = new Set(ids); commit("duplicate"); }
}
function startPlacing(fpId) {
  if (document.activeElement && document.activeElement.blur) document.activeElement.blur(); cv().focus();
  if (UI.mode === "calc") switchMode("pcb");
  if (UI.mode === "strip") {
    const fp = LIB_BY_ID[fpId]; if (!fp.pads.length || !fp.pads.every(p => p.drill && !p.slot)) { toast("That's a surface-mount part — it won't fit a stripboard. Pick a through-hole version.", "warn"); return; }
    const c = addComp(fpId, -20, -20); c.x = S.board.w + 10; c.y = 5 + S.comps.length * 3;
    commit("add for strip"); SV.placing = { id: c.id, rot: stripRotFor(c), x: 0, y: 0 }; refreshPanels(); return;
  }
  setTool("select"); UI.placing = { fp: fpId, rot: 0, side: "top" }; UI.dirty = true;
  $("#hint").innerHTML = `<b>Placing ${esc(LIB_BY_ID[fpId].name)}</b> — click to drop it. R rotates, F puts it on the back, Shift+click keeps placing, Esc stops.`;
}

// ---------- keyboard ----------
function onKey(e) {
  if (e.target.matches("input,textarea,select")) { if (e.key === "Escape") e.target.blur(); return; }
  if ($(".modal-wrap")) { if (e.key === "Escape") closeModal(); return; }
  if (UI.mode === "keeb" && kbKey(e)) return;
  const k = e.key.toLowerCase(), ctrl = e.ctrlKey || e.metaKey;
  if (ctrl && k === "z") { e.preventDefault(); e.shiftKey ? redo() : undo(); return; }
  if (ctrl && k === "y") { e.preventDefault(); redo(); return; }
  if (ctrl && k === "s") { e.preventDefault(); saveProject(); return; }
  if (ctrl && k === "o") { e.preventDefault(); openDialog(); return; }
  if (ctrl && k === "d") { e.preventDefault(); duplicateSel(); return; }
  if (ctrl && e.shiftKey && k === "p") { e.preventDefault(); openPartCreator(); return; }
  if (ctrl && k === "a" && UI.mode === "pcb") { e.preventDefault(); UI.sel = new Set(S.comps.map(c => c.id)); refreshPanels(); UI.dirty = true; return; }
  if (["calc", "pinnie", "power", "keeb"].includes(UI.mode)) return;
  if (e.key === " ") { UI.space = true; cv().style.cursor = "grab"; e.preventDefault(); return; }
  if (e.key === "Escape") { if (UI.track) { UI.track = null; } else if (UI.placing || SV.placing) { if (SV.placing) { removeComp(SV.placing.id); SV.placing = null; commit("cancel place"); } UI.placing = null; setTool(UI.tool); } else if (UI.wireFrom || SV.linkStart) { UI.wireFrom = null; SV.linkStart = null; } else { UI.sel.clear(); SV.sel = null; setTool("select"); refreshPanels(); } UI.dirty = true; return; }
  if (e.key === "Delete" || e.key === "Backspace") { if (UI.track) { if (UI.track.pts.length > 1) UI.track.pts.pop(); else UI.track = null; UI.dirty = true; } else deleteSel(); return; }
  if (e.key === "Home") { fitView(); return; }
  if (e.key === "Enter" && UI.track) { finishTrack(); return; }
  if (k === "r") return rotateSel(e.shiftKey ? 15 : 90);
  if (k === "f" && UI.mode === "pcb") return flipSel();
  if (UI.mode === "pcb" && UI.track && k === "v") {
    const T = UI.track; T.pts.push(...trackPreview(T.pts[T.pts.length - 1], trackTarget()));
    const end = T.pts[T.pts.length - 1]; const w = T.w, net = T.net;
    S.vias.push({ id: uid(), x: end[0], y: end[1], d: S.board.viaD, drill: S.board.viaDrill });
    finishTrack(); const nl = T.layer === "top" ? "bottom" : "top"; setLayer(nl); UI.track = { layer: nl, w, pts: [end], net }; return;
  }
  const map = UI.mode === "strip" ? { v: "select", w: "wire", c: "cut", l: "link" } : { v: "select", w: "wire", t: "track", i: "via", x: "text", m: "measure" };
  if (map[k]) return setTool(map[k]);
  if (UI.mode === "pcb") { if (k === "l" || k === "tab") { e.preventDefault(); setLayer(UI.layer === "top" ? "bottom" : "top"); } if (k === "1") setLayer("top"); if (k === "2") setLayer("bottom"); if (k === "g") { const gs = [0.5, 0.635, 1, 1.27, 2.54]; S.board.grid = gs[(gs.indexOf(S.board.grid) + 1) % gs.length]; toast(`Grid: ${S.board.grid} mm`); refreshPanels(); UI.dirty = true; } }
  if (e.key === "?") showHelp();
}

// ---------- panels ----------
function buildLibrary() {
  const q = ($("#libq").value || "").toLowerCase().trim();
  const words = q.split(/\s+/).filter(Boolean);
  const box = $("#liblist"); box.innerHTML = "";
  const frag = document.createDocumentFragment(); let total = 0;
  for (const cat of CATEGORIES) {
    const items = LIB.filter(f => f.cat === cat && words.every(w => f.search.includes(w)));
    if (!items.length && (cat !== MYCAT || q)) continue; total += items.length;
    const open = q ? true : (UI.libOpen || { Microcontrollers: true, "Raspberry Pi": true, [MYCAT]: true })[cat];
    const sec = document.createElement("div"); sec.className = "libcat" + (open ? " open" : "");
    sec.innerHTML = `<button class="cathead"><span>${cat}</span><em>${items.length}</em><svg class="chev" viewBox="0 0 24 24"><path d="M9 6l6 6-6 6" fill="none" stroke="currentColor" stroke-width="2"/></svg></button><div class="items"></div>`;
    sec.querySelector(".cathead").onclick = () => { UI.libOpen = UI.libOpen || { Microcontrollers: true, "Raspberry Pi": true }; UI.libOpen[cat] = !sec.classList.contains("open"); sec.classList.toggle("open"); if (sec.classList.contains("open")) lazyPreviews(sec); };
    const list = sec.querySelector(".items");
    if (cat === MYCAT) { const mk = document.createElement("button"); mk.className = "libnew"; mk.innerHTML = `${ico("new")}<span><b>Create a part</b><small>Make a footprint for anything not in the library</small></span>`; mk.onclick = () => openPartCreator(); list.appendChild(mk); }
    for (const f of items.slice(0, q ? 80 : 400)) {
      const it = document.createElement("div"); it.className = "libitem"; it.draggable = true; it.dataset.fp = f.id;
      it.innerHTML = `<canvas width="88" height="88"></canvas><div class="li-t"><b>${esc(f.name)}</b><small>${esc(f.desc || (f.pads.length + " pins" + (f.smd ? " · surface-mount" : "")))}</small></div>${f.verify ? `<span class="warnb" title="Check pin positions against your real module">!</span>` : ""}`;
      it.onclick = e => { if (e.target.closest(".libedit")) return; startPlacing(f.id); };
      if (f.user) { const ed = document.createElement("button"); ed.className = "libedit"; ed.title = "Edit this part"; ed.innerHTML = ico("text"); ed.onclick = () => openPartCreator(f.id); it.appendChild(ed); }
      it.ondragstart = ev => { ev.dataTransfer.setData("text/wirie-fp", f.id); };
      list.appendChild(it);
    }
    frag.appendChild(sec);
  }
  box.appendChild(frag);
  $("#libcount").textContent = q ? `${total} matches` : `${LIB.length} parts`;
  lazyPreviews(box);
}
const previewObs = new IntersectionObserver(ents => { for (const e of ents) if (e.isIntersecting) { const it = e.target; const fp = LIB_BY_ID[it.dataset.fp]; drawFPPreview(it.querySelector("canvas"), fp); previewObs.unobserve(it); } }, { root: null, rootMargin: "200px" });
function lazyPreviews(root) { root.querySelectorAll(".libcat.open .libitem").forEach(it => previewObs.observe(it)); }

function refreshPanels() {
  if (UI.mode === "pcb") renderProps();
  if (UI.mode === "strip") renderStripPanel();
  if (UI.mode === "keeb") kbPage();
  renderGuide(); renderStatus();
  $("#projname").value = S.name;
}
function renderStatus() {
  const A = analyze();
  $("#stats").innerHTML = UI.mode === "strip" ? (() => { const SA = stripAnalyze(); return `<span>${SA.placed.length} placed</span><span class="${SA.shorts.length ? "bad" : ""}">${SA.shorts.length} shorts</span><span class="${SA.missing.length ? "warn" : ""}">${SA.missing.length} unconnected</span>`; })()
    : `<span>${S.comps.length} parts</span><span>${A.nets.length} nets</span><span class="${A.unrouted ? "warn" : A.nets.length ? "good" : ""}">${A.unrouted ? A.unrouted + " to route" : A.nets.length ? "all routed ✓" : "no connections yet"}</span>`;
}
function fieldHTML(id, label, val, unit = "", type = "text") { return `<label class="fld"><span>${label}</span><div class="inp"><input id="${id}" type="${type}" value="${esc(val)}" spellcheck="false" autocomplete="off">${unit ? `<em>${unit}</em>` : ""}</div></label>`; }
function renderProps() {
  const P = $("#props"); const A = analyze();
  const tab = UI.rtab || "props";
  $$("#rtabs button").forEach(b => b.classList.toggle("on", b.dataset.t === tab));
  if (tab === "nets") return renderNets(P, A);
  if (tab === "check") return renderCheck(P);
  const selC = S.comps.filter(c => UI.sel.has(c.id));
  const selT = S.tracks.filter(t => UI.sel.has(t.id)), selV = S.vias.filter(v => UI.sel.has(v.id)), selX = S.texts.filter(t => UI.sel.has(t.id));
  const n = selC.length + selT.length + selV.length + selX.length;
  if (n === 0) return renderBoardProps(P, A);
  if (n > 1) { P.innerHTML = `<div class="ph"><h3>${n} items selected</h3><p class="dim">${selC.length} parts · ${selT.length} tracks · ${selV.length} vias · ${selX.length} texts</p></div><div class="btnrow"><button class="btn" id="p_rot">${ico("rotate")}Rotate</button><button class="btn" id="p_flip">${ico("flip")}Flip</button><button class="btn danger" id="p_del">${ico("trash")}Delete</button></div>`; $("#p_rot").onclick = rotateSel; $("#p_flip").onclick = flipSel; $("#p_del").onclick = deleteSel; return; }
  if (selC.length) {
    const c = selC[0], fp = compFP(c);
    const pads = padsOf(c);
    P.innerHTML = `<div class="ph comp"><canvas id="p_prev" width="240" height="150"></canvas><h3>${esc(fp.name)}</h3><p class="dim">${esc(fp.desc || fp.cat)}</p>${fp.verify ? `<div class="callout warn">${ico("info")}Pin positions vary between makers — measure your module before ordering boards.</div>` : ""}</div>
      ${fp.user ? `<button class="btn wide" id="p_editpart">${ico("chip")}Edit this part's design</button>` : ""}
      <div class="grid2">${fieldHTML("p_ref", "Reference", c.ref)}${fieldHTML("p_val", "Value / label", c.value)}</div>
      <div class="grid3">${fieldHTML("p_x", "X", c.x, "mm", "number")}${fieldHTML("p_y", "Y", c.y, "mm", "number")}${fieldHTML("p_ang", "Angle", c.rot, "°", "number")}</div>
      <div class="btnrow"><button class="btn" id="p_rot" title="Rotate 90° (R)">${ico("rotate")}Rotate</button><button class="btn icon" id="p_r15" title="Rotate 15° (Shift+R)">15°</button><button class="btn" id="p_flip">${ico("flip")}${c.side === "bottom" ? "Move to top" : "Move to back"}</button><button class="btn danger icon" id="p_del" title="Delete">${ico("trash")}</button></div>
      ${fp.pinLabels ? `<button class="btn primary wide" id="p_smart">${ico("magic")}Smart connect…</button><label class="chk"><input type="checkbox" id="p_silk" ${c.silkPins ? "checked" : ""}><span>Print pin names on the board</span></label>` : ""}
      <label class="chk"><input type="checkbox" id="p_hide" ${c.hideRef ? "checked" : ""}><span>Hide reference text (${esc(c.ref)})</span></label>
      <h4>Case / 3D <em>for Makie</em></h4><div class="grid2">${fieldHTML("p_ch", "Part height", caseInfo(fp, c).h, "mm", "number")}${fieldHTML("p_rz", "Raised above board", caseInfo(fp, c).raise ?? 0, "mm", "number")}</div>
      <h4>Pins <em>${pads.length}</em></h4><div class="pinlist">${pads.map(p => { const nn = A.netOfPad[p.key]; return `<div class="pin" data-k="${p.key}"><b>${esc(p.name)}</b><span>${nn ? `<span class="pill" style="--c:${nn.color}">${esc(nn.name)}</span>` : `<span class="dim">—</span>`}</span>${nn ? `<button class="x" data-dis="${p.key}" title="Disconnect">${ico("close")}</button>` : `<button class="x" data-con="${p.key}" title="Connect from here">${ico("wire")}</button>`}</div>`; }).join("")}</div>`;
    drawFPPreview($("#p_prev"), fp);
    const upd = () => { c.ref = $("#p_ref").value.trim() || c.ref; c.value = $("#p_val").value; const x = parseFloat($("#p_x").value), y = parseFloat($("#p_y").value); if (isFinite(x)) c.x = x; if (isFinite(y)) c.y = y; commit("props"); };
    ["#p_ref", "#p_val", "#p_x", "#p_y"].forEach(s => $(s).onchange = upd);
    $("#p_rot").onclick = () => rotateSel(90); $("#p_r15").onclick = () => rotateSel(15);
    $("#p_ang").onchange = () => { const v = parseFloat($("#p_ang").value); if (isFinite(v)) { turnComp(c, v); commit("angle"); refreshPanels(); } }; $("#p_flip").onclick = flipSel; $("#p_del").onclick = deleteSel;
    if ($("#p_editpart")) $("#p_editpart").onclick = () => openPartCreator(fp.id);
    if ($("#p_smart")) $("#p_smart").onclick = () => smartConnect(c);
    if ($("#p_silk")) $("#p_silk").onchange = e => { c.silkPins = e.target.checked; commit("silk"); };
    $("#p_hide").onchange = e => { c.hideRef = e.target.checked; commit("hide"); };
    $("#p_ch").onchange = () => { const v = parseFloat($("#p_ch").value); if (v >= 0) { c.caseH = v; commit("case h"); } };
    $("#p_rz").onchange = () => { const v = parseFloat($("#p_rz").value); if (v >= 0) { c.raise = v; commit("raise"); } };
    P.querySelectorAll("[data-dis]").forEach(b => b.onclick = () => { disconnectPad(b.dataset.dis); commit("disconnect"); });
    P.querySelectorAll("[data-con]").forEach(b => b.onclick = () => { setTool("wire"); UI.wireFrom = analyze().padByKey[b.dataset.con]; UI.dirty = true; toast("Now click the pin to connect it to."); });
    P.querySelectorAll(".pin").forEach(el => { el.onmouseenter = () => { UI.hoverNet = analyze().netOfPad[el.dataset.k] || null; UI.hoverPad = analyze().padByKey[el.dataset.k]; UI.dirty = true; }; el.onmouseleave = () => { UI.hoverNet = null; UI.hoverPad = null; UI.dirty = true; }; });
    return;
  }
  if (selT.length) {
    const t = selT[0]; const nn = A.trackNet[t.id];
    P.innerHTML = `<div class="ph"><h3>Copper track</h3><p class="dim">${nn ? `Net <span class="pill" style="--c:${nn.color}">${esc(nn.name)}</span>` : "Not connected to anything"}${t.auto ? " · auto-routed" : ""}</p></div>
      <div class="grid2">${fieldHTML("p_w", "Width", t.w, "mm", "number")}<label class="fld"><span>Side</span><div class="inp"><select id="p_l"><option value="top" ${t.layer === "top" ? "selected" : ""}>Top</option><option value="bottom" ${t.layer === "bottom" ? "selected" : ""}>Bottom</option></select></div></label></div>
      <p class="note">Rule of thumb: 0.3mm for signals, 0.6–1mm for power. Use Calcie's trace-width calculator for big currents.</p><div class="btnrow"><button class="btn danger" id="p_del">${ico("trash")}Delete track</button></div>`;
    $("#p_w").onchange = () => { const v = parseFloat($("#p_w").value); if (v > 0.05) { t.w = v; t.auto = false; commit("w"); } };
    $("#p_l").onchange = () => { t.layer = $("#p_l").value; commit("layer"); }; $("#p_del").onclick = deleteSel; return;
  }
  if (selV.length) { const v = selV[0]; P.innerHTML = `<div class="ph"><h3>Via</h3><p class="dim">Joins the top and bottom copper.</p></div><div class="grid2">${fieldHTML("p_vd", "Pad size", v.d, "mm", "number")}${fieldHTML("p_vh", "Drill", v.drill, "mm", "number")}</div><div class="btnrow"><button class="btn danger" id="p_del">${ico("trash")}Delete via</button></div>`; $("#p_vd").onchange = () => { v.d = parseFloat($("#p_vd").value) || v.d; commit(); }; $("#p_vh").onchange = () => { v.drill = parseFloat($("#p_vh").value) || v.drill; commit(); }; $("#p_del").onclick = deleteSel; return; }
  if (selX.length) {
    const t = selX[0];
    P.innerHTML = `<div class="ph"><h3>Silkscreen text</h3></div>${fieldHTML("p_tt", "Text", t.text)}<div class="grid2">${fieldHTML("p_ts", "Height", t.size, "mm", "number")}<label class="fld"><span>Side</span><div class="inp"><select id="p_tl"><option value="top" ${t.layer === "top" ? "selected" : ""}>Top</option><option value="bottom" ${t.layer === "bottom" ? "selected" : ""}>Bottom</option></select></div></label></div><div class="btnrow"><button class="btn" id="p_rot">${ico("rotate")}Rotate</button><button class="btn danger" id="p_del">${ico("trash")}Delete</button></div>`;
    $("#p_tt").oninput = () => { t.text = $("#p_tt").value; invalidate(); }; $("#p_tt").onchange = () => commit("text"); $("#p_ts").onchange = () => { t.size = Math.max(0.6, parseFloat($("#p_ts").value) || 1.5); commit(); }; $("#p_tl").onchange = () => { t.layer = $("#p_tl").value; commit(); }; $("#p_rot").onclick = rotateSel; $("#p_del").onclick = deleteSel;
  }
}
function renderBoardProps(P, A) {
  const B = S.board;
  P.innerHTML = `<div class="ph"><h3>${ico("board")}Board</h3><p class="dim">Nothing selected — these settings are for the whole board.</p></div>
    ${B.polys && B.polys.length ? `<div class="note">Custom outline${S.keeb ? " from Keebie" : ""}: ${B.w.toFixed(1)} × ${B.h.toFixed(1)} mm${B.cutouts && B.cutouts.length ? `, ${B.cutouts.length} cut-outs` : ""}.</div><div class="grid2">${S.keeb ? `<button class="btn" id="b_keeb">${ico("keyboard")}Edit in Keebie</button>` : ""}<button class="btn" id="b_rect">Use a rectangle</button></div>` : `<div class="grid2">${fieldHTML("b_w", "Width", B.w, "mm", "number")}${fieldHTML("b_h", "Height", B.h, "mm", "number")}</div>
    <div class="grid2">${fieldHTML("b_r", "Corner radius", B.r, "mm", "number")}<label class="fld"><span>Shape</span><div class="inp"><select id="b_shape"><option value="rect" ${!B.round ? "selected" : ""}>Rectangle</option><option value="round" ${B.round ? "selected" : ""}>Round</option></select></div></label></div>`}
    <button class="btn wide" id="b_tpl">${ico("place")}Board templates (Pi HAT, Pico, ESP32…)</button>
    <h4>Copper</h4>
    <div class="grid2">${fieldHTML("b_tw", "Signal track", B.track, "mm", "number")}${fieldHTML("b_pw", "Power track", B.power, "mm", "number")}</div>
    <div class="grid2">${fieldHTML("b_cl", "Clearance", B.clear, "mm", "number")}${fieldHTML("b_vd", "Via size / drill", B.viaD + " / " + B.viaDrill, "mm")}</div>
    <h4>Ground fill</h4><p class="note">Fills empty copper with a net (usually GND) — less routing, less noise.</p>
    <div class="grid2"><label class="chk"><input type="checkbox" id="b_pt" ${B.pourTop ? "checked" : ""}><span>Top side</span></label><label class="chk"><input type="checkbox" id="b_pb" ${B.pourBot ? "checked" : ""}><span>Bottom side</span></label></div>
    <label class="chk" data-tip="The auto-router also draws tracks for the fill net (usually GND), so every connection is visible and still works if you turn the fill off. Recommended."><input type="checkbox" id="b_pr" ${B.pourRoute !== false ? "checked" : ""}><span>Route the fill net with tracks too</span></label>
    <label class="fld"><span>Fill net</span><div class="inp"><select id="b_pn">${[...new Set(["GND", ...A.nets.map(n => n.name)])].map(n => `<option ${n === B.pourNet ? "selected" : ""}>${esc(n)}</option>`).join("")}</select></div></label>
    <h4>Grid</h4><div class="seg" id="b_grid">${[0.5, 0.635, 1, 1.27, 2.54].map(g => `<button data-g="${g}" class="${g === B.grid ? "on" : ""}">${g}</button>`).join("")}</div>
    <div class="statbox"><div><b>${S.comps.length}</b><span>parts</span></div><div><b>${A.nets.length}</b><span>nets</span></div><div><b>${S.tracks.length}</b><span>tracks</span></div><div><b>${S.vias.length}</b><span>vias</span></div></div>`;
  const num = id => parseFloat($(id).value);
  const ch = (id, f) => $(id).onchange = () => { f(); commit("board"); refreshPanels(); };
  if (B.polys && B.polys.length) {
    if ($("#b_keeb")) $("#b_keeb").onclick = () => switchMode("keeb");
    $("#b_rect").onclick = () => { delete B.polys; B.cutouts = []; commit("board rectangle"); refreshPanels(); };
  } else {
    ch("#b_w", () => { if (num("#b_w") >= 5) B.w = num("#b_w"); }); ch("#b_h", () => { if (num("#b_h") >= 5) B.h = num("#b_h"); }); ch("#b_r", () => { B.r = Math.max(0, num("#b_r") || 0); });
    ch("#b_shape", () => { B.round = $("#b_shape").value === "round"; if (B.round) B.h = B.w = Math.max(B.w, B.h); });
  }
  ch("#b_tw", () => { if (num("#b_tw") >= 0.1) B.track = num("#b_tw"); }); ch("#b_pw", () => { if (num("#b_pw") >= 0.1) B.power = num("#b_pw"); }); ch("#b_cl", () => { if (num("#b_cl") >= 0.1) B.clear = num("#b_cl"); });
  ch("#b_vd", () => { const [a, b] = $("#b_vd").value.split("/").map(parseFloat); if (a > 0.3) B.viaD = a; if (b > 0.15 && b < B.viaD) B.viaDrill = b; });
  ch("#b_pt", () => { B.pourTop = $("#b_pt").checked; }); ch("#b_pb", () => { B.pourBot = $("#b_pb").checked; }); ch("#b_pn", () => { B.pourNet = $("#b_pn").value; }); ch("#b_pr", () => { B.pourRoute = $("#b_pr").checked; });
  $$("#b_grid button").forEach(b => b.onclick = () => { B.grid = +b.dataset.g; commit("grid"); refreshPanels(); });
  $("#b_tpl").onclick = () => templatesDialog(false);
}
function renderNets(P, A) {
  if (!A.nets.length) { P.innerHTML = `<div class="empty">${ico("wire", "big")}<h3>No connections yet</h3><p>Use <b>Connect pins</b> (W) or <b>Smart connect</b> on a module to tell Wirie what should be wired together.</p></div>`; return; }
  P.innerHTML = `<div class="ph"><h3>Nets <em>${A.nets.length}</em></h3><p class="dim">A net is a group of pins that are wired together.</p></div><div class="netlist">${A.nets.map(n => `<div class="net" data-n="${esc(n.name)}"><i style="background:${n.color}"></i><b>${esc(n.name)}</b><span class="dim">${n.pads.length} pins</span>${n.islands > 1 ? `<button class="mini" data-r="${esc(n.name)}">${ico("route")}Route</button>` : `<span class="ok">✓</span>`}</div>`).join("")}</div>`;
  P.querySelectorAll(".net").forEach(el => { const n = A.netByName[el.dataset.n]; el.onmouseenter = () => { UI.hoverNet = n; UI.dirty = true; }; el.onmouseleave = () => { UI.hoverNet = null; UI.dirty = true; }; el.onclick = ev => { if (ev.target.closest("[data-r]")) return; el.title = n.pads.map(p => p.comp.ref + "." + p.name).join(", "); toast(`<b>${esc(n.name)}</b>: ${n.pads.map(p => p.comp.ref + "." + esc(p.name)).join(", ")}`); }; });
  P.querySelectorAll("[data-r]").forEach(b => b.onclick = () => runAutoroute([b.dataset.r]));
}
function renderCheck(P) {
  if (UI.drcStale) { UI.drc = runDRC(); UI.drcStale = false; UI.drcRan = true; }
  const errs = UI.drc.filter(d => d.sev === "err").length, warns = UI.drc.filter(d => d.sev === "warn").length;
  const ok = !errs && !warns && S.comps.length;
  P.innerHTML = `<div class="ph"><h3>${ico("check")}Design check</h3><p class="dim">Wirie checks spacing, shorts, unrouted nets and the board edge.</p></div>
    <div class="verdict ${ok ? "good" : errs ? "bad" : "warn"}">${ok ? `${ico("check")}<div><b>Ready to manufacture</b><span>No problems found.</span></div>` : errs ? `${ico("info")}<div><b>${errs} problem${errs > 1 ? "s" : ""} to fix</b><span>${warns} warning${warns === 1 ? "" : "s"}</span></div>` : `${ico("info")}<div><b>${warns} warning${warns > 1 ? "s" : ""}</b><span>Nothing serious, but worth a look.</span></div>`}</div>
    ${UI.drc.some(fixable) ? `<button class="btn primary wide" id="c_fix">${ico("magic")}Fix problems automatically</button>` : ""}
    <label class="chk"><input type="checkbox" id="c_auto" ${UI.autoFix ? "checked" : ""}><span>Auto-fix mode: fix problems every time the check runs</span></label>
    ${UI.fixReport ? `<div class="callout ${UI.fixReport.ok ? "good" : "warn"}">${ico(UI.fixReport.ok ? "check" : "info")}<div>${UI.fixReport.html}</div></div>` : ""}
    <label class="chk"><input type="checkbox" id="c_show" ${UI.drcShow ? "checked" : ""}><span>Circle problems on the board</span></label>
    <div class="drclist">${UI.drc.slice(0, 300).map((d, i) => `<button class="drc ${d.sev}" data-i="${i}"><i></i><span>${esc(d.msg)}</span></button>`).join("")}</div>`;
  $("#c_show").onchange = e => { UI.drcShow = e.target.checked; UI.dirty = true; };
  if ($("#c_fix")) $("#c_fix").onclick = () => runAutoFix();
  $("#c_auto").onchange = e => { UI.autoFix = e.target.checked; try { localStorage.setItem("wirie.autofix", UI.autoFix ? "1" : ""); } catch (_) {} if (UI.autoFix) runAutoFix(); };
  // auto-fix mode: fix whenever the check finds something new (not again on the result of its own fix)
  if (UI.autoFix && !UI.fixing && UNDO.last !== UI.fixedAt && UI.drc.some(fixable)) setTimeout(() => runAutoFix(), 0);
  P.querySelectorAll(".drc").forEach(b => b.onclick = () => { const d = UI.drc[+b.dataset.i]; focusAt(d.x, d.y); });
}
// problems the auto-fixer knows how to deal with
const fixable = d => ["clear", "short", "rat", "edge", "hole", "thin"].includes(d.code);
async function runAutoFix() {
  if (UI.fixing || ROUTER.running || !UI.drc.some(fixable)) return;
  UI.fixing = true;
  const m = modal(`<div class="routing"><div class="spinner"></div><h2>Fixing your board…</h2><p class="dim" id="fx_msg">Looking at the problems…</p><button class="btn ghost" id="fx_stop">Stop</button></div>`, "small");
  m.querySelector("#fx_stop").onclick = () => { ROUTER.cancel = true; UI.fixStop = true; };
  let r;
  try { r = await autoFix(t => { const el = $("#fx_msg"); if (el) el.textContent = t; }); }
  finally { closeModal(); UI.fixing = false; UI.fixStop = false; }
  commit("auto-fix"); UI.fixedAt = UNDO.last;
  UI.drc = r.left; UI.drcStale = false; UI.drcRan = true;
  const parts = [];
  if (r.moved.size) parts.push(`moved ${[...r.moved].slice(0, 6).join(", ")}${r.moved.size > 6 ? ` and ${r.moved.size - 6} more` : ""} to free spots`);
  if (r.tracks.size) parts.push(`redrew ${r.tracks.size} track${r.tracks.size > 1 ? "s" : ""}`);
  if (r.routed) parts.push(`routed ${r.routed} connection${r.routed > 1 ? "s" : ""}`);
  if (r.widened) parts.push(`widened ${r.widened} thin track${r.widened > 1 ? "s" : ""}`);
  const done = parts.length ? "Wirie " + parts.join(", ") + "." : "Nothing could be fixed automatically.";
  const rest = r.leftErr || r.leftWarn ? ` ${r.leftErr + r.leftWarn} left that need you (listed below).` : " No problems left.";
  const stuck = [...r.stuck].slice(0, 3).map(t => `<br><span class="dim">${esc(t)}</span>`).join("");
  UI.fixReport = { ok: !r.leftErr && !r.leftWarn, html: `<b>${parts.length ? "Auto-fix done" : "Auto-fix"}</b><br>${esc(done)}${esc(rest)}${stuck}<br><span class="dim">Not what you wanted? Press Ctrl+Z to undo the whole fix.</span>` };
  toast(`${ico("magic")} ${esc(done)}${esc(rest)}`, UI.fixReport.ok ? "good" : "warn");
  refreshPanels();
}
function focusAt(x, y) { const c = cv(); VIEW.zoom = Math.max(VIEW.zoom, 14); const [sx, sy] = [x * VIEW.zoom, y * VIEW.zoom]; VIEW.ox = c.clientWidth / 2 - (VIEW.flip ? (S.board.w - x) * VIEW.zoom : sx); VIEW.oy = c.clientHeight / 2 - sy; UI.focus = { x, y }; setTimeout(() => { UI.focus = null; UI.dirty = true; }, 1600); UI.dirty = true; }

// ---------- guide ----------
function guideSteps() {
  const A = analyze();
  return [
    ["Pick a board", "Choose a size or a template like a Pi HAT.", S.guide?.board || S.comps.length > 0, () => templatesDialog(false), "Templates"],
    ["Add parts", "Click parts in the library on the left.", S.comps.length >= 2, () => $("#libq").focus(), "Search parts"],
    ["Connect the pins", "Use Connect pins (W) or Smart connect.", S.conns.length > 0, () => { const m = S.comps.find(c => compFP(c).pinLabels && !compFP(c).mcu); if (m) smartConnect(m); else setTool("wire"); }, "Connect"],
    ["Route the tracks", "Let Wirie draw the copper for you.", S.conns.length > 0 && A.unrouted === 0, () => runAutoroute(), "Auto-route"],
    ["Check your design", "Find mistakes before you pay for boards.", UI.drcRan && !UI.drcStale && !UI.drc.some(d => d.sev === "err") && S.conns.length > 0, () => { UI.rtab = "check"; UI.drcStale = true; renderProps(); }, "Run check"],
    ["Order your boards", "Export Gerbers and upload to a PCB maker.", UI.exported, () => exportDialog(), "Export"],
  ];
}
function renderGuide() {
  const g = $("#guide"); if (UI.mode !== "pcb") { g.style.display = "none"; return; } g.style.display = "";
  const steps = guideSteps(); const done = steps.filter(s => s[2]).length; const next = steps.findIndex(s => !s[2]);
  g.classList.toggle("closed", !UI.guideOpen);
  g.innerHTML = `<button class="ghead" id="g_tog"><div class="ring" style="--p:${done / steps.length}"><span>${done}/${steps.length}</span></div><div><b>${done === steps.length ? "Board finished! 🎉" : "Getting started"}</b><small>${next >= 0 ? "Next: " + steps[next][0] : "Great work"}</small></div><svg class="chev" viewBox="0 0 24 24"><path d="M6 15l6-6 6 6" fill="none" stroke="currentColor" stroke-width="2"/></svg></button>
    <div class="gbody">${steps.map((s, i) => `<div class="gstep ${s[2] ? "done" : ""} ${i === next ? "next" : ""}"><i>${s[2] ? "✓" : i + 1}</i><div><b>${s[0]}</b><small>${s[1]}</small></div>${!s[2] ? `<button class="mini" data-g="${i}">${s[4]}</button>` : ""}</div>`).join("")}</div>`;
  $("#g_tog").onclick = () => { UI.guideOpen = !UI.guideOpen; renderGuide(); };
  g.querySelectorAll("[data-g]").forEach(b => b.onclick = () => steps[+b.dataset.g][3]());
}

// ---------- smart connect ----------
function smartConnect(mod) {
  const A = analyze();
  const mcus = S.comps.filter(c => compFP(c).mcu && c.id !== mod.id);
  const fpM = compFP(mod);
  let module = mod, mcu = mcus[0];
  if (fpM.mcu) { const others = S.comps.filter(c => c.id !== mod.id && compFP(c).pinLabels && !compFP(c).mcu); if (!others.length) { toast("Add a sensor or module first, then Smart connect wires it to this board.", "warn"); return; } module = others[0]; mcu = mod; }
  if (!mcu) { toast("Add a microcontroller (Pico, Arduino, ESP32, Pi header…) first.", "warn"); return; }
  const draw = () => {
    const fpm = compFP(module), fpc = compFP(mcu);
    const mcuPads = padsOf(mcu); const names = [...new Set(mcuPads.map(p => p.name))];
    const rail = UI.scRail || (mcuPads.some(p => p.func === "PWR3") ? "3V3" : "5V");
    const usedNames = new Set(mcuPads.filter(p => A.netOfPad[p.key]).map(p => p.name));
    const byFunc = f => mcuPads.find(p => p.func === f);
    const gpioRe = /^(GP\d+|D\d+|GPIO\d+|IO\d+|[ABC]\d+|\d+)$/;
    const gpioPool = (fpc.gpio || mcuPads.filter(p => gpioRe.test(p.name) && (!p.func || p.func === "ADC")).map(p => p.name)).filter(n => !usedNames.has(n));
    const take = pref => { const i = pref ? gpioPool.findIndex(n => mcuPads.find(p => p.name === n && p.func === pref)) : -1; const k = i >= 0 ? i : 0; return gpioPool.length ? gpioPool.splice(k, 1)[0] : ""; };
    const railPad = rail === "3V3" ? (byFunc("PWR3") || mcuPads.find(p => /^3\.?3V?$|3V3/.test(p.name))) : (byFunc("PWR5") || mcuPads.find(p => /^5V$|VBUS|VIN/.test(p.name)));
    const rows = padsOf(module).map(p => {
      const f = p.func; let tgt = "";
      if (f === "GND") tgt = "GND"; else if (f === "PWR") tgt = railPad ? railPad.name : ""; else if (f === "PWR5") tgt = (byFunc("PWR5") || {}).name || ""; else if (f === "PWR3") tgt = (byFunc("PWR3") || {}).name || "";
      else if (["SDA", "SCL", "MOSI", "MISO", "SCK"].includes(f)) tgt = (byFunc(f) || {}).name || take();
      else if (f === "CS") { const c = byFunc("CS"); tgt = c && !usedNames.has(c.name) ? c.name : take(); }
      else if (f === "TX") tgt = (byFunc("RX") || {}).name || take(); else if (f === "RX") tgt = (byFunc("TX") || {}).name || take();
      else if (f === "ADC") tgt = take("ADC"); else if (f === "SIG" || f === "DIN") tgt = take(); else tgt = "";
      if (!names.includes(tgt)) tgt = "";
      if (A.netOfPad[p.key]) tgt = UI.scKeep?.[p.key] ?? "";
      return { p, tgt };
    });
    const m = modal(`<div class="mhead">${ico("magic", "big")}<div><h2>Smart connect</h2><p class="dim">Wirie matched the pins by their names. Change anything you like, then press Connect.</p></div><button class="x" data-close>${ico("close")}</button></div>
      <div class="grid3"><label class="fld"><span>Module</span><div class="inp"><select id="sc_mod">${S.comps.filter(c => compFP(c).pinLabels && c.id !== mcu.id).map(c => `<option value="${c.id}" ${c.id === module.id ? "selected" : ""}>${esc(c.ref)} — ${esc(compFP(c).name)}</option>`).join("")}</select></div></label>
      <label class="fld"><span>Connect to</span><div class="inp"><select id="sc_mcu">${S.comps.filter(c => compFP(c).mcu && c.id !== module.id).map(c => `<option value="${c.id}" ${c.id === mcu.id ? "selected" : ""}>${esc(c.ref)} — ${esc(compFP(c).name)}</option>`).join("")}</select></div></label>
      <label class="fld"><span>Power the module from</span><div class="inp"><select id="sc_rail"><option ${rail === "3V3" ? "selected" : ""}>3V3</option><option ${rail === "5V" ? "selected" : ""}>5V</option></select></div></label></div>
      <div class="sctable">${rows.map((r, i) => `<div class="scrow"><b>${esc(r.p.name)}</b><span class="tag">${r.p.func ? r.p.func.replace(/^PWR\d?$/, "power").toLowerCase() : "—"}</span><svg viewBox="0 0 24 24" class="arrow"><path d="M4 12h14M13 6l6 6-6 6" fill="none" stroke="currentColor" stroke-width="2"/></svg><div class="inp"><select data-i="${i}"><option value="">(leave unconnected)</option>${names.map(n => `<option ${n === r.tgt ? "selected" : ""}>${esc(n)}</option>`).join("")}</select></div>${A.netOfPad[r.p.key] ? `<span class="dim">already: ${esc(A.netOfPad[r.p.key].name)}</span>` : ""}</div>`).join("")}</div>
      ${/5V/i.test(fpm.desc || "") && rail === "3V3" ? `<div class="callout warn">${ico("info")}${esc(fpm.desc)}</div>` : ""}
      <div class="mfoot"><span class="dim">${esc(fpc.name)}: ${fpc.desc ? esc(fpc.desc.split(".")[0]) : ""}</span><button class="btn ghost" data-close>Cancel</button><button class="btn primary" id="sc_go">${ico("wire")}Connect ${rows.filter(r => r.tgt).length} pins</button></div>`, "wide");
    m.querySelector("#sc_mod").onchange = e => { module = S.comps.find(c => c.id === e.target.value); draw(); };
    m.querySelector("#sc_mcu").onchange = e => { mcu = S.comps.find(c => c.id === e.target.value); draw(); };
    m.querySelector("#sc_rail").onchange = e => { UI.scRail = e.target.value; draw(); };
    m.querySelectorAll("select[data-i]").forEach(s => s.onchange = () => { rows[+s.dataset.i].tgt = s.value; m.querySelector("#sc_go").innerHTML = `${ico("wire")}Connect ${rows.filter(r => r.tgt).length} pins`; });
    m.querySelector("#sc_go").onclick = () => {
      let n = 0;
      for (const r of rows) { if (!r.tgt) continue; const tp = mcuPads.find(p => p.name === r.tgt); if (tp && addConn(r.p.key, tp.key)) n++; }
      closeModal(); commit("smart connect"); toast(`${ico("magic")} Connected <b>${n}</b> pins between ${module.ref} and ${mcu.ref}.`, "good");
    };
  };
  draw();
}

// ---------- autoroute ----------
async function runAutoroute(only) {
  const A = analyze();
  if (!A.nets.length) { toast("Connect some pins first (Connect pins tool or Smart connect) — then Wirie can route them.", "warn"); return; }
  if (!A.unrouted) { toast("Everything is already routed ✓", "good"); return; }
  const m = modal(`<div class="routing"><div class="spinner"></div><h2>Routing your board…</h2><p class="dim" id="rt_msg">Finding paths for ${A.unrouted} connections</p><div class="bar"><i id="rt_bar"></i></div><button class="btn ghost" id="rt_stop">Stop</button></div>`, "small");
  m.querySelector("#rt_stop").onclick = () => { ROUTER.cancel = true; };
  const before = A.unrouted;
  const res = await autoroute((pass, f, nm) => { const b = $("#rt_bar"); if (b) b.style.width = Math.round(f * 100) + "%"; const t = $("#rt_msg"); if (t) t.textContent = `${pass ? "Retry " + pass + " · " : ""}routing ${nm}`; }, only);
  closeModal(); commit("autoroute");
  const left = analyze().unrouted;
  if (!left) toast(`${ico("check")} All ${before} connections routed!`, "good");
  else toast(`Routed ${before - left} of ${before}. ${left} left — try moving parts apart, turning on ground fill, or draw those by hand.`, "warn");
  refreshPanels();
}

// ---------- dialogs ----------
function templatesDialog(welcome) {
  const m = modal(`<div class="mhead">${welcome ? LOGO : ico("place", "big")}<div><h2>${welcome ? "Welcome to Wirie" : "Board templates"}</h2><p class="dim">${welcome ? "Design your own circuit boards — no experience needed. Pick a starting point:" : "Pick a board shape. Parts already on your board are kept."}</p></div>${welcome ? "" : `<button class="x" data-close>${ico("close")}</button>`}</div>
    <div class="tplgrid">${welcome ? `<button class="tpl hero" data-ex="1"><div class="tplart ex">${ico("star", "big")}</div><b>Example project</b><small>A Pico with a screen, sensor, button and LED — ready to auto-route.</small></button>` : ""}${TEMPLATES.map(t => `<button class="tpl" data-t="${t.id}"><div class="tplart"><div class="tb2" style="aspect-ratio:${t.board.w}/${t.board.h};border-radius:${t.board.round ? "50%" : "6px"}"></div></div><b>${esc(t.name)}</b><small>${esc(t.desc)}</small></button>`).join("")}</div>
    ${welcome ? `<div class="mfoot"><button class="btn ghost" id="w_open">${ico("open")}Open a saved project</button><span class="credit">Wirie ${APP.version} · made by <b>${APP.author}</b></span></div>` : ""}`, "wide");
  m.querySelectorAll("[data-t]").forEach(b => b.onclick = () => { applyTemplate(TEMPLATES.find(t => t.id === b.dataset.t), welcome); closeModal(); });
  const ex = m.querySelector("[data-ex]"); if (ex) ex.onclick = () => { loadExample(); closeModal(); };
  const wo = m.querySelector("#w_open"); if (wo) wo.onclick = () => openDialog();
}
function applyTemplate(t, fresh) {
  if (fresh) { S = newProject(); S.name = t.name.split(" (")[0]; }
  Object.assign(S.board, { w: t.board.w, h: t.board.h, r: t.board.r || 0, round: !!t.board.round });
  S.guide = { board: true };
  for (const p of t.parts || []) addComp(p.fp, p.x, p.y, { rot: p.rot || 0 });
  if (fresh) resetHistory();
  commit("template"); fitView(); refreshPanels();
}
function loadExample() {
  S = newProject(); S.name = "Pico weather station"; S.guide = { board: true };
  Object.assign(S.board, { w: 80, h: 60, r: 3 });
  const pico = addComp("pico", 22, 30);
  const oled = addComp("oled096", 60, 14, { rot: 0 });
  const bme = addComp("bme280", 60, 42);
  const btn = addComp("sw6x6", 42, 50);
  const led = addComp("led5", 42, 8, { value: "Green" });
  const r = addComp("r_axial_7.62", 42, 16, { value: "330" });
  for (const [x, y] of [[4, 4], [76, 4], [4, 56], [76, 56]]) addComp("mh_M3", x, y);
  S.texts.push({ id: uid(), x: 60, y: 56, text: "WEATHER v1", size: 1.6, layer: "top", rot: 0 });
  const pk = (c, name) => padsOf(c).find(p => p.name === name).key;
  const smart = (m) => { const mp = padsOf(pico); for (const p of padsOf(m)) { const f = p.func; let t = f === "GND" ? "GND" : f === "PWR" ? "3V3" : f === "SDA" ? "GP4" : f === "SCL" ? "GP5" : null; if (t) addConn(p.key, mp.find(q => q.name === t).key); } };
  smart(oled); smart(bme);
  addConn(pk(btn, "A"), pk(pico, "GP15")); addConn(pk(btn, "B"), pk(pico, "GND"));
  addConn(pk(r, "1"), pk(pico, "GP14")); addConn(pk(r, "2"), pk(led, "A")); addConn(pk(led, "K"), pk(pico, "GND"));
  resetHistory(); commit("example"); fitView(); refreshPanels();
  toast(`${ico("star")} Example loaded! Press <b>Auto-route</b> at the top to draw all the tracks.`, "good");
}
async function openDialog() {
  let list = { items: [], folder: "" };
  try { list = await api("/api/list"); } catch (e) { }
  const m = modal(`<div class="mhead">${ico("open", "big")}<div><h2>Open a project</h2><p class="dim">Saved in ${esc(list.folder || "Documents\\Wirie\\Projects")}</p></div><button class="x" data-close>${ico("close")}</button></div>
    <div class="filelist">${list.items.length ? list.items.map(it => `<button class="file" data-n="${esc(it.name)}">${ico("board")}<b>${esc(it.name)}</b><span class="dim">${it.modified}</span></button>`).join("") : `<div class="empty"><p>No saved projects yet.</p></div>`}</div>
    <div class="mfoot"><label class="btn ghost">${ico("folder")}Import .wirie file…<input type="file" accept=".wirie,.json" id="o_file" hidden></label><button class="btn ghost" data-close>Cancel</button></div>`);
  m.querySelectorAll("[data-n]").forEach(b => b.onclick = async () => { try { const p = await api("/api/load?name=" + encodeURIComponent(b.dataset.n)); loadProject(p); closeModal(); toast(`Opened <b>${esc(p.name)}</b>`); } catch (e) { toast("Couldn't open: " + esc(e.message), "bad"); } });
  m.querySelector("#o_file").onchange = e => { const f = e.target.files[0]; if (!f) return; f.text().then(t => { try { loadProject(JSON.parse(t)); closeModal(); } catch (er) { toast("That file isn't a Wirie project.", "bad"); } }); };
}
function loadProject(p) {
  if (!p || p.app !== "wirie") throw new Error("not a Wirie project");
  registerProjectParts(p);
  const base = newProject(); S = Object.assign(base, p); S.board = Object.assign(newProject().board, p.board || {}); S.strip = Object.assign(newProject().strip, p.strip || {});
  S.comps = S.comps.filter(c => LIB_BY_ID[c.fp]);
  UI.sel.clear(); resetHistory(); invalidate(); fitView(); refreshPanels();
}
async function saveProject(asNew) {
  const go = async name => { S.name = name; embedUserParts(); try { const r = await api("/api/save", { name, data: S }); UI.lastSaved = UNDO.last; toast(`${ico("save")} Saved <b>${esc(r.name)}</b>`, "good"); $("#projname").value = S.name; } catch (e) { downloadFile(name + ".wirie", JSON.stringify(S)); toast("Saved as a download (app folder not reachable).", "warn"); } };
  if (asNew || !S.name || S.name === "Untitled board") askText("Save project", "Project name", S.name === "Untitled board" ? "" : S.name, go); else go(S.name);
}
function downloadFile(name, content, type = "application/octet-stream") { const b = new Blob([content], { type }); const a = document.createElement("a"); a.href = URL.createObjectURL(b); a.download = name; a.click(); setTimeout(() => URL.revokeObjectURL(a.href), 2000); }
function newDialog() {
  const m = modal(`<h2>Start a new board?</h2><p class="dim">Anything you haven't saved will be lost (your last session is still auto-saved until you change something).</p><div class="mfoot"><button class="btn ghost" data-close>Cancel</button><button class="btn" id="n_save">${ico("save")}Save first</button><button class="btn primary" id="n_go">New board</button></div>`, "small");
  m.querySelector("#n_go").onclick = () => { closeModal(); templatesDialog(true); };
  m.querySelector("#n_save").onclick = () => { closeModal(); saveProject(); };
}
function exportDialog() {
  const drc = runDRC(); UI.drc = drc; UI.drcStale = false; UI.drcRan = true;
  const errs = drc.filter(d => d.sev === "err"), A = analyze(), ver = S.comps.filter(c => compFP(c).verify);
  const m = modal(`<div class="mhead">${ico("export", "big")}<div><h2>Get your boards made</h2><p class="dim">Wirie creates the standard Gerber + drill files that every PCB factory accepts.</p></div><button class="x" data-close>${ico("close")}</button></div>
    <div class="checks">
      <div class="ck ${errs.length ? "bad" : "good"}">${errs.length ? ico("info") : ico("check")}<span>${errs.length ? `${errs.length} design problem${errs.length > 1 ? "s" : ""} — <a href="#" id="x_show">show me</a>` : "Design check passed"}</span></div>
      <div class="ck ${A.unrouted ? "warn" : "good"}">${A.unrouted ? ico("info") : ico("check")}<span>${A.unrouted ? `${A.unrouted} connections not routed yet` : "All connections routed"}</span></div>
      <div class="ck ${ver.length ? "warn" : "good"}">${ver.length ? ico("info") : ico("check")}<span>${ver.length ? `Double-check pin positions of ${ver.slice(0, 6).map(c => c.ref).join(", ")}${ver.length > 6 ? ` and ${ver.length - 6} more` : ""}` : "All footprints are standard"}</span></div>
    </div>
    <div class="exgrid">
      <button class="excard primary" id="x_zip">${ico("export", "big")}<b>Gerber ZIP for PCB makers</b><small>Upload to JLCPCB, PCBWay, OSH Park, Aisler… (2-layer, 1.6mm)</small></button>
      <button class="excard" id="x_etch">${ico("print", "big")}<b>Print for home etching</b><small>1:1 copper artwork for toner transfer or UV</small></button>
      <button class="excard" id="x_png">${ico("eye", "big")}<b>Board picture</b><small>A realistic PNG render to show off</small></button>
      <button class="excard makie" id="x_makie">${ico("box", "big")}<b>Case data for Makie</b><small>Board shape, holes, 3D part sizes & connector openings — Makie builds the enclosure</small></button>
      <button class="excard" id="x_bom">${ico("chip", "big")}<b>Parts list</b><small>Everything to buy, grouped, with quantities: copy it or save a CSV for Excel</small></button>
    </div><div id="x_res"></div>
    <div class="mfoot"><span class="credit">Files are tagged “Made with Wirie by ${APP.author}”</span><button class="btn ghost" data-close>Close</button></div>`, "wide");
  const sh = m.querySelector("#x_show"); if (sh) sh.onclick = e => { e.preventDefault(); closeModal(); UI.rtab = "check"; UI.drcShow = true; renderProps(); };
  m.querySelector("#x_zip").onclick = async () => {
    const files = buildManufacturingFiles();
    try { const r = await api("/api/export", { name: S.name || "board", files }); UI.exported = true; renderGuide(); m.querySelector("#x_res").innerHTML = `<div class="callout good">${ico("check")}<div><b>Saved ${Object.keys(files).length} files</b><br><span class="dim">${esc(r.path)}</span></div><button class="btn sm" id="x_rev">${ico("folder")}Show in folder</button></div>`; m.querySelector("#x_rev").onclick = () => api("/api/reveal", { path: r.path }); }
    catch (e) { toast("Couldn't write the ZIP: " + esc(e.message), "bad"); }
  };
  m.querySelector("#x_makie").onclick = async () => {
    try {
      const data = makieExport(); const txt = JSON.stringify(data, null, 1);
      const r = await api("/api/savefile", { name: S.name || "board", ext: ".makie.json", content: txt });
      m.querySelector("#x_res").innerHTML = `<div class="callout good">${ico("check")}<div><b>Case data saved for Makie</b><br><span class="dim">${data.parts.length} parts · ${data.openings.length} edge openings · ${data.lidFeatures.length} lid cut-outs · ${data.mountingHoles.length} mounting holes</span><br><span class="dim">${esc(r.path)}</span></div><button class="btn sm" id="x_rev">${ico("folder")}Show in folder</button></div>`;
      m.querySelector("#x_rev").onclick = () => api("/api/reveal", { path: r.path });
    } catch (e) { downloadFile((S.name || "board") + ".makie.json", JSON.stringify(makieExport(), null, 1), "application/json"); toast("Saved as a download", "warn"); }
  };
  m.querySelector("#x_etch").onclick = () => etchDialog();
  m.querySelector("#x_png").onclick = () => savePicture();
  m.querySelector("#x_bom").onclick = () => { closeModal(); partsListDialog(false); };
}
function etchDialog() {
  const m = modal(`<div class="mhead">${ico("print", "big")}<div><h2>Print for home etching</h2><p class="dim">Prints at exactly 1:1. In the print dialog set <b>Scale: 100% / Actual size</b> and turn off "fit to page".</p></div><button class="x" data-close>${ico("close")}</button></div>
    <div class="exgrid">
      <button class="excard" data-l="bottom" data-m="0"><b>Bottom copper</b><small>Ready for toner transfer (prints the right way round for single-sided boards)</small></button>
      <button class="excard" data-l="top" data-m="1"><b>Top copper</b><small>Mirrored for toner transfer</small></button>
      <button class="excard" data-l="top" data-m="0" data-w="silk"><b>Top silkscreen</b><small>Part outlines, as seen from above</small></button>
      <button class="excard" data-l="top" data-m="0" data-w="fit"><b>Paper fit test</b><small>Top view — check parts fit before etching</small></button>
    </div><div class="mfoot"><button class="btn ghost" data-close>Close</button></div>`, "wide");
  m.querySelectorAll("[data-l]").forEach(b => b.onclick = () => {
    const what = b.dataset.w === "silk" ? "silk" : "copper";
    let svg = svgLayer(b.dataset.l, b.dataset.m === "1", what);
    if (b.dataset.w === "fit") svg = svgLayer("top", false, "silk").replace("</g></svg>", svgLayer("top", false, "copper").replace(/^.*?<g[^>]*>/, "").replace("</g></svg>", "") + "</g></svg>");
    const w = window.open("", "_blank"); if (!w) { downloadFile((S.name || "board") + `-${b.dataset.l}.svg`, svg, "image/svg+xml"); return; }
    w.document.write(`<!doctype html><title>${esc(S.name)} — print 1:1</title><style>@page{margin:10mm}body{margin:0;font:12px Segoe UI,sans-serif}p{color:#666}</style><p>${esc(S.name)} · ${b.querySelector("b").textContent} · print at 100% · Made with Wirie by ${APP.author}</p>${svg}<script>setTimeout(()=>print(),300)<\/script>`); w.document.close();
  });
}
function savePicture() {
  const c = document.createElement("canvas"); const B = S.board; const sc = Math.min(4000 / B.w, 3000 / B.h, 40);
  c.width = Math.round((B.w + 16) * sc); c.height = Math.round((B.h + 16) * sc);
  const keep = { ...VIEW, show: { ...VIEW.show } }; const keepSel = UI.sel; UI.sel = new Set();
  Object.assign(VIEW, { zoom: sc, ox: 8 * sc, oy: 8 * sc, mode: "real", flip: false });
  const d = window.devicePixelRatio; Object.defineProperty(window, "devicePixelRatio", { value: 1, configurable: true });
  drawPCB(c);
  Object.defineProperty(window, "devicePixelRatio", { value: d, configurable: true });
  Object.assign(VIEW, keep); UI.sel = keepSel; UI.dirty = true;
  c.toBlob(b => { const a = document.createElement("a"); a.href = URL.createObjectURL(b); a.download = (S.name || "board") + ".png"; a.click(); });
  toast("Board picture saved to your Downloads");
}
function showHelp() {
  modal(`<div class="mhead">${ico("help", "big")}<div><h2>How Wirie works</h2><p class="dim">Four ideas and you're designing boards.</p></div><button class="x" data-close>${ico("close")}</button></div>
    <div class="helpgrid">
      <div><h4>1 · Parts</h4><p>Click any part in the library, then click on the board. Modules like the Pico or ESP32 come with every pin already labelled.</p></div>
      <div><h4>2 · Connections</h4><p>Tell Wirie what's wired to what: click a pin, then another (Connect pins, W). For sensors, <b>Smart connect</b> does it all at once.</p></div>
      <div><h4>3 · Tracks</h4><p>Yellow dashed lines are connections without copper yet. Press <b>Auto-route</b>, or draw tracks yourself (T). Red = top, blue = bottom.</p></div>
      <div><h4>4 · Make it</h4><p>Run the <b>Check</b>, then <b>Export</b> a Gerber ZIP for a PCB factory — or plan it on stripboard in the <b>Boardie</b> tab.</p></div>
    </div>
    <h4>Keyboard shortcuts</h4><div class="keys">${[["V", "Select"], ["W", "Connect pins"], ["T", "Track"], ["I", "Via"], ["X", "Text"], ["M", "Measure"], ["R", "Rotate 90°"], ["Shift+R", "Rotate 15°"], ["F", "Flip to back"], ["L / Tab", "Swap copper side"], ["V (while routing)", "Via + swap side"], ["G", "Change grid"], ["Del", "Delete"], ["Ctrl+Z / Y", "Undo / redo"], ["Ctrl+D", "Duplicate"], ["Ctrl+S", "Save"], ["Space / right-drag", "Pan"], ["Wheel", "Zoom"], ["Home", "Fit"], ["Ctrl+Shift+P", "Create a part"]].map(k => `<div><kbd>${k[0]}</kbd><span>${k[1]}</span></div>`).join("")}</div>`, "wide");
}
function showAbout() {
  modal(`<div class="about">${LOGO}<h1>Wirie</h1><p class="ver">Version ${APP.version}</p><p class="by">Created by <b>${APP.author}</b></p><p class="dim">A free, beginner-friendly circuit board designer — with Boardie (stripboard planner), Calcie (electronics calculators), Pinnie (interactive pinouts) and Powerie (battery planner) built in. Part of the ${APP.author} maker tool suite.</p>
    <div class="aboutstats"><div><b>${LIB.length}</b><span>parts</span></div><div><b>${TEMPLATES.length}</b><span>templates</span></div><div><b>12</b><span>calculators</span></div></div>
    <p class="note">Footprints marked <span class="warnb">!</span> vary between manufacturers — always measure your actual module before ordering boards.</p><button class="btn primary" data-close>Let's build something</button></div>`, "small");
}
function applyTraceWidth(kind, w) {
  if (!(w > 0)) return; const v = Math.max(0.15, Math.ceil(w * 20) / 20);
  if (kind === "track") S.board.track = v; else S.board.power = v;
  commit("trace width"); toast(`${kind === "track" ? "Signal" : "Power"} tracks set to <b>${v} mm</b> for new routing.`, "good");
}

// ---------- Boardie panel ----------
function renderStripPanel() {
  const P = $("#props"); const st = S.strip; const SA = stripAnalyze();
  $$("#rtabs button").forEach(b => b.classList.remove("on"));
  P.innerHTML = `<div class="ph"><h3>${ico("strip")}Boardie</h3><p class="dim">Build your circuit on stripboard or perfboard. Boardie uses the same parts & connections as your PCB.</p></div>
    <div class="seg" id="s_kind"><button data-k="strip" class="${st.kind === "strip" ? "on" : ""}">Stripboard</button><button data-k="perf" class="${st.kind === "perf" ? "on" : ""}">Perfboard</button></div>
    <div class="grid2">${fieldHTML("s_cols", "Holes across", st.cols, "", "number")}${fieldHTML("s_rows", "Strips (rows)", st.rows, "", "number")}</div>
    <button class="btn primary wide" id="s_all">${ico("magic")}Do it all — place, cut & link</button>
    <div class="btnrow three"><button class="btn sm" id="s_place">${ico("place")}Place</button><button class="btn sm" id="s_cuts" ${st.kind !== "strip" ? "disabled" : ""}>${ico("cut")}Cuts</button><button class="btn sm" id="s_links">${ico("link")}Links</button></div>
    <h4>View</h4><div class="seg" id="s_view"><button data-v="comp" class="${SV.view === "comp" ? "on" : ""}">Parts side</button><button data-v="xray" class="${SV.view === "xray" ? "on" : ""}">X-ray</button><button data-v="copper" class="${SV.view === "copper" ? "on" : ""}">Copper side</button></div>
    <div class="verdict ${SA.shorts.length ? "bad" : SA.missing.length || SA.unplaced.length ? "warn" : SA.placed.length ? "good" : ""}">${SA.shorts.length ? `${ico("info")}<div><b>${SA.shorts.length} short circuit${SA.shorts.length > 1 ? "s" : ""}</b><span>Add cuts or press Do it all</span></div>` : SA.missing.length ? `${ico("info")}<div><b>${SA.missing.length} net${SA.missing.length > 1 ? "s" : ""} not joined yet</b><span>Add wire links or press Do it all</span></div>` : SA.placed.length ? `${ico("check")}<div><b>Ready to solder</b><span>${SA.cutSet.size} cuts · ${st.links.length + st.autoLinks.length} links</span></div>` : `${ico("info")}<div><b>Nothing placed yet</b><span>Press Do it all to start</span></div>`}</div>
    ${SA.unplaced.length ? `<h4>Waiting to be placed <em>${SA.unplaced.length}</em></h4><div class="tray">${SA.unplaced.map(c => `<button class="trayitem" data-p="${c.id}"><b>${esc(c.ref)}</b><span>${esc(compFP(c).name)}</span></button>`).join("")}</div>` : ""}
    ${SA.notStrip.length ? `<div class="callout warn">${ico("info")}<span>${SA.notStrip.map(c => esc(c.ref)).join(", ")} ${SA.notStrip.length > 1 ? "are" : "is"} surface-mount and can't go on stripboard. Swap for through-hole versions.</span></div>` : ""}
    ${SA.offgrid.length ? `<div class="callout warn">${ico("info")}<span>${[...new Set(SA.offgrid.map(p => p.comp.ref))].join(", ")} ha${SA.offgrid.length > 1 ? "ve" : "s"} pins that don't land on the 0.1in grid — bend the legs or pick another footprint.</span></div>` : ""}
    ${SA.shorts.length ? `<h4>Shorts</h4><div class="drclist">${SA.shorts.map(s => `<div class="drc err"><i></i><span>${s.names.map(esc).join(" ↔ ")}</span></div>`).join("")}</div>` : ""}
    ${SA.missing.length ? `<h4>Not joined</h4><div class="drclist">${SA.missing.map(m => `<div class="drc warn"><i></i><span><span class="pill" style="--c:${m.net.color}">${esc(m.net.name)}</span> in ${m.groups.length} separate pieces</span></div>`).join("")}</div>` : ""}
    <div class="btnrow"><button class="btn" id="s_print">${ico("print")}Print build plan</button><button class="btn ghost" id="s_clear">${ico("trash")}Clear</button></div>`;
  $$("#s_kind button").forEach(b => b.onclick = () => { st.kind = b.dataset.k; if (st.kind === "perf") st.autoCuts = []; commit("kind"); refreshPanels(); });
  $("#s_cols").onchange = () => { st.cols = Math.max(5, Math.min(120, parseInt($("#s_cols").value) || st.cols)); commit("cols"); fitView(); refreshPanels(); };
  $("#s_rows").onchange = () => { st.rows = Math.max(5, Math.min(100, parseInt($("#s_rows").value) || st.rows)); commit("rows"); fitView(); refreshPanels(); };
  $("#s_all").onclick = () => { const before = SA.unplaced.length; stripDoAll(); commit("strip all"); const a = stripAnalyze(); refreshPanels(); toast(a.shorts.length || a.missing.length ? `Done — ${a.unplaced.length ? a.unplaced.length + " parts didn't fit (make the board bigger). " : ""}${a.shorts.length} shorts, ${a.missing.length} unjoined left.` : `${ico("check")} Placed, cut and linked — ${a.cutSet.size} cuts and ${S.strip.autoLinks.length + S.strip.links.length} wire links.`, a.shorts.length || a.missing.length ? "warn" : "good"); };
  $("#s_place").onclick = () => { const n = stripAutoPlace(false); commit("place"); refreshPanels(); toast(n ? `Placed ${n} parts` : "Nothing new to place — drag parts to move them."); };
  $("#s_cuts").onclick = () => { const n = stripAutoCuts(); commit("cuts"); refreshPanels(); toast(`${n} cuts suggested`); };
  $("#s_links").onclick = () => { const n = stripAutoLinks(); commit("links"); refreshPanels(); toast(`${n} wire links added`); };
  $$("#s_view button").forEach(b => b.onclick = () => { SV.view = b.dataset.v; UI.dirty = true; refreshPanels(); });
  $("#s_print").onclick = () => { const w = window.open("", "_blank"); const html = stripPlanHTML(); if (w) { w.document.write(html); w.document.close(); } else downloadFile((S.name || "board") + "-stripboard.html", html, "text/html"); };
  $("#s_clear").onclick = () => { st.place = {}; st.cuts = []; st.autoCuts = []; st.links = []; st.autoLinks = []; commit("strip clear"); refreshPanels(); };
  P.querySelectorAll("[data-p]").forEach(b => b.onclick = () => { const c = S.comps.find(q => q.id === b.dataset.p); SV.placing = { id: c.id, rot: stripRotFor(c), x: 0, y: 0 }; toast(`Click on the board to place ${c.ref}. R rotates.`); });
}

// ---------- mode switching ----------
function switchMode(m) {
  UI.mode = m; UI.placing = null; SV.placing = null; UI.track = null; UI.wireFrom = null;
  $$("#modes button").forEach(b => b.classList.toggle("on", b.dataset.m === m));
  document.body.dataset.mode = m;
  document.body.classList.toggle("page", ["calc", "pinnie", "power", "keeb"].includes(m));
  if (m === "calc") { if (!$("#calc").dataset.built) { buildCalcie($("#calc")); $("#calc").dataset.built = 1; } }
  else if (m === "pinnie") { if (!$("#pinnie").dataset.built) { buildPinnie($("#pinnie")); $("#pinnie").dataset.built = 1; } }
  else if (m === "power") { buildPowerie($("#power")); }
  else if (m === "keeb") { if (!$("#keeb").dataset.built) { buildKeebie($("#keeb")); $("#keeb").dataset.built = 1; } setTimeout(kbPage, 0); }
  else { buildTools(); resize(); if (m === "strip" && !SV.fitted) { SV.fitted = true; fitView(); } }
  refreshPanels(); UI.dirty = true;
}

// ---------- boot ----------
function boot() {
  document.title = `Wirie — by ${APP.author}`;
  $("#brand").innerHTML = `${LOGO}<div><b>Wirie</b><small>by ${APP.author}</small></div>`;
  $("#modes").innerHTML = `<button data-m="pcb" class="on">${ico("board")}PCB</button><button data-m="strip">${ico("strip")}Boardie</button><button data-m="calc">${ico("calc")}Calcie</button><button data-m="pinnie">${ico("chip")}Pinnie</button><button data-m="power">${ico("bolt")}Powerie</button><button data-m="keeb">${ico("keyboard")}Keebie</button>`;
  $$("#modes button").forEach(b => b.onclick = () => switchMode(b.dataset.m));
  $("#filebtns").innerHTML = `<button class="btn ghost icon" id="f_new" data-tip="New">${ico("new")}</button><button class="btn ghost icon" id="f_open" data-tip="Open (Ctrl+O)">${ico("open")}</button><button class="btn ghost icon" id="f_save" data-tip="Save (Ctrl+S)">${ico("save")}</button><button class="btn primary" id="f_export">${ico("export")}Export</button><button class="btn ghost icon" id="f_help" data-tip="Help (?)">${ico("help")}</button><button class="btn ghost icon" id="f_about" data-tip="About Wirie">${ico("info")}</button>`;
  $("#f_new").onclick = newDialog; $("#f_open").onclick = openDialog; $("#f_save").onclick = () => saveProject(); $("#f_export").onclick = exportDialog; $("#f_help").onclick = showHelp; $("#f_about").onclick = showAbout;
  $("#projname").onchange = e => { S.name = e.target.value.trim() || "Untitled board"; commit("rename"); };
  $("#topbar2").innerHTML = `<div class="seg" id="layerSw" data-tip="Active copper side (L)"><button data-l="top" class="on"><i class="dot top"></i>Top</button><button data-l="bottom"><i class="dot bot"></i>Bottom</button></div>
    <div class="seg" id="viewSw"><button data-v="design" class="on">Design</button><button data-v="real">Real</button><button data-v="flip" data-tip="Look at the back of the board">Flip</button></div>
    <span class="grow"></span>
    <button class="btn" id="a_route">${ico("route")}Auto-route</button><button class="btn ghost icon" id="a_unroute" data-tip="Remove auto-routed tracks">${ico("trash")}</button><button class="btn" id="a_check">${ico("check")}Check</button>`;
  $$("#layerSw button").forEach(b => b.onclick = () => setLayer(b.dataset.l));
  $$("#viewSw button").forEach(b => b.onclick = () => { if (b.dataset.v === "flip") { VIEW.flip = !VIEW.flip; b.classList.toggle("on", VIEW.flip); if (VIEW.flip) setLayer("bottom"); else setLayer("top"); } else { VIEW.mode = b.dataset.v; $$("#viewSw [data-v=design],#viewSw [data-v=real]").forEach(x => x.classList.toggle("on", x.dataset.v === VIEW.mode)); } UI.dirty = true; });
  $("#a_route").onclick = () => runAutoroute(); $("#a_unroute").onclick = () => { clearAutoRoutes(); commit("unroute"); toast("Removed auto-routed tracks"); };
  $("#a_check").onclick = () => { UI.rtab = "check"; UI.drcStale = true; renderProps(); };
  $$("#rtabs button").forEach(b => b.onclick = () => { UI.rtab = b.dataset.t; if (UI.mode !== "pcb") switchMode("pcb"); renderProps(); });
  $("#libq").oninput = buildLibrary;
  $("#credit").innerHTML = `Wirie ${APP.version} · made by <b>${APP.author}</b>`;
  const c = cv();
  c.addEventListener("mousedown", onDown); window.addEventListener("mousemove", e => { if (e.target === c || UI.drag) onMove(e); }); window.addEventListener("mouseup", onUp);
  c.addEventListener("dblclick", e => { if (UI.track) finishTrack(); else if (UI.mode === "pcb") { const hit = hitPCB(...s2w(...canvasXY(e))); if (hit && (hit.kind === "comp" || hit.kind === "pad")) { const comp = hit.kind === "pad" ? hit.obj.comp : hit.obj; UI.sel = new Set([comp.id]); UI.rtab = "props"; refreshPanels(); setTimeout(() => $("#p_val") && $("#p_val").focus(), 30); } } });
  c.addEventListener("contextmenu", e => e.preventDefault());
  c.addEventListener("mouseleave", () => { $("#tip").classList.remove("on"); });
  c.addEventListener("wheel", e => { e.preventDefault(); const [sx, sy] = canvasXY(e); zoomAt(sx, sy, Math.pow(1.0015, -e.deltaY)); }, { passive: false });
  c.addEventListener("dragover", e => e.preventDefault());
  c.addEventListener("drop", e => { e.preventDefault(); const id = e.dataTransfer.getData("text/wirie-fp"); if (!id) return; if (UI.mode === "strip") return startPlacing(id); const [wx, wy] = s2w(...canvasXY(e)); const [x, y] = snapPt(wx, wy); const comp = addComp(id, x, y); UI.sel = new Set([comp.id]); commit("drop"); });
  window.addEventListener("keydown", onKey); window.addEventListener("keyup", e => { if (e.key === " ") { UI.space = false; setTool(UI.tool); } });
  new ResizeObserver(resize).observe(c);
  // tooltips for buttons
  document.addEventListener("mouseover", e => { const t = e.target.closest("[data-tip]"); const tt = $("#btip"); if (!t) { tt.classList.remove("on"); return; } const r = t.getBoundingClientRect(); tt.textContent = t.dataset.tip; tt.classList.add("on"); const w = tt.offsetWidth; tt.style.left = Math.max(6, Math.min(innerWidth - w - 6, r.left + r.width / 2 - w / 2)) + "px"; tt.style.top = (r.bottom + 8 > innerHeight - 40 ? r.top - 34 : r.bottom + 8) + "px"; if (t.closest("#tools")) { tt.style.left = r.right + 10 + "px"; tt.style.top = r.top + r.height / 2 - 14 + "px"; } });
  buildLibrary(); buildTools(); resize(); fitView(); refreshPanels();
  loadUserParts().then(() => buildLibrary());
  requestAnimationFrame(frame);
  // heartbeat keeps the app alive while this window is open
  const ping = () => fetch("/api/ping").catch(() => {}); ping(); setInterval(ping, 5000);
  window.addEventListener("pagehide", () => navigator.sendBeacon && navigator.sendBeacon("/api/bye"));
  fetch("/api/autosave").then(r => (r.ok ? r.json() : null)).then(p => {
    if (p && p.app === "wirie" && (p.comps?.length || p.tracks?.length)) { loadProject(p); toast(`Welcome back! Restored <b>${esc(S.name)}</b>.`); }
    else { resetHistory(); templatesDialog(true); }
  }).catch(() => { resetHistory(); templatesDialog(true); });
}
window.addEventListener("DOMContentLoaded", boot);
