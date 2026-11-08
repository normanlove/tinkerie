//go:build windows

package meterie

// camera_windows.go: webcam capture through Media Foundation (mfplat / mf /
// mfreadwrite — all part of Windows), pure Go COM calls.

import (
	"errors"
	"fmt"
	"image"
	"runtime"
	"syscall"
	"unsafe"
)

var (
	mfplat      = sysDLL("mfplat.dll")
	mfdll       = sysDLL("mf.dll")
	mfreadwrite = sysDLL("mfreadwrite.dll")

	pCoInitializeEx       = ole32.NewProc("CoInitializeEx")
	pMFStartup            = mfplat.NewProc("MFStartup")
	pMFCreateAttributes   = mfplat.NewProc("MFCreateAttributes")
	pMFCreateMediaType    = mfplat.NewProc("MFCreateMediaType")
	pMFEnumDeviceSources  = mfdll.NewProc("MFEnumDeviceSources")
	pMFCreateSourceReader = mfreadwrite.NewProc("MFCreateSourceReaderFromMediaSource")
)

var (
	mfSourceType      = parseGUID("c60ac5fe-252a-478f-a0ef-bc8fa5f7cad3")
	mfSourceVidcap    = parseGUID("8ac3587a-4ae7-42d8-99e0-0a6013eef90f")
	mfFriendlyName    = parseGUID("60d0e559-52f8-4fa2-bbce-acdb34a8ec01")
	mfEnableVP        = parseGUID("fb394f3d-ccf1-42ee-bbb3-f9b845d5681d")
	mfMajorType       = parseGUID("48eba18e-f8c9-4687-bf11-0a74c9f96a8f")
	mfSubtype         = parseGUID("f7e34c9a-42e8-4714-b74b-cb29d72c35e5")
	mfVideo           = parseGUID("73646976-0000-0010-8000-00AA00389B71")
	mfRGB32           = parseGUID("00000016-0000-0010-8000-00AA00389B71")
	mfFrameSize       = parseGUID("1652c33d-d6b2-4012-b834-72030849a37d")
	mfDefaultStride   = parseGUID("644b4e48-1e02-4516-b0eb-c01ca9d49ac6")
	iidIMFMediaSource = parseGUID("279a808d-aec7-40c8-9c6b-a6b492c78a66")
)

const mfFirstVideo = 0xFFFFFFFC

func lockThread() { runtime.LockOSThread() }

// mfThread runs Media Foundation work on one COM-initialised OS thread.
type mfThread struct{ ch chan func() }

func newMFThread() (*mfThread, error) {
	t := &mfThread{ch: make(chan func())}
	ready := make(chan error)
	go func() {
		runtime.LockOSThread()
		pCoInitializeEx.Call(0, 0) // COINIT_MULTITHREADED
		if err := mfplat.Load(); err != nil {
			ready <- errors.New("Media Foundation isn't installed (Windows 'N' editions need the Media Feature Pack)")
			return
		}
		if hr, _, _ := pMFStartup.Call(0x00020070, 0); failed(hr) {
			ready <- fmt.Errorf("MFStartup failed (0x%08X)", uint32(hr))
			return
		}
		ready <- nil
		for f := range t.ch {
			f()
		}
	}()
	if err := <-ready; err != nil {
		return nil, err
	}
	return t, nil
}

func (t *mfThread) do(f func()) {
	done := make(chan struct{})
	t.ch <- func() {
		defer func() { recover(); close(done) }()
		f()
	}
	<-done
}

var mfShared *mfThread

func mfEnum() (uintptr, uint32, error) {
	var attr uintptr
	if hr, _, _ := pMFCreateAttributes.Call(uintptr(unsafe.Pointer(&attr)), 1); failed(hr) {
		return 0, 0, fmt.Errorf("MFCreateAttributes 0x%08X", uint32(hr))
	}
	defer release(attr)
	vcall(attr, 24, uintptr(unsafe.Pointer(&mfSourceType)), uintptr(unsafe.Pointer(&mfSourceVidcap))) // SetGUID
	var arr uintptr
	var n uint32
	if hr, _, _ := pMFEnumDeviceSources.Call(attr, uintptr(unsafe.Pointer(&arr)), uintptr(unsafe.Pointer(&n))); failed(hr) {
		return 0, 0, fmt.Errorf("no cameras (0x%08X)", uint32(hr))
	}
	return arr, n, nil
}

func platformListCameras() []string {
	if mfShared == nil {
		t, err := newMFThread()
		if err != nil {
			return nil
		}
		mfShared = t
	}
	var names []string
	mfShared.do(func() {
		arr, n, err := mfEnum()
		if err != nil {
			return
		}
		acts := unsafe.Slice((*uintptr)(unsafe.Pointer(arr)), n)
		for _, act := range acts {
			var p *uint16
			var l uint32
			name := "Camera"
			if hr := vcall(act, 13, uintptr(unsafe.Pointer(&mfFriendlyName)), uintptr(unsafe.Pointer(&p)), uintptr(unsafe.Pointer(&l))); !failed(hr) && p != nil {
				name = syscall.UTF16ToString(unsafe.Slice(p, l))
				pCoTaskMemFree.Call(uintptr(unsafe.Pointer(p)))
			}
			names = append(names, name)
			release(act)
		}
		pCoTaskMemFree.Call(arr)
	})
	return names
}

type mfCamera struct {
	t      *mfThread
	source uintptr
	reader uintptr
	w, h   int
	stride int
}

func platformOpenCamera(idx int) (FrameSource, error) {
	if mfShared == nil {
		t, err := newMFThread()
		if err != nil {
			return nil, err
		}
		mfShared = t
	}
	c := &mfCamera{t: mfShared}
	var outErr error
	c.t.do(func() { outErr = c.open(idx) })
	if outErr != nil {
		c.Close()
		return nil, outErr
	}
	return c, nil
}

func (c *mfCamera) open(idx int) error {
	arr, n, err := mfEnum()
	if err != nil {
		return err
	}
	acts := unsafe.Slice((*uintptr)(unsafe.Pointer(arr)), n)
	defer func() {
		for _, a := range acts {
			release(a)
		}
		pCoTaskMemFree.Call(arr)
	}()
	if idx < 0 || idx >= int(n) {
		return errors.New("no webcam found — try Phone / IP or a photo instead")
	}
	if hr := vcall(acts[idx], 33, uintptr(unsafe.Pointer(&iidIMFMediaSource)), uintptr(unsafe.Pointer(&c.source))); failed(hr) {
		return fmt.Errorf("can't open the camera (0x%08X) — is another app using it? Also check Windows camera privacy settings", uint32(hr))
	}
	var ra uintptr
	pMFCreateAttributes.Call(uintptr(unsafe.Pointer(&ra)), 1)
	vcall(ra, 21, uintptr(unsafe.Pointer(&mfEnableVP)), 1) // SetUINT32(ENABLE_VIDEO_PROCESSING, TRUE)
	hr, _, _ := pMFCreateSourceReader.Call(c.source, ra, uintptr(unsafe.Pointer(&c.reader)))
	release(ra)
	if failed(hr) {
		return fmt.Errorf("camera reader failed (0x%08X)", uint32(hr))
	}
	var mt uintptr
	pMFCreateMediaType.Call(uintptr(unsafe.Pointer(&mt)))
	vcall(mt, 24, uintptr(unsafe.Pointer(&mfMajorType)), uintptr(unsafe.Pointer(&mfVideo)))
	vcall(mt, 24, uintptr(unsafe.Pointer(&mfSubtype)), uintptr(unsafe.Pointer(&mfRGB32)))
	hr = vcall(c.reader, 7, mfFirstVideo, 0, mt) // SetCurrentMediaType
	release(mt)
	if failed(hr) {
		return fmt.Errorf("camera format not supported (0x%08X)", uint32(hr))
	}
	var cur uintptr
	if hr := vcall(c.reader, 6, mfFirstVideo, uintptr(unsafe.Pointer(&cur))); !failed(hr) && cur != 0 {
		var fs uint64
		vcall(cur, 8, uintptr(unsafe.Pointer(&mfFrameSize)), uintptr(unsafe.Pointer(&fs)))
		c.w, c.h = int(fs>>32), int(fs&0xFFFFFFFF)
		var st uint32
		if hr := vcall(cur, 7, uintptr(unsafe.Pointer(&mfDefaultStride)), uintptr(unsafe.Pointer(&st))); !failed(hr) {
			c.stride = int(int32(st))
		} else {
			c.stride = -c.w * 4 // RGB defaults to bottom-up
		}
		release(cur)
	}
	if c.w == 0 || c.h == 0 {
		return errors.New("camera reported no frame size")
	}
	return nil
}

func (c *mfCamera) Grab() (*image.Gray, error) {
	var img *image.Gray
	var outErr error
	c.t.do(func() {
		for tries := 0; tries < 10 && img == nil; tries++ {
			var idx, flags uint32
			var ts int64
			var sample uintptr
			hr := vcall(c.reader, 9, mfFirstVideo, 0, uintptr(unsafe.Pointer(&idx)), uintptr(unsafe.Pointer(&flags)), uintptr(unsafe.Pointer(&ts)), uintptr(unsafe.Pointer(&sample)))
			if failed(hr) {
				outErr = fmt.Errorf("camera read failed (0x%08X)", uint32(hr))
				return
			}
			if flags&0x2 != 0 { // end of stream
				outErr = errors.New("camera stopped")
				return
			}
			if sample == 0 {
				continue
			}
			var buf uintptr
			vcall(sample, 41, uintptr(unsafe.Pointer(&buf))) // ConvertToContiguousBuffer
			if buf != 0 {
				var p uintptr
				var maxLen, curLen uint32
				if hr := vcall(buf, 3, uintptr(unsafe.Pointer(&p)), uintptr(unsafe.Pointer(&maxLen)), uintptr(unsafe.Pointer(&curLen))); !failed(hr) && p != 0 {
					img = c.convert(unsafe.Slice((*byte)(unsafe.Pointer(p)), curLen))
					vcall(buf, 4)
				}
				release(buf)
			}
			release(sample)
		}
	})
	if img == nil && outErr == nil {
		outErr = errors.New("no frame from the camera")
	}
	return img, outErr
}

func (c *mfCamera) convert(b []byte) *image.Gray {
	w, h := c.w, c.h
	stride := c.stride
	abs := stride
	if abs < 0 {
		abs = -abs
	}
	if abs < w*4 {
		abs = w * 4
	}
	if len(b) < abs*(h-1)+w*4 {
		return nil
	}
	g := image.NewGray(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		sy := y
		if stride < 0 {
			sy = h - 1 - y
		}
		row := b[sy*abs:]
		o := g.Pix[y*g.Stride:]
		for x := 0; x < w; x++ {
			i := x * 4
			o[x] = uint8((uint32(row[i])*114 + uint32(row[i+1])*587 + uint32(row[i+2])*299) / 1000)
		}
	}
	return g
}

func (c *mfCamera) Close() {
	c.t.do(func() {
		release(c.reader)
		c.reader = 0
		if c.source != 0 {
			vcall(c.source, 12) // IMFMediaSource::Shutdown
			release(c.source)
			c.source = 0
		}
	})
}
