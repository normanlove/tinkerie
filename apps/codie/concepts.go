// Codie — concepts.go: the Concepts tab (a short explanation of each idea the
// code uses, with the lines that use it) and the Warnings tab.
// Part of the free nrmluvdev maker suite. Created by nrmluvdev.
package codie

import (
	"regexp"
	"sort"
	"strings"
)

var glossary = map[string][2]string{
	"variable":      {"Variables", "A variable is a named box holding a value, like score = 0. The program can read it and change it later."},
	"operators":     {"Changing values", "+=, -=, *= and friends change a variable using its own value: score += 10 adds 10 to the score."},
	"function":      {"Functions", "A function is a named set of steps. Define it once, then run (call) it whenever needed, giving it inputs (parameters) and maybe getting an answer back (return)."},
	"function-call": {"Calling functions", "Writing a function's name with brackets runs it. Values in the brackets are its inputs (arguments)."},
	"callback":      {"Callbacks and events", "Some functions you never call yourself: the engine, framework or system calls them at the right moment, such as Update every frame or a click handler when a button is pressed."},
	"class":         {"Classes and objects", "A class is a blueprint describing data and behaviour. Objects are made from it, each with its own copy of the data, like many enemies built from one Enemy class."},
	"inheritance":   {"Inheritance", "A class can build on another: it gets everything the parent has and adds or changes parts. Player : MonoBehaviour gets all of Unity's MonoBehaviour features."},
	"if":            {"If-checks (conditions)", "An if runs code only when a condition is true, with else for when it isn't. This is how programs make decisions."},
	"loop":          {"Loops", "A loop repeats code: a set number of times (for), for each item in a list (for-each), or while something stays true (while)."},
	"switch":        {"Switch / match", "Chooses one branch out of many based on a value. Tidier than a long chain of if / else if."},
	"errors":        {"Errors", "Things go wrong: missing files, bad input, lost connections. Languages handle it with try/catch blocks or, in Go, by returning an error value to check."},
	"import":        {"Imports and libraries", "Imports bring in code written elsewhere, from your own files or from libraries, so you don't have to write everything yourself."},
	"output":        {"Output", "Printing or showing text: the simplest way for a program to tell you what it's doing."},
	"input":         {"Input", "Getting information from the user: typed text, key presses, mouse clicks or a game controller."},
	"list":          {"Lists and collections", "Lists (arrays) hold many values in order; dictionaries (maps) look values up by a key. Games keep their enemies and bullets in lists."},
	"array":         {"Arrays", "An array is a numbered row of boxes under one name: A(1), A(2), A(3)…"},
	"table":         {"Lua tables", "Tables are Lua's one data structure: lists, records and objects are all tables."},
	"text":          {"Working with text", "Text values (strings) can be joined, cut up, searched and turned into numbers and back."},
	"maths":         {"Maths", "Square roots, angles and rounding: games use maths for distances, directions and smooth movement."},
	"random":        {"Random numbers", "Random numbers make things unpredictable: dice, spawn points, loot."},
	"timers":        {"Time and waiting", "Measuring time, waiting, and doing things later or regularly."},
	"files":         {"Files", "Reading and writing files lets a program keep information after it closes."},
	"data":          {"Data formats", "JSON and DATA lines store structured information in a form programs can read back."},
	"network":       {"Networking", "Talking to other computers: web pages, APIs, multiplayer servers."},
	"concurrency":   {"Doing things at the same time", "Threads and goroutines let several jobs run at once; locks and channels stop them tripping over each other."},
	"memory":        {"Memory and pointers", "A pointer holds the address of a value in memory instead of the value itself. Low-level code and the Windows API work with addresses."},
	"windows-go":    {"Windows Go", "Go can call the Windows API directly: syscall.NewLazyDLL loads a DLL, NewProc finds a function in it and Call runs it, passing every argument as a uintptr. Files marked //go:build windows only build for Windows, so the same project can have a Linux or macOS version of those parts. golang.org/x/sys/windows wraps much of the API more safely."},
	"build-tags":    {"Build constraints", "A //go:build line at the top of a Go file decides which systems it's compiled for. Files can also end in _windows.go or _linux.go, which does the same."},
	"cgo":           {"cgo", "import \"C\" lets Go call C code. It needs a C compiler, so many Windows Go programs avoid it and use syscall instead."},
	"embed":         {"Embedding files", "//go:embed packs files into the program, so it can be handed out as a single .exe."},
	"defer":         {"defer", "defer schedules a call for when the function ends, so clean-up always happens."},
	"async":         {"Async code", "Async functions can wait for slow work without freezing everything else."},
	"generics":      {"Generics / templates", "Code written once that works with many types."},
	"type":          {"Types", "A type says what kind of value something is: number, text, list, your own struct…"},
	"signals":       {"Signals and events", "An object announces that something happened; anything connected to that signal runs. Objects don't need to know about each other."},
	"subroutine":    {"Subroutines (GOSUB)", "GOSUB jumps to a group of lines and RETURN comes back. It's BASIC's original way of reusing code."},
	"goto":          {"GOTO and line numbers", "GOTO jumps to another line. Classic BASIC used it for loops and decisions; later languages replaced it with named loops and functions."},
	"hardware":      {"Talking to hardware", "Reading and writing hardware directly: memory addresses on old computers, pins on microcontrollers."},
	"dom":           {"The web page (DOM)", "JavaScript sees the web page as a tree of elements it can find, change, add and listen to."},
	"security":      {"Security", "Code that stops other people tricking a program, such as cleaning user input before showing it."},
	"saving":        {"Saving", "Keeping progress or settings after the program closes."},
	"game-engine":   {"Game engines and frameworks", "An engine handles windows, drawing, sound, physics and the game loop, so your code only describes your game."},
}

func collectConcepts(lines []LineOut) []ConceptUse {
	by := map[string]*ConceptUse{}
	var order []string
	for _, ln := range lines {
		for _, t := range ln.Tags {
			g, ok := glossary[t]
			if !ok {
				continue
			}
			c := by[t]
			if c == nil {
				c = &ConceptUse{ID: t, Title: g[0], Text: g[1]}
				by[t] = c
				order = append(order, t)
			}
			if len(c.Lines) == 0 || c.Lines[len(c.Lines)-1] != ln.N {
				c.Lines = append(c.Lines, ln.N)
			}
		}
	}
	out := []ConceptUse{}
	for _, id := range order {
		out = append(out, *by[id])
	}
	sort.SliceStable(out, func(i, j int) bool { return len(out[i].Lines) > len(out[j].Lines) })
	return out
}

// ---------------------------------------------------------------- warnings

type warnRule struct {
	langs map[string]bool
	re    *regexp.Regexp
	text  string
	fix   string
	check func(s srcLine) bool
}

var warnRules []warnRule

func W(langs, pat, text, fix string) *warnRule {
	w := warnRule{langs: map[string]bool{}, re: regexp.MustCompile(pat), text: text, fix: fix}
	for _, l := range strings.Split(langs, ",") {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, "@") {
			for _, x := range langs0() {
				if x.Family == l[1:] {
					w.langs[x.ID] = true
				}
			}
		} else {
			w.langs[l] = true
		}
	}
	warnRules = append(warnRules, w)
	return &warnRules[len(warnRules)-1]
}

func init() {
	W("c,cpp,csharp,java,javascript,typescript,php,gml", `\b(if|while)\s*\(\s*[\w.\[\]]+\s*=\s*[^=]`, "This condition uses a single = (which sets a value) where == (which compares) was probably meant.", "Use == to compare.")
	W("javascript,typescript", `[^=!]==[^=]|!=[^=]`, "== and != convert types before comparing (so \"5\" == 5 is true).", "Use === and !== for exact comparisons.")
	W("javascript,typescript", `^\s*var\s`, "var is the old way to declare variables; it ignores blocks and can cause surprises.", "Use let (or const if it never changes).")
	W("c,cpp", `\bgets\s*\(`, "gets() can't limit how much it reads and was removed from C because it causes buffer overflows.", "Use fgets(buffer, sizeof buffer, stdin).")
	W("c,cpp", `\bstrcpy\s*\(|\bsprintf\s*\(`, "This copies text without checking the buffer is big enough.", "Use strncpy/snprintf with the buffer size.")
	W("go", `(?:^|\s)_\s*(?:,\s*_)?\s*=\s*\w[\w.]*\(`, "An error (or result) is being thrown away with _.", "If the call can fail, check the error.")
	W("go", `(?:=|:=)\s*uintptr\(unsafe\.Pointer\(`, "uintptr(unsafe.Pointer(…)) is only safe inside the Call(...) itself, so the garbage collector can't move the memory first.", "Convert pointers in the call's argument list, not in a separate variable.")
	W("go", `syscall\.NewLazyDLL\(`, "NewLazyDLL can load a DLL from the program's own folder, so a planted DLL could be used instead.", "For Windows system DLLs, windows.NewLazySystemDLL (golang.org/x/sys/windows) only loads from System32.")
	W("go", `syscall\.StringToUTF16Ptr\(`, "StringToUTF16Ptr panics if the text contains a zero byte.", "Use syscall.UTF16PtrFromString and check its error.")
	W("python,gdscript", `def\s+\w+\(.*=\s*(\[\]|\{\})`, "A list or dictionary as a default value is shared between every call.", "Use None as the default and make the list inside the function.")
	W("python", `except\s*:`, "A bare except: catches every error, even typing mistakes and Ctrl+C, which hides bugs.", "Catch a specific error such as except ValueError:.")
	W("python", `^\s*from\s+\S+\s+import\s+\*`, "import * pulls in every name, which can silently replace your own.", "Import just the names you need.")
	W("lua", `\[0\]`, "Lua lists start at 1, so index 0 is usually empty.", "Start counting at 1, or use # for the length.")
	W("lua", `while\s+true\s+do\s*$`, "A while true loop with no wait inside will freeze the game (Roblox) or lock up the program.", "Make sure there's a task.wait() or a break inside.")
	W("csharp", `GetComponent<.*?>\(\)|GameObject\.Find\(|FindObjectOfType`, "This search is slow; if it runs every frame (in Update) the game can stutter.", "Call it once in Start or Awake and keep the result in a field.")
	W("csharp", `(?:tag\s*==|==\s*\S*tag)\s*`, "Comparing tag strings with == is slower and easy to misspell.", "Use other.CompareTag(\"Player\").")
	W("@basic", `(?i)\bGOTO\s+\d+.*\bGOTO\s+\d+`, "Several GOTOs on one line make the program hard to follow.", "Later BASICs have IF…THEN…ELSE blocks, loops and SUBs.")
	W("bash", `\brm\s+-rf\s+(/|\$\w+/?\s*$|"\$\w+"/?\s*$)`, "rm -rf on a variable or / can delete far more than intended if the variable is empty.", "Check the variable isn't empty first, or use \"${VAR:?}\".")
	W("bash", `^\s*\w+\s+=\s+`, "In shell scripts, spaces around = break an assignment (it runs a command instead).", "Write NAME=value with no spaces.")
	W("powershell", `(?i)\s(==|!=|<=|>=)\s|\s[<>]\s`, "PowerShell doesn't compare with == or < and >; > even writes to a file!", "Use -eq, -ne, -lt, -gt, -le and -ge.")
	W("sql", `(?i)^\s*(DELETE\s+FROM|UPDATE)\s+\w+\s*;?\s*$`, "This changes or deletes every row in the table: there's no WHERE.", "Add a WHERE clause to choose which rows.")
	W("sql", `(?i)SELECT\s+\*`, "SELECT * fetches every column, which is slower and breaks if columns change.", "List the columns you need.")
	W("php,javascript,python", `(?i)(query|execute|exec)\s*\(\s*["'].*(SELECT|INSERT|UPDATE|DELETE).*["']\s*(\+|\.)`, "SQL built by joining in values can be abused (SQL injection).", "Use prepared statements with placeholders (?).")
	W("javascript,typescript", `\.innerHTML\s*=`, "Setting innerHTML with text from users can let them run scripts in the page (XSS).", "Use textContent for plain text.")
	W("javascript,typescript", `setInterval\(\s*\w*(loop|update|draw|frame)`, "setInterval for a game loop can stutter and keeps running in hidden tabs.", "Use requestAnimationFrame.")
}

func findWarnings(l *Lang, src []srcLine) []Warning {
	out := []Warning{}
	seen := map[string]int{}
	for _, s := range src {
		if s.InBlock || s.InString > 0 || strings.TrimSpace(s.Code) == "" {
			continue
		}
		for _, w := range warnRules {
			if w.text == "" || !w.langs[l.ID] {
				continue
			}
			if w.re.MatchString(s.Code) {
				if seen[w.text] >= 2 {
					continue
				}
				seen[w.text]++
				out = append(out, Warning{Line: s.N, Text: w.text, Fix: w.fix})
			}
		}
	}
	return out
}
