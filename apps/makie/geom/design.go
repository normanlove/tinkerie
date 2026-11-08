package geom

import "math"

// Design rules for strong FDM prints (0.4 mm nozzle). Every generator that puts a screw,
// bolt or insert into printed material sizes the material around it with these, so holes
// are never left with a thin wall or too close to an edge.
const (
	MinWall      = 1.2 // thinnest wall on any load-bearing part (3 perimeters)
	MinScrewWall = 1.6 // thinnest wall around any screw / bolt / insert hole (4 perimeters)
)

// screwWall is the material needed around a hole of diameter d. Bigger holes carry bigger
// screws (more torque / hoop stress), so the wall grows with the hole.
func screwWall(d float64) float64 { return math.Max(MinScrewWall, 0.3*d+0.8) }

// bossOD is the smallest round boss that safely holds a hole of diameter d.
func bossOD(d float64) float64 { return d + 2*screwWall(d) }

// hexCorner is the across-corners size of a hex pocket with across-flats af.
func hexCorner(af float64) float64 { return af / math.Cos(math.Pi/6) }

// holeEdge is the minimum distance from a hole's centre to the nearest outside edge.
func holeEdge(d float64) float64 { return d/2 + screwWall(d) }
