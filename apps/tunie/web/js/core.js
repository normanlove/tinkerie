/* Tunie — core.js: helpers, API, saved data, icons, toasts and dialogs.
   Part of the nrmluvdev maker suite. Created by nrmluvdev. */
'use strict';

const $ = (s, el = document) => el.querySelector(s);
const $$ = (s, el = document) => [...el.querySelectorAll(s)];
const esc = (s) => String(s ?? '').replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
const uid = () => Date.now().toString(36) + Math.random().toString(36).slice(2, 7);
const clamp = (v, a, b) => Math.max(a, Math.min(b, v));
const round = (v, d = 2) => { const m = 10 ** d; return Math.round(v * m) / m; };
const fmt = (v, d = 2) => (v === undefined || v === null || isNaN(v)) ? '—' : String(round(+v, d));
const num = (v) => { const n = parseFloat(String(v).replace(',', '.')); return isNaN(n) ? NaN : n; };
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const dateStr = (t) => new Date(t).toLocaleString(undefined, { day: 'numeric', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' });
const debounce = (fn, ms) => { let t; return (...a) => { clearTimeout(t); t = setTimeout(() => fn(...a), ms); }; };

/* ------------------------------------------------------------------ API */
const api = {
  async req(method, path, body) {
    const r = await fetch(path, {
      method, headers: { 'X-Tunie': '1', 'Content-Type': 'application/json' },
      body: body === undefined ? undefined : (typeof body === 'string' ? body : JSON.stringify(body)),
    });
    const t = await r.text();
    let j; try { j = t ? JSON.parse(t) : {}; } catch { j = { raw: t }; }
    if (!r.ok) throw new Error((j && j.error) || t || r.statusText);
    return j;
  },
  get: (p) => api.req('GET', p),
  post: (p, b) => api.req('POST', p, b ?? {}),
  put: (p, b) => api.req('PUT', p, b),
};

/* ------------------------------------------------------------------ saved data */
const DB = {
  data: null,
  async load() {
    let d = {};
    try { d = await api.get('/api/data'); } catch (e) { console.error(e); }
    this.data = Object.assign({
      version: 1, printers: [], filaments: [], history: [], profiles: {}, checklist: {},
      activePrinter: null, activeFilament: null, conn: {}, ui: {},
    }, d || {});
    return this.data;
  },
  save: debounce(async function () {
    try { await api.put('/api/data', JSON.stringify(DB.data)); } catch (e) { toast('Could not save: ' + e.message, 'err'); }
    publishSuite();
  }, 350),
  printer(id = this.data.activePrinter) { return this.data.printers.find((p) => p.id === id) || this.data.printers[0]; },
  filament(id = this.data.activeFilament) { return this.data.filaments.find((f) => f.id === id) || this.data.filaments[0]; },
  profile(pid, fid) {
    const p = pid || this.printer()?.id, f = fid || this.filament()?.id;
    const k = p + '|' + f;
    if (!this.data.profiles[k]) this.data.profiles[k] = { values: {}, updated: 0 };
    return this.data.profiles[k];
  },
  checklist(pid = this.printer()?.id) {
    if (!this.data.checklist[pid]) this.data.checklist[pid] = {};
    return this.data.checklist[pid];
  },
};

/* the active profile is shared with Makie and the rest of the suite */
const publishSuite = debounce(async () => {
  const p = DB.printer(), f = DB.filament();
  if (!p || !f) return;
  const v = DB.profile().values;
  const lw = round(p.nozzle * 1.125, 3);
  const doc = {
    app: 'Tunie', author: 'nrmluvdev', version: APP.version, schema: 1, updated: new Date().toISOString(),
    printer: {
      id: p.id, name: p.name, firmware: p.firmware, bedShape: p.bedShape, bed: [p.bedX, p.bedY, p.bedZ],
      nozzle: p.nozzle, filamentDiameter: p.filamentDia, lineWidth: lw, maxSpeed: p.maxSpeed, maxAccel: p.maxAccel,
      values: p.values || {},
    },
    filament: {
      id: f.id, brand: f.brand, type: f.type, colour: f.colour, colourName: f.colourName,
      temperature: v.temp ?? f.temp, bedTemperature: f.bed, flow: v.flow ?? 100, pressureAdvance: v.pa ?? null,
      retractLength: v.retractLen ?? null, retractSpeed: v.retractSpeed ?? null, maxVolumetricFlow: v.maxFlow ?? null,
    },
    fit: v.tolerance ? {
      // diametral clearances in mm (hole diameter minus shaft diameter)
      press: v.tolerance.press ?? null, slide: v.tolerance.slide ?? null, loose: v.tolerance.loose ?? null,
      holeCompensation: v.xyComp != null ? -v.xyComp * 2 : null, measured: v.tolerance.date || null,
    } : null,
    shrinkage: (v.scaleX || v.scaleY || v.scaleZ) ? { x: v.scaleX ?? 100, y: v.scaleY ?? 100, z: v.scaleZ ?? 100 } : null,
    xyCompensation: v.xyComp ?? null,
    maxOverhang: v.maxOverhang ?? null, maxBridge: v.maxBridge ?? null,
  };
  try { await api.post('/api/suite/active', doc); } catch (e) { console.warn('suite', e); }
}, 600);

/* ------------------------------------------------------------------ icons (24px stroke) */
const ICONS = {
  home: 'M3 10.5 12 3l9 7.5V20a1 1 0 0 1-1 1h-5v-6h-6v6H4a1 1 0 0 1-1-1z',
  flask: 'M9 3h6M10 3v6L4.5 18.5A1.7 1.7 0 0 0 6 21h12a1.7 1.7 0 0 0 1.5-2.5L14 9V3M7 15h10',
  printer: 'M5 4h14v4H5zM3 8h18v10H3zM7 18v3h10v-3M7 12h10',
  spool: 'M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18zm0 6a3 3 0 1 0 0 6 3 3 0 0 0 0-6zM12 3v6M12 15v6M3 12h6M15 12h6',
  history: 'M3 12a9 9 0 1 0 3-6.7M3 4v4h4M12 7v5l3 3',
  share: 'M12 3v12M7 8l5-5 5 5M5 14v5a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2v-5',
  plug: 'M9 2v6M15 2v6M6 8h12v4a6 6 0 0 1-12 0zM12 18v4',
  info: 'M12 22a10 10 0 1 0 0-20 10 10 0 0 0 0 20zM12 16v-5M12 8h.01',
  grid: 'M3 3h7v7H3zM14 3h7v7h-7zM3 14h7v7H3zM14 14h7v7h-7z',
  mesh: 'M3 17l4-6 4 3 4-7 6 9M3 21h18',
  mic: 'M12 2a3 3 0 0 0-3 3v6a3 3 0 0 0 6 0V5a3 3 0 0 0-3-3zM5 11a7 7 0 0 0 14 0M12 18v4',
  thermo: 'M14 14.8V4a2 2 0 0 0-4 0v10.8a4 4 0 1 0 4 0zM12 9v8',
  flame: 'M12 22c4 0 7-3 7-7 0-4-3-6-4-9-1 3-3 4-4 4 0-2 0-4-2-6-1 4-4 6-4 11 0 4 3 7 7 7z',
  gear: 'M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6zM19.4 15a1.7 1.7 0 0 0 .3 1.8l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-1.8-.3 1.7 1.7 0 0 0-1 1.5V21a2 2 0 1 1-4 0v-.1a1.7 1.7 0 0 0-1.1-1.5 1.7 1.7 0 0 0-1.8.3l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.7 1.7 0 0 0 .3-1.8 1.7 1.7 0 0 0-1.5-1H3a2 2 0 1 1 0-4h.1a1.7 1.7 0 0 0 1.5-1.1 1.7 1.7 0 0 0-.3-1.8l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.7 1.7 0 0 0 1.8.3H9a1.7 1.7 0 0 0 1-1.5V3a2 2 0 1 1 4 0v.1a1.7 1.7 0 0 0 1 1.5 1.7 1.7 0 0 0 1.8-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.7 1.7 0 0 0-.3 1.8V9a1.7 1.7 0 0 0 1.5 1H21a2 2 0 1 1 0 4h-.1a1.7 1.7 0 0 0-1.5 1z',
  ruler: 'M3 17 17 3l4 4L7 21zM7 13l2 2M10 10l2 2M13 7l2 2',
  drop: 'M12 3s7 7.5 7 12a7 7 0 0 1-14 0c0-4.5 7-12 7-12z',
  tower: 'M8 21V7l4-4 4 4v14M8 11h8M8 15h8M5 21h14',
  retract: 'M4 12h12M12 8l4 4-4 4M20 5v14',
  wave: 'M2 12c2-6 4-6 6 0s4 6 6 0 4-6 6 0 2 3 2 3',
  zap: 'M13 2 4 14h7l-1 8 9-12h-7z',
  layers: 'M12 3 2 8l10 5 10-5zM2 13l10 5 10-5M2 17l10 5 10-5',
  cube: 'M12 2 3 7v10l9 5 9-5V7zM3 7l9 5 9-5M12 12v10',
  angle: 'M4 20h16M4 20 16 6M9 20a6 6 0 0 0-1.6-4',
  bridge: 'M3 18h4V10h10v8h4M3 10h18M7 14h10',
  fit: 'M12 4a8 8 0 1 0 0 16 8 8 0 0 0 0-16zm0 4a4 4 0 1 0 0 8 4 4 0 0 0 0-8z',
  strings: 'M6 21V8M18 21V8M6 10c4 3 8 3 12 0M6 14c4 2 8 2 12 0',
  check: 'M4 12.5l5 5L20 6.5',
  copy: 'M9 9h11v11H9zM5 15H4V4h11v1',
  send: 'M22 2 11 13M22 2l-7 20-4-9-9-4z',
  save: 'M5 3h11l5 5v13H5zM8 3v6h8M8 21v-7h8v7',
  sd: 'M7 2h8l4 4v16H5V4a2 2 0 0 1 2-2zM9 6v3M12 6v3M15 6v3',
  download: 'M12 3v12M7 10l5 5 5-5M5 21h14',
  folder: 'M3 6a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z',
  plus: 'M12 5v14M5 12h14',
  trash: 'M4 7h16M10 11v6M14 11v6M5 7l1 13h12l1-13M9 7V4h6v3',
  edit: 'M4 20h4L19 9l-4-4L4 16zM14 6l4 4',
  arrowR: 'M5 12h14M13 6l6 6-6 6',
  arrowL: 'M19 12H5M11 6l-6 6 6 6',
  x: 'M6 6l12 12M18 6 6 18',
  play: 'M7 4v16l13-8z',
  stop: 'M6 6h12v12H6z',
  alert: 'M12 3 2 21h20zM12 10v5M12 18h.01',
  star: 'M12 3l2.8 5.8 6.2.9-4.5 4.4 1 6.3L12 17.5 6.5 20.4l1-6.3L3 9.7l6.2-.9z',
  bambu: 'M4 4h16v16H4zM9 4v16M15 4v16',
  link: 'M10 14a5 5 0 0 0 7 0l3-3a5 5 0 0 0-7-7l-1 1M14 10a5 5 0 0 0-7 0l-3 3a5 5 0 0 0 7 7l1-1',
  image: 'M3 5h18v14H3zM3 16l5-5 4 4 3-3 6 6M15 9h.01',
  refresh: 'M20 11a8 8 0 1 0-2.3 5.7M20 4v7h-7',
  wand: 'M15 4V2M15 10V8M11 6h2M17 6h2M4 20 14 10M18 14l-1 1',
};
function icon(name, size = 18, extra = '') {
  const d = ICONS[name] || ICONS.info;
  return `<svg width="${size}" height="${size}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" ${extra}><path d="${d}"/></svg>`;
}

/* ------------------------------------------------------------------ toasts & dialogs */
function toast(msg, kind = 'ok', ms = 3200) {
  const el = document.createElement('div');
  el.className = 'toast ' + kind;
  el.innerHTML = icon(kind === 'err' ? 'alert' : kind === 'info' ? 'info' : 'check', 18) + `<div>${esc(msg)}</div>`;
  $('#toasts').appendChild(el);
  setTimeout(() => { el.classList.add('out'); setTimeout(() => el.remove(), 320); }, ms);
}

function modal({ title, body, actions = [], wide = false, onOpen }) {
  return new Promise((resolve) => {
    const bg = document.createElement('div');
    bg.className = 'modal-bg';
    bg.innerHTML = `<div class="modal ${wide ? 'wide' : ''}"><div class="modal-h"><h2>${title}</h2><div class="spacer"></div><button class="btn ghost icon" data-x>${icon('x')}</button></div><div class="modal-b">${body}</div>
      <div class="modal-f">${actions.map((a, i) => `<button class="btn ${a.kind || ''}" data-i="${i}">${a.icon ? icon(a.icon, 16) : ''}${esc(a.label)}</button>`).join('')}</div></div>`;
    const close = (v) => { bg.remove(); resolve(v); };
    bg.addEventListener('mousedown', (e) => { if (e.target === bg) close(null); });
    bg.querySelector('[data-x]').onclick = () => close(null);
    bg.querySelectorAll('[data-i]').forEach((b) => b.onclick = async () => {
      const a = actions[+b.dataset.i];
      const v = a.value ? await a.value(bg) : a.label;
      if (v === false) return;
      close(v);
    });
    document.body.appendChild(bg);
    onOpen && onOpen(bg);
  });
}
const confirmBox = (title, text, ok = 'OK', kind = 'primary') =>
  modal({ title, body: `<p class="muted">${text}</p>`, actions: [{ label: 'Cancel', kind: 'ghost', value: () => null }, { label: ok, kind, value: () => true }] });

async function copyText(t) {
  try { await navigator.clipboard.writeText(t); toast('Copied to clipboard'); }
  catch {
    const ta = document.createElement('textarea'); ta.value = t; document.body.appendChild(ta); ta.select();
    document.execCommand('copy'); ta.remove(); toast('Copied to clipboard');
  }
}

/* code block with copy (and optional send to printer) */
function codeBlock(title, text, { send = false, lang = '' } = {}) {
  const id = 'cb' + uid();
  CODE_REG[id] = text;
  const hl = esc(text).replace(/(;[^\n]*)/g, '<span class="c">$1</span>').replace(/^(\[[^\]]+\])/gm, '<span class="k">$1</span>');
  return `<div class="code"><div class="code-h">${icon(lang === 'cfg' ? 'gear' : 'zap', 14)}${esc(title)}<span class="spacer"></span>
    ${send ? `<button class="btn sm" data-send="${id}" ${CONN.connected ? '' : 'disabled title="Connect to your printer first"'}>${icon('send', 14)}Send</button>` : ''}
    <button class="btn sm" data-copy="${id}">${icon('copy', 14)}Copy</button></div><pre>${hl}</pre></div>`;
}
const CODE_REG = {};
document.addEventListener('click', async (e) => {
  const c = e.target.closest('[data-copy]');
  if (c) return copyText(CODE_REG[c.dataset.copy] || '');
  const s = e.target.closest('[data-send]');
  if (s) {
    const lines = (CODE_REG[s.dataset.send] || '').split('\n').map((l) => l.replace(/;.*$/, '').trim()).filter(Boolean);
    s.disabled = true; const old = s.innerHTML; s.innerHTML = '<span class="spin"></span>Sending';
    try {
      const r = await sendLines(lines);
      toast(r.error ? 'Printer said: ' + r.error : 'Sent to printer', r.error ? 'err' : 'ok');
      if (r.lines && r.lines.length) document.dispatchEvent(new CustomEvent('printer-reply', { detail: r.lines }));
    } catch (err) { toast(err.message, 'err'); }
    s.disabled = false; s.innerHTML = old;
  }
  const ext = e.target.closest('a[data-ext]');
  if (ext) { e.preventDefault(); api.post('/api/open-url', { url: ext.getAttribute('href') }).catch(() => {}); }
});

/* ------------------------------------------------------------------ connection state */
const CONN = { connected: false, kind: '', status: {}, job: null };
async function sendLines(lines) { return api.post('/api/conn/send', { lines }); }

/* form helpers */
function fieldHTML(f, value) {
  const id = 'f_' + f.k;
  const help = f.help ? `<div class="help">${f.help}</div>` : '';
  if (f.type === 'select') {
    return `<div class="field"><label for="${id}">${esc(f.label)}</label><div class="inp"><select id="${id}" data-k="${f.k}">${f.options.map((o) => {
      const [v, l] = Array.isArray(o) ? o : [o, o];
      return `<option value="${esc(v)}" ${String(v) === String(value) ? 'selected' : ''}>${esc(l)}</option>`;
    }).join('')}</select></div>${help}</div>`;
  }
  if (f.type === 'check') {
    return `<div class="field"><label>&nbsp;</label><label class="check"><input type="checkbox" id="${id}" data-k="${f.k}" ${value ? 'checked' : ''}>${esc(f.label)}</label>${help}</div>`;
  }
  if (f.type === 'text') {
    return `<div class="field ${f.wide ? 'span2' : ''}"><label for="${id}">${esc(f.label)}</label><div class="inp"><input id="${id}" data-k="${f.k}" value="${esc(value ?? '')}" placeholder="${esc(f.ph || '')}"></div>${help}</div>`;
  }
  if (f.type === 'color') {
    return `<div class="field"><label for="${id}">${esc(f.label)}</label><div class="inp"><input type="color" id="${id}" data-k="${f.k}" value="${esc(value || '#888888')}"></div>${help}</div>`;
  }
  const v = value === undefined || value === null || (typeof value === 'number' && isNaN(value)) ? '' : value;
  return `<div class="field"><label for="${id}">${esc(f.label)}</label><div class="inp"><input type="number" id="${id}" data-k="${f.k}" value="${esc(v)}" step="${f.step ?? 'any'}" ${f.min !== undefined ? `min="${f.min}"` : ''} ${f.max !== undefined ? `max="${f.max}"` : ''} placeholder="${esc(f.ph || '')}">${f.unit ? `<span class="unit">${esc(f.unit)}</span>` : ''}</div>${help}</div>`;
}
function readFields(root) {
  const o = {};
  $$('[data-k]', root).forEach((el) => {
    const k = el.dataset.k;
    if (el.type === 'checkbox') o[k] = el.checked;
    else if (el.type === 'number') o[k] = el.value === '' ? NaN : num(el.value);
    else o[k] = el.value;
  });
  return o;
}

const APP = { version: '' };
