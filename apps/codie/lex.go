// Codie — lex.go: splits code into lines and separates each line's real code
// from its comments and text values ("strings"), so the rules that explain a
// line never trip over a keyword written inside a comment or a message.
// Part of the free nrmluvdev maker suite. Created by nrmluvdev.
package codie

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// Text values are swapped for a marker \x01N\x02 while matching; strs[N] keeps
// the original so explanations can quote it.
const strOpen, strClose = '\x01', '\x02'

// STR is the regular-expression piece that matches one hidden text value.
const STR = "\x01\\d+\x02"

type srcLine struct {
	N        int
	Text     string // the line exactly as typed
	Code     string // without comments, text values hidden
	Comment  string // comment text on this line, if any
	Strs     []string
	InBlock  bool // the whole line is inside a /* ... */ comment
	InString int  // the whole line is inside a multi-line text value begun on this line number
}

type lexState struct {
	blockEnd string // inside a block comment: its closing marker
	strEnd   string // inside a multi-line string: its closing quote(s)
	strFrom  int
	strBuf   *strings.Builder
}

func lexLines(code string, l *Lang) []srcLine {
	code = strings.ReplaceAll(code, "\r\n", "\n")
	code = strings.ReplaceAll(code, "\r", "\n")
	code = strings.ReplaceAll(code, "\t", "    ")
	raw := strings.Split(code, "\n")
	out := make([]srcLine, 0, len(raw))
	st := &lexState{}
	for i, t := range raw {
		out = append(out, lexLine(i+1, t, l, st))
	}
	return out
}

var reBasicRem = regexp.MustCompile(`(?i)^(\s*\d*\s*|.*:\s*)REM\b`)

func lexLine(n int, t string, l *Lang, st *lexState) srcLine {
	ln := srcLine{N: n, Text: t}
	var code strings.Builder
	var comment []string
	quotes := l.Quotes
	if l.ID == "html" {
		quotes = ""
	}
	i := 0
	if st.blockEnd != "" {
		if j := strings.Index(t, st.blockEnd); j >= 0 {
			comment = append(comment, strings.TrimSpace(t[:j]))
			i = j + len(st.blockEnd)
			st.blockEnd = ""
		} else {
			ln.InBlock = true
			ln.Comment = strings.TrimSpace(t)
			return ln
		}
	}
	if st.strEnd != "" {
		if j := strings.Index(t, st.strEnd); j >= 0 {
			st.strBuf.WriteString(t[:j])
			ln.Strs = append(ln.Strs, st.strBuf.String())
			code.WriteString(string(strOpen) + "0" + string(strClose))
			i = j + len(st.strEnd)
			st.strEnd = ""
		} else {
			st.strBuf.WriteString(t + "\n")
			ln.InString = st.strFrom
			return ln
		}
	}
	// Batch files: comments only at the start of a line.
	if l.ID == "batch" {
		tt := strings.TrimSpace(t)
		lo := strings.ToLower(tt)
		if strings.HasPrefix(tt, "::") || lo == "rem" || strings.HasPrefix(lo, "rem ") || strings.HasPrefix(lo, "@rem") {
			ln.Comment = strings.TrimSpace(strings.TrimLeft(tt[min(len(tt), 3):], ": "))
			if strings.HasPrefix(lo, "rem") {
				ln.Comment = strings.TrimSpace(tt[3:])
			} else if strings.HasPrefix(lo, "@rem") {
				ln.Comment = strings.TrimSpace(tt[4:])
			}
			ln.Code = ""
			return ln
		}
	}
	for i < len(t) {
		rest := t[i:]
		// Block comment?
		opened := false
		for _, b := range l.Block {
			if strings.HasPrefix(rest, b[0]) {
				if b[0] == "=begin" && i != 0 {
					continue
				}
				if j := strings.Index(rest[len(b[0]):], b[1]); j >= 0 {
					comment = append(comment, strings.TrimSpace(rest[len(b[0]):len(b[0])+j]))
					i += len(b[0]) + j + len(b[1])
				} else {
					comment = append(comment, strings.TrimSpace(rest[len(b[0]):]))
					st.blockEnd = b[1]
					i = len(t)
				}
				opened = true
				break
			}
		}
		if opened {
			continue
		}
		// Line comment?
		if lc := lineComment(t, i, l, code.String()); lc >= 0 {
			comment = append(comment, strings.TrimSpace(t[lc:]))
			break
		}
		c := t[i]
		// Lua long strings [[ ... ]]
		if l.Family == "lua" && strings.HasPrefix(rest, "[[") {
			if j := strings.Index(rest[2:], "]]"); j >= 0 {
				code.WriteString(hideStr(&ln, rest[2:2+j]))
				i += 4 + j
			} else {
				st.strEnd, st.strFrom, st.strBuf = "]]", n, &strings.Builder{}
				st.strBuf.WriteString(rest[2:] + "\n")
				code.WriteString(hideStr(&ln, "…"))
				i = len(t)
			}
			continue
		}
		if quotes != "" && strings.IndexByte(quotes, c) >= 0 {
			// Python triple quotes and Go/JS back-quotes may run over lines.
			q := string(c)
			if l.Family == "py" && (strings.HasPrefix(rest, `"""`) || strings.HasPrefix(rest, `'''`)) {
				q = rest[:3]
			}
			multi := len(q) == 3 || c == '`'
			body := rest[len(q):]
			end := findClose(body, q, l)
			if end < 0 && multi {
				st.strEnd, st.strFrom, st.strBuf = q, n, &strings.Builder{}
				st.strBuf.WriteString(body + "\n")
				code.WriteString(hideStr(&ln, "…"))
				i = len(t)
				continue
			}
			if end < 0 {
				end = len(body) // unfinished text value: take the rest of the line
				code.WriteString(hideStr(&ln, body))
				i = len(t)
				continue
			}
			code.WriteString(hideStr(&ln, body[:end]))
			i += len(q) + end + len(q)
			continue
		}
		code.WriteByte(c)
		i++
	}
	ln.Code = strings.TrimRight(code.String(), " ")
	ln.Comment = strings.TrimSpace(strings.Join(nonEmpty(comment), " "))
	return ln
}

func hideStr(ln *srcLine, v string) string {
	ln.Strs = append(ln.Strs, v)
	return string(strOpen) + strconv.Itoa(len(ln.Strs)-1) + string(strClose)
}

func findClose(body, q string, l *Lang) int {
	escapes := q != "`" && l.Family != "basic" && l.Family != "sql" && l.ID != "batch" && l.ID != "powershell"
	for j := 0; j < len(body); j++ {
		if escapes && body[j] == '\\' {
			j++
			continue
		}
		if strings.HasPrefix(body[j:], q) {
			// SQL and BASIC write a quote inside text by doubling it.
			if (l.Family == "sql" || l.Family == "basic") && strings.HasPrefix(body[j+len(q):], q) {
				j += len(q)
				continue
			}
			return j
		}
	}
	return -1
}

// lineComment returns where a line comment starts at position i, or -1.
func lineComment(t string, i int, l *Lang, before string) int {
	rest := t[i:]
	if l.Family == "basic" {
		if rest[0] == '\'' {
			return i
		}
		if (i == 0 || !isWord(rune(t[i-1]))) && len(rest) >= 3 && strings.EqualFold(rest[:3], "REM") && (len(rest) == 3 || !isWord(rune(rest[3]))) {
			if reBasicRem.MatchString(before + "REM") {
				return i
			}
		}
		return -1
	}
	for _, m := range l.Line {
		if strings.HasPrefix(rest, m) {
			if m == "#" && (l.Family == "shell" || l.ID == "php") && i > 0 && !unicode.IsSpace(rune(t[i-1])) {
				continue // $# or ${#x} in a shell is not a comment
			}
			if m == "#" && l.ID == "powershell" && strings.HasPrefix(rest, "#>") {
				continue
			}
			return i
		}
	}
	return -1
}

func isWord(r rune) bool { return r == '_' || r == '$' || unicode.IsLetter(r) || unicode.IsDigit(r) }

func nonEmpty(a []string) []string {
	var o []string
	for _, s := range a {
		if s != "" {
			o = append(o, s)
		}
	}
	return o
}

// unhide puts the original text values back for showing to people.
func unhide(s string, strs []string) string {
	if !strings.ContainsRune(s, strOpen) {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == strOpen {
			j := strings.IndexByte(s[i:], strClose)
			if j > 0 {
				k, _ := strconv.Atoi(s[i+1 : i+j])
				v := ""
				if k < len(strs) {
					v = strs[k]
				}
				if r := []rune(v); len(r) > 40 {
					v = string(r[:38]) + "…"
				}
				b.WriteString(`"` + v + `"`)
				i += j
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
