/* Turns OCR words (with positions) into a grid of rows and columns.
   Pure functions, no network, no DOM. Works in the browser and in Node. */
(function (root) {
  'use strict';

  function median(a) {
    if (!a.length) return 0;
    var s = a.slice().sort(function (x, y) { return x - y; });
    var m = s.length >> 1;
    return s.length % 2 ? s[m] : (s[m - 1] + s[m]) / 2;
  }

  // Remove grid-line debris and low-confidence junk.
  function cleanWords(words) {
    var out = [];
    (words || []).forEach(function (w) {
      var t = String(w.text || '').replace(/^[|¦!]+(?=.)|(?<=.)[|¦]+$/g, '').trim();
      if (!t) return;
      if (/^[|¦_=~\\\/\[\]{}()<>]{1,}$/.test(t)) return;
      if (w.confidence !== undefined && w.confidence < 15) return;
      var b = w.bbox;
      if (!b || !(b.x1 > b.x0) || !(b.y1 > b.y0)) return;
      out.push({
        text: t, x0: b.x0, y0: b.y0, x1: b.x1, y1: b.y1,
        conf: typeof w.confidence === 'number' ? w.confidence : 100
      });
    });
    return out;
  }

  /* gapFactor: a gap between two words wider than gapFactor x the typical
     text height starts a new cell. Smaller = more columns.
     Returns { grid, conf }: conf has the same shape as grid and holds the
     lowest OCR confidence (0-100) of the words in each cell (100 if empty). */
  function buildTable(rawWords, gapFactor) {
    var words = cleanWords(rawWords);
    if (!words.length) return { grid: [], conf: [] };
    gapFactor = gapFactor > 0 ? gapFactor : 1.0;
    var medH = median(words.map(function (w) { return w.y1 - w.y0; })) || 10;

    // 1. rows
    words.sort(function (a, b) { return (a.y0 + a.y1) - (b.y0 + b.y1); });
    var rows = [];
    words.forEach(function (w) {
      var cy = (w.y0 + w.y1) / 2;
      var r = rows[rows.length - 1];
      if (r && Math.abs(cy - r.cy) < 0.55 * medH) {
        r.words.push(w);
        r.cy = (r.cy * (r.words.length - 1) + cy) / r.words.length;
      } else {
        rows.push({ cy: cy, words: [w] });
      }
    });

    // 2. cells within each row
    var cellRows = rows.map(function (r) {
      r.words.sort(function (a, b) { return a.x0 - b.x0; });
      var cells = [];
      r.words.forEach(function (w) {
        var c = cells[cells.length - 1];
        if (c && w.x0 - c.x1 < gapFactor * medH) {
          c.text += ' ' + w.text;
          c.x1 = Math.max(c.x1, w.x1);
          c.conf = Math.min(c.conf, w.conf);
        } else {
          cells.push({ text: w.text, x0: w.x0, x1: w.x1, conf: w.conf });
        }
      });
      return cells;
    });

    // 3. column boundaries: stretches of the page where no cell has any text
    var all = [];
    cellRows.forEach(function (cs) { cs.forEach(function (c) { all.push(c); }); });
    all.sort(function (a, b) { return a.x0 - b.x0; });
    var cols = [];
    all.forEach(function (c) {
      var k = cols[cols.length - 1];
      if (k && c.x0 <= k.x1) k.x1 = Math.max(k.x1, c.x1);
      else cols.push({ x0: c.x0, x1: c.x1 });
    });

    // 4. place cells in columns
    var grid = [], conf = [];
    cellRows.forEach(function (cs) {
      var row = new Array(cols.length).fill('');
      var crow = new Array(cols.length).fill(100);
      cs.forEach(function (c) {
        var mid = (c.x0 + c.x1) / 2, idx = -1;
        for (var i = 0; i < cols.length; i++) {
          if (mid >= cols[i].x0 && mid <= cols[i].x1) { idx = i; break; }
        }
        if (idx < 0) idx = 0;
        row[idx] = row[idx] ? row[idx] + ' ' + c.text : c.text;
        crow[idx] = Math.min(crow[idx], c.conf);
      });
      grid.push(row); conf.push(crow);
    });
    return trimTable(grid, conf);
  }

  function buildGrid(rawWords, gapFactor) { return buildTable(rawWords, gapFactor).grid; }

  /* Drops empty rows and columns, and pads short rows. If conf is given it is
     trimmed the same way. */
  function trimTable(grid, conf) {
    var keepR = [];
    grid.forEach(function (r, i) { if (r.some(function (c) { return String(c).trim() !== ''; })) keepR.push(i); });
    if (!keepR.length) return { grid: [], conf: [] };
    var n = Math.max.apply(null, keepR.map(function (i) { return grid[i].length; }));
    var keepC = [];
    for (var j = 0; j < n; j++) {
      if (keepR.some(function (i) { return String(grid[i][j] == null ? '' : grid[i][j]).trim() !== ''; })) keepC.push(j);
    }
    return {
      grid: keepR.map(function (i) { return keepC.map(function (j) { return grid[i][j] == null ? '' : grid[i][j]; }); }),
      conf: keepR.map(function (i) { return keepC.map(function (j) { return conf && conf[i] && conf[i][j] != null ? conf[i][j] : 100; }); })
    };
  }

  function trimGrid(grid) { return trimTable(grid).grid; }

  /* ---- number-column repair ----
     OCR often reads digits as look-alike letters (O for 0, l for 1, S for 5).
     In a column that is mostly numbers, fix those look-alikes, but only where
     the result is then a clean number. Row 0 is treated as a header and left
     alone. Returns { grid, changes: [{r, c, from, to}] }. */
  var LOOKALIKE = { O: '0', o: '0', Q: '0', D: '0', l: '1', I: '1', i: '1', '|': '1', '!': '1', S: '5', s: '5', B: '8', Z: '2', z: '2', g: '9', q: '9' };
  var NUM_RE = /^[(\-−+]?[£$€]?\s?\d[\d,. ]*\d?\s?[%)]?$|^[(\-−+]?[£$€]?\s?\d\s?[%)]?$/;

  function isNumberish(s) { return NUM_RE.test(s); }
  function swap(s) { return s.replace(/[OoQDlIi|!SsBZzgq]/g, function (ch) { return LOOKALIKE[ch]; }); }

  /* The pound sign is often read as F, L or E. Put it right when the cell is
     such a letter followed by an amount. Safe-guards: it must look like money
     (pence, or thousands commas) or the column must already contain a real £,
     so a column of part codes like "F12" is left alone. E is only trusted when
     the column already has a £ (E can be a genuine euro misread). */
  var CUR_RE = /^([FfLE])\s?(\d{1,3}(?:,\d{3})+(?:\.\d{1,2})?|\d+\.\d{2}|\d+)$/;
  function fixCurrency(out, changes) {
    if (!out.length) return;
    var ncol = out[0].length, c, r;
    for (c = 0; c < ncol; c++) {
      var hasPound = out.some(function (row) { return /£/.test(String(row[c])); });
      for (r = 1; r < out.length; r++) {
        var s = String(out[r][c]).trim(), m = CUR_RE.exec(s);
        if (!m) continue;
        var strong = /,|\.\d{2}$/.test(m[2]);
        if (m[1] === 'E' ? hasPound : (hasPound || strong)) {
          var t = '£' + m[2];
          changes.push({ r: r, c: c, from: s, to: t });
          out[r][c] = t;
        }
      }
    }
  }

  function fixNumberColumns(grid) {
    var out = grid.map(function (r) { return r.slice(); });
    var changes = [];
    fixCurrency(out, changes);
    if (out.length < 3) return { grid: out, changes: changes };
    var ncol = out[0].length;
    for (var c = 0; c < ncol; c++) {
      var filled = 0, numeric = 0, r;
      for (r = 1; r < out.length; r++) {
        var v = String(out[r][c]).trim();
        if (!v) continue;
        filled++;
        // counts as a number if it is one, or becomes one once look-alikes are
        // swapped AND it already contains a real digit (so "SOS" never counts)
        if (isNumberish(v) || (/\d/.test(v) && isNumberish(swap(v)))) numeric++;
      }
      if (filled < 2 || numeric / filled < 0.6) continue;     // not a number column
      for (r = 1; r < out.length; r++) {
        var s = String(out[r][c]).trim();
        if (!s || isNumberish(s)) continue;
        // only attempt cells made of digits, number punctuation and look-alikes
        if (!/^[\d.,%£$€()\-−+\s OoQDlIi|!SsBZzgq]+$/.test(s)) continue;
        var t = s.replace(/[OoQDlIi|!SsBZzgq]/g, function (ch) { return LOOKALIKE[ch]; });
        if (/\d/.test(s) || s.length <= 3) {
          if (isNumberish(t) && t !== s) { changes.push({ r: r, c: c, from: s, to: t }); out[r][c] = t; }
        }
      }
    }
    return { grid: out, changes: changes };
  }

  /* ---- second look at number-like words ----
     Words that look like a damaged number (digits mixed with F, O, l, S...) are
     re-read on their own with a numbers-only character set. Ordinary words are
     never candidates, so wording is not disturbed. */
  var NUMCLEAN = /^[(\-−+]?[£$€]?\s?\d[\d,. ]*[%)]?$/;
  var NUMCHARS = /^[\d.,%£$€()\-−+\/:FEfeLOoQDlIi|!SsBZzgq]+$/;

  function rereadCandidates(words) {
    var out = [];
    (words || []).forEach(function (w, i) {
      var t = String(w.text || '').trim();
      if (!t || t.length > 20 || !/\d/.test(t) || !NUMCHARS.test(t)) return;
      if (/\d[-\/:]\d/.test(t)) return;                                 // dates and times: leave as read
      var digits = (t.match(/\d/g) || []).length, alnum = (t.match(/[A-Za-z\d]/g) || []).length;
      if (digits * 2 < alnum) return;                                   // mostly letters: it is a word
      var conf = typeof w.confidence === 'number' ? w.confidence : 100;
      if (conf < 92 || !NUMCLEAN.test(t)) out.push(i);
    });
    return out;
  }

  function acceptReread(word, newText, newConf) {
    var t = String(newText || '').replace(/\s+/g, ' ').trim(), old = String(word.text || '').trim();
    if (!t || !NUMCLEAN.test(t) || !NUMCHARS.test(old)) return false;
    if (Math.abs(t.length - old.length) > 3) return false;
    return !(newConf < 30);
  }

  /* How good a reading looks: confidence weighted by amount of text. Used to
     choose between two attempts at the same image. */
  function scoreWords(words) {
    var s = 0;
    (words || []).forEach(function (w) {
      var c = typeof w.confidence === 'number' ? w.confidence : 100;
      if (c >= 15) s += c * String(w.text || '').trim().length;
    });
    return s;
  }

  /* ---- text size ----
     dark: Uint8Array (cw*ch), 1 = ink. Returns the typical height in pixels of
     a line of text (median height of runs of rows that contain ink), or 0 if it
     cannot tell. Long vertical and horizontal table lines are ignored so they
     do not join the lines of text together. */
  function estimateTextHeight(dark, cw, ch) {
    var skip = new Uint8Array(cw * ch), x, y, run, k;
    var minRun = Math.max(40, Math.round(ch * 0.08));
    for (x = 0; x < cw; x++) {
      run = 0;
      for (y = 0; y <= ch; y++) {
        if (y < ch && dark[y * cw + x]) run++;
        else { if (run >= minRun) for (k = y - run; k < y; k++) skip[k * cw + x] = 1; run = 0; }
      }
    }
    var lim = cw * 0.5, runs = [];
    run = 0;
    for (y = 0; y <= ch; y++) {
      var cnt = 0;
      if (y < ch) { var o = y * cw; for (x = 0; x < cw; x++) if (dark[o + x] && !skip[o + x]) cnt++; }
      if (y < ch && cnt > 0 && cnt < lim) run++;
      else { if (run >= 3) runs.push(run); run = 0; }
    }
    return runs.length ? median(runs) : 0;
  }

  /* ---- tilt detection ----
     dark: Uint8Array (cw*ch) with 1 where a pixel is ink. Finds the angle (in
     degrees, positive = page turned clockwise as seen on screen) at which the
     rows of ink line up best. To straighten the page, rotate it by -angle.
     Returns 0 when there is no clear tilt. */
  function estimateSkew(dark, cw, ch) {
    var step = Math.max(1, Math.round(Math.max(cw, ch) / 900));
    var gw = Math.ceil(cw / step), gh = Math.ceil(ch / step);
    var mark = new Uint8Array(gw * gh), x, y;
    for (y = 0; y < ch; y++) {
      var gy = ((y / step) | 0) * gw;
      for (x = 0; x < cw; x++) if (dark[y * cw + x]) mark[gy + ((x / step) | 0)] = 1;
    }
    var xs = [], ys = [];
    for (y = 0; y < gh; y++) for (x = 0; x < gw; x++) if (mark[y * gw + x]) { xs.push(x); ys.push(y); }
    var np = xs.length;
    if (np < 200) return 0;
    var off = gw * 0.25 + 4, nb = Math.ceil(gh + gw * 0.25 + off + 8);
    var bins = new Float32Array(nb);
    function score(deg) {
      var r = deg * Math.PI / 180, sn = Math.sin(r), cs = Math.cos(r), i, s = 0;
      bins.fill(0);
      for (i = 0; i < np; i++) bins[(ys[i] * cs - xs[i] * sn + off) | 0]++;
      for (i = 0; i < nb; i++) s += bins[i] * bins[i];
      return s;
    }
    var best = 0, bs = score(0), base = bs, a;
    for (a = -10; a <= 10; a += 0.5) { var sc = score(a); if (sc > bs) { bs = sc; best = a; } }
    var c0 = best;
    for (a = c0 - 0.5; a <= c0 + 0.5; a += 0.1) { var s2 = score(a); if (s2 > bs) { bs = s2; best = a; } }
    if (Math.abs(best) < 0.3 || bs < base * 1.03) return 0;
    return Math.round(best * 10) / 10;
  }

  function csvField(v) {
    v = String(v == null ? '' : v);
    return /[",\r\n]/.test(v) ? '"' + v.replace(/"/g, '""') + '"' : v;
  }

  function toCSV(grid) {
    return grid.map(function (r) { return r.map(csvField).join(','); }).join('\r\n') + '\r\n';
  }

  function toTSV(grid) {
    return grid.map(function (r) {
      return r.map(function (v) { return String(v).replace(/[\t\r\n]+/g, ' '); }).join('\t');
    }).join('\n');
  }

  var api = {
    buildGrid: buildGrid, buildTable: buildTable, trimGrid: trimGrid, trimTable: trimTable,
    fixNumberColumns: fixNumberColumns, estimateSkew: estimateSkew, estimateTextHeight: estimateTextHeight, rereadCandidates: rereadCandidates, acceptReread: acceptReread, scoreWords: scoreWords, toCSV: toCSV, toTSV: toTSV, cleanWords: cleanWords
  };
  if (typeof module !== 'undefined' && module.exports) module.exports = api;
  else root.SheetLayout = api;
})(typeof self !== 'undefined' ? self : this);
