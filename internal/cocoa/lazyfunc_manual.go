//go:build darwin

// Hand-written: the lazy binding shared by both native backends (see
// lazyfuncpurego_manual.go for the default purego get and lazyfunccgo_manual.go for the
// opt-in cgo backend, build tag gowtcgo). Generated native wrappers carry one lazyFunc[T]
// per native and call get("<name>", "<symbol>") on it (tooling/j2go/README.md
// "Native methods").

package cocoa

import "sync"

// lazyFunc binds a C symbol lazily on first call, so a missing/wrong symbol panics when
// that native is actually invoked, not at package init.
type lazyFunc[T any] struct {
	once sync.Once
	impl T
}
