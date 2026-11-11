//go:build !windows && !(darwin && cgo)

package scopie

// loadPico: Pico's libraries are used on Windows and the Mac build.
func loadPico(name string) picoLib { return nil }
