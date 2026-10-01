package org.eclipse.swt.internal;

/** Only what the widget code reads: the platform name and whether the library loaded. */
public class Platform {
	public static final String PLATFORM = "gtk";
	public static boolean isLoadable() { return true; }
}
