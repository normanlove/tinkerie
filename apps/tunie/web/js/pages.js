/* Tunie — pages.js: dashboard, test list, printers, filaments, history,
   exports, connection, Bambu notes and about.
   Part of the nrmluvdev maker suite. Created by nrmluvdev. */
'use strict';

function view(html, crumb) {
  const v = $('#view');
  v.innerHTML = `<div class="inner">${html}</div>`;
  v.scrollTop = 0;
  $('#crumbs').innerHTML = `${icon('wave', 16)}<span>Tunie</span>${crumb ? `<span class="dim">/</span><b>${esc(crumb)}</b>` : ''}`;
}

function ringSVG(pct, size = 120) {
  const r = size / 2 - 9, c = 2 * Math.PI * r;
  return `<svg width="${size}" height="${size}"><defs><linearGradient id="rg" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#7c5cff"/><stop offset="1" stop-color="#22d3ee"/></linearGradient></defs>
    <circle cx="${size / 2}" cy="${size / 2}" r="${r}" stroke="#222a3a" stroke-width="10" fill="none"/>
    <circle class="ring-fg" cx="${size / 2}" cy="${size / 2}" r="${r}" stroke="url(#rg)" stroke-width="10" fill="none" stroke-linecap="round" stroke-dasharray="${c}" stroke-dashoffset="${c}" data-target="${c * (1 - pct)}"/></svg>`;
}
function animateRings() { requestAnimationFrame(() => $$('.ring-fg').forEach((el) => el.style.strokeDashoffset = el.dataset.target)); }

/* ================================================================ home */
function renderHome() {
  const p = DB.printer(), f = DB.filament(), v = DB.profile().values;
  const cl = DB.checklist();
  const bambu = p.firmware === 'bambu';
  const items = CHECKLIST.map((it) => ({ ...it, done: !!cl[it.key]?.done, auto: bambu && BAMBU_AUTO[it.key]?.auto === true, value: cl[it.key]?.value }));
  const count = items.filter((i) => i.done || i.auto).length;
  const next = items.find((i) => !i.done && !i.auto);
  const recent = DB.data.history.filter((h) => h.printerId === p.id).slice(0, 4);
  const vrow = (l, val, unit = '') => `<dt>${l}</dt><dd>${val === undefined || val === null || val === '' ? '<span class="dim">—</span>' : esc(val) + (unit ? ` <span class="muted">${unit}</span>` : '')}</dd>`;
  view(`<div class="hero-banner mb"><svg class="wave" width="520" height="100%" viewBox="0 0 520 200" preserveAspectRatio="none"><path d="M0 120 C 60 20, 100 20, 140 110 S 220 170, 260 110 S 330 70, 360 105 S 430 125, 460 108 S 500 102, 520 104" stroke="url(#hg)" stroke-width="3" fill="none"/><defs><linearGradient id="hg"><stop offset="0" stop-color="#7c5cff"/><stop offset="1" stop-color="#22d3ee"/></linearGradient></defs></svg>
      <div style="position:relative"><span class="chip acc">${esc(FIRMWARES[p.firmware]?.name || p.firmware)}</span>
      <h1 class="mt-s">${count === items.length ? `${esc(p.name)} is fully tuned ✨` : `Let's tune your ${esc(p.name)}`}</h1>
      <p class="mt-s">${next ? `Next up: <b>${esc(next.title)}</b> — ${esc(next.why.toLowerCase())}.` : 'Every step of the checklist is done. Re-run tests whenever you change filament, nozzle or hardware.'}</p>
      <div class="row mt">${next ? `<a class="btn primary lg" href="#/test/${next.test}">${icon('play', 18)}Start ${esc(next.title)}</a>` : ''}<a class="btn lg" href="#/tests">${icon('flask', 18)}All tests</a></div></div></div>
    <div class="grid" style="grid-template-columns:minmax(0,1.35fr) minmax(0,1fr);gap:18px;align-items:start">
      <div class="card pad-lg"><div class="row" style="justify-content:space-between;margin-bottom:14px"><div><h2>Recommended order</h2><div class="muted small">For a new printer, work top to bottom. Progress is saved per printer.</div></div>
        <div class="ring">${ringSVG(count / items.length)}<div class="lbl"><div><b>${count}/${items.length}</b><small>done</small></div></div></div></div>
        <div class="checklist">${items.map((it, i) => `<div class="cl-item ${it.done ? 'done' : ''} ${it.auto ? 'auto' : ''} ${next && it.key === next.key ? 'next' : ''}" data-go="#/test/${it.test}">
          <span class="tick">${it.done ? '✓' : it.auto ? 'A' : i + 1}</span><div class="t"><b>${esc(it.title)}</b><span>${it.auto ? esc(BAMBU_AUTO[it.key].text) : esc(it.why)}</span></div>
          ${it.value ? `<span class="v">${esc(it.value)}</span>` : ''}
          <button class="btn sm ghost" data-tick="${it.key}" title="${it.done ? 'Mark as not done' : 'Mark as done'}">${icon(it.done ? 'x' : 'check', 15)}</button></div>`).join('')}</div></div>
      <div class="col">
        <div class="card"><h3>${icon('spool', 16)}Active profile</h3>
          <div class="row mb"><span class="swatch" style="background:${esc(f.colour)};width:16px;height:16px"></span><b>${esc(f.brand)} ${esc(f.type)}</b><span class="muted">${esc(f.colourName || '')}</span><span class="muted">on ${esc(p.name)}</span></div>
          <dl class="kv">${vrow('Temperature', v.temp ?? f.temp, '°C')}${vrow('Flow', v.flow, '%')}${vrow(p.firmware === 'marlin' ? 'Linear advance' : 'Pressure advance', v.pa)}
          ${vrow('Retraction', v.retractLen != null ? `${fmt(v.retractLen, 2)} mm @ ${fmt(v.retractSpeed ?? '', 0)}` : null, 'mm/s')}${vrow('Max flow', v.maxFlow, 'mm³/s')}
          ${vrow('Sliding fit', v.tolerance?.slide, 'mm')}${vrow('XY compensation', v.xyComp, 'mm')}</dl>
          <div class="row mt"><a class="btn sm" href="#/export">${icon('share', 14)}Export / share</a><a class="btn sm ghost" href="#/filaments">${icon('spool', 14)}Filaments</a></div></div>
        <div class="grid g2">
          <a class="card tile" href="#/test/firstlayer" style="min-height:0;color:inherit;text-decoration:none"><div class="ico alt2">${icon('layers', 20)}</div><h3>First layer</h3><p>Z offset & levelling check</p></a>
          <a class="card tile" href="#/mesh" style="min-height:0;color:inherit;text-decoration:none"><div class="ico">${icon('mesh', 20)}</div><h3>Bed mesh</h3><p>3D heightmap & screw turns</p></a>
          <a class="card tile" href="#/belt" style="min-height:0;color:inherit;text-decoration:none"><div class="ico alt3">${icon('mic', 20)}</div><h3>Belt tension</h3><p>Listen with your microphone</p></a>
          <a class="card tile" href="#/test/tolerance" style="min-height:0;color:inherit;text-decoration:none"><div class="ico alt">${icon('fit', 20)}</div><h3>Fit for Makie</h3><p>Hole & peg clearances</p></a>
        </div>
        <div class="card"><h3>${icon('history', 16)}Recent results</h3>${recent.length ? `<div class="col">${recent.map((h) => `<div class="list-item" style="padding:9px 12px"><div class="grow"><b>${esc(h.title)}</b><span>${esc(h.summary)} · ${esc(dateStr(h.date))}</span></div></div>`).join('')}</div>` : '<p class="muted small" style="margin:0">Results you save will show up here.</p>'}</div>
      </div></div>`, 'Home');
  animateRings();
  $$('[data-go]').forEach((el) => el.onclick = (e) => { if (e.target.closest('[data-tick]')) return; go(el.dataset.go); });
  $$('[data-tick]').forEach((b) => b.onclick = () => {
    const k = b.dataset.tick;
    if (cl[k]?.done) delete cl[k]; else cl[k] = { done: true, date: Date.now(), value: 'marked done' };
    DB.save(); renderHome();
  });
}

/* ================================================================ tests */
function renderTests() {
  const cats = ['Printer', 'Filament', 'Accuracy', 'Quality'];
  const cl = DB.checklist();
  const tile = (t) => `<a class="card tile" href="#/test/${t.id}" style="color:inherit;text-decoration:none"><div class="row" style="justify-content:space-between"><div class="ico ${t.tone}">${icon(t.icon, 20)}</div>${t.check && cl[t.check]?.done ? `<span class="chip ok">${icon('check', 12)}done</span>` : ''}</div>
    <h3>${esc(t.title)}</h3><p>${esc(t.summary)}</p><div class="meta"><span class="chip">${icon('history', 12)}${esc(t.time)}</span>${t.kind === 'commands' ? '<span class="chip">commands</span>' : '<span class="chip">G-code</span>'}</div></a>`;
  view(`<div class="page-head"><div><h1>Calibration tests</h1><div class="sub">Every test makes its own G-code — no slicer needed — and walks you through measuring and applying the result.</div></div></div>
    ${cats.map((c) => `<h2 class="mb" style="margin-top:6px">${c === 'Printer' ? 'Printer setup' : c === 'Filament' ? 'Per filament' : c}</h2><div class="grid gauto mb" style="margin-bottom:26px">${TESTS.filter((t) => t.cat === c).map(tile).join('')}
      ${c === 'Printer' ? `<a class="card tile" href="#/mesh" style="color:inherit;text-decoration:none"><div class="ico">${icon('mesh', 20)}</div><h3>Bed mesh viewer</h3><p>Read G29 / BED_MESH data, see a coloured 3D heightmap and get screw turns.</p><div class="meta"><span class="chip">tool</span></div></a>
      <a class="card tile" href="#/belt" style="color:inherit;text-decoration:none"><div class="ico alt3">${icon('mic', 20)}</div><h3>Belt tension</h3><p>Pluck the belt; the microphone tells you its frequency and whether both sides match.</p><div class="meta"><span class="chip">tool</span></div></a>` : ''}</div>`).join('')}`, 'Tests');
}

/* ================================================================ printers */
function presetPicker() {
  const brands = [...new Set(PRESETS.map((p) => p.brand))];
  return modal({
    title: 'Add a printer', wide: true,
    body: `<div class="inp mb">${icon('flask', 16, 'style="margin-left:10px;color:var(--dim)"')}<input id="pq" placeholder="Search printers…"></div>
      <div id="plist" style="max-height:58vh;overflow-y:auto">${brands.map((b) => `<div data-brand="${esc(b)}"><div class="nav-h" style="padding-left:2px">${esc(b)}</div><div class="grid g3">${PRESETS.filter((p) => p.brand === b).map((p) => `<button class="list-item" data-preset="${p.id}" data-name="${esc((p.brand + ' ' + p.name).toLowerCase())}" style="cursor:pointer;color:inherit;text-align:left"><div class="grow"><b>${esc(p.name)}</b><span>${p.bedShape === 'circle' ? 'Ø' + p.bedX : p.bedX + '×' + p.bedY}×${p.bedZ} · ${FIRMWARES[p.firmware].short}</span></div></button>`).join('')}</div></div>`).join('')}</div>`,
    actions: [{ label: 'Cancel', kind: 'ghost' }],
    onOpen: (bg) => {
      $('#pq', bg).focus();
      $('#pq', bg).oninput = (e) => { const q = e.target.value.toLowerCase(); $$('[data-preset]', bg).forEach((b) => b.style.display = b.dataset.name.includes(q) ? '' : 'none'); };
      $$('[data-preset]', bg).forEach((b) => b.onclick = () => {
        const p = printerFromPreset(b.dataset.preset);
        DB.data.printers.push(p); DB.data.activePrinter = p.id; DB.save(); bg.remove(); refreshPickers(); go('#/printer/' + p.id);
        toast('Added ' + p.name);
      });
    },
  });
}

function renderPrinters() {
  view(`<div class="page-head"><div><h1>Printers</h1><div class="sub">Each printer keeps its own settings, calibration values, checklist progress and connection.</div></div>
    <div class="actions"><button class="btn primary" id="addp">${icon('plus', 16)}Add printer</button></div></div>
    <div class="grid gauto">${DB.data.printers.map((p) => {
      const cl = DB.checklist(p.id); const n = CHECKLIST.filter((c) => cl[c.key]?.done).length;
      return `<div class="card tile ${p.id === DB.data.activePrinter ? 'sel' : ''}" data-id="${p.id}" style="min-height:0;${p.id === DB.data.activePrinter ? 'border-color:rgba(124,92,255,.6)' : ''}">
        <div class="row" style="justify-content:space-between"><div class="ico">${icon('printer', 20)}</div>${p.id === DB.data.activePrinter ? '<span class="chip acc">active</span>' : ''}</div>
        <h3>${esc(p.name)}</h3><p>${p.bedShape === 'circle' ? 'Ø' + p.bedX : p.bedX + ' × ' + p.bedY} × ${p.bedZ} mm · ${esc(FIRMWARES[p.firmware]?.short)} · ${p.nozzle} mm nozzle</p>
        <div class="row"><span class="chip">${n}/${CHECKLIST.length} calibrated</span><span class="chip">${esc(p.drive)}</span></div></div>`;
    }).join('')}</div>`, 'Printers');
  $('#addp').onclick = presetPicker;
  $$('[data-id]').forEach((el) => el.onclick = () => go('#/printer/' + el.dataset.id));
}

const PRINTER_FIELDS = [
  { k: 'name', label: 'Name', type: 'text', wide: true },
  { k: 'firmware', label: 'Firmware', type: 'select', options: Object.entries(FIRMWARES).map(([k, v]) => [k, v.name]) },
  { k: 'bedShape', label: 'Bed shape', type: 'select', options: [['rect', 'Rectangular'], ['circle', 'Round (delta)']] },
  { k: 'bedX', label: 'Bed width / diameter', unit: 'mm' }, { k: 'bedY', label: 'Bed depth', unit: 'mm' }, { k: 'bedZ', label: 'Max height', unit: 'mm' },
  { k: 'originCenter', label: '0,0 is the bed centre', type: 'check', help: 'Usual for deltas; most others have 0,0 at front-left.' },
  { k: 'nozzle', label: 'Nozzle', unit: 'mm', step: 0.05 }, { k: 'filamentDia', label: 'Filament diameter', unit: 'mm', step: 0.05 },
  { k: 'drive', label: 'Extruder', type: 'select', options: [['direct', 'Direct drive'], ['bowden', 'Bowden']] },
  { k: 'maxSpeed', label: 'Max speed', unit: 'mm/s' }, { k: 'maxAccel', label: 'Max acceleration', unit: 'mm/s²' },
  { k: 'maxHotend', label: 'Max hotend temp', unit: '°C' }, { k: 'maxBed', label: 'Max bed temp', unit: '°C' },
  { k: 'hasInputShaper', label: 'Has input shaping', type: 'check', help: 'Tunie switches it off during the ringing test.' },
  { k: 'mesh', label: 'Bed mesh at print start', type: 'select', options: [['none', 'None'], ['load', 'Load saved mesh'], ['probe', 'Probe every time']] },
];

function renderPrinterEdit(id) {
  const p = DB.data.printers.find((x) => x.id === id);
  if (!p) return go('#/printers');
  const vals = p.values || {};
  const vlist = Object.entries(vals).filter(([, v]) => v !== null && v !== undefined);
  view(`<div class="page-head"><div><h1>${esc(p.name)}</h1><div class="sub">${p.preset && p.preset !== 'custom' ? `Based on the ${esc(PRESETS.find((x) => x.id === p.preset)?.name || '')} preset — check the numbers match your machine.` : 'Custom printer — fill in your own numbers. Tiny and unusual printers are fine.'}</div></div>
    <div class="actions">${p.id !== DB.data.activePrinter ? `<button class="btn" id="pact">${icon('check', 16)}Make active</button>` : '<span class="chip acc">active printer</span>'}<button class="btn" id="pdup">${icon('copy', 16)}Duplicate</button><button class="btn danger" id="pdel">${icon('trash', 16)}Delete</button></div></div>
    <div class="grid" style="grid-template-columns:minmax(0,1.5fr) minmax(0,1fr);gap:18px;align-items:start">
      <div class="col"><div class="card pad-lg"><h3>${icon('printer', 16)}Machine</h3><div class="fields" id="pform">${PRINTER_FIELDS.map((f) => fieldHTML(f, p[f.k])).join('')}</div></div>
        <div class="card pad-lg"><h3>${icon('zap', 16)}Start & end G-code</h3><div class="card-sub">Leave empty to use Tunie's safe defaults for ${esc(FIRMWARES[p.firmware]?.short)}. Placeholders: <span class="mono">{hotend} {bed} {mesh} {bedX} {bedY} {nozzle}</span></div>
          <div class="row mb"><button class="btn sm" data-tpl="default">Show default</button>${p.firmware === 'klipper' ? '<button class="btn sm" data-tpl="macro">Use PRINT_START macro</button>' : ''}${p.firmware === 'prusa-buddy' ? '<button class="btn sm" data-tpl="mk4">Prusa MK4 style</button>' : ''}</div>
          <div class="grid g2"><div class="field"><label>Start G-code</label><textarea class="ta" id="sg" style="min-height:150px">${esc(p.startGcode || '')}</textarea></div><div class="field"><label>End G-code</label><textarea class="ta" id="eg" style="min-height:150px">${esc(p.endGcode || '')}</textarea></div></div></div></div>
      <div class="col"><div class="card"><h3>${icon('star', 16)}Calibrated values</h3>${vlist.length ? `<dl class="kv">${vlist.map(([k, v]) => `<dt>${esc(k)}</dt><dd class="mono">${esc(typeof v === 'object' ? Object.entries(v).filter(([kk]) => kk !== 'factor').map(([kk, vv]) => `${kk} ${fmt(vv, 3)}`).join(' · ') : v)}</dd>`).join('')}</dl><div class="row mt"><button class="btn sm ghost danger" id="pclr">${icon('trash', 14)}Clear values</button></div>` : '<p class="muted small" style="margin:0">Values from PID, E-steps, input shaper and accuracy tests are stored here.</p>'}</div>
        <div class="card"><h3>${icon('plug', 16)}Connection</h3><p class="muted small">${p.conn?.type ? `${esc(p.conn.type)} ${esc(p.conn.port || p.conn.url || '')}` : 'Not set up'}</p><a class="btn sm" href="#/connect">${icon('plug', 14)}Set up connection</a></div>
        ${p.firmware === 'bambu' ? `<div class="callout">${icon('bambu')}<div><p>Bambu printer — see <a href="#/bambu">what it calibrates by itself</a>.</p></div></div>` : ''}</div></div>`, p.name);
  const saveForm = () => {
    const v = readFields($('#pform'));
    Object.assign(p, v);
    if (p.bedShape === 'circle') { p.bedY = p.bedX; }
    p.startGcode = $('#sg').value; p.endGcode = $('#eg').value;
    DB.save(); refreshPickers();
  };
  $$('#pform [data-k]').forEach((el) => el.addEventListener('change', () => { const fw = p.firmware; saveForm(); if (p.firmware !== fw) renderPrinterEdit(id); }));
  $('#sg').addEventListener('change', saveForm); $('#eg').addEventListener('change', saveForm);
  $$('[data-tpl]').forEach((b) => b.onclick = () => {
    const t = b.dataset.tpl;
    if (t === 'default') { $('#sg').value = { rrf: 'T0\nM140 S{bed}\nM104 S{hotend}\nG28\n{mesh}\nM190 S{bed}\nM109 S{hotend}', 'prusa-mk3': 'M140 S{bed}\nM104 S{hotend}\nG28 W\n{mesh}\nM190 S{bed}\nM109 S{hotend}' }[p.firmware] || 'M140 S{bed}\nM104 S{hotend}\nG28\n{mesh}\nM190 S{bed}\nM109 S{hotend}'; $('#eg').value = 'M104 S0\nM140 S0\nM106 S0\nM84'; }
    if (t === 'macro') { $('#sg').value = 'PRINT_START BED={bed} EXTRUDER={hotend}'; $('#eg').value = 'PRINT_END'; }
    if (t === 'mk4') { $('#sg').value = 'M140 S{bed}\nM104 S170\nG28\nM190 S{bed}\nG29 ; mesh\nM109 S{hotend}'; $('#eg').value = 'M104 S0\nM140 S0\nM107\nM84'; }
    saveForm();
  });
  const act = $('#pact'); if (act) act.onclick = () => { DB.data.activePrinter = p.id; DB.save(); refreshPickers(); renderPrinterEdit(id); };
  $('#pdup').onclick = () => { const c = JSON.parse(JSON.stringify(p)); c.id = uid(); c.name += ' (copy)'; c.values = {}; DB.data.printers.push(c); DB.save(); refreshPickers(); go('#/printer/' + c.id); };
  $('#pdel').onclick = async () => {
    if (DB.data.printers.length < 2) return toast('Keep at least one printer', 'info');
    if (!(await confirmBox('Delete printer?', `Delete ${esc(p.name)} and its checklist progress? History entries are kept.`, 'Delete', 'danger'))) return;
    DB.data.printers = DB.data.printers.filter((x) => x.id !== p.id);
    if (DB.data.activePrinter === p.id) DB.data.activePrinter = DB.data.printers[0].id;
    DB.save(); refreshPickers(); go('#/printers');
  };
  const clr = $('#pclr'); if (clr) clr.onclick = async () => { if (await confirmBox('Clear calibrated values?', 'This only clears the stored numbers in Tunie — nothing on the printer changes.', 'Clear', 'danger')) { p.values = {}; DB.save(); renderPrinterEdit(id); } };
}

/* ================================================================ filaments */
const FIL_FIELDS = [
  { k: 'brand', label: 'Brand', type: 'text' }, { k: 'type', label: 'Type', type: 'select', options: [...Object.keys(FILAMENT_TYPES), 'Other'] },
  { k: 'colourName', label: 'Colour name', type: 'text' }, { k: 'colour', label: 'Colour', type: 'color' },
  { k: 'diameter', label: 'Diameter', unit: 'mm', step: 0.05 }, { k: 'temp', label: 'Print temperature', unit: '°C', step: 5 },
  { k: 'tempMin', label: 'Range: lowest', unit: '°C', step: 5 }, { k: 'tempMax', label: 'Range: highest', unit: '°C', step: 5 },
  { k: 'bed', label: 'Bed temperature', unit: '°C', step: 5 }, { k: 'fan', label: 'Part fan', unit: '%', step: 10 }, { k: 'density', label: 'Density', unit: 'g/cm³', step: 0.01 },
];

function renderFilaments() {
  const p = DB.printer();
  view(`<div class="page-head"><div><h1>Filaments</h1><div class="sub">Your spools, and what Tunie has learned about each one on <b>${esc(p.name)}</b>. Click a filament to make it active.</div></div>
    <div class="actions"><button class="btn primary" id="addf">${icon('plus', 16)}Add filament</button></div></div>
    <div class="grid gauto mb">${DB.data.filaments.map((f) => {
      const v = DB.profile(p.id, f.id).values;
      const n = Object.keys(v).length;
      return `<div class="card tile" data-id="${f.id}" style="min-height:0;${f.id === DB.data.activeFilament ? 'border-color:rgba(124,92,255,.6)' : ''}">
        <div class="row" style="justify-content:space-between"><span class="swatch" style="background:${esc(f.colour)};width:34px;height:34px;border-radius:10px"></span>${f.id === DB.data.activeFilament ? '<span class="chip acc">active</span>' : ''}</div>
        <h3>${esc(f.brand)} ${esc(f.type)}</h3><p>${esc(f.colourName || '')} · ${f.diameter} mm · ${v.temp ?? f.temp} °C${v.flow ? ` · flow ${v.flow} %` : ''}${v.pa != null ? ` · PA ${v.pa}` : ''}</p>
        <div class="row"><span class="chip ${n ? 'ok' : ''}">${n ? n + ' calibrated values' : 'not calibrated yet'}</span><button class="btn sm ghost" data-edit="${f.id}">${icon('edit', 14)}Edit</button></div></div>`;
    }).join('')}</div>
    <h2 class="mb">Filament library <span class="muted small" style="font-weight:500">— typical starting points; always follow your spool's label</span></h2>
    <div class="card" style="padding:6px 8px"><table class="tbl"><thead><tr><th>Type</th><th>Nozzle</th><th>Bed</th><th>Fan</th><th>Speed</th><th>Notes</th><th></th></tr></thead><tbody>
      ${Object.entries(FILAMENT_TYPES).map(([k, t]) => `<tr><td><b>${k}</b></td><td class="mono">${t.temp[0]}–${t.temp[1]} °C</td><td class="mono">${t.bed[0]}–${t.bed[1]} °C</td><td class="mono">${t.fan} %</td><td>${esc(t.speed)}</td><td class="muted small">${esc(t.notes)}</td><td><button class="btn sm" data-add="${k}">${icon('plus', 14)}Add</button></td></tr>`).join('')}</tbody></table></div>`, 'Filaments');
  $$('[data-id]').forEach((el) => el.onclick = (e) => { if (e.target.closest('[data-edit]')) return; DB.data.activeFilament = el.dataset.id; DB.save(); refreshPickers(); renderFilaments(); toast('Active filament changed'); });
  $$('[data-edit]').forEach((b) => b.onclick = () => editFilament(DB.data.filaments.find((f) => f.id === b.dataset.edit)));
  $$('[data-add]').forEach((b) => b.onclick = () => { const f = filamentFromType(b.dataset.add); editFilament(f, true); });
  $('#addf').onclick = () => editFilament(filamentFromType('PLA'), true);
}

async function editFilament(f, isNew = false) {
  const r = await modal({
    title: isNew ? 'Add filament' : `Edit ${esc(f.brand)} ${esc(f.type)}`,
    body: `<div class="fields" id="ff">${FIL_FIELDS.map((x) => fieldHTML(x, f[x.k])).join('')}</div>
      <p class="muted small mt">Changing the type fills in typical temperatures for it.</p>`,
    actions: [...(isNew ? [] : [{ label: 'Delete', kind: 'danger ghost', value: () => 'delete' }]), { label: 'Cancel', kind: 'ghost', value: () => null }, { label: isNew ? 'Add' : 'Save', kind: 'primary', value: (bg) => readFields($('#ff', bg)) }],
    onOpen: (bg) => {
      $('[data-k=type]', bg).onchange = (e) => {
        const t = FILAMENT_TYPES[e.target.value]; if (!t) return;
        const set = (k, v) => { const el = $(`[data-k=${k}]`, bg); if (el) el.value = v; };
        set('temp', t.start); set('tempMin', t.temp[0]); set('tempMax', t.temp[1]); set('bed', t.bedStart); set('fan', t.fan); set('density', t.density);
      };
    },
  });
  if (!r) return;
  if (r === 'delete') {
    if (DB.data.filaments.length < 2) return toast('Keep at least one filament', 'info');
    DB.data.filaments = DB.data.filaments.filter((x) => x.id !== f.id);
    if (DB.data.activeFilament === f.id) DB.data.activeFilament = DB.data.filaments[0].id;
  } else {
    Object.assign(f, r);
    if (isNew) { DB.data.filaments.push(f); DB.data.activeFilament = f.id; }
  }
  DB.save(); refreshPickers(); renderFilaments();
}

/* ================================================================ history */
function renderHistory() {
  const H = DB.data.history;
  const fp = DB.data.ui.hFilter || { printer: '', filament: '', test: '' };
  const rows = H.filter((h) => (!fp.printer || h.printerId === fp.printer) && (!fp.filament || h.filamentId === fp.filament) && (!fp.test || h.test === fp.test));
  view(`<div class="page-head"><div><h1>History</h1><div class="sub">Every saved result, per printer and filament. Load an older result to put its values back into your profile.</div></div></div>
    <div class="card mb"><div class="fields" id="hf">${fieldHTML({ k: 'printer', label: 'Printer', type: 'select', options: [['', 'All printers'], ...DB.data.printers.map((p) => [p.id, p.name])] }, fp.printer)}
      ${fieldHTML({ k: 'filament', label: 'Filament', type: 'select', options: [['', 'All filaments'], ...DB.data.filaments.map((f) => [f.id, `${f.brand} ${f.type} ${f.colourName || ''}`])] }, fp.filament)}
      ${fieldHTML({ k: 'test', label: 'Test', type: 'select', options: [['', 'All tests'], ...TESTS.map((t) => [t.id, t.title])] }, fp.test)}</div></div>
    ${rows.length ? `<div class="card" style="padding:6px 8px"><table class="tbl"><thead><tr><th>When</th><th>Test</th><th>Result</th><th>Printer</th><th>Filament</th><th></th></tr></thead><tbody>
      ${rows.map((h) => `<tr><td class="muted small">${esc(dateStr(h.date))}</td><td><b>${esc(h.title)}</b></td><td>${esc(h.summary)}</td><td class="muted">${esc(h.printerName)}</td><td class="muted">${esc(h.filamentName)}</td>
        <td style="white-space:nowrap"><button class="btn sm" data-load="${h.id}" title="Put these values back into the profile">${icon('refresh', 14)}Load</button><button class="btn sm ghost" data-del="${h.id}">${icon('trash', 14)}</button></td></tr>`).join('')}</tbody></table></div>`
      : `<div class="card empty">${icon('history', 40)}<h3>Nothing here yet</h3><p>Finish a test and click <b>Save result</b>.</p></div>`}`, 'History');
  $$('#hf [data-k]').forEach((el) => el.onchange = () => { DB.data.ui.hFilter = readFields($('#hf')); DB.save(); renderHistory(); });
  $$('[data-del]').forEach((b) => b.onclick = async () => { if (await confirmBox('Delete this entry?', 'This only removes it from History.', 'Delete', 'danger')) { DB.data.history = H.filter((h) => h.id !== b.dataset.del); DB.save(); renderHistory(); } });
  $$('[data-load]').forEach((b) => b.onclick = () => {
    const h = H.find((x) => x.id === b.dataset.load);
    const p = DB.data.printers.find((x) => x.id === h.printerId), f = DB.data.filaments.find((x) => x.id === h.filamentId);
    if (!p || !f) return toast('That printer or filament no longer exists', 'err');
    DB.data.activePrinter = p.id; DB.data.activeFilament = f.id;
    p.values = Object.assign(p.values || {}, h.save?.printer || {});
    const prof = DB.profile(p.id, f.id); Object.assign(prof.values, h.save?.profile || {}); prof.updated = Date.now();
    DB.save(); refreshPickers(); toast(`Loaded "${h.summary}" into ${p.name} · ${f.brand} ${f.type}`);
  });
}

/* ================================================================ export */
function profileRows() {
  const p = DB.printer(), f = DB.filament(), v = DB.profile().values, pv = p.values || {};
  const r = [];
  r.push(['Temperature', `${v.temp ?? f.temp} °C`]);
  if (v.flow) r.push(['Flow', `${fmt(v.flow, 1)} %`]);
  if (v.pa != null) r.push([p.firmware === 'marlin' ? 'Linear advance K' : 'Pressure advance', fmt(v.pa, 4)]);
  if (v.retractLen != null) r.push(['Retraction', `${fmt(v.retractLen, 2)} mm @ ${fmt(v.retractSpeed ?? 35, 0)} mm/s`]);
  if (v.maxFlow) r.push(['Max volumetric flow', `${fmt(v.maxFlow, 1)} mm³/s`]);
  if (v.tolerance) r.push(['Fit (press / slide / loose)', ['press', 'slide', 'loose'].map((k) => v.tolerance[k] != null ? fmt(v.tolerance[k], 2) : '—').join(' / ') + ' mm']);
  if (v.xyComp != null) r.push(['XY compensation', `${fmt(v.xyComp, 3)} mm`]);
  if (v.scaleX || v.scaleY || v.scaleZ) r.push(['Scale X / Y / Z', `${fmt(v.scaleX ?? 100, 2)} / ${fmt(v.scaleY ?? 100, 2)} / ${fmt(v.scaleZ ?? 100, 2)} %`]);
  if (pv.eSteps) r.push(['E-steps', `${fmt(pv.eSteps, 2)} steps/mm`]);
  if (pv.rotationDistance) r.push(['Rotation distance', fmt(pv.rotationDistance, 4)]);
  if (pv.shaperX) r.push(['Input shaper', `X ${fmt(pv.shaperX, 1)} Hz · Y ${fmt(pv.shaperY, 1)} Hz (${pv.shaperType || 'mzv'})`]);
  if (pv.pidHotend) r.push(['Hotend PID', `${fmt(pv.pidHotend.p)} / ${fmt(pv.pidHotend.i)} / ${fmt(pv.pidHotend.d)}`]);
  if (v.maxOverhang) r.push(['Clean overhang', `${v.maxOverhang}°`]);
  if (v.maxBridge) r.push(['Clean bridge', `${v.maxBridge} mm`]);
  return r;
}

function slicerExports() {
  const p = DB.printer(), f = DB.filament(), v = DB.profile().values;
  const name = `Tunie - ${f.brand} ${f.type} ${f.colourName || ''} (${p.name})`.replace(/\s+/g, ' ').trim();
  const t = v.temp ?? f.temp, flow = (v.flow ?? 100) / 100;
  const paLine = v.pa != null ? (p.firmware === 'klipper' ? `SET_PRESSURE_ADVANCE ADVANCE=${fmt(v.pa, 4)}` : p.firmware === 'rrf' ? `M572 D0 S${fmt(v.pa, 4)}` : p.firmware === 'prusa-buddy' ? `M572 S${fmt(v.pa, 4)}` : `M900 K${fmt(v.pa, 4)}`) : '';
  const prusa = [`# ${name}`, '# Made with Tunie by nrmluvdev — PrusaSlicer: File → Import → Import Config Bundle', `[filament:${name}]`,
    `filament_type = ${f.type}`, `filament_diameter = ${f.diameter}`, `filament_colour = ${f.colour}`, `filament_density = ${f.density ?? 1.24}`,
    `temperature = ${t}`, `first_layer_temperature = ${t + 5}`, `bed_temperature = ${f.bed}`, `first_layer_bed_temperature = ${f.bed}`,
    `extrusion_multiplier = ${fmt(flow, 3)}`, v.maxFlow ? `filament_max_volumetric_speed = ${fmt(v.maxFlow, 1)}` : null,
    v.retractLen != null ? `filament_retract_length = ${fmt(v.retractLen, 2)}` : null, v.retractSpeed != null ? `filament_retract_speed = ${fmt(v.retractSpeed, 0)}` : null,
    paLine ? `start_filament_gcode = "${paLine} ; Tunie"` : null, `max_fan_speed = ${f.fan ?? 100}`, `min_fan_speed = ${f.fan ?? 100}`,
  ].filter(Boolean).join('\n');
  const S = (x) => [String(x)];
  const orcaObj = {
    type: 'filament', name, from: 'User', instantiation: 'true', inherits: '',
    filament_type: S(f.type), filament_diameter: S(f.diameter), filament_colour: S(f.colour), filament_density: S(f.density ?? 1.24),
    nozzle_temperature: S(t), nozzle_temperature_initial_layer: S(t + 5),
    hot_plate_temp: S(f.bed), hot_plate_temp_initial_layer: S(f.bed), textured_plate_temp: S(f.bed), textured_plate_temp_initial_layer: S(f.bed),
    filament_flow_ratio: S(fmt(flow, 3)),
    ...(v.pa != null ? { enable_pressure_advance: S(1), pressure_advance: S(fmt(v.pa, 4)) } : {}),
    ...(v.maxFlow ? { filament_max_volumetric_speed: S(fmt(v.maxFlow, 1)) } : {}),
    ...(v.retractLen != null ? { filament_retraction_length: S(fmt(v.retractLen, 2)) } : {}),
    ...(v.retractSpeed != null ? { filament_retraction_speed: S(fmt(v.retractSpeed, 0)) } : {}),
    fan_max_speed: S(f.fan ?? 100), fan_min_speed: S(f.fan ?? 100),
    filament_notes: 'Made with Tunie by nrmluvdev',
  };
  const orca = JSON.stringify(orcaObj, null, 2);
  const cura = [`Cura — ${name}`, '(Cura has no import for these, so type them into your profile)', '',
    `Printing Temperature ............ ${t} °C`, `Printing Temperature Initial Layer ${t + 5} °C`, `Build Plate Temperature ......... ${f.bed} °C`,
    `Flow ............................ ${fmt(flow * 100, 1)} %`,
    v.retractLen != null ? `Retraction Distance ............. ${fmt(v.retractLen, 2)} mm` : null,
    v.retractSpeed != null ? `Retraction Speed ................ ${fmt(v.retractSpeed, 0)} mm/s` : null,
    v.xyComp != null ? `Horizontal Expansion ............ ${fmt(v.xyComp, 3)} mm` : null,
    v.maxOverhang ? `Support Overhang Angle .......... ${v.maxOverhang}°` : null,
    paLine ? `Start G-code (Machine Settings) .. add: ${paLine}` : null,
    v.maxFlow ? `Keep speeds under ............... ${fmt(v.maxFlow / (p.nozzle * 1.125 * p.nozzle * 0.5), 0)} mm/s at ${fmt(p.nozzle * 0.5, 2)} mm layers` : null,
  ].filter(Boolean).join('\n');
  return { name, prusa, orca, cura };
}

function firmwareExport() {
  const p = DB.printer(), pv = p.values || {}, v = DB.profile().values;
  const fw = p.firmware;
  if (fw === 'klipper') {
    const s = ['# printer.cfg values found by Tunie (nrmluvdev) for ' + p.name];
    const ex = [];
    if (pv.rotationDistance) ex.push(`rotation_distance: ${fmt(pv.rotationDistance, 4)}`);
    if (v.pa != null) ex.push(`pressure_advance: ${fmt(v.pa, 4)}`);
    if (pv.pidHotend) ex.push(`control: pid`, `pid_Kp: ${fmt(pv.pidHotend.p, 3)}`, `pid_Ki: ${fmt(pv.pidHotend.i, 3)}`, `pid_Kd: ${fmt(pv.pidHotend.d, 3)}`);
    if (ex.length) s.push('', '[extruder]', ...ex);
    if (pv.pidBed) s.push('', '[heater_bed]', 'control: pid', `pid_Kp: ${fmt(pv.pidBed.p, 3)}`, `pid_Ki: ${fmt(pv.pidBed.i, 3)}`, `pid_Kd: ${fmt(pv.pidBed.d, 3)}`);
    if (pv.shaperX) s.push('', '[input_shaper]', `shaper_freq_x: ${fmt(pv.shaperX, 1)}`, `shaper_freq_y: ${fmt(pv.shaperY, 1)}`, `shaper_type: ${pv.shaperType || 'mzv'}`);
    if (pv.skew) s.push('', '[skew_correction]', `# then: SET_SKEW XY=${fmt(pv.skew.ac, 2)},${fmt(pv.skew.bd, 2)},${fmt(pv.skew.ad, 2)}  and  SKEW_PROFILE SAVE=tunie`);
    return { title: 'printer.cfg snippet', text: s.join('\n'), lang: 'cfg', send: false };
  }
  if (fw === 'bambu') return { title: 'Bambu', text: '; Bambu printers manage firmware settings themselves.\n; Use the slicer exports above.', send: false };
  const c = [];
  if (pv.eSteps) c.push(`M92 E${fmt(pv.eSteps, 2)}`);
  if (pv.pidHotend) c.push(`M301 P${fmt(pv.pidHotend.p)} I${fmt(pv.pidHotend.i)} D${fmt(pv.pidHotend.d)}`);
  if (pv.pidBed && fw !== 'rrf') c.push(`M304 P${fmt(pv.pidBed.p)} I${fmt(pv.pidBed.i)} D${fmt(pv.pidBed.d)}`);
  if (v.pa != null) c.push(fw === 'rrf' ? `M572 D0 S${fmt(v.pa, 4)}` : fw === 'prusa-buddy' ? `M572 S${fmt(v.pa, 4)}` : `M900 K${fmt(v.pa, 4)}`);
  if (pv.shaperX) c.push(fw === 'rrf' ? `M593 P"${pv.shaperType || 'mzv'}" F${fmt(Math.min(pv.shaperX, pv.shaperY), 1)}` : `M593 X F${fmt(pv.shaperX, 1)}\nM593 Y F${fmt(pv.shaperY, 1)}`);
  if (pv.skew && fw !== 'rrf') c.push(`M852 S${fmt(pv.skew.factor, 5)}`);
  c.push(fw === 'rrf' ? 'M500 ; (and copy M92/M572/M593 into config.g)' : 'M500');
  return { title: `${FIRMWARES[fw].short} commands — all saved values at once`, text: c.join('\n'), send: true };
}

function renderExport() {
  const p = DB.printer(), f = DB.filament();
  const ex = slicerExports(), fwx = firmwareExport();
  const rows = profileRows();
  view(`<div class="page-head"><div><h1>Export & share</h1><div class="sub">A summary card to share, files for your slicer, and every firmware value in one place — for <b>${esc(p.name)}</b> with <b>${esc(f.brand)} ${esc(f.type)}</b>.</div></div></div>
    <div class="grid" style="grid-template-columns:minmax(0,1.2fr) minmax(0,1fr);gap:18px;align-items:start">
      <div class="col"><div class="card" style="padding:12px"><canvas id="card" style="width:100%;border-radius:12px;display:block;aspect-ratio:16/9"></canvas></div>
        <div class="row"><button class="btn primary" id="savecard">${icon('image', 16)}Save summary card (PNG)</button><button class="btn" id="copysum">${icon('copy', 16)}Copy as text</button></div>
        ${codeBlock(fwx.title, fwx.text, { send: fwx.send && p.firmware !== 'bambu', lang: fwx.lang })}</div>
      <div class="col">
        <div class="card"><h3>${icon('download', 16)}Slicer profiles</h3><div class="card-sub">Saved to your Tunie folder in Documents.</div>
          <div class="col">
            <button class="list-item" data-x="prusa" style="cursor:pointer;color:inherit;text-align:left"><div class="grow"><b>PrusaSlicer</b><span>Config bundle (.ini) — File → Import → Import Config Bundle</span></div>${icon('download', 16)}</button>
            <button class="list-item" data-x="orca" style="cursor:pointer;color:inherit;text-align:left"><div class="grow"><b>OrcaSlicer</b><span>Filament preset (.json) — File → Import → Import Configs</span></div>${icon('download', 16)}</button>
            <button class="list-item" data-x="bambu" style="cursor:pointer;color:inherit;text-align:left"><div class="grow"><b>Bambu Studio</b><span>Filament preset (.json) — File → Import → Import Configs</span></div>${icon('download', 16)}</button>
            <button class="list-item" data-x="cura" style="cursor:pointer;color:inherit;text-align:left"><div class="grow"><b>Cura</b><span>Settings list (.txt) — type them into your profile</span></div>${icon('download', 16)}</button>
          </div><p class="small dim mt" style="margin-bottom:0">If a slicer refuses a preset, make a copy of its Generic ${esc(f.type)} preset and type in the values from the card.</p></div>
        <div class="card"><h3>${icon('link', 16)}Shared with Makie</h3><p class="muted small">The active printer and filament — including fit clearances and shrinkage — are kept in a shared suite file for Makie to read when sizing holes and pegs.</p><div class="code"><pre id="suitepath">${esc(APP.suiteFile || '')}</pre></div></div>
      </div></div>`, 'Export');
  drawSummaryCard($('#card'), rows);
  $('#savecard').onclick = async () => {
    const cv = document.createElement('canvas'); drawSummaryCard(cv, rows, 1600, 900);
    const b64 = cv.toDataURL('image/png').split(',')[1];
    try { const r = await api.post('/api/save-file', { name: `Tunie_${p.name}_${f.brand}_${f.type}.png`.replace(/\s+/g, '_'), content: b64, encoding: 'base64' }); showSavedPath(r.path); } catch (e) { toast(e.message, 'err'); }
  };
  $('#copysum').onclick = () => copyText(`Tunie — ${p.name} · ${f.brand} ${f.type} ${f.colourName || ''}\n` + rows.map(([k, v]) => `${k}: ${v}`).join('\n'));
  $$('[data-x]').forEach((b) => b.onclick = async () => {
    const k = b.dataset.x;
    const base = ex.name.replace(/[^\w\- ()]+/g, '').replace(/\s+/g, '_');
    const file = { prusa: [base + '.ini', ex.prusa], orca: [base + '_Orca.json', ex.orca], bambu: [base + '_BambuStudio.json', ex.orca], cura: [base + '_Cura.txt', ex.cura] }[k];
    try { const r = await api.post('/api/save-file', { name: file[0], content: file[1] }); showSavedPath(r.path); } catch (e) { toast(e.message, 'err'); }
  });
}

function drawSummaryCard(cv, rows, W = 1280, H = 720) {
  const p = DB.printer(), f = DB.filament();
  cv.width = W; cv.height = H;
  const ctx = cv.getContext('2d');
  const s = W / 1280;
  const g = ctx.createLinearGradient(0, 0, W, H); g.addColorStop(0, '#141831'); g.addColorStop(0.6, '#0f1320'); g.addColorStop(1, '#0b1a22');
  ctx.fillStyle = g; ctx.fillRect(0, 0, W, H);
  const glow = (x, y, r, c) => { const rg = ctx.createRadialGradient(x, y, 0, x, y, r); rg.addColorStop(0, c); rg.addColorStop(1, 'transparent'); ctx.fillStyle = rg; ctx.fillRect(0, 0, W, H); };
  glow(W * 0.95, 0, 520 * s, 'rgba(34,211,238,.20)'); glow(0, H, 520 * s, 'rgba(124,92,255,.25)');
  const font = (w, px) => `${w} ${px * s}px "Segoe UI Variable Display", "Segoe UI", system-ui, sans-serif`;
  const img = new Image();
  const paint = () => {
    ctx.drawImage(img, 60 * s, 52 * s, 64 * s, 64 * s);
    ctx.fillStyle = '#e8ebf2'; ctx.font = font(700, 40); ctx.fillText('Calibrated profile', 142 * s, 92 * s);
    ctx.fillStyle = '#8d95aa'; ctx.font = font(500, 20); ctx.fillText(new Date().toLocaleDateString(undefined, { day: 'numeric', month: 'long', year: 'numeric' }), 144 * s, 120 * s);
    // printer + filament
    ctx.fillStyle = '#e8ebf2'; ctx.font = font(650, 28); ctx.fillText(p.name, 60 * s, 190 * s);
    ctx.fillStyle = f.colour || '#888'; ctx.beginPath(); ctx.arc(72 * s, 226 * s, 12 * s, 0, 7); ctx.fill();
    ctx.fillStyle = '#c4cbe0'; ctx.font = font(600, 22); ctx.fillText(`${f.brand} ${f.type} ${f.colourName || ''} · ${f.diameter} mm`, 94 * s, 234 * s);
    ctx.fillStyle = '#8d95aa'; ctx.font = font(500, 18); ctx.fillText(`${FIRMWARES[p.firmware]?.name || p.firmware} · ${p.nozzle} mm nozzle · ${p.bedShape === 'circle' ? 'Ø' + p.bedX : p.bedX + '×' + p.bedY} mm bed`, 60 * s, 266 * s);
    // grid
    const list = rows.slice(0, 12);
    const cols = 3, cw = (W - 120 * s - (cols - 1) * 18 * s) / cols, ch = 92 * s;
    list.forEach(([k, v], i) => {
      const x = 60 * s + (i % cols) * (cw + 18 * s), y = 300 * s + Math.floor(i / cols) * (ch + 14 * s);
      ctx.fillStyle = 'rgba(255,255,255,.045)'; ctx.strokeStyle = 'rgba(255,255,255,.08)'; ctx.lineWidth = 1.5 * s;
      ctx.beginPath(); ctx.roundRect(x, y, cw, ch, 16 * s); ctx.fill(); ctx.stroke();
      ctx.fillStyle = '#8d95aa'; ctx.font = font(600, 15); ctx.fillText(k.toUpperCase(), x + 20 * s, y + 32 * s);
      const tg = ctx.createLinearGradient(x, 0, x + cw, 0); tg.addColorStop(0, '#a796ff'); tg.addColorStop(1, '#5eead4');
      ctx.fillStyle = tg; ctx.font = font(700, 26);
      let txt = v; while (ctx.measureText(txt).width > cw - 40 * s && txt.length > 4) txt = txt.slice(0, -2);
      ctx.fillText(txt === v ? v : txt + '…', x + 20 * s, y + 68 * s);
    });
    if (!list.length) { ctx.fillStyle = '#8d95aa'; ctx.font = font(500, 22); ctx.fillText('No calibrated values yet — run a test and save the result.', 60 * s, 340 * s); }
    ctx.fillStyle = '#5f677c'; ctx.font = font(500, 16);
    ctx.fillText('Tunie · part of the nrmluvdev maker suite · made by nrmluvdev', 60 * s, H - 40 * s);
  };
  img.onload = paint; img.src = 'icon.png';
  if (img.complete) paint();
}

/* ================================================================ connection */
function renderConnect() {
  const p = DB.printer();
  p.conn = p.conn || { type: 'serial', baud: 115200 };
  const c = p.conn;
  view(`<div class="page-head"><div><h1>Connection</h1><div class="sub">Optional. Connect ${esc(p.name)} to send tests, read values and run tunes from Tunie. Without it, everything still works using files and copy-paste.</div></div></div>
    <div class="grid" style="grid-template-columns:minmax(0,1fr) minmax(0,1.3fr);gap:18px;align-items:start">
      <div class="col"><div class="card pad-lg">
        <div class="seg mb" id="ctype">${[['serial', 'USB cable'], ['moonraker', 'Klipper (Moonraker)'], ['octoprint', 'OctoPrint']].map(([k, l]) => `<button data-t="${k}" class="${c.type === k ? 'on' : ''}">${l}</button>`).join('')}</div>
        <div id="cform"></div>
        <div class="row mt"><button class="btn primary" id="cgo">${icon('plug', 16)}${CONN.connected ? 'Reconnect' : 'Connect'}</button>${CONN.connected ? `<button class="btn" id="cstop">${icon('x', 16)}Disconnect</button>` : ''}</div></div>
        <div class="card"><h3>${icon('info', 16)}Status</h3><div id="cstat" class="muted">${CONN.connected ? esc(CONN.kind) : 'Not connected'}</div></div>
        ${p.firmware === 'bambu' ? `<div class="callout warn">${icon('bambu')}<div><p>Bambu printers don't offer these connections. Save files to the SD card or send them from Bambu Studio.</p></div></div>` : ''}</div>
      <div class="col"><div class="card"><div class="row" style="justify-content:space-between;margin-bottom:10px"><h3 style="margin:0">${icon('zap', 16)}Console</h3><div class="row">${['M105', 'M503', 'G28', 'M84', 'M114'].map((g) => `<button class="btn sm" data-q="${g}">${g}</button>`).join('')}</div></div>
        <div class="console" id="console"></div>
        <div class="row mt-s"><div class="inp" style="flex:1"><input id="cin" placeholder="Type a G-code command and press Enter" class="mono"></div><button class="btn" id="csend">${icon('send', 16)}Send</button></div></div></div></div>`, 'Connection');
  const form = () => {
    const t = c.type;
    $('#cform').innerHTML = t === 'serial'
      ? `<div class="fields"><div class="field"><label>COM port</label><div class="inp"><select data-k="port" id="cport"><option>${esc(c.port || 'Loading…')}</option></select><button class="btn sm ghost" id="cref" title="Refresh">${icon('refresh', 14)}</button></div></div>
         ${fieldHTML({ k: 'baud', label: 'Baud rate', type: 'select', options: ['115200', '250000', '57600', '230400', '500000', '1000000'] }, String(c.baud || 115200))}</div>
         <p class="small dim mt" style="margin-bottom:0">Close other programs using the port (Cura, Pronterface, Arduino IDE). Most printers restart when connected — that's normal.</p>`
      : `<div class="fields">${fieldHTML({ k: 'url', label: t === 'moonraker' ? 'Printer address (Mainsail/Fluidd)' : 'OctoPrint address', type: 'text', wide: true, ph: 'http://192.168.1.50' }, c.url)}
         ${fieldHTML({ k: 'apiKey', label: t === 'moonraker' ? 'API key (only if required)' : 'API key', type: 'text', wide: true }, c.apiKey)}</div>
         <p class="small dim mt" style="margin-bottom:0">${t === 'octoprint' ? 'OctoPrint → Settings → Application Keys. Note: OctoPrint doesn\'t pass printer replies back to Tunie, so you\'ll paste results from its Terminal tab.' : 'Use the same address you open Mainsail or Fluidd with.'}</p>`;
    if (t === 'serial') {
      const load = async () => {
        let ports = [];
        try { ports = await api.get('/api/conn/ports'); } catch { }
        $('#cport').innerHTML = (ports || []).length ? ports.map((x) => `<option ${x === c.port ? 'selected' : ''}>${esc(x)}</option>`).join('') : '<option value="">No ports found — plug the printer in</option>';
      };
      load(); $('#cref').onclick = (e) => { e.preventDefault(); load(); };
    }
  };
  form();
  $$('#ctype button').forEach((b) => b.onclick = () => { c.type = b.dataset.t; DB.save(); $$('#ctype button').forEach((x) => x.classList.toggle('on', x === b)); form(); });
  $('#cgo').onclick = async () => {
    Object.assign(c, readFields($('#cform')));
    c.baud = parseInt(c.baud) || 115200;
    DB.save();
    const b = $('#cgo'); b.disabled = true; b.innerHTML = '<span class="spin"></span>Connecting…';
    try { await api.post('/api/conn/connect', { type: c.type, port: c.port, baud: c.baud, url: c.url, apiKey: c.apiKey }); toast('Connected'); await pollConn(); }
    catch (e) { toast(e.message, 'err', 6000); }
    renderConnect();
  };
  const st = $('#cstop'); if (st) st.onclick = async () => { await api.post('/api/conn/disconnect'); await pollConn(); renderConnect(); };
  const send = async (line) => {
    if (!line.trim()) return;
    try { const r = await sendLines([line]); if (r.error) toast(r.error, 'err'); if (r.lines?.length) document.dispatchEvent(new CustomEvent('printer-reply', { detail: r.lines })); } catch (e) { toast(e.message, 'err'); }
    pullLog();
  };
  $('#csend').onclick = () => { send($('#cin').value); $('#cin').value = ''; };
  $('#cin').onkeydown = (e) => { if (e.key === 'Enter') { send(e.target.value); e.target.value = ''; } };
  $$('[data-q]').forEach((b) => b.onclick = () => send(b.dataset.q));
  CONSOLE.seq = 0; $('#console').innerHTML = '';
  pullLog();
}
const CONSOLE = { seq: 0 };
async function pullLog() {
  const el = $('#console'); if (!el) return;
  try {
    const lines = await api.get('/api/conn/log?since=' + CONSOLE.seq);
    if (lines.length) {
      CONSOLE.seq = lines[lines.length - 1].seq;
      const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 40;
      el.insertAdjacentHTML('beforeend', lines.map((l) => `<div class="${l.dir}">${esc(l.text)}</div>`).join(''));
      if (atBottom) el.scrollTop = el.scrollHeight;
    }
  } catch { }
  if (!el.dataset.empty && !el.children.length) { el.dataset.empty = 1; el.innerHTML = '<div class="info">Connect to see what the printer says here.</div>'; }
}

/* ================================================================ Bambu */
function renderBambu() {
  view(`<div class="page-head"><div><h1>Bambu Lab printers</h1><div class="sub">Bambu printers calibrate a lot by themselves. Here's what's covered, and where Tunie still helps. (Exactly what runs depends on the model and firmware version.)</div></div></div>
    <div class="card" style="padding:6px 8px"><table class="tbl"><thead><tr><th>Calibration</th><th>Done by the printer?</th><th>What to do</th></tr></thead><tbody>
      ${CHECKLIST.map((it) => { const b = BAMBU_AUTO[it.key]; return `<tr><td><b>${esc(it.title)}</b></td><td>${b.auto === true ? '<span class="chip ok">Automatic</span>' : b.auto === 'partly' ? '<span class="chip wn">Some models</span>' : '<span class="chip">No</span>'}</td><td class="muted">${esc(b.text)}</td></tr>`; }).join('')}
      <tr><td><b>Bed levelling</b></td><td><span class="chip ok">Automatic</span></td><td class="muted">Auto bed levelling runs as part of the start-up / print-start calibration.</td></tr>
      <tr><td><b>Motor noise</b></td><td><span class="chip ok">Automatic</span></td><td class="muted">Motor noise cancellation is part of the calibration menu on models that support it.</td></tr>
      <tr><td><b>Max volumetric flow, tolerance, overhang, bridging</b></td><td><span class="chip">No</span></td><td class="muted">Bambu Studio's Calibration menu has max flowrate tests; Tunie's tolerance results also feed Makie.</td></tr></tbody></table></div>
    <div class="grid g2 mt">
      <div class="callout">${icon('info')}<div><p><b>Printing Tunie's files on a Bambu printer.</b> Bambu printers normally receive <span class="mono">.3mf</span> files from Bambu Studio. Tunie's plain <span class="mono">.gcode</span> uses a generic start sequence. Some models and firmware will list and print it from the SD card; if yours doesn't, the safer route is Bambu Studio's own Calibration tab for flow, pressure advance, temperature, retraction and max flow.</p></div></div>
      <div class="callout good">${icon('check')}<div><p><b>Where Tunie shines on Bambu:</b> the dimensional accuracy frame, tolerance/fit (shared with Makie), overhang and bridging limits, first-layer check, the belt tension helper, and keeping a history per filament with slicer exports for Bambu Studio and OrcaSlicer.</p></div></div></div>`, 'Bambu Lab');
}

/* ================================================================ about */
function renderAbout() {
  view(`<div class="hero-banner mb"><div class="row" style="gap:20px"><img src="icon.png" style="width:88px;height:88px;border-radius:22px;box-shadow:0 14px 40px -12px rgba(124,92,255,.8)">
      <div><h1>Tunie</h1><p style="margin:4px 0 0">3D printer calibration helper · version ${esc(APP.version)}<br>Created by <b>nrmluvdev</b> — free, part of the nrmluvdev maker suite.</p></div></div></div>
    <div class="grid g3">
      ${[['Wirie', 'Circuit board designer', 'grid'], ['Scopie', 'Oscilloscope', 'wave'], ['Makie', '3D part designer — can read Tunie\'s fit clearances', 'cube'], ['Flashie', 'Firmware flasher', 'zap'], ['Tunie', 'Printer calibration (you are here)', 'flask']].map(([n, d, ic]) => `<div class="card tile" style="min-height:0;cursor:default"><div class="ico ${n === 'Tunie' ? '' : 'alt2'}">${icon(ic, 20)}</div><h3>${n}</h3><p>${esc(d)}</p></div>`).join('')}
    </div>
    <div class="grid g2 mt">
      <div class="card"><h3>${icon('folder', 16)}Where your data lives</h3><dl class="kv"><dt>Profiles & history</dt><dd class="mono small">${esc(APP.dataDir || '')}</dd><dt>Exports</dt><dd class="mono small">${esc(APP.exportDir || '')}</dd><dt>Suite link file</dt><dd class="mono small">${esc(APP.suiteFile || '')}</dd></dl>
        <div class="row mt"><button class="btn sm" id="ofold">${icon('folder', 14)}Open exports folder</button><button class="btn sm" id="bk">${icon('download', 14)}Back up all data</button><button class="btn sm" id="rs">${icon('share', 14)}Restore backup</button></div></div>
      <div class="card"><h3>${icon('link', 16)}Opening Tunie from Makie</h3><p class="muted small">Makie can launch Tunie with:</p><div class="code"><pre>Tinkerie.exe --app tunie --from makie --printer "Ender 3 V2" --filament "Generic PLA" --page tolerance</pre></div>
        <p class="muted small mt">Whichever printer and filament are active are written to the suite link file, which Makie reads for clearances, shrinkage and line width.</p></div></div>`, 'About');
  $('#ofold').onclick = () => api.post('/api/open-folder', {});
  $('#bk').onclick = async () => { try { const r = await api.post('/api/save-file', { name: `Tunie_backup_${new Date().toISOString().slice(0, 10)}.json`, content: JSON.stringify(DB.data, null, 1) }); showSavedPath(r.path); } catch (e) { toast(e.message, 'err'); } };
  $('#rs').onclick = () => {
    const i = document.createElement('input'); i.type = 'file'; i.accept = '.json';
    i.onchange = async () => {
      try {
        const d = JSON.parse(await i.files[0].text());
        if (!d.printers || !d.filaments) throw new Error('That isn\'t a Tunie backup');
        if (!(await confirmBox('Restore backup?', 'This replaces all printers, filaments and history with the backup.', 'Restore', 'danger'))) return;
        DB.data = d; DB.save(); refreshPickers(); toast('Backup restored'); go('#/');
      } catch (e) { toast(e.message, 'err'); }
    };
    i.click();
  };
}
