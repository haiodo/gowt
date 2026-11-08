//go:build darwin && gowtcgo && cgo

// Hand-written companion of lazyfunc_manual.go for the opt-in cgo backend: the generated
// native wrappers carry a unique binding name next to the C symbol, so get(name, symbol)
// resolves the name once against the generated registry of typed C wrappers
// (<unit>_cgo.go, emitted by j2go with this build tag) and the steady-state call is the
// same indirect call as purego's - no per-call backend branching.

package cocoa

/*
#include <stdint.h>
*/
import "C"

import "unsafe"

// cgoNativeRegistry is filled by the generated <unit>_cgo.go func init()s, one entry per
// lazy native; get type-asserts the entry to the native's own func type.
var cgoNativeRegistry = map[string]any{}

func (l *lazyFunc[T]) get(name, symbol string) T {
	l.once.Do(func() {
		ensureFrameworks()
		l.impl = cgoNativeRegistry[name].(T)
	})
	return l.impl
}

// cgoBool converts a Go bool to the 1-byte C bool every generated C wrapper takes.
func cgoBool(b bool) C.uint8_t {
	if b {
		return 1
	}
	return 0
}

// cgoStruct reinterprets a value between the Go struct a native's signature uses and its
// layout-identical generated C twin (gowt_ typedef): the compiler rejects a plain
// conversion because cgo's field types/tags differ, the memory is the same.
func cgoStruct[T, U any](v U) T {
	return *(*T)(unsafe.Pointer(&v))
}
