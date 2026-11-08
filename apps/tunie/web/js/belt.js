/* Tunie — belt.js: belt tension helper. Listens through the PC microphone,
   finds the belt's note when you pluck it, and compares belts / targets.
   Physics: f = (1 / 2L) · √(T / μ)  (a plucked string)
   Part of the nrmluvdev maker suite. Created by nrmluvdev. */
'use strict';

const BELT = { ctx: null, an: null, stream: null, raf: 0, peakHold: 0, holdT: 0, hist: [] };

const BELT_TARGETS = [
  { id: 'generic', name: 'Typical 6 mm GT2 (most printers)', f: [90, 110], note: 'A commonly used range when plucked at the middle of the longest free span. Your printer maker\'s guide wins if it gives a number.' },
  { id: 'voron', name: 'Voron 2.4 / Trident (150 mm span)', f: [105, 115], note: 'Voron guides use about 110 Hz over a 150 mm span — move the toolhead so the span is 150 mm.' },
  { id: 'corexy-match', name: 'CoreXY: just match A and B', f: null, note: 'On CoreXY printers the two belts matching (within ~3 Hz) matters more than the exact value.' },
  { id: 'custom', name: 'Custom: from tension and span', f: null, note: 'Uses the formula with your belt\'s mass per metre.' },
];

function renderBelt() {
  const p = DB.printer();
  const bs = Object.assign({ target: 'generic', span: 150, mu: 0.0083, tension: 20, readings: {} }, p.belt || {});
  p.belt = bs;
  view(`<div class="page-head"><div><h1>Belt tension helper</h1><div class="sub">Pluck a belt like a guitar string — Tunie listens through your microphone and shows its note. Tight enough, and the same on both sides, is what you\'re after.</div></div></div>
    <div class="grid" style="grid-template-columns:minmax(0,1.4fr) minmax(0,1fr);gap:18px;align-items:start">
      <div class="col">
        <div class="card pad-lg"><div class="row" style="justify-content:space-between;align-items:flex-start">
          <div><div class="lbl muted small" style="font-weight:600;text-transform:uppercase;letter-spacing:.06em">Detected</div><div class="freq" id="bf"><span style="-webkit-text-fill-color:var(--dim)">--.-</span> <span style="font-size:24px">Hz</span></div><div class="muted small" id="bt">Press Start, then pluck the belt</div></div>
          <div class="col" style="align-items:flex-end"><button class="btn primary lg" id="bstart">${icon('mic', 18)}Start listening</button><span class="small dim" id="bdev"></span></div></div>
          <canvas id="bcv" style="width:100%;height:200px;display:block;margin-top:14px;border-radius:10px;background:#0b0e14;border:1px solid var(--line)"></canvas>
          <div class="row mt"><span class="small muted">Save this reading as:</span>${['X / A belt', 'Y / B belt', 'Z belt'].map((n, i) => `<button class="btn sm" data-save="${i}">${icon('save', 14)}${n}</button>`).join('')}</div>
        </div>
        <div class="grid g3" id="breads"></div>
      </div>
      <div class="col">
        <div class="card"><h3>${icon('star', 16)}Target</h3>
          <div class="fields" id="bset">${fieldHTML({ k: 'target', label: 'Guide', type: 'select', options: BELT_TARGETS.map((t) => [t.id, t.name]) }, bs.target)}
          ${fieldHTML({ k: 'span', label: 'Free span length', unit: 'mm', help: 'Distance between the two points the belt touches.' }, bs.span)}
          ${fieldHTML({ k: 'mu', label: 'Belt mass per metre', unit: 'kg/m', step: 0.0001, help: '6 mm GT2 ≈ 0.0083 · 9 mm GT2 ≈ 0.0125 (varies by brand)' }, bs.mu)}
          ${fieldHTML({ k: 'tension', label: 'Custom tension', unit: 'N' }, bs.tension)}</div>
          <div id="btarget" class="mt"></div></div>
        <div class="card"><div class="diagram" style="border:0;padding:0;background:none">${DG.belt()}</div>
          <ol class="muted small" style="padding-left:18px;margin:10px 0 0;line-height:1.7">
            <li>Turn the motors off (or let the printer sit idle) and move the head so the span matches the length above.</li>
            <li>Hold the microphone (or laptop) close to the belt in a quiet room.</li>
            <li>Pluck the belt in the middle of the span. The big number freezes on the strongest note for a moment.</li>
            <li>Adjust the tensioner and pluck again. On CoreXY printers, get both belts within ~3 Hz.</li></ol></div>
        <div class="callout warn">${icon('alert')}<div><p>Over-tight belts wear bearings and motors. If in doubt, stay at the low end of the range — matching both sides matters most.</p></div></div>
      </div></div>`, 'Belt tension');

  const setRead = () => {
    const v = readFields($('#bset'));
    Object.assign(bs, v); DB.save();
    const t = BELT_TARGETS.find((x) => x.id === bs.target);
    const L = bs.span / 1000;
    const fromT = (T) => (1 / (2 * L)) * Math.sqrt(T / bs.mu);
    let range = t.f;
    if (t.id === 'custom') { const f = fromT(bs.tension); range = [f * 0.97, f * 1.03]; }
    bs.range = range;
    $('#btarget').innerHTML = `${range ? `<div class="row"><span class="chip acc">${fmt(range[0], 0)}–${fmt(range[1], 0)} Hz</span><span class="small muted">≈ ${fmt(bs.mu * (2 * L * range[0]) ** 2, 1)}–${fmt(bs.mu * (2 * L * range[1]) ** 2, 1)} N tension</span></div>` : ''}<p class="small muted mt-s" style="margin-bottom:0">${esc(t.note)}</p>`;
    drawReads();
  };
  const drawReads = () => {
    const names = ['X / A belt', 'Y / B belt', 'Z belt'];
    const L = bs.span / 1000;
    const r = bs.readings || {};
    const vals = [0, 1].map((i) => r[i]).filter(Boolean);
    const diff = vals.length === 2 ? Math.abs(vals[0] - vals[1]) : null;
    $('#breads').innerHTML = names.map((n, i) => {
      const f = r[i];
      if (!f) return `<div class="card stat"><div class="lbl">${n}</div><div class="val" style="font-size:22px;-webkit-text-fill-color:var(--dim)">--.-</div><div class="sub">not measured</div></div>`;
      const inR = bs.range ? (f < bs.range[0] ? 'low' : f > bs.range[1] ? 'high' : 'ok') : 'ok';
      return `<div class="card stat ${inR === 'ok' ? 'hero' : ''}"><div class="lbl">${n}</div><div class="val" style="font-size:26px">${fmt(f, 1)}<small>Hz</small></div><div class="sub ${inR === 'ok' ? 'good' : 'warn'}">${inR === 'ok' ? 'in range' : inR === 'low' ? 'too loose — tighten' : 'too tight — loosen'} · ≈${fmt(bs.mu * (2 * L * f) ** 2, 1)} N</div></div>`;
    }).join('') + (diff !== null ? `<div class="callout ${diff <= 3 ? 'good' : 'warn'} span3">${icon(diff <= 3 ? 'check' : 'alert')}<div><p>X/A and Y/B differ by <b>${fmt(diff, 1)} Hz</b>. ${diff <= 3 ? 'Nicely matched.' : 'Bring the looser one up (or the tighter one down) until they match within ~3 Hz.'}</p></div></div>` : '');
  };
  $$('#bset [data-k]').forEach((el) => el.addEventListener('change', setRead));
  setRead();
  $$('[data-save]').forEach((b) => b.onclick = () => {
    if (!BELT.peakHold) return toast('Pluck the belt first', 'info');
    bs.readings = bs.readings || {}; bs.readings[b.dataset.save] = round(BELT.peakHold, 1); DB.save(); drawReads(); toast('Saved reading');
  });
  $('#bstart').onclick = () => BELT.stream ? beltStop() : beltStart();
  if (BELT.stream) { $('#bstart').innerHTML = icon('stop', 18) + 'Stop'; beltLoop(); }
}

async function beltStart() {
  try {
    BELT.stream = await navigator.mediaDevices.getUserMedia({ audio: { echoCancellation: false, noiseSuppression: false, autoGainControl: false } });
  } catch (e) { BELT.stream = null; return toast('Microphone not available: ' + e.message, 'err', 5000); }
  BELT.ctx = new (window.AudioContext || window.webkitAudioContext)();
  const src = BELT.ctx.createMediaStreamSource(BELT.stream);
  const hp = BELT.ctx.createBiquadFilter(); hp.type = 'highpass'; hp.frequency.value = 30;
  BELT.an = BELT.ctx.createAnalyser(); BELT.an.fftSize = 32768; BELT.an.smoothingTimeConstant = 0.2;
  src.connect(hp); hp.connect(BELT.an);
  const tr = BELT.stream.getAudioTracks()[0];
  const d = $('#bdev'); if (d) d.textContent = tr ? tr.label : '';
  const b = $('#bstart'); if (b) b.innerHTML = icon('stop', 18) + 'Stop';
  beltLoop();
}
function beltStop() {
  cancelAnimationFrame(BELT.raf);
  if (BELT.stream) BELT.stream.getTracks().forEach((t) => t.stop());
  if (BELT.ctx) BELT.ctx.close();
  BELT.stream = null; BELT.ctx = null;
  const b = $('#bstart'); if (b) b.innerHTML = icon('mic', 18) + 'Start listening';
}
function beltLoop() {
  const cv = $('#bcv');
  if (!cv || !BELT.an) { if (!cv) beltStop(); return; }
  const an = BELT.an, sr = BELT.ctx.sampleRate;
  const N = an.frequencyBinCount, buf = new Float32Array(N);
  an.getFloatFrequencyData(buf);
  const binHz = sr / an.fftSize;
  const lo = Math.floor(40 / binHz), hi = Math.ceil(400 / binHz);
  let best = lo, bv = -Infinity, sum = 0;
  for (let i = lo; i <= hi; i++) { sum += buf[i]; if (buf[i] > bv) { bv = buf[i]; best = i; } }
  const mean = sum / (hi - lo + 1);
  // parabolic interpolation for sub-bin accuracy
  const a = buf[best - 1], bb = buf[best], c = buf[best + 1];
  const off = (a - c) / (2 * (a - 2 * bb + c)) || 0;
  const f = (best + clamp(off, -0.5, 0.5)) * binHz;
  const strong = bv - mean > 25 && bv > -75;
  const now = performance.now();
  if (strong && (bv > BELT.holdV - 6 || now - BELT.holdT > 1500 || !BELT.peakHold)) { BELT.peakHold = f; BELT.holdV = bv; BELT.holdT = now; }
  if (now - BELT.holdT > 2500) BELT.holdV = -Infinity;
  const fEl = $('#bf'), tEl = $('#bt');
  if (fEl && BELT.peakHold) fEl.innerHTML = `${fmt(BELT.peakHold, 1)} <span style="font-size:24px">Hz</span>`;
  if (tEl) tEl.textContent = strong ? 'Hearing a clear note…' : BELT.peakHold ? 'Holding the last clear note' : 'Listening — pluck the belt';
  // spectrum
  const dpr = devicePixelRatio || 1, W = cv.clientWidth, H = cv.clientHeight;
  cv.width = W * dpr; cv.height = H * dpr;
  const ctx = cv.getContext('2d'); ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
  const range = DB.printer().belt?.range;
  const xOf = (hz) => (hz - 40) / 360 * W;
  if (range) { ctx.fillStyle = 'rgba(52,211,153,.10)'; ctx.fillRect(xOf(range[0]), 0, xOf(range[1]) - xOf(range[0]), H); }
  ctx.strokeStyle = 'rgba(140,150,180,.12)'; ctx.fillStyle = '#5f677c'; ctx.font = '11px Segoe UI'; ctx.lineWidth = 1;
  for (let hz = 50; hz <= 400; hz += 50) { const x = xOf(hz); ctx.beginPath(); ctx.moveTo(x, 0); ctx.lineTo(x, H); ctx.stroke(); ctx.fillText(hz + ' Hz', x + 3, H - 6); }
  const g = ctx.createLinearGradient(0, 0, W, 0); g.addColorStop(0, '#7c5cff'); g.addColorStop(1, '#22d3ee');
  ctx.strokeStyle = g; ctx.lineWidth = 2; ctx.beginPath();
  for (let x = 0; x < W; x++) {
    const hz = 40 + x / W * 360, i = Math.round(hz / binHz);
    const y = H - clamp((buf[i] + 110) / 90, 0, 1) * (H - 16);
    x ? ctx.lineTo(x, y) : ctx.moveTo(x, y);
  }
  ctx.stroke();
  if (BELT.peakHold) { const x = xOf(BELT.peakHold); ctx.strokeStyle = '#fbbf24'; ctx.setLineDash([4, 4]); ctx.beginPath(); ctx.moveTo(x, 0); ctx.lineTo(x, H); ctx.stroke(); ctx.setLineDash([]); }
  BELT.raf = requestAnimationFrame(beltLoop);
}
BELT.holdV = -Infinity;
