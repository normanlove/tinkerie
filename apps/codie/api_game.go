// Codie — api_game.go: game engines and libraries, BASIC keywords and the
// commands used in shell scripts.
// Format: languages | names | kind | what | why | tags  (see api.go)
// Part of the free nrmluvdev maker suite. Created by nrmluvdev.
package codie

var _ = addAPI(`
# ---------------------------------------------------------------- Unity (C#)
csharp | Debug.Log | print | Prints a message in Unity's Console window. | The quickest way to check a script is running and see values. | output
csharp | Time.deltaTime | prop | The seconds since the last frame (about 0.016 at 60 FPS). | Multiplying movement by Time.deltaTime makes it 'per second', so the game runs at the same speed whatever the frame rate. | delta-time
csharp | Time.fixedDeltaTime | prop | The time between physics steps (0.02 s by default). | | delta-time physics
csharp | Time.time | prop | The seconds since the game started. | Used for cooldowns: if (Time.time > nextFire) … | timers
csharp | Time.timeScale | prop | How fast game time runs: 1 normal, 0 paused, 0.5 slow motion. | Setting it to 0 is a simple pause. | timers
csharp | Input.GetKey | input | Answers true every frame a key is held down. | For continuous actions like walking. | input
csharp | Input.GetKeyDown | input | Answers true only on the frame the key goes down. | For one-off actions like jumping or shooting. | input
csharp | Input.GetKeyUp | input | Answers true on the frame the key is let go. | | input
csharp | Input.GetAxis | input | Reads a smoothed value from -1 to 1 for "Horizontal" (A/D, arrows, stick) or "Vertical". | One line handles keyboard and gamepad. GetAxisRaw gives -1, 0 or 1 without smoothing. | input
csharp | Input.GetAxisRaw | input | Reads -1, 0 or 1 for an input axis, with no smoothing. | Snappy controls for platformers. | input
csharp | Input.GetButtonDown,Input.GetButton | input | Reads a named button such as "Jump" or "Fire1" from the Input Manager. | | input
csharp | Input.GetMouseButtonDown,Input.mousePosition | input | Reads mouse clicks or the pointer's position. | | input
csharp | GetComponent | call | Finds another component on the same GameObject, such as its Rigidbody or Animator. | Call it once (in Start or Awake) and keep the result: it's slow to call every frame. | game-engine
csharp | Instantiate | call | Makes a copy of a prefab (or object) in the scene, such as a bullet or an enemy. | This is how games spawn things at run time. | spawning
csharp | Destroy | call | Removes a GameObject (or component) from the scene, optionally after a delay. | Used for dead enemies, collected coins, bullets that hit. | spawning
csharp | DontDestroyOnLoad | call | Keeps this object alive when a new scene loads. | Used for music players and game managers. | game-engine
csharp | transform.position,.position | prop | Where the object is in the world. | | movement
csharp | transform.Translate,.Translate | call | Moves the object by an amount. | Multiply by Time.deltaTime for smooth movement. | movement
csharp | transform.Rotate,.Rotate | call | Turns the object. | | movement
csharp | .LookAt | call | Turns the object to face a point. | | movement
csharp | .AddForce | call | Pushes a Rigidbody with a force; the physics engine moves it. | Use it in FixedUpdate. ForceMode.Impulse gives a sudden kick, good for jumps. | physics
csharp | .velocity,.linearVelocity | prop | A Rigidbody's current speed and direction. | Setting it directly gives instant, precise control. | physics movement
csharp | .MovePosition | call | Moves a Rigidbody smoothly to a new position, respecting physics. | | physics movement
csharp | Physics.Raycast,Physics2D.Raycast | call | Fires an invisible line and reports whether (and what) it hits. | Used for ground checks, line of sight and hitscan weapons. | physics collision
csharp | Physics2D.OverlapCircle,Physics.OverlapSphere | call | Lists the colliders inside a circle or sphere. | Common ground check: is anything solid under the feet? | collision
csharp | Vector3.zero,Vector3.up,Vector3.forward,Vector2.zero,Vector2.up | prop | A ready-made direction or zero vector. | | movement
csharp | Vector3.Distance,Vector2.Distance | call | The distance between two points. | Used for 'is the enemy close enough to attack?' | maths
csharp | Vector3.Lerp,Mathf.Lerp | call | Moves part of the way from one value to another. | Called every frame it gives smooth following and easing. | maths
csharp | .normalized | prop | The same direction with a length of 1. | So diagonal movement isn't faster than straight movement. | maths movement
csharp | Quaternion.Euler | call | Makes a rotation from angles in degrees. | | maths
csharp | Mathf.Clamp | call | Keeps a number between a minimum and a maximum. | Stops health going over max or the player leaving the screen. | maths
csharp | Mathf.Abs,Mathf.Max,Mathf.Min,Mathf.Sin,Mathf.Sqrt | call | A maths helper. | | maths
csharp | Random.Range | call | Gives a random number between two values (for whole numbers, the top number is excluded). | Random spawn points, damage and loot. | random
csharp | StartCoroutine | call | Starts a coroutine: a function that can pause (yield) and continue later. | Perfect for 'wait 2 seconds, then spawn' sequences. | timers
csharp | Invoke | call | Calls a method by name after a delay. | | timers
csharp | InvokeRepeating | call | Calls a method by name over and over at an interval. | A simple enemy spawner. | timers spawning
csharp | SceneManager.LoadScene | call | Loads another scene (level or menu). | | game-state
csharp | PlayerPrefs.SetInt,PlayerPrefs.SetFloat,PlayerPrefs.SetString | call | Saves a small value on the player's computer. | Used for high scores and settings. | saving
csharp | PlayerPrefs.GetInt,PlayerPrefs.GetFloat,PlayerPrefs.GetString | call | Loads a saved value (or a default). | | saving
csharp | .PlayOneShot | call | Plays a sound effect once, even if another is still playing. | | sound
csharp | .SetActive | call | Shows or hides (switches on or off) a GameObject. | | game-engine
csharp | .CompareTag | call | Answers whether the object has a tag, such as "Player" or "Enemy". | Faster and safer than comparing tag strings with ==. | collision
csharp | .SetTrigger,.SetBool,.SetFloat | call | Sends a value to the Animator, which chooses the animation to play. | | animation
csharp | Camera.main | prop | The scene's main camera. | | camera
csharp | FindObjectOfType,GameObject.Find | call | Searches the scene for an object. | Slow: do it once, not every frame. | game-engine
# MonoGame / XNA
csharp | Keyboard.GetState | input | Takes a snapshot of which keys are down right now. | Compare with last frame's snapshot to spot new presses. | input
csharp | .IsKeyDown | input | Answers whether a key was down in that snapshot. | | input
csharp | GamePad.GetState | input | Reads a controller's buttons and sticks. | | input
csharp | Content.Load | call | Loads a picture, sound or font that was added to the Content project. | | game-engine
csharp | GraphicsDevice.Clear | call | Fills the screen with one colour before drawing. | | rendering
csharp | .ElapsedGameTime,gameTime.ElapsedGameTime.TotalSeconds | prop | The time since the last update. | MonoGame's delta time. | delta-time
csharp | _spriteBatch.Begin,spriteBatch.Begin,_spriteBatch.End,spriteBatch.End | call | Starts (or finishes) a batch of sprite drawing. | Drawing in one batch is much faster. | rendering
csharp | _spriteBatch.Draw,spriteBatch.Draw | call | Draws a picture at a position. | | rendering
# Godot C#
csharp | GD.Print | print | Prints to Godot's Output panel. | | output
csharp | GetNode | call | Finds a child node by its path. | | game-engine
csharp | MoveAndSlide | call | Moves the character by its Velocity, sliding along walls and floors. | | physics collision
csharp | Input.IsActionPressed,Input.IsActionJustPressed | input | Reads an action from Godot's Input Map. | | input
csharp | QueueFree | call | Removes the node at the end of the frame. | | spawning

# ---------------------------------------------------------------- Unreal (C++)
cpp | UE_LOG | print | Writes a message to Unreal's Output Log. | | output
cpp | GEngine->AddOnScreenDebugMessage,AddOnScreenDebugMessage | print | Shows a message on the game screen for a few seconds. | | output
cpp | GetWorld | call | Gets the world (level) this actor lives in. | Needed to spawn actors, set timers and trace. | game-engine
cpp | SpawnActor | call | Creates a new actor in the level (a bullet, enemy, pickup). | | spawning
cpp | Destroy | call | Removes this actor from the level. | | spawning
cpp | SetActorLocation,GetActorLocation | call | Sets (or reads) where the actor is. | | movement
cpp | AddActorLocalOffset,AddActorWorldOffset | call | Moves the actor by an amount. | Multiply by DeltaTime for smooth movement. | movement
cpp | AddMovementInput | input | Asks the character's movement component to walk in a direction. | | movement input
cpp | BindAxis,BindAction | input | Connects an input (axis or button) to one of this class's functions. | | input
cpp | Jump | call | Makes an ACharacter jump. | | physics
cpp | GetWorldTimerManager,SetTimer | call | Runs a function after a delay, or repeatedly. | | timers
cpp | PrimaryActorTick.bCanEverTick | prop | Turns this actor's Tick on or off. | Turning Tick off for actors that don't need it saves time. | game-loop
cpp | FVector | call | A 3D vector: a position or direction with X, Y and Z. | | maths
cpp | FMath::RandRange | call | A random number between two values. | | random
cpp | UGameplayStatics::PlaySoundAtLocation,UGameplayStatics::PlaySound2D | call | Plays a sound. | | sound
cpp | UGameplayStatics::OpenLevel | call | Loads another level. | | game-state
cpp | CreateDefaultSubobject | call | Creates a component (mesh, camera, collision) inside this actor's constructor. | | game-engine

# ---------------------------------------------------------------- raylib (C, C++, Go via rl.)
c,cpp,go | InitWindow | call | Opens the game window with a size and title. | | rendering game-engine
c,cpp,go | SetTargetFPS | call | Limits the game to a number of frames per second. | Keeps the game loop steady (usually 60). | game-loop timing
c,cpp,go | WindowShouldClose | call | Answers true when the player closes the window or presses Esc. | Used as the condition of the main game loop. | game-loop
c,cpp,go | BeginDrawing,EndDrawing | call | Starts (or finishes) drawing this frame. EndDrawing shows the frame and waits for the next. | Everything between them is one frame. | rendering game-loop
c,cpp,go | ClearBackground | call | Fills the whole screen with a colour. | Done at the start of each frame. | rendering
c,cpp,go | DrawRectangle,DrawRectangleRec,DrawCircle,DrawLine,DrawText,DrawTexture,DrawTextureRec | call | Draws a shape, text or picture. | | rendering
c,cpp,go | IsKeyDown | input | Answers whether a key is held down. | | input
c,cpp,go | IsKeyPressed | input | Answers true only on the frame a key went down. | | input
c,cpp,go | GetFrameTime | call | Gives the seconds the last frame took (delta time). | Multiply speeds by it for smooth movement. | delta-time
c,cpp,go | CheckCollisionRecs,CheckCollisionCircles,CheckCollisionPointRec | call | Answers whether two shapes overlap. | raylib's ready-made collision tests. | collision
c,cpp,go | LoadTexture,LoadSound,UnloadTexture | call | Loads (or frees) a picture or sound file. | | game-engine
c,cpp,go | PlaySound | call | Plays a sound. | | sound
c,cpp,go | GetRandomValue | call | A random whole number between two values (both included). | | random
c,cpp,go | CloseWindow | call | Closes the window and tidies up. | | game-engine
# SDL
c,cpp | SDL_Init | call | Starts SDL's parts (video, audio, input). | | game-engine
c,cpp | SDL_CreateWindow,SDL_CreateRenderer | call | Creates the game window (or its renderer). | | rendering
c,cpp | SDL_PollEvent | input | Takes the next event (key, mouse, quit) from the queue; false when there are no more. | Loop it every frame so the window stays responsive. | input game-loop
c,cpp | SDL_RenderClear,SDL_RenderPresent | call | Clears the screen (or shows the finished frame). | | rendering
c,cpp | SDL_RenderFillRect,SDL_RenderCopy | call | Draws a rectangle or picture. | | rendering
c,cpp | SDL_GetTicks,SDL_GetTicks64 | call | Milliseconds since SDL started. | Used to work out delta time. | timers delta-time
c,cpp | SDL_Delay | call | Waits for some milliseconds. | Crude frame-rate limiting. | timers
c,cpp | SDL_GetKeyboardState | input | Gives which keys are held down right now. | | input
c,cpp | SDL_HasIntersection | call | Answers whether two rectangles overlap. | | collision
c,cpp | SDL_Quit | call | Shuts SDL down. | | game-engine

# ---------------------------------------------------------------- Ebitengine (Go)
go | ebiten.RunGame | call | Starts the game: Ebitengine then calls the game's Update (60 times a second) and Draw (every frame) until it ends. | Ebitengine runs the game loop for you. | game-loop game-engine
go | ebiten.SetWindowSize,ebiten.SetWindowTitle | call | Sets the window's size or title. | | game-engine
go | ebiten.IsKeyPressed | input | Answers whether a key is held down. | | input
go | inpututil.IsKeyJustPressed | input | Answers true only on the tick a key went down. | | input
go | ebitenutil.DebugPrint | print | Draws debug text on the screen. | | output
go | ebiten.NewImage | call | Makes a new blank image to draw on. | | rendering
go | .DrawImage | call | Draws one image on another, using options for position, scale and rotation. | | rendering
go | .Fill | call | Fills the whole image with one colour. | | rendering
go | ebiten.ActualTPS,ebiten.ActualFPS | call | The real updates (or frames) per second. | | timing

# ---------------------------------------------------------------- Rust (Bevy)
rust | App::new | call | Starts building a Bevy app. | Plugins and systems are added to it, then .run() starts the game loop. | game-engine
rust | .add_systems | call | Adds systems (functions Bevy runs) to a schedule such as Startup or Update. | Update systems run every frame. | game-loop
rust | .run | call | Starts the app's loop. | |
rust | commands.spawn,.spawn | call | Creates a new entity with components. | | spawning
rust | .despawn | call | Removes an entity. | | spawning
rust | time.delta_seconds,time.delta_secs,.delta_seconds,.delta_secs | call | Seconds since the last frame (delta time). | | delta-time
rust | keyboard_input.pressed,.pressed,.just_pressed | input | Answers whether a key is held (or just pressed). | | input
rust | Transform::from_xyz | call | A position at x, y, z. | | movement
rust | Vec::new,vec | call | Makes a new, growable list. | | list
rust | String::new,String::from | call | Makes text that can grow. | | text
rust | .unwrap | call | Takes the value out of an Option or Result, crashing if there isn't one. | Fine in quick code; real programs handle the error instead. | errors
rust | .expect | call | Like unwrap, but with your own crash message. | | errors
rust | .iter | call | Goes through the items one by one. | | loop
rust | .clone | call | Makes a full copy. | | memory
rust | std::io::stdin,stdin | input | The keyboard input. | | input
rust | .read_line | input | Reads a line typed by the user. | | input

# ---------------------------------------------------------------- JavaScript game libraries (Phaser)
javascript,typescript | this.physics.add.sprite | call | Adds a sprite with a physics body (Phaser). | | spawning physics
javascript,typescript | this.physics.add.collider | call | Makes two objects (or groups) bump into each other (Phaser). | | collision
javascript,typescript | this.physics.add.overlap | call | Runs a function when two objects overlap, without bouncing (Phaser). | Used for collecting coins and taking damage. | collision
javascript,typescript | this.input.keyboard.createCursorKeys | input | Makes an object for the arrow keys and space bar (Phaser). | | input
javascript,typescript | this.load.image,this.load.spritesheet | call | Loads a picture (or a sprite sheet) in preload (Phaser). | | game-engine
javascript,typescript | this.add.text,this.add.image,this.add.sprite | call | Adds text, a picture or a sprite to the scene (Phaser). | | rendering
javascript,typescript | .setVelocityX,.setVelocityY,.setVelocity | call | Sets the physics body's speed (Phaser). | | physics movement
javascript,typescript | .setBounce,.setCollideWorldBounds | call | Makes the body bounce, or stops it leaving the screen (Phaser). | | physics

# ---------------------------------------------------------------- Swift (SpriteKit)
swift | SKSpriteNode | call | Makes a sprite (a picture node). | | rendering
swift | addChild | call | Adds a node to the scene so it appears. | | spawning
swift | SKAction.move,SKAction.moveBy,SKAction.wait,SKAction.sequence,SKAction.repeatForever | call | Makes an action that moves, waits or repeats a node over time. | Actions animate without writing per-frame code. | animation
swift | .removeFromParent | call | Removes the node from the scene. | | spawning

# ---------------------------------------------------------------- pygame (Python)
python | pygame.init | call | Starts all of pygame's parts (display, sound, input). | Call it once before anything else in pygame. | game-engine
python | pygame.display.set_mode | call | Opens the game window with the size given, and gives back the screen surface to draw on. | | rendering game-engine
python | pygame.display.set_caption | call | Sets the window's title. | |
python | pygame.time.Clock | call | Makes a clock for controlling the frame rate. | | timing
python | .tick | call | Waits so the loop runs at most this many times a second, and gives back the milliseconds since the last tick. | clock.tick(60) caps the game at 60 FPS; dividing its result by 1000 gives delta time in seconds. | game-loop timing delta-time
python | pygame.event.get | input | Takes every event (keys, clicks, closing the window) that happened since last time. | Every pygame loop must call it, or the window freezes. | input game-loop
python | pygame.key.get_pressed | input | Gives which keys are held down right now. | For smooth movement while a key is held. | input
python | pygame.mouse.get_pos | input | Gives the mouse position. | | input
python | pygame.QUIT | prop | The event sent when the player closes the window. | | input
python | pygame.KEYDOWN | prop | The event sent when a key is pressed. | | input
python | pygame.K_LEFT,pygame.K_RIGHT,pygame.K_UP,pygame.K_DOWN,pygame.K_SPACE,pygame.K_ESCAPE | prop | A key code. | | input
python | .fill | call | Fills the whole surface (screen) with one colour. | Done at the start of each frame to clear the last one. | rendering
python | .blit | call | Draws one picture (surface) onto another at a position. | How sprites and text get onto the screen. | rendering
python | pygame.display.flip,pygame.display.update | call | Shows the finished frame on the screen. | Everything is drawn on a hidden buffer first, then shown at once, so there's no flicker. | rendering game-loop
python | pygame.draw.rect,pygame.draw.circle,pygame.draw.line,pygame.draw.polygon | call | Draws a shape on a surface. | | rendering
python | pygame.Rect | call | Makes a rectangle (x, y, width, height). | Rects are used for positions and collision checks. | collision
python | .colliderect | call | Answers whether two rectangles overlap. | The simplest collision test (AABB). | collision
python | .collidepoint | call | Answers whether a point (like the mouse) is inside the rectangle. | | collision
python | pygame.sprite.spritecollide,pygame.sprite.groupcollide | call | Finds the sprites that touch a sprite (or another group). | | collision
python | pygame.image.load | call | Loads a picture file. | Add .convert() or .convert_alpha() so it draws fast. | game-engine
python | pygame.mixer.Sound | call | Loads a sound effect. | | sound
python | pygame.mixer.music.load,pygame.mixer.music.play | call | Loads or plays background music. | | sound
python | pygame.font.Font,pygame.font.SysFont | call | Loads a font for drawing text. | | rendering
python | .render | call | Turns text into a picture that can be blitted. | | rendering
python | pygame.quit | call | Shuts pygame down and closes the window. | | game-engine
python | pygame.time.get_ticks | call | Milliseconds since pygame started. | Used for cooldowns and timers. | timers

# ---------------------------------------------------------------- Godot (GDScript)
gdscript | get_node | call | Finds a node by its path in the scene. | $Path is a shortcut for this. | game-engine
gdscript | Input.is_action_pressed | input | Answers whether an action from the Input Map (such as "ui_left" or "jump") is held. | Actions let players remap keys and use gamepads. | input
gdscript | Input.is_action_just_pressed | input | Answers true only on the frame the action was pressed. | For jumping, shooting, opening menus. | input
gdscript | Input.is_action_just_released | input | Answers true on the frame the action was let go. | Variable jump height: cut the jump short when released. | input
gdscript | Input.get_vector | input | Reads four actions as one direction (a Vector2 of length up to 1). | Handles diagonals and gamepad sticks in one line. | input movement
gdscript | Input.get_axis | input | Reads two actions as one number from -1 to 1. | | input
gdscript | move_and_slide | call | Moves the CharacterBody by its velocity, sliding along walls and floors instead of stopping dead. | Godot's main character-movement call; afterwards is_on_floor() tells you if you landed. | physics collision
gdscript | move_and_collide | call | Moves the body and stops at the first thing it hits, giving back the collision. | For bouncing balls and bullets. | physics collision
gdscript | is_on_floor | call | Answers whether the character is standing on the floor (after move_and_slide). | Used to allow jumping only when grounded. | physics
gdscript | queue_free | call | Removes this node (and its children) at the end of the frame. | Used for collected coins, dead enemies, finished bullets. | spawning
gdscript | add_child | call | Adds a node to the scene tree under this one. | Instanced scenes only appear once they're added. | spawning
gdscript | preload,load | call | Loads a scene or resource (preload does it when the script loads). | | game-engine
gdscript | .instantiate,.instance | call | Makes a new copy of a loaded scene (instance in Godot 3). | Then add_child() it to make it appear: this is spawning. | spawning
gdscript | emit_signal,.emit | call | Sends out a signal so any connected functions run. | | signals
gdscript | .connect,connect | call | Connects a signal to a function. | | signals
gdscript | get_tree | call | Gets the scene tree, which runs the game. | | game-engine
gdscript | .change_scene_to_file,.change_scene,.reload_current_scene | call | Switches to another scene, or restarts this one. | Level changes and game over. | game-state
gdscript | .create_timer | call | Makes a one-shot timer; await its timeout signal to wait. | | timers
gdscript | randf,randi,randf_range,randi_range,randomize | call | Random numbers (randomize() seeds the generator in Godot 3). | | random
gdscript | lerp,clamp,move_toward | call | Maths helpers: blend towards, keep within limits, step towards a value. | move_toward is great for acceleration and friction. | maths
gdscript | Vector2,Vector3 | call | A 2D (or 3D) vector: a position, direction or speed. | | maths
gdscript | .normalized | call | The same direction with length 1. | | maths
gdscript | .play | call | Plays an animation or sound. | | animation sound
gdscript | look_at | call | Turns the node to face a point. | | movement
gdscript | position,global_position | prop | Where the node is (relative to its parent, or in the world). | | movement
gdscript | velocity | prop | The character's speed and direction, used by move_and_slide. | | physics movement
gdscript | get_viewport | call | Gets the viewport (the screen or window area). | | camera

# ---------------------------------------------------------------- LÖVE / Lua
lua | print | print | Prints values to the console. | | output
lua | pairs,ipairs | call | Gives the entries of a table to loop over. | | loop table
lua | tostring,tonumber | call | Turns a value into text (or text into a number). | | text
lua | table.insert | call | Adds an item to a list table (at the end, or at a position). | Common for adding bullets or enemies. | list table
lua | table.remove | call | Removes an item from a list table. | Remove from the end backwards when looping, or items get skipped. | list table
lua | math.random | call | Gives a random number: math.random(6) is 1 to 6. | | random
lua | math.randomseed | call | Seeds the random numbers (often with os.time()). | | random
lua | math.floor,math.abs,math.max,math.min,math.sqrt,math.sin,math.cos,math.atan2,math.pi | call | A maths helper. | | maths
lua | string.format | call | Builds text from a template (%d, %s, %.2f). | | text
lua | string.sub,string.len,string.upper | call | Text helpers. | | text
lua | require | call | Loads another Lua file (module) and gives back what it returns. | | import
lua | os.time,os.clock | call | Reads the clock. | | timers
lua | setmetatable | call | Gives a table a metatable, which is how Lua does classes and inheritance. | | class
lua | love.graphics.rectangle | call | Draws a rectangle ("fill" or "line"). | | rendering
lua | love.graphics.circle | call | Draws a circle. | | rendering
lua | love.graphics.print,love.graphics.printf | call | Draws text on the screen. | | rendering
lua | love.graphics.setColor | call | Chooses the colour for the next things drawn (red, green, blue from 0 to 1). | | rendering
lua | love.graphics.draw | call | Draws a picture (or quad of a sprite sheet) at a position. | | rendering
lua | love.graphics.newImage | call | Loads a picture file. | Do it in love.load, not every frame. | game-engine
lua | love.graphics.newQuad | call | Cuts a frame out of a sprite sheet. | Used for animation. | animation
lua | love.graphics.getWidth,love.graphics.getHeight | call | The window's size. | | rendering
lua | love.keyboard.isDown | input | Answers whether a key is held down. | For smooth movement in love.update. | input
lua | love.mouse.getPosition,love.mouse.isDown | input | Reads the mouse. | | input
lua | love.audio.newSource | call | Loads a sound ("static" for effects, "stream" for music). | | sound
lua | love.audio.play,.play | call | Plays a sound. | | sound
lua | love.window.setMode,love.window.setTitle | call | Sets the window size or title. | | game-engine
lua | love.timer.getTime,love.timer.getFPS | call | Reads the clock or frames per second. | | timers
lua | love.math.random | call | A random number (better than math.random in LÖVE). | | random
lua | love.event.quit | call | Closes the game. | | game-engine
lua | love.physics.newWorld | call | Creates a Box2D physics world with gravity. | | physics
# PICO-8
~PICO-8 | cls | call | Clears the screen (to a colour, black by default). | Usually the first line of _draw. | rendering
~PICO-8 | spr | call | Draws sprite number n from the sprite sheet at x, y. | The 8×8 sprites are drawn in PICO-8's sprite editor. | rendering
~PICO-8 | sspr | call | Draws (and can stretch) any rectangle of the sprite sheet. | | rendering
~PICO-8 | map | call | Draws part of the map made in the map editor. | | rendering
~PICO-8 | mget | call | Reads which tile is at a map cell. | Used for wall collisions: is the tile I'm moving into solid? | collision
~PICO-8 | mset | call | Changes a map tile. | Collect a coin by replacing its tile. |
~PICO-8 | fget | call | Reads a sprite's flags (set in the sprite editor). | Mark solid tiles with a flag, then fget(mget(x,y),0) checks for walls. | collision
~PICO-8 | btn | input | Answers whether a button is held: 0 left, 1 right, 2 up, 3 down, 4 O (Z key), 5 X (X key). | For movement. | input
~PICO-8 | btnp | input | Answers true when a button was just pressed (and repeats if held). | For jumping, shooting and menus. | input
~PICO-8 | sfx | call | Plays sound effect n made in the sound editor. | | sound
~PICO-8 | music | call | Plays music starting at pattern n (-1 stops it). | | sound
~PICO-8 | print | print | Draws text on the screen (with optional x, y and colour). | | rendering
~PICO-8 | rect,rectfill,circ,circfill,line,pset,oval,ovalfill | call | Draws a shape or pixel. | | rendering
~PICO-8 | pget | call | Reads a pixel's colour. | | collision
~PICO-8 | rnd | call | A random decimal from 0 up to n. | flr(rnd(6)) gives 0 to 5. | random
~PICO-8 | flr,ceil,abs,min,max,sgn,sqrt | call | A maths helper. | | maths
~PICO-8 | mid | call | Gives the middle of three numbers, so mid(0, x, 120) keeps x between 0 and 120. | Handy for keeping the player on screen. | maths
~PICO-8 | sin,cos,atan2 | call | Trigonometry, PICO-8 style: angles are in turns (0 to 1), and sin is upside down compared with maths class. | | maths
~PICO-8 | add | call | Adds an item to the end of a table. | | list
~PICO-8 | del,deli | call | Removes an item from a table. | | list
~PICO-8 | all,foreach | call | Goes through every item in a table. | | loop
~PICO-8 | camera | call | Shifts everything drawn by x, y: a scrolling camera. | | camera
~PICO-8 | pal,palt | call | Swaps colours, or sets transparency. | Used for flashes when hit. | rendering
~PICO-8 | t,time | call | Seconds since the cartridge started. | | timers
# Roblox Luau
~Roblox | game:GetService,game.GetService | call | Gets one of Roblox's services, such as Players, RunService or UserInputService. | Services are Roblox's built-in systems. | game-engine
~Roblox | Instance.new | call | Creates a new Roblox object, such as a Part or a Sound. | Set its Parent to make it appear in the game. | spawning
~Roblox | task.wait,wait | call | Pauses this script for a number of seconds (or one frame). | Without a wait inside a while true loop, the game freezes. | timers
~Roblox | task.spawn,task.delay | call | Runs a function separately (now, or after a delay). | | timers concurrency
~Roblox | .FindFirstChild | call | Looks for a child by name, giving nil if it isn't there. | | game-engine
~Roblox | .WaitForChild | call | Waits until a child with this name exists, then gives it back. | Important on the client, where parts load a moment later. | game-engine
~Roblox | .Destroy | call | Removes the object for good. | | spawning
~Roblox | .Clone | call | Makes a copy of the object. | Clone a template, then set its Parent: spawning. | spawning
~Roblox | .FireServer,.FireClient,.FireAllClients | call | Sends a message through a RemoteEvent between a player's computer and the server. | Clients can't change the game for everyone, so they ask the server. | network
~Roblox | .GetPlayerFromCharacter | call | Finds which player a character model belongs to. | Used in Touched to see if a player hit the part. | collision
~Roblox | Vector3.new,CFrame.new | call | Makes a 3D position (or position and rotation). | | maths movement
~Roblox | TweenService:Create,.Create | call | Makes a tween: a smooth animation of properties. | Doors that slide, platforms that move. | animation
~Roblox | :GetDataStore,.GetDataStore,.GetAsync,.SetAsync | call | Loads or saves player data that lasts between visits. | | saving
~Roblox | .IsKeyDown | input | Answers whether a key is held (UserInputService). | | input

# ---------------------------------------------------------------- GameMaker (GML)
gml | keyboard_check | input | Answers whether a key is held (for example vk_left or ord("A")). | | input
gml | keyboard_check_pressed | input | Answers true only on the step the key went down. | | input
gml | place_meeting | call | Answers whether this instance would touch an object if it were at x, y. | The classic GameMaker collision check: test before moving. | collision
gml | instance_create_layer,instance_create_depth | call | Creates a new instance of an object (spawning). | | spawning
gml | instance_destroy | call | Destroys this (or another) instance. | | spawning
gml | instance_exists | call | Answers whether any instance of an object exists. | | game-engine
gml | draw_self | call | Draws the instance's own sprite as normal. | Needed when you add a Draw event. | rendering
gml | draw_sprite,draw_sprite_ext,draw_text,draw_rectangle | call | Draws a sprite, text or shape. | | rendering
gml | irandom,irandom_range,random,random_range,choose | call | Random numbers (choose picks one of its values). | | random
gml | room_goto,room_restart,room_goto_next | call | Changes to another room (level), or restarts this one. | | game-state
gml | audio_play_sound | call | Plays a sound. | | sound
gml | point_direction,point_distance | call | The angle (or distance) from one point to another. | | maths
gml | lengthdir_x,lengthdir_y | call | How far to move in x (or y) for a distance in a direction. | Move towards an angle without trigonometry. | maths movement
gml | sign,clamp,lerp,abs | call | A maths helper. | | maths
gml | show_debug_message | print | Prints a message to the output window. | | output
gml | alarm | prop | One of an instance's 12 countdown timers; when it reaches 0 its Alarm event runs. | | timers
gml | sprite_index,image_index,image_speed,image_xscale | prop | The instance's sprite, animation frame, animation speed or horizontal scale (-1 flips it). | | animation
`)

var _ = addAPI(`
# ---------------------------------------------------------------- BASIC keywords (statements and functions)
@basic | CLS | call | Clears the screen. | Games clear and redraw the screen to animate. | rendering
@basic | RND | prop | RND gives a random number between 0 and 1; INT(RND(1)*6)+1 gives a dice roll from 1 to 6. | | random
@basic | INT | prop | INT rounds a number down to a whole number. | | maths
@basic | ABS,SGN,SQR,SIN,COS,ATN,LOG,EXP | prop | A maths function. | | maths
@basic | LEN | prop | LEN gives the number of characters in a text. | | text
@basic | LEFT$,RIGHT$,MID$ | prop | Cuts a piece out of a text (from the left, right or middle). | | text
@basic | CHR$ | prop | CHR$ turns a character code into a character (CHR$(65) is "A"). | Codes also control the screen: colours, cursor moves, clear screen. | text
@basic | ASC | prop | ASC gives the code of a text's first character. | | text
@basic | STR$,VAL | prop | Turns a number into text (STR$) or text into a number (VAL). | | text
@basic | INKEY$ | prop | INKEY$ gives the key being pressed right now, or "" if none, without waiting. | This is how BASIC games read the keyboard every frame. | input
@basic | INKEY | prop | INKEY(n) waits up to n hundredths of a second for a key; INKEY(-n) checks if a particular key is held (BBC BASIC). | Negative INKEY reads several keys at once, ideal for games. | input
@basic | PEEK | prop | PEEK reads a number straight from a memory address. | On home computers that's how you read the joystick, the keyboard or what's on the screen. | hardware
@basic | TIMER | prop | TIMER gives the seconds since midnight (QBasic). | Used for timing and to seed RANDOMIZE TIMER. | timers
@basic | TI,TI$ | prop | The C64's clock: TI counts sixtieths of a second since switch-on. | | timers
@basic | TAB,SPC | prop | Moves the print position across the line. | | output
@basic | STICK,STRIG | prop | Reads a joystick's direction (STICK) or fire button (STRIG). | | input
@basic | POINT,ATTR,SCREEN$ | prop | Reads what is on the screen at a position: a pixel colour or character. | Simple collision detection: is there a wall where I'm going? | collision
@basic | INSTR | prop | Finds one text inside another. | | text
@basic | UCASE$,LCASE$ | prop | Changes text to upper or lower case. | | text
@basic | COLOR,COLOUR | call | Sets the colours for text and drawing. | | rendering
@basic | LOCATE | call | Moves the text cursor to a row and column. | Lets games print characters anywhere, such as the player's @ sign. | rendering
@basic | PSET,PRESET | call | Sets (or clears) one pixel. | | rendering
@basic | LINE | call | Draws a line, or a box with ,B (filled with ,BF). | | rendering
@basic | CIRCLE | call | Draws a circle. | | rendering
@basic | PAINT | call | Fills an area with colour, like a paint bucket. | | rendering
@basic | DRAW | call | Draws using a mini drawing language, or a line from the last point. | | rendering
@basic | PLOT | call | Plots a point (and on the BBC, lines and shapes, by number). | | rendering
@basic | MOVE | call | Moves the graphics cursor without drawing (BBC BASIC). | | rendering
@basic | GCOL | call | Sets the graphics colour (BBC BASIC). | | rendering
@basic | VDU | call | Sends raw codes to the screen driver (BBC BASIC): change mode, define characters, move the cursor. | | rendering hardware
@basic | CLG | call | Clears the graphics area (BBC BASIC). | | rendering
@basic | BORDER | call | Sets the colour of the border around the screen (ZX Spectrum). | | rendering
@basic | PAPER | call | Sets the background colour of the text (ZX Spectrum). | | rendering
@basic | INK | call | Sets the text and drawing colour (ZX Spectrum). | | rendering
@basic | BRIGHT,FLASH,INVERSE,OVER | call | Changes how text and graphics are drawn (brightness, flashing, swapped colours). | | rendering
@basic | BEEP | call | Plays a beep (on the ZX Spectrum: BEEP length, pitch). | | sound
@basic | SOUND | call | Plays a note (the numbers set channel, volume, pitch and length on the BBC; frequency and length in QBasic). | | sound
@basic | ENVELOPE | call | Shapes how a BBC sound rises and fades. | | sound
@basic | PLAY | call | Plays a tune written as letters, like "CDEFG" (QBasic). | | sound
@basic | SLEEP | call | Pauses for a number of seconds, or until a key is pressed. | | timers
@basic | PAUSE | call | Waits n fiftieths of a second (PAUSE 0 waits for a key) on the ZX Spectrum. | | timers
@basic | WAIT | call | Waits until a memory address has certain bits set (C64). | | hardware
@basic | HGR,HGR2 | call | Switches the Apple II to high-resolution graphics (280×192). | | rendering
@basic | HCOLOR | call | Sets the hi-res drawing colour (Apple II). | | rendering
@basic | HPLOT | call | Plots a point, or a line with TO, in hi-res graphics (Apple II). | | rendering
@basic | GR | call | Switches the Apple II to low-resolution colour graphics. | | rendering
@basic | HOME | call | Clears the text screen (Apple II). | | rendering
@basic | TEXT | call | Switches the Apple II back to text mode. | | rendering
@basic | HTAB,VTAB | call | Moves the cursor across (HTAB) or down (VTAB) (Apple II). | | rendering
@basic | NORMAL | call | Turns inverse and flashing text off (Apple II). | | rendering
@basic | _DISPLAY | call | Shows the finished frame all at once (QB64), so animation doesn't flicker. | | rendering game-loop
@basic | PCOPY | call | Copies one screen page to another (QBasic double-buffering). | Draw on a hidden page, then copy it to the visible one: smooth animation. | rendering
@basic | SWAP | call | Swaps the values of two variables. | |
@basic | CALL | call | Runs a SUB (or machine code). | | function
@basic | OPEN | call | Opens a file (FOR INPUT, OUTPUT or APPEND). | | files
@basic | CLOSE | call | Closes an open file. | | files
@basic | WRITE,PRINT# | call | Writes values into an open file. | Old games saved high scores this way. | files saving
@basic | KEY | call | Sets up function keys or key events. | | input
@basic | BLOAD,BSAVE,LOAD,SAVE | call | Loads or saves a program, picture or memory block. | | files
@basic | Console.WriteLine,Console.Write | print | Prints a line of text in the console (Visual Basic .NET). | | output
@basic | Console.ReadLine | input | Reads a line the user types (Visual Basic .NET). | | input
@basic | MsgBox | call | Shows a message box (Visual Basic). | | output

# ---------------------------------------------------------------- shell commands (Bash)
bash | cd | call | Changes the current folder. | | files
bash | ls | call | Lists the files in a folder. | | files
bash | pwd | call | Prints which folder you're in. | | files
bash | mkdir | call | Makes a folder (-p: and any missing folders above). | | files
bash | rm | call | Deletes files (-r folders too, -f without asking). | There's no recycle bin: deleted is gone. | files
bash | cp | call | Copies files (-r for folders). | | files
bash | mv | call | Moves or renames files. | | files
bash | touch | call | Creates an empty file (or updates its date). | | files
bash | cat | call | Prints a file's contents. | | files
bash | grep | call | Searches for lines containing some text. | | text
bash | sed | call | Edits text automatically, such as find-and-replace. | | text
bash | awk | call | Picks out and processes columns of text. | | text
bash | find | call | Searches for files by name, size or date. | | files
bash | chmod | call | Changes who can read, write or run a file (chmod +x makes it runnable). | | files
bash | chown | call | Changes who owns a file. | | files
bash | apt,apt-get | call | Installs or updates software packages (Debian, Ubuntu, Raspberry Pi OS). | Usually needs sudo. |
bash | pip,pip3 | call | Installs Python packages. | |
bash | git | call | Runs a git command (version control). | |
bash | curl,wget | call | Downloads something from the internet. | | network
bash | tar | call | Packs or unpacks .tar archives (-x unpack, -c create, -z gzip). | | files
bash | unzip,zip | call | Unpacks or makes a zip file. | | files
bash | ssh | call | Connects to another computer's terminal securely. | | network
bash | scp,rsync | call | Copies files to or from another computer. | | network
bash | systemctl | call | Starts, stops or enables a background service. | |
bash | sleep | call | Waits for some seconds. | | timers
bash | kill,pkill | call | Stops a running program. | |
bash | ps | call | Lists running programs. | |
bash | head,tail | call | Shows the first (or last) lines of a file. tail -f keeps watching it. | | files
bash | sort,uniq,wc | call | Sorts lines, removes repeats, or counts lines and words. | | text
bash | date | call | Prints (or formats) the date and time. | | timers
bash | python3,python,node,go,make,gcc | call | Runs a program or build tool. | |
bash | source,. | call | Runs another script inside this one, so its variables stay set. | |
bash | printf | print | Prints formatted text. | | output
bash | test | call | Checks a condition (the same as [ ]). | | if
bash | trap | call | Runs a command when the script is stopped or exits. | Used for clean-up. |
bash | dd | call | Copies raw data block by block, for example writing a disk image to an SD card. | Careful: the wrong of= target can wipe a disk. | files

# ---------------------------------------------------------------- PowerShell cmdlets
powershell | Write-Host | print | Prints text to the screen (it can't be piped onward). | | output
powershell | Write-Output | print | Sends values down the pipeline (and to the screen at the end). | | output
powershell | Read-Host | input | Asks the user to type something. | | input
powershell | Get-ChildItem | call | Lists files and folders (like dir or ls). | | files
powershell | Get-Content | call | Reads a file's lines. | | files
powershell | Set-Content,Out-File,Add-Content | call | Writes (or adds) text to a file. | | files
powershell | Copy-Item,Move-Item,Remove-Item,Rename-Item,New-Item | call | Copies, moves, deletes, renames or creates files and folders. | | files
powershell | Test-Path | call | Answers whether a file or folder exists. | | files
powershell | Join-Path | call | Joins parts of a path correctly. | | files
powershell | Get-Process,Stop-Process,Start-Process | call | Lists, stops or starts programs. | |
powershell | Start-Sleep | call | Waits for some seconds (-Milliseconds for shorter). | | timers
powershell | Get-Date | call | Gets the date and time. | | timers
powershell | Invoke-WebRequest,Invoke-RestMethod | call | Downloads a web page or calls a web API. | | network
powershell | Where-Object | call | Keeps only the items that pass a test ($_ is each item). | | list
powershell | ForEach-Object | call | Runs a block for each item in the pipeline ($_ is the item). | | loop
powershell | Select-Object | call | Picks certain properties, or the first/last few items. | | list
powershell | Sort-Object,Measure-Object,Group-Object | call | Sorts, counts or groups the items. | | list
powershell | ConvertTo-Json,ConvertFrom-Json,Import-Csv,Export-Csv | call | Converts data to or from JSON or CSV. | | data
powershell | Get-Service,Start-Service,Stop-Service | call | Lists, starts or stops Windows services. | | windows-go
powershell | Get-ItemProperty,Set-ItemProperty | call | Reads or writes properties, including registry values (HKLM:\…). | | windows-go
powershell | Set-ExecutionPolicy | call | Changes whether PowerShell scripts are allowed to run. | | windows-go security
powershell | Add-Type | call | Loads .NET code or assemblies, for example to show a Windows Forms message box. | | windows-go

# ---------------------------------------------------------------- batch commands
batch | pause | call | Shows "Press any key to continue…" and waits. | Stops the window closing before you can read it. | input
batch | cls | call | Clears the window. | | rendering
batch | title | call | Sets the window's title. | |
batch | color | call | Sets the window's colours (two hex digits: background then text, such as 0A). | | rendering
batch | cd,chdir | call | Changes folder (/d also changes drive). | %~dp0 is the folder the batch file is in. | files
batch | dir | call | Lists files. | | files
batch | copy,xcopy,robocopy | call | Copies files (robocopy is the most capable). | | files
batch | del,erase | call | Deletes files. | | files
batch | md,mkdir | call | Makes a folder. | | files
batch | rd,rmdir | call | Deletes a folder (/s with everything inside). | | files
batch | ren,rename,move | call | Renames or moves files. | | files
batch | start | call | Starts a program or opens a file, without waiting for it. | The first quoted text is the window title, so start "" "file.txt". |
batch | choice | input | Asks the user to press one of some keys; the answer is in %errorlevel%. | | input
batch | timeout | call | Waits some seconds (/t), or until a key is pressed. | | timers
batch | ping | call | Checks whether another computer answers over the network. | Old trick: ping -n 3 127.0.0.1 >nul waits about 2 seconds. | network
batch | ipconfig | call | Shows network settings. | | network
batch | tasklist,taskkill | call | Lists or stops running programs. | |
batch | shutdown | call | Shuts down or restarts the computer. | |
batch | reg | call | Reads or changes the registry. | | windows-go
batch | shift | call | Moves the arguments down: %2 becomes %1. | |
batch | where | call | Finds where a program is. | |
batch | type | call | Prints a text file. | | files
batch | findstr | call | Searches for text in files. | | text
batch | powershell | call | Runs a PowerShell command. | |
`)
