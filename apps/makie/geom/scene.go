package geom

import (
	"container/list"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
)

// Node is one object of the editor's scene tree.
type Node struct {
	ID       string     `json:"id"`
	Name     string     `json:"name,omitempty"`
	Type     string     `json:"type"` // a catalog type, "group" or "mesh"
	Params   Params     `json:"params,omitempty"`
	Pos      [3]float64 `json:"pos"`
	Rot      [3]float64 `json:"rot"`
	Scale    [3]float64 `json:"scale"`
	Hole     bool       `json:"hole"`
	Hidden   bool       `json:"hidden,omitempty"`
	Mode     string     `json:"mode,omitempty"` // group: "combine" (default) or "intersect"
	MeshRef  string     `json:"meshRef,omitempty"`
	Children []*Node    `json:"children,omitempty"`
	Color    string     `json:"color,omitempty"`
	Slot     int        `json:"slot,omitempty"`      // filament / AMS slot (1-based)
	Wirie    string     `json:"wirie,omitempty"`     // role in a Wirie board case: base, lid, extra
	Ref      bool       `json:"reference,omitempty"` // reference object (e.g. an imported board): not printed
	RefOut   bool       `json:"refExport,omitempty"` // …unless the user asks for it
}

// Skipped reports whether a node is left out of exports and print checks.
func (n *Node) Skipped() bool { return n.Hidden || n.Hole || GroupIsHole(n) || (n.Ref && !n.RefOut) }

func (n *Node) Matrix() Mat4 { return TRS(n.Pos, n.Rot, n.Scale) }

// ---- uploaded meshes (imports) ----

var (
	meshMu    sync.RWMutex
	meshStore = map[string]*Mesh{}
)

// StoreMesh keeps an imported mesh and returns its content hash.
func StoreMesh(m *Mesh) string {
	raw := m.EncodeRaw()
	h := sha256.Sum256(raw)
	id := hex.EncodeToString(h[:12])
	meshMu.Lock()
	meshStore[id] = m
	meshMu.Unlock()
	return id
}

func getMesh(id string) (*Mesh, bool) {
	meshMu.RLock()
	defer meshMu.RUnlock()
	m, ok := meshStore[id]
	return m, ok
}

// ErrMissingMesh signals the editor must re-upload an imported mesh.
type ErrMissingMesh struct{ Ref string }

func (e ErrMissingMesh) Error() string { return "missing mesh " + e.Ref }

// ---- build cache ----

type cacheEntry struct {
	key  string
	mesh *Mesh
}

type lru struct {
	mu    sync.Mutex
	cap   int
	ll    *list.List
	items map[string]*list.Element
	tris  int
}

var cache = &lru{cap: 400, ll: list.New(), items: map[string]*list.Element{}}

func (c *lru) get(k string) (*Mesh, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.items[k]; ok {
		c.ll.MoveToFront(e)
		return e.Value.(*cacheEntry).mesh, true
	}
	return nil, false
}

func (c *lru) put(k string, m *Mesh) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.items[k]; ok {
		c.ll.MoveToFront(e)
		return
	}
	e := c.ll.PushFront(&cacheEntry{k, m})
	c.items[k] = e
	c.tris += len(m.Tris)
	for c.ll.Len() > c.cap || c.tris > 6_000_000 {
		last := c.ll.Back()
		if last == nil {
			break
		}
		ent := last.Value.(*cacheEntry)
		c.tris -= len(ent.mesh.Tris)
		delete(c.items, ent.key)
		c.ll.Remove(last)
		if c.ll.Len() <= 1 {
			break
		}
	}
}

// key identifies a node's local geometry (its own transform excluded).
// Only fields that change the shape are included, so renaming or recolouring is free.
type keyNode struct {
	T string     `json:"t"`
	P Params     `json:"p,omitempty"`
	X [9]float64 `json:"x"`
	H bool       `json:"h,omitempty"`
	V bool       `json:"v,omitempty"`
	M string     `json:"m,omitempty"`
	R string     `json:"r,omitempty"`
	C []keyNode  `json:"c,omitempty"`
}

func toKey(n *Node, root bool) keyNode {
	k := keyNode{T: n.Type, P: n.Params, M: n.Mode, R: n.MeshRef}
	k.H = n.Hole // threads differ for solids and holes
	if !root {
		k.X = [9]float64{n.Pos[0], n.Pos[1], n.Pos[2], n.Rot[0], n.Rot[1], n.Rot[2], n.Scale[0], n.Scale[1], n.Scale[2]}
		k.H, k.V = n.Hole, n.Hidden
	}
	for _, c := range n.Children {
		k.C = append(k.C, toKey(c, false))
	}
	return k
}

func key(n *Node, q float64) string {
	b, _ := json.Marshal(struct {
		N keyNode
		Q float64
	}{toKey(n, true), q})
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// BuildLocal returns the node's mesh in its own coordinate space.
// For groups: union of solid children minus union of hole children (Tinkercad rules).
func BuildLocal(n *Node, q float64) (*Mesh, error) {
	k := key(n, q)
	if m, ok := cache.get(k); ok {
		return m, nil
	}
	var m *Mesh
	var err error
	switch n.Type {
	case "group":
		m, err = buildGroup(n, q)
	case "mesh":
		mm, ok := getMesh(n.MeshRef)
		if !ok {
			return nil, ErrMissingMesh{n.MeshRef}
		}
		m = mm
	default:
		p := Params{}
		for kk, v := range n.Params {
			p[kk] = v
		}
		p["_q"] = q
		p["_hole"] = n.Hole
		m, err = Generate(n.Type, p)
	}
	if err != nil {
		return nil, err
	}
	if m == nil {
		m = &Mesh{}
	}
	cache.put(k, m)
	return m, nil
}

func buildGroup(n *Node, q float64) (*Mesh, error) {
	var solids, holes []*Mesh
	allHoles := true
	for _, c := range n.Children {
		if c.Hidden {
			continue
		}
		lm, err := BuildLocal(c, q)
		if err != nil {
			return nil, err
		}
		wm := lm.Transformed(c.Matrix())
		if c.Hole {
			holes = append(holes, wm)
		} else {
			allHoles = false
			solids = append(solids, wm)
		}
	}
	if allHoles {
		// a group of only holes is itself a hole shape: union them
		return UnionAll(holes), nil
	}
	var res *Mesh
	if n.Mode == "intersect" {
		for i, s := range solids {
			if i == 0 {
				res = s.Clone()
			} else {
				res = Intersect(res, s)
			}
		}
	} else {
		res = UnionAll(solids)
	}
	base := res
	for _, h := range holes {
		res = Subtract(res, h)
	}
	// clean up BSP T-junctions so edges, normals and exports are watertight
	res = res.FixTJunctions()
	if len(holes) > 1 && openEdgeCount(res) > 0 {
		// many cuts in a row can pile up rounding errors; cutting all the
		// holes at once often comes out cleaner
		alt := Subtract(base, UnionAll(holes)).FixTJunctions()
		if openEdgeCount(alt) < openEdgeCount(res) {
			res = alt
		}
	}
	return res, nil
}

// BuildWorld returns the node's mesh in world space.
func BuildWorld(n *Node, q float64) (*Mesh, error) {
	m, err := BuildLocal(n, q)
	if err != nil {
		return nil, err
	}
	return m.Transformed(n.Matrix()), nil
}

// GroupIsHole reports whether a group made only of holes acts as a hole.
func GroupIsHole(n *Node) bool {
	if n.Hole || n.Type != "group" {
		return n.Hole
	}
	for _, c := range n.Children {
		if !c.Hidden && !c.Hole {
			return false
		}
	}
	return len(n.Children) > 0
}

func init() {
	_ = fmt.Sprint
}
