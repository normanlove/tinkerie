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

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import webinfo  # what each web version does and doesn't do

def launcher_apps():
    """The apps the desktop launcher has: every one must have a web version."""
    src = open(os.path.join(ROOT, 'launcher', 'launcher.go')).read()
    return re.findall(r'^\t\{"([a-z]+)", "[A-Z][a-z]+", "', src, re.M)

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
<body class="shell" data-app="{name}" data-devices="{devices}">
<div class="tk-bar">
  <a class="home" href="../" title="All Tinkerie apps"><img src="../icons/{icon}" alt="">{title}</a>
  <span class="tk-dim">{ver} · web</span>
  <span class="sp"></span>
  <span id="tkStatus"></span>
{devbtn}
  <a class="tk-btn" id="tkBug" target="_blank" rel="noopener" data-app="{title}" data-ver="{ver}" href="https://tinkerie.net/feedback.html" title="Something wrong, or an idea? Tell nrmluvdev">Report a bug</a>
  <button class="tk-btn" id="tkWebBtn" title="What the web version does and doesn’t do, compared with the desktop app">Web vs desktop</button>
  <button class="tk-btn" id="tkFilesBtn" title="Download what this app saved, or add files to it">Files</button>
</div>
<main>
  <div id="tkLoading"><div><div class="spin"></div>Loading {title}…</div></div>
  <iframe id="tkFrame" title="{title}" allow="serial; usb; hid; camera; microphone; clipboard-read; clipboard-write; fullscreen"></iframe>
</main>
<div class="tk-modal" id="tkFiles" hidden><div class="tk-box">
  <h2><span>{title} files</span><label class="tk-btn">Add files…<input type="file" id="tkImport" multiple></label><button class="tk-btn" id="tkFilesClose">Close</button></h2>
  <div class="body"><p class="tk-dim">Saved in this browser only, on this computer: nobody else's browser can see it. Download anything you want to keep.{note}</p><div id="tkFilesList"></div>
    <h3>On a shared computer?</h3>
    <p class="tk-dim">Anyone who uses this browser profile after you will see your {title} work. Use your own browser profile or a guest window, or clear it when you've finished.</p>
    <button class="tk-btn" id="tkClear" title="Delete everything {title} saved in this browser, and forget its devices">Clear my {title} data</button></div>
</div></div>
<div class="tk-modal" id="tkDevs" hidden><div class="tk-box">
  <h2><span>Connect a device</span><button class="tk-btn" id="tkDevsClose">Close</button></h2>
  <div class="body">
    <p class="tk-dim">The browser only lets {title} use a device you pick here. Plug it in, then click the kind it is and choose it in the list the browser shows. The browser remembers it, so next time {title} finds it by itself.</p>
    <div class="tk-ask" id="tkAsk"></div>
    <div class="tk-warn" id="tkNoDev" hidden>This browser can’t connect to devices. Open this page in <b>Chrome</b> or <b>Edge</b> on a computer (Windows, Mac, Linux or Chromebook).</div>
    <h3>Devices {title} may use</h3>
    <div id="tkGranted"></div>
  </div>
</div></div>
<div class="tk-modal" id="tkWeb" hidden><div class="tk-box">
  <h2><span>{title}: web and desktop</span><button class="tk-btn" id="tkWebClose">Close</button></h2>
  <div class="body">{webinfo}</div>
</div></div>
<div class="tk-twice" id="tkTwice" hidden>{title} is open in another tab too. Both tabs save to the same files in this browser, so one can overwrite the other’s work: use one tab, or close the other.</div>
<div id="tkToasts"></div>
<script src="../wasm_exec.js"></script>
<script src="../memfs.js"></script>
<script src="../tkdev.js"></script>
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
<p class="note">Made for Chrome and Edge on a computer (Windows, Mac, Linux or Chromebook): they connect to serial ports, USB scopes and analysers and meter cables
with the <b>Connect device</b> button. Firefox and Safari run the apps but can’t connect to devices. A few things need the desktop app
(writing SD cards, compiling Arduino code, network scopes and printers): each app’s <b>Web vs desktop</b> button lists them, and the
<a href="https://tinkerie.net">desktop download</a> is free too. Something wrong, or an idea? <a href="https://tinkerie.net/feedback.html?where=Tinkerie%20web%20(in%20Chrome)">Report it</a>. Made by nrmluvdev.</p>
<h2 class="priv">Your work is yours</h2>
<p class="note">There are no accounts and no server: each app runs in your own browser, and what you make is saved only in this browser on this computer,
so nobody else can see it. Each tab and each app keeps to itself. <b>On a shared computer</b> (a school, library or club), anyone who uses this
browser profile after you will see your work: use your own browser profile or a guest window, or clear it when you’ve finished.</p>
<p><button class="tk-btn" id="clearAll">Clear all my Tinkerie web data</button> <span class="tk-dim" id="clearMsg"></span></p>
</div>
<script>if (!window.isSecureContext) document.getElementById('insecure').hidden = false;</script>
<script src="home.js" data-apps="{apps}"></script>
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
    for f in ('memfs.js', 'shell.js', 'shell.css', 'sw.js', 'tkdev.js', 'iso.js', 'home.js'):
        shutil.copy(os.path.join(ROOT, 'webapp', 'static', f), OUT)
    shutil.copy(os.path.join(ROOT, 'launcher', 'web', 'tinkerie.png'), os.path.join(OUT, 'icons'))
    env = dict(os.environ, GOOS='js', GOARCH='wasm', CGO_ENABLED='0')
    missing = [a for a in launcher_apps() if a not in webinfo.APPS]
    if missing:
        sys.exit('webapp/webinfo.py has no web version for: ' + ', '.join(missing) +
                 ' (every app needs one, or its features listed under missing)')
    cards = []
    for name in webinfo.ORDER:
        info = webinfo.APPS[name]
        title, desc = info['title'], info['desc']
        icon = next(f for f in os.listdir(os.path.join(ROOT, 'launcher', 'web', 'icons')) if f.startswith(name + '.'))
        shutil.copy(os.path.join(ROOT, 'launcher', 'web', 'icons', icon), os.path.join(OUT, 'icons'))
        d = os.path.join(OUT, name)
        os.makedirs(d)
        print('building', name, flush=True)
        subprocess.check_call(['go', 'build', '-trimpath', '-ldflags', '-s -w', '-o', os.path.join(d, 'app.wasm'),
                               './webapp/apps/' + name], cwd=ROOT, env=env)
        av = app_version(name)
        devbtn = ('  <button class="tk-btn" id="tkDevBtn" title="Let %s use a serial port, USB device or meter cable">Connect device</button>' % html.escape(title)) if info['devices'] else ''
        note = ' Devices: pick them with Connect device (Chrome or Edge).' if info['devices'] else ''
        open(os.path.join(d, 'index.html'), 'w').write(SHELL.format(
            title=html.escape(title), name=name, icon=icon, ver=html.escape(av), devices=','.join(info['devices']),
            devbtn=devbtn, webinfo=webinfo.html_list(name, html.escape), note=note))
        short = info['missing'][0].split(':')[0] if info['missing'] else ''
        cards.append('<a class="card" href="%s/"><img src="icons/%s" alt=""><div><b>%s</b><div class="d">%s</div>%s</div></a>' % (
            name, icon, html.escape(title), html.escape(desc),
            '<div class="n">Desktop only: %s%s</div>' % (html.escape(short[0].lower() + short[1:]), ', and more' if len(info['missing']) > 1 else '') if short else ''))
    open(os.path.join(OUT, 'index.html'), 'w').write(INDEX.format(ver=html.escape(ver), cards='\n'.join(cards), apps=','.join(webinfo.ORDER)))
    json.dump({'version': ver, 'apps': webinfo.ORDER}, open(os.path.join(OUT, 'version.json'), 'w'))
    shutil.copy(os.path.join(ROOT, 'webapp', 'install-pi.sh'), os.path.join(ROOT, 'dist', 'web'))
    tgz = os.path.join(ROOT, 'dist', 'Tinkerie-web-%s.tar.gz' % ver)
    with tarfile.open(tgz, 'w:gz') as t:
        t.add(OUT, arcname='tinkerie')
        t.add(os.path.join(ROOT, 'webapp', 'install-pi.sh'), arcname='install-pi.sh')
    print('wrote', OUT, 'and', tgz)

if __name__ == '__main__':
    main()
