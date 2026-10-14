//go:build windows && arm64

package win32

import (
	"syscall"
	"unsafe"
)

// gdipFloatThunkAddr is the address of gdipFloatThunk (gdip_call_windows_arm64.s), set by the assembler.
var gdipFloatThunkAddr uintptr

// thunkBlock is what gdipFloatThunk loads: the target, x0-x7 and d0-d7.
type thunkBlock struct {
	fn uintptr
	x  [8]uintptr
	d  [8]uintptr
}

// gdipCall splits the arguments into integer and float registers by the AAPCS64 rule (separate counters).
// Ceiling: 8 integer and 8 float arguments, no stack arguments; GDI+ flat calls used here fit.
func gdipCall(fn uintptr, args []uintptr) uintptr {
	hasFloat := false
	for _, a := range args {
		if uint64(a)>>32 == f32Tag {
			hasFloat = true
			break
		}
	}
	if !hasFloat {
		r, _, _ := syscall.SyscallN(fn, args...)
		return r
	}
	var b thunkBlock
	b.fn = fn
	nx, nd := 0, 0
	for _, a := range args {
		if uint64(a)>>32 == f32Tag {
			b.d[nd] = a & 0xFFFFFFFF
			nd++
		} else {
			b.x[nx] = a
			nx++
		}
	}
	r, _, _ := syscall.SyscallN(gdipFloatThunkAddr, uintptr(unsafe.Pointer(&b)))
	return r
}
