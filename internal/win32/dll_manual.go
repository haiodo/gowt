//go:build windows

// Hand-written half of the Win32 bindings: lazy DLL procs behind the generated natives (see
// tooling/j2go/README.md "Round 20 (win32)").
package win32

import (
	"sync"
	"syscall"
	"unsafe"
)

// Search order of the system DLLs whose exports the natives name (os.c's import libraries).
var dllNames = []string{
	"user32.dll", "gdi32.dll", "kernel32.dll", "comctl32.dll", "comdlg32.dll", "shell32.dll",
	"ole32.dll", "oleaut32.dll", "uxtheme.dll", "imm32.dll", "msimg32.dll", "advapi32.dll",
	"usp10.dll", "dwmapi.dll", "shlwapi.dll", "oleacc.dll", "wininet.dll", "propsys.dll",
	"urlmon.dll", "winspool.drv", "gdiplus.dll", "ucrtbase.dll", "ntdll.dll",
}

var dlls = func() []*syscall.LazyDLL {
	l := make([]*syscall.LazyDLL, len(dllNames))
	for i, n := range dllNames {
		l[i] = syscall.NewLazyDLL(n)
	}
	return l
}()

// Proc is one exported function, found in the first DLL that has it (the W variant first: SWT
// builds with UNICODE, where CreateWindowEx is a macro for CreateWindowExW).
type Proc struct {
	name    string
	dynamic bool // missing is fine (an OS newer than the running one): the call returns 0
	once    sync.Once
	ptr     uintptr
}

func newProc(name string, dynamic ...bool) *Proc {
	return &Proc{name: name, dynamic: len(dynamic) > 0 && dynamic[0]}
}

// Lookup returns the address of name, or 0.
func Lookup(name string) uintptr {
	for _, suffix := range []string{"W", ""} {
		for _, d := range dlls {
			p := d.NewProc(name + suffix)
			if p.Find() == nil {
				return p.Addr()
			}
		}
	}
	return 0
}

var missing uintptr

// A stand-in for an absent dynamic proc: returns 0 to the caller.
func init() { missing = syscall.NewCallback(func() uintptr { return 0 }) }

func (p *Proc) addr() uintptr {
	p.once.Do(func() {
		p.ptr = Lookup(p.name)
		if p.ptr == 0 && p.dynamic {
			p.ptr = missing
		}
	})
	if p.ptr == 0 {
		panic("win32: no DLL exports " + p.name)
	}
	return p.ptr
}

func boolToUintptr(b bool) uintptr {
	if b {
		return 1
	}
	return 0
}

// structArg passes a C struct by value: up to 8 bytes in the register, larger ones by reference (Win64).
func structArg(p unsafe.Pointer, size uintptr) uintptr {
	switch size {
	case 1:
		return uintptr(*(*uint8)(p))
	case 2:
		return uintptr(*(*uint16)(p))
	case 4:
		return uintptr(*(*uint32)(p))
	case 8:
		return uintptr(*(*uint64)(p))
	}
	return uintptr(p)
}

// vtblFn is slot fn of the COM object at address obj (its first word points to the vtable).
func vtblFn(obj int64, fn int32) uintptr {
	vtbl := *(*uintptr)(unsafe.Add(unsafe.Pointer(nil), uintptr(obj)))
	return *(*uintptr)(unsafe.Add(unsafe.Pointer(nil), vtbl+uintptr(fn)*unsafe.Sizeof(uintptr(0))))
}
