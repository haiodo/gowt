//go:build windows

package fluent

import (
	"syscall"
	"unsafe"

	"github.com/haiodo/gowt/swt"
)

var pFindWindowEx = user32.NewProc("FindWindowExW")

const emSetRectMsg = 0xB3

// SpinnerCenter centres the line in the edit of a taller Spinner; the edit resets its formatting
// rectangle on resize, hence the listener.
func SpinnerCenter(s *swt.Spinner) {
	center := func() {
		sh := uintptr(s.Handle)
		class, _ := syscall.UTF16PtrFromString("Edit")
		e := call(pFindWindowEx, sh, 0, uintptr(unsafe.Pointer(class)), 0)
		if e == 0 {
			return
		}
		var cr rect
		call(pGetClientRect, e, uintptr(unsafe.Pointer(&cr)))
		top := (cr.bottom - textHeight(e)) / 2
		if top <= 0 {
			return
		}
		cr.top += top
		call(pSendMessage, e, emSetRectMsg, 0, uintptr(unsafe.Pointer(&cr)))
	}
	s.AddControlListener(swt.ControlListenerControlResizedAdapter(func(*swt.ControlEvent) { center() }))
	center()
}
