//go:build darwin

// Hand-written minimal ObjC runtime layer for the WebKit binding (webkit_manual.go) and blocks
// (block_manual.go): untyped message sends, selector cache, NSString/NSData conversion, runtime classes.
package cocoa

import (
	"runtime"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

var (
	objcOnce      sync.Once
	objcMsgSend   uintptr
	selRegister   func(name string) uintptr
	getClass      func(name string) uintptr
	getProtocol   func(name string) uintptr
	allocClass    func(super uintptr, name string, extra uintptr) uintptr
	registerClass func(cls uintptr)
	addMethod     func(cls, sel, imp uintptr, types string) bool
	addProtocol   func(cls, proto uintptr) bool
	msgRectOnly   func(self, sel uintptr, r NSRect)
	selCache      = map[string]uintptr{}
)

func objcInit() {
	objcOnce.Do(func() {
		ensureFrameworks()
		objcMsgSend, _ = purego.Dlsym(purego.RTLD_DEFAULT, "objc_msgSend")
		purego.RegisterLibFunc(&selRegister, purego.RTLD_DEFAULT, "sel_registerName")
		purego.RegisterLibFunc(&getClass, purego.RTLD_DEFAULT, "objc_getClass")
		purego.RegisterLibFunc(&getProtocol, purego.RTLD_DEFAULT, "objc_getProtocol")
		purego.RegisterLibFunc(&allocClass, purego.RTLD_DEFAULT, "objc_allocateClassPair")
		purego.RegisterLibFunc(&registerClass, purego.RTLD_DEFAULT, "objc_registerClassPair")
		purego.RegisterLibFunc(&addMethod, purego.RTLD_DEFAULT, "class_addMethod")
		purego.RegisterLibFunc(&addProtocol, purego.RTLD_DEFAULT, "class_addProtocol")
		purego.RegisterFunc(&msgRectOnly, objcMsgSend)
	})
}

// sel caches selector lookups; all callers run on the UI thread.
func sel(name string) uintptr {
	objcInit()
	s, ok := selCache[name]
	if !ok {
		s = selRegister(name)
		selCache[name] = s
	}
	return s
}

// msg sends an ObjC message with word-sized arguments and a word-sized (or void) result.
func msg(recv uintptr, name string, args ...uintptr) uintptr {
	objcInit()
	r, _, _ := purego.SyscallN(objcMsgSend, append([]uintptr{recv, sel(name)}, args...)...)
	return r
}

func class(name string) uintptr {
	objcInit()
	return getClass(name)
}

// nsString copies s into a new autoreleased NSString (bytes+length keeps embedded NULs).
func nsString(s string) uintptr {
	b := []byte(s)
	p := uintptr(0)
	if len(b) > 0 {
		p = uintptr(unsafe.Pointer(&b[0]))
	}
	o := msg(msg(class("NSString"), "alloc"), "initWithBytes:length:encoding:", p, uintptr(len(b)), 4)
	runtime.KeepAlive(b)
	return msg(o, "autorelease")
}

func nsData(b []byte) uintptr {
	p := uintptr(0)
	if len(b) > 0 {
		p = uintptr(unsafe.Pointer(&b[0]))
	}
	d := msg(class("NSData"), "dataWithBytes:length:", p, uintptr(len(b)))
	runtime.KeepAlive(b)
	return d
}

// goString reads an NSString (0 gives "").
func goString(s uintptr) string {
	if s == 0 {
		return ""
	}
	p := msg(s, "UTF8String")
	if p == 0 {
		return ""
	}
	return string(unsafe.Slice((*byte)(unsafe.Add(unsafe.Pointer(nil), p)), cstrlen(p)))
}

func cstrlen(p uintptr) int {
	n := 0
	for *(*byte)(unsafe.Add(unsafe.Pointer(nil), p+uintptr(n))) != 0 {
		n++
	}
	return n
}

type objcMethod struct {
	imp   uintptr
	types string
}

// newClass creates and registers a subclass of NSObject with the given IMPs (types are ObjC encodings).
func newClass(name string, protocols []string, methods map[string]objcMethod) uintptr {
	objcInit()
	cls := allocClass(class("NSObject"), name, 0)
	for sel, m := range methods {
		addMethod(cls, selRegister(sel), m.imp, m.types)
	}
	for _, p := range protocols {
		if proto := getProtocol(p); proto != 0 {
			addProtocol(cls, proto)
		}
	}
	registerClass(cls)
	return cls
}
