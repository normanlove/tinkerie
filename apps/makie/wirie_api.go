package makie

// HTTP endpoints for Wirie boards (see geom/wirie.go).

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"tinkerie/apps/makie/geom"
)

type wirieResp struct {
	Project  string              `json:"project"`
	Board    string              `json:"board"`
	Textures *geom.WirieTextures `json:"textures,omitempty"`
	Notes    []string            `json:"notes"`
	Params   []geom.Param        `json:"caseParams"`
	Gen      geom.WirieGenerator `json:"generator"`
	Parts    int                 `json:"parts"`
	Size     [3]float64          `json:"size"`
	File     string              `json:"file,omitempty"`
}

func wirieResult(b []byte, file string) (*wirieResp, error) {
	c, notes, err := geom.ParseWirie(b)
	if err != nil {
		return nil, err
	}
	if notes == nil {
		notes = []string{}
	}
	return &wirieResp{Project: c.Project, Board: c.Compact(), Textures: c.Textures, Notes: notes, Params: geom.CaseParams(c), Gen: c.Generator,
		Parts: len(c.Parts), Size: [3]float64{c.Board.Width, c.Board.Height, c.Board.Thickness}, File: file}, nil
}

func sendWirie(w http.ResponseWriter, b []byte, file string) {
	res, err := wirieResult(b, file)
	if err != nil {
		var we geom.WirieError
		if errors.As(err, &we) {
			http.Error(w, we.Msg, 422)
			return
		}
		http.Error(w, err.Error(), 422)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

// wirieDirs lists the folders Wirie exports into (Documents, and OneDrive's
// Documents when Windows keeps Documents in OneDrive).
func wirieDirs() []string {
	var roots []string
	if h, err := os.UserHomeDir(); err == nil {
		roots = append(roots, filepath.Join(h, "Documents"), filepath.Join(h, "OneDrive", "Documents"))
	}
	for _, env := range []string{"OneDrive", "OneDriveConsumer", "OneDriveCommercial"} {
		if v := os.Getenv(env); v != "" {
			roots = append(roots, filepath.Join(v, "Documents"))
		}
	}
	if v := os.Getenv("USERPROFILE"); v != "" {
		roots = append(roots, filepath.Join(v, "Documents"))
	}
	seen := map[string]bool{}
	var out []string
	for _, r := range roots {
		d := filepath.Join(r, "Wirie", "Exports")
		k := strings.ToLower(filepath.Clean(d))
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, d)
	}
	return out
}

type wirieFile struct {
	Path     string `json:"path"`
	Name     string `json:"name"`
	Project  string `json:"project"`
	Modified int64  `json:"modified"`
	Size     int64  `json:"size"`
	OK       bool   `json:"ok"`
	Problem  string `json:"problem,omitempty"`
}

var (
	wfMu    sync.Mutex
	wfCache = map[string]wirieFile{} // path|mtime → summary
)

func recentWirie() []wirieFile {
	var out []wirieFile
	for _, d := range wirieDirs() {
		ents, err := os.ReadDir(d)
		if err != nil {
			continue
		}
		for _, e := range ents {
			if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".makie.json") {
				continue
			}
			fi, err := e.Info()
			if err != nil {
				continue
			}
			p := filepath.Join(d, e.Name())
			key := p + "|" + fi.ModTime().String()
			wfMu.Lock()
			f, ok := wfCache[key]
			wfMu.Unlock()
			if !ok {
				f = wirieFile{Path: p, Name: strings.TrimSuffix(e.Name(), ".makie.json"), Modified: fi.ModTime().UnixMilli(), Size: fi.Size()}
				if fi.Size() > 64<<20 {
					f.Problem = "too big"
				} else if b, err := os.ReadFile(p); err == nil {
					if c, _, err := geom.ParseWirie(b); err == nil {
						f.OK, f.Project = true, c.Project
					} else {
						f.Problem = err.Error()
					}
				}
				wfMu.Lock()
				wfCache[key] = f
				wfMu.Unlock()
			}
			out = append(out, f)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Modified > out[j].Modified })
	if len(out) > 30 {
		out = out[:30]
	}
	return out
}

// only files inside the Wirie export folders can be opened by path
func allowedWiriePath(p string) bool {
	p = filepath.Clean(p)
	if !strings.HasSuffix(strings.ToLower(p), ".makie.json") {
		return false
	}
	for _, d := range wirieDirs() {
		if strings.EqualFold(filepath.Dir(p), filepath.Clean(d)) {
			return true
		}
	}
	return false
}

type wirieCaseReq struct {
	Board string      `json:"board"`
	Opts  geom.Params `json:"opts"`
	Q     float64     `json:"q"`
}

type wirieClashReq struct {
	Board *geom.Node   `json:"board"`
	Nodes []*geom.Node `json:"nodes"`
	Q     float64      `json:"q"`
}

func registerWirieAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/wirie/parse", func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(io.LimitReader(r.Body, 128<<20))
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		sendWirie(w, b, "")
	})
	mux.HandleFunc("/api/wirie/recent", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		files := recentWirie()
		if files == nil {
			files = []wirieFile{}
		}
		json.NewEncoder(w).Encode(map[string]any{"dirs": wirieDirs(), "files": files, "now": time.Now().UnixMilli()})
	})
	mux.HandleFunc("/api/wirie/open", func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Query().Get("path")
		if !allowedWiriePath(p) {
			http.Error(w, "Makie can only open .makie.json files from the Wirie export folder this way. Use File → Import Wirie board instead.", 403)
			return
		}
		b, err := os.ReadFile(p)
		if err != nil {
			http.Error(w, "Could not read "+filepath.Base(p)+": "+err.Error(), 404)
			return
		}
		sendWirie(w, b, p)
	})
	mux.HandleFunc("/api/wirie/case", func(w http.ResponseWriter, r *http.Request) {
		var req wirieCaseReq
		if err := json.NewDecoder(io.LimitReader(r.Body, 16<<20)).Decode(&req); err != nil {
			http.Error(w, "bad request", 400)
			return
		}
		c, err := geom.WirieFromBoardJSON(req.Board)
		if err != nil {
			http.Error(w, err.Error(), 422)
			return
		}
		if req.Opts == nil {
			req.Opts = geom.Params{}
		}
		nodes, info := c.BuildCase(req.Opts)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"nodes": nodes, "info": info})
	})
	mux.HandleFunc("/api/wirie/clash", func(w http.ResponseWriter, r *http.Request) {
		var req wirieClashReq
		if err := json.NewDecoder(io.LimitReader(r.Body, 64<<20)).Decode(&req); err != nil || req.Board == nil {
			http.Error(w, "bad request", 400)
			return
		}
		c, err := geom.WirieFromBoardJSON(req.Board.Params.S("board", ""))
		if err != nil {
			http.Error(w, err.Error(), 422)
			return
		}
		buildSem <- struct{}{}
		defer func() { <-buildSem }()
		cl, err := c.ClashCheck(req.Nodes, qOr(req.Q), req.Board.Matrix().InverseRigid())
		if err != nil {
			writeBuildErr(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cl)
	})
}
