// Tunie — 3D printer calibration helper
// Part of the nrmluvdev maker suite. Created by nrmluvdev.
//
// gcode.go: the G-code writer that every calibration test is built on.
// Model coordinates are always centred on the bed (0,0 = bed centre); the
// writer converts them to machine coordinates for the printer's origin.
package tunie

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// Printer describes the machine a test is generated for.
type Printer struct {
	Name           string  `json:"name"`
	Firmware       string  `json:"firmware"` // marlin | klipper | rrf | prusa-mk3 | prusa-buddy | bambu
	BedShape       string  `json:"bedShape"` // rect | circle
	BedX           float64 `json:"bedX"`     // width, or diameter for circular beds
	BedY           float64 `json:"bedY"`
	BedZ           float64 `json:"bedZ"`
	OriginCenter   bool    `json:"originCenter"`
	Nozzle         float64 `json:"nozzle"`
	FilamentDia    float64 `json:"filamentDia"`
	MaxHotend      float64 `json:"maxHotend"`
	MaxBed         float64 `json:"maxBed"`
	MaxSpeed       float64 `json:"maxSpeed"`
	MaxAccel       float64 `json:"maxAccel"`
	Drive          string  `json:"drive"` // direct | bowden
	StartGcode     string  `json:"startGcode"`
	EndGcode       string  `json:"endGcode"`
	HasInputShaper bool    `json:"hasInputShaper"`
}

// Settings are the shared print settings used by every printed test.
type Settings struct {
	Hotend       float64 `json:"hotend"`
	Bed          float64 `json:"bed"`
	LayerH       float64 `json:"layerH"`
	FirstLayerH  float64 `json:"firstLayerH"`
	LineW        float64 `json:"lineW"`
	Speed        float64 `json:"speed"`
	FirstSpeed   float64 `json:"firstSpeed"`
	TravelSpeed  float64 `json:"travelSpeed"`
	Fan          float64 `json:"fan"`
	RetractLen   float64 `json:"retractLen"`
	RetractSpeed float64 `json:"retractSpeed"`
	ZHop         float64 `json:"zHop"`
	Flow         float64 `json:"flow"`
	Accel        float64 `json:"accel"`
	Density      float64 `json:"density"`
	Mesh         string  `json:"mesh"` // none | load | probe
	FilamentName string  `json:"filamentName"`
}

type pt struct{ x, y float64 }

// GW is the G-code writer.
type GW struct {
	sb        strings.Builder
	p         Printer
	s         Settings
	title     string
	params    []string
	ox, oy    float64
	x, y, z   float64
	retracted bool
	hopped    bool
	lh, lw    float64
	flowMul   float64
	filArea   float64
	eTotal    float64
	timeS     float64
	layer     int
	lines     int
	warns     []string
	maxZ      float64
	curType   string
	fanPct    float64
	minX      float64
	maxX      float64
	minY      float64
	maxY      float64
	maxR      float64
	info      map[string]any
}

func newGW(p Printer, s Settings) *GW {
	g := &GW{p: p, s: s, flowMul: 1, lw: s.LineW, lh: s.FirstLayerH, info: map[string]any{}}
	r := p.FilamentDia / 2
	g.filArea = math.Pi * r * r
	if !p.OriginCenter {
		g.ox = p.BedX / 2
		g.oy = p.BedY / 2
	}
	g.minX, g.minY = math.Inf(1), math.Inf(1)
	g.maxX, g.maxY = math.Inf(-1), math.Inf(-1)
	return g
}

// num formats a float compactly (no trailing zeros).
func num(v float64, prec int) string {
	s := strconv.FormatFloat(v, 'f', prec, 64)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		s = strings.TrimSuffix(s, ".")
	}
	if s == "-0" {
		s = "0"
	}
	return s
}

func clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }

func (g *GW) w(format string, a ...any) {
	if len(a) == 0 {
		g.sb.WriteString(format)
	} else {
		fmt.Fprintf(&g.sb, format, a...)
	}
	g.sb.WriteByte('\n')
	g.lines++
}

func (g *GW) raw(block string) {
	for _, l := range strings.Split(strings.ReplaceAll(block, "\r", ""), "\n") {
		l = strings.TrimRight(l, " \t")
		if strings.TrimSpace(l) == "" {
			continue
		}
		g.w("%s", l)
	}
}

func (g *GW) warn(format string, a ...any) { g.warns = append(g.warns, fmt.Sprintf(format, a...)) }

func (g *GW) mx(x float64) float64 { return x + g.ox }
func (g *GW) my(y float64) float64 { return y + g.oy }

func (g *GW) track(x, y float64) {
	mx, my := g.mx(x), g.my(y)
	g.minX, g.maxX = math.Min(g.minX, mx), math.Max(g.maxX, mx)
	g.minY, g.maxY = math.Min(g.minY, my), math.Max(g.maxY, my)
	g.maxR = math.Max(g.maxR, math.Hypot(x, y))
}

func (g *GW) typ(t string) {
	if t != g.curType {
		g.curType = t
		if t != "" {
			g.w(";TYPE:%s", t)
		}
	}
}

func (g *GW) speed(v float64) float64 {
	if g.p.MaxSpeed > 0 && v > g.p.MaxSpeed {
		return g.p.MaxSpeed
	}
	if v < 1 {
		return 1
	}
	return v
}

func (g *GW) retract() {
	if g.retracted || g.s.RetractLen <= 0 {
		return
	}
	g.w("G1 E-%s F%d", num(g.s.RetractLen, 3), int(g.s.RetractSpeed*60))
	g.retracted = true
	g.timeS += g.s.RetractLen / g.s.RetractSpeed
}

func (g *GW) unretract() {
	if g.hopped {
		g.w("G0 Z%s F600", num(g.z, 3))
		g.hopped = false
	}
	if g.retracted {
		g.w("G1 E%s F%d", num(g.s.RetractLen, 3), int(g.s.RetractSpeed*60))
		g.retracted = false
		g.timeS += g.s.RetractLen / g.s.RetractSpeed
	}
}

// travel moves without extruding, retracting (and hopping) on longer moves.
func (g *GW) travel(x, y float64) {
	d := math.Hypot(x-g.x, y-g.y)
	if d < 0.001 {
		return
	}
	if d > 2.0 {
		g.retract()
		if g.s.ZHop > 0 && !g.hopped {
			g.w("G0 Z%s F600", num(g.z+g.s.ZHop, 3))
			g.hopped = true
		}
	}
	sp := g.speed(g.s.TravelSpeed)
	g.w("G0 X%s Y%s F%d", num(g.mx(x), 3), num(g.my(y), 3), int(sp*60))
	g.timeS += d / sp
	g.x, g.y = x, y
	g.track(x, y)
}

// extrudeTo prints a straight line from the current position.
func (g *GW) extrudeTo(x, y, sp float64) {
	d := math.Hypot(x-g.x, y-g.y)
	if d < 0.0005 {
		return
	}
	g.unretract()
	sp = g.speed(sp)
	e := d * g.lw * g.lh / g.filArea * (g.s.Flow / 100) * g.flowMul
	g.eTotal += e
	g.w("G1 X%s Y%s E%s F%d", num(g.mx(x), 3), num(g.my(y), 3), num(e, 5), int(sp*60))
	g.timeS += d / sp
	g.x, g.y = x, y
	g.track(x, y)
}

func (g *GW) newLayer(z, h float64) {
	g.z = z
	g.lh = h
	if z > g.maxZ {
		g.maxZ = z
	}
	g.w(";LAYER:%d", g.layer)
	g.layer++
	g.w("G0 Z%s F600", num(z, 3))
	g.hopped = false
	g.curType = ""
}

func (g *GW) setFan(pct float64) {
	pct = clamp(pct, 0, 100)
	if pct == g.fanPct {
		return
	}
	g.fanPct = pct
	if pct == 0 {
		g.w("M106 S0")
	} else {
		g.w("M106 S%d", int(math.Round(255*pct/100)))
	}
}

// layerFan switches the part fan on from the second layer.
func (g *GW) layerFan(i int) {
	if i == 0 {
		g.setFan(0)
	} else if i == 1 {
		g.setFan(g.s.Fan)
	}
}

func layerZs(total, first, lh float64) []float64 {
	zs := []float64{first}
	z := first
	for z+lh <= total+1e-6 {
		z += lh
		zs = append(zs, math.Round(z*10000)/10000)
	}
	return zs
}

// ---------------------------------------------------------------- shapes

func (g *GW) loop(pts []pt, sp float64) {
	if len(pts) < 2 {
		return
	}
	g.travel(pts[0].x, pts[0].y)
	for _, p := range pts[1:] {
		g.extrudeTo(p.x, p.y, sp)
	}
	g.extrudeTo(pts[0].x, pts[0].y, sp)
}

func (g *GW) rectLoop(x0, y0, x1, y1, sp float64) {
	g.loop([]pt{{x0, y0}, {x1, y0}, {x1, y1}, {x0, y1}}, sp)
}

// fillLines fills a region with zig-zag lines. Line centres span exactly the
// given rectangle. If solid, spacing is adjusted so the lines fit exactly.
func (g *GW) fillLines(x0, y0, x1, y1, spacing float64, horiz, solid bool, sp float64) {
	if x1 < x0-1e-6 || y1 < y0-1e-6 {
		return
	}
	a0, a1, b0, b1 := y0, y1, x0, x1 // a = across lines, b = along lines
	if !horiz {
		a0, a1, b0, b1 = x0, x1, y0, y1
	}
	span := a1 - a0
	var n int
	var step, start float64
	if solid {
		n = int(math.Round(span/spacing)) + 1
		if n < 1 {
			n = 1
		}
		if n > 1 {
			step = span / float64(n-1)
		}
		start = a0
		if n == 1 {
			start = (a0 + a1) / 2
		}
	} else {
		n = int(math.Floor(span/spacing+1e-6)) + 1
		step = spacing
		start = a0 + (span-float64(n-1)*spacing)/2
	}
	fwd := true
	for k := 0; k < n; k++ {
		a := start + float64(k)*step
		s, e := b0, b1
		if !fwd {
			s, e = b1, b0
		}
		var sx, sy, ex, ey float64
		if horiz {
			sx, sy, ex, ey = s, a, e, a
		} else {
			sx, sy, ex, ey = a, s, a, e
		}
		if k == 0 {
			g.travel(sx, sy)
		} else {
			g.extrudeTo(sx, sy, sp)
		}
		if math.Abs(e-s) < 0.01 {
			// a dot: extrude a tiny line so the spot is filled
			if horiz {
				ex = sx + 0.05
			} else {
				ey = sy + 0.05
			}
		}
		g.extrudeTo(ex, ey, sp)
		fwd = !fwd
	}
}

// solidRect prints a rectangle with perimeters and line infill.
// density: 1 = solid, 0 = no infill, otherwise sparse infill fraction.
func (g *GW) solidRect(x0, y0, x1, y1 float64, perims int, density float64, horiz bool, spP, spF float64) {
	lw := g.lw
	w, h := x1-x0, y1-y0
	if w <= 0 || h <= 0 {
		return
	}
	maxP := int(math.Min(w, h) / (2 * lw))
	if perims > maxP {
		perims = maxP
	}
	if perims <= 0 && math.Min(w, h) < lw*1.5 {
		// too thin for a loop: one line down the middle
		g.typ("WALL-OUTER")
		if w >= h {
			g.travel(x0+lw/2, (y0+y1)/2)
			g.extrudeTo(x1-lw/2, (y0+y1)/2, spP)
		} else {
			g.travel((x0+x1)/2, y0+lw/2)
			g.extrudeTo((x0+x1)/2, y1-lw/2, spP)
		}
		return
	}
	for i := perims - 1; i >= 0; i-- { // inner first, outer last = cleaner outer surface
		in := lw * (float64(i) + 0.5)
		if i == 0 {
			g.typ("WALL-OUTER")
		} else {
			g.typ("WALL-INNER")
		}
		g.rectLoop(x0+in, y0+in, x1-in, y1-in, spP)
	}
	if density <= 0 {
		return
	}
	border := lw * float64(perims)
	ov := 0.0
	if perims > 0 {
		ov = 0.15 * lw
	}
	fx0, fy0 := x0+border+lw/2-ov, y0+border+lw/2-ov
	fx1, fy1 := x1-border-lw/2+ov, y1-border-lw/2+ov
	if fx1 < fx0 {
		fx0, fx1 = (x0+x1)/2, (x0+x1)/2
	}
	if fy1 < fy0 {
		fy0, fy1 = (y0+y1)/2, (y0+y1)/2
	}
	if density >= 0.999 {
		g.typ("SOLID-FILL")
		g.fillLines(fx0, fy0, fx1, fy1, lw, horiz, true, spF)
	} else {
		g.typ("FILL")
		g.fillLines(fx0, fy0, fx1, fy1, lw/density, horiz, false, spF)
	}
}

func circlePts(cx, cy, r float64, start float64) []pt {
	n := int(math.Ceil(2 * math.Pi * r / 0.5))
	if n < 24 {
		n = 24
	}
	if n > 400 {
		n = 400
	}
	pts := make([]pt, n)
	for i := 0; i < n; i++ {
		a := start + 2*math.Pi*float64(i)/float64(n)
		pts[i] = pt{cx + r*math.Cos(a), cy + r*math.Sin(a)}
	}
	return pts
}

// disc prints concentric loops between rIn and rOut (rIn = 0 for a solid disc).
// The outer edge lands exactly on rOut and the inner edge exactly on rIn.
func (g *GW) disc(cx, cy, rOut, rIn, sp float64) {
	lw := g.lw
	if rIn <= 0 {
		n := int(math.Round(rOut / lw))
		if n < 1 {
			n = 1
		}
		for k := 0; k < n; k++ {
			r := rOut - lw/2 - float64(k)*lw
			if r < lw*0.35 {
				g.travel(cx, cy)
				g.extrudeTo(cx+0.1, cy, sp)
				break
			}
			if k == 0 {
				g.typ("WALL-OUTER")
			} else {
				g.typ("SOLID-FILL")
			}
			g.loop(circlePts(cx, cy, r, 0), sp)
		}
		return
	}
	n := int(math.Round((rOut - rIn) / lw))
	if n < 1 {
		n = 1
	}
	r0, r1 := rOut-lw/2, rIn+lw/2
	for k := 0; k < n; k++ {
		r := r0
		if n > 1 {
			r = r0 - (r0-r1)*float64(k)/float64(n-1)
		}
		if k == 0 || k == n-1 {
			g.typ("WALL-OUTER")
		} else {
			g.typ("SOLID-FILL")
		}
		g.loop(circlePts(cx, cy, r, 0), sp)
	}
}

// squareRing prints a square frame of the given outer size and wall width
// using concentric square loops (sharp, measurable corners).
func (g *GW) squareRing(cx, cy, size, wall, sp float64) {
	lw := g.lw
	n := int(math.Round(wall / lw))
	if n < 1 {
		n = 1
	}
	for k := 0; k < n; k++ {
		off := lw * (float64(k) + 0.5)
		hs := size/2 - off
		if hs <= lw/2 {
			break
		}
		if k == 0 {
			g.typ("WALL-OUTER")
		} else {
			g.typ("WALL-INNER")
		}
		g.rectLoop(cx-hs, cy-hs, cx+hs, cy+hs, sp)
	}
}

// ---------------------------------------------------------------- start / end

func meshCmd(fw, mode string) string {
	switch mode {
	case "probe":
		switch fw {
		case "klipper":
			return "BED_MESH_CALIBRATE"
		case "rrf":
			return "G29 S0"
		case "prusa-mk3":
			return "G80"
		default:
			return "G29"
		}
	case "load":
		switch fw {
		case "klipper":
			return "BED_MESH_PROFILE LOAD=default"
		case "rrf":
			return "G29 S1"
		case "marlin", "bambu":
			return "M420 S1"
		}
	}
	return ""
}

func defaultStart(fw string) string {
	switch fw {
	case "rrf":
		return "T0\nM140 S{bed}\nM104 S{hotend}\nG28\n{mesh}\nM190 S{bed}\nM109 S{hotend}"
	case "prusa-mk3":
		return "M140 S{bed}\nM104 S{hotend}\nG28 W\n{mesh}\nM190 S{bed}\nM109 S{hotend}"
	default:
		return "M140 S{bed}\nM104 S{hotend}\nG28\n{mesh}\nM190 S{bed}\nM109 S{hotend}"
	}
}

const defaultEnd = "M104 S0\nM140 S0\nM106 S0\nM84"

func (g *GW) fillTemplate(t string) string {
	r := strings.NewReplacer(
		"{hotend}", num(g.s.Hotend, 0),
		"{bed}", num(g.s.Bed, 0),
		"{mesh}", meshCmd(g.p.Firmware, g.s.Mesh),
		"{bedX}", num(g.p.BedX, 1),
		"{bedY}", num(g.p.BedY, 1),
		"{nozzle}", num(g.p.Nozzle, 2),
		"{filament}", g.s.FilamentName,
		"{firmware}", g.p.Firmware,
	)
	return r.Replace(t)
}

func (g *GW) start() {
	tpl := g.p.StartGcode
	if strings.TrimSpace(tpl) == "" {
		tpl = defaultStart(g.p.Firmware)
	}
	g.w("; --- start sequence ---")
	g.raw(g.fillTemplate(tpl))
	g.w("G90")
	g.w("M83 ; relative extrusion")
	g.w("G92 E0")
	g.w("M106 S0")
	g.fanPct = 0
	if g.s.Accel > 0 {
		g.w("%s", fwAccel(g.p.Firmware, g.s.Accel))
	}
	g.timeS += 240 // rough heat-up + homing allowance
}

func (g *GW) primeLine() {
	flh := g.s.FirstLayerH
	g.w("; --- prime line ---")
	g.typ("SKIRT")
	g.w("G0 Z5 F600")
	g.z, g.lh = 5, flh
	oldLW := g.lw
	g.lw = g.p.Nozzle * 1.5
	sp := g.s.FirstSpeed
	if g.p.BedShape == "circle" {
		R := g.p.BedX/2 - 5
		a0 := 200.0 * math.Pi / 180
		g.travel(R*math.Cos(a0), R*math.Sin(a0))
		g.z = flh
		g.w("G0 Z%s F600", num(flh, 3))
		for a := 200.0; a <= 260; a += 3 {
			r := a * math.Pi / 180
			g.extrudeTo(R*math.Cos(r), R*math.Sin(r), sp)
		}
		R -= 1
		for a := 260.0; a >= 200; a -= 3 {
			r := a * math.Pi / 180
			g.extrudeTo(R*math.Cos(r), R*math.Sin(r), sp)
		}
	} else {
		x := -g.p.BedX/2 + 3
		y0 := -g.p.BedY/2 + 10
		L := math.Min(100, g.p.BedY-25)
		if L < 10 {
			L = math.Max(5, g.p.BedY-15)
		}
		g.travel(x, y0)
		g.z = flh
		g.w("G0 Z%s F600", num(flh, 3))
		g.extrudeTo(x, y0+L, sp)
		g.extrudeTo(x+0.8, y0+L, sp)
		g.extrudeTo(x+0.8, y0+2, sp)
	}
	g.lw = oldLW
	g.retract()
}

func (g *GW) end() {
	g.w("; --- end sequence ---")
	g.curType = ""
	g.retract()
	g.w("G91")
	g.w("G0 Z10 F600")
	g.w("G90")
	if !g.p.OriginCenter && g.p.BedShape != "circle" {
		g.w("G0 X5 Y%s F3000 ; present the print", num(g.p.BedY-5, 1))
	}
	tpl := g.p.EndGcode
	if strings.TrimSpace(tpl) == "" {
		tpl = defaultEnd
	}
	g.raw(g.fillTemplate(tpl))
}

// fits reports whether a centred footprint fits the bed with a safe margin.
func (g *GW) fits(w, h float64) bool {
	margin := 8.0
	if g.p.BedShape == "circle" {
		return math.Hypot(w/2, h/2) <= g.p.BedX/2-margin
	}
	return w <= g.p.BedX-2*margin && h <= g.p.BedY-2*margin
}

// usable returns the largest centred square side that fits.
func (g *GW) usable() float64 {
	if g.p.BedShape == "circle" {
		return (g.p.BedX/2 - 8) * math.Sqrt2
	}
	return math.Min(g.p.BedX, g.p.BedY) - 16
}

func (g *GW) needFit(w, h float64) error {
	if !g.fits(w, h) {
		return fmt.Errorf("this test needs about %s × %s mm of bed space but your bed is %s × %s mm — make it smaller with the size options", num(w+16, 0), num(h+16, 0), num(g.p.BedX, 0), num(g.p.BedY, 0))
	}
	return nil
}

func (g *GW) needHeight(h float64) error {
	if g.p.BedZ > 0 && h > g.p.BedZ-2 {
		return fmt.Errorf("this test is %s mm tall but your printer only has %s mm of Z height — use fewer bands or a smaller step", num(h, 1), num(g.p.BedZ, 0))
	}
	return nil
}

// header is prepended once the body is complete so it can carry estimates.
func (g *GW) header() string {
	var h strings.Builder
	line := func(f string, a ...any) { fmt.Fprintf(&h, f+"\n", a...) }
	line("; ==========================================================")
	line("; Tunie calibration G-code — %s", g.title)
	line("; Made with Tunie v%s by %s (nrmluvdev maker suite)", appVersion, appAuthor)
	line("; Printer : %s  [%s]  bed %s x %s x %s mm  nozzle %s mm", g.p.Name, g.p.Firmware, num(g.p.BedX, 1), num(g.p.BedY, 1), num(g.p.BedZ, 1), num(g.p.Nozzle, 2))
	if g.s.FilamentName != "" {
		line("; Filament: %s", g.s.FilamentName)
	}
	line("; Settings: hotend %s°C bed %s°C layer %s first %s width %s speed %s fan %s%% retract %s@%s flow %s%%",
		num(g.s.Hotend, 0), num(g.s.Bed, 0), num(g.s.LayerH, 3), num(g.s.FirstLayerH, 3), num(g.s.LineW, 3),
		num(g.s.Speed, 0), num(g.s.Fan, 0), num(g.s.RetractLen, 2), num(g.s.RetractSpeed, 0), num(g.s.Flow, 1))
	if len(g.params) > 0 {
		line("; Test    : %s", strings.Join(g.params, "  "))
	}
	line("; Estimate: %s, %s m filament (%s g)", fmtDur(g.timeS), num(g.eTotal/1000, 2), num(g.grams(), 1))
	line("; Generated %s", time.Now().Format("2006-01-02 15:04"))
	line("; ==========================================================")
	return h.String()
}

func (g *GW) grams() float64 {
	vol := g.eTotal * g.filArea / 1000 // cm³
	return vol * g.s.Density
}

func fmtDur(s float64) string {
	m := int(s/60 + 0.5)
	if m < 60 {
		return fmt.Sprintf("~%d min", m)
	}
	return fmt.Sprintf("~%dh %02dm", m/60, m%60)
}

// ---------------------------------------------------------------- firmware helpers

func fwPA(fw string, k float64) string {
	switch fw {
	case "klipper":
		return "SET_PRESSURE_ADVANCE ADVANCE=" + num(k, 4)
	case "rrf":
		return "M572 D0 S" + num(k, 4)
	case "prusa-buddy":
		return "M572 S" + num(k, 4)
	default:
		return "M900 K" + num(k, 4)
	}
}

func fwAccel(fw string, a float64) string {
	ai := int(math.Round(a))
	switch fw {
	case "klipper":
		return fmt.Sprintf("SET_VELOCITY_LIMIT ACCEL=%d", ai)
	case "prusa-mk3", "bambu":
		return fmt.Sprintf("M204 S%d", ai)
	default:
		return fmt.Sprintf("M204 P%d T%d", ai, ai)
	}
}

func fwShaperOff(fw string) string {
	switch fw {
	case "klipper":
		return "SET_INPUT_SHAPER SHAPER_FREQ_X=0 SHAPER_FREQ_Y=0"
	case "rrf":
		return `M593 P"none"`
	case "marlin", "prusa-buddy":
		return "M593 F0"
	}
	return ""
}
