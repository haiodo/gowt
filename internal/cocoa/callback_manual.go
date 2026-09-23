// Hand-written replacement for org.eclipse.swt.internal.Callback (see tooling/j2go/README.md
// "Callback design" for the full reasoning and the purego.NewCallback verification).
package cocoa

import "github.com/ebitengine/purego"

// NewCallback registers fn as a native trampoline (e.g. an ObjC IMP, self/_cmd uintptr first)
// and returns its C function pointer. Go's static typing replaces Callback+reflection.
func NewCallback(fn any) uintptr {
	return purego.NewCallback(fn)
}
