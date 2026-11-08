// Build volumes of common 3D printers (X × Y × Z, mm), from the makers' published specs.
// Grouped by brand. "Custom" lets the user type their own.
export const PRINTERS = [
  ['Bambu Lab', [
    ['X1 Carbon', 256, 256, 256], ['X1E', 256, 256, 256], ['P1S', 256, 256, 256], ['P1P', 256, 256, 256],
    ['P2S', 256, 256, 256], ['A1', 256, 256, 256], ['A1 mini', 180, 180, 180], ['H2D', 350, 320, 325], ['H2S', 340, 320, 340],
  ]],
  ['Prusa', [
    ['Core One', 250, 220, 270], ['MK4S / MK4', 250, 210, 220], ['MK3S+', 250, 210, 210], ['MINI+', 180, 180, 180], ['XL', 360, 360, 360],
  ]],
  ['Creality', [
    ['Ender-3 / V2 / Pro', 220, 220, 250], ['K1 / K1C', 220, 220, 250], ['K1 Max', 300, 300, 300], ['K2 Plus', 350, 350, 350], ['Hi', 260, 260, 300], ['Ender-3 V3', 220, 220, 250], ['Ender-3 V3 SE', 220, 220, 250], ['Ender-3 V3 KE', 220, 220, 240],
    ['Ender-3 V3 Plus', 300, 300, 330], ['Ender-3 S1 / S1 Pro', 220, 220, 270], ['Ender-3 Max Neo', 300, 300, 320],
    ['Ender-5 S1', 220, 220, 280], ['Ender-5 Plus', 350, 350, 400], ['CR-10 SE', 220, 220, 265], ['CR-10 V3', 300, 300, 400], ['CR-10 S5', 500, 500, 500],
  ]],
  ['Elegoo', [
    ['Centauri Carbon', 256, 256, 256], ['Neptune 4 / 4 Pro', 225, 225, 265], ['Neptune 4 Plus', 320, 320, 385], ['Neptune 4 Max', 420, 420, 480], ['Neptune 3 Pro', 225, 225, 280],
  ]],
  ['Anycubic', [
    ['Kobra S1', 250, 250, 250], ['Kobra 3', 250, 250, 260], ['Kobra 2 / Neo / Pro', 220, 220, 250], ['Kobra 2 Plus', 320, 320, 400], ['Kobra 2 Max', 420, 420, 500],
    ['Vyper', 245, 245, 260], ['i3 Mega', 210, 210, 205],
  ]],
  ['Qidi', [['Q1 Pro', 245, 245, 240], ['Plus4', 305, 305, 280], ['X-Plus 3', 280, 280, 270], ['X-Max 3', 325, 325, 315]]],
  ['Sovol', [['SV06', 220, 220, 250], ['SV06 Plus', 300, 300, 340], ['SV07', 220, 220, 250], ['SV08', 350, 350, 345]]],
  ['Voron', [
    ['V0.2', 120, 120, 120], ['Trident 250', 250, 250, 250], ['Trident 300', 300, 300, 250], ['Trident 350', 350, 350, 250],
    ['2.4 250', 250, 250, 240], ['2.4 300', 300, 300, 290], ['2.4 350', 350, 350, 340],
  ]],
  ['Flashforge', [['Adventurer 5M / 5M Pro', 220, 220, 220], ['AD5X', 220, 220, 220]]],
  ['Artillery', [['Sidewinder X2', 300, 300, 400], ['Sidewinder X4 Plus', 300, 300, 400], ['Genius Pro', 220, 220, 250]]],
  ['Ultimaker', [['S3', 230, 190, 200], ['S5', 330, 240, 300], ['2+ Connect', 223, 220, 205]]],
  ['Others', [
    ['Anker M5', 235, 235, 250], ['Raise3D Pro3', 300, 300, 300], ['Snapmaker J1', 300, 200, 200], ['Snapmaker Artisan', 400, 400, 400],
    ['Snapmaker 2.0 A350', 320, 350, 330], ['LulzBot Mini 2', 160, 160, 180], ['LulzBot TAZ 6', 280, 280, 250], ['MakerBot Replicator+', 295, 195, 165],
  ]],
];

export const printerLabel = (brand, name) => brand + ' ' + name;

export function findPrinter(label) {
  for (const [brand, list] of PRINTERS) for (const p of list) if (printerLabel(brand, p[0]) === label) return { brand, name: p[0], bed: [p[1], p[2], p[3]] };
  return null;
}
