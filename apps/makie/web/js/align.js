// Auto-align and "snap together" (mate two faces) tools.
import { S, toast, hint } from './state.js';
import * as sc from './scene.js';
import * as vp from './viewport.js';
import { v3, mul, translate, apply, invert, rotBetween } from './math.js';

let C; // callbacks: commit, selRoots
export function initAlign(callbacks) { C = callbacks; }

function moveWorld(id, D) { sc.setWorld(sc.node(id), mul(D, sc.worldMatrix(id)), sc.parentWorld(id)); }

// Shift each part so it lines up exactly with the first selected part,
// choosing per axis the closest relation: centred, edges level, or touching.
export function autoAlign(ids) {
  if (ids.length < 2) { toast('Select the part to line up with first, then Shift+click the parts to move.'); return; }
  const target = sc.worldBox(ids[0]);
  if (!target) return;
  const report = [];
  for (const id of ids.slice(1)) {
    const b = sc.worldBox(id);
    if (!b) continue;
    const d = [0, 0, 0];
    const what = [];
    for (let a = 0; a < 3; a++) {
      const size = Math.max(b[1][a] - b[0][a], 0.1);
      const tc = (target[0][a] + target[1][a]) / 2, pc = (b[0][a] + b[1][a]) / 2;
      const cands = [
        [tc - pc, 'centred'],
        [target[0][a] - b[0][a], 'level'],
        [target[1][a] - b[1][a], 'level'],
        [target[1][a] - b[0][a], 'touching'],
        [target[0][a] - b[1][a], 'touching'],
      ];
      cands.sort((x, y) => Math.abs(x[0]) - Math.abs(y[0]));
      const [off, kind] = cands[0];
      // only nudge if it is "almost there": within half the part's size (or 8 mm)
      if (Math.abs(off) > 1e-6 && Math.abs(off) <= Math.max(8, size * 0.5)) { d[a] = off; what.push(`${'XYZ'[a]} ${kind}`); }
    }
    if (what.length) { moveWorld(id, translate(...d)); report.push(what.join(', ')); }
  }
  C.commit('Auto-align');
  toast(report.length ? 'Lined up: ' + report.join(' · ') : 'Already perfectly aligned.', 'ok', 4000);
}

// ---------- faces ----------
const qk = (x, y, z) => `${Math.round(x * 1e3)},${Math.round(y * 1e3)},${Math.round(z * 1e3)}`;

function adjacency(e) {
  if (e._adj) return e._adj;
  const P = e.positions, nt = P.length / 9;
  const byV = new Map(), normals = new Float64Array(nt * 3);
  for (let t = 0; t < nt; t++) {
    const i = t * 9;
    const a = [P[i], P[i + 1], P[i + 2]], b = [P[i + 3], P[i + 4], P[i + 5]], c = [P[i + 6], P[i + 7], P[i + 8]];
    const n = v3.norm(v3.cross(v3.sub(b, a), v3.sub(c, a)));
    normals.set(n, t * 3);
    for (const p of [a, b, c]) { const k = qk(...p); let l = byV.get(k); if (!l) byV.set(k, (l = [])); l.push(t); }
  }
  e._adj = { byV, normals };
  return e._adj;
}

// grow a region of connected triangles from the clicked one
function region(e, tri, smooth) {
  const { byV, normals } = adjacency(e);
  const P = e.positions;
  const n0 = [normals[tri * 3], normals[tri * 3 + 1], normals[tri * 3 + 2]];
  const p0 = [P[tri * 9], P[tri * 9 + 1], P[tri * 9 + 2]];
  const w0 = v3.dot(n0, p0);
  const seen = new Set([tri]), stack = [tri], out = [];
  const cosSmooth = Math.cos((32 * Math.PI) / 180);
  while (stack.length && out.length < 200000) {
    const t = stack.pop();
    out.push(t);
    const nt = [normals[t * 3], normals[t * 3 + 1], normals[t * 3 + 2]];
    for (let k = 0; k < 3; k++) {
      const i = t * 9 + k * 3;
      for (const u of byV.get(qk(P[i], P[i + 1], P[i + 2])) || []) {
        if (seen.has(u)) continue;
        const nu = [normals[u * 3], normals[u * 3 + 1], normals[u * 3 + 2]];
        const ok = smooth ? v3.dot(nu, nt) > cosSmooth
          : v3.dot(nu, n0) > 0.9999 && Math.abs(v3.dot(nu, [P[u * 9], P[u * 9 + 1], P[u * 9 + 2]]) - w0) < 1e-3;
        if (ok) { seen.add(u); stack.push(u); }
      }
    }
  }
  return out;
}

// describe the clicked face: a flat face (centre + normal) or a round one (axis + centre)
export function faceInfo(hit) {
  const e = hit.e, M = hit.M;
  const { normals } = adjacency(e);
  const P = e.positions;
  const flat = region(e, hit.tri, false);
  const NM = invert(M); // inverse for normals (transpose used below)
  const toW = (n) => v3.norm([NM[0] * n[0] + NM[1] * n[1] + NM[2] * n[2], NM[4] * n[0] + NM[5] * n[1] + NM[6] * n[2], NM[8] * n[0] + NM[9] * n[1] + NM[10] * n[2]]);
  const centroid = (tris) => {
    let A = 0, c = [0, 0, 0];
    for (const t of tris) {
      const i = t * 9;
      const a = apply(M, [P[i], P[i + 1], P[i + 2]]), b = apply(M, [P[i + 3], P[i + 4], P[i + 5]]), d = apply(M, [P[i + 6], P[i + 7], P[i + 8]]);
      const ar = v3.len(v3.cross(v3.sub(b, a), v3.sub(d, a))) / 2;
      A += ar; c = v3.add(c, v3.mul(v3.add(v3.add(a, b), d), ar / 3));
    }
    return { c: A ? v3.mul(c, 1 / A) : hit.point, A };
  };
  const nLocal = [normals[hit.tri * 3], normals[hit.tri * 3 + 1], normals[hit.tri * 3 + 2]];
  const fc = centroid(flat);
  // a single sliver of a curved surface → treat as a round face
  const smooth = region(e, hit.tri, true);
  const curved = smooth.length > flat.length * 4 && smooth.length >= 12;
  if (!curved) return { kind: 'flat', point: fc.c, normal: toW(nLocal) };
  // axis: cross product of the two most different normals in the region
  const n1 = nLocal;
  let n2 = n1, md = 1;
  for (const t of smooth) { const n = [normals[t * 3], normals[t * 3 + 1], normals[t * 3 + 2]]; const d = Math.abs(v3.dot(n, n1)); if (d < md) { md = d; n2 = n; } }
  const axis = v3.norm(v3.cross(toW(n1), toW(n2)));
  const sc2 = centroid(smooth).c;
  // axis line passes through the region's centroid (for a full round surface)
  return { kind: 'round', point: sc2, axis, normal: toW(nLocal) };
}

let step = null;
export function snapTogether(onDone) {
  step = { a: null };
  vp.startFacePick('<b>Snap together — step 1 of 2:</b> click the face of the part you want to <b>move</b>.', (h1) => {
    const top = sc.node(h1.id);
    if (top.locked) { toast('That part is locked.'); return; }
    step.a = { hit: h1, info: faceInfo(h1), id: h1.id };
    S.sel = new Set([h1.id]); C.refresh();
    vp.startFacePick('<b>Snap together — step 2 of 2:</b> click the face it should <b>attach to</b> on another part.', (h2) => {
      if (h2.id === step.a.id) { toast('Pick a face on a different part.'); return; }
      mate(step.a, { hit: h2, info: faceInfo(h2), id: h2.id });
      step = null;
      onDone && onDone();
    });
  });
}

function mate(A, B) {
  const a = A.info, b = B.info;
  let D;
  if (a.kind === 'round' && b.kind === 'round') {
    // concentric: line the axes up, keep the axial position
    let ax = a.axis, bx = b.axis;
    if (v3.dot(ax, bx) < 0) bx = v3.mul(bx, -1);
    const R = rotBetween(ax, bx);
    const Rp = mul(translate(...a.point), mul(R, translate(...v3.mul(a.point, -1))));
    const pa = apply(Rp, a.point);
    let d = v3.sub(b.point, pa);
    d = v3.sub(d, v3.mul(bx, v3.dot(d, bx))); // remove the along-axis part
    D = mul(translate(...d), Rp);
    toast('Snapped: the round faces are now concentric (same centre line). Slide along the axis with the arrows.', 'ok', 5000);
  } else {
    // flat (or flat + round): face A turns to face B, centres meet
    const na = a.kind === 'round' ? orientAxis(a.axis, b.normal) : a.normal;
    const nb = b.kind === 'round' ? orientAxis(b.axis, na) : b.normal;
    const R = rotBetween(na, v3.mul(nb, -1));
    const Rp = mul(translate(...a.point), mul(R, translate(...v3.mul(a.point, -1))));
    const pa = apply(Rp, a.point);
    D = mul(translate(...v3.sub(b.point, pa)), Rp);
    toast('Snapped: faces are touching and centred. Press Ctrl+Z to undo.', 'ok', 4000);
  }
  moveWorld(A.id, D);
  S.sel = new Set([A.id]);
  C.commit('Snap together');
  hint('');
}
const orientAxis = (axis, toward) => (v3.dot(axis, toward) > 0 ? v3.mul(axis, -1) : axis);
