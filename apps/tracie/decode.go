package tracie

import (
	"fmt"
	"math/bits"
	"sort"
	"strings"
)

// Protocol decoders for UART, I²C and SPI. They run over the captured
// samples (using the edge index for speed) and produce annotation rows.

type DecoderCfg struct {
	ID     string `json:"id"`
	Type   string `json:"type"` // uart, i2c, spi
	Name   string `json:"name"`
	Format string `json:"format"` // hex, dec, ascii, bin

	// UART
	Ch     int     `json:"ch"`
	Baud   float64 `json:"baud"`
	Bits   int     `json:"bits"`
	Parity string  `json:"parity"`
	Stop   float64 `json:"stop"`
	Invert bool    `json:"invert"`

	// I²C
	SCL int `json:"scl"`
	SDA int `json:"sda"`

	// SPI
	CLK      int  `json:"clk"`
	MOSI     int  `json:"mosi"`
	MISO     int  `json:"miso"`
	CS       int  `json:"cs"`
	CPOL     int  `json:"cpol"`
	CPHA     int  `json:"cpha"`
	LSB      bool `json:"lsb"`
	WordBits int  `json:"wordBits"`
	CSHigh   bool `json:"csHigh"`
}

const (
	kData = iota
	kAddr
	kCtrl
	kAck
	kErr
)

type Anno struct {
	S, E int
	T    string
	K    uint8
}

type DecRow struct {
	Dec   string
	Name  string
	Type  string
	Annos []Anno
}

const maxAnnos = 3_000_000

func runDecoders(c *Capture, cfgs []DecoderCfg) {
	var rows []*DecRow
	for _, d := range cfgs {
		name := d.Name
		switch d.Type {
		case "uart":
			if name == "" {
				name = "UART"
			}
			rows = append(rows, &DecRow{Dec: d.ID, Name: name, Type: "uart", Annos: decodeUART(c, d)})
		case "i2c":
			if name == "" {
				name = "I²C"
			}
			rows = append(rows, &DecRow{Dec: d.ID, Name: name, Type: "i2c", Annos: decodeI2C(c, d)})
		case "spi":
			if name == "" {
				name = "SPI"
			}
			mo, mi := decodeSPI(c, d)
			if d.MOSI >= 0 {
				rows = append(rows, &DecRow{Dec: d.ID, Name: name + " MOSI", Type: "spi", Annos: mo})
			}
			if d.MISO >= 0 {
				rows = append(rows, &DecRow{Dec: d.ID, Name: name + " MISO", Type: "spi", Annos: mi})
			}
		}
	}
	c.Rows = rows
}

func validCh(ch int) bool { return ch >= 0 && ch < 8 }

func fmtVal(v uint32, nbits int, format string) string {
	switch format {
	case "dec":
		return fmt.Sprintf("%d", v)
	case "bin":
		return fmt.Sprintf("%0*b", nbits, v)
	case "ascii":
		if v >= 32 && v < 127 {
			return fmt.Sprintf("'%c'", rune(v))
		}
		switch v {
		case '\n':
			return `\n`
		case '\r':
			return `\r`
		case '\t':
			return `\t`
		}
		return fmt.Sprintf("0x%02X", v)
	case "hexascii", "":
		if v >= 32 && v < 127 {
			return fmt.Sprintf("0x%02X '%c'", v, rune(v))
		}
		return fmt.Sprintf("0x%02X", v)
	}
	if nbits > 8 {
		return fmt.Sprintf("0x%0*X", (nbits+3)/4, v)
	}
	return fmt.Sprintf("0x%02X", v)
}

// ---------------------------------------------------------------- UART

func decodeUART(c *Capture, d DecoderCfg) []Anno {
	n := c.N()
	if !validCh(d.Ch) || d.Baud <= 0 {
		return nil
	}
	if d.Bits < 5 || d.Bits > 9 {
		d.Bits = 8
	}
	if d.Stop <= 0 {
		d.Stop = 1
	}
	bw := c.Rate / d.Baud
	if bw < 2.5 {
		return []Anno{{0, n - 1, fmt.Sprintf("Sample rate too low for %g baud — use at least %s", d.Baud, humanRate(d.Baud*4)), kErr}}
	}
	inv := byte(0)
	if d.Invert {
		inv = 1
	}
	bit := func(i int) byte { return c.bit(d.Ch, i) ^ inv }
	edges := c.Edges[d.Ch]
	var out []Anno
	pos := 0
	nb := 1 + d.Bits
	if d.Parity == "odd" || d.Parity == "even" {
		nb++
	}
	for len(out) < maxAnnos {
		k := countLE(edges, pos-1)
		st := -1
		for ; k < len(edges); k++ {
			e := int(edges[k])
			if bit(e) == 0 {
				st = e
				break
			}
		}
		if st < 0 {
			break
		}
		at := func(b float64) int { return st + int(bw*b) }
		if at(0.5) >= n {
			break
		}
		if bit(at(0.5)) != 0 { // glitch, not a start bit
			pos = st + 1
			continue
		}
		var v uint32
		for j := 0; j < d.Bits; j++ {
			if bit(at(1.5+float64(j))) == 1 {
				v |= 1 << j
			}
		}
		perr := false
		if d.Parity == "odd" || d.Parity == "even" {
			p := int(bit(at(1.5 + float64(d.Bits))))
			ones := bits.OnesCount32(v) + p
			perr = (d.Parity == "even" && ones%2 != 0) || (d.Parity == "odd" && ones%2 == 0)
		}
		sp := at(float64(nb) + 0.5)
		if sp >= n {
			break
		}
		ferr := bit(sp) != 1
		end := at(float64(nb) + d.Stop)
		if end >= n {
			end = n - 1
		}
		t := fmtVal(v, d.Bits, d.Format)
		k8 := uint8(kData)
		switch {
		case ferr:
			t += " ⚠ framing"
			k8 = kErr
		case perr:
			t += " ⚠ parity"
			k8 = kErr
		}
		out = append(out, Anno{st, end, t, k8})
		if ferr {
			pos = sp
		} else {
			pos = at(float64(nb) + 0.5)
		}
	}
	return out
}

// ---------------------------------------------------------------- I²C

func decodeI2C(c *Capture, d DecoderCfg) []Anno {
	if !validCh(d.SCL) || !validCh(d.SDA) || d.SCL == d.SDA {
		return nil
	}
	var out []Anno
	pscl, psda := c.bit(d.SCL, 0), c.bit(d.SDA, 0)
	inTx, first := false, false
	nbits := 0
	var val uint32
	byteStart, lastRise := 0, 0
	hp := int(c.Rate / 400000)
	if hp < 1 {
		hp = 1
	}
	for _, ci := range c.Changes {
		if len(out) >= maxAnnos {
			break
		}
		i := int(ci)
		ns, nd := c.bit(d.SCL, i), c.bit(d.SDA, i)
		if ns == pscl && nd == psda {
			continue
		}
		switch {
		case ns == 1 && pscl == 1 && nd != psda:
			if nd == 0 {
				t := "Start"
				if inTx {
					t = "Restart"
				}
				out = append(out, Anno{i, i + hp, t, kCtrl})
				inTx, first, nbits, val = true, true, 0, 0
			} else {
				out = append(out, Anno{i, i + hp, "Stop", kCtrl})
				inTx, nbits, val = false, 0, 0
			}
		case ns == 1 && pscl == 0 && inTx:
			if nbits < 8 {
				if nbits == 0 {
					byteStart = i
				}
				val = val<<1 | uint32(nd)
				nbits++
				lastRise = i
			} else {
				if lastRise > byteStart {
					hp = (lastRise - byteStart) / 14
					if hp < 1 {
						hp = 1
					}
				}
				if first {
					rw := "Write"
					if val&1 == 1 {
						rw = "Read"
					}
					out = append(out, Anno{byteStart - hp, lastRise + hp, fmt.Sprintf("%s 0x%02X", rw, val>>1), kAddr})
					first = false
				} else {
					out = append(out, Anno{byteStart - hp, lastRise + hp, fmtVal(val, 8, d.Format), kData})
				}
				if nd == 0 {
					out = append(out, Anno{i - hp, i + hp, "ACK", kAck})
				} else {
					out = append(out, Anno{i - hp, i + hp, "NACK", kErr})
				}
				nbits, val = 0, 0
			}
		}
		pscl, psda = ns, nd
	}
	return out
}

// ---------------------------------------------------------------- SPI

func decodeSPI(c *Capture, d DecoderCfg) (mosi, miso []Anno) {
	if !validCh(d.CLK) {
		return nil, nil
	}
	wb := d.WordBits
	if wb < 4 || wb > 32 {
		wb = 8
	}
	if !validCh(d.MOSI) {
		d.MOSI = -1
	}
	if !validCh(d.MISO) {
		d.MISO = -1
	}
	if !validCh(d.CS) {
		d.CS = -1
	}
	sampleRising := d.CPOL == d.CPHA
	active := func(i int) bool {
		if d.CS < 0 {
			return true
		}
		v := c.bit(d.CS, i)
		if d.CSHigh {
			return v == 1
		}
		return v == 0
	}
	pclk := c.bit(d.CLK, 0)
	pact := active(0)
	n := 0
	var vo, vi uint32
	start := 0
	for _, ci := range c.Changes {
		if len(mosi) >= maxAnnos {
			break
		}
		i := int(ci)
		act := active(i)
		if act != pact {
			n, vo, vi = 0, 0, 0
			pact = act
		}
		nclk := c.bit(d.CLK, i)
		if nclk == pclk {
			continue
		}
		pclk = nclk
		if !act || (nclk == 1) != sampleRising {
			continue
		}
		if n == 0 {
			start = i
		}
		var bo, bi uint32
		if d.MOSI >= 0 {
			bo = uint32(c.bit(d.MOSI, i))
		}
		if d.MISO >= 0 {
			bi = uint32(c.bit(d.MISO, i))
		}
		if d.LSB {
			vo |= bo << n
			vi |= bi << n
		} else {
			vo = vo<<1 | bo
			vi = vi<<1 | bi
		}
		n++
		if n == wb {
			half := (i - start) / (2 * (wb - 1))
			if half < 1 {
				half = 1
			}
			mosi = append(mosi, Anno{start - half, i + half, fmtVal(vo, wb, d.Format), kData})
			miso = append(miso, Anno{start - half, i + half, fmtVal(vi, wb, d.Format), kData})
			n, vo, vi = 0, 0, 0
		}
	}
	return
}

// ---------------------------------------------------------------- views

type rowInfo struct {
	Dec   string `json:"dec"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	Count int    `json:"count"`
}

func (c *Capture) rowInfo() []rowInfo {
	out := []rowInfo{}
	for _, r := range c.Rows {
		out = append(out, rowInfo{r.Dec, r.Name, r.Type, len(r.Annos)})
	}
	return out
}

// annos returns visible annotations per row as [s,e,kind,merged,text];
// anything narrower than a few pixels is merged into "busy" blocks.
func (c *Capture) annos(s, e float64, px int) [][][]any {
	spp := (e - s) / float64(px)
	out := make([][][]any, len(c.Rows))
	for ri, r := range c.Rows {
		list := r.Annos
		k := sort.Search(len(list), func(i int) bool { return float64(list[i].E) >= s })
		row := [][]any{}
		for ; k < len(list) && float64(list[k].S) <= e && len(row) < 6000; k++ {
			a := list[k]
			if float64(a.E-a.S)/spp < 3 {
				if l := len(row); l > 0 && row[l-1][3].(int) > 0 && float64(a.S-row[l-1][1].(int))/spp < 2 {
					row[l-1][1] = a.E
					row[l-1][3] = row[l-1][3].(int) + 1
					if a.K == kErr {
						row[l-1][2] = kErr
					}
					continue
				}
				row = append(row, []any{a.S, a.E, int(a.K), 1, ""})
				continue
			}
			row = append(row, []any{a.S, a.E, int(a.K), 0, a.T})
		}
		out[ri] = row
	}
	return out
}

// table lists decoded items in time order for the side panel.
func (c *Capture) table(limit int) [][]any {
	type item struct {
		r int
		a Anno
	}
	var all []item
	for ri, r := range c.Rows {
		for _, a := range r.Annos {
			if a.K == kAck {
				continue
			}
			all = append(all, item{ri, a})
			if len(all) > limit*4 {
				break
			}
		}
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].a.S < all[j].a.S })
	if len(all) > limit {
		all = all[:limit]
	}
	out := make([][]any, len(all))
	for i, it := range all {
		out[i] = []any{it.r, it.a.S, it.a.E, int(it.a.K), strings.TrimSpace(it.a.T)}
	}
	return out
}
