package probie

import (
	"fmt"
	"math"
)

// generateDemo synthesises a realistic bench: UART on CH0, I²C on CH1/CH2,
// SPI on CH3–CH6 and a sweeping PWM on CH7.
type sigWriter struct {
	edges [8][]demoEdge
	level [8]byte
}

type demoEdge struct {
	t float64
	v byte
}

func (w *sigWriter) set(ch int, t float64, v byte) {
	if w.level[ch] == v {
		return
	}
	w.level[ch] = v
	w.edges[ch] = append(w.edges[ch], demoEdge{t, v})
}

func generateDemo(rate float64, n int) []byte {
	dur := float64(n) / rate
	w := &sigWriter{}
	init := [8]byte{1, 1, 1, 0, 0, 0, 1, 0} // UART idle hi, I²C idle hi, CS idle hi
	w.level = init

	// UART 115200 8N1 on CH0
	baud := 115200.0
	bt := 1 / baud
	msg := 0
	for t := 0.0008; t < dur; {
		s := fmt.Sprintf("Hello from Probie! #%d\r\n", msg)
		msg++
		for _, c := range []byte(s) {
			w.set(0, t, 0)
			for b := 0; b < 8; b++ {
				w.set(0, t+bt*float64(1+b), (c>>b)&1)
			}
			w.set(0, t+bt*9, 1)
			t += bt * 10
		}
		t += 0.003
	}

	// I²C 100 kHz on CH1 (SCL) / CH2 (SDA)
	q := 2.5e-6
	i2cByte := func(t float64, v byte, ack bool) float64 {
		for b := 7; b >= 0; b-- {
			w.set(2, t+q, (v>>b)&1)
			w.set(1, t+2*q, 1)
			w.set(1, t+4*q, 0)
			t += 4 * q
		}
		a := byte(1)
		if ack {
			a = 0
		}
		w.set(2, t+q, a)
		w.set(1, t+2*q, 1)
		w.set(1, t+4*q, 0)
		return t + 4*q
	}
	tx := 0
	for t := 0.0011; t+0.001 < dur; t += 0.0025 {
		x := t
		w.set(2, x, 0) // START
		w.set(1, x+q, 0)
		x += q
		if tx%2 == 0 {
			x = i2cByte(x, 0x3C<<1, true)
			x = i2cByte(x, 0x00, true)
			x = i2cByte(x, byte(0xA0+tx%16), true)
		} else {
			x = i2cByte(x, 0x48<<1, true)
			x = i2cByte(x, 0x00, true)
			w.set(2, x+q, 1) // repeated start
			w.set(1, x+2*q, 1)
			w.set(2, x+3*q, 0)
			w.set(1, x+4*q, 0)
			x += 4 * q
			x = i2cByte(x, 0x48<<1|1, true)
			x = i2cByte(x, byte(0x19+tx%5), true)
			x = i2cByte(x, 0x80, false)
		}
		w.set(2, x+q, 0) // STOP
		w.set(1, x+2*q, 1)
		w.set(2, x+3*q, 1)
		tx++
	}

	// SPI mode 0, 1 MHz: CH3 SCK, CH4 MOSI, CH5 MISO, CH6 CS
	h := 0.5e-6
	frame := 0
	for t := 0.0004; t+0.0002 < dur; t += 0.0015 {
		out := []byte{0x9F, byte(frame), 0x00, 0x00}
		in := []byte{0xFF, 0xEF, 0x40, byte(0x16 + frame%3)}
		w.set(6, t, 0)
		x := t + h
		for i := range out {
			for b := 7; b >= 0; b-- {
				w.set(4, x, (out[i]>>b)&1)
				w.set(5, x, (in[i]>>b)&1)
				w.set(3, x+h, 1)
				w.set(3, x+2*h, 0)
				x += 2 * h
			}
		}
		w.set(4, x, 0)
		w.set(5, x, 0)
		w.set(6, x+h, 1)
		frame++
	}

	// PWM 2 kHz on CH7 with a slow duty sweep
	per := 0.0005
	for t := 0.0; t < dur; t += per {
		duty := 0.5 + 0.42*math.Sin(t*2*math.Pi*3)
		w.set(7, t, 1)
		w.set(7, t+per*duty, 0)
	}

	data := make([]byte, n)
	for ch := 0; ch < 8; ch++ {
		lvl := init[ch]
		idx := 0
		mask := byte(1 << ch)
		fill := func(to int) {
			if to > n {
				to = n
			}
			if lvl == 1 {
				for i := idx; i < to; i++ {
					data[i] |= mask
				}
			}
			if to > idx {
				idx = to
			}
		}
		for _, e := range w.edges[ch] {
			fill(int(e.t * rate))
			lvl = e.v
		}
		fill(n)
	}
	return data
}
