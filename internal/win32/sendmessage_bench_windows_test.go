//go:build windows

package win32

import (
	"runtime"
	"syscall"
	"testing"
	"unsafe"
)

// BenchmarkSendMessageMix is a message mix of a UI: 8 of 10 calls carry no pointer, one a text buffer,
// one a RECT out parameter.
func BenchmarkSendMessageMix(b *testing.B) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	cls, _ := syscall.UTF16PtrFromString("EDIT")
	r, _, err := procCreateWindow.Call(0, uintptr(unsafe.Pointer(cls)), 0, 0, 0, 0, 100, 20, hwndMessage, 0, 0, 0)
	if r == 0 {
		b.Skipf("no EDIT control: %v", err)
	}
	hwnd := int64(r)
	defer procDestroyWindowT.Call(r)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for range 8 {
			OSSendMessageOverload4(hwnd, 0x000E, 0, 0) // WM_GETTEXTLENGTH
		}
		buf := make([]uint16, 64)
		OSSendMessageOverload2(hwnd, 0x000D, 64, buf) // WM_GETTEXT
		var rc RECT
		OSSendMessageOverload13(hwnd, 0x00B2, 0, &rc) // EM_GETRECT
	}
}
