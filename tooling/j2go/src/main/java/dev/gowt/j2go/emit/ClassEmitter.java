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
			if (o instanceof MethodDeclaration md && Modifier.isDefault(md.getModifiers())) {
				emitDefaultMethod(md, ci, out);
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
		if (EmitUtil.isInnerClass(ci.binding)) {
			out.append('\t').append(EmitUtil.OUTER_FIELD).append(' ').append(dev.gowt.j2go.GoTypes.map(ci.binding.getDeclaringClass(), emitter)).append('\n');
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
		// Another package's instanceof/cast reads the dynamic type through this (cocoa's id has a
		// hand-written one): the impl field itself is unexported.
		if (ci == ci.root && needsImpl && ci.splitsDispatch()) {
			out.append("func (this *").append(ci.goTypeName).append(") Impl() ").append(ci.goTypeName)
					.append("Impl { return this.impl }\n\n");
		}
		if (ci.asMethodName != null) emitLikeAccessor(ci, out);

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
				// Abstract method (no body, e.g. Layout.computeSize): at most its exported wrapper -
				// only a concrete override further down has a real body.
				if (md.getBody() == null) {
					emitDispatchWrapper(md, ci, out);
					continue;
				}
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
		out.append(emitter.defaultForwarders(ci.binding, "*" + ci.goTypeName));
		for (Object o : td.bodyDeclarations()) {
			if (o instanceof TypeDeclaration nested && !Manual.isManual(nested.resolveBinding().getErasure().getQualifiedName())) {
				emitClass(nested, out);
			}
		}
		emitter.currentClassGoTypeName = savedClassGoTypeName;
		emitter.currentClassInfo = savedClassInfo;
	}

	/** func (this *C) AsC() *C { return this } + type CLike interface { AsC() *C } (README
	 * "Round 7 api") - every subclass satisfies CLike too, promoted through its embedded C. */
	private void emitLikeAccessor(TypeModel.ClassInfo ci, StringBuilder out) {
		out.append("func (this *").append(ci.goTypeName).append(") ").append(ci.asMethodName)
				.append("() *").append(ci.goTypeName).append(" { return this }\n\n");
		out.append("type ").append(ci.likeInterfaceName).append(" interface {\n\t")
				.append(ci.asMethodName).append("() *").append(ci.goTypeName).append("\n}\n\n");
	}

	/** Forwarders to <Iface>Default<M> for every interface default method `type` (a class, an
	 * anonymous class, or the interface itself for its Func adapter) doesn't implement. */
	String defaultForwarders(ITypeBinding type, String goRecvType) {
		StringBuilder b = new StringBuilder();
		Set<String> seen = new HashSet<>();
		List<ITypeBinding> todo = new ArrayList<>(type.isInterface() ? List.of(type) : List.of(type.getInterfaces()));
		while (!todo.isEmpty()) {
			ITypeBinding iface = todo.remove(0).getErasure();
			todo.addAll(List.of(iface.getInterfaces()));
			TypeModel.ClassInfo ici = emitter.model.lookup(iface);
			if (ici == null) continue;
			for (IMethodBinding m : iface.getDeclaredMethods()) {
				// An abstract class leaving an interface method to its subclasses still has to satisfy
				// the Go interface itself (its default forwarders pass `this`): a panicking stub.
				boolean abstractGap = !type.isInterface() && Modifier.isAbstract(type.getModifiers())
						&& Modifier.isAbstract(m.getModifiers()) && !Modifier.isStatic(m.getModifiers());
				if (!Modifier.isDefault(m.getModifiers()) && !abstractGap || !seen.add(m.getName() + TypeModel.signature(m))) continue;
				if (!type.isInterface() && implementsIn(type, m) || abstractGap && declaresIn(type, m)) continue;
				String goName = emitter.names.goMemberName(m, Names.capitalize(m.getName()));
				if (abstractGap) {
					b.append("func (this ").append(goRecvType).append(") ").append(goName).append('(').append(emitter.paramList(m, null))
							.append(") ").append(emitter.retType(m)).append(" {\n\tpanic(\"j2go: abstract ").append(goName).append("\")\n}\n\n");
					continue;
				}
				List<String> args = new ArrayList<>(List.of("this"));
				for (int i = 0; i < m.getParameterTypes().length; i++) args.add("a" + i);
				String ret = emitter.retType(m);
				b.append("func (this ").append(goRecvType).append(") ").append(goName).append('(').append(emitter.paramList(m, null))
						.append(") ").append(ret).append(" {\n\t").append(ret.isEmpty() ? "" : "return ")
						.append(emitter.qualify(ici.goFuncPrefix + "Default" + goName, ici)).append('(').append(String.join(", ", args)).append(")\n}\n\n");
			}
		}
		return b.toString();
	}

	private static boolean declaresIn(ITypeBinding type, IMethodBinding m) {
		for (ITypeBinding t = type; t != null; t = t.getSuperclass()) {
			for (IMethodBinding dm : t.getDeclaredMethods()) {
				if (dm.overrides(m) || dm.isEqualTo(m)) return true;
			}
		}
		return false;
	}

	private static boolean implementsIn(ITypeBinding type, IMethodBinding m) {
		for (ITypeBinding t = type; t != null; t = t.getSuperclass()) {
			for (IMethodBinding dm : t.getDeclaredMethods()) {
				if (!Modifier.isAbstract(dm.getModifiers()) && (dm.overrides(m) || dm.isEqualTo(m))) return true;
			}
		}
		return false;
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
			String goName = pub ? Names.capitalize(javaName) : EmitUtil.fieldIdent(javaName);
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
			String goName = staticFieldGoName(emitter, ci, javaName);
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

		// A cascade override, a Java-interface implementation, or a Type::method reference target
		// has a signature fixed elsewhere - only a plain method (or the wrapper) widens its params.
		boolean widen = !overridden && !implementsInterfaceMethod(mb) && !emitter.model.isMethodReferenceTarget(mb);
		emitDispatchWrapper(md, ci, out);
		List<String> pubPrelude = new ArrayList<>();
		String params = widen ? publicParamList(emitter, sigSource, md, pubPrelude) : emitter.paramList(sigSource, md);
		out.append("func (this *").append(ci.goTypeName).append(") ").append(goName)
				.append('(').append(params).append(") ")
				.append(emitter.retType(sigSource)).append(" {\n");
		for (String p : pubPrelude) out.append('\t').append(p).append('\n');
		emitter.currentReturnType = sigSource.getReturnType();
		out.append(emitter.block(md.getBody(), 1));
		emitter.currentReturnType = null;
		out.append("}\n\n");
	}

	/** At a split cascade's override point: the exported natural-name method, params widened and
	 * converted once, dispatching through this.impl (README "Round 9 api"). */
	private void emitDispatchWrapper(MethodDeclaration md, TypeModel.ClassInfo ci, StringBuilder out) {
		IMethodBinding mb = md.resolveBinding();
		String sig = TypeModel.signature(mb);
		if (!ci.splitsDispatch() || ci.overridePoint(sig) != ci) return;
		String natural = emitter.names.goMemberName(mb, Names.javaMethodBaseGoName(mb.getName()));
		boolean widen = !implementsInterfaceMethod(mb) && !emitter.model.isMethodReferenceTarget(mb);
		List<String> pre = new ArrayList<>();
		String params = widen ? publicParamList(emitter, mb, md, pre) : emitter.paramList(mb, md);
		String ret = emitter.retType(mb);
		out.append("func (this *").append(ci.goTypeName).append(") ").append(natural)
				.append('(').append(params).append(") ").append(ret).append(" {\n");
		for (String p : pre) {
			if (!p.startsWith("_ = ")) out.append('\t').append(p).append('\n');
		}
		out.append('\t').append(ret.isEmpty() ? "" : "return ").append("this.impl.")
				.append(ci.root.overriddenRootMethodGoNames.get(sig)).append('(').append(emitter.argNames(md)).append(")\n}\n\n");
	}

	// A default method's body as <Iface>Default<M>(this, ...); implementers that don't declare it
	// get a forwarding method (Emitter.defaultForwarders), so the Go interface lists it as usual.
	private void emitDefaultMethod(MethodDeclaration md, TypeModel.ClassInfo ci, StringBuilder out) {
		IMethodBinding mb = md.resolveBinding();
		String goName = emitter.names.goMemberName(mb, Names.capitalize(md.getName().getIdentifier()));
		String params = emitter.paramList(mb, md);
		out.append("func ").append(ci.goFuncPrefix).append("Default").append(goName).append("(this ").append(ci.goTypeName)
				.append(params.isEmpty() ? "" : ", " + params).append(") ").append(emitter.retType(mb)).append(" {\n");
		emitter.currentReturnType = mb.getReturnType();
		out.append(emitter.block(md.getBody(), 1));
		emitter.currentReturnType = null;
		out.append("}\n\n");
	}

	private void emitStaticMethod(MethodDeclaration md, TypeModel.ClassInfo ci, StringBuilder out) {
		IMethodBinding mb = md.resolveBinding();
		String goName = staticMethodGoName(mb, ci);
		List<String> pubPrelude = new ArrayList<>();
		String params = publicParamList(emitter, mb, md, pubPrelude);
		out.append("func ").append(goName).append('(').append(params).append(") ")
				.append(emitter.retType(mb)).append(" {\n");
		for (String p : pubPrelude) out.append('\t').append(p).append('\n');
		emitter.currentReturnType = mb.getReturnType();
		out.append(emitter.block(md.getBody(), 1));
		emitter.currentReturnType = null;
		out.append("}\n\n");
	}

	String staticMethodGoName(IMethodBinding mb, TypeModel.ClassInfo ci) {
		String bareMember = emitter.names.goMemberName(mb, Names.capitalize(mb.getName()));
		return EmitUtil.staticMethodGoName(emitter, ci, bareMember);
	}

}
