package org.eclipse.swt.internal.cocoa;

/** C-struct mirror: superIsObject + only toString makes it a Go value type (isStruct), the
 *  _stret native's result parameter. */
public class WidgetStruct {
	public double x;
	public double y;

	@Override
	public String toString() {
		return "WidgetStruct{" + x + "," + y + "}";
	}
}
