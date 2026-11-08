/* Tunie — firmware.js: the exact commands / config lines for each firmware.
   Part of the nrmluvdev maker suite. Created by nrmluvdev.
   Each builder returns code blocks: { title, text, send, lang } */
'use strict';

const r3 = (v) => fmt(v, 3), r4 = (v) => fmt(v, 4), r2 = (v) => fmt(v, 2);
const isMarlinish = (fw) => ['marlin', 'prusa-mk3', 'prusa-buddy', 'bambu'].includes(fw);

const FWC = {
  pidRun(fw, which, target, fan) {
    const out = [];
    const fanCmd = `M106 S${Math.round(255 * (fan ?? 100) / 100)} ; part fan on, like when printing`;
    if (which === 'hotend') {
      if (fw === 'klipper') out.push(`PID_CALIBRATE HEATER=extruder TARGET=${target}`);
      else if (fw === 'rrf') out.push(fanCmd, `M303 T0 S${target}`);
      else out.push(fanCmd, `M303 E0 S${target} C8 U1`);
    } else {
      if (fw === 'klipper') out.push(`PID_CALIBRATE HEATER=heater_bed TARGET=${target}`);
      else if (fw === 'rrf') out.push(`M303 H0 S${target}`);
      else out.push(`M303 E-1 S${target} C8 U1`);
    }
    return out.join('\n');
  },

  pidApply(fw, which, p, i, d, rrfLine) {
    const B = [];
    if (fw === 'klipper') {
      B.push({ title: 'Save (Klipper stores the tuned values for you)', text: 'SAVE_CONFIG', send: true });
      B.push({ title: 'Or paste into printer.cfg', lang: 'cfg', text: `[${which === 'hotend' ? 'extruder' : 'heater_bed'}]\ncontrol: pid\npid_Kp: ${r3(p)}\npid_Ki: ${r3(i)}\npid_Kd: ${r3(d)}` });
    } else if (fw === 'rrf') {
      B.push({ title: 'Apply and save', text: `${rrfLine || '; paste the M307 line the printer reported'}\nM500 ; needs M501 at the end of config.g`, send: true });
    } else {
      const cmd = which === 'hotend' ? 'M301' : 'M304';
      B.push({ title: 'Apply and save to EEPROM', text: `${cmd} P${r2(p)} I${r2(i)} D${r2(d)}\nM500`, send: true });
      if (fw === 'marlin') B.push({ title: 'Or set in Configuration.h', lang: 'cfg', text: which === 'hotend'
        ? `#define DEFAULT_Kp ${r2(p)}\n#define DEFAULT_Ki ${r2(i)}\n#define DEFAULT_Kd ${r2(d)}`
        : `#define DEFAULT_bedKp ${r2(p)}\n#define DEFAULT_bedKi ${r2(i)}\n#define DEFAULT_bedKd ${r2(d)}` });
    }
    return B;
  },

  esteps(fw, value, isRD) {
    if (fw === 'klipper') return [
      { title: 'Try it now (until restart)', text: `SET_EXTRUDER_ROTATION_DISTANCE EXTRUDER=extruder DISTANCE=${r4(value)}`, send: true },
      { title: 'Make it permanent — edit printer.cfg, then FIRMWARE_RESTART', lang: 'cfg', text: `[extruder]\nrotation_distance: ${r4(value)}` },
    ];
    if (fw === 'rrf') return [
      { title: 'Apply now', text: `M92 E${r2(value)}`, send: true },
      { title: 'Make it permanent — change the M92 line in config.g', lang: 'cfg', text: `M92 E${r2(value)}` },
    ];
    return [{ title: 'Apply and save to EEPROM', text: `M92 E${r2(value)}\nM500`, send: true }];
  },

  flowRuntime(pct) {
    return { title: 'Optional: apply on the printer right now (until restart)', text: `M221 S${Math.round(pct)}`, send: true };
  },

  pa(fw, k) {
    if (fw === 'klipper') return [
      { title: 'Try it now', text: `SET_PRESSURE_ADVANCE ADVANCE=${r4(k)}`, send: true },
      { title: 'Make it permanent — printer.cfg', lang: 'cfg', text: `[extruder]\npressure_advance: ${r4(k)}` },
      { title: 'Or per filament — add to the filament\'s start G-code in your slicer', text: `SET_PRESSURE_ADVANCE ADVANCE=${r4(k)}` },
    ];
    if (fw === 'rrf') return [
      { title: 'Apply now', text: `M572 D0 S${r4(k)}`, send: true },
      { title: 'Make it permanent — add to config.g (or filament config)', lang: 'cfg', text: `M572 D0 S${r4(k)}` },
    ];
    if (fw === 'prusa-buddy') return [
      { title: 'Apply now', text: `M572 S${r4(k)}`, send: true },
      { title: 'Per filament — add to the filament\'s start G-code in PrusaSlicer', text: `M572 S${r4(k)}` },
    ];
    if (fw === 'bambu') return [{ title: 'Bambu Studio / OrcaSlicer — Filament → Pressure advance (K)', text: `K = ${r4(k)}` }];
    return [
      { title: 'Apply and save', text: `M900 K${r4(k)}\nM500`, send: true },
      { title: 'Or per filament — add to the filament\'s start G-code in your slicer', text: `M900 K${r4(k)}` },
    ];
  },

  shaper(fw, fx, fy, type = 'mzv') {
    if (fw === 'klipper') return [
      { title: 'Try it now', text: `SET_INPUT_SHAPER SHAPER_FREQ_X=${r2(fx)} SHAPER_FREQ_Y=${r2(fy)} SHAPER_TYPE=${type}`, send: true },
      { title: 'Make it permanent — printer.cfg', lang: 'cfg', text: `[input_shaper]\nshaper_freq_x: ${r2(fx)}\nshaper_freq_y: ${r2(fy)}\nshaper_type: ${type}` },
    ];
    if (fw === 'rrf') return [{ title: 'Apply now and add to config.g (RRF uses one frequency — the lower one)', text: `M593 P"${type}" F${r2(Math.min(fx, fy))}`, send: true }];
    if (fw === 'marlin' || fw === 'prusa-buddy') return [{ title: 'Apply and save (needs input shaping enabled in firmware)', text: `M593 X F${r2(fx)}\nM593 Y F${r2(fy)}\nM500`, send: true }];
    return [{ title: 'Your firmware', text: `; X ${r2(fx)} Hz, Y ${r2(fy)} Hz\n; This firmware doesn't expose input shaping commands.` }];
  },

  accel(fw, a) {
    a = Math.round(a);
    if (fw === 'klipper') return [
      { title: 'Try it now', text: `SET_VELOCITY_LIMIT ACCEL=${a}`, send: true },
      { title: 'Make it permanent — printer.cfg', lang: 'cfg', text: `[printer]\nmax_accel: ${a}` },
    ];
    if (fw === 'rrf') return [{ title: 'Apply now and add to config.g', text: `M204 P${a} T${a}`, send: true }];
    if (fw === 'bambu') return [];
    return [{ title: 'Apply and save', text: `M204 P${a} T${a}\nM500`, send: true }];
  },

  zAdjust(fw, delta) {
    const d = r3(delta);
    if (fw === 'klipper') return [
      { title: 'Nudge while printing (live)', text: `SET_GCODE_OFFSET Z_ADJUST=${d} MOVE=1`, send: true },
      { title: 'Then save it (probe printers use the first line, endstop printers the second)', text: 'Z_OFFSET_APPLY_PROBE\n; or: Z_OFFSET_APPLY_ENDSTOP\nSAVE_CONFIG', send: true },
    ];
    if (fw === 'rrf') return [{ title: 'Babystep (live), then update G31 Z in config.g', text: `M290 S${d}`, send: true }];
    if (fw === 'bambu') return [];
    return [
      { title: 'Babystep while printing (live)', text: `M290 Z${d}`, send: true },
      { title: 'Probe printers: change the probe offset and save (use your current M851 Z plus the change)', text: `M851 ; shows the current offset\n; M851 Z<current ${delta < 0 ? '−' : '+'} ${fmt(Math.abs(delta), 3)}>\nM500` },
    ];
  },

  steps(fw, axes) {
    // axes: {X: new, Y: new, Z: new} in steps/mm (or rotation distance for Klipper)
    const keys = Object.keys(axes).filter((k) => isFinite(axes[k]));
    if (!keys.length) return [];
    if (fw === 'klipper') return [{ title: 'printer.cfg — rotation_distance (then FIRMWARE_RESTART)', lang: 'cfg', text: keys.map((k) => `[stepper_${k.toLowerCase()}]\nrotation_distance: ${r4(axes[k])}`).join('\n\n') }];
    const line = 'M92 ' + keys.map((k) => `${k}${r3(axes[k])}`).join(' ');
    if (fw === 'rrf') return [{ title: 'Apply now and update the M92 line in config.g', text: line, send: true }];
    return [{ title: 'Apply and save', text: line + '\nM500', send: true }];
  },

  skew(fw, ac, bd, ad, factor) {
    if (fw === 'klipper') return [
      { title: 'Needs [skew_correction] in printer.cfg. Then run, check, and save', text: `SET_SKEW XY=${r2(ac)},${r2(bd)},${r2(ad)}\nSKEW_PROFILE SAVE=tunie\nSAVE_CONFIG`, send: true },
      { title: 'Load it in your start G-code / PRINT_START macro', text: 'SKEW_PROFILE LOAD=tunie' },
    ];
    if (fw === 'rrf') return [{ title: 'Apply now and add to config.g (if it makes things worse, flip the sign)', text: `M556 S${r2(ad)} X${r3(factor * ad)}`, send: true }];
    if (fw === 'bambu') return [];
    return [
      { title: 'Apply and save (needs SKEW_CORRECTION enabled in Marlin)', text: `M852 S${fmt(factor, 5)}\nM500`, send: true },
      { title: 'Or set in Configuration.h', lang: 'cfg', text: `#define XY_DIAG_AC ${r2(ac)}\n#define XY_DIAG_BD ${r2(bd)}\n#define XY_SIDE_AD ${r2(ad)}` },
    ];
  },

  retraction(fw, len, speed) {
    const B = [];
    if (fw === 'klipper') B.push({ title: 'Only if you use firmware retraction ([firmware_retraction])', text: `SET_RETRACTION RETRACT_LENGTH=${r2(len)} RETRACT_SPEED=${Math.round(speed)}`, send: true });
    else if (isMarlinish(fw) && fw !== 'bambu') B.push({ title: 'Only if you use firmware retraction (G10/G11, FWRETRACT)', text: `M207 S${r2(len)} F${Math.round(speed * 60)}\nM500`, send: true });
    else if (fw === 'rrf') B.push({ title: 'Only if you use firmware retraction (G10/G11)', text: `M207 S${r2(len)} F${Math.round(speed * 60)}`, send: true });
    return B;
  },

  readCurrent(fw) {
    if (fw === 'klipper') return null;
    if (fw === 'rrf') return 'M92';
    return 'M503';
  },
};

/* Marlin's skew maths (same as Klipper's) */
function skewFactor(ac, bd, ad) {
  const side = Math.sqrt(2 * ac * ac + 2 * bd * bd - 4 * ad * ad) / 2;
  return Math.tan(Math.PI / 2 - Math.acos((ac * ac - side * side - ad * ad) / (2 * side * ad)));
}

/* Parse common printer replies */
const PARSE = {
  pid(text) {
    const t = String(text);
    let m = t.match(/pid_Kp\s*=\s*([\d.]+)\s+pid_Ki\s*=\s*([\d.]+)\s+pid_Kd\s*=\s*([\d.]+)/i);
    if (m) return { p: +m[1], i: +m[2], d: +m[3] };
    const grab = (l) => [...t.matchAll(new RegExp(`K${l}\\s*[:=]?\\s*([\\d.]+)`, 'g'))].pop();
    const kp = grab('p'), ki = grab('i'), kd = grab('d');
    if (kp && ki && kd) return { p: +kp[1], i: +ki[1], d: +kd[1] };
    m = t.match(/M30[14]\s+P([\d.]+)\s+I([\d.]+)\s+D([\d.]+)/i);
    if (m) return { p: +m[1], i: +m[2], d: +m[3] };
    m = t.match(/M307\s+H\d[^\n]*/i);
    if (m) return { rrf: m[0].trim() };
    return null;
  },
  esteps(text) {
    const m = String(text).match(/M92[^\n]*?E\s*([\d.]+)/i);
    return m ? +m[1] : null;
  },
  axisSteps(text) {
    const m = String(text).match(/M92\s+X\s*([\d.]+)\s+Y\s*([\d.]+)\s+Z\s*([\d.]+)/i);
    return m ? { X: +m[1], Y: +m[2], Z: +m[3] } : null;
  },
  shaper(text) {
    const t = String(text);
    const out = {};
    for (const ax of ['x', 'y']) {
      let m = t.match(new RegExp(`shaper_type_${ax}\\s*=\\s*(\\w+),\\s*shaper_freq_${ax}\\s*=\\s*([\\d.]+)`, 'i'));
      if (m) out[ax] = { type: m[1], f: +m[2] };
    }
    if (!out.x) {
      const all = [...t.matchAll(/Recommended shaper is (\w+) @ ([\d.]+) Hz/gi)];
      if (all[0]) out.x = { type: all[0][1], f: +all[0][2] };
      if (all[1]) out.y = { type: all[1][1], f: +all[1][2] };
    }
    return out.x ? out : null;
  },
};
