// Flashie — "Keyboard" tab: give every key, knob and layer of a keyboard
// designed in Keebie (Wirie) its job, then Flashie writes Arduino firmware for
// it (ATmega32U4, RP2040 or STM32), compiles it and flashes it.
// Made by nrmluvdev.
(() => {
const $k = s => document.querySelector(s);
const esc = s => String(s ?? '').replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));

// ------------------------------------------------------------------ keycodes
// t: 'k' = keyboard usage, 'c' = consumer (media) usage
const KC = {};
const G = [];
const def = (group, list) => { G.push(group); for (const [id, label, t, u, desc] of list) KC[id] = { id, label, t, u, group, desc: desc || '' }; };
def('Letters', 'ABCDEFGHIJKLMNOPQRSTUVWXYZ'.split('').map((c, i) => ['KC_' + c, c, 'k', 4 + i]));
def('Numbers', [...'1234567890'].map((c, i) => ['KC_' + c, c, 'k', 30 + i]));
def('Symbols', [['KC_MINS', '-', 'k', 45], ['KC_EQL', '=', 'k', 46], ['KC_LBRC', '[', 'k', 47], ['KC_RBRC', ']', 'k', 48], ['KC_BSLS', '\\', 'k', 49], ['KC_NUHS', '#', 'k', 50, 'The # key next to Enter on UK/ISO keyboards'], ['KC_SCLN', ';', 'k', 51], ['KC_QUOT', "'", 'k', 52], ['KC_GRV', '`', 'k', 53], ['KC_COMM', ',', 'k', 54], ['KC_DOT', '.', 'k', 55], ['KC_SLSH', '/', 'k', 56], ['KC_NUBS', '\\ (ISO)', 'k', 100, 'The extra key left of Z on UK/ISO keyboards']]);
def('Editing', [['KC_ESC', 'Esc', 'k', 41], ['KC_TAB', 'Tab', 'k', 43], ['KC_SPC', 'Space', 'k', 44], ['KC_ENT', 'Enter', 'k', 40], ['KC_BSPC', 'Bksp', 'k', 42], ['KC_DEL', 'Del', 'k', 76], ['KC_INS', 'Ins', 'k', 73], ['KC_CAPS', 'Caps', 'k', 57], ['KC_APP', 'Menu', 'k', 101], ['KC_PSCR', 'PrtSc', 'k', 70], ['KC_SCRL', 'ScrLk', 'k', 71], ['KC_PAUS', 'Pause', 'k', 72]]);
def('Navigation', [['KC_UP', '↑', 'k', 82], ['KC_DOWN', '↓', 'k', 81], ['KC_LEFT', '←', 'k', 80], ['KC_RGHT', '→', 'k', 79], ['KC_HOME', 'Home', 'k', 74], ['KC_END', 'End', 'k', 77], ['KC_PGUP', 'PgUp', 'k', 75], ['KC_PGDN', 'PgDn', 'k', 78]]);
def('Modifiers', [['KC_LSFT', 'Shift', 'k', 0xE1], ['KC_LCTL', 'Ctrl', 'k', 0xE0], ['KC_LALT', 'Alt', 'k', 0xE2], ['KC_LGUI', 'Win', 'k', 0xE3, 'Windows key / Cmd on a Mac'], ['KC_RSFT', 'R Shift', 'k', 0xE5], ['KC_RCTL', 'R Ctrl', 'k', 0xE4], ['KC_RALT', 'AltGr', 'k', 0xE6], ['KC_RGUI', 'R Win', 'k', 0xE7]]);
def('Function', Array.from({ length: 24 }, (_, i) => ['KC_F' + (i + 1), 'F' + (i + 1), 'k', i < 12 ? 58 + i : 104 + i - 12]));
def('Numpad', [['KC_NUM', 'Num', 'k', 83], ['KC_PSLS', 'P/', 'k', 84], ['KC_PAST', 'P*', 'k', 85], ['KC_PMNS', 'P-', 'k', 86], ['KC_PPLS', 'P+', 'k', 87], ['KC_PENT', 'PEnt', 'k', 88], ...[1, 2, 3, 4, 5, 6, 7, 8, 9].map(n => ['KC_P' + n, 'P' + n, 'k', 88 + n]), ['KC_P0', 'P0', 'k', 98], ['KC_PDOT', 'P.', 'k', 99]]);
def('Media', [['KC_VOLU', 'Vol +', 'c', 0xE9], ['KC_VOLD', 'Vol −', 'c', 0xEA], ['KC_MUTE', 'Mute', 'c', 0xE2], ['KC_MPLY', 'Play', 'c', 0xCD], ['KC_MNXT', 'Next', 'c', 0xB5], ['KC_MPRV', 'Prev', 'c', 0xB6], ['KC_MSTP', 'Stop', 'c', 0xB7], ['KC_BRIU', 'Bright +', 'c', 0x6F], ['KC_BRID', 'Bright −', 'c', 0x70], ['KC_CALC', 'Calc', 'c', 0x192], ['KC_MYCM', 'My PC', 'c', 0x194], ['KC_WSCH', 'Search', 'c', 0x221]]);
const MODS = [['LCTL', 'Ctrl', 1], ['LSFT', 'Shift', 2], ['LALT', 'Alt', 4], ['LGUI', 'Win', 8]];

// a keymap entry is a string: KC_A, LCTL(KC_C), MO(1), TG(2), TO(0), KC_TRNS, KC_NO, TEXT:hello
function parse(code) {
  code = String(code || 'KC_NO');
  let m;
  if (code === 'KC_TRNS' || code === '_______') return { kind: 'trns' };
  if (code === 'KC_NO' || code === 'XXXXXXX') return { kind: 'none' };
  if ((m = code.match(/^(MO|TG|TO)\((\d)\)$/))) return { kind: m[1].toLowerCase(), layer: +m[2] };
  if (code.startsWith('TEXT:')) return { kind: 'text', text: code.slice(5) };
  let mods = 0;
  while ((m = code.match(/^(LCTL|LSFT|LALT|LGUI|C|S|A|G)\((.*)\)$/))) { mods |= { LCTL: 1, C: 1, LSFT: 2, S: 2, LALT: 4, A: 4, LGUI: 8, G: 8 }[m[1]]; code = m[2]; }
  const k = KC[code] || KC[{ KC_ENTER: 'KC_ENT', KC_SPACE: 'KC_SPC', KC_ESCAPE: 'KC_ESC', KC_RIGHT: 'KC_RGHT', KC_BACKSPACE: 'KC_BSPC', KC_DELETE: 'KC_DEL' }[code]];
  if (!k) return { kind: 'none' };
  return { kind: k.t === 'c' ? 'media' : 'key', k, mods };
}
function label(code) {
  const p = parse(code);
  switch (p.kind) {
    case 'trns': return '▽';
    case 'none': return '';
    case 'mo': return 'Hold L' + p.layer;
    case 'tg': return 'Tog L' + p.layer;
    case 'to': return 'Go L' + p.layer;
    case 'text': return '“' + p.text.slice(0, 6) + (p.text.length > 6 ? '…' : '') + '”';
  }
  return MODS.filter(m => p.mods & m[2]).map(m => m[1] + '+').join('') + p.k.label;
}
function describe(code) {
  const p = parse(code);
  switch (p.kind) {
    case 'trns': return 'Transparent: uses whatever the layer underneath has here.';
    case 'none': return 'Does nothing.';
    case 'mo': return `While held, keys do what layer ${p.layer} says. Let go to go back.`;
    case 'tg': return `Each press turns layer ${p.layer} on or off.`;
    case 'to': return `Switches to layer ${p.layer} and stays there.`;
    case 'text': return `Types “${p.text}”.`;
  }
  const combo = MODS.filter(m => p.mods & m[2]).map(m => m[1]);
  return (combo.length ? `Holds ${combo.join(' + ')} and presses ` : 'Sends ') + p.k.label + (p.kind === 'media' ? ' (a media key)' : '') + (p.k.desc ? '. ' + p.k.desc : '') + '.';
}

// ------------------------------------------------------------------ state
const K = { list: [], file: null, path: '', map: null, layer: 0, sel: null, slot: 'press', cat: 'Letters', ports: [], port: '', busy: false, code: '', saveT: 0 };
const LAYERS_MAX = 4;

function defaultMap(f) {
  const keys = {}, knobs = {};
  for (const k of f.keys) keys[k.id] = [k.kind === 'knob' ? (k.qmk && k.qmk !== 'KC_NO' ? k.qmk : 'KC_MUTE') : (k.qmk || 'KC_NO'), 'KC_TRNS'];
  for (const n of f.knobs || []) knobs[n.key] = [{ cw: 'KC_VOLU', ccw: 'KC_VOLD' }, { cw: 'KC_TRNS', ccw: 'KC_TRNS' }];
  return {
    v: 1, layers: ['Base', 'Layer 1'], keys, knobs,
    screen: { title: f.name, layer: true, wpm: true, keys: false, flip: false },
    rgb: { mode: 'rainbow', color: '#7c5cff', bright: 40 },
    knob: { steps: 4, flip: false },
    pins: {}, texts: [],
  };
}
function fitMap(m, f) { // keep a saved keymap in step with the keyboard file
  const L = m.layers.length;
  for (const k of f.keys) { m.keys[k.id] = m.keys[k.id] || [k.qmk || 'KC_NO']; while (m.keys[k.id].length < L) m.keys[k.id].push('KC_TRNS'); }
  for (const n of f.knobs || []) { m.knobs[n.key] = m.knobs[n.key] || [{ cw: 'KC_VOLU', ccw: 'KC_VOLD' }]; while (m.knobs[n.key].length < L) m.knobs[n.key].push({ cw: 'KC_TRNS', ccw: 'KC_TRNS' }); }
  m.screen = Object.assign({ title: f.name, layer: true, wpm: true, keys: false, flip: false }, m.screen || {});
  m.rgb = Object.assign({ mode: 'rainbow', color: '#7c5cff', bright: 40 }, m.rgb || {});
  m.knob = Object.assign({ steps: 4, flip: false }, m.knob || {});
  m.pins = m.pins || {};
  return m;
}
function save() {
  clearTimeout(K.saveT);
  K.saveT = setTimeout(() => { if (K.file) fetch('/api/keyboard/keymap?name=' + encodeURIComponent(K.file.name), { method: 'POST', body: JSON.stringify(K.map) }).catch(() => {}); }, 500);
}

// pin overrides: the Wiring section lets people correct a pin by hand
const pinOf = (key, p) => (K.map.pins[key] ?? (p && p.pin)) || '';
function wiring() {
  const f = K.file, rows = f.rows.map((p, i) => ({ key: 'row' + i, label: 'Row ' + i, p })), cols = f.cols.map((p, i) => ({ key: 'col' + i, label: 'Column ' + i, p }));
  const list = [...rows, ...cols];
  (f.knobs || []).forEach((n, i) => { list.push({ key: `k${i}a`, label: `Knob ${i + 1} A`, p: n.a }, { key: `k${i}b`, label: `Knob ${i + 1} B`, p: n.b }); });
  if (f.screen) list.push({ key: 'sda', label: 'Screen SDA', p: f.screen.sda }, { key: 'scl', label: 'Screen SCL', p: f.screen.scl });
  if (f.rgb) list.push({ key: 'led', label: 'LED data', p: f.rgb.pin });
  return list;
}

// ------------------------------------------------------------------ firmware
function encode(code, texts) {
  const p = parse(code);
  switch (p.kind) {
    case 'trns': return 1;
    case 'none': return 0;
    case 'mo': return 0x2000 | p.layer;
    case 'tg': return 0x3000 | p.layer;
    case 'to': return 0x4000 | p.layer;
    case 'text': { let i = texts.indexOf(p.text); if (i < 0) { i = texts.length; texts.push(p.text); } return 0x5000 | i; }
    case 'media': return 0x1000 | p.k.u;
  }
  return (p.mods << 8) | p.k.u;
}
const hex = n => '0x' + n.toString(16).toUpperCase().padStart(4, '0');
const cstr = s => '"' + String(s).replace(/[\\"]/g, c => '\\' + c).replace(/[^\x20-\x7e]/g, '?') + '"';

function problems() {
  const f = K.file, out = [];
  if (!f) return ['Open a keyboard first.'];
  if (f.mcu.wireless) out.push(`The ${f.mcu.name} is a wireless board: it runs ZMK, not Arduino firmware. Use Keebie's QMK / ZMK files instead.`);
  else if (!f.mcu.core) out.push(`Flashie can't make firmware for the ${f.mcu.name}.`);
  if (f.split === 'two') out.push('This is a split keyboard in two halves. Flashie makes firmware for one-piece keyboards; use Keebie\'s QMK files for splits.');
  for (const w of wiring()) if (!String(pinOf(w.key, w.p)).trim()) out.push(`${w.label} has no pin. Set it under Wiring.`);
  return out;
}

function generate() {
  const f = K.file, m = K.map, L = m.layers.length, R = f.rows.length, C = f.cols.length, core = f.mcu.core;
  const texts = [];
  const grid = Array.from({ length: L }, () => Array.from({ length: R }, () => Array(C).fill(0)));
  for (const k of f.keys) if (k.row < R && k.col < C) for (let l = 0; l < L; l++) grid[l][k.row][k.col] = encode(m.keys[k.id][l] || 'KC_TRNS', texts);
  const knobs = f.knobs || [];
  const kcw = knobs.map(n => m.knobs[n.key].map(x => encode(x.cw, texts))), kccw = knobs.map(n => m.knobs[n.key].map(x => encode(x.ccw, texts)));
  const pins = list => list.join(', ');
  const rowPins = f.rows.map((p, i) => pinOf('row' + i, p)), colPins = f.cols.map((p, i) => pinOf('col' + i, p));
  const c2r = f.diode !== 'ROW2COL';
  const scr = f.screen, rgb = f.rgb;
  const sda = scr && pinOf('sda', scr.sda), scl = scr && pinOf('scl', scr.scl);
  const wire = core === 'rp2040' && scr ? ((+sda >> 1) & 1 ? 'Wire1' : 'Wire') : 'Wire';
  const libs = []; if (scr) libs.push('Adafruit SSD1306', 'Adafruit GFX Library'); if (rgb && m.rgb.mode !== 'off') libs.push('Adafruit NeoPixel');
  const useRgb = rgb && m.rgb.mode !== 'off';
  const col = m.rgb.color.replace('#', '');
  const rgbv = [0, 2, 4].map(i => parseInt(col.slice(i, i + 2), 16) || 0);
  const lines = [];
  const o = s => lines.push(s);
  o(`// ${f.name}: keyboard firmware made by Flashie from a Keebie design.`);
  o(`// Tinkerie maker tools, by nrmluvdev. Controller: ${f.mcu.name}. Board: ${f.mcu.fqbn}`);
  o(`// Edit the keymap in Flashie's Keyboard tab rather than here: Flashie rewrites this file.`);
  o(`#include <Arduino.h>`);
  o(`#if defined(ARDUINO_ARCH_AVR)`);
  o(`  #include <HID-Project.h>  // NicoHood's HID library: raw keycodes and media keys`);
  o(`#else`);
  o(`  #include <Keyboard.h>`);
  o(`#endif`);
  if (scr) o(`#include <Wire.h>\n#include <Adafruit_GFX.h>\n#include <Adafruit_SSD1306.h>`);
  if (useRgb) o(`#include <Adafruit_NeoPixel.h>`);
  o(``);
  o(`typedef uint32_t kbpin_t;`);
  o(`const uint8_t ROWS = ${R}, COLS = ${C}, LAYERS = ${L}, KNOBS = ${knobs.length};`);
  o(`const kbpin_t rowPins[ROWS] = { ${pins(rowPins)} };`);
  o(`const kbpin_t colPins[COLS] = { ${pins(colPins)} };`);
  o(`#define COL2ROW ${c2r ? 1 : 0}  // diodes point from the columns to the rows`);
  o(``);
  o(`// Keymap. 0x00xx = key (with Ctrl/Shift/Alt/Win in bits 8-11), 0x1xxx = media key,`);
  o(`// 0x20nn hold layer n, 0x30nn toggle layer n, 0x40nn go to layer n, 0x50nn type text n,`);
  o(`// 0x0001 = transparent (use the layer below), 0x0000 = nothing.`);
  o(`const uint16_t keymaps[LAYERS][ROWS][COLS] PROGMEM = {`);
  grid.forEach((lay, l) => { o(`  { // ${m.layers[l]}`); lay.forEach(row => o(`    { ${row.map(hex).join(', ')} },`)); o(`  },`); });
  o(`};`);
  o(`const char *const texts[] = { ${texts.length ? texts.map(cstr).join(', ') : '""'} };`);
  if (knobs.length) {
    o(`const kbpin_t knobA[KNOBS] = { ${pins(knobs.map((n, i) => pinOf(`k${i}a`, n.a)))} };`);
    o(`const kbpin_t knobB[KNOBS] = { ${pins(knobs.map((n, i) => pinOf(`k${i}b`, n.b)))} };`);
    o(`const uint16_t knobRight[KNOBS][LAYERS] = { ${kcw.map(a => '{ ' + a.map(hex).join(', ') + ' }').join(', ')} };`);
    o(`const uint16_t knobLeft[KNOBS][LAYERS] = { ${kccw.map(a => '{ ' + a.map(hex).join(', ') + ' }').join(', ')} };`);
    o(`const int8_t KNOB_STEPS = ${m.knob.steps | 0 || 4};  // pulses per click`);
    o(`const bool KNOB_FLIP = ${m.knob.flip ? 'true' : 'false'};`);
  }
  if (scr) o(`const char *const layerNames[LAYERS] = { ${m.layers.map(cstr).join(', ')} };`);
  o(``);
  o(`// ---------------------------------------------------------------- USB HID`);
  o(`static void hidKey(uint8_t u, bool down) {`);
  o(`#if defined(ARDUINO_ARCH_AVR)`);
  o(`  if (down) Keyboard.press(KeyboardKeycode(u)); else Keyboard.release(KeyboardKeycode(u));`);
  o(`#else`);
  o(`  uint8_t k = u >= 0xE0 ? (uint8_t)(0x80 + (u - 0xE0)) : (uint8_t)(u + 136);  // Keyboard.h raw keycodes`);
  o(`  if (down) Keyboard.press(k); else Keyboard.release(k);`);
  o(`#endif`);
  o(`}`);
  o(`static void hidMedia(uint16_t c, bool down) {`);
  o(`#if defined(ARDUINO_ARCH_AVR)`);
  o(`  if (down) Consumer.press(ConsumerKeycode(c)); else Consumer.release(ConsumerKeycode(c));`);
  o(`#elif defined(ARDUINO_ARCH_RP2040)`);
  o(`  if (down) Keyboard.consumerPress(c); else Keyboard.consumerRelease();`);
  o(`#else`);
  o(`  (void)c; (void)down;  // the STM32 core's Keyboard library has no media keys`);
  o(`#endif`);
  o(`}`);
  o(``);
  o(`// ---------------------------------------------------------------- layers and actions`);
  o(`uint8_t layerOn = 1;          // bit n = layer n is on (layer 0 always is)`);
  o(`uint8_t holdCount[LAYERS];     // momentary holds per layer`);
  o(`uint16_t presses = 0;`);
  o(`uint32_t pressTimes[24]; uint8_t pressHead = 0;  // for words per minute`);
  o(`static uint16_t lookup(uint8_t r, uint8_t c) {`);
  o(`  for (int8_t l = LAYERS - 1; l >= 0; l--) {`);
  o(`    if (!(layerOn & (1 << l))) continue;`);
  o(`    uint16_t k = pgm_read_word(&keymaps[l][r][c]);`);
  o(`    if (k != 1) return k;`);
  o(`  }`);
  o(`  return 0;`);
  o(`}`);
  o(`static uint8_t topLayer() { for (int8_t l = LAYERS - 1; l > 0; l--) if (layerOn & (1 << l)) return l; return 0; }`);
  o(`static void act(uint16_t k, bool down) {`);
  o(`  uint8_t type = k >> 12, n = k & 0xFF;`);
  o(`  switch (type) {`);
  o(`    case 0: {`);
  o(`      uint8_t mods = (k >> 8) & 0x0F;`);
  o(`      for (uint8_t b = 0; b < 4; b++) if (mods & (1 << b)) hidKey(0xE0 + b, down);`);
  o(`      if (n > 1) hidKey(n, down);`);
  o(`      break;`);
  o(`    }`);
  o(`    case 1: hidMedia(k & 0x0FFF, down); break;`);
  o(`    case 2: if (n < LAYERS) { if (down) holdCount[n]++; else if (holdCount[n]) holdCount[n]--; if (holdCount[n]) layerOn |= 1 << n; else layerOn &= ~(1 << n); } break;`);
  o(`    case 3: if (down && n < LAYERS) layerOn ^= 1 << n; break;`);
  o(`    case 4: if (down && n < LAYERS) layerOn = 1 | (1 << n); break;`);
  o(`    case 5: if (down && n < sizeof(texts) / sizeof(texts[0])) Keyboard.print(texts[n]); break;`);
  o(`  }`);
  o(`  layerOn |= 1;`);
  o(`  if (down && type != 2) { presses++; pressTimes[pressHead] = millis(); pressHead = (pressHead + 1) % 24; }`);
  o(`}`);
  o(`static void tap(uint16_t k) { if ((k >> 12) >= 2) return; act(k, true); delay(4); act(k, false); }`);
  o(``);
  o(`// ---------------------------------------------------------------- matrix`);
  o(`#if COL2ROW`);
  o(`  #define OUTS ROWS`);
  o(`  #define INS COLS`);
  o(`  #define OUTPIN(i) rowPins[i]`);
  o(`  #define INPIN(i) colPins[i]`);
  o(`#else`);
  o(`  #define OUTS COLS`);
  o(`  #define INS ROWS`);
  o(`  #define OUTPIN(i) colPins[i]`);
  o(`  #define INPIN(i) rowPins[i]`);
  o(`#endif`);
  o(`bool down[ROWS][COLS];`);
  o(`uint16_t changed[ROWS][COLS];`);
  o(`uint16_t sent[ROWS][COLS];   // what each key sent when pressed, so it releases the same thing`);
  o(`const uint16_t DEBOUNCE_MS = 5;`);
  o(`static void scanMatrix() {`);
  o(`  uint16_t now = (uint16_t)millis();`);
  o(`  for (uint8_t o = 0; o < OUTS; o++) {`);
  o(`    pinMode(OUTPIN(o), OUTPUT);`);
  o(`    digitalWrite(OUTPIN(o), LOW);`);
  o(`    delayMicroseconds(6);`);
  o(`    for (uint8_t i = 0; i < INS; i++) {`);
  o(`      bool d = digitalRead(INPIN(i)) == LOW;`);
  o(`#if COL2ROW`);
  o(`      uint8_t r = o, c = i;`);
  o(`#else`);
  o(`      uint8_t r = i, c = o;`);
  o(`#endif`);
  o(`      if (d != down[r][c] && (uint16_t)(now - changed[r][c]) >= DEBOUNCE_MS) {`);
  o(`        down[r][c] = d;`);
  o(`        changed[r][c] = now;`);
  o(`        if (d) { sent[r][c] = lookup(r, c); act(sent[r][c], true); }`);
  o(`        else act(sent[r][c], false);`);
  o(`      }`);
  o(`    }`);
  o(`    pinMode(OUTPIN(o), INPUT);`);
  o(`  }`);
  o(`}`);
  if (knobs.length) {
    o(``);
    o(`// ---------------------------------------------------------------- knobs`);
    o(`const int8_t KNOB_TABLE[16] = { 0, -1, 1, 0, 1, 0, 0, -1, -1, 0, 0, 1, 0, 1, -1, 0 };`);
    o(`uint8_t knobState[KNOBS]; int8_t knobCount[KNOBS];`);
    o(`static uint16_t knobAction(const uint16_t table[][LAYERS], uint8_t n) {`);
    o(`  for (int8_t l = LAYERS - 1; l >= 0; l--) if (layerOn & (1 << l)) { uint16_t k = table[n][l]; if (k != 1) return k; }`);
    o(`  return 0;`);
    o(`}`);
    o(`static void scanKnobs() {`);
    o(`  for (uint8_t n = 0; n < KNOBS; n++) {`);
    o(`    uint8_t s = (digitalRead(knobA[n]) << 1) | digitalRead(knobB[n]);`);
    o(`    knobState[n] = ((knobState[n] << 2) | s) & 0x0F;`);
    o(`    knobCount[n] += KNOB_TABLE[knobState[n]];`);
    o(`    if (knobCount[n] >= KNOB_STEPS || knobCount[n] <= -KNOB_STEPS) {`);
    o(`      bool right = (knobCount[n] > 0) != KNOB_FLIP;`);
    o(`      knobCount[n] = 0;`);
    o(`      tap(knobAction(right ? knobRight : knobLeft, n));`);
    o(`    }`);
    o(`  }`);
    o(`}`);
  }
  if (scr) {
    o(``);
    o(`// ---------------------------------------------------------------- screen (SSD1306)`);
    o(`Adafruit_SSD1306 screen(${scr.w}, ${scr.h}, &${wire}, -1);`);
    o(`bool screenOK = false;`);
    o(`static uint16_t wpm() {`);
    o(`  uint32_t now = millis(); uint8_t n = 0;`);
    o(`  for (uint8_t i = 0; i < 24; i++) if (pressTimes[i] && now - pressTimes[i] < 10000UL) n++;`);
    o(`  return n * 6 / 5;  // presses in 10 s → characters per minute → words (5 characters)`);
    o(`}`);
    o(`static void drawScreen() {`);
    o(`  if (!screenOK) return;`);
    o(`  screen.clearDisplay();`);
    o(`  screen.setTextColor(SSD1306_WHITE);`);
    o(`  screen.setCursor(0, 0);`);
    const big = scr.h >= 64;
    if (m.screen.title) { o(`  screen.setTextSize(${big ? 2 : 1});`); o(`  screen.println(${cstr(m.screen.title.slice(0, big ? 10 : 21))});`); }
    o(`  screen.setTextSize(1);`);
    if (m.screen.layer) o(`  screen.print("Layer: "); screen.println(layerNames[topLayer()]);`);
    if (m.screen.wpm) o(`  screen.print("WPM: "); screen.println(wpm());`);
    if (m.screen.keys) o(`  screen.print("Keys: "); screen.println(presses);`);
    o(`  screen.display();`);
    o(`}`);
  }
  if (useRgb) {
    o(``);
    o(`// ---------------------------------------------------------------- per-key RGB`);
    o(`Adafruit_NeoPixel leds(${rgb.count}, ${pinOf('led', rgb.pin)}, NEO_GRB + NEO_KHZ800);`);
    o(`static void drawLeds() {`);
    o(`  uint32_t t = millis();`);
    if (m.rgb.mode === 'rainbow') o(`  for (uint16_t i = 0; i < leds.numPixels(); i++) leds.setPixelColor(i, leds.gamma32(leds.ColorHSV((uint16_t)(t * 12 + i * 65536UL / leds.numPixels()))));`);
    else if (m.rgb.mode === 'breathe') { o(`  uint8_t b = (uint8_t)(127 + 127 * sin(t / 900.0));`); o(`  for (uint16_t i = 0; i < leds.numPixels(); i++) leds.setPixelColor(i, leds.Color(${rgbv[0]} * b / 255, ${rgbv[1]} * b / 255, ${rgbv[2]} * b / 255));`); }
    else o(`  for (uint16_t i = 0; i < leds.numPixels(); i++) leds.setPixelColor(i, leds.Color(${rgbv.join(', ')}));`);
    o(`  leds.show();`);
    o(`}`);
  }
  o(``);
  o(`// ---------------------------------------------------------------- start and loop`);
  o(`void setup() {`);
  o(`  for (uint8_t i = 0; i < INS; i++) pinMode(INPIN(i), INPUT_PULLUP);`);
  o(`  for (uint8_t o = 0; o < OUTS; o++) pinMode(OUTPIN(o), INPUT);`);
  if (knobs.length) o(`  for (uint8_t n = 0; n < KNOBS; n++) { pinMode(knobA[n], INPUT_PULLUP); pinMode(knobB[n], INPUT_PULLUP); }`);
  o(`  Keyboard.begin();`);
  o(`#if defined(ARDUINO_ARCH_AVR)`);
  o(`  Consumer.begin();`);
  o(`#endif`);
  if (scr) {
    if (core === 'rp2040') o(`  ${wire}.setSDA(${sda});\n  ${wire}.setSCL(${scl});`);
    if (core === 'stm32') o(`  Wire.setSDA(${sda});\n  Wire.setSCL(${scl});`);
    o(`  screenOK = screen.begin(SSD1306_SWITCHCAPVCC, 0x3C);`);
    if (m.screen.flip) o(`  screen.setRotation(2);`);
    o(`  drawScreen();`);
  }
  if (useRgb) o(`  leds.begin();\n  leds.setBrightness(${Math.round(Math.max(1, Math.min(100, m.rgb.bright)) * 2.55)});\n  leds.show();`);
  o(`}`);
  o(``);
  o(`void loop() {`);
  o(`  scanMatrix();`);
  if (knobs.length) o(`  scanKnobs();`);
  if (scr || useRgb) {
    o(`  static uint32_t lastDraw = 0;`);
    o(`  if (millis() - lastDraw > ${useRgb ? 30 : 150}) {`);
    o(`    lastDraw = millis();`);
    if (useRgb) o(`    drawLeds();`);
    if (scr) o(`    static uint8_t tick = 0;\n    if (++tick % ${useRgb ? 5 : 1} == 0) drawScreen();`);
    o(`  }`);
  }
  o(`}`);
  return { code: lines.join('\n') + '\n', libs };
}

// ------------------------------------------------------------------ UI
const css = `
#kbdView{display:flex;flex-direction:column;gap:16px}
#kbdView.hidden{display:none}
.kbd-grid{display:grid;grid-template-columns:minmax(0,1fr) 330px;gap:16px;align-items:start}
@media (max-width:1100px){.kbd-grid{grid-template-columns:1fr}}
.kbd-col{display:flex;flex-direction:column;gap:16px;min-width:0}
.kbd-row{display:flex;gap:8px;align-items:center;flex-wrap:wrap}
.kbd-row select,.kbd-in{background:var(--bg2);border:1px solid var(--line2);border-radius:10px;padding:8px 10px;color:var(--text);font:inherit;font-size:13px;min-width:0}
.kbd-row select{flex:1}
.kbd-sum{font-size:13px;color:var(--muted);line-height:1.5}
.kbd-sum b{color:var(--text)}
.kbd-note{font-size:12.5px;border-radius:10px;padding:8px 10px;background:rgba(255,190,92,.08);border:1px solid rgba(255,190,92,.35);color:var(--text)}
.kbd-note.bad{background:rgba(255,92,122,.08);border-color:rgba(255,92,122,.35)}
.kbd-layers{display:flex;gap:6px;flex-wrap:wrap;margin-left:auto}
.kbd-layers button{appearance:none;border:1px solid var(--line2);background:var(--card2);color:var(--muted);border-radius:999px;padding:5px 12px;font:600 12.5px var(--display);cursor:pointer}
.kbd-layers button.on{background:var(--grad);border-color:transparent;color:#fff}
.kbd-board{position:relative;margin:14px 0 4px;overflow:auto;padding:6px}
.kbd-key{position:absolute;box-sizing:border-box;border-radius:7px;background:#2a3040;border:1px solid #3a4256;color:var(--text);display:grid;place-items:center;text-align:center;font:600 11.5px/1.1 var(--display);cursor:pointer;padding:2px;overflow:hidden;box-shadow:0 2px 0 #0007;transform-origin:center}
.kbd-key:hover{border-color:#5a6584}
.kbd-key.trns{color:var(--dim)}
.kbd-key.layer{background:#2b2550;border-color:#5b47c9}
.kbd-key.media{background:#123447;border-color:#1d6b8f}
.kbd-key.sel{outline:2px solid var(--accent2);outline-offset:1px}
.kbd-key.knob{border-radius:50%;background:radial-gradient(circle at 40% 35%,#46506a,#20252f)}
.kbd-key small{display:block;font-size:9.5px;color:var(--muted);font-weight:500}
.kbd-pick{border-top:1px solid var(--line);margin-top:12px;padding-top:12px;display:flex;flex-direction:column;gap:10px}
.kbd-pick h3{margin:0;font:650 15px var(--display)}
.kbd-cats{display:flex;gap:6px;flex-wrap:wrap}
.kbd-cats button,.kbd-slot button{appearance:none;border:1px solid var(--line2);background:transparent;color:var(--muted);border-radius:8px;padding:4px 9px;font-size:12px;cursor:pointer}
.kbd-cats button.on,.kbd-slot button.on{background:var(--card2);color:var(--text);border-color:#4a5470}
.kbd-slot{display:flex;gap:6px}
.kbd-codes{display:flex;flex-wrap:wrap;gap:5px;max-height:190px;overflow:auto}
.kbd-codes button{appearance:none;min-width:42px;border:1px solid var(--line2);background:var(--card2);color:var(--text);border-radius:8px;padding:6px 8px;font:600 12px var(--display);cursor:pointer}
.kbd-codes button:hover{border-color:var(--accent)}
.kbd-codes button.on{background:var(--grad);border-color:transparent;color:#fff}
.kbd-desc{font-size:12.5px;color:var(--muted)}
.kbd-mods{display:flex;gap:12px;font-size:12.5px;color:var(--muted);flex-wrap:wrap}
.kbd-mods label{display:flex;gap:5px;align-items:center;cursor:pointer}
.kbd-f{display:flex;flex-direction:column;gap:8px;font-size:13px;color:var(--muted)}
.kbd-f label.c{display:flex;gap:8px;align-items:center;cursor:pointer}
.kbd-f .r{display:grid;grid-template-columns:1fr 1fr;gap:8px;align-items:center}
.kbd-pins{display:grid;grid-template-columns:1fr 60px 80px;gap:4px 8px;font-size:12.5px;align-items:center;max-height:260px;overflow:auto}
.kbd-pins .h{color:var(--dim);font-size:11.5px}
.kbd-pins input{width:100%;box-sizing:border-box;background:var(--bg2);border:1px solid var(--line2);border-radius:6px;padding:4px 6px;color:var(--text);font:12px var(--mono)}
.kbd-go{background:var(--grad);border-color:transparent;color:#fff;padding:12px;font-size:15px;width:100%}
.kbd-go:hover{filter:brightness(1.1);background:var(--grad)}
.kbd-stat{font-size:12.5px;color:var(--muted);min-height:18px}
.kbd-stat.ok{color:var(--ok)}.kbd-stat.bad{color:var(--bad)}
.kbd-bar{height:6px;border-radius:9px;background:var(--bg2);overflow:hidden}
.kbd-bar i{display:block;height:100%;width:0;background:var(--grad);transition:width .2s}
.kbd-con{margin:0;background:#07090d;border:1px solid var(--line);border-radius:10px;padding:8px;height:150px;overflow:auto;font:11.5px/1.4 var(--mono);color:#aab3c5;white-space:pre-wrap}
.kbd-code{margin:0;max-height:60vh;overflow:auto;font:12px/1.45 var(--mono);white-space:pre;padding:12px;background:#07090d;border-radius:10px}
.kbd-empty{padding:30px;text-align:center;color:var(--muted)}
details.kbd-d summary{cursor:pointer;color:var(--text);font-weight:600;font-size:13px}
`;

function build() {
  const st = document.createElement('style'); st.textContent = css; document.head.appendChild(st);
  const v = document.createElement('section');
  v.id = 'kbdView'; v.className = 'hidden';
  v.innerHTML = `
  <div class="card" data-help="Keyboard file"><div class="card-h"><div class="n">1</div><h2>Keyboard</h2><span class="sub">designed in Wirie's Keebie tab</span></div>
    <div class="kbd-row" style="margin-top:12px"><select id="kbdFile" data-help="Pick a keyboard"></select><button class="btn" id="kbdOpen" data-help="Open keyboard file">Open file…</button><button class="btn ghost" id="kbdRefresh" data-help="Refresh keyboards">↻</button></div>
    <div class="kbd-sum" id="kbdSum" style="margin-top:10px"></div>
    <input type="file" id="kbdFileIn" accept=".json" hidden>
  </div>
  <div class="kbd-grid">
    <div class="kbd-col">
      <div class="card" data-help="Keymap"><div class="card-h"><div class="n">2</div><h2>Keymap</h2><div class="kbd-layers" id="kbdLayers"></div></div>
        <div class="kbd-board" id="kbdBoard"></div>
        <div class="kbd-pick" id="kbdPick"></div>
      </div>
    </div>
    <div class="kbd-col" id="kbdSide"></div>
  </div>
  <div class="imgmodal hidden" id="kbdCodeBox"><div class="imgbox" style="width:min(900px,94vw)"><div class="imgbox-h"><h2>Firmware code (Arduino)</h2><button class="btn" id="kbdCodeSave">Save .ino</button><button class="btn ghost" id="kbdCodeClose">✕</button></div><div class="imgbox-b"><pre class="kbd-code" id="kbdCode"></pre></div></div></div>`;
  const app = document.querySelector('.app');
  app.insertBefore(v, document.querySelector('footer'));
  $k('#kbdFile').onchange = e => { const p = e.target.value; if (p) openPath(p); };
  $k('#kbdOpen').onclick = () => $k('#kbdFileIn').click();
  $k('#kbdRefresh').onclick = () => loadList();
  $k('#kbdFileIn').onchange = async e => { const f = e.target.files[0]; if (!f) return; try { useFile(JSON.parse(await f.text()), f.name); } catch (err) { alert("That file couldn't be read: " + err.message); } e.target.value = ''; };
  $k('#kbdCodeClose').onclick = () => $k('#kbdCodeBox').classList.add('hidden');
  $k('#kbdCodeSave').onclick = () => download((K.file.name || 'keyboard') + '.ino', new Blob([K.code], { type: 'text/plain' }));
  $k('#kbdBoard').addEventListener('click', e => { const b = e.target.closest('[data-k]'); if (!b) return; K.sel = b.dataset.k; K.slot = 'press'; renderBoard(); renderPick(); });
}

function download(name, blob) { const a = document.createElement('a'); a.href = URL.createObjectURL(blob); a.download = name; document.body.appendChild(a); a.click(); a.remove(); setTimeout(() => URL.revokeObjectURL(a.href), 3000); }

async function loadList(openInitial) {
  try {
    const r = await (await fetch('/api/keyboard/list')).json();
    K.list = r.files || [];
    if (r.initial && !K.list.some(f => f.path === r.initial)) K.list.unshift({ path: r.initial, name: r.initial.split(/[\\/]/).pop().replace(/\.keebie\.json$/i, '') });
    const sel = $k('#kbdFile');
    sel.innerHTML = `<option value="">${K.list.length ? 'Pick a keyboard from Wirie…' : 'No keyboards yet: use “Firmware in Flashie” in Keebie'}</option>` + K.list.map(f => `<option value="${esc(f.path)}" ${f.path === K.path ? 'selected' : ''}>${esc(f.name)}</option>`).join('');
    if (openInitial && r.initial) { await openPath(r.initial); if (window.flashieSetView) window.flashieSetView('keyboard'); }
  } catch {}
}
async function openPath(p) {
  try {
    const r = await fetch('/api/keyboard/open?path=' + encodeURIComponent(p));
    const j = await r.json();
    if (j.error) throw new Error(j.error);
    K.path = p; useFile(j, p);
  } catch (e) { alert("Couldn't open that keyboard: " + e.message); }
}
async function useFile(f, from) {
  if (!f || f.format !== 'keebie-keyboard') { alert('That isn\'t a Keebie keyboard file. In Wirie, open Keebie and press “Firmware in Flashie”.'); return; }
  K.file = f; K.sel = null; K.layer = 0;
  let saved = null;
  try { saved = await (await fetch('/api/keyboard/keymap?name=' + encodeURIComponent(f.name))).json(); } catch {}
  K.map = fitMap(saved && saved.layers ? saved : defaultMap(f), f);
  if ($k('#kbdFile').value !== K.path) $k('#kbdFile').value = K.path || '';
  renderAll();
}

function renderAll() { renderSummary(); renderLayers(); renderBoard(); renderPick(); renderSide(); }

function renderSummary() {
  const f = K.file, el = $k('#kbdSum');
  if (!f) { el.innerHTML = 'Design a keyboard in <b>Wirie → Keebie</b>, then press <b>Firmware in Flashie</b>. It opens here, ready to set up.'; return; }
  const nk = f.keys.filter(k => k.kind !== 'knob').length, nn = (f.knobs || []).length;
  const probs = problems().filter(p => !/has no pin/.test(p));
  el.innerHTML = `<b>${esc(f.name)}</b> · ${nk} keys${nn ? ` · ${nn} knob${nn > 1 ? 's' : ''}` : ''}${f.screen ? ' · ' + esc(f.screen.name) : ''}${f.rgb ? ' · RGB' : ''} · <b>${esc(f.mcu.name)}</b> · matrix ${f.rows.length} × ${f.cols.length} (${esc(f.diode)})`
    + (f.mcu.note ? `<div class="kbd-note" style="margin-top:8px">${esc(f.mcu.note)}</div>` : '')
    + (f.mcu.core === 'stm32' ? `<div class="kbd-note" style="margin-top:8px">Media keys (volume, play…) need an RP2040 or ATmega32U4: the STM32 Arduino core can't send them.</div>` : '')
    + probs.map(p => `<div class="kbd-note bad" style="margin-top:8px">${esc(p)}</div>`).join('');
}

function renderLayers() {
  const el = $k('#kbdLayers'); if (!K.file) { el.innerHTML = ''; return; }
  el.innerHTML = K.map.layers.map((n, i) => `<button data-l="${i}" class="${i === K.layer ? 'on' : ''}" title="Double-click to rename">${i} · ${esc(n)}</button>`).join('')
    + (K.map.layers.length < LAYERS_MAX ? `<button data-add="1" title="Add a layer" data-help="Add layer">+ Layer</button>` : '')
    + (K.map.layers.length > 1 ? `<button data-del="1" title="Remove the last layer" data-help="Remove layer">−</button>` : '');
  el.dataset.help = 'Layers';
  el.onclick = e => {
    const b = e.target.closest('button'); if (!b) return;
    if (b.dataset.add) { const L = K.map.layers.length; K.map.layers.push('Layer ' + L); for (const id in K.map.keys) K.map.keys[id].push('KC_TRNS'); for (const id in K.map.knobs) K.map.knobs[id].push({ cw: 'KC_TRNS', ccw: 'KC_TRNS' }); K.layer = L; save(); renderAll(); return; }
    if (b.dataset.del) { const L = K.map.layers.length - 1; if (!confirm(`Remove layer ${L} (${K.map.layers[L]})?`)) return; K.map.layers.pop(); for (const id in K.map.keys) K.map.keys[id].length = L; for (const id in K.map.knobs) K.map.knobs[id].length = L; K.layer = Math.min(K.layer, L - 1); save(); renderAll(); return; }
    K.layer = +b.dataset.l; renderLayers(); renderBoard(); renderPick();
  };
  el.ondblclick = e => { const b = e.target.closest('[data-l]'); if (!b) return; const i = +b.dataset.l; const n = prompt('Name for layer ' + i + ':', K.map.layers[i]); if (n && n.trim()) { K.map.layers[i] = n.trim().slice(0, 16); save(); renderLayers(); } };
}

function renderBoard() {
  const el = $k('#kbdBoard'), f = K.file;
  if (!f) { el.innerHTML = `<div class="kbd-empty">No keyboard open yet.</div>`; el.style.height = ''; return; }
  const W = Math.max(...f.keys.map(k => k.x + k.w)), H = Math.max(...f.keys.map(k => k.y + k.h));
  const avail = Math.max(300, el.clientWidth - 16);
  const u = Math.max(30, Math.min(64, avail / W));
  el.style.height = (H * u + 16) + 'px';
  el.innerHTML = f.keys.map(k => {
    const code = (K.map.keys[k.id] || [])[K.layer] || 'KC_TRNS', p = parse(code), knob = k.kind === 'knob';
    const cls = ['kbd-key', knob ? 'knob' : '', p.kind === 'trns' ? 'trns' : '', /mo|tg|to/.test(p.kind) ? 'layer' : '', p.kind === 'media' ? 'media' : '', K.sel === k.id ? 'sel' : ''].join(' ');
    let txt = esc(label(code));
    if (knob) { const kn = (K.map.knobs[k.id] || [])[K.layer]; if (kn) txt = `${esc(label(code)) || '·'}<small>${esc(label(kn.ccw))} ⟲ ⟳ ${esc(label(kn.cw))}</small>`; }
    const rot = k.r ? `transform:rotate(${k.r}deg);` : '';
    return `<div class="${cls}" data-k="${esc(k.id)}" title="${esc(k.label || '')} — ${esc(describe(code))}" style="left:${k.x * u + 8}px;top:${k.y * u + 8}px;width:${k.w * u - 4}px;height:${k.h * u - 4}px;${rot}">${txt}</div>`;
  }).join('');
}

function currentCode() {
  const id = K.sel; if (!id) return null;
  if (K.slot === 'press') return K.map.keys[id][K.layer];
  return K.map.knobs[id][K.layer][K.slot];
}
function setCode(c) {
  const id = K.sel; if (!id) return;
  if (K.slot === 'press') K.map.keys[id][K.layer] = c; else K.map.knobs[id][K.layer][K.slot] = c;
  save(); renderBoard(); renderPick();
}

function renderPick() {
  const el = $k('#kbdPick'), f = K.file;
  if (!f) { el.innerHTML = ''; return; }
  if (!K.sel) { el.innerHTML = `<div class="kbd-desc">Click a key (or a knob) to choose what it does on layer ${K.layer} (${esc(K.map.layers[K.layer])}). ▽ means the key falls through to the layer below.</div>`; return; }
  const key = f.keys.find(k => k.id === K.sel), knob = key.kind === 'knob' && K.map.knobs[key.id];
  const code = currentCode(), p = parse(code);
  const mods = p.kind === 'key' ? p.mods : 0;
  const base = p.kind === 'key' ? p.k.id : p.kind === 'media' ? p.k.id : '';
  const list = K.cat === 'Layers' ? [] : Object.values(KC).filter(k => k.group === K.cat);
  el.innerHTML = `<div class="kbd-row"><h3>${knob ? 'Knob' : 'Key'} ${esc(key.label || '')} · layer ${K.layer}</h3>
      ${knob ? `<div class="kbd-slot" data-help="Knob actions" style="margin-left:auto">${[['press', 'Press'], ['ccw', 'Turn left ⟲'], ['cw', 'Turn right ⟳']].map(([s, t]) => `<button data-slot="${s}" class="${K.slot === s ? 'on' : ''}">${t}</button>`).join('')}</div>` : ''}</div>
    <div class="kbd-desc"><b>${esc(label(code) || 'Nothing')}</b>: ${esc(describe(code))}</div>
    <div class="kbd-cats" data-help="Key groups">${[...G, 'Layers', 'Text'].map(c => `<button data-cat="${c}" class="${K.cat === c ? 'on' : ''}">${c}</button>`).join('')}<button data-set="KC_TRNS" title="Use the layer below">▽ Transparent</button><button data-set="KC_NO">None</button></div>
    ${K.cat === 'Layers' ? `<div class="kbd-codes" data-help="Layer keys">${K.map.layers.map((n, i) => [['MO', 'Hold'], ['TG', 'Toggle'], ['TO', 'Go to']].filter(([f2]) => i > 0 || f2 === 'TO').map(([f2, t]) => `<button data-set="${f2}(${i})" class="${code === `${f2}(${i})` ? 'on' : ''}" title="${t} layer ${i}">${t} ${i} · ${esc(n)}</button>`).join('')).join('')}</div>`
      : K.cat === 'Text' ? `<div class="kbd-row" data-help="Type text"><input class="kbd-in" id="kbdText" style="flex:1" placeholder="Text this key types, e.g. your email signature" value="${p.kind === 'text' ? esc(p.text) : ''}" maxlength="120"><button class="btn" id="kbdTextSet">Use text</button></div><div class="kbd-desc">Plain letters, numbers and symbols, typed as if on a US keyboard layout.</div>`
      : `<div class="kbd-codes" data-help="Key codes">${list.map(k => `<button data-code="${k.id}" class="${base === k.id ? 'on' : ''}" title="${esc(k.desc || k.id)}">${esc(k.label)}</button>`).join('')}</div>
        ${K.cat !== 'Media' && K.cat !== 'Modifiers' ? `<div class="kbd-mods" data-help="Shortcut modifiers"><span>Hold with it:</span>${MODS.map(([, t, b]) => `<label><input type="checkbox" data-mod="${b}" ${mods & b ? 'checked' : ''} ${p.kind !== 'key' ? 'disabled' : ''}> ${t}</label>`).join('')}</div>` : ''}`}`;
  el.onclick = e => {
    const b = e.target.closest('button'); if (!b) return;
    if (b.dataset.slot) { K.slot = b.dataset.slot; renderPick(); return; }
    if (b.dataset.cat) { K.cat = b.dataset.cat; renderPick(); return; }
    if (b.dataset.set) { setCode(b.dataset.set); return; }
    if (b.dataset.code) { const k = KC[b.dataset.code]; let c = k.id; if (k.t === 'k' && mods && k.group !== 'Modifiers') for (const [n, , bit] of MODS) if (mods & bit) c = `${n}(${c})`; setCode(c); return; }
    if (b.id === 'kbdTextSet') { const t = $k('#kbdText').value; if (t) setCode('TEXT:' + t); }
  };
  el.onchange = e => {
    const cb = e.target.closest('[data-mod]'); if (!cb || p.kind !== 'key') return;
    let m2 = 0; el.querySelectorAll('[data-mod]').forEach(x => { if (x.checked) m2 |= +x.dataset.mod; });
    let c = p.k.id; for (const [n, , bit] of MODS) if (m2 & bit) c = `${n}(${c})`; setCode(c);
  };
}

function renderSide() {
  const el = $k('#kbdSide'), f = K.file;
  if (!f) { el.innerHTML = ''; return; }
  const m = K.map, probs = problems(), canFlash = !!f.mcu.flash;
  el.innerHTML = `
    ${f.screen ? `<div class="card" data-help="Screen settings"><div class="card-h"><h2>Screen</h2><span class="sub">${esc(f.screen.name)}</span></div><div class="kbd-f" style="margin-top:10px">
      <label>Title<input class="kbd-in" id="kbdTitle" value="${esc(m.screen.title)}" maxlength="21"></label>
      <label class="c"><input type="checkbox" data-scr="layer" ${m.screen.layer ? 'checked' : ''}> Show the layer name</label>
      <label class="c"><input type="checkbox" data-scr="wpm" ${m.screen.wpm ? 'checked' : ''}> Show typing speed (WPM)</label>
      <label class="c"><input type="checkbox" data-scr="keys" ${m.screen.keys ? 'checked' : ''}> Count key presses</label>
      <label class="c"><input type="checkbox" data-scr="flip" ${m.screen.flip ? 'checked' : ''}> Turn the picture upside down</label></div></div>` : ''}
    ${f.rgb ? `<div class="card" data-help="RGB settings"><div class="card-h"><h2>Lights</h2><span class="sub">${f.rgb.count} LEDs</span></div><div class="kbd-f" style="margin-top:10px">
      <div class="r"><label>Effect<select class="kbd-in" id="kbdRgbMode">${[['off', 'Off'], ['solid', 'One colour'], ['breathe', 'Breathing'], ['rainbow', 'Rainbow']].map(([v, t]) => `<option value="${v}" ${m.rgb.mode === v ? 'selected' : ''}>${t}</option>`).join('')}</select></label><label>Colour<input type="color" id="kbdRgbCol" value="${esc(m.rgb.color)}" style="width:100%;height:34px;background:none;border:0"></label></div>
      <label>Brightness <input type="range" id="kbdRgbBr" min="5" max="100" value="${m.rgb.bright}"></label></div></div>` : ''}
    ${(f.knobs || []).length ? `<div class="card" data-help="Knob settings"><div class="card-h"><h2>Knobs</h2></div><div class="kbd-f" style="margin-top:10px">
      <label>Pulses per click<select class="kbd-in" id="kbdSteps">${[[4, '4 (most EC11 knobs)'], [2, '2'], [1, '1 (very sensitive)']].map(([v, t]) => `<option value="${v}" ${+m.knob.steps === v ? 'selected' : ''}>${t}</option>`).join('')}</select></label>
      <label class="c"><input type="checkbox" id="kbdKFlip" ${m.knob.flip ? 'checked' : ''}> Knob turns the wrong way: swap directions</label></div></div>` : ''}
    <div class="card" data-help="Wiring"><details class="kbd-d"><summary>Wiring (${wiring().length} pins)</summary><div class="kbd-desc" style="margin:8px 0">Arduino pin numbers worked out from your Keebie design. Only change one if you wired your board differently.</div>
      <div class="kbd-pins"><span class="h">Signal</span><span class="h">Pad</span><span class="h">Pin</span>${wiring().map(w => `<span>${esc(w.label)}</span><span class="h">${esc(w.p ? w.p.pad : '')}</span><input data-pin="${w.key}" value="${esc(pinOf(w.key, w.p))}">`).join('')}</div></details></div>
    <div class="card" data-help="Make firmware"><div class="card-h"><div class="n">3</div><h2>Make firmware</h2></div><div class="kbd-f" style="margin-top:10px">
      <label data-help="Keyboard port">Keyboard's port<div class="kbd-row"><select class="kbd-in" id="kbdPort"><option value="">— pick when it's plugged in —</option>${K.ports.map(p => `<option value="${esc(p.name)}" ${p.name === K.port ? 'selected' : ''}>${esc(p.friendly || p.name)}</option>`).join('')}</select><button class="btn ghost" id="kbdPorts" title="Look again">↻</button></div></label>
      <button class="btn kbd-go" id="kbdFlash" data-help="Make and flash" ${probs.length || !canFlash || K.busy ? 'disabled' : ''}>⚡ Make &amp; flash</button>
      <div class="kbd-row"><button class="btn" id="kbdMake" data-help="Make firmware only" ${probs.length || K.busy ? 'disabled' : ''} style="flex:1">Make only</button><button class="btn" id="kbdSave" data-help="Save firmware file" ${K.built ? '' : 'disabled'} style="flex:1">Save file</button><button class="btn ghost" id="kbdShow" data-help="Show firmware code" ${probs.length ? 'disabled' : ''}>Code</button></div>
      ${!canFlash && f.mcu.core ? `<div class="kbd-desc">Flashie can't write to this controller itself: press Make only, then Save file.</div>` : ''}
      <div class="kbd-bar"><i id="kbdBar"></i></div>
      <div class="kbd-stat" id="kbdStat">${K.stat ? esc(K.stat) : `The first build downloads the ${f.mcu.core === 'avr' ? 'Arduino AVR' : f.mcu.core === 'rp2040' ? 'RP2040' : 'STM32'} compiler once (a few hundred MB), then it's quick.`}</div>
      <pre class="kbd-con" id="kbdCon" data-help="Build output"></pre></div></div>`;
  const on = (id, ev, fn) => { const x = $k('#' + id); if (x) x[ev] = fn; };
  on('kbdTitle', 'oninput', e => { m.screen.title = e.target.value; save(); });
  el.querySelectorAll('[data-scr]').forEach(x => x.onchange = () => { m.screen[x.dataset.scr] = x.checked; save(); });
  on('kbdRgbMode', 'onchange', e => { m.rgb.mode = e.target.value; save(); });
  on('kbdRgbCol', 'oninput', e => { m.rgb.color = e.target.value; save(); });
  on('kbdRgbBr', 'onchange', e => { m.rgb.bright = +e.target.value; save(); });
  on('kbdSteps', 'onchange', e => { m.knob.steps = +e.target.value; save(); });
  on('kbdKFlip', 'onchange', e => { m.knob.flip = e.target.checked; save(); });
  el.querySelectorAll('[data-pin]').forEach(x => x.onchange = () => { const w = wiring().find(q => q.key === x.dataset.pin); const v = x.value.trim(); if (!v || (w.p && v === String(w.p.pin))) delete m.pins[x.dataset.pin]; else m.pins[x.dataset.pin] = v; save(); renderSummary(); });
  on('kbdPort', 'onchange', e => { K.port = e.target.value; });
  on('kbdPorts', 'onclick', loadPorts);
  on('kbdFlash', 'onclick', () => run(true));
  on('kbdMake', 'onclick', () => run(false));
  on('kbdSave', 'onclick', () => { window.location.href = '/api/keyboard/firmware'; });
  on('kbdShow', 'onclick', () => { K.code = generate().code; $k('#kbdCode').textContent = K.code; $k('#kbdCodeBox').classList.remove('hidden'); });
}

async function loadPorts() {
  try { K.ports = await (await fetch('/api/ports')).json(); } catch { K.ports = []; }
  if (!K.port) { const g = K.ports.find(p => p.kind === 'drive') || K.ports.find(p => p.usb); if (g) K.port = g.name; }
  if (K.file) renderSide();
}

function stat(t, cls) { const s = $k('#kbdStat'); if (s) { s.textContent = t; s.className = 'kbd-stat ' + (cls || ''); } K.stat = t; }
async function run(flash) {
  const f = K.file; const g = generate(); K.code = g.code;
  if (flash && !K.port) { stat('Plug the keyboard in and pick its port first. A Pico-style board in BOOTSEL mode shows up as a drive.', 'bad'); return; }
  K.busy = true; renderSide(); $k('#kbdCon').textContent = '';
  stat(flash ? 'Making and flashing the firmware…' : 'Making the firmware…');
  try {
    const r = await (await fetch('/api/keyboard/build', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ Core: f.mcu.core, Fqbn: f.mcu.fqbn, Code: g.code, Libs: g.libs, Port: flash ? K.port : '', Board: f.mcu.flash || '', Name: f.name, Flash: flash }) })).json();
    if (r.error) { K.busy = false; renderSide(); stat(r.error, 'bad'); }
  } catch (e) { K.busy = false; renderSide(); stat(e.message, 'bad'); }
}

function events() {
  const es = new EventSource('/api/events');
  es.onmessage = ev => {
    let m; try { m = JSON.parse(ev.data); } catch { return; }
    if (!K.busy) return;
    if (m.type === 'console') { const c = $k('#kbdCon'); if (c) { c.textContent += m.text; if (c.textContent.length > 60000) c.textContent = c.textContent.slice(-50000); c.scrollTop = c.scrollHeight; } }
    else if (m.type === 'kbstatus') stat(m.text);
    else if (m.type === 'kbprogress') { const b = $k('#kbdBar'); if (b) b.style.width = Math.floor(m.done * 100 / Math.max(1, m.total)) + '%'; }
    else if (m.type === 'kbdone') { K.busy = false; if (m.ok) K.built = true; const keep = $k('#kbdCon') ? $k('#kbdCon').textContent : ''; renderSide(); const c = $k('#kbdCon'); if (c) { c.textContent = keep; c.scrollTop = c.scrollHeight; } stat(m.text, m.ok ? 'ok' : 'bad'); const b = $k('#kbdBar'); if (b) b.style.width = m.ok ? '100%' : '0'; }
  };
}

window.kbdView = on => {
  $k('#kbdView').classList.toggle('hidden', !on);
  if (on) { loadList(); loadPorts(); renderAll(); }
};
// for tests
window.kbdTest = { parse, label, generate: () => generate(), K, useFile };

build();
events();
loadList(true);
window.addEventListener('resize', () => { if (K.file && !$k('#kbdView').classList.contains('hidden')) renderBoard(); });
})();
