/* Tunie — diagrams.js: simple illustrated guides showing what to print,
   where to put the calipers and what to look at.
   Part of the nrmluvdev maker suite. Created by nrmluvdev. */
'use strict';

const DG = (() => {
  const C = { obj: '#273047', objHi: '#34405c', edge: '#5b6787', meas: '#fbbf24', acc: '#22d3ee', vio: '#a78bfa', txt: '#c9cfe0', dim: '#7c86a0', good: '#34d399', bad: '#f87171' };
  const svg = (inner, w = 440, h = 230) => `<svg viewBox="0 0 ${w} ${h}" xmlns="http://www.w3.org/2000/svg" font-family="Segoe UI, system-ui, sans-serif">
    <defs><marker id="ah" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse"><path d="M0 0L10 5L0 10z" fill="${C.meas}"/></marker>
    <marker id="ahc" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse"><path d="M0 0L10 5L0 10z" fill="${C.acc}"/></marker>
    <linearGradient id="gObj" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#36425f"/><stop offset="1" stop-color="#232b40"/></linearGradient>
    <linearGradient id="gSide" x1="0" y1="0" x2="1" y2="0"><stop offset="0" stop-color="#1e2537"/><stop offset="1" stop-color="#2a3350"/></linearGradient></defs>${inner}</svg>`;
  const t = (x, y, s, o = {}) => `<text x="${x}" y="${y}" fill="${o.c || C.txt}" font-size="${o.size || 12}" font-weight="${o.w || 500}" text-anchor="${o.a || 'middle'}">${s}</text>`;
  const dim = (x1, y1, x2, y2, label, o = {}) => {
    const mx = (x1 + x2) / 2, my = (y1 + y2) / 2;
    return `<line x1="${x1}" y1="${y1}" x2="${x2}" y2="${y2}" stroke="${o.c || C.meas}" stroke-width="1.6" marker-start="url(#ah)" marker-end="url(#ah)"/>` +
      `<rect x="${mx - label.length * 3.6 - 6}" y="${my - 10 + (o.dy || 0)}" width="${label.length * 7.2 + 12}" height="19" rx="6" fill="#0d1017" stroke="${o.c || C.meas}" stroke-opacity=".5"/>` +
      t(mx, my + 4 + (o.dy || 0), label, { c: o.c || C.meas, w: 650, size: 11.5 });
  };
  const caliper = (x, y, gap, rot = 0) => `<g transform="translate(${x} ${y}) rotate(${rot})" opacity=".95">
    <rect x="-14" y="-62" width="${gap + 60}" height="14" rx="3" fill="#59647f"/>
    <rect x="-14" y="-62" width="12" height="62" rx="2" fill="#7a86a3"/><rect x="${gap + 2}" y="-62" width="12" height="62" rx="2" fill="#7a86a3"/>
    <rect x="${gap + 18}" y="-76" width="36" height="22" rx="4" fill="#1c2232" stroke="${C.acc}"/>${t(gap + 36, -61, '00.00', { c: C.acc, size: 10, w: 700 })}</g>`;
  const iso = (x, y, w, d, h, fill = 'url(#gObj)') => { // simple isometric box, (x,y) = front-bottom-left
    const dx = d * 0.6, dy = d * 0.35;
    return `<path d="M${x} ${y} h${w} v${-h} h${-w}z" fill="${fill}" stroke="${C.edge}"/>
      <path d="M${x + w} ${y} l${dx} ${-dy} v${-h} l${-dx} ${dy}z" fill="url(#gSide)" stroke="${C.edge}"/>
      <path d="M${x} ${y - h} l${dx} ${-dy} h${w} l${-dx} ${dy}z" fill="${C.objHi}" stroke="${C.edge}"/>`;
  };
  const bed = (y = 200) => `<rect x="20" y="${y}" width="400" height="10" rx="3" fill="#1a2030" stroke="#2a3246"/>`;

  return {
    pid: () => svg(`
      <line x1="40" y1="190" x2="420" y2="190" stroke="#2a3246"/><line x1="40" y1="20" x2="40" y2="190" stroke="#2a3246"/>
      <line x1="40" y1="70" x2="420" y2="70" stroke="${C.good}" stroke-dasharray="5 5" opacity=".7"/>${t(415, 62, 'target', { a: 'end', c: C.good, size: 11 })}
      <path d="M40 185 C 90 185, 95 30, 130 40 S 170 110, 195 95 S 230 55, 250 66 S 285 80, 300 72 S 340 69, 420 70" fill="none" stroke="${C.acc}" stroke-width="2.5"/>
      <path d="M40 185 C 80 185, 90 10, 120 25 S 170 140, 200 120 S 240 20, 270 40 S 320 120, 350 100 S 400 50, 420 60" fill="none" stroke="${C.bad}" stroke-width="1.6" opacity=".55" stroke-dasharray="4 4"/>
      ${t(130, 214, 'Before: overshoots and wobbles', { c: C.bad, size: 11.5 })}${t(320, 214, 'After: settles quickly and stays flat', { c: C.acc, size: 11.5 })}
      ${t(28, 30, '°C', { c: C.dim, size: 11 })}${t(420, 182, 'time', { a: 'end', c: C.dim, size: 11 })}`),

    esteps: () => svg(`
      <rect x="40" y="150" width="90" height="56" rx="8" fill="url(#gObj)" stroke="${C.edge}"/>${t(85, 182, 'Extruder', { size: 12 })}
      <rect x="77" y="30" width="16" height="120" fill="#e5e7eb" opacity=".85"/><rect x="77" y="58" width="16" height="4" fill="${C.bad}"/>
      ${t(105, 63, 'mark at 120 mm', { a: 'start', c: C.bad, size: 11.5, w: 650 })}
      <line x1="62" y1="60" x2="62" y2="150" stroke="${C.meas}" stroke-width="1.6" marker-start="url(#ah)" marker-end="url(#ah)"/>${t(52, 108, '120', { a: 'end', c: C.meas, size: 11.5, w: 700 })}
      <path d="M200 110 h60" stroke="${C.acc}" stroke-width="2" marker-end="url(#ahc)"/>${t(230, 98, 'extrude 100 mm', { c: C.acc, size: 11.5 })}
      <rect x="290" y="150" width="90" height="56" rx="8" fill="url(#gObj)" stroke="${C.edge}"/>${t(335, 182, 'Extruder', { size: 12 })}
      <rect x="327" y="110" width="16" height="40" fill="#e5e7eb" opacity=".85"/><rect x="327" y="128" width="16" height="4" fill="${C.bad}"/>
      ${dim(400, 130, 400, 150, 'left?', { dy: 0 })}
      ${t(335, 224, 'Ideal: exactly 20 mm left', { c: C.good, size: 11.5, w: 650 })}`),

    flowWall: () => svg(`
      ${bed(206)}${iso(140, 200, 150, 110, 120, 'none')}
      <path d="M140 200 h150 v-120 h-150z" fill="url(#gObj)" opacity=".55" stroke="${C.edge}"/>
      <path d="M146 80 l66 -38 h150" fill="none" stroke="${C.edge}" opacity=".6"/>
      <rect x="140" y="56" width="8" height="144" fill="${C.acc}" opacity=".5"/>
      ${caliper(128, 120, 30, 0)}
      ${t(220, 226, 'Measure mid-side, not at the corners', { size: 11.5, c: C.dim })}
      ${dim(118, 150, 160, 150, 'wall')}`),

    flowPatches: () => svg(`
      ${[0, 1, 2].map((r) => [0, 1, 2].map((c) => {
        const x = 90 + c * 90, y = 25 + r * 62, best = r === 1 && c === 1, over = r === 2, under = r === 0;
        return `<rect x="${x}" y="${y}" width="70" height="46" rx="4" fill="url(#gObj)" stroke="${best ? C.good : C.edge}" stroke-width="${best ? 2.5 : 1}"/>` +
          (under ? `<path d="M${x + 6} ${y + 12} h58 M${x + 6} ${y + 24} h58 M${x + 6} ${y + 36} h58" stroke="#0d1017" stroke-width="2.5"/>` : '') +
          (over ? `<path d="M${x + 8} ${y + 10} q10 6 20 0 t20 0 t20 0 M${x + 8} ${y + 26} q10 6 20 0 t20 0 t20 0" stroke="#7d89a8" fill="none" stroke-width="2"/>` : '');
      }).join('')).join('')}
      ${t(40, 52, 'gaps', { c: C.dim, size: 11 })}${t(40, 114, 'smooth', { c: C.good, size: 11, w: 650 })}${t(40, 176, 'ridges', { c: C.dim, size: 11 })}
      ${t(225, 222, 'Pick the patch with the smoothest top: no gaps, no ridges', { size: 11.5, c: C.dim })}`),

    tempTower: (labels = ['220', '215', '210', '205', '200']) => svg(`
      ${bed(206)}${labels.map((l, i) => {
        const y = 200 - i * 34;
        const best = i === 2;
        return `<rect x="150" y="${y - 30}" width="46" height="30" fill="url(#gObj)" stroke="${best ? C.good : C.edge}"/>` +
          `<rect x="226" y="${y - 30}" width="46" height="30" fill="url(#gObj)" stroke="${best ? C.good : C.edge}"/>` +
          `<rect x="150" y="${y - 34}" width="122" height="5" fill="${C.objHi}" stroke="${C.edge}"/>` +
          t(173, y - 10, l, { size: 11, w: 700, c: best ? C.good : C.txt }) +
          (i < 2 ? `<path d="M196 ${y - 18} q15 ${6 + i * 3} 30 0" stroke="#cbd5e1" fill="none" stroke-width=".8"/>` : '') +
          (i > 3 ? `<path d="M196 ${y - 30} q15 6 30 0" stroke="${C.bad}" fill="none" stroke-width="1.5"/>` : '');
      }).join('')}
      ${t(290, 172, '← too hot: strings', { a: 'start', size: 11.5, c: C.dim })}
      ${t(290, 47, '← too cold: droopy, weak', { a: 'start', size: 11.5, c: C.dim })}
      ${t(290, 110, '← just right', { a: 'start', size: 11.5, c: C.good, w: 650 })}
      ${t(132, 110, 'label', { a: 'end', size: 11, c: C.dim })}`),

    retraction: () => svg(`
      ${bed(206)}${[0, 1, 2, 3, 4].map((i) => {
        const y = 200 - i * 34;
        return `<rect x="140" y="${y - 30}" width="40" height="30" fill="url(#gObj)" stroke="${C.edge}"/><rect x="260" y="${y - 30}" width="40" height="30" fill="url(#gObj)" stroke="${C.edge}"/>` +
          t(160, y - 10, ['0.2', '0.6', '1.0', '1.4', '1.8'][i], { size: 10.5, w: 700 }) +
          (i < 2 ? Array.from({ length: 4 - i * 2 }, (_, k) => `<path d="M180 ${y - 6 - k * 6} q40 ${8 + k * 2} 80 0" stroke="#d1d5db" fill="none" stroke-width=".8"/>`).join('') : '');
      }).join('')}
      ${t(320, 175, '← lots of strings', { a: 'start', size: 11.5, c: C.dim })}${t(320, 75, '← clean: use the', { a: 'start', size: 11.5, c: C.good, w: 650 })}${t(320, 91, '   lowest clean band', { a: 'start', size: 11.5, c: C.good, w: 650 })}`),

    paLines: () => svg(`
      ${[0, 1, 2, 3, 4].map((i) => {
        const y = 40 + i * 34;
        const bulge = [8, 4, 0, -3, -6][i];
        return `<path d="M60 ${y} h70" stroke="#7d89a8" stroke-width="5" stroke-linecap="round"/>` +
          `<path d="M130 ${y} h160" stroke="${i === 2 ? C.good : '#9aa5c3'}" stroke-width="${i === 2 ? 5 : 3.5 + bulge * .15}" stroke-linecap="round"/>` +
          (bulge > 0 ? `<circle cx="130" cy="${y}" r="${4 + bulge * .35}" fill="#7d89a8"/><path d="M290 ${y} h-20" stroke="#4a5470" stroke-width="2"/>` : '') +
          (bulge < 0 ? `<path d="M134 ${y} h22" stroke="#0d1017" stroke-width="3"/>` : '') +
          `<path d="M290 ${y} h70" stroke="#7d89a8" stroke-width="5" stroke-linecap="round"/>` + t(395, y + 4, ['0.00', '0.02', '0.04', '0.06', '0.08'][i], { size: 11, c: i === 2 ? C.good : C.dim, w: 700 });
      }).join('')}
      ${t(95, 214, 'slow', { c: C.dim, size: 11 })}${t(210, 214, 'fast', { c: C.dim, size: 11 })}${t(325, 214, 'slow', { c: C.dim, size: 11 })}`),

    paTower: () => svg(`
      ${bed(206)}${iso(150, 200, 120, 90, 170)}
      <path d="M150 200 v-170" stroke="${C.meas}" stroke-width="3" opacity=".9"/>
      <path d="M140 150 q-6 -10 0 -20" stroke="${C.bad}" fill="none"/><path d="M140 80 q6 -10 0 -20" stroke="${C.bad}" fill="none"/>
      ${t(130, 144, 'bulging', { a: 'end', size: 11, c: C.bad })}${t(130, 74, 'rounded / gaps', { a: 'end', size: 11, c: C.bad })}
      <line x1="300" y1="200" x2="300" y2="110" stroke="${C.meas}" stroke-width="1.6" marker-end="url(#ah)"/>${t(310, 160, 'height where the', { a: 'start', size: 11.5 })}${t(310, 176, 'corners look sharpest', { a: 'start', size: 11.5 })}
      <path d="M150 110 h-8" stroke="${C.good}" stroke-width="3"/>${t(130, 114, 'sharp', { a: 'end', size: 11, c: C.good, w: 650 })}`),

    maxflow: () => svg(`
      ${bed(206)}<ellipse cx="220" cy="200" rx="80" ry="14" fill="#232b40" stroke="${C.edge}"/>
      <path d="M140 200 V60 a80 14 0 0 0 160 0 V200" fill="url(#gObj)" stroke="${C.edge}"/>
      <path d="M140 95 h160 V60 a80 14 0 0 1 -160 0z" fill="#1a2030" opacity=".8"/>
      ${[0, 1, 2, 3].map((k) => `<path d="M140 ${84 - k * 6} l10 4 -8 4 12 3" stroke="${C.bad}" fill="none" opacity=".8"/>`).join('')}
      ${t(220, 82, 'gaps / rough / clicking', { c: C.bad, size: 11.5, w: 650 })}
      ${[1, 2, 3].map((k) => `<path d="M300 ${200 - k * 40} h8" stroke="${C.acc}" stroke-width="2.5"/>`).join('')}${t(318, 164, 'tick every 5 mm', { a: 'start', c: C.acc, size: 11 })}
      ${dim(405, 200, 405, 98, 'height', { dy: 0 })}
      ${t(110, 150, 'measure from the', { a: 'end', size: 11, c: C.dim })}${t(110, 165, 'top of the base', { a: 'end', size: 11, c: C.dim })}`),

    ringing: () => svg(`
      ${bed(206)}${iso(90, 200, 150, 90, 160)}
      ${t(165, 130, 'X', { size: 22, w: 800, c: C.vio })}${t(290, 110, 'Y', { size: 22, w: 800, c: C.vio })}
      ${[0, 1, 2, 3].map((k) => `<path d="M${102 + k * 9} 60 v120" stroke="#5f6b8a" opacity="${.9 - k * .2}" stroke-width="1.6"/>`).join('')}
      ${dim(101, 50, 128, 50, 'distance D', { dy: -2 })}
      ${t(436, 150, 'Count ripples (N)', { a: 'end', size: 11.5 })}${t(436, 166, 'after a corner,', { a: 'end', size: 11.5 })}${t(436, 182, 'measure span (D)', { a: 'end', size: 11.5 })}`),

    firstLayer: () => svg(`
      ${['Too high', 'Just right', 'Too low'].map((l, i) => {
        const x = 30 + i * 140;
        const beads = [0, 1, 2, 3, 4].map((k) => {
          if (i === 0) return `<ellipse cx="${x + 20 + k * 20}" cy="120" rx="8" ry="9" fill="#9aa5c3"/>`;
          if (i === 1) return `<rect x="${x + 11 + k * 20}" y="114" width="21" height="11" rx="4" fill="#9aa5c3"/>`;
          return `<rect x="${x + 9 + k * 20}" y="119" width="23" height="6" rx="2" fill="#9aa5c3"/><path d="M${x + 12 + k * 20} 116 l3 -5 l3 5" fill="#9aa5c3"/>`;
        }).join('');
        return `<rect x="${x}" y="125" width="120" height="8" fill="#1a2030" stroke="#2a3246"/>${beads}` +
          t(x + 60, 160, l, { w: 700, c: i === 1 ? C.good : C.txt }) +
          t(x + 60, 178, ['round lines, gaps,', 'flat lines that just', 'thin, see-through or'][i], { size: 11, c: C.dim }) +
          t(x + 60, 193, ['peels off easily', 'touch — smooth top', 'rough, ridges/scraping'][i], { size: 11, c: C.dim });
      }).join('')}
      ${t(220, 40, 'Look closely at each patch (side view of the lines):', { size: 12.5 })}`),

    cube: () => svg(`
      ${bed(206)}${iso(150, 200, 120, 110, 120)}
      ${t(210, 150, 'X', { size: 24, w: 800, c: C.vio })}${t(305, 128, 'Y', { size: 24, w: 800, c: C.vio })}
      ${dim(150, 214, 270, 214, 'X', { dy: 0 })}${dim(285, 212, 352, 174, 'Y')}${dim(130, 200, 130, 80, 'Z')}
      ${t(24, 40, 'Measure each axis 2–3 times', { a: 'start', size: 11.5 })}${t(24, 56, 'and avoid the bottom edge', { a: 'start', size: 11.5, c: C.dim })}`),

    frame: () => svg(`
      <g transform="translate(130 20)">${[180, 130, 80].map((s) => `<rect x="${(180 - s) / 2}" y="${(180 - s) / 2}" width="${s}" height="${s}" fill="none" stroke="#5b6787" stroke-width="9"/>`).join('')}
      <line x1="-4" y1="184" x2="184" y2="-4" stroke="${C.meas}" stroke-width="1.6" marker-start="url(#ah)" marker-end="url(#ah)"/>
      <line x1="-4" y1="-4" x2="184" y2="184" stroke="${C.acc}" stroke-width="1.6" marker-start="url(#ahc)" marker-end="url(#ahc)"/>
      ${t(-14, 200, 'A', { w: 800, c: C.vio, size: 15 })}${t(-14, -2, 'B', { w: 800, c: C.vio, size: 15 })}${t(194, -2, 'C', { w: 800, c: C.vio, size: 15 })}${t(194, 200, 'D', { w: 800, c: C.vio, size: 15 })}
      ${t(140, 64, 'AC', { c: C.meas, w: 700 })}${t(40, 64, 'BD', { c: C.acc, w: 700 })}</g>
      ${t(70, 110, 'Measure', { size: 11.5 })}${t(70, 126, 'outside X & Y', { size: 11.5 })}${t(70, 142, 'of each square', { size: 11.5 })}
      ${t(370, 110, 'Diagonals of', { size: 11.5 })}${t(370, 126, 'the big square,', { size: 11.5 })}${t(370, 142, 'corner to corner', { size: 11.5 })}`),

    overhang: () => svg(`
      ${bed(206)}<rect x="60" y="188" width="320" height="12" fill="url(#gObj)" stroke="${C.edge}"/>
      ${[30, 40, 50, 60, 70].map((a, i) => {
        const x = 90 + i * 62, h = Math.min(120, 90 / Math.tan(a * Math.PI / 180) + 20), off = h * Math.tan(a * Math.PI / 180) * .5;
        return `<path d="M${x} 188 l${-off} ${-h} h14 l${off} ${h}z" fill="url(#gObj)" stroke="${i > 2 ? C.bad : C.edge}"/>` + t(x + 7, 222, a + '°', { size: 11.5, w: 700, c: i > 2 ? C.bad : C.txt }) +
          (i > 2 ? `<path d="M${x - off + 2} ${188 - h + 8} q-4 6 0 10 q4 4 0 10" stroke="${C.bad}" fill="none"/>` : '');
      }).join('')}
      ${t(420, 40, 'Steeper fins droop or curl —', { a: 'end', size: 11.5 })}${t(420, 56, 'note the last clean angle', { a: 'end', size: 11.5, c: C.good, w: 650 })}`),

    bridge: () => svg(`
      ${bed(206)}${[[40, 30, 1, '10 mm', C.good], [140, 70, 4, '30 mm', C.good], [270, 120, 16, '50 mm', C.bad]].map(([x, gap, sag, l, col]) =>
        `<rect x="${x}" y="120" width="16" height="80" fill="url(#gObj)" stroke="${C.edge}"/><rect x="${x + 16 + gap}" y="120" width="16" height="80" fill="url(#gObj)" stroke="${C.edge}"/>` +
        `<path d="M${x} 118 L${x + 16} 118 Q ${x + 16 + gap / 2} ${118 + sag * 2} ${x + 16 + gap} 118 L${x + 32 + gap} 118" stroke="${col}" stroke-width="6" fill="none" stroke-linecap="round"/>` +
        t(x + 16 + gap / 2, 224, l, { size: 11.5, w: 650, c: col })).join('')}
      ${t(220, 60, 'Side view: straight = good, sagging = too long', { size: 12 })}`),

    tolerance: () => svg(`
      ${bed(206)}${[0, 1, 2, 3].map((i) => {
        const cx = 90 + i * 80;
        return `<ellipse cx="${cx}" cy="170" rx="32" ry="12" fill="url(#gObj)" stroke="${C.edge}"/><ellipse cx="${cx}" cy="170" rx="${14 + i * 1.2}" ry="${5 + i * .4}" fill="#0d1017"/>` +
          t(cx, 200, ['0.0', '0.1', '0.2', '0.3'][i], { size: 11, w: 700 });
      }).join('')}
      <rect x="383" y="80" width="26" height="96" rx="3" fill="url(#gObj)" stroke="${C.acc}"/><ellipse cx="396" cy="80" rx="13" ry="5" fill="${C.objHi}" stroke="${C.acc}"/>
      ${t(396, 66, 'peg', { c: C.acc, size: 11.5, w: 650 })}
      <path d="M380 110 C 320 70, 260 80, 250 140" stroke="${C.acc}" fill="none" stroke-dasharray="4 4" marker-end="url(#ahc)"/>
      ${t(200, 40, 'Push the peg into each ring (top side first) and note:', { size: 12 })}
      ${t(200, 58, 'won\'t go · tight press · slides · loose/wobbly', { size: 11.5, c: C.dim })}`),

    stringing: () => svg(`
      ${bed(206)}${[90, 190, 290, 370].map((x, i) => `<path d="M${x - 14} 200 L${x - 5} 60 h10 L${x + 14} 200z" fill="url(#gObj)" stroke="${C.edge}"/>`).join('')}
      ${[0, 1, 2, 3, 4].map((k) => `<path d="M${100 + k * 3} ${80 + k * 18} Q 140 ${100 + k * 18} ${180 - k * 2} ${86 + k * 18}" stroke="#d1d5db" fill="none" stroke-width=".7"/>`).join('')}
      ${t(140, 40, 'hairs / strings', { size: 11.5, c: C.bad })}${t(330, 40, 'clean', { size: 11.5, c: C.good, w: 650 })}`),

    belt: () => svg(`
      <circle cx="70" cy="110" r="26" fill="url(#gObj)" stroke="${C.edge}"/><circle cx="370" cy="110" r="26" fill="url(#gObj)" stroke="${C.edge}"/>
      <path d="M70 84 H370" stroke="#9aa5c3" stroke-width="5"/><path d="M70 136 Q 220 ${136 + 22} 370 136" stroke="${C.acc}" stroke-width="5" fill="none"/>
      <path d="M70 136 Q 220 ${136 - 22} 370 136" stroke="${C.acc}" stroke-width="2" fill="none" opacity=".4"/>
      <path d="M220 190 v-30" stroke="${C.meas}" stroke-width="2" marker-end="url(#ah)"/>${t(220, 206, 'pluck the middle like a guitar string', { c: C.meas, size: 11.5, w: 650 })}
      ${dim(70, 40, 370, 40, 'free span length L')}`),

    mesh: () => svg(`<g transform="translate(220 120)">${Array.from({ length: 6 }, (_, i) => Array.from({ length: 6 }, (_, j) => {
      const x = (i - j) * 26, y = (i + j) * 13 - 70, z = Math.sin(i * .9) * 6 + Math.cos(j * .8) * 6;
      return `<circle cx="${x}" cy="${y - z}" r="3.5" fill="hsl(${200 - z * 12} 80% 60%)"/>`;
    }).join('')).join('')}</g>`),
  };
})();
