// Keebie footprints: keyboard switches, hot-swap sockets, stabilisers, the
// TRRS jack and the bare chips custom keyboards are built on (with their real
// pin names, so Keebie can wire the support circuitry).
// Part of Wirie, in the Tinkerie maker suite — by nrmluvdev.
"use strict";

const KB = "Keyboard";

// ---------------- switches ----------------
// Footprints are centred on the switch; pin 1 and pin 2 are the two switch contacts.
addFP({ id: "kb_mx_hs", name: "Cherry MX switch, Kailh hot-swap socket", cat: KB, prefix: "SW", value: "MX hot-swap", desc: "Socket soldered on the back; switches push in without soldering", tags: "cherry mx hotswap hot-swap kailh socket keyboard key switch",
  pads: [smd(1, -7.085, -2.54, 2.55, 2.5, { name: "1", opp: true }), smd(2, 5.842, -5.08, 2.55, 2.5, { name: "2", opp: true })],
  holes: [{ x: 0, y: 0, d: 4.0 }, { x: -5.08, y: 0, d: 1.75 }, { x: 5.08, y: 0, d: 1.75 }, { x: -3.81, y: -2.54, d: 3.0 }, { x: 2.54, y: -5.08, d: 3.0 }],
  body: { x: 0, y: 0, w: 14, h: 14, col: "#444" } });
addFP({ id: "kb_choc_hs", name: "Kailh Choc v1, hot-swap socket", cat: KB, prefix: "SW", value: "Choc hot-swap", desc: "Low-profile Choc switch with a Kailh PG1350 socket on the back", tags: "kailh choc low profile hotswap hot-swap socket keyboard key switch",
  pads: [smd(1, -3.275, -5.95, 2.6, 2.6, { name: "1", opp: true }), smd(2, 8.275, -3.75, 2.6, 2.6, { name: "2", opp: true })],
  holes: [{ x: 0, y: 0, d: 3.429 }, { x: -5.5, y: 0, d: 1.9 }, { x: 5.5, y: 0, d: 1.9 }, { x: 0, y: -5.95, d: 3.0 }, { x: 5, y: -3.75, d: 3.0 }],
  body: { x: 0, y: 0, w: 15, h: 15, col: "#444" } });
addFP({ id: "kb_alps", name: "Alps SKCM/SKCL switch (and Matias)", cat: KB, prefix: "SW", value: "Alps", desc: "Vintage Alps and modern Matias switches", tags: "alps skcm skcl matias keyboard key switch vintage", verify: true,
  pads: [tht(1, -2.5, -4.0, { d: 2.2, drill: 1.3, name: "1" }), tht(2, 2.5, -4.5, { d: 2.2, drill: 1.3, name: "2" })],
  body: { x: 0, y: 0, w: 15.5, h: 12.8, col: "#444" } });

// PCB-mount (screw-in) stabilisers for long keys. spacing = distance between the two wire mounts.
const STABS = { "2": 23.876, "3": 38.1, "6": 95.25, "6.25": 100.0, "7": 114.3, "8": 133.35 };
function stabSpacing(units) { if (units >= 8) return STABS["8"]; if (units >= 7) return STABS["7"]; if (units >= 6.25) return STABS["6.25"]; if (units >= 6) return STABS["6"]; if (units >= 3) return STABS["3"]; return STABS["2"]; }
for (const [u, s] of Object.entries(STABS)) addFP({ id: "kb_stab_" + u.replace(".", "_"), name: `Stabiliser holes (${u === "2" ? "2–2.75" : u}u)`, cat: KB, prefix: "ST", value: `Stab ${u}u`, desc: "Holes for PCB-mount (screw-in) Cherry-style stabilisers", tags: "stabiliser stabilizer stab spacebar shift enter keyboard", pads: [],
  holes: [{ x: -s / 2, y: -6.985, d: 3.05 }, { x: s / 2, y: -6.985, d: 3.05 }, { x: -s / 2, y: 8.255, d: 3.988 }, { x: s / 2, y: 8.255, d: 3.988 }], silk: "holes" });

// ---------------- connectors ----------------
addFP({ id: "kb_trrs", name: "TRRS jack PJ-320A (split keyboards)", cat: KB, prefix: "J", value: "TRRS", desc: "4-pole 3.5mm jack joining the two halves of a split keyboard. Never plug or unplug it while USB is connected", tags: "trrs jack pj320a pj-320a split keyboard 3.5mm", verify: true,
  pads: [tht(1, -2.3, 1.6, { d: 1.7, drill: 1.0, shape: "oval", w: 1.7, h: 2.4, name: "SLEEVE" }), tht(2, 2.3, 3.2, { d: 1.7, drill: 1.0, shape: "oval", w: 1.7, h: 2.4, name: "TIP" }), tht(3, 2.3, 6.2, { d: 1.7, drill: 1.0, shape: "oval", w: 1.7, h: 2.4, name: "RING1" }), tht(4, 2.3, 10.2, { d: 1.7, drill: 1.0, shape: "oval", w: 1.7, h: 2.4, name: "RING2" })],
  holes: [{ x: 0, y: 1.6, d: 1.2 }, { x: 0, y: 8.6, d: 1.2 }], body: { x: 0, y: 5.6, w: 6, h: 12.1, col: "#1b1b1b" } });
addFP({ id: "kb_fuse1206", name: "PTC resettable fuse 1206 (500mA)", cat: KB, prefix: "F", value: "500mA PTC", desc: "Protects the USB port if something shorts", tags: "fuse ptc polyfuse resettable 1206", pads: smd2("1206"), body: body2("1206", "#d4a017"), silk: "sides" });

// ---------------- bare keyboard chips ----------------
function namedQFP(id, name, n, pitch, span, pw, ph, names, bodyW, ep, desc, tags) {
  const pads = qfpPads(n, pitch, span, pw, ph).map((p, i) => Object.assign(p, { name: names[i] }));
  if (ep) pads.push(smd(n + 1, 0, 0, ep, ep, { name: names[n] }));
  return addFP({ id, name, cat: KB, prefix: "U", value: name.split(" ")[0], desc, tags, verify: true, pads, body: { x: 0, y: 0, w: bodyW, h: bodyW, col: "#1a1a1a", notch: true }, silk: "none", pinLabels: true, mcu: true });
}
namedQFP("kb_atmega32u4", "ATmega32U4-AU (TQFP-44)", 44, 0.8, 11.4, 1.5, 0.55,
  ["PE6", "UVCC", "D-", "D+", "UGND", "UCAP", "VBUS", "PB0", "PB1", "PB2", "PB3", "PB7", "RESET", "VCC", "GND", "XTAL2", "XTAL1", "PD0", "PD1", "PD2", "PD3", "PD5", "GND", "AVCC", "PD4", "PD6", "PD7", "PB4", "PB5", "PB6", "PC6", "PC7", "PE2", "VCC", "GND", "PF7", "PF6", "PF5", "PF4", "PF1", "PF0", "AREF", "GND", "AVCC"],
  10, 0, "The chip inside the Pro Micro: native USB, the classic QMK keyboard chip", "atmega32u4 32u4 avr usb keyboard qmk");
namedQFP("kb_rp2040", "RP2040 (QFN-56, 7x7mm)", 56, 0.4, 6.8, 0.85, 0.2,
  ["IOVDD", "GPIO0", "GPIO1", "GPIO2", "GPIO3", "GPIO4", "GPIO5", "GPIO6", "GPIO7", "IOVDD", "GPIO8", "GPIO9", "GPIO10", "GPIO11", "GPIO12", "GPIO13", "GPIO14", "GPIO15", "TESTEN", "XIN", "XOUT", "IOVDD", "DVDD", "SWCLK", "SWD", "RUN", "GPIO16", "GPIO17", "GPIO18", "GPIO19", "GPIO20", "GPIO21", "IOVDD", "GPIO22", "GPIO23", "GPIO24", "GPIO25", "GPIO26", "GPIO27", "GPIO28", "GPIO29", "IOVDD", "ADC_AVDD", "VREG_IN", "VREG_VOUT", "USB_DM", "USB_DP", "USB_VDD", "IOVDD", "DVDD", "QSPI_SD3", "QSPI_SCLK", "QSPI_SD0", "QSPI_SD2", "QSPI_SD1", "QSPI_SS", "GND"],
  7, 3.2, "Raspberry Pi's chip: needs a QSPI flash chip, 12MHz crystal and 3.3V regulator", "rp2040 raspberry pi qfn56 usb keyboard qmk");
const STM32_48 = (vcap) => ["VBAT", "PC13", "PC14", "PC15", vcap ? "PH0" : "PF0", vcap ? "PH1" : "PF1", "NRST", "VSSA", "VDDA", "PA0", "PA1", "PA2", "PA3", "PA4", "PA5", "PA6", "PA7", "PB0", "PB1", "PB2", "PB10", vcap ? "VCAP1" : "PB11", "VSS", "VDD", "PB12", "PB13", "PB14", "PB15", "PA8", "PA9", "PA10", "PA11", "PA12", "PA13", "VSS", vcap ? "VDD" : "VDDIO2", "PA14", "PA15", "PB3", "PB4", "PB5", "PB6", "PB7", "BOOT0", "PB8", "PB9", "VSS", "VDD", "VSS"];
namedQFP("kb_stm32f072", "STM32F072CBT6 (LQFP-48)", 48, 0.5, 8.6, 1.2, 0.3, STM32_48(false), 7, 0, "ST's keyboard favourite: crystal-less USB, used by many QMK boards", "stm32 stm32f072 lqfp48 arm usb keyboard qmk");
namedQFP("kb_stm32f411", "STM32F411CEU6 (UFQFPN-48)", 48, 0.5, 6.85, 0.8, 0.28, STM32_48(true), 7, 5.6, "The Black Pill's chip: fast, needs a 25MHz crystal", "stm32 stm32f411 f401 qfn48 arm usb keyboard qmk blackpill");
addFP({ id: "kb_w25q128", name: "W25Q128JVS flash (SOIC-8 208mil)", cat: KB, prefix: "U", value: "W25Q128JVS", desc: "16MB QSPI flash for the RP2040", tags: "flash qspi w25q128 w25q16 winbond spi memory", verify: true,
  pads: soPads(8, 1.27, 7.0, 1.6, 0.6).map((p, i) => Object.assign(p, { name: ["CS", "DO", "IO2", "GND", "DI", "CLK", "IO3", "VCC"][i] })), body: { x: 0, y: 0, w: 5.3, h: 5.3, col: "#1a1a1a", notch: true }, silk: "none", pinLabels: true });
