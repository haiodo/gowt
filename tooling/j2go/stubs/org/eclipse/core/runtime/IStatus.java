package org.eclipse.core.runtime;

/** Parser-only stub (no equinox sources locally); Go side: internal/jrt/core.go, mapped in j2go Manual. */
public interface IStatus {
	int OK = 0, INFO = 1, WARNING = 2, ERROR = 4, CANCEL = 8;
	int getSeverity();
	String getMessage();
	String getPlugin();
	int getCode();
	Throwable getException();
	boolean isOK();
}
