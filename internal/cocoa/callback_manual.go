// Hand-written replacement for org.eclipse.swt.internal.Callback's native half (callback.c) and
// for os.c's by-value-struct trampolines (see tooling/j2go/README.md "Callback design").
package cocoa

import (
	"runtime"
	"unsafe"

	"github.com/ebitengine/purego"
)

// NewCallback registers fn as a native trampoline (e.g. an ObjC IMP, self/_cmd uintptr first)
// and returns its C function pointer. Go's static typing replaces Callback+reflection.
func NewCallback(fn any) uintptr {
	return purego.NewCallback(fn)
}

// Generic callbacks by their C address: OS.CALLBACK_x(func) wraps one of these, not a C pointer.
var callbacks = map[int64]func(args []int64) int64{}

// NewCallbackN is callback.c's all-word-args trampoline: argCount uintptr args, uintptr return.
func NewCallbackN(argCount int, fn func(args []int64) int64) int64 {
	w := func(a ...uintptr) uintptr {
		args := make([]int64, len(a))
		for i, v := range a {
			args[i] = int64(v)
		}
		return uintptr(fn(args))
	}
	var f any
	switch argCount {
	case 0:
		f = func() uintptr { return w() }
	case 1:
		f = func(a0 uintptr) uintptr { return w(a0) }
	case 2:
		f = func(a0, a1 uintptr) uintptr { return w(a0, a1) }
	case 3:
		f = func(a0, a1, a2 uintptr) uintptr { return w(a0, a1, a2) }
	case 4:
		f = func(a0, a1, a2, a3 uintptr) uintptr { return w(a0, a1, a2, a3) }
	case 5:
		f = func(a0, a1, a2, a3, a4 uintptr) uintptr { return w(a0, a1, a2, a3, a4) }
	case 6:
		f = func(a0, a1, a2, a3, a4, a5 uintptr) uintptr { return w(a0, a1, a2, a3, a4, a5) }
	default:
		panic("gowt/internal/cocoa: unsupported callback arg count")
	}
	addr := int64(NewCallback(f))
	callbacks[addr] = fn
	return addr
}

func callbackFunc(addr int64) func(args []int64) int64 {
	fn := callbacks[addr]
	if fn == nil {
		panic("gowt/internal/cocoa: CALLBACK_ wrapper for an unknown callback")
	}
	return fn
}

// pinArg hands a by-value struct parameter's address to the generic callback as a long, like
// os.c's &arg: pinned so neither GC nor a stack move invalidates it during the call.
func pinArg[T any](pin *runtime.Pinner, v *T) int64 {
	pin.Pin(v)
	return int64(uintptr(unsafe.Pointer(v)))
}

// structResult copies a struct the generic callback returned as a C.malloc'd pointer (Display's
// windowProc does that for cellSize & co.) and frees it, as os.c's trampolines do.
func structResult[T any](ptr int64) T {
	var v T
	if ptr == 0 {
		return v
	}
	v = *(*T)(unsafe.Add(unsafe.Pointer(nil), uintptr(ptr)))
	CFree(ptr)
	return v
}

// OSIsFlipped_CALLBACK is os_custom.c's isFlippedProc: an IMP that always answers YES.
func OSIsFlipped_CALLBACK() int64 {
	return int64(NewCallback(func(id, sel uintptr) uintptr { return 1 }))
}

// OSCall is os.c's `((void (*)())proc)(id, sel)` (Display.cursorSetProc chains to the original
// -[NSCursor set] this way).
func OSCall(proc int64, id int64, sel int64) {
	purego.SyscallN(uintptr(proc), uintptr(id), uintptr(sel))
}
