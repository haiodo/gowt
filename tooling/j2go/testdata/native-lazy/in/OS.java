package org.eclipse.swt.internal.cocoa;

/** Fixture for the lazy-native codegen (NativeEmitter.emitLazyNative / emitStretNative):
 *  value-returning, void, past-8-int-args array-pointer, and a _stret struct-result shape. */
public class OS {
	public static final native long SimpleNative(int code);
	public static final native void VoidNative(long value);
	public static final native long WideNative(long a, int b, long c, int d, long e, int f, long g, int h, long i, int[] data);
	public static final native void msgSend_stret(WidgetStruct result, long id, long sel, boolean flag);
}
