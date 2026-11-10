// Scopie — beginner-friendly explanations for every control.
// Each entry is [title, explanation]. They show as tooltips and, in larger
// text, in the help bar under the screen. Created by nrmluvdev.
"use strict";

const TIPS = {
  // ---------------------------------------------------------------- toolbar
  run: ["Run / Stop",
    "Starts or freezes the display. While stopped, the last waveform stays on the screen so you can study it, move the cursors and read the measurements.\nShortcut: Space."],
  single: ["Single shot",
    "Waits for ONE trigger event, captures it, then stops. Perfect for catching a one-off event such as a button press or a power-on surge.\nShortcut: S."],
  autoset: ["Auto setup",
    "Lets the program look at your signal and choose sensible volts/div, time/div and trigger settings for you. A great first step whenever you connect a new signal.\nShortcut: A."],
  default: ["Default setup",
    "Puts every setting back to a known starting point. Use it when the screen looks confusing and you want to start again."],
  clear: ["Clear display", "Wipes persistence and averaging history so they start building up again."],
  screenshot: ["Save screenshot",
    "Saves a picture (PNG) of the scope screen into Documents\\Scopie, for reports or sharing.\nShortcut: Ctrl+P."],
  codie: ["Explain in Codie",
    "Saves the waveform on screen and opens it in Codie, which explains it in plain words: what each channel is (a sine wave, PWM, a servo signal, a supply rail…), its frequency and levels, any serial data it carries, and anything worth checking, such as clipping or slow edges."],
  csv: ["Export data (CSV)",
    "Saves the samples currently on screen as a spreadsheet file (time and voltage columns) that Excel, LibreOffice or Python can open. Files go into Documents\\Scopie.\nShortcut: Ctrl+S."],
  guide: ["Beginner's guide", "A short introduction to using an oscilloscope. Shortcut: F1."],
  menu_file: ["File", "Open and save waveforms, screenshots and setups."],
  menu_device: ["Device", "Connect a scope, try demo mode, or install USB drivers."],
  menu_help: ["Help", "The beginner's guide, supported scopes, licences and more."],

  // ---------------------------------------------------------------- channels
  ch_enable: ["Show channel",
    "Turns this input channel on or off. Switching CH2 off lets the scope use a higher sample rate on the remaining channel(s) on most USB scopes."],
  vdiv: ["Volts per division (vertical scale)",
    "How many volts each square of the grid represents vertically. Smaller values zoom in on small signals; bigger values fit large signals on screen.\nAim for a waveform that fills most of the screen height without going off the top or bottom."],
  probe: ["Probe attenuation",
    "Must match the switch on your probe. A x10 probe divides the voltage by 10 (it protects the scope and loads your circuit less); this setting multiplies the readings back up so the voltages shown are correct.\nIf your readings are exactly 10 times too big or small, this is the setting to check."],
  coupling: ["Input coupling",
    "DC shows the whole signal including any steady (DC) voltage.\nAC removes the steady part so you only see the changes - handy for looking at a small ripple sitting on top of a large DC voltage.\n(Scopes without AC hardware, such as the Hantek 6022BE, do AC coupling in software.)"],
  pos: ["Vertical position",
    "Moves this trace up or down the screen. It does not change the signal, only where it is drawn. You can also drag the coloured 0 V marker at the left of the screen."],
  pos_reset: ["Centre trace", "Puts the 0 V level of this channel back in the middle."],
  invert: ["Invert",
    "Flips the trace upside down. Useful when a signal is measured 'backwards' or to compare two signals that are opposites."],

  // ---------------------------------------------------------------- horizontal
  tdiv: ["Time per division (horizontal scale)",
    "How much time each square of the grid represents horizontally. Smaller values zoom in to see fast details; bigger values show slow changes.\nA good rule: show 2 to 5 cycles of a repeating signal."],
  trig_pos: ["Trigger position (horizontal position)",
    "Where on screen the trigger moment is placed. In the middle you see equal amounts of what happened before and after the trigger. Move it left to see more of what happened afterwards.\nYou can also drag the orange T marker at the top of the screen."],
  rate: ["Sample rate",
    "How many times per second the scope measures the voltage.\n'Auto' picks a rate to suit the time/div (recommended). Higher rates show faster detail, but the signal must change slower than half the sample rate or you will see misleading 'aliased' waveforms.\nOnly shown for USB scopes that stream samples; bench scopes, PicoScopes and handhelds choose their own rate from the time/div."],
  roll: ["Roll mode",
    "At slow time/div settings (100 ms/div and slower) the trace scrolls from right to left like a chart recorder, instead of waiting to fill the screen. Ideal for slowly changing things like a charging capacitor or a temperature sensor."],

  // ---------------------------------------------------------------- trigger
  trig_mode: ["Trigger mode",
    "AUTO: draws the trace even when nothing triggers, so you always see something (best for beginners).\nNORMAL: only draws when the trigger condition happens - the screen stays still until a matching edge arrives. Use it for rare or irregular events."],
  trig_source: ["Trigger source",
    "Which channel the scope watches to decide when to start drawing. Pick the channel with the clearest, most regular signal."],
  trig_edge: ["Trigger edge (slope)",
    "RISING: trigger when the signal crosses the level going UP.\nFALLING: trigger when it crosses going DOWN.\nEITHER: trigger on both."],
  trig_level: ["Trigger level",
    "The voltage the signal must cross to trigger. It must be somewhere between the signal's lowest and highest points, otherwise the scope never triggers and the picture drifts.\nYou can also drag the orange dashed line on the screen."],
  trig_50: ["Set level to 50 %",
    "Puts the trigger level exactly halfway between the signal's lowest and highest points - usually the most stable choice."],
  holdoff: ["Trigger holdoff",
    "After each trigger, ignore further triggers for this long. Helps make complex repeating patterns (like data bursts) stand still."],
  noise_reject: ["Noise reject",
    "Requires a bigger swing before the trigger re-arms, so noise wobbling around the level does not cause false triggers. Turn it on if the trace jitters left and right."],

  // ---------------------------------------------------------------- math
  math_enable: ["Math channel",
    "Shows an extra, purple trace calculated from two channels. For example CH1 - CH2 shows the voltage ACROSS a component when each probe is on one side of it."],
  math_a: ["Math input A", "The first channel used by the math trace."],
  math_b: ["Math input B", "The second channel used by the math trace."],
  math_op: ["Math operation",
    "How the two channels are combined to make the math trace. CH1 x CH2 can show power when one channel measures voltage and the other current."],
  math_vdiv: ["Math scale", "Vertical scale of the math trace, in units per division."],
  math_pos: ["Math position", "Moves the math trace up or down the screen."],

  // ---------------------------------------------------------------- display
  style: ["Trace style",
    "LINES joins the samples together (normal). DOTS shows only the actual measured samples - useful for understanding what the scope really captured."],
  interp: ["Interpolation",
    "When there are only a few samples across the screen (very fast time/div), Sin(x)/x draws the smooth curve the real signal must have followed. Linear just joins the dots with straight lines."],
  average: ["Averaging",
    "Averages many captured waveforms together. Random noise cancels out, so small repeating signals become clearer. Higher numbers = smoother but slower to react. Needs a stable trigger."],
  persist: ["Persistence",
    "Keeps old waveforms on screen and fades them out slowly, like an analog scope. Thousands of captured waveforms are overlaid, so rare glitches or jitter show up as faint traces. Brighter = happens more often."],
  grid: ["Grid brightness", "How bright the background grid lines are."],
  xy: ["XY mode",
    "Plots CH1 horizontally against CH2 vertically instead of against time. Two sine waves make Lissajous figures, which show frequency ratio and phase."],

  // ---------------------------------------------------------------- FFT
  fft_enable: ["Spectrum (FFT)",
    "Opens a second graph showing which frequencies make up the signal. A pure sine wave is a single spike; a square wave shows a spike at its frequency plus smaller ones at 3x, 5x, 7x..."],
  fft_source: ["Spectrum source", "Which trace the spectrum is calculated from."],
  fft_window: ["FFT window",
    "Shapes the data before the FFT. HANN is a good general choice. FLAT TOP gives the most accurate heights. RECTANGULAR gives the sharpest peaks but leaks energy into nearby frequencies."],
  fft_log: ["Logarithmic frequency axis",
    "Spaces the frequency axis by decades (10 Hz, 100 Hz, 1 kHz...) which is handy for audio work."],
  fft_plot: ["Spectrum display",
    "Height is in dBV: 0 dBV = 1 V RMS, -20 dBV = 0.1 V, -40 dBV = 0.01 V. Drag to pan, use the mouse wheel to zoom and double-click to reset."],

  // ---------------------------------------------------------------- cursors
  cursor_mode: ["Cursors",
    "Draggable green lines for measuring by hand.\nTIME cursors measure the time between two points (and the frequency it corresponds to).\nVOLTAGE cursors measure the voltage difference between two levels."],
  cursor_src: ["Cursor channel", "Which trace the voltage cursors read their values from."],
  cursor_line: ["Cursor", "Drag this line to the point you want to measure. The readings are shown below the screen."],

  // ---------------------------------------------------------------- probe cal / device
  cal_freq: ["Calibration output frequency",
    "The metal tab on the front of the scope outputs a 2 V square wave for testing. Hook a probe onto it to check everything works and to adjust a x10 probe's compensation screw until the corners of the square wave are sharp and flat (not rounded, not spiky)."],
  connect: ["Connect…",
    "Opens the Connect window, which searches USB (and any network addresses you type in) for supported oscilloscopes and lets you pick one.\nShortcut: Ctrl+K."],
  demo: ["Demo mode",
    "Uses built-in simulated signals instead of a real scope, so you can learn every feature without any hardware connected. There is a 2-channel USB-style demo and a 4-channel bench-scope demo."],
  demo_sig: ["Demo signal", "Which simulated signal is 'connected' to this channel in demo mode."],
  hw_acdc: ["Hardware AC/DC mod",
    "Hantek 6022BE/BL only: tick this if your scope has the popular community modification that adds AC coupling capacitors. Unmodified scopes do AC coupling in software."],

  // ---------------------------------------------------------------- readouts
  screen: ["Oscilloscope screen",
    "Time runs left to right, voltage bottom to top. Each square is one 'division'. The coloured numbers on the left and right edges show volts for the first two channels; along the bottom is time relative to the trigger moment.\nDrag the coloured markers to move traces, the orange lines to move the trigger."],
  ground_marker: ["0 V marker", "Shows where 0 volts is for this channel. Drag it up or down to move the trace."],
  trig_line: ["Trigger level line", "The signal must cross this voltage to trigger. Drag it to change the trigger level."],
  trig_pos_line: ["Trigger position",
    "The moment of the trigger (time zero). Drag it left or right to see more before or after the event."],
  measure_table: ["Automatic measurements",
    "Live measurements of each channel. Hover over a measurement name to find out what it means. Use the 'Measurements…' button to choose which ones to show."],
  measure_btn: ["Choose measurements", "Pick which measurements appear in the table."],
  cursor_readout: ["Cursor readings",
    "ΔT is the time between the two time cursors and 1/ΔT the frequency that corresponds to. ΔV is the difference between the voltage cursors."],
  status_state: ["Acquisition state",
    "TRIG'D: locked to the trigger - stable picture.\nAUTO: no trigger found, drawing anyway (the picture may drift - check the trigger level).\nWAIT: Normal mode, waiting for a trigger.\nROLL: roll mode.\nSTOP: frozen.\nFILE: showing a waveform from a file."],
  status_wfms: ["Waveform capture rate",
    "How many complete waveforms per second the scope is capturing and checking. This is much higher than the 60 fps screen refresh: with persistence or averaging, every captured waveform is used."],
  status_fps: ["Display refresh rate", "How many times per second the screen is redrawn."],
  status_usb: ["Data rate", "How much sample data per second is arriving from the scope."],
  status_dev: ["Device", "What the program is connected to. Hover for details."],
  chip_ch: ["Channel settings", "Volts per division, coupling and probe setting of this channel."],
  chip_h: ["Timebase", "Time per division and the sample rate in use."],
  chip_trig: ["Trigger summary", "Source channel, edge direction and level of the trigger."],

  // ---------------------------------------------------------------- connect window
  scan: ["Search again", "Looks again for connected oscilloscopes. Plug the scope in first and give it a few seconds to start."],
  device_list: ["Oscilloscopes found",
    "Every supported oscilloscope that was found. Greyed-out entries need something first - select one to see what."],
  lan: ["Network address",
    "Bench scopes with a LAN socket can be used over the network without any driver. Type the scope's IP address (shown in its Utility / I/O menu), e.g. 192.168.1.50, then click Search again."],
  import_fw: ["Import firmware",
    "Some scopes need firmware we are not allowed to include.\n• Old Hantek DSO-2090/2150/2250/5200: choose the driver file from Hantek's software (e.g. DSO2090X861.SYS) or the driver .zip from hantek.com.\n• Instrustar ISDS-205B / MDSO: choose isds205b-firmware.hex or mdso-firmware.hex from an OpenHantek6022 download.\nThe firmware is remembered for next time."],
  usb_driver: ["Install USB driver (Windows)",
    "USB scopes need the WinUSB driver on Windows. For the Hantek 6022BE/BL there is a one-step signed driver; every other USB scope gets it with Zadig (included)."],
  pico_sdk: ["PicoScope drivers",
    "PicoScopes need Pico Technology's free PicoSDK. Pico's licence doesn't allow other programs to include it, so this opens Pico's download page."],
  open_csv: ["Open waveform file (CSV)",
    "Shows a waveform saved as a CSV file: from this program, or saved on the USB stick / internal storage of a handheld or bench scope. The first column must be time; up to four voltage columns follow.\nShortcut: Ctrl+O. You can also drop a CSV file on the screen."],
  ch_more: ["More channels", "CH3 and CH4 appear when a 4-channel scope is connected."],
};

const DEFAULT_HELP = "Hover the mouse over any control or part of the screen to find out what it does. New to oscilloscopes? Press <b>F1</b> for the beginner's guide, or click <b>Auto setup</b> to get started.";

const GUIDE_HTML = `
<div class="help-hero"><img src="icon.png" alt=""><div><b>Welcome to your oscilloscope</b><br>
An oscilloscope draws a <b>graph of voltage against time</b>. It lets you <i>see</i> electricity: whether a signal is present, how big it is, how fast it changes and what shape it has.</div></div>

<h3>1. Your first waveform (2 minutes)</h3>
<ol>
<li>Plug the scope in (USB, or for bench scopes optionally the network) and click <b>Connect…</b>. Pick your scope from the list.</li>
<li>Connect a probe to the <b>CH1</b> socket.</li>
<li>Set the switch on the probe to <b>x10</b> and choose <b>Probe: x10</b> in the Channel 1 controls.</li>
<li>Clip the probe's <b>ground clip</b> (crocodile clip) to the scope's ground tab, and hook the probe tip onto the <b>calibration tab</b> on the front of the scope.</li>
<li>Click <b>Auto setup</b>. You should see a square wave about 2 V high, repeating 1000 times per second (1 kHz).</li>
</ol>
<p>No scope yet? Choose <b>Demo mode</b> and every feature works with simulated signals.</p>

<h3>2. Reading the screen</h3>
<ul>
<li>The grid is divided into <b>divisions</b>: usually 10 across (some scopes use 12 or 14) and 8 up.</li>
<li><b>Volts/div</b> sets the height of each square. At 1 V/div, a waveform 3 squares high is 3 V peak-to-peak.</li>
<li><b>Time/div</b> sets the width of each square. At 1 ms/div the whole screen shows 10 ms.</li>
<li>The coloured <b>0 V markers</b> on the left show where zero volts is for each channel.</li>
</ul>

<h3>3. Triggering - making the picture stand still</h3>
<p>A repeating signal would scroll across the screen like a blur. The <b>trigger</b> waits for the signal to cross a chosen voltage (the <b>trigger level</b>, orange dashed line) in a chosen direction (<b>rising</b> or <b>falling</b>), and always starts drawing from that moment - so each redraw lines up and the picture stands still.</p>
<p>If the picture drifts, the level is probably above or below the signal: click <b>50 %</b> in the trigger section, or drag the orange line into the middle of the waveform.</p>

<h3>4. Measuring</h3>
<ul>
<li>The <b>measurement table</b> under the screen shows frequency, peak-to-peak voltage and more. Hover over each name for an explanation.</li>
<li><b>Cursors</b> are lines you drag to measure anything by hand.</li>
<li><b>Spectrum (FFT)</b> shows which frequencies the signal contains.</li>
</ul>

<h3>5. Staying safe</h3>
<ul>
<li>On a USB or bench scope the probe's ground clip is connected to <b>earth / your computer's ground</b>. Only ever clip it to the ground (0 V) of your circuit.</li>
<li><b>Never</b> connect it to mains electricity or to anything mains-powered without isolation. Stick to batteries, USB-powered circuits and low-voltage power supplies.</li>
<li>Check your scope's maximum input voltage (for a Hantek 6022BE about &plusmn;5 V with a x1 probe, &plusmn;50 V with a x10 probe). Battery-powered handhelds are safer for floating measurements, but follow their CAT ratings.</li>
</ul>

<h3>Keyboard shortcuts</h3>
<div class="keys">
<span><kbd>Space</kbd> Run / Stop</span><span><kbd>S</kbd> Single</span>
<span><kbd>A</kbd> Auto setup</span><span><kbd>F1</kbd> This guide</span>
<span><kbd>Ctrl</kbd>+<kbd>K</kbd> Connect</span><span><kbd>Ctrl</kbd>+<kbd>O</kbd> Open CSV</span>
<span><kbd>Ctrl</kbd>+<kbd>S</kbd> Export CSV</span><span><kbd>Ctrl</kbd>+<kbd>P</kbd> Screenshot</span>
</div>
<div class="credit">Scopie — part of the free nrmluvdev maker suite · made by <b>nrmluvdev</b></div>`;

const SUPPORTED_HTML = `
<table class="tbl">
<tr><th>Family</th><th>Models</th><th>Connection</th><th>Notes</th></tr>
<tr><td>Hantek 6022 family</td><td>Hantek DSO-6022BE, 6022BL, 6021; Voltcraft DSO-2020; Sainsmart/BUUDAI DDS120; Instrustar ISDS-205B; YiXingDianZi MDSO</td><td>USB</td><td>Streams raw samples, so it can capture thousands of waveforms per second. Open firmware is built in and loaded automatically. For the ISDS-205B and MDSO, import their firmware once from an OpenHantek6022 download.</td></tr>
<tr><td>Bench scopes (SCPI)</td><td>Rigol DS1000Z (DS1054Z…), DS2000, MSO5000, DHO800/900/1000; Keysight/Agilent InfiniiVision (DSOX1000–4000); Tektronix TBS1000/TBS2000/TDS1000/TDS2000; Siglent SDS1000X-E/SDS2000X-E/SDS1000X</td><td>USB or LAN</td><td>Updates about 3–30 times a second, because bench scopes send whole screens. LAN needs no driver: type the IP address in the Connect window.</td></tr>
<tr><td>OWON</td><td>HDS200/HDS300 handhelds (HDS242, HDS272, HDS2102, HDS2202, HDS307S…), SDS/XDS/TAO bench scopes</td><td>USB or LAN</td><td>Reads the scope's screen data (600–1500 points).</td></tr>
<tr><td>Hantek handhelds</td><td>2C42, 2C72, 2D42, 2D72</td><td>USB</td><td><b>Experimental.</b></td></tr>
<tr><td>PicoScope</td><td>2000 (2204A/2205A), 2000A/B, 3000A/B/D, 4000A, 5000A/B/D</td><td>USB</td><td>Needs Pico's free PicoSDK, which we may not bundle.</td></tr>
<tr><td>Old Hantek DSO</td><td>DSO-2090, 2150, 2250, 5200, 5200A</td><td>USB</td><td><b>Experimental.</b> Needs Hantek's firmware, extracted from their driver with <i>Import firmware…</i>.</td></tr>
<tr><td>Demo mode</td><td>2-channel USB scope, 4-channel bench scope</td><td>—</td><td>Simulated signals, so every feature can be learned without hardware.</td></tr>
</table>
<h3>Drivers on Windows</h3>
<ul>
<li><b>Hantek 6022BE/BL:</b> Connect window ▸ <i>Install USB driver…</i> ▸ one-step signed driver.</li>
<li><b>Other USB scopes:</b> the same button opens Zadig (included) to give the scope the WinUSB driver. Hantek 6022 and DSO-2xxx scopes get a new USB ID once their firmware is loaded: after connecting once, run Zadig again for the new entry.</li>
<li><b>Bench scopes over USB</b> work straight away if NI-VISA or Keysight IO Libraries are installed (often installed with the scope maker's software). Otherwise give them WinUSB too, or use the network.</li>
</ul>
<p><b>Not supported: the FNIRSI DSO-152 / DSO-153.</b> Their USB-C port only charges the battery and updates firmware; there is no data connection to read waveforms from. Handhelds that save waveforms as CSV files can be viewed with <b>File ▸ Open CSV</b>.</p>
<p class="muted">Experimental drivers were written from published protocol documentation and other open-source programs. They have been tested against simulated instruments, not real hardware. Reports from real scopes are very welcome.</p>`;
