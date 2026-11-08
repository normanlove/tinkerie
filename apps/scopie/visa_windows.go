//go:build windows

// An installed VISA library (NI-VISA, Keysight IO Libraries, R&S VISA ...)
// used directly through its DLL. It is optional: Scopie never ships VISA,
// but when the scope's own software installed it, USB bench scopes work
// without changing their driver.
package scopie

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"unsafe"
)

const (
	viSuccess         = 0
	viSuccessTermChar = 0x3FFF0005
	viSuccessMaxCnt   = 0x3FFF0006
	viAttrTmoValue    = 0x3FFF001A
	viAttrTermcharEn  = 0x3FFF0038
	viErrorTmo        = -1073807339 // 0xBFFF0015
)

type visaLib struct {
	openRM, findRsrc, findNext, open, close, setAttr, write, read *syscall.Proc
	rm                                                            uint32
}

var visaOnce sync.Once
var visaL *visaLib

func loadVisa() *visaLib {
	visaOnce.Do(func() {
		sys := os.Getenv("SystemRoot")
		var names []string
		if unsafe.Sizeof(uintptr(0)) == 8 {
			names = []string{filepath.Join(sys, "System32", "visa64.dll"), filepath.Join(sys, "System32", "visa32.dll")}
		} else {
			names = []string{filepath.Join(sys, "System32", "visa32.dll"), filepath.Join(sys, "SysWOW64", "visa32.dll")}
		}
		for _, n := range names {
			if _, err := os.Stat(n); err != nil {
				continue
			}
			d, err := loadDLL(n)
			if err != nil {
				continue
			}
			l := &visaLib{}
			ok := true
			for name, p := range map[string]**syscall.Proc{
				"viOpenDefaultRM": &l.openRM, "viFindRsrc": &l.findRsrc, "viFindNext": &l.findNext,
				"viOpen": &l.open, "viClose": &l.close, "viSetAttribute": &l.setAttr,
				"viWrite": &l.write, "viRead": &l.read,
			} {
				pr, err := d.FindProc(name)
				if err != nil {
					ok = false
					break
				}
				*p = pr
			}
			if !ok {
				continue
			}
			if st, _, _ := l.openRM.Call(uintptr(unsafe.Pointer(&l.rm))); int32(st) < 0 {
				continue
			}
			visaL = l
			return
		}
	})
	return visaL
}

// visaFindUSB lists the USB instruments the VISA library knows about.
func visaFindUSB() []string {
	l := loadVisa()
	if l == nil {
		return nil
	}
	expr := cstr("USB?*::INSTR")
	var list, cnt uint32
	desc := make([]byte, 512)
	st, _, _ := l.findRsrc.Call(uintptr(l.rm), uintptr(unsafe.Pointer(&expr[0])), uintptr(unsafe.Pointer(&list)),
		uintptr(unsafe.Pointer(&cnt)), uintptr(unsafe.Pointer(&desc[0])))
	if int32(st) < 0 || cnt == 0 {
		return nil
	}
	out := []string{goStr(desc)}
	for i := uint32(1); i < cnt; i++ {
		if st, _, _ := l.findNext.Call(uintptr(list), uintptr(unsafe.Pointer(&desc[0]))); int32(st) < 0 {
			break
		}
		out = append(out, goStr(desc))
	}
	l.close.Call(uintptr(list))
	return out
}

type visaLink struct {
	rsrc string
	vi   uint32
	buf  []byte
}

func (v *visaLink) Name() string { return v.rsrc }

func (v *visaLink) Open() error {
	l := loadVisa()
	if l == nil {
		return errors.New("no VISA library is installed")
	}
	name := cstr(v.rsrc)
	st, _, _ := l.open.Call(uintptr(l.rm), uintptr(unsafe.Pointer(&name[0])), 0, 3000, uintptr(unsafe.Pointer(&v.vi)))
	if int32(st) < 0 {
		return fmt.Errorf("VISA could not open %s (status 0x%08X)", v.rsrc, uint32(st))
	}
	l.setAttr.Call(uintptr(v.vi), viAttrTermcharEn, 0)
	return nil
}

func (v *visaLink) Close() {
	if v.vi != 0 && visaL != nil {
		visaL.close.Call(uintptr(v.vi))
		v.vi = 0
	}
}

func (v *visaLink) Write(b []byte) error {
	if v.vi == 0 {
		return errors.New("not connected")
	}
	var n uint32
	visaL.setAttr.Call(uintptr(v.vi), viAttrTmoValue, 3000)
	st, _, _ := visaL.write.Call(uintptr(v.vi), uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)), uintptr(unsafe.Pointer(&n)))
	if int32(st) < 0 {
		return fmt.Errorf("VISA write failed (status 0x%08X)", uint32(st))
	}
	return nil
}

func (v *visaLink) Read(timeoutMs int) ([]byte, bool, error) {
	if v.vi == 0 {
		return nil, false, errors.New("not connected")
	}
	if v.buf == nil {
		v.buf = make([]byte, 1<<20)
	}
	visaL.setAttr.Call(uintptr(v.vi), viAttrTmoValue, uintptr(timeoutMs))
	var n uint32
	st, _, _ := visaL.read.Call(uintptr(v.vi), uintptr(unsafe.Pointer(&v.buf[0])), uintptr(len(v.buf)), uintptr(unsafe.Pointer(&n)))
	s := int32(st)
	if s < 0 {
		if s == viErrorTmo {
			return nil, false, errors.New("timed out")
		}
		return nil, false, fmt.Errorf("VISA read failed (status 0x%08X)", uint32(st))
	}
	out := append([]byte(nil), v.buf[:n]...)
	return out, s == viSuccess || s == viSuccessTermChar, nil
}
