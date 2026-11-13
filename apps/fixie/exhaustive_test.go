// The full check: every described device in all three guides (every kind of
// device, every combination of power and every combination of features, about
// a million), walked through every path, with every word and picture it would
// see checked against what it has. Created by nrmluvdev.
//
// Devices that answer every condition in the guide the same way see exactly
// the same steps, text and pictures, so each such group is walked once.
package fixie

import (
	"encoding/json"
	"fmt"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
)

// cond is a compiled condition: any term matches (all of req, none of forb).
type cond []struct{ req, forb uint64 }

type condSet struct {
	bit   map[string]uint
	conds map[string]cond
	list  []string
}

func (cs *condSet) tag(t string) uint64 {
	b, ok := cs.bit[t]
	if !ok {
		b = uint(len(cs.bit))
		cs.bit[t] = b
	}
	return 1 << b
}

func (cs *condSet) add(c string) {
	if _, ok := cs.conds[c]; ok || c == "" {
		return
	}
	var out cond
	for _, term := range strings.Split(c, "|") {
		var x struct{ req, forb uint64 }
		for _, t := range strings.Split(term, "&") {
			if strings.HasPrefix(t, "!") {
				x.forb |= cs.tag(t[1:])
			} else {
				x.req |= cs.tag(t)
			}
		}
		out = append(out, x)
	}
	cs.conds[c] = out
	cs.list = append(cs.list, c)
}

func (cs *condSet) is(c string, tags uint64) bool {
	if c == "" {
		return true
	}
	for _, x := range cs.conds[c] {
		if tags&x.req == x.req && tags&x.forb == 0 {
			return true
		}
	}
	return false
}

type line struct{ cond, text string }

func parseLines(raw []json.RawMessage) []line {
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

func TestEverything(t *testing.T) {
	g := loadGuide(t)
	var picInfo map[string]struct {
		KW   []string `json:"kw"`
		Need string   `json:"need"`
	}
	{
		var raw struct {
			PicInfo json.RawMessage `json:"picInfo"`
		}
		b, _ := jsonFile("web/guide.json")
		_ = json.Unmarshal(b, &raw)
		_ = json.Unmarshal(raw.PicInfo, &picInfo)
	}
	cs := &condSet{bit: map[string]uint{}, conds: map[string]cond{}}
	doL, fixL := map[string][]line{}, map[string][]line{}
	for id, s := range g.Steps {
		doL[id], fixL[id] = parseLines(s.Do), parseLines(s.Fix)
		cs.add(s.Only)
		for _, a := range s.A {
			cs.add(a.Only)
		}
		for _, l := range append(append([]line{}, doL[id]...), fixL[id]...) {
			cs.add(l.cond)
		}
		for _, p := range s.Pics {
			_, c := picIf(p)
			cs.add(c)
		}
	}
	for _, st := range g.starts() {
		cs.add(st.For)
	}
	for _, r := range wordRules {
		cs.add(r.need)
	}
	for _, p := range picInfo {
		cs.add(p.Need)
	}
	for _, d := range g.Derived {
		cs.tag(d.Tag)
	}
	for _, st := range g.starts() {
		if st.Implies != "" {
			cs.tag(st.Implies)
		}
	}
	if len(cs.bit) > 64 {
		t.Fatalf("%d tags: too many for the bitmask", len(cs.bit))
	}

	// every way to describe a device
	var feats []string
	for _, f := range g.Features {
		if f.ID != "heatmains" { // refused before any step
			feats = append(feats, f.ID)
		}
	}
	var powers []string
	for _, p := range g.Power {
		powers = append(powers, p.ID)
	}
	kinds := []string{""}
	for _, c := range g.Categories {
		if c.Kind != "" {
			kinds = append(kinds, c.Kind)
		}
	}
	modes := map[string][]gStart{}
	for _, st := range g.starts() {
		m := strings.Fields(st.ID)[0]
		modes[m] = append(modes[m], st)
	}
	type rep struct {
		kind       string
		power, has []string
		tags       uint64
		n          int
	}
	groups := map[string]map[string]*rep{} // mode -> fingerprint -> example
	total := 0
	for mode := range modes {
		groups[mode] = map[string]*rep{}
		ks := kinds
		if mode != "symptom" { // Board level and My build don't ask for a category
			ks = []string{""}
		}
		for _, kind := range ks {
			for pm := 1; pm < 1<<len(powers); pm++ { // at least one power source
				for fm := 0; fm < 1<<len(feats); fm++ {
					var pw, has []string
					for i, p := range powers {
						if pm&(1<<i) != 0 {
							pw = append(pw, p)
						}
					}
					for i, f := range feats {
						if fm&(1<<i) != 0 {
							has = append(has, f)
						}
					}
					var tags uint64
					for x := range g.tags(kind, pw, has) {
						tags |= cs.tag(x)
					}
					fp := make([]byte, (len(cs.list)+7)/8)
					for i, c := range cs.list {
						if cs.is(c, tags) {
							fp[i/8] |= 1 << (i % 8)
						}
					}
					k := string(fp)
					if r, ok := groups[mode][k]; ok {
						r.n++
					} else {
						groups[mode][k] = &rep{kind, pw, has, tags, 1}
					}
					total++
				}
			}
		}
	}

	var mu sync.Mutex
	problems := map[string]string{}
	reached := map[string]bool{}
	report := func(key, example string) {
		mu.Lock()
		if _, ok := problems[key]; !ok {
			problems[key] = example
		}
		mu.Unlock()
	}
	// which word rules each piece of text trips, worked out once up front
	ruleHits := map[string][]int{}
	addText := func(text string) {
		if _, ok := ruleHits[text]; ok {
			return
		}
		plain := wordAllow.ReplaceAllString(text, "")
		var h []int
		for i, r := range wordRules {
			if r.re.MatchString(plain) {
				h = append(h, i)
			}
		}
		ruleHits[text] = h
	}
	for id, s := range g.Steps {
		for _, x := range []string{s.T, s.Why, s.Q, s.Good, s.Bad, s.Cause, s.Pro} {
			addText(x)
		}
		for _, l := range append(append([]line{}, doL[id]...), fixL[id]...) {
			addText(l.text)
		}
		for _, a := range s.A {
			addText(a.T)
		}
		if m := s.Measure; m != nil {
			addText(m.Label)
			for _, b := range m.Bands {
				addText(b.Say)
			}
		}
	}
	hits := func(text string) []int { return ruleHits[text] }
	walkOne := func(mode string, r *rep, seen map[string]bool) {
		base, tags := r.tags, r.tags
		ex := fmt.Sprintf("%s: kind %q, power %v, has %v (and %d more like it)", mode, r.kind, r.power, r.has, r.n-1)
		read := func(id, where, text string) {
			for _, i := range hits(text) {
				if !cs.is(wordRules[i].need, tags) {
					report(fmt.Sprintf("%s %s: %q needs %s\n        %s", id, where, wordRules[i].re.FindString(wordAllow.ReplaceAllString(text, "")), wordRules[i].need, text), ex)
				}
			}
		}
		state := map[string]int{}
		var walk func(id string)
		walk = func(id string) {
			switch state[id] {
			case 1:
				report("loop at "+id, ex)
				return
			case 2:
				return
			}
			state[id] = 1
			defer func() { state[id] = 2 }()
			s, ok := g.Steps[id]
			if !ok {
				report("missing step "+id, ex)
				return
			}
			if !s.Result && s.Only != "" && !cs.is(s.Only, tags) {
				walk(s.Skip)
				return
			}
			seen[id] = true
			read(id, "title", s.T)
			for _, x := range []string{s.Why, s.Q, s.Good, s.Bad, s.Cause, s.Pro} {
				read(id, "text", x)
			}
			lines := 0
			for _, l := range append(append([]line{}, doL[id]...), fixL[id]...) {
				if cs.is(l.cond, tags) {
					lines++
					read(id, "line", l.text)
				}
			}
			if lines == 0 {
				report("step "+id+" has nothing to do or fix", ex)
			}
			for _, p := range s.Pics {
				pid, c := picIf(p)
				if !cs.is(c, tags) {
					continue
				}
				if need := picInfo[pid].Need; !cs.is(need, tags) {
					report(fmt.Sprintf("%s shows picture %s, which needs %s", id, pid, need), ex)
				}
			}
			if m := s.Measure; m != nil {
				read(id, "reading", m.Label)
				for _, b := range m.Bands {
					read(id, "reading", b.Say)
				}
			}
			if s.Result {
				if s.Next != "" {
					walk(s.Next)
				}
				return
			}
			n := 0
			for _, a := range s.A {
				if cs.is(a.Only, tags) {
					n++
					read(id, "answer", a.T)
					walk(a.Go)
				}
			}
			if n == 0 {
				report("step "+id+" has no answers", ex)
			}
		}
		for _, st := range modes[mode] {
			if cs.is(st.For, base) {
				state = map[string]int{}
				tags = base
				if st.Implies != "" {
					tags |= cs.tag(st.Implies)
				}
				walk(st.Start)
			}
		}
	}
	jobs := make(chan [2]interface{}, 256)
	var wg sync.WaitGroup
	for w := 0; w < runtime.NumCPU(); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			seen := map[string]bool{}
			for j := range jobs {
				walkOne(j[0].(string), j[1].(*rep), seen)
			}
			mu.Lock()
			for id := range seen {
				reached[id] = true
			}
			mu.Unlock()
		}()
	}
	walked := 0
	for mode, gr := range groups {
		for _, r := range gr {
			walked++
			jobs <- [2]interface{}{mode, r}
		}
	}
	close(jobs)
	wg.Wait()
	t.Logf("%d described devices in %d groups that see the same guide", total, walked)
	keys := make([]string, 0, len(problems))
	for k := range problems {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		t.Errorf("%s\n        e.g. %s", k, problems[k])
	}
	for id := range g.Steps {
		if !reached[id] {
			t.Errorf("step %s can't be reached by any device", id)
		}
	}
}

// TestPictureRelevance checks every picture on a step or part is about what
// that step or part talks about: one of the picture's subject words must be in
// its text.
func TestPictureRelevance(t *testing.T) {
	g := loadGuide(t)
	var raw struct {
		PicInfo map[string]struct {
			KW []string `json:"kw"`
		} `json:"picInfo"`
		Parts []struct {
			ID, Name, What string
			Pic            string
			Tests          []struct {
				T, Good, Bad, Pic string
				Do                []string
			}
		} `json:"parts"`
	}
	b, _ := jsonFile("web/guide.json")
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	about := func(where, pic, text string) {
		info, ok := raw.PicInfo[pic]
		if !ok {
			t.Errorf("%s: picture %s has no subject words", where, pic)
			return
		}
		low := strings.ToLower(text)
		for _, k := range info.KW {
			if strings.Contains(low, k) {
				return
			}
		}
		t.Errorf("%s: picture %s (about %s) doesn't match what it says", where, pic, strings.Join(info.KW, ", "))
	}
	for id, s := range g.Steps {
		text := s.T + " " + s.Why + " " + s.Q + " " + s.Good + " " + s.Bad
		for _, l := range parseLines(s.Do) {
			text += " " + l.text
		}
		for _, a := range s.A {
			text += " " + a.T
		}
		for _, a := range s.Apps {
			text += " " + a.How
		}
		for _, p := range s.Pics {
			about(id, picID(p), text)
		}
	}
	for _, p := range raw.Parts {
		all := p.Name + " " + p.What
		for _, x := range p.Tests {
			txt := x.T + " " + x.Good + " " + x.Bad + " " + strings.Join(x.Do, " ")
			all += " " + txt
			if x.Pic != "" {
				about("part "+p.ID+" test "+x.T, x.Pic, txt+" "+p.Name)
			}
		}
		if p.Pic != "" {
			about("part "+p.ID, p.Pic, all)
		}
	}
}
