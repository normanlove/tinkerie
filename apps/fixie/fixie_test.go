// Checks Fixie's repair guide: every path through it, for every device and
// symptom, must end at a result, and everything it refers to must exist.
// Created by nrmluvdev.
package fixie

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
)

type gAnswer struct {
	// (answers' text is checked by TestWording)
	T    string `json:"t"`
	Go   string `json:"go"`
	Only string `json:"only"`
}
type gBand struct {
	Lt  *float64 `json:"lt"`
	A   int      `json:"a"`
	Say string   `json:"say"`
}
type gStep struct {
	Result  bool              `json:"result"`
	T       string            `json:"t"`
	Why     string            `json:"why"`
	Good    string            `json:"good"`
	Bad     string            `json:"bad"`
	Cause   string            `json:"cause"`
	Pro     string            `json:"pro"`
	Do      []json.RawMessage `json:"do"`
	Q       string            `json:"q"`
	A       []gAnswer         `json:"a"`
	Tools   []string          `json:"tools"`
	Signs   []string          `json:"signs"`
	Open    bool              `json:"open"`
	Only    string            `json:"only"`
	Skip    string            `json:"skip"`
	Next    string            `json:"next"`
	Fix     []json.RawMessage `json:"fix"`
	Measure *struct {
		Label string  `json:"label"`
		Bands []gBand `json:"bands"`
		Near  *struct {
			Label string `json:"label"`
			Ok    int    `json:"ok"`
			Bad   int    `json:"bad"`
		} `json:"near"`
		Calc string `json:"calc"`
	} `json:"measure"`
}
type guide struct {
	Power      []struct{ ID string } `json:"power"`
	Features   []struct{ ID string } `json:"features"`
	Categories []struct {
		ID    string   `json:"id"`
		Kind  string   `json:"kind"`
		Power []string `json:"power"`
		Has   []string `json:"has"`
	} `json:"categories"`
	Derived []struct {
		Tag  string   `json:"tag"`
		All  []string `json:"all"`
		None []string `json:"none"`
	} `json:"derived"`
	Danger []struct {
		ID    string   `json:"id"`
		Words []string `json:"words"`
	} `json:"danger"`
	Legacy map[string]struct {
		Cat   string   `json:"cat"`
		Power []string `json:"power"`
		Has   []string `json:"has"`
	} `json:"legacy"`
	Symptoms []struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Desc  string `json:"desc"`
		For   string `json:"for"`
		Start string `json:"start"`
	} `json:"symptoms"`
	Steps map[string]gStep `json:"steps"`
	Tools []struct {
		ID  string `json:"id"`
		For string `json:"for"`
	} `json:"tools"`
	Signs []struct {
		ID  string `json:"id"`
		For string `json:"for"`
	} `json:"signs"`
}

// tags for a described device, as the window works them out
func (g guide) tags(kind string, power, has []string) map[string]bool {
	t := map[string]bool{}
	if kind != "" {
		t[kind] = true
	}
	for _, x := range power {
		t[x] = true
	}
	for _, x := range has {
		t[x] = true
	}
	for _, d := range g.Derived {
		ok := true
		for _, a := range d.All {
			ok = ok && t[a]
		}
		for _, n := range d.None {
			ok = ok && !t[n]
		}
		if ok {
			t[d.Tag] = true
		}
	}
	return t
}

// norm and dangerous are the same tests as the window's (app.js).
var nonWord = regexp.MustCompile(`[^a-z0-9]+`)

func norm(s string) string {
	return " " + strings.TrimSpace(nonWord.ReplaceAllString(strings.ToLower(s), " ")) + " "
}

func (g guide) dangerous(name string) string {
	n := norm(name)
	for _, d := range g.Danger {
		for _, w := range d.Words {
			w = strings.TrimSpace(norm(w))
			for _, end := range []string{"", "s", "es"} {
				if strings.Contains(n, " "+w+end+" ") {
					return d.ID
				}
			}
		}
	}
	return ""
}

// match is the same test as the window's: "a|b" is any of them, "a&b" is
// both, "!a" is not ("&" binds tighter than "|").
func match(cond string, tags map[string]bool) bool {
	if cond == "" {
		return true
	}
	for _, c := range strings.Split(cond, "|") {
		all := true
		for _, t := range strings.Split(c, "&") {
			if strings.HasPrefix(t, "!") {
				all = all && !tags[t[1:]]
			} else {
				all = all && tags[t]
			}
		}
		if all {
			return true
		}
	}
	return false
}

// conds lists each line's condition ("" for a line everyone sees).
func conds(raw []json.RawMessage) []string {
	var out []string
	for _, r := range raw {
		var c struct {
			If string `json:"if"`
		}
		_ = json.Unmarshal(r, &c)
		out = append(out, c.If)
	}
	return out
}

// lines is how many of a step's lines (or a result's fixes) this device sees.
func lines(cs []string, tags map[string]bool) int {
	n := 0
	for _, c := range cs {
		if match(c, tags) {
			n++
		}
	}
	return n
}

func loadGuide(t *testing.T) guide {
	b, err := os.ReadFile("web/guide.json")
	if err != nil {
		t.Fatal(err)
	}
	var g guide
	if err := json.Unmarshal(b, &g); err != nil {
		t.Fatal(err)
	}
	return g
}

func TestGuideRefs(t *testing.T) {
	g := loadGuide(t)
	tools, signs, known := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, x := range g.Tools {
		tools[x.ID] = true
	}
	for _, x := range g.Signs {
		signs[x.ID] = true
	}
	for _, x := range g.Power {
		known[x.ID] = true
	}
	for _, x := range g.Features {
		known[x.ID] = true
	}
	for _, c := range g.Categories {
		known[c.Kind] = true
		for _, x := range append(append([]string{}, c.Power...), c.Has...) {
			if !known[x] {
				t.Errorf("category %s: unknown answer %q", c.ID, x)
			}
		}
	}
	for _, d := range g.Derived {
		known[d.Tag] = true
	}
	checkCond := func(where, c string) {
		for _, p := range strings.FieldsFunc(c, func(r rune) bool { return r == '|' || r == '&' }) {
			if p = strings.TrimPrefix(p, "!"); p != "" && !known[p] {
				t.Errorf("%s: unknown tag %q", where, p)
			}
		}
	}
	for _, x := range g.Tools {
		checkCond("tool "+x.ID, x.For)
	}
	for _, x := range g.Signs {
		checkCond("sign "+x.ID, x.For)
	}
	for _, s := range g.Symptoms {
		checkCond("symptom "+s.ID, s.For)
		if _, ok := g.Steps[s.Start]; !ok {
			t.Errorf("symptom %s starts at missing step %s", s.ID, s.Start)
		}
	}
	for id, s := range g.Steps {
		if s.T == "" {
			t.Errorf("%s: no title", id)
		}
		for _, x := range s.Tools {
			if !tools[x] {
				t.Errorf("%s: unknown tool %q", id, x)
			}
		}
		for _, x := range s.Signs {
			if !signs[x] {
				t.Errorf("%s: unknown sign %q", id, x)
			}
		}
		if s.Result {
			if len(s.Fix) == 0 {
				t.Errorf("%s: result with no fix", id)
			}
			if _, ok := g.Steps[s.Next]; s.Next != "" && !ok {
				t.Errorf("%s: next %s missing", id, s.Next)
			}
			continue
		}
		if s.Q == "" || len(s.A) == 0 {
			t.Errorf("%s: no question or answers", id)
		}
		checkCond(id, s.Only)
		if s.Only != "" {
			if _, ok := g.Steps[s.Skip]; !ok {
				t.Errorf("%s: skip %q missing", id, s.Skip)
			}
		}
		for _, a := range s.A {
			checkCond(id+" answer", a.Only)
			if _, ok := g.Steps[a.Go]; !ok {
				t.Errorf("%s: answer %q goes to missing %s", id, a.T, a.Go)
			}
		}
		for _, raw := range append(append([]json.RawMessage{}, s.Do...), s.Fix...) {
			var c struct {
				If string `json:"if"`
			}
			if json.Unmarshal(raw, &c) == nil {
				checkCond(id+" do", c.If)
			}
		}
		if m := s.Measure; m != nil {
			for _, b := range m.Bands {
				if b.A < 0 || b.A >= len(s.A) {
					t.Errorf("%s: band answer %d out of range", id, b.A)
				}
			}
			if m.Near != nil && (m.Near.Ok >= len(s.A) || m.Near.Bad >= len(s.A)) {
				t.Errorf("%s: near answers out of range", id)
			}
			if len(m.Bands) == 0 && m.Near == nil && m.Calc == "" {
				t.Errorf("%s: measure has nothing to judge by", id)
			}
		}
	}
}

// TestGuidePaths checks, for every kind of device and every combination of
// power and features, that each answer leads somewhere, every path ends at a
// result, and nothing goes round in a loop.
func TestGuidePaths(t *testing.T) {
	g := loadGuide(t)
	reached := map[string]bool{}
	kinds := map[string]bool{"": true}
	for _, c := range g.Categories {
		kinds[c.Kind] = true
	}
	var feats []string
	for _, f := range g.Features {
		if f.ID != "heatmains" { // that one is refused before any step
			feats = append(feats, f.ID)
		}
	}
	doIf, fixIf := map[string][]string{}, map[string][]string{}
	for id, s := range g.Steps {
		doIf[id], fixIf[id] = conds(s.Do), conds(s.Fix)
	}
	problems := map[string]string{} // one example of each problem
	report := func(key, example string) {
		if _, ok := problems[key]; !ok {
			problems[key] = example
		}
	}
	powers := [][]string{{"mains"}, {"adapter"}, {"usbpow"}, {"battery"}, {"lipo"}, {"adapter", "lipo"}, {"usbpow", "lipo"}, {"mains", "lipo"}}
	fails := 0
	for kind := range kinds {
		for _, pw := range powers {
			for mask := 0; mask < 1<<len(feats); mask++ {
				var has []string
				for i, f := range feats {
					if mask&(1<<i) != 0 {
						has = append(has, f)
					}
				}
				tags := g.tags(kind, pw, has)
				state := map[string]int{} // 1 visiting, 2 done
				var walk func(id string) bool
				walk = func(id string) bool {
					switch state[id] {
					case 1:
						t.Errorf("loop at %s (kind %q, power %v, has %v)", id, kind, pw, has)
						return false
					case 2:
						return true
					}
					state[id] = 1
					defer func() { state[id] = 2 }()
					reached[id] = true
					s := g.Steps[id]
					if s.Result {
						if lines(fixIf[id], tags) == 0 {
							report("result "+id+" has no fix lines", fmt.Sprintf("kind %q, power %v, has %v", kind, pw, has))
						}
						return true
					}
					if s.Only != "" && !match(s.Only, tags) {
						return walk(s.Skip)
					}
					if lines(doIf[id], tags) == 0 {
						report("step "+id+" has nothing to do", fmt.Sprintf("kind %q, power %v, has %v", kind, pw, has))
					}
					n := 0
					for _, a := range s.A {
						if match(a.Only, tags) {
							n++
							if !walk(a.Go) {
								return false
							}
						}
					}
					if n == 0 {
						t.Errorf("step %s has no answers (kind %q, power %v, has %v)", id, kind, pw, has)
						return false
					}
					return true
				}
				for _, s := range g.Symptoms {
					if match(s.For, tags) && !walk(s.Start) {
						if fails++; fails > 10 {
							t.Fatal("too many failures")
						}
					}
				}
			}
		}
	}
	for k, ex := range problems {
		t.Errorf("%s (e.g. %s)", k, ex)
	}
	for id := range g.Steps {
		if !reached[id] {
			t.Errorf("step %s can't be reached", id)
		}
	}
}

// TestDanger checks Fixie refuses dangerous devices by name, and doesn't
// refuse ordinary ones by mistake.
func TestDanger(t *testing.T) {
	g := loadGuide(t)
	for _, n := range []string{"Microwave", "toaster", "Breville kettle", "Dyson hair dryer", "air-fryer", "Old CRT telly", "e-bike battery", "Smoke alarm",
		"electric oven", "space heaters", "washing machine", "camera flash", "tumble dryer", "hair straighteners", "Fan heater", "vape", "gas boiler", "soldering iron"} {
		if g.dangerous(n) == "" {
			t.Errorf("%q should be refused", n)
		}
	}
	for _, n := range []string{"Bluetooth speaker", "PS5 controller", "3D printer heated bed", "wireless charger", "Iron Man toy", "LED strip", "Raspberry Pi Pico",
		"electric toothbrush", "evening lamp", "gas sensor project", "laptop", "TV", "drone", "game boy", "Cordless drill", "desk fan", "headphones", "keyboard"} {
		if d := g.dangerous(n); d != "" {
			t.Errorf("%q refused as %s", n, d)
		}
	}
	for id, l := range g.Legacy {
		known := false
		for _, c := range g.Categories {
			known = known || c.ID == l.Cat
		}
		if !known {
			t.Errorf("legacy %s: unknown category %s", id, l.Cat)
		}
	}
}

// TestRelevance pins down steps that must never be shown for the usual
// answers of each kind of device (the first was a 3D printer being asked to
// put a Pico into BOOTSEL mode).
func TestRelevance(t *testing.T) {
	g := loadGuide(t)
	never := map[string][]string{
		"machine":  {"up.", "r.bootmode", "r.strap", "r.mcu", "ch.", "dead.3", "dead.flat", "l.1", "r.ram", "r.tcon", "r.runcap", "r.brushes", "bt.2"},
		"project":  {"dead.flat", "ch.", "h.", "m.", "dp.", "tr.", "r.ram"},
		"tool":     {"dead.3", "dead.3b", "dead.port", "ch.2", "ch.3", "up.", "dp.", "snd.", "cn.", "r.lint"},
		"toy":      {"l.1", "ch.", "up.", "dead.3", "dead.fuse", "tr.", "r.runcap", "r.brushes"},
		"computer": {"dead.6", "dp.2", "dp.3", "up.", "r.brushes", "r.runcap", "h.", "l.1"},
		"phone":    {"dead.fuse", "dead.6", "tr.", "m.", "up.", "h.", "l.1", "r.ram", "r.tcon"},
		"lights":   {"up.", "m.", "snd.", "dp.", "ch."},
		"audio":    {"up.", "m.", "dp.", "h.", "tr."},
		"screen":   {"up.", "m.", "ch.", "r.ram"},
		"input":    {"up.", "m.", "dp.", "ch.", "tr."},
		"console":  {"up.", "dp.", "ch.", "h.", "r.brushes", "r.runcap"},
	}
	for _, c := range g.Categories {
		bad := never[c.ID]
		if bad == nil {
			continue
		}
		tags := g.tags(c.Kind, c.Power, c.Has)
		seen := map[string]bool{}
		var walk func(id string)
		walk = func(id string) {
			if seen[id] {
				return
			}
			seen[id] = true
			s := g.Steps[id]
			if !s.Result && s.Only != "" && !match(s.Only, tags) {
				walk(s.Skip)
				return
			}
			for _, b := range bad {
				if id == b || (strings.HasSuffix(b, ".") && strings.HasPrefix(id, b)) {
					t.Errorf("%s: shown %s (%s)", c.ID, id, s.T)
				}
			}
			for _, a := range s.A {
				if match(a.Only, tags) {
					walk(a.Go)
				}
			}
			if s.Next != "" {
				walk(s.Next)
			}
		}
		for _, sy := range g.Symptoms {
			if match(sy.For, tags) {
				walk(sy.Start)
			}
		}
	}
}

// wordRules: words in what a person reads, and the device answers they only
// make sense with ("a|b" any of, as in the guide). A line that mentions a
// battery shown for a device with no battery is a mistake, and so on.
var wordRules = []struct {
	re   *regexp.Regexp
	need string
}{
	{regexp.MustCompile(`(?i)\bbatter(y|ies)\b|\bcells?\b|\bpacks?\b`), "battery|lipo|powertool"},
	{regexp.MustCompile(`(?i)\bcharg(e|er|ers|es|ing)\b`), "lipo|powertool|usb|usbpow|adapter"},
	{regexp.MustCompile(`(?i)\bscreens?\b|\bdisplay\b|\bpicture\b|\bbacklight\b|\bpanel\b`), "screen"},
	{regexp.MustCompile(`(?i)\bfans?\b`), "fan|motor"},
	{regexp.MustCompile(`(?i)\bmotors?\b|\bbrushes\b|\bwinding`), "motor|fan"},
	{regexp.MustCompile(`(?i)\bspeakers?\b|\bsound\b|\bheadphone|\bamplifier|\bamp\b`), "audio"},
	{regexp.MustCompile(`(?i)\bvolume\b`), "audio|phone"},
	{regexp.MustCompile(`(?i)\btouchscreen|\btouch (layer|panel|sensor|surface|controls)`), "touch"},
	{regexp.MustCompile(`(?i)\bbluetooth\b|\bwi-?fi\b|\bpairing\b|\bwireless`), "wireless|phone"},
	{regexp.MustCompile(`(?i)\busb\b`), "usb|usbpow|phone|pc|mcu|printer"},
	{regexp.MustCompile(`(?i)\bports?\b`), "usb|usbpow|lipo|phone|pc|mcu|printer"},
	{regexp.MustCompile(`(?i)\bwall socket`), "mains|adapter"},
	{regexp.MustCompile(`(?i)\bmains\b|\bplug fuse|\blive and neutral|\bearth pin|\brcd\b|\bfuse box`), "mains"},
	{regexp.MustCompile(`(?i)\badapter\b`), "adapter|usbpow|usb|lipo|mains"},
	{regexp.MustCompile(`(?i)\b(i?phones?|android)\b`), "phone"},
	{regexp.MustCompile(`(?i)\blaptops?\b`), "laptop|pc"},
	{regexp.MustCompile(`(?i)\bdesktop|\bbios\b|\bmemory stick|\bgraphics card|\bmotherboard`), "pc"},
	{regexp.MustCompile(`(?i)\bconsoles?\b`), "console|pc"},
	{regexp.MustCompile(`(?i)\btvs?\b|\bt-con\b`), "tv"},
	{regexp.MustCompile(`(?i)\b3d printer|\bstepper|\bhot end|\bthermistor|\bmintemp|\bmaxtemp|\bthermal runaway`), "printer|heater"},
	{regexp.MustCompile(`(?i)\bkeyboards?\b|\bkeycaps?\b`), "input|pc"},
	{regexp.MustCompile(`(?i)\bstrips?\b`), "lights|tv"},
	{regexp.MustCompile(`(?i)\barduino|\besp32|\besp8266|\bpico\b|\bbootloader|\bbootsel|\bcom port|\bgpio`), "mcu"},
	{regexp.MustCompile(`(?i)\bheaters?\b|\belement\b`), "heater"},
	{regexp.MustCompile(`(?i)\btrigger\b`), "powertool"},
	{regexp.MustCompile(`(?i)\bbuttons?\b`), "buttons|touch|phone|pc|console|mcu"},
}

// wordAllow: phrases that are fine for any device, though they contain a
// word above: the person's own phone used as a magnifier, the meter's own
// display, a PC's coin battery, the RCD's test button, the meter's ports,
// mains as what an adapter plugs into, and symptom names that list examples.
var wordAllow = regexp.MustCompile(`(?i)your phone's camera|coin battery|test button|monitor or TV|HDMI cable and port|sealed adapters: they hold mains voltage|'TV, monitor or display'|no sound, nothing happens|too hot to touch|USB, Wi-Fi or Bluetooth|it won't pair|no COM port|your other phones, tablets and computers`)

// TestWording reads everything a person can see on every step and fix, for
// every kind of device and every combination of power and features, and
// reports text that mentions something the device doesn't have.
func TestWording(t *testing.T) {
	g := loadGuide(t)
	type line struct{ cond, text string }
	parse := func(raw []json.RawMessage) []line {
		var out []line
		for _, r := range raw {
			var x string
			if json.Unmarshal(r, &x) == nil {
				out = append(out, line{"", x})
				continue
			}
			var c struct{ If, T string }
			_ = json.Unmarshal(r, &c)
			out = append(out, line{c.If, c.T})
		}
		return out
	}
	doL, fixL := map[string][]line{}, map[string][]line{}
	for id, s := range g.Steps {
		doL[id], fixL[id] = parse(s.Do), parse(s.Fix)
	}
	kinds := map[string]bool{"": true}
	for _, c := range g.Categories {
		kinds[c.Kind] = true
	}
	var feats []string
	for _, f := range g.Features {
		if f.ID != "heatmains" {
			feats = append(feats, f.ID)
		}
	}
	found := map[string]string{}
	cache := map[string][]int{}
	powers := [][]string{{"mains"}, {"adapter"}, {"usbpow"}, {"battery"}, {"lipo"}, {"adapter", "lipo"}, {"usbpow", "lipo"}, {"mains", "lipo"}}
	for kind := range kinds {
		for _, pw := range powers {
			for mask := 0; mask < 1<<len(feats); mask++ {
				var has []string
				for i, f := range feats {
					if mask&(1<<i) != 0 {
						has = append(has, f)
					}
				}
				tags := g.tags(kind, pw, has)
				check := func(id, where, text string) {
					hits, ok := cache[text]
					if !ok {
						plain := wordAllow.ReplaceAllString(text, "")
						for i, r := range wordRules {
							if r.re.MatchString(plain) {
								hits = append(hits, i)
							}
						}
						cache[text] = hits
					}
					for _, i := range hits {
						r := wordRules[i]
						if !match(r.need, tags) {
							key := fmt.Sprintf("%s %s: %q needs %s", id, where, r.re.FindString(text), r.need)
							if _, ok := found[key]; !ok {
								found[key] = fmt.Sprintf("%s\n        e.g. kind %q, power %v, has %v", text, kind, pw, has)
							}
						}
					}
				}
				seen := map[string]bool{}
				var walk func(id string)
				walk = func(id string) {
					if seen[id] {
						return
					}
					seen[id] = true
					s := g.Steps[id]
					if !s.Result && s.Only != "" && !match(s.Only, tags) {
						walk(s.Skip)
						return
					}
					check(id, "title", s.T)
					for _, x := range []string{s.Why, s.Q, s.Good, s.Bad, s.Cause, s.Pro} {
						check(id, "text", x)
					}
					for _, l := range append(append([]line{}, doL[id]...), fixL[id]...) {
						if match(l.cond, tags) {
							check(id, "line", l.text)
						}
					}
					if m := s.Measure; m != nil {
						check(id, "reading", m.Label)
						for _, b := range m.Bands {
							check(id, "reading", b.Say)
						}
						if m.Near != nil {
							check(id, "reading", m.Near.Label)
						}
					}
					for _, a := range s.A {
						if match(a.Only, tags) {
							check(id, "answer", a.T)
							walk(a.Go)
						}
					}
					if s.Next != "" {
						walk(s.Next)
					}
				}
				for _, sy := range g.Symptoms {
					if match(sy.For, tags) {
						check("symptom "+sy.ID, "name", sy.Name+" "+sy.Desc)
						walk(sy.Start)
					}
				}
			}
		}
	}
	keys := make([]string, 0, len(found))
	for k := range found {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		t.Errorf("%s\n        %s", k, found[k])
	}
}
