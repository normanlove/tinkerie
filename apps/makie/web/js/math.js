// Small linear-algebra kit. Matrices are column-major Float64Array(16) like WebGL.
export const DEG = Math.PI / 180;

export const v3 = {
  add: (a, b) => [a[0] + b[0], a[1] + b[1], a[2] + b[2]],
  sub: (a, b) => [a[0] - b[0], a[1] - b[1], a[2] - b[2]],
  mul: (a, s) => [a[0] * s, a[1] * s, a[2] * s],
  dot: (a, b) => a[0] * b[0] + a[1] * b[1] + a[2] * b[2],
  cross: (a, b) => [a[1] * b[2] - a[2] * b[1], a[2] * b[0] - a[0] * b[2], a[0] * b[1] - a[1] * b[0]],
  len: (a) => Math.hypot(a[0], a[1], a[2]),
  norm: (a) => { const l = Math.hypot(a[0], a[1], a[2]) || 1; return [a[0] / l, a[1] / l, a[2] / l]; },
  lerp: (a, b, t) => [a[0] + (b[0] - a[0]) * t, a[1] + (b[1] - a[1]) * t, a[2] + (b[2] - a[2]) * t],
  dist: (a, b) => Math.hypot(a[0] - b[0], a[1] - b[1], a[2] - b[2]),
  min: (a, b) => [Math.min(a[0], b[0]), Math.min(a[1], b[1]), Math.min(a[2], b[2])],
  max: (a, b) => [Math.max(a[0], b[0]), Math.max(a[1], b[1]), Math.max(a[2], b[2])],
};

export function m4() { const m = new Float64Array(16); m[0] = m[5] = m[10] = m[15] = 1; return m; }

export function mul(a, b) {
  const o = new Float64Array(16);
  for (let c = 0; c < 4; c++) for (let r = 0; r < 4; r++) {
    let s = 0;
    for (let k = 0; k < 4; k++) s += a[k * 4 + r] * b[c * 4 + k];
    o[c * 4 + r] = s;
  }
  return o;
}

export function translate(x, y, z) { const m = m4(); m[12] = x; m[13] = y; m[14] = z; return m; }
export function scale(x, y, z) { const m = m4(); m[0] = x; m[5] = y; m[10] = z; return m; }
export function rotX(d) { const m = m4(), c = Math.cos(d * DEG), s = Math.sin(d * DEG); m[5] = c; m[6] = s; m[9] = -s; m[10] = c; return m; }
export function rotY(d) { const m = m4(), c = Math.cos(d * DEG), s = Math.sin(d * DEG); m[0] = c; m[2] = -s; m[8] = s; m[10] = c; return m; }
export function rotZ(d) { const m = m4(), c = Math.cos(d * DEG), s = Math.sin(d * DEG); m[0] = c; m[1] = s; m[4] = -s; m[5] = c; return m; }

// Same order as the Go engine: T * Rz * Ry * Rx * S
export function trs(pos, rot, sc) {
  const s = sc || [1, 1, 1];
  return mul(translate(pos[0], pos[1], pos[2]), mul(rotZ(rot[2]), mul(rotY(rot[1]), mul(rotX(rot[0]), scale(s[0] || 1, s[1] || 1, s[2] || 1)))));
}

export function rotMat(rot) { return mul(rotZ(rot[2]), mul(rotY(rot[1]), rotX(rot[0]))); }

export function apply(m, p) {
  return [m[0] * p[0] + m[4] * p[1] + m[8] * p[2] + m[12], m[1] * p[0] + m[5] * p[1] + m[9] * p[2] + m[13], m[2] * p[0] + m[6] * p[1] + m[10] * p[2] + m[14]];
}
export function applyDir(m, p) {
  return [m[0] * p[0] + m[4] * p[1] + m[8] * p[2], m[1] * p[0] + m[5] * p[1] + m[9] * p[2], m[2] * p[0] + m[6] * p[1] + m[10] * p[2]];
}
export function project(m, p) { // with perspective divide
  const x = m[0] * p[0] + m[4] * p[1] + m[8] * p[2] + m[12];
  const y = m[1] * p[0] + m[5] * p[1] + m[9] * p[2] + m[13];
  const z = m[2] * p[0] + m[6] * p[1] + m[10] * p[2] + m[14];
  const w = m[3] * p[0] + m[7] * p[1] + m[11] * p[2] + m[15];
  return [x / w, y / w, z / w, w];
}

export function invert(m) {
  const inv = new Float64Array(16);
  inv[0] = m[5] * m[10] * m[15] - m[5] * m[11] * m[14] - m[9] * m[6] * m[15] + m[9] * m[7] * m[14] + m[13] * m[6] * m[11] - m[13] * m[7] * m[10];
  inv[4] = -m[4] * m[10] * m[15] + m[4] * m[11] * m[14] + m[8] * m[6] * m[15] - m[8] * m[7] * m[14] - m[12] * m[6] * m[11] + m[12] * m[7] * m[10];
  inv[8] = m[4] * m[9] * m[15] - m[4] * m[11] * m[13] - m[8] * m[5] * m[15] + m[8] * m[7] * m[13] + m[12] * m[5] * m[11] - m[12] * m[7] * m[9];
  inv[12] = -m[4] * m[9] * m[14] + m[4] * m[10] * m[13] + m[8] * m[5] * m[14] - m[8] * m[6] * m[13] - m[12] * m[5] * m[10] + m[12] * m[6] * m[9];
  inv[1] = -m[1] * m[10] * m[15] + m[1] * m[11] * m[14] + m[9] * m[2] * m[15] - m[9] * m[3] * m[14] - m[13] * m[2] * m[11] + m[13] * m[3] * m[10];
  inv[5] = m[0] * m[10] * m[15] - m[0] * m[11] * m[14] - m[8] * m[2] * m[15] + m[8] * m[3] * m[14] + m[12] * m[2] * m[11] - m[12] * m[3] * m[10];
  inv[9] = -m[0] * m[9] * m[15] + m[0] * m[11] * m[13] + m[8] * m[1] * m[15] - m[8] * m[3] * m[13] - m[12] * m[1] * m[11] + m[12] * m[3] * m[9];
  inv[13] = m[0] * m[9] * m[14] - m[0] * m[10] * m[13] - m[8] * m[1] * m[14] + m[8] * m[2] * m[13] + m[12] * m[1] * m[10] - m[12] * m[2] * m[9];
  inv[2] = m[1] * m[6] * m[15] - m[1] * m[7] * m[14] - m[5] * m[2] * m[15] + m[5] * m[3] * m[14] + m[13] * m[2] * m[7] - m[13] * m[3] * m[6];
  inv[6] = -m[0] * m[6] * m[15] + m[0] * m[7] * m[14] + m[4] * m[2] * m[15] - m[4] * m[3] * m[14] - m[12] * m[2] * m[7] + m[12] * m[3] * m[6];
  inv[10] = m[0] * m[5] * m[15] - m[0] * m[7] * m[13] - m[4] * m[1] * m[15] + m[4] * m[3] * m[13] + m[12] * m[1] * m[7] - m[12] * m[3] * m[5];
  inv[14] = -m[0] * m[5] * m[14] + m[0] * m[6] * m[13] + m[4] * m[1] * m[14] - m[4] * m[2] * m[13] - m[12] * m[1] * m[6] + m[12] * m[2] * m[5];
  inv[3] = -m[1] * m[6] * m[11] + m[1] * m[7] * m[10] + m[5] * m[2] * m[11] - m[5] * m[3] * m[10] - m[9] * m[2] * m[7] + m[9] * m[3] * m[6];
  inv[7] = m[0] * m[6] * m[11] - m[0] * m[7] * m[10] - m[4] * m[2] * m[11] + m[4] * m[3] * m[10] + m[8] * m[2] * m[7] - m[8] * m[3] * m[6];
  inv[11] = -m[0] * m[5] * m[11] + m[0] * m[7] * m[9] + m[4] * m[1] * m[11] - m[4] * m[3] * m[9] - m[8] * m[1] * m[7] + m[8] * m[3] * m[5];
  inv[15] = m[0] * m[5] * m[10] - m[0] * m[6] * m[9] - m[4] * m[1] * m[10] + m[4] * m[2] * m[9] + m[8] * m[1] * m[6] - m[8] * m[2] * m[5];
  let det = m[0] * inv[0] + m[1] * inv[4] + m[2] * inv[8] + m[3] * inv[12];
  if (Math.abs(det) < 1e-18) return m4();
  det = 1 / det;
  for (let i = 0; i < 16; i++) inv[i] *= det;
  return inv;
}

export function det3(m) {
  return m[0] * (m[5] * m[10] - m[9] * m[6]) - m[4] * (m[1] * m[10] - m[9] * m[2]) + m[8] * (m[1] * m[6] - m[5] * m[2]);
}

// normal matrix (inverse transpose of upper 3x3) as Float32Array(9)
export function normalMat(m) {
  const i = invert(m);
  return new Float32Array([i[0], i[4], i[8], i[1], i[5], i[9], i[2], i[6], i[10]]);
}

// Euler angles (deg) for R = Rz*Ry*Rx from a pure rotation matrix
export function eulerFromMat(m) {
  const r20 = m[2], r21 = m[6], r22 = m[10], r10 = m[1], r00 = m[0];
  const b = Math.asin(Math.max(-1, Math.min(1, -r20)));
  let a, c;
  if (Math.abs(r20) < 0.99999) { a = Math.atan2(r21, r22); c = Math.atan2(r10, r00); }
  else { a = 0; c = Math.atan2(-m[4], m[5]); }
  return [clean(a / DEG), clean(b / DEG), clean(c / DEG)];
}
function clean(x) { const r = Math.round(x * 1e4) / 1e4; return Object.is(r, -0) ? 0 : r; }

// Split an affine matrix into pos / rot / scale (handles mirroring)
export function decompose(m) {
  const pos = [m[12], m[13], m[14]];
  let sx = Math.hypot(m[0], m[1], m[2]), sy = Math.hypot(m[4], m[5], m[6]), sz = Math.hypot(m[8], m[9], m[10]);
  if (det3(m) < 0) sx = -sx;
  const r = m4();
  r[0] = m[0] / sx; r[1] = m[1] / sx; r[2] = m[2] / sx;
  r[4] = m[4] / sy; r[5] = m[5] / sy; r[6] = m[6] / sy;
  r[8] = m[8] / sz; r[9] = m[9] / sz; r[10] = m[10] / sz;
  const rnd = (x) => Math.round(x * 1e5) / 1e5;
  return { pos: pos.map(rnd), rot: eulerFromMat(r), scale: [rnd(sx), rnd(sy), rnd(sz)] };
}

// rotation matrix that maps +Z onto direction n
export function alignZ(n) {
  n = v3.norm(n);
  const x = Math.abs(n[2]) < 0.95 ? v3.norm(v3.cross([0, 0, 1], n)) : v3.norm(v3.cross([0, 1, 0], n));
  const y = v3.cross(n, x);
  const m = m4();
  m[0] = x[0]; m[1] = x[1]; m[2] = x[2];
  m[4] = y[0]; m[5] = y[1]; m[6] = y[2];
  m[8] = n[0]; m[9] = n[1]; m[10] = n[2];
  return m;
}

export function perspective(fovy, aspect, near, far) {
  const f = 1 / Math.tan(fovy / 2), nf = 1 / (near - far), m = new Float64Array(16);
  m[0] = f / aspect; m[5] = f; m[10] = (far + near) * nf; m[11] = -1; m[14] = 2 * far * near * nf;
  return m;
}
export function ortho(l, r, b, t, n, f) {
  const m = m4();
  m[0] = 2 / (r - l); m[5] = 2 / (t - b); m[10] = -2 / (f - n);
  m[12] = -(r + l) / (r - l); m[13] = -(t + b) / (t - b); m[14] = -(f + n) / (f - n);
  return m;
}
export function lookAt(eye, target, up) {
  const z = v3.norm(v3.sub(eye, target)), x = v3.norm(v3.cross(up, z)), y = v3.cross(z, x), m = m4();
  m[0] = x[0]; m[4] = x[1]; m[8] = x[2];
  m[1] = y[0]; m[5] = y[1]; m[9] = y[2];
  m[2] = z[0]; m[6] = z[1]; m[10] = z[2];
  m[12] = -v3.dot(x, eye); m[13] = -v3.dot(y, eye); m[14] = -v3.dot(z, eye);
  return m;
}

// Möller–Trumbore; returns t or -1
export function rayTri(o, d, a, b, c) {
  const e1x = b[0] - a[0], e1y = b[1] - a[1], e1z = b[2] - a[2];
  const e2x = c[0] - a[0], e2y = c[1] - a[1], e2z = c[2] - a[2];
  const px = d[1] * e2z - d[2] * e2y, py = d[2] * e2x - d[0] * e2z, pz = d[0] * e2y - d[1] * e2x;
  const det = e1x * px + e1y * py + e1z * pz;
  if (Math.abs(det) < 1e-12) return -1;
  const inv = 1 / det;
  const tx = o[0] - a[0], ty = o[1] - a[1], tz = o[2] - a[2];
  const u = (tx * px + ty * py + tz * pz) * inv;
  if (u < 0 || u > 1) return -1;
  const qx = ty * e1z - tz * e1y, qy = tz * e1x - tx * e1z, qz = tx * e1y - ty * e1x;
  const v = (d[0] * qx + d[1] * qy + d[2] * qz) * inv;
  if (v < 0 || u + v > 1) return -1;
  const t = (e2x * qx + e2y * qy + e2z * qz) * inv;
  return t > 1e-7 ? t : -1;
}

export function rayBox(o, d, lo, hi) {
  let t0 = -Infinity, t1 = Infinity;
  for (let i = 0; i < 3; i++) {
    if (Math.abs(d[i]) < 1e-12) { if (o[i] < lo[i] || o[i] > hi[i]) return false; continue; }
    let a = (lo[i] - o[i]) / d[i], b = (hi[i] - o[i]) / d[i];
    if (a > b) [a, b] = [b, a];
    t0 = Math.max(t0, a); t1 = Math.min(t1, b);
    if (t0 > t1) return false;
  }
  return t1 > 0;
}

// bbox of a local bbox transformed by m
export function xformBox(m, lo, hi) {
  let a = [Infinity, Infinity, Infinity], b = [-Infinity, -Infinity, -Infinity];
  for (let i = 0; i < 8; i++) {
    const p = apply(m, [i & 1 ? hi[0] : lo[0], i & 2 ? hi[1] : lo[1], i & 4 ? hi[2] : lo[2]]);
    a = v3.min(a, p); b = v3.max(b, p);
  }
  return [a, b];
}

export const round = (x, n = 2) => { const f = 10 ** n; const r = Math.round(x * f) / f; return Object.is(r, -0) ? 0 : r; };
export const snap = (x, s) => s > 0 ? Math.round(x / s) * s : x;

// rotation matrix turning unit vector a onto unit vector b
export function rotBetween(a, b) {
  a = v3.norm(a); b = v3.norm(b);
  const c = v3.dot(a, b);
  const m = m4();
  if (c > 0.999999) return m;
  if (c < -0.999999) {
    // 180° about any axis perpendicular to a
    let p = Math.abs(a[0]) < 0.9 ? [1, 0, 0] : [0, 1, 0];
    p = v3.norm(v3.cross(a, p));
    const R = [[2 * p[0] * p[0] - 1, 2 * p[0] * p[1], 2 * p[0] * p[2]], [2 * p[1] * p[0], 2 * p[1] * p[1] - 1, 2 * p[1] * p[2]], [2 * p[2] * p[0], 2 * p[2] * p[1], 2 * p[2] * p[2] - 1]];
    for (let r = 0; r < 3; r++) for (let k = 0; k < 3; k++) m[k * 4 + r] = R[r][k];
    return m;
  }
  const v = v3.cross(a, b), k = 1 / (1 + c);
  const R = [
    [1 - (v[1] * v[1] + v[2] * v[2]) * k, -v[2] + v[0] * v[1] * k, v[1] + v[0] * v[2] * k],
    [v[2] + v[0] * v[1] * k, 1 - (v[0] * v[0] + v[2] * v[2]) * k, -v[0] + v[1] * v[2] * k],
    [-v[1] + v[0] * v[2] * k, v[0] + v[1] * v[2] * k, 1 - (v[0] * v[0] + v[1] * v[1]) * k],
  ];
  for (let r = 0; r < 3; r++) for (let q = 0; q < 3; q++) m[q * 4 + r] = R[r][q];
  return m;
}

// rotation about an arbitrary axis through the origin (degrees)
export function rotAxis(axis, deg) {
  const [x, y, z] = v3.norm(axis), a = deg * DEG, c = Math.cos(a), s = Math.sin(a), t = 1 - c;
  const m = m4();
  const R = [[t * x * x + c, t * x * y - s * z, t * x * z + s * y], [t * x * y + s * z, t * y * y + c, t * y * z - s * x], [t * x * z - s * y, t * y * z + s * x, t * z * z + c]];
  for (let r = 0; r < 3; r++) for (let q = 0; q < 3; q++) m[q * 4 + r] = R[r][q];
  return m;
}
