package geom

// Wirie boards: reading the "wirie-case" (.makie.json) files exported by
// Wirie, nrmluvdev's PCB designer, and turning them into a reference board
// model plus a ready-to-print case (base, lid and extras).
//
// Coordinates follow the Wirie spec: origin at the board's bottom-left
// corner, +X right, +Y to the back, +Z up, board bottom at z = 0 and board
// top at z = thickness. Every case shape is built in this "board frame"; the
// editor places the whole case with the same transform as the board.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
)

// ---------------- file format ----------------

type WirieCase struct {
	Format      string         `json:"format"`
	Version     int            `json:"version"`
	Generator   WirieGenerator `json:"generator"`
	Project     string         `json:"project"`
	Units       string         `json:"units,omitempty"`
	Board       WirieBoard     `json:"board"`
	Holes       []WirieHole    `json:"mountingHoles"`
	Parts       []WiriePart    `json:"parts"`
	Openings    []WirieOpening `json:"openings"`
	LidFeatures []WirieFeature `json:"lidFeatures"`
	Envelope    WirieEnvelope  `json:"envelope"`
	Textures    *WirieTextures `json:"textures,omitempty"`
	Keyboard    *WirieKeyboard `json:"keyboard,omitempty"`
}

type WirieGenerator struct {
	App      string `json:"app,omitempty"`
	Version  string `json:"version,omitempty"`
	Author   string `json:"author,omitempty"`
	Exported string `json:"exported,omitempty"`
}

type WirieBoard struct {
	Width        float64      `json:"width"`
	Height       float64      `json:"height"`
	Thickness    float64      `json:"thickness"`
	Shape        string       `json:"shape"`
	CornerRadius float64      `json:"cornerRadius"`
	Outline      [][2]float64 `json:"outline"`
}

type WirieHole struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	D      float64 `json:"d"`
	Plated bool    `json:"plated,omitempty"`
	Ref    string  `json:"ref,omitempty"`
	Screw  string  `json:"screw,omitempty"`
}

type WirieBox struct {
	X0 float64 `json:"x0"`
	Y0 float64 `json:"y0"`
	X1 float64 `json:"x1"`
	Y1 float64 `json:"y1"`
	Z0 float64 `json:"z0"`
	Z1 float64 `json:"z1"`
}

type WirieRect struct {
	X0 float64 `json:"x0"`
	Y0 float64 `json:"y0"`
	X1 float64 `json:"x1"`
	Y1 float64 `json:"y1"`
}

type WiriePart struct {
	Ref          string     `json:"ref"`
	Value        string     `json:"value,omitempty"`
	Name         string     `json:"name,omitempty"`
	Footprint    string     `json:"footprint,omitempty"`
	Category     string     `json:"category,omitempty"`
	Kind         string     `json:"kind"`
	Side         string     `json:"side"`
	X            float64    `json:"x"`
	Y            float64    `json:"y"`
	Rotation     float64    `json:"rotation"`
	Box          WirieBox   `json:"box"`
	Body         *WirieRect `json:"body,omitempty"`
	Height       float64    `json:"height"`
	RaisedBy     float64    `json:"raisedBy"`
	Pins         int        `json:"pins,omitempty"`
	ApproxHeight bool       `json:"approxHeight,omitempty"`
}

type WirieOpening struct {
	Ref              string     `json:"ref"`
	Type             string     `json:"type"`
	Face             string     `json:"face"`
	X                float64    `json:"x"`
	Y                float64    `json:"y"`
	Z                float64    `json:"z"`
	Width            float64    `json:"width"`
	Height           float64    `json:"height"`
	PartEdge         [2]float64 `json:"partEdge"`
	Normal           [2]float64 `json:"normal"`
	DistanceFromEdge float64    `json:"distanceFromEdge"`
}

type WirieFeature struct {
	Ref     string  `json:"ref"`
	Type    string  `json:"type"`
	Shape   string  `json:"shape"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	D       float64 `json:"d,omitempty"`
	W       float64 `json:"w,omitempty"`
	H       float64 `json:"h,omitempty"`
	Side    string  `json:"side"`
	PartTop float64 `json:"partTop"`
	Approx  bool    `json:"approx,omitempty"`
}

type WirieEnvelope struct {
	MaxPartHeightTop      float64 `json:"maxPartHeightTop"`
	MaxPartHeightBottom   float64 `json:"maxPartHeightBottom"`
	SuggestedStandoff     float64 `json:"suggestedStandoff"`
	SuggestedLidClearance float64 `json:"suggestedLidClearance"`
	SuggestedWall         float64 `json:"suggestedWall"`
	SuggestedFit          float64 `json:"suggestedFit"`
}

type WirieTextures struct {
	PixelsPerMm float64 `json:"pixelsPerMm"`
	Top         string  `json:"top,omitempty"`
	Bottom      string  `json:"bottom,omitempty"`
}

// friendly error for the editor
type WirieError struct{ Msg string }

func (e WirieError) Error() string { return e.Msg }

func werr(f string, a ...any) error { return WirieError{fmt.Sprintf(f, a...)} }

// ParseWirie reads and checks a .makie.json file. Unknown keys are ignored.
// It returns the cleaned-up board plus notes worth showing to the user.
func ParseWirie(b []byte) (*WirieCase, []string, error) {
	b = bytes.TrimPrefix(b, []byte("\xef\xbb\xbf"))
	if len(bytes.TrimSpace(b)) == 0 {
		return nil, nil, werr("The file is empty.")
	}
	var head map[string]json.RawMessage
	if err := json.Unmarshal(b, &head); err != nil {
		if se, ok := err.(*json.SyntaxError); ok {
			return nil, nil, werr("This file isn't readable JSON (problem near character %d). Export it from Wirie again.", se.Offset)
		}
		return nil, nil, werr("This file isn't a Wirie board: it should be a JSON object.")
	}
	var format string
	_ = json.Unmarshal(head["format"], &format)
	if format != "wirie-case" {
		var app string
		_ = json.Unmarshal(head["app"], &app)
		if app == "Makie" || app == "Smithers" {
			return nil, nil, werr("That's a Makie project, not a Wirie board. Use File → Open project instead.")
		}
		if format == "" {
			return nil, nil, werr(`This isn't a Wirie board file. In Wirie, use Export → "Case data for Makie" to make a .makie.json file.`)
		}
		return nil, nil, werr(`This file's format is "%s", but Makie expects "wirie-case". In Wirie, use Export → "Case data for Makie".`, format)
	}
	var ver float64
	if err := json.Unmarshal(head["version"], &ver); err != nil || ver != math.Trunc(ver) {
		return nil, nil, werr("This Wirie file has no valid format version number.")
	}
	if ver > 1 {
		return nil, nil, werr("This board was exported by a newer Wirie (format version %d). Please update Makie to open it.", int(ver))
	}
	if ver < 1 {
		return nil, nil, werr("This Wirie file uses an unknown format version (%d).", int(ver))
	}
	c := &WirieCase{}
	if err := json.Unmarshal(b, c); err != nil {
		msg := err.Error()
		if te, ok := err.(*json.UnmarshalTypeError); ok {
			msg = fmt.Sprintf("“%s” should be a %s", te.Field, te.Type)
		}
		return nil, nil, werr("This Wirie file has a value Makie can't read: %s.", msg)
	}
	notes, err := c.normalize()
	if err != nil {
		return nil, nil, err
	}
	return c, notes, nil
}

func finite(v ...float64) bool {
	for _, x := range v {
		if math.IsNaN(x) || math.IsInf(x, 0) {
			return false
		}
	}
	return true
}

func (c *WirieCase) normalize() ([]string, error) {
	var notes []string
	bd := &c.Board
	if !finite(bd.Width, bd.Height, bd.Thickness, bd.CornerRadius) || bd.Width <= 0 || bd.Height <= 0 {
		return nil, werr("The board has no size (width and height must be above 0 mm).")
	}
	if bd.Width > 2000 || bd.Height > 2000 {
		return nil, werr("The board is %.0f × %.0f mm, which is too big for a 3D-printed case.", bd.Width, bd.Height)
	}
	if bd.Thickness <= 0 || bd.Thickness > 20 {
		bd.Thickness = 1.6
	}
	bd.Shape = strings.ToLower(bd.Shape)
	if bd.Shape == "round" {
		bd.CornerRadius = bd.Width / 2
	}
	bd.CornerRadius = clampF(bd.CornerRadius, 0, math.Min(bd.Width, bd.Height)/2)
	if c.Project == "" {
		c.Project = "Wirie board"
	}
	// outline: closed, CCW, no repeated end point
	var l Loop
	for _, p := range bd.Outline {
		if finite(p[0], p[1]) {
			l = append(l, P2{p[0], p[1]})
		}
	}
	if len(l) > 1 && samePt(l[0], l[len(l)-1]) {
		l = l[:len(l)-1]
	}
	l = l.Clean()
	if len(l) < 3 || math.Abs(l.Area()) < 1 {
		if len(bd.Outline) > 0 {
			notes = append(notes, "The board outline was unusable, so Makie used the board's width and height instead.")
		}
		l = c.defaultOutline()
	}
	if l.Area() < 0 {
		l = l.Reversed()
	}
	bd.Outline = bd.Outline[:0]
	for _, p := range l {
		bd.Outline = append(bd.Outline, [2]float64{round3(p.X), round3(p.Y)})
	}
	// mounting holes
	var hs []WirieHole
	for i, h := range c.Holes {
		if !finite(h.X, h.Y, h.D) || h.D <= 0.5 || h.D > 20 {
			notes = append(notes, fmt.Sprintf("Mounting hole %s was skipped (its size isn't usable).", orStr(h.Ref, fmt.Sprint(i+1))))
			continue
		}
		if _, ok := metrics[h.Screw]; !ok {
			h.Screw = screwForHole(h.D)
		}
		if h.Ref == "" {
			h.Ref = fmt.Sprintf("H%d", i+1)
		}
		hs = append(hs, h)
	}
	c.Holes = hs
	// parts
	var ps []WiriePart
	for i := range c.Parts {
		p := c.Parts[i]
		b := &p.Box
		if !finite(b.X0, b.Y0, b.X1, b.Y1, b.Z0, b.Z1) {
			continue
		}
		if b.X0 > b.X1 {
			b.X0, b.X1 = b.X1, b.X0
		}
		if b.Y0 > b.Y1 {
			b.Y0, b.Y1 = b.Y1, b.Y0
		}
		if b.Z0 > b.Z1 {
			b.Z0, b.Z1 = b.Z1, b.Z0
		}
		if p.Side != "bottom" {
			p.Side = "top"
		}
		if p.Kind == "" {
			p.Kind = "component"
		}
		if p.Ref == "" {
			p.Ref = fmt.Sprintf("P%d", i+1)
		}
		ps = append(ps, p)
	}
	c.Parts = ps
	var os []WirieOpening
	for _, o := range c.Openings {
		if !finite(o.X, o.Y, o.Z, o.Width, o.Height) || o.Width <= 0 || o.Height <= 0 {
			continue
		}
		o.Face = strings.ToLower(o.Face)
		if o.Normal[0] == 0 && o.Normal[1] == 0 {
			o.Normal = faceNormal2(o.Face)
		}
		n := math.Hypot(o.Normal[0], o.Normal[1])
		o.Normal = [2]float64{o.Normal[0] / n, o.Normal[1] / n}
		os = append(os, o)
	}
	c.Openings = os
	var fs []WirieFeature
	for _, f := range c.LidFeatures {
		if !finite(f.X, f.Y, f.D, f.W, f.H) {
			continue
		}
		f.Shape = strings.ToLower(f.Shape)
		if f.Shape != "rect" {
			f.Shape = "circle"
			if f.D <= 0 {
				f.D = math.Max(f.W, f.H)
			}
			if f.D <= 0 {
				continue
			}
		} else if f.W <= 0 || f.H <= 0 {
			continue
		}
		if f.Side != "bottom" {
			f.Side = "top"
		}
		fs = append(fs, f)
	}
	c.LidFeatures = fs
	// envelope: fill anything missing from the parts
	e := &c.Envelope
	top, bot := 0.0, 0.0
	for _, p := range c.Parts {
		top = math.Max(top, p.Box.Z1-bd.Thickness)
		bot = math.Max(bot, -p.Box.Z0)
	}
	if e.MaxPartHeightTop <= 0 {
		e.MaxPartHeightTop = top
	}
	if e.MaxPartHeightBottom <= 0 {
		e.MaxPartHeightBottom = bot
	}
	if e.SuggestedStandoff <= 0 {
		e.SuggestedStandoff = math.Max(3, e.MaxPartHeightBottom+1)
	}
	if e.SuggestedLidClearance <= 0 {
		e.SuggestedLidClearance = e.MaxPartHeightTop + 2
	}
	if e.SuggestedWall <= 0 {
		e.SuggestedWall = 2
	}
	if e.SuggestedFit <= 0 {
		e.SuggestedFit = 0.3
	}
	if c.Textures != nil {
		t := c.Textures
		if !strings.HasPrefix(t.Top, "data:image/") {
			t.Top = ""
		}
		if !strings.HasPrefix(t.Bottom, "data:image/") {
			t.Bottom = ""
		}
		if t.Top == "" && t.Bottom == "" {
			c.Textures = nil
		}
	}
	for _, o := range c.Openings {
		if o.DistanceFromEdge > 1 {
			notes = append(notes, fmt.Sprintf("%s (%s) sits %.1f mm back from the board edge, so its plug has to reach into the case.", o.Ref, o.Type, o.DistanceFromEdge))
		}
	}
	for _, f := range c.LidFeatures {
		if f.Approx {
			notes = append(notes, fmt.Sprintf("The %s for %s is Wirie's estimate (%s). Check it against the real part.", featureName(f.Type), f.Ref, featureSize(f)))
		}
	}
	return notes, nil
}

func capFirst(s string) string {
	if s == "" {
		return s
	}
	if strings.HasPrefix(s, "led ") {
		return "LED" + s[3:]
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func featureName(t string) string {
	n := strings.ReplaceAll(t, "-", " ")
	if !strings.HasSuffix(n, "window") && !strings.HasSuffix(n, "hole") && !strings.HasSuffix(n, "holes") {
		n += " hole"
	}
	return n
}

func featureSize(f WirieFeature) string {
	if f.Shape == "rect" {
		return fmt.Sprintf("%.1f × %.1f mm", f.W, f.H)
	}
	return fmt.Sprintf("Ø%.1f mm", f.D)
}

func orStr(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

func faceNormal2(face string) [2]float64 {
	switch face {
	case "left":
		return [2]float64{-1, 0}
	case "right":
		return [2]float64{1, 0}
	case "front":
		return [2]float64{0, -1}
	}
	return [2]float64{0, 1}
}

func screwForHole(d float64) string {
	switch {
	case d <= 2.3:
		return "M2"
	case d <= 2.85:
		return "M2.5"
	case d <= 3.7:
		return "M3"
	case d <= 4.8:
		return "M4"
	}
	return "M5"
}

func (c *WirieCase) defaultOutline() Loop {
	bd := c.Board
	if bd.Shape == "round" {
		return Circle(bd.Width/2, 72).translated(bd.Width/2, bd.Height/2)
	}
	return RoundedRect(bd.Width, bd.Height, bd.CornerRadius, 8).translated(bd.Width/2, bd.Height/2)
}

func (l Loop) translated(dx, dy float64) Loop {
	out := make(Loop, len(l))
	for i, p := range l {
		out[i] = P2{p.X + dx, p.Y + dy}
	}
	return out
}

// Compact returns the board as JSON without its pictures (that's what the
// editor keeps in shape settings; the pictures live alongside).
func (c *WirieCase) Compact() string {
	cp := *c
	cp.Textures = nil
	b, _ := json.Marshal(&cp)
	return string(b)
}

// wirieFromParams decodes the "board" setting of a Wirie shape.
func wirieFromParams(p Params) (*WirieCase, error) {
	s := p.S("board", "")
	if s == "" {
		return nil, fmt.Errorf("no board data")
	}
	c := &WirieCase{}
	if err := json.Unmarshal([]byte(s), c); err != nil {
		return nil, fmt.Errorf("board data is damaged")
	}
	if _, err := c.normalize(); err != nil {
		return nil, err
	}
	return c, nil
}

// WirieFromBoardJSON decodes the compact board kept by the editor.
func WirieFromBoardJSON(s string) (*WirieCase, error) {
	return wirieFromParams(Params{"board": s})
}

// ---------------- board outline helpers ----------------

func (c *WirieCase) loop() Loop {
	l := make(Loop, len(c.Board.Outline))
	for i, p := range c.Board.Outline {
		l[i] = P2{p[0], p[1]}
	}
	return l
}

func (c *WirieCase) isRect() bool { return c.Board.Shape == "rect" || c.Board.Shape == "" }

// outlineAt grows the board outline by g. For rectangular and round boards
// the corner radius is r (r < 0: follow the board's own corners).
func (c *WirieCase) outlineAt(g, r float64, segs int) Loop {
	bd := c.Board
	switch {
	case bd.Shape == "round":
		return Circle(bd.Width/2+g, segs*4).translated(bd.Width/2, bd.Height/2)
	case c.isRect():
		if r < 0 {
			r = bd.CornerRadius + g
		}
		r = math.Max(0, r)
		return RoundedRect(bd.Width+2*g, bd.Height+2*g, r, segs).translated(bd.Width/2, bd.Height/2)
	}
	if math.Abs(g) < 1e-9 {
		return c.loop()
	}
	if bd.Shape == "custom" {
		return rasterOffset(c.loop(), g)
	}
	return offsetLoop(c.loop(), g)
}

// ---------------- case options ----------------

type CaseOpts struct {
	Wall, Floor, Fit, Standoff, Clear, Radius, LidT float64
	Lid, Screw, Fixing                              string
	Vents, Caps, Pipes, LedPipes, Bezel, Csk        bool
	Text, Font, Logo                                string
	TextSize                                        float64
	segs                                            int
	raised                                          bool
	clearFrom, standFrom                            float64 // settings that had to be raised
}

var lidStyles = []string{"Screw-down", "Snap-fit", "Slide-in"}
var caseFixings = []string{"Self-tapping screw", "Heat-set insert"}

func caseOpts(c *WirieCase, p Params) CaseOpts {
	e := c.Envelope
	o := CaseOpts{
		Wall:     clampF(p.F("wall", math.Max(e.SuggestedWall, 1.6)), 1.2, 10),
		Floor:    clampF(p.F("floor", 2), 1.2, 10),
		Fit:      clampF(p.F("fit", e.SuggestedFit), 0, 3),
		Standoff: p.F("standoff", 0),
		Clear:    p.F("clear", 0),
		Radius:   p.F("radius", -1),
		LidT:     clampF(p.F("lidT", 2), 1.2, 10),
		Lid:      p.S("lid", "Screw-down"),
		Screw:    p.S("screw", "From board"),
		Fixing:   p.S("fixing", "Self-tapping screw"),
		Vents:    p.B("vents", false),
		Caps:     p.B("caps", true),
		Pipes:    p.B("pipes", true),
		LedPipes: p.B("ledPipes", false),
		Bezel:    p.B("bezel", false),
		Csk:      p.B("csk", false),
		Text:     strings.TrimSpace(p.S("text", "")),
		Font:     p.S("font", "Sans Bold"),
		Logo:     p.S("logo", "None"),
		TextSize: clampF(p.F("textSize", 6), 2, 40),
		segs:     int(math.Max(4, math.Round(4*p.Q()))),
	}
	if o.Standoff <= 0 {
		o.Standoff = e.SuggestedStandoff
	}
	if need := e.MaxPartHeightBottom + 0.5; o.Standoff < need {
		if p.F("standoff", 0) > 0 {
			o.standFrom = o.Standoff
		}
		o.Standoff = need
	}
	o.Standoff = clampF(o.Standoff, 1, 100)
	if o.Clear <= 0 {
		o.Clear = e.SuggestedLidClearance
	}
	if need := e.MaxPartHeightTop + 0.5; o.Clear < need {
		if p.F("clear", 0) > 0 {
			o.clearFrom = o.Clear
		}
		o.Clear = need
	}
	if o.Csk {
		o.LidT = math.Max(o.LidT, 2.6)
	}
	// heat-set inserts need enough plastic under the board
	if o.Fixing == "Heat-set insert" && len(c.Holes) > 0 {
		m := metrics[c.screwName(o.Screw)]
		if need := m.InsertL + 1.5 - o.Floor; m.InsertD > 0 && o.Standoff < need {
			o.Standoff, o.raised = round3(need), true
		}
	}
	return o
}

func (c *WirieCase) screwName(s string) string {
	if _, ok := metrics[s]; ok {
		return s
	}
	if len(c.Holes) > 0 {
		return c.Holes[0].Screw
	}
	return "M3"
}

// CaseParams lists the case settings, with defaults taken from the board.
func CaseParams(c *WirieCase) []Param {
	if c.isKeyboard() {
		return kbCaseParams(c)
	}
	e := c.Envelope
	screws := []string{"From board"}
	for _, s := range []string{"M2", "M2.5", "M3", "M4", "M5"} {
		screws = append(screws, s)
	}
	return append([]Param{
		sel("lid", "Lid style", "Screw-down", lidStyles...).help("Screw-down: one long screw per mounting hole holds the board and the lid. Snap-fit: the lid clicks on. Slide-in: the lid slides along grooves from the right-hand end."),
		num("wall", "Wall thickness", round3(math.Max(e.SuggestedWall, 1.6)), 1.2, 10, 0.2, "mm"),
		num("floor", "Floor thickness", 2, 1.2, 10, 0.2, "mm"),
		num("fit", "Fit gap around the board", round3(e.SuggestedFit), 0, 3, 0.05, "mm"),
		num("standoff", "Standoff height", round3(math.Max(e.SuggestedStandoff, e.MaxPartHeightBottom+0.5)), 1, 100, 0.5, "mm").help("Space between the case floor and the bottom of the board."),
		num("clear", "Lid clearance", round3(math.Max(e.SuggestedLidClearance, e.MaxPartHeightTop+0.5)), 1, 200, 0.5, "mm").help("Space between the top of the board and the underside of the lid."),
		num("lidT", "Lid thickness", 2, 1.2, 10, 0.2, "mm"),
		num("radius", "Corner radius (-1 = follow the board)", -1, -1, 50, 0.5, "mm"),
		sel("screw", "Screw size", "From board", screws...),
		sel("fixing", "Screws go into", "Self-tapping screw", caseFixings...),
		boolp("csk", "Countersunk lid screws", false).show("lid=Screw-down"),
		boolp("vents", "Vent slots in the walls", false),
		boolp("caps", "Button extenders", true),
		boolp("pipes", "Light pipes for light-pipe parts", true),
		boolp("ledPipes", "Light pipes for LEDs too", false),
		boolp("bezel", "Bezel around the display", false),
		{Key: "text", Label: "Text on the lid", Type: "text", Def: ""},
		sel("font", "Font", "Sans Bold", FontNames()...),
		num("textSize", "Text size", 6, 2, 40, 0.5, "mm"),
		sel("logo", "Logo on the lid", "None", "None", "Lightning bolt", "Heart", "Star"),
	}, feetParams(boardFeetDefault)...)
}

// board cases had a flat bottom before feet were a choice, so that stays the default
const boardFeetDefault = "No feet"

// derived heights of the case, in the board frame
type caseDims struct {
	t, zFloorBot, zFloorTop, zTop, zLidTop, zWallTop float64
	gd                                               float64 // slide-in groove depth
	outer, cavity                                    Loop
	lid                                              string
	m                                                metric
	screw                                            string
}

func (c *WirieCase) dims(o CaseOpts) caseDims {
	d := caseDims{t: c.Board.Thickness}
	d.zFloorTop = -o.Standoff
	d.zFloorBot = d.zFloorTop - o.Floor
	d.zTop = d.t + o.Clear
	d.zLidTop = d.zTop + o.LidT
	d.zWallTop = d.zTop
	d.lid = o.Lid
	if d.lid == "Slide-in" && !c.isRect() {
		d.lid = "Snap-fit"
	}
	if d.lid == "Screw-down" && len(c.Holes) == 0 {
		d.lid = "Snap-fit"
	}
	r := o.Radius
	if r < 0 {
		r = c.Board.CornerRadius + o.Fit + o.Wall
	}
	d.outer = c.outlineAt(o.Fit+o.Wall, r, o.segs)
	d.cavity = c.outlineAt(o.Fit, math.Max(0, r-o.Wall), o.segs)
	if d.lid == "Slide-in" {
		d.gd = clampF(o.Wall-1.2, 0.6, 1.5)
		d.zWallTop = d.zLidTop + 1.2
	}
	d.screw = c.screwName(o.Screw)
	d.m = metrics[d.screw]
	return d
}

func (c *WirieCase) cavityBounds(o CaseOpts) (x0, y0, x1, y1 float64) {
	return -o.Fit, -o.Fit, c.Board.Width + o.Fit, c.Board.Height + o.Fit
}

// ---------------- shapes ----------------

func wirieParams() []Param {
	return []Param{{Key: "board", Label: "Board", Type: "hidden", Def: ""}}
}

func init() {
	register(&ShapeDef{Type: "wirieboard", Name: "Wirie board", Category: "Wirie", Hidden: true, Fixed: true, Color: "#166534",
		Desc: "A circuit board imported from Wirie, shown with its parts as blocks.", Params: wirieParams(),
		Gen: func(p Params) (*Mesh, error) {
			c, err := wirieFromParams(p)
			if err != nil {
				return nil, err
			}
			return c.BoardMesh(p.Segs(4)), nil
		}})

	caseShape := func(typ, name string, hole bool, gen func(c *WirieCase, o CaseOpts, p Params) *Mesh) {
		register(&ShapeDef{Type: typ, Name: name, Category: "Wirie", Hidden: true, Hole: hole, Fixed: true, Color: "#64748b",
			Desc: "Part of a case made for a Wirie board.", Params: wirieParams(),
			Gen: func(p Params) (*Mesh, error) {
				c, err := wirieFromParams(p)
				if err != nil {
					return nil, err
				}
				return gen(c, caseOpts(c, p), p), nil
			}})
	}
	caseShape("wcase_tray", "Case tray", false, func(c *WirieCase, o CaseOpts, p Params) *Mesh { return c.trayMesh(o) })
	caseShape("wcase_lid", "Case lid plate", false, func(c *WirieCase, o CaseOpts, p Params) *Mesh { return c.lidMesh(o) })

	// simple parametric pieces (no board data needed)
	register(&ShapeDef{Type: "wcase_cutout", Name: "Case cut-out", Category: "Wirie", Hidden: true, Hole: true, Fixed: false, Color: "#64748b",
		Desc: "A rounded rectangular cut-out, wider on the outside (chamfered). Points along +Z.",
		Params: []Param{
			num("w", "Width", 10, 0.5, 500, 0.5, "mm"), num("h", "Height", 6, 0.5, 500, 0.5, "mm"),
			num("din", "Depth inside", 2, 0, 100, 0.5, "mm"), num("dout", "Depth outside", 2, 0.1, 100, 0.5, "mm"),
			num("cham", "Outer chamfer", 0.6, 0, 5, 0.1, "mm"), num("r", "Corner radius", 0.5, 0, 50, 0.1, "mm"),
			boolp("circle", "Round", false),
		},
		Gen: func(p Params) (*Mesh, error) {
			return cutoutMesh(p.F("w", 10), p.F("h", 6), p.F("din", 2), p.F("dout", 2), p.F("cham", 0.6), p.F("r", 0.5), p.B("circle", false), p.Segs(math.Max(p.F("w", 10), p.F("h", 6)))), nil
		}})
	register(&ShapeDef{Type: "wcase_standoff", Name: "Case standoff", Category: "Wirie", Hidden: true, Fixed: true, Color: "#64748b",
		Desc:   "A standoff with a flared foot.",
		Params: []Param{num("od", "Outer diameter", 7, 2, 50, 0.5, "mm"), num("h", "Height", 3, 0.5, 200, 0.5, "mm"), num("flare", "Foot flare", 1.5, 0, 10, 0.25, "mm")},
		Gen: func(p Params) (*Mesh, error) {
			od, h, fl := p.F("od", 7), p.F("h", 3), p.F("flare", 1.5)
			fl = math.Min(fl, h*0.6)
			segs := p.Segs(od + 2*fl)
			prof := []P2{{0, 0}, {od/2 + fl, 0}}
			for i := 1; i <= 4; i++ { // concave fillet into the floor
				a := float64(i) / 4 * math.Pi / 2
				prof = append(prof, P2{od/2 + fl - fl*math.Sin(a), fl - fl*math.Cos(a)})
			}
			prof = append(prof, P2{od / 2, h}, P2{0, h})
			return RevolveProfile(prof, segs), nil
		}})
	register(&ShapeDef{Type: "wcase_post", Name: "Lid screw post", Category: "Wirie", Hidden: true, Fixed: true, Color: "#64748b",
		Desc:   "A tube that hangs under the lid and presses the board down.",
		Params: []Param{num("od", "Outer diameter", 7, 2, 50, 0.5, "mm"), num("id", "Hole", 3.4, 0.5, 40, 0.1, "mm"), num("h", "Length", 10, 0.5, 200, 0.5, "mm")},
		Gen: func(p Params) (*Mesh, error) {
			od, id, h := p.F("od", 7), p.F("id", 3.4), p.F("h", 10)
			return ring(od, math.Min(id, od-2.4), 0, h, p.Segs(od)), nil
		}})
	register(&ShapeDef{Type: "wcase_grille", Name: "Hole grille", Category: "Wirie", Hidden: true, Hole: true, Color: "#64748b",
		Desc: "A grid of small round holes for sound or air, inside a circle or a rectangle.",
		Params: []Param{
			sel("shape", "Area", "circle", "circle", "rect"), num("d", "Area diameter", 12, 2, 500, 0.5, "mm"),
			num("w", "Area width", 12, 2, 500, 0.5, "mm"), num("hh", "Area depth", 12, 2, 500, 0.5, "mm"),
			num("hole", "Hole size", 1.6, 0.6, 10, 0.1, "mm"), num("pitch", "Spacing", 3, 1, 20, 0.1, "mm"), num("h", "Depth", 5, 0.2, 100, 0.5, "mm"),
		},
		Gen: func(p Params) (*Mesh, error) { return grilleMesh(p), nil }})
	register(&ShapeDef{Type: "wcase_plunger", Name: "Button extender / light pipe", Category: "Wirie", Hidden: true, Fixed: true, Color: "#f8fafc",
		Desc:   "A rod that pokes up through the lid. A 45° collar under the lid keeps it from falling out and lets it print upright.",
		Params: []Param{num("d", "Rod diameter", 3.4, 1, 30, 0.1, "mm"), num("len", "Length", 10, 1, 200, 0.5, "mm"), num("collar", "Collar diameter", 7, 2, 50, 0.5, "mm"), num("cz", "Collar top height", 6, 0.5, 200, 0.1, "mm"), boolp("dome", "Rounded top", true)},
		Gen: func(p Params) (*Mesh, error) {
			d, L, cd, cz := p.F("d", 3.4), p.F("len", 10), p.F("collar", 7), p.F("cz", 6)
			cd = math.Max(cd, d+1)
			cz = clampF(cz, 0.5, L-0.5)
			ct := 0.8
			taper := (cd - d) / 2
			z0 := math.Max(0, cz-ct-taper)
			prof := []P2{{0, 0}, {d / 2, 0}, {d / 2, z0}, {cd / 2, z0 + taper}, {cd / 2, cz}, {d / 2, cz}}
			if p.B("dome", true) && d > 1.5 {
				r := math.Min(d/2*0.6, 1.2)
				for i := 0; i <= 4; i++ {
					a := float64(i) / 4 * math.Pi / 2
					prof = append(prof, P2{d/2 - r + r*math.Cos(a), L - r + r*math.Sin(a)})
				}
				prof = append(prof, P2{0, L})
			} else {
				prof = append(prof, P2{d / 2, L}, P2{0, L})
			}
			return RevolveProfile(prof, p.Segs(cd)), nil
		}})
	register(&ShapeDef{Type: "wcase_bezel", Name: "Display bezel", Category: "Wirie", Hidden: true, Fixed: true, Color: "#0f172a",
		Desc:   "A frame that sits on the lid around a screen window, with a 45° inner edge.",
		Params: []Param{num("w", "Window width", 24, 2, 500, 0.5, "mm"), num("h", "Window depth", 20, 2, 500, 0.5, "mm"), num("border", "Border", 4, 1, 50, 0.5, "mm"), num("t", "Thickness", 1.6, 0.6, 20, 0.2, "mm")},
		Gen: func(p Params) (*Mesh, error) {
			w, h, b, t := p.F("w", 24), p.F("h", 20), p.F("border", 4), p.F("t", 1.6)
			outer := extrudeAt(Shape{Outer: RoundedRect(w+2*b, h+2*b, 1.5, 6)}, 0, t)
			ch := math.Min(t*0.6, b*0.5)
			hole := Loft([][]Vec3{rrRing(w, h, 0.5, 4, -1), rrRing(w, h, 0.5, 4, t-ch), rrRing(w+2*ch, h+2*ch, 0.5+ch, 4, t), rrRing(w+2*ch+2, h+2*ch+2, 1.5+ch, 4, t+1)}, true, true)
			return Subtract(outer, hole), nil
		}})
	register(&ShapeDef{Type: "wcase_logo", Name: "Lid logo", Category: "Wirie", Hidden: true, Color: "#e11d48",
		Desc:   "A simple symbol for the lid.",
		Params: []Param{sel("kind", "Symbol", "Lightning bolt", "Lightning bolt", "Heart", "Star"), num("size", "Size", 12, 2, 200, 0.5, "mm"), num("h", "Thickness", 1, 0.2, 50, 0.2, "mm")},
		Gen: func(p Params) (*Mesh, error) {
			return extrudeAt(Shape{Outer: logoLoop(p.S("kind", "Lightning bolt"), p.F("size", 12))}, 0, p.F("h", 1)), nil
		}})
}

func logoLoop(kind string, s float64) Loop {
	var l Loop
	switch kind {
	case "Heart":
		for i := 0; i < 64; i++ {
			t := float64(i) / 64 * 2 * math.Pi
			x := 16 * math.Pow(math.Sin(t), 3)
			y := 13*math.Cos(t) - 5*math.Cos(2*t) - 2*math.Cos(3*t) - math.Cos(4*t)
			l = append(l, P2{x / 34 * s, (y + 2) / 34 * s})
		}
	case "Star":
		for i := 0; i < 10; i++ {
			r := s / 2
			if i%2 == 1 {
				r = s * 0.2
			}
			a := math.Pi/2 + math.Pi*float64(i)/5
			l = append(l, P2{r * math.Cos(a), r * math.Sin(a)})
		}
	default: // lightning bolt
		pts := [][2]float64{{0.12, 0.5}, {-0.3, -0.04}, {-0.02, -0.04}, {-0.14, -0.5}, {0.3, 0.08}, {0.02, 0.08}}
		for _, q := range pts {
			l = append(l, P2{q[0] * s, q[1] * s})
		}
	}
	if l.Area() < 0 {
		l = l.Reversed()
	}
	return l
}

// rounded-rectangle ring in a plane of constant z (helper for lofts)
func rrRingC(w, h, r float64, segs int, z float64, circle bool) []Vec3 {
	if circle {
		n := segs * 4
		out := make([]Vec3, n)
		for i := range out {
			a := 2 * math.Pi * float64(i) / float64(n)
			out[i] = V(w/2*math.Cos(a), w/2*math.Sin(a), z)
		}
		return out
	}
	return rrRing(w, h, r, segs, z)
}

// cutoutMesh: rounded rectangle (or circle of diameter w) along +Z from
// z = -din to z = dout, flaring out by cham over the last cham mm.
func cutoutMesh(w, h, din, dout, cham, r float64, circle bool, segs int) *Mesh {
	q := int(math.Max(3, float64(segs)/4))
	if circle {
		h = w
	}
	r = clampF(r, 0.01, math.Min(w, h)/2-0.01)
	cham = clampF(cham, 0, dout-0.05)
	rings := [][]Vec3{rrRingC(w, h, r, q, -din, circle)}
	if cham > 0.01 {
		rings = append(rings, rrRingC(w, h, r, q, dout-cham, circle), rrRingC(w+2*cham, h+2*cham, r+cham, q, dout, circle), rrRingC(w+2*cham, h+2*cham, r+cham, q, dout+0.6, circle))
	} else {
		rings = append(rings, rrRingC(w, h, r, q, dout, circle))
	}
	return Loft(rings, true, true)
}

func grilleMesh(p Params) *Mesh {
	circle := p.S("shape", "circle") != "rect"
	hd, pitch, h := p.F("hole", 1.6), p.F("pitch", 3), p.F("h", 5)
	pitch = math.Max(pitch, hd+0.8)
	var w, d float64
	if circle {
		w, d = p.F("d", 12), p.F("d", 12)
	} else {
		w, d = p.F("w", 12), p.F("hh", 12)
	}
	segs := int(math.Max(8, math.Round(hd*math.Pi/0.6)))
	var holes []*Mesh
	rowH := pitch * math.Sqrt(3) / 2
	ny := int(math.Floor((d - hd) / rowH / 2))
	for j := -ny; j <= ny; j++ {
		y := float64(j) * rowH
		off := 0.0
		if j%2 != 0 {
			off = pitch / 2
		}
		nx := int(math.Ceil(w/pitch)) + 1
		for i := -nx; i <= nx; i++ {
			x := float64(i)*pitch + off
			if circle {
				if math.Hypot(x, y)+hd/2 > w/2 {
					continue
				}
			} else if math.Abs(x)+hd/2 > w/2 || math.Abs(y)+hd/2 > d/2 {
				continue
			}
			holes = append(holes, cylAt(x, y, hd, 0, h, segs))
		}
	}
	m := &Mesh{}
	for _, hm := range holes { // separate closed cylinders: no CSG needed
		m.Append(hm)
	}
	if len(m.Tris) == 0 {
		m = cylAt(0, 0, hd, 0, h, segs)
	}
	return m
}

// BoardMesh: the board slab with real mounting holes, then one 12-triangle
// block per part (in file order, skipping flat ones) so the editor can colour
// and pick each part.
func (c *WirieCase) BoardMesh(segs int) *Mesh {
	t := c.Board.Thickness
	s := Shape{Outer: c.loop()}
	for _, h := range c.Holes {
		s.Holes = append(s.Holes, Circle(h.D/2, int(math.Max(12, float64(segs)*3))).translated(h.X, h.Y))
	}
	m := extrudeAt(s.Normalize(), 0, t)
	for _, p := range c.Parts {
		if !partSolid(p) {
			continue
		}
		b := p.Box
		m.Append(box12(V(b.X0, b.Y0, b.Z0), V(b.X1, b.Y1, b.Z1)))
	}
	return m
}

func partSolid(p WiriePart) bool {
	return p.Box.Z1-p.Box.Z0 > 0.01 && p.Box.X1-p.Box.X0 > 0.01 && p.Box.Y1-p.Box.Y0 > 0.01
}

func box12(lo, hi Vec3) *Mesh {
	m := &Mesh{}
	c := func(i int) Vec3 {
		return V(pick(i&1 != 0, hi.X, lo.X), pick(i&2 != 0, hi.Y, lo.Y), pick(i&4 != 0, hi.Z, lo.Z))
	}
	m.AddQuad(c(0), c(2), c(3), c(1)) // bottom
	m.AddQuad(c(4), c(5), c(7), c(6)) // top
	m.AddQuad(c(0), c(1), c(5), c(4)) // front
	m.AddQuad(c(2), c(6), c(7), c(3)) // back
	m.AddQuad(c(0), c(4), c(6), c(2)) // left
	m.AddQuad(c(1), c(3), c(7), c(5)) // right
	return m
}

func pick(b bool, x, y float64) float64 {
	if b {
		return x
	}
	return y
}

// trayMesh: floor + walls (open top), with the lid's catch features.
func (c *WirieCase) trayMesh(o CaseOpts) *Mesh {
	d := c.dims(o)
	cup := Subtract(extrudeAt(Shape{Outer: d.outer}, d.zFloorBot, d.zWallTop), extrudeAt(Shape{Outer: d.cavity}, d.zFloorTop, d.zWallTop+1))
	switch d.lid {
	case "Snap-fit":
		for _, g := range c.snapSpots(o, d) {
			cup = Subtract(cup, g.groove)
		}
	case "Slide-in":
		_, y0, x1, y1 := c.cavityBounds(o)
		w := o.Fit + o.Wall
		// groove follows the rounded cavity at the closed end, straight out at the open end
		rg := math.Max(0, c.cornerR(o)-o.Wall) + d.gd
		groove := Union(extrudeAt(Shape{Outer: c.outlineAt(o.Fit+d.gd, rg, o.segs)}, d.zTop, d.zLidTop),
			boxAt(V(c.Board.Width/2, y0-d.gd, d.zTop), V(c.Board.Width+w+5, y1+d.gd, d.zLidTop)))
		// open end: everything above the lid seat goes, including the rounded
		// corners (they would leave thin slivers beside the groove)
		endX := math.Min(x1-0.01, c.Board.Width+w-c.cornerR(o)-0.01)
		end := boxAt(V(endX, y0-w-5, d.zTop), V(c.Board.Width+w+5, y1+w+5, d.zWallTop+1))
		cup = Subtract(Subtract(cup, groove), end)
	}
	return cup
}

type snapSpot struct{ bump, groove *Mesh }

// snap-fit catches: a ramp on the lid's lip and a groove in the wall, in the
// middle of each straight side of a rectangular case
func (c *WirieCase) snapSpots(o CaseOpts, d caseDims) []snapSpot {
	if !c.isRect() {
		return nil
	}
	x0, y0, x1, y1 := c.cavityBounds(o)
	zc := d.zTop - lipDepth(o) + 1.6
	var out []snapSpot
	add := func(cx, cy, length float64, axis int, outward float64) {
		// bump: triangle profile (sticks out 0.45 from the lip face)
		lipFace := -0.25
		tri := Loop{{lipFace - 0.05, zc - 0.8}, {lipFace + 0.45, zc}, {lipFace - 0.05, zc + 0.8}}
		gr := Loop{{-0.3, zc - 1.1}, {0.65, zc - 1.1}, {0.65, zc + 1.1}, {-0.3, zc + 1.1}}
		mk := func(l Loop, ext float64) *Mesh {
			m := shapeX(Shape{Outer: l}, -ext/2, ext/2) // profile in (y=outward, z)
			// orient: outward direction (+Y in profile) → the wall normal
			var rot Mat4
			switch {
			case axis == 0 && outward > 0:
				rot = Identity()
			case axis == 0:
				rot = RotZ(180)
			case outward > 0:
				rot = RotZ(-90)
			default:
				rot = RotZ(90)
			}
			return m.Transformed(Translate(cx, cy, 0).Mul(rot))
		}
		out = append(out, snapSpot{mk(tri, length), mk(gr, length+1)})
	}
	// keep the catches clear of wall openings (a groove right next to a
	// cut-out leaves a sliver of wall)
	gz0, gz1 := zc-1.1-1.5, zc+1.1+1.5
	free := func(face string, at, l float64) bool {
		for _, op := range c.Openings {
			if op.Face != face {
				continue
			}
			along := op.X
			if face == "left" || face == "right" {
				along = op.Y
			}
			lo, hi := op.Z-op.Height/2, op.Z+op.Height/2
			if op.DistanceFromEdge < -0.05 {
				hi = math.Inf(1)
			}
			if math.Abs(along-at) < op.Width/2+l/2+2 && hi > gz0 && lo < gz1 {
				return false
			}
		}
		return true
	}
	place := func(face string, a0, a1 float64, put func(c, l float64)) {
		for _, f := range []float64{0.5, 0.3, 0.7, 0.2, 0.8} {
			l := (a1 - a0) * 0.35
			if f != 0.5 {
				l = (a1 - a0) * 0.2
			}
			cc := a0 + (a1-a0)*f
			if free(face, cc, l) {
				put(cc, l)
				return
			}
		}
	}
	place("back", x0, x1, func(cc, l float64) { add(cc, y1, l, 0, 1) })
	place("front", x0, x1, func(cc, l float64) { add(cc, y0, l, 0, -1) })
	place("right", y0, y1, func(cc, l float64) { add(x1, cc, l, 1, 1) })
	place("left", y0, y1, func(cc, l float64) { add(x0, cc, l, 1, -1) })
	return out
}

func lipDepth(o CaseOpts) float64 {
	if o.Lid == "Snap-fit" {
		return math.Min(4, o.Clear-0.5)
	}
	return math.Min(2.5, o.Clear-0.5)
}

// lidMesh: the lid plate (in the board frame, sitting on the walls) with its
// locating lip and catches.
func (c *WirieCase) lidMesh(o CaseOpts) *Mesh {
	d := c.dims(o)
	if d.lid == "Slide-in" {
		_, y0, _, y1 := c.cavityBounds(o)
		w := o.Fit + o.Wall
		rg := math.Max(0, c.cornerR(o)-o.Wall) + d.gd - 0.25
		return Union(extrudeAt(Shape{Outer: c.outlineAt(o.Fit+d.gd-0.25, rg, o.segs)}, d.zTop+0.15, d.zLidTop-0.15),
			boxAt(V(c.Board.Width/2, y0-d.gd+0.25, d.zTop+0.15), V(c.Board.Width+w, y1+d.gd-0.25, d.zLidTop-0.15)))
	}
	plate := extrudeAt(Shape{Outer: d.outer}, d.zTop, d.zLidTop)
	ld := lipDepth(o)
	lipOut := c.outlineAt(o.Fit-0.25, math.Max(0, c.cornerR(o)-o.Wall-0.25), o.segs)
	lipIn := c.outlineAt(o.Fit-0.25-1.6, math.Max(0, c.cornerR(o)-o.Wall-1.85), o.segs)
	lip := extrudeAt(Shape{Outer: lipOut, Holes: []Loop{lipIn}}.Normalize(), d.zTop-ld, d.zTop+0.01)
	parts := []*Mesh{plate, lip}
	if d.lid == "Snap-fit" {
		for _, s := range c.snapSpots(o, d) {
			parts = append(parts, s.bump)
		}
	}
	return UnionAll(parts)
}

// edgeDist: distance from a point on the board to the cavity wall
func (c *WirieCase) edgeDist(x, y, fit float64) float64 {
	l := c.outlineAt(fit, -1, 8)
	best := math.Inf(1)
	for i := range l {
		a, b := l[i], l[(i+1)%len(l)]
		dx, dy := b.X-a.X, b.Y-a.Y
		t := clampF(((x-a.X)*dx+(y-a.Y)*dy)/math.Max(1e-12, dx*dx+dy*dy), 0, 1)
		best = math.Min(best, math.Hypot(a.X+t*dx-x, a.Y+t*dy-y))
	}
	return best
}

// tallest part on one side of the board, by reference
func (c *WirieCase) tallest(side string) string {
	best, ref := -math.MaxFloat64, "the tallest part"
	for _, p := range c.Parts {
		v := p.Box.Z1
		if side == "bottom" {
			v = -p.Box.Z0
		}
		if partSolid(p) && v > best {
			best, ref = v, p.Ref
		}
	}
	return ref
}

func (c *WirieCase) cornerR(o CaseOpts) float64 {
	if o.Radius >= 0 {
		return o.Radius
	}
	return c.Board.CornerRadius + o.Fit + o.Wall
}

// ---------------- case builder ----------------

// CaseInfo is what the editor shows next to the preview.
type CaseInfo struct {
	Lines    []string   `json:"lines"`
	Warnings []string   `json:"warnings"`
	Outer    [3]float64 `json:"outer"`
	LidStyle string     `json:"lidStyle"`
}

func newNode(name, typ string, p Params, pos, rot [3]float64, hole bool, role string) *Node {
	return &Node{Name: name, Type: typ, Params: p, Pos: pos, Rot: rot, Scale: [3]float64{1, 1, 1}, Hole: hole, Wirie: role}
}

var stdScrewLengths = []float64{4, 5, 6, 8, 10, 12, 14, 16, 18, 20, 22, 25, 30, 35, 40, 45, 50, 55, 60, 70, 80}

// wall rotation: local +Z → outward normal, local +Y → up
func wallRot(n [2]float64) [3]float64 {
	a := math.Atan2(n[1], n[0]) * 180 / math.Pi
	return [3]float64{90, 0, a + 90}
}

// BuildCase makes the editable node trees of the case: the base group, the
// lid group and loose extras. Everything is in the board frame.
func (c *WirieCase) BuildCase(p Params) ([]*Node, CaseInfo) {
	if c.isKeyboard() {
		return c.buildKeyboardCase(p)
	}
	o := caseOpts(c, p)
	d := c.dims(o)
	var info CaseInfo
	info.LidStyle = d.lid
	if d.lid != o.Lid {
		if o.Lid == "Slide-in" {
			info.Warnings = append(info.Warnings, "A slide-in lid needs a rectangular board, so this case uses a snap-fit lid.")
		} else {
			info.Warnings = append(info.Warnings, "This board has no mounting holes, so the lid snaps on instead of being screwed down.")
		}
	}
	bj := c.Compact()
	shared := func() Params {
		q := Params{"board": bj}
		for _, k := range []string{"wall", "floor", "fit", "standoff", "clear", "radius", "lidT", "lid", "csk"} {
			if v, ok := p[k]; ok {
				q[k] = v
			}
		}
		q["standoff"], q["clear"], q["lidT"], q["lid"] = o.Standoff, o.Clear, o.LidT, d.lid
		return q
	}
	f3 := func(x, y, z float64) [3]float64 { return [3]float64{round3(x), round3(y), round3(z)} }
	zero := [3]float64{}

	base := &Node{Name: "Case base — " + c.Project, Type: "group", Scale: [3]float64{1, 1, 1}, Wirie: "base", Color: "#64748b"}
	lid := &Node{Name: "Case lid — " + c.Project, Type: "group", Scale: [3]float64{1, 1, 1}, Wirie: "lid", Color: "#cbd5e1"}
	base.Children = append(base.Children, newNode("Tray", "wcase_tray", shared(), zero, zero, false, ""))
	lid.Children = append(lid.Children, newNode("Lid plate", "wcase_lid", shared(), zero, zero, false, ""))
	var extras []*Node

	// --- standoffs and screws ---
	m := d.m
	pilot := round3(m.D * 0.85)
	insert := m.InsertD > 0 && o.Fixing == "Heat-set insert"
	if o.clearFrom > 0 {
		info.Warnings = append(info.Warnings, fmt.Sprintf("The lid clearance was raised from %.1f to %.1f mm so the lid clears %s.", o.clearFrom, o.Clear, c.tallest("top")))
	}
	if o.standFrom > 0 {
		info.Warnings = append(info.Warnings, fmt.Sprintf("The standoffs were raised from %.1f to %.1f mm to clear %s underneath the board.", o.standFrom, o.Standoff, c.tallest("bottom")))
	}
	if o.raised {
		info.Warnings = append(info.Warnings, fmt.Sprintf("The standoffs were raised to %.1f mm so the %s heat-set inserts (%.1f mm long) fit.", o.Standoff, d.screw, m.InsertL))
	}
	holeBot := d.zFloorBot + 1.2 // keep a solid floor under blind holes
	for _, h := range c.Holes {
		od := round3(math.Max(h.D*2.2, bossOD(math.Max(pilot, h.D))))
		if insert {
			od = math.Max(od, round3(bossOD(m.InsertD)))
		}
		// keep the flared foot clear of the walls (a foot that just grazes a
		// curved wall makes a fragile sliver); a boss that is already in the wall
		// gets no foot
		flare := math.Min(1.5, o.Standoff*0.5)
		if gap := c.edgeDist(h.X, h.Y, o.Fit) - od/2; gap < flare+0.6 {
			flare = math.Max(0, gap-0.6)
			if flare < 0.3 {
				flare = 0
			}
		}
		base.Children = append(base.Children, newNode("Standoff "+h.Ref, "wcase_standoff", Params{"od": od, "h": round3(o.Standoff), "flare": round3(flare)}, f3(h.X, h.Y, d.zFloorTop), zero, false, ""))
		if insert {
			base.Children = append(base.Children, newNode("Insert hole "+h.Ref, "cylinder", Params{"d": m.InsertD, "h": round3(m.InsertL + 0.4)}, f3(h.X, h.Y, -m.InsertL-0.3), zero, true, ""))
		} else {
			base.Children = append(base.Children, newNode("Screw hole "+h.Ref, "cylinder", Params{"d": pilot, "h": round3(-holeBot + 0.2)}, f3(h.X, h.Y, holeBot), zero, true, ""))
		}
	}
	if len(c.Holes) == 0 {
		info.Warnings = append(info.Warnings, "No mounting holes: the board rests on the floor ledge-free. Add a little hot glue or foam pads, or add holes in Wirie.")
	}
	clear := m.Normal
	if d.lid == "Screw-down" {
		for _, h := range c.Holes {
			od := round3(math.Max(h.D*2.2, bossOD(clear)))
			lid.Children = append(lid.Children, newNode("Screw post "+h.Ref, "wcase_post", Params{"od": od, "id": clear, "h": round3(d.zTop - d.t + 0.01)}, f3(h.X, h.Y, d.t), zero, false, ""))
			lid.Children = append(lid.Children, newNode("Screw clearance "+h.Ref, "cylinder", Params{"d": clear, "h": round3(d.zLidTop - d.t + 2)}, f3(h.X, h.Y, d.t-1), zero, true, ""))
			if o.Csk && m.CskD > 0 {
				cd := m.CskD + 0.4
				depth := (cd - clear) / 2
				lid.Children = append(lid.Children, newNode("Countersink "+h.Ref, "cone", Params{"d1": clear, "d2": round3(cd + 2), "h": round3(depth + 1)}, f3(h.X, h.Y, d.zLidTop-depth), zero, true, ""))
			}
		}
		minL, maxL := d.zLidTop+1.2*m.D, d.zLidTop-holeBot
		if insert {
			minL, maxL = d.zLidTop+0.6*m.InsertL, d.zLidTop+m.InsertL+0.2
		}
		L := 0.0
		for _, s := range stdScrewLengths {
			if s >= minL && s <= maxL {
				L = s
				break
			}
		}
		if L == 0 {
			// settle for a slightly shorter screw if it still grips, and say how to fit the next size
			grip := d.zLidTop + 0.8*m.D
			if insert {
				grip = d.zLidTop + 0.4*m.InsertL
			}
			for i := len(stdScrewLengths) - 1; i >= 0; i-- {
				if s := stdScrewLengths[i]; s <= maxL && s >= grip {
					L = s
					if i+1 < len(stdScrewLengths) {
						info.Warnings = append(info.Warnings, fmt.Sprintf("Tip: raise the standoffs by %.1f mm to use %s × %.0f mm screws, which grip better.", stdScrewLengths[i+1]-maxL, d.screw, stdScrewLengths[i+1]))
					}
					break
				}
			}
		}
		if L == 0 {
			info.Warnings = append(info.Warnings, fmt.Sprintf("No standard screw length fits: you need about %s × %.0f mm. Change the standoff height or lid clearance.", d.screw, math.Ceil(minL)))
		} else {
			head := "pan / button head"
			if o.Csk {
				head = "countersunk"
			}
			info.Lines = append(info.Lines, fmt.Sprintf("Screws: %d × %s × %.0f mm %s — each goes through the lid, the board and into a standoff.", len(c.Holes), d.screw, L, head))
		}
	} else if len(c.Holes) > 0 {
		lo, hi, grip := d.t+1.2*m.D, d.t-holeBot, d.t+0.8*m.D
		if insert {
			lo, hi, grip = d.t+0.6*m.InsertL, d.t+m.InsertL+0.2, d.t+0.4*m.InsertL
		}
		L := 0.0
		for _, s := range stdScrewLengths {
			if s >= lo && s <= hi {
				L = s
				break
			}
		}
		for i := len(stdScrewLengths) - 1; L == 0 && i >= 0; i-- {
			if s := stdScrewLengths[i]; s <= hi && s >= grip {
				L = s
			}
		}
		if L > 0 {
			info.Lines = append(info.Lines, fmt.Sprintf("Board screws: %d × %s × %.0f mm.", len(c.Holes), d.screw, L))
		}
	}
	if insert {
		info.Lines = append(info.Lines, fmt.Sprintf("Heat-set inserts: %d × %s (hole Ø%.1f, %.1f mm deep).", len(c.Holes), d.screw, m.InsertD, m.InsertL))
	}

	// --- wall openings ---
	for _, op := range c.Openings {
		n := op.Normal
		inset := o.Fit + o.Wall/2
		cx, cy := op.X+n[0]*inset, op.Y+n[1]*inset
		w, h, zc := op.Width, op.Height, op.Z
		name := fmt.Sprintf("Opening %s (%s)", op.Ref, op.Type)
		// a connector hanging over the edge can't be lowered in past the wall:
		// open the slot to the top and let the lid close it
		dropIn := op.DistanceFromEdge < -0.05
		if dropIn {
			top := d.zWallTop + 1
			lo := zc - h/2
			h, zc = top-lo, (top+lo)/2
			name += " — open to the top so the board drops in"
		}
		circle := op.Type == "audio-3.5mm" || op.Type == "dc-jack" && w == h
		pr := Params{"w": round3(w), "h": round3(h), "din": round3(o.Wall/2 + 0.8), "dout": round3(o.Wall/2 + 0.01), "cham": round3(math.Min(0.6, o.Wall/2-0.1)), "r": round3(math.Min(1, math.Min(w, h)/4)), "circle": circle && !dropIn}
		base.Children = append(base.Children, newNode(name, "wcase_cutout", pr, f3(cx, cy, zc), wallRot(n), true, ""))
		if dropIn && d.lid != "Slide-in" {
			// a tab on the lid fills the slot down to just above the plug
			fillLo := op.Z + op.Height/2 + 0.3
			if fillLo < d.zTop-0.2 {
				tw := w - 0.4
				lid.Children = append(lid.Children, newNode("Slot filler "+op.Ref, "box", Params{"w": round3(tw), "d": round3(o.Wall - 0.2), "h": round3(d.zTop - fillLo + 0.01)}, f3(cx, cy, fillLo), [3]float64{0, 0, round3(math.Atan2(n[1], n[0])*180/math.Pi + 90)}, false, ""))
			}
		}
		if op.DistanceFromEdge > 1 {
			info.Warnings = append(info.Warnings, fmt.Sprintf("%s (%s) sits %.1f mm back from the board edge — the plug has to reach %.1f mm into the case.", op.Ref, op.Type, op.DistanceFromEdge, op.DistanceFromEdge+o.Fit))
		}
	}

	// --- lid and floor features ---
	type rect struct{ x0, y0, x1, y1 float64 }
	var busy []rect
	for _, h := range c.Holes {
		r := math.Max(h.D*1.1, bossOD(clear)/2) + 1
		busy = append(busy, rect{h.X - r, h.Y - r, h.X + r, h.Y + r})
	}
	for _, f := range c.LidFeatures {
		top := f.Side == "top"
		var z0, z1 float64
		if top {
			z0, z1 = d.zTop-1, d.zLidTop+1
		} else {
			z0, z1 = d.zFloorBot-1, d.zFloorTop+0.5
		}
		hw, hh := f.D/2, f.D/2
		if f.Shape == "rect" {
			hw, hh = f.W/2, f.H/2
		}
		if top {
			busy = append(busy, rect{f.X - hw, f.Y - hh, f.X + hw, f.Y + hh})
		}
		label := fmt.Sprintf("%s %s", capFirst(featureName(f.Type)), f.Ref)
		var hole *Node
		switch f.Type {
		case "sound-holes", "vent":
			pr := Params{"shape": f.Shape, "d": f.D, "w": f.W, "hh": f.H, "hole": 1.6, "pitch": 3, "h": round3(z1 - z0)}
			hole = newNode(label, "wcase_grille", pr, f3(f.X, f.Y, z0), zero, true, "")
		default:
			cham := 0.0
			if f.Type == "display-window" || f.Type == "sensor-window" {
				cham = math.Min(0.8, o.LidT*0.4)
			}
			w, h := f.D, f.D
			if f.Shape == "rect" {
				w, h = f.W, f.H
			}
			pr := Params{"w": round3(w), "h": round3(h), "din": round3(z1 - z0 - 1), "dout": 1, "cham": round3(cham), "r": 0.5, "circle": f.Shape != "rect"}
			// cut-out shape points along +Z: put its "surface" at the lid top (or floor bottom)
			if top {
				hole = newNode(label, "wcase_cutout", pr, f3(f.X, f.Y, d.zLidTop), zero, true, "")
			} else {
				hole = newNode(label, "wcase_cutout", pr, f3(f.X, f.Y, d.zFloorBot), [3]float64{180, 0, 0}, true, "")
			}
		}
		if top {
			lid.Children = append(lid.Children, hole)
		} else {
			base.Children = append(base.Children, hole)
		}
		if f.Approx {
			info.Warnings = append(info.Warnings, fmt.Sprintf("The %s for %s is Wirie's estimate (%s). Click %s on the board to adjust it.", featureName(f.Type), f.Ref, featureSize(f), f.Ref))
		}
		if !top {
			continue
		}
		// extras
		lidUnder := d.zTop
		switch {
		case f.Type == "button" && o.Caps && f.Shape == "circle":
			rod := round3(math.Max(1.5, f.D-0.6))
			bot := f.PartTop + 0.2
			length := d.zLidTop + 1.2 - bot
			collarTop := lidUnder - 0.4 - bot
			if collarTop > 1.5 {
				pr := Params{"d": rod, "len": round3(length), "collar": round3(f.D + 3), "cz": round3(collarTop), "dome": true}
				extras = append(extras, &Node{Name: "Button extender " + f.Ref, Type: "wcase_plunger", Params: pr, Pos: f3(f.X, f.Y, bot), Scale: [3]float64{1, 1, 1}, Wirie: "extra", Color: "#f8fafc"})
			} else {
				info.Lines = append(info.Lines, fmt.Sprintf("Button %s reaches the lid, so it needs no extender.", f.Ref))
			}
		case f.Shape == "circle" && ((f.Type == "light-pipe" && o.Pipes) || (f.Type == "led" && o.LedPipes)):
			rod := round3(math.Max(1.5, f.D-0.3))
			bot := f.PartTop + 0.3
			collarTop := lidUnder - 0.3 - bot
			if collarTop > 1.5 {
				pr := Params{"d": rod, "len": round3(d.zLidTop - bot), "collar": round3(f.D + 2.5), "cz": round3(collarTop), "dome": false}
				extras = append(extras, &Node{Name: "Light pipe " + f.Ref, Type: "wcase_plunger", Params: pr, Pos: f3(f.X, f.Y, bot), Scale: [3]float64{1, 1, 1}, Wirie: "extra", Color: "#e0f2fe"})
			}
		case f.Type == "display-window" && o.Bezel && f.Shape == "rect":
			pr := Params{"w": round3(f.W), "h": round3(f.H), "border": 4, "t": 1.6}
			extras = append(extras, &Node{Name: "Display bezel " + f.Ref, Type: "wcase_bezel", Params: pr, Pos: f3(f.X, f.Y, d.zLidTop), Scale: [3]float64{1, 1, 1}, Wirie: "extra", Color: "#0f172a"})
		}
	}

	// --- vent slots ---
	if o.Vents {
		n := c.addVents(base, o, d)
		if n == 0 {
			info.Warnings = append(info.Warnings, "There was no free wall space for vent slots.")
		}
	}

	// --- text and logo ---
	lx0, ly0, lx1, ly1 := -o.Fit+2, -o.Fit+2, c.Board.Width+o.Fit-2, c.Board.Height+o.Fit-2
	placeOnLid := func(w, h float64) (float64, float64, bool) {
		best, bx, by := math.Inf(1), 0.0, 0.0
		for y := ly0 + h/2; y <= ly1-h/2; y += 1 {
			for x := lx0 + w/2; x <= lx1-w/2; x += 1 {
				ok := true
				for _, r := range busy {
					if x+w/2+1.5 > r.x0 && x-w/2-1.5 < r.x1 && y+h/2+1.5 > r.y0 && y-h/2-1.5 < r.y1 {
						ok = false
						break
					}
				}
				if !ok {
					continue
				}
				score := math.Abs(x-c.Board.Width/2)*0.5 + (y - ly0) // front-centre first
				if score < best {
					best, bx, by = score, x, y
				}
			}
		}
		return bx, by, !math.IsInf(best, 1)
	}
	engrave := math.Min(0.6, o.LidT-0.8)
	if o.Logo != "" && o.Logo != "None" {
		s := clampF(o.TextSize*1.8, 6, 30)
		if x, y, ok := placeOnLid(s, s); ok {
			lid.Children = append(lid.Children, newNode("Logo", "wcase_logo", Params{"kind": o.Logo, "size": round3(s), "h": 2}, f3(x, y, d.zLidTop-engrave), zero, true, ""))
			busy = append(busy, rect{x - s/2, y - s/2, x + s/2, y + s/2})
		} else {
			info.Warnings = append(info.Warnings, "There's no free space on the lid for the logo.")
		}
	}
	if o.Text != "" {
		tp := Params{"text": o.Text, "font": o.Font, "size": o.TextSize, "h": 2, "align": "Center", "_q": 1.0}
		if tm, err := Generate("text", tp); err == nil && len(tm.Tris) > 0 {
			lo, hi := tm.Bounds()
			w, h := hi.X-lo.X, hi.Y-lo.Y
			if x, y, ok := placeOnLid(w, h); ok {
				delete(tp, "_q")
				px, py := x-(lo.X+hi.X)/2, y-(lo.Y+hi.Y)/2
				lid.Children = append(lid.Children, newNode("Lid text", "text", tp, f3(px, py, d.zLidTop-engrave), zero, true, ""))
			} else {
				info.Warnings = append(info.Warnings, "The lid text doesn't fit between the lid's holes — make it smaller or shorter.")
			}
		}
	}
	if d.lid == "Slide-in" {
		// finger grip at the open end
		w := o.Fit + o.Wall
		for i := 0; i < 3; i++ {
			x := c.Board.Width + w - 3 - float64(i)*2
			lid.Children = append(lid.Children, newNode("Grip line", "box", Params{"w": 0.8, "d": round3(math.Min(16, c.Board.Height*0.4)), "h": 1}, f3(x, c.Board.Height/2, d.zLidTop-0.15-0.5), zero, true, ""))
		}
	}

	// feet under the case
	var avoid []P2
	for _, h := range c.Holes {
		avoid = append(avoid, P2{h.X, h.Y})
	}
	lift := addFeet(base, d.outer, d.zFloorBot, o.Floor, p, avoid, "", boardFeetDefault, &info)
	if p.S("feet", boardFeetDefault) != "Printed feet" {
		lift = 0
	}

	ow := c.Board.Width + 2*(o.Fit+o.Wall)
	oh := c.Board.Height + 2*(o.Fit+o.Wall)
	info.Outer = [3]float64{round3(ow), round3(oh), round3(d.zLidTop - d.zFloorBot + lift)}
	if d.lid == "Slide-in" {
		info.Outer[2] = round3(d.zWallTop - d.zFloorBot)
	}
	info.Lines = append([]string{fmt.Sprintf("Case %.1f × %.1f × %.1f mm, %s lid.", info.Outer[0], info.Outer[1], info.Outer[2], strings.ToLower(d.lid))}, info.Lines...)
	switch d.lid {
	case "Snap-fit":
		info.Lines = append(info.Lines, "The lid's lip has small ramps that click into grooves in the walls.")
	case "Slide-in":
		info.Lines = append(info.Lines, "The lid slides in from the right-hand end along grooves in the walls.")
	}
	return append([]*Node{base, lid}, extras...), info
}

// vertical vent slots on the front and back walls, above the board, avoiding openings
func (c *WirieCase) addVents(base *Node, o CaseOpts, d caseDims) int {
	if !c.isRect() {
		return 0
	}
	z0 := d.t + 1.5
	z1 := d.zTop - lipDepth(o) - 1.5
	if z1-z0 < 3 {
		z0 = d.zFloorTop + 1.5
		z1 = d.zTop - lipDepth(o) - 1.5
	}
	if z1-z0 < 3 {
		return 0
	}
	count := 0
	W := c.Board.Width
	for _, face := range []string{"front", "back"} {
		n := faceNormal2(face)
		y := -o.Fit - o.Wall/2
		if face == "back" {
			y = c.Board.Height + o.Fit + o.Wall/2
		}
		for x := W * 0.3; x <= W*0.7+1e-6; x += 4 {
			blocked := false
			for _, op := range c.Openings {
				if op.Face == face && math.Abs(op.X-x) < op.Width/2+3 {
					blocked = true
				}
			}
			for _, h := range c.Holes {
				if math.Abs(h.X-x) < 6 && math.Abs(h.Y-(y-n[1]*o.Wall/2)) < 8 {
					blocked = true
				}
			}
			if blocked {
				continue
			}
			pr := Params{"w": 1.8, "h": round3(z1 - z0), "din": round3(o.Wall/2 + 0.5), "dout": round3(o.Wall/2 + 0.01), "cham": 0, "r": 0.3, "circle": false}
			base.Children = append(base.Children, newNode("Vent slot", "wcase_cutout", pr, [3]float64{round3(x), round3(y), round3((z0 + z1) / 2)}, wallRot(n), true, ""))
			count++
		}
	}
	return count
}

// ---------------- clash check ----------------

type WirieClash struct {
	Ref  string `json:"ref"`
	Part string `json:"part"`
	With string `json:"with"`
	ID   string `json:"id"`
}

// ClashCheck reports every case node whose geometry enters a part's box.
// Nodes are given in the board frame.
func (c *WirieCase) ClashCheck(nodes []*Node, q float64, toBoard ...Mat4) ([]WirieClash, error) {
	out := []WirieClash{}
	type pb struct {
		p      WiriePart
		lo, hi Vec3
	}
	var boxes []pb
	for _, p := range c.Parts {
		if !partSolid(p) {
			continue
		}
		e := 0.06
		boxes = append(boxes, pb{p, V(p.Box.X0+e, p.Box.Y0+e, p.Box.Z0+e), V(p.Box.X1-e, p.Box.Y1-e, p.Box.Z1-e)})
	}
	for _, n := range nodes {
		if n.Hidden || n.Hole || GroupIsHole(n) {
			continue
		}
		m, err := BuildWorld(n, q)
		if err != nil {
			return nil, err
		}
		if len(toBoard) > 0 {
			m = m.Transformed(toBoard[0])
		}
		lo, hi := m.Bounds()
		for _, b := range boxes {
			if lo.X > b.hi.X || hi.X < b.lo.X || lo.Y > b.hi.Y || hi.Y < b.lo.Y || lo.Z > b.hi.Z || hi.Z < b.lo.Z {
				continue
			}
			if meshEntersBox(m, b.lo, b.hi) {
				out = append(out, WirieClash{Ref: b.p.Ref, Part: orStr(b.p.Value, b.p.Name), With: n.Name, ID: n.ID})
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Ref < out[j].Ref })
	return out, nil
}

// meshEntersBox: any triangle crosses the box, or the box sits inside the mesh.
func meshEntersBox(m *Mesh, lo, hi Vec3) bool {
	c := lo.Add(hi).Mul(0.5)
	h := hi.Sub(lo).Mul(0.5)
	for _, t := range m.Tris {
		if triBoxOverlap(t, c, h) {
			return true
		}
	}
	// box fully inside a solid: ray-cast its centre
	return pointInMesh(m, c)
}

func pointInMesh(m *Mesh, p Vec3) bool {
	dir := V(0.5773, 0.5774, 0.5772)
	n := 0
	for _, t := range m.Tris {
		if _, ok := rayTri(p, dir, t); ok {
			n++
		}
	}
	return n%2 == 1
}

func rayTri(o, d Vec3, t Tri) (float64, bool) {
	e1, e2 := t[1].Sub(t[0]), t[2].Sub(t[0])
	pv := d.Cross(e2)
	det := e1.Dot(pv)
	if math.Abs(det) < 1e-12 {
		return 0, false
	}
	inv := 1 / det
	tv := o.Sub(t[0])
	u := tv.Dot(pv) * inv
	if u < 0 || u > 1 {
		return 0, false
	}
	qv := tv.Cross(e1)
	v := d.Dot(qv) * inv
	if v < 0 || u+v > 1 {
		return 0, false
	}
	s := e2.Dot(qv) * inv
	return s, s > 1e-9
}

// triBoxOverlap is the separating-axis test of Akenine-Möller.
func triBoxOverlap(t Tri, c, h Vec3) bool {
	v0, v1, v2 := t[0].Sub(c), t[1].Sub(c), t[2].Sub(c)
	ax := func(v Vec3, i int) float64 { return [3]float64{v.X, v.Y, v.Z}[i] }
	hv := [3]float64{h.X, h.Y, h.Z}
	for i := 0; i < 3; i++ {
		a, b, cc := ax(v0, i), ax(v1, i), ax(v2, i)
		if math.Min(a, math.Min(b, cc)) > hv[i] || math.Max(a, math.Max(b, cc)) < -hv[i] {
			return false
		}
	}
	e := []Vec3{v1.Sub(v0), v2.Sub(v1), v0.Sub(v2)}
	axes := []Vec3{V(1, 0, 0), V(0, 1, 0), V(0, 0, 1)}
	for _, ed := range e {
		for _, a := range axes {
			l := a.Cross(ed)
			if l.Dot(l) < 1e-18 {
				continue
			}
			p0, p1, p2 := v0.Dot(l), v1.Dot(l), v2.Dot(l)
			r := h.X*math.Abs(l.X) + h.Y*math.Abs(l.Y) + h.Z*math.Abs(l.Z)
			if math.Min(p0, math.Min(p1, p2)) > r || math.Max(p0, math.Max(p1, p2)) < -r {
				return false
			}
		}
	}
	n := e[0].Cross(e[1])
	if n.Dot(n) < 1e-18 {
		return false
	}
	dd := n.Dot(v0)
	r := h.X*math.Abs(n.X) + h.Y*math.Abs(n.Y) + h.Z*math.Abs(n.Z)
	return math.Abs(dd) <= r
}

// InverseRigid inverts a rotation + translation (+ uniform scale) transform.
func (m Mat4) InverseRigid() Mat4 {
	sx := V(m[0], m[4], m[8]).Len()
	if sx == 0 {
		sx = 1
	}
	s2 := 1 / (sx * sx)
	r := Mat4{m[0] * s2, m[4] * s2, m[8] * s2, 0, m[1] * s2, m[5] * s2, m[9] * s2, 0, m[2] * s2, m[6] * s2, m[10] * s2, 0, 0, 0, 0, 1}
	t := r.Apply(V(m[3], m[7], m[11]))
	r[3], r[7], r[11] = -t.X, -t.Y, -t.Z
	return r
}
