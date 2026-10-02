package org.eclipse.core.runtime;

/** Parser-only stub (no equinox sources locally); Go side: internal/jrt/core.go, mapped in j2go Manual. */
public class Status implements IStatus {
	public Status(int severity, String pluginId, String message) {}
	public Status(int severity, String pluginId, String message, Throwable exception) {}
	public int getSeverity() { return 0; }
	public String getMessage() { return null; }
	public String getPlugin() { return null; }
	public int getCode() { return 0; }
	public Throwable getException() { return null; }
	public boolean isOK() { return false; }
}
