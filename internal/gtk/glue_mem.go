//go:build linux

package gtk

import (
	"reflect"
	"unsafe"

	"github.com/ebitengine/purego"
)

// at turns a C address held as a Java-style long into a pointer for reading or writing C memory.
func at(p int64) unsafe.Pointer { return unsafe.Add(unsafe.Pointer(nil), uintptr(p)) }

func peek64(p int64) int64    { return *(*int64)(at(p)) }
func peek32(p int64) int32    { return *(*int32)(at(p)) }
func poke64(p int64, v int64) { *(*int64)(at(p)) = v }

// CPTR_SIZEOF is sizeof(void*) on the two supported targets (arm64 and amd64).
const CPTR_SIZEOF = 8

// addrOf is the memory address behind an argument of the memmove family: an int64 C address, a
// slice or a pointer to a Go struct.
func addrOf(x any) uintptr {
	switch v := x.(type) {
	case int64:
		return uintptr(v)
	case int:
		return uintptr(v)
	case uintptr:
		return v
	case int32:
		return uintptr(v)
	}
	rv := reflect.ValueOf(x)
	switch rv.Kind() {
	case reflect.Slice, reflect.Pointer, reflect.UnsafePointer:
		return rv.Pointer()
	}
	panic("gowt/internal/gtk: memmove operand of type " + rv.Type().String())
}

// memmove is C memmove over the operands of any of the Memmove overloads SWT's Java binding had:
// one Go function replaces them all because the operand kind picks the address.
func memmove(dst, src any, size ...int64) {
	var n int64
	if len(size) > 0 {
		n = size[0]
	} else {
		n = structSize(dst, src)
	}
	if n <= 0 {
		return
	}
	d, s := addrOf(dst), addrOf(src)
	copy(unsafe.Slice((*byte)(at(int64(d))), n), unsafe.Slice((*byte)(at(int64(s))), n))
	if u, ok := dst.(interface{ unpack() }); ok {
		u.unpack()
	}
}

// structSize is the size of the Go struct a size-less memmove call moves.
func structSize(ops ...any) int64 {
	for _, x := range ops {
		if t := reflect.TypeOf(x); t.Kind() == reflect.Pointer {
			return int64(t.Elem().Size())
		}
	}
	panic("gowt/internal/gtk: memmove without a size needs a struct pointer")
}

func cstrlen(p int64) int64 {
	n := int64(0)
	for *(*byte)(at(p + n)) != 0 {
		n++
	}
	return n
}

// goBytes copies n bytes of C memory (n < 0: up to the NUL).
func goBytes(p, n int64) []byte {
	if p == 0 {
		return nil
	}
	if n < 0 {
		n = cstrlen(p)
	}
	return append([]byte(nil), unsafe.Slice((*byte)(at(p)), n)...)
}

// cCall calls a C function pointer with integer/pointer arguments.
func cCall(fn int64, args ...uintptr) int64 {
	r, _, _ := purego.SyscallN(uintptr(fn), args...)
	return int64(r)
}

// OSCallProc* and OSCallFunction* invoke a native function pointer (SWT's OS.call overloads).
func OSCallProc(fn, a0, a1 int64) int64 { return cCall(fn, uintptr(a0), uintptr(a1)) }
func OSCallProcII(fn, a0 int64, nfds, timeout int32) int64 {
	return cCall(fn, uintptr(a0), uintptr(nfds), uintptr(timeout))
}
func OSCallFunctionArg0Arg1Arg2Arg3(fn, a0, a1, a2, a3 int64) int64 {
	return cCall(fn, uintptr(a0), uintptr(a1), uintptr(a2), uintptr(a3))
}
func OSCallFunctionArg0Arg1Arg2Arg3Arg4Arg5(fn, a0, a1, a2, a3, a4, a5 int64) int64 {
	return cCall(fn, uintptr(a0), uintptr(a1), uintptr(a2), uintptr(a3), uintptr(a4), uintptr(a5))
}
