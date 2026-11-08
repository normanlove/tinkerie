// Talks to the Go geometry engine and caches built meshes.
import { S, emit, toast } from './state.js';

const cache = new Map();     // key -> entry
const inflight = new Map();  // key -> promise
const lastGood = new Map();  // node id -> entry (shown while a rebuild runs)
let pending = 0;
let freeGpu = () => {};
export const setGpuFree = (f) => { freeGpu = f; };

export function stripForKey(n, root = true) {
  const o = { t: n.type };
  if (n.params) o.p = n.params;
  if (n.mode) o.m = n.mode;
  if (n.meshRef) o.r = n.meshRef;
  o.h = !!n.hole;
  if (!root) { o.x = [n.pos, n.rot, n.scale]; if (n.hidden) o.v = 1; }
  if (n.children) o.c = n.children.map((c) => stripForKey(c, false));
  return o;
}
export const geomKey = (n) => JSON.stringify(stripForKey(n)) + '|' + S.settings.quality;

function engineTree(n) {
  const o = { id: n.id, type: n.type, pos: n.pos, rot: n.rot, scale: n.scale, hole: !!n.hole };
  if (n.params) o.params = n.params;
  if (n.hidden) o.hidden = true;
  if (n.mode) o.mode = n.mode;
  if (n.meshRef) o.meshRef = n.meshRef;
  if (n.name) o.name = n.name;
  if (n.color) o.color = n.color;
  if (n.slot) o.slot = n.slot;
  if (n.reference) { o.reference = true; if (n.refExport) o.refExport = true; }
  if (n.children) o.children = n.children.map(engineTree);
  return o;
}
export { engineTree };

function decode(buf) {
  const dv = new DataView(buf);
  const tris = dv.getUint32(0, true), ev = dv.getUint32(4, true);
  const positions = new Float32Array(buf, 8, tris * 9);
  const normals = new Float32Array(buf, 8 + tris * 36, tris * 9);
  const edges = new Float32Array(buf, 8 + tris * 72, ev * 3);
  let lo = [Infinity, Infinity, Infinity], hi = [-Infinity, -Infinity, -Infinity];
  for (let i = 0; i < positions.length; i += 3) {
    const x = positions[i], y = positions[i + 1], z = positions[i + 2];
    if (x < lo[0]) lo[0] = x; if (y < lo[1]) lo[1] = y; if (z < lo[2]) lo[2] = z;
    if (x > hi[0]) hi[0] = x; if (y > hi[1]) hi[1] = y; if (z > hi[2]) hi[2] = z;
  }
  if (!tris) { lo = [0, 0, 0]; hi = [0, 0, 0]; }
  return { positions, normals, edges, lo, hi, tris };
}

function busy(d) {
  pending += d;
  const el = document.getElementById('engine');
  if (el) { el.classList.toggle('busy', pending > 0); el.title = pending > 0 ? 'Building geometry…' : 'Engine ready'; }
}

async function reupload(ref) {
  const data = S.meshData[ref];
  if (!data) throw new Error('An imported model is missing. Please import it again.');
  const r = await fetch('/api/raw', { method: 'POST', body: data });
  if (!r.ok) throw new Error('Could not restore imported model');
}

async function build(n, key) {
  busy(1);
  try {
    for (let attempt = 0; attempt < 3; attempt++) {
      const r = await fetch('/api/build', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ node: engineTree(n), q: S.settings.quality }) });
      if (r.status === 410) { await reupload(r.headers.get('X-Missing-Mesh')); continue; }
      if (!r.ok) throw new Error(await r.text());
      const e = decode(await r.arrayBuffer());
      e.key = key;
      const info = r.headers.get('X-Info');
      if (info) e.info = decodeURIComponent(info.replace(/\+/g, ' '));
      const ports = r.headers.get('X-Ports');
      if (ports) { try { e.ports = JSON.parse(ports); } catch {} }
      return e;
    }
    throw new Error('engine did not respond');
  } finally { busy(-1); }
}

function remember(key, e) {
  cache.set(key, e);
  if (cache.size > 260) {
    for (const [k, v] of cache) {
      if ([...lastGood.values()].includes(v)) continue;
      freeGpu(v); cache.delete(k);
      if (cache.size <= 220) break;
    }
  }
}

const running = new Map(); // node id -> key being built
const queued = new Map();  // node id -> latest {n, key} waiting

function start(n, key) {
  running.set(n.id, key);
  const pr = build(n, key).then((e) => {
    remember(key, e);
    return e;
  }).catch((err) => {
    const bad = { error: String(err.message || err), key, positions: new Float32Array(0), normals: new Float32Array(0), edges: new Float32Array(0), lo: [0, 0, 0], hi: [0, 0, 0], tris: 0 };
    remember(key, bad);
    toast('Could not build "' + (n.name || n.type) + '": ' + bad.error.slice(0, 140), 'error', 6000);
  }).finally(() => {
    inflight.delete(key);
    if (running.get(n.id) === key) running.delete(n.id);
    const q = queued.get(n.id);
    if (q) { queued.delete(n.id); if (!cache.has(q.key) && !inflight.has(q.key)) start(q.n, q.key); }
    emit('geom', n.id);
  });
  inflight.set(key, pr);
}

// Returns the mesh entry for a node (possibly a stale one while rebuilding).
export function getGeom(n) {
  const key = geomKey(n);
  const e = cache.get(key);
  if (e) { cache.delete(key); cache.set(key, e); lastGood.set(n.id, e); return e; }
  if (!inflight.has(key)) {
    if (running.has(n.id)) queued.set(n.id, { n: structuredClone(n), key });
    else start(n, key);
  }
  return lastGood.get(n.id) || null;
}

export function whenBuilt(n) {
  const key = geomKey(n);
  return new Promise((resolve) => {
    const check = () => { if (cache.has(key)) resolve(cache.get(key)); else setTimeout(check, 25); };
    getGeom(n); check();
  });
}

export function localBox(n) {
  const e = getGeom(n);
  if (!e || !e.tris) return e ? [[-1, -1, 0], [1, 1, 1]] : null;
  return [e.lo, e.hi];
}

export const isPending = () => pending > 0;

export async function api(path, body, raw) {
  const r = await fetch(path, { method: 'POST', headers: raw ? {} : { 'Content-Type': 'application/json' }, body: raw ? body : JSON.stringify(body) });
  if (r.status === 410) { await reupload(r.headers.get('X-Missing-Mesh')); return api(path, body, raw); }
  if (!r.ok) throw new Error(await r.text());
  return r;
}

export function dropCacheForQuality() { for (const v of cache.values()) freeGpu(v); cache.clear(); lastGood.clear(); }
