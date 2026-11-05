//go:build darwin

// Hand-written: one lazy C-symbol binding per generated native (see tooling/j2go/README.md
// "Native methods"). The generated wrappers used to carry a private `_impl func` + `_once
// sync.Once` pair each; NativeEmitter now declares one `lazyFunc[func(...)...]` per native and
// calls `get("<symbol>")(<args>)`, so sync/purego stay in this file instead of every part file.

package cocoa

import (
	"sync"

	"github.com/ebitengine/purego"
)

// lazyFunc binds a C symbol lazily on first call: ensureFrameworks() then
// purego.RegisterLibFunc, so a missing/wrong symbol panics when that native is
// actually invoked, not at package init.
type lazyFunc[T any] struct {
	once sync.Once
	impl T
}

// get returns the bound function. Call it on a package-level (or otherwise
// long-lived) variable only: a lazyFunc must not be copied after first use.
func (l *lazyFunc[T]) get(symbol string) T {
	l.once.Do(func() {
		ensureFrameworks()
		purego.RegisterLibFunc(&l.impl, purego.RTLD_DEFAULT, symbol)
	})
	return l.impl
}
