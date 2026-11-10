// Makie — a simple but powerful 3D part designer for 3D printing, by nrmluvdev.
// The Go program is the geometry engine and serves the editor UI, which it
// opens in a chromeless browser window (Edge/Chrome app mode when available).
package makie

import (
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"tinkerie/apps/makie/geom"
)

//go:embed web
var webFS embed.FS

const appName = "Makie"
const author = "nrmluvdev"
const version = "0.10.0-alpha"

var lastPing atomic.Int64

func Main() {
	port := flag.Int("port", 47811, "port for the local editor server")
	noBrowser := flag.Bool("nobrowser", false, "do not open a window (serve only)")
	idle := flag.Duration("idle", 4*time.Minute, "quit after the editor window has been closed this long (0 = never)")
	flag.Parse()
	migrateOldData()
	setupLog()

	addr := fmt.Sprintf("127.0.0.1:%d", *port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		// already running? hand it any files to open, or open another window onto it
		if alreadyRunning(addr) {
			if len(flag.Args()) > 0 && handOver(addr, flag.Args()) {
				return
			}
			if !*noBrowser {
				openWindow("http://" + addr + "/")
			}
			return
		}
		ln, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			log.Fatal(err)
		}
	}
	url := "http://" + ln.Addr().String() + "/"
	makeInboxToken()
	queueFiles(flag.Args())
	go registerFileTypes()
	log.Printf("%s %s listening on %s", appName, version, url)
	fmt.Println(appName, "is running at", url)

	mux := http.NewServeMux()
	sub, _ := fs.Sub(webFS, "web")
	files := http.FileServer(http.FS(sub))
	mux.Handle("/", noCache(files))
	registerAPI(mux)

	lastPing.Store(time.Now().Unix())
	if !*noBrowser {
		go func() {
			time.Sleep(150 * time.Millisecond)
			openWindow(url)
		}()
	}
	if *idle > 0 {
		go watchdog(*idle)
	}
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	log.Fatal(srv.Serve(ln))
}

func setupLog() {
	dir, err := os.UserConfigDir()
	if err != nil {
		return
	}
	dir = filepath.Join(dir, appName)
	_ = os.MkdirAll(dir, 0o755)
	f, err := os.OpenFile(filepath.Join(dir, "makie.log"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err == nil {
		log.SetOutput(io.MultiWriter(f, os.Stderr))
	}
}

func alreadyRunning(addr string) bool {
	c := http.Client{Timeout: time.Second}
	r, err := c.Get("http://" + addr + "/api/ping")
	if err != nil {
		return false
	}
	defer r.Body.Close()
	b, _ := io.ReadAll(r.Body)
	return strings.Contains(string(b), appName)
}

func watchdog(idle time.Duration) {
	for {
		time.Sleep(10 * time.Second)
		if time.Since(time.Unix(lastPing.Load(), 0)) > idle {
			log.Println("editor closed; exiting")
			os.Exit(0)
		}
	}
}

func noCache(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		h.ServeHTTP(w, r)
	})
}

// ---------------- API ----------------

var buildSem = make(chan struct{}, max(2, runtime.NumCPU()))

type buildReq struct {
	Node *geom.Node `json:"node"`
	Q    float64    `json:"q"`
}

type exportReq struct {
	Nodes    []*geom.Node `json:"nodes"`
	Q        float64      `json:"q"`
	Name     string       `json:"name"`
	Separate bool         `json:"separate"`
}

func registerAPI(mux *http.ServeMux) {
	registerWirieAPI(mux)
	registerInboxAPI(mux)
	mux.HandleFunc("/api/ping", func(w http.ResponseWriter, r *http.Request) {
		lastPing.Store(time.Now().Unix())
		fmt.Fprintf(w, `{"app":%q,"version":%q,"author":%q}`, appName, version, author)
	})

	mux.HandleFunc("/api/quit", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("bye"))
		go func() { time.Sleep(300 * time.Millisecond); os.Exit(0) }()
	})

	mux.HandleFunc("/api/catalog", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"shapes": geom.Catalog(), "fonts": geom.FontNames(), "version": version, "app": appName, "author": author})
	})

	mux.HandleFunc("/api/build", func(w http.ResponseWriter, r *http.Request) {
		lastPing.Store(time.Now().Unix())
		var req buildReq
		if err := json.NewDecoder(io.LimitReader(r.Body, 64<<20)).Decode(&req); err != nil || req.Node == nil {
			http.Error(w, "bad request", 400)
			return
		}
		buildSem <- struct{}{}
		defer func() { <-buildSem }()
		if r.Context().Err() != nil {
			return
		}
		m, err := geom.BuildLocal(req.Node, qOr(req.Q))
		if err != nil {
			writeBuildErr(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		if m.Info != "" {
			w.Header().Set("X-Info", url.QueryEscape(m.Info))
		}
		if len(m.Ports) > 0 {
			if b, err := json.Marshal(m.Ports); err == nil {
				w.Header().Set("X-Ports", string(b))
			}
		}
		w.Write(m.Encode(32))
	})

	mux.HandleFunc("/api/export", func(w http.ResponseWriter, r *http.Request) {
		var req exportReq
		if err := json.NewDecoder(io.LimitReader(r.Body, 64<<20)).Decode(&req); err != nil {
			http.Error(w, "bad request", 400)
			return
		}
		objs, err := worldObjects(req.Nodes, qOr(req.Q))
		if err != nil {
			writeBuildErr(w, err)
			return
		}
		if len(objs) == 0 {
			http.Error(w, "Nothing to export: add a solid shape first (holes on their own are not printed).", 422)
			return
		}
		f := r.URL.Query().Get("fmt")
		if !req.Separate && (f == "stl" || f == "") {
			all := &geom.Mesh{}
			for _, o := range objs {
				all.Append(o.Mesh)
			}
			objs = []geom.NamedMesh{{Name: req.Name, Color: objs[0].Color, Mesh: all}}
		}
		name := safeName(req.Name)
		switch f {
		case "obj":
			sendFile(w, name+".obj", "text/plain", geom.WriteOBJ(objs))
		case "stlzip", "colorzip":
			var b []byte
			var err error
			if f == "colorzip" {
				b, err = geom.WriteSTLPerFilament(objs, name)
			} else {
				b, err = geom.WriteSTLZip(objs)
			}
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			sendFile(w, name+".zip", "application/zip", b)
		case "3mf", "3mfasm":
			b, err := geom.Write3MF(objs, f == "3mfasm")
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			sendFile(w, name+".3mf", "model/3mf", b)
		default:
			sendFile(w, name+".stl", "model/stl", geom.WriteSTL(objs))
		}
	})

	mux.HandleFunc("/api/analyze", func(w http.ResponseWriter, r *http.Request) {
		var req exportReq
		if err := json.NewDecoder(io.LimitReader(r.Body, 64<<20)).Decode(&req); err != nil {
			http.Error(w, "bad request", 400)
			return
		}
		type item struct {
			ID    string         `json:"id"`
			Name  string         `json:"name"`
			Stats geom.Stats     `json:"stats"`
			Walls geom.Thickness `json:"walls"`
		}
		var out []item
		for _, n := range req.Nodes {
			if n.Skipped() {
				continue
			}
			m, err := geom.BuildWorld(n, qOr(req.Q))
			if err != nil {
				writeBuildErr(w, err)
				return
			}
			m = m.FixTJunctions()
			out = append(out, item{n.ID, n.Name, geom.MeshStats(m), geom.WallThickness(m, 4000, geom.MinWall-0.05, 0.95, false)})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(out)
	})

	// scan an imported (or any) model and describe it as a board spec
	mux.HandleFunc("/api/scan", func(w http.ResponseWriter, r *http.Request) {
		var req buildReq
		if err := json.NewDecoder(io.LimitReader(r.Body, 512<<20)).Decode(&req); err != nil || req.Node == nil {
			http.Error(w, "bad request", 400)
			return
		}
		m, err := geom.BuildWorld(req.Node, qOr(req.Q))
		if err != nil {
			writeBuildErr(w, err)
			return
		}
		name := req.Node.Name
		if name == "" {
			name = "Imported model"
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(geom.ScanModel(m, name))
	})

	// collision check: volume shared by every pair of solid parts
	mux.HandleFunc("/api/interference", func(w http.ResponseWriter, r *http.Request) {
		var req exportReq
		if err := json.NewDecoder(io.LimitReader(r.Body, 64<<20)).Decode(&req); err != nil {
			http.Error(w, "bad request", 400)
			return
		}
		type part struct {
			id string
			m  *geom.Mesh
			lo geom.Vec3
			hi geom.Vec3
		}
		var parts []part
		for _, n := range req.Nodes {
			if n.Skipped() {
				continue
			}
			m, err := geom.BuildWorld(n, qOr(req.Q))
			if err != nil {
				writeBuildErr(w, err)
				return
			}
			lo, hi := m.Bounds()
			parts = append(parts, part{n.ID, m, lo, hi})
		}
		type clash struct {
			A      string     `json:"a"`
			B      string     `json:"b"`
			Volume float64    `json:"volume"`
			Center [3]float64 `json:"center"`
		}
		out := []clash{}
		for i := 0; i < len(parts); i++ {
			for j := i + 1; j < len(parts); j++ {
				a, b := parts[i], parts[j]
				if a.lo.X > b.hi.X || b.lo.X > a.hi.X || a.lo.Y > b.hi.Y || b.lo.Y > a.hi.Y || a.lo.Z > b.hi.Z || b.lo.Z > a.hi.Z {
					continue
				}
				x := geom.Intersect(a.m, b.m)
				if v := x.Volume(); v > 0.05 {
					lo, hi := x.Bounds()
					out = append(out, clash{a.id, b.id, v, [3]float64{(lo.X + hi.X) / 2, (lo.Y + hi.Y) / 2, (lo.Z + hi.Z) / 2}})
				}
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(out)
	})

	// import STL/OBJ: returns a mesh reference, centred on X/Y and dropped to Z=0
	mux.HandleFunc("/api/import", func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(io.LimitReader(r.Body, 512<<20))
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		var m *geom.Mesh
		if strings.EqualFold(r.URL.Query().Get("fmt"), "obj") {
			m, err = geom.ReadOBJ(b)
		} else {
			m, err = geom.ReadSTL(b)
		}
		if err != nil {
			http.Error(w, err.Error(), 422)
			return
		}
		if len(m.Tris) > 1_500_000 {
			http.Error(w, "That model is too detailed (over 1.5 million triangles).", 422)
			return
		}
		lo, hi := m.Bounds()
		m = m.Transformed(geom.Translate(-(lo.X+hi.X)/2, -(lo.Y+hi.Y)/2, -lo.Z))
		m.Orient()
		ref := geom.StoreMesh(m)
		json.NewEncoder(w).Encode(map[string]any{"ref": ref, "tris": len(m.Tris)})
	})

	// raw float32 triangle data in/out, used to save imported meshes inside project files
	mux.HandleFunc("/api/raw", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			b, err := io.ReadAll(io.LimitReader(r.Body, 512<<20))
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			ref := geom.StoreMesh(geom.DecodeRaw(b))
			json.NewEncoder(w).Encode(map[string]any{"ref": ref})
			return
		}
		n := &geom.Node{Type: "mesh", MeshRef: r.URL.Query().Get("ref")}
		m, err := geom.BuildLocal(n, 1)
		if err != nil {
			writeBuildErr(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(m.EncodeRaw())
	})
}

func qOr(q float64) float64 {
	if q <= 0 {
		return 1
	}
	return q
}

func writeBuildErr(w http.ResponseWriter, err error) {
	var mm geom.ErrMissingMesh
	if errors.As(err, &mm) {
		w.Header().Set("X-Missing-Mesh", mm.Ref)
		http.Error(w, err.Error(), http.StatusGone)
		return
	}
	log.Println("build error:", err)
	http.Error(w, err.Error(), 422)
}

func worldObjects(nodes []*geom.Node, q float64) ([]geom.NamedMesh, error) {
	var out []geom.NamedMesh
	for _, n := range nodes {
		if n.Skipped() {
			continue
		}
		m, err := geom.BuildWorld(n, q)
		if err != nil {
			return nil, err
		}
		if len(m.Tris) == 0 {
			continue
		}
		name := n.Name
		if name == "" {
			name = n.Type
		}
		out = append(out, geom.NamedMesh{Name: name, Color: n.Color, Extruder: n.Slot, Mesh: m.FixTJunctions()})
	}
	return out, nil
}

func safeName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		s = "part"
	}
	return strings.Map(func(r rune) rune {
		if strings.ContainsRune(`<>:"/\|?*`, r) || r < 32 {
			return '_'
		}
		return r
	}, s)
}

func sendFile(w http.ResponseWriter, name, ctype string, b []byte) {
	w.Header().Set("Content-Type", ctype)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, name))
	w.Write(b)
}

// migrateOldData moves settings, autosave and the window profile from the
// app's previous name (Smithers) so nothing is lost after the rename.
func migrateOldData() {
	d, err := os.UserConfigDir()
	if err != nil {
		return
	}
	newDir := filepath.Join(d, appName)
	if _, err := os.Stat(newDir); err == nil {
		return
	}
	for _, old := range []string{"Smithers", "PartSmith"} {
		oldDir := filepath.Join(d, old)
		if st, err := os.Stat(oldDir); err == nil && st.IsDir() {
			_ = os.Rename(oldDir, newDir)
			return
		}
	}
}
