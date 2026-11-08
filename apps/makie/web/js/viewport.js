// 3D viewport: camera, drawing, picking, gizmo and mouse interaction.
import { S, emit, hexToRgb, hint } from './state.js';
import { Renderer, geoCylinder, geoTorus, geoBox, mergeGeo, orientGeo } from './renderer.js';
import { v3, m4, mul, trs, invert, apply, applyDir, project, perspective, ortho, lookAt, rayTri, rayBox, translate, rotX, rotY, rotZ, alignZ, eulerFromMat, snap, round, xformBox, DEG } from './math.js';
import { getGeom, setGpuFree } from './api.js';
import * as sc from './scene.js';
import { evalNum } from './ui.js';
import * as wy from './wirie.js';

let R, canvas, labels, cb;
const cam = { target: [0, 0, 10], yaw: -60, pitch: 28, dist: 260, ortho: false, fov: 40 };
let view = m4(), proj = m4(), vp = m4(), eye = [0, 0, 0];
let needs = true;
let hoverId = null, hoverHandle = null;
let gz = null;                       // gizmo geometry
let drag = null;                     // active interaction
let placing = null;                  // { node, hitInfo }
let measure = { pts: [], lines: [], cursor: null };
let gridGpu = null, bedGpu = null, bedBoxGpu = null, gridKey = '';
let tween = null;

export const camera = cam;
export const requestRender = () => { needs = true; };

export function initViewport(el, callbacks) {
  canvas = el; cb = callbacks;
  R = new Renderer(canvas);
  setGpuFree((e) => { R.free(e.gpu); R.free(e.egpu); e.gpu = e.egpu = null; });
  labels = document.getElementById('labels');
  buildGizmo();
  const ro = new ResizeObserver(() => { needs = true; });
  ro.observe(canvas);
  canvas.addEventListener('pointerdown', onDown);
  window.addEventListener('pointermove', onMove);
  window.addEventListener('pointerup', onUp);
  canvas.addEventListener('wheel', onWheel, { passive: false });
  canvas.addEventListener('dblclick', onDbl);
  canvas.addEventListener('contextmenu', (e) => e.preventDefault());
  canvas.addEventListener('dragover', onDragOver);
  canvas.addEventListener('dragleave', () => { if (placing?.fromDrag) { placing.hit = null; placing.visible = false; needs = true; } });
  canvas.addEventListener('drop', onDrop);
  labels.addEventListener('pointerdown', onLabelDown);
  initViewCube();
  loop();
}

function loop() {
  requestAnimationFrame(loop);
  if (tween) stepTween();
  if (needs) { needs = false; render(); }
}

// ---------------- camera ----------------
// keep the camera sane: never NaN, never miles away from the build plate
function camLimits() {
  const b = S.settings.bed || [220, 220, 250];
  const span = Math.max(b[0], b[1], b[2], 100);
  return { span, maxDist: span * 8, maxOff: span * 3 };
}
function sanitizeCamera() {
  const L = camLimits();
  const bad = (x) => !Number.isFinite(x);
  if (cam.target.some(bad) || bad(cam.dist) || bad(cam.yaw) || bad(cam.pitch)) {
    cam.target = [0, 0, 10]; cam.dist = L.span * 1.4; cam.yaw = -60; cam.pitch = 28; tween = null;
  }
  cam.dist = Math.max(2, Math.min(L.maxDist, cam.dist));
  cam.target = cam.target.map((x, i) => Math.max(-L.maxOff, Math.min(L.maxOff, x)));
}

function computeCamera() {
  R.resize();
  sanitizeCamera();
  const w = canvas.width, h = canvas.height, asp = w / Math.max(h, 1);
  const p = cam.pitch * DEG, y = cam.yaw * DEG;
  const dir = [Math.cos(p) * Math.cos(y), Math.cos(p) * Math.sin(y), Math.sin(p)];
  eye = v3.add(cam.target, v3.mul(dir, cam.dist));
  const up = Math.abs(cam.pitch) > 89.5 ? [-Math.cos(y) * Math.sign(cam.pitch), -Math.sin(y) * Math.sign(cam.pitch), 0] : [0, 0, 1];
  view = lookAt(eye, cam.target, up);
  const far = cam.dist * 20 + 5000, near = Math.max(0.05, cam.dist / 400);
  if (cam.ortho) {
    const hh = cam.dist * Math.tan((cam.fov * DEG) / 2);
    proj = ortho(-hh * asp, hh * asp, -hh, hh, -far, far);
  } else proj = perspective(cam.fov * DEG, asp, near, far);
  vp = mul(proj, view);
  R.setCamera(view, proj, eye);
}

export function setCam(target, yaw, pitch, dist, animate = true) {
  const to = { target: target || cam.target, yaw: yaw ?? cam.yaw, pitch: pitch ?? cam.pitch, dist: dist ?? cam.dist };
  // shortest yaw path
  let dy = to.yaw - cam.yaw; dy = ((dy + 540) % 360) - 180; to.yaw = cam.yaw + dy;
  if (!animate) { Object.assign(cam, to); needs = true; return; }
  tween = { from: { target: [...cam.target], yaw: cam.yaw, pitch: cam.pitch, dist: cam.dist }, to, t0: performance.now() };
}
function stepTween() {
  const t = Math.min(1, (performance.now() - tween.t0) / 280), e = t < 0.5 ? 2 * t * t : 1 - Math.pow(-2 * t + 2, 2) / 2;
  const f = tween.from, to = tween.to;
  cam.target = v3.lerp(f.target, to.target, e); cam.yaw = f.yaw + (to.yaw - f.yaw) * e;
  cam.pitch = f.pitch + (to.pitch - f.pitch) * e; cam.dist = f.dist + (to.dist - f.dist) * e;
  needs = true;
  if (t >= 1) tween = null;
}

export function setView(name) {
  const views = { front: [-90, 0], back: [90, 0], right: [0, 0], left: [180, 0], top: [-90, 89.9], bottom: [-90, -89.9], iso: [-60, 28] };
  const v = views[name] || views.iso;
  setCam(null, v[0], v[1]);
}

export function fitView(ids) {
  const okBox = (b) => b && b.flat().every(Number.isFinite) && b.flat().every((x) => Math.abs(x) < camLimits().maxOff);
  let box = ids && ids.length ? sc.selectionBox(ids) : null;
  if (!okBox(box)) box = sc.selectionBox(S.nodes.filter((n) => !n.hidden).map((n) => n.id));
  if (!okBox(box)) box = null;
  if (!box) box = [[-S.settings.bed[0] / 2, -S.settings.bed[1] / 2, 0], [S.settings.bed[0] / 2, S.settings.bed[1] / 2, 20]];
  const c = v3.mul(v3.add(box[0], box[1]), 0.5);
  const r = Math.max(5, v3.dist(box[0], box[1]) / 2);
  const d = r / Math.sin((cam.fov * DEG) / 2) * 1.1;
  setCam(c, null, null, d);
}

export function toggleOrtho() { cam.ortho = !cam.ortho; S.settings.ortho = cam.ortho; needs = true; return cam.ortho; }

// ---------------- rays ----------------
function rayAt(cx, cy) {
  const r = canvas.getBoundingClientRect();
  const x = ((cx - r.left) / r.width) * 2 - 1, y = -(((cy - r.top) / r.height) * 2 - 1);
  const inv = invert(vp);
  const a = project(inv, [x, y, -1]), b = project(inv, [x, y, 1]);
  const o = [a[0], a[1], a[2]];
  return { o, d: v3.norm(v3.sub([b[0], b[1], b[2]], o)), sx: cx - r.left, sy: cy - r.top };
}

export function toScreen(p) {
  const q = project(vp, p);
  const r = canvas.getBoundingClientRect();
  return [((q[0] + 1) / 2) * r.width, ((1 - q[1]) / 2) * r.height, q[2], q[3]];
}

function hitMesh(e, M, ray) {
  if (!e || !e.tris) return null;
  const inv = invert(M);
  const o = apply(inv, ray.o), d = applyDir(inv, ray.d);
  if (!rayBox(o, d, e.lo, e.hi)) return null;
  const P = e.positions;
  let best = Infinity, bi = -1;
  for (let i = 0; i < P.length; i += 9) {
    const t = rayTri(o, d, [P[i], P[i + 1], P[i + 2]], [P[i + 3], P[i + 4], P[i + 5]], [P[i + 6], P[i + 7], P[i + 8]]);
    if (t > 0 && t < best) { best = t; bi = i; }
  }
  if (bi < 0) return null;
  const a = [P[bi], P[bi + 1], P[bi + 2]], b = [P[bi + 3], P[bi + 4], P[bi + 5]], c = [P[bi + 6], P[bi + 7], P[bi + 8]];
  let n = v3.cross(v3.sub(b, a), v3.sub(c, a));
  // to world via inverse transpose
  const nw = [inv[0] * n[0] + inv[1] * n[1] + inv[2] * n[2], inv[4] * n[0] + inv[5] * n[1] + inv[6] * n[2], inv[8] * n[0] + inv[9] * n[1] + inv[10] * n[2]];
  let nn = v3.norm(nw);
  if (v3.dot(nn, ray.d) > 0) nn = v3.mul(nn, -1);
  return { t: best, point: v3.add(ray.o, v3.mul(ray.d, best)), normal: nn, tri: bi / 9, e, M };
}

// pick top-level objects (or children inside groups when deep=true)
// matrix used to show a top-level part: its design placement, or a temporary
// simulation / print-layout placement
export function topMatrix(n) {
  const o = S.override && S.override[n.id];
  return o ? mul(o, sc.localMatrix(n)) : sc.localMatrix(n);
}

export function pick(cx, cy, opts = {}) {
  const ray = cx.o ? cx : rayAt(cx, cy);
  let best = null;
  const test = (n, M, top) => {
    if (n.hidden || (opts.exclude && opts.exclude.has(top.id)) || (!opts.locked && top.locked && !top.reference)) return;
    if (opts.deep && n.type === 'group') {
      for (const c of n.children) test(c, mul(M, sc.localMatrix(c)), top);
      return;
    }
    if (opts.solidOnly && sc.isHole(n)) return;
    const h = hitMesh(getGeom(n), M, ray);
    if (h && (!best || h.t < best.t)) best = { ...h, id: top.id, nodeId: n.id };
  };
  for (const n of S.nodes) test(n, topMatrix(n), n);
  return best;
}

function groundHit(ray, z = 0) {
  if (Math.abs(ray.d[2]) < 1e-9) return null;
  const t = (z - ray.o[2]) / ray.d[2];
  if (t < 0) return null;
  return v3.add(ray.o, v3.mul(ray.d, t));
}

function planeHit(ray, p0, n) {
  const den = v3.dot(ray.d, n);
  if (Math.abs(den) < 1e-9) return null;
  const t = v3.dot(v3.sub(p0, ray.o), n) / den;
  return v3.add(ray.o, v3.mul(ray.d, t));
}

// parameter along axis line closest to the ray
function axisParam(ray, p0, a) {
  const w = v3.sub(p0, ray.o);
  const b = v3.dot(a, ray.d), d = v3.dot(a, w), e = v3.dot(ray.d, w);
  const den = 1 - b * b;
  if (Math.abs(den) < 1e-6) return null;
  return (b * e - d) / den;
}

// ---------------- drawing ----------------
function ensureGpu(e) {
  if (!e.gpu && e.tris) { e.gpu = R.createMesh(e.positions, e.normals); e.egpu = R.createLines(e.edges); }
}

function ensureGrid() {
  const st = S.settings, key = st.bed.join(',') + st.theme;
  if (key === gridKey) return;
  gridKey = key;
  [gridGpu, bedGpu, bedBoxGpu].forEach((g) => g && R.free(g));
  const [bx, by, bz] = st.bed, L = [];
  const hx = bx / 2, hy = by / 2;
  const step = 10;
  for (let x = -Math.floor(hx / step) * step; x <= hx + 1e-6; x += step) L.push(x, -hy, 0, x, hy, 0);
  for (let y = -Math.floor(hy / step) * step; y <= hy + 1e-6; y += step) L.push(-hx, y, 0, hx, y, 0);
  gridGpu = R.createLines(L);
  const g = geoBox([-hx, -hy, -0.6], [hx, hy, -0.02]);
  bedGpu = R.createMesh(g.positions, g.normals);
  const B = [];
  const c = [[-hx, -hy], [hx, -hy], [hx, hy], [-hx, hy]];
  for (let i = 0; i < 4; i++) { const a = c[i], b = c[(i + 1) % 4]; B.push(a[0], a[1], 0, b[0], b[1], 0, a[0], a[1], bz, b[0], b[1], bz, a[0], a[1], 0, a[0], a[1], bz); }
  bedBoxGpu = R.createLines(B);
}

const axisLines = { gpu: null };

function render() {
  computeCamera();
  ensureGrid();
  const dark = S.settings.theme === 'dark';
  R.clear(dark ? [0.105, 0.115, 0.14, 1] : [0.93, 0.945, 0.96, 1]);
  R.overhang = S.view.overhang;
  R.clip = null;
  if (S.view.section) {
    const ax = S.view.sectionAxis, n = [0, 0, 0]; n[ax] = 1;
    R.clip = [...n, S.view.sectionPos];
  }
  const I = m4();
  // bed
  // the plate itself is drawn after the parts, see-through, so you can look up at the underside
  const below = eye[2] < 0;
  R.drawLines(gridGpu, I, dark ? [0.32, 0.35, 0.42, below ? 0.45 : 1] : [0.7, 0.74, 0.8, below ? 0.5 : 1], { noClip: true, depthWrite: !below });
  if (!axisLines.gpu) axisLines.gpu = R.createLines([0, 0, 0.01, 30, 0, 0.01, 0, 0, 0.01, 0, 30, 0.01]);
  R.drawLines(axisLines.gpu, I, [0.9, 0.3, 0.3, 1], { noClip: true });
  if (S.settings.showBed) R.drawLines(bedBoxGpu, I, dark ? [0.4, 0.45, 0.55, 0.35] : [0.4, 0.45, 0.55, 0.35], { noClip: true });

  const holes = [];
  const edgesOn = S.settings.edges !== false;
  // opaque solids
  for (const n of S.nodes) {
    if (n.hidden) continue;
    const e = getGeom(n);
    if (!e || !e.tris) continue;
    ensureGpu(e);
    const M = topMatrix(n);
    const sel = S.sel.has(n.id);
    if (sc.isHole(n)) { holes.push({ n, e, M, sel }); continue; }
    if (n.type === 'wirieboard') { wy.drawBoard(R, n, e, M, { sel, hover: hoverId === n.id, edges: edgesOn, dark, wire: S.view.wire, redraw: () => { needs = true; } }); continue; }
    let col = hexToRgb(n.color);
    if (S.clash && S.clash.has(n.id)) col = [0.95, 0.15, 0.2];
    R.drawMesh(e.gpu, M, { color: col, highlight: sel ? 0.9 : hoverId === n.id ? 0.45 : 0, offset: true, opacity: S.view.wire ? 0.25 : 1, depthWrite: !S.view.wire });
    if (edgesOn || sel) R.drawLines(e.egpu, M, sel ? [0.3, 0.65, 1, 1] : dark ? [0, 0, 0, 0.35] : [0, 0, 0, 0.22]);
  }
  // placing preview (solid)
  let prevHole = null;
  if (placing && placing.visible) {
    const e = getGeom(placing.node);
    if (e && e.tris) {
      ensureGpu(e);
      const M = sc.localMatrix(placing.node);
      if (placing.node.hole) prevHole = { e, M };
      else { R.drawMesh(e.gpu, M, { color: hexToRgb(placing.node.color), opacity: 0.75, depthWrite: false, highlight: 0.6 }); R.drawLines(e.egpu, M, [0.3, 0.65, 1, 0.8]); }
    }
  }
  // build plate: mostly solid from above, nearly clear from underneath
  R.drawMesh(bedGpu, I, { color: dark ? [0.2, 0.22, 0.27] : [0.82, 0.85, 0.9], unlit: true, noOverhang: true, noClip: true, opacity: below ? 0.12 : 0.6, depthWrite: false });
  // holes (transparent)
  holes.sort((a, b) => v3.dist(eye, [b.M[12], b.M[13], b.M[14]]) - v3.dist(eye, [a.M[12], a.M[13], a.M[14]]));
  for (const h of holes) {
    R.drawMesh(h.e.gpu, h.M, { color: h.sel ? [0.55, 0.7, 0.95] : [0.62, 0.66, 0.74], opacity: h.sel ? 0.55 : 0.4, hole: true, depthWrite: false, highlight: h.sel ? 0.5 : hoverId === h.n.id ? 0.3 : 0, noOverhang: true });
    R.drawLines(h.e.egpu, h.M, h.sel ? [0.3, 0.65, 1, 0.95] : [0.75, 0.8, 0.9, 0.5]);
  }
  if (prevHole) R.drawMesh(prevHole.e.gpu, prevHole.M, { color: [0.6, 0.7, 0.9], opacity: 0.5, hole: true, depthWrite: false, noOverhang: true });
  if (prevHole) R.drawLines(prevHole.e.egpu, prevHole.M, [0.3, 0.65, 1, 0.9]);

  // selected children inside groups: ghost
  for (const id of S.sel) {
    const p = sc.parentOf(id);
    if (!p || S.workspace === 'layout' || S.workspace === 'motion') continue;
    const n = sc.node(id);
    const e = getGeom(n);
    if (!e || !e.tris) continue;
    ensureGpu(e);
    const M = sc.worldMatrix(id);
    R.drawMesh(e.gpu, M, { color: sc.isHole(n) ? [0.6, 0.7, 0.95] : hexToRgb(n.color), opacity: 0.35, depthWrite: false, hole: sc.isHole(n), highlight: 0.8, noOverhang: true });
    R.drawLines(e.egpu, M, [0.35, 0.7, 1, 0.9], { depthTest: false });
  }

  drawGuides();
  drawMeasure();
  drawGizmo();
  drawLabels();
  updateViewCube();
}

// ---------------- gizmo ----------------
const AX = [[1, 0, 0], [0, 1, 0], [0, 0, 1]];
const AXC = [[0.94, 0.27, 0.27], [0.3, 0.8, 0.35], [0.25, 0.5, 0.98]];

function buildGizmo() {
  const arrow = mergeGeo(geoCylinder(0.022, 0.12, 0.8, 10), geoCylinder(0.075, 0.8, 1.02, 16, 0));
  const cube = geoBox([-0.06, -0.06, 0.86], [0.06, 0.06, 0.98]);
  const scaleAx = mergeGeo(geoCylinder(0.02, 0.12, 0.86, 8), cube);
  const ring = geoTorus(0.88, 0.016, 72, 6);
  gz = {
    arrows: [0, 1, 2].map((a) => R.createMesh(...Object.values(orientGeo(arrow, a)))),
    scales: [0, 1, 2].map((a) => R.createMesh(...Object.values(orientGeo(scaleAx, a)))),
    rings: [0, 1, 2].map((a) => R.createMesh(...Object.values(orientGeo(ring, a)))),
    plane: R.createMesh(...Object.values(geoBox([0.25, 0.25, -0.004], [0.42, 0.42, 0.004]))),
    center: R.createMesh(...Object.values(geoBox([-0.07, -0.07, -0.07], [0.07, 0.07, 0.07]))),
  };
}

function selRoots() {
  // selected nodes, ignoring ones whose ancestor is also selected
  return [...S.sel].filter((id) => {
    let p = sc.parentOf(id);
    while (p) { if (S.sel.has(p.id)) return false; p = sc.parentOf(p.id); }
    return !!sc.node(id) && !sc.node(id).locked;
  });
}

function gizmoFrame() {
  if (S.workspace === 'motion' || S.workspace === 'layout') return null;
  const ids = selRoots();
  if (!ids.length || S.mode !== 'select') return null;
  const box = sc.selectionBox(ids);
  if (!box) return null;
  let pivot = v3.mul(v3.add(box[0], box[1]), 0.5);
  let axes = AX;
  if (S.tool === 'scale' && ids.length === 1) {
    const W = sc.worldMatrix(ids[0]);
    axes = [0, 1, 2].map((i) => v3.norm([W[i * 4], W[i * 4 + 1], W[i * 4 + 2]]));
  }
  if (S.tool === 'move') pivot = [pivot[0], pivot[1], box[0][2]];
  const size = cam.ortho ? cam.dist * Math.tan((cam.fov * DEG) / 2) * 0.28 : v3.dist(eye, pivot) * 0.14;
  return { ids, box, pivot, axes, size };
}

function gizmoMatrix(f, axes) {
  const m = m4();
  if (axes) for (let i = 0; i < 3; i++) { m[i * 4] = axes[i][0] * f.size; m[i * 4 + 1] = axes[i][1] * f.size; m[i * 4 + 2] = axes[i][2] * f.size; }
  else { m[0] = m[5] = m[10] = f.size; }
  m[12] = f.pivot[0]; m[13] = f.pivot[1]; m[14] = f.pivot[2];
  return m;
}

function drawGizmo() {
  const f = drag?.gizmo ? drag.frame : gizmoFrame();
  if (!f) return;
  const M = gizmoMatrix(f, f.axes);
  const active = drag?.handle || hoverHandle;
  const col = (i, h) => (active && active.kind === h && active.axis === i ? [1, 0.85, 0.2] : AXC[i]);
  const o = { unlit: true, depthTest: false, noOverhang: true, noClip: true };
  if (S.tool === 'move') {
    for (let i = 0; i < 3; i++) R.drawMesh(gz.arrows[i], M, { ...o, color: col(i, 'axis') });
    // plane handles
    const planes = [[0, 1, 2], [0, 2, 1], [1, 2, 0]];
    for (const [a, b, n] of planes) {
      const pm = m4();
      const ax = f.axes[a], bx = f.axes[b], nx = f.axes[n];
      for (let k = 0; k < 3; k++) { pm[k] = ax[k] * f.size; pm[4 + k] = bx[k] * f.size; pm[8 + k] = nx[k] * f.size; }
      pm[12] = f.pivot[0]; pm[13] = f.pivot[1]; pm[14] = f.pivot[2];
      const hot = active && active.kind === 'plane' && active.axis === n;
      R.drawMesh(gz.plane, pm, { ...o, color: hot ? [1, 0.85, 0.2] : AXC[n], opacity: hot ? 0.9 : 0.55 });
    }
  } else if (S.tool === 'rotate') {
    for (let i = 0; i < 3; i++) R.drawMesh(gz.rings[i], M, { ...o, color: col(i, 'ring') });
    if (drag?.handle?.kind === 'ring') drawAngleLabel();
  } else {
    for (let i = 0; i < 3; i++) R.drawMesh(gz.scales[i], M, { ...o, color: col(i, 'scale') });
    R.drawMesh(gz.center, M, { ...o, color: active && active.kind === 'uniform' ? [1, 0.85, 0.2] : [0.9, 0.9, 0.95] });
  }
}

function drawAngleLabel() {}

function gizmoHit(sx, sy) {
  const f = gizmoFrame();
  if (!f) return null;
  const P = (p) => toScreen(p);
  const c = P(f.pivot);
  let best = null;
  const consider = (d, h) => { if (d < 11 && (!best || d < best.d)) best = { ...h, d }; };
  const end = (i, k) => v3.add(f.pivot, v3.mul(f.axes[i], f.size * k));
  if (S.tool === 'move') {
    for (let i = 0; i < 3; i++) {
      const a = P(end(i, 0.15)), b = P(end(i, 1.02));
      consider(segDist(sx, sy, a, b), { kind: 'axis', axis: i });
    }
    const planes = [[0, 1, 2], [0, 2, 1], [1, 2, 0]];
    for (const [a, b, n] of planes) {
      const q = [[0.25, 0.25], [0.42, 0.25], [0.42, 0.42], [0.25, 0.42]].map(([u, v]) => P(v3.add(f.pivot, v3.add(v3.mul(f.axes[a], u * f.size), v3.mul(f.axes[b], v * f.size)))));
      if (inQuad(sx, sy, q)) consider(4, { kind: 'plane', axis: n });
    }
  } else if (S.tool === 'rotate') {
    for (let i = 0; i < 3; i++) {
      const u = f.axes[(i + 1) % 3], w = f.axes[(i + 2) % 3];
      let prev = null, md = Infinity;
      for (let k = 0; k <= 64; k++) {
        const a = (k / 64) * Math.PI * 2;
        const p = P(v3.add(f.pivot, v3.add(v3.mul(u, Math.cos(a) * 0.88 * f.size), v3.mul(w, Math.sin(a) * 0.88 * f.size))));
        if (prev) md = Math.min(md, segDist(sx, sy, prev, p));
        prev = p;
      }
      consider(md, { kind: 'ring', axis: i });
    }
  } else {
    for (let i = 0; i < 3; i++) {
      const a = P(end(i, 0.15)), b = P(end(i, 0.98));
      consider(segDist(sx, sy, a, b), { kind: 'scale', axis: i });
    }
    const d = Math.hypot(sx - c[0], sy - c[1]);
    if (d < 14) best = { kind: 'uniform', axis: -1, d: 0 };
  }
  return best;
}

function segDist(px, py, a, b) {
  const dx = b[0] - a[0], dy = b[1] - a[1], l2 = dx * dx + dy * dy || 1;
  const t = Math.max(0, Math.min(1, ((px - a[0]) * dx + (py - a[1]) * dy) / l2));
  return Math.hypot(px - a[0] - t * dx, py - a[1] - t * dy);
}
function inQuad(x, y, q) {
  let inside = false;
  for (let i = 0, j = q.length - 1; i < q.length; j = i++) {
    if ((q[i][1] > y) !== (q[j][1] > y) && x < ((q[j][0] - q[i][0]) * (y - q[i][1])) / (q[j][1] - q[i][1]) + q[i][0]) inside = !inside;
  }
  return inside;
}

// ---------------- labels ----------------
function drawLabels() {
  let html = '';
  const ids = selRoots();
  const lab = (p, text, cls = '') => {
    const s = toScreen(p);
    if (s[3] <= 0) return;
    html += `<div class="lbl ${cls}" style="left:${s[0].toFixed(1)}px;top:${s[1].toFixed(1)}px">${text}</div>`;
  };
  if (ids.length && S.mode === 'select' && !(drag && drag.kind === 'orbit') && !(S.workspace === 'motion' || S.workspace === 'layout')) {
    const b = sc.selectionBox(ids);
    if (b) {
      const [lo, hi] = b;
      const cx = (lo[0] + hi[0]) / 2, cy = (lo[1] + hi[1]) / 2;
      const ed = !drag ? ' edit' : '';
      // sizes sit just outside the box so they don't cover the resize handles
      const off = worldPerPixel([cx, cy, lo[2]]) * 26;
      lab([cx, lo[1] - off, lo[2]], fmt(hi[0] - lo[0]), 'dim dx' + ed + '" data-axis="0" title="Click to type a new width');
      lab([hi[0] + off, cy, lo[2]], fmt(hi[1] - lo[1]), 'dim dy' + ed + '" data-axis="1" title="Click to type a new depth');
      lab([hi[0], lo[1], (lo[2] + hi[2]) / 2], fmt(hi[2] - lo[2]), 'dim dz' + ed + '" data-axis="2" title="Click to type a new height');
      if (drag && drag.info) lab([cx, cy, hi[2]], drag.info, 'info');
      else if (lo[2] > 0.05 || lo[2] < -0.05) lab([cx, cy, lo[2]], 'Z ' + fmt(lo[2]) + ' above bed', 'info faint edit" data-elev="1" title="Click to type a new height above the bed');
      // Tinkercad-style resize handles
      resizeHandles().forEach((rh, i) => {
        const s = toScreen(rh.p);
        if (s[3] <= 0) return;
        const hot = drag?.kind === 'resize' && drag.hi === i;
        html += `<div class="rh ${rh.cls}${hot ? ' hot' : ''}" data-h="${i}" title="${rh.tip}" style="left:${s[0].toFixed(1)}px;top:${s[1].toFixed(1)}px"></div>`;
      });
    }
  }
  if (placing && placing.visible && placing.label) lab(placing.labelPos, placing.label, 'info');
  for (const t of thinSpots) lab(t.at, `⚠ ${fmt(t.t)} mm wall`, 'thin');
  for (const m of measure.lines) lab(v3.lerp(m.a, m.b, 0.5), m.text, 'measure');
  if (measure.pts.length === 1 && measure.cursor) {
    const d = v3.dist(measure.pts[0], measure.cursor);
    lab(v3.lerp(measure.pts[0], measure.cursor, 0.5), fmt(d) + ' mm', 'measure faint');
  }
  if (S.mode === 'select' && dragBand) html += `<div class="band" style="left:${Math.min(dragBand.x0, dragBand.x1)}px;top:${Math.min(dragBand.y0, dragBand.y1)}px;width:${Math.abs(dragBand.x1 - dragBand.x0)}px;height:${Math.abs(dragBand.y1 - dragBand.y0)}px"></div>`;
  labels.innerHTML = html;
}
const fmt = (x) => (Math.abs(x) >= 100 ? x.toFixed(1) : x.toFixed(2)).replace(/\.?0+$/, '');
let dragBand = null;

let guideGpu = null;
function drawGuides() {
  if (guideGpu) { R.free(guideGpu); guideGpu = null; }
  const L = drag && drag.guides ? drag.guides.flat() : [];
  if (!L.length) return;
  guideGpu = R.createLines(L);
  R.drawLines(guideGpu, m4(), [1, 0.25, 0.85, 1], { depthTest: false, noClip: true });
}

// ---------------- measure ----------------
let measureGpu = null;
function drawMeasure() {
  if (S.mode !== 'measure' && !measure.lines.length) return;
  const L = [];
  for (const m of measure.lines) L.push(...m.a, ...m.b);
  if (measure.pts.length === 1 && measure.cursor) L.push(...measure.pts[0], ...measure.cursor);
  for (const p of [...measure.pts, ...(measure.cursor ? [measure.cursor] : [])]) {
    const s = cam.dist * 0.008;
    L.push(p[0] - s, p[1], p[2], p[0] + s, p[1], p[2], p[0], p[1] - s, p[2], p[0], p[1] + s, p[2], p[0], p[1], p[2] - s, p[0], p[1], p[2] + s);
  }
  if (measureGpu) R.free(measureGpu);
  measureGpu = L.length ? R.createLines(L) : null;
  R.drawLines(measureGpu, m4(), [1, 0.8, 0.15, 1], { depthTest: false, noClip: true });
}

export function startMeasure() {
  S.mode = 'measure'; measure = { pts: [], lines: [], cursor: null };
  canvas.style.cursor = 'crosshair';
  hint('<b>Measure:</b> click two points on your parts or the bed. <kbd>Esc</kbd> to finish.');
  emit('mode'); needs = true;
}
export function stopMeasure() {
  S.mode = 'select'; measure = { pts: [], lines: [], cursor: null };
  canvas.style.cursor = '';
  emit('mode'); needs = true;
}
function measurePoint(e) {
  const ray = rayAt(e.clientX, e.clientY);
  const h = pick(ray, 0, { deep: false, locked: true });
  if (h) return snapVertex(h.point);
  return groundHit(ray);
}
function snapVertex(p) {
  // snap to nearby bounding-box corners / centers for clean numbers
  return p.map((x) => (Math.abs(x - Math.round(x * 10) / 10) < 0.02 ? Math.round(x * 10) / 10 : x));
}

// ---------------- click-a-face mode ----------------
let facePick = null;
export function startFacePick(msg, fn) {
  cancelPlacing(false);
  facePick = fn; S.mode = 'facepick';
  canvas.style.cursor = 'crosshair';
  hint(msg + ' <kbd>Esc</kbd> to cancel.');
  emit('mode'); needs = true;
}
export function stopFacePick() { if (!facePick) return; facePick = null; S.mode = 'select'; canvas.style.cursor = ''; hint(''); emit('mode'); needs = true; }

// ---------------- placing new parts ----------------
export function startPlacing(n, opts = {}) {
  cancelPlacing(false);
  placing = { node: n, visible: false, fromDrag: !!opts.fromDrag, keep: !!opts.keep };
  S.mode = 'place';
  canvas.style.cursor = 'copy';
  hint(`<b>Place ${n.name}:</b> click on the bed or on a face of a part. ${n.hole ? 'Holes drill into the face you click.' : ''} <kbd>Esc</kbd> to cancel · hold <kbd>Shift</kbd> to place several.`);
  getGeom(n);
  emit('mode');
}
export function cancelPlacing(notify = true) {
  if (!placing) return;
  placing = null; S.mode = 'select'; canvas.style.cursor = '';
  needs = true;
  if (notify) { hint(''); emit('mode'); }
}
export const isPlacing = () => !!placing;

const AUTO_DEPTH = new Set(['screwhole', 'threadhole', 'slot']);

function updatePlacing(cx, cy) {
  const n = placing.node;
  const ray = rayAt(cx, cy);
  const h = pick(ray, 0, { solidOnly: true });
  const lb = getGeom(n);
  const lo = lb && lb.tris ? lb.lo : [0, 0, 0];
  const g = S.settings.grid;
  placing.label = '';
  if (h) {
    let p = h.point;
    const flat = Math.abs(h.normal[2]) > 0.999;
    if (flat) p = [snap(p[0], g), snap(p[1], g), p[2]];
    const Rm = alignZ(h.normal);
    n.rot = eulerFromMat(Rm);
    n.pos = p;
    n.scale = [1, 1, 1];
    placing.hit = h;
    placing.label = 'on surface';
    // smart drill: holes go all the way through
    if (n.hole && S.settings.autoDepth && AUTO_DEPTH.has(n.type)) {
      const back = { o: v3.add(h.point, v3.mul(h.normal, -0.02)), d: v3.mul(h.normal, -1) };
      const top = sc.node(h.id);
      const e = getGeom(top);
      const exit = hitMesh(e, sc.localMatrix(top), back);
      if (exit && exit.t < 500) {
        const depth = Math.round((exit.t + 0.02 + 0.4) * 10) / 10;
        if (n.params.depth !== depth) { n.params = { ...n.params, depth }; }
        placing.label = `drills ${fmt(depth)} mm (through)`;
      }
    }
  } else {
    const p = groundHit(ray);
    if (!p) { placing.visible = false; needs = true; return; }
    n.rot = [0, 0, 0];
    n.pos = [snap(p[0], g), snap(p[1], g), -lo[2]];
    placing.hit = null;
  }
  placing.labelPos = [n.pos[0], n.pos[1], n.pos[2]];
  placing.visible = true;
  needs = true;
}

function commitPlacing(shift) {
  if (!placing || !placing.visible) return;
  const n = placing.node;
  const keep = shift || placing.keep;
  cb.onPlace(n);
  if (keep) {
    const c = sc.cloneNode(n);
    placing.node = c;
    getGeom(c);
  } else {
    placing = null; S.mode = 'select'; canvas.style.cursor = ''; hint(''); emit('mode');
  }
  needs = true;
}

function onDragOver(e) {
  if (!e.dataTransfer.types.includes('application/x-makie')) return;
  e.preventDefault();
  e.dataTransfer.dropEffect = 'copy';
  if (!placing || !placing.fromDrag) {
    const type = window.__dragType;
    if (!type) return;
    startPlacing(cb.make(type), { fromDrag: true });
  }
  updatePlacing(e.clientX, e.clientY);
}
function onDrop(e) {
  if (!placing) return;
  e.preventDefault();
  updatePlacing(e.clientX, e.clientY);
  commitPlacing(false);
}

// ---------------- pointer handling ----------------
function onDown(e) {
  canvas.setPointerCapture?.(e.pointerId);
  const x = e.clientX, y = e.clientY;
  if (tween) tween = null;
  if (S.mode === 'place') {
    if (e.button === 0) { updatePlacing(x, y); commitPlacing(e.shiftKey); }
    else if (e.button === 2) cancelPlacing();
    return;
  }
  if (S.mode === 'facepick') {
    if (e.button === 0) { const hit = pick(x, y); if (hit) { const f = facePick; stopFacePick(); f(hit); } }
    else stopFacePick();
    return;
  }
  if (S.mode === 'measure' && e.button === 0) {
    const p = measurePoint(e);
    if (!p) return;
    if (measure.pts.length === 0) measure.pts = [p];
    else {
      const a = measure.pts[0], b = p, d = v3.sub(b, a);
      measure.lines.push({ a, b, text: `${fmt(v3.len(d))} mm <small>Δ ${fmt(Math.abs(d[0]))} · ${fmt(Math.abs(d[1]))} · ${fmt(Math.abs(d[2]))}</small>` });
      measure.pts = [];
    }
    needs = true;
    return;
  }
  const r = canvas.getBoundingClientRect();
  const sx = x - r.left, sy = y - r.top;
  // middle button or shift+right: pan; right: orbit
  if (e.button === 1 || (e.button === 2 && e.shiftKey) || (e.button === 0 && e.shiftKey && e.altKey)) { drag = { kind: 'pan', x, y, moved: false }; return; }
  if (e.button === 2) { drag = { kind: 'orbit', x, y, moved: false, right: true }; return; }
  if (e.button !== 0) return;
  const gh = gizmoHit(sx, sy);
  if (gh) { startGizmo(gh, x, y); return; }
  const h = pick(x, y);
  // clicking a part on a Wirie board picks that part
  const hn = h && sc.node(h.id);
  if (hn && hn.type === 'wirieboard') { const ref = wy.partAtTri(hn, h.tri, getGeom(hn)?.tris || 0); S.wiriePart = ref ? { id: hn.id, ref } : null; }
  else if (h) S.wiriePart = null;
  if (h) {
    const already = S.sel.has(h.id) || [...S.sel].some((id) => sc.topOf(id)?.id === h.id);
    if (e.shiftKey || e.ctrlKey || e.metaKey) { cb.onSelect(h.id, 'toggle'); drag = { kind: 'none' }; return; }
    if (!already) cb.onSelect(h.id, 'set');
    drag = { kind: 'maybeMove', x, y, hit: h, alt: e.altKey };
    return;
  }
  if (e.ctrlKey || e.metaKey) { drag = { kind: 'band', x, y }; dragBand = { x0: sx, y0: sy, x1: sx, y1: sy }; return; }
  drag = { kind: 'orbit', x, y, moved: false, empty: true };
}

function onMove(e) {
  const x = e.clientX, y = e.clientY;
  if (S.mode === 'place' && placing && !placing.fromDrag) {
    if (e.target === canvas || drag) updatePlacing(x, y);
    return;
  }
  if (S.mode === 'measure' && e.target === canvas) { measure.cursor = measurePoint(e); needs = true; }
  if (!drag) {
    if (e.target !== canvas) return;
    const r = canvas.getBoundingClientRect();
    const gh = S.mode === 'select' ? gizmoHit(x - r.left, y - r.top) : null;
    const prevH = hoverHandle, prevId = hoverId;
    hoverHandle = gh;
    hoverId = gh ? null : (S.mode === 'select' ? pick(x, y)?.id || null : null);
    canvas.style.cursor = gh ? 'grab' : hoverId ? 'pointer' : S.mode === 'measure' ? 'crosshair' : '';
    if (prevH?.kind !== gh?.kind || prevH?.axis !== gh?.axis || prevId !== hoverId) needs = true;
    return;
  }
  const dx = x - drag.x, dy = y - drag.y;
  switch (drag.kind) {
    case 'orbit':
      if (Math.abs(dx) + Math.abs(dy) > 2) drag.moved = true;
      cam.yaw -= dx * 0.35; cam.pitch = Math.max(-89.9, Math.min(89.9, cam.pitch + dy * 0.35));
      drag.x = x; drag.y = y; needs = true; break;
    case 'pan': {
      drag.moved = true;
      const h = canvas.clientHeight;
      const k = (2 * cam.dist * Math.tan((cam.fov * DEG) / 2)) / h;
      const right = [view[0], view[4], view[8]], up = [view[1], view[5], view[9]];
      cam.target = v3.add(cam.target, v3.add(v3.mul(right, -dx * k), v3.mul(up, dy * k)));
      drag.x = x; drag.y = y; needs = true; break;
    }
    case 'band': {
      const r = canvas.getBoundingClientRect();
      dragBand.x1 = x - r.left; dragBand.y1 = y - r.top; needs = true; break;
    }
    case 'maybeMove':
      if (Math.abs(dx) + Math.abs(dy) > 4) startBodyMove(drag);
      break;
    case 'move': doBodyMove(e); break;
    case 'gizmo': doGizmo(e); break;
    case 'resize': doResize(e); break;
  }
}

function onUp(e) {
  if (!drag) return;
  const d = drag;
  drag = null;
  if (d.kind === 'orbit' && !d.moved) {
    if (d.right) { const h = pick(e.clientX, e.clientY); cb.onContext(e, h?.id); }
    else if (d.empty) cb.onSelect(null, 'clear');
  }
  if (d.kind === 'maybeMove') {
    // plain click on an object: select just it (child stays selected if it was)
    if (!e.shiftKey) cb.onSelect(d.hit.id, S.sel.has(d.hit.id) ? 'set' : 'keep');
  }
  if (d.kind === 'band') {
    const b = dragBand; dragBand = null;
    const x0 = Math.min(b.x0, b.x1), x1 = Math.max(b.x0, b.x1), y0 = Math.min(b.y0, b.y1), y1 = Math.max(b.y0, b.y1);
    const ids = S.nodes.filter((n) => !n.hidden && !n.locked).filter((n) => {
      const bx = sc.worldBox(n.id); if (!bx) return false;
      const s = toScreen(v3.mul(v3.add(bx[0], bx[1]), 0.5));
      return s[0] >= x0 && s[0] <= x1 && s[1] >= y0 && s[1] <= y1;
    }).map((n) => n.id);
    cb.onSelect(ids, 'add');
  }
  if (d.kind === 'resize') {
    bakeScales(d.ids);
    if (d.changed) cb.onCommit('Resize');
  }
  if (d.kind === 'move' || d.kind === 'gizmo') {
    if (d.kind === 'gizmo' && S.tool === 'scale') bakeScales(d.ids);
    if (d.changed) cb.onCommit(d.kind === 'move' ? 'Move' : S.tool === 'rotate' ? 'Rotate' : S.tool === 'scale' ? 'Resize' : 'Move');
  }
  needs = true;
}

function onWheel(e) {
  e.preventDefault();
  if (tween) tween = null;
  const f = Math.exp((e.deltaY * (e.deltaMode === 1 ? 30 : 1)) * 0.0012);
  // zoom toward the cursor
  const ray = rayAt(e.clientX, e.clientY);
  const h = pick(ray, 0);
  // zoom towards what's under the mouse — but only if it's a sensible point. At grazing angles the
  // bed hit can be kilometres away, which used to throw the camera off into the distance.
  let target = h ? h.point : groundHit(ray);
  const L = camLimits();
  if (!target || target.some((x) => !Number.isFinite(x)) || v3.dist(target, cam.target) > Math.max(cam.dist * 2, 40) || Math.abs(target[0]) > L.span * 2 || Math.abs(target[1]) > L.span * 2) target = cam.target;
  const nd = Math.max(2, Math.min(20000, cam.dist * f));
  const k = 1 - nd / cam.dist;
  cam.target = v3.add(cam.target, v3.mul(v3.sub(target, cam.target), k));
  cam.dist = nd;
  needs = true;
}

function onDbl(e) {
  // double-click a group to select the part inside it
  const h = pick(e.clientX, e.clientY, { deep: true });
  if (h && h.nodeId !== h.id) cb.onSelect(h.nodeId, 'set');
  else if (h) cb.onSelect(h.id, 'set');
}

// ----- moving by dragging the part itself -----
// ---------- smart snapping: line up with other parts' edges and centres ----------
function snapTargets(ids) {
  const skip = new Set(ids.map((id) => sc.topOf(id)?.id));
  const out = [];
  for (const n of S.nodes) {
    if (n.hidden || skip.has(n.id)) continue;
    const b = sc.worldBox(n.id);
    if (b) out.push(b);
  }
  return out;
}
function worldPerPixel(at) {
  const h = canvas.clientHeight || 1;
  const d = cam.ortho ? cam.dist : v3.dist(eye, at);
  return (2 * d * Math.tan((cam.fov * DEG) / 2)) / h;
}
// returns the extra shift needed on an axis to make an edge/centre line up, plus a guide
function smartAxis(box0, delta, axis, targets, tol) {
  if (S.settings.smartSnap === false) return { d: 0 };
  const lo = box0[0][axis] + delta[axis], hi = box0[1][axis] + delta[axis], c = (lo + hi) / 2;
  let best = null;
  for (const t of targets) {
    const tl = t[0][axis], th = t[1][axis], tc = (tl + th) / 2;
    for (const [m, label] of [[lo, 'edge'], [hi, 'edge'], [c, 'centre']]) {
      for (const v of [tl, th, tc]) {
        const d = v - m;
        if (Math.abs(d) <= tol && (!best || Math.abs(d) < Math.abs(best.d))) best = { d, v, t, label };
      }
    }
  }
  return best || { d: 0 };
}
function guideLines(box, axis, hit) {
  if (!hit || !hit.t) return [];
  const o = axis === 0 ? 1 : 0;
  const z = Math.min(box[0][2], hit.t[0][2]) + 0.05;
  const a = Math.min(box[0][o], hit.t[0][o]) - 4, b = Math.max(box[1][o], hit.t[1][o]) + 4;
  if (axis === 2) {
    const x = Math.max(box[1][0], hit.t[1][0]) + 3;
    return [[x - 40, box[0][1], hit.v, x + 3, box[0][1], hit.v]];
  }
  const p = [0, 0, z], q = [0, 0, z];
  p[axis] = q[axis] = hit.v; p[o] = a; q[o] = b;
  return [[...p, ...q]];
}

function startBodyMove(d) {
  if (S.workspace === 'motion' || S.workspace === 'layout') { hint('Switch back to <b>Design</b> to move parts. Motion and Print layout are previews.'); return; }
  let ids = selRoots();
  if (!ids.length) return;
  if (d.alt) { ids = cb.duplicate(ids, true); }
  const z = d.hit.point[2];
  drag = {
    kind: 'move', ids, z, start: groundHit(rayAt(d.x, d.y), z) || d.hit.point, changed: false,
    worlds: ids.map((id) => sc.worldMatrix(id)), parents: ids.map((id) => sc.parentWorld(id)),
    box0: sc.selectionBox(ids), targets: snapTargets(ids),
  };
  canvas.style.cursor = 'grabbing';
}

function doBodyMove(e) {
  const p = groundHit(rayAt(e.clientX, e.clientY), drag.z);
  if (!p) return;
  let dx = p[0] - drag.start[0], dy = p[1] - drag.start[1];
  if (e.shiftKey) { if (Math.abs(dx) > Math.abs(dy)) dy = 0; else dx = 0; }
  const g = e.altKey ? 0 : S.settings.grid;
  dx = snap(dx, g); dy = snap(dy, g);
  drag.guides = [];
  if (!e.altKey && drag.box0) {
    const tol = worldPerPixel(drag.start) * 9;
    const sx = smartAxis(drag.box0, [dx, dy, 0], 0, drag.targets, tol);
    const sy = smartAxis(drag.box0, [dx, dy, 0], 1, drag.targets, tol);
    dx += sx.d; dy += sy.d;
    const nb = [v3.add(drag.box0[0], [dx, dy, 0]), v3.add(drag.box0[1], [dx, dy, 0])];
    drag.guides = [...guideLines(nb, 0, sx.t && sx), ...guideLines(nb, 1, sy.t && sy)];
  }
  applyDelta(translate(dx, dy, 0));
  drag.info = `X ${fmt(dx)}  Y ${fmt(dy)}` + (drag.guides.length ? '  · lined up' : '');
}

function applyDelta(D) {
  drag.ids.forEach((id, i) => {
    const n = sc.node(id);
    sc.setWorld(n, mul(D, drag.worlds[i]), drag.parents[i]);
  });
  drag.changed = true;
  cb.onLive();
  needs = true;
}

// ----- gizmo dragging -----
function startGizmo(h, x, y) {
  const f = gizmoFrame();
  const ids = f.ids;
  const ray = rayAt(x, y);
  drag = {
    kind: 'gizmo', gizmo: true, handle: h, frame: f, ids, x, y, changed: false, box0: sc.selectionBox(ids), targets: snapTargets(ids),
    worlds: ids.map((id) => sc.worldMatrix(id)), parents: ids.map((id) => sc.parentWorld(id)),
    nodes0: ids.map((id) => structuredClone(sc.node(id))),
  };
  if (h.kind === 'axis' || h.kind === 'scale') drag.t0 = axisParam(ray, f.pivot, f.axes[h.axis]);
  if (h.kind === 'plane') drag.p0 = planeHit(ray, f.pivot, f.axes[h.axis]);
  if (h.kind === 'ring') {
    const p = planeHit(ray, f.pivot, f.axes[h.axis]);
    drag.v0 = p ? v3.sub(p, f.pivot) : null;
  }
  if (h.kind === 'uniform') drag.y0 = y;
}

function doGizmo(e) {
  const f = drag.frame, h = drag.handle;
  const ray = rayAt(e.clientX, e.clientY);
  const g = S.settings.grid;
  if (h.kind === 'axis') {
    const t = axisParam(ray, f.pivot, f.axes[h.axis]);
    if (t == null || drag.t0 == null) return;
    let d = snap(t - drag.t0, e.altKey ? 0 : g);
    drag.guides = [];
    if (!e.altKey && drag.box0) {
      const dv = [0, 0, 0]; dv[h.axis] = d;
      const sm = smartAxis(drag.box0, dv, h.axis, drag.targets, worldPerPixel(f.pivot) * 9);
      if (sm.t) { d += sm.d; dv[h.axis] = d; drag.guides = guideLines([v3.add(drag.box0[0], dv), v3.add(drag.box0[1], dv)], h.axis, sm); }
    }
    const v = v3.mul(f.axes[h.axis], d);
    applyDelta(translate(...v));
    drag.info = `${'XYZ'[h.axis]} ${d >= 0 ? '+' : ''}${fmt(d)} mm` + (drag.guides.length ? ' · lined up' : '');
  } else if (h.kind === 'plane') {
    const p = planeHit(ray, f.pivot, f.axes[h.axis]);
    if (!p || !drag.p0) return;
    const d = v3.sub(p, drag.p0).map((x) => snap(x, e.altKey ? 0 : g));
    d[h.axis] = 0;
    applyDelta(translate(...d));
    drag.info = d.map((x, i) => (i === h.axis ? '' : 'XYZ'[i] + ' ' + fmt(x))).join('  ');
  } else if (h.kind === 'ring') {
    const p = planeHit(ray, f.pivot, f.axes[h.axis]);
    if (!p || !drag.v0) return;
    const v = v3.sub(p, f.pivot), ax = f.axes[h.axis];
    let a = Math.atan2(v3.dot(v3.cross(drag.v0, v), ax), v3.dot(drag.v0, v)) / DEG;
    const st = e.altKey ? 1 : S.settings.rotSnap;
    a = snap(a, st);
    const Rm = [rotX, rotY, rotZ][h.axis](a);
    const D = mul(translate(...f.pivot), mul(Rm, translate(...v3.mul(f.pivot, -1))));
    applyDelta(D);
    drag.info = `${fmt(a)}°`;
  } else if (h.kind === 'scale' || h.kind === 'uniform') {
    let k;
    if (h.kind === 'uniform') k = Math.exp(-(e.clientY - drag.y0) * 0.008);
    else {
      const t = axisParam(ray, f.pivot, f.axes[h.axis]);
      if (t == null || !drag.t0) return;
      k = t / drag.t0;
    }
    k = Math.max(0.01, k);
    drag.ids.forEach((id, i) => {
      const n = sc.node(id), n0 = drag.nodes0[i];
      const def = S.byType[n.type];
      if (def?.fixed) return;
      const e0 = getGeom(n0);
      const ext = e0 && e0.tris ? v3.sub(e0.hi, e0.lo) : [1, 1, 1];
      const ns = [...n0.scale];
      const axes = h.kind === 'uniform' || e.shiftKey || S.settings.lockRatio ? [0, 1, 2] : [h.axis];
      let info = [];
      for (const a of axes) {
        let size = Math.abs(ext[a] * n0.scale[a]) * k;
        if (!e.altKey && g > 0) size = Math.max(g, snap(size, g));
        ns[a] = Math.sign(n0.scale[a] || 1) * (size / (ext[a] || 1));
        info.push('XYZ'[a] + ' ' + fmt(size));
      }
      n.scale = ns;
      drag.info = info.join('  ');
    });
    drag.changed = true; cb.onLive(); needs = true;
  }
}

// ---------------- Tinkercad-style resize handles & typed sizes ----------------
function resizeHandles() {
  if (S.mode !== 'select' || S.tool === 'rotate' || S.workspace === 'motion' || S.workspace === 'layout') return [];
  if (drag && drag.kind !== 'resize') return [];
  const ids = selRoots();
  if (!ids.length || ids.every((id) => S.byType[sc.node(id)?.type]?.fixed)) return [];
  const b = drag?.kind === 'resize' ? drag.box : sc.selectionBox(ids);
  if (!b) return [];
  const [lo, hi] = b, cx = (lo[0] + hi[0]) / 2, cy = (lo[1] + hi[1]) / 2, z = lo[2];
  const c = 'Drag to resize · Shift keeps proportions · Alt turns off snapping';
  return [
    { s: [-1, -1, 0], p: [lo[0], lo[1], z], cls: 'corner', tip: c }, { s: [1, -1, 0], p: [hi[0], lo[1], z], cls: 'corner', tip: c },
    { s: [1, 1, 0], p: [hi[0], hi[1], z], cls: 'corner', tip: c }, { s: [-1, 1, 0], p: [lo[0], hi[1], z], cls: 'corner', tip: c },
    { s: [0, -1, 0], p: [cx, lo[1], z], cls: 'side', tip: c }, { s: [1, 0, 0], p: [hi[0], cy, z], cls: 'side', tip: c },
    { s: [0, 1, 0], p: [cx, hi[1], z], cls: 'side', tip: c }, { s: [-1, 0, 0], p: [lo[0], cy, z], cls: 'side', tip: c },
    { s: [0, 0, 1], p: [cx, cy, hi[2]], cls: 'top', tip: 'Drag to change the height · Shift keeps proportions' },
  ];
}

function onLabelDown(e) {
  if (e.button !== 0) return;
  const rh = e.target.closest('.rh');
  if (rh) { e.preventDefault(); e.stopPropagation(); startResize(+rh.dataset.h, e); return; }
  const lb = e.target.closest('.lbl.edit');
  if (lb) { e.preventDefault(); e.stopPropagation(); editLabel(lb); }
}

function resizeState(ids) {
  return {
    ids, box0: sc.selectionBox(ids), worlds: ids.map((id) => sc.worldMatrix(id)), parents: ids.map((id) => sc.parentWorld(id)),
    nodes0: ids.map((id) => { const n = sc.node(id); return { pos: [...n.pos], rot: [...n.rot], scale: [...n.scale] }; }),
    boxes0: ids.map((id) => sc.worldBox(id)),
  };
}

function startResize(i, e) {
  const h = resizeHandles()[i];
  if (!h) return;
  const ids = selRoots();
  const st = resizeState(ids);
  if (!st.box0) return;
  const ray = rayAt(e.clientX, e.clientY);
  const [lo, hi] = st.box0;
  drag = { kind: 'resize', hi: i, h, ...st, box: st.box0, changed: false, x: e.clientX, y: e.clientY };
  if (h.s[2]) drag.t0 = axisParam(ray, h.p, [0, 0, 1]);
  else drag.p0 = groundHit(ray, lo[2]);
  canvas.style.cursor = 'nwse-resize';
  needs = true;
}

function doResize(e) {
  const h = drag.h, [lo, hi] = drag.box0;
  const ray = rayAt(e.clientX, e.clientY);
  const g = e.altKey ? 0 : S.settings.grid;
  const size0 = v3.sub(hi, lo);
  const size = [...size0];
  if (h.s[2]) {
    const t = axisParam(ray, h.p, [0, 0, 1]);
    if (t == null || drag.t0 == null) return;
    size[2] = size0[2] + (t - drag.t0);
  } else {
    const p = groundHit(ray, lo[2]);
    if (!p || !drag.p0) return;
    for (const a of [0, 1]) if (h.s[a]) size[a] = size0[a] + h.s[a] * (p[a] - drag.p0[a]);
  }
  const moved = [0, 1, 2].filter((a) => h.s[a]);
  for (const a of moved) { if (g > 0) size[a] = snap(size[a], g); size[a] = Math.max(0.1, size[a]); }
  if (e.shiftKey || S.settings.lockRatio) {
    // keep proportions: the axis that changed most sets the scale for all three
    let k = 1, best = -1;
    for (const a of moved) { const r = size[a] / (size0[a] || 1); if (Math.abs(r - 1) > best) { best = Math.abs(r - 1); k = r; } }
    for (const a of [0, 1, 2]) size[a] = size0[a] * k;
  }
  // which side stays put: the opposite side for dragged axes, the middle for the others, the bottom for Z
  const nlo = [...lo], nhi = [...hi];
  for (const a of [0, 1, 2]) {
    if (a === 2 || h.s[a] > 0) { nlo[a] = lo[a]; nhi[a] = lo[a] + size[a]; }
    else if (h.s[a] < 0) { nhi[a] = hi[a]; nlo[a] = hi[a] - size[a]; }
    else { const c = (lo[a] + hi[a]) / 2; nlo[a] = c - size[a] / 2; nhi[a] = c + size[a] / 2; }
  }
  applyResize(drag, nlo, nhi);
  drag.box = [nlo, nhi];
  drag.info = `${fmt(size[0])} × ${fmt(size[1])} × ${fmt(size[2])} mm`;
  drag.changed = true; cb.onLive(); needs = true;
}

// stretch every part in st so the selection box goes from st.box0 to [nlo, nhi]
function applyResize(st, nlo, nhi) {
  const [lo, hi] = st.box0;
  const k = [0, 1, 2].map((a) => (nhi[a] - nlo[a]) / ((hi[a] - lo[a]) || 1));
  const map = (v, a) => nlo[a] + (v - lo[a]) * k[a];
  st.ids.forEach((id, i) => {
    const n = sc.node(id), n0 = st.nodes0[i], b0 = st.boxes0[i];
    n.pos = [...n0.pos]; n.rot = [...n0.rot]; n.scale = [...n0.scale];
    if (!b0) return;
    const fixed = S.byType[n.type]?.fixed;
    if (!fixed) {
      const W = st.worlds[i];
      const cols = [0, 1, 2].map((j) => v3.norm([W[j * 4], W[j * 4 + 1], W[j * 4 + 2]]));
      for (const a of [0, 1, 2]) {
        if (Math.abs(k[a] - 1) < 1e-9) continue;
        let j = 0, best = -1;
        for (let c = 0; c < 3; c++) { const d = Math.abs(cols[c][a]); if (d > best) { best = d; j = c; } }
        n.scale[j] = n0.scale[j] * k[a];
      }
    }
    const nb = sc.worldBox(id);
    if (!nb) return;
    const d = [0, 1, 2].map((a) => (fixed
      ? map((b0[0][a] + b0[1][a]) / 2, a) - (nb[0][a] + nb[1][a]) / 2
      : map(b0[0][a], a) - nb[0][a]));
    sc.setWorld(n, mul(translate(...d), sc.worldMatrix(id)), st.parents[i]);
  });
}

// thin-wall markers from the print check (cleared on the next edit)
let thinSpots = [];
export function showThinSpots(list) {
  thinSpots = list || [];
  needs = true;
  if (thinSpots.length) hint('<b>Thin walls:</b> the orange pins mark walls under 1.2 mm. Make the part bigger there, move the hole away from the edge, or raise the wall setting. They clear when you edit.');
}
export function clearThinSpots() { if (thinSpots.length) { thinSpots = []; needs = true; } }

let editing = null;
function editLabel(el) {
  const ids = selRoots();
  if (!ids.length) return;
  editing?.remove();
  const r = el.getBoundingClientRect(), stage = canvas.parentElement.getBoundingClientRect();
  const axis = el.dataset.axis != null ? +el.dataset.axis : -1;
  const box = sc.selectionBox(ids);
  const cur = axis >= 0 ? box[1][axis] - box[0][axis] : box[0][2];
  const inp = document.createElement('input');
  inp.className = 'lbl-edit ' + (['dx', 'dy', 'dz'][axis] || 'elev');
  inp.value = fmt(cur);
  inp.style.left = (r.left + r.width / 2 - stage.left) + 'px';
  inp.style.top = (r.top + r.height / 2 - stage.top) + 'px';
  canvas.parentElement.append(inp);
  editing = inp;
  inp.focus(); inp.select();
  let done = false;
  const finish = (ok) => {
    if (done) return; done = true;
    inp.remove(); if (editing === inp) editing = null;
    if (!ok) return;
    const v = evalNum(inp.value);
    if (!isFinite(v)) return;
    if (axis < 0) {
      const d = v - box[0][2];
      for (const id of ids) sc.setWorld(sc.node(id), mul(translate(0, 0, d), sc.worldMatrix(id)), sc.parentWorld(id));
      cb.onCommit('Move'); needs = true; return;
    }
    if (v <= 0) return;
    if (ids.every((id) => S.byType[sc.node(id)?.type]?.fixed)) { hint('This part has a fixed size — change its settings on the right instead.'); return; }
    resizeSelectionTo(ids, axis, v, S.settings.lockRatio);
  };
  inp.addEventListener('keydown', (e) => {
    e.stopPropagation();
    if (e.key === 'Enter') finish(true);
    if (e.key === 'Escape') finish(false);
    if (e.key === 'Tab') {
      e.preventDefault(); finish(true);
      const next = labels.querySelector(`.lbl.edit[data-axis="${(axis + (e.shiftKey ? 2 : 1)) % 3}"]`);
      setTimeout(() => { const n2 = labels.querySelector(`.lbl.edit[data-axis="${(axis + (e.shiftKey ? 2 : 1)) % 3}"]`) || next; if (n2) editLabel(n2); }, 60);
    }
  });
  inp.addEventListener('blur', () => finish(true));
}

// set one overall dimension of the selection (X/Y stay centred, Z keeps its bottom)
export function resizeSelectionTo(ids, axis, v, keepRatio = false) {
  const st = resizeState(ids);
  if (!st.box0) return;
  const [lo, hi] = st.box0;
  const k = v / ((hi[axis] - lo[axis]) || 1);
  const nlo = [...lo], nhi = [...hi];
  for (const a of [0, 1, 2]) {
    if (a !== axis && !keepRatio) continue;
    const size = (hi[a] - lo[a]) * k;
    if (a === 2) { nhi[a] = lo[a] + size; continue; }
    const c = (lo[a] + hi[a]) / 2;
    nlo[a] = c - size / 2; nhi[a] = c + size / 2;
  }
  applyResize(st, nlo, nhi);
  bakeScales(ids);
  cb.onCommit('Resize');
  needs = true;
}

// turn scaling of simple shapes into real dimensions (so a 20mm box scaled 2x becomes a 40mm box)
export function bakeScales(ids) {
  for (const id of ids) {
    const n = sc.node(id);
    const def = S.byType[n?.type];
    if (!def || !def.axes) continue;
    const ax = def.axes, s = [...n.scale];
    const map = { x: 0, y: 1, z: 2 };
    const p = { ...n.params };
    const byParam = {};
    for (const [k, key] of Object.entries(ax)) (byParam[key] ||= []).push(map[k]);
    let changed = false;
    for (const [key, axs] of Object.entries(byParam)) {
      const vals = axs.map((a) => Math.abs(s[a]));
      if (vals.some((v) => Math.abs(v - vals[0]) > 1e-6)) continue; // non-uniform on a shared param
      if (Math.abs(vals[0] - 1) < 1e-9) continue;
      p[key] = round((p[key] ?? 1) * vals[0], 3);
      for (const a of axs) s[a] = Math.sign(s[a] || 1);
      changed = true;
    }
    if (changed) { n.params = p; n.scale = s; }
  }
}

// ---------------- view cube ----------------
const FACES = {
  front: { n: [0, -1, 0], u: [1, 0, 0], v: [0, 0, 1], label: 'Front' },
  back: { n: [0, 1, 0], u: [-1, 0, 0], v: [0, 0, 1], label: 'Back' },
  right: { n: [1, 0, 0], u: [0, 1, 0], v: [0, 0, 1], label: 'Right' },
  left: { n: [-1, 0, 0], u: [0, -1, 0], v: [0, 0, 1], label: 'Left' },
  top: { n: [0, 0, 1], u: [1, 0, 0], v: [0, 1, 0], label: 'Top' },
  bottom: { n: [0, 0, -1], u: [1, 0, 0], v: [0, -1, 0], label: 'Bottom' },
};
let cubeEls = {};
function initViewCube() {
  const box = document.getElementById('viewcube');
  let dragC = null;
  box.addEventListener('pointerdown', (e) => { dragC = { x: e.clientX, y: e.clientY, moved: false }; box.setPointerCapture(e.pointerId); });
  box.addEventListener('pointermove', (e) => {
    if (!dragC) return;
    const dx = e.clientX - dragC.x, dy = e.clientY - dragC.y;
    if (Math.abs(dx) + Math.abs(dy) > 2) dragC.moved = true;
    if (!dragC.moved) return;
    tween = null;
    cam.yaw -= dx * 0.6; cam.pitch = Math.max(-89.9, Math.min(89.9, cam.pitch + dy * 0.6));
    dragC.x = e.clientX; dragC.y = e.clientY; needs = true;
  });
  box.addEventListener('pointerup', (e) => {
    const was = dragC; dragC = null;
    if (was && !was.moved) {
      // pointer capture swallows the click on the face, so find the face under the pointer
      box.releasePointerCapture?.(e.pointerId);
      const el = document.elementFromPoint(e.clientX, e.clientY);
      const k = el && Object.keys(cubeEls).find((key) => cubeEls[key] === el);
      if (k) setView(k);
    }
  });
  for (const [k, f] of Object.entries(FACES)) {
    const d = document.createElement('div');
    d.className = 'vc-face'; d.textContent = f.label; d.title = f.label + ' view';
    box.appendChild(d); cubeEls[k] = d;
  }
}
function updateViewCube() {
  const half = 28;
  for (const [k, f] of Object.entries(FACES)) {
    const F = [f.u[0], f.u[1], f.u[2], 0, -f.v[0], -f.v[1], -f.v[2], 0, f.n[0], f.n[1], f.n[2], 0, f.n[0] * half, f.n[1] * half, f.n[2] * half, 1];
    // world -> camera rotation, then flip Y for CSS
    const M = new Array(16).fill(0);
    for (let c = 0; c < 4; c++) for (let r = 0; r < 4; r++) {
      let s = 0;
      for (let q = 0; q < 4; q++) {
        const vr = r < 3 && q < 3 ? view[q * 4 + r] : r === q ? 1 : 0;
        s += vr * F[c * 4 + q];
      }
      M[c * 4 + r] = r === 1 ? -s : s;
    }
    M[12] = M[12]; M[13] = M[13];
    cubeEls[k].style.transform = `matrix3d(${M.map((x) => x.toFixed(5)).join(',')})`;
    // shade each face by how much it faces the viewer so it reads as a solid cube
    const facing = view[2] * f.n[0] + view[6] * f.n[1] + view[10] * f.n[2];
    cubeEls[k].style.filter = `brightness(${(0.72 + 0.4 * Math.max(0, facing)).toFixed(3)})`;
  }
}

export function pickRay(cx, cy) { return rayAt(cx, cy); }
export { hitMesh };

export { canvas as glCanvas };
export const getCam = () => ({ eye, view });
