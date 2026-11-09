# nrmluvdev suite: shared timeline format & Meterie API

This document is for the suite's other apps (Scopie, Serialie / Logicie, Wirie's Calcie …) so
they can exchange captures with Meterie and read its live values.

## 1. nrmluv-timeline v1 (`.nst.csv`)

A plain CSV file. All times are **absolute UTC nanoseconds since the Unix epoch**, so files
from different apps line up on one timeline without any manual offset.

```
# nrmluv-timeline v1
# app=Meterie version=0.6.0-alpha author=nrmluvdev exported=2026-11-07T19:30:00Z
# t0_unix_ns=1794079800000000000
# channel id=1 name=CH1 unit=V kind=meter source=HoldPeak_HP-90EPC_·_COM5
# channel id=2 name=CH2 unit=A kind=meter source=UNI-T_UT61E+
# marker t_unix_ns=1794079812500000000 text=load_on
t_unix_ns,t_rel_s,channel_id,channel,value,unit,flags
1794079800000000000,0.000000,1,CH1,4.071,V,DC|AUTO
1794079800250000000,0.250000,2,CH2,0.473,A,DC|AUTO
```

Rules

* **Line 1 must be exactly** `# nrmluv-timeline v1` (that is how importers recognise the file).
* Further `#` lines are metadata; unknown ones must be ignored.
  * `# app=<name> version=<v> author=<a> exported=<RFC3339>`
  * `# t0_unix_ns=<int>` — the first sample's time (for `t_rel_s`).
  * `# channel id=<int> name=<n> unit=<u> kind=<k> source=<s>` — spaces in values are written as `_`.
  * `# marker t_unix_ns=<int> text=<text>` — annotations (spaces as `_`).
* Header row then one row per sample, sorted by time:
  * `t_unix_ns` — int64, absolute time (required)
  * `t_rel_s` — seconds since `t0_unix_ns` (informative)
  * `channel_id`, `channel` — id and display name
  * `value` — in **SI base units** (V, A, Ω, F, Hz, %, °C, W, Wh, Ah, s …); `OL` for overload
  * `unit` — base unit symbol
  * `flags` — `|`-separated: `AC DC AUTO MANUAL HOLD REL MIN MAX LOWBAT DIODE BEEP OL HV` (may be empty)
* Logic/scope apps can use the same layout: e.g. Logicie writes one row per edge with value `0`/`1`
  and unit empty; Scopie can write decimated samples (value in V) or one row per measurement.
  Meterie imports any such file and plots each channel on its own scale.

Import: **Export → Import a suite timeline** (or Ctrl+O and pick a `.csv`).

## 2. Live readings over HTTP (127.0.0.1 only)

Default port **47823** (Settings → Suite integration). If it's busy Meterie picks a free port
and records it in the discovery file below.

| Endpoint | Returns |
|---|---|
| `GET /api/v1/ping` | `{"app":"Meterie","version":"0.6.0-alpha","author":"nrmluvdev"}` |
| `GET /api/v1/readings` | all channels' latest readings (JSON, see below) |
| `GET /api/v1/reading` | the main channel's latest reading (`?channel=2` for another) |
| `GET /api/v1/value` | plain text, e.g. `4700 Ω` (main channel, base units) |
| `GET /api/v1/stream` | Server-Sent Events: one JSON reading per `data:` line, live |

Reading object:

```json
{
  "id": 1, "name": "CH1", "source": "HoldPeak HP-90EPC · COM5",
  "value": 4700.0, "unit": "Ω", "prefix": "k", "display": "4.700",
  "text": "4.70 kΩ", "mode": "RESISTANCE", "flags": "AUTO",
  "ol": false, "time": "2026-11-07T19:30:01.25Z", "unix_ms": 1794079801250, "main": true
}
```

`value` is always in base SI units (`null` when overloaded) — so a calculator can use it
directly: 4.7 kΩ arrives as `4700`.

### Example (Go, e.g. for Calcie)

```go
type MeterieReading struct {
    Value *float64 `json:"value"`
    Unit  string   `json:"unit"`
    Text  string   `json:"text"`
}

func pullFromMeterie() (float64, string, error) {
    c := http.Client{Timeout: 800 * time.Millisecond}
    resp, err := c.Get("http://127.0.0.1:47823/api/v1/reading")
    if err != nil {
        return 0, "", fmt.Errorf("Meterie isn't running")
    }
    defer resp.Body.Close()
    var r MeterieReading
    if err := json.NewDecoder(resp.Body).Decode(&r); err != nil || r.Value == nil {
        return 0, "", fmt.Errorf("no reading yet")
    }
    return *r.Value, r.Unit, nil
}
```

## 3. Files (no networking needed)

In `%APPDATA%\nrmluvdev\suite\`:

* `meterie.json` — discovery: `{"app":"Meterie","port":47823,"readings":"http://127.0.0.1:47823/api/v1/readings","pid":1234,...}` (removed when Meterie quits)
* `meterie-live.json` — the same content as `/api/v1/readings`, rewritten at most twice a second (written atomically via rename)

Other suite apps are encouraged to publish the same kind of `<app>.json` discovery file in this folder.
