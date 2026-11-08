/* Tunie — app.js: start-up, navigation, printer/filament pickers and live status.
   Part of the nrmluvdev maker suite. Created by nrmluvdev. */
'use strict';

const NAV = [
  ['Calibrate', null],
  ['#/', 'Home', 'home'], ['#/tests', 'Tests', 'flask'], ['#/mesh', 'Bed mesh', 'mesh'], ['#/belt', 'Belt tension', 'mic'],
  ['Library', null],
  ['#/printers', 'Printers', 'printer'], ['#/filaments', 'Filaments', 'spool'], ['#/history', 'History', 'history'], ['#/export', 'Export & share', 'share'],
  ['Setup', null],
  ['#/connect', 'Connection', 'plug'], ['#/bambu', 'Bambu Lab notes', 'bambu'], ['#/about', 'About', 'info'],
];

function go(hash) { if (location.hash === hash) route(); else location.hash = hash; }

function renderNav() {
  const h = location.hash || '#/';
  $('#nav').innerHTML = NAV.map(([a, b, ic]) => {
    if (b === null) return `<div class="nav-h">${a}</div>`;
    const on = a === '#/' ? (h === '#/' || h === '') : h.startsWith(a) || (a === '#/tests' && h.startsWith('#/test/')) || (a === '#/printers' && h.startsWith('#/printer/'));
    const badge = a === '#/history' && DB.data.history.length ? `<span class="badge">${DB.data.history.length}</span>` : '';
    return `<a href="${a}" class="${on ? 'on' : ''}">${icon(ic, 18)}<span>${b}</span>${badge}</a>`;
  }).join('');
}

function refreshPickers() {
  const sp = $('#pick-printer'), sf = $('#pick-filament');
  sp.innerHTML = DB.data.printers.map((p) => `<option value="${p.id}" ${p.id === DB.data.activePrinter ? 'selected' : ''}>${esc(p.name)}</option>`).join('') + '<option value="__add">+ Add printer…</option>';
  sf.innerHTML = DB.data.filaments.map((f) => `<option value="${f.id}" ${f.id === DB.data.activeFilament ? 'selected' : ''}>${esc(f.brand)} ${esc(f.type)} ${esc(f.colourName || '')}</option>`).join('') + '<option value="__add">+ Add filament…</option>';
  $('#fil-swatch').style.background = DB.filament()?.colour || '#888';
  $('#printer-ico').innerHTML = icon('printer', 16, 'style="color:var(--muted)"');
  publishSuite();
}

async function route() {
  if (BELT.stream && !location.hash.startsWith('#/belt')) beltStop();
  if (WIZ.preview && !location.hash.startsWith('#/test/')) { WIZ.preview.destroy(); WIZ.preview = null; }
  renderNav();
  const h = (location.hash || '#/').slice(1);
  const [, a, b] = h.split('/');
  switch (a || '') {
    case '': return renderHome();
    case 'tests': return renderTests();
    case 'test': return renderTest(b);
    case 'mesh': return renderMesh();
    case 'belt': return renderBelt();
    case 'printers': return renderPrinters();
    case 'printer': return renderPrinterEdit(b);
    case 'filaments': return renderFilaments();
    case 'history': return renderHistory();
    case 'export': return renderExport();
    case 'connect': return renderConnect();
    case 'bambu': return renderBambu();
    case 'about': return renderAbout();
    default: return renderHome();
  }
}

/* live connection status */
async function pollConn() {
  try {
    const s = await api.get('/api/conn/status');
    CONN.connected = !!s.connected; CONN.kind = s.kind || ''; CONN.status = s.status || {}; CONN.job = s.job;
  } catch { CONN.connected = false; }
  const pill = $('#conn-pill');
  pill.classList.toggle('on', CONN.connected);
  let txt = 'Not connected';
  if (CONN.connected) {
    const t = CONN.status.temps || {};
    const temp = (v) => v === undefined || v === null ? '' : Math.round(v) + '°';
    txt = CONN.kind.split(' ')[0];
    txt = { serial: 'USB', moonraker: 'Klipper', octoprint: 'OctoPrint' }[txt] || txt;
    if (t.hotend !== undefined) txt += ` · ${temp(t.hotend)}${t.bed !== undefined ? ' / ' + temp(t.bed) : ''}`;
  }
  if (CONN.job && CONN.job.state === 'running') txt += ` · printing ${Math.round(100 * CONN.job.done / Math.max(1, CONN.job.total))}%`;
  $('#conn-text').textContent = txt;
  if ($('#console')) pullLog();
}

async function firstRun() {
  const brands = [...new Set(PRESETS.map((p) => p.brand))];
  const r = await modal({
    title: 'Welcome to Tunie 👋', wide: true,
    body: `<p class="muted">Tunie walks you through calibrating your 3D printer: <b>print this → measure this → type the number → here's your new setting.</b> First, which printer do you have? (You can add more later, or make a fully custom one for small or unusual DIY printers.)</p>
      <div class="fields mt" id="fr">${fieldHTML({ k: 'preset', label: 'Printer', type: 'select', options: brands.flatMap((b) => PRESETS.filter((p) => p.brand === b).map((p) => [p.id, `${b} — ${p.name}`])) }, 'ender3v2')}
      ${fieldHTML({ k: 'fil', label: 'First filament', type: 'select', options: Object.keys(FILAMENT_TYPES) }, 'PLA')}
      ${fieldHTML({ k: 'brand', label: 'Filament brand', type: 'text', ph: 'e.g. eSun, Polymaker, Sunlu' }, 'Generic')}</div>`,
    actions: [{ label: "Let's go", kind: 'primary', icon: 'arrowR', value: (bg) => readFields($('#fr', bg)) }],
  });
  const v = r || { preset: 'ender3v2', fil: 'PLA', brand: 'Generic' };
  const p = printerFromPreset(v.preset);
  const f = filamentFromType(v.fil, { brand: v.brand || 'Generic' });
  DB.data.printers.push(p); DB.data.filaments.push(f);
  DB.data.activePrinter = p.id; DB.data.activeFilament = f.id;
  DB.save();
}

function applyLaunch(l) {
  if (!l) return;
  const seen = DB.data.ui.lastLaunch;
  if (l.seq && seen === l.seq) return;
  DB.data.ui.lastLaunch = l.seq;
  if (l.printer) { const p = DB.data.printers.find((x) => x.name.toLowerCase() === l.printer.toLowerCase()); if (p) DB.data.activePrinter = p.id; }
  if (l.filament) { const q = l.filament.toLowerCase(); const f = DB.data.filaments.find((x) => `${x.brand} ${x.type} ${x.colourName || ''}`.toLowerCase().includes(q) || `${x.brand} ${x.type}`.toLowerCase() === q); if (f) DB.data.activeFilament = f.id; }
  DB.save(); refreshPickers();
  if (l.from) toast(`Opened from ${l.from.charAt(0).toUpperCase() + l.from.slice(1)} — the active profile is shared with it`, 'info', 5000);
  if (l.page) go(TEST_BY_ID[l.page] ? '#/test/' + l.page : '#/' + l.page);
  else if (l.from) route();
}

(async function boot() {
  // the event stream also tells Tunie when this window closes (so it can exit)
  const es = new EventSource('/api/events');
  es.addEventListener('activate', (e) => { try { if (DB.data) applyLaunch(JSON.parse(e.data)); window.focus(); } catch { } });
  let info = {};
  try { info = await api.get('/api/info'); } catch { }
  Object.assign(APP, info);
  $('#ver').textContent = 'v' + (info.version || '');
  await DB.load();
  if (!DB.data.printers.length || !DB.data.filaments.length) await firstRun();
  if (!DB.data.printers.find((p) => p.id === DB.data.activePrinter)) DB.data.activePrinter = DB.data.printers[0].id;
  if (!DB.data.filaments.find((f) => f.id === DB.data.activeFilament)) DB.data.activeFilament = DB.data.filaments[0].id;
  refreshPickers();
  $('#pick-printer').onchange = (e) => {
    if (e.target.value === '__add') { refreshPickers(); return presetPicker(); }
    DB.data.activePrinter = e.target.value; DB.save(); refreshPickers(); route();
  };
  $('#pick-filament').onchange = (e) => {
    if (e.target.value === '__add') { refreshPickers(); return editFilament(filamentFromType('PLA'), true); }
    DB.data.activeFilament = e.target.value; DB.save(); refreshPickers(); route();
  };
  $('#conn-pill').onclick = () => go('#/connect');
  window.addEventListener('hashchange', route);
  await pollConn();
  setInterval(pollConn, 3000);
  route();
  applyLaunch(info.launch);
  document.addEventListener('keydown', (e) => { if (e.key === 'Escape') { const m = $('.modal-bg'); if (m) m.remove(); } });
})();
