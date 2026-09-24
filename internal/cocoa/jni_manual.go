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

func OSJNIGetObject(globalRef int64) any {
	return globalRefs[globalRef]
}
