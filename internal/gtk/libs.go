//go:build linux

// Package gtk binds GTK 3, GLib, Pango and cairo through purego (no cgo): gen_*.go come from GIR
// via tooling/girgen, glue_*.go are hand-written from the public GTK/GLib documentation.
package gtk

import (
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/haiodo/gowt/internal/jrt"
)

// Opened lazily and RTLD_GLOBAL, so libgtk-3's dependency closure resolves with one handle. Importing
// the package must not fail without GTK: the generated static initialisers of swt call into C at
// import, and gowt.Run returns the load error instead of the process dying before main.
var (
	libsOnce sync.Once
	libsList []uintptr
	loadErr  error
	libNames = []string{"libgtk-3.so.0", "libgdk-3.so.0", "libX11.so.6", "libcairo.so.2", "libpangocairo-1.0.so.0", "libfontconfig.so.1"}
)

func libs() []uintptr {
	libsOnce.Do(func() {
		libsList, loadErr = openLibs()
		if loadErr != nil {
			return
		}
		major, minor, micro := gtkVersionPart(libsList, "gtk_get_major_version"),
			gtkVersionPart(libsList, "gtk_get_minor_version"), gtkVersionPart(libsList, "gtk_get_micro_version")
		GTKGTK_VERSION = OSVERSION(major, minor, micro)
		jrt.SetProperty("org.eclipse.swt.internal.gtk.version", fmt.Sprintf("%d.%d.%d", major, minor, micro))
	})
	return libsList
}

func openLibs() ([]uintptr, error) {
	var hs []uintptr
	var firstErr error
	for _, n := range libNames {
		h, err := purego.Dlopen(n, purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		hs = append(hs, h)
	}
	if len(hs) == 0 {
		// A library must not exit the process: gowt.Run returns this as an error, plain swt users see it in the panic.
		return nil, fmt.Errorf("GTK 3 is not installed (%w); install it: apt install libgtk-3-0 | dnf install gtk3 | pacman -S gtk3 (glibc distributions only)", firstErr)
	}
	return hs, nil
}

// initDone is set by swt's last init(); until then GTK calls come from package initialisation.
var initDone atomic.Bool

// MarkInitDone ends the import-time phase: from now on a missing GTK panics instead of returning zeros.
func MarkInitDone() { initDone.Store(true) }

// missing reports that GTK could not be loaded. During package initialisation it returns true, so the
// static initialisers see zero values; afterwards it panics with the load error, which is what
// swt.NewDisplay and gowt.Run surface.
func missing() bool {
	if len(libs()) > 0 {
		return false
	}
	if initDone.Load() {
		panic(loadErr)
	}
	return true
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
	if missing() {
		var f F
		t := reflect.TypeOf(f)
		reflect.ValueOf(&f).Elem().Set(reflect.MakeFunc(t, func([]reflect.Value) []reflect.Value {
			out := make([]reflect.Value, t.NumOut())
			for i := range out {
				out[i] = reflect.Zero(t.Out(i))
			}
			return out
		}))
		return f
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
	if missing() {
		return 0
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
