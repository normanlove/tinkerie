# Bundled firmware

These Cypress FX2 firmware images are loaded into the scope's RAM each time it is plugged
in. They are open source (LGPL-2.1-or-later, fx2lib based) from the Hantek6022API project
(commit cf6a103). The corresponding source code is in
`third_party/source/Hantek6022API-firmware-source-cf6a103.zip` and at
https://github.com/Ho-Ro/Hantek6022API/tree/main/Firmware

| File | Scope |
|---|---|
| dso6022be-firmware.hex | Hantek DSO-6022BE, Voltcraft DSO-2020 |
| dso6022bl-firmware.hex | Hantek DSO-6022BL |
| dso6021-firmware.hex | Hantek DSO-6021 |
| dds120-firmware.hex | Sainsmart / BUUDAI DDS120 |

Not included (users import these with "Import firmware…" in the Connect window):

- **isds205b-firmware.hex, mdso-firmware.hex** (Instrustar ISDS-205B, YiXingDianZi MDSO). No source has been published, so they can't be redistributed under the GPL. They are in the OpenHantek6022 downloads.
- **Hantek's own DSO-2090/2150/2250/5200/5200A firmware** is Hantek's copyright. It is extracted on the user's computer from Hantek's driver.
