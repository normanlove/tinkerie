// Fixie's error log explainer: the window does the explaining (web/logscan.js
// and web/logrules.json, no AI and no internet); this file gives it the
// computer's own logs when asked ("Scan this computer"), using the tools the
// system already has. Created by nrmluvdev for Tinkerie.
package fixie

import (
	"bytes"
	"context"
	"encoding/binary"
	"net/http"
	"os/exec"
	"strings"
	"time"
	"unicode/utf16"
)

const maxLogSource = 4 << 20 // bytes kept from each source (the newest end)

// logSource is one place the computer keeps a log, and what happened reading it.
type logSource struct {
	Name string `json:"name"`
	OK   bool   `json:"ok"`
	Note string `json:"note,omitempty"`
	Size int    `json:"size"`
	text string
}

func (s *server) logRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/logs/sources", func(w http.ResponseWriter, r *http.Request) {
		ok, what := logsAvailable()
		writeJSON(w, map[string]any{"available": ok, "what": what})
	})
	mux.HandleFunc("/api/logs/scan", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST", http.StatusMethodNotAllowed)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 150*time.Second)
		defer cancel()
		srcs := readSystemLogs(ctx)
		var b strings.Builder
		for _, s := range srcs {
			if s.text != "" {
				b.WriteString(s.text)
				if !strings.HasSuffix(s.text, "\n") {
					b.WriteByte('\n')
				}
			}
		}
		writeJSON(w, map[string]any{"sources": srcs, "text": b.String()})
	})
}

// runLog runs a command and returns its output (the newest end, if it's long).
func runLog(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	hideWindow(cmd)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	t := decodeText(out.Bytes())
	if err != nil && strings.TrimSpace(t) == "" {
		msg := strings.TrimSpace(decodeText(errb.Bytes()))
		if msg == "" {
			msg = err.Error()
		}
		return "", errString(firstLine(msg))
	}
	return keepEnd(t), nil
}

type errString string

func (e errString) Error() string { return string(e) }

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}

func keepEnd(t string) string {
	if len(t) <= maxLogSource {
		return t
	}
	t = t[len(t)-maxLogSource:]
	if i := strings.IndexByte(t, '\n'); i >= 0 {
		t = t[i+1:]
	}
	return t
}

// decodeText turns UTF-16 output (some Windows tools) into UTF-8.
func decodeText(b []byte) string {
	if len(b) >= 2 && (b[0] == 0xFF && b[1] == 0xFE || len(b) >= 4 && b[1] == 0 && b[3] == 0) {
		if b[0] == 0xFF && b[1] == 0xFE {
			b = b[2:]
		}
		u := make([]uint16, len(b)/2)
		for i := range u {
			u[i] = binary.LittleEndian.Uint16(b[2*i:])
		}
		return string(utf16.Decode(u))
	}
	return string(bytes.TrimPrefix(b, []byte{0xEF, 0xBB, 0xBF}))
}

func source(ctx context.Context, name string, cmd string, args ...string) logSource {
	t, err := runLog(ctx, cmd, args...)
	if err != nil {
		return logSource{Name: name, Note: err.Error()}
	}
	return logSource{Name: name, OK: true, Size: len(t), text: t}
}
