package geom

import (
	"fmt"
	"math"
)

// ===================== IKEA SKÅDIS pegboard =====================
// SKÅDIS slots are 5 × 15 mm, in columns 20 mm apart with every other column
// shifted 20 mm, so slots in one row are 40 mm apart. The board is 5 mm thick.
// Parts here face +Y; the pegboard's front face is the plane y = 0.

const (
	skSlotW  = 5.0
	skSlotH  = 15.0
	skBoard  = 5.0
	skPitchX = 40.0
	skPitchZ = 40.0
)

// skPeg is one connector: a tab through the slot and a hook that drops behind the board.
// zc is the height of the slot centre.
func skPeg(x, zc float64) *Mesh {
	w := skSlotW - 0.5
	tabTop := zc + skSlotH/2 - 1.0
	tabBot := zc - skSlotH/2 + 0.3
	tab := boxAt(V(x-w/2, -skBoard-0.4, tabBot), V(x+w/2, 0.01, tabTop))
	hook := boxAt(V(x-w/2, -skBoard-0.4-2.6, tabBot-6), V(x+w/2, -skBoard-0.39, tabTop))
	// rounded hook tip so it slides in easily
	tip := along(cylAt(0, 0, 2.6, -w/2, w/2, 16), 0).Transformed(Translate(x, -skBoard-0.4-1.3, tabBot-6))
	return UnionAll([]*Mesh{tab, hook, tip})
}

// pegs along the top of a part of width W (centred), at height zc
func skPegs(W, zc float64) []*Mesh {
	n := int(math.Max(1, math.Floor((W-6)/skPitchX)+1))
	var out []*Mesh
	for i := 0; i < n; i++ {
		x := (float64(i) - float64(n-1)/2) * skPitchX
		out = append(out, skPeg(x, zc))
	}
	return out
}

func init() {
	sk := func(typ, name, desc string, params []Param, gen func(p Params) (*Mesh, error)) {
		register(&ShapeDef{Type: typ, Name: name, Category: "Pegboard", Desc: desc + " Fits IKEA SKÅDIS pegboards.", Tags: "skadis skådis ikea pegboard wall organiser storage", Color: "#e2e8f0", Params: params, Gen: gen})
	}
	sk("sk_hook", "SKÅDIS hook", "Hook for tools, headphones, keys or bags.",
		[]Param{num("reach", "Reach", 40, 10, 200, 1, "mm"), num("w", "Hook width", 8, 4.6, 40, 0.5, "mm"), num("lip", "Tip height", 10, 0, 60, 1, "mm")},
		func(p Params) (*Mesh, error) {
			reach, w, lip := p.F("reach", 40), p.F("w", 8), p.F("lip", 10)
			t := 4.0
			back := boxAt(V(-w/2, 0, -20), V(w/2, t, 8))
			arm := boxAt(V(-w/2, 0, -20), V(w/2, reach, -20+t))
			up := boxAt(V(-w/2, reach-t, -20), V(w/2, reach, -20+t+lip))
			// gusset under the arm joint
			g := prismX(Loop{{0, -20}, {math.Min(reach*0.4, 15), -20}, {0, -20 + math.Min(reach*0.4, 15)}}, w)
			return UnionAll([]*Mesh{back, arm, up, g, skPeg(0, 0)}), nil
		})
	sk("sk_shelf", "SKÅDIS shelf", "Small shelf with a front lip.",
		[]Param{num("w", "Width", 80, 20, 400, 1, "mm"), num("d", "Depth", 60, 15, 250, 1, "mm"), num("lip", "Front lip", 6, 0, 40, 0.5, "mm")},
		func(p Params) (*Mesh, error) {
			W, Dp, lip := p.F("w", 80), p.F("d", 60), p.F("lip", 6)
			t := 3.0
			back := boxAt(V(-W/2, 0, -30), V(W/2, t, 8))
			floorM := boxAt(V(-W/2, 0, -30), V(W/2, Dp, -30+t))
			front := boxAt(V(-W/2, Dp-2.4, -30), V(W/2, Dp, -30+t+lip))
			ms := []*Mesh{back, floorM, front}
			for _, x := range []float64{-W/2 + 1.2, W/2 - 1.2} {
				ms = append(ms, prismX(Loop{{0, -30}, {Dp * 0.7, -30}, {0, -30 + Dp*0.5}}, 2.4).Transformed(Translate(x, 0, 0)))
			}
			ms = append(ms, skPegs(W, 0)...)
			return UnionAll(ms), nil
		})
	sk("sk_bin", "SKÅDIS bin", "Open-top box for screws, pens or bits.",
		[]Param{num("w", "Width", 80, 20, 400, 1, "mm"), num("d", "Depth", 50, 15, 250, 1, "mm"), num("h", "Height", 40, 10, 200, 1, "mm"), num("div", "Dividers", 0, 0, 10, 1, "")},
		func(p Params) (*Mesh, error) {
			W, Dp, H := p.F("w", 80), p.F("d", 50), p.F("h", 40)
			t := 1.8
			z0 := -H + 8
			box := Subtract(boxAt(V(-W/2, 0, z0), V(W/2, Dp, 8)), boxAt(V(-W/2+t, t, z0+t), V(W/2-t, Dp-t, 9)))
			ms := []*Mesh{box}
			n := p.I("div", 0)
			for i := 1; i <= n; i++ {
				x := -W/2 + W*float64(i)/float64(n+1)
				ms = append(ms, boxAt(V(x-0.8, 0, z0), V(x+0.8, Dp, 8-4)))
			}
			ms = append(ms, skPegs(W, 0)...)
			return UnionAll(ms), nil
		})
	sk("sk_toolholder", "SKÅDIS tool holder", "Row of holes for screwdrivers, pliers, pens or brushes.",
		[]Param{num("n", "Holes", 5, 1, 30, 1, ""), num("hole", "Hole Ø", 12, 3, 60, 0.5, "mm"), num("d", "Depth", 30, 10, 200, 1, "mm")},
		func(p Params) (*Mesh, error) {
			n, hd, Dp := p.I("n", 5), p.F("hole", 12), p.F("d", 30)
			gap := math.Max(2.4, hd*0.25)
			W := float64(n)*(hd+gap) + gap
			t := 5.0
			back := boxAt(V(-W/2, 0, -24), V(W/2, 3, 8))
			shelf := boxAt(V(-W/2, 0, -t), V(W/2, Dp, 0))
			var holes []*Mesh
			for i := 0; i < n; i++ {
				x := -W/2 + gap + hd/2 + float64(i)*(hd+gap)
				holes = append(holes, cylAt(x, 3+hd/2+gap, hd, -t-1, 1, p.Segs(hd)))
			}
			if 3+hd+gap*2 > Dp {
				return nil, fmt.Errorf("make it deeper for %g mm holes", hd)
			}
			ms := []*Mesh{back, Subtract(shelf, UnionAll(holes))}
			for _, x := range []float64{-W/2 + 1.2, W/2 - 1.2} {
				ms = append(ms, prismX(Loop{{0, -t}, {Dp * 0.7, -t}, {0, -24}}, 2.4).Transformed(Translate(x, 0, 0)))
			}
			ms = append(ms, skPegs(W, 0)...)
			return UnionAll(ms), nil
		})
	sk("sk_clip", "SKÅDIS connector", "Plain connector plate with screw holes — screw or glue it to anything to hang it on the board.",
		[]Param{num("w", "Width", 50, 12, 400, 1, "mm"), num("h", "Height", 30, 15, 200, 1, "mm"), sel("screw", "Screw holes", "M3", "None", "M3", "M4")},
		func(p Params) (*Mesh, error) {
			W, H := p.F("w", 50), p.F("h", 30)
			t := 3.0
			plate := boxAt(V(-W/2, 0, -H+10), V(W/2, t, 10))
			ms := []*Mesh{plate}
			ms = append(ms, skPegs(W, 0)...)
			m := UnionAll(ms)
			if sz := p.S("screw", "M3"); sz != "None" {
				hd := metrics[sz].Normal + 0.2
				e := holeEdge(hd)
				var hs []*Mesh
				for _, x := range []float64{-W/2 + e, W/2 - e} {
					z := -H + 10 + e
					hs = append(hs, along(cylAt(0, 0, hd, -1, t+1, 20), 1).Transformed(Translate(x, 0, z)), along(coneAt(hd, hd+3.4, 0, 1.7, 20), 1).Transformed(Translate(x, t-1.7+0.01, z)))
				}
				m = Subtract(m, UnionAll(hs))
			}
			return m, nil
		})
	sk("sk_cable", "SKÅDIS cable holder", "Open loop that holds coiled cables, chargers or rope.",
		[]Param{num("d", "Loop Ø", 40, 10, 150, 1, "mm"), num("w", "Width", 10, 5, 40, 0.5, "mm")},
		func(p Params) (*Mesh, error) {
			d, w := p.F("d", 40), p.F("w", 10)
			t := 3.0
			back := boxAt(V(-w/2, 0, -14), V(w/2, t, 8))
			ringM := along(Subtract(cylAt(0, 0, d+2*t, -w/2, w/2, p.Segs(d)), cylAt(0, 0, d, -w, w, p.Segs(d))), 0).Transformed(Translate(0, d/2+t, -14+d/2+t))
			ringM = Subtract(ringM, boxAt(V(-w, d/2+t, -14+d/2+t), V(w, d+2*t+1, -14+d+2*t+1)))
			return UnionAll([]*Mesh{back, ringM, skPeg(0, 0)}), nil
		})
	sk("sk_spool", "SKÅDIS rod holder", "Two arms and a rod for tape, string, foil or filament spools.",
		[]Param{num("len", "Rod length", 80, 20, 400, 1, "mm"), num("reach", "Reach", 50, 15, 200, 1, "mm"), num("rod", "Rod Ø", 12, 4, 60, 0.5, "mm")},
		func(p Params) (*Mesh, error) {
			L, reach, rod := p.F("len", 80), p.F("reach", 50), p.F("rod", 12)
			t := 4.0
			W := math.Max(L, 40) + 2*t
			back := boxAt(V(-W/2, 0, -16), V(W/2, t, 8))
			var ms []*Mesh
			ms = append(ms, back)
			for _, x := range []float64{-W/2 + t/2, W/2 - t/2} {
				ms = append(ms, prismX(Loop{{0, -16}, {reach + rod/2 + t, -16}, {reach + rod/2 + t, -16 + rod + t}, {0, 8}}, t).Transformed(Translate(x, 0, 0)))
			}
			rodM := along(cylAt(0, 0, rod, -W/2, W/2, p.Segs(rod)), 0).Transformed(Translate(0, reach, -16+rod/2+t))
			ms = append(ms, rodM)
			ms = append(ms, skPegs(W, 0)...)
			return UnionAll(ms), nil
		})
	sk("sk_phone", "SKÅDIS phone / tablet stand", "Angled ledge that holds a phone or tablet facing out.",
		[]Param{num("w", "Width", 80, 40, 300, 1, "mm"), num("t", "Device thickness", 12, 5, 40, 0.5, "mm")},
		func(p Params) (*Mesh, error) {
			W, dt := p.F("w", 80), p.F("t", 12)
			back := boxAt(V(-W/2, 0, -40), V(W/2, 3, 8))
			ledge := prismX(Loop{{0, -40}, {dt + 8, -40}, {dt + 8, -40 + 14}, {dt + 5, -40 + 14}, {dt + 5, -37}, {3, -37}, {3, -30}, {0, -30}}, W)
			return UnionAll(append([]*Mesh{back, ledge}, skPegs(W, 0)...)), nil
		})
}

// prismX extrudes a profile drawn in the Y–Z plane symmetrically along X.
func prismX(l Loop, width float64) *Mesh {
	return shapeX(Shape{Outer: l}, -width/2, width/2)
}
