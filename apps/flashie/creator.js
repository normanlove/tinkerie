/* Flashie — Firmware Creator.
   A Scratch-style block editor that writes MicroPython (or Arduino C++ for
   AVR boards), explains every block and every line, and uploads + runs it.
   Made by nrmluvdev. */
'use strict';

/* ======================================================================
   Board knowledge: pins, LEDs, buttons, safety notes
   ====================================================================== */

const ESP_RULES = {
  'ESP32': {
    pins: [0,1,2,3,4,5,12,13,14,15,16,17,18,19,21,22,23,25,26,27,32,33,34,35,36,39],
    strap: {0:'Boot-mode pin: if it is held LOW while the board starts, the board waits to be flashed instead of running your program.',
            2:'Boot-mode pin: must not be held HIGH while the board starts, or flashing can fail.',
            5:'Boot pin (timing). Usually fine, but avoid things that pull it LOW at power-on.',
            12:'Sets the flash-chip voltage at power-on. If something pulls it HIGH while the board starts, the board may not boot at all.',
            15:'Boot pin (controls the boot messages). Usually fine as an output.'},
    inOnly: [34,35,36,39],
    adc: [32,33,34,35,36,39], adc2: [0,2,4,12,13,14,15,25,26,27],
    serial: [1,3], usb: [], boot: 0,
    i2c: [21,22], prefer: [27,22,26,25,33,32,4,13,14,16,17,18,19,21,23],
  },
  'ESP32-S2': {
    pins: [0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,21,33,34,35,36,37,38,39,40,41,42,45,46],
    strap: {0:'Boot-mode pin: held LOW at start-up, the board waits to be flashed.', 45:'Sets the flash voltage at start-up — avoid.', 46:'Boot-mode pin — avoid.'},
    inOnly: [46], adc: [1,2,3,4,5,6,7,8,9,10], adc2: [11,12,13,14,15,16,17,18,19,20], serial: [43,44], usb: [19,20], boot: 0,
    i2c: [8,9], prefer: [4,5,6,7,15,16,17,18,33,34,35,36],
  },
  'ESP32-S3': {
    pins: [0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,21,38,39,40,41,42,43,44,45,46,47,48],
    strap: {0:'Boot-mode pin: held LOW at start-up, the board waits to be flashed.', 3:'JTAG strapping pin — usually fine, avoid pulling it at start-up.',
            45:'Sets the flash voltage at start-up — avoid.', 46:'Boot-mode pin — avoid.'},
    inOnly: [], adc: [1,2,3,4,5,6,7,8,9,10], adc2: [11,12,13,14,15,16,17,18,19,20], serial: [43,44], usb: [19,20], boot: 0,
    i2c: [8,9], prefer: [4,5,6,7,15,16,17,18,8,9,10,11,12,13,14,21],
  },
  'ESP32-C3': {
    pins: [0,1,2,3,4,5,6,7,8,9,10,20,21],
    strap: {2:'Boot pin — must not be held LOW at start-up.', 8:'Boot pin — must be HIGH at start-up (often the built-in LED).', 9:'Boot button: held LOW at start-up, the board waits to be flashed.'},
    inOnly: [], adc: [0,1,2,3,4], adc2: [], serial: [20,21], usb: [18,19], boot: 9,
    i2c: [8,9], prefer: [4,5,6,7,10,3,1,0],
  },
  'ESP32-C6': {
    pins: [0,1,2,3,4,5,6,7,8,9,10,11,14,15,18,19,20,21,22,23],
    strap: {8:'Boot pin — avoid pulling it at start-up.', 9:'Boot button: held LOW at start-up, the board waits to be flashed.', 15:'Boot strapping pin — avoid.'},
    inOnly: [], adc: [0,1,2,3,4,5,6], adc2: [], serial: [16,17], usb: [12,13], boot: 9,
    i2c: [6,7], prefer: [4,5,6,7,10,11,18,19,20,21,22,23],
  },
  'ESP32-H2': {
    pins: [0,1,2,3,4,5,8,9,10,11,12,13,14,22,25], strap: {8:'Boot pin', 9:'Boot button', 25:'Boot pin'},
    inOnly: [], adc: [1,2,3,4,5], adc2: [], serial: [23,24], usb: [26,27], boot: 9, i2c: [12,22], prefer: [4,5,10,11,12,13,14,22],
  },
  'ESP8266': {
    pins: [0,1,2,3,4,5,12,13,14,15,16], dlabel: {16:'D0',5:'D1',4:'D2',0:'D3',2:'D4',14:'D5',12:'D6',13:'D7',15:'D8',3:'RX',1:'TX'},
    strap: {0:'FLASH button: held LOW at start-up, the board waits to be flashed.', 2:'Must be HIGH at start-up (it is also the built-in LED).', 15:'Must be LOW at start-up — has a pull-down resistor; use it only as an output.'},
    inOnly: [], adc: ['A0'], adc2: [], serial: [1,3], usb: [], boot: 0, i2c: [4,5], prefer: [5,4,14,12,13],
    notes: {16:'No PWM or interrupts on this pin.'},
  },
  'RP2040': {
    pins: [0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,26,27,28],
    strap: {}, inOnly: [], adc: [26,27,28], adc2: [], serial: [], usb: [], boot: null, i2c: [4,5], prefer: [15,16,17,14,13,12,11,10,2,3],
  },
};
ESP_RULES['RP2350'] = ESP_RULES['RP2040'];

const AVR_PINS = {
  std: {digital: [0,1,2,3,4,5,6,7,8,9,10,11,12,13], analog: ['A0','A1','A2','A3','A4','A5'], pwm: [3,5,6,9,10,11], i2c: ['A4','A5'], led: 13},
  nano: {digital: [0,1,2,3,4,5,6,7,8,9,10,11,12,13], analog: ['A0','A1','A2','A3','A4','A5','A6','A7'], analogOnly: ['A6','A7'], pwm: [3,5,6,9,10,11], i2c: ['A4','A5'], led: 13},
  mega: {digital: Array.from({length:54},(_,i)=>i), analog: Array.from({length:16},(_,i)=>'A'+i), pwm: [2,3,4,5,6,7,8,9,10,11,12,13,44,45,46], i2c: [20,21], led: 13},
  leo: {digital: [0,1,2,3,4,5,6,7,8,9,10,11,12,13], analog: ['A0','A1','A2','A3','A4','A5'], pwm: [3,5,6,9,10,11,13], i2c: [2,3], led: 13},
};

const BOARD_LED = {
  'esp32-devkit':{pin:2}, 'nodemcu-32s':{pin:2}, 'esp32-cam':{pin:33, low:true}, 'm5atom-lite':{pin:27, neo:true}, 'm5stamp-c3':{pin:2, neo:true},
  'xiao-c6':{pin:15, low:true}, 'ttgo-t-beam':{pin:4, low:true}, 'wemos-d1-mini32':{pin:2}, 'esp32-generic':{pin:2}, 'lolin32-lite':{pin:22, low:true},
  'm5stickc-plus':{pin:10, low:true}, 'esp32-s3-devkitc':{pin:48, neo:true}, 'xiao-s3':{pin:21, low:true}, 'c3-supermini':{pin:8, low:true},
  'esp32-c3-devkitm':{pin:8, neo:true}, 'esp32-c6-devkitc':{pin:8, neo:true}, 's3-zero':{pin:21, neo:true}, 'nano-esp32':{pin:48},
  'nodemcu-v2':{pin:2, low:true}, 'nodemcu-v3':{pin:2, low:true}, 'd1-mini':{pin:2, low:true}, 'esp01':{pin:2, low:true}, 'esp8266-generic':{pin:2, low:true},
  'pico':{pin:25}, 'pico2':{pin:25}, 'pico-w':{pin:'LED'}, 'pico2-w':{pin:'LED'}, 'rp2040-zero':{pin:16, neo:true}, 'xiao-rp2040':{pin:25, low:true},
  'rp2040-generic':{pin:25}, 'rp2350-generic':{pin:25}, 'esp32-s2-mini':{pin:15}, 'heltec-v3':{pin:35},
};

function creatorCtx(){
  const b = currentBoard();
  let chip = (S.chip && S.chip.family==='esp') ? S.chip.chip : (b && b.chip) || expectedChip() || '';
  let fam = b ? b.family : family();
  const generic = !chip;
  if(generic){ chip = 'ESP32'; fam = 'esp'; }
  if(/^RP2/.test(chip)) fam = 'rp2';
  const lang = fam==='avr' ? 'cpp' : (fam==='esp' || fam==='rp2') ? 'py' : null;
  const cyd = b && b.cyd ? (S.cyds||[]).find(c=>c.id===b.cyd) : null;
  const ctx = {board:b, boardId:b?b.id:'', boardName: b ? b.name : (generic ? 'Generic ESP32 (no board picked)' : chip), chip, fam, lang, cyd, generic};
  ctx.hasWifi = /^ESP32(?!-H2)|^ESP8266/.test(chip) || ['pico-w','pico2-w'].includes(ctx.boardId);
  ctx.pins = pinDB(ctx);
  ctx.led = ledFor(ctx);
  ctx.boot = fam==='avr' ? 2 : (ESP_RULES[chip]||{}).boot;
  const R = ESP_RULES[chip];
  ctx.i2c = fam==='avr' ? avrSet(ctx).i2c : (R ? R.i2c : [21,22]);
  if(cyd && fam==='esp'){ // the CYD's free I2C-capable pins (CN1 on the 2.8")
    const free = ctx.pins.filter(p=>p.level==='ok' && typeof p.id==='number' && !(R.inOnly||[]).includes(p.id)).map(p=>p.id);
    if(free.includes(22) && free.includes(27)) ctx.i2c = [27,22];
  }
  return ctx;
}

function avrSet(ctx){
  const id = ctx.boardId;
  if(/^mega/.test(id)) return AVR_PINS.mega;
  if(/leonardo|micro/.test(id)) return AVR_PINS.leo;
  if(/^nano|pro-mini/.test(id)) return AVR_PINS.nano;
  return AVR_PINS.std;
}

function pinLabel(ctx, id){
  if(id === 'LED') return 'built-in LED';
  if(ctx.fam==='avr') return typeof id==='number' ? 'D'+id : id;
  if(ctx.fam==='rp2') return 'GP'+id;
  const R = ESP_RULES[ctx.chip]||{};
  if(id==='A0') return 'A0 (analog)';
  return 'GPIO '+id + (R.dlabel && R.dlabel[id] ? ` (${R.dlabel[id]})` : '');
}

// pinDB lists every usable pin with a safety level: ok | care | used | avoid.
function pinDB(ctx){
  const out = [];
  const add = (id, level, notes, tags) => out.push({id, label: pinLabel(ctx,id), level, notes: notes||[], tags: tags||{}});
  if(ctx.fam==='avr'){
    const A = avrSet(ctx);
    for(const d of A.digital){
      if(d<=1) add(d, 'avoid', ['Used by the USB serial connection (uploading and Serial prints). Using it can stop uploads working.']);
      else if(d===A.led) add(d, 'care', ['Also drives the built-in LED.'], {pwm:A.pwm.includes(d)});
      else if(A.i2c.includes(d)) add(d, 'care', ['This is an I²C pin (SDA/SCL).'], {pwm:A.pwm.includes(d)});
      else add(d, 'ok', A.pwm.includes(d) ? ['Supports PWM (~ on the board) — good for brightness and servos.'] : [], {pwm:A.pwm.includes(d)});
    }
    for(const a of A.analog){
      const only = (A.analogOnly||[]).includes(a);
      const n = [only ? 'Analog input only.' : 'Analog input (can also be used as a digital pin).'];
      if(A.i2c.includes(a)) n.push('Also the I²C '+(a===A.i2c[0]?'SDA':'SCL')+' pin.');
      add(a, A.i2c.includes(a) ? 'care' : 'ok', n, {adc:true, inOnly:only});
    }
    return out;
  }
  const R = ESP_RULES[ctx.chip] || ESP_RULES['ESP32'];
  const cydUse = {}, cydExtra = {};
  if(ctx.cyd) for(const g of ctx.cyd.groups) for(const p of g.pins){
    const what = `${g.name.replace(/ — .*/, '')}${g.name.includes('—') ? ' ('+g.name.split('— ')[1]+')' : ''}: ${p.f}`;
    (/extras|Relay/.test(g.name) ? cydExtra : cydUse)[p.g] = what;
  }
  const led = BOARD_LED[ctx.boardId];
  const list = [...R.pins];
  if(led && typeof led.pin === 'number' && !list.includes(led.pin)) list.push(led.pin);
  if(R.adc.includes('A0')) list.push('A0');
  for(const id of list){
    const notes = [], tags = {};
    let level = 'ok';
    if(id === 'A0'){ add('A0', 'ok', ['The only analog input on the ESP8266 (0–1 V on a bare chip, 0–3.3 V on NodeMCU/D1 boards).'], {adc:true, inOnly:true}); continue; }
    if(R.adc.includes(id)) tags.adc = true;
    if(R.adc2.includes(id)){ tags.adc = true; tags.adc2 = true; }
    if(R.inOnly.includes(id)){ tags.inOnly = true; notes.push('Input only — it can read buttons and sensors but can’t switch things on, and it has no built-in pull-up resistor.'); level = 'care'; }
    if(R.strap[id]){ notes.push(R.strap[id]); level = 'care'; }
    if(R.serial.includes(id)){ notes.push('Used by the USB serial console (TX/RX). Using it breaks uploading and print() output.'); level = 'avoid'; }
    if(R.usb.includes(id)){ notes.push('Used by the chip’s built-in USB. Using it disconnects the board from your PC.'); level = 'avoid'; }
    if(R.notes && R.notes[id]) notes.push(R.notes[id]);
    if(ctx.chip==='RP2040' || ctx.chip==='RP2350'){ if([26,27,28].includes(id)) notes.push('Also an analog input (ADC).'); }
    if(led && led.pin === id){ notes.push('Wired to the built-in LED.'); if(level==='ok') level = 'care'; }
    if(cydUse[id]){ notes.unshift('Used by this CYD — '+cydUse[id]+'. Using it will break the screen, touch or SD card.'); level = 'used'; }
    else if(cydExtra[id]){ notes.unshift('Wired to this CYD’s '+cydExtra[id].split(': ')[1]+'.'); if(level==='ok') level = 'care'; }
    if(ctx.cyd && ctx.cyd.chip==='ESP32' && [22,27,35].includes(id) && !cydUse[id]) notes.push('Free pin on the CYD’s side connector (CN1/P3).');
    add(id, level, notes, tags);
  }
  return out;
}

function ledFor(ctx){
  if(ctx.fam==='avr') return {kind:'pin', pin: avrSet(ctx).led, low:false};
  if(ctx.cyd && ctx.cyd.groups.some(g=>/extras/.test(g.name) && g.pins.some(p=>/RGB LED red/.test(p.f)))) return {kind:'rgb', r:4, g:16, b:17, low:true};
  const l = BOARD_LED[ctx.boardId];
  if(!l) return null;
  if(l.pin === 'LED') return {kind:'name', pin:'LED'};
  return {kind: l.neo ? 'neo' : 'pin', pin: l.pin, low: !!l.low};
}

const pinInfo = (ctx, id) => ctx.pins.find(p => String(p.id) === String(id));

// pinWarn returns a plain-English problem for using a pin in a given way.
function pinWarn(ctx, id, mode){
  const p = pinInfo(ctx, id);
  if(!p) return `${pinLabel(ctx,id)} doesn’t exist on this board (or is used internally).`;
  if(p.level === 'used') return p.notes[0];
  if(p.level === 'avoid') return `${p.label}: ${p.notes[0]}`;
  if((mode==='out' || mode==='pwm') && p.tags.inOnly) return `${p.label} is input-only — it can’t switch an LED, servo or relay.`;
  if(mode==='adc' && !p.tags.adc) return `${p.label} can’t read analog values. Use one of: ${ctx.pins.filter(x=>x.tags.adc).map(x=>x.label).join(', ')}.`;
  if(mode==='adc' && p.tags.adc2 && ctx.hasWifi) return `${p.label} is on ADC2, which stops working while Wi-Fi is on. Prefer an ADC1 pin.`;
  if(mode==='pwm' && ctx.fam==='avr' && !p.tags.pwm) return `${p.label} doesn’t support PWM on this board — use a pin marked ~ (${ctx.pins.filter(x=>x.tags.pwm).map(x=>x.label).join(', ')}).`;
  if(mode==='in_up' && p.tags.inOnly && ctx.fam==='esp') return `${p.label} has no internal pull-up resistor, so a simple button won’t read reliably. Add a 10 kΩ resistor to 3.3 V or use another pin.`;
  if(p.level === 'care') return null; // shown as a soft note, not a warning
  return null;
}

function freePin(ctx, mode){
  if(ctx.fam==='avr') return mode==='adc' ? 'A0' : mode==='pwm' ? 9 : 7;
  const R = ESP_RULES[ctx.chip] || ESP_RULES.ESP32;
  if(mode==='adc'){ const a = ctx.pins.find(p=>p.tags.adc && !p.tags.adc2 && p.level!=='used' && p.level!=='avoid'); return a ? a.id : R.adc[0]; }
  for(const id of R.prefer){ const p = pinInfo(ctx, id); if(p && p.level==='ok' && !(mode!=='in' && p.tags.inOnly)) return id; }
  const p = ctx.pins.find(p=>p.level==='ok' && !p.tags.inOnly) || ctx.pins.find(p=>p.level==='care' && !p.tags.inOnly && !(BOARD_LED[ctx.boardId] && BOARD_LED[ctx.boardId].pin===p.id));
  return p ? p.id : R.prefer[0];
}

/* ======================================================================
   Block definitions
   ====================================================================== */

const CATS = [
  {id:'control', name:'Start & loops', color:'#ffab19'},
  {id:'pins', name:'Pins', color:'#4c97ff'},
  {id:'leds', name:'LEDs', color:'#9966ff'},
  {id:'buttons', name:'Buttons', color:'#ff6680'},
  {id:'vars', name:'Variables', color:'#ff8c1a'},
  {id:'ops', name:'Maths & logic', color:'#59c059'},
  {id:'output', name:'Console & files', color:'#5cb1d6'},
  {id:'wifi', name:'Wi-Fi & web', color:'#0fbd8c'},
  {id:'sensors', name:'Sensors', color:'#cf63cf'},
  {id:'motion', name:'Servos', color:'#e05c5c'},
  {id:'display', name:'Screens', color:'#d9a400'},
];
const catOf = id => CATS.find(c=>c.id===id);

// label part helpers
const F = (k, name, extra) => Object.assign({k, name}, extra||{});
const ONOFF = [['on','on'],['off','off']];
const COLORS = [['#ff0000','red'],['#00c800','green'],['#0050ff','blue'],['#ffffff','white'],['#ffd000','yellow'],['#ff00ff','magenta'],['#00e0e0','cyan'],['#ff7a00','orange'],['#000000','black / off']];

const B = {};
function def(type, d){ B[type] = Object.assign({type}, d); }

def('start', {cat:'control', shape:'hat', label:['when the board starts'],
  ex:'Your program begins here. Everything stacked underneath runs from top to bottom, once, every time the board is powered on or reset.'});
def('forever', {cat:'control', shape:'c', label:['forever'],
  ex:'Repeats the blocks inside it again and again, for as long as the board has power. Most programs end with one of these.'});
def('repeat', {cat:'control', shape:'c', label:['repeat', F('in','n',{acc:'val', lit:'num', def:10}), 'times'],
  ex:'Runs the blocks inside it a set number of times, then carries on with whatever is below.'});
def('while', {cat:'control', shape:'c', label:['repeat while', F('in','cond',{acc:'bool'})],
  ex:'Keeps running the blocks inside as long as the condition is true. It checks the condition before each round.'});
def('if', {cat:'control', shape:'c', label:['if', F('in','cond',{acc:'bool'}), 'then'],
  ex:'Only runs the blocks inside when the condition is true. If it is false, they are skipped.'});
def('ifelse', {cat:'control', shape:'cc', label:['if', F('in','cond',{acc:'bool'}), 'then'], label2:['else'],
  ex:'Chooses between two paths: the first group runs when the condition is true, the “else” group runs when it is false.'});
def('wait', {cat:'control', shape:'stmt', label:['wait', F('in','s',{acc:'val', lit:'num', def:1}), 'seconds'],
  ex:'Pauses the program. Nothing else happens during the wait. 0.5 means half a second.'});
def('waitms', {cat:'control', shape:'stmt', label:['wait', F('in','ms',{acc:'val', lit:'num', def:100}), 'milliseconds'],
  ex:'A short pause measured in thousandths of a second — 1000 ms is one second.'});

def('pinwrite', {cat:'pins', shape:'stmt', label:['set pin', F('pin','pin',{mode:'out'}), 'to', F('drop','state',{opts:ONOFF})],
  ex:'Switches a pin on (3.3 V, or 5 V on an Arduino) or off (0 V). An LED wired to that pin (through a resistor) will light up or go dark.'});
def('pintoggle', {cat:'pins', shape:'stmt', label:['toggle pin', F('pin','pin',{mode:'out'})],
  ex:'Flips a pin: if it was on it goes off, if it was off it goes on.'});
def('pinread', {cat:'pins', shape:'bool', label:['pin', F('pin','pin',{mode:'in'}), 'is HIGH', F('drop','pull',{opts:[['up','(pull-up)'],['none','(no pull)'],['down','(pull-down)']]})],
  ex:'Checks whether a pin is receiving voltage (HIGH) or not (LOW). A pull-up gently holds the pin HIGH when nothing is connected, so it doesn’t “float” and give random answers.'});
def('analogread', {cat:'pins', shape:'num', label:['analog level of pin', F('pin','pin',{mode:'adc'}), '(0–100 %)'],
  ex:'Measures a voltage between 0 V and 3.3 V (5 V on Arduino) and turns it into a percentage. Great for knobs (potentiometers) and light sensors.'});
def('pwm', {cat:'pins', shape:'stmt', label:['set brightness of pin', F('pin','pin',{mode:'pwm'}), 'to', F('in','pct',{acc:'val', lit:'num', def:50}), '%'],
  ex:'Dims an LED or slows a motor by switching the pin on and off very fast (PWM). 0% is off, 100% is fully on.'});

def('led', {cat:'leds', shape:'stmt', label:['turn the built-in LED', F('drop','state',{opts:ONOFF})],
  ex:'Switches the little LED that is already on your board — no wiring needed. Flashie knows which pin it is for your board.'});
def('ledtoggle', {cat:'leds', shape:'stmt', label:['toggle the built-in LED'],
  ex:'Flips the built-in LED: on becomes off and off becomes on.'});
def('neosetup', {cat:'leds', shape:'stmt', label:['NeoPixel strip on pin', F('pin','pin',{mode:'out'}), 'with', F('in','n',{acc:'val', lit:'num', def:8}), 'LEDs'],
  ex:'Tells the program you have a strip of “smart” RGB LEDs (WS2812B / NeoPixels) on one pin, and how many LEDs it has. Use this once before the other NeoPixel blocks.'});
def('neoset', {cat:'leds', shape:'stmt', label:['set NeoPixel number', F('in','i',{acc:'val', lit:'num', def:1}), 'to', F('color','color')],
  ex:'Chooses the colour of one LED in the strip (number 1 is the first). Nothing changes until you use “show NeoPixel colours”.'});
def('neofill', {cat:'leds', shape:'stmt', label:['set all NeoPixels to', F('color','color')],
  ex:'Gives every LED in the strip the same colour. Use “show NeoPixel colours” afterwards to light them up.'});
def('neoshow', {cat:'leds', shape:'stmt', label:['show NeoPixel colours'],
  ex:'Sends all the colours you have chosen to the strip at once, so they all change together.'});
def('cydrgb', {cat:'leds', shape:'stmt', label:['set the CYD’s RGB LED to', F('color','color')], need:'cydrgb',
  ex:'Lights the small RGB LED on the back of the Cheap Yellow Display in any colour, by mixing red, green and blue.'});

def('btnpressed', {cat:'buttons', shape:'bool', label:['button on pin', F('pin','pin',{mode:'in_up', boot:true}), 'is pressed'],
  ex:'True while a push-button is held down. The button should connect the pin to GND (ground) — the built-in pull-up keeps it HIGH otherwise, so pressed means LOW.'});
def('waitbtn', {cat:'buttons', shape:'stmt', label:['wait until button on pin', F('pin','pin',{mode:'in_up', boot:true}), 'is pressed'],
  ex:'Pauses the program until somebody presses the button.'});

def('setvar', {cat:'vars', shape:'stmt', label:['set', F('var','var'), 'to', F('in','val',{acc:'val', lit:'auto', def:0})],
  ex:'A variable is a named box that remembers a value. This puts a new value in the box (replacing what was there).'});
def('changevar', {cat:'vars', shape:'stmt', label:['change', F('var','var'), 'by', F('in','n',{acc:'val', lit:'num', def:1})],
  ex:'Adds a number to a variable — handy for counting. Use a negative number to subtract.'});
def('varget', {cat:'vars', shape:'num', label:[F('var','var')],
  ex:'Gives you the value currently stored in the variable.'});

def('arith', {cat:'ops', shape:'num', label:[F('in','a',{acc:'val', lit:'num', def:1}), F('drop','op',{opts:[['+','+'],['-','−'],['*','×'],['/','÷']]}), F('in','b',{acc:'val', lit:'num', def:1})],
  ex:'Does a sum with two numbers: add, subtract, multiply (×) or divide (÷).'});
def('compare', {cat:'ops', shape:'bool', label:[F('in','a',{acc:'val', lit:'auto', def:''}), F('drop','op',{opts:[['==','='],['<','<'],['>','>'],['<=','≤'],['>=','≥'],['!=','≠']]}), F('in','b',{acc:'val', lit:'auto', def:50})],
  ex:'Compares two values and answers true or false — for example “is the temperature greater than 25?”.'});
def('andor', {cat:'ops', shape:'bool', label:[F('in','a',{acc:'bool'}), F('drop','op',{opts:[['and','and'],['or','or']]}), F('in','b',{acc:'bool'})],
  ex:'Combines two true/false questions. “and” needs both to be true; “or” needs at least one.'});
def('not', {cat:'ops', shape:'bool', label:['not', F('in','a',{acc:'bool'})],
  ex:'Flips true into false and false into true.'});
def('random', {cat:'ops', shape:'num', label:['random number from', F('in','a',{acc:'val', lit:'num', def:1}), 'to', F('in','b',{acc:'val', lit:'num', def:10})],
  ex:'Picks a whole number at random between the two values (both included) — like rolling a dice.'});
def('join', {cat:'ops', shape:'text', label:['join', F('in','a',{acc:'val', lit:'text', def:'Temp: '}), F('in','b',{acc:'val', lit:'text', def:'20'})],
  ex:'Sticks two pieces of text (or numbers) together into one piece of text.'});

def('print', {cat:'output', shape:'stmt', label:['print', F('in','val',{acc:'val', lit:'text', def:'Hello!'}), 'to the console'],
  ex:'Sends a message from the board back to your PC. It appears in Flashie’s console — the easiest way to see what your program is doing.'});
def('logfile', {cat:'output', shape:'stmt', label:['add line', F('in','val',{acc:'val', lit:'text', def:'reading'}), 'to file', F('text','name',{def:'log.txt'})], py:true,
  ex:'Saves a line of text into a file on the board’s own storage, so you can collect readings even when the PC is unplugged.'});

def('wifi', {cat:'wifi', shape:'stmt', label:['connect to Wi-Fi', F('text','ssid',{def:'MyNetwork'}), 'password', F('text','pw',{def:'password', secret:true})], need:'wifi', py:true,
  ex:'Joins your Wi-Fi network (2.4 GHz only — these boards can’t use 5 GHz). It waits up to 15 seconds and prints the board’s IP address.'});
def('wifiok', {cat:'wifi', shape:'bool', label:['Wi-Fi is connected'], need:'wifi', py:true,
  ex:'True when the board is connected to Wi-Fi.'});
def('ip', {cat:'wifi', shape:'text', label:['my IP address'], need:'wifi', py:true,
  ex:'The board’s address on your network, like 192.168.1.42. Type it into a browser to visit the board’s web page.'});
def('webserver', {cat:'wifi', shape:'c', label:['every time a web page is requested'], need:'wifi', py:true,
  ex:'Turns the board into a tiny website. It waits for a browser to visit, runs the blocks inside, then waits for the next visitor — forever.'});
def('webpath', {cat:'wifi', shape:'text', label:['requested page address'], need:'wifi', py:true,
  ex:'The part of the web address after the board’s IP — “/” for the home page, “/on” if someone visits http://IP/on.'});
def('webreply', {cat:'wifi', shape:'stmt', label:['reply with web page', F('in','html',{acc:'val', lit:'text', def:'<h1>Hello from my board!</h1>'})], need:'wifi', py:true,
  ex:'Sends a web page back to the browser that asked for it. You can write simple HTML, like <h1>Title</h1> or <a href="/on">ON</a> links.'});

def('dhtread', {cat:'sensors', shape:'stmt', label:['measure', F('drop','model',{opts:[['DHT22','DHT22'],['DHT11','DHT11']]}), 'sensor on pin', F('pin','pin',{mode:'in'})],
  ex:'Asks a DHT11 or DHT22 sensor for a new temperature and humidity reading. The DHT22 is more accurate. Wait at least 2 seconds between readings.'});
def('dhttemp', {cat:'sensors', shape:'num', label:['temperature (°C)'],
  ex:'The temperature from the last DHT measurement, in degrees Celsius.'});
def('dhthum', {cat:'sensors', shape:'num', label:['humidity (%)'],
  ex:'How damp the air was at the last DHT measurement, as a percentage.'});
def('i2csetup', {cat:'sensors', shape:'stmt', label:['start I²C on SDA', F('pin','sda',{mode:'i2c', i2c:0}), 'SCL', F('pin','scl',{mode:'i2c', i2c:1})],
  ex:'I²C is a two-wire connection (SDA = data, SCL = clock) that many sensors and small screens share. Start it once before using I²C blocks.'});
def('i2cscan', {cat:'sensors', shape:'stmt', label:['list I²C devices in the console'],
  ex:'Looks for every I²C device connected and prints their addresses — the first thing to try when wiring a new sensor.'});
def('i2creg', {cat:'sensors', shape:'num', label:['read register', F('num','reg',{def:0}), 'of I²C device', F('text','addr',{def:'0x76'})],
  ex:'Reads one byte (0–255) from a numbered “register” (a memory slot) inside an I²C device. The sensor’s datasheet tells you what each register means.'});

def('servo', {cat:'motion', shape:'stmt', label:['set servo on pin', F('pin','pin',{mode:'pwm'}), 'to', F('in','angle',{acc:'val', lit:'num', def:90}), '°'],
  ex:'Turns a hobby servo (like the SG90) to an angle between 0° and 180°. Power bigger servos from a separate 5 V supply, sharing GND.'});

def('oledsetup', {cat:'display', shape:'stmt', label:['start OLED screen 128 ×', F('drop','h',{opts:[['64','64'],['32','32']]}), 'on I²C'],
  ex:'Sets up a small black-and-white SSD1306 OLED screen (the common 0.96" kind) connected over I²C.'});
def('oledtext', {cat:'display', shape:'stmt', label:['OLED: write', F('in','val',{acc:'val', lit:'text', def:'Hello'}), 'on line', F('num','line',{def:1})],
  ex:'Writes text on one line of the OLED (line 1 is the top). Use “OLED: update screen” to make it appear.'});
def('oledclear', {cat:'display', shape:'stmt', label:['OLED: clear'],
  ex:'Wipes everything you have drawn on the OLED (after the next update).'});
def('oledshow', {cat:'display', shape:'stmt', label:['OLED: update screen'],
  ex:'Sends your drawing to the OLED. Drawing happens in memory first, so the screen changes in one smooth step.'});
def('cydsetup', {cat:'display', shape:'stmt', label:['start the CYD screen', F('drop','rot',{opts:[['1','landscape'],['3','landscape (flipped)'],['0','portrait'],['2','portrait (flipped)']]})], need:'cyd', py:true,
  ex:'Switches on the Cheap Yellow Display’s colour screen (and touch), using the exact pins for your CYD model.'});
def('cydfill', {cat:'display', shape:'stmt', label:['fill the screen with', F('color','color',{def:'#000000'})], need:'cyd', py:true,
  ex:'Paints the whole screen one colour — a quick way to clear it.'});
def('cydtext', {cat:'display', shape:'stmt', label:['write', F('in','val',{acc:'val', lit:'text', def:'Hello!'}), 'at x', F('in','x',{acc:'val', lit:'num', def:10}), 'y', F('in','y',{acc:'val', lit:'num', def:10}), 'size', F('num','size',{def:2}), F('color','color',{def:'#ffffff'})], need:'cyd', py:true,
  ex:'Draws text on the screen. x counts pixels from the left edge, y from the top. Size 1 is tiny (8 pixels tall), size 3 is easy to read.'});
def('cydrect', {cat:'display', shape:'stmt', label:['draw box at x', F('in','x',{acc:'val', lit:'num', def:0}), 'y', F('in','y',{acc:'val', lit:'num', def:0}), 'w', F('in','w',{acc:'val', lit:'num', def:40}), 'h', F('in','h',{acc:'val', lit:'num', def:40}), F('color','color',{def:'#00c800'})], need:'cyd', py:true,
  ex:'Draws a filled rectangle. x and y are its top-left corner; w and h are its width and height in pixels.'});
def('cydtouched', {cat:'display', shape:'bool', label:['screen is being touched'], need:'cyd', py:true,
  ex:'True while a finger (or stylus) is pressing the screen. It also remembers where, for the “touch x/y” blocks.'});
def('cydtx', {cat:'display', shape:'num', label:['touch x'], need:'cyd', py:true, ex:'How far from the left edge the last touch was, in pixels.'});
def('cydty', {cat:'display', shape:'num', label:['touch y'], need:'cyd', py:true, ex:'How far from the top the last touch was, in pixels.'});

const isReporter = t => ['num','bool','text'].includes(B[t].shape);
const isStmt = t => ['stmt','c','cc'].includes(B[t].shape);

/* ======================================================================
   Workspace model
   ====================================================================== */

let uid = 1;
const nid = () => 'b' + (uid++);

function make(type, f, ins){
  const d = B[type], b = {id: nid(), t: type, f: {}, in: {}, sub: [], sub2: []};
  const parts = [...d.label, ...(d.label2||[])];
  for(const p of parts){
    if(typeof p === 'string') continue;
    if(p.k === 'in'){ b.in[p.name] = null; if(p.lit) b.f[p.name] = p.def; }
    else if(p.k === 'drop') b.f[p.name] = p.opts[0][0];
    else if(p.k === 'color') b.f[p.name] = p.def || '#ff0000';
    else if(p.k === 'var') b.f[p.name] = (W.vars[0] || 'count');
    else if(p.k === 'pin') b.f[p.name] = null;
    else b.f[p.name] = p.def;
  }
  Object.assign(b.f, f||{});
  Object.assign(b.in, ins||{});
  return b;
}

let W = {stacks: [], vars: ['count']};
let C = {ctx: null, sel: null, teach: true, textMode: false, text: '', gen: null, hoverBid: null, view: 'flash', session: false, mpy: {}, busy: false};

function fillPins(b, ctx){
  const d = B[b.t];
  for(const p of [...d.label, ...(d.label2||[])]){
    if(p.k === 'pin' && (b.f[p.name] === null || b.f[p.name] === undefined)){
      if(p.i2c !== undefined) b.f[p.name] = ctx.i2c[p.i2c];
      else if(p.boot && ctx.boot !== null && ctx.boot !== undefined) b.f[p.name] = ctx.boot;
      else b.f[p.name] = freePin(ctx, p.mode==='in_up' ? 'in' : p.mode);
    }
  }
  for(const k in b.in) if(b.in[k]) fillPins(b.in[k], ctx);
  b.sub.forEach(x=>fillPins(x, ctx)); b.sub2.forEach(x=>fillPins(x, ctx));
}

/* ---------------------------------------------------------------- templates */

const TEMPLATES = {
  blink: {name:'Blink', about:'Flash the built-in LED on and off — the “hello world” of electronics.', build: ctx => {
    if(!ctx.led){
      const p = freePin(ctx,'out');
      return [make('start'), make('print',{val:'Blinking an LED on '+pinLabel(ctx,p)}), loop([make('pinwrite',{pin:p,state:'on'}), make('wait',{s:0.5}), make('pinwrite',{pin:p,state:'off'}), make('wait',{s:0.5})])];
    }
    return [make('start'), make('print',{val:'Blinking!'}), loop([make('led',{state:'on'}), make('wait',{s:0.5}), make('led',{state:'off'}), make('wait',{s:0.5})])];
  }},
  button: {name:'Button + LED', about:'Light the LED while a button is held down (uses the BOOT button where the board has one).', build: ctx => {
    const on = ctx.led ? make('led',{state:'on'}) : make('pinwrite',{pin:freePin(ctx,'out'),state:'on'});
    const off = ctx.led ? make('led',{state:'off'}) : make('pinwrite',{pin:freePin(ctx,'out'),state:'off'});
    const btnPin = ctx.boot!==null && ctx.boot!==undefined ? ctx.boot : freePin(ctx,'in');
    const ie = make('ifelse', {}, {cond: make('btnpressed',{pin:btnPin})}); ie.sub = [on, make('print',{val:'Button pressed'})]; ie.sub2 = [off];
    return [make('start'), make('print',{val:'Press the button!'}), loop([ie, make('waitms',{ms:50})])];
  }},
  web: {name:'Wi-Fi web server', about:'Join Wi-Fi and serve a web page with ON/OFF links that control the LED.', py:true, need:'wifi', build: ctx => {
    const ledOn = ctx.led ? make('led',{state:'on'}) : make('pinwrite',{pin:freePin(ctx,'out'),state:'on'});
    const ledOff = ctx.led ? make('led',{state:'off'}) : make('pinwrite',{pin:freePin(ctx,'out'),state:'off'});
    const i1 = make('if', {}, {cond: make('compare',{op:'==', a:'', b:'/on'}, {a: make('webpath')})}); i1.sub = [ledOn];
    const i2 = make('if', {}, {cond: make('compare',{op:'==', a:'', b:'/off'}, {a: make('webpath')})}); i2.sub = [ledOff];
    const ws = make('webserver'); ws.sub = [i1, i2, make('webreply',{html:'<h1>My board</h1><p><a href="/on">LED ON</a> | <a href="/off">LED OFF</a></p>'})];
    return [make('start'), make('wifi'), make('print',{}, {val: make('join',{a:'Open this in a browser: http://'},{b: make('ip')})}), ws];
  }},
  cyd: {name:'CYD hello + touch', about:'Say hello on the Cheap Yellow Display and draw wherever you touch.', py:true, need:'cyd', build: ctx => {
    const touchIf = make('if', {}, {cond: make('cydtouched')});
    touchIf.sub = [make('cydrect',{w:6,h:6,color:'#ffd000'},{x: make('cydtx'), y: make('cydty')})];
    return [make('start'), make('cydsetup',{rot:'1'}), make('cydfill',{color:'#000000'}), make('cydtext',{val:'Hello from Flashie!', x:10, y:10, size:2, color:'#ffffff'}),
      make('cydtext',{val:'Touch the screen to draw', x:10, y:40, size:1, color:'#00e0e0'}), loop([touchIf, make('waitms',{ms:10})])];
  }},
  logger: {name:'Temperature logger', about:'Read a DHT22 every 2 seconds, print it and (in MicroPython) save it to a file.', build: ctx => {
    const p = freePin(ctx,'in');
    const body = [make('dhtread',{model:'DHT22', pin:p}),
      make('print',{}, {val: make('join',{a:'Temperature °C: '},{b: make('dhttemp')})}),
      make('print',{}, {val: make('join',{a:'Humidity %: '},{b: make('dhthum')})})];
    if(ctx.lang === 'py') body.push(make('logfile',{name:'log.txt'}, {val: make('join',{a:''},{a: make('dhttemp'), b: make('dhthum')})}));
    body.push(make('wait',{s:2}));
    return [make('start'), make('print',{val:'Temperature logger starting'}), loop(body)];
  }},
};
function loop(sub){ const f = make('forever'); f.sub = sub; return f; }

function loadTemplate(key){
  const ctx = creatorCtx(), T = TEMPLATES[key];
  W = {stacks: [{x: 40, y: 30, blocks: T.build(ctx)}], vars: ['count']};
  for(const s of W.stacks) s.blocks.forEach(b=>fillPins(b, ctx));
  C.textMode = false; C.sel = null;
  $c('#cBody').classList.remove('textmode');
  $c('#cMode').querySelectorAll('button').forEach(b => b.classList.toggle('on', b.dataset.m === 'blocks'));
  renderAll();
  let msg = `Loaded “${T.name}”. ${T.about}`;
  if(T.need==='cyd' && !ctx.cyd) msg += ' Pick your CYD model on the Flash tab so the screen pins are right.';
  if(T.need==='wifi') msg += ' Put your Wi-Fi name and password in the “connect to Wi-Fi” block.';
  if(T.py && ctx.lang==='cpp') msg += ' This one needs MicroPython, which your Arduino board can’t run.';
  teachMsg('Template', msg);
}

/* ======================================================================
   Code generation (MicroPython + Arduino C++)
   ====================================================================== */

const pyStr = s => JSON.stringify(String(s));
const cStr = s => JSON.stringify(String(s));
const hexRGB = c => { const n = parseInt(String(c).slice(1), 16) || 0; return [(n>>16)&255, (n>>8)&255, n&255]; };
const rgb565 = c => { const [r,g,b] = hexRGB(c); return '0x' + (((r&0xF8)<<8)|((g&0xFC)<<3)|(b>>3)).toString(16).toUpperCase().padStart(4,'0'); };
const colorName = c => (COLORS.find(x=>x[0]===c)||[0,c])[1];
const isNum = v => v !== '' && v !== null && !isNaN(Number(v)) && /^\s*-?\d*\.?\d+\s*$/.test(String(v));
const pyVar = n => { let s = String(n).trim().toLowerCase().replace(/[^a-z0-9_]+/g,'_').replace(/^_+|_+$/g,'') || 'v'; if(/^\d/.test(s) || PY_KW.has(s)) s = 'v_'+s; return s; };
const PY_KW = new Set('and as assert break class continue def del elif else except finally for from global if import in is lambda nonlocal not or pass raise return try while with yield True False None print time pin led'.split(' '));

function generate(ctx){
  const lang = ctx.lang || 'py';
  const G = {ctx, lang, imports: new Map(), setup: [], setupKeys: new Set(), lines: [], ind: 0, files: {}, warns: [], errs: [], libs: new Set(),
             includes: new Map(), globals: [], loopCounter: 0, used: {}, blockLines: {}};
  G.emit = (t, ex, bid) => G.lines.push({t: (lang === 'cpp' ? '  ' : '    ').repeat(G.ind) + t, ex, bid, ind: G.ind});
  G.imp = (line, ex) => { if(!G.imports.has(line)) G.imports.set(line, ex); };
  G.once = (key, lines) => { if(G.setupKeys.has(key)) return; G.setupKeys.add(key); for(const [t, ex, bid] of lines) G.setup.push({t, ex, bid}); };
  G.warn = (bid, text) => { if(!G.warns.some(w=>w.bid===bid && w.text===text)) G.warns.push({bid, text}); };
  G.err = (bid, text) => { if(!G.errs.some(w=>w.bid===bid && w.text===text)) G.errs.push({bid, text}); };

  const start = W.stacks.find(s => s.blocks[0] && s.blocks[0].t === 'start');
  if(W.stacks.filter(s => s.blocks[0] && s.blocks[0].t==='start').length > 1) G.warns.push({bid:null, text:'There is more than one “when the board starts” block — only the first one is used.'});
  W.stacks.forEach(s => { if(s.blocks[0] && s.blocks[0].t !== 'start') G.warns.push({bid: s.blocks[0].id, text:'Loose blocks (not attached under “when the board starts”) are ignored.'}); });

  if(lang === 'py') genPy(G, start); else genCpp(G, start);
  return G;
}

/* ---- shared checks ---- */
function checkNeeds(G, b){
  const d = B[b.t], ctx = G.ctx;
  if(d.py && G.lang === 'cpp') { G.err(b.id, `“${plainLabel(b)}” needs MicroPython (Wi-Fi, files and CYD screens aren’t available on ${ctx.boardName}).`); return false; }
  if(d.need === 'wifi' && !ctx.hasWifi){ G.err(b.id, `${ctx.boardName} has no Wi-Fi.`); return false; }
  if(d.need === 'cyd' && !ctx.cyd){ G.err(b.id, 'Screen blocks need a Cheap Yellow Display — pick your CYD model on the Flash tab.'); return false; }
  if(d.need === 'cyd' && ctx.cyd && !/^SPI/.test(ctx.cyd.displayBus) ){ G.err(b.id, `The ${ctx.cyd.model} screen is connected over ${ctx.cyd.displayBus}. Flashie’s MicroPython screen driver only handles SPI screens (the 2.4", 2.8", 3.2" and 3.5" CYDs).`); return false; }
  if(d.need === 'cyd' && ctx.cyd && /GC9A01/.test(ctx.cyd.display)){ G.err(b.id, 'The round GC9A01 screen isn’t supported by Flashie’s driver yet.'); return false; }
  if(d.need === 'cydrgb' && !(ctx.led && ctx.led.kind==='rgb')){ G.err(b.id, 'This board has no CYD RGB LED. Use the NeoPixel or built-in LED blocks instead.'); return false; }
  return true;
}
function plainLabel(b){
  return B[b.t].label.map(p => typeof p === 'string' ? p : p.k==='in' ? '…' : p.k==='pin' ? 'pin' : '').join(' ').replace(/\s+/g,' ').trim();
}
function pinCheck(G, b, pin, mode){
  const w = pinWarn(G.ctx, pin, mode);
  if(w) G.warn(b.id, w);
  const key = String(pin), role = mode === 'in' || mode === 'in_up' ? 'in' : mode;
  G.used[key] = G.used[key] || new Set(); G.used[key].add(role);
  if(G.used[key].size > 1 && !(G.used[key].size===2 && G.used[key].has('in') && G.used[key].has('adc')))
    G.warn(b.id, `${pinLabel(G.ctx, pin)} is used for more than one job (${[...G.used[key]].join(' and ')}) — each pin can only do one thing at a time.`);
}

/* ---- MicroPython ---- */
function pyPin(G, b, pin, mode, init){
  pinCheck(G, b, pin, mode);
  const P = typeof pin === 'string' && pin !== 'LED' ? pyStr(pin) : pin === 'LED' ? '"LED"' : pin;
  const id = String(pin).replace(/\W/g,'');
  G.imp('from machine import Pin', 'Brings in “Pin”, MicroPython’s tool for controlling the board’s pins.');
  if(mode === 'out'){ const v = 'pin'+id; G.once('out'+id, [[`${v} = Pin(${P}, Pin.OUT${init ? ', value=1' : ''})`, `Gives ${pinLabel(G.ctx,pin)} the name “${v}” and makes it an output, so the program can switch it on and off.${init ? ' “value=1” starts it HIGH, which keeps an active-low LED off until we turn it on.' : ''}`, b.id]]); return v; }
  if(mode === 'in_up' || mode === 'in' || mode === 'in_down'){
    const pull = mode==='in_up' ? ', Pin.PULL_UP' : mode==='in_down' ? ', Pin.PULL_DOWN' : '';
    const v = (mode==='in_up' ? 'button' : 'input') + id;
    G.once(mode+id, [[`${v} = Pin(${P}, Pin.IN${pull})`, `Makes ${pinLabel(G.ctx,pin)} an input called “${v}”${mode==='in_up' ? ', with the internal pull-up resistor switched on so it reads HIGH until a button connects it to GND' : mode==='in_down' ? ', with the pull-down resistor on so it reads LOW until something connects it to 3.3 V' : ''}.`, b.id]]);
    return v;
  }
  if(mode === 'adc'){ G.imp('from machine import ADC', 'Brings in “ADC”, the analog-to-digital converter that measures voltages.');
    const v = 'analog'+id; G.once('adc'+id, [[`${v} = ADC(Pin(${P}))`, `Sets up ${pinLabel(G.ctx,pin)} to measure voltages; it is called “${v}”.`, b.id]]);
    if(G.ctx.chip !== 'ESP8266' && G.ctx.fam === 'esp') G.once('atten'+id, [[`${v}.atten(ADC.ATTN_11DB)`, 'Lets the ESP32 measure the full 0–3.3 V range (without this it would only see up to about 1 V).', b.id]]);
    return v; }
  if(mode === 'pwm' || mode === 'servo'){ G.imp('from machine import PWM', 'Brings in “PWM” — rapid on/off switching used for dimming LEDs and driving servos.');
    const f = mode === 'servo' ? 50 : 1000, v = (mode==='servo' ? 'servo' : 'pwm') + id;
    G.once(mode+id, [[`${v} = PWM(Pin(${P}), freq=${f})`, mode==='servo' ? `Starts a 50 Hz pulse signal on ${pinLabel(G.ctx,pin)} — the rhythm hobby servos expect. It is called “${v}”.` : `Starts fast on/off switching (1000 times a second) on ${pinLabel(G.ctx,pin)}, called “${v}”.`, b.id]]);
    return v; }
}

function pyExpr(G, b, name, want){
  const r = b.in[name];
  if(r){
    if(!checkNeeds(G, r)) return want==='bool' ? 'False' : '0';
    return pyRep(G, r);
  }
  if(want === 'bool'){ G.warn(b.id, 'An empty condition slot counts as “false”. Drag a green or blue hexagon block into it.'); return 'False'; }
  const part = [...B[b.t].label, ...(B[b.t].label2||[])].find(p => p.name === name);
  const v = b.f[name];
  if(part && part.lit === 'num') return isNum(v) ? String(Number(v)) : '0';
  if(part && part.lit === 'auto') return isNum(v) ? String(Number(v)) : pyStr(v ?? '');
  return pyStr(v ?? '');
}

function pyLed(G, b){
  const L = G.ctx.led;
  if(!L){ G.warn(b.id, `Flashie doesn’t know where ${G.ctx.boardName}’s built-in LED is — using GPIO 2. Use “set pin” for an LED you wired yourself.`); return {kind:'pin', pin:2, low:false}; }
  return L;
}

function pyRep(G, r){
  const ctx = G.ctx, f = r.f;
  switch(r.t){
    case 'pinread': { const v = pyPin(G, r, f.pin, f.pull==='up' ? 'in_up' : f.pull==='down' ? 'in_down' : 'in'); return `${v}.value() == 1`; }
    case 'analogread': { const v = pyPin(G, r, f.pin, 'adc'); return `${v}.read_u16() * 100 // 65535`; }
    case 'btnpressed': { const v = pyPin(G, r, f.pin, 'in_up'); return `${v}.value() == 0`; }
    case 'varget': return pyVar(f.var);
    case 'arith': return `(${pyExpr(G, r, 'a')} ${f.op} ${pyExpr(G, r, 'b')})`;
    case 'compare': return `${pyExpr(G, r, 'a')} ${f.op} ${pyExpr(G, r, 'b')}`;
    case 'andor': return `(${pyExpr(G, r, 'a', 'bool')}) ${f.op} (${pyExpr(G, r, 'b', 'bool')})`;
    case 'not': return `not (${pyExpr(G, r, 'a', 'bool')})`;
    case 'random': G.imp('import random', 'Brings in “random”, for picking random numbers.'); return `random.randint(int(${pyExpr(G, r, 'a')}), int(${pyExpr(G, r, 'b')}))`;
    case 'join': { const sx = e => /^"/.test(e) ? e : `str(${e})`; return `${sx(pyExpr(G, r, 'a'))} + ${sx(pyExpr(G, r, 'b'))}`; }
    case 'wifiok': pyWlan(G, r); return 'wlan.isconnected()';
    case 'ip': pyWlan(G, r); return 'wlan.ifconfig()[0]';
    case 'webpath': if(!G.inWeb) G.warn(r.id, '“requested page address” only means something inside the “every time a web page is requested” block.'); return 'path';
    case 'dhttemp': if(!G.dht) G.warn(r.id, 'Use a “measure DHT sensor” block first.'); return 'sensor.temperature()';
    case 'dhthum': if(!G.dht) G.warn(r.id, 'Use a “measure DHT sensor” block first.'); return 'sensor.humidity()';
    case 'i2creg': { pyI2C(G, r); const a = String(f.addr).trim(); return `i2c.readfrom_mem(${/^(0x[0-9a-f]+|\d+)$/i.test(a) ? a : '0x76'}, ${isNum(f.reg) ? Number(f.reg) : 0}, 1)[0]`; }
    case 'cydtouched': pyScreen(G, r); return 'screen.touched()';
    case 'cydtx': pyScreen(G, r); return 'screen.tx';
    case 'cydty': pyScreen(G, r); return 'screen.ty';
  }
  return '0';
}

function pyWlan(G, b){
  G.imp('import network', 'Brings in “network”, MicroPython’s Wi-Fi tools.');
  G.once('wlan', [['wlan = network.WLAN(network.STA_IF)', 'Gets the board’s Wi-Fi radio in “station” mode — that means joining an existing network, like a phone does.', b.id]]);
}
function pyI2C(G, b, sda, scl){
  if(G.setupKeys.has('i2c')) return;
  sda = sda ?? G.ctx.i2c[0]; scl = scl ?? G.ctx.i2c[1];
  if(!G.i2cExplicit) G.warn(b.id, `No “start I²C” block — using ${pinLabel(G.ctx, sda)} (SDA) and ${pinLabel(G.ctx, scl)} (SCL).`);
  pinCheck(G, b, sda, 'i2c'); pinCheck(G, b, scl, 'i2c');
  G.imp('from machine import Pin', 'Brings in “Pin”, MicroPython’s tool for controlling the board’s pins.');
  G.imp('from machine import SoftI2C', 'Brings in “SoftI2C”, which talks to I²C sensors and screens on any two pins.');
  G.once('i2c', [[`i2c = SoftI2C(sda=Pin(${sda}), scl=Pin(${scl}), freq=400000)`, `Starts the I²C bus on ${pinLabel(G.ctx, sda)} (data) and ${pinLabel(G.ctx, scl)} (clock) at 400 kHz, and calls it “i2c”.`, b.id]]);
}
function pyScreen(G, b, rot){
  if(G.setupKeys.has('screen')) return;
  const c = G.ctx.cyd;
  if(!c) return;
  if(rot === undefined) G.warn(b.id, 'No “start the CYD screen” block — starting it in landscape automatically.');
  const disp = c.groups.find(g => /^Display/.test(g.name)), gp = f => (disp.pins.find(p=>p.f===f)||{}).g;
  const [w, h] = c.res.split('×').map(Number);
  const tg = c.groups.find(g => /^Touch/.test(g.name));
  let touch = 'None';
  if(tg){
    const tp = f => (tg.pins.find(p=>p.f===f)||{}).g;
    if(/XPT2046/.test(tg.name)){
      touch = /shares/.test(tg.bus||'') ? `("xpt", {"cs": ${tp('CS')}, "irq": ${tp('IRQ') ?? -1}, "shared": True})`
        : `("xpt", {"sck": ${tp('SCK')}, "mosi": ${tp('MOSI')}, "miso": ${tp('MISO')}, "cs": ${tp('CS')}, "irq": ${tp('IRQ') ?? -1}})`;
    } else if(/GT911/.test(tg.name)) touch = `("gt911", {"sda": ${tp('SDA')}, "scl": ${tp('SCL')}})`;
    else if(/CST8/.test(tg.name)) touch = `("cst", {"sda": ${tp('SDA')}, "scl": ${tp('SCL')}, "rst": ${tp('RST') ?? -1}})`;
  }
  const bgr = c.colorOrder !== 'RGB' ? 'True' : 'False';
  const inv = /ST7789/.test(c.display) && c.family === '2432S028' ? 'True' : 'False';
  const xoff = c.id === '1732s019c' ? 35 : 0;
  const args = [`${w}, ${h}`, `sck=${gp('SCK')}`, `mosi=${gp('MOSI')}`, `cs=${gp('CS')}`, `dc=${gp('DC')}`, `bl=${gp('Backlight') ?? -1}`];
  if(gp('MISO') !== undefined) args.push(`miso=${gp('MISO')}`);
  if(gp('RST') !== undefined) args.push(`rst=${gp('RST')}`);
  args.push(`rot=${rot ?? 1}`, `bgr=${bgr}`, `invert=${inv}`);
  if(xoff) args.push(`xoff=${xoff}`);
  args.push(`touch=${touch}`);
  G.imp('import flashie_cyd', 'Brings in Flashie’s screen driver. Flashie copies the file “flashie_cyd.py” onto the board for you.');
  G.files['flashie_cyd.py'] = true;
  G.once('screen', [[`screen = flashie_cyd.Screen(${args.join(', ')})`, `Starts the ${c.size} ${c.display} screen of your ${c.model} using its real wiring (clock on GPIO ${gp('SCK')}, data on GPIO ${gp('MOSI')}, backlight on GPIO ${gp('Backlight')})${tg ? ' and its ' + tg.name.split('— ')[1] + ' touch panel' : ''}.`, b.id]]);
}

function genPy(G, start){
  const ctx = G.ctx;
  G.imp('import time', 'Brings in “time”, which gives us waits and pauses.');
  const vars = new Set();
  const walk = list => list.forEach(b => { if(b.f && b.f.var !== undefined) vars.add(b.f.var); Object.values(b.in).forEach(x => x && walk([x])); walk(b.sub); walk(b.sub2); });
  if(start) walk(start.blocks);
  for(const v of vars) G.once('var'+v, [[`${pyVar(v)} = 0`, `Creates the variable “${v}” and starts it at 0.`, null]]);

  const body = (list, ownerBid) => {
    const before = G.lines.length;
    G.ind++;
    list.forEach(s => stmt(s));
    if(G.lines.length === before) G.emit('pass', 'This part is empty. Python needs at least one line here, and “pass” means “do nothing”.', ownerBid);
    G.ind--;
  };
  const stmt = b => {
    if(!checkNeeds(G, b)) { G.emit(`pass  # (skipped: ${plainLabel(b)})`, 'This block can’t be turned into code for this board — see the warnings below the blocks. “pass” means “do nothing”.', b.id); return; }
    const f = b.f, E = (t, ex) => G.emit(t, ex, b.id);
    switch(b.t){
      case 'forever': E('while True:', 'Starts a loop that never ends: the indented lines below repeat forever.'); body(b.sub, b.id); break;
      case 'repeat': { const v = '_' + 'ijkmn'[Math.min(G.loopCounter++, 4)]; E(`for ${v} in range(int(${pyExpr(G, b, 'n')})):`, `Repeats the indented lines ${isNum(f.n) && !b.in.n ? f.n : 'that many'} times. Programmers use “range” to count.`); body(b.sub, b.id); G.loopCounter--; break; }
      case 'while': E(`while ${pyExpr(G, b, 'cond', 'bool')}:`, 'Keeps repeating the indented lines while the condition stays true.'); body(b.sub, b.id); break;
      case 'if': E(`if ${pyExpr(G, b, 'cond', 'bool')}:`, 'Checks the condition — the indented lines only run when it is true.'); body(b.sub, b.id); break;
      case 'ifelse': E(`if ${pyExpr(G, b, 'cond', 'bool')}:`, 'Checks the condition — the first indented group runs when it is true…'); body(b.sub, b.id);
        E('else:', '…and this group runs when it is false.'); body(b.sub2, b.id); break;
      case 'wait': E(`time.sleep(${pyExpr(G, b, 's')})`, `Pauses for ${isNum(f.s) && !b.in.s ? f.s : 'that many'} second(s).`); break;
      case 'waitms': E(b.in.ms ? `time.sleep_ms(int(${pyExpr(G, b, 'ms')}))` : `time.sleep_ms(${pyExpr(G, b, 'ms')})`, `Pauses for ${isNum(f.ms) && !b.in.ms ? f.ms : 'that many'} milliseconds (thousandths of a second).`); break;
      case 'pinwrite': { const v = pyPin(G, b, f.pin, 'out'); E(`${v}.value(${f.state==='on'?1:0})`, `Switches ${pinLabel(ctx, f.pin)} ${f.state} — ${f.state==='on' ? '1 means 3.3 V comes out' : '0 means 0 V'}.`); break; }
      case 'pintoggle': { const v = pyPin(G, b, f.pin, 'out'); E(`${v}.value(not ${v}.value())`, `Reads whether ${pinLabel(ctx, f.pin)} is on, and sets it to the opposite.`); break; }
      case 'pwm': { const v = pyPin(G, b, f.pin, 'pwm'); E(`${v}.duty_u16(int(${pyExpr(G, b, 'pct')} * 65535 / 100))`, 'Sets how much of each tiny on/off cycle the pin spends ON. MicroPython counts this from 0 to 65535, so we scale the percentage.'); break; }
      case 'led': case 'ledtoggle': {
        const L = pyLed(G, b), on = f.state === 'on';
        if(L.kind === 'neo'){
          G.imp('import neopixel', 'Brings in “neopixel”, for colour (RGB) LEDs.');
          G.once('ledneo', [[`led = neopixel.NeoPixel(Pin(${L.pin}), 1)`, `This board’s built-in LED is a tiny colour NeoPixel on GPIO ${L.pin}. We call it “led”.`, b.id]]);
          G.imp('from machine import Pin', 'Brings in “Pin”, MicroPython’s tool for controlling the board’s pins.');
          if(b.t === 'ledtoggle'){ E('led[0] = (0, 0, 0) if led[0] != (0, 0, 0) else (0, 40, 0)', 'If the LED is lit, turn it off; otherwise make it green.'); }
          else E(`led[0] = ${on ? '(0, 40, 0)' : '(0, 0, 0)'}`, on ? 'Sets the LED to a soft green (red, green, blue values from 0 to 255).' : 'Sets the LED colour to (0, 0, 0) — all off.');
          E('led.write()', 'Sends the colour to the LED.');
        } else if(L.kind === 'rgb'){
          const v = pyPin(G, b, L.r, 'out', 1);
          if(b.t === 'ledtoggle') E(`${v}.value(not ${v}.value())`, 'Flips the CYD’s red LED.');
          else E(`${v}.value(${on ? 0 : 1})`, `The CYD’s LED is “active-low”: 0 turns it on, 1 turns it off. This switches the red part ${f.state}.`);
        } else {
          const v = L.kind==='name' ? (G.imp('from machine import Pin','Brings in “Pin”, MicroPython’s tool for controlling the board’s pins.'), G.once('ledname', [['led = Pin("LED", Pin.OUT)', 'On a Pico W the LED is wired through the Wi-Fi chip, so MicroPython gives it the special name "LED".', b.id]]), 'led')
            : pyPin(G, b, L.pin, 'out', L.low ? 1 : 0);
          if(b.t === 'ledtoggle') E(`${v}.value(not ${v}.value())`, 'Reads whether the LED is on and sets it to the opposite.');
          else E(`${v}.value(${L.low ? (on?0:1) : (on?1:0)})`, L.low ? `This board’s LED is “active-low” — it lights when the pin is 0 — so ${on?'0 turns it on':'1 turns it off'}.` : `Turns the built-in LED ${f.state}.`);
        }
        break; }
      case 'neosetup': { G.imp('import neopixel', 'Brings in “neopixel”, for colour (RGB) LEDs.'); pinCheck(G, b, f.pin, 'out');
        G.imp('from machine import Pin', 'Brings in “Pin”, MicroPython’s tool for controlling the board’s pins.');
        G.once('strip', [[`strip = neopixel.NeoPixel(Pin(${f.pin}), int(${pyExpr(G, b, 'n')}))`, `Sets up the NeoPixel strip on ${pinLabel(ctx, f.pin)} and calls it “strip”.`, b.id]]); G.strip = true; break; }
      case 'neoset': if(!G.strip) G.warn(b.id, 'Add a “NeoPixel strip on pin” block above this one.');
        E(`strip[int(${pyExpr(G, b, 'i')}) - 1] = (${hexRGB(f.color).join(', ')})`, `Sets one LED to ${colorName(f.color)}. We subtract 1 because Python counts from 0, but people count from 1.`); break;
      case 'neofill': if(!G.strip) G.warn(b.id, 'Add a “NeoPixel strip on pin” block above this one.');
        E(`strip.fill((${hexRGB(f.color).join(', ')}))`, `Sets every LED to ${colorName(f.color)} (red, green, blue from 0 to 255).`); break;
      case 'neoshow': if(!G.strip) G.warn(b.id, 'Add a “NeoPixel strip on pin” block above this one.'); E('strip.write()', 'Sends all the chosen colours to the strip at once.'); break;
      case 'cydrgb': { G.imp('from machine import PWM', 'Brings in “PWM” — rapid on/off switching used for dimming LEDs.');
        G.imp('from machine import Pin', 'Brings in “Pin”, MicroPython’s tool for controlling the board’s pins.');
        const L = ctx.led;
        G.once('cydrgb', [[`rgb = [PWM(Pin(${L.r}), freq=1000), PWM(Pin(${L.g}), freq=1000), PWM(Pin(${L.b}), freq=1000)]`, `The CYD’s RGB LED has red, green and blue parts on GPIO ${L.r}, ${L.g} and ${L.b}. PWM lets us mix any colour.`, b.id]]);
        const [r, g, bb] = hexRGB(f.color);
        E(`for part, level in zip(rgb, (${r}, ${g}, ${bb})):`, `Goes through red, green and blue, pairing each with how bright it should be for ${colorName(f.color)}.`);
        G.ind++; E('part.duty_u16(65535 - level * 257)', 'The LED is active-low (0 = fully on), so we flip the brightness: 255 becomes 0, 0 becomes 65535.'); G.ind--; break; }
      case 'btnpressed': break;
      case 'waitbtn': { const v = pyPin(G, b, f.pin, 'in_up'); E(`while ${v}.value() == 1:`, 'While the button is NOT pressed (the pin reads 1, HIGH)…'); G.ind++; E('time.sleep_ms(10)', '…check again every 10 milliseconds.'); G.ind--; break; }
      case 'setvar': E(`${pyVar(f.var)} = ${pyExpr(G, b, 'val')}`, `Stores a new value in the variable “${f.var}”.`); break;
      case 'changevar': E(`${pyVar(f.var)} += ${pyExpr(G, b, 'n')}`, `Adds to the variable “${f.var}”. “+=” means “add this to what’s already there”.`); break;
      case 'print': E(`print(${pyExpr(G, b, 'val')})`, 'Sends this to your PC — it shows up in Flashie’s console.'); break;
      case 'logfile': { const name = String(f.name || 'log.txt').replace(/[^\w.\-]/g,'_');
        E(`with open(${pyStr(name)}, "a") as f:`, `Opens the file “${name}” on the board for adding (“a” = append). “with” closes it again automatically.`);
        G.ind++; E(`f.write(str(${pyExpr(G, b, 'val')}) + "\\n")`, 'Writes the value as text, then “\\n” — a new-line character so the next entry starts on its own line.'); G.ind--; break; }
      case 'wifi': pyWlan(G, b);
        E('wlan.active(True)', 'Switches the Wi-Fi radio on.');
        E(`wlan.connect(${pyStr(f.ssid)}, ${pyStr(f.pw)})`, 'Starts joining your network with its name and password. (Anyone who sees this code sees your password — don’t share it.)');
        E('print("Connecting to Wi-Fi", end="")', 'Prints a message without starting a new line, so the dots that follow appear after it.');
        E('for _ in range(30):', 'Checks up to 30 times…'); G.ind++;
        E('if wlan.isconnected():', '…and if we are connected…'); G.ind++; E('break', '…stops waiting early.'); G.ind--;
        E('print(".", end="")', 'Prints a dot so you can see it is still trying.'); E('time.sleep(0.5)', 'Waits half a second between checks (30 × 0.5 s = 15 s in total).'); G.ind--;
        E('print()', 'Finishes the line of dots.');
        E('print("Wi-Fi connected:", wlan.isconnected(), "- IP address:", wlan.ifconfig()[0])', 'Reports whether it worked and the board’s IP address (its address on your network).'); break;
      case 'webserver': pyWlan(G, b);
        G.imp('import socket', 'Brings in “socket”, the basic building block for network connections like web servers.');
        E('server = socket.socket()', 'Creates a network “socket” — like a phone line the board can answer.');
        E('server.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)', 'Lets the program restart without waiting for the old connection to time out.');
        E('server.bind(("0.0.0.0", 80))', 'Answers on port 80, the standard port for web pages, on any of the board’s addresses.');
        E('server.listen(2)', 'Starts listening for browsers, with room for 2 waiting in line.');
        E('print("Web server running at http://" + wlan.ifconfig()[0])', 'Prints the address to type into your browser.');
        E('while True:', 'Serves web pages forever:'); G.ind++;
        E('client, addr = server.accept()', 'Waits here until a browser connects; “client” is the connection to that browser.');
        E('request = client.recv(1024).decode()', 'Reads the browser’s request (up to 1024 bytes) and turns it into text.');
        E('path = request.split(" ")[1] if " " in request else "/"', 'Pulls out the page address the browser asked for, like “/” or “/on”.');
        G.inWeb = true; const before = G.lines.length; list2(b.sub); G.inWeb = false;
        if(G.lines.length === before) E('pass', 'Nothing inside yet.');
        E('client.close()', 'Hangs up on this browser, ready for the next one.'); G.ind--; break;
      case 'webreply': if(!G.inWeb) G.err(b.id, '“reply with web page” only works inside “every time a web page is requested”.');
        E('client.send("HTTP/1.0 200 OK\\r\\nContent-Type: text/html\\r\\n\\r\\n")', 'Tells the browser “OK, here comes a web page” (this is the standard HTTP header).');
        E(`client.send(str(${pyExpr(G, b, 'html')}))`, 'Sends the page itself.'); break;
      case 'dhtread': { G.imp('import dht', 'Brings in “dht”, the built-in driver for DHT11/DHT22 temperature sensors.'); pinCheck(G, b, f.pin, 'in');
        G.imp('from machine import Pin', 'Brings in “Pin”, MicroPython’s tool for controlling the board’s pins.');
        G.once('dht', [[`sensor = dht.${f.model}(Pin(${f.pin}))`, `Sets up a ${f.model} sensor on ${pinLabel(ctx, f.pin)} and calls it “sensor”.`, b.id]]); G.dht = true;
        E('try:', 'Sensors sometimes fail to answer, so we “try” and catch problems instead of crashing:'); G.ind++;
        E('sensor.measure()', `Asks the ${f.model} for a fresh reading.`); G.ind--;
        E('except OSError:', 'If the sensor didn’t answer (loose wire, read too soon)…'); G.ind++;
        E('print("Sensor didn\'t answer - check the wiring")', '…print a helpful message and keep going with the last values.'); G.ind--; break; }
      case 'i2csetup': G.i2cExplicit = true; pyI2C(G, b, f.sda, f.scl); break;
      case 'i2cscan': pyI2C(G, b); E('print("I2C devices found:", [hex(a) for a in i2c.scan()])', 'Checks every I²C address and prints the ones that answered, in hex (like 0x3c for most OLEDs).'); break;
      case 'servo': { const v = pyPin(G, b, f.pin, 'servo'); E(`${v}.duty_u16(int(1638 + max(0, min(180, ${pyExpr(G, b, 'angle')})) * 6553 / 180))`, 'Turns the angle into a pulse between 0.5 ms (0°) and 2.5 ms (180°). “max/min” stop the angle going outside 0–180.'); break; }
      case 'oledsetup': pyI2C(G, b); G.imp('import flashie_oled', 'Brings in Flashie’s OLED driver (Flashie copies “flashie_oled.py” onto the board).'); G.files['flashie_oled.py'] = true;
        G.once('oled', [[`oled = flashie_oled.OLED(i2c, 128, ${f.h})`, `Starts a 128×${f.h} OLED on the I²C bus and calls it “oled”.`, b.id]]); G.oled = true; break;
      case 'oledtext': if(!G.oled) G.warn(b.id, 'Add a “start OLED screen” block above this one.');
        E(`oled.fill_rect(0, ${(Math.max(1, Number(f.line)||1)-1)*10}, 128, 10, 0)`, 'Clears that line first so old text doesn’t show through.');
        E(`oled.text(str(${pyExpr(G, b, 'val')}), 0, ${(Math.max(1, Number(f.line)||1)-1)*10})`, `Draws the text at the start of line ${f.line} (each line is 10 pixels tall).`); break;
      case 'oledclear': if(!G.oled) G.warn(b.id, 'Add a “start OLED screen” block above this one.'); E('oled.fill(0)', 'Fills the screen memory with black (0).'); break;
      case 'oledshow': if(!G.oled) G.warn(b.id, 'Add a “start OLED screen” block above this one.'); E('oled.show()', 'Sends the picture to the OLED.'); break;
      case 'cydsetup': pyScreen(G, b, Number(f.rot)); break;
      case 'cydfill': pyScreen(G, b); E(`screen.fill(${rgb565(f.color)})`, `Paints the whole screen ${colorName(f.color)}. Screen colours are written as 16-bit numbers; ${rgb565(f.color)} is ${colorName(f.color)}.`); break;
      case 'cydtext': pyScreen(G, b); E(`screen.text(${pyExpr(G, b, 'val')}, ${pyExpr(G, b, 'x')}, ${pyExpr(G, b, 'y')}, ${Number(f.size)||1}, ${rgb565(f.color)})`, `Writes text at size ${f.size} in ${colorName(f.color)}.`); break;
      case 'cydrect': pyScreen(G, b); E(`screen.rect(${pyExpr(G, b, 'x')}, ${pyExpr(G, b, 'y')}, ${pyExpr(G, b, 'w')}, ${pyExpr(G, b, 'h')}, ${rgb565(f.color)})`, `Draws a filled ${colorName(f.color)} box.`); break;
      default: E(`# ${b.t}`, '');
    }
  };
  const list2 = list => list.forEach(s => stmt(s));
  if(start) start.blocks.slice(1).forEach(s => stmt(s));
  else G.errs.push({bid:null, text:'Drag a “when the board starts” block onto the canvas and put your blocks under it.'});

  // assemble
  const head = [
    {t:'# Made with Flashie Firmware Creator (by nrmluvdev)', ex:'A comment: lines starting with # are notes for people. The board ignores them.'},
    {t:`# Board: ${ctx.boardName} (${ctx.chip})`, ex:'Another comment, recording which board this program was written for.'},
    {t:'', ex:''}];
  const imps = [...G.imports].sort((a,b)=> (a[0].startsWith('from') - b[0].startsWith('from')) || a[0].localeCompare(b[0])).map(([t, ex]) => ({t, ex}));
  const out = [...head, ...imps];
  if(G.setup.length) out.push({t:'', ex:''}, {t:'# --- set-up: runs once ---', ex:'A comment marking the set-up section.'}, ...G.setup);
  out.push({t:'', ex:''}, {t:'# --- your program ---', ex:'A comment marking where your blocks start.'}, ...G.lines);
  G.out = out;
  G.code = out.map(l => l.t).join('\n') + '\n';
}

/* ---- Arduino C++ ---- */
function cPin(G, b, pin, mode){
  pinCheck(G, b, pin, mode);
  const P = String(pin);
  if(mode === 'out') G.once('o'+P, [[`  pinMode(${P}, OUTPUT);`, `Makes pin ${pinLabel(G.ctx,pin)} an output so the program can switch it on and off.`, b.id]]);
  if(mode === 'in_up') G.once('u'+P, [[`  pinMode(${P}, INPUT_PULLUP);`, `Makes ${pinLabel(G.ctx,pin)} an input with the internal pull-up resistor on — it reads HIGH until a button connects it to GND.`, b.id]]);
  if(mode === 'in' || mode === 'in_down') G.once('i'+P, [[`  pinMode(${P}, INPUT);`, `Makes ${pinLabel(G.ctx,pin)} an input.`, b.id]]);
  return P;
}
function cExpr(G, b, name, want){
  const r = b.in[name];
  if(r){ if(!checkNeeds(G, r)) return want==='bool' ? 'false' : '0'; return cRep(G, r); }
  if(want === 'bool'){ G.warn(b.id, 'An empty condition slot counts as “false”.'); return 'false'; }
  const part = [...B[b.t].label, ...(B[b.t].label2||[])].find(p => p.name === name);
  const v = b.f[name];
  if(part && part.lit === 'num') return isNum(v) ? String(Number(v)) : '0';
  if(part && part.lit === 'auto') return isNum(v) ? String(Number(v)) : `String(${cStr(v ?? '')})`;
  return `String(${cStr(v ?? '')})`;
}
function cRep(G, r){
  const f = r.f;
  switch(r.t){
    case 'pinread': cPin(G, r, f.pin, f.pull==='up' ? 'in_up' : 'in'); return `(digitalRead(${f.pin}) == HIGH)`;
    case 'analogread': pinCheck(G, r, f.pin, 'adc'); return `(analogRead(${f.pin}) * 100L / 1023)`;
    case 'btnpressed': cPin(G, r, f.pin, 'in_up'); return `(digitalRead(${f.pin}) == LOW)`;
    case 'varget': return pyVar(f.var);
    case 'arith': return `(${cExpr(G, r, 'a')} ${f.op} ${cExpr(G, r, 'b')})`;
    case 'compare': return `(${cExpr(G, r, 'a')} ${f.op} ${cExpr(G, r, 'b')})`;
    case 'andor': return `(${cExpr(G, r, 'a', 'bool')} ${f.op==='and'?'&&':'||'} ${cExpr(G, r, 'b', 'bool')})`;
    case 'not': return `!(${cExpr(G, r, 'a', 'bool')})`;
    case 'random': return `random(${cExpr(G, r, 'a')}, ${cExpr(G, r, 'b')} + 1)`;
    case 'join': { const sx = e => /^String\(/.test(e) ? e : `String(${e})`; return `(${sx(cExpr(G, r, 'a'))} + ${sx(cExpr(G, r, 'b'))})`; }
    case 'dhttemp': return 'dhtTemperature';
    case 'dhthum': return 'dhtHumidity';
    case 'i2creg': cI2C(G, r); G.helperI2C = true; return `i2cReadRegister(${/^(0x[0-9a-f]+|\d+)$/i.test(String(f.addr).trim()) ? String(f.addr).trim() : '0x76'}, ${isNum(f.reg) ? Number(f.reg) : 0})`;
  }
  return '0';
}
function cInc(G, line, ex){ if(!G.includes.has(line)) G.includes.set(line, ex); }
function cI2C(G, b){
  cInc(G, '#include <Wire.h>', 'Brings in the Wire library for I²C devices.');
  G.once('wire', [['  Wire.begin();', `Starts I²C. On this board SDA and SCL are fixed: ${pinLabel(G.ctx, G.ctx.i2c[0])} (SDA) and ${pinLabel(G.ctx, G.ctx.i2c[1])} (SCL).`, b.id]]);
}

function genCpp(G, start){
  const ctx = G.ctx;
  const vars = new Set();
  const walk = list => list.forEach(b => { if(b.f && b.f.var !== undefined) vars.add(b.f.var); Object.values(b.in).forEach(x => x && walk([x])); walk(b.sub); walk(b.sub2); });
  if(start) walk(start.blocks);
  for(const v of vars) G.globals.push({t:`float ${pyVar(v)} = 0;`, ex:`Creates the variable “${v}” (a number that can have decimals) starting at 0.`});
  G.once('serial', [['  Serial.begin(115200);', 'Opens the USB serial connection at 115200 baud so print messages reach Flashie’s console.', null]]);

  const body = (list, ownerBid) => { G.ind++; const before = G.lines.length; list.forEach(stmt); if(G.lines.length === before) G.emit('// (nothing here yet)', 'This part is empty.', ownerBid); G.ind--; };
  const stmt = b => {
    if(!checkNeeds(G, b)) { G.emit(`// (skipped: ${plainLabel(b)})`, 'This block can’t be turned into Arduino code — see the warnings below the blocks.', b.id); return; }
    const f = b.f, E = (t, ex) => G.emit(t, ex, b.id);
    switch(b.t){
      case 'forever': E('while (true) {', 'A loop that never ends: everything between { and } repeats forever.'); body(b.sub, b.id); E('}', 'End of the forever loop.'); break;
      case 'repeat': { const v = 'ijkmn'[Math.min(G.loopCounter++, 4)]; E(`for (int ${v} = 0; ${v} < ${cExpr(G, b, 'n')}; ${v}++) {`, `Counts ${v} from 0 up, repeating the code inside once per count.`); body(b.sub, b.id); E('}', 'End of the repeat.'); G.loopCounter--; break; }
      case 'while': E(`while (${cExpr(G, b, 'cond', 'bool')}) {`, 'Repeats the code inside while the condition is true.'); body(b.sub, b.id); E('}', 'End of the loop.'); break;
      case 'if': E(`if (${cExpr(G, b, 'cond', 'bool')}) {`, 'Runs the code inside only when the condition is true.'); body(b.sub, b.id); E('}', 'End of the if.'); break;
      case 'ifelse': E(`if (${cExpr(G, b, 'cond', 'bool')}) {`, 'When the condition is true, run this part…'); body(b.sub, b.id); E('} else {', '…otherwise run this part.'); body(b.sub2, b.id); E('}', 'End of the if/else.'); break;
      case 'wait': E(`delay((unsigned long)(${cExpr(G, b, 's')} * 1000));`, 'Pauses. Arduino counts in milliseconds, so seconds × 1000.'); break;
      case 'waitms': E(`delay(${cExpr(G, b, 'ms')});`, 'Pauses for that many milliseconds.'); break;
      case 'pinwrite': cPin(G, b, f.pin, 'out'); E(`digitalWrite(${f.pin}, ${f.state==='on'?'HIGH':'LOW'});`, `Switches ${pinLabel(ctx, f.pin)} ${f.state} (${f.state==='on'?'HIGH = 5 V':'LOW = 0 V'}).`); break;
      case 'pintoggle': cPin(G, b, f.pin, 'out'); E(`digitalWrite(${f.pin}, !digitalRead(${f.pin}));`, 'Reads the pin and writes the opposite (“!” means “not”).'); break;
      case 'pwm': pinCheck(G, b, f.pin, 'pwm'); E(`analogWrite(${f.pin}, (int)(${cExpr(G, b, 'pct')} * 255 / 100));`, 'Sets the brightness. Arduino PWM goes from 0 to 255, so we scale the percentage.'); break;
      case 'led': case 'ledtoggle': G.once('oLED', [['  pinMode(LED_BUILTIN, OUTPUT);', 'Makes the built-in LED’s pin an output.', b.id]]);
        if(b.t==='ledtoggle') E('digitalWrite(LED_BUILTIN, !digitalRead(LED_BUILTIN));', 'Flips the built-in LED (on pin 13).');
        else E(`digitalWrite(LED_BUILTIN, ${f.state==='on'?'HIGH':'LOW'});`, `Turns the built-in LED (pin 13) ${f.state}.`); break;
      case 'neosetup': cInc(G, '#include <Adafruit_NeoPixel.h>', 'Brings in the Adafruit NeoPixel library (Flashie installs it for you).'); G.libs.add('Adafruit NeoPixel');
        pinCheck(G, b, f.pin, 'out');
        if(!G.strip) G.globals.push({t:`Adafruit_NeoPixel strip(${isNum(f.n) ? Number(f.n) : 8}, ${f.pin}, NEO_GRB + NEO_KHZ800);`, ex:`Describes the NeoPixel strip: how many LEDs and which pin. We call it “strip”.`, bid:b.id});
        G.once('strip', [['  strip.begin();', 'Gets the strip ready.', b.id]]); G.strip = true; break;
      case 'neoset': E(`strip.setPixelColor(${cExpr(G, b, 'i')} - 1, strip.Color(${hexRGB(f.color).join(', ')}));`, `Sets one LED to ${colorName(f.color)} (counting from 0, so we subtract 1).`); break;
      case 'neofill': E(`strip.fill(strip.Color(${hexRGB(f.color).join(', ')}));`, `Sets every LED to ${colorName(f.color)}.`); break;
      case 'neoshow': E('strip.show();', 'Sends the colours to the strip.'); break;
      case 'waitbtn': cPin(G, b, f.pin, 'in_up'); E(`while (digitalRead(${f.pin}) == HIGH) {`, 'While the button is not pressed…'); G.ind++; E('delay(10);', '…check again every 10 ms.'); G.ind--; E('}', 'End of the wait.'); break;
      case 'setvar': E(`${pyVar(f.var)} = ${cExpr(G, b, 'val')};`, `Stores a new value in “${f.var}”.`); break;
      case 'changevar': E(`${pyVar(f.var)} += ${cExpr(G, b, 'n')};`, `Adds to “${f.var}”.`); break;
      case 'print': E(`Serial.println(${cExpr(G, b, 'val')});`, 'Sends this to your PC — it appears in Flashie’s console.'); break;
      case 'dhtread': cInc(G, '#include <DHT.h>', 'Brings in the DHT sensor library (Flashie installs it for you).'); G.libs.add('DHT sensor library'); G.libs.add('Adafruit Unified Sensor');
        pinCheck(G, b, f.pin, 'in');
        if(!G.dht){ G.globals.push({t:`DHT dht(${f.pin}, ${f.model});`, ex:`Describes a ${f.model} sensor on pin ${f.pin}.`, bid:b.id}, {t:'float dhtTemperature = 0, dhtHumidity = 0;', ex:'Two variables to keep the latest readings.', bid:b.id}); }
        G.once('dht', [['  dht.begin();', 'Starts the sensor.', b.id]]); G.dht = true;
        E('dhtTemperature = dht.readTemperature();', 'Reads the temperature in °C.'); E('dhtHumidity = dht.readHumidity();', 'Reads the humidity in %.'); break;
      case 'i2csetup': cI2C(G, b); if(String(f.sda)!==String(ctx.i2c[0]) || String(f.scl)!==String(ctx.i2c[1])) G.warn(b.id, `On ${ctx.boardName} I²C always uses ${pinLabel(ctx, ctx.i2c[0])} (SDA) and ${pinLabel(ctx, ctx.i2c[1])} (SCL) — the pins in this block are ignored.`); break;
      case 'i2cscan': cI2C(G, b);
        E('Serial.print("I2C devices found:");', 'Starts the message.');
        E('for (byte address = 1; address < 127; address++) {', 'Tries every possible I²C address…'); G.ind++;
        E('Wire.beginTransmission(address);', '…knocks on that address…');
        E('if (Wire.endTransmission() == 0) {', '…and if something answered…'); G.ind++;
        E('Serial.print(" 0x"); Serial.print(address, HEX);', '…prints its address in hex.'); G.ind--; E('}', ''); G.ind--; E('}', ''); E('Serial.println();', 'Ends the line.'); break;
      case 'servo': { cInc(G, '#include <Servo.h>', 'Brings in the Servo library.'); G.libs.add('Servo'); pinCheck(G, b, f.pin, 'pwm');
        const v = 'servo' + String(f.pin).replace(/\W/g,'');
        if(!G.setupKeys.has('sv'+v)) G.globals.push({t:`Servo ${v};`, ex:`Creates a servo controller called “${v}”.`, bid:b.id});
        G.once('sv'+v, [[`  ${v}.attach(${f.pin});`, `Connects “${v}” to pin ${f.pin}.`, b.id]]);
        E(`${v}.write(constrain((int)(${cExpr(G, b, 'angle')}), 0, 180));`, 'Turns the servo to the angle (kept between 0 and 180).'); break; }
      case 'oledsetup': cI2C(G, b); cInc(G, '#include <Adafruit_GFX.h>', 'Brings in the Adafruit graphics library.'); cInc(G, '#include <Adafruit_SSD1306.h>', 'Brings in the SSD1306 OLED library.');
        G.libs.add('Adafruit SSD1306'); G.libs.add('Adafruit GFX Library');
        if(!G.oled) G.globals.push({t:`Adafruit_SSD1306 oled(128, ${f.h}, &Wire, -1);`, ex:`Describes a 128×${f.h} OLED on I²C.`, bid:b.id});
        G.once('oled', [['  oled.begin(SSD1306_SWITCHCAPVCC, 0x3C);', 'Starts the OLED at its usual address, 0x3C.', b.id], ['  oled.clearDisplay();', 'Clears it.', b.id], ['  oled.setTextColor(SSD1306_WHITE, SSD1306_BLACK);', 'White text on black.', b.id]]); G.oled = true; break;
      case 'oledtext': { const y = (Math.max(1, Number(f.line)||1)-1)*10; E(`oled.fillRect(0, ${y}, 128, 10, SSD1306_BLACK);`, 'Clears the line first.'); E(`oled.setCursor(0, ${y});`, `Moves to line ${f.line}.`); E(`oled.print(${cExpr(G, b, 'val')});`, 'Draws the text.'); break; }
      case 'oledclear': E('oled.clearDisplay();', 'Clears the OLED memory.'); break;
      case 'oledshow': E('oled.display();', 'Sends the picture to the OLED.'); break;
      default: E(`// ${b.t}`, '');
    }
  };
  // Last block "forever" becomes loop(); everything else runs in setup().
  const blocks = start ? start.blocks.slice(1) : [];
  const last = blocks[blocks.length - 1];
  const loopBody = last && last.t === 'forever' ? last : null;
  G.ind = 1;
  (loopBody ? blocks.slice(0, -1) : blocks).forEach(stmt);
  const setupLines = G.lines; G.lines = [];
  if(loopBody){ loopBody.sub.forEach(stmt); }
  const loopLines = G.lines;
  if(!start) G.errs.push({bid:null, text:'Drag a “when the board starts” block onto the canvas and put your blocks under it.'});

  const out = [{t:'// Made with Flashie Firmware Creator (by nrmluvdev)', ex:'A comment: lines starting with // are notes for people.'},
    {t:`// Board: ${ctx.boardName}`, ex:'Records which board this sketch is for.'}, {t:'', ex:''}];
  for(const [t, ex] of G.includes) out.push({t, ex});
  if(G.includes.size) out.push({t:'', ex:''});
  for(const g of G.globals) out.push(g);
  if(G.helperI2C){
    out.push({t:'', ex:''}, {t:'int i2cReadRegister(int address, int reg) {', ex:'A small helper function that reads one byte from an I²C device’s register.'},
      {t:'  Wire.beginTransmission(address);', ex:'Starts talking to the device.'}, {t:'  Wire.write(reg);', ex:'Says which register we want.'},
      {t:'  Wire.endTransmission(false);', ex:'Keeps the connection open…'}, {t:'  Wire.requestFrom(address, 1);', ex:'…and asks for 1 byte.'},
      {t:'  return Wire.available() ? Wire.read() : -1;', ex:'Returns the byte, or -1 if nothing came back.'}, {t:'}', ex:''});
  }
  if(G.globals.length || G.helperI2C) out.push({t:'', ex:''});
  out.push({t:'void setup() {', ex:'“setup()” runs once when the board starts.'}, ...G.setup.map(s=>({t:s.t, ex:s.ex, bid:s.bid})), ...setupLines, {t:'}', ex:'End of setup().'}, {t:'', ex:''});
  out.push({t:'void loop() {', ex: loopBody ? '“loop()” repeats forever — this is your “forever” block.' : '“loop()” repeats forever. Your program has no “forever” block at the end, so it is empty.', bid: loopBody ? loopBody.id : undefined},
    ...loopLines, {t:'}', ex:'End of loop().'});
  G.out = out;
  G.code = out.map(l => l.t).join('\n') + '\n';
}

/* ======================================================================
   Rendering: palette, canvas, blocks
   ====================================================================== */

const $c = s => document.querySelector(s);
let IDX = {}; // block id → {list, index} | {owner, slot}

function blockAvailable(type, ctx){
  const d = B[type];
  if(d.py && ctx.lang === 'cpp') return 'Needs MicroPython — not available on Arduino boards.';
  if(d.need === 'wifi' && !ctx.hasWifi) return `${ctx.boardName} has no Wi-Fi.`;
  if(d.need === 'cyd' && !ctx.cyd) return 'Pick a Cheap Yellow Display board to use these.';
  if(d.need === 'cydrgb' && !(ctx.led && ctx.led.kind==='rgb')) return 'Only on Cheap Yellow Displays.';
  return '';
}

function renderPalette(){
  const ctx = C.ctx, el = $c('#cPal');
  const cat = C.cat || 'control';
  let h = `<div class="pcats">${CATS.map(c => `<button class="pcat ${c.id===cat?'on':''}" data-cat="${c.id}" title="${esc(c.name)}"><i style="background:${c.color}"></i><span>${esc(c.name)}</span></button>`).join('')}</div><div class="plist">`;
  if(cat === 'vars') h += `<button class="btn" id="cNewVar" style="margin-bottom:6px">+ Make a variable</button>`;
  for(const t in B){
    if(B[t].cat !== cat) continue;
    const na = blockAvailable(t, ctx);
    h += `<div class="pitem ${na?'na':''}" data-type="${t}" title="${esc(na || B[t].ex)}">${blockHTML(previewBlock(t), true)}</div>`;
  }
  el.innerHTML = h + '</div>';
}
function previewBlock(t){
  const b = make(t);
  const d = B[t];
  for(const p of [...d.label, ...(d.label2||[])]) if(p.k === 'pin') b.f[p.name] = '…';
  return b;
}

function fieldHTML(b, p, preview){
  const v = b.f[p.name];
  const ctx = C.ctx;
  switch(p.k){
    case 'in': {
      const r = b.in[p.name];
      if(r) return `<span class="slot full" data-owner="${b.id}" data-slot="${p.name}">${blockHTML(r, preview)}</span>`;
      if(p.acc === 'bool') return `<span class="slot bool" data-owner="${b.id}" data-slot="${p.name}" data-acc="bool"></span>`;
      return `<span class="slot" data-owner="${b.id}" data-slot="${p.name}" data-acc="val"><input class="fin ${p.lit==='num'?'num':''}" data-f="${p.name}" value="${esc(v ?? '')}" size="${Math.max(2, String(v ?? '').length)}" ${preview?'tabindex="-1"':''}></span>`;
    }
    case 'num': return `<input class="fin num" data-f="${p.name}" value="${esc(v ?? '')}" size="${Math.max(2, String(v ?? '').length)}" ${preview?'tabindex="-1"':''}>`;
    case 'text': return `<input class="fin" data-f="${p.name}" ${p.secret?'type="password"':''} value="${esc(v ?? '')}" size="${Math.max(4, String(v ?? '').length)}" ${preview?'tabindex="-1"':''}>`;
    case 'drop': return `<select class="fdrop" data-f="${p.name}">${p.opts.map(([k, l]) => `<option value="${esc(k)}" ${k===v?'selected':''}>${esc(l)}</option>`).join('')}</select>`;
    case 'var': return `<select class="fdrop" data-f="${p.name}">${W.vars.map(n => `<option ${n===v?'selected':''}>${esc(n)}</option>`).join('')}</select>`;
    case 'color': return `<button class="fcolor" data-f="${p.name}" style="background:${esc(v)}" title="${esc(colorName(v))}"></button>`;
    case 'pin': {
      if(v === '…' || v === null || v === undefined) return `<button class="fpin" data-f="${p.name}" data-mode="${p.mode}">pin</button>`;
      const w = preview ? null : pinWarn(ctx, v, p.mode === 'i2c' ? 'i2c' : p.mode);
      const info = pinInfo(ctx, v);
      return `<button class="fpin ${w ? 'bad' : info && info.level==='care' ? 'care' : ''}" data-f="${p.name}" data-mode="${p.mode}">${esc(shortPin(ctx, v))}${w ? ' ⚠' : ''}</button>`;
    }
  }
  return '';
}
const shortPin = (ctx, id) => ctx.fam==='avr' ? (typeof id==='number' ? 'D'+id : id) : ctx.fam==='rp2' ? 'GP'+id : (id==='A0' ? 'A0' : String(id));

function blockHTML(b, preview){
  const d = B[b.t], cat = catOf(d.cat);
  const lab = parts => parts.map(p => typeof p === 'string' ? `<span class="bt">${esc(p)}</span>` : fieldHTML(b, p, preview)).join('');
  const na = C.ctx ? blockAvailable(b.t, C.ctx) : '';
  const err = !preview && C.gen && (C.gen.errs.some(e=>e.bid===b.id) || C.gen.warns.some(e=>e.bid===b.id));
  const cls = `blk s-${d.shape} ${C.sel===b.id && !preview?'sel':''} ${na && !preview?'na':''} ${err?'warn':''}`;
  const style = `--bc:${cat.color}`;
  if(d.shape === 'c' || d.shape === 'cc'){
    return `<div class="${cls}" data-id="${b.id}" style="${style}"><div class="brow">${lab(d.label)}</div>
      <div class="mouth"><div class="stack sub" data-owner="${b.id}" data-slot="sub">${preview ? '' : b.sub.map(x=>blockHTML(x)).join('')}</div></div>
      ${d.shape==='cc' ? `<div class="brow mid">${lab(d.label2)}</div><div class="mouth"><div class="stack sub" data-owner="${b.id}" data-slot="sub2">${preview ? '' : b.sub2.map(x=>blockHTML(x)).join('')}</div></div>` : ''}
      <div class="bfoot"></div></div>`;
  }
  return `<div class="${cls}" data-id="${b.id}" style="${style}"><div class="brow">${lab(d.label)}</div></div>`;
}

function indexWorkspace(){
  IDX = {};
  const visit = (list, owner, slot) => list.forEach((b, i) => {
    IDX[b.id] = {list, index: i, owner, slot, b};
    for(const k in b.in) if(b.in[k]){ IDX[b.in[k].id] = {owner: b, inSlot: k, b: b.in[k]}; visitRep(b.in[k]); }
    visit(b.sub, b, 'sub'); visit(b.sub2, b, 'sub2');
  });
  const visitRep = r => { for(const k in r.in) if(r.in[k]){ IDX[r.in[k].id] = {owner: r, inSlot: k, b: r.in[k]}; visitRep(r.in[k]); } };
  W.stacks.forEach(s => visit(s.blocks, null, null, s));
  W.stacks.forEach(s => s.blocks.forEach((b, i) => { IDX[b.id].stack = s; }));
}

function renderCanvas(){
  indexWorkspace();
  const el = $c('#cCanvas');
  let maxX = 0, maxY = 0;
  el.innerHTML = W.stacks.map((s, i) => {
    maxX = Math.max(maxX, s.x + 500); maxY = Math.max(maxY, s.y + 400 + s.blocks.length * 40);
    return `<div class="stack top" data-stack="${i}" style="left:${s.x}px;top:${s.y}px">${s.blocks.map(b=>blockHTML(b)).join('')}</div>`;
  }).join('') + `<div class="cspacer" style="left:${maxX}px;top:${maxY}px"></div>` + (W.stacks.length ? '' : `<div class="cempty">Drag blocks here from the left — or pick a template above.<br>Start with <b>“when the board starts”</b>.</div>`);
  // attach list refs to stack containers
  el.querySelectorAll('.stack.top').forEach(d => { d._list = W.stacks[+d.dataset.stack].blocks; d._stack = W.stacks[+d.dataset.stack]; });
  el.querySelectorAll('.stack.sub').forEach(d => { const o = IDX[d.dataset.owner]; if(o) d._list = o.b[d.dataset.slot]; });
}

function renderAll(){
  C.ctx = creatorCtx();
  renderTop();
  renderPalette();
  regen();
}

function regen(){
  if(!C.ctx) C.ctx = creatorCtx();
  C.gen = generate(C.ctx);
  renderCanvas();
  renderCode();
  renderWarnings();
  save();
}

/* ======================================================================
   Code view (blocks mode) and text editor (text mode)
   ====================================================================== */

const PY_HL = /(#.*$)|("(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*')|\b(def|class|if|elif|else|while|for|in|import|from|as|return|True|False|None|and|or|not|try|except|with|pass|break|continue|global|lambda|is)\b|\b(0x[0-9a-fA-F]+|\d+(?:\.\d+)?)\b|(\w+)(?=\()/g;
const CPP_HL = /(\/\/.*$)|("(?:[^"\\]|\\.)*")|(#include\b|\b(?:void|int|float|long|unsigned|byte|bool|true|false|if|else|for|while|return|const|String|HIGH|LOW|OUTPUT|INPUT|INPUT_PULLUP|LED_BUILTIN)\b)|\b(0x[0-9a-fA-F]+|\d+(?:\.\d+)?)\b|(\w+)(?=\()/g;
function hl(line, lang){
  const re = lang === 'cpp' ? CPP_HL : PY_HL;
  let out = '', last = 0;
  line.replace(re, (m, com, str, kw, num, fn, off) => {
    out += esc(line.slice(last, off));
    out += `<span class="${com?'tc':str?'ts':kw?'tk':num?'tn':'tf'}">${esc(m)}</span>`;
    last = off + m.length;
    return m;
  });
  return out + esc(line.slice(last));
}

function renderCode(){
  const lang = C.ctx.lang || 'py';
  $c('#cFile').textContent = lang === 'cpp' ? 'flashie_sketch.ino · Arduino C++' : 'main.py · MicroPython';
  if(C.textMode){ syncEditor(); return; }
  $c('#cEdWrap').classList.add('hidden');
  const cv = $c('#cCode');
  cv.classList.remove('hidden');
  cv.innerHTML = C.gen.out.map((l, i) => `<div class="ln ${l.bid && l.bid===C.hoverBid ? 'hl' : ''}" data-i="${i}" ${l.bid ? `data-bid="${l.bid}"` : ''}><span class="no">${i+1}</span><span class="tx">${hl(l.t, lang) || ' '}</span></div>`).join('');
}

function syncEditor(){
  const lang = C.ctx.lang || 'py';
  $c('#cCode').classList.add('hidden');
  $c('#cEdWrap').classList.remove('hidden');
  const ta = $c('#cEd');
  if(ta.value !== C.text) ta.value = C.text;
  paintEditor();
}
function paintEditor(){
  const lang = C.ctx.lang || 'py', ta = $c('#cEd');
  const lines = ta.value.split('\n');
  $c('#cEdHl').innerHTML = lines.map(l => hl(l, lang) || ' ').join('\n') + '\n';
  $c('#cEdNo').textContent = lines.map((_, i) => i+1).join('\n');
}

function setTextMode(on){
  if(on && !C.textMode){ C.text = C.gen.code; }
  if(!on && C.textMode && C.text !== C.gen.code && !confirm('Go back to blocks? Your typed changes will be thrown away (the blocks can’t be rebuilt from code).')) return;
  C.textMode = on;
  $c('#cMode').querySelectorAll('button').forEach(b => b.classList.toggle('on', (b.dataset.m === 'text') === on));
  $c('#cBody').classList.toggle('textmode', on);
  renderCode();
  teachMsg(on ? 'Typing your own code' : 'Back to blocks', on
    ? 'You can now edit the code directly. The blocks are paused — changes here don’t go back into blocks. Press “Upload & run” when you’re ready. Use 4 spaces for each level of indentation; Python uses indentation to know which lines belong together.'
    : 'Showing the code made from your blocks again.');
  save();
}

/* ======================================================================
   Teaching panel + warnings
   ====================================================================== */

function teachMsg(title, html){ $c('#cTeach').innerHTML = `<h4>${esc(title)}</h4><p>${esc(html)}</p>`; }

function teachBlock(b){
  if(!b) return;
  const d = B[b.t], ctx = C.ctx;
  const lines = C.gen.out.map((l, i) => ({...l, i})).filter(l => l.bid === b.id);
  let h = `<div class="tb"><span class="chip3" style="background:${catOf(d.cat).color}">${esc(catOf(d.cat).name)}</span><h4>${esc(plainLabel(b))}</h4></div><p>${esc(d.ex)}</p>`;
  const na = blockAvailable(b.t, ctx);
  if(na) h += note('warn', esc(na));
  for(const p of [...d.label, ...(d.label2||[])]){
    if(p.k === 'pin' && b.f[p.name] !== null && b.f[p.name] !== '…'){
      const info = pinInfo(ctx, b.f[p.name]), w = pinWarn(ctx, b.f[p.name], p.mode);
      if(w) h += note('bad', '<b>Pin problem:</b> ' + esc(w));
      else if(info && info.notes.length) h += note(info.level==='ok' ? 'info' : 'warn', `<b>${esc(info.label)}:</b> ${esc(info.notes.join(' '))}`);
      else if(info) h += note('ok', `<b>${esc(info.label)}</b> is a good, general-purpose pin.`);
    }
  }
  for(const e of C.gen.errs.filter(e=>e.bid===b.id)) h += note('bad', esc(e.text));
  for(const w of C.gen.warns.filter(e=>e.bid===b.id)) if(!h.includes(esc(w.text))) h += note('warn', esc(w.text));
  if(lines.length){
    h += `<div class="label" style="margin-top:12px">The code this block makes</div>`;
    h += lines.map(l => `<div class="tline"><code>${hl(l.t.trim(), ctx.lang)}</code><span>${esc(l.ex)}</span></div>`).join('');
  }
  $c('#cTeach').innerHTML = h;
}

function teachLine(i){
  const l = C.gen.out[i];
  if(!l) return;
  const b = l.bid && IDX[l.bid] ? IDX[l.bid].b : null;
  let h = `<div class="tb"><span class="chip3">Line ${i+1}</span></div><div class="tline big"><code>${hl(l.t.trim(), C.ctx.lang) || '<i>(blank line)</i>'}</code></div>`;
  h += `<p>${esc(l.ex || (l.t.trim() ? '' : 'Blank lines just make code easier to read.'))}</p>`;
  if(b) h += `<p class="src">Made by the block <b>“${esc(plainLabel(b))}”</b>. ${esc(B[b.t].ex)}</p>`;
  $c('#cTeach').innerHTML = h;
}

function renderWarnings(){
  const g = C.gen, el = $c('#cWarn');
  const items = [...g.errs.map(e => ['bad', e]), ...g.warns.map(e => ['warn', e])];
  el.innerHTML = items.length ? items.slice(0, 6).map(([k, e]) => `<div class="wi ${k}" ${e.bid?`data-bid="${e.bid}"`:''}>${k==='bad'?'⛔':'⚠'} ${esc(e.text)}</div>`).join('') + (items.length > 6 ? `<div class="wi">…and ${items.length-6} more</div>` : '')
    : `<div class="wi ok">✓ No problems found${C.ctx.lang==='cpp' ? ' — this will be compiled as Arduino C++' : ''}.</div>`;
}

/* ======================================================================
   Pin picker + colour picker popovers
   ====================================================================== */

function closePop(){ const p = $c('#cPop'); if(p) p.remove(); }
function openPinPicker(btn, b, name, mode){
  closePop();
  const ctx = C.ctx, cur = b.f[name];
  const groups = {ok:[], care:[], used:[], avoid:[]};
  for(const p of ctx.pins){
    let lvl = p.level;
    const w = pinWarn(ctx, p.id, mode);
    if(w && lvl !== 'used') lvl = 'avoid';
    groups[lvl].push({...p, why: w || p.notes.join(' ')});
  }
  const titles = {ok:'Good to use', care:'Use with care', used:'Used by the board', avoid:'Not suitable for this'};
  const pop = document.createElement('div');
  pop.id = 'cPop'; pop.className = 'pop pinpop';
  pop.innerHTML = `<div class="poph">Choose a pin <span class="src">${esc(ctx.boardName)}</span></div>` +
    Object.entries(groups).filter(([, a]) => a.length).map(([k, a]) => `<div class="pgh ${k}">${titles[k]}</div><div class="pgrid">${a.map(p =>
      `<button class="pp ${k} ${String(p.id)===String(cur)?'cur':''}" data-pin="${esc(String(p.id))}" title="${esc(p.why)}"><b>${esc(shortPin(ctx, p.id))}</b>${p.why ? `<span>${esc(p.why.slice(0, 70))}${p.why.length>70?'…':''}</span>` : ''}</button>`).join('')}</div>`).join('');
  document.body.appendChild(pop);
  placePop(pop, btn);
  pop.onclick = e => {
    const x = e.target.closest('.pp'); if(!x) return;
    const v = x.dataset.pin;
    b.f[name] = /^\d+$/.test(v) ? Number(v) : v;
    closePop(); regen(); teachBlock(b);
  };
  pop.onmouseover = e => { const x = e.target.closest('.pp'); if(x) $c('#cTeach').innerHTML = `<h4>${esc(pinLabel(ctx, /^\d+$/.test(x.dataset.pin) ? Number(x.dataset.pin) : x.dataset.pin))}</h4><p>${esc(x.title || 'A good, general-purpose pin.')}</p>`; };
}
function openColor(btn, b, name){
  closePop();
  const pop = document.createElement('div');
  pop.id = 'cPop'; pop.className = 'pop';
  pop.innerHTML = `<div class="poph">Pick a colour</div><div class="cgrid">${COLORS.map(([c, n]) => `<button class="cc" data-c="${c}" title="${n}" style="background:${c}"></button>`).join('')}</div>
    <label class="src" style="display:flex;gap:8px;align-items:center;margin-top:8px">Any colour: <input type="color" value="${esc(b.f[name])}"></label>`;
  document.body.appendChild(pop); placePop(pop, btn);
  pop.onclick = e => { const x = e.target.closest('.cc'); if(x){ b.f[name] = x.dataset.c; closePop(); regen(); } };
  pop.querySelector('input').oninput = e => { b.f[name] = e.target.value; regen(); };
}
function placePop(pop, btn){
  const r = btn.getBoundingClientRect();
  const w = pop.offsetWidth, h = pop.offsetHeight;
  pop.style.left = Math.min(window.innerWidth - w - 10, Math.max(10, r.left)) + 'px';
  pop.style.top = (r.bottom + h + 10 < window.innerHeight ? r.bottom + 6 : Math.max(10, r.top - h - 6)) + 'px';
}

/* ======================================================================
   Drag and drop
   ====================================================================== */

let DRAG = null;

function pointerDown(e){
  if(e.button !== 0) return;
  const t = e.target;
  if(t.closest('input, select, .fpin, .fcolor, #cNewVar')) return;
  const pal = t.closest('.pitem');
  const blk = t.closest('.blk');
  if(!blk) return;
  if(pal && pal.classList.contains('na')) { teachMsg(plainLabel(make(pal.dataset.type)), blockAvailable(pal.dataset.type, C.ctx)); return; }
  e.preventDefault();
  DRAG = {x0: e.clientX, y0: e.clientY, started: false, fromPal: !!pal, type: pal && pal.dataset.type, id: !pal && blk.dataset.id, el: blk};
  window.addEventListener('pointermove', pointerMove);
  window.addEventListener('pointerup', pointerUp, {once: true});
}

function startDrag(e){
  let blocks;
  const r = DRAG.el.getBoundingClientRect();
  DRAG.ox = e.clientX - r.left; DRAG.oy = e.clientY - r.top;
  if(DRAG.fromPal){
    const b = make(DRAG.type); fillPins(b, C.ctx);
    blocks = [b];
  } else {
    const loc = IDX[DRAG.id];
    if(!loc) return false;
    if(loc.inSlot){ blocks = [loc.b]; loc.owner.in[loc.inSlot] = null; }
    else {
      blocks = loc.list.splice(loc.index);
      if(loc.stack && loc.stack.blocks.length === 0) W.stacks.splice(W.stacks.indexOf(loc.stack), 1);
    }
  }
  DRAG.blocks = blocks;
  DRAG.kind = isReporter(blocks[0].t) ? B[blocks[0].t].shape : B[blocks[0].t].shape === 'hat' ? 'hat' : 'stmt';
  const g = document.createElement('div');
  g.className = 'dragghost stack';
  g.innerHTML = blocks.map(b => blockHTML(b)).join('');
  document.body.appendChild(g);
  DRAG.ghost = g;
  if(!DRAG.fromPal) { renderCanvas(); }
  DRAG.started = true;
  document.body.classList.add('dragging');
  return true;
}

function pointerMove(e){
  if(!DRAG) return;
  if(!DRAG.started){
    if(Math.hypot(e.clientX - DRAG.x0, e.clientY - DRAG.y0) < 5) return;
    if(!startDrag(e)) { DRAG = null; return; }
  }
  const gx = e.clientX - DRAG.ox, gy = e.clientY - DRAG.oy;
  DRAG.ghost.style.left = gx + 'px'; DRAG.ghost.style.top = gy + 'px';
  DRAG.target = findTarget(gx, gy);
  showTarget(DRAG.target);
  $c('#cPal').classList.toggle('trash', overPalette(e));
}

function overPalette(e){ const r = $c('#cPal').getBoundingClientRect(); return e.clientX < r.right && e.clientX > r.left && e.clientY > r.top && e.clientY < r.bottom; }

function findTarget(gx, gy){
  let best = null, bd = 36;
  if(DRAG.kind === 'stmt'){
    document.querySelectorAll('#cCanvas .stack').forEach(st => {
      const list = st._list; if(!list) return;
      const sr = st.getBoundingClientRect();
      const kids = [...st.children].filter(c => c.classList.contains('blk'));
      for(let i = 0; i <= kids.length; i++){
        if(st.classList.contains('top') && i === 0 && list[0] && B[list[0].t].shape === 'hat') continue;
        const y = i === 0 ? sr.top : kids[i-1].getBoundingClientRect().bottom;
        const d = Math.hypot(gx - sr.left, gy - y);
        if(d < bd){ bd = d; best = {list, index: i, x: sr.left, y, w: Math.max(120, sr.width)}; }
      }
      // dropping a stack so its bottom joins the top of a loose stack
      if(st.classList.contains('top') && list[0] && B[list[0].t].shape !== 'hat'){
        const d = Math.hypot(gx - sr.left, gy + DRAG.ghost.offsetHeight - sr.top);
        if(d < bd){ bd = d; best = {list, index: 0, x: sr.left, y: sr.top, w: sr.width, above: true}; }
      }
    });
  } else if(DRAG.kind !== 'hat'){
    document.querySelectorAll('#cCanvas .slot').forEach(sl => {
      if(sl.dataset.acc === 'bool' && DRAG.kind !== 'bool') return;
      if(sl.classList.contains('full')) return;
      const r = sl.getBoundingClientRect();
      const d = Math.hypot(gx - r.left, gy + 12 - (r.top + r.height/2));
      if(d < bd){ bd = d; best = {slot: sl, owner: sl.dataset.owner, name: sl.dataset.slot}; }
    });
  }
  return best;
}

function showTarget(t){
  let ind = $c('#cInd');
  document.querySelectorAll('.slot.hot').forEach(s => s.classList.remove('hot'));
  if(!t){ if(ind) ind.style.display = 'none'; return; }
  if(t.slot){ t.slot.classList.add('hot'); if(ind) ind.style.display = 'none'; return; }
  if(!ind){ ind = document.createElement('div'); ind.id = 'cInd'; document.body.appendChild(ind); }
  ind.style.display = 'block';
  ind.style.left = t.x + 'px'; ind.style.top = (t.y - 2) + 'px'; ind.style.width = Math.min(260, t.w) + 'px';
}

function pointerUp(e){
  window.removeEventListener('pointermove', pointerMove);
  if(!DRAG) return;
  if(!DRAG.started){
    // a click: select the block
    if(!DRAG.fromPal){ C.sel = DRAG.id; regen(); const loc = IDX[DRAG.id]; if(loc) teachBlock(loc.b); }
    else teachBlock(make(DRAG.type));
    DRAG = null; return;
  }
  const t = DRAG.target, blocks = DRAG.blocks;
  DRAG.ghost.remove();
  const ind = $c('#cInd'); if(ind) ind.style.display = 'none';
  $c('#cPal').classList.remove('trash');
  document.body.classList.remove('dragging');
  if(overPalette(e)){
    teachMsg('Deleted', 'Blocks dragged back to the block list are deleted.');
  } else if(t && t.slot){
    const o = IDX[t.owner]; if(o) o.b.in[t.name] = blocks[0];
  } else if(t && t.list){
    if(blocks[0].t === 'start' && t.list !== W.stacks.find(s => s.blocks === t.list)?.blocks){ addStack(e, blocks); }
    else t.list.splice(t.index, 0, ...blocks);
  } else addStack(e, blocks);
  C.sel = blocks[0].id;
  DRAG = null;
  regen();
  const loc = IDX[blocks[0].id]; if(loc) teachBlock(loc.b);
}

function addStack(e, blocks){
  const cv = $c('#cCanvas'), r = cv.getBoundingClientRect();
  W.stacks.push({x: Math.max(4, e.clientX - DRAG.ox - r.left + cv.scrollLeft), y: Math.max(4, e.clientY - DRAG.oy - r.top + cv.scrollTop), blocks});
}

function deleteSelected(){
  const loc = IDX[C.sel];
  if(!loc) return;
  if(loc.inSlot) loc.owner.in[loc.inSlot] = null;
  else { loc.list.splice(loc.index, 1); if(loc.stack && !loc.stack.blocks.length) W.stacks.splice(W.stacks.indexOf(loc.stack), 1); }
  C.sel = null; regen();
}

function tidy(){
  let y = 24;
  W.stacks.forEach(s => { s.x = 30; s.y = y; y += 60 + countBlocks(s.blocks) * 44; });
  regen();
}
const countBlocks = list => list.reduce((n, b) => n + 1 + countBlocks(b.sub) + countBlocks(b.sub2), 0);

/* ======================================================================
   Persistence
   ====================================================================== */

function save(){
  try{ localStorage.setItem('flashie.creator', JSON.stringify({W, textMode: C.textMode, text: C.text, uid})); }catch{}
}
function restore(){
  try{
    const s = JSON.parse(localStorage.getItem('flashie.creator') || 'null');
    if(s && s.W && Array.isArray(s.W.stacks)){ W = s.W; C.textMode = !!s.textMode; C.text = s.text || ''; uid = Math.max(uid, s.uid || 1000); return true; }
  }catch{}
  return false;
}

function download(name, text){
  const a = document.createElement('a');
  a.href = URL.createObjectURL(new Blob([text], {type:'text/plain'}));
  a.download = name; a.click();
  setTimeout(() => URL.revokeObjectURL(a.href), 2000);
}

/* ======================================================================
   Board actions: check, install MicroPython, upload & run, Arduino build
   ====================================================================== */

let DRIVERS = null;
async function drivers(){ if(!DRIVERS) DRIVERS = await api('/api/creator/drivers'); return DRIVERS; }

function cStatus(text, kind){
  const el = $c('#cStat');
  el.className = 'cstat ' + (kind || '');
  el.innerHTML = (kind === 'run' ? ICON.spin : kind === 'ok' ? ICON.tick : kind === 'bad' ? ICON.x : ICON.info) + `<span>${esc(text)}</span>`;
}

function setBusy(on){
  C.busy = on;
  ['#cRun', '#cCheck'].forEach(s => { const b = $c(s); if(b) b.disabled = on; });
}

function consoleAppend(text, src){
  const el = $c('#cCon');
  const atBottom = el.scrollTop + el.clientHeight > el.scrollHeight - 30;
  const span = document.createElement('span');
  span.className = 'c-' + (src || 'board');
  span.textContent = text.replace(/\r\n/g, '\n').replace(/\r/g, '');
  el.appendChild(span);
  while(el.childNodes.length > 1500) el.removeChild(el.firstChild);
  if(atBottom) el.scrollTop = el.scrollHeight;
}
function showTab(t){
  document.querySelectorAll('#cTabs button').forEach(b => b.classList.toggle('on', b.dataset.t === t));
  $c('#cTeach').classList.toggle('hidden', t !== 'teach');
  $c('#cConWrap').classList.toggle('hidden', t !== 'con');
}

async function checkBoard(quiet){
  const ctx = C.ctx;
  if(!S.port){ cStatus('Pick your board’s port on the Flash tab first.', 'bad'); return null; }
  if(ctx.lang !== 'py') return null;
  setBusy(true);
  cStatus('Looking for MicroPython on ' + S.port + '…', 'run');
  try{
    const r = await api('/api/creator/check', {method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify({port: S.port})});
    C.mpy[S.port] = r;
    if(r.micropython) cStatus(`MicroPython ${r.version} found (${r.machine || r.platform}). The console is connected.`, 'ok');
    else cStatus('MicroPython isn’t installed on this board yet.', quiet ? '' : 'bad');
    renderTop();
    return r;
  }catch(e){ cStatus(e.message, 'bad'); return null; }
  finally{ setBusy(false); }
}

function waitForDone(types){
  return new Promise(res => { C.waitDone = {types, res}; });
}

async function installMicroPython(){
  const ctx = C.ctx;
  if(ctx.generic){ cStatus('Pick your board on the Flash tab (or press Identify chip) so Flashie installs the right MicroPython.', 'bad'); return false; }
  if(!confirm(`Install MicroPython on ${ctx.boardName}?\n\nThis replaces the firmware that is on the board now. Flashie downloads the official MicroPython for ${ctx.chip}, flashes it, then uploads your program.`)) return false;
  setBusy(true); showTab('con');
  cStatus('Installing MicroPython (download + flash)…', 'run');
  consoleAppend('\n» Installing MicroPython for ' + ctx.chip + '…\n', 'info');
  const before = new Set(S.ports.map(p => p.name));
  try{
    await api('/api/library/flash', {method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify({item:'micropython', port:S.port, board: libBoardId(), chip: libChip(), baud: 460800, flashMB: 4, compress: true})});
  }catch(e){ setBusy(false); cStatus(e.message, 'bad'); return false; }
  const res = await waitForDone(['done']);
  if(!res.ok){ setBusy(false); cStatus('MicroPython install failed: ' + res.text, 'bad'); return false; }
  cStatus('MicroPython installed — waiting for the board to restart…', 'run');
  // Picos and native-USB boards come back on a new port.
  for(let i = 0; i < 20; i++){
    await new Promise(r => setTimeout(r, 500));
    await loadPorts(true);
    const fresh = S.ports.find(p => !before.has(p.name) && p.kind === 'serial');
    if(fresh){ S.port = fresh.name; renderPorts(); update(); break; }
    if(i > 4 && S.ports.some(p => p.name === S.port && p.kind === 'serial')) break;
  }
  setBusy(false);
  return true;
}

async function runProgram(){
  const ctx = C.ctx;
  showTab('con');
  if(!S.port){ cStatus('Pick your board’s port on the Flash tab first.', 'bad'); return; }
  const code = C.textMode ? C.text : C.gen.code;
  if(!C.textMode && C.gen.errs.length){ cStatus('Fix the problems shown under the blocks first.', 'bad'); return; }
  if(ctx.lang === 'cpp') return arduinoRun(code);
  if(!ctx.lang){ cStatus(`${ctx.boardName} can’t run MicroPython or Arduino code from the Creator.`, 'bad'); return; }
  const p = curPort();
  let info = p && p.kind === 'drive' ? {micropython:false} : await checkBoard(true);
  if(!info) return;
  if(!info.micropython){
    if(!await installMicroPython()) return;
    await new Promise(r => setTimeout(r, 1500));
    info = await checkBoard(true);
    if(!info || !info.micropython){ cStatus('MicroPython was installed, but the board isn’t answering yet. Unplug it, plug it back in, then press Upload & run again.', 'bad'); return; }
  }
  const files = {'main.py': code};
  const need = C.textMode ? ['flashie_cyd.py','flashie_oled.py'].filter(f => code.includes(f.replace('.py',''))) : Object.keys(C.gen.files);
  if(need.length){ const d = await drivers(); for(const f of need) files[f] = d[f]; }
  setBusy(true);
  $c('#cStop').disabled = false;
  cStatus('Uploading…', 'run');
  try{ await api('/api/creator/run', {method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify({port: S.port, files})}); }
  catch(e){ setBusy(false); cStatus(e.message, 'bad'); return; }
  const res = await waitForDone(['cdone']);
  setBusy(false);
  cStatus(res.text, res.ok ? 'ok' : 'bad');
}

async function arduinoRun(code){
  const ctx = C.ctx;
  if(!ctx.board){ cStatus('Pick your Arduino board (Uno, Nano…) on the Flash tab first.', 'bad'); return; }
  setBusy(true); showTab('con');
  cStatus('Building with arduino-cli (the first build downloads the compiler — a few minutes)…', 'run');
  try{ await api('/api/creator/arduino', {method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify({port: S.port, board: ctx.boardId, code, libs: [...C.gen.libs]})}); }
  catch(e){ setBusy(false); cStatus(e.message, 'bad'); return; }
  const res = await waitForDone(['cdone']);
  setBusy(false);
  cStatus(res.text, res.ok ? 'ok' : 'bad');
}

/* ======================================================================
   Layout + top bar
   ====================================================================== */

function renderTop(){
  const ctx = C.ctx;
  const m = S.port && C.mpy[S.port];
  const langTxt = ctx.lang === 'cpp' ? 'Arduino C++' : ctx.lang === 'py' ? 'MicroPython' : 'Not supported';
  $c('#cBoard').innerHTML = `<b>${esc(ctx.boardName)}</b><span>${esc(S.port || 'no port')} · ${langTxt}${ctx.lang==='py' ? (m ? (m.micropython ? ' ✓ v'+esc(m.version) : ' · not installed') : '') : ''}</span>`;
  $c('#cRunLbl').textContent = ctx.lang === 'cpp' ? 'Compile & upload' : 'Upload & run';
  $c('#cCheck').classList.toggle('hidden', ctx.lang !== 'py');
  $c('#cConIn').placeholder = ctx.lang === 'cpp' ? 'Type a line to send to the board’s Serial input and press Enter…' : 'Type Python here and press Enter (talks to the board’s REPL)…';
  let banner = '';
  if(ctx.generic) banner = 'No board picked — blocks and pins are for a generic ESP32. Pick your board on the Flash tab for its real pinout.';
  else if(!ctx.lang) banner = `${ctx.boardName} can’t run MicroPython, and Flashie can’t build Arduino code for it. The Creator supports ESP32, ESP8266, Raspberry Pi Pico and Arduino AVR boards.`;
  else if(ctx.lang === 'cpp') banner = `${ctx.boardName} is too small to run MicroPython (it has only ${/2560/.test(ctx.chip)?'8':'2'} KB of RAM), so Flashie turns your blocks into Arduino C++ instead and compiles it with arduino-cli (downloaded automatically the first time). Wi-Fi, file and CYD-screen blocks aren’t available.`;
  $c('#cBanner').innerHTML = banner ? note(ctx.lang ? 'info' : 'warn', esc(banner)) : '';
}

function buildLayout(){
  const app = document.querySelector('.app');
  const host = document.createElement('section');
  host.id = 'creatorView'; host.className = 'creator hidden';
  host.innerHTML = `
  <div class="ctool">
    <select id="cTpl" class="fsel"><option value="">Start from a template…</option>${Object.entries(TEMPLATES).map(([k, t]) => `<option value="${k}">${esc(t.name)}</option>`).join('')}</select>
    <div class="cboard" id="cBoard"></div>
    <div class="spacer"></div>
    <label class="teachsw" title="Teaching mode: explanations follow your mouse"><input type="checkbox" id="cTeachSw" checked><span class="sw"></span>Teaching mode</label>
    <div class="seg mini" id="cMode"><button data-m="blocks" class="on">Blocks</button><button data-m="text">Type code</button></div>
    <div class="cmenu"><button class="btn" id="cMore">⋯ File</button><div class="cmenu-pop hidden" id="cMoreMenu">
      <button data-a="savecode">Save code to a file</button><button data-a="saveproj">Save project (.flashie)</button><button data-a="openproj">Open project…</button><button data-a="tidy">Tidy up blocks</button><button data-a="clear">Clear everything</button></div></div>
    <button class="btn" id="cCheck">Check board</button>
    <button class="btn" id="cStop" disabled>■ Stop</button>
    <button class="flashbtn small" id="cRun"><svg viewBox="0 0 24 24" fill="#fff"><path d="M7 4v16l13-8z"/></svg><span id="cRunLbl">Upload & run</span></button>
  </div>
  <div id="cBanner"></div>
  <div class="cbody" id="cBody">
    <aside class="cpal" id="cPal"></aside>
    <div class="cmid">
      <div class="ccanvas" id="cCanvas"></div>
      <div class="cwarn" id="cWarn"></div>
      <div class="cfrozen">You’re typing code by hand — the blocks are paused. <button class="btn" id="cBack">Back to blocks</button></div>
    </div>
    <div class="cright">
      <div class="codeh"><span id="cFile">main.py</span><div class="spacer"></div><button class="btn ghost" id="cCopy">Copy</button></div>
      <div class="codev" id="cCode"></div>
      <div class="edwrap hidden" id="cEdWrap"><pre class="edno" id="cEdNo"></pre><div class="edstack"><pre class="edhl" id="cEdHl"></pre><textarea id="cEd" spellcheck="false" autocomplete="off" wrap="off"></textarea></div></div>
      <div class="ctabs" id="cTabs"><button data-t="teach" class="on">💡 Explain</button><button data-t="con">Console</button><div class="spacer"></div><div class="cstat" id="cStat"></div></div>
      <div class="teach" id="cTeach"></div>
      <div class="conwrap hidden" id="cConWrap"><pre class="con" id="cCon"></pre>
        <div class="conin"><input id="cConIn" placeholder="Type Python here and press Enter (talks to the board’s REPL)…" autocomplete="off"><button class="btn" id="cCtrlC" title="Interrupt the running program">Ctrl-C</button><button class="btn ghost" id="cClr">Clear</button></div></div>
    </div>
  </div>
  <input type="file" id="cOpenFile" accept=".flashie,.json" hidden>`;
  app.insertBefore(host, document.querySelector('footer'));

  // view tabs in the header
  const tabs = document.createElement('div');
  tabs.className = 'viewtabs';
  tabs.innerHTML = `<button data-v="flash" class="on">⚡ Flash</button><button data-v="create">🧩 Create</button><button data-v="image">💾 Write image</button><button data-v="keyboard">⌨ Keyboard</button>`;
  document.querySelector('header .spacer').before(tabs);
  tabs.onclick = e => { const b = e.target.closest('button'); if(b) setView(b.dataset.v); };
}

function setView(v){
  C.view = v;
  document.querySelectorAll('.viewtabs button').forEach(b => b.classList.toggle('on', b.dataset.v === v));
  const flashEls = ['.stepper', 'main.grid', '#cydPanel'];
  flashEls.forEach(s => { const el = document.querySelector(s); if(el) el.classList.toggle('creator-hide', v !== 'flash'); });
  $c('#creatorView').classList.toggle('hidden', v !== 'create');
  if(window.imgView) window.imgView(v === 'image');
  if(window.kbdView) window.kbdView(v === 'keyboard');
  document.querySelector('.app').classList.toggle('wide', v === 'create' || v === 'keyboard');
  if(v === 'create') renderAll();
  try{ localStorage.setItem('flashie.view', v); }catch{}
}

function wire(){
  const cv = $c('#cCanvas');
  document.addEventListener('pointerdown', e => { if(e.target.closest('#cCanvas, #cPal')) pointerDown(e); if(!e.target.closest('#cPop, .fpin, .fcolor')) closePop(); if(!e.target.closest('.cmenu')) $c('#cMoreMenu').classList.add('hidden'); });
  $c('#cPal').addEventListener('click', e => {
    const c = e.target.closest('.pcat'); if(c){ C.cat = c.dataset.cat; renderPalette(); return; }
    if(e.target.id === 'cNewVar'){ const n = prompt('Name for the new variable (for example: score, count, speed):'); if(n && n.trim()){ if(!W.vars.includes(n.trim())) W.vars.push(n.trim()); renderPalette(); regen(); } }
  });
  $c('#cPal').addEventListener('mouseover', e => { if(!C.teach || DRAG) return; const p = e.target.closest('.pitem'); if(p){ const b = make(p.dataset.type); teachBlockDef(b, p.classList.contains('na')); } });
  cv.addEventListener('input', e => {
    const inp = e.target.closest('.fin'); if(!inp) return;
    const b = IDX[inp.closest('.blk').dataset.id]; if(!b) return;
    b.b.f[inp.dataset.f] = inp.value; inp.size = Math.max(inp.classList.contains('num') ? 2 : 4, inp.value.length);
    C.gen = generate(C.ctx); renderCode(); renderWarnings(); save();
  });
  cv.addEventListener('change', e => {
    const s = e.target.closest('.fdrop'); if(!s) return;
    const b = IDX[s.closest('.blk').dataset.id]; if(!b) return;
    b.b.f[s.dataset.f] = s.value; regen(); teachBlock(b.b);
  });
  cv.addEventListener('click', e => {
    const pb = e.target.closest('.fpin'), cb = e.target.closest('.fcolor');
    if(!pb && !cb) return;
    const b = IDX[e.target.closest('.blk').dataset.id]; if(!b) return;
    if(pb) openPinPicker(pb, b.b, pb.dataset.f, pb.dataset.mode);
    if(cb) openColor(cb, b.b, cb.dataset.f);
  });
  cv.addEventListener('mouseover', e => {
    if(DRAG && DRAG.started) return;
    const blk = e.target.closest('.blk');
    const id = blk ? blk.dataset.id : null;
    if(id === C.hoverBid) return;
    C.hoverBid = id;
    document.querySelectorAll('#cCode .ln.hl').forEach(l => l.classList.remove('hl'));
    if(id){
      const ids = new Set([id]);
      const loc = IDX[id];
      if(loc){ const add = b => { ids.add(b.id); Object.values(b.in).forEach(x => x && add(x)); b.sub.forEach(add); b.sub2.forEach(add); }; add(loc.b); }
      let first = null;
      document.querySelectorAll('#cCode .ln').forEach(l => { if(ids.has(l.dataset.bid)){ l.classList.add('hl'); first = first || l; } });
      if(first) first.scrollIntoView({block: 'nearest'});
      if(C.teach && loc) teachBlock(loc.b);
    }
  });
  $c('#cCode').addEventListener('mouseover', e => {
    const l = e.target.closest('.ln'); if(!l) return;
    document.querySelectorAll('#cCanvas .blk.hl').forEach(b => b.classList.remove('hl'));
    if(l.dataset.bid){ const b = document.querySelector(`#cCanvas .blk[data-id="${l.dataset.bid}"]`); if(b) b.classList.add('hl'); }
    if(C.teach) teachLine(+l.dataset.i);
  });
  $c('#cCode').addEventListener('mouseleave', () => document.querySelectorAll('#cCanvas .blk.hl').forEach(b => b.classList.remove('hl')));
  $c('#cCode').addEventListener('click', e => { const l = e.target.closest('.ln'); if(l){ teachLine(+l.dataset.i); showTab('teach'); } });
  $c('#cWarn').addEventListener('click', e => { const w = e.target.closest('[data-bid]'); if(w){ C.sel = w.dataset.bid; regen(); const loc = IDX[w.dataset.bid]; if(loc){ teachBlock(loc.b); showTab('teach'); } } });
  document.addEventListener('keydown', e => {
    if(C.view !== 'create') return;
    if((e.key === 'Delete' || e.key === 'Backspace') && C.sel && !e.target.closest('input, textarea, select')){ e.preventDefault(); deleteSelected(); }
  });
  $c('#cTpl').onchange = e => { if(e.target.value){ const has = W.stacks.length && countBlocks(W.stacks.flatMap(s=>s.blocks)) > 1; if(!has || confirm('Replace your current blocks with this template?')) loadTemplate(e.target.value); e.target.value = ''; } };
  $c('#cTeachSw').onchange = e => { C.teach = e.target.checked; };
  $c('#cMode').onclick = e => { const b = e.target.closest('button'); if(b) setTextMode(b.dataset.m === 'text'); };
  $c('#cBack').onclick = () => setTextMode(false);
  $c('#cCopy').onclick = async () => { const t = C.textMode ? C.text : C.gen.code;
    try{ await navigator.clipboard.writeText(t); }catch{ const x = document.createElement('textarea'); x.value = t; document.body.appendChild(x); x.select(); document.execCommand('copy'); x.remove(); }
    toast('Code copied to the clipboard.'); };
  $c('#cTabs').onclick = e => { const b = e.target.closest('button'); if(b) showTab(b.dataset.t); };
  const ta = $c('#cEd');
  ta.addEventListener('input', () => { C.text = ta.value; paintEditor(); save(); });
  ta.addEventListener('scroll', () => { $c('#cEdHl').scrollTop = ta.scrollTop; $c('#cEdHl').scrollLeft = ta.scrollLeft; $c('#cEdNo').scrollTop = ta.scrollTop; });
  ta.addEventListener('keydown', e => {
    if(e.key === 'Tab'){ e.preventDefault(); const s = ta.selectionStart; ta.setRangeText('    ', s, ta.selectionEnd, 'end'); C.text = ta.value; paintEditor(); }
    if(e.key === 'Enter'){ // keep indentation, add a level after ':'
      e.preventDefault();
      const s = ta.selectionStart, line = ta.value.slice(0, s).split('\n').pop();
      let indent = line.match(/^\s*/)[0]; if(/:\s*$/.test(line) || /\{\s*$/.test(line)) indent += '    ';
      ta.setRangeText('\n' + indent, s, ta.selectionEnd, 'end'); C.text = ta.value; paintEditor();
    }
  });
  $c('#cRun').onclick = runProgram;
  $c('#cCheck').onclick = () => { showTab('con'); checkBoard(false); };
  $c('#cStop').onclick = () => api('/api/creator/stop', {method:'POST'}).then(() => cStatus('Sent Ctrl-C — the program was interrupted.', '')).catch(e => cStatus(e.message, 'bad'));
  $c('#cCtrlC').onclick = () => api('/api/creator/stop', {method:'POST'}).catch(e => consoleAppend('\n' + e.message + '\n', 'info'));
  $c('#cClr').onclick = () => { $c('#cCon').innerHTML = ''; };
  $c('#cConIn').addEventListener('keydown', e => {
    if(e.key !== 'Enter') return;
    const t = e.target.value; e.target.value = '';
    api('/api/creator/send', {method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify({text: t + '\r'})}).catch(err => consoleAppend('\n' + err.message + '\n', 'info'));
  });
  $c('#cMore').onclick = () => $c('#cMoreMenu').classList.toggle('hidden');
  $c('#cMoreMenu').onclick = e => {
    const a = e.target.dataset.a; if(!a) return;
    $c('#cMoreMenu').classList.add('hidden');
    if(a === 'savecode') download(C.ctx.lang === 'cpp' ? 'flashie_sketch.ino' : 'main.py', C.textMode ? C.text : C.gen.code);
    if(a === 'saveproj') download('my-program.flashie', JSON.stringify({flashie: 1, by: 'Flashie Firmware Creator (nrmluvdev)', W, textMode: C.textMode, text: C.text}, null, 1));
    if(a === 'openproj') $c('#cOpenFile').click();
    if(a === 'tidy') tidy();
    if(a === 'clear' && confirm('Remove all blocks?')){ W = {stacks: [], vars: ['count']}; C.textMode = false; regen(); }
  };
  $c('#cOpenFile').onchange = async e => {
    const f = e.target.files[0]; e.target.value = ''; if(!f) return;
    try{ const j = JSON.parse(await f.text()); if(!j.W) throw 0; W = j.W; C.textMode = !!j.textMode; C.text = j.text || '';
      const ids = []; const walk = l => l.forEach(b => { ids.push(+String(b.id).slice(1)||0); Object.values(b.in).forEach(x => x && walk([x])); walk(b.sub); walk(b.sub2); });
      W.stacks.forEach(s => walk(s.blocks)); uid = Math.max(uid, ...ids) + 1; renderAll(); toast('Project opened.'); }
    catch{ toast('That file isn’t a Flashie project.', true); }
  };
  // SSE for creator jobs
  const es = new EventSource('/api/events');
  es.onmessage = ev => {
    const m = JSON.parse(ev.data);
    if(m.type === 'console') consoleAppend(m.text, m.src);
    else if(m.type === 'cstatus') cStatus(m.text, 'run');
    else if(m.type === 'cprogress') cStatus(`Copying… ${Math.floor(m.done*100/Math.max(1,m.total))}%`, 'run');
    else if(m.type === 'csession'){ C.session = m.open; $c('#cStop').disabled = !m.open; }
    else if(m.type === 'status' && C.waitDone) cStatus(m.text, 'run');
    else if(m.type === 'log' && C.waitDone) consoleAppend(m.text + '\n', 'info');
    else if(m.type === 'progress' && C.waitDone) cStatus(`Installing MicroPython… ${Math.floor(m.done*100/Math.max(1,m.total))}%`, 'run');
    if(C.waitDone && C.waitDone.types.includes(m.type)){ const w = C.waitDone; C.waitDone = null; w.res(m); }
  };
  // follow board changes on the Flash tab
  setInterval(() => {
    if(C.view !== 'create' || DRAG) return;
    const k = libKey() + '|' + S.port + '|' + (S.chip ? S.chip.chip : '');
    if(k !== C.key){ C.key = k; renderAll(); }
  }, 1000);
}

function teachBlockDef(b, na){
  const d = B[b.t];
  $c('#cTeach').innerHTML = `<div class="tb"><span class="chip3" style="background:${catOf(d.cat).color}">${esc(catOf(d.cat).name)}</span><h4>${esc(plainLabel(b))}</h4></div><p>${esc(d.ex)}</p>` +
    (na ? note('warn', esc(blockAvailable(b.t, C.ctx))) : `<p class="src">Drag it onto the canvas to use it.</p>`);
}

/* ---------------------------------------------------------------- boot */
(function initCreator(){
  buildLayout();
  wire();
  if(!restore()){
    W = {stacks: [], vars: ['count']};
  }
  C.ctx = creatorCtx();
  if(!W.stacks.length){ const T = TEMPLATES.blink; W.stacks = [{x: 40, y: 30, blocks: T.build(C.ctx)}]; W.stacks[0].blocks.forEach(b => fillPins(b, C.ctx)); }
  teachMsg('Welcome to the Firmware Creator', 'Build a program by dragging blocks from the left and snapping them together under “when the board starts”. The real code appears on the right as you build — hover over any block or any line of code to see what it means. When you’re ready, press “Upload & run”.');
  cStatus('Ready.', '');
  if(C.textMode) setTimeout(() => setTextMode(true), 0);
  window.flashieSetView = setView;
  try{ const v = localStorage.getItem('flashie.view'); if(v === 'create' || v === 'keyboard') setView(v); }catch{}
})();

// Exposed for tests.
window.FlashieCreator = {generate, creatorCtx, TEMPLATES, B, make, fillPins, pinDB, pinWarn, setW: w => { W = w; }, getW: () => W};
