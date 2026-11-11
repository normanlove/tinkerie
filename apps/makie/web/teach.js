// Tinkerie teaching mode: point at any control to see a beginner-friendly
// explanation of what it does. F2 turns it on and off (remembered per app in a
// cookie, because some apps use a new port each run and cookies ignore the port).
//
// Each app loads its explanations first, as window.TEACH = { app, intro, tips },
// then this file. tips is { "label": "explanation" } or { "label": ["Title",
// "explanation"] }. Labels match a control's data-help, aria-label, title, its
// <label> or its own short text, case-insensitively, ignoring keyboard
// shortcuts in brackets and trailing "…". A control with a title but no tip
// shows its title, so every labelled button explains itself.
//
// This file is shared by all Tinkerie apps (a copy lives in each app's web
// folder). Created by nrmluvdev.
(function () {
  "use strict";
  const T = window.TEACH || { tips: {} };
  const app = T.app || document.title.split(/[ —-]/)[0] || "app";
  const cookie = "tinkerie_teach_" + app.toLowerCase().replace(/[^a-z0-9]/g, "");
  const norm = (s) => String(s || "")
    .replace(/\(([^)]*)\)\s*$/, (m, inner) => (/^(ctrl|shift|alt|f\d|[a-z0-9?]$|del|esc|home|space)/i.test(inner.trim()) ? "" : m))
    .replace(/[…\.:]+$/, "").replace(/^[^\p{L}\p{N}]+/u, "").replace(/\s+/g, " ").trim().toLowerCase();

  const tips = new Map();
  // tips every app shares (the Look & accessibility button comes from look.js)
  tips.set(norm("Look & accessibility"), ["Look & accessibility", "21 colour themes (dark, light, high contrast and colour-blind safe) and a choice of accent colour, text and control size, touch-friendly controls, reduced motion, a stronger focus outline, easier-to-read text and underlined links. Saved for this app. Ctrl+, opens it too."]);
  for (const [k, v] of Object.entries(T.tips || {})) {
    for (const key of k.split("|")) tips.set(norm(key), Array.isArray(v) ? v : [key.trim(), v]);
  }

  let on = !document.cookie.split("; ").includes(cookie + "=off");
  const remember = () => { document.cookie = `${cookie}=${on ? "on" : "off"}; path=/; max-age=315360000; SameSite=Strict`; };

  // ---------------------------------------------------------------- the card
  const css = document.createElement("style");
  css.textContent = `
#tk-teach{all:initial;display:block;box-sizing:border-box;position:fixed;left:50%;bottom:44px;transform:translate(-50%,8px);z-index:2147483000;
  max-width:min(640px,calc(100vw - 32px));padding:11px 16px 12px;border-radius:14px;pointer-events:none;
  background:rgba(21,26,36,.97);border:1px solid #2f3749;box-shadow:0 14px 40px rgba(0,0,0,.45);
  color:#e8ebf2;font:13.5px/1.5 "Segoe UI Variable Text","Segoe UI",system-ui,-apple-system,sans-serif;
  opacity:0;transition:opacity .16s,transform .16s;-webkit-user-select:none;user-select:none}
#tk-teach.tk-top{top:64px;bottom:auto;transform:translate(-50%,-8px)}
#tk-teach.tk-show{opacity:1;transform:translate(-50%,0)}
#tk-teach *{all:revert;box-sizing:border-box;font:inherit;color:inherit;margin:0;padding:0}
#tk-teach b{color:#fff;font-weight:650}
#tk-teach .tk-body{display:block}
#tk-teach .tk-h{display:flex;flex-direction:row;align-items:center;gap:8px;margin-bottom:2px}
#tk-teach .tk-tag{display:inline-block;flex:none;margin-left:auto;font-size:11px;color:#8d95aa;border:1px solid #2f3749;border-radius:6px;padding:0 6px}
@media (prefers-reduced-motion:reduce){#tk-teach{transition:none}}`;
  document.head.appendChild(css);
  const card = document.createElement("div");
  card.id = "tk-teach";
  card.setAttribute("role", "status");
  card.setAttribute("aria-live", "polite");
  const mount = () => document.body && !card.isConnected && document.body.appendChild(card);
  mount();
  document.addEventListener("DOMContentLoaded", mount);

  const esc = (s) => String(s).replace(/[&<>"]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" }[c]));
  let hideTimer, shown = null;
  function show(title, text, sticky) {
    mount();
    card.innerHTML = `<div class="tk-h">💡 <b>${esc(title)}</b><span class="tk-tag">Teaching mode · F2</span></div>` +
      `<div class="tk-body">${esc(text).replace(/\n/g, "<br>")}</div>`;
    card.classList.add("tk-show");
    clearTimeout(hideTimer);
    if (sticky) hideTimer = setTimeout(hide, sticky);
  }
  function hide() { card.classList.remove("tk-show"); shown = null; }

  // ---------------------------------------------------------- finding a tip
  function labelFor(el) {
    const out = [];
    if (el.dataset) {
      if (el.dataset.help) out.push(el.dataset.help);
      if (el.dataset.tip) out.push(el.dataset.tip);
    }
    if (el.id) out.push("#" + el.id);
    const aria = el.getAttribute && el.getAttribute("aria-label");
    if (aria) out.push(aria);
    if (el.labels && el.labels.length) out.push(el.labels[0].innerText);
    if (/^(INPUT|SELECT|TEXTAREA)$/.test(el.tagName)) {
      // The label next to the field: a sibling of the field or of one of its
      // nearest wrappers, never one belonging to another field.
      let n = el;
      for (let i = 0; i < 3 && n; i++, n = n.parentElement) {
        let p = n.previousElementSibling;
        while (p && !/^(LABEL|SPAN|DIV|B|H4|H5)$/.test(p.tagName)) p = p.previousElementSibling;
        if (p && !p.querySelector("input,select,textarea") && p.innerText.trim() && p.innerText.trim().length < 60) {
          out.push(p.innerText.trim().split("\n")[0]);
          break;
        }
      }
      if (el.placeholder) out.push(el.placeholder);
    }
    const title = el.getAttribute && el.getAttribute("title");
    if (title) out.push(title);
    const txt = (el.innerText || "").trim();
    if (txt && txt.length < 300) out.push(txt.split("\n")[0]);
    return { keys: out, title };
  }

  function find(target) {
    let el = target, depth = 0, fallback = null;
    while (el && el !== document.body && depth < 7) {
      if (el.nodeType === 1 && el !== card) {
        const { keys, title } = labelFor(el);
        for (const k of keys) {
          const t = tips.get(norm(k));
          if (t) return t;
        }
        // A labelled control with no tip of its own explains itself by its title.
        if (!fallback && title && title.length > 2) {
          const name = norm((el.innerText || "").split("\n")[0]) ? (el.innerText || "").trim().split("\n")[0] : "";
          fallback = name && name.length < 40 && name !== title ? [name, title] : [title.replace(/\s*\([^)]*\)\s*$/, ""), title];
        }
      }
      el = el.parentElement;
      depth++;
    }
    return fallback;
  }

  let lastTarget = null;
  document.addEventListener("mouseover", (e) => {
    if (!on || e.target === lastTarget) return;
    lastTarget = e.target;
    const t = find(e.target);
    if (!t) { if (shown) hideTimer = setTimeout(hide, 350); return; }
    card.classList.toggle("tk-top", e.clientY > innerHeight * 0.62);
    if (shown !== t) { shown = t; show(t[0], t[1]); }
    else clearTimeout(hideTimer);
  }, true);
  document.addEventListener("mouseleave", () => on && hide());

  // ------------------------------------------------------------- on and off
  function announce() {
    show(on ? "Teaching mode is on" : "Teaching mode is off",
      on ? (T.intro || "Point at anything to learn what it does.") + "\nPress F2 to turn these explanations off."
         : "Press F2 to turn the explanations back on.", 5000);
  }
  document.addEventListener("keydown", (e) => {
    if (e.key !== "F2" || e.ctrlKey || e.altKey || e.metaKey) return;
    e.preventDefault();
    on = !on;
    remember();
    hide();
    announce();
  }, true);
  // say it's on the first time only, so it doesn't greet you every time the app opens
  let greeted = false; try { greeted = !!localStorage.getItem("tinkerie.teach.greeted"); localStorage.setItem("tinkerie.teach.greeted", "1"); } catch (e) {}
  if (on && !greeted) setTimeout(announce, 900);
  window.tinkerieTeach = { show, hide, isOn: () => on };
})();
