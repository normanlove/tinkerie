// Codie — Signals mode: explains captures from Scopie, Tracie and Meterie the
// way the Code mode explains lines of code. Click a card to see where it
// happens; click the capture to find the card that explains it.
// Part of the Tinkerie maker suite. Created by nrmluvdev.
(() => {
  "use strict";
  const $ = (s, r = document) => r.querySelector(s);
  const $$ = (s, r = document) => [...r.querySelectorAll(s)];
  const store = {
    get(k) { try { return localStorage.getItem("codie." + k); } catch { return null; } },
    set(k, v) { try { localStorage.setItem("codie." + k, v); } catch {} },
  };
  const H = { "X-Codie": "1" };
  const getJSON = p => fetch(p, { headers: H }).then(r => r.json());
  const esc = s => String(s ?? "").replace(/[&<>"]/g, c => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" }[c]));
  const md = s => esc(s).replace(/\*\*([^*]+)\*\*/g, "<b>$1</b>").replace(/`([^`]+)`/g, "<code>$1</code>");
  const plain = s => String(s || "").replace(/\*\*/g, "").replace(/`/g, "");
  const COLORS = ["#34d399", "#22d3ee", "#a78bfa", "#fbbf24", "#f472b6", "#fb923c", "#60a5fa", "#f87171"];

  let R = null, sel = -1, view = { t0: 0, t1: 1 }, mode = "code", fileName = "";
  const cv = $("#sigCanvas"), plot = $("#sigPlot"), tip = $("#sigTip");

  // ------------------------------------------------------------ mode switch
  function setMode(m) {
    mode = m;
    document.body.classList.toggle("sig", m === "signals");
    $("#codeMain").hidden = m === "signals";
    $("#sigMain").hidden = m !== "signals";
    $$(".modes button").forEach(b => b.setAttribute("aria-selected", b.dataset.mode === m));
    $("#open").title = m === "signals" ? "Open a capture file" : "Open a code file";
    store.set("mode", m);
    if (m === "signals") { loadRecent(); requestAnimationFrame(draw); }
  }
  $$(".modes button").forEach(b => b.onclick = () => setMode(b.dataset.mode));

  // ------------------------------------------------------------ loading
  function status(t) { $("#sigStats").textContent = t; }
  function show(r, name) {
    if (!r || r.error) { status(r ? r.error : "Couldn't read that capture."); setMode("signals"); return; }
    R = r; fileName = name || r.file || ""; sel = -1;
    view = { t0: 0, t1: 1 };
    zoomAll();
    setMode("signals");
    render();
  }
  function openPath(path, name) {
    setMode("signals"); status("Reading " + (name || "the capture") + "…");
    getJSON("/api/signals/open?path=" + encodeURIComponent(path || "")).then(r => show(r, name)).catch(e => status("Couldn't open it: " + e.message));
  }
  function openDemo(name) {
    status("Loading the example…");
    getJSON("/api/signals/demo?name=" + encodeURIComponent(name)).then(r => show(r, name)).catch(e => status(e.message));
  }
  function openLocal(f) {
    setMode("signals");
    if (f.size > 96 * 1024 * 1024) { status("That capture is over 96 MB: export a shorter piece."); return; }
    status("Reading " + f.name + "…");
    fetch("/api/signals/analyse?name=" + encodeURIComponent(f.name), { method: "POST", headers: H, body: f })
      .then(r => r.json()).then(r => show(r, f.name)).catch(e => status("Couldn't read it: " + e.message));
  }
  // is a dropped/opened file a capture? (decides between Code and Signals)
  function take(f) {
    const n = f.name.toLowerCase();
    if (/\.(vcd|nst\.csv)$/.test(n)) { openLocal(f); return true; }
    if (/\.(csv|tsv)$/.test(n)) { openLocal(f); return true; }
    if (mode === "signals" && /\.(txt|log)$/.test(n)) { openLocal(f); return true; }
    return false;
  }
  function loadRecent() {
    getJSON("/api/signals/recent").then(list => {
      const s = $("#sigRecent");
      s.innerHTML = `<option value="">${list.length ? "Recent captures…" : "None saved yet"}</option>` +
        list.map(c => `<option value="${esc(c.path)}">${esc(c.app)} · ${esc(c.name)}</option>`).join("");
    }).catch(() => {});
  }
  $("#sigRecent").onchange = e => { const p = e.target.value; e.target.value = ""; if (p) openPath(p, p.split(/[\\/]/).pop()); };
  $("#sigExamples").onchange = e => { const n = e.target.value; e.target.value = ""; if (n) openDemo(n); };
  getJSON("/api/info").then(i => {
    $("#sigExamples").innerHTML = `<option value="">Pick an example…</option>` + (i.sigDemos || []).map(d => `<option value="${esc(d.file)}">${esc(d.label)}</option>`).join("");
  }).catch(() => {});
  plot.addEventListener("dragover", e => { e.preventDefault(); plot.classList.add("drop"); });
  plot.addEventListener("dragleave", () => plot.classList.remove("drop"));
  plot.addEventListener("drop", e => { e.preventDefault(); plot.classList.remove("drop"); const f = e.dataTransfer.files[0]; if (f) openLocal(f); });

  // ------------------------------------------------------------ the report
  $$("#sigTabs button").forEach(b => b.onclick = () => stab(b.dataset.stab));
  function stab(name) {
    $$("#sigTabs button").forEach(b => b.setAttribute("aria-selected", b.dataset.stab === name));
    $$("#sigMain .view").forEach(v => v.hidden = v.id !== "s-" + name);
  }
  const tagName = { overview: "capture", channel: "channel", protocol: "protocol", event: "event", power: "power", serial: "serial" };
  function render() {
    const r = R;
    $("#sigSource").textContent = r.source || "";
    $("#sigEmpty").hidden = true;
    $("#sigSummary").innerHTML = r.summary.map(p => `<p>${md(p)}</p>`).join("");
    $("#sigCards").innerHTML = r.cards.map((c, i) => `
      <div class="ln sig-card" data-i="${i}">
        <div class="no">${c.ch >= 0 ? `<span class="dot" style="background:${COLORS[c.ch % COLORS.length]}"></span>` : ""}${c.t1 > c.t0 || c.t0 ? esc(fmtT(c.t0)) : ""}</div>
        <div><div class="part"><div class="what"><span class="kind k-${c.tag}">${tagName[c.tag] || c.tag}</span><b>${esc(c.title)}</b></div>
          <div class="what" style="margin-top:4px">${md(c.what)}</div>${c.why ? `<div class="why">${md(c.why)}</div>` : ""}
          ${c.facts && c.facts.length ? `<dl class="facts">${c.facts.map(f => `<dt>${esc(f[0])}</dt><dd>${esc(f[1])}</dd>`).join("")}</dl>` : ""}</div></div>
      </div>`).join("");
    // decoded
    $("#nDecoded").textContent = r.decoded.length || (r.serial && r.serial.length) || "";
    let dec = "";
    if (r.decoded.length) dec += `<table class="dec"><thead><tr><th>Time</th><th>Channel</th><th>What</th><th>Value</th><th>Meaning</th></tr></thead><tbody>${r.decoded.slice(0, 3000).map((f, i) => `<tr data-d="${i}" class="k-${f.kind}"><td>${esc(fmtT(f.t0))}</td><td>${esc((r.chans[f.ch] || {}).name || "")}</td><td>${esc(f.proto)}</td><td class="mono">${esc(f.text)}</td><td>${esc(f.note)}</td></tr>`).join("")}</tbody></table>`;
    if (r.serial && r.serial.length) dec += `<h3 class="dec-h">Serial lines</h3><table class="dec"><thead><tr><th>Time</th><th></th><th>Text</th></tr></thead><tbody>${r.serial.map((l, i) => `<tr data-l="${i}"><td>${esc(fmtT(l.T))}</td><td>${esc(l.Dir)}</td><td class="mono">${esc(l.Text)}</td></tr>`).join("")}</tbody></table>`;
    $("#sigDecoded").innerHTML = dec || `<div class="empty">Nothing decoded.<br>Codie decodes UART serial, I2C, SPI, WS2812 LEDs, infrared remotes and 1-Wire from logic captures (and scope channels that switch between two levels).</div>`;
    // concepts
    $("#nSigConcepts").textContent = r.concepts.length || "";
    $("#sigConcepts").innerHTML = r.concepts.map(c => `<div class="card"><h3>${esc(c.name)}</h3><p>${esc(c.text)}</p>${c.chans && c.chans.length ? `<div class="chips">${c.chans.map(ch => `<button class="chip" data-ch="${ch}">${esc((r.chans[ch] || {}).name || "")}</button>`).join("")}</div>` : ""}</div>`).join("") || `<div class="empty">Nothing yet.</div>`;
    // warnings
    $("#nSigWarnings").textContent = r.warnings.length || "";
    $("#sigWarnings").innerHTML = r.warnings.length ? r.warnings.map((w, i) => `<div class="warn-card" data-w="${i}"><b>${esc(w.title)}</b><p>${esc(w.text)}</p></div>`).join("") : `<div class="empty"><b>Nothing to worry about</b>Codie didn't spot anything odd. It can't see everything, though: it only knows what was captured.</div>`;
    const n = r.chans.length;
    status(`${n} channel${n === 1 ? "" : "s"} · ${fmtT(r.duration)} · ${r.decoded.length} decoded item${r.decoded.length === 1 ? "" : "s"} · scroll to zoom, drag to move, double-click to show everything`);
    resize();
  }
  $("#sigCards").addEventListener("click", e => { const d = e.target.closest("[data-i]"); if (d) pick(+d.dataset.i, true); });
  $("#sigDecoded").addEventListener("click", e => {
    const d = e.target.closest("[data-d]");
    if (d) { const f = R.decoded[+d.dataset.d]; focusRange(f.t0, f.t1); $$("#sigDecoded tr.sel").forEach(x => x.classList.remove("sel")); d.classList.add("sel"); }
    const l = e.target.closest("[data-l]");
    if (l) { const t = R.serial[+l.dataset.l].T; focusRange(t, t); }
  });
  $("#sigWarnings").addEventListener("click", e => { const d = e.target.closest("[data-w]"); if (d) { const w = R.warnings[+d.dataset.w]; if (w.t1 > w.t0 || w.t0) focusRange(w.t0, w.t1); } });
  $("#sigConcepts").addEventListener("click", e => { const d = e.target.closest("[data-ch]"); if (d) { const i = R.cards.findIndex(c => c.ch === +d.dataset.ch && c.tag !== "overview"); if (i >= 0) { stab("explain"); pick(i, true); } } });

  function pick(i, zoom) {
    sel = i;
    $$("#sigCards .sel").forEach(x => x.classList.remove("sel"));
    const el = $(`#sigCards [data-i="${i}"]`);
    if (el) { el.classList.add("sel"); if (!zoom) el.scrollIntoView({ block: "nearest", behavior: "smooth" }); }
    const c = R.cards[i];
    if (zoom && c && c.tag !== "overview") focusRange(c.t0, c.t1); else draw();
  }
  function focusRange(a, b) {
    const span = Math.max(b - a, R.duration / 2000, 1e-9);
    const pad = span * 0.15;
    view = { t0: a - pad, t1: a + span + pad };
    clampView(); draw();
  }

  // ------------------------------------------------------------ drawing
  function fmtT(s) {
    if (s == null || !isFinite(s)) return "";
    const a = Math.abs(s);
    if (a === 0) return "0 s";
    if (a < 1e-6) return (s * 1e9).toPrecision(3) + " ns";
    if (a < 1e-3) return (s * 1e6).toPrecision(3) + " µs";
    if (a < 1) return (s * 1e3).toPrecision(3) + " ms";
    if (a < 120) return s.toPrecision(3) + " s";
    if (a < 7200) return (s / 60).toFixed(1) + " min";
    return (s / 3600).toFixed(1) + " h";
  }
  function fmtV(v, u) {
    if (!isFinite(v)) return "";
    const a = Math.abs(v);
    const p = [[1e6, "M"], [1e3, "k"], [1, ""], [1e-3, "m"], [1e-6, "µ"]];
    if (!u || u === "°C" || u === "%") return +v.toPrecision(4) + (u ? " " + u : "");
    for (const [f, n] of p) if (a >= f * 0.9995 || f === 1e-6) return +(v / f).toPrecision(3) + " " + n + u;
    return v + " " + u;
  }
  function lanes() {
    if (!R) return [];
    const out = []; let y = 26;
    R.chans.forEach((c, i) => { const h = c.kind === "digital" ? 38 : 92; out.push({ i, y, h, c }); y += h + 10; });
    return out;
  }
  function resize() {
    const r = plot.getBoundingClientRect();
    const L = lanes(); const need = L.length ? L[L.length - 1].y + L[L.length - 1].h + 30 : r.height;
    const dpr = window.devicePixelRatio || 1;
    cv.style.width = r.width + "px"; cv.style.height = Math.max(r.height - 2, need) + "px";
    cv.width = Math.round(r.width * dpr); cv.height = Math.round(Math.max(r.height - 2, need) * dpr);
    draw();
  }
  window.addEventListener("resize", () => { if (mode === "signals") resize(); });
  const LEFT = 108;
  const xOf = t => LEFT + (t - view.t0) / (view.t1 - view.t0) * (cv.clientWidth - LEFT - 12);
  const tOf = x => view.t0 + (x - LEFT) / (cv.clientWidth - LEFT - 12) * (view.t1 - view.t0);
  function draw() {
    const g = cv.getContext("2d"), dpr = window.devicePixelRatio || 1, W = cv.clientWidth, Hh = cv.clientHeight;
    g.setTransform(dpr, 0, 0, dpr, 0, 0);
    g.clearRect(0, 0, W, Hh);
    if (!R) return;
    const L = lanes();
    // time grid
    const span = view.t1 - view.t0, step = niceStep(span / 8);
    g.font = "11px " + getComputedStyle(document.body).fontFamily; g.fillStyle = "#5f677c"; g.strokeStyle = "rgba(255,255,255,.05)";
    for (let t = Math.ceil(view.t0 / step) * step; t <= view.t1; t += step) {
      const x = xOf(t); g.beginPath(); g.moveTo(x, 18); g.lineTo(x, Hh); g.stroke(); g.fillText(fmtT(t), x + 3, 13);
    }
    // the selected card's stretch of time
    const c = R.cards[sel];
    if (c && c.tag !== "overview") {
      const a = xOf(c.t0), b = Math.max(xOf(c.t1), a + 2);
      g.fillStyle = "rgba(52,211,153,.12)"; g.fillRect(a, 18, b - a, Hh);
      g.fillStyle = "rgba(52,211,153,.7)"; g.fillRect(a, 18, 1.5, Hh); g.fillRect(b - 1.5, 18, 1.5, Hh);
    }
    for (const m of R.markers || []) { const x = xOf(m.T); g.fillStyle = "#fbbf24"; g.fillRect(x, 18, 1, Hh); g.fillText(m.Text, x + 3, 28); }
    for (const ln of L) {
      const col = COLORS[ln.i % COLORS.length], ch = ln.c;
      g.fillStyle = "rgba(255,255,255,.025)"; g.fillRect(LEFT, ln.y, W - LEFT - 12, ln.h);
      g.fillStyle = col; g.font = "600 12px " + getComputedStyle(document.body).fontFamily;
      g.fillText(ch.name.slice(0, 13), 10, ln.y + 15);
      g.fillStyle = "#8d95aa"; g.font = "11px " + getComputedStyle(document.body).fontFamily;
      if (ch.kind !== "digital") { g.fillText(fmtV(ch.max, ch.unit), 10, ln.y + 31); g.fillText(fmtV(ch.min, ch.unit), 10, ln.y + ln.h - 4); }
      else g.fillText("logic", 10, ln.y + 31);
      g.save(); g.beginPath(); g.rect(LEFT, ln.y - 2, W - LEFT - 12, ln.h + 4); g.clip();
      g.strokeStyle = col; g.fillStyle = col; g.lineWidth = 1.4;
      if (ch.kind === "digital") drawDigital(g, ch, ln); else drawAnalog(g, ch, ln);
      g.restore();
      // decoded labels on this lane
      if (R.decoded.length) {
        g.font = "10.5px " + getComputedStyle(document.body).fontFamily;
        let lastX = -1e9;
        for (const f of R.decoded) {
          if (f.ch !== ln.i || f.t1 < view.t0 || f.t0 > view.t1) continue;
          const a = xOf(f.t0), b = xOf(f.t1);
          if (b - a < 14 || a < lastX) continue;
          const txt = f.text.length > 26 ? f.text.slice(0, 25) + "…" : f.text;
          const w = g.measureText(txt).width;
          if (w + 6 > b - a && b - a < 40) continue;
          g.fillStyle = f.kind === "error" || f.kind === "nack" ? "rgba(248,113,113,.25)" : "rgba(167,139,250,.22)";
          g.fillRect(a, ln.y + ln.h - 15, Math.max(b - a, w + 6), 14);
          g.fillStyle = "#e8ebf2"; g.fillText(txt, a + 3, ln.y + ln.h - 4);
          lastX = a + Math.max(b - a, w + 6);
        }
      }
    }
  }
  function niceStep(x) { const p = Math.pow(10, Math.floor(Math.log10(x))); const n = x / p; return (n < 1.5 ? 1 : n < 3.5 ? 2 : n < 7.5 ? 5 : 10) * p; }
  function drawAnalog(g, ch, ln) {
    const lo = ch.min, hi = ch.max === ch.min ? ch.min + 1 : ch.max;
    const yOf = v => ln.y + ln.h - 4 - (v - lo) / (hi - lo) * (ln.h - 8);
    const P = ch.plot || [];
    // binary search for the first visible point
    let a = 0, b = P.length;
    while (a < b) { const m = (a + b) >> 1; if (P[m][0] < view.t0) a = m + 1; else b = m; }
    a = Math.max(0, a - 1);
    g.beginPath();
    let first = true;
    for (let k = a; k < P.length; k++) {
      const [t, mn, mx] = P[k], x = xOf(t);
      if (first) { g.moveTo(x, yOf(mx)); first = false; } else g.lineTo(x, yOf(mx));
      if (mn !== mx) g.lineTo(x, yOf(mn));
      if (t > view.t1) break;
    }
    g.stroke();
    if (ch.kind === "meter" && P.length < 400) for (let k = a; k < P.length && P[k][0] <= view.t1; k++) { g.beginPath(); g.arc(xOf(P[k][0]), yOf(P[k][1]), 2, 0, 7); g.fill(); }
  }
  function drawDigital(g, ch, ln) {
    const E = ch.edges || [], top = ln.y + 6, bot = ln.y + ln.h - 6;
    g.beginPath();
    let px = null, py = null;
    for (let k = 0; k < E.length; k++) {
      const [t, v] = E[k], x = xOf(t);
      const nt = k + 1 < E.length ? E[k + 1][0] : R.duration + (R.chans.length ? 0 : 0);
      if (v === 2) { // busy: too many edges to draw
        const xb = xOf(nt);
        g.save(); g.globalAlpha = .35; g.fillRect(x, top, Math.max(1, xb - x), bot - top); g.restore();
        px = null; continue;
      }
      const y = v ? top : bot;
      if (px === null) g.moveTo(x, y); else { g.lineTo(x, py); g.lineTo(x, y); }
      px = x; py = y;
      if (t > view.t1) break;
    }
    if (px !== null) g.lineTo(xOf(Math.max(view.t1, R.duration)), py);
    g.stroke();
  }

  // ------------------------------------------------------------ zoom, pan, click
  function zoomAll() {
    if (!R) return;
    let a = Infinity, b = -Infinity;
    for (const c of R.chans) {
      const P = c.plot || [], E = c.edges || [];
      if (P.length) { a = Math.min(a, P[0][0]); b = Math.max(b, P[P.length - 1][0]); }
      if (E.length) { a = Math.min(a, E[0][0]); b = Math.max(b, E[E.length - 1][0]); }
    }
    for (const l of R.serial || []) { a = Math.min(a, l.T); b = Math.max(b, l.T); }
    if (!isFinite(a)) { a = 0; b = 1; }
    if (b <= a) b = a + 1e-6;
    const pad = (b - a) * 0.01;
    view = { t0: a - pad, t1: b + pad }; full = { ...view };
  }
  let full = { t0: 0, t1: 1 };
  function clampView() {
    const span = Math.max(view.t1 - view.t0, (full.t1 - full.t0) / 1e6);
    if (span > full.t1 - full.t0) { view = { ...full }; return; }
    if (view.t0 < full.t0) view = { t0: full.t0, t1: full.t0 + span };
    if (view.t1 > full.t1) view = { t0: full.t1 - span, t1: full.t1 };
  }
  function zoom(f, at) {
    const t = at ?? (view.t0 + view.t1) / 2;
    view = { t0: t - (t - view.t0) * f, t1: t + (view.t1 - t) * f };
    clampView(); draw();
  }
  $("#sigZoomIn").onclick = () => zoom(0.5);
  $("#sigZoomOut").onclick = () => zoom(2);
  $("#sigZoomAll").onclick = () => { zoomAll(); draw(); };
  cv.addEventListener("wheel", e => { if (!R) return; e.preventDefault(); const r = cv.getBoundingClientRect(); zoom(e.deltaY > 0 ? 1.25 : 0.8, tOf(e.clientX - r.left)); }, { passive: false });
  let drag = null;
  cv.addEventListener("mousedown", e => { if (!R) return; drag = { x: e.clientX, v: { ...view }, moved: false }; });
  window.addEventListener("mousemove", e => {
    if (drag) {
      const dx = e.clientX - drag.x; if (Math.abs(dx) > 3) drag.moved = true;
      const dt = dx / (cv.clientWidth - LEFT - 12) * (drag.v.t1 - drag.v.t0);
      view = { t0: drag.v.t0 - dt, t1: drag.v.t1 - dt }; clampView(); draw(); return;
    }
    if (!R || mode !== "signals" || e.target !== cv) { tip.hidden = true; return; }
    const r = cv.getBoundingClientRect(), x = e.clientX - r.left, y = e.clientY - r.top;
    if (x < LEFT) { tip.hidden = true; return; }
    const t = tOf(x), ln = lanes().find(l => y >= l.y && y <= l.y + l.h);
    let txt = fmtT(t);
    if (ln) {
      const ch = ln.c;
      if (ch.kind === "digital") { const E = ch.edges || []; let v = null; for (const p of E) { if (p[0] > t) break; v = p[1]; } if (v !== null) txt += ` · ${ch.name} ${v === 2 ? "busy" : v ? "HIGH" : "LOW"}`; }
      else { const P = ch.plot || []; let best = null; for (const p of P) { if (p[0] >= t) { best = p; break; } } if (best) txt += ` · ${ch.name} ${fmtV((best[1] + best[2]) / 2, ch.unit)}`; }
      const f = R.decoded.find(f => f.ch === ln.i && t >= f.t0 && t <= f.t1);
      if (f) txt += ` · ${f.proto}: ${f.text}`;
    }
    tip.textContent = txt; tip.hidden = false;
    tip.style.left = Math.min(x + 14, r.width - 260) + "px"; tip.style.top = (y + 14 + plot.scrollTop) + "px";
  });
  window.addEventListener("mouseup", e => {
    if (!drag) return;
    const d = drag; drag = null;
    if (d.moved || !R) return;
    // a click: find the card that explains this place
    const r = cv.getBoundingClientRect(), x = e.clientX - r.left, y = e.clientY - r.top;
    const t = tOf(x), ln = lanes().find(l => y >= l.y && y <= l.y + l.h);
    let best = -1, bs = Infinity;
    R.cards.forEach((c, i) => {
      if (c.tag === "overview") return;
      const span = Math.max(c.t1 - c.t0, 0);
      const hit = (ln ? c.ch === ln.i || c.ch === -1 && ln === undefined : true) && t >= c.t0 - (view.t1 - view.t0) * 0.01 && t <= c.t1 + (view.t1 - view.t0) * 0.01;
      if (hit && span < bs) { bs = span; best = i; }
    });
    if (best < 0 && ln) best = R.cards.findIndex(c => c.ch === ln.i);
    if (best >= 0) { stab("explain"); pick(best, false); }
  });
  cv.addEventListener("dblclick", () => { zoomAll(); draw(); });
  cv.addEventListener("mouseleave", () => { tip.hidden = true; });

  // ------------------------------------------------------------ save report
  $("#save").addEventListener("click", e => {
    if (mode !== "signals") return;
    e.stopImmediatePropagation();
    if (!R) { status("Open a capture first."); return; }
    let out = `# Codie signal report: ${R.file}\n\n${R.summary.map(plain).join(" ")}\n\n## Explanations\n\n`;
    for (const c of R.cards) {
      out += `### ${c.title}${c.tag !== "overview" ? ` (${fmtT(c.t0)}${c.t1 > c.t0 ? " – " + fmtT(c.t1) : ""})` : ""}\n\n${plain(c.what)}\n\n${c.why ? plain(c.why) + "\n\n" : ""}`;
      if (c.facts && c.facts.length) out += c.facts.map(f => `- ${f[0]}: ${f[1]}`).join("\n") + "\n\n";
    }
    if (R.decoded.length) out += `## Decoded\n\n| Time | Channel | Protocol | Value | Meaning |\n|---|---|---|---|---|\n${R.decoded.slice(0, 2000).map(f => `| ${fmtT(f.t0)} | ${(R.chans[f.ch] || {}).name || ""} | ${f.proto} | ${f.text.replace(/\|/g, "/")} | ${f.note} |`).join("\n")}\n\n`;
    if (R.warnings.length) out += `## Warnings\n\n${R.warnings.map(w => `- **${w.title}**: ${w.text}`).join("\n")}\n\n`;
    if (R.concepts.length) out += `## Concepts\n\n${R.concepts.map(c => `- **${c.name}**: ${c.text}`).join("\n")}\n\n`;
    out += `---\nMade with Codie, part of Tinkerie by nrmluvdev.\n`;
    const a = document.createElement("a");
    a.href = URL.createObjectURL(new Blob([out], { type: "text/markdown" }));
    a.download = (R.file || "capture").replace(/\.(nst\.)?[^.]+$/, "") + "-explained.md";
    document.body.append(a); a.click(); a.remove();
    setTimeout(() => URL.revokeObjectURL(a.href), 2000);
  }, true);

  window.codieSig = { openPath, openLocal, take, mode: setMode, test: { openDemo, get report() { return R; }, pick } };
  if (store.get("mode") === "signals") setMode("signals");
})();
