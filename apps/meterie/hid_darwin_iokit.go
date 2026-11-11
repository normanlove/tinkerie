//go:build darwin && cgo

package meterie

// hid_darwin_iokit.go: USB-HID meter cables on macOS through IOKit's HID
// manager. Input reports arrive on a run loop in a thread of their own and
// wait in a queue for readReport. Devices are named "hid:<registry ID>".

/*
#cgo LDFLAGS: -framework IOKit -framework CoreFoundation
#include <stdlib.h>
#include <string.h>
#include <pthread.h>
#include <CoreFoundation/CoreFoundation.h>
#include <IOKit/IOKitLib.h>
#include <IOKit/hid/IOHIDManager.h>
#include <IOKit/hid/IOHIDDevice.h>

#define TKQ 256
typedef struct {
	IOHIDDeviceRef dev;
	pthread_t th;
	CFRunLoopRef rl;
	pthread_mutex_t mu;
	pthread_cond_t cv;
	int ready, gone;
	unsigned char in[1024];
	unsigned char q[TKQ][130];
	int qlen[TKQ], head, tail;
} tkHID;

static void tk_in(void *ctx, IOReturn r, void *sender, IOHIDReportType type, uint32_t id, uint8_t *rep, CFIndex len) {
	tkHID *h = ctx;
	pthread_mutex_lock(&h->mu);
	int next = (h->tail + 1) % TKQ;
	if (next != h->head) { // full: drop the newest
		unsigned char *d = h->q[h->tail];
		int n = 0;
		// Windows' layout: the report ID first (0 when reports aren't numbered)
		if (id == 0 || len == 0 || rep[0] != id) d[n++] = (unsigned char)id;
		if (len > 128 - n) len = 128 - n;
		memcpy(d + n, rep, len);
		h->qlen[h->tail] = n + len;
		h->tail = next;
	}
	pthread_cond_signal(&h->cv);
	pthread_mutex_unlock(&h->mu);
}

static void tk_removed(void *ctx, IOReturn r, void *sender) {
	tkHID *h = ctx;
	pthread_mutex_lock(&h->mu);
	h->gone = 1;
	pthread_cond_signal(&h->cv);
	pthread_mutex_unlock(&h->mu);
}

static void *tk_loop(void *arg) {
	tkHID *h = arg;
	pthread_mutex_lock(&h->mu);
	h->rl = CFRunLoopGetCurrent();
	IOHIDDeviceRegisterInputReportCallback(h->dev, h->in, sizeof h->in, tk_in, h);
	IOHIDDeviceRegisterRemovalCallback(h->dev, tk_removed, h);
	IOHIDDeviceScheduleWithRunLoop(h->dev, h->rl, kCFRunLoopDefaultMode);
	h->ready = 1;
	pthread_cond_broadcast(&h->cv);
	pthread_mutex_unlock(&h->mu);
	CFRunLoopRun();
	return NULL;
}

static IOHIDDeviceRef tk_find(unsigned long long id) {
	io_service_t s = IOServiceGetMatchingService(0, IORegistryEntryIDMatching(id));
	if (!s) return NULL;
	IOHIDDeviceRef d = IOHIDDeviceCreate(kCFAllocatorDefault, s);
	IOObjectRelease(s);
	return d;
}

static tkHID *tk_hid_open(unsigned long long id, int *err) {
	IOHIDDeviceRef d = tk_find(id);
	if (!d) { *err = kIOReturnNoDevice; return NULL; }
	IOReturn kr = IOHIDDeviceOpen(d, kIOHIDOptionsTypeNone);
	if (kr != kIOReturnSuccess) { CFRelease(d); *err = kr; return NULL; }
	tkHID *h = calloc(1, sizeof(tkHID));
	h->dev = d;
	pthread_mutex_init(&h->mu, NULL);
	pthread_cond_init(&h->cv, NULL);
	pthread_create(&h->th, NULL, tk_loop, h);
	pthread_mutex_lock(&h->mu);
	while (!h->ready) pthread_cond_wait(&h->cv, &h->mu);
	pthread_mutex_unlock(&h->mu);
	*err = 0;
	return h;
}

// tk_hid_read: bytes copied, 0 on timeout, -1 if the device went away.
static int tk_hid_read(tkHID *h, unsigned char *buf, int cap, int ms) {
	struct timespec ts;
	clock_gettime(CLOCK_REALTIME, &ts);
	ts.tv_sec += ms / 1000;
	ts.tv_nsec += (long)(ms % 1000) * 1000000;
	if (ts.tv_nsec >= 1000000000) { ts.tv_sec++; ts.tv_nsec -= 1000000000; }
	pthread_mutex_lock(&h->mu);
	while (h->head == h->tail && !h->gone) {
		if (pthread_cond_timedwait(&h->cv, &h->mu, &ts) != 0) break;
	}
	int n = 0;
	if (h->head != h->tail) {
		n = h->qlen[h->head];
		if (n > cap) n = cap;
		memcpy(buf, h->q[h->head], n);
		h->head = (h->head + 1) % TKQ;
	} else if (h->gone) n = -1;
	pthread_mutex_unlock(&h->mu);
	return n;
}

// tk_hid_set sends an output (type 1) or feature (type 2) report; data[0] is
// the report ID, as on Windows, and isn't sent when it's 0.
static int tk_hid_set(tkHID *h, int type, unsigned char *data, int len) {
	if (len < 1) return kIOReturnBadArgument;
	CFIndex id = data[0];
	unsigned char *p = data;
	if (id == 0) { p++; len--; }
	return IOHIDDeviceSetReport(h->dev, type == 2 ? kIOHIDReportTypeFeature : kIOHIDReportTypeOutput, id, p, len);
}

static void tk_hid_close(tkHID *h) {
	IOHIDDeviceUnscheduleFromRunLoop(h->dev, h->rl, kCFRunLoopDefaultMode);
	CFRunLoopStop(h->rl);
	pthread_join(h->th, NULL);
	IOHIDDeviceClose(h->dev, kIOHIDOptionsTypeNone);
	CFRelease(h->dev);
	free(h);
}

// ---------------------------------------------------------------- listing
static CFArrayRef gHIDs = NULL;
static int tk_hid_list(void) {
	if (gHIDs) { CFRelease(gHIDs); gHIDs = NULL; }
	IOHIDManagerRef m = IOHIDManagerCreate(kCFAllocatorDefault, kIOHIDOptionsTypeNone);
	IOHIDManagerSetDeviceMatching(m, NULL);
	CFSetRef set = IOHIDManagerCopyDevices(m);
	int n = 0;
	if (set) {
		n = CFSetGetCount(set);
		const void **vals = malloc(sizeof(void *) * (n ? n : 1));
		CFSetGetValues(set, vals);
		gHIDs = CFArrayCreate(kCFAllocatorDefault, vals, n, &kCFTypeArrayCallBacks);
		free(vals);
		CFRelease(set);
	}
	CFRelease(m);
	return n;
}

static long long tk_hid_num(int i, CFStringRef key) {
	IOHIDDeviceRef d = (IOHIDDeviceRef)CFArrayGetValueAtIndex(gHIDs, i);
	CFTypeRef v = IOHIDDeviceGetProperty(d, key);
	long long n = -1;
	if (v && CFGetTypeID(v) == CFNumberGetTypeID()) CFNumberGetValue((CFNumberRef)v, kCFNumberLongLongType, &n);
	return n;
}

static void tk_hid_info(int i, long long *vid, long long *pid, unsigned long long *id, char *name, int cap) {
	IOHIDDeviceRef d = (IOHIDDeviceRef)CFArrayGetValueAtIndex(gHIDs, i);
	*vid = tk_hid_num(i, CFSTR(kIOHIDVendorIDKey));
	*pid = tk_hid_num(i, CFSTR(kIOHIDProductIDKey));
	uint64_t eid = 0;
	IORegistryEntryGetRegistryEntryID(IOHIDDeviceGetService(d), &eid);
	*id = eid;
	name[0] = 0;
	CFTypeRef v = IOHIDDeviceGetProperty(d, CFSTR(kIOHIDProductKey));
	if (v && CFGetTypeID(v) == CFStringGetTypeID()) CFStringGetCString((CFStringRef)v, name, cap, kCFStringEncodingUTF8);
}
*/
import "C"

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"unsafe"
)

type macHID struct {
	h              *C.tkHID
	outLen, featLn int
}

func openHIDDev(path string) (hidHandle, error) {
	id, err := strconv.ParseUint(strings.TrimPrefix(path, "hid:"), 16, 64)
	if err != nil {
		return nil, fmt.Errorf("not a HID device name: %s", path)
	}
	var e C.int
	h := C.tk_hid_open(C.ulonglong(id), &e)
	if h == nil {
		if uint32(e) == 0xE00002C5 {
			return nil, fmt.Errorf("can't open the USB cable: another app is using it (is the meter's own software running?)")
		}
		return nil, fmt.Errorf("can't open the USB cable (IOKit error 0x%08X)", uint32(e))
	}
	return &macHID{h: h}, nil
}

func (d *macHID) readReport(timeoutMs uint32) ([]byte, error) {
	buf := make([]byte, 130)
	n := C.tk_hid_read(d.h, (*C.uchar)(unsafe.Pointer(&buf[0])), 130, C.int(timeoutMs))
	if n < 0 {
		return nil, fmt.Errorf("the USB cable was unplugged")
	}
	if n == 0 {
		return nil, nil
	}
	return buf[:n], nil
}

func (d *macHID) set(kind int, b []byte, pad int) error {
	buf := make([]byte, max(pad, len(b)))
	copy(buf, b)
	p := C.CBytes(buf)
	defer C.free(p)
	if kr := C.tk_hid_set(d.h, C.int(kind), (*C.uchar)(p), C.int(len(buf))); kr != 0 {
		return fmt.Errorf("USB report failed (IOKit error 0x%08X)", uint32(kr))
	}
	return nil
}

func (d *macHID) writeReport(b []byte) error { return d.set(1, b, d.outLen) }
func (d *macHID) setFeature(b []byte) error  { return d.set(2, b, d.featLn) }
func (d *macHID) defaultLens(out, feat int) {
	if d.outLen == 0 {
		d.outLen = out
	}
	if d.featLn == 0 {
		d.featLn = feat
	}
}
func (d *macHID) forceFeatLen(n int) { d.featLn = n }
func (d *macHID) close()             { C.tk_hid_close(d.h) }

var hidListMu sync.Mutex

// listHID finds known meter cables among the Mac's HID devices.
func listHID() []PortInfo {
	hidListMu.Lock()
	defer hidListMu.Unlock()
	n := int(C.tk_hid_list())
	name := (*C.char)(C.malloc(256))
	defer C.free(unsafe.Pointer(name))
	var out []PortInfo
	seen := map[uint64]bool{}
	for i := 0; i < n; i++ {
		var vid, pid C.longlong
		var id C.ulonglong
		C.tk_hid_info(C.int(i), &vid, &pid, &id, name, 256)
		if vid < 0 || seen[uint64(id)] {
			continue
		}
		seen[uint64(id)] = true
		var kind Transport = -1
		for _, k := range []Transport{TrCH9325, TrCP2110, TrVictor, TrBU86X} {
			if u := lookupUSB(uint16(vid), uint16(pid), k); u != nil && u.Kind == k {
				kind = k
			}
		}
		if kind < 0 {
			continue
		}
		pi := PortInfo{Kind: kind, Path: fmt.Sprintf("hid:%x", uint64(id)), Name: C.GoString(name), VID: uint16(vid), PID: uint16(pid)}
		classifyPort(&pi)
		if s := C.GoString(name); s != "" {
			pi.Name = fmt.Sprintf("%s (%04X:%04X)", s, uint16(vid), uint16(pid))
		}
		out = append(out, pi)
	}
	return out
}
