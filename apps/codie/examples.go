// Codie — examples.go: the ready-made examples in the Examples menu.
// Part of the free nrmluvdev maker suite. Created by nrmluvdev.
package codie

type Sample struct {
	Group string `json:"group"`
	Name  string `json:"name"`
	Lang  string `json:"lang"`
	Code  string `json:"code"`
}

func fromLesson(group, name, lesson string, i int) Sample {
	l := LessonByID(lesson)
	e := l.Examples[i]
	return Sample{group, name, e.Lang, e.Code}
}

// Samples lists every example, in menu order.
func Samples() []Sample {
	return []Sample{
		{"Everyday languages", "Python: guess the number", "python", `import random

secret = random.randint(1, 100)
tries = 0

print("I'm thinking of a number from 1 to 100.")
while True:
    guess = int(input("Your guess: "))
    tries += 1
    if guess < secret:
        print("Higher!")
    elif guess > secret:
        print("Lower!")
    else:
        print(f"Got it in {tries} tries!")
        break
`},
		{"Everyday languages", "JavaScript: to-do list", "javascript", `const list = document.getElementById("todos");
const input = document.querySelector("#new-todo");
const todos = JSON.parse(localStorage.getItem("todos") || "[]");

function render() {
  list.innerHTML = "";
  for (const text of todos) {
    const li = document.createElement("li");
    li.textContent = text;
    list.appendChild(li);
  }
}

input.addEventListener("keydown", (e) => {
  if (e.key === "Enter" && input.value.trim() !== "") {
    todos.push(input.value.trim());
    localStorage.setItem("todos", JSON.stringify(todos));
    input.value = "";
    render();
  }
});

render();
`},
		{"Everyday languages", "C: temperature table", "c", `#include <stdio.h>

/* Print a table of Celsius and Fahrenheit. */
int main(void) {
    int celsius;
    for (celsius = 0; celsius <= 100; celsius += 10) {
        float f = celsius * 9.0f / 5.0f + 32.0f;
        printf("%3d C = %5.1f F\n", celsius, f);
    }
    return 0;
}
`},
		{"Everyday languages", "C++: inventory class", "cpp", `#include <iostream>
#include <string>
#include <vector>

class Inventory {
public:
    void add(const std::string& item) {
        items.push_back(item);
    }

    void show() const {
        for (const std::string& item : items) {
            std::cout << "- " << item << std::endl;
        }
    }

private:
    std::vector<std::string> items;
};

int main() {
    Inventory bag;
    bag.add("sword");
    bag.add("potion");
    bag.show();
    return 0;
}
`},
		{"Everyday languages", "C#: console greeter", "csharp", `using System;

namespace Hello
{
    class Program
    {
        static void Main(string[] args)
        {
            Console.Write("What's your name? ");
            string name = Console.ReadLine();
            int length = name.Length;
            if (length == 0)
            {
                Console.WriteLine("Hello, stranger!");
            }
            else
            {
                Console.WriteLine($"Hello, {name}! Your name has {length} letters.");
            }
        }
    }
}
`},
		{"Everyday languages", "Java: times tables", "java", `import java.util.Scanner;

public class Tables {
    public static void main(String[] args) {
        Scanner in = new Scanner(System.in);
        System.out.print("Which table? ");
        int n = in.nextInt();
        for (int i = 1; i <= 12; i++) {
            System.out.println(i + " x " + n + " = " + (i * n));
        }
    }
}
`},
		{"Everyday languages", "Go: count words in a file", "go", `package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	f, err := os.Open("story.txt")
	if err != nil {
		fmt.Println("couldn't open the file:", err)
		return
	}
	defer f.Close()

	counts := map[string]int{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		for _, word := range strings.Fields(scanner.Text()) {
			counts[strings.ToLower(word)]++
		}
	}
	fmt.Println("different words:", len(counts))
}
`},
		{"Everyday languages", "Rust: match a command", "rust", `use std::io;

fn main() {
    let mut input = String::new();
    println!("Type a command:");
    io::stdin().read_line(&mut input).expect("couldn't read");
    match input.trim() {
        "hello" => println!("Hi there!"),
        "quit" => println!("Bye!"),
        other => println!("I don't know '{}'", other),
    }
}
`},
		{"Everyday languages", "Kotlin: data class", "kotlin", `data class Player(val name: String, var score: Int)

fun main() {
    val players = mutableListOf(Player("Ana", 120), Player("Ben", 95))
    for (p in players) {
        println("${p.name}: ${p.score}")
    }
}
`},
		{"Everyday languages", "PHP: contact form", "php", `<?php
$name = $_POST["name"] ?? "";
if ($name !== "") {
    echo "Thanks, " . htmlspecialchars($name) . "!";
} else {
    echo "Please tell us your name.";
}
?>
`},
		{"Everyday languages", "Ruby: a class", "ruby", `class Dog
  attr_reader :name

  def initialize(name)
    @name = name
  end

  def speak
    puts "#{@name} says woof!"
  end
end

rex = Dog.new("Rex")
rex.speak
`},
		{"Everyday languages", "SQL: high-score table", "sql", `CREATE TABLE scores (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    player TEXT NOT NULL,
    points INTEGER DEFAULT 0
);

INSERT INTO scores (player, points) VALUES ('Ana', 1200);

SELECT player, points
FROM scores
WHERE points > 1000
ORDER BY points DESC
LIMIT 10;
`},
		{"Everyday languages", "HTML: a game page", "html", `<!DOCTYPE html>
<html>
<head>
  <title>My game</title>
  <link rel="stylesheet" href="style.css">
</head>
<body>
  <h1>Space Dodge</h1>
  <canvas id="game" width="640" height="480"></canvas>
  <p>Use the arrow keys to move.</p>
  <script src="game.js"></script>
</body>
</html>
`},
		{"Everyday languages", "CSS: page style", "css", `:root {
  --accent: #ff8a00;
}

body {
  background: #101418;
  color: white;
  font-family: system-ui, sans-serif;
}

canvas {
  border: 2px solid var(--accent);
  image-rendering: pixelated;
}

.button:hover {
  opacity: 0.8;
}
`},
		{"Windows Go", "Windows Go: message box and drives", "go", `//go:build windows

package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	user32           = syscall.NewLazyDLL("user32.dll")
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procMessageBoxW  = user32.NewProc("MessageBoxW")
	procGetDrives    = kernel32.NewProc("GetLogicalDrives")
)

func messageBox(title, text string) {
	t, _ := syscall.UTF16PtrFromString(text)
	c, _ := syscall.UTF16PtrFromString(title)
	procMessageBoxW.Call(0, uintptr(unsafe.Pointer(t)), uintptr(unsafe.Pointer(c)), 0x40)
}

func main() {
	mask, _, _ := procGetDrives.Call()
	drives := ""
	for i := 0; i < 26; i++ {
		if mask&(1<<uint(i)) != 0 {
			drives += string(rune('A'+i)) + ": "
		}
	}
	messageBox("Drives", fmt.Sprintf("This PC has: %s", drives))
}
`},
		{"Windows Go", "Windows Go: game keys (GetAsyncKeyState)", "go", `//go:build windows

package main

import (
	"fmt"
	"time"

	"golang.org/x/sys/windows"
)

var (
	user32       = windows.NewLazySystemDLL("user32.dll")
	getKeyState  = user32.NewProc("GetAsyncKeyState")
)

const (
	vkLeft  = 0x25
	vkRight = 0x27
	vkEsc   = 0x1B
)

func held(key int) bool {
	r, _, _ := getKeyState.Call(uintptr(key))
	return r&0x8000 != 0
}

func main() {
	x := 40
	ticker := time.NewTicker(time.Second / 30)
	defer ticker.Stop()
	for range ticker.C {
		if held(vkEsc) {
			break
		}
		if held(vkLeft) && x > 0 {
			x--
		}
		if held(vkRight) && x < 79 {
			x++
		}
		fmt.Printf("\r%*s@%*s", x, "", 79-x, "")
	}
}
`},
		{"Windows Go", "Windows Go: open a file with Windows", "go", `//go:build windows

package main

import (
	"log"

	"golang.org/x/sys/windows"
)

func main() {
	path, err := windows.UTF16PtrFromString(` + "`C:\\Windows\\win.ini`" + `)
	if err != nil {
		log.Fatal(err)
	}
	h, err := windows.CreateFile(path, windows.GENERIC_READ, windows.FILE_SHARE_READ, nil, windows.OPEN_EXISTING, 0, 0)
	if err != nil {
		log.Fatal(err)
	}
	defer windows.CloseHandle(h)

	buf := make([]byte, 256)
	var n uint32
	if err := windows.ReadFile(h, buf, &n, nil); err != nil {
		log.Fatal(err)
	}
	log.Printf("read %d bytes: %s", n, buf[:n])
}
`},
		{"Scripts and shells", "Bash: back up a folder", "bash", `#!/bin/bash
set -eu

SOURCE="$HOME/projects"
DEST="/media/usb/backup"
STAMP=$(date +%Y-%m-%d)

if [ ! -d "$DEST" ]; then
  echo "Plug in the backup drive first."
  exit 1
fi

for dir in "$SOURCE"/*; do
  name=$(basename "$dir")
  tar -czf "$DEST/$name-$STAMP.tar.gz" -C "$SOURCE" "$name"
  echo "Saved $name"
done
`},
		{"Scripts and shells", "PowerShell: tidy Downloads", "powershell", `param([int]$Days = 30)

$folder = Join-Path $env:USERPROFILE "Downloads"
$old = Get-ChildItem $folder -File | Where-Object { $_.LastWriteTime -lt (Get-Date).AddDays(-$Days) }

foreach ($file in $old) {
    Write-Host "Removing $($file.Name)"
    Remove-Item $file.FullName -WhatIf
}

Write-Host "$($old.Count) old files found."
`},
		{"Scripts and shells", "Windows batch: menu", "batch", `@echo off
title My menu
:menu
cls
echo 1. Show IP address
echo 2. List files
echo 3. Quit
set /p choice=Pick one:
if "%choice%"=="1" goto ip
if "%choice%"=="2" goto files
if "%choice%"=="3" exit /b 0
goto menu

:ip
ipconfig
pause
goto menu

:files
dir /b
pause
goto menu
`},
		fromLesson("Game engines", "Unity: platformer jump (C#)", "gravity", 1),
		{"Game engines", "Unity: player controller (C#)", "csharp", `using UnityEngine;

public class PlayerController : MonoBehaviour
{
    public float speed = 6f;
    [SerializeField] private int health = 3;
    private Rigidbody rb;

    void Start()
    {
        rb = GetComponent<Rigidbody>();
    }

    void Update()
    {
        float h = Input.GetAxis("Horizontal");
        float v = Input.GetAxis("Vertical");
        Vector3 move = new Vector3(h, 0, v).normalized;
        transform.Translate(move * speed * Time.deltaTime);
    }

    void OnTriggerEnter(Collider other)
    {
        if (other.CompareTag("Enemy"))
        {
            health -= 1;
            if (health <= 0)
            {
                Destroy(gameObject);
            }
        }
    }
}
`},
		fromLesson("Game engines", "Godot 4: platformer (GDScript)", "gravity", 0),
		{"Game engines", "Godot: coin pickup (GDScript)", "gdscript", `extends Area2D

signal picked_up(value)

@export var value := 10
@onready var sound = $PickupSound

func _ready():
	body_entered.connect(_on_body_entered)

func _on_body_entered(body):
	if body.is_in_group("player"):
		picked_up.emit(value)
		sound.play()
		hide()
		await sound.finished
		queue_free()
`},
		fromLesson("Game engines", "LÖVE: jumping (Lua)", "gravity", 2),
		{"Game engines", "PICO-8: collect the stars", "lua", `function _init()
  x, y = 64, 64
  score = 0
  star = {x = rnd(120), y = rnd(120)}
end

function _update()
  if btn(0) then x -= 2 end
  if btn(1) then x += 2 end
  if btn(2) then y -= 2 end
  if btn(3) then y += 2 end
  if abs(x - star.x) < 6 and abs(y - star.y) < 6 then
    score += 1
    sfx(0)
    star = {x = rnd(120), y = rnd(120)}
  end
end

function _draw()
  cls(1)
  spr(1, x, y)
  spr(2, star.x, star.y)
  print("score: "..score, 2, 2, 7)
end
`},
		{"Game engines", "Roblox: kill brick and coins (Luau)", "lua", `local Players = game:GetService("Players")
local part = script.Parent

part.Touched:Connect(function(hit)
  local humanoid = hit.Parent:FindFirstChild("Humanoid")
  if humanoid then
    humanoid.Health = 0
  end
end)

Players.PlayerAdded:Connect(function(player)
  local stats = Instance.new("Folder")
  stats.Name = "leaderstats"
  stats.Parent = player
  local coins = Instance.new("IntValue")
  coins.Name = "Coins"
  coins.Parent = stats
end)
`},
		fromLesson("Game engines", "pygame: game loop (Python)", "game-loop", 0),
		fromLesson("Game engines", "HTML canvas loop (JavaScript)", "game-loop", 1),
		{"Game engines", "Unreal: rotating pickup (C++)", "cpp", `#include "Pickup.h"

APickup::APickup()
{
    PrimaryActorTick.bCanEverTick = true;
}

void APickup::BeginPlay()
{
    Super::BeginPlay();
    StartZ = GetActorLocation().Z;
}

void APickup::Tick(float DeltaTime)
{
    Super::Tick(DeltaTime);
    Time += DeltaTime;
    FVector Location = GetActorLocation();
    Location.Z = StartZ + FMath::Sin(Time * 2.f) * 20.f;
    SetActorLocation(Location);
    AddActorLocalRotation(FRotator(0.f, 90.f * DeltaTime, 0.f));
}
`},
		fromLesson("Game engines", "raylib game loop (C)", "game-loop", 3),
		fromLesson("Game engines", "Ebitengine game loop (Go)", "game-loop", 4),
		fromLesson("Game engines", "GameMaker: walls (GML)", "collision", 5),
		fromLesson("Older machines", "Commodore 64: sprite and joystick", "retro", 0),
		fromLesson("Older machines", "ZX Spectrum: dodge game", "retro", 1),
		fromLesson("Older machines", "BBC Micro: move a dot", "retro", 2),
		fromLesson("Older machines", "QB64: jumping box", "gravity", 4),
		{"Older machines", "Classic BASIC: number guessing", "basic", `10 REM GUESS THE NUMBER
20 RANDOMIZE TIMER
30 N=INT(RND(1)*100)+1
40 T=0
50 INPUT "YOUR GUESS";G
60 T=T+1
70 IF G<N THEN PRINT "HIGHER" : GOTO 50
80 IF G>N THEN PRINT "LOWER" : GOTO 50
90 PRINT "CORRECT IN ";T;" TRIES!"
100 END
`},
		{"Older machines", "Apple II: hi-res lines", "basic", `10 HGR
20 HCOLOR=3
30 FOR X=0 TO 279 STEP 10
40 HPLOT 0,0 TO X,159
50 NEXT X
60 IF PEEK(-16384)<128 THEN 60
70 TEXT : HOME
`},
		{"Older machines", "Visual Basic .NET: console", "basic", `Module Program
    Sub Main()
        Dim name As String
        Console.Write("Name? ")
        name = Console.ReadLine()
        For i As Integer = 1 To 3
            Console.WriteLine("Hello " & name)
        Next
    End Sub
End Module
`},
	}
}
