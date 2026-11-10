// Codie — the window: editor, report tabs and Game school.
// Part of the Tinkerie maker suite. Created by nrmluvdev.
(() => {
  "use strict";
  const $ = (s, r = document) => r.querySelector(s);
  const $$ = (s, r = document) => [...r.querySelectorAll(s)];
  const LH = 21, PAD = 10;
  const store = {
    get(k) { try { return localStorage.getItem("codie." + k); } catch { return null; } },
    set(k, v) { try { localStorage.setItem("codie." + k, v); } catch {} },
  };
  const api = (path, body) => fetch(path, {
    method: body ? "POST" : "GET",
    headers: { "X-Codie": "1", "Content-Type": "application/json" },
    body: body ? JSON.stringify(body) : undefined,
  }).then(r => { if (!r.ok) throw new Error(r.statusText); return r.json(); });
  const esc = s => String(s ?? "").replace(/[&<>"]/g, c => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" }[c]));
  const md = s => esc(s).replace(/`([^`]+)`/g, "<code>$1</code>");

  const ta = $("#code"), gutter = $("#gutter"), band = $("#band"), hband = $("#hband");
  const langSel = $("#lang"), exSel = $("#examples");
  let info = null, report = null, lessons = [], fileName = "", curLine = 0, timer = 0, seq = 0, samples = [];

  // ------------------------------------------------------------ start-up
  Promise.all([api("/api/info"), api("/api/lessons")]).then(([i, l]) => {
    info = i; lessons = l; samples = i.samples;
    $("#ver").textContent = `Codie ${i.version}`;
    const groups = {};
    for (const g of i.langs) (groups[g.group] = groups[g.group] || []).push(g);
    for (const [g, list] of Object.entries(groups)) {
      const og = document.createElement("optgroup"); og.label = g;
      for (const l of list) og.append(new Option(l.name, l.id));
      langSel.append(og);
    }
    const eg = {};
    i.samples.forEach((s, n) => (eg[s.group] = eg[s.group] || []).push([s, n]));
    for (const [g, list] of Object.entries(eg)) {
      const og = document.createElement("optgroup"); og.label = g;
      for (const [s, n] of list) og.append(new Option(s.name, n));
      exSel.append(og);
    }
    buildLessonList();
    showLesson(store.get("lesson") || lessons[0].id);
    if (i.file && i.file.code) loadFile(i.file);
    else { const saved = store.get("code"); if (saved) { ta.value = saved; langSel.value = store.get("lang") || ""; } }
    syncGutter(); analyse();
    const t = store.get("tab"); if (t && t !== "explain") tab(t);
  }).catch(e => { $("#lines").innerHTML = `<div class="empty"><b>Codie couldn't start</b>${esc(e.message)}</div>`; });

  // Keep the stream open: Codie quits when the window closes, and files
  // opened with Codie while it's running arrive here.
  (function events() {
    const es = new EventSource("/api/events");
    es.addEventListener("file", e => { try { loadFile(JSON.parse(e.data)); } catch {} });
    es.onerror = () => { es.close(); setTimeout(events, 2000); };
  })();

  function loadFile(f) {
    if (!f || !f.code) return;
    ta.value = f.code; fileName = f.name || ""; langSel.value = "";
    syncGutter(); analyse(); setCur(1);
  }

  // ------------------------------------------------------------ editor
  function syncGutter() {
    const n = ta.value.split("\n").length;
    const warn = new Set((report?.warnings || []).map(w => w.line));
    const game = new Set();
    if (report?.game?.isGame) for (const l of report.lines) if (l.game && l.game.length) for (let k = l.n; k <= (l.end || l.n); k++) game.add(k);
    let h = "";
    for (let i = 1; i <= n; i++) h += `<div data-n="${i}" class="${i === curLine ? "cur " : ""}${warn.has(i) ? "warn " : ""}${game.has(i) ? "game" : ""}">${i}</div>`;
    gutter.innerHTML = h;
    gutter.scrollTop = ta.scrollTop;
    placeBands();
  }
  function placeBands() {
    const put = (el, n) => {
      if (!n) { el.style.display = "none"; return; }
      el.style.display = "block";
      el.style.top = (PAD + (n - 1) * LH - ta.scrollTop) + "px";
    };
    put(band, curLine); put(hband, hband.dataset.n ? +hband.dataset.n : 0);
  }
  ta.addEventListener("scroll", () => { gutter.scrollTop = ta.scrollTop; placeBands(); });
  const caretLine = () => ta.value.slice(0, ta.selectionStart).split("\n").length;
  ["keyup", "click", "select"].forEach(ev => ta.addEventListener(ev, () => { const n = caretLine(); if (n !== curLine) setCur(n, true); }));
  ta.addEventListener("input", () => {
    store.set("code", ta.value);
    syncGutter();
    if ($("#live").checked) { clearTimeout(timer); timer = setTimeout(analyse, 350); }
  });
  ta.addEventListener("keydown", e => {
    if (e.key === "Tab" && !e.shiftKey && !e.ctrlKey) {
      e.preventDefault();
      const s = ta.selectionStart;
      ta.setRangeText("    ", s, ta.selectionEnd, "end");
      ta.dispatchEvent(new Event("input"));
    }
  });
  gutter.addEventListener("click", e => { const d = e.target.closest("[data-n]"); if (d) goLine(+d.dataset.n); });

  function setCur(n, fromEditor) {
    curLine = n;
    $$(".gutter .cur").forEach(d => d.classList.remove("cur"));
    const g = $(`.gutter [data-n="${n}"]`); if (g) g.classList.add("cur");
    placeBands();
    $$(".ln.sel").forEach(c => c.classList.remove("sel"));
    const card = cardFor(n);
    if (card) {
      card.classList.add("sel");
      if (fromEditor && !$("#v-explain").hidden) card.scrollIntoView({ block: "nearest" });
    }
  }
  function cardFor(n) {
    return $$(".ln").find(c => +c.dataset.n <= n && n <= +c.dataset.end);
  }
  function goLine(n) {
    const lines = ta.value.split("\n");
    let pos = 0; for (let i = 0; i < n - 1 && i < lines.length; i++) pos += lines[i].length + 1;
    ta.focus({ preventScroll: true });
    ta.setSelectionRange(pos, pos);
    const top = (n - 1) * LH;
    if (top < ta.scrollTop || top > ta.scrollTop + ta.clientHeight - LH * 2) ta.scrollTop = Math.max(0, top - ta.clientHeight / 3);
    setCur(n);
  }

  // Drag a file onto the editor, or open one.
  const ed = $("#editor");
  ed.addEventListener("dragover", e => { e.preventDefault(); ed.classList.add("drop"); });
  ed.addEventListener("dragleave", () => ed.classList.remove("drop"));
  ed.addEventListener("drop", e => {
    e.preventDefault(); ed.classList.remove("drop");
    const f = e.dataTransfer.files[0]; if (f) readLocal(f);
  });
  $("#open").onclick = () => $("#file").click();
  $("#file").onchange = e => { const f = e.target.files[0]; if (f) readLocal(f); e.target.value = ""; };
  function readLocal(f) {
    if (f.size > 2 * 1024 * 1024) { alertBar("That file is over 2 MB: Codie is made for source files."); return; }
    const r = new FileReader();
    r.onload = () => loadFile({ name: f.name, code: String(r.result) });
    r.readAsText(f);
  }
  function alertBar(t) { $("#stats").textContent = t; }

  // ------------------------------------------------------------ toolbar
  $("#explain").onclick = analyse;
  $("#clear").onclick = () => { ta.value = ""; fileName = ""; curLine = 0; store.set("code", ""); syncGutter(); analyse(); ta.focus(); };
  langSel.onchange = () => { store.set("lang", langSel.value); analyse(); };
  exSel.onchange = () => {
    const s = samples[+exSel.value]; exSel.value = "";
    if (!s) return;
    ta.value = s.code; fileName = ""; langSel.value = ""; curLine = 0;
    store.set("code", ta.value); syncGutter(); analyse(); tab("explain"); ta.scrollTop = 0;
  };
  document.addEventListener("keydown", e => { if (e.key === "Enter" && (e.ctrlKey || e.metaKey)) { e.preventDefault(); analyse(); } });
  $("#teachBtn").onclick = () => document.dispatchEvent(new KeyboardEvent("keydown", { key: "F2", bubbles: true }));
  for (const [id, cls] of [["showWhy", "no-why"], ["showClose", "no-close"], ["showNotes", "no-notes"]]) {
    const cb = $("#" + id);
    if (store.get(id) === "0") cb.checked = false;
    const apply = () => { document.body.classList.toggle(cls, !cb.checked); store.set(id, cb.checked ? "1" : "0"); };
    cb.onchange = apply; apply();
  }
  if (store.get("live") === "0") $("#live").checked = false;
  $("#live").onchange = () => store.set("live", $("#live").checked ? "1" : "0");

  $$(".tabs button").forEach(b => b.onclick = () => tab(b.dataset.tab));
  function tab(name) {
    $$(".tabs button").forEach(b => b.setAttribute("aria-selected", b.dataset.tab === name));
    $$(".view").forEach(v => v.hidden = v.id !== "v-" + name);
    store.set("tab", name);
  }

  // ------------------------------------------------------------ analysis
  function analyse() {
    clearTimeout(timer);
    const code = ta.value;
    const my = ++seq;
    if (!code.trim()) { report = null; render(); return; }
    api("/api/analyse", { code, lang: langSel.value || fileName }).then(r => { if (my === seq) { report = r; render(); } })
      .catch(e => alertBar("Couldn't explain: " + e.message));
  }

  function lineLinks(nums, max = 30) {
    const shown = nums.slice(0, max).map(n => `<button class="chip ln-link" data-go="${n}" title="Go to line ${n}">${n}</button>`).join("");
    return shown + (nums.length > max ? ` <span class="chip">+${nums.length - max} more</span>` : "");
  }

  function render() {
    const r = report;
    for (const id of ["nOutline", "nConcepts", "nGame", "nWarnings"]) $("#" + id).textContent = "";
    if (!r) {
      $("#detected").textContent = ""; $("#summary").innerHTML = ""; $("#stats").textContent = "";
      $("#lines").innerHTML = `<div class="empty"><b>Paste some code to begin</b>Each line will be explained here: what it does, and why it's there.<br>Not sure where to start? Pick something from <i>Examples</i>, or open the <i>Game school</i>.</div>`;
      for (const v of ["outline", "concepts", "game", "warnings"]) $("#" + v).innerHTML = `<div class="empty">Nothing yet.</div>`;
      syncGutter(); return;
    }
    const L = r.lang;
    $("#detected").textContent = L.name + (L.dialect ? " · " + L.dialect : "") + (L.chosen ? "" : ` · ${Math.round(L.confidence * 100)}% sure`);
    $("#summary").innerHTML = r.summary.map(p => `<p>${md(p)}</p>`).join("");
    const st = r.stats;
    $("#stats").textContent = `${st.lines} lines · ${st.code} of code · ${st.explained} explained` + (L.others && L.others.length && !L.chosen && L.confidence < 0.8 ? ` · could also be ${L.others.map(o => (info.langs.find(x => x.id === o) || {}).name).join(", ")} (pick it from Language if so)` : "");

    const concept = Object.fromEntries(r.concepts.map(c => [c.id, c.title]));
    const gameTitle = Object.fromEntries(r.game.hits.map(h => [h.id, h.title]));
    let h = "";
    for (const ln of r.lines) {
      const end = ln.end || ln.n;
      const src = ln.kind === "comment" ? ln.parts[0].code : ln.text.trim();
      h += `<div class="ln k-${esc(ln.kind)}${ln.known ? "" : " unknown"}" data-n="${ln.n}" data-end="${end}">` +
        `<div class="no">${ln.n}${end > ln.n ? "–" + end : ""}</div><div>` +
        `<div class="src" style="--d:${Math.min(ln.depth, 8)}">${esc(src)}</div>`;
      for (const p of ln.parts) {
        h += `<div class="part"><div class="what"><span class="kind ${esc(p.kind)}">${esc(kindName(p.kind))}</span>${md(p.what)}</div>${p.why ? `<div class="why">${md(p.why)}</div>` : ""}</div>`;
      }
      if (ln.notes && ln.notes.length) h += `<ul class="notes">${ln.notes.map(n => `<li>${md(n)}</li>`).join("")}</ul>`;
      const chips = (ln.tags || []).filter(t => concept[t]).map(t => `<button class="chip" data-concept="${esc(t)}" title="Concept: ${esc(concept[t])}">${esc(concept[t])}</button>`)
        .concat((ln.game || []).filter(g => gameTitle[g]).map(g => `<button class="chip g" data-gamehit="${esc(g)}" title="Game logic: ${esc(gameTitle[g])}">🎮 ${esc(gameTitle[g])}</button>`));
      if (chips.length) h += `<div class="chips">${chips.join("")}</div>`;
      h += `</div></div>`;
    }
    $("#lines").innerHTML = h || `<div class="empty">Only blank lines so far.</div>`;

    // Outline
    $("#nOutline").textContent = r.outline.length || "";
    $("#outline").innerHTML = r.outline.length ? `<ul class="tree">${r.outline.map(o =>
      `<li data-go="${o.start}" style="--d:${o.depth}" title="Go to line ${o.start}"><span class="kind ${outlineKind(o.kind)}">${esc(o.kind)}</span><span class="nm">${esc(o.name || "")}</span><span class="rng">${o.start}${o.end > o.start ? "–" + o.end : ""}</span></li>`).join("")}</ul>`
      : `<div class="empty">No functions, classes or loops found.</div>`;

    // Concepts
    $("#nConcepts").textContent = r.concepts.length || "";
    $("#concepts").innerHTML = r.concepts.length ? r.concepts.map(c =>
      `<div class="card" id="c-${esc(c.id)}"><h3>${esc(c.title)}</h3><p>${md(c.text)}</p><div class="chips">${lineLinks(c.lines)}</div></div>`).join("")
      : `<div class="empty">No concepts spotted yet.</div>`;

    // Warnings
    $("#nWarnings").textContent = r.warnings.length || "";
    $("#warnings").innerHTML = r.warnings.length ? r.warnings.map(w =>
      `<div class="warn-item" data-go="${w.line}" title="Go to line ${w.line}"><span class="no">line ${w.line}</span><div><p>${md(w.text)}</p>${w.fix ? `<p class="fix">Tip: ${md(w.fix)}</p>` : ""}</div></div>`).join("")
      : `<div class="empty"><b>Nothing to warn about</b>Codie checks for common slips such as = instead of ==, ignored errors, slow calls in game loops and risky commands. It can't run the code, so it won't catch everything.</div>`;

    renderGame(r.game);
    syncGutter();
    if (curLine) setCur(curLine);
  }

  const kindNames = { def: "function", var: "variable", assign: "set", cont: "…", close: "end", open: "start", directive: "setting", jump: "jump", print: "output", input: "input", other: "line", string: "text" };
  const kindName = k => kindNames[k] || k;
  const outlineKind = k => ({ function: "def", method: "def", event: "def", class: "class", struct: "class", interface: "class", enum: "class", loop: "loop", if: "if", switch: "switch", try: "try" }[k] || "");

  function renderGame(g) {
    const n = g.hits.length;
    $("#nGame").textContent = g.isGame ? (g.engine ? "🎮" : n || "") : "";
    let h = "";
    if (g.engine) {
      const e = g.engine;
      h += `<div class="engine"><h3>🎮 ${esc(e.name)}</h3><p>${md(e.about)}</p><p><b>How the game loop works:</b> ${md(e.loop)}</p>` +
        `<div class="section-h">One frame, step by step</div><ol class="frame">${e.frame.map(f =>
          `<li class="${f.line ? "" : "missing"}"><span class="nm">${esc(f.name)}</span><span class="when">${esc(f.when)}</span>${f.line ? `<button class="chip ln-link" data-go="${f.line}" title="Go to line ${f.line}">line ${f.line}</button>` : `<span class="chip" title="This step isn't in the code you pasted">not used here</span>`}</li>`).join("")}</ol>` +
        (e.tips && e.tips.length ? `<div class="section-h">Tips</div><ul class="tips">${e.tips.map(t => `<li>${md(t)}</li>`).join("")}</ul>` : "") + `</div>`;
    }
    if (n) {
      h += `<div class="section-h">${g.isGame ? "Game logic in this code" : "Ideas that games also use"}</div><div class="cards">` +
        g.hits.map(x => `<div class="card game" id="g-${esc(x.id)}"><h3>${esc(x.title)}</h3><p>${md(x.text)}</p><div class="chips">${lineLinks(x.lines)}</div></div>`).join("") + `</div>`;
    }
    if (!g.isGame && !g.engine) {
      h = `<div class="empty"><b>This doesn't look like game code</b>When Codie spots a game engine or game logic (a game loop, input, movement, collisions…), this tab explains how it fits together.</div>` + h;
    }
    const ls = (g.lessons || []).map(id => lessons.find(l => l.id === id)).filter(Boolean);
    h += `<div class="section-h">Learn more in the Game school</div><div class="lesson-links">` +
      (ls.length ? ls : lessons.slice(0, 4)).map(l => `<button class="btn" data-lesson="${esc(l.id)}" title="Open the lesson: ${esc(l.title)}">${esc(l.title)}</button>`).join("") + `</div>`;
    $("#game").innerHTML = h;
  }

  // Clicks inside the report: go to a line, a concept or a lesson.
  document.addEventListener("click", e => {
    const go = e.target.closest("[data-go]");
    if (go) { goLine(+go.dataset.go); return; }
    const c = e.target.closest("[data-concept]");
    if (c) { tab("concepts"); const el = $("#c-" + c.dataset.concept); if (el) el.scrollIntoView({ block: "center" }); return; }
    const g = e.target.closest("[data-gamehit]");
    if (g) { tab("game"); const el = $("#g-" + g.dataset.gamehit); if (el) el.scrollIntoView({ block: "center" }); return; }
    const l = e.target.closest("[data-lesson]");
    if (l) { showLesson(l.dataset.lesson); tab("school"); return; }
    const card = e.target.closest(".ln");
    if (card) goLine(+card.dataset.n);
  });
  $("#lines").addEventListener("mouseover", e => {
    const card = e.target.closest(".ln");
    hband.dataset.n = card ? card.dataset.n : ""; placeBands();
  });
  $("#lines").addEventListener("mouseleave", () => { hband.dataset.n = ""; placeBands(); });

  // ------------------------------------------------------------ game school
  function buildLessonList() {
    $("#lessonList").innerHTML = lessons.map(l => `<button data-lesson="${esc(l.id)}" data-help="Lesson" title="${esc(l.title)}">${esc(l.title)}</button>`).join("");
  }
  function showLesson(id) {
    const l = lessons.find(x => x.id === id) || lessons[0];
    store.set("lesson", l.id);
    $$("#lessonList button").forEach(b => b.classList.toggle("on", b.dataset.lesson === l.id));
    $("#lesson").innerHTML = `<h2>${esc(l.title)}</h2><p class="intro">${md(l.intro)}</p><ul class="points">${l.points.map(p => `<li>${md(p)}</li>`).join("")}</ul>` +
      `<div class="ex-tabs" role="tablist">${l.examples.map((x, i) => `<button data-ex="${i}" data-help="Example language" class="${i ? "" : "on"}" title="Example: ${esc(x.label)}">${esc(x.label)}</button>`).join("")}</div>` +
      `<div class="ex"><pre id="exCode"></pre><div class="ex-foot"><button class="btn ghost" id="exCopy" title="Copy this example">Copy</button><button class="btn primary" id="exExplain" title="Explain this example">Explain this example</button></div></div>`;
    let cur = 0;
    const pick = i => { cur = i; $("#exCode").textContent = l.examples[i].code; $$(".ex-tabs button").forEach(b => b.classList.toggle("on", +b.dataset.ex === i)); };
    $$(".ex-tabs button").forEach(b => b.onclick = () => pick(+b.dataset.ex));
    pick(0);
    $("#exExplain").onclick = () => {
      const x = l.examples[cur];
      ta.value = x.code; fileName = ""; langSel.value = x.lang; curLine = 0; store.set("lang", x.lang); store.set("code", x.code);
      syncGutter(); analyse(); tab("explain"); ta.scrollTop = 0;
    };
    $("#exCopy").onclick = () => { navigator.clipboard?.writeText(l.examples[cur].code).then(() => { $("#exCopy").textContent = "Copied"; setTimeout(() => { const b = $("#exCopy"); if (b) b.textContent = "Copy"; }, 1400); }).catch(() => {}); };
    $("#v-school").scrollTop = 0;
  }

  // ------------------------------------------------------------ save report
  $("#save").onclick = () => {
    if (!report) { alertBar("Explain some code first."); return; }
    const r = report, L = r.lang, plain = s => String(s || "");
    let out = `# Codie report${fileName ? ": " + fileName : ""}\n\n${r.summary.map(plain).join(" ")}\n\nLanguage: ${L.name}${L.dialect ? " (" + L.dialect + ")" : ""}\n\n## Line by line\n\n`;
    for (const ln of r.lines) {
      const end = ln.end || ln.n;
      out += `**Line ${ln.n}${end > ln.n ? "–" + end : ""}**\n\n\`\`\`\n${ln.kind === "comment" ? ln.parts[0].code : ln.text.trim()}\n\`\`\`\n\n`;
      for (const p of ln.parts) out += `- ${plain(p.what)}${p.why ? "\n  - Why: " + plain(p.why) : ""}\n`;
      for (const n of ln.notes || []) out += `- Note: ${plain(n)}\n`;
      out += "\n";
    }
    if (r.outline.length) out += `## Outline\n\n${r.outline.map(o => `${"  ".repeat(o.depth)}- ${o.kind} ${o.name || ""} (lines ${o.start}–${o.end})`).join("\n")}\n\n`;
    if (r.concepts.length) out += `## Concepts\n\n${r.concepts.map(c => `- **${c.title}** (lines ${c.lines.join(", ")}): ${c.text}`).join("\n")}\n\n`;
    if (r.warnings.length) out += `## Warnings\n\n${r.warnings.map(w => `- Line ${w.line}: ${w.text}${w.fix ? " Tip: " + w.fix : ""}`).join("\n")}\n\n`;
    if (r.game.engine) out += `## Game logic: ${r.game.engine.name}\n\n${r.game.engine.loop}\n\n${r.game.engine.frame.map((f, i) => `${i + 1}. ${f.name}: ${f.when}${f.line ? " (line " + f.line + ")" : ""}`).join("\n")}\n\n`;
    if (r.game.hits.length && r.game.isGame) out += `${r.game.hits.map(h => `- **${h.title}** (lines ${h.lines.join(", ")}): ${h.text}`).join("\n")}\n\n`;
    out += `---\nMade with Codie, part of Tinkerie by nrmluvdev.\n`;
    const a = document.createElement("a");
    a.href = URL.createObjectURL(new Blob([out], { type: "text/markdown" }));
    a.download = (fileName ? fileName.replace(/\.[^.]+$/, "") : "codie") + "-explained.md";
    document.body.append(a); a.click(); a.remove();
    setTimeout(() => URL.revokeObjectURL(a.href), 2000);
  };
})();
