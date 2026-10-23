package org.eclipse.swt.internal;

/** Type information only; the Go side is swt/dbus_stub_manual_linux.go (upstream is LGPL). */
public class SessionManagerDBus {
	public interface IListener {
		boolean isReadyToExit();

		void stop();
	}

	public void addListener(IListener listener) {
	}

	public void dispose() {
	}
}
