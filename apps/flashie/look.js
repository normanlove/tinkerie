// look.js: Tinkerie's shared "Look & accessibility" settings, used by every app
// (each app has its own copy of this file; keep them identical to shared/look.js).
// Created by nrmluvdev.
//
//   Themes: Dark (each app's own look), nine more dark themes, seven light ones,
//   high contrast (dark and light) and colour-blind safe (dark and light), plus a
//   choice of accent colour that replaces the app's main colour.
//     The app's own stylesheets are re-coloured on the fly: every colour in them
//     is mapped by what it's used for (background, text, border), so new screens
//     pick the theme up without extra work.
//   Text size, extra-strong focus outlines, easier-to-read text, underlined links,
//   reduced motion, and touch-friendly controls (bigger buttons, plus finger
//   gestures on drawing areas: drag, long-press for the right-click menu,
//   pinch to zoom, two-finger drag to pan).
//
// <html data-look-slot="CSS selector"> puts the settings button in the app's own
// header; without it the button floats in the bottom-left corner.
// <html data-look-touch="mouse"> turns finger gestures on drawing areas into the
// mouse events the app already understands ("pointer" when it handles touch itself;
// an element can say data-look-touch="pointer" for itself too).
// <html data-look-accent="--name"> names the custom property holding the app's main
// colour, for the accent swap (otherwise --accent, --acc or --primary).
// <html data-look-button="off"> leaves the button out (for an app that draws its
// own settings, like Meterie); the touch gestures still work.
// Settings are remembered per app. Ctrl+, opens the panel.
(() => {
  "use strict";
  const H = document.documentElement;
  const KEY = "tinkerie.look";
  const DEF = { theme: "dark", accent: "app", size: 100, touch: "auto", motion: "auto", focus: false, readable: false, underline: false };
  let L = Object.assign({}, DEF);
  try { Object.assign(L, JSON.parse(localStorage.getItem(KEY) || "{}")); } catch (e) {}
  const save = () => { try { localStorage.setItem(KEY, JSON.stringify(L)); } catch (e) {} };
  const coarse = () => matchMedia("(pointer: coarse)").matches || (navigator.maxTouchPoints > 0 && !matchMedia("(pointer: fine)").matches);
  const listeners = [];

  // ------------------------------------------------------------ colours
  const NAMED = { white: [255, 255, 255], black: [0, 0, 0] };
  const COLOR_RE = /#[0-9a-fA-F]{3,8}\b|rgba?\([^)]*\)|hsla?\([^)]*\)|\b(?:white|black)\b/g;
  function parse(t) {
    t = t.trim();
    if (NAMED[t]) return [...NAMED[t], 1];
    if (t[0] === "#") {
      let h = t.slice(1); if (h.length <= 4) h = h.split("").map(c => c + c).join("");
      const n = parseInt(h.slice(0, 6), 16); return [n >> 16 & 255, n >> 8 & 255, n & 255, h.length === 8 ? parseInt(h.slice(6), 16) / 255 : 1];
    }
    const m = t.match(/^(rgba?|hsla?)\(([^)]*)\)$/); if (!m) return null;
    const p = m[2].replace(/\//, " ").split(/[\s,]+/).filter(Boolean);
    if (p.some(x => /var|calc/.test(x))) return null;
    const a = p[3] != null ? (p[3].endsWith("%") ? parseFloat(p[3]) / 100 : +p[3]) : 1;
    if (m[1][0] === "r") return [...p.slice(0, 3).map(x => x.endsWith("%") ? parseFloat(x) * 2.55 : +x), a];
    const [r, g, b] = hsl2rgb(parseFloat(p[0]) / 360, parseFloat(p[1]) / 100, parseFloat(p[2]) / 100); return [r, g, b, a];
  }
  function rgb2hsl(r, g, b) {
    r /= 255; g /= 255; b /= 255; const mx = Math.max(r, g, b), mn = Math.min(r, g, b), l = (mx + mn) / 2;
    if (mx === mn) return [0, 0, l];
    const d = mx - mn, s = l > 0.5 ? d / (2 - mx - mn) : d / (mx + mn);
    const h = mx === r ? (g - b) / d + (g < b ? 6 : 0) : mx === g ? (b - r) / d + 2 : (r - g) / d + 4;
    return [h / 6, s, l];
  }
  function hsl2rgb(h, s, l) {
    if (!s) return [l * 255, l * 255, l * 255];
    const q = l < 0.5 ? l * (1 + s) : l + s - l * s, p = 2 * l - q;
    const f = t => { t = (t + 1) % 1; return 255 * (t < 1 / 6 ? p + (q - p) * 6 * t : t < 1 / 2 ? q : t < 2 / 3 ? p + (q - p) * (2 / 3 - t) * 6 : p); };
    return [f(h + 1 / 3), f(h), f(h - 1 / 3)];
  }
  const fmt = (r, g, b, a) => a >= 0.999 ? `rgb(${Math.round(r)}, ${Math.round(g)}, ${Math.round(b)})` : `rgba(${Math.round(r)}, ${Math.round(g)}, ${Math.round(b)}, ${+a.toFixed(3)})`;
  const clamp = (x, a, b) => Math.max(a, Math.min(b, x));

  // what a colour is used for, from the property (or custom property's name)
  function roleOf(prop) {
    if (prop.startsWith("--")) {
      if (/shadow/.test(prop)) return "shadow";
      if (/(^|-)(bg|back|panel|card|surface|rail|bar|base|sheet|well|input|field|shade|hover|sel|tint|chip|pane)/.test(prop)) return "bg";
      if (/(^|-)(text|fg|ink|dim|mute|sub|faint|label|title|hint)/.test(prop)) return "text";
      if (/(^|-)(line|border|edge|sep|stroke|grid|rule|outline)/.test(prop)) return "line";
      return "auto";
    }
    if (/shadow/.test(prop)) return "shadow";
    if (/^background|^fill$/.test(prop)) return "bg";
    if (/^border|^outline|^stroke$|column-rule/.test(prop)) return "line";
    if (/^color$|caret|text-decoration|accent-color/.test(prop)) return "text";
    return "auto";
  }

  // the themes. base: built on the app's dark look, or turned light; h/s: the tint
  // given to backgrounds, borders and text (hue in degrees, saturation 0-1)
  const THEMES = [
    { id: "dark", name: "Dark", grp: "Dark", base: "dark", sw: ["#11151c", "#3cc8f0"] },
    { id: "midnight", name: "Midnight", grp: "Dark", base: "dark", h: 225, s: 0.42, sw: ["#0d1330", "#6d8cff"] },
    { id: "ocean", name: "Ocean", grp: "Dark", base: "dark", h: 198, s: 0.5, sw: ["#071c26", "#2fc4e8"] },
    { id: "forest", name: "Forest", grp: "Dark", base: "dark", h: 145, s: 0.32, sw: ["#0b1d14", "#46d68c"] },
    { id: "plum", name: "Plum", grp: "Dark", base: "dark", h: 285, s: 0.3, sw: ["#1a1023", "#c48cff"] },
    { id: "ember", name: "Ember", grp: "Dark", base: "dark", h: 12, s: 0.32, sw: ["#211210", "#ff8a5c"] },
    { id: "mocha", name: "Mocha", grp: "Dark", base: "dark", h: 30, s: 0.22, sw: ["#1d1712", "#e0a86a"] },
    { id: "slate", name: "Slate", grp: "Dark", base: "dark", h: 210, s: 0.16, sw: ["#161b21", "#8fb3d9"] },
    { id: "graphite", name: "Graphite", grp: "Dark", base: "dark", h: 0, s: 0, sw: ["#161616", "#bdbdbd"] },
    { id: "black", name: "True black", grp: "Dark", base: "black", sw: ["#000000", "#3cc8f0"] },
    { id: "light", name: "Light", grp: "Light", base: "light", sw: ["#f3f5f8", "#0a6f99"] },
    { id: "paper", name: "Paper", grp: "Light", base: "light", h: 45, s: 0.12, sw: ["#f7f5ef", "#556070"] },
    { id: "sepia", name: "Sepia", grp: "Light", base: "light", h: 36, s: 0.45, sw: ["#f4ead6", "#9a5b16"] },
    { id: "sky", name: "Sky", grp: "Light", base: "light", h: 205, s: 0.55, sw: ["#e9f3fb", "#1d6fb8"] },
    { id: "mint", name: "Mint", grp: "Light", base: "light", h: 155, s: 0.4, sw: ["#e8f6ef", "#16805a"] },
    { id: "rose", name: "Rose", grp: "Light", base: "light", h: 340, s: 0.45, sw: ["#fbecf1", "#b0305f"] },
    { id: "lavender", name: "Lavender", grp: "Light", base: "light", h: 265, s: 0.4, sw: ["#f1edfb", "#6a46c4"] },
    { id: "contrast", name: "High contrast", grp: "Accessibility", base: "contrast", sw: ["#000000", "#ffd400"] },
    { id: "contrastl", name: "High contrast light", grp: "Accessibility", base: "contrastl", sw: ["#ffffff", "#0033cc"] },
    { id: "cvd", name: "Colour-blind safe", grp: "Accessibility", base: "dark", cvd: true, sw: ["#0072b2", "#e69f00"] },
    { id: "cvdl", name: "Colour-blind safe light", grp: "Accessibility", base: "light", cvd: true, sw: ["#e8eef5", "#e69f00"] },
  ];
  // the app's main accent colour can be swapped for one of these
  const ACCENTS = [["app", "The app's own", null], ["blue", "Blue", 215], ["cyan", "Cyan", 190], ["teal", "Teal", 168], ["green", "Green", 140], ["lime", "Lime", 88], ["amber", "Amber", 40], ["orange", "Orange", 24], ["red", "Red", 355], ["pink", "Pink", 330], ["purple", "Purple", 272], ["indigo", "Indigo", 240]];
  let appAccentHue = null; // worked out from the app's stylesheets
  const hueDist = (a, b) => { const d = Math.abs(a - b) % 360; return Math.min(d, 360 - d); };

  // map one colour for a theme
  function mapColor([r, g, b, a], role, theme) {
    const T = THEMES.find(t => t.id === theme) || THEMES[0];
    let [h, s, l] = rgb2hsl(r, g, b);
    const neutral = s < 0.28 || l < 0.1 || l > 0.94;
    if (role === "auto") role = neutral ? (l < 0.4 ? "bg" : "text") : "accent";
    const surface = neutral || (role === "bg" && l < 0.22 && T.base !== "dark");
    // a different accent: colours close to the app's own accent take the new hue
    const acc = ACCENTS.find(x => x[0] === L.accent);
    if (!neutral && acc && acc[2] != null && appAccentHue != null && hueDist(h * 360, appAccentHue) < 32) { h = acc[2] / 360; s = Math.max(s, 0.55); }
    if (T.base === "light") {
      if (role === "shadow") return fmt(r, g, b, a * 0.35);
      if (neutral && a < 0.6 && (l > 0.85 || l < 0.12)) { const v = 255 - (r + g + b) / 3; return fmt(v, v, v, a); } // see-through overlays flip
      if (surface) {
        if (role === "bg") l = clamp(0.94 + (l - 0.04) * 0.45, 0.9, 1);
        else if (role === "line") l = clamp(0.95 - l * 0.5, 0.62, 0.9);
        else l = l < 0.25 ? 0.98 : clamp(1 - l * 1.03, 0.06, 0.58); // dark text sat on a bright button: keep it readable on the darker one
        s = Math.min(s, 0.25);
      } else if (role !== "bg" || a > 0.5) l = Math.min(l, 0.44); // accents dark enough to read on white
    } else if (T.base === "black") {
      if (surface && role === "bg") { l = l < 0.16 ? l * 0.15 : l * 0.8; s = 0; }
      else if (surface && role === "line") l = l * 0.85;
    } else if (T.base === "contrast" || T.base === "contrastl") {
      const dark = T.base === "contrast";
      if (role === "shadow") return fmt(0, 0, 0, 0);
      if (neutral && a < 0.6) return dark ? fmt(r, g, b, Math.min(1, a * 2.2)) : fmt(255 - r, 255 - g, 255 - b, Math.min(1, a * 2.2));
      if (surface || (role === "bg" && l < 0.22)) {
        s = 0;
        if (role === "bg") l = dark ? (l < 0.2 ? 0 : l < 0.5 ? 0.1 : 1) : (l < 0.2 ? 1 : l < 0.5 ? 0.9 : 0);
        else if (role === "line") { l = dark ? 0.8 : 0.15; a = 1; }
        else l = dark ? (l > 0.3 ? 1 : 0) : (l > 0.3 ? 0 : 1); // dark text was on a bright button
      } else { s = 1; l = dark ? clamp(l, 0.55, 0.68) : clamp(l, 0.22, 0.34); }
    }
    // the theme's tint on backgrounds, borders and text
    if (T.h != null && surface && role !== "shadow") {
      h = T.h / 360; s = T.s * (role === "bg" ? 1 : role === "line" ? 0.75 : 0.3);
      if (T.base === "dark" && role === "bg" && T.s > 0) l = Math.min(0.4, l * 1.2 + 0.015); // a touch lighter, so the tint shows
    }
    // colour-blind safe (red/green): greens become blue, reds become orange (Okabe-Ito-like)
    if (T.cvd && !neutral) {
      const deg = h * 360;
      if (deg >= 70 && deg < 175) { h = 205 / 360; s = Math.max(s, 0.6); }
      else if (deg >= 330 || deg < 18) { h = 30 / 360; l = clamp(l, T.base === "light" ? 0.38 : 0.45, 0.62); }
    }
    const [R, G, B] = hsl2rgb(h, s, l);
    return fmt(R, G, B, a);
  }
  const isLight = () => { const T = THEMES.find(t => t.id === L.theme); return !!T && (T.base === "light" || T.base === "contrastl"); };
  const mapValue = (v, role, theme) => v.replace(COLOR_RE, t => { const c = parse(t); return c ? mapColor(c, role, theme) : t; });

  // every colour declaration in the app's stylesheets, with its original value
  const decls = []; const seen = new WeakSet();
  function scanRules(rules) {
    for (const r of rules) {
      if (r.cssRules && !r.style) { scanRules(r.cssRules); continue; }
      if (!r.style || seen.has(r.style)) continue;
      if (r.selectorText && /\.lk-|#lkPanel|#lkBtn/.test(r.selectorText)) continue; // the panel's own styles
      if (r.selectorText && /(^|[\s,])body\.light\b|\[data-theme=["']?light/.test(r.selectorText)) continue; // an app's own light theme is already light
      seen.add(r.style);
      const clipText = /text/.test(r.style.getPropertyValue("background-clip") + r.style.getPropertyValue("-webkit-background-clip")); // gradient text
      for (let i = 0; i < r.style.length; i++) {
        const p = r.style[i], v = r.style.getPropertyValue(p);
        if (v && COLOR_RE.test(v)) decls.push({ st: r.style, p, v, pr: r.style.getPropertyPriority(p), role: clipText && /^background/.test(p) ? "text" : roleOf(p) });
        COLOR_RE.lastIndex = 0;
      }
      if (r.cssRules) scanRules(r.cssRules);
    }
  }
  function scan() { for (const sh of document.styleSheets) { try { if (sh.ownerNode && sh.ownerNode.id === "lkStyle") continue; scanRules(sh.cssRules); } catch (e) { /* another origin */ } } }
  // the app's main accent: the most-used colourful hue among its accent-like custom properties
  function findAccent() {
    if (appAccentHue != null) return;
    const hueOf = v => { for (const t of v.match(COLOR_RE) || []) { const c = parse(t); if (!c) continue; const [h, s, l] = rgb2hsl(c[0], c[1], c[2]); if (s >= 0.4 && l >= 0.25 && l <= 0.85) return Math.round(h * 360); } return null; };
    // a custom property named for it says it best
    for (const n of [H.dataset.lookAccent, "--accent", "--acc", "--primary", "--accent-color", "--brand-accent", "--cy", "--or"]) {
      const d = decls.find(d => d.p === n); const h = d && hueOf(d.v); if (h != null) { appAccentHue = h; return; }
    }
    const votes = new Map();
    for (const d of decls) {
      const named = /^--(acc|accent|primary|brand|cy|main|hi|blue|or|orange)\b/.test(d.p) ? 3 : d.p.startsWith("--") ? 0 : 1;
      if (!named) continue;
      for (const t of d.v.match(COLOR_RE) || []) { const c = parse(t); if (!c) continue; const [h, s, l] = rgb2hsl(c[0], c[1], c[2]); if (s < 0.4 || l < 0.25 || l > 0.85) continue; const k = Math.round(h * 36) * 10; votes.set(k, (votes.get(k) || 0) + named); }
    }
    let best = null, n = 0; for (const [k, v] of votes) if (v > n) { n = v; best = k; }
    appAccentHue = best;
  }
  function applyTheme() {
    scan(); findAccent();
    const plain = L.theme === "dark" && (L.accent || "app") === "app";
    for (const d of decls) d.st.setProperty(d.p, plain ? d.v : mapValue(d.v, d.role, L.theme), d.pr);
    H.style.colorScheme = isLight() ? "light" : "dark";
  }

  // ------------------------------------------------------------ the rest
  const css = `
  :root { --lk-bg: #151a23; --lk-fg: #e8edf5; --lk-dim: #9aa6b8; --lk-line: #2c3545; --lk-acc: #3cc8f0; --lk-acc-fg: #04131a; }
  html.lk-light { --lk-bg: #ffffff; --lk-fg: #12161d; --lk-dim: #586273; --lk-line: #d5dbe4; --lk-acc: #0a6f99; --lk-acc-fg: #fff; }
  html[data-look-theme=contrast] { --lk-bg: #000; --lk-fg: #fff; --lk-dim: #fff; --lk-line: #fff; --lk-acc: #ffd400; --lk-acc-fg: #000; }
  html[data-look-theme=contrastl] { --lk-bg: #fff; --lk-fg: #000; --lk-dim: #000; --lk-line: #000; --lk-acc: #0033cc; --lk-acc-fg: #fff; }
  :focus-visible { outline: 2px solid var(--lk-acc); outline-offset: 2px; }
  html.lk-focus :focus-visible, html[data-look-theme^=contrast] :focus-visible { outline: 3px solid #ffbf00 !important; outline-offset: 2px !important; box-shadow: 0 0 0 6px rgba(0,0,0,.85) !important; }
  html[data-look-theme^=contrast] button, html[data-look-theme^=contrast] select, html[data-look-theme^=contrast] input { border-width: 1.5px; }
  html.lk-readable body, html.lk-readable button, html.lk-readable input, html.lk-readable select, html.lk-readable textarea { font-family: Verdana, "DejaVu Sans", "Segoe UI", system-ui, sans-serif !important; letter-spacing: .02em; word-spacing: .06em; }
  html.lk-readable p, html.lk-readable li, html.lk-readable small { line-height: 1.6 !important; }
  html.lk-underline a { text-decoration: underline !important; }
  html.lk-still *, html.lk-still *::before, html.lk-still *::after { animation-duration: .001ms !important; animation-iteration-count: 1 !important; transition-duration: .001ms !important; scroll-behavior: auto !important; }
  html.lk-touch button, html.lk-touch select, html.lk-touch .btn, html.lk-touch [role=button], html.lk-touch input:not([type=checkbox]):not([type=radio]):not([type=range]) { min-height: 40px; }
  html.lk-touch button:not(:empty), html.lk-touch .btn { padding-left: max(10px, .9em); padding-right: max(10px, .9em); }
  html.lk-touch input[type=checkbox], html.lk-touch input[type=radio] { min-width: 22px; min-height: 22px; }
  html.lk-touch input[type=range] { min-height: 32px; }
  html.lk-touch ::-webkit-scrollbar { width: 14px; height: 14px; }
  html.lk-touch .lk-surface { touch-action: none; -webkit-user-select: none; user-select: none; }
  html.lk-touch * { -webkit-tap-highlight-color: transparent; }
  #lkBtn { font: 600 13px/1 system-ui, sans-serif; display: inline-flex; align-items: center; gap: 6px; cursor: pointer; border-radius: 8px; border: 1px solid var(--lk-line); background: var(--lk-bg); color: var(--lk-fg); padding: 6px 9px; }
  #lkBtn.lk-float { position: fixed; left: 10px; bottom: 34px; z-index: 99990; box-shadow: 0 4px 14px rgba(0,0,0,.35); }
  #lkBtn svg { width: 16px; height: 16px; } #lkBtn:not(.lk-float) .lk-btnt { display: none; } #lkBtn:not(.lk-float) { padding: 6px; }
  #lkPanel { position: fixed; inset: 0; z-index: 2147483600; display: flex; align-items: center; justify-content: center; background: rgba(0,0,0,.45); font: 14px/1.4 system-ui, "Segoe UI", sans-serif; }
  #lkPanel .lk-card { background: var(--lk-bg); color: var(--lk-fg); border: 1px solid var(--lk-line); border-radius: 14px; width: min(560px, calc(100vw - 24px)); max-height: calc(100vh - 24px); overflow: auto; padding: 18px 18px 14px; box-shadow: 0 20px 60px rgba(0,0,0,.5); }
  #lkPanel h2 { margin: 0 0 2px; font-size: 18px; } #lkPanel .lk-sub { color: var(--lk-dim); margin: 0 0 12px; font-size: 13px; }
  #lkPanel h3 { margin: 14px 0 6px; font-size: 12px; text-transform: uppercase; letter-spacing: .06em; color: var(--lk-dim); }
  #lkPanel .lk-seg { display: flex; flex-wrap: wrap; gap: 6px; }
  #lkPanel .lk-seg button, #lkPanel .lk-row button { font: inherit; color: var(--lk-fg); background: transparent; border: 1px solid var(--lk-line); border-radius: 9px; padding: 8px 11px; min-height: 40px; cursor: pointer; }
  #lkPanel .lk-seg button[aria-pressed=true] { background: var(--lk-acc); color: var(--lk-acc-fg); border-color: var(--lk-acc); font-weight: 700; }
  #lkPanel .lk-th { display: inline-flex; align-items: center; gap: 7px; } #lkPanel .lk-sw { width: 18px; height: 18px; border-radius: 5px; border: 1px solid var(--lk-line); flex: none; }
  #lkPanel label.lk-chk { display: flex; gap: 10px; align-items: flex-start; padding: 7px 0; cursor: pointer; } #lkPanel label.lk-chk input { width: 20px; height: 20px; margin: 1px 0 0; accent-color: var(--lk-acc); flex: none; }
  #lkPanel label.lk-chk small { display: block; color: var(--lk-dim); }
  #lkPanel .lk-row { display: flex; justify-content: space-between; gap: 8px; margin-top: 14px; }
  #lkPanel .lk-row .lk-done { background: var(--lk-acc); color: var(--lk-acc-fg); border-color: var(--lk-acc); font-weight: 700; }`;
  const style = document.createElement("style"); style.id = "lkStyle"; style.textContent = css;
  (document.head || H).appendChild(style);

  function apply() {
    H.dataset.lookTheme = L.theme; H.classList.toggle("lk-light", isLight());
    H.classList.toggle("lk-touch", L.touch === "on" || (L.touch === "auto" && coarse()));
    H.classList.toggle("lk-still", L.motion === "on" || (L.motion === "auto" && matchMedia("(prefers-reduced-motion: reduce)").matches));
    H.classList.toggle("lk-focus", !!L.focus);
    H.classList.toggle("lk-readable", !!L.readable);
    H.classList.toggle("lk-underline", !!L.underline);
    H.style.zoom = L.size === 100 ? "" : String(L.size / 100);
    applyTheme();
    for (const f of listeners) try { f(Object.assign({}, L)); } catch (e) {}
    dispatchEvent(new CustomEvent("tinkerie-look", { detail: Object.assign({}, L) }));
  }

  // ------------------------------------------------------------ the panel
  const ICON = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true"><circle cx="12" cy="12" r="9"/><path d="M12 3a9 9 0 0 0 0 18z" fill="currentColor"/></svg>`;
  const seg = (k, opts) => `<div class="lk-seg" role="group">${opts.map(([v, t, sw]) => `<button type="button" data-k="${k}" data-v="${v}" aria-pressed="${String(L[k]) === String(v)}">${sw ? `<span class="lk-th"><span class="lk-sw" style="background:${sw}"></span>${t}</span>` : t}</button>`).join("")}</div>`;
  const chk = (k, t, sub) => `<label class="lk-chk"><input type="checkbox" data-c="${k}" ${L[k] ? "checked" : ""}><span>${t}<small>${sub}</small></span></label>`;
  let panel = null, opener = null;
  function render() {
    panel.innerHTML = `<div class="lk-card" role="dialog" aria-modal="true" aria-labelledby="lkT">
      <h2 id="lkT">Look &amp; accessibility</h2><p class="lk-sub">Saved for this app. Ctrl+, opens this panel.</p>
      ${["Dark", "Light", "Accessibility"].map(g => `<h3>${g === "Accessibility" ? "Easier to see" : g + " themes"}</h3>${seg("theme", THEMES.filter(t => t.grp === g).map(t => [t.id, t.name, `linear-gradient(135deg,${t.sw[0]} 55%,${t.sw[1]} 55%)`]))}`).join("")}
      <h3>Accent colour</h3>${seg("accent", ACCENTS.map(([id, n, hue]) => [id, n, hue == null ? "conic-gradient(#3cc8f0,#ff8a3c,#9b6cff,#3cc8f0)" : `hsl(${hue} 75% 52%)`]))}
      <p class="lk-sub" style="margin:6px 0 0">Swaps the app's main colour (buttons, highlights) and keeps warning and status colours as they are.</p>
      <h3>Text and controls size</h3>${seg("size", [[90, "90%"], [100, "100%"], [115, "115%"], [130, "130%"], [150, "150%"]])}
      <h3>Touch-friendly controls</h3>${seg("touch", [["auto", "Automatic"], ["on", "On"], ["off", "Off"]])}
      <p class="lk-sub" style="margin:6px 0 0">Bigger buttons, and on drawing areas: drag with a finger, press and hold for the right-click menu, pinch to zoom, two fingers to pan.</p>
      <h3>Movement</h3>${seg("motion", [["auto", "Like my system"], ["on", "Reduce motion"], ["off", "Full animation"]])}
      <h3>Reading</h3>
      ${chk("focus", "Extra-strong focus outline", "A thick yellow outline shows where the keyboard is.")}
      ${chk("readable", "Easier-to-read text", "A plainer font with more space between letters and lines.")}
      ${chk("underline", "Underline links", "So links don't rely on colour alone.")}
      <div class="lk-row"><button type="button" data-reset>Reset to default</button><button type="button" class="lk-done" data-done>Done</button></div></div>`;
    panel.querySelectorAll("[data-k]").forEach(b => b.onclick = () => { const k = b.dataset.k; L[k] = typeof DEF[k] === "number" ? +b.dataset.v : b.dataset.v; save(); apply(); render(); panel.querySelector(`[data-k="${k}"][data-v="${b.dataset.v}"]`).focus(); });
    panel.querySelectorAll("[data-c]").forEach(c => c.onchange = () => { L[c.dataset.c] = c.checked; save(); apply(); });
    panel.querySelector("[data-reset]").onclick = () => { L = Object.assign({}, DEF); save(); apply(); render(); };
    panel.querySelector("[data-done]").onclick = close;
  }
  function open() {
    if (panel) return; opener = document.activeElement; window.tinkerieTeach && tinkerieTeach.hide();
    panel = document.createElement("div"); panel.id = "lkPanel"; document.body.appendChild(panel); render();
    panel.addEventListener("mousedown", e => { if (e.target === panel) close(); });
    panel.addEventListener("keydown", e => {
      if (e.key === "Escape") { e.stopPropagation(); close(); }
      if (e.key === "Tab") { const f = [...panel.querySelectorAll("button, input")]; const i = f.indexOf(document.activeElement); if (e.shiftKey && i <= 0) { e.preventDefault(); f[f.length - 1].focus(); } else if (!e.shiftKey && i === f.length - 1) { e.preventDefault(); f[0].focus(); } }
      e.stopPropagation(); // keep the app's own shortcuts out of the panel
    });
    panel.querySelector("[aria-pressed=true]").focus();
  }
  function close() { if (!panel) return; panel.remove(); panel = null; if (opener && opener.focus) opener.focus(); }
  addEventListener("keydown", e => { if (H.dataset.lookButton !== "off" && e.key === "," && (e.ctrlKey || e.metaKey) && !e.altKey) { e.preventDefault(); panel ? close() : open(); } }, true);

  function placeButton() {
    if (H.dataset.lookButton === "off" || document.getElementById("lkBtn")) return; // the app has its own settings for these
    const b = document.createElement("button"); b.id = "lkBtn"; b.type = "button";
    b.title = "Look & accessibility: themes, text size, touch (Ctrl+,)"; b.setAttribute("aria-label", "Look and accessibility settings"); b.dataset.help = "Look & accessibility";
    b.innerHTML = ICON + `<span class="lk-btnt">Look</span>`; b.onclick = open;
    const slot = H.dataset.lookSlot && document.querySelector(H.dataset.lookSlot);
    if (slot) slot.appendChild(b); else { b.classList.add("lk-float"); document.body.appendChild(b); }
  }

  // ------------------------------------------------------------ screen readers
  // icon-only buttons get a spoken name from their tooltip
  function label(root) {
    for (const el of (root || document).querySelectorAll("button:not([aria-label]), [role=button]:not([aria-label]), a:not([aria-label])")) {
      if (el.textContent.trim()) continue;
      const t = el.getAttribute("title") || el.dataset.help || el.dataset.tip; if (t) el.setAttribute("aria-label", t);
    }
    for (const c of (root || document).querySelectorAll("canvas:not([role])")) { c.setAttribute("role", "img"); if (!c.getAttribute("aria-label")) c.setAttribute("aria-label", c.getAttribute("title") || "Drawing area"); }
  }

  // ------------------------------------------------------------ fingers
  // Drawing areas (canvases, big SVGs, [data-touch]) get gestures; apps that only
  // understand the mouse get mouse events made from them.
  const mouseApp = H.dataset.lookTouch !== "pointer";
  const isSurface = el => el && el.closest && el.closest("canvas, [data-touch], svg.lk-surface");
  function markSurfaces() {
    for (const s of document.querySelectorAll("canvas, [data-touch]")) s.classList.add("lk-surface");
    for (const s of document.querySelectorAll("svg")) { const r = s.getBoundingClientRect(); if (r.width > 220 && r.height > 160 && !s.closest("button")) s.classList.add("lk-surface"); }
  }
  const pts = new Map(); let gm = true, mode = null, lastTap = 0, press = null, pinch = null;
  const fire = (type, x, y, button, buttons, target) => {
    const t = target || document.elementFromPoint(x, y) || document.body;
    t.dispatchEvent(new MouseEvent(type, { bubbles: true, cancelable: true, composed: true, view: window, clientX: x, clientY: y, screenX: x, screenY: y, button, buttons, detail: type === "dblclick" ? 2 : 1 }));
  };
  addEventListener("pointerdown", e => {
    if (e.pointerType !== "touch" || !H.classList.contains("lk-touch")) return;
    const surf = isSurface(e.target); if (!surf) return;
    pts.set(e.pointerId, { x: e.clientX, y: e.clientY, x0: e.clientX, y0: e.clientY, surf });
    if (pts.size === 1) {
      mode = "one"; gm = mouseApp && !surf.closest("[data-look-touch=pointer]"); // this area's own way of handling touch
      if (gm) { e.preventDefault(); fire("mousedown", e.clientX, e.clientY, 0, 1, e.target); }
      clearTimeout(press); press = setTimeout(() => { // press and hold: the right-click menu
        const p = pts.get(e.pointerId); if (!p || mode !== "one" || Math.hypot(p.x - p.x0, p.y - p.y0) > 10) return;
        mode = "held"; if (gm) fire("mouseup", p.x, p.y, 0, 0);
        fire("contextmenu", p.x, p.y, 2, 2);
      }, 550);
    } else if (pts.size === 2) {
      clearTimeout(press);
      const [a, b] = [...pts.values()];
      if (mode === "one" && gm) fire("mouseup", a.x, a.y, 0, 0); // a second finger: a pinch or pan, not a drag
      if (mode === "one" && !gm) { const [id0] = pts.keys(); a.surf.dispatchEvent(new PointerEvent("pointerup", { bubbles: true, cancelable: true, pointerId: id0, pointerType: "touch", isPrimary: true, clientX: a.x, clientY: a.y, button: 0, buttons: 0 })); }
      mode = "two"; pinch = { d: Math.hypot(a.x - b.x, a.y - b.y), cx: (a.x + b.x) / 2, cy: (a.y + b.y) / 2 };
      if (gm) fire("mousedown", pinch.cx, pinch.cy, 2, 2);
      e.stopImmediatePropagation(); e.preventDefault();
    } else { e.stopImmediatePropagation(); }
  }, { capture: true, passive: false });
  addEventListener("pointermove", e => {
    const p = pts.get(e.pointerId); if (!p) return;
    p.x = e.clientX; p.y = e.clientY;
    if (mode === "one") { if (gm) { e.preventDefault(); fire("mousemove", p.x, p.y, 0, 1); } if (Math.hypot(p.x - p.x0, p.y - p.y0) > 10) clearTimeout(press); return; }
    if (mode === "two" && pts.size >= 2) {
      e.stopImmediatePropagation(); e.preventDefault();
      const [a, b] = [...pts.values()], d = Math.hypot(a.x - b.x, a.y - b.y), cx = (a.x + b.x) / 2, cy = (a.y + b.y) / 2;
      if (Math.abs(d - pinch.d) > 2) { // pinch: the app's mouse-wheel zoom
        const t = document.elementFromPoint(cx, cy) || p.surf;
        t.dispatchEvent(new WheelEvent("wheel", { bubbles: true, cancelable: true, clientX: cx, clientY: cy, deltaY: (pinch.d - d) * 4, deltaMode: 0, ctrlKey: !gm }));
        pinch.d = d;
      }
      if (gm) fire("mousemove", cx, cy, 2, 2); // two fingers: right-drag, which pans
      pinch.cx = cx; pinch.cy = cy;
    }
  }, { capture: true, passive: false });
  const up = e => {
    const p = pts.get(e.pointerId); if (!p) return;
    pts.delete(e.pointerId); clearTimeout(press);
    if (mode === "one" && gm) {
      e.preventDefault(); fire("mouseup", p.x, p.y, 0, 0);
      const now = Date.now(); if (now - lastTap < 320 && Math.hypot(p.x - p.x0, p.y - p.y0) < 10) fire("dblclick", p.x, p.y, 0, 0); lastTap = now;
    }
    if (mode === "two") { e.stopImmediatePropagation(); if (pts.size < 2) { if (gm) fire("mouseup", pinch.cx, pinch.cy, 2, 0); mode = pts.size ? "rest" : null; } }
    if (!pts.size) mode = null;
  };
  addEventListener("pointerup", up, { capture: true, passive: false });
  addEventListener("pointercancel", up, { capture: true, passive: false });
  // the right-click menu that two-finger panning would otherwise open
  addEventListener("contextmenu", e => { if (mode === "two") { e.preventDefault(); e.stopImmediatePropagation(); } }, true);

  // ------------------------------------------------------------ start
  function start() {
    placeButton(); markSurfaces(); label(); apply();
    let t = 0;
    new MutationObserver(muts => {
      clearTimeout(t); t = setTimeout(() => {
        const added = muts.some(m => [...m.addedNodes].some(n => n.nodeName === "STYLE" || n.nodeName === "LINK"));
        if (added && L.theme !== "dark") applyTheme();
        markSurfaces(); label();
      }, 120);
    }).observe(document.documentElement, { childList: true, subtree: true });
    matchMedia("(pointer: coarse)").addEventListener?.("change", apply);
  }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", start); else start();
  addEventListener("load", () => { if (L.theme !== "dark") applyTheme(); }); // stylesheets that arrived late

  window.tinkerieLook = { get: () => Object.assign({}, L), isLight, open, accentHue: () => appAccentHue, onChange: f => listeners.push(f), mapColor: (c, role) => { const p = parse(c); return p && L.theme !== "dark" ? mapColor(p, role || "auto", L.theme) : c; } };
})();
