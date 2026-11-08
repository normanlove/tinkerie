package flashie

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

func diskutilPlist(args ...string) (map[string]any, error) {
	out, err := exec.Command("diskutil", append(args[:1:1], append([]string{"-plist"}, args[1:]...)...)...).Output()
	if err != nil {
		return nil, err
	}
	v, err := parsePlist(out)
	if err != nil {
		return nil, err
	}
	m, _ := v.(map[string]any)
	return m, nil
}

var wholeDiskRe = regexp.MustCompile(`^(disk\d+)`)

func bootDisks() map[string]bool {
	out := map[string]bool{}
	info, err := diskutilPlist("info", "/")
	if err != nil {
		return out
	}
	if p, ok := info["ParentWholeDisk"].(string); ok {
		out[p] = true
	}
	if stores, ok := info["APFSPhysicalStores"].([]any); ok {
		for _, s := range stores {
			if m, ok := s.(map[string]any); ok {
				if id, ok := m["APFSPhysicalStore"].(string); ok {
					if w := wholeDiskRe.FindString(id); w != "" {
						out[w] = true
					}
				}
			}
		}
	}
	return out
}

func listDisksOS() ([]Disk, error) {
	l, err := diskutilPlist("list", "physical")
	if err != nil {
		return nil, fmt.Errorf("couldn't list drives: %v", err)
	}
	boot := bootDisks()
	mountOut, _ := exec.Command("mount").Output()
	var out []Disk
	whole, _ := l["WholeDisks"].([]any)
	for _, w := range whole {
		id, _ := w.(string)
		if id == "" || boot[id] {
			continue
		}
		info, err := diskutilPlist("info", id)
		if err != nil {
			continue
		}
		internal, _ := info["Internal"].(bool)
		removable, _ := info["RemovableMedia"].(bool)
		if r, ok := info["Removable"].(bool); ok && r {
			removable = true
		}
		bus, _ := info["BusProtocol"].(string)
		sd := strings.Contains(bus, "Secure Digital") || strings.Contains(bus, "SD")
		if internal && !removable && !sd {
			continue
		}
		if v, _ := info["VirtualOrPhysical"].(string); v == "Virtual" {
			continue
		}
		size, _ := info["TotalSize"].(int64)
		if size == 0 {
			size, _ = info["Size"].(int64)
		}
		if size == 0 {
			continue
		}
		name, _ := info["MediaName"].(string)
		if name == "" {
			name, _ = info["IORegistryEntryName"].(string)
		}
		b := "USB"
		if sd {
			b = "SD card"
		} else if bus != "" && bus != "USB" {
			b = bus
		}
		d := Disk{ID: "/dev/" + id, Name: strings.TrimSpace(strings.TrimSuffix(name, " Media")), Size: size, Bus: b}
		if d.Name == "" {
			d.Name = b + " drive"
		}
		for _, line := range strings.Split(string(mountOut), "\n") {
			if f := strings.SplitN(line, " on ", 2); len(f) == 2 && wholeDiskRe.FindString(strings.TrimPrefix(f[0], "/dev/")) == id {
				dir := f[1]
				if i := strings.LastIndex(dir, " ("); i > 0 {
					dir = dir[:i]
				}
				d.Mounts = append(d.Mounts, dir)
			}
		}
		out = append(out, d)
	}
	return out, nil
}

func rawPath(id string) string { return strings.Replace(id, "/dev/disk", "/dev/rdisk", 1) }

func openDiskForWrite(d Disk) (rawDisk, error) {
	if t, ok := testDiskFile(d); ok {
		return t, nil
	}
	if out, err := exec.Command("diskutil", "unmountDisk", "force", d.ID).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("couldn't unmount the drive — close anything using it and try again (%s)", strings.TrimSpace(string(out)))
	}
	f, err := os.OpenFile(rawPath(d.ID), os.O_RDWR, 0)
	if err != nil {
		if errors.Is(err, os.ErrPermission) && os.Geteuid() == 0 {
			return nil, errors.New("the drive is read-only — if it's an SD card, slide its lock switch up, away from LOCK")
		}
		return nil, fmt.Errorf("couldn't open the drive: %v", err)
	}
	return f, nil
}

func finishDisk(d Disk, eject bool) {
	if _, ok := testDiskFile(d); ok {
		return
	}
	if eject {
		exec.Command("diskutil", "eject", d.ID).Run()
	}
}

func canWriteDirect(d Disk) bool {
	if _, ok := testDiskFile(d); ok {
		return true
	}
	return os.Geteuid() == 0
}

func shQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

func elevate(base, token string) (chan error, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	sh := strings.Join([]string{shQuote(exe), "--app", "flashie", "--img-helper", shQuote(base), shQuote(token), ">/dev/null", "2>&1"}, " ")
	as := strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(sh)
	script := `do shell script "` + as + `" with prompt "Flashie needs your password to write the drive." with administrator privileges`
	cmd := exec.Command("osascript", "-e", script)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	done := make(chan error, 1)
	go func() {
		err := cmd.Wait()
		if err != nil && strings.Contains(stderr.String(), "-128") {
			err = errors.New("the password prompt was cancelled")
		}
		done <- err
	}()
	return done, nil
}
