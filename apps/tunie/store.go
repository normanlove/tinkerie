// Tunie — store.go: saves printers, filaments, history and checklist progress,
// writes exports, and publishes the active profile for the rest of the suite.
// Part of the nrmluvdev maker suite. Created by nrmluvdev.
package tunie

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Store struct {
	mu        sync.Mutex
	dataDir   string
	suiteDir  string
	exportDir string
}

func NewStore() (*Store, error) {
	cfg, err := os.UserConfigDir()
	if err != nil || cfg == "" {
		cfg = os.TempDir()
	}
	s := &Store{
		dataDir:  filepath.Join(cfg, "nrmluvdev", "Tunie"),
		suiteDir: filepath.Join(cfg, "nrmluvdev", "Suite"),
	}
	home, _ := os.UserHomeDir()
	docs := filepath.Join(home, "Documents")
	if st, err := os.Stat(docs); err != nil || !st.IsDir() {
		docs = home
	}
	s.exportDir = filepath.Join(docs, "Tunie")
	for _, d := range []string{s.dataDir, s.suiteDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *Store) dbPath() string    { return filepath.Join(s.dataDir, "tunie-data.json") }
func (s *Store) suiteFile() string { return filepath.Join(s.suiteDir, "tunie-active.json") }

// Load returns the saved database (or an empty object).
func (s *Store) Load() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := os.ReadFile(s.dbPath())
	if err != nil || len(b) == 0 {
		// fall back to the backup if the main file is missing or damaged
		if bb, err2 := os.ReadFile(s.dbPath() + ".bak"); err2 == nil && len(bb) > 0 {
			return bb
		}
		return []byte("{}")
	}
	return b
}

// Save writes atomically and keeps the previous version as a backup.
func (s *Store) Save(b []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.dbPath()
	if _, err := os.Stat(p); err == nil {
		_ = copyFile(p, p+".bak")
	}
	return writeAtomic(p, b)
}

func (s *Store) WriteSuite(b []byte) error {
	return writeAtomic(s.suiteFile(), b)
}

var unsafeFile = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]+`)

// SaveExport writes a file into the export folder, or into dir (e.g. an SD card root).
func (s *Store) SaveExport(dir, name string, data []byte) (string, error) {
	name = strings.TrimSpace(unsafeFile.ReplaceAllString(name, "_"))
	if name == "" || name == "." || name == ".." {
		return "", errors.New("bad file name")
	}
	target := s.exportDir
	if dir != "" {
		ok := false
		for _, d := range listDrives() {
			if strings.EqualFold(filepath.Clean(d.Path), filepath.Clean(dir)) {
				ok = true
			}
		}
		if !ok {
			return "", errors.New("that drive is not available — is the SD card plugged in?")
		}
		target = dir
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		return "", err
	}
	p := filepath.Join(target, name)
	if err := os.WriteFile(p, data, 0o644); err != nil {
		return "", err
	}
	return p, nil
}

func writeAtomic(p string, b []byte) error {
	tmp := p + ".tmp-" + time.Now().Format("150405.000")
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, p); err != nil {
		_ = os.Remove(p)
		if err2 := os.Rename(tmp, p); err2 != nil {
			_ = os.Remove(tmp)
			return err2
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, b, 0o644)
}
