//go:build windows

package scopie

import "embed"

// Zadig (GPL-3.0, unmodified, with its licence) and the signed WinUSB driver
// packages for the Hantek 6022BE/BL (public domain, by fgrieu).
//
//go:embed drivers/zadig.exe drivers/zadig.ini drivers/ZADIG-LICENSE.txt drivers/inf
var driverFS embed.FS
