// Wirie parts library — by nrmluvdev
// All dimensions in millimetres. Footprints are centred on (0,0), Y grows downward.
// pad: {n:number-ish id, name:label, x, y, shape:'rect'|'circle'|'oval', w, h, drill (0 = SMD)}
"use strict";

const LIB = [];          // all footprints
const LIB_BY_ID = {};
const P = 2.54;          // standard pitch

function addFP(fp) {
  fp.pads = fp.pads || [];
  fp.holes = fp.holes || [];
  fp.prefix = fp.prefix || "U";
  fp.value = fp.value || fp.name;
  fp.smd = fp.pads.length > 0 && fp.pads.every(p => !p.drill);
  if (!fp.body) {
    let x0 = 1e9, y0 = 1e9, x1 = -1e9, y1 = -1e9;
    for (const p of fp.pads) { x0 = Math.min(x0, p.x - p.w / 2); x1 = Math.max(x1, p.x + p.w / 2); y0 = Math.min(y0, p.y - p.h / 2); y1 = Math.max(y1, p.y + p.h / 2); }
    for (const h of fp.holes) { x0 = Math.min(x0, h.x - h.d / 2); x1 = Math.max(x1, h.x + h.d / 2); y0 = Math.min(y0, h.y - h.d / 2); y1 = Math.max(y1, h.y + h.d / 2); }
    if (x0 > x1) { x0 = y0 = -1; x1 = y1 = 1; }
    fp.body = { x: (x0 + x1) / 2, y: (y0 + y1) / 2, w: x1 - x0 + 0.6, h: y1 - y0 + 0.6, col: fp.col || "#333" };
  }
  if (fp.silk === undefined) fp.silk = "box";
  fp.search = (fp.name + " " + (fp.desc || "") + " " + (fp.tags || "") + " " + fp.cat + " " + fp.pads.map(p => p.name).join(" ")).toLowerCase();
  LIB.push(fp);
  LIB_BY_ID[fp.id] = fp;
  return fp;
}

function tht(n, x, y, o = {}) {
  const d = o.d || 1.7;
  return { n: String(n), name: o.name || String(n), x, y, shape: o.shape || (o.sq ? "rect" : "circle"), w: o.w || d, h: o.h || d, drill: o.slot ? Math.min(...o.slot) : o.drill || 1.0, slot: o.slot || null, func: o.func || "" };
}
function smd(n, x, y, w, h, o = {}) {
  return { n: String(n), name: o.name || String(n), x, y, shape: o.shape || "rect", w, h, drill: 0, opp: !!o.opp, func: o.func || "" };
}

// ---------------- pin headers ----------------
function headerPads(cols, rows, pitch, names, padD, drill) {
  const pads = [];
  const w = (cols - 1) * pitch, h = (rows - 1) * pitch;
  let n = 1;
  for (let c = 0; c < cols; c++) for (let r = 0; r < rows; r++) {
    pads.push(tht(n, -w / 2 + c * pitch, -h / 2 + r * pitch, { d: padD, drill, sq: n === 1, name: names ? names[n - 1] : String(n) }));
    n++;
  }
  return pads;
}
for (let n = 1; n <= 40; n++) addFP({ id: `hdr1x${n}`, name: `Pin header 1x${n} (2.54mm)`, cat: "Headers & Sockets", prefix: "J", desc: "Straight male header or female socket, 0.1in pitch", tags: "pinheader socket dupont berg", pads: headerPads(n, 1, P, null, 1.7, 1.0), col: "#222" });
for (let n = 2; n <= 40; n++) addFP({ id: `hdr2x${n}`, name: `Pin header 2x${n} (2.54mm)`, cat: "Headers & Sockets", prefix: "J", desc: "Dual-row male header or female socket, 0.1in pitch", tags: "pinheader socket dupont idc", pads: (() => { const p = []; let k = 1; const w = (n - 1) * P; for (let c = 0; c < n; c++) for (let r = 0; r < 2; r++) { p.push(tht(k, -w / 2 + c * P, (r - 0.5) * P, { sq: k === 1 })); k++; } return p; })(), col: "#222" });
for (let n = 2; n <= 20; n++) addFP({ id: `hdr2mm1x${n}`, name: `Pin header 1x${n} (2.00mm)`, cat: "Headers & Sockets", prefix: "J", desc: "2mm pitch header", pads: headerPads(n, 1, 2.0, null, 1.35, 0.8), col: "#222" });
for (const n of [5, 7, 8, 10, 13, 17, 20]) addFP({ id: `idc2x${n}`, name: `IDC box header 2x${n}`, cat: "Headers & Sockets", prefix: "J", desc: "Shrouded ribbon-cable header", tags: "idc ribbon shrouded", pads: (() => { const p = []; let k = 1; const w = (n - 1) * P; for (let c = 0; c < n; c++) for (let r = 0; r < 2; r++) { p.push(tht(k, -w / 2 + c * P, (0.5 - r) * P, { sq: k === 1 })); k++; } return p; })(), body: { x: 0, y: 0, w: (n - 1) * P + 10.2, h: 8.9, col: "#222" } });

// ---------------- connectors ----------------
for (let n = 2; n <= 8; n++) addFP({ id: `jstxh${n}`, name: `JST-XH ${n}-pin (2.5mm)`, cat: "Connectors", prefix: "J", desc: "Common battery-balance / sensor connector", tags: "jst xh", pads: headerPads(n, 1, 2.5, null, 1.7, 0.95), body: { x: 0, y: 0.6, w: (n - 1) * 2.5 + 4.9, h: 5.75, col: "#eee" } });
for (let n = 2; n <= 6; n++) addFP({ id: `jstph${n}`, name: `JST-PH ${n}-pin (2.0mm)`, cat: "Connectors", prefix: "J", desc: "LiPo battery style connector (check battery polarity!)", tags: "jst ph lipo battery", pads: headerPads(n, 1, 2.0, null, 1.25, 0.75), body: { x: 0, y: 0.75, w: (n - 1) * 2 + 3.9, h: 4.5, col: "#eee" } });
for (let n = 2; n <= 6; n++) addFP({ id: `term5_${n}`, name: `Screw terminal ${n}-way (5.08mm)`, cat: "Connectors", prefix: "J", desc: "Wire terminal block, for power and motors", tags: "terminal block screw kf301", pads: headerPads(n, 1, 5.08, null, 2.4, 1.3), body: { x: 0, y: 0, w: n * 5.08, h: 7.6, col: "#2a7de1" } });
for (let n = 2; n <= 6; n++) addFP({ id: `term35_${n}`, name: `Screw terminal ${n}-way (3.5mm)`, cat: "Connectors", prefix: "J", desc: "Small wire terminal block", tags: "terminal block screw", pads: headerPads(n, 1, 3.5, null, 2.0, 1.1), body: { x: 0, y: 0, w: n * 3.5, h: 7, col: "#2a7de1" } });
addFP({ id: "wirepad2", name: "Wire solder pads (2)", cat: "Connectors", prefix: "J", desc: "Large pads to solder wires straight on", tags: "wire pad battery", pads: [tht(1, -1.905, 0, { d: 2.5, drill: 1.3, sq: true, name: "+" }), tht(2, 1.905, 0, { d: 2.5, drill: 1.3, name: "-" })] });
addFP({ id: "wirepad1", name: "Wire solder pad (1)", cat: "Connectors", prefix: "TP", desc: "Single large through-hole pad", pads: [tht(1, 0, 0, { d: 2.5, drill: 1.3 })] });
addFP({ id: "usbc_bo", name: "USB-C power breakout (VBUS/GND)", cat: "Connectors", prefix: "J", desc: "Plug-in USB-C power board with 2 pins", tags: "usb type-c power 5v", pads: headerPads(2, 1, P, ["VBUS", "GND"], 1.7, 1.0), body: { x: 0, y: -3.5, w: 10, h: 10, col: "#1b1b1b" } });
addFP({ id: "musb_bo", name: "Micro-USB breakout (5-pin)", cat: "Connectors", prefix: "J", desc: "Plug-in micro-USB board", tags: "usb micro", pads: headerPads(5, 1, P, ["VCC", "D-", "D+", "ID", "GND"], 1.7, 1.0), body: { x: 0, y: -4, w: 14, h: 11, col: "#8a1b1b" } });
addFP({ id: "tp", name: "Test point", cat: "Connectors", prefix: "TP", desc: "Probe point for your multimeter or scope", tags: "test probe", pads: [tht(1, 0, 0, { d: 2.0, drill: 1.0 })], silk: "none" });

// ---------------- passives: resistors / caps / inductors ----------------
const SMD_SIZES = { "0402": [0.6, 0.65, 0.48, 1.0, 0.5], "0603": [0.9, 0.95, 0.8, 1.6, 0.8], "0805": [1.0, 1.45, 0.95, 2.0, 1.25], "1206": [1.15, 1.8, 1.45, 3.2, 1.6] };
function smd2(size, n1 = "1", n2 = "2") { const s = SMD_SIZES[size]; return [smd(1, -s[2], 0, s[0], s[1], { name: n1 }), smd(2, s[2], 0, s[0], s[1], { name: n2 })]; }
function body2(size, col) { const s = SMD_SIZES[size]; return { x: 0, y: 0, w: s[3], h: s[4], col }; }
for (const [lbl, pitch] of [["7.62mm / 0.3in", 7.62], ["10.16mm / 0.4in", 10.16], ["12.7mm / 0.5in", 12.7]])
  addFP({ id: `r_axial_${pitch}`, name: `Resistor (through-hole, ${lbl})`, cat: "Resistors", prefix: "R", value: "10k", desc: "Standard 1/4W leaded resistor lying flat", tags: "resistor axial tht", pads: [tht(1, -pitch / 2, 0, { d: 1.6, drill: 0.8 }), tht(2, pitch / 2, 0, { d: 1.6, drill: 0.8 })], body: { x: 0, y: 0, w: pitch - 2.6, h: 2.5, col: "#d9c28f" } });
addFP({ id: "r_vert", name: "Resistor (through-hole, standing)", cat: "Resistors", prefix: "R", value: "10k", desc: "Leaded resistor standing upright, saves space", tags: "resistor vertical", pads: [tht(1, -1.27, 0, { d: 1.6, drill: 0.8 }), tht(2, 1.27, 0, { d: 1.6, drill: 0.8 })], body: { x: -1.27, y: 0, w: 2.6, h: 2.6, col: "#d9c28f" } });
for (const s of ["0402", "0603", "0805", "1206"]) addFP({ id: `r_${s}`, name: `Resistor SMD ${s}`, cat: "Resistors", prefix: "R", value: "10k", desc: `Surface-mount resistor, ${s} size${s === "0805" || s === "1206" ? " (hand-solderable)" : ""}`, tags: "resistor smd chip", pads: smd2(s), body: body2(s, "#222"), silk: "sides" });
addFP({ id: "trim3296", name: "Trimmer pot 3296W (inline)", cat: "Resistors", prefix: "RV", value: "10k", desc: "Multi-turn trimmer, 3 pins in a line", tags: "potentiometer trimmer variable", pads: headerPads(3, 1, P, ["1", "W", "3"], 1.6, 0.9), body: { x: 0, y: 0, w: 9.5, h: 4.8, col: "#2a5bd6" } });
addFP({ id: "pot16", name: "Potentiometer 16mm (panel)", cat: "Resistors", prefix: "RV", value: "10k", desc: "Big knob pot, 5mm pin spacing", tags: "potentiometer knob variable", pads: headerPads(3, 1, 5.0, ["1", "W", "3"], 2.0, 1.2), body: { x: 0, y: -7, w: 17, h: 17, col: "#555" } });
addFP({ id: "rv09", name: "Potentiometer RV09 (9mm)", cat: "Resistors", prefix: "RV", value: "10k", desc: "Small board-mount pot", tags: "potentiometer knob", pads: [...headerPads(3, 1, 2.5, ["1", "W", "3"], 1.8, 1.0)], holes: [], body: { x: 0, y: -6, w: 10, h: 11, col: "#555" } });

for (const [lbl, pitch] of [["2.54mm", 2.54], ["5.08mm", 5.08]])
  addFP({ id: `c_disc_${pitch}`, name: `Capacitor ceramic (through-hole, ${lbl})`, cat: "Capacitors", prefix: "C", value: "100nF", desc: "Blue/yellow disc or MLCC, not polarised", tags: "capacitor ceramic decoupling 100n", pads: [tht(1, -pitch / 2, 0, { d: 1.6, drill: 0.8 }), tht(2, pitch / 2, 0, { d: 1.6, drill: 0.8 })], body: { x: 0, y: 0, w: pitch + 2, h: 2.6, col: "#e2b02a" } });
for (const [d, p, dr] of [[5, 2.0, 0.8], [6.3, 2.5, 0.8], [8, 3.5, 0.8], [10, 5.0, 1.0], [12.5, 5.0, 1.0]])
  addFP({ id: `c_elec_${d}`, name: `Electrolytic capacitor D${d}mm`, cat: "Capacitors", prefix: "C", value: d < 8 ? "10uF" : "470uF", desc: "Polarised! Long leg (+) goes to the square pad", tags: "capacitor electrolytic polarised bulk", pads: [tht(1, -p / 2, 0, { d: Math.min(1.8, p * 0.8), drill: dr, sq: true, name: "+" }), tht(2, p / 2, 0, { d: Math.min(1.8, p * 0.8), drill: dr, name: "-" })], body: { x: 0, y: 0, w: d, h: d, round: true, col: "#1b3e8a" }, silk: "circle" });
for (const s of ["0402", "0603", "0805", "1206"]) addFP({ id: `c_${s}`, name: `Capacitor SMD ${s}`, cat: "Capacitors", prefix: "C", value: "100nF", desc: "Surface-mount ceramic capacitor", tags: "capacitor smd mlcc", pads: smd2(s), body: body2(s, "#b58a52"), silk: "sides" });
addFP({ id: "c_tant_a", name: "Tantalum capacitor SMD (A / 1206)", cat: "Capacitors", prefix: "C", value: "10uF", desc: "Polarised, stripe = +", tags: "tantalum", pads: [smd(1, -1.4, 0, 1.6, 1.5, { name: "+" }), smd(2, 1.4, 0, 1.6, 1.5, { name: "-" })], body: { x: 0, y: 0, w: 3.2, h: 1.6, col: "#d9a400" } });
addFP({ id: "l_axial", name: "Inductor (through-hole axial)", cat: "Capacitors", prefix: "L", value: "10uH", desc: "Leaded inductor / choke", tags: "inductor coil choke", pads: [tht(1, -5.08, 0, { d: 1.8, drill: 0.9 }), tht(2, 5.08, 0, { d: 1.8, drill: 0.9 })], body: { x: 0, y: 0, w: 7, h: 3.2, col: "#3a7a3a" } });
for (const s of ["0805", "1206"]) addFP({ id: `l_${s}`, name: `Inductor SMD ${s}`, cat: "Capacitors", prefix: "L", value: "10uH", tags: "inductor ferrite bead", pads: smd2(s), body: body2(s, "#444"), silk: "sides" });
addFP({ id: "xtal_hc49", name: "Crystal HC-49 (through-hole)", cat: "Capacitors", prefix: "Y", value: "16MHz", desc: "Clock crystal for microcontrollers", tags: "crystal oscillator 16mhz 8mhz", pads: [tht(1, -2.44, 0, { d: 1.5, drill: 0.8 }), tht(2, 2.44, 0, { d: 1.5, drill: 0.8 })], body: { x: 0, y: 0, w: 11.4, h: 4.6, col: "#bbb" } });

// ---------------- diodes & LEDs ----------------
addFP({ id: "d_do41", name: "Diode DO-41 (1N4007 / 1N5819)", cat: "Diodes & LEDs", prefix: "D", value: "1N4007", desc: "Rectifier / Schottky. Stripe = cathode (K)", tags: "diode rectifier schottky 1n4001 1n5819", pads: [tht(1, -5.08, 0, { sq: true, d: 2.0, drill: 1.0, name: "K" }), tht(2, 5.08, 0, { d: 2.0, drill: 1.0, name: "A" })], body: { x: 0, y: 0, w: 5.2, h: 2.7, col: "#222", stripe: true } });
addFP({ id: "d_do35", name: "Diode DO-35 (1N4148)", cat: "Diodes & LEDs", prefix: "D", value: "1N4148", desc: "Small signal diode. Stripe = cathode", tags: "diode signal zener", pads: [tht(1, -3.81, 0, { sq: true, d: 1.6, drill: 0.8, name: "K" }), tht(2, 3.81, 0, { d: 1.6, drill: 0.8, name: "A" })], body: { x: 0, y: 0, w: 4, h: 1.9, col: "#c47a3c", stripe: true } });
addFP({ id: "d_sod123", name: "Diode SMD SOD-123", cat: "Diodes & LEDs", prefix: "D", value: "1N4148W", tags: "diode smd", pads: [smd(1, -1.65, 0, 0.9, 1.2, { name: "K" }), smd(2, 1.65, 0, 0.9, 1.2, { name: "A" })], body: { x: 0, y: 0, w: 2.7, h: 1.6, col: "#222", stripe: true } });
addFP({ id: "d_sma", name: "Diode SMD SMA (DO-214AC)", cat: "Diodes & LEDs", prefix: "D", value: "SS14", tags: "diode schottky smd ss34 m7", pads: [smd(1, -2.0, 0, 2.5, 1.7, { name: "K" }), smd(2, 2.0, 0, 2.5, 1.7, { name: "A" })], body: { x: 0, y: 0, w: 4.3, h: 2.6, col: "#222", stripe: true } });
for (const d of [3, 5]) addFP({ id: `led${d}`, name: `LED ${d}mm (through-hole)`, cat: "Diodes & LEDs", prefix: "D", value: "Red", desc: "Long leg = anode (A). Flat side / square pad = cathode (K). Needs a resistor!", tags: "led light indicator", pads: [tht(1, -1.27, 0, { sq: true, d: 1.8, drill: 0.9, name: "K" }), tht(2, 1.27, 0, { d: 1.8, drill: 0.9, name: "A" })], body: { x: 0, y: 0, w: d + 0.8, h: d + 0.8, round: true, col: "#d22" }, silk: "circle" });
for (const s of ["0603", "0805", "1206"]) addFP({ id: `led_${s}`, name: `LED SMD ${s}`, cat: "Diodes & LEDs", prefix: "D", value: "Green", desc: "Tiny surface-mount LED. Needs a resistor!", tags: "led smd", pads: smd2(s, "K", "A"), body: body2(s, "#7d7"), silk: "sides" });
addFP({ id: "ws2812b", name: "WS2812B addressable RGB LED (5050)", cat: "Diodes & LEDs", prefix: "D", value: "WS2812B", desc: "NeoPixel. Chain DOUT to next DIN. Verify pad layout with your LED datasheet.", tags: "neopixel rgb addressable sk6812", verify: true, pads: [smd(1, -2.45, -1.65, 1.5, 0.9, { name: "VDD", func: "PWR5" }), smd(2, -2.45, 1.65, 1.5, 0.9, { name: "DOUT", func: "DOUT" }), smd(3, 2.45, 1.65, 1.5, 0.9, { name: "GND" }), smd(4, 2.45, -1.65, 1.5, 0.9, { name: "DIN", func: "DIN" })], body: { x: 0, y: 0, w: 5, h: 5, col: "#eee" } });
addFP({ id: "led_rgb5", name: "RGB LED 5mm common-cathode (4-pin)", cat: "Diodes & LEDs", prefix: "D", value: "RGB", tags: "led rgb", pads: headerPads(4, 1, P, ["R", "K", "G", "B"], 1.6, 0.9), body: { x: 0, y: 0, w: 6, h: 6, round: true, col: "#ddd" }, silk: "circle" });

// ---------------- transistors, regulators ----------------
function to92(id, name, pins, value, desc, tags) { return addFP({ id, name, cat: "Transistors & Power", prefix: "Q", value, desc, tags, pads: headerPads(3, 1, P, pins, 1.5, 0.8), body: { x: 0, y: 0.4, w: 5.2, h: 4.4, col: "#222", dshape: true } }); }
to92("to92_ebc", "Transistor TO-92 (E-B-C) 2N2222/2N3904", ["E", "B", "C"], "2N2222", "NPN switching transistor", "npn bjt transistor 2n3904 2n2222 pn2222");
to92("to92_cbe", "Transistor TO-92 (C-B-E) BC547/BC557", ["C", "B", "E"], "BC547", "European pinout NPN/PNP", "npn bjt bc547 bc337");
to92("to92_gds", "MOSFET TO-92 (S-G-D) 2N7000", ["S", "G", "D"], "2N7000", "Small N-channel MOSFET", "mosfet 2n7000 bs170");
to92("ds18b20", "DS18B20 temperature sensor (TO-92)", ["GND", "DQ", "VDD"], "DS18B20", "1-Wire temperature sensor. 4.7k pull-up from DQ to VDD", "temperature sensor onewire");
to92("to92_reg", "Voltage regulator TO-92 (78L05)", ["OUT", "GND", "IN"], "78L05", "Small 100mA regulator", "regulator ldo");
function to220(id, name, pins, value, desc, tags) { return addFP({ id, name, cat: "Transistors & Power", prefix: id.startsWith("reg") ? "U" : "Q", value, desc, tags, pads: [tht(1, -P, 0, { shape: "oval", w: 1.9, h: 2.6, drill: 1.1, sq: true, name: pins[0] }), tht(2, 0, 0, { shape: "oval", w: 1.9, h: 2.6, drill: 1.1, name: pins[1] }), tht(3, P, 0, { shape: "oval", w: 1.9, h: 2.6, drill: 1.1, name: pins[2] })].map((p, i) => (i === 0 ? Object.assign(p, { shape: "rect" }) : p)), body: { x: 0, y: -0.9, w: 10.4, h: 4.6, col: "#333" } }); }
to220("reg7805", "Voltage regulator 7805 (TO-220)", ["IN", "GND", "OUT"], "LM7805", "5V 1A regulator. Add 10uF in, 100nF out", "regulator 7805 7812 7833 5v");
to220("reglm317", "LM317 adjustable regulator (TO-220)", ["ADJ", "OUT", "IN"], "LM317", "Adjustable 1.25-37V regulator", "regulator adjustable lm317");
to220("q_to220_gds", "MOSFET TO-220 (G-D-S) IRLZ44N", ["G", "D", "S"], "IRLZ44N", "Logic-level N-MOSFET for motors / LED strips", "mosfet irlz44n irf540 power");
to220("q_to220_bce", "Transistor TO-220 (B-C-E) TIP120", ["B", "C", "E"], "TIP120", "Darlington power transistor", "darlington tip120 tip122");
addFP({ id: "sot23", name: "SOT-23 (3-pin)", cat: "Transistors & Power", prefix: "Q", value: "AO3400", desc: "Small SMD transistor/MOSFET. Pin names follow common MOSFET pinout", tags: "sot23 mosfet smd ao3400 bss138 mmbt3904", pads: [smd(1, -1.14, -0.95, 1.3, 0.6, { name: "G" }), smd(2, -1.14, 0.95, 1.3, 0.6, { name: "S" }), smd(3, 1.14, 0, 1.3, 0.6, { name: "D" })], body: { x: 0, y: 0, w: 1.4, h: 3, col: "#222" }, silk: "none" });
addFP({ id: "sot23_5", name: "SOT-23-5", cat: "Transistors & Power", prefix: "U", value: "LDO", tags: "sot23-5 ldo smd", pads: [smd(1, -1.14, -0.95, 1.3, 0.6), smd(2, -1.14, 0, 1.3, 0.6), smd(3, -1.14, 0.95, 1.3, 0.6), smd(4, 1.14, 0.95, 1.3, 0.6), smd(5, 1.14, -0.95, 1.3, 0.6)], body: { x: 0, y: 0, w: 1.6, h: 3, col: "#222" }, silk: "none" });
addFP({ id: "ams1117", name: "AMS1117 regulator (SOT-223)", cat: "Transistors & Power", prefix: "U", value: "AMS1117-3.3", desc: "3.3V 1A LDO, very common. Add 10uF caps in & out", tags: "regulator ldo 3.3v 3v3 sot223 ams1117", pads: [smd(1, -3.15, 2.3, 2, 1.5, { name: "GND" }), smd(2, -3.15, 0, 2, 1.5, { name: "OUT" }), smd(3, -3.15, -2.3, 2, 1.5, { name: "IN" }), smd(4, 3.15, 0, 2, 3.8, { name: "OUT" })], body: { x: 0, y: 0, w: 3.6, h: 6.6, col: "#222" } });
addFP({ id: "relay_srd", name: "Relay SRD-05VDC (Songle)", cat: "Transistors & Power", prefix: "K", value: "SRD-05VDC", desc: "5V relay. Needs transistor + flyback diode", tags: "relay songle switch mains", verify: true, pads: [tht(1, -6.0, -6.0, { d: 2.0, drill: 1.1, name: "COIL1" }), tht(2, -6.0, 6.0, { d: 2.0, drill: 1.1, name: "COIL2" }), tht(3, -8.0, 0, { d: 2.0, drill: 1.1, name: "COM" }), tht(4, 6.0, -6.0, { d: 2.0, drill: 1.1, name: "NC" }), tht(5, 6.0, 6.0, { d: 2.0, drill: 1.1, name: "NO" })], body: { x: -1, y: 0, w: 19.2, h: 15.6, col: "#1f5fc9" } });
addFP({ id: "buzzer12", name: "Buzzer 12mm (7.6mm pitch)", cat: "Transistors & Power", prefix: "BZ", value: "Buzzer", desc: "Active or passive buzzer", tags: "buzzer beeper piezo sound", pads: [tht(1, -3.8, 0, { sq: true, name: "+" }), tht(2, 3.8, 0, { name: "-" })], body: { x: 0, y: 0, w: 12, h: 12, round: true, col: "#222" }, silk: "circle" });
addFP({ id: "fuse5x20", name: "Polyfuse (through-hole, 5.08mm)", cat: "Transistors & Power", prefix: "F", value: "500mA", tags: "fuse ptc polyfuse resettable", pads: [tht(1, -2.54, 0, { d: 1.8, drill: 0.9 }), tht(2, 2.54, 0, { d: 1.8, drill: 0.9 })], body: { x: 0, y: 0, w: 7.5, h: 3, col: "#d9a400" } });

// ---------------- switches ----------------
addFP({ id: "sw6x6", name: "Push button 6x6mm (tactile)", cat: "Switches", prefix: "SW", value: "Button", desc: "Pins on the same side (1&2 / 3&4) are always connected", tags: "button tactile switch push", pads: [tht(1, -3.25, -2.25, { name: "A" }), tht(2, 3.25, -2.25, { name: "A" }), tht(3, -3.25, 2.25, { name: "B" }), tht(4, 3.25, 2.25, { name: "B" })], body: { x: 0, y: 0, w: 6, h: 6, col: "#333", btn: true } });
addFP({ id: "sw12x12", name: "Push button 12x12mm (tactile)", cat: "Switches", prefix: "SW", value: "Button", tags: "button tactile switch push big", pads: [tht(1, -6.25, -2.5, { d: 2.0, drill: 1.1, name: "A" }), tht(2, 6.25, -2.5, { d: 2.0, drill: 1.1, name: "A" }), tht(3, -6.25, 2.5, { d: 2.0, drill: 1.1, name: "B" }), tht(4, 6.25, 2.5, { d: 2.0, drill: 1.1, name: "B" })], body: { x: 0, y: 0, w: 12, h: 12, col: "#333", btn: true } });
addFP({ id: "sw_slide3", name: "Slide switch SPDT (2.54mm, 3-pin)", cat: "Switches", prefix: "SW", value: "Switch", desc: "On/off switch. Middle pin = common", tags: "slide switch toggle power on off", pads: headerPads(3, 1, P, ["A", "COM", "B"], 1.6, 0.9), body: { x: 0, y: 0, w: 8.6, h: 3.6, col: "#222" } });
for (const n of [2, 4, 8]) addFP({ id: `dip_sw${n}`, name: `DIP switch ${n}-way`, cat: "Switches", prefix: "SW", value: `DIP${n}`, tags: "dip switch config", pads: (() => { const p = []; for (let i = 0; i < n; i++) { p.push(tht(i + 1, -(n - 1) * P / 2 + i * P, 3.81, { sq: i === 0 })); p.push(tht(2 * n - i, -(n - 1) * P / 2 + i * P, -3.81)); } return p; })(), body: { x: 0, y: 0, w: n * P + 1.5, h: 9.8, col: "#c22" } });

// ---------------- ICs: DIP / SOIC / TSSOP / TQFP ----------------
function dipPads(n, rowSp, names) { const p = []; const h = n / 2; for (let i = 0; i < h; i++) p.push(tht(i + 1, -rowSp / 2, -(h - 1) * P / 2 + i * P, { sq: i === 0, d: 1.6, drill: 0.8, name: names ? names[i] : String(i + 1) })); for (let i = 0; i < h; i++) p.push(tht(h + i + 1, rowSp / 2, (h - 1) * P / 2 - i * P, { d: 1.6, drill: 0.8, name: names ? names[h + i] : String(h + i + 1) })); return p; }
function dipFP(id, name, n, rowSp, names, value, desc, tags) { return addFP({ id, name, cat: "Chips (ICs)", prefix: "U", value: value || name, desc: desc || "Through-hole chip. Tip: solder an IC socket instead", tags: (tags || "") + " dip ic chip", pads: dipPads(n, rowSp, names), body: { x: 0, y: 0, w: rowSp - 1.7, h: n / 2 * P + 0.5, col: "#1a1a1a", notch: true } }); }
for (const n of [4, 6, 8, 14, 16, 18, 20, 24, 28]) dipFP(`dip${n}`, `DIP-${n} (0.3in)`, n, 7.62, null, `DIP-${n}`, "Generic through-hole IC / socket");
for (const n of [24, 28, 32, 40]) dipFP(`dip${n}w`, `DIP-${n} wide (0.6in)`, n, 15.24, null, `DIP-${n}`, "Wide through-hole IC / socket");
function soPads(n, pitch, span, pw, ph) { const p = []; const h = n / 2; for (let i = 0; i < h; i++) p.push(smd(i + 1, -span / 2, -(h - 1) * pitch / 2 + i * pitch, pw, ph)); for (let i = 0; i < h; i++) p.push(smd(h + i + 1, span / 2, (h - 1) * pitch / 2 - i * pitch, pw, ph)); return p; }
for (const n of [8, 14, 16]) addFP({ id: `soic${n}`, name: `SOIC-${n} (narrow, 1.27mm)`, cat: "Chips (ICs)", prefix: "U", value: `SOIC-${n}`, tags: "soic so smd ic", pads: soPads(n, 1.27, 5.4, 1.95, 0.6), body: { x: 0, y: 0, w: 3.9, h: n / 2 * 1.27 + 0.4, col: "#1a1a1a", notch: true }, silk: "none" });
for (const n of [16, 18, 20, 24, 28]) addFP({ id: `soic${n}w`, name: `SOIC-${n} wide (1.27mm)`, cat: "Chips (ICs)", prefix: "U", value: `SOIC-${n}W`, tags: "soic wide smd ic", pads: soPads(n, 1.27, 9.3, 2.0, 0.6), body: { x: 0, y: 0, w: 7.5, h: n / 2 * 1.27 + 0.4, col: "#1a1a1a", notch: true }, silk: "none" });
for (const n of [8, 14, 16, 20, 24, 28]) addFP({ id: `tssop${n}`, name: `TSSOP-${n} (0.65mm)`, cat: "Chips (ICs)", prefix: "U", value: `TSSOP-${n}`, tags: "tssop smd ic fine pitch", pads: soPads(n, 0.65, 5.7, 1.5, 0.4), body: { x: 0, y: 0, w: 4.4, h: n / 2 * 0.65 + 0.6, col: "#1a1a1a", notch: true }, silk: "none" });
function qfpPads(n, pitch, span, pw, ph) { const p = []; const s = n / 4; let k = 1; const off = (s - 1) * pitch / 2; for (let i = 0; i < s; i++) p.push(smd(k++, -span / 2, -off + i * pitch, pw, ph)); for (let i = 0; i < s; i++) p.push(smd(k++, -off + i * pitch, span / 2, ph, pw)); for (let i = 0; i < s; i++) p.push(smd(k++, span / 2, off - i * pitch, pw, ph)); for (let i = 0; i < s; i++) p.push(smd(k++, off - i * pitch, -span / 2, ph, pw)); return p; }
addFP({ id: "tqfp32", name: "TQFP-32 (7x7mm, 0.8mm)", cat: "Chips (ICs)", prefix: "U", value: "TQFP-32", tags: "tqfp qfp smd atmega328p-au", verify: true, pads: qfpPads(32, 0.8, 8.4, 1.5, 0.55), body: { x: 0, y: 0, w: 7, h: 7, col: "#1a1a1a", notch: true }, silk: "none" });
addFP({ id: "tqfp44", name: "TQFP-44 (10x10mm, 0.8mm)", cat: "Chips (ICs)", prefix: "U", value: "TQFP-44", tags: "tqfp qfp smd atmega32u4", verify: true, pads: qfpPads(44, 0.8, 11.4, 1.5, 0.55), body: { x: 0, y: 0, w: 10, h: 10, col: "#1a1a1a", notch: true }, silk: "none" });

// named DIP chips
dipFP("atmega328p", "ATmega328P (DIP-28) — Arduino chip", 28, 7.62, ["RESET", "D0/RX", "D1/TX", "D2", "D3", "D4", "VCC", "GND", "XTAL1", "XTAL2", "D5", "D6", "D7", "D8", "D9", "D10/SS", "D11/MOSI", "D12/MISO", "D13/SCK", "AVCC", "AREF", "GND", "A0", "A1", "A2", "A3", "A4/SDA", "A5/SCL"], "ATmega328P", "The Arduino Uno chip. Needs 16MHz crystal + 2x22pF, 10k pull-up on RESET", "microcontroller arduino avr mcu");
dipFP("attiny85", "ATtiny85 (DIP-8)", 8, 7.62, ["RESET/PB5", "PB3", "PB4", "GND", "PB0/MOSI/SDA", "PB1/MISO", "PB2/SCK/SCL", "VCC"], "ATtiny85", "Tiny 8-pin microcontroller", "microcontroller avr attiny mcu digispark");
dipFP("ne555", "NE555 timer (DIP-8)", 8, 7.62, ["GND", "TRIG", "OUT", "RESET", "CTRL", "THR", "DIS", "VCC"], "NE555", "Classic timer chip for blinkers & tones", "555 timer oscillator");
dipFP("lm358", "LM358 dual op-amp (DIP-8)", 8, 7.62, ["OUT1", "IN1-", "IN1+", "GND", "IN2+", "IN2-", "OUT2", "VCC"], "LM358", "Dual op-amp", "opamp amplifier lm358 lm393");
dipFP("hc595", "74HC595 shift register (DIP-16)", 16, 7.62, ["Q1", "Q2", "Q3", "Q4", "Q5", "Q6", "Q7", "GND", "Q7S", "MR", "SHCP", "STCP", "OE", "DS", "Q0", "VCC"], "74HC595", "8 extra outputs from 3 pins", "shift register 595 output expander");
dipFP("uln2003", "ULN2003 driver (DIP-16)", 16, 7.62, ["IN1", "IN2", "IN3", "IN4", "IN5", "IN6", "IN7", "GND", "COM", "OUT7", "OUT6", "OUT5", "OUT4", "OUT3", "OUT2", "OUT1"], "ULN2003", "7x Darlington driver for relays / steppers", "driver darlington stepper 28byj");
dipFP("l293d", "L293D motor driver (DIP-16)", 16, 7.62, ["EN12", "1A", "1Y", "GND", "GND", "2Y", "2A", "VMOT", "EN34", "3A", "3Y", "GND", "GND", "4Y", "4A", "VCC"], "L293D", "Dual H-bridge for 2 DC motors", "motor driver h-bridge");
dipFP("pc817", "PC817 optocoupler (DIP-4)", 4, 7.62, ["A", "K", "E", "C"], "PC817", "Isolates two circuits with light", "optocoupler isolator");
dipFP("cd4017", "CD4017 decade counter (DIP-16)", 16, 7.62, ["Q5", "Q1", "Q0", "Q2", "Q6", "Q7", "Q3", "GND", "Q8", "Q4", "Q9", "CO", "EN", "CLK", "RST", "VCC"], "CD4017", "LED chaser counter", "counter 4017 chaser");
dipFP("mcp23017", "MCP23017 I/O expander (DIP-28)", 28, 7.62, ["GPB0", "GPB1", "GPB2", "GPB3", "GPB4", "GPB5", "GPB6", "GPB7", "VDD", "VSS", "NC", "SCL", "SDA", "NC", "A0", "A1", "A2", "RESET", "INTB", "INTA", "GPA0", "GPA1", "GPA2", "GPA3", "GPA4", "GPA5", "GPA6", "GPA7"], "MCP23017", "16 extra GPIO over I2C", "i2c expander gpio");

// ---------------- modules with named pins ----------------
// leftNames / rightNames listed top → bottom. funcs maps pin name → function tag for Smart Connect.
function moduleFP(o) {
  const pads = [];
  const L = o.left || [], R = o.right || [];
  const pitch = o.pitch || P, sp = o.rowSpacing;
  const nL = L.length, nR = R.length;
  const oy = o.pinOffsetY || 0;
  let k = 1;
  L.forEach((nm, i) => pads.push(tht(k++, -sp / 2, oy - (nL - 1) * pitch / 2 + i * pitch, { sq: i === 0, name: nm, func: (o.funcs || {})[nm] || "" })));
  for (let i = nR - 1; i >= 0; i--) pads.push(tht(k++, sp / 2, oy - (nR - 1) * pitch / 2 + i * pitch, { name: R[i], func: (o.funcs || {})[R[i]] || "" }));
  return addFP({ id: o.id, name: o.name, cat: o.cat || "Microcontrollers", prefix: o.prefix || "U", value: o.value || o.name, desc: o.desc, tags: o.tags, verify: o.verify, pads, holes: o.holes || [], body: { x: 0, y: 0, w: o.w, h: o.h, col: o.col || "#0d3b66", module: true, usb: o.usb }, pinLabels: true, mcu: o.mcu !== false && (o.cat || "Microcontrollers") === "Microcontrollers", gpio: o.gpio });
}
const F_I2C = (sda, scl) => ({ [sda]: "SDA", [scl]: "SCL" });
const F_SPI = (mosi, miso, sck, cs) => ({ [mosi]: "MOSI", [miso]: "MISO", [sck]: "SCK", [cs]: "CS" });
const F_UART = (tx, rx) => ({ [tx]: "TX", [rx]: "RX" });

const PICO_L = ["GP0", "GP1", "GND", "GP2", "GP3", "GP4", "GP5", "GND", "GP6", "GP7", "GP8", "GP9", "GND", "GP10", "GP11", "GP12", "GP13", "GND", "GP14", "GP15"];
const PICO_R = ["VBUS", "VSYS", "GND", "3V3_EN", "3V3", "ADC_VREF", "GP28", "GND", "GP27", "GP26", "RUN", "GP22", "GND", "GP21", "GP20", "GP19", "GP18", "GND", "GP17", "GP16"];
const PICO_F = Object.assign({}, F_UART("GP0", "GP1"), F_I2C("GP4", "GP5"), F_SPI("GP19", "GP16", "GP18", "GP17"), { GP26: "ADC", GP27: "ADC", GP28: "ADC", VBUS: "PWR5", VSYS: "VIN", "3V3": "PWR3" });
for (const [id, nm] of [["pico", "Raspberry Pi Pico"], ["picow", "Raspberry Pi Pico W"], ["pico2", "Raspberry Pi Pico 2"], ["pico2w", "Raspberry Pi Pico 2 W"]])
  moduleFP({ id, name: nm, value: nm.replace("Raspberry Pi ", ""), desc: "RP2040/RP2350 board, 3.3V logic. Solder directly or via 2x 1x20 sockets. USB at top.", tags: "rp2040 rp2350 pico raspberry micropython", left: PICO_L, right: PICO_R, rowSpacing: 17.78, w: 21, h: 51, col: "#0f4f2c", funcs: PICO_F, usb: "top" });
moduleFP({ id: "rp2040zero", name: "Waveshare RP2040-Zero", value: "RP2040-Zero", desc: "Tiny RP2040 board (edge pins only). USB at top.", tags: "rp2040 waveshare zero", verify: true, left: ["5V", "GND", "3V3", "GP29", "GP28", "GP27", "GP26", "GP15", "GP14"], right: ["GP0", "GP1", "GP2", "GP3", "GP4", "GP5", "GP6", "GP7", "GP8"], rowSpacing: 15.24, w: 18, h: 23.5, col: "#111", funcs: Object.assign({}, F_UART("GP0", "GP1"), F_I2C("GP4", "GP5"), F_SPI("GP3", "GP4", "GP2", "GP5"), { "5V": "PWR5", "3V3": "PWR3" }), usb: "top" });
const NANO_L = ["D13", "3V3", "AREF", "A0", "A1", "A2", "A3", "A4", "A5", "A6", "A7", "5V", "RST", "GND", "VIN"];
const NANO_R = ["D12", "D11", "D10", "D9", "D8", "D7", "D6", "D5", "D4", "D3", "D2", "GND", "RST", "RX0", "TX1"];
const NANO_F = Object.assign({}, F_UART("TX1", "RX0"), F_I2C("A4", "A5"), F_SPI("D11", "D12", "D13", "D10"), { "5V": "PWR5", "3V3": "PWR3", VIN: "VIN", A0: "ADC", A1: "ADC", A2: "ADC", A3: "ADC", A6: "ADC", A7: "ADC" });
moduleFP({ id: "nano", name: "Arduino Nano", value: "Arduino Nano", desc: "ATmega328P board, 5V logic. USB at top.", tags: "arduino nano atmega328p avr", left: NANO_L, right: NANO_R, rowSpacing: 15.24, w: 18, h: 45, col: "#1b5fa8", funcs: NANO_F, usb: "top" });
moduleFP({ id: "nanoevery", name: "Arduino Nano Every / Nano 33 / ESP32", value: "Arduino Nano (newer)", desc: "Same footprint as the classic Nano. Check voltage: 33/ESP32 are 3.3V!", tags: "arduino nano every 33 iot ble esp32", left: NANO_L, right: NANO_R, rowSpacing: 15.24, w: 18, h: 45, col: "#0f7c84", funcs: NANO_F, usb: "top" });
moduleFP({ id: "promicro", name: "Arduino Pro Micro (ATmega32U4)", value: "Pro Micro", desc: "Native-USB Arduino, great for keyboards & HID. USB at top.", tags: "arduino pro micro 32u4 keyboard hid leonardo rp2040", left: ["TX0", "RX1", "GND", "GND", "D2", "D3", "D4", "D5", "D6", "D7", "D8", "D9"], right: ["RAW", "GND", "RST", "VCC", "A3", "A2", "A1", "A0", "D15", "D14", "D16", "D10"], rowSpacing: 15.24, w: 18, h: 33, col: "#a8123a", funcs: Object.assign({}, F_UART("TX0", "RX1"), F_I2C("D2", "D3"), F_SPI("D16", "D14", "D15", "D10"), { VCC: "PWR5", RAW: "VIN" }), usb: "top" });
moduleFP({ id: "promini", name: "Arduino Pro Mini", value: "Pro Mini", desc: "Small ATmega328P board (3.3V or 5V version), side pins only", tags: "arduino pro mini atmega328p", left: ["TXO", "RXI", "RST", "GND", "D2", "D3", "D4", "D5", "D6", "D7", "D8", "D9"], right: ["RAW", "GND", "RST", "VCC", "A3", "A2", "A1", "A0", "D13", "D12", "D11", "D10"], rowSpacing: 15.24, w: 18, h: 33, col: "#a8123a", funcs: Object.assign({}, F_UART("TXO", "RXI"), F_SPI("D11", "D12", "D13", "D10"), { VCC: "PWR5", RAW: "VIN" }) });
const ESP30_L = ["EN", "VP", "VN", "D34", "D35", "D32", "D33", "D25", "D26", "D27", "D14", "D12", "D13", "GND", "VIN"];
const ESP30_R = ["D23", "D22", "TX0", "RX0", "D21", "D19", "D18", "D5", "TX2", "RX2", "D4", "D2", "D15", "GND", "3V3"];
moduleFP({ id: "esp32devkit30", name: "ESP32 DevKit V1 (30-pin)", value: "ESP32 DevKit", desc: "WiFi + Bluetooth, 3.3V logic. Wide board (1in rows) — use 2 sockets. USB at bottom.", tags: "esp32 wroom devkit wifi bluetooth doit", left: ESP30_L, right: ESP30_R, rowSpacing: 25.4, w: 28.3, h: 51.5, col: "#151515", funcs: Object.assign({}, F_UART("TX2", "RX2"), F_I2C("D21", "D22"), F_SPI("D23", "D19", "D18", "D5"), { VIN: "PWR5", "3V3": "PWR3", D32: "ADC", D33: "ADC", D34: "ADC", D35: "ADC", VP: "ADC", VN: "ADC" }), gpio: ["D4", "D13", "D14", "D15", "D16", "D17", "D25", "D26", "D27", "D32", "D33", "D12", "D2"], usb: "bottom" });
moduleFP({ id: "esp32devkitc38", name: "ESP32 DevKitC (38-pin)", value: "ESP32 DevKitC", desc: "Espressif DevKitC v4. Some clones have different row spacing — measure yours!", tags: "esp32 devkitc wroom 38", verify: true, left: ["3V3", "EN", "VP", "VN", "IO34", "IO35", "IO32", "IO33", "IO25", "IO26", "IO27", "IO14", "IO12", "GND", "IO13", "SD2", "SD3", "CMD", "5V"], right: ["GND", "IO23", "IO22", "TX", "RX", "IO21", "GND", "IO19", "IO18", "IO5", "IO17", "IO16", "IO4", "IO0", "IO2", "IO15", "SD1", "SD0", "CLK"], rowSpacing: 25.4, w: 28, h: 54.4, col: "#151515", funcs: Object.assign({}, F_I2C("IO21", "IO22"), F_SPI("IO23", "IO19", "IO18", "IO5"), F_UART("IO17", "IO16"), { "5V": "PWR5", "3V3": "PWR3" }), gpio: ["IO4", "IO13", "IO14", "IO25", "IO26", "IO27", "IO32", "IO33", "IO15", "IO2", "IO12"], usb: "bottom" });
moduleFP({ id: "nodemcu", name: "NodeMCU ESP8266 v2 (Amica)", value: "NodeMCU", desc: "ESP8266 WiFi board, 3.3V logic, 0.9in rows. USB at bottom.", tags: "esp8266 nodemcu wifi amica", left: ["A0", "RSV", "RSV", "SD3", "SD2", "SD1", "CMD", "SD0", "CLK", "GND", "3V3", "EN", "RST", "GND", "VIN"], right: ["D0", "D1", "D2", "D3", "D4", "3V3", "GND", "D5", "D6", "D7", "D8", "RX", "TX", "GND", "3V3"], rowSpacing: 22.86, w: 26, h: 48.5, col: "#151515", funcs: Object.assign({}, F_I2C("D2", "D1"), F_SPI("D7", "D6", "D5", "D8"), F_UART("TX", "RX"), { VIN: "PWR5", "3V3": "PWR3", A0: "ADC" }), usb: "bottom" });
moduleFP({ id: "d1mini", name: "Wemos D1 Mini (ESP8266/ESP32)", value: "D1 Mini", desc: "Tiny WiFi board, 3.3V logic. USB at bottom.", tags: "wemos lolin d1 mini esp8266 esp32 s2", left: ["RST", "A0", "D0", "D5", "D6", "D7", "D8", "3V3"], right: ["TX", "RX", "D1", "D2", "D3", "D4", "GND", "5V"], rowSpacing: 22.86, w: 25.6, h: 34.2, col: "#1d6fb8", funcs: Object.assign({}, F_I2C("D2", "D1"), F_SPI("D7", "D6", "D5", "D8"), F_UART("TX", "RX"), { "5V": "PWR5", "3V3": "PWR3", A0: "ADC" }), usb: "bottom" });
moduleFP({ id: "esp32c3sm", name: "ESP32-C3 SuperMini", value: "ESP32-C3 SuperMini", desc: "Thumb-sized WiFi/BLE board, 3.3V logic. USB at top.", tags: "esp32 c3 supermini tiny", verify: true, left: ["5V", "GND", "3V3", "GPIO4", "GPIO3", "GPIO2", "GPIO1", "GPIO0"], right: ["GPIO5", "GPIO6", "GPIO7", "GPIO8", "GPIO9", "GPIO10", "GPIO20", "GPIO21"], rowSpacing: 15.24, w: 18, h: 22.5, col: "#151515", funcs: Object.assign({}, F_I2C("GPIO8", "GPIO9"), F_SPI("GPIO6", "GPIO5", "GPIO4", "GPIO7"), F_UART("GPIO21", "GPIO20"), { "5V": "PWR5", "3V3": "PWR3" }), usb: "top" });
moduleFP({ id: "xiao", name: "Seeed Studio XIAO (all versions)", value: "XIAO", desc: "Tiny board: SAMD21 / RP2040 / ESP32-C3/S3 / nRF52840. 3.3V. USB at top.", tags: "seeed xiao samd21 rp2040 esp32c3 esp32s3 nrf52840", left: ["D0", "D1", "D2", "D3", "D4", "D5", "D6"], right: ["5V", "GND", "3V3", "D10", "D9", "D8", "D7"], rowSpacing: 15.24, w: 17.8, h: 21, col: "#222", funcs: Object.assign({}, F_I2C("D4", "D5"), F_SPI("D10", "D9", "D8", "D3"), F_UART("D6", "D7"), { "5V": "PWR5", "3V3": "PWR3", D0: "ADC", D1: "ADC", D2: "ADC" }), usb: "top" });
moduleFP({ id: "teensy40", name: "Teensy 4.0", value: "Teensy 4.0", desc: "600MHz ARM board, 3.3V logic (NOT 5V tolerant). USB at top.", tags: "teensy pjrc arm imxrt", left: ["GND", "0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12"], right: ["VIN", "GND", "3.3V", "23", "22", "21", "20", "19", "18", "17", "16", "15", "14", "13"], rowSpacing: 15.24, w: 17.8, h: 35.6, col: "#1d7a3a", funcs: Object.assign({}, F_UART("1", "0"), F_I2C("18", "19"), F_SPI("11", "12", "13", "10"), { VIN: "PWR5", "3.3V": "PWR3" }), usb: "top" });
moduleFP({ id: "bluepill", name: "STM32 Blue Pill (STM32F103)", value: "Blue Pill", desc: "Cheap 72MHz ARM board, 3.3V logic. USB at top.", tags: "stm32 bluepill f103 arm", left: ["B12", "B13", "B14", "B15", "A8", "A9", "A10", "A11", "A12", "A15", "B3", "B4", "B5", "B6", "B7", "B8", "B9", "5V", "GND", "3V3"], right: ["VB", "C13", "C14", "C15", "A0", "A1", "A2", "A3", "A4", "A5", "A6", "A7", "B0", "B1", "B10", "B11", "R", "3V3", "GND", "GND"], rowSpacing: 15.24, w: 22.9, h: 53, col: "#1846a8", funcs: Object.assign({}, F_UART("A9", "A10"), F_I2C("B7", "B6"), F_SPI("A7", "A6", "A5", "A4"), { "5V": "PWR5", "3V3": "PWR3", A0: "ADC", A1: "ADC" }), usb: "top" });
addFP({ id: "esp01", name: "ESP-01 / ESP-01S (2x4)", cat: "Microcontrollers", prefix: "U", value: "ESP-01", desc: "Tiny ESP8266 module. 3.3V ONLY! Pins at one end.", tags: "esp01 esp8266", pads: [tht(1, -3.81, -1.27, { sq: true, name: "GND" }), tht(2, -1.27, -1.27, { name: "IO2" }), tht(3, 1.27, -1.27, { name: "IO0" }), tht(4, 3.81, -1.27, { name: "RX", func: "RX" }), tht(5, -3.81, 1.27, { name: "TX", func: "TX" }), tht(6, -1.27, 1.27, { name: "EN" }), tht(7, 1.27, 1.27, { name: "RST" }), tht(8, 3.81, 1.27, { name: "3V3", func: "PWR3" })], body: { x: 0, y: -8, w: 14.4, h: 24.7, col: "#151515", module: true }, pinLabels: true, mcu: true });

// Raspberry Pi 40-pin header
const PI40 = ["3V3", "5V", "GPIO2", "5V", "GPIO3", "GND", "GPIO4", "GPIO14", "GND", "GPIO15", "GPIO17", "GPIO18", "GPIO27", "GND", "GPIO22", "GPIO23", "3V3", "GPIO24", "GPIO10", "GND", "GPIO9", "GPIO25", "GPIO11", "GPIO8", "GND", "GPIO7", "ID_SD", "ID_SC", "GPIO5", "GND", "GPIO6", "GPIO12", "GPIO13", "GND", "GPIO19", "GPIO16", "GPIO26", "GPIO20", "GND", "GPIO21"];
const PI_F = Object.assign({}, F_I2C("GPIO2", "GPIO3"), F_SPI("GPIO10", "GPIO9", "GPIO11", "GPIO8"), F_UART("GPIO14", "GPIO15"), { "5V": "PWR5", "3V3": "PWR3" });
function pi40Pads() { const p = []; for (let i = 0; i < 40; i++) { const col = Math.floor(i / 2), odd = i % 2 === 0; p.push(tht(i + 1, -24.13 + col * P, odd ? 1.27 : -1.27, { sq: i === 0, name: PI40[i], func: PI_F[PI40[i]] || "" })); } return p; }
addFP({ id: "pi40", name: "Raspberry Pi 40-pin GPIO header", cat: "Raspberry Pi", prefix: "J", value: "RPi GPIO", desc: "2x20 socket that plugs onto any Pi (Zero/3/4/5). 3.3V logic! Use the Pi HAT template for the right position.", tags: "raspberry pi gpio hat header 40 zero pi4 pi5", pads: pi40Pads(), body: { x: 0, y: 0, w: 51.3, h: 5.3, col: "#222" }, pinLabels: true, mcu: true, gpio: ["GPIO4", "GPIO17", "GPIO27", "GPIO22", "GPIO23", "GPIO24", "GPIO25", "GPIO5", "GPIO6", "GPIO12", "GPIO13", "GPIO16", "GPIO19", "GPIO20", "GPIO21", "GPIO26", "GPIO18"] });
addFP({ id: "pi_hat_holes", name: "Raspberry Pi HAT mounting holes (58x49)", cat: "Raspberry Pi", prefix: "H", value: "HAT holes", desc: "4x M2.5 holes in the Pi 3/4/5 pattern", tags: "raspberry pi mounting holes m2.5", pads: [], holes: [[-29, -24.5], [29, -24.5], [-29, 24.5], [29, 24.5]].map(([x, y]) => ({ x, y, d: 2.75 })), silk: "holes" });
addFP({ id: "pi_zero_holes", name: "Raspberry Pi Zero mounting holes (58x23)", cat: "Raspberry Pi", prefix: "H", value: "Zero holes", desc: "4x M2.5 holes in the Pi Zero pattern", tags: "raspberry pi zero mounting holes", pads: [], holes: [[-29, -11.5], [29, -11.5], [-29, 11.5], [29, 11.5]].map(([x, y]) => ({ x, y, d: 2.75 })), silk: "holes" });
addFP({ id: "pi_fan", name: "Pi fan connector (5V/GND)", cat: "Raspberry Pi", prefix: "J", value: "Fan", tags: "fan 5v raspberry", pads: headerPads(2, 1, P, ["5V", "GND"], 1.7, 1.0) });

// ---------------- sensor & add-on modules (1xN headers) ----------------
function stripModule(id, name, pins, o = {}) {
  const pitch = o.pitch || P;
  const pads = pins.map((nm, i) => tht(i + 1, -(pins.length - 1) * pitch / 2 + i * pitch, 0, { sq: i === 0, name: nm, func: o.funcs ? o.funcs[nm] || "" : autoFunc(nm) }));
  const w = o.w || pins.length * pitch + 2, h = o.h || 12;
  return addFP({ id, name, cat: o.cat || "Sensors & Modules", prefix: o.prefix || "M", value: o.value || name.split(" (")[0], desc: o.desc, tags: o.tags, verify: o.verify, pads, holes: o.holes || [], body: { x: 0, y: (o.pinsAt === "bottom" ? -(h / 2 - 1.6) : (h / 2 - 1.6)), w, h, col: o.col || "#1d4f91", module: true }, pinLabels: true });
}
function autoFunc(nm) {
  const n = nm.toUpperCase().replace(/[^A-Z0-9+]/g, "");
  if (/^(GND|VSS|G|0V|-)$/.test(n)) return "GND";
  if (/^(VCC|VIN|V\+|\+|VDD|5V|\+5V|3V3|33V|VBAT|\+V|VS)$/.test(n)) return "PWR";
  if (/^(SDA|SDI)$/.test(n)) return "SDA";
  if (/^(SCL|SCK)$/.test(n) && nm.toUpperCase() === "SCL") return "SCL";
  if (/^(MOSI|SI|DIN|DI)$/.test(n)) return n === "DIN" ? "SIG" : "MOSI";
  if (/^(MISO|SO|DO)$/.test(n)) return "MISO";
  if (/^(SCK|CLK|SCLK)$/.test(n)) return "SCK";
  if (/^(CS|SS|CSN|NSS|SDACS|CE0)$/.test(n)) return "CS";
  if (/^(TX|TXD|TXO)$/.test(n)) return "TX";
  if (/^(RX|RXD|RXI)$/.test(n)) return "RX";
  if (/^(AO|AOUT|A0|VRX|VRY|OUTA|ANALOG)$/.test(n)) return "ADC";
  if (n === "NC") return "";
  return "SIG";
}
stripModule("bme280", "BME280 / BMP280 (I2C, 4-pin)", ["VIN", "GND", "SCL", "SDA"], { desc: "Temperature, humidity & pressure sensor", tags: "temperature humidity pressure barometer sensor i2c bmp280", w: 13, h: 10.5, col: "#5b2a86" });
stripModule("bme280_6", "BME280 / BMP280 (6-pin)", ["VCC", "GND", "SCL", "SDA", "CSB", "SDO"], { desc: "6-pin version, I2C or SPI", tags: "temperature pressure sensor", w: 15.5, h: 11.5, col: "#5b2a86" });
stripModule("oled096", "OLED display 0.96in (I2C, GND-VCC-SCL-SDA)", ["GND", "VCC", "SCL", "SDA"], { desc: "SSD1306 128x64 screen. Some boards swap GND/VCC — check yours!", tags: "oled display screen ssd1306 i2c 128x64", w: 27.3, h: 27.8, col: "#123", pinsAt: "top" });
stripModule("oled13", "OLED display 1.3in (I2C, VCC-GND-SCL-SDA)", ["VCC", "GND", "SCL", "SDA"], { desc: "SH1106 128x64 screen", tags: "oled display sh1106 i2c", w: 35.4, h: 33.5, col: "#123", pinsAt: "top" });
stripModule("oled096spi", "OLED display 0.96in (SPI, 7-pin)", ["GND", "VCC", "D0", "D1", "RES", "DC", "CS"], { funcs: { GND: "GND", VCC: "PWR", D0: "SCK", D1: "MOSI", RES: "SIG", DC: "SIG", CS: "CS" }, desc: "SPI OLED screen", tags: "oled spi display ssd1306", w: 27.3, h: 27.8, col: "#123", pinsAt: "top" });
stripModule("tft_st7735", "TFT display 1.8in ST7735 (SPI, 8-pin)", ["GND", "VCC", "SCL", "SDA", "RES", "DC", "CS", "BL"], { funcs: { GND: "GND", VCC: "PWR", SCL: "SCK", SDA: "MOSI", RES: "SIG", DC: "SIG", CS: "CS", BL: "PWR" }, desc: "Colour TFT. Pin labels say SCL/SDA but it's SPI!", tags: "tft lcd colour display spi st7735 st7789", w: 34, h: 58, col: "#8a1b1b", pinsAt: "top" });
stripModule("lcd1602i2c", "LCD 16x2 with I2C backpack (4-pin)", ["GND", "VCC", "SDA", "SCL"], { desc: "Character LCD with PCF8574 backpack", tags: "lcd 1602 2004 character display i2c pcf8574", w: 80, h: 36, col: "#1b5e20", pinsAt: "top" });
stripModule("hcsr04", "HC-SR04 ultrasonic distance sensor", ["VCC", "TRIG", "ECHO", "GND"], { desc: "5V distance sensor. ECHO is 5V: use a divider on 3.3V boards!", tags: "ultrasonic distance sonar range", w: 45, h: 20, col: "#1d4f91", pinsAt: "bottom" });
stripModule("dht22m", "DHT11 / DHT22 module (3-pin)", ["VCC", "DATA", "GND"], { desc: "Temperature & humidity sensor on a board", tags: "dht22 dht11 am2302 temperature humidity", w: 15, h: 25, col: "#eee", pinsAt: "bottom" });
stripModule("dht22", "DHT22 bare sensor (4-pin)", ["VCC", "DATA", "NC", "GND"], { desc: "Needs 10k pull-up DATA to VCC", tags: "dht22 temperature humidity", w: 15.1, h: 25, col: "#eee", pinsAt: "bottom", prefix: "U" });
stripModule("mpu6050", "MPU-6050 accelerometer/gyro (GY-521)", ["VCC", "GND", "SCL", "SDA", "XDA", "XCL", "AD0", "INT"], { desc: "6-axis motion sensor", tags: "accelerometer gyro imu motion i2c gy521", w: 21, h: 16, col: "#1d4f91" });
stripModule("ads1115", "ADS1115 16-bit ADC (I2C)", ["VDD", "GND", "SCL", "SDA", "ADDR", "ALRT", "A0", "A1", "A2", "A3"], { funcs: { VDD: "PWR", GND: "GND", SCL: "SCL", SDA: "SDA", ADDR: "", ALRT: "", A0: "", A1: "", A2: "", A3: "" }, desc: "4-channel precise analog input", tags: "adc analog i2c ads1115", w: 28, h: 17.5, col: "#1d4f91" });
stripModule("ina219", "INA219 current sensor (I2C)", ["VCC", "GND", "SCL", "SDA", "VIN-", "VIN+"], { funcs: { VCC: "PWR", GND: "GND", SCL: "SCL", SDA: "SDA" }, desc: "Measures voltage & current — great for a DIY power meter", tags: "current power sensor i2c ina219 ina226", w: 25.5, h: 22.3, col: "#5b2a86" });
stripModule("ds3231", "DS3231 real-time clock (6-pin)", ["32K", "SQW", "SCL", "SDA", "VCC", "GND"], { funcs: { SCL: "SCL", SDA: "SDA", VCC: "PWR", GND: "GND" }, desc: "Keeps time with a coin cell", tags: "rtc clock time ds3231 ds1307", w: 38, h: 22, col: "#1d4f91" });
stripModule("vl53l0x", "VL53L0X laser distance (I2C)", ["VIN", "GND", "SCL", "SDA", "GPIO1", "XSHUT"], { funcs: { VIN: "PWR", GND: "GND", SCL: "SCL", SDA: "SDA" }, desc: "Time-of-flight distance sensor", tags: "tof laser distance i2c", w: 25, h: 13, col: "#5b2a86" });
stripModule("bh1750", "BH1750 light sensor (I2C)", ["VCC", "GND", "SCL", "SDA", "ADDR"], { funcs: { VCC: "PWR", GND: "GND", SCL: "SCL", SDA: "SDA", ADDR: "" }, desc: "Ambient light (lux)", tags: "light lux sensor i2c", w: 18.5, h: 13.9, col: "#1d4f91" });
stripModule("aht20", "AHT20 / SHT30 temp-humidity (I2C)", ["VIN", "GND", "SCL", "SDA"], { desc: "Accurate temperature & humidity", tags: "temperature humidity aht20 sht30 sht31 i2c", w: 12, h: 12, col: "#1d4f91" });
stripModule("nrf24", "nRF24L01 radio (2x4)", ["GND", "VCC", "CE", "CSN", "SCK", "MOSI", "MISO", "IRQ"], { desc: "2.4GHz radio. 3.3V ONLY. Add 10uF cap at VCC", tags: "radio wireless 2.4ghz nrf24", w: 29, h: 15 });
// nRF24 is actually 2x4: rebuild pads
(function () { const f = LIB_BY_ID.nrf24; const nm = ["GND", "VCC", "CE", "CSN", "SCK", "MOSI", "MISO", "IRQ"]; f.pads = nm.map((n, i) => tht(i + 1, -3.81 + Math.floor(i / 2) * P, (i % 2 ? -1.27 : 1.27), { sq: i === 0, name: n, func: autoFunc(n) })); f.body = { x: 9, y: 0, w: 29, h: 15, col: "#1b1b1b", module: true }; })();
stripModule("sdcard", "MicroSD card module (SPI, 6-pin)", ["GND", "VCC", "MISO", "MOSI", "SCK", "CS"], { desc: "Store logs on an SD card", tags: "sd card microsd storage spi", w: 20.5, h: 42, col: "#1d4f91", pinsAt: "top" });
stripModule("rc522", "RC522 RFID reader (8-pin)", ["SDA", "SCK", "MOSI", "MISO", "IRQ", "GND", "RST", "3V3"], { funcs: { SDA: "CS", SCK: "SCK", MOSI: "MOSI", MISO: "MISO", IRQ: "", GND: "GND", RST: "SIG", "3V3": "PWR" }, desc: "13.56MHz RFID / NFC card reader (SPI). 3.3V!", tags: "rfid nfc card reader spi", w: 40, h: 60, col: "#1d4f91", pinsAt: "bottom" });
stripModule("hc05", "HC-05 / HC-06 Bluetooth (6-pin)", ["EN", "VCC", "GND", "TXD", "RXD", "STATE"], { funcs: { EN: "", VCC: "PWR", GND: "GND", TXD: "TX", RXD: "RX", STATE: "" }, desc: "Bluetooth serial. RXD is 3.3V!", tags: "bluetooth serial uart hc05 hc06", w: 16, h: 37, col: "#1d4f91", pinsAt: "bottom" });
stripModule("gps6m", "GPS NEO-6M module (4-pin)", ["VCC", "RX", "TX", "GND"], { desc: "GPS receiver (UART)", tags: "gps gnss neo6m neo-m8n uart", w: 25, h: 35, col: "#1d4f91", pinsAt: "bottom" });
stripModule("pir", "PIR motion sensor HC-SR501", ["VCC", "OUT", "GND"], { desc: "Detects people moving", tags: "pir motion presence hc-sr501", w: 32, h: 24, col: "#1d7a3a" });
stripModule("relay1", "Relay module 1-channel", ["VCC", "GND", "IN"], { desc: "Switch big loads. Mains is dangerous — keep it low voltage!", tags: "relay module switch", w: 26, h: 50, col: "#1d4f91", pinsAt: "top" });
stripModule("relay2", "Relay module 2-channel", ["GND", "IN1", "IN2", "VCC"], { desc: "Two relays", tags: "relay module", w: 39, h: 51, col: "#1d4f91", pinsAt: "top" });
stripModule("servo", "Servo connector (GND-V+-SIG)", ["GND", "V+", "SIG"], { cat: "Connectors", prefix: "J", desc: "Plug an SG90/MG996R servo here. Brown=GND, Red=V+, Orange=SIG", tags: "servo sg90 mg996r rc pwm", w: 8.6, h: 3.6, col: "#222" });
stripModule("ws2812strip", "LED strip connector (5V-DIN-GND)", ["5V", "DIN", "GND"], { cat: "Connectors", prefix: "J", desc: "For WS2812/NeoPixel strips. Add 330R on DIN + 1000uF cap", tags: "neopixel led strip ws2812 addressable", w: 8.6, h: 3.6, col: "#222" });
stripModule("encoder", "Rotary encoder module KY-040", ["CLK", "DT", "SW", "+", "GND"], { funcs: { CLK: "SIG", DT: "SIG", SW: "SIG", "+": "PWR", GND: "GND" }, desc: "Knob that clicks & turns endlessly", tags: "rotary encoder knob ky040", w: 19, h: 26, col: "#1d4f91" });
stripModule("joystick", "Joystick module (5-pin)", ["GND", "+5V", "VRX", "VRY", "SW"], { desc: "2-axis analog thumb stick", tags: "joystick thumbstick analog ky023", w: 26, h: 34, col: "#1d4f91" });
stripModule("keypad4x4", "Membrane keypad 4x4 (8-pin)", ["R1", "R2", "R3", "R4", "C1", "C2", "C3", "C4"], { desc: "16-key keypad ribbon", tags: "keypad matrix buttons", w: 22, h: 6, col: "#222", prefix: "J", cat: "Connectors" });
stripModule("max7219", "MAX7219 LED matrix (5-pin)", ["VCC", "GND", "DIN", "CS", "CLK"], { funcs: { VCC: "PWR", GND: "GND", DIN: "MOSI", CS: "CS", CLK: "SCK" }, desc: "8x8 LED matrix / 7-seg driver (SPI)", tags: "max7219 led matrix 7segment spi", w: 32, h: 32, col: "#8a1b1b" });
stripModule("hx711", "HX711 load-cell amplifier", ["GND", "DT", "SCK", "VCC"], { funcs: { GND: "GND", DT: "SIG", SCK: "SIG", VCC: "PWR" }, desc: "For weighing scales", tags: "load cell scale weight hx711", w: 34, h: 20, col: "#1d7a3a" });
stripModule("inmp441", "INMP441 I2S microphone", ["L/R", "WS", "SCK", "SD", "VDD", "GND"], { funcs: { "L/R": "GND", WS: "SIG", SCK: "SIG", SD: "SIG", VDD: "PWR", GND: "GND" }, desc: "Digital microphone (I2S) for ESP32", tags: "microphone mic i2s audio", w: 14, h: 14, col: "#5b2a86", verify: true });
stripModule("max98357", "MAX98357A I2S amplifier", ["LRC", "BCLK", "DIN", "GAIN", "SD", "GND", "VIN"], { funcs: { LRC: "SIG", BCLK: "SIG", DIN: "SIG", GAIN: "", SD: "", GND: "GND", VIN: "PWR" }, desc: "3W speaker amp (I2S)", tags: "amplifier speaker audio i2s", w: 18, h: 18, col: "#5b2a86" });
stripModule("soil", "Soil moisture sensor (3-pin)", ["AOUT", "GND", "VCC"], { desc: "Capacitive soil moisture probe", tags: "soil moisture plant water analog", w: 23, h: 10, col: "#222" });
stripModule("ldrmod", "Light sensor module LDR (3-pin)", ["VCC", "GND", "AO"], { desc: "Light-dependent resistor board", tags: "ldr light photoresistor", w: 15, h: 30, col: "#1d4f91" });
stripModule("mcp2515", "MCP2515 CAN bus module (7-pin)", ["INT", "SCK", "SI", "SO", "CS", "GND", "VCC"], { funcs: { INT: "SIG", SCK: "SCK", SI: "MOSI", SO: "MISO", CS: "CS", GND: "GND", VCC: "PWR" }, desc: "CAN bus for cars & robots (SPI)", tags: "can bus mcp2515 automotive spi", w: 44, h: 28, col: "#1d4f91", pinsAt: "top" });
stripModule("acs712", "ACS712 current sensor (3-pin)", ["VCC", "OUT", "GND"], { funcs: { VCC: "PWR", OUT: "ADC", GND: "GND" }, desc: "Analog current sensor", tags: "current sensor acs712 hall", w: 31, h: 13, col: "#8a1b1b" });
stripModule("ir_rx", "IR receiver VS1838B (3-pin)", ["OUT", "GND", "VCC"], { desc: "TV remote receiver", tags: "infrared ir remote receiver vs1838 tsop", w: 7, h: 6, col: "#222", prefix: "U" });
stripModule("dfplayer", "DFPlayer Mini MP3 (16-pin, use left row)", ["VCC", "RX", "TX", "DAC_R", "DAC_L", "SPK1", "GND", "SPK2"], { funcs: { VCC: "PWR", RX: "RX", TX: "TX", GND: "GND" }, desc: "MP3 player — this footprint uses one 8-pin side; add a 1x8 header for the other", tags: "mp3 audio player dfplayer", w: 21, h: 21, col: "#1d4f91", verify: true });
// Stepper driver carrier 2x8 (0.5in rows)
moduleFP({ id: "stepstick", name: "Stepper driver A4988 / DRV8825 / TMC2208", cat: "Sensors & Modules", prefix: "M", value: "A4988", desc: "StepStick-format stepper driver. Check orientation of VMOT side! Add 100uF on VMOT.", tags: "stepper driver a4988 drv8825 tmc2208 tmc2209 stepstick 3d printer cnc", left: ["EN", "MS1", "MS2", "MS3", "RST", "SLP", "STEP", "DIR"], right: ["VMOT", "GND", "2B", "2A", "1A", "1B", "VDD", "GND"], rowSpacing: 12.7, w: 15.2, h: 20.3, col: "#1d7a3a", funcs: { STEP: "SIG", DIR: "SIG", EN: "SIG", VDD: "PWR", GND: "GND" }, mcu: false });
stripModule("mini360", "Buck converter (IN+ IN- OUT+ OUT-)", ["IN+", "IN-", "OUT+", "OUT-"], { cat: "Transistors & Power", funcs: {}, desc: "Use for LM2596 / MP1584 style buck boards wired with header pins", tags: "buck converter dc-dc step down lm2596 mp1584 mt3608 boost", w: 22, h: 17, col: "#1d4f91" });
stripModule("tp4056", "TP4056 LiPo charger (header version)", ["OUT+", "B+", "B-", "OUT-"], { cat: "Transistors & Power", funcs: {}, desc: "Charges single-cell LiPo/18650 from USB", tags: "lipo charger tp4056 18650 battery", w: 26, h: 17, col: "#1d4f91", verify: true });

// ---------------- mechanical ----------------
for (const [nm, d, pad] of [["M2", 2.2, 0], ["M2.5", 2.7, 0], ["M3", 3.2, 0], ["M4", 4.3, 0], ["M2 (plated)", 2.2, 4.0], ["M2.5 (plated)", 2.7, 5.0], ["M3 (plated)", 3.2, 6.0]]) {
  const id = "mh_" + nm.replace(/[^a-z0-9]/gi, "");
  if (pad) addFP({ id, name: `Mounting hole ${nm}`, cat: "Mechanical", prefix: "H", value: nm, desc: "Plated hole — can connect to GND", tags: "mounting hole screw standoff", pads: [{ n: "1", name: "MH", x: 0, y: 0, shape: "circle", w: pad, h: pad, drill: d }], silk: "none" });
  else addFP({ id, name: `Mounting hole ${nm}`, cat: "Mechanical", prefix: "H", value: nm, desc: "Plain (unplated) screw hole", tags: "mounting hole screw standoff npth", pads: [], holes: [{ x: 0, y: 0, d }], silk: "holes" });
}
addFP({ id: "fiducial", name: "Fiducial mark", cat: "Mechanical", prefix: "FID", value: "Fiducial", desc: "Alignment mark for pick-and-place assembly", tags: "fiducial assembly", pads: [smd(1, 0, 0, 1, 1, { shape: "circle" })], silk: "none" });
addFP({ id: "logo_space", name: "Keep-out area 10x10 (for logo)", cat: "Mechanical", prefix: "KO", value: "Keepout", desc: "A placeholder rectangle with no copper", tags: "keepout logo", pads: [], body: { x: 0, y: 0, w: 10, h: 10, col: "#0000" }, silk: "box" });

// ---------------- board templates ----------------
const TEMPLATES = [
  { id: "blank50", name: "Blank board 50 x 50 mm", desc: "A small square board to start from", board: { w: 50, h: 50, r: 2 } },
  { id: "blank100", name: "Blank board 100 x 80 mm", desc: "Fits most cheap 'up to 100mm' PCB deals", board: { w: 100, h: 80, r: 3 } },
  { id: "pihat", name: "Raspberry Pi HAT (65 x 56.5)", desc: "Official HAT size with GPIO header and holes in the right place", board: { w: 65, h: 56.5, r: 3 }, parts: [{ fp: "pi40", x: 32.5, y: 3.5 }, { fp: "pi_hat_holes", x: 32.5, y: 28 }] },
  { id: "piphat", name: "Raspberry Pi Zero pHAT (65 x 30)", desc: "Pi Zero sized add-on board", board: { w: 65, h: 30, r: 3 }, parts: [{ fp: "pi40", x: 32.5, y: 3.5 }, { fp: "pi_zero_holes", x: 32.5, y: 15 }] },
  { id: "picarrier", name: "Pico carrier board (70 x 60)", desc: "Pico in the middle with room for parts either side", board: { w: 70, h: 60, r: 3 }, parts: [{ fp: "pico", x: 35, y: 30 }, { fp: "mh_M3", x: 4, y: 4 }, { fp: "mh_M3", x: 66, y: 4 }, { fp: "mh_M3", x: 4, y: 56 }, { fp: "mh_M3", x: 66, y: 56 }] },
  { id: "nanocarrier", name: "Arduino Nano carrier (70 x 50)", desc: "Nano plus space for sensors", board: { w: 70, h: 50, r: 3 }, parts: [{ fp: "nano", x: 20, y: 25 }, { fp: "mh_M3", x: 4, y: 4 }, { fp: "mh_M3", x: 66, y: 4 }, { fp: "mh_M3", x: 4, y: 46 }, { fp: "mh_M3", x: 66, y: 46 }] },
  { id: "esp32carrier", name: "ESP32 DevKit carrier (80 x 60)", desc: "ESP32 30-pin with mounting holes", board: { w: 80, h: 60, r: 3 }, parts: [{ fp: "esp32devkit30", x: 22, y: 30 }, { fp: "mh_M3", x: 4, y: 4 }, { fp: "mh_M3", x: 76, y: 4 }, { fp: "mh_M3", x: 4, y: 56 }, { fp: "mh_M3", x: 76, y: 56 }] },
  { id: "d1shield", name: "Wemos D1 Mini shield (25.6 x 34.2)", desc: "Stacks on top of a D1 Mini", board: { w: 25.6, h: 34.2, r: 1 }, parts: [{ fp: "d1mini", x: 12.8, y: 17.1 }] },
  { id: "round40", name: "Round board 40 mm", desc: "Circular badge / LED ring", board: { w: 40, h: 40, round: true } },
];

const CATEGORIES = ["Microcontrollers", "Raspberry Pi", "Sensors & Modules", "Headers & Sockets", "Connectors", "Resistors", "Capacitors", "Diodes & LEDs", "Transistors & Power", "Switches", "Chips (ICs)", "Mechanical"];
