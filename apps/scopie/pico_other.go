//go:build !windows

package scopie

// loadPico: Pico's libraries are only used on Windows in this version.
func loadPico(name string) picoLib { return nil }
