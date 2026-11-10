package probie

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"
)

// ---------------------------------------------------------------- SSE hub

type hubMsg struct {
	event string
	data  []byte
}

type Hub struct {
	mu   sync.Mutex
	subs map[chan hubMsg]struct{}
}

func newHub() *Hub { return &Hub{subs: map[chan hubMsg]struct{}{}} }

func (h *Hub) subscribe() chan hubMsg {
	ch := make(chan hubMsg, 512)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *Hub) unsubscribe(ch chan hubMsg) {
	h.mu.Lock()
	if _, ok := h.subs[ch]; ok {
		delete(h.subs, ch)
		close(ch)
	}
	h.mu.Unlock()
}

// publish never blocks: a slow client just misses messages.
func (h *Hub) publish(event string, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		log.Printf("publish %s: %v", event, err)
		return
	}
	h.mu.Lock()
	for ch := range h.subs {
		select {
		case ch <- hubMsg{event, b}:
		default:
		}
	}
	h.mu.Unlock()
}

// ---------------------------------------------------------------- JSON helpers

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte("null")
	}
	return b
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": msg})
}

func readJSON(r *http.Request, v any) error {
	b, err := io.ReadAll(io.LimitReader(r.Body, 64<<20))
	if err != nil {
		return err
	}
	if len(b) == 0 {
		return nil
	}
	return json.Unmarshal(b, v)
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
