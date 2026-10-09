//go:build windows

// Replacement for org.eclipse.swt.internal.Callback's native half (callback.c): all-word arguments and
// result, like a window procedure. syscall.NewCallback slots are never freed (a pool of 2000).
package win32

import (
	"reflect"
	"syscall"
)

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
	case 9:
		f = func(a0, a1, a2, a3, a4, a5, a6, a7, a8 uintptr) uintptr { return w(a0, a1, a2, a3, a4, a5, a6, a7, a8) }
	case 10:
		f = func(a0, a1, a2, a3, a4, a5, a6, a7, a8, a9 uintptr) uintptr {
			return w(a0, a1, a2, a3, a4, a5, a6, a7, a8, a9)
		}
	case 11:
		f = func(a0, a1, a2, a3, a4, a5, a6, a7, a8, a9, a10 uintptr) uintptr {
			return w(a0, a1, a2, a3, a4, a5, a6, a7, a8, a9, a10)
		}
	case 12:
		f = func(a0, a1, a2, a3, a4, a5, a6, a7, a8, a9, a10, a11 uintptr) uintptr {
			return w(a0, a1, a2, a3, a4, a5, a6, a7, a8, a9, a10, a11)
		}
	case 13:
		f = func(a0, a1, a2, a3, a4, a5, a6, a7, a8, a9, a10, a11, a12 uintptr) uintptr {
			return w(a0, a1, a2, a3, a4, a5, a6, a7, a8, a9, a10, a11, a12)
		}
	default:
		panic("gowt/internal/win32: unsupported callback arg count")
	}
	return int64(syscall.NewCallback(f))
}

// NewCallbackAny is a callback whose Go func has typed integer arguments (int32, int64, ...) and a result or none;
// syscall.NewCallback wants exactly one uintptr result, so a func without one is wrapped.
func NewCallbackAny(fn any) int64 {
	v := reflect.ValueOf(fn)
	t := v.Type()
	if t.NumOut() == 1 {
		return int64(syscall.NewCallback(fn))
	}
	in := make([]reflect.Type, t.NumIn())
	for i := range in {
		in[i] = t.In(i)
	}
	wrapped := reflect.MakeFunc(reflect.FuncOf(in, []reflect.Type{reflect.TypeOf(uintptr(0))}, false), func(args []reflect.Value) []reflect.Value {
		v.Call(args)
		return []reflect.Value{reflect.ValueOf(uintptr(0))}
	})
	return int64(syscall.NewCallback(wrapped.Interface()))
}
