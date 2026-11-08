// Wirie Part Creator — make your own footprints — by nrmluvdev
"use strict";

const MYCAT = "My Parts";
if (!CATEGORIES.includes(MYCAT)) CATEGORIES.unshift(MYCAT);

// ---------- registry ----------
function registerUserPart(def) {
  const fp = JSON.parse(JSON.stringify(def));
  fp.cat = MYCAT; fp.user = true;
  delete fp.smd; delete fp.search;
  const old = LIB.findIndex(f => f.id === fp.id);
  if (old >= 0) LIB.splice(old, 1);
  delete LIB_BY_ID[fp.id];
  addFP(fp);
  invalidate();
  return LIB_BY_ID[fp.id];
}
async function loadUserParts() {
  try { const r = await api("/api/parts"); for (const d of r.items || []) if (d && d.id) registerUserPart(d); UI.partsFolder = r.folder; } catch (e) { }
}
// parts used in a project travel with it, so shared projects still open elsewhere
function embedUserParts() { S.customParts = {}; for (const c of S.comps) { const fp = compFP(c); if (fp && fp.user) S.customParts[fp.id] = cleanDef(fp); } }
function registerProjectParts(p) { for (const id in (p.customParts || {})) if (!LIB_BY_ID[id] || LIB_BY_ID[id].user) registerUserPart(p.customParts[id]); }
function cleanDef(fp) { const d = JSON.parse(JSON.stringify(fp)); delete d.search; delete d.smd; delete d.user; d.cat = MYCAT; return d; }

// ---------- generators ----------
const PE = { // editor state
  def: null, tpl: "rows", sel: -1, grid: 1.27,
  p: { rows: 2, left: 8, right: 8, pitch: 2.54, rowSp: 15.24, padD: 1.7, drill: 1.0, smd: false, padW: 1.5, padH: 0.9, pinStyle: "module",
    pkg: "dip300", pins: 8, twoPitch: 7.62, twoSmd: false, twoPadW: 1.6, twoPadH: 1.6, twoDrill: 0.8, names: "", namesR: "" },
};
const PKGS = {
  dip300: ["DIP (0.3in rows)", n => dipPads(n, 7.62)], dip600: ["DIP wide (0.6in rows)", n => dipPads(n, 15.24)],
  soic: ["SOIC (1.27mm)", n => soPads(n, 1.27, 5.4, 1.95, 0.6)], soicw: ["SOIC wide (1.27mm)", n => soPads(n, 1.27, 9.3, 2.0, 0.6)],
  ssop: ["SSOP (0.65mm)", n => soPads(n, 0.65, 7.2, 1.75, 0.45)], tssop: ["TSSOP (0.65mm)", n => soPads(n, 0.65, 5.7, 1.5, 0.4)],
  qfp08: ["QFP (0.8mm)", n => qfpPads(n, 0.8, (n / 4 - 1) * 0.8 + 4.4, 1.5, 0.55)], qfp05: ["QFP (0.5mm)", n => qfpPads(n, 0.5, (n / 4 - 1) * 0.5 + 4.4, 1.5, 0.3)],
  qfn05: ["QFN (0.5mm) + centre pad", n => { const s = n / 4, B = s * 0.5 + 0.5; const p = qfpPads(n, 0.5, B, 0.8, 0.28); p.push(smd(n + 1, 0, 0, B * 0.55, B * 0.55, { name: "EP" })); return p; }],
};
function splitNames(t) { return (t || "").split(/[\n,;]+/).map(s => s.trim()).filter(s => s.length); }
function generatePads() {
  const p = PE.p, t = PE.tpl;
  let pads = [];
  if (t === "rows") {
    const L = p.rows === 1 ? +p.left : +p.left, R = p.rows === 2 ? +p.right : 0;
    const nl = splitNames(p.names), nr = splitNames(p.namesR);
    const mk = (n, x, y, nm, first) => p.smd ? smd(n, x, y, +p.padW, +p.padH, { name: nm }) : tht(n, x, y, { d: +p.padD, drill: +p.drill, sq: first, name: nm });
    if (p.rows === 1) { for (let i = 0; i < L; i++) pads.push(mk(i + 1, -(L - 1) * p.pitch / 2 + i * p.pitch, 0, nl[i] || String(i + 1), i === 0)); }
    else if (p.pinStyle === "module") {
      let k = 1; for (let i = 0; i < L; i++) pads.push(mk(k++, -p.rowSp / 2, -(L - 1) * p.pitch / 2 + i * p.pitch, nl[i] || String(k - 1), i === 0));
      for (let i = R - 1; i >= 0; i--) pads.push(mk(k++, p.rowSp / 2, -(R - 1) * p.pitch / 2 + i * p.pitch, nr[i] || String(k - 1), false));
    } else { // header-style zig-zag numbering (1,2 across)
      const n = Math.max(L, R); let k = 1; for (let i = 0; i < n; i++) for (let r = 0; r < 2; r++) { const x = -(n - 1) * p.pitch / 2 + i * p.pitch; const all = splitNames(p.names); pads.push(mk(k, x, (r - 0.5) * p.rowSp, all[k - 1] || String(k), k === 1)); k++; }
    }
  } else if (t === "chip") {
    pads = PKGS[p.pkg][1](+p.pins);
    const nm = splitNames(p.names); pads.forEach((q, i) => { if (nm[i]) q.name = nm[i]; });
  } else if (t === "two") {
    const nm = splitNames(p.names);
    const a = nm[0] || "1", b = nm[1] || "2";
    pads = p.twoSmd ? [smd(1, -p.twoPitch / 2, 0, +p.twoPadW, +p.twoPadH, { name: a }), smd(2, p.twoPitch / 2, 0, +p.twoPadW, +p.twoPadH, { name: b })]
      : [tht(1, -p.twoPitch / 2, 0, { d: +p.twoPadW, drill: +p.twoDrill, sq: /^(\+|K|1)$/.test(a), name: a }), tht(2, p.twoPitch / 2, 0, { d: +p.twoPadW, drill: +p.twoDrill, name: b })];
  } else pads = PE.def.pads;
  return pads.map(q => Object.assign(q, { x: +(+q.x).toFixed(4), y: +(+q.y).toFixed(4) }));
}
function autoBody(pads, holes) {
  let x0 = 1e9, y0 = 1e9, x1 = -1e9, y1 = -1e9;
  for (const q of pads) { x0 = Math.min(x0, q.x - q.w / 2); x1 = Math.max(x1, q.x + q.w / 2); y0 = Math.min(y0, q.y - q.h / 2); y1 = Math.max(y1, q.y + q.h / 2); }
  for (const h of holes) { x0 = Math.min(x0, h.x - h.d / 2); x1 = Math.max(x1, h.x + h.d / 2); y0 = Math.min(y0, h.y - h.d / 2); y1 = Math.max(y1, h.y + h.d / 2); }
  if (x0 > x1) { x0 = y0 = -2; x1 = y1 = 2; }
  return { x: +((x0 + x1) / 2).toFixed(3), y: +((y0 + y1) / 2).toFixed(3), w: +(x1 - x0 + 1.2).toFixed(2), h: +(y1 - y0 + 1.2).toFixed(2) };
}
function rebuildDef() {
  const d = PE.def;
  if (PE.tpl !== "custom") d.pads = generatePads();
  const ab = autoBody(d.pads, d.holes || []);
  if (d.bodyAuto !== false) d.body = Object.assign({}, d.body, ab);
  d.body.col = d.body.col || "#1d4f91";
  d.body.module = !!d.pinLabels;
  d.template = { type: PE.tpl, params: { ...PE.p } };
}

// ---------- editor UI ----------
function newPartDef() {
  return { id: "", name: "", prefix: "U", value: "", desc: "", pads: [], holes: [], body: { x: 0, y: 0, w: 10, h: 10, col: "#1d4f91" }, bodyAuto: true, silk: "box", pinLabels: true, mcu: false, case: { h: 5, raise: 0 }, verify: false };
}
function openPartCreator(editId) {
  const ex = editId && LIB_BY_ID[editId];
  PE.def = ex ? cleanDef(ex) : newPartDef();
  PE.editing = ex ? editId : null;
  if (ex && ex.template) { PE.tpl = ex.template.type; Object.assign(PE.p, ex.template.params); } else if (ex) PE.tpl = "custom"; else PE.tpl = "rows";
  PE.sel = -1;
  if (!ex) { PE.def.name = "My module"; PE.def.value = "My module"; PE.p.names = "VCC\nGND\nSDA\nSCL\nD1\nD2\nD3\nD4"; PE.p.namesR = "5V\nGND\nTX\nRX\nA0\nA1\nA2\nA3"; }
  const d = PE.def, p = PE.p;
  const m = modal(`<div class="mhead">${ico("chip", "big")}<div><h2>${ex ? "Edit part" : "Create a part"}</h2><p class="dim">Pick a template, fill in the numbers from your part's datasheet, and watch the footprint build itself.</p></div><button class="x" data-close>${ico("close")}</button></div>
    <div class="pc">
      <div class="pcform">
        <div class="grid2">${fieldHTML("pc_name", "Part name", d.name)}${fieldHTML("pc_val", "Value / label", d.value)}</div>
        <div class="grid2">${fieldHTML("pc_pre", "Reference letter", d.prefix)}${fieldHTML("pc_desc", "Description", d.desc || "")}</div>
        <h4>Template</h4><div class="seg wide" id="pc_tpl">${[["rows", "Pin rows / module"], ["chip", "Chip package"], ["two", "2-pin part"], ["custom", "Custom pads"]].map(([k, l]) => `<button data-t="${k}" class="${PE.tpl === k ? "on" : ""}">${l}</button>`).join("")}</div>
        <div id="pc_params"></div>
        <h4>Body & behaviour</h4>
        <div class="grid3">${fieldHTML("pc_bw", "Body width", d.body.w, "mm", "number")}${fieldHTML("pc_bh", "Body height", d.body.h, "mm", "number")}<label class="fld"><span>Colour</span><div class="inp"><input id="pc_col" type="color" value="${d.body.col && d.body.col.length === 7 ? d.body.col : "#1d4f91"}"></div></label></div>
        <label class="chk"><input type="checkbox" id="pc_auto" ${d.bodyAuto !== false ? "checked" : ""}><span>Size the body around the pads automatically</span></label>
        <label class="chk"><input type="checkbox" id="pc_lbl" ${d.pinLabels ? "checked" : ""}><span>It's a module with labelled pins (same-name pins like GND are joined inside it)</span></label>
        <label class="chk"><input type="checkbox" id="pc_mcu" ${d.mcu ? "checked" : ""}><span>It's a microcontroller (other modules can Smart-connect to it)</span></label>
        <h4>3D / case <em>for Makie</em></h4>
        <div class="grid3">${fieldHTML("pc_h", "Height", d.case?.h ?? 5, "mm", "number")}${fieldHTML("pc_rz", "Raised above board", d.case?.raise ?? 0, "mm", "number")}<label class="fld"><span>Needs a case hole</span><div class="inp"><select id="pc_open">${[["", "None"], ["usb-c", "USB-C (wall)"], ["usb-micro", "Micro-USB (wall)"], ["connector", "Connector (wall)"], ["button", "Button (lid)"], ["led", "LED (lid)"], ["display-window", "Screen (lid)"], ["knob", "Knob (lid)"]].map(([k, l]) => `<option value="${k}" ${(d.case?.openType || "") === k ? "selected" : ""}>${l}</option>`).join("")}</select></div></label></div>
        <div class="grid2" id="pc_opendir"><label class="fld"><span>Wall hole faces</span><div class="inp"><select id="pc_side">${[["top", "Part's top edge"], ["bottom", "Part's bottom edge"], ["left", "Part's left edge"], ["right", "Part's right edge"]].map(([k, l]) => `<option value="${k}" ${(d.case?.openSide || "top") === k ? "selected" : ""}>${l}</option>`).join("")}</select></div></label>${fieldHTML("pc_osz", "Hole size (w × h or ⌀)", d.case?.openSize || "", "mm")}</div>
        <label class="chk"><input type="checkbox" id="pc_ver" ${d.verify ? "checked" : ""}><span>Mark as "check before ordering" (!)</span></label>
      </div>
      <div class="pcview"><div class="pcbar"><span class="dim" id="pc_info"></span><span class="grow"></span><div class="seg" id="pc_grid">${[0.5, 0.635, 1.27, 2.54].map(g => `<button data-g="${g}" class="${g === PE.grid ? "on" : ""}">${g}</button>`).join("")}</div></div><canvas id="pc_cv"></canvas><div id="pc_padedit"></div></div>
    </div>
    <div class="mfoot">${ex ? `<button class="btn danger" id="pc_del">${ico("trash")}Delete part</button>` : ""}<label class="btn ghost">${ico("open")}Import…<input type="file" id="pc_imp" accept=".json,.wpart" hidden></label><button class="btn ghost" id="pc_exp">${ico("export")}Export file</button><span class="grow"></span><button class="btn ghost" data-close>Cancel</button><button class="btn primary" id="pc_save">${ico("save")}Save to My Parts</button></div>`, "huge");
  const $m = s => m.querySelector(s);
  const params = () => {
    const P = $m("#pc_params");
    if (PE.tpl === "rows") P.innerHTML = `<div class="grid3"><label class="fld"><span>Rows</span><div class="inp"><select id="pp_rows"><option value="1" ${p.rows == 1 ? "selected" : ""}>1 row</option><option value="2" ${p.rows == 2 ? "selected" : ""}>2 rows</option></select></div></label>${fieldHTML("pp_left", p.rows == 2 ? "Pins in left row" : "Pins", p.left, "", "number")}${p.rows == 2 ? fieldHTML("pp_right", "Pins in right row", p.right, "", "number") : "<span></span>"}</div>
      <div class="grid3">${fieldHTML("pp_pitch", "Pin spacing (pitch)", p.pitch, "mm", "number")}${p.rows == 2 ? fieldHTML("pp_rowsp", "Gap between rows", p.rowSp, "mm", "number") : "<span></span>"}${p.rows == 2 ? `<label class="fld"><span>Pin numbering</span><div class="inp"><select id="pp_style"><option value="module" ${p.pinStyle === "module" ? "selected" : ""}>Down left, up right (modules/DIP)</option><option value="zigzag" ${p.pinStyle === "zigzag" ? "selected" : ""}>Zig-zag (2-row headers)</option></select></div></label>` : "<span></span>"}</div>
      <div class="grid3"><label class="chk"><input type="checkbox" id="pp_smd" ${p.smd ? "checked" : ""}><span>Surface-mount pads</span></label>${p.smd ? fieldHTML("pp_pw", "Pad width", p.padW, "mm", "number") + fieldHTML("pp_ph", "Pad height", p.padH, "mm", "number") : fieldHTML("pp_pd", "Pad size", p.padD, "mm", "number") + fieldHTML("pp_dr", "Drill", p.drill, "mm", "number")}</div>
      <div class="grid2"><label class="fld"><span>${p.rows == 2 && p.pinStyle === "module" ? "Left pin names (top → bottom)" : "Pin names (pin 1, 2, 3…)"}</span><textarea id="pp_names" rows="5">${esc(p.names)}</textarea></label>${p.rows == 2 && p.pinStyle === "module" ? `<label class="fld"><span>Right pin names (top → bottom)</span><textarea id="pp_namesr" rows="5">${esc(p.namesR)}</textarea></label>` : ""}</div>
      <p class="note">Common values: 2.54mm pitch for headers & modules · rows 15.24mm (Nano/Pico-style), 22.86 (D1 Mini/NodeMCU), 25.4 (ESP32 DevKit).</p>`;
    else if (PE.tpl === "chip") P.innerHTML = `<div class="grid2"><label class="fld"><span>Package</span><div class="inp"><select id="pp_pkg">${Object.entries(PKGS).map(([k, v]) => `<option value="${k}" ${p.pkg === k ? "selected" : ""}>${v[0]}</option>`).join("")}</select></div></label>${fieldHTML("pp_pins", "Number of pins", p.pins, "", "number")}</div>
      <label class="fld"><span>Pin names in order (pin 1, 2, 3… — one per line or comma separated)</span><textarea id="pp_names" rows="6">${esc(p.names)}</textarea></label>`;
    else if (PE.tpl === "two") P.innerHTML = `<div class="grid3">${fieldHTML("pp_tp", "Pin spacing", p.twoPitch, "mm", "number")}<label class="chk"><input type="checkbox" id="pp_tsmd" ${p.twoSmd ? "checked" : ""}><span>Surface-mount</span></label><span></span></div>
      <div class="grid3">${fieldHTML("pp_tw", p.twoSmd ? "Pad width" : "Pad size", p.twoPadW, "mm", "number")}${p.twoSmd ? fieldHTML("pp_th", "Pad height", p.twoPadH, "mm", "number") : fieldHTML("pp_tdr", "Drill", p.twoDrill, "mm", "number")}${fieldHTML("pp_names2", "Pin names", splitNames(p.names).slice(0, 2).join(", ") || "1, 2")}</div><p class="note">Tip: name polarised pins "+ , -" or "A, K" — pin 1 gets the square pad.</p>`;
    else P.innerHTML = `<p class="note">Click on the grid to add a pad. Click a pad to edit it, drag to move it, press Delete to remove it. Holes (no copper) can be added with the button below.</p><div class="btnrow"><button class="btn sm" id="pp_addpad">${ico("place")}Add pad at centre</button><button class="btn sm" id="pp_addhole">${ico("via")}Add mounting hole</button><button class="btn sm" id="pp_fromtpl">${ico("magic")}Start from current template</button></div>`;
    bind();
  };
  const num = (id, def) => { const el = $m(id); if (!el) return def; const v = parseFloat(el.value); return isFinite(v) ? v : def; };
  const readParams = () => {
    if (PE.tpl === "rows") { p.rows = +$m("#pp_rows").value; p.left = Math.max(1, Math.min(120, num("#pp_left", p.left) | 0)); if ($m("#pp_right")) p.right = Math.max(0, Math.min(120, num("#pp_right", p.right) | 0)); p.pitch = num("#pp_pitch", p.pitch); if ($m("#pp_rowsp")) p.rowSp = num("#pp_rowsp", p.rowSp); if ($m("#pp_style")) p.pinStyle = $m("#pp_style").value; p.smd = $m("#pp_smd").checked; if (p.smd) { p.padW = num("#pp_pw", p.padW); p.padH = num("#pp_ph", p.padH); } else { p.padD = num("#pp_pd", p.padD); p.drill = Math.min(num("#pp_dr", p.drill), p.padD - 0.2); } p.names = $m("#pp_names").value; if ($m("#pp_namesr")) p.namesR = $m("#pp_namesr").value; }
    if (PE.tpl === "chip") { p.pkg = $m("#pp_pkg").value; let n = Math.max(2, num("#pp_pins", p.pins) | 0); n = /qf/.test(p.pkg) ? Math.max(8, Math.round(n / 4) * 4) : Math.max(4, Math.round(n / 2) * 2); p.pins = n; p.names = $m("#pp_names").value; }
    if (PE.tpl === "two") { p.twoPitch = num("#pp_tp", p.twoPitch); p.twoSmd = $m("#pp_tsmd").checked; p.twoPadW = num("#pp_tw", p.twoPadW); if (p.twoSmd) p.twoPadH = num("#pp_th", p.twoPadH); else p.twoDrill = Math.min(num("#pp_tdr", p.twoDrill), p.twoPadW - 0.2); p.names = $m("#pp_names2").value; }
  };
  const readMeta = () => {
    d.name = $m("#pc_name").value.trim(); d.value = $m("#pc_val").value.trim() || d.name; d.prefix = ($m("#pc_pre").value.trim() || "U").replace(/[^A-Za-z]/g, "").slice(0, 4) || "U"; d.desc = $m("#pc_desc").value.trim();
    d.bodyAuto = $m("#pc_auto").checked; d.body.col = $m("#pc_col").value;
    if (!d.bodyAuto) { d.body.w = Math.max(0.5, num("#pc_bw", d.body.w)); d.body.h = Math.max(0.5, num("#pc_bh", d.body.h)); }
    d.pinLabels = $m("#pc_lbl").checked; d.mcu = $m("#pc_mcu").checked; d.verify = $m("#pc_ver").checked;
    const ot = $m("#pc_open").value, side = $m("#pc_side").value, sz = $m("#pc_osz").value.split(/[x×*, ]+/).map(parseFloat).filter(isFinite);
    d.case = { h: Math.max(0, num("#pc_h", 5)), raise: Math.max(0, num("#pc_rz", 0)), openType: ot, openSide: side, openSize: $m("#pc_osz").value };
    $m("#pc_opendir").style.display = ot && !["button", "led", "display-window", "knob"].includes(ot) ? "" : "none";
    applyCaseOpening(d, ot, side, sz);
  };
  const bind = () => {
    m.querySelectorAll("#pc_params input, #pc_params select, #pc_params textarea").forEach(el => { el.oninput = () => { readParams(); update(false); }; el.onchange = () => { readParams(); update(true); }; });
    const ap = $m("#pp_addpad"); if (ap) ap.onclick = () => { const n = String(d.pads.length + 1); d.pads.push({ n, name: n, x: 0, y: 0, shape: "circle", w: 1.7, h: 1.7, drill: 1.0, slot: null, func: "" }); PE.sel = d.pads.length - 1; update(); };
    const ah = $m("#pp_addhole"); if (ah) ah.onclick = () => { d.holes = d.holes || []; d.holes.push({ x: 0, y: 0, d: 3.2 }); update(); };
    const ft = $m("#pp_fromtpl"); if (ft) ft.onclick = () => { const keep = PE.tpl; PE.tpl = (PE.def.template && PE.def.template.type !== "custom") ? PE.def.template.type : "rows"; d.pads = generatePads(); PE.tpl = keep; update(); toast("Pads copied — now edit them freely."); };
  };
  const cv2 = $m("#pc_cv");
  const view = { s: 10, ox: 0, oy: 0 };
  const draw = () => {
    const dpr = window.devicePixelRatio || 1; const W = cv2.clientWidth, H = cv2.clientHeight; if (!cv2.isConnected || W < 20 || H < 20) return; cv2.width = W * dpr; cv2.height = H * dpr;
    const ctx = cv2.getContext("2d"); ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    ctx.fillStyle = "#0d1219"; ctx.fillRect(0, 0, W, H);
    const fp = d; const b = fp.body;
    let x0 = b.x - b.w / 2, x1 = b.x + b.w / 2, y0 = b.y - b.h / 2, y1 = b.y + b.h / 2;
    for (const q of fp.pads) { x0 = Math.min(x0, q.x - q.w); x1 = Math.max(x1, q.x + q.w); y0 = Math.min(y0, q.y - q.h); y1 = Math.max(y1, q.y + q.h); }
    view.s = Math.min((W - 60) / (x1 - x0 || 1), (H - 60) / (y1 - y0 || 1), 60); view.ox = W / 2 - (x0 + x1) / 2 * view.s; view.oy = H / 2 - (y0 + y1) / 2 * view.s;
    ctx.setTransform(dpr * view.s, 0, 0, dpr * view.s, dpr * view.ox, dpr * view.oy);
    const px = 1 / view.s;
    // grid
    let g = PE.grid; while (g * view.s < 8 && g < 1000) g *= 2;
    ctx.fillStyle = "#1f2938";
    const gx0 = Math.floor((-view.ox / view.s) / g) * g, gx1 = (W - view.ox) / view.s, gy0 = Math.floor((-view.oy / view.s) / g) * g, gy1 = (H - view.oy) / view.s;
    for (let x = gx0; x < gx1; x += g) for (let y = gy0; y < gy1; y += g) ctx.fillRect(x - px, y - px, 2 * px, 2 * px);
    ctx.strokeStyle = "rgba(63,224,208,.25)"; ctx.lineWidth = px; ctx.beginPath(); ctx.moveTo(-2, 0); ctx.lineTo(2, 0); ctx.moveTo(0, -2); ctx.lineTo(0, 2); ctx.stroke();
    // body
    ctx.fillStyle = (b.col || "#1d4f91") + "55"; ctx.strokeStyle = "rgba(255,255,255,.5)"; ctx.lineWidth = 1.5 * px; rrect(ctx, x0b(b), y0b(b), b.w, b.h, 0.6); ctx.fill(); ctx.stroke();
    for (const h of fp.holes || []) { ctx.fillStyle = "#05070a"; ctx.beginPath(); ctx.arc(h.x, h.y, h.d / 2, 0, 7); ctx.fill(); ctx.strokeStyle = "#8aa"; ctx.stroke(); }
    fp.pads.forEach((q, i) => {
      ctx.fillStyle = q.drill ? "#e2b04f" : "#ef6a43"; padPath(ctx, q); ctx.fill();
      if (q.drill) { ctx.fillStyle = "#0d1219"; drillPath(ctx, q); ctx.fill(); }
      if (i === PE.sel) { ctx.strokeStyle = "#3fe0d0"; ctx.lineWidth = 2.5 * px; padPath(ctx, q, 0.25); ctx.stroke(); }
      const fs = Math.min(q.w, q.h) * (q.name.length > 3 ? 0.26 : 0.34); if (fs * view.s > 6) { ctx.font = `700 ${fs}px Segoe UI, sans-serif`; ctx.textAlign = "center"; ctx.textBaseline = "middle"; ctx.lineWidth = fs * 0.22; ctx.strokeStyle = "rgba(0,0,0,.85)"; ctx.strokeText(q.name, q.x, q.y); ctx.fillStyle = "#fff"; ctx.fillText(q.name, q.x, q.y); }
    });
    $m("#pc_info").textContent = `${fp.pads.length} pads · body ${b.w} × ${b.h} mm${fp.pads.every(q => !q.drill) ? " · surface-mount" : ""}`;
  };
  const x0b = b => b.x - b.w / 2, y0b = b => b.y - b.h / 2;
  const padEditor = () => {
    const E = $m("#pc_padedit"); const q = d.pads[PE.sel];
    if (!q || PE.tpl !== "custom") { E.innerHTML = PE.tpl === "custom" ? `<p class="note">Select a pad to edit it.</p>` : ""; return; }
    E.innerHTML = `<div class="grid4">${fieldHTML("pe_name", "Name", q.name)}${fieldHTML("pe_x", "X", q.x, "mm", "number")}${fieldHTML("pe_y", "Y", q.y, "mm", "number")}<label class="fld"><span>Shape</span><div class="inp"><select id="pe_shape">${["circle", "rect", "oval"].map(s => `<option ${q.shape === s ? "selected" : ""}>${s}</option>`).join("")}</select></div></label>${fieldHTML("pe_w", "Width", q.w, "mm", "number")}${fieldHTML("pe_h", "Height", q.h, "mm", "number")}${fieldHTML("pe_d", "Drill (0 = SMD)", q.drill, "mm", "number")}<div class="fld"><span>&nbsp;</span><button class="btn danger sm" id="pe_del">${ico("trash")}Remove</button></div></div>`;
    const rd = () => { q.name = $m("#pe_name").value || q.n; q.x = num("#pe_x", q.x); q.y = num("#pe_y", q.y); q.shape = $m("#pe_shape").value; q.w = Math.max(0.2, num("#pe_w", q.w)); q.h = Math.max(0.2, num("#pe_h", q.h)); q.drill = Math.max(0, Math.min(num("#pe_d", q.drill), Math.min(q.w, q.h) - 0.15)); update(false); };
    E.querySelectorAll("input,select").forEach(el => el.onchange = rd);
    $m("#pe_del").onclick = () => { d.pads.splice(PE.sel, 1); d.pads.forEach((pp, i) => { if (/^\d+$/.test(pp.n)) pp.n = String(i + 1); }); PE.sel = -1; update(); };
  };
  const update = (full = true) => { readMeta(); rebuildDef(); if (d.bodyAuto !== false) { $m("#pc_bw").value = d.body.w; $m("#pc_bh").value = d.body.h; } draw(); if (full) padEditor(); };
  // canvas interaction (custom pads)
  let drag = null;
  const toW = e => { const r = cv2.getBoundingClientRect(); return [(e.clientX - r.left - view.ox) / view.s, (e.clientY - r.top - view.oy) / view.s]; };
  cv2.onmousedown = e => {
    if (PE.tpl !== "custom") { toast("Switch to <b>Custom pads</b> to place pads by hand (your current pads are kept).", ""); return; }
    const [wx, wy] = toW(e); const i = d.pads.findIndex(q => ptInShape(wx, wy, padShape(q), 0.2));
    if (i >= 0) { PE.sel = i; drag = { i }; }
    else { const g = PE.grid; const n = String(d.pads.length + 1); const last = d.pads[d.pads.length - 1]; d.pads.push(Object.assign({ n, name: n, x: +snapTo(wx, g).toFixed(4), y: +snapTo(wy, g).toFixed(4), shape: "circle", w: 1.7, h: 1.7, drill: 1.0, slot: null, func: "" }, last ? { shape: last.shape === "rect" && last.n === "1" ? "circle" : last.shape, w: last.w, h: last.h, drill: last.drill } : {})); PE.sel = d.pads.length - 1; drag = { i: PE.sel }; }
    update();
  };
  cv2.onmousemove = e => { if (!drag) return; const [wx, wy] = toW(e); const q = d.pads[drag.i]; q.x = +snapTo(wx, PE.grid).toFixed(4); q.y = +snapTo(wy, PE.grid).toFixed(4); rebuildDef(); draw(); };
  const mu = () => { if (!cv2.isConnected) { window.removeEventListener("mouseup", mu); return; } if (drag) { drag = null; padEditor(); } }; window.addEventListener("mouseup", mu);
  m.addEventListener("keydown", e => { if ((e.key === "Delete") && PE.tpl === "custom" && PE.sel >= 0 && !e.target.matches("input,textarea,select")) { d.pads.splice(PE.sel, 1); PE.sel = -1; update(); } });
  m.querySelectorAll("#pc_tpl button").forEach(b => b.onclick = () => { if (b.dataset.t === "custom" && PE.tpl !== "custom") { /* keep generated pads to edit */ } PE.tpl = b.dataset.t; m.querySelectorAll("#pc_tpl button").forEach(x => x.classList.toggle("on", x === b)); params(); update(); });
  m.querySelectorAll("#pc_grid button").forEach(b => b.onclick = () => { PE.grid = +b.dataset.g; m.querySelectorAll("#pc_grid button").forEach(x => x.classList.toggle("on", x === b)); draw(); });
  m.querySelectorAll(".pcform > .grid2 input, .pcform > .grid3 input, .pcform > .grid3 select, .pcform > .grid2 select, .pcform > .chk input").forEach(el => el.onchange = () => update(false));
  $m("#pc_save").onclick = async () => {
    update();
    if (!d.name) { toast("Give your part a name first.", "warn"); $m("#pc_name").focus(); return; }
    if (!d.pads.length && !(d.holes || []).length) { toast("The part has no pads yet.", "warn"); return; }
    const names = d.pads.map(q => q.n); if (new Set(names).size !== names.length) d.pads.forEach((q, i) => (q.n = String(i + 1)));
    if (!d.id) d.id = "user_" + d.name.toLowerCase().replace(/[^a-z0-9]+/g, "_").replace(/^_|_$/g, "").slice(0, 40) + "_" + Date.now().toString(36).slice(-4);
    d.tags = "my part custom " + (d.desc || "");
    const fp = registerUserPart(d);
    try { await api("/api/parts", { id: d.id, def: cleanDef(fp) }); } catch (e) { toast("Saved for this session only (couldn't write the Parts folder).", "warn"); }
    for (const c of S.comps) if (c.fp === d.id) invalidate();
    commit("part saved"); closeModal(); UI.libOpen = Object.assign(UI.libOpen || {}, { [MYCAT]: true }); buildLibrary();
    toast(`${ico("check")} <b>${esc(d.name)}</b> saved to My Parts — click it in the library to place it.`, "good");
  };
  $m("#pc_exp").onclick = async () => { update(); const txt = JSON.stringify(Object.assign(cleanDef(d), { id: d.id || "user_" + Date.now().toString(36) }), null, 1); try { const r = await api("/api/savefile", { name: d.name || "part", ext: ".wpart.json", content: txt }); toast(`Exported to <b>${esc(r.path)}</b>`, "good"); } catch (e) { downloadFile((d.name || "part") + ".wpart.json", txt, "application/json"); } };
  $m("#pc_imp").onchange = e => { const f = e.target.files[0]; if (!f) return; f.text().then(t => { try { const imp = JSON.parse(t); if (!imp.pads) throw 0; closeModal(); PE.def = null; registerUserPart(Object.assign(imp, { id: imp.id || "user_imp_" + Date.now().toString(36) })); api("/api/parts", { id: imp.id, def: cleanDef(LIB_BY_ID[imp.id]) }).catch(() => {}); buildLibrary(); openPartCreator(imp.id); } catch (er) { toast("That file isn't a Wirie part.", "bad"); } }); };
  const del = $m("#pc_del"); if (del) del.onclick = async () => {
    if (S.comps.some(c => c.fp === PE.editing)) { toast("That part is used on this board — remove it from the board first.", "warn"); return; }
    try { await api("/api/parts", { id: PE.editing, delete: true }); } catch (e) { }
    const i = LIB.findIndex(f => f.id === PE.editing); if (i >= 0) LIB.splice(i, 1); delete LIB_BY_ID[PE.editing]; closeModal(); buildLibrary(); toast("Part deleted");
  };
  params(); update();
  let lastW = 0; new ResizeObserver(() => { if (cv2.clientWidth !== lastW) { lastW = cv2.clientWidth; draw(); } }).observe(cv2);
}
function applyCaseOpening(d, ot, side, sz) {
  const b = d.body; const c = d.case; delete c.edge; delete c.lid;
  if (!ot) return;
  if (["button", "led", "display-window", "knob"].includes(ot)) {
    const dd = sz[0] || (ot === "led" ? 5.3 : ot === "button" ? 4 : ot === "knob" ? 7.5 : 0);
    c.lid = [ot === "display-window" ? { type: ot, shape: "rect", x: b.x, y: b.y, w: sz[0] || Math.max(4, b.w - 4), h: sz[1] || Math.max(4, b.h - 6), approx: !sz.length } : { type: ot, shape: "circle", x: b.x, y: b.y, d: dd }];
  } else {
    const v = { top: [0, -1], bottom: [0, 1], left: [-1, 0], right: [1, 0] }[side];
    const at = [b.x + v[0] * b.w / 2, b.y + v[1] * b.h / 2];
    c.edge = { type: ot, local: v, at, w: sz[0] || 12, h: sz[1] || 7 };
  }
}
