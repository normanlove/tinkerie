// Flashie — Arduino C++ builds for AVR boards via arduino-cli, downloaded on
// first use into Flashie's own folder (nothing is installed system-wide).
// Made by nrmluvdev.

package flashie

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// arduinoCLIURL is Arduino's "latest release" download for this system
// (https://arduino.github.io/arduino-cli/latest/installation/).
func arduinoCLIURL() string {
	plat := map[string]string{
		"windows/amd64": "Windows_64bit.zip", "windows/386": "Windows_32bit.zip",
		"linux/amd64": "Linux_64bit.tar.gz", "linux/arm64": "Linux_ARM64.tar.gz", "linux/arm": "Linux_ARMv7.tar.gz",
		"darwin/amd64": "macOS_64bit.tar.gz", "darwin/arm64": "macOS_ARM64.tar.gz",
	}[runtime.GOOS+"/"+runtime.GOARCH]
	if plat == "" {
		return ""
	}
	return "https://downloads.arduino.cc/arduino-cli/arduino-cli_latest_" + plat
}

var fqbnFor = map[string]string{
	"uno": "arduino:avr:uno", "uno-ch340": "arduino:avr:uno",
	"nano": "arduino:avr:nano:cpu=atmega328", "nano-old": "arduino:avr:nano:cpu=atmega328old", "nano-168": "arduino:avr:nano:cpu=atmega168",
	"pro-mini-5v": "arduino:avr:pro:cpu=16MHzatmega328", "pro-mini-3v3": "arduino:avr:pro:cpu=8MHzatmega328",
	"mega2560": "arduino:avr:mega:cpu=atmega2560", "mega2560-ch340": "arduino:avr:mega:cpu=atmega2560",
	"leonardo": "arduino:avr:leonardo", "micro": "arduino:avr:micro", "pro-micro": "arduino:avr:leonardo",
}

// Libraries the generated code may need, by Arduino library-manager name.
var arduinoLibAllow = map[string]bool{
	"Servo": true, "DHT sensor library": true, "Adafruit Unified Sensor": true,
	"Adafruit NeoPixel": true, "Adafruit SSD1306": true, "Adafruit GFX Library": true,
}

func arduinoHome() string {
	d := filepath.Join(filepath.Dir(cacheRoot()), "arduino")
	os.MkdirAll(d, 0o755)
	return d
}

func cliPath() string {
	name := "arduino-cli"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(arduinoHome(), "bin", name)
}

func ensureArduinoCLI(r *reporter) (string, error) {
	exe := cliPath()
	if _, err := os.Stat(exe); err == nil {
		return exe, nil
	}
	url := arduinoCLIURL()
	if url == "" {
		return "", errors.New("there's no arduino-cli download for this system — install arduino-cli and put it at " + exe)
	}
	r.status("Downloading arduino-cli (one-time, about 30 MB)…")
	resp, err := httpGet(url, "")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var buf bytes.Buffer
	total := int(resp.ContentLength)
	chunk := make([]byte, 128*1024)
	for {
		n, err := resp.Body.Read(chunk)
		buf.Write(chunk[:n])
		if total > 0 {
			r.progress(buf.Len(), total)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("arduino-cli download interrupted: %v", err)
		}
	}
	b, err := extractCLI(buf.Bytes(), strings.HasSuffix(url, ".zip"))
	if err != nil {
		return "", err
	}
	os.MkdirAll(filepath.Dir(exe), 0o755)
	if err := os.WriteFile(exe, b, 0o755); err != nil {
		return "", err
	}
	r.logf("arduino-cli saved to %s", exe)
	return exe, nil
}

// extractCLI takes the arduino-cli program out of the downloaded archive: a
// .zip on Windows, a .tar.gz everywhere else.
func extractCLI(data []byte, isZip bool) ([]byte, error) {
	want := "arduino-cli"
	if isZip {
		want += ".exe"
		zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return nil, fmt.Errorf("the arduino-cli download is damaged: %v", err)
		}
		for _, f := range zr.File {
			if strings.EqualFold(filepath.Base(f.Name), want) {
				rc, err := f.Open()
				if err != nil {
					return nil, err
				}
				defer rc.Close()
				return io.ReadAll(rc)
			}
		}
		return nil, errors.New(want + " wasn't in the download")
	}
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("the arduino-cli download is damaged: %v", err)
	}
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil, errors.New(want + " wasn't in the download")
		}
		if err != nil {
			return nil, fmt.Errorf("the arduino-cli download is damaged: %v", err)
		}
		if h.Typeflag == tar.TypeReg && filepath.Base(h.Name) == want {
			return io.ReadAll(tr)
		}
	}
}

// runCLI runs arduino-cli with Flashie's private folders, streaming output.
func runCLI(exe string, r *reporter, args ...string) error {
	home := arduinoHome()
	cmd := exec.Command(exe, args...)
	cmd.Env = append(os.Environ(),
		"ARDUINO_DIRECTORIES_DATA="+filepath.Join(home, "data"),
		"ARDUINO_DIRECTORIES_DOWNLOADS="+filepath.Join(home, "staging"),
		"ARDUINO_DIRECTORIES_USER="+filepath.Join(home, "user"),
		"ARDUINO_UPDATER_ENABLE_NOTIFICATION=false",
	)
	hideWindow(cmd)
	out, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = cmd.Stdout
	emitConsole("$ arduino-cli "+strings.Join(args, " ")+"\n", "build")
	if err := cmd.Start(); err != nil {
		return err
	}
	sc := bufio.NewScanner(out)
	sc.Buffer(make([]byte, 64*1024), 1<<20)
	for sc.Scan() {
		emitConsole(sc.Text()+"\n", "build")
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("arduino-cli %s failed (%v) — see the console for details", args[0], err)
	}
	return nil
}

// arduinoBuild compiles code for a board and returns the .hex contents.
func arduinoBuild(boardID, code string, libs []string, r *reporter) ([]byte, error) {
	fqbn, ok := fqbnFor[boardID]
	if !ok {
		return nil, errors.New("pick your Arduino board (Uno, Nano, Mega…) in the board list so Flashie knows what to compile for")
	}
	if demoMode {
		return demoArduinoBuild(code, r)
	}
	exe, err := ensureArduinoCLI(r)
	if err != nil {
		return nil, err
	}
	home := arduinoHome()
	marker := filepath.Join(home, "avr-core.ok")
	if _, err := os.Stat(marker); err != nil {
		r.status("Setting up the Arduino AVR compiler (one-time download, a few minutes)…")
		if err := runCLI(exe, r, "core", "update-index"); err != nil {
			return nil, err
		}
		if err := runCLI(exe, r, "core", "install", "arduino:avr"); err != nil {
			return nil, err
		}
		os.WriteFile(marker, []byte(time.Now().Format(time.RFC3339)), 0o644)
	}
	for _, l := range libs {
		if !arduinoLibAllow[l] {
			continue
		}
		lm := filepath.Join(home, "lib-"+strings.ReplaceAll(l, " ", "_")+".ok")
		if _, err := os.Stat(lm); err == nil {
			continue
		}
		r.status("Installing the " + l + " library…")
		if err := runCLI(exe, r, "lib", "install", l); err != nil {
			return nil, err
		}
		os.WriteFile(lm, nil, 0o644)
	}
	sketch := filepath.Join(home, "sketches", "flashie_sketch")
	os.MkdirAll(sketch, 0o755)
	if err := os.WriteFile(filepath.Join(sketch, "flashie_sketch.ino"), []byte(code), 0o644); err != nil {
		return nil, err
	}
	outDir := filepath.Join(home, "build")
	os.RemoveAll(outDir)
	r.status("Compiling your sketch…")
	if err := runCLI(exe, r, "compile", "--fqbn", fqbn, "--output-dir", outDir, sketch); err != nil {
		return nil, err
	}
	return os.ReadFile(filepath.Join(outDir, "flashie_sketch.ino.hex"))
}

func handleCreatorArduino(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Port, Board, Code string
		Libs              []string
	}
	if err := decodeJSON(r, &in); err != nil || in.Code == "" {
		jsonErr(w, "Nothing to compile.")
		return
	}
	if !busy.CompareAndSwap(false, true) {
		jsonErr(w, "Flashie is busy with another job — wait for it to finish.")
		return
	}
	cancelReq.Store(false)
	writeJSON(w, map[string]any{"started": true})
	go func() {
		defer busy.Store(false)
		rep := creatorReporter()
		stopSession()
		hexData, err := arduinoBuild(in.Board, in.Code, in.Libs, rep)
		if err != nil {
			emitCDone(false, friendly(err))
			return
		}
		if in.Port == "" {
			emitCDone(true, "Compiled OK. Pick the board's port to upload it.")
			return
		}
		pi, found := findDevice(in.Port)
		if !found {
			emitCDone(false, in.Port+" isn't connected any more.")
			return
		}
		b := boardByID(in.Board)
		pl := planFor(pi, b, nil)
		if pl == nil || pl.Family != famAVR {
			emitCDone(false, "The selected board isn't an Arduino AVR board.")
			return
		}
		rep.status("Uploading to the board…")
		msg, err := flashAVRJob(pi, pl, flashReq{Port: in.Port, Board: b, Data: hexData, FileName: "flashie_sketch.ino.hex"}, rep)
		if err != nil {
			emitCDone(false, friendly(err))
			return
		}
		time.Sleep(1200 * time.Millisecond)
		var p Port
		if demoMode {
			p, err = openReplPort(in.Port)
		} else {
			p, err = openSerialEx(in.Port, 115200, false)
		}
		if err == nil {
			startSession(p, in.Port, "serial")
		}
		emitCDone(true, msg+" Serial output appears in the console (115200 baud).")
	}()
}
