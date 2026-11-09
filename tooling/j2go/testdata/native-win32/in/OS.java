package org.eclipse.swt.internal.win32;

/** Fixture for the win32 native codegen (WinNativeEmitter): scalar, float, array and struct
 *  parameters, a dynamic optional export and the vtable call family. */
public class OS {
	public static final native int Simple(int code);
	public static final native int LockExternal(long obj, boolean fLock, boolean fLastUnlockReleases);
	public static final native double Distance(float x, double y);
	public static final native int FillRect(RECT rc);
	public static final native boolean GetInfo(char [] buffer, int [] sizes);
	/** @method flags=dynamic */
	public static final native int MaybePresent(long hwnd);
	/** @method flags=dynamic */
	public static final native void MaybeAbsent(long hwnd);
	public static final native int VtblCall(int fnNumber, long ppVtbl, double arg0);
	/** @param arg0 struct */
	public static final native int VtblCall(int fnNumber, long ppVtbl, RECT arg0, long arg1);
}
