package meterie

// mathx.go: math channels (V×I power, V/I resistance, energy, charge, custom
// expressions), statistics, E-series tables.

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// MathPreset identifies built-in math channel types.
type MathPreset int

const (
	MathPower MathPreset = iota
	MathResistance
	MathEnergy
	MathCharge
	MathDiff
	MathSum
	MathRatio
	MathCustom
)

var mathPresetNames = []string{
	"Power  P = A × B", "Resistance  R = A ÷ B", "Energy  ∫ A·B dt (Wh)", "Charge  ∫ A dt (mAh)",
	"Difference  A − B", "Sum  A + B", "Ratio  A ÷ B", "Custom formula",
}

// MathDef describes how a math channel is computed.
type MathDef struct {
	Preset MathPreset
	A, B   int // source channel IDs (B may be 0)
	Expr   string
	Unit   string
	ast    exprNode
	acc    float64 // integrator state
	lastT  int64
	lastY  float64
	hasLst bool
}

// Inputs returns the referenced channel IDs.
func (m *MathDef) Inputs() []int {
	ids := []int{m.A}
	if m.B != 0 && m.Preset != MathCharge {
		ids = append(ids, m.B)
	}
	if m.Preset == MathCustom {
		ids = exprChannels(m.ast)
	}
	return ids
}

func (m *MathDef) Prepare() error {
	if m.Preset != MathCustom {
		return nil
	}
	n, err := parseExpr(m.Expr)
	if err != nil {
		return err
	}
	m.ast = n
	return nil
}

func (m *MathDef) Quantity() (Quantity, string) {
	switch m.Preset {
	case MathPower:
		return QPower, "W"
	case MathResistance:
		return QResistance, "Ω"
	case MathEnergy:
		return QEnergy, "Wh"
	case MathCharge:
		return QCharge, "Ah"
	}
	return QUnitless, m.Unit
}

// Eval computes the new value; vals maps channel ID -> latest value. t is unix ns.
func (m *MathDef) Eval(vals map[int]float64, t int64) (float64, bool) {
	a, okA := vals[m.A]
	b, okB := vals[m.B]
	switch m.Preset {
	case MathPower:
		return a * b, okA && okB
	case MathResistance, MathRatio:
		if !okA || !okB || b == 0 {
			return 0, false
		}
		return a / b, true
	case MathDiff:
		return a - b, okA && okB
	case MathSum:
		return a + b, okA && okB
	case MathEnergy, MathCharge:
		y := a
		if m.Preset == MathEnergy {
			if !okA || !okB {
				return m.acc, false
			}
			y = a * b
		} else if !okA {
			return m.acc, false
		}
		if math.IsNaN(y) {
			return m.acc, true
		}
		if m.hasLst && t > m.lastT {
			dtH := float64(t-m.lastT) / 3.6e12
			if dtH < 1.0/60 { // ignore gaps longer than a minute
				m.acc += (y + m.lastY) / 2 * dtH
			}
		}
		m.lastT, m.lastY, m.hasLst = t, y, true
		return m.acc, true
	case MathCustom:
		if m.ast == nil {
			return 0, false
		}
		v, err := m.ast.eval(vals)
		if err != nil {
			return 0, false
		}
		return v, true
	}
	return 0, false
}

func (m *MathDef) Reset() { m.acc, m.lastT, m.lastY, m.hasLst = 0, 0, 0, false }

// ---------------------------------------------------------------------------
// Tiny expression language: numbers, ch1..chN (or A..H = channel by letter),
// + - * / ^, parentheses, functions abs sqrt min max log10 ln exp pow sin cos.

type exprNode interface {
	eval(v map[int]float64) (float64, error)
}

type numNode float64
type chanNode int
type unNode struct {
	op byte
	x  exprNode
}
type binNode struct {
	op   byte
	a, b exprNode
}
type callNode struct {
	fn   string
	args []exprNode
}

var errNoValue = errors.New("no value")

func (n numNode) eval(map[int]float64) (float64, error) { return float64(n), nil }
func (n chanNode) eval(v map[int]float64) (float64, error) {
	x, ok := v[int(n)]
	if !ok {
		return 0, errNoValue
	}
	return x, nil
}
func (n unNode) eval(v map[int]float64) (float64, error) {
	x, err := n.x.eval(v)
	return -x, err
}
func (n binNode) eval(v map[int]float64) (float64, error) {
	a, err := n.a.eval(v)
	if err != nil {
		return 0, err
	}
	b, err := n.b.eval(v)
	if err != nil {
		return 0, err
	}
	switch n.op {
	case '+':
		return a + b, nil
	case '-':
		return a - b, nil
	case '*':
		return a * b, nil
	case '/':
		return a / b, nil
	case '^':
		return math.Pow(a, b), nil
	}
	return 0, errors.New("bad op")
}
func (n callNode) eval(v map[int]float64) (float64, error) {
	xs := make([]float64, len(n.args))
	for i, a := range n.args {
		x, err := a.eval(v)
		if err != nil {
			return 0, err
		}
		xs[i] = x
	}
	need := func(k int) error {
		if len(xs) != k {
			return fmt.Errorf("%s needs %d argument(s)", n.fn, k)
		}
		return nil
	}
	switch n.fn {
	case "abs":
		if e := need(1); e != nil {
			return 0, e
		}
		return math.Abs(xs[0]), nil
	case "sqrt":
		if e := need(1); e != nil {
			return 0, e
		}
		return math.Sqrt(xs[0]), nil
	case "log10", "log":
		if e := need(1); e != nil {
			return 0, e
		}
		return math.Log10(xs[0]), nil
	case "ln":
		if e := need(1); e != nil {
			return 0, e
		}
		return math.Log(xs[0]), nil
	case "exp":
		if e := need(1); e != nil {
			return 0, e
		}
		return math.Exp(xs[0]), nil
	case "sin":
		if e := need(1); e != nil {
			return 0, e
		}
		return math.Sin(xs[0]), nil
	case "cos":
		if e := need(1); e != nil {
			return 0, e
		}
		return math.Cos(xs[0]), nil
	case "pow":
		if e := need(2); e != nil {
			return 0, e
		}
		return math.Pow(xs[0], xs[1]), nil
	case "min", "max":
		if len(xs) == 0 {
			return 0, errors.New("min/max need arguments")
		}
		r := xs[0]
		for _, x := range xs[1:] {
			if n.fn == "min" {
				r = math.Min(r, x)
			} else {
				r = math.Max(r, x)
			}
		}
		return r, nil
	}
	return 0, fmt.Errorf("unknown function %q", n.fn)
}

type exprParser struct {
	s   string
	pos int
}

func parseExpr(s string) (exprNode, error) {
	p := &exprParser{s: s}
	n, err := p.sum()
	if err != nil {
		return nil, err
	}
	p.ws()
	if p.pos != len(p.s) {
		return nil, fmt.Errorf("unexpected %q at %d", p.s[p.pos:], p.pos+1)
	}
	return n, nil
}

func (p *exprParser) ws() {
	for p.pos < len(p.s) && (p.s[p.pos] == ' ' || p.s[p.pos] == '\t') {
		p.pos++
	}
}
func (p *exprParser) peek() byte {
	p.ws()
	if p.pos < len(p.s) {
		return p.s[p.pos]
	}
	return 0
}
func (p *exprParser) sum() (exprNode, error) {
	a, err := p.prod()
	if err != nil {
		return nil, err
	}
	for {
		c := p.peek()
		if c != '+' && c != '-' {
			return a, nil
		}
		p.pos++
		b, err := p.prod()
		if err != nil {
			return nil, err
		}
		a = binNode{c, a, b}
	}
}
func (p *exprParser) prod() (exprNode, error) {
	a, err := p.unary()
	if err != nil {
		return nil, err
	}
	for {
		c := p.peek()
		if c == 'x' || c == '×' {
			c = '*'
		}
		if c != '*' && c != '/' {
			return a, nil
		}
		p.pos++
		b, err := p.unary()
		if err != nil {
			return nil, err
		}
		a = binNode{c, a, b}
	}
}
func (p *exprParser) unary() (exprNode, error) {
	if p.peek() == '-' {
		p.pos++
		x, err := p.unary()
		return unNode{'-', x}, err
	}
	if p.peek() == '+' {
		p.pos++
	}
	return p.power()
}
func (p *exprParser) power() (exprNode, error) {
	a, err := p.atom()
	if err != nil {
		return nil, err
	}
	if p.peek() == '^' {
		p.pos++
		b, err := p.unary()
		if err != nil {
			return nil, err
		}
		return binNode{'^', a, b}, nil
	}
	return a, nil
}
func (p *exprParser) atom() (exprNode, error) {
	c := p.peek()
	if c == 0 {
		return nil, errors.New("unexpected end of formula")
	}
	if c == '(' {
		p.pos++
		n, err := p.sum()
		if err != nil {
			return nil, err
		}
		if p.peek() != ')' {
			return nil, errors.New("missing )")
		}
		p.pos++
		return n, nil
	}
	if c >= '0' && c <= '9' || c == '.' {
		st := p.pos
		for p.pos < len(p.s) && (p.s[p.pos] >= '0' && p.s[p.pos] <= '9' || p.s[p.pos] == '.' || p.s[p.pos] == 'e' || p.s[p.pos] == 'E' ||
			((p.s[p.pos] == '-' || p.s[p.pos] == '+') && p.pos > st && (p.s[p.pos-1] == 'e' || p.s[p.pos-1] == 'E'))) {
			p.pos++
		}
		v, err := strconv.ParseFloat(p.s[st:p.pos], 64)
		if err != nil {
			return nil, fmt.Errorf("bad number %q", p.s[st:p.pos])
		}
		return numNode(v), nil
	}
	if unicode.IsLetter(rune(c)) {
		st := p.pos
		for p.pos < len(p.s) && (unicode.IsLetter(rune(p.s[p.pos])) || unicode.IsDigit(rune(p.s[p.pos]))) {
			p.pos++
		}
		id := strings.ToLower(p.s[st:p.pos])
		if p.peek() == '(' {
			p.pos++
			var args []exprNode
			if p.peek() != ')' {
				for {
					a, err := p.sum()
					if err != nil {
						return nil, err
					}
					args = append(args, a)
					if p.peek() == ',' {
						p.pos++
						continue
					}
					break
				}
			}
			if p.peek() != ')' {
				return nil, errors.New("missing ) after function arguments")
			}
			p.pos++
			return callNode{id, args}, nil
		}
		switch id {
		case "pi":
			return numNode(math.Pi), nil
		case "e":
			return numNode(math.E), nil
		}
		if strings.HasPrefix(id, "ch") {
			n, err := strconv.Atoi(id[2:])
			if err == nil && n > 0 {
				return chanNode(n), nil
			}
		}
		if len(id) == 1 && id[0] >= 'a' && id[0] <= 'h' {
			return chanNode(int(id[0]-'a') + 1), nil
		}
		return nil, fmt.Errorf("unknown name %q (use ch1, ch2 … or A, B …)", id)
	}
	return nil, fmt.Errorf("unexpected %q", string(c))
}

func exprChannels(n exprNode) []int {
	set := map[int]bool{}
	var walk func(exprNode)
	walk = func(n exprNode) {
		switch x := n.(type) {
		case chanNode:
			set[int(x)] = true
		case unNode:
			walk(x.x)
		case binNode:
			walk(x.a)
			walk(x.b)
		case callNode:
			for _, a := range x.args {
				walk(a)
			}
		}
	}
	if n != nil {
		walk(n)
	}
	var ids []int
	for k := range set {
		ids = append(ids, k)
	}
	sort.Ints(ids)
	return ids
}

// ---------------------------------------------------------------------------
// Statistics over a window of samples.

type Stats struct {
	N                        int
	Min, Max, Mean, Std, RMS float64
	PkPk                     float64
	First, Last              float64
	T0, T1                   int64
	Integral                 float64 // value·hours
}

func ComputeStats(s []Sample, t0, t1 int64, unit uint8) Stats {
	var st Stats
	st.Min, st.Max = math.Inf(1), math.Inf(-1)
	var sum, sum2 float64
	var prevT int64
	var prevV float64
	for _, p := range s {
		if p.T < t0 || p.T > t1 || p.U != unit || math.IsNaN(p.V) {
			continue
		}
		if st.N == 0 {
			st.First, st.T0 = p.V, p.T
		} else {
			dt := float64(p.T-prevT) / 3.6e12
			if dt > 0 && dt < 1.0/60 {
				st.Integral += (p.V + prevV) / 2 * dt
			}
		}
		st.N++
		sum += p.V
		sum2 += p.V * p.V
		st.Min = math.Min(st.Min, p.V)
		st.Max = math.Max(st.Max, p.V)
		st.Last, st.T1 = p.V, p.T
		prevT, prevV = p.T, p.V
	}
	if st.N == 0 {
		return Stats{}
	}
	n := float64(st.N)
	st.Mean = sum / n
	st.RMS = math.Sqrt(sum2 / n)
	v := sum2/n - st.Mean*st.Mean
	if v < 0 {
		v = 0
	}
	st.Std = math.Sqrt(v)
	st.PkPk = st.Max - st.Min
	return st
}

// ---------------------------------------------------------------------------
// E-series preferred values (for the resistor/capacitor sorter and LED wizard).

var eSeries = map[string][]float64{
	"E6":  {1.0, 1.5, 2.2, 3.3, 4.7, 6.8},
	"E12": {1.0, 1.2, 1.5, 1.8, 2.2, 2.7, 3.3, 3.9, 4.7, 5.6, 6.8, 8.2},
	"E24": {1.0, 1.1, 1.2, 1.3, 1.5, 1.6, 1.8, 2.0, 2.2, 2.4, 2.7, 3.0, 3.3, 3.6, 3.9, 4.3, 4.7, 5.1, 5.6, 6.2, 6.8, 7.5, 8.2, 9.1},
	"E96": {1.00, 1.02, 1.05, 1.07, 1.10, 1.13, 1.15, 1.18, 1.21, 1.24, 1.27, 1.30, 1.33, 1.37, 1.40, 1.43, 1.47, 1.50, 1.54, 1.58,
		1.62, 1.65, 1.69, 1.74, 1.78, 1.82, 1.87, 1.91, 1.96, 2.00, 2.05, 2.10, 2.15, 2.21, 2.26, 2.32, 2.37, 2.43, 2.49, 2.55,
		2.61, 2.67, 2.74, 2.80, 2.87, 2.94, 3.01, 3.09, 3.16, 3.24, 3.32, 3.40, 3.48, 3.57, 3.65, 3.74, 3.83, 3.92, 4.02, 4.12,
		4.22, 4.32, 4.42, 4.53, 4.64, 4.75, 4.87, 4.99, 5.11, 5.23, 5.36, 5.49, 5.62, 5.76, 5.90, 6.04, 6.19, 6.34, 6.49, 6.65,
		6.81, 6.98, 7.15, 7.32, 7.50, 7.68, 7.87, 8.06, 8.25, 8.45, 8.66, 8.87, 9.09, 9.31, 9.53, 9.76},
}

func init() {
	var e48 []float64
	for i, v := range eSeries["E96"] {
		if i%2 == 0 {
			e48 = append(e48, v)
		}
	}
	eSeries["E48"] = e48
}

// NearestE returns the nearest preferred value in the given series.
func NearestE(v float64, series string) float64 {
	tab, ok := eSeries[series]
	if !ok || v <= 0 || math.IsNaN(v) || math.IsInf(v, 0) {
		return v
	}
	dec := math.Floor(math.Log10(v))
	best, bestErr := v, math.Inf(1)
	for _, d := range []float64{dec - 1, dec, dec + 1} {
		m := math.Pow(10, d)
		for _, t := range tab {
			c := t * m
			e := math.Abs(math.Log(v / c))
			if e < bestErr {
				best, bestErr = c, e
			}
		}
	}
	// tidy floating point (e.g. 4.7*1000 = 4700.000000001)
	return math.Round(best*1e6/math.Pow(10, dec)) / 1e6 * math.Pow(10, dec)
}

// NearestEAbove returns the smallest preferred value >= v.
func NearestEAbove(v float64, series string) float64 {
	tab := eSeries[series]
	if v <= 0 || len(tab) == 0 {
		return v
	}
	dec := math.Floor(math.Log10(v))
	for _, d := range []float64{dec, dec + 1} {
		m := math.Pow(10, d)
		for _, t := range tab {
			if t*m >= v*(1-1e-9) {
				return math.Round(t*m*1e6/math.Pow(10, d)) / 1e6 * math.Pow(10, d)
			}
		}
	}
	return v
}
