// Tinkerie web: a small file system for Go programs running in the page.
// Go's browser runtime expects a Node-style `fs`; this one keeps files in
// memory and saves them to IndexedDB, so each app's settings, projects and
// autosaves are still there next time (per browser, per app).
// Made by nrmluvdev.
(function () {
  'use strict';
  const S_IFDIR = 0o040000, S_IFREG = 0o100000;
  const C = { O_RDONLY: 0, O_WRONLY: 1, O_RDWR: 2, O_CREAT: 64, O_EXCL: 128, O_TRUNC: 512, O_APPEND: 1024, O_DIRECTORY: 65536 };
  const err = (code) => { const e = new Error(code); e.code = code; return e; };

  // nodes: path -> {dir:true} | {data:Uint8Array, size:n}
  const nodes = new Map([['/', { dir: true, mtime: Date.now() }]]);
  const fds = new Map();
  let nextFd = 10;
  const dirty = new Set();
  let db = null, dbName = '', saveTimer = 0;

  const norm = (p) => {
    const out = [];
    for (const s of String(p).split('/')) {
      if (!s || s === '.') continue;
      if (s === '..') out.pop(); else out.push(s);
    }
    return '/' + out.join('/');
  };
  const parent = (p) => p === '/' ? '/' : (p.slice(0, p.lastIndexOf('/')) || '/');
  const persisted = (p) => !p.startsWith('/tmp');

  function touch(p) {
    if (!persisted(p)) return;
    dirty.add(p);
    if (db && !saveTimer) saveTimer = setTimeout(flush, 400);
  }
  function flush() {
    saveTimer = 0;
    if (!db || !dirty.size) return;
    const tx = db.transaction('files', 'readwrite'), st = tx.objectStore('files');
    for (const p of dirty) {
      const n = nodes.get(p);
      if (!n) st.delete(p);
      else if (n.dir) st.put({ dir: true, mtime: n.mtime }, p);
      else st.put({ data: n.data.slice(0, n.size), mtime: n.mtime }, p);
    }
    dirty.clear();
  }
  addEventListener('pagehide', flush);
  addEventListener('visibilitychange', () => { if (document.visibilityState === 'hidden') flush(); });

  function mkdirp(p) {
    p = norm(p);
    if (nodes.has(p)) return;
    mkdirp(parent(p));
    nodes.set(p, { dir: true, mtime: Date.now() });
    touch(p);
  }

  function statOf(p, n) {
    const size = n.dir ? 4096 : n.size;
    return {
      dev: 1, ino: hash(p), mode: (n.dir ? S_IFDIR | 0o755 : S_IFREG | 0o644), nlink: 1, uid: 1000, gid: 1000,
      rdev: 0, size, blksize: 4096, blocks: Math.ceil(size / 512), atimeMs: n.mtime, mtimeMs: n.mtime, ctimeMs: n.mtime,
      isDirectory: () => !!n.dir,
    };
  }
  function hash(s) { let h = 7; for (let i = 0; i < s.length; i++) h = (h * 31 + s.charCodeAt(i)) | 0; return h >>> 0; }

  function grow(n, need) {
    if (need <= n.data.length) return;
    const b = new Uint8Array(Math.max(need, n.data.length * 2, 256));
    b.set(n.data.subarray(0, n.size));
    n.data = b;
  }

  let outBuf = '';
  const dec = new TextDecoder();
  function console_(fd, buf) {
    outBuf += dec.decode(buf);
    const nl = outBuf.lastIndexOf('\n');
    if (nl !== -1) { (fd === 2 ? console.warn : console.log)(outBuf.slice(0, nl)); outBuf = outBuf.slice(nl + 1); }
    return buf.length;
  }

  const fs = {
    constants: C,
    writeSync(fd, buf) {
      if (fd === 1 || fd === 2) return console_(fd, buf);
      return writeAt(fd, buf, 0, buf.length, null);
    },
    write(fd, buf, off, len, pos, cb) {
      try {
        if (fd === 1 || fd === 2) return cb(null, console_(fd, buf.subarray(off, off + len)));
        cb(null, writeAt(fd, buf, off, len, pos));
      } catch (e) { cb(e); }
    },
    read(fd, buf, off, len, pos, cb) {
      const f = fds.get(fd);
      if (!f) return cb(err('EBADF'));
      const n = nodes.get(f.path);
      if (!n || n.dir) return cb(err(n ? 'EISDIR' : 'EBADF'));
      const at = pos === null || pos === undefined ? f.pos : pos;
      const k = Math.max(0, Math.min(len, n.size - at));
      buf.set(n.data.subarray(at, at + k), off);
      if (pos === null || pos === undefined) f.pos += k;
      cb(null, k);
    },
    open(path, flags, mode, cb) {
      const p = norm(path);
      let n = nodes.get(p);
      if (n && (flags & C.O_CREAT) && (flags & C.O_EXCL)) return cb(err('EEXIST'));
      if (!n) {
        if (!(flags & C.O_CREAT)) return cb(err('ENOENT'));
        const par = nodes.get(parent(p));
        if (!par) return cb(err('ENOENT'));
        if (!par.dir) return cb(err('ENOTDIR'));
        n = { data: new Uint8Array(0), size: 0, mtime: Date.now() };
        nodes.set(p, n);
        touch(p);
      }
      if (n.dir && (flags & 3) !== 0) return cb(err('EISDIR'));
      if ((flags & C.O_DIRECTORY) && !n.dir) return cb(err('ENOTDIR'));
      if (!n.dir && (flags & C.O_TRUNC) && (flags & 3) !== 0) { n.size = 0; n.mtime = Date.now(); touch(p); }
      const fd = nextFd++;
      fds.set(fd, { path: p, pos: 0, append: !!(flags & C.O_APPEND) });
      cb(null, fd);
    },
    close(fd, cb) { fds.delete(fd); cb(null); },
    fsync(fd, cb) { cb(null); },
    fstat(fd, cb) {
      const f = fds.get(fd);
      if (!f) return cb(err('EBADF'));
      const n = nodes.get(f.path);
      if (!n) return cb(err('ENOENT'));
      cb(null, statOf(f.path, n));
    },
    stat(path, cb) {
      const p = norm(path), n = nodes.get(p);
      n ? cb(null, statOf(p, n)) : cb(err('ENOENT'));
    },
    lstat(path, cb) { fs.stat(path, cb); },
    mkdir(path, perm, cb) {
      const p = norm(path);
      if (nodes.has(p)) return cb(err('EEXIST'));
      const par = nodes.get(parent(p));
      if (!par) return cb(err('ENOENT'));
      if (!par.dir) return cb(err('ENOTDIR'));
      nodes.set(p, { dir: true, mtime: Date.now() });
      touch(p);
      cb(null);
    },
    readdir(path, cb) {
      const p = norm(path), n = nodes.get(p);
      if (!n) return cb(err('ENOENT'));
      if (!n.dir) return cb(err('ENOTDIR'));
      const pre = p === '/' ? '/' : p + '/';
      const names = [];
      for (const k of nodes.keys()) {
        if (k !== p && k.startsWith(pre) && !k.slice(pre.length).includes('/')) names.push(k.slice(pre.length));
      }
      cb(null, names.sort());
    },
    unlink(path, cb) {
      const p = norm(path), n = nodes.get(p);
      if (!n) return cb(err('ENOENT'));
      if (n.dir) return cb(err('EISDIR'));
      nodes.delete(p); touch(p); cb(null);
    },
    rmdir(path, cb) {
      const p = norm(path), n = nodes.get(p);
      if (!n) return cb(err('ENOENT'));
      if (!n.dir) return cb(err('ENOTDIR'));
      const pre = p + '/';
      for (const k of nodes.keys()) if (k.startsWith(pre)) return cb(err('ENOTEMPTY'));
      nodes.delete(p); touch(p); cb(null);
    },
    rename(from, to, cb) {
      const a = norm(from), b = norm(to), n = nodes.get(a);
      if (!n) return cb(err('ENOENT'));
      if (!nodes.get(parent(b))) return cb(err('ENOENT'));
      const moves = [[a, b]];
      if (n.dir) for (const k of nodes.keys()) if (k.startsWith(a + '/')) moves.push([k, b + k.slice(a.length)]);
      const got = moves.map(([x, y]) => [y, nodes.get(x)]);
      for (const [x] of moves) { nodes.delete(x); touch(x); }
      for (const [y, v] of got) { nodes.set(y, v); touch(y); }
      cb(null);
    },
    ftruncate(fd, len, cb) {
      const f = fds.get(fd); if (!f) return cb(err('EBADF'));
      truncateNode(f.path, len); cb(null);
    },
    truncate(path, len, cb) {
      const p = norm(path); if (!nodes.get(p)) return cb(err('ENOENT'));
      truncateNode(p, len); cb(null);
    },
    utimes(path, atime, mtime, cb) {
      const n = nodes.get(norm(path)); if (!n) return cb(err('ENOENT'));
      n.mtime = mtime * 1000; touch(norm(path)); cb(null);
    },
    chmod(p, m, cb) { cb(null); }, fchmod(fd, m, cb) { cb(null); },
    chown(p, u, g, cb) { cb(null); }, fchown(fd, u, g, cb) { cb(null); }, lchown(p, u, g, cb) { cb(null); },
    link(a, b, cb) { cb(err('ENOSYS')); }, symlink(a, b, cb) { cb(err('ENOSYS')); }, readlink(p, cb) { cb(err('EINVAL')); },
  };

  function truncateNode(p, len) {
    const n = nodes.get(p);
    grow(n, len);
    if (len > n.size) n.data.fill(0, n.size, len);
    n.size = len; n.mtime = Date.now(); touch(p);
  }

  function writeAt(fd, buf, off, len, pos) {
    const f = fds.get(fd);
    if (!f) throw err('EBADF');
    const n = nodes.get(f.path);
    if (!n || n.dir) throw err('EBADF');
    let at = pos === null || pos === undefined ? (f.append ? n.size : f.pos) : pos;
    grow(n, at + len);
    if (at > n.size) n.data.fill(0, n.size, at);
    n.data.set(buf.subarray(off, off + len), at);
    n.size = Math.max(n.size, at + len);
    n.mtime = Date.now();
    if (pos === null || pos === undefined) f.pos = at + len;
    touch(f.path);
    return len;
  }

  globalThis.fs = fs;

  // tkFS: helpers for the shell (loading, downloads, import).
  globalThis.tkFS = {
    async open(name) {
      dbName = 'tinkerie-' + name;
      db = await new Promise((res, rej) => {
        const r = indexedDB.open(dbName, 1);
        r.onupgradeneeded = () => r.result.createObjectStore('files');
        r.onsuccess = () => res(r.result);
        r.onerror = () => rej(r.error);
      }).catch(() => null);
      if (!db) return;
      await new Promise((res) => {
        const tx = db.transaction('files', 'readonly'), st = tx.objectStore('files');
        const req = st.openCursor();
        req.onsuccess = () => {
          const c = req.result;
          if (!c) return res();
          const p = c.key, v = c.value;
          if (v.dir) nodes.set(p, { dir: true, mtime: v.mtime });
          else nodes.set(p, { data: new Uint8Array(v.data), size: v.data.byteLength, mtime: v.mtime });
          c.continue();
        };
        req.onerror = () => res();
      });
      for (const p of [...nodes.keys()]) mkdirp(parent(p));
    },
    mkdirp,
    write(path, bytes) {
      const p = norm(path);
      mkdirp(parent(p));
      nodes.set(p, { data: new Uint8Array(bytes), size: bytes.byteLength, mtime: Date.now() });
      touch(p);
    },
    read(path) {
      const n = nodes.get(norm(path));
      return n && !n.dir ? n.data.slice(0, n.size) : null;
    },
    list(dir) {
      const p = norm(dir), pre = p === '/' ? '/' : p + '/', out = [];
      for (const [k, n] of nodes) if (k.startsWith(pre) && !n.dir) out.push({ path: k, size: n.size, mtime: n.mtime });
      return out;
    },
    // Files written since `since` under dir (used to offer downloads of exports).
    changedSince(dir, since) { return this.list(dir).filter((f) => f.mtime >= since); },
    flush,
    async wipe() { nodes.clear(); nodes.set('/', { dir: true, mtime: Date.now() }); if (db) { db.transaction('files', 'readwrite').objectStore('files').clear(); } },
  };
})();
