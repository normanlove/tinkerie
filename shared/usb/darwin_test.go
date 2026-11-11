//go:build darwin && cgo

package usb

import "testing"

func TestDarwinList(t *testing.T) {
	for _, d := range List() { // the CI Mac has few USB devices; this checks IOKit enumeration works
		t.Logf("%s %04X:%04X %q %q %d interfaces", d.Path, d.VID, d.PID, d.Manufacturer, d.Product, len(d.Interfaces))
	}
	// a configuration descriptor with a vendor interface and two bulk endpoints
	cfg := []byte{9, 2, 32, 0, 1, 1, 0, 0x80, 50, 9, 4, 0, 0, 2, 0xFF, 0, 0, 0, 7, 5, 0x86, 2, 0, 2, 0, 7, 5, 0x02, 2, 0, 2, 0}
	it := parseConfig(cfg)
	if len(it) != 1 || it[0].Class != 0xFF || len(it[0].BulkIn) != 1 || it[0].BulkIn[0] != 0x86 || it[0].BulkOut[0] != 0x02 {
		t.Fatalf("parseConfig: %+v", it)
	}
}
