// Fixie — the window: pick a device and what it's doing, then go step by
// step to the fault. Also the tools, signs, safety and repair-log tabs.
// Part of the Tinkerie maker suite. Created by nrmluvdev.
(() => {
  "use strict";
  const $ = (s, r = document) => r.querySelector(s);
  const $$ = (s, r = document) => [...r.querySelectorAll(s)];
  const esc = s => String(s ?? "").replace(/[&<>"]/g, c => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" }[c]));
  const api = (path, body) => fetch(path, {
    method: body ? "POST" : "GET",
    headers: { "X-Fixie": "1", "Content-Type": "application/json" },
    body: body ? JSON.stringify(body) : undefined,
  }).then(r => { if (!r.ok) throw new Error(r.statusText); return r.json(); });
  const local = {
    get(k) { try { return localStorage.getItem("fixie." + k); } catch { return null; } },
    set(k, v) { try { localStorage.setItem("fixie." + k, v); } catch {} },
  };

  let G = null;                    // the guide
  let state = { repairs: [], have: [] };
  let cur = null;                  // the open repair
  let tab = "repair", serverOK = true, saveTimer = 0;
  const TOOL = {}, SIGN = {}, DEV = {}, SYM = {};

  // ------------------------------------------------------------ start-up
  Promise.all([fetch("guide.json").then(r => r.json()), api("/api/state").catch(() => { serverOK = false; return null; }), api("/api/info").catch(() => null)])
    .then(([g, st, info]) => {
      G = g;
      for (const t of G.tools) TOOL[t.id] = t;
      for (const s of G.signs) SIGN[s.id] = s;
      for (const d of G.devices) DEV[d.id] = d;
      for (const s of G.symptoms) SYM[s.id] = s;
      if (!st || Array.isArray(st)) { try { st = JSON.parse(local.get("state") || "null"); } catch { st = null; } }
      if (st && st.repairs) state = { repairs: st.repairs || [], have: st.have || [] };
      if (info) $("#ver").textContent = `Fixie ${info.version}`;
      const open = state.repairs.find(r => r.id === local.get("open") && r.status === "open");
      cur = open || null;
      renderAll();
    })
    .catch(e => { $("#v-repair").innerHTML = `<div class="empty">Fixie couldn't load its guide: ${esc(e.message)}</div>`; });

  // the window keeps this open; Fixie closes when it goes away
  (function events() {
    if (location.protocol === "file:") return;
    const es = new EventSource("/api/events");
    es.onerror = () => { es.close(); setTimeout(events, 2000); };
  })();

  function save() {
    clearTimeout(saveTimer);
    saveTimer = setTimeout(() => {
      local.set("state", JSON.stringify(state));
      local.set("open", cur ? cur.id : "");
      if (serverOK) api("/api/state", state).catch(() => {});
    }, 300);
    $("#nLog").textContent = state.repairs.length || "";
  }

  // ------------------------------------------------------------ the guide's rules
  const tagsOf = d => new Set([d.id, ...d.tags]);
  // "a|b" is any of them; "!a" is not a
  const match = (cond, tags) => !cond || cond.split("|").some(c => c[0] === "!" ? !tags.has(c.slice(1)) : tags.has(c));
  // where a step really leads for this device: steps that don't apply are skipped,
  // and steps inside a device that mustn't be opened go to the professional
  function resolve(id, dev) {
    const tags = tagsOf(dev);
    for (let n = 0; n < 50; n++) {
      const s = G.steps[id];
      if (!s) return id;
      if (!s.result && s.only && !match(s.only, tags)) { id = s.skip; continue; }
      if (!s.result && s.open && dev.noopen) return "r.pro";
      return id;
    }
    return id;
  }
  const symptomsFor = dev => G.symptoms.filter(s => match(s.for, tagsOf(dev)));

  // ------------------------------------------------------------ tabs
  $$(".tabs button").forEach(b => b.onclick = () => showTab(b.dataset.tab));
  function showTab(t) {
    tab = t;
    $$(".tabs button").forEach(b => b.setAttribute("aria-selected", String(b.dataset.tab === t)));
    $$(".view").forEach(v => v.hidden = v.id !== "v-" + t);
    render(t);
  }
  function render(t) {
    ({ repair: renderRepair, tools: renderTools, signs: renderSigns, safety: renderSafety, log: renderLog })[t]();
  }
  function renderAll() { renderRepair(); renderTools(); renderSigns(); renderSafety(); renderLog(); save(); }

  $("#newRepair").onclick = () => { cur = null; pick = { dev: null }; save(); showTab("repair"); };
  $("#saveReport").onclick = () => {
    const r = cur || state.repairs[0];
    if (!r) return toastPop("Nothing to save yet", "Start a repair first: the report lists every step, answer, reading and note.");
    download(fileName(r) + ".md", report(r));
  };

  // ------------------------------------------------------------ repair tab
  let pick = { dev: null };
  function renderRepair() {
    const v = $("#v-repair");
    if (!cur) return pick.dev ? renderSymptoms(v) : renderDevices(v);
    renderStep(v);
  }

  function renderDevices(v) {
    const last = state.repairs.find(r => r.status === "open");
    const groups = [...new Set(G.devices.map(d => d.group))];
    v.innerHTML = `<div class="wrap">
      ${last ? `<div class="resume">Carry on with <b>${esc(last.title)}</b>, at “${esc(stepTitle(last))}”. <button class="btn sm" id="resume" data-help="Resume repair">Carry on</button></div>` : ""}
      <h2 class="h">What are you fixing?</h2>
      <p class="lead">Fixie takes you through it one check at a time: what to do, which tools to use, and what a good and a bad result look like. Start with the easy checks; it only asks you to open things up when it's needed.</p>
      ${groups.map(g => `<div class="grp-h">${esc(g)}</div><div class="grid" data-help="Pick a device">${G.devices.filter(d => d.group === g).map(d =>
        `<button class="card-btn" data-dev="${d.id}">${ART.device(d.id)}<span><b>${esc(d.name)}</b>${d.noopen ? `<small>Outside checks only</small>` : ""}</span></button>`).join("")}</div>`).join("")}
      <div class="callout info"><span class="ic">i</span><div><b>New to repairs?</b>Read the Safety tab first, and check the Tools tab for the few tools that cover most jobs.</div></div>
    </div>`;
    $$("[data-dev]", v).forEach(b => b.onclick = () => { pick.dev = b.dataset.dev; renderRepair(); });
    if (last) $("#resume").onclick = () => { cur = last; save(); renderRepair(); };
  }

  function renderSymptoms(v) {
    const d = DEV[pick.dev];
    v.innerHTML = `<div class="wrap">
      <div class="crumbs"><button id="back">← All devices</button><span>${esc(d.name)}</span></div>
      <h2 class="h">What's it doing?</h2>
      <p class="lead">Pick the closest. If several fit, start with the first one in this list.</p>
      ${d.noopen ? `<div class="callout danger"><span class="ic">!</span><div><b>Don't open it</b>${esc(d.open)}</div></div>` : ""}
      <div class="grid sym" style="margin-top:12px" data-help="Pick a symptom">${symptomsFor(d).map(s =>
        `<button class="card-btn" data-sym="${s.id}"><span><b>${esc(s.name)}</b><small>${esc(s.desc)}</small></span></button>`).join("")}</div>
    </div>`;
    $("#back").onclick = () => { pick.dev = null; renderRepair(); };
    $$("[data-sym]", v).forEach(b => b.onclick = () => startRepair(d, SYM[b.dataset.sym]));
  }

  function startRepair(d, s) {
    const n = state.repairs.filter(r => r.device === d.id).length;
    cur = { id: "r" + Date.now().toString(36), title: d.name + (n ? ` (${n + 1})` : ""), device: d.id, symptom: s.id,
      created: new Date().toISOString(), updated: new Date().toISOString(), status: "open",
      path: [{ id: resolve(s.start, d) }] };
    state.repairs.unshift(cur);
    pick.dev = null; save(); renderRepair(); renderLog();
  }

  const here = r => r.path[r.path.length - 1];
  const stepTitle = r => { const s = G.steps[here(r).id]; return s ? s.t : ""; };
  const touch = () => { cur.updated = new Date().toISOString(); save(); };

  function doLines(s, tags) {
    return (s.do || []).map(x => typeof x === "string" ? x : match(x.if, tags) ? x.t : null).filter(Boolean);
  }
  const toolChips = ids => `<div class="chips">${ids.map(id => TOOL[id] ? `<button class="chip ${state.have.includes(id) ? "have" : TOOL[id].starter ? "" : "missing"}" data-tool="${id}" title="${esc(TOOL[id].why)}">${esc(TOOL[id].name)}</button>` : "").join("")}</div>`;
  const signChips = ids => `<div class="chips">${ids.map(id => SIGN[id] ? `<button class="chip" data-sign="${id}">${esc(SIGN[id].name)}</button>` : "").join("")}</div>`;

  function renderStep(v) {
    const d = DEV[cur.device], tags = tagsOf(d), at = here(cur), s = G.steps[at.id];
    const num = cur.path.length;
    const side = `<aside class="side">
      <div class="box"><h4>This repair</h4><input class="title-in" id="rTitle" value="${esc(cur.title)}" aria-label="Repair name" data-help="Repair name">
        <p style="margin-top:6px;color:var(--muted);font-size:12.5px">${esc(d.name)} · ${esc(SYM[cur.symptom].name)}</p></div>
      <div class="box" data-help="Your path"><h4>Your path</h4><ol class="trail">${cur.path.map((p, i) => {
        const st = G.steps[p.id], a = st && st.a && p.answer != null ? st.a[p.answer] : null;
        return `<li class="${i === cur.path.length - 1 ? "cur" : ""}"><button data-goto="${i}">${esc(st ? st.t : p.id)}</button>${a ? `<small>${esc(a.t)}${p.value != null && p.value !== "" ? ` · ${esc(p.value)} ${esc((st.measure || {}).unit || "")}` : ""}</small>` : ""}</li>`; }).join("")}</ol></div>
      ${d.open ? `<div class="box"><h4>Opening a ${esc(d.name.split(",")[0].toLowerCase())}</h4><p style="font-size:13px">${esc(d.open)}</p></div>` : ""}
    </aside>`;
    let body;
    if (s.result) body = resultHTML(s, d, at);
    else {
      const lines = doLines(s, tags), answers = s.a.map((a, i) => [a, i]).filter(([a]) => match(a.only, tags));
      const gate = s.open && (tags.has("mains") || tags.has("hv") || tags.has("lipo"));
      const gated = gate && !at.safe;
      body = `<article class="step">
        <div class="kicker">Step ${num} · ${esc(SYM[cur.symptom].name)}</div>
        <h2>${esc(s.t)}</h2><p class="why">${esc(s.why || "")}</p>
        ${gate ? `<div class="callout danger"><span class="ic">!</span><div><b>This step is inside the ${esc(d.name.toLowerCase())}</b>${tags.has("hv") ? "Unplug it at the wall and leave it a few minutes. Big capacitors can stay charged: check them with the meter (see Safety) before you touch anything." : tags.has("mains") ? "Unplug it at the wall: switching it off isn't enough." : "Disconnect the battery before touching the board."}
          <label data-help="Safety check"><input type="checkbox" id="safe" ${at.safe ? "checked" : ""}> ${tags.has("hv") ? "It's unplugged, and I've checked the big capacitors are discharged" : tags.has("mains") ? "It's unplugged at the wall" : "The battery is disconnected"}${s.live ? " (I'll reconnect power only to measure)" : ""}</label></div></div>` : ""}
        ${s.live ? `<div class="callout warn"><span class="ic">!</span><div><b>You'll measure with the power on</b>Use fine probes so they can't slip and short two pins. Keep one hand away from the board, and touch only the probe handles.${tags.has("mains") ? " Stay well clear of the mains side of the power supply." : ""}</div></div>` : ""}
        <div class="sect"><h4>What to do</h4><ol class="do">${lines.map(l => `<li>${esc(l)}</li>`).join("")}</ol></div>
        ${s.tools ? `<div class="sect" data-help="Tools for this step"><h4>Tools</h4>${toolChips(s.tools)}</div>` : ""}
        ${s.good || s.bad ? `<div class="sect"><h4>What to look for</h4><div class="gb">${s.good ? `<div class="g"><b>Good</b>${esc(s.good)}</div>` : ""}${s.bad ? `<div class="b"><b>Faulty</b>${esc(s.bad)}</div>` : ""}</div></div>` : ""}
        ${s.signs ? `<div class="sect" data-help="Signs for this step"><h4>What it looks like</h4>${signChips(s.signs)}</div>` : ""}
        ${s.measure ? `<div class="sect"><h4>Your reading</h4>${measureHTML(s.measure, at)}</div>` : ""}
        <div class="ask"><h3>${esc(s.q)}</h3><div class="answers" data-help="Answers">${answers.map(([a, i], k) =>
          `<button class="answer${at.answer === i ? " chosen" : ""}" data-a="${i}" ${gated ? "disabled" : ""}><kbd>${k + 1}</kbd>${esc(a.t)}</button>`).join("")}</div>
          ${gated ? `<p style="color:var(--muted);font-size:12.5px;margin-top:6px">Tick the safety check above to carry on.</p>` : ""}</div>
        <div class="sect notes"><h4>Notes</h4><textarea id="note" placeholder="Readings, part numbers, photos you took…" data-help="Step notes">${esc(at.note || "")}</textarea></div>
        <div class="navrow"><button class="btn" id="stepBack" ${cur.path.length < 2 ? "disabled" : ""} title="Go back a step (Alt+Left)">← Back</button><button class="btn ghost" id="restart" title="Start this repair again">Start again</button></div>
      </article>`;
    }
    v.innerHTML = `<div class="wrap stepper">${body}${side}</div>`;
    wireStep(v, s, at, d);
  }

  function measureHTML(m, at) {
    const inp = (id, label, val, unit) => `<label>${esc(label)}<span class="in"><input type="number" step="any" inputmode="decimal" id="${id}" value="${esc(val ?? "")}"><span class="u">${esc(unit)}</span></span></label>`;
    let h = `<div class="measure" data-help="Your reading">${inp("mv", m.label, at.value, m.unit)}`;
    if (m.near) h += inp("mn", m.near.label, at.near, m.unit);
    if (m.calc === "element") h += inp("mvolt", "Mains voltage", at.volts ?? 230, "V") + inp("mwatt", "Power on its label", at.watts, "W");
    return h + `<span class="verdict" id="verdict"></span></div>`;
  }

  // judge a reading: which answer it points to, and what to say
  function judge(m, at) {
    const v = parseFloat(at.value);
    if (!isFinite(v)) return null;
    if (m.bands) {
      for (const b of m.bands) if (b.lt == null || v < b.lt) return { a: b.a, say: b.say, cls: b.ok ? "ok" : "bad" };
    }
    if (m.near) {
      const n = parseFloat(at.near);
      if (!isFinite(n) || n <= 0) return { say: `Type in ${m.near.label.toLowerCase()} to compare.`, cls: "mid" };
      const r = v / n, ok = r >= m.near.lo && r <= m.near.hi;
      return { a: ok ? m.near.ok : m.near.bad, say: ok ? `Good: ${(r * 100).toFixed(0)}% of ${n} ${m.unit}.` : `${(r * 100).toFixed(0)}% of ${n} ${m.unit}: ${r < m.near.lo ? "too low" : "too high"}.`, cls: ok ? "ok" : "bad" };
    }
    if (m.calc === "element") {
      const V = parseFloat(at.volts), W = parseFloat(at.watts);
      if (v > 1e5) return { a: 0, say: "Open: the element is broken.", cls: "bad" };
      if (!(V > 0 && W > 0)) return { say: "Type in the voltage and power to work out what it should read.", cls: "mid" };
      const exp = V * V / W, ok = Math.abs(v - exp) / exp < 0.3;
      return { a: ok ? 1 : undefined, say: `It should read about ${exp < 10 ? exp.toFixed(1) : exp.toFixed(0)}Ω. ${ok ? "Yours is close: the element is fine." : v < exp ? "Yours is low: measure again across the element's own terminals." : "Yours is high: check the probes touch bare metal."}`, cls: ok ? "ok" : "mid" };
    }
    return null;
  }

  function showVerdict(s, at, v) {
    $$(".answer", v).forEach(b => b.classList.remove("suggest"));
    const el = $("#verdict", v); if (!el) return;
    const j = judge(s.measure, at);
    el.className = "verdict" + (j ? " " + j.cls : "");
    el.textContent = j ? j.say || "" : "";
    if (j && j.a != null) { const b = $(`.answer[data-a="${j.a}"]`, v); if (b) b.classList.add("suggest"); }
  }

  function wireStep(v, s, at, d) {
    $("#rTitle", v).onchange = e => { cur.title = e.target.value.trim() || cur.title; touch(); renderLog(); };
    $$("[data-goto]", v).forEach(b => b.onclick = () => { const i = +b.dataset.goto; if (i < cur.path.length - 1) { cur.path = cur.path.slice(0, i + 1); touch(); renderRepair(); } });
    wireChips(v);
    if (s.result) return wireResult(v, s, d);
    const note = $("#note", v); note.oninput = () => { at.note = note.value; touch(); };
    const safe = $("#safe", v); if (safe) safe.onchange = () => { at.safe = safe.checked; touch(); renderRepair(); };
    if (s.measure) {
      const bind = (id, key) => { const e = $("#" + id, v); if (e) e.oninput = () => { at[key] = e.value; touch(); showVerdict(s, at, v); }; };
      bind("mv", "value"); bind("mn", "near"); bind("mvolt", "volts"); bind("mwatt", "watts");
      if (s.measure.calc === "element" && at.volts == null) at.volts = 230;
      showVerdict(s, at, v);
    }
    $$(".answer", v).forEach(b => b.onclick = () => choose(+b.dataset.a));
    $("#stepBack", v).onclick = back;
    $("#restart", v).onclick = () => { cur.path = cur.path.slice(0, 1); delete cur.path[0].answer; touch(); renderRepair(); };
  }

  function choose(i) {
    const at = here(cur), s = G.steps[at.id], a = s.a[i];
    at.answer = i;
    cur.path.push({ id: resolve(a.go, DEV[cur.device]) });
    touch(); renderRepair(); $("#v-repair").scrollTop = 0;
  }
  function back() {
    if (cur && cur.path.length > 1) { cur.path.pop(); touch(); renderRepair(); }
  }

  function resultHTML(s, d, at) {
    return `<article class="step result">
      <div class="kicker">Likely fault</div>
      <h2>${esc(s.t)}<span class="level ${s.level}" title="How hard the fix is">${esc(s.level)}</span></h2>
      <p class="cause">${esc(s.cause)}</p>
      ${s.pro ? `<div class="callout warn"><span class="ic">!</span><div><b>Worth knowing</b>${esc(s.pro)}</div></div>` : ""}
      ${d.noopen && cur.path.length > 1 ? `<div class="callout danger"><span class="ic">!</span><div><b>The next check is inside the ${esc(d.name.toLowerCase())}</b>${esc(d.open)}</div></div>` : ""}
      <div class="sect"><h4>How to fix it</h4><ol class="do">${s.fix.map(f => `<li>${esc(f)}</li>`).join("")}</ol></div>
      ${s.parts ? `<div class="sect"><h4>Parts you may need</h4><ul class="plain">${s.parts.map(p => `<li>${esc(p)}</li>`).join("")}</ul></div>` : ""}
      ${s.tools ? `<div class="sect"><h4>Tools</h4>${toolChips(s.tools)}</div>` : ""}
      <div class="sect notes"><h4>Notes</h4><textarea id="note" placeholder="What you replaced, where you bought it, what it cost…" data-help="Step notes">${esc(at.note || "")}</textarea></div>
      <div class="done-row">
        <button class="btn primary" id="fixed" data-help="Fixed it">Fixed it</button>
        ${s.next ? `<button class="btn" id="notyet" data-help="Not fixed">Still not fixed: keep going</button>` : ""}
        <button class="btn" id="stepBack" title="Go back a step (Alt+Left)">← Try a different answer</button>
        <button class="btn ghost" id="gaveup" data-help="Stop here">Stop here</button>
        <button class="btn ghost" id="report" title="Save the repair report">Save report</button>
      </div>
      ${cur.status !== "open" ? `<p style="margin-top:10px"><span class="status ${cur.status === "fixed" ? "fixed" : "gaveup"}">${cur.status === "fixed" ? "Fixed" : "Stopped"}</span> Saved in My repairs.</p>` : ""}
    </article>`;
  }
  function wireResult(v, s, d) {
    const at = here(cur);
    const note = $("#note", v); note.oninput = () => { at.note = note.value; touch(); };
    $("#fixed", v).onclick = () => { cur.status = "fixed"; touch(); renderRepair(); renderLog(); };
    $("#gaveup", v).onclick = () => { cur.status = "stopped"; touch(); renderRepair(); renderLog(); };
    if ($("#notyet", v)) $("#notyet", v).onclick = () => { cur.status = "open"; cur.path.push({ id: resolve(s.next, d) }); touch(); renderRepair(); };
    $("#stepBack", v).onclick = () => { cur.status = "open"; back(); };
    $("#report", v).onclick = () => download(fileName(cur) + ".md", report(cur));
  }

  function wireChips(root) {
    $$("[data-tool]", root).forEach(b => b.onclick = () => popTool(b.dataset.tool));
    $$("[data-sign]", root).forEach(b => b.onclick = () => popSign(b.dataset.sign));
  }

  // ------------------------------------------------------------ popovers
  const pop = $("#pop");
  function openPop(html) {
    $("#popCard").innerHTML = `<button class="btn sm ghost x" id="popX" title="Close (Esc)">Close</button>` + html;
    pop.hidden = false; $("#popX").onclick = closePop; wireChips($("#popCard"));
    const h = $("#popCard input[data-have]"); if (h) h.onchange = () => setHave(h.dataset.have, h.checked);
  }
  function closePop() { pop.hidden = true; if (tab === "repair") renderRepair(); }
  pop.onclick = e => { if (e.target === pop) closePop(); };
  const toolCard = t => `<h3>${esc(t.name)}<span class="cost">${esc(t.cost)}</span></h3><p class="why">${esc(t.why)}</p>
    <ul class="plain">${t.how.map(h => `<li>${esc(h)}</li>`).join("")}</ul>
    ${t.app ? `<p class="applink">${esc(t.appWhy)}</p>` : ""}
    <label class="have"><input type="checkbox" data-have="${t.id}" ${state.have.includes(t.id) ? "checked" : ""}> I have this</label>`;
  const signCard = g => `<h3>${esc(g.name)}</h3>${ART.sign(g.id) ? `<div class="art">${ART.sign(g.id)}</div>` : ""}
    <dl><dt>What it looks like</dt><dd>${esc(g.look)}</dd><dt>What it causes</dt><dd>${esc(g.means)}</dd><dt>The fix</dt><dd>${esc(g.fix)}</dd></dl>`;
  const popTool = id => openPop(`<div class="ref" style="border:0;padding:0">${toolCard(TOOL[id])}</div>`);
  const popSign = id => openPop(`<div class="ref" style="border:0;padding:0">${signCard(SIGN[id])}</div>`);
  const toastPop = (t, m) => openPop(`<h3>${esc(t)}</h3><p>${esc(m)}</p>`);
  function setHave(id, on) {
    state.have = state.have.filter(x => x !== id); if (on) state.have.push(id);
    save(); renderTools();
  }

  // ------------------------------------------------------------ tools, signs, safety
  function renderTools() {
    const v = $("#v-tools"), starter = G.tools.filter(t => t.starter), extra = G.tools.filter(t => !t.starter);
    const have = starter.filter(t => state.have.includes(t.id)).length;
    const card = t => `<div class="ref">${toolCard(t)}</div>`;
    v.innerHTML = `<div class="wrap"><h2 class="h">Tools</h2>
      <p class="lead">A starter kit of about £60–£150 covers most repairs. Tick what you have: Fixie marks them in each step, and shows the ones you'd need to get with a dashed outline.</p>
      <div class="kit" data-help="Starter kit"><b>Starter kit</b><div class="meter-bar"><span style="width:${(have / starter.length * 100).toFixed(0)}%"></span></div><span>${have} of ${starter.length}</span></div>
      <div class="cards">${starter.map(card).join("")}</div>
      <div class="grp-h" style="margin-top:24px">For particular jobs</div><div class="cards">${extra.map(card).join("")}</div></div>`;
    $$("input[data-have]", v).forEach(i => i.onchange = () => setHave(i.dataset.have, i.checked));
  }
  function renderSigns() {
    $("#v-signs").innerHTML = `<div class="wrap"><h2 class="h">What to look for</h2>
      <p class="lead">The signs of a failed part. When you open something up, look over the whole board under a bright light, with a magnifier, before you measure anything.</p>
      <div class="cards">${G.signs.map(g => `<div class="ref">${signCard(g)}</div>`).join("")}</div></div>`;
  }
  function renderSafety() {
    const ic = { danger: "!", warn: "!", info: "i" };
    $("#v-safety").innerHTML = `<div class="wrap safety"><h2 class="h">Safety</h2>
      <p class="lead">Most repairs are safe with a few habits. A few things should never be opened at home.</p>
      <div class="cards">${G.safety.map(x => `<div class="ref ${x.level}"><h3><span class="callout ${x.level}" style="border:0;padding:0;margin:0;background:none"><span class="ic">${ic[x.level]}</span></span>${esc(x.t)}</h3>
        <ul class="plain">${x.points.map(p => `<li>${esc(p)}</li>`).join("")}</ul></div>`).join("")}</div></div>`;
  }

  // ------------------------------------------------------------ my repairs
  function renderLog() {
    const v = $("#v-log");
    $("#nLog").textContent = state.repairs.length || "";
    if (!state.repairs.length) { v.innerHTML = `<div class="wrap"><h2 class="h">My repairs</h2><div class="empty" style="margin-top:14px">No repairs yet. Press New repair to start one: each one is saved here as you go.</div></div>`; return; }
    v.innerHTML = `<div class="wrap"><h2 class="h">My repairs</h2><p class="lead">Every repair you start is saved on this computer, with each step, answer, reading and note.</p>
      ${state.repairs.map(r => { const d = DEV[r.device], s = SYM[r.symptom];
        return `<div class="logrow"><span style="width:30px;color:var(--accent)" class="card-btn" aria-hidden="true">${ART.device(r.device)}</span>
          <div class="main"><b>${esc(r.title)}</b> <span class="status ${r.status === "fixed" ? "fixed" : r.status === "stopped" ? "gaveup" : ""}">${r.status === "fixed" ? "Fixed" : r.status === "stopped" ? "Stopped" : "In progress"}</span>
          <small>${esc(d ? d.name : r.device)} · ${esc(s ? s.name : r.symptom)} · ${new Date(r.updated).toLocaleString()}</small><small>At: ${esc(stepTitle(r))} (${r.path.length} step${r.path.length > 1 ? "s" : ""})</small></div>
          <button class="btn sm" data-open="${r.id}" data-help="Open repair">Open</button><button class="btn sm ghost" data-rep="${r.id}" title="Save the repair report">Report</button><button class="btn sm ghost danger" data-del="${r.id}" data-help="Delete repair">Delete</button></div>`; }).join("")}</div>`;
    $$("[data-open]", v).forEach(b => b.onclick = () => { cur = state.repairs.find(r => r.id === b.dataset.open); save(); showTab("repair"); });
    $$("[data-rep]", v).forEach(b => b.onclick = () => { const r = state.repairs.find(x => x.id === b.dataset.rep); download(fileName(r) + ".md", report(r)); });
    $$("[data-del]", v).forEach(b => b.onclick = () => {
      if (b.dataset.sure !== "1") { b.dataset.sure = "1"; b.textContent = "Delete: sure?"; setTimeout(() => { if (b.isConnected) { b.dataset.sure = ""; b.textContent = "Delete"; } }, 4000); return; }
      state.repairs = state.repairs.filter(r => r.id !== b.dataset.del);
      if (cur && cur.id === b.dataset.del) cur = null;
      save(); renderLog(); renderRepair();
    });
  }

  // ------------------------------------------------------------ report
  const fileName = r => "Fixie - " + r.title.replace(/[\\/:*?"<>|]+/g, "-");
  function report(r) {
    const d = DEV[r.device], tags = tagsOf(d);
    let t = `# ${r.title}\n\n${d.name}: ${SYM[r.symptom].name}\n\nStarted ${new Date(r.created).toLocaleString()} · ${r.status === "fixed" ? "Fixed" : r.status === "stopped" ? "Stopped" : "In progress"}\n`;
    r.path.forEach((p, i) => {
      const s = G.steps[p.id]; if (!s) return;
      if (s.result) {
        t += `\n## Likely fault: ${s.t} (${s.level})\n\n${s.cause}\n\n` + s.fix.map(f => `- ${f}\n`).join("");
        if (s.parts) t += `\nParts: ${s.parts.join(", ")}\n`;
      } else {
        t += `\n## ${i + 1}. ${s.t}\n\n` + doLines(s, tags).map(l => `- ${l}\n`).join("");
        if (p.value != null && p.value !== "") t += `\nReading: ${p.value} ${s.measure ? s.measure.unit : ""}${p.near ? ` (expected ${p.near} ${s.measure.unit})` : ""}\n`;
        if (p.answer != null) t += `\n**${s.q}** ${s.a[p.answer].t}\n`;
      }
      if (p.note) t += `\nNotes: ${p.note}\n`;
    });
    return t + `\n---\nMade with Fixie (Tinkerie) by nrmluvdev\n`;
  }
  function download(name, text) {
    const a = document.createElement("a");
    a.href = URL.createObjectURL(new Blob([text], { type: "text/markdown" }));
    a.download = name; document.body.append(a); a.click(); a.remove();
    setTimeout(() => URL.revokeObjectURL(a.href), 2000);
  }

  // ------------------------------------------------------------ keys
  document.addEventListener("keydown", e => {
    if (e.key === "Escape" && !pop.hidden) return closePop();
    if (/INPUT|TEXTAREA|SELECT/.test(document.activeElement.tagName)) return;
    if (tab !== "repair" || !cur || !pop.hidden) return;
    if (e.altKey && e.key === "ArrowLeft") { e.preventDefault(); back(); return; }
    const n = parseInt(e.key, 10);
    if (n >= 1 && n <= 9 && !e.ctrlKey && !e.metaKey && !e.altKey) { const b = $$("#v-repair .answer")[n - 1]; if (b && !b.disabled) b.click(); }
  });
})();
