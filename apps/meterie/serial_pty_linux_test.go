package meterie

import (
	"os"
	"strconv"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

// TestHP90EPCOverRealSerial opens a pseudo-terminal with the HoldPeak
// HP-90EPC's settings (2400 baud, 8N1, DTR on) through Meterie's Linux serial
// code, sends an FS9721 packet from the "meter" end and decodes it.
func TestHP90EPCOverRealSerial(t *testing.T) {
	m, err := os.OpenFile("/dev/ptmx", os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		t.Skip("no pseudo-terminals here:", err)
	}
	defer m.Close()
	unlock := int32(0)
	syscall.Syscall(syscall.SYS_IOCTL, m.Fd(), 0x40045431, uintptr(unsafe.Pointer(&unlock)))
	var n uint32
	syscall.Syscall(syscall.SYS_IOCTL, m.Fd(), 0x80045430, uintptr(unsafe.Pointer(&n)))
	path := "/dev/pts/" + strconv.Itoa(int(n))

	var cfg SerialCfg
	for _, md := range meterCatalog {
		if md.ID == "holdpeak-hp90epc" {
			cfg = md.Serial
		}
	}
	if cfg.Baud != 2400 {
		t.Fatalf("HP-90EPC settings not found: %+v", cfg)
	}
	p, err := platformOpenSerial(path, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	pkt := []byte{0x17, 0x27, 0x3D, 0x4F, 0x5D, 0x67, 0x7D, 0x87, 0x9D, 0xA0, 0xB0, 0xC0, 0xD4, 0xE0}
	m.Write(pkt)
	d := NewFS9721(2, 4000)
	var got []Reading
	buf := make([]byte, 64)
	for deadline := time.Now().Add(2 * time.Second); len(got) == 0 && time.Now().Before(deadline); {
		k, err := p.Read(buf) // returns within 150 ms, like on Windows
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, d.Feed(buf[:k])...)
	}
	if len(got) != 1 || got[0].Unit != "V" || got[0].Display != "0.000" {
		t.Fatalf("decoded %+v", got)
	}
}
