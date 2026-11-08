// Scene tree helpers.
import { S, PALETTE } from './state.js';
import { trs, mul, invert, decompose, xformBox, v3 } from './math.js';
import { localBox } from './api.js';

let idc = Date.now() % 100000;
export const newId = () => 'n' + (idc++).toString(36) + Math.random().toString(36).slice(2, 5);

export function walk(nodes, fn, parent = null) {
  for (const n of nodes) {
    if (fn(n, parent) === false) return false;
    if (n.children && walk(n.children, fn, n) === false) return false;
  }
}

export function find(id) {
  let r = null;
  walk(S.nodes, (n, p) => { if (n.id === id) { r = { node: n, parent: p }; return false; } });
  return r;
}
export const node = (id) => find(id)?.node;
export const parentOf = (id) => find(id)?.parent || null;
export const siblings = (id) => { const p = parentOf(id); return p ? p.children : S.nodes; };

export function topOf(id) {
  let r = find(id);
  while (r && r.parent) r = find(r.parent.id);
  return r?.node;
}

export const localMatrix = (n) => trs(n.pos, n.rot, n.scale);

export function worldMatrix(id) {
  const chain = [];
  let r = find(id);
  while (r) { chain.unshift(r.node); r = r.parent ? find(r.parent.id) : null; }
  let m = trs([0, 0, 0], [0, 0, 0], [1, 1, 1]);
  for (const n of chain) m = mul(m, localMatrix(n));
  return m;
}

export function parentWorld(id) {
  const p = parentOf(id);
  return p ? worldMatrix(p.id) : trs([0, 0, 0], [0, 0, 0], [1, 1, 1]);
}

// set node transform so that its world matrix becomes W
export function setWorld(n, W, parentW) {
  const L = mul(invert(parentW), W);
  const d = decompose(L);
  n.pos = d.pos; n.rot = d.rot; n.scale = d.scale;
}

export function worldBox(id) {
  const n = node(id);
  const b = n && localBox(n);
  if (!b) return null;
  // in Motion / Print layout, top-level parts are shown at a temporary placement
  const o = S.override && !parentOf(id) && S.override[id];
  return xformBox(o ? mul(o, worldMatrix(id)) : worldMatrix(id), b[0], b[1]);
}

export function selectionBox(ids = [...S.sel]) {
  let lo = [Infinity, Infinity, Infinity], hi = [-Infinity, -Infinity, -Infinity], ok = false;
  for (const id of ids) {
    const b = worldBox(id);
    if (!b) continue;
    ok = true; lo = v3.min(lo, b[0]); hi = v3.max(hi, b[1]);
  }
  return ok ? [lo, hi] : null;
}

export function isHole(n) {
  if (n.type !== 'group') return !!n.hole;
  if (n.hole) return true;
  const kids = (n.children || []).filter((c) => !c.hidden);
  return kids.length > 0 && kids.every((c) => isHole(c));
}

let colorIdx = 0;
export function makeNode(type, params = {}, extra = {}) {
  const def = S.byType[type];
  const p = {};
  if (def) for (const pr of def.params) p[pr.key] = structuredClone(pr.def);
  Object.assign(p, params);
  let color = def?.color || PALETTE[colorIdx++ % PALETTE.length];
  if (def && !def.fixed && !def.hole && def.category === 'Basic') color = PALETTE[colorIdx++ % 12];
  return {
    id: newId(), name: def ? def.name : type, type, params: p,
    pos: [0, 0, 0], rot: [0, 0, 0], scale: [1, 1, 1],
    hole: !!def?.hole, color, ...extra,
  };
}

export function cloneNode(n) {
  const c = structuredClone(n);
  walk([c], (x) => { x.id = newId(); });
  return c;
}

// strip editor-only fields before sending to the engine
export function engineNode(n) {
  const o = { id: n.id, type: n.type, pos: n.pos, rot: n.rot, scale: n.scale, hole: !!n.hole };
  if (n.params) o.params = n.params;
  if (n.hidden) o.hidden = true;
  if (n.mode) o.mode = n.mode;
  if (n.meshRef) o.meshRef = n.meshRef;
  if (n.name) o.name = n.name;
  if (n.color) o.color = n.color;
  if (n.children) o.children = n.children.map(engineNode);
  return o;
}

export function removeNode(id) {
  const arr = siblings(id);
  const i = arr.findIndex((n) => n.id === id);
  if (i >= 0) arr.splice(i, 1);
}

// move a node into a new parent (null = top level), keeping its world placement
export function reparent(id, newParentId, index = -1) {
  const n = node(id);
  if (!n) return;
  if (newParentId && (newParentId === id || isDescendant(newParentId, id))) return;
  const W = worldMatrix(id);
  removeNode(id);
  const arr = newParentId ? node(newParentId).children : S.nodes;
  const PW = newParentId ? worldMatrix(newParentId) : trs([0, 0, 0], [0, 0, 0], [1, 1, 1]);
  setWorld(n, W, PW);
  if (index < 0 || index > arr.length) arr.push(n); else arr.splice(index, 0, n);
}

export function isDescendant(id, ancestorId) {
  const a = node(ancestorId);
  if (!a) return false;
  let found = false;
  walk(a.children || [], (n) => { if (n.id === id) { found = true; return false; } });
  return found;
}

// Group nodes (must share a parent). Returns the new group.
export function groupNodes(ids) {
  ids = ids.filter((id) => node(id));
  if (ids.length < 1) return null;
  const parent = parentOf(ids[0]);
  const arr = parent ? parent.children : S.nodes;
  // all must be siblings; otherwise bring them to the first one's parent
  for (const id of ids.slice(1)) if (parentOf(id) !== parent) reparent(id, parent ? parent.id : null);
  const box = selectionBox(ids) || [[0, 0, 0], [0, 0, 0]];
  const center = [(box[0][0] + box[1][0]) / 2, (box[0][1] + box[1][1]) / 2, box[0][2]];
  const PW = parent ? worldMatrix(parent.id) : trs([0, 0, 0], [0, 0, 0], [1, 1, 1]);
  const pInv = invert(PW);
  const lc = [pInv[0] * center[0] + pInv[4] * center[1] + pInv[8] * center[2] + pInv[12], pInv[1] * center[0] + pInv[5] * center[1] + pInv[9] * center[2] + pInv[13], pInv[2] * center[0] + pInv[6] * center[1] + pInv[10] * center[2] + pInv[14]];
  const first = arr.findIndex((n) => ids.includes(n.id));
  const solids = ids.map(node).filter((n) => !isHole(n));
  const g = {
    id: newId(), name: 'Group', type: 'group', pos: lc, rot: [0, 0, 0], scale: [1, 1, 1], hole: false,
    color: (solids[0] || node(ids[0])).color, children: [], expanded: false,
  };
  const GW = mul(PW, localMatrix(g));
  for (const id of ids) {
    const n = node(id);
    const W = worldMatrix(id);
    removeNode(id);
    setWorld(n, W, GW);
    g.children.push(n);
  }
  arr.splice(Math.max(0, Math.min(first, arr.length)), 0, g);
  return g;
}

export function ungroup(id) {
  const g = node(id);
  if (!g || g.type !== 'group') return [];
  const parent = parentOf(id);
  const arr = parent ? parent.children : S.nodes;
  const idx = arr.indexOf(g);
  const PW = parent ? worldMatrix(parent.id) : trs([0, 0, 0], [0, 0, 0], [1, 1, 1]);
  const kids = [];
  for (const c of [...g.children]) {
    const W = worldMatrix(c.id);
    setWorld(c, W, PW);
    // ungrouped holes keep acting as holes; colour of solids is kept
    kids.push(c);
  }
  arr.splice(idx, 1, ...kids);
  return kids.map((k) => k.id);
}

export function allIds() { const a = []; walk(S.nodes, (n) => a.push(n.id)); return a; }
