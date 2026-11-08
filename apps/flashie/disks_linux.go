package flashie

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

func readSys(p string) string {
	b, _ := os.ReadFile(p)
	return strings.TrimSpace(string(b))
}

// diskNamesFor maps a /sys/dev/block entry to the whole disks behind it,
// following partitions and device-mapper (LVM, LUKS) layers.
func diskNamesFor(sysPath string, out map[string]bool, depth int) {
	if depth > 8 {
		return
	}
	real, err := filepath.EvalSymlinks(sysPath)
	if err != nil {
		return
	}
	if _, err := os.Stat(filepath.Join(real, "partition")); err == nil {
		real = filepath.Dir(real)
	}
	name := filepath.Base(real)
	out[name] = true
	slaves, _ := os.ReadDir(filepath.Join(real, "slaves"))
	for _, s := range slaves {
		diskNamesFor(filepath.Join(real, "slaves", s.Name()), out, depth+1)
	}
}

// systemDisks returns the disks holding the running system, which are
// never offered for writing.
func systemDisks() map[string]bool {
	sys := map[string]bool{}
	important := map[string]bool{"/": true, "/boot": true, "/boot/firmware": true, "/boot/efi": true, "/usr": true, "/var": true, "/home": true}
	if f, err := os.Open("/proc/self/mountinfo"); err == nil {
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			fl := strings.Fields(sc.Text())
			if len(fl) < 10 || !important[unescapeMount(fl[4])] {
				continue
			}
			diskNamesFor("/sys/dev/block/"+fl[2], sys, 0)
			for i, x := range fl {
				if x == "-" && i+2 < len(fl) && strings.HasPrefix(fl[i+2], "/dev/") {
					devToDisks(fl[i+2], sys)
				}
			}
		}
		f.Close()
	}
	if b, err := os.ReadFile("/proc/swaps"); err == nil {
		for _, l := range strings.Split(string(b), "\n")[1:] {
			if fl := strings.Fields(l); len(fl) > 0 && strings.HasPrefix(fl[0], "/dev/") {
				devToDisks(fl[0], sys)
			}
		}
	}
	return sys
}

func devToDisks(dev string, out map[string]bool) {
	var st syscall.Stat_t
	if real, err := filepath.EvalSymlinks(dev); err == nil {
		dev = real
	}
	if syscall.Stat(dev, &st) != nil || st.Mode&syscall.S_IFMT != syscall.S_IFBLK {
		return
	}
	maj, min := (st.Rdev>>8)&0xfff|(st.Rdev>>32)&^0xfff, st.Rdev&0xff|(st.Rdev>>12)&^0xff
	diskNamesFor(fmt.Sprintf("/sys/dev/block/%d:%d", maj, min), out, 0)
}

func unescapeMount(s string) string {
	return strings.NewReplacer(`\040`, " ", `\011`, "\t", `\012`, "\n", `\134`, `\`).Replace(s)
}

// partOf reports whether dev (e.g. /dev/sdb1, /dev/mmcblk0p2) is disk or a partition of it.
func partOf(dev, disk string) bool {
	if dev == disk {
		return true
	}
	if !strings.HasPrefix(dev, disk) {
		return false
	}
	rest := dev[len(disk):]
	rest = strings.TrimPrefix(rest, "p")
	if rest == "" {
		return false
	}
	for _, c := range rest {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

type mountEnt struct{ dev, dir string }

func linuxMounts() []mountEnt {
	var out []mountEnt
	b, _ := os.ReadFile("/proc/self/mounts")
	for _, l := range strings.Split(string(b), "\n") {
		fl := strings.Fields(l)
		if len(fl) >= 2 && strings.HasPrefix(fl[0], "/dev/") {
			dev := fl[0]
			if real, err := filepath.EvalSymlinks(dev); err == nil {
				dev = real
			}
			out = append(out, mountEnt{dev, unescapeMount(fl[1])})
		}
	}
	return out
}

func listDisksOS() ([]Disk, error) {
	ents, err := os.ReadDir("/sys/block")
	if err != nil {
		return nil, err
	}
	sys := systemDisks()
	mounts := linuxMounts()
	var out []Disk
	for _, e := range ents {
		n := e.Name()
		skip := false
		for _, p := range []string{"loop", "ram", "zram", "dm-", "md", "sr", "nbd", "fd", "zd", "mtd"} {
			if strings.HasPrefix(n, p) {
				skip = true
			}
		}
		if skip || sys[n] || strings.Contains(n, "boot") || strings.Contains(n, "rpmb") {
			continue // also skip eMMC boot/rpmb areas (mmcblk0boot0)
		}
		base := "/sys/block/" + n
		real, _ := filepath.EvalSymlinks(base)
		bus := ""
		switch {
		case strings.Contains(real, "/usb"):
			bus = "USB"
		case strings.HasPrefix(n, "mmcblk"):
			if t := readSys(base + "/device/type"); t == "MMC" {
				continue // soldered-on eMMC storage, not a card
			}
			bus = "SD card"
		case readSys(base+"/removable") == "1":
			bus = "Removable"
		default:
			continue
		}
		var sectors int64
		fmt.Sscan(readSys(base+"/size"), &sectors)
		if sectors == 0 {
			continue // card reader with no card in it
		}
		name := strings.TrimSpace(readSys(base+"/device/vendor") + " " + readSys(base+"/device/model"))
		if bus == "SD card" {
			name = strings.TrimSpace(readSys(base + "/device/name"))
			if name == "" {
				name = "SD card"
			} else {
				name += " SD card"
			}
		}
		if name == "" {
			name = bus + " drive"
		}
		d := Disk{ID: "/dev/" + n, Name: name, Size: sectors * 512, Bus: bus}
		for _, m := range mounts {
			if partOf(m.dev, d.ID) {
				d.Mounts = append(d.Mounts, m.dir)
			}
		}
		out = append(out, d)
	}
	return out, nil
}

type linuxDisk struct{ *os.File }

func (l linuxDisk) dropCache() {
	syscall.Syscall(syscall.SYS_IOCTL, l.Fd(), 0x1261, 0) // BLKFLSBUF
}

func unmountAll(d Disk) error {
	for _, m := range linuxMounts() {
		if !partOf(m.dev, d.ID) {
			continue
		}
		if syscall.Unmount(m.dir, 0) == nil {
			continue
		}
		if exec.Command("udisksctl", "unmount", "--no-user-interaction", "-b", m.dev).Run() == nil {
			continue
		}
		if exec.Command("umount", m.dir).Run() == nil {
			continue
		}
		if syscall.Unmount(m.dir, 2) == nil { // MNT_DETACH, last resort
			continue
		}
		return fmt.Errorf("couldn't unmount %s — close any windows or programs using the drive and try again", m.dir)
	}
	return nil
}

func openDiskForWrite(d Disk) (rawDisk, error) {
	if t, ok := testDiskFile(d); ok {
		return t, nil
	}
	if err := unmountAll(d); err != nil {
		return nil, err
	}
	var f *os.File
	var err error
	for i := 0; i < 10; i++ { // the desktop may still be letting go of it
		f, err = os.OpenFile(d.ID, os.O_RDWR|syscall.O_EXCL|syscall.O_DIRECT, 0)
		if errors.Is(err, syscall.EINVAL) {
			f, err = os.OpenFile(d.ID, os.O_RDWR|syscall.O_EXCL, 0)
		}
		if !errors.Is(err, syscall.EBUSY) {
			break
		}
		unmountAll(d)
		sleepMS(500)
	}
	if err != nil {
		if errors.Is(err, syscall.EBUSY) {
			return nil, errors.New("the drive is still in use — close any windows or programs using it and try again")
		}
		if errors.Is(err, syscall.EROFS) || errors.Is(err, syscall.EACCES) && os.Geteuid() == 0 {
			return nil, errors.New("the drive is read-only — if it's an SD card, slide its lock switch up, away from LOCK")
		}
		return nil, fmt.Errorf("couldn't open the drive: %v", err)
	}
	return linuxDisk{f}, nil
}

func finishDisk(d Disk, eject bool) {
	if _, ok := testDiskFile(d); ok {
		return
	}
	if f, err := os.Open(d.ID); err == nil {
		syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), 0x125F, 0) // BLKRRPART: re-read partitions
		f.Close()
	}
	if eject {
		exec.Command("udisksctl", "power-off", "--no-user-interaction", "-b", d.ID).Run()
	}
}

// canWriteDirect reports whether Flashie itself may open the drive (root, or
// in the "disk" group) so no password prompt is needed.
func canWriteDirect(d Disk) bool {
	if _, ok := testDiskFile(d); ok {
		return true
	}
	f, err := os.OpenFile(d.ID, os.O_RDWR, 0)
	if err != nil {
		return false
	}
	f.Close()
	return true
}

// elevate starts the helper with administrator rights. done receives the
// helper's exit error once it ends.
func elevate(base, token string) (done chan error, err error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	args := []string{exe, "--app", "flashie", "--img-helper", base, token}
	var cmd *exec.Cmd
	if exec.Command("sudo", "-n", "true").Run() == nil {
		cmd = exec.Command("sudo", append([]string{"-n"}, args...)...) // Raspberry Pi OS: no password needed
	} else if p, err := exec.LookPath("pkexec"); err == nil {
		cmd = exec.Command(p, args...)
	} else {
		return nil, errors.New("writing a drive needs administrator rights, and pkexec isn't installed. Install it (sudo apt install pkexec), or start Tinkerie with: sudo " + exe + " --app flashie")
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	done = make(chan error, 1)
	go func() {
		err := cmd.Wait()
		if ee, ok := err.(*exec.ExitError); ok && (ee.ExitCode() == 126 || ee.ExitCode() == 127) {
			err = errors.New("the password prompt was cancelled")
		}
		done <- err
	}()
	return done, nil
}
