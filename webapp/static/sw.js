// Tinkerie web: passes each app frame's requests to that app, which runs
// inside its page (see webhost). Everything else goes to the network as usual.
// Made by nrmluvdev.
'use strict';
self.addEventListener('install', () => self.skipWaiting());
self.addEventListener('activate', (e) => e.waitUntil(self.clients.claim()));

const UI = /\/([a-z]+)\/ui(\/[^?#]*)?/; // .../<app>/ui/<path>

function sidOf(url) { try { return new URL(url).searchParams.get('tksid'); } catch { return null; } }

async function route(event) {
  const req = event.request, url = new URL(req.url);
  if (url.origin !== self.location.origin) return null;
  let sid = null, path = null;
  const base = new URL('./', self.registration.scope).pathname; // "/tinkerie/"
  const m = url.pathname.startsWith(base) ? url.pathname.slice(base.length - 1).match(UI) : null;
  if (req.mode === 'navigate') {
    if (!m) return null;
    sid = sidOf(req.url);
    url.searchParams.delete('tksid');
    path = (m[2] || '/') + url.search;
  } else {
    if (!event.clientId) return null;
    const c = await self.clients.get(event.clientId);
    if (!c || !UI.test(new URL(c.url).pathname)) {
      // Only an app frame asks for paths outside Tinkerie's folder; if it's
      // closing, don't send its /api/ calls to the real web server.
      return url.pathname.startsWith(base) ? null : new Response('', { status: 503 });
    }
    sid = sidOf(c.url);
    // Paths relative to the frame (…/ui/x) and absolute ones (/x) both go to the app.
    path = (m ? (m[2] || '/') : url.pathname) + url.search;
  }
  if (!sid) return null;
  const shells = await self.clients.matchAll({ type: 'window', includeUncontrolled: true });
  const shell = shells.find((s) => sidOf(s.url) === sid && !UI.test(new URL(s.url).pathname));
  if (!shell) return new Response('This Tinkerie app was closed — reopen it from the Tinkerie page.', { status: 503 });
  return forward(shell, sid, req, path);
}

async function forward(shell, sid, req, path) {
  const headers = {};
  req.headers.forEach((v, k) => { headers[k] = v; });
  const body = req.method === 'GET' || req.method === 'HEAD' ? null : new Uint8Array(await req.arrayBuffer());
  const ch = new MessageChannel();
  const head = new Promise((resolve) => {
    let ctrl = null, queued = [], done = false;
    const stream = new ReadableStream({
      start(c) { ctrl = c; for (const q of queued) c.enqueue(q); queued = null; if (done) c.close(); },
      cancel() { ch.port1.postMessage({ t: 'cancel' }); ch.port1.close(); },
    });
    ch.port1.onmessage = (ev) => {
      const m = ev.data;
      if (m.t === 'head') {
        const h = new Headers(m.headers || {});
        h.delete('content-length'); h.delete('content-encoding');
        resolve(new Response(req.method === 'HEAD' || m.status === 204 || m.status === 304 ? null : stream, { status: m.status, headers: h }));
      } else if (m.t === 'chunk') {
        ctrl ? ctrl.enqueue(m.chunk) : queued.push(m.chunk);
      } else if (m.t === 'end') {
        if (m.err) resolve(new Response('Tinkerie: ' + m.err, { status: 502 }));
        done = true;
        try { ctrl && ctrl.close(); } catch {}
        ch.port1.close();
      }
    };
  });
  shell.postMessage({ tk: 'fetch', sid, method: req.method, path, headers, body }, [ch.port2]);
  return head;
}

self.addEventListener('fetch', (event) => {
  const url = new URL(event.request.url);
  if (url.origin !== self.location.origin) return;
  event.respondWith(route(event).then((r) => r || fetch(event.request)));
});
