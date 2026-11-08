package meterie

// meters.go: the catalogue of supported meters and how to talk to them.

import (
	"strings"
)

// Transport kinds.
type Transport int

const (
	TrSerial Transport = iota // COM port (USB-serial cable: CH340, CP210x, PL2303, FTDI, or real RS-232)
	TrCH9325                  // UNI-T UT-D04 style HID cable (WCH CH9325 / Hoitek HE2325U)
	TrCP2110                  // Silicon Labs CP2110 HID-UART (UT61E+, UT161 ...)
	TrVictor                  // Victor 70C/86C HID cable (obfuscated FS9922)
	TrBLE                     // Bluetooth Low Energy notifications (WinRT)
	TrDemo
	TrManual
	TrCamera
	TrBU86X // Brymen BU-86X USB cable (HID 0820:0001) for BM869s/BM867s/BM859s
)

func (t Transport) String() string {
	return [...]string{"USB-serial / COM", "UT-D04 USB (HID)", "CP2110 USB (HID)", "Victor USB (HID)", "Bluetooth LE", "Demo", "Manual", "Camera", "Brymen BU-86X USB (HID)"}[t]
}

// SerialCfg is a UART line setting.
type SerialCfg struct {
	Baud     int
	DataBits int
	Parity   byte // 'N', 'O', 'E'
	StopBits int
	DTR, RTS bool
}

func (c SerialCfg) String() string {
	p := c.Parity
	if p == 0 {
		p = 'N'
	}
	return itoa(c.Baud) + " " + itoa(c.DataBits) + string(rune(p)) + itoa(c.StopBits)
}

func ser(baud, bits int, par byte, stop int) SerialCfg {
	return SerialCfg{Baud: baud, DataBits: bits, Parity: par, StopBits: stop, DTR: true, RTS: false}
}

// Support levels, shown honestly in the meter picker.
const (
	SupVerified     = "Protocol verified"
	SupCommunity    = "Community-documented protocol"
	SupExperimental = "Experimental"
)

// MeterModel describes one supported meter (or family).
type MeterModel struct {
	ID         string
	Brand      string
	Model      string
	Also       string // rebrands / compatible models
	Proto      string
	Transports []Transport
	Serial     SerialCfg
	TempBit    int // FS9721 °C flag bit (-1 none)
	Counts     int
	BLENames   []string // advertised name prefixes
	BLEService string   // 16-bit or 128-bit UUID string
	BLEChar    string
	Support    string
	Connect    string // beginner instructions for the PC link
	Featured   bool
}

func (m *MeterModel) Name() string { return m.Brand + " " + m.Model }

func (m *MeterModel) Has(t Transport) bool {
	for _, x := range m.Transports {
		if x == t {
			return true
		}
	}
	return false
}

const pcLinkFS = "Plug in the meter's USB cable (Windows may install a CH340/CP210x/PL2303 driver), switch the meter on, then press the meter's PC-link button (often labelled USB, RS232 or a long press on REL/HOLD) until the RS232/USB symbol appears on the LCD."

var meterCatalog = []*MeterModel{
	{ID: "holdpeak-hp90epc", Brand: "HoldPeak", Model: "HP-90EPC", Also: "HP-90EPC USB, HoldPeak 90EPC", Proto: "fs9721",
		Transports: []Transport{TrSerial}, Serial: ser(2400, 8, 'N', 1), TempBit: 2, Counts: 4000, Support: SupVerified, Featured: true,
		Connect: "Connect the supplied USB cable (install HoldPeak's USB driver if Windows doesn't create a COM port), turn the meter on and press its USB/PC button until the USB/RS232 icon shows on the LCD. Meterie reads it at 2400 baud (Fortune FS9721 protocol)."},
	{ID: "tekpower-tp4000zc", Brand: "TekPower", Model: "TP4000ZC", Also: "Digitek DT4000ZC, Tenma 72-7745", Proto: "fs9721",
		Transports: []Transport{TrSerial}, Serial: ser(2400, 8, 'N', 1), TempBit: -1, Counts: 4000, Support: SupVerified, Connect: pcLinkFS},
	{ID: "uni-t-ut60e", Brand: "UNI-T", Model: "UT60A / UT60E", Also: "UT60 series with RS232/USB", Proto: "fs9721",
		Transports: []Transport{TrSerial, TrCH9325}, Serial: ser(2400, 8, 'N', 1), TempBit: -1, Counts: 4000, Support: SupVerified, Connect: pcLinkFS},
	{ID: "va-va18b", Brand: "V&A", Model: "VA18B", Also: "Mastech VA18B", Proto: "fs9721",
		Transports: []Transport{TrSerial}, Serial: ser(2400, 8, 'N', 1), TempBit: -1, Counts: 6000, Support: SupVerified, Connect: pcLinkFS},
	{ID: "mastech-ms8250b", Brand: "Mastech", Model: "MS8250B", Proto: "fs9721",
		Transports: []Transport{TrSerial}, Serial: ser(2400, 8, 'N', 1), TempBit: -1, Counts: 4000, Support: SupVerified, Connect: pcLinkFS},
	{ID: "peaktech-3330", Brand: "PeakTech", Model: "3330 / 3415", Proto: "fs9721",
		Transports: []Transport{TrSerial}, Serial: ser(2400, 8, 'N', 1), TempBit: -1, Counts: 4000, Support: SupVerified, Connect: pcLinkFS},
	{ID: "voltcraft-vc820", Brand: "Voltcraft", Model: "VC-820 / VC-840", Proto: "fs9721",
		Transports: []Transport{TrSerial}, Serial: ser(2400, 8, 'N', 1), TempBit: 0, Counts: 4000, Support: SupVerified, Connect: pcLinkFS},
	{ID: "generic-fs9721", Brand: "Generic", Model: "FS9721 / FS9711 4000-count", Also: "most cheap 4000-count meters with a USB/RS232 cable", Proto: "fs9721",
		Transports: []Transport{TrSerial, TrCH9325, TrBLE}, Serial: ser(2400, 8, 'N', 1), TempBit: -2, Counts: 4000, Support: SupVerified, Connect: pcLinkFS},

	{ID: "uni-t-ut61bcd", Brand: "UNI-T", Model: "UT61B / UT61C / UT61D", Proto: "fs9922",
		Transports: []Transport{TrCH9325, TrSerial}, Serial: ser(2400, 8, 'N', 1), Counts: 6000, Support: SupVerified,
		Connect: "Use the UT-D04 USB cable (shows up as a USB HID device, no driver needed) or the UT-D02 RS-232 cable. Turn the meter on and press SELECT/RS232 (long press) until 'RS232' shows."},
	{ID: "voltcraft-vc830", Brand: "Voltcraft", Model: "VC-830", Proto: "fs9922",
		Transports: []Transport{TrSerial}, Serial: ser(2400, 8, 'N', 1), Counts: 6000, Support: SupVerified, Connect: pcLinkFS},
	{ID: "generic-fs9922", Brand: "Generic", Model: "FS9922 6000-count", Proto: "fs9922",
		Transports: []Transport{TrSerial, TrCH9325, TrBLE}, Serial: ser(2400, 8, 'N', 1), Counts: 6000, Support: SupVerified, Connect: pcLinkFS},

	{ID: "uni-t-ut61e", Brand: "UNI-T", Model: "UT61E", Also: "UT61E (original, not the E+)", Proto: "es51922",
		Transports: []Transport{TrCH9325, TrSerial}, Serial: SerialCfg{Baud: 19200, DataBits: 7, Parity: 'O', StopBits: 1, DTR: true}, Counts: 22000, Support: SupVerified,
		Connect: "Use the UT-D04 USB cable (HID, no driver) or the RS-232 cable. Press the meter's RS232 button so 'RS232' shows on the LCD."},

	{ID: "uni-t-ut61eplus", Brand: "UNI-T", Model: "UT61E+", Also: "UT61B+, UT61D+, UT161B/D/E", Proto: "ut61eplus",
		Transports: []Transport{TrCP2110}, Serial: ser(9600, 8, 'N', 1), Counts: 22000, Support: SupVerified, Featured: true,
		Connect: "Plug in the supplied USB cable (it is a CP2110 HID bridge — no driver needed). Meterie asks the meter for readings itself; just turn it on."},

	{ID: "owon-ble", Brand: "OWON", Model: "B35T+ / B41T+ / OW18E / OW16", Also: "OWON Bluetooth meters (6-byte protocol), original B35T", Proto: "owon",
		Transports: []Transport{TrBLE}, Counts: 6000, BLENames: []string{"BDM", "B35", "B41", "OW18", "OW16", "B35T", "B41T", "CM2100"},
		BLEService: "fff0", BLEChar: "fff4", Support: SupCommunity, Featured: true,
		Connect: "Turn on Bluetooth on the meter (long-press the Bluetooth/Hz button until the BT icon shows). Don't pair it in Windows settings — Meterie connects directly."},

	{ID: "fluke-287", Brand: "Fluke", Model: "287 / 289", Proto: "fluke28x",
		Transports: []Transport{TrSerial}, Serial: ser(115200, 8, 'N', 1), Counts: 50000, Support: SupVerified,
		Connect: "Clip the Fluke IR189USB (or compatible FTDI IR) cable onto the meter's IR port. It appears as a COM port at 115200 baud."},
	{ID: "fluke-189", Brand: "Fluke", Model: "187 / 189", Proto: "fluke18x",
		Transports: []Transport{TrSerial}, Serial: ser(9600, 8, 'N', 1), Counts: 50000, Support: SupVerified,
		Connect: "Use the Fluke IR serial/USB cable on the IR port (9600 baud)."},
	{ID: "fluke-89iv", Brand: "Fluke", Model: "87-IV / 89-IV", Proto: "fluke18x",
		Transports: []Transport{TrSerial}, Serial: SerialCfg{Baud: 9600, DataBits: 8, Parity: 'N', StopBits: 1, DTR: false, RTS: true}, Counts: 20000, Support: SupVerified,
		Connect: "Use the Fluke IR cable; it is powered from the RTS line, which Meterie switches on."},

	{ID: "mastech-mas345", Brand: "Mastech", Model: "MAS-345 / M-3850", Also: "Metex 14-byte ASCII meters, Voltcraft ME-xx / M-36xx", Proto: "metex14",
		Transports: []Transport{TrSerial}, Serial: SerialCfg{Baud: 600, DataBits: 7, Parity: 'N', StopBits: 2, DTR: true, RTS: false}, Counts: 4000, Support: SupVerified,
		Connect: "RS-232 cable (with a USB-serial adapter). These meters answer when asked; the baud rate is often 600 or 1200 — change it below if nothing arrives."},

	{ID: "owon-xdm", Brand: "OWON", Model: "XDM1041 / XDM1241 / XDM2041", Also: "SCPI bench meters over USB-serial", Proto: "scpi",
		Transports: []Transport{TrSerial}, Serial: ser(115200, 8, 'N', 1), Counts: 60000, Support: SupExperimental,
		Connect: "Connect the USB cable (CH340 virtual COM port). Meterie sends SCPI 'FUNC?' and 'MEAS?' queries."},

	{ID: "victor-86c", Brand: "Victor", Model: "70C / 86C / 86D", Also: "Pro'sKit MT-1820 (Victor cable)", Proto: "victor",
		Transports: []Transport{TrVictor}, Counts: 6000, Support: SupVerified,
		Connect: "Plug in the Victor USB cable (HID, no driver). Meterie removes the cable's scrambling exactly like sigrok does and only accepts frames that check out."},

	{ID: "uni-t-ut71", Brand: "UNI-T", Model: "UT71A / B / C / D / E", Also: "UNI-T UT804, Voltcraft VC-920/940/960, Tenma 72-7730/72-7732/72-9380A", Proto: "ut71x",
		Transports: []Transport{TrCH9325, TrSerial}, Serial: SerialCfg{Baud: 2400, DataBits: 7, Parity: 'O', StopBits: 1, DTR: true}, Counts: 40000, Support: SupVerified,
		Connect: "Use the UT-D04 USB cable (HID, no driver) or the UT-D02 RS-232 cable on the meter's IR port, then press the meter's PC/SEND button so the link symbol shows."},
	{ID: "uni-t-ut60g", Brand: "UNI-T", Model: "UT60G", Also: "Tenma 72-7750", Proto: "es519xx11-19200",
		Transports: []Transport{TrCH9325, TrSerial}, Serial: SerialCfg{Baud: 19200, DataBits: 7, Parity: 'O', StopBits: 1, DTR: true}, Counts: 22000, Support: SupVerified,
		Connect: "UT-D04 USB cable or UT-D02 RS-232 cable. Enable the meter's RS232 output (long press)."},
	{ID: "isotech-idm103n", Brand: "ISO-TECH", Model: "IDM103N", Proto: "es519xx11-2400",
		Transports: []Transport{TrSerial}, Serial: SerialCfg{Baud: 2400, DataBits: 7, Parity: 'O', StopBits: 1, DTR: true}, Counts: 4000, Support: SupVerified, Connect: pcLinkFS},

	{ID: "uni-t-ut181a", Brand: "UNI-T", Model: "UT181A", Proto: "ut181a",
		Transports: []Transport{TrCP2110}, Serial: ser(9600, 8, 'N', 1), Counts: 60000, Support: SupVerified, Featured: true,
		Connect: "Plug in the UT-D09 USB cable (CP2110 HID bridge, no driver). Turn on the meter's communication (SETUP → Communication ON). Meterie switches on live monitoring by itself."},

	{ID: "brymen-bm86x", Brand: "Brymen", Model: "BM869s / BM867s / BM859s", Also: "Greenlee DM-860A, Amprobe 'BM86x' rebrands", Proto: "bm86x",
		Transports: []Transport{TrBU86X}, Counts: 50000, Support: SupVerified,
		Connect: "Clip the Brymen BU-86X USB cable onto the meter's IR port (HID, no driver) and turn the meter on. Meterie polls the meter itself; main and secondary displays are both read."},
	{ID: "brymen-bm25x", Brand: "Brymen", Model: "BM25x (BM257s …)", Also: "BM235 owners: try this with Brymen's IR cable", Proto: "bm25x",
		Transports: []Transport{TrSerial}, Serial: SerialCfg{Baud: 9600, DataBits: 8, Parity: 'N', StopBits: 1, DTR: true, RTS: true}, Counts: 6000, Support: SupVerified,
		Connect: "Use Brymen's BRUA-20X / BU-2577 IR-to-USB cable (COM port). Meterie raises DTR and RTS to power the IR adapter, then the meter streams continuously."},

	{ID: "zoyi-ble", Brand: "ANENG / ZOYI / BSIDE", Model: "Bluetooth DMM family", Also: "ANENG AN9002, V05B, ST207, AN999S · ZOYI/BSIDE ZT-300AB, ZT-5B, ZT-5BQ, ZT-5566S/SE · BABATools AD900", Proto: "zoyi",
		Transports: []Transport{TrBLE}, Counts: 6000, BLENames: []string{"Bluetooth DMM", "ZT-", "AN9002", "AN999"},
		BLEService: "fff0", BLEChar: "fff4", Support: SupCommunity, Featured: true,
		Connect: "Switch on the meter's Bluetooth (long-press the BT/Hz key; the BT icon shows). It advertises as 'Bluetooth DMM'. Don't pair it in Windows — Meterie connects directly."},
}

func meterByID(id string) *MeterModel {
	for _, m := range meterCatalog {
		if m.ID == id {
			return m
		}
	}
	return nil
}

// Known USB IDs.
type usbID struct {
	VID, PID uint16
	Kind     Transport
	Label    string
	Meters   []string // likely meter IDs
}

var knownUSB = []usbID{
	{0x1A86, 0xE008, TrCH9325, "UNI-T UT-D04 cable (CH9325)", []string{"uni-t-ut61e", "uni-t-ut61bcd", "uni-t-ut60e", "uni-t-ut71", "uni-t-ut60g"}},
	{0x04FA, 0x2490, TrCH9325, "UNI-T UT-D04 cable (HE2325U)", []string{"uni-t-ut61e", "uni-t-ut61bcd", "uni-t-ut60e", "uni-t-ut71", "uni-t-ut60g"}},
	{0x10C4, 0xEA80, TrCP2110, "UNI-T CP2110 cable (UT61E+ / UT161 / UT181A)", []string{"uni-t-ut61eplus", "uni-t-ut181a"}},
	{0x1244, 0xD237, TrVictor, "Victor 70C/86C cable", []string{"victor-86c"}},
	{0x0820, 0x0001, TrBU86X, "Brymen BU-86X cable", []string{"brymen-bm86x"}},
	{0x1A86, 0x7523, TrSerial, "CH340 USB-serial", nil},
	{0x1A86, 0x7522, TrSerial, "CH340 USB-serial", nil},
	{0x1A86, 0x5523, TrSerial, "CH341 USB-serial", nil},
	{0x10C4, 0xEA60, TrSerial, "CP210x USB-serial", nil},
	{0x067B, 0x2303, TrSerial, "Prolific PL2303", nil},
	{0x067B, 0x23A3, TrSerial, "Prolific PL2303", nil},
	{0x0403, 0x6001, TrSerial, "FTDI USB-serial (Fluke IR cable?)", []string{"fluke-287"}},
	{0x0403, 0x6015, TrSerial, "FTDI USB-serial", nil},
	{0x04FA, 0x2490, TrSerial, "Hoitek", nil},
}

func lookupUSB(vid, pid uint16, kind Transport) *usbID {
	for i := range knownUSB {
		u := &knownUSB[i]
		if u.VID == vid && u.PID == pid && u.Kind == kind {
			return u
		}
	}
	for i := range knownUSB {
		u := &knownUSB[i]
		if u.VID == vid && u.PID == pid {
			return u
		}
	}
	return nil
}

// newDecoder builds the decoder for a meter model.
func newDecoder(m *MeterModel) Decoder {
	switch m.Proto {
	case "fs9721":
		return NewFS9721(m.TempBit, m.Counts)
	case "fs9922":
		return NewFS9922(m.Counts)
	case "es51922":
		return &ES51922{}
	case "ut61eplus":
		return &UT61EPlus{}
	case "owon":
		return &OwonBLE{}
	case "fluke28x":
		return &FlukeQM{Is28: true}
	case "fluke18x":
		return &FlukeQM{}
	case "metex14":
		return &Metex14{}
	case "scpi":
		return &SCPIMeter{}
	case "victor":
		return &VictorHID{variant: -1}
	case "ut71x":
		return NewUT71x()
	case "es519xx11-19200":
		return NewES519xx11(19200)
	case "es519xx11-2400":
		return NewES519xx11(2400)
	case "bm25x":
		return &BM25x{}
	case "bm86x":
		return &BM86x{}
	case "ut181a":
		return &UT181A{}
	case "zoyi":
		return ZoyiBLE{}
	}
	return RawDecoder{}
}

// bleNameMatch finds a catalogue entry advertising this name.
func bleNameMatch(name string) *MeterModel {
	up := strings.ToUpper(name)
	for _, m := range meterCatalog {
		for _, p := range m.BLENames {
			if strings.HasPrefix(up, strings.ToUpper(p)) {
				return m
			}
		}
	}
	return nil
}

// Probe plans: for a generic USB-serial port we try these in order.
type probePlan struct {
	MeterID string
	Cfg     SerialCfg
}

var serialProbePlans = []probePlan{
	{"generic-fs9721", ser(2400, 8, 'N', 1)},
	{"generic-fs9922", ser(2400, 8, 'N', 1)},
	{"uni-t-ut61e", SerialCfg{Baud: 19200, DataBits: 7, Parity: 'O', StopBits: 1, DTR: true}},
	{"uni-t-ut60g", SerialCfg{Baud: 19200, DataBits: 7, Parity: 'O', StopBits: 1, DTR: true}},
	{"uni-t-ut71", SerialCfg{Baud: 2400, DataBits: 7, Parity: 'O', StopBits: 1, DTR: true}},
	{"isotech-idm103n", SerialCfg{Baud: 2400, DataBits: 7, Parity: 'O', StopBits: 1, DTR: true}},
	{"brymen-bm25x", SerialCfg{Baud: 9600, DataBits: 8, Parity: 'N', StopBits: 1, DTR: true, RTS: true}},
	{"fluke-287", ser(115200, 8, 'N', 1)},
	{"fluke-189", ser(9600, 8, 'N', 1)},
	{"owon-xdm", ser(115200, 8, 'N', 1)},
	{"mastech-mas345", SerialCfg{Baud: 600, DataBits: 7, Parity: 'N', StopBits: 2, DTR: true}},
	{"mastech-mas345", SerialCfg{Baud: 1200, DataBits: 7, Parity: 'N', StopBits: 2, DTR: true}},
}

// Meters we know about but whose protocol we could not verify (shown honestly in the picker).
var unsupportedNotes = []struct{ Name, Why string }{
	{"UNI-T UT171A / B / C", "different packet layout from the UT181A that I couldn't verify — use Camera mode for now"},
	{"Brymen BM235 (and other BM23x)", "no documented PC protocol; if you have Brymen's IR cable try the BM25x entry, otherwise use Camera mode"},
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var b [20]byte
	p := len(b)
	for i > 0 {
		p--
		b[p] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		p--
		b[p] = '-'
	}
	return string(b[p:])
}
