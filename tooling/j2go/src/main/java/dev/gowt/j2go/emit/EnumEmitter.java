package dev.gowt.j2go.emit;

import dev.gowt.j2go.Manual;
import dev.gowt.j2go.Names;
import dev.gowt.j2go.TypeModel;
import org.eclipse.jdt.core.dom.*;

import java.util.*;

/** Enums and @interface types (README "Enums"). An enum is a pointer singleton per constant: the struct holds
 * name, ordinal and the Java fields, and a constant body's overrides live in func fields of that one struct. */
final class EnumEmitter {

	static final String NAME_FIELD = "enumName";
	static final String ORDINAL_FIELD = "enumOrdinal";
	/** Go names of java.lang.Enum's final methods, which every enum struct defines. */
	static final Set<String> FINAL_METHODS = Set.of("Name", "Ordinal", "CompareTo");
	static final String NAME_PARAMS = NAME_FIELD + " string, " + ORDINAL_FIELD + " int32";

	/** A method some constant body overrides: target is the overridden declaration, declared the enum's own MethodDeclaration of it or null. */
	private record Dispatch(IMethodBinding target, MethodDeclaration declared, String goName, String field) {}

	private final Emitter emitter;
	private final ClassEmitter classEmitter;

	EnumEmitter(Emitter emitter, ClassEmitter classEmitter) {
		this.emitter = emitter;
		this.classEmitter = classEmitter;
	}

	// ---------------------------------------------------------------- @interface

	/** Ceiling: an annotation type is only its members as a Go interface, so references compile; defaults and retention are task 23's. */
	void emitAnnotation(AnnotationTypeDeclaration ad, StringBuilder out) {
		TypeModel.ClassInfo ci = emitter.model.lookup(ad.resolveBinding());
		out.append("type ").append(ci.goTypeName).append(" interface {\n");
		for (Object o : ad.bodyDeclarations()) {
			if (!(o instanceof AnnotationTypeMemberDeclaration m)) continue;
			IMethodBinding mb = m.resolveBinding();
			out.append('\t').append(emitter.names.goMemberName(mb, Names.capitalize(m.getName().getIdentifier())))
					.append("() ").append(emitter.retType(mb)).append('\n');
		}
		out.append("}\n\n");
		for (Object o : ad.bodyDeclarations()) {
			if (o instanceof AbstractTypeDeclaration nested) classEmitter.emitTopLevelClass(nested, out);
		}
	}

	// ---------------------------------------------------------------- enum

	void emitEnum(EnumDeclaration ed, StringBuilder out) {
		TypeModel.ClassInfo ci = emitter.model.lookup(ed.resolveBinding());
		String savedName = emitter.currentClassGoTypeName;
		TypeModel.ClassInfo savedInfo = emitter.currentClassInfo;
		emitter.currentClassGoTypeName = ci.goTypeName;
		emitter.currentClassInfo = ci;
		Map<String, Dispatch> dispatch = dispatchMethods(ed);
		emitStruct(ed, ci, dispatch, out);
		emitBuiltins(ci, dispatch, out);
		emitConstants(ed, ci, dispatch, out);
		for (Object o : ed.bodyDeclarations()) {
			if (o instanceof FieldDeclaration fd && Modifier.isStatic(fd.getModifiers())) classEmitter.emitStaticFields(fd, ci, out);
			if (o instanceof Initializer init && Modifier.isStatic(init.getModifiers())) classEmitter.emitStaticInitBlock(init);
		}
		boolean hasCtor = false;
		for (Object o : ed.bodyDeclarations()) {
			if (o instanceof MethodDeclaration md && md.isConstructor()) {
				emitter.emitConstructor(md, ci, ed, out);
				hasCtor = true;
			}
		}
		if (!hasCtor) emitter.emitImplicitConstructor(ci, ed, out);
		emitMethods(ed, ci, dispatch, out);
		out.append(emitter.defaultForwarders(ci.binding, "*" + ci.goTypeName));
		for (Object o : ed.bodyDeclarations()) {
			if (o instanceof AbstractTypeDeclaration nested && !(nested instanceof RecordDeclaration)) classEmitter.emitTopLevelClass(nested, out);
		}
		emitter.currentClassGoTypeName = savedName;
		emitter.currentClassInfo = savedInfo;
	}

	private void emitStruct(EnumDeclaration ed, TypeModel.ClassInfo ci, Map<String, Dispatch> dispatch, StringBuilder out) {
		// Name, Ordinal and CompareTo are final in java.lang.Enum: a public field of that Go name gets the usual "_".
		Set<String> methodGoNames = classEmitter.collectMethodGoNames(ed);
		methodGoNames.addAll(FINAL_METHODS);
		out.append("type ").append(ci.goTypeName).append(" struct {\n\t").append(NAME_FIELD).append(" string\n\t")
				.append(ORDINAL_FIELD).append(" int32\n");
		for (Object o : ed.bodyDeclarations()) {
			if (o instanceof FieldDeclaration fd && !Modifier.isStatic(fd.getModifiers())) classEmitter.emitStructFields(fd, methodGoNames, out);
		}
		for (Dispatch d : dispatch.values()) {
			out.append('\t').append(d.field()).append(" func(").append(emitter.paramList(d.target(), null)).append(") ")
					.append(emitter.retType(d.target())).append('\n');
		}
		out.append("}\n\n");
	}

	private void emitBuiltins(TypeModel.ClassInfo ci, Map<String, Dispatch> dispatch, StringBuilder out) {
		String t = ci.goTypeName;
		out.append("func (this *").append(t).append(") Name() string { return this.").append(NAME_FIELD).append(" }\n\n");
		out.append("func (this *").append(t).append(") Ordinal() int32 { return this.").append(ORDINAL_FIELD).append(" }\n\n");
		out.append("func (this *").append(t).append(") CompareTo(o *").append(t).append(") int32 { return this.")
				.append(ORDINAL_FIELD).append(" - o.").append(ORDINAL_FIELD).append(" }\n\n");
		if (!ci.declaredMethods.containsKey("toString()") && !dispatch.containsKey("toString()")) {
			out.append("func (this *").append(t).append(") String() string { return this.").append(NAME_FIELD).append(" }\n\n");
		}
	}

	private void emitConstants(EnumDeclaration ed, TypeModel.ClassInfo ci, Map<String, Dispatch> dispatch, StringBuilder out) {
		List<String> goNames = new ArrayList<>();
		int ordinal = 0;
		for (Object o : ed.enumConstants()) {
			EnumConstantDeclaration ecd = (EnumConstantDeclaration) o;
			String goName = EmitUtil.staticFieldGoName(emitter, ci, ecd.getName().getIdentifier());
			goNames.add(goName);
			emitConstant(ed, ecd, ci, goName, ordinal++, dispatch, out);
		}
		String values = Names.decapitalize(ci.goFuncPrefix) + "Values";
		out.append("var ").append(values).append(" = []*").append(ci.goTypeName).append('{').append(String.join(", ", goNames)).append("}\n\n");
		out.append("func ").append(EmitUtil.staticMethodGoName(emitter, ci, "Values")).append("() []*").append(ci.goTypeName)
				.append(" {\n\treturn append([]*").append(ci.goTypeName).append("(nil), ").append(values).append("...)\n}\n\n");
		emitter.fileImports.add(Manual.JRT_IMPORT);
		out.append("func ").append(EmitUtil.staticMethodGoName(emitter, ci, "ValueOf")).append("(name string) *").append(ci.goTypeName).append(" {\n")
				.append("\tfor _, v := range ").append(values).append(" {\n\t\tif v.").append(NAME_FIELD).append(" == name {\n\t\t\treturn v\n\t\t}\n\t}\n")
				.append("\tpanic(jrt.NewIllegalArgumentException(\"No enum constant ").append(ci.binding.getQualifiedName()).append(".\" + name))\n}\n\n");
	}

	/** `var C = newE("C", ordinal, args)`; a constant with a body, or arguments that need statements, is built in a closure. */
	private void emitConstant(EnumDeclaration ed, EnumConstantDeclaration ecd, TypeModel.ClassInfo ci, String goName, int ordinal,
			Map<String, Dispatch> dispatch, StringBuilder out) {
		IMethodBinding ctor = ecd.resolveConstructorBinding();
		List<String> saved = emitter.prelude;
		emitter.prelude = new ArrayList<>();
		List<String> args = new ArrayList<>(List.of("\"" + ecd.getName().getIdentifier() + "\"", Integer.toString(ordinal)));
		args.addAll(emitter.buildArgs(ecd.arguments(), ctor));
		List<String> prelude = emitter.prelude;
		emitter.prelude = saved;
		String call = emitter.ctorGoName(ctor, "new" + ci.goFuncPrefix) + "(" + String.join(", ", args) + ")";
		AnonymousClassDeclaration body = ecd.getAnonymousClassDeclaration();
		if (prelude.isEmpty() && body == null) {
			out.append("var ").append(goName).append(" = ").append(call).append("\n\n");
			return;
		}
		out.append("var ").append(goName).append(" = func() *").append(ci.goTypeName).append(" {\n");
		for (String p : prelude) out.append('\t').append(p).append('\n');
		out.append("\tthis := ").append(call).append('\n');
		if (body != null) {
			for (Object o : body.bodyDeclarations()) {
				if (!(o instanceof MethodDeclaration md)) {
					emitter.unsupported.add("EnumConstantBody: " + ecd.getName() + " member other than a method skipped");
					continue;
				}
				Dispatch d = overriddenIn(md, ed);
				if (d == null) {
					emitter.unsupported.add("EnumConstantBody: " + ecd.getName() + "." + md.getName() + " overrides nothing, skipped");
					continue;
				}
				out.append("\tthis.").append(dispatch.get(TypeModel.signature(d.target())).field()).append(" = ").append(bodyFunc(md, d)).append('\n');
			}
		}
		out.append("\treturn this\n}()\n\n");
	}

	private String bodyFunc(MethodDeclaration md, Dispatch d) {
		Emitter.BodyState saved = emitter.enterFunctionBody(d.target().getReturnType());
		try {
			String ret = emitter.retType(d.target());
			return "func(" + emitter.paramList(d.target(), md) + ") " + (ret.isEmpty() ? "" : ret + " ") + "{\n" + emitter.block(md.getBody(), 2) + "\t}";
		} finally {
			emitter.exitFunctionBody(saved);
		}
	}

	private void emitMethods(EnumDeclaration ed, TypeModel.ClassInfo ci, Map<String, Dispatch> dispatch, StringBuilder out) {
		for (Object o : ed.bodyDeclarations()) {
			if (!(o instanceof MethodDeclaration md) || md.isConstructor() || Modifier.isStatic(md.getModifiers())) continue;
			Dispatch d = dispatch.get(TypeModel.signature(md.resolveBinding()));
			if (!md.typeParameters().isEmpty()) {
				emitter.unsupported.add("MethodDeclaration: generic method " + ci.binaryName + "." + md.getName() + " skipped");
				continue;
			}
			if (md.getBody() == null) {
				if (d != null) emitFallback(d, "panic(\"j2go: abstract " + d.goName() + "\")", out);
				continue;
			}
			classEmitter.emitInstanceMethod(md, ci, out, d == null ? "" : guard(d));
		}
		// Overridden in constant bodies but declared by an interface or java.lang.Enum, not by the enum itself.
		for (Dispatch d : dispatch.values()) {
			if (d.declared() != null) continue;
			boolean toString = d.target().getName().equals("toString");
			emitFallback(d, toString ? "return this." + NAME_FIELD : "panic(\"j2go: abstract " + d.goName() + "\")", out);
		}
		for (Object o : ed.bodyDeclarations()) {
			if (o instanceof MethodDeclaration md && !md.isConstructor() && Modifier.isStatic(md.getModifiers()) && !Modifier.isNative(md.getModifiers())) {
				classEmitter.emitStaticMethod(md, ci, out);
			}
		}
	}

	private void emitFallback(Dispatch d, String statement, StringBuilder out) {
		out.append("func (this *").append(emitter.currentClassGoTypeName).append(") ").append(d.goName()).append('(')
				.append(emitter.paramList(d.target(), d.declared())).append(") ").append(emitter.retType(d.target())).append(" {\n")
				.append(guard(d)).append('\t').append(statement).append("\n}\n\n");
	}

	/** First lines of a dispatched method: the constant body's func field wins when set. */
	private String guard(Dispatch d) {
		String args;
		if (d.declared() != null) {
			args = emitter.argNames(d.declared());
		} else {
			List<String> a = new ArrayList<>();
			for (int i = 0; i < d.target().getParameterTypes().length; i++) a.add("a" + i);
			args = String.join(", ", a);
		}
		boolean isVoid = emitter.retType(d.target()).isEmpty();
		return "\tif this." + d.field() + " != nil {\n\t\t" + (isVoid ? "" : "return ") + "this." + d.field() + "(" + args + ")\n"
				+ (isVoid ? "\t\treturn\n" : "") + "\t}\n";
	}

	// ---------------------------------------------------------------- constant-body overrides

	private Map<String, Dispatch> dispatchMethods(EnumDeclaration ed) {
		Map<String, Dispatch> out = new LinkedHashMap<>();
		for (Object o : ed.enumConstants()) {
			AnonymousClassDeclaration body = ((EnumConstantDeclaration) o).getAnonymousClassDeclaration();
			if (body == null) continue;
			for (Object m : body.bodyDeclarations()) {
				if (m instanceof MethodDeclaration md) {
					Dispatch d = overriddenIn(md, ed);
					if (d != null) out.putIfAbsent(TypeModel.signature(d.target()), d);
				}
			}
		}
		return out;
	}

	/** The declaration md overrides, searched from the enum type upward; null when it overrides nothing. */
	private Dispatch overriddenIn(MethodDeclaration md, EnumDeclaration ed) {
		IMethodBinding mb = md.resolveBinding();
		Deque<ITypeBinding> todo = new ArrayDeque<>(List.of(ed.resolveBinding()));
		while (!todo.isEmpty()) {
			ITypeBinding t = todo.removeFirst();
			for (IMethodBinding sm : t.getDeclaredMethods()) {
				if (!mb.overrides(sm)) continue;
				String goName = emitter.names.goMemberName(sm, Names.javaMethodBaseGoName(sm.getName()));
				return new Dispatch(sm, ownDeclaration(sm, ed), goName, "fn" + goName);
			}
			if (t.getSuperclass() != null) todo.add(t.getSuperclass());
			todo.addAll(List.of(t.getInterfaces()));
		}
		return null;
	}

	private static MethodDeclaration ownDeclaration(IMethodBinding sm, EnumDeclaration ed) {
		for (Object o : ed.bodyDeclarations()) {
			if (o instanceof MethodDeclaration md && sm.isEqualTo(md.resolveBinding())) return md;
		}
		return null;
	}
}
