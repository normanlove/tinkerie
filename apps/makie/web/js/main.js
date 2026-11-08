import { S, on, emit, toast, hint } from './state.js';
import * as sc from './scene.js';
import * as vp from './viewport.js';
import * as ui from './ui.js';
import { getGeom, whenBuilt, api, dropCacheForQuality, engineTree } from './api.js';
import { v3, mul, translate, scale as mscale, trs, round, snap, alignZ, invert } from './math.js';
import { openSketch } from './sketch.js';
import * as al from './align.js';
import * as mo from './motion.js';
import { PRINTERS, printerLabel, findPrinter } from './printers.js';
import * as gr from './gears.js';
import * as wy from './wirie.js';

const { h, $ } = ui;
const AUTOSAVE = 'partsmith.autosave.v1';
const SETTINGS = 'partsmith.settings.v1';

// ---------------- history ----------------
let undoStack = [], redoStack = [], lastSnap = '';
const snapState = () => JSON.stringify({ n: S.nodes, j: S.joints || [], f: S.filaments });
function pushHistory() {
  const s = snapState();
  if (s === lastSnap) return false;
  undoStack.push(lastSnap); if (undoStack.length > 150) undoStack.shift();
  redoStack = []; lastSnap = s;
  return true;
}
function restore(s) {
  const st = JSON.parse(s || '{}');
  S.nodes = st.n || []; S.joints = st.j || []; if (st.f) S.filaments = st.f;
  lastSnap = s;
  for (const id of [...S.sel]) if (!sc.node(id)) S.sel.delete(id);
  refresh(); autosave();
}
const undo = () => { if (!undoStack.length) return; redoStack.push(lastSnap); restore(undoStack.pop()); toast('Undo', 'info', 900); };
const redo = () => { if (!redoStack.length) return; undoStack.push(lastSnap); restore(redoStack.pop()); toast('Redo', 'info', 900); };

function commit(label) {
  vp.clearThinSpots();
  if (pushHistory()) { S.dirty = true; autosave(); }
  refresh();
  void label;
}

function refresh() {
  if (S.workspace === 'motion') { mo.renderMotion(); mo.applyMotion(); }
  if (S.workspace === 'layout') mo.computeLayout();
  ui.renderInspector(); ui.renderOutliner(); ui.refreshToolbar(); vp.requestRender(); status();
  $('#welcome').classList.toggle('hidden', S.nodes.length > 0 || welcomeDismissed);
}
let welcomeDismissed = false;

function status() {
  const n = S.sel.size;
  let txt = '';
  if (n === 1) {
    const node = sc.node([...S.sel][0]);
    const sz = node ? ui.sizeOf(node) : [0, 0, 0];
    txt = `${node?.name}: ${sz.map((x) => round(x, 2)).join(' × ')} mm`;
  } else if (n > 1) txt = `${n} parts selected`;
  else txt = `${countParts()} part${countParts() === 1 ? '' : 's'}`;
  $('#selinfo').textContent = txt;
  $('#snapinfo').innerHTML = `<button class="mini" id="snapbtn" title="Change snap grid">Snap ${S.settings.grid || 'off'}${S.settings.grid ? ' mm' : ''}</button>`;
  $('#snapbtn').onclick = (e) => ui.menu(e, [0, 0.1, 0.25, 0.5, 1, 2, 5, 10].map((g) => ({ label: g ? g + ' mm' : 'Off (free)', run: () => { S.settings.grid = g; saveSettings(); status(); } })));
}
const countParts = () => S.nodes.length;

// ---------------- selection ----------------
function select(id, mode = 'set') {
  if (mode === 'clear' || id == null) { if (mode !== 'add') S.sel.clear(); }
  else if (Array.isArray(id)) { if (mode !== 'add') S.sel.clear(); id.forEach((i) => S.sel.add(i)); }
  else if (mode === 'toggle') { S.sel.has(id) ? S.sel.delete(id) : S.sel.add(id); }
  else if (mode === 'keep') { /* child of this object stays selected */ }
  else { S.sel.clear(); S.sel.add(id); }
  // expand groups to show selected children
  for (const sid of S.sel) { let p = sc.parentOf(sid); while (p) { p.expanded = true; p = sc.parentOf(p.id); } }
  ui.renderInspector(); ui.renderOutliner(); ui.refreshToolbar(); vp.requestRender(); status();
}

const selRoots = () => [...S.sel].filter((id) => {
  if (!sc.node(id)) return false;
  let p = sc.parentOf(id);
  while (p) { if (S.sel.has(p.id)) return false; p = sc.parentOf(p.id); }
  return true;
});

// ---------------- commands ----------------
function update(id, patch, label) {
  const n = sc.node(id);
  if (!n) return;
  Object.assign(n, patch);
  commit(label);
}

function setParam(id, key, v) {
  const n = sc.node(id);
  if (!n) return;
  n.params = { ...n.params, [key]: v };
  commit('Change ' + key);
}

function setHole(ids, hole) {
  for (const id of ids) {
    const n = sc.node(id);
    if (!n) continue;
    n.hole = hole;
    if (!hole && n.type === 'group' && sc.isHole(n)) n.children.forEach((c) => { c.hole = false; });
    if (!hole && !n.color) n.color = '#3b82f6';
  }
  commit(hole ? 'Make hole' : 'Make solid');
  if (hole && ids.length) hint('Now select the hole <b>and</b> a solid, then press <b>Group</b> (Ctrl+G) to cut it out.');
}

function setPosition(id, axis, v) {
  const box = sc.worldBox(id);
  if (!box) return;
  const cur = axis === 2 ? box[0][2] : (box[0][axis] + box[1][axis]) / 2;
  const d = [0, 0, 0]; d[axis] = v - cur;
  moveWorld([id], d);
  commit('Move');
}

function moveWorld(ids, d) {
  for (const id of ids) {
    const n = sc.node(id);
    sc.setWorld(n, mul(translate(...d), sc.worldMatrix(id)), sc.parentWorld(id));
  }
}

function setSize(id, axis, v) {
  const n = sc.node(id);
  const sz = ui.sizeOf(n);
  if (!sz[axis]) return;
  const f = v / sz[axis];
  const axes = S.settings.lockRatio ? [0, 1, 2] : [axis];
  for (const a of axes) n.scale[a] = (n.scale[a] || 1) * f;
  // keep the bottom where it was
  const before = sc.worldBox(id);
  vp.bakeScales([id]);
  commit('Resize');
  whenBuilt(n).then(() => {
    const after = sc.worldBox(id);
    if (before && after && Math.abs(after[0][2] - before[0][2]) > 1e-4) { moveWorld([id], [0, 0, before[0][2] - after[0][2]]); commit('Resize'); }
  });
}

function group() {
  const ids = selRoots();
  if (ids.length < 2) { toast('Select two or more parts to group (Shift+click).'); return; }
  const hasHole = ids.some((id) => sc.isHole(sc.node(id))), hasSolid = ids.some((id) => !sc.isHole(sc.node(id)));
  const g = sc.groupNodes(ids);
  if (!g) return;
  S.sel = new Set([g.id]);
  commit('Group');
  if (hasHole && hasSolid) toast('Grouped — holes are cut out of the solids.', 'ok');
}

function ungroupSel() {
  const out = [];
  for (const id of selRoots()) {
    const n = sc.node(id);
    if (n?.type === 'group') out.push(...sc.ungroup(id));
    else out.push(id);
  }
  S.sel = new Set(out);
  commit('Ungroup');
}

function del() {
  const ids = selRoots();
  if (!ids.length) return;
  ids.forEach((id) => sc.removeNode(id));
  // remove groups left empty
  const prune = (arr) => { for (let i = arr.length - 1; i >= 0; i--) { const n = arr[i]; if (n.type === 'group') { prune(n.children); if (!n.children.length) arr.splice(i, 1); } } };
  prune(S.nodes);
  S.sel.clear();
  commit('Delete');
}

function duplicate(ids = selRoots(), inPlace = false) {
  if (!ids.length) return [];
  const box = sc.selectionBox(ids);
  const dx = inPlace || !box ? 0 : round(box[1][0] - box[0][0] + 4, 2);
  const out = [];
  for (const id of ids) {
    const n = sc.node(id);
    const arr = sc.siblings(id);
    const c = sc.cloneNode(n);
    if (n.name && !/copy/i.test(n.name)) c.name = n.name;
    arr.splice(arr.indexOf(n) + 1, 0, c);
    if (dx) moveWorld([c.id], [dx, 0, 0]);
    out.push(c.id);
  }
  S.sel = new Set(out);
  if (!inPlace) { commit('Duplicate'); toast('Copy placed next to the original'); } else refresh();
  return out;
}

function mirror(axis) {
  const ids = selRoots();
  const box = sc.selectionBox(ids);
  if (!box) return;
  const c = v3.mul(v3.add(box[0], box[1]), 0.5);
  const s = [1, 1, 1]; s[axis] = -1;
  const D = mul(translate(...c), mul(mscale(...s), translate(...v3.mul(c, -1))));
  for (const id of ids) sc.setWorld(sc.node(id), mul(D, sc.worldMatrix(id)), sc.parentWorld(id));
  commit('Mirror');
}

function align(ids, axis, k) {
  const box = sc.selectionBox(ids);
  if (!box) return;
  const target = k === 0 ? box[0][axis] : k === 2 ? box[1][axis] : (box[0][axis] + box[1][axis]) / 2;
  for (const id of ids) {
    const b = sc.worldBox(id);
    const cur = k === 0 ? b[0][axis] : k === 2 ? b[1][axis] : (b[0][axis] + b[1][axis]) / 2;
    const d = [0, 0, 0]; d[axis] = target - cur;
    moveWorld([id], d);
  }
  commit('Align');
}

function distribute(ids, axis) {
  if (ids.length < 3) { toast('Select three or more parts to distribute.'); return; }
  const items = ids.map((id) => { const b = sc.worldBox(id); return { id, c: (b[0][axis] + b[1][axis]) / 2 }; }).sort((a, b) => a.c - b.c);
  const a = items[0].c, z = items[items.length - 1].c, step = (z - a) / (items.length - 1);
  items.forEach((it, i) => { const d = [0, 0, 0]; d[axis] = a + step * i - it.c; moveWorld([it.id], d); });
  commit('Distribute');
}

function drop() {
  const ids = selRoots();
  for (const id of ids) {
    const b = sc.worldBox(id);
    if (!b) continue;
    let floor = 0;
    for (const n of S.nodes) {
      if (ids.includes(n.id) || n.hidden || sc.isHole(n) || sc.isDescendant(id, n.id)) continue;
      const o = sc.worldBox(n.id);
      if (!o) continue;
      const overlap = o[0][0] < b[1][0] - 1e-3 && o[1][0] > b[0][0] + 1e-3 && o[0][1] < b[1][1] - 1e-3 && o[1][1] > b[0][1] + 1e-3;
      if (overlap && o[1][2] <= b[0][2] + 1e-3 && o[1][2] > floor) floor = o[1][2];
    }
    moveWorld([id], [0, 0, floor - b[0][2]]);
  }
  commit('Drop');
}

function treeDrop(dragId, targetId, where) {
  if (where === 'root') { sc.reparent(dragId, null); commit('Move out of group'); return; }
  const target = sc.node(targetId);
  if (!target) return;
  if (where === 'in') {
    if (target.type === 'group') { sc.reparent(dragId, targetId); target.expanded = true; commit('Add to group'); toast(`Added to “${target.name}”`, 'ok'); }
    else {
      // dropping onto a part groups the two
      const p = sc.parentOf(targetId);
      sc.reparent(dragId, p ? p.id : null);
      const g = sc.groupNodes([targetId, dragId]);
      S.sel = new Set([g.id]);
      commit('Group');
      toast('Grouped' + (sc.isHole(sc.node(dragId)) || sc.isHole(target) ? ' — holes are cut out.' : '.'), 'ok');
    }
    return;
  }
  const p = sc.parentOf(targetId);
  sc.reparent(dragId, p ? p.id : null);
  const arr = sc.siblings(targetId);
  const dn = sc.node(dragId);
  arr.splice(arr.indexOf(dn), 1);
  const ti = arr.indexOf(target);
  // the list shows the last child on top, so "before" in the list means after in the array
  arr.splice(where === 'before' ? ti + 1 : ti, 0, dn);
  commit('Reorder');
}

// ---------------- placing ----------------
function make(type) {
  const n = sc.makeNode(type);
  return n;
}
function place(type) {
  vp.startPlacing(make(type));
}
function onPlace(n) {
  const node = structuredClone(n);
  node.id = sc.newId();
  S.nodes.push(node);
  S.sel = new Set([node.id]);
  commit('Add ' + node.name);
  welcomeDismissed = true;
}

// ---------------- dialogs ----------------
function patternDialog() {
  const ids = selRoots();
  if (!ids.length) return;
  const box = sc.selectionBox(ids);
  const w = box ? round(box[1][0] - box[0][0] + 5, 1) : 20, d = box ? round(box[1][1] - box[0][1] + 5, 1) : 20;
  const f = ui.form([
    { key: 'kind', label: 'Pattern', type: 'select', options: ['Row', 'Grid', 'Circle'], def: 'Row' },
    { key: 'count', label: 'Copies (total)', def: 4, min: 2, max: 500, step: 1, showIf: (v) => v.kind !== 'Grid' },
    { key: 'nx', label: 'Columns', def: 3, min: 1, max: 100, showIf: (v) => v.kind === 'Grid' },
    { key: 'ny', label: 'Rows', def: 2, min: 1, max: 100, showIf: (v) => v.kind === 'Grid' },
    { key: 'dx', label: 'Spacing X', def: w, step: 1, unit: 'mm', showIf: (v) => v.kind !== 'Circle' },
    { key: 'dy', label: 'Spacing Y', def: 0, step: 1, unit: 'mm', showIf: (v) => v.kind !== 'Circle' },
    { key: 'dz', label: 'Spacing Z', def: 0, step: 1, unit: 'mm', showIf: (v) => v.kind === 'Row' },
    { key: 'radius', label: 'Radius', def: 30, min: 0.1, step: 1, unit: 'mm', showIf: (v) => v.kind === 'Circle' },
    { key: 'angle', label: 'Total angle', def: 360, min: 1, max: 360, step: 5, unit: '°', showIf: (v) => v.kind === 'Circle' },
    { key: 'rotate', label: 'Rotate copies', type: 'bool', def: true, showIf: (v) => v.kind === 'Circle' },
    { key: 'group', label: 'Group the result', type: 'bool', def: false, help: 'Handy for a pattern of holes: group it, then group it with your part.' },
    { type: 'note', html: 'Circle patterns go around the current position of the selection.' },
  ]);
  ui.modal('Pattern copies', f.el, [{ label: 'Cancel' }, { label: 'Create', primary: true, run: () => makePattern(ids, f.vals, d) }]);
}

function makePattern(ids, v, depth) {
  const made = [...ids];
  const box = sc.selectionBox(ids);
  const c = v3.mul(v3.add(box[0], box[1]), 0.5);
  const addCopy = (D) => {
    for (const id of ids) {
      const n = sc.node(id), arr = sc.siblings(id);
      const cp = sc.cloneNode(n);
      arr.push(cp);
      sc.setWorld(cp, mul(D, sc.worldMatrix(id)), sc.parentWorld(id));
      made.push(cp.id);
    }
  };
  if (v.kind === 'Row') for (let i = 1; i < v.count; i++) addCopy(translate(v.dx * i, v.dy * i, v.dz * i));
  else if (v.kind === 'Grid') {
    const dy = v.dy || depth;
    for (let j = 0; j < v.ny; j++) for (let i = 0; i < v.nx; i++) if (i || j) addCopy(translate(v.dx * i, dy * j, 0));
  } else {
    const full = v.angle >= 359.99;
    const n = v.count, step = full ? v.angle / n : v.angle / Math.max(1, n - 1);
    // centre of the circle is offset so the original sits on it
    const ctr = [c[0] - v.radius, c[1], 0];
    for (let i = 1; i < n; i++) {
      const a = step * i;
      const R = mul(translate(ctr[0], ctr[1], 0), mul(rotZdeg(a), translate(-ctr[0], -ctr[1], 0)));
      let D = R;
      if (!v.rotate) {
        const p = rotPoint([c[0], c[1]], ctr, a);
        D = translate(p[0] - c[0], p[1] - c[1], 0);
      }
      addCopy(D);
    }
  }
  if (v.group) { const g = sc.groupNodes(made); g.name = 'Pattern'; S.sel = new Set([g.id]); }
  else S.sel = new Set(made);
  commit('Pattern');
}
function rotZdeg(a) { const r = trs([0, 0, 0], [0, 0, a], [1, 1, 1]); return r; }
function rotPoint(p, c, a) { const r = (a * Math.PI) / 180, x = p[0] - c[0], y = p[1] - c[1]; return [c[0] + x * Math.cos(r) - y * Math.sin(r), c[1] + x * Math.sin(r) + y * Math.cos(r)]; }

function cornerHoles(id) {
  const b = sc.worldBox(id);
  if (!b) return;
  const n = sc.node(id);
  const f = ui.form([
    { key: 'size', label: 'Screw size', type: 'select', options: ['M2', 'M2.5', 'M3', 'M4', 'M5', 'M6', 'M8'], def: 'M3' },
    { key: 'head', label: 'Head recess', type: 'select', options: ['None', 'Counterbore', 'Countersink'], def: 'Countersink' },
    { key: 'count', label: 'Holes', type: 'select', options: ['4 corners', '2 (left & right)', '2 (front & back)', '1 centre'], def: '4 corners' },
    { key: 'inset', label: 'Distance from edge', def: 5, min: 1, step: 0.5, unit: 'mm' },
    { key: 'cut', label: 'Cut them in now', type: 'bool', def: true, help: 'Groups the holes with the part so they are cut out straight away.' },
    { type: 'note', html: 'Holes are drilled straight down from the top face, all the way through.' },
  ]);
  ui.modal('Add screw holes to “' + n.name + '”', f.el, [{ label: 'Cancel' }, {
    label: 'Add holes', primary: true, run: () => {
      const v = f.vals, i = v.inset;
      const [lo, hi] = b, cx = (lo[0] + hi[0]) / 2, cy = (lo[1] + hi[1]) / 2;
      const pts = v.count === '4 corners' ? [[lo[0] + i, lo[1] + i], [hi[0] - i, lo[1] + i], [hi[0] - i, hi[1] - i], [lo[0] + i, hi[1] - i]]
        : v.count === '2 (left & right)' ? [[lo[0] + i, cy], [hi[0] - i, cy]] : v.count === '2 (front & back)' ? [[cx, lo[1] + i], [cx, hi[1] - i]] : [[cx, cy]];
      const holes = pts.map(([x, y]) => {
        const hn = sc.makeNode('screwhole', { size: v.size, head: v.head, depth: round(hi[2] - lo[2] + 0.4, 2) });
        hn.pos = [round(x, 3), round(y, 3), hi[2]];
        return hn;
      });
      const parent = sc.parentOf(id);
      const arr = parent ? parent.children : S.nodes;
      const PW = sc.parentWorld(id);
      for (const hn of holes) { const W = trs(hn.pos, [0, 0, 0], [1, 1, 1]); arr.push(hn); sc.setWorld(hn, W, PW); }
      if (v.cut) { const g = sc.groupNodes([id, ...holes.map((x) => x.id)]); g.name = n.name; S.sel = new Set([g.id]); }
      else S.sel = new Set(holes.map((x) => x.id));
      commit('Screw holes');
      toast(`${holes.length} ${v.size} hole${holes.length > 1 ? 's' : ''} added`, 'ok');
    },
  }]);
}

function matchingHole(id) {
  const n = sc.node(id);
  const b = sc.worldBox(id);
  const W = sc.worldMatrix(id);
  let hn;
  const size = n.params.size || 'M3';
  if (n.type === 'bolt') hn = sc.makeNode('screwhole', { size, depth: (n.params.length || 10) + 0.4, head: n.params.head === 'Countersunk' ? 'Countersink' : n.params.head === 'Socket cap' ? 'Counterbore' : 'None' });
  else if (n.type === 'nut' || n.type === 'sqnut') hn = sc.makeNode('nuttrap', { size, boltDepth: 0 });
  else hn = sc.makeNode('screwhole', { size, depth: round(b[1][2] - b[0][2] + 0.4, 2) });
  const arr = sc.siblings(id);
  arr.push(hn);
  if (n.type === 'bolt') sc.setWorld(hn, W, sc.parentWorld(id));
  else sc.setWorld(hn, mul(translate(0, 0, b[1][2] - W[14]), W), sc.parentWorld(id));
  S.sel = new Set([hn.id]);
  commit('Matching hole');
  toast('Matching hole added in the same place. Move the part away and group the hole with your design.', 'ok', 5000);
}

function splitDialog(id) {
  const n = sc.node(id), b = sc.worldBox(id);
  if (!b) return;
  const f = ui.form([
    { key: 'axis', label: 'Cut across', type: 'select', options: ['Z (horizontal cut)', 'X (left / right)', 'Y (front / back)'], def: 'Z (horizontal cut)' },
    { key: 'at', label: 'At (% of size)', def: 50, min: 1, max: 99, step: 5, unit: '%' },
    { key: 'gap', label: 'Move halves apart', def: 10, min: 0, step: 1, unit: 'mm' },
    { key: 'flat', label: 'Lay halves flat for printing', type: 'bool', def: true },
    { type: 'note', html: 'Splitting lets you print parts bigger than your printer, or avoid supports. Glue or bolt the halves together afterwards.' },
  ]);
  ui.modal('Split “' + n.name + '” in two', f.el, [{ label: 'Cancel' }, {
    label: 'Split', primary: true, run: () => {
      const ax = { Z: 2, X: 0, Y: 1 }[f.vals.axis[0]];
      const at = b[0][ax] + ((b[1][ax] - b[0][ax]) * f.vals.at) / 100;
      const W = sc.worldMatrix(id);
      const lo = b[0].map((x) => x - 2), hi = b[1].map((x) => x + 2);
      const halves = [];
      for (const side of [0, 1]) {
        const blo = [...lo], bhi = [...hi];
        if (side === 0) bhi[ax] = at; else blo[ax] = at;
        const box = sc.makeNode('box', { w: round(bhi[0] - blo[0], 3), d: round(bhi[1] - blo[1], 3), h: round(bhi[2] - blo[2], 3) });
        box.pos = [(blo[0] + bhi[0]) / 2, (blo[1] + bhi[1]) / 2, blo[2]];
        const c = sc.cloneNode(n);
        sc.setWorld(c, W, trs([0, 0, 0], [0, 0, 0], [1, 1, 1]));
        c.hole = false;
        const g = { id: sc.newId(), name: n.name + (side ? ' — part B' : ' — part A'), type: 'group', mode: 'intersect', pos: [0, 0, 0], rot: [0, 0, 0], scale: [1, 1, 1], hole: false, color: n.color, children: [c, box] };
        halves.push(g);
      }
      sc.removeNode(id);
      S.nodes.push(...halves);
      const d = [0, 0, 0]; d[ax] = f.vals.gap;
      moveWorld([halves[1].id], d);
      if (f.vals.flat && ax === 2) {
        // flip the top half so its cut face sits on the bed
        const c = [(b[0][0] + b[1][0]) / 2, (b[0][1] + b[1][1]) / 2, (at + b[1][2]) / 2 + f.vals.gap];
        const D = mul(translate(...c), mul(trs([0, 0, 0], [180, 0, 0], [1, 1, 1]), translate(...v3.mul(c, -1))));
        sc.setWorld(halves[1], mul(D, sc.worldMatrix(halves[1].id)), trs([0, 0, 0], [0, 0, 0], [1, 1, 1]));
        moveWorld([halves[1].id], [b[1][0] - b[0][0] + f.vals.gap, 0, 0]);
      }
      S.sel = new Set(halves.map((g) => g.id));
      commit('Split');
      Promise.all(halves.map((g) => whenBuilt(g))).then(() => { S.sel = new Set(halves.map((g) => g.id)); drop(); });
      toast('Split into two parts. Each half is a "Keep overlap" group, so you can still edit the original inside.', 'ok', 6000);
    },
  }]);
}

function layFlat(id) {
  vp.startFacePick('<b>Lay flat:</b> click the face that should sit on the print bed.', (hit) => {
    const root = selRoots().includes(hit.id) ? hit.id : id;
    const n = hit.normal;
    const D0 = mul(alignZ([0, 0, -1]), invert(alignZ(n)));
    const b = sc.worldBox(root);
    const c = v3.mul(v3.add(b[0], b[1]), 0.5);
    const D = mul(translate(...c), mul(D0, translate(...v3.mul(c, -1))));
    sc.setWorld(sc.node(root), mul(D, sc.worldMatrix(root)), sc.parentWorld(root));
    S.sel = new Set([root]);
    commit('Lay flat');
    whenBuilt(sc.node(root)).then(() => { S.sel = new Set([root]); drop(); });
  });
}

// create the mating half of a thread in the same place (screw <-> threaded hole)
function matchingThread(id) {
  const n = sc.node(id);
  let m;
  const tp = {};
  for (const k of Object.keys(n.params || {})) if (k.startsWith('t')) tp[k] = n.params[k];
  if (n.type === 'threadhole') {
    const sizes = { M2: 2, 'M2.5': 2.5, M3: 3, M4: 4, M5: 5, M6: 6, M8: 8, M10: 10, M12: 12 };
    const d = n.params.size === 'Custom' ? n.params.d : sizes[n.params.size] || 6;
    m = sc.makeNode('tscrew', { ...tp, d, h: n.params.depth + 5 });
    m.name = `Screw M${d}`;
    sc.siblings(id).push(m);
    sc.setWorld(m, mul(sc.worldMatrix(id), translate(0, 0, -n.params.depth)), sc.parentWorld(id));
  } else {
    const hole = !sc.isHole(n);
    m = sc.makeNode(hole ? 'tbore' : 'tscrew', { ...n.params });
    m.hole = hole;
    m.name = hole ? 'Threaded hole Ø' + n.params.d : 'Threaded post Ø' + n.params.d;
    sc.siblings(id).push(m);
    sc.setWorld(m, sc.worldMatrix(id), sc.parentWorld(id));
  }
  S.sel = new Set([m.id]);
  commit('Matching thread');
  toast('Matching thread made in the same spot. Move it to your other part and group it there — the fit clearance is built in.', 'ok', 6000);
}

// ---------------- scan a model and make parts for it ----------------
async function scanModel(id) {
  const n = sc.node(id);
  const top = sc.topOf(id) || n;
  const wait = ui.modal('Scanning “' + (n.name || 'model') + '”…', h('div', { class: 'note' }, 'Looking for the board outline, mounting holes, screen, connectors and heights…'), []);
  let res;
  try {
    const r = await api('/api/scan', { node: engineTree(top), q: 1 });
    res = await r.json();
  } catch (e) { wait.close(); toast('Scan failed: ' + e.message, 'error'); return; }
  wait.close();
  if (!res.ok) { ui.modal('Could not scan this model', h('div', { class: 'note' }, (res.report || []).join(' ')), [{ label: 'OK', primary: true }]); return; }
  const sp = res.spec;
  const hasHoles = sp.holes && sp.holes.length > 0;
  const hasWin = !!sp.window || (sp.circles && sp.circles.length > 0);
  const parts = [
    ['Mount plate', 'boardmount', hasHoles, 'Base plate with standoffs at the mounting holes'],
    ['Snap-in tray', 'boardmount', !hasHoles, 'Tray with ledges and snap lips (no screws)'],
    ['Standoffs only', 'boardmount', false, 'Just the standoffs, to add to your own design'],
    ['Case', 'boardcase', true, 'Enclosure with standoffs and connector openings'],
    ['Case lid', 'boardlid', true, hasWin ? 'Lid with a window for the screen / sensors' : 'Push-fit lid'],
    ['Front bezel', 'bezel', hasWin, 'Front frame that holds it from behind'],
    ['Mounting holes', 'pcbmount', false, 'Hole pattern to drill into another part'],
    ['Holder / cradle', 'boardmount', !hasHoles, 'Block with a pocket shaped exactly like it'],
  ];
  const chosen = new Set(parts.filter((q) => q[2]).map((q) => q[0]));
  const f = ui.form([
    { key: 'screw', label: 'Screw size', type: 'select', options: ['Auto (from board)', 'M2', 'M2.5', 'M3', 'M4'], def: 'Auto (from board)' },
    { key: 'fix', label: 'Fixing', type: 'select', options: ['Self-tapping screw', 'Heat-set insert', 'Through bolt + nut', 'Snap pins (no screws)'], def: hasHoles ? 'Self-tapping screw' : 'Snap pins (no screws)' },
    { key: 'tol', label: 'Fit clearance', def: 0.4, min: 0, max: 3, step: 0.1, unit: 'mm' },
    { key: 'wall', label: 'Wall thickness', def: 2, min: 0.8, max: 10, step: 0.2, unit: 'mm' },
    { key: 'standoff', label: 'Standoff height (0 = auto)', def: 0, min: 0, max: 100, step: 0.5, unit: 'mm' },
    { key: 'extra', label: 'Extra case height', def: 2, min: 0, max: 100, step: 0.5, unit: 'mm' },
  ]);
  const list = h('div', { class: 'scanlist' }, ...parts.map(([name, , , desc]) => {
    const cb = h('input', { type: 'checkbox' }); cb.checked = chosen.has(name);
    cb.onchange = () => { cb.checked ? chosen.add(name) : chosen.delete(name); };
    return h('label', { class: 'scanitem' }, cb, h('b', {}, name), h('span', { class: 'faint' }, desc));
  }));
  const body = h('div', {},
    h('div', { class: 'callout tip' }, h('b', {}, 'Found: '), h('ul', { class: 'found' }, ...res.report.map((t) => h('li', {}, t)))),
    h('div', { class: 'note' }, 'Not right? Lay the model flat with the board horizontal and its parts facing up, then scan again.'),
    h('div', { class: 'group-title' }, 'Make these parts'), list,
    h('div', { class: 'group-title' }, 'Settings'), f.el);
  ui.modal('Make parts for “' + (n.name || 'model') + '”', body, [{ label: 'Cancel' }, {
    label: 'Create parts', primary: true, run: () => {
      const v = f.vals;
      const [ox, oy, oz] = res.origin;
      const floor = 2, hs = v.standoff > 0 ? v.standoff : Math.max(3, sp.below + 1);
      const caseTotal = floor + hs + sp.t + sp.above + v.extra;
      const spec = JSON.stringify(sp);
      const made = [];
      for (const [name, type] of parts) {
        if (!chosen.has(name)) continue;
        const node = sc.makeNode(type, { spec, mode: name, screw: v.screw, fix: v.fix, tol: v.tol, wall: v.wall, standoff: v.standoff, extra: v.extra });
        node.name = `${name} for ${n.name || 'model'}`;
        const baseZ = oz - floor - hs;
        switch (name) {
          case 'Standoffs only': node.pos = [ox, oy, oz - hs]; break;
          case 'Case lid': node.pos = [ox, oy, baseZ + caseTotal + 2]; node.rot = [0, 180, 0]; node.printRot = [0, 180, 0]; break;
          case 'Front bezel': node.pos = [ox, oy, oz + sp.t + 2 + (sp.window ? sp.window.h : 1.6) + (sp.window ? 0 : 0)]; node.rot = [0, 180, 0]; node.printRot = [0, 180, 0]; break;
          case 'Mounting holes': node.pos = [ox + sp.w + 25, oy, 6]; break;
          case 'Holder / cradle': node.pos = [ox, oy + sp.d + 30, 0]; break;
          default: node.pos = [ox, oy, baseZ];
        }
        S.nodes.push(node); made.push(node.id);
      }
      if (!made.length) return;
      // keep the whole assembly on the bed
      whenBuilt(sc.node(made[0])).then(async () => {
        await Promise.all(made.map((m) => whenBuilt(sc.node(m))));
        const box = sc.selectionBox([...made, top.id]);
        if (box && box[0][2] < -0.01) { const dz = -box[0][2]; for (const m of [...made, top.id]) sc.node(m).pos[2] += dz; commit('Lift onto bed'); }
        vp.fitView([...made, top.id]);
      });
      S.sel = new Set(made);
      commit('Make parts for model');
      toast(`Made ${made.length} part${made.length > 1 ? 's' : ''} around your model so you can check the fit. Switch to Print layout to lay them out for printing.`, 'ok', 7000);
    },
  }], { wide: true });
}

function editSketch(id) {
  const n = sc.node(id);
  openSketch(n.params, (params) => { n.params = { ...n.params, ...params }; commit('Edit sketch'); });
}

function settingsDialog() {
  const st = S.settings;
  const f = ui.form([
    { key: 'bx', label: 'Bed width (X)', def: st.bed[0], min: 20, step: 5, unit: 'mm' },
    { key: 'by', label: 'Bed depth (Y)', def: st.bed[1], min: 20, step: 5, unit: 'mm' },
    { key: 'bz', label: 'Max height (Z)', def: st.bed[2], min: 20, step: 5, unit: 'mm' },
    { key: 'grid', label: 'Snap grid', type: 'select', options: ['Off', '0.1', '0.25', '0.5', '1', '2', '5', '10'], def: String(st.grid || 'Off') },
    { key: 'rot', label: 'Rotation snap', type: 'select', options: ['1', '5', '15', '22.5', '45', '90'], def: String(st.rotSnap) },
    { key: 'quality', label: 'Curve quality', type: 'select', options: ['Draft (fast)', 'Normal', 'Fine', 'Ultra'], def: ['Draft (fast)', 'Normal', 'Fine', 'Ultra'][[0.6, 1, 1.6, 2.4].indexOf(st.quality)] || 'Normal', help: 'Smoothness of round shapes. Fine is best for export, Draft is fastest to edit.' },
    { key: 'theme', label: 'Theme', type: 'select', options: ['Dark', 'Light'], def: st.theme === 'light' ? 'Light' : 'Dark' },
    { key: 'edges', label: 'Show part edges', type: 'bool', def: st.edges !== false },
    { key: 'showBed', label: 'Show build volume', type: 'bool', def: st.showBed },
    { key: 'autoDepth', label: 'Smart drilling', type: 'bool', def: st.autoDepth, help: 'Screw holes placed on a face automatically go all the way through.' },
    { key: 'lockRatio', label: 'Keep proportions when resizing', type: 'bool', def: !!st.lockRatio },
  ]);
  const presets = h('div', { class: 'btnrow', style: 'margin-bottom:10px' }, h('span', { class: 'note' }, 'Printer: ' + (st.printer || 'Custom')),
    h('button', { class: 'mini', onclick: () => { m.close(); printerDialog(settingsDialog); } }, 'Choose printer…'));
  const wrap = h('div', {}, presets, f.el);
  const m = ui.modal('Settings', wrap, [{ label: 'Cancel' }, {
    label: 'Save', primary: true, run: () => {
      const v = f.vals;
      const q = [0.6, 1, 1.6, 2.4][['Draft (fast)', 'Normal', 'Fine', 'Ultra'].indexOf(v.quality)] || 1;
      if (q !== st.quality) { st.quality = q; dropCacheForQuality(); }
      if (v.bx !== st.bed[0] || v.by !== st.bed[1] || v.bz !== st.bed[2]) st.printer = findPrinterBySize([v.bx, v.by, v.bz]);
      st.bed = [v.bx, v.by, v.bz];
      updatePrinterChip();
      st.grid = v.grid === 'Off' ? 0 : parseFloat(v.grid);
      st.rotSnap = parseFloat(v.rot);
      st.theme = v.theme === 'Light' ? 'light' : 'dark';
      Object.assign(st, { edges: v.edges, showBed: v.showBed, autoDepth: v.autoDepth, lockRatio: v.lockRatio });
      applyTheme(); saveSettings(); refresh();
    },
  }]);
}

function applyTheme() { document.body.classList.toggle('dark', S.settings.theme !== 'light'); document.body.classList.toggle('light', S.settings.theme === 'light'); vp.requestRender(); }
// ---------- printer / build plate ----------
function findPrinterBySize(b) {
  for (const [brand, list] of PRINTERS) for (const p of list) if (p[1] === b[0] && p[2] === b[1] && p[3] === b[2]) return printerLabel(brand, p[0]);
  return '';
}

function setPrinter(label, bed) {
  S.settings.printer = label;
  S.settings.bed = bed.map((x) => Math.max(20, Math.min(5000, +x || 0)));
  saveSettings(); updatePrinterChip(); vp.requestRender(); refresh();
  toast(`Build plate: ${label || 'Custom'} · ${S.settings.bed.join(' × ')} mm`, 'ok');
}

function updatePrinterChip() {
  const c = document.getElementById('printer-chip');
  if (!c) return;
  const b = S.settings.bed;
  c.innerHTML = '';
  c.append(h('span', {}, '🖨 ' + (S.settings.printer || findPrinterBySize(b) || 'Custom printer')), h('span', { class: 'dim' }, `${b[0]} × ${b[1]} × ${b[2]} mm`));
}

function printerDialog(after) {
  const cur = S.settings.printer || '';
  const list = h('div', { class: 'printer-list' });
  const search = h('input', { class: 'search', placeholder: 'Search printers — e.g. "bambu", "ender", "prusa"…' });
  let m;
  const pick = (label, bed) => { setPrinter(label, bed); m.close(); after?.(); };
  const draw = () => {
    const q = search.value.trim().toLowerCase();
    list.innerHTML = '';
    for (const [brand, ps] of PRINTERS) {
      const rows = ps.filter((p) => !q || (brand + ' ' + p[0]).toLowerCase().includes(q));
      if (!rows.length) continue;
      list.append(h('div', { class: 'brand' }, brand));
      for (const p of rows) {
        const label = printerLabel(brand, p[0]);
        list.append(h('div', { class: 'pr' + (label === cur ? ' on' : ''), onclick: () => pick(label, [p[1], p[2], p[3]]) },
          h('span', {}, p[0]), h('span', {}, `${p[1]} × ${p[2]} × ${p[3]}`)));
      }
    }
    if (!list.children.length) list.append(h('div', { class: 'note', style: 'padding:10px' }, 'No match — type your printer\'s size below.'));
  };
  search.addEventListener('input', draw);
  const b = S.settings.bed;
  const cx = h('input', { type: 'number', value: b[0], min: 20, step: 1 }), cy = h('input', { type: 'number', value: b[1], min: 20, step: 1 }), cz = h('input', { type: 'number', value: b[2], min: 20, step: 1 });
  const custom = h('div', { class: 'printer-custom' }, h('b', {}, 'Custom (mm)'), cx, cy, cz);
  const body = h('div', { class: 'printer-pick' }, search, list, h('div', { class: 'note' }, 'Not listed? Type the build volume — width (X), depth (Y), height (Z) — from your printer\'s spec sheet:'), custom);
  draw();
  m = ui.modal('Choose your printer', body, [{ label: 'Cancel', run: () => { after?.(); } }, {
    label: 'Use custom size', primary: true, run: () => {
      const bed = [+cx.value, +cy.value, +cz.value];
      if (bed.some((x) => !(x >= 20))) { toast('Each size must be at least 20 mm', 'error'); return false; }
      setPrinter(findPrinterBySize(bed), bed); after?.();
    },
  }]);
  setTimeout(() => search.focus(), 30);
}

function saveSettings() { try { localStorage.setItem(SETTINGS, JSON.stringify(S.settings)); } catch {} }

function helpDialog() {
  const keys = [
    ['Orbit', 'Left-drag empty space / Right-drag'], ['Pan', 'Middle-drag / Shift+Right-drag'], ['Zoom', 'Mouse wheel'],
    ['Select several', 'Shift+click or Ctrl+drag a box'], ['Edit inside a group', 'Double-click it'],
    ['Move / Rotate / Resize tool', 'W / E / R'], ['Move freely (no snap)', 'Hold Alt while dragging'], ['Move along one axis', 'Hold Shift while dragging'],
    ['Copy while dragging', 'Alt+drag'], ['Nudge', 'Arrow keys (Shift = 10×)'], ['Raise / lower', 'Page Up / Page Down'],
    ['Group / Ungroup', 'Ctrl+G / Ctrl+Shift+G'], ['Hole ↔ Solid', 'H'], ['Duplicate', 'Ctrl+D'], ['Copy / Paste', 'Ctrl+C / Ctrl+V'],
    ['Delete', 'Delete / Backspace'], ['Drop to bed', 'D'], ['Mirror', 'M'], ['Align panel', 'L'], ['Auto-align to first selected', 'Shift+L'],
    ['Snap two parts together (face to face)', 'J'], ['Measure', 'Q'], ['Smart snapping off while dragging', 'Hold Alt'],
    ['Zoom to fit', 'F'], ['Home view', 'Home'], ['Front / Top / Right view', '1 / 7 / 3'], ['Undo / Redo', 'Ctrl+Z / Ctrl+Y'],
    ['Save / Open', 'Ctrl+S / Ctrl+O'], ['Export STL', 'Ctrl+E'], ['Select all', 'Ctrl+A'], ['Cancel', 'Esc'],
  ];
  const body = h('div', {},
    h('div', { class: 'callout tip', html: '<b>The 3 ideas behind Makie</b><br>1) Build parts from simple shapes. 2) Any shape can be a <b>Hole</b>. 3) <b>Group</b> shapes to merge them — holes in a group are cut out. That\'s it!' }),
    h('div', { class: 'keys' }, ...keys.map(([a, b]) => h('div', {}, h('span', {}, a), h('kbd', {}, b)))),
    h('div', { class: 'callout', html: '<b>Wirie boards:</b> the <b>Wirie</b> button on the toolbar (or Ctrl+Shift+O) opens a <b>.makie.json</b> board. You can also drag one onto the window, use File → Open, or right-click it in Explorer → Open with → Makie. Makie shows your circuit board with every part as a block. It is a locked reference that is not printed. Click a part to check or change its height. Press <b>Make a case for this board</b> for a base and lid with standoffs, connector openings, button and LED holes, and a clash check. New exports in Documents\\Wirie\\Exports show up by themselves.' }),
    h('div', { class: 'callout', html: '<b>Threads:</b> turn on <i>Thread</i> on any Cylinder or Tube (or use Threaded post / Threaded bore). A threaded solid and a threaded hole with the same diameter screw together — the clearance is built in. The exact thread figures are listed under the shape settings.<br><b>Motion:</b> switch to <i>Motion</i> at the top of the view, add joints and press Play; gears, racks and lead screws are linked automatically. Parts are solid: a moving part stops when it hits another and both turn red. <i>Print layout</i> lays every part out separately for printing — your assembly is never changed.<br><b>AMS / multi-colour:</b> give parts a filament slot (1–16) and export <i>3MF — multi-colour</i>.<br><b>Resize like Tinkercad:</b> drag the white handles on a selected part, or click a size label and type a number.<br><b>Gears:</b> right-click a gear → <i>Make matching part…</i>, or use the <i>Gear ratio calculator</i> in the Gears section.<br><b>Strength:</b> <i>Print check</i> marks walls thinner than 1.2 mm.' }),
    h('p', { class: 'note', html: 'Exact sizes: type numbers into the panel on the right (maths like <b>25.4*2</b> works). Drag the X/Y/Z letters to scrub values.<br>Exports are in millimetres and are watertight, ready for any slicer (PrusaSlicer, Cura, Bambu Studio, OrcaSlicer).' }));
  body.append(h('p', { class: 'note', style: 'text-align:center;margin-top:14px', html: `<b>Makie</b> v${S.version || '1.0'} · designed and created by <b>nrmluvdev</b>` }));
  ui.modal('Help & shortcuts', body, [{ label: 'Close', primary: true }], { wide: true });
}

async function printCheck() {
  const nodes = S.nodes.filter((n) => !n.hidden);
  if (!nodes.length) { toast('Add a part first.'); return; }
  const m = ui.modal('Print check', h('div', { class: 'note' }, 'Analysing your model…'), []);
  try {
    const r = await api('/api/analyze', { nodes: nodes.map(engineTree), q: S.settings.quality });
    const items = await r.json() || [];
    m.close();
    const bed = S.settings.bed;
    const looseHoles = nodes.filter((n) => sc.isHole(n)).length;
    const rows = [];
    const thin = [];
    let issues = 0;
    for (const it of items) {
      const s = it.stats, size = s.max.map((x, i) => x - s.min[i]);
      const problems = [];
      if (s.min[2] > 0.05) problems.push(['warn', `Floating ${round(s.min[2], 2)} mm above the bed — use Drop`]);
      if (s.min[2] < -0.05) problems.push(['bad', `Goes ${round(-s.min[2], 2)} mm below the bed`]);
      if (s.min[0] < -bed[0] / 2 - 0.01 || s.max[0] > bed[0] / 2 + 0.01 || s.min[1] < -bed[1] / 2 - 0.01 || s.max[1] > bed[1] / 2 + 0.01) problems.push(['warn', 'Partly outside the print bed']);
      if (s.max[2] > bed[2]) problems.push(['bad', 'Taller than your printer']);
      if (s.openEdges > 0) problems.push(['warn', `${s.openEdges} open edges (slicer will repair)`]);
      if (s.overhang > Math.max(20, s.area * 0.02)) problems.push(['warn', `${round(s.overhang / 100, 1)} cm² of steep overhang — may need supports`]);
      if (s.volume < 1) problems.push(['bad', 'Empty — nothing to print']);
      const wl = it.walls;
      if (wl && wl.totalArea > 0) {
        if (wl.critArea > Math.max(2, 0.005 * wl.totalArea)) problems.push(['bad', `Walls under 1 mm (thinnest ≈ ${round(wl.spots?.[0]?.thickness ?? wl.min, 2)} mm) — likely to split or not print`]);
        else if (wl.thinArea > Math.max(8, 0.03 * wl.totalArea)) problems.push(['warn', `Some walls are under 1.2 mm thick — fragile`]);
        for (const sp of (wl.spots || [])) if (sp.thickness < 1.15) thin.push({ at: sp.at, t: sp.thickness });
      }
      issues += problems.length;
      const vcm = s.volume / 1000;
      const shell = Math.min(s.volume, s.area * 1.2) / 1000;
      const grams = 1.24 * (shell + 0.15 * Math.max(0, vcm - shell));
      rows.push(h('tr', {},
        h('td', {}, h('b', {}, it.name || 'Part')),
        h('td', {}, size.map((x) => round(x, 1)).join(' × ') + ' mm'),
        h('td', {}, `${round(vcm, 2)} cm³`, h('div', { class: 'faint' }, `≈ ${round(grams, 1)} g PLA`)),
        h('td', {}, problems.length ? problems.map(([c, t]) => h('div', { class: c }, '⚠ ' + t)) : h('span', { class: 'ok' }, '✓ Ready to print'))));
    }
    const body = h('div', {},
      h('table', {}, h('tr', {}, h('th', {}, 'Part'), h('th', {}, 'Size'), h('th', {}, 'Material'), h('th', {}, 'Checks')), ...rows),
      looseHoles ? h('div', { class: 'callout', html: `<span class="warn">⚠ ${looseHoles} hole${looseHoles > 1 ? 's are' : ' is'} not grouped with anything.</span> Holes only cut when grouped with a solid, so ${looseHoles > 1 ? 'they' : 'it'} won't be printed.` }) : null,
      h('p', { class: 'note' }, 'Weight is a rough estimate for ~15% infill with 2–3 walls. Wall check: 1.2 mm (3 lines with a 0.4 mm nozzle) is the thinnest wall that prints strong; 1.6 mm or more around screws.'));
    ui.modal(issues ? `Print check — ${issues} thing${issues > 1 ? 's' : ''} to look at` : 'Print check — all good!', body, [
      ...(thin.length ? [{ label: `Show ${thin.length} thin spot${thin.length > 1 ? 's' : ''}`, run: () => { vp.showThinSpots(thin); } }] : []),
      { label: 'Drop everything to the bed', run: () => { S.sel = new Set(S.nodes.map((n) => n.id)); drop(); } },
      { label: 'Export…', run: () => setTimeout(() => exportFile('stl'), 50) },
      { label: 'Close', primary: true }], { wide: true });
  } catch (e) { m.close(); toast('Check failed: ' + e.message, 'error'); }
}

// ---------------- files ----------------
function b64(buf) { let s = ''; const b = new Uint8Array(buf.buffer || buf); for (let i = 0; i < b.length; i += 0x8000) s += String.fromCharCode.apply(null, b.subarray(i, i + 0x8000)); return btoa(s); }
function unb64(s) { const bin = atob(s); const b = new Uint8Array(bin.length); for (let i = 0; i < bin.length; i++) b[i] = bin.charCodeAt(i); return new Float32Array(b.buffer); }

function projectJSON() {
  const refs = new Set(); sc.walk(S.nodes, (n) => { if (n.meshRef) refs.add(n.meshRef); });
  const meshes = {};
  for (const r of refs) if (S.meshData[r]) meshes[r] = b64(S.meshData[r]);
  const tex = {}; sc.walk(S.nodes, (n) => { if (n.texRef && S.textures?.[n.texRef]) tex[n.texRef] = S.textures[n.texRef]; });
  return JSON.stringify({ app: 'Makie', version: 2, name: S.projectName, nodes: S.nodes, joints: S.joints || [], filaments: S.filaments, meshes, textures: tex, bed: S.settings.bed }, null, 0);
}

async function saveAs(blob, name, types) {
  if (window.showSaveFilePicker) {
    try {
      const hnd = await showSaveFilePicker({ suggestedName: name, types });
      const w = await hnd.createWritable(); await w.write(blob); await w.close();
      return true;
    } catch (e) { if (e.name === 'AbortError') return false; }
  }
  const a = h('a', { href: URL.createObjectURL(blob), download: name });
  document.body.append(a); a.click(); a.remove();
  setTimeout(() => URL.revokeObjectURL(a.href), 4000);
  return true;
}

async function saveProject() {
  const ok = await saveAs(new Blob([projectJSON()], { type: 'application/json' }), safe(S.projectName) + '.makie', [{ description: 'Makie project', accept: { 'application/json': ['.makie'] } }]);
  if (ok) { S.dirty = false; toast('Project saved', 'ok'); }
}

async function loadProject(text) {
  const p = JSON.parse(text);
  if (!p.nodes) throw new Error('Not a Makie project');
  for (const [ref, data] of Object.entries(p.meshes || {})) S.meshData[ref] = unb64(data);
  S.textures = { ...(S.textures || {}), ...(p.textures || {}) };
  S.nodes = p.nodes; S.sel.clear(); S.joints = p.joints || [];
  if (p.filaments) S.filaments = p.filaments;
  if (S.workspace && S.workspace !== 'design') mo.setWorkspace('design');
  S.projectName = p.name || 'Untitled part';
  $('#projname').value = S.projectName;
  undoStack = []; redoStack = []; lastSnap = snapState();
  welcomeDismissed = true;
  refresh(); autosave();
  setTimeout(() => vp.fitView(), 400);
}

function openProject() {
  const inp = $('#file-open');
  inp.onchange = async () => {
    const f = inp.files[0]; inp.value = '';
    if (!f) return;
    const text = await f.text();
    if (f.name.toLowerCase().endsWith('.makie.json') || text.slice(0, 4096).includes('wirie-case')) return wy.importFile(f);
    try { await loadProject(text); toast('Opened ' + f.name, 'ok'); } catch (e) { toast('Could not open: ' + e.message, 'error'); }
  };
  inp.click();
}

// files opened with Makie from Windows (Open with, or dropped on Makie.exe)
async function checkInbox() {
  let items = [];
  try { items = await (await fetch('/api/inbox')).json(); } catch { return; }
  for (const it of items) {
    if (it.kind === 'wirie') await wy.addParsed(it.wirie, it.name);
    else if (it.kind === 'project') { try { await loadProject(it.text); toast('Opened ' + it.name, 'ok'); } catch (e) { toast('Could not open ' + it.name + ': ' + e.message, 'error'); } }
    else if (it.kind === 'mesh') { const b = Uint8Array.from(atob(it.data), (c) => c.charCodeAt(0)); await importMeshFiles([new File([b], it.name)]); }
    else if (it.kind === 'error') ui.modal('Could not open “' + it.name + '”', h('p', {}, it.error), [{ label: 'OK', primary: true }]);
  }
}

function newProject() {
  if (S.nodes.length && !confirm('Start a new project? Unsaved changes will be lost (you can still Undo).')) return;
  S.nodes = []; S.sel.clear(); S.projectName = 'Untitled part'; $('#projname').value = S.projectName;
  commit('New');
  welcomeDismissed = false; refresh();
}

function importMesh() {
  const inp = $('#file-import');
  inp.onchange = async () => { await importMeshFiles([...inp.files]); inp.value = ''; };
  inp.click();
}

async function importMeshFiles(files) {
  for (const f of files) {
    try {
      const fmt = f.name.toLowerCase().endsWith('.obj') ? 'obj' : 'stl';
      const r = await api('/api/import?fmt=' + fmt, await f.arrayBuffer(), true);
      const { ref, tris } = await r.json();
      const raw = await (await fetch('/api/raw?ref=' + ref)).arrayBuffer();
      S.meshData[ref] = new Float32Array(raw);
      const n = sc.makeNode('mesh', {}, { meshRef: ref, name: f.name.replace(/\.(stl|obj)$/i, ''), color: '#94a3b8' });
      delete n.params;
      S.nodes.push(n); S.sel = new Set([n.id]);
      commit('Import');
      toast(`Imported ${f.name} (${tris.toLocaleString()} triangles)`, 'ok');
      welcomeDismissed = true;
      whenBuilt(n).then(() => vp.fitView([n.id]));
    } catch (e) { toast('Import failed: ' + e.message, 'error', 6000); }
  }
}

// drag files from Explorer onto the window: Wirie boards, meshes or projects
function setupFileDrop() {
  const hasFiles = (e) => [...(e.dataTransfer?.types || [])].includes('Files');
  let veil = null;
  const show = (on) => {
    if (on && !veil) { veil = h('div', { class: 'drop-veil' }, h('div', {}, h('b', {}, 'Drop to open'), h('span', {}, 'Wirie boards (.makie.json), STL / OBJ models or Makie projects'))); document.body.append(veil); }
    if (!on && veil) { veil.remove(); veil = null; }
  };
  window.addEventListener('dragover', (e) => { if (!hasFiles(e)) return; e.preventDefault(); e.dataTransfer.dropEffect = 'copy'; show(true); });
  window.addEventListener('dragleave', (e) => { if (e.relatedTarget == null) show(false); });
  window.addEventListener('drop', async (e) => {
    if (!hasFiles(e)) return;
    e.preventDefault(); show(false);
    const meshes = [];
    for (const f of e.dataTransfer.files) {
      const nm = f.name.toLowerCase();
      if (nm.endsWith('.stl') || nm.endsWith('.obj')) { meshes.push(f); continue; }
      if (nm.endsWith('.makie.json')) { await wy.importFile(f); continue; }
      if (nm.endsWith('.json')) {
        const head = await f.slice(0, 4096).text();
        if (head.includes('wirie-case')) { await wy.importFile(f); continue; }
      }
      if (/\.(makie|smithers|partsmith|json)$/.test(nm)) {
        try { await loadProject(await f.text()); toast('Opened ' + f.name, 'ok'); } catch (err) { toast('Could not open ' + f.name + ': ' + err.message, 'error'); }
        continue;
      }
      toast(`Makie can't open “${f.name}”. Drop a .makie.json Wirie board, an STL / OBJ model or a Makie project.`, 'error', 6000);
    }
    if (meshes.length) importMeshFiles(meshes);
  });
}

const safe = (s) => (s || 'part').replace(/[<>:"/\\|?*]+/g, '_').trim() || 'part';

async function exportFile(fmt, onlySel = false, preset) {
  let nodes = onlySel ? selRoots().map(sc.topOf).filter((v, i, a) => a.indexOf(v) === i) : S.nodes;
  nodes = nodes.filter((n) => !n.hidden && !sc.isHole(n));
  let tree = preset || (S.workspace === 'layout' ? mo.layoutNodes() : S.workspace === 'motion' ? mo.posedNodes() : null);
  if (tree && onlySel) { const ids = new Set(nodes.map((n) => n.id)); tree = tree.filter((t) => ids.has(t.id)); }
  if (tree) tree = tree.filter((t) => { const n = sc.node(t.id); return n && !n.hidden && !sc.isHole(n); });
  if (!nodes.length) { toast(onlySel ? 'Select a solid part to export.' : 'Nothing to export — add a solid part first. Holes alone are not printed.', 'error'); return; }
  return sendExport(fmt, tree || nodes.map(engineTree), S.projectName);
}

// send engine trees to the exporter and save the file
async function sendExport(fmt, tree, projName) {
  toast('Preparing ' + fmt.toUpperCase() + '…', 'info', 1500);
  try {
    const name = safe(projName);
    const body = tree.map((t) => withFilament(t));
    const r = await api('/api/export?fmt=' + fmt, { nodes: body, q: Math.max(S.settings.quality, 1.6), name, separate: fmt !== 'stl' });
    const blob = await r.blob();
    const ext = { stl: 'stl', '3mf': '3mf', '3mfasm': '3mf', obj: 'obj', stlzip: 'zip', colorzip: 'zip' }[fmt];
    const types = { stl: [{ description: 'STL mesh', accept: { 'model/stl': ['.stl'] } }], '3mf': [{ description: '3MF', accept: { 'model/3mf': ['.3mf'] } }], obj: [{ description: 'OBJ', accept: { 'text/plain': ['.obj'] } }], zip: [{ description: 'ZIP archive', accept: { 'application/zip': ['.zip'] } }] }[ext];
    if (await saveAs(blob, name + '.' + ext, types)) toast(`Exported ${name}.${ext} — open it in your slicer.` + (fmt === '3mfasm' ? ' Each part keeps its filament slot.' : ''), 'ok', 5000);
  } catch (e) { toast('Export failed: ' + e.message, 'error', 6000); }
}

async function quit() {
  try { localStorage.setItem(AUTOSAVE, projectJSON()); } catch {}
  try { await fetch('/api/quit'); } catch {}
  document.body.innerHTML = '<div style="padding:60px;font:16px system-ui;color:#9aa3b2">Makie has closed. Your work was saved and will be restored next time. You can close this window.</div>';
  setTimeout(() => window.close(), 400);
}

// ---------------- filaments (AMS / multi-material) ----------------
const DEFAULT_FILAMENTS = [
  { name: 'White PLA', color: '#f1f5f9' }, { name: 'Black PLA', color: '#1f2937' },
  { name: 'Red PLA', color: '#dc2626' }, { name: 'Blue PLA', color: '#2563eb' },
];
function hexDist(a, b) { const p = (h) => [1, 3, 5].map((i) => parseInt(h.slice(i, i + 2), 16)); const x = p(a), y = p(b); return Math.hypot(x[0] - y[0], x[1] - y[1], x[2] - y[2]); }
// slot used when exporting: the part's chosen slot, otherwise the closest filament colour
export function slotFor(n) {
  if (n.slot && n.slot <= S.filaments.length) return n.slot;
  if (!n.color || !/^#[0-9a-f]{6}$/i.test(n.color)) return 1;
  let best = 1, bd = Infinity;
  S.filaments.forEach((f, i) => { const d = hexDist(n.color, f.color); if (d < bd) { bd = d; best = i + 1; } });
  return best;
}
function withFilament(t) {
  const n = sc.node(t.id);
  if (n) { t.slot = slotFor(n); if (n.slot) t.color = S.filaments[n.slot - 1]?.color || t.color; }
  return t;
}
function setSlot(ids, slot) {
  for (const id of ids) {
    const n = sc.node(id); if (!n) continue;
    n.slot = slot; if (slot) n.color = S.filaments[slot - 1].color;
  }
  commit('Filament');
}
function filamentsDialog() {
  const body = h('div');
  const draw = () => {
    body.innerHTML = '';
    body.append(h('p', { class: 'note', html: 'Set these to match the spools loaded in your AMS / multi-material unit (slot 1 = first spool). Parts using a slot are exported to that filament in <b>3MF — multi-colour</b>.' }));
    S.filaments.forEach((f, i) => {
      body.append(h('div', { class: 'fil-row' },
        h('div', { class: 'slot', style: `background:${f.color}` }, String(i + 1)),
        h('input', { class: 'inp', value: f.name, onkeydown: (e) => e.stopPropagation(), onchange: (e) => { f.name = e.target.value; } }),
        h('input', { type: 'color', value: f.color, onchange: (e) => { f.color = e.target.value; sc.walk(S.nodes, (n) => { if (n.slot === i + 1) n.color = f.color; }); draw(); vp.requestRender(); } }),
        h('button', { class: 'mini', title: 'Remove slot', disabled: S.filaments.length <= 1, onclick: () => { S.filaments.splice(i, 1); sc.walk(S.nodes, (n) => { if (n.slot === i + 1) n.slot = 0; else if (n.slot > i + 1) n.slot--; }); draw(); } }, '✕')));
    });
    body.append(h('div', { class: 'btnrow' },
      h('button', { class: 'btn small', disabled: S.filaments.length >= 16, onclick: () => { S.filaments.push({ name: 'Filament ' + (S.filaments.length + 1), color: ['#16a34a', '#f59e0b', '#9333ea', '#0891b2', '#e11d48', '#a16207', '#64748b', '#f472b6'][S.filaments.length % 8] }); draw(); } }, '+ Add slot'),
      h('button', { class: 'btn small', onclick: () => { S.filaments = structuredClone(DEFAULT_FILAMENTS); draw(); } }, 'Reset'),
      h('span', { class: 'note' }, 'AMS: 4 slots per unit, up to 16.')));
  };
  draw();
  ui.modal('Filaments / AMS slots', body, [{ label: 'Done', primary: true, run: () => { try { localStorage.setItem('partsmith.filaments', JSON.stringify(S.filaments)); } catch {} commit('Filaments'); } }]);
}

// ---------------- autosave ----------------
let asTimer;
function autosave() {
  clearTimeout(asTimer);
  asTimer = setTimeout(() => {
    try { localStorage.setItem(AUTOSAVE, projectJSON()); } catch {
      try { localStorage.setItem(AUTOSAVE, JSON.stringify({ app: 'Makie', nodes: S.nodes, name: S.projectName })); } catch {}
    }
  }, 500);
}

// ---------------- context menu ----------------
function context(e, id) {
  if (id && !S.sel.has(id) && ![...S.sel].some((s) => sc.topOf(s)?.id === id)) select(id, 'set');
  const n = S.sel.size, one = n === 1 ? sc.node([...S.sel][0]) : null;
  if (!n) {
    ui.menu(e, [
      { label: 'Paste', key: 'Ctrl+V', run: paste, disabled: !clipboard },
      { label: 'Select all', key: 'Ctrl+A', run: () => select(sc.allIds().filter((i) => !sc.parentOf(i)), 'set') },
      '-', { label: 'Zoom to fit', key: 'F', run: () => vp.fitView() }, { label: 'Home view', key: 'Home', run: () => vp.setView('iso') },
      '-', { label: 'Gear ratio calculator…', run: () => gr.calculatorDialog() },
    ]);
    return;
  }
  ui.menu(e, [
    { label: one && sc.isHole(one) ? 'Make solid' : 'Make hole', key: 'H', run: () => run('toggleHole') },
    { label: 'Group', key: 'Ctrl+G', run: group, disabled: n < 2 },
    { label: 'Ungroup', key: 'Ctrl+Shift+G', run: ungroupSel, disabled: !(one && one.type === 'group') },
    '-',
    { label: 'Duplicate', key: 'Ctrl+D', run: () => duplicate() },
    { label: 'Copy', key: 'Ctrl+C', run: copy },
    { label: 'Mirror left ↔ right', run: () => mirror(0) },
    { label: 'Drop to bed', key: 'D', run: drop },
    { label: 'Lay flat on a face…', run: () => layFlat(selRoots()[0]), disabled: n !== 1 },
    { label: 'Split in two…', run: () => splitDialog(selRoots()[0]), disabled: n !== 1 || (one && sc.isHole(one)) },
    { label: 'Make matching part…', run: () => gr.matchingDialog(selRoots()[0]), disabled: n !== 1 || !gr.hasMatch(one) || (one && sc.isHole(one)) },
    { label: 'Zoom to selection', key: 'F', run: () => vp.fitView([...S.sel]) },
    '-',
    { label: one?.hidden ? 'Show' : 'Hide', run: () => { selRoots().forEach((i) => { sc.node(i).hidden = !sc.node(i).hidden; }); commit('Hide'); } },
    { label: 'Delete', key: 'Del', run: del },
  ]);
}

let clipboard = null;
function copy() { const ids = selRoots(); if (!ids.length) return; clipboard = ids.map((id) => ({ n: structuredClone(sc.node(id)), W: Array.from(sc.worldMatrix(id)) })); toast(`Copied ${ids.length} part${ids.length > 1 ? 's' : ''}`, 'info', 1200); }
function paste() {
  if (!clipboard) return;
  const ids = [];
  for (const c of clipboard) {
    const n = sc.cloneNode(c.n);
    S.nodes.push(n);
    sc.setWorld(n, mul(translate(10, 10, 0), Float64Array.from(c.W)), trs([0, 0, 0], [0, 0, 0], [1, 1, 1]));
    ids.push(n.id);
  }
  clipboard = clipboard.map((c) => ({ ...c, W: Array.from(mul(translate(10, 10, 0), Float64Array.from(c.W))) }));
  S.sel = new Set(ids);
  commit('Paste');
}

// ---------------- action dispatcher ----------------
function run(act, e) {
  if (act.startsWith('tool:')) { S.tool = act.slice(5); if (S.mode !== 'select') { vp.stopMeasure(); vp.cancelPlacing(); } ui.refreshToolbar(); vp.requestRender(); return; }
  switch (act) {
    case 'fileMenu': return ui.menu(e, [
      { label: 'New project', key: 'Ctrl+N', run: newProject },
      { label: 'Open project…', key: 'Ctrl+O', run: openProject },
      { label: 'Save project…', key: 'Ctrl+S', run: saveProject },
      '-',
      { label: 'Import STL / OBJ…', run: importMesh },
      { label: 'Open Wirie board (.makie.json)…', run: wy.chooseFile },
      '-',
      { label: 'Load the example', run: demo },
      '-',
      { label: 'Quit Makie', run: quit },
    ]);
    case 'wirieMenu': {
      const b = [...S.sel].map(sc.node).find((n) => n?.type === 'wirieboard') || (S.nodes.filter((n) => n.type === 'wirieboard').length === 1 ? S.nodes.find((n) => n.type === 'wirieboard') : null);
      const rec = wy.recentMenuItems();
      return ui.menu(e, [
        { label: 'Open Wirie board (.makie.json)…', key: 'Ctrl+Shift+O', run: wy.chooseFile },
        ...(rec.length ? ['-', { label: 'Recent from Wirie', disabled: true }, ...rec] : []),
        { label: 'Show all Wirie exports…', run: wy.recentDialog },
        '-',
        { label: b ? 'Make a case for ' + (b.wirie?.project || 'this board') : 'Make a case (open a board first)', disabled: !b, run: () => wy.openCasePanel(b.id) },
      ]);
    }
    case 'viewMenu': return ui.menu(e, [
      { label: (S.view.section ? '✓ ' : '') + 'Section (cut-away) view', run: () => run('section') },
      { label: (S.view.overhang ? '✓ ' : '') + 'Show overhangs needing support', run: () => run('overhang') },
      { label: (S.view.wire ? '✓ ' : '') + 'X-ray (see-through)', run: () => run('xray') },
      { label: (S.settings.edges !== false ? '✓ ' : '') + 'Show part edges', run: () => { S.settings.edges = S.settings.edges === false; saveSettings(); vp.requestRender(); } },
      { label: 'Printer / build plate size…', run: () => printerDialog() },
      '-',
      { label: 'Front view', key: '1', run: () => vp.setView('front') },
      { label: 'Right view', key: '3', run: () => vp.setView('right') },
      { label: 'Top view', key: '7', run: () => vp.setView('top') },
      { label: 'Home view', key: 'Home', run: () => vp.setView('iso') },
      { label: 'Zoom to fit', key: 'F', run: () => vp.fitView() },
    ]);
    case 'newProject': return newProject();
    case 'openProject': return openProject();
    case 'saveProject': return saveProject();
    case 'importMesh': return importMesh();
    case 'exportMenu': return ui.menu(e, [
      { label: 'STL — whole plate', key: 'Ctrl+E', run: () => exportFile('stl') },
      { label: 'STL — selected parts only', run: () => exportFile('stl', true), disabled: !S.sel.size },
      { label: '3MF — multi-colour, one object (AMS)', run: () => exportFile('3mfasm') },
      { label: '3MF — separate objects', run: () => exportFile('3mf') },
      { label: 'STL — one file per part (zip)', run: () => exportFile('stlzip') },
      { label: 'STL — one file per filament (zip)', run: () => exportFile('colorzip') },
      { label: 'OBJ', run: () => exportFile('obj') },
      '-', { label: 'Filaments / AMS slots…', run: filamentsDialog },
      '-', { label: 'Print check…', run: printCheck },
    ]);
    case 'undo': return undo();
    case 'redo': return redo();
    case 'group': return group();
    case 'ungroup': return ungroupSel();
    case 'toggleHole': { const ids = selRoots(); if (!ids.length) return; const anySolid = ids.some((id) => !sc.isHole(sc.node(id))); return setHole(ids, anySolid); }
    case 'alignTools': return ui.menu(e, [
      { label: 'Auto-align to first selected', key: 'Shift+L', run: () => al.autoAlign(selRoots()), disabled: selRoots().length < 2 },
      { label: 'Snap together (face to face)…', key: 'J', run: () => al.snapTogether() },
      { label: 'Align panel…', key: 'L', run: () => run('alignMenu'), disabled: selRoots().length < 2 },
      '-',
      { label: (S.settings.smartSnap !== false ? '✓ ' : '') + 'Smart snapping while dragging', run: () => { S.settings.smartSnap = S.settings.smartSnap === false; saveSettings(); toast(S.settings.smartSnap ? 'Smart snapping on — parts line up with others as you drag (hold Alt to switch off).' : 'Smart snapping off'); } },
    ]);
    case 'alignMenu': {
      const ids = selRoots();
      if (ids.length < 2) { toast('Select two or more parts to align them.'); return; }
      const body = h('div', {}, h('p', { class: 'note' }, 'Line up the selected parts with each other:'), ui.alignGrid(ids));
      body.addEventListener('click', (ev) => { if (ev.target.tagName === 'BUTTON') m.close(); });
      const m = ui.modal('Align', body, [{ label: 'Done', primary: true }]);
      return;
    }
    case 'mirrorMenu': return ui.menu(e, [
      { label: 'Mirror left ↔ right (X)', run: () => mirror(0) },
      { label: 'Mirror front ↔ back (Y)', run: () => mirror(1) },
      { label: 'Flip upside down (Z)', run: () => mirror(2) },
    ]);
    case 'duplicate': return duplicate();
    case 'patternDialog': return patternDialog();
    case 'drop': return drop();
    case 'delete': return del();
    case 'measure': if (S.mode === 'measure') vp.stopMeasure(); else { vp.cancelPlacing(); vp.startMeasure(); } ui.refreshToolbar(); return;
    case 'section': {
      S.view.section = !S.view.section;
      $('#section-ctl').classList.toggle('hidden', !S.view.section);
      if (S.view.section) setupSection();
      ui.refreshToolbar(); vp.requestRender(); return;
    }
    case 'overhang': S.view.overhang = !S.view.overhang; ui.refreshToolbar(); vp.requestRender();
      if (S.view.overhang) toast('Red faces point steeply downward and will need supports (or a chamfer).', 'info', 4500); return;
    case 'xray': S.view.wire = !S.view.wire; ui.refreshToolbar(); vp.requestRender(); return;
    case 'printCheck': return printCheck();
    case 'settings': return settingsDialog();
    case 'printer': return printerDialog();
    case 'help': return helpDialog();
  }
}

function setupSection() {
  const box = sc.selectionBox(S.nodes.map((n) => n.id)) || [[-50, -50, 0], [50, 50, 50]];
  const ax = S.view.sectionAxis;
  const r = $('#section-pos');
  r.min = Math.floor(box[0][ax] - 1); r.max = Math.ceil(box[1][ax] + 1);
  S.view.sectionPos = round((box[0][ax] + box[1][ax]) / 2, 1);
  r.value = S.view.sectionPos;
  $('#section-val').textContent = S.view.sectionPos + ' mm';
}

// ---------------- keyboard ----------------
function onKey(e) {
  if (e.target.matches('input, textarea, select')) return;
  const k = e.key, ctrl = e.ctrlKey || e.metaKey;
  const nudge = (d) => { const ids = selRoots(); if (!ids.length) return; moveWorld(ids, d); commit('Nudge'); };
  const g = (S.settings.grid || 1) * (e.shiftKey ? 10 : 1);
  if (ctrl) {
    const map = { z: undo, y: redo, d: () => duplicate(), s: saveProject, o: openProject, e: () => exportFile('stl'), c: copy, v: paste, a: () => select(S.nodes.map((n) => n.id), 'set'), n: newProject };
    if (k.toLowerCase() === 'g') { e.preventDefault(); return e.shiftKey ? ungroupSel() : group(); }
    if (k.toLowerCase() === 'o' && e.shiftKey) { e.preventDefault(); return wy.chooseFile(); }
    if (k.toLowerCase() === 'z' && e.shiftKey) { e.preventDefault(); return redo(); }
    const f = map[k.toLowerCase()];
    if (f) { e.preventDefault(); f(); }
    return;
  }
  switch (k) {
    case 'Escape':
      if (S.mode === 'facepick') vp.stopFacePick(); else if (S.mode === 'place') vp.cancelPlacing(); else if (S.mode === 'measure') { vp.stopMeasure(); ui.refreshToolbar(); }
      else if (S.sel.size) { const id = [...S.sel][0]; const p = sc.parentOf(id); select(p ? p.id : null, p ? 'set' : 'clear'); }
      ui.hideMenu(); break;
    case 'Delete': case 'Backspace': del(); break;
    case 'w': case 'W': run('tool:move'); break;
    case 'e': case 'E': run('tool:rotate'); break;
    case 'r': case 'R': run('tool:scale'); break;
    case 'h': case 'H': run('toggleHole'); break;
    case 'd': case 'D': drop(); break;
    case 'm': case 'M': mirror(0); break;
    case 'l': case 'L': if (e.shiftKey) al.autoAlign(selRoots()); else run('alignMenu'); break;
    case 'j': case 'J': al.snapTogether(); break;
    case 'q': case 'Q': run('measure'); break;
    case 'f': case 'F': vp.fitView(S.sel.size ? [...S.sel] : null); break;
    case 'Home': vp.setView('iso'); break;
    case '1': vp.setView('front'); break; case '3': vp.setView('right'); break; case '7': vp.setView('top'); break; case '9': vp.setView('bottom'); break;
    case '?': helpDialog(); break;
    case 'ArrowLeft': e.preventDefault(); nudge([-g, 0, 0]); break;
    case 'ArrowRight': e.preventDefault(); nudge([g, 0, 0]); break;
    case 'ArrowUp': e.preventDefault(); nudge([0, g, 0]); break;
    case 'ArrowDown': e.preventDefault(); nudge([0, -g, 0]); break;
    case 'PageUp': e.preventDefault(); nudge([0, 0, g]); break;
    case 'PageDown': e.preventDefault(); nudge([0, 0, -g]); break;
  }
}

// ---------------- demo ----------------
function demo() {
  // A working hand-cranked 9 : 1 gearbox: a light-weighted base plate with axles,
  // a driver gear with a crank, a compound gear and an output gear — all linked in Motion.
  const m = 2, H = 4;
  const drv = [-40, -20], cmp = [8, -20], out = [8, 28];
  const plate = sc.makeNode('plate', { w: 120, d: 100, h: H, r: 8 }); plate.pos = [-10, 5, 0]; plate.color = '#334155'; plate.name = 'Base plate';
  const axle = (x, y, h) => { const a = sc.makeNode('cylinder', { d: 4.6, h, edges: 'Chamfered', edge: 0.6, edgeWhere: 'Top only' }); a.pos = [x, y, H - 0.01]; a.color = '#334155'; a.name = 'Axle'; return a; };
  const shoulder = sc.makeNode('cylinder', { d: 10, h: 6.5 }); shoulder.pos = [out[0], out[1], H - 0.01]; shoulder.color = '#334155'; shoulder.name = 'Spacer';
  const holes = [[-64, -39], [44, -39], [44, 49], [-64, 49]].map(([x, y]) => { const n = sc.makeNode('screwhole', { size: 'M3', head: 'Countersink', depth: H + 0.4 }); n.pos = [x, y, H]; return n; });
  const vor = sc.makeNode('patterncut', { kind: 'Voronoi (organic)', w: 30, d: 38, cell: 9, strut: 2, depth: H + 1 }); vor.pos = [-50, 26, H]; vor.name = 'Voronoi light-weighting';
  const label = sc.makeNode('text', { text: 'MAKIE 9:1', size: 6, h: 1, font: 'Inter Black' }); label.pos = [-30, -41, H - 0.8]; label.hole = true; label.name = 'Engraved text';
  const parts = [plate, axle(...drv, 18), axle(...cmp, 22), axle(...out, 20), shoulder, ...holes, vor, label];
  S.nodes.push(...parts);
  const base = sc.groupNodes(parts.map((x) => x.id));
  base.name = 'Gearbox base';
  const driver = sc.makeNode('gear', { module: m, teeth: 12, h: 6, bore: 5 }); driver.pos = [drv[0], drv[1], H + 0.2]; driver.color = '#f97316'; driver.name = 'Driver 12T';
  const crank = sc.makeNode('crank', { len: 18, t: 4, bore: 5 }); crank.pos = [drv[0], drv[1], H + 6.3]; crank.color = '#f97316'; crank.name = 'Crank';
  const comp = sc.makeNode('compoundgear', { module: m, teeth: 36, teeth2: 12, h: 6, h2: 7, bore: 5 }); comp.pos = [cmp[0], cmp[1], H + 0.2]; comp.rot = [0, 0, 180 - 180 / 36]; comp.color = '#22c55e'; comp.name = 'Compound 36/12T';
  const output = sc.makeNode('gear', { module: m, teeth: 36, h: 6, bore: 5 }); output.pos = [out[0], out[1], H + 6.7]; output.color = '#0ea5e9'; output.name = 'Output 36T';
  // phase the output against the compound's small gear (centre to centre along +Y)
  { const rotA = comp.rot[2], pa = 360 / 12, f = ((((90 - rotA) / pa) % 1) + 1) % 1; output.rot = [0, 0, 90 + 180 + (f + 0.5) * (360 / 36)]; }
  const knob = sc.makeNode('starknob', {}); knob.pos = [out[0], out[1], H + 12.8]; knob.color = '#0ea5e9'; knob.name = 'Output knob';
  S.nodes.push(driver, crank, comp, output, knob);
  const axis = [0, 0, 1];
  const jd = { id: sc.newId(), name: 'Turn the crank', type: 'rotate', part: driver.id, followers: [crank.id], origin: [drv[0], drv[1], H], axis, min: -180, max: 180, speed: 120, loop: false };
  const jc = { id: sc.newId(), name: 'Compound gear', type: 'rotate', part: comp.id, followers: [], origin: [cmp[0], cmp[1], H], axis, min: -180, max: 180, speed: 60, loop: false, driver: jd.id, ratio: -12 / 36 };
  const jo = { id: sc.newId(), name: 'Output', type: 'rotate', part: output.id, followers: [knob.id], origin: [out[0], out[1], H], axis, min: -180, max: 180, speed: 60, loop: false, driver: jc.id, ratio: -12 / 36 };
  S.joints = [jd, jc, jo];
  S.sel = new Set([driver.id]);
  S.projectName = 'Example gearbox'; $('#projname').value = S.projectName;
  welcomeDismissed = true;
  commit('Example');
  setTimeout(() => vp.fitView(), 900);
  toast('A working 9:1 hand-cranked gearbox. Switch to Motion and press Play to watch it turn — then try Print layout to lay every part out for printing. The base is a group: plate, axles, countersunk screw holes, a Voronoi cut-out and engraved text.', 'info', 12000);
}

// ---------------- boot ----------------
async function boot() {
  try { Object.assign(S.settings, JSON.parse(localStorage.getItem(SETTINGS) || '{}')); } catch {}
  applyTheme();
  updatePrinterChip();
  document.getElementById('printer-chip').onclick = () => printerDialog();
  let cat;
  try { cat = await (await fetch('/api/catalog')).json(); } catch { document.body.innerHTML = '<p style="padding:40px">The Makie engine is not running. Please start Makie again.</p>'; return; }
  S.catalog = cat.shapes.filter((d) => !d.hidden); S.fonts = cat.fonts; S.version = cat.version;
  for (const d of cat.shapes) S.byType[d.type] = d;
  const actions = {
    run, place, select, wirieInspector: (el, n) => wy.inspector(el, n),
    matchingPart: (id) => gr.matchingDialog(id), gearCalc: () => gr.calculatorDialog(),
    update, setParam, setHole, setPosition, setSize, align, distribute, treeDrop, editSketch, cornerHoles, matchingHole, matchingThread, scanModel, setSlot, filamentsDialog, addJoint: () => mo.addJointDialog(), splitDialog, layFlat, autoAlign: (ids) => al.autoAlign(ids), snapTogether: () => al.snapTogether(),
    reparent: (id, p) => { sc.reparent(id, p); commit('Take out of group'); }, context,
    canUndo: () => undoStack.length > 0, canRedo: () => redoStack.length > 0,
    endDrag: () => { if (vp.isPlacing() && S.mode === 'place') vp.cancelPlacing(); },
  };
  ui.initUI(actions);
  al.initAlign({ commit, refresh, selRoots });
  mo.initMotion({ commit, refresh, exportLayout: (f) => exportFile(f === '3mf' ? '3mfasm' : f) });
  gr.initGears({ commit, fit: (ids) => vp.fitView(ids) });
  wy.init({
    commit, refresh, select, run, fitView: (ids) => vp.fitView(ids), requestRender: vp.requestRender,
    refreshInspector: () => ui.renderInspector(), setWorkspace: (w) => mo.setWorkspace(w),
    exportTree: (fmt, tree, name) => sendExport(fmt, tree, name), dismissWelcome: () => { welcomeDismissed = true; refresh(); },
  });
  setupFileDrop();
  checkInbox(); setInterval(checkInbox, 1500);
  try { S.filaments = JSON.parse(localStorage.getItem('partsmith.filaments') || 'null') || structuredClone(DEFAULT_FILAMENTS); } catch { S.filaments = structuredClone(DEFAULT_FILAMENTS); }
  S.workspace = 'design';
  document.querySelectorAll('#modes button').forEach((b) => (b.onclick = () => { vp.cancelPlacing(); vp.stopMeasure(); mo.setWorkspace(b.dataset.ws); }));
  ui.renderToolbar();
  vp.initViewport($('#gl'), {
    onSelect: select, onCommit: commit, onLive: () => { ui.refreshInspectorValues(); status(); }, onPlace, make,
    onContext: context, duplicate: (ids, inPlace) => duplicate(ids, inPlace),
  });
  on('geom', (gid) => { vp.requestRender(); if (typeof gid === 'string' && gid.startsWith('thumb_')) return; const rel = [...S.sel].some((sid) => sid === gid || sc.topOf(sid)?.id === gid || sc.isDescendant(gid, sid)); if (!rel && S.sel.size) { status(); return; } clearTimeout(boot.t); boot.t = setTimeout(() => { if (!document.activeElement?.closest('#inspector')) ui.renderInspector(); else ui.refreshInspectorValues(); status(); }, 60); });
  on('mode', () => { ui.refreshToolbar(); });
  document.addEventListener('keydown', onKey);
  $('#libsearch').addEventListener('input', ui.renderLibrary);
  $('#libsearch').addEventListener('keydown', (e) => { e.stopPropagation(); if (e.key === 'Enter') { const t = document.querySelector('#libgrid .tile'); t?.click(); } });
  $('#projname').addEventListener('change', (e) => { S.projectName = e.target.value; autosave(); });
  $('#projname').addEventListener('keydown', (e) => { e.stopPropagation(); if (e.key === 'Enter') e.target.blur(); });
  $('#ol-group').onclick = group; $('#ol-ungroup').onclick = ungroupSel;
  document.querySelectorAll('[data-view]').forEach((b) => (b.onclick = () => vp.setView(b.dataset.view)));
  $('#btn-fit').onclick = () => vp.fitView(S.sel.size ? [...S.sel] : null);
  $('#btn-ortho').onclick = (e) => { const o = vp.toggleOrtho(); e.target.classList.toggle('on', o); e.target.textContent = o ? 'O' : 'P'; toast(o ? 'Orthographic view' : 'Perspective view', 'info', 1000); };
  $('#section-axis').onchange = (e) => { S.view.sectionAxis = +e.target.value; setupSection(); vp.requestRender(); };
  $('#section-pos').oninput = (e) => { S.view.sectionPos = +e.target.value; $('#section-val').textContent = S.view.sectionPos + ' mm'; vp.requestRender(); };
  $('#section-close').onclick = () => run('section');
  $('#w-demo').onclick = demo;
  $('#w-wirie').onclick = () => wy.chooseFile();
  $('#w-close').onclick = () => { welcomeDismissed = true; refresh(); };
  // splitter
  $('#splitter').addEventListener('pointerdown', (e) => {
    const side = $('#side'), ol = $('#outliner');
    const mv = (ev) => { const r = side.getBoundingClientRect(); const hh = Math.max(90, Math.min(r.height - 140, r.bottom - ev.clientY)); ol.style.flex = `0 0 ${hh}px`; };
    const up = () => { window.removeEventListener('pointermove', mv); window.removeEventListener('pointerup', up); };
    window.addEventListener('pointermove', mv); window.addEventListener('pointerup', up);
    e.preventDefault();
  });
  window.addEventListener('beforeunload', () => { try { localStorage.setItem(AUTOSAVE, projectJSON()); } catch {} });

  // restore the last session
  const saved = localStorage.getItem(AUTOSAVE);
  if (saved) {
    try {
      const p = JSON.parse(saved);
      if (p.nodes?.length) { await loadProject(saved); toast('Welcome back — your last session was restored.', 'ok'); }
    } catch {}
  }
  lastSnap = snapState();
  refresh();
  ui.renderLibrary();
  setTimeout(() => ui.makeThumbs(cat.version + ':' + S.catalog.length), 50);
  // keep the engine alive and detect if it stops
  const ping = async () => {
    try { await fetch('/api/ping'); $('#engine').classList.remove('down'); } catch { $('#engine').classList.add('down'); $('#engine').title = 'Engine not responding — restart Makie'; }
  };
  ping(); setInterval(ping, 20000);
  hint('Click a shape on the left to add it. Press <kbd>?</kbd> for help.');
  window.PS = { S, sc, vp, mo, al, run, select, commit, demo, exportFile }; // handy for debugging
}

boot();
