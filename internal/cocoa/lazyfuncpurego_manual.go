//go:build darwin && (!gowtcgo || !cgo)

// Hand-written: the default backend's lazy binding - purego. The generated wrappers call
// get(name, symbol); this build ignores name and binds symbol through purego. The opt-in
// cgo backend (darwin && cgo && gowtcgo, see lazyfunccgo_manual.go) ignores symbol and
// resolves name against the generated registry of typed C wrappers (<unit>_cgo.go).

package cocoa

import (
	"github.com/ebitengine/purego"
)

// get returns the bound function. Call it on a package-level (or otherwise
// long-lived) variable only: a lazyFunc must not be copied after first use.
func (l *lazyFunc[T]) get(name, symbol string) T {
	l.once.Do(func() {
		ensureFrameworks()
		purego.RegisterLibFunc(&l.impl, purego.RTLD_DEFAULT, symbol)
	})
	return l.impl
}
