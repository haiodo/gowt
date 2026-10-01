package org.eclipse.swt.internal;

/** A boxed native handle, as the widget code uses it (new LONG(h), .value). */
public class LONG {
	public long value;
	public LONG(long value) { this.value = value; }
}
