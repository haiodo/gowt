package org.eclipse.swt.internal.cocoa;

public class Handle {
	long id;

	public Handle(long id) {
		this.id = id;
	}

	// The cocoa PI idiom: a raw handle wrapped or nil - lowers to orNil(result, NewHandleX).
	public Handle wrap() {
		long result = this.id;
		return result != 0 ? new Handle(result) : null;
	}

	// Same idiom on a bare long field: the handle emits as this.id, still orNil.
	public Handle wrapField() {
		return id != 0 ? new Handle(id) : null;
	}

	// Reversed branches - not the orNil shape, stays a hoisted conditional.
	public Handle reversed() {
		long result = this.id;
		return result == 0 ? null : new Handle(result);
	}

	// Constructor argument differs from the checked handle - not orNil, stays hoisted.
	public Handle shifted(long factor) {
		long result = this.id;
		return result != 0 ? new Handle(result + factor) : null;
	}
}
