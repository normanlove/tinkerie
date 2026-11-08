/* Tracie — serial monitor + logic analyser on one timeline.
   Made by nrmluvdev. */
'use strict';

// ============================================================== basics
const TOKEN = window.TRACIE_TOKEN;
const $ = (s, r = document) => r.querySelector(s);
const $$ = (s, r = document) => Array.from(r.querySelectorAll(s));
const clamp = (v, a, b) => Math.max(a, Math.min(b, v));
const esc = s => s.replace(/[&<>"]/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]));
const debounce = (f, ms) => { let t; return (...a) => { clearTimeout(t); t = setTimeout(() => f(...a), ms); }; };
const DPR = () => window.devicePixelRatio || 1;

const CH_COLORS = ['#e5e7eb', '#d6a06b', '#fb7185', '#fb923c', '#facc15', '#4ade80', '#38bdf8', '#c084fc']; // resistor colour code (0 = white for contrast)
const PLOT_COLORS = ['#2dd4bf', '#818cf8', '#f472b6', '#fbbf24', '#38bdf8', '#a3e635', '#fb923c', '#c084fc', '#f87171', '#4ade80'];
const DEC_COLORS = ['#38bdf8', '#f472b6', '#a3e635', '#fb923c', '#c084fc', '#2dd4bf'];
const CMD_COLORS = ['#6366f1', '#2dd4bf', '#22c55e', '#f59e0b', '#ef4444', '#ec4899', '#a855f7', '#64748b'];
const KIND_COLORS = { 1: '#fbbf24', 2: '#2dd4bf', 3: '#64748b', 4: '#fb7185' };

async function api(path, body, raw) {
  const opt = { headers: { 'X-Tracie-Token': TOKEN } };
  if (raw !== undefined) { opt.method = 'POST'; opt.body = raw; opt.headers['Content-Type'] = 'application/octet-stream'; }
  else if (body !== undefined) { opt.method = 'POST'; opt.body = JSON.stringify(body); opt.headers['Content-Type'] = 'application/json'; }
  const r = await fetch(path, opt);
  const txt = await r.text();
  let j = null; try { j = JSON.parse(txt); } catch { }
  if (!r.ok) throw new Error((j && j.error) || txt || r.statusText);
  return j;
}

function toast(msg, kind = '', ms = 4200) {
  const el = document.createElement('div');
  el.className = 'toast ' + kind;
  el.innerHTML = `<i></i><div>${msg}</div>`;
  $('#toasts').appendChild(el);
  setTimeout(() => { el.classList.add('out'); setTimeout(() => el.remove(), 220); }, ms);
}

function fmtSec(s, digits = 3) {
  const a = Math.abs(s);
  const sign = s < 0 ? '−' : '';
  if (a === 0) return '0 s';
  if (a >= 60) return sign + Math.floor(a / 60) + 'm ' + (a % 60).toFixed(1) + 's';
  if (a >= 1) return sign + +a.toPrecision(digits + 1) + ' s';
  if (a >= 1e-3) return sign + +(a * 1e3).toPrecision(digits + 1) + ' ms';
  if (a >= 1e-6) return sign + +(a * 1e6).toPrecision(digits + 1) + ' µs';
  return sign + +(a * 1e9).toPrecision(digits + 1) + ' ns';
}
function fmtHz(f) {
  if (!isFinite(f) || f <= 0) return '—';
  if (f >= 1e6) return +(f / 1e6).toPrecision(5) + ' MHz';
  if (f >= 1e3) return +(f / 1e3).toPrecision(5) + ' kHz';
  return +f.toPrecision(5) + ' Hz';
}
function fmtBytes(n) {
  if (n >= 1 << 20) return (n / 1048576).toFixed(1) + ' MB';
  if (n >= 1024) return (n / 1024).toFixed(1) + ' kB';
  return n + ' B';
}
function fmtCount(n) {
  if (n >= 1e6) return +(n / 1e6).toPrecision(3) + 'M';
  if (n >= 1e3) return +(n / 1e3).toPrecision(3) + 'k';
  return String(n);
}
function clock(ms) {
  const d = new Date(ms);
  const p = (v, n = 2) => String(v).padStart(n, '0');
  return `${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}.${p(d.getMilliseconds(), 3)}`;
}
function b64bytes(s) {
  const bin = atob(s || '');
  const u = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) u[i] = bin.charCodeAt(i);
  return u;
}
function bytesB64(u) {
  let s = '';
  for (let i = 0; i < u.length; i++) s += String.fromCharCode(u[i]);
  return btoa(s);
}
function download(name, blob) {
  const a = document.createElement('a');
  a.href = URL.createObjectURL(blob);
  a.download = name;
  document.body.appendChild(a); a.click(); a.remove();
  setTimeout(() => URL.revokeObjectURL(a.href), 4000);
}
function lowerBound(arr, key, v) { // first index with arr[i][key] >= v
  let lo = 0, hi = arr.length;
  while (lo < hi) { const m = (lo + hi) >> 1; if (arr[m][key] < v) lo = m + 1; else hi = m; }
  return lo;
}
function lowerBoundNum(arr, v) {
  let lo = 0, hi = arr.length;
  while (lo < hi) { const m = (lo + hi) >> 1; if (arr[m] < v) lo = m + 1; else hi = m; }
  return lo;
}
function sizeCanvas(cv, w, h) {
  const d = DPR();
  const W = Math.max(1, Math.round(w * d)), H = Math.max(1, Math.round(h * d));
  if (cv.width !== W || cv.height !== H) { cv.width = W; cv.height = H; }
  cv.style.width = w + 'px'; cv.style.height = h + 'px';
  const ctx = cv.getContext('2d');
  ctx.setTransform(d, 0, 0, d, 0, 0);
  return ctx;
}

// ============================================================== config
const DEFAULT_CMDS = [
  { label: 'help', cmd: 'help', color: '#6366f1', eol: true },
  { label: 'ping', cmd: 'ping', color: '#2dd4bf', eol: true },
  { label: 'LED on', cmd: 'led on', color: '#22c55e', eol: true },
  { label: 'LED off', cmd: 'led off', color: '#64748b', eol: true },
];
const cfg = {
  tab: 'serial',
  serial: { port: '', baud: 115200, dataBits: 8, parity: 'none', stop: 1, dtr: true, rts: true, eol: '\n',
    ts: true, auto: true, echo: true, reconnect: true, plotWin: 10, plotStack: true, split: null },
  commands: DEFAULT_CMDS,
  history: [],
  logic: { device: 'demo', rate: 4e6, samples: 2e6, pre: 10, names: [0, 1, 2, 3, 4, 5, 6, 7].map(i => 'CH' + i),
    trig: Array(8).fill('x'), hidden: Array(8).fill(false), decoders: null, overlay: true, repeat: false },
};
function merge(dst, src) {
  for (const k in src) {
    if (src[k] && typeof src[k] === 'object' && !Array.isArray(src[k]) && dst[k] && typeof dst[k] === 'object' && !Array.isArray(dst[k])) merge(dst[k], src[k]);
    else if (src[k] !== undefined) dst[k] = src[k];
  }
}
const saveConfig = debounce(() => {
  fetch('/api/config', { method: 'PUT', headers: { 'X-Tracie-Token': TOKEN, 'Content-Type': 'application/json' }, body: JSON.stringify(cfg) }).catch(() => { });
}, 400);

// ============================================================== global time base
// All views share one origin (unix ms): the logic capture start if there is
// one, otherwise the first serial event.
const G = { origin: null, cap: null, table: [], rows: [], decCfg: [] };
function ensureOrigin(t) { if (G.origin == null) G.origin = t; }
function capOffset() { return G.cap ? (G.cap.t0 - G.origin) / 1000 : 0; }

// ============================================================== tabs
function setTab(name) {
  cfg.tab = name; saveConfig();
  $$('.tab').forEach(b => b.classList.toggle('active', b.dataset.tab === name));
  $$('.page').forEach(p => p.classList.toggle('active', p.id === 'page-' + name));
  moveTabInd();
  requestAnimationFrame(() => { waveLogic.resize(); waveTL.resize(); plot.dirty = true; });
}
function moveTabInd() {
  const a = $('.tab.active'); if (!a) return;
  const ind = $('#tabInd');
  ind.style.left = a.offsetLeft + 'px'; ind.style.width = a.offsetWidth + 'px';
}
$$('.tab').forEach(b => b.addEventListener('click', () => setTab(b.dataset.tab)));

// ============================================================== serial store
const SER = {
  lines: [],               // {t, dir, text}
  partial: { rx: null, tx: null },
  dec: { rx: new TextDecoder('utf-8'), tx: new TextDecoder('utf-8') },
  hexRows: [],             // {t, dir, bytes:[]}
  rx: 0, tx: 0,
  lastSeq: 0,
  state: { open: false, status: 'closed' },
  ports: [],
};

function handleSerialEvents(evs) {
  for (const ev of evs) {
    if (ev.q <= SER.lastSeq) continue;
    SER.lastSeq = ev.q;
    const bytes = b64bytes(ev.b);
    ensureOrigin(ev.t);
    if (ev.d === 'sys') { finishLine('rx', ev.t, true); addLine({ t: ev.t, dir: 'sys', text: new TextDecoder().decode(bytes) }); continue; }
    if (ev.d === 'rx') SER.rx += bytes.length; else SER.tx += bytes.length;
    addHex(ev.t, ev.d, bytes);
    const str = SER.dec[ev.d].decode(bytes, { stream: true });
    let p = SER.partial[ev.d];
    let start = 0;
    for (let i = 0; i < str.length; i++) {
      if (str.charCodeAt(i) === 10) {
        if (!p) p = { t: ev.t, text: '' };
        p.text += str.slice(start, i);
        SER.partial[ev.d] = p;
        finishLine(ev.d, ev.t);
        p = null; start = i + 1;
      }
    }
    if (start < str.length) {
      if (!p) p = { t: ev.t, text: '' };
      p.text += str.slice(start);
      if (p.text.length > 20000) { SER.partial[ev.d] = p; finishLine(ev.d, ev.t); p = null; }
    }
    SER.partial[ev.d] = p;
  }
  consoleUI.livePartial();
  statsDirty = true;
}

function finishLine(dir, t, force) {
  const p = SER.partial[dir];
  if (!p) return;
  SER.partial[dir] = null;
  if (force && !p.text) return;
  addLine({ t: p.t, dir: dir === 'tx' ? 'tx' : 'rx', text: p.text.replace(/\r+$/, '') });
}

function addLine(l) {
  SER.lines.push(l);
  if (SER.lines.length > 200000) SER.lines.splice(0, 50000);
  consoleUI.append(l);
  if (l.dir === 'rx') plot.ingest(l);
  tlList.dirty = true;
  waveTL.dirty = true;
  if (cfg.logic.overlay) waveLogic.dirty = true;
}

function addHex(t, dir, bytes) {
  let row = SER.hexRows[SER.hexRows.length - 1];
  for (let i = 0; i < bytes.length; i++) {
    if (!row || row.dir !== dir || row.bytes.length >= 16 || t - row.t > 50 && row.bytes.length && i === 0) {
      row = { t, dir, bytes: [] };
      SER.hexRows.push(row);
    }
    row.bytes.push(bytes[i]);
  }
  if (SER.hexRows.length > 40000) SER.hexRows.splice(0, 10000);
  consoleUI.hexDirty = true;
}

// ============================================================== console
const ANSI_RE = /\x1b\[([0-9;]*)([A-Za-z])/g;
function ansiHtml(text, hl) {
  // Render ESP-IDF style colours, drop other escapes and control chars.
  let out = '', last = 0, open = false, m;
  const clean = s => {
    s = s.replace(/[\x00-\x08\x0b-\x1a\x1c-\x1f\x7f]/g, '·');
    s = esc(s);
    if (hl) s = s.replace(hl, x => `<mark>${x}</mark>`);
    return s;
  };
  ANSI_RE.lastIndex = 0;
  while ((m = ANSI_RE.exec(text))) {
    out += clean(text.slice(last, m.index));
    last = ANSI_RE.lastIndex;
    if (m[2] !== 'm') continue;
    const codes = m[1].split(';').map(Number);
    if (open) { out += '</span>'; open = false; }
    const cls = [];
    for (const c of codes) {
      if (c === 1) cls.push('a-b');
      else if (c >= 30 && c <= 37) cls.push('a' + c);
      else if (c >= 90 && c <= 97) cls.push('a' + (c - 60));
    }
    if (cls.length) { out += `<span class="${cls.join(' ')}">`; open = true; }
  }
  out += clean(text.slice(last).replace(/\x1b/g, ''));
  if (open) out += '</span>';
  return out;
}

const consoleUI = {
  el: $('#console'), hexEl: $('#hexview'), queue: [], liveEl: null, hexDirty: false, hexShown: 0,
  mode: 'text', filter: '', filterRe: null, MAX: 3000,
  lineHTML(l) {
    const cls = l.dir === 'tx' ? 'out' : l.dir === 'sys' ? 'sys' : '';
    return `<div class="ln ${cls}"><span class="ts">${clock(l.t)}</span><span class="tx">${ansiHtml(l.text, this.filterRe) || '&nbsp;'}</span></div>`;
  },
  visible(l) {
    if (l.dir === 'tx' && !cfg.serial.echo) return false;
    if (this.filter && !l.text.toLowerCase().includes(this.filter)) return false;
    return true;
  },
  append(l) { if (this.visible(l)) this.queue.push(l); },
  livePartial() { this.liveDirty = true; },
  flush() {
    const el = this.el;
    const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 40;
    if (this.queue.length) {
      if (this.liveEl) { this.liveEl.remove(); this.liveEl = null; }
      let q = this.queue; this.queue = [];
      if (q.length > this.MAX) q = q.slice(-this.MAX);
      el.insertAdjacentHTML('beforeend', q.map(l => this.lineHTML(l)).join(''));
      let extra = el.childElementCount - this.MAX;
      while (extra-- > 0) el.firstElementChild.remove();
      this.liveDirty = true;
    }
    if (this.liveDirty) {
      this.liveDirty = false;
      const p = SER.partial.rx;
      if (p && p.text && !this.filter) {
        if (!this.liveEl) { this.liveEl = document.createElement('div'); this.liveEl.className = 'ln live'; el.appendChild(this.liveEl); }
        this.liveEl.innerHTML = `<span class="ts">${clock(p.t)}</span><span class="tx">${ansiHtml(p.text)}</span>`;
      } else if (this.liveEl) { this.liveEl.remove(); this.liveEl = null; }
    }
    if (cfg.serial.auto && atBottom || cfg.serial.auto && this.forceBottom) { el.scrollTop = el.scrollHeight; this.forceBottom = false; }
    $('#consoleEmpty').classList.toggle('hidden', SER.lines.length > 0 || !!SER.partial.rx || SER.hexRows.length > 0);
    if (this.mode === 'hex' && this.hexDirty) this.renderHex();
  },
  rebuild() {
    this.el.innerHTML = ''; this.liveEl = null; this.queue = [];
    const out = [];
    for (let i = SER.lines.length - 1; i >= 0 && out.length < this.MAX; i--) if (this.visible(SER.lines[i])) out.push(SER.lines[i]);
    out.reverse();
    this.queue = out;
    this.forceBottom = true;
    this.liveDirty = true;
    this.flush();
  },
  rowHTML(r) {
    let hx = '', as = '';
    for (let i = 0; i < 16; i++) {
      if (i < r.bytes.length) {
        const b = r.bytes[i];
        const h = b.toString(16).padStart(2, '0').toUpperCase();
        hx += (b === 10 || b === 13 ? `<b>${h}</b>` : h) + (i === 7 ? '  ' : ' ');
        as += b >= 32 && b < 127 ? esc(String.fromCharCode(b)) : '·';
      } else hx += '   ' + (i === 7 ? ' ' : '');
    }
    return `<div class="hrow ${r.dir === 'tx' ? 'out' : ''}"><span class="ts">${clock(r.t)}</span><span class="dir">${r.dir === 'tx' ? 'TX' : 'RX'}</span><span class="hx">${hx}</span><span class="as">${as}</span></div>`;
  },
  renderHex() {
    this.hexDirty = false;
    const el = this.hexEl;
    const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 40;
    const rows = SER.hexRows.filter(r => r.dir === 'rx' || cfg.serial.echo).slice(-1500);
    el.innerHTML = rows.map(r => this.rowHTML(r)).join('');
    if (cfg.serial.auto && (atBottom || this.forceBottomHex)) { el.scrollTop = el.scrollHeight; this.forceBottomHex = false; }
  },
  setMode(m) {
    this.mode = m;
    $$('#segView button').forEach(b => b.classList.toggle('active', b.dataset.v === m));
    this.el.classList.toggle('hidden', m !== 'text');
    this.hexEl.classList.toggle('hidden', m !== 'hex');
    if (m === 'hex') { this.forceBottomHex = true; this.renderHex(); }
  },
  clear() {
    SER.lines = []; SER.hexRows = []; SER.partial = { rx: null, tx: null }; SER.rx = SER.tx = 0;
    this.el.innerHTML = ''; this.hexEl.innerHTML = ''; this.liveEl = null; this.queue = [];
    if (!G.cap) G.origin = null;
    tlList.dirty = true; waveTL.dirty = true; waveLogic.dirty = true; statsDirty = true;
    api('/api/serial/clear', {}).catch(() => { });
  },
};

// ============================================================== serial controls
const BAUDS = [9600, 19200, 38400, 57600, 115200, 230400, 460800, 921600];
function renderBauds() {
  $('#baudPills').innerHTML = BAUDS.map(b => `<button data-b="${b}" class="${cfg.serial.baud === b ? 'active' : ''}">${b >= 1000 ? (b / 1000) + 'k' : b}</button>`).join('');
  $('#baudCustom').value = BAUDS.includes(cfg.serial.baud) ? '' : cfg.serial.baud;
}
$('#baudPills').addEventListener('click', e => {
  const b = e.target.closest('button'); if (!b) return;
  setBaud(+b.dataset.b);
});
$('#baudCustom').addEventListener('change', e => { const v = parseInt(e.target.value, 10); if (v > 0) setBaud(v); });
function setBaud(b) {
  cfg.serial.baud = b; saveConfig(); renderBauds();
  if (SER.state.open) connect(true);
}

const USB_ICON = '<svg viewBox="0 0 24 24"><path d="M12 3v15M9 6l3-3 3 3M7 11v2l5 3M17 9v3l-5 3"/><circle cx="12" cy="19.5" r="1.5"/><rect x="15.5" y="7" width="3" height="2.5"/><circle cx="7" cy="10" r="1.3"/></svg>';
const DEMO_ICON = '<svg viewBox="0 0 24 24"><path d="M12 3l1.8 4.7L18.5 9.5l-4.7 1.8L12 16l-1.8-4.7L5.5 9.5l4.7-1.8z"/><path d="M19 15l.8 2 2 .8-2 .8-.8 2-.8-2-2-.8 2-.8z"/></svg>';
async function refreshPorts(quiet) {
  try {
    const ports = await api('/api/serial/ports');
    const sig = JSON.stringify(ports);
    if (sig === SER.portSig && quiet) return;
    const before = new Set(SER.ports.map(p => p.name));
    SER.ports = ports; SER.portSig = sig;
    if (quiet) {
      const added = ports.filter(p => !before.has(p.name) && !p.demo);
      if (added.length && !SER.state.open) { cfg.serial.port = added[0].name; toast(`<b>${esc(added[0].name)}</b> plugged in${added[0].chip ? ' — ' + esc(added[0].chip) : ''}`); }
    }
    if (!ports.find(p => p.name === cfg.serial.port)) {
      const real = ports.find(p => !p.demo);
      cfg.serial.port = (real || ports[0]).name;
    }
    renderPorts();
  } catch (e) { if (!quiet) toast('Could not list ports: ' + esc(e.message), 'err'); }
}
function renderPorts() {
  const el = $('#portList');
  el.innerHTML = SER.ports.map(p => `
    <div class="port ${p.demo ? 'demo' : ''} ${p.name === cfg.serial.port ? 'sel' : ''}" data-p="${esc(p.name)}">
      <div class="pi">${p.demo ? DEMO_ICON : USB_ICON}</div>
      <div class="pt"><div class="pn">${esc(p.name)}</div><div class="pd" title="${esc(p.desc || '')}">${esc(p.desc || 'Serial port')}</div></div>
      ${p.chip ? `<span class="badge ${p.demo ? '' : 'ok'}">${esc(p.chip)}</span>` : ''}
    </div>`).join('') || '<div class="empty-list">No serial ports found — plug a board in.</div>';
}
$('#portList').addEventListener('click', e => {
  const p = e.target.closest('.port'); if (!p) return;
  if (cfg.serial.port === p.dataset.p) return;
  cfg.serial.port = p.dataset.p; saveConfig(); renderPorts();
  if (SER.state.open) connect(true);
});
$('#btnRefreshPorts').addEventListener('click', () => refreshPorts(false));

async function connect(reopen) {
  const s = cfg.serial;
  try {
    await api('/api/serial/open', { port: s.port, reconnect: s.reconnect,
      config: { baud: s.baud, dataBits: +s.dataBits, parity: s.parity, stopBits: +s.stop, dtr: s.dtr, rts: s.rts } });
    if (!reopen) consoleUI.forceBottom = true;
  } catch (e) { toast(esc(e.message), 'err', 6000); }
}
$('#btnConnect').addEventListener('click', async () => {
  if (SER.state.open) { await api('/api/serial/close', {}).catch(() => { }); return; }
  if (!cfg.serial.port) { toast('Pick a port first', 'warn'); return; }
  connect(false);
});
function applySerialState(st) {
  SER.state = st;
  const b = $('#btnConnect');
  b.classList.toggle('connected', st.open);
  b.querySelector('span').textContent = st.open ? (st.status === 'reconnecting' ? 'Waiting for board… (Disconnect)' : 'Disconnect') : 'Connect';
  const chip = $('#chipSerial');
  const dot = chip.querySelector('.dot');
  dot.className = 'dot ' + (st.status === 'open' ? 'on' : st.status === 'reconnecting' ? 'busy' : st.status === 'error' ? 'err' : '');
  chip.querySelector('.lbl').textContent = st.open ? `${st.port} · ${st.config.baud}` : st.status === 'error' ? 'Serial error' : 'Serial idle';
  $('#swLog').checked = st.logging;
  $('#logPath').textContent = st.logging ? st.logPath : 'Saves to Documents\\Tracie';
}
['DTR', 'RTS'].forEach(k => $('#sw' + k).addEventListener('change', e => {
  cfg.serial[k.toLowerCase()] = e.target.checked; saveConfig();
  if (SER.state.open) api('/api/serial/lines', { dtr: cfg.serial.dtr, rts: cfg.serial.rts }).catch(er => toast(esc(er.message), 'err'));
}));
$('#btnReset').addEventListener('click', () => api('/api/serial/reset', {}).catch(e => toast(esc(e.message), 'err')));
$('#swReconnect').addEventListener('change', e => { cfg.serial.reconnect = e.target.checked; saveConfig(); api('/api/serial/reconnect', { on: e.target.checked }).catch(() => { }); });
$('#selDataBits').addEventListener('change', e => { cfg.serial.dataBits = +e.target.value; saveConfig(); if (SER.state.open) connect(true); });
$('#selParity').addEventListener('change', e => { cfg.serial.parity = e.target.value; saveConfig(); if (SER.state.open) connect(true); });
$('#selStop').addEventListener('change', e => { cfg.serial.stop = +e.target.value; saveConfig(); if (SER.state.open) connect(true); });
$('#swLog').addEventListener('change', async e => {
  try {
    const st = await api('/api/serial/log', { on: e.target.checked });
    if (st.logging) toast('Recording to <span class="mono">' + esc(st.logPath) + '</span>');
  } catch (er) { toast(esc(er.message), 'err'); e.target.checked = false; }
});
$('#btnOpenLogs').addEventListener('click', () => api('/api/openfolder', { which: 'logs' }));
$('#btnSaveTxt').addEventListener('click', () => {
  const txt = SER.lines.map(l => `${clock(l.t)}  ${l.dir === 'tx' ? '-> ' : l.dir === 'sys' ? '** ' : ''}${l.text.replace(ANSI_RE, '')}`).join('\r\n');
  download(`serial_${new Date().toISOString().slice(0, 19).replace(/[:T]/g, '-')}.txt`, new Blob([txt], { type: 'text/plain' }));
});
$('#swTs').addEventListener('change', e => { cfg.serial.ts = e.target.checked; saveConfig(); $('#console').classList.toggle('no-ts', !e.target.checked); });
$('#swAuto').addEventListener('change', e => { cfg.serial.auto = e.target.checked; saveConfig(); });
$('#swEcho').addEventListener('change', e => { cfg.serial.echo = e.target.checked; saveConfig(); consoleUI.rebuild(); consoleUI.hexDirty = true; });
$('#segView').addEventListener('click', e => { const b = e.target.closest('button'); if (b) consoleUI.setMode(b.dataset.v); });
$('#consoleFilter').addEventListener('input', debounce(e => {
  const v = e.target.value.trim().toLowerCase();
  consoleUI.filter = v;
  consoleUI.filterRe = v ? new RegExp(esc(v).replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'gi') : null;
  consoleUI.rebuild();
}, 150));
$('#btnClear').addEventListener('click', () => consoleUI.clear());

// ---- sending
function unescapeCmd(s) {
  return s.replace(/\\(x[0-9a-fA-F]{2}|n|r|t|0|\\)/g, (m, g) => {
    if (g[0] === 'x') return String.fromCharCode(parseInt(g.slice(1), 16));
    return { n: '\n', r: '\r', t: '\t', '0': '\0', '\\': '\\' }[g];
  });
}
function strBytes(s) {
  // Keep \xNN escapes as raw bytes, encode everything else as UTF-8.
  const out = [];
  const enc = new TextEncoder();
  for (const ch of s) {
    const c = ch.codePointAt(0);
    if (c < 256 && c >= 128) out.push(c); else out.push(...enc.encode(ch));
  }
  return new Uint8Array(out);
}
async function sendText(text, addEol = true) {
  if (!SER.state.open) { toast('Connect to a port first', 'warn'); return false; }
  const data = unescapeCmd(text) + (addEol ? cfg.serial.eol : '');
  try { await api('/api/serial/send', { data: bytesB64(strBytes(data)) }); return true; }
  catch (e) { toast(esc(e.message), 'err'); return false; }
}
let histIdx = -1;
$('#sendBar').addEventListener('submit', async e => {
  e.preventDefault();
  const inp = $('#sendInput');
  const v = inp.value;
  if (!v && !cfg.serial.eol) return;
  if (await sendText(v)) {
    if (v && cfg.history[cfg.history.length - 1] !== v) { cfg.history.push(v); if (cfg.history.length > 100) cfg.history.shift(); saveConfig(); }
    inp.value = ''; histIdx = -1;
  }
});
$('#sendInput').addEventListener('keydown', e => {
  const h = cfg.history;
  if (e.key === 'ArrowUp' && h.length) { histIdx = histIdx < 0 ? h.length - 1 : Math.max(0, histIdx - 1); e.target.value = h[histIdx]; e.preventDefault(); }
  if (e.key === 'ArrowDown' && histIdx >= 0) { histIdx++; if (histIdx >= h.length) { histIdx = -1; e.target.value = ''; } else e.target.value = h[histIdx]; e.preventDefault(); }
});
$('#selEol').addEventListener('change', e => { cfg.serial.eol = unescapeCmd(e.target.value); saveConfig(); });

// ---- command buttons
let editingCmd = -1, cmdColor = CMD_COLORS[0];
function renderCmds() {
  $('#cmdGrid').innerHTML = cfg.commands.map((c, i) =>
    `<button class="cmd" data-i="${i}" style="--c:${c.color}" title="${esc(c.cmd)}">${esc(c.label || c.cmd)}</button>`).join('');
}
$('#cmdGrid').addEventListener('click', async e => {
  const b = e.target.closest('.cmd'); if (!b) return;
  const c = cfg.commands[+b.dataset.i];
  b.classList.remove('flash'); void b.offsetWidth; b.classList.add('flash');
  sendText(c.cmd, c.eol !== false);
});
$('#cmdGrid').addEventListener('contextmenu', e => {
  const b = e.target.closest('.cmd'); if (!b) return;
  e.preventDefault(); openCmd(+b.dataset.i);
});
$('#btnAddCmd').addEventListener('click', () => openCmd(-1));
function openCmd(i) {
  editingCmd = i;
  const c = i >= 0 ? cfg.commands[i] : { label: '', cmd: '', color: CMD_COLORS[cfg.commands.length % CMD_COLORS.length], eol: true };
  $('#cmdModalTitle').textContent = i >= 0 ? 'Edit command' : 'New command';
  $('#cmdLabel').value = c.label; $('#cmdText').value = c.cmd; $('#cmdEol').checked = c.eol !== false;
  cmdColor = c.color;
  $('#cmdColors').innerHTML = CMD_COLORS.map(col => `<button style="background:${col}" data-c="${col}" class="${col === cmdColor ? 'active' : ''}"></button>`).join('');
  $('#cmdDelete').classList.toggle('hidden', i < 0);
  showModal('#cmdModal');
  setTimeout(() => $('#cmdLabel').focus(), 50);
}
$('#cmdColors').addEventListener('click', e => {
  const b = e.target.closest('button'); if (!b) return;
  cmdColor = b.dataset.c;
  $$('#cmdColors button').forEach(x => x.classList.toggle('active', x === b));
});
$('#cmdSave').addEventListener('click', () => {
  const c = { label: $('#cmdLabel').value.trim(), cmd: $('#cmdText').value, color: cmdColor, eol: $('#cmdEol').checked };
  if (!c.cmd && !c.label) { hideModals(); return; }
  if (!c.label) c.label = c.cmd;
  if (editingCmd >= 0) cfg.commands[editingCmd] = c; else cfg.commands.push(c);
  saveConfig(); renderCmds(); hideModals();
});
$('#cmdDelete').addEventListener('click', () => {
  if (editingCmd >= 0) cfg.commands.splice(editingCmd, 1);
  saveConfig(); renderCmds(); hideModals();
});

// ---- modals
function showModal(sel) { $(sel).classList.remove('hidden'); }
function hideModals() { $$('.modal').forEach(m => m.classList.add('hidden')); }
$$('.modal').forEach(m => m.addEventListener('mousedown', e => { if (e.target === m || e.target.closest('[data-close]')) hideModals(); }));
$('#btnHelp').addEventListener('click', () => showModal('#helpModal'));

// ============================================================== plotter
const NUM_RE = /^[-+]?(\d+\.?\d*|\.\d+)([eE][-+]?\d+)?%?$/;
function parsePlot(text) {
  text = text.replace(ANSI_RE, '').trim();
  if (!text) return null;
  const toks = text.split(/[\s,;|\t]+/).filter(Boolean);
  const out = [];
  let label = null, unnamed = 0;
  for (let tok of toks) {
    let name = null;
    if (/[:=]$/.test(tok)) { label = tok.slice(0, -1); continue; }
    const m = tok.match(/^([^:=]+)[:=](.+)$/);
    if (m) { name = m[1]; tok = m[2]; }
    else if (label) { name = label; }
    label = null;
    if (!NUM_RE.test(tok)) return null; // a word that isn't data: not a plot line
    const v = parseFloat(tok);
    if (!isFinite(v)) return null;
    if (!name) name = 'v' + (++unnamed);
    out.push([name, v]);
  }
  return out.length && out.length <= 16 ? out : null;
}

const plot = {
  cv: $('#plotCanvas'), series: new Map(), order: [], paused: false, frozen: 0, hover: null, dirty: true, lastLegend: 0,
  ingest(l) {
    const vals = parsePlot(l.text);
    if (!vals) return;
    for (const [n, v] of vals) {
      let s = this.series.get(n);
      if (!s) {
        if (this.series.size >= 12) continue;
        s = { name: n, color: PLOT_COLORS[this.series.size % PLOT_COLORS.length], t: [], v: [], on: true, last: v };
        this.series.set(n, s); this.order.push(n);
        this.legendDirty = true;
      }
      s.t.push(l.t); s.v.push(v); s.last = v;
      if (s.t.length > 300000) { s.t.splice(0, 60000); s.v.splice(0, 60000); }
    }
    this.dirty = true;
  },
  clear() { this.series.clear(); this.order = []; this.legendDirty = true; this.dirty = true; waveTL.lanesDirty = true; },
  renderLegend() {
    this.legendDirty = false;
    $('#plotLegend').innerHTML = this.order.map(n => {
      const s = this.series.get(n);
      return `<div class="leg ${s.on ? '' : 'off'}" data-n="${esc(n)}"><i style="background:${s.color}"></i><span class="n">${esc(n)}</span><span class="v" style="color:${s.color}">${fmtVal(s.last)}</span></div>`;
    }).join('');
    $('#plotEmpty').classList.toggle('hidden', this.order.length > 0);
    waveTL.lanesDirty = true;
  },
  updateLegendValues() {
    $$('#plotLegend .leg').forEach(el => {
      const s = this.series.get(el.dataset.n);
      if (s) el.querySelector('.v').textContent = fmtVal(s.last);
    });
  },
  draw() {
    const wrap = this.cv.parentElement;
    const W = wrap.clientWidth, H = wrap.clientHeight;
    if (W < 10 || H < 10) return;
    const ctx = sizeCanvas(this.cv, W, H);
    ctx.clearRect(0, 0, W, H);
    const L = 56, R = 14, T = 12, B = 24;
    const pw = W - L - R, ph = H - T - B;
    const win = cfg.serial.plotWin * 1000;
    const t1 = this.paused ? this.frozen : Date.now();
    const t0 = t1 - win;
    let lo = Infinity, hi = -Infinity;
    const vis = [];
    for (const n of this.order) {
      const s = this.series.get(n);
      if (!s.on) continue;
      const a = Math.max(0, lowerBoundNum(s.t, t0) - 1), b = Math.min(s.t.length, lowerBoundNum(s.t, t1) + 1);
      for (let i = a; i < b; i++) { const v = s.v[i]; if (v < lo) lo = v; if (v > hi) hi = v; }
      vis.push([s, a, b]);
    }
    if (!isFinite(lo)) { lo = 0; hi = 1; }
    if (hi - lo < 1e-9) { lo -= 1; hi += 1; }
    const pad = (hi - lo) * 0.1; lo -= pad; hi += pad;
    const X = t => L + (t - t0) / win * pw;
    const Y = v => T + ph - (v - lo) / (hi - lo) * ph;
    const stacked = cfg.serial.plotStack && vis.length > 1;
    const bandH = ph / Math.max(1, vis.length);
    const Ys = vis.map(([s, a, b], i) => {
      if (!stacked) return Y;
      let l = Infinity, h = -Infinity;
      for (let k = a; k < b; k++) { if (s.v[k] < l) l = s.v[k]; if (s.v[k] > h) h = s.v[k]; }
      if (!isFinite(l)) { l = 0; h = 1; }
      if (h - l < 1e-9) { l -= 1; h += 1; }
      const top = T + i * bandH + 8, hh = bandH - 16;
      const f = v => top + hh - (v - l) / (h - l) * hh;
      f.lo = l; f.hi = h; f.top = top; f.hh = hh;
      return f;
    });

    // grid
    ctx.font = '11px ' + getComputedStyle(document.body).getPropertyValue('--mono');
    ctx.textBaseline = 'middle';
    const yt = stacked ? [] : niceTicks(lo, hi, Math.max(3, Math.floor(ph / 42)));
    if (stacked) Ys.forEach((f, i) => {
      ctx.fillStyle = 'rgba(255,255,255,.018)'; if (i % 2) ctx.fillRect(L, T + i * bandH, pw, bandH);
      ctx.fillStyle = '#5f6c80'; ctx.textAlign = 'right';
      ctx.fillText(fmtVal(f.hi), L - 8, f.top + 4); ctx.fillText(fmtVal(f.lo), L - 8, f.top + f.hh - 4);
    });
    for (const v of yt) {
      const y = Math.round(Y(v)) + .5;
      ctx.strokeStyle = 'rgba(255,255,255,.05)'; ctx.beginPath(); ctx.moveTo(L, y); ctx.lineTo(W - R, y); ctx.stroke();
      ctx.fillStyle = '#5f6c80'; ctx.textAlign = 'right'; ctx.fillText(fmtVal(v), L - 8, y);
    }
    const xstep = niceStep(win / Math.max(2, pw / 110));
    ctx.textAlign = 'center';
    for (let t = Math.ceil((t0 - t1) / xstep) * xstep; t <= 0; t += xstep) {
      const x = Math.round(X(t1 + t)) + .5;
      ctx.strokeStyle = 'rgba(255,255,255,.04)'; ctx.beginPath(); ctx.moveTo(x, T); ctx.lineTo(x, T + ph); ctx.stroke();
      ctx.fillStyle = '#5f6c80'; ctx.fillText(t === 0 ? 'now' : fmtSec(t / 1000, 2), x, H - 11);
    }

    // series
    ctx.save();
    ctx.beginPath(); ctx.rect(L, T - 2, pw, ph + 4); ctx.clip();
    vis.forEach(([s, a, b], si) => {
      const Y = Ys[si];
      ctx.lineWidth = 1.8; ctx.strokeStyle = s.color; ctx.lineJoin = 'round';
      ctx.beginPath();
      const n = b - a;
      if (n > pw * 3) {
        // min/max decimation per pixel column
        let col = -1, mn = 0, mx = 0;
        for (let i = a; i < b; i++) {
          const x = Math.floor(X(s.t[i])), v = s.v[i];
          if (x !== col) {
            if (col >= 0) { ctx.lineTo(col, Y(mn)); ctx.lineTo(col, Y(mx)); }
            col = x; mn = mx = v;
            if (i === a) ctx.moveTo(x, Y(v));
          } else { if (v < mn) mn = v; if (v > mx) mx = v; }
        }
      } else {
        for (let i = a; i < b; i++) { const x = X(s.t[i]), y = Y(s.v[i]); i === a ? ctx.moveTo(x, y) : ctx.lineTo(x, y); }
      }
      ctx.shadowColor = s.color; ctx.shadowBlur = 8;
      ctx.stroke();
      ctx.shadowBlur = 0;
      if ((si === 0 || stacked) && n > 1 && n <= pw * 3) {
        // soft area fill under the first series
        const base = stacked ? Y.top + Y.hh : T + ph;
        ctx.lineTo(X(s.t[b - 1]), base); ctx.lineTo(X(s.t[a]), base); ctx.closePath();
        const g = ctx.createLinearGradient(0, T, 0, T + ph);
        g.addColorStop(0, s.color + '33'); g.addColorStop(1, s.color + '00');
        ctx.fillStyle = g; ctx.fill();
      }
      if (n > 0 && !this.paused) {
        const x = X(s.t[b - 1]), y = Y(s.v[b - 1]);
        ctx.fillStyle = s.color; ctx.beginPath(); ctx.arc(x, y, 3.2, 0, 7); ctx.fill();
        ctx.fillStyle = s.color + '40'; ctx.beginPath(); ctx.arc(x, y, 7, 0, 7); ctx.fill();
      }
    });
    ctx.restore();

    // hover readout
    if (this.hover != null && this.hover > L && vis.length) {
      const t = t0 + (this.hover - L) / pw * win;
      ctx.strokeStyle = 'rgba(255,255,255,.25)'; ctx.setLineDash([3, 4]);
      ctx.beginPath(); ctx.moveTo(this.hover + .5, T); ctx.lineTo(this.hover + .5, T + ph); ctx.stroke(); ctx.setLineDash([]);
      const rows = [];
      vis.forEach(([s], si) => {
        const i = clamp(lowerBoundNum(s.t, t), 0, s.t.length - 1);
        if (s.t.length) { rows.push([s, s.v[i]]); ctx.fillStyle = s.color; ctx.beginPath(); ctx.arc(X(s.t[i]), Ys[si](s.v[i]), 3.5, 0, 7); ctx.fill(); }
      });
      const bw = 150, bh = 22 + rows.length * 18;
      let bx = this.hover + 12; if (bx + bw > W - 4) bx = this.hover - bw - 12;
      ctx.fillStyle = 'rgba(15,20,29,.94)'; ctx.strokeStyle = 'rgba(255,255,255,.12)';
      roundRect(ctx, bx, T + 6, bw, bh, 9); ctx.fill(); ctx.stroke();
      ctx.textAlign = 'left'; ctx.fillStyle = '#9ba8ba'; ctx.fillText(clock(t), bx + 10, T + 18);
      rows.forEach(([s, v], k) => {
        ctx.fillStyle = s.color; ctx.fillRect(bx + 10, T + 33 + k * 18, 8, 8);
        ctx.fillStyle = '#9ba8ba'; ctx.fillText(s.name, bx + 24, T + 37 + k * 18);
        ctx.fillStyle = '#fff'; ctx.textAlign = 'right'; ctx.fillText(fmtVal(v), bx + bw - 10, T + 37 + k * 18); ctx.textAlign = 'left';
      });
    }
  },
};
function fmtVal(v) {
  if (v == null || !isFinite(v)) return '—';
  const a = Math.abs(v);
  if (a !== 0 && (a >= 1e6 || a < 1e-3)) return v.toExponential(2);
  if (Number.isInteger(v)) return String(v);
  return +v.toFixed(a >= 100 ? 1 : a >= 10 ? 2 : 3) + '';
}
function niceStep(raw) {
  const p = Math.pow(10, Math.floor(Math.log10(raw)));
  const f = raw / p;
  return (f <= 1 ? 1 : f <= 2 ? 2 : f <= 5 ? 5 : 10) * p;
}
function niceTicks(lo, hi, n) {
  const st = niceStep((hi - lo) / n);
  const out = [];
  for (let v = Math.ceil(lo / st) * st; v <= hi + st * 1e-9; v += st) out.push(Math.abs(v) < st * 1e-9 ? 0 : v);
  return out;
}
function roundRect(ctx, x, y, w, h, r) {
  ctx.beginPath(); ctx.moveTo(x + r, y); ctx.arcTo(x + w, y, x + w, y + h, r); ctx.arcTo(x + w, y + h, x, y + h, r);
  ctx.arcTo(x, y + h, x, y, r); ctx.arcTo(x, y, x + w, y, r); ctx.closePath();
}
plot.cv.addEventListener('mousemove', e => { plot.hover = e.offsetX; plot.dirty = true; });
plot.cv.addEventListener('mouseleave', () => { plot.hover = null; plot.dirty = true; });
$('#plotLegend').addEventListener('click', e => {
  const l = e.target.closest('.leg'); if (!l) return;
  const s = plot.series.get(l.dataset.n); s.on = !s.on; l.classList.toggle('off', !s.on); plot.dirty = true;
});
$('#segWindow').addEventListener('click', e => {
  const b = e.target.closest('button'); if (!b) return;
  cfg.serial.plotWin = +b.dataset.w; saveConfig();
  $$('#segWindow button').forEach(x => x.classList.toggle('active', x === b)); plot.dirty = true;
});
$('#btnPlotPause').addEventListener('click', e => {
  plot.paused = !plot.paused; plot.frozen = Date.now();
  const b = e.currentTarget; b.classList.toggle('active', plot.paused);
  b.innerHTML = plot.paused ? '<svg viewBox="0 0 24 24"><path d="M7 4l13 8-13 8z"/></svg>' : '<svg viewBox="0 0 24 24"><path d="M8 5v14M16 5v14"/></svg>';
  b.title = plot.paused ? 'Resume plot' : 'Pause plot';
});
$('#btnPlotClear').addEventListener('click', () => plot.clear());
$('#btnPlotStack').addEventListener('click', e => {
  cfg.serial.plotStack = !cfg.serial.plotStack; saveConfig();
  e.currentTarget.classList.toggle('active', cfg.serial.plotStack); plot.dirty = true;
});

// splitter
(() => {
  const sp = $('#splitSerial'), pp = $('#plotPanel'), main = $('.serial-main');
  let drag = null;
  sp.addEventListener('mousedown', e => { drag = { y: e.clientY, h: pp.offsetHeight }; document.body.style.cursor = 'row-resize'; e.preventDefault(); });
  window.addEventListener('mousemove', e => {
    if (!drag) return;
    const h = clamp(drag.h - (e.clientY - drag.y), 90, main.clientHeight - 200);
    pp.style.flex = `0 0 ${h}px`; cfg.serial.split = h; plot.dirty = true;
  });
  window.addEventListener('mouseup', () => { if (drag) { drag = null; document.body.style.cursor = ''; saveConfig(); } });
})();

// ============================================================== waveform view (shared by Logic + Timeline)
const TRIG_CYCLE = ['x', 'r', 'f', '1', '0', 'e'];
const TRIG_ICON = {
  x: '<svg viewBox="0 0 24 24" style="opacity:.55"><path d="M4 16h5l3-8 3 8h5"/></svg>',
  r: '<svg viewBox="0 0 24 24"><path d="M3 18h7V6h11M10 13l0-4"/><path d="M7.5 11.5L10 9l2.5 2.5"/></svg>',
  f: '<svg viewBox="0 0 24 24"><path d="M3 6h7v12h11"/><path d="M7.5 12.5L10 15l2.5-2.5"/></svg>',
  '1': '1', '0': '0',
  e: '<svg viewBox="0 0 24 24"><path d="M3 18h5V6h5v12h8"/></svg>',
};
const TRIG_NAME = { r: '↑ rising', f: '↓ falling', '1': 'high', '0': 'low', e: '↕ any edge' };

class Wave {
  constructor(root, opts) {
    this.opts = opts;
    this.root = root;
    root.innerHTML = `
      <div class="wave-head"><div class="gh">${opts.title}</div><canvas class="ruler"></canvas></div>
      <div class="wave-body"><div class="gutter"></div><div class="wave-canvas-wrap"><canvas class="main"></canvas></div></div>
      <div class="wave-empty"><div class="es-art"><svg viewBox="0 0 120 60"><path d="M4 40h14V16h14v24h10V16h8v24h22l6-20 7 20h31"/></svg></div>
        <div class="es-t">${opts.emptyTitle}</div><div class="es-s">${opts.emptySub}</div></div>`;
    this.ruler = $('.ruler', root);
    this.cv = $('.main', root);
    this.body = $('.wave-body', root);
    this.gutter = $('.gutter', root);
    this.wrap = $('.wave-canvas-wrap', root);
    this.v0 = 0; this.v1 = 1;
    this.resp = null; this.inflight = false; this.want = null;
    this.dirty = true; this.lanesDirty = true;
    this.cursorMode = false; this.ca = null; this.cb = null; this.nextCur = 'a';
    this.hover = null; this.sel = -1;
    this.bind();
  }

  // ---------- lanes
  lanes() {
    const L = [];
    const tl = this.opts.timeline;
    if (G.cap) {
      for (let ch = 0; ch < 8; ch++) {
        const hid = cfg.logic.hidden[ch];
        if (tl && hid) continue;
        L.push({ type: 'ch', ch, h: hid ? 20 : tl ? 28 : 38 });
      }
      G.rows.forEach((r, i) => L.push({ type: 'dec', row: i, h: 30 }));
    } else if (!tl) {
      for (let ch = 0; ch < 8; ch++) L.push({ type: 'ch', ch, h: cfg.logic.hidden[ch] ? 20 : 38 });
    }
    if (tl || cfg.logic.overlay) L.push({ type: 'ser', h: 38 });
    if (tl) for (const n of plot.order) L.push({ type: 'plot', name: n, h: 64 });
    return L;
  }
  laneSig(L) {
    return JSON.stringify([L.map(l => [l.type, l.ch, l.row, l.name, l.h]), cfg.logic.names, cfg.logic.trig, G.rows.map(r => r.name + r.count), SER.state.port, this.sel]);
  }
  buildGutter() {
    const L = this.L;
    const sig = this.laneSig(L);
    if (sig === this.gutSig) return;
    this.gutSig = sig;
    const tl = this.opts.timeline;
    this.gutter.innerHTML = L.map((l, i) => {
      if (l.type === 'ch') {
        const t = cfg.logic.trig[l.ch];
        const hid = cfg.logic.hidden[l.ch];
        return `<div class="lane ch ${hid ? 'hid' : ''} ${this.sel === l.ch ? 'sel' : ''}" style="height:${l.h}px" data-i="${i}" data-ch="${l.ch}">
          <span class="cbar" style="background:${CH_COLORS[l.ch]}"></span>
          <span class="ln-name" spellcheck="false" title="Double-click to rename">${esc(cfg.logic.names[l.ch])}</span>
          ${hid ? '' : `<span class="ln-sub">${l.ch}</span>`}
          ${tl ? '' : `<button class="tbtn eye" data-eye="${l.ch}" title="${hid ? 'Show' : 'Hide'} channel"><svg viewBox="0 0 24 24">${hid ? '<path d="M3 3l18 18M10.6 6.1A9.6 9.6 0 0 1 12 6c6 0 9.5 6 9.5 6a17 17 0 0 1-2.6 3.3M6.6 6.6C3.9 8.3 2.5 12 2.5 12s3.5 6 9.5 6a9.7 9.7 0 0 0 4.4-1"/>' : '<path d="M2.5 12S6 6 12 6s9.5 6 9.5 6-3.5 6-9.5 6-9.5-6-9.5-6z"/><circle cx="12" cy="12" r="2.5"/>'}</svg></button>`}
          ${tl || hid ? '' : `<button class="tbtn ${t !== 'x' ? 'on' : ''}" data-trig="${l.ch}" title="Trigger: ${t === 'x' ? 'none' : TRIG_NAME[t]} (click to change)">${TRIG_ICON[t]}</button>`}
        </div>`;
      }
      if (l.type === 'dec') {
        const r = G.rows[l.row];
        return `<div class="lane dl" style="height:${l.h}px"><span class="cbar" style="background:${r.color}"></span>
          <span class="ln-name">${esc(r.name)}</span><span class="ln-sub">${fmtCount(r.count)}</span></div>`;
      }
      if (l.type === 'ser') return `<div class="lane ser" style="height:${l.h}px"><span class="cbar" style="background:#2dd4bf"></span>
          <span class="ln-name">Serial</span><span class="ln-sub">${esc(SER.state.open ? SER.state.port : 'log')}</span></div>`;
      const s = plot.series.get(l.name);
      return `<div class="lane plt" style="height:${l.h}px"><span class="cbar" style="background:${s ? s.color : '#888'}"></span>
          <span class="ln-name">${esc(l.name)}</span></div>`;
    }).join('');
  }

  resize() {
    if (!this.root.offsetParent) return;
    this.lanesDirty = true; this.dirty = true;
  }

  // ---------- geometry
  get W() { return this.wrap.clientWidth; }
  x(sec) { return (sec - this.v0) / (this.v1 - this.v0) * this.W; }
  sec(x) { return this.v0 + x / this.W * (this.v1 - this.v0); }
  sampleToSec(s) { return capOffset() + s / G.cap.rate; }

  bounds() {
    let a = Infinity, b = -Infinity;
    if (G.cap) { a = capOffset(); b = a + G.cap.n / G.cap.rate; }
    if (this.opts.timeline && SER.lines.length && G.origin != null) {
      a = Math.min(a, (SER.lines[0].t - G.origin) / 1000);
      b = Math.max(b, (SER.lines[SER.lines.length - 1].t - G.origin) / 1000);
    }
    if (!isFinite(a)) return [0, 1];
    if (b - a < 1e-6) { b = a + 1; }
    return [a, b];
  }
  fit() {
    const [a, b] = this.bounds();
    const pad = (b - a) * 0.02;
    this.v0 = a - pad; this.v1 = b + pad;
    this.follow(false);
    this.dirty = true;
  }
  zoom(f, atX) {
    if (atX == null) atX = this.W / 2;
    const c = this.sec(atX);
    let span = (this.v1 - this.v0) * f;
    span = clamp(span, 2e-8, 1e6);
    const r = atX / this.W;
    this.v0 = c - span * r; this.v1 = this.v0 + span;
    this.dirty = true;
  }
  pan(px) {
    const d = px / this.W * (this.v1 - this.v0);
    this.v0 += d; this.v1 += d; this.dirty = true;
  }
  centerOn(sec, span) {
    span = span || (this.v1 - this.v0);
    this.v0 = sec - span / 2; this.v1 = sec + span / 2; this.follow(false); this.dirty = true;
  }
  follow(on) {
    this.following = on;
    if (this.opts.followEl) this.opts.followEl.checked = on;
  }

  // ---------- data
  maybeRequest() {
    if (!G.cap) return;
    const W = this.W; if (W < 2) return;
    const off = capOffset(), rate = G.cap.rate;
    const s = (this.v0 - off) * rate, e = (this.v1 - off) * rate;
    const w = Math.min(4096, Math.round(W));
    const key = `${G.cap.id}|${s}|${e}|${w}`;
    if (this.resp && this.resp.key === key) return;
    if (this.inflight) { this.want = true; return; }
    this.inflight = true;
    api(`/api/logic/view?s=${s}&e=${e}&w=${w}`).then(r => {
      this.resp = { key, s, e, w: r.w, d: b64bytes(r.d), rows: r.rows, id: r.id };
      this.dirty = true;
    }).catch(() => { }).finally(() => {
      this.inflight = false;
      if (this.want) { this.want = false; this.dirty = true; }
    });
  }

  // ---------- drawing
  frame() {
    if (!this.root.offsetParent) return;
    if (this.following && this.opts.timeline) {
      const span = this.v1 - this.v0;
      if (G.origin != null) { this.v1 = (Date.now() - G.origin) / 1000 + span * 0.04; this.v0 = this.v1 - span; this.dirty = true; }
    }
    if (this.lanesDirty || this.dirty) {
      this.L = this.lanes();
      this.buildGutter();
      this.lanesDirty = false;
    }
    if (!this.dirty) return;
    this.dirty = false;
    this.maybeRequest();
    this.draw();
  }

  draw() {
    const L = this.L;
    const W = this.W;
    const H = L.reduce((a, l) => a + l.h, 0);
    const empty = !G.cap && (!this.opts.timeline || !SER.lines.length);
    $('.wave-empty', this.root).classList.toggle('hidden', !empty);
    const ctx = sizeCanvas(this.cv, W, Math.max(H, 1));
    this.wrap.style.height = H + 'px';
    ctx.clearRect(0, 0, W, H);
    const ticks = this.ticks();

    // background: alternating lanes + grid
    let y = 0;
    L.forEach((l, i) => {
      if (l.type === 'ch' && l.ch % 2 === 1) { ctx.fillStyle = 'rgba(255,255,255,.012)'; ctx.fillRect(0, y, W, l.h); }
      if (this.sel === l.ch && l.type === 'ch') { ctx.fillStyle = 'rgba(255,255,255,.03)'; ctx.fillRect(0, y, W, l.h); }
      ctx.fillStyle = 'rgba(255,255,255,.03)'; ctx.fillRect(0, y + l.h - 1, W, 1);
      l.y = y; y += l.h;
    });
    ctx.strokeStyle = 'rgba(255,255,255,.035)'; ctx.lineWidth = 1;
    ctx.beginPath();
    for (const t of ticks.major) { const x = Math.round(this.x(t)) + .5; ctx.moveTo(x, 0); ctx.lineTo(x, H); }
    ctx.stroke();

    // capture extent shading
    if (G.cap) {
      const a = this.x(capOffset()), b = this.x(capOffset() + G.cap.n / G.cap.rate);
      ctx.fillStyle = 'rgba(0,0,0,.25)';
      if (a > 0) ctx.fillRect(0, 0, Math.min(a, W), H);
      if (b < W) ctx.fillRect(Math.max(0, b), 0, W - Math.max(0, b), H);
    }

    const r = this.resp && G.cap && this.resp.id === G.cap.id ? this.resp : null;
    for (const l of L) {
      if (l.type === 'ch' && r && !cfg.logic.hidden[l.ch]) this.drawChannel(ctx, r, l);
      else if (l.type === 'dec' && r) this.drawDec(ctx, r, l);
      else if (l.type === 'ser') this.drawSerial(ctx, l);
      else if (l.type === 'plot') this.drawPlotLane(ctx, l);
    }

    // trigger marker
    if (G.cap && G.cap.trig >= 0) {
      const x = Math.round(this.x(this.sampleToSec(G.cap.trig))) + .5;
      ctx.strokeStyle = 'rgba(251,191,36,.7)'; ctx.setLineDash([4, 4]);
      ctx.beginPath(); ctx.moveTo(x, 0); ctx.lineTo(x, H); ctx.stroke(); ctx.setLineDash([]);
    }
    // cursors
    const drawCur = (s, col) => {
      if (s == null) return;
      const x = Math.round(this.x(s)) + .5;
      ctx.strokeStyle = col; ctx.lineWidth = 1.2;
      ctx.beginPath(); ctx.moveTo(x, 0); ctx.lineTo(x, H); ctx.stroke(); ctx.lineWidth = 1;
    };
    if (this.ca != null && this.cb != null) {
      const a = this.x(this.ca), b = this.x(this.cb);
      ctx.fillStyle = 'rgba(56,189,248,.06)'; ctx.fillRect(Math.min(a, b), 0, Math.abs(b - a), H);
    }
    drawCur(this.ca, '#fbbf24'); drawCur(this.cb, '#38bdf8');
    // hover
    if (this.hover != null) {
      const x = Math.round(this.x(this.hover)) + .5;
      ctx.strokeStyle = 'rgba(255,255,255,.22)'; ctx.setLineDash([2, 3]);
      ctx.beginPath(); ctx.moveTo(x, 0); ctx.lineTo(x, H); ctx.stroke(); ctx.setLineDash([]);
    }
    this.drawRuler(ticks);
    this.updateStatus(r);
  }

  ticks() {
    const W = this.W, span = this.v1 - this.v0;
    const st = niceStep(span / Math.max(1, W / 120));
    const major = [], minor = [];
    const first = Math.ceil(this.v0 / st) * st;
    for (let t = first; t <= this.v1; t += st) major.push(Math.abs(t) < st * 1e-6 ? 0 : t);
    const ms = st / 5;
    for (let t = Math.ceil(this.v0 / ms) * ms; t <= this.v1; t += ms) minor.push(t);
    return { major, minor, st };
  }

  drawRuler(ticks) {
    const W = this.W, H = 30;
    const ctx = sizeCanvas(this.ruler, W, H);
    ctx.clearRect(0, 0, W, H);
    ctx.fillStyle = 'rgba(0,0,0,.15)'; ctx.fillRect(0, 0, W, H);
    ctx.strokeStyle = 'rgba(255,255,255,.12)';
    ctx.beginPath();
    for (const t of ticks.minor) { const x = Math.round(this.x(t)) + .5; ctx.moveTo(x, H - 5); ctx.lineTo(x, H); }
    ctx.stroke();
    ctx.strokeStyle = 'rgba(255,255,255,.3)';
    ctx.beginPath();
    ctx.font = '11px ' + getComputedStyle(document.body).getPropertyValue('--mono');
    ctx.fillStyle = '#9ba8ba'; ctx.textAlign = 'left'; ctx.textBaseline = 'middle';
    for (const t of ticks.major) {
      const x = Math.round(this.x(t)) + .5;
      ctx.moveTo(x, H - 11); ctx.lineTo(x, H);
      ctx.fillText(fmtSec(t, 3), x + 4, 12);
    }
    ctx.stroke();
    // trigger flag
    if (G.cap && G.cap.trig >= 0) {
      const x = this.x(this.sampleToSec(G.cap.trig));
      ctx.fillStyle = '#fbbf24';
      ctx.beginPath(); ctx.moveTo(x - 6, H - 12); ctx.lineTo(x + 6, H - 12); ctx.lineTo(x, H - 2); ctx.closePath(); ctx.fill();
      ctx.fillStyle = '#1c1404'; ctx.font = 'bold 9px sans-serif'; ctx.textAlign = 'center'; ctx.fillText('T', x, H - 8.5);
    }
    const flag = (s, col, label) => {
      if (s == null) return;
      const x = this.x(s);
      ctx.fillStyle = col; roundRect(ctx, x - 9, 3, 18, 15, 4); ctx.fill();
      ctx.fillStyle = '#0b0f16'; ctx.font = 'bold 11px sans-serif'; ctx.textAlign = 'center'; ctx.fillText(label, x, 11);
      ctx.fillRect(x - .5, 18, 1, H - 18);
    };
    flag(this.ca, '#fbbf24', 'A'); flag(this.cb, '#38bdf8', 'B');
    if (this.hover != null) {
      const x = this.x(this.hover);
      const txt = fmtSec(this.hover, 5);
      ctx.font = '11px ' + getComputedStyle(document.body).getPropertyValue('--mono');
      const w = ctx.measureText(txt).width + 12;
      ctx.fillStyle = '#e7edf6'; roundRect(ctx, clamp(x - w / 2, 0, W - w), 3, w, 16, 5); ctx.fill();
      ctx.fillStyle = '#0b0f16'; ctx.textAlign = 'center'; ctx.fillText(txt, clamp(x, w / 2, W - w / 2), 11.5);
    }
  }

  // map response pixel p -> screen x (linear)
  respMap(r) {
    const spp = (r.e - r.s) / r.w;
    const x0 = this.x(this.sampleToSec(r.s));
    const step = this.x(this.sampleToSec(r.s + spp)) - x0;
    return [x0, step];
  }

  drawChannel(ctx, r, l) {
    const [x0, step] = this.respMap(r);
    const col = CH_COLORS[l.ch];
    const yHi = l.y + 8, yLo = l.y + l.h - 8;
    const off = l.ch * r.w;
    const line = new Path2D(), fill = new Path2D(), dense = new Path2D();
    let pen = false, W = this.W;
    for (let p = 0; p < r.w; p++) {
      const b = r.d[off + p];
      const xa = x0 + p * step, xb = xa + step;
      if (xb < -2 || xa > W + 2 || b === 255) { pen = false; continue; }
      const lvl = b & 1, n = b >> 1;
      const y = lvl ? yHi : yLo;
      if (!pen) { line.moveTo(xa, y); pen = true; }
      if (n === 0) {
        line.lineTo(xb, y);
        if (lvl) fill.rect(xa, yHi, xb - xa, yLo - yHi);
      } else if (n === 1) {
        const xm = step > 3 ? xa + step / 2 : xa;
        const y2 = lvl ? yLo : yHi;
        line.lineTo(xm, y); line.lineTo(xm, y2); line.lineTo(xb, y2);
        if (lvl) fill.rect(xa, yHi, xm - xa, yLo - yHi); else fill.rect(xm, yHi, xb - xm, yLo - yHi);
      } else {
        dense.rect(xa, yHi, Math.max(1, xb - xa), yLo - yHi);
        const e = lvl ^ (n & 1);
        line.moveTo(xb, e ? yHi : yLo);
      }
    }
    ctx.fillStyle = col + '1c'; ctx.fill(fill);
    ctx.fillStyle = col + '88'; ctx.fill(dense);
    ctx.strokeStyle = col; ctx.lineWidth = 1.6; ctx.lineJoin = 'miter';
    ctx.shadowColor = col + '99'; ctx.shadowBlur = 6;
    ctx.stroke(line);
    ctx.shadowBlur = 0; ctx.lineWidth = 1;
  }

  drawDec(ctx, r, l) {
    const row = G.rows[l.row];
    const items = r.rows && r.rows[l.row];
    if (!items) return;
    const y = l.y + 5, h = l.h - 10, mid = y + h / 2;
    ctx.font = '11.5px ' + getComputedStyle(document.body).getPropertyValue('--mono');
    ctx.textAlign = 'center'; ctx.textBaseline = 'middle';
    for (const [s, e, k, m, t] of items) {
      const xa = this.x(this.sampleToSec(s)), xb = this.x(this.sampleToSec(e));
      if (xb < 0 || xa > this.W) continue;
      const w = Math.max(1, xb - xa);
      const col = KIND_COLORS[k] || row.color;
      if (m > 0) {
        ctx.fillStyle = col + '55'; ctx.fillRect(xa, y + 3, w, h - 6);
        if (w > 60) { ctx.fillStyle = '#cbd5e1'; ctx.fillText(`${m} items`, xa + w / 2, mid); }
        continue;
      }
      ctx.beginPath();
      if (w > 10) {
        const c = Math.min(5, w / 4);
        ctx.moveTo(xa, mid); ctx.lineTo(xa + c, y); ctx.lineTo(xb - c, y); ctx.lineTo(xb, mid); ctx.lineTo(xb - c, y + h); ctx.lineTo(xa + c, y + h); ctx.closePath();
      } else ctx.rect(xa, y, w, h);
      ctx.fillStyle = col + (k === 3 ? '22' : '2e'); ctx.fill();
      ctx.strokeStyle = col + 'cc'; ctx.stroke();
      if (w > 16) {
        let txt = t;
        const max = w - 10;
        if (ctx.measureText(txt).width > max) {
          const short = t.replace(/ '.*'$/, '').replace(/^(Write|Read) /, (x, a) => a[0] + ' ');
          txt = short;
          while (txt.length > 1 && ctx.measureText(txt + '…').width > max) txt = txt.slice(0, -1);
          if (txt !== short) txt += '…';
          if (txt.length <= 1) continue;
        }
        ctx.fillStyle = k === 3 ? '#94a3b8' : '#f1f5f9';
        ctx.fillText(txt, xa + w / 2, mid + .5);
      }
    }
  }

  drawSerial(ctx, l) {
    if (G.origin == null) return;
    const lines = SER.lines;
    const t0 = G.origin + this.v0 * 1000, t1 = G.origin + this.v1 * 1000;
    let a = lowerBound(lines, 't', t0), b = lowerBound(lines, 't', t1);
    const y = l.y + 7, h = l.h - 14, mid = y + h / 2;
    ctx.font = '11.5px ' + getComputedStyle(document.body).getPropertyValue('--mono');
    ctx.textBaseline = 'middle'; ctx.textAlign = 'left';
    if (a >= b) {
      ctx.fillStyle = '#3b4556'; ctx.fillText('no serial data in this window', 10, mid);
      return;
    }
    const W = this.W;
    if (b - a > W / 4) {
      // dense: tick marks only
      ctx.fillStyle = 'rgba(45,212,191,.55)';
      let lastX = -1;
      for (let i = a; i < b; i++) {
        const x = Math.round(this.x((lines[i].t - G.origin) / 1000));
        if (x === lastX) continue; lastX = x;
        ctx.fillStyle = lines[i].dir === 'tx' ? 'rgba(196,181,253,.7)' : lines[i].dir === 'sys' ? 'rgba(148,163,184,.5)' : 'rgba(45,212,191,.55)';
        ctx.fillRect(x, y + 3, 1, h - 6);
      }
      return;
    }
    for (let i = a; i < b; i++) {
      const ln = lines[i];
      const x = this.x((ln.t - G.origin) / 1000);
      const nx = i + 1 < lines.length ? this.x((lines[i + 1].t - G.origin) / 1000) : W + 400;
      const col = ln.dir === 'tx' ? '#c4b5fd' : ln.dir === 'sys' ? '#94a3b8' : '#2dd4bf';
      ctx.fillStyle = col; ctx.fillRect(Math.round(x), y, 1.5, h);
      const avail = Math.min(nx - x - 5, 420);
      if (avail < 18) continue;
      let txt = (ln.dir === 'tx' ? '→ ' : '') + ln.text.replace(ANSI_RE, '');
      if (!txt) continue;
      let tw = ctx.measureText(txt).width;
      if (tw > avail - 12) {
        while (txt.length > 1 && ctx.measureText(txt + '…').width > avail - 12) txt = txt.slice(0, -1);
        txt += '…'; tw = ctx.measureText(txt).width;
      }
      ctx.fillStyle = col + '1f'; roundRect(ctx, x + 2, y + 2, tw + 12, h - 4, 6); ctx.fill();
      ctx.strokeStyle = col + '55'; ctx.stroke();
      ctx.fillStyle = ln.dir === 'sys' ? '#94a3b8' : '#e2e8f0'; ctx.fillText(txt, x + 8, mid + .5);
    }
  }

  drawPlotLane(ctx, l) {
    const s = plot.series.get(l.name);
    if (!s || G.origin == null) return;
    const t0 = G.origin + this.v0 * 1000, t1 = G.origin + this.v1 * 1000;
    const a = Math.max(0, lowerBoundNum(s.t, t0) - 1), b = Math.min(s.t.length, lowerBoundNum(s.t, t1) + 1);
    if (b - a < 1) return;
    let lo = Infinity, hi = -Infinity;
    for (let i = a; i < b; i++) { if (s.v[i] < lo) lo = s.v[i]; if (s.v[i] > hi) hi = s.v[i]; }
    if (hi - lo < 1e-12) { lo -= 1; hi += 1; }
    const y0 = l.y + 8, hh = l.h - 16;
    const Y = v => y0 + hh - (v - lo) / (hi - lo) * hh;
    const X = t => this.x((t - G.origin) / 1000);
    ctx.beginPath();
    let col = -1e9, mn = 0, mx = 0;
    const dec = b - a > this.W * 2;
    for (let i = a; i < b; i++) {
      const x = X(s.t[i]);
      if (dec) {
        const xi = Math.floor(x);
        if (xi !== col) { if (col > -1e9) { ctx.lineTo(col, Y(mn)); ctx.lineTo(col, Y(mx)); } else ctx.moveTo(xi, Y(s.v[i])); col = xi; mn = mx = s.v[i]; }
        else { mn = Math.min(mn, s.v[i]); mx = Math.max(mx, s.v[i]); }
      } else i === a ? ctx.moveTo(x, Y(s.v[i])) : ctx.lineTo(x, Y(s.v[i]));
    }
    ctx.strokeStyle = s.color; ctx.lineWidth = 1.5; ctx.shadowColor = s.color; ctx.shadowBlur = 6; ctx.stroke(); ctx.shadowBlur = 0; ctx.lineWidth = 1;
    ctx.font = '10.5px ' + getComputedStyle(document.body).getPropertyValue('--mono');
    ctx.textAlign = 'right'; ctx.textBaseline = 'middle';
    const tag = (txt, yy) => {
      const w = ctx.measureText(txt).width + 8;
      ctx.fillStyle = 'rgba(12,16,24,.85)'; roundRect(ctx, this.W - w - 4, yy - 7, w, 14, 4); ctx.fill();
      ctx.fillStyle = '#7b8798'; ctx.fillText(txt, this.W - 8, yy + .5);
    };
    tag(fmtVal(hi), l.y + 9); tag(fmtVal(lo), l.y + l.h - 9);
  }

  levelsAt(r, sec) {
    if (!r) return null;
    const [x0, step] = this.respMap(r);
    const p = Math.floor((this.x(sec) - x0) / step);
    if (p < 0 || p >= r.w) return null;
    const out = [];
    for (let ch = 0; ch < 8; ch++) { const b = r.d[ch * r.w + p]; out.push(b === 255 ? null : b & 1); }
    return out;
  }

  updateStatus(r) {
    const el = this.opts.statusEl;
    const parts = [];
    if (this.hover != null) {
      parts.push(`<span>t <b>${fmtSec(this.hover, 5)}</b></span>`);
      if (G.origin != null) parts.push(`<span>${clock(G.origin + this.hover * 1000)}</span>`);
      const lv = this.levelsAt(r, this.hover);
      if (lv) parts.push('<span>' + lv.map((v, i) => v == null ? '' : `<span style="color:${CH_COLORS[i]}">${esc(cfg.logic.names[i])}</span>=<b>${v}</b>`).filter(Boolean).join(' ') + '</span>');
    } else {
      parts.push(`<span>view <b>${fmtSec(this.v1 - this.v0)}</b></span>`);
      if (G.cap) parts.push(`<span>${fmtCount(G.cap.n)} samples @ ${fmtHz(G.cap.rate).replace('Hz', 'S/s')}</span>`, `<span>${fmtCount(G.cap.edges)} edges</span>`);
    }
    if (this.ca != null) parts.push(`<span class="ca">A ${fmtSec(this.ca, 5)}</span>`);
    if (this.cb != null) parts.push(`<span class="cb">B ${fmtSec(this.cb, 5)}</span>`);
    if (this.ca != null && this.cb != null) {
      const d = Math.abs(this.cb - this.ca);
      parts.push(`<span>Δ <b>${fmtSec(d, 5)}</b></span><span>1/Δ <b>${fmtHz(1 / d)}</b></span>`);
    }
    el.innerHTML = parts.join('');
    const vk = [this.v0, this.v1, this.ca, this.cb, this.sel, G.cap && G.cap.id].join();
    if (this.opts.onView && vk !== this.viewKey) { this.viewKey = vk; this.opts.onView(this); }
  }

  // ---------- interaction
  bind() {
    const cv = this.cv;
    let drag = null;
    cv.addEventListener('wheel', e => {
      e.preventDefault();
      const r = cv.getBoundingClientRect();
      const x = e.clientX - r.left;
      if (e.shiftKey || Math.abs(e.deltaX) > Math.abs(e.deltaY)) this.pan((e.shiftKey ? e.deltaY : e.deltaX) * 1.0);
      else this.zoom(Math.exp(e.deltaY * 0.0016), x);
      if (this.opts.timeline && this.following && !e.shiftKey) { /* keep following while zooming */ }
      else this.follow(false);
    }, { passive: false });
    this.ruler.addEventListener('wheel', e => { e.preventDefault(); const r = this.ruler.getBoundingClientRect(); this.zoom(Math.exp(e.deltaY * 0.0016), e.clientX - r.left); }, { passive: false });
    cv.addEventListener('mousedown', e => {
      if (e.button !== 0) return;
      drag = { x: e.clientX, v0: this.v0, v1: this.v1, moved: false };
      cv.classList.add('dragging');
    });
    window.addEventListener('mousemove', e => {
      if (!drag) return;
      const dx = e.clientX - drag.x;
      if (Math.abs(dx) > 3) drag.moved = true;
      if (drag.moved) {
        const d = dx / this.W * (drag.v1 - drag.v0);
        this.v0 = drag.v0 - d; this.v1 = drag.v1 - d; this.dirty = true; this.follow(false);
      }
    });
    window.addEventListener('mouseup', e => {
      if (!drag) return;
      cv.classList.remove('dragging');
      if (!drag.moved && this.cursorMode) {
        const r = cv.getBoundingClientRect();
        this.placeCursor(this.sec(e.clientX - r.left));
      }
      drag = null;
    });
    cv.addEventListener('dblclick', e => { const r = cv.getBoundingClientRect(); this.zoom(0.4, e.clientX - r.left); this.follow(false); });
    cv.addEventListener('mousemove', e => {
      const r = cv.getBoundingClientRect();
      this.hover = this.sec(e.clientX - r.left); this.dirty = true;
    });
    cv.addEventListener('mouseleave', () => { this.hover = null; this.dirty = true; });
    // ruler: drag cursor flags
    let rdrag = null;
    this.ruler.addEventListener('mousedown', e => {
      const r = this.ruler.getBoundingClientRect(), x = e.clientX - r.left;
      const near = (s) => s != null && Math.abs(this.x(s) - x) < 10;
      if (near(this.ca)) rdrag = 'a'; else if (near(this.cb)) rdrag = 'b';
      else { this.placeCursor(this.sec(x)); }
    });
    window.addEventListener('mousemove', e => {
      if (!rdrag) return;
      const r = this.ruler.getBoundingClientRect();
      this['c' + rdrag] = this.sec(e.clientX - r.left); this.dirty = true;
    });
    window.addEventListener('mouseup', () => { rdrag = null; });
    // gutter
    this.gutter.addEventListener('click', e => {
      const tb = e.target.closest('[data-trig]');
      if (tb) {
        const ch = +tb.dataset.trig;
        const cur = cfg.logic.trig[ch];
        cfg.logic.trig[ch] = TRIG_CYCLE[(TRIG_CYCLE.indexOf(cur) + 1) % TRIG_CYCLE.length];
        // only one edge condition makes sense at a time
        if ('rfe'.includes(cfg.logic.trig[ch])) cfg.logic.trig.forEach((t, i) => { if (i !== ch && 'rfe'.includes(t)) cfg.logic.trig[i] = 'x'; });
        saveConfig(); renderTrigSum(); waveLogic.lanesDirty = true; waveLogic.dirty = true;
        return;
      }
      const eye = e.target.closest('[data-eye]');
      if (eye) {
        const ch = +eye.dataset.eye;
        cfg.logic.hidden[ch] = !cfg.logic.hidden[ch]; saveConfig();
        waveLogic.lanesDirty = waveTL.lanesDirty = true; waveLogic.dirty = waveTL.dirty = true;
        return;
      }
      const lane = e.target.closest('.lane.ch');
      if (lane) { this.sel = +lane.dataset.ch; this.dirty = true; this.lanesDirty = true; measure(); }
    });
    this.gutter.addEventListener('dblclick', e => {
      const n = e.target.closest('.ln-name'); if (!n || !n.closest('.lane.ch')) return;
      n.contentEditable = 'true'; n.focus();
      document.getSelection().selectAllChildren(n);
    });
    this.gutter.addEventListener('keydown', e => { if (e.key === 'Enter' && e.target.isContentEditable) { e.preventDefault(); e.target.blur(); } });
    this.gutter.addEventListener('focusout', e => {
      const n = e.target; if (!n.isContentEditable) return;
      n.contentEditable = 'false';
      const ch = +n.closest('.lane').dataset.ch;
      cfg.logic.names[ch] = n.textContent.trim().slice(0, 24) || 'CH' + ch;
      saveConfig(); waveLogic.lanesDirty = waveTL.lanesDirty = true; waveLogic.dirty = waveTL.dirty = true; renderTrigSum(); renderDecoders();
    });
  }
  placeCursor(s) {
    if (this.nextCur === 'a') { this.ca = s; this.nextCur = 'b'; } else { this.cb = s; this.nextCur = 'a'; }
    this.dirty = true;
    measure();
  }
  setCursorMode(on, btn) {
    this.cursorMode = on;
    this.cv.classList.toggle('cursor-mode', on);
    btn.classList.toggle('active', on);
    if (!on) { this.ca = this.cb = null; this.nextCur = 'a'; }
    this.dirty = true;
  }
}

const waveLogic = new Wave($('#waveLogic'), {
  title: 'Channels', statusEl: $('#logicStatusLine'),
  emptyTitle: 'No capture yet', emptySub: 'Press <b>Capture</b> (Space). No hardware? Pick the <b>Demo signal generator</b>.',
  onView: () => measure(),
});
const waveTL = new Wave($('#waveTimeline'), {
  title: 'Lanes', timeline: true, statusEl: $('#tlStatusLine'), followEl: $('#swFollow'),
  emptyTitle: 'Nothing on the timeline yet', emptySub: 'Connect a serial port or run a logic capture — both land here on one clock.',
});

// ============================================================== logic controls
const LOGIC = { devices: [], rates: [], state: { status: 'idle' } };
const SAMPLE_OPTS = [10e3, 50e3, 100e3, 250e3, 500e3, 1e6, 2e6, 5e6, 10e6, 20e6, 50e6, 100e6];

async function refreshDevices() {
  try {
    const r = await api('/api/logic/devices');
    LOGIC.devices = r.devices; LOGIC.rates = r.rates;
    if (!LOGIC.devices.find(d => d.id === cfg.logic.device)) {
      const real = LOGIC.devices.find(d => !d.demo);
      cfg.logic.device = (real || LOGIC.devices[LOGIC.devices.length - 1]).id;
    }
    renderDevices(); renderRates();
  } catch (e) { toast('USB scan failed: ' + esc(e.message), 'err'); }
}
const LA_ICON = '<svg viewBox="0 0 24 24"><rect x="5" y="3" width="14" height="18" rx="2"/><path d="M9 7h6M9 11h6M9 15h6"/></svg>';
function renderDevices() {
  const st = { ready: ['ok', 'Ready'], driver: ['bad', 'Driver'], nofirmware: ['warn', 'Firmware'], error: ['bad', 'Error'] };
  $('#devList').innerHTML = LOGIC.devices.map(d => {
    const [c, t] = st[d.status] || ['', d.status];
    return `<div class="dev ${d.id === cfg.logic.device ? 'sel' : ''}" data-id="${esc(d.id)}">
      <div class="di">${d.demo ? DEMO_ICON : LA_ICON}</div>
      <div class="dt"><div class="dn">${esc(d.name)}</div><div class="dd">${esc(d.detail)}</div></div>
      <span class="badge ${c}">${t}</span></div>`;
  }).join('');
  const sel = LOGIC.devices.find(d => d.id === cfg.logic.device);
  $('#fwDrop').classList.toggle('hidden', !(sel && sel.status === 'nofirmware'));
  if (sel && sel.status === 'nofirmware') $('#fwName').textContent = (sel.detail.match(/fx2lafw[\w-]*\.fw/) || ['fx2lafw-saleae-logic.fw'])[0];
}
$('#devList').addEventListener('click', e => {
  const d = e.target.closest('.dev'); if (!d) return;
  cfg.logic.device = d.dataset.id; saveConfig(); renderDevices();
  const dev = LOGIC.devices.find(x => x.id === d.dataset.id);
  if (dev && dev.status === 'driver') toast('This analyser needs the WinUSB driver first — see <b>Help → Logic analyser setup</b>.', 'warn', 7000);
});
$('#btnRefreshDev').addEventListener('click', refreshDevices);

function renderRates() {
  const rates = LOGIC.rates.length ? LOGIC.rates : [1e6];
  if (!rates.includes(cfg.logic.rate)) cfg.logic.rate = rates.includes(4e6) ? 4e6 : rates[rates.length - 1];
  $('#selRate').innerHTML = rates.map(r => `<option value="${r}" ${r === cfg.logic.rate ? 'selected' : ''}>${fmtHz(r)}</option>`).join('');
  renderSamples();
}
function renderSamples() {
  if (!SAMPLE_OPTS.includes(cfg.logic.samples)) cfg.logic.samples = 2e6;
  $('#selSamples').innerHTML = SAMPLE_OPTS.map(n => `<option value="${n}" ${n === cfg.logic.samples ? 'selected' : ''}>${fmtCount(n)}  ·  ${fmtSec(n / cfg.logic.rate, 2)}</option>`).join('');
  const mem = cfg.logic.samples;
  $('#durText').textContent = `Records ${fmtSec(cfg.logic.samples / cfg.logic.rate)} · ${fmtBytes(mem)} RAM · resolution ${fmtSec(1 / cfg.logic.rate)}`;
}
$('#selRate').addEventListener('change', e => { cfg.logic.rate = +e.target.value; saveConfig(); renderSamples(); });
$('#selSamples').addEventListener('change', e => { cfg.logic.samples = +e.target.value; saveConfig(); renderSamples(); });
$('#rngPre').addEventListener('input', e => { cfg.logic.pre = +e.target.value; $('#preVal').textContent = cfg.logic.pre + '%'; saveConfig(); });
function renderTrigSum() {
  const t = cfg.logic.trig.map((v, i) => v === 'x' ? '' : `<span class="tchip">${esc(cfg.logic.names[i])} ${TRIG_NAME[v]}</span>`).filter(Boolean);
  $('#trigSum').innerHTML = t.length ? t.join('') + `<button class="btn ghost sm" id="btnTrigClear">Clear</button>` : '<span class="tnone">Free-running — captures immediately</span>';
  const c = $('#btnTrigClear');
  if (c) c.addEventListener('click', () => { cfg.logic.trig = Array(8).fill('x'); saveConfig(); renderTrigSum(); waveLogic.lanesDirty = true; waveLogic.dirty = true; });
}

// firmware upload
async function uploadFirmware(file) {
  try {
    const buf = await file.arrayBuffer();
    await api('/api/logic/firmware?name=' + encodeURIComponent(file.name), undefined, buf);
    toast(`Firmware <b>${esc(file.name)}</b> installed`);
    refreshDevices();
  } catch (e) { toast(esc(e.message), 'err'); }
}
$('#fwFile').addEventListener('change', e => { if (e.target.files[0]) uploadFirmware(e.target.files[0]); });
const fwd = $('#fwDrop');
fwd.addEventListener('dragover', e => { e.preventDefault(); fwd.classList.add('over'); });
fwd.addEventListener('dragleave', () => fwd.classList.remove('over'));
fwd.addEventListener('drop', e => { e.preventDefault(); fwd.classList.remove('over'); if (e.dataTransfer.files[0]) uploadFirmware(e.dataTransfer.files[0]); });

// run / stop
const BUSY = ['arming', 'waiting', 'capturing', 'processing'];
async function runCapture() {
  if (BUSY.includes(LOGIC.state.status)) { LOGIC.stopRequested = true; await api('/api/logic/stop', {}).catch(() => { }); return; }
  LOGIC.stopRequested = false;
  try {
    await api('/api/logic/start', { device: cfg.logic.device, rate: cfg.logic.rate, samples: cfg.logic.samples, pretrig: cfg.logic.pre / 100, trig: cfg.logic.trig });
  } catch (e) { toast(esc(e.message), 'err'); }
}
$('#btnRun').addEventListener('click', runCapture);
$('#swRepeat').addEventListener('change', e => { cfg.logic.repeat = e.target.checked; saveConfig(); });
$('#swOverlay').addEventListener('change', e => { cfg.logic.overlay = e.target.checked; saveConfig(); waveLogic.lanesDirty = true; waveLogic.dirty = true; });

function applyLogicState(st) {
  const prev = LOGIC.state;
  LOGIC.state = st;
  const busy = BUSY.includes(st.status);
  const b = $('#btnRun');
  b.classList.toggle('stop', busy);
  b.innerHTML = busy ? '<svg viewBox="0 0 24 24"><rect x="6" y="6" width="12" height="12" rx="2"/></svg><span>Stop</span>' : '<svg viewBox="0 0 24 24"><path d="M7 4l13 8-13 8z"/></svg><span>Capture</span>';
  const cs = $('#capStatus');
  cs.classList.toggle('waiting', st.status === 'waiting' || st.status === 'arming');
  $('#capBar').style.width = (st.status === 'done' ? 100 : (st.progress || 0) * 100) + '%';
  $('#capMsg').textContent = st.status === 'error' ? '⚠ ' + st.error : st.message || (st.status === 'idle' ? 'Ready' : st.status);
  const chip = $('#chipLogic');
  chip.querySelector('.dot').className = 'dot ' + (busy ? 'busy' : st.status === 'error' ? 'err' : G.cap ? 'on' : '');
  chip.querySelector('.lbl').textContent = busy ? (st.status === 'waiting' ? 'Waiting for trigger' : 'Capturing…') : st.status === 'error' ? 'Logic error' : G.cap ? 'Capture ready' : 'Logic idle';
  if (st.status === 'error' && prev.status !== 'error') toast(esc(st.error), 'err', 8000);
  if (st.status === 'done' && st.captureId && (!G.cap || G.cap.id !== st.captureId)) loadCapture();
}

async function loadCapture() {
  try {
    const info = await api('/api/logic/info');
    if (!info.has) return;
    const keep = G.cap && cfg.logic.repeat && G.cap.rate === info.rate && G.cap.n === info.n;
    const viewRel = keep ? [waveLogic.v0 - capOffset(), waveLogic.v1 - capOffset()] : null;
    G.cap = info;
    G.origin = info.t0;
    await runDecode(true);
    if (keep) { waveLogic.v0 = viewRel[0]; waveLogic.v1 = viewRel[1]; } else waveLogic.fit();
    waveTL.fit();
    waveLogic.lanesDirty = waveTL.lanesDirty = true; waveLogic.dirty = waveTL.dirty = true;
    applyLogicState(LOGIC.state);
    if (cfg.logic.repeat && !LOGIC.stopRequested) setTimeout(() => { if (cfg.logic.repeat && !BUSY.includes(LOGIC.state.status)) runCapture(); }, 300);
  } catch (e) { toast('Could not load capture: ' + esc(e.message), 'err'); }
}

// ============================================================== decoders
let decSeq = 0;
function defaultDecoders() {
  return [
    { id: 'd1', type: 'uart', name: 'UART', ch: 0, baud: 115200, bits: 8, parity: 'none', stop: 1, invert: false, format: 'hexascii' },
    { id: 'd2', type: 'i2c', name: 'I²C', scl: 1, sda: 2, format: 'hex' },
    { id: 'd3', type: 'spi', name: 'SPI', clk: 3, mosi: 4, miso: 5, cs: 6, cpol: 0, cpha: 0, lsb: false, wordBits: 8, csHigh: false, format: 'hex' },
  ];
}
function chOpts(sel, allowNone) {
  return (allowNone ? `<option value="-1" ${sel < 0 ? 'selected' : ''}>—</option>` : '') +
    cfg.logic.names.map((n, i) => `<option value="${i}" ${i === sel ? 'selected' : ''}>${esc(n)}</option>`).join('');
}
function fmtOpts(sel) {
  return [['hexascii', 'Hex + ASCII'], ['hex', 'Hex'], ['dec', 'Decimal'], ['ascii', 'ASCII'], ['bin', 'Binary']]
    .map(([v, t]) => `<option value="${v}" ${v === sel ? 'selected' : ''}>${t}</option>`).join('');
}
function renderDecoders() {
  const el = $('#decList');
  el.innerHTML = cfg.logic.decoders.map((d, i) => {
    const col = DEC_COLORS[i % DEC_COLORS.length];
    let body = '';
    if (d.type === 'uart') body = `
      <div class="grid2"><div><label class="field-l">RX channel</label><select class="inp" data-k="ch">${chOpts(d.ch)}</select></div>
      <div><label class="field-l">Baud</label><input class="inp mono" data-k="baud" value="${d.baud}" list="baudList"></div></div>
      <div class="grid3"><div><label class="field-l">Bits</label><select class="inp" data-k="bits">${[8, 7, 6, 5, 9].map(v => `<option ${v === d.bits ? 'selected' : ''}>${v}</option>`).join('')}</select></div>
      <div><label class="field-l">Parity</label><select class="inp" data-k="parity">${['none', 'even', 'odd'].map(v => `<option ${v === d.parity ? 'selected' : ''}>${v}</option>`).join('')}</select></div>
      <div><label class="field-l">Stop</label><select class="inp" data-k="stop">${[1, 1.5, 2].map(v => `<option ${v === d.stop ? 'selected' : ''}>${v}</option>`).join('')}</select></div></div>
      <div class="grid2"><div><label class="field-l">Show as</label><select class="inp" data-k="format">${fmtOpts(d.format)}</select></div>
      <div><label class="field-l">Polarity</label><select class="inp" data-k="invert"><option value="false" ${!d.invert ? 'selected' : ''}>Normal</option><option value="true" ${d.invert ? 'selected' : ''}>Inverted</option></select></div></div>`;
    else if (d.type === 'i2c') body = `
      <div class="grid3"><div><label class="field-l">SCL</label><select class="inp" data-k="scl">${chOpts(d.scl)}</select></div>
      <div><label class="field-l">SDA</label><select class="inp" data-k="sda">${chOpts(d.sda)}</select></div>
      <div><label class="field-l">Show as</label><select class="inp" data-k="format">${fmtOpts(d.format)}</select></div></div>`;
    else body = `
      <div class="grid2"><div><label class="field-l">CLK</label><select class="inp" data-k="clk">${chOpts(d.clk)}</select></div>
      <div><label class="field-l">CS</label><select class="inp" data-k="cs">${chOpts(d.cs, true)}</select></div>
      <div><label class="field-l">MOSI</label><select class="inp" data-k="mosi">${chOpts(d.mosi, true)}</select></div>
      <div><label class="field-l">MISO</label><select class="inp" data-k="miso">${chOpts(d.miso, true)}</select></div></div>
      <div class="grid3"><div><label class="field-l">Mode</label><select class="inp" data-k="mode">${[0, 1, 2, 3].map(m => `<option value="${m}" ${m === d.cpol * 2 + d.cpha ? 'selected' : ''}>Mode ${m}</option>`).join('')}</select></div>
      <div><label class="field-l">Order</label><select class="inp" data-k="lsb"><option value="false" ${!d.lsb ? 'selected' : ''}>MSB</option><option value="true" ${d.lsb ? 'selected' : ''}>LSB</option></select></div>
      <div><label class="field-l">Bits</label><select class="inp" data-k="wordBits">${[8, 16, 24, 32, 4, 12].map(v => `<option ${v === d.wordBits ? 'selected' : ''}>${v}</option>`).join('')}</select></div></div>
      <div class="grid2"><div><label class="field-l">Show as</label><select class="inp" data-k="format">${fmtOpts(d.format)}</select></div>
      <div><label class="field-l">CS active</label><select class="inp" data-k="csHigh"><option value="false" ${!d.csHigh ? 'selected' : ''}>Low</option><option value="true" ${d.csHigh ? 'selected' : ''}>High</option></select></div></div>`;
    return `<div class="dec" style="--dc:${col}" data-i="${i}">
      <div class="dec-h"><span class="badge" style="background:${col}22;color:${col}">${d.type.toUpperCase().replace('I2C', 'I²C')}</span>
        <input data-k="name" value="${esc(d.name)}" spellcheck="false">
        <button class="icon-btn sm" data-del title="Remove decoder"><svg viewBox="0 0 24 24"><path d="M6 6l12 12M18 6L6 18"/></svg></button></div>
      ${body}</div>`;
  }).join('') + `<datalist id="baudList">${BAUDS.map(b => `<option value="${b}">`).join('')}</datalist>`;
}
$('#decList').addEventListener('change', e => {
  const card = e.target.closest('.dec'); if (!card) return;
  const d = cfg.logic.decoders[+card.dataset.i];
  const k = e.target.dataset.k; let v = e.target.value;
  if (k === 'mode') { d.cpol = +v >> 1; d.cpha = +v & 1; }
  else if (['ch', 'scl', 'sda', 'clk', 'mosi', 'miso', 'cs', 'bits', 'wordBits', 'baud', 'stop'].includes(k)) d[k] = +v;
  else if (['invert', 'lsb', 'csHigh'].includes(k)) d[k] = v === 'true';
  else d[k] = v;
  saveConfig(); runDecode();
});
$('#decList').addEventListener('click', e => {
  if (!e.target.closest('[data-del]')) return;
  const card = e.target.closest('.dec');
  cfg.logic.decoders.splice(+card.dataset.i, 1);
  saveConfig(); renderDecoders(); runDecode();
});
$$('[data-add]').forEach(b => b.addEventListener('click', () => {
  const t = b.dataset.add;
  const id = 'd' + Date.now().toString(36) + (decSeq++);
  const base = defaultDecoders().find(d => d.type === t);
  cfg.logic.decoders.push({ ...base, id, name: base.name + (cfg.logic.decoders.filter(d => d.type === t).length ? ' ' + (cfg.logic.decoders.filter(d => d.type === t).length + 1) : '') });
  saveConfig(); renderDecoders(); runDecode();
}));

async function runDecode(silent) {
  try {
    const r = await api('/api/logic/decode', { decoders: cfg.logic.decoders });
    const decIdx = id => cfg.logic.decoders.findIndex(d => d.id === id);
    G.rows = (r.rows || []).map(x => ({ ...x, color: DEC_COLORS[Math.max(0, decIdx(x.dec)) % DEC_COLORS.length] }));
    G.table = G.cap ? await api('/api/logic/table') : [];
    decTable.setItems(filterDec());
    waveLogic.resp = waveTL.resp = null;
    waveLogic.lanesDirty = waveTL.lanesDirty = true; waveLogic.dirty = waveTL.dirty = true;
  } catch (e) { if (!silent) toast(esc(e.message), 'err'); }
}
const runDecodeSoon = debounce(runDecode, 250);

// ============================================================== virtual lists
class VList {
  constructor(el, rowH, render, onClick, empty) {
    this.el = el; this.h = rowH; this.render = render; this.items = []; this.empty = empty;
    el.innerHTML = '<div class="vspacer"></div><div class="vrows"></div>';
    this.sp = el.firstChild; this.rows = el.lastChild;
    el.addEventListener('scroll', () => this.paint());
    el.addEventListener('click', e => { const r = e.target.closest('.vrow'); if (r) onClick(this.items[+r.dataset.i]); });
  }
  setItems(items, stickBottom) {
    const atBottom = this.el.scrollHeight - this.el.scrollTop - this.el.clientHeight < 30;
    this.items = items;
    this.sp.style.height = items.length * this.h + 'px';
    if (stickBottom && atBottom) this.el.scrollTop = this.el.scrollHeight;
    this.paint();
  }
  paint() {
    if (!this.items.length) { this.rows.innerHTML = `<div class="vempty">${this.empty}</div>`; return; }
    const a = Math.max(0, Math.floor(this.el.scrollTop / this.h) - 4);
    const b = Math.min(this.items.length, a + Math.ceil(this.el.clientHeight / this.h) + 8);
    let html = '';
    for (let i = a; i < b; i++) html += this.render(this.items[i], i).replace('<div class="vrow', `<div style="top:${i * this.h}px" data-i="${i}" class="vrow`);
    this.rows.innerHTML = html;
  }
}

function filterDec() {
  const q = $('#decFilter').value.trim().toLowerCase();
  const t = q ? G.table.filter(x => x[4].toLowerCase().includes(q) || (G.rows[x[0]] && G.rows[x[0]].name.toLowerCase().includes(q))) : G.table;
  $('#decCount').textContent = G.table.length ? `${fmtCount(t.length)} items` : '';
  return t;
}
const decTable = new VList($('#decTable'), 26, x => {
  const row = G.rows[x[0]] || { color: '#888', name: '' };
  return `<div class="vrow k${x[3]}"><span class="t">${G.cap ? fmtSec(waveLogic.sampleToSec(x[1]), 4) : ''}</span><span class="r" style="background:${row.color}" title="${esc(row.name)}"></span><span class="v">${esc(x[4])}</span></div>`;
}, x => {
  const s = waveLogic.sampleToSec(x[1]), e = waveLogic.sampleToSec(x[2]);
  waveLogic.centerOn((s + e) / 2, Math.max((e - s) * 14, 20 / G.cap.rate));
}, 'Decoded bytes appear here after a capture.');
$('#decFilter').addEventListener('input', debounce(() => decTable.setItems(filterDec()), 120));

const tlList = {
  dirty: true, last: 0,
  vl: new VList($('#tlList'), 26, l => `<div class="vrow ${l.dir === 'tx' ? 'out' : l.dir === 'sys' ? 'sys' : ''}"><span class="t">${clock(l.t).slice(3)}</span><span class="v">${esc((l.dir === 'tx' ? '→ ' : '') + l.text.replace(ANSI_RE, ''))}</span></div>`,
    l => { waveTL.centerOn((l.t - G.origin) / 1000); }, 'Serial lines will appear here.'),
  update() {
    if (!this.dirty || performance.now() - this.last < 250) return;
    if (!$('#page-timeline').classList.contains('active')) return;
    this.dirty = false; this.last = performance.now();
    const q = $('#tlFilter').value.trim().toLowerCase();
    const items = q ? SER.lines.filter(l => l.text.toLowerCase().includes(q)) : SER.lines;
    $('#tlCount').textContent = items.length ? fmtCount(items.length) + ' lines' : '';
    this.vl.setItems(items, true);
  },
};
$('#tlFilter').addEventListener('input', () => { tlList.dirty = true; tlList.last = 0; });

// ============================================================== measurements
const measure = debounce(async () => {
  const w = waveLogic;
  const box = $('#measGrid');
  if (!G.cap) { box.innerHTML = '<span>Channel</span><span>click a name</span>'; return; }
  const ch = w.sel >= 0 ? w.sel : 0;
  let a = w.v0, b = w.v1, src = 'visible range';
  if (w.ca != null && w.cb != null) { a = Math.min(w.ca, w.cb); b = Math.max(w.ca, w.cb); src = 'A → B'; }
  const off = capOffset(), r = G.cap.rate;
  $('#measCh').textContent = `· ${cfg.logic.names[ch]} · ${src}`;
  try {
    const m = await api(`/api/logic/measure?ch=${ch}&s=${(a - off) * r}&e=${(b - off) * r}`);
    box.innerHTML = `<span>Frequency</span><span>${m.freq ? fmtHz(m.freq) : '—'}</span>
      <span>Period</span><span>${m.period ? fmtSec(m.period, 4) : '—'}</span>
      <span>Duty (high)</span><span>${(m.duty * 100).toFixed(1)} %</span>
      <span>Edges</span><span>${fmtCount(m.edges)}</span>`;
  } catch { }
}, 150);

// ============================================================== logic toolbar
function activeWave() { return cfg.tab === 'timeline' ? waveTL : waveLogic; }
$('#btnZoomIn').addEventListener('click', () => waveLogic.zoom(0.5));
$('#btnZoomOut').addEventListener('click', () => waveLogic.zoom(2));
$('#btnFit').addEventListener('click', () => waveLogic.fit());
$('#tlZoomIn').addEventListener('click', () => { waveTL.zoom(0.5); });
$('#tlZoomOut').addEventListener('click', () => { waveTL.zoom(2); });
$('#tlFit').addEventListener('click', () => waveTL.fit());
$('#btnCursors').addEventListener('click', e => waveLogic.setCursorMode(!waveLogic.cursorMode, e.currentTarget));
$('#tlCursors').addEventListener('click', e => waveTL.setCursorMode(!waveTL.cursorMode, e.currentTarget));
$('#swFollow').addEventListener('change', e => {
  waveTL.following = e.target.checked;
  if (e.target.checked) { const span = Math.min(waveTL.v1 - waveTL.v0, 20) || 10; waveTL.v0 = 0; waveTL.v1 = span; }
});
$('#btnExport').addEventListener('click', () => { if (!G.cap) { toast('Capture something first', 'warn'); return; } showModal('#exportModal'); });
$('#expVcd').addEventListener('click', () => {
  const a = document.createElement('a');
  a.href = `/api/logic/export.vcd?token=${TOKEN}&names=${encodeURIComponent(cfg.logic.names.join(','))}`;
  a.download = ''; document.body.appendChild(a); a.click(); a.remove(); hideModals();
});
$('#expDec').addEventListener('click', () => {
  const rows = [['time_s', 'decoder', 'value', 'kind']];
  const kinds = ['data', 'address', 'control', 'ack', 'error'];
  for (const x of G.table) rows.push([waveLogic.sampleToSec(x[1]).toFixed(9), G.rows[x[0]].name, x[4], kinds[x[3]]]);
  const csv = rows.map(r => r.map(v => /[",\n]/.test(String(v)) ? `"${String(v).replace(/"/g, '""')}"` : v).join(',')).join('\r\n');
  download(`tracie_decoded_${Date.now()}.csv`, new Blob([csv], { type: 'text/csv' })); hideModals();
});
$('#expPng').addEventListener('click', () => {
  const w = waveLogic, d = DPR();
  const gw = w.gutter.offsetWidth, W = w.W, H = w.cv.height / d;
  const c = document.createElement('canvas');
  c.width = (gw + W) * d; c.height = (H + 30) * d;
  const ctx = c.getContext('2d'); ctx.scale(d, d);
  ctx.fillStyle = '#0c1018'; ctx.fillRect(0, 0, gw + W, H + 30);
  ctx.drawImage(w.ruler, gw, 0, W, 30); ctx.drawImage(w.cv, gw, 30, W, H);
  ctx.font = '600 12px Segoe UI, sans-serif'; ctx.textBaseline = 'middle';
  w.L.forEach(l => {
    let name = '', col = '#888';
    if (l.type === 'ch') { name = cfg.logic.names[l.ch]; col = CH_COLORS[l.ch]; }
    else if (l.type === 'dec') { name = G.rows[l.row].name; col = G.rows[l.row].color; }
    else if (l.type === 'ser') { name = 'Serial'; col = '#2dd4bf'; }
    ctx.fillStyle = col; ctx.fillRect(0, 30 + l.y + 5, 3, l.h - 10);
    ctx.fillStyle = '#e7edf6'; ctx.fillText(name, 12, 30 + l.y + l.h / 2);
  });
  ctx.fillStyle = '#5f6c80'; ctx.fillText('Tracie · nrmluvdev', 12, 15);
  c.toBlob(b => download(`tracie_${Date.now()}.png`, b)); hideModals();
});

// ============================================================== keyboard
window.addEventListener('keydown', e => {
  const typing = e.target.closest('input, select, textarea, [contenteditable="true"]');
  if (e.key === 'Escape') { hideModals(); return; }
  if (e.key === 'F1') { e.preventDefault(); showModal('#helpModal'); return; }
  if (e.ctrlKey && ['1', '2', '3'].includes(e.key)) { e.preventDefault(); setTab(['serial', 'logic', 'timeline'][+e.key - 1]); return; }
  if (e.ctrlKey && e.key.toLowerCase() === 'l') { e.preventDefault(); consoleUI.clear(); return; }
  if (typing || e.ctrlKey || e.altKey || e.metaKey) return;
  if (cfg.tab === 'serial') {
    if (e.key.length === 1 && !$('.modal:not(.hidden)')) $('#sendInput').focus();
    return;
  }
  const w = activeWave();
  switch (e.key) {
    case ' ': if (cfg.tab === 'logic') { e.preventDefault(); runCapture(); } break;
    case 'f': case 'F': w.fit(); break;
    case '+': case '=': w.zoom(0.5); break;
    case '-': case '_': w.zoom(2); break;
    case 'ArrowLeft': w.pan(-w.W * 0.2); w.follow(false); break;
    case 'ArrowRight': w.pan(w.W * 0.2); w.follow(false); break;
    case 'c': case 'C': w.setCursorMode(!w.cursorMode, cfg.tab === 'logic' ? $('#btnCursors') : $('#tlCursors')); break;
  }
});
window.addEventListener('resize', () => { moveTabInd(); waveLogic.resize(); waveTL.resize(); plot.dirty = true; });

// ============================================================== main loop
let statsDirty = true, lastPlot = 0;
function loop(ts) {
  consoleUI.flush();
  if (plot.legendDirty) plot.renderLegend();
  if (ts - plot.lastLegend > 150 && plot.order.length) { plot.lastLegend = ts; plot.updateLegendValues(); }
  if ($('#page-serial').classList.contains('active') && (ts - lastPlot > 33) && (plot.dirty || (!plot.paused && plot.order.length))) {
    lastPlot = ts; plot.dirty = false; plot.draw();
  }
  if (statsDirty) { statsDirty = false; $('#serialStats').textContent = `RX ${fmtBytes(SER.rx)} · TX ${fmtBytes(SER.tx)} · ${fmtCount(SER.lines.length)} lines`; }
  waveLogic.frame();
  waveTL.frame();
  tlList.update();
  requestAnimationFrame(loop);
}

// ============================================================== startup
async function init() {
  try { merge(cfg, await api('/api/config')); } catch { }
  if (!Array.isArray(cfg.logic.decoders)) cfg.logic.decoders = defaultDecoders();
  if (cfg.logic.names.length !== 8) cfg.logic.names = [0, 1, 2, 3, 4, 5, 6, 7].map(i => 'CH' + i);

  // reflect config in the UI
  renderBauds(); renderCmds(); renderDecoders(); renderTrigSum();
  $('#selDataBits').value = cfg.serial.dataBits; $('#selParity').value = cfg.serial.parity; $('#selStop').value = String(cfg.serial.stop);
  $('#swDTR').checked = cfg.serial.dtr; $('#swRTS').checked = cfg.serial.rts; $('#swReconnect').checked = cfg.serial.reconnect;
  $('#swTs').checked = cfg.serial.ts; $('#console').classList.toggle('no-ts', !cfg.serial.ts);
  $('#swAuto').checked = cfg.serial.auto; $('#swEcho').checked = cfg.serial.echo;
  $('#selEol').value = cfg.serial.eol.replace(/\r/g, '\\r').replace(/\n/g, '\\n');
  $$('#segWindow button').forEach(b => b.classList.toggle('active', +b.dataset.w === cfg.serial.plotWin));
  $('#btnPlotStack').classList.toggle('active', cfg.serial.plotStack);
  if (cfg.serial.split) $('#plotPanel').style.flex = `0 0 ${cfg.serial.split}px`;
  $('#rngPre').value = cfg.logic.pre; $('#preVal').textContent = cfg.logic.pre + '%';
  $('#swRepeat').checked = cfg.logic.repeat; $('#swOverlay').checked = cfg.logic.overlay;
  setTab(cfg.tab || 'serial');

  // live events: buffer until history has been replayed
  let buffered = [], ready = false;
  const es = new EventSource('/api/events?token=' + TOKEN);
  es.addEventListener('serial', m => { const evs = JSON.parse(m.data); if (ready) handleSerialEvents(evs); else buffered.push(evs); });
  es.addEventListener('serialState', m => { applySerialState(JSON.parse(m.data)); waveLogic.lanesDirty = waveTL.lanesDirty = true; });
  es.addEventListener('logicState', m => applyLogicState(JSON.parse(m.data)));
  es.onerror = () => { $('#chipSerial .lbl').textContent = 'Reconnecting…'; };

  await Promise.all([refreshPorts(false), refreshDevices()]);
  try {
    const hist = await api('/api/serial/history');
    handleSerialEvents(hist);
  } catch { }
  ready = true;
  buffered.forEach(handleSerialEvents); buffered = [];
  consoleUI.forceBottom = true;

  try { const info = await api('/api/logic/info'); if (info.has) { LOGIC.state.captureId = info.id; await loadCapture(); } } catch { }

  setInterval(() => { if (!SER.state.open && cfg.tab === 'serial') refreshPorts(true); }, 2500);
  requestAnimationFrame(loop);
  moveTabInd();
  setTimeout(moveTabInd, 300);
}
init();
