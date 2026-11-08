package flashie

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestReplUploadAndRun(t *testing.T) {
	sim := newSimMPY("Generic ESP32 module with ESP32")
	sim.running = true // pretend an old program is running
	m := newRepl(sim)
	if err := m.enterRaw(); err != nil {
		t.Fatal(err)
	}
	inf, err := m.info()
	if err != nil || !inf.MicroPython || inf.Version != "1.28.0" || inf.Platform != "esp32" {
		t.Fatalf("%v %+v", err, inf)
	}
	code := []byte(strings.Repeat("print(\"hello 'quoted' \\n\")\n", 200)) // > several chunks
	done, total := 0, len(code)
	if err := m.writeFile("main.py", code, testRep(t), &done, &total); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sim.files["main.py"], code) || done != total {
		t.Fatalf("file mismatch: %d vs %d", len(sim.files["main.py"]), len(code))
	}
	if _, err := m.exec("raise ValueError()", time.Second); err == nil || !strings.Contains(err.Error(), "ValueError") {
		t.Fatalf("expected board error, got %v", err)
	}
	m.exitRaw()
	sim.Write([]byte{0x04})
	time.Sleep(20 * time.Millisecond)
	buf := make([]byte, 4096)
	var got []byte
	for i := 0; i < 5; i++ {
		n, _ := sim.ReadTimeout(buf, 20*time.Millisecond)
		got = append(got, buf[:n]...)
	}
	if !strings.Contains(string(got), "soft reboot") || !strings.Contains(string(got), "hello") {
		t.Fatalf("run output: %q", got)
	}
}

func TestReplNoMicroPython(t *testing.T) {
	m := newRepl(newSimChip(0, 0)) // ROM bootloader sim never answers the REPL
	start := time.Now()
	if err := m.enterRaw(); err == nil {
		t.Fatal("expected failure")
	}
	if time.Since(start) > 10*time.Second {
		t.Fatal("too slow")
	}
}
