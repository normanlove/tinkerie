// Pinnie — interactive pinout reference inside Wirie — by nrmluvdev
"use strict";

// ---------- function categories ----------
const PCAT = {
  PWR: { label: "Power", col: "#ef4444", info: "Supply pin. Never connect two supplies together, and check the voltage before you wire it." },
  GND: { label: "Ground", col: "#6b7280", info: "0V. Every part in your circuit needs a ground connection back to this." },
  GPIO: { label: "Digital I/O", col: "#22c55e", info: "General-purpose pin: read buttons, switch LEDs, talk to other chips." },
  PWM: { label: "PWM", col: "#a855f7", info: "Can output pulses for fading LEDs, motor speed control and servos." },
  ADC: { label: "Analog in", col: "#f59e0b", info: "Reads a voltage (potentiometers, light sensors, battery level)." },
  DAC: { label: "Analog out", col: "#fb923c", info: "Outputs a real voltage level, not just on/off — handy for audio." },
  I2C: { label: "I²C", col: "#3b82f6", info: "2-wire bus (SDA + SCL) shared by many sensors and screens. Needs pull-up resistors (most modules have them)." },
  SPI: { label: "SPI", col: "#ec4899", info: "Fast 4-wire bus (SCK, MOSI/TX, MISO/RX, CS) for screens, SD cards and radios." },
  UART: { label: "UART / Serial", col: "#14b8a6", info: "Serial port. Wire TX to the other device's RX and RX to its TX." },
  TOUCH: { label: "Touch", col: "#84cc16", info: "Capacitive touch sensing — a bare pad or wire becomes a button." },
  INT: { label: "Interrupt", col: "#38bdf8", info: "Can instantly react to a change (button presses, rotary encoders, pulses)." },
  CAN: { label: "CAN", col: "#0ea5e9", info: "CAN bus for cars and robots (needs a CAN transceiver chip)." },
  USB: { label: "USB", col: "#94a3b8", info: "USB data line — usually already used by the board's USB socket." },
  SYS: { label: "System", col: "#64748b", info: "Reset / enable / run control. Pull low to reset or switch the chip off." },
  WARN: { label: "Careful", col: "#fbbf24", info: "This pin has a gotcha — read the note before using it." },
  LED: { label: "On-board", col: "#eab308", info: "Shared with something on the board (LED, button, flash memory…)." },
};
function tagCat(t) {
  const s = t.toUpperCase();
  if (/⚠/.test(t)) return "WARN";
  if (/^(GND|VSS|AGND)/.test(s)) return "GND";
  if (/^(3V3|3\.3V|5V|VIN|VBUS|VSYS|VCC|VDD|RAW|VBAT|AVCC|AREF|VREF|VB\b|POWER)/.test(s)) return "PWR";
  if (/^(STRAP|INPUT ONLY|BOOT|5V TOLERANT|NOT 5V|NO |FLASH|USED BY|AVOID|ANALOG ONLY|ADC2 \(NOT|SHARED)/.test(s) || /⚠/.test(t)) return "WARN";
  if (/^PWM/.test(s)) return "PWM";
  if (/^(ADC|A\d|ANALOG)/.test(s)) return "ADC";
  if (/^DAC/.test(s)) return "DAC";
  if (/^I2C|SDA|SCL/.test(s)) return "I2C";
  if (/^(SPI|VSPI|HSPI|FSPI|MOSI|MISO|SCK|SS\b|CS\b)/.test(s)) return "SPI";
  if (/^(UART|SERIAL|TX|RX)/.test(s)) return "UART";
  if (/^TOUCH/.test(s)) return "TOUCH";
  if (/^(INT|WAKE|RTC)/.test(s)) return "INT";
  if (/^CAN/.test(s)) return "CAN";
  if (/^USB/.test(s)) return "USB";
  if (/^(RUN|EN|RESET|RST|CHIP EN|3V3_EN|SWD|JTAG|XTAL|OSC)/.test(s)) return "SYS";
  if (/^(LED|ON-BOARD|BUTTON)/.test(s)) return "LED";
  if (/^(GPIO|GP\d|D\d|IO\d|DIGITAL)/.test(s)) return "GPIO";
  return "GPIO";
}

// ---------- per-chip function generators ----------
function rp2040(n) {
  const t = [`GPIO${n}`, `PWM ${Math.floor(n / 2) % 8}${n % 2 ? "B" : "A"}`];
  const i2c = (n % 4 < 2 ? 0 : 1), sda = n % 2 === 0; t.push(`I2C${i2c} ${sda ? "SDA" : "SCL"}`);
  const spi = [0, 1, 1, 0][Math.floor(n / 8) % 4] === undefined ? 0 : (Math.floor(n / 8) % 2); t.push(`SPI${spi} ${["RX", "CSn", "SCK", "TX"][n % 4]}`);
  const u = [0, 1, 1, 0, 0, 1, 1, 0][Math.floor(n / 4)]; if (u !== undefined) t.push(`UART${u} ${["TX", "RX", "CTS", "RTS"][n % 4]}`);
  if (n >= 26 && n <= 29) t.push(`ADC${n - 26}`);
  t.push("INT (any edge)");
  return t;
}
const ESP32_ADC1 = { 36: 0, 37: 1, 38: 2, 39: 3, 32: 4, 33: 5, 34: 6, 35: 7 };
const ESP32_ADC2 = { 4: 0, 0: 1, 2: 2, 15: 3, 13: 4, 12: 5, 14: 6, 27: 7, 25: 8, 26: 9 };
const ESP32_TOUCH = { 4: 0, 0: 1, 2: 2, 15: 3, 13: 4, 12: 5, 14: 6, 27: 7, 33: 8, 32: 9 };
function esp32(n) {
  const t = [`GPIO${n}`];
  const inOnly = n >= 34 && n <= 39;
  if (!inOnly) t.push("PWM (LEDC)");
  if (ESP32_ADC1[n] !== undefined) t.push(`ADC1_CH${ESP32_ADC1[n]}`);
  if (ESP32_ADC2[n] !== undefined) t.push(`ADC2_CH${ESP32_ADC2[n]}`, "ADC2 (not while WiFi is on) ⚠");
  if (ESP32_TOUCH[n] !== undefined) t.push(`TOUCH${ESP32_TOUCH[n]}`);
  if (n === 25) t.push("DAC1"); if (n === 26) t.push("DAC2");
  const fx = { 21: ["I2C SDA (default)"], 22: ["I2C SCL (default)"], 23: ["VSPI MOSI"], 19: ["VSPI MISO"], 18: ["VSPI SCK"], 5: ["VSPI CS"], 13: ["HSPI MOSI"], 12: ["HSPI MISO"], 14: ["HSPI SCK"], 15: ["HSPI CS"], 1: ["UART0 TX (USB serial)"], 3: ["UART0 RX (USB serial)"], 17: ["UART2 TX"], 16: ["UART2 RX"] };
  if (fx[n]) t.push(...fx[n]);
  if (inOnly) t.push("Input only — no output, no pull-ups ⚠");
  if ([0, 2, 5, 12, 15].includes(n)) t.push({ 0: "Strapping: LOW at boot = flash mode ⚠", 2: "Strapping: must be LOW/floating at boot ⚠", 5: "Strapping pin (outputs PWM at boot) ⚠", 12: "Strapping: HIGH at boot breaks flash voltage ⚠", 15: "Strapping: prints boot log; HIGH at boot ⚠" }[n]);
  if (n === 1 || n === 3) t.push("Used by USB serial — avoid ⚠");
  if (n === 2) t.push("LED on many boards");
  if (n >= 6 && n <= 11) return [`GPIO${n}`, "Flash memory — do not use ⚠"];
  if ([0, 2, 4, 12, 13, 14, 15, 25, 26, 27, 32, 33, 34, 35, 36, 39].includes(n)) t.push("RTC wake from deep sleep");
  return t;
}
function esp32s3(n) {
  const t = [`GPIO${n}`, "PWM (LEDC)", "I2C (any pin)"];
  if (n >= 1 && n <= 10) t.push(`ADC1_CH${n - 1}`); if (n >= 11 && n <= 20) t.push(`ADC2_CH${n - 11}`, "ADC2 (not while WiFi is on) ⚠");
  if (n >= 1 && n <= 14) t.push(`TOUCH${n}`);
  const fx = { 43: ["UART0 TX"], 44: ["UART0 RX"], 19: ["USB D-"], 20: ["USB D+"], 11: ["FSPI MOSI"], 13: ["FSPI MISO"], 12: ["FSPI SCK"], 10: ["FSPI CS"], 48: ["RGB LED (DevKitC-1 v1.1)"], 38: ["RGB LED (DevKitC-1 v1.0)"] };
  if (fx[n]) t.push(...fx[n]);
  if ([0, 3, 45, 46].includes(n)) t.push("Strapping pin ⚠");
  if (n >= 35 && n <= 37) t.push("Used by octal PSRAM on R8 modules ⚠");
  return t;
}
function esp32c3(n) {
  const t = [`GPIO${n}`, "PWM (LEDC)", "I2C (any pin)"];
  if (n <= 4) t.push(`ADC1_CH${n}`); if (n === 5) t.push("ADC2_CH0 ⚠");
  const fx = { 4: ["SPI SCK (default)"], 5: ["SPI MISO (default)"], 6: ["SPI MOSI (default)"], 7: ["SPI CS (default)"], 8: ["I2C SDA (Arduino default)", "LED (SuperMini)"], 9: ["I2C SCL (Arduino default)", "BOOT button"], 20: ["UART0 RX"], 21: ["UART0 TX"] };
  if (fx[n]) t.push(...fx[n]);
  if ([2, 8, 9].includes(n)) t.push("Strapping pin ⚠");
  if (n <= 5) t.push("RTC wake from deep sleep");
  return t;
}
function esp8266(n) {
  if (n === "A0") return ["ADC0 (0–3.3V on boards, 0–1V bare chip)"];
  const t = [`GPIO${n}`];
  if (n !== 16) t.push("PWM (software)", "INT");
  const fx = { 16: ["Wake from deep sleep (wire to RST)", "No PWM / I2C / interrupts ⚠"], 5: ["I2C SCL (default)"], 4: ["I2C SDA (default)"], 0: ["Boot: LOW = flash mode ⚠", "Pulled up"], 2: ["Boot: must be HIGH ⚠", "LED (D1 Mini/ESP-12)"], 14: ["SPI SCK"], 12: ["SPI MISO"], 13: ["SPI MOSI"], 15: ["SPI CS", "Boot: must be LOW ⚠"], 1: ["UART0 TX (USB serial)", "Boot debug output ⚠"], 3: ["UART0 RX (USB serial)"] };
  if (fx[n]) t.push(...fx[n]);
  return t;
}
const AVR_PWM = [3, 5, 6, 9, 10, 11];
function arduino328(d) { // d = "D7" | "A3" | "A6"
  if (/^A[67]$/.test(d)) return [`ADC${d.slice(1)}`, "Analog only — can't be digital ⚠"];
  const t = [];
  if (/^A\d$/.test(d)) { const a = +d.slice(1); t.push(`D${14 + a} (digital)`, `ADC${a}`); if (a === 4) t.push("I2C SDA"); if (a === 5) t.push("I2C SCL"); t.push("Pin-change INT"); return t; }
  const n = +d.slice(1); t.push(`Digital ${n}`);
  if (AVR_PWM.includes(n)) t.push(`PWM (Timer${n === 5 || n === 6 ? 0 : n === 9 || n === 10 ? 1 : 2})`);
  if (n === 2) t.push("INT0"); if (n === 3) t.push("INT1");
  if (n === 0) t.push("UART RX (USB serial) ⚠"); if (n === 1) t.push("UART TX (USB serial) ⚠");
  if (n === 10) t.push("SPI SS"); if (n === 11) t.push("SPI MOSI"); if (n === 12) t.push("SPI MISO"); if (n === 13) t.push("SPI SCK", "LED (L)");
  t.push("Pin-change INT");
  return t;
}
function atmega32u4(d) {
  const m = { D0: ["Digital 0", "UART1 RX", "INT2"], D1: ["Digital 1", "UART1 TX", "INT3"], D2: ["Digital 2", "I2C SDA", "INT1"], D3: ["Digital 3", "I2C SCL", "PWM", "INT0"], D4: ["Digital 4", "ADC (A6)"], D5: ["Digital 5", "PWM"], D6: ["Digital 6", "PWM", "ADC (A7)"], D7: ["Digital 7", "INT6"], D8: ["Digital 8", "ADC (A8)"], D9: ["Digital 9", "PWM", "ADC (A9)"], D10: ["Digital 10", "PWM", "ADC (A10)"], D14: ["Digital 14", "SPI MISO"], D15: ["Digital 15", "SPI SCK"], D16: ["Digital 16", "SPI MOSI"], A0: ["ADC0", "Digital 18"], A1: ["ADC1", "Digital 19"], A2: ["ADC2", "Digital 20"], A3: ["ADC3", "Digital 21"], A4: ["ADC4", "Digital 22"], A5: ["ADC5", "Digital 23"], D11: ["Digital 11", "PWM"], D12: ["Digital 12", "ADC (A11)"], D13: ["Digital 13", "PWM", "LED"] };
  return m[d] || null;
}
function stm32f1(p) { // "A9"
  const t = [`P${p}`]; const port = p[0], n = +p.slice(1);
  const pwm = { A0: "TIM2_CH1", A1: "TIM2_CH2", A2: "TIM2_CH3", A3: "TIM2_CH4", A6: "TIM3_CH1", A7: "TIM3_CH2", A8: "TIM1_CH1", A9: "TIM1_CH2", A10: "TIM1_CH3", B0: "TIM3_CH3", B1: "TIM3_CH4", B6: "TIM4_CH1", B7: "TIM4_CH2", B8: "TIM4_CH3", B9: "TIM4_CH4" };
  if (pwm[p]) t.push(`PWM ${pwm[p]}`);
  if ((port === "A" && n <= 7) || p === "B0" || p === "B1") t.push(`ADC CH${port === "A" ? n : n + 8}`);
  const fx = { A9: ["UART1 TX"], A10: ["UART1 RX"], A2: ["UART2 TX"], A3: ["UART2 RX"], B10: ["UART3 TX", "I2C2 SCL"], B11: ["UART3 RX", "I2C2 SDA"], B6: ["I2C1 SCL"], B7: ["I2C1 SDA"], A4: ["SPI1 NSS"], A5: ["SPI1 SCK"], A6: ["SPI1 MISO"], A7: ["SPI1 MOSI"], B12: ["SPI2 NSS"], B13: ["SPI2 SCK"], B14: ["SPI2 MISO"], B15: ["SPI2 MOSI"], A11: ["USB D-", "CAN RX"], A12: ["USB D+", "CAN TX"], C13: ["LED (active low)", "Weak output (3mA) ⚠"], C14: ["OSC32 IN ⚠"], C15: ["OSC32 OUT ⚠"], B3: ["JTAG TDO (disable JTAG to use) ⚠"], B4: ["JTAG NJTRST ⚠"], A15: ["JTAG TDI ⚠"] };
  if (fx[p]) t.push(...fx[p]);
  if (!((port === "A" && n <= 7) || p === "B0" || p === "B1" || port === "C")) t.push("5V tolerant");
  t.push("INT (EXTI)");
  return t;
}
function stm32f4(p) { const t = stm32f1(p).filter(x => !/^PWM|5V tolerant|CAN|JTAG/.test(x)); const pwm = { A0: "TIM2_CH1", A1: "TIM2_CH2", A2: "TIM2_CH3", A3: "TIM2_CH4", A6: "TIM3_CH1", A7: "TIM3_CH2", A8: "TIM1_CH1", A9: "TIM1_CH2", A10: "TIM1_CH3", B0: "TIM3_CH3", B1: "TIM3_CH4", B6: "TIM4_CH1", B7: "TIM4_CH2", B8: "TIM4_CH3", B9: "TIM4_CH4", A15: "TIM2_CH1", B3: "TIM2_CH2", B4: "TIM3_CH1", B5: "TIM3_CH2", B10: "TIM2_CH3" }; if (pwm[p]) t.splice(1, 0, `PWM ${pwm[p]}`); if (p === "A0") t.push("KEY button (Black Pill)"); return t; }
function teensy4(n) {
  const t = [`Digital ${n}`];
  if ([0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 18, 19, 22, 23].includes(n)) t.push("PWM");
  if (n >= 14 && n <= 23) t.push(`ADC A${n - 14}`);
  const fx = { 0: ["UART1 RX", "CAN2 RX"], 1: ["UART1 TX", "CAN2 TX"], 7: ["UART2 RX"], 8: ["UART2 TX"], 15: ["UART3 RX"], 14: ["UART3 TX"], 16: ["UART4 RX", "I2C1 SCL"], 17: ["UART4 TX", "I2C1 SDA"], 21: ["UART5 RX"], 20: ["UART5 TX"], 18: ["I2C0 SDA"], 19: ["I2C0 SCL"], 10: ["SPI CS"], 11: ["SPI MOSI"], 12: ["SPI MISO"], 13: ["SPI SCK", "LED"], 22: ["CAN1 TX"], 23: ["CAN1 RX"] };
  if (fx[n]) t.push(...fx[n]);
  t.push("INT (any pin)");
  return t;
}
function samd21xiao(d) {
  const n = +d.slice(1); const t = [`Digital ${n}`, `ADC A${n}`, "INT"];
  if (n === 0) t.push("DAC (A0)");
  const fx = { 4: ["I2C SDA"], 5: ["I2C SCL"], 6: ["UART TX"], 7: ["UART RX"], 8: ["SPI SCK"], 9: ["SPI MISO"], 10: ["SPI MOSI"] };
  if (fx[n]) t.push(...fx[n]);
  t.push("PWM (most pins)");
  return t;
}
function attiny85(pb) { return ({ PB0: ["PB0", "PWM (OC0A)", "SPI MOSI / DI", "I2C SDA", "PCINT0"], PB1: ["PB1", "PWM (OC0B)", "SPI MISO / DO", "PCINT1"], PB2: ["PB2", "ADC1", "SPI SCK", "I2C SCL", "INT0"], PB3: ["PB3", "ADC3", "XTAL1", "PCINT3"], PB4: ["PB4", "ADC2", "PWM (OC1B)", "XTAL2", "PCINT4"], PB5: ["RESET", "ADC0", "Used for reset — reprogramming needs HVSP if disabled ⚠"] })[pb]; }
function piGpio(n) {
  const fx = { 2: ["I2C1 SDA", "Has 1.8k pull-up ⚠"], 3: ["I2C1 SCL", "Has 1.8k pull-up ⚠"], 4: ["GPCLK0", "1-Wire (default)"], 14: ["UART TXD", "Serial console ⚠"], 15: ["UART RXD", "Serial console ⚠"], 18: ["PWM0", "PCM CLK"], 12: ["PWM0"], 13: ["PWM1"], 19: ["PWM1", "PCM FS", "SPI1 MISO"], 10: ["SPI0 MOSI"], 9: ["SPI0 MISO"], 11: ["SPI0 SCLK"], 8: ["SPI0 CE0"], 7: ["SPI0 CE1"], 16: ["SPI1 CE2"], 20: ["PCM DIN", "SPI1 MOSI"], 21: ["PCM DOUT", "SPI1 SCLK"], 17: ["SPI1 CE1"], 26: [], 5: [], 6: [], 22: [], 23: [], 24: [], 25: [], 27: [] };
  return [`GPIO${n}`, ...(fx[n] || []), "INT (any pin)"];
}

// pin name → function tags, per board
function powerTags(name) {
  const n = name.toUpperCase();
  if (/^(GND|VSS)/.test(n)) return ["GND"];
  if (n === "3V3_EN") return ["3V3_EN: pull low to switch the 3.3V regulator off"];
  if (n === "ADC_VREF") return ["VREF for the ADC (filtered 3.3V)"];
  if (n === "AREF") return ["AREF: analog reference voltage"];
  if (/^(VBUS)$/.test(n)) return ["VBUS: 5V straight from USB"];
  if (/^(VSYS)$/.test(n)) return ["VSYS: main input, 1.8–5.5V (battery goes here)"];
  if (/^(VIN|RAW)$/.test(n)) return [`${name}: input to the on-board regulator`];
  if (/^(5V|\+5V)$/.test(n)) return ["5V rail"];
  if (/^(3V3|3\.3V|VCC|VDD|3V)$/.test(n)) return [`${name} rail${/3/.test(n) ? " (3.3V)" : ""}`];
  if (/^(VB|VBAT)$/.test(n)) return ["VBAT: backup battery for the RTC"];
  if (/^(RUN)$/.test(n)) return ["RUN: pull low to reset"];
  if (/^(EN|CHIP_EN|CH_PD)$/.test(n)) return ["EN: chip enable — pull low to reset"];
  if (/^(RST|RESET|R)$/.test(n)) return ["RESET (active low)"];
  return null;
}
function pinnieTags(fp, pad) {
  const id = fp.castellatedOf || fp.id, nm = pad.name; const first = nm.split("/")[0]; // castellated twins have the same pins
  const pw = powerTags(nm); if (pw) return pw;
  if (/^(SD[0-3]|CMD|CLK|CS0|MISO|MOSI|SCLK)$/.test(nm) && /^(nodemcu|esp12f|esp32devkitc38)$/.test(id)) return [nm, "Flash memory — do not use ⚠"];
  if (nm === "AVCC") return ["AVCC: powers the ADC — connect to VCC"];
  if (/^XTAL/.test(nm) || /XTAL[12]/.test(nm)) return [nm, "XTAL: crystal pin (16MHz + 22pF caps)"];
  let t = null;
  if (/^(pico|picow|pico2|pico2w|pico_smd|rp2040zero)$/.test(id) && /^GP\d+$/.test(nm)) { const n = +nm.slice(2); t = rp2040(n); if (id === "rp2040zero" && n === 16) t.push("RGB LED (WS2812)"); }
  else if (/^esp32devkit30$/.test(id)) { const m = { VP: 36, VN: 39, TX0: 1, RX0: 3, TX2: 17, RX2: 16 }; const n = m[nm] ?? (/^D\d+$/.test(nm) ? +nm.slice(1) : null); if (n !== null) t = esp32(n); }
  else if (/^(esp32devkitc38|esp32wroom|esp32cam)$/.test(id)) { const m = { VP: 36, VN: 39, SENSOR_VP: 36, SENSOR_VN: 39, TX: 1, RX: 3, TXD0: 1, RXD0: 3, U0T: 1, U0R: 3, SD2: 9, SD3: 10, CMD: 11, CLK: 6, SD0: 7, SD1: 8 }; const n = m[nm] ?? (/^IO\d+$/.test(nm) ? +nm.slice(2) : null); if (n !== null) { t = esp32(n); if (id === "esp32cam") { if (n === 4) t.push("Camera flash LED"); if ([2, 4, 12, 13, 14, 15].includes(n)) t.push("Shared with SD card slot ⚠"); if (n === 16) t.push("Used by PSRAM ⚠"); if (n === 0) t.push("Camera clock — and flash mode"); } } }
  else if (id === "esp32s3devkit" && /^\d+$/.test(nm)) t = esp32s3(+nm);
  else if (id === "esp32c3sm" && /^GPIO\d+$/.test(nm)) t = esp32c3(+nm.slice(4));
  else if (/^(d1mini|nodemcu)$/.test(id)) { const m = { D0: 16, D1: 5, D2: 4, D3: 0, D4: 2, D5: 14, D6: 12, D7: 13, D8: 15, RX: 3, TX: 1, A0: "A0", SD3: 10, SD2: 9 }; if (m[nm] !== undefined) t = m[nm] === 10 || m[nm] === 9 ? ["Flash memory — do not use ⚠"] : esp8266(m[nm]); if (t && m[nm] !== "A0") t[0] = `GPIO${m[nm]}`; }
  else if (/^(esp12f|esp01)$/.test(id)) { const n = /^GPIO(\d+)$/.test(nm) ? +nm.slice(4) : /^IO(\d+)$/.test(nm) ? +nm.slice(2) : nm === "TXD0" || nm === "TX" ? 1 : nm === "RXD0" || nm === "RX" ? 3 : nm === "ADC" ? "A0" : null; if (n !== null) t = esp8266(n); }
  else if (/^(nano|nanoevery|promini|arduino_micro)$/.test(id) || id === "atmega328p") {
    let d = first.replace(/^(TX1|TXO)$/, "D1").replace(/^(RX0|RXI)$/, "D0");
    if (id === "arduino_micro") { t = atmega32u4(d); }
    else if (/^[DA]\d+$/.test(d)) t = arduino328(d);
    if (t && /^(nanoevery)$/.test(id)) t.push("Nano Every/33/ESP32 differ — check your board ⚠");
    if (id === "atmega328p" && !t) { if (/XTAL/.test(nm)) t = ["XTAL: 16MHz crystal + 22pF caps"]; }
  }
  else if (id === "atmega328pau") { const m = nm.match(/^P([BCD])(\d)/); if (m) { const n = +m[2]; const d = m[1] === "D" ? `D${n}` : m[1] === "B" ? `D${8 + n}` : `A${n}`; t = m[1] === "B" && n >= 6 ? ["XTAL: crystal pins ⚠"] : m[1] === "C" && n === 6 ? ["RESET (active low)"] : arduino328(d); } else if (/^ADC[67]$/.test(nm)) t = arduino328("A" + nm.slice(3)); }
  else if (id === "promicro") { const d = nm.replace(/^TX0$/, "D1").replace(/^RX1$/, "D0"); t = atmega32u4(d); }
  else if (/^(attiny85|attiny85s)$/.test(id)) { const m = nm.match(/PB\d/); if (m) t = attiny85(m[0]); }
  else if (id === "teensy40" && /^\d+$/.test(nm)) t = teensy4(+nm);
  else if (id === "bluepill" && /^[ABC]\d+$/.test(nm)) t = stm32f1(nm);
  else if (id === "blackpill" && /^[ABC]\d+$/.test(nm)) t = stm32f4(nm);
  else if (id === "xiao" && /^D\d+$/.test(nm)) t = samd21xiao(nm);
  else if (id === "pi40") { if (/^GPIO\d+$/.test(nm)) t = piGpio(+nm.slice(4)); else if (/^ID_S[DC]$/.test(nm)) t = [nm, "Reserved for HAT ID EEPROM — don't use ⚠"]; }
  if (t) return t;
  // generic fallback from the footprint data
  const g = [nm];
  if (pad.func && pad.func !== "SIG") g.push({ PWR3: "3.3V power", PWR5: "5V power", PWR: "Power", VIN: "VIN power", SDA: "I2C SDA", SCL: "I2C SCL", MOSI: "SPI MOSI", MISO: "SPI MISO", SCK: "SPI SCK", CS: "SPI CS", TX: "UART TX", RX: "UART RX", ADC: "ADC" }[pad.func] || pad.func);
  for (const part of nm.split("/").slice(1)) g.push(/SDA|SCL/.test(part) ? "I2C " + part : /MOSI|MISO|SCK|SS/.test(part) ? "SPI " + part : /TX|RX/.test(part) ? "UART " + part : part);
  return g;
}
const PINNIE_BOARDS = [
  ["pico", "Boards", "3.3V logic · RP2040 dual-core 133MHz · ~12mA per pin (50mA total) · any GPIO can be any PWM; I²C/SPI/UART on the pins shown"],
  ["picow", "Boards", "Same as Pico. GP23/24/25/29 are used by the WiFi chip; the LED is on the WiFi chip ('LED' in code)"],
  ["pico2", "Boards", "RP2350 150MHz, same pinout as Pico · 3.3V logic"],
  ["rp2040zero", "Boards", "RP2040 in a tiny package · RGB LED on GP16 · 3.3V logic"],
  ["esp32devkit30", "Boards", "3.3V logic, 40mA max per pin · use ADC1 pins when WiFi is on · GPIO34–39 are input only · avoid strapping pins for buttons"],
  ["esp32devkitc38", "Boards", "As the 30-pin board plus GPIO0 and the flash pins (SD0–SD3, CMD, CLK) — never use those"],
  ["esp32s3devkit", "Boards", "ESP32-S3: native USB on 19/20, lots of ADC & touch · 3.3V logic"],
  ["esp32c3sm", "Boards", "ESP32-C3 RISC-V, WiFi + BLE · 3.3V · GPIO8 LED is inverted · keep GPIO9 high at boot"],
  ["esp32cam", "Boards", "Most free pins are shared with the SD card slot — use 1-bit SD mode to free some up"],
  ["d1mini", "Boards", "ESP8266 · 3.3V logic · only one analog input (A0) · D3/D4/D8 must be in the right state at boot"],
  ["nodemcu", "Boards", "ESP8266 · same pin rules as the D1 Mini"],
  ["nano", "Boards", "ATmega328P · 5V logic · 20mA per pin (40mA max) · PWM on 3,5,6,9,10,11 · A6/A7 are analog-only"],
  ["promini", "Boards", "ATmega328P · 3.3V/8MHz or 5V/16MHz versions · A4/A5 (I²C) are on inner pads"],
  ["promicro", "Boards", "ATmega32U4 with native USB (keyboards, game controllers) · 5V or 3.3V versions"],
  ["arduino_micro", "Boards", "ATmega32U4 · 5V logic · native USB"],
  ["xiao", "Boards", "Shown for the SAMD21 XIAO. RP2040 / ESP32-C3 / S3 / nRF52 XIAOs share the layout but some functions differ"],
  ["teensy40", "Boards", "600MHz ARM · 3.3V and NOT 5V tolerant · loads of PWM & serial ports"],
  ["bluepill", "Boards", "STM32F103 · 3.3V logic, most pins 5V tolerant · PC13 LED can only sink a few mA"],
  ["blackpill", "Boards", "STM32F401/411 · 3.3V · KEY button on PA0"],
  ["pi40", "Raspberry Pi", "Any Pi with a 40-pin header · 3.3V logic — 5V on a GPIO will damage the Pi · no analog inputs (use an ADS1115)"],
  ["atmega328p", "Chips", "The Arduino Uno chip · Arduino pin numbers shown · needs a 16MHz crystal or set the 8MHz internal clock"],
  ["atmega328pau", "Chips", "Surface-mount ATmega328P — has two extra analog pins (ADC6/ADC7)"],
  ["attiny85", "Chips", "8-pin AVR · 5 usable I/O · program with an Arduino as ISP"],
  ["esp32wroom", "Chips", "Bare ESP32 module — same pin rules as the DevKit"],
  ["esp12f", "Chips", "Bare ESP8266 module · GPIO15 low, GPIO0/2 high and EN high to boot"],
  ["esp01", "Chips", "ESP8266 with just 2 GPIO · 3.3V only"],
];

// ---------- UI ----------
const PIN = { id: "pico", filter: null, hover: null, locked: null, q: "" };
function buildPinnie(root) {
  root.innerHTML = `<div class="pin-head"><div><h1>Pinnie</h1><p>Hover any pin to see everything it can do. Click a function below to light up every pin that has it.</p></div><input id="pin_q" class="calc-search" placeholder="Find a board or chip…"></div>
    <div class="pin-layout"><aside class="pin-boards" id="pin_list"></aside><section class="pin-main"><div class="pin-filters" id="pin_f"></div><div class="pin-stage" id="pin_stage"></div></section><aside class="pin-side" id="pin_side"></aside></div>`;
  root.querySelector("#pin_q").oninput = e => { PIN.q = e.target.value.toLowerCase(); pinnieList(root); };
  pinnieList(root); pinnieDraw(root);
}
function pinnieList(root) {
  const L = root.querySelector("#pin_list"); const groups = {};
  for (const [id, g] of PINNIE_BOARDS) { const fp = LIB_BY_ID[id]; if (!fp) continue; if (PIN.q && !(fp.name + " " + fp.tags).toLowerCase().includes(PIN.q)) continue; (groups[g] = groups[g] || []).push(fp); }
  // custom/user MCUs too
  for (const fp of LIB) if (fp.user && fp.pinLabels && (!PIN.q || fp.name.toLowerCase().includes(PIN.q))) (groups["My Parts"] = groups["My Parts"] || []).push(fp);
  L.innerHTML = Object.entries(groups).map(([g, fps]) => `<h4>${g}</h4>${fps.map(fp => `<button class="pin-b ${fp.id === PIN.id ? "on" : ""}" data-id="${fp.id}"><canvas width="72" height="72"></canvas><span><b>${esc(fp.name)}</b><small>${fp.pads.length} pins</small></span></button>`).join("")}`).join("");
  L.querySelectorAll(".pin-b").forEach(b => { drawFPPreview(b.querySelector("canvas"), LIB_BY_ID[b.dataset.id]); b.onclick = () => { PIN.id = b.dataset.id; PIN.locked = null; PIN.hover = null; pinnieList(root); pinnieDraw(root); }; });
}
// short, prioritised labels for the drawing (the side panel shows everything in full)
const DRAW_ORDER = ["PWM", "ADC", "DAC", "TOUCH", "I2C", "SPI", "UART", "CAN", "USB", "LED", "SYS", "INT", "GPIO", "PWR", "GND"];
function compactTags(p) {
  const out = [], seen = new Set(); let warn = null;
  const clean = t => t.replace(/\s*⚠/g, "").replace(/\s*\([^)]*\)/g, "").split(" — ")[0].split(":")[0].replace(/^(Digital) (\d+)$/, "D$2").replace(/^RTC wake.*/, "RTC wake").replace(/^INT$/, "INT").replace(/^Pin-change INT$/, "PCINT").trim();
  const list = p.tags.map(t => ({ t, c: tagCat(t) }));
  for (const { t, c } of list) if (c === "WARN" && !warn) warn = "⚠ " + clean(t).replace(/^ADC2.*/, "ADC2+WiFi").replace(/^Used by USB serial.*/, "USB serial").slice(0, 14);
  for (const cat of DRAW_ORDER) for (const { t, c } of list) {
    if (c !== cat) continue; let s = clean(t);
    if (!s || s === p.name || seen.has(s)) continue;
    if (cat === "GPIO" && !/^GPIO\d+$/.test(s)) continue;
    if (cat === "INT" && out.length >= 2) continue;
    if (cat === "PWR" || cat === "GND" || cat === "SYS") s = s.length > 22 ? s.slice(0, 21) + "…" : s;
    seen.add(s); out.push(s);
  }
  const res = out.slice(0, warn ? 3 : 4); if (warn) res.push(warn);
  return res;
}
function pinnieData(fp) {
  const pads = fp.pads.map(p => ({ ...p, tags: pinnieTags(fp, p) }));
  for (const p of pads) p.cats = [...new Set(p.tags.map(tagCat))];
  return pads;
}
function pinnieDraw(root) {
  const fp = LIB_BY_ID[PIN.id]; if (!fp) return;
  const pads = pinnieData(fp);
  // orientation: show long side vertical
  let bw = fp.body.w, bh = fp.body.h; const rot = bw > bh * 1.15;
  const T = (x, y) => (rot ? [-y, x] : [x, y]);
  const pts = pads.map(p => { const [x, y] = T(p.x - fp.body.x, p.y - fp.body.y); return { p, x, y }; });
  if (rot) [bw, bh] = [bh, bw];
  let minPitch = 99; for (let i = 0; i < pts.length; i++) for (let j = i + 1; j < pts.length; j++) { const d = Math.hypot(pts[i].x - pts[j].x, pts[i].y - pts[j].y); if (d > 0.05 && d < minPitch) minPitch = d; }
  const k = Math.min(40, Math.max(24 / minPitch, Math.min(15, Math.max(7, 520 / bh)))); // px per mm
  const padR = Math.max(4, Math.min(k * 0.75, minPitch * k * 0.38));
  const cxs = pts.map(q => q.x);
  const mid = 0;
  const cols = new Set(pts.map(q => Math.round(q.x * 4))).size;
  const sideOf = q => { if (cols <= 2) return q.x < 0 ? "left" : "right"; const ax = Math.abs(q.x) / (bw / 2), ay = Math.abs(q.y) / (bh / 2); if (ay > ax * 1.25 && Math.abs(q.y) > bh / 2 - 3) return q.y < 0 ? "top" : "bottom"; return q.x < mid ? "left" : q.x > mid ? "right" : (pts.indexOf(q) % 2 ? "right" : "left"); };
  const CH = 6.7; // px per char at 11px
  const chipW = t => Math.round(t.replace(" ⚠", "").length * CH * 0.92 + (/⚠/.test(t) ? 28 : 16));
  const shortTag = t => t.length > 26 ? t.slice(0, 24) + "…" : t;
  const maxTags = 5;
  let html = []; let minX = -bw / 2 * k - 20, maxX = bw / 2 * k + 20, minY = -bh / 2 * k - 20, maxY = bh / 2 * k + 20;
  const f = PIN.filter;
  // board body
  const bodyCol = fp.body.col && fp.body.col !== "#0000" ? fp.body.col : "#1d4f91";
  html.push(`<defs><linearGradient id="pbg" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#fff" stop-opacity=".14"/><stop offset="1" stop-color="#000" stop-opacity=".25"/></linearGradient><filter id="psh" x="-20%" y="-20%" width="140%" height="140%"><feDropShadow dx="0" dy="6" stdDeviation="8" flood-opacity=".5"/></filter></defs>`);
  html.push(`<rect x="${-bw / 2 * k}" y="${-bh / 2 * k}" width="${bw * k}" height="${bh * k}" rx="${Math.min(14, k * 1.2)}" fill="${bodyCol}" filter="url(#psh)"/><rect x="${-bw / 2 * k}" y="${-bh / 2 * k}" width="${bw * k}" height="${bh * k}" rx="${Math.min(14, k * 1.2)}" fill="url(#pbg)" stroke="rgba(255,255,255,.18)"/>`);
  const usb = fp.body.usb; if (usb) { let [ux, uy] = T(0, usb === "top" ? -fp.body.h / 2 + 2 : fp.body.h / 2 - 2); html.push(`<rect x="${ux * k - 4.5 * k}" y="${uy * k - 2.8 * k}" width="${9 * k}" height="${5.6 * k}" rx="${0.8 * k}" fill="#c9ccd1" stroke="#8b9098"/><text x="${ux * k}" y="${uy * k + 4}" class="pin-usb">USB</text>`); }
  html.push(`<text x="0" y="0" class="pin-title" transform="rotate(${bh > bw * 1.3 ? -90 : 0})">${esc(fp.value || fp.name)}</text>`);
  for (const q of pts) {
    const p = q.p; const side = sideOf(q);
    const X = q.x * k, Y = q.y * k;
    const on = !f || p.cats.includes(f);
    const isHover = PIN.hover === p.n || PIN.locked === p.n;
    const mainCat = p.cats.find(c => !["WARN"].includes(c)) || "GPIO";
    const tagsShown = compactTags(p).map(shortTag);
    // label row
    const gap = 10; let cur = 0; const parts = [];
    const nameW = Math.max(34, Math.round(p.name.length * CH + 14));
    const dir = side === "left" ? -1 : side === "right" ? 1 : 0;
    const items = [{ t: p.name, w: nameW, cat: mainCat, name: true }, ...tagsShown.map(t => ({ t, w: chipW(t), cat: /^⚠/.test(t) ? "WARN" : tagCat(t) }))];
    if (dir) {
      let x = X + dir * (k * 0.9 + 14);
      const lineEnd = x;
      for (const it of items) { const x0 = dir > 0 ? x : x - it.w; const hl = f && tagCat(it.t) === f; parts.push(`<g class="chip${it.name ? " nm" : ""}${hl ? " hl" : ""}"><rect x="${x0}" y="${Y - 10}" width="${it.w}" height="20" rx="${it.name ? 5 : 10}" style="--c:${PCAT[it.cat].col}"/><text x="${x0 + it.w / 2}" y="${Y + 4}">${esc(it.t)}</text></g>`); x += dir * (it.w + 4); }
      minX = Math.min(minX, x - 10); maxX = Math.max(maxX, x + 10);
      html.push(`<g class="pinrow ${on ? "" : "dim"} ${isHover ? "hov" : ""}" data-n="${esc(p.n)}"><line x1="${X}" y1="${Y}" x2="${lineEnd}" y2="${Y}" class="lead"/>${parts.join("")}<circle cx="${X}" cy="${Y}" r="${padR}" class="pad ${p.shape === "rect" ? "sq" : ""}"/><rect x="${Math.min(X, x)}" y="${Y - 11}" width="${Math.abs(x - X)}" height="22" fill="transparent"/></g>`);
    } else {
      const up = side === "top" ? -1 : 1; let y = Y + up * (k * 0.9 + 14);
      for (const it of items.slice(0, 3)) { const y0 = up > 0 ? y : y - it.w; parts.push(`<g class="chip${it.name ? " nm" : ""}" transform="translate(${X},${y0 + it.w / 2}) rotate(-90)"><rect x="${-it.w / 2}" y="-10" width="${it.w}" height="20" rx="${it.name ? 5 : 10}" style="--c:${PCAT[it.cat].col}"/><text x="0" y="4">${esc(it.t)}</text></g>`); y += up * (it.w + 4); }
      minY = Math.min(minY, y - 10); maxY = Math.max(maxY, y + 10);
      html.push(`<g class="pinrow ${on ? "" : "dim"} ${isHover ? "hov" : ""}" data-n="${esc(p.n)}"><line x1="${X}" y1="${Y}" x2="${X}" y2="${Y + up * (k * 0.9 + 14)}" class="lead"/>${parts.join("")}<circle cx="${X}" cy="${Y}" r="${padR}" class="pad"/></g>`);
    }
  }
  const W = maxX - minX, H = maxY - minY;
  const stage = root.querySelector("#pin_stage");
  stage.innerHTML = `<svg viewBox="${minX} ${minY} ${W} ${H}" style="width:${Math.round(W)}px;min-width:${Math.round(W)}px">${html.join("")}</svg>`;
  stage.querySelectorAll(".pinrow").forEach(g => {
    g.onmouseenter = () => { PIN.hover = g.dataset.n; pinnieSide(root, fp, pads); stage.querySelectorAll(".pinrow").forEach(x => x.classList.toggle("hov", x.dataset.n === PIN.hover || x.dataset.n === PIN.locked)); };
    g.onmouseleave = () => { PIN.hover = null; pinnieSide(root, fp, pads); stage.querySelectorAll(".pinrow").forEach(x => x.classList.toggle("hov", x.dataset.n === PIN.locked)); };
    g.onclick = () => { PIN.locked = PIN.locked === g.dataset.n ? null : g.dataset.n; pinnieSide(root, fp, pads); stage.querySelectorAll(".pinrow").forEach(x => x.classList.toggle("hov", x.dataset.n === PIN.locked)); };
  });
  // filters with counts
  const counts = {}; for (const p of pads) for (const c of p.cats) counts[c] = (counts[c] || 0) + 1;
  if (PIN.filter && !counts[PIN.filter]) { PIN.filter = null; return pinnieDraw(root); }
  const F = root.querySelector("#pin_f");
  F.innerHTML = Object.keys(PCAT).filter(c => counts[c]).map(c => `<button class="pf ${f === c ? "on" : ""}" data-c="${c}" style="--c:${PCAT[c].col}"><i></i>${PCAT[c].label}<em>${counts[c]}</em></button>`).join("");
  F.querySelectorAll(".pf").forEach(b => b.onclick = () => { PIN.filter = PIN.filter === b.dataset.c ? null : b.dataset.c; pinnieDraw(root); });
  pinnieSide(root, fp, pads);
}
function pinnieSide(root, fp, pads) {
  const S2 = root.querySelector("#pin_side"); const n = PIN.locked || PIN.hover; const p = pads.find(q => q.n === n);
  const note = (PINNIE_BOARDS.find(b => b[0] === fp.id) || [])[2] || fp.desc || "";
  if (!p) {
    S2.innerHTML = `<div class="ps-card"><h3>${esc(fp.name)}</h3><p class="dim">${esc(note)}</p><div class="ps-stats"><div><b>${pads.length}</b><span>pins</span></div><div><b>${pads.filter(q => q.cats.includes("PWM")).length}</b><span>PWM</span></div><div><b>${pads.filter(q => q.cats.includes("ADC")).length}</b><span>analog</span></div></div>
      <button class="btn primary wide" id="ps_place">${ico("place")}Put it on my PCB</button>${fp.verify ? `<div class="callout warn">${ico("info")}<span>Pin positions vary between makers — check yours.</span></div>` : ""}<p class="note">Tip: click a pin to keep its details on screen.</p></div>`;
  } else {
    S2.innerHTML = `<div class="ps-card"><div class="ps-pin"><span class="ps-name">${esc(p.name)}</span><span class="dim">pin ${esc(p.n)} · ${p.drill ? "through-hole" : "surface-mount"}</span></div>
      <div class="ps-tags">${p.tags.map(t => { const c = tagCat(t); return `<div class="ps-tag" style="--c:${PCAT[c].col}"><i></i><div><b>${esc(t)}</b><small>${PCAT[c].label}</small></div></div>`; }).join("")}</div>
      <h4>What that means</h4>${[...new Set(p.tags.map(tagCat))].map(c => `<p class="ps-info"><b style="color:${PCAT[c].col}">${PCAT[c].label}:</b> ${PCAT[c].info}</p>`).join("")}</div>`;
  }
  const pl = S2.querySelector("#ps_place"); if (pl) pl.onclick = () => { switchMode("pcb"); startPlacing(fp.id); toast(`Click on your board to place the ${esc(fp.name)}.`); };
}
