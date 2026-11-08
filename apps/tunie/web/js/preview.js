/* Tunie — preview.js: G-code preview (layer view and rotatable 3D view).
   Part of the nrmluvdev maker suite. Created by nrmluvdev. */
'use strict';

const TYPE_COLORS = {
  'WALL-OUTER': '#8b7bff', 'WALL-INNER': '#5b8cff', 'SOLID-FILL': '#22d3ee', FILL: '#2fb7a3', BRIDGE: '#fbbf24',
  LABEL: '#f472b6', SKIRT: '#94a3b8', '': '#8b93a7',
};
const TYPE_NAMES = { 'WALL-OUTER': 'Outer wall', 'WALL-INNER': 'Inner wall', 'SOLID-FILL': 'Solid fill', FILL: 'Infill', BRIDGE: 'Bridge', LABEL: 'Labels', SKIRT: 'Prime line' };

function parseGcode(text) {
  const layers = [];
  let cur = { z: 0, segs: [] };
  let x = 0, y = 0, z = 0, type = '';
  let minX = Infinity, maxX = -Infinity, minY = Infinity, maxY = -Infinity;
  const types = new Set();
  const lines = text.split('\n');
  for (let i = 0; i < lines.length; i++) {
    const l = lines[i];
    if (l.charCodeAt(0) === 59) { // ';'
      if (l.startsWith(';LAYER:')) { if (cur.segs.length && l !== ';LAYER:0') { layers.push(cur); cur = { z, segs: [] }; } }
      else if (l.startsWith(';TYPE:')) type = l.slice(6).trim();
      continue;
    }
    if (!(l.startsWith('G1 ') || l.startsWith('G0 '))) continue;
    let nx = x, ny = y, nz = z, e = false;
    const parts = l.split(' ');
    for (let j = 1; j < parts.length; j++) {
      const p = parts[j], c = p[0];
      if (c === ';') break;
      const v = parseFloat(p.slice(1));
      if (c === 'X') nx = v; else if (c === 'Y') ny = v; else if (c === 'Z') nz = v; else if (c === 'E') e = v > 0;
    }
    z = nz;
    if (e && (nx !== x || ny !== y)) {
      cur.segs.push(x, y, nx, ny, type);
      types.add(type);
      if (type !== 'SKIRT') minX = Math.min(minX, x, nx), maxX = Math.max(maxX, x, nx), minY = Math.min(minY, y, ny), maxY = Math.max(maxY, y, ny);
      cur.z = z;
    }
    x = nx; y = ny;
  }
  if (cur.segs.length) layers.push(cur);
  return { layers, bounds: { minX, maxX, minY, maxY }, types: [...types] };
}

class Preview {
  constructor(root, printer) {
    this.root = root;
    this.printer = printer;
    this.mode = '3d';
    this.yaw = -0.7; this.pitch = 0.95; this.zoom = 1;
    root.innerHTML = `<div class="preview"><canvas></canvas>
      <div class="ov"><div class="seg"><button data-m="3d" class="on">3D</button><button data-m="2d">Layer</button></div></div>
      <div class="legend"></div>
      <div class="ov2"><span class="small muted" style="min-width:86px" data-lbl>Layer</span><input type="range" min="0" value="0"><span class="small mono" data-z style="min-width:64px;text-align:right"></span></div></div>`;
    this.cv = $('canvas', root);
    this.range = $('input[type=range]', root);
    $$('.seg button', root).forEach((b) => b.onclick = () => { this.mode = b.dataset.m; $$('.seg button', root).forEach((x) => x.classList.toggle('on', x === b)); this.draw(); });
    this.range.oninput = () => this.draw();
    let drag = null;
    this.cv.addEventListener('pointerdown', (e) => { drag = { x: e.clientX, y: e.clientY, yaw: this.yaw, pitch: this.pitch }; this.cv.setPointerCapture(e.pointerId); });
    this.cv.addEventListener('pointermove', (e) => {
      if (!drag || this.mode !== '3d') return;
      this.yaw = drag.yaw + (e.clientX - drag.x) * 0.008;
      this.pitch = clamp(drag.pitch + (e.clientY - drag.y) * 0.006, 0.15, 1.5);
      this.draw();
    });
    this.cv.addEventListener('pointerup', () => drag = null);
    this.cv.addEventListener('wheel', (e) => { e.preventDefault(); this.zoom = clamp(this.zoom * (e.deltaY < 0 ? 1.1 : 0.9), 0.4, 8); this.draw(); }, { passive: false });
    this.ro = new ResizeObserver(() => this.draw());
    this.ro.observe(this.cv);
  }
  load(gcode) {
    this.data = parseGcode(gcode);
    const n = this.data.layers.length;
    this.range.max = Math.max(0, n - 1);
    this.range.value = Math.max(0, n - 1);
    const legend = $('.legend', this.root);
    legend.innerHTML = this.data.types.filter((t) => TYPE_NAMES[t]).map((t) => `<span><i style="background:${TYPE_COLORS[t]}"></i>${TYPE_NAMES[t]}</span>`).join('');
    this.draw();
  }
  draw() {
    const cv = this.cv, ctx = cv.getContext('2d');
    const dpr = window.devicePixelRatio || 1;
    const W = cv.clientWidth, H = cv.clientHeight;
    if (!W || !H) return;
    cv.width = W * dpr; cv.height = H * dpr;
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    ctx.clearRect(0, 0, W, H);
    if (!this.data || !this.data.layers.length) return;
    const li = +this.range.value;
    const L = this.data.layers;
    $('[data-lbl]', this.root).textContent = `Layer ${li + 1} / ${L.length}`;
    $('[data-z]', this.root).textContent = `Z ${fmt(L[li].z, 2)} mm`;
    const p = this.printer;
    const round = p.bedShape === 'circle';
    const off = p.originCenter || round ? { x: 0, y: 0 } : { x: p.bedX / 2, y: p.bedY / 2 };
    const b = this.data.bounds;
    const cxm = (b.minX + b.maxX) / 2 - off.x, cym = (b.minY + b.maxY) / 2 - off.y;
    const span = Math.max(b.maxX - b.minX, b.maxY - b.minY, 20);
    if (this.mode === '2d') {
      const s = Math.min(W, H - 50) / (span * 1.25) * this.zoom;
      const tx = (x) => W / 2 + (x - off.x - cxm) * s, ty = (y) => (H - 40) / 2 - (y - off.y - cym) * s;
      this.drawBed2D(ctx, tx, ty, s, off);
      if (li > 0) this.strokeLayer(ctx, L[li - 1], tx, ty, 0.18, Math.max(1, s * 0.4));
      this.strokeLayer(ctx, L[li], tx, ty, 1, Math.max(1.2, s * 0.42));
      return;
    }
    // 3D
    const cy = Math.cos(this.yaw), sy = Math.sin(this.yaw), cp = Math.cos(this.pitch), sp = Math.sin(this.pitch);
    const maxZ = L[L.length - 1].z || 1;
    const sc = Math.min(W / (span * 1.6), (H - 70) / ((span * 1.42 * Math.cos(this.pitch) + maxZ * Math.sin(this.pitch)) * 1.12)) * this.zoom;
    const proj = (x, y, z) => {
      const X = x - off.x - cxm, Y = y - off.y - cym;
      const rx = X * cy - Y * sy, ry = X * sy + Y * cy;
      return [W / 2 + rx * sc, H / 2 - 20 + (-ry * cp - (z - maxZ / 2) * sp) * sc];
    };
    // bed grid
    ctx.lineWidth = 1;
    ctx.strokeStyle = 'rgba(120,135,170,.12)';
    const g = 10, ext = Math.ceil(span * 0.9 / g) * g;
    for (let k = -ext; k <= ext; k += g) {
      let a = proj(cxm + off.x + k, cym + off.y - ext, 0), c = proj(cxm + off.x + k, cym + off.y + ext, 0);
      ctx.beginPath(); ctx.moveTo(...a); ctx.lineTo(...c); ctx.stroke();
      a = proj(cxm + off.x - ext, cym + off.y + k, 0); c = proj(cxm + off.x + ext, cym + off.y + k, 0);
      ctx.beginPath(); ctx.moveTo(...a); ctx.lineTo(...c); ctx.stroke();
    }
    const step = Math.max(1, Math.floor(L.length / 400));
    for (let i = 0; i <= li; i += (i < li - 2 ? step : 1)) {
      const lay = L[i];
      const shade = 0.45 + 0.55 * (i / Math.max(1, li));
      const segs = lay.segs;
      let last = null;
      for (let k = 0; k < segs.length; k += 5) {
        const t = segs[k + 4];
        if (t !== last) {
          if (last !== null) ctx.stroke();
          ctx.beginPath();
          ctx.strokeStyle = TYPE_COLORS[t] || TYPE_COLORS[''];
          ctx.globalAlpha = i === li ? 1 : shade * 0.9;
          ctx.lineWidth = i === li ? 1.6 : 1.1;
          last = t;
        }
        const a = proj(segs[k], segs[k + 1], lay.z), c = proj(segs[k + 2], segs[k + 3], lay.z);
        ctx.moveTo(a[0], a[1]); ctx.lineTo(c[0], c[1]);
      }
      if (last !== null) ctx.stroke();
    }
    ctx.globalAlpha = 1;
  }
  drawBed2D(ctx, tx, ty, s, off) {
    const p = this.printer;
    ctx.fillStyle = 'rgba(255,255,255,.025)';
    ctx.strokeStyle = 'rgba(140,150,180,.25)';
    ctx.lineWidth = 1;
    ctx.beginPath();
    if (p.bedShape === 'circle') ctx.arc(tx(off.x), ty(off.y), p.bedX / 2 * s, 0, Math.PI * 2);
    else { const x0 = tx(off.x - p.bedX / 2), y0 = ty(off.y + p.bedY / 2); ctx.rect(x0, y0, p.bedX * s, p.bedY * s); }
    ctx.fill(); ctx.stroke();
    ctx.strokeStyle = 'rgba(140,150,180,.07)';
    for (let k = -p.bedX / 2; k <= p.bedX / 2; k += 10) { ctx.beginPath(); ctx.moveTo(tx(off.x + k), ty(off.y - p.bedY / 2)); ctx.lineTo(tx(off.x + k), ty(off.y + p.bedY / 2)); ctx.stroke(); }
    for (let k = -p.bedY / 2; k <= p.bedY / 2; k += 10) { ctx.beginPath(); ctx.moveTo(tx(off.x - p.bedX / 2), ty(off.y + k)); ctx.lineTo(tx(off.x + p.bedX / 2), ty(off.y + k)); ctx.stroke(); }
  }
  strokeLayer(ctx, lay, tx, ty, alpha, w) {
    ctx.globalAlpha = alpha; ctx.lineWidth = w; ctx.lineCap = 'round';
    let last = null;
    const segs = lay.segs;
    for (let k = 0; k < segs.length; k += 5) {
      const t = segs[k + 4];
      if (t !== last) { if (last !== null) ctx.stroke(); ctx.beginPath(); ctx.strokeStyle = TYPE_COLORS[t] || TYPE_COLORS['']; last = t; }
      ctx.moveTo(tx(segs[k]), ty(segs[k + 1])); ctx.lineTo(tx(segs[k + 2]), ty(segs[k + 3]));
    }
    if (last !== null) ctx.stroke();
    ctx.globalAlpha = 1;
  }
  destroy() { this.ro.disconnect(); }
}
