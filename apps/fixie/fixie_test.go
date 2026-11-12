// Checks Fixie's repair guide: every path through it, for every device and
// symptom, must end at a result, and everything it refers to must exist.
// Created by nrmluvdev.
package fixie

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

type gAnswer struct {
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
	Do      []json.RawMessage `json:"do"`
	Q       string            `json:"q"`
	A       []gAnswer         `json:"a"`
	Tools   []string          `json:"tools"`
	Signs   []string          `json:"signs"`
	Open    bool              `json:"open"`
	Only    string            `json:"only"`
	Skip    string            `json:"skip"`
	Next    string            `json:"next"`
	Fix     []string          `json:"fix"`
	Measure *struct {
		Label string  `json:"label"`
		Bands []gBand `json:"bands"`
		Near  *struct {
			Ok  int `json:"ok"`
			Bad int `json:"bad"`
		} `json:"near"`
		Calc string `json:"calc"`
	} `json:"measure"`
}
type guide struct {
	Devices []struct {
		ID     string   `json:"id"`
		Tags   []string `json:"tags"`
		NoOpen bool     `json:"noopen"`
	} `json:"devices"`
	Symptoms []struct {
		ID    string `json:"id"`
		For   string `json:"for"`
		Start string `json:"start"`
	} `json:"symptoms"`
	Steps map[string]gStep `json:"steps"`
	Tools []struct {
		ID string `json:"id"`
	} `json:"tools"`
	Signs []struct {
		ID string `json:"id"`
	} `json:"signs"`
}

// match is the same test as the window's: "a|b" is any of them, "!a" is not.
func match(cond string, tags map[string]bool) bool {
	if cond == "" {
		return true
	}
	for _, c := range strings.Split(cond, "|") {
		if strings.HasPrefix(c, "!") {
			if !tags[c[1:]] {
				return true
			}
		} else if tags[c] {
			return true
		}
	}
	return false
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
	for _, d := range g.Devices {
		known[d.ID] = true
		for _, tg := range d.Tags {
			known[tg] = true
		}
	}
	checkCond := func(where, c string) {
		for _, p := range strings.Split(c, "|") {
			if p = strings.TrimPrefix(p, "!"); p != "" && !known[p] {
				t.Errorf("%s: unknown tag %q", where, p)
			}
		}
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
		for _, raw := range s.Do {
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

// TestGuidePaths follows every answer, for every device and the symptoms it
// offers, and checks each path ends at a result without going round in a loop.
func TestGuidePaths(t *testing.T) {
	g := loadGuide(t)
	reached := map[string]bool{}
	for _, d := range g.Devices {
		tags := map[string]bool{d.ID: true}
		for _, tg := range d.Tags {
			tags[tg] = true
		}
		var walk func(id string, path []string)
		walk = func(id string, path []string) {
			for _, p := range path {
				if p == id {
					t.Errorf("%s: loop %s → %s", d.ID, strings.Join(path, " → "), id)
					return
				}
			}
			path = append(path, id)
			s := g.Steps[id]
			reached[id] = true
			if s.Result {
				return
			}
			if s.Only != "" && !match(s.Only, tags) {
				walk(s.Skip, path)
				return
			}
			if s.Open && d.NoOpen {
				walk("r.pro", path)
				return
			}
			n := 0
			for _, a := range s.A {
				if match(a.Only, tags) {
					n++
					walk(a.Go, path)
				}
			}
			if n == 0 {
				t.Errorf("%s: step %s has no answers for this device", d.ID, id)
			}
		}
		for _, s := range g.Symptoms {
			if match(s.For, tags) {
				walk(s.Start, nil)
			}
		}
	}
	for id := range g.Steps {
		if !reached[id] {
			t.Errorf("step %s can't be reached from any device and symptom", id)
		}
	}
}
