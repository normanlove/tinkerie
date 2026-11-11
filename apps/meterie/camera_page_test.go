//go:build linux || darwin

package meterie

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPageCamera(t *testing.T) {
	mux := http.NewServeMux()
	pageCam.routes(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()
	src, _ := platformOpenCamera(0)
	m := map[string]any{}
	pageCam.frameFields(m)
	if m["cam"] != 0 {
		t.Fatal("page not asked for camera 0:", m)
	}
	img := image.NewRGBA(image.Rect(0, 0, 64, 48))
	for i := range img.Pix {
		img.Pix[i] = 200
	}
	img.Set(10, 10, color.Black)
	var b bytes.Buffer
	jpeg.Encode(&b, img, nil)
	go http.Post(srv.URL+"/camframe", "image/jpeg", &b)
	g, err := src.Grab()
	if err != nil || g.Bounds().Dx() != 64 || g.Bounds().Dy() != 48 {
		t.Fatal(g, err)
	}
	src.Close()
	pageCam.frameFields(m)
	if m["cam"] != -1 {
		t.Fatal("camera not switched off")
	}
	http.Post(srv.URL+"/camerr", "text/plain", bytes.NewBufferString("not allowed"))
	src2, _ := platformOpenCamera(0)
	pageCam.mu.Lock()
	pageCam.err = "not allowed"
	pageCam.mu.Unlock()
	if _, err := src2.Grab(); err == nil || err.Error() != "not allowed" {
		t.Fatal("want the page's error, got", err)
	}
}
