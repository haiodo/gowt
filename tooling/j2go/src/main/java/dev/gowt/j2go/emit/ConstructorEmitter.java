package dev.gowt.j2go.emit;

import dev.gowt.j2go.Manual;
import dev.gowt.j2go.Names;
import dev.gowt.j2go.TypeModel;
import org.eclipse.jdt.core.dom.*;

import java.util.ArrayList;
import java.util.List;

/** Constructors, instance-initializer inlining, super(...)/this(...)/super.method() dispatch -
 * split out of ClassEmitter (same file, moved verbatim) to keep it under the line budget. */
final class ConstructorEmitter {

	private final Emitter emitter;

	ConstructorEmitter(Emitter emitter) {
		this.emitter = emitter;
	}

	// ---------------------------------------------------------------- constructors

	/** New<Class><ParamSuffix> for a constructor; `_`-suffixed when that spells another class's
	 * own constructor (Device(DeviceData data) -> NewDeviceData, DeviceData's no-arg one). */
	String ctorGoName(IMethodBinding ctor, String prefix) {
		String name = emitter.names.goMemberName(ctor, prefix);
		if (name.equals(prefix)) return name;
		String bare = name.substring(name.indexOf('.') + 1);
		for (TypeModel.ClassInfo c : emitter.model.all()) {
			if (bare.equals("New" + c.goFuncPrefix) || bare.equals("new" + c.goFuncPrefix)) return name + "_";
		}
		return name;
	}

	void emitConstructor(MethodDeclaration md, TypeModel.ClassInfo ci, TypeDeclaration td, StringBuilder out) {
		IMethodBinding mb = md.resolveBinding();
		boolean pub = Modifier.isPublic(md.getModifiers());
		String prefix = (pub ? "New" : "new") + ci.goFuncPrefix;
		String goName = emitter.ctorGoName(mb, prefix);
		String initName = emitter.names.goMemberName(mb, "init" + ci.goFuncPrefix); // init methods are always unexported by shape

		boolean needsImpl = !ci.root.children.isEmpty();
		out.append("func ").append(goName).append('(').append(emitter.paramList(mb, md)).append(") *")
				.append(ci.goTypeName).append(" {\n");
		out.append("\tthis := &").append(ci.goTypeName).append("{}\n");
		if (needsImpl) out.append("\tthis.impl = this\n");
		out.append("\tthis.").append(initName).append('(').append(emitter.argNames(md)).append(")\n");
		out.append("\treturn this\n}\n\n");

		out.append("func (this *").append(ci.goTypeName).append(") ").append(initName)
				.append('(').append(emitter.paramList(mb, md)).append(") {\n");
		emitter.currentReturnType = null;
		out.append(emitConstructorBody(md, ci, td));
		out.append("}\n\n");
	}

	/** Java's implicit no-arg constructor (JLS 8.8.9): same New<X>()/init<X>() split as
	 * emitConstructor, so a subclass's explicit ctor can still call this init<X>(). */
	void emitImplicitConstructor(TypeModel.ClassInfo ci, TypeDeclaration td, StringBuilder out) {
		boolean needsImpl = !ci.root.children.isEmpty();
		// JLS 8.8.9: the implicit constructor's own accessibility matches the class's.
		boolean pub = Modifier.isPublic(td.getModifiers());
		String initName = "init" + ci.goFuncPrefix; // init methods are always unexported by shape
		out.append("func ").append(pub ? "New" : "new").append(ci.goFuncPrefix).append("() *").append(ci.goTypeName).append(" {\n");
		out.append("\tthis := &").append(ci.goTypeName).append("{}\n");
		if (needsImpl) out.append("\tthis.impl = this\n");
		out.append("\tthis.").append(initName).append("()\n");
		out.append("\treturn this\n}\n\n");

		out.append("func (this *").append(ci.goTypeName).append(") ").append(initName).append("() {\n");
		out.append(emitZeroArgSuperInitCall(ci));
		out.append(emitInstanceInitializers(td, 1));
		out.append("}\n\n");
	}

	/** "this.Super.initSuper()" (or the manual-superclass equivalent), shared by an implicit
	 * constructor and an explicit one with no super(...)/this(...) of its own. */
	private String emitZeroArgSuperInitCall(TypeModel.ClassInfo ci) {
		if (ci.superclass != null) {
			IMethodBinding zeroArg = findZeroArgCtor(ci.superclass.binding);
			if (zeroArg == null) return "";
			String initName = emitter.names.goMemberName(zeroArg, "init" + ci.superclass.goFuncPrefix);
			return "\tthis." + ci.superclass.goTypeName + "." + initName + "()\n";
		}
		if (ci.manualSuperQualifiedName != null) {
			emitter.addManualImport(ci.manualSuperQualifiedName);
			return "\tthis." + Manual.manualSuperFieldName(ci.manualSuperQualifiedName) + " = "
					+ Manual.ctorFuncName(ci.manualSuperQualifiedName) + "()\n";
		}
		return "";
	}

	private String emitConstructorBody(MethodDeclaration md, TypeModel.ClassInfo ci, TypeDeclaration td) {
		List<Statement> stmts = md.getBody().statements();
		StringBuilder b = new StringBuilder();
		int start = 0;
		boolean delegatesViaThis = !stmts.isEmpty() && stmts.get(0) instanceof ConstructorInvocation;
		if (!stmts.isEmpty() && stmts.get(0) instanceof SuperConstructorInvocation sci) {
			b.append(emitSuperInvocation(sci, ci));
			start = 1;
		} else if (delegatesViaThis) {
			b.append(emitThisInvocation((ConstructorInvocation) stmts.get(0), ci));
			start = 1;
		} else {
			b.append(emitZeroArgSuperInitCall(ci));
		}
		// JLS 8.8.7/12.5: field initializers run right after super(...), never in a this(...)
		// delegate (the ultimately-invoked non-delegating constructor runs them once).
		if (!delegatesViaThis) {
			b.append(emitInstanceInitializers(td, 1));
		}
		for (int i = start; i < stmts.size(); i++) {
			b.append(emitter.stmt(stmts.get(i), 1));
		}
		return b.toString();
	}

	private String emitInstanceInitializers(TypeDeclaration td, int indent) {
		StringBuilder b = new StringBuilder();
		for (Object o : td.bodyDeclarations()) {
			if (o instanceof FieldDeclaration fd && !Modifier.isStatic(fd.getModifiers())) {
				ITypeBinding fieldType = fd.getType().resolveBinding();
				for (Object fo : fd.fragments()) {
					VariableDeclarationFragment f = (VariableDeclarationFragment) fo;
					if (f.getInitializer() == null) continue;
					IVariableBinding vb = f.resolveBinding();
					String init = emitter.adaptNumeric(emitter.exprInto(f.getInitializer(), b, indent),
							f.getInitializer().resolveTypeBinding(), fieldType);
					b.append(EmitUtil.ind(indent)).append("this.").append(emitter.fieldGoName(vb)).append(" = ").append(init).append('\n');
				}
			} else if (o instanceof Initializer init && !Modifier.isStatic(init.getModifiers())) {
				b.append(emitter.block(init.getBody(), indent));
			}
		}
		return b.toString();
	}

	private IMethodBinding findZeroArgCtor(ITypeBinding t) {
		for (IMethodBinding m : t.getDeclaredMethods()) {
			if (m.isConstructor() && m.getParameterTypes().length == 0) return m;
		}
		return null;
	}

	/** super(...)/this(...) build their args before any enclosing emitStatement has set up
	 * emitter.prelude - hoist (a ternary/instanceof argument) directly ahead of the call line. */
	private List<String> hoistArgs(List<?> javaArgs, IMethodBinding mb, StringBuilder out) {
		List<String> saved = emitter.prelude;
		emitter.prelude = new ArrayList<>();
		List<String> args = new ArrayList<>();
		for (Object a : javaArgs) args.add(emitter.adaptArg((Expression) a, mb, javaArgs.indexOf(a)));
		for (String p : emitter.prelude) out.append('\t').append(p).append('\n');
		emitter.prelude = saved;
		return args;
	}

	private String emitSuperInvocation(SuperConstructorInvocation sci, TypeModel.ClassInfo ci) {
		IMethodBinding mb = sci.resolveConstructorBinding();
		// An explicit `super();` with no translated/manual superclass at all is just Java's
		// implicit java.lang.Object() - a real no-op, not a gap (GridData.java has these).
		if (ci.superclass == null && ci.manualSuperQualifiedName == null) return "";
		StringBuilder out = new StringBuilder();
		List<String> args = hoistArgs(sci.arguments(), mb, out);
		if (ci.superclass == null && ci.manualSuperQualifiedName != null) {
			emitter.addManualImport(ci.manualSuperQualifiedName);
			return out + "\tthis." + Manual.manualSuperFieldName(ci.manualSuperQualifiedName) + " = "
					+ Manual.ctorFuncName(ci.manualSuperQualifiedName) + "(" + String.join(", ", args) + ")\n";
		}
		String initName = emitter.names.goMemberName(mb, "init" + ci.superclass.goFuncPrefix);
		return out + "\tthis." + ci.superclass.goTypeName + "." + initName + "(" + String.join(", ", args) + ")\n";
	}

	/** super.method(...): must not resolve back through the impl cascade - calls the specific
	 * ancestor's Go method directly via its embedded-field path (real superclass or manual). */
	String emitSuperMethodInvocation(SuperMethodInvocation smi) {
		IMethodBinding mb = smi.resolveMethodBinding();
		// A collision-renamed cascade method (Names.goMemberName's "On<Class>" suffix) must be
		// called by that same name, even though super.x() bypasses the cascade's .impl dispatch.
		String cascadeName = emitter.currentClassInfo.root.overriddenRootMethodGoNames.get(TypeModel.signature(mb));
		String base = cascadeName != null ? cascadeName : Names.javaMethodBaseGoName(smi.getName().getIdentifier());
		List<String> args = emitter.buildArgs(smi.arguments(), mb);
		String fieldPath;
		if (emitter.currentClassInfo.superclass != null) {
			fieldPath = emitter.currentClassInfo.superclass.goTypeName;
		} else if (emitter.currentClassInfo.manualSuperQualifiedName != null) {
			emitter.addManualImport(emitter.currentClassInfo.manualSuperQualifiedName);
			fieldPath = Manual.manualSuperFieldName(emitter.currentClassInfo.manualSuperQualifiedName);
		} else {
			emitter.unsupported.add("SuperMethodInvocation: " + emitter.currentClassGoTypeName + " has no known superclass");
			return emitter.panicClosure(smi, "unsupported super." + smi.getName().getIdentifier());
		}
		return "this." + fieldPath + "." + base + "(" + String.join(", ", args) + ")";
	}

	private String emitThisInvocation(ConstructorInvocation cti, TypeModel.ClassInfo ci) {
		IMethodBinding mb = cti.resolveConstructorBinding();
		String initName = emitter.names.goMemberName(mb, "init" + ci.goFuncPrefix);
		StringBuilder out = new StringBuilder();
		List<String> args = hoistArgs(cti.arguments(), mb, out);
		return out + "\tthis." + initName + "(" + String.join(", ", args) + ")\n";
	}
}
