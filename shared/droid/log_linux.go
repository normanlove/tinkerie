//go:build linux

package droid

import (
	"os"
	"path/filepath"
	"syscall"
)

// On Android each app's messages (and any crash report) go to
// Files › Tinkerie › logs › <app>.log, so problems can be reported. The
// launcher's are collected by the Android app itself.
func init() {
	if !Active() || App() == "launcher" {
		return
	}
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, "logs")
	os.MkdirAll(dir, 0o755)
	p := filepath.Join(dir, App()+".log")
	if st, err := os.Stat(p); err == nil && st.Size() > 2<<20 {
		os.Remove(p)
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	syscall.Dup3(int(f.Fd()), 2, 0)
	syscall.Dup3(int(f.Fd()), 1, 0)
}
