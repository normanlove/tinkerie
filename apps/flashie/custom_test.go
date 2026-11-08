package flashie

import (
	"os/exec"
	"strings"
	"testing"
)

func TestSHA512Crypt(t *testing.T) {
	// The SHA-crypt spec's own test vector.
	if got := sha512Crypt("Hello world!", "saltstring"); got != "$6$saltstring$svn8UoSVapNtMuq1ukKS4tPQd8iKwSMHWjl/O817G3uBnIFNjnQJuesI68u4OTLiBFdcbYEdFCoEOfaS35inz1" {
		t.Fatalf("got %s", got)
	}
	if _, err := exec.LookPath("openssl"); err != nil {
		return
	}
	for _, c := range [][2]string{{"x", "abc"}, {"pässwörd with spaces", "0123456789abcdef"}, {strings.Repeat("long", 40), "x"}, {"a'b\"c$d", "sALT./"}} {
		out, err := exec.Command("openssl", "passwd", "-6", "-salt", c[1], c[0]).Output()
		if err != nil {
			t.Fatal(err)
		}
		if got := sha512Crypt(c[0], c[1]); got != strings.TrimSpace(string(out)) {
			t.Fatalf("%q: %s, openssl %s", c[0], got, out)
		}
	}
}

func TestWifiPSK(t *testing.T) {
	// IEEE 802.11i test vector.
	if got := wifiPSK("password", "IEEE"); got != "f42c6fc52df0ebef9ebb4b90b38a5f902e83fe1b135a70e23aed762e9710a12e" {
		t.Fatal(got)
	}
}
