// Codie — generic.go: explanations for lines that no specific rule knows:
// assignments, calls and the like, worded from the names the author chose.
// Part of the free nrmluvdev maker suite. Created by nrmluvdev.
package codie

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

var (
	reIncr    = regexp.MustCompile(`^(?:(\+\+|--)\s*([\w$.\[\]>-]+)|([\w$.\[\]>-]+)\s*(\+\+|--));?$`)
	reAssign  = regexp.MustCompile(`^(?:(let|var|const|local|auto|val|my|our|global|static|dim|set|LET|DIM|SET|Local|Dim)\s+)?([\w$@.\[\]\x01\x02>:,\s*-]+?)\s*(\+|-|\*|/|%|\.\.|\|\||&&|\?\?|<<|>>|\||&|\^|:|\*\*|//)?=\s*([^=].*|)$`)
	reCall    = regexp.MustCompile(`^(?:await\s+|new\s+|yield\s+|call\s+|CALL\s+|go\s+|defer\s+)?([A-Za-z_$@][\w$]*(?:(?:\.|::|->|:)[A-Za-z_$][\w$]*)*)\s*(?:<[^()]*>)?\s*\((.*)\)\s*;?$`)
	reChain   = regexp.MustCompile(`^(.+?)(?:\.|:|->)([A-Za-z_]\w*)\s*\((.*)\)\s*;?$`)
	reTokCall = regexp.MustCompile(`([A-Za-z_$@][\w$]*(?:(?:\.|::|->|:)[A-Za-z_$][\w$]*)*)\s*\(`)
	reTokDot  = regexp.MustCompile(`[A-Za-z_][\w]*(?:\.[A-Za-z_]\w*)+`)
	reNum     = regexp.MustCompile(`^-?\d+(\.\d+)?[fFdDlLuU]?$|^0x[0-9a-fA-F]+$|^\$[0-9A-Fa-f]+$|^&[Hh][0-9A-Fa-f]+$`)
)

func generic(x *ctx, t string) (Part, bool) {
	t = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(t), ";"))
	if m := reIncr.FindStringSubmatch(t); m != nil {
		v, op := m[2], m[1]
		if v == "" {
			v, op = m[3], m[4]
		}
		w := "Adds 1 to " + code(v) + "."
		if op == "--" {
			w = "Takes 1 away from " + code(v) + "."
		}
		return Part{Kind: "assign", What: w, Why: varWhy(x, v, "Counting up or down by one is how loops, scores and timers usually move on.")}, true
	}
	if m := reAssign.FindStringSubmatch(t); m != nil && !strings.ContainsAny(m[2], "()") && strings.TrimSpace(m[2]) != "" {
		return assignPart(x, m[1], strings.TrimSpace(m[2]), m[3], strings.TrimSpace(m[4])), true
	}
	if m := reCall.FindStringSubmatch(t); m != nil {
		return callPart(x, m[1], m[2]), true
	}
	if m := reChain.FindStringSubmatch(t); m != nil {
		return callPart(x, m[2], m[3]), true
	}
	// A line that is just an API name (Lua: "cls()" is caught above; BASIC: "CLS").
	if e, key := lookupAPI(x, t); e != nil {
		x.tag(e.Tags...)
		return Part{Kind: kindOr(e.Kind, "call"), What: e.What, Why: e.Why}, key != ""
	}
	if p, ok := apiInLine(x, t); ok {
		return p, true
	}
	return Part{Kind: "other", What: "Codie isn't sure what this line does on its own.", Why: "It may continue the statement above, or use something Codie hasn't learned yet."}, false
}

func kindOr(k, d string) string {
	if k == "" {
		return d
	}
	return k
}

func code(s string) string { return "`" + clip(s, 80) + "`" }

func assignPart(x *ctx, decl, target, op, expr string) Part {
	shownT := unhide(target, x.strs)
	e := describeExpr(x, expr)
	var w string
	isNew := decl != "" || op == ":"
	if !isNew {
		if _, ok := x.vars[baseName(target)]; !ok && !strings.ContainsAny(target, ".[") {
			isNew = true
		}
	}
	switch op {
	case "", ":":
		if strings.Contains(target, ",") {
			w = "Sets " + code(shownT) + " (several values at once) to " + e + "."
		} else if isNew && decl != "set" && decl != "SET" && decl != "LET" {
			w = "Creates " + code(shownT) + " and sets it to " + e + "."
		} else {
			w = "Sets " + code(shownT) + " to " + e + "."
		}
		if decl == "const" {
			w = "Creates the constant " + code(shownT) + " (a value that never changes) set to " + e + "."
		}
	case "+":
		w = "Adds " + e + " to " + code(shownT) + "."
	case "-":
		w = "Takes " + e + " away from " + code(shownT) + "."
	case "*":
		w = "Multiplies " + code(shownT) + " by " + e + "."
	case "/", "//":
		w = "Divides " + code(shownT) + " by " + e + "."
	case "%":
		w = "Sets " + code(shownT) + " to the remainder after dividing it by " + e + "."
	case "**":
		w = "Raises " + code(shownT) + " to the power " + e + "."
	case "..":
		w = "Adds " + e + " to the end of the text in " + code(shownT) + "."
	case "||", "??":
		w = "Sets " + code(shownT) + " to " + e + ", but only if it doesn't already have a value."
	case "&&":
		w = "Keeps " + code(shownT) + " true only if " + e + " is true too."
	default:
		w = "Combines the bits of " + code(shownT) + " with " + e + " (" + op + "=, a bitwise operation)."
	}
	for _, v := range strings.Split(target, ",") {
		v = baseName(strings.TrimSpace(v))
		if v != "" {
			if _, ok := x.vars[v]; !ok {
				x.vars[v] = x.n
			}
		}
	}
	x.tag("variable")
	if op != "" && op != ":" {
		x.tag("operators")
	}
	apiNotes(x, expr, "")
	return Part{Kind: "assign", What: w, Why: varWhy(x, target, "")}
}

// baseName is the variable part of a.b[3] → a.b
func baseName(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimLeft(s, "*&$@")
	if i := strings.IndexAny(s, "[ "); i > 0 {
		s = s[:i]
	}
	return s
}

func varWhy(x *ctx, target, fallback string) string {
	base := baseName(target)
	short := base
	for _, p := range []string{"self.", "this.", "self:", "@"} {
		short = strings.TrimPrefix(short, p)
	}
	if i := strings.LastIndexAny(short, ".>:"); i >= 0 {
		short = short[i+1:]
	}
	meaning := nameMeaning(short)
	if segs := strings.FieldsFunc(base, func(r rune) bool { return r == '.' || r == ':' }); len(segs) > 1 {
		if axis, ok := map[string]string{"x": "sideways (x)", "y": "up/down (y)", "z": "depth (z)"}[strings.ToLower(short)]; ok {
			owner := segs[len(segs)-2]
			if om := nameMeaning(owner); om != "" {
				meaning = "is the " + axis + " part of " + code(owner) + ", which " + om
			} else {
				meaning = "is the " + axis + " part of " + code(owner)
			}
		}
	}
	var parts []string
	if first, ok := x.vars[base]; ok && first < x.n && !x.first {
		parts = append(parts, fmt.Sprintf("%s was first set on line %d.", code(base), first))
	}
	if meaning != "" {
		parts = append(parts, "From its name, "+code(short)+" probably "+meaning+".")
	} else if fallback == "" {
		parts = append(parts, "Variables are named boxes that hold a value the program needs later.")
	}
	if fallback != "" {
		parts = append(parts, fallback)
	}
	return strings.Join(parts, " ")
}

func describeExpr(x *ctx, e string) string {
	e = strings.TrimSpace(strings.TrimSuffix(e, ";"))
	shown := unhide(e, x.strs)
	lo := strings.ToLower(e)
	switch {
	case e == "":
		return "nothing yet"
	case reNum.MatchString(e):
		return code(shown)
	case regexp.MustCompile("^" + STR + "$").MatchString(e):
		return "the text " + shown
	case lo == "true" || lo == "false":
		return code(shown) + " (a yes/no value)"
	case lo == "null" || lo == "nil" || lo == "none" || lo == "undefined" || lo == "nothing" || lo == "$null":
		return code(shown) + " (no value yet)"
	case e == "[]" || e == "{}" || e == "()" || e == "@()" || e == "list()" || e == "dict()" || e == "{ }":
		if e == "{}" && (x.lang.Family == "lua") {
			return "an empty table"
		}
		if e == "{}" {
			return "an empty object/collection " + code(e)
		}
		return "an empty list " + code(e)
	case strings.HasPrefix(e, "new "):
		n := strings.TrimSpace(e[4:])
		if i := strings.IndexAny(n, "(<[{"); i > 0 {
			n = n[:i]
		}
		return "a new " + code(n) + " object"
	case strings.HasPrefix(e, "&") && reCall.MatchString(e[1:]) == false && strings.Contains(e, "{"):
		n := strings.TrimSpace(e[1:])
		if i := strings.Index(n, "{"); i > 0 {
			n = n[:i]
		}
		return "a pointer to a new " + code(n)
	}
	if m := reCall.FindStringSubmatch(e); m != nil {
		name := m[1]
		if en, _ := lookupAPIName(x, name); en != nil {
			return "the result of " + code(shownCall(shown)) + " (" + lowerFirst(strings.TrimSuffix(en.What, ".")) + ")"
		}
		if ln, ok := x.funcs[lastSeg(name)]; ok {
			return fmt.Sprintf("the result of %s (the function on line %d)", code(shownCall(shown)), ln)
		}
		return "the result of " + code(shownCall(shown))
	}
	if strings.HasPrefix(e, "[") || strings.HasPrefix(e, "{") {
		return "a list of values " + code(shown)
	}
	if strings.ContainsAny(e, "+-*/%") {
		return code(shown) + " (worked out from a sum)"
	}
	return code(shown)
}

func shownCall(s string) string { return clip(s, 60) }

func lastSeg(name string) string {
	if i := strings.LastIndexAny(name, ".:>"); i >= 0 {
		return name[i+1:]
	}
	return name
}

func callPart(x *ctx, name, args string) Part {
	last := lastSeg(name)
	args = strings.TrimSpace(args)
	argText := ""
	if args != "" {
		argText = " with " + code(unhide(args, x.strs))
	}
	if e, key := lookupAPIName(x, name); e != nil {
		x.tag(e.Tags...)
		apiNotes(x, args, key)
		w := e.What
		if args != "" && len(args) < 90 {
			w = strings.TrimSuffix(w, ".") + " — here with " + code(unhide(args, x.strs)) + "."
		}
		return Part{Kind: kindOr(e.Kind, "call"), What: w, Why: e.Why}
	}
	apiNotes(x, args, "")
	if ln, ok := x.funcs[last]; ok {
		x.tag("function-call")
		why := "Calling a function runs the steps written inside it, then comes back here."
		if v := verbPhrase(last); v != "" {
			why = "From its name, it " + v + ". " + why
		}
		return Part{Kind: "call", What: fmt.Sprintf("Runs %s, the function written on line %d%s.", code(last+"()"), ln, argText), Why: why}
	}
	x.tag("function-call")
	if v := verbPhrase(last); v != "" {
		obj := ""
		if strings.ContainsAny(name, ".:>") {
			obj = " on " + code(strings.TrimSuffix(name[:len(name)-len(last)], "."))
			obj = strings.TrimSuffix(strings.TrimSuffix(obj, ":`")+"`", "->`")
			obj = strings.ReplaceAll(obj, "``", "`")
		}
		return Part{Kind: "call", What: "Calls " + code(last+"()") + obj + argText + ".", Why: "From its name, it probably " + v + ". It isn't defined in this code, so it comes from another file or a library."}
	}
	return Part{Kind: "call", What: "Calls " + code(name+"()") + argText + ".", Why: "A function or method that isn't defined in this code; it probably comes from another file or a library."}
}

// apiInLine explains a line by the first known name it uses.
func apiInLine(x *ctx, t string) (Part, bool) {
	for _, m := range reTokCall.FindAllStringSubmatch(t, -1) {
		if e, _ := lookupAPIName(x, m[1]); e != nil {
			x.tag(e.Tags...)
			return Part{Kind: kindOr(e.Kind, "call"), What: "Uses " + code(m[1]+"()") + ": " + lowerFirst(e.What), Why: e.Why}, true
		}
	}
	return Part{}, false
}

// apiNotes adds short notes for known functions used inside a line.
func apiNotes(x *ctx, t, skip string) {
	if x.first {
		return
	}
	seen := map[string]bool{skip: true}
	add := func(name string) {
		e, key := lookupAPIName(x, name)
		if e == nil || seen[key] || len(x.notes) >= 4 {
			return
		}
		if x.skipPat != "" && strings.Contains(x.skipPat, strings.TrimPrefix(lastSeg(key), ".")) {
			return
		}
		seen[key] = true
		x.tag(e.Tags...)
		x.notes = append(x.notes, code(key)+": "+lowerFirst(e.What))
	}
	for _, m := range reTokCall.FindAllStringSubmatch(t, -1) {
		add(m[1])
	}
	for _, m := range reTokDot.FindAllString(t, -1) {
		if e, _ := lookupAPIName(x, m); e != nil && e.Prop {
			add(m)
		}
	}
	if x.lang.Family == "basic" || x.lang.ID == "lua" || x.lang.ID == "gml" {
		for _, w := range reWord.FindAllString(t, -1) {
			if e, _ := lookupAPIName(x, w); e != nil && e.Prop {
				add(w)
			}
		}
	}
}

var reWord = regexp.MustCompile(`[A-Za-z_][\w]*\$?`)

// ------------------------------------------------------------- names

func splitWords(name string) []string {
	var words []string
	var cur []rune
	rs := []rune(name)
	for i, r := range rs {
		if r == '_' || r == '-' || r == '$' || r == '.' {
			if len(cur) > 0 {
				words = append(words, strings.ToLower(string(cur)))
				cur = nil
			}
			continue
		}
		if unicode.IsUpper(r) && len(cur) > 0 && (unicode.IsLower(cur[len(cur)-1]) || (i+1 < len(rs) && unicode.IsLower(rs[i+1]))) {
			words = append(words, strings.ToLower(string(cur)))
			cur = nil
		}
		cur = append(cur, r)
	}
	if len(cur) > 0 {
		words = append(words, strings.ToLower(string(cur)))
	}
	return words
}

// verbPhrase turns updatePlayerPosition into "updates the player position".
func verbPhrase(name string) string {
	w := splitWords(name)
	if len(w) == 0 {
		return ""
	}
	if w[0] == "on" && len(w) > 1 {
		return "runs when " + strings.Join(w[1:], " ") + " happens (an event handler)"
	}
	if w[0] == "handle" && len(w) > 1 {
		return "deals with " + strings.Join(w[1:], " ")
	}
	if w[0] == "is" || w[0] == "has" || w[0] == "can" || w[0] == "should" {
		if len(w) == 1 {
			return ""
		}
		return "answers yes or no: " + strings.Join(w, " ") + "?"
	}
	v, ok := verbs[w[0]]
	if !ok {
		return ""
	}
	if len(w) == 1 {
		return v
	}
	return v + " the " + strings.Join(w[1:], " ")
}

var verbs = map[string]string{
	"get": "gets", "set": "sets", "add": "adds", "remove": "removes", "delete": "deletes", "update": "updates",
	"draw": "draws", "render": "draws", "paint": "paints", "load": "loads", "save": "saves", "init": "sets up",
	"initialize": "sets up", "initialise": "sets up", "setup": "sets up", "create": "creates", "make": "makes",
	"build": "builds", "spawn": "creates (spawns)", "destroy": "removes", "kill": "removes", "check": "checks",
	"test": "tests", "play": "plays", "stop": "stops", "start": "starts", "begin": "starts", "end": "ends",
	"finish": "finishes", "move": "moves", "print": "prints", "show": "shows", "hide": "hides", "read": "reads",
	"write": "writes", "open": "opens", "close": "closes", "reset": "resets", "clear": "clears", "calc": "works out",
	"calculate": "works out", "compute": "works out", "find": "finds", "search": "searches for", "sort": "sorts",
	"parse": "reads and splits up", "send": "sends", "receive": "receives", "connect": "connects", "disconnect": "disconnects",
	"apply": "applies", "fire": "fires", "shoot": "shoots", "jump": "makes something jump", "attack": "attacks",
	"damage": "damages", "hurt": "hurts", "heal": "heals", "collide": "handles a collision with", "die": "handles the death of",
	"respawn": "brings back", "pause": "pauses", "resume": "resumes", "toggle": "switches on/off", "enable": "turns on",
	"disable": "turns off", "register": "registers", "emit": "sends out", "dispatch": "sends out", "handle": "deals with",
	"process": "processes", "run": "runs", "execute": "runs", "do": "does", "tick": "advances by one step",
	"step": "advances by one step", "animate": "animates", "rotate": "turns", "scale": "resizes", "fill": "fills",
	"copy": "copies", "convert": "converts", "format": "formats", "to": "converts to", "validate": "checks",
	"log": "writes a log message about", "flash": "flashes", "blink": "blinks", "beep": "beeps", "wait": "waits for",
	"sleep": "pauses for", "listen": "listens for", "fetch": "fetches", "download": "downloads", "upload": "uploads",
	"count": "counts", "sum": "adds up", "insert": "inserts", "push": "adds", "pop": "takes off", "append": "adds",
	"generate": "generates", "pick": "picks", "choose": "chooses", "select": "selects", "equip": "equips",
	"collect": "collects", "pickup": "picks up", "score": "scores", "level": "levels", "next": "moves to the next",
	"go": "goes to", "enter": "enters", "exit": "leaves", "quit": "quits", "restart": "restarts", "lose": "handles losing",
	"win": "handles winning", "game": "handles the game", "main": "is the main routine of",
}

// nameMeaning guesses what a variable holds from its name.
func nameMeaning(name string) string {
	w := splitWords(name)
	if len(w) == 0 {
		return ""
	}
	full := strings.Join(w, "_")
	if m, ok := names[full]; ok {
		return m
	}
	if m, ok := names[w[len(w)-1]]; ok {
		return m
	}
	if m, ok := names[w[0]]; ok {
		return m
	}
	return ""
}

var names = map[string]string{
	"score": "keeps the player's points", "points": "keeps the player's points", "hiscore": "keeps the best score so far", "high_score": "keeps the best score so far", "highscore": "keeps the best score so far",
	"lives": "counts how many lives are left", "health": "holds how much health is left", "hp": "holds hit points (health)", "max_health": "is the most health possible",
	"ammo": "counts the ammunition left", "coins": "counts collected coins", "gold": "counts the gold collected", "level": "holds which level is being played",
	"speed": "is how fast something moves", "velocity": "is the speed and direction of movement", "vel": "is the speed and direction of movement",
	"vx": "is the sideways speed", "vy": "is the up/down speed", "dx": "is how far to move sideways each step", "dy": "is how far to move up/down each step",
	"hsp": "is the horizontal speed", "vsp": "is the vertical speed", "x": "is a position across the screen (left–right)", "y": "is a position on the screen (up–down)",
	"z": "is a depth or height position", "pos": "is a position", "position": "is a position", "dir": "is a direction", "direction": "is a direction",
	"gravity": "is how strongly things are pulled down", "jump": "is to do with jumping", "jump_force": "is how strong a jump is", "jump_speed": "is how fast a jump starts",
	"timer": "counts time", "time": "holds a time", "cooldown": "is the waiting time before something can happen again", "delay": "is a waiting time",
	"count": "counts something", "counter": "counts something", "i": "is a loop counter",
	"n": "is a number or a count", "total": "holds a running total", "sum": "adds values up", "index": "is a position in a list", "idx": "is a position in a list",
	"len": "is a length", "length": "is a length", "size": "is a size", "width": "is a width", "height": "is a height",
	"angle": "is an angle", "rotation": "is how far something is turned", "rot": "is how far something is turned", "color": "is a colour", "colour": "is a colour",
	"name": "holds a name", "player": "is the player's character", "enemy": "is an enemy", "enemies": "is the list of enemies", "bullet": "is a bullet",
	"bullets": "is the list of bullets", "state": "remembers what the game or object is doing right now", "game_state": "remembers which part of the game is running (menu, playing, game over…)",
	"frame": "is an animation frame or frame count", "fps": "is frames per second", "dt": "is delta time: the seconds since the last frame", "delta": "is delta time: the time since the last frame",
	"running": "says whether the program should keep going", "done": "says whether something has finished", "alive": "says whether something is still alive",
	"grounded": "says whether the character is standing on the ground", "on_ground": "says whether the character is standing on the ground", "is_on_floor": "says whether the character is on the floor",
	"input": "holds what the player pressed or typed", "key": "holds a key that was pressed", "result": "holds the answer of a calculation", "res": "holds a result",
	"err": "holds an error, or nothing if all went well", "error": "holds an error", "data": "holds data", "buf": "is a buffer: a block of memory for data", "buffer": "is a buffer: a block of memory for data",
	"file": "is a file", "path": "is a file or folder location", "url": "is a web address", "msg": "is a message", "message": "is a message",
	"text": "holds some text", "str": "holds text", "items": "is a list of items", "list": "is a list", "arr": "is a list (array)", "array": "is a list (array)",
	"map": "is a lookup table or a level map", "grid": "is a grid of cells", "board": "is a game board", "tiles": "are the tiles of a map", "config": "holds settings",
	"cfg": "holds settings", "settings": "holds settings", "options": "holds options", "ctx": "is a context: the drawing surface or shared state", "canvas": "is the drawing area",
	"screen": "is the screen or window", "window": "is the window", "image": "is a picture", "img": "is a picture", "sprite": "is a sprite: a picture that moves around",
	"sound": "is a sound", "music": "is music", "rect": "is a rectangle (position and size)", "target": "is what is being aimed at", "handle": "is a handle: Windows' ID for something it opened",
	"hwnd": "is a window handle (Windows)", "dll": "is a loaded DLL (a library of Windows functions)", "proc": "is a function inside a DLL", "ptr": "is a pointer: a memory address",
	"user": "is a user", "users": "is a list of users", "client": "is a connection to a server", "server": "is a server", "conn": "is a connection", "port": "is a port number or serial port",
	"pin": "is a pin number on a board", "led": "is an LED", "sensor": "is a sensor", "value": "holds a value", "val": "holds a value", "temp": "is a temporary value or a temperature",
	"random": "holds a random number", "rnd": "holds a random number", "seed": "starts the random number sequence", "max": "is the biggest allowed value", "min": "is the smallest allowed value",
	"start": "is a starting point", "end": "is an end point", "now": "is the current time", "last": "is the previous value", "prev": "is the previous value", "next": "is the next value",
	"age": "holds an age", "price": "holds a price", "cost": "holds a cost", "id": "is an identifying number", "key_pressed": "says whether a key is pressed", "paused": "says whether the game is paused",
	"wave": "is the current wave of enemies", "spawn_timer": "counts time until the next thing appears", "inventory": "holds the player's items", "camera": "is the camera (the view of the world)",
	"world": "is the game world", "scene": "is the current scene", "node": "is a node (a thing in the scene)", "body": "is a physics body or the page body", "collider": "is a collision shape",
	"anim": "is an animation", "animation": "is an animation", "tile": "is a map tile", "room": "is the current room", "keys": "holds which keys are pressed", "mouse": "is the mouse",
}
