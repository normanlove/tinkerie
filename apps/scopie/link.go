// Message transports for instruments that speak SCPI text commands:
//
//   - VXI-11 over the network (what "TCPIP::<ip>::INSTR" means in VISA),
//   - a raw TCP socket (ports 5025/5555, or "ip:port"),
//   - USBTMC over WinUSB,
//   - the installed VISA library (NI-VISA / Keysight IO), when present,
//   - plain USB bulk pipes (OWON handhelds).
//
// No driver or library is needed for the network.
package scopie

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

type link interface {
	Open() error
	Close()
	Write(b []byte) error
	// Read returns the next chunk of a reply and whether the message ended.
	Read(timeoutMs int) ([]byte, bool, error)
	Name() string
}

// ---------------------------------------------------------------- helpers

func lWrite(l link, cmd string) error {
	if !strings.HasSuffix(cmd, "\n") {
		cmd += "\n"
	}
	return l.Write([]byte(cmd))
}

// lReadMsg reads one complete text reply.
func lReadMsg(l link, timeoutMs int) ([]byte, error) {
	var buf []byte
	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	for {
		left := int(time.Until(deadline) / time.Millisecond)
		if left < 1 {
			if len(buf) > 0 {
				return buf, nil
			}
			return nil, errors.New("timed out waiting for a reply")
		}
		b, eom, err := l.Read(left)
		buf = append(buf, b...)
		if err != nil {
			if len(buf) > 0 && bytes.HasSuffix(buf, []byte("\n")) {
				return buf, nil
			}
			return buf, err
		}
		if eom || bytes.HasSuffix(buf, []byte("\n")) {
			return buf, nil
		}
	}
}

func lQuery(l link, cmd string, timeoutMs int) (string, error) {
	if err := lWrite(l, cmd); err != nil {
		return "", err
	}
	b, err := lReadMsg(l, timeoutMs)
	if err != nil {
		return "", err
	}
	return strings.Trim(string(b), "\x00\r\n "), nil
}

var blockRe = regexp.MustCompile(`#(\d)`)

// lReadBlock reads an IEEE-488.2 definite-length block reply '#<n><len><data>'.
func lReadBlock(l link, timeoutMs int) ([]byte, error) {
	var raw []byte
	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	need := -1
	for {
		left := int(time.Until(deadline) / time.Millisecond)
		if left < 1 {
			return nil, errors.New("timed out while reading waveform data")
		}
		b, eom, err := l.Read(left)
		raw = append(raw, b...)
		if err != nil {
			return nil, err
		}
		if need < 0 {
			if m := blockRe.FindSubmatchIndex(raw); m != nil {
				nd := int(raw[m[2]] - '0')
				if nd == 0 {
					need = 0
				} else if len(raw) >= m[1]+nd {
					ln, e := strconv.Atoi(string(raw[m[1] : m[1]+nd]))
					if e != nil {
						return nil, devErr("The scope sent waveform data in an unexpected format.")
					}
					need = m[1] + nd + ln
				}
			}
		}
		if need > 0 && len(raw) >= need {
			return parseBlock(raw)
		}
		if eom && (need <= 0 || len(raw) >= need) {
			return parseBlock(raw)
		}
		if eom && need > 0 && len(raw) < need {
			continue // message-based links may split big blocks
		}
	}
}

// parseBlock extracts the payload of an IEEE-488.2 block.
func parseBlock(raw []byte) ([]byte, error) {
	i := bytes.IndexByte(raw, '#')
	if i < 0 || i+1 >= len(raw) {
		return nil, devErr("The scope sent waveform data in an unexpected format.")
	}
	nd := int(raw[i+1] - '0')
	if nd < 0 || nd > 9 {
		return nil, devErr("The scope sent waveform data in an unexpected format.")
	}
	if nd == 0 {
		return bytes.TrimRight(raw[i+2:], "\n"), nil
	}
	if i+2+nd > len(raw) {
		return nil, devErr("The scope sent a short reply.")
	}
	ln, err := strconv.Atoi(string(raw[i+2 : i+2+nd]))
	if err != nil {
		return nil, devErr("The scope sent waveform data in an unexpected format.")
	}
	start := i + 2 + nd
	end := start + ln
	if end > len(raw) {
		end = len(raw)
	}
	return raw[start:end], nil
}

// lReadLen4 reads a reply with a 4-byte little-endian length prefix (OWON).
func lReadLen4(l link, timeoutMs int) ([]byte, error) {
	var buf []byte
	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	for {
		left := int(time.Until(deadline) / time.Millisecond)
		if left < 1 {
			return nil, devErr("The scope did not answer in time.")
		}
		b, _, err := l.Read(left)
		if err != nil {
			return nil, devErr("The scope did not answer (%v).", err)
		}
		buf = append(buf, b...)
		if len(buf) >= 4 {
			n := int(binary.LittleEndian.Uint32(buf))
			if n > 50_000_000 {
				return nil, devErr("The scope sent an implausible reply.")
			}
			if len(buf) >= n+4 {
				return buf[4 : 4+n], nil
			}
		}
	}
}

// ---------------------------------------------------------------- addresses

// netAddr parses what users type for a network scope: "192.168.1.50",
// "192.168.1.50:5555", "TCPIP::192.168.1.50::INSTR", "TCPIP0::host::5025::SOCKET".
func netAddr(s string) (host string, port int, ok bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", 0, false
	}
	up := strings.ToUpper(s)
	if strings.HasPrefix(up, "TCPIP") {
		parts := strings.Split(s, "::")
		if len(parts) < 2 {
			return "", 0, false
		}
		host = parts[1]
		if len(parts) >= 4 && strings.EqualFold(parts[len(parts)-1], "SOCKET") {
			port, _ = strconv.Atoi(parts[2])
		}
		return host, port, true
	}
	if h, p, err := net.SplitHostPort(s); err == nil {
		port, _ = strconv.Atoi(p)
		return h, port, true
	}
	return s, 0, true
}

// openNetLink finds a way to talk to a network instrument: VXI-11 first,
// then raw sockets.
func openNetLink(addr string, probe func(link) string) (link, string, error) {
	host, port, ok := netAddr(addr)
	if !ok {
		return nil, "", fmt.Errorf("bad address %q", addr)
	}
	var tries []link
	if port > 0 {
		tries = append(tries, &rawLink{host: host, port: port})
	} else {
		tries = append(tries, &vxiLink{host: host}, &rawLink{host: host, port: 5025}, &rawLink{host: host, port: 5555})
	}
	var lastErr error
	for _, l := range tries {
		if err := l.Open(); err != nil {
			lastErr = err
			continue
		}
		if idn := probe(l); idn != "" {
			return l, idn, nil
		}
		l.Close()
	}
	if lastErr == nil {
		lastErr = errors.New("no answer")
	}
	return nil, "", lastErr
}

// ---------------------------------------------------------------- raw socket

type rawLink struct {
	host string
	port int
	c    net.Conn
}

func (r *rawLink) Name() string { return fmt.Sprintf("TCPIP::%s::%d::SOCKET", r.host, r.port) }

func (r *rawLink) Open() error {
	c, err := net.DialTimeout("tcp", net.JoinHostPort(r.host, strconv.Itoa(r.port)), 1500*time.Millisecond)
	if err != nil {
		return err
	}
	r.c = c
	return nil
}

func (r *rawLink) Close() {
	if r.c != nil {
		r.c.Close()
		r.c = nil
	}
}

func (r *rawLink) Write(b []byte) error {
	if r.c == nil {
		return errors.New("not connected")
	}
	// drop anything left over from an earlier reply (e.g. a block's trailing newlines)
	_ = r.c.SetReadDeadline(time.Now().Add(time.Millisecond))
	junk := make([]byte, 4096)
	for {
		n, err := r.c.Read(junk)
		if n == 0 || err != nil {
			break
		}
	}
	_ = r.c.SetWriteDeadline(time.Now().Add(3 * time.Second))
	_, err := r.c.Write(b)
	return err
}

func (r *rawLink) Read(timeoutMs int) ([]byte, bool, error) {
	if r.c == nil {
		return nil, false, errors.New("not connected")
	}
	_ = r.c.SetReadDeadline(time.Now().Add(time.Duration(timeoutMs) * time.Millisecond))
	buf := make([]byte, 1<<20)
	n, err := r.c.Read(buf)
	if err != nil && n == 0 {
		return nil, false, err
	}
	return buf[:n], false, nil
}

// ---------------------------------------------------------------- VXI-11

const (
	vxiCore   = 0x0607AF
	rpcPmap   = 100000
	vxiEndBit = 8
)

type vxiLink struct {
	host  string
	c     net.Conn
	xid   uint32
	lid   uint32
	maxRx uint32
}

var vxiClientID int32

// vxiPortmap is the ONC-RPC portmapper port (tests point it elsewhere).
var vxiPortmap = "111"

func (v *vxiLink) Name() string { return fmt.Sprintf("TCPIP::%s::INSTR", v.host) }

type xdr struct{ bytes.Buffer }

func (x *xdr) u32(v uint32) { _ = binary.Write(&x.Buffer, binary.BigEndian, v) }
func (x *xdr) opaque(b []byte) {
	x.u32(uint32(len(b)))
	x.Write(b)
	for i := len(b); i%4 != 0; i++ {
		x.WriteByte(0)
	}
}

type xdrR struct {
	b []byte
	p int
}

func (x *xdrR) u32() uint32 {
	if x.p+4 > len(x.b) {
		x.p = len(x.b)
		return 0
	}
	v := binary.BigEndian.Uint32(x.b[x.p:])
	x.p += 4
	return v
}

func (x *xdrR) opaque() []byte {
	n := int(x.u32())
	if x.p+n > len(x.b) {
		n = len(x.b) - x.p
	}
	b := x.b[x.p : x.p+n]
	x.p += (n + 3) &^ 3
	if x.p > len(x.b) {
		x.p = len(x.b)
	}
	return b
}

// rpcCall sends one ONC-RPC call over a TCP connection and returns the results.
func rpcCall(c net.Conn, xid *uint32, prog, vers, proc uint32, args []byte, timeout time.Duration) ([]byte, error) {
	*xid++
	var m xdr
	m.u32(*xid)
	m.u32(0) // CALL
	m.u32(2) // RPC version
	m.u32(prog)
	m.u32(vers)
	m.u32(proc)
	m.u32(0)
	m.u32(0) // AUTH_NULL credentials
	m.u32(0)
	m.u32(0) // AUTH_NULL verifier
	m.Write(args)
	hdr := make([]byte, 4)
	binary.BigEndian.PutUint32(hdr, 0x80000000|uint32(m.Len()))
	_ = c.SetDeadline(time.Now().Add(timeout))
	if _, err := c.Write(append(hdr, m.Bytes()...)); err != nil {
		return nil, err
	}
	for {
		var rec []byte
		for {
			if _, err := io.ReadFull(c, hdr); err != nil {
				return nil, err
			}
			n := binary.BigEndian.Uint32(hdr)
			frag := make([]byte, n&0x7FFFFFFF)
			if _, err := io.ReadFull(c, frag); err != nil {
				return nil, err
			}
			rec = append(rec, frag...)
			if n&0x80000000 != 0 {
				break
			}
		}
		r := &xdrR{b: rec}
		if r.u32() != *xid {
			continue // a stale reply
		}
		if r.u32() != 1 || r.u32() != 0 { // REPLY, MSG_ACCEPTED
			return nil, errors.New("RPC call rejected")
		}
		r.u32()
		r.opaque() // verifier
		if st := r.u32(); st != 0 {
			return nil, fmt.Errorf("RPC error %d", st)
		}
		return rec[r.p:], nil
	}
}

func (v *vxiLink) Open() error {
	// ask the portmapper where the VXI-11 core channel is
	pm, err := net.DialTimeout("tcp", net.JoinHostPort(v.host, vxiPortmap), 1500*time.Millisecond)
	if err != nil {
		return err
	}
	var a xdr
	a.u32(vxiCore)
	a.u32(1)
	a.u32(6) // TCP
	a.u32(0)
	var xid uint32 = uint32(time.Now().UnixNano())
	res, err := rpcCall(pm, &xid, rpcPmap, 2, 3, a.Bytes(), 2*time.Second)
	pm.Close()
	if err != nil {
		return err
	}
	port := (&xdrR{b: res}).u32()
	if port == 0 {
		return errors.New("the instrument has no VXI-11 service")
	}
	c, err := net.DialTimeout("tcp", net.JoinHostPort(v.host, strconv.Itoa(int(port))), 1500*time.Millisecond)
	if err != nil {
		return err
	}
	v.c, v.xid = c, xid
	var cl xdr
	cl.u32(uint32(atomic.AddInt32(&vxiClientID, 1)))
	cl.u32(0)    // no lock
	cl.u32(1000) // lock timeout
	cl.opaque([]byte("inst0"))
	res, err = rpcCall(c, &v.xid, vxiCore, 1, 10, cl.Bytes(), 3*time.Second)
	if err != nil {
		c.Close()
		v.c = nil
		return err
	}
	r := &xdrR{b: res}
	if e := r.u32(); e != 0 {
		c.Close()
		v.c = nil
		return fmt.Errorf("VXI-11 create_link error %d", e)
	}
	v.lid = r.u32()
	r.u32() // abort port
	v.maxRx = r.u32()
	if v.maxRx == 0 {
		v.maxRx = 1024
	}
	return nil
}

func (v *vxiLink) Close() {
	if v.c == nil {
		return
	}
	var a xdr
	a.u32(v.lid)
	_, _ = rpcCall(v.c, &v.xid, vxiCore, 1, 23, a.Bytes(), time.Second)
	v.c.Close()
	v.c = nil
}

func (v *vxiLink) Write(b []byte) error {
	if v.c == nil {
		return errors.New("not connected")
	}
	for len(b) > 0 {
		n := minInt(len(b), int(v.maxRx))
		var a xdr
		a.u32(v.lid)
		a.u32(3000) // io timeout ms
		a.u32(1000) // lock timeout
		flags := uint32(0)
		if n == len(b) {
			flags = vxiEndBit
		}
		a.u32(flags)
		a.opaque(b[:n])
		res, err := rpcCall(v.c, &v.xid, vxiCore, 1, 11, a.Bytes(), 5*time.Second)
		if err != nil {
			return err
		}
		if e := (&xdrR{b: res}).u32(); e != 0 {
			return fmt.Errorf("VXI-11 write error %d", e)
		}
		b = b[n:]
	}
	return nil
}

func (v *vxiLink) Read(timeoutMs int) ([]byte, bool, error) {
	if v.c == nil {
		return nil, false, errors.New("not connected")
	}
	var a xdr
	a.u32(v.lid)
	a.u32(1 << 20)           // request size
	a.u32(uint32(timeoutMs)) // io timeout
	a.u32(1000)              // lock timeout
	a.u32(0)                 // flags: no term char
	a.u32(0)
	res, err := rpcCall(v.c, &v.xid, vxiCore, 1, 12, a.Bytes(), time.Duration(timeoutMs+2000)*time.Millisecond)
	if err != nil {
		return nil, false, err
	}
	r := &xdrR{b: res}
	if e := r.u32(); e != 0 {
		if e == 15 { // I/O timeout
			return nil, false, errors.New("timed out")
		}
		return nil, false, fmt.Errorf("VXI-11 read error %d", e)
	}
	reason := r.u32()
	data := append([]byte(nil), r.opaque()...)
	return data, reason&4 != 0, nil
}

// ---------------------------------------------------------------- USBTMC (WinUSB)

type usbtmcLink struct {
	u       usbInfo
	d       usbDevice
	in, out uint8
	tag     uint8
}

func (t *usbtmcLink) Name() string {
	return fmt.Sprintf("USB::0x%04X::0x%04X::%s::INSTR", t.u.VID, t.u.PID, t.u.Serial)
}

func (t *usbtmcLink) Open() error {
	if t.d != nil {
		return nil
	}
	d, err := openUSB(&t.u)
	if err != nil {
		return err
	}
	in, out, _, _ := d.Endpoints()
	if len(in) == 0 || len(out) == 0 {
		d.Close()
		return devErr("The instrument's USB interface has no data pipes.")
	}
	t.d, t.in, t.out = d, in[0], out[0]
	d.ResetPipe(t.in)
	d.ResetPipe(t.out)
	return nil
}

func (t *usbtmcLink) Close() {
	if t.d != nil {
		t.d.Close()
		t.d = nil
	}
}

func (t *usbtmcLink) nextTag() uint8 {
	t.tag++
	if t.tag == 0 {
		t.tag = 1
	}
	return t.tag
}

func tmcHeader(id, tag uint8, size uint32, attr, term uint8) []byte {
	h := make([]byte, 12)
	h[0], h[1], h[2] = id, tag, ^tag
	binary.LittleEndian.PutUint32(h[4:], size)
	h[8], h[9] = attr, term
	return h
}

func (t *usbtmcLink) Write(b []byte) error {
	if t.d == nil {
		return errors.New("not connected")
	}
	msg := append(tmcHeader(1, t.nextTag(), uint32(len(b)), 1, 0), b...)
	for len(msg)%4 != 0 {
		msg = append(msg, 0)
	}
	_, err := t.d.BulkWrite(t.out, msg, 3000)
	return err
}

func (t *usbtmcLink) Read(timeoutMs int) ([]byte, bool, error) {
	if t.d == nil {
		return nil, false, errors.New("not connected")
	}
	const max = 1 << 20
	if _, err := t.d.BulkWrite(t.out, tmcHeader(2, t.nextTag(), max, 0, 0), 2000); err != nil {
		return nil, false, err
	}
	buf := make([]byte, max+64)
	n, err := t.d.BulkRead(t.in, buf, timeoutMs)
	if err != nil {
		return nil, false, err
	}
	if n < 12 || buf[0] != 2 {
		return nil, false, errors.New("bad USBTMC reply")
	}
	size := int(binary.LittleEndian.Uint32(buf[4:]))
	eom := buf[8]&1 != 0
	got := buf[:n]
	for len(got) < 12+size {
		k, err := t.d.BulkRead(t.in, buf[len(got):], timeoutMs)
		if err != nil {
			return nil, false, err
		}
		if k == 0 {
			break
		}
		got = buf[:len(got)+k]
	}
	end := minInt(len(got), 12+size)
	return append([]byte(nil), got[12:end]...), eom, nil
}

// ---------------------------------------------------------------- plain bulk pipes

type bulkLink struct {
	u       usbInfo
	d       usbDevice
	in, out uint8
}

func (b *bulkLink) Name() string { return fmt.Sprintf("USB %04X:%04X", b.u.VID, b.u.PID) }

func (b *bulkLink) Open() error {
	d, err := openUSB(&b.u)
	if err != nil {
		return err
	}
	b.d, b.in, b.out = d, 0x81, 0x01
	in, out, _, _ := d.Endpoints()
	if len(in) > 0 {
		b.in = in[0]
	}
	if len(out) > 0 {
		b.out = out[0]
	}
	return nil
}

func (b *bulkLink) Close() {
	if b.d != nil {
		b.d.Close()
		b.d = nil
	}
}

func (b *bulkLink) Write(p []byte) error {
	if b.d == nil {
		return errors.New("not connected")
	}
	_, err := b.d.BulkWrite(b.out, p, 2000)
	return err
}

func (b *bulkLink) Read(timeoutMs int) ([]byte, bool, error) {
	if b.d == nil {
		return nil, false, errors.New("not connected")
	}
	buf := make([]byte, 0x10000)
	n, err := b.d.BulkRead(b.in, buf, timeoutMs)
	if err != nil {
		return nil, false, err
	}
	return buf[:n], n < len(buf), nil
}
