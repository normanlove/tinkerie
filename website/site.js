// Tinkerie website: menus, theme, download suggestions, screenshot zoom.
// Made by nrmluvdev.
(() => {
  const $ = (s, r = document) => r.querySelector(s);
  const $$ = (s, r = document) => [...r.querySelectorAll(s)];
  const store = {
    get(k) { try { return localStorage.getItem(k); } catch { return null; } },
    set(k, v) { try { localStorage.setItem(k, v); } catch {} },
  };

  // Theme: follows the system unless the visitor picks one.
  const root = document.documentElement;
  const saved = store.get('tinkerie.theme');
  if (saved === 'light' || saved === 'dark') root.dataset.theme = saved;
  const themeBtn = $('#theme');
  const isDark = () => root.dataset.theme ? root.dataset.theme === 'dark' : matchMedia('(prefers-color-scheme: dark)').matches;
  const showTheme = () => { if (themeBtn) themeBtn.setAttribute('aria-label', isDark() ? 'Use the light theme' : 'Use the dark theme'); };
  if (themeBtn) {
    showTheme();
    themeBtn.onclick = () => { const t = isDark() ? 'light' : 'dark'; root.dataset.theme = t; store.set('tinkerie.theme', t); showTheme(); };
  }

  // Apps menu.
  const menu = $('.apps-menu');
  if (menu) {
    const b = $('button', menu);
    b.onclick = e => { e.stopPropagation(); const o = menu.classList.toggle('open'); b.setAttribute('aria-expanded', o); };
    document.addEventListener('click', e => { if (!menu.contains(e.target)) { menu.classList.remove('open'); b.setAttribute('aria-expanded', 'false'); } });
    document.addEventListener('keydown', e => { if (e.key === 'Escape') { menu.classList.remove('open'); b.setAttribute('aria-expanded', 'false'); } });
  }

  // Ruler numbers (every 80 px is "10 mm").
  $$('.ruler').forEach(r => {
    const n = Math.ceil(Math.max(r.clientWidth, 1600) / 80);
    for (let i = 1; i < n; i++) { const s = document.createElement('span'); s.style.left = (i * 80) + 'px'; s.textContent = i * 10; r.appendChild(s); }
  });

  // Which download suits this computer?
  const ua = navigator.userAgent, plat = (navigator.userAgentData && navigator.userAgentData.platform) || navigator.platform || '';
  let os = '';
  if (/Win/i.test(plat) || /Windows/i.test(ua)) os = 'windows';
  else if (/Android/i.test(ua)) os = 'android';
  else if (/Mac/i.test(plat) || /Mac OS X/.test(ua)) os = /iPhone|iPad/.test(ua) ? '' : 'mac';
  else if (/aarch64|arm64/i.test(plat + ua) && /Linux/i.test(plat + ua) && !/Android/i.test(ua)) os = 'pi';
  else if (/armv7|armv6/i.test(plat + ua) && !/Android/i.test(ua)) os = 'pi32';
  else if (/Linux/i.test(plat) && !/Android/i.test(ua)) os = 'linux';
  if (os) {
    $$(`.dl a.row[data-os="${os}"]`).forEach(r => { r.classList.add('mine'); const t = $('.os', r); if (t && !$('.you', t)) t.insertAdjacentHTML('beforeend', '<span class="you">Your computer</span>'); });
    const hb = $('#heroDownload'), mine = $(`.dl a.row[data-os="${os}"]`);
    if (hb && mine) { hb.href = mine.href; const l = $('span', hb); if (l) l.textContent = 'Download for ' + mine.dataset.name; }
  }

  // Getting-started tabs.
  $$('.tabs').forEach(t => {
    const btns = $$('button', t);
    btns.forEach(b => b.onclick = () => {
      btns.forEach(x => { x.setAttribute('aria-selected', x === b); $('#' + x.getAttribute('aria-controls')).hidden = x !== b; });
    });
  });
  if (os) { const pick = $(`.tabs button[data-os="${os === 'pi32' ? 'pi' : os}"]`); if (pick) pick.click(); }

  // Screenshot zoom.
  $$('button.zoom').forEach(b => b.onclick = () => {
    const img = $('img', b), cap = b.dataset.caption || img.alt;
    const box = document.createElement('div');
    box.className = 'lightbox'; box.setAttribute('role', 'dialog'); box.setAttribute('aria-modal', 'true'); box.setAttribute('aria-label', cap);
    box.innerHTML = `<button type="button">Close</button><figure><img alt=""><p></p></figure>`;
    $('img', box).src = b.dataset.full || img.src; $('img', box).alt = img.alt; $('p', box).textContent = cap;
    const close = () => { box.remove(); document.removeEventListener('keydown', esc); b.focus(); };
    const esc = e => { if (e.key === 'Escape') close(); };
    box.onclick = e => { if (e.target === box || e.target.tagName === 'BUTTON') close(); };
    document.addEventListener('keydown', esc);
    document.body.appendChild(box);
    $('button', box).focus();
  });
})();
