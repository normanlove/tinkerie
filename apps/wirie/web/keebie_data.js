// Keebie data: switch types, keycap profiles, keyboard controllers (with the
// support circuitry each one needs) and ready-made layouts.
// Part of Wirie, in the Tinkerie maker suite — by nrmluvdev.
"use strict";

const KB_SWITCHES = [
  { id: "mx_hs", name: "MX-style, hot-swap sockets", fp: "kb_mx_hs", pitch: [19.05, 19.05], stabs: true, cut: 14, fam: "mx", note: "Cherry, Gateron, Kailh Box, Akko… push in, no soldering. Kailh MX sockets on the back." },
  { id: "mx", name: "MX-style, soldered", fp: "mx_switch", pitch: [19.05, 19.05], stabs: true, cut: 14, fam: "mx", note: "The classic: switches are soldered in. 5-pin (PCB-mount) switches fit too." },
  { id: "choc_hs", name: "Kailh Choc v1 low-profile, hot-swap", fp: "kb_choc_hs", pitch: [18, 17], stabs: false, cut: 13.8, fam: "choc", note: "Thin keyboards. Choc keys are spaced 18 × 17 mm and need Choc keycaps." },
  { id: "choc", name: "Kailh Choc v1 low-profile, soldered", fp: "choc_switch", pitch: [18, 17], stabs: false, cut: 13.8, fam: "choc", note: "Thin keyboards. Choc keys are spaced 18 × 17 mm and need Choc keycaps." },
  { id: "alps", name: "Alps SKCM/SKCL & Matias", fp: "kb_alps", pitch: [19.05, 19.05], stabs: false, cut: 15.5, cutH: 12.8, fam: "alps", note: "Vintage Alps or modern Matias switches. They need Alps-mount keycaps and Alps-style stabilisers." },
];

// keycap profiles: height is the row-3 cap height; sculpt = rows have different shapes
const KB_PROFILES = [
  { id: "cherry", name: "Cherry", fam: "mx", sculpt: true, h: "9.4 mm", note: "Low, sculpted: the most common profile for enthusiast sets." },
  { id: "oem", name: "OEM", fam: "mx", sculpt: true, h: "11.9 mm", note: "What most pre-built keyboards ship with." },
  { id: "sa", name: "SA", fam: "mx", sculpt: true, h: "13.4 mm", note: "Tall and spherical, retro typewriter feel." },
  { id: "mt3", name: "MT3", fam: "mx", sculpt: true, h: "~13 mm", note: "Tall, deep-dished, vintage-inspired." },
  { id: "kat", name: "KAT", fam: "mx", sculpt: true, h: "~11 mm", note: "Medium-height spherical sculpted caps." },
  { id: "dsa", name: "DSA", fam: "mx", sculpt: false, h: "7.6 mm", note: "Flat, uniform: every row is the same, ideal for ortholinear and odd layouts." },
  { id: "xda", name: "XDA", fam: "mx", sculpt: false, h: "9.1 mm", note: "Uniform with a wide, flat top." },
  { id: "kam", name: "KAM", fam: "mx", sculpt: false, h: "~9.5 mm", note: "Uniform spherical caps." },
  { id: "mbk", name: "MBK (Choc)", fam: "choc", sculpt: false, h: "~3.5 mm", note: "Uniform low-profile caps for Choc switches." },
  { id: "cfx", name: "CFX (Choc)", fam: "choc", sculpt: true, h: "low", note: "Sculpted low-profile caps for Choc switches." },
  { id: "chicago", name: "Chicago Steno (Choc)", fam: "choc", sculpt: false, h: "low", note: "Uniform, MX-sized low-profile caps." },
  { id: "alps", name: "Alps-mount (SKCM)", fam: "alps", sculpt: true, h: "varies", note: "Vintage and Matias-compatible caps." },
];

const KB_SIZES = [[1, 1], [1.25, 1], [1.5, 1], [1.75, 1], [2, 1], [2.25, 1], [2.75, 1], [3, 1], [6.25, 1], [7, 1], [1, 2], [1, 1.5]];

// ---------------- controllers ----------------
// pins: GPIO usable for the matrix (pad names on the footprint), in preference order.
// serial: pin for split-keyboard data; led: data pin for per-key RGB.
// pwr: the pad for 5V (LEDs), logic: the pad that powers the other half over TRRS.
const PROMICRO_PINS = ["D4", "D5", "D6", "D7", "D8", "D9", "D10", "D16", "D14", "D15", "A0", "A1", "A2", "A3", "D2", "RX1", "TX0", "D3"];
const PROMICRO_AVR = { TX0: "D3", RX1: "D2", D2: "D1", D3: "D0", D4: "D4", D5: "C6", D6: "D7", D7: "E6", D8: "B4", D9: "B5", D10: "B6", D16: "B2", D14: "B3", D15: "B1", A0: "F7", A1: "F6", A2: "F5", A3: "F4" };
const PROMICRO_ARDUINO = { TX0: 1, RX1: 0, D2: 2, D3: 3, D4: 4, D5: 5, D6: 6, D7: 7, D8: 8, D9: 9, D10: 10, D16: 16, D14: 14, D15: 15, A0: 18, A1: 19, A2: 20, A3: 21 };
const XIAO_RP = { D0: "GP26", D1: "GP27", D2: "GP28", D3: "GP29", D4: "GP6", D5: "GP7", D6: "GP0", D7: "GP1", D8: "GP2", D9: "GP4", D10: "GP3" };
const range = (pre, a, b, skip = []) => { const o = []; for (let i = a; i <= b; i++) if (!skip.includes(i)) o.push(pre + i); return o; };

const KB_MCUS = [
  // ----- boards you solder on (easiest)
  { id: "promicro", group: "Boards you solder on", name: "Pro Micro (ATmega32U4)", fp: "promicro", fw: "QMK · Vial", volt: 5, pins: PROMICRO_PINS, serial: "D3", led: "TX0", pwr: "VCC", logic: "VCC", reset: ["RST", "GND"],
    qmk: { board: "promicro", bootloader: "caterina", map: n => PROMICRO_AVR[n] }, desc: "The classic keyboard controller: cheap, 18 pins, QMK. Micro-USB on most; USB-C clones exist." },
  { id: "elitec", group: "Boards you solder on", name: "Elite-C (ATmega32U4, USB-C)", fp: "promicro", fw: "QMK · Vial", volt: 5, pins: PROMICRO_PINS, serial: "D3", led: "TX0", pwr: "VCC", logic: "VCC", reset: ["RST", "GND"],
    qmk: { board: "elite_c", bootloader: "atmel-dfu", map: n => PROMICRO_AVR[n] }, desc: "A sturdier Pro Micro with USB-C and a DFU bootloader. Same footprint (its extra bottom pins aren't used here)." },
  { id: "rp2040pm", group: "Boards you solder on", name: "RP2040 Pro Micro (KB2040, Sea-Picro, Elite-Pi…)", fp: "promicro", fw: "QMK · Vial · KMK", volt: 3.3, pins: PROMICRO_PINS, serial: "D3", led: "TX0", pwr: "RAW", logic: "RAW", reset: ["RST", "GND"],
    qmk: { board: "promicro", bootloader: "caterina", convert: "kb2040", map: n => PROMICRO_AVR[n] }, desc: "Pro Micro-shaped RP2040 boards: more memory, USB-C, cheap. Build QMK with CONVERT_TO=kb2040 (or your board's converter)." },
  { id: "nicenano", group: "Wireless (Bluetooth)", name: "nice!nano v2 (nRF52840)", fp: "promicro", fw: "ZMK (Bluetooth)", volt: 3.3, wireless: true, pins: PROMICRO_PINS, led: "TX0", pwr: "VCC", logic: "VCC", reset: ["RST", "GND"],
    zmk: { node: "&pro_micro", map: n => PROMICRO_ARDUINO[n] }, desc: "Wireless Pro Micro. Keebie adds a battery connector and on/off switch. Split halves talk over Bluetooth, so no TRRS cable." },
  { id: "pico", group: "Boards you solder on", name: "Raspberry Pi Pico (RP2040)", fp: "pico", fw: "QMK · Vial · KMK", volt: 3.3, pins: [...range("GP", 2, 22), "GP26", "GP27", "GP28"], serial: "GP1", led: "GP0", pwr: "VBUS", logic: "VSYS", reset: ["RUN", "GND"],
    qmk: { processor: "RP2040", bootloader: "rp2040", map: n => n }, desc: "Lots of pins (26) for big keyboards. Larger than a Pro Micro." },
  { id: "rp2040zero", group: "Boards you solder on", name: "Waveshare RP2040-Zero", fp: "rp2040zero", fw: "QMK · KMK", volt: 3.3, pins: [...range("GP", 2, 15), "GP26", "GP27", "GP28", "GP29"], serial: "GP1", led: "GP0", pwr: "5V", logic: "5V",
    qmk: { processor: "RP2040", bootloader: "rp2040", map: n => n }, desc: "Tiny RP2040 with USB-C and 20 pins." },
  { id: "xiaorp", group: "Boards you solder on", name: "Seeed XIAO RP2040", fp: "xiao", fw: "QMK · KMK", volt: 3.3, pins: ["D0", "D1", "D2", "D3", "D4", "D5", "D8", "D9", "D10"], serial: "D6", led: "D7", pwr: "5V", logic: "5V",
    qmk: { processor: "RP2040", bootloader: "rp2040", map: n => XIAO_RP[n] }, desc: "Thumb-sized, only 11 pins: macropads and small splits." },
  { id: "xiaoble", group: "Wireless (Bluetooth)", name: "Seeed XIAO nRF52840", fp: "xiao", fw: "ZMK (Bluetooth)", volt: 3.3, wireless: true, pins: ["D0", "D1", "D2", "D3", "D4", "D5", "D6", "D7", "D8", "D9", "D10"], pwr: "5V", logic: "3V3",
    zmk: { node: "&xiao_d", map: n => +n.slice(1) }, desc: "Tiny wireless board. Battery pads are underneath (wire a battery there)." },
  { id: "blackpill", group: "Boards you solder on", name: "WeAct Black Pill (STM32F411)", fp: "blackpill", fw: "QMK · Vial", volt: 3.3, pins: ["B12", "B13", "B14", "B15", "A8", "A10", "A15", "B3", "B4", "B5", "B6", "B7", "B8", "B9", "A0", "A1", "A2", "A3", "A4", "A5", "A6", "A7", "B0", "B10", "C14", "C15", "B2"], serial: "A9", led: "B1", pwr: "5V", logic: "5V", reset: ["R", "GND"],
    qmk: { board: "BLACKPILL_STM32_F411", processor: "STM32F411", bootloader: "stm32-dfu", map: n => n }, desc: "Fast ARM board with 27 usable pins and USB-C." },
  // ----- bare chips (Keebie adds USB-C and everything the chip needs)
  { id: "atmega32u4", group: "Bare chips (all circuitry added)", name: "ATmega32U4 chip", fp: "kb_atmega32u4", bare: true, fw: "QMK · Vial", volt: 5,
    pins: ["PB0", "PB1", "PB2", "PB3", "PB4", "PB5", "PB6", "PB7", "PC6", "PC7", "PD1", "PD2", "PD4", "PD5", "PD6", "PD7", "PE6", "PF0", "PF1", "PF4", "PF5", "PF6", "PF7", "PD3", "PD0"], serial: "PD0", led: "PD3",
    qmk: { processor: "atmega32u4", bootloader: "atmel-dfu", map: n => n.slice(1) }, desc: "The Pro Micro's chip on your own board: USB-C, 16MHz crystal and 25 pins. Factory assembly recommended (0.8mm pin pitch)." },
  { id: "rp2040", group: "Bare chips (all circuitry added)", name: "RP2040 chip", fp: "kb_rp2040", bare: true, fw: "QMK · Vial · KMK", volt: 3.3,
    pins: range("GPIO", 2, 29), serial: "GPIO1", led: "GPIO0",
    qmk: { processor: "RP2040", bootloader: "rp2040", map: n => "GP" + n.slice(4) }, desc: "Raspberry Pi's chip with flash, crystal, regulator and USB-C: 30 pins. Needs factory assembly (0.4mm QFN)." },
  { id: "stm32f072", group: "Bare chips (all circuitry added)", name: "STM32F072 chip", fp: "kb_stm32f072", bare: true, fw: "QMK · Vial", volt: 3.3,
    pins: [...range("PA", 0, 8), "PA10", "PA15", ...range("PB", 0, 15, [2]), "PC13", "PC14", "PC15"], serial: "PA9", led: "PB15",
    qmk: { processor: "STM32F072", bootloader: "stm32-dfu", map: n => n.slice(1) }, desc: "Used by many commercial QMK boards: USB without a crystal, 32 pins, LQFP-48." },
  { id: "stm32f411", group: "Bare chips (all circuitry added)", name: "STM32F411 chip", fp: "kb_stm32f411", bare: true, fw: "QMK · Vial", volt: 3.3,
    pins: [...range("PA", 0, 8), "PA10", "PA15", ...range("PB", 0, 10, [2]), ...range("PB", 12, 15), "PC13", "PC14", "PC15"], serial: "PA9", led: "PA7",
    qmk: { processor: "STM32F411", bootloader: "stm32-dfu", map: n => n.slice(1) }, desc: "The Black Pill's chip with a 25MHz crystal: fast, 30 pins. Factory assembly recommended." },
];
const KB_MCU = Object.fromEntries(KB_MCUS.map(m => [m.id, m]));

// Support circuitry for the bare chips. B gives: add(fp, value, ref?) → part,
// net(name, ...[part, pinName]) and usb(): a USB-C socket with fuse and CC resistors.
function kbSupport(mcu, B, U) {
  const C = (v, fp = "c_0805") => B.add(fp, v);
  const R = (v) => B.add("r_0805", v);
  const usb = B.usb(); // nets VBUS, 5V, GND, and returns { dp: [[J,"DP1"],[J,"DP2"]], dn: [...] }
  if (mcu.id === "atmega32u4") {
    B.net("5V", [U, "UVCC"], [U, "VCC"], [U, "AVCC"], [U, "VBUS"]);
    B.net("GND", [U, "GND"], [U, "UGND"]);
    const rp = R("22R"), rn = R("22R");
    B.net("USB_D+_C", ...usb.dp, [rp, "1"]); B.net("USB_D+", [rp, "2"], [U, "D+"]);
    B.net("USB_D-_C", ...usb.dn, [rn, "1"]); B.net("USB_D-", [rn, "2"], [U, "D-"]);
    const y = B.add("xtal_3225", "16MHz"), c1 = C("22pF"), c2 = C("22pF");
    B.net("XTAL1", [U, "XTAL1"], [y, "XIN"], [c1, "1"]); B.net("XTAL2", [U, "XTAL2"], [y, "XOUT"], [c2, "1"]); B.net("GND", [y, "GND"], [c1, "2"], [c2, "2"]);
    const cu = C("1uF"); B.net("UCAP", [U, "UCAP"], [cu, "1"]); B.net("GND", [cu, "2"]);
    for (let i = 0; i < 4; i++) { const c = C("100nF"); B.net("5V", [c, "1"]); B.net("GND", [c, "2"]); }
    const cb = C("10uF"); B.net("5V", [cb, "1"]); B.net("GND", [cb, "2"]);
    const rr = R("10k"), sw = B.add("sw4x4smd", "Reset", "SW_RST"); B.net("RESET", [U, "RESET"], [rr, "1"], [sw, "A"]); B.net("5V", [rr, "2"]); B.net("GND", [sw, "B"]);
    const rh = R("10k"); B.net("HWB", [U, "PE2"], [rh, "1"]); B.net("GND", [rh, "2"]);
    return { vcc: "5V" };
  }
  // the 3.3V chips: regulator first
  const ldo = B.add("ap2112", "AP2112K-3.3"), ci = C("10uF"), co = C("10uF");
  B.net("5V", [ldo, "VIN"], [ldo, "EN"], [ci, "1"]); B.net("3V3", [ldo, "VOUT"], [co, "1"]); B.net("GND", [ldo, "GND"], [ci, "2"], [co, "2"]);
  const decouple = n => { for (let i = 0; i < n; i++) { const c = C("100nF"); B.net("3V3", [c, "1"]); B.net("GND", [c, "2"]); } };
  if (mcu.id === "rp2040") {
    B.net("3V3", [U, "IOVDD"], [U, "USB_VDD"], [U, "ADC_AVDD"], [U, "VREG_IN"]); B.net("GND", [U, "GND"], [U, "TESTEN"]);
    decouple(7);
    const c11a = C("1uF"), c11b = C("1uF"); B.net("1V1", [U, "VREG_VOUT"], [U, "DVDD"], [c11a, "1"], [c11b, "1"]); B.net("GND", [c11a, "2"], [c11b, "2"]);
    const rp = R("27R"), rn = R("27R");
    B.net("USB_D+_C", ...usb.dp, [rp, "1"]); B.net("USB_D+", [rp, "2"], [U, "USB_DP"]);
    B.net("USB_D-_C", ...usb.dn, [rn, "1"]); B.net("USB_D-", [rn, "2"], [U, "USB_DM"]);
    const y = B.add("xtal_3225", "12MHz"), c1 = C("15pF"), c2 = C("15pF"), rx = R("1k");
    B.net("XIN", [U, "XIN"], [y, "XIN"], [c1, "1"]); B.net("XOUT", [U, "XOUT"], [rx, "1"]); B.net("XOUT_Y", [rx, "2"], [y, "XOUT"], [c2, "1"]); B.net("GND", [y, "GND"], [c1, "2"], [c2, "2"]);
    const f = B.add("kb_w25q128", "W25Q128JVS"), cf = C("100nF");
    B.net("QSPI_SS", [U, "QSPI_SS"], [f, "CS"]); B.net("QSPI_SD0", [U, "QSPI_SD0"], [f, "DI"]); B.net("QSPI_SD1", [U, "QSPI_SD1"], [f, "DO"]);
    B.net("QSPI_SD2", [U, "QSPI_SD2"], [f, "IO2"]); B.net("QSPI_SD3", [U, "QSPI_SD3"], [f, "IO3"]); B.net("QSPI_SCLK", [U, "QSPI_SCLK"], [f, "CLK"]);
    B.net("3V3", [f, "VCC"], [cf, "1"]); B.net("GND", [f, "GND"], [cf, "2"]);
    const rb = R("1k"), sb = B.add("sw4x4smd", "Boot", "SW_BOOT"); B.net("QSPI_SS", [rb, "1"]); B.net("BOOT", [rb, "2"], [sb, "A"]); B.net("GND", [sb, "B"]);
    const sr = B.add("sw4x4smd", "Reset", "SW_RST"), rr = R("10k"); B.net("RUN", [U, "RUN"], [sr, "A"], [rr, "1"]); B.net("GND", [sr, "B"]); B.net("3V3", [rr, "2"]);
    return { vcc: "3V3" };
  }
  // STM32F072 / F411
  const f4 = mcu.id === "stm32f411";
  B.net("3V3", [U, "VDD"], [U, "VDDA"], [U, "VBAT"]); if (!f4) B.net("3V3", [U, "VDDIO2"]);
  B.net("GND", [U, "VSS"], [U, "VSSA"]);
  decouple(f4 ? 4 : 5);
  const ca = C("1uF"); B.net("3V3", [ca, "1"]); B.net("GND", [ca, "2"]);
  B.net("USB_D+", ...usb.dp, [U, "PA12"]); B.net("USB_D-", ...usb.dn, [U, "PA11"]);
  if (f4) {
    const y = B.add("xtal_3225", "25MHz"), c1 = C("10pF"), c2 = C("10pF");
    B.net("OSC_IN", [U, "PH0"], [y, "XIN"], [c1, "1"]); B.net("OSC_OUT", [U, "PH1"], [y, "XOUT"], [c2, "1"]); B.net("GND", [y, "GND"], [c1, "2"], [c2, "2"]);
    const cv = C("4.7uF"); B.net("VCAP1", [U, "VCAP1"], [cv, "1"]); B.net("GND", [cv, "2"]);
  }
  const rb = R("10k"), sb = B.add("sw4x4smd", "Boot", "SW_BOOT"); B.net("BOOT0", [U, "BOOT0"], [rb, "1"], [sb, "A"]); B.net("GND", [rb, "2"]); B.net("3V3", [sb, "B"]);
  const cn = C("100nF"), sr = B.add("sw4x4smd", "Reset", "SW_RST"); B.net("NRST", [U, "NRST"], [cn, "1"], [sr, "A"]); B.net("GND", [cn, "2"], [sr, "B"]);
  return { vcc: "3V3" };
}

// ---------------- layouts ----------------
// Each row: { y, keys: "tokens" }. Tokens: Label|w|h|shape, +gap, @x (absolute x).
// "_" as a label means a blank key. Labels with spaces use ~ (e.g. Caps~Lock).
const ROW_60 = [
  "Esc 1 2 3 4 5 6 7 8 9 0 - = Backspace|2",
  "Tab|1.5 Q W E R T Y U I O P [ ] \\|1.5",
  "Caps|1.75 A S D F G H J K L ; ' Enter|2.25",
  "Shift|2.25 Z X C V B N M , . / Shift|2.75",
  "Ctrl|1.25 Win|1.25 Alt|1.25 Space|6.25 Alt|1.25 Win|1.25 Menu|1.25 Ctrl|1.25",
];
const KB_PRESETS = [
  { id: "macro9", name: "Macropad 3×3", rows: ["7 8 9", "4 5 6", "1 2 3"] },
  { id: "macro16", name: "Macropad 4×4", rows: ["Esc F1 F2 F3", "7 8 9 /", "4 5 6 *", "1 2 3 -"] },
  { id: "numpad", name: "Numpad (17 keys)", rows: ["Num / * -", "7 8 9 +|1|2", "4 5 6", "1 2 3 Enter|1|2", "0|2 ."] },
  { id: "ortho40", name: "40% ortholinear 4×12 (Planck-style)", rows: ["Tab Q W E R T Y U I O P Bksp", "Esc A S D F G H J K L ; '", "Shift Z X C V B N M , . / Enter", "Fn Ctrl Alt Win Lower Space Space Raise Left Down Up Right"] },
  { id: "ansi60", name: "60% ANSI (61 keys)", rows: ROW_60 },
  { id: "iso60", name: "60% ISO (62 keys)", rows: ["Esc 1 2 3 4 5 6 7 8 9 0 - = Backspace|2", "Tab|1.5 Q W E R T Y U I O P [ ] Enter|1.25|2|iso", "Caps|1.75 A S D F G H J K L ; ' #", "Shift|1.25 \\ Z X C V B N M , . / Shift|2.75", ROW_60[4]] },
  { id: "ansi65", name: "65% (68 keys)", rows: ["Esc 1 2 3 4 5 6 7 8 9 0 - = Backspace|2 Del", "Tab|1.5 Q W E R T Y U I O P [ ] \\|1.5 PgUp", "Caps|1.75 A S D F G H J K L ; ' Enter|2.25 PgDn", "Shift|2.25 Z X C V B N M , . / Shift|1.75 Up End", "Ctrl|1.25 Win|1.25 Alt|1.25 Space|6.25 Alt Fn Ctrl Left Down Right"] },
  { id: "ansi75", name: "75% (84 keys)", rows: ["Esc F1 F2 F3 F4 F5 F6 F7 F8 F9 F10 F11 F12 PrtSc Ins Del", "` 1 2 3 4 5 6 7 8 9 0 - = Backspace|2 Home", "Tab|1.5 Q W E R T Y U I O P [ ] \\|1.5 PgUp", "Caps|1.75 A S D F G H J K L ; ' Enter|2.25 PgDn", "Shift|2.25 Z X C V B N M , . / Shift|1.75 Up End", "Ctrl|1.25 Win|1.25 Alt|1.25 Space|6.25 Alt Fn Ctrl Left Down Right"] },
  { id: "tkl", name: "Tenkeyless / TKL (87 keys)", rows: [{ y: 0, k: "Esc +1 F1 F2 F3 F4 +0.5 F5 F6 F7 F8 +0.5 F9 F10 F11 F12 @15.25 PrtSc ScrLk Pause" }, { y: 1.25, k: "` 1 2 3 4 5 6 7 8 9 0 - = Backspace|2 @15.25 Ins Home PgUp" }, "Tab|1.5 Q W E R T Y U I O P [ ] \\|1.5 @15.25 Del End PgDn", "Caps|1.75 A S D F G H J K L ; ' Enter|2.25", "Shift|2.25 Z X C V B N M , . / Shift|2.75 @16.25 Up", ROW_60[4] + " @15.25 Left Down Right"] },
  { id: "full", name: "Full size (104 keys)", rows: [{ y: 0, k: "Esc +1 F1 F2 F3 F4 +0.5 F5 F6 F7 F8 +0.5 F9 F10 F11 F12 @15.25 PrtSc ScrLk Pause" }, { y: 1.25, k: "` 1 2 3 4 5 6 7 8 9 0 - = Backspace|2 @15.25 Ins Home PgUp @18.5 Num / * -" }, "Tab|1.5 Q W E R T Y U I O P [ ] \\|1.5 @15.25 Del End PgDn @18.5 7 8 9 +|1|2", "Caps|1.75 A S D F G H J K L ; ' Enter|2.25 @18.5 4 5 6", "Shift|2.25 Z X C V B N M , . / Shift|2.75 @16.25 Up @18.5 1 2 3 Enter|1|2", ROW_60[4] + " @15.25 Left Down Right @18.5 0|2 ."] },
  { id: "alice", name: "Alice-style (angled 65%)", rows: ["Esc 1 2 3 4 5 6 +1 7 8 9 0 - = Backspace|2", "Tab|1.5 Q W E R T +1 Y U I O P [ ] \\|1.5", "Caps|1.75 A S D F G +1 H J K L ; ' Enter|2.25", "Shift|2.25 Z X C V B +1 B N M , . / Shift|1.75 Fn", "Ctrl|1.5 +1 Alt|1.5 Space|2.25 Win|1.25 +1 Space|2.75 Alt|1.5 +1 Ctrl|1.5"], split: { mode: "angled", at: 7.5, gap: 0, angle: 10 } },
  { id: "corne", name: "Split 3×6 + 3 thumbs (Corne-style, 42)", ergo: { rows: 3, cols: 6, thumbs: 3, stagger: [0.5, 0.5, 0.25, 0, 0.25, 0.375] } },
  { id: "lily", name: "Split 4×6 + 4 thumbs (Lily58-style, 56)", ergo: { rows: 4, cols: 6, thumbs: 4, stagger: [0.5, 0.5, 0.25, 0, 0.25, 0.375] } },
  { id: "sweep", name: "Split 3×5 + 2 thumbs (Sweep-style, 34)", ergo: { rows: 3, cols: 5, thumbs: 2, stagger: [0.5, 0.25, 0, 0.25, 0.375] } },
];

const KB_ERGO_LABELS = [["Tab", "Q", "W", "E", "R", "T"], ["Ctrl", "A", "S", "D", "F", "G"], ["Shift", "Z", "X", "C", "V", "B"], ["Esc", "1", "2", "3", "4", "5"]];
const KB_ERGO_LABELS_R = [["Y", "U", "I", "O", "P", "Bksp"], ["H", "J", "K", "L", ";", "'"], ["N", "M", ",", ".", "/", "Shift"], ["6", "7", "8", "9", "0", "Del"]];

// QMK keycodes guessed from key labels
const KB_KEYCODES = { Esc: "KC_ESC", Tab: "KC_TAB", Caps: "KC_CAPS", Shift: "KC_LSFT", Ctrl: "KC_LCTL", Win: "KC_LGUI", Alt: "KC_LALT", Menu: "KC_APP", Space: "KC_SPC", Enter: "KC_ENT", Backspace: "KC_BSPC", Bksp: "KC_BSPC", Del: "KC_DEL", Ins: "KC_INS", Home: "KC_HOME", End: "KC_END", PgUp: "KC_PGUP", PgDn: "KC_PGDN", Up: "KC_UP", Down: "KC_DOWN", Left: "KC_LEFT", Right: "KC_RGHT", PrtSc: "KC_PSCR", ScrLk: "KC_SCRL", Pause: "KC_PAUS", Num: "KC_NUM", Fn: "MO(1)", Lower: "MO(1)", Raise: "MO(2)",
  "`": "KC_GRV", "-": "KC_MINS", "=": "KC_EQL", "[": "KC_LBRC", "]": "KC_RBRC", "\\": "KC_BSLS", ";": "KC_SCLN", "'": "KC_QUOT", ",": "KC_COMM", ".": "KC_DOT", "/": "KC_SLSH", "#": "KC_NUHS", "*": "KC_PAST", "+": "KC_PPLS" };
function kbKeycode(label, right) {
  const l = (label || "").trim();
  if (/^[A-Z]$/.test(l)) return "KC_" + l;
  if (/^[0-9]$/.test(l)) return "KC_" + l;
  if (/^F([1-9]|1[0-2])$/.test(l)) return "KC_" + l;
  if (right && /^(Shift|Ctrl|Alt|Win)$/.test(l)) return { Shift: "KC_RSFT", Ctrl: "KC_RCTL", Alt: "KC_RALT", Win: "KC_RGUI" }[l];
  return KB_KEYCODES[l] || "KC_NO";
}
