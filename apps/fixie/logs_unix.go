//go:build linux || darwin

package fixie

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"tinkerie/shared/droid"
)

func hideWindow(cmd *exec.Cmd) {}

func logsAvailable() (bool, string) {
	if droid.Active() {
		return false, "Android only lets an app read its own log: open a log you've saved instead (for example from adb logcat or a bug report)"
	}
	if runtime.GOOS == "darwin" {
		return true, "macOS's log (errors and faults from the last day, and why it last shut down) and its panic and crash reports"
	}
	return true, "the system journal (warnings and errors from the last three days), the kernel log, and on a Raspberry Pi its power and heat record"
}

func readSystemLogs(ctx context.Context) []logSource {
	if runtime.GOOS == "darwin" {
		return macLogs(ctx)
	}
	return linuxLogs(ctx)
}

func have(cmd string) bool { _, err := exec.LookPath(cmd); return err == nil }

func linuxLogs(ctx context.Context) []logSource {
	var out []logSource
	if have("journalctl") {
		s := source(ctx, "System journal (warnings and errors, last 3 days)", "journalctl", "--no-pager", "-o", "short-iso", "-p", "warning", "--since", "-3d", "-n", "6000")
		if s.OK && strings.Contains(s.text, "not seeing messages from other users and the system") {
			s.Note = "Only your own messages: add yourself to the 'adm' or 'systemd-journal' group (sudo usermod -aG adm $USER) to see the system's"
		}
		out = append(out, s)
	}
	if have("dmesg") {
		s := source(ctx, "Kernel log (dmesg)", "dmesg", "--level=emerg,alert,crit,err,warn", "--ctime")
		if !s.OK {
			s.Note = "The kernel log needs permission (" + s.Note + "); the journal above includes it if you can read that"
		}
		out = append(out, s)
	}
	if len(out) == 0 || !out[0].OK {
		for _, f := range []string{"/var/log/syslog", "/var/log/messages", "/var/log/kern.log"} {
			if b, err := os.ReadFile(f); err == nil {
				out = append(out, logSource{Name: f, OK: true, Size: len(b), text: keepEnd(string(b))})
			}
		}
	}
	if have("vcgencmd") { // Raspberry Pi: under-voltage and heat since start-up
		out = append(out, source(ctx, "Raspberry Pi power and heat (vcgencmd get_throttled)", "vcgencmd", "get_throttled"))
	}
	return out
}

func macLogs(ctx context.Context) []logSource {
	out := []logSource{
		source(ctx, "Why the Mac last shut down (last 7 days)", "log", "show", "--last", "7d", "--style", "syslog",
			"--predicate", `eventMessage CONTAINS "Previous shutdown cause"`),
		source(ctx, "Errors and faults (last day)", "log", "show", "--last", "1d", "--style", "syslog",
			"--predicate", `messageType == fault OR (messageType == error AND (process == "kernel" OR subsystem BEGINSWITH "com.apple.iokit" OR subsystem == "com.apple.apfs"))`),
	}
	home, _ := os.UserHomeDir()
	reports := newest([]string{"/Library/Logs/DiagnosticReports/*.panic", "/Library/Logs/DiagnosticReports/*panic*.ips",
		"/Library/Logs/DiagnosticReports/*.ips", filepath.Join(home, "Library/Logs/DiagnosticReports/*.ips"),
		filepath.Join(home, "Library/Logs/DiagnosticReports/*.crash")}, 15)
	var b strings.Builder
	for _, f := range reports {
		if d, err := os.ReadFile(f); err == nil {
			if len(d) > 64<<10 {
				d = d[:64<<10]
			}
			b.WriteString("==== " + filepath.Base(f) + "\n")
			// newer reports are JSON with the panic text inside a string
			b.WriteString(strings.NewReplacer(`\n`, "\n", `\t`, " ", `\"`, `"`).Replace(string(d)))
			b.WriteString("\n")
		}
	}
	rs := logSource{Name: "Panic and crash reports (newest 15)", OK: true, Size: b.Len(), text: b.String()}
	if len(reports) == 0 {
		rs.Note = "None found (good news)"
	}
	return append(out, rs)
}

func newest(globs []string, n int) []string {
	type f struct {
		p string
		t int64
	}
	var all []f
	seen := map[string]bool{}
	for _, g := range globs {
		m, _ := filepath.Glob(g)
		for _, p := range m {
			if st, err := os.Stat(p); err == nil && !seen[p] {
				seen[p] = true
				all = append(all, f{p, st.ModTime().UnixNano()})
			}
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].t > all[j].t })
	var out []string
	for i := 0; i < len(all) && i < n; i++ {
		out = append(out, all[i].p)
	}
	return out
}
