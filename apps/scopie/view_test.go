package scopie

import (
	"encoding/binary"
	"encoding/json"
	"os"
	"sync"
	"testing"
	"time"
)

// Everything the page and the background loops do at once, under -race.
func TestViewConcurrent(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("APPDATA", dir)
	os.Setenv("XDG_CONFIG_HOME", dir)
	eng := newEngine()
	eng.Start()
	defer eng.Quit()
	v := newView(eng, newHub(), newSettingsStore())
	v.Connect(scanSim(nil, nil)[0])
	s := v.Settings()
	s.FFTOn, s.MathOn, s.Persist, s.Average = true, true, 1, 4
	s.Meas = []string{"freq", "vpp", "phase"}
	v.SetSettings(s)

	stop := make(chan struct{})
	var wg sync.WaitGroup
	run := func(f func()) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					f()
				}
			}
		}()
	}
	var frames, bad int
	var mu sync.Mutex
	have := int64(-1)
	run(func() {
		b := v.Frame(have)
		if b != nil {
			hl := binary.LittleEndian.Uint32(b)
			var hdr map[string]any
			if json.Unmarshal(b[4:4+hl], &hdr) != nil {
				mu.Lock()
				bad++
				mu.Unlock()
			}
			have = int64(hdr["seq"].(float64))
			mu.Lock()
			frames++
			mu.Unlock()
		}
		time.Sleep(5 * time.Millisecond)
	})
	run(func() { v.updateMeasurements(); time.Sleep(20 * time.Millisecond) })
	run(func() { v.updateFFT(); time.Sleep(20 * time.Millisecond) })
	run(func() { _ = v.status(); time.Sleep(30 * time.Millisecond) })
	run(func() {
		s := v.Settings()
		s.Ch[0].Pos = float64(time.Now().UnixNano()%300)/100 - 1.5
		v.SetSettings(s)
		time.Sleep(15 * time.Millisecond)
	})
	time.Sleep(500 * time.Millisecond)
	if msg := v.Autoset(); msg != "" {
		t.Fatalf("autoset: %s", msg)
	}
	time.Sleep(2500 * time.Millisecond)
	close(stop)
	wg.Wait()
	st := v.status()
	if frames < 20 || bad > 0 {
		t.Errorf("frames %d, bad %d", frames, bad)
	}
	if st.Meas["freq"][0] == "" || st.Meas["freq"][0] == "--" {
		t.Errorf("no frequency measured: %v", st.Meas)
	}
	if v.fft == nil || v.fft.peakF < 900 || v.fft.peakF > 1100 {
		t.Errorf("FFT peak wrong: %+v", v.fft)
	}
	t.Logf("%d frames, freq %s, FFT peak %.0f Hz", frames, st.Meas["freq"][0], v.fft.peakF)
}
