package dev.gowt.j2go.emit;

import dev.gowt.j2go.GoTypes;
import dev.gowt.j2go.Manual;
import org.eclipse.jdt.core.dom.IMethodBinding;
import org.eclipse.jdt.core.dom.ITypeBinding;

/** A member declared with a type variable is Go any; the use site sees the substituted type. */
final class ErasedGenerics {

	private ErasedGenerics() {}

	/** For a call: only a method of a translated class whose own signature has no type parameter (AssertThrows[T] is typed already). */
	static String castCall(Emitter emitter, String text, IMethodBinding mb) {
		IMethodBinding decl = mb.getMethodDeclaration();
		if (decl.getTypeParameters().length > 0 || emitter.model.lookup(decl.getDeclaringClass()) == null) return text;
		return cast(emitter, text, decl.getReturnType(), mb.getReturnType());
	}

	static String cast(Emitter emitter, String text, ITypeBinding declared, ITypeBinding seen) {
		if (declared == null || !declared.isTypeVariable() || seen == null || seen.isTypeVariable() || seen.isCapture() || seen.isNullType()) return text;
		String go = GoTypes.map(seen, emitter);
		if (go.isEmpty() || go.equals("any") || go.startsWith("unsupported_")) return text;
		emitter.fileImports.add(Manual.JRT_IMPORT);
		return "jrt.Cast[" + go + "](" + text + ")";
	}
}
