// Fixie's pictures for Board level, My build and Parts tests: what each part
// looks like (with its pins), how to connect the meter, scope or logic
// analyser, and what a good and a bad signal look like. All drawn here as SVG,
// coloured by app.css classes so they follow the theme. Created by nrmluvdev.
//
// PICS["id"] = { t: caption title, svg: "<svg…>", say: one line on what to notice }
window.PICS = {};
(() => {
  const P = window.PICS;
  const W = 340, H = 190;
  const svg = (inner, h = H, w = W) => `<svg viewBox="0 0 ${w} ${h}" role="img">${inner}</svg>`;
  const t = (x, y, s, cls = "pk-txt", a = "middle") => `<text class="${cls}" x="${x}" y="${y}" text-anchor="${a}">${s}</text>`;
  const tl = (x, y, lines, cls = "pk-small", a = "middle", gap = 13) => lines.map((l, i) => t(x, y + i * gap, l, cls, a)).join("");
  const line = (x1, y1, x2, y2, cls = "pk-wire") => `<line class="${cls}" x1="${x1}" y1="${y1}" x2="${x2}" y2="${y2}"/>`;
  const path = (d, cls = "pk-wire") => `<path class="${cls}" d="${d}"/>`;
  const rect = (x, y, w, h, cls, rx = 3) => `<rect class="${cls}" x="${x}" y="${y}" width="${w}" height="${h}" rx="${rx}"/>`;
  const circ = (x, y, r, cls) => `<circle class="${cls}" cx="${x}" cy="${y}" r="${r}"/>`;
  const gnd = (x, y) => path(`M${x} ${y}v6M${x - 8} ${y + 6}h16M${x - 5} ${y + 10}h10M${x - 2} ${y + 14}h4`);
  // a meter probe: a lead from (x1,y1) ending in a tip at (x2,y2); red or black
  const probe = (x1, y1, x2, y2, red) => `${path(`M${x1} ${y1}Q${(x1 + x2) / 2} ${Math.min(y1, y2) - 30} ${x2} ${y2}`, red ? "pk-red" : "pk-blk")}${circ(x2, y2, 3.5, red ? "pk-redf" : "pk-blkf")}`;
  // a small multimeter with a display and a mode
  const meter = (x, y, disp, mode) => `${rect(x, y, 78, 96, "pk-meter", 8)}${rect(x + 8, y + 8, 62, 26, "pk-screen", 3)}${t(x + 39, y + 27, disp, "pk-lcd")}
    ${circ(x + 39, y + 58, 13, "pk-body2")}${t(x + 39, y + 62, mode, "pk-small")}${circ(x + 22, y + 86, 4, "pk-blkf")}${circ(x + 56, y + 86, 4, "pk-redf")}`;
  // a DIP chip: n pins, labels {pin: text}, name
  function dip(x, y, n, labels = {}, name = "", hl = []) {
    const half = n / 2, w = 84, h = half * 16 + 10;
    let s = rect(x, y, w, h, "pk-body", 3) + path(`M${x + w / 2 - 7} ${y}a7 7 0 0 0 14 0`, "pk-notch") + circ(x + 6, y + 6, 2.2, "pk-dot");
    for (let i = 0; i < half; i++) {
      const py = y + 14 + i * 16, l = i + 1, r = n - i;
      s += rect(x - 9, py - 3, 9, 6, hl.includes(l) ? "pk-hlpin" : "pk-metal", 1) + rect(x + w, py - 3, 9, 6, hl.includes(r) ? "pk-hlpin" : "pk-metal", 1);
      s += t(x + 8, py + 3.5, l, "pk-tiny", "start") + t(x + w - 8, py + 3.5, r, "pk-tiny", "end");
      if (labels[l]) s += t(x - 13, py + 3.5, labels[l], "pk-small", "end");
      if (labels[r]) s += t(x + w + 13, py + 3.5, labels[r], "pk-small", "start");
    }
    return s + (name ? t(x + w / 2, y + h / 2 + 4, name, "pk-chipsm") : "");
  }
  // TO-220 (front, leads down) and TO-92 (flat side facing you)
  const to220 = (x, y, pins, name) => `${rect(x - 4, y - 22, 64, 26, "pk-metal", 3)}${circ(x + 28, y - 10, 5, "pk-screen")}${rect(x, y, 56, 40, "pk-body", 2)}${t(x + 28, y + 24, name, "pk-chipsm")}
    ${pins.map((p, i) => `${rect(x + 8 + i * 18, y + 40, 4, 30, "pk-metal", 1)}${t(x + 10 + i * 18, y + 82, i + 1, "pk-tiny")}${t(x + 10 + i * 18, y + 96 + (i % 2) * 12, p, "pk-small")}`).join("")}`;
  const to92 = (x, y, pins, name) => `${path(`M${x} ${y}h36v26a18 18 0 0 1-36 0z`, "pk-body")}${t(x + 18, y + 18, name, "pk-chipsm")}
    ${pins.map((p, i) => `${rect(x + 5 + i * 12, y + 40, 3, 28, "pk-metal", 1)}${t(x + 6.5 + i * 12, y + 82, p, "pk-small")}`).join("")}${t(x + 18, y - 6, "flat side facing you", "pk-tiny")}`;
  // an oscilloscope screen with a trace (points 0..1 across, -1..1 up)
  function scope(x, y, w, h, fn, opts = {}) {
    let g = rect(x, y, w, h, "pk-screen", 4);
    for (let i = 1; i < 10; i++) g += line(x + w * i / 10, y, x + w * i / 10, y + h, "pk-grid");
    for (let i = 1; i < 8; i++) g += line(x, y + h * i / 8, x + w, y + h * i / 8, "pk-grid");
    const traces = Array.isArray(fn) ? fn : [fn];
    traces.forEach((f, k) => {
      let d = "";
      for (let i = 0; i <= 400; i++) { const u = i / 400, v = Math.max(-1.05, Math.min(1.05, f(u))); d += (i ? "L" : "M") + (x + u * w).toFixed(1) + " " + (y + h / 2 - v * h * 0.4).toFixed(1); }
      g += path(d, k ? "pk-trace2" : "pk-trace");
    });
    if (opts.label) g += t(x + 6, y + 13, opts.label, "pk-scopetxt", "start");
    if (opts.right) g += t(x + w - 6, y + 13, opts.right, "pk-scopetxt", "end");
    return g;
  }
  const sq = (u, cycles, duty = 0.5) => ((u * cycles) % 1) < duty ? 0.85 : -0.85;
  // a digital trace drawn from a string of 0/1 levels across the width
  const bits = levels => u => levels[Math.min(levels.length - 1, Math.floor(u * levels.length))] === "1" ? 0.7 : -0.7;
  const good = (x, y) => t(x, y, "GOOD", "pk-goodt", "start"), bad = (x, y) => t(x, y, "FAULTY", "pk-badt", "start");
  const add = (id, title, inner, say, h) => { P[id] = { t: title, svg: svg(inner, h), say }; };

  // ================================================================ parts: what they look like
  add("res", "Resistor", `
    ${line(20, 60, 60, 60, "pk-lead")}${rect(60, 46, 90, 28, "pk-resbody", 12)}${rect(72, 46, 8, 28, "c-yellow", 0)}${rect(86, 46, 8, 28, "c-violet", 0)}${rect(100, 46, 8, 28, "c-red", 0)}${rect(130, 46, 8, 28, "c-gold", 0)}${line(150, 60, 190, 60, "pk-lead")}
    ${tl(105, 98, ["yellow-violet-red-gold", "= 4.7kΩ ±5%"])}
    ${rect(220, 44, 50, 30, "pk-body", 2)}${rect(214, 44, 8, 30, "pk-metal", 1)}${rect(268, 44, 8, 30, "pk-metal", 1)}${t(245, 64, "472", "pk-chip")}${tl(245, 98, ["SMD: 472 = 47 × 10²", "= 4.7kΩ"])}
    ${tl(170, 146, ["Through-hole: read the bands from the end they're bunched at.", "Surface-mount: 3 or 4 digits, or 2 digits + a letter (EIA-96)."])}`,
    "Use the colour-code calculator in Reference to read the bands.");
  add("ecap", "Electrolytic capacitor", `
    ${rect(60, 30, 60, 100, "pk-capbody", 8)}${rect(60, 30, 16, 100, "pk-capstripe", 0)}${t(68, 70, "−", "pk-big")}${t(68, 100, "−", "pk-big")}
    ${path("M66 32l24 0M78 32l-6 10", "pk-score")}${line(76, 130, 76, 165, "pk-lead")}${line(104, 130, 104, 172, "pk-lead")}${t(72, 182, "−", "pk-txt")}${t(100, 182, "+ (longer leg)", "pk-small", "start")}
    ${t(95, 20, "Top scored to vent", "pk-small")}
    ${rect(200, 50, 60, 60, "pk-body", 4)}${circ(230, 80, 22, "pk-capbody")}${path("M212 62h12l-12 12z", "pk-capstripe")}${t(230, 128, "SMD can: dark segment = −", "pk-small")}
    ${t(230, 160, "Stripe and short leg are negative.", "pk-small")}`,
    "Fit them the same way round as the one you remove: a reversed electrolytic can burst.");
  add("ccap", "Ceramic and film capacitors", `
    ${circ(60, 60, 26, "pk-ceram")}${t(60, 64, "104", "pk-chip")}${line(52, 84, 52, 130, "pk-lead")}${line(68, 84, 68, 130, "pk-lead")}${t(60, 148, "104 = 100nF", "pk-small")}
    ${rect(130, 40, 56, 44, "pk-film", 4)}${t(158, 60, "X2", "pk-chip")}${t(158, 75, "0.1µF", "pk-small")}${line(146, 84, 146, 130, "pk-lead")}${line(170, 84, 170, 130, "pk-lead")}${t(158, 148, "Film (X2 = mains rated)", "pk-small")}
    ${rect(240, 56, 40, 22, "pk-mlcc", 2)}${rect(234, 56, 8, 22, "pk-metal", 1)}${rect(278, 56, 8, 22, "pk-metal", 1)}${t(260, 100, "SMD ceramic: no marking.", "pk-small")}${t(260, 114, "The usual shorted part", "pk-small")}${t(260, 128, "on a dead rail.", "pk-small")}`,
    "Ceramic codes are in picofarads: 104 = 10 × 10⁴ pF = 100nF. Use the decoder in Reference.");
  add("diode", "Diode", `
    ${line(20, 50, 80, 50, "pk-lead")}${rect(80, 38, 70, 24, "pk-body", 8)}${rect(134, 38, 10, 24, "pk-band", 0)}${line(150, 50, 210, 50, "pk-lead")}${t(50, 40, "anode +", "pk-small")}${t(180, 40, "cathode −", "pk-small")}
    ${path("M120 110l0 40 34-20z", "pk-symf")}${line(154, 110, 154, 150, "pk-sym")}${line(90, 130, 120, 130, "pk-sym")}${line(154, 130, 190, 130, "pk-sym")}${t(105, 172, "A", "pk-small")}${t(165, 172, "K", "pk-small")}
    ${rect(250, 110, 46, 26, "pk-body", 2)}${rect(254, 110, 6, 26, "pk-band", 0)}${t(273, 160, "SMD: line = cathode", "pk-small")}`,
    "The band (or the bar in the symbol) is the cathode: current flows towards it.");
  add("led", "LED", `
    ${path("M80 40a26 26 0 0 1 52 0v44h-52z", "pk-led")}${line(80, 70, 80, 84, "pk-flat")}${line(94, 84, 94, 160, "pk-lead")}${line(118, 84, 118, 172, "pk-lead")}
    ${t(94, 178, "−", "pk-txt")}${t(126, 186, "+ (long)", "pk-small")}${t(60, 76, "flat = −", "pk-small", "end")}
    ${t(250, 70, "Diode test, red on +:", "pk-small")}${t(250, 86, "it glows faintly, or reads", "pk-small")}${t(250, 102, "about 1.6V (red) to 3.2V", "pk-small")}${t(250, 118, "(blue, white).", "pk-small")}`,
    "Long leg and round side are +; short leg and flat side are −.");
  add("bridge", "Bridge rectifier", `
    ${rect(40, 40, 70, 70, "pk-body", 6)}${t(75, 66, "~   +", "pk-chip")}${t(75, 94, "~   −", "pk-chip")}${[52, 68, 84, 100].map((x, i) => rect(x - 2, 110, 4, 30, "pk-metal", 1) + t(x, 154, ["~", "~", "+", "−"][i], "pk-small")).join("")}
    ${(() => { const cx = 245, cy = 90, r = 50; const d = (x1, y1, x2, y2) => line(x1, y1, x2, y2, "pk-sym"); return d(cx - r, cy, cx, cy - r) + d(cx, cy - r, cx + r, cy) + d(cx + r, cy, cx, cy + r) + d(cx, cy + r, cx - r, cy) + t(cx - r - 10, cy + 4, "~", "pk-txt") + t(cx + r + 10, cy + 4, "~", "pk-txt") + t(cx, cy - r - 8, "+", "pk-txt") + t(cx, cy + r + 16, "−", "pk-txt"); })()}
    ${t(245, 180, "Four diodes: test each one", "pk-small")}`,
    "Diode-test ~ to + and − to ~ (four tests), then reverse them: every one should block.");
  add("bjt", "Transistor (BJT)", `
    ${to92(40, 40, ["E", "B", "C"], "2N3904")}${t(58, 140, "2N3904, 2N2222A", "pk-small")}
    ${to92(150, 40, ["C", "B", "E"], "BC547")}${t(168, 140, "BC547, BC337", "pk-small")}
    ${path("M270 60v60M270 78l30-18M270 102l30 18M300 120l-8-2 4-7", "pk-sym")}${line(250, 90, 270, 90, "pk-sym")}${t(244, 94, "B", "pk-small", "end")}${t(306, 56, "C", "pk-small", "start")}${t(306, 128, "E", "pk-small", "start")}${t(280, 160, "NPN", "pk-small")}`,
    "Pinouts differ between part numbers: always check the datasheet (see Reference: pinouts).");
  add("mosfet", "MOSFET", `
    ${to220(40, 50, ["G", "D", "S"], "IRFZ44N")}${t(60, 170, "Tab is the drain", "pk-small")}
    ${path("M220 60v60M232 60v16M232 82v16M232 104v16M232 68h30v-14M232 112h30v14M232 90h16l-6-4M262 112v-44", "pk-sym")}${line(200, 90, 220, 90, "pk-sym")}
    ${t(196, 94, "G", "pk-small", "end")}${t(268, 52, "D", "pk-small", "start")}${t(268, 134, "S", "pk-small", "start")}${path("M262 100l-6 10h12z", "pk-symf")}${t(282, 104, "body diode", "pk-tiny", "start")}${t(240, 160, "N-channel", "pk-small")}`,
    "Most TO-220 power MOSFETs are G-D-S from the left with the label facing you.");
  add("reg", "Linear regulators: pinouts differ", `
    ${to220(20, 40, ["IN", "GND", "OUT"], "7805")}${to220(126, 40, ["ADJ", "OUT", "IN"], "LM317")}
    ${rect(240, 56, 60, 30, "pk-body", 2)}${rect(255, 38, 30, 18, "pk-metal", 1)}${t(270, 52, "OUT", "pk-tiny")}${t(270, 76, "1117", "pk-chip")}${[0, 1, 2].map(i => rect(247 + i * 20, 86, 6, 14, "pk-metal", 1) + t(250 + i * 20, 114 + (i % 2) * 12, ["GND", "OUT", "IN"][i], "pk-small")).join("")}${t(270, 150, "AMS1117 (SOT-223)", "pk-small")}`,
    "Same package, different pins: a 7805 is IN-GND-OUT, an LM317 ADJ-OUT-IN, an AMS1117 GND-OUT-IN.", 200);
  add("buck", "Switching regulator (buck)", `
    ${rect(30, 60, 60, 50, "pk-body", 3)}${t(60, 90, "chip", "pk-chip")}${rect(130, 66, 44, 38, "pk-inductor", 6)}${t(152, 90, "4R7", "pk-chipsm")}${rect(210, 70, 44, 30, "pk-capbody", 4)}${t(232, 90, "C out", "pk-tiny")}
    ${line(90, 85, 130, 85)}${line(174, 85, 210, 85)}${line(254, 85, 300, 85)}${circ(110, 85, 5, "pk-hlf")}${t(110, 70, "SW", "pk-small")}${t(300, 76, "VOUT", "pk-small")}${t(20, 76, "VIN", "pk-small", "end")}${line(0, 85, 30, 85)}
    ${t(60, 128, "EN: high = on", "pk-small")}${t(152, 128, "inductor (coil)", "pk-small")}
    ${t(170, 148, "SW node: a square wave to about VIN, 100kHz–2MHz", "pk-small")}${scope(60, 154, 220, 38, u => sq(u, 6, 0.4))}`,
    "Probe the switching node (SW, between the chip and the coil): a square wave means it's working.", 198);
  add("crystal", "Crystal", `
    ${rect(40, 50, 90, 40, "pk-metal", 18)}${t(85, 75, "16.000", "pk-chipdark")}${line(66, 90, 66, 130, "pk-lead")}${line(104, 90, 104, 130, "pk-lead")}
    ${rect(160, 60, 36, 22, "pk-body", 2)}${t(178, 76, "16M", "pk-tiny")}${t(178, 100, "SMD crystal", "pk-small")}
    ${tl(276, 56, ["Two small capacitors", "(10–33pF) go from", "each leg to ground.", "", "Probe the output side", "(XOUT), ×10 probe."], "pk-small", "middle", 14)}`,
    "The marking is the frequency in MHz (16.000) or kHz (32.768).");
  add("ic", "Logic chip: finding pin 1", `
    ${dip(50, 30, 14, { 7: "GND", 14: "VCC" }, "74HC04", [7, 14])}
    ${t(260, 50, "Pin 1 is next to the", "pk-small")}${t(260, 64, "notch or dot; pins count", "pk-small")}${t(260, 78, "down one side and back", "pk-small")}${t(260, 92, "up the other.", "pk-small")}${t(260, 116, "14-pin logic: GND is 7,", "pk-small")}${t(260, 130, "VCC is 14.", "pk-small")}`,
    "Check the highlighted power pins first: a chip with no power can't do anything.");
  add("mcu", "Microcontroller: pin 1 on a square chip", `
    ${rect(70, 40, 110, 110, "pk-body", 4)}${circ(86, 56, 4, "pk-dot")}${path("M70 52l12-12", "pk-score")}
    ${[...Array(8)].map((_, i) => rect(56, 50 + i * 12, 14, 5, "pk-metal", 1) + rect(180, 50 + i * 12, 14, 5, "pk-metal", 1) + rect(80 + i * 12, 26, 5, 14, "pk-metal", 1) + rect(80 + i * 12, 150, 5, 14, "pk-metal", 1)).join("")}
    ${t(50, 56, "1", "pk-small", "end")}${t(125, 100, "MCU", "pk-chip")}${tl(234, 56, ["Pin 1: dot or", "cut corner; pins", "count anticlockwise.", "", "Check every VCC", "and VDD pin, then", "the clock and reset."], "pk-small", "start", 14)}`,
    "On square chips the pins count anticlockwise from the dot, looking from above.");
  add("opamp", "Dual op-amp (TL072, NE5532, LM358)", `
    ${dip(120, 40, 8, { 1: "OUT A", 2: "IN− A", 3: "IN+ A", 4: "V−", 5: "IN+ B", 6: "IN− B", 7: "OUT B", 8: "V+" }, "TL072", [4, 8])}`,
    "The common 8-pin dual op-amp pinout: power on 8 and 4, outputs on 1 and 7.", 130);
  add("timer555", "555 timer", `${dip(120, 40, 8, { 1: "GND", 2: "TRIG", 3: "OUT", 4: "RESET", 5: "CTRL", 6: "THR", 7: "DIS", 8: "VCC" }, "NE555", [1, 8])}`,
    "A classic in old gadgets: pin 3 should switch if it's running.", 130);
  add("opto", "Optocoupler (PC817)", `
    ${dip(120, 50, 4, { 1: "anode", 2: "cathode", 3: "emitter", 4: "collector" }, "817")}
    ${t(170, 150, "LED inside on 1–2; transistor on 3–4", "pk-small")}`,
    "In power supplies the opto carries the voltage feedback across the isolation gap.", 170);
  add("fuse", "Fuses", `
    ${rect(30, 50, 120, 30, "pk-glass", 10)}${rect(20, 48, 22, 34, "pk-metal", 3)}${rect(138, 48, 22, 34, "pk-metal", 3)}${line(42, 65, 138, 65)}${good(30, 40)}
    ${rect(190, 50, 120, 30, "pk-glass", 10)}${rect(180, 48, 22, 34, "pk-metal", 3)}${rect(298, 48, 22, 34, "pk-metal", 3)}${path("M202 65h40l4-6M298 65h-40l-4 7")}${`<ellipse class="pk-soot" cx="250" cy="65" rx="26" ry="11"/>`}${bad(190, 40)}
    ${rect(60, 120, 40, 20, "pk-body", 2)}${t(80, 134, "F1", "pk-tiny")}${t(80, 160, "SMD fuse (marked F)", "pk-small")}${rect(210, 116, 40, 26, "pk-ptc", 4)}${t(230, 160, "Polyfuse (resettable)", "pk-small")}`,
    "Glass fuses show a broken wire or soot; others need a continuity test.");
  add("relay", "Relay", `
    ${rect(40, 40, 110, 80, "pk-relay", 6)}${t(95, 76, "5V DC", "pk-chipdark")}${t(95, 92, "10A 250VAC", "pk-tinydark")}
    ${path("M200 50v70", "pk-sym")}${path("M190 60h20M190 70h20M190 80h20M190 90h20M190 100h20", "pk-sym")}${t(200, 135, "coil", "pk-small")}
    ${circ(270, 85, 4, "pk-symf")}${path("M270 85l40-22", "pk-sym")}${circ(312, 60, 4, "pk-symf")}${circ(312, 110, 4, "pk-symf")}
    ${t(262, 100, "COM", "pk-small", "end")}${t(312, 50, "NC", "pk-small")}${t(312, 128, "NO", "pk-small")}`,
    "Coil reads tens to hundreds of ohms; COM–NC beeps at rest, COM–NO when it's switched.");
  add("switch", "Tactile button", `
    ${rect(80, 50, 60, 60, "pk-body", 4)}${circ(110, 80, 16, "pk-body2")}${[[70, 60], [150, 60], [70, 100], [150, 100]].map(([x, y]) => rect(x - 8, y - 3, 16, 6, "pk-metal", 1)).join("")}
    ${path("M62 60v40M158 60v40", "pk-hl")}${t(46, 84, "joined", "pk-tiny", "end")}${t(174, 84, "joined", "pk-tiny", "start")}
    ${t(260, 64, "Legs on each side are", "pk-small")}${t(260, 78, "always joined. Test", "pk-small")}${t(260, 92, "across a corner: it", "pk-small")}${t(260, 106, "beeps only when pressed.", "pk-small")}`,
    "Four legs, two pairs: test diagonally.");
  add("liion", "Lithium-ion cell (18650)", `
    ${rect(40, 60, 200, 60, "pk-cell", 12)}${rect(240, 74, 10, 32, "pk-metal", 2)}${t(256, 94, "+", "pk-big", "start")}${t(28, 96, "−", "pk-big", "end")}${t(140, 96, "18650  3.7V", "pk-chipdark")}
    ${tl(170, 146, ["3.0–4.2V normal · below 2.5V over-discharged", "swollen, dented or leaking: recycle it"])}`,
    "Never solder to a cell: packs are spot-welded. Store and test away from anything flammable.");
  add("lcd1602", "Character LCD (16×2)", `
    ${rect(20, 30, 300, 90, "pk-pcbblue", 4)}${rect(50, 48, 240, 54, "pk-lcdglass", 2)}${[...Array(16)].map((_, i) => rect(58 + i * 14.5, 56, 11, 16, "pk-lcdcell", 1)).join("")}
    ${[...Array(16)].map((_, i) => circ(40 + i * 10, 22, 3, "pk-copperf")).join("")}${t(40, 14, "1", "pk-tiny")}${t(190, 14, "16", "pk-tiny")}
    ${tl(170, 138, ["1 GND · 2 VCC · 3 contrast · 4 RS · 5 RW · 6 E", "7–14 data · 15 backlight + · 16 backlight −", "Powered, contrast turned up: solid blocks on the top row."])}`,
    "Blocks on the top row with no data mean the display is fine and waiting.", 182);
  add("dcjack", "DC barrel plug polarity", `
    ${circ(70, 80, 30, "pk-metal")}${circ(70, 80, 18, "pk-screen")}${circ(70, 80, 6, "pk-metal")}${tl(70, 130, ["Centre pin is +", "on most adapters"])}
    ${circ(220, 80, 14, "pk-symo")}${circ(220, 80, 4, "pk-symf")}${path("M234 80h30M206 80h-30M176 74v12", "pk-sym")}${t(170, 84, "−", "pk-txt", "end")}${t(272, 84, "+", "pk-txt", "start")}${tl(220, 130, ["This symbol on the", "label means centre +"])}`,
    "Check the symbol on the adapter's label: most are centre-positive, but not all.");
  add("usbpins", "USB pins and wires", `
    ${rect(40, 50, 120, 50, "pk-metal", 6)}${rect(52, 60, 96, 30, "pk-screen", 3)}${[0, 1, 2, 3].map(i => rect(60 + i * 22, 66, 14, 8, "pk-copperf", 1) + t(67 + i * 22, 112, ["5V", "D−", "D+", "GND"][i], "pk-small")).join("")}
    ${[["red", "5V (VBUS)"], ["white", "D−"], ["green", "D+"], ["black", "GND"]].map(([c, s], i) => rect(210, 50 + i * 22, 22, 12, "c-" + c, 2) + t(240, 60 + i * 22, s, "pk-small", "start")).join("")}
    ${t(170, 160, "A charge-only cable has only red and black.", "pk-small")}`,
    "USB-A pin order (looking into the socket, contacts up) and the usual wire colours.");

  // ================================================================ how to connect
  add("meter-dcv", "Measuring a voltage", `
    ${rect(130, 40, 190, 120, "pk-board", 6)}${rect(160, 60, 40, 30, "pk-body", 2)}${t(180, 80, "REG", "pk-chipsm")}${circ(230, 120, 6, "pk-copperf")}${t(230, 140, "5V", "pk-small")}${circ(290, 140, 8, "pk-metalf")}${t(286, 162, "GND (screw)", "pk-small", "end")}
    ${meter(20, 50, "4.98", "V⎓")}${probe(76, 136, 230, 120, true)}${probe(42, 136, 290, 140, false)}`,
    "Black on ground (a screw hole, shield or GND pin), red on the point. Meter on DC volts.");
  add("meter-diode", "Diode test: both ways", `
    ${meter(10, 40, "0.58", "▶|")}${line(150, 80, 180, 80, "pk-lead")}${rect(180, 70, 46, 20, "pk-body", 6)}${rect(214, 70, 7, 20, "pk-band", 0)}${line(226, 80, 256, 80, "pk-lead")}
    ${probe(66, 126, 150, 80, true)}${probe(32, 126, 256, 80, false)}${t(203, 120, "Red on anode: 0.5–0.7V", "pk-small")}${t(203, 136, "Swap probes: OL", "pk-small")}${t(203, 158, "Both ~0V: shorted · both OL: open", "pk-small")}`,
    "Meter on the diode symbol. Forward one way, blocks the other.");
  add("meter-res", "Measuring resistance: lift a leg", `
    ${rect(150, 70, 170, 80, "pk-board", 6)}${line(190, 100, 190, 76, "pk-lead")}${path("M250 100v-20l20-12", "pk-lead")}${rect(196, 92, 50, 16, "pk-resbody", 7)}${t(270, 64, "lifted", "pk-small")}
    ${meter(10, 40, "4.69k", "Ω")}${probe(66, 126, 190, 80, true)}${probe(32, 126, 270, 68, false)}${tl(235, 166, ["Power off. In the circuit, other", "parts can make it read lower."])}`,
    "Out of the circuit (or one leg lifted), nothing else can change the reading.", 196);
  add("meter-continuity", "Continuity: beeping out a wire", `
    ${rect(150, 40, 70, 40, "pk-body", 3)}${rect(260, 100, 70, 40, "pk-body", 3)}${path("M185 80C185 120 230 100 295 100", "pk-wire")}${circ(185, 80, 4, "pk-copperf")}${circ(295, 100, 4, "pk-copperf")}
    ${meter(10, 40, "0.3Ω", "•)))")}${probe(66, 126, 185, 80, true)}${probe(32, 126, 295, 100, false)}${tl(240, 162, ["Beep = connected.", "Flex the wire while it beeps."])}`,
    "Power off. A steady beep means joined; OL means a break.");
  add("meter-current", "Measuring current: in series", `
    ${rect(20, 60, 60, 40, "pk-psu", 4)}${t(50, 84, "PSU", "pk-chipsm")}${meter(130, 70, "62.4m", "A")}${rect(260, 60, 70, 50, "pk-board", 4)}${t(295, 90, "circuit", "pk-small")}
    ${path("M80 70h40l30 70", "pk-red")}${path("M186 156l20-66h54", "pk-red")}${path("M80 92h-10v80h270v-62h-10", "pk-blk")}
    ${t(200, 30, "Break the + wire and put the meter in the gap.", "pk-small")}${t(200, 46, "Red lead in the A or mA socket.", "pk-small")}`,
    "Current goes through the meter, so it must be in series. Move the red lead back afterwards.");
  add("scope-probe", "Scope probe on a crystal", `
    ${rect(130, 40, 190, 120, "pk-board", 6)}${rect(150, 70, 40, 40, "pk-body", 2)}${t(170, 94, "MCU", "pk-chipsm")}${rect(220, 76, 50, 22, "pk-metal", 10)}${t(245, 92, "16M", "pk-tinydark")}
    ${circ(270, 110, 4, "pk-copperf")}${t(282, 124, "XOUT", "pk-small", "start")}${circ(300, 145, 5, "pk-metalf")}${t(300, 170, "GND", "pk-small")}
    ${rect(30, 60, 50, 16, "pk-probe", 6)}${t(55, 52, "×10", "pk-small")}${path("M80 68Q190 30 270 110", "pk-trace")}${path("M60 76Q120 190 300 145", "pk-blk")}`,
    "×10 probe with the ground lead short; a ×1 probe can stop a crystal oscillating.");
  add("logic-i2c", "Logic analyser on an I2C bus", `
    ${rect(10, 50, 90, 70, "pk-la", 6)}${t(55, 80, "logic", "pk-chipsm")}${t(55, 96, "analyser", "pk-chipsm")}
    ${rect(200, 40, 130, 110, "pk-board", 6)}${rect(220, 60, 40, 40, "pk-body", 2)}${t(240, 84, "MCU", "pk-chipsm")}${rect(280, 70, 30, 30, "pk-body", 2)}${t(295, 90, "sens", "pk-tiny")}
    ${path("M260 74h20M260 88h20")}${circ(270, 74, 3, "pk-copperf")}${circ(270, 88, 3, "pk-copperf")}${circ(230, 130, 5, "pk-metalf")}
    ${path("M100 66Q190 40 270 74", "pk-trace")}${path("M100 82Q190 70 270 88", "pk-trace2")}${path("M100 110Q170 150 230 130", "pk-blk")}
    ${t(150, 40, "D0 → SDA", "pk-small")}${t(150, 100, "D1 → SCL", "pk-small")}${t(150, 160, "GND → GND", "pk-small")}`,
    "Ground first, then one channel per wire. Sample at 4× the bus speed or more.");
  add("serial-hookup", "Reading boot messages with a USB-serial adapter", `
    ${rect(10, 60, 90, 60, "pk-la", 6)}${t(48, 86, "USB-serial", "pk-chipsm")}${t(48, 102, "(3.3V)", "pk-tiny")}
    ${rect(220, 50, 110, 90, "pk-board", 6)}${[["TX", 70], ["RX", 90], ["GND", 110]].map(([s, y]) => circ(240, y, 4, "pk-copperf") + t(250, y + 4, s, "pk-small", "start")).join("")}
    ${t(106, 58, "RX", "pk-tiny", "start")}${t(106, 78, "TX", "pk-tiny", "start")}${t(106, 98, "GND", "pk-tiny", "start")}
    ${path("M100 62C160 62 180 70 236 70", "pk-trace")}${path("M100 82C160 82 180 90 236 90", "pk-trace2")}${path("M100 102C160 102 180 110 236 110", "pk-blk")}
    ${tl(170, 158, ["Cross over: board TX → adapter RX.", "Don't connect the adapter's power pin."])}`,
    "Match the voltage (most boards are 3.3V), cross TX and RX, join grounds.", 190);
  add("inject-short", "Finding a shorted part by injecting current", `
    ${rect(10, 50, 90, 70, "pk-psu", 6)}${t(55, 74, "3.3V", "pk-lcdbig")}${t(55, 94, "limit 1.0A", "pk-small")}${t(55, 110, "CC", "pk-badt")}
    ${rect(150, 30, 180, 130, "pk-board", 6)}${[[190, 60], [230, 60], [270, 60], [190, 110], [230, 110]].map(([x, y]) => rect(x, y, 22, 12, "pk-mlcc", 2)).join("")}
    ${circ(281, 116, 18, "pk-heat")}${rect(270, 110, 22, 12, "pk-mlccHot", 2)}${t(281, 150, "warm!", "pk-badt")}
    ${path("M100 70C130 70 140 50 170 50", "pk-red")}${path("M100 100C130 100 140 140 170 140", "pk-blk")}${t(170, 46, "rail", "pk-tiny", "start")}${t(174, 144, "GND", "pk-tiny", "start")}`,
    "Supply at the rail's voltage, current limited. The shorted part warms first: feel it, film it with IPA, or use a thermal camera.");
  add("signal-trace", "Following a test tone through an audio circuit", `
    ${[["in", 20], ["C", 80], ["pre-amp", 140], ["vol", 200], ["amp", 260], ["spk", 316]].map(([s, x], i) => rect(x - 22, 70, 44, 30, i === 0 || i === 5 ? "pk-body2" : "pk-body", 4) + t(x, 89, s, "pk-tiny")).join("")}
    ${[42, 102, 162, 222, 282].map(x => line(x, 85, x + 16, 85)).join("")}${[50, 110, 170, 230, 290].map((x, i) => circ(x, 85, 4, i < 3 ? "pk-goodf" : "pk-badf")).join("")}
    ${scope(30, 116, 120, 66, u => 0.7 * Math.sin(u * 2 * Math.PI * 4), { label: "here: tone" })}${scope(190, 116, 120, 66, u => 0.03 * Math.sin(u * 40), { label: "here: nothing" })}
    ${t(170, 50, "The fault is between the last good point and the first bad one.", "pk-small")}`,
    "Move the probe one point at a time from the input.", 190);
  add("key-matrix", "Keyboard matrix scanning", `
    ${[0, 1, 2].map(r => line(40, 50 + r * 30, 220, 50 + r * 30) + t(32, 54 + r * 30, "R" + r, "pk-small", "end")).join("")}${[0, 1, 2, 3].map(c => line(70 + c * 45, 36, 70 + c * 45, 128) + t(70 + c * 45, 144, "C" + c, "pk-small")).join("")}
    ${[0, 1, 2].map(r => [0, 1, 2, 3].map(c => circ(70 + c * 45, 50 + r * 30, 5, r === 1 && c === 2 ? "pk-hlf" : "pk-symo")).join("")).join("")}
    ${scope(240, 30, 96, 120, [u => (u > 0.33 && u < 0.5 ? -0.8 : 0.8) * 0.5 + 0.4, u => (u > 0.33 && u < 0.5 ? -0.8 : 0.8) * 0.5 - 0.5], { label: "R1 / C2" })}
    ${tl(170, 168, ["Rows are pulsed in turn; a pressed key", "copies its row's pulse onto its column."])}`,
    "Hold one key and capture its row and column: the column should dip with the row.", 196);
  add("driver-mosfet", "A MOSFET switching a load", `
    ${line(240, 20, 240, 50)}${t(250, 26, "+12V", "pk-small", "start")}${rect(225, 50, 30, 40, "pk-body2", 4)}${t(240, 74, "M", "pk-chipsm")}${path("M275 50v40", "pk-sym")}${path("M268 70l7-8 7 8z", "pk-symf")}${path("M240 50h35M240 90h35", "pk-wire")}${t(290, 74, "flyback", "pk-tiny", "start")}
    ${line(240, 90, 240, 110)}${path("M226 110v40M238 110v40M238 118h20v-8M238 142h20v20M258 118v24", "pk-sym")}${line(180, 130, 226, 130)}${gnd(258, 162)}
    ${circ(160, 130, 4, "pk-hlf")}${t(150, 134, "gate", "pk-small", "end")}${scope(20, 30, 120, 60, u => sq(u, 3, 0.3), { label: "gate: PWM" })}
    ${t(80, 116, "from a 3.3V MCU pin", "pk-tiny")}`,
    "Gate signal present but the load doesn't switch: test the MOSFET and the flyback diode.", 185);
  add("breadboard", "Breadboard power rails", `
    ${rect(10, 40, 320, 110, "pk-bb", 6)}${line(20, 54, 160, 54, "pk-red")}${line(180, 54, 320, 54, "pk-red")}${line(20, 136, 160, 136, "pk-blk")}${line(180, 136, 320, 136, "pk-blk")}
    ${path("M160 54C170 40 170 40 180 54", "pk-hl")}${t(170, 32, "gap!", "pk-badt")}${[...Array(30)].map((_, i) => circ(22 + i * 10, 80, 2, "pk-hole") + circ(22 + i * 10, 110, 2, "pk-hole")).join("")}
    ${tl(170, 168, ["Many long breadboards split their rails in the", "middle: bridge the gap with a wire."])}`,
    "Measure the rail at both ends.", 196);
  add("i2c-pullups", "I2C needs pull-up resistors", `
    ${line(20, 40, 320, 40, "pk-red")}${t(26, 34, "3.3V", "pk-small", "start")}${[110, 150].map(x => rect(x - 6, 50, 12, 36, "pk-resbody", 5) + line(x, 40, x, 50) + line(x, 86, x, x === 110 ? 110 : 140)).join("")}
    ${line(40, 110, 320, 110)}${line(40, 140, 320, 140)}${t(32, 114, "SDA", "pk-small", "end")}${t(32, 144, "SCL", "pk-small", "end")}${rect(200, 96, 50, 60, "pk-body", 3)}${t(225, 130, "sensor", "pk-tiny")}
    ${t(130, 72, "4.7k", "pk-small")}${t(270, 70, "Idle: both lines high", "pk-small")}`,
    "Without pull-ups (or with them too weak), the lines can't rise cleanly.", 170);
  add("ws2812-hookup", "Addressable LED strip wiring", `
    ${rect(10, 60, 70, 60, "pk-board", 4)}${t(45, 94, "MCU", "pk-chipsm")}${rect(180, 70, 150, 40, "pk-strip", 4)}${[200, 240, 280, 320].map(x => rect(x - 12, 80, 18, 18, "pk-ledsq", 2)).join("")}
    ${path("M80 90h40", "pk-trace")}${rect(120, 84, 26, 12, "pk-resbody", 5)}${t(133, 78, "330Ω", "pk-tiny")}${path("M146 90h34", "pk-trace")}
    ${path("M180 76h-20v-46h-80", "pk-red")}${path("M180 104h-20v50h-110v-34", "pk-blk")}${t(76, 26, "5V supply", "pk-small", "end")}${rect(150, 120, 16, 24, "pk-capbody", 3)}${t(174, 136, "1000µF", "pk-tiny", "start")}
    ${t(250, 136, "across 5V", "pk-tiny", "start")}${t(170, 180, "Join the grounds. 3.3V data may need a level shifter.", "pk-small")}`,
    "Resistor in the data line, big capacitor across the supply, shared ground.", 190);
  add("decoupling", "Decoupling capacitors", `
    ${dip(110, 40, 8, {}, "chip", [4, 8])}${rect(250, 46, 22, 14, "pk-mlcc", 2)}${path("M203 54h47M272 54h18v56h-87", "pk-wire")}
    ${t(212, 46, "8 VCC", "pk-tiny", "start")}${t(212, 104, "4 GND", "pk-tiny", "start")}${t(261, 36, "100nF", "pk-small")}${t(170, 160, "Every chip gets one, as close to its power pins as possible.", "pk-small")}`,
    "Missing decoupling makes chips glitch when something switches.", 175);
  add("smps-map", "Inside a mains power supply", `
    ${rect(10, 30, 170, 130, "pk-hot", 6)}${rect(186, 30, 144, 130, "pk-cold", 6)}${t(95, 24, "HOT side: mains, dangerous", "pk-badt")}${t(258, 24, "COLD side: low voltage", "pk-goodt")}
    ${[["fuse", 40, 60], ["bridge", 90, 60], ["bulk cap", 140, 60], ["MOSFET", 60, 120], ["controller", 130, 120]].map(([s, x, y]) => rect(x - 26, y - 14, 52, 28, "pk-body", 3) + t(x, y + 4, s, "pk-tiny")).join("")}
    ${rect(166, 70, 40, 60, "pk-xfmr", 3)}${t(186, 104, "T", "pk-chipsm")}
    ${[["diode", 240, 60], ["out caps", 300, 60], ["TL431", 300, 120], ["opto", 230, 120]].map(([s, x, y]) => rect(x - 24, y - 14, 48, 28, "pk-body", 3) + t(x, y + 4, s, "pk-tiny")).join("")}
    ${tl(170, 174, ["Opto and TL431 carry the feedback;", "startup resistors feed the controller."])}`,
    "Test the hot side only unplugged with the bulk capacitor discharged. Measure live only on the cold side.", 200);
  add("cell-pack", "Measuring each cell in a pack", `
    ${[0, 1, 2].map(i => rect(30 + i * 90, 60, 80, 40, "pk-cell", 8) + t(70 + i * 90, 85, ["3.92V", "3.90V", "2.41V"][i], i === 2 ? "pk-badt" : "pk-chipdark")).join("")}
    ${rect(30, 130, 260, 26, "pk-board", 4)}${t(160, 148, "protection board (BMS)", "pk-tiny")}${path("M110 80h0M200 80h0")}
    ${t(160, 40, "A cell or group far below the rest drags the pack down.", "pk-small")}`,
    "Measure across each cell or parallel group, not just the whole pack.", 170);

  // ================================================================ what the signals should look like
  const two = (title, l, r, say, h = 150) => add(title[0], title[1], `${good(10, 14)}${bad(176, 14)}${l}${r}`, say, h);
  two(["wave-crystal", "Crystal: running vs stopped"], scope(10, 22, 154, 116, u => 0.75 * Math.sin(u * 2 * Math.PI * 6), { label: "16.00 MHz" }), scope(176, 22, 154, 116, u => 0.02 * Math.sin(u * 50), { label: "flat" }),
    "A crystal output is a sine or rounded square at its marked frequency.");
  two(["wave-switching", "Switching regulator node"], scope(10, 22, 154, 116, u => sq(u, 6, 0.42), { label: "square, ~VIN" }), scope(176, 22, 154, 116, () => -0.85, { label: "stuck low" }),
    "A buck converter's switch node is a square wave from 0 to about the input voltage.");
  two(["wave-i2c-edges", "I2C edges: pull-ups"], scope(10, 22, 154, 116, bits("1100110011110000"), { label: "sharp edges" }),
    scope(176, 22, 154, 116, u => { const b = bits("1100110011110000")(u); const k = (u * 16) % 1; return b > 0 ? -0.7 + 1.4 * (1 - Math.exp(-k * 3.5)) : -0.7; }, { label: "slow rise" }),
    "Rounded, slow rising edges mean the pull-ups are too weak or missing.");
  add("wave-i2c", "I2C: an answer (ACK) vs no answer (NAK)", `
    ${scope(10, 14, 320, 70, [bits("1110000110011001100110011001100111"), bits("1101010101010101010101010101010111")], { label: "SDA / SCL", right: "addr 0x3C · ACK" })}
    ${scope(10, 92, 320, 70, [bits("1110000110011001100110011001100111".slice(0, 29) + "11111"), bits("1101010101010101010101010101010111")], { label: "SDA / SCL", right: "addr 0x3C · NAK" })}
    ${tl(170, 176, ["After the 8 address bits, the chip pulls SDA low for one clock (ACK).", "Staying high (NAK) means nobody answered."], "pk-tiny", "middle", 11)}`,
    "Probie's I2C decoder marks ACK and NAK for you.", 192);
  add("wave-uart", "Serial (UART): one byte", `
    ${scope(10, 20, 320, 100, bits("1110110010110111"), { label: "idle · start · 8 data bits · stop" })}
    ${t(170, 140, "Bit width = 1 ÷ baud rate: 8.7µs at 115200, 104µs at 9600.", "pk-small")}${t(170, 158, "Garbled text = wrong baud rate (or wrong voltage level).", "pk-small")}`,
    "Measure one bit's width to find the baud rate.", 170);
  add("wave-reset", "Reset line: held vs pulsing", `${good(10, 14)}${bad(176, 14)}${scope(10, 22, 154, 116, u => (u < 0.08 ? -0.8 : 0.8), { label: "low, then high" })}
    ${scope(176, 22, 154, 116, u => ((u * 4) % 1) < 0.08 ? -0.8 : 0.8, { label: "keeps pulsing" })}`,
    "Released once at power-up is right. Repeated pulses mean something keeps resetting it.", 150);
  add("wave-composite", "Composite video: one line", `
    ${scope(10, 20, 320, 110, u => u < 0.07 ? -0.9 : u < 0.11 ? -0.55 : u < 0.15 ? -0.55 + 0.12 * Math.sin(u * 900) : u < 0.95 ? -0.55 + 1.3 * Math.abs(Math.sin(u * 9)) * 0.9 : -0.55, { label: "sync · colour burst · picture", right: "64µs" })}
    ${t(170, 150, "About 1V peak to peak; sync pulses every 64µs (15.6kHz).", "pk-small")}`,
    "No sync pulses: the video chip isn't running. Sync but no picture: the output stage.", 165);
  add("wave-vga", "VGA sync", `${scope(10, 20, 320, 60, u => sq(u, 10, 0.88), { label: "HSYNC ~31.5kHz", right: "pin 13" })}${scope(10, 88, 320, 60, u => sq(u, 1.3, 0.97), { label: "VSYNC 60Hz", right: "pin 14" })}`,
    "Both should be steady pulse trains; RGB on pins 1–3 swings 0–0.7V.", 155);
  two(["wave-pwm", "PWM: what duty cycle means"], scope(10, 22, 154, 116, u => sq(u, 4, 0.25), { label: "25% duty" }), scope(176, 22, 154, 116, u => sq(u, 4, 0.75), { label: "75% duty" }),
    "Same frequency; the on-time sets the average (brightness, speed).");
  add("wave-servo", "Servo signal", `${scope(10, 20, 320, 100, u => ((u * 2) % 1) < 0.075 ? 0.8 : -0.8, { label: "50Hz (20ms)", right: "1–2ms pulse" })}
    ${t(170, 140, "1ms = one end, 1.5ms = middle, 2ms = the other end.", "pk-small")}`, "Probie shows about 50Hz and 5–10% duty.", 155);
  two(["wave-brownout", "Supply when a load switches on"], scope(10, 22, 154, 116, u => 0.6 + 0.02 * Math.sin(u * 80), { label: "steady" }),
    scope(176, 22, 154, 116, u => u > 0.4 && u < 0.6 ? 0.6 - 1.2 * Math.sin((u - 0.4) * Math.PI / 0.2) : 0.6, { label: "dips" }),
    "A dip below the chip's minimum resets it (brownout): add capacitance or a separate supply.");
  two(["wave-noise", "Clean vs noisy signal"], scope(10, 22, 154, 116, u => sq(u, 3), { label: "clean" }),
    scope(176, 22, 154, 116, u => sq(u, 3) + 0.15 * Math.sin(u * 700) + ((u * 3) % 0.5 < 0.02 ? 0.35 : 0), { label: "ringing, noise" }),
    "Ringing and noise come from long wires, missing decoupling or a shared ground.");
  two(["wave-tone", "Test tone"], scope(10, 22, 154, 116, u => 0.7 * Math.sin(u * 2 * Math.PI * 4), { label: "1kHz sine" }),
    scope(176, 22, 154, 116, u => Math.max(-0.45, Math.min(0.45, 0.9 * Math.sin(u * 2 * Math.PI * 4))), { label: "clipped" }),
    "Flattened tops mean the stage is clipping: too much level, or its bias is off.");
  add("bulb", "Dim bulb tester: what the bulb tells you", `
    ${[["flash, then dim", "starts normally", "pk-goodt", 0.35], ["stays bright", "still a short", "pk-badt", 1], ["pulses", "cycling", "pk-txt", 0.7]].map(([a, b, c, k], i) =>
      `<circle class="pk-glow" cx="${60 + i * 110}" cy="70" r="${18 + 18 * k}" opacity="${0.25 + 0.6 * k}"/>${circ(60 + i * 110, 70, 16, "pk-bulb")}${t(60 + i * 110, 130, a, c)}${t(60 + i * 110, 146, b, "pk-small")}`).join("")}`,
    "Use an old filament bulb, in a ready-made or properly enclosed tester.", 160);
})();
