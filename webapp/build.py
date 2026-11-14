#!/usr/bin/env python3
"""Build Tinkerie web: every app that can run in a browser, as static files.

    python3 webapp/build.py            -> dist/web/tinkerie/  (+ dist/Tinkerie-web-VERSION.tar.gz)

Each app's Go code is compiled to WebAssembly and runs inside its page
(see webapp/webhost). The output is plain files for any web server; the
Pi installer (webapp/install-pi.sh) puts them under Apache.
Made by nrmluvdev.
"""
import html, json, os, re, shutil, subprocess, sys, tarfile

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
OUT = os.path.join(ROOT, 'dist', 'web', 'tinkerie')

# name, title, one-line description, what the web version leaves out
APPS = [
    ('wirie', 'Wirie', 'Wiring diagrams, PCBs and Keebie keyboards', ''),
    ('makie', 'Makie', '3D part designer for 3D printing', ''),
    ('codie', 'Codie', 'Explains code, line by line, in plain words', ''),
    ('fixie', 'Fixie', 'Step-by-step repair guide for almost any device', 'Readings are typed in: the web version can’t take them straight from Meterie, Scopie or Probie.'),
]

def version():
    m = re.search(r'^VERSION=(\S+)', open(os.path.join(ROOT, 'build.sh')).read(), re.M)
    return m.group(1)

def app_version(name):
    src = open(os.path.join(ROOT, 'launcher', 'launcher.go')).read()
    m = re.search(r'\{"%s", "[^"]*", "[^"]*", "([^"]+)"' % name, src)
    return m.group(1) if m else ''

SHELL = '''<!doctype html>
<html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{title}</title>
<link rel="icon" href="../icons/{icon}">
<link rel="stylesheet" href="../shell.css">
</head>
<body class="shell" data-app="{name}">
<div class="tk-bar">
  <a class="home" href="../" title="All Tinkerie apps"><img src="../icons/{icon}" alt="">{title}</a>
  <span class="tk-dim">{ver} · web</span>
  <span class="sp"></span>
  <span id="tkStatus"></span>
  <button class="tk-btn" id="tkFilesBtn" title="Download what this app saved, or add files to it">Files</button>
</div>
<main>
  <div id="tkLoading"><div><div class="spin"></div>Loading {title}…</div></div>
  <iframe id="tkFrame" title="{title}" allow="serial; usb; hid; camera; clipboard-read; clipboard-write; fullscreen"></iframe>
</main>
<div class="tk-modal" id="tkFiles" hidden><div class="tk-box">
  <h2><span>{title} files</span><label class="tk-btn">Add files…<input type="file" id="tkImport" multiple></label><button class="tk-btn" id="tkFilesClose">Close</button></h2>
  <div class="body"><p class="tk-dim">Saved in this browser only. Download anything you want to keep.{note}</p><div id="tkFilesList"></div></div>
</div></div>
<div id="tkToasts"></div>
<script src="../wasm_exec.js"></script>
<script src="../memfs.js"></script>
<script src="../shell.js"></script>
</body></html>
'''

INDEX = '''<!doctype html>
<html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Tinkerie web</title>
<link rel="icon" href="icons/tinkerie.png">
<link rel="stylesheet" href="shell.css">
</head>
<body class="home"><div class="wrap">
<h1><img src="icons/tinkerie.png" alt="">Tinkerie web</h1>
<p class="lead">Free tools for makers, running right in your browser. Nothing to install: everything runs on your own computer,
and what you make is saved in this browser. Version {ver}.</p>
<div class="warn" id="insecure" hidden>This page isn’t on a secure (https://) address, so the apps can’t start. Open it with https:// — see the installer’s notes.</div>
<div class="grid">
{cards}
</div>
<p class="note">Works best in Chrome or Edge on a computer. Firefox and Safari run the apps; the parts that talk to USB or serial devices
need Chrome or Edge. For flashing SD cards, compiling Arduino code and network scopes, use the desktop download from
<a href="https://tinkerie.net">tinkerie.net</a>. Made by nrmluvdev.</p>
</div>
<script>if (!window.isSecureContext) document.getElementById('insecure').hidden = false;</script>
</body></html>
'''

def main():
    ver = version()
    if os.path.isdir(OUT):
        shutil.rmtree(OUT)
    os.makedirs(os.path.join(OUT, 'icons'))
    goroot = subprocess.check_output(['go', 'env', 'GOROOT'], text=True).strip()
    for d in ('lib/wasm', 'misc/wasm'):
        p = os.path.join(goroot, d, 'wasm_exec.js')
        if os.path.exists(p):
            shutil.copy(p, OUT)
            break
    for f in ('memfs.js', 'shell.js', 'shell.css', 'sw.js'):
        shutil.copy(os.path.join(ROOT, 'webapp', 'static', f), OUT)
    shutil.copy(os.path.join(ROOT, 'launcher', 'web', 'tinkerie.png'), os.path.join(OUT, 'icons'))
    env = dict(os.environ, GOOS='js', GOARCH='wasm', CGO_ENABLED='0')
    cards = []
    for name, title, desc, note in APPS:
        icon = next(f for f in os.listdir(os.path.join(ROOT, 'launcher', 'web', 'icons')) if f.startswith(name + '.'))
        shutil.copy(os.path.join(ROOT, 'launcher', 'web', 'icons', icon), os.path.join(OUT, 'icons'))
        d = os.path.join(OUT, name)
        os.makedirs(d)
        print('building', name, flush=True)
        subprocess.check_call(['go', 'build', '-trimpath', '-ldflags', '-s -w', '-o', os.path.join(d, 'app.wasm'),
                               './webapp/apps/' + name], cwd=ROOT, env=env)
        av = app_version(name)
        open(os.path.join(d, 'index.html'), 'w').write(SHELL.format(
            title=html.escape(title), name=name, icon=icon, ver=html.escape(av),
            note=(' ' + html.escape(note)) if note else ''))
        cards.append('<a class="card" href="%s/"><img src="icons/%s" alt=""><div><b>%s</b><div class="d">%s</div>%s</div></a>' % (
            name, icon, html.escape(title), html.escape(desc),
            '<div class="n">%s</div>' % html.escape(note) if note else ''))
    open(os.path.join(OUT, 'index.html'), 'w').write(INDEX.format(ver=html.escape(ver), cards='\n'.join(cards)))
    json.dump({'version': ver, 'apps': [a[0] for a in APPS]}, open(os.path.join(OUT, 'version.json'), 'w'))
    shutil.copy(os.path.join(ROOT, 'webapp', 'install-pi.sh'), os.path.join(ROOT, 'dist', 'web'))
    tgz = os.path.join(ROOT, 'dist', 'Tinkerie-web-%s.tar.gz' % ver)
    with tarfile.open(tgz, 'w:gz') as t:
        t.add(OUT, arcname='tinkerie')
        t.add(os.path.join(ROOT, 'webapp', 'install-pi.sh'), arcname='install-pi.sh')
    print('wrote', OUT, 'and', tgz)

if __name__ == '__main__':
    main()
