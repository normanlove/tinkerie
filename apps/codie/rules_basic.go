// Codie — rules_basic.go: BASIC, from the 1980s home computers (Commodore 64,
// ZX Spectrum, BBC Micro, Apple II) to QBasic, QB64 and Visual Basic.
// Part of the free nrmluvdev maker suite. Created by nrmluvdev.
package codie

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Memory addresses that BASIC programs POKE and PEEK, by machine.
var c64Addr = map[int]string{
	53280: "the border colour (0–15)", 53281: "the background colour (0–15)", 646: "the colour of new text",
	53269: "which sprites are switched on (one bit per sprite)", 53264: "the extra 'past 255' X bit for each sprite",
	53271: "which sprites are stretched tall", 53277: "which sprites are stretched wide", 53276: "which sprites are multicolour",
	53275: "which sprites go behind the background", 53278: "sprite-to-sprite collisions (reading it also clears it)",
	53279: "sprite-to-background collisions", 53265: "screen control: on/off, rows, scrolling, bitmap mode",
	53266: "the raster line the TV beam is drawing right now", 53272: "where the screen and character set are in memory",
	53270: "screen control: columns, horizontal scroll, multicolour", 56320: "joystick port 2 (bits go to 0 when pushed: 1 up, 2 down, 4 left, 8 right, 16 fire)",
	56321: "joystick port 1 (and part of the keyboard)", 197: "the key being held down right now (64 = none)", 203: "the key being held down",
	198: "how many keys are waiting in the keyboard buffer", 650: "key repeat (128 = all keys repeat)", 162: "the 'jiffy' clock, ticking 60 times a second",
	54296: "the SID chip's volume (0–15) and filter mode", 54272: "voice 1 frequency, low byte", 54273: "voice 1 frequency, high byte",
	54276: "voice 1 control: waveform and gate (start/stop the note)", 54277: "voice 1 attack and decay", 54278: "voice 1 sustain and release",
	54283: "voice 2 control", 54290: "voice 3 control", 54299: "voice 3's output, often read as a random number", 1: "the memory layout (which ROMs are switched in)",
	2040: "sprite 0's picture pointer (picture = value × 64)", 808: "the RUN/STOP key handler (poking it disables the key)",
}

var zxAddr = map[int]string{
	23560: "LAST K: the last key pressed", 23658: "FLAGS2: caps lock", 23692: "the scroll counter (poke 255 to stop 'scroll?')",
	23672: "FRAMES: a clock counting 50 times a second", 23606: "CHARS: where the character set is", 23675: "UDG: where user-defined graphics are",
	23609: "the keyboard click length", 23624: "BORDCR: border colour",
}

var appleAddr = map[int]string{
	-16384: "the keyboard: above 127 when a key is waiting", 49152: "the keyboard: above 127 when a key is waiting",
	-16368: "clears the keyboard so the next key can be read", 49168: "clears the keyboard so the next key can be read",
	-16336: "clicks the speaker (do it fast in a loop to make a tone)", 49200: "clicks the speaker",
	-16287: "paddle/joystick button 0 (above 127 = pressed)", 49249: "button 0", -16286: "button 1", -16304: "switches to graphics",
	-16303: "switches to text", -16302: "full-screen graphics", -16301: "graphics with 4 lines of text",
}

var reBasicNum = regexp.MustCompile(`^\s*(\d+)\s*(.*)$`)
var rePeekPoke = regexp.MustCompile(`(?i)\b(PEEK|POKE)\s*\(?\s*(-?\d+)`)

func addrMeaning(x *ctx, a int) string {
	switch {
	case strings.Contains(x.dialect, "Commodore"):
		if d, ok := c64Addr[a]; ok {
			return d
		}
		switch {
		case a >= 1024 && a <= 2023:
			r, c := (a-1024)/40, (a-1024)%40
			return fmt.Sprintf("screen memory: the character at row %d, column %d", r, c)
		case a >= 55296 && a <= 56295:
			r, c := (a-55296)/40, (a-55296)%40
			return fmt.Sprintf("colour memory: the colour of row %d, column %d", r, c)
		case a >= 53248 && a <= 53263:
			n := (a - 53248) / 2
			if (a-53248)%2 == 0 {
				return fmt.Sprintf("sprite %d's X position", n)
			}
			return fmt.Sprintf("sprite %d's Y position", n)
		case a >= 53287 && a <= 53294:
			return fmt.Sprintf("sprite %d's colour", a-53287)
		case a >= 2040 && a <= 2047:
			return fmt.Sprintf("sprite %d's picture pointer", a-2040)
		case a >= 54272 && a <= 54300:
			return "a SID sound chip register"
		case a >= 53248 && a <= 53294:
			return "a VIC-II graphics chip register"
		}
	case strings.Contains(x.dialect, "Spectrum"):
		if d, ok := zxAddr[a]; ok {
			return d
		}
		switch {
		case a >= 16384 && a <= 22527:
			return "screen memory (the pixels)"
		case a >= 22528 && a <= 23295:
			return "attribute memory (the colours of each 8×8 square)"
		}
	case strings.Contains(x.dialect, "Apple"):
		if d, ok := appleAddr[a]; ok {
			return d
		}
	}
	return ""
}

var reAddrSum = regexp.MustCompile(`^\s*([A-Za-z]\w*|-?\d+)\s*(?:([+-])\s*(\d+))?\s*$`)

// evalAddr works out addresses such as V+21 when V was set to a number.
func evalAddr(x *ctx, e string) (int, bool) {
	m := reAddrSum.FindStringSubmatch(e)
	if m == nil {
		return 0, false
	}
	base, err := strconv.Atoi(m[1])
	if err != nil {
		v, ok := x.basicVals[strings.ToUpper(m[1])]
		if !ok {
			return 0, false
		}
		base = v
	}
	if m[2] != "" {
		n, _ := strconv.Atoi(m[3])
		if m[2] == "-" {
			n = -n
		}
		base += n
	}
	return base, true
}

func basicMem(x *ctx, t string) {
	if x.first {
		return
	}
	for _, m := range rePeekPoke.FindAllStringSubmatch(t, 3) {
		a, err := strconv.Atoi(m[2])
		if err != nil {
			continue
		}
		if d := addrMeaning(x, a); d != "" {
			x.notes = append(x.notes, fmt.Sprintf("Address %d is %s.", a, d))
			x.tag("hardware")
		}
	}
}

func basicJump(x *ctx, target, kind string) (string, string) {
	n, err := strconv.Atoi(target)
	if kind == "GOSUB" {
		x.tag("subroutine")
		return "Runs the subroutine starting at line " + code(target) + ", then comes back here when it reaches RETURN.", "GOSUB…RETURN is classic BASIC's way of reusing a set of steps, before named functions existed."
	}
	x.tag("goto")
	if err == nil && x.basicLine > 0 && n <= x.basicLine {
		x.tag("loop")
		if n == x.basicLine {
			return "Jumps back to the start of this same line, so it keeps repeating.", "A one-line loop: it repeats until something on the line jumps away, often used to wait for a key."
		}
		return "Jumps back to line " + code(target) + ", so the lines from there to here repeat.", "This is how line-numbered BASIC makes a loop, including a game's main loop: do a frame, then GOTO the top."
	}
	return "Jumps to line " + code(target) + " and carries on from there.", "GOTO moves the program to another line. Too many make 'spaghetti code', which is why later BASICs added named SUBs and loops."
}

func init() {
	b := "@basic"
	R(b, `^(?:PRINT|\?)\s+AT\s+(.+?)\s*,\s*(.+?)\s*;\s*(.*)$`, "print", "Prints $3 at row $1, column $2.", "ZX Spectrum PRINT AT places text anywhere on the 22×32 screen: perfect for games.", "output")
	R(b, `^(?:PRINT|\?)\s*(.*)$`, "print", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("output")
		v := strings.TrimSpace(m[1])
		if v == "" {
			return "Prints an empty line.", "Moves the cursor down a line."
		}
		why := "PRINT shows text or numbers on the screen."
		if strings.HasSuffix(v, ";") {
			why += " The ; at the end keeps the cursor on the same line for the next PRINT."
		} else if strings.Contains(v, ";") {
			why += " Items joined with ; are printed right next to each other."
		}
		if strings.Contains(v, ",") && !strings.Contains(v, "(") {
			why += " A comma jumps to the next column (tab zone)."
		}
		uv := strings.ToUpper(v)
		if strings.Contains(uv, "CHR$(147)") {
			return "Clears the screen (character 147 is the C64's clear-screen code).", "On the Commodore 64, PRINT CHR$(147) is the usual way to clear the screen; you can also type the clear key inside quotes."
		}
		if strings.HasPrefix(uv, "TAB(") || strings.HasPrefix(uv, "SPC(") {
			why += " TAB/SPC move the cursor across first."
		}
		return "Prints " + code(unhide(v, x.strs)) + " on the screen.", why
	})
	R(b, `^LINE\s+INPUT\s+(?:({S})\s*[;,]\s*)?(.+)$`, "input", "Asks the user to type a whole line (commas and all) and stores it in $2.", "", "input")
	R(b, `^INPUT\s+(?:({S})\s*[;,]\s*)?(.+)$`, "input", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("input")
		w := "Waits for the user to type something and press Enter, then stores it in " + code(m[2]) + "."
		if m[1] != "" {
			w = "Shows the question " + expand("$1", m, x) + ", waits for the user to type an answer, and stores it in " + code(m[2]) + "."
		}
		why := "INPUT stops the program until the user answers."
		if strings.Contains(m[2], "$") {
			why += " The $ means the variable holds text."
		} else {
			why += " Without a $ the variable holds a number, so typing letters gives an error (or ?REDO FROM START)."
		}
		return w, why
	}).setsVar(2)
	R(b, `^GET\s+(\w+\$?)$`, "input", "Reads a key from the keyboard without waiting; $1 is empty if no key is pressed.", "Unlike INPUT, GET doesn't stop the program, so games can check keys every frame.", "input").setsVar(1)
	R(b, `^IF\s+(.+?)\s+THEN\s*(.*)$`, "if", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("if")
		rest := strings.TrimSpace(m[2])
		cond := checks(x, m[1])
		if rest == "" {
			x.pending = "if-block"
			return "If-check: " + cond + ". If so, run the lines down to END IF.", "A block IF (QBasic / Visual Basic style)."
		}
		rest2, els := rest, ""
		if i := strings.Index(strings.ToUpper(rest), " ELSE "); i > 0 {
			rest2, els = strings.TrimSpace(rest[:i]), strings.TrimSpace(rest[i+6:])
		}
		var w string
		if _, err := strconv.Atoi(rest2); err == nil {
			w = "If-check: " + cond + ". If so, jump to line " + code(rest2) + "."
			x.tag("goto")
		} else {
			w = "If-check: " + cond + ". If so, do " + code(unhide(rest2, x.strs)) + "."
		}
		if els != "" {
			w += " Otherwise do " + code(unhide(els, x.strs)) + "."
		}
		why := "In classic BASIC everything after THEN on the same line only runs when the check is true; otherwise the program goes straight to the next line."
		if strings.Contains(m[1], "=") && !strings.ContainsAny(m[1], "<>") {
			why += " Inside IF, = means 'equals' (comparing), not 'set'."
		}
		basicMem(x, m[1])
		return w, why
	})
	R(b, `^IF\s+(.+?)\s+GOTO\s+(\d+)$`, "if", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("if", "goto")
		return "If-check: " + checks(x, m[1]) + ". If so, jump to line " + code(m[2]) + ".", "IF … GOTO is a shortcut for IF … THEN GOTO."
	})
	R(b, `^ELSE\s*IF\s+(.+?)\s+THEN$`, "if", "", "").with(func(m []string, x *ctx) (string, string) {
		return "Otherwise, if-check: " + checks(x, m[1]) + ".", ""
	})
	R(b, `^ELSE$`, "if", "Otherwise (when the IF above was false), run the lines below.", "", "if")
	R(b, `^END\s*IF$`, "close", "", "").close()
	R(b, `^GOTO\s+(\w+)$`, "jump", "", "").with(func(m []string, x *ctx) (string, string) { return basicJump(x, m[1], "GOTO") })
	R(b, `^GOSUB\s+(\w+)$`, "call", "", "").with(func(m []string, x *ctx) (string, string) { return basicJump(x, m[1], "GOSUB") })
	R(b, `^RETURN$`, "return", "Goes back to the line just after the GOSUB that called this subroutine.", "Every GOSUB needs a RETURN; without one the program never goes back.", "subroutine")
	R(b, `^ON\s+(.+?)\s+(GOTO|GOSUB)\s+(.+)$`, "jump", "Jumps to one of the lines $3, picked by the value of $1 (1 = first, 2 = second…).", "A quick menu: ON choice GOTO 100,200,300.", "goto")
	R(b, `^FOR\s+({I})\s*=\s*(.+?)\s+TO\s+(.+?)(?:\s+STEP\s+(.+))?$`, "loop", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("loop")
		w := "Counts " + code(m[1]) + " from " + code(m[2]) + " to " + code(m[3])
		if m[4] != "" {
			w += " in steps of " + code(m[4])
		}
		return w + ", running the lines down to NEXT each time.", "A FOR…NEXT loop repeats a set number of times. Old BASIC games also used empty FOR loops just to wait."
	}).open("loop", 1).setsVar(1)
	R(b, `^NEXT\s*({I}(?:\s*,\s*{I})*)?$`, "close", "", "").with(func(m []string, x *ctx) (string, string) {
		if m[1] != "" {
			return "Adds the step to " + code(m[1]) + " and goes back to the FOR, unless it has passed the end value.", "When the count is finished, the program carries on below NEXT."
		}
		return "Goes back to the FOR for the next count, unless it has finished.", ""
	}).close()
	R(b, `^WHILE\s+(.+)$`, "loop", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("loop")
		return "A while-loop: " + checks(x, m[1]) + ", and repeats the lines down to WEND while it stays true.", ""
	}).open("loop", 0)
	R(b, `^(?:WEND|END\s+WHILE)$`, "close", "Goes back to the WHILE to check again.", "", "loop").close()
	R(b, `^DO(?:\s+(WHILE|UNTIL)\s+(.+))?$`, "loop", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("loop")
		if m[1] == "" {
			return "Starts a DO…LOOP: the lines down to LOOP repeat.", "Without a condition here it repeats until a condition on the LOOP line, or EXIT DO."
		}
		return "Starts a loop that repeats " + strings.ToLower(m[1]) + " " + code(unhide(m[2], x.strs)) + ".", ""
	}).open("loop", 0)
	R(b, `^LOOP(?:\s+(WHILE|UNTIL)\s+(.+))?$`, "close", "", "").with(func(m []string, x *ctx) (string, string) {
		if m[1] == "" {
			return "Goes back to DO.", ""
		}
		if strings.Contains(strings.ToUpper(m[2]), "INKEY$") {
			x.tag("input")
		}
		return "Goes back to DO " + strings.ToLower(m[1]) + " " + code(unhide(m[2], x.strs)) + ".", "Checking at the bottom means the loop always runs at least once."
	}).close()
	R(b, `^REPEAT$`, "loop", "Starts a REPEAT…UNTIL loop (BBC BASIC).", "The block runs at least once.", "loop").open("loop", 0)
	R(b, `^UNTIL\s+(.+)$`, "close", "…and repeats until $1.", "", "loop").close()
	R(b, `^EXIT\s+(FOR|DO|SUB|FUNCTION|WHILE)$`, "jump", "Leaves the $1 straight away.", "")
	R(b, `^(?:(?:PUBLIC|PRIVATE|SHARED|STATIC|FRIEND|OVERRIDES)\s+)*SUB\s+({I})\s*(?:\((.*)\))?(?:\s+(?:STATIC|Handles\s+.+))?$`, "def", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("function")
		name := m[1]
		w := "Defines the subroutine " + code(name) + "."
		if strings.EqualFold(name, "Main") {
			w += " Visual Basic starts the program here."
		}
		if strings.Contains(name, "_") && (strings.HasSuffix(strings.ToLower(name), "_click") || strings.HasSuffix(strings.ToLower(name), "_load") || strings.HasSuffix(strings.ToLower(name), "_tick")) {
			w += " Its name says it runs on an event: " + strings.ReplaceAll(name, "_", " ") + "."
			x.tag("callback")
		}
		return w, "A SUB is a named block of steps you run with its name (or CALL). " + takes(x.lang, m[2])
	}).open("function", 1)
	R(b, `^DECLARE\s+(SUB|FUNCTION)\s+(\w+)(.*)$`, "def", "Tells QBasic the $1 $2 exists further down.", "QBasic needs this so it can be called before it's written.")
	R(b, `^(?:(?:PUBLIC|PRIVATE|SHARED|STATIC)\s+)*FUNCTION\s+({I}[$%!#]?)\s*(?:\((.*)\))?(?:\s+As\s+(\w+))?$`, "def", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("function")
		return "Defines the function " + code(m[1]) + ".", "A FUNCTION works out a value and gives it back. " + takes(x.lang, m[2])
	}).open("function", 1)
	R(b, `^(?:PUBLIC\s+|PRIVATE\s+|FRIEND\s+)?(MODULE|CLASS|STRUCTURE)\s+(\w+)$`, "class", "Starts the $1 $2.", "In Visual Basic .NET, code lives inside modules and classes; a Sub Main inside starts the program.", "class").open("class", 2)
	R(b, `^IMPORTS\s+([\w.]+)$`, "import", "Lets this file use $1 by short names.", "", "import")
	R(b, `^FOR\s+EACH\s+(\w+)(?:\s+AS\s+\w+)?\s+IN\s+(.+)$`, "loop", "Goes through each item in $2, calling it $1.", "", "loop").open("loop", 1).setsVar(1)
	R(b, `^FOR\s+({I})\s+AS\s+\w+\s*=\s*(.+?)\s+TO\s+(.+?)(?:\s+STEP\s+(.+))?$`, "loop", "Counts $1 from $2 to $3, running the lines down to Next each time.", "As Integer makes the counter a whole number.", "loop").open("loop", 1).setsVar(1)
	R(b, `^END\s+(SUB|FUNCTION|CLASS|MODULE|PROPERTY|STRUCTURE|SELECT)$`, "close", "", "").close()
	R(b, `^DEF\s*PROC_?(\w+)(?:\((.*)\))?$`, "def", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("function")
		return "Defines the procedure PROC" + m[1] + " (BBC BASIC).", "Procedures are named subroutines; call them with PROC" + m[1] + ". " + takes(x.lang, m[2])
	}).open("function", 1)
	R(b, `^ENDPROC$`, "close", "", "").close()
	R(b, `^PROC_?(\w+)(?:\((.*)\))?$`, "call", "Runs the procedure PROC$1.", "After it reaches ENDPROC, the program comes back here.", "function")
	R(b, `^DEF\s*FN_?(\w+)\s*\((.*?)\)\s*=\s*(.+)$`, "def", "Defines a one-line function FN$1 that works out $3.", "Use it as FN$1(…) inside a sum.", "function")
	R(b, `^SELECT\s+CASE\s+(.+)$`, "switch", "Chooses what to do based on $1.", "Each CASE below handles one value.", "switch").open("switch", 0)
	R(b, `^CASE\s+ELSE$`, "case", "If no other CASE matched, run the lines below.", "")
	R(b, `^CASE\s+(.+)$`, "case", "If the value is $1, run the lines below.", "")
	R(b, `^END$`, "jump", "Ends the program.", "Without it the program would run on into the subroutines below.")
	R(b, `^STOP$`, "jump", "Stops the program (it can be continued with CONT).", "Handy while testing.")
	R(b, `^DATA\s+(.+)$`, "data", "Stores the values $1 inside the program for READ to pick up.", "DATA lines are never run; they hold level maps, sprite shapes, music notes and so on.", "data")
	R(b, `^READ\s+(.+)$`, "data", "Takes the next value(s) from the DATA lines and puts them in $1.", "Each READ moves on through the DATA, in order.", "data").setsVar(1)
	R(b, `^RESTORE\s*(\d*)$`, "data", "Goes back to the start of the DATA (or to line $1), so READ starts again.", "", "data")
	R(b, `^POKE\s+(.+?)\s*,\s*(.+)$`, "call", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("hardware")
		w := "Writes the value " + code(m[2]) + " straight into memory address " + code(m[1]) + "."
		if a, ok := evalAddr(x, m[1]); ok {
			if d := addrMeaning(x, a); d != "" {
				w = "Sets " + d + " to " + code(m[2]) + " (POKE " + strings.TrimSpace(m[1]) + ", which is address " + strconv.Itoa(a) + ")."
			}
		}
		return w, "On old machines the screen, colours, sound and sprites are all controlled by memory addresses, so POKE is how BASIC talks to the hardware."
	})
	R(b, `^SYS\s+(.+)$`, "call", "Runs machine code starting at address $1.", "Used for speed: games often loaded fast assembly routines and started them with SYS.", "hardware")
	R(b, `^CALL\s+(-?\d+)$`, "call", "Runs machine code at address $1 (Apple II).", "", "hardware")
	R(b, `^RANDOMIZE\s*(.*)$`, "call", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("random")
		if strings.Contains(strings.ToUpper(m[1]), "TIMER") {
			return "Starts the random number generator from the clock.", "Without it, RND gives the same 'random' numbers every run."
		}
		return "Starts (seeds) the random number generator.", "So RND gives different numbers each run."
	})
	R(b, `^(?:LET\s+)?({I}[$%!#]?(?:\(.*?\))?)\s*=\s*(.+)$`, "assign", "", "").with(func(m []string, x *ctx) (string, string) {
		if n, err := strconv.Atoi(strings.TrimSpace(m[2])); err == nil {
			x.basicVals[strings.ToUpper(m[1])] = n
		}
		p := assignPart(x, "", m[1], "", m[2])
		ue := strings.ToUpper(m[2])
		why := p.Why
		switch {
		case strings.Contains(ue, "INKEY$"):
			x.tag("input")
			why = "INKEY$ checks the keyboard without waiting: it's empty when no key is down, which is how BASIC games read controls."
		case strings.Contains(ue, "RND"):
			x.tag("random")
		case strings.Contains(ue, "PEEK"):
			x.tag("hardware")
		}
		if strings.HasSuffix(baseName(m[1]), "$") {
			why += " The $ means it holds text."
		}
		if strings.HasSuffix(baseName(m[1]), "%") {
			why += " The % means a whole number."
		}
		return p.What, why
	}).setsVar(1)
	R(b, `^DIM\s+({I})\s+AS\s+(NEW\s+)?([\w.()]+)(?:\s*=\s*(.+))?$`, "var", "", "").with(func(m []string, x *ctx) (string, string) {
		w := "Creates the variable " + code(m[1]) + " to hold " + typeName(strings.ToLower(m[3]))
		if m[2] != "" {
			w = "Creates " + code(m[1]) + " as a new " + code(m[3]) + " object"
		}
		if m[4] != "" {
			w += ", set to " + describeExpr(x, m[4])
		}
		x.tag("variable")
		return w + ".", varWhy(x, m[1], "")
	}).setsVar(1)
	R(b, `^DIM\s+(.+)$`, "var", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("array")
		return "Makes room for the list(s) (arrays) " + code(m[1]) + ".", "DIM A(10) gives A(0) to A(10): eleven slots. Arrays hold things like the positions of every enemy."
	})
	R(b, `^SCREEN\s+(\d+)(.*)$`, "call", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("rendering")
		d := map[string]string{"0": "text mode (80×25)", "1": "320×200 graphics with 4 colours", "2": "640×200 graphics in 2 colours", "7": "320×200 with 16 colours and several pages", "9": "640×350 with 16 colours", "12": "640×480 with 16 colours", "13": "320×200 with 256 colours, the favourite mode for QBasic games"}[m[1]]
		if d == "" {
			return "Switches to screen mode " + code(m[1]) + ".", ""
		}
		w := "Switches the screen to mode " + m[1] + ": " + d + "."
		if strings.Count(m[2], ",") >= 2 {
			w += " The extra numbers choose which page is shown and which is drawn on, for flicker-free animation."
		}
		return w, "The screen mode decides the resolution and colours available."
	}).first()
	R(b, `^_LIMIT\s+(\d+)$`, "call", "Slows the loop down to at most $1 rounds a second (QB64).", "Keeps a game running at the same speed on every computer, like a frame-rate cap.", "game-loop", "timing")
	R(b, `^MODE\s+(\d+)$`, "call", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("rendering")
		d := map[string]string{"0": "640×256 in 2 colours, 80 columns", "1": "320×256 in 4 colours", "2": "160×256 in 8 colours (16 with flashing)", "4": "320×256 in 2 colours", "5": "160×256 in 4 colours", "7": "Teletext mode: colourful text and blocky graphics"}[m[1]]
		return "Switches the BBC Micro to screen MODE " + m[1] + ": " + d + ".", "Lower modes have more pixels; higher modes save memory for the program."
	})
	R(b, `^\*(\w+)\s*(.*)$`, "call", "Sends the operating-system command *$1 $2.", "Star commands go straight to the BBC Micro's MOS; *FX changes system settings.", "hardware")
	R(b, `^({I}\$?)\s*(.*)$`, "call", "", "").with(func(m []string, x *ctx) (string, string) {
		e := findKey(x, strings.ToUpper(m[1]))
		if e == nil {
			if ln, ok := x.funcs[m[1]]; ok && strings.TrimSpace(m[2]) == "" || ok {
				x.tag("function")
				return fmt.Sprintf("Runs the subroutine %s, written on line %d.", code(m[1]), ln), "Calling a SUB runs its steps, then comes back here."
			}
			return "", ""
		}
		x.tag(e.Tags...)
		w := e.What
		if a := strings.TrimSpace(m[2]); a != "" {
			w = strings.TrimSuffix(w, ".") + " (here: " + code(unhide(a, x.strs)) + ")."
		}
		return w, e.Why
	})
}
