// Tunie — font.go: a tiny 5×7 pixel font used to print labels on tests,
// either flat on the bed or raised (embossed) on the side of a tower.
// Part of the nrmluvdev maker suite. Created by nrmluvdev.
package tunie

import "math"

var font5x7 = map[rune][7]string{
	'0': {"01110", "10001", "10011", "10101", "11001", "10001", "01110"},
	'1': {"00100", "01100", "00100", "00100", "00100", "00100", "01110"},
	'2': {"01110", "10001", "00001", "00010", "00100", "01000", "11111"},
	'3': {"11111", "00010", "00100", "00010", "00001", "10001", "01110"},
	'4': {"00010", "00110", "01010", "10010", "11111", "00010", "00010"},
	'5': {"11111", "10000", "11110", "00001", "00001", "10001", "01110"},
	'6': {"00110", "01000", "10000", "11110", "10001", "10001", "01110"},
	'7': {"11111", "00001", "00010", "00100", "01000", "01000", "01000"},
	'8': {"01110", "10001", "10001", "01110", "10001", "10001", "01110"},
	'9': {"01110", "10001", "10001", "01111", "00001", "00010", "01100"},
	'.': {"0", "0", "0", "0", "0", "0", "1"},
	'-': {"000", "000", "000", "111", "000", "000", "000"},
	'+': {"00000", "00100", "00100", "11111", "00100", "00100", "00000"},
	' ': {"00", "00", "00", "00", "00", "00", "00"},
	'A': {"01110", "10001", "10001", "11111", "10001", "10001", "10001"},
	'B': {"11110", "10001", "10001", "11110", "10001", "10001", "11110"},
	'C': {"01110", "10001", "10000", "10000", "10000", "10001", "01110"},
	'D': {"11100", "10010", "10001", "10001", "10001", "10010", "11100"},
	'E': {"11111", "10000", "10000", "11110", "10000", "10000", "11111"},
	'F': {"11111", "10000", "10000", "11110", "10000", "10000", "10000"},
	'G': {"01110", "10001", "10000", "10111", "10001", "10001", "01111"},
	'H': {"10001", "10001", "10001", "11111", "10001", "10001", "10001"},
	'I': {"111", "010", "010", "010", "010", "010", "111"},
	'K': {"10001", "10010", "10100", "11000", "10100", "10010", "10001"},
	'L': {"10000", "10000", "10000", "10000", "10000", "10000", "11111"},
	'M': {"10001", "11011", "10101", "10101", "10001", "10001", "10001"},
	'N': {"10001", "10001", "11001", "10101", "10011", "10001", "10001"},
	'O': {"01110", "10001", "10001", "10001", "10001", "10001", "01110"},
	'P': {"11110", "10001", "10001", "11110", "10000", "10000", "10000"},
	'R': {"11110", "10001", "10001", "11110", "10100", "10010", "10001"},
	'S': {"01111", "10000", "10000", "01110", "00001", "00001", "11110"},
	'T': {"11111", "00100", "00100", "00100", "00100", "00100", "00100"},
	'U': {"10001", "10001", "10001", "10001", "10001", "10001", "01110"},
	'V': {"10001", "10001", "10001", "10001", "10001", "01010", "00100"},
	'W': {"10001", "10001", "10001", "10101", "10101", "10101", "01010"},
	'X': {"10001", "10001", "01010", "00100", "01010", "10001", "10001"},
	'Y': {"10001", "10001", "01010", "00100", "00100", "00100", "00100"},
	'Z': {"11111", "00001", "00010", "00100", "01000", "10000", "11111"},
}

func glyph(r rune) [7]string {
	if gl, ok := font5x7[r]; ok {
		return gl
	}
	return font5x7[' ']
}

// textCols returns the width of a string in font columns (with 1-col gaps).
func textCols(s string) int {
	n := 0
	for i, r := range []rune(s) {
		if i > 0 {
			n++
		}
		n += len(glyph(r)[0])
	}
	return n
}

// textRuns returns [start,end] column runs of lit pixels for one font row (0 = top).
func textRuns(s string, row int) [][2]int {
	var lit []bool
	for i, r := range []rune(s) {
		if i > 0 {
			lit = append(lit, false)
		}
		for _, c := range glyph(r)[row] {
			lit = append(lit, c == '1')
		}
	}
	var runs [][2]int
	start := -1
	for i := 0; i <= len(lit); i++ {
		on := i < len(lit) && lit[i]
		if on && start < 0 {
			start = i
		}
		if !on && start >= 0 {
			runs = append(runs, [2]int{start, i - 1})
			start = -1
		}
	}
	return runs
}

// flatText prints text lying flat, bottom-left corner at (x, y), pixel size p.
func (g *GW) flatText(s string, x, y, p, sp float64) {
	lw := g.lw
	k := int(math.Max(1, math.Round(p/lw)))
	g.typ("LABEL")
	for row := 0; row < 7; row++ {
		runs := textRuns(s, row)
		if len(runs) == 0 {
			continue
		}
		rowBottom := y + float64(6-row)*p
		for j := 0; j < k; j++ {
			ly := rowBottom + p*(float64(j)+0.5)/float64(k)
			fwd := j%2 == 0
			order := runs
			if !fwd {
				order = make([][2]int, len(runs))
				for i := range runs {
					order[i] = runs[len(runs)-1-i]
				}
			}
			for _, r := range order {
				xs := x + float64(r[0])*p + lw/2
				xe := x + float64(r[1]+1)*p - lw/2
				if xe < xs {
					xs = (xs + xe) / 2
					xe = xs + 0.05
				}
				if !fwd {
					xs, xe = xe, xs
				}
				g.travel(xs, ly)
				g.extrudeTo(xe, ly, sp)
			}
		}
	}
}

func textWidth(s string, p float64) float64 { return float64(textCols(s)) * p }

// emboss prints raised text on a vertical face during the current layer.
// face: 'F' = front face (normal -Y, text runs +X), 'R' = right face (normal +X, text runs +Y).
// a0..a1 is the face span; faceC the face coordinate (y for F, x for R).
// z0 is where the bottom of the label starts; ph the pixel height (≥ one layer).
func (g *GW) emboss(face byte, a0, a1, faceC, z0, ph, pw float64, s string, sp float64) {
	cols := textCols(s)
	if cols == 0 {
		return
	}
	if maxPw := (a1 - a0 - 1) / float64(cols); pw > maxPw {
		pw = maxPw
	}
	if pw < g.lw*0.9 {
		return // face too narrow to label legibly
	}
	if g.z <= z0+1e-6 || g.z > z0+7*ph+1e-6 {
		return
	}
	ri := int((g.z - z0 - 1e-6) / ph) // 0 = bottom row
	row := 6 - ri
	if row < 0 || row > 6 {
		return
	}
	runs := textRuns(s, row)
	if len(runs) == 0 {
		return
	}
	start := (a0+a1)/2 - float64(cols)*pw/2
	off := g.lw * 0.5
	g.typ("LABEL")
	for _, r := range runs {
		s0 := start + float64(r[0])*pw
		s1 := start + float64(r[1]+1)*pw
		switch face {
		case 'F':
			y := faceC - off
			g.travel(s0, y)
			g.extrudeTo(s1, y, sp)
		case 'R':
			x := faceC + off
			g.travel(x, s0)
			g.extrudeTo(x, s1, sp)
		}
	}
}

// pixelHeight rounds a desired label pixel height to whole layers.
func (g *GW) pixelHeight(want float64) float64 {
	n := math.Max(1, math.Round(want/g.s.LayerH))
	return n * g.s.LayerH
}
