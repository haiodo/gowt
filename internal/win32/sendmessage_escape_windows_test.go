//go:build windows

package win32

import (
	"testing"
	"unsafe"
)

// A window procedure that fills the buffer SendMessage was given, after it has grown the stack.
func TestSendMessageSurvivesStackGrowth(t *testing.T) {
	onFreshThread(func() {
		hwnd := testWindow(t, hwndMessage, func(msg uint32, wp, lp uintptr) (uintptr, bool) {
			switch msg {
			case testWMUser + 5:
				grow()
				*(*[4]int32)(unsafe.Add(unsafe.Pointer(nil), lp)) = [4]int32{1, 2, 3, 4}
				return 1, true
			case testWMUser + 6:
				grow()
				*(*[2]uint16)(unsafe.Add(unsafe.Pointer(nil), lp)) = [2]uint16{'o', 'k'}
				return 1, true
			}
			return 0, false
		})
		defer procDestroyWindowT.Call(hwnd)
		var rc RECT
		OSSendMessageOverload13(int64(hwnd), testWMUser+5, 0, &rc)
		if rc != (RECT{Left: 1, Top: 2, Right: 3, Bottom: 4}) {
			t.Errorf("rect = %+v: the window procedure wrote through a stale address", rc)
		}
		buf := make([]uint16, 2)
		OSSendMessageOverload2(int64(hwnd), testWMUser+6, 0, buf)
		if buf[0] != 'o' || buf[1] != 'k' {
			t.Errorf("text = %v: the window procedure wrote through a stale address", buf)
		}
	})
}
