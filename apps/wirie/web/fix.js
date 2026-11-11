// Wirie auto-fix: works through the design check's problems and fixes the
// ones it can. Created by nrmluvdev.
//
//   tracks or vias too close to something, shorted, or poking past the edge
//       -> removed, then the auto-router draws them again with proper spacing
//   parts whose pins clash, sit past the board edge or on a mounting hole
//       -> moved to the nearest free spot (their tracks are redrawn)
//   tracks too thin to make -> widened to the board's track width
//   connections not routed yet -> auto-routed
// Anything else (unknown parts, pin positions to double-check) is left for
// the person, and listed. The whole fix is one step to undo.

// Keyboard switches and knobs stay on their keys; everything else may move.
function fixCanMove(c) {
  if (c.locked) return false;
  return !(S.keeb && S.keeb.built && /^(SW|ENC)\d/.test(c.ref));
}

// Is this pad's copper inside the board with the edge clearance?
function fixPadInside(p) {
  const B = S.board, e = B.edgeClear, s = padShape(p), nc = !!compFP(p.comp).cutout;
  if (s.t === "poly") return s.pts.every(([x, y]) => insideBoard(x, y, e, nc));
  if (s.t === "rect") return [[-1, -1], [1, -1], [1, 1], [-1, 1]].every(([a, b]) => insideBoard(s.cx + a * s.hw, s.cy + b * s.hh, e, nc));
  return insideBoard(s.ax, s.ay, e + s.r, nc) && insideBoard(s.bx, s.by, e + s.r, nc);
}

// Would part c fit where it is now: inside the board, clear of other parts'
// pins and of mounting holes?
function fixSpotOK(c) {
  const B = S.board, pads = padsOf(c);
  if (!pads.every(fixPadInside)) return false;
  const box = compBox(c), near = S.comps.filter(q => q !== c && (() => { const b = compBox(q); return b.x0 < box.x1 + 3 && b.x1 > box.x0 - 3 && b.y0 < box.y1 + 3 && b.y1 > box.y0 - 3; })());
  const shapes = pads.map(p => [p, padShape(p)]);
  for (const q of near) for (const o of padsOf(q)) {
    const os = padShape(o);
    for (const [p, ps] of shapes) if (p.layers.some(l => o.layers.includes(l)) && shapeDist(ps, os) < B.clear + 0.05) return false;
  }
  for (const q of S.comps) for (const h of holesOf(q)) {
    if (q === c) continue;
    for (const [, ps] of shapes) if (shapeDist(ps, { t: "cap", ax: h.x, ay: h.y, bx: h.x, by: h.y, r: h.d / 2 }) < B.clear + 0.05) return false;
  }
  return true;
}

// Move part c to the nearest spot that fits, trying the direction (dx, dy)
// first and then turning further and further from it. Returns true if moved.
function fixRelocate(c, dx, dy) {
  const L = Math.hypot(dx, dy); if (L < 1e-6) { dx = S.board.w / 2 - c.x; dy = S.board.h / 2 - c.y; }
  let a0 = Math.atan2(dy, dx); if (!isFinite(a0)) a0 = 0;
  const x0 = c.x, y0 = c.y, g = Math.min(0.25, S.board.grid || 0.25);
  const turns = [0, 30, -30, 60, -60, 90, -90, 135, -135, 180].map(t => t * Math.PI / 180);
  for (let r = g; r <= 30; r += g) for (const t of turns) {
    c.x = +snapTo(x0 + Math.cos(a0 + t) * r, g / 2).toFixed(4); c.y = +snapTo(y0 + Math.sin(a0 + t) * r, g / 2).toFixed(4);
    invalidate();
    if (fixSpotOK(c)) return true;
  }
  c.x = x0; c.y = y0; invalidate();
  return false;
}

// Tracks with a point on one of part c's pins (they'd be left behind if it moves).
function fixTracksOn(c) {
  const pads = padsOf(c).map(p => padShape(p));
  return S.tracks.filter(t => t.pts.some(([x, y]) => pads.some(s => ptInShape(x, y, s, 0.01))));
}

// a short description of the problems auto-fix can work on, to tell whether a
// pass (or a whole run) changed anything
function fixSignature(drc) { return drc.filter(d => ["clear", "short", "rat", "edge", "hole", "thin"].includes(d.code)).map(d => d.code + ":" + d.msg).sort().join("|"); }

async function autoFix(progress) {
  const B = S.board, did = { tracks: new Set(), moved: new Set(), widened: 0, routed: 0, stuck: new Set() };
  const say = m => progress && progress(m);
  let lastSig = null, lastUnrouted = -1;
  try {
  for (let pass = 0; pass < 5; pass++) {
    if (typeof UI !== "undefined" && UI.fixStop) break;
    say(pass ? `Checking again (round ${pass + 1})…` : "Looking at the problems…");
    const A = analyze(), drc = runDRC(), sig = fixSignature(drc);
    if (!sig) break; // nothing left to fix
    if (sig === lastSig && pass > 1) break; // the same problems as last time: more of the same won't help
    lastSig = sig;
    const rip = new Set(), moves = [], nets = new Set();
    const ripIt = it => { if (it.kind === "seg" || it.kind === "via") { rip.add(it.ref); if (it.cnet) nets.add(it.cnet.name); return true; } return false; };
    const moveFor = (it, ax, ay) => { const c = it.ref.comp; if (!fixCanMove(c)) return false; moves.push([c, it.ref.x - ax, it.ref.y - ay]); return true; };
    // shorts: tracks and vias joining two nets come out; overlapping pins get pushed apart
    for (const it of A.items) if (it.short) ripIt(it);
    const sp = A.items.filter(it => it.short && it.kind === "pad");
    for (let i = 0; i < sp.length; i++) for (let j = i + 1; j < sp.length; j++) {
      const a = sp[i], b = sp[j];
      if (a.ref.comp === b.ref.comp || a.net === b.net || shapeDist(a.shape, b.shape) > 0.001) continue;
      const [m, o] = padsOf(a.ref.comp).length <= padsOf(b.ref.comp).length ? [a, b] : [b, a];
      if (!moveFor(m, o.ref.x, o.ref.y)) moveFor(o, m.ref.x, m.ref.y);
    }
    for (const d of drc) {
      if (d.code === "clear") {
        const { a, b } = d, at = a.kind !== "pad", bt = b.kind !== "pad";
        if (at && bt) {
          // two tracks: the first time redraw one; if that didn't help, redraw both, and
          // as a last resort make a wide power track normal width so they fit
          if (pass === 0) ripIt(a.ref.auto || !b.ref.auto ? a : b);
          else if (pass < 3) { ripIt(a); ripIt(b); }
          else { for (const t of [a.ref, b.ref]) if (t.w > B.track + 0.01) { t.w = B.track; did.widened++; } ripIt(a); ripIt(b); }
        } else if (at || bt) ripIt(at ? a : b);
        else { const [m, o] = padsOf(a.ref.comp).length <= padsOf(b.ref.comp).length ? [a, b] : [b, a]; if (!moveFor(m, o.ref.x, o.ref.y) && !moveFor(o, m.ref.x, m.ref.y)) did.stuck.add(d.msg); }
      } else if (d.code === "edge") {
        if (!ripIt(d.it)) { const c = d.it.ref.comp; if (fixCanMove(c)) moves.push([c, B.w / 2 - c.x, B.h / 2 - c.y]); else did.stuck.add(d.msg); }
      } else if (d.code === "hole") {
        if (!ripIt(d.it) && !moveFor(d.it, d.h.x, d.h.y)) did.stuck.add(d.msg);
      } else if (d.code === "thin") {
        if (d.t.auto) rip.add(d.t); else { d.t.w = Math.max(0.15, B.track || 0.25); did.widened++; }
      }
    }
    // move parts (once each), taking their tracks with them to be redrawn
    const seen = new Set();
    for (const [c, dx, dy] of moves) {
      if (seen.has(c)) continue; seen.add(c);
      say(`Moving ${c.ref} to a free spot…`);
      const on = fixTracksOn(c);
      if (fixRelocate(c, dx, dy)) { did.moved.add(c.ref); for (const t of on) rip.add(t); }
      else did.stuck.add(`${c.ref}: no free spot nearby — make the board a little bigger or move it by hand`);
    }
    if (rip.size) {
      for (const o of rip) did.tracks.add(o.id);
      S.tracks = S.tracks.filter(t => !rip.has(t)); S.vias = S.vias.filter(v => !rip.has(v)); invalidate();
    }
    const A2 = analyze(), before = A2.unrouted;
    // nothing taken up or moved, and routing already failed to finish these: stop trying
    if (!rip.size && !seen.size && before === lastUnrouted) break;
    if (before) {
      say(`Routing ${before} connection${before > 1 ? "s" : ""}…`);
      // each round keeps a little more room round tracks, so a redrawn one doesn't land in the same spot
      ROUTER.slack = Math.min(0.35, pass * 0.12);
      const only = seen.size ? null : A2.nets.filter(n => n.islands > 1).map(n => n.name); // just the nets that need it
      await autoroute((p, f, nm) => say(`Routing ${nm}…`), only && only.length ? only : undefined);
      ROUTER.slack = 0;
      const after = analyze().unrouted;
      did.routed += Math.max(0, before - after); lastUnrouted = after;
    }
    if (!rip.size && !seen.size && !before) break; // nothing more this pass can do
    await new Promise(r => setTimeout(r, 0));
  }
  } finally { ROUTER.slack = 0; }
  const left = runDRC();
  for (const d of left) if (d.code === "rat") did.stuck.add(`${d.msg}: the router found no way through. Moving nearby parts apart or making the board a little bigger usually helps.`);
  return { ...did, left, leftErr: left.filter(d => d.sev === "err").length, leftWarn: left.filter(d => d.sev === "warn").length };
}
