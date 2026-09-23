package dev.gowt.j2go.emit;

import dev.gowt.j2go.Manual;
import dev.gowt.j2go.Names;
import dev.gowt.j2go.TypeModel;
import org.eclipse.jdt.core.dom.*;

import java.util.ArrayList;
import java.util.List;

/** Method invocation, new, argument building/adaptation. */
final class InvocationEmitter {

	private final Emitter emitter;

	InvocationEmitter(Emitter emitter) {
		this.emitter = emitter;
	}

	// ---------------------------------------------------------------- calls / new

	String emitMethodInvocation(MethodInvocation mi) {
		IMethodBinding mb = mi.resolveMethodBinding();
		ITypeBinding declaring = mb.getDeclaringClass();
		String qualified = declaring.getErasure().getQualifiedName();

		String intrinsic = emitter.tryIntrinsic(mi, mb);
		if (intrinsic != null) return intrinsic;

		List<String> args = buildArgs(mi.arguments(), mb);

		// Per-signature override first: the generic per-type Manual dispatch below has no
		// overload awareness, so an overloaded manual method (e.g. Display.map) needs this.
		String manualMethodGoName = Manual.manualMethod(Names.erasureKey(mb));
		if (manualMethodGoName != null) {
			if (Modifier.isStatic(mb.getModifiers())) {
				return manualMethodGoName + "(" + String.join(", ", args) + ")";
			}
			String manualRecv = mi.getExpression() != null ? emitter.expr(mi.getExpression()) : "this";
			return manualRecv + "." + manualMethodGoName + "(" + String.join(", ", args) + ")";
		}

		if (Manual.isManual(qualified)) {
			if (Modifier.isStatic(mb.getModifiers())) {
				return Manual.staticMember(qualified, mb.getName()) + "(" + String.join(", ", args) + ")";
			}
			String recv = mi.getExpression() != null ? emitter.expr(mi.getExpression()) : "this";
			return recv + "." + Manual.instanceMember(mb.getName()) + "(" + String.join(", ", args) + ")";
		}

		TypeModel.ClassInfo ci = emitter.model.lookup(declaring);
		if (ci == null) {
			emitter.unsupported.add("MethodInvocation: unresolved declaring type " + qualified + "." + mb.getName());
			// args was already evaluated but the panic closure below never references it - blank
			// assign so a local used only here doesn't go "declared and not used" (Go, not Java).
			// Skip a static call's receiver - it's a type qualifier (Integer.toHexString), not
			// a value.
			if (mi.getExpression() != null && !Modifier.isStatic(mb.getModifiers())) {
				emitter.prelude.add("_ = " + emitter.expr(mi.getExpression()));
			}
			for (String a : args) emitter.prelude.add("_ = " + a);
			return emitter.panicClosure(mi, "unresolved call " + mb.getName());
		}

		if (Modifier.isStatic(mb.getModifiers())) {
			String goName = emitter.staticMethodGoName(mb, ci);
			// objc_msgSend_stret(result, ...): result became *Struct at the declaration (see
			// emitStretNative) - the call site's own struct-valued argument needs its address.
			if (Modifier.isNative(mb.getModifiers()) && mb.getName().endsWith("_stret") && !args.isEmpty()
					&& emitter.structParamTarget(mb.getParameterTypes()[0]) != null) {
				args.set(0, "&" + args.get(0));
			}
			return goName + "(" + String.join(", ", args) + ")";
		}

		String recv = mi.getExpression() != null ? emitter.expr(mi.getExpression()) : "this";
		String base = Names.javaMethodBaseGoName(mb.getName());
		String sig = TypeModel.signature(mb);
		boolean overridden = ci.overridePoint(sig) != null;
		String goName = overridden ? ci.root.overriddenRootMethodGoNames.get(sig) : emitter.names.goMemberName(mb, base);
		String callText = overridden
				? recv + ".Impl." + goName + "(" + String.join(", ", args) + ")"
				: recv + "." + goName + "(" + String.join(", ", args) + ")";

		if (overridden) {
			ITypeBinding staticReturnType = mb.getReturnType(); // as resolved at THIS call site (covariant-aware)
			TypeModel.ClassInfo rootCi = ci.root;
			if (staticReturnType != null && !staticReturnType.getErasure().getBinaryName().equals(rootCi.binaryName)) {
				TypeModel.ClassInfo target = emitter.model.lookup(staticReturnType);
				if (target != null) {
					String tmp = "t" + (++emitter.tempCounter);
					emitter.prelude.add(tmp + " := " + callText);
					String helper = emitter.ensureCascadeHelper(rootCi, target);
					String tmp2 = "t" + (++emitter.tempCounter);
					emitter.prelude.add(tmp2 + ", _ := " + helper + "(" + tmp + ".Impl)");
					return tmp2;
				}
			}
		}
		return callText;
	}

	List<String> buildArgs(List<?> javaArgs, IMethodBinding mb) {
		List<String> args = new ArrayList<>();
		ITypeBinding[] paramTypes = mb.getParameterTypes();
		// Varargs param type is already an array (paramList gets []int32 for free); only the call
		// site needs fixing, collecting trailing args into a slice literal.
		int fixedCount = mb.isVarargs() ? paramTypes.length - 1 : paramTypes.length;
		for (int i = 0; i < fixedCount && i < javaArgs.size(); i++) {
			Expression a = (Expression) javaArgs.get(i);
			args.add(emitter.adaptNumeric(emitter.expr(a), a.resolveTypeBinding(), paramTypes[i]));
		}
		if (!mb.isVarargs()) {
			for (int i = fixedCount; i < javaArgs.size(); i++) {
				args.add(emitter.expr((Expression) javaArgs.get(i)));
			}
			return args;
		}
		ITypeBinding varargArrayType = paramTypes[paramTypes.length - 1];
		boolean passedAsArray = javaArgs.size() == paramTypes.length && isArrayTyped((Expression) javaArgs.get(javaArgs.size() - 1));
		if (passedAsArray) {
			args.add(emitter.expr((Expression) javaArgs.get(javaArgs.size() - 1)));
			return args;
		}
		ITypeBinding componentType = varargArrayType.getComponentType();
		List<String> elems = new ArrayList<>();
		for (int i = fixedCount; i < javaArgs.size(); i++) {
			Expression a = (Expression) javaArgs.get(i);
			elems.add(emitter.adaptNumeric(emitter.expr(a), a.resolveTypeBinding(), componentType));
		}
		args.add(dev.gowt.j2go.GoTypes.map(varargArrayType, emitter) + "{" + String.join(", ", elems) + "}");
		return args;
	}

	private boolean isArrayTyped(Expression e) {
		ITypeBinding t = e.resolveTypeBinding();
		return t != null && t.isArray();
	}

	String adaptArg(Expression a, IMethodBinding mb, int i) {
		ITypeBinding[] paramTypes = mb.getParameterTypes();
		String text = emitter.expr(a);
		if (i < paramTypes.length) text = emitter.adaptNumeric(text, a.resolveTypeBinding(), paramTypes[i]);
		return text;
	}

	String emitNew(ClassInstanceCreation cic) {
		IMethodBinding ctor = cic.resolveConstructorBinding();
		ITypeBinding declaring = ctor.getDeclaringClass();
		TypeModel.ClassInfo ci = emitter.model.lookup(declaring);
		if (ci == null) {
			// Anonymous class body not translated (README "anonymous classes") - typed as a
			// pointer to its real base class so it still satisfies the caller's expected type.
			if (cic.getAnonymousClassDeclaration() != null) {
				TypeModel.ClassInfo baseCi = emitter.model.lookup(declaring.getSuperclass());
				if (baseCi == null) {
					for (ITypeBinding iface : declaring.getInterfaces()) {
						baseCi = emitter.model.lookup(iface);
						if (baseCi != null) break;
					}
				}
				if (baseCi != null) {
					String baseName = emitter.qualifiedTypeName(baseCi);
					emitter.unsupported.add("ClassInstanceCreation: anonymous class body based on " + baseName + " not translated");
					return emitter.panicClosureTyped("*" + baseName, "unsupported anonymous class");
				}
			}
			String qualified = declaring.getErasure().getQualifiedName();
			// new String(char[]): the only java.lang.String constructor used in the translated
			// set (NSString.getString()) - buffer holds UTF-16 code units, same as Java's char[].
			if (qualified.equals("java.lang.String") && ctor.getParameterTypes().length == 1
					&& ctor.getParameterTypes()[0].isArray()
					&& ctor.getParameterTypes()[0].getComponentType().getName().equals("char")) {
				emitter.fileImports.add("unicode/utf16");
				String arg = emitter.expr((Expression) cic.arguments().get(0));
				return "string(utf16.Decode(" + arg + "))";
			}
			if (Manual.isManual(qualified)) {
				emitter.addManualImport(qualified);
				List<String> manualArgs = buildArgs(cic.arguments(), ctor);
				return Manual.ctorFuncName(qualified) + "(" + String.join(", ", manualArgs) + ")";
			}
			emitter.unsupported.add("ClassInstanceCreation: unresolved type " + declaring.getQualifiedName());
			return emitter.panicClosure(cic, "unresolved new " + declaring.getName());
		}
		// struct classes (NSPoint, NSRect, ...) have no declared constructor in the Java source
		// (JLS implicit no-arg ctor only): "new X()" is a zero-value composite literal.
		if (ci.isStruct) return emitter.qualifiedTypeName(ci) + "{}";
		boolean pub = Modifier.isPublic(ctor.getModifiers());
		String prefix = emitter.qualify((pub ? "New" : "new") + ci.goFuncPrefix, ci);
		String goName = emitter.names.goMemberName(ctor, prefix);
		List<String> args = buildArgs(cic.arguments(), ctor);
		return goName + "(" + String.join(", ", args) + ")";
	}
}
