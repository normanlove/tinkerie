// Tinkerie web: loads one app (Go compiled to WebAssembly) into this page,
// answers its frame's requests, and adds a small bar with the things a
// desktop gives an app for free (files, devices). Made by nrmluvdev.
(async function () {
  'use strict';
  const app = document.body.dataset.app;
  const $ = (s) => document.querySelector(s);
  const status = (t, bad) => { const el = $('#tkStatus'); el.textContent = t; el.classList.toggle('bad', !!bad); };

  if (!('serviceWorker' in navigator) || !window.isSecureContext) {
    status('This needs a secure page (https://) and a modern browser.', true);
    return;
  }
  // One id per open tab, so tabs never mix: the service worker sends each
  // app frame's requests only to the tab with its id. The id is random and
  // unguessable, kept for reloads, and claimed with a browser lock while the
  // tab is open, so a duplicated tab (which copies the address and
  // sessionStorage) notices and takes a new one.
  const newId = () => Array.from(crypto.getRandomValues(new Uint8Array(16)), (b) => b.toString(16).padStart(2, '0')).join('');
  const u = new URL(location.href);
  let sid = u.searchParams.get('tksid') || sessionStorage.getItem('tksid-' + app);
  if (!sid || !/^[0-9a-f]{32}$/.test(sid)) sid = newId();
  const claim = (id) => new Promise((resolve) => {
    if (!navigator.locks) return resolve(true);
    navigator.locks.request('tinkerie-tab-' + id, { ifAvailable: true }, (lock) => {
      resolve(!!lock);
      return lock ? new Promise(() => {}) : undefined; // held until the tab closes or reloads
    });
  });
  while (!(await claim(sid))) sid = newId();
  sessionStorage.setItem('tksid-' + app, sid);
  // The service worker finds this tab by the id in its address, so it must be
  // there from the start (a rewritten address isn't seen by service workers).
  if (u.searchParams.get('tksid') !== sid) { u.searchParams.set('tksid', sid); location.replace(u); return; }

  // The same app in another tab shares this browser's saved files: say so.
  if (navigator.locks) {
    navigator.locks.request('tinkerie-app-' + app, { ifAvailable: true }, (lock) => {
      if (lock) return new Promise(() => {});
      const w = $('#tkTwice');
      if (w) w.hidden = false;
      // when the other tab closes, this one takes over and the warning goes
      navigator.locks.request('tinkerie-app-' + app, () => { if (w) w.hidden = true; return new Promise(() => {}); });
    });
  }

  status('Starting ' + document.title + '…');
  const reg = await navigator.serviceWorker.register('../sw.js', { scope: '../' });
  await navigator.serviceWorker.ready;

  const pending = [];
  let ready = false;
  navigator.serviceWorker.addEventListener('message', (ev) => {
    const m = ev.data;
    if (!m || m.tk !== 'fetch' || m.sid !== sid) return;
    ready ? handle(m, ev.ports[0]) : pending.push([m, ev.ports[0]]);
  });
  function handle(m, port) {
    const stop = globalThis.tkFetch(m.method, m.path, m.headers, m.body,
      (status, headers) => port.postMessage({ t: 'head', status, headers }),
      (chunk) => port.postMessage({ t: 'chunk', chunk }, [chunk.buffer]),
      (err) => { port.postMessage({ t: 'end', err }); });
    port.onmessage = (e) => { if (e.data && e.data.t === 'cancel') stop(); };
  }

  await tkFS.open(app);
  globalThis.tkReady = () => {
    ready = true;
    for (const [m, p] of pending.splice(0)) handle(m, p);
    const f = $('#tkFrame');
    f.src = 'ui/?tksid=' + sid;
    f.addEventListener('load', () => { $('#tkLoading').hidden = true; }, { once: true });
    status('');
  };
  globalThis.tkFailed = (msg) => status(msg, true);

  const go = new Go();
  let wasm;
  try {
    const res = fetch('app.wasm');
    wasm = WebAssembly.instantiateStreaming
      ? await WebAssembly.instantiateStreaming(res, go.importObject).catch(async () => WebAssembly.instantiate(await (await fetch('app.wasm')).arrayBuffer(), go.importObject))
      : await WebAssembly.instantiate(await (await res).arrayBuffer(), go.importObject);
  } catch (e) {
    status('Couldn’t load the app: ' + e.message, true);
    return;
  }
  go.run(wasm.instance).then(() => status(document.title + ' stopped — reload the page to start it again.', true));

  // ---- Files: anything the app saves can be downloaded; files can be added.
  const home = '/home/web';
  const isData = (p) => /\.(json|log|lock|tmp|cfg|ini)$/i.test(p) || /\/(window|cache|\.cache)\//.test(p);
  let seen = Date.now();
  const offered = new Set();
  setInterval(() => {
    const fresh = tkFS.changedSince(home, seen).filter((f) => !isData(f.path) && !offered.has(f.path + f.mtime));
    seen = Date.now() - 50;
    for (const f of fresh) { offered.add(f.path + f.mtime); toast(f); }
  }, 700);
  function download(path) {
    const data = tkFS.read(path);
    if (!data) return;
    const a = document.createElement('a');
    a.href = URL.createObjectURL(new Blob([data]));
    a.download = path.split('/').pop();
    document.body.appendChild(a); a.click(); a.remove();
    setTimeout(() => URL.revokeObjectURL(a.href), 4000);
  }
  function toast(f) {
    const t = document.createElement('div');
    t.className = 'tk-toast';
    const name = f.path.split('/').pop();
    t.innerHTML = `<span>Saved <b></b></span><button>Download</button><button class="x" title="Dismiss">✕</button>`;
    t.querySelector('b').textContent = name;
    t.querySelector('button').onclick = () => { download(f.path); t.remove(); };
    t.querySelector('.x').onclick = () => t.remove();
    $('#tkToasts').appendChild(t);
    setTimeout(() => t.remove(), 15000);
  }
  const fmt = (n) => n < 1024 ? n + ' B' : n < 1 << 20 ? (n / 1024).toFixed(1) + ' KB' : (n / 1048576).toFixed(1) + ' MB';
  function showFiles() {
    const list = tkFS.list(home).filter((f) => !isData(f.path)).sort((a, b) => b.mtime - a.mtime);
    const body = $('#tkFilesList');
    body.innerHTML = list.length ? '' : '<p class="tk-dim">Nothing saved yet. Files this app saves show up here to download.</p>';
    for (const f of list) {
      const row = document.createElement('div');
      row.className = 'tk-file';
      row.innerHTML = `<span class="n"></span><span class="tk-dim"></span><button>Download</button>`;
      row.querySelector('.n').textContent = f.path.slice(home.length + 1);
      row.querySelector('.tk-dim').textContent = fmt(f.size) + ' · ' + new Date(f.mtime).toLocaleString();
      row.querySelector('button').onclick = () => download(f.path);
      body.appendChild(row);
    }
    $('#tkFiles').hidden = false;
  }
  // ---- Report a bug: the website's form, filled in with this app and browser
  const bug = $('#tkBug');
  if (bug) {
    const where = /Edg\//.test(navigator.userAgent) ? 'Tinkerie web (in Edge)' : /Chrome\//.test(navigator.userAgent) ? 'Tinkerie web (in Chrome)' : 'Tinkerie web (another browser)';
    bug.href = 'https://tinkerie.net/feedback.html?' + new URLSearchParams({ app: bug.dataset.app, version: bug.dataset.ver, where });
  }

  // ---- Web vs desktop notes
  $('#tkWebBtn').onclick = () => { $('#tkWeb').hidden = false; };
  $('#tkWebClose').onclick = () => { $('#tkWeb').hidden = true; };

  // ---- Devices: the browser only lets the page use what the person picks.
  const kinds = (document.body.dataset.devices || '').split(',').filter(Boolean);
  const ASK = {
    serial: ['A serial port', 'USB-serial adapters, Arduino, ESP32, Raspberry Pi Pico, 3D printers, meter cables that show up as a COM port', 'serial'],
    usb: ['A USB device', 'USB oscilloscopes and logic analysers', 'usb'],
    hid: ['A USB-HID meter cable', 'UNI-T UT-D04, Victor, Brymen BU-86X and CP2110 cables', 'hid'],
  };
  const devBtn = $('#tkDevBtn');
  if (devBtn) {
    const supported = (k) => tkDev[k].supported();
    async function showDevs() {
      const ask = $('#tkAsk');
      ask.innerHTML = '';
      $('#tkNoDev').hidden = kinds.some(supported);
      for (const k of kinds) {
        if (!supported(k)) continue;
        const b = document.createElement('button');
        b.innerHTML = '<b></b><span></span>';
        b.querySelector('b').textContent = ASK[k][0] + '…';
        b.querySelector('span').textContent = ASK[k][1];
        b.onclick = async () => {
          try { await tkDev.ask[k](); status('Connected. Pick it in ' + document.title + '’s device list (refresh the list if it’s open).'); setTimeout(() => status(''), 7000); }
          catch (e) { if (e.name !== 'NotFoundError') status('The browser said: ' + e.message, true); }
          listDevs();
        };
        ask.appendChild(b);
      }
      await listDevs();
      $('#tkDevs').hidden = false;
    }
    async function listDevs() {
      const box = $('#tkGranted');
      const all = (await tkDev.granted()).filter((d) => kinds.includes(d.kind));
      box.innerHTML = all.length ? '' : '<p class="tk-dim">None yet.</p>';
      for (const d of all) {
        const row = document.createElement('div');
        row.className = 'tk-dev';
        row.innerHTML = '<span class="n"></span><button class="tk-btn" title="Stop letting this page use it">Forget</button>';
        row.querySelector('.n').textContent = d.name;
        row.querySelector('button').onclick = async () => { try { await d.obj.forget(); } catch {} listDevs(); };
        box.appendChild(row);
      }
      devBtn.classList.toggle('need', !all.length);
    }
    devBtn.onclick = showDevs;
    $('#tkDevsClose').onclick = () => { $('#tkDevs').hidden = true; };
    listDevs();
    for (const k of kinds) {
      const api = { serial: navigator.serial, usb: navigator.usb, hid: navigator.hid }[k];
      if (api) api.addEventListener('connect', () => listDevs()), api.addEventListener('disconnect', () => listDevs());
    }
  }

  // ---- Clear my data: for shared computers
  const clear = $('#tkClear');
  if (clear) clear.onclick = async () => {
    if (!confirm('Delete everything ' + document.title + ' has saved in this browser (files, autosaves and settings), and forget the devices it may use? Download anything you want to keep first.')) return;
    await tkFS.wipe();
    try {
      for (const k of Object.keys(localStorage)) if (k.startsWith('tk.' + app + '.')) localStorage.removeItem(k);
      for (const c of document.cookie.split('; ')) { const n = c.split('=')[0]; if (n.startsWith('tk_' + app + '_')) document.cookie = n + '=; path=/; max-age=0'; }
    } catch {}
    for (const d of await tkDev.granted()) if (kinds.includes(d.kind)) { try { await d.obj.forget(); } catch {} }
    sessionStorage.removeItem('tksid-' + app);
    location.href = location.pathname;
  };

  $('#tkFilesBtn').onclick = showFiles;
  $('#tkFilesClose').onclick = () => { $('#tkFiles').hidden = true; };
  $('#tkImport').onchange = async (e) => {
    for (const file of e.target.files) {
      tkFS.write(home + '/Imported/' + file.name, new Uint8Array(await file.arrayBuffer()));
    }
    e.target.value = '';
    showFiles();
    status('Added to Imported — open it from the app’s own Open list.');
    setTimeout(() => status(''), 5000);
  };
})();
