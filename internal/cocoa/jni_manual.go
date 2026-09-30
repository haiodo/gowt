// JNI global references (OS.NewGlobalRef/JNIGetObject/DeleteGlobalRef): SWT stores a widget's
// handle in its NSView's SWT_OBJECT ivar and maps it back in Display.getWidget. A Go handle table
// does the same job - the ivar holds a small integer, never a Go pointer.
package cocoa

var (
	globalRefs    = map[int64]any{}
	nextGlobalRef int64
)

func OSNewGlobalRef(object any) int64 {
	nextGlobalRef++
	globalRefs[nextGlobalRef] = object
	return nextGlobalRef
}

func OSDeleteGlobalRef(globalRef int64) {
	delete(globalRefs, globalRef)
}

// A disposed widget, or one without a Display, is not gettable: in Java a late native callback finds null.
func OSJNIGetObject(globalRef int64) any {
	o := globalRefs[globalRef]
	if d, ok := o.(interface{ IsDisposed() bool }); ok && d.IsDisposed() {
		return nil
	}
	if l, ok := o.(interface{ HandleLost() bool }); ok && l.HandleLost() {
		return nil
	}
	return o
}
