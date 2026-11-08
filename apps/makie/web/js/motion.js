// Motion simulation (joints) and the print-layout view. Both only change how parts are
// *shown* (S.override); the design itself is never modified, so parts stay separate for printing.
import { S, toast, hint, emit } from './state.js';
import * as sc from './scene.js';
import * as vp from './viewport.js';
import * as ui from './ui.js';
import { getGeom, whenBuilt, api, engineTree } from './api.js';
import { v3, mul, translate, rotAxis, rotBetween, apply, decompose, trs, round, m4, invert } from './math.js';
import { faceInfo } from './align.js';
import { partsOverlap } from './collide.js';

const { h, $ } = ui;
let C; // { commit, refresh }
export function initMotion(cb) { C = cb; S.joints ||= []; }

// ---------------- workspace modes ----------------
export function setWorkspace(mode) {
  stopPlay();
  S.workspace = mode;
  S.override = {};
  S.clash = null;
  document.querySelectorAll('#modes button').forEach((b) => b.classList.toggle('on', b.dataset.ws === mode));
  document.body.dataset.ws = mode;
  $('#motion-panel').classList.toggle('hidden', mode !== 'motion');
  $('#layout-panel').classList.toggle('hidden', mode !== 'layout');
  if (mode === 'motion') { renderMotion(); applyMotion(); hint('<b>Motion mode:</b> add joints, then drag the sliders or press Play. Your design is not changed.'); }
  else if (mode === 'layout') { renderLayout(); computeLayout(); hint('<b>Print layout:</b> every part laid out separately on the bed, ready to export. Your assembly is untouched.'); }
  else hint('');
  vp.requestRender(); emit('workspace');
}
export const previewing = () => S.workspace === 'motion' || S.workspace === 'layout';

// ---------------- joints ----------------
const jointValue = (j) => {
  if (j.driver) {
    const d = S.joints.find((x) => x.id === j.driver);
    if (d) return jointValue(d) * (j.ratio ?? 1) * (j.reverse ? -1 : 1);
  }
  return S.simValues?.[j.id] ?? 0;
};

function jointMatrix(j, seen = new Set()) {
  if (seen.has(j.id)) return m4();
  seen.add(j.id);
  const v = jointValue(j);
  let M = j.type === 'slide' ? translate(...v3.mul(j.axis, v)) : mul(translate(...j.origin), mul(rotAxis(j.axis, v), translate(...v3.mul(j.origin, -1))));
  if (j.on) { const p = S.joints.find((x) => x.id === j.on); if (p) M = mul(jointMatrix(p, seen), M); }
  return M;
}

export function applyMotion() {
  S.override = {};
  for (const j of S.joints) {
    const M = jointMatrix(j);
    for (const id of [j.part, ...(j.followers || [])]) if (sc.node(id)) S.override[id] = M;
  }
  vp.requestRender();
}

function partLabel(id) { return sc.node(id)?.name || 'missing part'; }

function suggestRatio(driver, part, type) {
  const a = sc.node(driver.part), b = sc.node(part);
  if (!a || !b) return { ratio: 1, why: '' };
  const ta = a.params?.teeth, tb = b.params?.teeth;
  const G = ['gear', 'compoundgear', 'lantern', 'intermittent', 'sectorgear', 'bevelgear', 'crowngear', 'sprocket', 'gt2pulley'];
  if (driver.type === 'rotate' && type === 'rotate' && G.includes(a.type) && G.includes(b.type) && ta && tb)
    return { ratio: -ta / tb, reverse: false, why: `gears ${ta}:${tb} → ${round(ta / tb, 3)}× speed, opposite direction` };
  if (driver.type === 'rotate' && type === 'slide' && G.includes(a.type) && b.type === 'rack')
    return { ratio: (Math.PI * a.params.module * ta) / 360, why: `rack & pinion: ${round(Math.PI * a.params.module * ta, 2)} mm per turn` };
  const info = getGeom(a)?.info || '';
  const lead = info.match(/Lead ([\d.]+) mm/);
  if (driver.type === 'rotate' && type === 'slide' && lead) return { ratio: parseFloat(lead[1]) / 360, why: `lead screw: ${lead[1]} mm per turn` };
  return { ratio: 1, why: '' };
}

export function addJointDialog() {
  const id = [...S.sel].map((i) => sc.topOf(i)?.id).find(Boolean);
  if (!id) { toast('First click the part that should move, then add a joint.'); return; }
  const others = S.joints.filter((j) => j.part !== id);
  const f = ui.form([
    { key: 'type', label: 'Movement', type: 'select', options: ['Rotate (hinge, wheel, gear)', 'Slide (rail, piston, rack)'], def: 'Rotate (hinge, wheel, gear)' },
    { key: 'axis', label: 'Axis', type: 'select', options: ['Click a face to set it', 'Z (up)', 'X', 'Y'], def: 'Click a face to set it', help: 'Click the top of a gear, a round shaft or a hole and Makie finds its centre line.' },
    { key: 'driver', label: 'Driven by', type: 'select', options: ['Nothing — I move it', ...others.map((j) => j.name)], def: 'Nothing — I move it' },
    { key: 'on', label: 'Rides on', type: 'select', options: ['Fixed to the ground', ...others.map((j) => j.name)], def: 'Fixed to the ground', help: 'For linkages: the joint moves along with another moving part (e.g. a pin on a turning wheel).' },
    { key: 'min', label: 'Lower limit', def: -180, step: 5, showIf: (v) => v.driver.startsWith('Nothing') },
    { key: 'max', label: 'Upper limit', def: 180, step: 5, showIf: (v) => v.driver.startsWith('Nothing') },
    { key: 'speed', label: 'Play speed (per second)', def: 60, step: 5, showIf: (v) => v.driver.startsWith('Nothing') },
    { key: 'loop', label: 'Back and forth', type: 'bool', def: false, showIf: (v) => v.driver.startsWith('Nothing') },
    { key: 'followers', label: 'Also move the parts selected with it', type: 'bool', def: true },
  ]);
  ui.modal('Add a joint to “' + partLabel(id) + '”', f.el, [{ label: 'Cancel' }, {
    label: 'Next', primary: true, run: () => {
      const v = f.vals;
      const type = v.type.startsWith('Slide') ? 'slide' : 'rotate';
      const box = sc.worldBox(id);
      const ctr = v3.mul(v3.add(box[0], box[1]), 0.5);
      const followers = v.followers ? [...S.sel].map((i) => sc.topOf(i)?.id).filter((x) => x && x !== id) : [];
      const make = (origin, axis) => {
        const driver = others.find((j) => j.name === v.driver);
        const on = others.find((j) => j.name === v.on);
        const j = { id: sc.newId(), name: `${type === 'rotate' ? 'Turn' : 'Slide'} ${partLabel(id)}`, type, part: id, followers, origin, axis: v3.norm(axis), min: v.min, max: v.max, speed: v.speed, loop: v.loop };
        if (driver) { const sg = suggestRatio(driver, id, type); j.driver = driver.id; j.ratio = sg.ratio; if (sg.why) toast('Linked: ' + sg.why, 'ok', 5000); }
        if (on) j.on = on.id;
        if (type === 'slide') { j.min = Math.min(v.min, 0) === -180 ? -20 : v.min; j.max = v.max === 180 ? 20 : v.max; j.speed = v.speed === 60 ? 10 : v.speed; }
        S.joints.push(j);
        C.commit('Add joint'); renderMotion(); applyMotion();
      };
      if (v.axis.startsWith('Click')) {
        setTimeout(() => vp.startFacePick('<b>Joint axis:</b> click a face — the top of a gear, a round shaft, a hole or a flat face.', (hit) => {
          const fi = faceInfo(hit);
          if (fi.kind === 'round') make(fi.point, fi.axis);
          else make(fi.point, fi.normal);
        }), 30);
      } else make(ctr, v.axis[0] === 'Z' ? [0, 0, 1] : v.axis[0] === 'X' ? [1, 0, 0] : [0, 1, 0]);
    },
  }]);
}

// ---------------- playback ----------------
let raf = 0, last = 0;
function stopPlay() { cancelAnimationFrame(raf); raf = 0; const b = $('#sim-play'); if (b) b.textContent = '▶ Play'; }

// advance every free (driver) joint by dt seconds
function stepValues(dt) {
  S.simValues ||= {};
  for (const j of S.joints) {
    if (j.driver) continue;
    let v = (S.simValues[j.id] || 0) + (j.dir || 1) * (j.speed || 30) * dt;
    if (j.loop || j.type === 'slide') {
      if (v > j.max) { v = j.max; j.dir = -1; } if (v < j.min) { v = j.min; j.dir = 1; }
    } else if (j.type === 'rotate' && (j.max - j.min) < 359) {
      if (v > j.max || v < j.min) { j.dir = -(j.dir || 1); v = Math.max(j.min, Math.min(j.max, v)); }
    }
    S.simValues[j.id] = v;
  }
}

function togglePlay() {
  if (raf) return stopPlay();
  computeBaseline();
  S.clash = null;
  $('#sim-play').textContent = '❚❚ Pause';
  last = performance.now();
  const step = (t) => {
    let dt = Math.min(0.1, (t - last) / 1000) * (S.simSpeed || 1); last = t;
    // With collision checks on, never jump more than a couple of check-steps per frame:
    // on a slow computer the machine runs a little slower but stays smooth and exact.
    if (S.collide !== false) {
      let fastest = 0;
      for (const j of S.joints) if (!j.driver) fastest = Math.max(fastest, (j.speed || 30) / (j.type === 'rotate' ? 3 : 0.75));
      if (fastest > 0) dt = Math.min(dt, 2 / fastest);
    }
    const prev = { ...(S.simValues || {}) };
    stepValues(dt);
    if (!moveTo(prev, { ...S.simValues })) {
      // jammed: next Play runs the other way, out of the jam
      for (const j of S.joints) if (!j.driver) j.dir = -(j.dir || 1);
      stopPlay(); syncSliders(); return;
    }
    syncSliders();
    raf = requestAnimationFrame(step);
  };
  raf = requestAnimationFrame(step);
}

// ---------------- live collision detection ----------------
// Parts that already overlap in the starting position (an axle pushed into a hole, parts
// meant to be glued together) are ignored; any *new* overlap during the movement counts.
let baseline = new Set();
let startClash = null;
const pairKey = (a, b) => (a < b ? a + '|' + b : b + '|' + a);

function bodies() {
  return S.nodes.filter((n) => !n.hidden && !sc.isHole(n)).map((n) => {
    const e = getGeom(n);
    const o = S.override?.[n.id];
    const W = sc.worldMatrix(n.id);
    return { id: n.id, e, M: o ? mul(o, W) : W, o };
  }).filter((b) => b.e && b.e.tris);
}

const sameMove = (a, b) => {
  if (!a.o && !b.o) return true;
  if (!a.o || !b.o) return false;
  for (let i = 0; i < 16; i++) if (Math.abs(a.o[i] - b.o[i]) > 1e-9) return false;
  return true;
};

function overlaps(onlyMoving = true) {
  const bs = bodies();
  const hits = [];
  for (let i = 0; i < bs.length; i++) for (let k = i + 1; k < bs.length; k++) {
    const a = bs[i], b = bs[k];
    if (onlyMoving && sameMove(a, b)) continue; // not moving relative to each other
    const key = pairKey(a.id, b.id);
    if (onlyMoving && baseline.has(key)) continue;
    const at = partsOverlap(a.e, a.M, b.e, b.M);
    if (at) hits.push({ a: a.id, b: b.id, at, key });
  }
  return hits;
}

// Parts that move against each other but already overlap in the starting position would be
// jammed in the real machine. They're shown in red and listed, and then ignored so the rest of
// the movement can still be explored.
function computeBaseline() {
  const saved = { ...(S.simValues || {}) };
  S.simValues = {};
  applyMotion();
  const bs = bodies();
  const start = [];
  for (let i = 0; i < bs.length; i++) for (let k = i + 1; k < bs.length; k++) {
    if (sameMove(bs[i], bs[k]) && !moverPair(bs[i].id, bs[k].id)) continue;
    if (partsOverlap(bs[i].e, bs[i].M, bs[k].e, bs[k].M)) start.push({ a: bs[i].id, b: bs[k].id, key: pairKey(bs[i].id, bs[k].id) });
  }
  baseline = new Set(start.map((h) => h.key));
  startClash = start.length ? new Set(start.flatMap((h) => [h.a, h.b])) : null;
  S.simValues = saved;
  applyMotion();
  if (start.length && S.collide !== false) {
    const msg = start.map((h) => `${partLabel(h.a)} ↔ ${partLabel(h.b)}`).join(' · ');
    S.clash = new Set(start.flatMap((h) => [h.a, h.b]));
    const st = $('#sim-status');
    if (st) st.innerHTML = `<span class="bad">⚠ Already overlapping before anything moves (would be jammed): ${msg}</span>`;
    toast('⚠ These parts move against each other but already overlap at the start, so the real machine would be stuck: ' + msg, 'error', 8000);
  }
  return start;
}

// do these two parts move relative to each other at some point? (a joint moves one but not the other)
function moverPair(a, b) {
  const moves = (id) => S.joints.some((j) => j.part === id || (j.followers || []).includes(id));
  if (!moves(a) && !moves(b)) return false;
  const jointOf = (id) => S.joints.find((j) => j.part === id || (j.followers || []).includes(id))?.id;
  return jointOf(a) !== jointOf(b);
}

const lerpVals = (a, b, f) => { const o = {}; for (const k of new Set([...Object.keys(a), ...Object.keys(b)])) o[k] = (a[k] ?? 0) + ((b[k] ?? 0) - (a[k] ?? 0)) * f; return o; };

// Move from joint values `from` to `to`. If parts would run into each other, stop at the
// moment they touch, show them in red and return false.
function moveTo(from, to) {
  if (S.collide === false) { S.simValues = to; applyMotion(); return true; }
  // walk there in small steps so a fast move can't jump straight through a part
  let span = 0;
  for (const j of S.joints) {
    if (j.driver) continue;
    const d = Math.abs((to[j.id] ?? 0) - (from[j.id] ?? 0));
    span = Math.max(span, j.type === 'rotate' ? d / 3 : d / 0.75);
  }
  const n = Math.max(1, Math.min(240, Math.ceil(span)));
  let hits = [], f0 = 0, f1 = 1;
  for (let k = 1; k <= n; k++) {
    S.simValues = lerpVals(from, to, k / n);
    applyMotion();
    hits = overlaps();
    if (hits.length) { f0 = (k - 1) / n; f1 = k / n; break; }
  }
  if (!hits.length) { if (S.clash !== startClash) { S.clash = startClash; vp.requestRender(); } return true; }
  let lo = f0, hi = f1;
  for (let i = 0; i < 10; i++) {
    const mid = (lo + hi) / 2;
    S.simValues = lerpVals(from, to, mid);
    applyMotion();
    const hm = overlaps();
    if (hm.length) { hi = mid; hits = hm; } else lo = mid;
  }
  S.simValues = lerpVals(from, to, lo);
  applyMotion();
  S.clash = new Set(hits.flatMap((h) => [h.a, h.b]));
  vp.requestRender();
  const msg = hits.map((h) => `${partLabel(h.a)} hits ${partLabel(h.b)}`).join(' · ');
  const st = $('#sim-status');
  if (st) st.innerHTML = `<span class="bad">⛔ Blocked: ${msg}</span>`;
  toast('⛔ The machine jams here: ' + msg + '. The parts in red would run into each other.', 'error', 6000);
  return false;
}

function syncSliders() {
  document.querySelectorAll('#motion-panel input[data-j]').forEach((inp) => {
    const v = S.simValues?.[inp.dataset.j] ?? 0;
    inp.value = ((v - +inp.min) % (+inp.max - +inp.min + 1e-9) + (+inp.max - +inp.min)) % (+inp.max - +inp.min) + +inp.min;
    const lab = inp.parentElement.querySelector('.val'); if (lab) lab.textContent = round(v, 1) + (inp.dataset.u || '');
  });
}

// ---------------- collisions ----------------
function posedNodes() {
  return S.nodes.filter((n) => !n.hidden).map((n) => {
    const o = S.override?.[n.id];
    const t = engineTree(n);
    if (o) { const d = decompose(mul(o, sc.localMatrix(n))); t.pos = d.pos; t.rot = d.rot; t.scale = d.scale; }
    return t;
  });
}
export { posedNodes };

async function checkCollisions(quiet) {
  const r = await api('/api/interference', { nodes: posedNodes(), q: 0.8 });
  const list = await r.json();
  S.clash = new Set(list.flatMap((c) => [c.a, c.b]));
  vp.requestRender();
  if (!quiet) {
    if (!list.length) toast('No collisions in this position ✓', 'ok');
    else toast(list.map((c) => `${partLabel(c.a)} hits ${partLabel(c.b)} (${round(c.volume, 1)} mm³)`).join(' · '), 'error', 7000);
  }
  return list;
}

async function sweepCheck() {
  const drivers = S.joints.filter((j) => !j.driver);
  if (!drivers.length) { toast('Add a joint first.'); return; }
  stopPlay();
  await Promise.all(S.nodes.map(whenBuilt));
  computeBaseline();
  const steps = 360, found = new Map();
  const saved = { ...(S.simValues || {}) };
  const status = $('#sim-status');
  for (let i = 0; i <= steps; i++) {
    for (const j of drivers) {
      const lo = j.type === 'rotate' && j.max - j.min >= 359 ? 0 : j.min, hi = j.type === 'rotate' && j.max - j.min >= 359 ? 360 : j.max;
      S.simValues[j.id] = lo + ((hi - lo) * i) / steps;
    }
    applyMotion();
    for (const c of overlaps()) {
      const f = found.get(c.key);
      if (!f) found.set(c.key, { ...c, first: i, last: i }); else f.last = i;
    }
    if (i % 24 === 0) { syncSliders(); status.textContent = `Checking… ${Math.round((100 * i) / steps)} %`; await new Promise((r) => setTimeout(r, 0)); }
  }
  S.simValues = saved; applyMotion(); syncSliders();
  const res = [...found.values()];
  status.innerHTML = res.length ? `<span class="bad">⛔ ${res.length} collision${res.length > 1 ? 's' : ''} during the movement</span>` : '✓ Moves freely through the whole movement';
  S.clash = new Set(res.flatMap((c) => [c.a, c.b]));
  vp.requestRender();
  const where = (k) => drivers.map((j) => { const lo = j.type === 'rotate' && j.max - j.min >= 359 ? 0 : j.min, hi = j.type === 'rotate' && j.max - j.min >= 359 ? 360 : j.max; return `${partLabel(j.part)} ${round(lo + ((hi - lo) * k) / steps, 1)}${j.type === 'rotate' ? '°' : ' mm'}`; }).join(', ');
  if (res.length) ui.modal('Collisions during the movement', h('table', {}, h('tr', {}, h('th', {}, 'Parts'), h('th', {}, 'First touch at'), h('th', {}, 'Clear again after')),
    ...res.map((c) => h('tr', {}, h('td', {}, `${partLabel(c.a)} ↔ ${partLabel(c.b)}`), h('td', { class: 'bad' }, where(c.first)), h('td', {}, where(c.last))))), [{ label: 'Close', primary: true }]);
  else toast('No collisions through the whole movement ✓', 'ok');
}

// ---------------- motion panel ----------------
export function renderMotion() {
  const p = $('#motion-panel');
  p.innerHTML = '';
  p.append(h('div', { class: 'mp-head' }, h('b', {}, 'Motion'), h('span', { class: 'faint' }, 'preview only — parts stay separate')));
  if (!S.joints.length) p.append(h('div', { class: 'note', html: 'Make parts move to test your machine:<br>1. Click a part (a gear, a lid, a slider).<br>2. Press <b>Add joint</b> and click its axis face.<br>3. For a second gear, choose <b>Driven by</b> the first — the ratio is worked out from the teeth.' }));
  for (const j of S.joints) {
    const unit = j.type === 'rotate' ? '°' : ' mm';
    const row = h('div', { class: 'joint' });
    const top = h('div', { class: 'jt' }, h('span', { class: 'jicon' }, j.type === 'rotate' ? '⟳' : '⇆'), h('span', { class: 'jn', title: j.name }, j.name),
      h('button', { class: 'ic', title: 'Remove joint', onclick: () => { S.joints = S.joints.filter((x) => x !== j); S.joints.forEach((x) => { if (x.driver === j.id) delete x.driver; if (x.on === j.id) delete x.on; }); C.commit('Remove joint'); renderMotion(); applyMotion(); } }, '✕'));
    row.append(top);
    if (j.driver) {
      const d = S.joints.find((x) => x.id === j.driver);
      row.append(h('div', { class: 'jd' }, `follows “${d?.name}” × `,
        h('input', { class: 'inp tiny', value: round(j.ratio, 4), onkeydown: (e) => e.stopPropagation(), onchange: (e) => { const v = parseFloat(e.target.value); if (isFinite(v)) { j.ratio = v; C.commit('Ratio'); applyMotion(); } } }),
        h('button', { class: 'mini', title: 'Reverse direction', onclick: () => { j.reverse = !j.reverse; C.commit('Reverse'); applyMotion(); } }, j.reverse ? '↺ reversed' : '↻'),
        h('span', { class: 'val' }, '')));
    } else {
      const lo = j.type === 'rotate' && j.max - j.min >= 359 ? -180 : j.min, hi = j.type === 'rotate' && j.max - j.min >= 359 ? 180 : j.max;
      const r = h('input', { type: 'range', min: lo, max: hi, step: j.type === 'rotate' ? 1 : 0.1, value: S.simValues?.[j.id] ?? 0, 'data-j': j.id, 'data-u': unit });
      r.onpointerdown = () => computeBaseline();
      r.oninput = () => {
        S.simValues ||= {};
        const prev = { ...S.simValues };
        moveTo(prev, { ...prev, [j.id]: +r.value });
        syncSliders();
      };
      row.append(h('div', { class: 'jd' }, r, h('span', { class: 'val' }, round(S.simValues?.[j.id] ?? 0, 1) + unit)));
    }
    p.append(row);
  }
  p.append(h('div', { class: 'btnrow' },
    h('button', { class: 'btn small primary', onclick: addJointDialog }, '+ Add joint'),
    h('button', { class: 'btn small', id: 'sim-play', onclick: togglePlay, disabled: !S.joints.length }, '▶ Play'),
    h('select', { title: 'Speed', onchange: (e) => { S.simSpeed = +e.target.value; } }, ...[0.25, 0.5, 1, 2, 4].map((s) => h('option', { value: s, selected: (S.simSpeed || 1) === s }, s + '×'))),
    h('button', { class: 'btn small', onclick: () => { S.simValues = {}; S.clash = null; applyMotion(); syncSliders(); const st = $('#sim-status'); if (st) st.textContent = ''; }, disabled: !S.joints.length }, 'Reset')));
  p.append(h('label', { class: 'mp-collide', title: 'Parts stop when they touch, like the real machine would' },
    h('input', { type: 'checkbox', checked: S.collide !== false, onchange: (e) => { S.collide = e.target.checked; if (!S.collide) { S.clash = null; vp.requestRender(); } } }),
    ' Solid parts — stop when they collide'));
  p.append(h('div', { class: 'btnrow' },
    h('button', { class: 'btn small', onclick: () => checkCollisions(false), title: 'Do any parts overlap in this position?' }, 'Check collisions'),
    h('button', { class: 'btn small', onclick: sweepCheck, disabled: !S.joints.length, title: 'Run through the whole movement and report collisions' }, 'Check whole movement'),
    h('button', { class: 'btn small', onclick: () => setWorkspace('layout') }, 'Lay out for printing →')));
  p.append(h('div', { class: 'note', id: 'sim-status' }, ''));
  syncSliders();
}

// ---------------- print layout ----------------
const printable = (n) => !n.hidden && !sc.isHole(n) && (!n.reference || n.refExport);
function printParts() { return S.nodes.filter(printable); }

function orientMatrix(n) {
  const r = n.printRot;
  return r ? trs([0, 0, 0], r, [1, 1, 1]) : m4();
}

// part box after applying its print orientation (about its own centre)
function orientedBox(n) {
  const e = getGeom(n);
  if (!e || !e.tris) return null;
  const W = sc.localMatrix(n), R = orientMatrix(n);
  const b0 = sc.worldBox(n.id), c = v3.mul(v3.add(b0[0], b0[1]), 0.5);
  const M = mul(translate(...c), mul(R, mul(translate(...v3.mul(c, -1)), W)));
  // exact bounds from vertices (rotations make boxes loose otherwise)
  let lo = [Infinity, Infinity, Infinity], hi = [-Infinity, -Infinity, -Infinity];
  const P = e.positions, stepN = Math.max(3, Math.floor(P.length / 3 / 20000) * 3);
  for (let i = 0; i < P.length; i += stepN) { const p = apply(M, [P[i], P[i + 1], P[i + 2]]); lo = v3.min(lo, p); hi = v3.max(hi, p); }
  return { M, lo, hi };
}

export function computeLayout() {
  if (S.workspace !== 'layout') return;
  const gap = S.layoutGap ?? 5, bed = S.settings.bed;
  S.override = {}; // measure parts at their design placement
  const items = printParts().map((n) => ({ n, b: orientedBox(n) })).filter((x) => x.b);
  items.sort((a, b) => (b.b.hi[1] - b.b.lo[1]) - (a.b.hi[1] - a.b.lo[1]));
  // shelf packing across the bed width
  let x = 0, y = 0, rowH = 0, plate = 0, maxW = 0;
  const placed = [];
  for (const it of items) {
    const w = it.b.hi[0] - it.b.lo[0], d = it.b.hi[1] - it.b.lo[1];
    if (x > 0 && x + w > bed[0]) { x = 0; y += rowH + gap; rowH = 0; }
    if (y > 0 && y + d > bed[1]) { plate++; x = 0; y = 0; rowH = 0; }
    placed.push({ ...it, x, y, plate });
    x += w + gap; rowH = Math.max(rowH, d); maxW = Math.max(maxW, x);
  }
  S.override = {};
  // centre what is on plate 1; further plates go to the right of the bed
  const usedW = Math.min(bed[0], Math.max(...placed.filter((p) => !p.plate).map((p) => p.x + p.b.hi[0] - p.b.lo[0]), 0));
  const usedD = Math.max(...placed.filter((p) => !p.plate).map((p) => p.y + p.b.hi[1] - p.b.lo[1]), 0);
  for (const p of placed) {
    const ox = -usedW / 2 + p.plate * (bed[0] + 30), oy = -usedD / 2;
    const t = translate(ox + p.x - p.b.lo[0], oy + p.y - p.b.lo[1], -p.b.lo[2]);
    S.override[p.n.id] = mul(mul(t, p.b.M), invert(sc.localMatrix(p.n)));
  }
  // holes and hidden parts are not printed
  for (const n of S.nodes) if (!S.override[n.id]) S.override[n.id] = translate(0, 0, -100000);
  S.plates = plate + 1;
  const st = $('#layout-status');
  if (st) st.innerHTML = `${placed.length} part${placed.length === 1 ? '' : 's'} on ${S.plates} plate${S.plates > 1 ? 's' : ''}` + (S.plates > 1 ? ' — <span class="warn">does not fit on one bed</span>' : ' ✓');
  vp.requestRender();
}

// put the biggest flat face of each part down
export function autoOrient(ids) {
  S.override = {};
  for (const n of ids ? ids.map(sc.node) : printParts()) {
    const e = getGeom(n);
    if (!e || !e.tris) continue;
    const W = sc.localMatrix(n), P = e.positions;
    const dirs = [[0, 0, -1], [0, 0, 1], [1, 0, 0], [-1, 0, 0], [0, 1, 0], [0, -1, 0]];
    const b = sc.worldBox(n.id);
    let best = 0, bestA = -1;
    dirs.forEach((d, k) => {
      const ext = Math.max(...[0, 1, 2].map((i) => (d[i] > 0 ? b[1][i] : d[i] < 0 ? -b[0][i] : -Infinity)));
      let A = 0;
      for (let i = 0; i < P.length; i += 9) {
        const a = apply(W, [P[i], P[i + 1], P[i + 2]]), bb = apply(W, [P[i + 3], P[i + 4], P[i + 5]]), c = apply(W, [P[i + 6], P[i + 7], P[i + 8]]);
        const cr = v3.cross(v3.sub(bb, a), v3.sub(c, a)), l = v3.len(cr);
        if (!l || v3.dot(cr, d) / l < 0.985) continue;
        if (Math.abs(v3.dot(a, d) - ext) > 0.3) continue; // must be on the outside
        A += l / 2;
      }
      if (A > bestA * 1.05) { bestA = A; best = k; }
    });
    const R = rotBetween(dirs[best], [0, 0, -1]);
    n.printRot = decompose(R).rot;
  }
  C.commit('Auto-orient'); computeLayout();
}

function layFlatPrint() {
  vp.startFacePick('<b>Print orientation:</b> click the face that should sit on the bed.', (hit) => {
    const n = sc.node(hit.id);
    // the clicked normal is already in the laid-out pose: turn it to face down
    const R = mul(rotBetween(hit.normal, [0, 0, -1]), orientMatrix(n));
    n.printRot = decompose(R).rot;
    C.commit('Print orientation'); computeLayout();
  });
}

export function renderLayout() {
  const p = $('#layout-panel');
  p.innerHTML = '';
  p.append(h('div', { class: 'mp-head' }, h('b', {}, 'Print layout'), h('span', { class: 'faint' }, 'your assembly is untouched')));
  p.append(h('div', { class: 'note', id: 'layout-status' }, ''));
  p.append(h('div', { class: 'btnrow' },
    h('button', { class: 'btn small primary', onclick: () => autoOrient(), title: 'Turn every part so its largest flat face is on the bed' }, '✨ Auto-orient all'),
    h('button', { class: 'btn small', onclick: layFlatPrint }, 'Lay a part flat…'),
    h('button', { class: 'btn small', onclick: () => { S.nodes.forEach((n) => delete n.printRot); C.commit('Reset orientation'); computeLayout(); } }, 'Reset orientation')));
  p.append(h('div', { class: 'field' }, h('label', {}, 'Space between parts'), h('select', { onchange: (e) => { S.layoutGap = +e.target.value; computeLayout(); } }, ...[2, 5, 10, 15].map((g) => h('option', { value: g, selected: (S.layoutGap ?? 5) === g }, g + ' mm')))));
  p.append(h('div', { class: 'btnrow' },
    h('button', { class: 'btn small primary', onclick: () => C.exportLayout('3mf') }, 'Export plate (3MF)'),
    h('button', { class: 'btn small', onclick: () => C.exportLayout('stl') }, 'STL'),
    h('button', { class: 'btn small', onclick: () => C.exportLayout('stlzip') }, 'One STL per part (zip)')));
}

export function layoutNodes() { return posedNodes().filter((t) => { const n = sc.node(t.id); return n && printable(n); }); }
