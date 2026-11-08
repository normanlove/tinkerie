//go:build windows

package meterie

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"
)

func Main() {
	defer func() {
		if r := recover(); r != nil {
			// write a crash report next to the settings so problems can be reported
			os.MkdirAll(configDir(), 0o755)
			p := filepath.Join(configDir(), "crash-"+time.Now().Format("20060102-150405")+".txt")
			os.WriteFile(p, []byte(fmt.Sprintf("Meterie %s crashed: %v\n\n%s", AppVersion, r, debug.Stack())), 0o644)
			fatalBox(fmt.Sprintf("Meterie hit an unexpected error and has to close.\n\n%v\n\nA crash report was saved to:\n%s", r, p))
		}
	}()
	runWindows()
}
