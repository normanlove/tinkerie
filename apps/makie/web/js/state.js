// Shared application state + a tiny event bus.
export const S = {
  catalog: [],          // shape definitions from the engine
  byType: {},           // type -> definition
  fonts: [],
  nodes: [],            // top-level scene nodes
  sel: new Set(),       // selected node ids
  tool: 'move',         // move | rotate | scale
  mode: 'select',       // select | place | measure
  settings: {
    bed: [220, 220, 250],
    grid: 1,
    rotSnap: 15,
    quality: 1,
    theme: 'dark',
    showBed: true,
    autoDepth: true,
    ortho: false,
    lockRatio: false,
  },
  view: { overhang: false, section: false, sectionAxis: 2, sectionPos: 10, wire: false },
  meshData: {},         // meshRef -> Float32Array (imported meshes, saved in projects)
  projectName: 'Untitled part',
  dirty: false,
};

const handlers = {};
export function on(ev, fn) { (handlers[ev] ||= []).push(fn); }
export function emit(ev, ...a) { (handlers[ev] || []).forEach((f) => f(...a)); }

export const PALETTE = ['#ef4444', '#f97316', '#eab308', '#22c55e', '#14b8a6', '#06b6d4', '#3b82f6', '#6366f1', '#a855f7', '#ec4899', '#f43f5e', '#84cc16', '#64748b', '#e2e8f0', '#1e293b', '#a16207'];

export function hexToRgb(h) {
  h = (h || '#888888').replace('#', '');
  if (h.length === 3) h = h.split('').map((c) => c + c).join('');
  const n = parseInt(h, 16);
  return [((n >> 16) & 255) / 255, ((n >> 8) & 255) / 255, (n & 255) / 255];
}

export function toast(msg, kind = 'info', ms = 3200) {
  const t = document.createElement('div');
  t.className = 'toast ' + kind;
  t.textContent = msg;
  document.getElementById('toasts').appendChild(t);
  setTimeout(() => t.classList.add('out'), ms);
  setTimeout(() => t.remove(), ms + 400);
}

export function hint(msg) {
  const h = document.getElementById('hint');
  if (h) h.innerHTML = msg;
}
