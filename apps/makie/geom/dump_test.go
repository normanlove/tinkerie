package geom

import (
	"encoding/binary"
	"math"
	"os"
	"strings"
	"testing"
)

func TestDumpShapes(t *testing.T) {
	dir := os.Getenv("DUMP")
	if dir == "" {
		t.Skip()
	}
	types := strings.Fields(os.Getenv("TYPES"))
	for _, ty := range types {
		parts := strings.SplitN(ty, ":", 2)
		p := Params{"_q": 0.8}
		if len(parts) == 2 {
			for _, kv := range strings.Split(parts[1], ",") {
				a := strings.SplitN(kv, "=", 2)
				p[a[0]] = strings.ReplaceAll(a[1], "_", " ")
			}
		}
		m, err := Generate(parts[0], p)
		if err != nil {
			t.Error(ty, err)
			continue
		}
		b := make([]byte, 0, len(m.Tris)*36)
		for _, tr := range m.Tris {
			for _, v := range tr {
				for _, c := range []float64{v.X, v.Y, v.Z} {
					b = binary.LittleEndian.AppendUint32(b, math.Float32bits(float32(c)))
				}
			}
		}
		os.WriteFile(dir+"/"+strings.ReplaceAll(ty, "/", "_")+".bin", b, 0o644)
	}
}
