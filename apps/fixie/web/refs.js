// Fixie's quick reference: calculators and tables that help at the bench —
// resistor colour codes, SMD and capacitor markings, common pinouts, board
// labels, finding pin 1, rail voltages, and Ohm's law. Shown on the Parts tests
// tab and from a step's reference chips. Created by nrmluvdev.
window.REFS = {};
(() => {
  const R = window.REFS;
  const esc = s => String(s).replace(/[&<>"]/g, c => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" }[c]));
  const fmt = (v, unit) => {
    if (!isFinite(v)) return "?";
    const pre = [[1e9, "G"], [1e6, "M"], [1e3, "k"], [1, ""], [1e-3, "m"], [1e-6, "µ"], [1e-9, "n"], [1e-12, "p"]];
    for (const [m, p] of pre) if (Math.abs(v) >= m * 0.9999) return `${+(v / m).toPrecision(3)}${p}${unit}`;
    return `${v}${unit}`;
  };
  const add = (id, t, html, wire, more) => { R[id] = { t, html, wire: wire || (() => {}), ...more }; };

  // ---- resistor colour code
  const COL = [["black", "#111"], ["brown", "#7c3f12"], ["red", "#dc2626"], ["orange", "#ea7a14"], ["yellow", "#eab308"], ["green", "#16a34a"], ["blue", "#2563eb"], ["violet", "#8b5cf6"], ["grey", "#8b8b8b"], ["white", "#f3f4f6"]];
  const MULT = [...COL.map(([n, c], i) => [n, c, Math.pow(10, i)]), ["gold", "#c9a14a", 0.1], ["silver", "#c0c4cc", 0.01]];
  const TOL = [["gold", "±5%"], ["silver", "±10%"], ["brown", "±1%"], ["red", "±2%"], ["green", "±0.5%"], ["blue", "±0.25%"], ["violet", "±0.1%"], ["none", "±20%"]];
  const sel = (cls, opts, val) => `<select class="${cls}">${opts.map(([n, c], i) => `<option value="${i}"${i === val ? " selected" : ""}>${n}</option>`).join("")}</select>`;
  add("colour", "Resistor colour code", () => `
    <p>Pick the bands, reading from the end they're bunched at (gold or silver goes last).</p>
    <div class="calc"><select class="nb"><option value="4">4 bands</option><option value="5">5 bands</option></select>
      ${sel("b1", COL, 4)}${sel("b2", COL, 7)}<span class="b3w">${sel("b3", COL, 0)}</span>${sel("bm", MULT, 2)}${sel("bt", TOL, 0)}</div>
    <div class="calc"><span class="sw"></span><span class="out"></span></div>`,
    root => {
      const q = s => root.querySelector(s);
      const go = () => {
        const five = q(".nb").value === "5"; q(".b3w").hidden = !five;
        const d = [+q(".b1").value, +q(".b2").value].concat(five ? [+q(".b3").value] : []);
        const v = +d.join("") * MULT[+q(".bm").value][2];
        q(".out").textContent = `${fmt(v, "Ω")} ${TOL[+q(".bt").value][1]}`;
        q(".sw").innerHTML = [...d.map(i => COL[i][1]), MULT[+q(".bm").value][1], TOL[+q(".bt").value][0] === "none" ? "transparent" : (COL.find(c => c[0] === TOL[+q(".bt").value][0]) || MULT.find(c => c[0] === TOL[+q(".bt").value][0]))[1]].map(c => `<span class="swatch" style="background:${c}"></span>`).join("");
      };
      root.querySelectorAll("select").forEach(s => s.onchange = go); go();
    });

  // ---- SMD resistor codes, including EIA-96
  const E96 = [100, 102, 105, 107, 110, 113, 115, 118, 121, 124, 127, 130, 133, 137, 140, 143, 147, 150, 154, 158, 162, 165, 169, 174, 178, 182, 187, 191, 196, 200, 205, 210, 215, 221, 226, 232, 237, 243, 249, 255, 261, 267, 274, 280, 287, 294, 301, 309, 316, 324, 332, 340, 348, 357, 365, 374, 383, 392, 402, 412, 422, 432, 442, 453, 464, 475, 487, 499, 511, 523, 536, 549, 562, 576, 590, 604, 619, 634, 649, 665, 681, 698, 715, 732, 750, 768, 787, 806, 825, 845, 866, 887, 909, 931, 953, 976];
  const E96M = { Z: 0.001, Y: 0.01, R: 0.01, X: 0.1, S: 0.1, A: 1, B: 10, H: 10, C: 100, D: 1000, E: 10000, F: 100000 };
  function smd(code) {
    const c = code.trim().toUpperCase();
    if (/^\d*R\d*$/.test(c) && c !== "R") return +c.replace("R", ".");                 // 4R7 = 4.7Ω, R10 = 0.1Ω
    if (/^\d{3}$/.test(c)) return +c.slice(0, 2) * Math.pow(10, +c[2]);                  // 472 = 4.7k
    if (/^\d{4}$/.test(c)) return +c.slice(0, 3) * Math.pow(10, +c[3]);                  // 4702 = 47k
    const m = c.match(/^(\d{2})([ZYRXSABHCDEF])$/);                                      // EIA-96: 01C = 10k
    if (m && +m[1] >= 1 && +m[1] <= 96) return E96[+m[1] - 1] * E96M[m[2]];
    if (/^0+$/.test(c)) return 0;
    return NaN;
  }
  R.smdValue = smd;
  add("smd", "SMD resistor markings", () => `
    <p>Type the code printed on the resistor: 3 digits (472), 4 digits (4702), an R for the decimal point (4R7), or 2 digits and a letter (01C, EIA-96).</p>
    <div class="calc"><input class="code" placeholder="e.g. 103" maxlength="5"><span class="out"></span></div>
    <p class="dim" style="margin-top:6px">Last digit = how many zeros. 000 or 0 is a zero-ohm link.</p>`,
    root => { const i = root.querySelector(".code"), o = root.querySelector(".out"); i.oninput = () => { const v = smd(i.value); o.textContent = i.value.trim() ? (isNaN(v) ? "not a code I know" : fmt(v, "Ω")) : ""; }; });

  // ---- capacitor codes
  function capCode(code) {
    const c = code.trim().replace(/µ/g, "u").toLowerCase();
    let m = c.match(/^(\d)(\d)(\d)$/); if (m) return (+(m[1] + m[2])) * Math.pow(10, +m[3]) * 1e-12;   // 104 = 100nF
    m = c.match(/^(\d*)([pnu])(\d*)$/); if (m && (m[1] || m[3])) return +((m[1] || "0") + "." + (m[3] || "0")) * { p: 1e-12, n: 1e-9, u: 1e-6 }[m[2]]; // 4n7, 2u2
    m = c.match(/^(\d+(?:\.\d+)?)$/); if (m) return +m[1] * (+m[1] < 1 ? 1e-6 : 1e-12);                // 0.1 = 0.1µF, 47 = 47pF
    return NaN;
  }
  R.capValue = capCode;
  add("cap", "Capacitor markings", () => `
    <p>Ceramic and film capacitors: 3 digits in picofarads (104), or a letter for the decimal point (4n7, 2u2). A letter after the code is the tolerance (K = ±10%, J = ±5%, M = ±20%).</p>
    <div class="calc"><input class="code" placeholder="e.g. 104" maxlength="6"><span class="out"></span></div>
    <table class="reftab"><tr><th>Code</th><th>Value</th></tr><tr><td><code>101</code></td><td>100pF</td></tr><tr><td><code>222</code></td><td>2.2nF</td></tr><tr><td><code>103</code></td><td>10nF</td></tr><tr><td><code>104</code></td><td>100nF (0.1µF)</td></tr><tr><td><code>105</code></td><td>1µF</td></tr></table>`,
    root => { const i = root.querySelector(".code"), o = root.querySelector(".out"); i.oninput = () => { const v = capCode(i.value.replace(/[jkm]$/i, "")); o.textContent = i.value.trim() ? (isNaN(v) ? "not a code I know" : fmt(v, "F")) : ""; }; });

  // ---- pinouts
  add("pinouts", "Common pinouts", () => `
    <p>Pins numbered from the left, label facing you, leads down (TO-92: flat side facing you). Always check the datasheet: lookalike parts differ.</p>
    <table class="reftab"><tr><th>Part</th><th>Pins 1 · 2 · 3</th></tr>
      <tr><td>78xx regulators (7805, 7812)</td><td>IN · GND · OUT</td></tr>
      <tr><td>79xx negative regulators</td><td>GND · IN · OUT</td></tr>
      <tr><td>LM317</td><td>ADJ · OUT · IN</td></tr>
      <tr><td>AMS1117, LM1117 (SOT-223)</td><td>GND/ADJ · OUT · IN (tab = OUT)</td></tr>
      <tr><td>TL431 (TO-92)</td><td>REF · ANODE · CATHODE</td></tr>
      <tr><td>2N3904, 2N2222A, 2N3906</td><td>E · B · C</td></tr>
      <tr><td>BC547, BC557, BC337</td><td>C · B · E</td></tr>
      <tr><td>Power MOSFETs (IRF520, IRFZ44N, IRLZ44N)</td><td>G · D · S (tab = D)</td></tr>
      <tr><td>Optocoupler PC817 (DIP-4)</td><td>1 anode · 2 cathode · 3 emitter · 4 collector</td></tr>
      <tr><td>555 timer (DIP-8)</td><td>1 GND · 2 TRIG · 3 OUT · 4 RESET · 5 CTRL · 6 THR · 7 DIS · 8 VCC</td></tr>
      <tr><td>Dual op-amp (TL072, LM358, NE5532)</td><td>1 OUT A · 2 IN− A · 3 IN+ A · 4 V− · 5 IN+ B · 6 IN− B · 7 OUT B · 8 V+</td></tr>
      <tr><td>14-pin logic (74xx)</td><td>7 GND · 14 VCC</td></tr>
      <tr><td>16-pin logic</td><td>8 GND · 16 VCC</td></tr>
      <tr><td>USB-A (contacts up, looking in)</td><td>1 5V · 2 D− · 3 D+ · 4 GND</td></tr>
      <tr><td>VGA</td><td>1 R · 2 G · 3 B · 13 HSYNC · 14 VSYNC · 5–8, 10 GND</td></tr>
    </table>`, null, { pics: ["reg", "bjt", "mosfet", "opamp"] });

  // ---- board labels
  add("labels", "Reading a board: what the labels mean", () => `
    <table class="reftab"><tr><th>Label</th><th>Part</th></tr>
      ${[["R", "Resistor"], ["C", "Capacitor"], ["L", "Inductor (coil)"], ["FB", "Ferrite bead"], ["D", "Diode"], ["ZD, DZ", "Zener diode"], ["LED", "LED"], ["Q, T", "Transistor or MOSFET"], ["U, IC", "Chip"], ["Y, X, XTAL", "Crystal or resonator"], ["F, FU", "Fuse"], ["PTC, RT", "Polyfuse or thermistor"], ["VR, RV, P", "Potentiometer or trimmer"], ["K, RL, RLY", "Relay"], ["T, TR", "Transformer"], ["SW, S", "Switch"], ["J, P, CN, CON", "Connector"], ["TP", "Test point (handy to measure on)"], ["BT, BAT", "Battery"], ["JP", "Jumper"], ["MOV, VDR, Z", "Varistor (surge protection)"], ["PC, OC, U", "Optocoupler"]].map(([a, b]) => `<tr><td><code>${a}</code></td><td>${b}</td></tr>`).join("")}
    </table><p class="dim" style="margin-top:6px">Labels with a number (R12, C3) match the part to the schematic or service manual. The silkscreen outline often shows a diode's band, a capacitor's −, or pin 1.</p>`);

  // ---- pin 1
  add("pin1", "Finding pin 1", () => `<p>Pin 1 is marked by a notch, dot, cut corner or a square pad underneath. Pins count anticlockwise looking down on the chip.</p>`, null, { pics: ["ic", "mcu"] });

  // ---- rails
  add("rails", "Common rail voltages", () => `
    <table class="reftab"><tr><th>Rail</th><th>Usually feeds</th></tr>
      <tr><td><code>12V, 19V, 24V</code></td><td>Input from an adapter; motors, backlights, amplifiers</td></tr>
      <tr><td><code>5V</code></td><td>USB, older logic, many modules and LCDs</td></tr>
      <tr><td><code>3.3V</code></td><td>Most modern chips: microcontrollers, flash, sensors, Wi-Fi</td></tr>
      <tr><td><code>2.5V, 1.8V</code></td><td>Memory, I/O on newer chips</td></tr>
      <tr><td><code>1.2V, 1.0V, 0.9V</code></td><td>Processor and FPGA cores</td></tr>
      <tr><td><code>±12V, ±15V</code></td><td>Op-amps in audio gear and old equipment</td></tr>
    </table><p class="dim" style="margin-top:6px">Most rails should be within about 5% of their value. Labels near regulators and test points usually say which rail is which.</p>`);

  // ---- Ohm's law and friends
  add("ohm", "Ohm's law and quick sums", () => `
    <p>Type any two of voltage, current and resistance.</p>
    <div class="calc"><input class="v" placeholder="volts"><input class="i" placeholder="amps (0.02 = 20mA)"><input class="r" placeholder="ohms"><span class="out"></span></div>
    <p style="margin-top:10px"><b>LED resistor</b>: supply, LED forward voltage, current.</p>
    <div class="calc"><input class="ls" placeholder="supply V"><input class="lf" placeholder="LED V (2 red, 3 blue)"><input class="li" placeholder="mA (e.g. 10)"><span class="lout out"></span></div>
    <p style="margin-top:10px"><b>Linear regulator heat</b>: (in − out) × current.</p>
    <div class="calc"><input class="ri" placeholder="in V"><input class="ro" placeholder="out V"><input class="ra" placeholder="mA"><span class="rout out"></span></div>`,
    root => {
      const q = s => root.querySelector(s), n = s => parseFloat(q(s).value);
      const ohm = () => {
        const V = n(".v"), I = n(".i"), Rr = n(".r"); let s = "";
        if (isFinite(V) && isFinite(I) && !isFinite(Rr)) s = `R = ${fmt(V / I, "Ω")} · P = ${fmt(V * I, "W")}`;
        else if (isFinite(V) && isFinite(Rr) && !isFinite(I)) s = `I = ${fmt(V / Rr, "A")} · P = ${fmt(V * V / Rr, "W")}`;
        else if (isFinite(I) && isFinite(Rr) && !isFinite(V)) s = `V = ${fmt(I * Rr, "V")} · P = ${fmt(I * I * Rr, "W")}`;
        q(".out").textContent = s;
      };
      const led = () => { const S = n(".ls"), F = n(".lf"), I = n(".li") / 1000; q(".lout").textContent = isFinite(S) && isFinite(F) && I > 0 && S > F ? `${fmt((S - F) / I, "Ω")} (${fmt((S - F) * I, "W")})` : ""; };
      const reg = () => { const a = n(".ri"), b = n(".ro"), I = n(".ra") / 1000; if (!(isFinite(a) && isFinite(b) && I > 0)) return q(".rout").textContent = ""; const P = (a - b) * I; q(".rout").textContent = `${fmt(P, "W")}${P > 1 ? ": needs a heatsink, or a buck module" : P > 0.5 ? ": warm; fine on TO-220, too hot for small packages" : ""}`; };
      [".v", ".i", ".r"].forEach(s => q(s).oninput = ohm); [".ls", ".lf", ".li"].forEach(s => q(s).oninput = led); [".ri", ".ro", ".ra"].forEach(s => q(s).oninput = reg);
    });
})();
