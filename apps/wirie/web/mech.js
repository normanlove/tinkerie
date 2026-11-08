// Wirie → Makie mechanical export ("wirie-board-mech" v1) — by nrmluvdev
// Describes the board as solid geometry so Makie can build cases, lids,
// standoffs and cut-outs automatically. Spec: see MAKIE_FORMAT.md.
"use strict";

const ELEC_H = { 4: 7, 5: 11, 6.3: 11, 8: 12, 10: 16, 12.5: 20, 16: 25, 18: 35 };
// estimated height of the part above the board surface (mm)
function partHeight(fp) {
  const id = fp.id, b = fp.body;
  const rules = [
    [/^hdr2mm/, 6], [/^hdr127/, 5], [/^hdr_smd/, 8.5], [/^hdr/, 8.5], [/^idc/, 9], [/^jstxh/, 7], [/^jstph/, 6], [/^jstzh/, 4.5], [/^jstsh/, 3], [/^term5/, 10], [/^term35/, 8.5], [/^microfit/, 9.8], [/^atx24/, 13],
    [/^r_axial|^r_pow_1W/, 3], [/^r_pow_2W/, 5], [/^r_pow_5W/, 10], [/^r_vert/, 11], [/^rnet_sip/, 5], [/^trim3296/, 10], [/^pot16/, 25], [/^rv09/, 18], [/^ec11/, 20],
    [/^c_elec_smd_(\d+)/, m => (+m[1] >= 10 ? 10 : +m[1] >= 8 ? 6.5 : 5.4)], [/^c_elec_([\d.]+)/, m => ELEC_H[m[1]] || 12], [/^c_film_([\d.]+)/, m => ({ 5: 7, 7.5: 9, 10: 11, 15: 14 }[m[1]] || 10)], [/^c_disc|^c_mlcc_rad/, 5], [/^supercap/, 6], [/^c_tant/, 2], [/^l_axial/, 4], [/^l_radial/, 10], [/^l_toroid/, 8], [/^l_pow_(\d+)/, m => (+m[1] >= 10 ? 5 : 3)],
    [/^led5|^photodiode5|^led_rgb5/, 8.6], [/^led3/, 5.3], [/^ldr5/, 4], [/^ntc/, 5], [/^ws2812b$|^led5050|^sk6812/, 1.6], [/^led/, 0.8],
    [/^d_do41|^d_do35/, 3], [/^d_do201/, 5.5], [/^bridge_kbp/, 18], [/^bridge_db/, 4], [/^to92|^ds18b20|^lm35|^a3144/, 5], [/^reg78|^reglm317|^q_to220/, 19], [/^to247/, 21], [/^dpak|^d2pak|^lm2596/, 2.5],
    [/^relay_srd/, 15.5], [/^buzzer/, 9.5], [/^fuse/, 4], [/^xtal_hc49$/, 3.5], [/^xtal_32k_tht/, 2.2], [/^resonator/, 6], [/^osc_dip/, 5.5], [/^xtal/, 1], [/^osc/, 1],
    [/^sw6x6$/, 5], [/^sw12x12/, 7.5], [/^sw6x6smd/, 4.3], [/^sw4x4/, 1.5], [/^sw3x6/, 2.5], [/^mx_switch/, 11.6], [/^choc/, 5.8], [/^toggle/, 18], [/^slide_ss/, 8], [/^slide_smd/, 2], [/^microswitch/, 6.5], [/^dip_sw/, 6], [/^sw_slide3/, 8],
    [/^usbc/, 3.3], [/^musb_smd/, 2.6], [/^usba/, 7], [/^dcjack/, 11], [/^jack35$/, 6], [/^rj45/, 13.5], [/^rca/, 13], [/^minidin/, 13], [/^de9|^de15|^db25/, 12.5], [/^hdmi/, 6], [/^sd_socket/, 2], [/^xt30/, 9], [/^xt60/, 16],
    [/^fpc|^ffc|^picam|^pidsi/, 2.2], [/^pcie|^pci_|^isa|_cart$/, 0], [/^edgesock/, 10], [/^carbon/, 0], [/^zif/, 14],
    [/^bat18650/, 21], [/^cr2032/, 5.5], [/^isp|^swd10|^jtag/, 9], [/^mh_|^pi_hat_holes|^pi_zero_holes|^fiducial|^tp$|^wirepad|^logo/, 0],
    [/^(dip|atmega328p$|attiny85$|ne555$|lm358$|hc595$|uln2003$|l293d|pc817|cd4017|mcp23017$|lm324$|hc165|hc4051|atmega32_dip|cpu|eprom|flash39|sram|sn76489|ay38910|sid6581|ym|cic|tms|cia|ic74)/, 5], [/^sdip/, 5],
    [/^soic|^ssop|^ch340|^w25q|^eeprom24|^max485|^ne555s|^lm358s|^lm324s|^hc595s|^uln2003s|^pcf8574s|^ds3231s|^mcp23017s|^tp4056s|^attiny85s/, 1.75], [/^tssop|^msop|^ads1115s/, 1.1], [/^qfn/, 0.9], [/^lqfp|^tqfp|^atmega328pau/, 1.6], [/^plcc/, 4.5],
    [/^sot|^ams1117|^xc6206|^mcp1700|^ap2112/, 1.6], [/^[rcl]_\d{4}$|^d_sod|^d_sma|^d_smb|^d_smc|^d_mini|^rarray/, 0.8],
    [/^pico_smd|^esp32wroom|^esp12f/, 3.2],
  ];
  for (const [re, h] of rules) { const m = id.match(re); if (m) return typeof h === "function" ? h(m) : h; }
  if (b.module) return fp.cat === "Microcontrollers" || fp.cat === "Displays & LEDs" ? 13 : 12; // plugged into a socket header
  return fp.smd ? 1.2 : 4;
}
// suggested case openings, in the part's own coordinates (top view, mm)
function partOpenings(fp) {
  const id = fp.id, b = fp.body; const o = [];
  const side = () => o.push({ face: "side" });
  const circ = (d, x = b.x, y = b.y) => o.push({ face: "top", shape: "circle", diameter: +d.toFixed(2), x, y });
  const rect = (w, h, x = b.x, y = b.y) => o.push({ face: "top", shape: "rect", width: +w.toFixed(2), height: +h.toFixed(2), x, y });
  if (/^(usbc|musb|usba|dcjack|jack35$|rj45|rca|minidin|de9|de15|db25|hdmi|sd_socket|usbc_bo|musb_bo|xt30|xt60|atx24)/.test(id)) side();
  else if (/^(pcie|pci_|isa8)|_cart$/.test(id)) o.push({ face: "side", kind: "card-edge" });
  else if (/^(led5|led3|led_rgb5|photodiode5|ldr5)/.test(id)) circ(b.w + 0.4);
  else if (/^(ws2812b|led_|led3528|led5050|sk6812|ws2812b_2020)/.test(id)) circ(Math.max(2.5, Math.min(b.w, b.h) + 0.5));
  else if (/^(sw6x6$|sw6x6smd)/.test(id)) circ(3.8); else if (/^sw12x12/.test(id)) circ(7.5); else if (/^(sw4x4|sw3x6)/.test(id)) circ(2.6);
  else if (/^arcade/.test(id)) circ(30); else if (/^mx_switch/.test(id)) rect(14, 14); else if (/^choc/.test(id)) rect(13.8, 13.8);
  else if (/^(pot16|ec11)/.test(id)) circ(7.5, 0, 0); else if (/^rv09/.test(id)) circ(6.5, b.x, b.y); else if (/^toggle/.test(id)) circ(6.5, 0, 0);
  else if (/^(slide_ss|sw_slide3)/.test(id)) rect(4.5, 2.5, 0, 0); else if (/^buzzer/.test(id)) circ(3);
  else if (id === "pir") circ(23.5); else if (id === "hcsr04") { circ(16.5, b.x - 13, b.y); circ(16.5, b.x + 13, b.y); }
  else if (/^(max4466|sound_mod|inmp441)/.test(id)) circ(3); else if (/^(bme280|aht20|dht22|scd40|ccs811|bmp180)/.test(id)) rect(Math.min(b.w, 10), Math.min(b.h, 8));
  else if (/^(ldrmod|bh1750|veml7700|apds9960|max30102|vl53|tcrt5000|ir_obstacle|ir_rx|flame)/.test(id)) circ(5);
  else if (fp.cat === "Displays & LEDs" && b.module && !/^ws2812_panel|^ht16k33|^max7219$/.test(id)) rect(b.w * 0.86, b.h * (/^lcd|epaper/.test(id) ? 0.62 : 0.66), b.x, b.y);
  else if (/^(seg7|matrix8x8|ledbar10|max7219)/.test(id)) rect(b.w, b.h);
  else if (/^carbon/.test(id)) circ(b.w);
  return o;
}
