//go:build darwin && cgo

package meterie

// ble_darwin.go: Bluetooth LE meters on macOS through CoreBluetooth. Macs
// don't reveal Bluetooth addresses, so a meter is known by the identifier
// macOS gives it (shown as its Path), turned into a number for BLEAddr.

/*
#cgo CFLAGS: -x objective-c -fobjc-arc -mmacosx-version-min=11.0
#cgo LDFLAGS: -framework CoreBluetooth -framework Foundation
#import <Foundation/Foundation.h>
#import <CoreBluetooth/CoreBluetooth.h>
#include <pthread.h>
#include <stdlib.h>
#include <string.h>

#define TKQ 256
#define TKMAX 8
typedef struct {
	pthread_mutex_t mu;
	pthread_cond_t cv;
	unsigned char q[TKQ][256];
	int qlen[TKQ], head, tail, gone, used;
} tkBleQ;
static tkBleQ tkq[TKMAX];

@interface TKBle : NSObject <CBCentralManagerDelegate, CBPeripheralDelegate>
@property (strong) CBCentralManager *cm;
@property (strong) dispatch_queue_t q;
@property (strong) NSMutableDictionary *seen, *rssi, *names; // by identifier
@property (strong) NSMutableArray *conns;                     // CBPeripheral or NSNull, by handle-1
@property (strong) dispatch_semaphore_t ready, step;
@property (assign) int stepErr, pendingSvcs, gotState;
@property (strong) CBPeripheral *cur;
@property (strong) CBUUID *wantSvc, *wantChr;
@property (strong) CBCharacteristic *chosen;
@end

@implementation TKBle
- (instancetype)init {
	self = [super init];
	_q = dispatch_queue_create("tinkerie.ble", DISPATCH_QUEUE_SERIAL);
	_seen = [NSMutableDictionary new]; _rssi = [NSMutableDictionary new]; _names = [NSMutableDictionary new];
	_conns = [NSMutableArray new];
	for (int i = 0; i < TKMAX; i++) [_conns addObject:[NSNull null]];
	_ready = dispatch_semaphore_create(0);
	_step = dispatch_semaphore_create(0);
	_cm = [[CBCentralManager alloc] initWithDelegate:self queue:_q];
	return self;
}
- (void)centralManagerDidUpdateState:(CBCentralManager *)c {
	if (!_gotState) { _gotState = 1; dispatch_semaphore_signal(_ready); }
}
- (void)centralManager:(CBCentralManager *)c didDiscoverPeripheral:(CBPeripheral *)p advertisementData:(NSDictionary *)ad RSSI:(NSNumber *)r {
	NSString *k = p.identifier.UUIDString;
	_seen[k] = p;
	_rssi[k] = r;
	NSString *n = ad[CBAdvertisementDataLocalNameKey] ?: p.name;
	if (n.length) _names[k] = n;
}
- (void)centralManager:(CBCentralManager *)c didConnectPeripheral:(CBPeripheral *)p {
	if (p == _cur) { _stepErr = 0; dispatch_semaphore_signal(_step); }
}
- (void)centralManager:(CBCentralManager *)c didFailToConnectPeripheral:(CBPeripheral *)p error:(NSError *)e {
	if (p == _cur) { _stepErr = 1; dispatch_semaphore_signal(_step); }
}
- (void)centralManager:(CBCentralManager *)c didDisconnectPeripheral:(CBPeripheral *)p error:(NSError *)e {
	for (int i = 0; i < TKMAX; i++) if (_conns[i] == p) {
		pthread_mutex_lock(&tkq[i].mu); tkq[i].gone = 1; pthread_cond_signal(&tkq[i].cv); pthread_mutex_unlock(&tkq[i].mu);
	}
	if (p == _cur) { _stepErr = 2; dispatch_semaphore_signal(_step); }
}
- (void)peripheral:(CBPeripheral *)p didDiscoverServices:(NSError *)e {
	if (e || p.services.count == 0) { _stepErr = 3; dispatch_semaphore_signal(_step); return; }
	_pendingSvcs = (int)p.services.count;
	for (CBService *s in p.services) [p discoverCharacteristics:nil forService:s];
}
- (void)peripheral:(CBPeripheral *)p didDiscoverCharacteristicsForService:(CBService *)s error:(NSError *)e {
	if (!e && (!_wantSvc || [s.UUID isEqual:_wantSvc])) {
		for (CBCharacteristic *c in s.characteristics) {
			if (_wantChr) { if ([c.UUID isEqual:_wantChr]) _chosen = c; }
			else if (!_chosen && (c.properties & (CBCharacteristicPropertyNotify | CBCharacteristicPropertyIndicate))) _chosen = c;
		}
	}
	if (--_pendingSvcs == 0) { _stepErr = _chosen ? 0 : 4; dispatch_semaphore_signal(_step); }
}
- (void)peripheral:(CBPeripheral *)p didUpdateNotificationStateForCharacteristic:(CBCharacteristic *)c error:(NSError *)e {
	if (p == _cur) { _stepErr = e ? 5 : 0; dispatch_semaphore_signal(_step); }
}
- (void)peripheral:(CBPeripheral *)p didUpdateValueForCharacteristic:(CBCharacteristic *)c error:(NSError *)e {
	if (e || !c.value.length) return;
	for (int i = 0; i < TKMAX; i++) if (_conns[i] == p) {
		tkBleQ *q = &tkq[i];
		pthread_mutex_lock(&q->mu);
		int next = (q->tail + 1) % TKQ;
		if (next != q->head) {
			int n = (int)MIN(c.value.length, 256);
			memcpy(q->q[q->tail], c.value.bytes, n);
			q->qlen[q->tail] = n;
			q->tail = next;
		}
		pthread_cond_signal(&q->cv);
		pthread_mutex_unlock(&q->mu);
	}
}
@end

static TKBle *tk = nil;

// tk_ble_init returns CoreBluetooth's state: 5 on, 4 off, 3 not allowed, 2 unsupported.
static int tk_ble_init(void) {
	static dispatch_once_t once;
	dispatch_once(&once, ^{
		for (int i = 0; i < TKMAX; i++) { pthread_mutex_init(&tkq[i].mu, NULL); pthread_cond_init(&tkq[i].cv, NULL); }
		tk = [TKBle new];
		dispatch_semaphore_wait(tk.ready, dispatch_time(DISPATCH_TIME_NOW, 4 * NSEC_PER_SEC));
	});
	__block int st = 0;
	dispatch_sync(tk.q, ^{ st = (int)tk.cm.state; });
	return st;
}

static void tk_ble_scan(int on) {
	dispatch_sync(tk.q, ^{
		if (on) [tk.cm scanForPeripheralsWithServices:nil options:@{CBCentralManagerScanOptionAllowDuplicatesKey: @YES}];
		else [tk.cm stopScan];
	});
}

typedef struct { char ident[40]; char name[64]; int rssi; } tkBleResult;

static int tk_ble_results(tkBleResult *out, int cap) {
	__block int n = 0;
	dispatch_sync(tk.q, ^{
		for (NSString *k in tk.seen) {
			if (n >= cap) break;
			strlcpy(out[n].ident, k.UTF8String, sizeof out[n].ident);
			NSString *nm = tk.names[k];
			strlcpy(out[n].name, nm ? nm.UTF8String : "", sizeof out[n].name);
			out[n].rssi = [tk.rssi[k] intValue];
			n++;
		}
	});
	return n;
}

static int tk_wait(int ms) {
	if (dispatch_semaphore_wait(tk.step, dispatch_time(DISPATCH_TIME_NOW, (int64_t)ms * NSEC_PER_MSEC)) != 0) return -1;
	return tk.stepErr;
}

// tk_ble_connect returns a handle (1..8), or minus an error step.
static int tk_ble_connect(const char *ident, const char *svc, const char *chr) {
	NSString *k = [NSString stringWithUTF8String:ident];
	__block CBPeripheral *p = nil;
	__block int slot = -1;
	dispatch_sync(tk.q, ^{
		p = tk.seen[k];
		if (!p) {
			NSUUID *u = [[NSUUID alloc] initWithUUIDString:k];
			if (u) p = [tk.cm retrievePeripheralsWithIdentifiers:@[u]].firstObject;
		}
		for (int i = 0; i < TKMAX && p; i++) if (tk.conns[i] == [NSNull null]) { slot = i; break; }
		if (p && slot >= 0) {
			tk.cur = p; tk.chosen = nil;
			tk.wantSvc = svc[0] ? [CBUUID UUIDWithString:[NSString stringWithUTF8String:svc]] : nil;
			tk.wantChr = chr[0] ? [CBUUID UUIDWithString:[NSString stringWithUTF8String:chr]] : nil;
			p.delegate = tk;
			[tk.cm connectPeripheral:p options:nil];
		}
	});
	if (!p) return -10;
	if (slot < 0) return -11;
	int e = tk_wait(30000);
	if (e != 0) { dispatch_sync(tk.q, ^{ [tk.cm cancelPeripheralConnection:p]; tk.cur = nil; }); return e < 0 ? -1 : -e; }
	dispatch_sync(tk.q, ^{ [p discoverServices:tk.wantSvc ? @[tk.wantSvc] : nil]; });
	e = tk_wait(20000);
	if (e == 0) {
		dispatch_sync(tk.q, ^{
			tkq[slot].head = tkq[slot].tail = tkq[slot].gone = 0;
			tk.conns[slot] = p;
			[p setNotifyValue:YES forCharacteristic:tk.chosen];
		});
		e = tk_wait(10000);
	}
	dispatch_sync(tk.q, ^{ tk.cur = nil; });
	if (e != 0) {
		dispatch_sync(tk.q, ^{ tk.conns[slot] = [NSNull null]; [tk.cm cancelPeripheralConnection:p]; });
		return e < 0 ? -1 : -e;
	}
	return slot + 1;
}

static int tk_ble_read(int h, unsigned char *buf, int cap, int ms) {
	tkBleQ *q = &tkq[h - 1];
	struct timespec ts;
	clock_gettime(CLOCK_REALTIME, &ts);
	ts.tv_nsec += (long)ms * 1000000;
	ts.tv_sec += ts.tv_nsec / 1000000000;
	ts.tv_nsec %= 1000000000;
	pthread_mutex_lock(&q->mu);
	while (q->head == q->tail && !q->gone) if (pthread_cond_timedwait(&q->cv, &q->mu, &ts) != 0) break;
	int n = 0;
	if (q->head != q->tail) {
		n = q->qlen[q->head] < cap ? q->qlen[q->head] : cap;
		memcpy(buf, q->q[q->head], n);
		q->head = (q->head + 1) % TKQ;
	} else if (q->gone) n = -1;
	pthread_mutex_unlock(&q->mu);
	return n;
}

static void tk_ble_close(int h) {
	dispatch_sync(tk.q, ^{
		id p = tk.conns[h - 1];
		if (p != [NSNull null]) [tk.cm cancelPeripheralConnection:p];
		tk.conns[h - 1] = [NSNull null];
	});
}
*/
import "C"

import (
	"errors"
	"hash/fnv"
	"sync"
	"time"
	"unsafe"
)

var macBLE struct {
	mu  sync.Mutex
	ids map[uint64]string // BLEAddr → macOS identifier
}

func bleAddrOf(id string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(id))
	return h.Sum64() & 0xFFFFFFFFFFFF
}

func bleReady() error {
	switch C.tk_ble_init() {
	case 5:
		return nil
	case 4:
		return errors.New("Bluetooth is switched off: switch it on in Control Centre")
	case 3:
		return errors.New("Tinkerie isn't allowed to use Bluetooth: allow it in System Settings › Privacy & Security › Bluetooth")
	case 2:
		return errors.New("this Mac doesn't have Bluetooth LE")
	}
	return errors.New("Bluetooth isn't ready yet: try again in a moment")
}

func platformBLEScan(on bool) error {
	if err := bleReady(); err != nil {
		if !on {
			return nil
		}
		return err
	}
	C.tk_ble_scan(boolInt(on))
	return nil
}

func boolInt(b bool) C.int {
	if b {
		return 1
	}
	return 0
}

func platformBLEResults() []PortInfo {
	if C.tk_ble_init() != 5 {
		return nil
	}
	res := make([]C.tkBleResult, 128)
	n := int(C.tk_ble_results(&res[0], 128))
	macBLE.mu.Lock()
	defer macBLE.mu.Unlock()
	if macBLE.ids == nil {
		macBLE.ids = map[uint64]string{}
	}
	var out []PortInfo
	for _, r := range res[:n] {
		id, name := C.GoString(&r.ident[0]), C.GoString(&r.name[0])
		addr := bleAddrOf(id)
		macBLE.ids[addr] = id
		p := PortInfo{Kind: TrBLE, Path: id, Name: name, BLEAddr: addr, RSSI: int(r.rssi)}
		if p.Name == "" && bleNameMatch(p.Name) == nil && p.RSSI < -85 {
			continue
		}
		out = append(out, p)
	}
	return out
}

type macBLEPort struct {
	h      C.int
	closed chan struct{}
	once   sync.Once
}

func platformOpenBLE(addr uint64, svc, chr string) (Port, error) {
	if err := bleReady(); err != nil {
		return nil, err
	}
	macBLE.mu.Lock()
	id := macBLE.ids[addr]
	macBLE.mu.Unlock()
	if id == "" { // not seen since Meterie started: look for it
		C.tk_ble_scan(1)
		for i := 0; i < 20 && id == ""; i++ {
			time.Sleep(500 * time.Millisecond)
			platformBLEResults()
			macBLE.mu.Lock()
			id = macBLE.ids[addr]
			macBLE.mu.Unlock()
		}
		C.tk_ble_scan(0)
		if id == "" {
			return nil, errors.New("the meter isn't in range: switch its Bluetooth on and scan again")
		}
	}
	cid, csvc, cchr := C.CString(id), C.CString(svc), C.CString(chr)
	defer C.free(unsafe.Pointer(cid))
	defer C.free(unsafe.Pointer(csvc))
	defer C.free(unsafe.Pointer(cchr))
	h := C.tk_ble_connect(cid, csvc, cchr)
	switch {
	case h > 0:
		return &macBLEPort{h: h, closed: make(chan struct{})}, nil
	case h == -1:
		return nil, errors.New("the meter didn't answer: is it on and close by?")
	case h == -4:
		return nil, errors.New("the meter doesn't offer the expected data service: is it the right model?")
	case h == -11:
		return nil, errors.New("too many Bluetooth meters are connected")
	}
	return nil, errors.New("couldn't connect to the meter")
}

func (p *macBLEPort) Read(b []byte) (int, error) {
	select {
	case <-p.closed:
		return 0, errors.New("Bluetooth connection closed")
	default:
	}
	buf := make([]byte, 256)
	n := int(C.tk_ble_read(p.h, (*C.uchar)(unsafe.Pointer(&buf[0])), 256, 200))
	if n < 0 {
		return 0, errors.New("the meter disconnected")
	}
	return copy(b, buf[:n]), nil
}

func (p *macBLEPort) Write(b []byte) (int, error) { return len(b), nil }

func (p *macBLEPort) Close() error {
	p.once.Do(func() {
		close(p.closed)
		C.tk_ble_close(p.h)
	})
	return nil
}
