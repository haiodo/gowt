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
				TypeModel.ClassInfo declCi = emitter.model.lookup(declaring);
				String fn = declCi != null ? emitter.qualify(manualMethodGoName, declCi) : manualMethodGoName;
				return fn + "(" + String.join(", ", args) + ")";
			}
			String manualRecv = mi.getExpression() != null ? emitter.expr(mi.getExpression()) : "this";
			return manualRecv + "." + manualMethodGoName + "(" + String.join(", ", args) + ")";
		}

		// A manual value type backed by a bare "any" (java.util.Map, ...) has no real Go method to
		// dispatch to - fall through to the ordinary "unresolved call" degrade below instead.
		if (Manual.isManual(qualified) && !Manual.isBareAny(qualified)) {
			if (Modifier.isStatic(mb.getModifiers())) {
				return Manual.staticMember(qualified, mb.getName()) + "(" + String.join(", ", args) + ")";
			}
			String recv = mi.getExpression() != null ? emitter.expr(mi.getExpression()) : "this";
			return castErased(recv + "." + Manual.instanceMember(mb.getName()) + "(" + String.join(", ", args) + ")", mb);
		}

		// Receiver is manual but inherits a real translated method (Caret/IME extend Widget) - no
		// .impl to dispatch through, and Manual.instanceMember's bare capitalize picks the wrong
		// overload (sendEvent(int) -> SendEventEventType); name it as that class would instead.
		if (mi.getExpression() != null && !Modifier.isStatic(mb.getModifiers())) {
			ITypeBinding receiverType = mi.getExpression().resolveTypeBinding();
			String receiverQualified = receiverType == null ? null : receiverType.getErasure().getQualifiedName();
			if (receiverQualified != null && Manual.isManual(receiverQualified) && !Manual.isBareAny(receiverQualified)
					&& emitter.model.lookup(receiverType) == null) {
				String recv = emitter.expr(mi.getExpression());
				TypeModel.ClassInfo declCi = emitter.model.lookup(declaring);
				String goName;
				if (declCi != null) {
					String sig = TypeModel.signature(mb);
					TypeModel.ClassInfo point = declCi.overridePoint(sig);
					goName = point != null ? declCi.root.overriddenRootMethodGoNames.get(sig)
							: emitter.names.goMemberName(mb, Names.javaMethodBaseGoName(mb.getName()));
				} else {
					goName = Manual.instanceMember(mb.getName());
				}
				return recv + "." + goName + "(" + String.join(", ", args) + ")";
			}
		}

		TypeModel.ClassInfo ci = emitter.model.lookup(declaring);
		if (ci == null) {
			emitter.unsupported.add("MethodInvocation: unresolved declaring type " + qualified + "." + mb.getName());
			// Receiver/args are referenced inside the panic closure (a local used only here must not
			// go "declared and not used"), not hoisted before it: evaluating them eagerly would break
			// && / || short-circuiting around a call that is never reached. A static call's
			// receiver is a type qualifier (Integer.toHexString), not a value.
			List<String> uses = new ArrayList<>(args);
			if (mi.getExpression() != null && !Modifier.isStatic(mb.getModifiers())) uses.add(0, emitter.expr(mi.getExpression()));
			return usingArgs(emitter.panicClosure(mi, "unresolved call " + mb.getName()), uses);
		}

		if (Modifier.isStatic(mb.getModifiers())) {
			String goName = emitter.staticMethodGoName(mb, ci);
			// A struct JNI passes by pointer (memmove's dest, objc_msgSend_stret's result, see
			// NativeEmitter.paramType): Java mutates the caller's object, Go needs its address.
			for (int i = 0; i < args.size() && Modifier.isNative(mb.getModifiers()); i++) {
				if (emitter.model.isNativeStructPointerParam(mb, i)) args.set(i, addressOf((Expression) mi.arguments().get(i), args.get(i)));
			}
			return goName + "(" + String.join(", ", args) + ")";
		}

		String recv = mi.getExpression() != null ? emitter.expr(mi.getExpression()) : emitter.implicitThis(declaring);
		String base = Names.javaMethodBaseGoName(mb.getName());
		String sig = TypeModel.signature(mb);
		boolean overridden = ci.overridePoint(sig) != null;
		String goName = overridden ? ci.root.overriddenRootMethodGoNames.get(sig) : emitter.names.goMemberName(mb, base);
		String callText = overridden
				? recv + emitter.implAccess(ci.root) + "." + goName + "(" + String.join(", ", args) + ")"
				: recv + "." + goName + "(" + String.join(", ", args) + ")";

		if (overridden) {
			ITypeBinding staticReturnType = mb.getReturnType(); // as resolved at THIS call site (covariant-aware)
			TypeModel.ClassInfo rootCi = ci.root;
			ITypeBinding declared = ci.overridePoint(sig).declaredBinding(sig).getReturnType();
			if (staticReturnType != null && !staticReturnType.getErasure().isEqualTo(declared.getErasure())) {
				TypeModel.ClassInfo target = emitter.model.lookup(staticReturnType);
				if (target != null && !target.isStruct) {
					String tmp = "t" + (++emitter.tempCounter);
					emitter.prelude.add(tmp + " := " + callText);
					String helper = emitter.ensureCascadeHelper(rootCi, target);
					String tmp2 = "t" + (++emitter.tempCounter);
					// tmp's own Go type is target (the covariant return type), not rootCi's package.
					emitter.prelude.add(tmp2 + ", _ := " + helper + "(" + tmp + emitter.implAccess(target.root) + ")");
					return tmp2;
				}
			}
		}
		return callText;
	}

	private String addressOf(Expression e, String text) {
		if (e instanceof Name || e instanceof FieldAccess || e instanceof ArrayAccess || e instanceof ClassInstanceCreation) {
			return "&" + text;
		}
		String tmp = "t" + (++emitter.tempCounter);
		emitter.prelude.add(tmp + " := " + text);
		return "&" + tmp;
	}

	/** A manual container's generic method (Map.get, Queue.poll) returns any in Go; the call site
	 * expects the type argument's Go type. */
	private String castErased(String call, IMethodBinding mb) {
		if (!mb.getMethodDeclaration().getReturnType().isTypeVariable()) return call;
		String goType = dev.gowt.j2go.GoTypes.map(mb.getReturnType(), emitter);
		if (goType.equals("any")) return call;
		emitter.fileImports.add("github.com/haiodo/gowt/internal/jrt");
		return "jrt.Cast[" + goType + "](" + call + ")";
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

	// Java exception classes are Go error values (GoTypes) backed by internal/jrt's structs.
	private String newJavaException(String qualified, ClassInstanceCreation cic) {
		String goType = switch (qualified) {
			case Manual.JAVA_RUNTIME_EXCEPTION, Manual.JAVA_EXCEPTION -> "jrt.RuntimeException";
			case Manual.JAVA_ERROR, Manual.JAVA_THROWABLE -> "jrt.JavaError";
			default -> null;
		};
		if (goType == null) return null;
		emitter.fileImports.add("github.com/haiodo/gowt/internal/jrt");
		List<?> args = cic.arguments();
		String msg = args.isEmpty() ? "" : "Message: " + emitter.expr((Expression) args.get(0));
		return "&" + goType + "{" + msg + "}";
	}

	/** new Callback(target, "method", argCount) -> a Go closure calling that method directly
	 * (README "Callback design"). */
	private String emitCallback(ClassInstanceCreation cic) {
		List<?> a = cic.arguments();
		Expression recv = (Expression) a.get(0);
		if (a.size() != 3 || !(a.get(1) instanceof StringLiteral name) || !(a.get(2) instanceof NumberLiteral argc)) {
			emitter.unsupported.add("ClassInstanceCreation: Callback shape " + cic);
			return emitter.panicClosure(cic, "unsupported Callback");
		}
		int n = Integer.parseInt(argc.getToken());
		// `Class<?> clazz = getClass(); new Callback(clazz, ...)`: the local is otherwise unused.
		if (recv instanceof SimpleName) emitter.prelude.add("_ = " + emitter.expr(recv));
		ITypeBinding cls = recv instanceof TypeLiteral tl ? tl.getType().resolveBinding() : emitter.currentClassInfo.binding;
		IMethodBinding target = findCallbackTarget(cls, name.getLiteralValue(), n);
		if (target == null) {
			emitter.unsupported.add("ClassInstanceCreation: Callback target not found " + cic);
			return emitter.panicClosure(cic, "unsupported Callback");
		}
		List<String> args = new ArrayList<>();
		for (int i = 0; i < n; i++) args.add("args[" + i + "]");
		String call;
		TypeModel.ClassInfo ci = emitter.model.lookup(target.getDeclaringClass());
		if (Modifier.isStatic(target.getModifiers())) {
			call = emitter.staticMethodGoName(target, ci) + "(" + String.join(", ", args) + ")";
		} else {
			call = callText(recv instanceof ThisExpression ? emitter.expr(recv) : "this", target, ci, args);
		}
		String body = target.getReturnType().getName().equals("void") ? call + "; return 0" : "return " + call;
		return "NewCallbackFn(func(args []int64) int64 { " + body + " }, " + n + ")";
	}

	private IMethodBinding findCallbackTarget(ITypeBinding cls, String name, int argc) {
		for (ITypeBinding t = cls; t != null; t = t.getSuperclass()) {
			for (IMethodBinding m : t.getDeclaredMethods()) {
				if (m.getName().equals(name) && m.getParameterTypes().length == argc) return m;
			}
		}
		return null;
	}

	/** recv.goName(args), through the impl cascade when mb is an override point. */
	private String callText(String recv, IMethodBinding mb, TypeModel.ClassInfo ci, List<String> args) {
		String sig = TypeModel.signature(mb);
		boolean overridden = ci.overridePoint(sig) != null;
		String goName = overridden ? ci.root.overriddenRootMethodGoNames.get(sig)
				: emitter.names.goMemberName(mb, Names.javaMethodBaseGoName(mb.getName()));
		return recv + (overridden ? emitter.implAccess(ci.root) : "") + "." + goName + "(" + String.join(", ", args) + ")";
	}

	// References the args inside the panic closure: a local used only there must not go unused.
	private static String usingArgs(String closure, List<String> uses) {
		if (uses.isEmpty()) return closure;
		int brace = closure.indexOf("{ ") + 2;
		return closure.substring(0, brace) + "_ = []any{" + String.join(", ", uses) + "}; " + closure.substring(brace);
	}

	String emitNew(ClassInstanceCreation cic) {
		IMethodBinding ctor = cic.resolveConstructorBinding();
		ITypeBinding declaring = ctor.getDeclaringClass();
		TypeModel.ClassInfo ci = emitter.model.lookup(declaring);
		if (cic.getAnonymousClassDeclaration() != null) return emitter.emitAnonymous(cic);
		if (ci == null) {
			String qualified = declaring.getErasure().getQualifiedName();
			if (qualified.equals("org.eclipse.swt.internal.Callback")) return emitCallback(cic);
			// A bare lock object (`trackingLock = new Object()`): only its identity matters.
			if (qualified.equals("java.lang.Object")) return "any(&struct{}{})";
			String exception = newJavaException(qualified, cic);
			if (exception != null) return exception;
			// new String(char[]): the only java.lang.String constructor used in the translated
			// set (NSString.getString()) - buffer holds UTF-16 code units, same as Java's char[].
			// new String(char[], offset, count) (TextLayout) slices the same buffer.
			int n = ctor.getParameterTypes().length;
			if (qualified.equals("java.lang.String") && (n == 1 || n == 3)
					&& ctor.getParameterTypes()[0].isArray()
					&& ctor.getParameterTypes()[0].getComponentType().getName().equals("char")) {
				emitter.fileImports.add("unicode/utf16");
				String arg = emitter.expr((Expression) cic.arguments().get(0));
				if (n == 3) {
					String off = emitter.expr((Expression) cic.arguments().get(1));
					arg += "[" + off + ":" + off + "+" + emitter.expr((Expression) cic.arguments().get(2)) + "]";
				}
				return "string(utf16.Decode(" + arg + "))";
			}
			if (Manual.isManual(qualified)) {
				// A value-type manual entry (any/error/...) has no real constructor function -
				// "new X()" is just its zero value, same as an ordinary unresolved type would get.
				if (Manual.isValueType(qualified)) return emitter.zeroValue(declaring);
				emitter.addManualImport(qualified);
				List<String> manualArgs = buildArgs(cic.arguments(), ctor);
				return Manual.ctorFuncName(qualified) + "(" + String.join(", ", manualArgs) + ")";
			}
			emitter.unsupported.add("ClassInstanceCreation: unresolved type " + declaring.getQualifiedName());
			return usingArgs(emitter.panicClosure(cic, "unresolved new " + declaring.getName()), buildArgs(cic.arguments(), ctor));
		}
		// struct classes (NSPoint, NSRect, ...) have no declared constructor in the Java source
		// (JLS implicit no-arg ctor only): "new X()" is a zero-value composite literal.
		if (ci.isStruct) return emitter.qualifiedTypeName(ci) + "{}";
		boolean pub = Modifier.isPublic(ctor.getModifiers());
		String prefix = emitter.qualify((pub ? "New" : "new") + ci.goFuncPrefix, ci);
		String goName = emitter.ctorGoName(ctor, prefix);
		List<String> args = buildArgs(cic.arguments(), ctor);
		if (!EmitUtil.isInnerClass(declaring)) return goName + "(" + String.join(", ", args) + ")";
		// Set after construction: fine as long as the inner ctor itself doesn't reach the outer.
		String tmp = "inner" + (++emitter.tempCounter);
		emitter.prelude.add(tmp + " := " + goName + "(" + String.join(", ", args) + ")");
		emitter.prelude.add(tmp + "." + EmitUtil.OUTER_FIELD + " = " + (cic.getExpression() != null ? emitter.expr(cic.getExpression()) : "this"));
		return tmp;
	}
}
