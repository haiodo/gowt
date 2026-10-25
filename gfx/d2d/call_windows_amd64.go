//go:build windows && amd64

package d2d

import (
	"runtime"
	"syscall"
)

// call invokes fn with the arguments. On x64 the first four arguments reach both the integer and the
// float registers, so a float goes in as its bits and a 2 float struct as one 8 byte slot.
func call(fn uintptr, args ...arg) uintptr {
	var raw [16]uintptr
	for i, a := range args {
		raw[i] = a.v
	}
	r, _, _ := syscall.SyscallN(fn, raw[:len(args)]...)
	runtime.KeepAlive(args)
	return r
}
