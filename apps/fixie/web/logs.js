// Fixie — the Error logs tab: open, paste or (in the desktop app) scan the
// computer's logs, and Fixie says which error messages it found and what they
// mean, from its own list (logrules.json, matched by logscan.js). No AI, and
// nothing leaves the computer. Part of the Tinkerie maker suite. Created by nrmluvdev.
(() => {
  "use strict";
  const $ = (s, r = document) => r.querySelector(s);
  const esc = s => String(s ?? "").replace(/[&<>"]/g, c => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" }[c]));
  const MAXREAD = 64 << 20;
  const SEVNAME = { critical: "Act now", problem: "Problem", warning: "Worth checking", info: "Good to know", harmless: "Usually harmless" };
  const OSNAME = { windows: "Windows", macos: "macOS", linux: "Linux", android: "Android", "": "not sure which system" };
  const CATNAME = { disk: "Drive", memory: "Memory", power: "Power", heat: "Heat", cpu: "Processor", graphics: "Graphics", usb: "USB", network: "Network",
    boot: "Crash or start-up", driver: "Driver", software: "Software", security: "Security", filesystem: "File system", battery: "Battery", audio: "Sound", hardware: "Hardware" };
  let H = null, rules = null, sources = null, last = null, input = { name: "", text: "" }, filter = "", osPick = "";

  const HOWTO = [
    ["Windows", `<ol><li>Open <b>Event Viewer</b> (right-click Start → Event Viewer).</li>
      <li>Open <b>Windows Logs → System</b> (then Application too), and choose <b>Filter Current Log…</b>: tick Critical, Error and Warning.</li>
      <li>Choose <b>Save Filtered Log File As…</b> and save it as <b>XML</b>, <b>Text</b> or <b>CSV</b> (Fixie can't read .evtx).</li></ol>
      <p>Or in PowerShell: <code>Get-WinEvent -LogName System -MaxEvents 1000 | Format-List | Out-File system.txt</code></p>
      <p>Blue screen dump files (.dmp) can't be read, but the System log has the stop code.</p>`],
    ["macOS", `<p>Open <b>Console</b> (Applications → Utilities), choose <b>Crash Reports</b> or <b>Log Reports</b>, select one and <b>File → Save</b>. Panic reports are in <code>/Library/Logs/DiagnosticReports</code>.</p>
      <p>Or in Terminal: <code>log show --last 1d --predicate 'messageType == error' &gt; ~/Desktop/log.txt</code></p>`],
    ["Linux and Raspberry Pi", `<p>In a terminal:</p><ul><li>After a crash or freeze, the previous start-up: <code>journalctl -b -1 -p warning &gt; log.txt</code></li>
      <li>The kernel's messages: <code>sudo dmesg &gt; dmesg.txt</code></li><li>Or open <code>/var/log/syslog</code> or <code>/var/log/kern.log</code></li>
      <li>Raspberry Pi power and heat: <code>vcgencmd get_throttled</code></li></ul>`],
    ["Android", `<p>Turn on <b>Developer options</b> (tap Build number seven times in Settings → About phone), then <b>Developer options → Take bug report</b>. Unzip it and open the big .txt file.</p>
      <p>Or with a computer and USB debugging: <code>adb logcat -d &gt; log.txt</code></p>`],
    ["ChromeOS", `<p>Open <code>chrome://system</code> and expand a log (syslog or dmesg), or turn on Linux and use <code>dmesg</code>. Fixie reads them as Linux logs.</p>`],
  ];

  function render(view, helpers) {
    H = helpers;
    if (!view.dataset.ready) build(view);
    if (!rules) H.loadJSON("logrules.json").then(j => { rules = LogScan.compile(j.rules); $("#lgCount").textContent = rules.length; }).catch(() => { $("#lgOut").innerHTML = `<div class="empty">Fixie couldn't load its list of error messages.</div>`; });
    if (!sources) H.api("/api/logs/sources").then(s => { sources = s; showSources(); }).catch(() => { sources = { available: false, what: "" }; showSources(); });
  }

  function build(view) {
    view.dataset.ready = "1";
    view.innerHTML = `<div class="wrap logs">
      <h2 class="h">Error logs</h2>
      <p class="lead">Fixie reads error logs from Windows, macOS, Linux (and Raspberry Pi OS and ChromeOS) and Android, and tells you which error messages it found, what each one means and what to do. It compares every line with a list of <b id="lgCount">160+</b> known messages it carries with it, written by hand: no AI, and nothing is sent anywhere.</p>
      <div class="lg-in" id="lgDrop">
        <div class="lg-btns">
          <button class="btn primary" id="lgScan" data-help="Scan this computer" hidden>Scan this computer</button>
          <label class="btn" data-help="Open a log file">Open a log file…<input type="file" id="lgFile" multiple hidden accept=".txt,.log,.xml,.csv,.ips,.crash,.panic,.json,text/*"></label>
          <button class="btn" id="lgPasteBtn" data-help="Paste a log">Paste a log</button>
          <label class="lg-os" data-help="Which system the log is from">From <select id="lgOS"><option value="">Work it out</option><option value="windows">Windows</option><option value="macos">macOS</option><option value="linux">Linux / Raspberry Pi</option><option value="android">Android</option></select></label>
          <button class="btn ghost" id="lgHowBtn" data-help="How to get a log">How to get a log</button>
        </div>
        <p class="lg-hint" id="lgSrcNote"></p>
        <div class="lg-paste" id="lgPaste" hidden><textarea id="lgText" rows="8" spellcheck="false" placeholder="Paste log lines here (Ctrl+V), then Explain" data-help="Log text"></textarea>
          <div><button class="btn primary" id="lgGo" data-help="Explain the log">Explain</button> <button class="btn ghost" id="lgClear">Clear</button></div></div>
        <div class="lg-how" id="lgHow" hidden>${HOWTO.map(([n, h]) => `<details><summary>${esc(n)}</summary>${h}</details>`).join("")}</div>
        <p class="lg-drop">…or drop log files anywhere on this box.</p>
      </div>
      <div id="lgOut"></div></div>`;
    $("#lgPasteBtn").onclick = () => { $("#lgPaste").hidden = !$("#lgPaste").hidden; if (!$("#lgPaste").hidden) $("#lgText").focus(); };
    $("#lgHowBtn").onclick = () => { $("#lgHow").hidden = !$("#lgHow").hidden; };
    $("#lgClear").onclick = () => { $("#lgText").value = ""; };
    $("#lgGo").onclick = () => { const t = $("#lgText").value; if (!t.trim()) return H.toast("Nothing to explain", "Paste some log lines first."); run({ name: "Pasted text", text: t }); };
    $("#lgOS").onchange = e => { osPick = e.target.value; if (input.text) run(input); };
    $("#lgFile").onchange = e => { readFiles([...e.target.files]); e.target.value = ""; };
    $("#lgScan").onclick = scanComputer;
    const drop = $("#lgDrop");
    drop.addEventListener("dragover", e => { e.preventDefault(); drop.classList.add("over"); });
    drop.addEventListener("dragleave", () => drop.classList.remove("over"));
    drop.addEventListener("drop", e => { e.preventDefault(); drop.classList.remove("over"); if (e.dataTransfer.files.length) readFiles([...e.dataTransfer.files]); });
    $("#lgOut").innerHTML = `<div class="empty">Open, paste or drop a log to see what's in it.</div>`;
  }

  function showSources() {
    const b = $("#lgScan"), n = $("#lgSrcNote");
    if (!b) return;
    b.hidden = !sources.available;
    b.title = sources.available ? "Reads " + sources.what : "";
    n.textContent = sources.available ? "Scan this computer reads " + sources.what + "." : (sources.what || "");
  }

  // Text from a file: UTF-8 or UTF-16 (Windows PowerShell's Out-File), and
  // a clear word about files Fixie can't read.
  async function fileText(f) {
    const big = f.size > MAXREAD;
    const buf = new Uint8Array(await (big ? f.slice(f.size - MAXREAD) : f).arrayBuffer());
    const head = String.fromCharCode(...buf.slice(0, 8));
    if (head.startsWith("ElfFile")) throw new Error("it's a Windows .evtx file, which only Event Viewer reads: open it there and save it as XML, Text or CSV (How to get a log)");
    if (head.startsWith("PK")) throw new Error("it's a zip file: unzip it first (an Android bug report has its log in the biggest .txt file)");
    if (head.startsWith("MDMP") || head.startsWith("PAGEDU")) throw new Error("it's a crash dump, which needs a debugger; the System event log has the blue screen's stop code instead");
    let text;
    if (buf[0] === 0xFF && buf[1] === 0xFE) text = new TextDecoder("utf-16le").decode(buf.subarray(2));
    else if (buf[0] === 0xFE && buf[1] === 0xFF) text = new TextDecoder("utf-16be").decode(buf.subarray(2));
    else if (buf.length > 3 && buf[1] === 0 && buf[3] === 0) text = new TextDecoder("utf-16le").decode(buf);
    else text = new TextDecoder("utf-8").decode(buf);
    if (/[\x00-\x08\x0E-\x1F]/.test(text.slice(0, 2000).replace(/[\t\r\n\f]/g, ""))) throw new Error("it doesn't look like a text log");
    return { text, big };
  }
  async function readFiles(files) {
    const parts = [], names = [];
    for (const f of files) {
      try {
        const { text, big } = await fileText(f);
        parts.push(files.length > 1 ? `==== ${f.name}\n${text}` : text); names.push(f.name);
        if (big) H.toast("Big file", `${f.name} is over 64 MB: Fixie read the newest end of it.`);
      } catch (e) { H.toast("Couldn't read " + f.name, e.message); }
    }
    if (parts.length) run({ name: names.join(", "), text: parts.join("\n") });
  }

  async function scanComputer() {
    const b = $("#lgScan");
    b.disabled = true; b.textContent = "Reading the logs…";
    $("#lgOut").innerHTML = `<div class="empty">Reading this computer's logs. This can take up to a minute…</div>`;
    try {
      const r = await fetch("/api/logs/scan", { method: "POST", headers: { "X-Fixie": "1" } }).then(x => { if (!x.ok) throw new Error(x.statusText); return x.json(); });
      run({ name: "This computer", text: r.text, sources: r.sources });
    } catch (e) {
      $("#lgOut").innerHTML = `<div class="empty">Couldn't read the logs: ${esc(e.message)}</div>`;
    } finally { b.disabled = false; b.textContent = "Scan this computer"; }
  }

  function run(inp) {
    input = inp;
    if (!rules) return setTimeout(() => run(inp), 200);
    $("#lgOut").innerHTML = `<div class="empty">Reading ${esc(inp.name)}…</div>`;
    setTimeout(() => {
      const t0 = performance.now();
      last = Object.assign(LogScan.scan(inp.text, rules, { os: osPick }), { name: inp.name, sources: inp.sources, ms: performance.now() - t0, when: new Date() });
      filter = "";
      show();
    }, 30);
  }

  function show() {
    const r = last, out = $("#lgOut");
    const by = {};
    for (const f of r.findings) by[f.rule.sev] = (by[f.rule.sev] || 0) + 1;
    const worst = r.findings.find(f => f.rule.sev !== "harmless");
    const verdict = !r.findings.length ? (r.unknown.length ? "No messages Fixie knows, but some lines look like errors (below)." : "No error messages found.")
      : worst ? `The most important: <b>${esc(worst.rule.t)}</b>.` : "Only messages that are usually harmless.";
    const src = r.sources ? `<details class="lg-src"><summary>What was read</summary><ul>${r.sources.map(s => `<li class="${s.ok ? "ok" : "no"}">${s.ok ? "✓" : "✗"} ${esc(s.name)}${s.ok ? ` (${(s.size / 1024).toFixed(0)} KB)` : ""}${s.note ? ` <span>${esc(s.note)}</span>` : ""}</li>`).join("")}</ul></details>` : "";
    const shown = r.findings.filter(f => !filter || f.rule.sev === filter);
    out.innerHTML = `<div class="lg-sum">
        <p><b>${esc(r.name)}</b>: ${r.records.toLocaleString()} ${/event/i.test(r.format) ? "events" : "lines"} (${esc(r.format)}, ${esc(OSNAME[r.os] || r.os)}${osPick ? ", as you chose" : ""}). ${verdict}</p>
        <div class="lg-chips" data-help="Show only one kind">${LogScan.SEV.filter(s => by[s]).map(s => `<button class="chip toggle sev-${s}${filter === s ? " on" : ""}" data-sev="${s}">${SEVNAME[s]} · ${by[s]}</button>`).join("")}
          ${r.unknown.length ? `<a class="chip" href="#lgUnknown">Not recognised · ${r.unknown.length}</a>` : ""}</div>${src}</div>
      ${shown.map(card).join("") || (r.findings.length ? "" : `<div class="empty">Fixie didn't find any messages it knows${r.unknown.length ? ", but some lines look like errors: see below" : ""}.</div>`)}
      ${r.unknown.length ? `<section class="lg-unknown" id="lgUnknown" data-help="Not recognised"><h3>Not recognised (${r.unknown.length})</h3>
        <p class="lead">These lines look like errors, but they aren't in Fixie's list. Searching for the exact message (leave out numbers and names) usually finds what it means. Most are harmless.</p>
        <ul>${r.unknown.slice(0, 60).map(u => `<li><code>${esc(u.sample.slice(0, 300))}</code>${u.count > 1 ? ` <i>× ${u.count}</i>` : ""} <small>line ${u.n}</small></li>`).join("")}</ul>${r.unknown.length > 60 ? `<p class="lg-hint">…and ${r.unknown.length - 60} more.</p>` : ""}</section>` : ""}
      <p class="lg-hint">Checked against ${rules.length} known messages in ${(r.ms / 1000).toFixed(1)} s, on this computer. A log shows what the system noticed, not always the cause: use it with Fixie's guides and your own checks.</p>`;
    out.querySelectorAll(".lg-chips [data-sev]").forEach(b => b.onclick = () => { filter = filter === b.dataset.sev ? "" : b.dataset.sev; show(); });
  }

  function card(f) {
    const r = f.rule;
    return `<article class="lg-card sev-${r.sev}" data-help="Error message found">
      <header><span class="lg-sev">${SEVNAME[r.sev]}</span><h3>${esc(r.t)}</h3><span class="lg-n">${f.count > 1 ? `× ${f.count.toLocaleString()}` : "once"}</span></header>
      <p>${esc(r.means)}</p>
      <h4>What to do</h4><ol>${r.do.map(d => `<li>${esc(d)}</li>`).join("")}</ol>
      <details data-help="The lines it found"><summary>${f.count > f.samples.length ? `First ${f.samples.length} of ${f.count} lines` : f.count > 1 ? `The ${f.count} lines` : "The line"} · ${esc(CATNAME[r.cat] || r.cat)}</summary>
        <pre>${f.samples.map(s => `<span>${s.n}</span>${esc(s.time ? s.time + "  " : "")}${esc(s.text)}`).join("\n")}</pre></details>
    </article>`;
  }

  // The report, as Markdown (Save report while on this tab).
  function report() {
    if (!last) return null;
    const r = last;
    let t = `# Error log report: ${r.name}\n\n${r.when.toLocaleString()} · ${r.records} ${/event/i.test(r.format) ? "events" : "lines"} (${r.format}, ${OSNAME[r.os] || r.os})\n`;
    if (r.sources) t += "\n" + r.sources.map(s => `- ${s.ok ? "Read" : "Couldn't read"}: ${s.name}${s.note ? ` (${s.note})` : ""}`).join("\n") + "\n";
    for (const f of r.findings) {
      t += `\n## ${SEVNAME[f.rule.sev]}: ${f.rule.t} (${f.count}×)\n\n${f.rule.means}\n\nWhat to do:\n${f.rule.do.map((d, i) => `${i + 1}. ${d}`).join("\n")}\n\n` +
        "```\n" + f.samples.map(s => `${s.n}: ${s.text}`).join("\n") + "\n```\n";
    }
    if (r.unknown.length) t += `\n## Not recognised\n\n` + r.unknown.slice(0, 100).map(u => `- ${u.sample.slice(0, 300)}${u.count > 1 ? ` (×${u.count})` : ""}`).join("\n") + "\n";
    return t + `\n---\nMade with Fixie (Tinkerie) by nrmluvdev. Explained from Fixie's own list of known messages: no AI, nothing sent online.\n`;
  }

  window.FixieLogs = { render, report };
})();
