// Wirie auto-router: two-layer grid A* with rip-up & retry — by nrmluvdev
"use strict";

// slack: extra room (mm) the router keeps round everything. Auto-fix raises it
// when a track it redrew still came out too close (a diagonal cutting a corner).
const ROUTER = { running: false, cancel: false, noPour: false, slack: 0 };

function isPowerNet(name) { return /^(GND|3V3|5V|VIN|VCC|RAW|VSYS|VBAT|VMOT|12V|VBUS)/i.test(name); }

function buildRouteGrid(g, routeW) {
  const B = S.board, A = analyze();
  const W = Math.ceil(B.w / g) + 1, H = Math.ceil(B.h / g) + 1, N = W * H;
  const grid = [new Int32Array(N), new Int32Array(N)];
  const L = { top: 0, bottom: 1 };
  const inf = B.clear + routeW / 2 + (ROUTER.slack || 0);
  // board edge
  const edge = B.edgeClear + routeW / 2;
  for (let j = 0; j < H; j++) for (let i = 0; i < W; i++) if (!insideBoard(i * g, j * g, edge)) { grid[0][j * W + i] = -1; grid[1][j * W + i] = -1; }
  const mark = (shape, extra, layers, val, force) => {
    const b = shapeBox(shape);
    const i0 = Math.max(0, Math.floor((b[0] - extra) / g)), i1 = Math.min(W - 1, Math.ceil((b[2] + extra) / g));
    const j0 = Math.max(0, Math.floor((b[1] - extra) / g)), j1 = Math.min(H - 1, Math.ceil((b[3] + extra) / g));
    for (const l of layers) { const gr = grid[L[l]]; for (let j = j0; j <= j1; j++) for (let i = i0; i <= i1; i++) {
      if (!ptInShape(i * g, j * g, shape, extra)) continue;
      const k = j * W + i, cur = gr[k];
      if (force) { if (cur !== -1 || val !== -1) gr[k] = val; }
      else if (cur === 0) gr[k] = val; else if (cur !== val) gr[k] = -1;
    } }
  };
  for (const h of A.holes) mark({ t: "cap", ax: h.x, ay: h.y, bx: h.x, by: h.y, r: h.d / 2 }, inf, ["top", "bottom"], -1, false);
  const netVal = it => (it.cnet ? it.cnet.idx + 1 : -1);
  for (const it of A.items) mark(it.shape, inf, it.layers, netVal(it), false);
  for (const it of A.items) if (it.cnet) mark(it.shape, 0, it.layers, netVal(it), true);
  return { grid, W, H, g, inf, mark };
}

class Heap { constructor() { this.k = []; this.v = []; } push(key, val) { const k = this.k, v = this.v; let i = k.length; k.push(key); v.push(val); while (i > 0) { const p = (i - 1) >> 1; if (k[p] <= key) break; k[i] = k[p]; v[i] = v[p]; i = p; } k[i] = key; v[i] = val; } pop() { const k = this.k, v = this.v; const top = v[0]; const lk = k.pop(), lv = v.pop(); if (k.length) { let i = 0; const n = k.length; while (true) { let c = 2 * i + 1; if (c >= n) break; if (c + 1 < n && k[c + 1] < k[c]) c++; if (k[c] >= lk) break; k[i] = k[c]; v[i] = v[c]; i = c; } k[i] = lk; v[i] = lv; } return top; } get size() { return this.k.length; } }

const DIRS = [[1, 0, 1], [-1, 0, 1], [0, 1, 1], [0, -1, 1], [1, 1, 1.4142], [1, -1, 1.4142], [-1, 1, 1.4142], [-1, -1, 1.4142]];

// route one disconnected island of a net to the rest. returns true if something was added.
function routeNetOnce(net, RG, opts) {
  const A = analyze();
  const { grid, W, H, g } = RG; const N = W * H;
  const nv = net.idx + 1;
  const w = isPowerNet(net.name) ? Math.max(S.board.power, S.board.track) : S.board.track;
  const extraR = Math.max(0, Math.ceil(((w - S.board.track) / 2) / g - 0.25));
  // islands of this net
  const roots = {}; const its = A.items.filter(it => it.cnet === net);
  for (const it of its) { const r = A.cu.find(it.id); (roots[r] = roots[r] || []).push(it); }
  const isl = Object.values(roots);
  if (isl.length < 2) return "done";
  isl.sort((a, b) => b.filter(x => x.kind === "pad").length - a.filter(x => x.kind === "pad").length);
  const imap = [new Int16Array(N), new Int16Array(N)];
  isl.forEach((group, gi) => { for (const it of group) { const b = shapeBox(it.shape); for (const l of it.layers) { const li = l === "top" ? 0 : 1; for (let j = Math.max(0, Math.floor(b[1] / g)); j <= Math.min(H - 1, Math.ceil(b[3] / g)); j++) for (let i = Math.max(0, Math.floor(b[0] / g)); i <= Math.min(W - 1, Math.ceil(b[2] / g)); i++) if (ptInShape(i * g, j * g, it.shape, 0)) imap[li][j * W + i] = gi + 1; } } });
  const targets = []; for (let gi = 1; gi < isl.length; gi++) for (const it of isl[gi]) if (it.kind === "pad") targets.push([it.ref.x / g, it.ref.y / g]); else targets.push([it.shape.ax / g, it.shape.ay / g]);
  const hfun = (i, j) => { let m = 1e9; for (const t of targets) { const dx = Math.abs(i - t[0]), dy = Math.abs(j - t[1]); const d = Math.max(dx, dy) + 0.4142 * Math.min(dx, dy); if (d < m) m = d; } return m; };
  const freeCell = (li, i, j) => { if (i < 0 || j < 0 || i >= W || j >= H) return false; const k = j * W + i; if (imap[li][k]) return true; const v = grid[li][k]; return v === 0 || v === nv; };
  const pass = (li, i, j) => {
    if (!freeCell(li, i, j)) return false;
    if (extraR && !imap[li][j * W + i]) { for (let dj = -extraR; dj <= extraR; dj++) for (let di = -extraR; di <= extraR; di++) if (di * di + dj * dj <= extraR * extraR + 0.5 && !freeCell(li, i + di, j + dj)) return false; }
    return true;
  };
  const viaR = Math.ceil(((S.board.viaD - S.board.track) / 2) / g + 0.75);
  const viaOK = (i, j) => { for (let li = 0; li < 2; li++) for (let dj = -viaR; dj <= viaR; dj++) for (let di = -viaR; di <= viaR; di++) if (di * di + dj * dj <= viaR * viaR + 0.5 && !freeCell(li, i + di, j + dj)) return false; return true; };
  const G = new Float32Array(2 * N).fill(Infinity), from = new Int32Array(2 * N).fill(-1), dir = new Int8Array(2 * N).fill(-1);
  const closed = new Uint8Array(2 * N);
  const heap = new Heap();
  let starts = 0;
  for (let li = 0; li < 2; li++) for (let k = 0; k < N; k++) if (imap[li][k] === 1) { const s = li * N + k; G[s] = 0; heap.push(hfun(k % W, (k / W) | 0), s); starts++; }
  if (!starts) return "fail";
  const viaCost = opts.viaCost || 10, turn = 0.6, pref = 0.25;
  let exp = 0, goal = -1;
  const maxExp = opts.maxExp || 2500000;
  while (heap.size) {
    const s = heap.pop();
    if (closed[s]) continue; closed[s] = 1;
    const li = s >= N ? 1 : 0, k = s - li * N, i = k % W, j = (k / W) | 0;
    const im = imap[li][k];
    if (im > 1) { goal = s; break; }
    if (++exp > maxExp) break;
    const gs = G[s], pd = dir[s];
    for (let d = 0; d < 8; d++) {
      const [di, dj, c] = DIRS[d]; const ni = i + di, nj = j + dj;
      if (!pass(li, ni, nj)) continue;
      if (di && dj && (!pass(li, i + di, j) || !pass(li, i, j + dj))) continue;
      const nk = nj * W + ni, ns = li * N + nk;
      if (closed[ns]) continue;
      let cost = c;
      if (pd >= 0 && pd !== d) cost += turn;
      if (li === 0 ? (dj && !di) : (di && !dj)) cost += pref;
      // don't wander across own-net copper of the source island needlessly
      const ng = gs + cost;
      if (ng < G[ns]) { G[ns] = ng; from[ns] = s; dir[ns] = d; heap.push(ng + hfun(ni, nj), ns); }
    }
    // layer change (via)
    const ol = 1 - li, os = ol * N + k;
    if (!closed[os] && pass(ol, i, j) && ((im && imap[ol][k] === im) || viaOK(i, j))) {
      const ng = gs + ((im && imap[ol][k] === im) ? 0.01 : viaCost);
      if (ng < G[os]) { G[os] = ng; from[os] = s; dir[os] = pd; heap.push(ng + hfun(i, j), os); }
    }
  }
  if (goal < 0) return "fail";
  // reconstruct
  const path = []; for (let s = goal; s >= 0; s = from[s]) path.push(s);
  path.reverse();
  // trim leading cells that are still inside the source island and trailing inside target
  let a = 0; while (a < path.length - 1 && imaps(path[a + 1]) === 1) a++;
  let b = path.length - 1; while (b > a + 1 && imaps(path[b - 1]) > 1 && layerOf(path[b - 1]) === layerOf(path[b])) b--;
  function layerOf(s) { return s >= N ? 1 : 0; }
  function imaps(s) { const li = layerOf(s); return imap[li][s - li * N]; }
  const P = path.slice(a, b + 1);
  const runs = []; let cur = null;
  for (const s of P) { const li = layerOf(s), k = s - li * N; const pt = [(k % W) * g, ((k / W) | 0) * g]; if (!cur || cur.li !== li) { if (cur) runs.push(cur); cur = { li, pts: [] }; } cur.pts.push(pt); }
  if (cur) runs.push(cur);
  const added = { tracks: [], vias: [] };
  // snap first/last point to pad centre where possible
  const snapEnd = (pt, li) => { for (const p of A.pads) { if (A.netOfPad[p.key] !== net) continue; if (!p.layers.includes(li ? "bottom" : "top")) continue; if (ptInShape(pt[0], pt[1], padShape(p), 0.01)) return [p.x, p.y]; } return null; };
  runs.forEach((r, ri) => {
    let pts = simplify(r.pts);
    if (ri === 0) { const sp = snapEnd(pts[0], r.li); if (sp) pts.unshift(sp); }
    if (ri === runs.length - 1) { const sp = snapEnd(pts[pts.length - 1], r.li); if (sp) pts.push(sp); }
    pts = simplify(pts.map(p => [+p[0].toFixed(4), +p[1].toFixed(4)]));
    if (ri > 0) { const v = r.pts[0]; const vk = Math.round(v[1] / g) * W + Math.round(v[0] / g); if (!(imap[0][vk] && imap[0][vk] === imap[1][vk])) added.vias.push({ id: uid(), x: v[0], y: v[1], d: S.board.viaD, drill: S.board.viaDrill, auto: true }); }
    if (pts.length >= 2) added.tracks.push({ id: uid(), layer: r.li ? "bottom" : "top", w, pts, auto: true });
  });
  for (const t of added.tracks) { S.tracks.push(t); for (let i = 0; i < t.pts.length - 1; i++) { const sh = { t: "cap", ax: t.pts[i][0], ay: t.pts[i][1], bx: t.pts[i + 1][0], by: t.pts[i + 1][1], r: w / 2 }; RG.mark(sh, RG.inf, [t.layer], nv, false); RG.mark(sh, 0, [t.layer], nv, true); } }
  for (const v of added.vias) { S.vias.push(v); const sh = { t: "cap", ax: v.x, ay: v.y, bx: v.x, by: v.y, r: v.d / 2 }; RG.mark(sh, RG.inf, ["top", "bottom"], nv, false); RG.mark(sh, 0, ["top", "bottom"], nv, true); }
  invalidate();
  return "ok";
}
function simplify(pts) {
  const out = [];
  for (const p of pts) {
    if (out.length && Math.abs(out[out.length - 1][0] - p[0]) < 1e-6 && Math.abs(out[out.length - 1][1] - p[1]) < 1e-6) continue;
    if (out.length >= 2) { const a = out[out.length - 2], b = out[out.length - 1]; const cr = (b[0] - a[0]) * (p[1] - b[1]) - (b[1] - a[1]) * (p[0] - b[0]); const dot = (b[0] - a[0]) * (p[0] - b[0]) + (b[1] - a[1]) * (p[1] - b[1]); if (Math.abs(cr) < 1e-6 && dot >= 0) { out[out.length - 1] = p; continue; } }
    out.push(p);
  }
  return out;
}

async function autoroute(progress, onlyNets) {
  if (ROUTER.running) return;
  ROUTER.running = true; ROUTER.cancel = false;
  // Route the fill net (GND) with tracks too, unless the board says not to:
  // a circuit whose ground only goes through the fill looks unfinished, and
  // breaks if the fill is turned off.
  ROUTER.noPour = S.board.pourRoute !== false; invalidate();
  const B = S.board;
  const g = B.w * B.h > 12000 ? 0.3175 : 0.254;
  const base = { tracks: S.tracks.slice(), vias: S.vias.slice() };
  let best = null; let order = null;
  try {
    for (let passN = 0; passN < 3 && !ROUTER.cancel; passN++) {
      S.tracks = base.tracks.slice(); S.vias = base.vias.slice(); invalidate();
      let A = analyze();
      let nets = A.nets.filter(n => n.islands > 1 && (!onlyNets || onlyNets.includes(n.name)));
      if (!order) {
        const len = n => { let x0 = 1e9, y0 = 1e9, x1 = -1e9, y1 = -1e9; for (const p of n.pads) { x0 = Math.min(x0, p.x); x1 = Math.max(x1, p.x); y0 = Math.min(y0, p.y); y1 = Math.max(y1, p.y); } return (x1 - x0 + y1 - y0) * (isPowerNet(n.name) ? 0.6 : 1); };
        nets.sort((a, b) => len(a) - len(b)); order = nets.map(n => n.name);
      }
      const names = order.filter(nm => nets.some(n => n.name === nm));
      const failed = []; let done = 0;
      const RG = buildRouteGrid(g, B.track);
      for (const nm of names) {
        if (ROUTER.cancel) break;
        let guard = 0;
        while (guard++ < 60) {
          A = analyze(); const net = A.netByName[nm]; if (!net) break;
          const r = routeNetOnce(net, RG, {});
          if (r === "done") break;
          if (r === "fail") { failed.push(nm); break; }
        }
        done++;
        progress && progress(passN, done / names.length, nm);
        await new Promise(r => setTimeout(r, 0));
      }
      // Routing one net can cut the ground fill off from another net's pins
      // (or split it), so sweep once more for anything still unconnected.
      for (const n of analyze().nets) {
        if (ROUTER.cancel || n.islands <= 1 || (onlyNets && !onlyNets.includes(n.name))) continue;
        let guard = 0;
        while (guard++ < 60) {
          const net = analyze().netByName[n.name]; if (!net) break;
          const r = routeNetOnce(net, RG, {});
          if (r === "done") break;
          if (r === "fail") { failed.push(n.name); break; }
        }
        if (!order.includes(n.name)) order.push(n.name);
        await new Promise(r => setTimeout(r, 0));
      }
      const res = { tracks: S.tracks.slice(), vias: S.vias.slice(), failed: [...new Set(failed)], unrouted: analyze().unrouted };
      if (!best || res.unrouted < best.unrouted) best = res;
      if (!res.unrouted) break;
      order = [...res.failed, ...order.filter(n => !res.failed.includes(n))];
    }
  } finally {
    if (best) { S.tracks = best.tracks; S.vias = best.vias; }
    ROUTER.noPour = false; invalidate(); ROUTER.running = false;
  }
  return best;
}
function clearAutoRoutes() { S.tracks = S.tracks.filter(t => !t.auto); S.vias = S.vias.filter(v => !v.auto); invalidate(); }
