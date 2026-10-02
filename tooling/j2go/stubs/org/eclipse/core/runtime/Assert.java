package org.eclipse.core.runtime;

/** Parser-only stub of org.eclipse.core.runtime.Assert (not in any local repo); Go side: internal/jrt/core.go. */
public final class Assert {
	public static boolean isLegal(boolean expression) { return true; }
	public static boolean isLegal(boolean expression, String message) { return true; }
	public static void isNotNull(Object object) {}
	public static void isNotNull(Object object, String message) {}
	public static boolean isTrue(boolean expression) { return true; }
	public static boolean isTrue(boolean expression, String message) { return true; }
}
