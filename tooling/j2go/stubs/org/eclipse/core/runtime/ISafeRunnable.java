package org.eclipse.core.runtime;

/** Parser-only stub (no equinox sources locally); Go side: internal/jrt/core.go, mapped in j2go Manual. */
public interface ISafeRunnable {
	void handleException(Throwable exception);
	void run() throws Exception;
}
