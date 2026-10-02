package org.eclipse.core.runtime;

/** Parser-only stub (no equinox sources locally); Go side: internal/jrt/core.go, mapped in j2go Manual. */
public class ListenerList<E> {
	public void add(E listener) {}
	public void remove(Object listener) {}
	public Object[] getListeners() { return null; }
	public int size() { return 0; }
	public boolean isEmpty() { return true; }
	public void clear() {}
}
