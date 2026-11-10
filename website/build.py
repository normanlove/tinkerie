#!/usr/bin/env python3
"""Builds the Tinkerie website into ../docs (served by GitHub Pages).

    python3 website/build.py

Pages: index.html, start.html and one page per app. The text lives in
content.py; the look in site.css. Download links use the version in
build.sh, so run this again after a new release. Made by nrmluvdev.
"""
import html, os, re, shutil, sys

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
from content import APPS, APP, FAQ  # noqa: E402

ROOT = os.path.dirname(HERE)
OUT = os.path.join(ROOT, "docs")
REPO = "https://github.com/normanlove/tinkerie"
VERSION = re.search(r"^VERSION=(\S+)", open(os.path.join(ROOT, "build.sh")).read(), re.M).group(1)
REL = f"{REPO}/releases/download/v{VERSION}"
FONTS = ("https://fonts.googleapis.com/css2?family=Atkinson+Hyperlegible:ital,wght@0,400;0,700;1,400"
         "&family=JetBrains+Mono:wght@500;700&family=Unbounded:wght@500;600;700&display=swap")

e = html.escape

ICON = {
  "download": '<path d="M12 3v12m0 0-5-5m5 5 5-5M4 19h16"/>',
  "arrow": '<path d="M5 12h14m-6-6 6 6-6 6"/>',
  "back": '<path d="M19 12H5m6-6-6 6 6 6"/>',
  "free": '<path d="M20 12v8H4v-8M2 7h20v5H2zM12 20V7m0 0H7.5a2.5 2.5 0 1 1 0-5C11 2 12 7 12 7zm0 0h4.5a2.5 2.5 0 1 0 0-5C13 2 12 7 12 7z"/>',
  "box": '<path d="M3 7l9-4 9 4-9 4-9-4zm0 0v10l9 4 9-4V7M12 11v10"/>',
  "teach": '<path d="M9 18h6m-5 3h4M12 3a6 6 0 0 0-4 10.5c.8.8 1 1.5 1 2.5h6c0-1 .2-1.7 1-2.5A6 6 0 0 0 12 3z"/>',
  "demo": '<path d="M5 3l14 9-14 9V3z"/>',
  "systems": '<rect x="3" y="4" width="18" height="12" rx="2"/><path d="M8 20h8m-4-4v4"/>',
  "github": '<path d="M9 19c-4 1.3-4-2-6-2.5M15 21v-3.5c0-1 .1-1.4-.5-2 2.8-.3 5.5-1.4 5.5-6a4.6 4.6 0 0 0-1.3-3.2 4.2 4.2 0 0 0-.1-3.2s-1-.3-3.4 1.3a11.6 11.6 0 0 0-6 0C6.8 2.8 5.8 3.1 5.8 3.1a4.2 4.2 0 0 0-.1 3.2A4.6 4.6 0 0 0 4.4 9.5c0 4.6 2.7 5.7 5.5 6-.6.6-.6 1.2-.5 2V21"/>',
  "info": '<circle cx="12" cy="12" r="9"/><path d="M12 11v6m0-9.5v.5"/>',
  "warn": '<path d="M12 3 2 20h20L12 3zm0 7v5m0 3v.5"/>',
  "theme": '<path d="M12 3a9 9 0 1 0 9 9 7 7 0 0 1-9-9z"/>',
  "chev": '<path d="m6 9 6 6 6-6"/>',
  "zoom": '<circle cx="11" cy="11" r="7"/><path d="m20 20-4-4M11 8v6M8 11h6"/>',
}


def svg(name, cls=""):
    c = f' class="{cls}"' if cls else ""
    return (f'<svg{c} viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" '
            f'stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">{ICON[name]}</svg>')


DOWNLOADS = [  # os key, label for the button, row label, file, note
  ("windows", "Windows", "Windows 10 or 11", "Tinkerie.exe", "Just run it. Or get the zip below if your network blocks .exe files."),
  ("windows-zip", "Windows", "Windows (zip, with source)", f"Tinkerie-{VERSION}-windows.zip", "The same program in a zip, with the source code."),
  ("mac", "Mac (Apple silicon)", "Mac with M1, M2, M3 or M4", f"Tinkerie-{VERSION}-mac-apple-silicon.zip", "macOS 11 or newer."),
  ("mac-intel", "Mac (Intel)", "Mac with an Intel processor", f"Tinkerie-{VERSION}-mac-intel.zip", "macOS 11 or newer."),
  ("linux", "Linux", "Linux PC (64-bit)", f"Tinkerie-{VERSION}-linux-x86_64.tar.gz", "Intel or AMD processor."),
  ("pi", "Raspberry Pi", "Raspberry Pi OS 64-bit", f"Tinkerie-{VERSION}-pi-64bit.tar.gz", "Pi 3, 4, 5 and newer."),
  ("pi32", "Raspberry Pi (32-bit)", "Raspberry Pi OS 32-bit", f"Tinkerie-{VERSION}-pi-32bit.tar.gz", "Pi 2, 3, 4, 5 and Zero 2 W."),
]


def head(title, desc, extra_style=""):
    return f"""<title>{e(title)}</title>
<meta name="claude:viewport-fit-sides" content="cover">
<meta name="description" content="{e(desc)}">
<meta property="og:title" content="{e(title)}">
<meta property="og:description" content="{e(desc)}">
<link rel="icon" href="img/icons/tinkerie.png">
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link rel="stylesheet" href="{FONTS}">
<link rel="stylesheet" href="site.css">{extra_style}"""


def header(current):
    menu = "".join(
        f'<a href="{a["id"]}.html"><img src="img/icons/{a["icon"]}" alt=""><span><b>{a["name"]}</b><small>{e(a["short"].split(",")[0].split(".")[0])}</small></span></a>'
        for a in APPS)
    cur = lambda k: ' aria-current="page"' if current == k else ""
    return f"""<a class="skip" href="#main">Skip to the content</a>
<header class="top"><div class="wrap">
  <a class="brand" href="index.html"><img src="img/icons/tinkerie.png" alt="">Tinkerie</a>
  <nav class="nav" aria-label="Main">
    <div class="apps-menu"><button type="button" aria-expanded="false" aria-haspopup="true">Apps {svg("chev")}</button><div class="menu">{menu}<a class="menu-start" href="start.html">Getting started guide →</a></div></div>
    <a class="hide-s" href="start.html"{cur("start")}>Getting started</a>
    <a class="hide-s" href="{REPO}">GitHub</a>
    <a class="btn btn-go" href="index.html#download" aria-label="Download">{svg("download")}<span class="lbl">Download</span></a>
    <button class="theme" id="theme" type="button" aria-label="Switch theme">{svg("theme")}</button>
  </nav>
</div></header>"""


def footer():
    links = "".join(f'<li><a href="{a["id"]}.html">{a["name"]}</a></li>' for a in APPS)
    return f"""<footer><div class="wrap">
  <div class="stack">
    <a class="brand" href="index.html"><img src="img/icons/tinkerie.png" alt="">Tinkerie</a>
    <p>Free maker tools, designed and created by <b>nrmluvdev</b>. Version {VERSION}: Tinkerie is in alpha.</p>
    <p>Free software under the GNU GPL, version 3 or later. It comes with no warranty.</p>
  </div>
  <div><h4>The apps</h4><ul>{links}</ul></div>
  <div><h4>Help and source</h4><ul>
    <li><a href="start.html">Getting started</a></li>
    <li><a href="index.html#faq">Questions</a></li>
    <li><a href="{REPO}/releases">All downloads</a></li>
    <li><a href="{REPO}">Source code on GitHub</a></li>
    <li><a href="{REPO}/issues">Report a problem</a></li>
    <li><a href="code-signing.html">Code signing policy</a></li>
  </ul></div>
</div></footer>
<script src="site.js"></script>"""


def page(fname, title, desc, current, body, app=None, style=""):
    accent = f' style="--app:{app["color"]}"' if app else ""
    content = f"""{head(title, desc, style)}
<div{accent}>
{header(current)}
<main id="main">
{body}
</main>
{footer()}
</div>"""
    full = f'<!doctype html>\n<html lang="en-GB">\n<head>\n<meta charset="utf-8">\n<meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">\n{head(title, desc, style)}\n</head>\n<body>\n<div{accent}>\n{header(current)}\n<main id="main">\n{body}\n</main>\n{footer()}\n</div>\n</body>\n</html>\n'
    with open(os.path.join(OUT, fname), "w", encoding="utf-8") as f:
        f.write(full)
    # The same page without the document wrapper, for publishing as an artifact.
    os.makedirs(os.path.join(HERE, "artifact"), exist_ok=True)
    with open(os.path.join(HERE, "artifact", fname), "w", encoding="utf-8") as f:
        f.write(content if fname == "index.html" else full)


def shot(name, caption, tag=None, sm=False, eager=False):
    src = f"img/shots/{name}{'-sm' if sm else ''}.webp"
    t = f'<span class="tag">{e(tag)}</span>' if tag else ""
    load = "" if eager else ' loading="lazy"'
    return (f'<figure><div class="shot">{t}<button class="zoom" type="button" data-full="img/shots/{name}.webp" '
            f'data-caption="{e(caption)}" aria-label="Enlarge: {e(caption)}"><img src="{src}" alt="{e(caption)}" '
            f'width="{720 if sm else 1440}" height="{450 if sm else 900}"{load}></button></div>'
            f'<figcaption>{e(caption)}</figcaption></figure>')


def dl_rows():
    rows = []
    for key, short, label, file, note in DOWNLOADS:
        os_attr = key if key not in ("windows-zip",) else "windows-zip"
        rows.append(f'<a class="row" data-os="{os_attr}" data-name="{e(short)}" href="{REL}/{file}">'
                    f'<span class="os">{e(label)}</span><span><span class="file">{e(file)}</span><br><span class="muted" style="font-size:.9rem">{e(note)}</span></span>'
                    f'<span class="get">{svg("download")} Download</span></a>')
    return "".join(rows)


# ------------------------------------------------------------------ home

def build_index():
    cards = "".join(f"""<a class="app-card" href="{a['id']}.html" style="--app:{a['color']}">
  <div class="pic"><img src="img/shots/{a['shots'][0][0]}-sm.webp" alt="{e(a['name'])}: {e(a['shots'][0][1])}" loading="lazy" width="720" height="450"></div>
  <div class="body"><div class="name"><img src="img/icons/{a['icon']}" alt=""><h3>{a['name']}</h3></div><p>{e(a['short'])}</p></div>
  <div class="foot"><span class="need">{e(a['needshort'])}</span><span class="more">Explore {svg("arrow")}</span></div>
</a>""" for a in APPS)
    cards += f"""<a class="app-card intro" href="start.html">
  <span class="label" style="color:inherit;opacity:.8">New to all this?</span>
  <div class="stack"><h3>Start with the getting-started guide</h3><p>How to open Tinkerie on your computer, how teaching mode works, and an easy first thing to try in every app.</p></div>
  <span class="more">Get started {svg("arrow")}</span>
</a>"""
    faq = "".join(f'<details><summary>{e(q)}</summary><div>{"".join(f"<p>{e(p)}</p>" for p in ps)}</div></details>' for q, ps in FAQ)
    body = f"""<div class="ruler" aria-hidden="true"></div>
<section class="hero"><div class="wrap hero-grid">
  <div>
    <span class="tape">Free · Alpha · v{VERSION.split("-")[0]}</span>
    <h1>Every tool on your workbench, in one free program.</h1>
    <p class="lead">Design circuit boards and 3D prints, flash microcontrollers, write SD cards, and turn your PC into an oscilloscope, multimeter and serial monitor.</p>
    <div class="cta">
      <a class="btn btn-go" id="heroDownload" href="#download">{svg("download")}<span>Download Tinkerie</span></a>
      <a class="btn btn-line" href="start.html">New here? Start here</a>
    </div>
    <div class="facts">
      <span>{svg("free")}Free, no account</span>
      <span>{svg("box")}Nothing to install</span>
      <span>{svg("systems")}Windows, Mac, Linux, Raspberry Pi</span>
    </div>
  </div>
  <figure><div class="shot"><span class="tag">The launcher</span><button class="zoom" type="button" data-full="img/shots/launcher.webp" data-caption="The Tinkerie launcher: pick an app to start it." aria-label="Enlarge the launcher screenshot"><img src="img/shots/launcher.webp" alt="The Tinkerie launcher listing all eight apps" width="1470" height="960"></button></div>
  <figcaption>Open Tinkerie, pick an app. Each one runs in its own window.</figcaption></figure>
</div></section>

<section class="band" style="padding-block:0 40px"><div class="wrap">
  <div class="callout alpha">{svg("info")}<div class="stack" style="gap:6px"><p><b>Tinkerie is in alpha.</b> Every app works and you can use them today, but they're still being tested on real hardware, so expect a few rough edges and changes between versions.</p><p class="muted">Found something that doesn't work? <a href="{REPO}/issues">Tell us on GitHub</a>. It really helps.</p></div></div>
</div></section>

<section class="band alt" id="apps"><div class="wrap">
  <div class="head"><div><span class="label">Eight apps</span><h2>What's in the box</h2><p>Each app has its own page with screenshots, a getting-started guide and everything it can do.</p></div></div>
  <div class="apps">{cards}</div>
</div></section>

<section class="band"><div class="wrap">
  <div class="head"><div><span class="label">Made for beginners</span><h2>You don't need to know it all already</h2></div></div>
  <div class="pillars">
    <div class="pillar"><div class="ic">{svg("teach")}</div><h3>Every button explains itself</h3><p>Teaching mode is on in every app. Point at anything and a card says what it does, in plain language. Press <kbd>F2</kbd> when you don't need it any more.</p></div>
    <div class="pillar"><div class="ic">{svg("demo")}</div><h3>Try it without hardware</h3><p>Scopie, Meterie, Tracie and Flashie have demo modes with pretend scopes, meters and boards, so you can learn before you buy anything.</p></div>
    <div class="pillar"><div class="ic">{svg("box")}</div><h3>Nothing to set up</h3><p>One program, no installer, no Python, no extra drivers for most things. Download it, open it, and pick an app.</p></div>
  </div>
</div></section>

<section class="band alt"><div class="wrap">
  <div class="head"><div><span class="label">How they fit together</span><h2>From idea to working gadget</h2><p>The apps share their work, so a project can flow from one to the next.</p></div></div>
  <div class="flow">
    <div><span class="step">DESIGN</span><h3>Draw the circuit</h3><p><a href="wirie.html">Wirie</a> lays out the board, or plans it on stripboard.</p></div>
    <div><span class="step">MAKE</span><h3>Print a case</h3><p><a href="makie.html">Makie</a> builds a case around your Wirie board. <a href="tunie.html">Tunie</a> gets your printer dialled in.</p></div>
    <div><span class="step">PROGRAM</span><h3>Flash the firmware</h3><p><a href="flashie.html">Flashie</a> puts firmware on the board, or builds a program from blocks.</p></div>
    <div><span class="step">TEST</span><h3>See what it's doing</h3><p><a href="tracie.html">Tracie</a>, <a href="scopie.html">Scopie</a> and <a href="meterie.html">Meterie</a> show its messages, signals and voltages.</p></div>
  </div>
</div></section>

<section class="band" id="download"><div class="wrap">
  <div class="head"><div><span class="label">Download · alpha version {VERSION}</span><h2>Get Tinkerie</h2><p>The one marked "Your computer" should suit you. Each download has a READ ME FIRST file that explains how to start it.</p></div>
  <a class="btn btn-line" href="{REPO}/releases">All files and release notes {svg("arrow")}</a></div>
  <div class="dl">{dl_rows()}</div>
  <p class="osnote">Windows: see the <a href="code-signing.html">Code signing policy</a>. Free code signing provided by <a href="https://signpath.io">SignPath.io</a>, certificate by <a href="https://signpath.org">SignPath Foundation</a>.</p>
</div></section>

<section class="band alt" id="faq"><div class="wrap">
  <div class="head"><div><span class="label">Questions</span><h2>Good to know</h2></div></div>
  <div class="faq">{faq}</div>
</div></section>"""
    page("index.html", "Tinkerie", "Free maker tools in one program: circuit boards, 3D parts, firmware, SD cards, oscilloscope, multimeter and serial monitor. By nrmluvdev.", "home", body)


# ------------------------------------------------------- getting started

def build_start():
    os_steps = {
      "windows": ("Windows", [
        ("Download Tinkerie.exe", f'From the <a href="index.html#download">download list</a>. If your network blocks .exe files, take the Windows zip and unzip it.'),
        ("Double-click it", "Windows may say it's an unrecognised app, because it isn't signed with a paid certificate. Click <b>More info</b>, then <b>Run anyway</b>."),
        ("Pick an app", "The launcher opens. Click an app, or press its number key (1–7)."),
        ("Optional: pin it", "Right-click Tinkerie.exe and choose Pin to Start or Pin to taskbar."),
      ]),
      "mac": ("Mac", [
        ("Download the right zip", "Apple silicon (M1 or newer) or Intel. Check under  › About This Mac if you're not sure."),
        ("Move it to Applications", "Unzip it and drag <b>Tinkerie.app</b> into your Applications folder."),
        ("Allow it once", "The first time, macOS blocks it. Open it once, then go to <b>System Settings › Privacy &amp; Security</b> and click <b>Open Anyway</b>."),
        ("Pick an app", "The launcher opens in Chrome, Edge or Brave if you have one, otherwise in your normal browser."),
      ]),
      "linux": ("Linux", [
        ("Download and unpack", "Get the Linux .tar.gz and unpack it."),
        ("Start it", "<pre>cd Tinkerie-*-linux-x86_64\n./tinkerie</pre>"),
        ("Add it to your menu", "<pre>./install.sh</pre>Remove it again with <code>./install.sh --remove</code>."),
        ("For USB devices", "For serial boards, join the dialout group: <pre>sudo usermod -aG dialout $USER</pre>For USB scopes and logic analysers, run <code>./install-usb.sh</code> once. Then log out and back in."),
      ]),
      "pi": ("Raspberry Pi", [
        ("Pick 64-bit or 32-bit", "Most Raspberry Pi OS installs are 64-bit now. Type <code>uname -m</code> in a terminal: <code>aarch64</code> means 64-bit."),
        ("Download and unpack", "<pre>tar xzf Tinkerie-*-pi-64bit.tar.gz\ncd Tinkerie-*-pi-64bit</pre>"),
        ("Start it and add it to the menu", "<pre>./tinkerie\n./install.sh</pre>"),
        ("For USB devices", "<pre>sudo usermod -aG dialout $USER\n./install-usb.sh</pre>Then log out and back in."),
      ]),
    }
    tabs = "".join(f'<button type="button" role="tab" data-os="{k}" id="tab-{k}" aria-controls="os-{k}" aria-selected="{str(i == 0).lower()}">{v[0]}</button>' for i, (k, v) in enumerate(os_steps.items()))
    panels = "".join(
        f'<div id="os-{k}" role="tabpanel" aria-labelledby="tab-{k}"{"" if i == 0 else " hidden"}><ol class="steps">'
        + "".join(f"<li><div><h3>{e(t)}</h3><p>{d}</p></div></li>" for t, d in steps) + "</ol></div>"
        for i, (k, (_, steps)) in enumerate(os_steps.items()))
    tries = "".join(f"""<a class="pair" href="{a['id']}.html" style="--app:{a['color']}"><img src="img/icons/{a['icon']}" alt=""><div><b>{a['name']}</b><p>{e(a['firsts'][0][1])}</p></div></a>""" for a in APPS)
    body = f"""<div class="ruler" aria-hidden="true"></div>
<section class="hero"><div class="wrap">
  <span class="tape">Getting started</span>
  <h1>Up and running in five minutes</h1>
  <p class="lead" style="max-width:52ch">Download, open, pick an app. Here's exactly what to do on your computer, and a good first thing to try in each app.</p>
</div></section>
<section class="band alt" style="padding-top:40px"><div class="wrap">
  <div class="head"><div><span class="label">Step by step</span><h2>Install on your computer</h2></div></div>
  <div class="tabs" role="tablist" aria-label="Your computer">{tabs}</div>
  {panels}
</div></section>
<section class="band"><div class="wrap">
  <div class="head"><div><span class="label">Learning as you go</span><h2>Teaching mode</h2></div></div>
  <div class="hero-grid" style="align-items:start">
    <div class="stack prose">
      <p>Every app starts with teaching mode switched on. Rest the mouse on any button, setting or panel and a card appears explaining what it does and when you'd use it.</p>
      <p>Once you know your way around, press <kbd>F2</kbd> to turn the cards off. Press it again any time to bring them back. Each app remembers your choice.</p>
      <p>Scopie keeps its explanations in a help bar under the scope screen, and has a beginner's guide on <kbd>F1</kbd>.</p>
    </div>
    {shot("flashie-settings", "Teaching mode in Flashie: pointing at a setting explains it.", sm=True)}
  </div>
</div></section>
<section class="band alt"><div class="wrap">
  <div class="head"><div><span class="label">First things to try</span><h2>One easy win in each app</h2><p>None of these needs any hardware.</p></div></div>
  <div class="pairs">{tries}</div>
</div></section>
<section class="band"><div class="wrap">
  <div class="head"><div><span class="label">Handy to know</span><h2>Tips</h2></div></div>
  <div class="features">
    <div class="feature"><h3>Start an app directly</h3><p>Make a desktop shortcut to <code>Tinkerie.exe --app makie</code>, or rename a copy of Tinkerie.exe to <code>Makie.exe</code>. On Linux and Mac, a link called <code>makie</code> does the same.</p></div>
    <div class="feature"><h3>Run several at once</h3><p>Each app opens in its own window and keeps running after you close the launcher, so you can have Flashie and Tracie open side by side.</p></div>
    <div class="feature"><h3>Your work is saved</h3><p>Wirie and Makie save as you go and pick up where you left off next time.</p></div>
  </div>
</div></section>"""
    page("start.html", "Getting Started with Tinkerie", "How to download, open and start using Tinkerie on Windows, Mac, Linux and Raspberry Pi.", "start", body)


# ---------------------------------------------------------------- apps

def build_app(i, a):
    prev_a, next_a = APPS[i - 1], APPS[(i + 1) % len(APPS)]
    chips = "".join(f'<span class="chip">{e(c)}</span>' for c in a["chips"])
    firsts = "".join(f"<li><h3>{e(t)}</h3><p>{e(d)}</p></li>" for t, d in a["firsts"])
    main_shot = shot(a["shots"][0][0], a["shots"][0][1], tag=a["name"], eager=True)
    gallery = ""
    if len(a["shots"]) > 1:
        gallery = '<div class="thumbs">' + "".join(shot(n, c, sm=True) for n, c in a["shots"][1:]) + "</div>"
    feats = "".join(f'<div class="feature"><h3>{e(t)}</h3><p>{e(d)}</p><ul>{"".join(f"<li>{e(x)}</li>" for x in items)}</ul></div>' for t, d, items in a["features"])
    table = ""
    if a["table"]:
        title, cols, rows = a["table"]
        table = f"""<section class="band alt"><div class="wrap">
  <div class="head"><div><span class="label">Compatibility</span><h2>{e(title)}</h2></div></div>
  <div class="table-wrap"><table><thead><tr>{"".join(f"<th>{e(c)}</th>" for c in cols)}</tr></thead><tbody>{"".join("<tr>" + "".join(f"<td>{e(x)}</td>" for x in r) + "</tr>" for r in rows)}</tbody></table></div>
</div></section>"""
    calls = "".join(f'<div class="callout{" warn" if k == "warn" else ""}">{svg("warn" if k == "warn" else "info")}<p>{e(t)}</p></div>' for k, t in a["callouts"])
    pairs = "".join(f"""<a class="pair" href="{p}.html"><img src="img/icons/{APP[p]['icon']}" alt=""><div><b>{APP[p]['name']}</b><p>{e(why)}</p></div></a>""" for p, why in a["pairs"])
    alt_after = "" if a["table"] else " alt"
    body = f"""<div class="ruler" aria-hidden="true"></div>
<section class="app-hero"><div class="wrap">
  <nav class="crumbs" aria-label="You are here"><a href="index.html">Tinkerie</a><span aria-hidden="true">›</span><a href="index.html#apps">Apps</a><span aria-hidden="true">›</span><span>{a['name']}</span></nav>
  <div class="grid">
    <div>
      <div class="app-title"><img src="img/icons/{a['icon']}" alt=""><div><h1>{a['name']}</h1><span class="ver">Version {a['ver']} · in Tinkerie {VERSION}</span><span class="alpha-pill">Alpha</span></div></div>
      <p class="lead">{e(a['tagline'])}</p>
      <div class="plain"><span class="label">{e(a['plain'][0])}</span><p>{e(a['plain'][1])}</p></div>
      <div class="chips">{chips}</div>
      <div class="hero" style="padding:0"><div class="cta"><a class="btn btn-go" href="index.html#download">{svg("download")}Download Tinkerie</a><a class="btn btn-line" href="#first">First five minutes</a></div><p class="osnote">You'll need: {e(a['need'])}.</p></div>
    </div>
    {main_shot}
  </div>
</div></section>

<section class="band alt" id="first"><div class="wrap">
  <div class="head"><div><span class="label">Your first five minutes</span><h2>Try this first</h2></div></div>
  <ol class="firsts">{firsts}</ol>
</div></section>

{f'<section class="band"><div class="wrap"><div class="head"><div><span class="label">Screenshots</span><h2>A closer look</h2><p>Click any picture to see it bigger.</p></div></div>{gallery}</div></section>' if gallery else ''}

<section class="band{'' if gallery else ''}"><div class="wrap">
  <div class="head"><div><span class="label">Everything it does</span><h2>Features</h2></div></div>
  <div class="features">{feats}</div>
</div></section>

{table}

<section class="band{alt_after}"><div class="wrap stack">
  {f'<div class="head" style="margin:0"><div><span class="label">Before you start</span><h2>Good to know</h2></div></div><div class="stack">{calls}</div>' if calls else ''}
  <div class="head" style="margin:{'30px' if calls else '0'} 0 0"><div><span class="label">Works well with</span><h2>Pairs nicely with</h2></div></div>
  <div class="pairs">{pairs}</div>
  <nav class="pager" aria-label="Other apps" style="margin-top:30px">
    <a href="{prev_a['id']}.html"><span class="muted">← Previous app</span><b>{prev_a['name']}</b></a>
    <a href="{next_a['id']}.html"><span class="muted">Next app →</span><b>{next_a['name']}</b></a>
  </nav>
</div></section>"""
    page(f"{a['id']}.html", f"{a['name']} · Tinkerie", f"{a['name']}: {a['short']} Part of Tinkerie, free maker tools by nrmluvdev.", a["id"], body, app=a)


PRIVACY = [('Flashie', "downloads firmware, the Arduino board list and Raspberry Pi's OS list from those projects' own servers (GitHub, Raspberry Pi, MicroPython, CircuitPython and the others shown in its library) when you open its library or choose something to download."), ('Tunie', 'talks to your own printer (OctoPrint or Moonraker) only at the address you type in.'), ('Scopie', 'connects to a network oscilloscope only at the address you choose.'), ('Everything else', "Wirie, Makie, Meterie, Tracie, Codie and the launcher never use the internet. Each app's page is served only to your own computer (localhost).")]

POLICY = '<div class="ruler" aria-hidden="true"></div>\n<section class="hero" style="padding-bottom:30px"><div class="wrap">\n  <span class="tape">Code signing policy</span>\n  <h1>Code signing policy</h1>\n  <p class="lead" style="max-width:56ch">How Tinkerie\'s Windows program is signed, who decides what gets signed, and what Tinkerie does (and doesn\'t do) with your data.</p>\n</div></section>\n<section class="band alt" style="padding-top:40px"><div class="wrap"><div class="stack" style="max-width:860px">\n  <div class="callout">{info}<p>Free code signing provided by <a href="https://signpath.io">SignPath.io</a>, certificate by <a href="https://signpath.org">SignPath Foundation</a>.</p></div>\n  <p class="muted">Signing is being set up. Releases made before it\'s in place are not signed, and Windows shows an "unrecognised app" warning for them.</p>\n</div></div></section>\n<section class="band"><div class="wrap"><div class="stack prose">\n  <h2 id="what-is-signed">What is signed</h2>\n  <p>The Windows program, <code>Tinkerie.exe</code>, attached to each <a href="{repo}/releases">GitHub release</a>, on its own and inside the Windows zip. It is built from this project\'s source code by GitHub Actions, on GitHub\'s own servers, using the public <a href="{repo}/blob/main/.github/workflows/release.yml">release workflow</a>. Nothing built anywhere else is signed.</p>\n  <p>Signing proves the file came from this project\'s source code and hasn\'t been changed since. The Mac and Linux downloads are not part of this signing.</p>\n  <h2 id="team">Team roles</h2>\n  <div class="table-wrap"><table><thead><tr><th>Role</th><th>Who</th><th>What it means</th></tr></thead><tbody>\n    <tr><td>Committers and reviewers</td><td><a href="https://github.com/normanlove">nrmluvdev (normanlove)</a></td><td>Can change the source code. Every change from anyone else, such as a pull request, is reviewed by them before it\'s added.</td></tr>\n    <tr><td>Approvers</td><td><a href="https://github.com/normanlove">nrmluvdev (normanlove)</a></td><td>Approves each release by hand before it\'s signed.</td></tr>\n  </tbody></table></div>\n  <p>Everyone in these roles uses two-factor authentication on GitHub and SignPath.</p>\n  <h2 id="privacy">Privacy</h2>\n  <p><b>This program will not transfer any information to other networked systems unless specifically requested by the user or the person installing or operating it.</b> Tinkerie has no analytics, no tracking, no accounts and no automatic update checks.</p>\n  <p>These are the only times it uses the network, each one because you asked:</p>\n  <ul>{priv}</ul>\n  <p>Those sites have their own privacy policies. This website is hosted by GitHub Pages and loads its fonts from Google Fonts.</p>\n  <h2>Changes to your computer</h2>\n  <p>Tinkerie doesn\'t need installing: delete it to remove it. Two optional steps change your system, and both ask first: on Windows, Scopie\'s <i>Install USB driver</i> button runs Zadig to install a USB driver (you\'ll see the administrator prompt; Tracie\'s help explains the same step for logic analysers), and on Linux and Raspberry Pi, <code>install.sh</code> and <code>install-usb.sh</code> add a menu entry and USB permissions. Both scripts undo themselves with <code>--remove</code>. The Windows driver can be removed in Device Manager.</p>\n  <h2>Licence</h2>\n  <p>Tinkerie is free software under the GNU General Public License, version 3 or later. The source code is on <a href="{repo}">GitHub</a>.</p>\n</div></div></section>'


def build_policy():
    priv = "".join(f"<li><b>{e(a)}</b> {e(t)}</li>" for a, t in PRIVACY)
    body = POLICY.format(info=svg("info"), repo=REPO, priv=priv)
    page("code-signing.html", "Code Signing Policy · Tinkerie", "How Tinkerie's Windows program is signed through SignPath, who approves releases, and Tinkerie's privacy statement.", "policy", body)


def main():
    os.makedirs(OUT, exist_ok=True)
    shutil.copy(os.path.join(HERE, "site.css"), OUT)
    shutil.copy(os.path.join(HERE, "site.js"), OUT)
    open(os.path.join(OUT, ".nojekyll"), "w").close()
    build_index()
    build_start()
    build_policy()
    for i, a in enumerate(APPS):
        build_app(i, a)
    print("built", len(APPS) + 2, "pages into", OUT, "for version", VERSION)


if __name__ == "__main__":
    main()
