// Scopie — interface. Part of the free nrmluvdev maker suite. Created by nrmluvdev.
//
// Scopie's Go side does all the heavy lifting (capture, triggering, averaging,
// persistence, measurements, FFT). This page shows the controls, sends every
// change as the full settings object, and draws what /api/frame returns.
"use strict";

const $ = (id) => document.getElementById(id);
const COLORS = ["#ffd400", "#2fd5ff", "#ff5ad2", "#4f8dff", "#c38bff"];
const TRIG = "#ff8c1a", CUR = "#7dff7a", XYCOL = "#9dff6a";
const VY = 4, MAXCH = 4, MATH = 4;
const MATH_OPS = ["A + B", "A − B", "A × B"];

let META = null, S = null, INFO = null, CONNECTED = false, STATUS = null;
let FR = null;             // last frame payload {hdr, traces, persist, xy, fft}
let seq = -1, frames = 0, fpsT = performance.now(), fps = 0;
let needDraw = true; // redraw only when something changed
const redraw = () => { needDraw = true; };

// ================================================================ helpers

async function api(path, body, method) {
  const opt = { method: method || (body === undefined ? "GET" : "POST"), headers: { "X-Scopie": "1" } };
  if (body !== undefined) { opt.headers["Content-Type"] = "application/json"; opt.body = JSON.stringify(body); }
  const r = await fetch("/api/" + path, opt);
  const j = await r.json().catch(() => ({}));
  if (!r.ok) throw new Error(j.error || r.statusText);
  return j;
}

function esc(s) { return String(s).replace(/[&<>"]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" }[c])); }
function el(tag, cls, html) { const e = document.createElement(tag); if (cls) e.className = cls; if (html != null) e.innerHTML = html; return e; }
const clamp = (v, lo, hi) => Math.max(lo, Math.min(hi, v));

function stripZeros(s) { return s.includes(".") ? s.replace(/\.?0+$/, "") : s; }
// gfmt mimics Python's "{:.Ng}"
function gfmt(v, d) {
  if (v === 0) return "0";
  const r = Number(v.toPrecision(d));
  const ex = Math.floor(Math.log10(Math.abs(r)));
  if (ex < -4 || ex >= d) {
    let [m, e] = r.toExponential(d - 1).split("e");
    const n = parseInt(e, 10);
    return stripZeros(m) + "e" + (n < 0 ? "-" : "+") + String(Math.abs(n)).padStart(2, "0");
  }
  return stripZeros(r.toFixed(Math.max(0, d - 1 - ex)));
}
function fmtSI(v, unit = "", digits = 4) {
  if (v == null || !isFinite(v)) return "--";
  if (v === 0) return ("0 " + unit).trim();
  const P = [[1e9, "G"], [1e6, "M"], [1e3, "k"], [1, ""], [1e-3, "m"], [1e-6, "µ"], [1e-9, "n"], [1e-12, "p"]];
  const a = Math.abs(v);
  let sc = 1e-12, p = "p";
  for (const [s, q] of P) if (a >= s * 0.9995) { sc = s; p = q; break; }
  return (gfmt(v / sc, digits) + " " + p + unit).trim();
}
const fmtDiv = (v, u) => fmtSI(v, u, 3).replace(" ", "");
function nearestIndex(list, v) {
  if (!list || !list.length || !(v > 0)) return 0;
  let bi = 0, bd = Infinity;
  list.forEach((x, i) => { const d = Math.abs(Math.log(x / v)); if (d < bd) { bd = d; bi = i; } });
  return bi;
}
const nch = () => Math.max(1, Math.min(MAXCH, (INFO && INFO.channels) || 2));
const hx = () => ((INFO && INFO.hdivs) || 10) / 2;
const vdivTip = (c) => S.ch[c].vdiv * S.ch[c].probe;
const sign = (c) => (S.ch[c].invert ? -1 : 1);
const chOn = (c) => c < nch() && S.ch[c].on;
const isStream = () => INFO && INFO.kind === "stream";
const isRoll = () => INFO && INFO.rollMode && S.roll && S.tdiv >= 100e-3 && !S.xy;
const trigSrc = () => clamp(S.trigSource, 0, nch() - 1);
function stamp() {
  const d = new Date(), p = (n) => String(n).padStart(2, "0");
  return `${d.getFullYear()}${p(d.getMonth() + 1)}${p(d.getDate())}-${p(d.getHours())}${p(d.getMinutes())}${p(d.getSeconds())}`;
}

// ================================================================ help bar & toasts

let helpSticky = null;
function tipHTML(key) {
  const t = TIPS[key];
  if (!t) return key;
  return `<b>${esc(t[0])}</b> — ${esc(t[1]).replace(/\n/g, "<br>")}`;
}
function showHelp(key) {
  const bar = $("helpBar");
  let html;
  if (key == null) html = helpSticky || DEFAULT_HELP;
  else { helpSticky = null; html = TIPS[key] ? tipHTML(key) : key; }
  html = `<span class="bulb">💡</span>${html}`;
  if (bar._h !== html) { bar.innerHTML = html; bar._h = html; }
}
function stickyHelp(html) { helpSticky = html; showHelp(null); }
document.addEventListener("mouseover", (e) => {
  const t = e.target.closest && e.target.closest("[data-help]");
  if (t && t.id !== "scopeWrap") showHelp(t.dataset.help);
});
document.addEventListener("mouseout", (e) => {
  const t = e.target.closest && e.target.closest("[data-help]");
  const to = e.relatedTarget && e.relatedTarget.closest && e.relatedTarget.closest("[data-help]");
  if (t && !to) showHelp(null);
});

function toast(msg, kind = "", action) {
  const t = el("div", "toast " + kind);
  t.innerHTML = `<i></i><div>${esc(msg).replace(/\n/g, "<br>")}</div>`;
  if (action) {
    const a = el("a", "", esc(action.label));
    a.onclick = action.fn;
    t.lastChild.appendChild(a);
  }
  $("toasts").appendChild(t);
  setTimeout(() => { t.classList.add("out"); setTimeout(() => t.remove(), 220); }, kind === "err" ? 7000 : 4500);
}
const showFolder = { label: "Show folder", fn: () => api("open-folder", {}) };

// ================================================================ modals

function modal(title, bodyHTML, buttons, cls = "") {
  const m = el("div", "modal");
  const c = el("div", "modal-card " + cls);
  c.innerHTML = `<div class="modal-h"><span>${esc(title)}</span><button class="icon-btn sm x"><svg viewBox="0 0 24 24"><path d="M6 6l12 12M18 6L6 18"/></svg></button></div><div class="modal-b"></div><div class="modal-f"></div>`;
  const b = c.querySelector(".modal-b");
  if (typeof bodyHTML === "string") b.innerHTML = bodyHTML; else if (bodyHTML) b.appendChild(bodyHTML);
  const f = c.querySelector(".modal-f");
  f.appendChild(el("div", "grow"));
  const close = () => { m.remove(); document.removeEventListener("keydown", onKey, true); };
  (buttons || [{ label: "Close", primary: true }]).forEach((bt) => {
    const btn = el("button", "btn" + (bt.primary ? " primary" : "") + (bt.ghost ? " ghost" : ""), esc(bt.label));
    if (bt.left) f.insertBefore(btn, f.firstChild);
    else f.appendChild(btn);
    btn.onclick = async () => { if (bt.fn && (await bt.fn(btn)) === false) return; close(); };
    bt.el = btn;
  });
  c.querySelector(".x").onclick = close;
  m.addEventListener("mousedown", (e) => { if (e.target === m) close(); });
  const onKey = (e) => { if (e.key === "Escape") { e.stopPropagation(); close(); } };
  document.addEventListener("keydown", onKey, true);
  m.appendChild(c);
  $("modals").appendChild(m);
  return { el: m, body: b, foot: f, close };
}
function message(title, text) {
  return modal(title, `<div class="msgtext">${esc(text)}</div>`, [{ label: "OK", primary: true }], "small");
}

// ================================================================ side panel

const REFRESH = [];
function onRefresh(fn) { REFRESH.push(fn); return fn; }
function refreshControls() { needDraw = true; REFRESH.forEach((f) => f()); updateChips(); buildMeasTable(); updateCursors(); layoutFFT(); }

let pushTimer = null, pushSeq = 0;
function changed(now) {
  needDraw = true;
  pushSeq++;
  clearTimeout(pushTimer);
  pushTimer = setTimeout(pushSettings, now ? 0 : 30);
  updateChips();
  updateCursors();
}
async function pushSettings() {
  const my = pushSeq;
  try {
    const ns = await api("settings", S, "PUT");
    if (my === pushSeq) { S = ns; refreshControls(); }
  } catch (e) { toast(e.message, "err"); }
}

function collapsed(id) { try { return localStorage.getItem("card." + id) === "1"; } catch { return false; } }
function setCollapsed(id, v) { try { localStorage.setItem("card." + id, v ? "1" : "0"); } catch { /* ignore */ } }

function card(id, title, color, help) {
  const c = el("div", "card" + (collapsed(id) ? " closed" : ""));
  c.id = "card-" + id;
  if (color) c.style.borderColor = color + "55";
  const h = el("div", "card-h", `${color ? `<span class="sw" style="background:${color}"></span>` : ""}<span>${esc(title)}</span><svg class="chev" viewBox="0 0 24 24"><path d="M6 9l6 6 6-6"/></svg>`);
  if (help) h.dataset.help = help;
  h.onclick = () => { c.classList.toggle("closed"); setCollapsed(id, c.classList.contains("closed")); };
  const b = el("div", "card-b");
  c.append(h, b);
  $("side").appendChild(c);
  return { card: c, body: b };
}
function row(parent, label, help) {
  const r = el("div", "row");
  const l = el("label", "l", esc(label));
  const f = el("div", "f");
  r.append(l, f);
  if (help) r.dataset.help = help;
  parent.appendChild(r);
  return { row: r, f };
}
// select whose options come from opts() (rebuilt when they change)
function select(parent, opts, get, set, help) {
  const s = el("select", "inp");
  if (help) s.dataset.help = help;
  let sig = "";
  onRefresh(() => {
    const o = opts();
    const k = JSON.stringify(o);
    if (k !== sig) {
      sig = k;
      s.innerHTML = "";
      o.forEach((x, i) => {
        const op = el("option", "", esc(x.t));
        op.value = i;
        if (x.disabled) op.disabled = true;
        if (x.title) op.title = x.title;
        s.appendChild(op);
      });
    }
    const i = get(o);
    if (i >= 0 && String(i) !== s.value) s.value = String(i);
  });
  s.onchange = () => { set(opts()[+s.value], +s.value); changed(); };
  parent.appendChild(s);
  return s;
}
function seg(parent, items, get, set, help, disabled) {
  const s = el("div", "seg");
  if (help) s.dataset.help = help;
  const bs = items.map(([v, t]) => {
    const b = el("button", "", esc(t));
    b.onclick = () => { if (!b.disabled) { set(v); changed(); refreshControls(); } };
    s.appendChild(b);
    return b;
  });
  onRefresh(() => bs.forEach((b, i) => {
    b.classList.toggle("active", get() === items[i][0]);
    b.disabled = !!(disabled && disabled(items[i][0]));
  }));
  parent.appendChild(s);
  return s;
}
function toggle(parent, text, get, set, help) {
  const l = el("label", "tgl", `<input type="checkbox"><span class="tr"></span><span>${esc(text)}</span>`);
  const i = l.querySelector("input");
  if (help) l.dataset.help = help;
  i.onchange = () => { set(i.checked); changed(); refreshControls(); };
  onRefresh(() => { i.checked = !!get(); });
  parent.appendChild(l);
  return l;
}
function slider(parent, lo, hi, step, get, set, help, color) {
  const s = el("input");
  s.type = "range";
  s.step = step;
  if (help) s.dataset.help = help;
  if (color) s.style.setProperty("--thumb", color + "88");
  s.oninput = () => { set(+s.value); changed(); };
  onRefresh(() => {
    const [a, b] = typeof lo === "function" ? [lo(), hi()] : [lo, hi];
    s.min = a; s.max = b;
    if (document.activeElement !== s) s.value = get();
  });
  parent.appendChild(s);
  return s;
}
function button(parent, text, fn, help, cls = "btn sm") {
  const b = el("button", cls, text);
  b.onclick = fn;
  if (help) b.dataset.help = help;
  parent.appendChild(b);
  return b;
}
function shown(elm, fn) { onRefresh(() => elm.classList.toggle("hidden", !fn())); }

function chanOpts(withMath) {
  const o = [];
  for (let c = 0; c < nch(); c++) o.push({ t: "CH" + (c + 1), v: c });
  if (withMath) o.push({ t: "Math", v: MATH });
  return o;
}
const idxOf = (o, v) => Math.max(0, o.findIndex((x) => x.v === v));

function buildSide() {
  // ---------------------------------------------------- oscilloscope
  {
    const { body } = card("dev", "Oscilloscope");
    const name = el("div", "devname");
    name.dataset.help = "status_dev";
    const note = el("div", "devnote");
    body.append(name, note);
    onRefresh(() => {
      name.textContent = (INFO && INFO.name) || "Not connected";
      note.textContent = CONNECTED ? `${INFO.channels} channels · ${INFO.kind === "stream" ? "streams samples" : "triggers by itself"}` : "Use Connect…, or try Demo mode";
    });
    const r = el("div", "row");
    button(r, `<svg viewBox="0 0 24 24"><path d="M9 7V3M15 7V3M7 7h10v4a5 5 0 0 1-10 0zM12 16v5"/></svg>Connect…`, () => openConnect(), "connect", "btn sm");
    button(r, "Demo mode", () => connectDemo("stream"), "demo", "btn sm ghost");
    body.appendChild(r);
    const cal = row(body, "Cal output", "cal_freq");
    select(cal.f, () => ((INFO && INFO.calFreqs) || [1000]).map((f) => ({ t: fmtSI(f, "Hz"), v: f })),
      (o) => nearestIndex(o.map((x) => x.v), S.calFreq), (x) => { S.calFreq = x.v; }, "cal_freq");
    shown(cal.row, () => INFO && INFO.calOutput);
    const ac = toggle(body, "Hardware AC/DC mod installed", () => S.hwACDC, (v) => { S.hwACDC = v; }, "hw_acdc");
    shown(ac, () => INFO && INFO.acdcModOption);
    for (let c = 0; c < MAXCH; c++) {
      const d = row(body, `Demo CH${c + 1}`, "demo_sig");
      d.row.querySelector("label").style.color = COLORS[c];
      select(d.f, () => META.demoSignals.map((s) => ({ t: s.Label, v: s.Key, title: s.Tip })),
        (o) => idxOf(o, S.ch[c].demo), (x) => { S.ch[c].demo = x.v; }, "demo_sig");
      shown(d.row, () => INFO && INFO.simulated && c < nch());
    }
  }
  // ---------------------------------------------------- channels
  for (let c = 0; c < MAXCH; c++) {
    const { card: cd, body } = card("ch" + c, `Channel ${c + 1}`, COLORS[c]);
    shown(cd, () => c < nch());
    const t = el("div", "tgls");
    toggle(t, `Show CH${c + 1}`, () => S.ch[c].on, (v) => { S.ch[c].on = v; }, "ch_enable");
    toggle(t, "Invert", () => S.ch[c].invert, (v) => { S.ch[c].invert = v; }, "invert");
    body.appendChild(t);
    select(row(body, "Volts/div", "vdiv").f, () => ((INFO && INFO.vdivs) || []).map((v) => ({ t: fmtDiv(v * S.ch[c].probe, "V"), v })),
      (o) => nearestIndex(o.map((x) => x.v), S.ch[c].vdiv), (x) => { S.ch[c].vdiv = x.v; }, "vdiv");
    select(row(body, "Probe", "probe").f, () => [1, 10, 100].map((p) => ({ t: "x" + p, v: p })),
      (o) => idxOf(o, S.ch[c].probe), (x) => {
        // keep the same volts/div at the probe tip where possible
        const tip = vdivTip(c);
        S.ch[c].probe = x.v;
        const l = INFO.vdivs || [];
        if (l.length) S.ch[c].vdiv = l[nearestIndex(l, tip / x.v)];
        refreshControls();
      }, "probe");
    seg(row(body, "Coupling", "coupling").f, [["DC", "DC"], ["AC", "AC"]], () => S.ch[c].coupling, (v) => { S.ch[c].coupling = v; }, "coupling");
    const p = row(body, "Position", "pos");
    slider(p.f, -4, 4, 0.01, () => S.ch[c].pos, (v) => { S.ch[c].pos = v; }, "pos", COLORS[c]);
    button(p.f, "⟲", () => { S.ch[c].pos = 0; changed(); refreshControls(); }, "pos_reset", "icon-btn sm");
  }
  // ---------------------------------------------------- horizontal
  {
    const { body } = card("h", "Horizontal (time)");
    select(row(body, "Time/div", "tdiv").f, () => ((INFO && INFO.tdivs) || []).map((t) => ({ t: fmtDiv(t, "s"), v: t })),
      (o) => nearestIndex(o.map((x) => x.v), S.tdiv), (x) => { S.tdiv = x.v; }, "tdiv");
    slider(row(body, "Trigger pos.", "trig_pos").f, () => -hx(), () => hx(), 0.01, () => S.trigX, (v) => { S.trigX = v; }, "trig_pos", TRIG);
    const rr = row(body, "Sample rate", "rate");
    select(rr.f, () => [{ t: "Auto (recommended)", v: 0 }].concat(((INFO && INFO.sampleRates) || []).map((r) => ({ t: fmtSI(r, "S/s"), v: r }))),
      (o) => (S.rate > 0 ? nearestIndex(o.slice(1).map((x) => x.v), S.rate) + 1 : 0), (x) => { S.rate = x.v; }, "rate");
    shown(rr.row, isStream);
    const roll = toggle(body, "Roll mode at slow time/div", () => S.roll, (v) => { S.roll = v; }, "roll");
    shown(roll, () => isStream() && INFO.rollMode);
  }
  // ---------------------------------------------------- trigger
  {
    const { body } = card("trig", "Trigger", TRIG);
    seg(row(body, "Mode", "trig_mode").f, [["auto", "Auto"], ["normal", "Normal"]], () => S.trigMode, (v) => { S.trigMode = v; }, "trig_mode");
    select(row(body, "Source", "trig_source").f, () => chanOpts(false), (o) => idxOf(o, trigSrc()), (x) => { S.trigSource = x.v; }, "trig_source");
    seg(row(body, "Edge", "trig_edge").f, [["rising", "Rising ↑"], ["falling", "Falling ↓"], ["both", "Either ↕"]],
      () => S.trigEdge, (v) => { S.trigEdge = v; }, "trig_edge", (v) => v === "both" && INFO && !INFO.triggerBothEdges);
    const lv = row(body, "Level", "trig_level");
    const num = el("input", "inp mono");
    num.type = "number";
    num.dataset.help = "trig_level";
    num.onchange = () => { const v = parseFloat(num.value); if (isFinite(v)) { S.trigLevel = v; changed(); refreshControls(); } };
    onRefresh(() => {
      const src = trigSrc(), fs = (4 + Math.abs(S.ch[src].pos)) * vdivTip(src);
      num.step = String(Number((vdivTip(src) / 10).toPrecision(2)));
      num.min = String(-2 * fs); num.max = String(2 * fs);
      if (document.activeElement !== num) num.value = Number(S.trigLevel.toFixed(4));
    });
    lv.f.append(num, el("span", "muted", "V"));
    button(lv.f, "50 %", () => api("trig50", {}), "trig_50");
    const ho = row(body, "Holdoff", "holdoff");
    const hn = el("input", "inp mono");
    hn.type = "number"; hn.min = "0"; hn.max = "10000"; hn.step = "10";
    hn.onchange = () => { const v = parseFloat(hn.value); if (isFinite(v)) { S.holdoff = clamp(v, 0, 10000); changed(); } };
    onRefresh(() => { if (document.activeElement !== hn) hn.value = S.holdoff; });
    ho.f.append(hn, el("span", "muted", "µs"));
    shown(ho.row, () => isStream() && INFO.triggerHoldoff);
    const nr = toggle(body, "Noise reject", () => S.noiseReject, (v) => { S.noiseReject = v; }, "noise_reject");
    shown(nr, isStream);
  }
  // ---------------------------------------------------- math
  {
    const { body } = card("math", "Math", COLORS[MATH]);
    toggle(body, "Show math trace", () => S.mathOn, (v) => { S.mathOn = v; }, "math_enable");
    const f = row(body, "Formula", "math_op").f;
    select(f, () => chanOpts(false), (o) => idxOf(o, S.mathA), (x) => { S.mathA = x.v; }, "math_a");
    select(f, () => MATH_OPS.map((t, i) => ({ t: t.replace("A", "").replace("B", "").trim(), v: i })), (o) => S.mathOp, (x) => { S.mathOp = x.v; }, "math_op");
    select(f, () => chanOpts(false), (o) => idxOf(o, S.mathB), (x) => { S.mathB = x.v; }, "math_b");
    select(row(body, "Scale", "math_vdiv").f, () => META.mathScales.map((v) => ({ t: fmtDiv(v, "") + "/div", v })),
      (o) => nearestIndex(o.map((x) => x.v), S.mathVdiv), (x) => { S.mathVdiv = x.v; }, "math_vdiv");
    slider(row(body, "Position", "math_pos").f, -4, 4, 0.01, () => S.mathPos, (v) => { S.mathPos = v; }, "math_pos", COLORS[MATH]);
  }
  // ---------------------------------------------------- display
  {
    const { body } = card("disp", "Display");
    seg(row(body, "Style", "style").f, [["lines", "Lines"], ["dots", "Dots"]], () => S.style, (v) => { S.style = v; }, "style");
    seg(row(body, "Interpolation", "interp").f, [["sinc", "Sin(x)/x"], ["linear", "Linear"]], () => S.interp, (v) => { S.interp = v; }, "interp");
    select(row(body, "Averaging", "average").f, () => META.avgOpts.map((n) => ({ t: n === 1 ? "Off" : n + " waveforms", v: n })),
      (o) => idxOf(o, S.average), (x) => { S.average = x.v; }, "average");
    const pr = row(body, "Persistence", "persist");
    select(pr.f, () => META.persistOpts.map((p) => ({ t: p === 0 ? "Off" : p < 0 ? "Infinite" : p + " s", v: p })),
      (o) => idxOf(o, S.persist), (x) => { S.persist = x.v; }, "persist");
    button(pr.f, "Clear", () => api("clear", {}), "clear");
    slider(row(body, "Grid", "grid").f, 0, 100, 1, () => S.grid, (v) => { S.grid = v; }, "grid");
    toggle(body, "XY mode (CH1 → X, CH2 → Y)", () => S.xy, (v) => { S.xy = v; }, "xy");
  }
  // ---------------------------------------------------- spectrum
  {
    const { body } = card("fft", "Spectrum (FFT)");
    toggle(body, "Show spectrum", () => S.fftOn, (v) => { S.fftOn = v; layoutFFT(); }, "fft_enable");
    select(row(body, "Source", "fft_source").f, () => chanOpts(true), (o) => idxOf(o, S.fftSource), (x) => { S.fftSource = x.v; }, "fft_source");
    select(row(body, "Window", "fft_window").f, () => META.fftWindows.map((w) => ({ t: w, v: w })), (o) => idxOf(o, S.fftWindow), (x) => { S.fftWindow = x.v; }, "fft_window");
    toggle(body, "Logarithmic frequency axis", () => S.fftLog, (v) => { S.fftLog = v; fftView = null; }, "fft_log");
  }
  // ---------------------------------------------------- cursors
  {
    const { body } = card("cur", "Cursors", CUR);
    select(row(body, "Show", "cursor_mode").f, () => ["Off", "Time", "Voltage", "Time + Voltage"].map((t, i) => ({ t, v: i })),
      (o) => S.cursorMode, (x) => { S.cursorMode = x.v; }, "cursor_mode");
    select(row(body, "Channel", "cursor_src").f, () => chanOpts(true), (o) => idxOf(o, S.cursorSource), (x) => { S.cursorSource = x.v; }, "cursor_src");
  }
}

// ================================================================ chips, measurements, cursors

function updateChips() {
  if (!S || !INFO) return;
  const clipped = (STATUS && STATUS.clipped) || [];
  for (let c = 0; c < MAXCH; c++) {
    const chip = $("chipC" + c);
    chip.classList.toggle("hidden", c >= nch());
    chip.style.color = COLORS[c];
    chip.style.borderColor = "";
    chip.classList.remove("clip");
    if (!chOn(c)) { chip.textContent = `CH${c + 1} off`; chip.style.opacity = ".55"; continue; }
    chip.style.opacity = "1";
    let t = `CH${c + 1}  ${fmtDiv(vdivTip(c), "V")}/div  ${S.ch[c].coupling}  x${S.ch[c].probe}`;
    if (S.ch[c].invert) t += "  INV";
    if (clipped[c]) { t += "  ⚠ CLIPPED"; chip.classList.add("clip"); chip.title = "Signal clipped! The signal goes beyond what the scope can measure at this setting. Increase volts/div (or centre the trace with Position) to see the whole waveform."; }
    else chip.title = "";
    chip.textContent = t;
  }
  const m = $("chipM");
  m.classList.toggle("hidden", !S.mathOn);
  m.style.color = COLORS[MATH];
  m.textContent = `M  CH${S.mathA + 1} ${MATH_OPS[S.mathOp][2]} CH${S.mathB + 1}  ${fmtDiv(S.mathVdiv, "")}/div`;
  let h = `${fmtDiv(S.tdiv, "s")}/div`;
  const f = FR && FR.hdr.frame;
  if (f && f.rate) h += `   ${fmtSI(f.rate, "S/s")}   ${fmtSI(f.points, "", 3)} pts`;
  if (isRoll()) h += "   ROLL";
  $("chipH").textContent = h;
  $("chipH").style.color = "#e0e6ea";
  const src = trigSrc();
  $("chipT").style.color = TRIG;
  $("chipT").textContent = `Trig CH${src + 1} ${{ rising: "↑", falling: "↓", both: "↕" }[S.trigEdge]} ${fmtSI(S.trigLevel, "V")}  ${S.trigMode === "auto" ? "Auto" : "Normal"}`;
}

let measSig = "";
function buildMeasTable() {
  if (!S || !META) return;
  const n = nch();
  const sig = JSON.stringify([S.meas, n, S.mathOn]);
  if (sig === measSig) return;
  measSig = sig;
  const t = $("measTable");
  const defs = {};
  META.measurements.forEach((m) => { defs[m.Key] = m; });
  defs.phase = { Label: "Phase CH1→CH2", Tip: "How far CH2's waveform lags behind CH1's, as an angle (360° = one whole cycle). Both channels need the same frequency." };
  defs.delay = { Label: "Delay CH1→CH2", Tip: "Time from a CH1 rising edge to the next CH2 rising edge." };
  let html = "<thead><tr><th></th>";
  for (let c = 0; c < n; c++) html += `<th style="color:${COLORS[c]}">CH${c + 1}</th>`;
  html += `<th style="color:${COLORS[MATH]}">Math</th></tr></thead><tbody>`;
  if (!S.meas.length) html += `<tr><td class="empty" colspan="${n + 2}">No measurements chosen — click <b>Measurements…</b></td></tr>`;
  S.meas.forEach((k) => {
    const d = defs[k] || { Label: k, Tip: "" };
    html += `<tr data-k="${k}"><th title="${esc(d.Tip)}">${esc(d.Label)}</th>`;
    for (let c = 0; c < n; c++) html += `<td data-c="${c}">--</td>`;
    html += `<td data-c="${MATH}"></td></tr>`;
  });
  t.innerHTML = html + "</tbody>";
  updateMeasValues();
}
function updateMeasValues() {
  const m = (STATUS && STATUS.meas) || {};
  $("measTable").querySelectorAll("tbody tr[data-k]").forEach((tr) => {
    const vals = m[tr.dataset.k] || [];
    tr.querySelectorAll("td").forEach((td) => {
      const v = vals[+td.dataset.c] || (+td.dataset.c === MATH ? "" : "");
      if (td.textContent !== v) td.textContent = v;
    });
  });
}

function measPopover() {
  const pop = $("measPop");
  if (!pop.classList.contains("hidden")) { pop.classList.add("hidden"); return; }
  const items = META.measurements.map((m) => [m.Key, m.Label, m.Tip]).concat([
    ["phase", "Phase CH1→CH2", "How far CH2 lags behind CH1, as an angle."], ["delay", "Delay CH1→CH2", "Time from a CH1 rising edge to the next CH2 rising edge."]]);
  pop.innerHTML = "";
  items.forEach(([k, lab, tip]) => {
    const l = el("label", "", `<input type="checkbox" ${S.meas.includes(k) ? "checked" : ""}><span>${esc(lab)}</span>`);
    l.title = tip;
    l.querySelector("input").onchange = () => {
      const on = new Set(S.meas);
      if (l.querySelector("input").checked) on.add(k); else on.delete(k);
      S.meas = items.map((x) => x[0]).filter((x) => on.has(x));
      changed(true);
      buildMeasTable();
    };
    pop.appendChild(l);
  });
  const r = $("btnMeas").getBoundingClientRect();
  pop.style.left = Math.min(r.left, innerWidth - 270) + "px";
  pop.style.top = "auto";
  pop.style.bottom = innerHeight - r.top + 6 + "px";
  pop.classList.remove("hidden");
}
document.addEventListener("mousedown", (e) => {
  const pop = $("measPop");
  if (!pop.classList.contains("hidden") && !pop.contains(e.target) && e.target.id !== "btnMeas") pop.classList.add("hidden");
  document.querySelectorAll(".menu.open").forEach((m) => { if (!m.contains(e.target)) m.classList.remove("open"); });
});

function updateCursors() {
  if (!S) return;
  const out = $("curRead");
  const mode = S.cursorMode;
  if (mode === 0) { out.innerHTML = `<span class="off">Cursors are off.<br>Turn them on in the <b style="color:var(--tx)">Cursors</b> section on the right.</span>`; return; }
  const lines = [];
  if ((mode === 1 || mode === 3) && !S.xy) {
    const ref = isRoll() ? hx() : S.trigX;
    const ta = (S.cursors[0] - ref) * S.tdiv, tb = (S.cursors[1] - ref) * S.tdiv, dt = tb - ta;
    lines.push(`A:  ${fmtSI(ta, "s")}`, `B:  ${fmtSI(tb, "s")}`, `<b>ΔT: ${fmtSI(dt, "s")}</b>`, `<b>1/ΔT: ${dt ? fmtSI(1 / Math.abs(dt), "Hz") : "--"}</b>`);
  }
  if (mode === 2 || mode === 3) {
    let src = S.cursorSource;
    if (S.xy) src = 1;
    let toV, unit = "V";
    if (src < nch()) toV = (y) => (y - S.ch[src].pos) * vdivTip(src) * sign(src);
    else { toV = (y) => (y - S.mathPos) * S.mathVdiv; if (S.mathOp === 2) unit = "V²"; }
    const va = toV(S.cursors[2]), vb = toV(S.cursors[3]);
    if (lines.length) lines.push("");
    lines.push(`C:  ${fmtSI(va, unit)}`, `D:  ${fmtSI(vb, unit)}`, `<b>ΔV: ${fmtSI(vb - va, unit)}</b>`);
  }
  const h = lines.join("\n");
  if (out._h !== h) { out.innerHTML = h; out._h = h; }
}

// ================================================================ status

function applyStatus(st) {
  STATUS = st;
  const run = $("btnRun");
  run.classList.toggle("stopped", !st.running);
  run.querySelector("span").textContent = st.running ? "Stop" : "Run";
  const dot = $("devChip").querySelector(".dot");
  dot.className = "dot" + (st.state === "connecting" ? " busy" : st.connected && st.running ? " on" : "");
  const dev = st.state === "connecting" ? (st.message || "Connecting…") : st.device;
  $("devChip").querySelector(".lbl").textContent = st.imported ? "File: " + st.imported : dev;
  $("stDev").textContent = st.state === "connecting" ? dev : st.connected ? st.device : "Not connected - use Connect…, or Demo mode";
  $("stDev").title = st.notes || "";
  $("stWfm").textContent = Math.round(st.wfmRate).toLocaleString() + " wfm/s";
  $("stUsb").textContent = st.mbps.toFixed(2) + " MB/s";
  $("stFps").textContent = fps.toFixed(0) + " fps";
  document.title = "Scopie — " + (st.connected ? st.device : "not connected");
  if (st.connected !== CONNECTED) { CONNECTED = st.connected; refreshControls(); }
  updateMeasValues();
  updateChips();
  updateScreenMsg();
}

function updateScreenMsg() {
  const st = STATUS;
  let msg = "";
  if (st) {
    if (st.state === "disconnected" && !st.imported) msg = `No scope connected<small>Click Connect… or Demo mode</small>`;
    else if (st.state === "connecting") msg = esc(st.message || "Connecting…");
    else if (st.state === "waiting" && !(FR && FR.traces && FR.traces.length)) msg = "Waiting for trigger…";
  }
  const m = $("screenMsg");
  if (m._h !== msg) { m.innerHTML = msg; m._h = msg; }
  if (FR) {
    const cs = $("chipState");
    cs.textContent = FR.hdr.chip;
    cs.style.background = FR.hdr.chipColor;
  }
}

// ================================================================ server events

function connectSSE() {
  const es = new EventSource("/api/events");
  es.addEventListener("status", (e) => applyStatus(JSON.parse(e.data)));
  es.addEventListener("settings", (e) => { S = JSON.parse(e.data); refreshControls(); });
  es.addEventListener("hello", redraw);
  es.addEventListener("info", (e) => {
    const d = JSON.parse(e.data);
    INFO = d.info; CONNECTED = d.connected;
    fftView = null;
    refreshControls();
  });
  es.addEventListener("error", (e) => { if (e.data) { const d = JSON.parse(e.data); errorBox(d.message); } });
  es.addEventListener("help", (e) => stickyHelp(JSON.parse(e.data).html));
}

let errOpen = null;
function errorBox(msg) {
  if (errOpen && document.body.contains(errOpen.el)) errOpen.close();
  errOpen = message("Oscilloscope", msg);
}

// ================================================================ frame loop

let inflight = false;
async function poll() {
  if (inflight) return;
  inflight = true;
  try {
    const r = await fetch("/api/frame?have=" + seq, { headers: { "X-Scopie": "1" } });
    if (r.status === 200) {
      FR = parseFrame(await r.arrayBuffer());
      seq = FR.hdr.seq;
      frames++;
      needDraw = true;
    }
  } catch { /* server restarting or closed */ }
  inflight = false;
}

function parseFrame(buf) {
  const dv = new DataView(buf);
  const hl = dv.getUint32(0, true);
  const hdr = JSON.parse(new TextDecoder().decode(new Uint8Array(buf, 4, hl)));
  const base = 4 + hl;
  const f32 = (off, n) => new Float32Array(buf, base + off, n);
  const fr = { hdr, traces: [], persist: [], xy: null, fft: null };
  (hdr.traces || []).forEach((t) => fr.traces.push({ c: t.c, dots: t.dots, x: f32(t.x, t.n), y: f32(t.y, t.n) }));
  (hdr.persist || []).forEach((p) => fr.persist.push(Object.assign({}, p, { img: persistCanvas(p.c, new Uint8Array(buf, base + p.off, p.cols * p.bins), p.cols, p.bins) })));
  if (hdr.xy) fr.xy = { x: f32(hdr.xy.x, hdr.xy.n), y: f32(hdr.xy.y, hdr.xy.n) };
  if (hdr.fft) fr.fft = Object.assign({}, hdr.fft, { f: f32(hdr.fft.f, hdr.fft.n), db: f32(hdr.fft.db, hdr.fft.n) });
  if (hdr.hdivs && INFO && INFO.hdivs !== hdr.hdivs) INFO.hdivs = hdr.hdivs;
  return fr;
}

// persistence colour maps: black -> channel colour -> white-ish
const LUTS = COLORS.map((hex) => {
  const r = parseInt(hex.slice(1, 3), 16), g = parseInt(hex.slice(3, 5), 16), b = parseInt(hex.slice(5, 7), 16);
  const L = new Uint8ClampedArray(256 * 4);
  for (let i = 0; i < 256; i++) {
    const x = i / 255, base = Math.min(1, x * 1.6), w = clamp((x - 0.75) * 3, 0, 1) * 0.6;
    L[i * 4] = r * base + 255 * w; L[i * 4 + 1] = g * base + 255 * w; L[i * 4 + 2] = b * base + 255 * w; L[i * 4 + 3] = 255;
  }
  L[0] = L[1] = L[2] = 0;
  return L;
});
const PCANV = [];
function persistCanvas(c, img, cols, bins) {
  let cv = PCANV[c];
  if (!cv || cv.width !== cols || cv.height !== bins) { cv = PCANV[c] = document.createElement("canvas"); cv.width = cols; cv.height = bins; cv._id = null; }
  const ctx = cv.getContext("2d");
  const id = ctx.createImageData(cols, bins);
  const d = id.data, L = LUTS[c];
  for (let col = 0; col < cols; col++) {
    const o = col * bins;
    for (let b = 0; b < bins; b++) {
      const v = img[o + b], q = (b * cols + col) * 4, l = v * 4;
      d[q] = L[l]; d[q + 1] = L[l + 1]; d[q + 2] = L[l + 2]; d[q + 3] = 255;
    }
  }
  ctx.putImageData(id, 0, 0);
  return cv;
}

let lastW = 0, lastH = 0;
function loop() {
  poll();
  const sw = $("scopeWrap"), w = sw.clientWidth, h = sw.clientHeight + $("fftWrap").clientHeight;
  if (w !== lastW || h !== lastH) { lastW = w; lastH = h; needDraw = true; }
  if (needDraw) {
    needDraw = false;
    draw();
    drawFFT();
  }
  const now = performance.now();
  if (now - fpsT > 1000) { fps = frames * 1000 / (now - fpsT); frames = 0; fpsT = now; }
  requestAnimationFrame(loop);
}

// ================================================================ scope drawing

const cv = $("scope"), ctx = cv.getContext("2d");
let R = { x: 0, y: 0, w: 1, h: 1 }, DPR = 1, hover = null, drag = null;
const MARG = { l: 60, r: 60, t: 10, b: 26 };
const FONT = getComputedStyle(document.body).getPropertyValue("--font");
const MONO = getComputedStyle(document.body).getPropertyValue("--mono");

function sizeCanvas(c) {
  DPR = window.devicePixelRatio || 1;
  const w = Math.max(1, Math.round(c.clientWidth * DPR)), h = Math.max(1, Math.round(c.clientHeight * DPR));
  if (c.width !== w || c.height !== h) { c.width = w; c.height = h; }
  return [c.clientWidth, c.clientHeight];
}
const X = (d) => R.x + (d + hx()) / (2 * hx()) * R.w;
const Y = (d) => R.y + (VY - d) / (2 * VY) * R.h;
const DX = (px) => (px - R.x) / R.w * 2 * hx() - hx();
const DY = (py) => VY - (py - R.y) / R.h * 2 * VY;

function draw() {
  if (!S || !INFO) return;
  const [w, h] = sizeCanvas(cv);
  R = { x: MARG.l, y: MARG.t, w: Math.max(10, w - MARG.l - MARG.r), h: Math.max(10, h - MARG.t - MARG.b) };
  ctx.setTransform(DPR, 0, 0, DPR, 0, 0);
  ctx.fillStyle = "#05080a";
  ctx.fillRect(0, 0, w, h);
  drawGrid(w, h);
  ctx.save();
  ctx.beginPath();
  ctx.rect(R.x, R.y, R.w, R.h);
  ctx.clip();
  if (FR) drawTraces();
  drawMarkers();
  ctx.restore();
  drawAxisLabels(w, h);
  drawMarkerLabels();
}

function drawGrid() {
  const H = hx(), a = 20 + 2 * S.grid;
  ctx.lineWidth = 1;
  ctx.strokeStyle = `rgba(110,130,140,${Math.min(255, a / 2) / 255})`;
  ctx.setLineDash([1, 3]);
  ctx.beginPath();
  for (let i = -Math.floor(H); i <= Math.floor(H); i++) { const x = Math.round(X(i)) + 0.5; ctx.moveTo(x, R.y); ctx.lineTo(x, R.y + R.h); }
  for (let j = -VY; j <= VY; j++) { const y = Math.round(Y(j)) + 0.5; ctx.moveTo(R.x, y); ctx.lineTo(R.x + R.w, y); }
  ctx.stroke();
  ctx.setLineDash([]);
  ctx.strokeStyle = `rgba(140,160,170,${Math.min(255, a) / 255})`;
  ctx.beginPath();
  const cx = Math.round(X(0)) + 0.5, cy = Math.round(Y(0)) + 0.5;
  ctx.moveTo(R.x, cy); ctx.lineTo(R.x + R.w, cy); ctx.moveTo(cx, R.y); ctx.lineTo(cx, R.y + R.h);
  const tk = 0.08 * R.h / (2 * VY);
  for (let t = -H; t <= H + 0.001; t += 0.2) { const x = X(t); ctx.moveTo(x, cy - tk); ctx.lineTo(x, cy + tk); }
  for (let t = -VY; t <= VY + 0.001; t += 0.2) { const y = Y(t); ctx.moveTo(cx - tk, y); ctx.lineTo(cx + tk, y); }
  ctx.stroke();
  ctx.strokeStyle = "#1f2a31";
  ctx.strokeRect(R.x + 0.5, R.y + 0.5, R.w - 1, R.h - 1);
}

function axisChannels() {
  if (S.xy) return [null, 1];
  const on = [];
  for (let c = 0; c < nch(); c++) if (chOn(c)) on.push(c);
  return [on.length ? on[0] : null, on.length > 1 ? on[1] : null];
}

function drawAxisLabels(w, h) {
  ctx.font = "11px " + MONO;
  ctx.textBaseline = "middle";
  const H = Math.floor(hx());
  ctx.fillStyle = "#b8c4ca";
  ctx.textAlign = "center";
  for (let i = -H; i <= H; i++) {
    let t;
    if (S.xy) t = fmtSI((i - S.ch[0].pos) * vdivTip(0) * sign(0), "V", 3);
    else if (isRoll()) t = fmtSI((i - hx()) * S.tdiv, "s", 3);
    else t = fmtSI((i - S.trigX) * S.tdiv, "s", 3);
    let x = X(i);
    if (i === -H) { ctx.textAlign = "left"; x = Math.max(x - 14, 2); } else if (i === H) { ctx.textAlign = "right"; x = Math.min(x + 14, w - 2); } else ctx.textAlign = "center";
    ctx.fillText(t, x, R.y + R.h + 13);
  }
  const [lc, rc] = axisChannels();
  [[lc, R.x - 6, "right"], [rc, R.x + R.w + 6, "left"]].forEach(([c, x, al]) => {
    if (c == null) return;
    ctx.fillStyle = COLORS[c];
    ctx.textAlign = al;
    for (let j = -VY; j <= VY; j++) ctx.fillText(fmtSI((j - S.ch[c].pos) * vdivTip(c) * sign(c), "V", 3), x, clamp(Y(j), R.y + 6, R.y + R.h - 6));
  });
}

function drawTraces() {
  const f = FR;
  if (S.xy) {
    if (f.xy) {
      ctx.strokeStyle = XYCOL;
      ctx.lineWidth = 1.2;
      ctx.beginPath();
      const { x, y } = f.xy;
      for (let i = 0; i < x.length; i++) { const px = X(x[i]), py = Y(y[i]); i ? ctx.lineTo(px, py) : ctx.moveTo(px, py); }
      ctx.stroke();
    }
    return;
  }
  ctx.save();
  ctx.globalCompositeOperation = "lighter";
  ctx.imageSmoothingEnabled = true;
  f.persist.forEach((p) => {
    ctx.save();
    const kx = R.w / (2 * hx()), ky = R.h / (2 * VY);
    ctx.transform(p.sx * kx, 0, 0, -p.sy * ky, X(p.x0), Y(p.y0));
    ctx.drawImage(p.img, 0, 0);
    ctx.restore();
  });
  ctx.restore();
  f.traces.forEach((t) => {
    const col = COLORS[t.c];
    const { x, y } = t;
    if (t.dots) {
      ctx.fillStyle = col;
      for (let i = 0; i < x.length; i++) ctx.fillRect(X(x[i]) - 1, Y(y[i]) - 1, 2.4, 2.4);
      return;
    }
    ctx.beginPath();
    for (let i = 0; i < x.length; i++) { const px = X(x[i]), py = Y(y[i]); i ? ctx.lineTo(px, py) : ctx.moveTo(px, py); }
    if (x.length < 1500) { // a soft glow looks lovely on clean traces; dense ones don't need it
      ctx.lineJoin = "round";
      ctx.strokeStyle = col + "30";
      ctx.lineWidth = 4;
      ctx.stroke();
    }
    ctx.lineJoin = "bevel";
    ctx.strokeStyle = col;
    ctx.lineWidth = 1.5;
    ctx.stroke();
  });
}

// markers: everything draggable on the screen
function markerList() {
  const out = [];
  if (!S.xy) {
    for (let c = 0; c < nch(); c++) if (chOn(c)) out.push({ kind: "gnd", c, horiz: true, pos: S.ch[c].pos, col: COLORS[c], help: "ground_marker" });
    if (S.mathOn) out.push({ kind: "gnd", c: MATH, horiz: true, pos: S.mathPos, col: COLORS[MATH], help: "ground_marker" });
    if (!isRoll()) {
      const src = trigSrc();
      out.push({ kind: "tlevel", horiz: true, pos: clamp(S.trigLevel * sign(src) / vdivTip(src) + S.ch[src].pos, -VY, VY), col: TRIG, help: "trig_line" });
      out.push({ kind: "tpos", horiz: false, pos: S.trigX, col: TRIG, help: "trig_pos_line" });
    }
  }
  const m = S.cursorMode;
  if ((m === 1 || m === 3) && !S.xy) [0, 1].forEach((i) => out.push({ kind: "cur", i, horiz: false, pos: S.cursors[i], col: CUR, help: "cursor_line", name: "AB"[i] }));
  if (m === 2 || m === 3) [2, 3].forEach((i) => out.push({ kind: "cur", i, horiz: true, pos: S.cursors[i], col: CUR, help: "cursor_line", name: "CD"[i - 2] }));
  return out;
}
const same = (a, b) => a && b && a.kind === b.kind && a.c === b.c && a.i === b.i;

function drawMarkers() {
  markerList().forEach((m) => {
    const active = same(m, hover) || same(m, drag);
    ctx.lineWidth = active ? 2 : 1;
    if (m.kind === "gnd") { ctx.strokeStyle = active ? m.col : m.col + "46"; ctx.setLineDash([5, 4]); }
    else if (m.kind === "tlevel") { ctx.strokeStyle = m.col; ctx.setLineDash([5, 4]); }
    else if (m.kind === "tpos") { ctx.strokeStyle = active ? m.col : m.col + "6e"; ctx.setLineDash([]); }
    else { ctx.strokeStyle = m.col; ctx.setLineDash([7, 3, 2, 3]); }
    ctx.beginPath();
    if (m.horiz) { const y = Math.round(Y(m.pos)) + 0.5; ctx.moveTo(R.x, y); ctx.lineTo(R.x + R.w, y); }
    else { const x = Math.round(X(m.pos)) + 0.5; ctx.moveTo(x, R.y); ctx.lineTo(x, R.y + R.h); }
    ctx.stroke();
  });
  ctx.setLineDash([]);
}

function drawMarkerLabels() {
  ctx.font = "bold 12px " + FONT;
  ctx.textBaseline = "middle";
  markerList().forEach((m) => {
    ctx.fillStyle = m.col;
    if (m.kind === "gnd") {
      const y = clamp(Y(m.pos), R.y + 7, R.y + R.h - 7);
      const t = (m.c === MATH ? "M" : String(m.c + 1)) + " ▸";
      ctx.textAlign = "left";
      ctx.fillStyle = "#05080acc";
      ctx.fillRect(R.x + 2, y - 8, 26, 16);
      ctx.fillStyle = m.col;
      ctx.fillText(t, R.x + 5, y);
    } else if (m.kind === "tlevel") {
      ctx.textAlign = "right";
      ctx.fillText("T ◂", R.x + R.w - 4, clamp(Y(m.pos), R.y + 7, R.y + R.h - 7));
    } else if (m.kind === "tpos") {
      ctx.textAlign = "center";
      ctx.fillText("▼T", X(m.pos), R.y + 8);
    } else if (m.horiz) {
      ctx.textAlign = "left";
      ctx.fillText(m.name, R.x + R.w * 0.12, Y(m.pos) - 9);
    } else {
      ctx.textAlign = "left";
      ctx.fillText(m.name, X(m.pos) + 4, R.y + R.h * 0.94);
    }
  });
}

function hitTest(px, py) {
  let best = null, bd = 7;
  const pri = { cur: 0, tpos: 1, tlevel: 1, gnd: 2 };
  markerList().forEach((m) => {
    let d;
    if (m.horiz) {
      d = Math.abs(py - Y(m.pos));
      if (m.kind === "gnd" && px < R.x + 30) d -= 3; // the label is easy to grab
    } else d = Math.abs(px - X(m.pos));
    if (d < bd || (best && d === bd && pri[m.kind] < pri[best.kind])) { bd = d; best = m; }
  });
  return best;
}

function dragTo(m, px, py) {
  const r2 = (v) => Math.round(v * 100) / 100;
  if (m.horiz) {
    const d = clamp(DY(py), -VY, VY);
    if (m.kind === "gnd") { if (m.c === MATH) S.mathPos = r2(d); else S.ch[m.c].pos = r2(d); }
    else if (m.kind === "tlevel") { const src = trigSrc(); S.trigLevel = (d - S.ch[src].pos) * vdivTip(src) * sign(src); }
    else S.cursors[m.i] = d;
  } else {
    const d = clamp(DX(px), -hx(), hx());
    if (m.kind === "tpos") S.trigX = r2(d); else S.cursors[m.i] = d;
  }
  changed();
  REFRESH.forEach((f) => f());
}

cv.addEventListener("mousemove", (e) => {
  needDraw = true;
  const r = cv.getBoundingClientRect(), px = e.clientX - r.left, py = e.clientY - r.top;
  if (drag) { dragTo(drag, px, py); return; }
  hover = hitTest(px, py);
  cv.style.cursor = hover ? (hover.horiz ? "ns-resize" : "ew-resize") : "default";
  showHelp(hover ? hover.help : "screen");
});
cv.addEventListener("mousedown", (e) => {
  if (e.button !== 0) return;
  const r = cv.getBoundingClientRect();
  drag = hitTest(e.clientX - r.left, e.clientY - r.top);
  if (drag) e.preventDefault();
});
window.addEventListener("mouseup", () => { if (drag) { drag = null; changed(true); } });
cv.addEventListener("mouseleave", () => { hover = null; needDraw = true; if (!drag) showHelp(null); });

// ================================================================ spectrum drawing

const fcv = $("fft"), fctx = fcv.getContext("2d");
let fftView = null, fftDrag = null;
const FM = { l: 64, r: 40, t: 10, b: 26 };
let FR2 = { x: 0, y: 0, w: 1, h: 1 };

function layoutFFT() {
  const on = S && S.fftOn;
  $("fftWrap").classList.toggle("hidden", !on);
  $("splitter").classList.toggle("hidden", !on);
}
function fftX(f) {
  const v = fftView, t = S.fftLog ? Math.log10(Math.max(f, 1e-12)) : f;
  return FR2.x + (t - v.x0) / (v.x1 - v.x0) * FR2.w;
}
const fftY = (db) => FR2.y + (fftView.y1 - db) / (fftView.y1 - fftView.y0) * FR2.h;
function niceStep(span, n) {
  const raw = span / n, p = Math.pow(10, Math.floor(Math.log10(raw))), m = raw / p;
  return (m < 1.5 ? 1 : m < 3.5 ? 2 : m < 7.5 ? 5 : 10) * p;
}

function drawFFT() {
  if (!S || !S.fftOn) return;
  const [w, h] = sizeCanvas(fcv);
  fctx.setTransform(DPR, 0, 0, DPR, 0, 0);
  fctx.fillStyle = "#05080a";
  fctx.fillRect(0, 0, w, h);
  FR2 = { x: FM.l, y: FM.t, w: Math.max(10, w - FM.l - FM.r), h: Math.max(10, h - FM.t - FM.b) };
  const sp = FR && FR.fft;
  if (!sp || sp.f.length < 2) {
    fctx.fillStyle = "#5f6c80"; fctx.font = "13px " + FONT;
    fctx.textAlign = "center"; fctx.fillText("Spectrum: waiting for a waveform…", w / 2, h / 2);
    return;
  }
  const fmin = sp.f[0], fmax = sp.f[sp.f.length - 1];
  if (!fftView || fftView.fmax !== fmax || fftView.log !== S.fftLog) {
    fftView = S.fftLog ? { x0: Math.log10(Math.max(fmin, 1e-3)), x1: Math.log10(fmax) } : { x0: 0, x1: fmax };
    Object.assign(fftView, { y0: -100, y1: 20, fmax, log: S.fftLog });
  }
  const v = fftView, mono = MONO;
  fctx.font = "11px " + mono;
  fctx.lineWidth = 1;
  fctx.strokeStyle = "rgba(140,160,170,.16)";
  fctx.fillStyle = "#b8c4ca";
  fctx.textBaseline = "middle";
  fctx.beginPath();
  const ys = niceStep(v.y1 - v.y0, 6);
  fctx.textAlign = "right";
  for (let d = Math.ceil(v.y0 / ys) * ys; d <= v.y1 + 1e-9; d += ys) {
    const y = Math.round(fftY(d)) + 0.5;
    fctx.moveTo(FR2.x, y); fctx.lineTo(FR2.x + FR2.w, y);
    fctx.fillText(gfmt(d, 4) + " dBV", FR2.x - 5, y);
  }
  fctx.textAlign = "center";
  if (S.fftLog) {
    for (let e = Math.floor(v.x0); e <= Math.ceil(v.x1); e++) {
      for (let k = 1; k < 10; k++) {
        const lf = e + Math.log10(k);
        if (lf < v.x0 || lf > v.x1) continue;
        const x = Math.round(FR2.x + (lf - v.x0) / (v.x1 - v.x0) * FR2.w) + 0.5;
        if (k === 1) { fctx.moveTo(x, FR2.y); fctx.lineTo(x, FR2.y + FR2.h); fctx.fillText(fmtSI(Math.pow(10, e), "Hz", 3), x, FR2.y + FR2.h + 13); }
      }
    }
  } else {
    const xs = niceStep(v.x1 - v.x0, 8);
    for (let f = Math.ceil(v.x0 / xs) * xs; f <= v.x1 + 1e-9; f += xs) {
      const x = Math.round(fftX(f)) + 0.5;
      fctx.moveTo(x, FR2.y); fctx.lineTo(x, FR2.y + FR2.h);
      fctx.fillText(fmtSI(f, "Hz", 3), x, FR2.y + FR2.h + 13);
    }
  }
  fctx.stroke();
  fctx.save();
  fctx.beginPath();
  fctx.rect(FR2.x, FR2.y, FR2.w, FR2.h);
  fctx.clip();
  fctx.strokeStyle = sp.color;
  fctx.lineWidth = 1.2;
  fctx.beginPath();
  for (let i = 0; i < sp.f.length; i++) { const x = fftX(sp.f[i]), y = fftY(sp.db[i]); i ? fctx.lineTo(x, y) : fctx.moveTo(x, y); }
  fctx.stroke();
  const px = fftX(sp.peakF), py = fftY(sp.peakDb);
  fctx.strokeStyle = "#fff";
  fctx.beginPath(); fctx.arc(px, py, 4, 0, 7); fctx.stroke();
  fctx.fillStyle = "#fff";
  fctx.textAlign = px > FR2.x + FR2.w - 200 ? "right" : "left";
  fctx.font = "12px " + mono;
  fctx.fillText(`peak ${fmtSI(sp.peakF, "Hz")}  ${sp.peakDb.toFixed(1)} dBV`, px + (fctx.textAlign === "left" ? 8 : -8), Math.max(FR2.y + 8, py - 12));
  fctx.restore();
  fctx.strokeStyle = "#1f2a31";
  fctx.strokeRect(FR2.x + 0.5, FR2.y + 0.5, FR2.w - 1, FR2.h - 1);
}
fcv.addEventListener("wheel", (e) => {
  if (!fftView) return;
  needDraw = true;
  e.preventDefault();
  const r = fcv.getBoundingClientRect(), k = e.deltaY > 0 ? 1.25 : 0.8;
  if (e.ctrlKey || e.shiftKey) {
    const at = fftView.y1 - (e.clientY - r.top - FR2.y) / FR2.h * (fftView.y1 - fftView.y0);
    fftView.y0 = at + (fftView.y0 - at) * k; fftView.y1 = at + (fftView.y1 - at) * k;
  } else {
    const at = fftView.x0 + (e.clientX - r.left - FR2.x) / FR2.w * (fftView.x1 - fftView.x0);
    fftView.x0 = at + (fftView.x0 - at) * k; fftView.x1 = at + (fftView.x1 - at) * k;
  }
}, { passive: false });
fcv.addEventListener("mousedown", (e) => { if (fftView) fftDrag = { x: e.clientX, y: e.clientY, v: Object.assign({}, fftView) }; });
window.addEventListener("mousemove", (e) => {
  if (!fftDrag) return;
  needDraw = true;
  const dx = (e.clientX - fftDrag.x) / FR2.w * (fftDrag.v.x1 - fftDrag.v.x0), dy = (e.clientY - fftDrag.y) / FR2.h * (fftDrag.v.y1 - fftDrag.v.y0);
  Object.assign(fftView, { x0: fftDrag.v.x0 - dx, x1: fftDrag.v.x1 - dx, y0: fftDrag.v.y0 + dy, y1: fftDrag.v.y1 + dy });
});
window.addEventListener("mouseup", () => { fftDrag = null; });
fcv.addEventListener("dblclick", () => { fftView = null; needDraw = true; });

// splitter between the scope and the spectrum
{
  let sd = null;
  $("splitter").addEventListener("mousedown", (e) => { sd = { y: e.clientY, a: $("scopeWrap").offsetHeight, b: $("fftWrap").offsetHeight }; e.preventDefault(); });
  window.addEventListener("mousemove", (e) => {
    if (!sd) return;
    const d = e.clientY - sd.y, a = Math.max(140, sd.a + d), b = Math.max(100, sd.b - d);
    $("scopeWrap").style.flex = `${a} 1 0`; $("fftWrap").style.flex = `${b} 1 0`;
  });
  window.addEventListener("mouseup", () => { sd = null; });
}

// ================================================================ actions

async function toggleRun() { try { await api("run", {}); } catch (e) { toast(e.message, "err"); } }
async function single() { await api("single", {}); }
async function autoset() {
  const r = await api("autoset", {});
  if (r.message) stickyHelp(esc(r.message));
}
async function connectDemo(kind) {
  await api("connect", { demo: kind });
  toast(kind === "bench" ? "Demo mode: 4-channel bench scope" : "Demo mode: simulated USB scope");
}

async function explainProbie() {
  try {
    const r = await api("explain-probie", {});
    toast(`Opening the waveform in Probie…\n${r.path}`);
  } catch (e) { message("Explain in Probie", e.message); }
}

async function exportCSV() {
  try {
    const r = await api("export-csv", {});
    toast(`Exported ${r.samples.toLocaleString()} samples to\n${r.path}`, "", showFolder);
  } catch (e) { message("Export", e.message); }
}

function screenshot() {
  const chips = [...document.querySelectorAll("#chips .schip:not(.hidden), #chipState")];
  const ch = 34 * DPR;
  const showF = S.fftOn;
  const out = document.createElement("canvas");
  out.width = cv.width;
  out.height = ch + cv.height + (showF ? fcv.height : 0);
  const c = out.getContext("2d");
  c.fillStyle = "#05080a";
  c.fillRect(0, 0, out.width, out.height);
  c.font = `600 ${12 * DPR}px ${MONO}`;
  c.textBaseline = "middle";
  let x = 8 * DPR;
  chips.forEach((chip) => {
    const t = chip.textContent, w = c.measureText(t).width + 18 * DPR;
    const isState = chip.id === "chipState";
    if (isState) x = Math.max(x, out.width - w - 8 * DPR);
    c.fillStyle = isState ? chip.style.background || "#888" : "#11181c";
    c.fillRect(x, 5 * DPR, w, 24 * DPR);
    c.fillStyle = isState ? "#000" : chip.style.color || "#fff";
    c.fillText(t, x + 9 * DPR, 17 * DPR);
    x += w + 6 * DPR;
  });
  c.drawImage(cv, 0, ch);
  if (showF) c.drawImage(fcv, 0, ch + cv.height);
  const data = out.toDataURL("image/png").split(",")[1];
  api("save-file", { name: `scope-${stamp()}.png`, data, base64: true })
    .then((r) => toast("Screenshot saved to\n" + r.path, "", showFolder))
    .catch((e) => toast(e.message, "err"));
}

function readFile(file, asText) {
  return new Promise((res, rej) => {
    const r = new FileReader();
    r.onload = () => res(asText ? r.result : r.result.split(",")[1]);
    r.onerror = () => rej(r.error);
    if (asText) r.readAsText(file); else r.readAsDataURL(file);
  });
}
async function openCSVFile(file) {
  try {
    await api("open-csv", { name: file.name, text: await readFile(file, true) });
    toast(`Showing ${file.name} - press Run to go back to live data`);
  } catch (e) { message("Open waveform", e.message); }
}
$("fileCSV").onchange = (e) => { const f = e.target.files[0]; e.target.value = ""; if (f) openCSVFile(f); };
$("fileSetup").onchange = async (e) => {
  const f = e.target.files[0]; e.target.value = "";
  if (!f) return;
  try { await api("load-setup", { text: await readFile(f, true) }); toast("Setup loaded: " + f.name); }
  catch (er) { message("Load setup", er.message); }
};
$("fileFW").onchange = async (e) => {
  const f = e.target.files[0]; e.target.value = "";
  if (!f) return;
  try {
    const r = await api("import-firmware", { name: f.name, data: await readFile(f, false) });
    message("Import firmware", "Firmware saved for: " + r.saved.join(", "));
    if (connectDlg) startScan();
  } catch (er) { message("Import firmware", er.message); }
};
async function saveSetup() {
  const s = await api("settings");
  const r = await api("save-file", { name: "scope-setup.json", data: JSON.stringify(s, null, 1) });
  toast("Setup saved to\n" + r.path, "", showFolder);
}

// drop a CSV file onto the screen
{
  const w = $("scopeWrap");
  ["dragenter", "dragover"].forEach((t) => w.addEventListener(t, (e) => { e.preventDefault(); w.classList.add("dragging"); }));
  ["dragleave", "drop"].forEach((t) => w.addEventListener(t, (e) => { e.preventDefault(); w.classList.remove("dragging"); }));
  w.addEventListener("drop", (e) => { const f = e.dataTransfer.files[0]; if (f) openCSVFile(f); });
}

// ================================================================ connect window

let connectDlg = null, cands = [], selIdx = -1;
function openConnect(pre) {
  if (connectDlg && document.body.contains(connectDlg.el)) return;
  const body = el("div");
  body.innerHTML = `
    <p style="margin-top:0">Plug your scope in and wait a few seconds. Supported scopes found on USB are listed below. Bench scopes can also be reached over the network: type their IP address.</p>
    <div class="clist" id="cList" data-help="device_list"></div>
    <div class="cdetail" id="cDetail"></div>
    <div class="lanrow" data-help="lan"><span style="white-space:nowrap">Network scope (IP):</span>
      <input class="inp mono" id="cLAN" placeholder="e.g. 192.168.1.50   (several: separate with commas)">
      <button class="btn" id="cScan" data-help="scan">Search again</button></div>
    <div class="toolsrow">
      <button class="btn sm" id="cFW" data-help="import_fw">Import firmware…</button>
      <button class="btn sm win-only" id="cDrv" data-help="usb_driver">Install USB driver…</button>
      <button class="btn sm ghost" id="cPico" data-help="pico_sdk">Get PicoScope drivers…</button>
    </div>
    <div class="cnotes" id="cNotes"></div>`;
  const btns = [{ label: "Cancel", ghost: true }, { label: "Connect", primary: true, fn: () => connectSelected() }];
  connectDlg = modal("Connect to an oscilloscope", body, btns, "");
  connectDlg.connectBtn = btns[1].el;
  $("cLAN").value = S.lan || "";
  $("cScan").onclick = startScan;
  $("cLAN").onkeydown = (e) => { if (e.key === "Enter") startScan(); };
  $("cFW").onclick = () => $("fileFW").click();
  $("cDrv").onclick = driverChooser;
  $("cPico").onclick = () => api("driver", { kind: "pico" });
  if (META.os !== "windows") $("cDrv").classList.add("hidden");
  if (pre && pre.candidates) showCands(pre.candidates, pre.notes); else startScan();
}
async function startScan() {
  const list = $("cList");
  if (!list) return;
  list.innerHTML = `<div class="cempty"><span class="spin"></span>&nbsp; Searching…</div>`;
  $("cDetail").textContent = "";
  $("cNotes").textContent = "";
  connectDlg.connectBtn.disabled = true;
  $("cScan").disabled = true;
  const lan = $("cLAN").value.trim();
  if (lan !== (S.lan || "")) { S.lan = lan; changed(true); }
  try {
    const r = await api("scan", { lan });
    showCands(r.candidates, r.notes);
  } catch (e) { showCands([], ["Search failed: " + e.message]); }
  if ($("cScan")) $("cScan").disabled = false;
}
function showCands(list, notes) {
  const box = $("cList");
  if (!box) return;
  cands = list;
  selIdx = -1;
  box.innerHTML = "";
  const real = list.filter((c) => c.family !== "Demo mode");
  if (!real.length) notes = ["No oscilloscope was found. Check the USB cable, give the scope a few seconds after plugging in, and on Windows make sure the USB driver is installed. You can still explore everything with a Demo entry."].concat(notes || []);
  list.forEach((c, i) => {
    const demo = c.family === "Demo mode";
    const it = el("div", "citem" + (c.ready ? "" : " nr"));
    it.innerHTML = `<div class="ci">${demo ? "🧪" : c.family.startsWith("Bench") ? "🖥️" : c.family.includes("hand") || c.family === "OWON" ? "📟" : "🔌"}</div>
      <div class="ct"><div class="cl">${esc(c.label)}</div><div class="cf">${esc(demo ? c.detail : c.family)}</div></div>
      <span class="badge ${c.ready ? "ok" : ""}">${c.ready ? "ready" : "needs setup"}</span>`;
    it.onclick = () => select(i);
    it.ondblclick = () => { select(i); connectSelected().then((ok) => ok !== false && connectDlg.close()); };
    box.appendChild(it);
  });
  $("cNotes").textContent = (notes || []).map((n) => "• " + n).join("\n");
  if (list.length) select(0);
  function select(i) {
    selIdx = i;
    [...box.children].forEach((x, k) => x.classList.toggle("sel", k === i));
    const c = cands[i];
    $("cDetail").innerHTML = `<b>${esc(c.label)}</b> — ${esc(c.family)}\n${esc(c.detail)}`;
    connectDlg.connectBtn.disabled = false;
  }
}
async function connectSelected() {
  const c = cands[selIdx];
  if (!c) return false;
  if (!c.ready) { message("Not ready yet", c.detail); return false; }
  try { await api("connect", { id: c.id }); } catch (e) { message("Connect", e.message); return false; }
  return true;
}

function driverChooser() {
  const b = el("div");
  b.innerHTML = `
    <button class="opt" data-k="6022">Hantek 6022BE or 6022BL<span>One-step signed WinUSB driver (recommended). Windows asks for permission once.</span></button>
    <button class="opt" data-k="zadig">Any other USB scope<span>Hantek 6021, DDS120, DSO-2020, ISDS-205B, MDSO, DSO-2090/2150/2250/5200, Hantek 2C/2D handhelds and OWON HDS handhelds: WinUSB with Zadig (included).</span></button>
    <button class="opt" data-k="bench">Rigol / Siglent / Keysight / Tektronix over USB<span>Works if NI-VISA or Keysight IO Libraries are installed (often installed with the scope maker's software). Otherwise use Zadig, or connect over the network.</span></button>
    <button class="opt" data-k="pico">PicoScope<span>Needs Pico's free PicoSDK, which we may not include. Opens Pico's download page.</span></button>`;
  const d = modal("Install a USB driver", b, [{ label: "Close" }], "small");
  b.querySelectorAll(".opt").forEach((o) => o.onclick = async () => {
    const k = o.dataset.k;
    if (k === "bench") { d.close(); message("Bench scopes over USB", "If NI-VISA or Keysight IO Libraries are installed, Scopie finds USB bench scopes by itself.\n\nOtherwise give the scope the WinUSB driver with Zadig (choose 'Any other USB scope'), or simply use the network: type the scope's IP address in the Connect window - no driver is needed for that."); return; }
    if (k === "pico") { api("driver", { kind: "pico" }); d.close(); return; }
    if (k === "zadig") {
      d.close();
      modal("Install WinUSB with Zadig", `<ol>
        <li>Plug the scope in (and switch it on).</li>
        <li>Zadig opens with "List All Devices" already on. Pick your scope in the list (Hantek 6022: "DSO-6022BE" or USB ID 04B4 6022).</li>
        <li>Make sure the box on the right of the green arrow says <b>WinUSB</b>.</li>
        <li>Click <b>Install Driver</b> (or <b>Replace Driver</b>) and wait.</li></ol>
        <p>Hantek 6022 and DSO-2xxx scopes get a <b>new USB ID</b> once Scopie has loaded their firmware: after connecting the first time, run Zadig again for the new entry.</p>`,
        [{ label: "Cancel", ghost: true }, { label: "Open Zadig", primary: true, fn: async () => { try { await api("driver", { kind: "zadig" }); } catch (e) { message("Zadig", e.message); } } }], "small");
      return;
    }
    o.innerHTML = `<span class="spin"></span>&nbsp; Installing… (Windows will ask for permission)`;
    try {
      await api("driver", { kind: "6022" });
      d.close();
      message("Driver installed", "The Hantek 6022 driver is installed. If the scope was plugged in, unplug it and plug it back in, then click Search again.");
    } catch (e) { d.close(); message("Driver", e.message); }
  });
}

function noScopeBox(r) {
  const nr = (r.candidates || []).filter((c) => !c.ready && c.family !== "Demo mode");
  let html = `<p class="msgtext" style="margin-top:0">No ready-to-use oscilloscope was found.</p>`;
  if (nr.length) html += `<ul>${nr.map((c) => `<li><b>${esc(c.label)}</b>: ${esc(c.detail)}</li>`).join("")}</ul>`;
  html += `<p>You can explore everything in demo mode, or open the Connect window to search again, add a network scope or install drivers.</p>`;
  modal("No oscilloscope found", html, [
    { label: "Close", ghost: true },
    { label: "Connect…", fn: () => setTimeout(() => openConnect(r), 0) },
    { label: "Use demo mode", primary: true, fn: () => connectDemo("stream") }], "small");
}

// ================================================================ documents

function guide() { modal("Beginner's guide", GUIDE_HTML, null, ""); }
function supported() { modal("Supported oscilloscopes", SUPPORTED_HTML, null, "wide"); }
function about() {
  modal("About Scopie", `<div class="help-hero"><img src="icon.png" alt=""><div><b style="font-size:16px">Scopie ${esc(META.version)}</b><br>A beginner-friendly program for many USB, bench and handheld oscilloscopes.<br>Part of the free <b>nrmluvdev</b> maker suite.</div></div>
    <p>Created by <b>nrmluvdev</b>. Copyright © 2026 nrmluvdev.</p>
    <p>This program is free software: you can redistribute it and/or modify it under the terms of the GNU General Public License, version 3 or (at your option) any later version. It is distributed in the hope that it will be useful, but <b>WITHOUT ANY WARRANTY</b>; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See Help ▸ Licences for details and for the projects Scopie builds on.</p>
    <p>Scopie is not affiliated with or endorsed by any oscilloscope manufacturer. Product names are trademarks of their owners.</p>
    <p class="muted">Settings: ${esc(META.dataDir)}<br>Saved files: ${esc(META.exportDir)}</p>`, null, "small");
}
function md(text) {
  // just enough Markdown for the notices: headings, tables, lists, bold, links
  const inline = (s) => esc(s).replace(/\*\*(.+?)\*\*/g, "<b>$1</b>").replace(/`(.+?)`/g, "<code>$1</code>").replace(/\[(.+?)\]\((.+?)\)/g, "$1 ($2)");
  const out = [];
  const lines = text.split(/\r?\n/);
  for (let i = 0; i < lines.length; i++) {
    const l = lines[i];
    if (/^#{1,3} /.test(l)) out.push(`<h3>${inline(l.replace(/^#+ /, ""))}</h3>`);
    else if (l.startsWith("|")) {
      const rows = [];
      while (i < lines.length && lines[i].startsWith("|")) { if (!/^\|[-| :]+\|$/.test(lines[i])) rows.push(lines[i]); i++; }
      i--;
      out.push(`<table class="tbl">${rows.map((r, k) => "<tr>" + r.slice(1, -1).split("|").map((c) => (k ? "<td>" : "<th>") + inline(c.trim()) + (k ? "</td>" : "</th>")).join("") + "</tr>").join("")}</table>`);
    } else if (/^[-*] /.test(l)) out.push(`<ul><li>${inline(l.slice(2))}</li></ul>`);
    else if (l.trim()) out.push(`<p>${inline(l)}</p>`);
  }
  return out.join("");
}
async function licences() {
  const get = (n) => fetch("/api/doc?name=" + n, { headers: { "X-Scopie": "1" } }).then((r) => r.text());
  const [notes, lic] = await Promise.all([get("notices"), get("license")]);
  const b = el("div");
  const tabs = el("div", "seg");
  tabs.style.marginBottom = "10px";
  const body = el("div", "md");
  [["Third-party notices", () => { body.innerHTML = md(notes); }], ["GNU GPL v3", () => { body.innerHTML = `<pre>${esc(lic)}</pre>`; }]].forEach(([t, fn], i) => {
    const bt = el("button", i ? "" : "active", t);
    bt.onclick = () => { tabs.querySelectorAll("button").forEach((x) => x.classList.remove("active")); bt.classList.add("active"); fn(); };
    tabs.appendChild(bt);
  });
  b.append(tabs, body);
  body.innerHTML = md(notes);
  modal("Licences", b, null, "wide");
}

// ================================================================ menus, toolbar, keys

const ACTIONS = {
  openCSV: () => $("fileCSV").click(), exportCSV, explainProbie, screenshot, saveSetup, loadSetup: () => $("fileSetup").click(),
  openExports: () => api("open-folder", {}), connect: () => openConnect(), demoStream: () => connectDemo("stream"),
  demoBench: () => connectDemo("bench"), disconnect: () => api("disconnect", {}), drivers: driverChooser,
  pico: () => api("driver", { kind: "pico" }), guide, supported, logs: () => api("open-folder", { which: "data" }), licences, about,
};
document.querySelectorAll(".menu").forEach((m) => {
  m.querySelector(".mbtn").onclick = () => {
    const open = m.classList.contains("open");
    document.querySelectorAll(".menu.open").forEach((x) => x.classList.remove("open"));
    if (!open) m.classList.add("open");
  };
  m.querySelector(".mbtn").onmouseenter = () => {
    if (document.querySelector(".menu.open") && !m.classList.contains("open")) {
      document.querySelectorAll(".menu.open").forEach((x) => x.classList.remove("open"));
      m.classList.add("open");
    }
  };
  m.querySelectorAll(".mdrop button").forEach((b) => b.onclick = () => { m.classList.remove("open"); ACTIONS[b.dataset.act](); });
});
$("btnRun").onclick = toggleRun;
$("btnSingle").onclick = single;
$("btnAuto").onclick = autoset;
$("btnDefault").onclick = () => api("default", {});
$("btnClear").onclick = () => api("clear", {});
$("btnConnect").onclick = () => openConnect();
$("btnShot").onclick = screenshot;
$("btnCSV").onclick = exportCSV;
$("btnProbie").onclick = explainProbie;
$("btnOpen").onclick = () => $("fileCSV").click();
$("btnMeas").onclick = measPopover;

document.addEventListener("keydown", (e) => {
  const t = e.target, typing = t && (t.tagName === "INPUT" || t.tagName === "SELECT" || t.tagName === "TEXTAREA");
  const modalOpen = $("modals").children.length > 0;
  const k = e.key.toLowerCase();
  if (e.key === "F1") { e.preventDefault(); guide(); return; }
  if (e.ctrlKey && !e.altKey) {
    const map = { k: () => openConnect(), o: () => $("fileCSV").click(), s: exportCSV, p: screenshot };
    if (map[k]) { e.preventDefault(); if (!modalOpen || k === "o") map[k](); }
    return;
  }
  if (typing || modalOpen || e.altKey || e.metaKey) return;
  if (e.key === " ") { e.preventDefault(); toggleRun(); }
  else if (k === "s") single();
  else if (k === "a") autoset();
});

// ================================================================ start

async function init() {
  META = await api("info");
  S = META.settings;
  INFO = META.device.info;
  CONNECTED = META.device.connected;
  $("ver").textContent = META.version;
  if (META.os !== "windows") document.querySelectorAll(".win-only").forEach((x) => x.classList.add("hidden"));
  buildSide();
  layoutFFT();
  refreshControls();
  showHelp(null);
  connectSSE();
  requestAnimationFrame(loop);
  try {
    const r = await api("startup", {});
    if (r.action === "connected") toast("Connected to " + r.label);
    else if (r.action === "choose") openConnect(r);
    else if (r.action === "noscope") noScopeBox(r);
  } catch (e) { toast(e.message, "err"); }
}
init().catch((e) => message("Scopie", "Scopie could not start its interface:\n" + e.message));
