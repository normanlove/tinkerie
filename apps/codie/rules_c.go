// Codie — rules_c.go: C-style languages (C, C++, C#, Java, JavaScript,
// TypeScript, Go including Windows Go, Rust, Kotlin, Swift, PHP, GameMaker).
// Part of the free nrmluvdev maker suite. Created by nrmluvdev.
package codie

import (
	"fmt"
	"regexp"
	"strings"
)

func (x *ctx) top() string {
	if len(x.stack) == 0 {
		return ""
	}
	return x.stack[len(x.stack)-1].Kind
}

func (x *ctx) opensBrace() bool { return strings.HasSuffix(x.cur, "{") }

var cKeywords = map[string]bool{"if": true, "for": true, "while": true, "switch": true, "catch": true, "return": true, "else": true, "new": true, "delete": true, "throw": true, "case": true, "goto": true, "using": true, "typedef": true, "sizeof": true, "do": true, "foreach": true, "lock": true, "yield": true, "await": true, "function": true, "typeof": true, "defer": true, "go": true, "when": true, "match": true, "loop": true, "repeat": true, "with": true, "echo": true, "print": true, "fixed": true, "elif": true, "import": true, "package": true, "in": true, "of": true, "is": true, "as": true, "not": true, "and": true, "or": true, "unsafe": true, "static_assert": true, "co_return": true, "co_await": true}

var headers = map[string]string{
	"stdio.h":                   "standard input and output: «printf», «scanf», files",
	"stdlib.h":                  "general tools: memory («malloc», «free»), random numbers, «exit»",
	"string.h":                  "working with text and memory blocks: «strlen», «strcpy», «memcpy»",
	"math.h":                    "maths: «sqrt», «sin», «cos», «pow»",
	"stdint.h":                  "exact-size whole numbers such as «uint8_t» and «int32_t»",
	"stdbool.h":                 "«bool», «true» and «false» in C",
	"time.h":                    "clocks and dates: «time», «clock»",
	"ctype.h":                   "checking characters: «isdigit», «toupper»",
	"windows.h":                 "the whole Windows API: windows, messages, files, DLLs",
	"iostream":                  "C++ input and output streams: «std::cout» and «std::cin»",
	"vector":                    "«std::vector», a list that grows as you add to it",
	"string":                    "«std::string», C++ text",
	"map":                       "«std::map», a dictionary of keys and values",
	"memory":                    "smart pointers that free memory for you («unique_ptr», «shared_ptr»)",
	"algorithm":                 "ready-made algorithms: «sort», «find», «min», «max»",
	"cmath":                     "maths functions for C++",
	"SDL.h":                     "SDL, a library for windows, graphics, sound and input in games",
	"SDL2/SDL.h":                "SDL, a library for windows, graphics, sound and input in games",
	"raylib.h":                  "raylib, a simple library for making games",
	"Arduino.h":                 "the Arduino core: «pinMode», «digitalWrite», «delay»",
	"CoreMinimal.h":             "Unreal Engine's basic types",
	"GameFramework/Actor.h":     "Unreal's «AActor», the base of everything placed in a level",
	"GameFramework/Character.h": "Unreal's «ACharacter», a walking player or enemy",
}

var goPackages = map[string]string{
	"fmt":                                   "printing and formatting text",
	"os":                                    "files, folders, environment variables and exiting the program",
	"io":                                    "reading and writing streams of bytes",
	"bufio":                                 "buffered reading, for example line by line",
	"strings":                               "working with text",
	"strconv":                               "turning text into numbers and back",
	"time":                                  "clocks, timers and waiting",
	"math":                                  "maths functions",
	"math/rand":                             "random numbers",
	"sync":                                  "safely sharing data between goroutines",
	"errors":                                "making and checking errors",
	"log":                                   "writing log messages",
	"net":                                   "network connections",
	"net/http":                              "web servers and web requests",
	"encoding/json":                         "reading and writing JSON data",
	"path/filepath":                         "building file paths that work on every system",
	"os/exec":                               "starting other programs",
	"context":                               "cancelling work and setting time limits",
	"embed":                                 "packing files into the program (with //go:embed)",
	"flag":                                  "reading command-line options",
	"syscall":                               "low-level calls into the operating system; on Windows, loading DLLs and calling Windows functions",
	"unsafe":                                "raw memory pointers, needed to pass data to Windows functions",
	"golang.org/x/sys/windows":              "Go's official, extended Windows API package: handles, files, services, the registry and safer DLL calls",
	"golang.org/x/sys/windows/registry":     "reading and writing the Windows registry",
	"golang.org/x/sys/windows/svc":          "running a program as a Windows service",
	"C":                                     "cgo: lets Go call C code written in the comment just above",
	"github.com/hajimehoshi/ebiten/v2":      "Ebitengine, a 2D game library for Go",
	"github.com/gen2brain/raylib-go/raylib": "raylib for Go, a simple game library",
	"image/color":                           "colours",
	"runtime":                               "facts about the running program, such as runtime.GOOS (the operating system)",
}

var (
	reBase   = regexp.MustCompile(`(?::|\bextends|\bimplements)\s*(?:public\s+|private\s+|protected\s+|virtual\s+)?([\w.:<>]+)`)
	reGoRecv = regexp.MustCompile(`^\((\w+)\s+(\*?)(\w+)\)`)
)

// special method names in engines and frameworks
var lifecycle = map[string][2]string{
	// Unity
	"Awake":              {"Unity runs «Awake» once, as soon as the object is created, before «Start».", "Use it to find components and set up references other scripts need early."},
	"Start":              {"Unity runs «Start» once, just before the object's first frame.", "Set up starting values here: health, speed, references to other objects."},
	"Update":             {"Unity runs «Update» once every frame (often 60 or more times a second).", "This is the heart of the game loop for this object: read input, move things, check timers. Multiply movement by «Time.deltaTime» so speed doesn't depend on the frame rate."},
	"FixedUpdate":        {"Unity runs «FixedUpdate» at a steady rate (50 times a second by default), in step with the physics engine.", "Put physics here (forces on a Rigidbody), so it behaves the same on fast and slow computers."},
	"LateUpdate":         {"Unity runs «LateUpdate» every frame, after all the «Update» functions.", "Good for cameras that follow the player, because the player has already moved this frame."},
	"OnEnable":           {"Unity runs this whenever the object or script is switched on.", "Used to subscribe to events or reset state."},
	"OnDisable":          {"Unity runs this whenever the object or script is switched off.", "Used to unsubscribe from events and tidy up."},
	"OnDestroy":          {"Unity runs this just before the object is removed.", "Tidy up: save, stop sounds, tell other objects."},
	"OnCollisionEnter":   {"Unity runs this when this object's collider hits another solid collider.", "React to impacts: take damage, play a sound, bounce."},
	"OnCollisionEnter2D": {"Unity runs this when this 2D object bumps into another solid 2D collider.", "React to impacts: landing, hitting a wall or enemy."},
	"OnTriggerEnter":     {"Unity runs this when another collider enters this object's trigger zone (a collider marked Is Trigger).", "Triggers are invisible zones: pick-ups, checkpoints, damage areas, doors."},
	"OnTriggerEnter2D":   {"Unity runs this when another 2D collider enters this object's trigger zone.", "Used for coins, power-ups, checkpoints and hazards."},
	"OnTriggerExit":      {"Unity runs this when a collider leaves this trigger zone.", "Undo what entering did: close the door, stop the healing."},
	"OnTriggerExit2D":    {"Unity runs this when a 2D collider leaves this trigger zone.", "Undo what entering did."},
	"OnTriggerStay":      {"Unity runs this every physics step while a collider stays inside the trigger.", "For continuous effects such as damage over time."},
	"OnMouseDown":        {"Unity runs this when the player clicks on this object.", "Simple clickable objects without a UI system."},
	"OnGUI":              {"Unity's old immediate-mode GUI, run several times a frame.", "Quick debug buttons and text; real menus use the UI system."},
	// Unreal
	"BeginPlay":                 {"Unreal runs «BeginPlay» once when the actor enters the game.", "Set starting values and timers here."},
	"Tick":                      {"Unreal runs «Tick» every frame and passes «DeltaTime», the seconds since the last frame.", "Unreal's per-frame update: move things and check conditions, scaled by DeltaTime."},
	"SetupPlayerInputComponent": {"Unreal runs this to connect keys and buttons to this character's functions.", "Input bindings live here: «BindAxis» for movement, «BindAction» for jump or fire."},
	"EndPlay":                   {"Unreal runs this when the actor leaves the game.", "Tidy up timers and references."},
	// MonoGame / XNA
	"Initialize":  {"MonoGame runs «Initialize» once at start-up, before content loads.", "Set up non-graphics things such as settings and starting values."},
	"LoadContent": {"MonoGame runs «LoadContent» once to load pictures, sounds and fonts.", "Load everything the game draws here, not every frame, so it stays fast."},
	"Draw":        {"The framework runs «Draw» every frame to paint the screen.", "Only draw here; changing game state belongs in Update."},
	// Godot C#
	"_Ready":          {"Godot runs «_Ready» once, when the node and its children have entered the scene.", "Grab child nodes and set starting values here."},
	"_Process":        {"Godot runs «_Process» every frame, passing «delta» (seconds since the last frame).", "Per-frame logic: input, animation, timers. Multiply movement by delta."},
	"_PhysicsProcess": {"Godot runs «_PhysicsProcess» at a fixed rate (60 per second by default).", "Movement with physics and «MoveAndSlide» goes here."},
	// Arduino / C
	"setup": {"Arduino runs «setup» once when the board powers up or resets.", "Set pin modes, start Serial and set starting values."},
	"loop":  {"Arduino runs «loop» over and over, forever, after setup.", "Your program's main work goes here; it's a microcontroller's version of a game loop."},
	"main":  {"«main» is where the program starts running.", "Every C, C++, Go, Rust, Java and C# program begins at its main function."},
	// Ebitengine (Go)
	"Layout": {"Ebitengine calls «Layout» to ask how big the game's screen is.", "Returning a fixed size gives crisp pixel art that scales to the window."},
	// Swift SpriteKit
	"didMove":      {"SpriteKit runs «didMove(to:)» when the scene appears.", "Build the scene here: add sprites, physics and labels."},
	"touchesBegan": {"Runs when the player's finger touches the screen.", "Read touches for jumps, taps and controls."},
	// Java (Android / LibGDX)
	"create":   {"LibGDX runs «create» once when the game starts.", "Load textures and set up the camera and world."},
	"render":   {"LibGDX runs «render» every frame: it both updates and draws.", "Use Gdx.graphics.getDeltaTime() to keep movement smooth."},
	"onCreate": {"Android runs «onCreate» when the screen (activity) is first made.", "Set the layout and connect buttons here."},
}

func funcExplain(x *ctx, name, params, ret, recv string) (string, string) {
	what := "Defines a function called " + code(name) + "."
	if recv != "" {
		what = "Defines the method " + code(name) + " for the type " + code(recv) + "."
	}
	why := takes(x.lang, params)
	if ret != "" && ret != "void" && ret != "Unit" {
		why += " It gives back " + code(unhide(ret, x.strs)) + "."
	}
	if lc, ok := lifecycle[name]; ok && (x.lang.ID != "go" || name == "main" || name == "Update" || name == "Draw" || name == "Layout") {
		what += " " + codeMarks.Replace(lc[0])
		why = codeMarks.Replace(lc[1]) + " " + why
		if name != "main" {
			x.tag("callback")
		}
		switch name {
		case "Update", "FixedUpdate", "LateUpdate", "Tick", "_Process", "_PhysicsProcess", "loop", "render", "Draw":
			x.tag("game-loop")
		}
	} else if v := verbPhrase(name); v != "" {
		why = "From its name, it " + v + ". " + why
	}
	if !x.opensBrace() && strings.HasSuffix(x.cur, ";") && x.lang.ID != "go" {
		what = strings.Replace(what, "Defines", "Declares", 1) + " (Only its shape; the body is written elsewhere.)"
	}
	x.tag("function")
	return what, why
}

func classExplain(x *ctx, kind, name, rest string) (string, string) {
	k := kind
	what := ""
	why := ""
	switch kind {
	case "class":
		what = "Defines a class called " + code(name) + "."
		why = "A class is a blueprint: it bundles data (fields) and actions (methods). You can make many objects from one class."
	case "struct":
		what = "Defines a structure called " + code(name) + "."
		why = "A struct groups related values under one name, such as a player's x, y and health."
	case "interface", "trait", "protocol":
		what = "Defines the " + kind + " " + code(name) + ": a list of methods a type must provide."
		why = "It lets code work with anything that has those methods, without caring which exact type it is."
		k = "interface"
	case "enum":
		what = "Defines " + code(name) + ", a list of named choices (an enum)."
		why = "Named choices such as Idle, Running, Jumping are clearer and safer than plain numbers."
	case "record", "data":
		what = "Defines the record " + code(name) + ", a simple bundle of values."
		why = "Records are classes made only to hold data; the language writes the boring parts for you."
	case "object":
		what = "Defines " + code(name) + ", a single ready-made object."
		why = "Kotlin «object» makes exactly one instance, often for shared settings or helpers."
	case "union":
		what = "Defines the union " + code(name) + ": several types sharing the same memory."
		why = "Saves memory when only one of the values is used at a time."
	default:
		what = "Defines " + code(name) + "."
	}
	if m := reBase.FindStringSubmatch(rest); m != nil {
		base := m[1]
		what += " It builds on " + code(base) + "."
		switch {
		case base == "MonoBehaviour":
			why = "Inheriting from MonoBehaviour makes this a Unity script you can attach to a GameObject; Unity then calls its Start and Update for you."
			x.tag("game-engine")
		case strings.HasPrefix(base, "A") && (base == "AActor" || base == "ACharacter" || base == "APawn" || base == "AGameModeBase"):
			why = "In Unreal, classes starting with A are Actors: things that can be placed in a level. Unreal calls BeginPlay and Tick on them."
			x.tag("game-engine")
		case base == "Game" || base == "Microsoft.Xna.Framework.Game":
			why = "Inheriting from Game makes this the main MonoGame class, with Initialize, LoadContent, Update and Draw."
			x.tag("game-engine")
		case strings.HasSuffix(base, "Node2D") || strings.HasSuffix(base, "Body2D") || strings.HasSuffix(base, "Body3D") || base == "Node" || base == "Node3D" || base == "Area2D":
			why = "This is a Godot node script in C#: Godot calls _Ready and _Process on it."
			x.tag("game-engine")
		case base == "SKScene":
			why = "An SKScene is one screen of a SpriteKit game; SpriteKit calls its update every frame."
			x.tag("game-engine")
		default:
			why += " It inherits (gets a copy of) everything " + code(base) + " has, and can add or change parts."
		}
		x.tag("inheritance")
	}
	x.tag("class")
	_ = k
	return what, why
}

func init() {
	c := "@c"
	// ------------------------------------------------ preprocessor & files
	R("c,cpp", `^#\s*include\s*(?:<([^>]+)>|{S})`, "import", "", "").with(func(m []string, x *ctx) (string, string) {
		if m[1] == "" && len(x.strs) > 0 {
			m[1] = x.strs[0]
			why := "Quotes mean a header from this project (not a system library). It holds the declarations this file needs."
			if d, ok := headers[m[1]]; ok {
				return "Brings in " + code(m[1]) + ": " + codeMarks.Replace(d) + ".", why
			}
			return "Brings in the project's own header file " + code(m[1]) + ".", why
		}
		why := "C and C++ only know about functions that were declared earlier; header files hold those declarations."
		if d, ok := headers[m[1]]; ok {
			return "Brings in " + code(m[1]) + ": " + codeMarks.Replace(d) + ".", why
		}
		return "Brings in the header file " + code(m[1]) + ".", why
	}).Tags = []string{"import"}
	R("c,cpp", `^#\s*define\s+({I})(\([^)]*\))?\s*(.*)$`, "directive", "", "").with(func(m []string, x *ctx) (string, string) {
		if m[2] != "" {
			return "Defines the macro " + code(m[1]+m[2]) + ", which is replaced by " + code(m[3]) + " before compiling.", "Macros are text replacements done before the real compile; they work like tiny functions with no type checks."
		}
		if strings.TrimSpace(m[3]) == "" {
			return "Defines the name " + code(m[1]) + " (with no value) for the preprocessor.", "Usually tested later with #ifdef to switch parts of the code on or off."
		}
		return "Defines " + code(m[1]) + " to mean " + code(m[3]) + " everywhere below.", "A named constant: change it once here instead of in many places."
	})
	R("c,cpp,csharp", `^#\s*(ifdef|ifndef|if|elif)\b\s*(.*)$`, "directive", "Only compiles the following lines if the condition $2 holds (preprocessor $1).", "Lets one source file build differently, for example for Windows and Linux or debug and release.")
	R("c,cpp,csharp", `^#\s*else$`, "directive", "Otherwise, compiles the following lines instead.", "The other half of an #if.")
	R("c,cpp,csharp", `^#\s*endif$`, "directive", "Ends the #if section.", "")
	R("c,cpp", `^#\s*pragma\s+once$`, "directive", "Makes sure this header is only read once, even if included many times.", "Prevents errors from the same thing being declared twice.")
	R("c,cpp,csharp", `^#\s*pragma\s+(.*)$`, "directive", "A special instruction to the compiler: $1.", "Pragmas change how the compiler treats this file.")
	R("csharp", `^#\s*region\b\s*(.*)$`, "directive", "Starts a foldable section called $1.", "Regions just let the editor fold code away; they change nothing when it runs.")
	R("csharp", `^#\s*endregion\b`, "directive", "Ends the foldable section.", "")

	R("go", `^package\s+(\w+)$`, "import", "", "").with(func(m []string, x *ctx) (string, string) {
		if m[1] == "main" {
			return "Says this file belongs to package «main».", "In Go, package main is a program you can run; it must have a func main. Any other name makes a library."
		}
		return "Says this file belongs to the package " + code(m[1]) + ".", "Every Go file starts with its package. Files in the same folder share one package and can use each other's names."
	})
	R("java,kotlin", `^package\s+([\w.]+)$`, "import", "Puts this file in the package $1.", "Packages group related classes, like folders, and stop names clashing.")
	goImp := func(m []string, x *ctx) (string, string) {
		path := ""
		for _, s := range x.strs {
			path = s
		}
		alias := strings.TrimSpace(m[1])
		d, ok := goPackages[path]
		what := "Imports the package " + code(path)
		if ok {
			what += ": " + d
		}
		what += "."
		why := "Importing a package lets this file use its functions, by writing the package's name and a dot."
		switch {
		case alias == "_":
			why = "The underscore imports the package only for its side effects (it registers itself), without using its name."
		case alias == ".":
			why = "The dot lets the package's names be used without the package prefix."
		case alias != "":
			why = "The package is used under the short name " + code(alias) + "."
		}
		switch path {
		case "syscall", "unsafe", "golang.org/x/sys/windows", "golang.org/x/sys/windows/registry", "golang.org/x/sys/windows/svc":
			x.tag("windows-go")
		case "C":
			x.tag("cgo")
			why = "This is cgo: the comment right above the import holds C code that Go can then call as C.something. It needs a C compiler (and on Windows, usually MinGW)."
		}
		x.tag("import")
		return what, why
	}
	R("go", `^import\s+(\w+\s+|_\s+|\.\s+)?{S}$`, "import", "", "").with(goImp)
	R("go", `^(\w+\s+|_\s+|\.\s+)?{S}$`, "import", "", "").with(func(m []string, x *ctx) (string, string) {
		if x.top() != "import" && x.pending != "import" {
			return "", ""
		}
		return goImp(m, x)
	})
	R("go", `^(import|const|var|type)\s*\($`, "import", "", "").with(func(m []string, x *ctx) (string, string) {
		x.stack = append(x.stack, &block{Kind: m[1], Start: x.n, Depth: len(x.stack), outlineIdx: -1})
		switch m[1] {
		case "import":
			return "Starts a list of packages to import.", "Grouping imports in brackets is the usual Go style."
		case "const":
			return "Starts a group of constants.", "Constants are values fixed when the program is built. «iota» counts 0, 1, 2… down the group."
		case "var":
			return "Starts a group of package-level variables.", "These are shared by every function in the package."
		}
		return "Starts a group of type definitions.", ""
	})
	R("go", `^\)$`, "close", "", "").with(func(m []string, x *ctx) (string, string) {
		if t := x.top(); t != "import" && t != "const" && t != "var" && t != "type" {
			return "Closes the bracket opened above.", "A long statement was split over several lines."
		}
		if b := x.pop(x.n); b != nil {
			return "Ends the " + b.Kind + " group from line " + fmt.Sprint(b.Start) + ".", ""
		}
		return "Closes the bracket opened above.", "A long statement was split over several lines."
	})
	R("java,kotlin", `^import\s+(static\s+)?([\w.*]+)$`, "import", "Imports $2 so this file can use it by its short name.", "Without imports you'd have to write the full package name every time.", "import")
	R("csharp", `^using\s+(static\s+)?([\w.]+)$`, "import", "", "").with(func(m []string, x *ctx) (string, string) {
		ns := m[2]
		d := map[string]string{"System": "basic types, the console, maths", "UnityEngine": "Unity's game classes: GameObject, Transform, Vector3, Input, Time", "System.Collections": "collections and coroutines (IEnumerator)", "System.Collections.Generic": "List, Dictionary and other collections", "Godot": "Godot's node classes for C#", "Microsoft.Xna.Framework": "MonoGame's core game classes", "Microsoft.Xna.Framework.Input": "MonoGame keyboard, mouse and gamepad input", "System.Linq": "LINQ: querying lists with Where, Select, OrderBy", "UnityEngine.SceneManagement": "loading and switching Unity scenes", "UnityEngine.UI": "Unity's UI: buttons, text, images", "System.IO": "files and folders", "System.Windows.Forms": "classic Windows desktop forms", "System.Runtime.InteropServices": "calling Windows DLL functions with [DllImport]"}[ns]
		w := "Uses the namespace " + code(ns)
		if d != "" {
			w += ": " + d
		}
		return w + ".", "With «using» you can write short names like Vector3 instead of UnityEngine.Vector3."
	}).Tags = []string{"import"}
	R("javascript,typescript", `^import\s+(.+?)\s+from\s+({S})$`, "import", "Imports $1 from the module $2.", "Modules let code be split into files and shared; this brings in only the parts named.", "import")
	R("javascript,typescript", `^import\s+({S})$`, "import", "Loads the module $1 for its side effects.", "", "import")
	R("javascript,typescript", `^(?:const|let|var)\s+(.+?)\s*=\s*require\(\s*({S})\s*\)$`, "import", "Loads the module $2 (Node.js style) into $1.", "«require» is the older Node.js way of importing code from another file or package.", "import")
	R("javascript,typescript", `^export\s+default\s+(.+)$`, "import", "Makes $1 the main thing this file shares with other files.", "Other files can then import it by any name they like.")
	R("javascript,typescript", `^export\s+\{(.*)\}$`, "import", "Shares $1 with other files.", "Only exported names can be imported elsewhere.")
	R("javascript,typescript", `^module\.exports\s*=\s*(.+)$`, "import", "Shares $1 with other files (Node.js style).", "Whatever is put in module.exports is what require() gives back.")
	R("php", `^(?:require|include)(?:_once)?\s*\(?\s*({S})\s*\)?$`, "import", "Pulls in the PHP file $1 here.", "_once makes sure it is only loaded once.", "import")
	R("php", `^<\?php$`, "directive", "Starts PHP code.", "Everything outside <?php … ?> is sent to the browser as plain HTML.")
	R("php", `^\?>$`, "directive", "Ends PHP code; anything after this is plain HTML.", "")
	R("php", `^namespace\s+([\w\\]+)$`, "import", "Puts this file in the namespace $1.", "")
	R("php", `^use\s+([\w\\]+)(\s+as\s+\w+)?$`, "import", "Lets this file use $1 by its short name.", "", "import")
	R("rust", `^(?:pub\s+)?use\s+(.+)$`, "import", "Brings $1 into scope.", "After «use», the item can be named without its full path.", "import")
	R("rust", `^(?:pub\s+)?mod\s+(\w+)$`, "import", "Includes the module $1 (from the file $1.rs).", "Modules split a Rust program into files.")
	R("swift", `^import\s+(\w+)$`, "import", "Imports the $1 framework.", "SpriteKit is for 2D games, SwiftUI and UIKit for app screens, Foundation for basics.", "import")
	R("csharp,cpp", `^namespace\s+([\w.:]+)$`, "import", "Puts the code below into the namespace $1.", "Namespaces stop names clashing between libraries, like surnames for code.").open("namespace", 1)
	R("cpp", `^using\s+namespace\s+([\w:]+)$`, "import", "Lets the code use names from $1 without writing $1:: every time.", "Handy in small programs; big programs often avoid it to prevent clashes.")
	R("cpp", `^(?:std::)?cout\s*<<\s*(.+)$`, "print", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("output")
		return "Prints " + code(unhide(m[1], x.strs)) + " to the console.", "<< sends each value to the output in turn; std::endl ends the line."
	})
	R("cpp", `^(?:std::)?cin\s*>>\s*(.+)$`, "input", "Reads what the user types into $1.", ">> takes values from the keyboard input.", "input")
	R("cpp", `^template\s*<(.*)>$`, "directive", "Makes the next class or function a template that works with any type ($1).", "Templates let one piece of code work for ints, floats, your own types…", "generics")
	R("cpp", `^(public|private|protected)\s*:$`, "directive", "Members below this line are $1.", "public: anyone can use them; private: only the class itself; protected: the class and classes built on it.", "class")

	// ------------------------------------------------ attributes / annotations
	R("csharp", `^\[(\w+)(?:\((.*)\))?\]$`, "directive", "", "").with(func(m []string, x *ctx) (string, string) {
		switch m[1] {
		case "SerializeField":
			return "Shows the next private field in Unity's Inspector.", "You can then tune it in the editor while keeping it hidden from other scripts."
		case "Header":
			return "Adds the heading " + expand("$2", m, x) + " above the next fields in Unity's Inspector.", "Just makes the Inspector tidier."
		case "RequireComponent":
			return "Tells Unity this script needs the component " + expand("$2", m, x) + "; Unity adds it automatically.", "Prevents errors from a missing Rigidbody or Collider."
		case "Range":
			return "Shows the next number as a slider from " + expand("$2", m, x) + " in the Inspector.", ""
		case "DllImport":
			x.tag("windows-go")
			return "Says the next method lives in the Windows DLL " + expand("$2", m, x) + ".", "This is how C# calls Windows API functions (P/Invoke)."
		case "Export":
			return "Shows the next property in Godot's Inspector.", ""
		}
		return "Adds the attribute " + code(m[1]) + " to the next item.", "Attributes are labels that tools and frameworks read."
	})
	R("java,kotlin,typescript", `^@(\w+)(\(.*\))?$`, "directive", "", "").with(func(m []string, x *ctx) (string, string) {
		if m[1] == "Override" {
			return "Marks the next method as replacing one from the parent class (@Override).", "The compiler checks the parent really has it, catching spelling mistakes."
		}
		return "Adds the annotation " + code("@"+m[1]) + " to the next item.", "Annotations are labels read by the compiler or a framework."
	})
	R("rust", `^#!?\[(.+)\]$`, "directive", "", "").with(func(m []string, x *ctx) (string, string) {
		if strings.HasPrefix(m[1], "derive(") {
			return "Asks the compiler to write standard code for " + code(m[1][7:len(m[1])-1]) + " automatically.", "derive saves writing printing (Debug), copying (Clone) or comparing (PartialEq) by hand."
		}
		return "An attribute: " + code(m[1]) + ".", "Attributes change how the compiler treats the next item."
	})
	R("cpp", `^(UCLASS|UPROPERTY|UFUNCTION|USTRUCT|UENUM)\((.*)\)$`, "directive", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("game-engine")
		return "Unreal's " + code(m[1]) + " marker: it tells Unreal's reflection system about the next item.", "That's what lets the item show in the editor, be used in Blueprints and be saved."
	})
	R("cpp", `^GENERATED_BODY\(\)$`, "directive", "Unreal puts its generated code here.", "Every Unreal class needs it; leave it alone.")

	// ------------------------------------------------ types
	R("go", `^type\s+({I})\s+struct$`, "class", "", "").with(func(m []string, x *ctx) (string, string) { return classExplain(x, "struct", m[1], "") }).open("struct", 1)
	R("go", `^type\s+({I})\s+interface$`, "class", "", "").with(func(m []string, x *ctx) (string, string) { return classExplain(x, "interface", m[1], "") }).open("interface", 1)
	R("go", `^type\s+({I})\s+(.+)$`, "class", "Defines a new type $1 based on $2.", "A named type makes code clearer and lets you add methods to it.", "type")
	R("go", `^({I})\s+(.+)$`, "var", "", "").with(func(m []string, x *ctx) (string, string) {
		if x.top() == "type" {
			return "Defines the type " + code(m[1]) + " as " + code(unhide(m[2], x.strs)) + ".", ""
		}
		if x.top() != "struct" {
			return "", ""
		}
		f := strings.SplitN(m[2], "\x01", 2)
		w := "A field " + code(m[1]) + " of type " + code(strings.TrimSpace(f[0])) + "."
		if len(f) > 1 {
			w += " Its tag " + code(unhide("\x01"+f[1], x.strs)) + " tells packages such as encoding/json how to name it."
		}
		why := "Every value of this struct has its own " + code(m[1]) + "."
		if 'A' <= m[1][0] && m[1][0] <= 'Z' {
			why += " A capital first letter makes it visible to other packages."
		}
		return w, why
	})
	R("go", `^({I})\((.*)\)\s*(.*)$`, "def", "", "").with(func(m []string, x *ctx) (string, string) {
		if x.top() != "interface" {
			return "", ""
		}
		return "Says every type that fits this interface must have a method " + code(m[1]+"("+m[2]+")") + ".", takes(x.lang, m[2])
	})
	R("c,cpp", `^typedef\s+struct\s*({I})?$`, "class", "Defines a structure type.", "typedef lets the struct be used by a short name.").open("struct", 1)
	R("c,cpp", `^typedef\s+(.+?)\s+(\**{I})$`, "class", "Creates $2 as another name for $1.", "Shorter, clearer names for types.")
	R("c,cpp", `^}\s*({I})$`, "close", "Ends the structure and names it $1.", "")
	for _, k := range [][2]string{{"enum", "enum"}, {"interface|trait|protocol", "interface"}, {"struct|union", "struct"}} {
		k := k
		R(c, `^(?:(?:public|private|protected|internal|abstract|static|sealed|final|export|default|partial|open|data|pub(?:\(\w+\))?|inner|virtual|readonly|ref|unsafe|declare)\s+)*(`+k[0]+`)\s+(?:class\s+)?({I})(.*)$`, "class", "", "").with(func(m []string, x *ctx) (string, string) {
			return classExplain(x, m[1], m[2], m[3])
		}).open(k[1], 2)
	}
	R(c, `^(?:(?:public|private|protected|internal|abstract|static|sealed|final|export|default|partial|open|data|pub(?:\(\w+\))?|inner|virtual|readonly|ref|unsafe|declare)\s+)*(class|struct|interface|enum|trait|record|object|union|protocol)\s+(?:class\s+)?({I})(.*)$`, "class", "", "").with(func(m []string, x *ctx) (string, string) {
		return classExplain(x, m[1], m[2], m[3])
	}).open("class", 2)
	R("kotlin", `^data\s+class\s+({I})(.*)$`, "class", "", "").with(func(m []string, x *ctx) (string, string) { return classExplain(x, "data", m[1], m[2]) }).open("class", 1)
	R("rust", `^(?:pub\s+)?impl(?:<[^>]*>)?\s+(?:([\w:<>]+)\s+for\s+)?([\w<>]+)$`, "class", "", "").with(func(m []string, x *ctx) (string, string) {
		if m[1] != "" {
			return "Implements the trait " + code(m[1]) + " for " + code(m[2]) + ".", "This gives " + code(m[2]) + " the methods the trait promises."
		}
		return "Adds methods to " + code(m[2]) + ".", "In Rust, data (struct) and behaviour (impl) are written separately."
	}).open("class", 2)
	R("typescript", `^(?:export\s+)?type\s+({I})(?:<.*>)?\s*=\s*(.+)$`, "class", "Defines the type name $1 as $2.", "Type names describe the shape of data so the editor can catch mistakes.", "type")
	R("typescript", `^({I})(\??)\s*:\s*([^=]+?);?$`, "var", "", "").with(func(m []string, x *ctx) (string, string) {
		if x.top() != "interface" && x.top() != "class" {
			return "", ""
		}
		opt := ""
		if m[2] == "?" {
			opt = " It's optional (it may be missing)."
		}
		return "The field " + code(m[1]) + " holds a " + code(m[3]) + "." + opt, ""
	})
	R(c, `^({I})(?:\s*=\s*(.+?))?,?$`, "var", "", "").with(func(m []string, x *ctx) (string, string) {
		if x.top() != "enum" && !(x.top() == "class" && false) {
			return "", ""
		}
		if m[2] != "" {
			return "The choice " + code(m[1]) + ", which has the value " + code(m[2]) + ".", ""
		}
		return "One of the choices: " + code(m[1]) + ".", ""
	})
	R("javascript,typescript,gml,php", `^({I}|{S})\s*:\s*(.+?),?$`, "assign", "", "").with(func(m []string, x *ctx) (string, string) {
		if x.top() != "block" && x.top() != "object" {
			return "", ""
		}
		return "Sets the property " + expand("$1", m, x) + " to " + describeExpr(x, strings.TrimSuffix(m[2], ",")) + ".", "Objects hold named values (properties) together, like a labelled box of settings."
	})

	// ------------------------------------------------ functions
	R("go", `^func\s+(\([^)]*\)\s*)?({I})\s*(?:\[[^\]]*\])?\s*\((.*?)\)\s*(.*)$`, "def", "", "").with(func(m []string, x *ctx) (string, string) {
		recv := ""
		if r := reGoRecv.FindStringSubmatch(strings.TrimSpace(m[1])); r != nil {
			recv = r[3]
			if r[2] == "*" {
				recv = "*" + r[3]
			}
		}
		ret := strings.TrimSpace(m[4])
		w, y := funcExplain(x, m[2], m[3], ret, recv)
		if strings.Contains(ret, "error") {
			y += " If something goes wrong it returns an error, which the caller should check."
		}
		if m[2] == "init" {
			w = "Defines an «init» function, which Go runs automatically before «main»."
			w = codeMarks.Replace(w)
			y = "Used to set things up when the package loads."
		}
		return w, y
	}).open("function", 2)
	R("go", `^({I})\s*:?=\s*func\s*\((.*?)\)\s*(.*)$`, "def", "Stores a new function in $1.", "Functions are values in Go: they can be kept in variables and passed around.", "function").open("function", 1).setsVar(1)
	R("javascript,typescript,gml", `^(?:export\s+)?(?:default\s+)?(?:async\s+)?function\s*\*?\s*({I})?\s*(?:<[^>]*>)?\s*\((.*?)\)(?:\s*:\s*(.+))?$`, "def", "", "").with(func(m []string, x *ctx) (string, string) {
		if m[1] == "" {
			return "Starts a function with no name (an anonymous function).", takes(x.lang, m[2]) + " It's usually passed straight to something else to run later."
		}
		w, y := funcExplain(x, m[1], m[2], m[3], "")
		if strings.Contains(x.cur, "async") {
			y += " It's async: it can wait for slow things (downloads, files) without freezing the page."
			x.tag("async")
		}
		return w, y
	}).open("function", 1)
	R("javascript,typescript", `^(?:export\s+)?(?:const|let|var)\s+({I})\s*(?::\s*[^=]+)?=\s*(async\s+)?(?:function\s*\w*\s*\((.*?)\)|\(?([\w\s,={}\[\]:.?]*)\)?\s*=>)(.*)$`, "def", "", "").with(func(m []string, x *ctx) (string, string) {
		p := m[3]
		if p == "" {
			p = m[4]
		}
		w, y := funcExplain(x, m[1], p, "", "")
		w = strings.Replace(w, "Defines a function", "Defines a function (stored in a constant)", 1)
		if strings.Contains(m[5], "=>") || strings.Contains(x.cur, "=>") {
			y += " The => is arrow-function shorthand."
		}
		return w, y
	}).open("function", 1).setsVar(1)
	R("javascript,typescript", `^(?:(?:static|async|get|set|public|private|protected|readonly|override)\s+)*({I})\s*\((.*?)\)(?:\s*:\s*[\w<>\[\]|. ]+)?$`, "def", "", "").with(func(m []string, x *ctx) (string, string) {
		if cKeywords[m[1]] || !x.opensBrace() || x.top() != "class" {
			return "", ""
		}
		if m[1] == "constructor" {
			return "Defines the constructor: the code that runs when a new object of this class is made with «new».", takes(x.lang, m[2]) + " It usually stores starting values on «this»."
		}
		w, y := funcExplain(x, m[1], m[2], "", "")
		return strings.Replace(w, "a function", "a method", 1), y
	}).open("method", 1)
	R("rust", `^(?:pub(?:\([^)]*\))?\s+)?(?:const\s+)?(?:async\s+)?(?:unsafe\s+)?(?:extern\s+\S+\s+)?fn\s+({I})\s*(?:<.*?>)?\s*\((.*?)\)\s*(?:->\s*(.+?))?(?:\s+where\b.*)?$`, "def", "", "").with(func(m []string, x *ctx) (string, string) {
		return funcExplain(x, m[1], m[2], m[3], "")
	}).open("function", 1)
	R("kotlin", `^(?:(?:public|private|override|open|suspend|internal|protected|inline|operator|abstract)\s+)*fun\s+(?:<.*?>\s*)?(?:[\w.]+\.)?({I})\s*\((.*?)\)\s*(?::\s*([\w<>?, .]+))?(?:\s*=.*)?$`, "def", "", "").with(func(m []string, x *ctx) (string, string) {
		return funcExplain(x, m[1], m[2], m[3], "")
	}).open("function", 1)
	R("swift", `^(?:(?:public|private|override|static|mutating|@objc|fileprivate|internal|open|class|final)\s+)*func\s+({I})\s*(?:<.*?>)?\s*\((.*?)\)\s*(?:(?:throws|async)\s*)*(?:->\s*(.+?))?$`, "def", "", "").with(func(m []string, x *ctx) (string, string) {
		return funcExplain(x, m[1], m[2], m[3], "")
	}).open("function", 1)
	R("php", `^(?:(?:public|private|protected|static|abstract|final)\s+)*function\s+({I})\s*\((.*?)\)(?:\s*:\s*\??\w+)?$`, "def", "", "").with(func(m []string, x *ctx) (string, string) {
		if m[1] == "__construct" {
			return "Defines the constructor, run when a new object is made with «new».", takes(x.lang, m[2])
		}
		return funcExplain(x, m[1], m[2], "", "")
	}).open("function", 1)
	R("cpp", `^({I})::(~?{I})\s*\((.*?)\)\s*(?::\s*(.+))?$`, "def", "", "").with(func(m []string, x *ctx) (string, string) {
		if strings.HasSuffix(x.cur, ";") {
			return "", ""
		}
		if m[1] == m[2] {
			w := "Defines the constructor of " + code(m[1]) + ": the code run when a new " + m[1] + " is made."
			if m[4] != "" {
				w += " The part after the colon sets members first: " + code(m[4]) + "."
			}
			return w, takes(x.lang, m[3])
		}
		if strings.HasPrefix(m[2], "~") {
			return "Defines the destructor of " + code(m[1]) + ", run when an object is destroyed.", "Frees memory and other resources the object was holding."
		}
		return funcExplain(x, m[2], m[3], "", m[1])
	}).open("method", 2)
	R("c,cpp,csharp,java", `^((?:(?:public|private|protected|internal|static|virtual|override|abstract|final|async|extern|inline|unsigned|signed|synchronized|sealed|new|explicit|constexpr|friend|unsafe|partial|FORCEINLINE|UFUNCTION\(.*?\))\s+)*)((?:const\s+)?{T})\s+([*&]*)({I}|{I}::~?{I}|operator\s*\S+)\s*\((.*?)\)\s*(const|override|noexcept|final|throws\s+[\w, ]+)?\s*(?::.*)?$`, "def", "", "").with(func(m []string, x *ctx) (string, string) {
		typ := strings.TrimSpace(m[2])
		if cKeywords[typ] || cKeywords[m[4]] || typ == "return" || typ == "else" {
			return "", ""
		}
		name := m[4]
		recv := ""
		if i := strings.Index(name, "::"); i > 0 {
			recv, name = name[:i], name[i+2:]
		}
		return funcExplain(x, name, m[5], typ, recv)
	}).open("function", 4)
	R("csharp,java", `^(?:public|private|protected|internal)\s+({I})\s*\((.*?)\)(?:\s*:\s*(?:base|this)\(.*\))?$`, "def", "", "").with(func(m []string, x *ctx) (string, string) {
		return "Defines the constructor of " + code(m[1]) + ": the code that runs when a new " + m[1] + " is made with «new».", takes(x.lang, m[2]) + " It usually sets the starting values of the object's fields."
	}).open("method", 1)
	R("csharp", `^(?:(?:public|private|protected|internal|static)\s+)*delegate\s+(.+)$`, "def", "Declares a delegate type: a kind of variable that can hold a function.", "Delegates and events let one object tell others when something happens.")

	// ------------------------------------------------ declarations
	R("go", `^(?:var|const)\s+({I}(?:\s*,\s*{I})*)\s+([^=]+?)(?:\s*=\s*(.+))?$`, "var", "", "").with(func(m []string, x *ctx) (string, string) {
		w := "Creates the variable " + code(m[1]) + " of type " + code(m[2])
		if strings.HasPrefix(x.cur, "const") {
			w = "Creates the constant " + code(m[1]) + " of type " + code(m[2])
		}
		if m[3] != "" {
			w += ", set to " + describeExpr(x, m[3]) + "."
		} else {
			w += ". It starts at its zero value (0, \"\", false or nil)."
		}
		return w, varWhy(x, m[1], "")
	}).setsVar(1)
	R("go", `^(?:var|const)\s+({I})\s*=\s*(.+)$`, "var", "", "").with(func(m []string, x *ctx) (string, string) {
		p := assignPart(x, "var", m[1], "", m[2])
		if strings.HasPrefix(x.cur, "const") {
			return "Creates the constant " + code(m[1]) + " with the value " + describeExpr(x, m[2]) + ".", "Constants can't change while the program runs."
		}
		return p.What, p.Why
	}).setsVar(1)
	R("go", `^({I})(?:\s*=\s*iota.*)?$`, "var", "", "").with(func(m []string, x *ctx) (string, string) {
		if x.top() != "const" {
			return "", ""
		}
		return "Adds the constant " + code(m[1]) + " to the group.", "Constants in a group with «iota» get the next number automatically (0, 1, 2…), like an enum."
	})
	R("go", `^({I}(?:\s*,\s*{I})*)\s*:=\s*(.+)$`, "var", "", "").with(func(m []string, x *ctx) (string, string) {
		vars := splitTop(m[1], ',')
		if len(vars) == 2 && vars[1] == "_" {
			return "Gets " + describeExpr(x, m[2]) + ", keeping the first answer in " + code(vars[0]) + " and ignoring the second (the _ throws it away).", "The second result is usually an error; throwing it away is fine only when the call can't really fail."
		}
		if len(vars) == 2 && (vars[1] == "err" || vars[1] == "ok") {
			e := describeExpr(x, m[2])
			if vars[1] == "err" {
				return "Gets " + e + ", keeping the answer in " + code(vars[0]) + " and any error in " + code("err") + ".", "Go functions report problems by returning an error as their last value; the next lines usually check «if err != nil»."
			}
			return "Gets " + e + " into " + code(vars[0]) + ", with " + code("ok") + " saying whether it worked (or was found).", "The comma-ok pattern: ok is false when a map has no such key, a type check fails or a channel is closed."
		}
		if strings.HasPrefix(strings.TrimSpace(m[2]), "range ") {
			return "", ""
		}
		p := assignPart(x, "", m[1], ":", m[2])
		return p.What, p.Why + " (:= creates new variables and works out their type.)"
	}).setsVar(1)
	R("javascript,typescript", `^(?:export\s+)?(const|let|var)\s+(.+?)\s*(?::\s*[^=]+)?=\s*(.+)$`, "var", "", "").with(func(m []string, x *ctx) (string, string) {
		p := assignPart(x, m[1], m[2], "", m[3])
		switch m[1] {
		case "let":
			p.Why += " «let» makes a variable that can be changed later."
		case "var":
			p.Why += " «var» is the older way to make a variable; «let» and «const» are safer."
		}
		return codeMarks.Replace(p.What), codeMarks.Replace(p.Why)
	}).setsVar(2)
	R("javascript,typescript", `^(let|var)\s+({I}(?:\s*,\s*{I})*)$`, "var", "Creates the variable $2 with no value yet.", "It will be given a value later.", "variable").setsVar(2)
	R("rust", `^let\s+(mut\s+)?(\(.*?\)|{I})(?:\s*:\s*(.+?))?\s*=\s*(.+)$`, "var", "", "").with(func(m []string, x *ctx) (string, string) {
		p := assignPart(x, "let", m[2], "", m[4])
		if m[1] != "" {
			p.Why += " «mut» means it may change later; Rust variables can't change unless marked mut."
		} else {
			p.Why += " Without «mut» it can't be changed later."
		}
		return codeMarks.Replace(p.What), codeMarks.Replace(p.Why)
	}).setsVar(2)
	R("kotlin,swift", `^(?:(?:private|public|internal|lateinit|static|override|weak|lazy|@\w+)\s+)*(val|var|let)\s+({I})(?:\s*:\s*([^=]+?))?(?:\s*=\s*(.+))?$`, "var", "", "").with(func(m []string, x *ctx) (string, string) {
		if m[4] == "" {
			return "Creates " + code(m[2]) + " of type " + code(m[3]) + ", to be given a value later.", varWhy(x, m[2], "")
		}
		p := assignPart(x, m[1], m[2], "", m[4])
		if m[1] == "val" || m[1] == "let" {
			p.Why += " It's fixed once set («" + m[1] + "»); use «var» for something that changes."
		}
		return codeMarks.Replace(p.What), codeMarks.Replace(p.Why)
	}).setsVar(2)
	R("csharp,java,c,cpp", `^((?:(?:public|private|protected|internal|static|final|const|readonly|volatile|extern|unsigned|signed|register|constexpr|mutable|transient|\[SerializeField\]|\[Header\(.*?\)\]|UPROPERTY\(.*?\))\s+)*)((?:const\s+)?{T})\s+([*&]*{I})\s*(\[[^\]]*\])?\s*(.*)$`, "var", "", "").with(func(m []string, x *ctx) (string, string) {
		typ, name, arr, rest := strings.TrimSpace(m[2]), m[3], m[4], strings.TrimSpace(m[5])
		if cKeywords[typ] || typ == "return" || typ == "else" || typ == "goto" {
			return "", ""
		}
		if rest != "" && !strings.HasPrefix(rest, "=") && !strings.HasPrefix(rest, ",") && !strings.HasPrefix(rest, "{") && !strings.HasPrefix(rest, "(") {
			return "", ""
		}
		if strings.HasPrefix(rest, "(") && (x.opensBrace() || x.pending != "") {
			return "", ""
		}
		mods := m[1]
		ptr := strings.HasPrefix(name, "*")
		name = strings.TrimLeft(name, "*&")
		kind := "variable"
		if strings.Contains(mods, "const") || strings.Contains(mods, "final") || strings.Contains(mods, "readonly") {
			kind = "constant"
		}
		tn := typeName(typ)
		w := "Creates the " + kind + " " + code(name)
		if typ == "var" || typ == "auto" {
			w += " (its type is worked out from the value)"
		} else {
			w += ", which holds " + tn
		}
		if arr != "" {
			w += " — a list (array) of " + strings.Trim(arr, "[] ") + " of them"
		}
		if ptr {
			w += " (as a pointer: it stores the address of the value)"
		}
		switch {
		case strings.HasPrefix(rest, "="):
			w += ", set to " + describeExpr(x, strings.TrimSpace(rest[1:])) + "."
		case strings.HasPrefix(rest, "{"):
			w += ", filled with " + code(rest) + "."
		case strings.HasPrefix(rest, "("):
			w += ", made with " + code(unhide(rest, x.strs)) + "."
		case strings.HasPrefix(rest, ","):
			w += " (and the others listed after it)."
		default:
			w += "."
		}
		why := varWhy(x, name, "")
		if strings.Contains(mods, "public") && x.lang.ID == "csharp" && strings.Contains(x.dialect, "Unity") {
			why += " Public fields show in Unity's Inspector, so you can tune them without changing code."
		}
		if strings.Contains(mods, "SerializeField") {
			why += " [SerializeField] shows it in Unity's Inspector while keeping it private."
		}
		if strings.Contains(mods, "static") {
			why += " «static» means one shared copy for the whole class (or file), not one per object."
		}
		x.tag("variable")
		if x.top() == "class" || x.top() == "struct" {
			x.tag("class")
			w = strings.Replace(w, "Creates the variable", "Gives each object a field", 1)
		}
		return codeMarks.Replace(w), codeMarks.Replace(why)
	}).setsVar(3)
	R("csharp", `^((?:(?:public|private|protected|internal|static|virtual|override|abstract)\s+)*)({T})\s+({I})\s*\{\s*(get.*)\}(?:\s*=\s*(.+?))?;?$`, "var", "", "").with(func(m []string, x *ctx) (string, string) {
		w := "Defines the property " + code(m[3]) + " holding " + typeName(m[2]) + "."
		why := "A property looks like a field from outside but can control reading (get) and changing (set)."
		if !strings.Contains(m[4], "set") || strings.Contains(m[4], "private set") {
			why += " Other classes can read it but not change it."
		}
		return w, why
	}).setsVar(3)

	// ------------------------------------------------ control flow
	ifFn := func(m []string, x *ctx) (string, string) {
		cond := m[1]
		body := ""
		if len(m) > 2 {
			body = strings.TrimSpace(m[2])
		}
		w := "If-check: " + checks(x, cond) + "."
		why := "The code inside only runs when the condition is true; otherwise the program skips it."
		if body != "" && body != "{" {
			w = "If-check: " + checks(x, cond) + "; if so, it runs " + code(unhide(body, x.strs)) + "."
			why = "A one-line if: the action and the check share the line."
		}
		cl := strings.ToLower(cond)
		switch {
		case strings.Contains(cl, "err != nil"):
			w = "Checks whether the last step failed (err isn't nil)."
			why = "This is Go's normal error check: if something went wrong, deal with it here (often by returning the error)."
			x.tag("errors")
		case strings.Contains(cl, "== null") || strings.Contains(cl, "!= null") || strings.Contains(cl, "== nil") || strings.Contains(cl, "!= nil") || strings.Contains(cl, "== nullptr"):
			why += " Checking for null/nil avoids crashing on something that doesn't exist."
		}
		x.tag("if")
		return w, why
	}
	R(c, `^if\s*(\(.*)$`, "if", "", "").with(func(m []string, x *ctx) (string, string) {
		cond, rest, ok := parenGroup(m[1])
		if !ok {
			return "", ""
		}
		return ifFn([]string{"", cond, rest}, x)
	}).open("if", 0)
	R("go,rust,swift", `^if\s+(?:(.+?);\s*)?(.+)$`, "if", "", "").with(func(m []string, x *ctx) (string, string) {
		if m[1] != "" {
			w, y := ifFn([]string{"", m[2]}, x)
			return "First runs " + code(unhide(m[1], x.strs)) + ", then " + lowerFirst(w), y + " Variables made in the first part only exist inside this if."
		}
		if strings.HasPrefix(m[2], "let ") {
			return "Checks whether " + code(unhide(m[2][4:], x.strs)) + " has a value and, if so, uses it inside.", "«if let» unwraps an optional: the code inside only runs when the value isn't nil/None."
		}
		return ifFn([]string{"", m[2]}, x)
	}).open("if", 0)
	R("swift", `^guard\s+(.+?)\s+else$`, "if", "Makes sure $1; if not, the else part leaves early.", "guard keeps the main code un-indented by handling the bad case first.", "if").open("if", 0)
	R(c, `^else\s+if\s*\(?(.*?)\)?$`, "if", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("if")
		return "Otherwise, if-check: " + checks(x, m[1]) + ".", "Only tested when the checks above were false, so exactly one branch runs."
	}).open("if", 0)
	R(c, `^else$`, "if", "Otherwise (if none of the checks above were true), run the following block.", "The fallback branch of an if.", "if").open("if", 0)
	R(c, `^else\s+(.+)$`, "if", "Otherwise, runs $1.", "The fallback branch of an if, on one line.", "if")
	R(c, `^for\s*\(\s*(.*?)\s*;\s*(.*?)\s*;\s*(.*?)\s*\)\s*(.*)$`, "loop", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("loop")
		if m[1] == "" && m[2] == "" && m[3] == "" {
			return "Starts a loop that repeats forever (for(;;)).", "It only stops when something inside uses break or return. Main loops and game loops are often written like this."
		}
		return "A counting loop: first " + code(unhide(m[1], x.strs)) + ", then repeats while " + code(unhide(m[2], x.strs)) + ", doing " + code(unhide(m[3], x.strs)) + " after each round.", "Repeats the block a set number of times, usually to go through a list or count."
	}).open("loop", 0)
	R("java,cpp,c", `^for\s*\(\s*(?:final\s+)?(?:const\s+)?([\w<>\[\]&*:]+\s*&?)\s*&?\s*(\w+)\s*:\s*(.+)\)$`, "loop", "Goes through each item in $3, calling it $2 each time.", "A for-each loop: the same steps run for every item, without counting by hand.", "loop").open("loop", 0).setsVar(2)
	R("csharp", `^foreach\s*\(\s*(?:var|[\w<>\[\],. ]+?)\s+(\w+)\s+in\s+(.+)\)$`, "loop", "Goes through each item in $2, calling it $1 each time.", "A for-each loop runs the same steps for every item: every enemy, every bullet, every line of a file.", "loop").open("loop", 0).setsVar(1)
	R("javascript,typescript", `^for\s*\(\s*(?:const|let|var)?\s*(.+?)\s+(of|in)\s+(.+)\)$`, "loop", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("loop")
		if m[2] == "of" {
			return "Goes through each value in " + code(m[3]) + ", calling it " + code(m[1]) + " each time.", "for…of visits every item of a list, one at a time."
		}
		return "Goes through each property name (key) of " + code(m[3]) + ", calling it " + code(m[1]) + ".", "for…in visits the names, not the values; for lists, for…of is usually what you want."
	}).open("loop", 0).setsVar(1)
	R("php", `^foreach\s*\((.+?)\s+as\s+(.+)\)$`, "loop", "Goes through each item in $1 as $2.", "With $key => $value you get both the position and the item.", "loop").open("loop", 0)
	R("go", `^for\s+(?:(\w+)(?:\s*,\s*(\w+))?\s*:?=\s*)?range\s+(.+)$`, "loop", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("loop")
		what := "Goes through every item of " + code(unhide(m[3], x.strs))
		switch {
		case m[1] == "" && m[2] == "":
			what += " (without using the items)."
		case m[2] == "" && m[1] != "_":
			what += ", with " + code(m[1]) + " as the position (index or key) each time."
		case m[1] == "_":
			what += ", with " + code(m[2]) + " as each value (the underscore ignores the position)."
		default:
			what += ", with " + code(m[1]) + " as the position and " + code(m[2]) + " as the value."
		}
		return what, "«range» walks through lists, maps, text and channels one item at a time."
	}).open("loop", 0)
	R("go", `^for\s+(.+?);\s*(.+?);\s*(.+)$`, "loop", "A counting loop: starts with $1, repeats while $2, and does $3 after each round.", "Go's only loop keyword is for; this is the classic counting form.", "loop").open("loop", 0)
	R("go,rust", `^(?:for|loop)$`, "loop", "Starts a loop that repeats forever.", "It stops only when the code inside uses break or return. Servers and game loops often look like this.", "loop").open("loop", 0)
	R("go", `^for\s+(.+)$`, "loop", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("loop")
		return "Repeats while " + code(unhide(m[1], x.strs)) + " stays true (" + condWords(x, m[1]) + ").", "Go has no while keyword: a for with just a condition does the same job."
	}).open("loop", 0)
	R("rust,swift", `^for\s+(.+?)\s+in\s+(.+)$`, "loop", "Goes through each item of $2, calling it $1.", "Ranges like 0..10 count from 0 up to 9.", "loop").open("loop", 0).setsVar(1)
	R("kotlin", `^for\s*\((\w+)\s+in\s+(.+)\)$`, "loop", "Goes through each item of $2, calling it $1.", "1..10 counts from 1 to 10; until stops one short.", "loop").open("loop", 0).setsVar(1)
	R(c, `^while\s*(\(.*)$`, "loop", "", "").with(func(m []string, x *ctx) (string, string) {
		cond, rest, ok := parenGroup(m[1])
		if !ok {
			return "", ""
		}
		m = []string{"", cond, rest}
		x.tag("loop")
		if strings.HasPrefix(x.cur, "}") || (strings.HasSuffix(x.cur, ";") && m[2] == "") {
			return "…and repeats the do-block above while " + code(unhide(m[1], x.strs)) + ".", "A do…while loop always runs at least once, then checks."
		}
		c := strings.TrimSpace(m[1])
		if c == "true" || c == "1" || c == "running" || c == "!quit" || c == "!done" || c == "!WindowShouldClose()" || c == "!glfwWindowShouldClose(window)" {
			x.tag("game-loop")
			return "Starts a loop that keeps going " + map[bool]string{true: "forever", false: "while " + code(c)}[c == "true" || c == "1"] + ".", "A main loop: programs and games repeat 'read input, update, draw' until told to stop."
		}
		return "A while-loop: " + checks(x, m[1]) + ", and repeats the block for as long as it stays true.", "Use while when you don't know in advance how many times to repeat."
	}).open("loop", 0)
	R("rust,swift,kotlin", `^while\s+(.+)$`, "loop", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("loop")
		return "A while-loop: repeats while " + code(unhide(m[1], x.strs)) + " (" + condWords(x, m[1]) + ").", ""
	}).open("loop", 0)
	R(c, `^do$`, "loop", "Starts a do…while loop: the block runs once, then repeats while the check at the end is true.", "Useful when the steps must happen at least once, such as asking for input.", "loop").open("loop", 0)
	R("swift", `^repeat$`, "loop", "Starts a repeat…while loop: the block runs at least once.", "", "loop").open("loop", 0)
	R("gml", `^repeat\s*\((.+)\)$`, "loop", "Repeats the block $1 times.", "GameMaker's simple counted loop.", "loop").open("loop", 0)
	R("gml", `^with\s*\((.+)\)$`, "loop", "Runs the block as if it were inside every instance of $1.", "GameMaker's with: a quick way to change all enemies or one object at once.", "loop").open("loop", 0)
	R(c, `^switch\s*\((.*)\)$`, "switch", "Chooses what to do based on the value of $1.", "A switch is a tidy replacement for a long chain of if / else if, one case per value.", "switch").open("switch", 0)
	R("go", `^switch\s*(.*)$`, "switch", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("switch")
		if strings.TrimSpace(m[1]) == "" {
			return "Starts a switch with no value: each case is its own condition, checked from top to bottom.", "A neat way to write a chain of if / else if."
		}
		return "Chooses what to do based on " + code(unhide(m[1], x.strs)) + ".", "Go runs only the matching case; there's no need for break."
	}).open("switch", 0)
	R("go", `^select$`, "switch", "Waits until one of several channel operations below is ready, then runs that case.", "select lets a goroutine listen to several channels at once, such as messages and a timeout.", "concurrency").open("switch", 0)
	R("rust", `^match\s+(.+)$`, "switch", "Compares $1 against each pattern below and runs the first that fits.", "Rust checks every possible case is handled, so nothing is forgotten.", "switch").open("switch", 0)
	R("kotlin", `^when\s*(?:\((.*)\))?$`, "switch", "Chooses a branch based on $1.", "Kotlin's when is a powerful switch.", "switch").open("switch", 0)
	R(c, `^case\s+(.+?)\s*:\s*(.*)$`, "case", "", "").with(func(m []string, x *ctx) (string, string) {
		w := "If the value is " + code(unhide(m[1], x.strs)) + ", run the lines below."
		if m[2] != "" {
			w = "If the value is " + code(unhide(m[1], x.strs)) + ", run " + code(unhide(m[2], x.strs)) + "."
		}
		why := "One branch of the switch."
		if x.lang.ID != "go" && x.lang.ID != "swift" {
			why += " Without a break at its end, the program carries on into the next case."
		}
		return w, why
	})
	R(c, `^default\s*:\s*(.*)$`, "case", "If no other case matched, run the lines below.", "The catch-all branch of a switch.")
	R("rust", `^(.+?)\s*=>\s*(.*?),?$`, "case", "If the value matches $1, the result is $2.", "One arm of a match.")
	R("kotlin", `^(.+?)\s*->\s*(.*)$`, "case", "If it's $1, do $2.", "One branch of when.")
	R(c, `^break$`, "jump", "Stops the loop (or switch) straight away and carries on after it.", "Used when the job is done early, for example once an item is found.", "loop")
	R(c, `^continue$`, "jump", "Skips the rest of this round of the loop and starts the next round.", "Handy for ignoring items that don't matter.", "loop")
	R(c, `^return\s*(.*)$`, "return", "", "").with(func(m []string, x *ctx) (string, string) {
		v := strings.TrimSpace(strings.TrimSuffix(m[1], ";"))
		x.tag("function")
		if v == "" {
			return "Leaves the function now.", "Nothing after this line in the function runs; the program goes back to where the function was called."
		}
		if x.lang.ID == "go" && (strings.HasSuffix(v, "err") || strings.Contains(v, "fmt.Errorf") || strings.Contains(v, "errors.New")) {
			x.tag("errors")
			return "Leaves the function and passes the error back: " + code(unhide(v, x.strs)) + ".", "The caller will see the error and decide what to do; that's how Go reports problems."
		}
		return "Leaves the function and gives back " + describeExpr(x, v) + ".", "The value goes to whoever called the function, so they can use the answer."
	})
	R("csharp", `^yield\s+return\s+(.+)$`, "return", "", "").with(func(m []string, x *ctx) (string, string) {
		v := unhide(m[1], x.strs)
		x.tag("timers")
		if strings.Contains(v, "WaitForSeconds") {
			return "Pauses this coroutine for " + code(v) + " and then carries on from here.", "Unity coroutines let you write 'wait 2 seconds, then…' without freezing the game: other code keeps running meanwhile."
		}
		if strings.Contains(v, "null") {
			return "Pauses this coroutine until the next frame.", "In Unity, yield return null waits exactly one frame; handy for spreading work out."
		}
		return "Hands back " + code(v) + " and pauses until asked for the next one.", "yield return makes an iterator (or a Unity coroutine)."
	})
	R(c, `^goto\s+(\w+)$`, "jump", "Jumps to the label $1.", "goto makes code hard to follow; modern code rarely uses it except for cleanup in C.")
	R(c, `^({I})\s*:$`, "label", "A label called $1, which goto, break or continue can jump to.", "")
	R(c, `^try$`, "try", "Starts a try block: if anything inside fails, the program jumps to the catch part instead of crashing.", "Used around risky steps such as reading files or network requests.", "errors").open("try", 0)
	R(c, `^catch\s*(?:\((.*)\))?$`, "try", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("errors")
		if m[1] == "" {
			return "Handles any error from the try block above.", "This is where the program recovers: show a message, use a default, or tidy up."
		}
		return "Handles an error from the try block above; the error is " + code(m[1]) + ".", "This is where the program recovers: show a message, use a default, or tidy up."
	}).open("try", 0)
	R(c, `^finally$`, "try", "Code that always runs after the try (whether it failed or not).", "Used for clean-up such as closing files.", "errors").open("try", 0)
	R(c, `^throw\s+(.+)$`, "error", "Raises the error $1.", "Stops this function and sends the error up to the nearest catch.", "errors")
	R("go", `^panic\((.*)\)$`, "error", "Stops the program with the message $1 (a panic).", "Panics are for 'this should never happen' situations; normal problems are returned as errors.", "errors")
	R("go", `^defer\s+(.+)$`, "call", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("defer")
		return "Schedules " + code(unhide(m[1], x.strs)) + " to run when this function finishes.", "defer makes sure clean-up (closing a file, unlocking, closing a Windows handle) happens however the function ends."
	})
	R("go", `^go\s+(.+)$`, "call", "Starts $1 running in a new goroutine, alongside the rest of the program.", "Goroutines are Go's lightweight threads: the program carries on without waiting for this to finish.", "concurrency")
	R("go", `^({D})\s*<-\s*(.+)$`, "call", "Sends $2 into the channel $1.", "Channels pass values safely between goroutines; this waits until someone receives it (unless the channel has room).", "concurrency")
	R("go", `^<-\s*(.+)$`, "call", "Waits for a value from the channel $1 (and throws it away).", "Often used to wait for a signal that some work has finished.", "concurrency")
	R("go", `^case\s+(.+?)\s*:?=\s*<-\s*(.+?):$`, "case", "If a value arrives on $2, store it in $1 and run the lines below.", "One branch of a select.", "concurrency")
	R("rust", `^({I})!\s*\((.*)\)$`, "call", "", "").with(func(m []string, x *ctx) (string, string) {
		d := map[string][2]string{
			"println":   {"Prints a line of text: " + expand("$2", m, x) + ".", "{} in the text is replaced by the next value; {:?} prints a debug view."},
			"print":     {"Prints text without a new line: " + expand("$2", m, x) + ".", ""},
			"eprintln":  {"Prints an error message to the error output.", "Kept separate from normal output so errors stand out."},
			"panic":     {"Stops the program with the message " + expand("$2", m, x) + ".", "For situations that should never happen."},
			"assert":    {"Checks that " + expand("$2", m, x) + " is true, and stops the program if not.", "Catches mistakes early, mostly in tests."},
			"assert_eq": {"Checks the two values " + expand("$2", m, x) + " are equal, and stops if not.", "Mainly used in tests."},
			"todo":      {"Marks unfinished code; the program stops if it gets here.", ""},
			"dbg":       {"Prints a value with its file and line, for debugging.", ""},
			"write":     {"Writes formatted text into something, such as a file or string.", ""},
		}[m[1]]
		if d[0] == "" {
			return "Uses the macro " + code(m[1]+"!") + " with " + expand("$2", m, x) + ".", "Names ending in ! are macros: they write code for you when compiling."
		}
		x.tag("output")
		return d[0], d[1]
	})
	R("cpp", `^delete(\[\])?\s+(.+)$`, "call", "Frees the memory used by $2.", "Anything made with new must be deleted once, or memory leaks.", "memory")
	R("javascript,typescript", `^delete\s+(.+)$`, "call", "Removes the property $1 from its object.", "")
	R("csharp", `^using\s*\((.+)\)$`, "try", "Uses $1 and automatically closes or frees it at the end of the block.", "Ensures files and connections are always closed, even after an error.").open("block", 0)
	R("csharp,java", `^(?:lock|synchronized)\s*\((.+)\)$`, "try", "Lets only one thread at a time run the block, using $1 as the lock.", "Stops two threads changing the same data at once.", "concurrency").open("block", 0)
	R("rust,csharp", `^unsafe$`, "directive", "Starts an unsafe block: the compiler stops checking memory safety inside.", "Needed for raw pointers and calling C or Windows functions; it's up to the programmer to get it right.").open("block", 0)
	R("go", `^if\s+err\s*:?=\s*(.+);\s*err\s*!=\s*nil$`, "if", "Runs $1 and checks straight away whether it failed.", "Go style: do the step and check the error in one line.", "errors").first().open("if", 0)
	R("php", `^echo\s+(.+)$`, "print", "Sends $1 to the web page (or console).", "echo is how PHP writes output, usually HTML for the browser.", "output")
	R("javascript,typescript", `^(?:"use strict"|'use strict'|{S})$`, "directive", "", "").with(func(m []string, x *ctx) (string, string) {
		if len(x.strs) > 0 && x.strs[0] == "use strict" {
			return "Turns on strict mode.", "Strict mode turns some silent mistakes into errors, which makes bugs easier to find."
		}
		return "", ""
	})
	R(c, `^(?:super|base)\s*[.(]\s*(.*)$`, "call", "Runs the parent class's version ($1).", "So the parent's own setup still happens before (or after) this class adds its part.", "inheritance")
	R("cpp", `^Super::(\w+)\((.*)\)$`, "call", "Runs the parent class's $1 first.", "In Unreal you call Super:: so the engine's own code for this event still runs.", "inheritance")
}

// parenGroup splits "(a && (b)) rest" into "a && (b)" and "rest".
func parenGroup(s string) (string, string, bool) {
	d := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			d++
		case ')':
			d--
			if d == 0 {
				return s[1:i], strings.TrimSpace(s[i+1:]), true
			}
		}
	}
	return "", "", false
}

func typeName(t string) string {
	t = strings.TrimSpace(t)
	m := map[string]string{
		"int": "a whole number", "long": "a big whole number", "short": "a small whole number", "byte": "a small number (0–255 or -128–127)",
		"float": "a number with a decimal point", "double": "a precise decimal number", "bool": "true or false", "boolean": "true or false",
		"char": "a single character", "string": "text", "String": "text", "std::string": "text", "uint8_t": "a byte (0–255)", "int32_t": "a 32-bit whole number",
		"uint32_t": "a 32-bit whole number (no minus)", "size_t": "a size or count", "var": "a value", "auto": "a value", "void*": "a pointer to anything",
		"Vector2": "a 2D position or direction (x, y)", "Vector3": "a 3D position or direction (x, y, z)", "FVector": "a 3D position or direction (Unreal)",
		"GameObject": "a Unity GameObject (a thing in the scene)", "Transform": "a position, rotation and scale", "Rigidbody": "a physics body", "Rigidbody2D": "a 2D physics body",
		"AudioSource": "a sound player", "Animator": "an animation controller", "Texture2D": "a picture", "SpriteBatch": "a sprite drawer", "HANDLE": "a Windows handle",
		"HWND": "a window handle", "DWORD": "a 32-bit Windows number", "LPCWSTR": "a pointer to wide (UTF-16) Windows text", "BOOL": "a Windows true/false (a number)",
	}
	if d, ok := m[t]; ok {
		return d + " (" + code(t) + ")"
	}
	if strings.HasPrefix(t, "List<") || strings.HasPrefix(t, "std::vector<") || strings.HasPrefix(t, "ArrayList<") || strings.HasPrefix(t, "vector<") {
		return "a growable list " + code(t)
	}
	if strings.HasPrefix(t, "Dictionary<") || strings.HasPrefix(t, "std::map<") || strings.HasPrefix(t, "HashMap<") || strings.HasPrefix(t, "Map<") {
		return "a lookup table " + code(t)
	}
	if strings.HasSuffix(t, "[]") {
		return "a list of " + code(strings.TrimSuffix(t, "[]"))
	}
	if strings.HasSuffix(t, "*") {
		return "a pointer to " + code(strings.TrimSpace(strings.TrimSuffix(t, "*")))
	}
	return "a " + code(t)
}
