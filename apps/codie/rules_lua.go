// Codie — rules_lua.go: Lua (with LÖVE, PICO-8, Roblox Luau and Solar2D)
// and Ruby, the two "keyword … end" languages.
// Part of the free nrmluvdev maker suite. Created by nrmluvdev.
package codie

import (
	"strings"
)

var luaCallbacks = map[string][2]string{
	"love.load":         {"LÖVE runs «love.load» once when the game starts.", "Load pictures and sounds and set starting values here."},
	"love.update":       {"LÖVE runs «love.update(dt)» every frame; «dt» is the seconds since the last frame.", "All game logic goes here: input, movement, collisions, timers. Multiply speeds by dt so the game runs the same on every computer."},
	"love.draw":         {"LÖVE runs «love.draw» every frame, after update, to paint the screen.", "Only drawing goes here; the screen is cleared before each call."},
	"love.keypressed":   {"LÖVE runs «love.keypressed(key)» once each time a key goes down.", "Good for single actions: jump, pause, open a menu. For holding keys, check love.keyboard.isDown in update."},
	"love.keyreleased":  {"LÖVE runs this when a key is let go.", ""},
	"love.mousepressed": {"LÖVE runs this when a mouse button is pressed, with the position and button.", ""},
	"love.conf":         {"LÖVE reads «love.conf(t)» (in conf.lua) before starting, to set the window size and title.", ""},
	"love.quit":         {"LÖVE runs this when the game is closing.", "A good place to save."},
	"_init":             {"PICO-8 runs «_init» once when the cartridge starts.", "Set up starting values here."},
	"_update":           {"PICO-8 runs «_update» 30 times a second.", "Game logic goes here: read buttons with btn(), move things, check collisions. PICO-8 runs at a fixed speed, so there's no delta time."},
	"_update60":         {"PICO-8 runs «_update60» 60 times a second (instead of _update).", "Smoother movement; game logic goes here."},
	"_draw":             {"PICO-8 runs «_draw» after each update to paint the 128×128 screen.", "Usually starts with cls() and then draws everything with spr(), map() and print()."},
}

func init() {
	lua := "lua"
	R(lua, `^(local\s+)?function\s+({I}(?:[.:]{I})*)\s*\((.*?)\)(.*)$`, "def", "", "").with(func(m []string, x *ctx) (string, string) {
		name := m[2]
		what := "Defines a function called " + code(name) + "."
		why := takes(x.lang, m[3])
		if m[1] != "" {
			what = "Defines a local function " + code(name) + " (only usable in this file or block)."
		}
		if strings.Contains(name, ":") {
			i := strings.Index(name, ":")
			what = "Defines the method " + code(name[i+1:]) + " for " + code(name[:i]) + "."
			why += " The colon gives it a hidden first input, «self»: the object it was called on."
		}
		if cb, ok := luaCallbacks[name]; ok {
			what += " " + codeMarks.Replace(cb[0])
			why = codeMarks.Replace(cb[1]) + " " + why
			x.tag("callback")
			if name == "love.update" || name == "_update" || name == "_update60" {
				x.tag("game-loop")
			}
			if name == "love.update" {
				x.tag("delta-time")
			}
		} else if v := verbPhrase(lastSeg(name)); v != "" {
			why = "From its name, it " + v + ". " + why
		}
		if strings.TrimSpace(m[4]) != "" && strings.HasSuffix(strings.TrimSpace(m[4]), "end") {
			what += " The whole function fits on this line."
		}
		x.tag("function")
		return codeMarks.Replace(what), codeMarks.Replace(why)
	}).open("function", 2)
	R(lua, `^(local\s+)?({I}(?:\.{I})*)\s*=\s*function\s*\((.*?)\)(.*)$`, "def", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("function")
		return "Stores a new function in " + code(m[2]) + ".", "In Lua, functions are values you can store in variables and tables. " + takes(x.lang, m[3])
	}).open("function", 2).setsVar(2)
	R(lua, `^local\s+(.+?)\s*=\s*(.+)$`, "var", "", "").with(func(m []string, x *ctx) (string, string) {
		p := assignPart(x, "local", m[1], "", m[2])
		if strings.TrimSpace(m[2]) == "{}" {
			p.What = "Creates " + code(m[1]) + " as a new, empty table."
			p.Why = "Tables are Lua's one data structure: they work as lists ({1,2,3}), records ({x=1}) and objects. " + p.Why
			x.tag("table")
		}
		if strings.HasPrefix(strings.TrimSpace(m[2]), "require") {
			p.Why = "require loads another Lua file (module) and gives back what it returns."
			x.tag("import")
		}
		return p.What, p.Why + " «local» keeps it private to this file or block, which is faster and avoids clashes."
	}).setsVar(1)
	R(lua, `^local\s+({I}(?:\s*,\s*{I})*)$`, "var", "Creates the local variable(s) $1, empty for now (nil).", "They'll be given values later.", "variable").setsVar(1)
	R(lua, `^if\s+(.+?)\s+then(?:\s+(.*?))?$`, "if", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("if")
		body := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(m[2]), "end"))
		if body != "" {
			return "If-check: " + checks(x, m[1]) + "; if so, runs " + code(unhide(body, x.strs)) + ".", "A one-line if…then…end."
		}
		return "If-check: " + checks(x, m[1]) + ".", "The lines up to the matching end (or else) only run when it's true. In Lua only nil and false count as false; 0 counts as true."
	}).open("if", 0)
	R(lua, `^elseif\s+(.+?)\s+then$`, "if", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("if")
		return "Otherwise, if-check: " + checks(x, m[1]) + ".", "Only tested when the checks above were false."
	})
	R(lua, `^else$`, "if", "Otherwise, run the lines below (up to end).", "", "if")
	R(lua, `^for\s+({I})\s*=\s*(.+?)\s*,\s*(.+?)(?:\s*,\s*(.+?))?\s+do$`, "loop", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("loop")
		w := "Counts " + code(m[1]) + " from " + code(m[2]) + " to " + code(m[3])
		if m[4] != "" {
			w += " in steps of " + code(m[4])
		}
		return w + ", running the block each time.", "Lua's counting loop includes both ends: for i = 1, 3 gives 1, 2 and 3."
	}).open("loop", 0).setsVar(1)
	R(lua, `^for\s+(.+?)\s+in\s+(ipairs|pairs)\((.+)\)\s+do$`, "loop", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("loop", "table")
		v := splitTop(m[1], ',')
		if m[2] == "ipairs" {
			return "Goes through the list " + code(m[3]) + " in order, with " + code(v[0]) + " as the position (1, 2, 3…)" + andValue(v) + ".", "ipairs walks a list from 1 until it finds an empty slot. Lua lists start at 1, not 0."
		}
		return "Goes through every key and value in the table " + code(m[3]) + andValue(v) + ".", "pairs visits every entry, in no particular order."
	}).open("loop", 0).setsVar(1)
	R(lua, `^for\s+(.+?)\s+in\s+(.+?)\s+do$`, "loop", "Goes through what $2 produces, calling it $1 each time.", "A generic for loop.", "loop").open("loop", 0).setsVar(1)
	R(lua, `^while\s+(.+?)\s+do$`, "loop", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("loop")
		if strings.TrimSpace(m[1]) == "true" {
			x.tag("game-loop")
			return "Repeats forever (while true).", "In Roblox this is a common way to run something every few seconds, with task.wait() inside so it doesn't freeze the game."
		}
		return "A while-loop: " + checks(x, m[1]) + ", and repeats for as long as it stays true.", ""
	}).open("loop", 0)
	R(lua, `^repeat$`, "loop", "Starts a repeat…until loop: the block runs at least once.", "", "loop").open("loop", 0)
	R(lua, `^until\s+(.+)$`, "close", "…and repeats until $1 becomes true.", "", "loop").close()
	R(lua, `^end\)?$`, "close", "", "").close()
	R(lua, `^return\s*(.*)$`, "return", "", "").with(func(m []string, x *ctx) (string, string) {
		if m[1] == "" {
			return "Leaves the function now.", ""
		}
		if len(x.stack) == 0 {
			return "Gives back " + describeExpr(x, m[1]) + " as this file's result.", "A module file returns its table; whoever requires it receives that table."
		}
		return "Leaves the function and gives back " + describeExpr(x, m[1]) + ".", "The caller receives this value."
	})
	R(lua, `^break$`, "jump", "Stops the loop straight away.", "", "loop")
	R(lua, `^goto\s+(\w+)$`, "jump", "Jumps to the label ::$1::.", "Lua 5.2+ uses goto mostly as a 'continue' for loops.")
	R(lua, `^::(\w+)::$`, "label", "A label called $1 that goto can jump to.", "")
	R(lua, `^({D})\.({I}):Connect\((.*)\)$`, "call", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("signals")
		ev := m[2]
		d := map[string]string{"Touched": "when something touches this part", "Heartbeat": "every frame, after physics", "RenderStepped": "every frame, before drawing (client only)", "Stepped": "every frame, before physics", "PlayerAdded": "when a player joins", "PlayerRemoving": "when a player leaves", "InputBegan": "when a key, mouse button or touch starts", "InputEnded": "when a key or button is let go", "MouseButton1Click": "when the button is clicked", "Died": "when the character dies", "CharacterAdded": "when the player's character spawns", "OnServerEvent": "when a client fires this RemoteEvent", "OnClientEvent": "when the server fires this RemoteEvent", "Changed": "whenever a property changes"}[ev]
		w := "Connects a function to the " + code(ev) + " event of " + code(m[1])
		if d != "" {
			w += ": it will run " + d
		}
		if ev == "Heartbeat" || ev == "RenderStepped" || ev == "Stepped" {
			x.tag("game-loop", "delta-time")
		}
		if ev == "Touched" {
			x.tag("collision")
		}
		if ev == "InputBegan" {
			x.tag("input")
		}
		return w + ".", "Roblox is event-driven: instead of checking every frame, you tell it which function to run when something happens."
	}).open("event", 2)
	R("ruby", `^def\s+(self\.)?({I}[?!=]?)\s*(?:\((.*?)\)|\s+(.*))?$`, "def", "", "").with(func(m []string, x *ctx) (string, string) {
		p := m[3]
		if p == "" {
			p = m[4]
		}
		name := m[2]
		w := "Defines the method " + code(name) + "."
		why := takes(x.lang, p)
		switch {
		case name == "initialize":
			w = "Defines initialize: the setup code run when «.new» makes an object."
		case strings.HasSuffix(name, "?"):
			why = "Methods ending in ? answer yes or no. " + why
		case strings.HasSuffix(name, "!"):
			why = "Methods ending in ! usually change the object itself or are 'dangerous'. " + why
		}
		if m[1] != "" {
			w = "Defines the class method " + code(name) + " (called on the class, not an object)."
		}
		x.tag("function")
		return codeMarks.Replace(w), why
	}).open("function", 2)
	R("ruby", `^class\s+({I}(?:::{I})*)(?:\s*<\s*(.+))?$`, "class", "", "").with(func(m []string, x *ctx) (string, string) {
		if m[2] != "" {
			x.tag("inheritance")
			return "Defines the class " + code(m[1]) + ", built on " + code(m[2]) + ".", "It inherits everything " + m[2] + " can do."
		}
		return classExplain(x, "class", m[1], "")
	}).open("class", 1)
	R("ruby", `^module\s+(\w+)$`, "class", "Defines the module $1: a named group of methods and constants.", "Modules organise code and can be mixed into classes.").open("module", 1)
	R("ruby", `^attr_(accessor|reader|writer)\s+(.+)$`, "var", "", "").with(func(m []string, x *ctx) (string, string) {
		d := map[string]string{"accessor": "read and change", "reader": "read", "writer": "change"}[m[1]]
		return "Lets other code " + d + " " + code(m[2]) + " on objects of this class.", "A shortcut that writes the getter/setter methods for you."
	})
	R("ruby", `^(if|unless|elsif)\s+(.+?)(?:\s+then)?$`, "if", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("if")
		switch m[1] {
		case "unless":
			return "Runs the block unless " + code(unhide(m[2], x.strs)) + " (only when it's false).", "unless is if-not."
		case "elsif":
			return "Otherwise, if-check: " + checks(x, m[2]) + ".", ""
		}
		return "If-check: " + checks(x, m[2]) + ".", "The lines up to end only run when it's true."
	}).open("if", 0)
	R("ruby", `^else$`, "if", "Otherwise, run the lines below.", "", "if")
	R("ruby", `^(while|until)\s+(.+?)(?:\s+do)?$`, "loop", "Repeats $1 $2.", "", "loop").open("loop", 0)
	R("ruby", `^(.+?)\.each(?:_with_index)?\s+do\s*\|(.+?)\|$`, "loop", "Goes through each item of $1, calling it $2.", "each runs the block once per item: Ruby's usual loop.", "loop").open("loop", 0)
	R("ruby", `^(\d+|\w+)\.times\s+do(?:\s*\|(.+?)\|)?$`, "loop", "Repeats the block $1 times.", "", "loop").open("loop", 0)
	R("ruby", `^end$`, "close", "", "").close()
	R("ruby", `^puts\s+(.+)$`, "print", "Prints $1 followed by a new line.", "puts is Ruby's usual way to show text.", "output")
	R("ruby", `^require(?:_relative)?\s+({S})$`, "import", "Loads the library or file $1.", "", "import")
	R("ruby", `^return\s*(.*)$`, "return", "Leaves the method, giving back $1.", "Ruby also returns the last value automatically.")
	R("ruby", `^({I})\.({I}[?!]?)(?:\s+(.+))?$`, "call", "", "").with(func(m []string, x *ctx) (string, string) {
		p := callPart(x, m[1]+"."+m[2], m[3])
		return p.What, p.Why + " Ruby lets you leave out the brackets."
	})
	R("ruby", `^(.+?)\s+if\s+(.+)$`, "if", "Does $1, but only if $2.", "Ruby lets you put a short if at the end of a line.", "if")
}

func andValue(v []string) string {
	if len(v) > 1 {
		return " and " + code(v[1]) + " as the value"
	}
	return ""
}
