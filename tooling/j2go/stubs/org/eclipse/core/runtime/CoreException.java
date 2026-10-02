package org.eclipse.core.runtime;

/** Parser-only stub (no equinox sources locally); Go side: internal/jrt/core.go, mapped in j2go Manual. */
public class CoreException extends Exception {
	public CoreException(IStatus status) {}
	public final IStatus getStatus() { return null; }
}
