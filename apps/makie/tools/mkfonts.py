#!/usr/bin/env python3
"""Adds open-licence fonts to geom/fonts/fonts.json as flattened glyph outlines.
Usage: python3 tools/mkfonts.py   (needs fontTools)"""
import json, os, sys
from fontTools.ttLib import TTFont
from fontTools.pens.basePen import BasePen

CF = '/mnt/skills/examples/canvas-design/canvas-fonts/'
TG = '/usr/share/texmf/fonts/opentype/public/tex-gyre/'
FONTS = [
    ('Inter', '/usr/share/fonts/opentype/inter/Inter-Regular.otf'),
    ('Inter Bold', '/usr/share/fonts/opentype/inter/Inter-Bold.otf'),
    ('Inter Black', '/usr/share/fonts/opentype/inter/Inter-Black.otf'),
    ('Liberation Sans Bold', '/usr/share/fonts/truetype/liberation/LiberationSans-Bold.ttf'),
    ('Liberation Serif Bold', '/usr/share/fonts/truetype/liberation/LiberationSerif-Bold.ttf'),
    ('Poppins', CF + 'Poppins-Regular.ttf'), ('Poppins Bold', CF + 'Poppins-Bold.ttf'),
            ('Lora Bold', CF + 'Lora-Bold.ttf'), ('Libre Baskerville', CF + 'LibreBaskerville-Regular.ttf'),
    ('IBM Plex Serif Bold', CF + 'IBMPlexSerif-Bold.ttf'),
    ('Young Serif', CF + 'YoungSerif-Regular.ttf'),     ('Big Shoulders Bold', CF + 'BigShoulders-Bold.ttf'), ('Boldonse', CF + 'Boldonse-Regular.ttf'),
    ('Erica One', CF + 'EricaOne-Regular.ttf'), ('National Park Bold', CF + 'NationalPark-Bold.ttf'),
    ('Tektur', CF + 'Tektur-Medium.ttf'),     ('Pixelify Sans', CF + 'PixelifySans-Medium.ttf'), ('Italiana', CF + 'Italiana-Regular.ttf'),
    ('Poiret One', CF + 'PoiretOne-Regular.ttf'), ('Handwriting', CF + 'NothingYouCouldDo-Regular.ttf'),
    ('Jura', CF + 'Jura-Medium.ttf'), ('Arsenal SC', CF + 'ArsenalSC-Regular.ttf'),
    ('JetBrains Mono Bold', CF + 'JetBrainsMono-Bold.ttf'), ('IBM Plex Mono Bold', CF + 'IBMPlexMono-Bold.ttf'),
    ('Geist Mono Bold', CF + 'GeistMono-Bold.ttf'), ('DM Mono', CF + 'DMMono-Regular.ttf'), ('Red Hat Mono Bold', CF + 'RedHatMono-Bold.ttf'),
    ('Geometric Bold (TeX Gyre Adventor)', TG + 'texgyreadventor-bold.otf'),
    ('Helvetica-like Bold (TeX Gyre Heros)', TG + 'texgyreheros-bold.otf'),
    ('Courier-like Bold (TeX Gyre Cursor)', TG + 'texgyrecursor-bold.otf'),
    ('Palatino-like Bold (TeX Gyre Pagella)', TG + 'texgyrepagella-bold.otf'),
    ('Times-like Bold (TeX Gyre Termes)', TG + 'texgyretermes-bold.otf'),
]
# fonts with overlapping contours (variable fonts) are left out — the text
# builder needs clean outlines.
CHARS = ''.join(chr(c) for c in range(32, 127)) + '£©®°±µ×÷€éèêëàâäôöüçñÉÀÜÖÄß'

class Flat(BasePen):
    def __init__(self, gs, scale):
        super().__init__(gs); self.s = scale; self.loops = []; self.cur = None
    def _moveTo(self, p):
        self.cur = [p]; self.loops.append(self.cur)
    def _lineTo(self, p): self.cur.append(p)
    def _curveToOne(self, a, b, c):
        p0 = self.cur[-1]
        for i in range(1, 7):
            t = i / 6; mt = 1 - t
            self.cur.append((mt**3*p0[0] + 3*mt*mt*t*a[0] + 3*mt*t*t*b[0] + t**3*c[0], mt**3*p0[1] + 3*mt*mt*t*a[1] + 3*mt*t*t*b[1] + t**3*c[1]))
    def _qCurveToOne(self, a, b):
        p0 = self.cur[-1]
        for i in range(1, 6):
            t = i / 5; mt = 1 - t
            self.cur.append((mt*mt*p0[0] + 2*mt*t*a[0] + t*t*b[0], mt*mt*p0[1] + 2*mt*t*a[1] + t*t*b[1]))
    def _closePath(self): pass
    def _endPath(self): pass

def glyphs(path):
    f = TTFont(path)
    gs = f.getGlyphSet(); cmap = f.getBestCmap(); upm = f['head'].unitsPerEm
    out = {}
    for ch in CHARS:
        gn = cmap.get(ord(ch))
        if gn is None: continue
        pen = Flat(gs, 1 / upm)
        gs[gn].draw(pen)
        loops = []
        for l in pen.loops:
            pts = []
            for (x, y) in l:
                q = (round(x / upm, 4), round(y / upm, 4))
                if pts and abs(pts[-2] - q[0]) < 1e-5 and abs(pts[-1] - q[1]) < 1e-5: continue
                pts += [q[0], q[1]]
            if len(pts) >= 6:
                if abs(pts[0] - pts[-2]) < 1e-5 and abs(pts[1] - pts[-1]) < 1e-5: pts = pts[:-2]
                loops.append(pts)
        out[ch] = {'a': round(gs[gn].width / upm, 4), 'l': loops}
    return out

here = os.path.dirname(os.path.abspath(__file__))
dst = os.path.join(here, '..', 'geom', 'fonts', 'fonts.json')
data = json.load(open(dst))
for name, path in FONTS:
    if not os.path.exists(path):
        print('missing', path, file=sys.stderr); continue
    data[name] = glyphs(path)
    print('added', name, len(data[name]))
json.dump(data, open(dst, 'w'), separators=(',', ':'))
print('fonts:', len(data), 'bytes:', os.path.getsize(dst))
