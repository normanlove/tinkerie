// Tinkerie web: devices. Gives the apps (Go compiled to WebAssembly, see
// shared/webjs) serial ports, USB devices and USB-HID devices through Chrome
// and Edge's Web Serial, WebUSB and WebHID. A page can only use a device the
// person has picked once, so the bar's Connect button asks for one; after
// that the app finds it by itself. Made by nrmluvdev.
(function () {
  'use strict';
  const ids = new WeakMap();
  let nextId = 1;
  const idOf = (o) => { if (!ids.has(o)) ids.set(o, nextId++); return ids.get(o); };
  const handles = new Map();
  let nextH = 1;
  const keep = (o) => { const h = nextH++; handles.set(h, o); return h; };
  const get = (h) => { const o = handles.get(h); if (!o) throw new Error('that device was closed'); return o; };
  const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
  // wait for a queue to have something, at most ms (0 = for ever)
  function waitFor(q, ms) {
    if (q.items.length || q.closed) return Promise.resolve();
    return new Promise((resolve) => {
      const t = ms > 0 ? setTimeout(done, ms) : 0;
      function done() { clearTimeout(t); const i = q.waiters.indexOf(done); if (i >= 0) q.waiters.splice(i, 1); resolve(); }
      q.waiters.push(done);
    });
  }
  const queue = () => ({ items: [], waiters: [], closed: false, err: null });
  function push(q, item) { q.items.push(item); for (const w of q.waiters.splice(0)) w(); }
  function shut(q, err) { q.closed = true; q.err = err || null; for (const w of q.waiters.splice(0)) w(); }

  // ---------------------------------------------------------------- serial
  const serial = {
    supported: () => !!navigator.serial,
    async list() {
      if (!navigator.serial) return [];
      const ports = await navigator.serial.getPorts();
      return ports.map((p) => { const i = p.getInfo(); return { id: idOf(p), vid: i.usbVendorId || 0, pid: i.usbProductId || 0 }; });
    },
    async open(id, o) {
      const ports = navigator.serial ? await navigator.serial.getPorts() : [];
      const port = ports.find((p) => idOf(p) === id);
      if (!port) throw new Error('that serial port isn’t connected any more');
      const s = { port, o, q: queue(), reader: null, loop: null };
      await openPort(s);
      return keep(s);
    },
    async read(h, max, ms) {
      const s = get(h);
      await waitFor(s.q, ms);
      if (!s.q.items.length) {
        if (s.q.closed) throw new Error(s.q.err || 'the port was closed or unplugged');
        return new Uint8Array(0);
      }
      const out = new Uint8Array(max);
      let n = 0;
      while (s.q.items.length && n < max) {
        const c = s.q.items[0];
        const k = Math.min(c.length, max - n);
        out.set(c.subarray(0, k), n); n += k;
        if (k === c.length) s.q.items.shift(); else s.q.items[0] = c.subarray(k);
      }
      return out.subarray(0, n);
    },
    async write(h, data) {
      const s = get(h);
      const w = s.port.writable.getWriter();
      try { await w.write(data); } finally { w.releaseLock(); }
      return data.length;
    },
    async signals(h, dtr, rts) {
      const s = get(h);
      s.o.dtr = dtr; s.o.rts = rts;
      await s.port.setSignals({ dataTerminalReady: dtr, requestToSend: rts });
    },
    async reconfigure(h, o) { // a new baud rate or framing: close and reopen
      const s = get(h);
      await closePort(s);
      s.o = Object.assign(s.o, o);
      s.q = Object.assign(queue(), { items: s.q.items });
      await openPort(s);
    },
    purge(h) { get(h).q.items.length = 0; },
    async close(h) { const s = handles.get(h); if (!s) return; handles.delete(h); await closePort(s); },
  };
  async function openPort(s) {
    const o = s.o;
    try {
      await s.port.open({ baudRate: o.baud, dataBits: o.dataBits, stopBits: o.stopBits, parity: o.parity, bufferSize: 65536, flowControl: 'none' });
    } catch (e) {
      if (/already open/i.test(e.message)) throw new Error('the port is already open in another app or tab: close it there first');
      throw new Error('couldn’t open the port: ' + e.message);
    }
    try { await s.port.setSignals({ dataTerminalReady: !!o.dtr, requestToSend: !!o.rts }); } catch (e) { /* some adapters have no lines */ }
    s.loop = (async () => {
      while (s.port.readable) {
        s.reader = s.port.readable.getReader();
        try {
          for (;;) {
            const { value, done } = await s.reader.read();
            if (done) break;
            if (value && value.length) push(s.q, value);
          }
        } catch (e) {
          if (s.closing) break;
          if (e.name === 'NetworkError' || /lost|disconnected/i.test(e.message)) { shut(s.q, 'the device was unplugged'); break; }
          // framing/parity/overrun errors: keep reading
        } finally {
          try { s.reader.releaseLock(); } catch {}
          s.reader = null;
        }
        if (s.closing) break;
      }
      if (!s.closing) shut(s.q, 'the device was unplugged');
    })();
  }
  async function closePort(s) {
    s.closing = true;
    try { if (s.reader) await s.reader.cancel(); } catch {}
    try { await s.loop; } catch {}
    try { await s.port.close(); } catch {}
    s.closing = false;
    shut(s.q);
  }

  // ------------------------------------------------------------------- USB
  function ifaceInfo(cfg) {
    const out = [];
    for (const it of (cfg ? cfg.interfaces : [])) {
      const alt = it.alternates[0] || it.alternate;
      if (!alt) continue;
      const bin = [], bout = [], iin = [];
      for (const ep of alt.endpoints) {
        const addr = ep.endpointNumber | (ep.direction === 'in' ? 0x80 : 0);
        if (ep.type === 'bulk') (ep.direction === 'in' ? bin : bout).push(addr);
        if (ep.type === 'interrupt' && ep.direction === 'in') iin.push(addr);
      }
      out.push({ number: it.interfaceNumber, cls: alt.interfaceClass, sub: alt.interfaceSubclass, proto: alt.interfaceProtocol, bulkIn: bin, bulkOut: bout, intIn: iin });
    }
    return out;
  }
  const kinds = ['standard', 'class', 'vendor'];
  const recips = ['device', 'interface', 'endpoint', 'other'];
  const setup = (rt, req, val, idx) => ({ requestType: kinds[(rt >> 5) & 3] || 'vendor', recipient: recips[rt & 0x1f] || 'device', request: req, value: val, index: idx });
  function check(r, what) {
    if (r.status === 'stall') throw new Error(what + ': the device refused it (stall)');
    if (r.status === 'babble') throw new Error(what + ': the device sent too much (babble)');
  }
  const usb = {
    supported: () => !!navigator.usb,
    async list() {
      if (!navigator.usb) return [];
      const ds = await navigator.usb.getDevices();
      return ds.map((d) => ({
        id: idOf(d), vid: d.vendorId, pid: d.productId,
        bcd: (d.deviceVersionMajor << 8) | (d.deviceVersionMinor << 4) | d.deviceVersionSubminor,
        manufacturer: d.manufacturerName || '', product: d.productName || '', serial: d.serialNumber || '',
        interfaces: ifaceInfo(d.configuration || d.configurations[0]),
      }));
    },
    async open(id, iface) {
      const ds = navigator.usb ? await navigator.usb.getDevices() : [];
      const d = ds.find((x) => idOf(x) === id);
      if (!d) throw new Error('that USB device isn’t connected any more');
      try {
        if (!d.opened) await d.open();
        if (!d.configuration) await d.selectConfiguration(d.configurations[0].configurationValue);
      } catch (e) {
        throw new Error('couldn’t open the USB device (is another app or tab using it?): ' + e.message);
      }
      const u = { d, claimed: new Set(), pend: new Map(), streams: new Map() };
      await usb.claimIn(u, iface);
      return keep(u);
    },
    async claimIn(u, iface) {
      if (u.claimed.has(iface)) return;
      try { await u.d.claimInterface(iface); } catch (e) {
        throw new Error('the browser can’t use this USB device' + (/protected/i.test(e.message) ? ' (the system keeps it for itself)' : '') + ': ' + e.message +
          '. On Windows it needs the WinUSB driver (Zadig can install it).');
      }
      u.claimed.add(iface);
    },
    claim(h, iface) { return usb.claimIn(get(h), iface); },
    async controlIn(h, rt, req, val, idx, len) {
      const r = await get(h).d.controlTransferIn(setup(rt, req, val, idx), len);
      check(r, 'control transfer');
      return new Uint8Array(r.data.buffer, r.data.byteOffset, r.data.byteLength);
    },
    async controlOut(h, rt, req, val, idx, data) {
      const r = await get(h).d.controlTransferOut(setup(rt, req, val, idx), data);
      check(r, 'control transfer');
      return r.bytesWritten;
    },
    // A transfer that timed out is still waiting in the browser; the next read
    // of that endpoint picks up its result instead of losing it.
    async bulkIn(h, ep, len, ms) {
      const u = get(h);
      let p = u.pend.get(ep);
      if (!p) {
        p = u.d.transferIn(ep & 0x7f, len).then((r) => { check(r, 'USB read'); return new Uint8Array(r.data.buffer, r.data.byteOffset, r.data.byteLength); });
        p.catch(() => {});
        u.pend.set(ep, p);
      }
      const t = ms > 0 ? sleep(ms).then(() => null) : new Promise(() => {});
      const r = await Promise.race([p, t]);
      if (r !== null) u.pend.delete(ep);
      return r; // null: timed out
    },
    async bulkOut(h, ep, data) {
      const r = await get(h).d.transferOut(ep & 0x7f, data);
      check(r, 'USB write');
      return r.bytesWritten;
    },
    async clearHalt(h, ep) { try { await get(h).d.clearHalt(ep & 0x80 ? 'in' : 'out', ep & 0x7f); } catch {} },
    // Streaming: keep several reads queued so no data is missed.
    streamStart(h, ep, size, count) {
      const u = get(h);
      const s = { q: queue(), stop: false };
      u.streams.set(ep, s);
      const one = async () => {
        while (!s.stop) {
          try {
            const r = await u.d.transferIn(ep & 0x7f, size);
            if (s.stop) break;
            if (r.status === 'stall') { await u.d.clearHalt('in', ep & 0x7f); continue; }
            push(s.q, new Uint8Array(r.data.buffer, r.data.byteOffset, r.data.byteLength));
          } catch (e) { if (!s.stop) shut(s.q, e.message); break; }
        }
      };
      for (let i = 0; i < count; i++) one();
    },
    async streamNext(h, ep, ms) {
      const s = get(h).streams.get(ep);
      if (!s) throw new Error('not streaming');
      await waitFor(s.q, ms);
      if (s.q.items.length) return s.q.items.shift();
      if (s.q.closed) throw new Error(s.q.err || 'stopped');
      return null;
    },
    streamStop(h, ep) { const u = handles.get(h); const s = u && u.streams.get(ep); if (s) { s.stop = true; shut(s.q); u.streams.delete(ep); } },
    async close(h) {
      const u = handles.get(h);
      if (!u) return;
      handles.delete(h);
      for (const s of u.streams.values()) { s.stop = true; shut(s.q); }
      for (const i of u.claimed) { try { await u.d.releaseInterface(i); } catch {} }
      try { await u.d.close(); } catch {}
    },
  };

  // ------------------------------------------------------------------- HID
  const hid = {
    supported: () => !!navigator.hid,
    async list() {
      if (!navigator.hid) return [];
      const ds = await navigator.hid.getDevices();
      const seen = new Set();
      return ds.filter((d) => { const k = idOf(d); if (seen.has(k)) return false; seen.add(k); return true; }).map((d) => ({
        id: idOf(d), vid: d.vendorId, pid: d.productId, name: d.productName || '',
      }));
    },
    async open(id) {
      const ds = navigator.hid ? await navigator.hid.getDevices() : [];
      const d = ds.find((x) => idOf(x) === id);
      if (!d) throw new Error('that USB cable isn’t connected any more');
      if (!d.opened) {
        try { await d.open(); } catch (e) { throw new Error('couldn’t open the USB cable (is another app or tab using it?): ' + e.message); }
      }
      const o = { d, q: queue() };
      o.on = (ev) => {
        const b = new Uint8Array(ev.data.byteLength + 1);
        b[0] = ev.reportId;
        b.set(new Uint8Array(ev.data.buffer, ev.data.byteOffset, ev.data.byteLength), 1);
        if (o.q.items.length > 512) o.q.items.shift();
        push(o.q, b);
      };
      d.addEventListener('inputreport', o.on);
      return keep(o);
    },
    async read(h, ms) {
      const o = get(h);
      await waitFor(o.q, ms);
      if (o.q.items.length) return o.q.items.shift();
      if (!o.d.opened) throw new Error('the USB cable was unplugged');
      return null;
    },
    async write(h, b) { await get(h).d.sendReport(b[0], b.subarray(1)); },
    async feature(h, b) { await get(h).d.sendFeatureReport(b[0], b.subarray(1)); },
    async close(h) {
      const o = handles.get(h);
      if (!o) return;
      handles.delete(h);
      o.d.removeEventListener('inputreport', o.on);
      shut(o.q);
      try { await o.d.close(); } catch {}
    },
  };

  // --------------------------------------------- asking for a device (the bar)
  const ask = {
    async serial() { return navigator.serial.requestPort({}); },
    async usb() { return navigator.usb.requestDevice({ filters: [] }); },
    async hid() { const ds = await navigator.hid.requestDevice({ filters: [] }); if (!ds.length) throw new DOMException('none chosen', 'NotFoundError'); return ds[0]; },
  };
  async function granted() {
    const out = [];
    if (navigator.serial) for (const p of await navigator.serial.getPorts()) {
      const i = p.getInfo();
      out.push({ kind: 'serial', name: i.usbVendorId ? 'Serial port ' + hex(i.usbVendorId) + ':' + hex(i.usbProductId) : 'Serial port', obj: p });
    }
    if (navigator.usb) for (const d of await navigator.usb.getDevices()) out.push({ kind: 'usb', name: (d.productName || 'USB device') + ' ' + hex(d.vendorId) + ':' + hex(d.productId), obj: d });
    if (navigator.hid) {
      const seen = new Set();
      for (const d of await navigator.hid.getDevices()) {
        if (seen.has(d)) continue; seen.add(d);
        out.push({ kind: 'hid', name: (d.productName || 'USB-HID device') + ' ' + hex(d.vendorId) + ':' + hex(d.productId), obj: d });
      }
    }
    return out;
  }
  const hex = (n) => (n || 0).toString(16).toUpperCase().padStart(4, '0');

  globalThis.tkDev = { serial, usb, hid, ask, granted, idOf };
})();
