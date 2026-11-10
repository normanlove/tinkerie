// Codie — game.go: the game-development view. It recognises the engine or
// framework, explains how that engine runs its game loop, and finds the game
// logic in the code: input, movement, collisions, spawning, timers and more.
// Part of the free nrmluvdev maker suite. Created by nrmluvdev.
package codie

import (
	"regexp"
	"sort"
	"strings"
)

type Callback struct {
	Name string `json:"name"`
	When string `json:"when"`
	Line int    `json:"line,omitempty"`
}

type Engine struct {
	ID    string     `json:"id"`
	Name  string     `json:"name"`
	About string     `json:"about"`
	Loop  string     `json:"loop"`
	Frame []Callback `json:"frame"`
	Tips  []string   `json:"tips"`
	langs []string
	sig   *regexp.Regexp
}

type GameHit struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Text  string `json:"text"`
	Lines []int  `json:"lines"`
}

type GameReport struct {
	Engine  *Engine   `json:"engine"`
	Hits    []GameHit `json:"hits"`
	Lessons []string  `json:"lessons"`
	IsGame  bool      `json:"isGame"`
}

func cb(name, when string) Callback { return Callback{Name: name, When: when} }

var engines = []*Engine{
	{ID: "unity", Name: "Unity", langs: []string{"csharp"}, sig: regexp.MustCompile(`MonoBehaviour|using UnityEngine`),
		About: "Unity is a popular engine for 2D, 3D and mobile games. You write C# scripts and attach them to GameObjects in the editor.",
		Loop:  "You don't write the game loop in Unity: the engine runs it and calls your script's methods at the right moments, every frame, for every active object.",
		Frame: []Callback{cb("Awake", "once, when the object is created"), cb("Start", "once, before its first frame"), cb("FixedUpdate", "at a steady 50 times a second: physics"), cb("OnTriggerEnter2D", "when physics finds an overlap"), cb("OnCollisionEnter2D", "when physics finds a collision"), cb("Update", "every frame: input and game logic"), cb("LateUpdate", "every frame, after all Updates: cameras"), cb("OnDestroy", "when the object is removed")},
		Tips:  []string{"Read input in Update and apply physics forces in FixedUpdate.", "Multiply movement by Time.deltaTime so it's per second, not per frame.", "Cache GetComponent results in Start instead of calling it every frame.", "Use public or [SerializeField] fields to tune values in the Inspector."}},
	{ID: "unreal", Name: "Unreal Engine", langs: []string{"cpp"}, sig: regexp.MustCompile(`UCLASS\(|AActor|ACharacter|GENERATED_BODY\(|\bA[A-Z]\w+::|Super::(Tick|BeginPlay)`),
		About: "Unreal Engine is a powerful 3D engine. Gameplay is written in C++ classes (or Blueprints), mostly Actors placed in levels.",
		Loop:  "Unreal runs the loop: every frame it ticks each Actor that has ticking switched on, passing DeltaTime.",
		Frame: []Callback{cb("BeginPlay", "once, when the actor enters the game"), cb("SetupPlayerInputComponent", "once, to bind keys to functions"), cb("Tick", "every frame, with DeltaTime"), cb("EndPlay", "when the actor leaves the game")},
		Tips:  []string{"Always call Super::BeginPlay() and Super::Tick(DeltaTime).", "Turn off ticking (PrimaryActorTick.bCanEverTick = false) on actors that don't need it.", "UPROPERTY(EditAnywhere) lets designers tweak values in the editor."}},
	{ID: "godot", Name: "Godot", langs: []string{"gdscript", "csharp"}, sig: regexp.MustCompile(`(?m)^extends \w+|using Godot|func _ready\(|func _process\(|_PhysicsProcess\(`),
		About: "Godot is a free, open-source engine. Games are built from scenes of nodes, each node with an optional script.",
		Loop:  "Godot runs the loop and calls each node's callbacks: _process every frame and _physics_process at a fixed rate, both with delta.",
		Frame: []Callback{cb("_ready", "once, when the node and its children are in the scene"), cb("_input", "whenever a key, click or touch happens"), cb("_physics_process", "60 times a second: movement and physics"), cb("_process", "every frame: animation, timers, other logic"), cb("_draw", "when the node is redrawn")},
		Tips:  []string{"Movement with velocity and move_and_slide() goes in _physics_process.", "Use Input Map actions (Project Settings) instead of raw keys, so players can remap them.", "Signals (like body_entered) are the Godot way for nodes to react to each other."}},
	{ID: "love", Name: "LÖVE", langs: []string{"lua"}, sig: regexp.MustCompile(`\blove\.\w+`),
		About: "LÖVE (Love2D) is a free framework for 2D games in Lua. You write a main.lua with callback functions.",
		Loop:  "LÖVE runs the loop for you: it calls love.update(dt) and then love.draw() every frame, and other callbacks when events happen.",
		Frame: []Callback{cb("love.load", "once, at the start"), cb("love.keypressed", "when a key goes down"), cb("love.update", "every frame, with dt"), cb("love.draw", "every frame, after update")},
		Tips:  []string{"Keep all changes to the game in love.update and only draw in love.draw.", "Multiply speeds by dt.", "Load images and sounds once in love.load."}},
	{ID: "pico8", Name: "PICO-8", langs: []string{"lua"}, sig: regexp.MustCompile(`(?m)^\s*function _(init|update|update60|draw)\(`),
		About: "PICO-8 is a 'fantasy console': a tiny pretend 1980s machine with a 128×128 screen, 16 colours and built-in sprite, map and sound editors.",
		Loop:  "PICO-8 calls _update (30 times a second, or _update60 for 60) and then _draw, forever. The speed is fixed, so there's no delta time.",
		Frame: []Callback{cb("_init", "once, when the cartridge starts"), cb("_update", "30 times a second: game logic"), cb("_update60", "60 times a second instead of _update"), cb("_draw", "after each update: drawing")},
		Tips:  []string{"Read buttons with btn() for holding and btnp() for single presses.", "Start _draw with cls() so the last frame is wiped.", "Use sprite flags (fget) to mark solid map tiles for collisions."}},
	{ID: "roblox", Name: "Roblox", langs: []string{"lua"}, sig: regexp.MustCompile(`game:GetService\(|Instance\.new\(|\bworkspace\b|task\.wait\(|:Connect\(`),
		About: "Roblox games are scripted in Luau, a faster Lua. Scripts run on the server (Script) or on each player's device (LocalScript).",
		Loop:  "Roblox is event-driven: you connect functions to events (Touched, PlayerAdded, InputBegan) and to RunService.Heartbeat for per-frame code.",
		Frame: []Callback{cb("PlayerAdded", "when a player joins"), cb("InputBegan", "when a key or touch starts (LocalScript)"), cb("Stepped", "every frame, before physics"), cb("Heartbeat", "every frame, after physics, with the time step"), cb("Touched", "when parts touch")},
		Tips:  []string{"Never trust the client: important changes (scores, damage) should happen in server Scripts.", "A while true loop needs task.wait() inside, or the game freezes.", "Use RemoteEvents to send messages between LocalScripts and the server."}},
	{ID: "solar2d", Name: "Solar2D (Corona)", langs: []string{"lua"}, sig: regexp.MustCompile(`display\.new\w+\(|Runtime:addEventListener\(`),
		About: "Solar2D is a free Lua framework for 2D mobile games.",
		Loop:  "Solar2D sends an enterFrame event every frame; you attach a listener to run per-frame code.",
		Frame: []Callback{cb("enterFrame", "every frame"), cb("touch", "when the screen is touched"), cb("collision", "when physics bodies collide")}},
	{ID: "pygame", Name: "pygame", langs: []string{"python"}, sig: regexp.MustCompile(`\bpygame\b`),
		About: "pygame is a Python library for 2D games: windows, drawing, sound and input.",
		Loop:  "In pygame you write the game loop yourself: a while loop that handles events, updates the game, draws everything, flips the display and ticks the clock.",
		Frame: []Callback{cb("pygame.event.get", "each frame: read every event"), cb("update", "move things and check collisions"), cb("fill", "clear the screen"), cb("blit", "draw sprites"), cb("pygame.display.flip", "show the finished frame"), cb("tick", "wait to keep a steady frame rate")},
		Tips:  []string{"Call clock.tick(60) once per loop to cap the frame rate.", "Use pygame.Rect and colliderect for simple collisions.", "Load and .convert() images once, before the loop."}},
	{ID: "phaser", Name: "Phaser", langs: []string{"javascript", "typescript"}, sig: regexp.MustCompile(`\bPhaser\.|this\.physics\.add|this\.load\.(image|spritesheet)`),
		About: "Phaser is a popular JavaScript framework for 2D browser games.",
		Loop:  "Each Phaser scene has preload, create and update functions; Phaser calls update every frame with the time and delta.",
		Frame: []Callback{cb("preload", "once: load pictures and sounds"), cb("create", "once: build the scene"), cb("update", "every frame")},
		Tips:  []string{"Use arcade physics colliders and overlaps instead of hand-written collision checks.", "this.input.keyboard.createCursorKeys() gives easy arrow-key input."}},
	{ID: "canvas", Name: "HTML canvas (JavaScript)", langs: []string{"javascript", "typescript", "html"}, sig: regexp.MustCompile(`requestAnimationFrame|getContext\(\s*\S2d\S\s*\)|getContext\(`),
		About: "A browser game drawn on a <canvas> with plain JavaScript.",
		Loop:  "The game loop is a function that updates, draws, and then calls requestAnimationFrame(loop) to run again before the next screen refresh.",
		Frame: []Callback{cb("addEventListener", "when keys or clicks happen (store them for the loop)"), cb("update", "move things using the time since last frame"), cb("clearRect", "wipe the canvas"), cb("draw", "paint the frame"), cb("requestAnimationFrame", "ask for the next frame")},
		Tips:  []string{"requestAnimationFrame passes a timestamp: subtract the last one for delta time.", "Keep a 'keys held' object updated by keydown/keyup events, and read it in update."}},
	{ID: "raylib", Name: "raylib", langs: []string{"c", "cpp", "go"}, sig: regexp.MustCompile(`InitWindow\(|raylib|WindowShouldClose\(`),
		About: "raylib is a simple C library for games, also usable from C++, Go and many other languages.",
		Loop:  "You write the loop: while (!WindowShouldClose()) { update; BeginDrawing(); draw; EndDrawing(); }. SetTargetFPS sets its speed.",
		Frame: []Callback{cb("WindowShouldClose", "each round: should the game stop?"), cb("IsKeyDown", "read input"), cb("GetFrameTime", "delta time"), cb("BeginDrawing", "start the frame"), cb("ClearBackground", "wipe the screen"), cb("EndDrawing", "show the frame")}},
	{ID: "sdl", Name: "SDL", langs: []string{"c", "cpp"}, sig: regexp.MustCompile(`SDL_\w+\(`),
		About: "SDL is a long-standing C library for windows, graphics, sound and input, used by many commercial games.",
		Loop:  "You write the loop: poll events, update, clear, draw, present.",
		Frame: []Callback{cb("SDL_PollEvent", "read every event"), cb("SDL_GetTicks", "measure time"), cb("SDL_RenderClear", "wipe the screen"), cb("SDL_RenderPresent", "show the frame")}},
	{ID: "monogame", Name: "MonoGame / XNA", langs: []string{"csharp"}, sig: regexp.MustCompile(`Microsoft\.Xna\.Framework|\bGameTime\b`),
		About: "MonoGame (the successor of Microsoft XNA) is a C# framework used for games such as Stardew Valley and Celeste.",
		Loop:  "Your class inherits from Game; MonoGame calls Update(gameTime) and then Draw(gameTime) every frame.",
		Frame: []Callback{cb("Initialize", "once"), cb("LoadContent", "once: load pictures and sounds"), cb("Update", "every frame: input and logic"), cb("Draw", "every frame: drawing")}},
	{ID: "gamemaker", Name: "GameMaker", langs: []string{"gml"}, sig: regexp.MustCompile(`.`),
		About: "GameMaker uses objects with events. GML code goes in an object's events instead of in functions you call yourself.",
		Loop:  "Every step (60 times a second by default) GameMaker runs each instance's Step event, then its Draw event. Create runs once; Alarms count down and fire their events.",
		Frame: []Callback{cb("Create", "once, when the instance is made"), cb("Step", "every step: input, movement, collisions"), cb("Collision", "when it touches another object"), cb("Alarm", "when an alarm reaches zero"), cb("Draw", "every step, to draw")},
		Tips:  []string{"Check place_meeting before moving to stop at walls.", "Use alarms (alarm[0] = 60) for timers."}},
	{ID: "ebiten", Name: "Ebitengine", langs: []string{"go"}, sig: regexp.MustCompile(`ebiten\.`),
		About: "Ebitengine is a simple 2D game library for Go that builds for Windows, macOS, Linux, the web and phones.",
		Loop:  "Your Game type has Update, Draw and Layout methods. Ebitengine calls Update 60 times a second (a fixed tick) and Draw every frame.",
		Frame: []Callback{cb("Update", "60 ticks a second: game logic"), cb("Draw", "every frame: drawing"), cb("Layout", "when the window size changes")},
		Tips:  []string{"Update runs at a fixed tick rate, so you can move by a fixed amount per tick.", "Return an error from Update to end the game."}},
	{ID: "bevy", Name: "Bevy", langs: []string{"rust"}, sig: regexp.MustCompile(`\bbevy::|App::new\(\)`),
		About: "Bevy is a Rust engine built around ECS: entities (things), components (data) and systems (functions).",
		Loop:  "Bevy runs the loop and calls every system added to the Update schedule each frame, giving each the data it asks for.",
		Frame: []Callback{cb("Startup", "once: spawn the world"), cb("Update", "every frame: all game systems")}},
	{ID: "spritekit", Name: "SpriteKit", langs: []string{"swift"}, sig: regexp.MustCompile(`SpriteKit|SKScene`),
		About: "SpriteKit is Apple's 2D game framework for iPhone, iPad and Mac.",
		Loop:  "SpriteKit calls the scene's update(_ currentTime:) every frame, then runs actions and physics, then draws.",
		Frame: []Callback{cb("didMove", "once, when the scene appears"), cb("touchesBegan", "when the screen is touched"), cb("update", "every frame"), cb("didBegin", "when physics bodies touch")}},
	{ID: "arduino", Name: "Arduino", langs: []string{"cpp", "c"}, sig: regexp.MustCompile(`void setup\s*\(\)|void loop\s*\(\)`),
		About: "Arduino boards run one program that starts with setup() and then repeats loop() forever, the same shape as a game loop.",
		Loop:  "The board calls setup() once and then loop() over and over as fast as it can.",
		Frame: []Callback{cb("setup", "once, at power-up"), cb("loop", "forever")}},
	{ID: "c64", Name: "Commodore 64 BASIC", langs: []string{"basic"}, sig: regexp.MustCompile(`(?i)POKE\s*5328[01]|PEEK\s*\(\s*5632[01]|POKE\s*532(4[89]|[5-9]\d)|CHR\$\(147\)|POKE\s*54\d{3}`),
		About: "The Commodore 64 (1982) is the best-selling home computer ever. Its BASIC has no graphics or sound commands, so games POKE the VIC-II (graphics) and SID (sound) chips directly.",
		Loop:  "There's no engine: the program's main loop is a set of lines ending in GOTO back to the top. Each pass reads the joystick (PEEK(56320)), moves sprites (POKE 53248+…) and checks collisions (PEEK(53278)).",
		Frame: []Callback{cb("PEEK(56320)", "read joystick 2"), cb("POKE 53248", "move sprite 0"), cb("PEEK(53278)", "read sprite collisions"), cb("GOTO", "back to the top of the loop")},
		Tips:  []string{"Joystick bits go to 0 when pushed: (PEEK(56320) AND 1)=0 means up.", "Sprite X positions above 255 need the bit in 53264.", "BASIC is slow: real C64 games used machine code, often started with SYS."}},
	{ID: "zx", Name: "ZX Spectrum BASIC", langs: []string{"basic"}, sig: regexp.MustCompile(`(?i)\bBORDER\s+\d|\bPAPER\s+\d|\bINK\s+\d|PRINT AT\b|\bBEEP\s`),
		About: "The Sinclair ZX Spectrum (1982) was Britain's favourite games machine. Its BASIC has PRINT AT, INKEY$ and BEEP built in.",
		Loop:  "The main loop is lines ending in GOTO: read keys with INKEY$, PRINT AT the new positions, check what's on the screen with ATTR or SCREEN$, repeat.",
		Frame: []Callback{cb("INKEY$", "read the keyboard"), cb("PRINT AT", "draw at a row and column"), cb("ATTR", "check colours on screen for collisions"), cb("GOTO", "back to the top")},
		Tips:  []string{"Rub out the old position (PRINT AT y,x;\" \") before printing the new one.", "UDGs (user-defined graphics) give custom 8×8 characters for sprites."}},
	{ID: "bbc", Name: "BBC Micro BASIC", langs: []string{"basic"}, sig: regexp.MustCompile(`(?i)\bDEF ?PROC|\bGCOL\b|\bVDU\s*\d|\bMODE\s*\d|INKEY\s*\(\s*-`),
		About: "The BBC Micro (1981) was in nearly every British school. BBC BASIC is fast and structured, with procedures, REPEAT…UNTIL and good graphics commands.",
		Loop:  "A REPEAT … UNTIL loop: read keys with INKEY(-key), move, redraw with MOVE/DRAW/PLOT or VDU, and repeat until the game ends.",
		Frame: []Callback{cb("INKEY(-", "check held keys"), cb("PROC", "update and draw procedures"), cb("UNTIL", "repeat until game over")},
		Tips:  []string{"Negative INKEY numbers check keys directly: INKEY(-98) is Z.", "*FX 15,0 empties the keyboard buffer."}},
	{ID: "qbasic", Name: "QBasic / QB64", langs: []string{"basic"}, sig: regexp.MustCompile(`(?i)\bSCREEN\s+\d+|\b_LIMIT\b|\bINKEY\$|\bPSET\b|\bLINE\s*\(`),
		About: "QBasic came with MS-DOS; QB64 runs the same code on modern Windows, macOS and Linux with extras like _LIMIT and _DISPLAY.",
		Loop:  "A DO … LOOP: read INKEY$, update, draw, and in QB64 call _DISPLAY and _LIMIT 60 to show the frame and keep a steady speed.",
		Frame: []Callback{cb("INKEY$", "read a key"), cb("CLS", "clear"), cb("LINE", "draw"), cb("_DISPLAY", "show the frame"), cb("_LIMIT", "cap the frame rate"), cb("LOOP", "repeat")},
		Tips:  []string{"SCREEN 13 gives 320×200 with 256 colours.", "_LIMIT 60 stops the game running too fast on modern PCs."}},
	{ID: "apple2", Name: "Apple II BASIC", langs: []string{"basic"}, sig: regexp.MustCompile(`(?i)\bHGR\b|\bHPLOT\b|\bPDL\s*\(`),
		About: "The Apple II (1977) used Applesoft BASIC with HGR hi-res graphics and paddle controllers.",
		Loop:  "Lines ending in GOTO: read paddles with PDL(0) or keys with PEEK(-16384), HPLOT the new picture, repeat.",
		Frame: []Callback{cb("PDL(0)", "read the paddle"), cb("PEEK(-16384)", "read the keyboard"), cb("HPLOT", "draw"), cb("GOTO", "repeat")}},
}

// Game concepts found by pattern, in any language.
type gameConcept struct {
	id, title, text string
	re              *regexp.Regexp
}

func gc(id, title, text, pat string) gameConcept {
	return gameConcept{id, title, text, regexp.MustCompile(`(?i)` + pat)}
}

var gameConcepts = []gameConcept{
	gc("game-loop", "Game loop", "The heartbeat of every game: over and over, read input, update the world and draw it. Each pass is one frame.", `requestAnimationFrame|clock\.tick|while\s*\(?\s*(true|running|1|!quit)\b|WindowShouldClose|love\.update|function _update|func _process|void Update\s*\(|Tick\(float|\b_LIMIT\b|RunService\.\w+:Connect|ebiten\.RunGame`),
	gc("delta-time", "Delta time", "Delta time is the time since the last frame. Multiplying speeds by it keeps the game the same speed on fast and slow computers.", `deltaTime|\bdelta\b|\bdt\b|GetFrameTime|ElapsedGameTime|delta_seconds|delta_secs`),
	gc("input", "Player input", "Reading what the player presses. 'Held' checks give smooth movement; 'just pressed' checks give one action per press.", `GetKey|GetAxis|is_action|keyboard\.isDown|\bbtnp?\(|INKEY|keyboard_check|IsKeyDown|IsKeyPressed|key\.get_pressed|addEventListener\(\s*\S(keydown|keyup)|GetAsyncKeyState|PEEK\s*\(\s*5632[01]|\bSTICK\b|\bPDL\s*\(|InputBegan|event\.key|K_LEFT|K_SPACE|keyCode|e\.key\b`),
	gc("movement", "Movement", "Moving things by changing their position a little every frame, usually by a velocity (speed with a direction).", `\b(x|y|pos\w*|position)\s*[+-]=|velocity|\bvel[xy]?\b|\b[vd][xy]\b|\bhsp\b|\bvsp\b|Translate\(|MovePosition|move_toward|SetActorLocation|AddMovementInput`),
	gc("physics", "Gravity and physics", "Gravity is just adding a little to the downward speed every frame. Jumping sets a big upward speed; gravity pulls it back.", `gravity|\bjump|AddForce|rigidbody|move_and_slide|is_on_floor|on_ground|grounded|friction|bounce|impulse|\bvy\s*[+-]=|physics`),
	gc("collision", "Collision detection", "Checking whether two things touch. The simplest test compares rectangles (axis-aligned bounding boxes, AABB): they overlap if they overlap on both x and y.", `collid|overlap|intersect|place_meeting|OnTrigger|OnCollision|Touched|body_entered|area_entered|CheckCollision|colliderect|hitTest|\baabb\b|\bfget\s*\(|\bmget\s*\(|PEEK\s*\(\s*5327[89]|\bATTR\s*\(|\bPOINT\s*\(|Raycast`),
	gc("spawning", "Spawning and removing", "Creating new things while the game runs (bullets, enemies, particles) and removing them when they're done, so the game doesn't slow down.", `Instantiate|spawn|instance_create|Instance\.new|add_child|queue_free|Destroy\(|instance_destroy|\.instantiate\(|\.Clone\(\)|table\.insert\(\s*\w*(bullet|enem|particle)|\bdel\(|\.splice\(|\.remove\(|despawn`),
	gc("timers", "Timers and cooldowns", "Counting time so things happen later or not too often: reload times, spawn waves, invincibility after a hit.", `timer|cooldown|WaitForSeconds|create_timer|\bwait\(|task\.wait|Invoke\(|setTimeout|setInterval|\balarm\[|SetTimer|\bTIMER\b|get_ticks|Time\.time`),
	gc("random", "Randomness", "Random numbers make every game different: where enemies appear, how much damage a hit does, which item drops.", `random|\brnd\b|\brand\(|randi|randf|GetRandomValue|irandom|choose\(|\bRND\b`),
	gc("score", "Score, lives and health", "Numbers that track how the player is doing, shown on screen and checked for winning or game over.", `\bscore|\blives\b|\bhealth|\bhp\b|\bammo\b|\bcoins?\b|hiscore|high_score`),
	gc("state-machine", "Game states", "A state machine remembers what the game (or a character) is doing right now: menu, playing, paused, game over; idle, running, jumping. Each state has its own rules.", `\bstate\s*(==|=|:=)|switch\s*\(\s*\w*state|match\s+\w*state|game_?state|current_?state|enum\s+\w*State|STATE_|\bmode\s*==`),
	gc("rendering", "Drawing the frame", "Each frame the screen is cleared and everything is drawn again in its new place, which is what makes things look animated.", `\bdraw|render|\bspr\(|blit|rectfill|fillRect|love\.graphics|\bcls\(|ClearBackground|\.fill\(|clearRect|\bPSET\b|\bPLOT\b|\bHPLOT\b|\bCLS\b|_DISPLAY|display\.flip`),
	gc("sound", "Sound and music", "Sound effects give feedback (a hit, a jump, a coin) and music sets the mood.", `\bsfx\(|\bsound|\bmusic|PlayOneShot|audio|\bBEEP\b|\bPLAY\b|POKE\s*54\d{3}`),
	gc("animation", "Animation", "Animation flips between pictures (frames) over time, like a flip-book, or smoothly changes a value.", `anim|\bframe\s*[+=]|image_index|sprite_index|newQuad|SetTrigger|AnimatedSprite|\.play\(\s*\S`),
	gc("camera", "Camera and scrolling", "A camera decides which part of the game world is shown, so the world can be bigger than the screen.", `camera|viewport|scroll`),
	gc("saving", "Saving progress", "Writing the score or progress somewhere that lasts after the game closes.", `PlayerPrefs|localStorage|DataStore|save_?game|SetAsync|json\.dump|FileAccess`),
	gc("hardware", "Talking to the hardware", "Old machines had no game engine: games read the joystick and drove the graphics and sound chips by reading and writing memory addresses.", `\bPOKE\b|\bPEEK\b|\bSYS\s*\d|\bVDU\b`),
}

var tagToGame = map[string]string{"game-loop": "game-loop", "delta-time": "delta-time", "input": "input", "collision": "collision", "physics": "physics", "spawning": "spawning", "timers": "timers", "random": "random", "rendering": "rendering", "sound": "sound", "animation": "animation", "camera": "camera", "saving": "saving", "game-state": "state-machine", "state-machine": "state-machine", "movement": "movement", "hardware": "hardware", "callback": "game-loop"}

func analyseGame(det Detection, code string, lines []LineOut, outline []Outline) GameReport {
	rep := GameReport{Hits: []GameHit{}, Lessons: []string{}}
	for _, e := range engines {
		if !contains(e.langs, det.ID) {
			continue
		}
		if e.ID == "gamemaker" || e.sig.MatchString(code) {
			c := *e
			c.Frame = nil
			for _, f := range e.Frame {
				f.Line = findCallback(f.Name, lines)
				c.Frame = append(c.Frame, f)
			}
			rep.Engine = &c
			break
		}
	}
	hits := map[string]*GameHit{}
	order := []string{}
	add := func(id string, n int) {
		h := hits[id]
		if h == nil {
			for _, g := range gameConcepts {
				if g.id == id {
					h = &GameHit{ID: id, Title: g.title, Text: g.text}
				}
			}
			if h == nil {
				return
			}
			hits[id] = h
			order = append(order, id)
		}
		if len(h.Lines) == 0 || h.Lines[len(h.Lines)-1] != n {
			h.Lines = append(h.Lines, n)
		}
	}
	for i := range lines {
		ln := &lines[i]
		if ln.Kind == "comment" || ln.Kind == "string" {
			continue
		}
		var ids []string
		text := ln.Text
		for _, g := range gameConcepts {
			if g.re.MatchString(text) {
				ids = append(ids, g.id)
			}
		}
		for _, t := range ln.Tags {
			if g, ok := tagToGame[t]; ok {
				ids = append(ids, g)
			}
		}
		ids = uniq(ids)
		// "rendering" and "score" alone are common words in non-game code
		for _, id := range ids {
			add(id, ln.N)
		}
		ln.Game = ids
	}
	strong := 0
	for _, id := range order {
		switch id {
		case "game-loop", "delta-time", "input", "collision", "physics", "spawning":
			strong++
		}
	}
	rep.IsGame = rep.Engine != nil && rep.Engine.ID != "arduino" || strong >= 2
	if !rep.IsGame {
		// Not game code: keep only the stronger signals, and don't mark lines.
		for i := range lines {
			lines[i].Game = nil
		}
	}
	for _, id := range order {
		rep.Hits = append(rep.Hits, *hits[id])
	}
	sort.SliceStable(rep.Hits, func(i, j int) bool { return rep.Hits[i].Lines[0] < rep.Hits[j].Lines[0] })
	seen := map[string]bool{}
	for _, h := range rep.Hits {
		for _, l := range lessons {
			if l.Concept == h.ID && !seen[l.ID] {
				seen[l.ID] = true
				rep.Lessons = append(rep.Lessons, l.ID)
			}
		}
	}
	if rep.IsGame && !seen["game-loop"] {
		rep.Lessons = append([]string{"game-loop"}, rep.Lessons...)
	}
	return rep
}

func findCallback(name string, lines []LineOut) int {
	short := name
	if i := strings.IndexAny(short, "( "); i > 0 {
		short = short[:i]
	}
	re := regexp.MustCompile(`(?i)(func|function|def|void|override|sub|proc|fn)\s+` + regexp.QuoteMeta(short) + `\b|` + regexp.QuoteMeta(short) + `\s*[:.]\s*Connect|\b` + regexp.QuoteMeta(short) + `\b`)
	best := 0
	for _, ln := range lines {
		if ln.Kind == "comment" {
			continue
		}
		if m := re.FindStringSubmatch(ln.Text); m != nil {
			if m[1] != "" {
				return ln.N // a definition beats a mention
			}
			if best == 0 {
				best = ln.N
			}
		}
	}
	return best
}

func contains(a []string, s string) bool {
	for _, x := range a {
		if x == s {
			return true
		}
	}
	return false
}

func gameTitles(h []GameHit, n int) []string {
	var o []string
	for _, x := range h {
		if len(o) == n {
			break
		}
		o = append(o, strings.ToLower(x.Title))
	}
	return o
}
