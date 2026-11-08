// Fast mesh-against-mesh collision tests for Motion mode (runs every frame while playing).
// Each part's triangles get a bounding-volume tree in the part's own frame; two parts are
// compared by walking both trees, so only triangles that are actually close get tested.

const trees = new WeakMap();

export function treeOf(e) {
  if (!e || !e.tris) return null;
  let t = trees.get(e);
  if (!t) { t = build(e.positions); trees.set(e, t); }
  return t;
}

// Long thin triangles (gear faces are fans from the centre, plates are two big triangles)
// would give huge boxes and make the search slow, so the collision copy splits them up.
function subdivide(P0) {
  let lo = [Infinity, Infinity, Infinity], hi = [-Infinity, -Infinity, -Infinity];
  for (let i = 0; i < P0.length; i += 3) for (let a = 0; a < 3; a++) { const v = P0[i + a]; if (v < lo[a]) lo[a] = v; if (v > hi[a]) hi[a] = v; }
  const diag = Math.hypot(hi[0] - lo[0], hi[1] - lo[1], hi[2] - lo[2]);
  const n0 = P0.length / 9;
  const maxE = Math.max(1.5, diag / 25);
  const out = [];
  const budget = Math.max(200000, n0 * 6);
  const split = (t, depth) => {
    const d01 = Math.hypot(t[3] - t[0], t[4] - t[1], t[5] - t[2]), d12 = Math.hypot(t[6] - t[3], t[7] - t[4], t[8] - t[5]), d20 = Math.hypot(t[0] - t[6], t[1] - t[7], t[2] - t[8]);
    const m = Math.max(d01, d12, d20);
    if (m <= maxE || depth > 12 || out.length / 9 > budget) { for (let k = 0; k < 9; k++) out.push(t[k]); return; }
    // split the longest edge in half
    let a, b, c;
    if (m === d01) { a = 0; b = 1; c = 2; } else if (m === d12) { a = 1; b = 2; c = 0; } else { a = 2; b = 0; c = 1; }
    const A = [t[a * 3], t[a * 3 + 1], t[a * 3 + 2]], B = [t[b * 3], t[b * 3 + 1], t[b * 3 + 2]], Cc = [t[c * 3], t[c * 3 + 1], t[c * 3 + 2]];
    const M = [(A[0] + B[0]) / 2, (A[1] + B[1]) / 2, (A[2] + B[2]) / 2];
    split([...A, ...M, ...Cc], depth + 1);
    split([...M, ...B, ...Cc], depth + 1);
  };
  for (let i = 0; i < n0; i++) split(Array.from(P0.subarray(i * 9, i * 9 + 9)), 0);
  return new Float32Array(out);
}

function build(P0) {
  const P = subdivide(P0);
  const n = P.length / 9;
  const idx = new Uint32Array(n);
  const cen = new Float32Array(n * 3);
  for (let i = 0; i < n; i++) {
    idx[i] = i;
    const o = i * 9;
    cen[i * 3] = (P[o] + P[o + 3] + P[o + 6]) / 3;
    cen[i * 3 + 1] = (P[o + 1] + P[o + 4] + P[o + 7]) / 3;
    cen[i * 3 + 2] = (P[o + 2] + P[o + 5] + P[o + 8]) / 3;
  }
  const nodes = [];
  const make = (s, e) => {
    const lo = [Infinity, Infinity, Infinity], hi = [-Infinity, -Infinity, -Infinity];
    for (let i = s; i < e; i++) {
      const o = idx[i] * 9;
      for (let k = 0; k < 9; k++) { const a = k % 3, v = P[o + k]; if (v < lo[a]) lo[a] = v; if (v > hi[a]) hi[a] = v; }
    }
    const node = { c: [(lo[0] + hi[0]) / 2, (lo[1] + hi[1]) / 2, (lo[2] + hi[2]) / 2], h: [(hi[0] - lo[0]) / 2, (hi[1] - lo[1]) / 2, (hi[2] - lo[2]) / 2], s, e, l: null, r: null };
    nodes.push(node);
    if (e - s > 4) {
      const ax = node.h[0] >= node.h[1] && node.h[0] >= node.h[2] ? 0 : node.h[1] >= node.h[2] ? 1 : 2;
      const sub = Array.from(idx.subarray(s, e)).sort((a, b) => cen[a * 3 + ax] - cen[b * 3 + ax]);
      idx.set(sub, s);
      const m = (s + e) >> 1;
      node.l = make(s, m); node.r = make(m, e);
    }
    return node;
  };
  const root = n ? make(0, n) : null;
  return { P, idx, root };
}

// ---- small maths on column-major 4×4 matrices ----
function relMatrix(MA, MB) {
  // T = inverse(MA) · MB  (MA is a rigid/scaled affine matrix)
  const a = MA, inv = new Float64Array(16);
  const det = a[0] * (a[5] * a[10] - a[9] * a[6]) - a[4] * (a[1] * a[10] - a[9] * a[2]) + a[8] * (a[1] * a[6] - a[5] * a[2]);
  const id = 1 / det;
  inv[0] = (a[5] * a[10] - a[9] * a[6]) * id; inv[4] = -(a[4] * a[10] - a[8] * a[6]) * id; inv[8] = (a[4] * a[9] - a[8] * a[5]) * id;
  inv[1] = -(a[1] * a[10] - a[9] * a[2]) * id; inv[5] = (a[0] * a[10] - a[8] * a[2]) * id; inv[9] = -(a[0] * a[9] - a[8] * a[1]) * id;
  inv[2] = (a[1] * a[6] - a[5] * a[2]) * id; inv[6] = -(a[0] * a[6] - a[4] * a[2]) * id; inv[10] = (a[0] * a[5] - a[4] * a[1]) * id;
  inv[12] = -(inv[0] * a[12] + inv[4] * a[13] + inv[8] * a[14]);
  inv[13] = -(inv[1] * a[12] + inv[5] * a[13] + inv[9] * a[14]);
  inv[14] = -(inv[2] * a[12] + inv[6] * a[13] + inv[10] * a[14]);
  inv[15] = 1;
  const T = new Float64Array(16);
  for (let c = 0; c < 4; c++) for (let r = 0; r < 4; r++) {
    T[c * 4 + r] = inv[r] * MB[c * 4] + inv[4 + r] * MB[c * 4 + 1] + inv[8 + r] * MB[c * 4 + 2] + inv[12 + r] * MB[c * 4 + 3];
  }
  return T;
}

// does segment p→q pass through the inside of triangle (v0,v1,v2)? Grazing contact
// (touching edges, faces lying flat on each other) is not counted — only real overlap.
const EPS = 1e-4;
function segTri(px, py, pz, qx, qy, qz, t) {
  const dx = qx - px, dy = qy - py, dz = qz - pz;
  const e1x = t[3] - t[0], e1y = t[4] - t[1], e1z = t[5] - t[2];
  const e2x = t[6] - t[0], e2y = t[7] - t[1], e2z = t[8] - t[2];
  const hx = dy * e2z - dz * e2y, hy = dz * e2x - dx * e2z, hz = dx * e2y - dy * e2x;
  const det = e1x * hx + e1y * hy + e1z * hz;
  const scale = Math.hypot(dx, dy, dz) * Math.hypot(e1x, e1y, e1z) * Math.hypot(e2x, e2y, e2z);
  if (Math.abs(det) <= 1e-9 * scale) return false; // parallel / coplanar
  const f = 1 / det;
  const sx = px - t[0], sy = py - t[1], sz = pz - t[2];
  const u = f * (sx * hx + sy * hy + sz * hz);
  if (u <= EPS || u >= 1 - EPS) return false;
  const qx2 = sy * e1z - sz * e1y, qy2 = sz * e1x - sx * e1z, qz2 = sx * e1y - sy * e1x;
  const v = f * (dx * qx2 + dy * qy2 + dz * qz2);
  if (v <= EPS || u + v >= 1 - EPS) return false;
  const s = f * (e2x * qx2 + e2y * qy2 + e2z * qz2);
  return s > EPS && s < 1 - EPS;
}

function triTri(a, b) {
  return segTri(a[0], a[1], a[2], a[3], a[4], a[5], b) || segTri(a[3], a[4], a[5], a[6], a[7], a[8], b) || segTri(a[6], a[7], a[8], a[0], a[1], a[2], b)
    || segTri(b[0], b[1], b[2], b[3], b[4], b[5], a) || segTri(b[3], b[4], b[5], b[6], b[7], b[8], a) || segTri(b[6], b[7], b[8], b[0], b[1], b[2], a);
}

/**
 * Do two posed parts overlap? Returns the contact point (in world space) or null.
 * eA/eB are geometry entries (local triangles), MA/MB their world matrices.
 */
export function partsOverlap(eA, MA, eB, MB) {
  const A = treeOf(eA), B = treeOf(eB);
  if (!A?.root || !B?.root) return null;
  const T = relMatrix(MA, MB);
  const PA = A.P, PB = B.P;
  // B's axes expressed in A's frame (unit columns + scale), for the box-vs-box test
  const sc = [Math.hypot(T[0], T[1], T[2]), Math.hypot(T[4], T[5], T[6]), Math.hypot(T[8], T[9], T[10])];
  const R = [[T[0] / sc[0], T[4] / sc[1], T[8] / sc[2]], [T[1] / sc[0], T[5] / sc[1], T[9] / sc[2]], [T[2] / sc[0], T[6] / sc[1], T[10] / sc[2]]];
  const AR = R.map((r) => r.map((v) => Math.abs(v) + 1e-9));
  const t = [0, 0, 0], bh = [0, 0, 0];
  // oriented-box overlap test (separating axis theorem, 15 axes)
  const obb = (na, nb) => {
    const c = nb.c;
    const cx = T[0] * c[0] + T[4] * c[1] + T[8] * c[2] + T[12] - na.c[0];
    const cy = T[1] * c[0] + T[5] * c[1] + T[9] * c[2] + T[13] - na.c[1];
    const cz = T[2] * c[0] + T[6] * c[1] + T[10] * c[2] + T[14] - na.c[2];
    t[0] = cx; t[1] = cy; t[2] = cz;
    const a = na.h;
    bh[0] = nb.h[0] * sc[0]; bh[1] = nb.h[1] * sc[1]; bh[2] = nb.h[2] * sc[2];
    for (let i = 0; i < 3; i++) if (Math.abs(t[i]) > a[i] + bh[0] * AR[i][0] + bh[1] * AR[i][1] + bh[2] * AR[i][2]) return false;
    for (let j = 0; j < 3; j++) {
      const tj = t[0] * R[0][j] + t[1] * R[1][j] + t[2] * R[2][j];
      if (Math.abs(tj) > a[0] * AR[0][j] + a[1] * AR[1][j] + a[2] * AR[2][j] + bh[j]) return false;
    }
    for (let i = 0; i < 3; i++) {
      const i1 = (i + 1) % 3, i2 = (i + 2) % 3;
      for (let j = 0; j < 3; j++) {
        const j1 = (j + 1) % 3, j2 = (j + 2) % 3;
        const ra = a[i1] * AR[i2][j] + a[i2] * AR[i1][j];
        const rb = bh[j1] * AR[i][j2] + bh[j2] * AR[i][j1];
        if (Math.abs(t[i2] * R[i1][j] - t[i1] * R[i2][j]) > ra + rb) return false;
      }
    }
    return true;
  };
  const ta = new Float64Array(9);
  const bt = new Float64Array(9 * 8), bb = new Float64Array(6 * 8);
  const tri = new Float64Array(9);
  const stack = [A.root, B.root];
  let tests = 0;
  while (stack.length) {
    const nb = stack.pop(), na = stack.pop();
    if (!obb(na, nb)) continue;
    const leafA = !na.l, leafB = !nb.l;
    if (leafA && leafB) {
      // move B's few triangles into A's frame once, with their boxes
      const nB = nb.e - nb.s;
      for (let j = 0; j < nB; j++) {
        const ob = B.idx[nb.s + j] * 9, o = j * 9;
        let lx = Infinity, ly = Infinity, lz = Infinity, hx = -Infinity, hy = -Infinity, hz = -Infinity;
        for (let k = 0; k < 3; k++) {
          const x = PB[ob + k * 3], y = PB[ob + k * 3 + 1], z = PB[ob + k * 3 + 2];
          const X = T[0] * x + T[4] * y + T[8] * z + T[12], Y = T[1] * x + T[5] * y + T[9] * z + T[13], Z = T[2] * x + T[6] * y + T[10] * z + T[14];
          bt[o + k * 3] = X; bt[o + k * 3 + 1] = Y; bt[o + k * 3 + 2] = Z;
          if (X < lx) lx = X; if (X > hx) hx = X; if (Y < ly) ly = Y; if (Y > hy) hy = Y; if (Z < lz) lz = Z; if (Z > hz) hz = Z;
        }
        bb[j * 6] = lx; bb[j * 6 + 1] = ly; bb[j * 6 + 2] = lz; bb[j * 6 + 3] = hx; bb[j * 6 + 4] = hy; bb[j * 6 + 5] = hz;
      }
      for (let i = na.s; i < na.e; i++) {
        const oa = A.idx[i] * 9;
        for (let k = 0; k < 9; k++) ta[k] = PA[oa + k];
        const lx = Math.min(ta[0], ta[3], ta[6]), hx = Math.max(ta[0], ta[3], ta[6]);
        const ly = Math.min(ta[1], ta[4], ta[7]), hy = Math.max(ta[1], ta[4], ta[7]);
        const lz = Math.min(ta[2], ta[5], ta[8]), hz = Math.max(ta[2], ta[5], ta[8]);
        for (let j = 0; j < nB; j++) {
          const q = j * 6;
          if (bb[q] > hx || bb[q + 3] < lx || bb[q + 1] > hy || bb[q + 4] < ly || bb[q + 2] > hz || bb[q + 5] < lz) continue;
          for (let k = 0; k < 9; k++) tri[k] = bt[j * 9 + k];
          tests++;
          if (triTri(ta, tri)) {
            const cx = (ta[0] + ta[3] + ta[6]) / 3, cy = (ta[1] + ta[4] + ta[7]) / 3, cz = (ta[2] + ta[5] + ta[8]) / 3;
            stats.tests += tests;
            return [MA[0] * cx + MA[4] * cy + MA[8] * cz + MA[12], MA[1] * cx + MA[5] * cy + MA[9] * cz + MA[13], MA[2] * cx + MA[6] * cy + MA[10] * cz + MA[14]];
          }
        }
      }
      continue;
    }
    // open up the bigger box
    const sa = na.h[0] + na.h[1] + na.h[2], sb = (nb.h[0] * sc[0] + nb.h[1] * sc[1] + nb.h[2] * sc[2]);
    if (leafB || (!leafA && sa >= sb)) { stack.push(na.l, nb, na.r, nb); }
    else { stack.push(na, nb.l, na, nb.r); }
  }
  stats.tests += tests;
  return null;
}
export const stats = { tests: 0 };
