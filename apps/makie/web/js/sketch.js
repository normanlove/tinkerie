// 2D outline editor for the "Custom shape" (sketch) part.
import { modal, h } from './ui.js';
import { round } from './math.js';

const PRESETS = {
  'Rectangle': () => [[-15, -10], [15, -10], [15, 10], [-15, 10]],
  'Rounded square': () => { const p = []; const r = 5, w = 15; for (const [cx, cy, a0] of [[w - r, -w + r, -90], [w - r, w - r, 0], [-w + r, w - r, 90], [-w + r, -w + r, 180]]) for (let i = 0; i <= 4; i++) { const a = ((a0 + i * 22.5) * Math.PI) / 180; p.push([round(cx + r * Math.cos(a), 2), round(cy + r * Math.sin(a), 2)]); } return p; },
  'Circle': () => Array.from({ length: 24 }, (_, i) => [round(12 * Math.cos((i / 24) * Math.PI * 2), 2), round(12 * Math.sin((i / 24) * Math.PI * 2), 2)]),
  'L-shape': () => [[0, 0], [30, 0], [30, 6], [6, 6], [6, 30], [0, 30]],
  'T-slot': () => [[-10, 0], [10, 0], [10, 4], [3, 4], [3, 10], [-3, 10], [-3, 4], [-10, 4]],
  'Arrow': () => [[-15, -4], [5, -4], [5, -10], [16, 0], [5, 10], [5, 4], [-15, 4]],
  'Heart': () => Array.from({ length: 40 }, (_, i) => { const t = (i / 40) * Math.PI * 2; return [round(0.9 * 16 * Math.sin(t) ** 3, 2), round(0.9 * (13 * Math.cos(t) - 5 * Math.cos(2 * t) - 2 * Math.cos(3 * t) - Math.cos(4 * t)), 2)]; }),
  'Vase profile (revolve)': () => [[0, 0], [14, 0], [16, 8], [11, 25], [9, 34], [12, 42], [11, 42], [8, 34], [10, 25], [14.5, 8], [12.5, 1.5], [0, 1.5]],
  'Knob profile (revolve)': () => [[0, 0], [15, 0], [16, 2], [16, 10], [13, 14], [0, 14]],
  'Pulley profile (revolve)': () => [[4, 0], [16, 0], [16, 1.5], [12, 5], [12, 7], [16, 10.5], [16, 12], [4, 12]],
};

export function openSketch(params, onDone) {
  let pts = (params.points || []).map((p) => [p[0], p[1]]);
  let mode = params.mode || 'Extrude', smooth = !!params.smooth;
  let snapStep = 1, sel = -1, dragging = -1, hover = -1, hoverEdge = null, cursor = null;
  let view = { cx: 0, cy: 0, s: 8 };

  const canvas = h('canvas', { id: 'sk-canvas' });
  const coords = h('div', { class: 'note' }, ' ');
  const ptEdit = h('div');
  const side = h('div', { class: 'sk-side' });
  const rebuildSide = () => {
    side.innerHTML = '';
    side.append(h('h4', {}, 'Make 3D by'),
      h('div', { class: 'seg' },
        h('button', { class: mode === 'Extrude' ? 'on' : '', onclick: () => { mode = 'Extrude'; rebuildSide(); draw(); } }, 'Extrude'),
        h('button', { class: mode === 'Revolve' ? 'on' : '', onclick: () => { mode = 'Revolve'; rebuildSide(); draw(); } }, 'Revolve')),
      h('div', { class: 'note' }, mode === 'Revolve' ? 'Spins the outline around the dashed axis (X = 0). Draw on the right side only.' : 'Pushes the outline straight up into a solid.'),
      h('h4', {}, 'Options'),
      h('div', { class: 'field' }, h('label', {}, 'Smooth curves'), h('button', { class: 'toggle' + (smooth ? ' on' : ''), onclick: () => { smooth = !smooth; rebuildSide(); draw(); } })),
      h('div', { class: 'field' }, h('label', {}, 'Snap'), h('select', { onchange: (e) => { snapStep = +e.target.value; } }, ...[0, 0.1, 0.5, 1, 2, 5].map((s) => h('option', { value: s, selected: s === snapStep }, s ? s + ' mm' : 'Off')))),
      h('h4', {}, 'Start from'),
      h('select', { class: 'inp', onchange: (e) => { if (PRESETS[e.target.value]) { pts = PRESETS[e.target.value](); if (/revolve/.test(e.target.value)) mode = 'Revolve'; else mode = 'Extrude'; sel = -1; fit(); rebuildSide(); draw(); } e.target.value = ''; } },
        h('option', { value: '' }, 'Choose a preset…'), ...Object.keys(PRESETS).map((k) => h('option', { value: k }, k))),
      h('h4', {}, 'Edit'),
      h('button', { class: 'btn', onclick: () => { pts = []; sel = -1; draw(); } }, 'Clear all points'),
      h('button', { class: 'btn', onclick: () => { pts = pts.map(([x, y]) => [-x, y]).reverse(); draw(); } }, 'Mirror left ↔ right'),
      h('button', { class: 'btn', onclick: () => { fit(); draw(); } }, 'Zoom to fit'),
      ptEdit,
      h('div', { class: 'note', html: '<b>Click</b> to add a point (click on a line to insert one).<br><b>Drag</b> points to move them.<br><b>Right-click</b> a point to delete it.<br>Scroll to zoom, middle-drag to pan.' }));
    updatePtEdit();
  };
  const updatePtEdit = () => {
    ptEdit.innerHTML = '';
    if (sel < 0 || !pts[sel]) return;
    const mk = (i) => h('input', { class: 'inp', value: pts[sel][i], style: 'width:70px', onkeydown: (e) => e.stopPropagation(), onchange: (e) => { const v = parseFloat(e.target.value); if (isFinite(v)) { pts[sel][i] = v; draw(); } } });
    ptEdit.append(h('h4', {}, `Point ${sel + 1}`), h('div', { style: 'display:flex;gap:6px;align-items:center' }, 'X', mk(0), 'Y', mk(1)), h('button', { class: 'btn small danger', style: 'margin-top:6px', onclick: () => { pts.splice(sel, 1); sel = -1; draw(); updatePtEdit(); } }, 'Delete point'));
  };

  const body = h('div', { class: 'sketch-wrap' }, h('div', {}, canvas, coords), side);
  const m = modal('Draw your shape', body, [{ label: 'Cancel' }, {
    label: 'Done', primary: true, run: () => {
      if (pts.length < 3) { alert('An outline needs at least 3 points.'); return false; }
      onDone({ points: pts.map(([x, y]) => [round(x, 3), round(y, 3)]), mode, smooth });
    },
  }], { wide: true, sticky: true });
  void m;
  rebuildSide();

  const ctx = canvas.getContext('2d');
  const toS = ([x, y]) => [canvas.width / 2 + (x - view.cx) * view.s, canvas.height / 2 - (y - view.cy) * view.s];
  const toW = (sx, sy) => [view.cx + (sx - canvas.width / 2) / view.s, view.cy - (sy - canvas.height / 2) / view.s];
  const snapP = (p) => (snapStep ? p.map((v) => Math.round(v / snapStep) * snapStep) : p).map((v) => round(v, 3));

  function fit() {
    const r = canvas.getBoundingClientRect();
    canvas.width = r.width * devicePixelRatio; canvas.height = r.height * devicePixelRatio;
    if (!pts.length) { view = { cx: 0, cy: 0, s: Math.min(canvas.width, canvas.height) / 60 }; return; }
    let x0 = Infinity, y0 = Infinity, x1 = -Infinity, y1 = -Infinity;
    for (const [x, y] of pts) { x0 = Math.min(x0, x); y0 = Math.min(y0, y); x1 = Math.max(x1, x); y1 = Math.max(y1, y); }
    if (mode === 'Revolve') x0 = Math.min(x0, -5);
    view.cx = (x0 + x1) / 2; view.cy = (y0 + y1) / 2;
    view.s = Math.min(canvas.width / Math.max(10, (x1 - x0) * 1.4), canvas.height / Math.max(10, (y1 - y0) * 1.4));
  }

  function smoothPts(p) {
    if (!smooth || p.length < 3) return p;
    const out = [], n = p.length;
    for (let i = 0; i < n; i++) {
      const p0 = p[(i - 1 + n) % n], p1 = p[i], p2 = p[(i + 1) % n], p3 = p[(i + 2) % n];
      for (let k = 0; k < 8; k++) {
        const t = k / 8, t2 = t * t, t3 = t2 * t;
        out.push([0, 1].map((j) => 0.5 * (2 * p1[j] + (-p0[j] + p2[j]) * t + (2 * p0[j] - 5 * p1[j] + 4 * p2[j] - p3[j]) * t2 + (-p0[j] + 3 * p1[j] - 3 * p2[j] + p3[j]) * t3)));
      }
    }
    return out;
  }

  function draw() {
    const W = canvas.width, H = canvas.height, dpr = devicePixelRatio;
    const dark = document.body.classList.contains('dark');
    ctx.clearRect(0, 0, W, H);
    // grid
    const minor = view.s > 14 ? 1 : view.s > 4 ? 5 : 10;
    const [wx0, wy1] = toW(0, 0), [wx1, wy0] = toW(W, H);
    ctx.lineWidth = 1;
    for (let x = Math.floor(wx0 / minor) * minor; x <= wx1; x += minor) {
      const sx = toS([x, 0])[0];
      ctx.strokeStyle = x === 0 ? (mode === 'Revolve' ? '#f59e0b' : '#ef4444') : Math.abs(x % (minor * 10)) < 1e-6 ? (dark ? '#3a4150' : '#c5ccd6') : (dark ? '#232833' : '#e3e7ed');
      if (x === 0 && mode === 'Revolve') ctx.setLineDash([8 * dpr, 6 * dpr]);
      ctx.beginPath(); ctx.moveTo(sx, 0); ctx.lineTo(sx, H); ctx.stroke(); ctx.setLineDash([]);
    }
    for (let y = Math.floor(wy0 / minor) * minor; y <= wy1; y += minor) {
      const sy = toS([0, y])[1];
      ctx.strokeStyle = y === 0 ? '#22c55e' : Math.abs(y % (minor * 10)) < 1e-6 ? (dark ? '#3a4150' : '#c5ccd6') : (dark ? '#232833' : '#e3e7ed');
      ctx.beginPath(); ctx.moveTo(0, sy); ctx.lineTo(W, sy); ctx.stroke();
    }
    // revolve ghost
    if (mode === 'Revolve' && pts.length > 1) {
      ctx.fillStyle = 'rgba(245,158,11,0.08)';
      ctx.beginPath(); pts.forEach((p, i) => { const s = toS([-Math.abs(p[0]), p[1]]); i ? ctx.lineTo(...s) : ctx.moveTo(...s); }); ctx.closePath(); ctx.fill();
    }
    // shape
    const sp = smoothPts(pts);
    if (sp.length) {
      ctx.fillStyle = dark ? 'rgba(20,184,166,0.22)' : 'rgba(13,148,136,0.18)';
      ctx.strokeStyle = '#14b8a6'; ctx.lineWidth = 2 * dpr;
      ctx.beginPath(); sp.forEach((p, i) => { const s = toS(p); i ? ctx.lineTo(...s) : ctx.moveTo(...s); });
      if (pts.length > 2) ctx.closePath();
      ctx.fill(); ctx.stroke();
    }
    if (smooth && pts.length > 1) {
      ctx.strokeStyle = 'rgba(148,163,184,0.5)'; ctx.lineWidth = 1 * dpr; ctx.setLineDash([4 * dpr, 4 * dpr]);
      ctx.beginPath(); pts.forEach((p, i) => { const s = toS(p); i ? ctx.lineTo(...s) : ctx.moveTo(...s); }); ctx.closePath(); ctx.stroke(); ctx.setLineDash([]);
    }
    if (hoverEdge && hover < 0) { ctx.fillStyle = '#60a5fa'; const s = toS(hoverEdge.p); ctx.beginPath(); ctx.arc(s[0], s[1], 4 * dpr, 0, 7); ctx.fill(); }
    pts.forEach((p, i) => {
      const s = toS(p);
      ctx.fillStyle = i === sel ? '#f59e0b' : i === hover ? '#60a5fa' : i === 0 ? '#22c55e' : '#e2e8f0';
      ctx.strokeStyle = '#0f172a'; ctx.lineWidth = 1.5 * dpr;
      ctx.beginPath(); ctx.arc(s[0], s[1], (i === hover || i === sel ? 6 : 4.5) * dpr, 0, 7); ctx.fill(); ctx.stroke();
    });
    if (cursor) coords.textContent = `X ${round(cursor[0], 2)}  Y ${round(cursor[1], 2)} mm  ·  ${pts.length} points`;
  }

  const evtW = (e) => { const r = canvas.getBoundingClientRect(); return [(e.clientX - r.left) * devicePixelRatio, (e.clientY - r.top) * devicePixelRatio]; };
  const nearPt = (sx, sy) => { let b = -1, bd = 10 * devicePixelRatio; pts.forEach((p, i) => { const s = toS(p); const d = Math.hypot(s[0] - sx, s[1] - sy); if (d < bd) { bd = d; b = i; } }); return b; };
  const nearEdge = (sx, sy) => {
    let best = null;
    for (let i = 0; i < pts.length && pts.length > 1; i++) {
      const a = toS(pts[i]), b = toS(pts[(i + 1) % pts.length]);
      const dx = b[0] - a[0], dy = b[1] - a[1], l2 = dx * dx + dy * dy || 1;
      const t = Math.max(0, Math.min(1, ((sx - a[0]) * dx + (sy - a[1]) * dy) / l2));
      const d = Math.hypot(sx - a[0] - t * dx, sy - a[1] - t * dy);
      if (d < 8 * devicePixelRatio && (!best || d < best.d)) best = { d, i, p: toW(a[0] + t * dx, a[1] + t * dy) };
    }
    return best;
  };
  let pan = null;
  canvas.addEventListener('pointerdown', (e) => {
    canvas.setPointerCapture(e.pointerId);
    const [sx, sy] = evtW(e);
    if (e.button === 1) { pan = { sx, sy, cx: view.cx, cy: view.cy }; return; }
    const i = nearPt(sx, sy);
    if (e.button === 2) { if (i >= 0) { pts.splice(i, 1); sel = -1; updatePtEdit(); draw(); } return; }
    if (i >= 0) { dragging = i; sel = i; updatePtEdit(); draw(); return; }
    const ed = nearEdge(sx, sy);
    const p = snapP(toW(sx, sy));
    if (ed) { pts.splice(ed.i + 1, 0, snapP(ed.p)); sel = dragging = ed.i + 1; }
    else { pts.push(p); sel = dragging = pts.length - 1; }
    updatePtEdit(); draw();
  });
  canvas.addEventListener('pointermove', (e) => {
    const [sx, sy] = evtW(e);
    cursor = toW(sx, sy);
    if (pan) { view.cx = pan.cx - (sx - pan.sx) / view.s; view.cy = pan.cy + (sy - pan.sy) / view.s; draw(); return; }
    if (dragging >= 0) { pts[dragging] = snapP(cursor); if (mode === 'Revolve') pts[dragging][0] = Math.max(0, pts[dragging][0]); draw(); return; }
    hover = nearPt(sx, sy); hoverEdge = hover < 0 ? nearEdge(sx, sy) : null;
    draw();
  });
  canvas.addEventListener('pointerup', () => { if (dragging >= 0) updatePtEdit(); dragging = -1; pan = null; });
  canvas.addEventListener('contextmenu', (e) => e.preventDefault());
  canvas.addEventListener('wheel', (e) => {
    e.preventDefault();
    const [sx, sy] = evtW(e), before = toW(sx, sy);
    view.s = Math.max(0.5, Math.min(200, view.s * Math.exp(-e.deltaY * 0.0015)));
    const after = toW(sx, sy);
    view.cx += before[0] - after[0]; view.cy += before[1] - after[1];
    draw();
  }, { passive: false });
  requestAnimationFrame(() => { fit(); draw(); });
}
