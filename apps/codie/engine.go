// Codie — engine.go: reads code line by line and builds the report: what each
// line does and why, the outline of functions, classes and loops, the ideas
// (concepts) the code uses, warnings about common mistakes, and the game logic.
// Part of the free nrmluvdev maker suite. Created by nrmluvdev.
package codie

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// MaxCode is the most code Codie reads in one go.
const MaxCode = 512 << 10

// Part is one statement's explanation. Most lines have one part; a BASIC line
// such as 10 CLS : PRINT "HI" : GOTO 10 has three.
type Part struct {
	Code string `json:"code"`
	Kind string `json:"kind"`
	What string `json:"what"`
	Why  string `json:"why,omitempty"`
}

type LineOut struct {
	N     int      `json:"n"`
	End   int      `json:"end,omitempty"` // for a run of comment lines
	Text  string   `json:"text"`
	Kind  string   `json:"kind"`
	Parts []Part   `json:"parts"`
	Notes []string `json:"notes,omitempty"`
	Tags  []string `json:"tags,omitempty"`
	Game  []string `json:"game,omitempty"`
	Depth int      `json:"depth"`
	Known bool     `json:"known"` // explained by a real rule, not a guess
}

type Outline struct {
	Kind  string `json:"kind"`
	Name  string `json:"name"`
	Start int    `json:"start"`
	End   int    `json:"end"`
	Depth int    `json:"depth"`
	Note  string `json:"note,omitempty"`
}

type Warning struct {
	Line int    `json:"line"`
	Text string `json:"text"`
	Fix  string `json:"fix,omitempty"`
}

type ConceptUse struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Text  string `json:"text"`
	Lines []int  `json:"lines"`
	Game  bool   `json:"game,omitempty"`
}

type Stats struct {
	Lines     int `json:"lines"`
	Code      int `json:"code"`
	Comments  int `json:"comments"`
	Blank     int `json:"blank"`
	Explained int `json:"explained"`
	Functions int `json:"functions"`
	Classes   int `json:"classes"`
	Loops     int `json:"loops"`
}

type Report struct {
	Lang     Detection    `json:"lang"`
	Summary  []string     `json:"summary"`
	Lines    []LineOut    `json:"lines"`
	Outline  []Outline    `json:"outline"`
	Concepts []ConceptUse `json:"concepts"`
	Warnings []Warning    `json:"warnings"`
	Game     GameReport   `json:"game"`
	Purpose  Purpose      `json:"purpose"`
	Stats    Stats        `json:"stats"`
}

// ------------------------------------------------------------- rules

// Rule explains statements that match a pattern.
type Rule struct {
	langs map[string]bool
	all   bool
	re    *regexp.Regexp
	Kind  string
	What  string
	Why   string
	Tags  []string
	Open  string // the kind of block this statement starts
	Name  int    // capture holding the block's or variable's name
	Close bool   // ends a block (keyword languages)
	Var   int    // capture holding a variable being set
	fn    func(m []string, x *ctx) (what, why string)
	prio  int
}

var rules []*Rule

// macros used in rule patterns
var macro = strings.NewReplacer(
	"{S}", STR,
	"{I}", `[A-Za-z_]\w*`,
	"{D}", `[A-Za-z_$@][\w$]*(?:(?:\.|::|->|:)[A-Za-z_$][\w$]*)*`,
	"{T}", `[A-Za-z_][\w.:]*(?:<[^()]*?>)?(?:\[\])*\**&?`,
)

// R adds a rule. langs is a comma list of language ids, @family names, or *.
func R(langs, pattern, kind, what, why string, tags ...string) *Rule {
	r := &Rule{langs: map[string]bool{}, Kind: kind, What: what, Why: why, Tags: tags}
	nocase := false
	for _, l := range strings.Split(langs, ",") {
		l = strings.TrimSpace(l)
		switch {
		case l == "*":
			r.all, r.prio = true, 2
		case strings.HasPrefix(l, "@"):
			r.prio = 1
			for _, x := range langs0() {
				if x.Family == l[1:] {
					r.langs[x.ID] = true
					nocase = nocase || x.NoCase
				}
			}
		default:
			r.langs[l] = true
			for _, x := range langs0() {
				if x.ID == l && x.NoCase {
					nocase = true
				}
			}
		}
	}
	p := macro.Replace(pattern)
	if nocase {
		p = "(?i)" + p
	}
	r.re = regexp.MustCompile(p)
	rules = append(rules, r)
	return r
}

func langs0() []*Lang { return langs }

func (r *Rule) open(kind string, name int) *Rule                   { r.Open, r.Name = kind, name; return r }
func (r *Rule) close() *Rule                                       { r.Close = true; return r }
func (r *Rule) setsVar(i int) *Rule                                { r.Var = i; return r }
func (r *Rule) with(f func([]string, *ctx) (string, string)) *Rule { r.fn = f; return r }
func (r *Rule) first() *Rule                                       { r.prio = -1; return r }

func (r *Rule) applies(id string) bool { return r.all || r.langs[id] }

var sortOnce bool

func sortedRules() []*Rule {
	if !sortOnce {
		sort.SliceStable(rules, func(i, j int) bool { return rules[i].prio < rules[j].prio })
		sortOnce = true
	}
	return rules
}

// ------------------------------------------------------------- context

type block struct {
	Kind, Name string
	Start, End int
	Depth      int
	Indent     int
	outlineIdx int
}

type ctx struct {
	lang      *Lang
	dialect   string
	funcs     map[string]int
	vars      map[string]int
	stack     []*block
	outline   []Outline
	n         int
	strs      []string
	tags      []string
	notes     []string
	pending   string // C-style: the last line opened this kind but its { is on the next line
	pendName  string
	pendLine  int
	first     bool           // first pass: only collect names
	cur       string         // the statement being explained, as typed
	basicLine int            // BASIC: this line's number
	numNote   bool           // BASIC: line numbers explained already
	skipPat   string         // notes for names in this pattern would repeat the explanation
	basicVals map[string]int // BASIC: variables set to plain numbers, for working out POKE addresses
}

func (x *ctx) push(kind, name string, line, indent int) {
	b := &block{Kind: kind, Name: name, Start: line, Depth: len(x.stack), Indent: indent, outlineIdx: -1}
	if kind != "" && kind != "block" {
		b.outlineIdx = len(x.outline)
		x.outline = append(x.outline, Outline{Kind: kind, Name: name, Start: line, End: line, Depth: len(x.stack)})
	}
	x.stack = append(x.stack, b)
}

func (x *ctx) pop(end int) *block {
	if len(x.stack) == 0 {
		return nil
	}
	b := x.stack[len(x.stack)-1]
	x.stack = x.stack[:len(x.stack)-1]
	b.End = end
	if b.outlineIdx >= 0 {
		x.outline[b.outlineIdx].End = end
	}
	return b
}

func (x *ctx) tag(t ...string) { x.tags = append(x.tags, t...) }

// ------------------------------------------------------------- analysis

// Analyse explains code. hint is a language id, a file name or "".
func Analyse(code, hint string) Report {
	if len(code) > MaxCode {
		code = code[:MaxCode]
	}
	det := Detect(code, hint)
	l := langByID[det.ID]
	src := lexLines(code, l)
	// First pass collects the names of functions and variables, so a call on
	// line 3 to a function written on line 40 can be explained.
	pre := &ctx{lang: l, dialect: det.Dialect, funcs: map[string]int{}, vars: map[string]int{}, basicVals: map[string]int{}, first: true}
	run(pre, src)
	x := &ctx{lang: l, dialect: det.Dialect, funcs: pre.funcs, vars: map[string]int{}, basicVals: map[string]int{}}
	lines := run(x, src)

	rep := Report{Lang: det, Lines: lines, Outline: x.outline}
	if rep.Outline == nil {
		rep.Outline = []Outline{}
	}
	// stats
	rep.Stats.Lines = len(src)
	for _, s := range src {
		switch {
		case strings.TrimSpace(s.Text) == "":
			rep.Stats.Blank++
		case s.InBlock || (s.Code == "" && s.Comment != ""):
			rep.Stats.Comments++
		default:
			rep.Stats.Code++
		}
	}
	for _, ln := range lines {
		if ln.Known && ln.Kind != "comment" {
			rep.Stats.Explained++
		}
	}
	for _, o := range rep.Outline {
		switch o.Kind {
		case "function", "method", "event":
			rep.Stats.Functions++
		case "class":
			rep.Stats.Classes++
		case "loop":
			rep.Stats.Loops++
		}
	}
	rep.Concepts = collectConcepts(lines)
	rep.Warnings = findWarnings(l, src)
	rep.Game = analyseGame(det, code, lines, rep.Outline)
	rep.Purpose = analysePurpose(rep, src)
	rep.Summary = summarise(rep)
	return rep
}

func run(x *ctx, src []srcLine) []LineOut {
	var out []LineOut
	prevCode, prevLine := "", 0
	for _, s := range src {
		x.n = s.N
		x.strs = s.Strs
		x.tags, x.notes = nil, nil
		trim := strings.TrimSpace(s.Text)
		if trim == "" {
			continue
		}
		depth := len(x.stack)
		if s.InBlock {
			out = addComment(out, s, depth, s.Comment)
			continue
		}
		if s.InString > 0 {
			out = append(out, LineOut{N: s.N, Text: s.Text, Kind: "string", Depth: depth, Known: true,
				Parts: []Part{{Code: trim, Kind: "string", What: fmt.Sprintf("Still inside the long text value that began on line %d.", s.InString), Why: "Some languages let text run over several lines, for messages, help text or embedded data."}}})
			continue
		}
		code := strings.TrimSpace(s.Code)
		if code == "" {
			if p, ok := specialComment(x, trim); ok {
				out = append(out, LineOut{N: s.N, Text: s.Text, Kind: p.Kind, Parts: []Part{p}, Depth: depth, Known: true, Tags: x.tags})
				continue
			}
			out = addComment(out, s, depth, s.Comment)
			continue
		}
		// Python and GDScript: blocks end when the indentation goes back.
		if x.lang.Family == "py" {
			ind := indentOf(s.Text)
			for len(x.stack) > 0 && x.stack[len(x.stack)-1].Indent >= ind {
				x.pop(lastCodeLine(out, s.N))
			}
			depth = len(x.stack)
		}
		lo := explainLine(x, s, code)
		if !lo.Known && continues(x.lang, prevCode, code) {
			lo.Parts = []Part{{Code: strings.TrimSpace(unhide(code, s.Strs)), Kind: "cont", What: fmt.Sprintf("Continues the statement from line %d.", prevLine), Why: "Long statements can be split over several lines to keep them readable."}}
			lo.Kind, lo.Known = "cont", true
		}
		prevCode, prevLine = code, s.N
		lo.Depth = depth
		if x.lang.Family != "py" && x.lang.Family != "c" && len(x.stack) < depth {
			lo.Depth = len(x.stack) // a closing keyword sits at the outer level
		}
		if x.lang.Family == "c" || x.lang.ID == "powershell" || x.lang.ID == "css" {
			if strings.HasPrefix(code, "}") {
				lo.Depth = len(x.stack)
			}
		}
		if s.Comment != "" {
			lo.Notes = append(lo.Notes, "The comment at the end says: “"+clip(s.Comment, 120)+"”")
		}
		out = append(out, lo)
	}
	// Blocks still open at the end (Python indentation, a missing "end") finish at the last line.
	for len(x.stack) > 0 {
		x.pop(lastCodeLine(out, len(src)+1))
	}
	return out
}

var (
	reContEnd   = regexp.MustCompile(`(&&|\|\||[-+*/%,(\[=?:<>]|\band|\bor|\.\.|\\|\|)\s*$`)
	reContStart = regexp.MustCompile(`^(&&|\|\||\.[A-Za-z_]|[)\]?:+*/]|\band\b|\bor\b|\.\.)`)
)

// continues reports whether a line carries on the statement above it.
func continues(l *Lang, prev, cur string) bool {
	if prev == "" {
		return false
	}
	prev = strings.TrimSpace(prev)
	if l.Family == "py" && strings.HasSuffix(prev, ":") {
		return false
	}
	return reContEnd.MatchString(prev) || reContStart.MatchString(strings.TrimSpace(cur))
}

func lastCodeLine(out []LineOut, cur int) int {
	for i := len(out) - 1; i >= 0; i-- {
		if out[i].Kind != "comment" {
			if out[i].End > 0 {
				return out[i].End
			}
			return out[i].N
		}
	}
	return cur - 1
}

func indentOf(t string) int {
	n := 0
	for _, c := range t {
		if c == ' ' {
			n++
		} else {
			break
		}
	}
	return n
}

func addComment(out []LineOut, s srcLine, depth int, text string) []LineOut {
	if k := len(out) - 1; k >= 0 && out[k].Kind == "comment" && (max(out[k].N, out[k].End) == s.N-1) {
		p := &out[k].Parts[0]
		if text != "" {
			p.Code += "\n" + strings.TrimSpace(s.Text)
			if len(p.What) < 600 {
				p.What = strings.TrimSuffix(p.What, "”") + " " + clip(text, 200) + "”"
			}
		}
		out[k].End = s.N
		return out
	}
	what := "A comment."
	if text != "" {
		what = "A comment: “" + clip(text, 200) + "”"
	}
	why := "Comments are notes for people reading the code. The computer skips them completely."
	if strings.Contains(strings.ToUpper(text), "TODO") || strings.Contains(strings.ToUpper(text), "FIXME") {
		why = "A reminder the author left: something still to do or fix here. The computer skips comments."
	}
	return append(out, LineOut{N: s.N, Text: s.Text, Kind: "comment", Depth: depth, Known: true,
		Parts: []Part{{Code: strings.TrimSpace(s.Text), Kind: "comment", What: what, Why: why}}})
}

// specialComment handles lines that look like comments but mean something.
func specialComment(x *ctx, t string) (Part, bool) {
	switch {
	case strings.HasPrefix(t, "#!"):
		return Part{Code: t, Kind: "directive", What: "The “shebang” line: tells Linux or macOS which program runs this script (" + strings.TrimSpace(t[2:]) + ").", Why: "With it, the script can be started directly (./script) once it is marked as runnable with chmod +x."}, true
	case x.lang.ID == "go" && (strings.HasPrefix(t, "//go:build") || strings.HasPrefix(t, "// +build")):
		cond := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(t, "//go:build"), "// +build"))
		x.tag("build-tags")
		why := "Go leaves a file out of the build when its condition is false, so one project can have separate files for each system."
		if strings.Contains(cond, "windows") {
			why = "This is how Windows-only Go is kept apart: the file is compiled only when building for Windows (GOOS=windows), so it may call Windows DLLs. A partner file marked //go:build !windows usually does the same job on Linux and macOS."
			x.tag("windows-go")
		}
		return Part{Code: t, Kind: "directive", What: "A build constraint: this file is only compiled when “" + cond + "” is true.", Why: why}, true
	case x.lang.ID == "go" && strings.HasPrefix(t, "//go:embed"):
		x.tag("embed")
		return Part{Code: t, Kind: "directive", What: "Packs the file(s) " + strings.TrimSpace(t[10:]) + " into the program itself, into the variable on the next line.", Why: "The finished program then needs no separate files: web pages, pictures or data travel inside the .exe."}, true
	case x.lang.ID == "go" && strings.HasPrefix(t, "//go:generate"):
		return Part{Code: t, Kind: "directive", What: "Tells `go generate` to run: " + strings.TrimSpace(t[13:]) + ".", Why: "Used to create code or resources (such as a Windows icon .syso file) before building."}, true
	case x.lang.ID == "go" && strings.HasPrefix(t, "//sys "):
		x.tag("windows-go")
		return Part{Code: t, Kind: "directive", What: "A note for the mksyscall tool, which writes the Go code that calls this Windows function.", Why: "golang.org/x/sys/windows uses these lines to generate safe wrappers around Windows DLL functions."}, true
	case x.lang.ID == "go" && strings.HasPrefix(t, "//export "):
		return Part{Code: t, Kind: "directive", What: "Makes the next Go function callable from C code (cgo) under the name " + strings.TrimSpace(t[9:]) + ".", Why: "Used when building a DLL or a library other languages can call."}, true
	case x.lang.ID == "gdscript" && strings.HasPrefix(t, "##"):
		return Part{Code: t, Kind: "comment", What: "A documentation comment: “" + clip(strings.TrimSpace(t[2:]), 160) + "”", Why: "Godot shows ## comments in its built-in help and in the Inspector tooltip."}, true
	}
	return Part{}, false
}

// explainLine explains one line of code.
func explainLine(x *ctx, s srcLine, code string) LineOut {
	lo := LineOut{N: s.N, Text: s.Text}
	fam := x.lang.Family
	if fam == "basic" {
		x.basicLine = 0
		if m := reBasicNum.FindStringSubmatch(code); m != nil && m[2] != "" {
			x.basicLine, _ = strconv.Atoi(m[1])
			code = m[2]
			if !x.numNote {
				x.numNote = true
				x.notes = append(x.notes, "`"+m[1]+"` is the line number. Classic BASIC runs lines in number order, and GOTO and GOSUB jump to them by number. Programmers counted in tens (10, 20, 30…) to leave room for new lines.")
			}
		} else if m != nil && m[2] == "" {
			lo.Parts = []Part{{Code: code, Kind: "other", What: "An empty numbered line.", Why: ""}}
			lo.Kind, lo.Known = "other", true
			return lo
		}
		basicMem(x, code)
	}
	stmts := splitStatements(x.lang, code)
	known := true
	openKind, openName := "", ""
	closed := []*block{}
	braces := fam == "c" || x.lang.ID == "powershell" || x.lang.ID == "css"
	keywords := !braces && fam != "py" && fam != "lua" && fam != "rb"

	for _, st := range stmts {
		lead := ""
		if braces {
			// leading } closes blocks, e.g. "} else {"
			for strings.HasPrefix(st, "}") || (lead != "" && (strings.HasPrefix(st, ")") || strings.HasPrefix(st, ";") || strings.HasPrefix(st, ","))) {
				if st[0] == '}' {
					if b := x.pop(s.N); b != nil {
						closed = append(closed, b)
					}
				}
				lead += st[:1]
				st = strings.TrimSpace(st[1:])
			}
		}
		if st == "" {
			if len(closed) > 0 {
				lo.Parts = append(lo.Parts, closerPart(x, strings.TrimSpace(lead), closed))
				closed = nil
			}
			continue
		}
		if st == "{" && braces {
			k, nm := x.pending, x.pendName
			if k == "" {
				k = "block"
			}
			start := s.N
			if x.pendLine > 0 {
				start = x.pendLine
			}
			x.push(k, nm, start, 0)
			x.pending = ""
			what := "Opens the block for the line above."
			if x.pendLine > 0 {
				what = fmt.Sprintf("Opens the block that belongs to line %d.", x.pendLine)
			}
			lo.Parts = append(lo.Parts, Part{Code: "{", Kind: "open", What: what, Why: "Everything between this { and its matching } belongs together."})
			continue
		}
		p, r, ok := matchRules(x, st)
		if !ok {
			known = false
		}
		if r != nil && r.Open != "" && openKind == "" {
			openKind = r.Open
			if r.Name > 0 {
				if m := r.re.FindStringSubmatch(trimStmt(x.lang, st)); m != nil && r.Name < len(m) {
					openName = unhide(m[r.Name], x.strs)
				}
			}
		}
		if len(closed) > 0 {
			// "} else {": mention what was closed.
			cp := closerPart(x, lead, closed)
			p.What = cp.What + " Then: " + lowerFirst(p.What)
			closed = nil
		}
		if keywords {
			if r != nil && r.Open != "" {
				x.push(r.Open, openName, s.N, 0)
				openKind = ""
			}
			if x.pending == "if-block" {
				x.push("if", "", s.N, 0)
				x.pending = ""
			}
		}
		if r != nil && r.Close && keywords {
			b := x.pop(s.N)
			if b != nil && p.What == "" {
				p = closerPart(x, st, []*block{b})
			} else if b != nil {
				p.What += " " + closerPart(x, st, []*block{b}).What
			}
		}
		lo.Parts = append(lo.Parts, p)
	}
	if len(closed) > 0 {
		lo.Parts = append(lo.Parts, closerPart(x, "}", closed))
	}

	// Block bookkeeping for brace languages: count { and } after the leading ones.
	if braces {
		body := strings.TrimLeft(code, "}); ,")
		if strings.TrimSpace(body) != "{" {
			opens, closes := 0, 0
			for _, c := range body {
				switch c {
				case '{':
					opens++
				case '}':
					closes++
				}
			}
			// pair up { } on the same line (e.g. "if (x) { y(); }")
			for opens > 0 && closes > 0 {
				opens--
				closes--
			}
			for i := 0; i < closes; i++ {
				x.pop(s.N)
			}
			k := openKind
			if k == "" {
				k = "block"
			}
			for i := 0; i < opens; i++ {
				x.push(k, openName, s.N, 0)
				k, openName = "block", ""
			}
			x.pending, x.pendName, x.pendLine = "", "", 0
			if opens == 0 && openKind != "" && !strings.HasSuffix(code, ";") {
				x.pending, x.pendName, x.pendLine = openKind, openName, s.N
			}
		}
	}
	switch fam {
	case "py":
		if strings.HasSuffix(strings.TrimSpace(code), ":") {
			k := openKind
			if k == "" {
				k = "block"
			}
			x.push(k, openName, s.N, indentOf(s.Text))
		}
	case "lua", "rb":
		opens, closes := keywordBlocks(x.lang, code)
		for opens > 0 && closes > 0 {
			opens--
			closes--
		}
		for i := 0; i < closes; i++ {
			if b := x.pop(s.N); b != nil && len(lo.Parts) > 0 && lo.Parts[len(lo.Parts)-1].Kind == "close" && lo.Parts[len(lo.Parts)-1].What == "" {
				lo.Parts[len(lo.Parts)-1] = closerPart(x, code, []*block{b})
			}
		}
		k := openKind
		if k == "" {
			k = "block"
		}
		for i := 0; i < opens; i++ {
			x.push(k, openName, s.N, 0)
			k, openName = "block", ""
		}
	}

	// Kind of the line = kind of its first part.
	if len(lo.Parts) > 0 {
		lo.Kind = lo.Parts[0].Kind
	} else {
		lo.Kind = "other"
	}
	lo.Known = known
	lo.Tags = uniq(x.tags)
	lo.Notes = append(lo.Notes, uniq(x.notes)...)
	return lo
}

// keywordBlocks counts the blocks a Lua or Ruby line opens and closes.
var (
	reLuaOpen  = regexp.MustCompile(`\b(function|if|do|repeat)\b`)
	reLuaClose = regexp.MustCompile(`\b(end|until)\b`)
	reRbOpen   = regexp.MustCompile(`^\s*(def|class|module|if|unless|while|until|case|begin|for)\b|\bdo\s*(\|[^|]*\|)?\s*$`)
	reRbClose  = regexp.MustCompile(`(^|;)\s*end\b`)
)

func keywordBlocks(l *Lang, code string) (int, int) {
	if l.Family == "lua" {
		// "while x do" and "for ... do" open one block: only "do" is counted.
		return len(reLuaOpen.FindAllString(code, -1)), len(reLuaClose.FindAllString(code, -1))
	}
	o := 0
	if reRbOpen.MatchString(code) {
		o = 1
	}
	return o, len(reRbClose.FindAllString(code, -1))
}

func closerPart(x *ctx, code string, bs []*block) Part {
	if code == "" {
		code = "}"
	}
	var names []string
	for _, b := range bs {
		names = append(names, describeBlock(b))
	}
	what := "Ends " + strings.Join(names, ", then ") + "."
	why := "Marks where that part of the code finishes; the lines between belong to it."
	if len(bs) == 1 {
		switch bs[0].Kind {
		case "loop":
			why = "When the program reaches this point it jumps back to the top of the loop and checks whether to go round again."
		case "function", "method", "event":
			why = "When the program gets here, the function is finished and goes back to whatever called it."
		case "if":
			why = "After this point the program carries on normally, whatever the check decided."
		case "class":
			why = "Everything this class (blueprint) contains has been listed."
		}
	}
	return Part{Code: code, Kind: "close", What: what, Why: why}
}

func describeBlock(b *block) string {
	k := b.Kind
	name := ""
	if b.Name != "" {
		name = " " + "`" + b.Name + "`"
	}
	switch k {
	case "function", "method":
		return fmt.Sprintf("the function%s (from line %d)", name, b.Start)
	case "event":
		return fmt.Sprintf("the event handler%s (from line %d)", name, b.Start)
	case "class":
		return fmt.Sprintf("the class%s (from line %d)", name, b.Start)
	case "loop":
		return fmt.Sprintf("the loop that started on line %d", b.Start)
	case "if":
		return fmt.Sprintf("the if-check from line %d", b.Start)
	case "switch":
		return fmt.Sprintf("the choice (switch) from line %d", b.Start)
	case "try":
		return fmt.Sprintf("the try block from line %d", b.Start)
	case "namespace", "module":
		return fmt.Sprintf("the %s%s (from line %d)", k, name, b.Start)
	case "struct", "enum", "interface", "table", "object", "list", "rule":
		return fmt.Sprintf("the %s%s from line %d", k, name, b.Start)
	}
	return fmt.Sprintf("the block that started on line %d", b.Start)
}

// trimStmt removes endings that rules don't need to see.
func trimStmt(l *Lang, st string) string {
	st = strings.TrimSpace(st)
	switch l.Family {
	case "c":
		st = strings.TrimSpace(strings.TrimRight(st, "{; "))
	case "py":
		if l.ID == "python" || l.ID == "gdscript" {
			st = strings.TrimSpace(strings.TrimSuffix(st, ":"))
		}
	case "web":
		if l.ID == "css" {
			st = strings.TrimSpace(strings.TrimRight(st, "{; "))
		}
	}
	if l.ID == "powershell" || l.ID == "bash" {
		st = strings.TrimSpace(strings.TrimSuffix(st, "{"))
	}
	return st
}

func matchRules(x *ctx, st string) (Part, *Rule, bool) {
	t := trimStmt(x.lang, st)
	x.cur = strings.TrimSpace(st)
	shown := unhide(strings.TrimSpace(st), x.strs)
	for _, r := range sortedRules() {
		if !r.applies(x.lang.ID) {
			continue
		}
		idx := r.re.FindStringSubmatchIndex(t)
		if idx == nil {
			continue
		}
		m := make([]string, len(idx)/2)
		for i := range m {
			if idx[2*i] >= 0 {
				m[i] = t[idx[2*i]:idx[2*i+1]]
			}
		}
		what, why := r.What, r.Why
		if r.fn != nil {
			w, y := r.fn(m, x)
			if w == "" && y == "" {
				continue // the function decided this rule doesn't fit
			}
			if w != "" {
				what = w
			}
			if y != "" {
				why = y
			}
		}
		what = codeMarks.Replace(expand(what, m, x))
		why = codeMarks.Replace(expand(why, m, x))
		if r.Var > 0 && r.Var < len(m) {
			for _, v := range strings.Split(m[r.Var], ",") {
				v = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(v), "*"))
				if v != "" && v != "_" {
					if _, ok := x.vars[v]; !ok {
						x.vars[v] = x.n
					}
				}
			}
		}
		if r.Open == "function" || r.Open == "method" || r.Open == "event" {
			if r.Name > 0 && r.Name < len(m) && x.first {
				if _, ok := x.funcs[m[r.Name]]; !ok {
					x.funcs[m[r.Name]] = x.n
				}
			}
		}
		x.tag(r.Tags...)
		x.skipPat = r.re.String()
		apiNotes(x, t, "")
		x.skipPat = ""
		return Part{Code: shown, Kind: r.Kind, What: what, Why: why}, r, true
	}
	// Nothing specific: try the general shapes.
	p, ok := generic(x, t)
	p.Code = shown
	return p, nil, ok
}

var reVarRef = regexp.MustCompile(`\$(\d)`)

// expand fills $1..$9 in a template with captured text (shown as code).
func expand(tpl string, m []string, x *ctx) string {
	return reVarRef.ReplaceAllStringFunc(tpl, func(s string) string {
		i := int(s[1] - '0')
		if i >= len(m) {
			return ""
		}
		v := strings.TrimSpace(unhide(m[i], x.strs))
		if v == "" {
			return "nothing"
		}
		return "`" + clip(v, 80) + "`"
	})
}

// splitStatements cuts a line holding several statements into parts.
func splitStatements(l *Lang, code string) []string {
	sep := byte(0)
	switch {
	case l.Family == "basic":
		sep = ':'
	case l.Family == "c" && l.ID != "go":
		sep = ';'
	case l.Family == "lua" || l.Family == "py" || l.Family == "rb":
		sep = ';'
	}
	if sep == 0 {
		return []string{code}
	}
	var out []string
	depth, start := 0, 0
	for i := 0; i < len(code); i++ {
		switch code[i] {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		case sep:
			if depth == 0 {
				p := strings.TrimSpace(code[start:i])
				if sep == ';' && p != "" {
					p += ";"
				}
				if p != "" && p != ";" {
					out = append(out, p)
				}
				start = i + 1
			}
		}
	}
	if p := strings.TrimSpace(code[start:]); p != "" {
		out = append(out, p)
	}
	if len(out) == 0 {
		return []string{code}
	}
	// In C-like languages, keep "x = 1; }" style lines together with braces.
	if l.Family == "c" && len(out) > 1 {
		var merged []string
		for _, p := range out {
			if (p == "}" || p == "};") && len(merged) > 0 {
				merged = append(merged, p)
				continue
			}
			merged = append(merged, p)
		}
		out = merged
	}
	// BASIC: "IF X THEN A=1 : B=2" — the statements after THEN belong to the IF.
	if l.Family == "basic" {
		var merged []string
		for _, p := range out {
			if n := len(merged); n > 0 && reBasicThenTail.MatchString(merged[n-1]) && !reBasicThenEnd.MatchString(merged[n-1]) {
				merged[n-1] += " : " + p
				continue
			}
			merged = append(merged, p)
		}
		out = merged
	}
	return out
}

var (
	reBasicThenTail = regexp.MustCompile(`(?i)^IF\b.*\bTHEN\b\s*\S`)
	reBasicThenEnd  = regexp.MustCompile(`(?i)\bTHEN\s*$`)
)

func clip(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= n {
		return string(r)
	}
	return string(r[:n-1]) + "…"
}

func lowerFirst(s string) string {
	if s == "" || strings.HasPrefix(s, "`") {
		return s
	}
	r := []rune(s)
	if len(r) > 1 && r[1] >= 'A' && r[1] <= 'Z' {
		return s // an abbreviation such as "OK"
	}
	return strings.ToLower(string(r[0])) + string(r[1:])
}

func uniq(a []string) []string {
	seen := map[string]bool{}
	var o []string
	for _, s := range a {
		if s != "" && !seen[s] {
			seen[s] = true
			o = append(o, s)
		}
	}
	return o
}

// summarise writes the few sentences at the top of the report.
func summarise(r Report) []string {
	var out []string
	l := langByID[r.Lang.ID]
	name := r.Lang.Name
	if r.Lang.Dialect != "" {
		name += " (" + r.Lang.Dialect + ")"
	}
	how := "This looks like " + name + "."
	if r.Lang.Chosen {
		how = "Reading this as " + name + "."
	}
	out = append(out, how+" "+l.About)
	if r.Purpose.Headline != "" {
		out = append(out, "**What it does:** "+r.Purpose.Headline+" See the *What it does* tab for how Codie worked that out.")
	}
	var bits []string
	if r.Stats.Functions > 0 {
		bits = append(bits, plural(r.Stats.Functions, "function"))
	}
	if r.Stats.Classes > 0 {
		bits = append(bits, plural(r.Stats.Classes, "class", "classes"))
	}
	if r.Stats.Loops > 0 {
		bits = append(bits, plural(r.Stats.Loops, "loop"))
	}
	s := fmt.Sprintf("%s of code", plural(r.Stats.Code, "line"))
	if r.Stats.Comments > 0 {
		s += " and " + plural(r.Stats.Comments, "comment line")
	}
	if len(bits) > 0 {
		s += ", with " + joinAnd(bits)
	}
	out = append(out, s+".")
	if r.Game.Engine != nil {
		out = append(out, "It's game code for "+r.Game.Engine.Name+": see the Game logic tab for how its game loop works.")
	} else if len(r.Game.Hits) >= 2 {
		out = append(out, "It contains game logic ("+joinAnd(gameTitles(r.Game.Hits, 3))+"): see the Game logic tab.")
	}
	if len(r.Warnings) > 0 {
		out = append(out, plural(len(r.Warnings), "thing")+" worth checking: see Warnings.")
	}
	return out
}

func plural(n int, one string, many ...string) string {
	if n == 1 {
		return "1 " + one
	}
	if len(many) > 0 {
		return fmt.Sprintf("%d %s", n, many[0])
	}
	return fmt.Sprintf("%d %ss", n, one)
}

func joinAnd(a []string) string {
	switch len(a) {
	case 0:
		return ""
	case 1:
		return a[0]
	}
	return strings.Join(a[:len(a)-1], ", ") + " and " + a[len(a)-1]
}
