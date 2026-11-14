package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestUSBRulesCoverEveryDevice checks that build/install-usb.sh (the Linux
// and Raspberry Pi permission rule) covers every USB scope, logic analyser
// and HID meter cable the apps know, so none is found but then refused with
// "permission denied".
func TestUSBRulesCoverEveryDevice(t *testing.T) {
	b, err := os.ReadFile("build/install-usb.sh")
	if err != nil {
		t.Fatal(err)
	}
	script := string(b)
	vendors := map[string]bool{}
	if m := regexp.MustCompile(`for v in ([0-9a-f ]+); do`).FindStringSubmatch(script); m != nil {
		for _, v := range strings.Fields(m[1]) {
			vendors[v] = true
		}
	}
	pairs := map[string]bool{}
	for _, m := range regexp.MustCompile(`idVendor}==\\"([0-9a-f]{4})\\", ATTR{idProduct}==\\"([0-9a-f]{4})\\"`).FindAllStringSubmatch(script, -1) {
		pairs[m[1]+":"+m[2]] = true
	}
	hidraw := map[string]bool{}
	if m := regexp.MustCompile(`for vp in ([0-9a-f: ]+); do`).FindStringSubmatch(script); m != nil {
		for _, vp := range strings.Fields(m[1]) {
			hidraw[vp] = true
		}
	}
	if len(vendors) == 0 || len(hidraw) == 0 {
		t.Fatal("couldn't read the rules from install-usb.sh")
	}
	pair := regexp.MustCompile(`0x([0-9A-Fa-f]{4}),\s*0x([0-9A-Fa-f]{4})`)
	for _, app := range []string{"scopie", "probie"} {
		files, _ := filepath.Glob("apps/" + app + "/dev_*.go")
		more, _ := filepath.Glob("apps/" + app + "/fx2*.go")
		for _, f := range append(files, more...) {
			if strings.HasSuffix(f, "_test.go") {
				continue
			}
			src, _ := os.ReadFile(f)
			for _, line := range strings.Split(string(src), "\n") {
				if !strings.HasPrefix(strings.TrimSpace(line), "{") {
					continue // only table entries
				}
				for _, m := range pair.FindAllStringSubmatch(line, -1) {
					v, p := strings.ToLower(m[1]), strings.ToLower(m[2])
					if !vendors[v] && !pairs[v+":"+p] {
						t.Errorf("%s: USB %s:%s isn't in install-usb.sh", f, v, p)
					}
				}
			}
		}
	}
	src, _ := os.ReadFile("apps/meterie/meters.go")
	hidKinds := regexp.MustCompile(`TrCH9325|TrCP2110|TrVictor|TrBU86X`)
	n := 0
	for _, line := range strings.Split(string(src), "\n") {
		if !hidKinds.MatchString(line) || !strings.HasPrefix(strings.TrimSpace(line), "{0x") {
			continue
		}
		m := pair.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		n++
		if vp := strings.ToLower(m[1] + ":" + m[2]); !hidraw[vp] {
			t.Errorf("Meterie HID cable %s isn't in install-usb.sh's hidraw rule", vp)
		}
	}
	if n == 0 {
		t.Fatal("found no Meterie HID cables to check")
	}
}
