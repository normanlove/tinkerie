// Codie — words.go: helpers that turn bits of code into plain English
// (conditions, parameter lists, loop headers).
// Part of the free nrmluvdev maker suite. Created by nrmluvdev.
package codie

import (
	"fmt"
	"regexp"
	"strings"
)

var reCondOp = regexp.MustCompile(`<<|>>|===|!==|==|!=|~=|<>|>=|<=|=<|=>|&&|\|\||->|::|\band\b|\bor\b|\bnot in\b|\bis not\b|\bnot\b|\bin\b|\bis\b|\bAND\b|\bOR\b|\bNOT\b|-eq\b|-ne\b|-gt\b|-lt\b|-ge\b|-le\b|-and\b|-or\b|-not\b|-like\b|-match\b|-contains\b|-z\b|-n\b|-f\b|-d\b|-e\b|[<>!=]`)

// condWords rewrites a condition such as "score > 100 && !dead" in words.
func condWords(x *ctx, cond string) string {
	cond = strings.TrimSpace(cond)
	for strings.HasPrefix(cond, "(") && strings.HasSuffix(cond, ")") && balanced(cond[1:len(cond)-1]) {
		cond = strings.TrimSpace(cond[1 : len(cond)-1])
	}
	basicEq := x.lang.Family == "basic" || x.lang.Family == "sql"
	shell := x.lang.Family == "shell"
	out := reCondOp.ReplaceAllStringFunc(cond, func(op string) string {
		switch strings.ToLower(op) {
		case "<<", ">>":
			return op
		case "===":
			return " is exactly "
		case "!==":
			return " is not exactly "
		case "==", "-eq":
			return " equals "
		case "!=", "~=", "<>", "-ne", "is not":
			return " is not "
		case ">=", "=>", "-ge":
			if op == "=>" && x.lang.Family != "basic" {
				return op
			}
			return " is at least "
		case "<=", "=<", "-le":
			return " is at most "
		case ">", "-gt":
			return " is more than "
		case "<", "-lt":
			return " is less than "
		case "&&", "and", "-and":
			return " and "
		case "||", "or", "-or":
			return " or "
		case "!", "not", "-not":
			return " not "
		case "not in":
			return " is not in "
		case "in":
			return " is in "
		case "is":
			return " is "
		case "=":
			if basicEq || shell {
				return " equals "
			}
			return " is set to (!) "
		case "-like", "-match":
			return " matches "
		case "-contains":
			return " contains "
		case "-z":
			if shell {
				return " is empty: "
			}
		case "-n":
			if shell {
				return " is not empty: "
			}
		case "-f":
			if shell {
				return " a file exists: "
			}
		case "-d":
			if shell {
				return " a folder exists: "
			}
		case "-e":
			if shell {
				return " exists: "
			}
		}
		return op
	})
	out = strings.Join(strings.Fields(unhide(out, x.strs)), " ")
	out = strings.ReplaceAll(out, "( ", "(")
	out = strings.ReplaceAll(out, " )", ")")
	return out
}

func balanced(s string) bool {
	d := 0
	for _, c := range s {
		switch c {
		case '(':
			d++
		case ')':
			d--
			if d < 0 {
				return false
			}
		}
	}
	return d == 0
}

// checks returns "Checks whether `cond` (in words: …)".
func checks(x *ctx, cond string) string {
	c := strings.TrimSpace(cond)
	for strings.HasPrefix(c, "(") && strings.HasSuffix(c, ")") && balanced(c[1:len(c)-1]) {
		c = strings.TrimSpace(c[1 : len(c)-1])
	}
	shown := unhide(c, x.strs)
	w := condWords(x, c)
	if strings.Join(strings.Fields(shown), " ") == w {
		return "checks whether " + code(shown) + " is true"
	}
	return "checks whether " + code(shown) + ", that is: " + w
}

// splitTop splits on commas that aren't inside brackets.
func splitTop(s string, sep byte) []string {
	var out []string
	d, start := 0, 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(', '[', '{', '<':
			d++
		case ')', ']', '}', '>':
			if d > 0 {
				d--
			}
		case sep:
			if d == 0 {
				out = append(out, strings.TrimSpace(s[start:i]))
				start = i + 1
			}
		}
	}
	if t := strings.TrimSpace(s[start:]); t != "" {
		out = append(out, t)
	}
	return out
}

var reLastIdent = regexp.MustCompile(`([A-Za-z_$][\w$]*)\s*(\[\s*\])?\s*$`)

// paramNames pulls the names out of a parameter list in most languages.
func paramNames(l *Lang, params string) []string {
	var names []string
	for _, p := range splitTop(params, ',') {
		p = strings.TrimSpace(p)
		if p == "" || p == "void" || p == "self" || p == "&self" || p == "&mut self" || p == "this" || p == "cls" || p == "..." {
			continue
		}
		if i := strings.Index(p, "="); i > 0 {
			p = strings.TrimSpace(p[:i])
		}
		switch {
		case strings.Contains(p, ":") && l.ID != "cpp" && l.ID != "c":
			// name: type (TypeScript, Rust, Kotlin, Swift, Python, GDScript)
			p = strings.TrimSpace(p[:strings.Index(p, ":")])
			f := strings.Fields(p)
			if len(f) > 0 {
				p = f[len(f)-1]
			}
		case l.ID == "go":
			f := strings.Fields(p)
			if len(f) > 0 {
				p = f[0]
			}
		default:
			if m := reLastIdent.FindStringSubmatch(p); m != nil {
				p = m[1]
			}
		}
		p = strings.TrimLeft(p, "*&$@")
		if p != "" {
			names = append(names, p)
		}
	}
	return names
}

func takes(l *Lang, params string) string {
	n := paramNames(l, params)
	switch len(n) {
	case 0:
		return "It takes no inputs."
	case 1:
		return "It takes one input, " + code(n[0]) + "."
	}
	var c []string
	for _, s := range n {
		c = append(c, code(s))
	}
	return fmt.Sprintf("It takes %d inputs: %s.", len(n), joinAnd(c))
}
