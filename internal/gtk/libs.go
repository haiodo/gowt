//go:build linux

// Package gtk binds GTK 3, GLib, Pango and cairo through purego (no cgo): gen_*.go come from GIR
// via tooling/girgen, glue_*.go are hand-written from the public GTK/GLib documentation.
package gtk

import (
	"fmt"
	"reflect"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

// Opened lazily and RTLD_GLOBAL, so libgtk-3's dependency closure resolves with one handle; package
// vars in the glue call into C before init order would reach an eagerly opened list.
var (
	libsOnce sync.Once
	libsList []uintptr
)

func libs() []uintptr {
	libsOnce.Do(func() { libsList = openLibs() })
	return libsList
}

func openLibs() []uintptr {
	var hs []uintptr
	for _, n := range []string{"libgtk-3.so.0", "libgdk-3.so.0", "libX11.so.6", "libcairo.so.2", "libpangocairo-1.0.so.0", "libfontconfig.so.1"} {
		if h, err := purego.Dlopen(n, purego.RTLD_NOW|purego.RTLD_GLOBAL); err == nil {
			hs = append(hs, h)
		}
	}
	if len(hs) == 0 {
		panic("gowt/internal/gtk: libgtk-3.so.0 not found")
	}
	return hs
}

func sym(name string) (uintptr, bool) {
	for _, h := range libs() {
		if a, err := purego.Dlsym(h, name); err == nil && a != 0 {
			return a, true
		}
	}
	return 0, false
}

// lz is a C function bound on first use, so symbols missing from this GTK (the GTK 4 only ones)
// cost nothing until called; calling one panics.
type lz[F any] struct {
	sym string
	f   F
	ok  bool
}

func (l *lz[F]) get() F {
	if l.ok {
		return l.f
	}
	if a, found := sym(l.sym); found {
		purego.RegisterFunc(&l.f, a)
	} else {
		name := l.sym
		reflect.ValueOf(&l.f).Elem().Set(reflect.MakeFunc(reflect.TypeOf(l.f), func([]reflect.Value) []reflect.Value {
			panic(fmt.Sprintf("gowt/internal/gtk: %s is not in the loaded GTK", name))
		}))
	}
	l.ok = true
	return l.f
}

func b2i(b bool) int32 {
	if b {
		return 1
	}
	return 0
}

func sp[T any](s []T) unsafe.Pointer { return unsafe.Pointer(unsafe.SliceData(s)) }

// cs is a NUL-terminated C string as SWT's byte[] constants.
func cs(s string) []int8 {
	b := make([]int8, len(s)+1)
	for i := 0; i < len(s); i++ {
		b[i] = int8(s[i])
	}
	return b
}

var typeFns = map[string]int64{}

// gtypeOf calls a <x>_get_type() function by symbol name, caching the result.
func gtypeOf(symName string) int64 {
	if t, ok := typeFns[symName]; ok {
		return t
	}
	a, ok := sym(symName)
	if !ok {
		panic("gowt/internal/gtk: " + symName + " is not in the loaded GTK")
	}
	var f func() uintptr
	purego.RegisterFunc(&f, a)
	t := int64(f())
	typeFns[symName] = t
	return t
}

var gTypeCheckInstanceIsA = lz[func(uintptr, uintptr) int32]{sym: "g_type_check_instance_is_a"}

func isA(obj, gtype int64) bool {
	return obj != 0 && gTypeCheckInstanceIsA.get()(uintptr(obj), uintptr(gtype)) != 0
}

// anyp converts a C string argument given as a Go string, a C-string byte array, nil or a raw
// address; the generated wrappers use it where call sites mix these.
func anyp(x any) unsafe.Pointer {
	switch v := x.(type) {
	case nil:
		return nil
	case string:
		return sp(utf8z(v, true))
	case []int8:
		return sp(v)
	case []byte:
		return sp(v)
	case int64:
		return at(v)
	}
	panic(fmt.Sprintf("gowt/internal/gtk: %T is not a C string argument", x))
}
