//go:build windows

// Replacement for org.eclipse.swt.internal.Callback's native half (callback.c): all-word arguments and
// result, like a window procedure. syscall.NewCallback slots are never freed (a pool of 2000).
package win32

import "syscall"

// NewCallbackN returns the C address of a stdcall function taking argCount words, which calls fn.
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
	case 7:
		f = func(a0, a1, a2, a3, a4, a5, a6 uintptr) uintptr { return w(a0, a1, a2, a3, a4, a5, a6) }
	case 8:
		f = func(a0, a1, a2, a3, a4, a5, a6, a7 uintptr) uintptr { return w(a0, a1, a2, a3, a4, a5, a6, a7) }
	default:
		panic("gowt/internal/win32: unsupported callback arg count")
	}
	return int64(syscall.NewCallback(f))
}
