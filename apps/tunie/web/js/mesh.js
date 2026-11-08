/* Tunie — mesh.js: bed-level mesh viewer with a coloured 3D heightmap and
   suggested levelling-screw turns. Reads Marlin (G29 / M420 V), Klipper
   (BED_MESH_OUTPUT or Moonraker) and RepRapFirmware (heightmap.csv).
   Part of the nrmluvdev maker suite. Created by nrmluvdev. */
'use strict';

const THREADS = { M3: 0.5, M4: 0.7, M5: 0.8, M6: 1.0 };

function parseMesh(text) {
  const t = String(text || '').trim();
  if (!t) return null;
  // Moonraker JSON
  if (t[0] === '{') {
    try {
      const j = JSON.parse(t);
      const bm = j?.result?.status?.bed_mesh || j.bed_mesh || j;
      const mat = bm.probed_matrix || bm.mesh_matrix;
      if (mat && mat.length && mat[0].length) return { rows: mat, min: bm.mesh_min, max: bm.mesh_max, source: 'Klipper (Moonraker)', flipped: false };
    } catch { }
  }
  // RepRapFirmware heightmap.csv
  if (/height\s*map/i.test(t)) {
    const lines = t.split(/\r?\n/);
    const hi = lines.findIndex((l) => /xmin/i.test(l));
    if (hi >= 0) {
      const head = lines[hi].split(',').map((s) => s.trim().toLowerCase());
      const vals = lines[hi + 1].split(',').map(Number);
      const g = (k) => vals[head.indexOf(k)];
      const rows = lines.slice(hi + 2).map((l) => l.split(',').map((x) => x.trim() === '0' && false ? NaN : Number(x))).filter((r) => r.length > 1 && r.every((v) => isFinite(v)));
      if (rows.length) return { rows, min: [g('xmin') ?? g('min0'), g('ymin') ?? g('min1')], max: [g('xmax') ?? g('max0'), g('ymax') ?? g('max1')], source: 'RepRapFirmware heightmap', flipped: false };
    }
  }
  let min = null, max = null;
  const mm = t.match(/Min:\s*\(?\s*([-\d.]+)\s*,\s*([-\d.]+)\)?.*?Max:\s*\(?\s*([-\d.]+)\s*,\s*([-\d.]+)/is);
  if (mm) { min = [+mm[1], +mm[2]]; max = [+mm[3], +mm[4]]; }
  const isKlipper = /probed z positions|mesh leveling/i.test(t);
  const isUBL = /\[\s*[-+]?\d*\.\d+\s*\]|Unified Bed Leveling|UBL/i.test(t);
  const groups = [];
  let cur = [];
  for (const raw of t.split(/\r?\n/)) {
    const line = raw.replace(/^\/\/\s?/, '').replace(/[[\]()|]/g, ' ');
    const toks = line.trim().split(/[\s,]+/).filter(Boolean);
    let nums = toks.map((x) => /^[-+]?(\d+\.?\d*|\.\d+)$/.test(x) ? parseFloat(x) : NaN);
    if (nums.some((v) => isNaN(v)) || !toks.some((x) => x.includes('.'))) { if (cur.length) { groups.push(cur); cur = []; } continue; }
    if (toks.length > 2 && !toks[0].includes('.') && toks.slice(1).every((x) => x.includes('.'))) nums = nums.slice(1);
    if (nums.length < 3) { if (cur.length) { groups.push(cur); cur = []; } continue; }
    if (cur.length && cur[0].length !== nums.length) { groups.push(cur); cur = []; }
    cur.push(nums);
  }
  if (cur.length) groups.push(cur);
  const g = groups.find((x) => x.length >= 3) || groups.sort((a, b) => b.length - a.length)[0];
  if (!g || g.length < 2) return null;
  return { rows: g, min, max, source: isKlipper ? 'Klipper' : isUBL ? 'Marlin UBL' : 'Marlin', flipped: isUBL };
}

function meshStats(rows) {
  const v = rows.flat().filter(isFinite);
  const mn = Math.min(...v), mx = Math.max(...v), mean = avg(v);
  const sd = Math.sqrt(avg(v.map((x) => (x - mean) ** 2)));
  return { min: mn, max: mx, range: mx - mn, mean, sd };
}

function meshAt(rows, min, max, x, y) {
  const ny = rows.length, nx = rows[0].length;
  const fx = clamp((x - min[0]) / (max[0] - min[0]) * (nx - 1), 0, nx - 1);
  const fy = clamp((y - min[1]) / (max[1] - min[1]) * (ny - 1), 0, ny - 1);
  const x0 = Math.floor(fx), y0 = Math.floor(fy), x1 = Math.min(nx - 1, x0 + 1), y1 = Math.min(ny - 1, y0 + 1);
  const tx = fx - x0, ty = fy - y0;
  const a = rows[y0][x0] * (1 - tx) + rows[y0][x1] * tx;
  const b = rows[y1][x0] * (1 - tx) + rows[y1][x1] * tx;
  return a * (1 - ty) + b * ty;
}

function heatColor(t) { // 0..1 → blue → cyan → green → amber → red
  const stops = [[59, 130, 246], [34, 211, 238], [52, 211, 153], [251, 191, 36], [248, 113, 113]];
  t = clamp(t, 0, 1) * (stops.length - 1);
  const i = Math.min(stops.length - 2, Math.floor(t)), f = t - i;
  return stops[i].map((c, k) => Math.round(c + (stops[i + 1][k] - c) * f));
}

function renderMesh() {
  const p = DB.printer();
  p.mesh = p.mesh || {};
  const M = p.meshData || null;
  const screws = p.screws || { thread: 'M3', cw: 'lowers', ref: 0, points: null };
  view(`<div class="page-head"><div><h1>Bed mesh viewer</h1><div class="sub">See how flat your bed is as a coloured 3D map, and get suggested turns for the levelling screws.</div></div>
    <div class="actions"><button class="btn" id="m-read" ${CONN.connected ? '' : 'disabled title="Connect to your printer first"'}>${icon('download', 16)}Read from printer</button><button class="btn primary" id="m-probe" ${CONN.connected ? '' : 'disabled'}>${icon('play', 16)}Probe new mesh</button></div></div>
    <div class="grid" style="grid-template-columns:minmax(0,1.6fr) minmax(0,1fr);gap:18px;align-items:start">
      <div class="col">
        <div class="card" style="padding:0;overflow:hidden"><div class="meshwrap" style="position:relative">${M ? '<canvas id="mcv"></canvas>' : `<div class="empty" style="padding:90px 20px">${DG.mesh()}<h3>No mesh yet</h3><p>Paste your printer's mesh output on the right, or read it straight from the printer.</p></div>`}
          ${M ? `<div class="ov" style="position:absolute;left:14px;top:12px;display:flex;gap:10px;align-items:center"><span class="chip">${esc(M.source || '')}</span><span class="chip">${M.rows[0].length} × ${M.rows.length} points</span></div>
          <div style="position:absolute;left:14px;right:14px;bottom:12px;display:flex;gap:14px;align-items:center;background:rgba(14,17,24,.8);border:1px solid var(--line);border-radius:10px;padding:8px 12px">
            <span class="small muted">Height ×</span><input type="range" id="mex" min="20" max="400" value="${DB.data.ui.meshEx || 120}"><label class="check small"><input type="checkbox" id="mflip" ${M.flipped ? 'checked' : ''}>Rows start at back</label></div>` : ''}</div></div>
        ${M ? (() => {
          const st = meshStats(M.rows);
          const verdict = st.range < 0.1 ? ['Excellent', 'good', 'Flatter than most — nothing to do.'] : st.range < 0.2 ? ['Good', 'good', 'A bed mesh easily handles this.'] : st.range < 0.4 ? ['Fair', 'warn', 'Levelling the screws first will help the mesh do its job.'] : ['Needs levelling', 'bad', 'Level with the screws (below), then probe again.'];
          return `<div class="grid g4">
            <div class="card stat hero"><div class="lbl">Range</div><div class="val">${fmt(st.range, 3)}<small>mm</small></div><div class="sub ${verdict[1]}">${verdict[0]}</div></div>
            <div class="card stat"><div class="lbl">Lowest</div><div class="val" style="font-size:24px">${fmt(st.min, 3)}</div></div>
            <div class="card stat"><div class="lbl">Highest</div><div class="val" style="font-size:24px">${fmt(st.max, 3)}</div></div>
            <div class="card stat"><div class="lbl">Std. dev.</div><div class="val" style="font-size:24px">${fmt(st.sd, 3)}</div></div></div>
            <div class="callout ${verdict[1] === 'good' ? 'good' : verdict[1] === 'warn' ? 'warn' : 'bad'}">${icon('info')}<div><p>${verdict[2]}</p></div></div>
            <div class="row"><span class="small muted">${fmt(st.min, 2)}</span><div class="scale-bar" style="flex:1"></div><span class="small muted">${fmt(st.max, 2)} mm</span></div>`;
        })() : ''}
      </div>
      <div class="col">
        <div class="card"><h3>${icon('copy', 16)}Paste mesh data</h3><div class="card-sub">Marlin: <span class="mono">M420 V</span> or <span class="mono">G29 T</span> · Klipper: <span class="mono">BED_MESH_OUTPUT</span> · RRF: contents of <span class="mono">heightmap.csv</span></div>
          <textarea class="ta" id="mtxt" style="min-height:130px" placeholder="Paste here…"></textarea><div class="row mt-s"><button class="btn sm primary" id="mparse">${icon('wand', 14)}Show heightmap</button></div></div>
        <div class="card"><h3>${icon('grid', 16)}Mesh area</h3><div class="card-sub">Where the first and last probe points are on the bed (from your firmware's mesh settings).</div>
          <div class="fields" id="marea">${fieldHTML({ k: 'x0', label: 'Min X', unit: 'mm' }, M?.min?.[0] ?? p.mesh.min?.[0] ?? round(p.bedX * 0.1, 0))}${fieldHTML({ k: 'y0', label: 'Min Y', unit: 'mm' }, M?.min?.[1] ?? p.mesh.min?.[1] ?? round(p.bedY * 0.1, 0))}
          ${fieldHTML({ k: 'x1', label: 'Max X', unit: 'mm' }, M?.max?.[0] ?? p.mesh.max?.[0] ?? round(p.bedX * 0.9, 0))}${fieldHTML({ k: 'y1', label: 'Max Y', unit: 'mm' }, M?.max?.[1] ?? p.mesh.max?.[1] ?? round(p.bedY * 0.9, 0))}</div></div>
        ${M ? `<div class="card"><h3>${icon('gear', 16)}Levelling screws</h3><div class="card-sub">Turn each knob by the amount shown, then probe again. Test the direction first: turn a knob clockwise a little and watch which way that corner moves.</div>
          <div class="fields">${fieldHTML({ k: 'thread', label: 'Screw thread', type: 'select', options: Object.keys(THREADS).map((k) => [k, `${k} (${THREADS[k]} mm/turn)`]) }, screws.thread)}
          ${fieldHTML({ k: 'cw', label: 'Clockwise moves the bed', type: 'select', options: [['lowers', 'Down (away from nozzle)'], ['raises', 'Up (towards nozzle)']] }, screws.cw)}</div>
          <div id="screws" class="col mt"></div>
          <div class="row mt-s"><button class="btn sm ghost" id="sedit">${icon('edit', 14)}Edit screw positions</button></div></div>` : ''}
      </div></div>`, 'Bed mesh');

  $('#mparse').onclick = () => {
    const m = parseMesh($('#mtxt').value);
    if (!m) return toast("Couldn't find a mesh in that text", 'err');
    setMesh(m);
  };
  const setMesh = (m) => {
    if (!m.rows || !m.rows.length) return;
    if (m.flipped) { m.rows = m.rows.slice().reverse(); }
    m.flipped = false;
    p.meshData = { rows: m.rows, source: m.source, flipped: false, date: Date.now() };
    if (m.min && m.max) { p.mesh.min = m.min; p.mesh.max = m.max; p.meshData.min = m.min; p.meshData.max = m.max; }
    DB.save(); renderMesh();
  };
  const readArea = () => { const a = readFields($('#marea')); p.mesh.min = [a.x0, a.y0]; p.mesh.max = [a.x1, a.y1]; if (p.meshData) { p.meshData.min = p.mesh.min; p.meshData.max = p.mesh.max; } DB.save(); };
  $$('#marea [data-k]').forEach((el) => el.addEventListener('change', () => { readArea(); renderMesh(); }));

  $('#m-read').onclick = async () => {
    try {
      if (CONN.kind.startsWith('moonraker')) {
        const r = await api.get('/api/conn/query?what=mesh');
        const m = parseMesh(JSON.stringify(r));
        if (!m) return toast('No mesh is loaded on the printer — probe one first', 'err');
        setMesh(m);
      } else {
        const r = await sendLines([p.firmware === 'klipper' ? 'BED_MESH_OUTPUT' : 'M420 V']);
        const m = parseMesh((r.lines || []).join('\n'));
        if (!m) return toast('The printer did not send a mesh — try "Probe new mesh"', 'err');
        setMesh(m);
      }
    } catch (e) { toast(e.message, 'err'); }
  };
  $('#m-probe').onclick = async () => {
    if (!(await confirmBox('Probe a new mesh?', 'The printer will home and probe the bed. Make sure the bed is clear and at printing temperature for the most accurate result.', 'Start probing'))) return;
    const b = $('#m-probe'); b.disabled = true; b.innerHTML = '<span class="spin"></span>Probing…';
    try {
      const fw = p.firmware;
      const cmds = fw === 'klipper' ? ['G28', 'BED_MESH_CALIBRATE'] : fw === 'rrf' ? ['G28', 'G29 S0'] : fw === 'prusa-mk3' ? ['G28 W', 'G80', 'G81'] : ['G28', 'G29', 'M420 V'];
      const r = await sendLines(cmds);
      if (CONN.kind.startsWith('moonraker')) { const q = await api.get('/api/conn/query?what=mesh'); const m = parseMesh(JSON.stringify(q)); if (m) return setMesh(m); }
      const m = parseMesh((r.lines || []).join('\n'));
      if (m) setMesh(m); else toast('Probed — but no mesh text came back. Try "Read from printer".', 'info');
    } catch (e) { toast(e.message, 'err'); }
    b.disabled = false; b.innerHTML = icon('play', 16) + 'Probe new mesh';
  };

  if (!M) return;
  const min = M.min || p.mesh.min || [p.bedX * 0.1, p.bedY * 0.1];
  const max = M.max || p.mesh.max || [p.bedX * 0.9, p.bedY * 0.9];
  // screws
  const defaultScrews = () => {
    const ix = Math.min(30, p.bedX * 0.15), iy = Math.min(30, p.bedY * 0.15);
    return [[ix, iy, 'Front left'], [p.bedX - ix, iy, 'Front right'], [p.bedX - ix, p.bedY - iy, 'Back right'], [ix, p.bedY - iy, 'Back left']];
  };
  const pts = screws.points || defaultScrews();
  const drawScrews = () => {
    const sc = readFields($('#screws').parentElement);
    screws.thread = sc.thread || screws.thread; screws.cw = sc.cw || screws.cw;
    p.screws = Object.assign(screws, { points: pts }); DB.save();
    const pitch = THREADS[screws.thread] || 0.5;
    const zs = pts.map(([x, y]) => meshAt(M.rows, min, max, x, y));
    const ref = screws.ref || 0;
    $('#screws').innerHTML = pts.map(([x, y, name], i) => {
      const d = zs[i] - zs[ref];
      if (i === ref) return `<div class="list-item sel"><div class="grow"><b>${esc(name)}</b><span>Reference screw — leave as is</span></div><span class="chip acc">base</span></div>`;
      const turns = Math.abs(d) / pitch;
      const needDown = d > 0; // this corner is high → lower it
      const cw = (needDown && screws.cw === 'lowers') || (!needDown && screws.cw === 'raises');
      const mins = Math.round(turns * 60);
      const frac = turns < 0.06 ? 'nothing' : turns < 0.19 ? '⅛ turn' : turns < 0.31 ? '¼ turn' : turns < 0.44 ? '⅜ turn' : turns < 0.6 ? '½ turn' : turns < 0.85 ? '¾ turn' : fmt(turns, 1) + ' turns';
      const ok = turns < 0.06;
      return `<div class="list-item"><div class="grow"><b>${esc(name)}</b><span>${ok ? 'Already level with the reference' : `${needDown ? 'High' : 'Low'} by ${fmt(Math.abs(d), 3)} mm — turn ${cw ? 'clockwise ↻' : 'anticlockwise ↺'} ${frac}`}</span></div>
        <span class="chip ${ok ? 'ok' : 'wn'} mono">${ok ? '✓' : `${cw ? 'CW' : 'CCW'} ${String(Math.floor(mins / 60)).padStart(2, '0')}:${String(mins % 60).padStart(2, '0')}`}</span>
        <button class="btn sm ghost" data-ref="${i}" title="Use as reference">${icon('star', 14)}</button></div>`;
    }).join('') + '<p class="small dim" style="margin:4px 0 0">Times are like a clock face: 00:15 = a quarter turn. Then probe again and repeat until all are ✓.</p>';
    $$('[data-ref]').forEach((b) => b.onclick = () => { screws.ref = +b.dataset.ref; drawScrews(); });
  };
  $$('[data-k="thread"],[data-k="cw"]').forEach((el) => el.addEventListener('change', drawScrews));
  drawScrews();
  $('#sedit').onclick = async () => {
    const body = pts.map(([x, y, n], i) => `<div class="fields" style="grid-template-columns:2fr 1fr 1fr;margin-bottom:8px">${fieldHTML({ k: 'n' + i, label: 'Name', type: 'text' }, n)}${fieldHTML({ k: 'x' + i, label: 'X', unit: 'mm' }, x)}${fieldHTML({ k: 'y' + i, label: 'Y', unit: 'mm' }, y)}</div>`).join('');
    const r = await modal({ title: 'Screw positions', body: `<p class="muted">Measured from the front-left corner of the bed (X to the right, Y to the back).</p>${body}<button class="btn sm" id="addscrew">${icon('plus', 14)}Add a screw</button>`, actions: [{ label: 'Reset', kind: 'ghost', value: () => 'reset' }, { label: 'Save', kind: 'primary', value: (bg) => readFields(bg) }],
      onOpen: (bg) => { $('#addscrew', bg).onclick = () => { pts.push([p.bedX / 2, p.bedY - 20, 'Back middle']); bg.remove(); $('#sedit').click(); }; } });
    if (r === 'reset') { screws.points = null; p.screws = screws; DB.save(); return renderMesh(); }
    if (!r) return;
    pts.forEach((pt, i) => { pt[0] = r['x' + i]; pt[1] = r['y' + i]; pt[2] = r['n' + i]; });
    screws.points = pts; p.screws = screws; DB.save(); renderMesh();
  };
  $('#mflip').onchange = () => { M.rows = M.rows.slice().reverse(); DB.save(); draw(); drawScrews(); };

  // 3D heightmap
  const cv = $('#mcv');
  let yaw = -0.65, pitch = 0.9;
  const st = meshStats(M.rows);
  const ro = new ResizeObserver(() => draw());
  const draw = () => {
    if (!cv.isConnected || !$('#mex')) { ro.disconnect(); return; }
    const ex = +$('#mex').value;
    DB.data.ui.meshEx = ex;
    const dpr = devicePixelRatio || 1, W = cv.clientWidth, H = cv.clientHeight;
    cv.width = W * dpr; cv.height = H * dpr;
    const ctx = cv.getContext('2d');
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    ctx.clearRect(0, 0, W, H);
    const rows = M.rows, ny = rows.length, nx = rows[0].length;
    const bw = p.bedX, bh = p.bedY;
    const s = Math.min(W, H) / (Math.max(bw, bh) * 1.35);
    const cy = Math.cos(yaw), sy = Math.sin(yaw), cp = Math.cos(pitch), sp = Math.sin(pitch);
    const P = (x, y, z) => {
      const X = x - bw / 2, Y = y - bh / 2;
      const rx = X * cy - Y * sy, ry = X * sy + Y * cy;
      return [W / 2 + rx * s, H / 2 - 10 + (-ry * cp - z * ex * sp) * s];
    };
    // bed plate
    ctx.fillStyle = 'rgba(255,255,255,.03)'; ctx.strokeStyle = 'rgba(140,150,180,.3)';
    ctx.beginPath(); [[0, 0], [bw, 0], [bw, bh], [0, bh]].forEach(([x, y], i) => { const q = P(x, y, st.min - 0.02); i ? ctx.lineTo(...q) : ctx.moveTo(...q); }); ctx.closePath(); ctx.fill(); ctx.stroke();
    const lab = (x, y, t) => { const q = P(x, y, st.min - 0.02); ctx.fillStyle = '#8d95aa'; ctx.font = '600 12px Segoe UI, sans-serif'; ctx.textAlign = 'center'; ctx.fillText(t, q[0], q[1]); };
    lab(bw / 2, -bh * 0.06, 'FRONT'); lab(bw / 2, bh * 1.06, 'BACK');
    const gx = (i) => min[0] + (max[0] - min[0]) * i / (nx - 1), gy = (j) => min[1] + (max[1] - min[1]) * j / (ny - 1);
    // quads sorted back to front
    const quads = [];
    for (let j = 0; j < ny - 1; j++) for (let i = 0; i < nx - 1; i++) {
      const zc = (rows[j][i] + rows[j][i + 1] + rows[j + 1][i] + rows[j + 1][i + 1]) / 4;
      const c = P(gx(i + .5), gy(j + .5), zc);
      quads.push({ i, j, zc, d: c[1] });
    }
    quads.sort((a, b) => a.d - b.d);
    for (const q of quads) {
      const { i, j } = q;
      const pts = [[i, j], [i + 1, j], [i + 1, j + 1], [i, j + 1]].map(([a, b]) => P(gx(a), gy(b), rows[b][a] - st.mean));
      const col = heatColor((q.zc - st.min) / (st.range || 1));
      ctx.beginPath(); pts.forEach((pt, k) => k ? ctx.lineTo(...pt) : ctx.moveTo(...pt)); ctx.closePath();
      ctx.fillStyle = `rgba(${col.join(',')},.88)`; ctx.fill();
      ctx.strokeStyle = 'rgba(10,12,17,.55)'; ctx.lineWidth = 1; ctx.stroke();
    }
    // probe points
    for (let j = 0; j < ny; j++) for (let i = 0; i < nx; i++) {
      const q = P(gx(i), gy(j), rows[j][i] - st.mean);
      ctx.fillStyle = '#e8ebf2'; ctx.beginPath(); ctx.arc(q[0], q[1], 2.2, 0, 7); ctx.fill();
    }
    // screws
    pts.forEach(([x, y, n]) => {
      const q = P(x, y, st.min - 0.02);
      ctx.strokeStyle = '#a78bfa'; ctx.lineWidth = 2; ctx.beginPath(); ctx.arc(q[0], q[1], 6, 0, 7); ctx.stroke();
    });
  };
  let drag = null;
  cv.addEventListener('pointerdown', (e) => { drag = { x: e.clientX, y: e.clientY, yaw, pitch }; cv.setPointerCapture(e.pointerId); });
  cv.addEventListener('pointermove', (e) => { if (!drag) return; yaw = drag.yaw + (e.clientX - drag.x) * 0.008; pitch = clamp(drag.pitch + (e.clientY - drag.y) * 0.006, 0.2, 1.5); draw(); });
  cv.addEventListener('pointerup', () => drag = null);
  $('#mex').oninput = () => { draw(); };
  $('#mex').onchange = () => DB.save();
  ro.observe(cv);
  draw();
}
