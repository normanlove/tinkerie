package geom

// feet.go: feet under Wirie cases and keyboard cases: printed feet, shallow
// indents that locate stick-on rubber feet, or none at all. Keyboard cases can
// also tilt towards you on taller printed feet at the back.
// Created by nrmluvdev for Tinkerie.

import (
	"fmt"
	"math"
	"strings"
)

// Feet choices, and the common sizes of stick-on rubber feet ("bumpons"):
// diameter × height in mm.
var feetStyles = []string{"Rubber feet in indents", "Printed feet", "No feet"}

type footSize struct {
	name string
	d, h float64
}

var footSizes = []footSize{
	{"8 × 2 mm (small round)", 8, 2},
	{"10 × 3 mm", 10, 3},
	{"12.7 × 3.5 mm (½ inch)", 12.7, 3.5},
	{"6.4 × 1.9 mm (tiny)", 6.4, 1.9},
	{"20 × 5 mm (large)", 20, 5},
}

func footSizeNames() []string {
	out := make([]string, len(footSizes))
	for i, f := range footSizes {
		out[i] = f.name
	}
	return out
}

func footSizeOf(s string) footSize {
	for _, f := range footSizes {
		if f.name == s {
			return f
		}
	}
	return footSizes[1]
}

// feetParams: the settings, shown in the case panel.
func feetParams(def string) []Param {
	return []Param{
		sel("feet", "Feet", def, feetStyles...).help("Rubber feet in indents: shallow round pockets under the case so stick-on rubber feet sit in the right place and don't slide off. Printed feet: little round feet printed with the case. No feet: a flat bottom."),
		sel("footSize", "Foot size", footSizes[1].name, footSizeNames()...).help("Matches the common stick-on rubber feet (bumpons). Printed feet use the same size.").show("feet!=No feet"),
	}
}

// footSpots: where feet go under an outline: near each corner, and halfway
// along on long cases, clear of the edge and of screw holes. rearOnly picks the
// two at the back (high Y), frontOnly the ones at the front.
func footSpots(outer Loop, r float64, avoid []P2, which string) []P2 {
	x0, y0, x1, y1 := loopBox(outer)
	in := r + 3
	targets := []P2{{x0 + in, y0 + in}, {x1 - in, y0 + in}, {x0 + in, y1 - in}, {x1 - in, y1 - in}}
	if x1-x0 > 180 { // long cases: one more in the middle, front and back
		targets = append(targets, P2{(x0 + x1) / 2, y0 + in}, P2{(x0 + x1) / 2, y1 - in})
	}
	cx, cy := (x0+x1)/2, (y0+y1)/2
	var out []P2
	for _, q := range targets {
		back := q.Y > cy
		if (which == "rear" && !back) || (which == "front" && back) {
			continue
		}
		ok := func(q P2) bool {
			if !outer.Contains(q) || loopEdgeDist(outer, q) < r+1.5 {
				return false
			}
			for _, a := range avoid {
				if math.Hypot(a.X-q.X, a.Y-q.Y) < r+2.5 {
					return false
				}
			}
			return true
		}
		for i := 0; i < 60 && !ok(q); i++ { // slide towards the middle until it fits
			q = P2{q.X + (cx-q.X)*0.06, q.Y + (cy-q.Y)*0.06}
		}
		if ok(q) {
			out = append(out, q)
		}
	}
	return out
}

// addFeet adds the chosen feet under a case whose underside is at zBot.
// lift is how far the case already stands on other feet (for keyboards'
// rear tilt feet); it returns how far these feet raise the case.
func addFeet(base *Node, outer Loop, zBot, floor float64, p Params, avoid []P2, which, def string, info *CaseInfo) float64 {
	style := p.S("feet", def)
	if style == "No feet" {
		return 0
	}
	fs := footSizeOf(p.S("footSize", footSizes[1].name))
	r := fs.d / 2
	spots := footSpots(outer, r, avoid, which)
	if len(spots) == 0 {
		info.Warnings = append(info.Warnings, "There's no room for feet under this case.")
		return 0
	}
	f3 := func(x, y, z float64) [3]float64 { return [3]float64{round3(x), round3(y), round3(z)} }
	zero := [3]float64{}
	if style == "Printed feet" {
		for i, q := range spots {
			base.Children = append(base.Children, newNode(fmt.Sprintf("Foot %d", i+1), "cylinder", Params{"d": round3(fs.d), "h": round3(fs.h + 0.3)}, f3(q.X, q.Y, zBot-fs.h), zero, false, ""))
		}
		info.Lines = append(info.Lines, fmt.Sprintf("%d printed feet, %g mm across and %g mm tall.", len(spots), fs.d, fs.h))
		return fs.h
	}
	// rubber feet: a shallow pocket each, leaving at least 0.8 mm of floor
	depth := math.Min(fs.h*0.45, floor-0.8)
	if depth < 0.4 {
		info.Warnings = append(info.Warnings, fmt.Sprintf("The floor is too thin for rubber-foot indents: make it at least %.1f mm.", 0.8+0.4))
		return fs.h
	}
	for i, q := range spots {
		base.Children = append(base.Children, newNode(fmt.Sprintf("Rubber foot indent %d", i+1), "cylinder", Params{"d": round3(fs.d + 0.6), "h": round3(depth + 0.3)}, f3(q.X, q.Y, zBot-0.3), zero, true, ""))
	}
	info.Lines = append(info.Lines, fmt.Sprintf("%d indents %.1f mm deep for %s rubber feet: stick one in each.", len(spots), depth, strings.Split(fs.name, " (")[0]))
	return fs.h - depth
}

// rubberIndentUnder: a pocket for a rubber foot under a printed foot (keyboard
// tilt feet), so the back doesn't slide either.
func rubberIndentUnder(base *Node, q P2, zBot float64, p Params, name, def string) {
	if p.S("feet", def) != "Rubber feet in indents" {
		return
	}
	fs := footSizeOf(p.S("footSize", footSizes[1].name))
	depth := math.Min(fs.h*0.45, 1.2)
	base.Children = append(base.Children, newNode(name, "cylinder", Params{"d": round3(math.Min(fs.d+0.6, 11.4)), "h": round3(depth + 0.3)},
		[3]float64{round3(q.X), round3(q.Y), round3(zBot - 0.3)}, [3]float64{}, true, ""))
}
