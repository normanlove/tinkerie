package meterie

// winrtiid.go: WinRT parameterised interface IDs (pinterface GUIDs) are the
// SHA-1 based UUIDv5 of a type signature in a fixed namespace. Platform
// independent so it can be unit tested.

import (
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"strings"
)

type guidT struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

func parseGUID(s string) guidT {
	s = strings.Trim(strings.TrimSpace(s), "{}")
	var g guidT
	var d4 [8]byte
	var a, b, c, d, e uint64
	fmt.Sscanf(strings.ReplaceAll(s, "-", " "), "%x %x %x %x %x", &a, &b, &c, &d, &e)
	g.Data1, g.Data2, g.Data3 = uint32(a), uint16(b), uint16(c)
	d4[0], d4[1] = byte(d>>8), byte(d)
	for i := 0; i < 6; i++ {
		d4[2+i] = byte(e >> (8 * (5 - i)))
	}
	g.Data4 = d4
	return g
}

func (g guidT) String() string {
	return fmt.Sprintf("{%08x-%04x-%04x-%02x%02x-%02x%02x%02x%02x%02x%02x}", g.Data1, g.Data2, g.Data3,
		g.Data4[0], g.Data4[1], g.Data4[2], g.Data4[3], g.Data4[4], g.Data4[5], g.Data4[6], g.Data4[7])
}

func (g guidT) bytesBE() []byte {
	b := make([]byte, 16)
	binary.BigEndian.PutUint32(b[0:], g.Data1)
	binary.BigEndian.PutUint16(b[4:], g.Data2)
	binary.BigEndian.PutUint16(b[6:], g.Data3)
	copy(b[8:], g.Data4[:])
	return b
}

var winrtPinterfaceNS = parseGUID("11f47ad5-7b73-42c0-abae-878b1e16adee")

// pinterfaceIID computes the IID for a WinRT parameterised type signature.
func pinterfaceIID(sig string) guidT {
	h := sha1.New()
	h.Write(winrtPinterfaceNS.bytesBE())
	h.Write([]byte(sig))
	sum := h.Sum(nil)
	var g guidT
	g.Data1 = binary.BigEndian.Uint32(sum[0:])
	g.Data2 = binary.BigEndian.Uint16(sum[4:])
	g.Data3 = (binary.BigEndian.Uint16(sum[6:]) & 0x0FFF) | 0x5000
	copy(g.Data4[:], sum[8:16])
	g.Data4[0] = (g.Data4[0] & 0x3F) | 0x80
	return g
}

// typedEventHandlerSig builds the signature of TypedEventHandler<Sender, Args>.
func typedEventHandlerSig(senderClass, senderIID, argsClass, argsIID string) string {
	return "pinterface({9de1c534-6ae1-11e0-84e1-18a905bcc53f};rc(" + senderClass + ";" + strings.ToLower(senderIID) + ");rc(" + argsClass + ";" + strings.ToLower(argsIID) + "))"
}

// BLE UUID helpers: "fff4" -> 0000fff4-0000-1000-8000-00805f9b34fb
func bleUUID(s string) guidT {
	s = strings.TrimSpace(strings.ToLower(s))
	if len(s) == 4 {
		s = "0000" + s + "-0000-1000-8000-00805f9b34fb"
	} else if len(s) == 8 {
		s = s + "-0000-1000-8000-00805f9b34fb"
	}
	return parseGUID(s)
}
