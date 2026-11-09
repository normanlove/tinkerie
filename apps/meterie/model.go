// Meterie — PC multimeter app for the nrmluvdev hobbyist tool suite.
// Copyright (c) nrmluvdev. Free software for hobbyists.
//
// model.go: the measurement data model shared by every part of the app.
package meterie

import (
	"fmt"
	"math"
	"strings"
	"time"
)

const (
	AppName    = "Meterie"
	AppVersion = "0.6.0-alpha"
	AppAuthor  = "nrmluvdev"
)

// Quantity is what physical thing a reading measures.
type Quantity int

const (
	QNone Quantity = iota
	QVoltage
	QCurrent
	QResistance
	QCapacitance
	QFrequency
	QDuty
	QTemperature
	QDiode
	QContinuity
	QPower
	QEnergy
	QCharge
	QHFE
	QInductance
	QPercent
	QUnitless
	QTime
	qCount
)

var quantityNames = [...]string{
	"—", "Voltage", "Current", "Resistance", "Capacitance", "Frequency", "Duty cycle",
	"Temperature", "Diode", "Continuity", "Power", "Energy", "Charge", "hFE",
	"Inductance", "Percent", "Value", "Time",
}

func (q Quantity) String() string {
	if q >= 0 && int(q) < len(quantityNames) {
		return quantityNames[q]
	}
	return "?"
}

// BaseUnit returns the SI base symbol used to store values of this quantity.
func (q Quantity) BaseUnit() string {
	switch q {
	case QVoltage, QDiode:
		return "V"
	case QCurrent:
		return "A"
	case QResistance, QContinuity:
		return "Ω"
	case QCapacitance:
		return "F"
	case QFrequency:
		return "Hz"
	case QDuty, QPercent:
		return "%"
	case QTemperature:
		return "°C"
	case QPower:
		return "W"
	case QEnergy:
		return "Wh"
	case QCharge:
		return "Ah"
	case QInductance:
		return "H"
	case QTime:
		return "s"
	}
	return ""
}

// Flags mirror the annunciators on a meter's LCD.
type Flags uint32

const (
	FAC Flags = 1 << iota
	FDC
	FAuto
	FHold
	FRel
	FMin
	FMax
	FLowBat
	FDiode
	FBeep
	FOL // overload / out of range ("OL")
	FHV // high-voltage warning
	FPeakMax
	FPeakMin
	FManual
	FFahrenheit
	FUnderload
)

func (f Flags) Has(x Flags) bool { return f&x != 0 }

// Reading is one decoded sample from a meter.
type Reading struct {
	T       time.Time
	Value   float64  // value in base units (V, A, Ω, F, Hz, %, °C ...). NaN when OL.
	Q       Quantity // what it measures
	Unit    string   // base unit symbol, e.g. "V"
	Prefix  string   // display prefix the meter used ("m", "k", "µ" ...)
	Display string   // the digits exactly as the meter showed them, e.g. "-0.123" or "OL"
	Flags   Flags
	Counts  int    // full-scale counts of the active range, for the bar graph (0 = unknown)
	Note    string // optional free text (manual entry, OCR source ...)
}

// IsOL reports whether the reading is an overload.
func (r Reading) IsOL() bool {
	return r.Flags.Has(FOL) || math.IsNaN(r.Value) || math.IsInf(r.Value, 0)
}

// ModeLabel gives the big "DC VOLTAGE" style label.
func (r Reading) ModeLabel() string {
	q := r.Q
	s := strings.ToUpper(q.String())
	switch q {
	case QVoltage, QCurrent:
		if r.Flags.Has(FAC) && r.Flags.Has(FDC) {
			s = "AC+DC " + s
		} else if r.Flags.Has(FAC) {
			s = "AC " + s
		} else if r.Flags.Has(FDC) {
			s = "DC " + s
		}
	case QDiode:
		s = "DIODE TEST"
	case QContinuity:
		s = "CONTINUITY"
	}
	return s
}

// Unit prefixes.
var siPrefixes = []struct {
	p string
	m float64
}{
	{"p", 1e-12}, {"n", 1e-9}, {"µ", 1e-6}, {"m", 1e-3}, {"", 1}, {"k", 1e3}, {"M", 1e6}, {"G", 1e9},
}

func prefixMult(p string) float64 {
	if p == "u" {
		p = "µ"
	}
	for _, sp := range siPrefixes {
		if sp.p == p {
			return sp.m
		}
	}
	return 1
}

// unitAllowsPrefix: % and temperatures are never prefixed.
func unitAllowsPrefix(unit string) bool {
	switch unit {
	case "%", "°C", "°F", "", "hFE":
		return false
	}
	return true
}

// FormatSI formats v with an engineering prefix and ~sig significant digits.
func FormatSI(v float64, unit string, sig int) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "OL " + unit
	}
	if !unitAllowsPrefix(unit) {
		return strings.TrimSpace(fmtSig(v, sig) + " " + unit)
	}
	p, m := pickPrefix(v)
	return fmtSig(v/m, sig) + " " + p + unit
}

// SplitSI returns number text and prefixed unit separately (for the big readout).
func SplitSI(v float64, unit string, sig int) (string, string) {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "OL", unit
	}
	if !unitAllowsPrefix(unit) {
		return fmtSig(v, sig), unit
	}
	p, m := pickPrefix(v)
	return fmtSig(v/m, sig), p + unit
}

func pickPrefix(v float64) (string, float64) {
	a := math.Abs(v)
	if a == 0 {
		return "", 1
	}
	best := siPrefixes[0]
	for _, sp := range siPrefixes {
		if a >= sp.m*0.9999995 {
			best = sp
		}
	}
	return best.p, best.m
}

// fmtSig formats with sig significant digits but never in exponent form.
func fmtSig(v float64, sig int) string {
	if sig < 1 {
		sig = 1
	}
	if v == 0 {
		return "0"
	}
	a := math.Abs(v)
	digitsBefore := int(math.Floor(math.Log10(a))) + 1
	dec := sig - digitsBefore
	if dec < 0 {
		dec = 0
	}
	if dec > 9 {
		dec = 9
	}
	return fmt.Sprintf("%.*f", dec, v)
}

// FormatValue formats for tables/CSV with a fixed number of significant digits.
func FormatValue(v float64) string {
	if math.IsNaN(v) {
		return "OL"
	}
	return fmt.Sprintf("%.7g", v)
}

// Sample is one point stored in a channel's timeline.
type Sample struct {
	T int64   // unix nanoseconds
	V float64 // base units (relative offset is NOT applied here)
	U uint8   // index into Channel.Units (lets us break the line when the meter changes mode)
	F Flags
}

// ChannelKind tells where a channel's data comes from.
type ChannelKind int

const (
	KindMeter ChannelKind = iota
	KindMath
	KindImport
	KindManual
	KindCamera
)

// Channel is one line on the shared timeline.
type Channel struct {
	ID       int
	Name     string
	Color    int // palette index
	Kind     ChannelKind
	Source   string // device description
	DeviceID int    // which connection feeds it (0 = none)
	Samples  []Sample
	Units    []unitKey // distinct (quantity, unit, AC/DC) seen
	Last     Reading
	HasLast  bool
	Visible  bool
	RelOn    bool
	RelValue float64
	// Math channel definition (KindMath only)
	Math *MathDef
	// smoothed display state
	barAnim float64
}

type unitKey struct {
	Q    Quantity
	Unit string
	AC   bool
}

func (c *Channel) unitIndex(r Reading) uint8 {
	k := unitKey{r.Q, r.Unit, r.Flags.Has(FAC) && !r.Flags.Has(FDC)}
	for i, u := range c.Units {
		if u == k {
			return uint8(i)
		}
	}
	if len(c.Units) >= 250 {
		return 0
	}
	c.Units = append(c.Units, k)
	return uint8(len(c.Units) - 1)
}

// CurrentUnit returns the unit index of the latest reading (or 0).
func (c *Channel) CurrentUnit() uint8 {
	if len(c.Samples) == 0 {
		return 0
	}
	return c.Samples[len(c.Samples)-1].U
}

func (c *Channel) UnitLabel() string {
	if !c.HasLast {
		if len(c.Units) > 0 {
			return c.Units[0].Unit
		}
		return ""
	}
	return c.Last.Unit
}

// Add appends a reading to the timeline.
func (c *Channel) Add(r Reading) {
	u := c.unitIndex(r)
	v := r.Value
	if r.IsOL() {
		v = math.NaN()
	}
	c.Samples = append(c.Samples, Sample{T: r.T.UnixNano(), V: v, U: u, F: r.Flags})
	c.Last = r
	c.HasLast = true
}

// DisplayValue applies relative mode.
func (c *Channel) DisplayValue(v float64) float64 {
	if c.RelOn {
		return v - c.RelValue
	}
	return v
}

// Palette of channel colours (dark-theme friendly).
var channelPalette = []Color{
	RGB(0x4C, 0xC9, 0xF0), // cyan
	RGB(0xFF, 0xB0, 0x20), // amber
	RGB(0xF4, 0x72, 0xB6), // pink
	RGB(0x8B, 0xE8, 0x6B), // lime
	RGB(0xA7, 0x8B, 0xFA), // violet
	RGB(0xFF, 0x8A, 0x4C), // orange
	RGB(0x3D, 0xDC, 0x97), // mint
	RGB(0xF8, 0x71, 0x71), // red
}

func chanColor(i int) Color {
	if i < 0 {
		i = 0
	}
	return channelPalette[i%len(channelPalette)]
}
