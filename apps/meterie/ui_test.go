//go:build !windows

package meterie

import (
	"testing"
	"time"
)

type uiDriver struct {
	t *testing.T
	a *App
	c *Canvas
}

func newDriver(t *testing.T) *uiDriver {
	drainInbox()
	a := NewApp()
	a.settings.ShowWelcome = false
	a.settings.APIEnabled = false
	a.ui.Rects = map[string]Rect{}
	d := &uiDriver{t: t, a: a, c: NewCanvas(1440, 900)}
	d.frame(Input{MX: -1, MY: -1})
	return d
}

func (d *uiDriver) frame(in Input) {
	d.a.lastFrame = time.Now().Add(-20 * time.Millisecond)
	d.a.Frame(d.c, &previewText{}, in)
}

func (d *uiDriver) click(id string) {
	d.frame(Input{MX: -1, MY: -1}) // make sure rects are current
	r, ok := d.a.ui.Rects[id]
	if !ok {
		d.t.Fatalf("widget %q not on screen", id)
	}
	x, y := r.CX(), r.CY()
	d.frame(Input{MX: x, MY: y, HasMouse: true})
	d.frame(Input{MX: x, MY: y, HasMouse: true, Down: true, Pressed: true})
	d.frame(Input{MX: x, MY: y, HasMouse: true, Released: true})
	d.frame(Input{MX: x, MY: y, HasMouse: true})
}

func (d *uiDriver) typeText(s string, keys ...Key) {
	d.frame(Input{MX: -1, MY: -1, Chars: []rune(s), Keys: keys})
}

func TestUIFlows(t *testing.T) {
	d := newDriver(t)
	a := d.a
	// navigate to Meters via the rail, open Demo tab, start the solar demo (two channels)
	d.click("rail1")
	if a.page != PageMeters {
		t.Fatalf("rail click failed, page=%d", a.page)
	}
	d.click("mt#tabs#3")
	if a.mp.tab != 3 {
		t.Fatalf("tab click failed")
	}
	d.click("demo#6")
	d.click("demo#start")
	if len(a.devices) != 1 || len(a.channels) != 2 || a.page != PageLive {
		t.Fatalf("demo start: devices=%d channels=%d page=%d", len(a.devices), len(a.channels), a.page)
	}
	// let the simulator run and readings flow in
	time.Sleep(700 * time.Millisecond)
	d.frame(Input{MX: -1, MY: -1})
	if !a.channels[0].HasLast || !a.channels[1].HasLast {
		t.Fatalf("no demo readings")
	}
	// add a power math channel through the UI: dropdowns default to A=ch1, B=ch2
	d.click("rail4")
	d.click("math#add")
	var mc *Channel
	for _, c := range a.channels {
		if c.Kind == KindMath {
			mc = c
		}
	}
	if mc == nil {
		t.Fatalf("math channel not added")
	}
	time.Sleep(600 * time.Millisecond)
	d.frame(Input{MX: -1, MY: -1})
	if !mc.HasLast || mc.Last.Unit != "W" {
		t.Fatalf("math channel produced nothing: %+v", mc.Last)
	}
	// dropdown: open preset list, pick "Custom formula", type a formula, add
	d.click("math#preset")
	if a.ui.popupID != "math#preset" {
		t.Fatalf("dropdown didn't open")
	}
	d.click("math#preset#i7")
	if a.math.preset != 7 || a.ui.popupID != "" {
		t.Fatalf("dropdown selection failed: %d", a.math.preset)
	}
	d.click("math#expr")
	d.typeText("ch1*2")
	if a.math.expr != "ch1*2" {
		t.Fatalf("typing failed: %q", a.math.expr)
	}
	d.typeText("", KBack)
	d.typeText("3")
	if a.math.expr != "ch1*3" {
		t.Fatalf("backspace failed: %q", a.math.expr)
	}
	// clicking the open dropdown's header closes it (and doesn't reopen)
	d.click("math#preset")
	d.click("math#preset")
	if a.ui.popupID != "" {
		t.Fatalf("dropdown header toggle broken")
	}
	// marker dialog via header button, type a label, confirm with Enter
	n := len(a.markers)
	d.click("hdr#marker")
	if a.modal != "marker" {
		t.Fatalf("marker modal not open")
	}
	d.typeText("probe moved", KEnter)
	if len(a.markers) != n+1 || a.markers[len(a.markers)-1].Text != "probe moved" {
		t.Fatalf("marker not added: %+v", a.markers)
	}
	// alarms: add alarm, then visit every page once to be sure nothing panics
	d.click("rail3")
	d.click("al#add")
	if len(a.alarms) != 1 {
		t.Fatalf("alarm not added")
	}
	for p := 0; p < pageCount; p++ {
		d.click("rail" + itoa(p))
	}
	for i := 0; i < len(wizardInfo); i++ {
		d.click("rail5")
		d.click("wz#" + itoa(i))
		if a.wiz.which != i {
			t.Fatalf("wizard %d didn't open", i)
		}
		d.frame(Input{MX: -1, MY: -1})
		d.click("wz#back")
	}
	for _, dev := range a.devices {
		dev.Stop()
	}
}
