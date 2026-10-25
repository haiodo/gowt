//go:build windows && arm64

package d2d

import (
	"runtime"
	"syscall"
	"unsafe"
)

// floatThunkAddr is the address of floatThunk (call_windows_arm64.s), set by the assembler.
var floatThunkAddr uintptr

// thunkBlock is what floatThunk loads: the target, x0-x7 and d0-d7.
type thunkBlock struct {
	fn uintptr
	x  [8]uintptr
	d  [8]uintptr
}

// call splits the arguments into integer and float registers (AAPCS64: separate counters; a 2 float
// struct takes two float registers). Ceiling: with a float, at most 8 integer and 8 float arguments.
func call(fn uintptr, args ...arg) uintptr {
	hasFloat := false
	for _, a := range args {
		if a.kind != kInt {
			hasFloat = true
			break
		}
	}
	if !hasFloat {
		var raw [16]uintptr
		for i, a := range args {
			raw[i] = a.v
		}
		r, _, _ := syscall.SyscallN(fn, raw[:len(args)]...)
		runtime.KeepAlive(args)
		return r
	}
	var b thunkBlock
	b.fn = fn
	nx, nd := 0, 0
	for _, a := range args {
		switch a.kind {
		case kInt:
			b.x[nx] = a.v
			nx++
		case kF32:
			b.d[nd] = a.v & 0xFFFFFFFF
			nd++
		case kPt2:
			b.d[nd] = a.v & 0xFFFFFFFF
			b.d[nd+1] = a.v >> 32
			nd += 2
		}
	}
	r, _, _ := syscall.SyscallN(floatThunkAddr, uintptr(unsafe.Pointer(&b)))
	runtime.KeepAlive(args)
	runtime.KeepAlive(&b)
	return r
}
