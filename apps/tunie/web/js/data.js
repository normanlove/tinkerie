/* Tunie — data.js: printer presets, filament library, firmware info, checklist.
   Part of the nrmluvdev maker suite. Created by nrmluvdev.
   Preset numbers are typical published specs — always check your own printer. */
'use strict';

const FIRMWARES = {
  marlin: { name: 'Marlin', save: 'M500', short: 'Marlin' },
  klipper: { name: 'Klipper', save: 'SAVE_CONFIG', short: 'Klipper' },
  rrf: { name: 'RepRapFirmware (Duet)', save: 'M500', short: 'RRF' },
  'prusa-mk3': { name: 'Prusa firmware (MK3 / 8-bit)', save: 'M500', short: 'Prusa MK3' },
  'prusa-buddy': { name: 'Prusa Buddy firmware (MK4 / Mini / Core One)', save: 'M500', short: 'Prusa Buddy' },
  bambu: { name: 'Bambu Lab', save: '', short: 'Bambu' },
};

// [id, brand, name, firmware, shape, x, y, z, drive, maxSpeed, maxAccel, maxHotend, maxBed, shaper, originCenter]
const PRESET_ROWS = [
  ['ender3', 'Creality', 'Ender 3 / Ender 3 Pro', 'marlin', 'rect', 220, 220, 250, 'bowden', 150, 1000, 250, 100, false],
  ['ender3v2', 'Creality', 'Ender 3 V2 / V2 Neo', 'marlin', 'rect', 220, 220, 250, 'bowden', 150, 1000, 250, 100, false],
  ['ender3s1', 'Creality', 'Ender 3 S1 / S1 Pro', 'marlin', 'rect', 220, 220, 270, 'direct', 150, 1000, 260, 100, false],
  ['ender3v3se', 'Creality', 'Ender 3 V3 SE', 'marlin', 'rect', 220, 220, 250, 'direct', 250, 2500, 260, 100, false],
  ['ender3v3ke', 'Creality', 'Ender 3 V3 KE', 'klipper', 'rect', 220, 220, 240, 'direct', 500, 8000, 300, 100, true],
  ['ender5', 'Creality', 'Ender 5 / 5 Pro', 'marlin', 'rect', 220, 220, 300, 'bowden', 150, 1000, 250, 100, false],
  ['ender5s1', 'Creality', 'Ender 5 S1', 'marlin', 'rect', 220, 220, 280, 'direct', 250, 2000, 300, 110, false],
  ['cr10', 'Creality', 'CR-10 / CR-10 V2', 'marlin', 'rect', 300, 300, 400, 'bowden', 150, 1000, 250, 100, false],
  ['k1', 'Creality', 'K1 / K1C', 'klipper', 'rect', 220, 220, 250, 'direct', 600, 20000, 300, 100, true],
  ['k1max', 'Creality', 'K1 Max', 'klipper', 'rect', 300, 300, 300, 'direct', 600, 20000, 300, 100, true],
  ['mk3s', 'Prusa', 'Original Prusa MK3S+', 'prusa-mk3', 'rect', 250, 210, 210, 'direct', 200, 1250, 300, 120, false],
  ['mk4', 'Prusa', 'Original Prusa MK4 / MK4S', 'prusa-buddy', 'rect', 250, 210, 220, 'direct', 300, 4000, 290, 120, true],
  ['mini', 'Prusa', 'Original Prusa Mini / Mini+', 'prusa-buddy', 'rect', 180, 180, 180, 'bowden', 180, 1250, 280, 100, false],
  ['coreone', 'Prusa', 'Prusa Core One', 'prusa-buddy', 'rect', 250, 220, 270, 'direct', 400, 10000, 290, 120, true],
  ['x1c', 'Bambu Lab', 'X1 Carbon / X1E', 'bambu', 'rect', 256, 256, 256, 'direct', 500, 20000, 300, 110, true],
  ['p1s', 'Bambu Lab', 'P1S / P1P', 'bambu', 'rect', 256, 256, 256, 'direct', 500, 20000, 300, 100, true],
  ['a1', 'Bambu Lab', 'A1', 'bambu', 'rect', 256, 256, 256, 'direct', 500, 10000, 300, 100, true],
  ['a1mini', 'Bambu Lab', 'A1 mini', 'bambu', 'rect', 180, 180, 180, 'direct', 500, 10000, 300, 80, true],
  ['v24-250', 'Voron', 'Voron 2.4 (250)', 'klipper', 'rect', 250, 250, 230, 'direct', 300, 5000, 300, 120, true],
  ['v24-300', 'Voron', 'Voron 2.4 (300)', 'klipper', 'rect', 300, 300, 280, 'direct', 300, 5000, 300, 120, true],
  ['v24-350', 'Voron', 'Voron 2.4 (350)', 'klipper', 'rect', 350, 350, 330, 'direct', 300, 5000, 300, 120, true],
  ['trident', 'Voron', 'Voron Trident (300)', 'klipper', 'rect', 300, 300, 250, 'direct', 300, 5000, 300, 120, true],
  ['v02', 'Voron', 'Voron 0.2', 'klipper', 'rect', 120, 120, 120, 'direct', 300, 5000, 300, 110, true],
  ['switchwire', 'Voron', 'Voron Switchwire', 'klipper', 'rect', 250, 210, 240, 'direct', 250, 4000, 300, 110, true],
  ['sv06', 'Sovol', 'SV06', 'marlin', 'rect', 220, 220, 250, 'direct', 180, 1500, 300, 100, false],
  ['sv06plus', 'Sovol', 'SV06 Plus', 'marlin', 'rect', 300, 300, 340, 'direct', 180, 1500, 300, 100, false],
  ['sv07', 'Sovol', 'SV07', 'klipper', 'rect', 220, 220, 250, 'direct', 500, 8000, 300, 100, true],
  ['sv08', 'Sovol', 'SV08', 'klipper', 'rect', 350, 350, 345, 'direct', 700, 20000, 300, 100, true],
  ['n3pro', 'Elegoo', 'Neptune 3 Pro', 'marlin', 'rect', 225, 225, 280, 'direct', 180, 1500, 260, 110, false],
  ['n4', 'Elegoo', 'Neptune 4 / 4 Pro', 'klipper', 'rect', 225, 225, 265, 'direct', 500, 10000, 300, 110, true],
  ['n4max', 'Elegoo', 'Neptune 4 Max', 'klipper', 'rect', 420, 420, 480, 'direct', 500, 10000, 300, 110, true],
  ['kobra2', 'Anycubic', 'Kobra 2 / Kobra 2 Neo', 'marlin', 'rect', 220, 220, 250, 'direct', 250, 2500, 260, 110, false],
  ['kobra2max', 'Anycubic', 'Kobra 2 Max', 'marlin', 'rect', 420, 420, 500, 'direct', 250, 2500, 260, 110, false],
  ['kobra3', 'Anycubic', 'Kobra 3', 'klipper', 'rect', 250, 250, 260, 'direct', 600, 20000, 300, 110, true],
  ['vyper', 'Anycubic', 'Vyper', 'marlin', 'rect', 245, 245, 260, 'direct', 150, 1500, 260, 110, false],
  ['megas', 'Anycubic', 'i3 Mega S', 'marlin', 'rect', 210, 210, 205, 'direct', 100, 1000, 260, 100, false],
  ['kossel', 'Anycubic', 'Kossel Linear Plus (delta)', 'marlin', 'circle', 230, 230, 300, 'bowden', 200, 3000, 250, 100, false, true],
  ['flsunv400', 'FLSUN', 'V400 (delta)', 'klipper', 'circle', 300, 300, 410, 'direct', 400, 20000, 300, 100, true, true],
  ['sidewinderx2', 'Artillery', 'Sidewinder X2', 'marlin', 'rect', 300, 300, 400, 'direct', 150, 1000, 260, 110, false],
  ['tiny', 'DIY / custom', 'Tiny DIY printer (60 mm)', 'marlin', 'rect', 60, 60, 60, 'direct', 60, 800, 250, 80, false],
  ['custom', 'DIY / custom', 'Fully custom printer', 'marlin', 'rect', 200, 200, 200, 'direct', 100, 1000, 250, 100, false],
];

const PRESETS = PRESET_ROWS.map((r) => ({
  id: r[0], brand: r[1], name: r[2], firmware: r[3], bedShape: r[4], bedX: r[5], bedY: r[6], bedZ: r[7], drive: r[8],
  maxSpeed: r[9], maxAccel: r[10], maxHotend: r[11], maxBed: r[12], hasInputShaper: r[13], originCenter: !!r[14],
}));

function printerFromPreset(id, name) {
  const p = PRESETS.find((x) => x.id === id) || PRESETS[0];
  return {
    id: uid(), preset: p.id, name: name || p.name, firmware: p.firmware, bedShape: p.bedShape, bedX: p.bedX, bedY: p.bedY, bedZ: p.bedZ,
    originCenter: p.originCenter, nozzle: 0.4, filamentDia: 1.75, maxHotend: p.maxHotend, maxBed: p.maxBed, maxSpeed: p.maxSpeed,
    maxAccel: p.maxAccel, drive: p.drive, hasInputShaper: p.hasInputShaper, startGcode: '', endGcode: '', mesh: 'none',
    conn: { type: p.firmware === 'klipper' ? 'moonraker' : 'serial', baud: 115200, port: '', url: '', apiKey: '' },
    values: {}, screws: null, created: Date.now(),
  };
}

/* Typical starting points. Always follow the range printed on your spool. */
const FILAMENT_TYPES = {
  PLA: { temp: [190, 220], start: 205, bed: [50, 65], bedStart: 60, fan: 100, density: 1.24, speed: 'Any', notes: 'Easiest to print. Full cooling. Softens around 55–60 °C, so avoid hot cars.' },
  PETG: { temp: [225, 250], start: 235, bed: [70, 85], bedStart: 80, fan: 40, density: 1.27, speed: 'Medium', notes: 'Tough and slightly flexible. Strings easily — dry it and tune retraction. Can stick too well to smooth PEI; use glue stick or textured sheets.' },
  ABS: { temp: [230, 260], start: 245, bed: [95, 110], bedStart: 100, fan: 0, density: 1.04, speed: 'Medium', notes: 'Needs an enclosure to avoid warping and cracking. Ventilate — it gives off fumes.' },
  ASA: { temp: [235, 260], start: 250, bed: [90, 110], bedStart: 100, fan: 10, density: 1.07, speed: 'Medium', notes: 'Like ABS but UV-resistant for outdoor parts. Enclosure recommended; ventilate.' },
  TPU: { temp: [210, 235], start: 225, bed: [40, 60], bedStart: 50, fan: 60, density: 1.21, speed: 'Slow (15–30 mm/s)', notes: 'Flexible. Print slowly with minimal retraction; direct drive strongly recommended.' },
  Nylon: { temp: [240, 270], start: 255, bed: [70, 90], bedStart: 80, fan: 0, density: 1.14, speed: 'Medium', notes: 'Very strong and tough but absorbs water fast — dry before printing and print from a dry box. Use garolite or glue stick.' },
};

function filamentFromType(type, extra = {}) {
  const t = FILAMENT_TYPES[type] || FILAMENT_TYPES.PLA;
  return Object.assign({
    id: uid(), brand: 'Generic', type, colourName: 'Natural', colour: '#d8d8d8', diameter: 1.75,
    temp: t.start, tempMin: t.temp[0], tempMax: t.temp[1], bed: t.bedStart, fan: t.fan, density: t.density, created: Date.now(),
  }, extra);
}

/* Recommended order for a new printer */
const CHECKLIST = [
  { key: 'pid', test: 'pid', title: 'PID tuning', why: 'Steady hotend and bed temperatures' },
  { key: 'esteps', test: 'esteps', title: 'E-steps / rotation distance', why: 'The extruder pushes exactly as much filament as asked' },
  { key: 'flow', test: 'flow', title: 'Flow rate', why: 'Walls come out exactly as thick as planned' },
  { key: 'temp', test: 'temp', title: 'Temperature', why: 'Best temperature for this filament' },
  { key: 'retraction', test: 'retraction', title: 'Retraction', why: 'No strings or blobs when travelling' },
  { key: 'pa', test: 'pa', title: 'Pressure / linear advance', why: 'Sharp corners and even lines at any speed' },
  { key: 'shaper', test: 'shaper', title: 'Input shaping', why: 'No ringing or ghosting at high speed' },
  { key: 'dims', test: 'dims', title: 'Dimensional accuracy', why: 'Parts come out the right size and square' },
];

/* What Bambu Lab printers already do for you (depends on model and firmware). */
const BAMBU_AUTO = {
  pid: { auto: true, text: 'Heater control is handled by Bambu firmware — there is nothing to tune.' },
  esteps: { auto: true, text: 'The extruder is factory calibrated and not user-adjustable.' },
  flow: { auto: false, text: 'Use Bambu Studio → Calibration → Flow rate, or Tunie\'s flow patches (export).' },
  temp: { auto: false, text: 'Use Bambu Studio → Calibration → Temperature, or Tunie\'s temperature tower (export).' },
  retraction: { auto: false, text: 'Use Bambu Studio → Calibration → Retraction, or Tunie\'s retraction tower (export).' },
  pa: { auto: 'partly', text: '"Flow dynamics" (pressure advance) can run automatically before prints on X1 and A1 series; otherwise use Bambu Studio → Calibration → Flow dynamics.' },
  shaper: { auto: true, text: 'Vibration compensation (input shaping) runs during the printer\'s start-up calibration.' },
  dims: { auto: false, text: 'Not covered by Bambu — use Tunie\'s XYZ cube or accuracy frame.' },
};
