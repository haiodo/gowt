package dev.gowt.j2go.emit;

import org.eclipse.jdt.core.dom.ITypeBinding;

/** Outer.this / the implicit outer instance of an inner class: the this_0 chain, upcast to the wanted class. */
final class OuterThis {

	private OuterThis() {}

	static String path(Emitter emitter, ITypeBinding target) {
		if (emitter.anonThis != null || emitter.currentClassInfo == null || target == null) return emitter.anonThis != null ? emitter.anonThis : "this";
		String p = "this";
		ITypeBinding cur = emitter.currentClassInfo.binding;
		while (cur != null && !cur.getErasure().isSubTypeCompatible(target.getErasure()) && cur.getDeclaringClass() != null) {
			p += "." + EmitUtil.OUTER_FIELD;
			cur = cur.getDeclaringClass();
		}
		return cur == null ? p : emitter.upcastObject(p, cur, target);
	}
}
