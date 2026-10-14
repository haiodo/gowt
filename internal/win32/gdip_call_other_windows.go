//go:build windows && !arm64

package win32

import "syscall"

// gdipCall: on x64 the first four arguments reach both the integer and the float registers, so floats go in as bits.
func gdipCall(fn uintptr, args []uintptr) uintptr {
	for i, a := range args {
		if uint64(a)>>32 == f32Tag {
			args[i] = a & 0xFFFFFFFF
		}
	}
	r, _, _ := syscall.SyscallN(fn, args...)
	return r
}
