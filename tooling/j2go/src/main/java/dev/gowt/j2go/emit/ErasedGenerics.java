package dev.gowt.j2go.emit;

import dev.gowt.j2go.GoTypes;
import dev.gowt.j2go.Manual;
import org.eclipse.jdt.core.dom.*;
import org.eclipse.jdt.core.dom.ITypeBinding;

/** A member declared with a type variable is Go any; the use site sees the substituted type. */
final class ErasedGenerics {

	private ErasedGenerics() {}

	/** For a call: only a method of a translated class whose own signature has no type parameter (AssertThrows[T] is typed already). */
	static String castCall(Emitter emitter, String text, IMethodBinding mb) {
		IMethodBinding decl = mb.getMethodDeclaration();
		if (emitter.model.lookup(decl.getDeclaringClass()) == null) return text;
		return cast(emitter, text, decl.getReturnType(), mb.getReturnType());
	}

	/** A lambda passed to a type-parameterised method of a translated class: the method's param is the erased func type,
	 * so the literal (typed by inference: func(h) bool) is wrapped into a func with the erased signature. */
	static String erasedFunc(Emitter emitter, String text, Expression arg, IMethodBinding call, int i) {
		IMethodBinding decl = call.getMethodDeclaration();
		if (decl.getTypeParameters().length == 0 || emitter.model.lookup(decl.getDeclaringClass()) == null
				|| GoTypes.map(decl.getParameterTypes()[i], emitter).equals(GoTypes.map(call.getParameterTypes()[i], emitter))) return text;
		IMethodBinding sam = decl.getParameterTypes()[i].getFunctionalInterfaceMethod();
		IMethodBinding inferred = call.getParameterTypes()[i].getFunctionalInterfaceMethod();
		if (sam == null || inferred == null) return text;
		java.util.List<String> params = new java.util.ArrayList<>(), args = new java.util.ArrayList<>();
		for (int k = 0; k < sam.getParameterTypes().length; k++) {
			String erased = GoTypes.map(sam.getParameterTypes()[k], emitter), seen = GoTypes.map(inferred.getParameterTypes()[k], emitter);
			params.add("a" + k + " " + erased);
			args.add(erased.equals(seen) || erased.equals("any") && !seen.equals("any") ? (erased.equals(seen) ? "a" + k : cast(emitter, "a" + k, sam.getParameterTypes()[k], inferred.getParameterTypes()[k])) : "a" + k);
		}
		String ret = GoTypes.map(sam.getReturnType(), emitter);
		return "func(" + String.join(", ", params) + ") " + ret + " { " + (ret.isEmpty() ? "" : "return ") + text + "(" + String.join(", ", args) + ") }";
	}

	static String cast(Emitter emitter, String text, ITypeBinding declared, ITypeBinding seen) {
		if (declared == null || !declared.isTypeVariable() || seen == null || seen.isCapture() || seen.isNullType()) return text;
		String go = GoTypes.map(seen, emitter);
		if (go.isEmpty() || go.equals("any") || go.startsWith("unsupported_")) return text;
		// A type variable seen through another one (F of a subclass for F of its base) differs only when the bounds do.
		if (seen.isTypeVariable() && go.equals(GoTypes.map(declared, emitter))) return text;
		// A bound that is a translated class: the value is the bound's pointer, read back as the subclass via its impl.
		if (emitter.model.lookup(declared) != null) {
			String narrowed = emitter.typeTestEmitter.narrow(text, declared, seen.isTypeVariable() ? seen.getErasure() : seen);
			if (narrowed != null) return narrowed;
		}
		emitter.fileImports.add(Manual.JRT_IMPORT);
		return "jrt.Cast[" + go + "](" + text + ")";
	}
}
