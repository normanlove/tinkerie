# Meterie changes

## 0.6.0-alpha

New meters (every protocol checked against libsigrok's decoder source or the community
reference, with unit tests):

* UNI-T **UT71A/B/C/D/E** and UT804, Voltcraft VC-920/940/960, Tenma 72-7730/72-7732/72-9380A
  (UT-D04 USB or RS-232)
* UNI-T **UT181A** (CP2110 USB). Auto-detect now tells a UT181A from a UT61E+ on the same cable.
* **Brymen BM869s / BM867s / BM859s** with the BU-86X USB cable (main display)
* **Brymen BM25x** (BM257s …) with Brymen's IR-to-USB cable
* UNI-T UT60G, Tenma 72-7750, ISO-TECH IDM103N (Cyrustek ES519xx 11-byte)
* **ANENG / ZOYI / BSIDE / BABATools Bluetooth meters** ("Bluetooth DMM": AN9002, ZT-300AB,
  ZT-5B, ZT-5BQ, ZT-5566S …), checked against the published test vectors
* Victor 70C/86C/86D USB is now de-scrambled exactly like sigrok and marked verified

Improvements:

* **Exact model** menu on each connected meter's card. Auto-detect finds the protocol family
  (e.g. "an FS9721 meter"), and you pick your model (e.g. HoldPeak HP-90EPC) so model-specific
  symbols such as the temperature flag decode correctly. Meterie remembers the choice.
* Fixed: typing in a text field and pressing Enter in the same instant (or the marker dialog's
  pre-filled label) could append to the old text instead of replacing it.

## First alpha

First release.
