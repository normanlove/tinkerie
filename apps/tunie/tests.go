// Tunie — tests.go: every calibration test's G-code generator.
// No slicer needed: each test is built from simple shapes directly.
// Part of the nrmluvdev maker suite. Created by nrmluvdev.
package tunie

import (
	"fmt"
	"math"
	"regexp"
	"strings"
)

// GenRequest is what the UI sends to generate a test.
type GenRequest struct {
	Test     string             `json:"test"`
	Printer  Printer            `json:"printer"`
	Settings Settings           `json:"settings"`
	Params   map[string]float64 `json:"params"`
	Str      map[string]string  `json:"str"`
}

// GenResult is the generated file plus estimates and notes for the UI.
type GenResult struct {
	Gcode      string         `json:"gcode"`
	FileName   string         `json:"fileName"`
	Lines      int            `json:"lines"`
	TimeSec    float64        `json:"timeSec"`
	FilamentMM float64        `json:"filamentMM"`
	Grams      float64        `json:"grams"`
	Warnings   []string       `json:"warnings"`
	Height     float64        `json:"height"`
	Info       map[string]any `json:"info"`
}

type genFn func(g *GW, r *GenRequest) error

type generator struct {
	title string
	fn    genFn
}

var generators = map[string]generator{
	"pid":            {"PID tuning", genPID},
	"esteps":         {"E-steps / rotation distance", genESteps},
	"flow-wall":      {"Flow rate — single-wall cube", genFlowWall},
	"flow-patches":   {"Flow rate — step patches", genFlowPatches},
	"temp-tower":     {"Temperature tower", genTempTower},
	"retract-tower":  {"Retraction tower", genRetractTower},
	"pa-lines":       {"Pressure / linear advance — line pattern", genPALines},
	"pa-tower":       {"Pressure / linear advance — tower", genPATower},
	"maxflow":        {"Max volumetric flow tower", genMaxFlow},
	"ringing-tower":  {"Input shaper / ringing tower", genRinging},
	"first-layer":    {"First-layer test patches", genFirstLayer},
	"cube":           {"XYZ calibration cube", genCube},
	"accuracy-frame": {"Dimensional accuracy & skew frame", genAccuracyFrame},
	"overhang":       {"Overhang test", genOverhang},
	"bridge":         {"Bridging test", genBridge},
	"tolerance":      {"Tolerance / fit test (peg in hole)", genTolerance},
	"stringing":      {"Stringing test", genStringing},
}

func (r *GenRequest) P(name string, def float64) float64 {
	if v, ok := r.Params[name]; ok && !math.IsNaN(v) && !math.IsInf(v, 0) {
		return v
	}
	return def
}

func (r *GenRequest) S(name, def string) string {
	if v, ok := r.Str[name]; ok && v != "" {
		return v
	}
	return def
}

func (g *GW) param(name string, v any) {
	g.params = append(g.params, fmt.Sprintf("%s=%v", name, v))
}

func sanitize(p *Printer, s *Settings) {
	def := func(v *float64, d float64) {
		if *v <= 0 || math.IsNaN(*v) || math.IsInf(*v, 0) {
			*v = d
		}
	}
	if p.Name == "" {
		p.Name = "My printer"
	}
	if p.Firmware == "" {
		p.Firmware = "marlin"
	}
	if p.BedShape != "circle" {
		p.BedShape = "rect"
	}
	def(&p.Nozzle, 0.4)
	def(&p.FilamentDia, 1.75)
	def(&p.BedX, 220)
	def(&p.BedY, p.BedX)
	if p.BedShape == "circle" {
		p.BedY = p.BedX
	}
	def(&p.BedZ, 250)
	def(&p.MaxSpeed, 150)
	def(&p.MaxAccel, 1000)
	def(&p.MaxHotend, 260)
	def(&p.MaxBed, 100)
	def(&s.LayerH, math.Round(p.Nozzle*50)/100)
	def(&s.FirstLayerH, math.Round(p.Nozzle*60)/100)
	def(&s.LineW, math.Round(p.Nozzle*1125)/1000)
	def(&s.Speed, 40)
	def(&s.FirstSpeed, 20)
	def(&s.TravelSpeed, 120)
	def(&s.Hotend, 210)
	def(&s.Flow, 100)
	def(&s.Density, 1.24)
	def(&s.RetractSpeed, 35)
	s.Bed = math.Max(0, s.Bed)
	s.Fan = clamp(s.Fan, 0, 100)
	s.RetractLen = clamp(s.RetractLen, 0, 15)
	s.ZHop = clamp(s.ZHop, 0, 5)
	s.Speed = math.Min(s.Speed, p.MaxSpeed)
	s.FirstSpeed = math.Min(s.FirstSpeed, p.MaxSpeed)
	s.TravelSpeed = math.Min(s.TravelSpeed, p.MaxSpeed)
	if s.Mesh == "" {
		s.Mesh = "none"
	}
}

func Generate(r GenRequest) (GenResult, error) {
	sanitize(&r.Printer, &r.Settings)
	gen, ok := generators[r.Test]
	if !ok {
		return GenResult{}, fmt.Errorf("unknown test %q", r.Test)
	}
	g := newGW(r.Printer, r.Settings)
	g.title = gen.title
	if g.s.Hotend > g.p.MaxHotend {
		return GenResult{}, fmt.Errorf("hotend temperature %s°C is above this printer's maximum of %s°C", num(g.s.Hotend, 0), num(g.p.MaxHotend, 0))
	}
	if g.s.Bed > g.p.MaxBed {
		return GenResult{}, fmt.Errorf("bed temperature %s°C is above this printer's maximum of %s°C", num(g.s.Bed, 0), num(g.p.MaxBed, 0))
	}
	if g.s.LayerH > g.p.Nozzle*0.8 {
		g.warn("Layer height %s mm is more than 80%% of the nozzle size — results may be rough.", num(g.s.LayerH, 2))
	}
	if g.p.Firmware == "bambu" {
		g.warn("Bambu Lab printers normally print .3mf files sent from Bambu Studio. This plain G-code uses a generic start sequence — see the Bambu notes before printing.")
	}
	if err := gen.fn(g, &r); err != nil {
		return GenResult{}, err
	}
	g.checkBounds()
	body := g.sb.String()
	res := GenResult{
		Gcode:      g.header() + body,
		FileName:   fileName(r.Test, r.Printer.Name),
		Lines:      g.lines,
		TimeSec:    g.timeS,
		FilamentMM: g.eTotal,
		Grams:      g.grams(),
		Warnings:   g.warns,
		Height:     g.maxZ,
		Info:       g.info,
	}
	return res, nil
}

var badName = regexp.MustCompile(`[^A-Za-z0-9_\-]+`)

func fileName(test, printer string) string {
	p := strings.Trim(badName.ReplaceAllString(printer, "_"), "_")
	if p == "" {
		p = "printer"
	}
	return fmt.Sprintf("Tunie_%s_%s.gcode", test, p)
}

func (g *GW) checkBounds() {
	if math.IsInf(g.minX, 1) {
		return
	}
	const tol = 0.5
	bad := false
	if g.p.BedShape == "circle" {
		if g.maxR > g.p.BedX/2+tol {
			bad = true
		}
	} else {
		lo := 0.0
		hiX, hiY := g.p.BedX, g.p.BedY
		if g.p.OriginCenter {
			lo = -g.p.BedX / 2
			hiX, hiY = g.p.BedX/2, g.p.BedY/2
			if g.minY < -g.p.BedY/2-tol {
				bad = true
			}
		} else if g.minY < -tol {
			bad = true
		}
		if g.minX < lo-tol || g.maxX > hiX+tol || g.maxY > hiY+tol {
			bad = true
		}
	}
	if bad {
		g.warn("Some moves reach the edge of the bed (X %s…%s, Y %s…%s). Check your bed size before printing.", num(g.minX, 1), num(g.maxX, 1), num(g.minY, 1), num(g.maxY, 1))
	}
}

// standard layer loop helper
func (g *GW) eachLayer(total float64, fn func(i int, z float64, sp float64)) {
	zs := layerZs(total, g.s.FirstLayerH, g.s.LayerH)
	for i, z := range zs {
		h := g.s.LayerH
		if i == 0 {
			h = g.s.FirstLayerH
		}
		g.newLayer(z, h)
		g.layerFan(i)
		sp := g.s.Speed
		if i == 0 {
			sp = g.s.FirstSpeed
		}
		fn(i, z, sp)
	}
}

func (g *GW) begin() {
	g.start()
	g.primeLine()
}

// ================================================================ command-only tests

func genPID(g *GW, r *GenRequest) error {
	target := r.P("hotendTarget", g.s.Hotend)
	bed := r.P("bedTarget", g.s.Bed)
	cycles := int(r.P("cycles", 8))
	which := r.S("which", "both")
	g.param("hotend", target)
	g.param("bed", bed)
	g.w("; PID autotune — the printer heats up and cools down several times.")
	g.w("; Watch the printer while this runs. Results appear on the printer's console/screen.")
	g.w("M106 S%d ; part fan at print speed so the tune matches real printing", int(math.Round(255*g.s.Fan/100)))
	fw := g.p.Firmware
	if which != "bed" {
		switch fw {
		case "klipper":
			g.w("PID_CALIBRATE HEATER=extruder TARGET=%s", num(target, 0))
		case "rrf":
			g.w("M303 T0 S%s", num(target, 0))
		default:
			g.w("M303 E0 S%s C%d U1", num(target, 0), cycles)
		}
	}
	if which != "hotend" && bed > 0 {
		g.w("M106 S0")
		switch fw {
		case "klipper":
			g.w("PID_CALIBRATE HEATER=heater_bed TARGET=%s", num(bed, 0))
		case "rrf":
			g.w("M303 H0 S%s", num(bed, 0))
		default:
			g.w("M303 E-1 S%s C%d U1", num(bed, 0), cycles)
		}
	}
	g.w("M106 S0")
	switch fw {
	case "klipper":
		g.w("SAVE_CONFIG ; saves the new PID values and restarts Klipper")
	default:
		g.w("M500 ; save to EEPROM / config-override")
	}
	g.timeS = 600
	return nil
}

func genESteps(g *GW, r *GenRequest) error {
	length := r.P("length", 100)
	feed := r.P("feed", 100) // mm/min
	g.param("length", length)
	g.w("; E-steps test: mark the filament 120 mm above the extruder inlet first!")
	g.w("M104 S%s", num(g.s.Hotend, 0))
	g.w("G28 ; home so we can lift the nozzle")
	g.w("G90")
	g.w("G0 Z%s F600 ; lift nozzle clear of the bed", num(math.Min(50, g.p.BedZ*0.4), 0))
	g.w("M109 S%s ; wait for the hotend", num(g.s.Hotend, 0))
	g.w("M83 ; relative extrusion")
	g.w("G92 E0")
	g.w("G1 E%s F%s ; extrude slowly", num(length, 1), num(feed, 0))
	g.w("M400 ; wait until finished")
	g.w("M104 S0")
	g.eTotal = length
	g.timeS = 180 + length/(feed/60)
	return nil
}

// ================================================================ flow

func genFlowWall(g *GW, r *GenRequest) error {
	size := r.P("size", 20)
	height := r.P("height", 10)
	g.param("size", size)
	g.param("lineWidth", g.lw)
	if err := g.needFit(size, size); err != nil {
		return err
	}
	if err := g.needHeight(height); err != nil {
		return err
	}
	g.info["expectedWall"] = g.lw
	g.begin()
	h := size / 2
	g.eachLayer(height, func(i int, z, sp float64) {
		if i < 2 {
			g.solidRect(-h, -h, h, h, 1, 1, i%2 == 0, sp, sp)
			return
		}
		g.typ("WALL-OUTER")
		in := g.lw / 2
		g.rectLoop(-h+in, -h+in, h-in, h-in, sp)
	})
	g.end()
	return nil
}

func genFlowPatches(g *GW, r *GenRequest) error {
	from := r.P("from", -10)
	to := r.P("to", 10)
	step := math.Abs(r.P("step", 5))
	if step == 0 {
		step = 5
	}
	if to < from {
		from, to = to, from
	}
	n := int(math.Floor((to-from)/step+1e-6)) + 1
	if n > 12 {
		n = 12
		g.warn("Limited to 12 patches.")
	}
	ps := r.P("patch", 18)
	gap := 6.0
	cols := int(math.Ceil(math.Sqrt(float64(n))))
	rows := int(math.Ceil(float64(n) / float64(cols)))
	cell := ps + gap
	labelH := 7 * 0.6
	cellY := ps + labelH + gap
	W := float64(cols)*cell - gap
	H := float64(rows)*cellY - gap
	if err := g.needFit(W, H); err != nil {
		return err
	}
	g.param("flowSteps", fmt.Sprintf("%s..%s step %s %%", num(from, 1), num(to, 1), num(step, 1)))
	offs := []float64{}
	g.begin()
	type patch struct{ x0, y0, f float64 }
	var patches []patch
	for k := 0; k < n; k++ {
		f := from + float64(k)*step
		offs = append(offs, f)
		c, rw := k%cols, k/cols
		x0 := -W/2 + float64(c)*cell
		y0 := H/2 - float64(rw+1)*cellY + gap + labelH
		patches = append(patches, patch{x0, y0, f})
	}
	g.info["offsets"] = offs
	g.eachLayer(g.s.FirstLayerH+3*g.s.LayerH, func(i int, z, sp float64) {
		for _, p := range patches {
			g.flowMul = 1 + p.f/100
			g.w("; patch flow %s%%", num(p.f, 1))
			g.solidRect(p.x0, p.y0, p.x0+ps, p.y0+ps, 1, 1, i%2 == 0, sp, sp)
			g.flowMul = 1
			if i == 0 {
				label := num(p.f, 1)
				if p.f > 0 {
					label = "+" + label
				}
				tw := textWidth(label, 0.6)
				g.flatText(label, p.x0+ps/2-tw/2, p.y0-labelH-1, 0.6, sp)
			}
		}
	})
	g.end()
	return nil
}

// ================================================================ towers

func bandValues(start, end, step float64, maxN int) ([]float64, error) {
	step = math.Abs(step)
	if step == 0 {
		return nil, fmt.Errorf("step must be more than zero")
	}
	n := int(math.Floor(math.Abs(end-start)/step+1e-6)) + 1
	if n > maxN {
		return nil, fmt.Errorf("that makes %d bands — the most is %d; use a bigger step", n, maxN)
	}
	dir := 1.0
	if end < start {
		dir = -1
	}
	v := make([]float64, n)
	for i := range v {
		v[i] = math.Round((start+dir*step*float64(i))*10000) / 10000
	}
	return v, nil
}

func genTempTower(g *GW, r *GenRequest) error {
	temps, err := bandValues(r.P("start", 220), r.P("end", 190), r.P("step", 5), 15)
	if err != nil {
		return err
	}
	for _, t := range temps {
		if t > g.p.MaxHotend {
			return fmt.Errorf("%s°C is above this printer's max hotend temperature (%s°C)", num(t, 0), num(g.p.MaxHotend, 0))
		}
	}
	bandH := math.Max(8, r.P("bandH", 10))
	baseH := g.s.FirstLayerH + g.s.LayerH
	H := baseH + float64(len(temps))*bandH
	if err := g.needFit(40, 16); err != nil {
		return err
	}
	if err := g.needHeight(H); err != nil {
		return err
	}
	g.param("temps", fmt.Sprint(temps))
	g.param("bandH", bandH)
	g.info["bands"] = temps
	g.info["bandH"] = bandH
	g.info["baseH"] = baseH
	g.s.Hotend = temps[0]
	g.begin()
	ph := g.pixelHeight(0.8)
	band := -1
	lh := g.s.LayerH
	g.eachLayer(H, func(i int, z, sp float64) {
		if z <= baseH+1e-6 {
			g.solidRect(-18, -7, 18, 7, 2, 1, i%2 == 0, sp, sp)
			return
		}
		b := int((z - baseH - 1e-6) / bandH)
		if b >= len(temps) {
			b = len(temps) - 1
		}
		if b != band {
			band = b
			if b > 0 {
				g.w("M104 S%s ; ---- band %d: %s°C ----", num(temps[b], 0), b+1, num(temps[b], 0))
			} else {
				g.w("; ---- band 1: %s°C ----", num(temps[0], 0))
			}
		}
		bBase := baseH + float64(b)*bandH
		zin := z - bBase
		if zin > bandH-3*lh+1e-6 {
			// bridge across the gap at the top of each band
			g.typ("BRIDGE")
			g.fillLines(-16+g.lw/2, -5+g.lw/2, 16-g.lw/2, 5-g.lw/2, g.lw, true, true, sp*0.7)
			return
		}
		d := 0.3
		if zin > bandH-3*lh-0.8+1e-6 || zin < 0.6 {
			d = 1 // solid just under and over each bridge
		}
		g.solidRect(-16, -5, -4, 5, 2, d, i%2 == 0, sp, sp*1.3)
		g.emboss('F', -16, -4, -5, bBase+1.0, ph, 0.7, num(temps[b], 0), sp*0.6)
		g.solidRect(4, -5, 16, 5, 2, d, i%2 == 0, sp, sp*1.3)
	})
	g.end()
	return nil
}

func genRetractTower(g *GW, r *GenRequest) error {
	mode := r.S("mode", "distance")
	var vals []float64
	var err error
	if mode == "speed" {
		vals, err = bandValues(r.P("start", 20), r.P("end", 60), r.P("step", 10), 12)
	} else {
		def, end, st := 0.2, 2.0, 0.2
		if g.p.Drive == "bowden" {
			def, end, st = 2, 7, 0.5
		}
		vals, err = bandValues(r.P("start", def), r.P("end", end), r.P("step", st), 12)
	}
	if err != nil {
		return err
	}
	bandH := math.Max(5.5, r.P("bandH", 6))
	baseH := g.s.FirstLayerH + g.s.LayerH
	H := baseH + float64(len(vals))*bandH
	if err := g.needFit(42, 16); err != nil {
		return err
	}
	if err := g.needHeight(H); err != nil {
		return err
	}
	g.param("mode", mode)
	g.param("values", fmt.Sprint(vals))
	g.info["bands"] = vals
	g.info["bandH"] = bandH
	g.info["baseH"] = baseH
	g.info["mode"] = mode
	fixedLen, fixedSpeed := g.s.RetractLen, g.s.RetractSpeed
	if mode == "speed" {
		if fixedLen <= 0 {
			fixedLen = 1
		}
		g.s.RetractLen = fixedLen
		g.s.RetractSpeed = vals[0]
	} else {
		g.s.RetractLen = vals[0]
	}
	g.begin()
	ph := g.pixelHeight(0.6)
	band := -1
	g.eachLayer(H, func(i int, z, sp float64) {
		if z <= baseH+1e-6 {
			g.solidRect(-19, -7, 19, 7, 2, 1, i%2 == 0, sp, sp)
			return
		}
		b := int((z - baseH - 1e-6) / bandH)
		if b >= len(vals) {
			b = len(vals) - 1
		}
		if b != band {
			band = b
			if mode == "speed" {
				g.s.RetractSpeed = vals[b]
				g.s.RetractLen = fixedLen
			} else {
				g.s.RetractLen = vals[b]
				g.s.RetractSpeed = fixedSpeed
			}
			g.w("; ---- band %d: retraction %s mm @ %s mm/s ----", b+1, num(g.s.RetractLen, 2), num(g.s.RetractSpeed, 0))
		}
		bBase := baseH + float64(b)*bandH
		label := num(vals[b], 1)
		if mode == "speed" {
			label = num(vals[b], 0)
		}
		d := 0.3
		if z > H-0.8 {
			d = 1
		}
		g.solidRect(-17, -5, -7, 5, 2, d, i%2 == 0, sp, sp*1.3)
		g.emboss('F', -17, -7, -5, bBase+0.6, ph, 0.7, label, sp*0.6)
		g.solidRect(7, -5, 17, 5, 2, d, i%2 == 0, sp, sp*1.3)
	})
	g.end()
	return nil
}

func decimalsFor(step float64) int {
	if step <= 0 {
		return 3
	}
	d := int(math.Ceil(-math.Log10(step) - 1e-9))
	if d < 0 {
		d = 0
	}
	if d > 4 {
		d = 4
	}
	return d
}

func genPALines(g *GW, r *GenRequest) error {
	start := r.P("start", 0)
	end := r.P("end", 0.1)
	step := r.P("step", 0.005)
	vals, err := bandValues(start, end, step, 40)
	if err != nil {
		return err
	}
	slow := r.P("slow", 20)
	fast := g.speed(r.P("fast", 100))
	pitch := 5.0
	L := math.Min(80, g.usable()-22)
	if L < 30 {
		return fmt.Errorf("the bed is too small for the line pattern — use the tower method instead")
	}
	dec := decimalsFor(step)
	maxLabel := 0.0
	for _, v := range vals {
		maxLabel = math.Max(maxLabel, textWidth(num(v, dec), 0.55))
	}
	W := L + 3 + maxLabel
	H := float64(len(vals)) * pitch
	if err := g.needFit(W, H); err != nil {
		return err
	}
	g.param("K", fmt.Sprintf("%s..%s step %s", num(start, 4), num(end, 4), num(step, 4)))
	g.param("slow/fast", fmt.Sprintf("%s/%s mm/s", num(slow, 0), num(fast, 0)))
	g.info["values"] = vals
	g.begin()
	g.newLayer(g.s.FirstLayerH, g.s.FirstLayerH)
	x0 := -W / 2
	y0 := -H/2 + pitch/2
	for k, v := range vals {
		y := y0 + float64(k)*pitch
		g.w("%s ; line %d", fwPA(g.p.Firmware, v), k+1)
		g.typ("WALL-OUTER")
		g.travel(x0, y)
		g.extrudeTo(x0+L/4, y, slow)
		g.extrudeTo(x0+3*L/4, y, fast)
		g.extrudeTo(x0+L, y, slow)
	}
	g.w("%s ; back to the start value", fwPA(g.p.Firmware, start))
	for k, v := range vals {
		y := y0 + float64(k)*pitch
		g.flatText(num(v, dec), x0+L+3, y-7*0.55/2, 0.55, g.s.FirstSpeed)
	}
	// two reference marks show where the fast section starts/ends
	g.typ("SKIRT")
	for _, mx := range []float64{x0 + L/4, x0 + 3*L/4} {
		g.travel(mx, y0-pitch*0.8)
		g.extrudeTo(mx, y0-pitch*0.4, g.s.FirstSpeed)
	}
	g.end()
	return nil
}

func genPATower(g *GW, r *GenRequest) error {
	start := r.P("start", 0)
	factor := r.P("factor", 0.005)
	H := r.P("height", 30)
	size := r.P("size", 30)
	sp := g.speed(r.P("speedTower", 100))
	accel := r.P("accelTower", 500)
	if err := g.needFit(size, size); err != nil {
		return err
	}
	if err := g.needHeight(H); err != nil {
		return err
	}
	g.param("start", start)
	g.param("factor/mm", factor)
	g.info["start"] = start
	g.info["factor"] = factor
	g.begin()
	g.w("%s", fwAccel(g.p.Firmware, accel))
	if g.p.Firmware == "klipper" {
		g.w("SET_VELOCITY_LIMIT SQUARE_CORNER_VELOCITY=1")
	}
	h := size / 2
	lastK := -1.0
	g.eachLayer(H, func(i int, z, lsp float64) {
		if i < 2 {
			g.solidRect(-h, -h, h, h, 2, 1, i%2 == 0, lsp, lsp)
			return
		}
		k := math.Round((start+factor*z)*10000) / 10000
		if k != lastK {
			g.w("%s", fwPA(g.p.Firmware, k))
			lastK = k
		}
		g.solidRect(-h, -h, h, h, 2, 0, true, sp, sp)
	})
	if g.p.Firmware == "klipper" {
		g.w("SET_VELOCITY_LIMIT SQUARE_CORNER_VELOCITY=5")
	}
	g.end()
	return nil
}

func genMaxFlow(g *GW, r *GenRequest) error {
	start := r.P("start", 5)
	end := r.P("end", 25)
	perMM := math.Max(0.05, r.P("perMM", 0.5))
	lw := r.P("lineWidth", math.Round(g.p.Nozzle*1.2*100)/100)
	lh := r.P("layerHeight", math.Round(g.p.Nozzle*0.6*100)/100)
	radius := r.P("radius", 20)
	if end <= start {
		return fmt.Errorf("end flow must be higher than start flow")
	}
	baseH := g.s.FirstLayerH + g.s.LayerH
	H := baseH + (end-start)/perMM
	if err := g.needFit(2*radius+6, 2*radius+6); err != nil {
		return err
	}
	if err := g.needHeight(H); err != nil {
		return err
	}
	maxTestable := g.p.MaxSpeed * lw * lh
	if maxTestable < end {
		g.warn("At your max speed (%s mm/s) this line can only reach %s mm³/s — above that height the flow stops increasing. Use a wider line or taller layer to test higher.", num(g.p.MaxSpeed, 0), num(maxTestable, 1))
	}
	g.param("flow", fmt.Sprintf("%s..%s mm³/s, +%s per mm", num(start, 1), num(end, 1), num(perMM, 2)))
	g.info["start"] = start
	g.info["perMM"] = perMM
	g.info["baseH"] = baseH
	g.begin()
	baseLW := g.lw
	zs := []float64{g.s.FirstLayerH, baseH}
	for z := baseH + lh; z <= H+1e-6; z += lh {
		zs = append(zs, math.Round(z*10000)/10000)
	}
	prev := 0.0
	lastTick := -1
	for i, z := range zs {
		h := z - prev
		prev = z
		g.newLayer(z, h)
		g.layerFan(i)
		if i < 2 {
			g.lw = baseLW
			sp := g.s.FirstSpeed
			if i == 1 {
				sp = g.s.Speed
			}
			g.disc(0, 0, radius+lw/2, 0, sp)
			continue
		}
		g.lw = lw
		flow := start + (z-baseH)*perMM
		sp := g.speed(flow / (lw * h))
		g.w("; flow %s mm³/s -> %s mm/s", num(flow, 2), num(sp, 1))
		g.typ("WALL-OUTER")
		g.loop(circlePts(0, 0, radius, 0), sp)
		// a tick mark every 5 mm of height makes measuring easier
		tick := int((z - baseH) / 5)
		if tick != lastTick && z-baseH-float64(tick)*5 < 2*lh {
			g.typ("LABEL")
			g.extrudeTo(radius+2.5, 0, math.Min(sp, 30))
			g.extrudeTo(radius, 0, math.Min(sp, 30))
			if z-baseH-float64(tick)*5 >= lh {
				lastTick = tick
			}
		}
	}
	g.lw = baseLW
	g.end()
	return nil
}

func genRinging(g *GW, r *GenRequest) error {
	size := r.P("size", 40)
	a0 := r.P("accelStart", 1500)
	aStep := r.P("accelStep", 500)
	bands := int(math.Max(1, r.P("bands", 8)))
	bandH := math.Max(3, r.P("bandH", 5))
	sp := g.speed(r.P("speedTower", 100))
	perims := int(math.Max(1, r.P("perims", 2)))
	baseH := g.s.FirstLayerH + g.s.LayerH
	H := baseH + float64(bands)*bandH
	if err := g.needFit(size+4, size+4); err != nil {
		return err
	}
	if err := g.needHeight(H); err != nil {
		return err
	}
	accels := make([]float64, bands)
	for i := range accels {
		accels[i] = a0 + aStep*float64(i)
		if g.p.MaxAccel > 0 && accels[i] > g.p.MaxAccel {
			g.warn("Band %d asks for %s mm/s² but your profile max accel is %s — the printer may cap it.", i+1, num(accels[i], 0), num(g.p.MaxAccel, 0))
			break
		}
	}
	g.param("accel", fmt.Sprint(accels))
	g.param("speed", sp)
	g.info["bands"] = accels
	g.info["bandH"] = bandH
	g.info["baseH"] = baseH
	g.info["speed"] = sp
	g.begin()
	if g.p.HasInputShaper {
		if c := fwShaperOff(g.p.Firmware); c != "" {
			g.w("%s ; input shaping off for the test", c)
		}
	}
	h := size / 2
	band := -1
	ph := g.pixelHeight(0.8)
	g.eachLayer(H, func(i int, z, lsp float64) {
		if z <= baseH+1e-6 {
			g.solidRect(-h, -h, h, h, 2, 1, i%2 == 0, lsp, lsp)
			return
		}
		b := int((z - baseH - 1e-6) / bandH)
		if b >= bands {
			b = bands - 1
		}
		if b != band {
			band = b
			g.w("%s ; ---- band %d: accel %s ----", fwAccel(g.p.Firmware, accels[b]), b+1, num(accels[b], 0))
		}
		g.solidRect(-h, -h, h, h, perims, 0, true, sp, sp)
		g.emboss('F', -h, h, -h, baseH+0.4, ph, 0.9, "X", lsp)
		g.emboss('R', -h, h, h, baseH+0.4, ph, 0.9, "Y", lsp)
	})
	g.end()
	return nil
}

func genFirstLayer(g *GW, r *GenRequest) error {
	s := r.P("patch", 30)
	layers := int(clamp(r.P("layers", 1), 1, 3))
	var centres []pt
	centres = append(centres, pt{0, 0})
	if g.p.BedShape == "circle" {
		rr := g.p.BedX/2 - 14 - s*0.71
		if rr > s {
			for _, a := range []float64{45, 135, 225, 315} {
				ar := a * math.Pi / 180
				centres = append(centres, pt{rr * math.Cos(ar), rr * math.Sin(ar)})
			}
		}
	} else {
		dx := g.p.BedX/2 - 15 - s/2
		dy := g.p.BedY/2 - 15 - s/2
		if dx < s*0.8 || dy < s*0.8 {
			s = math.Min(s, math.Min(g.p.BedX, g.p.BedY)/3)
			dx = g.p.BedX/2 - 12 - s/2
			dy = g.p.BedY/2 - 12 - s/2
		}
		if dx > s*0.6 && dy > s*0.6 {
			centres = append(centres, pt{-dx, -dy}, pt{dx, -dy}, pt{dx, dy}, pt{-dx, dy})
		} else {
			g.warn("Bed is small, so only the centre patch is printed.")
		}
	}
	if err := g.needFit(s, s); err != nil {
		return err
	}
	g.param("patch", s)
	g.param("layers", layers)
	g.begin()
	total := g.s.FirstLayerH + float64(layers-1)*g.s.LayerH
	g.eachLayer(total, func(i int, z, sp float64) {
		for _, c := range centres {
			g.solidRect(c.x-s/2, c.y-s/2, c.x+s/2, c.y+s/2, 2, 1, i%2 == 0, sp, sp)
		}
	})
	g.end()
	return nil
}

// ================================================================ dimensional

func genCube(g *GW, r *GenRequest) error {
	size := r.P("size", 20)
	infill := clamp(r.P("infill", 20), 0, 100) / 100
	perims := int(clamp(r.P("perims", 3), 1, 6))
	if err := g.needFit(size+2, size+2); err != nil {
		return err
	}
	if err := g.needHeight(size); err != nil {
		return err
	}
	g.param("size", size)
	g.info["size"] = size
	g.begin()
	h := size / 2
	shell := math.Max(0.8, 3*g.s.LayerH)
	ph := g.pixelHeight(size / 25)
	pw := size / 20
	z0 := size/2 - 3.5*ph
	g.eachLayer(size, func(i int, z, sp float64) {
		d := infill
		if z <= shell+1e-6 || z > size-shell+1e-6 {
			d = 1
		}
		g.solidRect(-h, -h, h, h, perims, d, i%2 == 0, sp, sp*1.2)
		g.emboss('F', -h, h, -h, z0, ph, pw, "X", sp*0.6)
		g.emboss('R', -h, h, h, z0, ph, pw, "Y", sp*0.6)
	})
	g.end()
	return nil
}

func genAccuracyFrame(g *GW, r *GenRequest) error {
	step := math.Max(10, r.P("step", 20))
	maxS := r.P("max", 100)
	wall := r.P("wall", 3)
	height := r.P("height", 3)
	limit := g.usable() - 14
	if maxS > limit {
		maxS = math.Floor(limit/step) * step
		g.warn("Largest square reduced to %s mm to fit your bed.", num(maxS, 0))
	}
	if maxS < step {
		return fmt.Errorf("the bed is too small for this frame — use the XYZ cube instead")
	}
	var sizes []float64
	for s := step; s <= maxS+1e-6; s += step {
		sizes = append(sizes, s)
	}
	S := sizes[len(sizes)-1]
	g.param("sizes", fmt.Sprint(sizes))
	g.info["sizes"] = sizes
	g.begin()
	p := 0.6
	th := 7 * p
	g.eachLayer(height, func(i int, z, sp float64) {
		for k := len(sizes) - 1; k >= 0; k-- {
			g.squareRing(0, 0, sizes[k], math.Min(wall, sizes[k]/2-1), sp)
		}
		if i < 2 {
			tw := textWidth("A", p)
			g.flatText("A", -S/2-2-tw, -S/2-2-th, p, g.s.FirstSpeed)
			g.flatText("B", -S/2-2-tw, S/2+2, p, g.s.FirstSpeed)
			g.flatText("C", S/2+2, S/2+2, p, g.s.FirstSpeed)
			g.flatText("D", S/2+2, -S/2-2-th, p, g.s.FirstSpeed)
		}
	})
	g.end()
	return nil
}

// ================================================================ quality tests

func genOverhang(g *GW, r *GenRequest) error {
	angles, err := bandValues(r.P("start", 30), r.P("end", 70), r.P("step", 10), 9)
	if err != nil {
		return err
	}
	reach := r.P("reach", 10)
	maxH := r.P("maxHeight", 15)
	spacing := 12.0
	n := len(angles)
	W := float64(n-1)*spacing + 8 + 6
	if err := g.needFit(W, 30); err != nil {
		return err
	}
	g.param("angles", fmt.Sprint(angles))
	g.info["angles"] = angles
	g.begin()
	plateLayers := 3
	plateTop := g.s.FirstLayerH + float64(plateLayers-1)*g.s.LayerH
	x0 := -float64(n-1) * spacing / 2
	px0, px1 := x0-4-3, x0+float64(n-1)*spacing+4+3
	g.eachLayer(plateTop+maxH, func(i int, z, sp float64) {
		if i < plateLayers {
			g.solidRect(px0, -3, px1, 10, 2, 1, i%2 == 0, sp, sp)
			return
		}
		for k, a := range angles {
			xc := x0 + float64(k)*spacing
			ta := math.Tan(a * math.Pi / 180)
			hmax := math.Min(maxH, reach/ta)
			zf := z - plateTop
			if zf > hmax+1e-6 {
				continue
			}
			off := zf * ta
			g.solidRect(xc-4, -off, xc+4, 2-off, 2, 1, i%2 == 0, sp*0.8, sp*0.8)
			if i < plateLayers+2 {
				lb := num(a, 0)
				g.flatText(lb, xc-textWidth(lb, 0.6)/2, 4, 0.6, g.s.FirstSpeed)
			}
		}
	})
	g.end()
	return nil
}

func genBridge(g *GW, r *GenRequest) error {
	gaps, err := bandValues(r.P("start", 10), r.P("end", 50), r.P("step", 10), 8)
	if err != nil {
		return err
	}
	pillarH := r.P("pillarH", 5)
	bridgeSp := r.P("bridgeSpeed", 20)
	maxGap := 0.0
	for _, gp := range gaps {
		maxGap = math.Max(maxGap, gp)
	}
	pitch := 16.0
	n := len(gaps)
	labelW := textWidth("50", 0.6) + 2
	W := maxGap + 10 + 2*labelW
	H := float64(n)*pitch - 6
	if err := g.needFit(W, H); err != nil {
		return err
	}
	g.param("gaps", fmt.Sprint(gaps))
	g.info["gaps"] = gaps
	g.begin()
	bridgeLayers := 2
	lh := g.s.LayerH
	total := pillarH + float64(bridgeLayers)*lh
	g.eachLayer(total, func(i int, z, sp float64) {
		inBridge := z > pillarH+1e-6
		firstBridge := inBridge && z <= pillarH+lh+1e-6
		if firstBridge {
			g.setFan(100)
		}
		for k, gp := range gaps {
			yc := -H/2 + 5 + float64(k)*pitch
			xl := -gp/2 - 5
			if !inBridge {
				g.solidRect(xl, yc-5, -gp/2, yc+5, 2, 1, i%2 == 0, sp, sp)
				g.solidRect(gp/2, yc-5, gp/2+5, yc+5, 2, 1, i%2 == 0, sp, sp)
				if i < 2 {
					lb := num(gp, 0)
					g.flatText(lb, xl-2-textWidth(lb, 0.6), yc-2.1, 0.6, g.s.FirstSpeed)
				}
			} else {
				g.typ("BRIDGE")
				bs := sp
				if firstBridge {
					bs = bridgeSp
				}
				g.fillLines(xl+g.lw/2, yc-5+g.lw/2, gp/2+5-g.lw/2, yc+5-g.lw/2, g.lw, true, true, bs)
			}
		}
		if firstBridge {
			g.setFan(g.s.Fan)
		}
	})
	g.end()
	return nil
}

func genTolerance(g *GW, r *GenRequest) error {
	clears, err := bandValues(r.P("start", 0), r.P("end", 0.6), r.P("step", 0.1), 10)
	if err != nil {
		return err
	}
	D := r.P("diameter", 10)
	wall := r.P("wall", 3)
	ringH := r.P("ringH", 6)
	pegH := r.P("pegH", 12)
	maxC := clears[len(clears)-1]
	outer := D + maxC + 2*wall
	p := 0.6
	labelH := 7 * p
	cellW := outer + 4
	cellH := outer + labelH + 5
	items := len(clears) + 1
	avail := g.usable()
	cols := int(math.Max(1, math.Floor((avail+4)/cellW)))
	if cols > items {
		cols = items
	}
	rows := int(math.Ceil(float64(items) / float64(cols)))
	W := float64(cols)*cellW - 4
	H := float64(rows)*cellH - 5
	if err := g.needFit(W, H); err != nil {
		return err
	}
	if err := g.needHeight(pegH); err != nil {
		return err
	}
	g.param("diameter", D)
	g.param("clearances", fmt.Sprint(clears))
	g.info["clearances"] = clears
	g.info["diameter"] = D
	type item struct {
		cx, cy float64
		c      float64
		peg    bool
	}
	var its []item
	for k := 0; k < items; k++ {
		c, rw := k%cols, k/cols
		cx := -W/2 + float64(c)*cellW + outer/2
		cy := H/2 - float64(rw)*cellH - outer/2
		if k < len(clears) {
			its = append(its, item{cx, cy, clears[k], false})
		} else {
			its = append(its, item{cx, cy, 0, true})
		}
	}
	g.begin()
	g.eachLayer(pegH, func(i int, z, sp float64) {
		for _, it := range its {
			if it.peg {
				g.disc(it.cx, it.cy, D/2, 0, sp*0.8)
				if i == 0 {
					g.flatText("P", it.cx-textWidth("P", p)/2, it.cy-outer/2-labelH-1, p, g.s.FirstSpeed)
				}
				continue
			}
			if z > ringH+1e-6 {
				continue
			}
			ri := (D + it.c) / 2
			g.disc(it.cx, it.cy, ri+wall, ri, sp*0.8)
			if i < 2 {
				lb := num(it.c, 2)
				g.flatText(lb, it.cx-textWidth(lb, p)/2, it.cy-(ri+wall)-labelH-1.5, p, g.s.FirstSpeed)
			}
		}
	})
	g.end()
	return nil
}

func genStringing(g *GW, r *GenRequest) error {
	dist := r.P("distance", 30)
	H := r.P("height", 25)
	if err := g.needFit(dist+14, dist+14); err != nil {
		return err
	}
	if err := g.needHeight(H); err != nil {
		return err
	}
	g.param("distance", dist)
	g.param("retract", fmt.Sprintf("%s mm @ %s mm/s", num(g.s.RetractLen, 2), num(g.s.RetractSpeed, 0)))
	d := dist / 2
	ps := []pt{{-d, -d}, {d, d}, {-d, d}, {d, -d}}
	g.begin()
	g.eachLayer(H, func(i int, z, sp float64) {
		rad := 3.5 - 2.0*(z/H)
		for _, p := range ps {
			if i == 0 {
				g.disc(p.x, p.y, 6, 0, sp)
			} else {
				g.disc(p.x, p.y, rad, 0, sp)
			}
		}
	})
	g.end()
	return nil
}
