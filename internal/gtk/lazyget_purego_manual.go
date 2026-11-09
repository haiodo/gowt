//go:build linux && (!gowtcgo || !cgo)

// Hand-written: the default backend's lazy binding - purego (see libs.go for the
// shared lz type and semantics). The opt-in cgo backend (linux && gowtcgo && cgo,
// lazyget_cgo_manual.go) resolves the generated wrappers' names in the registry of
// typed C wrappers; glue natives (empty name) take this purego path in both backends.

package gtk

import (
	"fmt"
	"reflect"

	"github.com/ebitengine/purego"
)

func (l *lz[F]) get() F {
	if l.ok {
		return l.f
	}
	if missing() {
		return zeroFunc[F]()
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
