package flashie

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"strings"
	"testing"
)

func TestExtractCLI(t *testing.T) {
	var tgz bytes.Buffer
	gz := gzip.NewWriter(&tgz)
	tw := tar.NewWriter(gz)
	for _, f := range []struct{ name, body string }{{"LICENSE.txt", "licence"}, {"arduino-cli", "ELF-binary"}} {
		tw.WriteHeader(&tar.Header{Name: f.name, Mode: 0o755, Size: int64(len(f.body)), Typeflag: tar.TypeReg})
		tw.Write([]byte(f.body))
	}
	tw.Close()
	gz.Close()
	if b, err := extractCLI(tgz.Bytes(), false); err != nil || string(b) != "ELF-binary" {
		t.Errorf("tar.gz: %q %v", b, err)
	}

	var z bytes.Buffer
	zw := zip.NewWriter(&z)
	w, _ := zw.Create("arduino-cli.exe")
	w.Write([]byte("MZ-binary"))
	zw.Close()
	if b, err := extractCLI(z.Bytes(), true); err != nil || string(b) != "MZ-binary" {
		t.Errorf("zip: %q %v", b, err)
	}

	if _, err := extractCLI([]byte("not an archive"), false); err == nil || !strings.Contains(err.Error(), "damaged") {
		t.Errorf("bad archive: %v", err)
	}
	if !strings.HasSuffix(arduinoCLIURL(), ".tar.gz") && !strings.HasSuffix(arduinoCLIURL(), ".zip") {
		t.Errorf("no arduino-cli download for this system: %q", arduinoCLIURL())
	}
}
