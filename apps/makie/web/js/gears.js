// Gear ratio calculator and "make a matching part" (gears, racks, nuts, cut-outs …)
import { S, toast } from './state.js';
import * as sc from './scene.js';
import { mul, trs, round } from './math.js';
import { localBox, whenBuilt } from './api.js';
import { h } from './ui.js';
import * as ui from './ui.js';

let C = null; // { commit, select }
export function initGears(cb) { C = cb; }

const T = (n, k, d) => +(n.params?.[k] ?? d);

// put a new node next to `ref`, at offset/rotation given in ref's own frame
function addRelative(ref, type, params, pos, rotDeg, name) {
  const m = sc.makeNode(type, params);
  if (name) m.name = name;
  m.color = ref.color;
  sc.siblings(ref.id).push(m);
  sc.setWorld(m, mul(sc.worldMatrix(ref.id), trs(pos, rotDeg, [1, 1, 1])), sc.parentWorld(ref.id));
  return m;
}

function addJointPair(a, b, ratio) {
  S.joints ||= [];
  const W = (id) => sc.worldMatrix(id);
  const axisOf = (id) => { const w = W(id); return [w[8], w[9], w[10]]; };
  const originOf = (id) => { const w = W(id); return [w[12], w[13], w[14]]; };
  let ja = S.joints.find((j) => j.part === a.id);
  if (!ja) {
    ja = { id: sc.newId(), name: 'Turn ' + a.name, type: 'rotate', part: a.id, followers: [], origin: originOf(a.id), axis: axisOf(a.id), min: -180, max: 180, speed: 60, loop: false };
    S.joints.push(ja);
  }
  S.joints.push({ id: sc.newId(), name: 'Turn ' + b.name, type: 'rotate', part: b.id, followers: [], origin: originOf(b.id), axis: axisOf(b.id), min: -180, max: 180, speed: 60, loop: false, driver: ja.id, ratio });
}

// ---------------- matching part ----------------
const SPUR_LIKE = ['gear', 'compoundgear', 'intermittent', 'sectorgear'];

export function hasMatch(n) {
  if (!n || n.type === 'group') return false;
  return true; // every solid can at least get a matching cut-out
}

export function matchingDialog(id) {
  const n = sc.node(id);
  if (!n) return;
  const t = n.type;
  const opts = [];
  const mod = T(n, 'module', 1.5);
  if (SPUR_LIKE.includes(t) || t === 'lantern') opts.push('Spur gear', 'Internal ring gear', 'Gear rack');
  if (t === 'gear') opts.push('Lantern pinion');
  if (t === 'rack') opts.push('Spur gear');
  if (t === 'ringgear') opts.push('Spur gear');
  if (t === 'bevelgear') opts.push('Bevel gear');
  if (t === 'worm') opts.push('Worm wheel');
  if (t === 'crowngear') opts.push('Spur pinion');
  if (t === 'nautilus' || t === 'ellgear') opts.push('Mating gear');
  if (t === 'sprocket' || t === 'gt2pulley') opts.push('Same-chain ' + (t === 'sprocket' ? 'sprocket' : 'pulley'));
  if (t === 'pbolt') opts.push('Printable nut');
  if (t === 'pnut') opts.push('Printable bolt');
  if (t === 'pbayonet') opts.push('Other half');
  if (!sc.isHole(n)) opts.push('Cut-out that fits it (hole)');
  const teethish = (o) => /Spur|ring|Lantern|Bevel|Worm wheel|sprocket|pulley|pinion/i.test(o);
  const f = ui.form([
    { key: 'kind', label: 'Make', type: 'select', options: opts, def: opts[0] },
    { key: 'teeth', label: 'Teeth', def: t === 'ringgear' ? 15 : Math.max(8, Math.round(T(n, 'teeth', 20))), min: 5, max: 400, step: 1, showIf: (v) => teethish(v.kind) && !/ring/i.test(v.kind) },
    { key: 'ringTeeth', label: 'Ring teeth', def: Math.round(T(n, 'teeth', 20) * 2 + 12), min: 12, max: 600, step: 1, showIf: (v) => /ring/i.test(v.kind) },
    { key: 'clr', label: 'Clearance', def: 0.3, min: 0, max: 3, step: 0.05, showIf: (v) => /Cut-out/.test(v.kind) },
    { key: 'joint', label: 'Link them in Motion (turns together)', type: 'bool', def: true, showIf: (v) => teethish(v.kind) || /Mating/.test(v.kind) },
  ]);
  const info = h('div', { class: 'note' });
  const upd = () => {
    const v = f.vals;
    const z1 = T(n, 'teeth', 20);
    if (/Spur|Lantern|pinion/.test(v.kind) && z1) {
      const z2 = v.teeth;
      info.textContent = `Ratio ${z1}:${z2} — the new gear turns ${round(z1 / z2, 3)}× as fast. Centres ${round(mod * (z1 + z2) / 2, 2)} mm apart.`;
    } else if (/ring/i.test(v.kind)) info.textContent = `Gear sits inside the ring, centres ${round(mod * (v.ringTeeth - z1) / 2, 2)} mm apart.`;
    else info.textContent = '';
  };
  f.el.addEventListener('input', upd); f.el.addEventListener('change', upd); f.el.addEventListener('click', () => setTimeout(upd, 0)); upd();
  ui.modal('Make a matching part for “' + n.name + '”', h('div', {}, f.el, info), [{ label: 'Cancel' }, { label: 'Make it', primary: true, run: () => make(n, f.vals) }]);
}

function make(n, v) {
  const t = n.type;
  const P = n.params || {};
  const mod = T(n, 'module', 1.5);
  const z1 = T(n, 'teeth', 20);
  const h1 = T(n, 'h', 6);
  let m = null, ratio = null;
  const common = { module: mod, h: h1, pa: P.pa ?? 20, backlash: P.backlash ?? 0.15 };
  switch (v.kind) {
    case 'Spur gear': {
      const z2 = Math.round(v.teeth);
      if (t === 'rack') {
        // pinion above the rack's teeth
        const base = T(n, 'base', 4);
        const rp = mod * z2 / 2;
        m = addRelative(n, 'gear', { ...common, teeth: z2, h: T(n, 'h', 6), bore: 5 }, [0, -T(n, 'h', 6) / 2, base + 1.25 * mod + rp], [-90, 0, 0], `Gear ${z2}T`);
        ratio = null;
        break;
      }
      if (t === 'ringgear') {
        const zr = z1;
        const c = mod * (zr - z2) / 2;
        m = addRelative(n, 'gear', { ...common, teeth: z2, bore: 5 }, [c, 0, 0], [0, 0, 180 - 180 / z2], `Gear ${z2}T`);
        ratio = zr / z2;
        break;
      }
      const c = mod * (z1 + z2) / 2;
      // half a tooth turn so a tooth meets a gap
      m = addRelative(n, 'gear', { ...common, teeth: z2, bore: P.bore ?? 5 }, [c, 0, 0], [0, 0, 180 - 180 / z2], `Gear ${z2}T`);
      ratio = -z1 / z2;
      break;
    }
    case 'Lantern pinion': {
      const z2 = Math.round(v.teeth);
      const c = mod * (z1 + z2) / 2;
      m = addRelative(n, 'lantern', { module: mod, teeth: z2, h: h1, bore: 5 }, [c, 0, -2], [0, 0, 180 - 180 / z2], `Lantern ${z2}`);
      ratio = -z1 / z2;
      break;
    }
    case 'Internal ring gear': {
      const zr = Math.round(v.ringTeeth);
      if (zr < z1 + 10) { toast('The ring needs at least ' + (z1 + 10) + ' teeth for this gear.', 'error'); return; }
      const c = mod * (zr - z1) / 2;
      m = addRelative(n, 'ringgear', { module: mod, teeth: zr, h: h1, rim: 4 }, [-c, 0, 0], [0, 0, 0], `Ring ${zr}T`);
      ratio = z1 / zr;
      break;
    }
    case 'Gear rack': {
      const nt = Math.max(8, Math.round(z1 * 1.2));
      const base = 4;
      const rp = mod * z1 / 2;
      const p = Math.PI * mod;
      const L = p * nt;
      // line a rack gap up with the gear tooth nearest the bottom
      const ang = (k) => (2 * Math.PI * k) / z1;
      let best = 0, bd = 9;
      for (let k = 0; k < z1; k++) { const d = Math.abs(((ang(k) + Math.PI / 2 + 4 * Math.PI) % (2 * Math.PI)) - 0); const dd = Math.min(d, 2 * Math.PI - d); if (dd < bd) { bd = dd; best = k; } }
      const xt = rp * Math.cos(ang(best));
      const g0 = -L / 2;
      const shift = xt - g0 - p * Math.round((xt - g0) / p);
      m = addRelative(n, 'rack', { module: mod, teeth: nt, h: h1, base }, [shift, -rp + base + 1.25 * mod, h1 / 2], [90, 0, 0], `Rack ${nt}T`);
      break;
    }
    case 'Bevel gear': {
      const z2 = Math.round(v.teeth);
      const gA = Math.atan(z1 / z2) * 180 / Math.PI;
      n.params.cone = round(gA, 2);
      const rpA = mod * z1 / 2;
      const apexA = rpA / Math.tan(gA * Math.PI / 180);
      const apexB = (mod * z2 / 2) / Math.tan((90 - gA) * Math.PI / 180);
      m = addRelative(n, 'bevelgear', { module: mod, teeth: z2, h: h1, cone: round(90 - gA, 2), bore: P.bore ?? 5 }, [apexB, 0, apexA], [0, -90, 0], `Bevel ${z2}T`);
      ratio = -z1 / z2;
      break;
    }
    case 'Worm wheel': {
      const z2 = Math.round(v.teeth);
      const d = T(n, 'd', 12), L = T(n, 'len', 30), starts = T(n, 'starts', 1);
      const c = (d + mod * z2) / 2;
      const lead = Math.atan((mod * starts) / d) * 180 / Math.PI;
      m = addRelative(n, 'gear', { module: mod, teeth: z2, h: Math.min(10, d * 0.8), style: 'Helical', helix: round(lead, 1), bore: 5 }, [c, Math.min(10, d * 0.8) / 2, L / 2], [90, 0, 0], `Worm wheel ${z2}T`);
      ratio = starts / z2;
      break;
    }
    case 'Spur pinion': {
      const z2 = Math.round(v.teeth);
      const rpc = mod * z1 / 2, fw = T(n, 'face', 8), base = T(n, 'base', 4);
      const rpp = mod * z2 / 2;
      m = addRelative(n, 'gear', { module: mod, teeth: z2, h: fw, bore: 4 }, [rpc - fw / 2, 0, base + 1.25 * mod + rpp], [0, 90, 0], `Pinion ${z2}T`);
      ratio = z1 / z2;
      break;
    }
    case 'Mating gear': {
      const c = t === 'nautilus' ? T(n, 'rmin', 15) + T(n, 'rmax', 30) : 2 * T(n, 'a', 25);
      m = addRelative(n, t, { ...P, mate: !P.mate }, [c, 0, 0], [0, 0, 180], 'Mating ' + n.name);
      ratio = -1;
      break;
    }
    case 'Same-chain sprocket':
    case 'Same-chain pulley': {
      const z2 = Math.round(v.teeth);
      const box = localBox(n);
      const w = box ? box[1][0] - box[0][0] : 40;
      m = addRelative(n, t, { ...P, teeth: z2 }, [w * 1.8 + 20, 0, 0], [0, 0, 0], `${t === 'sprocket' ? 'Sprocket' : 'Pulley'} ${z2}T`);
      ratio = z1 / z2;
      break;
    }
    case 'Printable nut':
      m = addRelative(n, 'pnut', { d: P.d, pitch: P.pitch, profile: P.profile, clr: P.clr }, [T(n, 'd', 12) * 2.5, 0, 0], [0, 0, 0], 'Nut for ' + n.name);
      break;
    case 'Printable bolt':
      m = addRelative(n, 'pbolt', { d: P.d, pitch: P.pitch, profile: P.profile, clr: P.clr }, [T(n, 'd', 12) * 2.5, 0, 0], [0, 0, 0], 'Bolt for ' + n.name);
      break;
    case 'Other half':
      m = addRelative(n, 'pbayonet', { ...P, part: P.part === 'Male' ? 'Female' : 'Male' }, [T(n, 'd', 30) * 1.8, 0, 0], [0, 0, 0]);
      break;
    case 'Cut-out that fits it (hole)': {
      const box = localBox(n);
      if (!box) return;
      const sz = [0, 1, 2].map((i) => box[1][i] - box[0][i]);
      const c = structuredClone(n);
      sc.walk([c], (x) => { x.id = sc.newId(); });
      c.hole = true; c.name = 'Cut-out for ' + n.name;
      const k = sz.map((s) => (s + 2 * v.clr) / Math.max(s, 0.01));
      c.scale = c.scale.map((s, i) => s * k[i]);
      sc.siblings(n.id).push(c);
      // grow about the part's centre
      const ctr = [0, 1, 2].map((i) => (box[0][i] + box[1][i]) / 2);
      sc.setWorld(c, mul(sc.worldMatrix(n.id), mul(trs(ctr, [0, 0, 0], k), trs(ctr.map((x) => -x), [0, 0, 0], [1, 1, 1]))), sc.parentWorld(n.id));
      c.scale = c.scale.map((s, i) => s); // keep
      m = c;
      toast(`Cut-out made ${v.clr} mm bigger all round. Move it into the part that should hold it, then Group.`, 'ok', 6000);
      break;
    }
  }
  if (!m) return;
  if (v.joint && ratio != null && !sc.parentOf(n.id)) addJointPair(n, m, ratio);
  S.sel = new Set([m.id]);
  C.commit('Matching part');
  if (ratio != null && v.joint) toast('Made and linked — switch to Motion and press Play to watch them turn.', 'ok', 5000);
}

// ---------------- gear ratio calculator ----------------
function bestPairs(target, zmin, zmax) {
  const out = [];
  for (let a = zmin; a <= zmax; a++) for (let b = zmin; b <= zmax; b++) {
    const r = b / a;
    out.push({ a, b, r, err: Math.abs(r - target) / target });
  }
  out.sort((x, y) => x.err - y.err || (x.a + x.b) - (y.a + y.b));
  const seen = new Set();
  return out.filter((p) => { const k = round(p.r, 4); if (seen.has(k)) return false; seen.add(k); return true; }).slice(0, 6);
}
function bestTwoStage(target, zmin, zmax) {
  const out = [];
  const s = Math.sqrt(target);
  for (let a = zmin; a <= Math.min(zmax, 40); a++) for (let b = Math.max(zmin, Math.floor(a * s * 0.6)); b <= Math.min(zmax, Math.ceil(a * s * 1.6)); b++)
    for (let c = zmin; c <= Math.min(zmax, 40); c++) {
      const d = Math.round((target * a * c) / b);
      if (d < zmin || d > zmax) continue;
      const r = (b / a) * (d / c);
      out.push({ a, b, c, d, r, err: Math.abs(r - target) / target, size: a + b + c + d });
    }
  out.sort((x, y) => x.err - y.err || x.size - y.size);
  return out.slice(0, 5);
}
function parseRatio(s) {
  s = String(s).trim();
  const m = s.match(/^([\d.]+)\s*[:/]\s*([\d.]+)$/);
  if (m) return +m[1] / +m[2];
  return +s;
}

export function calculatorDialog() {
  const f = ui.form([
    { key: 'ratio', label: 'Reduction wanted (e.g. 3 or 3:1)', type: 'text', def: '4:1', help: 'Bigger than 1 = slower and stronger output. Less than 1 = faster output.' },
    { key: 'module', label: 'Module (tooth size)', def: 1.5, min: 0.3, max: 10, step: 0.1, help: '1–2 mm prints well. Gears only mesh with the same module.' },
    { key: 'zmin', label: 'Fewest teeth', def: 10, min: 6, max: 60, step: 1, help: 'Below about 10 teeth, printed gears get weak.' },
    { key: 'zmax', label: 'Most teeth', def: 80, min: 10, max: 300, step: 1 },
    { key: 'rpm', label: 'Input speed', def: 100, min: 0, max: 100000, step: 1, unit: 'rpm' },
    { key: 'torque', label: 'Input torque', def: 1, min: 0, max: 100000, step: 0.1, unit: 'N·m' },
  ]);
  const out = h('div', { class: 'gc-out' });
  let lastKey = '';
  const draw = () => {
    const v = f.vals;
    const key = JSON.stringify(v);
    if (key === lastKey) return; // blur/change with nothing new: keep the table (and its buttons) as is
    lastKey = key;
    const R = parseRatio(v.ratio);
    out.innerHTML = '';
    if (!(R > 0)) { out.append(h('div', { class: 'note bad' }, 'Type a ratio like 3, 0.5 or 5:2.')); return; }
    const m = v.module;
    const row = (cells, add) => h('tr', {}, ...cells.map((c) => h('td', {}, c)), h('td', {}, add ? h('button', { class: 'mini', onclick: add }, 'Add gears') : ''));
    const one = bestPairs(R, v.zmin, v.zmax);
    out.append(h('div', { class: 'group-title' }, 'One pair'),
      h('table', { class: 'gc' }, h('tr', {}, ...['Driver', 'Driven', 'Ratio', 'Error', 'Centres', 'Output', ''].map((x) => h('th', {}, x))),
        ...one.map((p) => row([p.a + 'T', p.b + 'T', round(p.r, 4) + ':1', round(p.err * 100, 2) + '%', round(m * (p.a + p.b) / 2, 2) + ' mm', `${round(v.rpm / p.r, 2)} rpm · ${round(v.torque * p.r, 2)} N·m · reversed`],
          () => addTrain([p.a, p.b], m)))));
    if (R > 4 || R < 0.25 || one[0].err > 0.005) {
      const two = bestTwoStage(R, v.zmin, v.zmax);
      if (two.length) out.append(h('div', { class: 'group-title' }, 'Two stages (compound gear in the middle)'),
        h('table', { class: 'gc' }, h('tr', {}, ...['Gears', 'Ratio', 'Error', 'Output', ''].map((x) => h('th', {}, x))),
          ...two.map((p) => row([`${p.a}→${p.b} · ${p.c}→${p.d}`, round(p.r, 4) + ':1', round(p.err * 100, 2) + '%', `${round(v.rpm / p.r, 2)} rpm · ${round(v.torque * p.r, 2)} N·m · same direction`],
            () => addTrain([p.a, p.b, p.c, p.d], m)))));
    }
    out.append(h('p', { class: 'note' }, 'Torque ignores friction (printed gears lose roughly 5–10 % per pair). Centres = distance between the shafts.'));
  };
  f.el.addEventListener('input', draw); f.el.addEventListener('change', draw); f.el.addEventListener('click', () => setTimeout(draw, 0));
  draw();
  ui.modal('Gear ratio calculator', h('div', { class: 'gc-wrap' }, f.el, out), [{ label: 'Close', primary: true }], { wide: true });
}

// rotation for gear B so a gap meets gear A's tooth: A is turned rotA, B sits in direction theta from A
function meshRot(rotA, zA, theta, zB) {
  const pa = 360 / zA;
  const f = ((((theta - rotA) / pa) % 1) + 1) % 1;
  return theta + 180 + (f + 0.5) * (360 / zB);
}

// build a working train: [a,b] = one pair, [a,b,c,d] = a → (b+c compound) → d
function addTrain(z, m) {
  const h1 = 6;
  const g1 = sc.makeNode('gear', { module: m, teeth: z[0], h: h1, bore: 5 });
  g1.name = `Driver ${z[0]}T`;
  S.nodes.push(g1);
  let made = [g1];
  if (z.length === 2) {
    const g2 = addRelative(g1, 'gear', { module: m, teeth: z[1], h: h1, bore: 5 }, [m * (z[0] + z[1]) / 2, 0, 0], [0, 0, 180 - 180 / z[1]], `Driven ${z[1]}T`);
    addJointPair(g1, g2, -z[0] / z[1]);
    made.push(g2);
  } else {
    const c1 = m * (z[0] + z[1]) / 2;
    const cg = addRelative(g1, 'compoundgear', { module: m, teeth: z[1], teeth2: z[2], h: h1, h2: h1, bore: 5 }, [c1, 0, 0], [0, 0, 180 - 180 / z[1]], `Compound ${z[1]}/${z[2]}T`);
    // the last gear meshes with the small half, up one layer, off at 90°
    const c2 = m * (z[2] + z[3]) / 2;
    const rotC = 180 - 180 / z[1];
    const g3 = addRelative(g1, 'gear', { module: m, teeth: z[3], h: h1, bore: 5 }, [c1, c2, h1], [0, 0, meshRot(rotC, z[2], 90, z[3])], `Output ${z[3]}T`);
    addJointPair(g1, cg, -z[0] / z[1]);
    S.joints.push({ id: sc.newId(), name: 'Turn ' + g3.name, type: 'rotate', part: g3.id, followers: [], origin: [sc.worldMatrix(g3.id)[12], sc.worldMatrix(g3.id)[13], sc.worldMatrix(g3.id)[14]], axis: [0, 0, 1], min: -180, max: 180, speed: 60, loop: false, driver: S.joints.find((j) => j.part === cg.id).id, ratio: -z[2] / z[3] });
    made.push(cg, g3);
  }
  S.sel = new Set(made.map((x) => x.id));
  C.commit('Add gear train');
  Promise.all(made.map(whenBuilt)).then(() => C.fit?.(made.map((x) => x.id)));
  toast('Gears added and linked. Switch to Motion and press Play to see them turn.', 'ok', 5000);
}
