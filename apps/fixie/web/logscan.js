// Fixie — the error log explainer's scanner. Reads a log from Windows,
// macOS, Linux (and Raspberry Pi OS, ChromeOS) or Android, finds the
// messages in logrules.json and says what each one means. It's plain pattern
// matching against a list written by hand: no AI, nothing leaves the computer.
// Works in the window and in Node (for fixie's tests).
// Part of the Tinkerie maker suite. Created by nrmluvdev.
(function (root) {
  "use strict";
  const MAXLINE = 2000;
  const LEVELS = { 1: "Critical", 2: "Error", 3: "Warning", 4: "Information", 5: "Verbose", 0: "Information" };
  const SEV = ["critical", "problem", "warning", "info", "harmless"];

  // ---------------------------------------------------------------- formats
  const xmlText = s => s.replace(/<!\[CDATA\[([\s\S]*?)\]\]>/g, "$1").replace(/<[^>]+>/g, " ")
    .replace(/&lt;/g, "<").replace(/&gt;/g, ">").replace(/&quot;/g, '"').replace(/&apos;/g, "'").replace(/&#(\d+);/g, (_, n) => String.fromCharCode(+n)).replace(/&amp;/g, "&");
  const one = s => String(s || "").replace(/\s+/g, " ").trim();
  const evLine = (src, id, level, msg) => `[event] ${one(src)} | id ${String(id).trim()} | ${one(level) || "Information"} | ${one(msg)}`;

  // Windows Event Viewer "Save as XML", wevtutil /f:xml or /f:RenderedXml
  function fromXML(text) {
    const out = [];
    const re = /<Event[\s>][\s\S]*?<\/Event>/g;
    let m;
    while ((m = re.exec(text))) {
      const e = m[0];
      const src = (e.match(/<Provider\b[^>]*\bName=['"]([^'"]+)/) || e.match(/EventSourceName=['"]([^'"]+)/) || [])[1] || "";
      const id = (e.match(/<EventID\b[^>]*>(\d+)</) || [])[1] || "";
      const lv = (e.match(/<Level>(\d+)</) || [])[1];
      const lvText = (e.match(/<RenderingInfo[\s\S]*?<Level>([^<]+)</) || [])[1];
      const time = (e.match(/<TimeCreated\b[^>]*SystemTime=['"]([^'"]+)/) || [])[1] || "";
      let msg = (e.match(/<Message>([\s\S]*?)<\/Message>/) || [])[1];
      if (msg == null) msg = [...e.matchAll(/<Data\b[^>]*>([\s\S]*?)<\/Data>/g)].map(d => d[1]).filter(Boolean).join(" ");
      out.push({ text: evLine(src, id, lvText || LEVELS[lv] || "", xmlText(msg)), time, n: lineOf(text, m.index) });
    }
    return out;
  }
  function lineOf(text, i) { let n = 1; for (let k = text.indexOf("\n"); k !== -1 && k < i; k = text.indexOf("\n", k + 1)) n++; return n; }

  // Event Viewer "Save as text", wevtutil /f:text, PowerShell Format-List
  function fromBlocks(lines) {
    const out = [];
    let cur = null, inDesc = false;
    const flush = () => {
      if (cur && (cur.src || cur.id)) out.push({ text: evLine(cur.src, cur.id, cur.level, cur.msg.join(" ")), time: cur.time, n: cur.n });
      cur = null; inDesc = false;
    };
    lines.forEach((l, i) => {
      const t = l.trim();
      if (/^(?:Log Name|LogName)\s*:/i.test(t) || /^Event\[\d+\]:?$/i.test(t) || (/^TimeCreated\s*:/i.test(t) && (!cur || cur.id))) { flush(); cur = { src: "", id: "", level: "", time: "", msg: [], n: i + 1 }; }
      if (!cur) return;
      const kv = t.match(/^(Source|ProviderName|Provider Name|Event ID|Id|EventID|Level|LevelDisplayName|Date|TimeCreated|Description|Message)\s*:\s?(.*)$/i);
      if (kv && !(inDesc && !/^(?:Log Name|Event\[)/i.test(t) && !/^(?:Source|Event ID|Level|Date)\b/i.test(kv[1]))) {
        const k = kv[1].toLowerCase().replace(/\s/g, "");
        if (k === "source" || k === "providername") cur.src = kv[2];
        else if (k === "eventid" || k === "id") cur.id = kv[2];
        else if (k === "level" || k === "leveldisplayname") cur.level = kv[2];
        else if (k === "date" || k === "timecreated") cur.time = kv[2];
        else if (k === "description" || k === "message") { inDesc = true; if (kv[2]) cur.msg.push(kv[2]); }
        return;
      }
      if (inDesc && t) cur.msg.push(t);
    });
    flush();
    return out;
  }

  // Event Viewer "Save as CSV" (Level, Date and Time, Source, Event ID, Task Category, then the message)
  function parseCSV(text) {
    const rows = [];
    let row = [], f = "", q = false, line = 1, start = 1;
    for (let i = 0; i < text.length; i++) {
      const c = text[i];
      if (q) {
        if (c === '"') { if (text[i + 1] === '"') { f += '"'; i++; } else q = false; }
        else { if (c === "\n") line++; f += c; }
      } else if (c === '"') q = true;
      else if (c === "," || c === "\t") { row.push(f); f = ""; }
      else if (c === "\n" || c === "\r") {
        if (c === "\r" && text[i + 1] === "\n") i++;
        row.push(f); f = ""; if (row.some(x => x !== "")) rows.push({ row, n: start }); row = []; line++; start = line;
      } else f += c;
    }
    row.push(f); if (row.some(x => x !== "")) rows.push({ row, n: start });
    return rows;
  }
  function fromCSV(text) {
    const rows = parseCSV(text);
    const head = rows.shift().row.map(h => h.trim().toLowerCase());
    const col = (...names) => head.findIndex(h => names.includes(h));
    const cl = col("level", "leveldisplayname", "type"), cs = col("source", "providername", "provider"), ci = col("event id", "eventid", "id");
    const ct = col("date and time", "timecreated", "date"), cm = col("message", "description", "general");
    return rows.map(({ row, n }) => ({
      text: evLine(row[cs], row[ci], row[cl], cm >= 0 ? row[cm] : row.slice(Math.max(cs, ci, cl) + 2).join(" ")), time: row[ct] || "", n,
    }));
  }

  function guessOS(text) {
    const s = text.slice(0, 200000);
    if (/<Event[\s>]|^\s*(?:Log Name|Event\[\d+\])|^"?Level"?,"?Date and Time|ProviderName\s*:|\bFaulting application name\b|\bBugCheck\b|C:\\Windows/im.test(s)) return "windows";
    if (/panic\(cpu|com\.apple\.|\bmacOS\b|Mac OS X|\bkernel\[0\]|EXC_BAD_ACCESS|Darwin Kernel/.test(s)) return "macos";
    if (/AndroidRuntime|ActivityManager|^\s*(?:\d\d-\d\d )?[\d:.]+\s+\d+\s+\d+ [VDIWEF] |beginning of (?:main|system|crash)|logcat/m.test(s)) return "android";
    if (/\bkernel:|systemd\[\d+\]|^\[\s*\d+\.\d+\]|journalctl|\bdmesg\b|\/dev\/(?:sd[a-z]|nvme|mmcblk)|raspberrypi/m.test(s)) return "linux";
    return "";
  }

  // Splits a log into records: one line each, Windows events as one line each.
  function records(text) {
    text = String(text).replace(/^\uFEFF/, "");
    if (/<Event[\s>][\s\S]*?<EventID/.test(text.slice(0, 50000))) return { format: "Windows events (XML)", recs: fromXML(text) };
    const first = text.slice(0, 500).split(/\r?\n/).find(l => l.trim()) || "";
    if (/(?:^|[,\t])"?level"?[,\t]/i.test(first) && /event ?id/i.test(first)) return { format: "Windows events (CSV)", recs: fromCSV(text) };
    const lines = text.split(/\r?\n/);
    if (/^\s*(?:Log Name\s*:|Event\[\d+\])/m.test(text.slice(0, 20000)) || /^\s*ProviderName\s*:/m.test(text.slice(0, 20000)))
      return { format: "Windows events (text)", recs: fromBlocks(lines) };
    return { format: "", recs: lines.map((l, i) => ({ text: l, n: i + 1 })) };
  }

  // ------------------------------------------------------------------ scan
  const ERRORISH = /\b(?:error|errors|err|fail(?:ed|ure|s)?|fatal|panic|critical|crit|exception|denied|timed? ?out|corrupt(?:ed|ion)?|unable to|cannot|can't|segfault|abort(?:ed)?|warn(?:ing)?)\b/i;
  const shape = s => s.toLowerCase()
    .replace(/^\s*(?:\[\s*\d+\.\d+\]|\w{3}\s+\d+\s+[\d:]+|\d{4}-\d\d-\d\d[t ][\d:.,]+(?:[+-]\d\d:?\d\d|z)?|\d\d-\d\d\s+[\d:.]+)\s*/, "")
    .replace(/\b[0-9a-f]{2}(?::[0-9a-f]{2}){5}\b/g, "#mac").replace(/0x[0-9a-f]+/g, "#").replace(/\b\d+(?:\.\d+)*\b/g, "#")
    .replace(/[a-z]:\\[^\s"']+|\/(?:[\w.-]+\/)+[\w.-]*/g, "#path").replace(/\s+/g, " ").trim();

  function compile(rules) {
    return rules.map(r => Object.assign({}, r, { rx: new RegExp(r.re, "i") }));
  }

  function scan(text, rules, opt) {
    opt = opt || {};
    const R = rules[0] && rules[0].rx ? rules : compile(rules);
    const { format, recs } = records(text);
    // only the guessed system's messages (and those for any system), unless
    // it can't tell or is told otherwise
    const os = opt.os || guessOS(text);
    const use = os ? R.filter(r => r.os === os || r.os === "any" || (os === "android" && r.os === "linux")) : R; // Android runs Linux
    const found = new Map(), other = new Map();
    let errorish = 0;
    for (const rec of recs) {
      let t = rec.text;
      if (!t || !t.trim()) continue;
      if (t.length > MAXLINE) t = t.slice(0, MAXLINE);
      const low = t.toLowerCase();
      let hit = false;
      for (const r of use) {
        if (r.kw && !low.includes(r.kw)) continue;
        if (!r.rx.test(t)) continue;
        hit = true;
        let f = found.get(r.id);
        if (!f) found.set(r.id, f = { rule: r, count: 0, first: rec.n, last: rec.n, samples: [] });
        f.count++; f.last = rec.n;
        if (f.samples.length < (opt.samples || 5)) f.samples.push({ n: rec.n, text: t, time: rec.time || "" });
      }
      if (!hit && ERRORISH.test(t)) {
        errorish++;
        const k = shape(t);
        let o = other.get(k);
        if (!o) other.set(k, o = { count: 0, sample: t, n: rec.n });
        o.count++;
      }
    }
    const findings = [...found.values()].sort((a, b) => SEV.indexOf(a.rule.sev) - SEV.indexOf(b.rule.sev) || b.count - a.count);
    const unknown = [...other.values()].sort((a, b) => b.count - a.count);
    return { format: format || "Text log", os, records: recs.length, findings, unknown, errorish };
  }

  const api = { scan, records, guessOS, compile, SEV };
  if (typeof module === "object" && module.exports) module.exports = api;
  else root.LogScan = api;
})(typeof self !== "undefined" ? self : this);
