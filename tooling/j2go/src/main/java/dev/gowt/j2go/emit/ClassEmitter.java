package dev.gowt.j2go.emit;

import dev.gowt.j2go.Manual;
import dev.gowt.j2go.Names;
import dev.gowt.j2go.TypeModel;
import org.eclipse.jdt.core.dom.*;

import java.util.*;

import static dev.gowt.j2go.emit.EmitUtil.*;

/** Compilation unit layout: classes, interfaces, struct fields, static fields/init, constructors,
 * instance initializers, method shells, super/this invocations. */
final class ClassEmitter {

	private final Emitter emitter;

	ClassEmitter(Emitter emitter) {
		this.emitter = emitter;
	}

	// ---------------------------------------------------------------- classes

	void emitTopLevelClass(TypeDeclaration td, StringBuilder out) {
		if (td.isInterface()) {
			emitInterface(td, out);
			return;
		}
		emitClass(td, out);
	}

	private void emitInterface(TypeDeclaration td, StringBuilder out) {
		TypeModel.ClassInfo ci = emitter.model.lookup(td.resolveBinding());
		String savedClassGoTypeName = emitter.currentClassGoTypeName;
		TypeModel.ClassInfo savedClassInfo = emitter.currentClassInfo;
		emitter.currentClassGoTypeName = ci.goTypeName;
		emitter.currentClassInfo = ci;
		out.append("type ").append(ci.goTypeName).append(" interface {\n");
		for (Object o : td.superInterfaceTypes()) {
			ITypeBinding ib = ((Type) o).resolveBinding();
			TypeModel.ClassInfo superCi = ib == null ? null : emitter.model.lookup(ib);
			// An unresolved/manual super-interface (java.util.EventListener) is a content-free
			// marker in Java - nothing to embed on the Go side, so it is simply dropped.
			if (superCi != null) out.append('\t').append(superCi.goTypeName).append('\n');
		}
		for (Object o : td.bodyDeclarations()) {
			if (o instanceof MethodDeclaration md && !Modifier.isStatic(md.getModifiers())) {
				IMethodBinding mb = md.resolveBinding();
				String goName = emitter.names.goMemberName(mb, Names.capitalize(md.getName().getIdentifier()));
				out.append('\t').append(goName).append('(').append(emitter.paramList(mb, md)).append(") ")
						.append(emitter.retType(mb)).append('\n');
			}
		}
		out.append("}\n\n");
		for (Object o : td.bodyDeclarations()) {
			if (o instanceof MethodDeclaration md && Modifier.isStatic(md.getModifiers())) {
				emitStaticMethod(md, ci, out);
			}
		}
		emitter.currentClassGoTypeName = savedClassGoTypeName;
		emitter.currentClassInfo = savedClassInfo;
	}

	private void emitClass(TypeDeclaration td, StringBuilder out) {
		TypeModel.ClassInfo ci = emitter.model.lookup(td.resolveBinding());
		String savedClassGoTypeName = emitter.currentClassGoTypeName;
		TypeModel.ClassInfo savedClassInfo = emitter.currentClassInfo;
		emitter.currentClassGoTypeName = ci.goTypeName;
		emitter.currentClassInfo = ci;
		boolean needsImpl = !ci.root.children.isEmpty();

		if (ci == ci.root && needsImpl) {
			out.append("type ").append(ci.goTypeName).append("Impl interface {\n");
			for (var e : ci.root.overriddenRootMethods.entrySet()) {
				IMethodBinding rootDecl = e.getValue();
				out.append('\t').append(ci.root.overriddenRootMethodGoNames.get(e.getKey()))
						.append("(").append(emitter.paramList(rootDecl, null)).append(") ")
						.append(dev.gowt.j2go.GoTypes.map(rootDecl.getReturnType(), emitter)).append('\n');
			}
			out.append("}\n\n");
			// Root has no Java declaration when the override point is below it; give it a default too.
			for (var e : ci.root.overriddenRootMethods.entrySet()) {
				if (ci.root.declaredMethods.containsKey(e.getKey())) continue;
				IMethodBinding rootDecl = e.getValue();
				String goName = ci.root.overriddenRootMethodGoNames.get(e.getKey());
				out.append("func (this *").append(ci.goTypeName).append(") ").append(goName)
						.append('(').append(emitter.paramList(rootDecl, null)).append(") ")
						.append(dev.gowt.j2go.GoTypes.map(rootDecl.getReturnType(), emitter)).append(" {\n")
						.append("\tpanic(\"j2go: ").append(goName).append(" has no default on ")
						.append(ci.goTypeName).append("\")\n}\n\n");
			}
		}

		out.append("type ").append(ci.goTypeName).append(" struct {\n");
		if (ci.superclass != null) {
			out.append('\t').append(ci.superclass.goTypeName).append('\n');
		} else if (ci.manualSuperQualifiedName != null) {
			emitter.addManualImport(ci.manualSuperQualifiedName);
			out.append('\t').append(Manual.goTypeName(ci.manualSuperQualifiedName)).append('\n');
		}
		Set<String> methodGoNames = collectMethodGoNames(td);
		for (Object o : td.bodyDeclarations()) {
			if (o instanceof FieldDeclaration fd && !Modifier.isStatic(fd.getModifiers())) {
				emitStructFields(fd, methodGoNames, out);
			}
		}
		if (ci == ci.root && needsImpl) {
			out.append("\tImpl ").append(ci.goTypeName).append("Impl\n");
		}
		out.append("}\n\n");

		for (Object o : td.bodyDeclarations()) {
			if (o instanceof FieldDeclaration fd && Modifier.isStatic(fd.getModifiers())) {
				emitStaticFields(fd, ci, out);
			}
		}
		for (Object o : td.bodyDeclarations()) {
			if (o instanceof Initializer init && Modifier.isStatic(init.getModifiers())) {
				emitStaticInitBlock(init);
			}
		}
		boolean hasExplicitCtor = false;
		for (Object o : td.bodyDeclarations()) {
			if (o instanceof MethodDeclaration md && md.isConstructor()) {
				emitConstructor(md, ci, td, out);
				hasExplicitCtor = true;
			}
		}
		// A class with no declared constructor (e.g. Event.java: field declarations only) still
		// gets Java's implicit public no-arg one - "new X()" elsewhere needs a matching New<X>().
		if (!hasExplicitCtor && !ci.isStruct) emitImplicitConstructor(ci, td, out);
		for (Object o : td.bodyDeclarations()) {
			if (o instanceof MethodDeclaration md && !md.isConstructor() && !Modifier.isStatic(md.getModifiers())) {
				if (Manual.manualMethod(Names.erasureKey(md.resolveBinding())) != null) continue;
				// Abstract method (no body, e.g. Layout.computeSize): nothing to emit at this
				// declaring class - only a concrete override further down has a real body.
				if (md.getBody() == null) continue;
				// A generic method (e.g. getTypedListeners<L>) has no Go equivalent for its own
				// signature (no generics, no Stream) - skipped entirely rather than emitted broken.
				if (!md.typeParameters().isEmpty()) {
					emitter.unsupported.add("MethodDeclaration: generic method " + ci.binaryName + "." + md.getName() + " skipped");
					continue;
				}
				if (Modifier.isNative(md.getModifiers())) emitter.emitUnsupportedNative(md, ci, out);
				else emitInstanceMethod(md, ci, out);
			}
		}
		for (Object o : td.bodyDeclarations()) {
			if (o instanceof MethodDeclaration md && !md.isConstructor() && Modifier.isStatic(md.getModifiers())) {
				IMethodBinding smb = md.resolveBinding();
				String declClass = smb.getDeclaringClass().getErasure().getQualifiedName();
				if (Manual.isSkippedMethod(declClass, smb.getName())) continue;
				if (Modifier.isNative(md.getModifiers())) emitter.emitStaticNativeMethod(md, ci, out);
				else emitStaticMethod(md, ci, out);
			}
		}
		for (Object o : td.bodyDeclarations()) {
			if (o instanceof TypeDeclaration nested) {
				emitClass(nested, out);
			}
		}
		emitter.currentClassGoTypeName = savedClassGoTypeName;
	}

	private Set<String> collectMethodGoNames(TypeDeclaration td) {
		Set<String> s = new HashSet<>();
		for (Object o : td.bodyDeclarations()) {
			if (o instanceof MethodDeclaration md && !md.isConstructor() && !Modifier.isStatic(md.getModifiers())) {
				s.add(emitter.names.goMemberName(md.resolveBinding(), Names.capitalize(md.getName().getIdentifier())));
			}
		}
		return s;
	}

	private void emitStructFields(FieldDeclaration fd, Set<String> methodGoNames, StringBuilder out) {
		boolean pub = Modifier.isPublic(fd.getModifiers());
		List<String> goNames = new ArrayList<>();
		for (Object o : fd.fragments()) {
			VariableDeclarationFragment f = (VariableDeclarationFragment) o;
			String javaName = f.getName().getIdentifier();
			if (javaName.equals("serialVersionUID")) continue;
			String goName = pub ? Names.capitalize(javaName) : javaName;
			if (pub && methodGoNames.contains(goName)) {
				goName = goName + "_";
				emitter.unsupported.add("FieldMethodNameClash: " + javaName + " clashes with a method Go name, suffixed _");
			}
			goNames.add(goName);
		}
		if (goNames.isEmpty()) return;
		out.append('\t').append(String.join(", ", goNames)).append(' ')
				.append(dev.gowt.j2go.GoTypes.map(fd.getType().resolveBinding(), emitter)).append('\n');
	}

	private void emitStaticFields(FieldDeclaration fd, TypeModel.ClassInfo ci, StringBuilder out) {
		boolean isFinal = Modifier.isFinal(fd.getModifiers());
		ITypeBinding type = fd.getType().resolveBinding();
		boolean maybeConst = isFinal && (type.isPrimitive() || type.getQualifiedName().equals("java.lang.String"));
		for (Object o : fd.fragments()) {
			VariableDeclarationFragment f = (VariableDeclarationFragment) o;
			String javaName = f.getName().getIdentifier();
			if (javaName.equals("serialVersionUID")) continue;
			if (Manual.isSkippedField(ci.binding.getErasure().getQualifiedName(), javaName)) continue;
			String goName = ci.goFuncPrefix + Names.capitalize(javaName);
			if (staticFieldClashesWithMethod(ci.binding, javaName)) {
				goName = goName + "_";
				emitter.unsupported.add("StaticFieldMethodNameClash: " + javaName + " clashes with a static method Go name, suffixed _");
			}
			Expression initExpr = f.getInitializer();
			if (initExpr == null) {
				out.append("var ").append(goName).append(" ").append(dev.gowt.j2go.GoTypes.map(type, emitter))
						.append(" = ").append(emitter.zeroValue(type)).append('\n');
				continue;
			}
			// A call (native not wired up yet) or prelude (ternary/instanceof hoisting) can't
			// live in a bare `var X = expr` - move it into func init() instead.
			List<String> saved = emitter.prelude;
			emitter.prelude = new ArrayList<>();
			String text = emitter.expr(initExpr);
			List<String> myPrelude = emitter.prelude;
			emitter.prelude = saved;
			if (!myPrelude.isEmpty() || emitter.containsCall(initExpr)) {
				out.append("var ").append(goName).append(" ").append(dev.gowt.j2go.GoTypes.map(type, emitter)).append('\n');
				StringBuilder b = new StringBuilder();
				for (String p : myPrelude) b.append('\t').append(p).append('\n');
				b.append('\t').append(goName).append(" = ").append(text).append('\n');
				emitter.deferredStaticInits.add(b.toString());
				emitter.deferredStaticInitLabels.add(goName);
				continue;
			}
			out.append(maybeConst ? "const " : "var ").append(goName).append(" ")
					.append(dev.gowt.j2go.GoTypes.map(type, emitter)).append(" = ").append(text).append('\n');
		}
		out.append('\n');
	}

	// static {} block: folded into the same deferred func init() as static field assignments
	// (see README) so it runs after the class_x/sel_x fields it reads, not its own func init().
	private void emitStaticInitBlock(Initializer init) {
		emitter.currentReturnType = null;
		emitter.deferredStaticInits.add(emitter.block(init.getBody(), 1));
		emitter.deferredStaticInitLabels.add(emitter.currentClassGoTypeName + " static{}");
	}

	// ---------------------------------------------------------------- constructors

	private void emitConstructor(MethodDeclaration md, TypeModel.ClassInfo ci, TypeDeclaration td, StringBuilder out) {
		IMethodBinding mb = md.resolveBinding();
		boolean pub = Modifier.isPublic(md.getModifiers());
		String prefix = (pub ? "New" : "new") + ci.goFuncPrefix;
		String goName = emitter.names.goMemberName(mb, prefix);
		String initName = emitter.names.goMemberName(mb, "init" + ci.goFuncPrefix); // init methods are always unexported by shape

		boolean needsImpl = !ci.root.children.isEmpty();
		out.append("func ").append(goName).append('(').append(emitter.paramList(mb, md)).append(") *")
				.append(ci.goTypeName).append(" {\n");
		out.append("\tthis := &").append(ci.goTypeName).append("{}\n");
		if (needsImpl) out.append("\tthis.Impl = this\n");
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
	private void emitImplicitConstructor(TypeModel.ClassInfo ci, TypeDeclaration td, StringBuilder out) {
		boolean needsImpl = !ci.root.children.isEmpty();
		// JLS 8.8.9: the implicit constructor's own accessibility matches the class's.
		boolean pub = Modifier.isPublic(td.getModifiers());
		String initName = "init" + ci.goFuncPrefix; // init methods are always unexported by shape
		out.append("func ").append(pub ? "New" : "new").append(ci.goFuncPrefix).append("() *").append(ci.goTypeName).append(" {\n");
		out.append("\tthis := &").append(ci.goTypeName).append("{}\n");
		if (needsImpl) out.append("\tthis.Impl = this\n");
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
					b.append(ind(indent)).append("this.").append(emitter.fieldGoName(vb)).append(" = ").append(init).append('\n');
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

	private String emitSuperInvocation(SuperConstructorInvocation sci, TypeModel.ClassInfo ci) {
		IMethodBinding mb = sci.resolveConstructorBinding();
		List<String> args = new ArrayList<>();
		for (Object a : sci.arguments()) {
			args.add(emitter.adaptArg((Expression) a, mb, sci.arguments().indexOf(a)));
		}
		if (ci.superclass == null && ci.manualSuperQualifiedName != null) {
			emitter.addManualImport(ci.manualSuperQualifiedName);
			return "\tthis." + Manual.manualSuperFieldName(ci.manualSuperQualifiedName) + " = "
					+ Manual.ctorFuncName(ci.manualSuperQualifiedName) + "(" + String.join(", ", args) + ")\n";
		}
		String initName = emitter.names.goMemberName(mb, "init" + ci.superclass.goFuncPrefix);
		return "\tthis." + ci.superclass.goTypeName + "." + initName + "(" + String.join(", ", args) + ")\n";
	}

	/** super.method(...): must not resolve back through the impl cascade - calls the specific
	 * ancestor's Go method directly via its embedded-field path (real superclass or manual). */
	String emitSuperMethodInvocation(SuperMethodInvocation smi) {
		IMethodBinding mb = smi.resolveMethodBinding();
		// A collision-renamed cascade method (Names.goMemberName's "On<Class>" suffix) must be
		// called by that same name, even though super.x() bypasses the cascade's .Impl dispatch.
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
		List<String> args = new ArrayList<>();
		for (Object a : cti.arguments()) {
			args.add(emitter.adaptArg((Expression) a, mb, cti.arguments().indexOf(a)));
		}
		return "\tthis." + initName + "(" + String.join(", ", args) + ")\n";
	}

	// ---------------------------------------------------------------- methods

	private void emitInstanceMethod(MethodDeclaration md, TypeModel.ClassInfo ci, StringBuilder out) {
		IMethodBinding mb = md.resolveBinding();
		String javaName = md.getName().getIdentifier();
		String base = Names.javaMethodBaseGoName(javaName);
		String sig = TypeModel.signature(mb);
		TypeModel.ClassInfo overridePoint = ci.overridePoint(sig);
		boolean overridden = overridePoint != null;
		IMethodBinding sigSource = overridden ? overridePoint.declaredBinding(sig) : mb;
		String goName = overridden ? ci.root.overriddenRootMethodGoNames.get(sig) : emitter.names.goMemberName(mb, base);

		out.append("func (this *").append(ci.goTypeName).append(") ").append(goName)
				.append('(').append(emitter.paramList(sigSource, md)).append(") ")
				.append(emitter.retType(sigSource)).append(" {\n");
		emitter.currentReturnType = sigSource.getReturnType();
		out.append(emitter.block(md.getBody(), 1));
		emitter.currentReturnType = null;
		out.append("}\n\n");
	}

	private void emitStaticMethod(MethodDeclaration md, TypeModel.ClassInfo ci, StringBuilder out) {
		IMethodBinding mb = md.resolveBinding();
		String goName = staticMethodGoName(mb, ci);
		out.append("func ").append(goName).append('(').append(emitter.paramList(mb, md)).append(") ")
				.append(emitter.retType(mb)).append(" {\n");
		emitter.currentReturnType = mb.getReturnType();
		out.append(emitter.block(md.getBody(), 1));
		emitter.currentReturnType = null;
		out.append("}\n\n");
	}

	// A static method's Go name can coincidentally spell out another translated type's own name
	// (SWT.error -> "SWTError", colliding with class SWTError) - Go rejects that, so disambiguate.
	String staticMethodGoName(IMethodBinding mb, TypeModel.ClassInfo ci) {
		String goName = emitter.qualifiedFuncPrefix(ci) + emitter.names.goMemberName(mb, Names.capitalize(mb.getName()));
		return collidesWithTypeName(goName) ? goName + "Fn" : goName;
	}

	private boolean collidesWithTypeName(String name) {
		for (TypeModel.ClassInfo c : emitter.model.all()) {
			if (c.goTypeName.equals(name)) return true;
		}
		return false;
	}

}
