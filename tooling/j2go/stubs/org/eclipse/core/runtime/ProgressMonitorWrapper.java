package org.eclipse.core.runtime;

/** Parser-only stub (no equinox sources locally); Go side: internal/jrt/core.go, mapped in j2go Manual. */
public abstract class ProgressMonitorWrapper implements IProgressMonitor {
	protected ProgressMonitorWrapper(IProgressMonitor monitor) {}
	public IProgressMonitor getWrappedProgressMonitor() { return null; }
	public void beginTask(String name, int totalWork) {}
	public void done() {}
	public void internalWorked(double work) {}
	public boolean isCanceled() { return false; }
	public void setCanceled(boolean value) {}
	public void setTaskName(String name) {}
	public void subTask(String name) {}
	public void worked(int work) {}
}
