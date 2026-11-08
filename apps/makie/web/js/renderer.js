// Minimal WebGL2 renderer for Makie.
const MESH_VS = `#version 300 es
in vec3 aPos; in vec3 aNor;
uniform mat4 uModel, uVP; uniform mat3 uNM;
out vec3 vN; out vec3 vW;
void main(){ vec4 w = uModel*vec4(aPos,1.0); vW = w.xyz; vN = uNM*aNor; gl_Position = uVP*w; }`;

const MESH_FS = `#version 300 es
precision highp float;
in vec3 vN; in vec3 vW;
uniform vec3 uColor; uniform float uOpacity; uniform int uHole; uniform int uUnlit; uniform int uOverhang;
uniform vec4 uClip; uniform int uClipOn; uniform vec3 uEye; uniform float uHi; uniform float uFloorZ; uniform int uMirror;
out vec4 frag;
void main(){
  if (uClipOn==1 && dot(vW,uClip.xyz) > uClip.w) discard;
  vec3 n = normalize(vN);
  bool back = !gl_FrontFacing;
  if (uMirror==1) back = !back;
  if (back) n = -n;
  vec3 base = uColor;
  if (uUnlit==1){ frag = vec4(base,uOpacity); return; }
  if (uOverhang==1 && n.z < -0.7071 && vW.z > uFloorZ+0.15) base = mix(base, vec3(0.95,0.12,0.12), 0.8);
  vec3 V = normalize(uEye - vW);
  vec3 L1 = normalize(V + vec3(0.3,0.2,0.9));
  vec3 L2 = normalize(vec3(-0.5,-0.7,0.4));
  float d = 0.58*max(dot(n,L1),0.0) + 0.22*max(dot(n,L2),0.0);
  float hemi = 0.30 + 0.12*n.z;
  vec3 H = normalize(L1+V);
  float spec = pow(max(dot(n,H),0.0), 48.0)*0.22;
  vec3 c = base*(d+hemi) + vec3(spec);
  float rim = pow(1.0-max(dot(n,V),0.0), 2.5);
  c += uHi*rim*vec3(0.25,0.6,1.0);
  if (uClipOn==1 && back) c = vec3(0.85,0.25,0.25)*0.7; // cut surface
  float a = uOpacity;
  if (uHole==1){
    float s = step(0.5, fract((gl_FragCoord.x + gl_FragCoord.y)/16.0));
    c = mix(c, c*0.72, s);
  }
  frag = vec4(c, a);
}`;

const LINE_VS = `#version 300 es
in vec3 aPos; uniform mat4 uModel, uVP; out vec3 vW;
void main(){ vec4 w = uModel*vec4(aPos,1.0); vW=w.xyz; gl_Position = uVP*w; }`;
const LINE_FS = `#version 300 es
precision highp float; in vec3 vW; uniform vec4 uColor; uniform vec4 uClip; uniform int uClipOn; out vec4 frag;
void main(){ if (uClipOn==1 && dot(vW,uClip.xyz) > uClip.w) discard; frag = uColor; }`;

// textured board faces: UVs come from the board-frame position
const TEX_VS = `#version 300 es
in vec3 aPos; uniform mat4 uModel, uVP; uniform vec4 uRect; uniform int uMirror;
out vec2 vUV; out vec3 vW;
void main(){ vec4 w = uModel*vec4(aPos,1.0); vW = w.xyz;
  float u = (aPos.x-uRect.x)/(uRect.z-uRect.x); float v = 1.0-(aPos.y-uRect.y)/(uRect.w-uRect.y);
  if (uMirror==1) u = 1.0-u;
  vUV = vec2(u,v); gl_Position = uVP*w; }`;
const TEX_FS = `#version 300 es
precision highp float; in vec2 vUV; in vec3 vW; uniform sampler2D uTex; uniform float uOpacity; uniform vec4 uClip; uniform int uClipOn; out vec4 frag;
void main(){ if (uClipOn==1 && dot(vW,uClip.xyz) > uClip.w) discard; vec4 c = texture(uTex, vUV); if (c.a < 0.08) discard; frag = vec4(c.rgb, c.a*uOpacity); }`;

function compile(gl, vs, fs) {
  const p = gl.createProgram();
  for (const [type, src] of [[gl.VERTEX_SHADER, vs], [gl.FRAGMENT_SHADER, fs]]) {
    const s = gl.createShader(type);
    gl.shaderSource(s, src); gl.compileShader(s);
    if (!gl.getShaderParameter(s, gl.COMPILE_STATUS)) throw new Error(gl.getShaderInfoLog(s));
    gl.attachShader(p, s);
  }
  gl.bindAttribLocation(p, 0, 'aPos'); gl.bindAttribLocation(p, 1, 'aNor');
  gl.linkProgram(p);
  if (!gl.getProgramParameter(p, gl.LINK_STATUS)) throw new Error(gl.getProgramInfoLog(p));
  const u = {};
  const n = gl.getProgramParameter(p, gl.ACTIVE_UNIFORMS);
  for (let i = 0; i < n; i++) { const info = gl.getActiveUniform(p, i); u[info.name] = gl.getUniformLocation(p, info.name); }
  return { p, u };
}

const f32 = (m) => (m instanceof Float32Array ? m : new Float32Array(m));

export class Renderer {
  constructor(canvas, opts = {}) {
    const gl = canvas.getContext('webgl2', { antialias: true, alpha: true, preserveDrawingBuffer: !!opts.preserve, premultipliedAlpha: false });
    if (!gl) throw new Error('WebGL2 is not available. Please update your graphics driver or browser.');
    this.gl = gl; this.canvas = canvas;
    this.mesh = compile(gl, MESH_VS, MESH_FS);
    this.line = compile(gl, LINE_VS, LINE_FS);
    this.tex = compile(gl, TEX_VS, TEX_FS);
    this.vp = new Float32Array(16); this.eye = [0, 0, 100];
    this.clip = null; this.overhang = false; this.floorZ = 0;
    gl.enable(gl.DEPTH_TEST);
    gl.enable(gl.BLEND);
    gl.blendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA);
  }

  resize() {
    const dpr = Math.min(window.devicePixelRatio || 1, 2);
    const w = Math.max(1, Math.round(this.canvas.clientWidth * dpr)), h = Math.max(1, Math.round(this.canvas.clientHeight * dpr));
    if (this.canvas.width !== w || this.canvas.height !== h) { this.canvas.width = w; this.canvas.height = h; }
    this.gl.viewport(0, 0, w, h);
  }

  createMesh(positions, normals) {
    const gl = this.gl;
    const vao = gl.createVertexArray(); gl.bindVertexArray(vao);
    const pb = gl.createBuffer(); gl.bindBuffer(gl.ARRAY_BUFFER, pb); gl.bufferData(gl.ARRAY_BUFFER, positions, gl.STATIC_DRAW);
    gl.enableVertexAttribArray(0); gl.vertexAttribPointer(0, 3, gl.FLOAT, false, 0, 0);
    const nb = gl.createBuffer(); gl.bindBuffer(gl.ARRAY_BUFFER, nb); gl.bufferData(gl.ARRAY_BUFFER, normals, gl.STATIC_DRAW);
    gl.enableVertexAttribArray(1); gl.vertexAttribPointer(1, 3, gl.FLOAT, false, 0, 0);
    gl.bindVertexArray(null);
    return { vao, bufs: [pb, nb], count: positions.length / 3 };
  }

  createLines(positions) {
    const gl = this.gl;
    const vao = gl.createVertexArray(); gl.bindVertexArray(vao);
    const pb = gl.createBuffer(); gl.bindBuffer(gl.ARRAY_BUFFER, pb); gl.bufferData(gl.ARRAY_BUFFER, f32(positions), gl.STATIC_DRAW);
    gl.enableVertexAttribArray(0); gl.vertexAttribPointer(0, 3, gl.FLOAT, false, 0, 0);
    gl.bindVertexArray(null);
    return { vao, bufs: [pb], count: positions.length / 3 };
  }

  free(obj) {
    if (!obj) return;
    const gl = this.gl;
    obj.bufs.forEach((b) => gl.deleteBuffer(b));
    gl.deleteVertexArray(obj.vao);
  }

  setCamera(view, proj, eye) {
    const vp = new Float64Array(16);
    for (let c = 0; c < 4; c++) for (let r = 0; r < 4; r++) {
      let s = 0; for (let k = 0; k < 4; k++) s += proj[k * 4 + r] * view[c * 4 + k]; vp[c * 4 + r] = s;
    }
    this.vp = new Float32Array(vp); this.vp64 = vp; this.eye = eye;
  }

  clear(rgb) {
    const gl = this.gl;
    gl.clearColor(rgb[0], rgb[1], rgb[2], rgb[3] ?? 1);
    gl.clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT);
  }

  _clipUniforms(u) {
    const gl = this.gl;
    if (this.clip) { gl.uniform1i(u.uClipOn, 1); gl.uniform4f(u.uClip, ...this.clip); } else gl.uniform1i(u.uClipOn, 0);
  }

  drawMesh(m, model, o = {}) {
    if (!m) return;
    const gl = this.gl, { p, u } = this.mesh;
    gl.useProgram(p);
    gl.uniformMatrix4fv(u.uModel, false, f32(model));
    gl.uniformMatrix4fv(u.uVP, false, this.vp);
    gl.uniformMatrix3fv(u.uNM, false, o.nm || identity3(model));
    gl.uniform3fv(u.uColor, o.color || [0.5, 0.5, 0.5]);
    gl.uniform1f(u.uOpacity, o.opacity ?? 1);
    gl.uniform1i(u.uHole, o.hole ? 1 : 0);
    gl.uniform1i(u.uUnlit, o.unlit ? 1 : 0);
    gl.uniform1i(u.uOverhang, this.overhang && !o.noOverhang ? 1 : 0);
    gl.uniform1f(u.uHi, o.highlight || 0);
    gl.uniform1f(u.uFloorZ, this.floorZ);
    gl.uniform3fv(u.uEye, this.eye);
    gl.uniform1i(u.uMirror, det3x3(model) < 0 ? 1 : 0);
    if (o.noClip) gl.uniform1i(u.uClipOn, 0); else this._clipUniforms(u);
    if (o.depthTest === false) gl.disable(gl.DEPTH_TEST); else gl.enable(gl.DEPTH_TEST);
    gl.depthMask(o.depthWrite !== false);
    if (o.offset) { gl.enable(gl.POLYGON_OFFSET_FILL); gl.polygonOffset(1, 1); }
    gl.bindVertexArray(m.vao);
    gl.drawArrays(gl.TRIANGLES, o.first || 0, o.count ?? m.count - (o.first || 0));
    gl.bindVertexArray(null);
    gl.disable(gl.POLYGON_OFFSET_FILL);
    gl.depthMask(true); gl.enable(gl.DEPTH_TEST);
  }

  // texture from an <img> (premultiplied off, mipmapped)
  createTexture(img) {
    const gl = this.gl, t = gl.createTexture();
    gl.bindTexture(gl.TEXTURE_2D, t);
    gl.pixelStorei(gl.UNPACK_FLIP_Y_WEBGL, false);
    gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA, gl.RGBA, gl.UNSIGNED_BYTE, img);
    gl.generateMipmap(gl.TEXTURE_2D);
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_LINEAR);
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR);
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE);
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE);
    const ext = gl.getExtension('EXT_texture_filter_anisotropic');
    if (ext) gl.texParameterf(gl.TEXTURE_2D, ext.TEXTURE_MAX_ANISOTROPY_EXT, 8);
    gl.bindTexture(gl.TEXTURE_2D, null);
    return t;
  }

  freeTexture(t) { if (t) this.gl.deleteTexture(t); }

  // draw a flat quad (mesh made with createMesh) with a picture mapped over rect [x0,y0,x1,y1]
  drawTextured(m, tex, model, rect, o = {}) {
    if (!m || !tex) return;
    const gl = this.gl, { p, u } = this.tex;
    gl.useProgram(p);
    gl.uniformMatrix4fv(u.uModel, false, f32(model));
    gl.uniformMatrix4fv(u.uVP, false, this.vp);
    gl.uniform4f(u.uRect, rect[0], rect[1], rect[2], rect[3]);
    gl.uniform1i(u.uMirror, o.mirror ? 1 : 0);
    gl.uniform1f(u.uOpacity, o.opacity ?? 1);
    this._clipUniforms(u);
    gl.activeTexture(gl.TEXTURE0); gl.bindTexture(gl.TEXTURE_2D, tex); gl.uniform1i(u.uTex, 0);
    gl.enable(gl.POLYGON_OFFSET_FILL); gl.polygonOffset(-1, -2);
    gl.bindVertexArray(m.vao);
    gl.drawArrays(gl.TRIANGLES, 0, m.count);
    gl.bindVertexArray(null);
    gl.disable(gl.POLYGON_OFFSET_FILL);
    gl.bindTexture(gl.TEXTURE_2D, null);
  }

  drawLines(l, model, color, o = {}) {
    if (!l || !l.count) return;
    const gl = this.gl, { p, u } = this.line;
    gl.useProgram(p);
    gl.uniformMatrix4fv(u.uModel, false, f32(model));
    gl.uniformMatrix4fv(u.uVP, false, this.vp);
    gl.uniform4fv(u.uColor, color.length === 4 ? color : [...color, 1]);
    if (o.noClip) gl.uniform1i(u.uClipOn, 0); else this._clipUniforms(u);
    if (o.depthTest === false) gl.disable(gl.DEPTH_TEST); else gl.enable(gl.DEPTH_TEST);
    gl.depthMask(o.depthWrite !== false);
    gl.bindVertexArray(l.vao);
    gl.drawArrays(o.strip ? gl.LINE_STRIP : gl.LINES, 0, l.count);
    gl.bindVertexArray(null);
    gl.depthMask(true); gl.enable(gl.DEPTH_TEST);
  }
}

function det3x3(m) { return m[0] * (m[5] * m[10] - m[9] * m[6]) - m[4] * (m[1] * m[10] - m[9] * m[2]) + m[8] * (m[1] * m[6] - m[5] * m[2]); }

function identity3(m) {
  // inverse-transpose of the upper 3x3
  const a = m[0], b = m[1], c = m[2], d = m[4], e = m[5], f = m[6], g = m[8], h = m[9], i = m[10];
  const A = e * i - f * h, B = -(d * i - f * g), C = d * h - e * g;
  const det = a * A + b * B + c * C || 1;
  return new Float32Array([
    A / det, B / det, C / det,
    -(b * i - c * h) / det, (a * i - c * g) / det, -(a * h - b * g) / det,
    (b * f - c * e) / det, -(a * f - c * d) / det, (a * e - b * d) / det,
  ]);
}

// ---------- simple JS geometry for gizmos and helpers ----------
export function geoCylinder(r, z0, z1, n = 16, r2) {
  r2 = r2 ?? r;
  const P = [], N = [];
  for (let i = 0; i < n; i++) {
    const a0 = (i / n) * Math.PI * 2, a1 = ((i + 1) / n) * Math.PI * 2;
    const c0 = Math.cos(a0), s0 = Math.sin(a0), c1 = Math.cos(a1), s1 = Math.sin(a1);
    const q = [[r * c0, r * s0, z0], [r * c1, r * s1, z0], [r2 * c1, r2 * s1, z1], [r2 * c0, r2 * s0, z1]];
    const nn = [[c0, s0, 0.3], [c1, s1, 0.3], [c1, s1, 0.3], [c0, s0, 0.3]];
    for (const k of [0, 1, 2, 0, 2, 3]) { P.push(...q[k]); N.push(...nn[k]); }
    P.push(0, 0, z0, r * c1, r * s1, z0, r * c0, r * s0, z0); N.push(0, 0, -1, 0, 0, -1, 0, 0, -1);
    if (r2 > 0) { P.push(0, 0, z1, r2 * c0, r2 * s0, z1, r2 * c1, r2 * s1, z1); N.push(0, 0, 1, 0, 0, 1, 0, 0, 1); }
  }
  return { positions: new Float32Array(P), normals: new Float32Array(N) };
}

export function geoTorus(R, r, n = 64, m = 8) {
  const P = [], N = [];
  const pt = (i, j) => {
    const u = (i / n) * Math.PI * 2, v = (j / m) * Math.PI * 2;
    const cx = Math.cos(u), sx = Math.sin(u);
    return [[(R + r * Math.cos(v)) * cx, (R + r * Math.cos(v)) * sx, r * Math.sin(v)], [Math.cos(v) * cx, Math.cos(v) * sx, Math.sin(v)]];
  };
  for (let i = 0; i < n; i++) for (let j = 0; j < m; j++) {
    const q = [pt(i, j), pt(i + 1, j), pt(i + 1, j + 1), pt(i, j + 1)];
    for (const k of [0, 1, 2, 0, 2, 3]) { P.push(...q[k][0]); N.push(...q[k][1]); }
  }
  return { positions: new Float32Array(P), normals: new Float32Array(N) };
}

export function geoBox(lo, hi) {
  const P = [], N = [];
  const f = (a, b, c, d, n) => { for (const p of [a, b, c, a, c, d]) { P.push(...p); N.push(...n); } };
  const [x0, y0, z0] = lo, [x1, y1, z1] = hi;
  f([x0, y0, z0], [x0, y1, z0], [x1, y1, z0], [x1, y0, z0], [0, 0, -1]);
  f([x0, y0, z1], [x1, y0, z1], [x1, y1, z1], [x0, y1, z1], [0, 0, 1]);
  f([x0, y0, z0], [x1, y0, z0], [x1, y0, z1], [x0, y0, z1], [0, -1, 0]);
  f([x0, y1, z0], [x0, y1, z1], [x1, y1, z1], [x1, y1, z0], [0, 1, 0]);
  f([x0, y0, z0], [x0, y0, z1], [x0, y1, z1], [x0, y1, z0], [-1, 0, 0]);
  f([x1, y0, z0], [x1, y1, z0], [x1, y1, z1], [x1, y0, z1], [1, 0, 0]);
  return { positions: new Float32Array(P), normals: new Float32Array(N) };
}

export function mergeGeo(...gs) {
  const n = gs.reduce((s, g) => s + g.positions.length, 0);
  const P = new Float32Array(n), N = new Float32Array(n);
  let o = 0;
  for (const g of gs) { P.set(g.positions, o); N.set(g.normals, o); o += g.positions.length; }
  return { positions: P, normals: N };
}

// rotate geometry so its +Z axis points along an axis (0=x,1=y,2=z)
export function orientGeo(g, axis) {
  if (axis === 2) return g;
  const P = new Float32Array(g.positions.length), N = new Float32Array(g.normals.length);
  const map = axis === 0 ? (x, y, z) => [z, x, y] : (x, y, z) => [y, z, x];
  for (let i = 0; i < P.length; i += 3) {
    const p = map(g.positions[i], g.positions[i + 1], g.positions[i + 2]);
    const q = map(g.normals[i], g.normals[i + 1], g.normals[i + 2]);
    P.set(p, i); N.set(q, i);
  }
  return { positions: P, normals: N };
}
