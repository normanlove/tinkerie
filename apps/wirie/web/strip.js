// Boardie — stripboard / perfboard planner inside Wirie — by nrmluvdev
"use strict";

const SV = { zoom: 9, ox: 120, oy: 90, view: "comp", tool: "select", drag: null, hover: null, linkStart: null, placing: null, sel: null };
const SP = 2.54, SM = 2.54; // pitch, margin
const rowName = r => { let s = ""; r++; while (r > 0) { const m = (r - 1) % 26; s = String.fromCharCode(65 + m) + s; r = Math.floor((r - 1) / 26); } return s; };
const holeXY = (c, r) => [SM + c * SP, SM + r * SP];
function stripSize() { const st = S.strip; return [SM * 2 + (st.cols - 1) * SP, SM * 2 + (st.rows - 1) * SP]; }
function sw2s(x, y) { const [bw] = stripSize(); const fx = SV.view === "copper" ? bw - x : x; return [fx * SV.zoom + SV.ox, y * SV.zoom + SV.oy]; }
function ss2w(x, y) { const [bw] = stripSize(); let wx = (x - SV.ox) / SV.zoom; if (SV.view === "copper") wx = bw - wx; return [wx, (y - SV.oy) / SV.zoom]; }
function stripComp(c) { const p = S.strip.place[c.id]; return p ? { id: c.id, fp: c.fp, ref: c.ref, value: c.value, x: p.x, y: p.y, rot: p.rot || 0, side: "top" } : null; }
function stripable(c) { const fp = compFP(c); return fp && fp.pads.length > 0 && fp.pads.every(p => p.drill && !p.slot); }
function stripPads(c) {
  const sc = stripComp(c); if (!sc) return [];
  return padsOf(sc).map(p => { const cc = Math.round((p.x - SM) / SP), rr = Math.round((p.y - SM) / SP); const [hx, hy] = holeXY(cc, rr); return Object.assign(p, { c: cc, r: rr, off: Math.hypot(hx - p.x, hy - p.y) > 0.3, key: c.id + ":" + p.n }); });
}
function snapStripPos(fpId, x, y, rot) {
  const fp = LIB_BY_ID[fpId]; const p1 = fp.pads[0];
  const [rx, ry] = rotPt(p1.x, p1.y, rot);
  const px = x + rx, py = y + ry;
  const c = Math.round((px - SM) / SP), r = Math.round((py - SM) / SP);
  const [hx, hy] = holeXY(c, r);
  return [hx - rx, hy - ry];
}
function stripBodyHoles(c) {
  const sc = stripComp(c); if (!sc) return [];
  const b = bodyRect(sc); const out = [];
  for (let r = Math.ceil((b.y0 - SM) / SP); r <= Math.floor((b.y1 - SM) / SP); r++) for (let cc = Math.ceil((b.x0 - SM) / SP); cc <= Math.floor((b.x1 - SM) / SP); cc++) out.push([cc, r]);
  return out;
}

let STRIP_CACHE = null;
function stripInvalidate() { STRIP_CACHE = null; }
function stripAnalyze() {
  if (STRIP_CACHE) return STRIP_CACHE;
  const st = S.strip, A = analyze();
  const occ = {}, conflicts = [], offgrid = [], outside = [];
  const placed = [], unplaced = [], notStrip = [];
  for (const c of S.comps) {
    if (!stripable(c)) { if (compFP(c) && compFP(c).pads.length) notStrip.push(c); continue; }
    if (!st.place[c.id]) { unplaced.push(c); continue; }
    placed.push(c);
    for (const p of stripPads(c)) {
      if (p.off) offgrid.push(p);
      if (p.c < 0 || p.r < 0 || p.c >= st.cols || p.r >= st.rows) { outside.push(p); continue; }
      const k = p.c + "," + p.r;
      if (occ[k] && occ[k].comp.id !== c.id) conflicts.push(p); else occ[k] = p;
    }
  }
  const cutSet = new Set([...st.cuts, ...st.autoCuts]);
  const uf = new UF();
  const hk = (c, r) => c + "," + r;
  if (st.kind === "strip") for (let r = 0; r < st.rows; r++) for (let c = 0; c < st.cols - 1; c++) if (!cutSet.has(hk(c, r))) uf.union(hk(c, r), hk(c + 1, r));
  for (const l of [...st.links, ...st.autoLinks]) uf.union(hk(...l.a), hk(...l.b));
  // net per hole group
  const netKey = p => { const n = A.netOfPad[p.key]; return n ? n.name : null; };
  const groupNets = {};
  for (const k in occ) { const p = occ[k]; const nk = netKey(p); const root = uf.find(k); (groupNets[root] = groupNets[root] || new Map()); if (nk) groupNets[root].set(nk, p); else groupNets[root].set("·" + p.key, p); }
  const shorts = [];
  for (const root in groupNets) { const m = groupNets[root]; const real = [...m.keys()]; if (real.length > 1) { const named = real.filter(k => !k.startsWith("·")); const pads = [...m.values()]; shorts.push({ root, names: real.map(k => k.startsWith("·") ? `${m.get(k).comp.ref}.${m.get(k).name}` : k), pads }); } }
  // islands per net
  const missing = []; const islandsByNet = {};
  for (const n of A.nets) {
    const ps = []; for (const p of n.pads) { const pl = S.strip.place[p.comp.id]; if (!pl) continue; const sp = stripPads(p.comp).find(q => q.n === p.n); if (sp && occ[sp.c + "," + sp.r] === sp || (sp && occ[sp.c + "," + sp.r] && occ[sp.c + "," + sp.r].key === sp.key)) ps.push(sp); }
    if (ps.length < 2) continue;
    const isl = {}; for (const p of ps) { const rt = uf.find(p.c + "," + p.r); (isl[rt] = isl[rt] || []).push(p); }
    const groups = Object.values(isl); islandsByNet[n.name] = groups;
    if (groups.length > 1) missing.push({ net: n, groups });
  }
  STRIP_CACHE = { v: UNDO.last, occ, conflicts, offgrid, outside, placed, unplaced, notStrip, cutSet, uf, shorts, missing, islandsByNet };
  return STRIP_CACHE;
}

// ---------- automatic helpers ----------
function stripRotFor(c) {
  const fp = compFP(c); let best = 0, bestScore = -1;
  for (const rot of [0, 90]) {
    const rows = new Set(fp.pads.map(p => Math.round(rotPt(p.x, p.y, rot)[1] / SP)));
    const score = rows.size * 10 + (rot === 0 ? 1 : 0);
    if (score > bestScore) { bestScore = score; best = rot; }
  }
  return best;
}
function stripAutoPlace(all) {
  const st = S.strip;
  const todo = S.comps.filter(c => stripable(c) && (all || !st.place[c.id]));
  if (all) for (const c of todo) delete st.place[c.id];
  stripInvalidate();
  if (!todo.length) return 0;
  let x0 = 1e9, y0 = 1e9, x1 = -1e9, y1 = -1e9;
  for (const c of todo) { x0 = Math.min(x0, c.x); x1 = Math.max(x1, c.x); y0 = Math.min(y0, c.y); y1 = Math.max(y1, c.y); }
  const [bw, bh] = stripSize();
  const sx = Math.min(1, (bw - 12) / Math.max(1, x1 - x0)), sy = Math.min(1, (bh - 12) / Math.max(1, y1 - y0));
  const s = Math.min(sx, sy);
  const used = new Set(), body = new Set();
  for (const c of S.comps) if (st.place[c.id]) { for (const p of stripPads(c)) used.add(p.c + "," + p.r); for (const h of stripBodyHoles(c)) body.add(h.join(",")); }
  todo.sort((a, b) => compFP(b).pads.length - compFP(a).pads.length);
  let placedN = 0;
  for (const c of todo) {
    const rot = stripRotFor(c);
    const tx = 6 + (c.x - x0) * s + (bw - 12 - (x1 - x0) * s) / 2, ty = 6 + (c.y - y0) * s + (bh - 12 - (y1 - y0) * s) / 2;
    let done = false;
    for (let rad = 0; rad < Math.max(st.cols, st.rows) && !done; rad++) {
      for (let dy = -rad; dy <= rad && !done; dy++) for (let dx = -rad; dx <= rad && !done; dx++) {
        if (Math.max(Math.abs(dx), Math.abs(dy)) !== rad) continue;
        const [x, y] = snapStripPos(c.fp, tx + dx * SP, ty + dy * SP, rot);
        st.place[c.id] = { x, y, rot };
        const pads = stripPads(c);
        const holes = stripBodyHoles(c).map(h => h.join(","));
        const ok = pads.every(p => p.c >= 0 && p.r >= 0 && p.c < st.cols && p.r < st.rows && !used.has(p.c + "," + p.r) && !body.has(p.c + "," + p.r))
          && holes.every(h => !used.has(h) && !body.has(h)) && pads.every(p => !holes.includes(p.c + "," + p.r) || true);
        // keep a 1-hole gap around bodies so wire links have room
        if (ok) { done = true; for (const p of pads) used.add(p.c + "," + p.r); for (const h of holes) body.add(h); for (const p of pads) body.add(p.c + "," + p.r); placedN++; }
      }
    }
    if (!done) delete st.place[c.id];
  }
  stripInvalidate();
  return placedN;
}
function stripAutoCuts() {
  const st = S.strip; st.autoCuts = []; stripInvalidate();
  if (st.kind !== "strip") return 0;
  const A = analyze(); const SA = stripAnalyze();
  const manual = new Set(st.cuts);
  for (let r = 0; r < st.rows; r++) {
    const row = [];
    for (let c = 0; c < st.cols; c++) { const p = SA.occ[c + "," + r]; if (p) { const n = A.netOfPad[p.key]; row.push({ c, k: n ? n.name : "·" + p.key }); } }
    for (let i = 0; i < row.length - 1; i++) {
      const a = row[i], b = row[i + 1]; if (a.k === b.k) continue;
      let already = false; for (let c = a.c; c < b.c; c++) if (manual.has(c + "," + r)) already = true;
      if (already) continue;
      const m = Math.floor((a.c + b.c - 1) / 2);
      st.autoCuts.push(m + "," + r);
    }
  }
  stripInvalidate();
  return st.autoCuts.length;
}
function stripAutoLinks() {
  const st = S.strip; st.autoLinks = []; stripInvalidate();
  let SA = stripAnalyze();
  const usedEnds = new Set(); for (const l of st.links) { usedEnds.add(l.a.join(",")); usedEnds.add(l.b.join(",")); }
  const blocked = new Set(Object.keys(SA.occ));
  for (const c of SA.placed) for (const h of stripBodyHoles(c)) blocked.add(h.join(","));
  const bodies = SA.placed.map(c => bodyRect(stripComp(c)));
  const crossesBody = (a, b) => { const [ax, ay] = holeXY(...a), [bx, by] = holeXY(...b); let n = 0; for (const r of bodies) if (segRectDist(ax, ay, bx, by, { cx: (r.x0 + r.x1) / 2, cy: (r.y0 + r.y1) / 2, hw: (r.x1 - r.x0) / 2 - 0.3, hh: (r.y1 - r.y0) / 2 - 0.3 }) <= 0) n++; return n; };
  const crossesLinks = (a, b) => { const [ax, ay] = holeXY(...a), [bx, by] = holeXY(...b); let n = 0; for (const l of [...st.links, ...st.autoLinks]) { const [cx, cy] = holeXY(...l.a), [dx, dy] = holeXY(...l.b); if (segIntersect(ax, ay, bx, by, cx, cy, dx, dy)) n++; } return n; };
  // holes per connectivity root
  const holesByRoot = {};
  for (let r = 0; r < st.rows; r++) for (let c = 0; c < st.cols; c++) { const k = c + "," + r; (holesByRoot[SA.uf.find(k)] = holesByRoot[SA.uf.find(k)] || []).push([c, r]); }
  let made = 0;
  for (const m of SA.missing) {
    const groups = m.groups.map(g => {
      const root = SA.uf.find(g[0].c + "," + g[0].r);
      let cands = st.kind === "strip" ? (holesByRoot[root] || []).filter(h => !blocked.has(h.join(",")) && !usedEnds.has(h.join(","))) : [];
      if (!cands.length) cands = g.map(p => [p.c, p.r]); // perfboard / no free hole: wire straight to the pin
      return cands;
    });
    const inT = [0]; const rest = groups.map((_, i) => i).slice(1);
    while (rest.length) {
      let best = null;
      for (const ti of inT) for (const ri of rest) for (const a of groups[ti]) for (const b of groups[ri]) {
        const dc = Math.abs(a[0] - b[0]), dr = Math.abs(a[1] - b[1]);
        let cost = dc * 1.3 + dr + (dc && dr ? 8 + 3 * Math.min(dc, dr) : 0);
        if (best && cost >= best.cost) continue;
        cost += crossesBody(a, b) * 25 + crossesLinks(a, b) * 4;
        if (!best || cost < best.cost) best = { cost, a, b, ri };
      }
      if (!best) break;
      st.autoLinks.push({ a: best.a, b: best.b, net: m.net.name });
      usedEnds.add(best.a.join(",")); usedEnds.add(best.b.join(","));
      for (const g of groups) for (let i = g.length - 1; i >= 0; i--) if (usedEnds.has(g[i].join(",")) && g.length > 1 && st.kind === "strip") g.splice(i, 1);
      inT.push(best.ri); rest.splice(rest.indexOf(best.ri), 1); made++;
    }
  }
  stripInvalidate();
  return made;
}
function stripDoAll() { stripAutoPlace(false); stripAutoCuts(); stripAutoLinks(); }

// ---------- rendering ----------
function netWireColor(name) { if (!name) return "#e6e6e6"; if (/GND/i.test(name)) return "#1d1d1f"; if (/^(5V|VIN|VBUS|VCC|RAW|12V)/i.test(name)) return "#e23b3b"; if (/3V3/i.test(name)) return "#f08a24"; const n = analyze().netByName[name]; return n ? n.color : "#4fd1c5"; }
function drawStrip(cv) {
  const ctx = cv.getContext("2d"), dpr = window.devicePixelRatio || 1, W = cv.width, H = cv.height;
  const st = S.strip, SA = stripAnalyze(), A = analyze();
  const [bw, bh] = stripSize(); const copper = SV.view === "copper", xray = SV.view === "xray";
  ctx.setTransform(1, 0, 0, 1, 0, 0);
  const bg = ctx.createRadialGradient(W / 2, H / 2, 0, W / 2, H / 2, Math.max(W, H) * 0.7); bg.addColorStop(0, "#161b24"); bg.addColorStop(1, "#0a0d12"); ctx.fillStyle = bg; ctx.fillRect(0, 0, W, H);
  const z = SV.zoom * dpr;
  if (copper) ctx.setTransform(-z, 0, 0, z, (SV.ox + bw * SV.zoom) * dpr, SV.oy * dpr); else ctx.setTransform(z, 0, 0, z, SV.ox * dpr, SV.oy * dpr);
  const px = 1 / SV.zoom;
  // board
  ctx.save(); ctx.shadowColor = "rgba(0,0,0,.6)"; ctx.shadowBlur = 28 * dpr; ctx.shadowOffsetY = 8 * dpr;
  ctx.fillStyle = st.kind === "strip" ? "#cfa874" : "#b9834a"; rrect(ctx, 0, 0, bw, bh, 0.8); ctx.fill(); ctx.restore();
  const tex = ctx.createLinearGradient(0, 0, bw, bh); tex.addColorStop(0, "rgba(255,255,255,.10)"); tex.addColorStop(1, "rgba(0,0,0,.12)"); ctx.fillStyle = tex; rrect(ctx, 0, 0, bw, bh, 0.8); ctx.fill();
  // strips
  const stripAlpha = copper ? 1 : xray ? 0.55 : 0.16;
  if (st.kind === "strip") {
    for (let r = 0; r < st.rows; r++) {
      const [, y] = holeXY(0, r);
      const g = ctx.createLinearGradient(0, y - 1, 0, y + 1); g.addColorStop(0, `rgba(232,152,86,${stripAlpha})`); g.addColorStop(0.5, `rgba(205,120,58,${stripAlpha})`); g.addColorStop(1, `rgba(150,80,34,${stripAlpha})`);
      ctx.fillStyle = g;
      // draw segments between cuts
      let start = 0;
      for (let c = 0; c <= st.cols - 1; c++) {
        const isCut = SA.cutSet.has(c + "," + r) || c === st.cols - 1;
        if (isCut) { const [x0] = holeXY(start, r), [x1] = holeXY(c, r); rrect(ctx, x0 - 1.05, y - 0.98, x1 - x0 + 2.1, 1.96, 0.3); ctx.fill(); start = c + 1; }
      }
    }
  } else if (copper || xray) {
    for (let r = 0; r < st.rows; r++) for (let c = 0; c < st.cols; c++) { const [x, y] = holeXY(c, r); ctx.fillStyle = `rgba(205,130,60,${stripAlpha})`; ctx.beginPath(); ctx.arc(x, y, 0.95, 0, 7); ctx.fill(); }
  }
  // short highlights
  for (const s of SA.shorts) for (const p of s.pads) { ctx.strokeStyle = "rgba(255,77,109,.9)"; ctx.lineWidth = 3 * px; ctx.beginPath(); ctx.arc(...holeXY(p.c, p.r), 1.5, 0, 7); ctx.stroke(); }
  // holes
  for (let r = 0; r < st.rows; r++) for (let c = 0; c < st.cols; c++) { const [x, y] = holeXY(c, r); ctx.fillStyle = "#2a1d12"; ctx.beginPath(); ctx.arc(x, y, 0.5, 0, 7); ctx.fill(); }
  // cuts
  for (const k of SA.cutSet) {
    const [c, r] = k.split(",").map(Number); const [x0, y] = holeXY(c, r); const x = x0 + SP / 2;
    const manual = st.cuts.includes(k);
    if (copper || xray) { ctx.fillStyle = "#cfa874"; ctx.fillRect(x - 0.45, y - 1.05, 0.9, 2.1); ctx.fillStyle = "rgba(0,0,0,.18)"; ctx.fillRect(x - 0.45, y - 1.05, 0.12, 2.1); }
    const s = copper || xray ? 0.62 : 0.5;
    ctx.lineCap = "round"; ctx.strokeStyle = "rgba(60,0,10,.6)"; ctx.lineWidth = Math.max(0.42, 3.4 * px);
    ctx.beginPath(); ctx.moveTo(x - s, y - s); ctx.lineTo(x + s, y + s); ctx.moveTo(x + s, y - s); ctx.lineTo(x - s, y + s); ctx.stroke();
    ctx.strokeStyle = manual ? "#ff3d68" : "#ff5a7a"; ctx.lineWidth = Math.max(0.26, 2.2 * px);
    ctx.beginPath(); ctx.moveTo(x - s, y - s); ctx.lineTo(x + s, y + s); ctx.moveTo(x + s, y - s); ctx.lineTo(x - s, y + s); ctx.stroke();
  }
  // components
  for (const c of SA.placed) {
    const sc = stripComp(c); const pads = stripPads(c);
    if (!copper) drawPartBody(ctx, sc, px * 4);
    for (const p of pads) {
      const bad = p.off || SA.conflicts.includes(p);
      if (copper || xray) { const g = ctx.createRadialGradient(p.x - 0.2, p.y - 0.2, 0, p.x, p.y, 0.9); g.addColorStop(0, "#f4f6f8"); g.addColorStop(1, "#9aa3ad"); ctx.fillStyle = g; ctx.beginPath(); ctx.arc(p.x, p.y, 0.85, 0, 7); ctx.fill(); }
      else { ctx.fillStyle = "#d7dbe0"; ctx.beginPath(); ctx.arc(p.x, p.y, 0.42, 0, 7); ctx.fill(); }
      if (bad) { ctx.strokeStyle = "#ff4d6d"; ctx.lineWidth = 2.5 * px; ctx.beginPath(); ctx.arc(p.x, p.y, 1.1, 0, 7); ctx.stroke(); }
    }
    if (SV.sel === c.id) { const b = compBox(sc); ctx.strokeStyle = COL.sel; ctx.lineWidth = 2 * px; ctx.setLineDash([4 * px, 3 * px]); ctx.strokeRect(b.x0 - 0.5, b.y0 - 0.5, b.x1 - b.x0 + 1, b.y1 - b.y0 + 1); ctx.setLineDash([]); }
    // ref label
    ctx.save(); const b = compBox(sc); ctx.translate((b.x0 + b.x1) / 2, b.y0 - 0.9); if (copper) ctx.scale(-1, 1);
    ctx.font = `700 1.15px Segoe UI, system-ui, sans-serif`; ctx.textAlign = "center"; ctx.textBaseline = "middle"; ctx.fillStyle = copper ? "rgba(60,30,10,.75)" : "rgba(30,20,10,.85)"; ctx.fillText(c.ref, 0, 0); ctx.restore();
  }
  // links
  const drawLink = (l, auto) => {
    const [ax, ay] = holeXY(...l.a), [bx, by] = holeXY(...l.b);
    if (copper) { ctx.fillStyle = "#c9ced4"; for (const [x, y] of [[ax, ay], [bx, by]]) { ctx.beginPath(); ctx.arc(x, y, 0.75, 0, 7); ctx.fill(); } return; }
    const col = netWireColor(l.net);
    ctx.lineCap = "round";
    ctx.strokeStyle = "rgba(0,0,0,.35)"; ctx.lineWidth = 1.05; ctx.beginPath(); ctx.moveTo(ax + 0.15, ay + 0.25); ctx.lineTo(bx + 0.15, by + 0.25); ctx.stroke();
    ctx.strokeStyle = col; ctx.lineWidth = 0.9; ctx.beginPath(); ctx.moveTo(ax, ay); ctx.lineTo(bx, by); ctx.stroke();
    ctx.strokeStyle = "rgba(255,255,255,.35)"; ctx.lineWidth = 0.22; ctx.beginPath(); ctx.moveTo(ax, ay - 0.2); ctx.lineTo(bx, by - 0.2); ctx.stroke();
    ctx.fillStyle = "#cfd4da"; for (const [x, y] of [[ax, ay], [bx, by]]) { ctx.beginPath(); ctx.arc(x, y, 0.45, 0, 7); ctx.fill(); }
    if (SV.sel === l) { ctx.strokeStyle = COL.sel; ctx.lineWidth = 3 * px; ctx.beginPath(); ctx.moveTo(ax, ay); ctx.lineTo(bx, by); ctx.stroke(); }
  };
  for (const l of st.autoLinks) drawLink(l, true);
  for (const l of st.links) drawLink(l, false);
  // missing connections
  ctx.setLineDash([4 * px, 3 * px]); ctx.lineWidth = 1.5 * px;
  for (const m of SA.missing) { ctx.strokeStyle = m.net.color; for (let i = 1; i < m.groups.length; i++) { const a = m.groups[0][0], b = m.groups[i][0]; ctx.beginPath(); ctx.moveTo(...holeXY(a.c, a.r)); ctx.lineTo(...holeXY(b.c, b.r)); ctx.stroke(); } }
  ctx.setLineDash([]);
  // labels
  ctx.save();
  ctx.fillStyle = "rgba(60,38,18,.75)"; ctx.font = `600 1.0px Segoe UI, system-ui, sans-serif`; ctx.textAlign = "center"; ctx.textBaseline = "middle";
  const every = SV.zoom < 7 ? 5 : 1;
  for (let c = 0; c < st.cols; c++) if ((c + 1) % every === 0 || c === 0) { const [x] = holeXY(c, 0); ctx.save(); ctx.translate(x, 1.0); if (copper) ctx.scale(-1, 1); ctx.fillText(String(c + 1), 0, 0); ctx.restore(); }
  for (let r = 0; r < st.rows; r++) { const [, y] = holeXY(0, r); ctx.save(); ctx.translate(1.0, y); if (copper) ctx.scale(-1, 1); ctx.fillText(rowName(r), 0, 0); ctx.restore(); }
  ctx.restore();
  // hover & tool previews
  if (SV.hover) {
    const [x, y] = holeXY(SV.hover.c, SV.hover.r);
    if (SV.tool === "cut") { const gx = x + SP / 2; ctx.strokeStyle = "rgba(255,61,104,.8)"; ctx.lineWidth = 2 * px; ctx.strokeRect(gx - 0.6, y - 1.1, 1.2, 2.2); }
    else { ctx.strokeStyle = "rgba(63,224,208,.9)"; ctx.lineWidth = 2 * px; ctx.beginPath(); ctx.arc(x, y, 1.0, 0, 7); ctx.stroke(); }
    if (SV.linkStart) { const [ax, ay] = holeXY(...SV.linkStart); ctx.strokeStyle = "rgba(63,224,208,.8)"; ctx.lineWidth = 0.8; ctx.lineCap = "round"; ctx.beginPath(); ctx.moveTo(ax, ay); ctx.lineTo(x, y); ctx.stroke(); }
  }
  if (SV.placing) { const c = S.comps.find(q => q.id === SV.placing.id); if (c) { const sc = { id: c.id, fp: c.fp, value: c.value, x: SV.placing.x, y: SV.placing.y, rot: SV.placing.rot, side: "top" }; ctx.globalAlpha = 0.7; drawPartBody(ctx, sc, px * 4); for (const p of padsOf(sc)) { ctx.fillStyle = "#3fe0d0"; ctx.beginPath(); ctx.arc(p.x, p.y, 0.5, 0, 7); ctx.fill(); } ctx.globalAlpha = 1; } }
  if (UI.wireFrom && UI.mode === "strip") { const p = UI.wireFrom; ctx.strokeStyle = "#ffd166"; ctx.lineWidth = 1.5 * px; ctx.setLineDash([4 * px, 3 * px]); ctx.beginPath(); ctx.moveTo(p.x, p.y); ctx.lineTo(UI.mouseW[0], UI.mouseW[1]); ctx.stroke(); ctx.setLineDash([]); }
}

function stripHit(wx, wy) {
  const st = S.strip; const SA = stripAnalyze();
  const c = Math.round((wx - SM) / SP), r = Math.round((wy - SM) / SP);
  const inBoard = c >= 0 && r >= 0 && c < st.cols && r < st.rows;
  // pad
  let pad = null; for (const comp of SA.placed) for (const p of stripPads(comp)) if (Math.hypot(p.x - wx, p.y - wy) < 1.0) pad = p;
  let link = null; for (const l of [...st.links, ...st.autoLinks]) { const [ax, ay] = holeXY(...l.a), [bx, by] = holeXY(...l.b); if (ptSegDist(wx, wy, ax, ay, bx, by) < 0.6) link = l; }
  let comp = null; for (const cc of SA.placed) { const b = bodyRect(stripComp(cc)); if (wx >= b.x0 && wx <= b.x1 && wy >= b.y0 && wy <= b.y1) comp = cc; }
  const gc = Math.floor((wx - SM) / SP); // gap to the right of this column
  return { c, r, inBoard, pad, link, comp, gap: [gc, r] };
}

// plan printout
function stripPlanHTML() {
  const st = S.strip; const SA = stripAnalyze(); const [bw, bh] = stripSize();
  const save = { view: SV.view, zoom: SV.zoom, ox: SV.ox, oy: SV.oy, sel: SV.sel, hover: SV.hover };
  const shot = view => { const cv = document.createElement("canvas"); const s = 12; cv.width = (bw + 6) * s; cv.height = (bh + 6) * s; SV.view = view; SV.zoom = s; SV.ox = 3 * s; SV.oy = 3 * s; SV.sel = null; SV.hover = null; const d = window.devicePixelRatio; Object.defineProperty(window, "devicePixelRatio", { value: 1, configurable: true }); drawStrip(cv); Object.defineProperty(window, "devicePixelRatio", { value: d, configurable: true }); return cv.toDataURL("image/png"); };
  const comp = shot("comp"), cop = shot("copper");
  Object.assign(SV, save);
  const cuts = [...SA.cutSet].map(k => k.split(",").map(Number)).sort((a, b) => a[1] - b[1] || a[0] - b[0]);
  const links = [...st.links, ...st.autoLinks];
  const hn = (c, r) => rowName(r) + (c + 1);
  const parts = SA.placed.map(c => { const ps = stripPads(c); const p1 = ps[0]; return `<tr><td>${c.ref}</td><td>${esc(c.value)}</td><td>${esc(compFP(c).name)}</td><td>pin 1 at ${hn(p1.c, p1.r)}</td></tr>`; }).join("");
  return `<!doctype html><html><head><meta charset="utf-8"><title>${esc(S.name)} — stripboard plan</title><style>body{font:13px Segoe UI,system-ui,sans-serif;color:#111;margin:24px}h1{font-size:20px;margin:0}h2{font-size:15px;margin:18px 0 6px}.sub{color:#666;margin-bottom:12px}img{max-width:100%;border:1px solid #ddd;border-radius:6px}.two{display:grid;grid-template-columns:1fr 1fr;gap:16px}table{border-collapse:collapse;width:100%}td,th{border-bottom:1px solid #e5e5e5;padding:4px 6px;text-align:left}th{background:#f5f5f5}.cols{columns:3}.foot{margin-top:24px;color:#888;font-size:11px}@media print{.noprint{display:none}}</style></head><body>
  <button class="noprint" onclick="print()" style="float:right;padding:8px 16px">Print</button>
  <h1>${esc(S.name)} — ${st.kind === "strip" ? "stripboard" : "perfboard"} plan</h1><div class="sub">${st.cols} × ${st.rows} holes · ${SA.placed.length} parts · ${cuts.length} track cuts · ${links.length} wire links · ${SA.shorts.length ? `<b style="color:#c00">${SA.shorts.length} shorts!</b>` : "no shorts"} · ${SA.missing.length ? `<b style="color:#c60">${SA.missing.length} nets still unconnected</b>` : "all connected"}</div>
  <div class="two"><div><h2>Component side (top)</h2><img src="${comp}"></div><div><h2>Copper side (flipped over)</h2><img src="${cop}"></div></div>
  ${st.kind === "strip" ? `<h2>Track cuts (${cuts.length})</h2><div class="cols">${cuts.map(([c, r]) => `<div>☐ Strip <b>${rowName(r)}</b>: cut between ${c + 1} and ${c + 2}</div>`).join("") || "None"}</div>` : ""}
  <h2>Wire links (${links.length})</h2><div class="cols">${links.map(l => `<div>☐ ${hn(...l.a)} → ${hn(...l.b)}${l.net ? ` <span style="color:#777">(${esc(l.net)})</span>` : ""}</div>`).join("") || "None"}</div>
  <h2>Parts</h2><table><tr><th>Ref</th><th>Value</th><th>Part</th><th>Position</th></tr>${parts}</table>
  <div class="foot">Made with Boardie in Wirie ${APP.version} by ${APP.author}. Tip: cut tracks with a 3.5mm drill bit or a craft knife, then check every cut with a multimeter in continuity mode.</div></body></html>`;
}
function esc(s) { return String(s).replace(/[&<>"]/g, c => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" }[c])); }
