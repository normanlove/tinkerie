package meterie

import (
	"errors"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

func formatG(v float64) string { return strconv.FormatFloat(v, 'g', 6, 64) }

// parseNumberSI parses "4.7k", "2u2", "10 m", "1e-3", "100n", "4k7" (resistor style).
func parseNumberSI(s string) (float64, error) {
	s = strings.TrimSpace(strings.ReplaceAll(s, ",", "."))
	s = strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(s, "Ω"), "F"), "V")
	s = strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(s, "ohm"), "A"), "Hz"))
	if s == "" {
		return 0, errors.New("empty")
	}
	if v, err := strconv.ParseFloat(s, 64); err == nil {
		return v, nil
	}
	mults := map[rune]float64{'p': 1e-12, 'n': 1e-9, 'u': 1e-6, 'µ': 1e-6, 'm': 1e-3, 'k': 1e3, 'K': 1e3, 'M': 1e6, 'G': 1e9, 'R': 1, 'r': 1}
	rs := []rune(s)
	for i, c := range rs {
		if m, ok := mults[c]; ok {
			a := string(rs[:i])
			b := string(rs[i+1:])
			if a == "" {
				a = "0"
			}
			num := a
			if b != "" {
				num = a + "." + b
			}
			v, err := strconv.ParseFloat(strings.TrimSpace(num), 64)
			if err != nil {
				return 0, err
			}
			return v * m, nil
		}
	}
	return 0, errors.New("not a number")
}

func fmtDur(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return itoa(h) + "h " + pad2(m) + "m " + pad2(s) + "s"
	}
	if m > 0 {
		return itoa(m) + "m " + pad2(s) + "s"
	}
	return strconv.FormatFloat(d.Seconds(), 'f', 1, 64) + "s"
}

func pad2(i int) string {
	if i < 10 {
		return "0" + itoa(i)
	}
	return itoa(i)
}

func absf(v float64) float64 { return math.Abs(v) }

func newRand() *rand.Rand { return rand.New(rand.NewSource(time.Now().UnixNano())) }
