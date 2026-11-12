// Fixie's drawings: a line icon for each kind of device, and side-by-side
// "good" and "faulty" pictures for the signs of a failed part. Colours come
// from app.css classes, so they follow the chosen theme. Created by nrmluvdev.
window.ART = (() => {
  const ic = p => `<svg viewBox="0 0 24 24" aria-hidden="true"><g class="ico">${p}</g></svg>`;
  const icons = {
    phone: '<rect x="6.5" y="2.5" width="11" height="19" rx="2.2"/><path d="M10.5 18.5h3"/>',
    laptop: '<rect x="4.5" y="5" width="15" height="10" rx="1.2"/><path d="M2.5 18.5h19l-1.5-3.5h-16z"/>',
    desktop: '<rect x="3" y="4" width="13" height="10" rx="1.2"/><path d="M7 18h5M9.5 14v4"/><rect x="18" y="5" width="3.5" height="14" rx=".8"/>',
    console: '<path d="M7 8h10a4 4 0 0 1 4 4.5l-.6 3.3a2.4 2.4 0 0 1-4.1 1.2L14.5 15h-5l-1.8 2a2.4 2.4 0 0 1-4.1-1.2L3 12.5A4 4 0 0 1 7 8z"/><path d="M7.5 11v3M6 12.5h3"/><circle cx="16" cy="12" r=".6"/><circle cx="17.6" cy="13.4" r=".6"/>',
    controller: '<path d="M7 8h10a4 4 0 0 1 4 4.5l-.6 3.3a2.4 2.4 0 0 1-4.1 1.2L14.5 15h-5l-1.8 2a2.4 2.4 0 0 1-4.1-1.2L3 12.5A4 4 0 0 1 7 8z"/><path d="M7.5 11v3M6 12.5h3"/><circle cx="16" cy="12" r=".6"/><circle cx="17.6" cy="13.4" r=".6"/>',
    remote: '<rect x="8" y="2.5" width="8" height="19" rx="2.5"/><circle cx="12" cy="7" r="1.5"/><path d="M10.5 11.5h3M10.5 14.5h3M10.5 17.5h3"/>',
    tv: '<rect x="2.5" y="4" width="19" height="12.5" rx="1.2"/><path d="M8 20.5h8M12 16.5v4"/>',
    audio: '<rect x="3" y="7" width="18" height="10" rx="1.5"/><circle cx="8" cy="12" r="2.5"/><path d="M13.5 10.5h5M13.5 13.5h5"/>',
    earbuds: '<path d="M4 14v-2a8 8 0 0 1 16 0v2"/><rect x="2.5" y="13.5" width="4" height="6.5" rx="1.5"/><rect x="17.5" y="13.5" width="4" height="6.5" rx="1.5"/>',
    heater: '<path d="M6 9h10l-1 11H7z"/><path d="M16 11h1.5a2 2 0 0 1 0 4H15.6"/><path d="M8 3.5c0 1.5 1.5 1.5 1.5 3M11.5 3.5c0 1.5 1.5 1.5 1.5 3"/>',
    motorhome: '<circle cx="12" cy="11" r="7.5"/><circle cx="12" cy="11" r="1.3"/><path d="M12 9.7c0-3 1.5-4.5 3-4.5M13.2 11.6c2.5 1.5 3 3.5 2.2 4.8M10.8 11.6c-2.5 1.5-4.3.8-4.6-.8"/><path d="M9 21h6M12 18.5V21"/>',
    washer: '<rect x="4" y="2.5" width="16" height="19" rx="1.5"/><path d="M4 7h16"/><circle cx="12" cy="14" r="4.2"/><circle cx="7" cy="4.8" r=".6"/>',
    powertool: '<path d="M3 6h13v5H9l-1 3H6V11H3z"/><path d="M16 7.5h4M8 14l-1.5 7h5L13 14"/>',
    lights: '<path d="M9 17h6M10 20h4"/><path d="M12 3a6 6 0 0 0-3.5 10.9c.5.4.5 1.1.5 1.6V17h6v-1.5c0-.5 0-1.2.5-1.6A6 6 0 0 0 12 3z"/>',
    diy: '<rect x="5" y="5" width="14" height="14" rx="1.5"/><rect x="9" y="9" width="6" height="6"/><path d="M8 2v3M12 2v3M16 2v3M8 19v3M12 19v3M16 19v3M2 8h3M2 12h3M2 16h3M19 8h3M19 12h3M19 16h3"/>',
    printer3d: '<path d="M4 3h16v18H4zM4 7h16"/><path d="M10 7v4l2 2 2-2V7"/><path d="M8 19h8"/>',
    toy: '<rect x="5" y="9" width="14" height="10" rx="2"/><circle cx="9" cy="13.5" r="1.3"/><circle cx="15" cy="13.5" r="1.3"/><path d="M12 9V5M10 4h4"/>',
    keyboard: '<rect x="2.5" y="6" width="19" height="12" rx="1.5"/><path d="M6 10h.01M9 10h.01M12 10h.01M15 10h.01M18 10h.01M6 14h.01M18 14h.01M9 14h6"/>',
    microwave: '<rect x="2.5" y="5" width="19" height="14" rx="1.5"/><rect x="5" y="8" width="10" height="8" rx="1"/><path d="M18 8.5v.01M18 11v.01M18 13.5h.01"/>',
  };
  const device = id => ic(icons[id] || icons.diy);

  // ---- sign drawings: 300×130, good on the left, faulty on the right
  const frame = (good, bad) => `<svg viewBox="0 0 300 130" role="img" aria-label="Good and faulty examples">
    <rect class="sv-board" x="4" y="14" width="140" height="104" rx="6"/><rect class="sv-board" x="156" y="14" width="140" height="104" rx="6"/>
    <text class="sv-good" x="10" y="10">GOOD</text><text class="sv-badtxt" x="162" y="10">FAULTY</text>${good}${bad}</svg>`;
  const cap = (x, bulge) => `<rect class="sv-cap" x="${x - 18}" y="40" width="36" height="58" rx="5"/>
    ${bulge ? `<path class="sv-top" d="M${x - 18} 44 Q${x} 24 ${x + 18} 44 Z"/><path class="sv-line" d="M${x - 8} 36 l16 0"/><ellipse class="sv-burn" cx="${x + 10}" cy="100" rx="16" ry="5"/>` : `<rect class="sv-top" x="${x - 18}" y="40" width="36" height="7" rx="3"/><path stroke="#2a2f3b" stroke-width="1.5" d="M${x - 10} 44h20M${x} 41v6"/>`}
    <rect class="sv-dark" x="${x + 9}" y="47" width="5" height="50"/>`;
  const fuse = (x, blown) => `<rect class="sv-glass" x="${x - 45}" y="52" width="90" height="28" rx="6"/>
    <rect class="sv-metal" x="${x - 52}" y="50" width="14" height="32" rx="3"/><rect class="sv-metal" x="${x + 38}" y="50" width="14" height="32" rx="3"/>
    ${blown ? `<path class="sv-wire" d="M${x - 38} 66 L${x - 10} 66 l4 -6"/><path class="sv-wire" d="M${x + 38} 66 L${x + 10} 66 l-4 7"/><ellipse class="sv-burn" opacity=".75" cx="${x}" cy="66" rx="20" ry="11"/>` : `<path class="sv-wire" d="M${x - 38} 66 H${x + 38}"/>`}`;
  const batt = (x, swollen) => swollen
    ? `<path class="sv-batt" d="M${x - 50} 52 Q${x} 18 ${x + 50} 52 L${x + 50} 88 Q${x} 116 ${x - 50} 88 Z"/><path class="sv-line" d="M${x - 30} 40 Q${x} 26 ${x + 30} 40"/>`
    : `<rect class="sv-batt" x="${x - 50}" y="55" width="100" height="30" rx="4"/><rect class="sv-metal" x="${x - 10}" y="50" width="20" height="6" rx="2"/>`;
  const joint = (x, cracked) => `<ellipse class="sv-pad" cx="${x}" cy="92" rx="38" ry="10"/>
    <path class="sv-metal" d="M${x - 30} 92 Q${x - 6} 86 ${x - 5} 56 L${x + 5} 56 Q${x + 6} 86 ${x + 30} 92 Z"/>
    <rect class="sv-dark" x="${x - 4}" y="30" width="8" height="30"/>
    ${cracked ? `<path class="sv-line" d="M${x - 9} 70 Q${x} 76 ${x + 9} 70"/><path class="sv-line" d="M${x - 14} 82 l4 -4M${x + 12} 80 l3 4"/>` : ""}`;
  const contact = (x, crust) => `<rect class="sv-metal" x="${x - 30}" y="40" width="60" height="54" rx="4"/><path stroke="#2a2f3b" stroke-width="3" fill="none" d="M${x - 18} 58 q6 -8 12 0 t12 0 t12 0M${x - 18} 74 q6 -8 12 0 t12 0 t12 0"/>
    ${crust ? `<circle class="sv-crust" cx="${x - 14}" cy="52" r="10"/><circle class="sv-crust" cx="${x + 6}" cy="80" r="13"/><circle class="sv-crust" cx="${x + 18}" cy="58" r="7"/><circle class="sv-crust" cx="${x - 20}" cy="86" r="6"/>` : ""}`;
  const resistor = (x, burnt) => `<path class="sv-wire" d="M${x - 60} 66 H${x + 60}"/><rect class="sv-top" x="${x - 34}" y="54" width="68" height="24" rx="10"/>
    <rect x="${x - 22}" y="54" width="6" height="24" fill="#8b4513"/><rect x="${x - 10}" y="54" width="6" height="24" fill="#111"/><rect x="${x + 2}" y="54" width="6" height="24" fill="#d00"/>
    ${burnt ? `<ellipse class="sv-burn" cx="${x + 4}" cy="66" rx="34" ry="22" opacity=".85"/><path class="sv-line" d="M${x - 4} 58 l6 8 -5 8"/>` : ""}`;
  const port = (x, bad) => `<rect class="sv-metal" x="${x - 34}" y="46" width="68" height="40" rx="12"/><rect class="sv-dark" x="${x - 26}" y="54" width="52" height="24" rx="8"/>
    <rect class="sv-pad" x="${x - 16}" y="62" width="32" height="8" rx="2"/>
    ${bad ? `<path fill="#9aa1ad" d="M${x - 24} 76 q10 -14 20 -4 q10 -12 22 0 v4 h-42z"/><path class="sv-line" d="M${x + 30} 46 l8 -8M${x + 34} 86 l8 8"/>` : ""}`;
  const pics = {
    caps: frame(cap(74, false), cap(226, true)),
    fuse: frame(fuse(74, false), fuse(226, true)),
    battery: frame(batt(74, false), batt(226, true)),
    joint: frame(joint(74, false), joint(226, true)),
    corrosion: frame(contact(74, false), contact(226, true)),
    burnt: frame(resistor(74, false), resistor(226, true)),
    connector: frame(port(74, false), port(226, true)),
  };
  return { device, sign: id => pics[id] || "" };
})();
