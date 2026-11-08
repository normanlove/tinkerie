// Calcie — the everyday electronics maths page inside Wirie — by nrmluvdev
"use strict";

const PREFIX = { p: 1e-12, n: 1e-9, u: 1e-6, "µ": 1e-6, m: 1e-3, "": 1, k: 1e3, K: 1e3, M: 1e6, G: 1e9 };
function parseEng(s) {
  if (typeof s === "number") return s;
  s = String(s || "").trim().replace(/\s+/g, "").replace(/(ohms?|Ω|F|H|V|A|W|Hz)$/i, "").replace(/,/g, ".");
  if (!s) return NaN;
  let m = s.match(/^([0-9]*)([pnuµmkKMGR])([0-9]+)$/); // 4k7, R47, 2u2
  if (m) { const mult = m[2] === "R" ? 1 : PREFIX[m[2]]; return parseFloat((m[1] || "0") + "." + m[3]) * mult; }
  m = s.match(/^(-?[0-9]*\.?[0-9]+(?:e-?[0-9]+)?)([pnuµmkKMGR]?)$/);
  if (m) return parseFloat(m[1]) * (m[2] === "R" ? 1 : PREFIX[m[2]]);
  return NaN;
}
function fmtEng(v, unit = "", sig = 3) {
  if (!isFinite(v)) return "—";
  if (v === 0) return "0 " + unit;
  const a = Math.abs(v); const P = [[1e9, "G"], [1e6, "M"], [1e3, "k"], [1, ""], [1e-3, "m"], [1e-6, "µ"], [1e-9, "n"], [1e-12, "p"]];
  for (const [m, p] of P) if (a >= m * 0.9995) { let n = v / m; return `${+n.toPrecision(sig)} ${p}${unit}`; }
  return `${+(v / 1e-12).toPrecision(sig)} p${unit}`;
}
const E_SERIES = {
  E6: [1.0, 1.5, 2.2, 3.3, 4.7, 6.8],
  E12: [1.0, 1.2, 1.5, 1.8, 2.2, 2.7, 3.3, 3.9, 4.7, 5.6, 6.8, 8.2],
  E24: [1.0, 1.1, 1.2, 1.3, 1.5, 1.6, 1.8, 2.0, 2.2, 2.4, 2.7, 3.0, 3.3, 3.6, 3.9, 4.3, 4.7, 5.1, 5.6, 6.2, 6.8, 7.5, 8.2, 9.1],
  E96: [1.00, 1.02, 1.05, 1.07, 1.10, 1.13, 1.15, 1.18, 1.21, 1.24, 1.27, 1.30, 1.33, 1.37, 1.40, 1.43, 1.47, 1.50, 1.54, 1.58, 1.62, 1.65, 1.69, 1.74, 1.78, 1.82, 1.87, 1.91, 1.96, 2.00, 2.05, 2.10, 2.15, 2.21, 2.26, 2.32, 2.37, 2.43, 2.49, 2.55, 2.61, 2.67, 2.74, 2.80, 2.87, 2.94, 3.01, 3.09, 3.16, 3.24, 3.32, 3.40, 3.48, 3.57, 3.65, 3.74, 3.83, 3.92, 4.02, 4.12, 4.22, 4.32, 4.42, 4.53, 4.64, 4.75, 4.87, 4.99, 5.11, 5.23, 5.36, 5.49, 5.62, 5.76, 5.90, 6.04, 6.19, 6.34, 6.49, 6.65, 6.81, 6.98, 7.15, 7.32, 7.50, 7.68, 7.87, 8.06, 8.25, 8.45, 8.66, 8.87, 9.09, 9.31, 9.53, 9.76],
};
function eValues(series, lo = 1, hi = 1e7) { const out = []; for (let d = -1; d <= 7; d++) for (const b of E_SERIES[series]) { const v = +(b * Math.pow(10, d)).toPrecision(3); if (v >= lo && v <= hi) out.push(v); } return out; }
function nearestE(v, series = "E24", mode = "near") {
  if (!(v > 0)) return NaN;
  const d = Math.floor(Math.log10(v)); const m = v / Math.pow(10, d); const list = [...E_SERIES[series], 10];
  let best = null;
  for (const b of list) { const cand = b * Math.pow(10, d); if (mode === "up" && cand < v * 0.9999) continue; if (mode === "down" && cand > v * 1.0001) continue; if (!best || Math.abs(Math.log(cand / v)) < Math.abs(Math.log(best / v))) best = cand; }
  if (best === null && mode === "down") best = E_SERIES[series][E_SERIES[series].length - 1] * Math.pow(10, d - 1);
  return +best.toPrecision(3);
}
const RCOL = [["Black", "#111"], ["Brown", "#7a4a1a"], ["Red", "#d22"], ["Orange", "#f07a10"], ["Yellow", "#f3d21b"], ["Green", "#1f9d3a"], ["Blue", "#1d55d6"], ["Violet", "#8a3ad6"], ["Grey", "#888"], ["White", "#f5f5f5"]];
const RMULT = [...RCOL.map((c, i) => [c[0], c[1], Math.pow(10, i)]), ["Gold", "#c9a227", 0.1], ["Silver", "#c0c0c0", 0.01]];
const RTOL = [["Brown", "#7a4a1a", "±1%"], ["Red", "#d22", "±2%"], ["Gold", "#c9a227", "±5%"], ["Silver", "#c0c0c0", "±10%"], ["Green", "#1f9d3a", "±0.5%"], ["Blue", "#1d55d6", "±0.25%"]];
const AWG = [[10, 2.588, 5.26, 0.00328, 55, 15], [12, 2.053, 3.31, 0.00521, 41, 9.3], [14, 1.628, 2.08, 0.00829, 32, 5.9], [16, 1.291, 1.31, 0.0132, 22, 3.7], [18, 1.024, 0.823, 0.0210, 16, 2.3], [20, 0.812, 0.518, 0.0333, 11, 1.5], [22, 0.644, 0.326, 0.0530, 7, 0.92], [24, 0.511, 0.205, 0.0842, 3.5, 0.577], [26, 0.405, 0.129, 0.134, 2.2, 0.361], [28, 0.321, 0.0810, 0.213, 1.4, 0.226], [30, 0.255, 0.0509, 0.339, 0.86, 0.142]];
// [awg, dia mm, area mm², ohm/m, chassis-wiring amps (approx), power-transmission amps]

const ICONS = {
  ohm: '<path d="M4 18h4a6 6 0 1 1 8 0h4" />', band: '<path d="M2 12h3M19 12h3"/><rect x="5" y="8" width="14" height="8" rx="4"/><path d="M9 8v8M12 8v8M15 8v8"/>',
  led: '<path d="M8 14V9a4 4 0 0 1 8 0v5"/><path d="M6 14h12M10 14v6M14 14v4"/><path d="M18 4l2-2M20 7h2"/>', div: '<path d="M12 2v4M12 18v4"/><rect x="9" y="6" width="6" height="5" rx="1"/><rect x="9" y="13" width="6" height="5" rx="1"/><path d="M15 12h5"/>',
  timer: '<circle cx="12" cy="13" r="8"/><path d="M12 9v4l3 2M9 2h6"/>', wave: '<path d="M2 12c3-8 5-8 8 0s5 8 8 0 3-4 4-4"/>', bat: '<rect x="2" y="7" width="17" height="10" rx="2"/><path d="M22 11v2M6 10v4M9 10v4"/>',
  wire: '<path d="M3 17c4 0 4-10 9-10s5 10 9 10"/><circle cx="3" cy="17" r="1.5"/><circle cx="21" cy="17" r="1.5"/>', trace: '<path d="M3 6h7l4 6h7M3 18h18"/>', e: '<path d="M4 20V4h10M4 12h8M4 20h10"/>', sp: '<path d="M3 8h4l2-3 2 6 2-6 2 3h6M3 16h18"/>', cap: '<path d="M4 12h6M14 12h6M10 5v14M14 5v14"/>',
};
function icon(n) { return `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">${ICONS[n]}</svg>`; }
const field = (id, label, val, unit, hint) => `<label class="fld"><span>${label}</span><div class="inp"><input id="${id}" value="${val}" spellcheck="false" autocomplete="off">${unit ? `<em>${unit}</em>` : ""}</div>${hint ? `<small>${hint}</small>` : ""}</label>`;
const sel = (id, label, opts, v) => `<label class="fld"><span>${label}</span><div class="inp"><select id="${id}">${opts.map(o => `<option value="${o[0]}" ${o[0] == v ? "selected" : ""}>${o[1]}</option>`).join("")}</select></div></label>`;
const out = (id, label, big) => `<div class="out${big ? " big" : ""}"><span>${label}</span><b id="${id}">—</b></div>`;

function buildCalcie(root) {
  root.innerHTML = `
  <div class="calc-head"><div><h1>Calcie</h1><p>All the everyday electronics maths in one place. Type values like <code>4k7</code>, <code>100n</code>, <code>2.2u</code> or <code>1M</code> — results update as you type.</p></div><input id="cq" class="calc-search" placeholder="Find a calculator…"></div>
  <div class="calc-grid">
  <section class="card" data-k="ohm law voltage current resistance power"><h3>${icon("ohm")}Ohm's law & power</h3><p class="sub">Know any two, get the rest.</p>
    <div class="row">${sel("ohm_k", "I know", [["VI", "Voltage & current"], ["VR", "Voltage & resistance"], ["IR", "Current & resistance"], ["PV", "Power & voltage"], ["PI", "Power & current"], ["PR", "Power & resistance"]], "VR")}</div>
    <div class="row">${field("ohm_a", "Voltage", "5", "V")}${field("ohm_b", "Resistance", "220", "Ω")}</div>
    <div class="outs">${out("ohm_V", "Voltage")}${out("ohm_I", "Current")}${out("ohm_R", "Resistance")}${out("ohm_P", "Power")}</div></section>

  <section class="card" data-k="resistor colour color code bands"><h3>${icon("band")}Resistor colour code</h3><p class="sub">Pick the bands — or type a value to see its colours.</p>
    <div class="resvis" id="rc_vis"></div>
    <div class="row">${sel("rc_n", "Bands", [["4", "4 bands"], ["5", "5 bands"]], "4")}${field("rc_val", "Value → colours", "4k7", "Ω")}</div>
    <div class="bands" id="rc_bands"></div>
    <div class="outs">${out("rc_out", "Resistance", true)}${out("rc_tol", "Tolerance")}</div></section>

  <section class="card" data-k="led resistor current limiting"><h3>${icon("led")}LED resistor</h3><p class="sub">Every LED needs one. This finds it.</p>
    <div class="row">${field("led_vs", "Supply", "5", "V")}${sel("led_c", "LED colour", [["2.0", "Red (2.0V)"], ["2.1", "Yellow (2.1V)"], ["2.2", "Orange (2.2V)"], ["3.0", "Green (3.0V)"], ["3.1", "Blue (3.1V)"], ["3.2", "White (3.2V)"], ["1.3", "Infrared (1.3V)"], ["custom", "Custom…"]], "2.0")}</div>
    <div class="row">${field("led_vf", "Forward voltage", "2.0", "V")}${field("led_i", "Current", "10", "mA", "5–20mA for normal LEDs")}${field("led_n", "LEDs in series", "1", "")}</div>
    <div class="outs">${out("led_r", "Exact")}${out("led_e", "Use this (E12)", true)}${out("led_p", "Resistor power")}${out("led_w", "Rating")}</div></section>

  <section class="card" data-k="voltage divider resistor ratio adc"><h3>${icon("div")}Voltage divider</h3><p class="sub">Vout = Vin × R2 / (R1 + R2). Or let Calcie pick resistors.</p>
    <div class="row">${field("vd_in", "Vin", "5", "V")}${field("vd_r1", "R1 (top)", "10k", "Ω")}${field("vd_r2", "R2 (bottom)", "20k", "Ω")}</div>
    <div class="outs">${out("vd_out", "Vout", true)}${out("vd_i", "Current draw")}</div>
    <div class="row">${field("vd_t", "Target Vout", "3.3", "V", "finds the best E24 pair")}</div>
    <div class="outs">${out("vd_pair", "Best pair", true)}${out("vd_err", "Gives")}</div></section>

  <section class="card" data-k="555 timer astable monostable oscillator blink"><h3>${icon("timer")}555 timer</h3><p class="sub">Blinkers, tones and one-shot pulses.</p>
    <div class="row">${sel("t5_m", "Mode", [["a", "Astable (oscillator)"], ["m", "Monostable (one pulse)"]], "a")}</div>
    <div class="row">${field("t5_r1", "R1", "1k", "Ω")}${field("t5_r2", "R2", "10k", "Ω")}${field("t5_c", "C", "10u", "F")}</div>
    <div class="outs">${out("t5_f", "Frequency", true)}${out("t5_d", "Duty cycle")}${out("t5_h", "Time high")}${out("t5_l", "Time low")}</div></section>

  <section class="card" data-k="rc lc filter cutoff frequency resonance time constant"><h3>${icon("wave")}RC / LC filters</h3><p class="sub">Cut-off frequencies, time constants and resonance.</p>
    <div class="row">${field("f_r", "R", "10k", "Ω")}${field("f_c", "C", "100n", "F")}${field("f_l", "L", "10u", "H")}</div>
    <div class="outs">${out("f_rc", "RC cut-off", true)}${out("f_tau", "RC time constant τ")}${out("f_lc", "LC resonance")}${out("f_rl", "RL cut-off")}</div></section>

  <section class="card" data-k="battery runtime life mah sleep"><h3>${icon("bat")}Battery runtime</h3><p class="sub">How long will it last? Includes sleep mode.</p>
    <div class="row">${field("b_cap", "Capacity", "2000", "mAh")}${field("b_on", "Active current", "80", "mA")}${field("b_duty", "Active time", "100", "%")}</div>
    <div class="row">${field("b_sl", "Sleep current", "50", "µA")}${field("b_der", "Usable capacity", "85", "%", "batteries never give 100%")}</div>
    <div class="outs">${out("b_avg", "Average current")}${out("b_h", "Runtime", true)}</div></section>

  <section class="card wide" data-k="wire gauge awg current cable voltage drop"><h3>${icon("wire")}Wire gauge vs current</h3><p class="sub">Pick a wire thick enough (with a 40% safety margin) — and see the voltage you'll lose.</p>
    <div class="row">${field("w_i", "Current", "3", "A")}${field("w_len", "Wire length (one way)", "1", "m")}${field("w_v", "Supply", "12", "V")}</div>
    <div class="outs">${out("w_rec", "Recommended", true)}${out("w_drop", "Voltage drop")}</div>
    <div class="tbl" id="w_tbl"></div></section>

  <section class="card" data-k="pcb trace track width ipc 2221 current"><h3>${icon("trace")}PCB trace width</h3><p class="sub">IPC-2221 — how wide should a track be for this current?</p>
    <div class="row">${field("tw_i", "Current", "1", "A")}${field("tw_t", "Temp rise", "10", "°C")}</div>
    <div class="row">${sel("tw_oz", "Copper", [["0.5", "0.5 oz (17µm)"], ["1", "1 oz (35µm)"], ["2", "2 oz (70µm)"]], "1")}${sel("tw_l", "Layer", [["e", "Outer (top/bottom)"], ["i", "Inner"]], "e")}${field("tw_len", "Length", "50", "mm")}</div>
    <div class="outs">${out("tw_w", "Minimum width", true)}${out("tw_mil", "In mils")}${out("tw_r", "Resistance")}${out("tw_d", "Voltage drop")}</div>
    <div class="row btns"><button class="btn sm" id="tw_sig">Use for signal tracks</button><button class="btn sm" id="tw_pow">Use for power tracks</button></div></section>

  <section class="card" data-k="e series standard resistor value e12 e24 e96 nearest"><h3>${icon("e")}Nearest standard value</h3><p class="sub">Round any value to real parts you can buy.</p>
    <div class="row">${field("es_v", "Value", "3300.5", "")}${sel("es_s", "Series", [["E6", "E6 (±20%)"], ["E12", "E12 (±10%)"], ["E24", "E24 (±5%)"], ["E96", "E96 (±1%)"]], "E12")}</div>
    <div class="outs">${out("es_d", "Next down")}${out("es_n", "Nearest", true)}${out("es_u", "Next up")}</div></section>

  <section class="card" data-k="series parallel resistors combine"><h3>${icon("sp")}Series & parallel</h3><p class="sub">Combine resistors (or capacitors). Separate with commas.</p>
    <div class="row">${field("sp_v", "Values", "10k, 4k7, 2k2", "")}</div>
    <div class="outs">${out("sp_s", "Resistors in series")}${out("sp_p", "Resistors in parallel", true)}</div><p class="note">Capacitors are the other way round: parallel adds up.</p></section>

  <section class="card" data-k="capacitor code 104 marking ceramic"><h3>${icon("cap")}Capacitor code</h3><p class="sub">Decode the 3-digit marking (e.g. 104) or make one.</p>
    <div class="row">${field("cc_code", "Code on part", "104", "")}${field("cc_val", "Value → code", "4.7n", "F")}</div>
    <div class="outs">${out("cc_out", "Value", true)}${out("cc_code_out", "Code")}</div></section>
  </div>
  <div class="calc-foot">Calcie is part of Wirie · made by nrmluvdev · results are guides — always check datasheets for real-world limits.</div>`;
  const $ = id => root.querySelector("#" + id);
  const num = id => parseEng($(id).value);
  const set = (id, v) => { $(id).textContent = v; };

  function ohm() {
    const k = $("ohm_k").value; const lab = { V: ["Voltage", "V"], I: ["Current", "A"], R: ["Resistance", "Ω"], P: ["Power", "W"] };
    const la = $("ohm_a").closest(".fld"), lb = $("ohm_b").closest(".fld");
    la.querySelector("span").textContent = lab[k[0]][0]; la.querySelector("em").textContent = lab[k[0]][1];
    lb.querySelector("span").textContent = lab[k[1]][0]; lb.querySelector("em").textContent = lab[k[1]][1];
    const a = num("ohm_a"), b = num("ohm_b"); let V, I, R;
    switch (k) { case "VI": V = a; I = b; R = V / I; break; case "VR": V = a; R = b; I = V / R; break; case "IR": I = a; R = b; V = I * R; break; case "PV": V = b; I = a / V; R = V / I; break; case "PI": I = b; V = a / I; R = V / I; break; case "PR": R = b; I = Math.sqrt(a / R); V = I * R; break; }
    set("ohm_V", fmtEng(V, "V")); set("ohm_I", fmtEng(I, "A")); set("ohm_R", fmtEng(R, "Ω")); set("ohm_P", fmtEng(V * I, "W"));
  }
  // resistor colours
  const rcState = { b: [4, 7, 2, 2], n: 4 };
  function rcRender() {
    const n = +$("rc_n").value; rcState.n = n;
    const defs = n === 4 ? [["1st digit", RCOL], ["2nd digit", RCOL], ["Multiplier", RMULT], ["Tolerance", RTOL]] : [["1st digit", RCOL], ["2nd digit", RCOL], ["3rd digit", RCOL], ["Multiplier", RMULT], ["Tolerance", RTOL]];
    while (rcState.b.length < n) rcState.b.splice(n - 2, 0, 0); while (rcState.b.length > n) rcState.b.splice(2, 1);
    $("rc_bands").innerHTML = defs.map((d, i) => `<div class="band"><span>${d[0]}</span><div class="sw">${d[1].map((c, j) => `<button title="${c[0]}" data-b="${i}" data-j="${j}" class="${rcState.b[i] === j ? "on" : ""}" style="background:${c[1]}"></button>`).join("")}</div></div>`).join("");
    const digits = rcState.b.slice(0, n - 2).reduce((a, d) => a * 10 + d, 0);
    const mult = RMULT[rcState.b[n - 2]][2]; const ohms = digits * mult;
    set("rc_out", fmtEng(ohms, "Ω")); set("rc_tol", RTOL[rcState.b[n - 1]][2]);
    const cols = rcState.b.map((j, i) => (i === n - 1 ? RTOL[j][1] : i === n - 2 ? RMULT[j][1] : RCOL[j][1]));
    $("rc_vis").innerHTML = `<svg viewBox="0 0 260 60"><path d="M0 30h50M210 30h50" stroke="#bfc5cc" stroke-width="4"/><rect x="45" y="12" width="170" height="36" rx="18" fill="#d9c28f"/><rect x="45" y="12" width="170" height="36" rx="18" fill="url(#rg)"/>${cols.map((c, i) => `<rect x="${(i === n - 1 ? 182 : 72 + i * (n === 4 ? 24 : 20))}" y="12" width="11" height="36" fill="${c}"/>`).join("")}<defs><linearGradient id="rg" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#fff" stop-opacity=".45"/><stop offset=".5" stop-color="#fff" stop-opacity="0"/><stop offset="1" stop-color="#000" stop-opacity=".25"/></linearGradient></defs></svg>`;
    root.querySelectorAll("#rc_bands button").forEach(b => b.onclick = () => { rcState.b[+b.dataset.b] = +b.dataset.j; rcRender(); });
  }
  function rcFromValue() {
    const v = num("rc_val"); if (!(v > 0)) return; const n = +$("rc_n").value; const digits = n - 2;
    let e = Math.floor(Math.log10(v)) - (digits - 1); let sig = Math.round(v / Math.pow(10, e)); if (sig >= Math.pow(10, digits)) { sig = Math.round(sig / 10); e++; }
    const mi = e === -1 ? 10 : e === -2 ? 11 : e; if (mi < 0 || mi > 11) return;
    const ds = String(sig).padStart(digits, "0").split("").map(Number);
    rcState.b = [...ds, mi, n === 4 ? 2 : 0]; rcRender();
  }
  function led() {
    if ($("led_c").value !== "custom") $("led_vf").value = $("led_c").value;
    const vs = num("led_vs"), vf = num("led_vf") * (num("led_n") || 1), i = num("led_i") / 1000;
    if (vs <= vf) { set("led_r", "Supply too low"); set("led_e", "—"); set("led_p", "—"); set("led_w", "—"); return; }
    const r = (vs - vf) / i, e = nearestE(r, "E12", "up"), p = Math.pow((vs - vf) / e, 2) * e;
    set("led_r", fmtEng(r, "Ω")); set("led_e", fmtEng(e, "Ω")); set("led_p", fmtEng(p, "W")); set("led_w", p < 0.125 ? "1/8 W is fine" : p < 0.25 ? "use 1/4 W" : p < 0.5 ? "use 1/2 W" : "use 1 W+");
  }
  function vdiv() {
    const vin = num("vd_in"), r1 = num("vd_r1"), r2 = num("vd_r2");
    set("vd_out", fmtEng(vin * r2 / (r1 + r2), "V")); set("vd_i", fmtEng(vin / (r1 + r2), "A"));
    const t = num("vd_t"); if (!(t > 0 && t < vin)) { set("vd_pair", "target must be below Vin"); set("vd_err", "—"); return; }
    let best = null; const vals = eValues("E24", 1000, 100000);
    for (const a of vals) for (const b of vals) { const v = vin * b / (a + b); const err = Math.abs(v - t); if (!best || err < best.err - 1e-12 || (Math.abs(err - best.err) < 1e-12 && a + b < best.a + best.b)) best = { a, b, err, v }; }
    set("vd_pair", `R1 ${fmtEng(best.a, "Ω")}  ·  R2 ${fmtEng(best.b, "Ω")}`); set("vd_err", `${fmtEng(best.v, "V", 4)} (${(best.v / t * 100 - 100).toFixed(2)}%)`);
  }
  function t555() {
    const m = $("t5_m").value; const r1 = num("t5_r1"), r2 = num("t5_r2"), c = num("t5_c");
    $("t5_r2").closest(".fld").style.opacity = m === "m" ? 0.35 : 1;
    if (m === "a") { const th = 0.693 * (r1 + r2) * c, tl = 0.693 * r2 * c; set("t5_f", fmtEng(1 / (th + tl), "Hz")); set("t5_d", ((th / (th + tl)) * 100).toFixed(1) + " %"); set("t5_h", fmtEng(th, "s")); set("t5_l", fmtEng(tl, "s")); }
    else { const t = 1.1 * r1 * c; set("t5_f", "pulse " + fmtEng(t, "s")); set("t5_d", "—"); set("t5_h", fmtEng(t, "s")); set("t5_l", "—"); }
  }
  function filt() { const r = num("f_r"), c = num("f_c"), l = num("f_l"); set("f_rc", fmtEng(1 / (2 * Math.PI * r * c), "Hz")); set("f_tau", fmtEng(r * c, "s")); set("f_lc", fmtEng(1 / (2 * Math.PI * Math.sqrt(l * c)), "Hz")); set("f_rl", fmtEng(r / (2 * Math.PI * l), "Hz")); }
  function batt() {
    const cap = num("b_cap"), on = num("b_on"), duty = Math.min(100, Math.max(0, num("b_duty"))) / 100, sl = num("b_sl") / 1000, der = num("b_der") / 100;
    const avg = on * duty + sl * (1 - duty); const h = cap * der / avg;
    set("b_avg", fmtEng(avg / 1000, "A")); set("b_h", h > 48 ? `${(h / 24).toFixed(1)} days (${h.toFixed(0)} h)` : h > 1 ? `${h.toFixed(1)} hours` : `${(h * 60).toFixed(0)} minutes`);
  }
  function wire() {
    const I = num("w_i"), len = num("w_len"), V = num("w_v");
    let rec = null;
    for (const [g, d, a, r, ch] of AWG) if (ch * 0.6 >= I) rec = [g, a, I * r * len * 2];
    const rows = AWG.map(([g, d, a, r, ch, pt]) => { const drop = I * r * len * 2; const ok = ch * 0.6 >= I; return `<tr class="${ok ? "" : "bad"}"><td>${g} AWG</td><td>${a} mm²</td><td>${d} mm</td><td>${ch} A</td><td>${drop.toFixed(2)} V</td><td>${V > 0 ? ((drop / V) * 100).toFixed(1) + "%" : "—"}</td></tr>`; }).reverse();
    $("w_tbl").innerHTML = `<table><tr><th>Gauge</th><th>Area</th><th>Diameter</th><th>Max (chassis)</th><th>Drop (there & back)</th><th>Loss</th></tr>${rows.join("")}</table>`;
    if (rec) { set("w_rec", `${rec[0]} AWG (${rec[1]} mm²) or thicker`); set("w_drop", `${rec[2].toFixed(2)} V (${((rec[2] / V) * 100).toFixed(1)}%)`); } else { set("w_rec", "thicker than 10 AWG!"); set("w_drop", "—"); }
  }
  let twLast = 0;
  function trace() {
    const I = num("tw_i"), dT = num("tw_t"), oz = +$("tw_oz").value, k = $("tw_l").value === "e" ? 0.048 : 0.024, len = num("tw_len") / 1000;
    const area = Math.pow(I / (k * Math.pow(dT, 0.44)), 1 / 0.725); const th = 1.378 * oz; const wmil = area / th; const wmm = wmil * 0.0254;
    twLast = wmm;
    const R = 1.72e-8 * len / ((wmm / 1000) * (th * 25.4e-6));
    set("tw_w", `${wmm.toFixed(2)} mm`); set("tw_mil", `${wmil.toFixed(1)} mil`); set("tw_r", fmtEng(R, "Ω")); set("tw_d", fmtEng(R * I, "V"));
  }
  function es() { const v = num("es_v"), s = $("es_s").value; set("es_n", fmtEng(nearestE(v, s), "")); set("es_d", fmtEng(nearestE(v, s, "down"), "")); set("es_u", fmtEng(nearestE(v, s, "up"), "")); }
  function sp() { const vs = $("sp_v").value.split(/[,;]+/).map(parseEng).filter(v => v > 0); set("sp_s", fmtEng(vs.reduce((a, b) => a + b, 0), "Ω")); set("sp_p", fmtEng(1 / vs.reduce((a, b) => a + 1 / b, 0), "Ω")); }
  function cc() {
    const code = $("cc_code").value.trim(); const m = code.match(/^(\d)(\d)(\d)$/);
    set("cc_out", m ? fmtEng((+m[1] * 10 + +m[2]) * Math.pow(10, +m[3]) * 1e-12, "F") : /^\d{1,2}$/.test(code) ? fmtEng(+code * 1e-12, "F") : "—");
    const v = num("cc_val") / 1e-12; if (v >= 10) { let e = Math.floor(Math.log10(v)) - 1; const sig = Math.round(v / Math.pow(10, e)); set("cc_code_out", e >= 0 && e <= 9 ? `${sig}${e}` : "—"); } else set("cc_code_out", v > 0 ? String(Math.round(v)) : "—");
  }
  const all = () => { ohm(); led(); vdiv(); t555(); filt(); batt(); wire(); trace(); es(); sp(); cc(); };
  root.querySelectorAll("input,select").forEach(el => el.addEventListener("input", () => { if (el.id === "rc_val") return rcFromValue(); if (el.id === "rc_n") { rcRender(); return rcFromValue(); } if (el.id === "led_vf") $("led_c").value = "custom"; if (el.id === "cq") return; all(); }));
  $("cq").addEventListener("input", () => { const q = $("cq").value.toLowerCase().trim(); root.querySelectorAll(".card").forEach(c => c.style.display = !q || (c.dataset.k + " " + c.textContent).toLowerCase().includes(q) ? "" : "none"); });
  $("tw_sig").onclick = () => { if (typeof applyTraceWidth === "function") applyTraceWidth("track", twLast); };
  $("tw_pow").onclick = () => { if (typeof applyTraceWidth === "function") applyTraceWidth("power", twLast); };
  rcRender(); rcFromValue(); all();
}
