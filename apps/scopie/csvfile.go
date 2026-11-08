// Reading waveforms saved as CSV files: from Scopie, or saved on the USB stick
// or internal storage of a handheld or bench scope. The first column must be
// time; up to four voltage columns follow.
package scopie

import (
	"bytes"
	"errors"
	"math"
	"sort"
	"strconv"
	"strings"
)

// sniffDelimiter picks the separator that splits the first lines most consistently.
func sniffDelimiter(sample string) byte {
	lines := strings.Split(sample, "\n")
	if len(lines) > 20 {
		lines = lines[:20]
	}
	best, bestScore := byte(','), -1
	for _, d := range []byte{',', ';', '\t'} {
		counts := map[int]int{}
		for _, ln := range lines {
			if n := strings.Count(ln, string(d)); n > 0 {
				counts[n]++
			}
		}
		score := 0
		for _, k := range counts {
			if k > score {
				score = k
			}
		}
		if score > bestScore {
			best, bestScore = d, score
		}
	}
	return best
}

func loadCSV(text []byte) (*Frame, error) {
	text = bytes.TrimPrefix(text, []byte("\xEF\xBB\xBF"))
	s := strings.ReplaceAll(string(text), "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	sample := s
	if len(sample) > 4096 {
		sample = sample[:4096]
	}
	delim := sniffDelimiter(sample)
	var rows [][]float64
	for _, ln := range strings.Split(s, "\n") {
		cells := strings.Split(ln, string(delim))
		var vals []float64
		ok := true
		for _, cell := range cells {
			c := strings.ReplaceAll(strings.TrimSpace(strings.Trim(cell, `"`)), " ", "")
			if c == "" {
				ok = false
				break
			}
			f, err := strconv.ParseFloat(c, 64)
			if err != nil {
				ok = false
				break
			}
			vals = append(vals, f)
		}
		if ok && len(vals) >= 2 {
			rows = append(rows, vals)
		}
	}
	if len(rows) < 4 {
		return nil, errors.New("no numeric time/voltage columns were found")
	}
	width := len(rows[0])
	for _, r := range rows {
		width = minInt(width, len(r))
	}
	n := len(rows)
	diffs := make([]float64, 0, n-1)
	for i := 1; i < n; i++ {
		diffs = append(diffs, rows[i][0]-rows[i-1][0])
	}
	sort.Float64s(diffs)
	dt := diffs[len(diffs)/2]
	if len(diffs)%2 == 0 {
		dt = (diffs[len(diffs)/2-1] + diffs[len(diffs)/2]) / 2
	}
	f := &Frame{Triggered: true, Gen: -99, Seq: -1}
	first := 1
	t0 := rows[0][0]
	if !(dt > 0) || math.IsInf(dt, 0) { // first column isn't time: use the sample number
		dt, first, t0 = 1e-3, 0, 0
	}
	for c := 0; c < maxCh && first+c < width; c++ {
		d := make([]float32, n)
		lo, hi := math.Inf(1), math.Inf(-1)
		for i, r := range rows {
			d[i] = float32(r[first+c])
			lo, hi = math.Min(lo, r[first+c]), math.Max(hi, r[first+c])
		}
		if hi <= lo {
			hi = lo + 1
		}
		f.Data[c] = d
		f.Ranges[c] = CodeRange{lo - 1e-9, hi + 1e-9}
	}
	for c := 0; c < maxCh; c++ {
		f.Scales[c] = Scale{1, 0}
		if f.Data[c] == nil {
			f.Ranges[c] = CodeRange{0, 1}
		}
	}
	f.DT, f.TFirst = dt, t0
	f.PreLen = int(math.Round(-t0 / dt))
	return f, nil
}
