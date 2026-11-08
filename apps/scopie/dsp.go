// Signal processing: triggering, measurements, FFT, sin(x)/x interpolation.
// A straight port of Scopie's numpy code, checked against it by dsp_test.go.
package scopie

import (
	"math"
	"math/cmplx"
	"sort"
	"sync"
)

var nan = math.NaN()

// ---------------------------------------------------------------- triggers

// findRisingEdges finds rising crossings of level with hysteresis. A trigger
// fires on the first sample >= level after the signal has been below
// level-hyst. armed is the state carried over from the previous block.
func findRisingEdges(x []float64, level, hyst float64, armed bool) ([]int, bool) {
	var hits []int
	low := level - hyst
	for i, v := range x {
		if v < low {
			armed = true
		} else if v >= level {
			if armed {
				hits = append(hits, i)
			}
			armed = false
		}
	}
	return hits, armed
}

func negate(x []float64) []float64 {
	out := make([]float64, len(x))
	for i, v := range x {
		out[i] = -v
	}
	return out
}

// crossingFraction says where between sample a (t=-1) and b (t=0) the level was crossed: 0..1.
func crossingFraction(a, b, level float64) float64 {
	d := b - a
	if d == 0 {
		return 1
	}
	return clampF((level-a)/d, 0, 1)
}

func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// ---------------------------------------------------------------- display helpers

// peakDecimate reduces y to about maxPoints values while keeping every peak:
// it returns (min,max) pairs of blocks of `step` samples. The trailing
// samples that don't fill a block are dropped (as the original did).
func peakDecimate(y []float64, maxPoints int) (out []float64, step int) {
	n := len(y)
	if n <= maxPoints {
		return y, 1
	}
	pairs := maxPoints / 2
	step = n / pairs
	out = make([]float64, 0, pairs*2)
	for p := 0; p < pairs; p++ {
		blk := y[p*step : (p+1)*step]
		lo, hi := blk[0], blk[0]
		for _, v := range blk[1:] {
			if v < lo {
				lo = v
			} else if v > hi {
				hi = v
			}
		}
		out = append(out, lo, hi)
	}
	return out, step
}

const lanczosA = 6

var kernelMu sync.Mutex
var kernelCache = map[int][][]float64{}

func sinc(x float64) float64 {
	if x == 0 {
		return 1
	}
	px := math.Pi * x
	return math.Sin(px) / px
}

func lanczosKernels(factor int) [][]float64 {
	kernelMu.Lock()
	defer kernelMu.Unlock()
	if k, ok := kernelCache[factor]; ok {
		return k
	}
	k := make([][]float64, factor)
	for j := 0; j < factor; j++ {
		frac := float64(j) / float64(factor)
		w := make([]float64, 2*lanczosA)
		sum := 0.0
		for q := range w {
			t := float64(q-lanczosA+1) - frac
			w[q] = sinc(t) * sinc(t/lanczosA)
			sum += w[q]
		}
		for q := range w {
			w[q] /= sum
		}
		k[j] = w
	}
	kernelCache[factor] = k
	return k
}

// sincInterpolate upsamples y by factor with a Lanczos (sin(x)/x) kernel, the
// way real scopes join sparse samples. Output length is (n-1)*factor+1.
func sincInterpolate(y []float64, factor int) []float64 {
	n := len(y)
	if n < 4 || factor <= 1 {
		return append([]float64(nil), y...)
	}
	ks := lanczosKernels(factor)
	out := make([]float64, (n-1)*factor+1)
	at := func(i int) float64 {
		if i < 0 {
			return y[0]
		}
		if i >= n {
			return y[n-1]
		}
		return y[i]
	}
	for i := 0; i < n; i++ {
		out[i*factor] = y[i]
	}
	for j := 1; j < factor; j++ {
		w := ks[j]
		for i := 0; i < n-1; i++ {
			s := 0.0
			for q, wq := range w {
				s += at(i+q-lanczosA+1) * wq
			}
			out[i*factor+j] = s
		}
	}
	return out
}

// ---------------------------------------------------------------- measurements

func minMax(v []float64) (lo, hi float64) {
	lo, hi = v[0], v[0]
	for _, x := range v[1:] {
		if x < lo {
			lo = x
		}
		if x > hi {
			hi = x
		}
	}
	return
}

func mean(v []float64) float64 {
	s := 0.0
	for _, x := range v {
		s += x
	}
	return s / float64(len(v))
}

func median(v []float64) float64 {
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	n := len(s)
	if n%2 == 1 {
		return s[n/2]
	}
	return (s[n/2-1] + s[n/2]) / 2
}

// edgeTimes returns sub-sample times of mid-level crossings, with 10 % hysteresis.
func edgeTimes(v []float64, dt float64, rising bool) ([]float64, float64) {
	vmin, vmax := minMax(v)
	span := vmax - vmin
	if span <= 0 {
		return nil, (vmin + vmax) / 2
	}
	mid := (vmax + vmin) / 2
	hyst := 0.1 * span
	x, lvl := v, mid
	if !rising {
		x, lvl = negate(v), -mid
	}
	idx, _ := findRisingEdges(x, lvl, hyst, false)
	var out []float64
	for _, i := range idx {
		if i <= 0 {
			continue
		}
		a, b := x[i-1], x[i]
		d := b - a
		if d == 0 {
			d = 1
		}
		frac := clampF((lvl-a)/d, 0, 1)
		out = append(out, (float64(i-1)+frac)*dt)
	}
	return out, mid
}

// topBase finds the most common high and low levels (a scope's "top" and "base").
func topBase(v []float64) (top, base float64) {
	vmin, vmax := minMax(v)
	if vmax == vmin {
		return vmax, vmin
	}
	const bins = 64
	var hist [bins]int
	norm := bins / (vmax - vmin)
	for _, x := range v {
		b := int((x - vmin) * norm)
		if b >= bins {
			b = bins - 1
		}
		if b < 0 {
			b = 0
		}
		hist[b]++
	}
	edge := func(i int) float64 { return vmin + (vmax-vmin)*float64(i)/bins }
	lo, hi := 0, bins/2
	for i := 1; i < bins/2; i++ {
		if hist[i] > hist[lo] {
			lo = i
		}
	}
	for i := bins/2 + 1; i < bins; i++ {
		if hist[i] > hist[hi] {
			hi = i
		}
	}
	n := float64(len(v))
	if float64(hist[hi]) < 0.05*n || float64(hist[lo]) < 0.05*n {
		return vmax, vmin
	}
	selMean := func(b int) float64 {
		a, c := edge(b), edge(b+1)
		s, k := 0.0, 0
		for _, x := range v {
			if x >= a && x <= c {
				s += x
				k++
			}
		}
		if k == 0 {
			return a
		}
		return s / float64(k)
	}
	return selMean(hi), selMean(lo)
}

func transitionTime(v []float64, dt float64, rising bool) float64 {
	top, base := topBase(v)
	span := top - base
	if span <= 0 {
		return nan
	}
	lo := base + 0.1*span
	hi := base + 0.9*span
	x, l1, l2 := v, lo, hi
	if !rising {
		x, l1, l2 = negate(v), -hi, -lo
	}
	t1, _ := findRisingEdges(x, l1, 0.05*span, false)
	t2, _ := findRisingEdges(x, l2, 0.05*span, false)
	if len(t1) == 0 || len(t2) == 0 {
		return nan
	}
	var times []float64
	j := 0
	for _, a := range t1 {
		for j < len(t2) && t2[j] < a {
			j++
		}
		if j >= len(t2) {
			break
		}
		times = append(times, float64(t2[j]-a))
	}
	if len(times) == 0 {
		return nan
	}
	return median(times) * dt
}

type MeasDef struct {
	Key, Label, Unit, Tip string
}

var measurements = []MeasDef{
	{"freq", "Frequency", "Hz", "How many times per second the waveform repeats.\nNeeds at least two full cycles on screen."},
	{"period", "Period", "s", "Time taken by one complete cycle of the waveform (1 / frequency)."},
	{"vpp", "Peak-to-peak", "V", "Difference between the highest and lowest voltage on screen."},
	{"vmax", "Maximum", "V", "The highest voltage on screen."},
	{"vmin", "Minimum", "V", "The lowest voltage on screen."},
	{"mean", "Mean (DC)", "V", "Average voltage. For a pure AC signal this is close to 0 V;\nfor a battery it is the battery voltage."},
	{"rms", "RMS", "V", "Root-mean-square voltage including any DC part: the 'heating' value.\nA 230 V mains supply is 230 V RMS."},
	{"acrms", "AC RMS", "V", "RMS of only the changing (AC) part, with the DC average removed.\nThis is what most multimeters show in AC mode."},
	{"amp", "Amplitude", "V", "Top level minus base level. For square waves this ignores overshoot spikes."},
	{"duty", "Duty cycle", "%", "Percentage of each cycle the signal spends in its high state.\nA symmetrical square wave is 50 %."},
	{"pwidth", "+Width", "s", "How long each high pulse lasts."},
	{"rise", "Rise time", "s", "Time to go from 10 % to 90 % of the step on a rising edge.\nUse a fast time/div setting so the edge is spread over several divisions."},
	{"fall", "Fall time", "s", "Time to go from 90 % to 10 % on a falling edge."},
}

// Meas holds every measurement of one trace; NaN means "not measurable".
type Meas map[string]float64

// measure computes all measurements for one voltage array.
func measure(v []float64, dt float64) Meas {
	r := Meas{}
	for _, m := range measurements {
		r[m.Key] = nan
	}
	if len(v) < 4 {
		return r
	}
	vmin, vmax := minMax(v)
	mn := mean(v)
	r["vmax"], r["vmin"], r["vpp"], r["mean"] = vmax, vmin, vmax-vmin, mn
	sq, dev := 0.0, 0.0
	for _, x := range v {
		sq += x * x
		dev += (x - mn) * (x - mn)
	}
	r["rms"] = math.Sqrt(sq / float64(len(v)))
	r["acrms"] = math.Sqrt(dev / float64(len(v)))
	top, base := topBase(v)
	r["amp"] = top - base
	if vmax-vmin < 1e-9 {
		return r
	}
	rises, _ := edgeTimes(v, dt, true)
	falls, _ := edgeTimes(v, dt, false)
	if len(rises) >= 2 {
		period := (rises[len(rises)-1] - rises[0]) / float64(len(rises)-1)
		if period > 0 {
			r["period"] = period
			r["freq"] = 1 / period
			var widths []float64
			for _, t := range rises[:len(rises)-1] {
				for _, f := range falls {
					if f > t {
						if w := f - t; w < period {
							widths = append(widths, w)
						}
						break
					}
				}
			}
			if len(widths) > 0 {
				r["pwidth"] = mean(widths)
				r["duty"] = 100 * r["pwidth"] / period
			}
		}
	}
	r["rise"] = transitionTime(v, dt, true)
	r["fall"] = transitionTime(v, dt, false)
	return r
}

// phaseDelay returns the delay of CH2's rising edge after CH1's, and the phase in degrees.
func phaseDelay(v1, v2 []float64, dt float64) (float64, float64) {
	if len(v1) < 4 || len(v2) < 4 {
		return nan, nan
	}
	r1, _ := edgeTimes(v1, dt, true)
	r2, _ := edgeTimes(v2, dt, true)
	if len(r1) < 2 || len(r2) < 1 {
		return nan, nan
	}
	period := (r1[len(r1)-1] - r1[0]) / float64(len(r1)-1)
	if period <= 0 {
		return nan, nan
	}
	t1 := r1[0]
	for _, t := range r2 {
		if t >= t1 {
			d := math.Mod(t-t1, period)
			ph := 360 * d / period
			if ph > 180 {
				ph -= 360
				d -= period
			}
			return d, ph
		}
	}
	return nan, nan
}

// ---------------------------------------------------------------- FFT

var fftWindows = []string{"Hann", "Hamming", "Blackman", "Flat top", "Rectangular"}

func window(name string, n int) []float64 {
	w := make([]float64, n)
	if n == 1 {
		w[0] = 1
		return w
	}
	m := float64(n - 1)
	for k := range w {
		x := 2 * math.Pi * float64(k) / m
		switch name {
		case "Hamming":
			w[k] = 0.54 - 0.46*math.Cos(x)
		case "Blackman":
			w[k] = 0.42 - 0.5*math.Cos(x) + 0.08*math.Cos(2*x)
		case "Flat top":
			w[k] = 0.21557895 - 0.41663158*math.Cos(x) + 0.277263158*math.Cos(2*x) -
				0.083578947*math.Cos(3*x) + 0.006947368*math.Cos(4*x)
		case "Rectangular":
			w[k] = 1
		default: // Hann
			w[k] = 0.5 - 0.5*math.Cos(x)
		}
	}
	return w
}

// spectrum returns the single-sided amplitude spectrum in dBV (RMS) and the
// bin spacing in Hz.
func spectrum(v []float64, dt float64, win string, maxPoints int) (db []float64, df float64) {
	if len(v) > maxPoints {
		v = v[len(v)-maxPoints:]
	}
	n := len(v)
	if n < 16 {
		return nil, 0
	}
	w := window(win, n)
	xw := make([]float64, n)
	wsum := 0.0
	for i := range v {
		xw[i] = v[i] * w[i]
		wsum += w[i]
	}
	X := rfft(xw)
	nb := n/2 + 1
	db = make([]float64, nb)
	for k := 0; k < nb; k++ {
		mag := cmplx.Abs(X[k]) / wsum
		if k > 0 {
			mag = mag * 2 / math.Sqrt2
		}
		db[k] = 20 * math.Log10(math.Max(mag, 1e-7))
	}
	return db, 1 / (float64(n) * dt)
}

// rfft returns bins 0..n/2 of the DFT of a real signal, using one complex
// FFT of half the length (the usual "two real halves" trick).
func rfft(x []float64) []complex128 {
	n := len(x)
	if n%2 != 0 || n < 4 {
		c := make([]complex128, n)
		for i, v := range x {
			c[i] = complex(v, 0)
		}
		return fft(c)[:n/2+1]
	}
	h := n / 2
	z := make([]complex128, h)
	for k := 0; k < h; k++ {
		z[k] = complex(x[2*k], x[2*k+1])
	}
	Z := fft(z)
	out := make([]complex128, h+1)
	for k := 0; k <= h; k++ {
		zk := Z[k%h]
		zc := cmplx.Conj(Z[(h-k)%h])
		even := (zk + zc) / 2
		odd := (zk - zc) / complex(0, 2)
		a := -2 * math.Pi * float64(k) / float64(n)
		out[k] = even + complex(math.Cos(a), math.Sin(a))*odd
	}
	return out
}

// fft computes the discrete Fourier transform for any length (radix-2, or
// Bluestein's algorithm for other sizes).
func fft(x []complex128) []complex128 {
	n := len(x)
	if n&(n-1) == 0 {
		out := append([]complex128(nil), x...)
		fft2(out, false)
		return out
	}
	m := 1
	for m < 2*n-1 {
		m <<= 1
	}
	w := make([]complex128, n)
	for k := 0; k < n; k++ {
		// exp(-i*pi*k^2/n), with k^2 reduced mod 2n for accuracy
		kk := (int64(k) * int64(k)) % int64(2*n)
		a := -math.Pi * float64(kk) / float64(n)
		w[k] = complex(math.Cos(a), math.Sin(a))
	}
	a := make([]complex128, m)
	b := make([]complex128, m)
	for k := 0; k < n; k++ {
		a[k] = x[k] * w[k]
	}
	b[0] = cmplx.Conj(w[0])
	for k := 1; k < n; k++ {
		b[k] = cmplx.Conj(w[k])
		b[m-k] = b[k]
	}
	fft2(a, false)
	fft2(b, false)
	for i := range a {
		a[i] *= b[i]
	}
	fft2(a, true)
	out := make([]complex128, n)
	inv := 1 / float64(m)
	for k := 0; k < n; k++ {
		out[k] = a[k] * complex(inv, 0) * w[k]
	}
	return out
}

// fft2 is an in-place iterative radix-2 FFT (inverse without the 1/n scale).
func fft2(a []complex128, inverse bool) {
	n := len(a)
	for i, j := 1, 0; i < n; i++ {
		bit := n >> 1
		for ; j&bit != 0; bit >>= 1 {
			j ^= bit
		}
		j ^= bit
		if i < j {
			a[i], a[j] = a[j], a[i]
		}
	}
	sign := -1.0
	if inverse {
		sign = 1
	}
	for size := 2; size <= n; size <<= 1 {
		ang := sign * 2 * math.Pi / float64(size)
		wl := complex(math.Cos(ang), math.Sin(ang))
		half := size / 2
		// precompute twiddles for this stage
		tw := make([]complex128, half)
		tw[0] = 1
		for k := 1; k < half; k++ {
			if k%64 == 0 { // refresh to avoid drift
				a := ang * float64(k)
				tw[k] = complex(math.Cos(a), math.Sin(a))
			} else {
				tw[k] = tw[k-1] * wl
			}
		}
		for i := 0; i < n; i += size {
			for k := 0; k < half; k++ {
				u := a[i+k]
				t := a[i+k+half] * tw[k]
				a[i+k] = u + t
				a[i+k+half] = u - t
			}
		}
	}
}
