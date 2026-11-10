// Codie — rules_py.go: Python and GDScript (Godot), which share a shape:
// blocks start with a colon and are marked by indentation.
// Part of the free nrmluvdev maker suite. Created by nrmluvdev.
package codie

import (
	"strings"
)

var godotCallbacks = map[string][2]string{
	"_ready":            {"Godot runs «_ready» once, when this node and all its children have entered the scene.", "Set starting values and grab child nodes here (they're guaranteed to exist now)."},
	"_init":             {"Runs when the object is created, before it enters the scene (the constructor).", "Set values that don't need other nodes."},
	"_process":          {"Godot runs «_process» every frame, passing «delta»: the seconds since the last frame.", "Per-frame game logic: input, animation, timers. Multiply movement by delta so the speed is the same at 30 or 144 frames per second."},
	"_physics_process":  {"Godot runs «_physics_process» at a fixed rate (60 times a second by default), passing «delta».", "Movement that uses physics, such as velocity and move_and_slide(), goes here so it's smooth and consistent."},
	"_input":            {"Godot runs «_input» whenever there's an input event: a key, mouse click or touch.", "React to single presses here (open the menu, jump) instead of checking every frame."},
	"_unhandled_input":  {"Runs for input events that no UI element used.", "Game controls go here so clicking a button doesn't also make the player jump."},
	"_draw":             {"Godot runs «_draw» when the node needs repainting; queue_redraw() asks for it.", "Custom drawing with draw_line, draw_circle and friends."},
	"_enter_tree":       {"Runs when the node is added to the scene tree.", ""},
	"_exit_tree":        {"Runs when the node leaves the scene tree.", "Tidy up here."},
	"_on_body_entered":  {"A signal handler, usually connected to an Area2D's body_entered signal.", "Runs when a physics body (like the player) walks into the area: pick-ups, hazards, triggers."},
	"_on_area_entered":  {"A signal handler for an area overlapping another area.", "Used for hitboxes and hurtboxes."},
	"_on_timer_timeout": {"A signal handler for a Timer node's timeout.", "Runs every time the timer finishes: spawning, cooldowns, countdowns."},
}

var pygameNames = map[string][2]string{
	"main": {"«main» holds the program's main steps.", "Calling it from the if __name__ block keeps the file importable."},
}

func init() {
	py := "@py"
	R(py, `^def\s+({I})\s*\((.*?)\)\s*(?:->\s*(.+))?$`, "def", "", "").with(func(m []string, x *ctx) (string, string) {
		name := m[1]
		why := takes(x.lang, m[2])
		what := "Defines a function called " + code(name) + "."
		if strings.HasPrefix(strings.TrimSpace(m[2]), "self") {
			what = "Defines the method " + code(name) + " (a function that belongs to the class above)."
			why += " «self» is the object the method is working on."
		}
		switch name {
		case "__init__":
			what = "Defines «__init__», the setup code that runs when a new object of this class is made."
			why = takes(x.lang, m[2]) + " It usually stores starting values on «self»."
		case "__str__", "__repr__":
			what = "Defines how this object looks when printed."
		case "update":
			x.tag("game-loop")
		}
		if m[3] != "" {
			why += " It gives back " + code(m[3]) + "."
		}
		if v := verbPhrase(name); v != "" && !strings.HasPrefix(name, "__") {
			why = "From its name, it " + v + ". " + why
		}
		x.tag("function")
		return codeMarks.Replace(what), codeMarks.Replace(why)
	}).open("function", 1)
	R("gdscript", `^(static\s+)?func\s+({I})\s*\((.*?)\)\s*(?:->\s*(.+))?$`, "def", "", "").with(func(m []string, x *ctx) (string, string) {
		name := m[2]
		what := "Defines a function called " + code(name) + "."
		why := takes(x.lang, m[3])
		if cb, ok := godotCallbacks[name]; ok {
			what += " " + codeMarks.Replace(cb[0])
			why = codeMarks.Replace(cb[1]) + " " + why
			x.tag("callback")
			if name == "_process" || name == "_physics_process" {
				x.tag("game-loop", "delta-time")
			}
		} else if strings.HasPrefix(name, "_on_") {
			what += " Its name says it handles a signal: " + strings.ReplaceAll(strings.TrimPrefix(name, "_on_"), "_", " ") + "."
			why = "Godot calls it when that signal is emitted (connect it in the Node tab or with connect()). " + why
			x.tag("signals")
		} else if v := verbPhrase(name); v != "" {
			why = "From its name, it " + v + ". " + why
		}
		if m[4] != "" && m[4] != "void" {
			why += " It gives back a " + code(m[4]) + "."
		}
		x.tag("function")
		return what, why
	}).open("function", 2)
	R(py, `^(?:async\s+)?def\s+({I})\s*\((.*?)\)\s*(?:->\s*(.+))?$`, "def", "Defines the async function $1.", "Async functions can pause while waiting (await) and let other work run meanwhile.", "function", "async").open("function", 1)
	R(py, `^class\s+({I})\s*(?:\((.*)\))?$`, "class", "", "").with(func(m []string, x *ctx) (string, string) {
		w := "Defines a class called " + code(m[1]) + "."
		why := "A class is a blueprint for objects: it bundles values and the methods that work on them."
		if b := strings.TrimSpace(m[2]); b != "" && b != "object" {
			w += " It builds on " + code(b) + "."
			why += " It inherits everything " + code(b) + " can do."
			if strings.Contains(b, "pygame.sprite.Sprite") || b == "Sprite" {
				why = "Inheriting from pygame's Sprite lets it go in sprite groups that update, draw and check collisions for many objects at once."
				x.tag("game-engine")
			}
			x.tag("inheritance")
		}
		x.tag("class")
		return w, why
	}).open("class", 1)
	R(py, `^if\s+__name__\s*==\s*{S}$`, "if", "Runs the code below only when this file is started directly (not imported).", "Python's usual way to mark the program's starting point, so other files can import its functions without running it.", "if").first().open("if", 0)
	R(py, `^if\s+(.+)$`, "if", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("if")
		if one := strings.Index(m[1], ":"); one > 0 && !strings.Contains(m[1][:one], "[") && strings.TrimSpace(m[1][one+1:]) != "" {
			return "If-check: " + checks(x, m[1][:one]) + "; if so, runs " + code(unhide(strings.TrimSpace(m[1][one+1:]), x.strs)) + ".", "A one-line if."
		}
		return "If-check: " + checks(x, m[1]) + ".", "The indented lines below only run when it's true."
	}).open("if", 0)
	R(py, `^elif\s+(.+)$`, "if", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("if")
		return "Otherwise, if-check: " + checks(x, m[1]) + ".", "elif means 'else if': only tested when the checks above were false."
	}).open("if", 0)
	R(py, `^else$`, "if", "Otherwise (when none of the checks above were true), run the indented lines below.", "", "if").open("if", 0)
	R("python", `^for\s+(.+?)\s+in\s+range\((.*)\)$`, "loop", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("loop")
		a := splitTop(m[2], ',')
		v := code(m[1])
		switch len(a) {
		case 1:
			return "Repeats the block " + code(a[0]) + " times, with " + v + " counting 0, 1, 2… up to one less than " + code(a[0]) + ".", "range counts from 0 and stops just before the number, so range(10) gives 0 to 9."
		case 2:
			return "Counts " + v + " from " + code(a[0]) + " up to (but not including) " + code(a[1]) + ", running the block each time.", "range never includes the end number."
		case 3:
			return "Counts " + v + " from " + code(a[0]) + " towards " + code(a[1]) + " in steps of " + code(a[2]) + ".", "The third number is the step size; a negative step counts down."
		}
		return "", ""
	}).open("loop", 0).setsVar(1)
	R("gdscript", `^for\s+(\w+)\s+in\s+range\((.*)\)$`, "loop", "Counts $1 through range($2), running the block each time.", "range(5) gives 0, 1, 2, 3, 4.", "loop").open("loop", 0).setsVar(1)
	R(py, `^for\s+(.+?)\s+in\s+enumerate\((.*)\)$`, "loop", "Goes through $2, giving both the position and the item each time ($1).", "enumerate saves keeping a separate counter.", "loop").open("loop", 0).setsVar(1)
	R(py, `^for\s+(.+?)\s+in\s+(.+)$`, "loop", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("loop")
		src := strings.TrimSpace(m[2])
		if strings.HasPrefix(src, "pygame.event.get()") {
			x.tag("input", "game-loop")
			return "Goes through every event (key press, mouse click, window close…) that happened since the last frame, calling each one " + code(m[1]) + ".", "Every pygame game loop must read the events, or the window stops responding."
		}
		return "Goes through each item in " + code(unhide(src, x.strs)) + ", calling it " + code(m[1]) + " each time.", "A for loop runs the indented lines once for every item."
	}).open("loop", 0).setsVar(1)
	R(py, `^while\s+(.+)$`, "loop", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("loop")
		c := strings.TrimSpace(m[1])
		if c == "True" || c == "true" || c == "running" || c == "not done" || c == "run" || c == "1" {
			x.tag("game-loop")
			return "Starts the main loop: keeps repeating while " + code(c) + ".", "Games and programs that wait for input run a loop like this until told to stop (often by setting " + code(c) + " to False or using break)."
		}
		return "A while-loop: " + checks(x, c) + ", and repeats the indented lines for as long as that stays true.", "Use while when you don't know in advance how many times to repeat."
	}).open("loop", 0)
	R(py, `^try$`, "try", "Tries the indented lines; if any of them fail, Python jumps to the except part instead of crashing.", "Used around steps that can fail: files, networks, turning text into numbers.", "errors").open("try", 0)
	R(py, `^except\s*(.*?)(?:\s+as\s+(\w+))?$`, "try", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("errors")
		if m[1] == "" {
			return "Handles any error from the try block above.", "Catching every error can hide bugs; naming the error type is safer."
		}
		w := "Handles a " + code(m[1]) + " error from the try block above"
		if m[2] != "" {
			w += ", keeping the error in " + code(m[2])
		}
		return w + ".", "This is where the program recovers from the problem."
	}).open("try", 0)
	R(py, `^finally$`, "try", "Code that always runs after the try, error or not.", "Clean-up like closing files.", "errors").open("try", 0)
	R("python", `^with\s+(.+?)(?:\s+as\s+(\w+))?$`, "try", "", "").with(func(m []string, x *ctx) (string, string) {
		w := "Uses " + code(unhide(m[1], x.strs))
		if m[2] != "" {
			w += " as " + code(m[2])
		}
		return w + " for the indented block, closing it automatically afterwards.", "with makes sure files and connections are always closed, even if an error happens."
	}).open("block", 0).setsVar(2)
	R(py, `^return\s*(.*)$`, "return", "", "").with(func(m []string, x *ctx) (string, string) {
		if m[1] == "" {
			return "Leaves the function now.", "The rest of the function is skipped."
		}
		return "Leaves the function and gives back " + describeExpr(x, m[1]) + ".", "The caller receives this value."
	})
	R("python", `^yield\s+(.+)$`, "return", "Hands back $1 and pauses here until the next value is asked for.", "This makes the function a generator: it produces values one at a time.")
	R(py, `^break$`, "jump", "Stops the loop straight away.", "", "loop")
	R(py, `^continue$`, "jump", "Skips to the next round of the loop.", "", "loop")
	R(py, `^pass$`, "other", "Does nothing.", "Python needs at least one line in a block; pass is a placeholder.")
	R(py, `^raise\s*(.*)$`, "error", "Raises the error $1.", "Stops the function and passes the problem to the nearest except.", "errors")
	R(py, `^assert\s+(.+)$`, "error", "Checks that $1 is true, and stops with an error if not.", "A quick sanity check, mostly while developing.", "errors")
	R(py, `^global\s+(.+)$`, "var", "Says $1 means the variable shared by the whole file, not a new one inside this function.", "Without it, assigning inside a function makes a separate local variable.", "variable")
	R("python", `^import\s+([\w.]+)(?:\s+as\s+(\w+))?$`, "import", "", "").with(func(m []string, x *ctx) (string, string) {
		d := map[string]string{"pygame": "a library for making 2D games: windows, drawing, sound, input", "random": "random numbers", "math": "maths functions", "time": "clocks and waiting", "os": "files, folders and the system", "sys": "facts about Python itself and the program's arguments", "json": "reading and writing JSON", "turtle": "turtle drawing", "tkinter": "simple desktop windows and buttons", "requests": "web requests", "numpy": "fast number arrays", "serial": "serial ports (pyserial)", "RPi.GPIO": "Raspberry Pi pins", "machine": "MicroPython hardware pins", "arcade": "the Arcade 2D game library", "pyglet": "the pyglet game library"}[m[1]]
		w := "Imports the module " + code(m[1])
		if d != "" {
			w += ": " + d
		}
		if m[2] != "" {
			w += ", under the short name " + code(m[2])
		}
		if m[1] == "pygame" || m[1] == "arcade" || m[1] == "pyglet" {
			x.tag("game-engine")
		}
		x.tag("import")
		return w + ".", "After importing, its functions are used as " + m[1] + ".something."
	})
	R("python", `^from\s+([\w.]+)\s+import\s+(.+)$`, "import", "Imports $2 from the module $1.", "These names can then be used directly, without the module name in front.", "import")
	R(py, `^@(\w[\w.]*)(\(.*\))?$`, "directive", "", "").with(func(m []string, x *ctx) (string, string) {
		switch m[1] {
		case "staticmethod":
			return "Makes the next method a static method: it doesn't need an object (no self).", ""
		case "classmethod":
			return "Makes the next method work on the class itself (cls) rather than one object.", ""
		case "property":
			return "Lets the next method be read like a value (obj.name, not obj.name()).", ""
		case "dataclass":
			return "Turns the next class into a data class: Python writes __init__ and printing for you.", ""
		case "export", "onready", "tool", "icon":
			return "", ""
		}
		return "Wraps the next function with " + code(m[1]) + " (a decorator).", "A decorator adds behaviour to a function without changing its code."
	})
	R(py, `^lambda\s+(.*?):\s*(.+)$`, "def", "A tiny unnamed function taking $1 and giving back $2.", "", "function")

	// ------------------------------------------------------------ GDScript
	R("gdscript", `^extends\s+(.+)$`, "class", "", "").with(func(m []string, x *ctx) (string, string) {
		b := strings.TrimSpace(m[1])
		d := map[string]string{
			"CharacterBody2D": "a 2D character moved by code (player, enemy) with velocity and move_and_slide()",
			"CharacterBody3D": "a 3D character moved by code with velocity and move_and_slide()",
			"KinematicBody2D": "a 2D character moved by code (Godot 3; called CharacterBody2D in Godot 4)",
			"RigidBody2D":     "a 2D object moved by the physics engine (falls, bounces)",
			"RigidBody3D":     "a 3D object moved by the physics engine",
			"Area2D":          "an invisible 2D zone that detects things entering it (pick-ups, triggers)",
			"Node2D":          "a basic 2D object with a position",
			"Node":            "the most basic Godot node",
			"Sprite2D":        "a 2D picture",
			"Control":         "a user-interface element",
			"Camera2D":        "a 2D camera",
			"Node3D":          "a basic 3D object",
		}[b]
		w := "Makes this script build on the Godot class " + code(b) + "."
		if d != "" {
			w = "Makes this script a " + code(b) + ": " + d + "."
		}
		x.tag("inheritance", "game-engine")
		return w, "The script gets everything that node type can do, and Godot calls its _ready and _process functions."
	})
	R("gdscript", `^class_name\s+(\w+)`, "class", "Registers this script as a new type called $1.", "Other scripts can then use $1 directly, and it appears in Godot's Add Node list.", "class")
	R("gdscript", `^signal\s+({I})\s*(?:\((.*)\))?$`, "def", "Declares the signal $1, which this object can send out.", "Signals let objects announce events (died, coin_collected) without knowing who's listening.", "signals")
	R("gdscript", `^(?:@export(?:_\w+)?(?:\(.*?\))?|export(?:\(.*?\))?)\s+var\s+({I})\s*(?::\s*(\w+))?\s*(?::?=\s*(.+))?$`, "var", "", "").with(func(m []string, x *ctx) (string, string) {
		w := "Creates " + code(m[1]) + " and shows it in Godot's Inspector"
		if m[3] != "" {
			w += ", starting at " + describeExpr(x, m[3])
		}
		x.tag("variable")
		return w + ".", "Exported variables can be tuned in the editor for each copy of the object, without touching code. " + varWhy(x, m[1], "")
	}).setsVar(1)
	R("gdscript", `^(?:@onready|onready)\s+var\s+({I})\s*(?::\s*\w+)?\s*:?=\s*(.+)$`, "var", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("variable")
		return "Creates " + code(m[1]) + ", filled with " + describeExpr(x, m[2]) + " once the node is ready.", "@onready waits until _ready, when child nodes like $Sprite exist, so getting them won't fail."
	}).setsVar(1)
	R("gdscript", `^(var|const)\s+({I})\s*(?::\s*([\w\[\]]+))?\s*(:?=)?\s*(.*)$`, "var", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("variable")
		if m[5] == "" {
			w := "Creates the variable " + code(m[2])
			if m[3] != "" {
				w += " to hold a " + code(m[3])
			}
			return w + ".", varWhy(x, m[2], "")
		}
		p := assignPart(x, m[1], m[2], "", m[5])
		if m[4] == ":=" {
			p.Why += " := makes Godot fix the type from the value."
		}
		if m[1] == "const" {
			p.What = "Creates the constant " + code(m[2]) + " with the value " + describeExpr(x, m[5]) + "."
		}
		return p.What, p.Why
	}).setsVar(2)
	R("gdscript", `^enum\s*(\w*)\s*\{(.*)\}$`, "class", "Defines the named choices $2.", "Enums give names to states, such as IDLE, RUN, JUMP, which is clearer than numbers.", "state-machine")
	R("gdscript", `^match\s+(.+)$`, "switch", "Chooses what to do based on $1.", "GDScript's match is like a switch: each pattern below is checked in turn.", "switch").open("switch", 0)
	R("gdscript", `^([\w.]+|{S}|_)\s*$`, "case", "", "").with(func(m []string, x *ctx) (string, string) {
		if x.top() != "switch" || !strings.HasSuffix(x.cur, ":") {
			return "", ""
		}
		if m[1] == "_" {
			return "If nothing else matched, run the lines below.", ""
		}
		return "If the value is " + expand("$1", m, x) + ", run the indented lines below.", ""
	}).open("case", 0)
	R("gdscript", `^(await|yield)\s*\(?(.+?)\)?$`, "call", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("timers")
		if strings.Contains(m[2], "create_timer") {
			return "Pauses this function until the timer finishes, then carries on.", "await lets you write 'wait 1 second, then…' without stopping the rest of the game."
		}
		return "Pauses this function until " + code(unhide(m[2], x.strs)) + " happens.", "The rest of the game keeps running while this function waits."
	})
	R("gdscript", `^velocity\s*=\s*move_and_slide\((.*)\)$`, "call", "Moves the body using its velocity, sliding along walls and floors, and keeps the new velocity.", "This is Godot 3's character movement; Godot 4 just calls move_and_slide() and uses the velocity property.", "physics", "collision")
	R("gdscript", `^\$({I}(?:/{I})*)\.(\w+)\((.*)\)$`, "call", "", "").with(func(m []string, x *ctx) (string, string) {
		w := "Calls " + code(m[2]+"()") + " on the child node " + code(m[1])
		if m[3] != "" {
			w += " with " + code(unhide(m[3], x.strs))
		}
		why := "$Name is short for get_node(\"Name\"): it finds a child node in this scene."
		if e, _ := lookupAPIName(x, "."+m[2]); e != nil {
			why = lowerFirst(e.What) + " " + why
			why = strings.ToUpper(why[:1]) + why[1:]
		}
		return w + ".", why
	})
}
