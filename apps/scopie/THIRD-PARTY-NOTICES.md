# Third-party notices

Scopie is free software under the **GNU General Public License, version 3 or later**
(`LICENSE`). Copyright © 2026 nrmluvdev. It includes or builds on the components below.
The full licence texts are in the `licenses/` folder of the source.

Scopie is not affiliated with or endorsed by Hantek, Rigol, Siglent, Keysight,
Tektronix, OWON, Pico Technology, FNIRSI or any other manufacturer. Product names are
used only to say which hardware works with Scopie, and are trademarks of their owners.

## Built into Scopie.exe

| Component | Licence | Source |
|---|---|---|
| Hantek 6022BE/BL, 6021, DDS120 firmware (`firmware/*.hex`) | LGPL-2.1-or-later (fx2lib based; the Hantek6022API project as a whole is GPL-3.0) | `third_party/Hantek6022API-firmware-source-cf6a103.zip`, upstream https://github.com/Ho-Ro/Hantek6022API (commit cf6a103) |
| Zadig 2.9 (`drivers/zadig.exe`, unmodified; unpacked only when you click Install USB driver) | GPL-3.0 (Zadig), LGPL-3.0 (libwdi) | `third_party/libwdi-1.5.1-zadig-2.9-source.zip`, upstream https://github.com/pbatard/libwdi tag v1.5.1 |
| Signed WinUSB INF/CAT files for the Hantek 6022BE/BL (`drivers/inf/`) | Public domain (by fgrieu, signed by Spirtech) | https://github.com/OpenHantek/OpenHantek6022/tree/main/utils/windows_drivers/fgrieu |
| The Go standard library | BSD-3-Clause | https://go.dev |

Scopie uses no other libraries: USB (WinUSB), VISA, PicoSDK and the Windows APIs are
called directly through DLLs that ship with Windows or with the user's own drivers.

## Protocol information and adapted code

Scopie's drivers are original code. Some follow the protocols documented by these
projects, or port small parts of them:

| Project | Licence | Used for |
|---|---|---|
| OpenHantek6022 (https://github.com/OpenHantek/OpenHantek6022) | GPL-3.0 | Hantek 6022-family models, scaling, rate tables |
| Hantek6022API (https://github.com/Ho-Ro/Hantek6022API) | GPL-3.0 | 6022 USB protocol and calibration layout |
| OpenHantek (legacy) (https://github.com/OpenHantek/openhantek) | GPL-2.0-or-later / GPL-3.0 | DSO-2090/2150/2250/5200 protocol |
| libsigrok (https://sigrok.org) | GPL-3.0-or-later | DSO-2xxx protocol cross-check |
| sigrok-util `sigrok-fwextract-hantek-dso` | GPL-3.0-or-later | firmware extractor (ported) |
| owon-scope-web (https://github.com/fmesasc/owon-scope-web) | MIT, © 2026 Francisco Mesas Cervilla | OWON SCPI waveform decoding (adapted) |
| owon-hds200-capture (https://github.com/linux4life798/owon-hds200-capture) | see project | OWON HDS200 command reference |
| hanteker (https://github.com/hkoosha/hanteker), lucaoli/Hantek | GPL-3.0 | Hantek 2D42/2D72 protocol |
| picosdk-python-wrappers (https://github.com/picotech/picosdk-python-wrappers) | ISC, © 2019 Pico Technology Ltd | PicoSDK function signatures |
| VXI-11 (ONC-RPC) specification, USBTMC specification | public specifications | network and USB instrument links |

## Not shipped (each user obtains it themselves)

- **PicoSDK** (Pico Technology). Pico's licence doesn't allow redistribution, so Scopie links to Pico's download page.
- **Hantek DSO-2xxx/5xxx firmware.** It is Hantek's copyright. Users extract it on their own computer from Hantek's driver.
- **Instrustar ISDS-205B and MDSO firmware.** No source has been published, so it isn't bundled. Users import it from an OpenHantek6022 download.
- **NI-VISA / Keysight IO Libraries.** Proprietary and not needed; Scopie uses one if it is already installed.
