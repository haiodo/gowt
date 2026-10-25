//go:build windows

package d2d

import (
	"errors"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/haiodo/gowt/gfx"
)

var (
	ole32         = syscall.NewLazyDLL("ole32.dll")
	procCoInit    = ole32.NewProc("CoInitializeEx")
	procCoUninit  = ole32.NewProc("CoUninitialize")
	procCoCreate  = ole32.NewProc("CoCreateInstance")
	clsidWICFact  = mkGUID(0xcacaf262, 0x9370, 0x4615, 0xa1, 0x3b, 0x9f, 0x55, 0x39, 0xda, 0x4c, 0x0a)
	iidWICFact    = mkGUID(0xec5ec8a9, 0xc395, 0x4314, 0x9c, 0x77, 0x54, 0xd7, 0xa9, 0x35, 0xff, 0x70)
	fmtPNG        = mkGUID(0x1b7cfaf4, 0x713f, 0x473c, 0xbb, 0xcd, 0x61, 0x37, 0x42, 0x5f, 0xae, 0xaf)
	pixFmt32BGRA  = mkGUID(0x6fddc324, 0x4e03, 0x4bfe, 0xb1, 0x85, 0x3d, 0x77, 0x76, 0x8d, 0xc9, 0x0f)
	errPNGEncoder = errors.New("gfx/d2d: WIC PNG encoder failed")
)

// EncodePNG encodes r as a PNG with the Windows Imaging Component, 8 bits per channel with alpha.
// It needs no window; it runs COM on the calling thread for the duration of the call.
func EncodePNG(r *gfx.Raster) ([]byte, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	// MULTITHREADED apartment; S_FALSE and RPC_E_CHANGED_MODE mean COM is already usable on this thread
	hr, _, _ := procCoInit.Call(0, 0)
	if !failed(hr) {
		defer procCoUninit.Call()
	}
	var fac uintptr
	hr, _, _ = procCoCreate.Call(uintptr(unsafe.Pointer(clsidWICFact)), 0, 1, uintptr(unsafe.Pointer(iidWICFact)), uintptr(unsafe.Pointer(&fac)))
	if failed(hr) {
		return nil, hrErr("CoCreateInstance(WICImagingFactory)", hr)
	}
	defer release(fac)

	out := make([]byte, len(r.Pix)+64<<10)
	var stream uintptr
	if hr := vcall(fac, 14, ref(&stream)); failed(hr) { // CreateStream
		return nil, hrErr("CreateStream", hr)
	}
	defer release(stream)
	if hr := vcall(stream, 16, buf(out), ui(uintptr(len(out)))); failed(hr) { // InitializeFromMemory
		return nil, hrErr("InitializeFromMemory", hr)
	}
	var enc uintptr
	if hr := vcall(fac, 8, ref(fmtPNG), ui(0), ref(&enc)); failed(hr) { // CreateEncoder
		return nil, hrErr("CreateEncoder", hr)
	}
	defer release(enc)
	if hr := vcall(enc, 3, ui(stream), ui(2)); failed(hr) { // Initialize, WICBitmapEncoderNoCache
		return nil, hrErr("Initialize", hr)
	}
	var frame, bag uintptr
	if hr := vcall(enc, 10, ref(&frame), ref(&bag)); failed(hr) { // CreateNewFrame
		return nil, hrErr("CreateNewFrame", hr)
	}
	defer release(frame)
	defer release(bag)
	if hr := vcall(frame, 3, ui(bag)); failed(hr) { // Initialize
		return nil, hrErr("frame Initialize", hr)
	}
	vcall(frame, 4, ui(uintptr(r.W)), ui(uintptr(r.H))) // SetSize
	pf := *pixFmt32BGRA
	if hr := vcall(frame, 6, ref(&pf)); failed(hr) { // SetPixelFormat
		return nil, hrErr("SetPixelFormat", hr)
	}
	if pf != *pixFmt32BGRA {
		return nil, errPNGEncoder
	}
	bgra := make([]byte, len(r.Pix))
	for i := 0; i+3 < len(r.Pix); i += 4 {
		bgra[i], bgra[i+1], bgra[i+2], bgra[i+3] = r.Pix[i+2], r.Pix[i+1], r.Pix[i], r.Pix[i+3]
	}
	if hr := vcall(frame, 10, ui(uintptr(r.H)), ui(uintptr(r.W*4)), ui(uintptr(len(bgra))), buf(bgra)); failed(hr) { // WritePixels
		return nil, hrErr("WritePixels", hr)
	}
	if hr := vcall(frame, 12); failed(hr) { // Commit
		return nil, hrErr("frame Commit", hr)
	}
	if hr := vcall(enc, 11); failed(hr) { // Commit
		return nil, hrErr("encoder Commit", hr)
	}
	var n uint64
	if hr := vcall(stream, 5, ui(0), ui(1), ref(&n)); failed(hr) { // Seek(0, STREAM_SEEK_CUR)
		return nil, hrErr("Seek", hr)
	}
	return out[:n], nil
}
