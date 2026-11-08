/* Tunie — tests.js: every calibration test, in plain English.
   Each test: what it checks, why it matters, what good looks like, how to measure,
   the settings for its G-code, what to measure, and the calculator.
   Part of the nrmluvdev maker suite. Created by nrmluvdev. */
'use strict';

const RATE5 = [['', '— choose —'], ['-2', 'Too high (gaps, round lines)'], ['-1', 'A bit high'], ['0', 'Just right'], ['1', 'A bit low'], ['2', 'Too low (scraping, ridges)']];

function slicerBlock(rows, title = 'Put these in your slicer') {
  return { title, text: rows.map(([k, v]) => `${k.padEnd(34, ' ')} ${v}`).join('\n') };
}
const optList = (vals, d = 2, suffix = '') => [['', '— choose —'], ...(vals || []).map((v) => [String(v), fmt(v, d) + suffix])];
const avg = (a) => { const v = a.filter((x) => isFinite(x)); return v.length ? v.reduce((s, x) => s + x, 0) / v.length : NaN; };
const isBowden = (c) => c.printer.drive === 'bowden';

function linreg(xs, ys) {
  const pts = xs.map((x, i) => [x, ys[i]]).filter(([x, y]) => isFinite(x) && isFinite(y));
  const n = pts.length;
  if (n < 2) return null;
  const mx = avg(pts.map((p) => p[0])), my = avg(pts.map((p) => p[1]));
  let sxy = 0, sxx = 0;
  pts.forEach(([x, y]) => { sxy += (x - mx) * (y - my); sxx += (x - mx) ** 2; });
  const a = sxy / sxx;
  return { a, b: my - a * mx, n };
}

const TESTS = [
  /* ================================================================ PID */
  {
    id: 'pid', title: 'PID tuning', icon: 'flame', tone: 'alt3', cat: 'Printer', check: 'pid', time: '5–15 min', kind: 'commands',
    summary: 'Teach the printer to heat up quickly and hold its temperatures steady.',
    learn: {
      what: 'Tunes how your printer controls its heaters, so the hotend and bed reach temperature quickly and then hold it steady instead of swinging up and down.',
      why: 'Temperature swings cause uneven extrusion, banding and "thermal runaway" errors. Do it first on a new printer, and again after changing the hotend, heater, thermistor, silicone sock or fan duct.',
      good: 'The temperature settles within about ±1 °C of the target and stays there while printing.',
      how: 'Nothing to measure. The printer heats and cools several times by itself (5–10 minutes each) and then reports three numbers: Kp, Ki and Kd. If you\'re connected, Tunie reads them for you; otherwise copy them from the printer\'s screen or console.',
    },
    diagram: () => DG.pid(),
    params: (c) => [
      { k: 'which', label: 'Heater', type: 'select', def: 'hotend', options: [['hotend', 'Hotend'], ['bed', 'Bed'], ['both', 'Both (one after the other)']] },
      { k: 'hotendTarget', label: 'Hotend target', unit: '°C', def: c.settings.hotend, step: 5, help: 'Your usual printing temperature.' },
      { k: 'bedTarget', label: 'Bed target', unit: '°C', def: c.settings.bed, step: 5 },
      { k: 'fan', label: 'Part fan during tune', unit: '%', def: c.settings.fan, step: 10, help: 'Same as when printing, so the tune matches real use.' },
    ],
    gen: 'pid',
    run(c) {
      const p = c.params;
      const blocks = [];
      if (p.which !== 'bed') blocks.push(codeBlock('Hotend autotune', FWC.pidRun(c.fw, 'hotend', p.hotendTarget, p.fan), { send: true }));
      if (p.which !== 'hotend') blocks.push(codeBlock('Bed autotune', FWC.pidRun(c.fw, 'bed', p.bedTarget, 0), { send: true }));
      return `<div class="callout warn">${icon('alert')}<div><p><b>Stay near the printer.</b> It will heat up to ${p.hotendTarget} °C${p.which !== 'hotend' ? ` / ${p.bedTarget} °C bed` : ''} several times. Start with the nozzle clean and nothing on the bed.</p></div></div>
        <div class="grid mt">${blocks.join('')}</div>
        <p class="muted small mt">Not connected? Type these into your printer's console (OctoPrint / Mainsail / Fluidd / Pronterface terminal), or save them as a G-code file and "print" it from the SD card.</p>`;
    },
    measure: (c) => ({
      intro: 'Paste what the printer printed at the end of the tune, or type the three numbers.',
      paste: true,
      fields: [
        ...(c.params.which !== 'bed' ? [{ k: 'kp', label: 'Hotend Kp' }, { k: 'ki', label: 'Hotend Ki' }, { k: 'kd', label: 'Hotend Kd' }] : []),
        ...(c.params.which !== 'hotend' ? [{ k: 'bkp', label: 'Bed Kp' }, { k: 'bki', label: 'Bed Ki' }, { k: 'bkd', label: 'Bed Kd' }] : []),
      ],
    }),
    onPaste(c, text, m) {
      // fills hotend first, then bed if both
      const parts = String(text).split(/(?=PID_CALIBRATE|M303|PID Autotune start|PID parameters)/i);
      const found = parts.map((t) => PARSE.pid(t)).filter(Boolean);
      if (!found.length) { const one = PARSE.pid(text); if (one) found.push(one); }
      if (!found.length) return false;
      if (c.params.which === 'bed') Object.assign(m, { bkp: found[0].p, bki: found[0].i, bkd: found[0].d, rrf: found[0].rrf });
      else {
        Object.assign(m, { kp: found[0].p, ki: found[0].i, kd: found[0].d, rrf: found[0].rrf });
        if (found[1]) Object.assign(m, { bkp: found[1].p, bki: found[1].i, bkd: found[1].d, rrfBed: found[1].rrf });
      }
      return true;
    },
    compute(c, m) {
      const stats = [], blocks = [], save = { printer: {} };
      if (c.params.which !== 'bed' && (isFinite(m.kp) || m.rrf)) {
        stats.push({ label: 'Hotend PID', value: isFinite(m.kp) ? `${fmt(m.kp)} / ${fmt(m.ki)} / ${fmt(m.kd)}` : 'from M307', hero: true, sub: 'Kp / Ki / Kd' });
        blocks.push(...FWC.pidApply(c.fw, 'hotend', m.kp, m.ki, m.kd, m.rrf));
        save.printer.pidHotend = { p: m.kp, i: m.ki, d: m.kd, target: c.params.hotendTarget };
      }
      if (c.params.which !== 'hotend' && (isFinite(m.bkp) || m.rrfBed)) {
        stats.push({ label: 'Bed PID', value: isFinite(m.bkp) ? `${fmt(m.bkp)} / ${fmt(m.bki)} / ${fmt(m.bkd)}` : 'from M307', sub: 'Kp / Ki / Kd' });
        blocks.push(...FWC.pidApply(c.fw, 'bed', m.bkp, m.bki, m.bkd, m.rrfBed));
        save.printer.pidBed = { p: m.bkp, i: m.bki, d: m.bkd, target: c.params.bedTarget };
      }
      if (!stats.length) return { error: 'Enter the Kp, Ki and Kd numbers (or paste the printer output).' };
      return { stats, blocks, save, summary: stats.map((s) => `${s.label}: ${s.value}`).join(', '),
        notes: ['Afterwards, heat to your usual temperature and watch the graph — it should settle and stay flat within about 1 °C.'] };
    },
  },

  /* ================================================================ E-steps */
  {
    id: 'esteps', title: 'E-steps / rotation distance', icon: 'ruler', tone: 'alt3', cat: 'Printer', check: 'esteps', time: '10 min', kind: 'commands',
    summary: 'Make sure 100 mm of filament asked for is 100 mm pushed.',
    learn: {
      what: 'Checks that when the printer asks for 100 mm of filament, the extruder really pushes 100 mm. Marlin, Prusa and RepRap call this "E-steps"; Klipper calls it "rotation distance".',
      why: 'If it pushes too much or too little, every print is over- or under-extruded and later tests (flow, pressure advance) won\'t come out right. Do it once per printer, and after changing the extruder or its gears.',
      good: 'Between 99 and 101 mm extruded — that means 19 to 21 mm left between the extruder and your mark.',
      how: 'Heat the nozzle, mark the filament exactly 120 mm above where it enters the extruder (a fine marker or a strip of tape), extrude 100 mm slowly, then measure from the extruder entry to the mark again.',
    },
    diagram: () => DG.esteps(),
    params: (c) => [
      { k: 'hotend', label: 'Nozzle temperature', unit: '°C', def: c.settings.hotend, step: 5 },
      { k: 'length', label: 'Extrude length', unit: 'mm', def: 100, step: 10 },
      { k: 'feed', label: 'Extrude speed', unit: 'mm/min', def: 100, step: 10, help: 'Slow, so the hotend keeps up and nothing slips.' },
      { k: 'mark', label: 'Mark distance', unit: 'mm', def: 120, step: 5, help: 'How far above the extruder you mark the filament.' },
    ],
    gen: 'esteps',
    run(c) {
      const p = c.params;
      const read = FWC.readCurrent(c.fw);
      return `<div class="callout">${icon('info')}<div><p><b>1.</b> Mark the filament exactly <b>${p.mark} mm</b> above where it enters the extruder.<br><b>2.</b> Send the commands below (they home, lift the nozzle, heat to ${p.hotend} °C, and slowly extrude ${p.length} mm).<br><b>3.</b> When it stops, measure from the extruder entry to your mark.</p></div></div>
        <div class="grid mt">${codeBlock('Extrude test', `G28\nG90\nG1 Z${Math.round(Math.min(50, c.printer.bedZ * .4))} F600\nM109 S${p.hotend}\nM83\nG92 E0\nG1 E${p.length} F${p.feed}\nM400`, { send: true })}
        ${read ? codeBlock('Read the current value (look for "M92 … E…")', read, { send: true }) : codeBlock('Klipper: current value is "rotation_distance" in the [extruder] section of printer.cfg', '; Tunie reads it automatically over Moonraker on the next step')}</div>`;
    },
    measure: (c) => ({
      intro: 'Measure from the top of the extruder inlet to the mark.',
      readCurrent: true,
      fields: [
        c.fw === 'klipper'
          ? { k: 'current', label: 'Current rotation distance', unit: 'mm', def: c.printer.values?.rotationDistance, help: 'From printer.cfg [extruder].' }
          : { k: 'current', label: 'Current E-steps', unit: 'steps/mm', def: c.printer.values?.eSteps, help: c.fw === 'marlin' ? 'Send M503 and look for "M92 … E…". Ender 3 stock is 93.' : 'Send M503 / M92 to read it.' },
        { k: 'left', label: 'Distance left to the mark', unit: 'mm', help: `Ideal: ${c.params.mark - c.params.length} mm` },
      ],
    }),
    compute(c, m) {
      const req = c.params.length;
      const extruded = c.params.mark - m.left;
      if (!isFinite(m.left)) return { error: 'Enter how much is left between the extruder and the mark.' };
      if (extruded <= 0) return { error: 'That would mean nothing was extruded — check the numbers.' };
      const pct = (extruded / req - 1) * 100;
      const out = {
        stats: [{ label: 'Actually extruded', value: fmt(extruded, 2), unit: 'mm', sub: `${pct >= 0 ? '+' : ''}${fmt(pct, 1)} % vs ${req} mm` }],
        blocks: [], save: { printer: {} }, notes: [],
      };
      if (Math.abs(pct) <= 1) out.notes.push('That\'s within 1 % — already good. You can still apply the fine-tuned value below.');
      if (Math.abs(pct) > 15) out.notes.push('That\'s a big difference. Check the extruder tension arm, that the filament isn\'t tangled, and that the nozzle isn\'t partly blocked — then repeat before changing settings.');
      if (!isFinite(m.current)) { out.notes.push('Enter your current value to get the new one.'); return out; }
      if (c.fw === 'klipper') {
        const nv = m.current * extruded / req;
        out.stats.unshift({ label: 'New rotation distance', value: fmt(nv, 4), unit: 'mm', hero: true, sub: `was ${fmt(m.current, 4)}` });
        out.blocks = FWC.esteps(c.fw, nv, true);
        out.save.printer.rotationDistance = round(nv, 4);
        out.summary = `rotation_distance ${fmt(nv, 4)}`;
      } else {
        const nv = m.current * req / extruded;
        out.stats.unshift({ label: 'New E-steps', value: fmt(nv, 2), unit: 'steps/mm', hero: true, sub: `was ${fmt(m.current, 2)}` });
        out.blocks = FWC.esteps(c.fw, nv);
        out.save.printer.eSteps = round(nv, 2);
        out.summary = `E-steps ${fmt(nv, 2)}`;
      }
      out.notes.push('Repeat once with the new value to confirm you now get 99–101 mm.');
      return out;
    },
  },

  /* ================================================================ Flow */
  {
    id: 'flow', title: 'Flow rate', icon: 'drop', tone: '', cat: 'Filament', check: 'flow', time: '10–25 min', kind: 'print',
    summary: 'Fine-tune how much plastic each line gets, per filament.',
    learn: {
      what: 'Fine-tunes the "flow rate" (also called extrusion multiplier or flow ratio) — how much plastic comes out for each line — for one specific filament.',
      why: 'Too much flow gives blobby, rough tops and parts that are too big. Too little gives gaps, see-through walls and weak layers. Every filament brand is slightly different, so it\'s worth doing per spool type.',
      good: 'Single-wall cube: the measured wall is within ±0.02 mm of the expected width. Patches: a smooth, closed top with no gaps and no ridges.',
      how: 'Single-wall method: measure the wall thickness with calipers in the middle of each side (not the corners) and enter the readings. Patch method: run your fingernail over the tops and pick the smoothest one.',
    },
    diagram: (c) => c.method === 'flow-patches' ? DG.flowPatches() : DG.flowWall(),
    methods: [
      { id: 'flow-wall', label: 'Single-wall cube', desc: 'Most precise — needs calipers.' },
      { id: 'flow-patches', label: 'Flow-step patches', desc: 'Visual — no calipers needed.' },
    ],
    params: (c) => c.method === 'flow-patches' ? [
      { k: 'from', label: 'Lowest change', unit: '%', def: -10, step: 1 },
      { k: 'to', label: 'Highest change', unit: '%', def: 10, step: 1 },
      { k: 'step', label: 'Step', unit: '%', def: 5, step: 1 },
      { k: 'patch', label: 'Patch size', unit: 'mm', def: c.small ? 12 : 18, step: 1 },
    ] : [
      { k: 'size', label: 'Cube size', unit: 'mm', def: c.small ? 15 : 20, step: 1 },
      { k: 'height', label: 'Height', unit: 'mm', def: 10, step: 1 },
    ],
    measure: (c) => c.method === 'flow-patches' ? ({
      intro: 'Which patch has the smoothest, most even top surface?',
      fields: [{ k: 'best', label: 'Best patch', type: 'select', options: optList(c.info.offsets, 1, ' %') }, { k: 'current', label: 'Flow used for this print', unit: '%', def: c.settings.flow }],
    }) : ({
      intro: `Measure the wall on all four sides. It should be ${fmt(c.info.expectedWall ?? c.settings.lineW, 3)} mm.`,
      fields: [
        { k: 'm1', label: 'Front', unit: 'mm', step: 0.01 }, { k: 'm2', label: 'Right', unit: 'mm', step: 0.01 },
        { k: 'm3', label: 'Back', unit: 'mm', step: 0.01 }, { k: 'm4', label: 'Left', unit: 'mm', step: 0.01 },
        { k: 'expected', label: 'Expected wall', unit: 'mm', def: c.info.expectedWall ?? c.settings.lineW, step: 0.01 },
        { k: 'current', label: 'Flow used for this print', unit: '%', def: c.settings.flow },
      ],
    }),
    compute(c, m) {
      let nf;
      const cur = isFinite(m.current) ? m.current : 100;
      const stats = [];
      if (c.method === 'flow-patches') {
        if (m.best === '' || m.best === undefined) return { error: 'Choose the best patch.' };
        nf = cur * (1 + num(m.best) / 100);
        stats.push({ label: 'Best patch', value: (num(m.best) > 0 ? '+' : '') + m.best, unit: '%' });
      } else {
        const a = avg([m.m1, m.m2, m.m3, m.m4]);
        if (!isFinite(a)) return { error: 'Enter at least one wall measurement.' };
        nf = cur * m.expected / a;
        stats.push({ label: 'Average wall', value: fmt(a, 3), unit: 'mm', sub: `expected ${fmt(m.expected, 3)} mm` });
      }
      nf = clamp(nf, 50, 150);
      const notes = [];
      if (Math.abs(nf - cur) > 10) notes.push('That\'s a big change. Double-check E-steps first — flow should only need small corrections (a few %).');
      return {
        stats: [{ label: 'New flow rate', value: fmt(nf, 1), unit: '%', hero: true, sub: `was ${fmt(cur, 1)} %` }, ...stats],
        blocks: [slicerBlock([
          ['PrusaSlicer → Filament → Extrusion multiplier', fmt(nf / 100, 3)],
          ['OrcaSlicer / Bambu Studio → Flow ratio', fmt(nf / 100, 3)],
          ['Cura → Material → Flow', fmt(nf, 1) + ' %'],
        ]), FWC.flowRuntime(nf)],
        save: { profile: { flow: round(nf, 1) } }, notes, summary: `Flow ${fmt(nf, 1)} %`,
      };
    },
  },

  /* ================================================================ Temperature */
  {
    id: 'temp', gen: 'temp-tower', title: 'Temperature tower', icon: 'thermo', tone: 'alt', cat: 'Filament', check: 'temp', time: '45–75 min', kind: 'print',
    summary: 'Find the best printing temperature for a filament.',
    learn: {
      what: 'Prints a tower where each band is printed at a different temperature. Each band has its temperature printed on the front, a gap for stringing, and a bridge on top.',
      why: 'Every filament has a sweet spot. Too hot gives strings, blobs and droopy bridges. Too cold gives weak layers that snap apart, a rough surface and under-extrusion.',
      good: 'A clean bridge at the top of the band, few strings between the pillars, smooth walls — and the band doesn\'t snap easily when you flex it.',
      how: 'Look at each band in good light. Pick the coolest band that still looks good, then check it\'s strong by trying to snap the tower at that band with pliers.',
    },
    diagram: (c) => DG.tempTower((c.info.bands || [220, 215, 210, 205, 200]).slice(0, 5).map((v) => String(v))),
    params: (c) => {
      const f = c.filament;
      const hi = Math.min(c.printer.maxHotend, f.tempMax ?? f.temp + 15);
      const lo = f.tempMin ?? f.temp - 15;
      return [
        { k: 'start', label: 'Bottom band (hottest)', unit: '°C', def: hi, step: 5, help: 'Towers go from hot at the bottom to cool at the top.' },
        { k: 'end', label: 'Top band (coolest)', unit: '°C', def: lo, step: 5 },
        { k: 'step', label: 'Step per band', unit: '°C', def: 5, step: 1 },
        { k: 'bandH', label: 'Band height', unit: 'mm', def: 10, step: 1, min: 8 },
      ];
    },
    measure: (c) => ({
      intro: 'Which band looks best overall (and is still strong)?',
      fields: [{ k: 'best', label: 'Best band', type: 'select', options: optList(c.info.bands, 0, ' °C') }],
    }),
    compute(c, m) {
      if (!m.best) return { error: 'Choose the best band.' };
      const t = num(m.best);
      const notes = [];
      const bands = c.info.bands || [];
      if (bands.length && (t === Math.max(...bands) || t === Math.min(...bands))) notes.push('Your best band is at the edge of the tested range — consider printing another tower that goes a little further in that direction.');
      notes.push('Many people print the first layer 5 °C hotter for better bed adhesion.');
      return {
        stats: [{ label: 'Printing temperature', value: fmt(t, 0), unit: '°C', hero: true, sub: `${c.filament.brand} ${c.filament.type}` }, { label: 'First layer', value: fmt(t + 5, 0), unit: '°C', sub: 'suggested' }],
        blocks: [slicerBlock([
          ['PrusaSlicer → Filament → Temperature', `${t} °C (first layer ${t + 5} °C)`],
          ['OrcaSlicer / Bambu → Nozzle temperature', `${t} °C (initial layer ${t + 5} °C)`],
          ['Cura → Printing Temperature', `${t} °C (initial layer ${t + 5} °C)`],
        ])],
        save: { profile: { temp: t } }, notes, summary: `Temperature ${t} °C`,
      };
    },
  },

  /* ================================================================ Retraction */
  {
    id: 'retraction', gen: 'retract-tower', title: 'Retraction tower', icon: 'retract', tone: 'alt', cat: 'Filament', check: 'retraction', time: '30–50 min', kind: 'print',
    summary: 'Stop strings and blobs when the nozzle travels.',
    learn: {
      what: 'Prints two pillars side by side. Each band uses a different retraction distance (or speed), labelled on the front of the left pillar.',
      why: 'When the nozzle travels between parts, melted plastic oozes out and leaves strings. Retraction pulls the filament back a little to stop that — but too much causes gaps, clicking and even jams.',
      good: 'The lowest band with no strings between the pillars. More retraction than needed only slows printing and wears the filament.',
      how: 'Hold the tower against a dark background and look between the pillars from the side. Find the lowest band that is clean. Print the temperature tower first — temperature matters more than retraction.',
    },
    diagram: () => DG.retraction(),
    params: (c) => {
      const b = isBowden(c);
      return [
        { k: 'mode', label: 'What to vary', type: 'select', def: 'distance', options: [['distance', 'Distance'], ['speed', 'Speed']], str: true },
        { k: 'start', label: 'Start', unit: 'mm or mm/s', def: b ? 2 : 0.2, step: 0.1 },
        { k: 'end', label: 'End', unit: 'mm or mm/s', def: b ? 7 : 2, step: 0.1 },
        { k: 'step', label: 'Step', def: b ? 0.5 : 0.2, step: 0.1 },
        { k: 'bandH', label: 'Band height', unit: 'mm', def: 6, step: 0.5 },
      ];
    },
    paramsHint: 'For speed, try 20 → 60 mm/s in steps of 10. Distance uses the speed in Print settings, and vice versa.',
    measure: (c) => ({
      intro: `Which is the lowest band without strings? (${c.info.mode === 'speed' ? 'speeds in mm/s' : 'distances in mm'})`,
      fields: [{ k: 'best', label: 'Best band', type: 'select', options: optList(c.info.bands, 2, c.info.mode === 'speed' ? ' mm/s' : ' mm') }],
    }),
    compute(c, m) {
      if (!m.best) return { error: 'Choose the best band.' };
      const v = num(m.best);
      const speedMode = c.info.mode === 'speed';
      const len = speedMode ? c.settings.retractLen : v, spd = speedMode ? v : c.settings.retractSpeed;
      const notes = [];
      if (!speedMode && v === Math.min(...(c.info.bands || [v]))) notes.push('Even the lowest band was clean — your printer may need very little retraction.');
      if (!speedMode && v === Math.max(...(c.info.bands || [v]))) notes.push('Only the highest band was clean. Before going higher, try a cooler temperature and make sure the filament is dry.');
      return {
        stats: [{ label: speedMode ? 'Retraction speed' : 'Retraction distance', value: fmt(v, 2), unit: speedMode ? 'mm/s' : 'mm', hero: true },
          { label: speedMode ? 'Distance used' : 'Speed used', value: fmt(speedMode ? len : spd, 2), unit: speedMode ? 'mm' : 'mm/s' }],
        blocks: [slicerBlock([
          ['PrusaSlicer → Filament overrides → Retraction', `length ${fmt(len, 2)} mm, speed ${fmt(spd, 0)} mm/s`],
          ['OrcaSlicer / Bambu → Filament → Retraction', `length ${fmt(len, 2)} mm, speed ${fmt(spd, 0)} mm/s`],
          ['Cura → Retraction Distance / Speed', `${fmt(len, 2)} mm / ${fmt(spd, 0)} mm/s`],
        ]), ...FWC.retraction(c.fw, len, spd)],
        save: { profile: speedMode ? { retractSpeed: v } : { retractLen: v, retractSpeed: spd } }, notes,
        summary: `Retraction ${fmt(len, 2)} mm @ ${fmt(spd, 0)} mm/s`,
      };
    },
  },

  /* ================================================================ Pressure / linear advance */
  {
    id: 'pa', title: 'Pressure / linear advance', icon: 'zap', tone: '', cat: 'Filament', check: 'pa', time: '10–30 min', kind: 'print',
    summary: 'Sharp corners and even lines when the speed changes.',
    learn: {
      what: 'Pressure advance (Klipper, RepRap, Prusa) — called linear advance in Marlin — makes the extruder push a little ahead when the print head speeds up, and ease off when it slows down.',
      why: 'Without it, pressure in the nozzle lags behind speed changes: corners bulge, line ends blob and gaps appear after corners. It matters more at higher speeds and much more with bowden tubes.',
      good: 'Lines: the line whose fast middle part is as thick and even as the slow ends, with no bulge or gap where the speed changes. Tower: corners that are sharp — not bulging, not rounded.',
      how: 'Lines: look closely (a torch at a low angle helps) and pick the most even line — the number is printed next to it. Tower: find the height where the corners look best and measure it from the bed with calipers.',
    },
    diagram: (c) => c.method === 'pa-tower' ? DG.paTower() : DG.paLines(),
    methods: [
      { id: 'pa-lines', label: 'Line pattern', desc: 'Fast, one layer, read the number.' },
      { id: 'pa-tower', label: 'Tower', desc: 'Measure the best height.' },
    ],
    params: (c) => {
      const b = isBowden(c);
      return c.method === 'pa-tower' ? [
        { k: 'start', label: 'Start value', def: 0, step: 0.001 },
        { k: 'factor', label: 'Increase per mm', def: b ? 0.02 : 0.005, step: 0.001, help: 'Direct drive ~0.005, bowden ~0.02.' },
        { k: 'height', label: 'Height', unit: 'mm', def: c.small ? 20 : 30, step: 1 },
        { k: 'size', label: 'Tower size', unit: 'mm', def: c.small ? 20 : 30, step: 1 },
        { k: 'speedTower', label: 'Speed', unit: 'mm/s', def: Math.min(100, c.printer.maxSpeed), step: 5 },
        { k: 'accelTower', label: 'Acceleration', unit: 'mm/s²', def: 500, step: 100 },
      ] : [
        { k: 'start', label: 'Start value', def: 0, step: 0.001 },
        { k: 'end', label: 'End value', def: b ? 1.0 : 0.1, step: 0.005 },
        { k: 'step', label: 'Step', def: b ? 0.05 : 0.005, step: 0.001 },
        { k: 'slow', label: 'Slow speed', unit: 'mm/s', def: 20, step: 5 },
        { k: 'fast', label: 'Fast speed', unit: 'mm/s', def: Math.min(100, c.printer.maxSpeed), step: 10 },
      ];
    },
    measure: (c) => c.method === 'pa-tower' ? ({
      intro: 'Measure the height (from the bed) where the corners look best.',
      fields: [{ k: 'height', label: 'Best height', unit: 'mm', step: 0.1 }],
    }) : ({
      intro: 'Which line is the most even?',
      fields: [{ k: 'best', label: 'Best line', type: 'select', options: optList(c.info.values, 4) }],
    }),
    compute(c, m) {
      let k;
      if (c.method === 'pa-tower') {
        if (!isFinite(m.height)) return { error: 'Enter the best height.' };
        k = c.params.start + c.params.factor * m.height;
      } else {
        if (!m.best) return { error: 'Choose the best line.' };
        k = num(m.best);
      }
      k = round(k, 4);
      const name = c.fw === 'marlin' || c.fw === 'prusa-mk3' ? 'Linear advance K' : 'Pressure advance';
      const notes = [];
      if (c.fw === 'marlin') notes.push('Marlin needs LIN_ADVANCE enabled. If M900 is "unknown command", your firmware was built without it.');
      if (c.fw === 'bambu') notes.push('Enter this as the filament\'s Pressure advance (K) in Bambu Studio / OrcaSlicer, or let the printer\'s flow dynamics calibration handle it.');
      return {
        stats: [{ label: name, value: fmt(k, 4), hero: true, sub: `${c.filament.brand} ${c.filament.type}` }],
        blocks: [...FWC.pa(c.fw, k), slicerBlock([['OrcaSlicer → Filament → Pressure advance', `enable, K = ${fmt(k, 4)}`], ['PrusaSlicer → Filament → Start G-code', c.fw === 'klipper' ? `SET_PRESSURE_ADVANCE ADVANCE=${fmt(k, 4)}` : (c.fw === 'prusa-buddy' ? `M572 S${fmt(k, 4)}` : `M900 K${fmt(k, 4)}`)]], 'Per-filament slicer setting')],
        save: { profile: { pa: k } }, notes, summary: `${name} ${fmt(k, 4)}`,
      };
    },
  },

  /* ================================================================ Max volumetric flow */
  {
    id: 'maxflow', gen: 'maxflow', title: 'Max volumetric flow', icon: 'flask', tone: 'alt2', cat: 'Filament', time: '20–40 min', kind: 'print',
    summary: 'How fast your hotend can melt plastic — so you can print as fast as is safe.',
    learn: {
      what: 'Prints a round tower that gets faster as it gets taller, until the hotend can\'t melt plastic fast enough.',
      why: 'Every hotend has a limit, measured in mm³ of plastic per second. Above it you get thin, weak walls and a clicking extruder. Setting this limit in your slicer lets it print as fast as is safe without you guessing speeds.',
      good: 'Find the height where the wall first looks thin, rough or gappy — or where you heard the extruder click.',
      how: 'Measure from the top of the solid base to the first bad layer. Tick marks stick out every 5 mm to help you read the height. Tunie turns the height into mm³/s.',
    },
    diagram: () => DG.maxflow(),
    params: (c) => [
      { k: 'start', label: 'Start flow', unit: 'mm³/s', def: 5, step: 1 },
      { k: 'end', label: 'End flow', unit: 'mm³/s', def: c.printer.maxSpeed >= 300 ? 35 : 22, step: 1 },
      { k: 'perMM', label: 'Increase per mm height', unit: 'mm³/s', def: 0.5, step: 0.1 },
      { k: 'lineWidth', label: 'Line width', unit: 'mm', def: round(c.printer.nozzle * 1.2, 2), step: 0.01 },
      { k: 'layerHeight', label: 'Layer height', unit: 'mm', def: round(c.printer.nozzle * 0.6, 2), step: 0.01 },
      { k: 'radius', label: 'Tower radius', unit: 'mm', def: c.small ? 10 : 20, step: 1 },
    ],
    paramsHint: 'Print hot (top of the filament\'s range) — that\'s how you\'ll print fast.',
    measure: () => ({
      intro: 'Height from the top of the solid base to the first bad layer.',
      fields: [{ k: 'height', label: 'Height of first problem', unit: 'mm', step: 0.5 }],
    }),
    compute(c, m) {
      if (!isFinite(m.height)) return { error: 'Enter the height where problems start.' };
      const f = c.params.start + m.height * c.params.perMM;
      const safe = f * 0.85;
      const lw = c.settings.lineW, lh = c.settings.layerH;
      return {
        stats: [{ label: 'Safe max flow', value: fmt(safe, 1), unit: 'mm³/s', hero: true, sub: '85 % of the measured limit' },
          { label: 'Measured limit', value: fmt(f, 1), unit: 'mm³/s' },
          { label: 'Max speed for normal lines', value: fmt(safe / (lw * lh), 0), unit: 'mm/s', sub: `${fmt(lw, 2)} × ${fmt(lh, 2)} mm lines` }],
        blocks: [slicerBlock([
          ['PrusaSlicer → Filament → Max volumetric speed', fmt(safe, 1) + ' mm³/s'],
          ['OrcaSlicer / Bambu → Max volumetric speed', fmt(safe, 1) + ' mm³/s'],
          ['Cura (no setting) → keep speeds under', fmt(safe / (lw * lh), 0) + ' mm/s at ' + fmt(lh, 2) + ' mm layers'],
        ])],
        save: { profile: { maxFlow: round(safe, 1) } }, notes: ['Hotter printing raises the limit; repeat if you change temperature a lot or swap the nozzle/hotend.'],
        summary: `Max flow ${fmt(safe, 1)} mm³/s`,
      };
    },
  },

  /* ================================================================ Input shaper */
  {
    id: 'shaper', gen: 'ringing-tower', title: 'Input shaper / ringing', icon: 'wave', tone: '', cat: 'Printer', check: 'shaper', time: '30–45 min', kind: 'print',
    summary: 'Cancel ghosting/ringing so you can print faster and cleaner.',
    learn: {
      what: 'Measures how your printer\'s frame vibrates, so input shaping can cancel the "ringing" (ghost echoes of edges) that appears after sharp corners.',
      why: 'Ringing limits how fast you can print cleanly. Input shaping (Klipper, Marlin 2.1+, Prusa MK4/Core One, RepRap 3.4+) cancels it, which lets you raise acceleration a lot without ugly surfaces.',
      good: 'With the right frequency set, the ripples after corners disappear even at high acceleration.',
      how: 'The tower is printed with input shaping switched off and acceleration rising band by band. On the face marked X, find ripples just after a corner, count several and measure how far they span. Do the same on the Y face. Klipper users with an accelerometer (ADXL345 etc.) can measure this automatically — see the measure step.',
    },
    diagram: () => DG.ringing(),
    params: (c) => [
      { k: 'speedTower', label: 'Speed', unit: 'mm/s', def: Math.min(100, c.printer.maxSpeed), step: 10, help: 'Fast enough to make ringing visible.' },
      { k: 'accelStart', label: 'Start acceleration', unit: 'mm/s²', def: Math.max(100, Math.round(c.printer.maxAccel * 0.3 / 100) * 100), step: 100 },
      { k: 'accelStep', label: 'Increase per band', unit: 'mm/s²', def: Math.max(50, Math.floor((c.printer.maxAccel - Math.max(100, Math.round(c.printer.maxAccel * 0.3 / 100) * 100)) / 7 / 50) * 50), step: 50, help: 'Defaults end at your profile\'s max acceleration.' },
      { k: 'bands', label: 'Bands', def: 8, step: 1 },
      { k: 'bandH', label: 'Band height', unit: 'mm', def: 5, step: 1 },
      { k: 'size', label: 'Tower size', unit: 'mm', def: c.small ? 30 : 40, step: 5 },
    ],
    paramsHint: 'Tick "has input shaping" on the printer profile so Tunie switches it off for the test.',
    measure: (c) => ({
      intro: 'Count the ripples after a corner on each face and measure the distance they cover.',
      klipperAccel: c.fw === 'klipper',
      fields: [
        { k: 'v', label: 'Print speed', unit: 'mm/s', def: c.info.speed ?? c.params.speedTower },
        { k: 'nx', label: 'X face: ripples counted', def: 5, step: 1 }, { k: 'dx', label: 'X face: distance', unit: 'mm', step: 0.1 },
        { k: 'ny', label: 'Y face: ripples counted', def: 5, step: 1 }, { k: 'dy', label: 'Y face: distance', unit: 'mm', step: 0.1 },
        { k: 'goodBand', label: 'Last clean band (accel)', type: 'select', options: [['', '— optional —'], ...(c.info.bands || []).map((a) => [String(a), fmt(a, 0) + ' mm/s²'])] },
        { k: 'shaperType', label: 'Shaper type', type: 'select', def: 'mzv', options: [['mzv', 'MZV (recommended start)'], ['ei', 'EI (more smoothing)'], ['zv', 'ZV (sharpest)'], ['2hump_ei', '2HUMP_EI'], ['3hump_ei', '3HUMP_EI']] },
        { k: 'fx', label: 'Or: X frequency (from accelerometer)', unit: 'Hz', step: 0.1 },
        { k: 'fy', label: 'Or: Y frequency', unit: 'Hz', step: 0.1 },
      ],
    }),
    onPaste(c, text, m) {
      const r = PARSE.shaper(text);
      if (!r) return false;
      m.fx = r.x.f; if (r.y) m.fy = r.y.f; m.shaperType = (r.x.type || 'mzv').toLowerCase();
      return true;
    },
    compute(c, m) {
      let fx = m.fx, fy = m.fy;
      if (!isFinite(fx) && isFinite(m.dx) && m.dx > 0) fx = m.v * m.nx / m.dx;
      if (!isFinite(fy) && isFinite(m.dy) && m.dy > 0) fy = m.v * m.ny / m.dy;
      if (!isFinite(fx) && !isFinite(fy)) return { error: 'Enter ripple counts and distances (or frequencies from an accelerometer).' };
      if (!isFinite(fx)) fx = fy; if (!isFinite(fy)) fy = fx;
      const stats = [{ label: 'X resonance', value: fmt(fx, 1), unit: 'Hz', hero: true }, { label: 'Y resonance', value: fmt(fy, 1), unit: 'Hz', hero: true }];
      const blocks = [...FWC.shaper(c.fw, fx, fy, m.shaperType || 'mzv')];
      const save = { printer: { shaperX: round(fx, 1), shaperY: round(fy, 1), shaperType: m.shaperType || 'mzv' } };
      const notes = [];
      if (fx < 15 || fy < 15 || fx > 150 || fy > 150) notes.push('That frequency is unusual (most printers are 25–100 Hz). Re-check the count and distance.');
      if (m.goodBand) {
        const a = num(m.goodBand);
        stats.push({ label: 'Clean acceleration (no shaping)', value: fmt(a, 0), unit: 'mm/s²', sub: 'with shaping you can usually go higher' });
        blocks.push(...FWC.accel(c.fw, a));
        save.printer.cleanAccel = a;
      }
      notes.push('Re-print the tower with input shaping on to confirm the ripples have gone, then raise acceleration.');
      return { stats, blocks, save, notes, summary: `Input shaper X ${fmt(fx, 1)} Hz, Y ${fmt(fy, 1)} Hz` };
    },
  },

  /* ================================================================ First layer */
  {
    id: 'firstlayer', gen: 'first-layer', title: 'First layer', icon: 'layers', tone: 'alt2', cat: 'Printer', time: '5–10 min', kind: 'print',
    summary: 'Check Z offset and bed level with patches in the centre and corners.',
    learn: {
      what: 'Prints thin patches in the centre and near each corner, so you can check the gap between the nozzle and the bed (Z offset) and whether the bed is level.',
      why: 'Almost every failed print starts with a bad first layer. Too high and it won\'t stick; too low and the nozzle scrapes, there\'s nowhere for the plastic to go, and you get ridges and clogs.',
      good: 'Lines are slightly squished and touch each other, the top is smooth, and every patch looks the same.',
      how: 'Watch it print and then look at each patch closely (and feel it). If all patches are off the same way, change the Z offset. If they differ, level the bed — the Bed mesh viewer can tell you which screws to turn.',
    },
    diagram: () => DG.firstLayer(),
    params: (c) => [
      { k: 'patch', label: 'Patch size', unit: 'mm', def: c.small ? 15 : 30, step: 1 },
      { k: 'layers', label: 'Layers', def: 1, step: 1, min: 1, max: 3 },
    ],
    measure: (c) => ({
      intro: 'Rate each patch.',
      fields: [
        { k: 'c', label: 'Centre', type: 'select', options: RATE5 },
        { k: 'fl', label: 'Front left', type: 'select', options: RATE5 }, { k: 'fr', label: 'Front right', type: 'select', options: RATE5 },
        { k: 'bl', label: 'Back left', type: 'select', options: RATE5 }, { k: 'br', label: 'Back right', type: 'select', options: RATE5 },
      ],
    }),
    compute(c, m) {
      const vals = ['c', 'fl', 'fr', 'bl', 'br'].map((k) => m[k] === '' || m[k] === undefined ? NaN : num(m[k]));
      const got = vals.filter(isFinite);
      if (!got.length) return { error: 'Rate at least the centre patch.' };
      const mean = avg(got);
      const spread = Math.max(...got) - Math.min(...got);
      const dz = round(mean * 0.025, 3);
      const notes = [];
      if (spread >= 2) notes.push('The patches differ a lot — level the bed (or re-run your bed mesh) before changing the Z offset. Open the Bed mesh viewer for screw turns.');
      else if (spread >= 1) notes.push('Small differences between patches — a bed mesh will usually take care of these.');
      if (Math.abs(dz) < 0.005) notes.push('Your Z offset looks good.');
      const blocks = Math.abs(dz) >= 0.005 ? FWC.zAdjust(c.fw, -dz) : [];
      return {
        stats: [{ label: 'Suggested Z change', value: (dz > 0 ? '+' : dz < 0 ? '−' : '') + fmt(Math.abs(dz), 3), unit: 'mm', hero: true, sub: dz > 0 ? 'raise the nozzle' : dz < 0 ? 'lower the nozzle' : 'no change' },
          { label: 'Level across the bed', value: spread >= 2 ? 'Needs levelling' : spread >= 1 ? 'Slightly uneven' : 'Even', sub: `${got.length} patches rated` }],
        blocks, notes: ['Change in small steps (0.02 mm), print again and compare.', ...notes], save: {}, summary: `First layer: Z ${dz >= 0 ? '+' : ''}${fmt(dz, 3)} mm`,
      };
    },
  },

  /* ================================================================ Dimensional accuracy */
  {
    id: 'dims', title: 'Dimensional accuracy', icon: 'cube', tone: 'alt2', cat: 'Accuracy', check: 'dims', time: '20–60 min', kind: 'print',
    summary: 'Right size, right shape: shrinkage, scaling and skew.',
    learn: {
      what: 'Prints a test object and compares its real size with what it should be. The XYZ cube is quick; the multi-size frame also separates scaling from over-extrusion and measures skew (frame not square).',
      why: 'If parts come out too big, too small or not square, things won\'t fit together. Some shrinkage is normal and depends on the filament; skew comes from a frame that isn\'t perfectly square.',
      good: 'Within ±0.1 mm on a 20 mm cube. On the frame, the two diagonals match within about 0.2 mm.',
      how: 'Use digital calipers. Measure each axis two or three times and use the average. Measure above the first few layers (the bottom often flares out — "elephant\'s foot").',
    },
    diagram: (c) => c.method === 'accuracy-frame' ? DG.frame() : DG.cube(),
    methods: [
      { id: 'cube', label: 'XYZ cube', desc: 'Quick check of all three axes.' },
      { id: 'accuracy-frame', label: 'Multi-size frame + skew', desc: 'Best for scaling, hole fit and squareness.' },
    ],
    params: (c) => c.method === 'accuracy-frame' ? [
      { k: 'step', label: 'Size step', unit: 'mm', def: c.small ? 10 : 20, step: 5 },
      { k: 'max', label: 'Largest square', unit: 'mm', def: c.small ? 40 : 100, step: 10 },
      { k: 'wall', label: 'Wall width', unit: 'mm', def: 3, step: 0.5 },
      { k: 'height', label: 'Height', unit: 'mm', def: 3, step: 0.5 },
    ] : [
      { k: 'size', label: 'Cube size', unit: 'mm', def: c.small ? 15 : 20, step: 1 },
      { k: 'infill', label: 'Infill', unit: '%', def: 20, step: 5 },
      { k: 'perims', label: 'Walls', def: 3, step: 1 },
    ],
    measure: (c) => c.method === 'accuracy-frame' ? ({
      intro: 'Outside size of each square in X (left–right) and Y (front–back), then the big square\'s diagonals.',
      table: (c.info.sizes || []).map((s) => ({ nominal: s, kx: 'x' + s, ky: 'y' + s })),
      fields: [
        { k: 'ac', label: 'Diagonal A–C', unit: 'mm', step: 0.01 }, { k: 'bd', label: 'Diagonal B–D', unit: 'mm', step: 0.01 },
        { k: 'ad', label: 'Side A–D (front edge)', unit: 'mm', step: 0.01 },
      ],
    }) : ({
      intro: `Measure the cube. It should be ${c.params.size} mm on every side.`,
      readCurrent: true,
      fields: [
        { k: 'x', label: 'X (face with X)', unit: 'mm', step: 0.01 }, { k: 'y', label: 'Y (face with Y)', unit: 'mm', step: 0.01 }, { k: 'z', label: 'Z (height)', unit: 'mm', step: 0.01 },
        { k: 'sx', label: c.fw === 'klipper' ? 'X rotation distance (optional)' : 'X steps/mm (optional)', def: c.printer.values?.axisX },
        { k: 'sy', label: c.fw === 'klipper' ? 'Y rotation distance (optional)' : 'Y steps/mm (optional)', def: c.printer.values?.axisY },
        { k: 'sz', label: c.fw === 'klipper' ? 'Z rotation distance (optional)' : 'Z steps/mm (optional)', def: c.printer.values?.axisZ },
      ],
    }),
    compute(c, m) {
      const blocks = [], notes = [], save = { profile: {}, printer: {} }, stats = [];
      if (c.method === 'accuracy-frame') {
        const sizes = c.info.sizes || [];
        const fx = linreg(sizes, sizes.map((s) => m['x' + s])), fy = linreg(sizes, sizes.map((s) => m['y' + s]));
        if (!fx && !fy && !isFinite(m.ac)) return { error: 'Measure at least two squares (X and/or Y), or the diagonals.' };
        let comp = [];
        for (const [ax, f] of [['X', fx], ['Y', fy]]) {
          if (!f) continue;
          const scale = 100 / f.a;
          comp.push(-f.b / 2);
          stats.push({ label: `${ax} scale`, value: fmt(scale, 2), unit: '%', sub: `size error ${fmt((f.a - 1) * 100, 2)} %` });
          save.profile['scale' + ax] = round(scale, 2);
        }
        if (comp.length) {
          const xy = round(avg(comp), 3);
          stats.unshift({ label: 'XY size compensation', value: (xy > 0 ? '+' : '') + fmt(xy, 3), unit: 'mm', hero: true, sub: xy < 0 ? 'outlines print too big' : 'outlines print too small' });
          save.profile.xyComp = xy;
          blocks.push(slicerBlock([
            ['PrusaSlicer → XY size compensation', fmt(xy, 3) + ' mm'],
            ['OrcaSlicer / Bambu → X-Y contour compensation', fmt(xy, 3) + ' mm'],
            ['Cura → Horizontal Expansion', fmt(xy, 3) + ' mm'],
            ...(save.profile.scaleX ? [['Scale X (per filament shrinkage)', fmt(save.profile.scaleX, 2) + ' %']] : []),
            ...(save.profile.scaleY ? [['Scale Y', fmt(save.profile.scaleY, 2) + ' %']] : []),
          ]));
          notes.push('Scale fixes size errors that grow with size (shrinkage, belts). Size compensation fixes a constant error (too much plastic on outlines).');
        }
        if (isFinite(m.ac) && isFinite(m.bd)) {
          const ad = isFinite(m.ad) ? m.ad : sizes[sizes.length - 1];
          const f = skewFactor(m.ac, m.bd, ad);
          const deg = Math.atan(f) * 180 / Math.PI;
          stats.push({ label: 'Skew', value: fmt(deg, 3), unit: '°', sub: `diagonals differ by ${fmt(Math.abs(m.ac - m.bd), 2)} mm` });
          save.printer.skew = { ac: m.ac, bd: m.bd, ad, factor: f };
          if (Math.abs(m.ac - m.bd) < 0.2) notes.push('Diagonals match within 0.2 mm — your frame is square enough; skew correction is optional.');
          else { notes.push('Fixing the frame mechanically (squaring the gantry) is better than software skew correction, when possible.'); blocks.push(...FWC.skew(c.fw, m.ac, m.bd, ad, f)); }
        }
        return { stats, blocks, notes, save, summary: stats.map((s) => `${s.label} ${s.value}${s.unit || ''}`).join(', ') };
      }
      const n = c.params.size;
      const axes = { X: m.x, Y: m.y, Z: m.z };
      const newSteps = {};
      let any = false;
      for (const [ax, v] of Object.entries(axes)) {
        if (!isFinite(v)) continue;
        any = true;
        const err = (v - n) / n * 100;
        const scale = n / v * 100;
        stats.push({ label: `${ax}`, value: fmt(v, 2), unit: 'mm', sub: `${err >= 0 ? '+' : ''}${fmt(err, 2)} % · scale ${fmt(scale, 2)} %`, hero: false });
        save.profile['scale' + ax] = round(scale, 2);
        const cur = m['s' + ax.toLowerCase()];
        if (isFinite(cur) && Math.abs(err) > 0.5) newSteps[ax] = c.fw === 'klipper' ? cur * v / n : cur * n / v;
      }
      if (!any) return { error: 'Enter at least one measurement.' };
      const worst = Math.max(...Object.values(axes).filter(isFinite).map((v) => Math.abs(v - n)));
      stats.unshift({ label: 'Largest error', value: fmt(worst, 2), unit: 'mm', hero: true, sub: worst <= 0.1 ? 'excellent' : worst <= 0.2 ? 'good' : 'worth correcting' });
      blocks.push(slicerBlock([
        ['Scale per axis (slicer, per filament)', ['X', 'Y', 'Z'].filter((a) => save.profile['scale' + a]).map((a) => `${a} ${fmt(save.profile['scale' + a], 2)} %`).join('  ')],
      ]));
      if (Object.keys(newSteps).length) {
        notes.push('Only change steps/rotation distance if the error is the same on bigger prints too (use the multi-size frame to check). For small errors, slicer scaling or size compensation is safer.');
        blocks.push(...FWC.steps(c.fw, newSteps));
      }
      if (worst <= 0.1) notes.push('Within 0.1 mm — great. Small errors on a single cube are mostly over-extrusion or elephant\'s foot, not scaling.');
      return { stats, blocks, notes, save, summary: `Cube ${['X', 'Y', 'Z'].filter((a) => isFinite(axes[a])).map((a) => `${a} ${fmt(axes[a], 2)}`).join(' ')}` };
    },
  },

  /* ================================================================ Overhang */
  {
    id: 'overhang', gen: 'overhang', title: 'Overhangs', icon: 'angle', tone: 'alt', cat: 'Quality', time: '20–30 min', kind: 'print',
    summary: 'How steep you can print without supports.',
    learn: {
      what: 'Prints fins that lean further and further out (angles measured from vertical), each labelled on the base.',
      why: 'Knowing your printer\'s overhang limit tells you when you really need supports — saving time, plastic and clean-up.',
      good: 'Fins with a smooth underside and no curling or drooping edges.',
      how: 'Look at the underside of each fin. Note the steepest angle that still looks clean.',
    },
    diagram: () => DG.overhang(),
    params: () => [
      { k: 'start', label: 'First angle', unit: '°', def: 30, step: 5 },
      { k: 'end', label: 'Last angle', unit: '°', def: 70, step: 5 },
      { k: 'step', label: 'Step', unit: '°', def: 10, step: 5 },
      { k: 'reach', label: 'Max reach', unit: 'mm', def: 10, step: 1 },
    ],
    measure: (c) => ({ intro: 'Which is the steepest clean fin?', fields: [{ k: 'best', label: 'Steepest clean angle', type: 'select', options: optList(c.info.angles, 0, '°') }] }),
    compute(c, m) {
      if (!m.best) return { error: 'Choose an angle.' };
      const a = num(m.best);
      return {
        stats: [{ label: 'Clean overhang limit', value: fmt(a, 0), unit: '°', hero: true, sub: 'from vertical' }],
        blocks: [slicerBlock([
          ['Cura → Support Overhang Angle', `${a}°`],
          ['PrusaSlicer → Support threshold angle', `${90 - a}°`],
          ['OrcaSlicer / Bambu → Threshold angle', `${90 - a}°`],
        ], 'Support settings (start here, adjust to taste)')],
        notes: ['More part cooling, slightly lower temperature and slower outer walls all improve overhangs.'],
        save: { profile: { maxOverhang: a } }, summary: `Overhangs clean to ${a}°`,
      };
    },
  },

  /* ================================================================ Bridging */
  {
    id: 'bridge', gen: 'bridge', title: 'Bridging', icon: 'bridge', tone: 'alt', cat: 'Quality', time: '15–25 min', kind: 'print',
    summary: 'How far you can span across thin air.',
    learn: {
      what: 'Prints pairs of pillars at different distances apart, with a bridge printed across the top of each pair.',
      why: 'Shows how long a gap you can span without supports, and whether your cooling and bridge settings are good.',
      good: 'Bridges that are straight and tight, without sagging strands underneath.',
      how: 'Look from the side and underneath. Note the longest gap that still bridges cleanly.',
    },
    diagram: () => DG.bridge(),
    params: (c) => [
      { k: 'start', label: 'Shortest gap', unit: 'mm', def: c.small ? 5 : 10, step: 5 },
      { k: 'end', label: 'Longest gap', unit: 'mm', def: c.small ? 15 : 50, step: 5 },
      { k: 'step', label: 'Step', unit: 'mm', def: c.small ? 5 : 10, step: 5 },
      { k: 'bridgeSpeed', label: 'Bridge speed', unit: 'mm/s', def: 20, step: 5 },
    ],
    measure: (c) => ({ intro: 'Longest clean bridge?', fields: [{ k: 'best', label: 'Longest clean gap', type: 'select', options: optList(c.info.gaps, 0, ' mm') }] }),
    compute(c, m) {
      if (!m.best) return { error: 'Choose a gap.' };
      const g = num(m.best);
      return {
        stats: [{ label: 'Clean bridge length', value: fmt(g, 0), unit: 'mm', hero: true }],
        blocks: [slicerBlock([['Bridge fan', '100 %'], ['Bridge speed', `${c.params.bridgeSpeed} mm/s (try slower if sagging)`], ['Bridge flow', '90–100 %']], 'Good bridge settings')],
        notes: ['Longer gaps than this need supports, or a design change (a chamfer or a teardrop shape).'],
        save: { profile: { maxBridge: g } }, summary: `Bridges clean to ${g} mm`,
      };
    },
  },

  /* ================================================================ Tolerance */
  {
    id: 'tolerance', gen: 'tolerance', title: 'Tolerance / fit', icon: 'fit', tone: 'alt2', cat: 'Accuracy', time: '30–45 min', kind: 'print',
    summary: 'Find the clearance for press, sliding and loose fits — shared with Makie.',
    learn: {
      what: 'Prints rings with holes slightly bigger than a matching peg (0.0, 0.1, 0.2 … mm bigger), each labelled with its clearance, plus a peg.',
      why: 'Printed holes come out smaller and pegs bigger than designed. Knowing your clearances lets you design parts that snap, slide or press together first time. Tunie saves the results to the suite link file so Makie can use them to suggest hole sizes.',
      good: 'You find a ring where the peg presses in firmly, one where it slides smoothly, and one where it\'s loose.',
      how: 'Push the top end of the peg into each ring from the top side (the bottom often has elephant\'s foot). Record how each one fits.',
    },
    diagram: () => DG.tolerance(),
    params: (c) => [
      { k: 'diameter', label: 'Peg diameter', unit: 'mm', def: c.small ? 6 : 10, step: 1 },
      { k: 'start', label: 'Smallest clearance', unit: 'mm', def: 0, step: 0.05 },
      { k: 'end', label: 'Largest clearance', unit: 'mm', def: 0.6, step: 0.05 },
      { k: 'step', label: 'Step', unit: 'mm', def: 0.1, step: 0.05 },
      { k: 'wall', label: 'Ring wall', unit: 'mm', def: c.small ? 2 : 3, step: 0.5 },
    ],
    measure: (c) => ({
      intro: 'How does the peg fit in each ring?',
      fields: (c.info.clearances || []).map((v) => ({ k: 'c' + v, label: `Clearance ${fmt(v, 2)} mm`, type: 'select', options: [['', '— choose —'], ['none', "Won't go in"], ['press', 'Tight / press fit'], ['slide', 'Slides smoothly'], ['loose', 'Loose / wobbly']] })),
    }),
    compute(c, m) {
      const cl = c.info.clearances || [];
      const first = (k) => cl.find((v) => m['c' + v] === k);
      let press = first('press'), slide = first('slide'), loose = first('loose');
      if (press === undefined && slide === undefined && loose === undefined) return { error: 'Rate at least one ring.' };
      if (press === undefined && slide !== undefined) press = Math.max(0, slide - 0.1);
      const save = { profile: { tolerance: { press: press ?? null, slide: slide ?? null, loose: loose ?? null, diameter: c.params.diameter, date: new Date().toISOString() } } };
      const st = (l, v, sub) => ({ label: l, value: v === undefined || v === null ? '—' : fmt(v, 2), unit: v === undefined || v === null ? '' : 'mm', sub });
      return {
        stats: [st('Sliding fit', slide, 'hole = shaft + this'), st('Press fit', press, 'tight, stays put'), st('Loose fit', loose, 'free to rotate / wobble')].map((s, i) => i === 0 ? { ...s, hero: true } : s),
        blocks: [{ title: 'Designing parts with these clearances', text: `; Diameter clearances for ${c.filament.brand} ${c.filament.type} on ${c.printer.name}\n; hole diameter = shaft diameter + clearance\n; e.g. a 5 mm shaft:\n; press fit hole : ${press != null ? fmt(5 + press, 2) : '—'} mm\n; sliding hole   : ${slide != null ? fmt(5 + slide, 2) : '—'} mm\n; loose hole     : ${loose != null ? fmt(5 + loose, 2) : '—'} mm` }],
        notes: ['Clearance changes a little with hole size and orientation — re-test if you design very small or very large holes.', 'Saved to the suite link file, ready for Makie to read when you add holes and pegs.'],
        save, summary: `Fit: press ${fmt(press, 2)} / slide ${fmt(slide, 2)} / loose ${fmt(loose, 2)} mm`,
      };
    },
  },

  /* ================================================================ Stringing */
  {
    id: 'stringing', gen: 'stringing', title: 'Stringing', icon: 'strings', tone: 'alt', cat: 'Quality', time: '15–25 min', kind: 'print',
    summary: 'Check your temperature and retraction together.',
    learn: {
      what: 'Prints four thin spires far apart, so the nozzle travels a lot between them using your current temperature and retraction settings.',
      why: 'A final check after the temperature and retraction towers — or a quick test of a new spool.',
      good: 'No hairs or strings between the spires, and the tips are clean.',
      how: 'Look between the spires against a dark background. A few wispy hairs that brush off are fine; strings that hold their shape are not.',
    },
    diagram: () => DG.stringing(),
    params: (c) => [
      { k: 'distance', label: 'Distance between spires', unit: 'mm', def: c.small ? 20 : 30, step: 5 },
      { k: 'height', label: 'Height', unit: 'mm', def: c.small ? 15 : 25, step: 5 },
    ],
    measure: () => ({
      intro: 'How much stringing do you see?',
      fields: [{ k: 'rating', label: 'Stringing', type: 'select', options: [['', '— choose —'], ['0', 'None — clean'], ['1', 'A few wispy hairs'], ['2', 'Some strings'], ['3', 'Lots of strings / blobs']] },
        { k: 'wet', label: 'Filament crackles/pops or steams while printing', type: 'check' }],
    }),
    compute(c, m) {
      if (m.rating === '' || m.rating === undefined) return { error: 'Choose how much stringing you see.' };
      const r = num(m.rating);
      const tips = [];
      if (m.wet) tips.push('Crackling or steam means the filament is wet. Dry it (PLA ~45–50 °C, PETG ~65 °C, nylon ~70–80 °C for 4–8 h) — no setting fixes wet filament.');
      if (r >= 2) tips.push(`Lower the temperature by 5 °C (you\'re at ${c.settings.hotend} °C).`, `Increase retraction a little (you\'re at ${fmt(c.settings.retractLen, 2)} mm) — or run the retraction tower.`, 'Raise travel speed so the nozzle has less time to ooze.', 'Turn on "avoid crossing perimeters" / "wipe" in your slicer.');
      if (r === 1) tips.push('A few hairs are normal for PETG and TPU; a quick pass with a heat gun or lighter removes them.');
      if (r === 0) tips.push('Your temperature and retraction are well matched for this filament.');
      return {
        stats: [{ label: 'Stringing', value: ['None', 'Minimal', 'Some', 'Heavy'][r], hero: true, sub: `${c.settings.hotend} °C · ${fmt(c.settings.retractLen, 2)} mm @ ${fmt(c.settings.retractSpeed, 0)} mm/s` }],
        blocks: [], notes: tips, save: {}, summary: `Stringing: ${['none', 'minimal', 'some', 'heavy'][r]}`,
      };
    },
  },
];

const TEST_BY_ID = Object.fromEntries(TESTS.map((t) => [t.id, t]));
