/* Tunie — wizard.js: the guided "print this → measure this → type the number → new setting" flow.
   Part of the nrmluvdev maker suite. Created by nrmluvdev. */
'use strict';

const SETTINGS_FIELDS = [
  { k: 'hotend', label: 'Nozzle temp', unit: '°C', step: 5 },
  { k: 'bed', label: 'Bed temp', unit: '°C', step: 5 },
  { k: 'fan', label: 'Part fan', unit: '%', step: 10 },
  { k: 'layerH', label: 'Layer height', unit: 'mm', step: 0.02 },
  { k: 'firstLayerH', label: 'First layer height', unit: 'mm', step: 0.02 },
  { k: 'lineW', label: 'Line width', unit: 'mm', step: 0.01 },
  { k: 'speed', label: 'Print speed', unit: 'mm/s', step: 5 },
  { k: 'firstSpeed', label: 'First layer speed', unit: 'mm/s', step: 5 },
  { k: 'travelSpeed', label: 'Travel speed', unit: 'mm/s', step: 10 },
  { k: 'retractLen', label: 'Retraction', unit: 'mm', step: 0.1 },
  { k: 'retractSpeed', label: 'Retraction speed', unit: 'mm/s', step: 5 },
  { k: 'zHop', label: 'Z hop', unit: 'mm', step: 0.1 },
  { k: 'flow', label: 'Flow', unit: '%', step: 1 },
  { k: 'accel', label: 'Acceleration', unit: 'mm/s²', step: 100, help: '0 = leave as is' },
  { k: 'mesh', label: 'Bed mesh at start', type: 'select', options: [['none', 'None'], ['load', 'Load saved mesh'], ['probe', 'Probe a new mesh']] },
];

function defaultSettings(p, f, v) {
  const tpu = f.type === 'TPU';
  return {
    hotend: v.temp ?? f.temp, bed: f.bed, fan: f.fan ?? 100,
    layerH: round(p.nozzle * 0.5, 2), firstLayerH: round(p.nozzle * 0.6, 2), lineW: round(p.nozzle * 1.125, 3),
    speed: Math.min(tpu ? 25 : 50, p.maxSpeed), firstSpeed: tpu ? 15 : 20, travelSpeed: Math.min(150, p.maxSpeed),
    retractLen: v.retractLen ?? (tpu ? 0.5 : p.drive === 'bowden' ? 5 : 0.8), retractSpeed: v.retractSpeed ?? (p.drive === 'bowden' ? 40 : 35),
    zHop: 0, flow: v.flow ?? 100, accel: 0, mesh: p.mesh || 'none',
  };
}

const WIZ = { gen: null, preview: null };

function sessionFor(t, p, f) {
  DB.data.ui.sess = DB.data.ui.sess || {};
  const key = `${t.id}|${p.id}|${f.id}`;
  if (!DB.data.ui.sess[key]) DB.data.ui.sess[key] = { step: 0, method: t.methods ? t.methods[0].id : t.gen || t.id, params: {}, settings: null, info: {}, measure: {}, generated: false, saved: false };
  return DB.data.ui.sess[key];
}

function buildCtx(t, s) {
  const printer = DB.printer(), filament = DB.filament(), prof = DB.profile();
  printer.values = printer.values || {};
  const settings = Object.assign(defaultSettings(printer, filament, prof.values), s.settings || {});
  const base = { printer, filament, profile: prof.values, settings, method: s.method, fw: printer.firmware, info: s.info || {}, small: Math.min(printer.bedX, printer.bedY) < 120 };
  const fields = t.params ? t.params(Object.assign({}, base, { params: {} })) : [];
  const params = {};
  fields.forEach((f) => { params[f.k] = s.params[f.k] !== undefined && !(typeof s.params[f.k] === 'number' && isNaN(s.params[f.k])) ? s.params[f.k] : f.def; });
  return Object.assign(base, { params, paramFields: fields });
}

function renderTest(id) {
  const t = TEST_BY_ID[id];
  if (!t) return go('#/tests');
  const p = DB.printer(), f = DB.filament();
  const s = sessionFor(t, p, f);
  const steps = [
    ['Learn', 'What & why'], [t.kind === 'commands' ? 'Run' : 'Print', t.kind === 'commands' ? 'Send the commands' : 'Make the G-code'],
    ['Measure', 'Type what you see'], ['Result', 'Your new setting'],
  ];
  const bambu = p.firmware === 'bambu' && t.check && BAMBU_AUTO[t.check];
  const done = t.check && DB.checklist()[t.check]?.done;
  const html = `<div class="page-head"><div class="row" style="gap:14px"><div class="tile" style="min-height:0;padding:0;border:0;background:none;box-shadow:none;cursor:default"><div class="ico ${t.tone}" style="width:52px;height:52px;border-radius:15px">${icon(t.icon, 26)}</div></div>
      <div><h1>${esc(t.title)}</h1><div class="sub">${esc(t.summary)}</div></div></div>
      <div class="actions"><span class="chip">${icon('history', 13)}${esc(t.time)}</span><span class="chip acc">${esc(t.cat)}</span>${done ? `<span class="chip ok">${icon('check', 13)}Done for this printer</span>` : ''}
      <button class="btn sm ghost" id="wiz-reset" title="Start this test again">${icon('refresh', 15)}Restart</button></div></div>
    ${bambu ? `<div class="callout ${bambu.auto === true ? 'good' : ''} mb">${icon('bambu')}<div><p><b>Bambu Lab:</b> ${esc(bambu.text)}</p></div></div>` : ''}
    <div class="stepper">${steps.map(([a, b], i) => `<button data-step="${i}" class="${i === s.step ? 'on' : ''} ${i < s.step ? 'done' : ''}"><span class="n">${i < s.step ? '✓' : i + 1}</span><span>${a}<small>${b}</small></span></button>`).join('')}</div>
    <div id="wiz-body" class="wiz-body"></div>`;
  view(html, t.title);
  $$('.stepper [data-step]').forEach((b) => b.onclick = () => { s.step = +b.dataset.step; DB.save(); renderTest(id); });
  $('#wiz-reset').onclick = async () => {
    if (!(await confirmBox('Restart this test?', 'Your measurements for this test will be cleared. Saved results stay in History.', 'Restart'))) return;
    delete DB.data.ui.sess[`${t.id}|${p.id}|${f.id}`]; DB.save(); WIZ.gen = null; renderTest(id);
  };
  const body = $('#wiz-body');
  [stepLearn, stepPrint, stepMeasure, stepResult][s.step](t, s, body);
}

function wizNav(body, s, t, { back = true, next = 'Next', onNext, nextDisabled = false } = {}) {
  const nav = document.createElement('div');
  nav.className = 'wiz-nav';
  nav.innerHTML = `${back && s.step > 0 ? `<button class="btn ghost" data-b>${icon('arrowL', 16)}Back</button>` : '<span></span>'}
    ${next ? `<button class="btn primary lg" data-n ${nextDisabled ? 'disabled' : ''}>${esc(next)}${icon('arrowR', 16)}</button>` : ''}`;
  body.appendChild(nav);
  const b = $('[data-b]', nav); if (b) b.onclick = () => { s.step--; DB.save(); renderTest(t.id); };
  const n = $('[data-n]', nav); if (n) n.onclick = async () => { if (onNext && (await onNext()) === false) return; s.step++; DB.save(); renderTest(t.id); };
  return nav;
}

/* ---------------------------------------------------------------- step 1: learn */
function stepLearn(t, s, body) {
  const c = buildCtx(t, s);
  const L = t.learn;
  body.innerHTML = `<div class="grid" style="grid-template-columns:minmax(0,1.15fr) minmax(0,1fr);gap:18px">
    <div class="explain">
      ${[['What it checks', L.what, 'flask'], ['Why it matters', L.why, 'star'], ['What good looks like', L.good, 'check'], ['How to measure', L.how, 'ruler']].map(([h, txt, ic]) => `<div class="card"><div class="row" style="margin-bottom:8px"><span class="ico-s">${icon(ic, 16)}</span><h3 style="margin:0">${h}</h3></div><div class="big">${esc(txt)}</div></div>`).join('')}
    </div>
    <div class="col"><div class="diagram">${t.diagram(c)}<div class="cap">${t.kind === 'commands' ? 'What happens' : 'What to look at / where to measure'}</div></div>
    ${t.methods ? `<div class="card flat"><h3>Choose a method</h3><div class="col">${t.methods.map((m) => `<label class="list-item ${m.id === s.method ? 'sel' : ''}" style="cursor:pointer"><input type="radio" name="meth" value="${m.id}" ${m.id === s.method ? 'checked' : ''} style="accent-color:var(--accent)"><div class="grow"><b>${esc(m.label)}</b><span>${esc(m.desc)}</span></div></label>`).join('')}</div></div>` : ''}
    </div></div>`;
  $$('input[name=meth]', body).forEach((r) => r.onchange = () => { s.method = r.value; s.params = {}; s.info = {}; s.measure = {}; s.generated = false; WIZ.gen = null; DB.save(); renderTest(t.id); });
  wizNav(body, s, t, { next: t.kind === 'commands' ? 'Next: run it' : 'Next: make the G-code' });
}

/* ---------------------------------------------------------------- step 2: print / run */
function stepPrint(t, s, body) {
  const c = buildCtx(t, s);
  const p = c.printer;
  const settingsShown = t.kind !== 'commands';
  body.innerHTML = `<div class="grid" style="grid-template-columns:minmax(0,1fr) minmax(0,1.25fr);gap:18px;align-items:start">
    <div class="col">
      ${t.methods ? `<div class="seg" id="meth">${t.methods.map((m) => `<button data-m="${m.id}" class="${m.id === s.method ? 'on' : ''}">${esc(m.label)}</button>`).join('')}</div>` : ''}
      <div class="card"><h3>${icon('wand', 16)}Test settings</h3>${t.paramsHint ? `<div class="card-sub">${esc(t.paramsHint)}</div>` : ''}
        <div class="fields" id="pf">${c.paramFields.map((f) => fieldHTML(f, c.params[f.k])).join('')}</div></div>
      ${settingsShown ? `<details class="card" id="sdet" ${DB.data.ui.settingsOpen ? 'open' : ''}><summary style="cursor:pointer;font-weight:650;display:flex;align-items:center;gap:8px">${icon('gear', 16)}Print settings <span class="muted small" style="font-weight:500">— from ${esc(p.name)} + ${esc(DB.filament().brand)} ${esc(DB.filament().type)}</span></summary>
        <div class="fields mt" id="sf">${SETTINGS_FIELDS.map((f) => fieldHTML(f, c.settings[f.k])).join('')}</div>
        <div class="row mt"><button class="btn sm ghost" id="sreset">${icon('refresh', 14)}Reset to profile values</button></div></details>` : ''}
      ${t.kind !== 'commands' ? `<button class="btn primary lg" id="gen">${icon('wand', 18)}Generate G-code</button>` : ''}
    </div>
    <div class="col" id="out"></div></div>`;
  if (t.methods) $$('#meth button', body).forEach((b) => b.onclick = () => { s.method = b.dataset.m; s.params = {}; s.info = {}; s.generated = false; WIZ.gen = null; DB.save(); renderTest(t.id); });
  const save = () => {
    s.params = Object.assign(s.params, readFields($('#pf', body)));
    if ($('#sf', body)) s.settings = readFields($('#sf', body));
    DB.save();
  };
  $$('#pf [data-k], #sf [data-k]', body).forEach((el) => el.addEventListener('change', () => { save(); if (t.kind === 'commands') renderOut(); }));
  const det = $('#sdet', body); if (det) det.addEventListener('toggle', () => { DB.data.ui.settingsOpen = det.open; DB.save(); });
  const sr = $('#sreset', body); if (sr) sr.onclick = () => { s.settings = null; DB.save(); renderTest(t.id); };
  const out = $('#out', body);
  const renderOut = () => {
    const cc = buildCtx(t, s);
    if (t.kind === 'commands') {
      out.innerHTML = `<div class="card">${t.run(cc)}<div class="row mt"><button class="btn" id="gen">${icon('save', 16)}Save as G-code file</button></div></div>`;
      $('#gen', out).onclick = () => generate(t, s, out, true);
      if (t.onPaste) {
        const onReply = (e) => {
          const m = Object.assign({}, s.measure || {});
          if (t.onPaste(cc, e.detail.join('\n'), m)) { s.measure = m; DB.save(); toast('Got the results from your printer — click Next: measure'); }
        };
        document.addEventListener('printer-reply', onReply, { once: true });
      }
    } else if (WIZ.gen && WIZ.gen.key === genKey(t, s)) showGen(t, s, out, WIZ.gen.res);
    else out.innerHTML = `<div class="card empty" style="padding:70px 20px">${icon('cube', 40)}<h3>Preview appears here</h3><p class="muted">Adjust the settings if you like, then click <b>Generate G-code</b>.</p></div>`;
  };
  renderOut();
  const gb = $('#gen', body);
  if (gb && t.kind !== 'commands') gb.onclick = () => { save(); generate(t, s, out); };
  wizNav(body, s, t, { next: 'Next: measure', onNext: () => { save(); if (t.kind !== 'commands' && !s.generated) { toast('Generate and print the test first (or skip ahead from the steps bar).', 'info'); return false; } } });
}

function genKey(t, s) { return JSON.stringify([t.id, s.method, s.params, s.settings, DB.printer().id, DB.filament().id]); }

async function generate(t, s, out, fileOnly = false) {
  const c = buildCtx(t, s);
  const settings = Object.assign({}, c.settings, { density: DB.filament().density || 1.24, filamentName: `${DB.filament().brand} ${DB.filament().type} ${DB.filament().colourName || ''}`.trim() });
  if (isFinite(c.params.hotend)) settings.hotend = c.params.hotend;
  const params = {}, str = {};
  c.paramFields.forEach((f) => {
    const v = c.params[f.k];
    if (f.type === 'select') { str[f.k] = String(v); if (isFinite(num(v))) params[f.k] = num(v); }
    else if (f.type === 'check') params[f.k] = v ? 1 : 0;
    else if (isFinite(v)) params[f.k] = v;
  });
  if (c.params.fan !== undefined && t.id === 'pid') settings.fan = c.params.fan;
  const btn = $('#gen');
  const old = btn && btn.innerHTML;
  if (btn) { btn.disabled = true; btn.innerHTML = '<span class="spin"></span>Generating…'; }
  try {
    const res = await api.post('/api/generate', { test: t.methods ? s.method : t.gen, printer: printerForGen(c.printer), settings, params, str });
    s.info = res.info || {};
    s.generated = true;
    s.measure = s.measure || {};
    DB.save();
    WIZ.gen = { key: genKey(t, s), res };
    if (fileOnly) return saveGcode(res);
    showGen(t, s, out, res);
  } catch (e) {
    out.innerHTML = `<div class="callout bad">${icon('alert')}<div><p><b>Couldn't make this test.</b> ${esc(e.message)}</p></div></div>`;
  } finally { if (btn) { btn.disabled = false; btn.innerHTML = old; } }
}

function printerForGen(p) {
  const o = {};
  ['name', 'firmware', 'bedShape', 'bedX', 'bedY', 'bedZ', 'originCenter', 'nozzle', 'filamentDia', 'maxHotend', 'maxBed', 'maxSpeed', 'maxAccel', 'drive', 'startGcode', 'endGcode', 'hasInputShaper'].forEach((k) => o[k] = p[k]);
  return o;
}

function showGen(t, s, out, res) {
  const mins = Math.round(res.timeSec / 60);
  const time = mins >= 60 ? `${Math.floor(mins / 60)} h ${mins % 60} min` : `${mins} min`;
  out.innerHTML = `
    <div class="grid g4">
      <div class="card stat"><div class="lbl">Print time</div><div class="val" style="font-size:22px">~${time}</div></div>
      <div class="card stat"><div class="lbl">Filament</div><div class="val" style="font-size:22px">${fmt(res.grams, 1)}<small>g</small></div><div class="sub">${fmt(res.filamentMM / 1000, 2)} m</div></div>
      <div class="card stat"><div class="lbl">Height</div><div class="val" style="font-size:22px">${fmt(res.height, 1)}<small>mm</small></div></div>
      <div class="card stat"><div class="lbl">Lines</div><div class="val" style="font-size:22px">${(res.lines / 1000).toFixed(1)}<small>k</small></div></div>
    </div>
    ${(res.warnings || []).map((w) => `<div class="callout warn">${icon('alert')}<div><p>${esc(w)}</p></div></div>`).join('')}
    <div id="pv"></div>
    <div class="card"><h3>${icon('send', 16)}Get it to your printer</h3>
      <div class="row">
        <button class="btn primary" id="g-save">${icon('save', 16)}Save .gcode</button>
        <button class="btn" id="g-sd">${icon('sd', 16)}Save to SD card</button>
        <button class="btn" id="g-send" ${CONN.connected ? '' : 'disabled title="Connect to your printer on the Connection page"'}>${icon('send', 16)}${CONN.kind.startsWith('serial') ? 'Print over USB' : 'Upload & print'}</button>
        <button class="btn ghost" id="g-dl">${icon('download', 16)}Download</button>
      </div>
      <p class="muted small mt-s" style="margin:10px 0 0">File name: <span class="mono">${esc(res.fileName)}</span>. Then print it, come back and click <b>Next: measure</b>.</p>
    </div>`;
  if (WIZ.preview) WIZ.preview.destroy();
  WIZ.preview = new Preview($('#pv', out), DB.printer());
  WIZ.preview.load(res.gcode);
  $('#g-save', out).onclick = () => saveGcode(res);
  $('#g-sd', out).onclick = () => saveToSD(res);
  $('#g-dl', out).onclick = () => downloadFile(res.fileName, res.gcode);
  $('#g-send', out).onclick = () => sendPrint(res);
}

async function saveGcode(res) {
  try {
    const r = await api.post('/api/save-file', { name: res.fileName, content: res.gcode });
    toast('Saved to ' + r.path);
    showSavedPath(r.path);
  } catch (e) { toast(e.message, 'err'); }
}
function showSavedPath(path) {
  modal({ title: 'Saved', body: `<p class="muted">Your file is here:</p><div class="code"><pre>${esc(path)}</pre></div>`, actions: [{ label: 'Show in folder', icon: 'folder', value: async () => { await api.post('/api/open-folder', { path }); return true; } }, { label: 'Done', kind: 'primary' }] });
}
async function saveToSD(res) {
  const pick = async (bg) => {
    const list = $('.drives', bg);
    list.innerHTML = '<span class="spin"></span>';
    let d = [];
    try { d = await api.get('/api/drives'); } catch { }
    list.innerHTML = d && d.length ? d.map((x) => `<button class="list-item" data-p="${esc(x.path)}" style="width:100%;cursor:pointer;color:inherit;text-align:left">${icon('sd', 20)}<div class="grow"><b>${esc(x.label || 'Removable drive')}</b><span>${esc(x.path)}</span></div>${icon('arrowR', 16)}</button>`).join('')
      : `<div class="empty" style="padding:24px">${icon('sd', 30)}<p>No SD card or USB stick found. Insert it, then click Refresh.</p></div>`;
    $$('[data-p]', list).forEach((b) => b.onclick = async () => {
      try { const r = await api.post('/api/save-file', { name: res.fileName.replace(/^Tunie_/, ''), content: res.gcode, dir: b.dataset.p }); toast('Saved to ' + r.path); bg.remove(); }
      catch (e) { toast(e.message, 'err'); }
    });
  };
  modal({ title: 'Save to SD card', body: '<p class="muted">Choose the card. Remember to eject it safely before taking it out.</p><div class="col drives"></div>', actions: [{ label: 'Refresh', icon: 'refresh', value: async (bg) => { await pick(bg); return false; } }, { label: 'Close', kind: 'ghost' }], onOpen: pick });
}
function downloadFile(name, text, mime = 'text/plain') {
  const a = document.createElement('a');
  a.href = URL.createObjectURL(new Blob([text], { type: mime }));
  a.download = name; a.click();
  setTimeout(() => URL.revokeObjectURL(a.href), 2000);
}
async function sendPrint(res) {
  if (CONN.kind.startsWith('serial')) {
    if (!(await confirmBox('Print over USB?', 'Tunie will stream the file line by line. Keep this window open and the computer awake until it finishes. Is the bed clear?', 'Start printing'))) return;
    try { await api.post('/api/conn/print', { name: res.fileName, gcode: res.gcode }); toast('Printing — progress shows in the top bar'); pollConn(); } catch (e) { toast(e.message, 'err'); }
    return;
  }
  const v = await modal({ title: 'Send to printer', body: `<p class="muted">Upload <b>${esc(res.fileName)}</b> to ${esc(CONN.kind)}.</p>`, actions: [{ label: 'Upload only', value: () => 'upload' }, { label: 'Upload & print', kind: 'primary', icon: 'play', value: () => 'print' }] });
  if (!v) return;
  try { await api.post('/api/conn/print', { name: res.fileName, gcode: res.gcode, start: v === 'print' }); toast(v === 'print' ? 'Uploaded — printing' : 'Uploaded'); } catch (e) { toast(e.message, 'err'); }
}

/* ---------------------------------------------------------------- step 3: measure */
function stepMeasure(t, s, body) {
  const c = buildCtx(t, s);
  if (!t.measure) return wizNav(body, s, t, {});
  const M = t.measure(c);
  const vals = s.measure || {};
  const fieldsHTML = M.fields.map((f) => fieldHTML(f, vals[f.k] !== undefined && !(typeof vals[f.k] === 'number' && isNaN(vals[f.k])) ? vals[f.k] : f.def)).join('');
  const table = M.table ? `<table class="tbl"><thead><tr><th>Square</th><th>X (left–right)</th><th>Y (front–back)</th></tr></thead><tbody>${M.table.map((r) => `<tr><td><b>${r.nominal} mm</b></td><td>${fieldHTML({ k: r.kx, label: '', unit: 'mm', step: 0.01 }, vals[r.kx]).replace('<label for="f_' + r.kx + '"></label>', '')}</td><td>${fieldHTML({ k: r.ky, label: '', unit: 'mm', step: 0.01 }, vals[r.ky]).replace('<label for="f_' + r.ky + '"></label>', '')}</td></tr>`).join('')}</tbody></table>` : '';
  const klipperAccel = M.klipperAccel ? `<details class="card"><summary style="cursor:pointer;font-weight:650">${icon('zap', 16)} Have an accelerometer? Let Klipper measure it automatically</summary>
      <p class="muted mt">With an ADXL345 (or similar) wired to your printer and set up in printer.cfg, Klipper can measure the resonances directly. This is quicker and more accurate than counting ripples.</p>
      <div class="grid">${codeBlock('1 · Example config (adjust pins to your board) — printer.cfg', '[adxl345]\ncs_pin: <your CS pin>\n\n[resonance_tester]\naccel_chip: adxl345\nprobe_points: ' + `${fmt(c.printer.bedX / 2, 0)}, ${fmt(c.printer.bedY / 2, 0)}, 20`, { lang: 'cfg' })}
      ${codeBlock('2 · Check the sensor works (should show three numbers)', 'ACCELEROMETER_QUERY\nMEASURE_AXES_NOISE', { send: true })}
      ${codeBlock('3 · Measure both axes and pick the best shaper (takes a few minutes, it will buzz loudly)', 'G28\nSHAPER_CALIBRATE', { send: true })}
      ${codeBlock('4 · Save the result', 'SAVE_CONFIG', { send: true })}</div>
      <p class="muted small mt">Tunie reads the "Recommended shaper" lines automatically when you send step 3 over Moonraker — or paste them in the box on the left.</p></details>` : '';
  body.innerHTML = `<div class="grid" style="grid-template-columns:minmax(0,1fr) minmax(0,1fr);gap:18px;align-items:start">
    <div class="col"><div class="card"><h3>${icon('ruler', 16)}Your measurements</h3><div class="card-sub">${esc(M.intro || '')}</div>
      ${table}<div class="fields ${table ? 'mt' : ''}" id="mf">${fieldsHTML}</div>
      ${M.readCurrent ? `<div class="row mt"><button class="btn sm" id="readcur" ${CONN.connected ? '' : 'disabled title="Connect to your printer first"'}>${icon('download', 14)}Read current values from printer</button></div>` : ''}</div>
      ${(t.onPaste) ? `<div class="card"><h3>${icon('copy', 16)}Paste printer output</h3><div class="card-sub">Copy the lines from your printer's console and paste them here — Tunie finds the numbers.</div><textarea class="ta" id="paste" placeholder="e.g.  #define DEFAULT_Kp 21.73 …  or  pid_Kp=22.865 pid_Ki=1.292 pid_Kd=101.178"></textarea><div class="row mt-s"><button class="btn sm" id="parse">${icon('wand', 14)}Find the numbers</button></div></div>` : ''}
    </div>
    <div class="col"><div class="diagram">${t.diagram(c)}<div class="cap">Where to look and measure</div></div>${klipperAccel}</div></div>`;
  const keep = () => { s.measure = Object.assign(s.measure || {}, readFields(body)); DB.save(); };
  $$('[data-k]', body).forEach((el) => el.addEventListener('input', keep));
  const fill = (m) => { Object.assign(s.measure, m); DB.save(); renderTest(t.id); };
  const pb = $('#parse', body);
  if (pb) pb.onclick = () => {
    keep();
    const m = Object.assign({}, s.measure);
    if (t.onPaste(c, $('#paste', body).value, m)) { toast('Found the values'); fill(m); }
    else toast("Couldn't find the numbers in that text", 'err');
  };
  const rc = $('#readcur', body);
  if (rc) rc.onclick = async () => {
    keep();
    rc.disabled = true;
    try {
      const m = Object.assign({}, s.measure);
      if (c.fw === 'klipper' && CONN.kind.startsWith('moonraker')) {
        const r = await api.get('/api/conn/query?what=config');
        const st = r?.result?.status?.configfile?.settings || {};
        if (t.id === 'esteps') m.current = st.extruder?.rotation_distance;
        else { m.sx = st.stepper_x?.rotation_distance; m.sy = st.stepper_y?.rotation_distance; m.sz = st.stepper_z?.rotation_distance; }
      } else {
        const r = await sendLines([FWC.readCurrent(c.fw) || 'M503']);
        const txt = (r.lines || []).join('\n');
        if (t.id === 'esteps') { const e = PARSE.esteps(txt); if (e) m.current = e; }
        else { const a = PARSE.axisSteps(txt); if (a) { m.sx = a.X; m.sy = a.Y; m.sz = a.Z; } }
      }
      toast('Read current values'); fill(m);
    } catch (e) { toast(e.message, 'err'); rc.disabled = false; }
  };
  if (t.onPaste) {
    const onReply = (e) => {
      const m = Object.assign({}, s.measure);
      if (t.onPaste(c, e.detail.join('\n'), m)) { toast('Read the results from your printer'); fill(m); }
    };
    document.addEventListener('printer-reply', onReply, { once: true });
  }
  wizNav(body, s, t, { next: 'Calculate', onNext: () => keep() });
}

/* ---------------------------------------------------------------- step 4: result */
function stepResult(t, s, body) {
  const c = buildCtx(t, s);
  const m = Object.assign({}, s.measure || {});
  let res;
  try { res = t.compute(c, m); } catch (e) { res = { error: 'Something went wrong in the calculation: ' + e.message }; console.error(e); }
  if (res.error) {
    body.innerHTML = `<div class="callout warn">${icon('alert')}<div><p>${esc(res.error)}</p></div></div>`;
    wizNav(body, s, t, { next: null });
    return;
  }
  const stats = res.stats || [];
  body.innerHTML = `<div class="grid ${stats.length >= 3 ? 'g3' : stats.length === 2 ? 'g2' : ''}">${stats.map((st) => `<div class="card stat ${st.hero ? 'hero' : ''}"><div class="lbl">${esc(st.label)}</div><div class="val">${esc(st.value)}${st.unit ? `<small>${esc(st.unit)}</small>` : ''}</div>${st.sub ? `<div class="sub">${esc(st.sub)}</div>` : ''}</div>`).join('')}</div>
    ${(res.notes || []).length ? `<div class="callout mt">${icon('info')}<div>${res.notes.map((n) => `<p>${esc(n)}</p>`).join('')}</div></div>` : ''}
    <div class="grid mt">${(res.blocks || []).map((b) => codeBlock(b.title, b.text, { send: !!b.send && DB.printer().firmware !== 'bambu', lang: b.lang })).join('')}</div>
    <div class="card mt row" style="justify-content:space-between">
      <div><b>${s.saved ? 'Saved to your profile' : 'Save this result'}</b><div class="muted small">${s.saved ? 'It\'s in History and in the suite link file for Makie.' : `Stores the new values for ${esc(c.printer.name)} · ${esc(c.filament.brand)} ${esc(c.filament.type)}${t.check ? ' and ticks it off the checklist' : ''}.`}</div></div>
      <div class="row"><button class="btn ${s.saved ? '' : 'primary'} lg" id="saveres">${icon(s.saved ? 'check' : 'save', 18)}${s.saved ? 'Saved' : 'Save result'}</button></div></div>`;
  $('#saveres', body).onclick = () => {
    saveResult(t, c, res, m);
    s.saved = true; DB.save();
    toast('Saved — ' + (res.summary || t.title));
    renderTest(t.id);
  };
  const nav = wizNav(body, s, t, { next: null });
  const nextItem = nextChecklistItem();
  if (nextItem && nextItem.test !== t.id) {
    const b = document.createElement('button');
    b.className = 'btn';
    b.innerHTML = `Next in checklist: ${esc(nextItem.title)} ${icon('arrowR', 16)}`;
    b.onclick = () => go('#/test/' + nextItem.test);
    nav.appendChild(b);
  }
}

function saveResult(t, c, res, m) {
  const p = DB.printer(), f = DB.filament(), prof = DB.profile();
  p.values = p.values || {};
  const entry = {
    id: uid(), date: Date.now(), test: t.id, title: t.title, method: c.method, printerId: p.id, printerName: p.name,
    filamentId: f.id, filamentName: `${f.brand} ${f.type} ${f.colourName || ''}`.trim(), params: c.params, inputs: m,
    summary: res.summary || '', stats: (res.stats || []).map((s) => ({ label: s.label, value: s.value, unit: s.unit || '' })), save: res.save || {},
  };
  DB.data.history.unshift(entry);
  Object.assign(p.values, (res.save && res.save.printer) || {});
  Object.assign(prof.values, (res.save && res.save.profile) || {});
  prof.updated = Date.now();
  if (t.check) DB.checklist(p.id)[t.check] = { done: true, date: Date.now(), value: res.summary || '' };
  DB.save();
}

function nextChecklistItem() {
  const cl = DB.checklist();
  const p = DB.printer();
  return CHECKLIST.find((it) => !cl[it.key]?.done && !(p.firmware === 'bambu' && BAMBU_AUTO[it.key]?.auto === true));
}
