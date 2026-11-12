package launcher

// Every app keeps its own copy of the shared teaching mode (teach.js) and Look &
// accessibility (look.js) scripts. This test makes sure no copy has drifted.

import (
	"bytes"
	"os"
	"testing"
)

func TestSharedCopies(t *testing.T) {
	copies := map[string][]string{
		"../shared/teach.js": {"../apps/codie/web", "../apps/fixie/web", "../apps/makie/web", "../apps/probie/web", "../apps/tunie/web", "../apps/wirie/web", "../apps/flashie"},
		"../shared/look.js":  {"../apps/codie/web", "../apps/fixie/web", "../apps/makie/web", "../apps/probie/web", "../apps/scopie/web", "../apps/tunie/web", "../apps/wirie/web", "../apps/flashie", "../apps/meterie/web", "web"},
	}
	for src, dirs := range copies {
		want, err := os.ReadFile(src)
		if err != nil {
			t.Fatal(err)
		}
		name := src[len("../shared/"):]
		for _, d := range dirs {
			got, err := os.ReadFile(d + "/" + name)
			if err != nil || !bytes.Equal(got, want) {
				t.Errorf("%s/%s differs from %s: copy it again (cp %s %s/)", d, name, src, src, d)
			}
		}
	}
}
