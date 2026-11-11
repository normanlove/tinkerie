//go:build darwin && cgo

package usb

// darwin.go: USB on macOS through IOKit's user-space USB interface (the only
// way in on a Mac, hence cgo for this file; the Mac build is made on a Mac).
// Devices are named "usb:<locationID>", which stays the same while a device
// stays in the same port. Created by nrmluvdev for Tinkerie.

/*
#cgo LDFLAGS: -framework IOKit -framework CoreFoundation
#include <stdlib.h>
#include <string.h>
#include <CoreFoundation/CoreFoundation.h>
#include <IOKit/IOKitLib.h>
#include <IOKit/IOCFPlugIn.h>
#include <IOKit/usb/IOUSBLib.h>

typedef IOUSBDeviceInterface320 **tkDev;
typedef IOUSBInterfaceInterface300 **tkIntf;

// ---------------------------------------------------------------- listing

static io_service_t *gDevs = NULL;
static int gN = 0;

static void tk_list_end(void) {
	for (int i = 0; i < gN; i++) IOObjectRelease(gDevs[i]);
	free(gDevs);
	gDevs = NULL;
	gN = 0;
}

static int tk_collect(const char *cls) {
	io_iterator_t it = 0;
	if (IOServiceGetMatchingServices(0, IOServiceMatching(cls), &it) != KERN_SUCCESS) return 0;
	io_service_t s;
	while ((s = IOIteratorNext(it))) {
		gDevs = realloc(gDevs, sizeof(io_service_t) * (gN + 1));
		gDevs[gN++] = s;
	}
	IOObjectRelease(it);
	return gN;
}

static int tk_list_begin(void) {
	tk_list_end();
	if (tk_collect("IOUSBHostDevice") == 0) tk_collect(kIOUSBDeviceClassName); // newer, then older USB stacks
	return gN;
}

static long long tk_num(io_service_t s, const char *key) {
	CFStringRef k = CFStringCreateWithCString(NULL, key, kCFStringEncodingUTF8);
	CFTypeRef v = IORegistryEntryCreateCFProperty(s, k, kCFAllocatorDefault, 0);
	CFRelease(k);
	long long n = -1;
	if (v && CFGetTypeID(v) == CFNumberGetTypeID()) CFNumberGetValue((CFNumberRef)v, kCFNumberLongLongType, &n);
	if (v) CFRelease(v);
	return n;
}

static int tk_str(io_service_t s, const char *key, char *buf, int cap) {
	buf[0] = 0;
	CFStringRef k = CFStringCreateWithCString(NULL, key, kCFStringEncodingUTF8);
	CFTypeRef v = IORegistryEntryCreateCFProperty(s, k, kCFAllocatorDefault, 0);
	CFRelease(k);
	int ok = 0;
	if (v && CFGetTypeID(v) == CFStringGetTypeID()) ok = CFStringGetCString((CFStringRef)v, buf, cap, kCFStringEncodingUTF8);
	if (v) CFRelease(v);
	return ok;
}

static void tk_info(int i, long long *loc, long long *vid, long long *pid, long long *bcd) {
	io_service_t s = gDevs[i];
	*loc = tk_num(s, "locationID");
	*vid = tk_num(s, "idVendor");
	*pid = tk_num(s, "idProduct");
	*bcd = tk_num(s, "bcdDevice");
}

static void tk_strings(int i, char *man, char *prod, char *ser, int cap) {
	io_service_t s = gDevs[i];
	if (!tk_str(s, "USB Vendor Name", man, cap)) tk_str(s, "kUSBVendorString", man, cap);
	if (!tk_str(s, "USB Product Name", prod, cap)) tk_str(s, "kUSBProductString", prod, cap);
	if (!tk_str(s, "USB Serial Number", ser, cap)) tk_str(s, "kUSBSerialNumberString", ser, cap);
}

static tkDev tk_mkdev(io_service_t s) {
	IOCFPlugInInterface **plug = NULL;
	SInt32 score;
	if (IOCreatePlugInInterfaceForService(s, kIOUSBDeviceUserClientTypeID, kIOCFPlugInInterfaceID, &plug, &score) != KERN_SUCCESS || !plug) return NULL;
	tkDev dev = NULL;
	(*plug)->QueryInterface(plug, CFUUIDGetUUIDBytes(kIOUSBDeviceInterfaceID320), (LPVOID *)&dev);
	IODestroyPlugInInterface(plug);
	return dev;
}

// tk_config copies the first configuration descriptor; the device needn't be open.
static int tk_config(int i, unsigned char *buf, int cap) {
	tkDev dev = tk_mkdev(gDevs[i]);
	if (!dev) return 0;
	IOUSBConfigurationDescriptorPtr d = NULL;
	int n = 0;
	if ((*dev)->GetConfigurationDescriptorPtr(dev, 0, &d) == kIOReturnSuccess && d) {
		n = USBToHostWord(d->wTotalLength);
		if (n > cap) n = cap;
		memcpy(buf, d, n);
	}
	(*dev)->Release(dev);
	return n;
}

// ---------------------------------------------------------------- open devices

typedef struct {
	tkDev dev;
	int devOpen;
	tkIntf intf[32];
	int nIntf;
	unsigned char pipeIntf[256], pipeRef[256]; // by endpoint address; pipeRef 0 = none
} tkUSB;

static io_service_t tk_find(long long loc) {
	io_service_t found = 0;
	const char *classes[] = {"IOUSBHostDevice", kIOUSBDeviceClassName};
	for (int c = 0; c < 2 && !found; c++) {
		io_iterator_t it = 0;
		if (IOServiceGetMatchingServices(0, IOServiceMatching(classes[c]), &it) != KERN_SUCCESS) continue;
		io_service_t s;
		while ((s = IOIteratorNext(it))) {
			if (!found && tk_num(s, "locationID") == loc) found = s;
			else IOObjectRelease(s);
		}
		IOObjectRelease(it);
	}
	return found;
}

// tk_claim opens one interface and learns its pipes. Returns an IOReturn.
static int tk_claim(tkUSB *h, int want) {
	IOUSBFindInterfaceRequest req;
	req.bInterfaceClass = req.bInterfaceSubClass = req.bInterfaceProtocol = req.bAlternateSetting = kIOUSBFindInterfaceDontCare;
	io_iterator_t it = 0;
	IOReturn kr = (*h->dev)->CreateInterfaceIterator(h->dev, &req, &it);
	if (kr != kIOReturnSuccess) return kr;
	io_service_t s;
	kr = kIOReturnNotFound;
	while ((s = IOIteratorNext(it))) {
		IOCFPlugInInterface **plug = NULL;
		SInt32 score;
		tkIntf intf = NULL;
		if (IOCreatePlugInInterfaceForService(s, kIOUSBInterfaceUserClientTypeID, kIOCFPlugInInterfaceID, &plug, &score) == KERN_SUCCESS && plug) {
			(*plug)->QueryInterface(plug, CFUUIDGetUUIDBytes(kIOUSBInterfaceInterfaceID300), (LPVOID *)&intf);
			IODestroyPlugInInterface(plug);
		}
		IOObjectRelease(s);
		if (!intf) continue;
		UInt8 num = 0xFF, alt = 0;
		(*intf)->GetInterfaceNumber(intf, &num);
		(*intf)->GetAlternateSetting(intf, &alt);
		if (num != want || alt != 0 || h->nIntf >= 32) {
			(*intf)->Release(intf);
			continue;
		}
		kr = (*intf)->USBInterfaceOpen(intf);
		if (kr != kIOReturnSuccess) kr = (*intf)->USBInterfaceOpenSeize(intf);
		if (kr != kIOReturnSuccess) {
			(*intf)->Release(intf);
			break;
		}
		int idx = h->nIntf++;
		h->intf[idx] = intf;
		UInt8 np = 0;
		(*intf)->GetNumEndpoints(intf, &np);
		for (UInt8 p = 1; p <= np; p++) {
			UInt8 dir, number, type, interval;
			UInt16 mps;
			if ((*intf)->GetPipeProperties(intf, p, &dir, &number, &type, &mps, &interval) != kIOReturnSuccess) continue;
			unsigned char addr = number | (dir == kUSBIn ? 0x80 : 0);
			h->pipeIntf[addr] = idx;
			h->pipeRef[addr] = p;
		}
		break;
	}
	IOObjectRelease(it);
	return kr;
}

static tkUSB *tk_open(long long loc, int iface, int *err) {
	io_service_t s = tk_find(loc);
	if (!s) { *err = kIOReturnNoDevice; return NULL; }
	tkDev dev = tk_mkdev(s);
	IOObjectRelease(s);
	if (!dev) { *err = kIOReturnNoResources; return NULL; }
	tkUSB *h = calloc(1, sizeof(tkUSB));
	h->dev = dev;
	if ((*dev)->USBDeviceOpen(dev) == kIOReturnSuccess || (*dev)->USBDeviceOpenSeize(dev) == kIOReturnSuccess) h->devOpen = 1;
	UInt8 cfg = 0;
	if ((*dev)->GetConfiguration(dev, &cfg) == kIOReturnSuccess && cfg == 0 && h->devOpen) {
		IOUSBConfigurationDescriptorPtr d = NULL;
		if ((*dev)->GetConfigurationDescriptorPtr(dev, 0, &d) == kIOReturnSuccess && d) (*dev)->SetConfiguration(dev, d->bConfigurationValue);
	}
	int kr = tk_claim(h, iface);
	if (kr != kIOReturnSuccess) {
		*err = kr;
		if (h->devOpen) (*dev)->USBDeviceClose(dev);
		(*dev)->Release(dev);
		free(h);
		return NULL;
	}
	*err = 0;
	return h;
}

static void tk_close(tkUSB *h) {
	for (int i = 0; i < h->nIntf; i++) {
		(*h->intf[i])->USBInterfaceClose(h->intf[i]);
		(*h->intf[i])->Release(h->intf[i]);
	}
	if (h->devOpen) (*h->dev)->USBDeviceClose(h->dev);
	(*h->dev)->Release(h->dev);
	free(h);
}

// tk_control returns bytes moved, or a negative IOReturn.
static long long tk_control(tkUSB *h, int rt, int req, int val, int idx, void *data, int len, int tmo) {
	IOUSBDevRequestTO r;
	memset(&r, 0, sizeof r);
	r.bmRequestType = rt; r.bRequest = req; r.wValue = val; r.wIndex = idx; r.wLength = len;
	r.pData = data; r.noDataTimeout = tmo; r.completionTimeout = tmo;
	IOReturn kr = h->nIntf > 0 ? (*h->intf[0])->ControlRequestTO(h->intf[0], 0, &r) : (*h->dev)->DeviceRequestTO(h->dev, &r);
	if (kr != kIOReturnSuccess) return -(long long)(unsigned int)kr;
	return r.wLenDone;
}

// tk_bulk returns bytes moved, or a negative IOReturn (timeouts give 0).
static long long tk_bulk(tkUSB *h, int ep, void *buf, int len, int tmo) {
	if (!h->pipeRef[ep & 0xFF]) return -(long long)(unsigned int)kIOReturnBadArgument;
	tkIntf intf = h->intf[h->pipeIntf[ep & 0xFF]];
	UInt8 pipe = h->pipeRef[ep & 0xFF];
	IOReturn kr;
	UInt32 n = len;
	if (ep & 0x80) kr = (*intf)->ReadPipeTO(intf, pipe, buf, &n, tmo, tmo);
	else kr = (*intf)->WritePipeTO(intf, pipe, buf, len, tmo, tmo);
	if (kr == kIOUSBTransactionTimeout || kr == kIOReturnTimeout) {
		(*intf)->ClearPipeStallBothEnds(intf, pipe); // a timed-out pipe must be cleared before reuse
		return 0;
	}
	if (kr != kIOReturnSuccess) return -(long long)(unsigned int)kr;
	return (ep & 0x80) ? (long long)n : len;
}

static void tk_reset_pipe(tkUSB *h, int ep) {
	if (!h->pipeRef[ep & 0xFF]) return;
	tkIntf intf = h->intf[h->pipeIntf[ep & 0xFF]];
	(*intf)->ClearPipeStallBothEnds(intf, h->pipeRef[ep & 0xFF]);
}
*/
import "C"

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"
)

var listMu sync.Mutex

// List finds every USB device (IOKit's registry; no device is opened).
func List() []Info {
	listMu.Lock()
	defer listMu.Unlock()
	n := int(C.tk_list_begin())
	defer C.tk_list_end()
	out := make([]Info, 0, n)
	man, prod, ser := (*C.char)(C.malloc(256)), (*C.char)(C.malloc(256)), (*C.char)(C.malloc(256))
	cfg := (*C.uchar)(C.malloc(4096))
	defer C.free(unsafe.Pointer(man))
	defer C.free(unsafe.Pointer(prod))
	defer C.free(unsafe.Pointer(ser))
	defer C.free(unsafe.Pointer(cfg))
	for i := 0; i < n; i++ {
		var loc, vid, pid, bcd C.longlong
		C.tk_info(C.int(i), &loc, &vid, &pid, &bcd)
		if loc < 0 || vid < 0 {
			continue
		}
		C.tk_strings(C.int(i), man, prod, ser, 256)
		path := fmt.Sprintf("usb:%08x", uint32(loc))
		in := Info{Path: path, SysPath: path, VID: uint16(vid), PID: uint16(pid), BCD: uint16(max(bcd, 0)),
			Manufacturer: C.GoString(man), Product: C.GoString(prod), Serial: C.GoString(ser), Accessible: true}
		if k := int(C.tk_config(C.int(i), cfg, 4096)); k > 0 {
			in.Interfaces = parseConfig(C.GoBytes(unsafe.Pointer(cfg), C.int(k)))
		}
		out = append(out, in)
	}
	return out
}

// parseConfig reads interfaces (first alternate setting) and their bulk
// endpoints from a configuration descriptor.
func parseConfig(b []byte) []Interface {
	var out []Interface
	cur := -1
	for i := 0; i+1 < len(b); {
		l, t := int(b[i]), b[i+1]
		if l < 2 || i+l > len(b) {
			break
		}
		switch {
		case t == 4 && l >= 9: // interface
			cur = -1
			if b[i+3] == 0 { // alternate setting 0
				out = append(out, Interface{Number: int(b[i+2]), Class: b[i+5], Sub: b[i+6]})
				cur = len(out) - 1
			}
		case t == 5 && l >= 7 && cur >= 0 && b[i+3]&3 == 2: // bulk endpoint
			if a := b[i+2]; a&0x80 != 0 {
				out[cur].BulkIn = append(out[cur].BulkIn, a)
			} else {
				out[cur].BulkOut = append(out[cur].BulkOut, a)
			}
		}
		i += l
	}
	return out
}

// Device is an open USB device with one claimed interface (more with ClaimExtra).
type Device struct {
	h     *C.tkUSB
	iface int
	Info  Info
	mu    sync.Mutex
}

func ioErr(kr int64, what string) error {
	switch uint32(kr) {
	case 0xE00002C5: // kIOReturnExclusiveAccess
		return fmt.Errorf("%s: another app or a macOS driver is using it (quit the device's own software and replug it)", what)
	case 0xE00002E2, 0xE00002C0: // not permitted / no device
		return fmt.Errorf("%s: the device was unplugged or isn't available", what)
	case 0xE00002F0: // kIOReturnNotFound
		return fmt.Errorf("%s: it doesn't have that interface", what)
	}
	return fmt.Errorf("%s: IOKit error 0x%08X", what, uint32(kr))
}

// Open opens a device ("usb:<locationID>") and claims an interface.
func Open(path string, iface int) (*Device, error) {
	hex, ok := strings.CutPrefix(path, "usb:")
	if !ok {
		return nil, fmt.Errorf("not a USB device name: %s", path)
	}
	loc, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return nil, err
	}
	var e C.int
	h := C.tk_open(C.longlong(loc), C.int(iface), &e)
	if h == nil {
		return nil, ioErr(int64(uint32(e)), "couldn't open the USB device")
	}
	d := &Device{h: h, iface: iface}
	for _, in := range List() {
		if in.Path == path {
			d.Info = in
		}
	}
	return d, nil
}

// ClaimExtra claims another interface too.
func (d *Device) ClaimExtra(iface int) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if iface == d.iface {
		return nil
	}
	if kr := C.tk_claim(d.h, C.int(iface)); kr != 0 {
		return ioErr(int64(uint32(kr)), fmt.Sprintf("couldn't claim USB interface %d", iface))
	}
	return nil
}

// Interface returns the claimed interface's details.
func (d *Device) Interface() Interface {
	for _, it := range d.Info.Interfaces {
		if it.Number == d.iface {
			return it
		}
	}
	return Interface{Number: d.iface}
}

// Control does a control transfer. The direction comes from rt's top bit.
func (d *Device) Control(rt, req uint8, val, idx uint16, data []byte, timeoutMs int) (int, error) {
	var p unsafe.Pointer
	if len(data) > 0 {
		p = C.CBytes(data)
		defer C.free(p)
	}
	n := int64(C.tk_control(d.h, C.int(rt), C.int(req), C.int(val), C.int(idx), p, C.int(len(data)), C.int(timeoutMs)))
	if n < 0 {
		return 0, ioErr(-n, "USB control transfer failed")
	}
	if rt&0x80 != 0 && n > 0 {
		copy(data, unsafe.Slice((*byte)(p), n))
	}
	return int(n), nil
}

func (d *Device) bulk(ep uint8, data []byte, timeoutMs int) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}
	p := C.malloc(C.size_t(len(data)))
	defer C.free(p)
	if ep&0x80 == 0 {
		copy(unsafe.Slice((*byte)(p), len(data)), data)
	}
	n := int64(C.tk_bulk(d.h, C.int(ep), p, C.int(len(data)), C.int(timeoutMs)))
	if n < 0 {
		return 0, ioErr(-n, "USB transfer failed")
	}
	if ep&0x80 != 0 {
		copy(data, unsafe.Slice((*byte)(p), n))
	}
	return int(n), nil
}

// BulkRead reads from a bulk IN endpoint; (0, nil) on timeout.
func (d *Device) BulkRead(ep uint8, buf []byte, timeoutMs int) (int, error) {
	return d.bulk(ep|0x80, buf, timeoutMs)
}

// BulkWrite writes to a bulk OUT endpoint.
func (d *Device) BulkWrite(ep uint8, data []byte, timeoutMs int) (int, error) {
	return d.bulk(ep&0x7F, data, timeoutMs)
}

// ResetPipe clears a stalled endpoint.
func (d *Device) ResetPipe(ep uint8) { C.tk_reset_pipe(d.h, C.int(ep)) }

// Stream reads size-byte chunks from ep until fn returns false or stop
// closes. On a Mac the reads run back to back from one goroutine (count is
// how many are allowed to be in flight on Linux; IOKit's synchronous reads
// go one at a time), with a second goroutine handing chunks to fn so the
// next read starts straight away.
func (d *Device) Stream(ep uint8, size, count int, stop <-chan struct{}, fn func([]byte) bool) error {
	chunks := make(chan []byte, max(count, 2))
	done := make(chan struct{})
	var ferr error
	go func() {
		defer close(done)
		for b := range chunks {
			if !fn(b) {
				for range chunks {
				}
				return
			}
		}
	}()
	defer func() {
		close(chunks)
		<-done
	}()
	for {
		select {
		case <-stop:
			return ferr
		case <-done:
			return ferr
		default:
		}
		buf := make([]byte, size)
		n, err := d.BulkRead(ep, buf, 1000)
		if err != nil {
			return err
		}
		if n == 0 {
			continue
		}
		select {
		case chunks <- buf[:n]:
		case <-done:
			return ferr
		case <-time.After(2 * time.Second):
			return errors.New("the data couldn't be handled fast enough")
		}
	}
}

// Close releases the device.
func (d *Device) Close() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.h != nil {
		C.tk_close(d.h)
		d.h = nil
	}
}
