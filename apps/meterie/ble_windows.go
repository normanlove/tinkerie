//go:build windows

package meterie

// ble_windows.go: Bluetooth LE via Windows Runtime (WinRT) in pure Go —
// advertisement scanning, GATT connect, and characteristic notifications.
// Uses combase.dll only (part of Windows 10/11).

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

var (
	combase                    = sysDLL("combase.dll")
	pRoInitialize              = combase.NewProc("RoInitialize")
	pRoActivateInstance        = combase.NewProc("RoActivateInstance")
	pRoGetActivationFactory    = combase.NewProc("RoGetActivationFactory")
	pWindowsCreateString       = combase.NewProc("WindowsCreateString")
	pWindowsDeleteString       = combase.NewProc("WindowsDeleteString")
	pWindowsGetStringRawBuffer = combase.NewProc("WindowsGetStringRawBuffer")
)

var (
	iidIUnknown        = parseGUID("00000000-0000-0000-C000-000000000046")
	iidAgile           = parseGUID("94ea2b94-e9cc-49e0-c0ff-ee64ca8f5b90")
	iidAsyncInfo       = parseGUID("00000036-0000-0000-C000-000000000046")
	iidClosable        = parseGUID("30D5A829-7FA4-4026-83BB-D75BAE4EA99E")
	iidWatcher         = parseGUID("A6AC336F-F3D3-4297-8D6C-C81EA6623F40")
	iidDevStatics      = parseGUID("C8CF1A19-F0B6-4BF0-8689-41303DE2D9F4")
	iidDevice3         = parseGUID("AEE9E493-44AC-40DC-AF33-B2C13C01CA46")
	iidService3        = parseGUID("B293A950-0C53-437C-A9B3-5C3210C6E569")
	iidBufferAccess    = parseGUID("905a0fef-bc53-11df-8c49-001e4fc686da")
	iidReceivedHandler = pinterfaceIID(typedEventHandlerSig(
		"Windows.Devices.Bluetooth.Advertisement.BluetoothLEAdvertisementWatcher", "{a6ac336f-f3d3-4297-8d6c-c81ea6623f40}",
		"Windows.Devices.Bluetooth.Advertisement.BluetoothLEAdvertisementReceivedEventArgs", "{27987ddf-e596-41be-8d43-9e6731d4a913}"))
	iidValueChangedHandler = pinterfaceIID(typedEventHandlerSig(
		"Windows.Devices.Bluetooth.GenericAttributeProfile.GattCharacteristic", "{59cb50c1-5934-4f68-a198-eb864fa44e6b}",
		"Windows.Devices.Bluetooth.GenericAttributeProfile.GattValueChangedEventArgs", "{d21bdb54-06e3-4ed8-a263-acfac8ba7313}"))
)

// vcall invokes vtable method idx on a COM/WinRT object.
func vcall(obj uintptr, idx int, args ...uintptr) uintptr {
	vtbl := *(*uintptr)(unsafe.Pointer(obj))
	fn := *(*uintptr)(unsafe.Pointer(vtbl + uintptr(idx)*unsafe.Sizeof(uintptr(0))))
	r, _, _ := syscall.SyscallN(fn, append([]uintptr{obj}, args...)...)
	return r
}

func failed(hr uintptr) bool { return int32(hr) < 0 }

func release(obj uintptr) {
	if obj != 0 {
		vcall(obj, 2)
	}
}

func qi(obj uintptr, iid *guidT) (uintptr, error) {
	var out uintptr
	if hr := vcall(obj, 0, uintptr(unsafe.Pointer(iid)), uintptr(unsafe.Pointer(&out))); failed(hr) {
		return 0, fmt.Errorf("QueryInterface %s: 0x%08X", iid, uint32(hr))
	}
	return out, nil
}

func hstring(s string) uintptr {
	w, _ := syscall.UTF16FromString(s)
	var h uintptr
	pWindowsCreateString.Call(uintptr(unsafe.Pointer(&w[0])), uintptr(len(w)-1), uintptr(unsafe.Pointer(&h)))
	return h
}

func hstringToGo(h uintptr) string {
	if h == 0 {
		return ""
	}
	var n uint32
	p, _, _ := pWindowsGetStringRawBuffer.Call(h, uintptr(unsafe.Pointer(&n)))
	if p == 0 || n == 0 {
		return ""
	}
	return syscall.UTF16ToString(unsafe.Slice((*uint16)(unsafe.Pointer(p)), n))
}

// ---------------------------------------------------------------------------
// A dedicated OS thread initialised for WinRT (multi-threaded apartment).

var (
	winrtOnce sync.Once
	winrtCh   chan func()
	winrtErr  error
)

func winrtDo(f func()) error {
	winrtOnce.Do(func() {
		winrtCh = make(chan func())
		ready := make(chan error)
		go func() {
			lockThread()
			hr, _, _ := pRoInitialize.Call(1) // RO_INIT_MULTITHREADED
			if failed(hr) && uint32(hr) != 0x80010106 {
				ready <- fmt.Errorf("RoInitialize failed: 0x%08X", uint32(hr))
				return
			}
			ready <- nil
			for f := range winrtCh {
				f()
			}
		}()
		winrtErr = <-ready
		if err := combase.Load(); err != nil {
			winrtErr = errors.New("Windows Runtime not available (Bluetooth LE needs Windows 10 or 11)")
		}
	})
	if winrtErr != nil {
		return winrtErr
	}
	done := make(chan struct{})
	winrtCh <- func() {
		defer func() {
			if r := recover(); r != nil {
				winrtErr = fmt.Errorf("bluetooth error: %v", r)
			}
			close(done)
		}()
		f()
	}
	<-done
	return nil
}

// ---------------------------------------------------------------------------
// Delegate objects implemented in Go (TypedEventHandler<,>).

type delegateVtbl struct {
	QueryInterface, AddRef, Release, Invoke uintptr
}

type delegate struct {
	vtbl *delegateVtbl
	refs int32
	iid  guidT
	fn   func(sender, args uintptr)
}

var (
	delegateVT   *delegateVtbl
	delegatesMu  sync.Mutex
	delegateKeep = map[uintptr]*delegate{}
)

func newDelegate(iid guidT, fn func(sender, args uintptr)) *delegate {
	if delegateVT == nil {
		delegateVT = &delegateVtbl{
			QueryInterface: syscall.NewCallback(func(this *delegate, riid *guidT, out *uintptr) uintptr {
				if *riid == this.iid || *riid == iidIUnknown || *riid == iidAgile {
					*out = uintptr(unsafe.Pointer(this))
					this.refs++
					return 0
				}
				*out = 0
				return 0x80004002 // E_NOINTERFACE
			}),
			AddRef: syscall.NewCallback(func(this *delegate) uintptr {
				this.refs++
				return uintptr(this.refs)
			}),
			Release: syscall.NewCallback(func(this *delegate) uintptr {
				this.refs--
				return uintptr(this.refs)
			}),
			Invoke: syscall.NewCallback(func(this *delegate, sender, args uintptr) uintptr {
				func() {
					defer func() { recover() }()
					this.fn(sender, args)
				}()
				return 0
			}),
		}
	}
	d := &delegate{vtbl: delegateVT, refs: 1, iid: iid, fn: fn}
	delegatesMu.Lock()
	delegateKeep[uintptr(unsafe.Pointer(d))] = d // keep alive: COM holds a raw pointer
	delegatesMu.Unlock()
	return d
}

// await polls an IAsyncOperation until done and returns its result.
func await(op uintptr, timeout time.Duration) (uintptr, error) {
	if op == 0 {
		return 0, errors.New("no async operation")
	}
	defer release(op)
	info, err := qi(op, &iidAsyncInfo)
	if err != nil {
		return 0, err
	}
	defer release(info)
	end := time.Now().Add(timeout)
	for {
		var st int32
		vcall(info, 7, uintptr(unsafe.Pointer(&st)))
		switch st {
		case 1: // Completed
			var res uintptr
			if hr := vcall(op, 8, uintptr(unsafe.Pointer(&res))); failed(hr) {
				return 0, fmt.Errorf("GetResults: 0x%08X", uint32(hr))
			}
			return res, nil
		case 2:
			return 0, errors.New("cancelled")
		case 3:
			var code int32
			vcall(info, 8, uintptr(unsafe.Pointer(&code)))
			return 0, fmt.Errorf("Bluetooth operation failed (0x%08X)", uint32(code))
		}
		if time.Now().After(end) {
			vcall(info, 9) // Cancel
			return 0, errors.New("Bluetooth operation timed out")
		}
		time.Sleep(15 * time.Millisecond)
	}
}

// ---------------------------------------------------------------------------
// Scanning

var ble struct {
	mu      sync.Mutex
	results map[uint64]*PortInfo
	watcher uintptr
	token   int64
	handler *delegate
}

func fmtBTAddr(a uint64) string {
	return fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X", byte(a>>40), byte(a>>32), byte(a>>24), byte(a>>16), byte(a>>8), byte(a))
}

func platformBLEScan(on bool) error {
	var outErr error
	err := winrtDo(func() {
		if !on {
			if ble.watcher != 0 {
				vcall(ble.watcher, 18) // Stop
				vcall(ble.watcher, 20, uintptr(ble.token))
				release(ble.watcher)
				ble.watcher = 0
			}
			return
		}
		if ble.watcher != 0 {
			return
		}
		ble.mu.Lock()
		if ble.results == nil {
			ble.results = map[uint64]*PortInfo{}
		}
		ble.mu.Unlock()
		cls := hstring("Windows.Devices.Bluetooth.Advertisement.BluetoothLEAdvertisementWatcher")
		defer pWindowsDeleteString.Call(cls)
		var insp uintptr
		if hr, _, _ := pRoActivateInstance.Call(cls, uintptr(unsafe.Pointer(&insp))); failed(hr) || insp == 0 {
			outErr = fmt.Errorf("can't start Bluetooth scanning (0x%08X) — is Bluetooth switched on?", uint32(hr))
			return
		}
		w, err := qi(insp, &iidWatcher)
		release(insp)
		if err != nil {
			outErr = err
			return
		}
		vcall(w, 12, 1) // put_ScanningMode(Active) to receive names in scan responses
		if ble.handler == nil {
			ble.handler = newDelegate(iidReceivedHandler, onAdvert)
		}
		if hr := vcall(w, 19, uintptr(unsafe.Pointer(ble.handler)), uintptr(unsafe.Pointer(&ble.token))); failed(hr) {
			release(w)
			outErr = fmt.Errorf("add_Received failed (0x%08X)", uint32(hr))
			return
		}
		if hr := vcall(w, 17); failed(hr) {
			release(w)
			outErr = fmt.Errorf("Bluetooth scan failed to start (0x%08X) — turn Bluetooth on in Windows settings", uint32(hr))
			return
		}
		ble.watcher = w
	})
	if err != nil {
		return err
	}
	return outErr
}

func onAdvert(sender, args uintptr) {
	var addr uint64
	var rssi int16
	vcall(args, 7, uintptr(unsafe.Pointer(&addr)))
	vcall(args, 6, uintptr(unsafe.Pointer(&rssi)))
	name := ""
	var adv uintptr
	if hr := vcall(args, 10, uintptr(unsafe.Pointer(&adv))); !failed(hr) && adv != 0 {
		var hs uintptr
		if hr := vcall(adv, 8, uintptr(unsafe.Pointer(&hs))); !failed(hr) {
			name = hstringToGo(hs)
			pWindowsDeleteString.Call(hs)
		}
		release(adv)
	}
	ble.mu.Lock()
	p := ble.results[addr]
	if p == nil {
		p = &PortInfo{Kind: TrBLE, Path: fmtBTAddr(addr), BLEAddr: addr}
		ble.results[addr] = p
	}
	if name != "" {
		p.Name = name
	}
	p.RSSI = int(rssi)
	ble.mu.Unlock()
	appWake()
}

func platformBLEResults() []PortInfo {
	ble.mu.Lock()
	defer ble.mu.Unlock()
	var out []PortInfo
	for _, p := range ble.results {
		if p.Name == "" && bleNameMatch(p.Name) == nil && p.RSSI < -85 {
			continue
		}
		out = append(out, *p)
	}
	return out
}

// ---------------------------------------------------------------------------
// GATT connection

type blePort struct {
	dev, char uintptr
	token     int64
	frames    chan []byte
	closed    chan struct{}
	once      sync.Once
	handler   *delegate
}

func platformOpenBLE(addr uint64, svc, chr string) (Port, error) {
	if addr == 0 {
		return nil, errors.New("no Bluetooth address — scan again")
	}
	p := &blePort{frames: make(chan []byte, 256), closed: make(chan struct{})}
	var outErr error
	err := winrtDo(func() { outErr = p.connect(addr, svc, chr) })
	if err != nil {
		return nil, err
	}
	if outErr != nil {
		p.Close()
		return nil, outErr
	}
	return p, nil
}

func (p *blePort) connect(addr uint64, svc, chr string) error {
	cls := hstring("Windows.Devices.Bluetooth.BluetoothLEDevice")
	defer pWindowsDeleteString.Call(cls)
	var statics uintptr
	if hr, _, _ := pRoGetActivationFactory.Call(cls, uintptr(unsafe.Pointer(&iidDevStatics)), uintptr(unsafe.Pointer(&statics))); failed(hr) {
		return fmt.Errorf("Bluetooth LE unavailable (0x%08X)", uint32(hr))
	}
	defer release(statics)
	var op uintptr
	if hr := vcall(statics, 7, uintptr(addr), uintptr(unsafe.Pointer(&op))); failed(hr) {
		return fmt.Errorf("FromBluetoothAddressAsync failed (0x%08X)", uint32(hr))
	}
	dev, err := await(op, 15*time.Second)
	if err != nil {
		return err
	}
	if dev == 0 {
		return errors.New("meter not found — is its Bluetooth on and in range?")
	}
	p.dev = dev
	dev3, err := qi(dev, &iidDevice3)
	if err != nil {
		return errors.New("this Windows version is too old for Bluetooth LE GATT (needs Windows 10 1803+)")
	}
	defer release(dev3)
	if svc != "" {
		g := bleUUID(svc)
		hr := vcall(dev3, 10, uintptr(unsafe.Pointer(&g)), uintptr(unsafe.Pointer(&op))) // GetGattServicesForUuidAsync
		if failed(hr) {
			return fmt.Errorf("service lookup failed (0x%08X)", uint32(hr))
		}
	} else if hr := vcall(dev3, 8, uintptr(unsafe.Pointer(&op))); failed(hr) { // GetGattServicesAsync
		return fmt.Errorf("service discovery failed (0x%08X)", uint32(hr))
	}
	res, err := await(op, 20*time.Second)
	if err != nil {
		return err
	}
	defer release(res)
	var status int32
	vcall(res, 6, uintptr(unsafe.Pointer(&status)))
	if status != 0 {
		return fmt.Errorf("couldn't reach the meter (%s) — move closer or toggle its Bluetooth", gattStatus(status))
	}
	var services uintptr
	vcall(res, 8, uintptr(unsafe.Pointer(&services)))
	if services == 0 {
		return errors.New("no GATT services")
	}
	defer release(services)
	var n uint32
	vcall(services, 7, uintptr(unsafe.Pointer(&n)))
	want := guidT{}
	if chr != "" {
		want = bleUUID(chr)
	}
	for i := uint32(0); i < n && p.char == 0; i++ {
		var s uintptr
		vcall(services, 6, uintptr(i), uintptr(unsafe.Pointer(&s)))
		if s == 0 {
			continue
		}
		s3, err := qi(s, &iidService3)
		release(s)
		if err != nil {
			continue
		}
		var cop uintptr
		if hr := vcall(s3, 11, uintptr(unsafe.Pointer(&cop))); failed(hr) { // GetCharacteristicsAsync
			release(s3)
			continue
		}
		cres, err := await(cop, 15*time.Second)
		release(s3)
		if err != nil || cres == 0 {
			continue
		}
		var chars uintptr
		vcall(cres, 8, uintptr(unsafe.Pointer(&chars)))
		release(cres)
		if chars == 0 {
			continue
		}
		var cn uint32
		vcall(chars, 7, uintptr(unsafe.Pointer(&cn)))
		for j := uint32(0); j < cn; j++ {
			var c uintptr
			vcall(chars, 6, uintptr(j), uintptr(unsafe.Pointer(&c)))
			if c == 0 {
				continue
			}
			var id guidT
			var props uint32
			vcall(c, 11, uintptr(unsafe.Pointer(&id)))
			vcall(c, 7, uintptr(unsafe.Pointer(&props)))
			match := false
			if chr != "" {
				match = id == want
			} else {
				match = props&0x30 != 0 // Notify|Indicate
			}
			if match && p.char == 0 {
				p.char = c
				if props&0x10 == 0 && props&0x20 != 0 {
					p.token = 2 // remember: indicate
				}
			} else {
				release(c)
			}
		}
		release(chars)
	}
	if p.char == 0 {
		return errors.New("the meter's data characteristic wasn't found — wrong protocol selected?")
	}
	mode := uintptr(1)
	if p.token == 2 {
		mode = 2
	}
	p.token = 0
	var wop uintptr
	if hr := vcall(p.char, 19, mode, uintptr(unsafe.Pointer(&wop))); failed(hr) {
		return fmt.Errorf("couldn't enable notifications (0x%08X)", uint32(hr))
	}
	if st, err := await(wop, 10*time.Second); err != nil {
		return err
	} else if st != 0 {
		return fmt.Errorf("meter refused notifications (%s)", gattStatus(int32(st)))
	}
	p.handler = newDelegate(iidValueChangedHandler, p.onValue)
	if hr := vcall(p.char, 20, uintptr(unsafe.Pointer(p.handler)), uintptr(unsafe.Pointer(&p.token))); failed(hr) {
		return fmt.Errorf("add_ValueChanged failed (0x%08X)", uint32(hr))
	}
	return nil
}

func gattStatus(s int32) string {
	switch s {
	case 1:
		return "unreachable"
	case 2:
		return "protocol error"
	case 3:
		return "access denied"
	}
	return fmt.Sprintf("status %d", s)
}

func (p *blePort) onValue(sender, args uintptr) {
	var buf uintptr
	if hr := vcall(args, 6, uintptr(unsafe.Pointer(&buf))); failed(hr) || buf == 0 {
		return
	}
	defer release(buf)
	var n uint32
	vcall(buf, 7, uintptr(unsafe.Pointer(&n)))
	acc, err := qi(buf, &iidBufferAccess)
	if err != nil || n == 0 {
		return
	}
	defer release(acc)
	var ptr uintptr
	vcall(acc, 3, uintptr(unsafe.Pointer(&ptr)))
	if ptr == 0 {
		return
	}
	data := append([]byte(nil), unsafe.Slice((*byte)(unsafe.Pointer(ptr)), n)...)
	select {
	case p.frames <- data:
	default:
	}
}

func (p *blePort) Read(b []byte) (int, error) {
	select {
	case f := <-p.frames:
		return copy(b, f), nil
	case <-p.closed:
		return 0, errors.New("Bluetooth connection closed")
	case <-time.After(200 * time.Millisecond):
		return 0, nil
	}
}

func (p *blePort) Write(b []byte) (int, error) { return len(b), nil }

func (p *blePort) Close() error {
	p.once.Do(func() {
		close(p.closed)
		winrtDo(func() {
			if p.char != 0 {
				if p.token != 0 {
					vcall(p.char, 21, uintptr(p.token))
				}
				release(p.char)
				p.char = 0
			}
			if p.dev != 0 {
				if c, err := qi(p.dev, &iidClosable); err == nil {
					vcall(c, 6)
					release(c)
				}
				release(p.dev)
				p.dev = 0
			}
		})
	})
	return nil
}

var _ = strings.ToLower
