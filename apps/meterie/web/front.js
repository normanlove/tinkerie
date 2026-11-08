// Meterie's window on macOS and Linux. Meterie renders every frame itself (the
// same renderer as on Windows) and streams changed tiles here. Text and icons
// are drawn into Meterie's frames from glyphs this page renders with the
// embedded fonts, so dialogs and overlays layer exactly as on Windows.
"use strict";
const cv = document.getElementById("c");
const ctx = cv.getContext("2d", { alpha: false });
const FACES = { r: "MeterieR", s: "MeterieS", b: "MeterieB", m: "MeterieM" };
const CURSORS = ["default", "pointer", "text", "move", "ew-resize", "crosshair"];
let seq = 0, alive = true, sounds = {}, audio = null;

// ------------------------------------------------------------------ input
let queue = [], sending = false, pendingMove = null;
function send(ev) {
  if (ev.t === "move") { pendingMove = ev; } else { if (pendingMove) { queue.push(pendingMove); pendingMove = null; } queue.push(ev); }
  flush();
}
async function flush() {
  if (sending) return;
  sending = true;
  while (alive && (queue.length || pendingMove)) {
    if (pendingMove) { queue.push(pendingMove); pendingMove = null; }
    const batch = queue; queue = [];
    try { await fetch("input", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(batch) }); } catch (e) {}
  }
  sending = false;
}
const dpr = () => window.devicePixelRatio || 1;
const pos = (e) => ({ x: Math.round(e.clientX * dpr()), y: Math.round(e.clientY * dpr()) });
cv.addEventListener("mousemove", (e) => send({ t: "move", ...pos(e) }));
cv.addEventListener("mousedown", (e) => { cv.focus(); send({ t: "down", b: e.button, ...pos(e), ctrl: e.ctrlKey || e.metaKey, shift: e.shiftKey }); e.preventDefault(); });
window.addEventListener("mouseup", (e) => send({ t: "up", b: e.button, ...pos(e) }));
cv.addEventListener("dblclick", (e) => send({ t: "dbl", ...pos(e) }));
cv.addEventListener("mouseleave", () => send({ t: "leave" }));
cv.addEventListener("contextmenu", (e) => e.preventDefault());
cv.addEventListener("wheel", (e) => {
  const lines = e.deltaMode === 1 ? e.deltaY / 3 : e.deltaMode === 2 ? e.deltaY * 3 : e.deltaY / 100;
  send({ t: "wheel", d: -lines, ...pos(e) });
  e.preventDefault();
}, { passive: false });

const KEYS = { Backspace: "Back", Delete: "Delete", ArrowLeft: "Left", ArrowRight: "Right", ArrowUp: "Up", ArrowDown: "Down",
  Home: "Home", End: "End", Enter: "Enter", Escape: "Escape", Tab: "Tab", " ": "Space", F1: "F1", F2: "F2",
  "+": "Plus", "=": "Plus", "-": "Minus", "0": "0" };
const LETTERS = "acvxsonzhrm";
window.addEventListener("keydown", (e) => {
  const ctrl = e.ctrlKey || e.metaKey; // ⌘ works like Ctrl on a Mac
  let k = KEYS[e.key];
  if (!k && e.key.length === 1 && LETTERS.includes(e.key.toLowerCase())) k = e.key.toUpperCase();
  if (ctrl && e.key.toLowerCase() === "v") return; // the paste event carries the text
  if (k) send({ t: "key", k, ctrl, shift: e.shiftKey });
  if (e.key.length === 1 && !ctrl && !e.altKey) send({ t: "char", c: e.key });
  if (k || (e.key.length === 1 && !ctrl) || ctrl) e.preventDefault();
});
window.addEventListener("keyup", (e) => send({ t: "mods", ctrl: e.ctrlKey || e.metaKey, shift: e.shiftKey }));
window.addEventListener("paste", (e) => {
  send({ t: "paste", s: (e.clipboardData || window.clipboardData).getData("text") });
  send({ t: "key", k: "V", ctrl: true, shift: false });
  e.preventDefault();
});
window.addEventListener("blur", () => send({ t: "mods", ctrl: false, shift: false }));

// ------------------------------------------------------------------ glyphs
const gcv = document.createElement("canvas");
const gx = gcv.getContext("2d", { willReadFrequently: true });
function renderGlyph(n) {
  // n: {k, cp, px, face, icon, w, h, bx, by}
  gcv.width = n.w; gcv.height = n.h;
  gx.clearRect(0, 0, n.w, n.h);
  gx.fillStyle = gx.strokeStyle = "#fff";
  if (n.icon) {
    const d = ICONS[n.cp];
    if (d) {
      const fill = d.startsWith("f:");
      const p = new Path2D(fill ? d.slice(2) : d);
      gx.save();
      gx.translate(n.bx, n.by);
      gx.scale(n.px / 24, n.px / 24);
      gx.lineWidth = 1.8; gx.lineCap = "round"; gx.lineJoin = "round";
      fill ? gx.fill(p) : gx.stroke(p);
      gx.restore();
    }
  } else {
    gx.font = `${n.px}px ${FACES[n.face]}`;
    gx.textBaseline = "alphabetic";
    gx.fillText(String.fromCodePoint(n.cp), n.bx, n.by);
  }
  const px = gx.getImageData(0, 0, n.w, n.h).data;
  const a = new Uint8Array(n.w * n.h);
  for (let i = 0; i < a.length; i++) a[i] = px[i * 4 + 3];
  return a;
}
async function uploadGlyphs(need) {
  const enc = new TextEncoder();
  const parts = [];
  for (const n of need) {
    const k = enc.encode(n.k), a = renderGlyph(n);
    const h = new DataView(new ArrayBuffer(6));
    h.setUint16(0, k.length, true); h.setUint16(2, n.w, true); h.setUint16(4, n.h, true);
    parts.push(h.buffer, k, a);
  }
  await fetch("glyphs", { method: "POST", body: new Blob(parts) });
}

// ------------------------------------------------------------------ frames
async function inflate(buf) {
  const s = new Blob([buf]).stream().pipeThrough(new DecompressionStream("deflate-raw"));
  return new Uint8Array(await new Response(s).arrayBuffer());
}
function size() {
  const w = Math.max(1, Math.round(innerWidth * dpr())), h = Math.max(1, Math.round(innerHeight * dpr()));
  return { w, h };
}
async function loop() {
  await Promise.all(Object.values(FACES).map((f) => document.fonts.load(`16px ${f}`)));
  let fails = 0;
  while (alive) {
    const { w, h } = size();
    let r;
    try {
      r = await fetch(`frame?since=${seq}&w=${w}&h=${h}&dpr=${dpr()}`);
      if (!r.ok) throw new Error(r.status);
      fails = 0;
    } catch (e) {
      if (++fails > 3) { alive = false; cv.style.display = "none"; document.getElementById("gone").style.display = "flex"; return; }
      await new Promise((ok) => setTimeout(ok, 500));
      continue;
    }
    const buf = await r.arrayBuffer();
    const dv = new DataView(buf);
    const jl = dv.getUint32(0, true);
    const m = JSON.parse(new TextDecoder().decode(new Uint8Array(buf, 4, jl)));
    if (m.quit) { alive = false; window.close(); document.getElementById("gone").style.display = "flex"; cv.style.display = "none"; return; }
    if (cv.width !== m.w || cv.height !== m.h) { cv.width = m.w; cv.height = m.h; }
    if (buf.byteLength > 4 + jl) {
      const t = await inflate(buf.slice(4 + jl));
      const tv = new DataView(t.buffer);
      for (let o = 0; o < t.length;) {
        const x = tv.getUint16(o, true), y = tv.getUint16(o + 2, true), tw = tv.getUint16(o + 4, true), th = tv.getUint16(o + 6, true);
        o += 8;
        ctx.putImageData(new ImageData(new Uint8ClampedArray(t.buffer, o, tw * th * 4), tw, th), x, y);
        o += tw * th * 4;
      }
    }
    seq = m.seq;
    cv.style.cursor = CURSORS[m.cursor] || "default";
    if (m.title && document.title !== m.title) document.title = m.title;
    if (m.clip != null) navigator.clipboard?.writeText(m.clip).catch(() => {});
    if (m.stopSound && audio) { audio.pause(); audio = null; }
    if (m.sound) { if (audio) audio.pause(); audio = new Audio(`sound?id=${m.sound.id}`); audio.loop = m.sound.loop; audio.play().catch(() => {}); }
    if (m.need && m.need.length) uploadGlyphs(m.need);
  }
}
addEventListener("resize", () => send({ t: "resize" }));
cv.focus();
loop();
