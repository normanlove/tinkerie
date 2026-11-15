// Node check for the error log explainer (run by fixie's Go tests, or:
// node apps/fixie/logscan_check.js). Every rule's examples must match it in
// JavaScript, and each sample log in testdata/logs must give the findings in
// expect.json. Created by nrmluvdev for Tinkerie.
"use strict";
const fs = require("fs"), path = require("path");
const L = require("./web/logscan.js");
const rules = L.compile(JSON.parse(fs.readFileSync(path.join(__dirname, "web/logrules.json"), "utf8")).rules);
const bad = [];
for (const r of rules) for (const e of r.ex) {
  if (r.kw && !e.toLowerCase().includes(r.kw)) bad.push(`${r.id}: example lacks its keyword "${r.kw}": ${e}`);
  if (!r.rx.test(e)) bad.push(`${r.id}: example doesn't match in JavaScript: ${e}`);
}
const dir = path.join(__dirname, "testdata/logs");
const expect = JSON.parse(fs.readFileSync(path.join(dir, "expect.json"), "utf8"));
for (const [file, want] of Object.entries(expect)) {
  const res = L.scan(fs.readFileSync(path.join(dir, file), "utf8"), rules);
  const ids = res.findings.map(f => f.rule.id);
  if (want.os && res.os !== want.os) bad.push(`${file}: system guessed as "${res.os}", want ${want.os}`);
  for (const id of want.has) if (!ids.includes(id)) bad.push(`${file}: didn't find ${id} (found ${ids.join(", ")})`);
  if (want.worst) { const w = res.findings[0] && res.findings[0].rule.sev; if (w !== want.worst) bad.push(`${file}: worst finding is ${w}, want ${want.worst} (${ids.join(", ")})`); }
  for (const u of want.unknown || []) if (!res.unknown.some(x => x.sample.includes(u))) bad.push(`${file}: "${u}" isn't listed as not recognised`);
  console.log(`${file}: ${res.format}, ${res.os}, ${res.records} records -> ${ids.join(", ")}${res.unknown.length ? " | not recognised: " + res.unknown.length : ""}`);
}
if (bad.length) { console.error(bad.join("\n")); process.exit(1); }
console.log(`ok: ${rules.length} rules`);
