package dev.gowt.j2go.emit;

import dev.gowt.j2go.TypeModel;
import org.eclipse.jdt.core.dom.IMethodBinding;
import org.eclipse.jdt.core.dom.Modifier;

/** Which override carries the exported wrapper of a dispatch point. */
final class WrapperRules {

	private WrapperRules() {}

	/** A public override of a package-private root method (win32's Control.getMenu): the API name starts here. win32 only. */
	static boolean firstPublicBelowHidden(IMethodBinding mb, TypeModel.ClassInfo ci, String sig) {
		if (dev.gowt.j2go.GoTypes.platform != dev.gowt.j2go.Platform.WIN32 || !isApi(mb)) return false;
		TypeModel.ClassInfo point = ci.overridePoint(sig);
		if (point == null || isApi(point.declaredBinding(sig))) return false;
		for (TypeModel.ClassInfo c = ci.superclass; c != null && c != point; c = c.superclass) {
			IMethodBinding o = c.declaredBinding(sig);
			if (o != null && isApi(o)) return false;
		}
		return true;
	}

	private static boolean isApi(IMethodBinding mb) {
		return Modifier.isPublic(mb.getModifiers()) || Modifier.isProtected(mb.getModifiers());
	}
}
