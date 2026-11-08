// DOM panels: toolbar, library, inspector, outliner, menus and dialogs.
import { S, PALETTE, hexToRgb } from './state.js';
import * as sc from './scene.js';
import { getGeom, whenBuilt } from './api.js';
import { Renderer } from './renderer.js';
import { lookAt, perspective, v3, round } from './math.js';

let A; // actions supplied by main.js
export function initUI(actions) { A = actions; }

const $ = (s, r = document) => r.querySelector(s);
const h = (tag, attrs = {}, ...kids) => {
  const e = document.createElement(tag);
  for (const [k, v] of Object.entries(attrs || {})) {
    if (v == null || v === false) continue;
    if (k.startsWith('on')) e.addEventListener(k.slice(2), v);
    else if (k === 'class') e.className = v;
    else if (k === 'html') e.innerHTML = v;
    else if (k === 'style') e.style.cssText = v;
    else e.setAttribute(k, v === true ? '' : v);
  }
  for (const c of kids.flat()) if (c != null && c !== false) e.append(c.nodeType ? c : document.createTextNode(c));
  return e;
};
export { h, $ };

export const ICON = {
  new: '<path d="M6 3h8l4 4v14H6z"/><path d="M14 3v4h4"/><path d="M12 11v6M9 14h6"/>',
  open: '<path d="M3 7h6l2 2h10v10H3z"/>',
  save: '<path d="M5 3h12l3 3v15H5z"/><path d="M8 3v6h8V3"/><rect x="8" y="13" width="8" height="6"/>',
  import: '<path d="M12 3v12M7 10l5 5 5-5"/><path d="M4 17v4h16v-4"/>',
  export: '<path d="M12 15V3M7 8l5-5 5 5"/><path d="M4 17v4h16v-4"/>',
  undo: '<path d="M9 14 4 9l5-5"/><path d="M4 9h11a5 5 0 0 1 0 10h-3"/>',
  redo: '<path d="m15 14 5-5-5-5"/><path d="M20 9H9a5 5 0 0 0 0 10h3"/>',
  move: '<path d="M12 3v18M3 12h18"/><path d="m9 6 3-3 3 3M9 18l3 3 3-3M6 9l-3 3 3 3M18 9l3 3-3 3"/>',
  rotate: '<path d="M20 12a8 8 0 1 1-2.3-5.6"/><path d="M20 4v5h-5"/>',
  scale: '<rect x="3" y="11" width="10" height="10"/><path d="M13 11l8-8M15 3h6v6"/>',
  group: '<rect x="3" y="3" width="11" height="11" rx="1"/><rect x="10" y="10" width="11" height="11" rx="1"/>',
  ungroup: '<rect x="3" y="3" width="9" height="9" rx="1"/><rect x="13" y="13" width="8" height="8" rx="1" stroke-dasharray="2 2"/>',
  wirie: '<rect x="6" y="6" width="12" height="12" rx="1.5"/><path d="M9 6V3M12 6V3M15 6V3M9 21v-3M12 21v-3M15 21v-3M6 9H3M6 15H3M21 9h-3M21 15h-3"/><path d="m12.8 8.5-2.3 3.7h3l-2.3 3.3"/>',
  hole: '<rect x="4" y="4" width="16" height="16" rx="2" stroke-dasharray="3 2"/><circle cx="12" cy="12" r="3.5"/>',
  align: '<path d="M4 3v18"/><rect x="7" y="6" width="12" height="4" rx="1"/><rect x="7" y="14" width="7" height="4" rx="1"/>',
  mirror: '<path d="M12 3v18" stroke-dasharray="2 2"/><path d="M9 6 3 18h6zM15 6l6 12h-6z"/>',
  dup: '<rect x="8" y="8" width="12" height="12" rx="2"/><path d="M16 8V5a1 1 0 0 0-1-1H5a1 1 0 0 0-1 1v10a1 1 0 0 0 1 1h3"/>',
  pattern: '<circle cx="6" cy="6" r="2.5"/><circle cx="18" cy="6" r="2.5"/><circle cx="6" cy="18" r="2.5"/><circle cx="18" cy="18" r="2.5"/>',
  del: '<path d="M4 7h16M10 11v6M14 11v6M6 7l1 13h10l1-13M9 7V4h6v3"/>',
  measure: '<path d="m3 17 14-14 4 4L7 21z"/><path d="m7 13 2 2M10 10l2 2M13 7l2 2"/>',
  section: '<path d="M12 3 21 8v8l-9 5-9-5V8z"/><path d="M3 12h18" stroke-dasharray="2 2"/>',
  overhang: '<path d="M4 20h7V10h9"/><path d="M14 10l6 6" stroke-dasharray="2 2"/>',
  check: '<rect x="3" y="3" width="18" height="18" rx="3"/><path d="M8 3v5h8V3"/><path d="m8.5 14 2.5 2.5 4.5-5"/>',
  settings: '<circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.7 1.7 0 0 0 .3 1.8l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-1.8-.3 1.7 1.7 0 0 0-1 1.5V21a2 2 0 1 1-4 0v-.1a1.7 1.7 0 0 0-1.1-1.5 1.7 1.7 0 0 0-1.8.3l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.7 1.7 0 0 0 .3-1.8 1.7 1.7 0 0 0-1.5-1H3a2 2 0 1 1 0-4h.1a1.7 1.7 0 0 0 1.5-1.1 1.7 1.7 0 0 0-.3-1.8l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.7 1.7 0 0 0 1.8.3H9a1.7 1.7 0 0 0 1-1.5V3a2 2 0 1 1 4 0v.1a1.7 1.7 0 0 0 1 1.5 1.7 1.7 0 0 0 1.8-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.7 1.7 0 0 0-.3 1.8V9a1.7 1.7 0 0 0 1.5 1H21a2 2 0 1 1 0 4h-.1a1.7 1.7 0 0 0-1.5 1z"/>',
  help: '<circle cx="12" cy="12" r="9"/><path d="M9.5 9a2.5 2.5 0 1 1 3.5 2.3c-.7.3-1 .9-1 1.7M12 17h.01"/>',
  drop: '<path d="M12 4v12M7 11l5 5 5-5"/><path d="M4 20h16"/>',
  eye: '<path d="M2 12s4-7 10-7 10 7 10 7-4 7-10 7S2 12 2 12z"/><circle cx="12" cy="12" r="3"/>',
  wire: '<path d="M12 3 21 8v8l-9 5-9-5V8z"/><path d="m3 8 9 5 9-5M12 13v8"/>',
  sketch: '<path d="M4 20 8 6l6 4 6-6"/><circle cx="4" cy="20" r="1.5"/><circle cx="8" cy="6" r="1.5"/><circle cx="14" cy="10" r="1.5"/><circle cx="20" cy="4" r="1.5"/>',
};
const ink = (hex) => { const n = parseInt((hex || '#888888').slice(1), 16); return ((n >> 16) * 0.299 + ((n >> 8) & 255) * 0.587 + (n & 255) * 0.114) > 160 ? '#111' : '#fff'; };
const svg = (name) => `<svg viewBox="0 0 24 24">${ICON[name] || ''}</svg>`;

// ---------------- toolbar ----------------
const TOOLBAR = [
  ['open', 'File', 'fileMenu', 'New, open, save and import', true],
  ['export', 'Export', 'exportMenu', 'Export for 3D printing (Ctrl+E)', true],
  ['wirie', 'Wirie', 'wirieMenu', 'Open a circuit board from Wirie and make a case for it', true],
  '|',
  ['undo', 'Undo', 'undo', 'Undo (Ctrl+Z)'],
  ['redo', 'Redo', 'redo', 'Redo (Ctrl+Y)'],
  '|',
  ['move', 'Move', 'tool:move', 'Move tool (W)'],
  ['rotate', 'Rotate', 'tool:rotate', 'Rotate tool (E)'],
  ['scale', 'Resize', 'tool:scale', 'Resize tool (R)'],
  '|',
  ['group', 'Group', 'group', 'Group: join parts; holes cut into solids (Ctrl+G)'],
  ['ungroup', 'Ungroup', 'ungroup', 'Ungroup (Ctrl+Shift+G)'],
  ['hole', 'Hole', 'toggleHole', 'Switch between solid and hole (H)'],
  '|',
  ['align', 'Align', 'alignTools', 'Align, auto-align and snap parts together', true],
  ['mirror', 'Mirror', 'mirrorMenu', 'Mirror / flip (M)', true],
  ['dup', 'Duplicate', 'duplicate', 'Duplicate (Ctrl+D)'],
  ['pattern', 'Pattern', 'patternDialog', 'Make a row, grid or circle of copies'],
  ['drop', 'Drop', 'drop', 'Drop onto the bed or the part below (D)'],
  ['del', 'Delete', 'delete', 'Delete (Del)'],
  '|',
  ['measure', 'Measure', 'measure', 'Measure distances (Q)'],
  ['section', 'View', 'viewMenu', 'Section view, overhangs, X-ray', true],
];
const TOOLBAR_RIGHT = [
  ['check', 'Print check', 'printCheck', 'Check your model before printing', false, 'primary'],
  ['settings', 'Settings', 'settings', 'Printer bed, grid and quality'],
  ['help', 'Help', 'help', 'Shortcuts and tips (?)'],
];

export function renderToolbar() {
  const mk = (el, list) => {
    el.innerHTML = '';
    for (const b of list) {
      if (b === '|') { el.append(h('div', { class: 'sep' })); continue; }
      const [ic, label, act, title, caret, cls] = b;
      const btn = h('button', { class: 'tb ' + (cls || ''), title, 'data-act': act, html: svg(ic) + `<span>${label}</span>` + (caret ? '<i class="caret">▼</i>' : '') });
      btn.addEventListener('click', (e) => A.run(act, e));
      el.append(btn);
    }
  };
  mk($('#tools'), TOOLBAR);
  mk($('#tools-right'), TOOLBAR_RIGHT);
  refreshToolbar();
}

export function refreshToolbar() {
  const set = (act, on) => document.querySelectorAll(`[data-act="${act}"]`).forEach((b) => b.classList.toggle('on', !!on));
  for (const t of ['move', 'rotate', 'scale']) set('tool:' + t, S.tool === t);
  set('measure', S.mode === 'measure');
  set('viewMenu', S.view.section || S.view.overhang || S.view.wire);
  const n = S.sel.size;
  const one = n === 1 ? sc.node([...S.sel][0]) : null;
  const dis = (act, d) => document.querySelectorAll(`[data-act="${act}"]`).forEach((b) => { b.disabled = !!d; });
  dis('group', n < 2); dis('ungroup', !(one && one.type === 'group') && ![...S.sel].some((id) => sc.node(id)?.type === 'group'));
  ['toggleHole', 'mirrorMenu', 'duplicate', 'patternDialog', 'delete', 'drop'].forEach((a) => dis(a, n === 0));

  dis('undo', !A.canUndo()); dis('redo', !A.canRedo());
  $('#ol-group').disabled = n < 2;
  $('#ol-ungroup').disabled = !(one && one.type === 'group');
  const hb = document.querySelector('[data-act="toggleHole"] span');
  if (hb) hb.textContent = one && sc.isHole(one) ? 'Solid' : 'Hole';
}

// ---------------- library ----------------
let libCat = 'All';
const CATS = ['All', 'Basic', 'Holes', 'Hardware', 'Mechanical', 'Gears', 'Hoses', 'Pegboard', 'Electronics', 'Sketch'];
const thumbs = {};
let thumbR = null;

export function showCategory(c) { libCat = c; $('#libsearch').value = ''; renderLibrary(); }

export function renderLibrary() {
  const cats = $('#libcats');
  cats.innerHTML = '';
  for (const c of CATS) cats.append(h('button', { class: 'cat' + (c === libCat ? ' on' : ''), onclick: () => { libCat = c; renderLibrary(); } }, c));
  const q = $('#libsearch').value.trim().toLowerCase();
  const grid = $('#libgrid');
  grid.innerHTML = '';
  delete grid.dataset.mrDone; delete grid.dataset.grDone;
  let lastCat = null;
  const items = S.catalog.filter((d) => (libCat === 'All' || d.category === libCat) && (!q || (d.name + ' ' + d.tags + ' ' + d.category + ' ' + d.desc).toLowerCase().includes(q)));
  const order = CATS.slice(1);
  items.sort((a, b) => order.indexOf(a.category) - order.indexOf(b.category));
  for (const d of items) {
    if (libCat === 'All' && d.category !== lastCat && !q) { grid.append(h('div', { class: 'libhead' }, d.category)); lastCat = d.category; }
    if (d.category === 'Gears' && grid.dataset.grDone !== '1' && !q) {
      grid.dataset.grDone = '1';
      grid.append(h('div', { class: 'tile build', title: 'Work out which gears give the ratio you need, then add them', onclick: () => A.gearCalc() }, h('div', { class: 'ph plus' }, '⚙'), h('span', {}, 'Gear ratio calculator')));
    }
    const tile = h('div', { class: 'tile' + (d.hole ? ' hole' : ''), draggable: 'true', title: d.desc, 'data-type': d.type });
    tile.append(thumbs[d.type] ? h('img', { src: thumbs[d.type] }) : h('div', { class: 'ph' }), h('span', {}, d.name));
    tile.addEventListener('click', () => A.place(d.type));
    tile.addEventListener('dragstart', (e) => {
      window.__dragType = d.type;
      e.dataTransfer.setData('application/x-makie', d.type);
      e.dataTransfer.effectAllowed = 'copy';
      const img = new Image(); img.src = thumbs[d.type] || ''; e.dataTransfer.setDragImage(img, 29, 29);
    });
    tile.addEventListener('dragend', () => { window.__dragType = null; A.endDrag(); });
    grid.append(tile);
  }
  if (!items.length) grid.append(h('div', { class: 'note', style: 'grid-column:1/-1' }, 'No shapes match your search.'));
}

// render small preview images of every shape with our own renderer
const THUMBS = 'partsmith.thumbs.v3';
export async function makeThumbs(version) {
  try {
    const cached = JSON.parse(localStorage.getItem(THUMBS) || 'null');
    if (cached && cached.v === version && S.catalog.every((d) => cached.t[d.type])) { Object.assign(thumbs, cached.t); renderLibrary(); return; }
  } catch {}
  const c = document.createElement('canvas');
  c.width = c.height = 116;
  c.style.cssText = 'width:116px;height:116px;position:fixed;left:-500px;top:0';
  document.body.append(c);
  try { thumbR = new Renderer(c, { preserve: true }); } catch { return; }
  const dark = document.body.classList.contains('dark');
  for (const d of S.catalog) {
    const n = sc.makeNode(d.type);
    n.id = 'thumb_' + d.type;
    const e = await whenBuilt(n);
    if (!e || !e.tris) continue;
    const g = thumbR.createMesh(e.positions, e.normals);
    const ctr = v3.mul(v3.add(e.lo, e.hi), 0.5), r = Math.max(1, v3.dist(e.lo, e.hi) / 2);
    const dir = v3.norm([0.62, -0.9, 0.6]);
    const eye = v3.add(ctr, v3.mul(dir, r * 2.9));
    thumbR.gl.viewport(0, 0, 116, 116);
    thumbR.setCamera(lookAt(eye, ctr, [0, 0, 1]), perspective(0.7, 1, r * 0.2, r * 10), eye);
    thumbR.clear([0, 0, 0, 0]);
    const col = d.hole ? [0.6, 0.65, 0.75] : hexToRgb(d.color);
    thumbR.drawMesh(g, [1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1], { color: col, hole: d.hole, opacity: d.hole ? 0.85 : 1, noOverhang: true });
    thumbs[d.type] = c.toDataURL();
    thumbR.free(g);
    const ph = document.querySelector(`.tile[data-type="${d.type}"] .ph`);
    if (ph) ph.replaceWith(h('img', { src: thumbs[d.type] }));
  }
  c.remove();
  try { localStorage.setItem(THUMBS, JSON.stringify({ v: version, t: thumbs })); } catch {}
}

// ---------------- number inputs ----------------
export function evalNum(s) {
  s = String(s).replace(/,/g, '.').trim();
  if (!/^[-+*/().\d\s e]+$/i.test(s)) return NaN;
  try { const v = Function('"use strict";return (' + s + ')')(); return typeof v === 'number' && isFinite(v) ? v : NaN; } catch { return NaN; }
}

function numInput({ value, step = 1, min, max, axis, onCommit, disabled, unit, key }) {
  const wrap = h('div', { class: axis ? 'num' : 'unitwrap' });
  const inp = h('input', { type: 'text', inputmode: 'decimal', value: fmtNum(value), disabled, 'data-k': key || null, spellcheck: 'false' });
  const commit = (v) => {
    if (!isFinite(v)) { inp.value = fmtNum(value); return; }
    if (min != null) v = Math.max(min, v);
    if (max != null) v = Math.min(max, v);
    value = v; inp.value = fmtNum(v); onCommit(v);
  };
  inp.addEventListener('keydown', (e) => {
    e.stopPropagation();
    if (e.key === 'Enter') { commit(evalNum(inp.value)); inp.select(); }
    if (e.key === 'Escape') { inp.value = fmtNum(value); inp.blur(); }
    if (e.key === 'ArrowUp' || e.key === 'ArrowDown') {
      e.preventDefault();
      const s = step * (e.shiftKey ? 10 : e.altKey ? 0.1 : 1) * (e.key === 'ArrowUp' ? 1 : -1);
      commit(round((evalNum(inp.value) || 0) + s, 4));
    }
  });
  inp.addEventListener('blur', () => { const v = evalNum(inp.value); if (v !== value) commit(v); });
  inp.addEventListener('focus', () => setTimeout(() => inp.select(), 0));
  if (axis) {
    const lab = h('span', { class: 'ax ' + axis.toLowerCase() }, axis);
    // drag the axis letter to scrub the value
    lab.addEventListener('pointerdown', (e) => {
      if (disabled) return;
      e.preventDefault();
      const x0 = e.clientX, v0 = value;
      const mv = (ev) => { const v = round(v0 + Math.round((ev.clientX - x0) / 4) * step, 4); inp.value = fmtNum(v); };
      const up = () => { window.removeEventListener('pointermove', mv); window.removeEventListener('pointerup', up); commit(evalNum(inp.value)); };
      window.addEventListener('pointermove', mv); window.addEventListener('pointerup', up);
    });
    wrap.append(lab);
  }
  wrap.append(inp);
  if (unit && !axis) wrap.append(h('span', {}, unit));
  return wrap;
}
const fmtNum = (v) => (v == null || !isFinite(v) ? '' : String(round(v, 3)));

function helpDot(text) { return text ? h('span', { class: 'help', title: text }, '?') : null; }

function showIf(rule, params) {
  if (!rule) return true;
  return rule.split(';').filter(Boolean).every((r) => {
    const m = r.match(/^(\w+)(!?=)(.*)$/);
    if (!m) return true;
    const v = String(params[m[1]] ?? '');
    return m[2] === '=' ? v === m[3] : v !== m[3];
  });
}

// ---------------- inspector ----------------
export function renderInspector() {
  const el = $('#inspector');
  const ids = [...S.sel].filter((id) => sc.node(id));
  el.innerHTML = '';
  if (!ids.length) { el.append(emptyInspector()); return; }
  if (ids.length > 1) { el.append(...multiInspector(ids)); return; }
  const n = sc.node(ids[0]);
  if (n.type === 'wirieboard') { A.wirieInspector(el, n); return; }
  const def = S.byType[n.type];
  const parent = sc.parentOf(n.id);

  const title = h('div', { class: 'insp-title' }, h('input', { value: n.name || '', spellcheck: 'false', onchange: (e) => A.update(n.id, { name: e.target.value }, 'Rename') , onkeydown: (e) => e.stopPropagation() }));
  el.append(title);
  el.append(h('div', { class: 'insp-sub' }, (def ? def.category + ' · ' + def.name : n.type === 'group' ? `Group of ${n.children.length}` : 'Imported model') + (parent ? ` · inside “${parent.name}”` : '')));

  // solid / hole
  const hole = sc.isHole(n);
  const seg = h('div', { class: 'seg' },
    h('button', { class: !hole ? 'on' : '', onclick: () => A.setHole([n.id], false), title: 'A solid part that gets printed' }, h('i', { class: 'chip', style: `color:${n.color}` }), 'Solid'),
    h('button', { class: hole ? 'on holebtn' : 'holebtn', onclick: () => A.setHole([n.id], true), title: 'A hole cuts away from solids it is grouped with' }, h('i', { class: 'chip h' }), 'Hole'));
  el.append(seg);
  if (!hole) el.append(colorPicker(n));
  if (hole && !parent) el.append(h('div', { class: 'callout tip', html: 'This is a <b>hole</b>. Select it together with a solid and press <b>Group</b> to cut it out.' }));

  if (n.type === 'group') {
    el.append(h('div', { class: 'group-title' }, 'Group'));
    el.append(h('div', { class: 'seg' },
      h('button', { class: n.mode !== 'intersect' ? 'on' : '', onclick: () => A.update(n.id, { mode: '' }, 'Group mode'), title: 'Join solids and cut holes (normal)' }, 'Combine'),
      h('button', { class: n.mode === 'intersect' ? 'on' : '', onclick: () => A.update(n.id, { mode: 'intersect' }, 'Group mode'), title: 'Keep only where solids overlap' }, 'Keep overlap')));
    el.append(h('div', { class: 'note', html: `Solids are joined together and <b>holes are cut out</b>. Double-click the group, or pick a part in the list below, to edit what is inside.` }));
  }

  transformFields(el, n, def);

  if (def && def.params.length) {
    el.append(h('div', { class: 'group-title' }, 'Shape settings'));
    for (const p of def.params) {
      if (!showIf(p.showIf, n.params)) continue;
      el.append(paramField(n, p));
    }
  }
  const ge = getGeom(n);
  if (ge && ge.info) {
    el.append(h('div', { class: 'group-title' }, /Major Ø/.test(ge.info) ? 'Thread specification' : 'Details'));
    el.append(h('pre', { class: 'spec' }, ge.info));
  }
  if (def && n.type !== 'group') el.append(h('div', { class: 'note' }, def.desc));

  if (!hole && (n.type === 'mesh' || n.type === 'group')) {
    el.append(h('button', { class: 'btn primary', style: 'width:100%;justify-content:center;margin-top:12px', title: 'Makie measures this model and makes mounts, a case, a lid or a bezel that fit it',
      onclick: () => A.scanModel(n.id), html: '✨ ' + (n.type === 'mesh' ? 'Make parts for this model…' : 'Make mounts / case for this part…') }));
  }
  el.append(h('div', { class: 'group-title' }, 'Actions'));
  const acts = h('div', { class: 'btnrow' });
  const b = (label, icon, fn, cls = '') => acts.append(h('button', { class: 'btn small ' + cls, onclick: fn, html: (icon ? svg(icon) : '') + label }));
  b('Duplicate', 'dup', () => A.run('duplicate'));
  b('Mirror', 'mirror', (e) => A.run('mirrorMenu', e));
  b('Drop to bed', 'drop', () => A.run('drop'));
  b('Lay flat…', 'drop', () => A.layFlat(n.id));
  b('Snap to…', 'align', () => A.snapTogether());
  if (!parent && !hole) b('Add motion joint…', 'rotate', () => { document.querySelector('#modes [data-ws=motion]').click(); A.addJoint(); });
  if (!hole) b('Split…', 'section', () => A.splitDialog(n.id));
  if (n.type === 'group') b('Ungroup', 'ungroup', () => A.run('ungroup'));
  if (n.type === 'sketch') b('Edit outline…', 'sketch', () => A.editSketch(n.id), 'primary');
  if (!hole && n.type !== 'group' && !def?.fixed) b('Add screw holes…', 'hole', () => A.cornerHoles(n.id));
  if (['bolt', 'nut', 'sqnut', 'rod', 'washer'].includes(n.type)) b('Add matching hole', 'hole', () => A.matchingHole(n.id));
  if (!hole && n.type !== 'group') b('Make matching part…', 'pattern', () => A.matchingPart(n.id), def?.category === 'Gears' || ['pbolt', 'pnut', 'pbayonet'].includes(n.type) ? 'primary' : '');
  if (def?.category === 'Gears') b('Gear calculator…', 'measure', () => A.gearCalc());
  const threaded = (n.type === 'cylinder' || n.type === 'tscrew' || n.type === 'tbore') && n.params?.thread === 'Threaded';
  if (threaded) b(hole ? 'Make matching screw' : 'Make matching threaded hole', 'pattern', () => A.matchingThread(n.id), 'primary');
  if (n.type === 'threadhole') b('Make matching screw', 'pattern', () => A.matchingThread(n.id), 'primary');
  if ((n.type === 'cylinder' || n.type === 'tube') && n.params?.thread === 'None') b('Add thread', 'pattern', () => A.setParam(n.id, 'thread', n.type === 'tube' ? 'Inside' : 'Threaded'));
  if (parent) b('Take out of group', 'ungroup', () => A.reparent(n.id, null));
  b(n.locked ? 'Unlock' : 'Lock', '', () => A.update(n.id, { locked: !n.locked }, 'Lock'));
  b('Delete', 'del', () => A.run('delete'), 'danger');
  el.append(acts);
}

function emptyInspector() {
  return h('div', { class: 'empty-insp', html: `<div class="big">⬚</div><b>Nothing selected</b><p>Click a part to edit it. Add shapes from the left — click one, then click on the bed.</p><p class="note">Tip: <b>holes</b> are shapes set to “Hole”. Group a hole with a solid to cut it out.</p>` });
}

function colorPicker(n) {
  const wrap = h('div');
  const slots = h('div', { class: 'slots' }, h('span', { class: 'note', style: 'margin:0 4px 0 0' }, 'Filament'));
  (S.filaments || []).forEach((f, i) => slots.append(h('div', { class: 'slot' + (n.slot === i + 1 ? ' on' : ''), style: `background:${f.color};color:${ink(f.color)};text-shadow:none`, title: `Slot ${i + 1}: ${f.name}`, onclick: () => A.setSlot([n.id], i + 1) }, String(i + 1))));
  slots.append(h('button', { class: 'mini', onclick: () => A.filamentsDialog(), title: 'Edit filament slots (AMS)' }, 'Edit…'));
  wrap.append(slots);
  const row = h('div', { class: 'colors' });
  for (const c of PALETTE) row.append(h('div', { class: 'c' + (n.color === c && !n.slot ? ' on' : ''), style: `background:${c}`, title: c + ' (display colour — exported to the closest filament)', onclick: () => A.update(n.id, { color: c, slot: 0 }, 'Colour') }));
  row.append(h('input', { type: 'color', value: n.color || '#888888', title: 'Custom colour', onchange: (e) => A.update(n.id, { color: e.target.value, slot: 0 }, 'Colour') }));
  wrap.append(row);
  return wrap;
}

export function transformFields(el, n, def) {
  el.append(h('div', { class: 'group-title' }, 'Position & size', h('span', { class: 'help', title: 'Values are in millimetres. You can type maths like 25.4*2 or 10+3.' }, '?')));
  const row = (label, vals, keyBase, step, onSet, opts = {}) => {
    const r = h('div', { class: 'xyz' }, h('label', {}, label));
    ['X', 'Y', 'Z'].forEach((ax, i) => r.append(numInput({ value: vals[i], step, axis: ax, key: keyBase + i, disabled: opts.disabled, onCommit: (v) => onSet(i, v), min: opts.min })));
    el.append(r);
  };
  const box = sc.worldBox(n.id);
  const P = sc.parentOf(n.id);
  // position: shown as the bottom-centre of the part's box (world), like Tinkercad
  const pos = box ? [(box[0][0] + box[1][0]) / 2, (box[0][1] + box[1][1]) / 2, box[0][2]] : n.pos;
  row('Position', pos.map((x) => round(x, 3)), 'p', S.settings.grid || 1, (i, v) => A.setPosition(n.id, i, v));
  row('Rotation', n.rot, 'r', S.settings.rotSnap || 15, (i, v) => { const r = [...n.rot]; r[i] = v; A.update(n.id, { rot: r }, 'Rotate'); });
  const sz = sizeOf(n);
  row('Size', sz, 's', S.settings.grid || 1, (i, v) => A.setSize(n.id, i, v), { disabled: def?.fixed, min: 0.01 });
  if (def?.fixed) el.append(h('div', { class: 'note' }, 'This is a real-world part: change its size with the settings below.'));
  void P;
}

export function sizeOf(n) {
  const e = getGeom(n);
  if (!e || !e.tris) return [0, 0, 0];
  const ext = v3.sub(e.hi, e.lo);
  return ext.map((x, i) => round(Math.abs(x * (n.scale[i] || 1)), 3));
}

// update numbers in place during live drags (keeps the panel stable)
export function refreshInspectorValues() {
  const ids = [...S.sel];
  if (ids.length !== 1) return;
  const n = sc.node(ids[0]);
  if (!n) return;
  const box = sc.worldBox(n.id);
  const pos = box ? [(box[0][0] + box[1][0]) / 2, (box[0][1] + box[1][1]) / 2, box[0][2]] : n.pos;
  const sz = sizeOf(n);
  const set = (k, v) => { const i = document.querySelector(`#inspector input[data-k="${k}"]`); if (i && document.activeElement !== i) i.value = fmtNum(round(v, 3)); };
  for (let i = 0; i < 3; i++) { set('p' + i, pos[i]); set('r' + i, n.rot[i]); set('s' + i, sz[i]); }
}

function paramField(n, p) {
  const val = n.params?.[p.key] ?? p.def;
  const set = (v) => A.setParam(n.id, p.key, v);
  const lab = h('label', {}, p.label, helpDot(p.help));
  let ctl;
  switch (p.type) {
    case 'hidden':
      return h('span');
    case 'select': {
      ctl = h('select', { onchange: (e) => set(e.target.value), onkeydown: (e) => e.stopPropagation() });
      if (p.options.some((o) => o.includes(' — '))) {
        // long grouped lists (boards): one optgroup per family
        const groups = new Map();
        for (const o of p.options) { const [g, nm] = o.split(' — '); if (!groups.has(g)) groups.set(g, []); groups.get(g).push([o, nm]); }
        for (const [g, items] of groups) ctl.append(h('optgroup', { label: g }, ...items.map(([o, nm]) => h('option', { value: o, selected: String(val) === o }, nm))));
      } else ctl.append(...p.options.map((o) => h('option', { value: o, selected: String(val) === o }, o)));
      if (n.params?.spec && p.key === 'board') { ctl = h('div', { class: 'note' }, 'Scanned from your imported model'); }
      break;
    }
    case 'bool':
      ctl = h('button', { class: 'toggle' + (val ? ' on' : ''), onclick: () => set(!val), title: val ? 'On' : 'Off' });
      break;
    case 'text':
      ctl = h('textarea', { rows: String(Math.min(4, String(val).split('\n').length)), spellcheck: 'false', onkeydown: (e) => e.stopPropagation(), oninput: debounce((e) => set(e.target.value), 250) });
      ctl.value = val;
      return h('div', { class: 'field full' }, lab, ctl);
    case 'points':
      return h('div', { class: 'field' }, lab, h('button', { class: 'btn small primary', onclick: () => A.editSketch(n.id) }, `Edit outline (${(val || []).length} pts)`));
    default:
      ctl = numInput({ value: val, step: p.step || 1, min: p.min, max: p.max || undefined, unit: p.unit, onCommit: set });
  }
  return h('div', { class: 'field' }, lab, ctl);
}

function multiInspector(ids) {
  const out = [];
  const nodes = ids.map(sc.node);
  out.push(h('div', { class: 'insp-title' }, h('b', { style: 'font-size:15px;padding:3px 6px' }, `${ids.length} parts selected`)));
  const allHole = nodes.every(sc.isHole), anyHole = nodes.some(sc.isHole);
  out.push(h('div', { class: 'seg' },
    h('button', { class: !anyHole ? 'on' : '', onclick: () => A.setHole(ids, false) }, h('i', { class: 'chip' }), 'Solid'),
    h('button', { class: allHole ? 'on holebtn' : 'holebtn', onclick: () => A.setHole(ids, true) }, h('i', { class: 'chip h' }), 'Hole')));
  out.push(h('div', { class: 'slots' }, h('span', { class: 'note', style: 'margin:0 4px 0 0' }, 'Filament'), ...(S.filaments || []).map((f, i) => h('div', { class: 'slot', style: `background:${f.color};color:${ink(f.color)};text-shadow:none`, title: f.name, onclick: () => A.setSlot(ids, i + 1) }, String(i + 1)))));
  out.push(h('button', { class: 'btn primary', style: 'width:100%;justify-content:center', onclick: () => A.run('group'), html: svg('group') + 'Group these parts' }));
  if (anyHole && !allHole) out.push(h('div', { class: 'callout tip', html: 'Grouping will <b>cut the holes</b> out of the solids.' }));
  out.push(h('div', { class: 'group-title' }, 'Align'));
  out.push(h('button', { class: 'btn small primary', style: 'margin-bottom:8px', onclick: () => A.autoAlign(ids), title: 'Moves the other parts so they line up exactly with the first one you selected' }, '✨ Auto-align to “' + (sc.node(ids[0])?.name || 'first') + '”'));
  out.push(alignGrid(ids));
  out.push(h('div', { class: 'group-title' }, 'Distribute evenly'));
  out.push(h('div', { class: 'btnrow' }, ...['X', 'Y', 'Z'].map((a, i) => h('button', { class: 'btn small', onclick: () => A.distribute(ids, i) }, 'Along ' + a))));
  out.push(h('div', { class: 'group-title' }, 'Actions'));
  out.push(h('div', { class: 'btnrow' },
    h('button', { class: 'btn small', onclick: () => A.run('duplicate'), html: svg('dup') + 'Duplicate' }),
    h('button', { class: 'btn small', onclick: (e) => A.run('mirrorMenu', e), html: svg('mirror') + 'Mirror' }),
    h('button', { class: 'btn small', onclick: () => A.run('drop'), html: svg('drop') + 'Drop' }),
    h('button', { class: 'btn small danger', onclick: () => A.run('delete'), html: svg('del') + 'Delete' })));
  return out;
}

export function alignGrid(ids) {
  const g = h('div', { class: 'align-grid' });
  ['X', 'Y', 'Z'].forEach((ax, i) => {
    g.append(h('b', {}, ax));
    const names = i === 2 ? ['Bottom', 'Middle', 'Top'] : i === 0 ? ['Left', 'Centre', 'Right'] : ['Front', 'Centre', 'Back'];
    names.forEach((nm, k) => g.append(h('button', { class: 'mini', onclick: () => A.align(ids, i, k) }, nm)));
  });
  return g;
}

// ---------------- outliner ----------------
let dragId = null;
export function renderOutliner() {
  const tree = $('#tree');
  tree.innerHTML = '';
  if (!S.nodes.length) { tree.append(h('div', { class: 'empty-tree' }, 'Your parts will be listed here.')); return; }
  const rows = (nodes, depth) => {
    for (const n of [...nodes].reverse()) {
      const hole = sc.isHole(n);
      const row = h('div', { class: 'row' + (S.sel.has(n.id) ? ' sel' : '') + (n.hidden ? ' hidden-node' : ''), draggable: 'true', 'data-id': n.id, style: `padding-left:${6 + depth * 16}px` });
      const tw = h('span', { class: 'tw' }, n.type === 'group' ? (n.expanded ? '▾' : '▸') : '');
      tw.addEventListener('click', (e) => { e.stopPropagation(); n.expanded = !n.expanded; renderOutliner(); });
      const nm = h('span', { class: 'nm' }, n.name || n.type);
      nm.addEventListener('dblclick', (e) => {
        e.stopPropagation();
        const inp = h('input', { value: n.name || '' });
        nm.innerHTML = ''; nm.append(inp); inp.focus(); inp.select();
        const done = () => A.update(n.id, { name: inp.value || n.name }, 'Rename');
        inp.addEventListener('keydown', (ev) => { ev.stopPropagation(); if (ev.key === 'Enter') done(); if (ev.key === 'Escape') renderOutliner(); });
        inp.addEventListener('blur', done);
      });
      row.append(tw, h('span', { class: 'sw' + (hole ? ' hole' : ''), style: hole ? '' : `background:${n.color}` }), nm,
        h('span', { class: 'tag' }, n.type === 'group' ? `${n.children.length}` : ''),
        h('button', { class: 'ic' + (n.locked ? ' on' : ''), title: n.locked ? 'Unlock' : 'Lock', onclick: (e) => { e.stopPropagation(); A.update(n.id, { locked: !n.locked }, 'Lock'); } }, n.locked ? '🔒' : '🔓'),
        h('button', { class: 'ic' + (n.hidden ? ' on' : ''), title: n.hidden ? 'Show' : 'Hide', onclick: (e) => { e.stopPropagation(); A.update(n.id, { hidden: !n.hidden }, n.hidden ? 'Show' : 'Hide'); } }, n.hidden ? '◌' : '◉'));
      row.addEventListener('click', (e) => A.select(n.id, e.shiftKey || e.ctrlKey || e.metaKey ? 'toggle' : 'set'));
      row.addEventListener('contextmenu', (e) => { e.preventDefault(); if (!S.sel.has(n.id)) A.select(n.id, 'set'); A.context(e, n.id); });
      // drag & drop grouping
      row.addEventListener('dragstart', (e) => { dragId = n.id; e.dataTransfer.setData('text/plain', n.id); e.dataTransfer.effectAllowed = 'move'; });
      row.addEventListener('dragend', () => { dragId = null; clearDrop(); });
      row.addEventListener('dragover', (e) => {
        if (!dragId || dragId === n.id || sc.isDescendant(n.id, dragId)) return;
        e.preventDefault(); e.stopPropagation();
        const r = row.getBoundingClientRect(), y = (e.clientY - r.top) / r.height;
        clearDrop();
        row.classList.add(y < 0.25 ? 'drop-before' : y > 0.75 ? 'drop-after' : 'drop-in');
      });
      row.addEventListener('dragleave', () => row.classList.remove('drop-before', 'drop-after', 'drop-in'));
      row.addEventListener('drop', (e) => {
        e.preventDefault(); e.stopPropagation();
        const where = row.classList.contains('drop-in') ? 'in' : row.classList.contains('drop-before') ? 'before' : 'after';
        clearDrop();
        if (dragId) A.treeDrop(dragId, n.id, where);
      });
      tree.append(row);
      if (n.type === 'group' && n.expanded) rows(n.children, depth + 1);
    }
  };
  rows(S.nodes, 0);
  tree.ondragover = (e) => { if (dragId) { e.preventDefault(); tree.classList.add('drop-root'); } };
  tree.ondragleave = (e) => { if (e.target === tree) tree.classList.remove('drop-root'); };
  tree.ondrop = (e) => { e.preventDefault(); clearDrop(); if (dragId) A.treeDrop(dragId, null, 'root'); };
}
function clearDrop() {
  document.querySelectorAll('.drop-before,.drop-after,.drop-in').forEach((r) => r.classList.remove('drop-before', 'drop-after', 'drop-in'));
  $('#tree').classList.remove('drop-root');
}

// ---------------- menus & modals ----------------
export function menu(e, items) {
  const m = $('#menu');
  m.innerHTML = '';
  for (const it of items) {
    if (it === '-') { m.append(h('hr')); continue; }
    m.append(h('button', { disabled: it.disabled, onclick: () => { hideMenu(); it.run(); } }, h('span', {}, it.label), it.key ? h('span', { class: 'k' }, it.key) : null));
  }
  m.classList.remove('hidden');
  let x = e.clientX, y = e.clientY;
  if (e.currentTarget && e.currentTarget.getBoundingClientRect && e.type === 'click') { const r = e.currentTarget.getBoundingClientRect(); x = r.left; y = r.bottom + 4; }
  const mw = m.offsetWidth, mh = m.offsetHeight;
  m.style.left = Math.min(x, innerWidth - mw - 6) + 'px';
  m.style.top = Math.min(y, innerHeight - mh - 6) + 'px';
  setTimeout(() => window.addEventListener('pointerdown', closer, true), 0);
}
function closer(e) { if (!$('#menu').contains(e.target)) hideMenu(); }
export function hideMenu() { $('#menu').classList.add('hidden'); window.removeEventListener('pointerdown', closer, true); }

export function modal(title, body, buttons = [], opts = {}) {
  const root = $('#modal-root');
  const bd = h('div', { class: 'backdrop' });
  const box = h('div', { class: 'modal' + (opts.wide ? ' wide' : '') });
  box.append(h('h3', {}, title));
  if (typeof body === 'string') box.append(h('div', { html: body })); else box.append(body);
  const close = () => { bd.remove(); document.removeEventListener('keydown', key, true); opts.onClose?.(); };
  const acts = h('div', { class: 'actions' });
  const fire = (b) => {
    try { if (b.run?.() !== false) close(); }
    catch (err) { console.error(err); close(); import('./state.js').then((m) => m.toast('Something went wrong: ' + err.message, 'error', 6000)); }
  };
  for (const b of buttons) acts.append(h('button', { class: b.primary ? 'primary' : 'btn', onclick: () => fire(b) }, b.label));
  if (buttons.length) box.append(acts);
  bd.append(box);
  bd.addEventListener('pointerdown', (e) => { if (e.target === bd && !opts.sticky) close(); });
  const key = (e) => {
    if (e.key === 'Escape') { e.stopPropagation(); close(); }
    if (e.key === 'Enter' && e.target.tagName !== 'TEXTAREA') { const p = buttons.find((b) => b.primary); if (p) { e.preventDefault(); fire(p); } }
    e.stopPropagation();
  };
  document.addEventListener('keydown', key, true);
  root.append(bd);
  return { box, close };
}

// tiny form builder for dialogs
export function form(fields, onChange) {
  const el = h('div');
  const vals = {};
  const changed = () => onChange?.(vals);
  const draw = () => {
    el.innerHTML = '';
    for (const f of fields) {
      if (f.showIf && !f.showIf(vals)) continue;
      if (f.type === 'note') { el.append(h('div', { class: 'note', html: f.html })); continue; }
      vals[f.key] ??= f.def;
      let ctl;
      if (f.type === 'select') {
        ctl = h('select', { onchange: (e) => { vals[f.key] = e.target.value; draw(); changed(); }, onkeydown: (e) => e.stopPropagation() }, ...f.options.map((o) => h('option', { value: o, selected: vals[f.key] === o }, o)));
      } else if (f.type === 'bool') {
        ctl = h('button', { class: 'toggle' + (vals[f.key] ? ' on' : ''), onclick: () => { vals[f.key] = !vals[f.key]; draw(); changed(); } });
      } else if (f.type === 'text') {
        ctl = h('input', { type: 'text', value: vals[f.key], spellcheck: 'false', onkeydown: (e) => e.stopPropagation(), oninput: (e) => { vals[f.key] = e.target.value; changed(); } });
      } else {
        ctl = numInput({ value: vals[f.key], step: f.step || 1, min: f.min, max: f.max, unit: f.unit, onCommit: (v) => { vals[f.key] = v; changed(); } });
        ctl.querySelector('input').addEventListener('input', (e) => { const v = evalNum(e.target.value); if (isFinite(v)) vals[f.key] = v; });
      }
      el.append(h('div', { class: 'field' }, h('label', {}, f.label, helpDot(f.help)), ctl));
    }
  };
  draw();
  return { el, vals, redraw: draw };
}

export function debounce(fn, ms) { let t; return (...a) => { clearTimeout(t); t = setTimeout(() => fn(...a), ms); }; }
