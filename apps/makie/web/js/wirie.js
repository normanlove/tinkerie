// Wirie boards: import .makie.json boards from Wirie (nrmluvdev's PCB
// designer), show them as a locked reference model, and make a case for them.
//
// The board node is an ordinary scene node of type "wirieboard" with
//   params.board  – the board data (JSON text, no pictures)
//   reference     – true: drawn and pickable, but left out of exports
//   locked        – true by default, so it can't be nudged by accident
//   texRef        – key of its pictures in S.textures (saved with the project)
//   wirie         – { file, project, notes, caseParams, caseOpts }
// Case nodes carry wirieCase = <board id> and wirieRole = base | lid | extra.
import { S, toast, hexToRgb } from './state.js';
import * as sc from './scene.js';
import * as ui from './ui.js';
import { api, engineTree, whenBuilt, getGeom } from './api.js';
import { mul, invert, decompose, translate, rotX, trs, round } from './math.js';
import { geoBox } from './renderer.js';

const { h, $ } = ui;
let C; // { commit, refresh, select, fitView, requestRender, exportTree, setWorkspace }

export const KIND_COLORS = {
  module: '#2563eb', display: '#111827', sensor: '#9333ea', control: '#f59e0b', connector: '#cbd5e1',
  button: '#475569', switch: '#f97316', led: '#22c55e', mechanical: '#78716c', component: '#b45309',
};
const PCB = '#1f7a3a';

// ---------------- board data ----------------
const infoCache = new Map(); // board JSON text → parsed info
export function boardInfo(n) {
  const s = n?.params?.board;
  if (!s) return null;
  let inf = infoCache.get(s);
  if (!inf) {
    try {
      const data = JSON.parse(s);
      const solid = [];
      (data.parts || []).forEach((p, i) => {
        const b = p.box;
        if (b && b.z1 - b.z0 > 0.01 && b.x1 - b.x0 > 0.01 && b.y1 - b.y0 > 0.01) solid.push(i);
      });
      inf = { data, solid };
    } catch { inf = { data: null, solid: [] }; }
    if (infoCache.size > 40) infoCache.clear();
    infoCache.set(s, inf);
  }
  return inf;
}

// which part a picked triangle belongs to (null = the board itself)
export function partAtTri(n, tri, totalTris) {
  const inf = boardInfo(n);
  if (!inf?.data) return null;
  const slab = totalTris - inf.solid.length * 12;
  if (tri < slab) return null;
  const k = Math.floor((tri - slab) / 12);
  const i = inf.solid[k];
  return i == null ? null : inf.data.parts[i].ref;
}

export const isBoard = (n) => n?.type === 'wirieboard';
export const caseNodesOf = (boardId) => S.nodes.filter((n) => n.wirieCase === boardId);
const boards = () => S.nodes.filter(isBoard);

// ---------------- pictures ----------------
function hashStr(s) {
  let h1 = 0x811c9dc5;
  for (let i = 0; i < s.length; i += Math.max(1, Math.floor(s.length / 4096))) { h1 ^= s.charCodeAt(i); h1 = Math.imul(h1, 16777619); }
  return 'tx' + (h1 >>> 0).toString(36) + s.length.toString(36);
}
function storeTextures(tex) {
  if (!tex || (!tex.top && !tex.bottom)) return null;
  const ref = hashStr((tex.top || '') + '|' + (tex.bottom || ''));
  S.textures ||= {};
  S.textures[ref] = { top: tex.top || '', bottom: tex.bottom || '', ppm: tex.pixelsPerMm || 8 };
  return ref;
}

const gpuTex = new Map(); // ref → { top, bottom, loading }
function texturesFor(R, ref, redraw) {
  if (!ref || !S.textures?.[ref]) return null;
  let t = gpuTex.get(ref);
  if (t) return t;
  t = { top: null, bottom: null };
  gpuTex.set(ref, t);
  for (const side of ['top', 'bottom']) {
    const src = S.textures[ref][side];
    if (!src) continue;
    const img = new Image();
    img.onload = () => { t[side] = R.createTexture(img); redraw(); };
    img.src = src;
  }
  return t;
}

const quadCache = new Map();
function quads(R, W, H, T) {
  const k = `${W},${H},${T}`;
  let q = quadCache.get(k);
  if (!q) {
    const top = geoBox([0, 0, T], [W, H, T]), bot = geoBox([0, 0, 0], [W, H, 0]);
    // keep just the flat faces (first two faces of geoBox are bottom and top)
    const pick = (g, face) => ({ positions: g.positions.slice(face * 18, face * 18 + 18), normals: g.normals.slice(face * 18, face * 18 + 18) });
    const a = pick(top, 1), b = pick(bot, 0);
    q = { top: R.createMesh(a.positions, a.normals), bottom: R.createMesh(b.positions, b.normals) };
    quadCache.set(k, q);
  }
  return q;
}

// ---------------- drawing (called by the viewport) ----------------
export function drawBoard(R, n, e, M, o) {
  const inf = boardInfo(n);
  const total = e.tris;
  const solid = inf?.solid || [];
  const slab = Math.max(0, total - solid.length * 12);
  const hi = o.sel ? 0.55 : o.hover ? 0.3 : 0;
  const op = o.wire ? 0.25 : 1;
  R.drawMesh(e.gpu, M, { color: hexToRgb(PCB), highlight: hi, offset: true, first: 0, count: slab * 3, opacity: op, depthWrite: !o.wire });
  const d = inf?.data;
  if (d && n.showTex !== false) {
    const tx = texturesFor(R, n.texRef, o.redraw);
    if (tx) {
      const q = quads(R, d.board.width, d.board.height, d.board.thickness);
      const rect = [0, 0, d.board.width, d.board.height];
      if (tx.top) R.drawTextured(q.top, tx.top, M, rect, { opacity: op });
      if (tx.bottom) R.drawTextured(q.bottom, tx.bottom, M, rect, { mirror: true, opacity: op });
    }
  }
  if (d && n.showParts !== false) {
    const clash = S.wirieClash || new Set();
    solid.forEach((pi, k) => {
      const p = d.parts[pi];
      let col = hexToRgb(KIND_COLORS[p.kind] || KIND_COLORS.component);
      if (clash.has(p.ref)) col = [0.95, 0.15, 0.2];
      const selPart = S.wiriePart && S.wiriePart.id === n.id && S.wiriePart.ref === p.ref;
      if (selPart) col = col.map((c, i) => c * 0.35 + [0.45, 0.75, 1][i] * 0.65);
      R.drawMesh(e.gpu, M, { color: col, highlight: selPart ? 0.95 : hi * 0.6, offset: true, first: (slab + k * 12) * 3, count: 36, opacity: o.wire ? 0.25 : 0.94, depthWrite: !o.wire });
    });
  }
  if (o.edges || o.sel) R.drawLines(e.egpu, M, o.sel ? [0.3, 0.65, 1, 0.9] : o.dark ? [0, 0, 0, 0.45] : [0, 0, 0, 0.3]);
}

// ---------------- import ----------------
async function friendlyFail(title, err) {
  const msg = String(err?.message || err).replace(/^Error:\s*/, '');
  ui.modal(title, h('div', {}, h('p', {}, msg), h('p', { class: 'note', html: 'Makie reads the <b>.makie.json</b> case files Wirie exports (Export → “Case data for Makie”, format <code>wirie-case</code>, version 1).' })), [{ label: 'OK', primary: true }]);
}

export async function importFile(file) {
  try {
    const r = await fetch('/api/wirie/parse', { method: 'POST', body: await file.arrayBuffer() });
    if (!r.ok) throw new Error(await r.text());
    await addBoard(await r.json(), file.name);
  } catch (e) { friendlyFail('Could not open “' + file.name + '”', e); }
}

export async function openPath(path) {
  try {
    const r = await fetch('/api/wirie/open?path=' + encodeURIComponent(path));
    if (!r.ok) throw new Error(await r.text());
    await addBoard(await r.json(), path.split(/[\\/]/).pop());
    markSeen(path);
  } catch (e) { friendlyFail('Could not open that Wirie board', e); }
}

export function chooseFile() {
  const inp = h('input', { type: 'file', accept: '.json,.makie.json', hidden: true, multiple: true });
  inp.onchange = async () => { for (const f of inp.files) await importFile(f); inp.remove(); };
  document.body.append(inp);
  inp.click();
}

function defaultsFrom(params) {
  const o = {};
  for (const p of params || []) o[p.key] = p.def;
  return o;
}

export async function addParsed(res, fileName) { return addBoard(res, fileName); }

async function addBoard(res, fileName) {
  const same = boards().find((b) => b.wirie?.project === res.project);
  if (same) {
    const choice = await new Promise((resolve) => {
      ui.modal('Update “' + res.project + '”?', h('div', {}, h('p', {}, 'This project already has a Wirie board called “' + res.project + '”.'),
        h('p', { class: 'note' }, 'Replacing it keeps its position, and rebuilds its case with the same settings.')),
      [{ label: 'Add another copy', run: () => resolve('add') }, { label: 'Replace the board', primary: true, run: () => resolve('replace') }], { onClose: () => resolve(null) });
    });
    if (!choice) return;
    if (choice === 'replace') return replaceBoard(same, res);
  }
  const n = sc.makeNode('wirieboard', { board: res.board }, {
    name: res.project + ' (Wirie board)', color: PCB, locked: true, reference: true,
  });
  n.texRef = storeTextures(res.textures);
  n.wirie = { project: res.project, file: res.file || '', fileName, notes: res.notes || [], caseParams: res.caseParams, generator: res.generator };
  // sit it where the case floor would be on the bed, centred
  const def = defaultsFrom(res.caseParams);
  n.pos = [-res.size[0] / 2, -res.size[1] / 2, round((def.standoff || 3) + (def.floor || 2), 3)];
  S.nodes.push(n);
  S.sel = new Set([n.id]);
  S.wiriePart = null;
  C.commit('Import Wirie board');
  C.dismissWelcome?.();
  whenBuilt(n).then(() => C.fitView([n.id]));
  const g = res.generator?.app ? ` from ${res.generator.app} ${res.generator.version || ''}`.trimEnd() : '';
  toast(`Imported “${res.project}”${g}: ${res.parts} parts, ${res.size[0]} × ${res.size[1]} mm. It's a locked reference — press “Make a case for this board” to build one.`, 'ok', 7000);
}

function replaceBoard(n, res) {
  n.params = { ...n.params, board: res.board };
  n.texRef = storeTextures(res.textures) || n.texRef;
  n.wirie = { ...n.wirie, notes: res.notes || [], caseParams: res.caseParams, file: res.file || n.wirie?.file, generator: res.generator };
  S.sel = new Set([n.id]);
  C.commit('Update Wirie board');
  if (caseNodesOf(n.id).length) regenerate(n, n.wirie.caseOpts || defaultsFrom(res.caseParams)).then(() => toast('Board updated and its case rebuilt.', 'ok'));
  else toast('Board updated from Wirie.', 'ok');
}

// ---------------- recent boards & watching the export folder ----------------
const SEEN = 'makie.wirie.seen';
let recent = [], dirs = [], banner = null;
function seen() { try { return JSON.parse(localStorage.getItem(SEEN) || 'null'); } catch { return null; } }
function markSeen(path) {
  const s = seen() || {};
  const f = recent.find((x) => x.path === path);
  s[path] = f ? f.modified : Date.now();
  try { localStorage.setItem(SEEN, JSON.stringify(s)); } catch {}
  if (banner && banner.dataset.path === path) { banner.remove(); banner = null; }
}

async function poll() {
  try {
    const r = await (await fetch('/api/wirie/recent')).json();
    recent = r.files || []; dirs = r.dirs || [];
  } catch { return; }
  let s = seen();
  if (!s) { // first run: everything already there counts as seen
    s = {}; for (const f of recent) s[f.path] = f.modified;
    try { localStorage.setItem(SEEN, JSON.stringify(s)); } catch {}
    return;
  }
  const fresh = recent.find((f) => f.ok && !(s[f.path] >= f.modified));
  if (fresh && (!banner || banner.dataset.path !== fresh.path)) showBanner(fresh);
}

function showBanner(f) {
  banner?.remove();
  const known = boards().some((b) => b.wirie?.project === f.project);
  banner = h('div', { class: 'wirie-banner', 'data-path': f.path },
    h('span', { class: 'wb-ic' }, '⚡'),
    h('span', {}, h('b', {}, known ? 'Updated Wirie board: ' : 'New Wirie board: '), f.project || f.name),
    h('button', { class: 'primary small', onclick: () => openPath(f.path) }, known ? 'Update' : 'Open'),
    h('button', { class: 'ic', title: 'Dismiss', onclick: () => markSeen(f.path) }, '✕'));
  $('#stage').append(banner);
}

export function startWatch() {
  poll();
  setInterval(poll, 15000);
  window.addEventListener('focus', poll);
}

export function recentMenuItems() {
  const items = recent.filter((f) => f.ok).slice(0, 6).map((f) => ({ label: '⚡ ' + (f.project || f.name) + '  ·  ' + ago(f.modified), run: () => openPath(f.path) }));
  return items;
}

function ago(ms) {
  const s = (Date.now() - ms) / 1000;
  if (s < 90) return 'just now';
  if (s < 3600) return Math.round(s / 60) + ' min ago';
  if (s < 86400) return Math.round(s / 3600) + ' h ago';
  return new Date(ms).toLocaleDateString();
}

export async function recentDialog() {
  await poll();
  const list = h('div', { class: 'wirie-list' });
  if (!recent.length) list.append(h('div', { class: 'note' }, 'No Wirie boards found yet. In Wirie, use Export → “Case data for Makie” and the file appears here by itself.'));
  for (const f of recent) {
    list.append(h('div', { class: 'wl-row' + (f.ok ? '' : ' bad') },
      h('div', {}, h('b', {}, f.project || f.name), h('div', { class: 'faint' }, f.name + '.makie.json · ' + ago(f.modified) + (f.ok ? '' : ' · ' + f.problem))),
      h('button', { class: 'btn small', disabled: !f.ok, onclick: () => { m.close(); openPath(f.path); } }, 'Open')));
  }
  const body = h('div', { class: 'wirie-dlg' }, list,
    h('div', { class: 'note' }, 'Makie watches these folders for new exports:', h('ul', { class: 'found' }, ...dirs.map((d) => h('li', {}, d)))),
    h('div', { class: 'note' }, 'You can also drag a .makie.json file onto the Makie window.'));
  const m = ui.modal('Wirie boards', body, [{ label: 'Browse…', run: () => { chooseFile(); } }, { label: 'Close', primary: true }], { wide: true });
}

// ---------------- inspector ----------------
const fmt = (v) => (v == null || !isFinite(v) ? '–' : String(round(v, 2)));

export function inspector(el, n) {
  const inf = boardInfo(n);
  const d = inf?.data;
  el.append(h('div', { class: 'insp-title' }, h('input', { value: n.name || '', spellcheck: 'false', onchange: (e) => { n.name = e.target.value; C.commit('Rename'); }, onkeydown: (e) => e.stopPropagation() })));
  if (!d) { el.append(h('div', { class: 'callout' }, 'The board data is missing.')); return; }
  const g = n.wirie?.generator;
  el.append(h('div', { class: 'insp-sub' }, `Wirie board · ${fmt(d.board.width)} × ${fmt(d.board.height)} × ${fmt(d.board.thickness)} mm · ${d.parts.length} parts` + (g?.app ? ` · ${g.app} ${g.version || ''}` : '')));
  el.append(h('button', { class: 'btn primary', style: 'width:100%;justify-content:center;margin:8px 0 4px', onclick: () => openCasePanel(n.id), html: caseNodesOf(n.id).length ? '✨ Edit the case…' : '✨ Make a case for this board' }));
  el.append(h('div', { class: 'callout tip', html: `<b>Reference object.</b> It shows where everything on the board is, so you can design around it. ${n.refExport ? 'It <b>will</b> be exported.' : 'It is <b>not</b> exported or printed.'}` }));

  const tg = (label, on, fn, help) => h('div', { class: 'field' }, h('label', { title: help || '' }, label), h('button', { class: 'toggle' + (on ? ' on' : ''), onclick: fn }));
  el.append(tg('Locked in place', !!n.locked, () => { n.locked = !n.locked; C.commit('Lock'); }));
  el.append(tg('Include in exports', !!n.refExport, () => { n.refExport = !n.refExport; C.commit('Board export'); }, 'Export the board model too, e.g. to check the fit in your slicer.'));
  if (n.texRef) el.append(tg('Show board pictures', n.showTex !== false, () => { n.showTex = n.showTex === false; C.requestRender(); C.refreshInspector(); }));
  el.append(tg('Show parts', n.showParts !== false, () => { n.showParts = n.showParts === false; C.requestRender(); C.refreshInspector(); }));
  if (!n.locked) ui.transformFields(el, n, S.byType[n.type]);

  for (const t of n.wirie?.notes || []) el.append(h('div', { class: 'note warn-note' }, '⚠ ' + t));

  // selected part
  const sel = S.wiriePart && S.wiriePart.id === n.id ? d.parts.find((p) => p.ref === S.wiriePart.ref) : null;
  if (sel) el.append(partEditor(n, d, sel));

  el.append(h('div', { class: 'group-title' }, `Parts (${d.parts.length}) — click one to edit`));
  const tbl = h('div', { class: 'wparts' });
  for (const p of d.parts) {
    const flat = !(p.box.z1 - p.box.z0 > 0.01);
    const on = sel && sel.ref === p.ref;
    tbl.append(h('div', { class: 'wp-row' + (on ? ' on' : '') + ((S.wirieClash || new Set()).has(p.ref) ? ' clash' : ''), title: p.name || p.value || '', onclick: () => { S.wiriePart = on ? null : { id: n.id, ref: p.ref }; C.refreshInspector(); C.requestRender(); } },
      h('i', { class: 'chip', style: `color:${KIND_COLORS[p.kind] || KIND_COLORS.component}` }),
      h('b', {}, p.ref), h('span', { class: 'wp-val' }, p.value || p.name || ''), h('span', { class: 'wp-h' }, flat ? '' : fmt(p.height) + ' mm' + (p.approxHeight ? '*' : ''))));
  }
  el.append(tbl);
  if (d.parts.some((p) => p.approxHeight)) el.append(h('div', { class: 'note' }, '* height estimated by Wirie — click the part to set the real height.'));

  el.append(h('div', { class: 'group-title' }, 'Actions'));
  const acts = h('div', { class: 'btnrow' });
  if (n.wirie?.file) acts.append(h('button', { class: 'btn small', onclick: () => openPath(n.wirie.file) }, '⟳ Reload from Wirie'));
  acts.append(h('button', { class: 'btn small', onclick: () => C.fitView([n.id]) }, 'Zoom to board'));
  acts.append(h('button', { class: 'btn small danger', onclick: () => { C.run('delete'); } }, 'Delete board'));
  el.append(acts);
}

// edit one part's height / raise and the size of its holes
function partEditor(n, d, p) {
  const box = h('div', { class: 'wpart' });
  box.append(h('div', { class: 'wpart-head' }, h('i', { class: 'chip', style: `color:${KIND_COLORS[p.kind] || '#999'}` }), h('b', {}, p.ref), ' ', p.value || '',
    h('button', { class: 'ic', title: 'Close', onclick: () => { S.wiriePart = null; C.refreshInspector(); C.requestRender(); } }, '✕')));
  box.append(h('div', { class: 'note' }, [p.name, p.kind, p.side === 'bottom' ? 'underneath' : 'on top', p.footprint].filter(Boolean).join(' · ')));
  const num = (label, v, onSet, unit = 'mm', min = 0) => {
    const inp = h('input', { class: 'inp', value: fmt(v), onkeydown: (e) => { e.stopPropagation(); if (e.key === 'Enter') e.target.blur(); }, onchange: (e) => { const x = ui.evalNum(e.target.value); if (isFinite(x) && x >= min) onSet(x); else e.target.value = fmt(v); } });
    return h('div', { class: 'field' }, h('label', {}, label), h('div', { class: 'unitwrap' }, inp, h('span', {}, unit)));
  };
  const flat = !(p.box.z1 - p.box.z0 > 0.01);
  if (!flat || p.kind !== 'mechanical') {
    box.append(num('Height' + (p.approxHeight ? ' (estimate)' : ''), p.height, (v) => editBoard(n, (b) => setPartZ(b, p.ref, v, null), 'Part height')));
    box.append(num('Raised by', p.raisedBy || 0, (v) => editBoard(n, (b) => setPartZ(b, p.ref, null, v), 'Part raise')));
    box.append(h('div', { class: 'note' }, `Occupies z ${fmt(p.box.z0)} → ${fmt(p.box.z1)} mm above the board's underside.`));
  }
  (d.lidFeatures || []).forEach((f, i) => {
    if (f.ref !== p.ref) return;
    box.append(h('div', { class: 'wsub' }, (f.side === 'bottom' ? 'Floor hole: ' : 'Lid hole: ') + (f.type === 'led' ? 'LED' : f.type.replace(/-/g, ' ')) + (f.approx ? ' (estimated)' : '')));
    if (f.shape === 'rect') {
      box.append(num('Width', f.w, (v) => editBoard(n, (b) => { b.lidFeatures[i].w = v; b.lidFeatures[i].approx = false; }, 'Lid hole size')));
      box.append(num('Depth', f.h, (v) => editBoard(n, (b) => { b.lidFeatures[i].h = v; b.lidFeatures[i].approx = false; }, 'Lid hole size')));
    } else box.append(num('Diameter', f.d, (v) => editBoard(n, (b) => { b.lidFeatures[i].d = v; b.lidFeatures[i].approx = false; }, 'Lid hole size')));
  });
  (d.openings || []).forEach((o, i) => {
    if (o.ref !== p.ref) return;
    box.append(h('div', { class: 'wsub' }, `Wall opening (${o.face} wall): ${o.type}` + (o.distanceFromEdge > 1 ? ` · ${fmt(o.distanceFromEdge)} mm back from the edge` : o.distanceFromEdge < -0.05 ? ` · sticks out ${fmt(-o.distanceFromEdge)} mm` : '')));
    box.append(num('Width', o.width, (v) => editBoard(n, (b) => { b.openings[i].width = v; }, 'Opening size')));
    box.append(num('Height', o.height, (v) => editBoard(n, (b) => { b.openings[i].height = v; }, 'Opening size')));
  });
  return box;
}

// change a part's height and/or raise, keeping its holes in step
function setPartZ(b, ref, height, raised) {
  const p = b.parts.find((x) => x.ref === ref);
  if (!p) return;
  const t = b.board.thickness;
  const oldZ0 = p.box.z0, oldTop = p.side === 'bottom' ? p.box.z0 : p.box.z1;
  if (height != null) p.height = height;
  if (raised != null) p.raisedBy = raised;
  if (p.side === 'bottom') { p.box.z1 = -(p.raisedBy || 0); p.box.z0 = p.box.z1 - p.height; }
  else { p.box.z0 = t + (p.raisedBy || 0); p.box.z1 = p.box.z0 + p.height; }
  p.approxHeight = false;
  const newTop = p.side === 'bottom' ? p.box.z0 : p.box.z1;
  for (const f of b.lidFeatures || []) if (f.ref === ref && Math.abs(f.partTop - oldTop) < 1e-6 + Math.abs(f.partTop - oldTop)) f.partTop = newTop;
  for (const o of b.openings || []) if (o.ref === ref) o.z = round(o.z + (p.box.z0 - oldZ0), 3);
  // envelope: tallest parts on each side
  let top = 0, bot = 0;
  for (const q of b.parts) { top = Math.max(top, q.box.z1 - t); bot = Math.max(bot, -q.box.z0); }
  const e = b.envelope || (b.envelope = {});
  e.maxPartHeightTop = round(top, 3); e.maxPartHeightBottom = round(bot, 3);
  e.suggestedLidClearance = Math.max(e.suggestedLidClearance || 0, round(top + 2, 3));
}

function editBoard(n, fn, label) {
  const b = JSON.parse(n.params.board);
  fn(b);
  n.params = { ...n.params, board: JSON.stringify(b) };
  C.commit(label);
  if (caseNodesOf(n.id).length && n.wirie?.caseOpts) regenerate(n, n.wirie.caseOpts);
}

// ---------------- case generator ----------------
let panel = null; // { boardId, opts, info, clashes }

function boardMatrix(b) { return sc.worldMatrix(b.id); }

// fill in default settings of every shape the engine sent
function adopt(t, boardId, role) {
  const def = S.byType[t.type];
  const n = { id: sc.newId(), name: t.name || def?.name || t.type, type: t.type, params: t.params || undefined, pos: t.pos || [0, 0, 0], rot: t.rot || [0, 0, 0], scale: t.scale || [1, 1, 1], hole: !!t.hole };
  if (def && t.type !== 'group') { const p = {}; for (const pr of def.params) p[pr.key] = structuredClone(pr.def); n.params = Object.assign(p, t.params || {}); }
  if (t.type === 'group') { delete n.params; n.children = (t.children || []).map((c) => adopt(c, null, null)); n.expanded = false; }
  n.color = t.color || def?.color || '#94a3b8';
  if (t.slot) n.slot = t.slot;
  if (boardId) { n.wirieCase = boardId; n.wirieRole = role; if (role === 'lid') n.printRot = [180, 0, 0]; }
  return n;
}

async function regenerate(b, opts, keepView = true) {
  const r = await api('/api/wirie/case', { board: b.params.board, opts });
  const res = await r.json();
  const old = caseNodesOf(b.id);
  const looks = {}; for (const o of old) looks[o.wirieRole + '|' + (o.wirieRole === 'extra' ? o.name : '')] = { color: o.color, slot: o.slot };
  const lidView = panel?.lidView || 'Closed';
  const W = boardMatrix(b);
  const lift = liftOf(res.info);
  const made = res.nodes.map((t) => {
    const role = t.wirie || 'extra';
    const n = adopt(t, b.id, role);
    const lk = looks[role + '|' + (role === 'extra' ? n.name : '')];
    if (lk) { n.color = lk.color; if (lk.slot) n.slot = lk.slot; }
    const onLid = role === 'lid' || n.type === 'wcase_bezel';
    if (lidView === 'Lifted') n.pos = [n.pos[0], n.pos[1], n.pos[2] + previewLift(n, lift)];
    if (lidView === 'Hidden' && onLid) n.hidden = true;
    sc.setWorld(n, mul(W, trs(n.pos, n.rot, n.scale)), trs([0, 0, 0], [0, 0, 0], [1, 1, 1]));
    return n;
  });
  // replace the old case where it was in the list
  const first = S.nodes.findIndex((n) => n.wirieCase === b.id);
  S.nodes = S.nodes.filter((n) => n.wirieCase !== b.id);
  S.nodes.splice(first >= 0 ? first : S.nodes.length, 0, ...made);
  S.sel = new Set([...S.sel].filter((id) => sc.node(id)));
  b.wirie = { ...(b.wirie || {}), caseOpts: { ...opts } };
  if (panel && panel.boardId === b.id) { panel.info = res.info; panel.clashes = null; renderPanel(); }
  C.commit('Case for board');
  if (!keepView) C.fitView([b.id, ...made.map((m) => m.id)]);
  checkClashes(b);
  return made;
}

const liftOf = (info) => Math.max(25, (info?.outer?.[2] || 20) * 0.9);
// how far a case part is raised in the "Lifted" preview
const previewLift = (n, lift) => (n.wirieRole === 'lid' || n.type === 'wcase_bezel' ? lift : n.type === 'wcase_plunger' ? lift / 2 : 0);

let clashSeq = 0;
async function checkClashes(b) {
  const seq = ++clashSeq;
  const nodes = caseNodesOf(b.id).map((n) => ({ ...engineTree(n), hidden: false }));
  // check with the lid closed, whatever the preview shows
  const lidUp = panel?.lidView === 'Lifted' ? liftOf(panel.info) : 0;
  if (lidUp) {
    const W = boardMatrix(b), Wi = invert(W);
    for (const t of nodes) {
      const n = sc.node(t.id);
      const dz = previewLift(n, lidUp);
      if (!dz) continue;
      const d = decompose(mul(W, mul(translate(0, 0, -dz), mul(Wi, sc.worldMatrix(n.id)))));
      t.pos = d.pos; t.rot = d.rot; t.scale = d.scale;
    }
  }
  if (panel) { panel.checking = true; renderPanel(); }
  try {
    const r = await api('/api/wirie/clash', { board: engineTree(b), nodes, q: Math.min(1, S.settings.quality) });
    const cl = await r.json();
    if (seq !== clashSeq) return;
    S.wirieClash = new Set(cl.map((c) => c.ref));
    S.clash = new Set(cl.map((c) => c.id));
    if (panel && panel.boardId === b.id) { panel.clashes = cl; panel.checking = false; renderPanel(); }
  } catch (e) {
    if (panel) { panel.checking = false; panel.clashes = null; renderPanel(); }
    toast('Clash check failed: ' + e.message, 'error');
  }
  C.requestRender(); C.refreshInspector();
}

function fieldsFrom(params) {
  // engine params → dialog fields (showIf "key=value" → function)
  return (params || []).map((p) => {
    const f = { key: p.key, label: p.label, type: p.type === 'number' ? undefined : p.type, def: p.def, min: p.min, max: p.max, step: p.step, unit: p.unit, options: p.options, help: p.help };
    if (p.showIf) f.showIf = (v) => p.showIf.split(';').every((r) => { const m = r.match(/^(\w+)(!?=)(.*)$/); if (!m) return true; const x = String(v[m[1]] ?? ''); return m[2] === '=' ? x === m[3] : x !== m[3]; });
    return f;
  });
}

export async function openCasePanel(boardId) {
  const b = sc.node(boardId);
  if (!b) return;
  if (S.workspace !== 'design') C.setWorkspace('design');
  if (!b.wirie?.caseParams) {
    // older projects: ask the engine for the settings list
    try {
      const r = await fetch('/api/wirie/parse', { method: 'POST', body: b.params.board });
      const res = await r.json(); b.wirie = { ...(b.wirie || {}), caseParams: res.caseParams, project: res.project };
    } catch {}
  }
  const opts = { ...defaultsFrom(b.wirie?.caseParams), ...(b.wirie?.caseOpts || {}) };
  panel = { boardId, opts, info: null, clashes: null, lidView: 'Closed' };
  buildPanel(b);
  await regenerate(b, opts, false);
}

export function closeCasePanel() {
  panel = null;
  $('#case-panel')?.classList.add('hidden');
  S.clash = null; S.wirieClash = null;
  C.requestRender(); C.refreshInspector();
}

export const casePanelOpen = () => !!panel;

const rebuild = ui.debounce(() => { const b = panel && sc.node(panel.boardId); if (b) regenerate(b, panel.opts); }, 400);

// the settings form is built once (so typing isn't interrupted); the status
// part is redrawn after every rebuild and clash check
function buildPanel(b) {
  const p = $('#case-panel');
  p.innerHTML = '';
  p.append(h('div', { class: 'mp-head' }, h('b', {}, 'Case for ' + (b.wirie?.project || 'board')), h('button', { class: 'mini', title: 'Close the case settings', onclick: closeCasePanel }, '✕')));
  panel.statusEl = h('div', { class: 'case-status' });
  p.append(panel.statusEl);
  panel.segEl = h('div', { class: 'seg small' });
  p.append(h('div', { class: 'field' }, h('label', {}, 'Lid in the preview'), panel.segEl));
  const f = ui.form(fieldsFrom(b.wirie?.caseParams), (vals) => { Object.assign(panel.opts, vals); rebuild(); });
  Object.assign(f.vals, panel.opts);
  f.redraw();
  p.append(h('div', { class: 'group-title' }, 'Settings'), f.el);
  const acts = h('div', { class: 'btnrow' });
  acts.append(h('button', { class: 'btn small', onclick: () => checkClashes(b) }, 'Check clashes'));
  acts.append(h('button', { class: 'btn small', onclick: (e) => exportMenu(e, b) }, 'Export for printing ▾'));
  acts.append(h('button', { class: 'btn small danger', onclick: () => { S.nodes = S.nodes.filter((n) => n.wirieCase !== b.id); if (b.wirie) delete b.wirie.caseOpts; C.commit('Remove case'); closeCasePanel(); } }, 'Remove case'));
  acts.append(h('button', { class: 'primary small', onclick: () => finish(b) }, 'Done'));
  p.append(acts);
  p.append(h('div', { class: 'note' }, 'Changing a setting rebuilds the case. Edits you make to the case by hand afterwards are kept until you rebuild it.'));
  p.classList.remove('hidden');
  renderPanel();
}

function finish(b) {
  // put the lid back on before leaving the preview
  const after = () => { const ids = caseNodesOf(b.id).map((n) => n.id); closeCasePanel(); C.select(ids, 'set'); toast('Your case is normal Makie geometry now: edit it like any other part. To change the case settings, select the board and press “Edit the case”.', 'ok', 7000); };
  if (panel.lidView !== 'Closed') { panel.lidView = 'Closed'; regenerate(b, panel.opts).then(after); } else after();
}

function renderPanel() {
  if (!panel?.statusEl) return;
  const b = sc.node(panel.boardId);
  if (!b) { closeCasePanel(); return; }
  const el = panel.statusEl;
  el.innerHTML = '';
  const info = panel.info;
  if (info) el.append(h('div', { class: 'case-info' }, ...info.lines.map((l) => h('div', {}, l))));
  else el.append(h('div', { class: 'note' }, 'Building the case…'));
  const cl = panel.clashes;
  if (panel.checking) el.append(h('div', { class: 'clash-st busy' }, 'Checking the case against every part…'));
  else if (cl && !cl.length) el.append(h('div', { class: 'clash-st ok' }, '✓ No clashes: the case clears every part on the board.'));
  else if (cl) el.append(h('div', { class: 'clash-st bad' }, h('b', {}, `⚠ ${cl.length} clash${cl.length > 1 ? 'es' : ''} (shown in red):`), h('ul', {}, ...cl.map((c) => h('li', {}, `${c.ref}${c.part ? ' (' + c.part + ')' : ''} touches “${c.with}”`)))));
  for (const w of info?.warnings || []) el.append(h('div', { class: 'note warn-note' }, '⚠ ' + w));
  panel.segEl.innerHTML = '';
  panel.segEl.append(...['Closed', 'Lifted', 'Hidden'].map((v) => h('button', { class: panel.lidView === v ? 'on' : '', onclick: () => { if (panel.lidView === v) return; panel.lidView = v; const bb = sc.node(panel.boardId); if (bb) regenerate(bb, panel.opts); } }, v)));
}

// ---------------- export, laid out for printing ----------------
function exportMenu(e, b) {
  ui.menu(e, [
    { label: 'Base — STL', run: () => exportCase(b, 'stl', ['base']) },
    { label: 'Lid — STL (flipped for printing)', run: () => exportCase(b, 'stl', ['lid']) },
    { label: 'Extras — STL', run: () => exportCase(b, 'stl', ['extra']), disabled: !caseNodesOf(b.id).some((n) => n.wirieRole === 'extra') },
    '-',
    { label: 'Everything — 3MF, one plate (AMS colours)', run: () => exportCase(b, '3mfasm', ['base', 'lid', 'extra']) },
    { label: 'Everything — 3MF, separate objects', run: () => exportCase(b, '3mf', ['base', 'lid', 'extra']) },
    { label: 'Everything — one STL per part (zip)', run: () => exportCase(b, 'stlzip', ['base', 'lid', 'extra']) },
    { label: 'Everything — one STL per filament (zip)', run: () => exportCase(b, 'colorzip', ['base', 'lid', 'extra']) },
  ]);
}

// print orientation: base as it is, lid upside down, button caps upright, bezel flat
export async function printTree(b, roles) {
  const nodes = caseNodesOf(b.id).filter((n) => roles.includes(n.wirieRole) && !sc.isHole(n));
  const Wi = invert(boardMatrix(b));
  const items = [];
  for (const n of nodes) {
    const t = { ...engineTree(n), hidden: false };
    const e = await whenBuilt({ ...n, hidden: false });
    // in the board frame, with the preview lift removed
    let L = mul(Wi, sc.worldMatrix(n.id));
    if (panel?.lidView === 'Lifted') L = mul(translate(0, 0, -previewLift(n, liftOf(panel.info))), L);
    if (n.wirieRole === 'lid') L = mul(rotX(180), L);
    // bounds of the oriented part
    const P = e.positions;
    let lo = [Infinity, Infinity, Infinity], hi = [-Infinity, -Infinity, -Infinity];
    for (let i = 0; i < P.length; i += 3) {
      const x = P[i], y = P[i + 1], z = P[i + 2];
      const q = [L[0] * x + L[4] * y + L[8] * z + L[12], L[1] * x + L[5] * y + L[9] * z + L[13], L[2] * x + L[6] * y + L[10] * z + L[14]];
      for (let k = 0; k < 3; k++) { if (q[k] < lo[k]) lo[k] = q[k]; if (q[k] > hi[k]) hi[k] = q[k]; }
    }
    items.push({ n, t, L, lo, hi });
  }
  // a row along X, centred on the bed, everything on z = 0
  const gap = 8;
  const total = items.reduce((s, it) => s + (it.hi[0] - it.lo[0]), 0) + gap * Math.max(0, items.length - 1);
  let x = -total / 2;
  const out = [];
  for (const it of items) {
    const cy = (it.lo[1] + it.hi[1]) / 2;
    const M = mul(translate(x - it.lo[0], -cy, -it.lo[2]), it.L);
    const d = decompose(M);
    out.push({ ...it.t, pos: d.pos, rot: d.rot, scale: d.scale });
    x += it.hi[0] - it.lo[0] + gap;
  }
  return out;
}

async function exportCase(b, fmt, roles) {
  const tree = await printTree(b, roles);
  if (!tree.length) { toast('There is nothing to export yet.', 'error'); return; }
  const what = roles.length === 1 ? { base: 'base', lid: 'lid', extra: 'extras' }[roles[0]] : 'case';
  C.exportTree(fmt, tree, `${b.wirie?.project || 'board'} - ${what}`);
}

export function init(ctx) {
  C = ctx;
  startWatch();
  if (!$('#case-panel')) $('#stage').append(h('div', { id: 'case-panel', class: 'side-float hidden' }));
}
