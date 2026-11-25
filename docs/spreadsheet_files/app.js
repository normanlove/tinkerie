(function () {
  'use strict';
  var $ = function (id) { return document.getElementById(id); };

  /* ---------- sign-in ----------
     This is a simple gate, not real security: the page is static, so the
     check happens in the browser. The password is not stored in readable form. */
  var SALT = 'sheet-gate-v1-7f3a9c';
  var ITER = 150000;
  var HASH = 'b577bc511a0e3efd0d74259eda87bb18c5bd9a6c12872f3ff3fc7657124d127a';

  function hex(buf) { return Array.prototype.map.call(new Uint8Array(buf), function (b) { return ('0' + b.toString(16)).slice(-2); }).join(''); }
  async function derive(user, pass) {
    var enc = new TextEncoder();
    var key = await crypto.subtle.importKey('raw', enc.encode(user + '\n' + pass), 'PBKDF2', false, ['deriveBits']);
    return hex(await crypto.subtle.deriveBits({ name: 'PBKDF2', hash: 'SHA-256', salt: enc.encode(SALT), iterations: ITER }, key, 256));
  }

  function openApp() {
    setTimeout(preload, 0);
    $('login').classList.add('hidden');
    $('app').classList.remove('hidden');
    try { sessionStorage.setItem('sheet-ok', HASH); } catch (e) {}
  }
  if (!window.crypto || !crypto.subtle) {
    $('loginerr').textContent = 'This page needs a modern browser opened over https.';
    $('loginbtn').disabled = true;
  }
  try { if (sessionStorage.getItem('sheet-ok') === HASH) openApp(); } catch (e) {}

  $('login').addEventListener('submit', async function (ev) {
    ev.preventDefault();
    $('loginerr').textContent = '';
    $('loginbtn').disabled = true;
    try {
      var h = await derive($('u').value.trim(), $('p').value);
      if (h === HASH) { $('p').value = ''; openApp(); }
      else $('loginerr').textContent = 'Wrong username or password.';
    } catch (e) { $('loginerr').textContent = 'Could not check the password in this browser.'; }
    $('loginbtn').disabled = false;
  });

  /* ---------- image handling ---------- */
  var img = null;           // HTMLImageElement
  var words = null;         // OCR result, kept so the slider can re-split quickly
  var worker = null;
  var quiet = false, passLabel = '';
  var NUM_WL = '0123456789.,-+%£$€()/: ';

  function setErr(m) { $('err').textContent = m || ''; }
  function setStatus(m) { $('status').textContent = m || ''; }

  function loadFile(f) {
    setErr('');
    if (!f || !/^image\//.test(f.type)) { setErr('Please choose an image file (PNG, JPG, WebP...).'); return; }
    var url = URL.createObjectURL(f);
    var im = new Image();
    im.onload = function () {
      img = im; words = null;
      $('preview').src = url;
      $('preview').classList.remove('hidden');
      $('go').disabled = false;
      $('result').classList.add('hidden');
      setStatus(im.naturalWidth + ' x ' + im.naturalHeight + ' pixels');
    };
    im.onerror = function () { setErr('That image could not be read.'); };
    im.src = url;
  }

  $('pick').addEventListener('click', function () { $('file').click(); });
  $('file').addEventListener('change', function () { if (this.files[0]) loadFile(this.files[0]); this.value = ''; });
  ['dragenter', 'dragover'].forEach(function (n) { $('drop').addEventListener(n, function (e) { e.preventDefault(); $('drop').classList.add('over'); }); });
  ['dragleave', 'drop'].forEach(function (n) { $('drop').addEventListener(n, function (e) { e.preventDefault(); $('drop').classList.remove('over'); }); });
  $('drop').addEventListener('drop', function (e) { if (e.dataTransfer.files[0]) loadFile(e.dataTransfer.files[0]); });
  document.addEventListener('paste', function (e) {
    if ($('app').classList.contains('hidden')) return;
    var items = (e.clipboardData && e.clipboardData.items) || [];
    for (var i = 0; i < items.length; i++) if (items[i].type.indexOf('image/') === 0) { loadFile(items[i].getAsFile()); break; }
  });

  /* Grey-scale a canvas and decide which pixels are ink. Returns the image data
     plus a 0/1 array (1 = ink). "adaptive" compares each pixel with its
     neighbourhood, which copes with shadows and uneven light in photos; the
     default single global cut-off (Otsu) is better for clean screenshots. */
  function binarize(ctx, cw, ch, adaptive) {
    var d = ctx.getImageData(0, 0, cw, ch), p = d.data, n = cw * ch, i;
    var g = new Uint8Array(n), hist = new Uint32Array(256);
    for (i = 0; i < n; i++) { var v = (p[i*4] * 77 + p[i*4+1] * 150 + p[i*4+2] * 29) >> 8; g[i] = v; hist[v]++; }
    var dark = new Uint8Array(n), x, y;
    if (adaptive) {
      var W = cw + 1, I = new Float64Array(W * (ch + 1)), rs;
      for (y = 0; y < ch; y++) {
        rs = 0;
        for (x = 0; x < cw; x++) { rs += g[y * cw + x]; I[(y + 1) * W + x + 1] = I[y * W + x + 1] + rs; }
      }
      var hs = (Math.max(15, Math.round(cw / 16)) | 1) >> 1;
      for (y = 0; y < ch; y++) {
        var y0 = Math.max(0, y - hs), y1 = Math.min(ch - 1, y + hs);
        for (x = 0; x < cw; x++) {
          var x0 = Math.max(0, x - hs), x1 = Math.min(cw - 1, x + hs);
          var cnt = (x1 - x0 + 1) * (y1 - y0 + 1);
          var sum = I[(y1 + 1) * W + x1 + 1] - I[y0 * W + x1 + 1] - I[(y1 + 1) * W + x0] + I[y0 * W + x0];
          dark[y * cw + x] = g[y * cw + x] * cnt < sum * 0.85 ? 1 : 0;
        }
      }
    } else {
      var sumAll = 0, t; for (t = 0; t < 256; t++) sumAll += t * hist[t];
      var wB = 0, sB = 0, best = 0, thr = 128;
      for (t = 0; t < 256; t++) {
        wB += hist[t]; if (!wB) continue;
        var wF = n - wB; if (!wF) break;
        sB += t * hist[t];
        var mB = sB / wB, mF = (sumAll - sB) / wF, bt = wB * wF * (mB - mF) * (mB - mF);
        if (bt > best) { best = bt; thr = t; }
      }
      for (i = 0; i < n; i++) dark[i] = g[i] <= thr ? 1 : 0;
    }
    var inkCount = 0;
    for (i = 0; i < n; i++) inkCount += dark[i];
    if (inkCount > n / 2) for (i = 0; i < n; i++) dark[i] = 1 - dark[i];   // light text on a dark background
    return { d: d, dark: dark };
  }

  /* Enlarge small images, straighten a tilted page, and optionally erase long
     table lines. opts: { removeGrid, adaptive, deskew } */
  function prepare(opts) {
    var w = img.naturalWidth, h = img.naturalHeight;
    var scale = 1;
    if (w > 4000) scale = 4000 / w;
    else {
      // Measure the text at its real size; only enlarge when it is small (OCR time grows with pixel count).
      var m = document.createElement('canvas'); m.width = w; m.height = h;
      var mx = m.getContext('2d', { willReadFrequently: true });
      mx.fillStyle = '#fff'; mx.fillRect(0, 0, w, h); mx.drawImage(img, 0, 0);
      var th = SheetLayout.estimateTextHeight(binarize(mx, w, h, false).dark, w, h);
      if (th) scale = Math.min(3, Math.max(1, 26 / th));
      else if (w < 2000) scale = Math.min(3, 2000 / w);
      if (scale < 1.05) scale = 1;
    }
    if (w * h * scale * scale > 12e6) scale = Math.sqrt(12e6 / (w * h));   // keep memory sane
    var cw = Math.round(w * scale), ch = Math.round(h * scale);
    var c = document.createElement('canvas');
    c.width = cw; c.height = ch;
    var ctx = c.getContext('2d', { willReadFrequently: true });
    ctx.fillStyle = '#fff'; ctx.fillRect(0, 0, cw, ch);
    ctx.imageSmoothingQuality = 'high';
    ctx.drawImage(img, 0, 0, cw, ch);

    var angle = 0, bin = null;
    if (opts.deskew) {
      bin = binarize(ctx, cw, ch, opts.adaptive);
      angle = SheetLayout.estimateSkew(bin.dark, cw, ch);
      if (angle) {
        var r = angle * Math.PI / 180, s = Math.abs(Math.sin(r)), co = Math.abs(Math.cos(r));
        var nw = Math.round(cw * co + ch * s), nh = Math.round(cw * s + ch * co);
        var c2 = document.createElement('canvas');
        c2.width = nw; c2.height = nh;
        var x2 = c2.getContext('2d', { willReadFrequently: true });
        x2.fillStyle = '#fff'; x2.fillRect(0, 0, nw, nh);
        x2.imageSmoothingQuality = 'high';
        x2.translate(nw / 2, nh / 2); x2.rotate(-r); x2.drawImage(c, -cw / 2, -ch / 2);
        c = c2; ctx = x2; cw = nw; ch = nh; bin = null;
      }
    }
    if (!bin) bin = binarize(ctx, cw, ch, opts.adaptive);
    var d = bin.d, p = d.data, dark = bin.dark, n = cw * ch, i;

    if (opts.removeGrid) {
      // Table lines run across much of the page; letter strokes are far shorter.
      var minRun = Math.max(100, Math.round(Math.min(cw, ch) * 0.12));
      var x, y, run, k;
      for (y = 0; y < ch; y++) {           // horizontal lines
        run = 0;
        for (x = 0; x <= cw; x++) {
          if (x < cw && dark[y * cw + x]) run++;
          else { if (run >= minRun) for (k = x - run; k < x; k++) dark[y * cw + k] = 2; run = 0; }
        }
      }
      for (x = 0; x < cw; x++) {           // vertical lines
        run = 0;
        for (y = 0; y <= ch; y++) {
          if (y < ch && dark[y * cw + x]) run++;
          else { if (run >= minRun) for (k = y - run; k < y; k++) dark[k * cw + x] = 2; run = 0; }
        }
      }
    }
    for (i = 0; i < n; i++) { var o = dark[i] === 1 ? 0 : 255; p[i*4] = p[i*4+1] = p[i*4+2] = o; p[i*4+3] = 255; }
    ctx.putImageData(d, 0, 0);
    return { canvas: c, scale: scale, angle: angle };
  }

  /* ---------- OCR (engine, language data and worker are all files next to this page) ---------- */
  function loadScript(src) {
    return new Promise(function (res, rej) {
      var s = document.createElement('script');
      s.src = src; s.onload = res; s.onerror = function () { rej(new Error('Could not load ' + src)); };
      document.head.appendChild(s);
    });
  }

  var workerP = null;
  function getWorker() {
    if (worker) return Promise.resolve(worker);
    if (!workerP) workerP = makeWorker().then(function (w) { worker = w; return w; }, function (e) { workerP = null; throw e; });
    return workerP;
  }
  function preload() {
    quiet = true;
    getWorker().then(function () { quiet = false; }, function () { quiet = false; });
  }

  async function makeWorker() {
    if (worker) return worker;
    if (!window.Tesseract) await loadScript('spreadsheet_files/tesseract.min.js');
    var base = new URL('spreadsheet_files/', location.href).href.replace(/\/$/, '');
    worker = await Tesseract.createWorker('eng', 1, {
      workerPath: base + '/worker.min.js',
      corePath: base,
      langPath: base,
      workerBlobURL: false,
      cacheMethod: 'none',
      gzip: true,
      logger: function (m) {
        if (quiet) return;
        if (m.status === 'recognizing text') { $('bar').value = m.progress; setStatus(passLabel + 'Reading the text... ' + Math.round(m.progress * 100) + '%'); }
        else if (m.status) setStatus(m.status.charAt(0).toUpperCase() + m.status.slice(1) + '...');
      }
    });
    return worker;
  }

  function extractWords(data) {
    var list = data.words;
    if (!list || !list.length) {
      list = [];
      (data.lines || []).forEach(function (l) { (l.words || []).forEach(function (w) { list.push(w); }); });
    }
    return list.map(function (w) { return { text: w.text, confidence: w.confidence, bbox: w.bbox }; });
  }

  /* Second look: words that look like a damaged number are cut out of the
     prepared image and read again on their own with a numbers-only character
     set. Ordinary words are never re-read. */
  async function refineNumbers(list, canvas, userParams) {
    var idx = SheetLayout.rereadCandidates(list).slice(0, 400);
    if (!idx.length) return 0;
    var changed = 0;
    quiet = true;
    try {
      await worker.setParameters({ tessedit_pageseg_mode: '8', tessedit_char_whitelist: NUM_WL });
      for (var k = 0; k < idx.length; k++) {
        setStatus('Double-checking numbers... ' + (k + 1) + ' of ' + idx.length);
        $('bar').value = (k + 1) / idx.length;
        var w = list[idx[k]], b = w.bbox;
        var bw = Math.max(1, b.x1 - b.x0), bh = Math.max(1, b.y1 - b.y0), pad = Math.round(bh * 0.5) + 8;
        var c = document.createElement('canvas');
        c.width = bw + pad * 2; c.height = bh + pad * 2;
        var x = c.getContext('2d');
        x.fillStyle = '#fff'; x.fillRect(0, 0, c.width, c.height);
        x.drawImage(canvas, Math.max(0, b.x0), Math.max(0, b.y0), bw, bh, pad, pad, bw, bh);
        try {
          var r = await worker.recognize(c);
          var t = (r.data.text || '').replace(/\s+/g, ' ').trim();
          if (SheetLayout.acceptReread(w, t, r.data.confidence)) {
            w.text = t; w.confidence = r.data.confidence; changed++;
          }
        } catch (e) { /* keep the original reading */ }
      }
    } finally {
      quiet = false;
      try { await worker.setParameters(userParams); } catch (e) {}
    }
    return changed;
  }

  $('go').addEventListener('click', async function () {
    if (!img) return;
    setErr(''); $('go').disabled = true;
    $('bar').classList.remove('hidden'); $('bar').value = 0;
    $('result').classList.add('hidden');
    try {
      var t0 = performance.now();
      setStatus('Preparing the image...');
      await new Promise(function (r) { setTimeout(r, 30); });
      var wk = await getWorker();
      var userParams = {
        tessedit_pageseg_mode: $('psm').value,
        preserve_interword_spaces: '1',
        tessedit_char_whitelist: $('content').value === 'numbers' ? NUM_WL : ''
      };
      await wk.setParameters(userParams);
      var mode = $('light').value;
      var variants = mode === 'auto' ? [false, true] : [mode === 'uneven'];
      var best = null;
      for (var vi = 0; vi < variants.length; vi++) {
        passLabel = variants.length > 1 ? 'Pass ' + (vi + 1) + ' of ' + variants.length + ': ' : '';
        setStatus(passLabel + 'Preparing the image...');
        await new Promise(function (r) { setTimeout(r, 30); });
        var pr = prepare({ removeGrid: $('grid').value === 'remove', adaptive: variants[vi], deskew: $('deskew').checked });
        $('bar').value = 0;
        var res = await wk.recognize(pr.canvas);
        var got = extractWords(res.data), sc = SheetLayout.scoreWords(got);
        if (!best || sc > best.score) best = { list: got, prep: pr, score: sc };
      }
      passLabel = '';
      var prep = best.prep, list = best.list;
      if (list.length && $('fixdigits').checked) await refineNumbers(list, prep.canvas, userParams);
      words = list;
      if (!words.length) { setErr('No text was found in that image. Try a clearer or larger picture.'); setStatus(''); }
      else { render(); setStatus('Done in ' + ((performance.now() - t0) / 1000).toFixed(1) + ' s' + (prep.angle ? ' (page straightened by ' + Math.abs(prep.angle) + ' degrees).' : '.')); $('result').classList.remove('hidden'); }
    } catch (e) {
      setErr('Something went wrong: ' + (e && e.message ? e.message : e));
      setStatus('');
      try { if (worker) await worker.terminate(); } catch (x) {}
      worker = null; workerP = null;
    }
    $('bar').classList.add('hidden');
    $('go').disabled = false;
  });

  /* ---------- table ---------- */
  var LOW = 75, VLOW = 55;      // OCR confidence (0-100) below which a cell is shaded
  function render() {
    var tab = SheetLayout.buildTable(words, parseFloat($('gap').value));
    var grid = tab.grid, conf = tab.conf, fixed = {};
    if ($('fixdigits').checked) {
      var fx = SheetLayout.fixNumberColumns(grid);
      grid = fx.grid;
      fx.changes.forEach(function (c) { fixed[c.r + ',' + c.c] = c.from; });
    }
    var t = $('tbl'); t.textContent = '';
    var doubtful = 0;
    grid.forEach(function (r, ri) {
      var tr = document.createElement('tr');
      r.forEach(function (v, ci) {
        var td = document.createElement('td'); td.contentEditable = 'true'; td.spellcheck = false; td.textContent = v;
        var cf = conf[ri][ci], key = ri + ',' + ci;
        if (v && cf < VLOW) { td.className = 'vlow'; td.title = 'Very unsure (' + Math.round(cf) + '%): please check'; doubtful++; }
        else if (v && cf < LOW) { td.className = 'low'; td.title = 'Unsure (' + Math.round(cf) + '%): please check'; doubtful++; }
        if (fixed.hasOwnProperty(key)) {
          td.classList.add('fixed');
          td.title = 'Digits corrected from "' + fixed[key] + '"' + (td.title ? '. ' + td.title : '');
        }
        tr.appendChild(td);
      });
      t.appendChild(tr);
    });
    var nfix = Object.keys(fixed).length;
    $('size').textContent = grid.length + ' rows x ' + (grid[0] ? grid[0].length : 0) + ' columns';
    $('flagcount').textContent = doubtful + ' doubtful cell' + (doubtful === 1 ? '' : 's') + (nfix ? ', ' + nfix + ' corrected' : '');
    $('nextlow').disabled = !doubtful;
  }
  function readTable() {
    return Array.prototype.map.call($('tbl').rows, function (tr) {
      return Array.prototype.map.call(tr.cells, function (td) { return td.textContent.replace(/\s+/g, ' ').trim(); });
    });
  }
  var timer;
  $('gap').addEventListener('input', function () {
    if (!words) return;
    // Moving the slider rebuilds the table, replacing any hand edits.
    clearTimeout(timer); timer = setTimeout(render, 80);
  });

  $('tbl').addEventListener('input', function (e) {
    var td = e.target.closest ? e.target.closest('td') : null;
    if (td && (td.classList.contains('low') || td.classList.contains('vlow'))) {
      td.classList.remove('low', 'vlow'); td.removeAttribute('title');
      var left = $('tbl').querySelectorAll('td.low, td.vlow').length;
      $('flagcount').textContent = left + ' doubtful cell' + (left === 1 ? '' : 's');
      $('nextlow').disabled = !left;
    }
  });
  $('nextlow').addEventListener('click', function () {
    var cells = $('tbl').querySelectorAll('td.vlow, td.low');
    if (!cells.length) return;
    var cur = document.activeElement, idx = -1, i;
    for (i = 0; i < cells.length; i++) if (cells[i] === cur) { idx = i; break; }
    var next = cells[(idx + 1) % cells.length];
    next.scrollIntoView({ block: 'center', inline: 'center' });
    next.focus();
  });
  $('fixdigits').addEventListener('change', function () { if (words) render(); });

  $('csv').addEventListener('click', function () {
    var csv = SheetLayout.toCSV(SheetLayout.trimGrid(readTable()));
    var blob = new Blob(['﻿' + csv], { type: 'text/csv;charset=utf-8' });
    var a = document.createElement('a');
    a.href = URL.createObjectURL(blob); a.download = 'spreadsheet.csv';
    document.body.appendChild(a); a.click(); a.remove();
    setTimeout(function () { URL.revokeObjectURL(a.href); }, 2000);
  });
  $('copy').addEventListener('click', async function () {
    var tsv = SheetLayout.toTSV(readTable());
    try { await navigator.clipboard.writeText(tsv); $('copy').textContent = 'Copied'; }
    catch (e) { $('copy').textContent = 'Copy failed'; }
    setTimeout(function () { $('copy').textContent = 'Copy for pasting into Excel'; }, 1800);
  });
})();
