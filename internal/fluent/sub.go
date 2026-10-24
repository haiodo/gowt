//go:build windows

package fluent

import "syscall"

// handler sees a message of a subclassed window first. It returns the result and true to end the
// message there; def runs the rest of the chain (SWT's window procedure, then the control's own).
type handler func(hwnd, msg, wp, lp uintptr, def func() uintptr) (uintptr, bool)

const subID = 0x676f7774

// syscall.NewCallback slots are never freed, so one callback serves every window.
var subProc uintptr

func init() { subProc = syscall.NewCallback(subclassProc) }

var handlers = map[uintptr]handler{}

func subclassProc(hwnd, msg, wp, lp, _, _ uintptr) (ret uintptr) {
	def := func() uintptr { return call(pDefSubclassProc, hwnd, msg, wp, lp) }
	if msg == wmNCDestroy {
		delete(handlers, hwnd)
		call(pRemoveWindowSubclass, hwnd, subProc, subID)
		return def()
	}
	h := handlers[hwnd]
	if h == nil {
		return def()
	}
	// A panic must not unwind through native frames.
	defer func() {
		if recover() != nil {
			ret = def()
		}
	}()
	if r, ok := h(hwnd, msg, wp, lp, def); ok {
		return r
	}
	return def()
}

// attach makes h the handler of hwnd, replacing an earlier one.
func attach(hwnd uintptr, h handler) {
	if handlers[hwnd] == nil {
		call(pSetWindowSubclass, hwnd, subProc, subID, 0)
	}
	handlers[hwnd] = h
}
