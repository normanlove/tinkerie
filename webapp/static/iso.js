// Tinkerie web: keeps each app's browser storage to itself. On the desktop
// every app is its own little web server on its own port, so the browser keeps
// their localStorage, sessionStorage and cookies apart; on tinkerie.net they
// all share one address. The service worker puts this script first in every
// page an app serves, and it gives that app its own named part of each store
// (keys "tk.<app>.<key>", cookies "tk_<app>_<name>"), so apps never see or
// overwrite each other's settings, autosaves or teaching-mode choices.
// Made by nrmluvdev.
(function () {
  'use strict';
  var me = document.currentScript && new URL(document.currentScript.src).searchParams.get('app');
  if (!me || !/^[a-z]+$/.test(me) || window.__tkIso) return;
  window.__tkIso = me;
  var pre = 'tk.' + me + '.';

  function wrap(real) {
    function keys() {
      var out = [];
      for (var i = 0; i < real.length; i++) { var k = real.key(i); if (k && k.indexOf(pre) === 0) out.push(k.slice(pre.length)); }
      return out;
    }
    var api = {
      getItem: function (k) { return real.getItem(pre + k); },
      setItem: function (k, v) { real.setItem(pre + k, String(v)); },
      removeItem: function (k) { real.removeItem(pre + k); },
      clear: function () { keys().forEach(function (k) { real.removeItem(pre + k); }); },
      key: function (i) { var k = keys()[i]; return k === undefined ? null : k; },
      get length() { return keys().length; },
    };
    return new Proxy(api, {
      get: function (t, p) { if (p in t) return t[p]; if (typeof p !== 'string') return undefined; var v = real.getItem(pre + p); return v === null ? undefined : v; },
      set: function (t, p, v) { if (typeof p === 'string') real.setItem(pre + p, String(v)); return true; },
      deleteProperty: function (t, p) { if (typeof p === 'string') real.removeItem(pre + p); return true; },
      has: function (t, p) { return p in t || (typeof p === 'string' && real.getItem(pre + p) !== null); },
      ownKeys: function () { return keys(); },
      getOwnPropertyDescriptor: function (t, p) { var v = typeof p === 'string' ? real.getItem(pre + p) : null; return v === null ? undefined : { value: v, enumerable: true, configurable: true, writable: true }; },
    });
  }
  function swap(name) {
    try {
      var real = window[name];
      if (!real) return null;
      var w = wrap(real);
      Object.defineProperty(window, name, { configurable: true, get: function () { return w; } });
      return real;
    } catch (e) { return null; }
  }
  var ls = swap('localStorage');
  swap('sessionStorage');

  // Settings saved before apps had their own part (Tinkerie web 0.31): copy
  // this app's once, plus the shared look and teaching-mode greeting.
  try {
    if (ls && ls.getItem('tk.' + me + ':moved') === null) {
      var mine = { makie: ['partsmith.', 'makie.'], wirie: ['wirie.', 'keebie.'], fixie: ['fixie.'], codie: ['codie.'], probie: ['probie.'],
        tunie: ['tunie.'], flashie: ['flashie.', 'card.'], scopie: ['scopie.'], meterie: ['meterie.'] }[me] || [];
      for (var i = 0; i < ls.length; i++) {
        var k = ls.key(i);
        if (!k || k.indexOf('tk.') === 0) continue;
        if (k === 'tinkerie.look' || k === 'tinkerie.teach.greeted' || mine.some(function (m) { return k.indexOf(m) === 0; })) {
          if (ls.getItem(pre + k) === null) ls.setItem(pre + k, ls.getItem(k));
        }
      }
      ls.setItem('tk.' + me + ':moved', '1');
    }
  } catch (e) {}

  // Cookies (teaching mode remembers on/off in one)
  try {
    var d = Object.getOwnPropertyDescriptor(Document.prototype, 'cookie');
    var cpre = 'tk_' + me + '_';
    Object.defineProperty(document, 'cookie', {
      configurable: true,
      get: function () {
        return d.get.call(document).split('; ').filter(function (c) { return c.indexOf(cpre) === 0; })
          .map(function (c) { return c.slice(cpre.length); }).join('; ');
      },
      set: function (v) { d.set.call(document, cpre + String(v).replace(/^\s+/, '')); },
    });
  } catch (e) {}
})();
