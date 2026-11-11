package codie

// purpose.go: "What it does": a look at the code as a whole. Codie matches the
// clues in the code (the libraries and calls it uses, its own names and
// comments, where it starts and what it reads and writes) against common kinds
// of program, then describes the program's job and how it flows, showing the
// clues so you can check its reasoning. It's pattern matching, not AI, so it
// says how sure it is. Created by nrmluvdev.

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
)

type Clue struct {
	Line int    `json:"line"`
	Code string `json:"code"`
	Why  string `json:"why"`
}

type PurposeKind struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	About string `json:"about"`
	Score int    `json:"score"`
	Clues []Clue `json:"clues"`
}

type FlowStep struct {
	Step  string `json:"step"` // "Starts", "Takes in", "Works", "Gives out"
	Text  string `json:"text"`
	Clues []Clue `json:"clues"`
}

type Purpose struct {
	Headline string        `json:"headline"`
	Sure     string        `json:"sure"` // "fairly sure", "best guess", "not sure"
	Kinds    []PurposeKind `json:"kinds"`
	Also     []string      `json:"also"` // smaller jobs it does too
	Flow     []FlowStep    `json:"flow"`
	Names    string        `json:"names"`   // what the code's own names suggest
	Comment  string        `json:"comment"` // its first comment, often a description
}

// a clue: a pattern in the code, how much it counts, and why it points that way
type psig struct {
	re    *regexp.Regexp
	w     int
	why   string
	langs string // "" = any; else space-separated language ids
}

type kindDef struct {
	id, title, about string
	words            []string // words in names and comments that point here
	sigs             []psig
}

func ps(w int, pat, why string) psig { return psig{re: regexp.MustCompile(pat), w: w, why: why} }

var kindDefs = []kindDef{
	{"micro", "a microcontroller program", "a program for a small board such as an Arduino, ESP32 or Raspberry Pi Pico, which controls electronics: lights, sensors, motors and so on",
		[]string{"led", "sensor", "pin", "arduino", "esp32", "pico", "blink", "servo", "motor"},
		[]psig{ps(4, `\bvoid\s+setup\s*\(`, "Arduino sketches start in setup()"), ps(4, `\bvoid\s+loop\s*\(`, "Arduino sketches repeat loop() forever"),
			ps(3, `\bpinMode\s*\(`, "pinMode sets a board pin as an input or output"), ps(3, `\bdigital(Write|Read)\s*\(|\banalog(Read|Write)\s*\(`, "it switches or reads the board's pins"),
			ps(4, `\bfrom\s+machine\s+import|\bimport\s+machine\b|machine\.Pin\(`, "MicroPython's machine module controls a board's pins"),
			ps(3, `\bSerial\.begin\s*\(`, "Serial.begin starts the USB link to the computer"), ps(2, `\bboard\.(D|GP|A)\d+|\bimport\s+board\b|digitalio`, "CircuitPython's board module names the pins")}},
	{"keyboard", "keyboard firmware", "the program inside a keyboard, which works out which keys are pressed and sends them to the computer",
		[]string{"keymap", "keyboard", "layer", "macro", "keycode"},
		[]psig{ps(5, `\bKC_[A-Z0-9_]+\b|\bLAYOUT\w*\s*\(|keymaps\s*\[`, "KC_ keycodes and keymaps are QMK keyboard firmware"), ps(4, `\bKeyboard\.(press|release|write|begin)\s*\(`, "Keyboard.press sends key presses to the computer"),
			ps(4, `\bkmk\b|KMKKeyboard`, "KMK is keyboard firmware for CircuitPython"), ps(3, `&kp\s+[A-Z]`, "&kp bindings are ZMK keyboard firmware")}},
	{"game", "a game", "a game: it reads the player's input, updates the game world and draws it, many times a second", []string{"game", "player", "enemy", "score", "level", "sprite"}, nil},
	{"webserver", "a web server (the server side of a website or app)", "a program that waits for web requests and sends back pages or data, so a browser or another program can use it",
		[]string{"server", "route", "endpoint", "api", "request", "handler"},
		[]psig{ps(5, `\bhttp\.(ListenAndServe|HandleFunc|NewServeMux)\b`, "Go's net/http serves web requests"), ps(5, `\bexpress\s*\(\s*\)|\bapp\.(get|post|put|delete|listen)\s*\(`, "Express routes web requests in Node.js"),
			ps(5, `@app\.(route|get|post)\b|\bFlask\s*\(|\bFastAPI\s*\(`, "Flask and FastAPI route web requests in Python"), ps(4, `\bhttp\.createServer\b|\bcreateServer\s*\(`, "createServer answers web requests"),
			ps(4, `@(Get|Post|Request)Mapping|@RestController|\bMapGet\s*\(|\bWebApplication\b`, "these mark web endpoints in Java or C#"), ps(3, `\bHTTPServer\b|\bBaseHTTPRequestHandler\b`, "Python's http.server answers web requests"),
			ps(4, `\$_(POST|GET|REQUEST|SERVER)\b`, "PHP reads what a web form or address sent")}},
	{"webpage", "an interactive web page", "code that runs in a web browser: it reacts to clicks and typing and changes what's on the page",
		[]string{"page", "button", "click", "form", "dom", "ui"},
		[]psig{ps(4, `\bdocument\.(getElementById|querySelector(All)?|createElement)\b`, "it finds or makes parts of the web page"), ps(3, `\.addEventListener\s*\(\s*["']?`, "it reacts to clicks, typing and other events"),
			ps(3, `\.innerHTML\b|\.textContent\b|\.classList\.`, "it changes what the page shows"), ps(4, `\buse(State|Effect)\s*\(|\bReactDOM\b|\bcreateApp\s*\(`, "React or Vue builds the page from components"),
			ps(4, `(?i)<(html|body|div|button|form)\b`, "HTML is the structure of a web page")}},
	{"fetch", "a program that gets data from the internet", "a program that asks a website or online service (an API) for data and uses what comes back",
		[]string{"api", "fetch", "download", "weather", "url", "json"},
		[]psig{ps(4, `\brequests\.(get|post)\s*\(|\burllib\.request\b|\burlopen\s*\(`, "it fetches a web address from Python"), ps(4, `\bfetch\s*\(|\baxios\.(get|post)\b`, "fetch / axios ask a web address for data"),
			ps(4, `\bhttp\.(Get|Post)\s*\(|\bHttpClient\b|\bcurl_exec\b|\bInvoke-(RestMethod|WebRequest)\b`, "it requests a web address"), ps(2, `\bjson\.(loads|load)\b|\bJSON\.parse\b|\.json\(\)`, "it reads JSON, the usual format of web data")}},
	{"scraper", "a web scraper", "a program that downloads web pages and picks information out of them", []string{"scrape", "crawl", "parse", "html"},
		[]psig{ps(5, `\bBeautifulSoup\b|\bsoup\.(find|select)`, "BeautifulSoup picks information out of web pages"), ps(5, `\bscrapy\b|\bcheerio\b|\bpuppeteer\b|\bselenium\b`, "these tools read or drive web pages")}},
	{"data", "a data-analysis script", "a script that loads a table of data, works out figures from it and often draws charts",
		[]string{"data", "csv", "chart", "plot", "average", "stats"},
		[]psig{ps(5, `\bimport\s+pandas\b|\bpd\.read_(csv|excel)\b|\bdf\.(groupby|describe|plot)\b`, "pandas works with tables of data"), ps(4, `\bmatplotlib\b|\bplt\.(plot|show|bar)\b`, "matplotlib draws charts"),
			ps(3, `\bimport\s+numpy\b|\bnp\.(array|mean|sum)\b`, "numpy does maths on arrays of numbers")}},
	{"ml", "a machine-learning program", "a program that trains a model on examples so it can make predictions", []string{"model", "train", "predict", "neural"},
		[]psig{ps(5, `\bsklearn\b|\btensorflow\b|\bimport\s+torch\b|\bkeras\b`, "these are machine-learning libraries"), ps(3, `\.fit\s*\(|\.predict\s*\(`, "fit trains a model; predict uses it")}},
	{"files", "a file-processing script", "a script that reads files, changes or sorts out what's in them, and writes the results", []string{"file", "folder", "rename", "csv", "log", "backup"},
		[]psig{ps(3, `\bopen\s*\([^)]*\)|\bwith\s+open\b`, "it opens files"), ps(3, `\bos\.(listdir|walk|rename|remove)\b|\bglob\.glob\b|\bshutil\.`, "it works through folders of files"),
			ps(3, `\bFile\.(ReadAll|WriteAll|Open|Copy|Move)\w*|\bos\.(ReadFile|WriteFile|Create|Open)\b|\bfs\.(readFile|writeFile)\w*`, "it reads or writes files"), ps(2, `\bcsv\.(reader|writer|DictReader)\b`, "it reads or writes CSV tables"),
			ps(3, `\bGet-ChildItem\b|\bCopy-Item\b|\bfor\s+%%?\w\s+in\b`, "it works through files and folders")}},
	{"database", "a database program", "a program that stores and looks up information in a database", []string{"db", "database", "table", "record", "query"},
		[]psig{ps(4, `(?i)\b(SELECT\s+.+\s+FROM|INSERT\s+INTO|CREATE\s+TABLE|UPDATE\s+\w+\s+SET)\b`, "SQL reads and changes database tables"), ps(4, `\bsqlite3\b|\bcursor\.execute\b|\bmongoose\b|\bSqlConnection\b`, "it connects to a database")}},
	{"gui", "a desktop app with windows and buttons", "a program with its own window, buttons and boxes to type in", []string{"window", "button", "gui", "app", "form"},
		[]psig{ps(5, `\btkinter\b|\bTk\s*\(\s*\)|\bPyQt\d?\b|\bQApplication\b`, "tkinter and Qt make windows in Python"), ps(5, `\bJFrame\b|\bApplication\.Run\s*\(|\bSystem\.Windows\.Forms\b|\bWPF\b|\bGtk\.`, "these make desktop windows"),
			ps(3, `\.mainloop\s*\(`, "mainloop keeps the window open and answers clicks")}},
	{"bot", "a chat bot", "a bot that joins a chat service (such as Discord or Telegram) and answers messages or commands", []string{"bot", "discord", "telegram", "command", "chat"},
		[]psig{ps(5, `\bdiscord\.(Client|Intents)|\bcommands\.Bot\b|\btelebot\b|\bTelegramBot|\bslack_bolt\b`, "it connects to a chat service as a bot"), ps(3, `@\w+\.(command|event)\b|\bon_message\b`, "it answers messages and commands")}},
	{"network", "a network program", "a program that talks to other computers directly over the network", []string{"socket", "client", "server", "port"},
		[]psig{ps(4, `\bsocket\.socket\b|\bnet\.(Dial|Listen)\b|\bTcp(Client|Listener)\b|\bnew\s+Socket\b`, "sockets send data between computers"), ps(2, `\.(bind|connect|accept|recv|send)\s*\(`, "it connects, listens or sends")}},
	{"automation", "an automation script", "a script that does a chore for you: runs other programs, moves files, or sets things up", []string{"backup", "deploy", "install", "clean", "setup", "build"},
		[]psig{ps(3, `\bsubprocess\.(run|call|Popen)\b|\bos\.system\b|\bexec\.Command\b|\bStart-Process\b`, "it runs other programs"), ps(2, `\bschedule\.\w+|\bcron\b|\btime\.sleep\s*\(\s*\d{2,}`, "it runs on a timer")}},
	{"cli", "a command-line tool", "a tool you run from the command line, giving it settings when you start it", []string{"cli", "tool", "args", "options", "command"},
		[]psig{ps(4, `\bargparse\b|\bflag\.(Parse|String|Int|Bool)\b|\bprocess\.argv\b|\bsys\.argv\b|\bos\.Args\b|\bgetopt\b|\bclap::`, "it reads the settings typed after the program's name"), ps(2, `\bstring\[\]\s*args\b|\bargc\b`, "main receives the words typed after the program's name")}},
	{"interactive", "an interactive console program", "a program that asks you questions in a text window and answers you", []string{"quiz", "guess", "menu", "ask", "answer", "calculator"},
		[]psig{ps(3, `\binput\s*\(|\braw_input\s*\(`, "input() asks the person a question"), ps(3, `\bscanf\s*\(|\bcin\s*>>|\bConsole\.ReadLine\s*\(|\breadLine\s*\(|\bgets\b|\bRead-Host\b|\bset\s+/p\b`, "it reads what the person types"),
			ps(3, `(?i)^\s*\d*\s*INPUT\b`, "INPUT asks the person to type something")}},
	{"maths", "a maths or algorithm program", "a program that works something out: a calculation, a sequence of numbers, or a classic algorithm such as sorting or searching", []string{"sort", "search", "prime", "fib", "factorial", "sum", "average", "calc", "gcd", "matrix"},
		[]psig{ps(1, `[-+*/%]\s*\w`, "it does arithmetic"), ps(2, `\bmath\.\w+|\bMath\.\w+`, "it uses maths functions")}},
	{"library", "a set of building blocks (a library or module)", "code that defines functions or classes for other programs to use, rather than doing a job on its own", []string{"util", "helper", "lib", "module"}, nil},
	{"style", "a style sheet", "CSS that decides how a web page looks: colours, sizes and layout", nil, []psig{ps(4, `[.#]?[\w-]+\s*\{|^\s*[\w-]+\s*:\s*[^;]+;`, "CSS rules style the page")}},
}

// smaller jobs worth mentioning whatever the program is
var jobs = []struct {
	re   *regexp.Regexp
	text string
}{
	{regexp.MustCompile(`\b(analogRead|readTemperature|readHumidity|dht\.|DHT\d*|DS18B20|bme\.|bmp\.|ultrasonic|pulseIn)\b`), "reads a sensor"},
	{regexp.MustCompile(`\bdigitalWrite\s*\(.*\b(LED|led|13|HIGH)\b|\bled\.(on|off|toggle|value)\b|LED_BUILTIN`), "switches an LED"},
	{regexp.MustCompile(`\bServo\b|\.write\s*\(\s*\d+\s*\)|\bstepper\b|\bmotor\b`), "moves a motor or servo"},
	{regexp.MustCompile(`\b(lcd|oled|display|tft|screen)\.(print|show|drawString|setCursor|clear)`), "shows things on a small screen"},
	{regexp.MustCompile(`\bWiFi\.(begin|connect)|\bnetwork\.WLAN\b|\bwlan\.connect\b`), "connects to Wi-Fi"},
	{regexp.MustCompile(`\b(Serial\.print|Serial\.println)\b`), "sends messages to the computer over USB"},
	{regexp.MustCompile(`\bdelay\s*\(|\bsleep\s*\(|\btime\.sleep\b|\butime\.sleep`), "waits between steps"},
	{regexp.MustCompile(`\brandom\b|\bRandom\b|\brand\s*\(|\bMath\.random\b|\brandint\b|\bRND\b`), "uses random numbers"},
	{regexp.MustCompile(`\b(sort|sorted|Sort|qsort|sort_by)\s*\(|\.sort\s*\(`), "sorts things into order"},
	{regexp.MustCompile(`\btry\b|\bexcept\b|\bcatch\b|\berr\s*!=\s*nil\b`), "handles errors"},
	{regexp.MustCompile(`\blocalStorage\b|\bpickle\b|\bjson\.dump\b|\bJSON\.stringify\b|\bsave\w*\s*\(`), "saves information for later"},
	{regexp.MustCompile(`(?i)\bsmtplib\b|\bsendmail\b|\bnodemailer\b|\bSend-MailMessage\b|\bmail\s*\(`), "sends email"},
}

func clueOf(l srcLine, why string) Clue {
	c := strings.TrimSpace(l.Text)
	if len(c) > 90 {
		c = c[:87] + "…"
	}
	return Clue{Line: l.N, Code: c, Why: why}
}

// splitName turns readTemperature / read_temperature into "read temperature"
func splitName(n string) string {
	var b strings.Builder
	rs := []rune(n)
	for i, r := range rs {
		if r == '_' || r == '-' {
			b.WriteRune(' ')
			continue
		}
		if i > 0 && unicode.IsUpper(r) && (unicode.IsLower(rs[i-1]) || (i+1 < len(rs) && unicode.IsLower(rs[i+1]) && unicode.IsUpper(rs[i-1]))) {
			b.WriteRune(' ')
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

var commonNames = map[string]bool{"main": true, "init": true, "setup": true, "loop": true, "run": true, "start": true, "update": true, "draw": true, "test": true, "constructor": true, "__init__": true, "new": true}

func analysePurpose(rep Report, src []srcLine) Purpose {
	var p Purpose
	lang := rep.Lang.ID
	// words from the code's own names and comments
	var names []string
	words := map[string]int{}
	addWords := func(s string, w int) {
		for _, t := range strings.Fields(strings.ToLower(regexp.MustCompile(`[^A-Za-z]+`).ReplaceAllString(splitName(s), " "))) {
			words[t] += w
		}
	}
	for _, o := range rep.Outline {
		if (o.Kind == "function" || o.Kind == "method" || o.Kind == "class" || o.Kind == "event") && o.Name != "" {
			addWords(o.Name, 2)
			if !commonNames[strings.ToLower(o.Name)] && len(names) < 8 {
				names = append(names, splitName(o.Name))
			}
		}
	}
	// the first real comment often says what the program is for
	for _, l := range src {
		if strings.TrimSpace(l.Code) != "" && !l.InBlock {
			if p.Comment != "" {
				break
			}
			continue
		}
		c := strings.TrimSpace(strings.Trim(strings.TrimSpace(l.Comment), "/*#-=!;"))
		if c == "" || strings.HasPrefix(strings.ToLower(c), "!") || strings.Contains(c, "coding:") || strings.HasPrefix(c, "@") {
			continue
		}
		if p.Comment != "" {
			p.Comment += " "
		}
		p.Comment += c
		if len(p.Comment) > 220 {
			p.Comment = p.Comment[:217] + "…"
			break
		}
	}
	addWords(p.Comment, 1)
	if len(names) > 0 {
		p.Names = "Its own names talk about: " + strings.Join(names, ", ") + "."
	}

	// score each kind of program
	var kinds []PurposeKind
	for _, k := range kindDefs {
		pk := PurposeKind{ID: k.id, Title: k.title, About: k.about}
		if k.id == "style" && lang != "css" { // braces and colons are everywhere else
			kinds = append(kinds, pk)
			continue
		}
		for _, sg := range k.sigs {
			hits := 0
			for _, l := range src {
				code := l.Code
				if k.id == "webpage" && lang == "html" {
					code = l.Text
				}
				if code == "" || !sg.re.MatchString(code) {
					continue
				}
				if hits == 0 {
					pk.Clues = append(pk.Clues, clueOf(l, sg.why))
				}
				hits++
			}
			if hits > 0 {
				pk.Score += sg.w + min(hits-1, 3) // more uses count a little more
			}
		}
		for _, w := range k.words {
			if words[w] > 0 {
				pk.Score += min(words[w], 3)
			}
		}
		kinds = append(kinds, pk)
	}
	byID := map[string]*PurposeKind{}
	for i := range kinds {
		byID[kinds[i].ID] = &kinds[i]
	}
	// what other parts of Codie already know
	// Codie's game tab treats an Arduino sketch's setup()/loop() as a game loop;
	// that alone doesn't make it a game
	arduinoOnly := rep.Game.Engine != nil && rep.Game.Engine.ID == "arduino" && len(rep.Game.Hits) < 2
	if g := rep.Game; (g.Engine != nil || g.IsGame) && !arduinoOnly {
		k := byID["game"]
		k.Score += 6
		if g.Engine != nil {
			k.Score += 4
			k.Clues = append(k.Clues, Clue{Why: "it's written for " + g.Engine.Name + ", a game engine"})
		} else {
			k.Clues = append(k.Clues, Clue{Why: "it has game logic: " + joinAnd(gameTitles(g.Hits, 3))})
		}
	}
	switch lang {
	case "css":
		byID["style"].Score += 6
	case "html":
		byID["webpage"].Score += 3
	case "sql":
		byID["database"].Score += 6
	}
	// only definitions, nothing that runs: a library
	if rep.Stats.Functions+rep.Stats.Classes >= 2 && !hasEntry(src, lang) {
		byID["library"].Score += 4
		byID["library"].Clues = append(byID["library"].Clues, Clue{Why: "it defines functions or classes but has no starting point of its own"})
	}
	// a bit of arithmetic is everywhere: maths only counts when little else does
	if m := byID["maths"]; m.Score > 0 && len(m.Clues) > 0 && words["calc"]+words["prime"]+words["fib"]+words["factorial"]+words["sort"]+words["search"]+words["average"]+words["sum"] == 0 {
		m.Score = min(m.Score, 2)
	}
	if k := byID["micro"]; k.Score > 0 && byID["keyboard"].Score >= 5 {
		k.Score -= 3 // a keyboard's firmware runs on a microcontroller: the keyboard is the better answer
	}
	sort.SliceStable(kinds, func(i, j int) bool { return kinds[i].Score > kinds[j].Score })
	for _, k := range kinds {
		if k.Score >= 3 && len(p.Kinds) < 3 {
			p.Kinds = append(p.Kinds, k)
		}
	}

	// smaller jobs
	seen := map[string]bool{}
	for _, l := range src {
		for _, j := range jobs {
			if !seen[j.text] && l.Code != "" && j.re.MatchString(l.Code) {
				seen[j.text] = true
				p.Also = append(p.Also, j.text)
			}
		}
	}
	p.Flow = flowOf(rep, src, lang)

	// the headline
	switch {
	case len(p.Kinds) == 0:
		p.Sure = "not sure"
		// no common kind matched: describe it from what it takes in and gives out
		var bits []string
		for _, f := range p.Flow {
			switch f.Step {
			case "Takes in":
				bits = append(bits, "takes in "+f.Text)
			case "Works":
				bits = append(bits, strings.TrimPrefix(f.Text, "it "))
			case "Gives out":
				bits = append(bits, "gives out "+f.Text)
			}
		}
		bits = append(bits, first(p.Also, 2)...)
		if len(bits) == 0 {
			p.Headline = "It doesn't match a common kind of program, and it's too short to say more."
		} else {
			p.Headline = "It doesn't match a common kind of program, but from its parts it " + joinAnd(first(bits, 4)) + "."
		}
	default:
		top := p.Kinds[0]
		switch {
		case top.Score >= 8 && (len(p.Kinds) == 1 || top.Score >= p.Kinds[1].Score+3):
			p.Sure = "fairly sure"
		default:
			p.Sure = "best guess"
		}
		h := "This is " + map[string]string{"fairly sure": "", "best guess": "probably "}[p.Sure] + top.Title
		if len(p.Kinds) > 1 && p.Kinds[1].Score >= 4 && p.Kinds[1].Score >= top.Score-2 && len(p.Kinds[1].Clues) > 0 && p.Kinds[1].ID != "library" {
			h += ", and also " + p.Kinds[1].Title
		}
		if len(p.Also) > 0 {
			h += " that " + joinAnd(first(p.Also, 3))
		}
		p.Headline = h + "."
	}
	return p
}

func first(s []string, n int) []string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

var entryRe = regexp.MustCompile(`\bfunc\s+main\s*\(|\b(int|void)\s+main\s*\(|static\s+void\s+Main\s*\(|public\s+static\s+void\s+main\s*\(|__name__\s*==\s*.__main__.|\bvoid\s+setup\s*\(|\bfn\s+main\s*\(|\bfun\s+main\s*\(`)

func hasEntry(src []srcLine, lang string) bool {
	top := 0
	for _, l := range src {
		c := l.Code
		if entryRe.MatchString(c) {
			return true
		}
		// scripts run their top-level lines (Python, JS, Lua, shell, BASIC…)
		if c != "" && len(l.Text) > 0 && l.Text[0] != ' ' && l.Text[0] != '\t' && !regexp.MustCompile(`^\s*(def|class|function|import|from|const|let|var|export|package|using|#include|local\s+function|}|\)|@|//)`).MatchString(c) {
			top++
		}
	}
	return top >= 2 && lang != "java" && lang != "csharp" && lang != "c" && lang != "cpp" && lang != "go" && lang != "rust"
}

// how the program flows: where it starts, what it takes in, what it does, what comes out
var (
	startRes = []struct {
		re  *regexp.Regexp
		txt string
	}{
		{regexp.MustCompile(`\bvoid\s+setup\s*\(`), "setup() runs once when the board powers on, then loop() runs over and over"},
		{regexp.MustCompile(`__name__\s*==\s*.__main__.`), "it starts at the if __name__ == \"__main__\" block when you run the file"},
		{regexp.MustCompile(`\bfunc\s+main\s*\(|\b(int|void)\s+main\s*\(|static\s+void\s+Main\s*\(|public\s+static\s+void\s+main\s*\(|\bfn\s+main\s*\(|\bfun\s+main\s*\(`), "it starts in main(), the first function the computer runs"},
		{regexp.MustCompile(`\.addEventListener\s*\(|\bonclick\b|\bonload\b|DOMContentLoaded`), "it waits for things to happen on the page (clicks, typing, loading) and runs code when they do"},
		{regexp.MustCompile(`@app\.(route|get|post)|\bapp\.(get|post)\s*\(|\bHandleFunc\s*\(`), "each part runs when a web request arrives at its address"},
		{regexp.MustCompile(`\.mainloop\s*\(|Application\.Run\s*\(`), "it opens its window, then waits for clicks and typing"},
	}
	inRes = []struct {
		re  *regexp.Regexp
		txt string
	}{
		{regexp.MustCompile(`\binput\s*\(|\bscanf\s*\(|\bcin\s*>>|\bConsole\.ReadLine|\breadLine\s*\(|\bRead-Host\b|(?i)^\s*\d*\s*INPUT\b`), "what the person types"},
		{regexp.MustCompile(`\bsys\.argv\b|\bos\.Args\b|\bprocess\.argv\b|\bargparse\b|\bflag\.Parse\b|\bargs\[`), "settings typed after the program's name"},
		{regexp.MustCompile(`\bopen\s*\(|\bread_csv\b|\bReadFile\b|\bFile\.Read|\bfs\.readFile|\bcsv\.reader\b|\bGet-Content\b`), "files"},
		{regexp.MustCompile(`\banalogRead\s*\(|\bdigitalRead\s*\(|\bread(Temperature|Humidity)\b|\.value\(\)|\bpulseIn\b`), "sensors and buttons on the board's pins"},
		{regexp.MustCompile(`\brequests\.get\b|\bfetch\s*\(|\bhttp\.Get\b|\burlopen\b|\baxios\.get\b|\bInvoke-RestMethod\b`), "data from the internet"},
		{regexp.MustCompile(`\.addEventListener\s*\(\s*.(click|key|input|change|mouse|touch|pointer)|\bonclick\b|\.value\b`), "clicks and typing on the page"},
		{regexp.MustCompile(`(?i)\bSELECT\b.+\bFROM\b|\bcursor\.execute\b|\.find\s*\(\s*\{`), "a database"},
		{regexp.MustCompile(`\brequest\.(args|form|json|body)\b|\breq\.(body|params|query)\b|\br\.(URL|Body|FormValue)\b`), "what web requests send it"},
		{regexp.MustCompile(`\bInput\.(GetKey|GetAxis|GetButton)|\bis_action_pressed|\bkeyboard_check|\bIsKeyDown\b|\bbtn\s*\(|\blove\.keyboard\b`), "the player's keys and controller"},
	}
	outRes = []struct {
		re  *regexp.Regexp
		txt string
	}{
		{regexp.MustCompile(`\bprint\s*\(|\bprintf\s*\(|\bprintln\s*[!(]|\bconsole\.log\b|\bcout\s*<<|\bConsole\.Write|\bfmt\.Print|\becho\b|\bputs\b|\bWrite-(Host|Output)\b|(?i)^\s*\d*\s*PRINT\b`), "text on the screen"},
		{regexp.MustCompile(`\bSerial\.print`), "messages to the computer over USB"},
		{regexp.MustCompile(`\bdigitalWrite\s*\(|\banalogWrite\s*\(|\bled\.(on|off|value|toggle)\b|\.duty`), "electricity to LEDs, motors or other parts on the board's pins"},
		{regexp.MustCompile(`\b(lcd|oled|display|tft|screen)\.(print|show|drawString)`), "a small screen"},
		{regexp.MustCompile(`\.innerHTML\b|\.textContent\s*=|\bdocument\.write\b|\.appendChild\s*\(|\bsetState\b|\bset[A-Z]\w*\s*\(`), "changes on the web page"},
		{regexp.MustCompile(`\bwrite\s*\(|\bWriteFile\b|\bFile\.Write|\bfs\.writeFile|\bto_csv\b|\bcsv\.writer\b|\bSet-Content\b|\bOut-File\b|>>?\s*\S+\.(txt|csv|log)`), "files"},
		{regexp.MustCompile(`\bplt\.(show|savefig)\b|\.plot\s*\(`), "charts"},
		{regexp.MustCompile(`\bres\.(send|json|render)\b|\bjsonify\b|\bw\.Write\b|\breturn\s+render_template\b|\bfmt\.Fprint`), "replies to web requests"},
		{regexp.MustCompile(`(?i)\bINSERT\s+INTO\b|\bUPDATE\s+\w+\s+SET\b|\.save\s*\(\s*\)|\bcommit\s*\(\s*\)`), "a database"},
		{regexp.MustCompile(`\bdraw\w*\s*\(|\bblit\b|\brender\s*\(|\bDraw\w*\s*\(|\bspr\s*\(|\blove\.graphics\b`), "pictures drawn on the screen"},
		{regexp.MustCompile(`\bKeyboard\.(press|write)\b|\bKC_`), "key presses sent to the computer"},
	}
)

func flowOf(rep Report, src []srcLine, lang string) []FlowStep {
	var flow []FlowStep
	find := func(re *regexp.Regexp) (Clue, bool) {
		for _, l := range src {
			if l.Code != "" && re.MatchString(l.Code) {
				return clueOf(l, ""), true
			}
		}
		return Clue{}, false
	}
	for _, s := range startRes {
		if c, ok := find(s.re); ok {
			flow = append(flow, FlowStep{Step: "Starts", Text: s.txt, Clues: []Clue{c}})
			break
		}
	}
	if len(flow) == 0 && rep.Game.Engine != nil {
		flow = append(flow, FlowStep{Step: "Starts", Text: rep.Game.Engine.Name + " starts it and calls its code every frame (see Game logic)"})
	}
	if len(flow) == 0 {
		flow = append(flow, FlowStep{Step: "Starts", Text: "it runs from the top, one line after another"})
	}
	gather := func(step string, list []struct {
		re  *regexp.Regexp
		txt string
	}) {
		var txt []string
		var cl []Clue
		for _, s := range list {
			if c, ok := find(s.re); ok {
				txt = append(txt, s.txt)
				cl = append(cl, c)
			}
		}
		if len(txt) > 0 {
			flow = append(flow, FlowStep{Step: step, Text: joinAnd(first(txt, 3)), Clues: first2(cl, 3)})
		}
	}
	gather("Takes in", inRes)
	// the work in between: what the code is built from
	var work []string
	if rep.Stats.Loops > 0 {
		work = append(work, "repeats steps in "+countA(rep.Stats.Loops, "loop"))
	}
	for _, c := range rep.Concepts {
		switch c.ID {
		case "if", "switch":
			work = append(work, "makes decisions")
		case "list", "array", "table":
			work = append(work, "keeps lists of things")
		case "maths":
			work = append(work, "does calculations")
		case "class":
			work = append(work, "groups data and actions into objects")
		case "timers":
			work = append(work, "times things")
		}
	}
	if rep.Stats.Functions > 0 {
		if rep.Stats.Functions == 1 {
			work = append(work, "has one function")
		} else {
			work = append(work, "is split into "+plural(rep.Stats.Functions, "function"))
		}
	}
	if len(work) > 0 {
		flow = append(flow, FlowStep{Step: "Works", Text: "it " + joinAnd(dedupe(first(work, 4)))})
	}
	gather("Gives out", outRes)
	return flow
}

// countA: "a loop", "3 loops"
func countA(n int, word string) string {
	if n == 1 {
		return "a " + word
	}
	return plural(n, word)
}

func first2(c []Clue, n int) []Clue {
	if len(c) > n {
		return c[:n]
	}
	return c
}

func dedupe(s []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, x := range s {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	return out
}
