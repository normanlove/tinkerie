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
  let cur = null;                  // the open repair in the guide on screen
  const GUIDES = ["repair", "board", "build"];   // Quick checks, Board level, My build
  const openIn = { repair: null, board: null, build: null };
  let tab = "repair", serverOK = true, saveTimer = 0;
  const TOOL = {}, SIGN = {}, CAT = {}, SYM = {}, SEC = {}, PRB = {};
  const modeOf = r => r.mode || "repair";
  // where a repair starts: a Quick checks symptom, a Board level section or a My build problem
  const entryOf = r => (modeOf(r) === "board" ? SEC : modeOf(r) === "build" ? PRB : SYM)[r.symptom] || { name: r.symptom || "" };
  function setCur(r) { cur = r; if (r) openIn[modeOf(r)] = r; else if (GUIDES.includes(tab)) openIn[tab] = null; }

  // ------------------------------------------------------------ start-up
  Promise.all([fetch("guide.json").then(r => r.json()), api("/api/state").catch(() => { serverOK = false; return null; }), api("/api/info").catch(() => null)])
    .then(([g, st, info]) => {
      G = g;
      for (const t of G.tools) TOOL[t.id] = t;
      for (const s of G.signs) SIGN[s.id] = s;
      for (const c of G.categories) CAT[c.id] = c;
      for (const s of G.symptoms) SYM[s.id] = s;
      for (const s of G.board.sections) SEC[s.id] = s;
      for (const s of G.build.problems) PRB[s.id] = s;
      if (!st || Array.isArray(st)) { try { st = JSON.parse(local.get("state") || "null"); } catch { st = null; } }
      if (st && st.repairs) state = { repairs: st.repairs || [], have: st.have || [] };
      // repairs saved before Fixie asked about the device carry the old device's id
      for (const r of state.repairs) if (!r.profile) { const l = G.legacy[r.device] || { cat: "other", power: [], has: [] }; r.profile = { name: r.title, cat: l.cat, power: [...l.power], has: [...l.has] }; }
      if (info) $("#ver").textContent = `Fixie ${info.version}`;
      const open = state.repairs.find(r => r.id === local.get("open") && r.status === "open");
      if (open) { setCur(open); tab = modeOf(open); }
      renderAll(); showTab(tab);
    })
    .catch(e => { console.error(e); $("#v-repair").innerHTML = `<div class="empty">Fixie couldn't load its guide: ${esc(e.message)}</div>`; });

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
  const tagsOf = d => new Set(d.tags);
  // a described device: its category (a shortcut only), how it's powered and what it has
  function profileTags(p) {
    const t = new Set([...(p.power || []), ...(p.has || [])]), c = CAT[p.cat];
    if (c && c.kind) t.add(c.kind);
    for (const d of G.derived) if (d.all.every(x => t.has(x)) && !(d.none || []).some(x => t.has(x))) t.add(d.tag);
    return t;
  }
  // Fixie refuses dangerous devices outright, by name or by answer (heating from the
  // mains). The same test is in fixie_test.go.
  const norm = s => " " + String(s || "").toLowerCase().replace(/[^a-z0-9]+/g, " ").trim() + " ";
  function dangerOf(p) {
    const n = norm(p.name);
    for (const d of G.danger) for (const w of d.words) { const k = norm(w).trim(); if (["", "s", "es"].some(e => n.includes(" " + k + e + " "))) return d; }
    if ((p.has || []).includes("heatmains")) return G.danger.find(d => d.id === "heat");
    return null;
  }
  function devOf(p) {
    const c = CAT[p.cat] || CAT.other;
    return { id: c.id, name: (p.name || "").trim() || c.name, tags: [...profileTags(p)], open: c.open, danger: dangerOf(p) };
  }
  // "a|b" is any of them, "a&b" is both, "!a" is not a ("&" binds tighter than "|")
  const match = (cond, tags) => !cond || cond.split("|").some(c => c.split("&").every(t => t[0] === "!" ? !tags.has(t.slice(1)) : tags.has(t)));
  // where a step really leads for this device: steps that don't apply are skipped,
  // and steps inside a device that mustn't be opened go to the professional
  function resolve(id, dev) {
    const tags = tagsOf(dev);
    for (let n = 0; n < 50; n++) {
      const s = G.steps[id];
      if (!s) return id;
      if (!s.result && s.only && !match(s.only, tags)) { id = s.skip; continue; }
      return id;
    }
    return id;
  }
  const symptomsFor = dev => G.symptoms.filter(s => match(s.for, tagsOf(dev)));

  // ------------------------------------------------------------ tabs
  $$(".tabs button").forEach(b => b.onclick = () => showTab(b.dataset.tab));
  function showTab(t) {
    tab = t;
    if (GUIDES.includes(t)) cur = openIn[t];
    $$(".tabs button").forEach(b => b.setAttribute("aria-selected", String(b.dataset.tab === t)));
    $$(".view").forEach(v => v.hidden = v.id !== "v-" + t);
    render(t);
  }
  function render(t) {
    if (GUIDES.includes(t)) return renderGuide(t);
    ({ parts: renderParts, tools: renderTools, signs: renderSigns, safety: renderSafety, log: renderLog })[t]();
  }
  function renderAll() { for (const g of GUIDES) { cur = openIn[g]; renderGuide(g); } cur = GUIDES.includes(tab) ? openIn[tab] : cur; renderParts(); renderTools(); renderSigns(); renderSafety(); renderLog(); save(); }

  $("#newRepair").onclick = () => { const g = GUIDES.includes(tab) ? tab : "repair"; openIn[g] = null; cur = null; picks[g] = newPick(); save(); showTab(g); };
  $("#saveReport").onclick = () => {
    const r = cur || state.repairs[0];
    if (!r) return toastPop("Nothing to save yet", "Start a repair first: the report lists every step, answer, reading and note.");
    download(fileName(r) + ".md", report(r));
  };

  // ------------------------------------------------------------ repair tab
  const newPick = () => ({ name: "", cat: null, power: [], has: [], stage: "describe" });
  const picks = { repair: newPick(), board: newPick(), build: newPick() };
  let pick = picks.repair, mode = "repair";       // the guide being drawn
  function renderGuide(m) {
    mode = m; pick = picks[m];
    const v = $("#v-" + m);
    if (!cur) return pick.stage === "symptom" ? renderSymptoms(v) : renderDescribe(v);
    renderStep(v);
  }
  // redraw the guide on screen (or the open repair's own guide)
  function renderRepair() { renderGuide(GUIDES.includes(tab) ? tab : cur ? modeOf(cur) : "repair"); }
  const viewNow = () => $("#v-" + mode);
  const TEXT = {
    repair: { h: "What are you fixing?", lead: "Describe it in a few taps. Fixie picks the checks from how it's powered and what it has, so it works for almost anything electronic, not just a list of products.", next: "Next: what's it doing?", pickH: "What's it doing?", pickLead: "Pick the closest. If several fit, start with the first one in this list.", ph: "For example: Bluetooth speaker, games controller, LED desk lamp" },
    board: { h: "Board level: find the failed part", lead: "For repairing and reusing old tech: go round the board section by section, measuring and testing parts, with Meterie, Scopie and Probie doing the measuring. Rescuing a board is the best way to keep it out of e-waste.", next: "Next: where to look", pickH: "Where do you want to look?", pickLead: "If you're not sure, start from the beginning: Fixie goes round the board in order.", ph: "For example: 1990s hi-fi amplifier, old games console, synth" },
    build: { h: "My build: troubleshoot your own project", lead: "For circuits you've made yourself: check power, wiring, code, buses and signals, with Meterie, Scopie, Probie, Flashie and Wirie.", next: "Next: what's wrong", pickH: "What's wrong with it?", pickLead: "Pick the closest problem.", ph: "For example: weather station, macro pad, robot arm" },
  };
  const entriesFor = d => (mode === "board" ? G.board.sections : mode === "build" ? G.build.problems : G.symptoms).filter(s => match(s.for, tagsOf(d)));

  const stopHTML = (d, extra = "") => `<div class="stopcard" role="alert"><div class="callout danger" style="margin:0"><span class="ic">!</span><div>
    <b style="font-size:17px">Fixie can't help with this</b><p style="margin:6px 0"><b>${esc(d.t)}.</b> ${esc(d.why)}</p>
    <p>For your safety Fixie gives no repair steps for these at all. Take it to the maker or a qualified repairer, or recycle it.</p>
    <p style="margin-top:6px;color:var(--muted)">If it's sparking, smoking or burning: switch it off at the wall only if that's safe to reach, keep away from it, and call the fire service.</p>${extra}</div></div></div>`;
  const dangerList = () => `<ul class="plain">${G.danger.map(d => `<li><b>${esc(d.t)}</b>: ${esc(d.why)}</li>`).join("")}</ul>`;

  function renderDescribe(v) {
    const last = state.repairs.find(r => r.status === "open" && modeOf(r) === mode && !devOf(r.profile).danger), T = TEXT[mode];
    const chip = (kind, x, on, hint) => `<button class="chip toggle${on ? " on" : ""}" data-${kind}="${x.id}" aria-pressed="${on}"${hint ? ` title="${esc(hint)}"` : ""}>${esc(x.name)}</button>`;
    v.innerHTML = `<div class="wrap describe">
      ${last ? `<div class="resume"><span>Carry on with <b>${esc(last.title)}</b>, at “${esc(stepTitle(last))}”.</span><button class="btn sm" id="resume" data-help="Resume repair">Carry on</button></div>` : ""}
      <h2 class="h">${esc(T.h)}</h2>
      <p class="lead">${esc(T.lead)}</p>
      <label class="namefield" data-help="Device name">What is it?<input id="dName" maxlength="80" autocomplete="off" placeholder="${esc(T.ph)}" value="${esc(pick.name)}"></label>
      <div id="dStop"></div>
      <div id="dRest">
        ${mode === "repair" ? `<div class="grp-h">Closest kind of thing <span class="dim">(fills in the usual answers below)</span></div>
        <div class="grid cats" data-help="Closest kind">${G.categories.map(c => `<button class="card-btn${pick.cat === c.id ? " on" : ""}" data-cat="${c.id}">${ART.device(c.id)}<span><b>${esc(c.name)}</b></span></button>`).join("")}</div>` : ""}
        <div class="grp-h">How is it powered? <span class="dim">(all that apply)</span></div>
        <div class="chips big" data-help="How it's powered">${G.power.map(x => chip("pw", x, pick.power.includes(x.id), x.hint)).join("")}</div>
        <div class="grp-h">What does it have? <span class="dim">(all that apply)</span></div>
        <div class="chips big" data-help="What it has">${G.features.map(x => chip("ft", x, pick.has.includes(x.id))).join("")}</div>
        <div class="navrow"><button class="btn primary" id="dNext" data-help="Next: symptoms">${esc(T.next)}</button><span id="dNeed" class="verdict bad" hidden>Pick at least one way it's powered.</span></div>
      </div>
      <details class="nohelp" data-help="Not covered"><summary>What Fixie won't help with</summary>${dangerList()}</details>
      ${mode === "repair" ? `<div class="callout info"><span class="ic">i</span><div><b>New to repairs?</b>Read the Safety tab first, and check the Tools tab for the few tools that cover most jobs. For component-level fault finding, use Board level.</div></div>` : `<div class="callout info"><span class="ic">i</span><div><b>Your instruments</b>Steps say which Tinkerie app to use and open it for you. With Meterie, Scopie or Probie running, a step can fill in its reading straight from them. The Parts tests tab shows how to test each kind of part.</div></div>`}
    </div>`;
    const check = () => {
      const d = dangerOf(pick), byName = dangerOf({ name: pick.name });
      $("#dStop", v).innerHTML = d ? stopHTML(d, byName ? "" : `<p style="margin-top:6px">Ticked by mistake? Untick “Heats things up from the mains” below.</p>`) : "";
      $("#dRest", v).hidden = !!byName;
      $("#dNext", v).disabled = !!d;
    };
    $("#dName", v).oninput = e => { pick.name = e.target.value; check(); };
    $$("[data-cat]", v).forEach(b => b.onclick = () => {
      const c = CAT[b.dataset.cat]; pick.cat = c.id; pick.power = [...c.power]; pick.has = [...c.has];
      renderDescribe(v); $("#dName", v).focus();
    });
    const toggle = (list, id) => list.includes(id) ? list.filter(x => x !== id) : [...list, id];
    $$("[data-pw]", v).forEach(b => b.onclick = () => { pick.power = toggle(pick.power, b.dataset.pw); b.classList.toggle("on"); b.setAttribute("aria-pressed", b.classList.contains("on")); });
    $$("[data-ft]", v).forEach(b => b.onclick = () => { pick.has = toggle(pick.has, b.dataset.ft); b.classList.toggle("on"); b.setAttribute("aria-pressed", b.classList.contains("on")); check(); });
    $("#dNext", v).onclick = () => {
      if (dangerOf(pick)) return check();
      if (!pick.power.length) { $("#dNeed", v).hidden = false; return; }
      pick.stage = "symptom"; renderRepair();
    };
    if (last) $("#resume", v).onclick = () => { setCur(last); save(); renderRepair(); };
    check();
  }

  function renderSymptoms(v) {
    const d = devOf(pick);
    if (d.danger) { pick.stage = "describe"; return renderDescribe(v); }
    v.innerHTML = `<div class="wrap">
      <div class="crumbs"><button id="back">← Change the description</button><span>${esc(d.name)}</span></div>
      <h2 class="h">${esc(TEXT[mode].pickH)}</h2>
      <p class="lead">${esc(TEXT[mode].pickLead)}</p>
      <div class="grid sym" style="margin-top:12px" data-help="Pick a symptom">${entriesFor(d).map(s =>
        `<button class="card-btn" data-sym="${s.id}"><span><b>${esc(s.name)}</b><small>${esc(s.desc)}</small></span></button>`).join("")}</div>
    </div>`;
    $("#back", v).onclick = () => { pick.stage = "describe"; renderRepair(); };
    $$("[data-sym]", v).forEach(b => b.onclick = () => startRepair(d, entriesFor(d).find(x => x.id === b.dataset.sym)));
  }

  function startRepair(d, s) {
    const n = state.repairs.filter(r => devOf(r.profile).name === d.name).length;
    setCur({ id: "r" + Date.now().toString(36), mode, title: d.name + (n ? ` (${n + 1})` : ""), symptom: s.id,
      // a Board level section about a feature (Audio path, Video path…) means the device has it
      profile: { name: pick.name.trim(), cat: pick.cat || "other", power: [...pick.power], has: [...new Set([...pick.has, ...(s.implies ? [s.implies] : [])])] },
      created: new Date().toISOString(), updated: new Date().toISOString(), status: "open",
      path: [{ id: resolve(s.start, d) }] });
    state.repairs.unshift(cur);
    picks[mode] = newPick(); save(); renderRepair(); renderLog();
  }

  const here = r => r.path[r.path.length - 1];
  const stepTitle = r => { const s = G.steps[here(r).id]; return s ? s.t : ""; };
  const touch = () => { cur.updated = new Date().toISOString(); save(); };

  function doLines(s, tags) {
    return (s.do || []).map(x => typeof x === "string" ? x : match(x.if, tags) ? x.t : null).filter(Boolean);
  }
  // a step's tool and sign chips: only the ones that fit this device
  const toolChips = (ids, tags) => `<div class="chips">${ids.filter(id => !tags || !TOOL[id] || match(TOOL[id].for, tags)).map(id => TOOL[id] ? `<button class="chip ${state.have.includes(id) ? "have" : TOOL[id].starter ? "" : "missing"}" data-tool="${id}" title="${esc(TOOL[id].why)}">${esc(TOOL[id].name)}</button>` : "").join("")}</div>`;
  const signChips = (ids, tags) => `<div class="chips">${ids.filter(id => !tags || !SIGN[id] || match(SIGN[id].for, tags)).map(id => SIGN[id] ? `<button class="chip" data-sign="${id}">${esc(SIGN[id].name)}</button>` : "").join("")}</div>`;

  function renderStep(v) {
    const d = devOf(cur.profile), tags = tagsOf(d), at = here(cur), s = G.steps[at.id];
    const num = cur.path.length;
    if (d.danger) {
      v.innerHTML = `<div class="wrap">${stopHTML(d.danger, `<div class="navrow"><button class="btn" id="dDel">Delete this repair</button></div>`)}</div>`;
      $("#dDel", v).onclick = () => { state.repairs = state.repairs.filter(r => r !== cur); setCur(null); save(); renderRepair(); renderLog(); };
      return;
    }
    const side = `<aside class="side">
      <div class="box"><h4>This repair</h4><input class="title-in" id="rTitle" value="${esc(cur.title)}" aria-label="Repair name" data-help="Repair name">
        <p style="margin-top:6px;color:var(--muted);font-size:12.5px">${esc(entryOf(cur).name)}</p>
        <p style="margin-top:4px;color:var(--muted);font-size:12.5px">${esc([...(cur.profile.power || []).map(x => (G.power.find(p => p.id === x) || {}).name), ...(cur.profile.has || []).map(x => (G.features.find(f => f.id === x) || {}).name)].filter(Boolean).join(" · "))}</p></div>
      <div class="box" data-help="Your path"><h4>Your path</h4><ol class="trail">${cur.path.map((p, i) => {
        const st = G.steps[p.id], a = st && st.a && p.answer != null ? st.a[p.answer] : null;
        return `<li class="${i === cur.path.length - 1 ? "cur" : ""}"><button data-goto="${i}">${esc(st ? st.t : p.id)}</button>${a ? `<small>${esc(a.t)}${p.value != null && p.value !== "" ? ` · ${esc(p.value)} ${esc((st.measure || {}).unit || "")}` : ""}</small>` : ""}</li>`; }).join("")}</ol></div>
      ${d.open && mode === "repair" ? `<div class="box"><h4>Opening it up</h4><p style="font-size:13px">${esc(d.open)}</p></div>` : ""}
    </aside>`;
    let body;
    if (s.result) body = resultHTML(s, d, at);
    else {
      const lines = doLines(s, tags), answers = s.a.map((a, i) => [a, i]).filter(([a]) => match(a.only, tags));
      const gate = s.open && (tags.has("mains") || tags.has("hv") || tags.has("lipo"));
      const gated = gate && !at.safe;
      body = `<article class="step">
        <div class="kicker">Step ${num} · ${esc(entryOf(cur).name)}</div>
        <h2>${esc(s.t)}</h2><p class="why">${esc(s.why || "")}</p>
        ${gate ? `<div class="callout danger"><span class="ic">!</span><div><b>This step is inside it</b>${tags.has("hv") ? "Unplug it at the wall and leave it a few minutes. Big capacitors can stay charged: check them with the meter (see Safety) before you touch anything." : tags.has("mains") ? "Unplug it at the wall: switching it off isn't enough." : "Disconnect the battery before touching the board."}
          <label data-help="Safety check"><input type="checkbox" id="safe" ${at.safe ? "checked" : ""}> ${tags.has("hv") ? "It's unplugged, and I've checked the big capacitors are discharged" : tags.has("mains") ? "It's unplugged at the wall" : "The battery is disconnected"}${s.live ? " (I'll reconnect power only to measure)" : ""}</label></div></div>` : ""}
        ${s.live ? `<div class="callout warn"><span class="ic">!</span><div><b>You'll measure with the power on</b>Use fine probes so they can't slip and short two pins. Keep one hand away from the board, and touch only the probe handles.${tags.has("mains") ? " Stay well clear of the mains side of the power supply." : ""}</div></div>` : ""}
        <div class="sect"><h4>What to do</h4><ol class="do">${lines.map(l => `<li>${esc(l)}</li>`).join("")}</ol></div>
        ${s.tools ? `<div class="sect" data-help="Tools for this step"><h4>Tools</h4>${toolChips(s.tools, tags)}</div>` : ""}
        ${s.good || s.bad ? `<div class="sect"><h4>What to look for</h4><div class="gb">${s.good ? `<div class="g"><b>Good</b>${esc(s.good)}</div>` : ""}${s.bad ? `<div class="b"><b>Faulty</b>${esc(s.bad)}</div>` : ""}</div></div>` : ""}
        ${s.signs ? `<div class="sect" data-help="Signs for this step"><h4>What it looks like</h4>${signChips(s.signs, tags)}</div>` : ""}
        ${s.pics && picsHTML(s.pics, tags) !== '<div class="pics"></div>' ? `<div class="sect" data-help="Pictures for this step"><h4>Pictures</h4>${picsHTML(s.pics, tags)}</div>` : ""}
        ${s.refs ? `<div class="sect" data-help="Reference for this step"><h4>Reference</h4>${refChips(s.refs)}</div>` : ""}
        ${s.apps ? `<div class="sect" data-help="Tinkerie apps for this step"><h4>Use Tinkerie</h4>${appsHTML(s.apps)}</div>` : ""}
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
    if (m.calc === "element") h += inp("mvolt", "Supply voltage", at.volts ?? 24, "V") + inp("mwatt", "Power on its label", at.watts, "W");
    return h + `${srcHTML(m)}<span class="verdict" id="verdict"></span></div>`;
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
      if (s.measure.calc === "element" && at.volts == null) at.volts = 24;
      wireSrc(v, s.measure, val => { at.value = val; $("#mv", v).value = val; touch(); showVerdict(s, at, v); });
      showVerdict(s, at, v);
    }
    $$(".answer", v).forEach(b => b.onclick = () => choose(+b.dataset.a));
    $("#stepBack", v).onclick = back;
    $("#restart", v).onclick = () => { cur.path = cur.path.slice(0, 1); delete cur.path[0].answer; touch(); renderRepair(); };
  }

  function choose(i) {
    const at = here(cur), s = G.steps[at.id], a = s.a[i];
    at.answer = i;
    cur.path.push({ id: resolve(a.go, devOf(cur.profile)) });
    touch(); renderRepair(); viewNow().scrollTop = 0;
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
      <div class="sect"><h4>How to fix it</h4><ol class="do">${doLines({ do: s.fix }, tagsOf(d)).map(f => `<li>${esc(f)}</li>`).join("")}</ol></div>
      ${s.salvage ? `<div class="callout info"><span class="ic">♻</span><div><b>Keep it out of e-waste</b>${esc(s.salvage)}</div></div>` : ""}
      ${s.parts ? `<div class="sect"><h4>Parts you may need</h4><ul class="plain">${s.parts.map(p => `<li>${esc(p)}</li>`).join("")}</ul></div>` : ""}
      ${s.tools ? `<div class="sect"><h4>Tools</h4>${toolChips(s.tools, tagsOf(d))}</div>` : ""}
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

  // drawings (pics.js) and reference cards (refs.js)
  const picHTML = id => PICS[id] ? `<figure class="pic" data-pic="${id}" title="Click to enlarge">${PICS[id].svg}<figcaption><b>${esc(PICS[id].t)}</b><span>${esc(PICS[id].say || "")}</span></figcaption></figure>` : "";
  // a step's pictures: plain ids, or {if, pic} shown only when the device fits
  const picsHTML = (list, tags) => `<div class="pics">${list.map(x => typeof x === "string" ? x : !tags || match(x.if, tags) ? x.pic : null).filter(Boolean).map(picHTML).join("")}</div>`;
  const refChips = ids => `<div class="chips">${ids.filter(id => REFS[id]).map(id => `<button class="chip" data-ref="${id}">${esc(REFS[id].t)}</button>`).join("")}</div>`;
  const refCard = id => `<div class="ref${REFS[id].pics ? " wide" : ""}" data-refcard="${id}"><h3>${esc(REFS[id].t)}</h3>${REFS[id].html()}${REFS[id].pics ? picsHTML(REFS[id].pics) : ""}</div>`;
  function wireRefs(root) { $$("[data-refcard]", root).forEach(c => REFS[c.dataset.refcard].wire(c)); }
  function wireChips(root) {
    $$("[data-pic]", root).forEach(f => f.onclick = () => { if (!f.closest(".pop-card")) openPop(`<div class="pic">${PICS[f.dataset.pic].svg}<figcaption><b>${esc(PICS[f.dataset.pic].t)}</b><span>${esc(PICS[f.dataset.pic].say || "")}</span></figcaption></div>`, true); });
    $$("[data-ref]", root).forEach(b => b.onclick = () => { openPop(refCard(b.dataset.ref).replace('class="ref"', 'class="ref" style="border:0;padding:0"'), true); wireRefs($("#popCard")); });
    $$("[data-openapp]", root).forEach(b => b.onclick = () => openApp(b.dataset.openapp, b));
    $$("[data-tool]", root).forEach(b => b.onclick = () => popTool(b.dataset.tool));
    $$("[data-sign]", root).forEach(b => b.onclick = () => popSign(b.dataset.sign));
  }

  // ------------------------------------------------------------ the other Tinkerie apps
  const APPS = {
    meterie: { name: "Meterie", what: "your multimeter on the computer" },
    scopie: { name: "Scopie", what: "your oscilloscope" },
    probie: { name: "Probie", what: "logic analyser and serial monitor" },
    flashie: { name: "Flashie", what: "firmware flasher" },
    wirie: { name: "Wirie", what: "your circuit design" },
  };
  const appsHTML = list => `<div class="apps">${list.map(a => `<div class="appcard"><div><b>${esc(APPS[a.app].name)}</b> <span class="dim">${esc(APPS[a.app].what)}</span><p>${esc(a.how)}</p></div>
    <button class="btn sm" data-openapp="${a.app}" title="Open ${esc(APPS[a.app].name)}">Open ${esc(APPS[a.app].name)}</button></div>`).join("")}</div>`;
  function openApp(app, btn) {
    const was = btn.textContent; btn.disabled = true; btn.textContent = "Opening…";
    api("/api/open-app", { app }).then(() => { btn.textContent = "Opened"; })
      .catch(() => { btn.textContent = `Open ${APPS[app].name} from the launcher`; })
      .finally(() => setTimeout(() => { btn.disabled = false; btn.textContent = was; }, 2500));
  }
  // a reading straight from another app: Meterie's meter, a Scopie measurement,
  // or a Probie logic channel ("meterie", "scopie:freq", "probie:duty")
  const MEAS = { freq: "frequency", period: "period", vpp: "peak-to-peak", vmax: "maximum", vmin: "minimum", mean: "mean", rms: "RMS", acrms: "AC RMS", amp: "amplitude", duty: "duty cycle", pwidth: "pulse width", rise: "rise time" };
  function srcHTML(m) {
    if (!m.src) return "";
    const [app, what] = m.src.split(":");
    const chans = app === "scopie" ? [1, 2, 3, 4].map(n => `<option value="${n - 1}">CH${n}</option>`) : app === "probie" ? [0, 1, 2, 3, 4, 5, 6, 7].map(n => `<option value="${n}">D${n}</option>`) : null;
    return `<span class="src" data-help="Reading from another app"><button class="btn sm" id="mSrc">Use ${esc(APPS[app].name)}'s ${esc(app === "meterie" ? "reading" : MEAS[what] || what)}</button>${chans ? `<select id="mCh" aria-label="Channel">${chans.join("")}</select>` : ""}<span class="dim" id="mSrcSay"></span></span>`;
  }
  const PREFIX = { p: 1e-12, n: 1e-9, u: 1e-6, "µ": 1e-6, m: 1e-3, k: 1e3, M: 1e6, G: 1e9 };
  // convert a value in base units (V, Ω, F, Hz, A, %) to the step's unit (mV, kΩ, µF, MHz…)
  function inUnit(value, base, unit) {
    if (unit === base || !unit) return value;
    if (unit.length > 1 && PREFIX[unit[0]] && unit.slice(1) === base) return value / PREFIX[unit[0]];
    return null;
  }
  const tidy = v => Math.abs(v) >= 100 ? +v.toFixed(1) : +v.toPrecision(4);
  function wireSrc(root, m, set) {
    const b = $("#mSrc", root); if (!b) return;
    const say = $("#mSrcSay", root), [app, what] = m.src.split(":");
    b.onclick = () => {
      const ch = $("#mCh", root) ? $("#mCh", root).value : 0;
      say.textContent = "Reading…"; say.className = "dim";
      api(`/api/peek?src=${app}&m=${what || ""}&ch=${ch}`).then(r => {
        if (r.ol) { set(String(1e9)); say.textContent = `${APPS[app].name} shows OL (open).`; return; }
        const v = inUnit(r.value, r.unit, m.unit);
        if (v == null) { say.textContent = `${APPS[app].name} is measuring ${r.unit || "something else"}; this needs ${m.unit}. Switch its mode and try again.`; say.className = "verdict bad"; return; }
        set(String(tidy(v))); say.textContent = `From ${APPS[app].name}: ${r.text}`;
      }).catch(() => fetch(`/api/peek?src=${app}&m=${what || ""}&ch=${ch}`, { headers: { "X-Fixie": "1" } }).then(x => x.json()).then(j => { say.textContent = j.error || "Couldn't reach it."; say.className = "verdict bad"; })
        .catch(() => { say.textContent = `Couldn't reach ${APPS[app].name}.`; say.className = "verdict bad"; }));
    };
  }

  // ------------------------------------------------------------ parts tests
  function renderParts() {
    const v = $("#v-parts"), groups = [...new Set(G.parts.map(p => p.group))];
    v.innerHTML = `<div class="wrap"><h2 class="h">Parts tests</h2>
      <p class="lead">How to test each kind of part: what it looks like, how it fails, and how to test it out of the circuit, in it, and powered up. Type a reading (or take it from Meterie or Scopie) and Fixie says whether it's good. Parts worth saving from scrap are marked.</p>
      <div class="grp-h">Reference and calculators</div>
      <div class="refgrid" data-help="Reference cards">${Object.keys(REFS).filter(k => typeof REFS[k] === "object" && REFS[k].html).map(refCard).join("")}</div>
      ${groups.map(g => `<div class="grp-h">${esc(g)}</div><div class="parts">${G.parts.filter(p => p.group === g).map(p => `
        <details class="part" data-help="Part test"><summary><b>${esc(p.name)}</b>${p.salvage && !/^(no|not)\b/i.test(p.salvage) ? `<span class="salv" title="${esc(p.salvage)}">♻ worth saving</span>` : ""}</summary>
          <p>${esc(p.what)}</p>
          ${p.pic ? `<div class="pics" style="margin-top:8px">${picHTML(p.pic)}</div>` : ""}
          ${p.refs ? `<div class="sect"><h4>Reference</h4>${refChips(p.refs)}</div>` : ""}
          <div class="sect"><h4>How it fails</h4><ul class="plain">${p.fails.map(f => `<li>${esc(f)}</li>`).join("")}</ul></div>
          ${p.tests.map((t, i) => `<div class="sect ptest"><h4>${esc(t.t)}</h4><ol class="do">${t.do.map(x => `<li>${esc(x)}</li>`).join("")}</ol>
            ${t.pic ? `<div class="pics" style="margin-top:8px">${picHTML(t.pic)}</div>` : ""}
            <div class="gb"><div class="g"><b>Good</b>${esc(t.good)}</div><div class="b"><b>Faulty</b>${esc(t.bad)}</div></div>
            ${t.app ? appsHTML([t.app]) : ""}
            ${t.measure ? `<div class="measure" data-pt="${p.id}" data-ti="${i}"><label>${esc(t.measure.label)}<span class="in"><input type="number" step="any" class="pv"><span class="u">${esc(t.measure.unit)}</span></span></label>${t.measure.near ? `<label>${esc(t.measure.near.label)}<span class="in"><input type="number" step="any" class="pn"><span class="u">${esc(t.measure.unit)}</span></span></label>` : ""}${srcHTML(t.measure)}<span class="verdict"></span></div>` : ""}
          </div>`).join("")}
          <div class="sect"><h4>Replacing it</h4><p>${esc(p.replace)}</p></div>
          ${p.salvage ? `<div class="sect"><h4>Saving it from scrap</h4><p>${esc(p.salvage)}</p></div>` : ""}
        </details>`).join("")}</div>`).join("")}</div>`;
    wireChips(v); wireRefs(v);
    $$(".measure[data-pt]", v).forEach(box => {
      const t = G.parts.find(p => p.id === box.dataset.pt).tests[+box.dataset.ti], m = t.measure, at = {};
      const show = () => {
        const j = judge(m, at), el = $(".verdict", box);
        el.className = "verdict" + (j ? " " + j.cls : ""); el.textContent = j ? j.say : "";
      };
      $(".pv", box).oninput = e => { at.value = e.target.value; show(); };
      if ($(".pn", box)) $(".pn", box).oninput = e => { at.near = e.target.value; show(); };
      wireSrc(box, m, val => { at.value = val; $(".pv", box).value = val; show(); });
    });
  }

  // ------------------------------------------------------------ popovers
  const pop = $("#pop");
  function openPop(html, wide) {
    $("#popCard").classList.toggle("wide", !!wide);
    $("#popCard").innerHTML = `<button class="btn sm ghost x" id="popX" title="Close (Esc)">Close</button>` + html;
    pop.hidden = false; $("#popX").onclick = closePop; wireChips($("#popCard"));
    const h = $("#popCard input[data-have]"); if (h) h.onchange = () => setHave(h.dataset.have, h.checked);
  }
  function closePop() { pop.hidden = true; if (GUIDES.includes(tab)) renderRepair(); }
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
      <p class="lead">Most repairs are safe with a few habits. Some things should never be opened at home, and Fixie won't give repair steps for them.</p>
      <div class="cards"><div class="ref danger" style="grid-column:1/-1"><h3><span class="callout danger" style="border:0;padding:0;margin:0;background:none"><span class="ic">!</span></span>Fixie won't help with these</h3>${dangerList()}
        <p style="color:var(--muted)">Take them to the maker or a qualified repairer, or recycle them.</p></div>${G.safety.map(x => `<div class="ref ${x.level}"><h3><span class="callout ${x.level}" style="border:0;padding:0;margin:0;background:none"><span class="ic">${ic[x.level]}</span></span>${esc(x.t)}</h3>
        <ul class="plain">${x.points.map(p => `<li>${esc(p)}</li>`).join("")}</ul></div>`).join("")}</div></div>`;
  }

  // ------------------------------------------------------------ my repairs
  function renderLog() {
    const v = $("#v-log");
    $("#nLog").textContent = state.repairs.length || "";
    if (!state.repairs.length) { v.innerHTML = `<div class="wrap"><h2 class="h">My repairs</h2><div class="empty" style="margin-top:14px">No repairs yet. Press New repair to start one: each one is saved here as you go.</div></div>`; return; }
    v.innerHTML = `<div class="wrap"><h2 class="h">My repairs</h2><p class="lead">Every repair you start is saved on this computer, with each step, answer, reading and note.</p>
      ${state.repairs.map(r => { const d = devOf(r.profile), s = entryOf(r);
        return `<div class="logrow"><span style="width:30px;color:var(--accent)" class="card-btn" aria-hidden="true">${ART.device(d.id)}</span>
          <div class="main"><b>${esc(r.title)}</b> <span class="status ${r.status === "fixed" ? "fixed" : r.status === "stopped" ? "gaveup" : ""}">${r.status === "fixed" ? "Fixed" : r.status === "stopped" ? "Stopped" : "In progress"}</span>
          <small>${d.danger ? "Not covered · " : ""}${esc({ repair: "Quick checks", board: "Board level", build: "My build" }[modeOf(r)])} · ${esc(d.name)} · ${esc(s.name)} · ${new Date(r.updated).toLocaleString()}</small><small>${d.danger ? "Fixie won't give repair steps for this, for safety." : `At: ${esc(stepTitle(r))} (${r.path.length} step${r.path.length > 1 ? "s" : ""})`}</small></div>
          <button class="btn sm" data-open="${r.id}" data-help="Open repair">Open</button><button class="btn sm ghost" data-rep="${r.id}" title="Save the repair report">Report</button><button class="btn sm ghost danger" data-del="${r.id}" data-help="Delete repair">Delete</button></div>`; }).join("")}</div>`;
    $$("[data-open]", v).forEach(b => b.onclick = () => { const r = state.repairs.find(x => x.id === b.dataset.open); setCur(r); save(); showTab(modeOf(r)); });
    $$("[data-rep]", v).forEach(b => b.onclick = () => { const r = state.repairs.find(x => x.id === b.dataset.rep); download(fileName(r) + ".md", report(r)); });
    $$("[data-del]", v).forEach(b => b.onclick = () => {
      if (b.dataset.sure !== "1") { b.dataset.sure = "1"; b.textContent = "Delete: sure?"; setTimeout(() => { if (b.isConnected) { b.dataset.sure = ""; b.textContent = "Delete"; } }, 4000); return; }
      state.repairs = state.repairs.filter(r => r.id !== b.dataset.del);
      for (const g of GUIDES) if (openIn[g] && openIn[g].id === b.dataset.del) openIn[g] = null;
      if (cur && cur.id === b.dataset.del) cur = null;
      save(); renderLog(); renderRepair();
    });
  }

  // ------------------------------------------------------------ report
  const fileName = r => "Fixie - " + r.title.replace(/[\\/:*?"<>|]+/g, "-");
  function report(r) {
    const d = devOf(r.profile), tags = tagsOf(d);
    if (d.danger) return `# ${r.title}\n\nFixie can't help with this. ${d.danger.t}: ${d.danger.why}\n\nTake it to the maker or a qualified repairer, or recycle it.\n\n---\nMade with Fixie (Tinkerie) by nrmluvdev\n`;
    let t = `# ${r.title}\n\n${d.name}: ${entryOf(r).name}\n\nStarted ${new Date(r.created).toLocaleString()} · ${r.status === "fixed" ? "Fixed" : r.status === "stopped" ? "Stopped" : "In progress"}\n`;
    r.path.forEach((p, i) => {
      const s = G.steps[p.id]; if (!s) return;
      if (s.result) {
        t += `\n## Likely fault: ${s.t} (${s.level})\n\n${s.cause}\n\n` + doLines({ do: s.fix }, tags).map(f => `- ${f}\n`).join("");
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
    if (!GUIDES.includes(tab) || !cur || !pop.hidden) return;
    if (e.altKey && e.key === "ArrowLeft") { e.preventDefault(); back(); return; }
    const n = parseInt(e.key, 10);
    if (n >= 1 && n <= 9 && !e.ctrlKey && !e.metaKey && !e.altKey) { const b = $$(`#v-${tab} .answer`)[n - 1]; if (b && !b.disabled) b.click(); }
  });
})();
