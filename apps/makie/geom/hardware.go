package geom

import (
	"math"
	"sort"
	"strings"
)

// Metric ISO coarse thread hardware data (mm).
type metric struct {
	D, Pitch              float64
	SocketD, SocketH, Key float64 // ISO 4762 socket head cap
	HexAF, HexH           float64 // ISO 4017 hex head
	NutAF, NutH           float64 // ISO 4032 nut
	WashOD, WashT         float64 // ISO 7089 washer
	CskD                  float64 // ISO 10642 countersunk head diameter
	ButtonD, ButtonH      float64 // ISO 7380 button head
	Close, Normal, Loose  float64 // ISO 273 clearance holes
	InsertD, InsertL      float64 // typical heat-set insert hole
	SqAF, SqH             float64 // square nut
}

var metrics = map[string]metric{
	"M2":   {2, 0.4, 3.8, 2, 1.5, 4, 1.4, 4, 1.6, 5, 0.3, 3.8, 3.5, 1.3, 2.2, 2.4, 2.6, 3.2, 4, 4, 1.2},
	"M2.5": {2.5, 0.45, 4.5, 2.5, 2, 5, 1.7, 5, 2, 6, 0.5, 4.7, 4.7, 1.5, 2.7, 2.9, 3.1, 3.6, 5, 5, 1.6},
	"M3":   {3, 0.5, 5.5, 3, 2.5, 5.5, 2, 5.5, 2.4, 7, 0.5, 6, 5.7, 1.65, 3.2, 3.4, 3.6, 4.0, 5.7, 5.5, 1.8},
	"M4":   {4, 0.7, 7, 4, 3, 7, 2.8, 7, 3.2, 9, 0.8, 8, 7.6, 2.2, 4.3, 4.5, 4.8, 5.6, 8.1, 7, 2.2},
	"M5":   {5, 0.8, 8.5, 5, 4, 8, 3.5, 8, 4.7, 10, 1, 10, 9.5, 2.75, 5.3, 5.5, 5.8, 6.4, 9.5, 8, 2.7},
	"M6":   {6, 1.0, 10, 6, 5, 10, 4, 10, 5.2, 12, 1.6, 12, 10.5, 3.3, 6.4, 6.6, 7, 8.0, 12.7, 10, 3.2},
	"M8":   {8, 1.25, 13, 8, 6, 13, 5.3, 13, 6.8, 16, 1.6, 16, 14, 4.4, 8.4, 9, 10, 10, 12.7, 13, 4},
	"M10":  {10, 1.5, 16, 10, 8, 16, 6.4, 16, 8.4, 20, 2, 20, 17.5, 5.5, 10.5, 11, 12, 12.5, 15, 17, 5},
	"M12":  {12, 1.75, 18, 12, 10, 18, 7.5, 18, 10.8, 24, 2.5, 24, 21, 6.6, 13, 13.5, 14.5, 15, 18, 19, 6},
}

func metricSizes() []string {
	ks := make([]string, 0, len(metrics))
	for k := range metrics {
		ks = append(ks, k)
	}
	sort.Slice(ks, func(i, j int) bool { return metrics[ks[i]].D < metrics[ks[j]].D })
	return ks
}

func getMetric(p Params) metric {
	if m, ok := metrics[p.S("size", "M3")]; ok {
		return m
	}
	return metrics["M3"]
}

// ThreadedCylinder builds an ISO-style external thread between z0 and z1.
// majorD is the crest diameter. Ends are chamfered so it prints and looks right.
func ThreadedCylinder(majorD, pitch, z0, z1 float64, q float64, chamferEnds bool) *Mesh {
	H := 0.866025 * pitch
	depth := 5.0 / 8.0 * H
	rMaj := majorD / 2
	rMin := rMaj - depth
	segs := int(math.Max(24, math.Min(64, majorD*math.Pi/0.6)) * q)
	if segs < 16 {
		segs = 16
	}
	zsteps := int(math.Max(4, 6*q))
	L := z1 - z0
	nz := int(math.Ceil(L/pitch*float64(zsteps))) + 1
	prof := func(t float64) float64 {
		t -= math.Floor(t)
		switch {
		case t < 0.375:
			return t / 0.375
		case t < 0.5:
			return 1
		case t < 0.875:
			return 1 - (t-0.5)/0.375
		default:
			return 0
		}
	}
	var rings [][]Vec3
	for k := 0; k <= nz; k++ {
		z := z0 + L*float64(k)/float64(nz)
		var ring []Vec3
		for i := 0; i < segs; i++ {
			a := 2 * math.Pi * float64(i) / float64(segs)
			t := (z-z0)/pitch - a/(2*math.Pi)
			r := rMin + depth*prof(t)
			if chamferEnds {
				// 45° chamfer at both ends down to the minor diameter
				lim := rMin + math.Min(z-z0, z1-z)
				lim = math.Max(lim, rMin-depth*0.5)
				if r > lim {
					r = lim
				}
			}
			s, c := math.Sincos(a)
			ring = append(ring, V(r*c, r*s, z))
		}
		rings = append(rings, ring)
	}
	return Loft(rings, true, true)
}

func hexPrism(af, z0, z1 float64) *Mesh {
	r := af / math.Sqrt(3)
	return Extrude(Shape{Outer: NGon(r, 6, 0)}, z1-z0, 0, 1, 1).Transformed(Translate(0, 0, z0))
}

func init() {
	sizes := metricSizes()

	// ---------- BOLTS ----------
	register(&ShapeDef{Type: "bolt", Name: "Bolt / screw", Category: "Hardware", Fixed: true, Color: "#94a3b8",
		Desc: "Metric bolt (ISO). Origin is under the head, so placed on a surface it goes straight in.",
		Tags: "screw bolt socket cap hex countersunk button machine",
		Params: []Param{
			sel("size", "Size", "M3", sizes...),
			num("length", "Length", 12, 2, 300, 1, "mm"),
			sel("head", "Head", "Socket cap", "Socket cap", "Hex", "Button", "Countersunk", "Pan", "None (grub)"),
			boolp("threads", "Show real threads", true).help("Real threads look great but are heavy. Turn off for faster editing."),
			sel("tStd", "Pitch", "Metric coarse", "Metric coarse", "Metric fine").show("threads=true"),
			sel("tHand", "Hand", "Right-hand", "Right-hand", "Left-hand").show("threads=true"),
		},
		Gen: func(p Params) (*Mesh, error) {
			m := getMetric(p)
			L := p.F("length", 12)
			q := p.Q()
			segs := p.Segs(m.SocketD)
			var shank *Mesh
			var info string
			if p.B("threads", true) {
				sp := ThreadSpec{Major: m.D, Pitch: hwPitch(p, m), Profile: "ISO metric 60°", Starts: 1, Left: p.S("tHand", "") == "Left-hand", Chamfer: true}
				shank = Thread(sp, -L, 0.01, q)
				info = ThreadInfo(sp, m.D, false)
			} else {
				c := m.D * 0.08
				shank = RevolveProfile([]P2{{0, -L}, {m.D/2 - c, -L}, {m.D / 2, -L + c}, {m.D / 2, 0.01}, {0, 0.01}}, segs)
			}
			head := p.S("head", "Socket cap")
			var hm *Mesh
			switch head {
			case "Hex":
				hm = hexPrism(m.HexAF, 0, m.HexH)
			case "Button":
				r, h := m.ButtonD/2, m.ButtonH
				var pts []P2
				pts = append(pts, P2{0, 0}, P2{r, 0})
				for i := 1; i <= 8; i++ {
					a := math.Pi / 2 * float64(i) / 8
					pts = append(pts, P2{r * math.Cos(a), h * math.Sin(a)})
				}
				pts[len(pts)-1].X = 0
				hm = RevolveProfile(pts, segs)
				hm = Subtract(hm, hexPrism(m.Key, h*0.35, h+1))
			case "Countersunk":
				r := m.CskD / 2
				hh := (r - m.D/2) // 90° head
				hm = RevolveProfile([]P2{{0, -hh}, {m.D / 2, -hh}, {r, 0}, {0, 0}}, segs)
				hm = Subtract(hm, hexPrism(m.Key, -hh*0.6, 1))
				// countersunk origin sits flush with the surface: shank starts under the cone
				shank = shank.Transformed(Translate(0, 0, 0))
			case "Pan":
				r, h := m.ButtonD/2, m.ButtonH*1.2
				hm = RevolveProfile([]P2{{0, 0}, {r, 0}, {r, h * 0.7}, {r * 0.85, h}, {0, h}}, segs)
				// cross slot
				hm = Subtract(hm, boxAt(V(-r, -m.D*0.12, h*0.45), V(r, m.D*0.12, h+1)))
				hm = Subtract(hm, boxAt(V(-m.D*0.12, -r, h*0.45), V(m.D*0.12, r, h+1)))
			case "None (grub)":
				hm = nil
				shank = Subtract(shank, hexPrism(m.Key*0.7, -m.D*0.6, 0.5))
			default: // socket cap
				r, h := m.SocketD/2, m.SocketH
				c := h * 0.08
				hm = RevolveProfile([]P2{{0, 0}, {r, 0}, {r, h - c}, {r - c, h}, {0, h}}, segs)
				hm = Subtract(hm, hexPrism(m.Key, h*0.4, h+1))
			}
			if hm == nil {
				shank.Info = info
				return shank, nil
			}
			res := Union(hm, shank)
			res.Info = info
			return res, nil
		}})

	register(&ShapeDef{Type: "nut", Name: "Hex nut", Category: "Hardware", Fixed: true, Color: "#94a3b8",
		Desc: "Metric hex nut (ISO 4032).", Tags: "nut hex",
		Params: []Param{
			sel("size", "Size", "M3", sizes...),
			sel("kind", "Type", "Regular", "Regular", "Thin (jam)", "Nyloc"),
			boolp("threads", "Show real threads", true),
			sel("tStd", "Pitch", "Metric coarse", "Metric coarse", "Metric fine").show("threads=true"),
			sel("tHand", "Hand", "Right-hand", "Right-hand", "Left-hand").show("threads=true"),
		},
		Gen: func(p Params) (*Mesh, error) {
			m := getMetric(p)
			h := m.NutH
			if p.S("kind", "") == "Thin (jam)" {
				h *= 0.55
			}
			if p.S("kind", "") == "Nyloc" {
				h *= 1.35
			}
			body := hexPrism(m.NutAF, 0, h)
			// chamfer corners like a real nut
			r := m.NutAF/math.Sqrt(3) + 0.05
			a := m.NutAF / 2 * 0.97
			e := math.Min((r-a)*0.577, h/2-0.02)
			cone := RevolveProfile([]P2{{0, -0.01}, {a, -0.01}, {r, e}, {r, h - e}, {a, h + 0.01}, {0, h + 0.01}}, p.Segs(m.NutAF*1.2))
			body = Intersect(body, cone)
			var hole *Mesh
			info := ""
			if p.B("threads", true) {
				sp := ThreadSpec{Major: m.D + 0.05, Pitch: hwPitch(p, m), Profile: "ISO metric 60°", Starts: 1, Left: p.S("tHand", "") == "Left-hand"}
				hole = Thread(sp, -0.5, h+0.5, p.Q())
				info = ThreadInfo(sp, m.D, true)
			} else {
				hole = cylAt(0, 0, m.D, -1, h+1, p.Segs(m.D))
			}
			res := Subtract(body, hole)
			res.Info = info
			return res, nil
		}})

	register(&ShapeDef{Type: "sqnut", Name: "Square nut", Category: "Hardware", Fixed: true, Color: "#94a3b8",
		Desc: "Metric square nut — slides into slots.", Tags: "square nut",
		Params: []Param{sel("size", "Size", "M3", sizes...)},
		Gen: func(p Params) (*Mesh, error) {
			m := getMetric(p)
			b := boxAt(V(-m.SqAF/2, -m.SqAF/2, 0), V(m.SqAF/2, m.SqAF/2, m.SqH))
			return Subtract(b, cylAt(0, 0, m.D, -1, m.SqH+1, p.Segs(m.D))), nil
		}})

	register(&ShapeDef{Type: "washer", Name: "Washer", Category: "Hardware", Fixed: true, Color: "#94a3b8",
		Desc: "Flat washer (ISO 7089).", Tags: "washer spacer",
		Params: []Param{sel("size", "Size", "M3", sizes...)},
		Gen: func(p Params) (*Mesh, error) {
			m := getMetric(p)
			return Revolve(Shape{Outer: Loop{{m.Normal / 2, 0}, {m.WashOD / 2, 0}, {m.WashOD / 2, m.WashT}, {m.Normal / 2, m.WashT}}}, 360, p.Segs(m.WashOD)), nil
		}})

	register(&ShapeDef{Type: "rod", Name: "Threaded rod", Category: "Hardware", Fixed: true, Color: "#94a3b8",
		Desc: "Metric threaded rod / stud.", Tags: "threaded rod stud thread",
		Params: []Param{sel("size", "Size", "M5", sizes...), num("length", "Length", 30, 2, 1000, 1, "mm"),
			sel("tStd", "Pitch", "Metric coarse", "Metric coarse", "Metric fine"), sel("tHand", "Hand", "Right-hand", "Right-hand", "Left-hand")},
		Gen: func(p Params) (*Mesh, error) {
			m := getMetric(p)
			sp := ThreadSpec{Major: m.D, Pitch: hwPitch(p, m), Profile: "ISO metric 60°", Starts: 1, Left: p.S("tHand", "") == "Left-hand", Chamfer: true}
			r := Thread(sp, 0, p.F("length", 30), p.Q())
			r.Info = ThreadInfo(sp, m.D, false)
			return r, nil
		}})

	register(&ShapeDef{Type: "standoff", Name: "Standoff / spacer", Category: "Hardware", Color: "#d97706",
		Desc: "Printable hex or round standoff with a hole for a screw or heat-set insert.", Tags: "standoff spacer pcb post boss",
		Params: []Param{
			sel("size", "Screw size", "M3", sizes...),
			num("h", "Height", 8, 1, 200, 0.5, "mm"),
			sel("shape", "Outside", "Round", "Round", "Hex"),
			num("wall", "Wall", 2, 0.8, 20, 0.2, "mm"),
			sel("hole", "Hole type", "Heat-set insert", "Heat-set insert", "Self-tapping", "Clearance"),
		},
		Gen: func(p Params) (*Mesh, error) {
			m := getMetric(p)
			h := p.F("h", 8)
			hd := m.D * 0.85
			switch p.S("hole", "") {
			case "Heat-set insert":
				hd = m.InsertD
			case "Clearance":
				hd = m.Normal
			}
			od := hd + 2*p.F("wall", 2)
			var body *Mesh
			if p.S("shape", "") == "Hex" {
				body = hexPrism(od, 0, h)
			} else {
				body = cylAt(0, 0, od, 0, h, p.Segs(od))
			}
			return Subtract(body, cylAt(0, 0, hd, -1, h+1, p.Segs(hd))), nil
		}})

	// ---------- HOLES (set as holes by default; origin = the surface you place them on) ----------
	register(&ShapeDef{Type: "screwhole", Name: "Screw hole", Category: "Holes", Hole: true, Fixed: true, Color: "#64748b",
		Desc: "Clearance hole for a metric screw with optional counterbore or countersink. Click a face to drill it.",
		Tags: "hole clearance counterbore countersink screw bolt drill",
		Params: []Param{
			sel("size", "Screw size", "M3", sizes...),
			sel("fit", "Fit", "Normal", "Close", "Normal", "Loose").help("Close = tight, Loose = easy fit. Normal works for most printers."),
			num("depth", "Depth", 10, 0.5, 500, 0.5, "mm"),
			sel("head", "Head recess", "None", "None", "Counterbore", "Countersink"),
			num("cbDepth", "Counterbore depth", 3, 0.2, 100, 0.2, "mm").show("head=Counterbore"),
			num("tol", "Extra tolerance", 0.2, 0, 2, 0.05, "mm").help("Added to the hole diameter to compensate for printing shrinkage."),
		},
		Gen: func(p Params) (*Mesh, error) {
			m := getMetric(p)
			d := m.Normal
			switch p.S("fit", "") {
			case "Close":
				d = m.Close
			case "Loose":
				d = m.Loose
			}
			tol := p.F("tol", 0.2)
			d += tol
			depth := p.F("depth", 10)
			over := 0.6
			segs := p.Segs(m.SocketD + 1)
			switch p.S("head", "None") {
			case "Counterbore":
				hd := m.SocketD + 0.6 + tol
				cb := math.Min(p.F("cbDepth", m.SocketH), depth)
				return RevolveProfile([]P2{{0, -depth}, {d / 2, -depth}, {d / 2, -cb}, {hd / 2, -cb}, {hd / 2, over}, {0, over}}, segs), nil
			case "Countersink":
				hd := m.CskD + 0.4 + tol
				ch := (hd - d) / 2
				return RevolveProfile([]P2{{0, -depth}, {d / 2, -depth}, {d / 2, -ch}, {hd / 2, 0}, {hd / 2, over}, {0, over}}, segs), nil
			}
			return RevolveProfile([]P2{{0, -depth}, {d / 2, -depth}, {d / 2, over}, {0, over}}, p.Segs(d)), nil
		}})

	register(&ShapeDef{Type: "pilothole", Name: "Self-tapping hole", Category: "Holes", Hole: true, Fixed: true, Color: "#64748b",
		Desc:   "Undersized pilot hole so a screw cuts its own thread into the plastic.",
		Tags:   "pilot tap hole self tapping screw thread",
		Params: []Param{sel("size", "Screw size", "M3", sizes...), num("depth", "Depth", 8, 0.5, 500, 0.5, "mm"), num("ratio", "Diameter ratio", 0.85, 0.6, 1, 0.01, "×")},
		Gen: func(p Params) (*Mesh, error) {
			m := getMetric(p)
			d := m.D * p.F("ratio", 0.85)
			dep := p.F("depth", 8)
			return RevolveProfile([]P2{{0, -dep - d*0.3}, {d / 2, -dep}, {d / 2, -0.3}, {d/2 + 0.4, 0}, {d/2 + 0.4, 0.6}, {0, 0.6}}, p.Segs(d)), nil
		}})

	register(&ShapeDef{Type: "threadhole", Name: "Threaded hole", Category: "Holes", Hole: true, Fixed: true, Color: "#64748b",
		Desc: "Cuts a real internal thread (best for M5 and larger, or any custom size and profile). A Cylinder with Thread on and the same diameter screws into it.",
		Tags: "threaded hole thread internal tapped nut female",
		Params: append([]Param{
			sel("size", "Size", "M6", append(append([]string{}, sizes...), "Custom")...),
			num("d", "Nominal diameter", 10, 1, 500, 0.5, "mm").show("size=Custom"),
			num("depth", "Depth", 10, 1, 500, 0.5, "mm"),
		}, threadParams("", "Metric coarse")...),
		Gen: func(p Params) (*Mesh, error) {
			nom := p.F("d", 10)
			if mt, ok := metrics[p.S("size", "M6")]; ok {
				nom = mt.D
			}
			dep := p.F("depth", 10)
			sp := specFrom(p, nom, true)
			t := Thread(sp, -dep, 0.6, p.Q())
			minor := sp.Major - 2*ThreadDepth(sp.Profile, sp.Pitch)
			lead := RevolveProfile([]P2{{0, -0.01}, {minor/2 - 0.1, -0.01}, {sp.Major/2 + 0.3, 0}, {sp.Major/2 + 0.3, 0.6}, {0, 0.6}}, p.Segs(nom))
			m := Union(t, lead)
			m.Info = ThreadInfo(sp, nom, true)
			return m, nil
		}})

	register(&ShapeDef{Type: "nuttrap", Name: "Nut trap", Category: "Holes", Hole: true, Fixed: true, Color: "#64748b",
		Desc: "Hexagonal pocket that holds a nut, with an optional bolt hole through it. Or a slot to slide a nut in from the side.",
		Tags: "nut trap hex pocket captive slot",
		Params: []Param{
			sel("size", "Size", "M3", sizes...),
			sel("kind", "Style", "Pocket", "Pocket", "Side slot"),
			num("nutDepth", "Pocket depth", 2.6, 0.5, 50, 0.1, "mm"),
			num("boltDepth", "Bolt hole depth", 10, 0, 500, 0.5, "mm"),
			num("slotLen", "Slot length", 10, 1, 200, 0.5, "mm").show("kind=Side slot"),
			num("tol", "Tolerance", 0.25, 0, 2, 0.05, "mm"),
		},
		Gen: func(p Params) (*Mesh, error) {
			m := getMetric(p)
			tol := p.F("tol", 0.25)
			nd := p.F("nutDepth", m.NutH+0.2)
			af := m.NutAF + 2*tol
			trap := hexPrism(af, -nd, 0.6)
			if p.S("kind", "") == "Side slot" {
				sl := p.F("slotLen", 10)
				trap = Union(trap, boxAt(V(0, -af/2, -nd), V(sl, af/2, 0.6)))
			}
			if bd := p.F("boltDepth", 10); bd > 0 {
				trap = Union(trap, cylAt(0, 0, m.Normal+tol, -nd-bd, 0.6, p.Segs(m.Normal)))
			}
			return trap, nil
		}})

	register(&ShapeDef{Type: "inserthole", Name: "Heat-set insert hole", Category: "Holes", Hole: true, Fixed: true, Color: "#64748b",
		Desc:   "Correctly sized hole for brass heat-set inserts, with a small lead-in.",
		Tags:   "heat set insert brass threaded insert",
		Params: []Param{sel("size", "Insert size", "M3", "M2", "M2.5", "M3", "M4", "M5", "M6", "M8"), num("extra", "Extra depth", 1, 0, 20, 0.5, "mm")},
		Gen: func(p Params) (*Mesh, error) {
			m := getMetric(p)
			d, l := m.InsertD, m.InsertL+p.F("extra", 1)
			return RevolveProfile([]P2{{0, -l}, {d / 2, -l}, {d / 2, -0.4}, {d/2 + 0.4, 0}, {d/2 + 0.4, 0.6}, {0, 0.6}}, p.Segs(d)), nil
		}})

	register(&ShapeDef{Type: "magnethole", Name: "Magnet pocket", Category: "Holes", Hole: true, Fixed: true, Color: "#64748b",
		Desc: "Round pocket for disc magnets (e.g. 6×3, 8×2, 10×3).", Tags: "magnet pocket disc",
		Params: []Param{num("d", "Magnet diameter", 6, 1, 100, 0.5, "mm"), num("h", "Magnet thickness", 3, 0.5, 50, 0.5, "mm"), num("tol", "Tolerance", 0.2, 0, 1, 0.05, "mm")},
		Gen: func(p Params) (*Mesh, error) {
			d := p.F("d", 6) + p.F("tol", 0.2)
			h := p.F("h", 3) + 0.2
			return cylAt(0, 0, d, -h, 0.6, p.Segs(d)), nil
		}})

	bearings := map[string][3]float64{"623 (3×10×4)": {3, 10, 4}, "625 (5×16×5)": {5, 16, 5}, "608 (8×22×7)": {8, 22, 7}, "688 (8×16×5)": {8, 16, 5}, "6001 (12×28×8)": {12, 28, 8}, "6200 (10×30×9)": {10, 30, 9}, "MR105 (5×10×4)": {5, 10, 4}}
	bnames := []string{"623 (3×10×4)", "MR105 (5×10×4)", "625 (5×16×5)", "688 (8×16×5)", "608 (8×22×7)", "6001 (12×28×8)", "6200 (10×30×9)"}
	register(&ShapeDef{Type: "bearinghole", Name: "Bearing seat", Category: "Holes", Hole: true, Fixed: true, Color: "#64748b",
		Desc: "Press-fit pocket for a ball bearing, with a clearance hole for the shaft.", Tags: "bearing seat 608 press fit",
		Params: []Param{sel("bearing", "Bearing", "608 (8×22×7)", bnames...), num("tol", "Tolerance", 0.1, -0.3, 1, 0.05, "mm"), num("through", "Shaft hole depth", 5, 0, 200, 0.5, "mm")},
		Gen: func(p Params) (*Mesh, error) {
			b := bearings[p.S("bearing", "608 (8×22×7)")]
			if b[1] == 0 {
				b = bearings["608 (8×22×7)"]
			}
			od := b[1] + p.F("tol", 0.1)
			m := cylAt(0, 0, od, -b[2], 0.6, p.Segs(od))
			if t := p.F("through", 5); t > 0 {
				m = Union(m, cylAt(0, 0, b[0]+(od-b[0])*0.35, -b[2]-t, -b[2]+0.01, p.Segs(od)))
			}
			return m, nil
		}})

	register(&ShapeDef{Type: "slot", Name: "Slot hole", Category: "Holes", Hole: true, Color: "#64748b",
		Desc: "Elongated hole for adjustable mounting.", Tags: "slot oblong adjust",
		Axes:   map[string]string{"x": "len", "z": "depth"},
		Params: []Param{num("w", "Width", 3.4, 0.5, 100, 0.1, "mm"), num("len", "Length", 12, 1, 500, 0.5, "mm"), num("depth", "Depth", 10, 0.5, 500, 0.5, "mm")},
		Gen: func(p Params) (*Mesh, error) {
			w, l, dep := p.F("w", 3.4), p.F("len", 12), p.F("depth", 10)
			r := w / 2
			l = math.Max(l, w+0.01)
			var lp Loop
			n := p.Segs(w) / 2
			for i := 0; i <= n; i++ {
				a := -math.Pi/2 + math.Pi*float64(i)/float64(n)
				lp = append(lp, P2{l/2 - r + r*math.Cos(a), r * math.Sin(a)})
			}
			for i := 0; i <= n; i++ {
				a := math.Pi/2 + math.Pi*float64(i)/float64(n)
				lp = append(lp, P2{-l/2 + r + r*math.Cos(a), r * math.Sin(a)})
			}
			return Extrude(Shape{Outer: lp}, dep+0.6, 0, 1, 1).Transformed(Translate(0, 0, -dep)), nil
		}})

	register(&ShapeDef{Type: "ziptie", Name: "Zip-tie channel", Category: "Holes", Hole: true, Color: "#64748b",
		Desc: "A U-shaped channel to thread a cable tie through.", Tags: "zip tie cable channel",
		Params: []Param{num("w", "Tie width", 4, 1, 20, 0.5, "mm"), num("t", "Tie thickness", 1.6, 0.5, 5, 0.1, "mm"), num("span", "Span", 10, 3, 200, 1, "mm"), num("depth", "Depth", 4, 1, 100, 0.5, "mm")},
		Gen: func(p Params) (*Mesh, error) {
			w, t, s, dep := p.F("w", 4)+0.4, p.F("t", 1.6)+0.4, p.F("span", 10), p.F("depth", 4)
			a := boxAt(V(-s/2-t, -w/2, -dep-t), V(-s/2, w/2, 0.6))
			b := boxAt(V(s/2, -w/2, -dep-t), V(s/2+t, w/2, 0.6))
			c := boxAt(V(-s/2-t, -w/2, -dep-t), V(s/2+t, w/2, -dep))
			return UnionAll([]*Mesh{a, b, c}), nil
		}})
}

func containsAny(s string, subs ...string) bool {
	for _, x := range subs {
		if strings.Contains(s, x) {
			return true
		}
	}
	return false
}

func hwPitch(p Params, m metric) float64 {
	if p.S("tStd", "") == "Metric fine" {
		return nearestPitch(finePitch, m.D)
	}
	return m.Pitch
}
