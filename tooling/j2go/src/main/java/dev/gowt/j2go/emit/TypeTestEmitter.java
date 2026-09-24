package dev.gowt.j2go.emit;

import dev.gowt.j2go.Names;
import dev.gowt.j2go.TypeModel;
import org.eclipse.jdt.core.dom.*;

import java.util.ArrayList;
import java.util.List;

/** instanceof (plain and pattern), cast checks, cascade helpers. */
final class TypeTestEmitter {

	private final Emitter emitter;

	TypeTestEmitter(Emitter emitter) {
		this.emitter = emitter;
	}

	String emitCast(CastExpression ce) {
		ITypeBinding t = ce.getType().resolveBinding();
		// A narrowing constant cast ((byte)0xFC): Go rejects an overflowing constant conversion.
		Object cv = ce.resolveConstantExpressionValue();
		if (cv instanceof Character c) cv = (int) c;
		if (t.isPrimitive() && (cv instanceof Byte || cv instanceof Short || cv instanceof Integer || cv instanceof Long)) {
			return dev.gowt.j2go.GoTypes.map(t, emitter) + "(" + cv + ")";
		}
		String expr = emitter.expr(ce.getExpression());
		// (Display) null: a disambiguating cast Java needs to pick an overload, not a runtime
		// check - Go's nil has no interface to assert against, so this stays bare "nil".
		if (ce.getExpression() instanceof NullLiteral) return "nil";
		// A Java reference cast (e.g. (id)other) is a type assertion in Go, not a conversion:
		// Go's T(x) conversion syntax doesn't apply between an interface and an unrelated pointer type.
		TypeModel.ClassInfo target = emitter.model.lookup(t);
		// (ImageDataProvider) zoom -> data: a lambda/method ref already has the target type.
		if (ce.getExpression() instanceof LambdaExpression || ce.getExpression() instanceof MethodReference) return expr;
		if (target != null && target.isInterface) return expr + ".(" + emitter.qualifiedTypeName(target) + ")";
		if (target != null && !target.isStruct) return castHelper(ce.getExpression(), target) + "(" + expr + ")";
		String qualified = t.getErasure().getQualifiedName();
		// A manual (untranslated) reference type - Composite, Shell, ... - needs the same
		// assertion syntax as a translated one; only a manual VALUE type (any/error) converts.
		if (target == null && dev.gowt.j2go.Manual.isManual(qualified) && !dev.gowt.j2go.Manual.isValueType(qualified)) {
			return implSubject(expr, ce.getExpression()) + ".(*" + dev.gowt.j2go.Manual.goTypeName(qualified) + ")";
		}
		String goType = dev.gowt.j2go.GoTypes.map(t, emitter);
		// A cast off a java.lang.Object-typed (Go "any") expression is a runtime-checked downcast
		// in Java, same as the branches above - Go's T(x) conversion doesn't apply from an
		// interface to a concrete array/string/struct type, only assertion does.
		ITypeBinding exprType = ce.getExpression().resolveTypeBinding();
		if (exprType != null && exprType.getErasure().getQualifiedName().equals("java.lang.Object") && !goType.equals("any")) {
			return expr + ".(" + goType + ")";
		}
		return goType + "(" + expr + ")";
	}

	/** A downcast through the impl cascade: (NSWindow) new SWTWindow().alloc() holds a *SWTWindow,
	 * so a plain Go assertion to *NSWindow would fail. null casts to null, a mismatch panics. */
	private String castHelper(Expression subject, TypeModel.ClassInfo target) {
		ITypeBinding st = subject.resolveTypeBinding();
		String fromGo = dev.gowt.j2go.GoTypes.map(st, emitter);
		TypeModel.ClassInfo subjectCi = emitter.model.lookup(st);
		String targetName = emitter.qualifiedTypeName(target);
		String name = "cast" + fromGo.replaceAll("[*.]", "") + "To" + targetName.replaceAll("[*.]", "");
		if (emitter.generatedHelpers.add(name)) {
			String impl = subjectCi == null || subjectCi.isInterface ? "x" : "x" + emitter.implAccess(subjectCi.root);
			String as = ensureCascadeHelper(target.root, target);
			emitter.fileHelperSource.add("func " + name + "(x " + fromGo + ") *" + targetName + " {\n\tif x == nil {\n\t\treturn nil\n\t}\n"
					+ "\tv, ok := " + as + "(" + impl + ")\n\tif !ok {\n\t\tpanic(\"java.lang.ClassCastException: " + targetName + "\")\n\t}\n\treturn v\n}\n\n");
		}
		return name;
	}

	String emitPlainInstanceof(InstanceofExpression ioe) {
		ITypeBinding target = ioe.getRightOperand().resolveBinding();
		String okVar = "ok" + (++emitter.tempCounter);
		String helperOrAssert = instanceofCheck(ioe.getLeftOperand(), target, "_", okVar);
		emitter.prelude.add(helperOrAssert);
		return okVar;
	}

	String emitPatternInstanceof(PatternInstanceofExpression pie) {
		TypePattern tp = (TypePattern) pie.getPattern();
		String varName = tp.getPatternVariable().getName().getIdentifier();
		ITypeBinding target = tp.getPatternVariable().getType().resolveBinding();
		String okVar = "ok" + (++emitter.tempCounter);
		emitter.prelude.add(instanceofCheck(pie.getLeftOperand(), target, varName, okVar));
		return okVar;
	}

	/** A struct-typed subject carries its dynamic subtype in its impl field, an INTERFACE value -
	 * needed for a plain Go type assertion to compile. Interface/Object subjects hold it directly. */
	private String implSubject(String subjectText, Expression subject) {
		ITypeBinding subjectType = subject.resolveTypeBinding();
		TypeModel.ClassInfo subjectCi = subjectType == null ? null : emitter.model.lookup(subjectType);
		return (subjectCi == null || subjectCi.isInterface) ? subjectText : subjectText + emitter.implAccess(subjectCi.root);
	}

	private String instanceofCheck(Expression subject, ITypeBinding target, String varName, String okVar) {
		TypeModel.ClassInfo targetCi = emitter.model.lookup(target);
		String subjectText = emitter.expr(subject);
		String implSubjectText = implSubject(subjectText, subject);
		if (targetCi == null) {
			String qualified = target.getErasure().getQualifiedName();
			// A manual (untranslated) reference type - Shell, Composite, ... - has no impl-cascade
			// entry of its own; asserting against .impl always compiles (it's an interface) but can
			// never actually match today, since nothing manual ever gets assigned into .impl.
			if (dev.gowt.j2go.Manual.isManual(qualified) && !dev.gowt.j2go.Manual.isValueType(qualified)) {
				return varName + ", " + okVar + " := " + implSubjectText + ".(*" + dev.gowt.j2go.Manual.goTypeName(qualified) + ")";
			}
			emitter.unsupported.add("instanceof: unresolved target type " + target.getQualifiedName());
			return varName + ", " + okVar + " := any(nil), false";
		}
		// An interface target is an ordinary Go type assertion - no impl-cascade dynamic-type
		// tracking involved, the interface value itself already carries its dynamic type.
		if (targetCi.isInterface) {
			return varName + ", " + okVar + " := " + subjectText + ".(" + emitter.qualifiedTypeName(targetCi) + ")";
		}
		ITypeBinding st = subject.resolveTypeBinding();
		TypeModel.ClassInfo subjectCi = st == null ? null : emitter.model.lookup(st);
		if (subjectCi == null || subjectCi.isInterface) {
			return varName + ", " + okVar + " := " + ensureCascadeHelper(targetCi.root, targetCi) + "(" + subjectText + ")";
		}
		return varName + ", " + okVar + " := " + nilSafeInstanceof(st, subjectCi, targetCi) + "(" + subjectText + ")";
	}

	/** `x instanceof T` on a class-typed x: false for null, where reading x.impl would panic. */
	private String nilSafeInstanceof(ITypeBinding st, TypeModel.ClassInfo subjectCi, TypeModel.ClassInfo target) {
		String fromGo = dev.gowt.j2go.GoTypes.map(st, emitter);
		String targetName = emitter.qualifiedTypeName(target);
		String name = "is" + fromGo.replaceAll("[*.]", "") + "To" + targetName.replaceAll("[*.]", "");
		if (emitter.generatedHelpers.add(name)) {
			String as = ensureCascadeHelper(target.root, target);
			emitter.fileHelperSource.add("func " + name + "(x " + fromGo + ") (*" + targetName + ", bool) {\n\tif x == nil {\n\t\treturn nil, false\n\t}\n"
					+ "\treturn " + as + "(x" + emitter.implAccess(subjectCi.root) + ")\n}\n\n");
		}
		return name;
	}

	String ensureCascadeHelper(TypeModel.ClassInfo root, TypeModel.ClassInfo target) {
		// Nested classes (Point/Point.OfFloat) share a textual prefix; unrelated top-level roots
		// (Widget/Shell) don't - fall back to target's own full name for those.
		String targetLabel = target == root ? target.goFuncPrefix
				: target.goFuncPrefix.startsWith(root.goFuncPrefix)
				? target.goFuncPrefix.substring(root.goFuncPrefix.length())
				: target.goFuncPrefix;
		String name = Names.decapitalize(root.goFuncPrefix) + "ImplAs" + targetLabel;
		if (emitter.generatedHelpers.add(name)) {
			emitter.fileHelperSource.add(buildCascadeHelper(name, target));
		}
		return name;
	}

	private String buildCascadeHelper(String name, TypeModel.ClassInfo target) {
		List<TypeModel.ClassInfo> concrete = new ArrayList<>();
		collectDescendants(target, concrete);
		String targetName = emitter.qualifiedTypeName(target);
		StringBuilder b = new StringBuilder();
		b.append("// j2go: instanceof helper for ").append(targetName)
				.append(" and its subclasses within the translated set.\n");
		b.append("func ").append(name).append("(x any) (*").append(targetName).append(", bool) {\n");
		b.append("\tswitch v := x.(type) {\n");
		for (TypeModel.ClassInfo c : concrete) {
			b.append("\tcase *").append(emitter.qualifiedTypeName(c)).append(":\n");
			if (c == target) {
				b.append("\t\treturn v, true\n");
			} else {
				b.append("\t\treturn &v.").append(target.goTypeName).append(", true\n");
			}
		}
		b.append("\t}\n\treturn nil, false\n}\n\n");
		return b.toString();
	}

	private void collectDescendants(TypeModel.ClassInfo c, List<TypeModel.ClassInfo> out) {
		out.add(c);
		for (TypeModel.ClassInfo child : c.children) collectDescendants(child, out);
	}
}
