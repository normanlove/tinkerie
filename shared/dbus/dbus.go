//go:build linux

// Package dbus is a small D-Bus client in plain Go, enough for Tinkerie to
// talk to BlueZ (Linux's Bluetooth service) on the system bus: method calls
// with typed arguments, replies decoded into Go values, and signals.
//
// Decoded values: y→byte, b→bool, n/i/x→int64, q/u/t→uint64, d→float64,
// s/o/g→string, ay→[]byte, a{..}→map[string]any (keys as strings),
// other arrays→[]any, structs→[]any, v→the value inside.
// Created by nrmluvdev for Tinkerie.
package dbus

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Conn is a connection to a message bus.
type Conn struct {
	c       net.Conn
	mu      sync.Mutex // writes
	serial  uint32
	pending map[uint32]chan *Message
	pmu     sync.Mutex
	signals []chan *Message
	calls   chan *Message // method calls made to us (for small services and tests)
	closed  chan struct{}
	err     error
}

// Message is a received message.
type Message struct {
	Type        byte // 1 call, 2 return, 3 error, 4 signal
	Serial      uint32
	ReplySerial uint32
	Path        string
	Interface   string
	Member      string
	ErrorName   string
	Sender      string
	Signature   string
	Body        []any
}

// SystemBus connects to the system bus (where BlueZ lives).
func SystemBus() (*Conn, error) {
	addr := os.Getenv("DBUS_SYSTEM_BUS_ADDRESS")
	path := "/var/run/dbus/system_bus_socket"
	if p, ok := strings.CutPrefix(addr, "unix:path="); ok {
		path, _, _ = strings.Cut(p, ",")
	}
	c, err := net.DialTimeout("unix", path, 3*time.Second)
	if err != nil {
		return nil, fmt.Errorf("can't reach the system's message bus: %v", err)
	}
	// authenticate as this user
	uid := hex.EncodeToString([]byte(strconv.Itoa(os.Getuid())))
	c.SetDeadline(time.Now().Add(5 * time.Second))
	if _, err := c.Write([]byte("\x00AUTH EXTERNAL " + uid + "\r\n")); err != nil {
		c.Close()
		return nil, err
	}
	r := bufio.NewReader(c)
	line, err := r.ReadString('\n')
	if err != nil || !strings.HasPrefix(line, "OK") {
		c.Close()
		return nil, fmt.Errorf("the message bus didn't accept us: %q %v", line, err)
	}
	if _, err := c.Write([]byte("BEGIN\r\n")); err != nil {
		c.Close()
		return nil, err
	}
	c.SetDeadline(time.Time{})
	conn := &Conn{c: c, pending: map[uint32]chan *Message{}, closed: make(chan struct{}), calls: make(chan *Message, 64)}
	go conn.readLoop(r)
	if _, err := conn.Call("org.freedesktop.DBus", "/org/freedesktop/DBus", "org.freedesktop.DBus", "Hello", ""); err != nil {
		conn.Close()
		return nil, err
	}
	return conn, nil
}

// Close ends the connection.
func (c *Conn) Close() error {
	select {
	case <-c.closed:
	default:
		close(c.closed)
	}
	return c.c.Close()
}

// Signals returns a channel receiving every signal (after AddMatch).
func (c *Conn) Signals() <-chan *Message {
	ch := make(chan *Message, 256)
	c.pmu.Lock()
	c.signals = append(c.signals, ch)
	c.pmu.Unlock()
	return ch
}

// Calls returns method calls made to this connection.
func (c *Conn) Calls() <-chan *Message { return c.calls }

// RequestName asks to own a bus name.
func (c *Conn) RequestName(name string) error {
	_, err := c.Call("org.freedesktop.DBus", "/org/freedesktop/DBus", "org.freedesktop.DBus", "RequestName", "su", name, uint32(4))
	return err
}

func (c *Conn) send(typ byte, f fields, sig string, args []any) error {
	var body encoder
	if err := body.values(sig, args); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.serial++
	f.sig = sig
	msg := append(f.header(typ, c.serial, body.b.Len()), body.b.Bytes()...)
	_, err := c.c.Write(msg)
	return err
}

// Reply answers a method call.
func (c *Conn) Reply(to *Message, sig string, args ...any) error {
	return c.send(2, fields{replySerial: to.Serial, dest: to.Sender}, sig, args)
}

// ReplyError answers a method call with an error.
func (c *Conn) ReplyError(to *Message, name, text string) error {
	return c.send(3, fields{replySerial: to.Serial, dest: to.Sender, errName: name}, "s", []any{text})
}

// Emit sends a signal.
func (c *Conn) Emit(path, iface, member, sig string, args ...any) error {
	return c.send(4, fields{path: path, iface: iface, member: member}, sig, args)
}

// AddMatch asks the bus to send us signals matching rule.
func (c *Conn) AddMatch(rule string) error {
	_, err := c.Call("org.freedesktop.DBus", "/org/freedesktop/DBus", "org.freedesktop.DBus", "AddMatch", "s", rule)
	return err
}

// Error is a D-Bus error reply.
type Error struct {
	Name, Text string
}

func (e *Error) Error() string {
	if e.Text != "" {
		return e.Text
	}
	return e.Name
}

// Call calls a method and waits up to 30 s for its reply body.
func (c *Conn) Call(dest, path, iface, member, sig string, args ...any) ([]any, error) {
	return c.CallTimeout(30*time.Second, dest, path, iface, member, sig, args...)
}

// CallTimeout is Call with a chosen timeout.
func (c *Conn) CallTimeout(d time.Duration, dest, path, iface, member, sig string, args ...any) ([]any, error) {
	var body encoder
	if err := body.values(sig, args); err != nil {
		return nil, err
	}
	c.mu.Lock()
	c.serial++
	serial := c.serial
	ch := make(chan *Message, 1)
	c.pmu.Lock()
	c.pending[serial] = ch
	c.pmu.Unlock()
	msg := fields{path: path, iface: iface, member: member, dest: dest, sig: sig}.header(1, serial, body.b.Len())
	msg = append(msg, body.b.Bytes()...)
	_, err := c.c.Write(msg)
	c.mu.Unlock()
	defer func() {
		c.pmu.Lock()
		delete(c.pending, serial)
		c.pmu.Unlock()
	}()
	if err != nil {
		return nil, err
	}
	select {
	case m := <-ch:
		if m.Type == 3 {
			e := &Error{Name: m.ErrorName}
			if len(m.Body) > 0 {
				e.Text, _ = m.Body[0].(string)
			}
			return nil, e
		}
		return m.Body, nil
	case <-c.closed:
		return nil, errors.New("the message bus connection closed")
	case <-time.After(d):
		return nil, fmt.Errorf("%s.%s didn't answer", iface, member)
	}
}

func (c *Conn) readLoop(r *bufio.Reader) {
	defer c.Close()
	for {
		m, err := readMessage(r)
		if err != nil {
			c.err = err
			return
		}
		switch m.Type {
		case 2, 3:
			c.pmu.Lock()
			ch := c.pending[m.ReplySerial]
			c.pmu.Unlock()
			if ch != nil {
				ch <- m
			}
		case 1:
			select {
			case c.calls <- m:
			default:
			}
		case 4:
			c.pmu.Lock()
			for _, s := range c.signals {
				select {
				case s <- m:
				default:
				}
			}
			c.pmu.Unlock()
		}
	}
}

// ---------------------------------------------------------------- encoding

type encoder struct{ b bytes.Buffer }

func (e *encoder) pad(n int) {
	for e.b.Len()%n != 0 {
		e.b.WriteByte(0)
	}
}

func (e *encoder) u32(v uint32) {
	e.pad(4)
	binary.Write(&e.b, binary.LittleEndian, v)
}

func (e *encoder) str(s string) {
	e.u32(uint32(len(s)))
	e.b.WriteString(s)
	e.b.WriteByte(0)
}

func (e *encoder) sig(s string) {
	e.b.WriteByte(byte(len(s)))
	e.b.WriteString(s)
	e.b.WriteByte(0)
}

// Variant wraps a value with its signature, for "v" and a{sv} arguments.
type Variant struct {
	Sig   string
	Value any
}

func (e *encoder) values(sig string, args []any) error {
	types := splitSig(sig)
	if len(types) != len(args) {
		return fmt.Errorf("signature %q wants %d values, got %d", sig, len(types), len(args))
	}
	for i, t := range types {
		if err := e.value(t, args[i]); err != nil {
			return err
		}
	}
	return nil
}

func (e *encoder) value(t string, v any) error {
	bad := fmt.Errorf("can't send %T as %q", v, t)
	switch t[0] {
	case 'y':
		b, ok := v.(byte)
		if !ok {
			return bad
		}
		e.b.WriteByte(b)
	case 'b':
		b, ok := v.(bool)
		if !ok {
			return bad
		}
		n := uint32(0)
		if b {
			n = 1
		}
		e.u32(n)
	case 'u', 'i':
		n, ok := toU64(v)
		if !ok {
			return bad
		}
		e.u32(uint32(n))
	case 'q', 'n':
		n, ok := toU64(v)
		if !ok {
			return bad
		}
		e.pad(2)
		binary.Write(&e.b, binary.LittleEndian, uint16(n))
	case 't', 'x':
		n, ok := toU64(v)
		if !ok {
			return bad
		}
		e.pad(8)
		binary.Write(&e.b, binary.LittleEndian, n)
	case 'd':
		f, ok := v.(float64)
		if !ok {
			return bad
		}
		e.pad(8)
		binary.Write(&e.b, binary.LittleEndian, math.Float64bits(f))
	case 's', 'o':
		s, ok := v.(string)
		if !ok {
			return bad
		}
		e.str(s)
	case 'g':
		s, ok := v.(string)
		if !ok {
			return bad
		}
		e.sig(s)
	case 'v':
		vv, ok := v.(Variant)
		if !ok {
			return bad
		}
		e.sig(vv.Sig)
		return e.value(vv.Sig, vv.Value)
	case 'a':
		elem := t[1:]
		e.u32(0) // length, filled in below
		lenAt := e.b.Len() - 4
		e.pad(align(elem))
		start := e.b.Len()
		switch x := v.(type) {
		case []byte:
			e.b.Write(x)
		case []string:
			for _, s := range x {
				if err := e.value(elem, s); err != nil {
					return err
				}
			}
		case map[string]any: // a{s…} / a{o…} with any value type
			kt, vt := dictTypes(elem)
			for k, val := range x {
				e.pad(8)
				if err := e.value(kt, k); err != nil {
					return err
				}
				if err := e.value(vt, val); err != nil {
					return err
				}
			}
		case map[string]Variant: // a{sv}
			kt, vt := dictTypes(elem)
			for k, val := range x {
				e.pad(8)
				if err := e.value(kt, k); err != nil {
					return err
				}
				if err := e.value(vt, val); err != nil {
					return err
				}
			}
		case []any:
			for _, it := range x {
				if err := e.value(elem, it); err != nil {
					return err
				}
			}
		default:
			return bad
		}
		binary.LittleEndian.PutUint32(e.b.Bytes()[lenAt:], uint32(e.b.Len()-start))
	default:
		return bad
	}
	return nil
}

func toU64(v any) (uint64, bool) {
	switch n := v.(type) {
	case int:
		return uint64(n), true
	case int16:
		return uint64(n), true
	case int32:
		return uint64(n), true
	case int64:
		return uint64(n), true
	case uint16:
		return uint64(n), true
	case uint32:
		return uint64(n), true
	case uint64:
		return n, true
	}
	return 0, false
}

func align(t string) int {
	switch t[0] {
	case 'y', 'g', 'v':
		return 1
	case 'n', 'q':
		return 2
	case 'x', 't', 'd', '(', '{':
		return 8
	}
	return 4
}

// splitSig splits a signature into complete types.
func splitSig(s string) []string {
	var out []string
	for len(s) > 0 {
		n := typeLen(s)
		out = append(out, s[:n])
		s = s[n:]
	}
	return out
}

func typeLen(s string) int {
	switch s[0] {
	case 'a':
		return 1 + typeLen(s[1:])
	case '(', '{':
		depth := 0
		for i, c := range s {
			switch c {
			case '(', '{':
				depth++
			case ')', '}':
				depth--
				if depth == 0 {
					return i + 1
				}
			}
		}
	}
	return 1
}

func dictTypes(elem string) (string, string) {
	in := elem[1 : len(elem)-1]
	k := in[:typeLen(in)]
	return k, in[len(k):]
}

type fields struct {
	path, iface, member, dest, sig, errName string
	replySerial                             uint32
}

func header(typ byte, serial uint32, bodyLen int, path, iface, member, dest, sig string) []byte {
	return fields{path: path, iface: iface, member: member, dest: dest, sig: sig}.header(typ, serial, bodyLen)
}

func (f fields) header(typ byte, serial uint32, bodyLen int) []byte {
	var e encoder
	e.b.Write([]byte{'l', typ, 0, 1})
	e.u32(uint32(bodyLen))
	e.u32(serial)
	e.u32(0)
	lenAt := e.b.Len() - 4
	start := e.b.Len()
	field := func(code byte, t string, v any) {
		e.pad(8)
		e.b.WriteByte(code)
		e.sig(t)
		e.value(t, v)
	}
	if f.path != "" {
		field(1, "o", f.path)
	}
	if f.iface != "" {
		field(2, "s", f.iface)
	}
	if f.member != "" {
		field(3, "s", f.member)
	}
	if f.errName != "" {
		field(4, "s", f.errName)
	}
	if f.replySerial != 0 {
		field(5, "u", f.replySerial)
	}
	if f.dest != "" {
		field(6, "s", f.dest)
	}
	if f.sig != "" {
		field(8, "g", f.sig)
	}
	binary.LittleEndian.PutUint32(e.b.Bytes()[lenAt:], uint32(e.b.Len()-start))
	e.pad(8)
	return e.b.Bytes()
}

// ---------------------------------------------------------------- decoding

type decoder struct {
	b   []byte
	pos int
	bo  binary.ByteOrder
}

var errShort = errors.New("short D-Bus message")

func (d *decoder) pad(n int) {
	for d.pos%n != 0 {
		d.pos++
	}
}

func (d *decoder) need(n int) error {
	if d.pos+n > len(d.b) {
		return errShort
	}
	return nil
}

func (d *decoder) u32() (uint32, error) {
	d.pad(4)
	if err := d.need(4); err != nil {
		return 0, err
	}
	v := d.bo.Uint32(d.b[d.pos:])
	d.pos += 4
	return v, nil
}

func (d *decoder) value(t string) (any, error) {
	switch t[0] {
	case 'y':
		if err := d.need(1); err != nil {
			return nil, err
		}
		d.pos++
		return d.b[d.pos-1], nil
	case 'b':
		v, err := d.u32()
		return v != 0, err
	case 'u':
		v, err := d.u32()
		return uint64(v), err
	case 'i', 'h':
		v, err := d.u32()
		return int64(int32(v)), err
	case 'n', 'q':
		d.pad(2)
		if err := d.need(2); err != nil {
			return nil, err
		}
		v := d.bo.Uint16(d.b[d.pos:])
		d.pos += 2
		if t[0] == 'n' {
			return int64(int16(v)), nil
		}
		return uint64(v), nil
	case 'x', 't', 'd':
		d.pad(8)
		if err := d.need(8); err != nil {
			return nil, err
		}
		v := d.bo.Uint64(d.b[d.pos:])
		d.pos += 8
		switch t[0] {
		case 'x':
			return int64(v), nil
		case 'd':
			return math.Float64frombits(v), nil
		}
		return v, nil
	case 's', 'o':
		n, err := d.u32()
		if err != nil {
			return nil, err
		}
		if err := d.need(int(n) + 1); err != nil {
			return nil, err
		}
		s := string(d.b[d.pos : d.pos+int(n)])
		d.pos += int(n) + 1
		return s, nil
	case 'g':
		if err := d.need(1); err != nil {
			return nil, err
		}
		n := int(d.b[d.pos])
		if err := d.need(n + 2); err != nil {
			return nil, err
		}
		s := string(d.b[d.pos+1 : d.pos+1+n])
		d.pos += n + 2
		return s, nil
	case 'v':
		s, err := d.value("g")
		if err != nil {
			return nil, err
		}
		return d.value(s.(string))
	case '(':
		d.pad(8)
		var out []any
		for _, f := range splitSig(t[1 : len(t)-1]) {
			v, err := d.value(f)
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
		return out, nil
	case 'a':
		n, err := d.u32()
		if err != nil {
			return nil, err
		}
		elem := t[1:]
		d.pad(align(elem))
		end := d.pos + int(n)
		if end > len(d.b) {
			return nil, errShort
		}
		if elem == "y" {
			b := append([]byte(nil), d.b[d.pos:end]...)
			d.pos = end
			return b, nil
		}
		if elem[0] == '{' {
			kt, vt := dictTypes(elem)
			m := map[string]any{}
			for d.pos < end {
				d.pad(8)
				k, err := d.value(kt)
				if err != nil {
					return nil, err
				}
				v, err := d.value(vt)
				if err != nil {
					return nil, err
				}
				m[fmt.Sprint(k)] = v
			}
			return m, nil
		}
		var out []any
		for d.pos < end {
			v, err := d.value(elem)
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
		return out, nil
	}
	return nil, fmt.Errorf("D-Bus type %q isn't handled", t)
}

func readMessage(r io.Reader) (*Message, error) {
	fixed := make([]byte, 16)
	if _, err := io.ReadFull(r, fixed); err != nil {
		return nil, err
	}
	var bo binary.ByteOrder = binary.LittleEndian
	if fixed[0] == 'B' {
		bo = binary.BigEndian
	}
	bodyLen, fieldsLen := bo.Uint32(fixed[4:]), bo.Uint32(fixed[12:])
	if bodyLen > 64<<20 || fieldsLen > 1<<20 {
		return nil, errors.New("D-Bus message too big")
	}
	hlen := 16 + int(fieldsLen)
	total := (hlen+7)&^7 + int(bodyLen)
	buf := make([]byte, total)
	copy(buf, fixed)
	if _, err := io.ReadFull(r, buf[16:]); err != nil {
		return nil, err
	}
	m := &Message{Type: fixed[1], Serial: bo.Uint32(fixed[8:])}
	d := &decoder{b: buf[:hlen], pos: 16, bo: bo}
	for d.pos < hlen {
		d.pad(8)
		if d.pos >= hlen {
			break
		}
		code := d.b[d.pos]
		d.pos++
		v, err := d.value("v")
		if err != nil {
			return nil, err
		}
		s, _ := v.(string)
		switch code {
		case 1:
			m.Path = s
		case 2:
			m.Interface = s
		case 3:
			m.Member = s
		case 4:
			m.ErrorName = s
		case 5:
			n, _ := v.(uint64)
			m.ReplySerial = uint32(n)
		case 7:
			m.Sender = s
		case 8:
			m.Signature = s
		}
	}
	bd := &decoder{b: buf, pos: (hlen + 7) &^ 7, bo: bo}
	for _, t := range splitSig(m.Signature) {
		v, err := bd.value(t)
		if err != nil {
			return nil, err
		}
		m.Body = append(m.Body, v)
	}
	return m, nil
}
