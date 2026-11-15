package fixie

import (
	"encoding/json"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

// The error log explainer's rules (web/logrules.json, from logs.py): every
// pattern must work in Go's RE2 as well as JavaScript (so it stays simple and
// portable), match its own examples, and the scanner must give the expected
// findings for each sample log (logscan_check.js, run with Node when it's
// installed; the website workflow always runs it).
func TestLogRules(t *testing.T) {
	var db struct {
		Rules []struct {
			ID, OS, Sev, Cat, T, Means, Re, Kw string
			Do, Ex                              []string
		}
	}
	b, err := os.ReadFile("web/logrules.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &db); err != nil {
		t.Fatal(err)
	}
	if len(db.Rules) < 150 {
		t.Fatalf("only %d rules", len(db.Rules))
	}
	seen := map[string]bool{}
	for _, r := range db.Rules {
		if seen[r.ID] {
			t.Errorf("%s: used twice", r.ID)
		}
		seen[r.ID] = true
		if r.T == "" || r.Means == "" || len(r.Do) == 0 || len(r.Ex) == 0 {
			t.Errorf("%s: needs a title, meaning, what to do and examples", r.ID)
		}
		if !strings.Contains("windows macos linux android any", r.OS) {
			t.Errorf("%s: unknown system %q", r.ID, r.OS)
		}
		rx, err := regexp.Compile("(?i)" + r.Re)
		if err != nil {
			t.Errorf("%s: not a Go regular expression: %v", r.ID, err)
			continue
		}
		for _, e := range r.Ex {
			if !rx.MatchString(e) {
				t.Errorf("%s: example doesn't match in Go: %s", r.ID, e)
			}
			if r.Kw != "" && !strings.Contains(strings.ToLower(e), r.Kw) {
				t.Errorf("%s: example lacks keyword %q: %s", r.ID, r.Kw, e)
			}
		}
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Log("node isn't installed: skipping the JavaScript scanner check")
		return
	}
	out, err := exec.Command(node, "logscan_check.js").CombinedOutput()
	if err != nil {
		t.Fatalf("logscan_check.js:\n%s", out)
	}
	t.Logf("%s", out)
}

func TestDecodeText(t *testing.T) {
	u16 := []byte{0xFF, 0xFE, 'h', 0, 'i', 0}
	if decodeText(u16) != "hi" || decodeText([]byte("\xEF\xBB\xBFok")) != "ok" || decodeText([]byte("<Event>")) != "<Event>" {
		t.Fatal("decodeText")
	}
	if got := keepEnd(strings.Repeat("x\n", maxLogSource)); len(got) > maxLogSource {
		t.Fatal("keepEnd")
	}
}
