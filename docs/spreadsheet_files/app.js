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

  /* Grey-scale, enlarge small images, and optionally erase long table lines. */
  function prepare(removeGrid) {
    var w = img.naturalWidth, h = img.naturalHeight;
    var scale = 1;
    if (w < 2000) scale = Math.min(3, 2000 / w);
    else if (w > 4000) scale = 4000 / w;
    var cw = Math.round(w * scale), ch = Math.round(h * scale);
    var c = document.createElement('canvas');
    c.width = cw; c.height = ch;
    var ctx = c.getContext('2d', { willReadFrequently: true });
    ctx.fillStyle = '#fff'; ctx.fillRect(0, 0, cw, ch);
    ctx.imageSmoothingQuality = 'high';
    ctx.drawImage(img, 0, 0, cw, ch);
    var d = ctx.getImageData(0, 0, cw, ch), p = d.data, n = cw * ch;
    var g = new Uint8Array(n), hist = new Uint32Array(256), i;
    for (i = 0; i < n; i++) { var v = (p[i*4] * 77 + p[i*4+1] * 150 + p[i*4+2] * 29) >> 8; g[i] = v; hist[v]++; }
    // Otsu threshold
    var sum = 0, t; for (t = 0; t < 256; t++) sum += t * hist[t];
    var wB = 0, sB = 0, best = 0, thr = 128;
    for (t = 0; t < 256; t++) {
      wB += hist[t]; if (!wB) continue;
      var wF = n - wB; if (!wF) break;
      sB += t * hist[t];
      var mB = sB / wB, mF = (sum - sB) / wF, bt = wB * wF * (mB - mF) * (mB - mF);
      if (bt > best) { best = bt; thr = t; }
    }
    var dark = new Uint8Array(n);
    for (i = 0; i < n; i++) dark[i] = g[i] <= thr ? 1 : 0;
    if (removeGrid) {
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
    return { canvas: c, scale: scale };
  }

  /* ---------- OCR (engine, language data and worker are all files next to this page) ---------- */
  function loadScript(src) {
    return new Promise(function (res, rej) {
      var s = document.createElement('script');
      s.src = src; s.onload = res; s.onerror = function () { rej(new Error('Could not load ' + src)); };
      document.head.appendChild(s);
    });
  }

  async function getWorker() {
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
        if (m.status === 'recognizing text') { $('bar').value = m.progress; setStatus('Reading the text... ' + Math.round(m.progress * 100) + '%'); }
        else if (m.status) setStatus(m.status.charAt(0).toUpperCase() + m.status.slice(1) + '...');
      }
    });
    return worker;
  }

  $('go').addEventListener('click', async function () {
    if (!img) return;
    setErr(''); $('go').disabled = true;
    $('bar').classList.remove('hidden'); $('bar').value = 0;
    $('result').classList.add('hidden');
    try {
      setStatus('Preparing the image...');
      await new Promise(function (r) { setTimeout(r, 30); });
      var prep = prepare($('grid').value === 'remove');
      var wk = await getWorker();
      await wk.setParameters({ tessedit_pageseg_mode: $('psm').value, preserve_interword_spaces: '1' });
      var res = await wk.recognize(prep.canvas);
      var data = res.data, list = data.words;
      if (!list || !list.length) {
        list = [];
        (data.lines || []).forEach(function (l) { (l.words || []).forEach(function (w) { list.push(w); }); });
      }
      words = list;
      if (!words.length) { setErr('No text was found in that image. Try a clearer or larger picture.'); setStatus(''); }
      else { render(); setStatus('Done.'); $('result').classList.remove('hidden'); }
    } catch (e) {
      setErr('Something went wrong: ' + (e && e.message ? e.message : e));
      setStatus('');
      try { if (worker) await worker.terminate(); } catch (x) {}
      worker = null;
    }
    $('bar').classList.add('hidden');
    $('go').disabled = false;
  });

  /* ---------- table ---------- */
  function render() {
    var grid = SheetLayout.buildGrid(words, parseFloat($('gap').value));
    var t = $('tbl'); t.textContent = '';
    grid.forEach(function (r) {
      var tr = document.createElement('tr');
      r.forEach(function (v) { var td = document.createElement('td'); td.contentEditable = 'true'; td.spellcheck = false; td.textContent = v; tr.appendChild(td); });
      t.appendChild(tr);
    });
    $('size').textContent = grid.length + ' rows x ' + (grid[0] ? grid[0].length : 0) + ' columns';
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
