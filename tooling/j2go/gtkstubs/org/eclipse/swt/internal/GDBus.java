package org.eclipse.swt.internal;

import java.util.function.Function;

/** Type information only; the Go side is swt/dbus_stub_manual_linux.go (upstream is LGPL). */
public class GDBus {
	public static class GDBusMethod {
		public GDBusMethod(String name, String[][] in, String[][] out, Function<Object[], Object[]> handler) {
		}
	}

	public static void init(GDBusMethod[] methods, String appName) {
	}
}
