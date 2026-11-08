package flashie

// OS customisation for the "Write image" tab, like Raspberry Pi Imager's:
// hostname, user and password, Wi-Fi, SSH, time zone and keyboard. After the
// image is written and checked, Flashie edits the FAT boot partition on the
// drive directly (no mounting), in one of the two ways Raspberry Pi OS
// understands:
//
//   - cloud-init (Raspberry Pi OS "Trixie" and later, Ubuntu): user-data and
//     network-config files on the boot partition.
//   - firstrun.sh (Raspberry Pi OS "Bookworm" and older): a script run once
//     at first boot through systemd.run= in cmdline.txt, as Imager 1.x did.
//
// Passwords are never written in plain text: the login password becomes a
// SHA-512 crypt hash, and for firstrun.sh the Wi-Fi password becomes its
// WPA key. Made by nrmluvdev.

import (
	"bytes"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"tinkerie/shared/fat"
)

type customise struct {
	Hostname    string `json:"hostname,omitempty"`
	User        string `json:"user,omitempty"`
	Password    string `json:"password,omitempty"` // from the page; replaced by PassHash before the job runs
	PassHash    string `json:"passHash,omitempty"`
	WifiSSID    string `json:"wifiSSID,omitempty"`
	WifiPass    string `json:"wifiPass,omitempty"`
	WifiHidden  bool   `json:"wifiHidden,omitempty"`
	WifiCountry string `json:"wifiCountry,omitempty"`
	Timezone    string `json:"timezone,omitempty"`
	Keymap      string `json:"keymap,omitempty"`
	SSH         bool   `json:"ssh,omitempty"`
	SSHPassword bool   `json:"sshPassword,omitempty"` // allow password logins (else keys only)
	SSHKeys     string `json:"sshKeys,omitempty"`
	Format      string `json:"format,omitempty"` // catalogue init_format hint
}

var (
	reHost    = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?$`)
	reUser    = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,31}$`)
	reCountry = regexp.MustCompile(`^[A-Z]{2}$`)
	reTZ      = regexp.MustCompile(`^[A-Za-z0-9_+/-]{1,64}$`)
	reKeymap  = regexp.MustCompile(`^[a-z0-9_()-]{1,32}$`)
	reHex64   = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)
)

// prepare checks the settings and hashes the password. It returns nil if
// nothing is to be changed.
func (c *customise) prepare() (*customise, error) {
	if c == nil {
		return nil, nil
	}
	x := *c
	x.Hostname = strings.TrimSpace(x.Hostname)
	x.User = strings.TrimSpace(x.User)
	x.WifiCountry = strings.ToUpper(strings.TrimSpace(x.WifiCountry))
	x.Timezone, x.Keymap = strings.TrimSpace(x.Timezone), strings.TrimSpace(x.Keymap)
	if x.Hostname != "" && !reHost.MatchString(x.Hostname) {
		return nil, errors.New("the hostname can only use letters, numbers and dashes (up to 63), and can't start or end with a dash")
	}
	if x.User != "" {
		if !reUser.MatchString(x.User) || x.User == "root" {
			return nil, errors.New("the username must start with a small letter and use only small letters, numbers, - and _ (and it can't be root)")
		}
		if x.Password == "" && x.PassHash == "" {
			return nil, errors.New("choose a password for the user")
		}
	}
	if x.Password != "" {
		if x.User == "" {
			return nil, errors.New("choose a username for the password")
		}
		x.PassHash = sha512Crypt(x.Password, randomSalt())
		x.Password = ""
	}
	if x.WifiSSID != "" {
		if len(x.WifiSSID) > 32 {
			return nil, errors.New("the Wi-Fi name (SSID) is at most 32 characters")
		}
		if n := len(x.WifiPass); n != 0 && (n < 8 || n > 63) && !reHex64.MatchString(x.WifiPass) {
			return nil, errors.New("a Wi-Fi password is 8 to 63 characters")
		}
		if !reCountry.MatchString(x.WifiCountry) {
			return nil, errors.New("choose the Wi-Fi country (the law sets which channels may be used)")
		}
	} else {
		x.WifiPass, x.WifiHidden = "", false
	}
	if x.WifiCountry != "" && !reCountry.MatchString(x.WifiCountry) {
		return nil, errors.New("the Wi-Fi country is a two-letter code such as GB")
	}
	if x.Timezone != "" && !reTZ.MatchString(x.Timezone) {
		return nil, errors.New("that time zone doesn't look right (it's like Europe/London)")
	}
	if x.Keymap != "" && !reKeymap.MatchString(x.Keymap) {
		return nil, errors.New("that keyboard layout doesn't look right (it's like gb or us)")
	}
	var keys []string
	for _, k := range strings.Split(x.SSHKeys, "\n") {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		if !strings.HasPrefix(k, "ssh-") && !strings.HasPrefix(k, "ecdsa-") && !strings.HasPrefix(k, "sk-") {
			return nil, errors.New("an SSH public key starts with ssh-ed25519, ssh-rsa or ecdsa-… (paste the .pub file)")
		}
		if strings.ContainsAny(k, "'\"\\`$") {
			return nil, errors.New("an SSH key can't contain quotes, $, ` or \\")
		}
		keys = append(keys, k)
	}
	x.SSHKeys = strings.Join(keys, "\n")
	if x.SSH && !x.SSHPassword && x.SSHKeys == "" {
		return nil, errors.New("for key-only SSH, paste at least one public key")
	}
	if x.SSH && x.User == "" {
		return nil, errors.New("SSH needs a username and password set too")
	}
	if x.Hostname == "" && x.User == "" && x.WifiSSID == "" && x.Timezone == "" && x.Keymap == "" && !x.SSH {
		return nil, nil
	}
	return &x, nil
}

// ------------------------------------------------------------- hashing ---

const cryptB64 = "./0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func randomSalt() string {
	b := make([]byte, 16)
	rand.Read(b)
	s := make([]byte, 16)
	for i := range s {
		s[i] = cryptB64[int(b[i])%64]
	}
	return string(s)
}

// sha512Crypt is glibc's "$6$" password hash (Ulrich Drepper's SHA-crypt,
// 5000 rounds), which Linux login, chpasswd -e and cloud-init accept.
func sha512Crypt(pw, salt string) string {
	if len(salt) > 16 {
		salt = salt[:16]
	}
	p, s := []byte(pw), []byte(salt)
	h := sha512.New()
	h.Write(p)
	h.Write(s)
	h.Write(p)
	b := h.Sum(nil)
	h.Reset()
	h.Write(p)
	h.Write(s)
	for n := len(p); n > 0; n -= 64 {
		h.Write(b[:min(n, 64)])
	}
	for n := len(p); n > 0; n >>= 1 {
		if n&1 != 0 {
			h.Write(b)
		} else {
			h.Write(p)
		}
	}
	a := h.Sum(nil)
	h.Reset()
	for range p {
		h.Write(p)
	}
	dp := h.Sum(nil)
	pp := make([]byte, 0, len(p))
	for len(pp) < len(p) {
		pp = append(pp, dp[:min(64, len(p)-len(pp))]...)
	}
	h.Reset()
	for i := 0; i < 16+int(a[0]); i++ {
		h.Write(s)
	}
	ds := h.Sum(nil)
	ss := ds[:len(s)]
	c := a
	for i := 0; i < 5000; i++ {
		h.Reset()
		if i&1 != 0 {
			h.Write(pp)
		} else {
			h.Write(c)
		}
		if i%3 != 0 {
			h.Write(ss)
		}
		if i%7 != 0 {
			h.Write(pp)
		}
		if i&1 != 0 {
			h.Write(c)
		} else {
			h.Write(pp)
		}
		c = h.Sum(nil)
	}
	var out strings.Builder
	out.WriteString("$6$" + salt + "$")
	put := func(b2, b1, b0 byte, n int) {
		w := uint(b2)<<16 | uint(b1)<<8 | uint(b0)
		for ; n > 0; n-- {
			out.WriteByte(cryptB64[w&63])
			w >>= 6
		}
	}
	for _, t := range sha512Order {
		put(c[t[0]], c[t[1]], c[t[2]], 4)
	}
	put(0, 0, c[63], 2)
	return out.String()
}

// sha512Order is the byte order of the encoded hash, from the SHA-crypt spec.
var sha512Order = [21][3]int{
	{0, 21, 42}, {22, 43, 1}, {44, 2, 23}, {3, 24, 45}, {25, 46, 4}, {47, 5, 26}, {6, 27, 48},
	{28, 49, 7}, {50, 8, 29}, {9, 30, 51}, {31, 52, 10}, {53, 11, 32}, {12, 33, 54}, {34, 55, 13},
	{56, 14, 35}, {15, 36, 57}, {37, 58, 16}, {59, 17, 38}, {18, 39, 60}, {40, 61, 19}, {62, 20, 41},
}

// wifiPSK is the WPA key wpa_supplicant derives from a password (PBKDF2,
// SHA-1, 4096 rounds, salted with the network name).
func wifiPSK(pass, ssid string) string {
	if reHex64.MatchString(pass) {
		return strings.ToLower(pass)
	}
	k, _ := pbkdf2.Key(sha1.New, pass, []byte(ssid), 4096, 32)
	return hex.EncodeToString(k)
}

// ------------------------------------------------------------ partitions ---

// bootPartition finds the first FAT partition in the image's partition
// table (MBR, or GPT behind a protective MBR) and returns its byte offset.
func bootPartition(dev fat.Device, mbr []byte) (int64, error) {
	if len(mbr) < 512 || mbr[510] != 0x55 || mbr[511] != 0xAA {
		return 0, errors.New("the image has no partition table")
	}
	for i := 0; i < 4; i++ {
		e := mbr[446+16*i:]
		start := int64(binary.LittleEndian.Uint32(e[8:]))
		switch e[4] {
		case 0x01, 0x04, 0x06, 0x0B, 0x0C, 0x0E, 0xEF:
			if start > 0 {
				return start * 512, nil
			}
		case 0xEE:
			return gptFAT(dev)
		}
	}
	return 0, errors.New("the image has no FAT boot partition")
}

func gptFAT(dev fat.Device) (int64, error) {
	h := make([]byte, 512)
	if _, err := dev.ReadAt(h, 512); err != nil || string(h[:8]) != "EFI PART" {
		return 0, errors.New("the image's GPT partition table can't be read")
	}
	lba := int64(binary.LittleEndian.Uint64(h[72:]))
	n := int(binary.LittleEndian.Uint32(h[80:]))
	sz := int(binary.LittleEndian.Uint32(h[84:]))
	if sz < 128 || n > 256 {
		return 0, errors.New("the image's GPT partition table can't be read")
	}
	tab := make([]byte, n*sz)
	if _, err := dev.ReadAt(tab, lba*512); err != nil {
		return 0, err
	}
	basic, _ := hex.DecodeString("a2a0d0ebe5b9334487c068b6b72699c7") // Microsoft basic data
	esp, _ := hex.DecodeString("28732ac11ff8d211ba4b00a0c93ec93b")   // EFI system
	for i := 0; i < n; i++ {
		e := tab[i*sz:]
		if bytes.Equal(e[:16], basic) || bytes.Equal(e[:16], esp) {
			return int64(binary.LittleEndian.Uint64(e[32:])) * 512, nil
		}
	}
	return 0, errors.New("the image has no FAT boot partition")
}

// ------------------------------------------------------------- applying ---

var errNotPiImage = errors.New("this image has no Raspberry Pi boot partition, so the OS settings weren't applied")

// applyCustom edits the boot partition. It returns which method it used.
func applyCustom(dev fat.Device, mbr []byte, c *customise) (string, error) {
	off, err := bootPartition(dev, mbr)
	if err != nil {
		return "", errNotPiImage
	}
	fs, err := fat.Open(dev, off)
	if err != nil {
		return "", errNotPiImage
	}
	hasCmd := fs.Exists("cmdline.txt")
	cloud := fs.Exists("user-data") || fs.Exists("meta-data") || strings.HasPrefix(c.Format, "cloudinit")
	switch {
	case cloud:
		if err := writeCloudInit(fs, c); err != nil {
			return "", err
		}
		return "cloud-init", fs.Flush()
	case hasCmd:
		if err := writeFirstrun(fs, c); err != nil {
			return "", err
		}
		return "firstrun.sh", fs.Flush()
	}
	return "", errNotPiImage
}

// yq quotes a YAML string (JSON strings are valid YAML double-quoted ones).
func yq(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func cloudUserData(c *customise) string {
	var b strings.Builder
	b.WriteString("#cloud-config\n# Written by Flashie (Tinkerie, by nrmluvdev) for first boot.\n")
	if c.Hostname != "" {
		fmt.Fprintf(&b, "hostname: %s\nmanage_etc_hosts: true\n", c.Hostname)
	}
	if c.Timezone != "" {
		fmt.Fprintf(&b, "timezone: %s\n", c.Timezone)
	}
	if c.Keymap != "" {
		fmt.Fprintf(&b, "keyboard:\n  model: pc105\n  layout: %s\n", yq(c.Keymap))
	}
	if c.User != "" {
		fmt.Fprintf(&b, "users:\n  - name: %s\n", c.User)
		b.WriteString("    groups: users,adm,dialout,audio,netdev,video,plugdev,cdrom,games,input,gpio,spi,i2c,render,sudo\n")
		b.WriteString("    shell: /bin/bash\n    lock_passwd: false\n")
		fmt.Fprintf(&b, "    passwd: %s\n", yq(c.PassHash))
		b.WriteString("    sudo: ALL=(ALL) NOPASSWD:ALL\n")
		if c.SSH && c.SSHKeys != "" {
			b.WriteString("    ssh_authorized_keys:\n")
			for _, k := range strings.Split(c.SSHKeys, "\n") {
				fmt.Fprintf(&b, "      - %s\n", yq(k))
			}
		}
	}
	if c.SSH {
		b.WriteString("enable_ssh: true\n")
		fmt.Fprintf(&b, "ssh_pwauth: %v\n", c.SSHPassword)
	}
	return b.String()
}

func cloudNetwork(c *customise) string {
	var b strings.Builder
	b.WriteString("# Written by Flashie (Tinkerie, by nrmluvdev).\nnetwork:\n  version: 2\n  wifis:\n    renderer: NetworkManager\n    wlan0:\n      dhcp4: true\n")
	fmt.Fprintf(&b, "      regulatory-domain: %s\n", yq(c.WifiCountry))
	fmt.Fprintf(&b, "      access-points:\n        %s:\n", yq(c.WifiSSID))
	if c.WifiPass != "" {
		fmt.Fprintf(&b, "          password: %s\n", yq(c.WifiPass))
	} else {
		b.WriteString("          {}\n")
	}
	if c.WifiHidden {
		b.WriteString("          hidden: true\n")
	}
	b.WriteString("      optional: true\n")
	return b.String()
}

func writeCloudInit(fs *fat.FS, c *customise) error {
	if err := fs.WriteFile("user-data", []byte(cloudUserData(c))); err != nil {
		return err
	}
	if c.WifiSSID != "" {
		if err := fs.WriteFile("network-config", []byte(cloudNetwork(c))); err != nil {
			return err
		}
	}
	if !fs.Exists("meta-data") {
		id := make([]byte, 6)
		rand.Read(id)
		if err := fs.WriteFile("meta-data", []byte("instance-id: flashie-"+hex.EncodeToString(id)+"\n")); err != nil {
			return err
		}
	}
	return nil
}

// sq quotes a string for the shell.
func sq(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

func firstrunScript(c *customise) string {
	var b strings.Builder
	b.WriteString("#!/bin/bash\n# Written by Flashie (Tinkerie, by nrmluvdev). Runs once at first boot, then removes itself.\n\nset +e\n\n")
	b.WriteString("BOOT=/boot/firmware\n[ -f $BOOT/cmdline.txt ] || BOOT=/boot\nIC=/usr/lib/raspberrypi-sys-mods/imager_custom\n\n")
	if c.Hostname != "" {
		h := sq(c.Hostname)
		b.WriteString("CURRENT_HOSTNAME=`cat /etc/hostname | tr -d \" \\t\\n\\r\"`\n")
		fmt.Fprintf(&b, "if [ -f $IC ]; then\n   $IC set_hostname %s\nelse\n   echo %s >/etc/hostname\n   sed -i \"s/127.0.1.1.*$CURRENT_HOSTNAME/127.0.1.1\\t\"%s\"/g\" /etc/hosts\nfi\n", h, h, h)
	}
	if c.User != "" {
		b.WriteString("FIRSTUSER=`getent passwd 1000 | cut -d: -f1`\nFIRSTUSERHOME=`getent passwd 1000 | cut -d: -f6`\n")
		if c.SSH && c.SSHKeys != "" {
			b.WriteString("install -o \"$FIRSTUSER\" -m 700 -d \"$FIRSTUSERHOME/.ssh\"\n")
			b.WriteString("install -o \"$FIRSTUSER\" -m 600 <(printf '%s\\n' " + sq(c.SSHKeys) + ") \"$FIRSTUSERHOME/.ssh/authorized_keys\"\n")
		}
	}
	if c.SSH {
		if !c.SSHPassword {
			b.WriteString("echo 'PasswordAuthentication no' >>/etc/ssh/sshd_config\n")
		}
		b.WriteString("if [ -f $IC ]; then\n   $IC enable_ssh\nelse\n   systemctl enable ssh\nfi\n")
	}
	if c.User != "" {
		u, p := sq(c.User), sq(c.PassHash)
		fmt.Fprintf(&b, `if [ -f /usr/lib/userconf-pi/userconf ]; then
   /usr/lib/userconf-pi/userconf %[1]s %[2]s
else
   echo "$FIRSTUSER:"%[2]s | chpasswd -e
   if [ "$FIRSTUSER" != %[1]s ]; then
      usermod -l %[1]s "$FIRSTUSER"
      usermod -m -d "/home/"%[1]s %[1]s
      groupmod -n %[1]s "$FIRSTUSER"
      if grep -q "^autologin-user=" /etc/lightdm/lightdm.conf ; then
         sed /etc/lightdm/lightdm.conf -i -e "s/^autologin-user=.*/autologin-user="%[1]s"/"
      fi
      if [ -f /etc/systemd/system/getty@tty1.service.d/autologin.conf ]; then
         sed /etc/systemd/system/getty@tty1.service.d/autologin.conf -i -e "s/$FIRSTUSER/"%[1]s"/"
      fi
      if [ -f /etc/sudoers.d/010_pi-nopasswd ]; then
         sed -i "s/^$FIRSTUSER /"%[1]s" /" /etc/sudoers.d/010_pi-nopasswd
      fi
   fi
fi
`, u, p)
	}
	if c.WifiSSID != "" {
		ssid, cc := sq(c.WifiSSID), sq(c.WifiCountry)
		psk := ""
		if c.WifiPass != "" {
			psk = wifiPSK(c.WifiPass, c.WifiSSID)
		}
		hidden := ""
		if c.WifiHidden {
			hidden = "-h "
		}
		key := "key_mgmt=NONE"
		if psk != "" {
			key = "psk=" + psk
		}
		scan := ""
		if c.WifiHidden {
			scan = "\tscan_ssid=1\n"
		}
		fmt.Fprintf(&b, "if [ -f $IC ]; then\n   $IC set_wlan %s%s %s %s\nelse\ncat >/etc/wpa_supplicant/wpa_supplicant.conf <<'WPAEOF'\ncountry=%s\nctrl_interface=DIR=/var/run/wpa_supplicant GROUP=netdev\nap_scan=1\n\nupdate_config=1\nnetwork={\n%s\tssid=%s\n\t%s\n}\n\nWPAEOF\n   chmod 600 /etc/wpa_supplicant/wpa_supplicant.conf\n   rfkill unblock wifi\n   for filename in /var/lib/systemd/rfkill/*:wlan ; do\n       echo 0 > $filename\n   done\nfi\n",
			hidden, ssid, sq(psk), cc, c.WifiCountry, scan, wpaQuote(c.WifiSSID), key)
	}
	if c.Keymap != "" || c.Timezone != "" {
		b.WriteString("if [ -f $IC ]; then\n")
		if c.Keymap != "" {
			fmt.Fprintf(&b, "   $IC set_keymap %s\n", sq(c.Keymap))
		}
		if c.Timezone != "" {
			fmt.Fprintf(&b, "   $IC set_timezone %s\n", sq(c.Timezone))
		}
		b.WriteString("else\n")
		if c.Timezone != "" {
			fmt.Fprintf(&b, "   rm -f /etc/localtime\n   echo %s >/etc/timezone\n   dpkg-reconfigure -f noninteractive tzdata\n", sq(c.Timezone))
		}
		if c.Keymap != "" {
			fmt.Fprintf(&b, "cat >/etc/default/keyboard <<'KBEOF'\nXKBMODEL=\"pc105\"\nXKBLAYOUT=\"%s\"\nXKBVARIANT=\"\"\nXKBOPTIONS=\"\"\n\nKBEOF\n   dpkg-reconfigure -f noninteractive keyboard-configuration\n", c.Keymap)
		}
		b.WriteString("fi\n")
	}
	b.WriteString("rm -f $BOOT/firstrun.sh\nsed -i 's| systemd.run.*||g' $BOOT/cmdline.txt\nexit 0\n")
	return b.String()
}

// wpaQuote writes an SSID for wpa_supplicant.conf: quoted if plain, else hex.
func wpaQuote(s string) string {
	for _, r := range s {
		if r < 32 || r > 126 || r == '"' || r == '\\' {
			return hex.EncodeToString([]byte(s))
		}
	}
	return `"` + s + `"`
}

// bootDir works out where Raspberry Pi OS mounts the boot partition:
// /boot/firmware from Bookworm (October 2023) on, /boot before.
func bootDir(fs *fat.FS) string {
	b, err := fs.ReadFile("issue.txt")
	if err != nil {
		return "/boot/firmware"
	}
	m := regexp.MustCompile(`(\d{4}-\d{2}-\d{2})`).FindString(string(b))
	if t, err := time.Parse("2006-01-02", m); err == nil && t.Before(time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC)) {
		return "/boot"
	}
	return "/boot/firmware"
}

func writeFirstrun(fs *fat.FS, c *customise) error {
	cmd, err := fs.ReadFile("cmdline.txt")
	if err != nil {
		return err
	}
	line := strings.TrimSpace(strings.SplitN(string(cmd), "\n", 2)[0])
	if i := strings.Index(line, " systemd.run"); i >= 0 {
		line = line[:i] // a previous customisation
	}
	if err := fs.WriteFile("firstrun.sh", []byte(firstrunScript(c))); err != nil {
		return err
	}
	line += " systemd.run=" + bootDir(fs) + "/firstrun.sh systemd.run_success_action=reboot systemd.unit=kernel-command-line.target\n"
	return fs.WriteFile("cmdline.txt", []byte(line))
}
