// Codie — lessons.go: Game logic school. Each lesson explains one idea that
// almost every game uses, with short working examples in the languages games
// are written in, from Unity and Godot to PICO-8 and Commodore 64 BASIC.
// Every example can be sent to the explainer to read it line by line.
// Part of the free nrmluvdev maker suite. Created by nrmluvdev.
package codie

type Example struct {
	Lang  string `json:"lang"`
	Label string `json:"label"`
	Code  string `json:"code"`
}

type Lesson struct {
	ID       string    `json:"id"`
	Concept  string    `json:"concept"`
	Title    string    `json:"title"`
	Intro    string    `json:"intro"`
	Points   []string  `json:"points"`
	Examples []Example `json:"examples"`
}

var lessons = []Lesson{
	{ID: "game-loop", Concept: "game-loop", Title: "The game loop",
		Intro:  "Every game, from Pong to the newest 3D adventure, is one loop that repeats many times a second. Each pass is a frame: read the input, update the world, draw it. Engines such as Unity, Godot and LÖVE run the loop for you and call your functions; with pygame, raylib or BASIC you write the loop yourself.",
		Points: []string{"Input: what is the player pressing?", "Update: move things, check collisions, count timers, change the score.", "Draw: clear the screen and draw everything in its new place.", "Wait: keep a steady speed (often 60 frames per second).", "Keep update and draw separate: drawing should never change the game."},
		Examples: []Example{
			{"python", "pygame (Python)", `import pygame

pygame.init()
screen = pygame.display.set_mode((640, 480))
clock = pygame.time.Clock()
x = 100
running = True

while running:
    for event in pygame.event.get():
        if event.type == pygame.QUIT:
            running = False
    x = x + 2
    screen.fill((0, 0, 0))
    pygame.draw.rect(screen, (255, 200, 0), (x, 200, 32, 32))
    pygame.display.flip()
    clock.tick(60)

pygame.quit()
`},
			{"javascript", "HTML canvas (JavaScript)", `const canvas = document.getElementById("game");
const ctx = canvas.getContext("2d");
let x = 100;
let last = performance.now();

function loop(now) {
  const dt = (now - last) / 1000;
  last = now;
  x += 120 * dt;
  ctx.clearRect(0, 0, canvas.width, canvas.height);
  ctx.fillStyle = "orange";
  ctx.fillRect(x, 200, 32, 32);
  requestAnimationFrame(loop);
}

requestAnimationFrame(loop);
`},
			{"lua", "LÖVE (Lua)", `function love.load()
  x = 100
end

function love.update(dt)
  x = x + 120 * dt
end

function love.draw()
  love.graphics.rectangle("fill", x, 200, 32, 32)
end
`},
			{"c", "raylib (C)", `#include "raylib.h"

int main(void) {
    InitWindow(640, 480, "Game loop");
    SetTargetFPS(60);
    float x = 100;
    while (!WindowShouldClose()) {
        x += 120 * GetFrameTime();
        BeginDrawing();
        ClearBackground(BLACK);
        DrawRectangle((int)x, 200, 32, 32, ORANGE);
        EndDrawing();
    }
    CloseWindow();
    return 0;
}
`},
			{"go", "Ebitengine (Go)", `package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type Game struct{ x float32 }

func (g *Game) Update() error {
	g.x += 2
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	vector.DrawFilledRect(screen, g.x, 200, 32, 32, color.RGBA{255, 200, 0, 255}, false)
}

func (g *Game) Layout(w, h int) (int, int) { return 640, 480 }

func main() {
	ebiten.SetWindowTitle("Game loop")
	ebiten.RunGame(&Game{x: 100})
}
`},
			{"basic", "Commodore 64 BASIC", `10 REM GAME LOOP: MOVE A BALL ACROSS THE SCREEN
20 PRINT CHR$(147)
30 X=0
40 POKE 1024+X,32
50 X=X+1 : IF X>39 THEN X=0
60 POKE 1024+X,81
70 FOR D=1 TO 30 : NEXT D
80 GOTO 40
`},
			{"basic", "QB64", `SCREEN 13
x = 0
DO
    CLS
    LINE (x, 90)-(x + 16, 106), 14, BF
    x = x + 1
    IF x > 320 THEN x = 0
    _DISPLAY
    _LIMIT 60
LOOP UNTIL INKEY$ = CHR$(27)
`},
		}},
	{ID: "delta-time", Concept: "delta-time", Title: "Delta time: same speed on every computer",
		Intro:  "If a game moves the player 5 pixels every frame, it runs twice as fast at 120 frames per second as at 60. Delta time (dt) is the time the last frame took, in seconds. Multiply speeds by it and they become 'pixels per second', the same on any computer.",
		Points: []string{"position += speed * dt", "At 60 FPS dt is about 0.0167; at 30 FPS about 0.033, so each frame moves twice as far and the speed per second stays the same.", "Physics often uses a fixed step instead (Unity's FixedUpdate, Godot's _physics_process) for steadier results.", "Fixed-speed machines (PICO-8, old consoles and home computers) skip delta time and move by whole pixels per frame."},
		Examples: []Example{
			{"csharp", "Unity (C#)", `using UnityEngine;

public class Mover : MonoBehaviour
{
    public float speed = 5f;

    void Update()
    {
        float h = Input.GetAxis("Horizontal");
        transform.Translate(h * speed * Time.deltaTime, 0, 0);
    }
}
`},
			{"gdscript", "Godot (GDScript)", `extends Node2D

var speed := 200.0

func _process(delta):
	if Input.is_action_pressed("ui_right"):
		position.x += speed * delta
	if Input.is_action_pressed("ui_left"):
		position.x -= speed * delta
`},
			{"lua", "LÖVE (Lua)", `local player = { x = 100, y = 100, speed = 200 }

function love.update(dt)
  if love.keyboard.isDown("right") then
    player.x = player.x + player.speed * dt
  end
  if love.keyboard.isDown("left") then
    player.x = player.x - player.speed * dt
  end
end
`},
			{"python", "pygame (Python)", `dt = clock.tick(60) / 1000
keys = pygame.key.get_pressed()
if keys[pygame.K_RIGHT]:
    player_x += 200 * dt
if keys[pygame.K_LEFT]:
    player_x -= 200 * dt
`},
			{"cpp", "Unreal (C++)", `void AMover::Tick(float DeltaTime)
{
    Super::Tick(DeltaTime);
    FVector Location = GetActorLocation();
    Location.X += Speed * DeltaTime;
    SetActorLocation(Location);
}
`},
		}},
	{ID: "input", Concept: "input", Title: "Reading the player's input",
		Intro:  "Games ask two questions about each control: is it held down right now (for walking), and was it just pressed this frame (for jumping, shooting, menus)? Mixing them up gives a player who jumps every frame the key is held.",
		Points: []string{"Held: GetKey, is_action_pressed, love.keyboard.isDown, btn, IsKeyDown.", "Just pressed: GetKeyDown, is_action_just_pressed, love.keypressed, btnp, IsKeyPressed.", "Input maps (named actions like \"jump\") let players remap keys and use gamepads.", "On old home computers, games read the joystick or keyboard straight from memory with PEEK, or with INKEY$."},
		Examples: []Example{
			{"csharp", "Unity (C#)", `void Update()
{
    if (Input.GetKey(KeyCode.RightArrow))
        transform.position += Vector3.right * speed * Time.deltaTime;

    if (Input.GetKeyDown(KeyCode.Space))
        Jump();
}
`},
			{"gdscript", "Godot (GDScript)", `func _physics_process(delta):
	var direction := Input.get_axis("move_left", "move_right")
	velocity.x = direction * speed
	if Input.is_action_just_pressed("jump") and is_on_floor():
		velocity.y = jump_speed
	move_and_slide()
`},
			{"lua", "PICO-8", `function _update()
  if btn(0) then x -= 1 end
  if btn(1) then x += 1 end
  if btnp(4) then
    sfx(0)
    shoot()
  end
end
`},
			{"javascript", "Browser (JavaScript)", `const keys = {};

addEventListener("keydown", e => { keys[e.key] = true; });
addEventListener("keyup", e => { keys[e.key] = false; });

function update(dt) {
  if (keys["ArrowLeft"]) player.x -= player.speed * dt;
  if (keys["ArrowRight"]) player.x += player.speed * dt;
}
`},
			{"basic", "Commodore 64 BASIC (joystick)", `10 REM READ JOYSTICK IN PORT 2
20 J=PEEK(56320)
30 IF (J AND 4)=0 THEN X=X-1
40 IF (J AND 8)=0 THEN X=X+1
50 IF (J AND 16)=0 THEN PRINT "FIRE!"
60 GOTO 20
`},
			{"basic", "ZX Spectrum BASIC", `10 LET x=15
20 LET k$=INKEY$
30 IF k$="o" THEN LET x=x-1
40 IF k$="p" THEN LET x=x+1
50 PRINT AT 20,x;"A"
60 GOTO 20
`},
			{"basic", "BBC BASIC (keys held)", `10 X%=20
20 REPEAT
30 IF INKEY(-98) THEN X%=X%-1
40 IF INKEY(-67) THEN X%=X%+1
50 PRINT TAB(X%,20);"*"
60 UNTIL FALSE
`},
		}},
	{ID: "gravity", Concept: "physics", Title: "Movement, gravity and jumping",
		Intro:  "Characters have a velocity: how far they move each second, across and up/down. Gravity adds a little to the downward velocity every frame, so things fall faster and faster. A jump sets a big upward velocity; gravity then slows it, stops it and brings the player back down.",
		Points: []string{"velocity.y += gravity * dt  (falling speeds up)", "position += velocity * dt  (move by the velocity)", "Only allow a jump when standing on the ground, or the player can fly.", "In many engines, y grows downwards, so 'up' is a negative number."},
		Examples: []Example{
			{"gdscript", "Godot 4 platformer (GDScript)", `extends CharacterBody2D

const SPEED = 300.0
const JUMP_VELOCITY = -400.0
var gravity = 980.0

func _physics_process(delta):
	if not is_on_floor():
		velocity.y += gravity * delta
	if Input.is_action_just_pressed("ui_accept") and is_on_floor():
		velocity.y = JUMP_VELOCITY
	var direction = Input.get_axis("ui_left", "ui_right")
	velocity.x = direction * SPEED
	move_and_slide()
`},
			{"csharp", "Unity 2D (C#)", `using UnityEngine;

public class Jumper : MonoBehaviour
{
    public float jumpForce = 7f;
    public float speed = 5f;
    private Rigidbody2D rb;
    private bool grounded;

    void Start()
    {
        rb = GetComponent<Rigidbody2D>();
    }

    void Update()
    {
        float h = Input.GetAxisRaw("Horizontal");
        rb.linearVelocity = new Vector2(h * speed, rb.linearVelocity.y);
        if (Input.GetButtonDown("Jump") && grounded)
            rb.AddForce(Vector2.up * jumpForce, ForceMode2D.Impulse);
    }

    void OnCollisionEnter2D(Collision2D other)
    {
        grounded = true;
    }

    void OnCollisionExit2D(Collision2D other)
    {
        grounded = false;
    }
}
`},
			{"lua", "LÖVE (Lua)", `local p = { x = 100, y = 300, vy = 0, onGround = true }
local GRAVITY = 900
local JUMP = -400

function love.update(dt)
  p.vy = p.vy + GRAVITY * dt
  p.y = p.y + p.vy * dt
  if p.y >= 300 then
    p.y = 300
    p.vy = 0
    p.onGround = true
  end
end

function love.keypressed(key)
  if key == "space" and p.onGround then
    p.vy = JUMP
    p.onGround = false
  end
end
`},
			{"lua", "PICO-8", `function _init()
  x, y, dy = 60, 100, 0
end

function _update()
  dy += 0.3
  y += dy
  if y > 100 then
    y = 100
    dy = 0
  end
  if btnp(4) and y == 100 then dy = -5 end
end

function _draw()
  cls()
  spr(1, x, y)
end
`},
			{"basic", "QBasic", `SCREEN 13
y = 150: vy = 0
DO
    k$ = INKEY$
    IF k$ = " " AND y = 150 THEN vy = -8
    vy = vy + .5
    y = y + vy
    IF y > 150 THEN y = 150: vy = 0
    CLS
    LINE (150, y)-(166, y + 16), 12, BF
    _LIMIT 60
LOOP UNTIL k$ = CHR$(27)
`},
		}},
	{ID: "collision", Concept: "collision", Title: "Collision detection",
		Intro:  "Collision detection asks: are these two things touching? The simplest and most used test treats each object as a rectangle (an axis-aligned bounding box, AABB). Two rectangles overlap when they overlap on the x axis AND on the y axis. Engines do this for you, but it's worth knowing what they're doing.",
		Points: []string{"Overlap if: a.x < b.x + b.w and a.x + a.w > b.x and a.y < b.y + b.h and a.y + a.h > b.y", "Circles: they touch when the distance between centres is less than the two radii added together.", "Tile games check the map: what tile is at the place I'm about to move to?", "Check before moving to stop at walls; check after moving to react to hits."},
		Examples: []Example{
			{"javascript", "AABB test (JavaScript)", `function overlaps(a, b) {
  return a.x < b.x + b.w &&
         a.x + a.w > b.x &&
         a.y < b.y + b.h &&
         a.y + a.h > b.y;
}

for (const coin of coins) {
  if (overlaps(player, coin)) {
    score += 10;
    coin.collected = true;
  }
}
`},
			{"python", "pygame Rects (Python)", `player = pygame.Rect(100, 100, 32, 32)
enemy = pygame.Rect(200, 100, 32, 32)

if player.colliderect(enemy):
    lives -= 1
    player.x = 100
`},
			{"lua", "Circle test (Lua)", `local function touching(a, b)
  local dx = a.x - b.x
  local dy = a.y - b.y
  return dx * dx + dy * dy < (a.r + b.r) ^ 2
end
`},
			{"csharp", "Unity triggers (C#)", `void OnTriggerEnter2D(Collider2D other)
{
    if (other.CompareTag("Coin"))
    {
        score += 1;
        Destroy(other.gameObject);
    }
}
`},
			{"gdscript", "Godot signals (GDScript)", `extends Area2D

signal collected

func _on_body_entered(body):
	if body.name == "Player":
		collected.emit()
		queue_free()
`},
			{"gml", "GameMaker (GML)", `var hsp = (keyboard_check(vk_right) - keyboard_check(vk_left)) * 4;
if (place_meeting(x + hsp, y, obj_wall)) {
    hsp = 0;
}
x += hsp;
`},
			{"lua", "PICO-8 map tiles", `function solid(x, y)
  return fget(mget(flr(x / 8), flr(y / 8)), 0)
end

function _update()
  local nx = x
  if btn(1) then nx = x + 1 end
  if not solid(nx + 7, y) then x = nx end
end
`},
			{"basic", "C64 sprite collisions", `10 REM SPRITE COLLISION REGISTER
20 C=PEEK(53278)
30 IF (C AND 3)=3 THEN PRINT "SPRITES 0 AND 1 HIT!"
40 GOTO 20
`},
		}},
	{ID: "states", Concept: "state-machine", Title: "Game states (state machines)",
		Intro:  "A game is always in one state: title screen, playing, paused, game over. A character is too: idle, running, jumping, hurt. Keeping the state in one variable and using a switch (or match) to pick the rules for each state keeps the code tidy and stops impossible mixes like jumping while dead.",
		Points: []string{"Store the state in one variable, ideally an enum of named choices.", "Each frame, do only what the current state allows.", "Change state in one place, for example a change_state() function, so entering a state can start its animation or sound."},
		Examples: []Example{
			{"gdscript", "Godot (GDScript)", `enum State { IDLE, RUN, JUMP }
var state = State.IDLE

func _physics_process(delta):
	match state:
		State.IDLE:
			if Input.is_action_pressed("ui_right"):
				state = State.RUN
		State.RUN:
			velocity.x = 200
			if Input.is_action_just_pressed("ui_accept"):
				state = State.JUMP
		State.JUMP:
			if is_on_floor():
				state = State.IDLE
	move_and_slide()
`},
			{"csharp", "C# (Unity or MonoGame)", `enum GameState { Title, Playing, Paused, GameOver }
GameState state = GameState.Title;

void Update()
{
    switch (state)
    {
        case GameState.Title:
            if (Input.GetKeyDown(KeyCode.Return)) state = GameState.Playing;
            break;
        case GameState.Playing:
            if (lives <= 0) state = GameState.GameOver;
            break;
        case GameState.GameOver:
            if (Input.GetKeyDown(KeyCode.R)) Restart();
            break;
    }
}
`},
			{"lua", "LÖVE (Lua)", `local state = "title"

function love.update(dt)
  if state == "playing" then
    updateGame(dt)
  end
end

function love.keypressed(key)
  if state == "title" and key == "return" then
    state = "playing"
  elseif state == "playing" and key == "p" then
    state = "paused"
  elseif state == "paused" and key == "p" then
    state = "playing"
  end
end
`},
			{"basic", "QBasic", `state = 1
DO
    k$ = INKEY$
    SELECT CASE state
        CASE 1
            LOCATE 10, 15: PRINT "PRESS SPACE"
            IF k$ = " " THEN state = 2: CLS
        CASE 2
            LOCATE 1, 1: PRINT "PLAYING... ESC TO END"
            IF k$ = CHR$(27) THEN state = 3
        CASE 3
            LOCATE 10, 15: PRINT "GAME OVER"
    END SELECT
LOOP UNTIL state = 3
`},
		}},
	{ID: "spawning", Concept: "spawning", Title: "Spawning: bullets, enemies and lists",
		Intro:  "Games keep the things that come and go (bullets, enemies, particles) in a list. Spawning adds a new one; every frame the game moves each item; when one is finished (off screen, hit something) it's removed. Engines do the list for you: Instantiate and Destroy in Unity, instantiate() and queue_free() in Godot.",
		Points: []string{"Spawn: make a new object and add it to the list (or the scene).", "Update all: loop through the list every frame.", "Remove the finished ones, or the game slowly gets slower.", "When removing while looping, go backwards, or items get skipped."},
		Examples: []Example{
			{"javascript", "JavaScript", `const bullets = [];

function shoot() {
  bullets.push({ x: player.x, y: player.y, speed: 400 });
}

function updateBullets(dt) {
  for (const b of bullets) {
    b.y -= b.speed * dt;
  }
  for (let i = bullets.length - 1; i >= 0; i--) {
    if (bullets[i].y < 0) bullets.splice(i, 1);
  }
}
`},
			{"lua", "LÖVE (Lua)", `bullets = {}

function shoot()
  table.insert(bullets, { x = player.x, y = player.y })
end

function love.update(dt)
  for i = #bullets, 1, -1 do
    local b = bullets[i]
    b.y = b.y - 400 * dt
    if b.y < 0 then
      table.remove(bullets, i)
    end
  end
end
`},
			{"csharp", "Unity (C#)", `public GameObject bulletPrefab;

void Update()
{
    if (Input.GetKeyDown(KeyCode.Space))
    {
        GameObject b = Instantiate(bulletPrefab, transform.position, Quaternion.identity);
        Destroy(b, 2f);
    }
}
`},
			{"gdscript", "Godot (GDScript)", `const Bullet = preload("res://bullet.tscn")

func _unhandled_input(event):
	if event.is_action_pressed("shoot"):
		var b = Bullet.instantiate()
		b.position = position
		get_parent().add_child(b)
`},
			{"python", "pygame (Python)", `bullets = []

for event in pygame.event.get():
    if event.type == pygame.KEYDOWN and event.key == pygame.K_SPACE:
        bullets.append(pygame.Rect(player.x, player.y, 4, 10))

for b in bullets:
    b.y -= 8
bullets = [b for b in bullets if b.y > 0]
`},
			{"basic", "BASIC arrays", `10 DIM BX(10),BY(10),BA(10)
20 REM FIND A FREE SLOT AND FIRE
30 FOR I=1 TO 10
40 IF BA(I)=0 THEN BA(I)=1 : BX(I)=PX : BY(I)=PY : I=10
50 NEXT I
60 RETURN
`},
		}},
	{ID: "timers", Concept: "timers", Title: "Timers and cooldowns",
		Intro:  "Lots of game rules are about time: you can only fire twice a second, enemies arrive every 5 seconds, you're invincible for a moment after a hit. A timer is a number that counts down by the frame time; when it reaches zero, the thing happens and the timer resets.",
		Points: []string{"Countdown: timer -= dt; if timer <= 0 then do it and set timer = delay.", "Or remember when it last happened: if now - lastShot > cooldown.", "Engines also have timer tools: coroutines (Unity), Timer nodes and await (Godot), alarms (GameMaker), task.wait (Roblox)."},
		Examples: []Example{
			{"lua", "LÖVE countdown (Lua)", `local spawnTimer = 0
local SPAWN_EVERY = 2

function love.update(dt)
  spawnTimer = spawnTimer - dt
  if spawnTimer <= 0 then
    spawnEnemy()
    spawnTimer = SPAWN_EVERY
  end
end
`},
			{"csharp", "Unity cooldown and coroutine (C#)", `public float fireRate = 0.5f;
private float nextFire = 0f;

void Update()
{
    if (Input.GetButton("Fire1") && Time.time >= nextFire)
    {
        nextFire = Time.time + fireRate;
        Shoot();
    }
}

IEnumerator Invincible()
{
    invincible = true;
    yield return new WaitForSeconds(2f);
    invincible = false;
}
`},
			{"gdscript", "Godot await (GDScript)", `func take_hit():
	invincible = true
	$AnimationPlayer.play("flash")
	await get_tree().create_timer(1.5).timeout
	invincible = false
`},
			{"lua", "Roblox (Luau)", `while true do
  task.wait(5)
  local enemy = template:Clone()
  enemy.Parent = workspace
end
`},
			{"lua", "PICO-8 frame counter", `function _init()
  t = 0
end

function _update()
  t += 1
  if t % 30 == 0 then
    spawn_enemy()
  end
end
`},
		}},
	{ID: "random", Concept: "random", Title: "Randomness",
		Intro:  "Random numbers make each game a little different: where the next enemy appears, which item drops, how much damage a hit does. Computers make 'pseudo-random' numbers from a starting value called a seed; seeding from the clock gives a different game every time.",
		Points: []string{"A whole number from 1 to 6: floor(random * 6) + 1.", "Pick from a list: list[random index].", "Seed once at the start (from the time), never every frame.", "Using the same seed on purpose gives the same 'random' level every time, which is how games share level seeds."},
		Examples: []Example{
			{"python", "Python", `import random

roll = random.randint(1, 6)
enemy_x = random.randint(0, 640)
loot = random.choice(["sword", "shield", "potion"])
`},
			{"javascript", "JavaScript", `const roll = Math.floor(Math.random() * 6) + 1;
const spawnX = Math.random() * canvas.width;
`},
			{"csharp", "Unity (C#)", `int roll = Random.Range(1, 7);
Vector3 spot = new Vector3(Random.Range(-8f, 8f), 6f, 0f);
Instantiate(enemyPrefab, spot, Quaternion.identity);
`},
			{"lua", "Lua", `math.randomseed(os.time())
local roll = math.random(1, 6)
`},
			{"basic", "Classic BASIC", `10 RANDOMIZE TIMER
20 R=INT(RND(1)*6)+1
30 PRINT "YOU ROLLED ";R
`},
		}},
	{ID: "score", Concept: "score", Title: "Score, lives and saving the high score",
		Intro:  "Score and lives are just numbers the game changes when things happen and draws on the screen. Game over is a check: are the lives used up? A high score has to be written somewhere that lasts after the game closes.",
		Points: []string{"Change the score in one place (add_score) so the display and sounds stay in sync.", "Check for game over after changing lives.", "Save small things (high score, settings) with the platform's simple save tool."},
		Examples: []Example{
			{"csharp", "Unity PlayerPrefs (C#)", `void AddScore(int points)
{
    score += points;
    if (score > PlayerPrefs.GetInt("HighScore", 0))
    {
        PlayerPrefs.SetInt("HighScore", score);
    }
}
`},
			{"javascript", "Browser localStorage (JavaScript)", `let best = Number(localStorage.getItem("best") || 0);

function gameOver() {
  if (score > best) {
    best = score;
    localStorage.setItem("best", best);
  }
}
`},
			{"gdscript", "Godot (GDScript)", `func save_high_score(value):
	var file = FileAccess.open("user://best.save", FileAccess.WRITE)
	file.store_32(value)
`},
			{"basic", "BASIC", `100 REM PLAYER HIT
110 L=L-1
120 IF L=0 THEN GOTO 900
130 RETURN
900 PRINT "GAME OVER! SCORE: ";S
910 IF S>H THEN H=S : PRINT "NEW HIGH SCORE!"
`},
		}},
	{ID: "animation", Concept: "animation", Title: "Animation",
		Intro:  "Sprite animation works like a flip-book: show frame 1, then frame 2, then frame 3, changing every few hundredths of a second. The trick is to count time and pick the frame from it, rather than changing frame every single game frame.",
		Points: []string{"frame = floor(time * framesPerSecond) % numberOfFrames", "Sprite sheets put every frame in one picture; the game draws just one part of it.", "Engines have animation tools (Animator in Unity, AnimatedSprite2D and AnimationPlayer in Godot)."},
		Examples: []Example{
			{"lua", "PICO-8", `function _draw()
  cls()
  local frame = 1 + flr(t() * 8) % 4
  spr(frame, x, y)
end
`},
			{"javascript", "Sprite sheet (JavaScript)", `const FRAME_W = 32;
let frame = 0;
let frameTime = 0;

function update(dt) {
  frameTime += dt;
  if (frameTime > 0.1) {
    frame = (frame + 1) % 4;
    frameTime = 0;
  }
}

function draw() {
  ctx.drawImage(sheet, frame * FRAME_W, 0, FRAME_W, 32, player.x, player.y, 32, 32);
}
`},
			{"gdscript", "Godot (GDScript)", `func _physics_process(delta):
	if velocity.x != 0:
		$AnimatedSprite2D.play("run")
	else:
		$AnimatedSprite2D.play("idle")
	$AnimatedSprite2D.flip_h = velocity.x < 0
`},
		}},
	{ID: "retro", Concept: "hardware", Title: "Games on 8-bit machines",
		Intro:  "1980s home computers had no game engines and very little memory. BASIC games drew with characters, read the keyboard with INKEY$ or GET, and on machines like the Commodore 64 controlled the graphics and sound chips by POKEing numbers into memory. Their main loop was simply a run of lines ending in GOTO back to the top.",
		Points: []string{"Screen memory: on the C64, POKE 1024+X+40*Y puts a character on the screen; 55296+… sets its colour.", "Sprites (C64): switch on with 53269, position with 53248/53249, picture with 2040.", "ZX Spectrum: PRINT AT row,col places characters; ATTR(y,x) reads colours for collisions.", "BBC Micro: MODE picks the screen, GCOL the colour, INKEY(-n) checks held keys.", "BASIC was slow, so many classic games were written in machine code and started with SYS or CALL."},
		Examples: []Example{
			{"basic", "C64 sprite", `10 V=53248
20 POKE V+21,1
30 POKE 2040,13
40 FOR I=0 TO 62 : POKE 832+I,255 : NEXT I
50 POKE V+39,7
60 X=100 : Y=100
70 POKE V,X : POKE V+1,Y
80 J=PEEK(56320)
90 IF (J AND 4)=0 THEN X=X-2
100 IF (J AND 8)=0 THEN X=X+2
110 GOTO 70
`},
			{"basic", "ZX Spectrum dodge game", `10 BORDER 0: PAPER 0: INK 7: CLS
20 LET x=15: LET s=0
30 LET r=INT (RND*32)
40 PRINT AT 0,r; INK 2;"*"
50 IF INKEY$="o" AND x>0 THEN LET x=x-1
60 IF INKEY$="p" AND x<31 THEN LET x=x+1
70 PRINT AT 21,x; INK 6;"A"
80 LET s=s+1
90 BEEP 0.01,s/10
100 GOTO 30
`},
			{"basic", "BBC Micro", `10 MODE 2
20 X%=640 : Y%=512
30 REPEAT
40 GCOL 0,0 : PLOT 69,X%,Y%
50 IF INKEY(-98) THEN X%=X%-8
60 IF INKEY(-67) THEN X%=X%+8
70 GCOL 0,3 : PLOT 69,X%,Y%
80 UNTIL INKEY(-113)
`},
		}},
}

// LessonByID finds a lesson.
func LessonByID(id string) *Lesson {
	for i := range lessons {
		if lessons[i].ID == id {
			return &lessons[i]
		}
	}
	return nil
}
