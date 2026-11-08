package flashie

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"tinkerie/shared/exfat"
	"tinkerie/shared/fat"
	"tinkerie/shared/udf/udftest"
)

type memDev struct{ b []byte }

func (m *memDev) ReadAt(p []byte, off int64) (int, error)  { return copy(p, m.b[off:]), nil }
func (m *memDev) WriteAt(p []byte, off int64) (int, error) { return copy(m.b[off:], p), nil }

type fileDev struct{ *os.File }

// sparseDisk is a pretend drive that doesn't take real disk space.
func sparseDisk(t *testing.T, mb int) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "card.bin")
	f, _ := os.Create(p)
	f.Truncate(int64(mb) << 20)
	f.WriteAt([]byte("OLD CARD CONTENTS"), 0)
	f.Close()
	t.Setenv("FLASHIE_IMG_TESTDISK", p+":"+strconv.Itoa(mb))
	return p
}

// piImage makes a Raspberry Pi OS-like image: an MBR, a FAT32 boot
// partition with the given files, and a (pretend) Linux partition.
func piImage(t *testing.T, files map[string]string) string {
	t.Helper()
	const start, size = 4 << 20, 300 << 20
	img := make([]byte, start+size+(1<<20))
	m := &memDev{img}
	fs, err := fat.Format32(m, start, size, "bootfs", 0x1234)
	if err != nil {
		t.Fatal(err)
	}
	for n, c := range files {
		if err := fs.WriteFile(n, []byte(c)); err != nil {
			t.Fatal(err)
		}
	}
	fs.Flush()
	e := img[446:]
	e[4] = 0x0C
	binary.LittleEndian.PutUint32(e[8:], start/512)
	binary.LittleEndian.PutUint32(e[12:], size/512)
	e = img[462:]
	e[4] = 0x83
	binary.LittleEndian.PutUint32(e[8:], (start+size)/512)
	binary.LittleEndian.PutUint32(e[12:], 2048)
	img[510], img[511] = 0x55, 0xAA
	p := filepath.Join(t.TempDir(), "pi.img")
	os.WriteFile(p, img, 0o644)
	return p
}

func bootFiles(t *testing.T, disk string) *fat.FS {
	t.Helper()
	f, err := os.Open(disk)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })
	fs, err := fat.Open(fileDev{f}, 4<<20)
	if err != nil {
		t.Fatal(err)
	}
	return fs
}

func sh(t *testing.T, script string) {
	t.Helper()
	if _, err := exec.LookPath("bash"); err != nil {
		return
	}
	c := exec.Command("bash", "-n")
	c.Stdin = strings.NewReader(script)
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("firstrun.sh isn't valid bash: %v %s\n%s", err, out, script)
	}
}

func yamlOK(t *testing.T, doc string) {
	t.Helper()
	c := exec.Command("python3", "-I", "-c", "import sys,yaml,json; print(json.dumps(yaml.safe_load(sys.stdin)))")
	c.Stdin = strings.NewReader(doc)
	out, err := c.CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), "No module named") || strings.Contains(err.Error(), "not found") {
			return
		}
		t.Fatalf("not valid YAML: %v %s\n%s", err, out, doc)
	}
}

var settings = customise{
	Hostname: "pi-shed", User: "graeme", Password: "s3cret 'pw'", WifiSSID: "My \"Home\" Wi-Fi", WifiPass: "wifi password!",
	WifiCountry: "gb", Timezone: "Europe/London", Keymap: "gb", SSH: true, SSHPassword: false,
	SSHKeys: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIExample nrmluvdev@laptop\n",
}

func TestCustomiseFirstrun(t *testing.T) {
	img := piImage(t, map[string]string{
		"cmdline.txt": "console=serial0,115200 console=tty1 root=PARTUUID=1234abcd-02 rootfstype=ext4 fsck.repair=yes rootwait quiet init=/usr/lib/raspberrypi-sys-mods/firstboot\n",
		"config.txt":  "dtparam=audio=on\n",
		"issue.txt":   "Raspberry Pi reference 2024-11-19\nGenerated using pi-gen\n",
	})
	disk := sparseDisk(t, 320)
	info, _ := sniffImage(img)
	c, err := settings.prepare()
	if err != nil {
		t.Fatal(err)
	}
	if c.Password != "" || !strings.HasPrefix(c.PassHash, "$6$") {
		t.Fatal("password not hashed")
	}
	warn, err := runWrite(writeSpec{Device: disk, Source: info, Verify: true, Custom: c}, func(progress) bool { return false })
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(warn, "firstrun.sh") {
		t.Fatalf("warn %q", warn)
	}
	fs := bootFiles(t, disk)
	cmd, _ := fs.ReadFile("cmdline.txt")
	if !strings.HasSuffix(string(cmd), "init=/usr/lib/raspberrypi-sys-mods/firstboot systemd.run=/boot/firmware/firstrun.sh systemd.run_success_action=reboot systemd.unit=kernel-command-line.target\n") {
		t.Fatalf("cmdline.txt: %q", cmd)
	}
	fr, err := fs.ReadFile("firstrun.sh")
	if err != nil {
		t.Fatal(err)
	}
	s := string(fr)
	sh(t, s)
	for _, want := range []string{"set_hostname 'pi-shed'", "userconf 'graeme' '$6$", "set_wlan 'My \"Home\" Wi-Fi' '", " 'GB'", "set_keymap 'gb'", "set_timezone 'Europe/London'", "PasswordAuthentication no", "authorized_keys", "ssh-ed25519 AAAA"} {
		if !strings.Contains(s, want) {
			t.Errorf("firstrun.sh lacks %q", want)
		}
	}
	if strings.Contains(s, "s3cret") || strings.Contains(s, "wifi password!") {
		t.Fatal("a password was written in plain text")
	}
	// An older (Bullseye) image keeps its boot files in /boot.
	img = piImage(t, map[string]string{"cmdline.txt": "root=x rootwait\n", "issue.txt": "Raspberry Pi reference 2022-01-28\n"})
	info, _ = sniffImage(img)
	disk = sparseDisk(t, 320)
	if _, err := runWrite(writeSpec{Device: disk, Source: info, Custom: c}, func(progress) bool { return false }); err != nil {
		t.Fatal(err)
	}
	cmd, _ = bootFiles(t, disk).ReadFile("cmdline.txt")
	if string(cmd) != "root=x rootwait systemd.run=/boot/firstrun.sh systemd.run_success_action=reboot systemd.unit=kernel-command-line.target\n" {
		t.Fatalf("cmdline.txt: %q", cmd)
	}
}

func TestCustomiseCloudInit(t *testing.T) {
	img := piImage(t, map[string]string{
		"cmdline.txt":    "root=PARTUUID=1-02 rootwait ds=nocloud\n",
		"user-data":      "#cloud-config\n# default\n",
		"meta-data":      "instance-id: rpios-image\n",
		"network-config": "network:\n  version: 2\n",
	})
	disk := sparseDisk(t, 320)
	info, _ := sniffImage(img)
	c, _ := settings.prepare()
	warn, err := runWrite(writeSpec{Device: disk, Source: info, Verify: true, Custom: c}, func(progress) bool { return false })
	if err != nil || !strings.Contains(warn, "cloud-init") {
		t.Fatal(warn, err)
	}
	fs := bootFiles(t, disk)
	ud, _ := fs.ReadFile("user-data")
	nc, _ := fs.ReadFile("network-config")
	md, _ := fs.ReadFile("meta-data")
	cmd, _ := fs.ReadFile("cmdline.txt")
	yamlOK(t, string(ud))
	yamlOK(t, string(nc))
	if string(md) != "instance-id: rpios-image\n" || strings.Contains(string(cmd), "systemd.run") || fs.Exists("firstrun.sh") {
		t.Fatal("cloud-init images must keep meta-data and cmdline.txt, and get no firstrun.sh")
	}
	for _, want := range []string{"#cloud-config\n", "hostname: pi-shed\n", "- name: graeme\n", `passwd: "$6$`, "timezone: Europe/London", `layout: "gb"`, "enable_ssh: true", "ssh_pwauth: false", `- "ssh-ed25519 AAAA`} {
		if !strings.Contains(string(ud), want) {
			t.Errorf("user-data lacks %q:\n%s", want, ud)
		}
	}
	for _, want := range []string{`regulatory-domain: "GB"`, `"My \"Home\" Wi-Fi":`, `password: "wifi password!"`, "renderer: NetworkManager"} {
		if !strings.Contains(string(nc), want) {
			t.Errorf("network-config lacks %q:\n%s", want, nc)
		}
	}
}

func TestCustomiseNotPi(t *testing.T) {
	disk := useTestDisk(t, 32)
	vs := writeVariants(t, t.TempDir(), fakeDiskImage(1<<20))
	info, _ := sniffImage(vs["raw"])
	c, _ := settings.prepare()
	warn, err := runWrite(writeSpec{Device: disk, Source: info, Custom: c}, func(progress) bool { return false })
	if err != nil || !strings.Contains(warn, "weren't applied") {
		t.Fatal(warn, err)
	}
}

func TestCustomiseChecks(t *testing.T) {
	for _, c := range []customise{
		{Hostname: "-bad"}, {User: "Root", Password: "x"}, {User: "root", Password: "x"}, {User: "pi"},
		{WifiSSID: "x", WifiPass: "short", WifiCountry: "GB"}, {WifiSSID: "x", WifiPass: "longenough"},
		{Timezone: "Europe/London; rm -rf /"}, {SSH: true, User: "pi", Password: "x"}, {User: "pi", Password: "x", SSH: true, SSHKeys: "not a key"},
	} {
		if _, err := c.prepare(); err == nil {
			t.Errorf("%+v accepted", c)
		}
	}
	if c, err := (&customise{}).prepare(); c != nil || err != nil {
		t.Fatal("empty settings should mean no customisation")
	}
}

// ------------------------------------------------------------ Windows ---

func winISO(t *testing.T, want map[string][]byte) string {
	t.Helper()
	r := rand.New(rand.NewSource(9))
	data := func(p string, n int) []byte {
		b := make([]byte, n)
		r.Read(b)
		want[p] = b
		return b
	}
	type N = udftest.Node
	root := &N{Dir: true, Kids: []*N{
		{Name: "bootmgr", Data: data("bootmgr", 400000)},
		{Name: "bootmgr.efi", Data: data("bootmgr.efi", 1500000)},
		{Name: "autorun.inf", Data: data("autorun.inf", 100), Kind: 2},
		{Name: "efi", Dir: true, Kids: []*N{{Name: "boot", Dir: true, Kids: []*N{{Name: "bootx64.efi", Data: data("efi/boot/bootx64.efi", 2100000)}}},
			{Name: "microsoft", Dir: true, Kids: []*N{{Name: "boot", Dir: true, Kids: []*N{{Name: "BCD", Data: data("efi/microsoft/boot/BCD", 16384)}}}}}}},
		{Name: "sources", Dir: true, Kids: []*N{
			{Name: "boot.wim", Data: data("sources/boot.wim", 3<<20+1234)},
			{Name: "install.wim", Data: data("sources/install.wim", 9<<20+77), Split: 4},
			{Name: "setup.exe", Data: data("sources/setup.exe", 300000), Kind: 1},
			{Name: "ei.cfg", Data: nil},
		}},
		{Name: "setup.exe", Data: data("setup.exe", 80000)},
	}}
	want["sources/ei.cfg"] = nil
	p := filepath.Join(t.TempDir(), "Win11_24H2_EnglishInternational_x64.iso")
	os.WriteFile(p, udftest.Build(root), 0o644)
	return p
}

func readPart(t *testing.T, disk []byte, i int) (int64, int64, byte) {
	e := disk[446+16*i:]
	return int64(binary.LittleEndian.Uint32(e[8:])) * 512, int64(binary.LittleEndian.Uint32(e[12:])) * 512, e[4]
}

func TestWindowsInstaller(t *testing.T) {
	for _, big := range []bool{false, true} {
		want := map[string][]byte{}
		iso := winISO(t, want)
		if big {
			old := fat32Limit
			fat32Limit = 8 << 20 // install.wim counts as "too big for FAT32"
			defer func() { fat32Limit = old }()
		}
		info, err := sniffImage(iso)
		if err != nil || info.Windows == nil || info.Windows.Arch != "x64" {
			t.Fatalf("sniff: %+v %v", info, err)
		}
		if layout := map[bool]string{false: "FAT32", true: "FAT32 + exFAT"}[big]; info.Windows.Layout != layout {
			t.Fatalf("layout %s", info.Windows.Layout)
		}
		diskP := sparseDisk(t, 800)
		var phases []string
		_, err = runWrite(writeSpec{Device: diskP, Source: info, Layout: "windows", Verify: true}, func(p progress) bool {
			if len(phases) == 0 || phases[len(phases)-1] != p.Phase {
				phases = append(phases, p.Phase)
			}
			return false
		})
		if err != nil {
			t.Fatal(err)
		}
		if strings.Join(phases, ",") != "prepare,write,verify,finish" {
			t.Fatalf("phases %v", phases)
		}
		f, _ := os.Open(diskP)
		head := make([]byte, 512)
		f.ReadAt(head, 0)
		s1, n1, ty1 := readPart(t, head, 0)
		if head[446] != 0x80 || ty1 != 0x0C || s1 != 1<<20 || head[510] != 0x55 {
			t.Fatalf("partition 1: %d %d %x", s1, n1, ty1)
		}
		fs, err := fat.Open(fileDev{f}, s1)
		if err != nil {
			t.Fatal(err)
		}
		for p, b := range want {
			got, err := fs.ReadFile(p)
			onFAT := !big || !strings.HasPrefix(p, "sources/") || p == "sources/boot.wim"
			if onFAT != (err == nil) {
				t.Fatalf("big=%v %s on FAT32: %v", big, p, err)
			}
			if err == nil && !bytes.Equal(got, b) {
				t.Fatalf("%s differs on FAT32", p)
			}
		}
		if big {
			s2, n2, ty2 := readPart(t, head, 1)
			if ty2 != 0x07 || s2 != s1+n1 || s2+n2 > 800<<20 {
				t.Fatalf("partition 2: %d %d %x", s2, n2, ty2)
			}
			for p, b := range want {
				off, n, err := exfat.Lookup(fileDev{f}, s2, p)
				if err != nil || n != int64(len(b)) {
					t.Fatalf("%s on exFAT: %v", p, err)
				}
				if n > 0 {
					got := make([]byte, n)
					f.ReadAt(got, off)
					if !bytes.Equal(got, b) {
						t.Fatalf("%s differs on exFAT", p)
					}
				}
			}
		} else if _, _, ty := readPart(t, head, 1); ty != 0 {
			t.Fatal("unexpected second partition")
		}
		f.Close()
	}
}

func TestWindowsTooSmallAndLinuxISO(t *testing.T) {
	iso := winISO(t, map[string][]byte{})
	info, _ := sniffImage(iso)
	disk := sparseDisk(t, 320)
	if _, err := runWrite(writeSpec{Device: disk, Source: info, Layout: "windows"}, func(progress) bool { return false }); err != nil {
		t.Fatal(err) // 320 MB is enough for this small one
	}
	old := fat32Limit
	fat32Limit = 8 << 20
	defer func() { fat32Limit = old }()
	info, _ = sniffImage(iso)
	if _, err := runWrite(writeSpec{Device: disk, Source: info, Layout: "windows"}, func(progress) bool { return false }); err == nil || !strings.Contains(err.Error(), "at least") {
		t.Fatalf("too small: %v", err)
	}
	// A CD-only ISO 9660 image gets a warning; a hybrid one doesn't.
	cd := make([]byte, 40*2048)
	copy(cd[16*2048:], "\x01CD001")
	p := filepath.Join(t.TempDir(), "old.iso")
	os.WriteFile(p, cd, 0o644)
	if i, _ := sniffImage(p); i.Note == "" || i.Windows != nil {
		t.Fatalf("CD-only ISO: %+v", i)
	}
	cd[510], cd[511] = 0x55, 0xAA
	os.WriteFile(p, cd, 0o644)
	if i, _ := sniffImage(p); i.Note != "" {
		t.Fatalf("hybrid ISO: %+v", i)
	}
}
