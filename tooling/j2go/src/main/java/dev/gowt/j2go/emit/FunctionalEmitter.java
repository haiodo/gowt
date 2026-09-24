package dev.gowt.j2go.emit;

import dev.gowt.j2go.GoTypes;
import dev.gowt.j2go.Names;
import dev.gowt.j2go.TypeModel;
import org.eclipse.jdt.core.dom.*;

import java.util.ArrayList;
import java.util.List;

import static dev.gowt.j2go.emit.EmitUtil.ind;

/** Lambdas, method references, anonymous classes: a Go func literal wrapped for its target
 * interface, or a generated struct for an anonymous subclass (README "Functional values"). */
final class FunctionalEmitter {

	private static final String JRT_IMPORT = "github.com/haiodo/gowt/internal/jrt";

	private final Emitter emitter;

	FunctionalEmitter(Emitter emitter) {
		this.emitter = emitter;
	}

	// ---------------------------------------------------------------- lambdas / method refs

	String emitLambda(LambdaExpression le) {
		IMethodBinding lmb = le.resolveMethodBinding();
		List<String> names = new ArrayList<>();
		for (Object p : le.parameters()) names.add(((VariableDeclaration) p).getName().getIdentifier());
		String fn = lmb == null ? null : funcLiteral(lmb, names, le.getBody());
		String wrapped = fn == null ? null : wrap(le.resolveTypeBinding(), fn);
		return wrapped != null ? wrapped : marker(le, "LambdaExpression");
	}

	/** `expr::method`: a func literal forwarding the SAM's params to the bound method. */
	String emitMethodReference(ExpressionMethodReference emr) {
		IMethodBinding target = emr.resolveMethodBinding();
		ITypeBinding fType = emr.resolveTypeBinding();
		IMethodBinding sam = fType == null ? null : fType.getFunctionalInterfaceMethod();
		TypeModel.ClassInfo declCi = target == null ? null : emitter.model.lookup(target.getDeclaringClass());
		if (sam == null || declCi == null) return marker(emr, "ExpressionMethodReference");
		String goName = emitter.names.goMemberName(target, Names.javaMethodBaseGoName(target.getName()));
		String wrapped = wrap(fType, emitter.expr(emr.getExpression()) + "." + goName);
		return wrapped != null ? wrapped : marker(emr, "ExpressionMethodReference");
	}

	private String marker(Expression e, String kind) {
		String oneLine = e.toString().replace("\n", " ").trim();
		emitter.unsupported.add(kind + ": " + oneLine);
		return emitter.panicClosure(e, "unsupported " + kind);
	}

	/** Wraps a Go func value for the functional interface it is assigned to, or null. */
	private String wrap(ITypeBinding target, String fn) {
		if (target == null) return null;
		String qualified = target.getErasure().getQualifiedName();
		if (qualified.equals("java.lang.Runnable")) {
			emitter.fileImports.add(JRT_IMPORT);
			return "jrt.NewRunnable(" + fn + ")";
		}
		if (qualified.equals("java.util.function.Consumer")) return fn;
		TypeModel.ClassInfo ifaceCi = emitter.model.lookup(target);
		IMethodBinding sam = target.getFunctionalInterfaceMethod();
		if (ifaceCi == null || !ifaceCi.isInterface || sam == null) return null;
		return "&" + ensureFuncAdapter(ifaceCi, sam) + "{fn: " + fn + "}";
	}

	/** Pointer adapter, not a named func type: a Go func in an interface panics on ==, and
	 * EventTable.unhook looks listeners up with ==. Generated once per interface. */
	private String ensureFuncAdapter(TypeModel.ClassInfo ifaceCi, IMethodBinding sam) {
		String adapterName = ifaceCi.goTypeName + "Func";
		if (emitter.generatedHelpers.add(adapterName)) {
			String params = emitter.paramList(sam, null);
			String ret = emitter.retType(sam);
			String samGoName = emitter.names.goMemberName(sam, Names.javaMethodBaseGoName(sam.getName()));
			List<String> argNames = new ArrayList<>();
			for (int i = 0; i < sam.getParameterTypes().length; i++) argNames.add("a" + i);
			String args = String.join(", ", argNames);
			StringBuilder b = new StringBuilder();
			b.append("// j2go: func adapter for ").append(ifaceCi.goTypeName).append(".\n");
			b.append("type ").append(adapterName).append(" struct {\n\tfn func(").append(params).append(") ").append(ret).append("\n}\n\n");
			b.append("func (f *").append(adapterName).append(") ").append(samGoName).append('(').append(params).append(") ").append(ret).append(" {\n");
			b.append(ret.isEmpty() ? "\tf.fn(" + args + ")\n" : "\treturn f.fn(" + args + ")\n");
			b.append("}\n\n");
			emitter.fileHelperSource.add(b.toString());
		}
		return adapterName;
	}

	/** `func(params) ret { body }` for a lambda/anonymous-method body (Block or Expression), or
	 * null for an expression body that can't stand as a Go statement. */
	private String funcLiteral(IMethodBinding sig, List<String> paramNames, ASTNode body) {
		ITypeBinding[] types = sig.getParameterTypes();
		List<String> params = new ArrayList<>();
		for (int i = 0; i < types.length; i++) {
			params.add(emitter.sanitizeIdent(paramNames.get(i)) + " " + GoTypes.map(types[i], emitter));
		}
		String ret = emitter.retType(sig);
		Emitter.BodyState saved = emitter.enterFunctionBody(sig.getReturnType());
		String text;
		try {
			text = bodyText(body, ret, sig.getReturnType());
		} finally {
			emitter.exitFunctionBody(saved);
		}
		if (text == null) return null;
		return "func(" + String.join(", ", params) + ") " + ret + " {\n" + text + "}";
	}

	private String bodyText(ASTNode body, String ret, ITypeBinding retType) {
		if (body instanceof Block b) return emitter.block(b, 1);
		Expression e = (Expression) body;
		StringBuilder sb = new StringBuilder();
		String t = emitter.exprInto(e, sb, 1);
		if (!ret.isEmpty()) {
			sb.append(ind(1)).append("return ").append(emitter.adaptNumeric(t, e.resolveTypeBinding(), retType)).append('\n');
			return sb.toString();
		}
		boolean statementShaped = e instanceof MethodInvocation || e instanceof ClassInstanceCreation
				|| e instanceof SuperMethodInvocation;
		if (!statementShaped) return null;
		sb.append(ind(1)).append(t).append('\n');
		return sb.toString();
	}

	// ---------------------------------------------------------------- anonymous classes

	/** Ceiling: an anonymous struct type isn't in the impl cascade - it is only dispatched to
	 * through an interface (listeners/adapters), never through its base class type. */
	String emitAnonymous(ClassInstanceCreation cic) {
		AnonymousClassDeclaration acd = cic.getAnonymousClassDeclaration();
		ITypeBinding anonType = acd.resolveBinding();
		ITypeBinding base = anonType.getSuperclass();
		if (base == null || base.getQualifiedName().equals("java.lang.Object")) {
			base = anonType.getInterfaces().length > 0 ? anonType.getInterfaces()[0] : null;
		}
		List<MethodDeclaration> methods = new ArrayList<>();
		boolean onlyMethods = true;
		for (Object o : acd.bodyDeclarations()) {
			if (o instanceof MethodDeclaration md) methods.add(md);
			else onlyMethods = false;
		}
		if (base == null || !onlyMethods) return marker(cic, "AnonymousClass");
		TypeModel.ClassInfo baseCi = emitter.model.lookup(base);
		boolean runnable = base.getErasure().getQualifiedName().equals("java.lang.Runnable");
		boolean singleMethodIface = baseCi != null && baseCi.isInterface && methods.size() == 1
				&& base.getFunctionalInterfaceMethod() != null;
		if ((runnable || singleMethodIface) && methods.size() == 1) return emitFunctionalAnon(cic, base, anonType, methods.get(0));
		if (baseCi == null) return marker(cic, "AnonymousClass");
		return emitStructAnon(cic, baseCi, anonType, methods);
	}

	private String emitFunctionalAnon(ClassInstanceCreation cic, ITypeBinding base, ITypeBinding anonType, MethodDeclaration md) {
		// Declared first so the body can refer to itself (`timerExec(rate, this)`).
		String v = "anon" + (++emitter.tempCounter);
		String nilFn = "nil";
		String holder = wrap(base, nilFn);
		if (holder == null) return marker(cic, "AnonymousClass");
		emitter.prelude.add(v + " := " + holder);
		String fn = withAnonThis(v, anonType, () -> funcLiteral(md.resolveBinding(), paramNames(md), md.getBody()));
		boolean runnable = base.getErasure().getQualifiedName().equals("java.lang.Runnable");
		addPreludeLines(v + (runnable ? ".Fn = " : ".fn = ") + fn);
		return v;
	}

	private String emitStructAnon(ClassInstanceCreation cic, TypeModel.ClassInfo baseCi, ITypeBinding anonType,
			List<MethodDeclaration> methods) {
		String typeName = emitter.currentClassGoTypeName + "Anon" + (++emitter.anonCounter);
		StringBuilder decl = new StringBuilder("// j2go: anonymous ").append(baseCi.goTypeName).append(" subclass.\n");
		decl.append("type ").append(typeName).append(" struct {\n");
		if (!baseCi.isInterface) decl.append('\t').append(emitter.qualifiedTypeName(baseCi)).append('\n');
		StringBuilder forwarders = new StringBuilder();
		String v = "anon" + (++emitter.tempCounter);
		List<String> assigns = new ArrayList<>();
		for (MethodDeclaration md : methods) {
			IMethodBinding mb = md.resolveBinding();
			IMethodBinding overridden = findOverridden(mb, anonType);
			IMethodBinding sigSource = overridden != null ? overridden : mb;
			String goName = overridden != null ? memberGoName(overridden) : Names.javaMethodBaseGoName(mb.getName());
			String params = emitter.paramList(sigSource, null);
			String ret = emitter.retType(sigSource);
			String field = "fn" + goName;
			decl.append('\t').append(field).append(" func(").append(params).append(") ").append(ret).append('\n');
			List<String> args = new ArrayList<>();
			for (int i = 0; i < sigSource.getParameterTypes().length; i++) args.add("a" + i);
			forwarders.append("func (this *").append(typeName).append(") ").append(goName).append('(').append(params)
					.append(") ").append(ret).append(" {\n\t").append(ret.isEmpty() ? "" : "return ")
					.append("this.").append(field).append('(').append(String.join(", ", args)).append(")\n}\n\n");
			String fn = withAnonThis(v, anonType, () -> funcLiteral(sigSource, paramNames(md), md.getBody()));
			if (fn == null) return marker(cic, "AnonymousClass");
			assigns.add(v + "." + field + " = " + fn);
		}
		decl.append("}\n\n").append(forwarders);
		emitter.fileHelperSource.add(decl.toString());
		emitter.prelude.add(v + " := &" + typeName + "{}");
		if (!baseCi.isInterface) {
			if (!baseCi.root.children.isEmpty()) emitter.prelude.add(v + ".impl = " + v);
			emitter.prelude.add(v + "." + superInitCall(cic, baseCi));
		}
		for (String a : assigns) addPreludeLines(a);
		return v;
	}

	/** init<Base>(args) of the base constructor this anonymous class's implicit one delegates to. */
	private String superInitCall(ClassInstanceCreation cic, TypeModel.ClassInfo baseCi) {
		List<?> args = cic.arguments();
		IMethodBinding ctor = null;
		for (IMethodBinding m : baseCi.binding.getDeclaredMethods()) {
			if (m.isConstructor() && m.getParameterTypes().length == args.size()) ctor = m;
		}
		String init = "init" + baseCi.goFuncPrefix;
		if (ctor == null) return init + "()";
		return emitter.names.goMemberName(ctor, init) + "(" + String.join(", ", emitter.buildArgs(args, ctor)) + ")";
	}

	private IMethodBinding findOverridden(IMethodBinding m, ITypeBinding anonType) {
		List<ITypeBinding> todo = new ArrayList<>();
		if (anonType.getSuperclass() != null) todo.add(anonType.getSuperclass());
		todo.addAll(List.of(anonType.getInterfaces()));
		while (!todo.isEmpty()) {
			ITypeBinding t = todo.remove(0);
			for (IMethodBinding sm : t.getDeclaredMethods()) {
				if (m.overrides(sm)) return sm;
			}
			if (t.getSuperclass() != null) todo.add(t.getSuperclass());
			todo.addAll(List.of(t.getInterfaces()));
		}
		return null;
	}

	/** The Go name a call to m resolves to - the cascade name when m is an override point. */
	private String memberGoName(IMethodBinding m) {
		TypeModel.ClassInfo ci = emitter.model.lookup(m.getDeclaringClass());
		String sig = TypeModel.signature(m);
		if (ci != null && ci.overridePoint(sig) != null) return ci.root.overriddenRootMethodGoNames.get(sig);
		return emitter.names.goMemberName(m, Names.javaMethodBaseGoName(m.getName()));
	}

	private static List<String> paramNames(MethodDeclaration md) {
		List<String> n = new ArrayList<>();
		for (Object p : md.parameters()) n.add(((SingleVariableDeclaration) p).getName().getIdentifier());
		return n;
	}

	private String withAnonThis(String v, ITypeBinding anonType, java.util.function.Supplier<String> body) {
		String savedThis = emitter.anonThis;
		ITypeBinding savedType = emitter.anonType;
		emitter.anonThis = v;
		emitter.anonType = anonType;
		try {
			return body.get();
		} finally {
			emitter.anonThis = savedThis;
			emitter.anonType = savedType;
		}
	}

	// A func literal spans lines; the prelude is a list of lines, each indented by the caller.
	private void addPreludeLines(String text) {
		for (String line : text.split("\n")) emitter.prelude.add(line);
	}
}
