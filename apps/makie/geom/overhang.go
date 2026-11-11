package geom

// overhang.go: room in the case walls for parts that stick out past the board
// edge, such as a microcontroller module whose USB end hangs over, a USB-C
// socket, a button or an antenna. Each gets a pocket in the wall that's open at
// the top, so the board can still be lowered in; where the part comes out
// through the wall, that's an opening. The lid gets a filler to close the slot
// above the part. Created by nrmluvdev for Tinkerie.

import (
	"fmt"
	"math"
)

type overhang struct {
	p    WiriePart
	over float64 // how far it sticks out past the board edge (mm)
	side string  // "left", "right", "front", "back" (the main direction)
}

// overhangs lists the parts whose box reaches past the board outline.
func (c *WirieCase) overhangs() []overhang {
	outline := c.loop()
	if len(outline) < 3 {
		return nil
	}
	x0, y0, x1, y1 := loopBox(outline)
	var out []overhang
	for _, p := range c.Parts {
		if p.Kind == "mechanical" || p.Kind == "switch" && c.isKeyboard() {
			continue
		}
		b := p.Box
		if b.X1-b.X0 <= 0 || b.Y1-b.Y0 <= 0 {
			continue
		}
		// sample the box's edge and see how far any of it lies outside the board
		over := 0.0
		for i := 0; i <= 8; i++ {
			t := float64(i) / 8
			for _, q := range []P2{{b.X0 + (b.X1-b.X0)*t, b.Y0}, {b.X0 + (b.X1-b.X0)*t, b.Y1}, {b.X0, b.Y0 + (b.Y1-b.Y0)*t}, {b.X1, b.Y0 + (b.Y1-b.Y0)*t}} {
				if !outline.Contains(q) {
					over = math.Max(over, loopEdgeDist(outline, q))
				}
			}
		}
		if over < 0.05 {
			continue
		}
		side, best := "", -1e9
		for _, s := range []struct {
			n string
			v float64
		}{{"left", x0 - b.X0}, {"right", b.X1 - x1}, {"front", y0 - b.Y0}, {"back", b.Y1 - y1}} {
			if s.v > best {
				best, side = s.v, s.n
			}
		}
		out = append(out, overhang{p, over, side})
	}
	return out
}

// overhangPockets cuts the wall pockets (into base) and adds the lid fillers
// (into lid, if there is one). zWallTop is the top of the walls, zLid the
// underside of the lid.
func (c *WirieCase) overhangPockets(base, lid *Node, o CaseOpts, zWallTop, zLid float64, info *CaseInfo) {
	f3 := func(x, y, z float64) [3]float64 { return [3]float64{round3(x), round3(y), round3(z)} }
	zero := [3]float64{}
	_, _, bx1, by1 := loopBox(c.loop())
	bx0, by0, _, _ := loopBox(c.loop())
	g := o.Fit + 0.3 // room round the part
	for _, oh := range c.overhangs() {
		b := oh.p.Box
		lo := math.Min(b.Z0, b.Z1) - 0.3
		// the pocket: the part's box with room round it, reaching right through the
		// wall on the side it sticks out, and up past the top so the board drops in
		px0, py0, px1, py1 := b.X0-g, b.Y0-g, b.X1+g, b.Y1+g
		reach := o.Fit + o.Wall + 1
		switch oh.side {
		case "left":
			px0 = math.Min(px0, bx0-reach)
		case "right":
			px1 = math.Max(px1, bx1+reach)
		case "front":
			py0 = math.Min(py0, by0-reach)
		case "back":
			py1 = math.Max(py1, by1+reach)
		}
		top := zWallTop + 1
		base.Children = append(base.Children, newNode(fmt.Sprintf("Room for %s (sticks out %.1f mm)", oh.p.Ref, oh.over), "box",
			Params{"w": round3(px1 - px0), "d": round3(py1 - py0), "h": round3(top - lo)}, f3((px0+px1)/2, (py0+py1)/2, lo), zero, true, ""))
		// close the slot above the part from the lid, where the wall was
		partTop := math.Max(b.Z0, b.Z1) + 0.4
		if lid != nil && zLid-partTop > 0.8 {
			fx0, fy0, fx1, fy1 := px0+0.2, py0+0.2, px1-0.2, py1-0.2
			switch oh.side { // only the wall's thickness
			case "left":
				fx0, fx1 = bx0-o.Fit-o.Wall+0.1, bx0-o.Fit-0.1
			case "right":
				fx0, fx1 = bx1+o.Fit+0.1, bx1+o.Fit+o.Wall-0.1
			case "front":
				fy0, fy1 = by0-o.Fit-o.Wall+0.1, by0-o.Fit-0.1
			case "back":
				fy0, fy1 = by1+o.Fit+0.1, by1+o.Fit+o.Wall-0.1
			}
			if fx1 > fx0 && fy1 > fy0 {
				lid.Children = append(lid.Children, newNode("Slot filler for "+oh.p.Ref, "box",
					Params{"w": round3(fx1 - fx0), "d": round3(fy1 - fy0), "h": round3(zLid - partTop + 0.01)}, f3((fx0+fx1)/2, (fy0+fy1)/2, partTop), zero, false, ""))
			}
		}
		what := oh.p.Ref
		if oh.p.Name != "" {
			what += " (" + oh.p.Name + ")"
		}
		through := ""
		if oh.over > o.Fit+o.Wall-0.4 {
			through = ", and comes out through the wall"
		}
		info.Lines = append(info.Lines, fmt.Sprintf("%s sticks out %.1f mm past the board's %s edge%s: the wall has room for it, open at the top so the board drops in.", what, oh.over, oh.side, through))
	}
}
