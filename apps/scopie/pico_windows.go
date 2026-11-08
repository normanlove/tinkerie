//go:build windows

package scopie

import (
	"os"
	"path/filepath"
	"sync"
	"syscall"
)

type winPicoLib struct {
	dll   *syscall.DLL
	mu    sync.Mutex
	procs map[string]*syscall.Proc
}

func (l *winPicoLib) call(name string, args ...uintptr) (uintptr, error) {
	l.mu.Lock()
	p, ok := l.procs[name]
	if !ok {
		var err error
		p, err = l.dll.FindProc(name)
		if err != nil {
			l.mu.Unlock()
			return 0, err
		}
		l.procs[name] = p
	}
	l.mu.Unlock()
	r, _, _ := p.Call(args...)
	return r, nil
}

var picoMu sync.Mutex
var picoLibs = map[string]picoLib{}

// loadPico finds one of Pico's driver DLLs (installed by PicoSDK or PicoScope 7).
func loadPico(name string) picoLib {
	picoMu.Lock()
	defer picoMu.Unlock()
	if l, ok := picoLibs[name]; ok {
		return l
	}
	var paths []string
	for _, env := range []string{"ProgramFiles", "ProgramW6432"} {
		base := os.Getenv(env)
		if base == "" {
			continue
		}
		paths = append(paths,
			filepath.Join(base, "Pico Technology", "SDK", "lib", name+".dll"),
			filepath.Join(base, "Pico Technology", "PicoScope 7 T&M Stable", name+".dll"),
			filepath.Join(base, "Pico Technology", "PicoScope 7 T&M Early Access", name+".dll"),
			filepath.Join(base, "Pico Technology", "PicoScope 6", name+".dll"))
	}
	var lib picoLib
	for _, p := range paths {
		if _, err := os.Stat(p); err != nil {
			continue
		}
		if d, err := loadDLL(p); err == nil {
			lib = &winPicoLib{dll: d, procs: map[string]*syscall.Proc{}}
			break
		}
	}
	picoLibs[name] = lib
	return lib
}
