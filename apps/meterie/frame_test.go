//go:build !windows

package meterie

import (
	"testing"
	"time"
)

func TestFrameSpeed(t *testing.T) {
	a := NewApp()
	a.settings.ShowWelcome = false
	a.settings.APIEnabled = false
	seedPreview(a)
	for p := 0; p < pageCount; p++ {
		a.page = p
		c := NewCanvas(1440, 900)
		st := time.Now()
		for i := 0; i < 3; i++ {
			a.Frame(c, &previewText{}, Input{MX: -100, MY: -100})
		}
		t.Logf("page %d: %v/frame", p, time.Since(st)/3)
	}
}
