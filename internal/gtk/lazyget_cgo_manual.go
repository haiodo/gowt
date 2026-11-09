//go:build linux && gowtcgo && cgo

// Hand-written companion for the opt-in cgo backend (see libs.go for the shared lz
// type and semantics): generated wrappers carry a unique name next to the C symbol,
// resolved once against the generated registry of typed C wrappers (gen_*_cgo.go,
// emitted by girgen with this build tag). Symbol resolution still happens here, so a
// symbol missing from the loaded GTK becomes the same panicking stub as in the
// purego backend. Glue natives (empty name) fall through to purego, so callback
// registration and hand-written bindings behave identically in both backends.

package gtk

import (
	"fmt"
	"reflect"

	"github.com/ebitengine/purego"
)

// cgoNativeRegistry is filled by the generated gen_*_cgo.go package-level
// initializers: one func(uintptr) F constructor per generated native, turning the
// dlsym'd address into the typed wrapper whose steady-state call goes straight to C.
var cgoNativeRegistry = map[string]any{}

func (l *lz[F]) get() F {
	if l.ok {
		return l.f
	}
	if missing() {
		return zeroFunc[F]()
	}
	if mk, ok := cgoNativeRegistry[l.name]; ok {
		if a, found := sym(l.sym); found {
			l.f = mk.(func(uintptr) F)(a)
		} else {
			name := l.sym
			reflect.ValueOf(&l.f).Elem().Set(reflect.MakeFunc(reflect.TypeOf(l.f), func([]reflect.Value) []reflect.Value {
				panic(fmt.Sprintf("gowt/internal/gtk: %s is not in the loaded GTK", name))
			}))
		}
		l.ok = true
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
