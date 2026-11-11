//go:build darwin && cgo

package scopie

// pico_darwin.go: PicoScopes on macOS through Pico's own driver libraries
// (PicoSDK for macOS, or the ones inside PicoScope 7). Every function Scopie
// uses takes whole numbers and pointers, so one C routine calls them all:
// up to ten integer arguments, with the floating-point registers zeroed in
// case a function also takes a float (Scopie always passes 0 there).

/*
#include <dlfcn.h>
#include <stdint.h>
#include <stdlib.h>

typedef uintptr_t (*tkPicoFn)(uintptr_t, uintptr_t, uintptr_t, uintptr_t, uintptr_t,
	uintptr_t, uintptr_t, uintptr_t, uintptr_t, uintptr_t, float, float, float, float);

static uintptr_t tk_pico_call(void *f, uintptr_t *a) {
	return ((tkPicoFn)f)(a[0], a[1], a[2], a[3], a[4], a[5], a[6], a[7], a[8], a[9], 0, 0, 0, 0);
}
static void *tk_dlopen(const char *p) { return dlopen(p, RTLD_NOW | RTLD_LOCAL); }
static void *tk_dlsym(void *h, const char *n) { return dlsym(h, n); }
*/
import "C"

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"unsafe"
)

type macPicoLib struct {
	h   unsafe.Pointer
	mu  sync.Mutex
	fns map[string]unsafe.Pointer
}

func (l *macPicoLib) call(name string, args ...uintptr) (uintptr, error) {
	if len(args) > 10 {
		return 0, errors.New("too many arguments for " + name)
	}
	l.mu.Lock()
	f, ok := l.fns[name]
	if !ok {
		cn := C.CString(name)
		f = C.tk_dlsym(l.h, cn)
		C.free(unsafe.Pointer(cn))
		l.fns[name] = f
	}
	l.mu.Unlock()
	if f == nil {
		return 0, errors.New("the PicoScope driver has no " + name)
	}
	var a [10]C.uintptr_t
	for i, v := range args {
		a[i] = C.uintptr_t(v)
	}
	return uintptr(C.tk_pico_call(f, &a[0])), nil
}

var picoMu sync.Mutex
var picoLibs = map[string]picoLib{}

// loadPico finds one of Pico's driver libraries.
func loadPico(name string) picoLib {
	picoMu.Lock()
	defer picoMu.Unlock()
	if l, ok := picoLibs[name]; ok {
		return l
	}
	file := "lib" + name + ".dylib"
	paths := []string{
		filepath.Join("/Library/Frameworks/PicoSDK.framework/Libraries", "lib"+name, file),
		filepath.Join("/Library/Frameworks/PicoSDK.framework/Libraries", file),
	}
	apps, _ := filepath.Glob("/Applications/PicoScope*.app/Contents/Resources/lib")
	for _, a := range apps {
		paths = append(paths, filepath.Join(a, file))
	}
	paths = append(paths, file) // anywhere the system looks
	var lib picoLib
	for _, p := range paths {
		if filepath.IsAbs(p) {
			if _, err := os.Stat(p); err != nil {
				continue
			}
		}
		cp := C.CString(p)
		h := C.tk_dlopen(cp)
		C.free(unsafe.Pointer(cp))
		if h != nil {
			lib = &macPicoLib{h: h, fns: map[string]unsafe.Pointer{}}
			break
		}
	}
	picoLibs[name] = lib
	return lib
}
