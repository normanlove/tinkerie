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
      out.push({ text: t, x0: b.x0, y0: b.y0, x1: b.x1, y1: b.y1 });
    });
    return out;
  }

  /* gapFactor: a gap between two words wider than gapFactor x the typical
     text height starts a new cell. Smaller = more columns. */
  function buildGrid(rawWords, gapFactor) {
    var words = cleanWords(rawWords);
    if (!words.length) return [];
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
        } else {
          cells.push({ text: w.text, x0: w.x0, x1: w.x1 });
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
    var grid = cellRows.map(function (cs) {
      var row = new Array(cols.length).fill('');
      cs.forEach(function (c) {
        var mid = (c.x0 + c.x1) / 2, idx = -1;
        for (var i = 0; i < cols.length; i++) {
          if (mid >= cols[i].x0 && mid <= cols[i].x1) { idx = i; break; }
        }
        if (idx < 0) idx = 0;
        row[idx] = row[idx] ? row[idx] + ' ' + c.text : c.text;
      });
      return row;
    });
    return trimGrid(grid);
  }

  function trimGrid(grid) {
    grid = grid.filter(function (r) { return r.some(function (c) { return String(c).trim() !== ''; }); });
    if (!grid.length) return [];
    var n = Math.max.apply(null, grid.map(function (r) { return r.length; }));
    grid = grid.map(function (r) { r = r.slice(); while (r.length < n) r.push(''); return r; });
    var keep = [];
    for (var j = 0; j < n; j++) {
      if (grid.some(function (r) { return String(r[j]).trim() !== ''; })) keep.push(j);
    }
    return grid.map(function (r) { return keep.map(function (j) { return r[j]; }); });
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

  var api = { buildGrid: buildGrid, trimGrid: trimGrid, toCSV: toCSV, toTSV: toTSV, cleanWords: cleanWords };
  if (typeof module !== 'undefined' && module.exports) module.exports = api;
  else root.SheetLayout = api;
})(typeof self !== 'undefined' ? self : this);
