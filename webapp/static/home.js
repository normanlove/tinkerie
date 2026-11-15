// Tinkerie web's home page: "Clear all my Tinkerie web data", for shared
// computers. Deletes every app's saved files (IndexedDB "tinkerie-<app>"), its
// settings and cookies, and the devices the apps were allowed to use.
// Made by nrmluvdev.
(function () {
  'use strict';
  const apps = (document.currentScript.dataset.apps || '').split(',').filter(Boolean);
  const btn = document.getElementById('clearAll'), msg = document.getElementById('clearMsg');
  if (!btn) return;
  const gone = (name) => new Promise((r) => { const q = indexedDB.deleteDatabase(name); q.onsuccess = q.onerror = q.onblocked = () => r(); });
  btn.onclick = async () => {
    if (!confirm('Delete everything every Tinkerie app has saved in this browser (files, autosaves and settings), and forget the devices they may use? Close any open Tinkerie apps first, and download anything you want to keep.')) return;
    try {
      const names = new Set(apps.map((a) => 'tinkerie-' + a));
      if (indexedDB.databases) for (const d of await indexedDB.databases()) if (d.name && d.name.startsWith('tinkerie-')) names.add(d.name);
      await Promise.all([...names].map(gone));
      // the website itself keeps nothing, so everything here is the apps'
      localStorage.clear();
      sessionStorage.clear();
      for (const c of document.cookie.split('; ')) { const n = c.split('=')[0]; if (n) document.cookie = n + '=; path=/; max-age=0'; }
      const lists = [navigator.serial && navigator.serial.getPorts(), navigator.usb && navigator.usb.getDevices(), navigator.hid && navigator.hid.getDevices()];
      for (const l of lists) if (l) for (const d of await l) { try { await d.forget(); } catch (e) { /* older browsers */ } }
      msg.textContent = 'Done: everything Tinkerie web saved is cleared from this browser.';
    } catch (e) { msg.textContent = 'Couldn’t clear everything: ' + e.message; }
  };
})();
