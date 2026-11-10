// Codie — api.go: the lookup that recognises well-known functions and
// commands (print, Time.deltaTime, love.graphics.rectangle, POKE…).
// The word lists live in api_*.go as simple tables, one entry per line:
//
//	languages | name[,other names] | kind | what it does | why you'd use it | concept tags
//
// «x» in the text is shown as code. Part of the nrmluvdev maker suite.
// Created by nrmluvdev.
package codie

import (
	"strings"
)

type apiEntry struct {
	langs   map[string]bool
	all     bool
	dialect string
	Key     string
	Kind    string
	What    string
	Why     string
	Tags    []string
	Prop    bool // a value or property rather than something you call
}

var (
	apiExact = map[string][]*apiEntry{}
	apiLower = map[string][]*apiEntry{}
)

// tables registered by the api_*.go files
var apiTables []string

func init() {
	for _, t := range apiTables {
		loadAPI(t)
	}
}

var codeMarks = strings.NewReplacer("«", "`", "»", "`")

func loadAPI(table string) {
	for _, ln := range strings.Split(table, "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" || strings.HasPrefix(ln, "#") {
			continue
		}
		for strings.Contains(ln, "| |") {
			ln = strings.ReplaceAll(ln, "| |", "|  |")
		}
		if strings.HasSuffix(ln, " |") {
			ln += " "
		}
		f := strings.Split(ln, " | ")
		for len(f) < 6 {
			f = append(f, "")
		}
		e := &apiEntry{langs: map[string]bool{}, Kind: strings.TrimSpace(f[2]), What: codeMarks.Replace(strings.TrimSpace(f[3])), Why: codeMarks.Replace(strings.TrimSpace(f[4]))}
		if t := strings.TrimSpace(f[5]); t != "" {
			e.Tags = strings.Fields(strings.ReplaceAll(t, ",", " "))
		}
		if e.Kind == "prop" {
			e.Prop, e.Kind = true, "call"
		}
		for _, l := range strings.Split(f[0], ",") {
			l = strings.TrimSpace(l)
			switch {
			case l == "*":
				e.all = true
			case strings.HasPrefix(l, "@"):
				for _, x := range langs {
					if x.Family == l[1:] {
						e.langs[x.ID] = true
					}
				}
			case strings.HasPrefix(l, "~"): // a dialect, e.g. ~PICO-8 (implies Lua)
				e.dialect = l[1:]
			default:
				e.langs[l] = true
			}
		}
		if e.dialect != "" && len(e.langs) == 0 && !e.all {
			e.langs["lua"] = true
		}
		for _, k := range strings.Split(f[1], ",") {
			k = strings.TrimSpace(k)
			if k == "" {
				continue
			}
			c := *e
			c.Key = k
			apiExact[k] = append(apiExact[k], &c)
			apiLower[strings.ToLower(k)] = append(apiLower[strings.ToLower(k)], &c)
		}
	}
}

func (e *apiEntry) fits(x *ctx) bool {
	if !e.all && !e.langs[x.lang.ID] {
		return false
	}
	if e.dialect != "" && !strings.Contains(x.dialect, e.dialect) {
		return false
	}
	return true
}

func findKey(x *ctx, k string) *apiEntry {
	list := apiExact[k]
	if x.lang.NoCase || x.lang.Family == "basic" {
		list = apiLower[strings.ToLower(k)]
	}
	// dialect-specific entries win
	for _, e := range list {
		if e.dialect != "" && e.fits(x) {
			return e
		}
	}
	for _, e := range list {
		if e.dialect == "" && e.fits(x) {
			return e
		}
	}
	return nil
}

// prefixes that don't change what a call means
var namePrefixes = []string{"self.", "this.", "self:", "this->", "rl.", "raylib.", "std::", "base.", "Super::", "super.", "@", "$"}

// lookupAPIName finds a known name; it also returns the key that matched.
func lookupAPIName(x *ctx, name string) (*apiEntry, string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ""
	}
	if !strings.Contains(name, "::") {
		name = strings.ReplaceAll(name, ":", ".")
	}
	name = strings.ReplaceAll(name, "->", ".")
	cands := []string{name}
	for _, p := range namePrefixes {
		if strings.HasPrefix(name, p) && len(name) > len(p) {
			cands = append(cands, name[len(p):])
		}
	}
	segs := strings.FieldsFunc(name, func(r rune) bool { return r == '.' })
	if len(segs) > 2 {
		cands = append(cands, strings.Join(segs[len(segs)-2:], "."))
	}
	if len(segs) > 1 {
		cands = append(cands, "."+segs[len(segs)-1])
	}
	for _, c := range cands {
		if e := findKey(x, c); e != nil {
			return e, c
		}
	}
	return nil, ""
}

// lookupAPI looks up a whole short statement such as "CLS" or "cls()".
func lookupAPI(x *ctx, t string) (*apiEntry, string) {
	t = strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(strings.TrimSpace(t), ";"), "()"))
	if strings.ContainsAny(t, " (=") {
		return nil, ""
	}
	return lookupAPIName(x, t)
}
