package org.eclipse.core.runtime;

/** Parser-only stub (no equinox sources locally); Go side: internal/jrt/core.go, mapped in j2go Manual. */
public interface IProgressMonitor {
	int UNKNOWN = -1;
	void beginTask(String name, int totalWork);
	void done();
	void internalWorked(double work);
	boolean isCanceled();
	void setCanceled(boolean value);
	void setTaskName(String name);
	void subTask(String name);
	void worked(int work);
}
