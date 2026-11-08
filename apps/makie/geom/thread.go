package geom

import (
	"fmt"
	"math"
)

// ThreadSpec fully describes a helical thread.
type ThreadSpec struct {
	Major   float64 // crest (outer) diameter of the thread surface generated
	Pitch   float64 // distance between neighbouring crests
	Profile string  // "ISO metric 60°", "Trapezoidal 30°", "Square", "Buttress"
	Starts  int
	Left    bool
	Chamfer bool
	Axial   float64 // flank clearance in mm along the axis: + widens the crest (holes), - narrows it (screws)
}

var threadProfiles = []string{"ISO metric 60°", "Trapezoidal 30°", "Square", "Buttress"}

// profile fractions: rise, crest, fall, root (sum 1) and depth as a fraction of pitch
func profileShape(name string) (rise, crest, fall float64, depth float64) {
	switch name {
	case "Trapezoidal 30°":
		return 0.134, 0.366, 0.134, 0.5
	case "Square":
		return 0.05, 0.45, 0.05, 0.5 // slight draft so it prints cleanly
	case "Buttress":
		return 0.07, 0.2, 0.53, 0.6
	}
	return 0.375, 0.125, 0.375, 5.0 / 8.0 * 0.866025
}

// ThreadDepth returns the radial depth of the profile.
func ThreadDepth(profile string, pitch float64) float64 {
	_, _, _, d := profileShape(profile)
	return d * pitch
}

// Thread builds a closed threaded cylinder between z0 and z1.
func Thread(s ThreadSpec, z0, z1, q float64) *Mesh {
	rise, crest, fall, dfrac := profileShape(s.Profile)
	if s.Pitch <= 0 {
		s.Pitch = 1
	}
	if s.Axial != 0 {
		d := s.Axial / s.Pitch
		root := 1 - rise - crest - fall
		d = math.Max(-crest+0.02, math.Min(root-0.02, d))
		crest += d
	}
	if s.Starts < 1 {
		s.Starts = 1
	}
	depth := dfrac * s.Pitch
	rMaj := s.Major / 2
	rMin := math.Max(rMaj-depth, rMaj*0.2)
	depth = rMaj - rMin
	segs := int(math.Max(24, math.Min(96, s.Major*math.Pi/0.6)) * q)
	if segs < 16 {
		segs = 16
	}
	zsteps := int(math.Max(6, 8*q))
	// steep flanks are approximated between rings; keep that error below the flank clearance
	if s.Profile != "ISO metric 60°" {
		need := int(math.Ceil(s.Pitch / math.Max(0.06, math.Abs(s.Axial)*0.7)))
		zsteps = int(math.Min(48, math.Max(float64(zsteps), float64(need))))
	}
	L := z1 - z0
	nz := int(math.Ceil(L/s.Pitch*float64(zsteps))) + 1
	prof := func(t float64) float64 {
		t -= math.Floor(t)
		switch {
		case t < rise:
			return t / rise
		case t < rise+crest:
			return 1
		case t < rise+crest+fall:
			return 1 - (t-rise-crest)/fall
		default:
			return 0
		}
	}
	dir := 1.0
	if s.Left {
		dir = -1
	}
	var rings [][]Vec3
	for k := 0; k <= nz; k++ {
		z := z0 + L*float64(k)/float64(nz)
		ring := make([]Vec3, 0, segs)
		for i := 0; i < segs; i++ {
			a := 2 * math.Pi * float64(i) / float64(segs)
			t := (z-z0)/s.Pitch - dir*float64(s.Starts)*a/(2*math.Pi)
			r := rMin + depth*prof(t)
			if s.Chamfer {
				lim := rMin + math.Min(z-z0, z1-z)
				lim = math.Max(lim, rMin-depth*0.5)
				if r > lim {
					r = lim
				}
			}
			sn, cs := math.Sincos(a)
			ring = append(ring, V(r*cs, r*sn, z))
		}
		rings = append(rings, ring)
	}
	return Loft(rings, true, true)
}

var coarsePitch = [][2]float64{{1, 0.25}, {1.6, 0.35}, {2, 0.4}, {2.5, 0.45}, {3, 0.5}, {3.5, 0.6}, {4, 0.7}, {5, 0.8}, {6, 1}, {8, 1.25}, {10, 1.5}, {12, 1.75}, {14, 2}, {16, 2}, {18, 2.5}, {20, 2.5}, {22, 2.5}, {24, 3}, {27, 3}, {30, 3.5}, {36, 4}, {42, 4.5}, {48, 5}, {56, 5.5}, {64, 6}}
var finePitch = [][2]float64{{3, 0.35}, {4, 0.5}, {5, 0.5}, {6, 0.75}, {8, 1}, {10, 1.25}, {12, 1.25}, {14, 1.5}, {16, 1.5}, {18, 1.5}, {20, 1.5}, {24, 2}, {30, 2}, {36, 3}, {42, 3}, {48, 3}, {64, 4}}
var trPitch = [][2]float64{{8, 1.5}, {10, 2}, {12, 3}, {14, 3}, {16, 4}, {20, 4}, {24, 5}, {28, 5}, {32, 6}, {36, 6}, {40, 7}, {50, 8}, {60, 9}, {80, 10}, {100, 12}}

func nearestPitch(tab [][2]float64, d float64) float64 {
	best, bd := tab[0][1], math.MaxFloat64
	for _, e := range tab {
		if x := math.Abs(e[0] - d); x < bd {
			bd, best = x, e[1]
		}
	}
	// very large custom diameters: scale gently
	if last := tab[len(tab)-1]; d > last[0]*1.3 {
		best = last[1] * math.Sqrt(d/last[0])
	}
	return best
}

// StandardPitch returns the standard pitch for a nominal diameter.
func StandardPitch(std, profile string, d float64) float64 {
	if profile == "Trapezoidal 30°" || profile == "Square" || profile == "Buttress" {
		return nearestPitch(trPitch, d)
	}
	if std == "Metric fine" {
		return nearestPitch(finePitch, d)
	}
	return nearestPitch(coarsePitch, d)
}

var fitClearance = map[string]float64{"Exact (0)": 0, "Tight (0.2 mm)": 0.2, "Normal (0.35 mm)": 0.35, "Loose (0.5 mm)": 0.5, "Very loose (0.7 mm)": 0.7}

// threadParams are the thread controls shared by cylinders, tubes and holes.
func threadParams(cond string, defStd string) []Param {
	s := func(p Param) Param { p.ShowIf = cond; return p }
	return []Param{
		s(sel("tProfile", "Thread profile", "ISO metric 60°", threadProfiles...).help("ISO metric = normal screws. Trapezoidal / square = lead screws and jar lids. Buttress = takes load in one direction.")),
		s(sel("tStd", "Pitch", defStd, "Metric coarse", "Metric fine", "Custom")),
		s(num("tPitch", "Custom pitch", 2, 0.2, 20, 0.05, "mm")).show(cond + ";tStd=Custom"),
		s(num("tStarts", "Starts", 1, 1, 8, 1, "").help("More starts = faster travel per turn (e.g. bottle caps use 2–4).")),
		s(sel("tHand", "Hand", "Right-hand", "Right-hand", "Left-hand")),
		s(sel("tFit", "Fit clearance", "Normal (0.35 mm)", "Exact (0)", "Tight (0.2 mm)", "Normal (0.35 mm)", "Loose (0.5 mm)", "Very loose (0.7 mm)").help("Total diametral play between a screw and its nut. Half is taken off solids and half added to holes, so a threaded solid and a threaded hole of the same diameter fit together.")),
		s(boolp("tChamfer", "Chamfer thread ends", true)),
	}
}

// specFrom builds a thread spec from shared parameters; internal threads grow by half the fit.
func specFrom(p Params, nominal float64, internal bool) ThreadSpec {
	prof := p.S("tProfile", "ISO metric 60°")
	pitch := StandardPitch(p.S("tStd", "Metric coarse"), prof, nominal)
	if p.S("tStd", "") == "Custom" {
		pitch = math.Max(0.2, p.F("tPitch", 2))
	}
	c := fitClearance[p.S("tFit", "Normal (0.35 mm)")]
	major := nominal - c/2
	if internal {
		major = nominal + c/2
	}
	ax := -c / 2
	if internal {
		ax = c / 2
	}
	return ThreadSpec{Major: major, Pitch: pitch, Profile: prof, Starts: p.I("tStarts", 1), Left: p.S("tHand", "") == "Left-hand", Chamfer: p.B("tChamfer", true) && !internal, Axial: ax}
}

// ThreadInfo describes the thread in words for the editor.
func ThreadInfo(s ThreadSpec, nominal float64, internal bool) string {
	depth := math.Min(ThreadDepth(s.Profile, s.Pitch), s.Major*0.4)
	minor := s.Major - 2*depth
	pd := s.Major - depth
	lead := s.Pitch * float64(max(s.Starts, 1))
	helix := math.Atan(lead/(math.Pi*pd)) * 180 / math.Pi
	kind := "External (bolt)"
	if internal {
		kind = "Internal (nut / hole)"
	}
	hand := "right-hand"
	if s.Left {
		hand = "left-hand"
	}
	flank := map[string]string{"ISO metric 60°": "60°", "Trapezoidal 30°": "30°", "Square": "0° (square)", "Buttress": "7° / 45°"}[s.Profile]
	name := fmt.Sprintf("M%g×%g", nominal, s.Pitch)
	if s.Profile != "ISO metric 60°" {
		name = fmt.Sprintf("%s %g×%g", map[string]string{"Trapezoidal 30°": "Tr", "Square": "Sq", "Buttress": "S"}[s.Profile], nominal, lead)
		if s.Starts > 1 {
			name += fmt.Sprintf(" (P%g)", s.Pitch)
		}
	}
	return fmt.Sprintf("%s · %s · %s, %d start\nMajor Ø %.3f · Pitch-dia Ø %.3f · Minor Ø %.3f mm\nPitch %.3f · Lead %.3f mm/turn · Depth %.3f mm\nFlank angle %s · Helix angle %.2f°",
		name, kind, hand, max(s.Starts, 1), s.Major, pd, minor, s.Pitch, lead, depth, flank, helix)
}
