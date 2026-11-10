// Codie — lang.go: the languages Codie knows, how their comments and text
// values look, and how Codie guesses which language some pasted code is in.
// Part of the free nrmluvdev maker suite. Created by nrmluvdev.
package codie

import (
	"regexp"
	"sort"
	"strings"
)

// Lang describes one language well enough to read it line by line.
type Lang struct {
	ID     string      `json:"id"`
	Name   string      `json:"name"`
	Family string      `json:"family"` // c, py, rb, lua, basic, shell, sql, web
	Group  string      `json:"group"`  // menu heading
	About  string      `json:"about"`
	Line   []string    `json:"-"` // line-comment markers
	Block  [][2]string `json:"-"` // block-comment pairs
	Quotes string      `json:"-"` // characters that start text values
	NoCase bool        `json:"-"` // keywords ignore upper/lower case
	Exts   []string    `json:"-"`
}

var langs = []*Lang{
	// Everyday languages
	{ID: "python", Name: "Python", Family: "py", Group: "Everyday languages", About: "A friendly general-purpose language used for scripts, science, websites, AI and games (pygame).", Line: []string{"#"}, Quotes: `"'`, Exts: []string{"py", "pyw"}},
	{ID: "javascript", Name: "JavaScript", Family: "c", Group: "Everyday languages", About: "The language of web pages; also runs servers (Node.js) and browser games (canvas, Phaser).", Line: []string{"//"}, Block: [][2]string{{"/*", "*/"}}, Quotes: "\"'`", Exts: []string{"js", "mjs", "cjs", "jsx"}},
	{ID: "typescript", Name: "TypeScript", Family: "c", Group: "Everyday languages", About: "JavaScript with types added, so mistakes are caught before the code runs.", Line: []string{"//"}, Block: [][2]string{{"/*", "*/"}}, Quotes: "\"'`", Exts: []string{"ts", "tsx"}},
	{ID: "java", Name: "Java", Family: "c", Group: "Everyday languages", About: "A widely used language for Android apps, big business systems and Minecraft mods.", Line: []string{"//"}, Block: [][2]string{{"/*", "*/"}}, Quotes: `"'`, Exts: []string{"java"}},
	{ID: "csharp", Name: "C#", Family: "c", Group: "Everyday languages", About: "Microsoft's main language: Windows apps, web servers, and Unity and Godot games.", Line: []string{"//"}, Block: [][2]string{{"/*", "*/"}}, Quotes: `"'`, Exts: []string{"cs"}},
	{ID: "c", Name: "C", Family: "c", Group: "Everyday languages", About: "A small, fast language close to the hardware: operating systems, microcontrollers and classic games.", Line: []string{"//"}, Block: [][2]string{{"/*", "*/"}}, Quotes: `"'`, Exts: []string{"c", "h"}},
	{ID: "cpp", Name: "C++", Family: "c", Group: "Everyday languages", About: "C with classes and much more: game engines (Unreal), Arduino sketches, desktop apps.", Line: []string{"//"}, Block: [][2]string{{"/*", "*/"}}, Quotes: `"'`, Exts: []string{"cpp", "cc", "cxx", "hpp", "hh", "ino"}},
	{ID: "go", Name: "Go", Family: "c", Group: "Everyday languages", About: "Google's simple, fast language for tools and servers. Tinkerie itself is written in Go, including its Windows parts.", Line: []string{"//"}, Block: [][2]string{{"/*", "*/"}}, Quotes: "\"'`", Exts: []string{"go"}},
	{ID: "rust", Name: "Rust", Family: "c", Group: "Everyday languages", About: "A fast language that checks memory safety while compiling; used for tools, browsers and games (Bevy).", Line: []string{"//"}, Block: [][2]string{{"/*", "*/"}}, Quotes: `"`, Exts: []string{"rs"}},
	{ID: "kotlin", Name: "Kotlin", Family: "c", Group: "Everyday languages", About: "A modern language for Android apps that runs alongside Java.", Line: []string{"//"}, Block: [][2]string{{"/*", "*/"}}, Quotes: `"'`, Exts: []string{"kt", "kts"}},
	{ID: "swift", Name: "Swift", Family: "c", Group: "Everyday languages", About: "Apple's language for iPhone, iPad and Mac apps and games (SpriteKit).", Line: []string{"//"}, Block: [][2]string{{"/*", "*/"}}, Quotes: `"`, Exts: []string{"swift"}},
	{ID: "php", Name: "PHP", Family: "c", Group: "Everyday languages", About: "A language that builds web pages on the server; WordPress is written in it.", Line: []string{"//", "#"}, Block: [][2]string{{"/*", "*/"}}, Quotes: `"'`, Exts: []string{"php"}},
	{ID: "ruby", Name: "Ruby", Family: "rb", Group: "Everyday languages", About: "A readable scripting language, best known for the Ruby on Rails web framework.", Line: []string{"#"}, Block: [][2]string{{"=begin", "=end"}}, Quotes: `"'`, Exts: []string{"rb"}},
	{ID: "sql", Name: "SQL", Family: "sql", Group: "Everyday languages", About: "The language for asking databases questions and changing what they store.", Line: []string{"--"}, Block: [][2]string{{"/*", "*/"}}, Quotes: `'"`, NoCase: true, Exts: []string{"sql"}},
	{ID: "html", Name: "HTML", Family: "web", Group: "Everyday languages", About: "Describes what is on a web page: headings, text, pictures, buttons, a game's canvas.", Block: [][2]string{{"<!--", "-->"}}, Quotes: `"'`, NoCase: true, Exts: []string{"html", "htm"}},
	{ID: "css", Name: "CSS", Family: "web", Group: "Everyday languages", About: "Describes how a web page looks: colours, sizes, fonts and layout.", Block: [][2]string{{"/*", "*/"}}, Quotes: `"'`, Exts: []string{"css"}},
	// Scripts and shells
	{ID: "bash", Name: "Bash / shell", Family: "shell", Group: "Scripts and shells", About: "The command language of Linux, macOS and the Raspberry Pi terminal.", Line: []string{"#"}, Quotes: `"'`, Exts: []string{"sh", "bash"}},
	{ID: "powershell", Name: "PowerShell", Family: "shell", Group: "Scripts and shells", About: "Windows' modern command language, built from Verb-Noun commands.", Line: []string{"#"}, Block: [][2]string{{"<#", "#>"}}, Quotes: `"'`, NoCase: true, Exts: []string{"ps1", "psm1"}},
	{ID: "batch", Name: "Windows batch", Family: "shell", Group: "Scripts and shells", About: "The classic .bat / .cmd scripts that run in the Windows Command Prompt.", Line: []string{"::", "rem ", "REM ", "@rem ", "@REM "}, Quotes: `"`, NoCase: true, Exts: []string{"bat", "cmd"}},
	// Game languages
	{ID: "lua", Name: "Lua", Family: "lua", Group: "Game languages", About: "A tiny scripting language built into many games: LÖVE, PICO-8, Roblox (Luau), Defold, Solar2D, WoW add-ons.", Line: []string{"--"}, Block: [][2]string{{"--[[", "]]"}}, Quotes: `"'`, Exts: []string{"lua", "luau", "p8"}},
	{ID: "gdscript", Name: "GDScript", Family: "py", Group: "Game languages", About: "Godot's own scripting language. It looks like Python and is made for games.", Line: []string{"#"}, Quotes: `"'`, Exts: []string{"gd"}},
	{ID: "gml", Name: "GameMaker (GML)", Family: "c", Group: "Game languages", About: "GameMaker's language, used for many 2D indie games.", Line: []string{"//"}, Block: [][2]string{{"/*", "*/"}}, Quotes: `"'`, Exts: []string{"gml"}},
	// Retro
	{ID: "basic", Name: "BASIC", Family: "basic", Group: "Older machines", About: "The language built into 1980s home computers (Commodore 64, ZX Spectrum, BBC Micro, Apple II) and later QBasic, QB64 and Visual Basic.", Line: []string{"REM", "'"}, Quotes: `"`, NoCase: true, Exts: []string{"bas", "bbc", "vb", "vbs"}},
}

var langByID = map[string]*Lang{}

func init() {
	for _, l := range langs {
		langByID[l.ID] = l
	}
}

// LangList is what the language menu shows.
func LangList() []*Lang { return langs }

// ------------------------------------------------------------ detection

type sig struct {
	re *regexp.Regexp
	w  float64
}

func s(w float64, pat string) sig { return sig{regexp.MustCompile(pat), w} }

// Each language has a list of telltale patterns and how much each counts.
var sigs = map[string][]sig{
	"go":         {s(6, `(?m)^package \w+\s*$`), s(4, `(?m)^func (\(\w+ \*?\w+\) )?\w+\(.*\{\s*$`), s(3, `\w := `), s(4, `\bfmt\.`), s(3, `(?m)^import \($`), s(3, `err != nil`), s(2, `\bchan\b|\bgo func\b|\bdefer\b`), s(6, `(?m)^//go:build`), s(4, `syscall\.NewLazyDLL|golang\.org/x/sys/windows`), s(3, `ebiten\.`)},
	"c":          {s(3, `#include\s*<(stdio|stdlib|string|math|stdint|stdbool)\.h>`), s(3, `\bprintf\s*\(`), s(2, `\bint main\s*\(`), s(2, `\bmalloc\s*\(|\bfree\s*\(`), s(1, `(?m)^#(include|define)`), s(1, `->`), s(1, `\bstruct \w+\s*\{`), s(2, `\btypedef\b`)},
	"cpp":        {s(4, `#include\s*<(iostream|vector|string|map|memory|algorithm)>`), s(4, `\bstd::`), s(3, `\bcout\s*<<`), s(2, `\bnullptr\b|\btemplate\s*<|\bnamespace \w+|\bauto\b`), s(1, `(?m)^#include`), s(5, `\bUCLASS\(|\bUFUNCTION\(|\bUPROPERTY\(|\bAActor\b|\bGENERATED_BODY\(`), s(3, `\bvoid setup\s*\(\)|\bvoid loop\s*\(\)|\bdigitalWrite\b|\bSerial\.begin`), s(2, `\w::\w+\(`), s(1, `\bclass \w+`), s(4, `\bSuper::\w+\(|\bFVector\b|\bFRotator\b|\bFMath::|\bfloat DeltaTime\b`), s(3, `(?m)^\w+\s+A\w+::\w+\(|(?m)^A\w+::A\w+\(`)},
	"csharp":     {s(4, `(?m)^using (System|UnityEngine|Godot)[\w.]*;`), s(5, `:\s*MonoBehaviour\b`), s(3, `\bConsole\.Write(Line)?\(`), s(2, `\bnamespace [\w.]+`), s(2, `\{\s*get;\s*(set;)?\s*\}`), s(2, `\b(public|private) (void|int|float|string|bool) \w+`), s(2, `\bvar \w+ = new\b`), s(3, `\bTime\.deltaTime\b|\bGetComponent<|\bVector[23]\b`), s(1, `\bforeach\s*\(`), s(4, `\bvoid (Update|Start|Awake|FixedUpdate|LateUpdate|OnTrigger\w*|OnCollision\w*)\s*\(`), s(3, `\bInput\.Get\w+\(|\bPlayerPrefs\.|\bCompareTag\(|\bDebug\.Log\(|\bQuaternion\.|\bIEnumerator\b|\bgameObject\b`)},
	"java":       {s(5, `public static void main\s*\(\s*String`), s(4, `\bSystem\.out\.print`), s(3, `(?m)^import java\.`), s(2, `@Override`), s(1, `\b(public|private|protected) (static )?(final )?\w+ \w+\s*[=;(]`), s(1, `\bextends \w+`), s(2, `\bnew \w+\(.*\);`), s(1, `\bString\[\]`)},
	"javascript": {s(3, `\bfunction\s*\w*\s*\(`), s(2, `\b(const|let) \w+ =`), s(2, `=>`), s(4, `\bconsole\.log\(`), s(3, `\bdocument\.|\bwindow\.`), s(2, `===|!==`), s(2, `\brequire\(|\bmodule\.exports\b`), s(3, `requestAnimationFrame|getContext\(['"]2d`), s(1, `\bvar \w+ =`)},
	"typescript": {s(4, `:\s*(string|number|boolean|void|any)\b`), s(4, `\binterface \w+\s*\{`), s(3, `\btype \w+ =`), s(2, `\bexport (default )?(class|function|const)`), s(2, `<\w+>\(`), s(2, `\bprivate \w+:`)},
	"rust":       {s(5, `\bfn \w+\s*[(<]`), s(4, `\blet mut\b`), s(4, `\bprintln!\(`), s(3, `\bimpl\b`), s(3, `(?m)^use [\w:]+`), s(2, `&mut\b|\bSome\(|\bOk\(|\bunwrap\(\)`), s(2, `\bmatch \w+ \{`), s(2, `\bpub (fn|struct)\b`), s(1, `->`)},
	"kotlin":     {s(5, `\bfun \w+\s*\(`), s(3, `\bval \w+`), s(1, `\bvar \w+`), s(2, `\bprintln\(`), s(3, `\bdata class\b|\bcompanion object\b|\bwhen\s*\(`)},
	"swift":      {s(4, `\bfunc \w+\s*\(.*\{`), s(4, `(?m)^import (UIKit|SwiftUI|SpriteKit|Foundation|GameplayKit)`), s(3, `\bguard let\b|\bif let\b`), s(3, `var body: some View`), s(1, `\blet \w+ =`), s(2, `\bSKScene\b|\bSKSpriteNode\b`)},
	"php":        {s(8, `<\?php`), s(3, `\$\w+\s*=`), s(2, `\becho\b`), s(1, `->`), s(1, `\bfunction \w+\s*\(`)},
	"python":     {s(5, `(?m)^\s*def \w+\(.*\):\s*$`), s(3, `(?m)^(import \w+|from [\w.]+ import)`), s(2, `\bprint\(`), s(2, `\bself\.`), s(3, `\belif\b`), s(4, `if __name__ == ['"]__main__['"]`), s(1, `(?m):\s*$`), s(3, `\bpygame\.`), s(2, `\bTrue\b|\bFalse\b|\bNone\b`), s(1, `\brange\(`)},
	"gdscript":   {s(6, `(?m)^extends \w+`), s(5, `\bfunc _(ready|process|physics_process|input|unhandled_input)\(`), s(4, `@onready|@export|\bonready var\b|\bexport\(`), s(5, `(?m)^\s*func \w+\(.*\)( -> \w+)?:\s*$`), s(2, `\$\w+`), s(2, `\bvar \w+\s*(:=|: \w+)`), s(3, `\bmove_and_slide\(|\bInput\.is_action|\bsignal \w+|\bemit_signal\(|\bqueue_free\(\)`), s(2, `(?m)^class_name `), s(3, `\bawait\b.*\.timeout|\$\w+\.\w+\(|\bget_tree\(\)|\bis_on_floor\(\)`)},
	"ruby":       {s(3, `(?m)^\s*def \w+[?!]?(\(.*\))?\s*$`), s(2, `(?m)^\s*end\s*$`), s(3, `\bputs\b`), s(3, `\.each( do)? \|`), s(3, `\battr_(accessor|reader)\b`), s(2, `(?m)^require ['"]`), s(2, `@\w+`), s(1, `\bunless\b`)},
	"lua":        {s(3, `(?m)^\s*local \w+`), s(3, `\bfunction [\w.:]*\(`), s(2, `(?m)^\s*end\s*\)?\s*$`), s(2, `\bthen\s*$|\bthen\b`), s(2, `~=`), s(2, `\.\.\s*["'\w]`), s(5, `\blove\.\w+`), s(4, `(?m)^\s*function _(init|update|update60|draw)\(`), s(3, `\bbtnp?\(\d`), s(4, `\bgame:GetService\(|\bInstance\.new\(|:Connect\(`), s(2, `\bipairs\(|\bpairs\(`), s(2, `\bnil\b`), s(2, `(?m)^\s*--`)},
	"basic":      {s(5, `(?m)^\s*\d+\s*[A-Za-z?]`), s(3, `(?i)\bPRINT\b`), s(4, `(?i)\bGOTO\b|\bGOSUB\b`), s(3, `(?i)\bPOKE\b|\bPEEK\s*\(`), s(2, `\w\$`), s(2, `(?i)\bNEXT\b|\bWEND\b`), s(2, `(?m)(?i)^\s*(\d+\s*)?REM\b`), s(3, `(?i)\bEND (IF|SUB|FUNCTION|SELECT)\b|\bDim \w+ As\b`), s(3, `(?i)\bDEF PROC|\bENDPROC\b|\bGCOL\b|\bINKEY\$`), s(2, `(?i)\bTHEN\b`), s(2, `(?i)\bSCREEN \d+|\bLOCATE\b|\bRANDOMIZE\b|\bRND\b`)},
	"bash":       {s(8, `(?m)^#!.*\b(ba|z)?sh\b`), s(3, `(?m)^\s*(fi|done|esac)\s*$`), s(2, `\$\{?\w+\}?`), s(2, `(?m)^\s*echo\b`), s(3, `\[\[ |(?m)^\s*if \[`), s(2, `\bsudo\b|\bapt(-get)?\b|\bchmod\b|\bgrep\b`), s(1, `(?m)^\w+=\S`)},
	"powershell": {s(4, `\$\w+\s*=\s*`), s(5, `\b(Get|Set|New|Remove|Write|Start|Stop|Invoke|Test|Import|Out|Select|Where|ForEach)-[A-Z]\w+`), s(3, `-(eq|ne|lt|gt|le|ge|like|match)\b`), s(3, `(?m)^\s*param\s*\(`), s(2, `\|\s*%\s*\{|\$_\.`), s(2, `\[(string|int|switch|bool)\]`)},
	"batch":      {s(8, `(?im)^@echo off`), s(4, `(?im)^\s*set (/[ap] )?\w+=`), s(4, `%~?\w+%|%%\w|%~?\d`), s(3, `(?im)^\s*goto :?\w+|^:\w+`), s(3, `(?im)\bif (not )?(exist|errorlevel|defined)\b`), s(2, `(?im)^\s*pause\b|^\s*rem\b`)},
	"sql":        {s(5, `(?i)\bSELECT\b[\s\S]+?\bFROM\b`), s(5, `(?i)\bCREATE TABLE\b|\bINSERT INTO\b|\bALTER TABLE\b`), s(3, `(?i)\bWHERE\b|\bJOIN\b|\bGROUP BY\b|\bORDER BY\b`), s(3, `(?i)\bUPDATE \w+ SET\b|\bDELETE FROM\b`), s(1, `;\s*$`)},
	"html":       {s(8, `(?i)<!DOCTYPE html|<html`), s(4, `(?i)<(div|body|head|script|canvas|p|span|a|img|button|section)\b`), s(2, `</\w+>`)},
	"css":        {s(4, `(?m)^\s*[.#]?[\w-]+(\s*[,>+~ ]\s*[.#:]?[\w-]+)*\s*\{\s*$`), s(4, `(?m)^\s*[a-z-]+\s*:\s*[^:;{}()]+;\s*$`), s(3, `@media|@keyframes|:root|var\(--`), s(2, `\b\d+(px|rem|em|vh|vw|%)`)},
	"gml":        {s(5, `\binstance_create(_layer|_depth)?\(|\bplace_meeting\(|\bkeyboard_check(_pressed)?\(`), s(4, `\bdraw_sprite(_ext)?\(|\bsprite_index\b|\bimage_index\b|\binstance_destroy\(`), s(3, `\bvk_\w+|\bord\(["']`), s(2, `\bhsp\b|\bvsp\b|\broom_goto`)},
}

// Detection is how sure Codie is about a language.
type Detection struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Confidence float64  `json:"confidence"`
	Others     []string `json:"others"`
	Dialect    string   `json:"dialect,omitempty"`
	Chosen     bool     `json:"chosen"` // picked by the user rather than guessed
}

// Detect guesses the language of code. hint may be a language id, a file
// name with an extension, or empty.
func Detect(code, hint string) Detection {
	hint = strings.ToLower(strings.TrimSpace(hint))
	if l := langByID[hint]; l != nil {
		return Detection{ID: l.ID, Name: l.Name, Confidence: 1, Chosen: true, Dialect: dialect(l.ID, code)}
	}
	if i := strings.LastIndex(hint, "."); i >= 0 {
		ext := hint[i+1:]
		for _, l := range langs {
			for _, e := range l.Exts {
				if e == ext {
					return Detection{ID: l.ID, Name: l.Name, Confidence: 1, Chosen: true, Dialect: dialect(l.ID, code)}
				}
			}
		}
	}
	type sc struct {
		id string
		v  float64
	}
	var all []sc
	for id, list := range sigs {
		v := 0.0
		for _, g := range list {
			n := len(g.re.FindAllStringIndex(code, 4))
			if n > 0 {
				v += g.w * (1 + 0.25*float64(n-1))
			}
		}
		all = append(all, sc{id, v})
	}
	// Close relatives: settle them with the features only one of them has.
	sort.Slice(all, func(i, j int) bool {
		if all[i].v != all[j].v {
			return all[i].v > all[j].v
		}
		return all[i].id < all[j].id
	})
	if len(all) == 0 || all[0].v == 0 {
		return Detection{ID: "javascript", Name: langByID["javascript"].Name, Confidence: 0}
	}
	top := all[0]
	// TypeScript is JavaScript plus types; C++ is C plus more; GDScript looks
	// like Python. Prefer the specialised one when its own marks are present.
	score := map[string]float64{}
	for _, a := range all {
		score[a.id] = a.v
	}
	if top.id == "javascript" && score["typescript"] >= 4 {
		top = sc{"typescript", score["typescript"] + score["javascript"]}
	}
	if top.id == "c" && score["cpp"] >= 4 {
		top = sc{"cpp", score["cpp"] + score["c"]}
	}
	if top.id == "python" && score["gdscript"] >= 5 {
		top = sc{"gdscript", score["gdscript"] + score["python"]}
	}
	if (top.id == "javascript" || top.id == "c" || top.id == "cpp" || top.id == "csharp") && score["gml"] >= 5 {
		top = sc{"gml", score["gml"] + 1}
	}
	d := Detection{ID: top.id, Name: langByID[top.id].Name}
	second := 0.0
	for _, a := range all {
		if a.id == top.id {
			continue
		}
		if a.v > second {
			second = a.v
		}
		if a.v > 0 && len(d.Others) < 3 {
			d.Others = append(d.Others, a.id)
		}
	}
	d.Confidence = top.v / (top.v + second + 1)
	d.Dialect = dialect(top.id, code)
	return d
}

// dialect names the flavour of a language when it matters for explaining it.
func dialect(id, code string) string {
	has := func(p string) bool { return regexp.MustCompile(p).MatchString(code) }
	switch id {
	case "basic":
		switch {
		case has(`(?i)\bDim \w+ As\b|\bEnd Sub\b|\bConsole\.WriteLine\b|\bMsgBox\b`) && !has(`(?m)^\s*\d+\s`):
			if has(`(?i)\bConsole\.|\bModule\b|\bImports\b`) {
				return "Visual Basic .NET"
			}
			if has(`(?i)\bSCREEN \d+|\b_LIMIT\b|\bPSET\b`) {
				return "QBasic / QB64"
			}
			return "Visual Basic"
		case has(`(?i)POKE\s*5328[01]|PEEK\s*\(\s*5632[01]|PEEK\s*\(\s*197\s*\)|CHR\$\(147\)|\bSYS\s*\d|POKE\s*532(4[89]|[5-9]\d)|POKE\s*54\d\d\d|\bV\s*=\s*53248`):
			return "Commodore 64 BASIC"
		case has(`(?i)\bBORDER\s+\d|\bPAPER\s+\d|\bINK\s+\d|\bPRINT AT\b|\bBEEP\s|\bIN\s+31\b|\bRANDOMIZE USR\b`):
			return "ZX Spectrum BASIC"
		case has(`(?i)\bDEF ?PROC|\bENDPROC\b|\bGCOL\b|\bVDU\s*\d|\bMODE\s*\d|\bINKEY\s*\(\s*-\d|\*FX`):
			return "BBC BASIC"
		case has(`(?i)\bHGR\b|\bHPLOT\b|\bHCOLOR\s*=|\bPDL\s*\(|\bHOME\b|PEEK\s*\(\s*-16384`):
			return "Applesoft BASIC (Apple II)"
		case has(`(?i)\bSCREEN \d+|\bLOCATE\b|\bSUB \w+|\bEND SUB\b|\b_LIMIT\b|\bDO\b|\bLOOP\b|\bSLEEP\b|\bPSET\b`):
			return "QBasic / QB64"
		}
		if has(`(?m)^\s*\d+\s`) {
			return "Classic line-numbered BASIC"
		}
		return "BASIC"
	case "lua":
		switch {
		case has(`\blove\.\w+`):
			return "LÖVE (Lua game framework)"
		case has(`(?m)^\s*function _(init|update|update60|draw)\(|\bbtnp?\(|\bspr\(|\bcls\(`):
			return "PICO-8"
		case has(`\bgame:GetService\(|\bInstance\.new\(|\bworkspace\b|\btask\.wait\(`):
			return "Roblox Luau"
		case has(`\bdisplay\.new\w+\(|\bRuntime:addEventListener\(`):
			return "Solar2D (Corona)"
		}
	case "cpp":
		switch {
		case has(`\bUCLASS\(|\bAActor\b|\bGENERATED_BODY\(`):
			return "Unreal Engine C++"
		case has(`\bvoid setup\s*\(\)|\bdigitalWrite\b|\bSerial\.begin`):
			return "Arduino"
		}
	case "csharp":
		switch {
		case has(`\bMonoBehaviour\b|using UnityEngine`):
			return "Unity"
		case has(`using Godot`):
			return "Godot C#"
		case has(`Microsoft\.Xna\.Framework|\bGameTime\b`):
			return "MonoGame / XNA"
		}
	case "go":
		switch {
		case has(`(?m)^//go:build\s+windows|\+build windows|syscall\.NewLazyDLL|golang\.org/x/sys/windows|\bwindows\.\w+\(`):
			return "Windows Go"
		case has(`ebiten\.`):
			return "Ebitengine (Go game library)"
		}
	}
	return ""
}
