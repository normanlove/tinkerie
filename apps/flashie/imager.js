// Flashie — "Write image" tab: put an OS image on an SD card or USB drive,
// like Raspberry Pi Imager. Made by nrmluvdev.
(() => {
const $i = s => document.querySelector(s);
const escI = s => String(s ?? '').replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
const hb = n => { if(!n) return ''; const u=['bytes','KB','MB','GB','TB']; let i=0, f=n; while(f>=1000 && i<4){ f/=1000; i++; } return i ? f.toFixed(f<10?1:0)+' '+u[i] : n+' bytes'; };

const I = { custom:{on:false}, secret:{password:'', wifiPass:''}, os:null, file:null, disk:null, devices:[], device:'', stack:[], busy:false, disks:[], phase:'', started:0, lastDone:0, lastT:0, speed:0 };

const css = `
#imgView{display:flex;flex-direction:column;gap:20px}
.imgrow{display:grid;grid-template-columns:repeat(3,1fr);gap:16px}
@media (max-width:860px){.imgrow{grid-template-columns:1fr}}
.imgpick{width:100%;text-align:left;justify-content:flex-start;padding:14px;min-height:64px;flex-direction:column;align-items:flex-start;gap:2px}
.imgpick b{font:650 15px/1.25 var(--display)}
.imgpick span{color:var(--muted);font-weight:500;font-size:12.5px}
.imgpick.set{border-color:rgba(124,92,255,.5)}
.imgopts{display:flex;flex-direction:column;gap:8px;font-size:13px;color:var(--muted)}
.imgopts label{display:flex;gap:8px;align-items:center;cursor:pointer}
.imgwrite{background:var(--grad);border-color:transparent;color:#fff;padding:13px;font-size:15px}
.imgwrite:hover{filter:brightness(1.1);background:var(--grad)}
.imgmodal{position:fixed;inset:0;background:rgba(5,7,10,.66);backdrop-filter:blur(6px);display:grid;place-items:center;z-index:50}
.imgmodal.hidden{display:none}
.imgbox{width:min(720px,94vw);max-height:84vh;display:flex;flex-direction:column;background:var(--card);border:1px solid var(--line2);border-radius:var(--r);overflow:hidden;box-shadow:0 30px 80px -20px #000}
.imgbox-h{display:flex;align-items:center;gap:10px;padding:16px 18px;border-bottom:1px solid var(--line)}
.imgbox-h h2{margin:0;font:650 17px/1.2 var(--display);flex:1}
.imgbox-b{overflow:auto;padding:8px}
.imgbox-f{padding:12px 18px;border-top:1px solid var(--line);display:flex;gap:10px;align-items:center;font-size:12.5px;color:var(--muted)}
.imgitem{display:flex;gap:12px;align-items:center;padding:10px 12px;border-radius:12px;cursor:pointer;border:1px solid transparent}
.imgitem:hover{background:var(--card2);border-color:var(--line2)}
.imgitem img{width:36px;height:36px;object-fit:contain;flex:none}
.imgitem .ph{width:36px;height:36px;border-radius:9px;background:var(--card2);display:grid;place-items:center;flex:none;font-size:18px}
.imgitem .t{flex:1;min-width:0}
.imgitem .t b{display:block;font-weight:650}
.imgitem .t span{display:block;color:var(--muted);font-size:12.5px}
.imgitem .m{color:var(--dim);font-size:12px;white-space:nowrap;text-align:right}
.imgitem.off{opacity:.45;cursor:not-allowed}
.imgempty{padding:30px;text-align:center;color:var(--muted)}
.imgwarn{background:rgba(255,92,122,.08);border:1px solid rgba(255,92,122,.35);border-radius:12px;padding:14px;color:var(--text)}
.imgwarn b{color:var(--bad)}
.imgsel{background:var(--bg2);border:1px solid var(--line2);border-radius:10px;padding:6px 10px}
.imgnote{font-size:12.5px;color:var(--text);background:rgba(255,190,92,.08);border:1px solid rgba(255,190,92,.35);border-radius:10px;padding:8px 10px}
.imgcus{justify-content:flex-start;gap:8px}
.imgcus b{font-weight:650}
.imgform{display:flex;flex-direction:column;gap:14px;padding:10px}
.imgform fieldset{border:1px solid var(--line2);border-radius:12px;padding:12px 14px;margin:0;display:flex;flex-direction:column;gap:10px}
.imgform legend{padding:0 6px;font-weight:650}
.imgform .r{display:grid;grid-template-columns:150px 1fr;gap:10px;align-items:center;font-size:13px;color:var(--muted)}
@media (max-width:560px){.imgform .r{grid-template-columns:1fr}}
.imgform .chk{display:flex;gap:8px;align-items:center;font-size:13px;cursor:pointer}
.imgform textarea{min-height:64px;resize:vertical;font:12px/1.4 var(--mono,monospace)}
.imgform .off{opacity:.45;pointer-events:none}
.imgform .hint{font-size:12px;color:var(--dim)}
`;

function build(){
  const st = document.createElement('style'); st.textContent = css; document.head.appendChild(st);
  const v = document.createElement('section');
  v.id = 'imgView'; v.className = 'hidden';
  v.innerHTML = `
  <div class="imgrow">
    <div class="card"><div class="card-h"><div class="n">1</div><h2>Operating system</h2></div>
      <button class="btn imgpick" id="imgOS" data-help="Choose OS"><b>Choose OS</b><span>Raspberry Pi OS, Linux, a Windows installer, or your own image file</span></button>
      <div class="imgopts"><label data-help="Device filter">For <select class="imgsel" id="imgDev"><option value="">any device</option></select></label></div>
      <div class="imgnote hidden" id="imgNote" data-help="Image note"></div>
    </div>
    <div class="card"><div class="card-h"><div class="n">2</div><h2>Storage</h2></div>
      <button class="btn imgpick" id="imgDisk" data-help="Choose storage"><b>Choose storage</b><span>The SD card or USB drive to write to</span></button>
      <div class="imgopts"><span>Only removable drives are listed — never your computer's own disk.</span></div>
    </div>
    <div class="card"><div class="card-h"><div class="n">3</div><h2>Write</h2></div>
      <button class="btn imgwrite" id="imgGo" data-help="Write button" disabled>Write</button>
      <div class="imgopts">
        <label data-help="Verify after writing"><input type="checkbox" id="imgVerify" checked> Check the drive after writing</label>
        <label data-help="Eject when done"><input type="checkbox" id="imgEject" checked> Eject when done</label>
        <button class="btn ghost imgcus" id="imgCustom" data-help="OS settings">⚙ <span>OS settings: <b id="imgCustomState">off</b></span></button>
      </div>
    </div>
  </div>
  <section class="card" data-help="Image progress">
    <div class="row" style="align-items:flex-end;display:flex;gap:10px">
      <div class="pct" id="imgPct">0%</div><div class="grow" style="flex:1"></div>
      <button class="btn ghost hidden" id="imgCancel" data-help="Cancel image write">Cancel</button>
    </div>
    <div class="bar idle" id="imgBar"><i></i></div>
    <div class="pstats"><span id="imgText">Choose an OS and a drive, then press <b>Write</b>.</span><span id="imgSpeed"></span></div>
  </section>
  <div class="imgmodal hidden" id="imgModal"><div class="imgbox">
    <div class="imgbox-h"><button class="btn ghost icon hidden" id="imgBack" title="Back">←</button><h2 id="imgMTitle"></h2><button class="btn ghost icon" id="imgClose" title="Close (Esc)">✕</button></div>
    <div class="imgbox-b" id="imgMBody"></div>
    <div class="imgbox-f" id="imgMFoot"></div>
  </div></div>
  <input type="file" id="imgFileIn" hidden>`;
  document.querySelector('.app').insertBefore(v, document.querySelector('footer'));

  $i('#imgOS').onclick = openOS;
  $i('#imgDisk').onclick = openDisks;
  $i('#imgCustom').onclick = openCustom;
  try{ Object.assign(I.custom, JSON.parse(localStorage.getItem('flashie.imgcustom') || '{}')); }catch{}
  showCustomState();
  $i('#imgGo').onclick = confirmWrite;
  $i('#imgCancel').onclick = () => fetch('/api/img/cancel', {method:'POST'});
  $i('#imgClose').onclick = closeModal;
  $i('#imgModal').onclick = e => { if(e.target.id === 'imgModal') closeModal(); };
  $i('#imgBack').onclick = () => { I.stack.pop(); renderOS(); };
  $i('#imgDev').onchange = e => { I.device = e.target.value; try{ localStorage.setItem('flashie.imgdev', I.device); }catch{} if(I.os && !fits(I.os)) setOS(null); if(!$i('#imgModal').classList.contains('hidden')) renderOS(); };
  $i('#imgFileIn').onchange = e => { const f = e.target.files[0]; e.target.value=''; if(f) upload(f); };
  document.addEventListener('keydown', e => { if(e.key === 'Escape' && !$i('#imgModal').classList.contains('hidden')) closeModal(); });
  try{ I.device = localStorage.getItem('flashie.imgdev') || ''; }catch{}

  const es = new EventSource('/api/events');
  es.onmessage = ev => { const m = JSON.parse(ev.data); if(m.type === 'img') onProgress(m); };
}

window.imgView = on => {
  $i('#imgView').classList.toggle('hidden', !on);
  if(on && !I.devices.length) loadCatalogue('', true);
};

/* ---------- modal ---------- */
function modal(title, body, foot, back){
  $i('#imgMTitle').textContent = title;
  $i('#imgMBody').innerHTML = body;
  $i('#imgMFoot').innerHTML = foot || '';
  $i('#imgBack').classList.toggle('hidden', !back);
  $i('#imgModal').classList.remove('hidden');
}
function closeModal(){ $i('#imgModal').classList.add('hidden'); clearInterval(I.diskTimer); }

/* ---------- OS catalogue ---------- */
async function loadCatalogue(url, quiet){
  const r = await fetch('/api/img/oslist' + (url ? '?url=' + encodeURIComponent(url) : '')).then(r => r.json()).catch(e => ({error: String(e)}));
  if(r.error){ if(!quiet) toastI(r.error, true); return null; }
  if(!url){
    I.devices = r.devices || [];
    const sel = $i('#imgDev');
    sel.innerHTML = '<option value="">any device</option>' + I.devices.map((d, i) => `<option value="${i}">${escI(d.name)}</option>`).join('');
    if(I.device && I.devices[I.device]) sel.value = I.device; else I.device = '';
    I.top = r.items; I.cached = r.cached;
  }
  return r.items;
}

function fits(it){
  if(I.device === '' || !I.devices[I.device]) return true;
  const tags = I.devices[I.device].tags || [];
  if(!tags.length) return true;
  const own = it.devices || [];
  if(own.length && !own.some(t => tags.includes(t))) return false;
  if(it.subitems && !it.url) return it.subitems.some(fits);
  return true;
}

async function openOS(){
  if(I.busy) return;
  I.stack = [];
  modal('Choose an operating system', '<div class="imgempty">Loading the OS list…</div>');
  if(!I.top) await loadCatalogue('');
  renderOS();
}

function renderOS(){
  const level = I.stack[I.stack.length - 1];
  const items = level ? level.items : (I.top || []);
  const title = level ? level.name : 'Choose an operating system';
  let rows = items.filter(fits).map((it, i) => {
    const sz = it.extract_size ? hb(it.extract_size) : '';
    const meta = [it.release_date, sz].filter(Boolean).join('<br>');
    const ic = it.icon ? `<img src="${escI(it.icon)}" alt="" onerror="this.outerHTML='<div class=ph>💿</div>'">` : '<div class="ph">💿</div>';
    return `<div class="imgitem" data-i="${items.indexOf(it)}">${ic}<div class="t"><b>${escI(it.name)}</b><span>${escI(it.description || '')}</span></div><div class="m">${it.url ? meta : '›'}</div></div>`;
  }).join('');
  if(!level){
    rows += `<div class="imgitem" data-own="1" data-help="Use custom image"><div class="ph">📁</div><div class="t"><b>Use my own image file</b><span>An .img or .iso (Linux, or a Windows 10/11 installer), or one packed as .zip, .xz, .zst, .gz or .bz2</span></div><div class="m">›</div></div>`;
  }
  if(!rows) rows = '<div class="imgempty">Nothing here suits the device you picked.</div>';
  const foot = I.top ? `<span>${I.cached ? 'Offline — showing the last OS list Flashie saw.' : 'From Raspberry Pi’s official OS list, the same one Raspberry Pi Imager uses.'}</span><div style="flex:1"></div><button class="btn ghost" id="imgClear" data-help="Clear image cache">Clear image cache</button>` : '';
  modal(title, rows, foot, !!level);
  $i('#imgMBody').onclick = async e => {
    const row = e.target.closest('.imgitem'); if(!row) return;
    if(row.dataset.own){ chooseFile(); return; }
    const it = items[+row.dataset.i];
    if(it.url){ setOS(it); closeModal(); return; }
    let sub = it.subitems;
    if((!sub || !sub.length) && it.subitems_url){
      $i('#imgMBody').innerHTML = '<div class="imgempty">Loading…</div>';
      sub = await loadCatalogue(it.subitems_url);
      if(!sub){ renderOS(); return; }
      it.subitems = sub;
    }
    I.stack.push({name: it.name, items: sub || []});
    renderOS();
  };
  const c = $i('#imgClear');
  if(c) c.onclick = async () => {
    const r = await fetch('/api/img/cache', {method:'POST'}).then(r => r.json());
    toastI(r.error ? r.error : 'Image cache cleared', !!r.error);
  };
}

function setOS(it){
  I.os = it; I.file = null;
  const b = $i('#imgOS');
  if(it){
    b.innerHTML = `<b>${escI(it.name)}</b><span>${escI([it.release_date, it.image_download_size ? hb(it.image_download_size) + ' download' : ''].filter(Boolean).join(' · '))}</span>`;
  } else {
    b.innerHTML = '<b>Choose OS</b><span>Raspberry Pi OS and more, or your own image file</span>';
  }
  b.classList.toggle('set', !!it);
  setNote('');
  showCustomState();
  update();
}

function setNote(t){ const n = $i('#imgNote'); n.textContent = t || ''; n.classList.toggle('hidden', !t); }

async function chooseFile(){
  modal('Use my own image file', '<div class="imgempty">Choose the file in the window that opened…</div>');
  const r = await fetch('/api/img/pick', {method:'POST'}).then(r => r.json()).catch(() => ({fallback:true}));
  if(r.fallback){
    modal('Use my own image file', `<div class="imgempty">Choose the image file to copy into Flashie.<br><br><button class="btn" id="imgBrowse">Choose file…</button></div>`);
    $i('#imgBrowse').onclick = () => $i('#imgFileIn').click();
    return;
  }
  if(r.none){ renderOS(); return; }
  if(r.error){ toastI(r.error, true); renderOS(); return; }
  setFile(r); closeModal();
}

async function upload(f){
  modal('Use my own image file', `<div class="imgempty">Copying ${escI(f.name)} into Flashie…</div>`);
  const r = await fetch('/api/img/upload?name=' + encodeURIComponent(f.name), {method:'POST', body:f}).then(r => r.json()).catch(e => ({error:String(e)}));
  if(r.error){ toastI(r.error, true); renderOS(); return; }
  setFile(r); closeModal();
}

function setFile(info){
  I.file = info; I.os = null;
  const kinds = {raw:'image', xz:'xz-packed image', zstd:'zstd-packed image', gzip:'gzip-packed image', bzip2:'bzip2-packed image', zip:'zipped image'};
  const w = info.windows;
  const what = w ? `Windows installer (${escI(w.arch)}) · ${hb(w.total)} · ${escI(w.layout)}`
    : `${escI(kinds[info.kind] || info.kind)}${info.size ? ' · ' + hb(info.size) : ' · ' + hb(info.packed) + ' packed'}`;
  $i('#imgOS').innerHTML = `<b>${escI(info.name)}</b><span>${what}</span>`;
  $i('#imgOS').classList.add('set');
  setNote(w ? 'Flashie will format the drive and copy the installer onto it, for PCs that start in UEFI mode (all Windows 11 PCs).' : info.note);
  showCustomState();
  update();
}

/* ---------- OS customisation ---------- */
function customWhy(){
  if(I.file && I.file.windows) return 'Windows installers don’t take these settings.';
  if(I.os && !I.os.init_format) return 'This OS doesn’t take these settings.';
  return '';
}
function customOn(){ return I.custom.on && !customWhy(); }
function showCustomState(){
  const why = customWhy();
  $i('#imgCustomState').textContent = why ? 'not for this OS' : (I.custom.on ? 'on' : 'off');
  $i('#imgCustom').title = why;
}
function guessLocale(){
  let tz = '', cc = '';
  try{ tz = Intl.DateTimeFormat().resolvedOptions().timeZone || ''; }catch{}
  const m = /-([A-Za-z]{2})\b/.exec(navigator.language || ''); if(m) cc = m[1].toUpperCase();
  return {tz, cc, kb: cc === 'GB' ? 'gb' : cc === 'US' ? 'us' : (cc || '').toLowerCase()};
}
function openCustom(){
  if(I.busy) return;
  const why = customWhy();
  const c = I.custom, g = guessLocale();
  const v = k => escI(c[k] ?? '');
  const ck = k => c[k] ? 'checked' : '';
  modal('OS settings', `<form class="imgform" id="imgForm" autocomplete="off">
    ${why ? `<div class="imgnote">${escI(why)} You can still set them up for next time.</div>` : ''}
    <div class="hint">Set up the system before its first start, like Raspberry Pi Imager does — handy for a Raspberry Pi with no screen or keyboard. These work with Raspberry Pi OS and Ubuntu for Raspberry Pi.</div>
    <fieldset data-help="Hostname setting"><legend><label class="chk"><input type="checkbox" name="useHost" ${ck('useHost')}> Hostname</label></legend>
      <div class="r"><span>Name on the network</span><input class="input" name="hostname" value="${v('hostname')}" placeholder="raspberrypi" maxlength="63"></div>
      <div class="hint">Reach it as <i>name</i>.local from other computers.</div></fieldset>
    <fieldset data-help="User setting"><legend><label class="chk"><input type="checkbox" name="useUser" ${ck('useUser')}> Username and password</label></legend>
      <div class="r"><span>Username</span><input class="input" name="user" value="${v('user')}" placeholder="pi" maxlength="32" autocapitalize="off"></div>
      <div class="r"><span>Password</span><input class="input" type="password" name="password" value="${escI(I.secret.password)}" placeholder="${c.useUser ? 'type it again (not saved)' : ''}"></div></fieldset>
    <fieldset data-help="Wi-Fi setting"><legend><label class="chk"><input type="checkbox" name="useWifi" ${ck('useWifi')}> Wi-Fi</label></legend>
      <div class="r"><span>Network name (SSID)</span><input class="input" name="wifiSSID" value="${v('wifiSSID')}" maxlength="32"></div>
      <div class="r"><span>Password</span><input class="input" type="password" name="wifiPass" value="${escI(I.secret.wifiPass)}" placeholder="leave empty for an open network"></div>
      <div class="r"><span>Country</span><input class="input" name="wifiCountry" value="${escI(c.wifiCountry || g.cc)}" maxlength="2" placeholder="GB" style="max-width:80px;text-transform:uppercase"></div>
      <label class="chk" data-help="Hidden network"><input type="checkbox" name="wifiHidden" ${ck('wifiHidden')}> It’s a hidden network</label></fieldset>
    <fieldset data-help="Locale setting"><legend><label class="chk"><input type="checkbox" name="useLocale" ${ck('useLocale')}> Time zone and keyboard</label></legend>
      <div class="r"><span>Time zone</span><input class="input" name="timezone" value="${escI(c.timezone || g.tz)}" placeholder="Europe/London" list="imgTZ"></div>
      <div class="r"><span>Keyboard layout</span><input class="input" name="keymap" value="${escI(c.keymap || g.kb)}" placeholder="gb" maxlength="32" list="imgKB" style="max-width:120px"></div>
      <datalist id="imgTZ">${(Intl.supportedValuesOf ? Intl.supportedValuesOf('timeZone') : []).map(z => `<option value="${escI(z)}">`).join('')}</datalist>
      <datalist id="imgKB">${['gb','us','de','fr','es','it','ie','pt','br','nl','be','ch','se','no','dk','fi','pl','cz','jp','kr','cn','in','au','ca'].map(k => `<option value="${k}">`).join('')}</datalist></fieldset>
    <fieldset data-help="SSH setting"><legend><label class="chk"><input type="checkbox" name="ssh" ${ck('ssh')}> SSH (log in over the network)</label></legend>
      <label class="chk"><input type="radio" name="sshMode" value="pw" ${c.sshPassword !== false ? 'checked' : ''}> With the password</label>
      <label class="chk"><input type="radio" name="sshMode" value="key" ${c.sshPassword === false ? 'checked' : ''}> Only with these public keys (safer)</label>
      <textarea class="input" name="sshKeys" data-help="SSH keys" placeholder="ssh-ed25519 AAAA… you@computer">${v('sshKeys')}</textarea></fieldset>
  </form>`, `<button class="btn ghost" id="imgCusOff" data-help="Turn settings off">Don’t use settings</button><div style="flex:1"></div><button class="btn" id="imgCusNo">Cancel</button><button class="btn imgwrite" id="imgCusYes" style="padding:9px 16px;font-size:13px" data-help="Save settings">Save</button>`);
  const f = $i('#imgForm');
  const sync = () => {
    for(const [k, sel] of [['useHost','hostname'],['useUser','user,password'],['useWifi','wifiSSID,wifiPass,wifiCountry,wifiHidden'],['useLocale','timezone,keymap'],['ssh','sshMode,sshKeys']])
      for(const n of sel.split(',')) f.querySelectorAll(`[name=${n}]`).forEach(el => el.closest('.r,.chk,textarea') ? (el.closest('.r,.chk') || el).classList.toggle('off', !f[k].checked) : 0);
  };
  f.onchange = sync; sync();
  $i('#imgCusNo').onclick = closeModal;
  $i('#imgCusOff').onclick = () => { I.custom.on = false; saveCustom(); closeModal(); };
  $i('#imgCusYes').onclick = () => {
    const x = {on:true};
    for(const el of f.elements){ if(!el.name || el.type === 'radio') continue; x[el.name] = el.type === 'checkbox' ? el.checked : el.value.trim(); }
    x.sshPassword = f.sshMode.value !== 'key';
    I.secret = {password: x.password, wifiPass: x.wifiPass}; delete x.password; delete x.wifiPass;
    if(x.useUser && !x.user) return toastI('Type a username, or untick Username and password', true);
    if(x.useUser && !I.secret.password) return toastI('Type a password for ' + x.user, true);
    if(x.ssh && !x.useUser) return toastI('SSH needs the username and password set too', true);
    if(x.ssh && !x.sshPassword && !x.sshKeys) return toastI('Paste a public key, or choose “With the password”', true);
    if(x.useWifi && !x.wifiSSID) return toastI('Type the Wi-Fi name, or untick Wi-Fi', true);
    x.on = !!(x.useHost || x.useUser || x.useWifi || x.useLocale || x.ssh);
    I.custom = x; saveCustom(); closeModal();
  };
}
function saveCustom(){ try{ localStorage.setItem('flashie.imgcustom', JSON.stringify(I.custom)); }catch{} showCustomState(); }
function customBody(){
  const c = I.custom, o = {};
  if(c.useHost) o.hostname = c.hostname;
  if(c.useUser){ o.user = c.user; o.password = I.secret.password; }
  if(c.useWifi){ o.wifiSSID = c.wifiSSID; o.wifiPass = I.secret.wifiPass; o.wifiCountry = c.wifiCountry; o.wifiHidden = !!c.wifiHidden; }
  if(c.useLocale){ o.timezone = c.timezone; o.keymap = c.keymap; }
  if(c.ssh){ o.ssh = true; o.sshPassword = c.sshPassword; o.sshKeys = c.sshKeys; }
  return o;
}

/* ---------- drives ---------- */
async function openDisks(){
  if(I.busy) return;
  modal('Choose storage', '<div class="imgempty">Looking for drives…</div>', '<span>Plug in an SD card reader or USB drive — the list updates by itself.</span>');
  await renderDisks();
  clearInterval(I.diskTimer);
  I.diskTimer = setInterval(renderDisks, 2000);
}

async function loadDisks(){
  const r = await fetch('/api/img/disks').then(r => r.json()).catch(() => ({disks:[]}));
  I.disks = r.disks || [];
  I.diskErr = r.error;
  if(I.disk && !I.disks.some(d => d.id === I.disk.id && d.size === I.disk.size)) setDisk(null);
  return I.disks;
}

async function renderDisks(){
  const ds = await loadDisks();
  if($i('#imgMTitle').textContent !== 'Choose storage') return;
  const need = I.os ? I.os.extract_size : (I.file ? (I.file.windows ? I.file.windows.need : I.file.size) : 0);
  const rows = ds.map((d, i) => {
    const small = need && d.size < need;
    const where = d.mounts && d.mounts.length ? ' · ' + d.mounts.map(escI).join(', ') : '';
    return `<div class="imgitem ${small ? 'off' : ''}" data-i="${i}" title="${small ? 'Too small for this image' : ''}"><div class="ph">${d.bus === 'SD card' ? '💳' : '🔌'}</div><div class="t"><b>${escI(d.name)}</b><span>${escI(d.bus)} · ${escI(d.id)}${where}</span></div><div class="m">${hb(d.size)}${small ? '<br>too small' : ''}</div></div>`;
  }).join('') || `<div class="imgempty">${I.diskErr ? escI(I.diskErr) : 'No SD cards or USB drives found.<br>Plug one in and it will appear here.'}</div>`;
  $i('#imgMBody').innerHTML = rows;
  $i('#imgMBody').onclick = e => {
    const row = e.target.closest('.imgitem'); if(!row || row.classList.contains('off')) return;
    setDisk(I.disks[+row.dataset.i]); closeModal();
  };
}

function setDisk(d){
  I.disk = d;
  const b = $i('#imgDisk');
  b.innerHTML = d ? `<b>${escI(d.name)}</b><span>${hb(d.size)} · ${escI(d.bus)}</span>` : '<b>Choose storage</b><span>The SD card or USB drive to write to</span>';
  b.classList.toggle('set', !!d);
  update();
}

/* ---------- writing ---------- */
function update(){
  $i('#imgGo').disabled = I.busy || !I.disk || !(I.os || I.file);
  $i('#imgOS').disabled = $i('#imgDisk').disabled = I.busy;
}

function confirmWrite(){
  const d = I.disk; if(!d) return;
  const what = I.os ? I.os.name : I.file.name;
  const mounts = d.mounts && d.mounts.length ? ` (shown on this computer as ${d.mounts.map(escI).join(', ')})` : '';
  modal('Erase this drive?', `<div style="padding:12px 10px;display:flex;flex-direction:column;gap:12px">
    <div class="imgwarn"><b>Everything on ${escI(d.name)} — ${hb(d.size)}${mounts} will be erased</b> and replaced with ${escI(what)}.</div>
    ${I.file && I.file.windows ? `<div style="color:var(--muted)">Flashie will make ${I.file.windows.layout === 'FAT32' ? 'one FAT32 partition' : 'a FAT32 start-up partition and an exFAT one (install.wim is too big for FAT32)'} and copy the Windows installer onto it.</div>` : ''}
    ${customOn() ? '<div style="color:var(--muted)">Your OS settings will be added after writing.</div>' : ''}
    <div style="color:var(--muted)">Make sure this is the right drive. Flashie will ask for your password (or an administrator OK) so it can write to it.</div></div>`,
    `<div style="flex:1"></div><button class="btn" id="imgNo">Cancel</button><button class="btn imgwrite" id="imgYes" data-help="Erase and write" style="padding:9px 16px;font-size:13px">Erase and write</button>`);
  $i('#imgNo').onclick = closeModal;
  $i('#imgYes').onclick = () => { closeModal(); startWrite(); };
}

async function startWrite(){
  const body = {disk: I.disk.id, diskSize: I.disk.size, verify: $i('#imgVerify').checked, eject: $i('#imgEject').checked};
  if(I.os) body.item = I.os; else body.path = I.file.path;
  if(customOn()){
    if(I.custom.useUser && !I.secret.password){ openCustom(); toastI('Type the password again — Flashie doesn’t save it', true); return; }
    body.custom = customBody();
  }
  I.busy = true; I.phase = ''; update();
  setBar(0, 'run'); setText('Starting…');
  $i('#imgCancel').classList.remove('hidden');
  const r = await fetch('/api/img/write', {method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify(body)}).then(r => r.json()).catch(e => ({error:String(e)}));
  if(r.error) done(false, r.error);
}

function setBar(f, cls){
  const b = $i('#imgBar'); b.className = 'bar ' + (cls || '');
  b.querySelector('i').style.width = (f * 100).toFixed(1) + '%';
  $i('#imgPct').textContent = Math.floor(f * 100) + '%';
}
function setText(t){ $i('#imgText').innerHTML = t; }

const phaseName = {download:'Downloading', prepare:'Getting ready', write:'Writing', verify:'Checking', customise:'Adding your settings', finish:'Finishing'};

function onProgress(m){
  if(!I.busy) return;
  if(m.finished){ done(m.ok, m.err, m.warn); return; }
  if(m.phase !== I.phase){ I.phase = m.phase; I.lastDone = 0; I.lastT = 0; I.speed = 0; }
  const f = m.total ? Math.min(1, m.done / m.total) : 0;
  setBar(f, m.phase === 'prepare' ? 'idle' : '');
  let t = escI(m.text || phaseName[m.phase] || '');
  if(m.phase === 'write' && m.written) t += ' ' + hb(m.written) + ' written';
  if(m.phase === 'download' && m.done) t += ' ' + hb(m.done) + (m.total ? ' of ' + hb(m.total) : '');
  setText(t);
  const now = performance.now();
  const amt = m.phase === 'write' ? m.written : m.done;
  if(I.lastT && now - I.lastT > 900){
    const s = (amt - I.lastDone) / ((now - I.lastT) / 1000);
    I.speed = I.speed ? I.speed * 0.7 + s * 0.3 : s;
    I.lastT = now; I.lastDone = amt;
  } else if(!I.lastT){ I.lastT = now; I.lastDone = amt; }
  let sp = I.speed > 0 ? hb(I.speed) + '/s' : '';
  if(I.speed > 0 && m.total && m.phase !== 'write'){ const left = (m.total - m.done) / I.speed; if(left > 5) sp += ' · ' + fmtLeft(left) + ' left'; }
  $i('#imgSpeed').textContent = sp;
}

function fmtLeft(s){ return s >= 3600 ? Math.floor(s/3600) + ' h ' + Math.round(s%3600/60) + ' min' : s >= 60 ? Math.round(s/60) + ' min' : Math.round(s) + ' s'; }

function done(ok, err, warn){
  I.busy = false; update();
  $i('#imgCancel').classList.add('hidden');
  $i('#imgSpeed').textContent = '';
  if(ok){
    setBar(1, 'ok');
    setText(`<b>Done!</b> ${escI(I.os ? I.os.name : I.file.name)} is on ${escI(I.disk.name)}. ${$i('#imgEject').checked ? 'You can take it out now.' : 'Eject it before taking it out.'}${warn ? '<br>' + escI(warn) : ''}`);
    toastI('Image written');
  } else {
    setBar(1, 'bad');
    setText('<b>Didn’t finish:</b> ' + escI(err || 'something went wrong'));
  }
  setTimeout(loadDisks, 1500);
}

function toastI(msg, bad){ if(typeof toast === 'function') toast(msg, bad); }

build();
setTimeout(() => { try{ if(localStorage.getItem('flashie.view') === 'image'){ const b = document.querySelector('.viewtabs button[data-v="image"]'); if(b) b.click(); } }catch{} }, 50);
})();
