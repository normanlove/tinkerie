// Powerie — battery power budget planner inside Wirie — by nrmluvdev
"use strict";

const BATTERIES = [
  { id: "cr2032", chem: "coin", name: "CR2032 coin cell", v: 3.0, vmin: 2.0, cap: 220, max: 15, self: 0.1, usable: 85 },
  { id: "2aa", chem: "alk", name: "2× AA alkaline", v: 3.0, vmin: 1.8, cap: 2500, max: 1000, self: 0.2, usable: 85 },
  { id: "3aa", chem: "alk", name: "3× AA alkaline", v: 4.5, vmin: 2.7, cap: 2500, max: 1000, self: 0.2, usable: 85 },
  { id: "4aa", chem: "alk", name: "4× AA alkaline", v: 6.0, vmin: 3.6, cap: 2500, max: 1000, self: 0.2, usable: 85 },
  { id: "4nimh", chem: "nimh", name: "4× AA NiMH (low self-discharge)", v: 4.8, vmin: 4.0, cap: 2000, max: 4000, self: 1.5, usable: 90 },
  { id: "3aaa", chem: "alk", name: "3× AAA alkaline", v: 4.5, vmin: 2.7, cap: 1000, max: 500, self: 0.2, usable: 85 },
  { id: "9v", chem: "alk", name: "9V (PP3) alkaline", v: 9.0, vmin: 5.4, cap: 550, max: 300, self: 0.2, usable: 85 },
  { id: "18650", chem: "li", name: "18650 Li-ion cell", v: 3.7, vmin: 3.0, cap: 3000, max: 8000, self: 2, usable: 90 },
  { id: "2s18650", chem: "li", name: "2× 18650 in series (2S)", v: 7.4, vmin: 6.0, cap: 3000, max: 8000, self: 2, usable: 90 },
  { id: "lipo500", chem: "li", name: "LiPo 500mAh", v: 3.7, vmin: 3.3, cap: 500, max: 1000, self: 2, usable: 90 },
  { id: "lipo1000", chem: "li", name: "LiPo 1000mAh", v: 3.7, vmin: 3.3, cap: 1000, max: 2000, self: 2, usable: 90 },
  { id: "lipo2000", chem: "li", name: "LiPo 2000mAh", v: 3.7, vmin: 3.3, cap: 2000, max: 4000, self: 2, usable: 90 },
  { id: "bank", chem: "flat", name: "USB power bank 10,000mAh", v: 5.0, vmin: 5.0, cap: 6300, max: 2000, self: 2, usable: 90, note: "A 10,000mAh bank gives roughly 6,300mAh at 5V after conversion. Many banks switch off below ~50–100mA!" },
  { id: "custom", chem: "lin", name: "Custom…", v: 3.7, vmin: 3.0, cap: 1000, max: 1000, self: 2, usable: 85 },
];
const REGULATORS = [
  { id: "none", name: "None (powered straight from the battery)", type: "none", vout: 0, iq: 0, eff: 100, drop: 0, max: 99999 },
  { id: "ams1117", name: "AMS1117-3.3 (LDO)", type: "ldo", vout: 3.3, iq: 5000, drop: 1.1, max: 800, note: "Very common but wasteful: draws 5mA all the time, even when your project sleeps." },
  { id: "mcp1700", name: "MCP1700-3.3 (low-power LDO)", type: "ldo", vout: 3.3, iq: 1.6, drop: 0.18, max: 250 },
  { id: "ht7333", name: "HT7333 (low-power LDO)", type: "ldo", vout: 3.3, iq: 4, drop: 0.09, max: 250 },
  { id: "xc6206", name: "XC6206 3.3V (low-power LDO)", type: "ldo", vout: 3.3, iq: 1, drop: 0.25, max: 200 },
  { id: "me6211", name: "ME6211 3.3V (LDO)", type: "ldo", vout: 3.3, iq: 40, drop: 0.1, max: 500 },
  { id: "ap2112", name: "AP2112K-3.3 (LDO)", type: "ldo", vout: 3.3, iq: 55, drop: 0.25, max: 600 },
  { id: "l7805", name: "7805 (5V linear)", type: "ldo", vout: 5.0, iq: 5000, drop: 2.0, max: 1000, note: "Burns the extra voltage as heat — fine for mains adapters, poor for batteries." },
  { id: "mp1584", name: "MP1584 buck module", type: "buck", vout: 3.3, iq: 100, eff: 88, drop: 0.8, max: 3000 },
  { id: "lm2596", name: "LM2596 buck module", type: "buck", vout: 5.0, iq: 5000, eff: 80, drop: 1.5, max: 3000 },
  { id: "tps62160", name: "TPS62160 low-power buck", type: "buck", vout: 3.3, iq: 17, eff: 92, drop: 0.3, max: 1000 },
  { id: "mt3608", name: "MT3608 boost module", type: "boost", vout: 5.0, iq: 1000, eff: 88, drop: 0, max: 1000 },
  { id: "tps61023", name: "TPS61023 boost", type: "boost", vout: 5.0, iq: 20, eff: 90, drop: 0, max: 1500 },
  { id: "tps63020", name: "TPS63020 buck-boost", type: "buckboost", vout: 3.3, iq: 25, eff: 90, drop: 0, max: 2000 },
  { id: "custom", name: "Custom…", type: "ldo", vout: 3.3, iq: 50, eff: 85, drop: 0.3, max: 500 },
];
// [name, awake mA, sleep µA, default mode, note, footprint regex]
const LOADS = [
  ["— Microcontrollers & boards —"],
  ["ESP32 DevKit board (WiFi on)", 120, 10000, "sched", "Dev boards keep their USB chip, LED and regulator powered — deep sleep is ~10mA, not µA", /^esp32devkit/],
  ["ESP32 bare module (WiFi on)", 120, 10, "sched", "Deep sleep ~10µA on a well-designed board", /^esp32wroom$/],
  ["ESP32 (CPU only, WiFi off)", 30, 10, "sched", "", null],
  ["ESP32-C3 SuperMini (WiFi on)", 85, 45, "sched", "", /^esp32c3sm$/],
  ["ESP32-S3 (WiFi on)", 100, 10, "sched", "", /^esp32s3/],
  ["ESP32-CAM (camera + WiFi)", 180, 6000, "sched", "Peaks of ~300mA when the flash LED fires", /^esp32cam$/],
  ["ESP8266 D1 Mini / NodeMCU (WiFi on)", 75, 200, "sched", "Wire D0 to RST for deep-sleep wake-up", /^(d1mini|nodemcu)$/],
  ["ESP-12F / ESP-01 bare (WiFi on)", 75, 20, "sched", "", /^(esp12f|esp01)$/],
  ["Raspberry Pi Pico (running)", 25, 1300, "sched", "Dormant mode on the Pico board is ~1.3mA", /^(pico|pico2|pico_smd|rp2040zero)$/],
  ["Raspberry Pi Pico W (WiFi on)", 45, 1300, "sched", "", /^(picow|pico2w)$/],
  ["Arduino Nano / Uno board", 20, 6000, "sched", "The power LED, USB chip and regulator keep drawing ~6mA even in sleep", /^(nano|nanoevery|arduino_micro|promicro)$/],
  ["Arduino Pro Mini 3.3V (LED & regulator removed)", 4, 5, "sched", "The classic low-power hack", /^promini$/],
  ["ATmega328P bare chip (8MHz)", 4, 0.2, "sched", "Power-down sleep with brown-out detection off", /^atmega328p/],
  ["ATtiny85 bare chip (1MHz)", 1, 0.2, "sched", "", /^attiny85/],
  ["Seeed XIAO ESP32-C3 (WiFi on)", 80, 45, "sched", "", /^xiao$/],
  ["Teensy 4.0", 100, 100, "always", "Fast but power-hungry; deep sleep needs a library", /^teensy40$/],
  ["STM32 Blue/Black Pill", 25, 2000, "sched", "On-board power LED and regulator limit sleep current", /^(bluepill|blackpill)$/],
  ["Raspberry Pi Zero 2 W (idle)", 120, 0, "always", "A full Linux computer — expect hours, not days, on batteries", null],
  ["Raspberry Pi 4 (idle)", 600, 0, "always", "", null],
  ["— Sensors —"],
  ["BME280 / BMP280", 0.7, 0.1, "sched", "", /^bme280/],
  ["DHT11 / DHT22", 1.5, 50, "sched", "", /^dht/],
  ["DS18B20 temperature", 1.5, 1, "sched", "", /^ds18b20$/],
  ["AHT20 / SHT30", 0.6, 0.2, "sched", "", /^aht20/],
  ["BH1750 light", 0.12, 0.01, "sched", "", /^bh1750$/],
  ["VL53L0X distance", 19, 5, "sched", "", /^vl53l0x$/],
  ["MPU-6050 motion", 3.9, 5, "sched", "", /^mpu6050$/],
  ["HC-SR04 ultrasonic", 15, 2000, "sched", "Has no sleep mode — power it through a transistor to save energy", /^hcsr04$/],
  ["PIR motion sensor HC-SR501", 0.065, 65, "always", "Must stay on to detect motion", /^pir$/],
  ["Capacitive soil moisture", 5, 5000, "sched", "Power it from a GPIO pin so it's off between readings", /^soil$/],
  ["GPS NEO-6M (tracking)", 45, 0, "sched", "Takes 30s+ to get a fix from cold — budget wake time accordingly", /^gps/],
  ["INA219 current sensor", 1, 6, "sched", "", /^ina219$/],
  ["— Displays & outputs —"],
  ["OLED 0.96in SSD1306 (half the pixels on)", 12, 10, "sched", "Send the display-off command before sleeping", /^oled/],
  ["TFT 1.8in + backlight", 40, 1000, "sched", "", /^tft/],
  ["E-paper 2.9in", 8, 0.1, "sched", "Keeps showing the picture with zero power", /^epaper/],
  ["LCD 16x2 + I2C backpack (backlight on)", 25, 2000, "sched", "", /^lcd/],
  ["WS2812B / NeoPixel LED (each, mixed colour)", 20, 1000, "sched", "Each LED draws ~1mA even when OFF — cut their power when sleeping", /^(ws2812b|ws2812strip|sk6812)/],
  ["LED with resistor (~5mA)", 5, 0, "sched", "", /^(led3|led5|led_)/],
  ["Power-indicator LED (always on)", 2, 2000, "always", "Removing power LEDs is the easiest saving there is", null],
  ["Active buzzer", 30, 0, "pct", "", /^buzzer/],
  ["Servo SG90 (moving)", 150, 6000, "pct", "Holding position still draws a few mA", /^servo$/],
  ["Relay module (coil on)", 70, 0, "pct", "", /^relay/],
  ["28BYJ-48 stepper + ULN2003", 240, 0, "pct", "", null],
  ["— Radios & storage —"],
  ["microSD card (writing)", 50, 200, "sched", "Idle cards can draw 0.2–1mA; power them off if you can", /^sdcard$/],
  ["nRF24L01 (transmit)", 11.3, 0.9, "sched", "", /^nrf24/],
  ["LoRa SX1276 / RFM95 (transmit burst)", 120, 0.2, "sched", "", /^(lora|rfm95|ra02)/],
  ["HC-05 Bluetooth (connected)", 30, 8000, "always", "Can't really sleep", /^hc05$/],
  ["SIM800L GSM (idle on network)", 20, 1000, "always", "Needs 2A peaks when transmitting — use a big capacitor", null],
  ["— Other —"],
  ["Custom part", 10, 10, "sched", "", null],
];

// remaining-capacity vs per-cell voltage (light load), used to work out how much a regulator's dropout wastes
const CURVES = { li: [3.7, [[4.2, 1], [3.9, .75], [3.8, .6], [3.7, .45], [3.6, .25], [3.5, .1], [3.4, .05], [3.0, 0]]], alk: [1.5, [[1.6, 1], [1.4, .8], [1.3, .55], [1.2, .3], [1.1, .12], [1.0, .04], [0.9, 0]]], nimh: [1.2, [[1.4, 1], [1.3, .85], [1.25, .6], [1.2, .3], [1.1, .08], [1.0, 0]]], coin: [3.0, [[3.2, 1], [2.9, .8], [2.8, .4], [2.6, .12], [2.4, .05], [2.0, 0]]] };
function remainingAt(P, vReq) {
  const b = BATTERIES.find(x => x.id === P.bat) || {}; const ch = CURVES[b.chem];
  if (b.chem === "flat") return vReq > P.v ? 1 : 0;
  if (!ch) return Math.max(0, Math.min(1, (vReq - P.vmin) / Math.max(0.01, P.v - P.vmin)));
  const cells = Math.max(1, Math.round(P.v / ch[0])); const v = vReq / cells; const pts = ch[1];
  if (v >= pts[0][0]) return 1; if (v <= pts[pts.length - 1][0]) return 0;
  for (let i = 0; i < pts.length - 1; i++) { const [v0, f0] = pts[i], [v1, f1] = pts[i + 1]; if (v <= v0 && v >= v1) return f1 + (f0 - f1) * (v - v1) / (v0 - v1); }
  return 0;
}
function powerState() {
  if (!S.power) S.power = { bat: "18650", cap: 3000, v: 3.7, vmin: 3.0, usable: 90, self: 2, max: 8000, reg: "mcp1700", vout: 3.3, iq: 1.6, eff: 100, drop: 0.18, rmax: 250, rtype: "ldo", every: 10, awake: 5, loads: [] };
  return S.power;
}
function addLoad(i, qty = 1, fromPCB = "") {
  const L = LOADS[i]; const P = powerState();
  P.loads.push({ name: L[0] + (fromPCB ? ` (${fromPCB})` : ""), preset: i, qty, rail: /Pi 4|Zero 2|stepper|Relay|Servo|SIM800/.test(L[0]) ? "batt" : "reg", on: L[1], sleep: L[2], mode: L[3], pct: L[3] === "pct" ? 5 : 50 });
}
function powerCalc() {
  const P = powerState();
  const period = Math.max(1, P.every * 60), duty = Math.min(1, Math.max(0, P.awake) / period);
  const vb = P.v, reg = { type: P.rtype, vout: P.vout, iq: P.iq / 1000, eff: Math.max(1, P.eff) / 100 };
  const toBatt = (iLoad, rail) => {
    if (rail === "batt" || reg.type === "none") return iLoad;
    if (reg.type === "ldo") return iLoad;
    return (reg.vout * iLoad / reg.eff) / vb;
  };
  const rows = P.loads.map(l => {
    const f = l.mode === "always" ? 1 : l.mode === "pct" ? Math.min(100, Math.max(0, l.pct)) / 100 : duty;
    const on = l.on * l.qty, sl = l.sleep / 1000 * l.qty;
    const avgLoad = on * f + sl * (1 - f);
    const awakeLoad = l.mode === "always" ? on : l.mode === "pct" ? on : on;
    const sleepLoad = l.mode === "always" ? on : sl;
    return { l, f, avgLoad, avgBatt: toBatt(avgLoad, l.rail), awakeBatt: toBatt(awakeLoad, l.rail), sleepBatt: toBatt(l.mode === "pct" ? sl : sleepLoad, l.rail), peakBatt: toBatt(on, l.rail), regLoad: l.rail === "reg" ? avgLoad : 0, regPeak: l.rail === "reg" ? on : 0 };
  });
  const iq = reg.type === "none" ? 0 : reg.iq;
  const avg = rows.reduce((a, r) => a + r.avgBatt, 0) + iq;
  const awakeI = rows.reduce((a, r) => a + (r.l.mode === "sched" ? r.awakeBatt : r.l.mode === "always" ? r.awakeBatt : r.sleepBatt + (r.awakeBatt - r.sleepBatt) * r.f), 0) + iq;
  const sleepI = rows.reduce((a, r) => a + (r.l.mode === "sched" ? r.sleepBatt : r.l.mode === "always" ? r.awakeBatt : r.sleepBatt + (r.awakeBatt - r.sleepBatt) * r.f), 0) + iq;
  const peak = rows.reduce((a, r) => a + r.peakBatt, 0) + iq;
  const regAvg = rows.reduce((a, r) => a + r.regLoad, 0), regPeak = rows.reduce((a, r) => a + r.regPeak, 0);
  const selfI = P.cap * (P.self / 100) / 730; // mA equivalent
  const usable = P.cap * P.usable / 100;
  // LDO dropout: part of the battery below vout+drop is unusable
  let dropNote = null, effUsable = usable;
  if (reg.type === "ldo" && P.vout + P.drop > P.vmin) {
    const lost = remainingAt(P, P.vout + P.drop);
    if (P.vout + P.drop >= P.v) { dropNote = `The regulator needs at least ${(P.vout + P.drop).toFixed(2)}V in, but the battery is only ${P.v}V — it can't make ${P.vout}V.`; effUsable = 0; }
    else if (lost > 0.02) { effUsable = usable * (1 - lost); dropNote = `The regulator stops working below ${(P.vout + P.drop).toFixed(2)}V, so about ${Math.round(lost * 100)}% of the battery can't be used.`; }
  }
  if ((reg.type === "buck") && P.vout + P.drop > P.vmin) { const lost = remainingAt(P, P.vout + P.drop); effUsable = usable * (1 - lost); if (lost > 0.02) dropNote =  `A buck converter needs more than ${(P.vout + P.drop).toFixed(1)}V in — the battery drops below that before it's empty. A buck-boost would use it all.`; }
  if (reg.type === "boost" && P.vmin > P.vout) dropNote = `A boost converter can't step ${P.v}V down to ${P.vout}V — use a buck or LDO instead.`;
  const total = avg + selfI;
  const hours = total > 0 ? effUsable / total : Infinity;
  // regulator loss (average, W)
  let loss = 0; const vbatt = P.v;
  if (reg.type === "ldo") loss = (vbatt - P.vout) * regAvg / 1000 + vbatt * iq / 1000;
  else if (reg.type !== "none") loss = (P.vout * regAvg / reg.eff - P.vout * regAvg) / 1000 + vbatt * iq / 1000;
  const loadPower = rows.reduce((a, r) => a + (r.l.rail === "reg" && reg.type !== "none" ? P.vout : vbatt) * r.avgLoad, 0) / 1000;
  const regEff = loadPower + loss > 0 ? loadPower / (loadPower + loss) : 1;
  return { rows, duty, avg, awakeI, sleepI, peak, regPeak, iq, selfI, hours, usable: effUsable, loss, regEff, dropNote, total };
}
function fmtTime(h) {
  if (!isFinite(h)) return "Forever (nothing draws power)";
  if (h > 24 * 365 * 2) return `${(h / 24 / 365).toFixed(1)} years`;
  if (h > 24 * 60) return `${(h / 24 / 30.4).toFixed(1)} months`;
  if (h > 48) return `${(h / 24).toFixed(1)} days`;
  if (h >= 1) return `${h.toFixed(1)} hours`;
  return `${Math.round(h * 60)} minutes`;
}
const fmtI = mA => (mA >= 1 ? `${+mA.toFixed(mA >= 100 ? 0 : 2)} mA` : `${+(mA * 1000).toFixed(mA * 1000 >= 10 ? 0 : 1)} µA`);

function buildPowerie(root) {
  const P = powerState();
  const opt = (arr, cur) => arr.map(b => `<option value="${b.id}" ${b.id === cur ? "selected" : ""}>${esc(b.name)}</option>`).join("");
  const loadOpts = () => LOADS.map((l, i) => l.length === 1 ? `<option disabled>${esc(l[0])}</option>` : `<option value="${i}">${esc(l[0])}</option>`).join("");
  root.innerHTML = `<div class="calc-head"><div><h1 class="pw-h1">Powerie</h1><p>Plan the battery life of your project. Add every part, say how often it wakes up, and Powerie adds up the currents, regulator losses and sleep modes.</p></div><button class="btn" id="pw_pcb">${ico("board")}Add parts from my PCB</button></div>
  <div class="pw-layout"><div class="pw-left">
    <div class="pw-top">
      <section class="card"><h3>${icon("bat")}Battery</h3>
        <label class="fld"><span>Battery</span><div class="inp"><select id="pw_bat">${opt(BATTERIES, P.bat)}</select></div></label>
        <div class="grid3">${field("pw_cap", "Capacity", P.cap, "mAh")}${field("pw_v", "Voltage", P.v, "V")}${field("pw_vmin", "Empty at", P.vmin, "V")}</div>
        <div class="grid3">${field("pw_use", "Usable", P.usable, "%")}${field("pw_self", "Self-discharge", P.self, "%/mo")}${field("pw_max", "Max current", P.max, "mA")}</div><p class="note" id="pw_batnote"></p></section>
      <section class="card"><h3>${icon("div")}Regulator</h3>
        <label class="fld"><span>Regulator between battery and circuit</span><div class="inp"><select id="pw_reg">${opt(REGULATORS, P.reg)}</select></div></label>
        <div class="grid3">${field("pw_vout", "Output", P.vout, "V")}${field("pw_iq", "Idle draw (Iq)", P.iq, "µA")}${field("pw_eff", "Efficiency", P.eff, "%")}</div>
        <div class="grid3">${field("pw_drop", "Dropout", P.drop, "V")}${field("pw_rmax", "Max output", P.rmax, "mA")}<label class="fld"><span>Type</span><div class="inp"><select id="pw_rtype">${[["none", "None"], ["ldo", "Linear (LDO)"], ["buck", "Buck"], ["boost", "Boost"], ["buckboost", "Buck-boost"]].map(([k, l]) => `<option value="${k}" ${P.rtype === k ? "selected" : ""}>${l}</option>`).join("")}</select></div></label></div><p class="note" id="pw_regnote"></p></section>
      <section class="card"><h3>${icon("timer")}Wake-up schedule</h3><p class="sub">How often does your project wake up, and for how long?</p>
        <div class="grid2">${field("pw_every", "Wake up every", P.every, "min")}${field("pw_awake", "Stay awake for", P.awake, "s")}</div>
        <div class="seg wide" id="pw_sp">${[["Always on", 1, 60], ["Every minute", 1, 2], ["Every 10 min", 10, 5], ["Every hour", 60, 8], ["Once a day", 1440, 15]].map(([l, e, a]) => `<button data-e="${e}" data-a="${a}">${l}</button>`).join("")}</div><p class="note" id="pw_duty"></p></section>
    </div>
    <section class="card pw-loads"><div class="pw-lh"><h3>${icon("led")}Parts</h3><span class="grow"></span><div class="inp pw-add"><select id="pw_addsel"><option value="">＋ Add a part…</option>${loadOpts()}</select></div></div>
      <div class="pw-table" id="pw_rows"></div></section>
  </div>
  <aside class="pw-right"><section class="card pw-result" id="pw_res"></section></aside></div>`;
  const $p = s => root.querySelector(s);
  const num = (id, d) => { const v = parseEng($p(id).value); return isFinite(v) ? v : d; };
  let saveT = null; const save = () => { clearTimeout(saveT); saveT = setTimeout(() => commit("power"), 600); };
  const applyBat = () => { const b = BATTERIES.find(x => x.id === $p("#pw_bat").value); P.bat = b.id; if (b.id !== "custom") { Object.assign(P, { cap: b.cap, v: b.v, vmin: b.vmin, usable: b.usable, self: b.self, max: b.max }); ["cap", "v", "vmin", "use", "self", "max"].forEach(k => $p("#pw_" + k).value = P[{ use: "usable" }[k] || k]); } $p("#pw_batnote").textContent = b.note || ""; };
  const applyReg = () => { const r = REGULATORS.find(x => x.id === $p("#pw_reg").value); P.reg = r.id; if (r.id !== "custom") { Object.assign(P, { vout: r.vout, iq: r.iq, eff: r.eff || 100, drop: r.drop, rmax: r.max, rtype: r.type }); $p("#pw_vout").value = P.vout; $p("#pw_iq").value = P.iq; $p("#pw_eff").value = P.eff; $p("#pw_drop").value = P.drop; $p("#pw_rmax").value = P.rmax; $p("#pw_rtype").value = P.rtype; } $p("#pw_regnote").textContent = r.note || ""; };
  const readTop = () => { P.cap = num("#pw_cap", P.cap); P.v = num("#pw_v", P.v); P.vmin = num("#pw_vmin", P.vmin); P.usable = num("#pw_use", P.usable); P.self = num("#pw_self", P.self); P.max = num("#pw_max", P.max); P.vout = num("#pw_vout", P.vout); P.iq = num("#pw_iq", P.iq); P.eff = num("#pw_eff", P.eff); P.drop = num("#pw_drop", P.drop); P.rmax = num("#pw_rmax", P.rmax); P.rtype = $p("#pw_rtype").value; P.every = Math.max(0.01, num("#pw_every", P.every)); P.awake = Math.max(0, num("#pw_awake", P.awake)); };
  $p("#pw_bat").onchange = () => { applyBat(); readTop(); refresh(); save(); };
  $p("#pw_reg").onchange = () => { applyReg(); readTop(); refresh(); save(); };
  root.querySelectorAll(".pw-top input, #pw_rtype").forEach(el => el.addEventListener("input", () => {
    const card = el.closest(".card");
    if (card.contains($p("#pw_bat"))) { P.bat = "custom"; $p("#pw_bat").value = "custom"; $p("#pw_batnote").textContent = ""; }
    else if (card.contains($p("#pw_reg"))) { P.reg = "custom"; $p("#pw_reg").value = "custom"; $p("#pw_regnote").textContent = ""; }
    readTop(); refresh(); save();
  }));
  root.querySelectorAll("#pw_sp button").forEach(b => b.onclick = () => { $p("#pw_every").value = b.dataset.e; $p("#pw_awake").value = b.dataset.a; readTop(); refresh(); save(); });
  $p("#pw_addsel").onchange = e => { const i = +e.target.value; if (i) { addLoad(i); e.target.value = ""; drawRows(); refresh(); save(); } };
  $p("#pw_pcb").onclick = () => {
    let n = 0; const seen = {};
    for (const c of S.comps) { const fp = compFP(c); if (!fp) continue; const i = LOADS.findIndex(l => l[5] && l[5].test(fp.castellatedOf || fp.id)); if (i < 0) continue; seen[i] = seen[i] || { qty: 0, refs: [] }; seen[i].qty++; seen[i].refs.push(c.ref); }
    for (const i in seen) { if (P.loads.some(l => l.preset === +i && l.name.includes(seen[i].refs[0]))) continue; addLoad(+i, seen[i].qty, seen[i].refs.join(", ")); n++; }
    drawRows(); refresh(); save();
    toast(n ? `Added ${n} part type${n > 1 ? "s" : ""} from your board — check the currents match how you'll use them.` : "No parts with known power figures on your PCB yet (or they're already listed).", n ? "good" : "");
  };
  function drawRows() {
    const R = $p("#pw_rows");
    if (!P.loads.length) { R.innerHTML = `<div class="empty">${icon("led")}<h3>No parts yet</h3><p>Pick parts from the menu above, or press <b>Add parts from my PCB</b>.</p></div>`; return; }
    R.innerHTML = `<div class="pw-row pw-hdr"><span>Part</span><span>Qty</span><span>Powered from</span><span>Awake</span><span>Asleep</span><span>When it's on</span><span>Average</span><span></span></div>` +
      P.loads.map((l, i) => `<div class="pw-row" data-i="${i}"><div class="inp"><input data-k="name" value="${esc(l.name)}"></div><div class="inp"><input data-k="qty" type="number" min="1" value="${l.qty}"></div><div class="inp"><select data-k="rail"><option value="reg" ${l.rail === "reg" ? "selected" : ""}>Regulator</option><option value="batt" ${l.rail === "batt" ? "selected" : ""}>Battery direct</option></select></div><div class="inp"><input data-k="on" value="${l.on}"><em>mA</em></div><div class="inp"><input data-k="sleep" value="${l.sleep}"><em>µA</em></div><div class="pw-mode"><div class="inp"><select data-k="mode"><option value="sched" ${l.mode === "sched" ? "selected" : ""}>Wakes with schedule</option><option value="always" ${l.mode === "always" ? "selected" : ""}>Always on</option><option value="pct" ${l.mode === "pct" ? "selected" : ""}>On for % of time</option></select></div>${l.mode === "pct" ? `<div class="inp pct"><input data-k="pct" value="${l.pct}"><em>%</em></div>` : ""}</div><b class="pw-avg" data-avg="${i}">—</b><button class="x" data-del="${i}" title="Remove">${ico("close")}</button>${LOADS[l.preset] && LOADS[l.preset][4] ? `<small class="pw-note">${esc(LOADS[l.preset][4])}</small>` : ""}</div>`).join("");
    R.querySelectorAll("[data-k]").forEach(el => el.addEventListener(el.tagName === "SELECT" ? "change" : "input", () => {
      const l = P.loads[+el.closest(".pw-row").dataset.i]; const k = el.dataset.k;
      if (k === "name" || k === "rail" || k === "mode") l[k] = el.value; else { const v = parseEng(el.value); if (isFinite(v) && v >= 0) l[k] = k === "qty" ? Math.max(1, Math.round(v)) : v; }
      if (k === "mode") drawRows(); refresh(); save();
    }));
    R.querySelectorAll("[data-del]").forEach(b => b.onclick = () => { P.loads.splice(+b.dataset.del, 1); drawRows(); refresh(); save(); });
  }
  function refresh() {
    const r = powerCalc(); const res = $p("#pw_res");
    $p("#pw_duty").textContent = r.duty >= 1 ? "Always awake." : `Awake ${(r.duty * 100).toFixed(r.duty < 0.01 ? 3 : 1)}% of the time.`;
    r.rows.forEach((row, i) => { const el = root.querySelector(`[data-avg="${i}"]`); if (el) el.textContent = fmtI(row.avgBatt); });
    const tips = [];
    if (r.dropNote) tips.push(["bad", r.dropNote]);
    if (r.peak > P.max) tips.push(["bad", `Peak draw ${fmtI(r.peak)} is more than the battery can supply (${fmtI(P.max)}).${P.bat === "cr2032" ? " Coin cells can only give a few mA — add a big capacitor or use AA/Li-ion." : ""}`]);
    if (P.rtype !== "none" && r.regPeak > P.rmax) tips.push(["bad", `Parts on the regulator can pull ${fmtI(r.regPeak)}, but it's only rated for ${fmtI(P.rmax)}.`]);
    const share = r.rows.map(x => ({ n: x.l.name, v: x.avgBatt })).concat(r.iq ? [{ n: "Regulator idle current", v: r.iq }] : []).sort((a, b) => b.v - a.v);
    const tot = share.reduce((a, b) => a + b.v, 0) || 1;
    if (share[0] && share[0].n !== "Regulator idle current" && share[0].v / tot > 0.5 && r.rows.length > 1) tips.push(["warn", `<b>${esc(share[0].n)}</b> uses ${Math.round(share[0].v / tot * 100)}% of your power — that's the one to optimise.`]);
    if (r.iq > 0.5 && r.iq / Math.max(r.sleepI, 1e-9) > 0.3) tips.push(["warn", `The regulator's idle current (${fmtI(r.iq)}) is a big part of your sleep current. A low-Iq LDO like the MCP1700 or HT7333 draws only a few µA.`]);
    if (P.rtype === "ldo" && P.v - P.vout > 2 && r.regEff < 0.6) tips.push(["warn", `Your LDO is turning ${Math.round((1 - r.regEff) * 100)}% of the battery's energy into heat (${P.v}V → ${P.vout}V). A buck converter would last much longer.`]);
    const devBoard = r.rows.find(x => /DevKit board|Nano \/ Uno|Blue\/Black|D1 Mini/.test(x.l.name) && x.l.mode !== "always");
    if (devBoard && r.duty < 0.2) tips.push(["warn", `${esc(devBoard.l.name.split(" (")[0])} boards waste power in sleep (LED, USB chip, regulator). For long battery life use a bare module or a low-power board.`]);
    if (r.rows.some(x => /NeoPixel/.test(x.l.name)) && r.duty < 0.5) tips.push(["info", "Switch the NeoPixels' power off with a MOSFET while sleeping — each one draws ~1mA even when dark."]);
    if (r.selfI > r.avg * 0.3 && isFinite(r.hours)) tips.push(["info", `Self-discharge matters at this low current: the battery loses ${P.self}% a month on its own.`]);
    if (!tips.length && P.loads.length) tips.push(["good", "Looks healthy — no obvious power problems."]);
    const days = r.hours / 24;
    res.innerHTML = `<div class="pw-big"><span>Estimated runtime</span><b>${P.loads.length ? fmtTime(r.hours) : "—"}</b><small>${P.loads.length && isFinite(r.hours) ? `${r.hours.toFixed(0)} hours on ${Math.round(r.usable)} usable mAh` : "Add some parts to get started"}</small></div>
      <div class="outs">${out("pw_o1", "Average draw", true)}${out("pw_o2", "While awake")}${out("pw_o3", "While asleep")}${out("pw_o4", "Peak")}${out("pw_o5", "Regulator loss")}${out("pw_o6", "Power efficiency")}</div>
      <h4>Where the power goes</h4><canvas id="pw_chart" width="600" height="${Math.max(60, share.length * 26 + 8)}"></canvas>
      <h4>One wake-up cycle</h4><canvas id="pw_tl" width="600" height="110"></canvas>
      <h4>Tips</h4><div class="pw-tips">${tips.map(([k, t]) => `<div class="pw-tip ${k}"><i></i><span>${t}</span></div>`).join("")}</div>
      <p class="note">Estimates — real batteries lose capacity in the cold and under heavy loads. Measure your real currents with a multimeter in series to refine them.</p>`;
    const set = (id, v) => { const e = res.querySelector("#" + id); if (e) e.textContent = v; };
    set("pw_o1", fmtI(r.total)); set("pw_o2", fmtI(r.awakeI)); set("pw_o3", fmtI(r.sleepI)); set("pw_o4", fmtI(r.peak)); set("pw_o5", r.loss >= 0.001 ? `${(r.loss * 1000).toFixed(r.loss < 0.01 ? 2 : 0)} mW` : `${(r.loss * 1e6).toFixed(1)} µW`); set("pw_o6", `${Math.round(r.regEff * 100)} %`);
    drawShare(res.querySelector("#pw_chart"), share, tot); drawTimeline(res.querySelector("#pw_tl"), r);
  }
  function drawShare(c, share, tot) {
    const dpr = window.devicePixelRatio || 1; const W = c.clientWidth || 360, H = share.length * 26 + 8; c.width = W * dpr; c.height = H * dpr; c.style.height = H + "px";
    const x = c.getContext("2d"); x.scale(dpr, dpr); x.font = "600 11.5px Segoe UI, system-ui, sans-serif"; x.textBaseline = "middle";
    const cols = ["#3fe0d0", "#f08a4b", "#a855f7", "#facc15", "#3b82f6", "#ec4899", "#22c55e", "#ef4444", "#94a3b8"];
    share.forEach((s, i) => { const y = 4 + i * 26; const f = s.v / tot; const bw = Math.max(2, (W - 150) * f);
      x.fillStyle = "#1b2330"; x.beginPath(); x.roundRect ? x.roundRect(0, y, W - 150, 20, 6) : x.rect(0, y, W - 150, 20); x.fill();
      x.fillStyle = cols[i % cols.length]; x.beginPath(); x.roundRect ? x.roundRect(0, y, bw, 20, 6) : x.rect(0, y, bw, 20); x.fill();
      x.fillStyle = "#e7ecf3"; x.fillText(s.n.length > 30 ? s.n.slice(0, 29) + "…" : s.n, 8, y + 10.5);
      x.fillStyle = "#8c98ab"; x.textAlign = "right"; x.fillText(`${fmtI(s.v)} · ${Math.round(f * 100)}%`, W - 4, y + 10.5); x.textAlign = "left"; });
  }
  function drawTimeline(c, r) {
    const dpr = window.devicePixelRatio || 1; const W = c.clientWidth || 360, H = 110; c.width = W * dpr; c.height = H * dpr; const x = c.getContext("2d"); x.scale(dpr, dpr);
    const P2 = powerState(); const period = P2.every * 60, awake = Math.min(P2.awake, period);
    const maxI = Math.max(r.awakeI, r.sleepI, 0.001); const ly = i => H - 22 - (Math.log10(1 + i * 1000) / Math.log10(1 + maxI * 1000)) * (H - 40);
    x.strokeStyle = "#232c3a"; x.lineWidth = 1; for (let g = 0; g <= 4; g++) { const y = 10 + g * (H - 32) / 4; x.beginPath(); x.moveTo(40, y); x.lineTo(W, y); x.stroke(); }
    const aw = awake >= period ? W - 40 : Math.max(6, (W - 40) * Math.min(0.5, awake / period * 4));
    const grad = x.createLinearGradient(0, 0, 0, H); grad.addColorStop(0, "rgba(240,138,75,.55)"); grad.addColorStop(1, "rgba(240,138,75,.05)");
    x.beginPath(); x.moveTo(40, H - 22); x.lineTo(40, ly(r.awakeI)); x.lineTo(40 + aw, ly(r.awakeI)); x.lineTo(40 + aw, ly(r.sleepI)); x.lineTo(W, ly(r.sleepI)); x.lineTo(W, H - 22); x.closePath(); x.fillStyle = grad; x.fill();
    x.strokeStyle = "#f08a4b"; x.lineWidth = 2; x.beginPath(); x.moveTo(40, ly(r.awakeI)); x.lineTo(40 + aw, ly(r.awakeI)); x.lineTo(40 + aw, ly(r.sleepI)); x.lineTo(W, ly(r.sleepI)); x.stroke();
    x.fillStyle = "#8c98ab"; x.font = "11px Segoe UI, sans-serif"; x.fillText(fmtI(r.awakeI), 44, ly(r.awakeI) - 6); x.fillText(fmtI(r.sleepI), Math.max(44 + aw + 4, W - 70), ly(r.sleepI) - 6);
    x.fillText(awake >= period ? "always awake" : `awake ${awake}s${awake / period < 0.125 ? " (stretched)" : ""}`, 40, H - 6); x.textAlign = "right"; x.fillText(`repeats every ${P2.every >= 60 ? (P2.every / 60).toFixed(P2.every % 60 ? 1 : 0) + " h" : P2.every + " min"}`, W - 2, H - 6); x.textAlign = "left";
  }
  applyBat(); applyReg(); readTop();
  drawRows(); refresh();
  let lastW = 0; new ResizeObserver(() => { const w = root.clientWidth; if (w && w !== lastW) { lastW = w; refresh(); } }).observe(root);
}
