// teach.go: teaching mode. Point at any control to see a beginner-friendly
// explanation in a card. F2 or Settings turns it on and off. This matches the
// teaching mode in the other Tinkerie apps. Created by nrmluvdev.
package meterie

import "strings"

// teachTips explains controls, by widget id (numbers at the end are ignored,
// so "dev#rm3" finds "dev#rm"), by teach-zone id, or by the control's label.
var teachTips = map[string][2]string{
	// ------------------------------------------------------------- pages
	"Live":                 {"Live", "The big readout from your meter, its statistics and a graph of every reading over time."},
	"Meters":               {"Meters", "Connect a meter over USB, Bluetooth or the camera, or try the demo meter. You can use several meters at once."},
	"Logging":              {"Logging", "Records readings to a CSV spreadsheet file, every so often or only when something happens (for example when the voltage drops below a level)."},
	"Alarms":               {"Alarms", "Warns you with a sound and a red flash when a reading goes above or below a limit you set."},
	"Math":                 {"Math", "Works out new values from your meters: power from volts and amps, resistance, or energy used over time (Wh and mAh)."},
	"Wizards":              {"Wizards", "Step-by-step tests: battery capacity, a battery's internal resistance, sorting resistors and capacitors, LED forward voltage, and loading a solar panel or power supply."},
	"Camera":               {"Camera", "Reads a meter that has no PC connection by pointing a webcam or phone camera at its display."},
	"What am I measuring?": {"What am I measuring?", "Explains every meter mode (volts, amps, ohms, continuity, diode, capacitance, frequency, temperature), how to connect the probes and stay safe."},
	"Settings":             {"Settings", "Colours, sounds, size of everything on screen, where files are saved, and the link that lets other Tinkerie apps read your meter."},
	"rail#new":             {"New session", "Clears everything and starts a fresh session. Save first if you want to keep your readings."},
	"rail#open":            {"Open session (Ctrl+O)", "Opens a saved session: readings, markers, alarms and math channels."},
	"rail#save":            {"Save session (Ctrl+S)", "Saves everything (readings, markers, alarms and math channels) so you can reopen it later."},

	// --------------------------------------------------------------- live
	"readout":      {"Readout", "The latest reading, as big as a bench meter's display. The icons show the meter's mode: AC or DC, auto or manual range, HOLD, and low battery."},
	"stats":        {"Statistics", "The smallest, largest and average readings, the peak-to-peak range and how many readings there are, worked out over the window you pick at the top."},
	"stats#win":    {"Statistics window", "Which readings the statistics use: what's visible on the graph, a part you selected (Shift+drag on the graph), the last 10 seconds, or everything."},
	"st#rel":       {"Relative (REL)", "Sets the current reading as zero, so you see only the change from it. Handy for removing test-lead resistance or watching a small drift."},
	"st#copy":      {"Copy", "Copies the statistics to the clipboard, to paste into notes or a spreadsheet."},
	"st#clear":     {"Clear", "Clears the readings of this channel and starts the statistics again."},
	"strip":        {"Channels", "Every meter and math channel. Click one to show it in the big readout."},
	"graph":        {"Graph", "Every reading over time. Scroll to zoom, drag to pan, Shift+drag to select a part, right-click to add a marker, double-click to go back to live."},
	"graph#win":    {"Time window", "How much time the graph shows across its width."},
	"graph#png":    {"Export graph as PNG", "Saves a picture of the graph, for reports or sharing."},
	"graph#marker": {"Add marker (Ctrl+M)", "Puts a labelled line on the graph at this moment, for example \"switched the motor on\", so you can find it later."},
	"graph#fit":    {"Back to live", "Goes back to following the newest readings with automatic scaling."},
	"graph#auto":   {"Auto-scale", "Fits the graph's height to the readings automatically. Turn it off to keep a fixed scale."},
	"m#add":        {"Add marker", "Adds the marker with the label you typed."},

	// ------------------------------------------------------------- meters
	"mt#tabs":     {"How to connect", "Pick how your meter talks to the PC: auto-detect, a USB or serial model, Bluetooth, the camera, or typing readings in by hand."},
	"ad#scan":     {"Scan", "Looks for meters on every USB and serial port."},
	"ad#go":       {"Detect & connect", "Tries the known meter types on this port until one answers, then connects to it."},
	"mdl#port":    {"Port", "The USB or serial (COM) port the meter's cable is plugged into."},
	"mdl#baud":    {"Baud rate", "How fast the cable sends data. Meterie fills this in for each model; only change it if the meter's manual says so."},
	"mdl#tr":      {"Connection type", "Whether this meter's cable shows up as a serial (COM) port or as a USB HID device. Each model uses one or the other."},
	"mdl#connect": {"Connect", "Connects to the meter with these settings."},
	"mdl#rescan":  {"Rescan ports", "Looks again for ports, for example after plugging the cable in."},
	"mdl#toble":   {"Go to Bluetooth", "This meter connects over Bluetooth, not a cable."},
	"ble#scan":    {"Scan for Bluetooth meters", "Searches for nearby Bluetooth meters. Switch the meter's Bluetooth on first (often a long press on a button)."},
	"ble#c":       {"Connect", "Connects to this Bluetooth meter."},
	"ble#proto":   {"Protocol", "The data format this Bluetooth meter uses. Meterie usually picks it for you."},
	"dev#mdl":     {"Exact model", "Auto-detect can only tell the chip family, so choose your exact meter model here once. Meterie remembers it."},
	"dev#raw":     {"Raw bytes", "Shows the exact bytes arriving from the meter. Only needed for fixing connection problems."},
	"dev#rm":      {"Disconnect", "Disconnects this meter and removes it from the list."},
	"demo#start":  {"Demo meter", "Starts a pretend meter so you can explore Meterie without any hardware."},
	"man#add":     {"Manual channel", "Makes a channel for readings you type in yourself, for meters that can't connect at all."},
	"manual#add":  {"Add reading", "Adds the number you typed as a new reading, with the current time."},
	"manual#u":    {"Unit", "The unit of the readings you type in: V, A, Ω and so on."},

	// ------------------------------------------------------------ logging
	"log#choose":  {"Log file", "Where the CSV log file will be saved."},
	"log#iv":      {"Sample interval", "How often a reading is written to the log. Faster gives more detail but bigger files."},
	"log#start":   {"Start condition", "When logging begins: straight away, or when a reading crosses a level you set."},
	"log#startch": {"Start channel", "Which channel the start condition watches."},
	"log#stop":    {"Stop condition", "When logging ends: when you press Stop, after a time, or when a reading crosses a level."},
	"log#stopch":  {"Stop channel", "Which channel the stop condition watches."},
	"log#ch":      {"Channels to log", "Tick the channels whose readings go into the log file."},
	"log#go":      {"Start logging", "Starts recording to the log file using these settings."},
	"log#stopbtn": {"Stop logging", "Stops recording and closes the file."},
	"log#open":    {"Open log folder", "Opens the folder with your log files."},

	// ------------------------------------------------------------- alarms
	"al#add":     {"Add alarm", "Adds a high or low limit for a channel. When a reading crosses it, Meterie beeps and flashes."},
	"snd#master": {"Sound", "Turns all of Meterie's sounds on or off."},
	"snd#cont":   {"Continuity beep", "Beeps on the PC while a continuity reading is below the threshold, like the beep on the meter itself, so you can keep your eyes on the probes."},
	"snd#t1":     {"Test beep", "Plays the continuity beep so you can check your speakers."},
	"snd#t2":     {"Test alarm", "Plays the alarm sound."},

	// --------------------------------------------------------------- math
	"math#preset": {"Formula", "What to work out: power (V × I), resistance (V ÷ I), or energy and charge added up over time."},
	"math#a":      {"First input", "The channel to use as the first value in the formula, usually the voltage."},
	"math#b":      {"Second input", "The channel to use as the second value, usually the current."},
	"math#add":    {"Add math channel", "Creates the new channel. It appears in the channel strip and the graph like a real meter."},
	"mc#rs":       {"Reset to zero", "Starts counting energy or charge again from zero."},
	"mc#rm":       {"Remove", "Deletes this math channel."},

	// ------------------------------------------------------------ wizards
	"wz#back":   {"Wizards", "Back to the list of guided tests."},
	"bc#v":      {"Voltage channel", "The meter measuring the battery's voltage."},
	"bc#i":      {"Current channel", "The meter measuring the current flowing out of the battery."},
	"bc#user":   {"No ammeter", "If you only have one meter, Meterie can work out the current from the voltage and the value of your load resistor (I = V ÷ R)."},
	"bc#start":  {"Start discharge", "Starts the capacity test: Meterie adds up the charge until the battery reaches its cut-off voltage."},
	"bc#stop":   {"Stop test", "Stops the capacity test now."},
	"ir#c1":     {"Open-circuit voltage", "Measure the battery with nothing connected, then capture it."},
	"ir#c2":     {"Loaded voltage", "Connect a known load, then capture. The voltage drop shows the battery's internal resistance."},
	"ir#again":  {"Measure again", "Repeats the internal resistance test."},
	"so#type":   {"Part type", "Sort resistors or capacitors."},
	"so#ser":    {"E-series", "The standard value series parts are sold in. E12 parts have about 10% tolerance, E24 about 5%, E96 about 1%."},
	"so#export": {"Export CSV", "Saves your sorted parts list as a spreadsheet file."},
	"so#clear":  {"Clear list", "Empties the list of measured parts."},
	"led#cap":   {"Save to list", "Saves this LED's forward voltage and current to the list."},
	"lt#mode":   {"What you're testing", "A solar panel (finds its maximum power point) or a power supply (checks how its voltage holds up under load)."},
	"lt#v":      {"Voltage channel", "The meter measuring voltage across the source."},
	"lt#i":      {"Current channel", "The meter measuring current from the source."},
	"lt#user":   {"No ammeter", "Work the current out from the voltage and your load resistor instead of measuring it."},
	"lt#auto":   {"Auto-capture", "Saves a point by itself whenever the reading stops changing."},
	"lt#cap":    {"Capture point", "Saves the current voltage and current as one point on the curve."},
	"lt#clr":    {"Clear", "Removes all captured points."},

	// ------------------------------------------------------------- camera
	"cam#src":     {"Camera source", "A webcam, a phone used as an IP camera, or a photo of the meter's display."},
	"cam#dev":     {"Camera", "Which camera to use."},
	"cam#rescan":  {"Find cameras", "Looks again for cameras."},
	"cam#start":   {"Start camera", "Switches the camera on."},
	"cam#stop":    {"Stop camera", "Switches the camera off."},
	"cam#type":    {"Display type", "LCD displays have dark digits on a light background; LED or backlit displays have bright digits."},
	"cam#digits":  {"Number of digits", "How many digits the meter's display has."},
	"cam#calib":   {"Calibrate", "Type exactly what the meter's display shows right now, then press Auto-fit."},
	"cam#calbtn":  {"Auto-fit", "Adjusts the reader (digit slant, brightness threshold and segment thickness) until it reads exactly what you typed. Do this once whenever the lighting or camera position changes."},
	"cam#autothr": {"Automatic threshold", "Works out by itself which parts of the picture are lit segments."},
	"cam#dp":      {"Decimal points", "Looks for the decimal point between digits."},
	"cam#unit":    {"Unit", "The unit the meter is showing, because the camera can't always read it."},
	"cam#feed":    {"Send to timeline", "Adds the camera's readings to the graph and logs like a connected meter."},
	"cam#go":      {"Camera mode", "Reads a meter's display with a camera."},

	// ----------------------------------------------------------- settings
	"set#scale":   {"Size", "Makes everything on screen bigger or smaller."},
	"set#api":     {"Suite link", "Lets other Tinkerie apps on this PC (like Wirie's Calcie) read your live meter reading."},
	"set#copyurl": {"Copy URL", "Copies the web address other programs use to read your meter."},
	"set#welcome": {"Welcome tips", "Shows the welcome tips when Meterie starts."},
	"set#teach":   {"Teaching mode", "Shows a card like this one explaining whatever you point at. F2 turns it on and off."},
	"set#open":    {"Open", "Opens the folder."},
	"learn#mine":  {"Explain my meter's mode", "Explains the mode your connected meter is in right now."},
	"w#connect":   {"Connect a meter", "Goes to the Meters page to connect your meter."},
	"hdr#connect": {"Connect meter", "Goes to the Meters page to connect a meter by USB, Bluetooth or camera."},
	"hdr#export":  {"Export", "Saves your readings as a CSV spreadsheet, a suite timeline that other Tinkerie apps can line up with, a picture of the graph, or a whole session. It also brings in captures from Scopie and other apps."},
	"exp#codie":   {"Explain in Codie", "Saves these readings and opens them in Codie, which explains them in plain words: a battery charging or running down, a gadget that sleeps and wakes up, power and energy used, temperatures warming up or settling, and anything worth checking."},
	"hdr#marker":  {"Marker", "Puts a labelled line on the graph at this moment, so you can find it later."},
	"learn#":      {"Meter modes", "Pick a mode to learn what it measures, which jacks the probes go in, and how to measure safely."},
	"wz#":         {"Guided test", "Opens this step-by-step test. Each one tells you what to connect and does the maths for you."},
	"snd#thr":     {"Continuity threshold", "Below this resistance the PC beeps for continuity. Most meters beep below about 30 Ω."},
	"math#name":   {"Channel name", "A name for the new math channel, shown in the channel strip and on the graph."},
	"set#max":     {"Readings kept", "How many readings Meterie keeps in memory per channel. More lets you scroll back further but uses more memory."},
	"w#demo":      {"Demo meter", "Starts a pretend meter so you can try everything without hardware."},
}

// TeachZone marks an area that isn't a control (a readout, a chart) so
// teaching mode can explain it when the mouse is over it.
func (u *UI) TeachZone(id string, r Rect) {
	if u.Teach && u.Hover(r) {
		u.teachZone = id
	}
}

// noteLabel remembers the label of the hovered control.
func (u *UI) noteLabel(id, label string) {
	if u.hot == id && label != "" {
		u.teachLabel = label
	}
}

// teachTip finds the explanation for what's under the mouse.
func (u *UI) teachTip() (string, string) {
	keys := []string{}
	if u.hot != "" {
		base := strings.TrimRight(u.hot, "0123456789")                  // "dev#rm3" -> "dev#rm"
		keys = append(keys, u.hot, base, strings.TrimSuffix(base, "#")) // "stats#win#1" -> "stats#win"
	}
	if u.teachLabel != "" {
		keys = append(keys, u.teachLabel)
	}
	if u.hot == "" && u.teachZone != "" {
		keys = append(keys, u.teachZone)
	}
	for _, k := range keys {
		if t, ok := teachTips[k]; ok {
			return t[0], t[1]
		}
	}
	if u.tooltip != "" { // a control with a tooltip explains itself
		first, rest, _ := strings.Cut(u.tooltip, "\n")
		if rest == "" {
			return orStr(u.teachLabel, "Tip"), first
		}
		return first, rest
	}
	return "", ""
}

func orStr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// drawTeach draws the teaching card. It returns true if it drew one.
func (u *UI) drawTeach() bool {
	title, text := "", ""
	if u.teachIntro > 0 {
		u.teachIntro -= u.dt
		u.Animating = true
		title, text = "Teaching mode is on", "Point at anything to learn what it does. Press F2 to turn these explanations off."
		if !u.Teach {
			title, text = "Teaching mode is off", "Press F2 to turn the explanations back on."
		}
	}
	if u.Teach && (u.hot != "" || u.teachZone != "" || u.tooltip != "") {
		if t, x := u.teachTip(); t != "" {
			title, text = t, x
		}
	}
	if title == "" {
		return false
	}
	pad := u.Px(16)
	w := min(u.Px(640), u.C.W-u.Px(32))
	fb, f := u.FB(13.5), u.F(13)
	lines := u.WrapLines(text, w-2*pad, f)
	lh := u.T.LineHeight(f)
	h := pad + u.T.LineHeight(fb) + u.Px(4) + len(lines)*lh + pad - u.Px(4)
	r := Rect{(u.C.W - w) / 2, u.C.H - h - u.Px(44), w, h}
	if u.In.MY > u.C.H*62/100 {
		r.Y = u.Px(64)
	}
	u.C.Shadow(r, u.Pf(14), u.Pf(16), u.Pf(5), RGBA(0, 0, 0, 130))
	u.C.FillRoundRect(r, u.Pf(14), th.Card)
	u.C.StrokeRoundRect(r, u.Pf(14), 1, th.BorderHi)
	y := r.Y + pad - u.Px(4)
	tag := "Teaching mode · F2"
	ft := u.F(11)
	u.T.Draw(tag, r.Right()-pad-u.T.Measure(tag, ft), y+u.Px(2), ft, th.Sub)
	u.T.Draw(title, r.X+pad, y, fb, th.Text)
	y += u.T.LineHeight(fb) + u.Px(4)
	for _, l := range lines {
		u.T.Draw(l, r.X+pad, y, f, th.Sub)
		y += lh
	}
	return true
}

// ToggleTeach switches teaching mode and shows a short notice.
func (u *UI) ToggleTeach() {
	u.Teach = !u.Teach
	u.teachIntro = 4
}
