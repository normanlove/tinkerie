//go:build linux || darwin

package meterie

// camera_page.go: Camera mode on macOS and Linux. Meterie's window there is a
// browser page (web_front.go), and browsers already know every webcam, so the
// page opens the camera (the browser asks the person first) and sends
// frames here as JPEGs; the display reader then works exactly as on Windows.

import (
	"bytes"
	"encoding/json"
	"errors"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"sync"
	"time"
)

type pageCamera struct {
	mu      sync.Mutex
	want    int  // camera the page should run, -1 for none
	listReq bool // ask the page for its cameras
	names   []string
	frame   *image.Gray
	seq     int
	err     string
	newF    chan struct{}
}

var pageCam = &pageCamera{want: -1, newF: make(chan struct{}, 1)}

func (p *pageCamera) poke() {
	if front != nil {
		front.mu.Lock()
		front.wantDraw = true
		front.mu.Unlock()
	}
}

// frameFields adds the camera wishes to each frame sent to the page.
func (p *pageCamera) frameFields(m map[string]any) {
	p.mu.Lock()
	defer p.mu.Unlock()
	m["cam"] = p.want
	if p.listReq {
		m["camList"] = true
		p.listReq = false
	}
}

func (p *pageCamera) routes(mux *http.ServeMux) {
	mux.HandleFunc("/camlist", func(w http.ResponseWriter, r *http.Request) {
		var names []string
		json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&names)
		p.mu.Lock()
		p.names = names
		p.mu.Unlock()
		if wakeFn != nil {
			wakeFn()
		}
	})
	mux.HandleFunc("/camframe", func(w http.ResponseWriter, r *http.Request) {
		img, err := jpeg.Decode(io.LimitReader(r.Body, 16<<20))
		if err != nil {
			return
		}
		g := toGray(img)
		p.mu.Lock()
		p.frame, p.err = g, ""
		p.seq++
		p.mu.Unlock()
		select {
		case p.newF <- struct{}{}:
		default:
		}
	})
	mux.HandleFunc("/camerr", func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(io.LimitReader(r.Body, 4096))
		p.mu.Lock()
		p.err = string(bytes.TrimSpace(b))
		p.mu.Unlock()
		select {
		case p.newF <- struct{}{}:
		default:
		}
	})
}

func platformListCameras() []string {
	p := pageCam
	p.mu.Lock()
	p.listReq = true
	names := append([]string(nil), p.names...)
	p.mu.Unlock()
	p.poke()
	if len(names) == 0 {
		// the page fills in names once it's allowed to; the refresh button asks again
		return []string{"Webcam (your browser asks first)"}
	}
	return names
}

type pageSource struct {
	p    *pageCamera
	seen int
}

func platformOpenCamera(i int) (FrameSource, error) {
	p := pageCam
	p.mu.Lock()
	p.want, p.err, p.frame = i, "", nil
	seen := p.seq
	p.mu.Unlock()
	p.poke()
	return &pageSource{p: p, seen: seen}, nil
}

func (s *pageSource) Grab() (*image.Gray, error) {
	deadline := time.After(8 * time.Second) // the first frame waits for "Allow"
	for {
		s.p.mu.Lock()
		f, seq, err := s.p.frame, s.p.seq, s.p.err
		s.p.mu.Unlock()
		if err != "" {
			return nil, errors.New(err)
		}
		if f != nil && seq != s.seen {
			s.seen = seq
			return f, nil
		}
		select {
		case <-s.p.newF:
		case <-deadline:
			return nil, errors.New("no picture from the camera yet: allow the camera in Meterie's window")
		}
	}
}

func (s *pageSource) Close() {
	s.p.mu.Lock()
	s.p.want = -1
	s.p.mu.Unlock()
	s.p.poke()
}
