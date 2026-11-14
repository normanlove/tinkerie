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
  // One id per open tab, so two tabs of the same app don't mix.
  let sid = sessionStorage.getItem('tksid-' + app);
  if (!sid) { sid = Math.random().toString(36).slice(2, 10); sessionStorage.setItem('tksid-' + app, sid); }
  // The service worker finds this tab by the id in its address, so it must be
  // there from the start (a rewritten address isn't seen by service workers).
  const u = new URL(location.href);
  if (u.searchParams.get('tksid') !== sid) { u.searchParams.set('tksid', sid); location.replace(u); return; }

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
