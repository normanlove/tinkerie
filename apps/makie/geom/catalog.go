package geom

import (
	"fmt"
	"math"
	"strconv"
)

// Param describes one editable setting of a shape. It is sent to the editor as JSON.
type Param struct {
	Key     string   `json:"key"`
	Label   string   `json:"label"`
	Type    string   `json:"type"` // number, select, bool, text, points
	Def     any      `json:"def"`
	Min     float64  `json:"min,omitempty"`
	Max     float64  `json:"max,omitempty"`
	Step    float64  `json:"step,omitempty"`
	Unit    string   `json:"unit,omitempty"`
	Options []string `json:"options,omitempty"`
	Help    string   `json:"help,omitempty"`
	ShowIf  string   `json:"showIf,omitempty"` // "key=value" visibility rule
}

// ShapeDef is one entry of the shape library.
type ShapeDef struct {
	Type     string                        `json:"type"`
	Name     string                        `json:"name"`
	Category string                        `json:"category"`
	Desc     string                        `json:"desc"`
	Params   []Param                       `json:"params"`
	Hole     bool                          `json:"hole"`
	Fixed    bool                          `json:"fixed"` // real-world sized part: no free scaling
	Axes     map[string]string             `json:"axes,omitempty"`
	Color    string                        `json:"color"`
	Tags     string                        `json:"tags,omitempty"`
	Client   bool                          `json:"client,omitempty"`
	Hidden   bool                          `json:"hidden,omitempty"` // made by a tool, not listed in the library
	Gen      func(p Params) (*Mesh, error) `json:"-"`
}

// Params is the decoded parameter map of a node.
type Params map[string]any

func (p Params) F(k string, def float64) float64 {
	switch v := p[k].(type) {
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return def
		}
		return v
	case int:
		return float64(v)
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	case bool:
		if v {
			return 1
		}
		return 0
	}
	return def
}

func (p Params) I(k string, def int) int { return int(math.Round(p.F(k, float64(def)))) }

func (p Params) S(k, def string) string {
	if v, ok := p[k].(string); ok && v != "" {
		return v
	}
	if v, ok := p[k].(float64); ok {
		return strconv.FormatFloat(v, 'f', -1, 64)
	}
	return def
}

func (p Params) B(k string, def bool) bool {
	switch v := p[k].(type) {
	case bool:
		return v
	case float64:
		return v != 0
	case string:
		return v == "true" || v == "1" || v == "yes" || v == "On"
	}
	return def
}

// Pts reads a list of [x,y] points.
func (p Params) Pts(k string) []P2 {
	raw, ok := p[k].([]any)
	if !ok {
		return nil
	}
	var out []P2
	for _, r := range raw {
		if a, ok := r.([]any); ok && len(a) >= 2 {
			x, _ := a[0].(float64)
			y, _ := a[1].(float64)
			out = append(out, P2{x, y})
		}
	}
	return out
}

// Quality multiplier for curve resolution (set by the editor).
func (p Params) Q() float64 {
	q := p.F("_q", 1)
	if q < 0.3 {
		q = 0.3
	}
	if q > 3 {
		q = 3
	}
	return q
}

// Segs picks a sensible number of segments for a circle of diameter d.
func (p Params) Segs(d float64) int {
	n := int(math.Round(d * math.Pi / 0.7 * p.Q()))
	lo, hi := int(20*p.Q()), int(144*p.Q())
	if n < lo {
		n = lo
	}
	if n > hi {
		n = hi
	}
	return (n + 3) / 4 * 4
}

var registry = map[string]*ShapeDef{}
var order []string

func register(d *ShapeDef) {
	if gearTypes[d.Type] {
		d.Category = "Gears"
	}
	if d.Color == "" {
		d.Color = "#3b82f6"
	}
	registry[d.Type] = d
	order = append(order, d.Type)
}

// Catalog returns all shape definitions in display order.
func Catalog() []*ShapeDef {
	out := make([]*ShapeDef, 0, len(order))
	for _, k := range order {
		out = append(out, registry[k])
	}
	return out
}

// Generate builds the local mesh of a primitive.
func Generate(typ string, p Params) (m *Mesh, err error) {
	d, ok := registry[typ]
	if !ok || d.Gen == nil {
		return nil, fmt.Errorf("unknown shape %q", typ)
	}
	// fill defaults
	full := Params{}
	for _, pr := range d.Params {
		full[pr.Key] = pr.Def
	}
	for k, v := range p {
		full[k] = v
	}
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("could not build %s: %v", d.Name, r)
		}
	}()
	m, err = d.Gen(full)
	if err == nil && m != nil {
		info, ports := m.Info, m.Ports
		if openEdgeCount(m) > 0 {
			m = m.FixTJunctions()
		}
		m.Orient()
		m.Info, m.Ports = info, ports
	}
	return
}

func num(key, label string, def, min, max, step float64, unit string) Param {
	return Param{Key: key, Label: label, Type: "number", Def: def, Min: min, Max: max, Step: step, Unit: unit}
}
func sel(key, label string, def string, opts ...string) Param {
	return Param{Key: key, Label: label, Type: "select", Def: def, Options: opts}
}
func boolp(key, label string, def bool) Param {
	return Param{Key: key, Label: label, Type: "bool", Def: def}
}
func (p Param) help(h string) Param { p.Help = h; return p }
func (p Param) show(s string) Param { p.ShowIf = s; return p }

func clampF(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }
