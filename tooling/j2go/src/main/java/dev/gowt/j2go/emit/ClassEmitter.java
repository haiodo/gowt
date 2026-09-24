package dev.gowt.j2go.emit;

import dev.gowt.j2go.Manual;
import dev.gowt.j2go.Names;
import dev.gowt.j2go.TypeModel;
import org.eclipse.jdt.core.dom.*;

import java.util.*;

import static dev.gowt.j2go.emit.EmitUtil.*;

/** Compilation unit layout: classes, interfaces, struct fields, static fields/init, method
 * shells. Constructors/instance-initializer/super-this dispatch live in ConstructorEmitter. */
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
			// Root gets a default when it has no Java declaration for this signature at all, or
			// only an abstract one (no body, e.g. Layout.computeSize) - either way nothing concrete
			// ends up on the root's own Go type.
			for (var e : ci.root.overriddenRootMethods.entrySet()) {
				IMethodBinding rootOwnDecl = ci.root.declaredMethods.get(e.getKey());
				if (rootOwnDecl != null && !Modifier.isAbstract(rootOwnDecl.getModifiers())) continue;
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
			out.append("\timpl ").append(ci.goTypeName).append("Impl\n");
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
				emitter.emitConstructor(md, ci, td, out);
				hasExplicitCtor = true;
			}
		}
		// A class with no declared constructor (e.g. Event.java: field declarations only) still
		// gets Java's implicit public no-arg one - "new X()" elsewhere needs a matching New<X>().
		if (!hasExplicitCtor && !ci.isStruct) emitter.emitImplicitConstructor(ci, td, out);
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
				if (Manual.manualMethod(Names.erasureKey(smb)) != null) continue;
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
		emitter.currentClassInfo = savedClassInfo;
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
		for (Object o : fd.fragments()) {
			VariableDeclarationFragment f = (VariableDeclarationFragment) o;
			String javaName = f.getName().getIdentifier();
			if (javaName.equals("serialVersionUID")) continue;
			String goName = pub ? Names.capitalize(javaName) : javaName;
			if (pub && methodGoNames.contains(goName)) {
				goName = goName + "_";
				emitter.unsupported.add("FieldMethodNameClash: " + javaName + " clashes with a method Go name, suffixed _");
			}
			// Per fragment: a C-style `Runnable timerList []` puts the [] on the fragment, not the type.
			ITypeBinding t = f.resolveBinding() != null ? f.resolveBinding().getType() : fd.getType().resolveBinding();
			out.append('\t').append(goName).append(' ').append(dev.gowt.j2go.GoTypes.map(t, emitter)).append('\n');
		}
	}

	private void emitStaticFields(FieldDeclaration fd, TypeModel.ClassInfo ci, StringBuilder out) {
		boolean isFinal = Modifier.isFinal(fd.getModifiers());
		for (Object o : fd.fragments()) {
			VariableDeclarationFragment f = (VariableDeclarationFragment) o;
			ITypeBinding type = f.resolveBinding() != null ? f.resolveBinding().getType() : fd.getType().resolveBinding();
			boolean maybeConst = isFinal && (type.isPrimitive() || type.getQualifiedName().equals("java.lang.String"));
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
			text = emitter.adaptNumeric(text, initExpr.resolveTypeBinding(), type);
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
