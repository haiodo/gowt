//go:build darwin

// Runtime-class plumbing for the view-based Table/Tree (swt/widgets_viewtable_manual_darwin.go, docs/view-based-table.md).
package cocoa

import "github.com/ebitengine/purego"

// OnClassRegistered runs fn after every OSObjc_registerClassPair, so the manual code can extend the classes Display.initClasses
// registers without a patch of the translated Display. It must be called before the first registration; it reports false if too late.
func OnClassRegistered(fn func(cls int64)) bool {
	done := false
	var register func(cls int64)
	OSObjc_registerClassPair_once.Do(func() {
		ensureFrameworks()
		purego.RegisterLibFunc(&register, purego.RTLD_DEFAULT, "objc_registerClassPair")
		OSObjc_registerClassPair_impl = func(cls int64) {
			register(cls)
			fn(cls)
		}
		done = true
	})
	return done
}

// Msg sends selector to recv with word-sized arguments and a word-sized (or void) result.
func Msg(recv int64, selector string, args ...int64) int64 {
	words := make([]uintptr, len(args))
	for i, a := range args {
		words[i] = uintptr(a)
	}
	return int64(msg(uintptr(recv), selector, words...))
}

// NSStringID is an autoreleased NSString with the contents of s.
func NSStringID(s string) int64 { return int64(nsString(s)) }
